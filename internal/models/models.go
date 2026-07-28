package models

import "time"

type User struct {
	ID              int64     `json:"id"`
	Username        string    `json:"username"`
	Nickname        string    `json:"nickname"`
	PasswordHash    string    `json:"-"`
	Balance         int       `json:"balance"`
	XP              int       `json:"xp"`
	Streak          int       `json:"streak"`
	LastLogin       string    `json:"last_login"`
	LastNickChange  string    `json:"last_nick_change"`
	IsAdmin         bool      `json:"is_admin"`
	Badges          []string  `json:"badges"`
	Avatars         []string  `json:"avatars"`
	ActiveAvatar    string    `json:"active_avatar"`
	CompletedTopics []string  `json:"completed_topics"`
	PromoUsed       []string  `json:"promo_used"`
	FavoriteGames   []string  `json:"favorite_games"`
	DailyTasksDate  string    `json:"daily_tasks_date"`
	DailyTasksDone  int       `json:"daily_tasks_done"`
	GamesWonToday   int       `json:"games_won_today"`
	GamesWonTypes   []string  `json:"games_won_types"`
	LastDailyClaim  string    `json:"last_daily_claim"`
	CreatedAt       time.Time `json:"created_at"`
}

type LeaderboardEntry struct {
	Rank     int    `json:"rank"`
	Username string `json:"username"`
	Nickname string `json:"nickname"`
	XP       int    `json:"xp"`
	Balance  int    `json:"balance"`
	Badges   int    `json:"badges_count"`
	Streak   int    `json:"streak"`
}

type PromoCode struct {
	Code       string `json:"code"`
	Reward     string `json:"reward"` // "coins", "badge", "avatar", "xp"
	Value      int    `json:"value"`
	BadgeName  string `json:"badge_name"`
	AvatarName string `json:"avatar_name"`
	Uses       int    `json:"uses"` // -1 = unlimited
	UsedCount  int    `json:"used_count"`
}

type GameResult struct {
	UserID      int64  `json:"user_id"`
	GameType    string `json:"game_type"`
	Score       int    `json:"score"`
	XPEarned    int    `json:"xp_earned"`
	CoinsEarned int    `json:"coins_earned"`
	PlayedAt    string `json:"played_at"`
}

type TestResult struct {
	UserID      int64  `json:"user_id"`
	Score       int    `json:"score"`
	Passed      bool   `json:"passed"`
	Level       string `json:"level"`
	BadgeEarned string `json:"badge_earned"`
	PlayedAt    string `json:"played_at"`
}

type CaseResult struct {
	UserID       int64  `json:"user_id"`
	CaseType     string `json:"case_type"`
	ItemEmoji    string `json:"item_emoji"`
	ItemRarity   string `json:"item_rarity"`
	IsDuplicate  bool   `json:"is_duplicate"`
	Compensation int    `json:"compensation"`
	PlayedAt     string `json:"played_at"`
}

// ── КСПОЯ ─────────────────────────────────────────────────────────────────────

// KspoyaSession — попытка прохождения теста. Список вопросов хранится на
// сервере: клиент никогда не получает правильные ответы до завершения теста.
type KspoyaSession struct {
	ID          string    `json:"id"`
	Username    string    `json:"username"`
	QuestionIDs []int     `json:"-"`
	Answers     []int     `json:"-"` // сохраняются, чтобы можно было открыть разбор позже
	StartedAt   time.Time `json:"started_at"`
	ExpiresAt   time.Time `json:"expires_at"`
	Status      string    `json:"status"` // active | completed | aborted
	RawScore    int       `json:"raw_score"`
	Percent     int       `json:"percent"`
	LevelKey    string    `json:"level_key"`
}

// KspoyaAttempt — строка в списке прошлых попыток.
type KspoyaAttempt struct {
	ID         string    `json:"id"`
	FinishedAt time.Time `json:"finished_at"`
	Level      string    `json:"level"`
	LevelLabel string    `json:"level_label"`
	LevelBadge string    `json:"level_badge"`
	Correct    int       `json:"correct"`
	Total      int       `json:"total"`
	Percent    int       `json:"percent"`
	// HasReview = false у попыток, сделанных до появления колонки answers:
	// разбор для них восстановить нечем, кнопку «Анализ» показывать не нужно.
	HasReview bool `json:"has_review"`
}

// KspoyaQuestionClient — вопрос в том виде, в каком он уходит в браузер:
// без правильного ответа, без разбора и без метки сложности.
type KspoyaQuestionClient struct {
	ID      int      `json:"id"`
	Text    string   `json:"question"`
	Options []string `json:"options"`
	Topic   string   `json:"topic"`
}

// KspoyaReviewItem — разбор одного вопроса, отдаётся ТОЛЬКО после завершения.
type KspoyaReviewItem struct {
	ID         int      `json:"id"`
	Text       string   `json:"question"`
	Options    []string `json:"options"`
	Topic      string   `json:"topic"`
	Level      string   `json:"level"`
	UserAnswer int      `json:"user_answer"` // -1 = не ответил
	Correct    int      `json:"correct"`
	IsCorrect  bool     `json:"is_correct"`
	Explain    string   `json:"explain"`
}

// KspoyaStartRequest пустой: набор вопросов выбирает сервер.

// KspoyaSubmitRequest — ответы на выданные вопросы в порядке выдачи.
type KspoyaSubmitRequest struct {
	SessionID string `json:"session_id"`
	Answers   []int  `json:"answers"` // -1 = вопрос пропущен
}

// API request/response types
type RegisterRequest struct {
	Username string `json:"username"`
	Nickname string `json:"nickname"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type PromoRequest struct {
	Code string `json:"code"`
}

type PromoResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Reward  string `json:"reward"`
	Value   int    `json:"value"`
}

type GameSubmitRequest struct {
	GameType string `json:"game_type"`
	Score    int    `json:"score"`
}

type CaseOpenRequest struct {
	CaseType   string `json:"case_type"`
	Price      int    `json:"price"`
	ItemEmoji  string `json:"item_emoji"`
	ItemRarity string `json:"item_rarity"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type SuccessResponse struct {
	Message string `json:"message"`
}
