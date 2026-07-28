package handlers

import (
	"net/http"
	"time"

	"rootry/internal/kspoya"
	"rootry/internal/models"
	"rootry/internal/store"
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
		session.ID, username, outcome.Correct, outcome.Percent, outcome.Level)

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

	// Разбор: правильные ответы и объяснения уходят клиенту только здесь.
	review := make([]models.KspoyaReviewItem, 0, len(session.QuestionIDs))
	for i, id := range session.QuestionIDs {
		question, exists := kspoya.ByID[id]
		if !exists {
			continue
		}
		answer := -1
		if i < len(req.Answers) {
			answer = req.Answers[i]
		}
		options, correctIdx := kspoya.ShuffleOptions(session.ID, question)
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

	writeJSON(w, http.StatusOK, map[string]any{
		"correct":       outcome.Correct,
		"total":         outcome.Total,
		"answered":      outcome.Answered,
		"percent":       outcome.Percent,
		"level":         outcome.Level,
		"level_label":   reward.Label,
		"level_badge":   reward.Badge,
		"capped":        outcome.Capped,
		"by_level":      outcome.ByLevel,
		"by_topic":      outcome.ByTopic,
		"xp_earned":     xpEarned,
		"coins_earned":  coinsEarned,
		"badge_earned":  badgeEarned,
		"reward_repeat": awarded && badgeEarned == "",
		"new_xp":        user.XP,
		"new_balance":   user.Balance,
		"review":        review,
	})
}

// POST /api/kspoya/abort — отмена попытки (античит или отказ ученика).
func (h *Handler) KspoyaAbort(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	username := getUsernameFromCtx(r)
	var req struct {
		SessionID string `json:"session_id"`
	}
	h.parseBody(r, &req) // тело необязательно: без id отменяем все активные
	h.store.AbortKspoyaSession(req.SessionID, username)
	writeJSON(w, http.StatusOK, models.SuccessResponse{Message: "Попытка отменена"})
}

// GET /api/kspoya/status — есть ли незавершённая попытка и сколько времени осталось.
func (h *Handler) KspoyaStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	username := getUsernameFromCtx(r)
	levels, attempts := h.store.BestKspoyaLevel(username)

	best := ""
	for _, level := range levels {
		if best == "" || kspoya.LevelIndex(level) > kspoya.LevelIndex(best) {
			best = level
		}
	}

	resp := map[string]any{
		"has_active": false,
		"attempts":   attempts,
		"best_level": best,
		"bank_size":  len(kspoya.Bank),
		"per_test":   kspoya.QuestionsPerTest,
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
