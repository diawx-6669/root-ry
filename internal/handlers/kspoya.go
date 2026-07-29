package handlers

import (
	"net/http"
	"strconv"
	"time"

	"rootry/internal/kspoya"
	"rootry/internal/models"
	"rootry/internal/store"
)

// buildReview собирает разбор попытки: правильные ответы и объяснения.
// Используется и при сдаче теста, и при открытии прошлой попытки.
func buildReview(sessionID string, questionIDs, answers []int) []models.KspoyaReviewItem {
	review := make([]models.KspoyaReviewItem, 0, len(questionIDs))
	for i, id := range questionIDs {
		question, ok := kspoya.ByID[id]
		if !ok {
			continue
		}
		answer := -1
		if i < len(answers) {
			answer = answers[i]
		}
		options, correctIdx := kspoya.ShuffleOptions(sessionID, question)
		review = append(review, models.KspoyaReviewItem{
			ID:         question.ID,
			Text:       question.Text,
			Options:    options,
			Topic:      question.Topic,
			Level:      question.Level,
			UserAnswer: answer,
			Correct:    correctIdx,
			IsCorrect:  answer == correctIdx,
			Explain:    question.Explain,
		})
	}
	return review
}

// Античит: сколько нарушений допускается и насколько блокируется тест.
// Первое нарушение — предупреждение, второе — блокировка.
const (
	kspoyaMaxStrikes = 2
	kspoyaBanFor     = 24 * time.Hour
)

// POST /api/kspoya/start
//
// Создаёт попытку и отдаёт 40 вопросов БЕЗ правильных ответов и без пометок
// сложности. Если активная попытка уже есть, возвращает её же — так тест
// переживает перезагрузку страницы, но не даёт «перебирать» наборы вопросов.
func (h *Handler) KspoyaStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	username := getUsernameFromCtx(r)
	if _, ok := h.store.GetUserByUsername(username); !ok {
		writeError(w, http.StatusNotFound, "User not found")
		return
	}

	// Блокировка проверяется на сервере: обойти её из браузера нельзя.
	if ban := h.store.KspoyaBanState(username); ban.BannedUntil != nil {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"error":        "Тест заблокирован за нарушение правил",
			"banned_until": ban.BannedUntil,
			"seconds_left": int(time.Until(*ban.BannedUntil).Seconds()),
		})
		return
	}

	session, err := h.store.StartKspoyaSession(
		username, kspoya.SelectQuestions(), kspoya.TestMinutes*time.Minute)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Не удалось начать тест")
		return
	}

	questions := make([]models.KspoyaQuestionClient, 0, len(session.QuestionIDs))
	for _, id := range session.QuestionIDs {
		question, ok := kspoya.ByID[id]
		if !ok {
			continue
		}
		options, _ := kspoya.ShuffleOptions(session.ID, question)
		questions = append(questions, models.KspoyaQuestionClient{
			ID:      question.ID,
			Text:    question.Text,
			Options: options,
			Topic:   question.Topic,
		})
	}

	secondsLeft := int(time.Until(session.ExpiresAt).Seconds())
	if secondsLeft < 0 {
		secondsLeft = 0
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"session_id":   session.ID,
		"questions":    questions,
		"total":        len(questions),
		"seconds_left": secondsLeft,
		"expires_at":   session.ExpiresAt,
	})
}

// POST /api/kspoya/submit
//
// Проверяет ответы на сервере, определяет уровень, начисляет награду и только
// теперь возвращает полный разбор: правильные ответы и объяснения.
func (h *Handler) KspoyaSubmit(w http.ResponseWriter, r *http.Request) {
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

	var req models.KspoyaSubmitRequest
	if err := h.parseBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	session, found := h.store.GetKspoyaSession(req.SessionID, username)
	if !found {
		writeError(w, http.StatusNotFound, "Сессия теста не найдена")
		return
	}
	if session.Status != "active" {
		writeError(w, http.StatusConflict, "Эта попытка уже завершена")
		return
	}

	outcome := kspoya.Grade(session.ID, session.QuestionIDs, req.Answers)

	// Закрываем сессию. Если она уже была закрыта параллельным запросом или
	// просрочена, награда не начисляется, но разбор ученик всё равно увидит.
	awarded := h.store.FinishKspoyaSession(
		session.ID, username, outcome.Correct, outcome.Percent, outcome.Level, req.Answers)

	reward := kspoya.Rewards[outcome.Level]
	xpEarned, coinsEarned, badgeEarned := 0, 0, ""

	if awarded {
		// Награда выдаётся только за уровень, который ещё не был подтверждён.
		// Иначе тест можно было бы перепроходить ради монет.
		if reward.Badge != "" && !store.HasBadge(user.Badges, reward.Badge) {
			user.Badges = append(user.Badges, reward.Badge)
			badgeEarned = reward.Badge
			xpEarned = reward.XP
			coinsEarned = reward.Coins
			user.XP += xpEarned
			user.Balance += coinsEarned
			h.store.UpdateUser(user)
		}
		h.store.SaveTestResult(models.TestResult{
			UserID: user.ID, Score: outcome.Correct,
			Passed: kspoya.LevelIndex(outcome.Level) >= kspoya.LevelIndex("A2"),
			Level:  outcome.Level, BadgeEarned: badgeEarned,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"correct":       outcome.Correct,
		"total":         outcome.Total,
		"answered":      outcome.Answered,
		"best_streak":   outcome.BestStreak,
		"percent":       outcome.Percent,
		"level":         outcome.Level,
		"level_label":   reward.Label,
		"level_badge":   reward.Badge,
		"level_scale":   kspoya.LevelThresholds,
		"by_level":      outcome.ByLevel,
		"by_topic":      outcome.ByTopic,
		"xp_earned":     xpEarned,
		"coins_earned":  coinsEarned,
		"badge_earned":  badgeEarned,
		"reward_repeat": awarded && badgeEarned == "",
		"new_xp":        user.XP,
		"new_balance":   user.Balance,
		"review":        buildReview(session.ID, session.QuestionIDs, req.Answers),
	})
}

// GET /api/kspoya/history — список прошлых попыток для стартового экрана.
func (h *Handler) KspoyaHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	username := getUsernameFromCtx(r)
	attempts := h.store.ListKspoyaAttempts(username, 20)

	best := ""
	for i := range attempts {
		reward := kspoya.Rewards[attempts[i].Level]
		attempts[i].LevelLabel = reward.Label
		attempts[i].LevelBadge = reward.Badge
		if best == "" || kspoya.LevelIndex(attempts[i].Level) > kspoya.LevelIndex(best) {
			best = attempts[i].Level
		}
	}
	if attempts == nil {
		attempts = []models.KspoyaAttempt{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"attempts":   attempts,
		"best_level": best,
	})
}

// GET /api/kspoya/attempt?id=... — детальный разбор завершённой попытки.
func (h *Handler) KspoyaAttempt(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	username := getUsernameFromCtx(r)

	id := r.URL.Query().Get("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "Не указана попытка")
		return
	}

	session, ok := h.store.GetKspoyaSession(id, username)
	if !ok {
		writeError(w, http.StatusNotFound, "Попытка не найдена")
		return
	}
	if session.Status != "completed" {
		writeError(w, http.StatusConflict, "Эта попытка не была завершена")
		return
	}

	// Пересчитываем разбивку из сохранённых ответов.
	outcome := kspoya.Grade(session.ID, session.QuestionIDs, session.Answers)
	reward := kspoya.Rewards[session.LevelKey]

	writeJSON(w, http.StatusOK, map[string]any{
		"id":          session.ID,
		"started_at":  session.StartedAt,
		"correct":     session.RawScore,
		"total":       len(session.QuestionIDs),
		"answered":    outcome.Answered,
		"best_streak": outcome.BestStreak,
		"percent":     session.Percent,
		"level":       session.LevelKey,
		"level_label": reward.Label,
		"level_badge": reward.Badge,
		"level_scale": kspoya.LevelThresholds,
		"by_level":    outcome.ByLevel,
		"by_topic":    outcome.ByTopic,
		"review":      buildReview(session.ID, session.QuestionIDs, session.Answers),
	})
}

// POST /api/kspoya/abort — отмена попытки.
//
// Если передан violation=true (ушёл со вкладки, вышел из полноэкранного
// режима), нарушение засчитывается: первое даёт предупреждение, второе
// блокирует тест на сутки.
func (h *Handler) KspoyaAbort(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	username := getUsernameFromCtx(r)
	var req struct {
		SessionID string `json:"session_id"`
		Violation bool   `json:"violation"`
		Reason    string `json:"reason"`
	}
	h.parseBody(r, &req) // тело необязательно: без id отменяем все активные
	h.store.AbortKspoyaSession(req.SessionID, username)

	state := models.KspoyaBanState{MaxStrikes: kspoyaMaxStrikes}
	if req.Violation {
		state = h.store.RegisterKspoyaViolation(username, kspoyaMaxStrikes, kspoyaBanFor)
		state.MaxStrikes = kspoyaMaxStrikes
	}

	resp := map[string]any{
		"warnings":    state.Warnings,
		"max_strikes": kspoyaMaxStrikes,
		"banned":      state.BannedUntil != nil,
	}
	if state.BannedUntil != nil {
		resp["banned_until"] = state.BannedUntil
		resp["seconds_left"] = int(time.Until(*state.BannedUntil).Seconds())
	}
	writeJSON(w, http.StatusOK, resp)
}

// GET /api/kspoya/leaderboard — рейтинг по лучшей попытке каждого ученика.
func (h *Handler) KspoyaLeaderboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	limit := 6
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}

	entries := h.store.KspoyaLeaderboard(limit)
	for i := range entries {
		reward := kspoya.Rewards[entries[i].Level]
		entries[i].LevelLabel = reward.Label
		entries[i].LevelBadge = reward.Badge
	}
	if entries == nil {
		entries = []models.KspoyaLeaderEntry{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"entries": entries})
}

// GET /api/kspoya/status — есть ли незавершённая попытка и сколько времени осталось.
func (h *Handler) KspoyaStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	username := getUsernameFromCtx(r)

	ban := h.store.KspoyaBanState(username)

	resp := map[string]any{
		"has_active":   false,
		"bank_size":    len(kspoya.Bank),
		"per_test":     kspoya.QuestionsPerTest,
		"options":      kspoya.OptionsPerQuestion,
		"level_scale":  kspoya.LevelThresholds,
		"time_minutes": kspoya.TestMinutes,
		"warnings":     ban.Warnings,
		"max_strikes":  kspoyaMaxStrikes,
		"banned":       ban.BannedUntil != nil,
	}
	if ban.BannedUntil != nil {
		resp["banned_until"] = ban.BannedUntil
		resp["seconds_left_ban"] = int(time.Until(*ban.BannedUntil).Seconds())
	}

	// Лучший результат — для кнопки «Лучший результат» под стартом.
	if id, score, level, ok := h.store.BestKspoyaAttemptID(username); ok {
		reward := kspoya.Rewards[level]
		resp["best"] = map[string]any{
			"attempt_id":  id,
			"score":       score,
			"total":       kspoya.QuestionsPerTest,
			"level":       level,
			"level_label": reward.Label,
			"level_badge": reward.Badge,
		}
	}
	if session, ok := h.store.ActiveKspoyaSession(username); ok {
		secondsLeft := int(time.Until(session.ExpiresAt).Seconds())
		if secondsLeft < 0 {
			secondsLeft = 0
		}
		resp["has_active"] = true
		resp["session_id"] = session.ID
		resp["seconds_left"] = secondsLeft
	}
	writeJSON(w, http.StatusOK, resp)
}
