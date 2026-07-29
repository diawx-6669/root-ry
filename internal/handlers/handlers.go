package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"rootry/internal/middleware"
	"rootry/internal/models"
	"rootry/internal/store"
)

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
	return json.NewDecoder(r.Body).Decode(v)
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
	// Sanitize nickname to prevent XSS
	req.Nickname = store.EscapeHTML(req.Nickname)

	if len(req.Username) < 3 || !isLatinOnly(req.Username) {
		writeError(w, http.StatusBadRequest, "Логин: только латиница, от 3 символов")
		return
	}
	if len(req.Nickname) < 1 || len(req.Nickname) > 32 {
		writeError(w, http.StatusBadRequest, "Никнейм: от 1 до 32 символов")
		return
	}
	if len(req.Password) < 6 {
		writeError(w, http.StatusBadRequest, "Пароль: минимум 6 символов")
		return
	}

	user, err := h.store.CreateUser(req.Username, req.Nickname, req.Password)
	if err != nil {
		writeError(w, http.StatusConflict, "Логин уже занят")
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

	// Серия и ежедневный бонус считаются в одном запросе к базе.
	if streak, balance, awarded := h.store.TouchDailyLogin(user.Username); awarded {
		user.Streak = streak
		user.Balance = balance
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

	today := time.Now().Format("2006-01-02")

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

// POST /api/topic/complete
func (h *Handler) TopicComplete(w http.ResponseWriter, r *http.Request) {
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

	var req struct {
		Topic string `json:"topic"`
	}
	if err := h.parseBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	for _, t := range user.CompletedTopics {
		if t == req.Topic {
			writeJSON(w, http.StatusOK, map[string]any{"already_done": true, "xp_earned": 0})
			return
		}
	}

	user.CompletedTopics = append(user.CompletedTopics, req.Topic)
	user.XP += 50
	user.Balance += 10
	h.store.UpdateUser(user)

	writeJSON(w, http.StatusOK, map[string]any{
		"xp_earned": 50, "coins_earned": 10,
		"new_xp": user.XP, "completed_count": len(user.CompletedTopics),
	})
}

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

// /api/case/free
//
//	GET  — доступен ли бесплатный кейс сегодня;
//	POST — открыть его.
//
// Ограничение «раз в сутки» проверяется в базе, а не в браузере.
func (h *Handler) FreeCase(w http.ResponseWriter, r *http.Request) {
	username := getUsernameFromCtx(r)
	user, ok := h.store.GetUserByUsername(username)
	if !ok {
		writeError(w, http.StatusNotFound, "User not found")
		return
	}

	if r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, map[string]any{
			"available": h.store.FreeCaseAvailable(username),
		})
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Отметка ставится до розыгрыша: если кейс сегодня уже брали,
	// запрос не пройдёт и предмет не выдастся.
	if !h.store.ClaimFreeCase(username) {
		writeJSON(w, http.StatusOK, map[string]any{
			"available": false,
			"message":   "Бесплатный кейс сегодня уже открыт. Возвращайтесь завтра!",
		})
		return
	}

	itemEmoji, itemRarity, isDuplicate := store.RollCase("free", user.Avatars, user.Badges, false)

	compensation := 0
	if isDuplicate {
		compensation = store.CompensationForRarity(itemRarity)
		user.Balance += compensation
	} else {
		user.Avatars = append(user.Avatars, itemEmoji)
	}

	xpGained := 10 + store.XPForRarity(itemRarity)
	user.XP += xpGained

	h.store.UpdateUser(user)
	h.store.SaveCaseResult(models.CaseResult{
		UserID: user.ID, CaseType: "free",
		ItemEmoji: itemEmoji, ItemRarity: itemRarity,
		IsDuplicate: isDuplicate, Compensation: compensation,
		PlayedAt: time.Now().Format(time.RFC3339),
	})

	writeJSON(w, http.StatusOK, map[string]any{
		"available":    true,
		"item_emoji":   itemEmoji,
		"item_rarity":  itemRarity,
		"is_duplicate": isDuplicate,
		"compensation": compensation,
		"xp_gained":    xpGained,
		"new_balance":  user.Balance,
		"new_xp":       user.XP,
	})
}

// GET /api/daily/claim
func (h *Handler) DailyClaim(w http.ResponseWriter, r *http.Request) {
	username := getUsernameFromCtx(r)
	user, ok := h.store.GetUserByUsername(username)
	if !ok {
		writeError(w, http.StatusNotFound, "User not found")
		return
	}
	today := time.Now().Format("2006-01-02")
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
		last, err := time.Parse("2006-01-02", user.LastNickChange)
		if err == nil && time.Since(last) < 7*24*time.Hour {
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
	nick := store.EscapeHTML(strings.TrimSpace(req.Nickname))
	if len(nick) < 1 || len(nick) > 32 {
		writeError(w, http.StatusBadRequest, "Никнейм: от 1 до 32 символов")
		return
	}

	user.Nickname = nick
	user.LastNickChange = time.Now().Format("2006-01-02")
	h.store.UpdateUser(user)
	writeJSON(w, http.StatusOK, user)
}

// GET /api/admin/users
func (h *Handler) AdminUsers(w http.ResponseWriter, r *http.Request) {
	users := h.store.GetAllUsers()
	writeJSON(w, http.StatusOK, users)
}

// GET /api/admin/stats
func (h *Handler) AdminStats(w http.ResponseWriter, r *http.Request) {
	users := h.store.GetAllUsers()
	results := h.store.GetGameResults()
	caseResults := h.store.GetCaseResults()
	totalXP, totalBalance, totalUsers := 0, 0, 0
	totalBadges, totalAvatars := 0, 0
	activeToday := 0
	today := time.Now().Format("2006-01-02")
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

func isLatinOnly(s string) bool {
	for _, c := range s {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
			return false
		}
	}
	return true
}
