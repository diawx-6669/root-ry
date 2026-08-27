package handlers

import (
	"encoding/csv"
	"net/http"
	"strconv"
	"strings"

	"rootry/internal/store"
)

// Исследовательский контур.
//
// РКНП — конкурс научных проектов, и половина оценки — не функции, а
// доказательство, что они работают. Доказательство собирается здесь:
// две группы, два замера на одном банке вопросов, обезличенная выгрузка
// и сводка, которую видно в любой момент, а не за вечер до защиты.

// GET /api/admin/research/summary — сводка по группам эксперимента.
func (h *Handler) ResearchSummary(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	stats, err := h.store.ResearchSummary()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Не удалось посчитать сводку")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"groups": stats,
		// Что означают группы — прямо в ответе, чтобы отчёт можно было
		// читать без исходников.
		"legend": map[string]string{
			store.GroupFull:    "полный продукт: интервальные повторения и тетрадь ошибок",
			store.GroupControl: "контроль: те же уроки без возвратов к пройденному",
		},
	})
}

// GET /api/admin/research/export — обезличенная выгрузка эксперимента.
func (h *Handler) ResearchExport(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	rows, err := h.store.ResearchExport()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Не удалось собрать выгрузку")
		return
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="rootry-research.csv"`)
	// BOM и точка с запятой: без них Excel открывает русский текст
	// кракозябрами и сваливает всю строку в одну ячейку.
	_, _ = w.Write([]byte{0xEF, 0xBB, 0xBF})

	cw := csv.NewWriter(w)
	cw.Comma = ';'
	defer cw.Flush()

	_ = cw.Write([]string{
		"участник", "группа", "предтест", "посттест", "прирост", "вопросов_в_тесте",
		"тем_начато", "тем_освоено", "забываний", "ответов_в_повторениях",
		"ответов_всего", "ответов_с_разбором", "дней_занятий",
	})

	for _, r := range rows {
		pre, post, gain := "", "", ""
		if r.Pre.Valid {
			pre = strconv.FormatInt(r.Pre.Int64, 10)
		}
		if r.Post.Valid {
			post = strconv.FormatInt(r.Post.Int64, 10)
		}
		if r.Pre.Valid && r.Post.Valid {
			gain = strconv.FormatInt(r.Post.Int64-r.Pre.Int64, 10)
		}
		_ = cw.Write([]string{
			r.Subject, r.Group, pre, post, gain, strconv.Itoa(r.Total),
			strconv.Itoa(r.Topics), strconv.Itoa(r.Mastered), strconv.Itoa(r.Lapses),
			strconv.Itoa(r.Reviews), strconv.Itoa(r.Answers), strconv.Itoa(r.Hinted),
			strconv.Itoa(r.Days),
		})
	}
}

// POST /api/admin/research/group — отнести ученика к группе.
//
// Распределение делает человек, а не код. Случайное распределение
// прямо в приложении выглядело бы научнее, но на практике учитель
// формирует группы из готовых классов, и притворяться, что это
// рандомизация, нечестно — так и надо будет написать в работе.
func (h *Handler) ResearchSetGroup(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	var req struct {
		Username string `json:"username"`
		Group    string `json:"group"`
	}
	if err := h.parseBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}
	group := strings.ToUpper(strings.TrimSpace(req.Group))
	if err := h.store.SetStudyGroup(strings.TrimSpace(req.Username), group); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"username": req.Username, "group": group})
}

// POST /api/research/consent — согласие ученика на участие.
//
// Отдельно от группы и обязательно: обезличенные данные всё равно
// остаются данными школьников, и брать их без спроса нельзя. Без
// согласия ученик не попадает в выгрузку ни при какой группе.
func (h *Handler) ResearchConsent(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	user, ok := h.store.GetUserByUsername(getUsernameFromCtx(r))
	if !ok {
		writeError(w, http.StatusNotFound, "User not found")
		return
	}
	var req struct {
		Agree bool `json:"agree"`
	}
	if err := h.parseBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}
	if err := h.store.SetResearchConsent(user.ID, req.Agree); err != nil {
		writeError(w, http.StatusInternalServerError, "Не удалось сохранить согласие")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"consent": req.Agree})
}
