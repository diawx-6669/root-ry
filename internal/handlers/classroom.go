package handlers

import (
	"net/http"
	"strings"

	"rootry/internal/kspoya"
	"rootry/internal/store"
	"rootry/internal/topics"
)

// ═══════════════════════════════════════════════════════════════════
//  Классрум: участники эксперимента видят друг друга, администратор —
//  полную картину по каждому.
//
//  Вход — галочка «Участие в исследовании» в профиле. Без неё ученик
//  в классрум не попадает и чужие карточки не видит: согласие на общий
//  класс и согласие на исследование — одно и то же решение ученика.
// ═══════════════════════════════════════════════════════════════════

// classroomAccess: может ли пользователь смотреть классрум и админ ли он.
func (h *Handler) classroomAccess(w http.ResponseWriter, r *http.Request) (username string, admin, ok bool) {
	username = getUsernameFromCtx(r)
	admin = h.store.IsAdmin(username)
	if !admin && !h.store.InClassroom(username) {
		writeError(w, http.StatusForbidden,
			"Классрум открывается после согласия на участие в исследовании (профиль → настройки)")
		return username, false, false
	}
	return username, admin, true
}

// GET /api/classroom — участники классрума.
func (h *Handler) Classroom(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	me, admin, ok := h.classroomAccess(w, r)
	if !ok {
		return
	}
	members, err := h.store.ClassroomMembers()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Не удалось загрузить классрум")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"me":           me,
		"is_admin":     admin,
		"members":      members,
		"total_topics": topics.Count(),
	})
}

// kspoyaMistake — вопрос КСПОЯ, на который ученик ответил неверно.
type kspoyaMistake struct {
	Text    string `json:"text"`
	Topic   string `json:"topic"`
	Level   string `json:"level"`
	Given   string `json:"given"`
	Correct string `json:"correct"`
	Explain string `json:"explain"`
}

// GET /api/classroom/student?u=логин — карточка участника.
// Ученик получает открытую часть, администратор — ещё и отчёт.
func (h *Handler) ClassroomStudent(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	_, admin, ok := h.classroomAccess(w, r)
	if !ok {
		return
	}
	target := strings.TrimSpace(r.URL.Query().Get("u"))
	member, found := h.store.ClassroomMember(target)
	if !found {
		writeError(w, http.StatusNotFound, "Такого участника в классруме нет")
		return
	}
	resp := map[string]any{
		"member":       member,
		"total_topics": topics.Count(),
	}
	if !admin {
		writeJSON(w, http.StatusOK, resp)
		return
	}

	rep, err := h.store.StudentReport(target)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Не удалось собрать отчёт")
		return
	}

	// Темы: что пройдено, что нет — в порядке дерева.
	done := map[string]bool{}
	for _, t := range rep.CompletedTopics {
		done[t] = true
	}
	type topicRow struct {
		ID      string `json:"id"`
		Title   string `json:"title"`
		Section string `json:"section"`
		Done    bool   `json:"done"`
	}
	tree := make([]topicRow, 0, topics.Count())
	for _, t := range topics.All {
		tree = append(tree, topicRow{t.ID, t.Title, t.Section, done[t.ID]})
	}

	// КСПОЯ: попытки и вопросы с ошибками последней попытки.
	type kspoyaRow struct {
		store.KspoyaRun
		Mistakes []kspoyaMistake `json:"mistakes"`
	}
	runs := make([]kspoyaRow, 0, len(rep.Kspoya))
	for i, k := range rep.Kspoya {
		row := kspoyaRow{KspoyaRun: k, Mistakes: []kspoyaMistake{}}
		// Разбор отдаём для трёх последних попыток: дальше он уже
		// неактуален, а ответ раздувается до сотен килобайт.
		if i < 3 {
			for _, it := range buildReview(k.ID, k.QuestionIDs, k.Answers) {
				if it.IsCorrect {
					continue
				}
				given := "нет ответа"
				if it.UserAnswer >= 0 && it.UserAnswer < len(it.Options) {
					given = it.Options[it.UserAnswer]
				}
				correct := ""
				if it.Correct >= 0 && it.Correct < len(it.Options) {
					correct = it.Options[it.Correct]
				}
				row.Mistakes = append(row.Mistakes, kspoyaMistake{
					Text: it.Text, Topic: it.Topic, Level: it.Level,
					Given: given, Correct: correct, Explain: it.Explain,
				})
			}
		}
		runs = append(runs, row)
	}

	titles := make(map[string]string, topics.Count())
	for _, t := range topics.All {
		titles[t.ID] = t.Title
	}
	resp["report"] = map[string]any{
		"tree":          tree,
		"sources":       rep.Sources,
		"topics":        rep.Topics,
		"recent_wrong":  rep.RecentWrong,
		"open_mistakes": rep.OpenMistakes,
		"games":         rep.Games,
		"kspoya":        runs,
		"active_days":   rep.ActiveDays,
		"total_answers": rep.TotalAnswers,
		"topic_titles":  titles,
		"kspoya_levels": kspoya.LevelThresholds,
	}
	writeJSON(w, http.StatusOK, resp)
}
