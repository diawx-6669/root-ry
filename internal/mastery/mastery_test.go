package mastery

import (
	"testing"
	"time"
)

// d — короткая запись даты для сценариев ниже.
func d(year int, month time.Month, dayOfMonth int) time.Time {
	return time.Date(year, month, dayOfMonth, 12, 0, 0, 0, time.UTC)
}

// TestPromotionNeedsSeparateDays — главное свойство всей модели.
//
// Ученик может решить восемь заданий подряд за один вечер и выглядеть
// отличником. Это ничего не значит: правило он прочитал десять минут назад.
// Тема поднимается по коробкам только за верные ответы в РАЗНЫЕ дни.
func TestPromotionNeedsSeparateDays(t *testing.T) {
	day1 := d(2026, time.September, 1)

	s := State{}
	for i := 0; i < 8; i++ {
		s = Apply(s, true, day1)
	}
	if s.Box != 1 {
		t.Errorf("восемь верных ответов за один день подняли тему до коробки %d, ожидалась 1", s.Box)
	}
	if s.Total != 8 || s.Correct != 8 {
		t.Errorf("счётчики ответов сбились: correct=%d total=%d", s.Correct, s.Total)
	}

	// Следующий день — первый засчитанный день серии.
	s = Apply(s, true, day1.AddDate(0, 0, 1))
	if s.Box != 1 {
		t.Errorf("после первого дня серии коробка %d, ожидалась 1", s.Box)
	}
	if s.StreakDays != 1 {
		t.Errorf("серия %d дней, ожидался 1", s.StreakDays)
	}

	// Второй засчитанный день — подъём.
	s = Apply(s, true, day1.AddDate(0, 0, 2))
	if s.Box != 2 {
		t.Errorf("после двух разных дней коробка %d, ожидалась 2", s.Box)
	}
	if s.StreakDays != 0 {
		t.Errorf("после подъёма серия должна обнуляться, получено %d", s.StreakDays)
	}
}

// TestIntervalsExpand проверяет, что интервал растёт вместе с коробкой:
// освоенная тема должна всплывать всё реже, иначе повторения превращаются
// в бессмысленную ежедневную обязаловку.
func TestIntervalsExpand(t *testing.T) {
	today := d(2026, time.September, 1)
	s := State{}

	seen := make([]int, 0, MaxBox)
	cur := today
	for box := 1; box <= MaxBox; box++ {
		// Два разных дня подряд поднимают на коробку выше.
		s = Apply(s, true, cur)
		cur = cur.AddDate(0, 0, 1)
		s = Apply(s, true, cur)
		cur = cur.AddDate(0, 0, 1)
		gap := int(day(s.DueOn).Sub(day(s.LastSeen)).Hours() / 24)
		seen = append(seen, gap)
	}

	for i := 1; i < len(seen); i++ {
		if seen[i] < seen[i-1] {
			t.Errorf("интервал сократился: %v", seen)
			break
		}
	}
	if s.Box != MaxBox {
		t.Errorf("тема не дошла до последней коробки: %d", s.Box)
	}
	// Дальше последней коробки подниматься некуда.
	before := s.Box
	s = Apply(s, true, cur.AddDate(0, 0, 1))
	s = Apply(s, true, cur.AddDate(0, 0, 2))
	if s.Box != before {
		t.Errorf("коробка ушла за предел: %d > %d", s.Box, before)
	}
}

// TestWrongAnswerResets — ошибка возвращает тему в самое начало и назначает
// повторение на завтра. Без этого забытая тема осталась бы «освоенной».
func TestWrongAnswerResets(t *testing.T) {
	today := d(2026, time.September, 10)
	s := State{Box: 4, StreakDays: 1, LastSeen: today.AddDate(0, 0, -10)}

	s = Apply(s, false, today)

	if s.Box != 1 {
		t.Errorf("после ошибки коробка %d, ожидалась 1", s.Box)
	}
	if s.StreakDays != 0 {
		t.Errorf("после ошибки серия %d, ожидался 0", s.StreakDays)
	}
	if s.Lapses != 1 {
		t.Errorf("забывание не засчиталось: lapses=%d", s.Lapses)
	}
	if want := day(today.AddDate(0, 0, 1)); !day(s.DueOn).Equal(want) {
		t.Errorf("повторение назначено на %v, ожидалось %v", day(s.DueOn), want)
	}
}

// TestFirstFailureIsNotALapse — провал на первом знакомстве с темой это не
// «забыл», а «ещё не знал». Иначе счётчик забываний завышал бы статистику
// в исследовании.
func TestFirstFailureIsNotALapse(t *testing.T) {
	s := Apply(State{}, false, d(2026, time.September, 1))
	if s.Lapses != 0 {
		t.Errorf("первая же ошибка засчиталась как забывание: lapses=%d", s.Lapses)
	}
	if s.Box != 1 {
		t.Errorf("коробка %d, ожидалась 1", s.Box)
	}
}

// TestIsDueAndStatus — как тема выглядит в дереве на каждой стадии.
func TestIsDueAndStatus(t *testing.T) {
	today := d(2026, time.September, 20)

	cases := []struct {
		name   string
		state  State
		due    bool
		status Status
	}{
		{"не изучали", State{}, false, StatusNew},
		{"изучается, срок не подошёл",
			State{Box: 2, DueOn: today.AddDate(0, 0, 2), LastSeen: today}, false, StatusLearning},
		{"освоена, срок не подошёл",
			State{Box: 4, DueOn: today.AddDate(0, 0, 5), LastSeen: today}, false, StatusMastered},
		{"срок сегодня — пора повторить",
			State{Box: 4, DueOn: today, LastSeen: today.AddDate(0, 0, -16)}, true, StatusDue},
		{"просрочено на неделю",
			State{Box: 3, DueOn: today.AddDate(0, 0, -7), LastSeen: today.AddDate(0, 0, -14)}, true, StatusDue},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.state.IsDue(today); got != c.due {
				t.Errorf("IsDue = %v, ожидалось %v", got, c.due)
			}
			if got := c.state.Status(today); got != c.status {
				t.Errorf("Status = %q, ожидалось %q", got, c.status)
			}
		})
	}
}

// TestNewTopicIsNotDue — самая обидная ошибка, если её допустить: все 74
// темы разом свалились бы в план повторения ещё до первого урока.
func TestNewTopicIsNotDue(t *testing.T) {
	if (State{}).IsDue(d(2026, time.September, 1)) {
		t.Error("нетронутая тема попала в план повторения")
	}
	if (State{}).Strength(d(2026, time.September, 1)) != 0 {
		t.Error("у нетронутой темы ненулевая сила")
	}
}

// TestStrengthDecays — просроченная тема должна выцветать в дереве, иначе
// экран прогресса расходится с реальными знаниями.
func TestStrengthDecays(t *testing.T) {
	today := d(2026, time.September, 20)
	fresh := State{Box: 4, DueOn: today.AddDate(0, 0, 3), LastSeen: today}
	stale := State{Box: 4, DueOn: today.AddDate(0, 0, -16), LastSeen: today.AddDate(0, 0, -32)}

	if stale.Strength(today) >= fresh.Strength(today) {
		t.Errorf("просроченная тема не потеряла силу: %.2f против %.2f",
			stale.Strength(today), fresh.Strength(today))
	}
	if s := fresh.Strength(today); s <= 0 || s > 1 {
		t.Errorf("сила вне диапазона 0..1: %.2f", s)
	}
	if s := stale.Strength(today); s < 0 || s > 1 {
		t.Errorf("сила вне диапазона 0..1: %.2f", s)
	}
}

// TestOverdueSortsPlan — план на сегодня сортируется по просрочке, чтобы
// самое забытое шло первым.
func TestOverdueSortsPlan(t *testing.T) {
	today := d(2026, time.September, 20)
	s := State{Box: 2, DueOn: today.AddDate(0, 0, -5)}
	if got := s.Overdue(today); got != 5 {
		t.Errorf("просрочка %d дней, ожидалось 5", got)
	}
	future := State{Box: 2, DueOn: today.AddDate(0, 0, 3)}
	if got := future.Overdue(today); got != 0 {
		t.Errorf("будущая дата дала просрочку %d, ожидался 0", got)
	}
}

// TestResolvedClosesMistake — задание уходит из тетради ошибок только после
// двух верных ответов в разные дни. Ответ через минуту после разбора — это
// память о только что прочитанном.
func TestResolvedClosesMistake(t *testing.T) {
	day1 := d(2026, time.September, 1)

	s := Apply(State{}, false, day1) // ошибся, задание попало в тетрадь
	if s.Resolved() {
		t.Fatal("задание закрылось сразу после ошибки")
	}

	s = Apply(s, true, day1) // тут же переделал верно
	if s.Resolved() {
		t.Error("задание закрылось верным ответом в тот же день")
	}

	s = Apply(s, true, day1.AddDate(0, 0, 1))
	if s.Resolved() {
		t.Error("задание закрылось после одного засчитанного дня")
	}

	s = Apply(s, true, day1.AddDate(0, 0, 2))
	if !s.Resolved() {
		t.Error("задание не закрылось после двух верных дней подряд")
	}
}

// TestSameDayRepeatDoesNotPunish — переделывать задание в тот же день должно
// быть безопасно. Если бы повтор сбрасывал серию, работа над ошибками
// наказывала бы ученика за старательность.
func TestSameDayRepeatDoesNotPunish(t *testing.T) {
	day1 := d(2026, time.September, 1)
	s := Apply(State{}, true, day1)
	s = Apply(s, true, day1.AddDate(0, 0, 1)) // первый засчитанный день
	before := s.StreakDays

	s = Apply(s, true, day1.AddDate(0, 0, 1)) // ещё раз в тот же день
	if s.StreakDays != before {
		t.Errorf("повтор в тот же день сдвинул серию: было %d, стало %d", before, s.StreakDays)
	}
}

// TestApplyIsPure — Apply не должна менять переданное состояние на месте:
// обработчик рассчитывает награду по разнице «было / стало».
func TestApplyIsPure(t *testing.T) {
	today := d(2026, time.September, 1)
	before := State{Box: 3, StreakDays: 1, Total: 10, Correct: 8}
	snapshot := before

	_ = Apply(before, false, today)

	if before != snapshot {
		t.Error("Apply изменила исходное состояние")
	}
}

// TestAssistedAnswerDoesNotConfirm — ответ, данный после открытого разбора,
// не двигает тему вперёд.
//
// Это то, ради чего подсказки вообще устроены так, а не через уменьшение
// XP: иначе выгоднее всего было бы сразу открыть разбор и получить чуть
// меньше монет, но полный прогресс.
func TestAssistedAnswerDoesNotConfirm(t *testing.T) {
	day1 := d(2026, time.September, 1)

	// Самостоятельно набираем один засчитанный день.
	s := Apply(State{}, true, day1)
	s = Apply(s, true, day1.AddDate(0, 0, 1))
	if s.StreakDays != 1 {
		t.Fatalf("подготовка не удалась: серия %d", s.StreakDays)
	}

	before := s
	s = ApplyAnswer(s, Answer{Correct: true, Assisted: true}, day1.AddDate(0, 0, 2))

	if s.StreakDays != before.StreakDays {
		t.Errorf("подсказанный ответ сдвинул серию: было %d, стало %d",
			before.StreakDays, s.StreakDays)
	}
	if s.Box != before.Box {
		t.Errorf("подсказанный ответ поднял коробку: было %d, стало %d", before.Box, s.Box)
	}
	if !day(s.DueOn).Equal(day(before.DueOn)) {
		t.Errorf("подсказанный ответ отодвинул повторение: было %v, стало %v",
			day(before.DueOn), day(s.DueOn))
	}
	// Но ответ всё равно засчитан в статистику: он был.
	if s.Total != before.Total+1 || s.Correct != before.Correct+1 {
		t.Errorf("подсказанный ответ не попал в счётчики: correct=%d total=%d",
			s.Correct, s.Total)
	}
}

// TestAssistedAnswerDoesNotPunish — открыть разбор не должно быть страшно.
// Это нормальный способ разобраться, а не провинность.
func TestAssistedAnswerDoesNotPunish(t *testing.T) {
	today := d(2026, time.September, 10)
	before := State{Box: 4, StreakDays: 1, DueOn: today.AddDate(0, 0, -2),
		LastSeen: today.AddDate(0, 0, -18)}

	s := ApplyAnswer(before, Answer{Correct: true, Assisted: true}, today)

	if s.Box != before.Box {
		t.Errorf("коробка изменилась: было %d, стало %d", before.Box, s.Box)
	}
	if s.Lapses != before.Lapses {
		t.Errorf("засчитано забывание: было %d, стало %d", before.Lapses, s.Lapses)
	}
	// Просроченная тема остаётся просроченной: ученик её ещё не подтвердил.
	if !s.IsDue(today) {
		t.Error("тема перестала быть просроченной после подсказанного ответа")
	}
}

// TestAssistedWrongAnswerStillDrops — если ученик открыл разбор и всё равно
// ответил неверно, это обычная ошибка со всеми последствиями.
func TestAssistedWrongAnswerStillDrops(t *testing.T) {
	today := d(2026, time.September, 10)
	before := State{Box: 4, StreakDays: 1, LastSeen: today.AddDate(0, 0, -10)}

	s := ApplyAnswer(before, Answer{Correct: false, Assisted: true}, today)

	if s.Box != 1 {
		t.Errorf("после ошибки коробка %d, ожидалась 1", s.Box)
	}
	if s.Lapses != 1 {
		t.Errorf("забывание не засчиталось: %d", s.Lapses)
	}
}

// TestAssistedFirstEncounterStartsTopic — первое знакомство с темой через
// подсказку всё равно должно завести тему, иначе она не появится ни в
// дереве, ни в плане повторения.
func TestAssistedFirstEncounterStartsTopic(t *testing.T) {
	today := d(2026, time.September, 1)
	s := ApplyAnswer(State{}, Answer{Correct: true, Assisted: true}, today)

	if s.Box != 1 {
		t.Errorf("тема не завелась: коробка %d", s.Box)
	}
	if s.DueOn.IsZero() {
		t.Error("не назначена дата повторения")
	}
	if !s.IsDue(today.AddDate(0, 0, 3)) {
		t.Error("тема не всплыла в плане через три дня")
	}
}
