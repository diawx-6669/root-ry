package store

import (
	crand "crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"github.com/lib/pq"
	"rootry/internal/models"
)

// newSessionID возвращает непредсказуемый идентификатор сессии.
func newSessionID() (string, error) {
	var b [16]byte
	if _, err := crand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func intsToArray(v []int) pq.Int64Array {
	out := make(pq.Int64Array, len(v))
	for i, n := range v {
		out[i] = int64(n)
	}
	return out
}

func arrayToInts(v pq.Int64Array) []int {
	out := make([]int, len(v))
	for i, n := range v {
		out[i] = int(n)
	}
	return out
}

const kspoyaSelect = `
	SELECT id, username, question_ids, COALESCE(answers, '{}'),
	       started_at, expires_at, status,
	       COALESCE(raw_score,0), COALESCE(percent,0), COALESCE(level_key,'')
	FROM kspoya_sessions`

func scanKspoyaSession(row interface{ Scan(dest ...any) error }) (*models.KspoyaSession, error) {
	var s models.KspoyaSession
	var ids, answers pq.Int64Array
	err := row.Scan(&s.ID, &s.Username, &ids, &answers,
		&s.StartedAt, &s.ExpiresAt, &s.Status,
		&s.RawScore, &s.Percent, &s.LevelKey)
	if err != nil {
		return nil, err
	}
	s.QuestionIDs = arrayToInts(ids)
	s.Answers = arrayToInts(answers)
	return &s, nil
}

// expireKspoyaSessions переводит просроченные активные сессии в статус aborted,
// иначе частичный уникальный индекс не даст начать новую попытку.
func (s *Store) expireKspoyaSessions(username string) {
	_, err := s.db.Exec(`
		UPDATE kspoya_sessions
		SET status = 'aborted', finished_at = now()
		WHERE username = $1 AND status = 'active' AND expires_at <= now()`, username)
	if err != nil {
		log.Printf("expireKspoyaSessions(%s): %v", username, err)
	}
}

// ActiveKspoyaSession возвращает незавершённую попытку пользователя, если она есть.
func (s *Store) ActiveKspoyaSession(username string) (*models.KspoyaSession, bool) {
	row := s.db.QueryRow(kspoyaSelect+`
		WHERE username = $1 AND status = 'active' AND expires_at > now()`, username)
	session, err := scanKspoyaSession(row)
	if err != nil {
		return nil, false
	}
	return session, true
}

// StartKspoyaSession создаёт новую попытку. Если у пользователя уже есть
// активная и не просроченная сессия, возвращает её — это позволяет продолжить
// тест после перезагрузки страницы, не выдавая новый набор вопросов.
func (s *Store) StartKspoyaSession(username string, questionIDs []int, ttl time.Duration) (*models.KspoyaSession, error) {
	s.expireKspoyaSessions(username)

	if existing, ok := s.ActiveKspoyaSession(username); ok {
		return existing, nil
	}

	id, err := newSessionID()
	if err != nil {
		return nil, err
	}

	row := s.db.QueryRow(`
		INSERT INTO kspoya_sessions (id, username, question_ids, expires_at)
		VALUES ($1, $2, $3::integer[], now() + $4::interval)
		RETURNING id, username, question_ids, '{}'::integer[],
		          started_at, expires_at, status, 0, 0, ''`,
		id, username, intsToArray(questionIDs),
		fmt.Sprintf("%d seconds", int(ttl.Seconds())),
	)
	session, err := scanKspoyaSession(row)
	if err != nil {
		// Гонка: параллельный запрос успел создать сессию раньше — отдаём её.
		if existing, ok := s.ActiveKspoyaSession(username); ok {
			return existing, nil
		}
		return nil, err
	}
	return session, nil
}

// GetKspoyaSession читает сессию по идентификатору и владельцу.
func (s *Store) GetKspoyaSession(id, username string) (*models.KspoyaSession, bool) {
	row := s.db.QueryRow(kspoyaSelect+` WHERE id = $1 AND username = $2`, id, username)
	session, err := scanKspoyaSession(row)
	if err != nil {
		return nil, false
	}
	return session, true
}

// FinishKspoyaSession закрывает попытку и сохраняет ответы, чтобы разбор можно
// было открыть позже. Возвращает true только если сессия действительно была
// активной — благодаря этому повторная отправка не начислит награду дважды.
func (s *Store) FinishKspoyaSession(id, username string, raw, percent int, level string, answers []int) bool {
	res, err := s.db.Exec(`
		UPDATE kspoya_sessions
		SET status = 'completed', raw_score = $1, percent = $2,
		    level_key = $3, answers = $4::integer[], finished_at = now()
		WHERE id = $5 AND username = $6 AND status = 'active'
		  AND expires_at + interval '2 minutes' > now()`,
		raw, percent, level, intsToArray(answers), id, username,
	)
	if err != nil {
		log.Printf("FinishKspoyaSession(%s): %v", id, err)
		return false
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false
	}
	return n == 1
}

// AbortKspoyaSession отменяет активную попытку (античит, отказ от теста).
func (s *Store) AbortKspoyaSession(id, username string) {
	var err error
	if id == "" {
		_, err = s.db.Exec(`
			UPDATE kspoya_sessions SET status = 'aborted', finished_at = now()
			WHERE username = $1 AND status = 'active'`, username)
	} else {
		_, err = s.db.Exec(`
			UPDATE kspoya_sessions SET status = 'aborted', finished_at = now()
			WHERE id = $1 AND username = $2 AND status = 'active'`, id, username)
	}
	if err != nil {
		log.Printf("AbortKspoyaSession(%s): %v", username, err)
	}
}

// bestAttemptSQL — лучшая попытка пользователя: наибольший балл, а при равном
// балле та, что завершена раньше.
const bestAttemptSQL = `
	SELECT id, raw_score, level_key, finished_at,
	       COALESCE(array_length(question_ids, 1), 0) AS total
	FROM kspoya_sessions
	WHERE username = u.username AND status = 'completed' AND level_key <> ''
	ORDER BY raw_score DESC, finished_at ASC
	LIMIT 1`

// KspoyaLeaderboard возвращает рейтинг по лучшей попытке каждого ученика.
// Администраторы в рейтинг не попадают. При равном балле выше тот,
// кто прошёл тест раньше.
func (s *Store) KspoyaLeaderboard(limit int) []models.KspoyaLeaderEntry {
	rows, err := s.db.Query(`
		SELECT u.username, u.nickname, best.raw_score, best.total,
		       best.level_key, best.finished_at
		FROM users u
		JOIN LATERAL (`+bestAttemptSQL+`) best ON TRUE
		WHERE u.is_admin = FALSE
		ORDER BY best.raw_score DESC, best.finished_at ASC
		LIMIT $1`, limit)
	if err != nil {
		log.Printf("KspoyaLeaderboard: %v", err)
		return nil
	}
	defer rows.Close()

	var out []models.KspoyaLeaderEntry
	rank := 1
	for rows.Next() {
		var e models.KspoyaLeaderEntry
		if err := rows.Scan(&e.Username, &e.Nickname, &e.Score,
			&e.Total, &e.Level, &e.FinishedAt); err != nil {
			continue
		}
		e.Rank = rank
		rank++
		out = append(out, e)
	}
	return out
}

// BestKspoyaAttemptID возвращает идентификатор лучшей попытки пользователя.
func (s *Store) BestKspoyaAttemptID(username string) (id string, score int, level string, ok bool) {
	row := s.db.QueryRow(`
		SELECT id, raw_score, level_key
		FROM kspoya_sessions
		WHERE username = $1 AND status = 'completed' AND level_key <> ''
		ORDER BY raw_score DESC, finished_at ASC
		LIMIT 1`, username)
	if err := row.Scan(&id, &score, &level); err != nil {
		return "", 0, "", false
	}
	return id, score, level, true
}

// ── Античит ──────────────────────────────────────────────────────────────

// KspoyaBanState читает счётчик нарушений и срок блокировки.
func (s *Store) KspoyaBanState(username string) models.KspoyaBanState {
	var state models.KspoyaBanState
	var until sql.NullTime
	err := s.db.QueryRow(`
		SELECT COALESCE(kspoya_warnings, 0), kspoya_ban_until
		FROM users WHERE username = $1`, username).Scan(&state.Warnings, &until)
	if err != nil {
		log.Printf("KspoyaBanState(%s): %v", username, err)
		return state
	}
	if until.Valid && until.Time.After(time.Now()) {
		t := until.Time
		state.BannedUntil = &t
	}
	return state
}

// RegisterKspoyaViolation засчитывает нарушение античита. Когда нарушений
// набирается maxStrikes, пользователь блокируется на banFor, а счётчик
// обнуляется — чтобы после снятия блокировки отсчёт начинался заново.
func (s *Store) RegisterKspoyaViolation(username string, maxStrikes int, banFor time.Duration) models.KspoyaBanState {
	state := models.KspoyaBanState{MaxStrikes: maxStrikes}
	var until sql.NullTime

	err := s.db.QueryRow(`
		UPDATE users SET
		  kspoya_warnings = CASE
		      WHEN COALESCE(kspoya_warnings, 0) + 1 >= $2 THEN 0
		      ELSE COALESCE(kspoya_warnings, 0) + 1 END,
		  kspoya_ban_until = CASE
		      WHEN COALESCE(kspoya_warnings, 0) + 1 >= $2 THEN now() + $3::interval
		      ELSE kspoya_ban_until END
		WHERE username = $1
		RETURNING kspoya_warnings, kspoya_ban_until`,
		username, maxStrikes, fmt.Sprintf("%d seconds", int(banFor.Seconds())),
	).Scan(&state.Warnings, &until)
	if err != nil {
		log.Printf("RegisterKspoyaViolation(%s): %v", username, err)
		return state
	}
	if until.Valid && until.Time.After(time.Now()) {
		t := until.Time
		state.BannedUntil = &t
	}
	return state
}

// ListKspoyaAttempts возвращает завершённые попытки, новые сверху.
func (s *Store) ListKspoyaAttempts(username string, limit int) []models.KspoyaAttempt {
	rows, err := s.db.Query(`
		SELECT id, finished_at, COALESCE(level_key,''),
		       COALESCE(raw_score,0), COALESCE(percent,0),
		       COALESCE(array_length(question_ids, 1), 0),
		       COALESCE(array_length(answers, 1), 0) > 0
		FROM kspoya_sessions
		WHERE username = $1 AND status = 'completed' AND level_key <> ''
		ORDER BY finished_at DESC
		LIMIT $2`, username, limit)
	if err != nil {
		log.Printf("ListKspoyaAttempts(%s): %v", username, err)
		return nil
	}
	defer rows.Close()

	var out []models.KspoyaAttempt
	for rows.Next() {
		var a models.KspoyaAttempt
		if err := rows.Scan(&a.ID, &a.FinishedAt, &a.Level,
			&a.Correct, &a.Percent, &a.Total, &a.HasReview); err != nil {
			continue
		}
		out = append(out, a)
	}
	return out
}
