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
	// Выбранная аватарка: рейтинг рисовал робота по логину и игнорировал её.
	ActiveAvatar string `json:"active_avatar"`
	// Лучший результат КСПОЯ. KspoyaLevel пуст, если тест ещё не сдавали.
	KspoyaScore int    `json:"kspoya_score"`
	KspoyaLevel string `json:"kspoya_level"`
	KspoyaBadge string `json:"kspoya_badge"`
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

// KspoyaLeaderEntry — строка рейтинга КСПОЯ: лучшая попытка пользователя.
type KspoyaLeaderEntry struct {
	Rank         int       `json:"rank"`
	Username     string    `json:"username"`
	Nickname     string    `json:"nickname"`
	Score        int       `json:"score"`
	Total        int       `json:"total"`
	Level        string    `json:"level"`
	ActiveAvatar string    `json:"active_avatar"`
	LevelLabel   string    `json:"level_label"`
	LevelBadge   string    `json:"level_badge"`
	FinishedAt   time.Time `json:"finished_at"`
}

// KspoyaBanState — состояние античита у пользователя.
type KspoyaBanState struct {
	Warnings    int        `json:"warnings"`
	MaxStrikes  int        `json:"max_strikes"`
	BannedUntil *time.Time `json:"banned_until,omitempty"`
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

// ── Модель знаний ─────────────────────────────────────────────────────────────

// MistakeRef — ссылка на незакрытое задание в тетради ошибок.
//
// Текста задания здесь нет намеренно: он лежит в static/lessons/*.js, и
// клиент подставляет его сам. Иначе пришлось бы держать второй источник
// правды и следить, чтобы он не разъехался с первым.
type MistakeRef struct {
	ItemID     string `json:"item_id"`
	TopicID    string `json:"topic_id"`
	WrongCount int    `json:"wrong_count"`
	StreakDays int    `json:"streak_days"`
	LastWrong  string `json:"last_wrong"`
}

// TopicProgress — состояние одной темы для дерева грамматики.
type TopicProgress struct {
	TopicID string `json:"topic_id"`
	// Status: new | learning | mastered | due
	Status      string  `json:"status"`
	Box         int     `json:"box"`
	MaxBox      int     `json:"max_box"`
	Strength    float64 `json:"strength"`
	OverdueDays int     `json:"overdue_days"`
	DueOn       string  `json:"due_on"`
	Correct     int     `json:"correct"`
	Total       int     `json:"total"`
	Lapses      int     `json:"lapses"`
}

// AttemptRequest — один ответ ученика, как его присылает браузер.
//
// Ни правильность, ни награду клиент не сообщает: правильность приходит
// как факт ответа, а всё остальное считает сервер.
type AttemptRequest struct {
	Topic   string `json:"topic"`
	Item    string `json:"item"`
	Source  string `json:"source"`
	Correct bool   `json:"correct"`
	Hints   int    `json:"hints"`
	TimeMs  int    `json:"time_ms"`
}

// AttemptResponse — что ученик увидит после ответа.
type AttemptResponse struct {
	Status        string `json:"status"`
	Box           int    `json:"box"`
	MaxBox        int    `json:"max_box"`
	DueOn         string `json:"due_on"`
	TopicPromoted bool   `json:"topic_promoted"`
	// Assisted — ответ дан с открытым разбором и знание не подтверждает.
	// Клиент показывает это ученику: прогресс по теме нужно заработать сам.
	Assisted      bool `json:"assisted"`
	MistakeClosed bool `json:"mistake_closed"`
	MistakeCount  int  `json:"mistake_count"`
}

// ReviewPlanItem — тема, которую пора повторить сегодня.
type ReviewPlanItem struct {
	TopicID     string `json:"topic_id"`
	Title       string `json:"title"`
	Section     string `json:"section"`
	Box         int    `json:"box"`
	OverdueDays int    `json:"overdue_days"`
	DueOn       string `json:"due_on"`
	// XP, который ученик получит за повторение. Меньше, чем за первое
	// прохождение, но не ноль: возвращаться должно быть выгодно.
	XP    int `json:"xp"`
	Coins int `json:"coins"`
}

// ReviewPlan — план занятий на сегодня.
type ReviewPlan struct {
	Today      string           `json:"today"`
	Due        []ReviewPlanItem `json:"due"`
	Mistakes   int              `json:"mistakes"`
	Learned    int              `json:"learned"`
	Mastered   int              `json:"mastered"`
	TotalTopic int              `json:"total_topics"`
}

// ── Классы и учитель ──────────────────────────────────────────────────────────

// Class — учебная группа.
type Class struct {
	ID        int64     `json:"id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	TeacherID int64     `json:"teacher_id"`
	CreatedAt time.Time `json:"created_at"`
	// Students заполняется в списке учителя, TeacherName — в списке ученика.
	Students    int    `json:"students,omitempty"`
	TeacherName string `json:"teacher_name,omitempty"`
}

// ClassStudent — строка ученика в тепловой карте класса.
type ClassStudent struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Nickname string `json:"nickname"`
	XP       int    `json:"xp"`
	Started  int    `json:"started"`  // тем начато
	Mastered int    `json:"mastered"` // тем освоено
	Due      int    `json:"due"`      // тем просрочено
	Mistakes int    `json:"mistakes"` // заданий в тетради ошибок
}

// ClassHeatmap — состояние тем у всех учеников класса.
//
// Cells устроен как «тема → ученик → состояние», а не плоским списком:
// карта рисуется строками по темам, и такая форма отдаётся клиенту
// готовой к отрисовке.
type ClassHeatmap struct {
	Students []ClassStudent               `json:"students"`
	Cells    map[string]map[string]string `json:"cells"`
}
