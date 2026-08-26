package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"rootry/internal/kspoya"
	"rootry/internal/middleware"
	"rootry/internal/models"
	"rootry/internal/store"
	"rootry/internal/timeutil"
	"rootry/internal/topics"
)

// Ограничения на текстовые поля. Считаются в символах, а не в байтах:
// len() для кириллицы даёт вдвое больше, и ник из 17 русских букв раньше
// не проходил проверку «до 32 символов».
const (
	usernameMin = 3
	usernameMax = 20
	nicknameMin = 1
	nicknameMax = 32
	passwordMin = 6
)

// maxBodyBytes — предел размера тела запроса. Без него отправка гигабайтного
// JSON заставляла сервер съесть всю память.
const maxBodyBytes = 64 << 10

type Handler struct {
	store *store.Store
}

func New(s *store.Store) *Handler {
	return &Handler{store: s}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, models.ErrorResponse{Error: msg})
}

func (h *Handler) parseBody(r *http.Request, v any) error {
	return json.NewDecoder(http.MaxBytesReader(nil, r.Body, maxBodyBytes)).Decode(v)
}

// requireMethod отвечает 405 и возвращает false, если метод не тот.
// Раньше три обработчика проверку метода просто не делали.
func requireMethod(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method != method {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return false
	}
	return true
}

func getUsernameFromCtx(r *http.Request) string {
	if v := r.Context().Value(middleware.UsernameKey); v != nil {
		return v.(string)
	}
	return ""
}

// POST /api/register
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	var req models.RegisterRequest
	if err := h.parseBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	req.Nickname = strings.TrimSpace(req.Nickname)

	// Никнейм НЕ экранируется при записи: в базе он должен лежать таким,
	// каким его ввёл ученик. Раньше «Вася & Петя» сохранялся как
	// «Вася &amp; Петя» и в таком виде и показывался. Экранирование —
	// задача вывода, и на фронте для этого есть esc().
	if err := validateUsername(req.Username); err != "" {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := validateNickname(req.Nickname); err != "" {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if utf8.RuneCountInString(req.Password) < passwordMin {
		writeError(w, http.StatusBadRequest, "Пароль: минимум 6 символов")
		return
	}

	user, err := h.store.CreateUser(req.Username, req.Nickname, req.Password)
	if err != nil {
		// Занятый логин — ошибка ученика, всё остальное — авария сервера.
		if errors.Is(err, store.ErrUsernameTaken) {
			writeError(w, http.StatusConflict, "Логин уже занят")
			return
		}
		writeError(w, http.StatusInternalServerError, "Не удалось создать аккаунт, попробуйте позже")
		return
	}

	token, _ := middleware.GenerateToken(user.ID, user.Username, user.IsAdmin)
	writeJSON(w, http.StatusCreated, models.LoginResponse{Token: token, User: *user})
}

// POST /api/login
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	var req models.LoginRequest
	if err := h.parseBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	user, ok := h.store.ValidatePassword(req.Username, req.Password)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Неверный логин или пароль")
		return
	}

	// Серия входов продлевается одним запросом к базе. Монеты за день
	// выдаёт только /api/daily/claim.
	if streak, updated := h.store.TouchDailyLogin(user.Username); updated {
		user.Streak = streak
	}

	token, _ := middleware.GenerateToken(user.ID, user.Username, user.IsAdmin)
	writeJSON(w, http.StatusOK, models.LoginResponse{Token: token, User: *user})
}

// GET /api/me
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	username := getUsernameFromCtx(r)

	// Заход на любую страницу продлевает серию. Раньше она обновлялась
	// только при вводе логина и пароля, а с 30-дневным токеном этого
	// почти никогда не происходит.
	h.store.TouchDailyLogin(username)

	user, ok := h.store.GetUserByUsername(username)
	if !ok {
		writeError(w, http.StatusNotFound, "User not found")
		return
	}
	writeJSON(w, http.StatusOK, user)
}

// PUT /api/profile/avatar — выбор активной аватарки.
//
// Раньше выбор жил только в localStorage браузера и слетал при первом же
// обновлении данных с сервера.
func (h *Handler) UpdateAvatar(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	username := getUsernameFromCtx(r)
	user, ok := h.store.GetUserByUsername(username)
	if !ok {
		writeError(w, http.StatusNotFound, "User not found")
		return
	}

	var req struct {
		Avatar string `json:"avatar"`
	}
	if err := h.parseBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	// Ставить можно только ту аватарку, которая уже есть в инвентаре.
	if !store.HasAvatar(user.Avatars, req.Avatar) {
		writeError(w, http.StatusBadRequest, "Эта аватарка вам не принадлежит")
		return
	}

	user.ActiveAvatar = req.Avatar
	h.store.UpdateUser(user)
	writeJSON(w, http.StatusOK, user)
}

// GET /api/leaderboard
func (h *Handler) Leaderboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	entries := h.store.GetLeaderboard()
	// Значок уровня КСПОЯ считается по уровню, а не хранится в базе.
	// Раньше поле оставалось пустым: рейтинг КСПОЯ его так и не показывал.
	for i := range entries {
		if entries[i].KspoyaLevel == "" {
			continue
		}
		entries[i].KspoyaBadge = kspoya.Rewards[entries[i].KspoyaLevel].Badge
	}
	if entries == nil {
		entries = []models.LeaderboardEntry{}
	}
	writeJSON(w, http.StatusOK, entries)
}

// POST /api/promo
func (h *Handler) RedeemPromo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	username := getUsernameFromCtx(r)
	user, ok := h.store.GetUserByUsername(username)
	if !ok {
		writeError(w, http.StatusNotFound, "User not found")
		return
	}

	var req models.PromoRequest
	if err := h.parseBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}
	code := strings.ToUpper(strings.TrimSpace(req.Code))

	for _, used := range user.PromoUsed {
		if used == code {
			writeJSON(w, http.StatusOK, models.PromoResponse{Success: false, Message: "Промокод уже активирован"})
			return
		}
	}

	promo, found := h.store.GetPromo(code)
	if !found {
		writeJSON(w, http.StatusOK, models.PromoResponse{Success: false, Message: "Промокод не найден"})
		return
	}
	if promo.Uses != -1 && promo.UsedCount >= promo.Uses {
		writeJSON(w, http.StatusOK, models.PromoResponse{Success: false, Message: "Промокод исчерпан"})
		return
	}

	resp := models.PromoResponse{Success: true, Value: promo.Value}
	switch promo.Reward {
	case "coins":
		user.Balance += promo.Value
		resp.Message = "Получено монет: " + store.Itoa(promo.Value)
		resp.Reward = "coins"
	case "xp":
		user.XP += promo.Value
		resp.Message = "Получено XP: " + store.Itoa(promo.Value)
		resp.Reward = "xp"
	case "badge":
		if promo.BadgeName != "" && !store.HasBadge(user.Badges, promo.BadgeName) {
			user.Badges = append(user.Badges, promo.BadgeName)
		}
		resp.Message = "Получен значок: " + promo.BadgeName
		resp.Reward = "badge"
	case "avatar":
		if promo.AvatarName != "" && !store.HasAvatar(user.Avatars, promo.AvatarName) {
			user.Avatars = append(user.Avatars, promo.AvatarName)
		}
		resp.Message = "Получена аватарка: " + promo.AvatarName
		resp.Reward = "avatar"
	case "admin":
		user.IsAdmin = true
		resp.Message = "Права администратора выданы!"
		resp.Reward = "admin"
	}

	user.PromoUsed = append(user.PromoUsed, code)
	h.store.UpdateUser(user)
	h.store.UsePromo(code)
	writeJSON(w, http.StatusOK, resp)
}

// POST /api/game/submit
func (h *Handler) GameSubmit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	username := getUsernameFromCtx(r)
	user, ok := h.store.GetUserByUsername(username)
	if !ok {
		writeError(w, http.StatusNotFound, "User not found")
		return
	}

	var req models.GameSubmitRequest
	if err := h.parseBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}
	if req.Score < 0 || req.Score > 10000 {
		writeError(w, http.StatusBadRequest, "Неверный счёт")
		return
	}

	today := timeutil.Today()

	// Reset daily counters if new day
	if user.DailyTasksDate != today {
		user.DailyTasksDate = today
		user.DailyTasksDone = 0
		user.GamesWonToday = 0
		user.GamesWonTypes = []string{}
	}

	xpEarned := 0
	coinsEarned := 0
	firstWin := false
	questBonusEarned := false
	isWin := req.Score > 0

	// Coins and XP only for first win of each game type per day
	if isWin {
		alreadyWon := false
		for _, gt := range user.GamesWonTypes {
			if gt == req.GameType {
				alreadyWon = true
				break
			}
		}
		if !alreadyWon {
			xpEarned = 10
			coinsEarned = 50
			firstWin = true
			user.GamesWonTypes = append(user.GamesWonTypes, req.GameType)
			user.GamesWonToday++

			// Quest bonus: after 5 unique game type wins today
			if user.GamesWonToday >= 5 && user.DailyTasksDone == 0 {
				coinsEarned += 50
				questBonusEarned = true
				user.DailyTasksDone = 1
			}

			user.XP += xpEarned
			user.Balance += coinsEarned
		}
	}

	badgeEarned := ""
	if isWin && req.Score >= 90 && !store.HasBadge(user.Badges, "🏆") {
		user.Badges = append(user.Badges, "🏆")
		badgeEarned = "🏆"
	}

	h.store.UpdateUser(user)
	h.store.SaveGameResult(models.GameResult{
		UserID: user.ID, GameType: req.GameType, Score: req.Score,
		XPEarned: xpEarned, CoinsEarned: coinsEarned,
		PlayedAt: time.Now().Format(time.RFC3339),
	})

	writeJSON(w, http.StatusOK, map[string]any{
		"xp_earned":          xpEarned,
		"coins_earned":       coinsEarned,
		"new_balance":        user.Balance,
		"new_xp":             user.XP,
		"badge_earned":       badgeEarned,
		"quest_bonus_earned": questBonusEarned,
		"first_win":          firstWin,
		"games_won_today":    user.GamesWonToday,
	})
}

// POST /api/topic/complete — засчитать пройденный урок дерева грамматики.
//
// Тема проверяется по реестру internal/topics, а размер награды берётся
// оттуда же. Раньше поле topic не проверялось вообще: любая новая случайная
// строка приносила 50 XP и 10 монет, и опыт фармился бесконечно.
func (h *Handler) TopicComplete(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	username := getUsernameFromCtx(r)
	user, ok := h.store.GetUserByUsername(username)
	if !ok {
		writeError(w, http.StatusNotFound, "User not found")
		return
	}

	var req struct {
		Topic   string `json:"topic"`
		Correct int    `json:"correct"`
		Total   int    `json:"total"`
	}
	if err := h.parseBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	topic, known := topics.Get(strings.TrimSpace(req.Topic))
	if !known {
		writeError(w, http.StatusBadRequest, "Неизвестная тема")
		return
	}

	// Награду назначает интервал повторения, а не факт открытия урока.
	// Первое прохождение оплачивается полностью, повторение просроченной
	// темы — частично, повторение «свежей» темы не оплачивается вовсе.
	now := timeutil.Now()
	kind, nextReward, err := h.store.ClaimTopicReward(user.ID, topic.ID, now)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Не удалось засчитать тему")
		return
	}

	var xpEarned, coinsEarned int
	switch kind {
	case store.RewardFirst:
		xpEarned, coinsEarned = topic.XP, topic.Coins
	case store.RewardReview:
		xpEarned, coinsEarned = topics.ReviewReward(topic)
	}

	// completed_topics остаётся источником правды для «сколько тем пройдено»:
	// на него смотрят профиль, дерево и значок за всё дерево.
	firstTime := !contains(user.CompletedTopics, topic.ID)
	if firstTime {
		user.CompletedTopics = append(user.CompletedTopics, topic.ID)
	}
	user.XP += xpEarned
	user.Balance += coinsEarned

	// Значок за прохождение всего дерева.
	badgeEarned := ""
	if len(user.CompletedTopics) >= topics.Count() && !store.HasBadge(user.Badges, treeMasterBadge) {
		user.Badges = append(user.Badges, treeMasterBadge)
		badgeEarned = treeMasterBadge
	}

	if xpEarned > 0 || coinsEarned > 0 || firstTime || badgeEarned != "" {
		h.store.UpdateUser(user)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		// already_done оставлен для старых сборок фронта, которые на него
		// смотрят. Новый признак — reward.
		"already_done":    kind != store.RewardFirst,
		"reward":          kind,
		"xp_earned":       xpEarned,
		"coins_earned":    coinsEarned,
		"new_xp":          user.XP,
		"new_balance":     user.Balance,
		"badge_earned":    badgeEarned,
		"next_reward_on":  nextReward.Format("2006-01-02"),
		"completed_count": len(user.CompletedTopics),
		"total_topics":    topics.Count(),
	})
}

// treeMasterBadge — значок за пройденное дерево грамматики целиком.
// Не пересекается ни с пулом магазина, ни со значками КСПОЯ.
const treeMasterBadge = "🌳"

// POST /api/case/open — server-side roll
func (h *Handler) CaseOpen(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	username := getUsernameFromCtx(r)
	user, ok := h.store.GetUserByUsername(username)
	if !ok {
		writeError(w, http.StatusNotFound, "User not found")
		return
	}

	var req models.CaseOpenRequest
	if err := h.parseBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	prices := map[string]int{"common": 600, "rare": 1500, "epic": 3000, "legendary": 10000, "badge": 300}
	expectedPrice, ok2 := prices[req.CaseType]
	if !ok2 {
		writeError(w, http.StatusBadRequest, "Неверный тип кейса")
		return
	}
	if user.Balance < expectedPrice {
		writeError(w, http.StatusBadRequest, "Недостаточно монет")
		return
	}

	user.Balance -= expectedPrice

	// Server-side roll
	isBadgeCase := req.CaseType == "badge"
	itemEmoji, itemRarity, isDuplicate := store.RollCase(req.CaseType, user.Avatars, user.Badges, isBadgeCase)

	compensation := 0
	if isDuplicate {
		if isBadgeCase {
			compensation = 100
		} else {
			compensation = store.CompensationForRarity(itemRarity)
		}
		user.Balance += compensation
	} else {
		if isBadgeCase {
			user.Badges = append(user.Badges, itemEmoji)
		} else {
			user.Avatars = append(user.Avatars, itemEmoji)
		}
	}

	// XP for opening case (+10 base per PDF, + rarity bonus)
	xpGained := 10 + store.XPForRarity(itemRarity)
	user.XP += xpGained

	h.store.UpdateUser(user)
	h.store.SaveCaseResult(models.CaseResult{
		UserID: user.ID, CaseType: req.CaseType,
		ItemEmoji: itemEmoji, ItemRarity: itemRarity,
		IsDuplicate: isDuplicate, Compensation: compensation,
		PlayedAt: time.Now().Format(time.RFC3339),
	})

	writeJSON(w, http.StatusOK, map[string]any{
		"item_emoji":   itemEmoji,
		"item_rarity":  itemRarity,
		"is_duplicate": isDuplicate,
		"compensation": compensation,
		"xp_gained":    xpGained,
		"new_balance":  user.Balance,
		"new_xp":       user.XP,
	})
}

// POST /api/daily/claim — единственный источник ежедневного бонуса.
func (h *Handler) DailyClaim(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	username := getUsernameFromCtx(r)
	user, ok := h.store.GetUserByUsername(username)
	if !ok {
		writeError(w, http.StatusNotFound, "User not found")
		return
	}
	today := timeutil.Today()
	if user.LastDailyClaim == today {
		writeJSON(w, http.StatusOK, map[string]any{"already_claimed": true})
		return
	}
	// Streak-based reward matching frontend formula: 10 + floor(streak/7)*5
	streakDays := user.Streak
	if streakDays < 0 {
		streakDays = 0
	}
	reward := 10 + (streakDays/7)*5
	user.LastDailyClaim = today
	user.Balance += reward
	h.store.UpdateUser(user)
	writeJSON(w, http.StatusOK, map[string]any{
		"coins_earned": reward, "new_balance": user.Balance, "streak": user.Streak,
	})
}

// PUT /api/profile/nickname
func (h *Handler) UpdateNickname(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	username := getUsernameFromCtx(r)
	user, ok := h.store.GetUserByUsername(username)
	if !ok {
		writeError(w, http.StatusNotFound, "User not found")
		return
	}

	// Per PDF: nickname can be changed no more than once per week
	if user.LastNickChange != "" {
		last, err := time.ParseInLocation("2006-01-02", user.LastNickChange, timeutil.Location())
		if err == nil && timeutil.Now().Sub(last) < 7*24*time.Hour {
			writeError(w, http.StatusBadRequest, "Никнейм можно менять не чаще раза в неделю")
			return
		}
	}

	var req struct {
		Nickname string `json:"nickname"`
	}
	if err := h.parseBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}
	nick := strings.TrimSpace(req.Nickname)
	if msg := validateNickname(nick); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}

	user.Nickname = nick
	user.LastNickChange = timeutil.Today()
	h.store.UpdateUser(user)
	writeJSON(w, http.StatusOK, user)
}

// GET /api/admin/users
func (h *Handler) AdminUsers(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	users := h.store.GetAllUsers()
	if users == nil {
		users = []*models.User{}
	}
	writeJSON(w, http.StatusOK, users)
}

// GET /api/admin/stats
func (h *Handler) AdminStats(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	users := h.store.GetAllUsers()
	results := h.store.GetGameResults()
	caseResults := h.store.GetCaseResults()
	totalXP, totalBalance, totalUsers := 0, 0, 0
	totalBadges, totalAvatars := 0, 0
	activeToday := 0
	today := timeutil.Today()
	for _, u := range users {
		if !u.IsAdmin {
			totalXP += u.XP
			totalBalance += u.Balance
			totalUsers++
			totalBadges += len(u.Badges)
			totalAvatars += len(u.Avatars)
			if u.LastLogin == today || u.DailyTasksDate == today {
				activeToday++
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"total_users":   totalUsers,
		"total_games":   len(results),
		"total_cases":   len(caseResults),
		"total_xp":      totalXP,
		"total_balance": totalBalance,
		"total_badges":  totalBadges,
		"total_avatars": totalAvatars,
		"active_today":  activeToday,
	})
}

// GET /api/topics — реестр тем дерева и прогресс ученика.
//
// Дерево на клиенте рисуется из статического файла, а вот сколько тем
// существует и какие пройдены — источник правды на сервере.
func (h *Handler) Topics(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	username := getUsernameFromCtx(r)
	user, ok := h.store.GetUserByUsername(username)
	if !ok {
		writeError(w, http.StatusNotFound, "User not found")
		return
	}
	completed := user.CompletedTopics
	if completed == nil {
		completed = []string{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"topics":    topics.All,
		"completed": completed,
		"total":     topics.Count(),
	})
}

// PUT /api/profile/favorites — избранные игры.
//
// Колонка favorite_games была в базе и в модели, но фронт хранил избранное
// в localStorage: список терялся при смене устройства и при очистке браузера.
func (h *Handler) UpdateFavorites(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPut) {
		return
	}
	username := getUsernameFromCtx(r)
	user, ok := h.store.GetUserByUsername(username)
	if !ok {
		writeError(w, http.StatusNotFound, "User not found")
		return
	}

	var req struct {
		Games []string `json:"games"`
	}
	if err := h.parseBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	// Принимаем только известные игры и не больше, чем их всего есть:
	// иначе в колонку можно было бы положить произвольный список любой длины.
	favorites := make([]string, 0, len(knownGames))
	seen := map[string]bool{}
	for _, g := range req.Games {
		if !knownGames[g] || seen[g] {
			continue
		}
		seen[g] = true
		favorites = append(favorites, g)
	}

	user.FavoriteGames = favorites
	h.store.UpdateUser(user)
	writeJSON(w, http.StatusOK, map[string]any{"favorite_games": favorites})
}

// knownGames — идентификаторы мини-игр, совпадают с game_type в /api/game/submit.
var knownGames = map[string]bool{
	"comma_ninja":    true,
	"stress_space":   true,
	"word_alchemist": true,
	"minefield":      true,
	"detective_case": true,
}

// validateUsername проверяет логин и возвращает текст ошибки («» — всё в порядке).
func validateUsername(s string) string {
	n := utf8.RuneCountInString(s)
	if n < usernameMin || !isLatinOnly(s) {
		return "Логин: только латиница, цифры и _, от 3 символов"
	}
	if n > usernameMax {
		// Верхней границы не было вовсе — логин мог быть любой длины.
		return "Логин: не длиннее 20 символов"
	}
	return ""
}

// validateNickname проверяет никнейм в символах, а не в байтах.
func validateNickname(s string) string {
	n := utf8.RuneCountInString(s)
	if n < nicknameMin || n > nicknameMax {
		return "Никнейм: от 1 до 32 символов"
	}
	return ""
}

func isLatinOnly(s string) bool {
	for _, c := range s {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
			return false
		}
	}
	return true
}
