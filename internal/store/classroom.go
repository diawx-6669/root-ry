package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lib/pq"
)

// ═══════════════════════════════════════════════════════════════════
//  Классрум — общий класс участников эксперимента.
//
//  Попадают в него ученики, которые в профиле поставили галочку
//  «Участие в исследовании» (research_consent). Ученики видят друг
//  друга в открытом виде: ник, уровень, серию, значки. Администратор
//  видит по каждому полную картину: какие темы пройдены, где ошибки
//  в уроках, играх и КСПОЯ.
//
//  Всё считается из уже существующих журналов (attempts, topic_mastery,
//  item_state, game_results, kspoya_sessions): отдельных таблиц не
//  заводим, иначе второй источник правды рано или поздно разъедется.
// ═══════════════════════════════════════════════════════════════════

// ClassroomMember — открытая карточка участника.
type ClassroomMember struct {
	Username     string   `json:"username"`
	Nickname     string   `json:"nickname"`
	ActiveAvatar string   `json:"active_avatar"`
	XP           int      `json:"xp"`
	Streak       int      `json:"streak"`
	Badges       []string `json:"badges"`
	TopicsDone   int      `json:"topics_done"`
	KspoyaScore  int      `json:"kspoya_score"`
	KspoyaLevel  string   `json:"kspoya_level"`
	LastActive   string   `json:"last_active"`
	JoinedAt     string   `json:"joined_at"`
}

// InClassroom — состоит ли ученик в классруме (дал согласие).
func (s *Store) InClassroom(username string) bool {
	var on bool
	s.db.QueryRow(`SELECT research_consent FROM users WHERE username=$1`, username).Scan(&on)
	return on
}

const classroomMemberSelect = `
	SELECT u.username, u.nickname, u.active_avatar, u.xp, u.streak, u.badges,
	       jsonb_array_length(u.completed_topics),
	       COALESCE(best.raw_score, 0), COALESCE(best.level_key, ''),
	       COALESCE(to_char(GREATEST(u.last_login, act.last_answer), 'YYYY-MM-DD'), ''),
	       to_char(u.created_at, 'YYYY-MM-DD')
	FROM users u
	LEFT JOIN LATERAL (
	    SELECT raw_score, level_key FROM kspoya_sessions
	    WHERE username = u.username AND status = 'completed' AND level_key <> ''
	    ORDER BY raw_score DESC, finished_at ASC LIMIT 1
	) best ON TRUE
	LEFT JOIN LATERAL (
	    SELECT max(answered_at)::date AS last_answer FROM attempts WHERE user_id = u.id
	) act ON TRUE`

func scanMember(row interface{ Scan(...any) error }) (ClassroomMember, error) {
	var (
		m      ClassroomMember
		badges []byte
	)
	err := row.Scan(&m.Username, &m.Nickname, &m.ActiveAvatar, &m.XP, &m.Streak, &badges,
		&m.TopicsDone, &m.KspoyaScore, &m.KspoyaLevel, &m.LastActive, &m.JoinedAt)
	if err != nil {
		return m, err
	}
	m.Badges = decodeStrings(badges)
	return m, nil
}

// ClassroomMembers — все участники, у кого больше опыта — выше.
// Администраторы в список не попадают: они смотрят, а не учатся.
func (s *Store) ClassroomMembers() ([]ClassroomMember, error) {
	rows, err := s.db.Query(classroomMemberSelect + `
		WHERE u.research_consent = TRUE AND u.is_admin = FALSE
		ORDER BY u.xp DESC, u.username
		LIMIT 500`)
	if err != nil {
		return nil, fmt.Errorf("ClassroomMembers: %w", err)
	}
	defer rows.Close()
	out := []ClassroomMember{}
	for rows.Next() {
		m, err := scanMember(rows)
		if err != nil {
			return nil, fmt.Errorf("ClassroomMembers: scan: %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// ClassroomMember — карточка одного участника; false, если его нет в классруме.
func (s *Store) ClassroomMember(username string) (ClassroomMember, bool) {
	m, err := scanMember(s.db.QueryRow(classroomMemberSelect+`
		WHERE u.username = $1 AND u.research_consent = TRUE AND u.is_admin = FALSE`, username))
	if err != nil {
		return m, false
	}
	return m, true
}

// ── Полная картина для администратора ─────────────────────────────

// SourceStat — сколько ответов и сколько верных из одного источника.
type SourceStat struct {
	Source  string `json:"source"` // lesson | game | kspoya | review
	Total   int    `json:"total"`
	Correct int    `json:"correct"`
}

// TopicStat — точность по теме по журналу ответов.
type TopicStat struct {
	TopicID string `json:"topic_id"`
	Total   int    `json:"total"`
	Correct int    `json:"correct"`
	Lapses  int    `json:"lapses"`
	Box     int    `json:"box"`
	Last    string `json:"last"`
}

// WrongAnswer — одна ошибка из журнала.
type WrongAnswer struct {
	TopicID string `json:"topic_id"`
	ItemID  string `json:"item_id"`
	Source  string `json:"source"`
	At      string `json:"at"`
}

// GameStat — сводка по одной игре.
type GameStat struct {
	GameType  string `json:"game_type"`
	Played    int    `json:"played"`
	BestScore int    `json:"best_score"`
	AvgScore  int    `json:"avg_score"`
	Coins     int    `json:"coins"`
	Last      string `json:"last"`
}

// KspoyaRun — одна завершённая попытка КСПОЯ с ответами.
type KspoyaRun struct {
	ID          string    `json:"id"`
	FinishedAt  time.Time `json:"finished_at"`
	Score       int       `json:"score"`
	Total       int       `json:"total"`
	Level       string    `json:"level"`
	QuestionIDs []int     `json:"-"`
	Answers     []int     `json:"-"`
}

// StudentReport — всё, что администратор видит по ученику.
type StudentReport struct {
	UserID          int64         `json:"-"`
	CompletedTopics []string      `json:"completed_topics"`
	Sources         []SourceStat  `json:"sources"`
	Topics          []TopicStat   `json:"topics"`
	RecentWrong     []WrongAnswer `json:"recent_wrong"`
	OpenMistakes    int           `json:"open_mistakes"`
	Games           []GameStat    `json:"games"`
	Kspoya          []KspoyaRun   `json:"-"`
	ActiveDays      int           `json:"active_days"`
	TotalAnswers    int           `json:"total_answers"`
}

// StudentReport собирает полную картину по ученику.
func (s *Store) StudentReport(username string) (*StudentReport, error) {
	var (
		rep       StudentReport
		completed []byte
	)
	if err := s.db.QueryRow(`SELECT id, completed_topics FROM users WHERE username=$1`, username).
		Scan(&rep.UserID, &completed); err != nil {
		return nil, fmt.Errorf("StudentReport: user: %w", err)
	}
	rep.CompletedTopics = decodeStrings(completed)
	id := rep.UserID

	// Ответы по источникам: урок, игра, КСПОЯ, повторение.
	rows, err := s.db.Query(`
		SELECT source, count(*), count(*) FILTER (WHERE correct)
		FROM attempts WHERE user_id=$1 GROUP BY source ORDER BY source`, id)
	if err != nil {
		return nil, fmt.Errorf("StudentReport: sources: %w", err)
	}
	rep.Sources = []SourceStat{}
	for rows.Next() {
		var st SourceStat
		if err := rows.Scan(&st.Source, &st.Total, &st.Correct); err != nil {
			rows.Close()
			return nil, err
		}
		rep.TotalAnswers += st.Total
		rep.Sources = append(rep.Sources, st)
	}
	rows.Close()

	// Точность по темам: журнал ответов плюс коробка Лейтнера.
	rows, err = s.db.Query(`
		SELECT a.topic_id, count(*), count(*) FILTER (WHERE a.correct),
		       COALESCE(m.lapses, 0), COALESCE(m.box, 0),
		       to_char(max(a.answered_at), 'YYYY-MM-DD')
		FROM attempts a
		LEFT JOIN topic_mastery m ON m.user_id = a.user_id AND m.topic_id = a.topic_id
		WHERE a.user_id=$1
		GROUP BY a.topic_id, m.lapses, m.box
		ORDER BY (count(*) FILTER (WHERE a.correct))::float / count(*), count(*) DESC`, id)
	if err != nil {
		return nil, fmt.Errorf("StudentReport: topics: %w", err)
	}
	rep.Topics = []TopicStat{}
	for rows.Next() {
		var t TopicStat
		if err := rows.Scan(&t.TopicID, &t.Total, &t.Correct, &t.Lapses, &t.Box, &t.Last); err != nil {
			rows.Close()
			return nil, err
		}
		rep.Topics = append(rep.Topics, t)
	}
	rows.Close()

	// Последние ошибки — откуда угодно.
	rows, err = s.db.Query(`
		SELECT topic_id, item_id, source, to_char(answered_at AT TIME ZONE $2, 'YYYY-MM-DD HH24:MI')
		FROM attempts WHERE user_id=$1 AND NOT correct
		ORDER BY answered_at DESC LIMIT 60`, id, localZone)
	if err != nil {
		return nil, fmt.Errorf("StudentReport: wrong: %w", err)
	}
	rep.RecentWrong = []WrongAnswer{}
	for rows.Next() {
		var wa WrongAnswer
		if err := rows.Scan(&wa.TopicID, &wa.ItemID, &wa.Source, &wa.At); err != nil {
			rows.Close()
			return nil, err
		}
		rep.RecentWrong = append(rep.RecentWrong, wa)
	}
	rows.Close()

	rep.OpenMistakes = s.MistakeCount(id)
	s.db.QueryRow(`SELECT count(DISTINCT answered_at::date) FROM attempts WHERE user_id=$1`, id).Scan(&rep.ActiveDays)

	// Игры.
	rows, err = s.db.Query(`
		SELECT game_type, count(*), max(score), round(avg(score))::int, sum(coins_earned),
		       to_char(max(played_at) AT TIME ZONE $2, 'YYYY-MM-DD')
		FROM game_results WHERE user_id=$1
		GROUP BY game_type ORDER BY count(*) DESC`, id, localZone)
	if err != nil {
		return nil, fmt.Errorf("StudentReport: games: %w", err)
	}
	rep.Games = []GameStat{}
	for rows.Next() {
		var g GameStat
		if err := rows.Scan(&g.GameType, &g.Played, &g.BestScore, &g.AvgScore, &g.Coins, &g.Last); err != nil {
			rows.Close()
			return nil, err
		}
		rep.Games = append(rep.Games, g)
	}
	rows.Close()

	// КСПОЯ: последние попытки вместе с ответами — для разбора ошибок.
	rows, err = s.db.Query(`
		SELECT id, finished_at, COALESCE(raw_score,0),
		       COALESCE(array_length(question_ids,1),0), COALESCE(level_key,''),
		       question_ids, COALESCE(answers,'{}')
		FROM kspoya_sessions
		WHERE username=$1 AND status='completed' AND level_key <> ''
		ORDER BY finished_at DESC LIMIT 10`, username)
	if err != nil {
		return nil, fmt.Errorf("StudentReport: kspoya: %w", err)
	}
	for rows.Next() {
		var (
			k        KspoyaRun
			ids, ans pq.Int64Array
			fin      sql.NullTime
		)
		if err := rows.Scan(&k.ID, &fin, &k.Score, &k.Total, &k.Level, &ids, &ans); err != nil {
			rows.Close()
			return nil, err
		}
		k.FinishedAt = fin.Time
		k.QuestionIDs = arrayToInts(ids)
		k.Answers = arrayToInts(ans)
		rep.Kspoya = append(rep.Kspoya, k)
	}
	rows.Close()
	return &rep, nil
}

func decodeStrings(b []byte) []string {
	var out []string
	if len(b) > 0 {
		_ = json.Unmarshal(b, &out)
	}
	if out == nil {
		out = []string{}
	}
	return out
}
