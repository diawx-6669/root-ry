package store

import (
	"database/sql"
	"fmt"
	"time"

	"rootry/internal/mastery"
	"rootry/internal/models"
)

// AttemptInput — один ответ ученика на одно задание.
type AttemptInput struct {
	UserID  int64
	TopicID string
	ItemID  string
	Source  string // lesson | game | kspoya | review
	Correct bool
	Hints   int
	TimeMs  int
}

// AttemptOutcome — что изменилось в модели знаний после ответа.
//
// Состояния «до» нужны обработчику, чтобы понять, что именно произошло:
// тема поднялась на коробку выше, задание закрылось в тетради ошибок,
// ученик вернулся к просроченной теме. Награда считается по разнице, а не
// по факту ответа — иначе можно было бы фармить XP на одном задании.
type AttemptOutcome struct {
	TopicWas mastery.State
	Topic    mastery.State
	ItemWas  mastery.State
	Item     mastery.State
}

// TopicPromoted — тема поднялась на коробку выше именно этим ответом.
//
// Переход из нулевой коробки в первую повышением не считается: это не
// подтверждение знания, а первое открытие темы. Иначе интерфейс поздравлял
// бы с «продвижением» на первом же вопросе первого урока.
func (o AttemptOutcome) TopicPromoted() bool {
	return o.TopicWas.Box > 0 && o.Topic.Box > o.TopicWas.Box
}

// MistakeClosed — задание только что ушло из тетради ошибок.
func (o AttemptOutcome) MistakeClosed() bool {
	return o.ItemWas.Total > 0 && !o.ItemWas.Resolved() && o.Item.Resolved()
}

// MistakeOpened — задание только что попало в тетрадь ошибок.
func (o AttemptOutcome) MistakeOpened() bool {
	return o.ItemWas.Resolved() != o.Item.Resolved() && !o.Item.Resolved()
}

// RecordAttempt пишет ответ в журнал и пересчитывает состояние темы и задания.
//
// Всё в одной транзакции со строчными блокировками: два ответа, пришедших
// одновременно (ученик открыл урок в двух вкладках), иначе затёрли бы друг
// друга и серия подтверждений посчиталась бы неверно.
func (s *Store) RecordAttempt(in AttemptInput, now time.Time) (AttemptOutcome, error) {
	var out AttemptOutcome

	tx, err := s.db.Begin()
	if err != nil {
		return out, fmt.Errorf("RecordAttempt: begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(`
		INSERT INTO attempts (user_id, topic_id, item_id, source, correct, hints_used, time_ms)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		in.UserID, in.TopicID, in.ItemID, in.Source, in.Correct, in.Hints, in.TimeMs,
	); err != nil {
		return out, fmt.Errorf("RecordAttempt: insert: %w", err)
	}

	out.TopicWas, err = lockTopicMastery(tx, in.UserID, in.TopicID)
	if err != nil {
		return out, err
	}
	out.Topic = mastery.Apply(out.TopicWas, in.Correct, now)
	if err := saveTopicMastery(tx, in.UserID, in.TopicID, out.Topic); err != nil {
		return out, err
	}

	out.ItemWas, err = lockItemState(tx, in.UserID, in.ItemID)
	if err != nil {
		return out, err
	}
	out.Item = mastery.Apply(out.ItemWas, in.Correct, now)
	if err := saveItemState(tx, in, out.Item, now); err != nil {
		return out, err
	}

	if err := tx.Commit(); err != nil {
		return out, fmt.Errorf("RecordAttempt: commit: %w", err)
	}
	return out, nil
}

// lockTopicMastery читает состояние темы и удерживает строку до конца
// транзакции. Отсутствие строки — нормальная ситуация: тему ещё не открывали.
func lockTopicMastery(tx *sql.Tx, userID int64, topicID string) (mastery.State, error) {
	row := tx.QueryRow(`
		SELECT box, streak_days, lapses, correct, total, due_on, last_seen, first_done
		FROM topic_mastery WHERE user_id=$1 AND topic_id=$2 FOR UPDATE`,
		userID, topicID)

	var (
		st                  mastery.State
		dueOn               time.Time
		lastSeen, firstDone sql.NullTime
	)
	err := row.Scan(&st.Box, &st.StreakDays, &st.Lapses, &st.Correct, &st.Total,
		&dueOn, &lastSeen, &firstDone)
	switch {
	case err == sql.ErrNoRows:
		return mastery.State{}, nil
	case err != nil:
		return mastery.State{}, fmt.Errorf("lockTopicMastery: %w", err)
	}
	st.DueOn = dueOn
	st.LastSeen = lastSeen.Time
	st.FirstDone = firstDone.Time
	return st, nil
}

func saveTopicMastery(tx *sql.Tx, userID int64, topicID string, st mastery.State) error {
	_, err := tx.Exec(`
		INSERT INTO topic_mastery
			(user_id, topic_id, box, streak_days, lapses, correct, total, due_on, last_seen, first_done)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		ON CONFLICT (user_id, topic_id) DO UPDATE SET
			box         = EXCLUDED.box,
			streak_days = EXCLUDED.streak_days,
			lapses      = EXCLUDED.lapses,
			correct     = EXCLUDED.correct,
			total       = EXCLUDED.total,
			due_on      = EXCLUDED.due_on,
			last_seen   = EXCLUDED.last_seen`,
		userID, topicID, st.Box, st.StreakDays, st.Lapses, st.Correct, st.Total,
		st.DueOn, nullDate(st.LastSeen), nullDate(st.FirstDone))
	if err != nil {
		return fmt.Errorf("saveTopicMastery: %w", err)
	}
	return nil
}

func lockItemState(tx *sql.Tx, userID int64, itemID string) (mastery.State, error) {
	row := tx.QueryRow(`
		SELECT box, streak_days, correct, total, last_seen
		FROM item_state WHERE user_id=$1 AND item_id=$2 FOR UPDATE`,
		userID, itemID)

	var (
		st       mastery.State
		lastSeen sql.NullTime
	)
	err := row.Scan(&st.Box, &st.StreakDays, &st.Correct, &st.Total, &lastSeen)
	switch {
	case err == sql.ErrNoRows:
		return mastery.State{}, nil
	case err != nil:
		return mastery.State{}, fmt.Errorf("lockItemState: %w", err)
	}
	st.LastSeen = lastSeen.Time
	return st, nil
}

// saveItemState обновляет строку задания. wrong_count растёт только на
// неверных ответах, last_wrong двигается только вместе с ним — иначе
// тетрадь ошибок сортировалась бы по дате последнего касания, а не по
// дате последней ошибки.
func saveItemState(tx *sql.Tx, in AttemptInput, st mastery.State, now time.Time) error {
	wrongInc := 0
	if !in.Correct {
		wrongInc = 1
	}
	_, err := tx.Exec(`
		INSERT INTO item_state
			(user_id, item_id, topic_id, box, streak_days, wrong_count, correct, total, resolved, last_wrong, last_seen)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		ON CONFLICT (user_id, item_id) DO UPDATE SET
			topic_id    = EXCLUDED.topic_id,
			box         = EXCLUDED.box,
			streak_days = EXCLUDED.streak_days,
			wrong_count = item_state.wrong_count + $6,
			correct     = EXCLUDED.correct,
			total       = EXCLUDED.total,
			resolved    = EXCLUDED.resolved,
			last_wrong  = CASE WHEN $12 THEN EXCLUDED.last_wrong ELSE item_state.last_wrong END,
			last_seen   = EXCLUDED.last_seen`,
		in.UserID, in.ItemID, in.TopicID, st.Box, st.StreakDays, wrongInc,
		st.Correct, st.Total, st.Resolved(),
		nullDateIf(!in.Correct, now), now, !in.Correct)
	if err != nil {
		return fmt.Errorf("saveItemState: %w", err)
	}
	return nil
}

// ── Чтение ────────────────────────────────────────────────────────────────────

// TopicMasteryMap отдаёт состояние всех тем ученика одним запросом.
// Дерево грамматики рисует 74 узла, и делать под каждый отдельный запрос
// нельзя.
func (s *Store) TopicMasteryMap(userID int64) (map[string]mastery.State, error) {
	rows, err := s.db.Query(`
		SELECT topic_id, box, streak_days, lapses, correct, total, due_on, last_seen, first_done
		FROM topic_mastery WHERE user_id=$1`, userID)
	if err != nil {
		return nil, fmt.Errorf("TopicMasteryMap: %w", err)
	}
	defer rows.Close()

	out := make(map[string]mastery.State)
	for rows.Next() {
		var (
			topicID             string
			st                  mastery.State
			dueOn               time.Time
			lastSeen, firstDone sql.NullTime
		)
		if err := rows.Scan(&topicID, &st.Box, &st.StreakDays, &st.Lapses,
			&st.Correct, &st.Total, &dueOn, &lastSeen, &firstDone); err != nil {
			return nil, fmt.Errorf("TopicMasteryMap: scan: %w", err)
		}
		st.DueOn = dueOn
		st.LastSeen = lastSeen.Time
		st.FirstDone = firstDone.Time
		out[topicID] = st
	}
	return out, rows.Err()
}

// GameItemMark — метка задания, пришедшего из игры.
//
// Идентификаторы заданий устроены как «тема#номер» для уроков и
// «тема#g-игра-ключ» для игр (см. recordGameAnswer в static/api.js).
const GameItemMark = "#g-"

// OpenMistakes — незакрытые задания для тетради ошибок, самые свежие первыми.
//
// Возвращаются только идентификаторы: текст задания и разбор живут в
// static/lessons/*.js, и дублировать их в базе значило бы завести второй
// источник правды, который рано или поздно разъедется с первым.
//
// Задания из игр в тетрадь не попадают. Не потому, что ошибка в игре менее
// важна — тему она роняет так же, и в план повторения та вернётся, — а
// потому, что открыть одно задание игры отдельно нельзя: оно живёт внутри
// своей механики. Показывать в тетради строку, которую невозможно
// прорешать, значит обещать то, чего нет.
func (s *Store) OpenMistakes(userID int64, limit int) ([]models.MistakeRef, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	rows, err := s.db.Query(`
		SELECT item_id, topic_id, wrong_count, streak_days, last_wrong
		FROM item_state
		WHERE user_id=$1 AND resolved=FALSE AND wrong_count > 0
		  AND position($3 in item_id) = 0
		ORDER BY last_wrong DESC NULLS LAST, item_id
		LIMIT $2`, userID, limit, GameItemMark)
	if err != nil {
		return nil, fmt.Errorf("OpenMistakes: %w", err)
	}
	defer rows.Close()

	out := []models.MistakeRef{}
	for rows.Next() {
		var (
			m         models.MistakeRef
			lastWrong sql.NullTime
		)
		if err := rows.Scan(&m.ItemID, &m.TopicID, &m.WrongCount, &m.StreakDays, &lastWrong); err != nil {
			return nil, fmt.Errorf("OpenMistakes: scan: %w", err)
		}
		if lastWrong.Valid {
			m.LastWrong = lastWrong.Time.Format("2006-01-02")
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// MistakeCount — сколько заданий сейчас в тетради. Нужен для счётчика в
// шапке, ради которого не стоит тянуть весь список.
func (s *Store) MistakeCount(userID int64) int {
	var n int
	err := s.db.QueryRow(`
		SELECT COUNT(*) FROM item_state
		WHERE user_id=$1 AND resolved=FALSE AND wrong_count > 0
		  AND position($2 in item_id) = 0`, userID, GameItemMark).Scan(&n)
	if err != nil {
		return 0
	}
	return n
}

// nullDate превращает нулевое время в NULL: колонка DATE не должна хранить
// «0001-01-01» как признак отсутствия значения.
func nullDate(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t
}

func nullDateIf(cond bool, t time.Time) any {
	if !cond {
		return nil
	}
	return t
}

// ── Награда за тему ───────────────────────────────────────────────────────────

// Виды награды, которые возвращает ClaimTopicReward.
const (
	// RewardFirst — тема закрыта впервые, награда полная.
	RewardFirst = "first"
	// RewardReview — повторение просроченной темы, награда уменьшенная.
	RewardReview = "review"
	// RewardNone — тему уже награждали в этом интервале.
	RewardNone = "none"
)

// ClaimTopicReward решает, положена ли ученику награда за пройденную тему.
//
// Ключевая мысль: награда привязана не к ответам, а к интервалу. Если бы она
// зависела от прохождения урока, ученик закрывал бы один и тот же урок в
// цикле. Поэтому есть отдельная дата next_reward_on, которая двигается
// только в момент выдачи и ровно на длину текущего интервала повторения.
//
// Возвращает вид награды и дату, когда за тему можно будет получить
// следующую — её показываем ученику, чтобы отказ не выглядел произволом.
func (s *Store) ClaimTopicReward(userID int64, topicID string, today time.Time) (kind string, nextOn time.Time, err error) {
	tx, err := s.db.Begin()
	if err != nil {
		return RewardNone, time.Time{}, fmt.Errorf("ClaimTopicReward: begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var (
		box        int
		dueOn      time.Time
		nextReward sql.NullTime
	)
	row := tx.QueryRow(`
		SELECT box, due_on, next_reward_on FROM topic_mastery
		WHERE user_id=$1 AND topic_id=$2 FOR UPDATE`, userID, topicID)

	switch err := row.Scan(&box, &dueOn, &nextReward); {
	case err == sql.ErrNoRows:
		// Урок завершён, но ни одного ответа не записалось. Так ведёт себя
		// старый клиент, который ещё не умеет слать /api/attempt. Заводим
		// тему в первой коробке, чтобы прогресс не потерялся.
		dueOn = day(today).AddDate(0, 0, mastery.Intervals[0])
		if _, err := tx.Exec(`
			INSERT INTO topic_mastery
				(user_id, topic_id, box, streak_days, lapses, correct, total, due_on, first_done, next_reward_on)
			VALUES ($1,$2,1,0,0,0,0,$3,$4,$3)`,
			userID, topicID, dueOn, day(today)); err != nil {
			return RewardNone, time.Time{}, fmt.Errorf("ClaimTopicReward: insert: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return RewardNone, time.Time{}, fmt.Errorf("ClaimTopicReward: commit: %w", err)
		}
		return RewardFirst, dueOn, nil

	case err != nil:
		return RewardNone, time.Time{}, fmt.Errorf("ClaimTopicReward: %w", err)
	}

	switch {
	case !nextReward.Valid:
		kind = RewardFirst
	case !day(today).Before(day(nextReward.Time)):
		kind = RewardReview
	default:
		// Награда уже выдана в текущем интервале. Прогресс от повторного
		// прохождения при этом сохраняется — не сохраняется только оплата.
		return RewardNone, day(nextReward.Time), nil
	}

	// Новый срок считается от СЕГОДНЯ, а не берётся из due_on.
	//
	// due_on двигают ответы, а не завершение урока. Если ученик закрыл тему
	// заново, не ответив ни на что нового (или ответив на всё в прошлый
	// заход), due_on остался бы в прошлом — и награду можно было бы забирать
	// сколько угодно раз подряд. Ровно это и ловил тест.
	//
	// due_on учитывается, только если он дальше в будущее: тогда возвращать
	// ученика раньше срока незачем.
	nextOn = day(today).AddDate(0, 0, mastery.Intervals[boxIndex(box)])
	if day(dueOn).After(nextOn) {
		nextOn = day(dueOn)
	}

	if _, err := tx.Exec(`
		UPDATE topic_mastery SET next_reward_on=$3
		WHERE user_id=$1 AND topic_id=$2`, userID, topicID, nextOn); err != nil {
		return RewardNone, time.Time{}, fmt.Errorf("ClaimTopicReward: update: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return RewardNone, time.Time{}, fmt.Errorf("ClaimTopicReward: commit: %w", err)
	}
	return kind, nextOn, nil
}

// boxIndex переводит номер коробки в индекс таблицы интервалов, зажимая
// значение в допустимый диапазон: в базе может лежать что угодно, а выход
// за границы среза уронил бы процесс.
func boxIndex(box int) int {
	if box < 1 {
		return 0
	}
	if box > len(mastery.Intervals) {
		return len(mastery.Intervals) - 1
	}
	return box - 1
}

// day отбрасывает время, оставляя календарную дату.
func day(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}
