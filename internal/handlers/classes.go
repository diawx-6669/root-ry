package handlers

import (
	"encoding/csv"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"rootry/internal/models"
	"rootry/internal/store"
	"rootry/internal/timeutil"
	"rootry/internal/topics"
)

// Пределы на текстовые поля класса. Название видит весь класс, и
// простыня на экран здесь никому не нужна.
const (
	classNameMin = 2
	classNameMax = 60
)

// POST /api/class/create — учитель заводит класс.
func (h *Handler) ClassCreate(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	user, ok := h.store.GetUserByUsername(getUsernameFromCtx(r))
	if !ok {
		writeError(w, http.StatusNotFound, "User not found")
		return
	}
	if !h.store.IsTeacher(user.ID) {
		writeError(w, http.StatusForbidden, "Нужна роль учителя")
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := h.parseBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}
	name := strings.TrimSpace(req.Name)
	if n := len([]rune(name)); n < classNameMin || n > classNameMax {
		writeError(w, http.StatusBadRequest,
			fmt.Sprintf("Название класса — от %d до %d символов", classNameMin, classNameMax))
		return
	}

	class, err := h.store.CreateClass(user.ID, name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Не удалось создать класс")
		return
	}
	writeJSON(w, http.StatusOK, class)
}

// POST /api/class/join — ученик вступает в класс по коду.
func (h *Handler) ClassJoin(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	user, ok := h.store.GetUserByUsername(getUsernameFromCtx(r))
	if !ok {
		writeError(w, http.StatusNotFound, "User not found")
		return
	}

	var req struct {
		Code string `json:"code"`
	}
	if err := h.parseBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	class, err := h.store.JoinClass(user.ID, req.Code)
	switch {
	case errors.Is(err, store.ErrClassNotFound):
		writeError(w, http.StatusNotFound, "Класса с таким кодом нет")
		return
	case err != nil:
		writeError(w, http.StatusInternalServerError, "Не удалось вступить в класс")
		return
	}
	writeJSON(w, http.StatusOK, class)
}

// GET /api/class/my — какие классы у меня есть.
//
// Один эндпоинт на обе роли: страница класса не должна заранее знать,
// кто её открыл, иначе ученик-учитель (а такие будут) увидит половину.
func (h *Handler) ClassMy(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	user, ok := h.store.GetUserByUsername(getUsernameFromCtx(r))
	if !ok {
		writeError(w, http.StatusNotFound, "User not found")
		return
	}

	teaching := []models.Class{}
	isTeacher := h.store.IsTeacher(user.ID)
	if isTeacher {
		var err error
		teaching, err = h.store.TeacherClasses(user.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Не удалось прочитать классы")
			return
		}
	}

	member, err := h.store.StudentClasses(user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Не удалось прочитать классы")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"is_teacher": isTeacher,
		"teaching":   teaching,
		"member":     member,
	})
}

// GET /api/class/heatmap?id=… — тепловая карта класса.
//
// Главный экран учителя. Красная полоса поперёк карты означает, что тему
// провалил весь класс, и это разговор для урока; полоса вдоль — отстал
// один ученик, и это разговор с ним.
func (h *Handler) ClassHeatmap(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	user, class, ok := h.ownedClassFromQuery(w, r)
	if !ok {
		return
	}
	_ = user

	heat, err := h.store.ClassHeatmap(class.ID, timeutil.Now())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Не удалось построить карту")
		return
	}

	// Темы отдаём вместе с картой: клиенту нужен их порядок и названия,
	// а держать второй список в JS значило бы завести источник правды,
	// который разъедется с реестром.
	list := make([]map[string]string, 0, len(topics.All))
	for _, t := range topics.All {
		list = append(list, map[string]string{
			"id": t.ID, "title": t.Title, "section": t.Section,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"class":    class,
		"students": heat.Students,
		"cells":    heat.Cells,
		"topics":   list,
	})
}

// GET /api/class/export?id=… — обезличенная выгрузка по классу в CSV.
//
// Формат выбран под то, чем реально пользуются: Excel и Google Таблицы.
// Разделитель — точка с запятой, кодировка с BOM, иначе русский текст в
// Excel открывается кракозябрами и учитель решает, что выгрузка битая.
func (h *Handler) ClassExport(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	_, class, ok := h.ownedClassFromQuery(w, r)
	if !ok {
		return
	}

	rows, err := h.store.ClassExport(class.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Не удалось собрать выгрузку")
		return
	}

	title := make(map[string]string, len(topics.All))
	for _, t := range topics.All {
		title[t.ID] = t.Title
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition",
		fmt.Sprintf("attachment; filename=\"rootry-class-%d.csv\"", class.ID))
	_, _ = w.Write([]byte{0xEF, 0xBB, 0xBF})

	cw := csv.NewWriter(w)
	cw.Comma = ';'
	defer cw.Flush()

	_ = cw.Write([]string{
		"ученик", "тема", "название темы", "коробка",
		"верных", "всего", "забываний", "повторить", "изучено",
	})
	for _, row := range rows {
		_ = cw.Write([]string{
			row.Student, row.TopicID, title[row.TopicID], strconv.Itoa(row.Box),
			strconv.Itoa(row.Correct), strconv.Itoa(row.Total), strconv.Itoa(row.Lapses),
			row.DueOn, row.FirstDone,
		})
	}
}

// ownedClassFromQuery достаёт класс из ?id= и проверяет, что он принадлежит
// этому учителю.
//
// Чужой класс отвечает 403, а не 404, намеренно: если бы «не мой» и
// «не существует» отвечали одинаково, по кодам ответа можно было бы
// перебрать, какие классы вообще есть.
func (h *Handler) ownedClassFromQuery(w http.ResponseWriter, r *http.Request) (*models.User, models.Class, bool) {
	user, ok := h.store.GetUserByUsername(getUsernameFromCtx(r))
	if !ok {
		writeError(w, http.StatusNotFound, "User not found")
		return nil, models.Class{}, false
	}

	id, err := strconv.ParseInt(strings.TrimSpace(r.URL.Query().Get("id")), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "Не указан класс")
		return nil, models.Class{}, false
	}

	class, err := h.store.OwnedClass(user.ID, id)
	switch {
	case errors.Is(err, store.ErrClassNotFound):
		writeError(w, http.StatusNotFound, "Класс не найден")
		return nil, models.Class{}, false
	case errors.Is(err, store.ErrNotTeacher):
		writeError(w, http.StatusForbidden, "Это не ваш класс")
		return nil, models.Class{}, false
	case err != nil:
		writeError(w, http.StatusInternalServerError, "Не удалось прочитать класс")
		return nil, models.Class{}, false
	}
	return user, class, true
}

// POST /api/admin/teacher — выдать или снять роль учителя. Только админ.
func (h *Handler) AdminSetTeacher(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	var req struct {
		Username string `json:"username"`
		On       bool   `json:"on"`
	}
	if err := h.parseBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}
	if err := h.store.SetTeacher(strings.TrimSpace(req.Username), req.On); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"username": req.Username, "is_teacher": req.On})
}
