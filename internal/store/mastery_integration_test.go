package store

import (
	"os"
	"testing"
	"time"

	"rootry/internal/mastery"
)

// Интеграционные тесты модели знаний.
//
// Проверяют то, что чистыми тестами не проверить: транзакцию записи ответа,
// апсерты, поведение SQL с датами. Запускаются только если задан
// DATABASE_URL — на локальной базе из devtools/localdb:
//
//	cd devtools/localdb && go run .
//	DATABASE_URL="postgres://rootry:rootry@localhost:5433/rootry?sslmode=disable" go test ./internal/store/
//
// Без переменной молча пропускаются, чтобы обычный `go test ./...` работал
// на машине без базы.

func testStore(t *testing.T) *Store {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL не задан — интеграционные тесты пропущены")
	}
	s, err := NewWithDSN(dsn)
	if err != nil {
		t.Fatalf("не подключиться к базе: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

// testUser создаёт временного ученика и удаляет его после теста.
// Каскад по внешним ключам уносит и ответы, и состояние тем.
func testUser(t *testing.T, s *Store) int64 {
	t.Helper()
	name := "t_" + time.Now().Format("150405.000000")
	u, err := s.CreateUser(name, "Тест", "password123")
	if err != nil {
		t.Fatalf("не создать пользователя: %v", err)
	}
	t.Cleanup(func() { _, _ = s.db.Exec(`DELETE FROM users WHERE id=$1`, u.ID) })
	return u.ID
}

// TestRecordAttemptTracksTopicAndItem — базовый круг: ответ записывается,
// состояние темы и задания появляется.
func TestRecordAttemptTracksTopicAndItem(t *testing.T) {
	s := testStore(t)
	uid := testUser(t, s)
	today := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)

	in := AttemptInput{
		UserID: uid, TopicID: "fon-zvuki", ItemID: "fon-zvuki#0",
		Source: "lesson", Correct: false, TimeMs: 4200,
	}
	out, err := s.RecordAttempt(in, today)
	if err != nil {
		t.Fatalf("RecordAttempt: %v", err)
	}
	if out.TopicWas.Box != 0 {
		t.Errorf("до первого ответа коробка была %d, ожидался 0", out.TopicWas.Box)
	}
	if out.Topic.Box != 1 {
		t.Errorf("после ответа коробка %d, ожидалась 1", out.Topic.Box)
	}
	if out.TopicPromoted() {
		t.Error("первое открытие темы засчиталось как повышение коробки")
	}

	if n := s.MistakeCount(uid); n != 1 {
		t.Errorf("в тетради %d заданий, ожидалось 1", n)
	}

	states, err := s.TopicMasteryMap(uid)
	if err != nil {
		t.Fatalf("TopicMasteryMap: %v", err)
	}
	st, ok := states["fon-zvuki"]
	if !ok {
		t.Fatal("тема не попала в карту прогресса")
	}
	if st.Total != 1 || st.Correct != 0 {
		t.Errorf("счётчики темы: correct=%d total=%d, ожидалось 0/1", st.Correct, st.Total)
	}
}

// TestMistakeClosesAfterTwoDays — задание уходит из тетради только после
// двух верных ответов в разные дни, и ни одним раньше.
func TestMistakeClosesAfterTwoDays(t *testing.T) {
	s := testStore(t)
	uid := testUser(t, s)
	day1 := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)

	in := AttemptInput{
		UserID: uid, TopicID: "orf-n-nn", ItemID: "orf-n-nn#3",
		Source: "lesson", Correct: false,
	}
	if _, err := s.RecordAttempt(in, day1); err != nil {
		t.Fatalf("ошибка записи: %v", err)
	}
	if n := s.MistakeCount(uid); n != 1 {
		t.Fatalf("после ошибки в тетради %d, ожидалось 1", n)
	}

	// Тут же переделал верно — этого мало.
	in.Correct = true
	in.Source = "review"
	if _, err := s.RecordAttempt(in, day1); err != nil {
		t.Fatalf("ошибка записи: %v", err)
	}
	if n := s.MistakeCount(uid); n != 1 {
		t.Errorf("задание закрылось верным ответом в тот же день (осталось %d)", n)
	}

	// Второй день.
	if _, err := s.RecordAttempt(in, day1.AddDate(0, 0, 1)); err != nil {
		t.Fatalf("ошибка записи: %v", err)
	}
	if n := s.MistakeCount(uid); n != 1 {
		t.Errorf("задание закрылось после одного дня возврата (осталось %d)", n)
	}

	// Третий день — закрывается.
	out, err := s.RecordAttempt(in, day1.AddDate(0, 0, 2))
	if err != nil {
		t.Fatalf("ошибка записи: %v", err)
	}
	if !out.MistakeClosed() {
		t.Error("MistakeClosed не сработал на закрывающем ответе")
	}
	if n := s.MistakeCount(uid); n != 0 {
		t.Errorf("задание не ушло из тетради: осталось %d", n)
	}

	open, err := s.OpenMistakes(uid, 50)
	if err != nil {
		t.Fatalf("OpenMistakes: %v", err)
	}
	if len(open) != 0 {
		t.Errorf("OpenMistakes вернул %d записей, ожидался 0", len(open))
	}
}

// TestWrongCountOnlyGrowsOnErrors — счётчик ошибок не должен расти от
// верных ответов, иначе сортировка тетради врёт.
func TestWrongCountOnlyGrowsOnErrors(t *testing.T) {
	s := testStore(t)
	uid := testUser(t, s)
	day1 := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)

	in := AttemptInput{
		UserID: uid, TopicID: "lex-frazeologiya", ItemID: "lex-frazeologiya#1",
		Source: "lesson",
	}
	in.Correct = false
	mustRecord(t, s, in, day1)
	in.Correct = false
	mustRecord(t, s, in, day1.AddDate(0, 0, 1))
	in.Correct = true
	mustRecord(t, s, in, day1.AddDate(0, 0, 2))

	open, err := s.OpenMistakes(uid, 50)
	if err != nil {
		t.Fatalf("OpenMistakes: %v", err)
	}
	if len(open) != 1 {
		t.Fatalf("ожидалась одна запись, получено %d", len(open))
	}
	if open[0].WrongCount != 2 {
		t.Errorf("ошибок насчитано %d, ожидалось 2", open[0].WrongCount)
	}
	// last_wrong обязан остаться на дате ошибки, а не съехать на верный ответ.
	if want := day1.AddDate(0, 0, 1).Format("2006-01-02"); open[0].LastWrong != want {
		t.Errorf("дата последней ошибки %q, ожидалась %q", open[0].LastWrong, want)
	}
}

// TestClaimTopicRewardOncePerInterval — главная защита экономики: получить
// награду за тему дважды подряд нельзя.
func TestClaimTopicRewardOncePerInterval(t *testing.T) {
	s := testStore(t)
	uid := testUser(t, s)
	day1 := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)

	in := AttemptInput{
		UserID: uid, TopicID: "tex-abzac", ItemID: "tex-abzac#0",
		Source: "lesson", Correct: true,
	}
	mustRecord(t, s, in, day1)

	kind, next, err := s.ClaimTopicReward(uid, "tex-abzac", day1)
	if err != nil {
		t.Fatalf("ClaimTopicReward: %v", err)
	}
	if kind != RewardFirst {
		t.Errorf("первое прохождение дало %q, ожидалось %q", kind, RewardFirst)
	}

	// Повторный заход в тот же день — награды нет.
	kind, _, err = s.ClaimTopicReward(uid, "tex-abzac", day1)
	if err != nil {
		t.Fatalf("ClaimTopicReward: %v", err)
	}
	if kind != RewardNone {
		t.Errorf("повтор в тот же день дал %q, ожидалось %q", kind, RewardNone)
	}

	// Когда интервал вышел — повторение оплачивается.
	kind, _, err = s.ClaimTopicReward(uid, "tex-abzac", next)
	if err != nil {
		t.Fatalf("ClaimTopicReward: %v", err)
	}
	if kind != RewardReview {
		t.Errorf("просроченная тема дала %q, ожидалось %q", kind, RewardReview)
	}

	// И сразу после этого — снова ничего.
	kind, _, err = s.ClaimTopicReward(uid, "tex-abzac", next)
	if err != nil {
		t.Fatalf("ClaimTopicReward: %v", err)
	}
	if kind != RewardNone {
		t.Errorf("две награды подряд за одно повторение: %q", kind)
	}
}

// TestClaimRewardWithoutAttempts — урок пройден старым клиентом, который
// не шлёт ответы. Прогресс всё равно не должен потеряться.
func TestClaimRewardWithoutAttempts(t *testing.T) {
	s := testStore(t)
	uid := testUser(t, s)
	today := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)

	kind, next, err := s.ClaimTopicReward(uid, "graf-alfavit", today)
	if err != nil {
		t.Fatalf("ClaimTopicReward: %v", err)
	}
	if kind != RewardFirst {
		t.Errorf("получено %q, ожидалось %q", kind, RewardFirst)
	}
	if !next.After(today) {
		t.Errorf("следующая награда назначена на %v, а сегодня %v", next, today)
	}

	states, err := s.TopicMasteryMap(uid)
	if err != nil {
		t.Fatalf("TopicMasteryMap: %v", err)
	}
	if st, ok := states["graf-alfavit"]; !ok || st.Box != 1 {
		t.Errorf("тема не завелась в первой коробке: %+v", st)
	}
}

// TestDueTopicsAppearInMap — просроченная тема должна опознаваться как
// требующая повторения. Ровно на этом держится план на сегодня.
func TestDueTopicsAppearInMap(t *testing.T) {
	s := testStore(t)
	uid := testUser(t, s)
	past := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)

	in := AttemptInput{
		UserID: uid, TopicID: "kul-etiket", ItemID: "kul-etiket#0",
		Source: "lesson", Correct: true,
	}
	mustRecord(t, s, in, past)

	states, err := s.TopicMasteryMap(uid)
	if err != nil {
		t.Fatalf("TopicMasteryMap: %v", err)
	}
	st := states["kul-etiket"]

	later := past.AddDate(0, 0, 10)
	if !st.IsDue(later) {
		t.Errorf("тема из первой коробки не просрочилась за 10 дней: due_on=%v", st.DueOn)
	}
	if got := st.Overdue(later); got != 9 {
		t.Errorf("просрочка %d дней, ожидалось 9", got)
	}
	if st.Status(later) != mastery.StatusDue {
		t.Errorf("статус %q, ожидался %q", st.Status(later), mastery.StatusDue)
	}
}

func mustRecord(t *testing.T, s *Store, in AttemptInput, at time.Time) AttemptOutcome {
	t.Helper()
	out, err := s.RecordAttempt(in, at)
	if err != nil {
		t.Fatalf("RecordAttempt(%s, correct=%v): %v", in.ItemID, in.Correct, err)
	}
	return out
}

// TestGameMistakesStayOutOfNotebook — ошибка в игре роняет тему, но в
// тетрадь ошибок не попадает: отдельное задание игры нельзя открыть,
// значит и показывать его как «прорешать» нечестно.
func TestGameMistakesStayOutOfNotebook(t *testing.T) {
	s := testStore(t)
	uid := testUser(t, s)
	today := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)

	mustRecord(t, s, AttemptInput{
		UserID: uid, TopicID: "orf-n-nn", ItemID: "orf-n-nn#g-grammar-derevyannyj",
		Source: "game", Correct: false,
	}, today)
	mustRecord(t, s, AttemptInput{
		UserID: uid, TopicID: "orf-n-nn", ItemID: "orf-n-nn#2",
		Source: "lesson", Correct: false,
	}, today)

	open, err := s.OpenMistakes(uid, 50)
	if err != nil {
		t.Fatalf("OpenMistakes: %v", err)
	}
	if len(open) != 1 {
		t.Fatalf("в тетради %d записей, ожидалась одна (только из урока): %+v", len(open), open)
	}
	if open[0].ItemID != "orf-n-nn#2" {
		t.Errorf("в тетради оказалось %q, ожидалось задание урока", open[0].ItemID)
	}
	if n := s.MistakeCount(uid); n != 1 {
		t.Errorf("счётчик тетради %d, ожидался 1", n)
	}

	// Но тема всё равно упала и вернётся в план повторения.
	states, err := s.TopicMasteryMap(uid)
	if err != nil {
		t.Fatalf("TopicMasteryMap: %v", err)
	}
	if st := states["orf-n-nn"]; st.Box != 1 || st.Total != 2 {
		t.Errorf("ошибка в игре не отразилась на теме: %+v", st)
	}
}
