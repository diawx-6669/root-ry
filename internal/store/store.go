package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
	"rootry/internal/models"
)

// Store wraps a *sql.DB and exposes the same interface the handlers expect.
type Store struct {
	db *sql.DB
}

// ── Constructor ───────────────────────────────────────────────────────────────

func New(_ string) *Store {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL env var is not set")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("store: sql.Open: %v", err)
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	if err := db.Ping(); err != nil {
		log.Fatalf("store: db.Ping: %v", err)
	}
	s := &Store{db: db}
	s.seedPromos()
	s.seedDemo()
	return s
}

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
	)
	if err != nil {
		return nil, err
	}
	u.LastLogin = lastLogin.String
	u.LastNickChange = lastNickChange.String
	u.DailyTasksDate = dailyTasksDate.String
	u.LastDailyClaim = lastDailyClaim.String

	// Unmarshal JSONB arrays
	json.Unmarshal(badges, &u.Badges)
	json.Unmarshal(avatars, &u.Avatars)
	json.Unmarshal(completedTopics, &u.CompletedTopics)
	json.Unmarshal(promoUsed, &u.PromoUsed)
	json.Unmarshal(favoriteGames, &u.FavoriteGames)
	json.Unmarshal(gamesWonTypes, &u.GamesWonTypes)

	// Ensure nil slices become empty slices (cleaner JSON output)
	if u.Badges == nil {
		u.Badges = []string{}
	}
	if u.Avatars == nil {
		u.Avatars = []string{"🐱"}
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
	       to_char(last_daily_claim,'YYYY-MM-DD'), created_at
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
		`["🎓","⭐"]`, `["🐱"]`, "🐱",
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
		VALUES ($1,$2,$3, 500,0,0, '[]','["🐱"]','🐱', '[]','[]','[]','[]')
		RETURNING id`,
		username, nickname, string(hash),
	)
	var id int64
	if err := row.Scan(&id); err != nil {
		// unique violation → conflict
		return nil, fmt.Errorf("conflict")
	}

	// ИСПРАВЛЕННЫЙ БЛОК: правильно обрабатываем true/false и превращаем в ошибку
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
func (s *Store) GetLeaderboard() []models.LeaderboardEntry {
	rows, err := s.db.Query(`
		SELECT u.username, u.nickname, u.xp, u.balance,
		       jsonb_array_length(u.badges), u.streak,
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
		ORDER BY u.xp DESC`)
	if err != nil {
		log.Printf("GetLeaderboard: %v", err)
		return nil
	}
	defer rows.Close()
	var entries []models.LeaderboardEntry
	rank := 1
	for rows.Next() {
		var e models.LeaderboardEntry
		rows.Scan(&e.Username, &e.Nickname, &e.XP, &e.Balance, &e.Badges, &e.Streak,
			&e.KspoyaScore, &e.KspoyaLevel)
		e.Rank = rank
		rank++
		entries = append(entries, e)
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
		if err == nil {
			out = append(out, u)
		}
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
		rows.Scan(&r.UserID, &r.GameType, &r.Score, &r.XPEarned, &r.CoinsEarned, &r.PlayedAt)
		out = append(out, r)
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
		rows.Scan(&r.UserID, &r.CaseType, &r.ItemEmoji, &r.ItemRarity,
			&r.IsDuplicate, &r.Compensation, &r.PlayedAt)
		out = append(out, r)
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

func RollCase(caseType string, userAvatars []string, userBadges []string, isBadgeCase bool) (string, string, bool) {
	seed := uint64(time.Now().UnixNano())
	seed ^= seed >> 33
	seed *= 0xff51afd7ed558ccd
	seed ^= seed >> 33

	r := float64(seed%10000) / 100.0

	if isBadgeCase {
		badgePool := map[string][]string{
			"common":    {"📚", "✏️", "📝", "🎒"},
			"rare":      {"⭐", "🔥", "💡"},
			"epic":      {"🏆", "💎"},
			"legendary": {"👑"},
		}
		var rarity string
		switch {
		case r < 40:
			rarity = "common"
		case r < 70:
			rarity = "rare"
		case r < 90:
			rarity = "epic"
		default:
			rarity = "legendary"
		}
		pool := badgePool[rarity]
		idx := int(seed>>8) % len(pool)
		item := pool[idx]
		return item, rarity, hasBadge(userBadges, item)
	}

	avatarPool := map[string][]string{
		"common":    {"🐱", "🐶", "🦊", "🐼", "🐨", "🦁", "🐯", "🐻", "🐸"},
		"rare":      {"🦄", "🐉", "🦋", "🦚", "🦜", "🦩", "🐬"},
		"epic":      {"🧙", "🧛", "🧜", "🧝", "🦸"},
		"legendary": {"👑", "🌟", "💫"},
		"mythic":    {"🌈"},
	}
	chances := map[string]map[string]float64{
		"common":    {"common": 70, "rare": 18, "epic": 10, "legendary": 2, "mythic": 0},
		"rare":      {"common": 40, "rare": 42.5, "epic": 12.5, "legendary": 4.5, "mythic": 0.5},
		"epic":      {"common": 20, "rare": 52.5, "epic": 22.5, "legendary": 9, "mythic": 1},
		"legendary": {"common": 0, "rare": 20, "epic": 40, "legendary": 35, "mythic": 5},
	}
	order := []string{"common", "rare", "epic", "legendary", "mythic"}
	ch := chances[caseType]
	if ch == nil {
		ch = chances["common"]
	}
	rarity := "common"
	cum := 0.0
	for _, rar := range order {
		cum += ch[rar]
		if r < cum {
			rarity = rar
			break
		}
	}
	pool := avatarPool[rarity]
	idx := int(seed>>8) % len(pool)
	item := pool[idx]
	return item, rarity, hasAvatar(userAvatars, item)
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
