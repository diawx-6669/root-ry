package handlers

import (
	"net/http"
	"sort"
	"strings"

	"rootry/internal/mastery"
	"rootry/internal/models"
	"rootry/internal/store"
	"rootry/internal/timeutil"
	"rootry/internal/topics"
)

// allowedSources — откуда может прийти ответ. Список закрытый: source
// попадает в исследовательские выгрузки, и «lesson», «Lesson» и «уроk»
// в одной колонке сделали бы их бесполезными.
var allowedSources = map[string]bool{
	"lesson": true,
	"game":   true,
	"kspoya": true,
	"review": true,
}

// maxItemIDLen — предел длины идентификатора задания. Он приходит от
// клиента и уходит в базу, так что размер ограничиваем явно.
const maxItemIDLen = 96

// FullHintLevel — уровень подсказки, на котором ученику показан полный
// разбор. Подсказки идут ступенями: 1 — намёк, 2 — правило, 3 — разбор.
//
// Начиная с этого уровня верный ответ перестаёт быть подтверждением
// знания темы (см. mastery.Answer.Assisted). Первые две ступени на модель
// знаний не влияют: намёк и правило — это помощь думать, а не ответ.
const FullHintLevel = 3

// POST /api/attempt — записать один ответ ученика.
//
// Это самый горячий эндпоинт продукта: он вызывается на каждое задание в
// каждом уроке и в каждой игре. Награду он не выдаёт намеренно — иначе
// достаточно было бы слать ответы в цикле. Награда считается там, где её
// нельзя повторить: при завершении темы.
func (h *Handler) Attempt(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	user, ok := h.store.GetUserByUsername(getUsernameFromCtx(r))
	if !ok {
		writeError(w, http.StatusNotFound, "User not found")
		return
	}

	var req models.AttemptRequest
	if err := h.parseBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	topic, known := topics.Get(strings.TrimSpace(req.Topic))
	if !known {
		writeError(w, http.StatusBadRequest, "Неизвестная тема")
		return
	}

	item := strings.TrimSpace(req.Item)
	if item == "" || len(item) > maxItemIDLen {
		writeError(w, http.StatusBadRequest, "Неверный идентификатор задания")
		return
	}
	// Задание обязано принадлежать своей теме: иначе один и тот же item_id
	// мог бы приезжать с разными темами и портить статистику по разделам.
	if !strings.HasPrefix(item, topic.ID+"#") {
		writeError(w, http.StatusBadRequest, "Задание не принадлежит теме")
		return
	}

	source := strings.TrimSpace(req.Source)
	if !allowedSources[source] {
		writeError(w, http.StatusBadRequest, "Неизвестный источник ответа")
		return
	}

	now := timeutil.Now()
	outcome, err := h.store.RecordAttempt(store.AttemptInput{
		UserID:   user.ID,
		TopicID:  topic.ID,
		ItemID:   item,
		Source:   source,
		Correct:  req.Correct,
		Hints:    clampInt(req.Hints, 0, FullHintLevel),
		TimeMs:   clampInt(req.TimeMs, 0, 30*60*1000),
		Assisted: req.Hints >= FullHintLevel,
	}, now)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Не удалось записать ответ")
		return
	}

	writeJSON(w, http.StatusOK, models.AttemptResponse{
		Status:        string(outcome.Topic.Status(now)),
		Box:           outcome.Topic.Box,
		MaxBox:        mastery.MaxBox,
		DueOn:         outcome.Topic.DueOn.Format("2006-01-02"),
		TopicPromoted: outcome.TopicPromoted(),
		Assisted:      req.Hints >= FullHintLevel,
		MistakeClosed: outcome.MistakeClosed(),
		MistakeCount:  h.store.MistakeCount(user.ID),
	})
}

// GET /api/progress — состояние всех тем для дерева грамматики.
//
// Отдаются только темы, которых ученик уже касался. Нетронутые узлы дерево
// и так рисует состоянием «не открыта», и гонять по сети 74 пустые записи
// незачем.
func (h *Handler) Progress(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	user, ok := h.store.GetUserByUsername(getUsernameFromCtx(r))
	if !ok {
		writeError(w, http.StatusNotFound, "User not found")
		return
	}

	states, err := h.store.TopicMasteryMap(user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Не удалось прочитать прогресс")
		return
	}

	now := timeutil.Now()
	out := make([]models.TopicProgress, 0, len(states))
	for id, st := range states {
		out = append(out, models.TopicProgress{
			TopicID:     id,
			Status:      string(st.Status(now)),
			Box:         st.Box,
			MaxBox:      mastery.MaxBox,
			Strength:    round2(st.Strength(now)),
			OverdueDays: st.Overdue(now),
			DueOn:       st.DueOn.Format("2006-01-02"),
			Correct:     st.Correct,
			Total:       st.Total,
			Lapses:      st.Lapses,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TopicID < out[j].TopicID })

	writeJSON(w, http.StatusOK, map[string]any{
		"topics":       out,
		"total_topics": topics.Count(),
		"mistakes":     h.store.MistakeCount(user.ID),
	})
}

// GET /api/review/plan — что повторить сегодня.
//
// Порядок — по просрочке: самое забытое первым. Так ученик, вернувшийся
// после месяца перерыва, начинает с того, что развалилось сильнее всего,
// а не с того, что случайно оказалось первым в дереве.
func (h *Handler) ReviewPlan(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	user, ok := h.store.GetUserByUsername(getUsernameFromCtx(r))
	if !ok {
		writeError(w, http.StatusNotFound, "User not found")
		return
	}

	states, err := h.store.TopicMasteryMap(user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Не удалось построить план")
		return
	}

	now := timeutil.Now()
	plan := models.ReviewPlan{
		Today:      timeutil.Today(),
		Due:        []models.ReviewPlanItem{},
		Mistakes:   h.store.MistakeCount(user.ID),
		TotalTopic: topics.Count(),
	}

	for id, st := range states {
		topic, known := topics.Get(id)
		if !known {
			// Тема исчезла из реестра после правки дерева — молча пропускаем,
			// иначе план сломается на одной устаревшей записи.
			continue
		}
		plan.Learned++
		if st.Status(now) == mastery.StatusMastered {
			plan.Mastered++
		}
		if !st.IsDue(now) {
			continue
		}
		xp, coins := topics.ReviewReward(topic)
		plan.Due = append(plan.Due, models.ReviewPlanItem{
			TopicID:     topic.ID,
			Title:       topic.Title,
			Section:     topic.Section,
			Box:         st.Box,
			OverdueDays: st.Overdue(now),
			DueOn:       st.DueOn.Format("2006-01-02"),
			XP:          xp,
			Coins:       coins,
		})
	}

	sort.Slice(plan.Due, func(i, j int) bool {
		if plan.Due[i].OverdueDays != plan.Due[j].OverdueDays {
			return plan.Due[i].OverdueDays > plan.Due[j].OverdueDays
		}
		return plan.Due[i].TopicID < plan.Due[j].TopicID
	})

	writeJSON(w, http.StatusOK, plan)
}

// GET /api/mistakes — тетрадь ошибок: незакрытые задания ученика.
func (h *Handler) Mistakes(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	user, ok := h.store.GetUserByUsername(getUsernameFromCtx(r))
	if !ok {
		writeError(w, http.StatusNotFound, "User not found")
		return
	}

	items, err := h.store.OpenMistakes(user.ID, 200)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Не удалось прочитать тетрадь ошибок")
		return
	}

	// Сколько разных дней подряд нужно ответить верно, чтобы задание ушло из
	// тетради. Клиент показывает это ученику, чтобы прогресс был понятен.
	writeJSON(w, http.StatusOK, map[string]any{
		"items":           items,
		"count":           len(items),
		"days_to_resolve": mastery.PromoteAfterDays,
	})
}

// clampInt зажимает присланное клиентом число в разумные границы.
func clampInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// round2 округляет до двух знаков: в JSON не нужна вся мантисса float64.
func round2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}

// contains — есть ли строка в срезе. В store есть HasBadge с той же
// механикой, но звать «HasBadge» для списка тем — значит запутать
// следующего, кто будет это читать.
func contains(list []string, v string) bool {
	for _, item := range list {
		if item == v {
			return true
		}
	}
	return false
}
