package store

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"strings"
	"unicode/utf8"

	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"

	"rootry/internal/models"
)

// GetUserByGoogleSub находит ученика по идентификатору аккаунта Google.
func (s *Store) GetUserByGoogleSub(sub string) (*models.User, bool) {
	row := s.db.QueryRow(userSelect+`
		WHERE id = (SELECT user_id FROM google_accounts WHERE google_sub=$1)`, sub)
	u, err := scanUser(row)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			// Чаще всего это значит, что не накатана migrations/008_google.sql.
			log.Printf("GetUserByGoogleSub: %v", err)
		}
		return nil, false
	}
	return u, true
}

// CreateGoogleUser заводит нового ученика для аккаунта Google.
//
// Логин придумывается из почты: ученик его не вводил, но он нужен
// рейтингу, классам и профилю. Пароль — случайный: войти по паролю
// такой ученик не может, только через Google.
func (s *Store) CreateGoogleUser(sub, email, name string) (*models.User, error) {
	if u, ok := s.GetUserByGoogleSub(sub); ok {
		return u, nil
	}

	nickname := googleNickname(name, email)
	base := usernameFromEmail(email)

	secret := make([]byte, 32)
	rand.Read(secret)
	hash, err := bcrypt.GenerateFromPassword([]byte(hex.EncodeToString(secret)), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	for attempt := 0; attempt < 10; attempt++ {
		username := base
		if attempt > 0 {
			username = withSuffix(base)
		}
		user, err := s.insertGoogleUser(sub, email, username, nickname, string(hash))
		if err == nil {
			return user, nil
		}
		if errors.Is(err, ErrUsernameTaken) {
			continue
		}
		// Тот же аккаунт Google мог зарегистрироваться параллельным
		// запросом (двойной клик) — тогда просто отдаём его.
		if u, ok := s.GetUserByGoogleSub(sub); ok {
			return u, nil
		}
		return nil, err
	}
	return nil, fmt.Errorf("create google user: no free username for %q", base)
}

func (s *Store) insertGoogleUser(sub, email, username, nickname, hash string) (*models.User, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var id int64
	err = tx.QueryRow(`
		INSERT INTO users
		  (username, nickname, password_hash, balance, xp, streak,
		   badges, avatars, active_avatar,
		   completed_topics, promo_used, favorite_games, games_won_types)
		VALUES ($1,$2,$3, 500,0,0, '[]','["cat"]','cat', '[]','[]','[]','[]')
		RETURNING id`,
		username, nickname, hash,
	).Scan(&id)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == uniqueViolation {
			return nil, ErrUsernameTaken
		}
		log.Printf("CreateGoogleUser(%s): %v", username, err)
		return nil, fmt.Errorf("create google user: %w", err)
	}

	if _, err := tx.Exec(`
		INSERT INTO google_accounts (google_sub, user_id, email) VALUES ($1,$2,$3)`,
		sub, id, email,
	); err != nil {
		log.Printf("CreateGoogleUser link(%s): %v", username, err)
		return nil, fmt.Errorf("link google account: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	user, ok := s.GetUserByUsername(username)
	if !ok {
		return nil, fmt.Errorf("failed to retrieve created user")
	}
	return user, nil
}

// usernameFromEmail превращает «Ivan.Petrov-2010@gmail.com» в
// «ivan_petrov_2010»: логин в RootRy — только латиница, цифры и _,
// от 3 до 20 символов. Хвост в 4 символа оставлен под суффикс.
func usernameFromEmail(email string) string {
	local, _, _ := strings.Cut(strings.ToLower(email), "@")
	var b strings.Builder
	for _, c := range local {
		switch {
		case (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9'):
			b.WriteRune(c)
		case c == '_' || c == '.' || c == '-' || c == '+':
			b.WriteByte('_')
		}
	}
	name := strings.Trim(b.String(), "_")
	if len(name) > 16 {
		name = strings.TrimRight(name[:16], "_")
	}
	if len(name) < 3 {
		name = "user" + name
	}
	return name
}

func withSuffix(base string) string {
	n := make([]byte, 2)
	rand.Read(n)
	return fmt.Sprintf("%s_%03d", base, (int(n[0])<<8|int(n[1]))%1000)
}

// googleNickname берёт имя из профиля Google, а без него — логин из почты.
func googleNickname(name, email string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		name, _, _ = strings.Cut(email, "@")
	}
	if utf8.RuneCountInString(name) > 32 {
		name = string([]rune(name)[:32])
	}
	if name == "" {
		name = "Ученик"
	}
	return name
}
