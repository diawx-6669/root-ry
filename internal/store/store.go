package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
	"rootry/internal/models"
	"rootry/internal/timeutil"
)

// uniqueViolation — код ошибки PostgreSQL при нарушении UNIQUE-ограничения.
const uniqueViolation = "23505"

// ErrUsernameTaken возвращается, когда логин уже занят. Все прочие ошибки
// вставки — это авария, и обработчик должен отвечать 500, а не 409.
var ErrUsernameTaken = errors.New("username taken")

// Store wraps a *sql.DB and exposes the same interface the handlers expect.
type Store struct {
	db *sql.DB
}

// ── Constructor ───────────────────────────────────────────────────────────────

func New() *Store {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL env var is not set")
	}
	s, err := NewWithDSN(dsn)
	if err != nil {
		log.Fatalf("store: %v", err)
	}
	return s
}

// NewWithDSN открывает хранилище по явной строке подключения.
//
// Отдельно от New, потому что New обязан валить процесс: приложение без
// базы бессмысленно. Тестам же нужно подключиться к своей базе и получить
// ошибку, а не убитый процесс.
func NewWithDSN(dsn string) (*Store, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("db.Ping: %w", err)
	}
	s := &Store{db: db}
	s.seedPromos()
	s.seedDemo()
	return s, nil
}

// Close закрывает пул соединений. Нужен тестам, которые поднимают
// несколько хранилищ подряд.
func (s *Store) Close() error { return s.db.Close() }

// ── Internal helpers ──────────────────────────────────────────────────────────

// jsonbCol marshals a Go slice into a JSON string for a JSONB column.
func jsonbCol(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// scanUser reads one row from the users table into a models.User.
// The SELECT must follow the column order defined in userColumns.
func scanUser(row interface {
	Scan(dest ...any) error
}) (*models.User, error) {
	var u models.User
	var (
		badges, avatars, completedTopics []byte
		promoUsed, favoriteGames         []byte
		gamesWonTypes                    []byte
		lastLogin, lastNickChange        sql.NullString
		dailyTasksDate, lastDailyClaim   sql.NullString
		studyGroup                       sql.NullString
	)
	err := row.Scan(
		&u.ID, &u.Username, &u.Nickname, &u.PasswordHash,
		&u.Balance, &u.XP, &u.Streak,
		&lastLogin, &lastNickChange,
		&u.IsAdmin,
		&badges, &avatars, &u.ActiveAvatar,
		&completedTopics, &promoUsed, &favoriteGames,
		&dailyTasksDate, &u.DailyTasksDone,
		&u.GamesWonToday, &gamesWonTypes,
		&lastDailyClaim, &u.CreatedAt,
		&u.IsTeacher, &u.ResearchConsent, &studyGroup,
	)
	if err != nil {
		return nil, err
	}
	u.LastLogin = lastLogin.String
	u.LastNickChange = lastNickChange.String
	u.DailyTasksDate = dailyTasksDate.String
	u.LastDailyClaim = lastDailyClaim.String
	u.StudyGroup = studyGroup.String

	// Unmarshal JSONB arrays
	json.Unmarshal(badges, &u.Badges)
	json.Unmarshal(avatars, &u.Avatars)

	// Аккаунты, заведённые до перехода на SVG, хранят аватарки и значки
	// эмодзи. Приводим их к новым идентификаторам здесь — так остальному
	// коду не нужно знать о старом формате (см. legacy.go).
	u.Badges = NormalizeBadges(u.Badges)
	u.Avatars = NormalizeAvatars(u.Avatars)
	u.ActiveAvatar = NormalizeAvatar(u.ActiveAvatar)
	json.Unmarshal(completedTopics, &u.CompletedTopics)
	json.Unmarshal(promoUsed, &u.PromoUsed)
	json.Unmarshal(favoriteGames, &u.FavoriteGames)
	json.Unmarshal(gamesWonTypes, &u.GamesWonTypes)

	// Ensure nil slices become empty slices (cleaner JSON output)
	if u.Badges == nil {
		u.Badges = []string{}
	}
	if u.Avatars == nil {
		u.Avatars = []string{"cat"}
	}
	if u.CompletedTopics == nil {
		u.CompletedTopics = []string{}
	}
	if u.PromoUsed == nil {
		u.PromoUsed = []string{}
	}
	if u.FavoriteGames == nil {
		u.FavoriteGames = []string{}
	}
	if u.GamesWonTypes == nil {
		u.GamesWonTypes = []string{}
	}
	return &u, nil
}

const userSelect = `
	SELECT id, username, nickname, password_hash,
	       balance, xp, streak,
	       to_char(last_login,'YYYY-MM-DD'), to_char(last_nick_change,'YYYY-MM-DD'),
	       is_admin,
	       badges, avatars, active_avatar,
	       completed_topics, promo_used, favorite_games,
	       to_char(daily_tasks_date,'YYYY-MM-DD'), daily_tasks_done,
	       games_won_today, games_won_types,
	       to_char(last_daily_claim,'YYYY-MM-DD'), created_at,
	       is_teacher, research_consent, study_group
	FROM users`

// ── Seeds ─────────────────────────────────────────────────────────────────────

func (s *Store) seedDemo() {
	var exists bool
	s.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM users WHERE username=$1)`, "demo").Scan(&exists)
	if exists {
		return
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte("demo123"), bcrypt.DefaultCost)
	_, err := s.db.Exec(`
		INSERT INTO users
		  (username, nickname, password_hash, balance, xp, streak,
		   badges, avatars, active_avatar, completed_topics, promo_used,
		   favorite_games, games_won_types, is_admin)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		"demo", "Демо Игрок", string(hash), 1500, 420, 7,
		`["tree","star"]`, `["cat"]`, "cat",
		`[]`, `[]`, `[]`, `[]`, false,
	)
	if err != nil {
		log.Printf("seedDemo: %v", err)
	}
}

func (s *Store) seedPromos() {
	adminPromo := os.Getenv("ADMIN_PROMO")
	if adminPromo == "" {
		adminPromo = "ADMIN240411"
	}
	promos := []struct {
		code, reward string
		value, uses  int
	}{
		{"MEGACOINS", "coins", 10_000_000, -1},
		{"777", "coins", 10_000_000, -1},
		{adminPromo, "admin", 0, 10},
	}
	for _, p := range promos {
		_, err := s.db.Exec(`
			INSERT INTO promos (code, reward, value, uses)
			VALUES ($1,$2,$3,$4)
			ON CONFLICT (code) DO NOTHING`,
			p.code, p.reward, p.value, p.uses,
		)
		if err != nil {
			log.Printf("seedPromos %s: %v", p.code, err)
		}
	}
}

// ── Public API (same signatures as the old in-memory Store) ───────────────────

func (s *Store) CreateUser(username, nickname, password string) (*models.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	row := s.db.QueryRow(`
		INSERT INTO users
		  (username, nickname, password_hash, balance, xp, streak,
		   badges, avatars, active_avatar,
		   completed_topics, promo_used, favorite_games, games_won_types)
		VALUES ($1,$2,$3, 500,0,0, '[]','["cat"]','cat', '[]','[]','[]','[]')
		RETURNING id`,
		username, nickname, string(hash),
	)
	var id int64
	if err := row.Scan(&id); err != nil {
		// Занятый логин и упавшая база — разные вещи. Раньше любая ошибка
		// вставки превращалась в «Логин уже занят», и настоящая авария базы
		// выглядела для ученика как обычная опечатка в логине.
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == uniqueViolation {
			return nil, ErrUsernameTaken
		}
		log.Printf("CreateUser(%s): %v", username, err)
		return nil, fmt.Errorf("create user: %w", err)
	}

	user, ok := s.GetUserByUsername(username)
	if !ok {
		return nil, fmt.Errorf("failed to retrieve created user")
	}
	return user, nil
}

func (s *Store) GetUserByUsername(username string) (*models.User, bool) {
	row := s.db.QueryRow(userSelect+` WHERE username=$1`, username)
	u, err := scanUser(row)
	if err != nil {
		return nil, false
	}
	return u, true
}

func (s *Store) ValidatePassword(username, password string) (*models.User, bool) {
	u, ok := s.GetUserByUsername(username)
	if !ok {
		return nil, false
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) != nil {
		return nil, false
	}
	return u, true
}

// UpdateUser writes all mutable user fields back to Postgres.
func (s *Store) UpdateUser(u *models.User) {
	_, err := s.db.Exec(`
		UPDATE users SET
		  nickname         = $1,
		  balance          = $2,
		  xp               = $3,
		  streak           = $4,
		  last_login       = NULLIF($5,'')::DATE,
		  last_nick_change = NULLIF($6,'')::DATE,
		  is_admin         = $7,
		  badges           = $8::JSONB,
		  avatars          = $9::JSONB,
		  active_avatar    = $10,
		  completed_topics = $11::JSONB,
		  promo_used       = $12::JSONB,
		  favorite_games   = $13::JSONB,
		  daily_tasks_date = NULLIF($14,'')::DATE,
		  daily_tasks_done = $15,
		  games_won_today  = $16,
		  games_won_types  = $17::JSONB,
		  last_daily_claim = NULLIF($18,'')::DATE
		WHERE username = $19`,
		u.Nickname,
		u.Balance,
		u.XP,
		u.Streak,
		u.LastLogin,
		u.LastNickChange,
		u.IsAdmin,
		jsonbCol(u.Badges),
		jsonbCol(u.Avatars),
		u.ActiveAvatar,
		jsonbCol(u.CompletedTopics),
		jsonbCol(u.PromoUsed),
		jsonbCol(u.FavoriteGames),
		u.DailyTasksDate,
		u.DailyTasksDone,
		u.GamesWonToday,
		jsonbCol(u.GamesWonTypes),
		u.LastDailyClaim,
		u.Username,
	)
	if err != nil {
		log.Printf("UpdateUser(%s): %v", u.Username, err)
	}
}

// GetLeaderboard отдаёт общий рейтинг. Лучший результат КСПОЯ подтягивается
// отдельным подзапросом: наибольший балл, при равенстве — более ранняя попытка.
// localZone — часовой пояс, по которому считается «сегодня».
// Сервер живёт в UTC, а ученики в Казахстане: без этого день сменялся бы
// в пять утра по местному времени. Тот же пояс использует Go-код через
// пакет timeutil — «сегодня» во всём проекте должно быть одним и тем же.
const localZone = timeutil.Zone

// TouchDailyLogin отмечает заход за сегодня: продлевает серию, если вчера
// заход был, иначе начинает её заново.
//
// Вызывается не только при вводе логина и пароля, но и при любом обращении
// к /api/me. Токен живёт 30 дней, поэтому вернувшийся ученик страницу входа
// обычно не открывает — раньше из-за этого серия навсегда застревала на 1.
//
// Монеты здесь БОЛЬШЕ НЕ начисляются: за ежедневный бонус отвечает только
// /api/daily/claim. Раньше бонусов было два — молчаливый +10 за любой заход
// и кнопка в магазине, — и ученик получал за день вдвое больше, чем показывал
// интерфейс.
//
// Всё делается одним UPDATE: условие в WHERE гарантирует, что серия
// продлится ровно один раз за день, даже если запросы придут параллельно.
func (s *Store) TouchDailyLogin(username string) (streak int, updated bool) {
	row := s.db.QueryRow(`
		UPDATE users SET
		  streak = CASE
		      WHEN last_login = ((now() AT TIME ZONE $2)::date - 1) THEN streak + 1
		      ELSE 1
		  END,
		  last_login = (now() AT TIME ZONE $2)::date
		WHERE username = $1
		  AND last_login IS DISTINCT FROM (now() AT TIME ZONE $2)::date
		RETURNING streak`,
		username, localZone,
	)
	if err := row.Scan(&streak); err != nil {
		if err != sql.ErrNoRows {
			log.Printf("TouchDailyLogin(%s): %v", username, err)
		}
		return 0, false // сегодня уже отмечались
	}
	return streak, true
}

// LeaderboardLimit — сколько строк отдаёт рейтинг. Без ограничения запрос
// возвращал вообще всех зарегистрированных, и с ростом числа учеников
// страница рейтинга тянула бы всю таблицу пользователей целиком.
const LeaderboardLimit = 200

func (s *Store) GetLeaderboard() []models.LeaderboardEntry {
	rows, err := s.db.Query(`
		SELECT u.username, u.nickname, u.xp, u.balance,
		       jsonb_array_length(u.badges), u.streak, u.active_avatar,
		       COALESCE(best.raw_score, 0), COALESCE(best.level_key, '')
		FROM users u
		LEFT JOIN LATERAL (
		    SELECT raw_score, level_key
		    FROM kspoya_sessions
		    WHERE username = u.username AND status = 'completed' AND level_key <> ''
		    ORDER BY raw_score DESC, finished_at ASC
		    LIMIT 1
		) best ON TRUE
		WHERE u.is_admin = FALSE
		ORDER BY u.xp DESC, u.username ASC
		LIMIT $1`, LeaderboardLimit)
	if err != nil {
		log.Printf("GetLeaderboard: %v", err)
		return nil
	}
	defer rows.Close()
	var entries []models.LeaderboardEntry
	rank := 1
	for rows.Next() {
		var e models.LeaderboardEntry
		if err := rows.Scan(&e.Username, &e.Nickname, &e.XP, &e.Balance, &e.Badges, &e.Streak,
			&e.ActiveAvatar, &e.KspoyaScore, &e.KspoyaLevel); err != nil {
			log.Printf("GetLeaderboard scan: %v", err)
			continue
		}
		// Рейтинг читает active_avatar прямо из SQL, минуя GetUser, где
		// старые эмодзи переводятся в идентификаторы. Без этой строки
		// аккаунты, заведённые до перехода на SVG, отдавали бы в рейтинг
		// «🐱» — и картинку пришлось бы чинить на клиенте.
		e.ActiveAvatar = NormalizeAvatar(e.ActiveAvatar)
		e.Rank = rank
		rank++
		entries = append(entries, e)
	}
	// Без этой проверки оборванное соединение выглядело бы как «рейтинг
	// закончился»: ученик увидел бы усечённый список и ничего об этом не узнал.
	if err := rows.Err(); err != nil {
		log.Printf("GetLeaderboard rows: %v", err)
	}
	return entries
}

// ── Promo codes ───────────────────────────────────────────────────────────────

func (s *Store) GetPromo(code string) (*models.PromoCode, bool) {
	row := s.db.QueryRow(`
		SELECT code, reward, value, badge_name, avatar_name, uses, used_count
		FROM promos WHERE code=$1`, code)
	var p models.PromoCode
	err := row.Scan(&p.Code, &p.Reward, &p.Value, &p.BadgeName, &p.AvatarName, &p.Uses, &p.UsedCount)
	if err != nil {
		return nil, false
	}
	return &p, true
}

func (s *Store) UsePromo(code string) {
	_, err := s.db.Exec(`UPDATE promos SET used_count = used_count+1 WHERE code=$1`, code)
	if err != nil {
		log.Printf("UsePromo(%s): %v", code, err)
	}
}

// ── Result logging ────────────────────────────────────────────────────────────

func (s *Store) SaveGameResult(r models.GameResult) {
	_, err := s.db.Exec(`
		INSERT INTO game_results (user_id, game_type, score, xp_earned, coins_earned)
		VALUES ($1,$2,$3,$4,$5)`,
		r.UserID, r.GameType, r.Score, r.XPEarned, r.CoinsEarned,
	)
	if err != nil {
		log.Printf("SaveGameResult: %v", err)
	}
}

func (s *Store) SaveTestResult(r models.TestResult) {
	_, err := s.db.Exec(`
		INSERT INTO test_results (user_id, score, passed, level, badge_earned)
		VALUES ($1,$2,$3,$4,$5)`,
		r.UserID, r.Score, r.Passed, r.Level, r.BadgeEarned,
	)
	if err != nil {
		log.Printf("SaveTestResult: %v", err)
	}
}

func (s *Store) SaveCaseResult(r models.CaseResult) {
	_, err := s.db.Exec(`
		INSERT INTO case_results (user_id, case_type, item_emoji, item_rarity, is_duplicate, compensation)
		VALUES ($1,$2,$3,$4,$5,$6)`,
		r.UserID, r.CaseType, r.ItemEmoji, r.ItemRarity, r.IsDuplicate, r.Compensation,
	)
	if err != nil {
		log.Printf("SaveCaseResult: %v", err)
	}
}

// ── Admin helpers ─────────────────────────────────────────────────────────────

func (s *Store) GetAllUsers() []*models.User {
	rows, err := s.db.Query(userSelect + ` ORDER BY id`)
	if err != nil {
		log.Printf("GetAllUsers: %v", err)
		return nil
	}
	defer rows.Close()
	var out []*models.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			log.Printf("GetAllUsers scan: %v", err)
			continue
		}
		out = append(out, u)
	}
	if err := rows.Err(); err != nil {
		log.Printf("GetAllUsers rows: %v", err)
	}
	return out
}

func (s *Store) GetGameResults() []models.GameResult {
	rows, err := s.db.Query(`
		SELECT user_id, game_type, score, xp_earned, coins_earned,
		       to_char(played_at,'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		FROM game_results ORDER BY played_at DESC`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []models.GameResult
	for rows.Next() {
		var r models.GameResult
		if err := rows.Scan(&r.UserID, &r.GameType, &r.Score, &r.XPEarned, &r.CoinsEarned, &r.PlayedAt); err != nil {
			log.Printf("GetGameResults scan: %v", err)
			continue
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		log.Printf("GetGameResults rows: %v", err)
	}
	return out
}

func (s *Store) GetCaseResults() []models.CaseResult {
	rows, err := s.db.Query(`
		SELECT user_id, case_type, item_emoji, item_rarity, is_duplicate, compensation,
		       to_char(played_at,'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		FROM case_results ORDER BY played_at DESC`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []models.CaseResult
	for rows.Next() {
		var r models.CaseResult
		if err := rows.Scan(&r.UserID, &r.CaseType, &r.ItemEmoji, &r.ItemRarity,
			&r.IsDuplicate, &r.Compensation, &r.PlayedAt); err != nil {
			log.Printf("GetCaseResults scan: %v", err)
			continue
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		log.Printf("GetCaseResults rows: %v", err)
	}
	return out
}

// ── Pure utility functions (no DB) ───────────────────────────────────────────

func EscapeHTML(s string) string {
	result := ""
	for _, c := range s {
		switch c {
		case '<':
			result += "&lt;"
		case '>':
			result += "&gt;"
		case '&':
			result += "&amp;"
		case '"':
			result += "&quot;"
		case '\'':
			result += "&#39;"
		default:
			result += string(c)
		}
	}
	return result
}

func Itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	var buf [20]byte
	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}

func CompensationForRarity(rarity string) int {
	switch rarity {
	case "common":
		return 100
	case "rare":
		return 250
	case "epic":
		return 500
	case "legendary":
		return 2500
	case "mythic":
		return 10000
	default:
		return 100
	}
}

func XPForRarity(rarity string) int {
	switch rarity {
	case "common":
		return 1
	case "rare":
		return 5
	case "epic":
		return 10
	case "legendary":
		return 20
	case "mythic":
		return 100
	default:
		return 1
	}
}

func hasBadge(badges []string, b string) bool {
	for _, v := range badges {
		if v == b {
			return true
		}
	}
	return false
}

func hasAvatar(avatars []string, a string) bool {
	for _, v := range avatars {
		if v == a {
			return true
		}
	}
	return false
}

func HasBadge(badges []string, b string) bool   { return hasBadge(badges, b) }
func HasAvatar(avatars []string, a string) bool { return hasAvatar(avatars, a) }
