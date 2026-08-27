package store

import (
	crand "crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"rootry/internal/mastery"
	"rootry/internal/models"
)

// ErrClassNotFound возвращается, когда кода такого класса нет.
var ErrClassNotFound = errors.New("class not found")

// ErrNotTeacher возвращается, когда чужой класс пытаются посмотреть или
// изменить. Отдельная ошибка, а не «не найден»: обработчик должен ответить
// 403, а не 404, иначе по кодам ответа можно перебирать чужие классы.
var ErrNotTeacher = errors.New("not the teacher of this class")

// codeAlphabet — символы кода приглашения.
//
// Ноль, О, единица, I и L выброшены намеренно: код диктуют вслух у доски
// и переписывают с экрана, а «0» и «O» на слух неразличимы.
const codeAlphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"

// codeLength — длина кода. Шесть символов из 31 дают около 900 миллионов
// комбинаций: подобрать перебором нереально, продиктовать легко.
const codeLength = 6

func newClassCode() (string, error) {
	var b strings.Builder
	for i := 0; i < codeLength; i++ {
		n, err := crand.Int(crand.Reader, big.NewInt(int64(len(codeAlphabet))))
		if err != nil {
			return "", err
		}
		b.WriteByte(codeAlphabet[n.Int64()])
	}
	return b.String(), nil
}

// CreateClass заводит класс и возвращает его вместе с кодом приглашения.
func (s *Store) CreateClass(teacherID int64, name string) (models.Class, error) {
	// Коллизия кода маловероятна, но не невозможна. Несколько попыток
	// дешевле, чем объяснять учителю, почему класс не создался.
	for attempt := 0; attempt < 5; attempt++ {
		code, err := newClassCode()
		if err != nil {
			return models.Class{}, fmt.Errorf("CreateClass: code: %w", err)
		}

		var c models.Class
		err = s.db.QueryRow(`
			INSERT INTO classes (code, name, teacher_id)
			VALUES ($1,$2,$3)
			RETURNING id, code, name, teacher_id, created_at`,
			code, name, teacherID,
		).Scan(&c.ID, &c.Code, &c.Name, &c.TeacherID, &c.CreatedAt)

		if err == nil {
			return c, nil
		}
		if isUniqueViolation(err) {
			continue
		}
		return models.Class{}, fmt.Errorf("CreateClass: %w", err)
	}
	return models.Class{}, errors.New("CreateClass: не удалось подобрать свободный код")
}

// JoinClass добавляет ученика в класс по коду.
//
// Повторное вступление ошибкой не считается: ученик мог ввести код дважды
// или уже состоять в классе, и пугать его сообщением незачем.
func (s *Store) JoinClass(userID int64, code string) (models.Class, error) {
	code = strings.ToUpper(strings.TrimSpace(code))

	var c models.Class
	err := s.db.QueryRow(`
		SELECT id, code, name, teacher_id, created_at
		FROM classes WHERE code=$1 AND NOT archived`, code,
	).Scan(&c.ID, &c.Code, &c.Name, &c.TeacherID, &c.CreatedAt)
	switch {
	case err == sql.ErrNoRows:
		return models.Class{}, ErrClassNotFound
	case err != nil:
		return models.Class{}, fmt.Errorf("JoinClass: %w", err)
	}

	if _, err := s.db.Exec(`
		INSERT INTO class_members (class_id, user_id) VALUES ($1,$2)
		ON CONFLICT (class_id, user_id) DO NOTHING`, c.ID, userID); err != nil {
		return models.Class{}, fmt.Errorf("JoinClass: insert: %w", err)
	}
	return c, nil
}

// TeacherClasses — классы, которые ведёт учитель, со счётчиком учеников.
func (s *Store) TeacherClasses(teacherID int64) ([]models.Class, error) {
	rows, err := s.db.Query(`
		SELECT c.id, c.code, c.name, c.teacher_id, c.created_at,
		       (SELECT COUNT(*) FROM class_members m WHERE m.class_id = c.id)
		FROM classes c
		WHERE c.teacher_id=$1 AND NOT c.archived
		ORDER BY c.created_at DESC`, teacherID)
	if err != nil {
		return nil, fmt.Errorf("TeacherClasses: %w", err)
	}
	defer rows.Close()

	out := []models.Class{}
	for rows.Next() {
		var c models.Class
		if err := rows.Scan(&c.ID, &c.Code, &c.Name, &c.TeacherID, &c.CreatedAt, &c.Students); err != nil {
			return nil, fmt.Errorf("TeacherClasses: scan: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// StudentClasses — классы, в которых состоит ученик.
func (s *Store) StudentClasses(userID int64) ([]models.Class, error) {
	rows, err := s.db.Query(`
		SELECT c.id, c.code, c.name, c.teacher_id, c.created_at, u.nickname
		FROM class_members m
		JOIN classes c ON c.id = m.class_id
		JOIN users u   ON u.id = c.teacher_id
		WHERE m.user_id=$1 AND NOT c.archived
		ORDER BY m.joined_at DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("StudentClasses: %w", err)
	}
	defer rows.Close()

	out := []models.Class{}
	for rows.Next() {
		var c models.Class
		if err := rows.Scan(&c.ID, &c.Code, &c.Name, &c.TeacherID, &c.CreatedAt, &c.TeacherName); err != nil {
			return nil, fmt.Errorf("StudentClasses: scan: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// OwnedClass читает класс и проверяет, что он принадлежит этому учителю.
func (s *Store) OwnedClass(teacherID, classID int64) (models.Class, error) {
	var c models.Class
	err := s.db.QueryRow(`
		SELECT id, code, name, teacher_id, created_at
		FROM classes WHERE id=$1 AND NOT archived`, classID,
	).Scan(&c.ID, &c.Code, &c.Name, &c.TeacherID, &c.CreatedAt)
	switch {
	case err == sql.ErrNoRows:
		return models.Class{}, ErrClassNotFound
	case err != nil:
		return models.Class{}, fmt.Errorf("OwnedClass: %w", err)
	}
	if c.TeacherID != teacherID {
		return models.Class{}, ErrNotTeacher
	}
	return c, nil
}

// ClassHeatmap собирает состояние всех тем у всех учеников класса.
//
// Это главный экран учителя. Красная полоса поперёк карты означает, что
// тему провалил весь класс, — и это разговор для урока, а не для
// индивидуальной работы. Красная полоса вдоль — отстал один ученик.
//
// Два запроса вместо N+1: учеников в классе тридцать, тем семьдесят
// четыре, и запрос на каждую пару превратил бы экран в минутное ожидание.
func (s *Store) ClassHeatmap(classID int64, now time.Time) (models.ClassHeatmap, error) {
	out := models.ClassHeatmap{
		Students: []models.ClassStudent{},
		Cells:    map[string]map[string]string{},
	}

	rows, err := s.db.Query(`
		SELECT u.id, u.username, u.nickname, u.xp,
		       (SELECT COUNT(*) FROM item_state i
		         WHERE i.user_id = u.id AND NOT i.resolved AND i.wrong_count > 0
		           AND position('#g-' in i.item_id) = 0)
		FROM class_members m
		JOIN users u ON u.id = m.user_id
		WHERE m.class_id=$1
		ORDER BY u.nickname, u.username`, classID)
	if err != nil {
		return out, fmt.Errorf("ClassHeatmap: students: %w", err)
	}
	defer rows.Close()

	ids := []int64{}
	for rows.Next() {
		var st models.ClassStudent
		if err := rows.Scan(&st.ID, &st.Username, &st.Nickname, &st.XP, &st.Mistakes); err != nil {
			return out, fmt.Errorf("ClassHeatmap: scan student: %w", err)
		}
		out.Students = append(out.Students, st)
		ids = append(ids, st.ID)
	}
	if err := rows.Err(); err != nil {
		return out, err
	}
	if len(ids) == 0 {
		return out, nil
	}

	mrows, err := s.db.Query(`
		SELECT tm.user_id, tm.topic_id, tm.box, tm.due_on
		FROM topic_mastery tm
		JOIN class_members m ON m.user_id = tm.user_id AND m.class_id = $1`, classID)
	if err != nil {
		return out, fmt.Errorf("ClassHeatmap: mastery: %w", err)
	}
	defer mrows.Close()

	byUser := map[int64]int{}
	for i, st := range out.Students {
		byUser[st.ID] = i
	}

	for mrows.Next() {
		var (
			userID  int64
			topicID string
			st      mastery.State
			dueOn   time.Time
		)
		if err := mrows.Scan(&userID, &topicID, &st.Box, &dueOn); err != nil {
			return out, fmt.Errorf("ClassHeatmap: scan mastery: %w", err)
		}
		st.DueOn = dueOn

		if out.Cells[topicID] == nil {
			out.Cells[topicID] = map[string]string{}
		}
		out.Cells[topicID][itoa64(userID)] = string(st.Status(now))

		if i, ok := byUser[userID]; ok {
			out.Students[i].Started++
			if st.Status(now) == mastery.StatusMastered {
				out.Students[i].Mastered++
			}
			if st.IsDue(now) {
				out.Students[i].Due++
			}
		}
	}
	return out, mrows.Err()
}

// ClassExportRow — строка исследовательской выгрузки.
type ClassExportRow struct {
	Student   string
	TopicID   string
	Box       int
	Correct   int
	Total     int
	Lapses    int
	DueOn     string
	FirstDone string
}

// ClassExport отдаёт обезличенную выгрузку по классу.
//
// Логины в выгрузку не попадают: для исследования нужен идентификатор,
// который позволяет связать строки одного ученика между собой, а не
// узнать, кто это. Настоящее сопоставление остаётся у учителя.
func (s *Store) ClassExport(classID int64) ([]ClassExportRow, error) {
	rows, err := s.db.Query(`
		SELECT tm.user_id, tm.topic_id, tm.box, tm.correct, tm.total,
		       tm.lapses, tm.due_on, tm.first_done
		FROM topic_mastery tm
		JOIN class_members m ON m.user_id = tm.user_id AND m.class_id = $1
		ORDER BY tm.user_id, tm.topic_id`, classID)
	if err != nil {
		return nil, fmt.Errorf("ClassExport: %w", err)
	}
	defer rows.Close()

	// Порядковый номер вместо идентификатора: даже id ученика не должен
	// уезжать в файл, который потом окажется в приложении к работе.
	seq := map[int64]int{}
	out := []ClassExportRow{}
	for rows.Next() {
		var (
			userID    int64
			r         ClassExportRow
			dueOn     time.Time
			firstDone sql.NullTime
		)
		if err := rows.Scan(&userID, &r.TopicID, &r.Box, &r.Correct, &r.Total,
			&r.Lapses, &dueOn, &firstDone); err != nil {
			return nil, fmt.Errorf("ClassExport: scan: %w", err)
		}
		n, ok := seq[userID]
		if !ok {
			n = len(seq) + 1
			seq[userID] = n
		}
		r.Student = "ученик-" + itoa64(int64(n))
		r.DueOn = dueOn.Format("2006-01-02")
		if firstDone.Valid {
			r.FirstDone = firstDone.Time.Format("2006-01-02")
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// SetTeacher выдаёт или снимает роль учителя. Вызывается только из админки.
func (s *Store) SetTeacher(username string, on bool) error {
	res, err := s.db.Exec(`UPDATE users SET is_teacher=$2 WHERE username=$1`, username, on)
	if err != nil {
		return fmt.Errorf("SetTeacher: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return errors.New("пользователь не найден")
	}
	return nil
}

// IsTeacher сообщает, есть ли у пользователя роль учителя.
func (s *Store) IsTeacher(userID int64) bool {
	var on bool
	if err := s.db.QueryRow(`SELECT is_teacher FROM users WHERE id=$1`, userID).Scan(&on); err != nil {
		return false
	}
	return on
}

// isUniqueViolation — нарушение UNIQUE, а не любая ошибка вставки.
func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), uniqueViolation)
}

func itoa64(n int64) string {
	return fmt.Sprintf("%d", n)
}
