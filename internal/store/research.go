package store

import (
	"database/sql"
	"fmt"
	"math"
)

// Группы эксперимента.
const (
	// GroupFull — полный продукт: интервальные повторения и тетрадь ошибок.
	GroupFull = "A"
	// GroupControl — контроль: те же уроки, но без возвратов к пройденному.
	GroupControl = "B"
)

// ValidGroup проверяет метку группы. Всё остальное — не участвует.
func ValidGroup(g string) bool { return g == GroupFull || g == GroupControl }

// SaveMeasurement записывает замер КСПОЯ.
//
// Повторный замер той же фазы отбрасывается, а не перезаписывается:
// иначе посттест можно переписывать, пока не выйдет нужная цифра, и
// исследование перестанет что-либо значить. Возвращает false, если замер
// этой фазы уже был.
func (s *Store) SaveMeasurement(userID int64, phase, sessionID string, score, total int, level string) (bool, error) {
	res, err := s.db.Exec(`
		INSERT INTO research_measurements (user_id, phase, session_id, score, total, level)
		VALUES ($1,$2,$3,$4,$5,$6)
		ON CONFLICT (user_id, phase) DO NOTHING`,
		userID, phase, sessionID, score, total, level)
	if err != nil {
		return false, fmt.Errorf("SaveMeasurement: %w", err)
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// ResearchRow — строка выгрузки по одному ученику.
type ResearchRow struct {
	Subject  string // обезличенный номер
	Group    string
	Pre      sql.NullInt64
	Post     sql.NullInt64
	Total    int
	Topics   int // тем начато
	Mastered int // тем освоено
	Lapses   int // сколько раз темы забывались
	Reviews  int // ответов из повторений и тетради ошибок
	Answers  int // всего ответов
	Hinted   int // ответов с открытым разбором
	Days     int // разных дней занятий
}

// ResearchExport собирает данные эксперимента по всем участникам.
//
// В выгрузку попадают только те, у кого проставлена группа И получено
// согласие. Логины не выгружаются вообще: для исследования нужен
// идентификатор, связывающий строки одного человека, а не его имя.
func (s *Store) ResearchExport() ([]ResearchRow, error) {
	rows, err := s.db.Query(`
		SELECT
		    u.id,
		    u.study_group,
		    (SELECT m.score FROM research_measurements m
		      WHERE m.user_id = u.id AND m.phase = 'pre'),
		    (SELECT m.score FROM research_measurements m
		      WHERE m.user_id = u.id AND m.phase = 'post'),
		    COALESCE((SELECT m.total FROM research_measurements m
		      WHERE m.user_id = u.id AND m.phase = 'pre'), 0),
		    (SELECT COUNT(*) FROM topic_mastery t WHERE t.user_id = u.id),
		    (SELECT COUNT(*) FROM topic_mastery t WHERE t.user_id = u.id AND t.box >= 3),
		    COALESCE((SELECT SUM(t.lapses) FROM topic_mastery t WHERE t.user_id = u.id), 0),
		    (SELECT COUNT(*) FROM attempts a
		      WHERE a.user_id = u.id AND a.source IN ('review')),
		    (SELECT COUNT(*) FROM attempts a WHERE a.user_id = u.id),
		    (SELECT COUNT(*) FROM attempts a WHERE a.user_id = u.id AND a.hints_used >= 3),
		    (SELECT COUNT(DISTINCT a.answered_at::date) FROM attempts a WHERE a.user_id = u.id)
		FROM users u
		WHERE u.study_group IS NOT NULL AND u.research_consent
		ORDER BY u.study_group, u.id`)
	if err != nil {
		return nil, fmt.Errorf("ResearchExport: %w", err)
	}
	defer rows.Close()

	out := []ResearchRow{}
	seq := 0
	for rows.Next() {
		var (
			id int64
			r  ResearchRow
		)
		if err := rows.Scan(&id, &r.Group, &r.Pre, &r.Post, &r.Total,
			&r.Topics, &r.Mastered, &r.Lapses, &r.Reviews, &r.Answers,
			&r.Hinted, &r.Days); err != nil {
			return nil, fmt.Errorf("ResearchExport: scan: %w", err)
		}
		seq++
		r.Subject = fmt.Sprintf("%s-%03d", r.Group, seq)
		out = append(out, r)
	}
	return out, rows.Err()
}

// GroupStats — сводка по одной группе.
type GroupStats struct {
	Group string `json:"group"`
	// N — сколько участников всего в группе.
	N int `json:"n"`
	// Paired — у скольких есть оба замера. Именно на них считается прирост:
	// участник без посттеста ничего не говорит о результате.
	Paired   int     `json:"paired"`
	MeanPre  float64 `json:"mean_pre"`
	MeanPost float64 `json:"mean_post"`
	MeanGain float64 `json:"mean_gain"`
	SDGain   float64 `json:"sd_gain"`
	// MedianDays — медиана числа дней занятий: показывает, сравнимы ли
	// группы по вовлечённости. Если одна занималась вдвое дольше, разница
	// в приросте объясняется не методикой.
	MedianDays float64 `json:"median_days"`
}

// ResearchSummary считает сводку по группам.
//
// Считается на сервере, а не в таблице после выгрузки, по одной причине:
// цифру, которую покажут на защите, должно быть видно в любой момент, а
// не за вечер до неё. Полная выгрузка при этом остаётся — проверить
// расчёт по сырым данным должно быть можно.
func (s *Store) ResearchSummary() ([]GroupStats, error) {
	rows, err := s.ResearchExport()
	if err != nil {
		return nil, err
	}

	byGroup := map[string][]ResearchRow{}
	for _, r := range rows {
		byGroup[r.Group] = append(byGroup[r.Group], r)
	}

	out := []GroupStats{}
	for _, g := range []string{GroupFull, GroupControl} {
		list := byGroup[g]
		st := GroupStats{Group: g, N: len(list)}

		gains, pres, posts, days := []float64{}, []float64{}, []float64{}, []float64{}
		for _, r := range list {
			days = append(days, float64(r.Days))
			if !r.Pre.Valid || !r.Post.Valid {
				continue
			}
			st.Paired++
			pres = append(pres, float64(r.Pre.Int64))
			posts = append(posts, float64(r.Post.Int64))
			gains = append(gains, float64(r.Post.Int64-r.Pre.Int64))
		}

		st.MeanPre = round2f(mean(pres))
		st.MeanPost = round2f(mean(posts))
		st.MeanGain = round2f(mean(gains))
		st.SDGain = round2f(stdDev(gains))
		st.MedianDays = round2f(median(days))
		out = append(out, st)
	}
	return out, nil
}

func mean(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	sum := 0.0
	for _, x := range v {
		sum += x
	}
	return sum / float64(len(v))
}

// stdDev — выборочное стандартное отклонение (делитель n-1).
//
// Именно выборочное, а не по всей совокупности: класс — это выборка из
// всех школьников, а не вся генеральная совокупность, и делитель n
// занижал бы разброс.
func stdDev(v []float64) float64 {
	if len(v) < 2 {
		return 0
	}
	m := mean(v)
	sum := 0.0
	for _, x := range v {
		sum += (x - m) * (x - m)
	}
	return math.Sqrt(sum / float64(len(v)-1))
}

func median(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	s := append([]float64(nil), v...)
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
	mid := len(s) / 2
	if len(s)%2 == 1 {
		return s[mid]
	}
	return (s[mid-1] + s[mid]) / 2
}

func round2f(v float64) float64 { return math.Round(v*100) / 100 }

// SetStudyGroup помечает ученика группой эксперимента.
// Пустая строка убирает его из выборки.
func (s *Store) SetStudyGroup(username, group string) error {
	var val any
	if group != "" {
		if !ValidGroup(group) {
			return fmt.Errorf("неизвестная группа %q, ожидалось %q или %q", group, GroupFull, GroupControl)
		}
		val = group
	}
	res, err := s.db.Exec(`UPDATE users SET study_group=$2 WHERE username=$1`, username, val)
	if err != nil {
		return fmt.Errorf("SetStudyGroup: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("пользователь %q не найден", username)
	}
	return nil
}

// SetResearchConsent сохраняет согласие на использование обезличенных
// данных. Без него ученик не попадает в выгрузку ни при какой группе.
func (s *Store) SetResearchConsent(userID int64, on bool) error {
	_, err := s.db.Exec(`UPDATE users SET research_consent=$2 WHERE id=$1`, userID, on)
	if err != nil {
		return fmt.Errorf("SetResearchConsent: %w", err)
	}
	return nil
}

// StudyGroupOf — группа ученика и наличие согласия.
func (s *Store) StudyGroupOf(userID int64) (group string, consent bool) {
	var g sql.NullString
	if err := s.db.QueryRow(
		`SELECT study_group, research_consent FROM users WHERE id=$1`, userID,
	).Scan(&g, &consent); err != nil {
		return "", false
	}
	return g.String, consent
}

// MeasurementPhase решает, каким замером считать эту попытку теста.
//
// Первая попытка участника — предтест. Посттест ставится только вручную,
// когда эксперимент закончен: иначе вторая попытка на второй день
// объявила бы себя итогом.
func (s *Store) MeasurementPhase(userID int64) string {
	var n int
	if err := s.db.QueryRow(
		`SELECT COUNT(*) FROM research_measurements WHERE user_id=$1`, userID,
	).Scan(&n); err != nil {
		return ""
	}
	if n == 0 {
		return "pre"
	}
	return ""
}
