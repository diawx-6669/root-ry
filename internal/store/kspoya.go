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

const kspoyaSelect = `
	SELECT id, username, question_ids, started_at, expires_at, status,
	       COALESCE(raw_score,0), COALESCE(percent,0), COALESCE(level_key,'')
	FROM kspoya_sessions`

func scanKspoyaSession(row interface{ Scan(dest ...any) error }) (*models.KspoyaSession, error) {
	var s models.KspoyaSession
	var ids pq.Int64Array
	err := row.Scan(&s.ID, &s.Username, &ids, &s.StartedAt, &s.ExpiresAt,
		&s.Status, &s.RawScore, &s.Percent, &s.LevelKey)
	if err != nil {
		return nil, err
	}
	s.QuestionIDs = make([]int, len(ids))
	for i, v := range ids {
		s.QuestionIDs[i] = int(v)
	}
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
	ids := make(pq.Int64Array, len(questionIDs))
	for i, v := range questionIDs {
		ids[i] = int64(v)
	}

	row := s.db.QueryRow(`
		INSERT INTO kspoya_sessions (id, username, question_ids, expires_at)
		VALUES ($1, $2, $3::integer[], now() + $4::interval)
		RETURNING id, username, question_ids, started_at, expires_at, status, 0, 0, ''`,
		id, username, ids, fmt.Sprintf("%d seconds", int(ttl.Seconds())),
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

// FinishKspoyaSession закрывает попытку. Возвращает true только если сессия
// действительно была активной — благодаря этому повторная отправка тех же
// ответов не начислит награду второй раз.
func (s *Store) FinishKspoyaSession(id, username string, raw, percent int, level string) bool {
	res, err := s.db.Exec(`
		UPDATE kspoya_sessions
		SET status = 'completed', raw_score = $1, percent = $2,
		    level_key = $3, finished_at = now()
		WHERE id = $4 AND username = $5 AND status = 'active'
		  AND expires_at + interval '2 minutes' > now()`,
		raw, percent, level, id, username,
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

// BestKspoyaLevel возвращает лучший подтверждённый уровень пользователя
// и число завершённых попыток.
func (s *Store) BestKspoyaLevel(username string) (levels []string, attempts int) {
	rows, err := s.db.Query(`
		SELECT level_key FROM kspoya_sessions
		WHERE username = $1 AND status = 'completed' AND level_key <> ''`, username)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Printf("BestKspoyaLevel(%s): %v", username, err)
		}
		return nil, 0
	}
	defer rows.Close()
	for rows.Next() {
		var level string
		if rows.Scan(&level) == nil {
			levels = append(levels, level)
			attempts++
		}
	}
	return levels, attempts
}
