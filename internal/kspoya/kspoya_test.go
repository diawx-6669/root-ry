package kspoya

import (
	"math/rand"
	"strconv"
	"testing"
)

func TestBankIntegrity(t *testing.T) {
	if len(Bank) != 120 {
		t.Fatalf("в банке %d вопросов, ожидалось 120", len(Bank))
	}

	seenID := map[int]bool{}
	seenText := map[string]bool{}
	perLevel := map[string]int{}

	for _, q := range Bank {
		if seenID[q.ID] {
			t.Errorf("вопрос %d: идентификатор не уникален", q.ID)
		}
		seenID[q.ID] = true

		if seenText[q.Text] {
			t.Errorf("вопрос %d: дублирует формулировку другого вопроса", q.ID)
		}
		seenText[q.Text] = true

		if len(q.Options) != 4 {
			t.Errorf("вопрос %d: %d вариантов вместо 4", q.ID, len(q.Options))
		}
		if q.Correct < 0 || q.Correct >= len(q.Options) {
			t.Errorf("вопрос %d: correct=%d вне диапазона", q.ID, q.Correct)
		}
		if q.Text == "" || q.Topic == "" || q.Explain == "" {
			t.Errorf("вопрос %d: пустой текст, тема или разбор", q.ID)
		}
		if _, ok := weight[q.Level]; !ok {
			t.Errorf("вопрос %d: неизвестный уровень %q", q.ID, q.Level)
		}

		opts := map[string]bool{}
		for _, o := range q.Options {
			if o == "" {
				t.Errorf("вопрос %d: пустой вариант ответа", q.ID)
			}
			if opts[o] {
				t.Errorf("вопрос %d: вариант %q повторяется", q.ID, o)
			}
			opts[o] = true
		}
		perLevel[q.Level]++
	}

	for _, level := range Levels {
		if perLevel[level] != 20 {
			t.Errorf("уровень %s: %d вопросов, ожидалось 20", level, perLevel[level])
		}
	}
}

// После перемешивания правильный ответ должен равномерно распределяться по
// позициям A/B/C/D, иначе тест проходится стратегией «всегда жать вариант B».
func TestShuffledAnswersAreSpread(t *testing.T) {
	positions := map[int]int{}
	total := 0
	for session := 0; session < 200; session++ {
		sid := "session-" + strconv.Itoa(session)
		for _, question := range Bank {
			_, correct := ShuffleOptions(sid, question)
			positions[correct]++
			total++
		}
	}
	for pos := 0; pos < 4; pos++ {
		share := float64(positions[pos]) / float64(total)
		if share < 0.22 || share > 0.28 {
			t.Errorf("позиция %d — %.1f%% правильных ответов, ожидалось около 25%%",
				pos, share*100)
		}
	}
}

// Перестановка обязана быть стабильной: выдача и проверка должны совпадать.
func TestShuffleOptionsIsDeterministic(t *testing.T) {
	for _, question := range Bank[:10] {
		first, correctA := ShuffleOptions("fixed-session", question)
		second, correctB := ShuffleOptions("fixed-session", question)
		if correctA != correctB {
			t.Fatalf("вопрос %d: индекс правильного ответа нестабилен", question.ID)
		}
		for i := range first {
			if first[i] != second[i] {
				t.Fatalf("вопрос %d: порядок вариантов нестабилен", question.ID)
			}
		}
		if first[correctA] != question.Options[question.Correct] {
			t.Fatalf("вопрос %d: после перестановки правильным считается другой вариант", question.ID)
		}
	}
}

// А вот между разными сессиями порядок должен различаться.
func TestShuffleOptionsDiffersBetweenSessions(t *testing.T) {
	same := 0
	for _, question := range Bank {
		_, a := ShuffleOptions("session-a", question)
		_, b := ShuffleOptions("session-b", question)
		if a == b {
			same++
		}
	}
	if same > len(Bank)/2 {
		t.Errorf("в %d из %d вопросов позиция ответа совпала в двух разных сессиях",
			same, len(Bank))
	}
}

func TestSelectQuestions(t *testing.T) {
	total := 0
	for _, n := range Quota {
		total += n
	}
	if total != QuestionsPerTest {
		t.Fatalf("сумма квот %d, ожидалось %d", total, QuestionsPerTest)
	}

	ids := SelectQuestions()
	if len(ids) != QuestionsPerTest {
		t.Fatalf("отобрано %d вопросов, ожидалось %d", len(ids), QuestionsPerTest)
	}

	seen := map[int]bool{}
	perLevel := map[string]int{}
	for _, id := range ids {
		if seen[id] {
			t.Errorf("вопрос %d выдан дважды в одном тесте", id)
		}
		seen[id] = true
		perLevel[ByID[id].Level]++
	}
	for level, want := range Quota {
		if perLevel[level] != want {
			t.Errorf("уровень %s: выдано %d вопросов, ожидалось %d", level, perLevel[level], want)
		}
	}
}

// Наборы вопросов должны меняться от попытки к попытке.
func TestSelectQuestionsVaries(t *testing.T) {
	first := SelectQuestions()
	identical := 0
	for i := 0; i < 20; i++ {
		next := SelectQuestions()
		same := true
		for j := range first {
			if first[j] != next[j] {
				same = false
				break
			}
		}
		if same {
			identical++
		}
	}
	if identical > 0 {
		t.Errorf("%d из 20 наборов полностью совпали с первым", identical)
	}
}

// simulate разыгрывает попытку ученика с истинным уровнем trueLevel:
// вопросы своего уровня и ниже он решает с вероятностью 0.95,
// вопросы выше — угадывает наугад (1 из 4).
func simulate(rng *rand.Rand, trueLevel int, sessionID string, ids []int) []int {
	answers := make([]int, len(ids))
	for i, id := range ids {
		question := ByID[id]
		_, correct := ShuffleOptions(sessionID, question)
		if LevelIndex(question.Level) <= trueLevel && rng.Float64() < 0.95 {
			answers[i] = correct
			continue
		}
		answers[i] = rng.Intn(4)
	}
	return answers
}

// Ключевая проверка: тест должен возвращать примерно тот уровень,
// которым ученик реально владеет.
func TestGradeCalibration(t *testing.T) {
	// Оба генератора с фиксированным сидом: и выбор вопросов, и ответы
	// ученика. Иначе тест плавает от запуска к запуску и ничего не проверяет.
	rng := rand.New(rand.NewSource(42))
	pick := rand.New(rand.NewSource(2024))
	const runs = 400

	for trueLevel, levelName := range Levels {
		hits, offByOne, offByMore := 0, 0, 0
		for i := 0; i < runs; i++ {
			sid := "calib-" + strconv.Itoa(trueLevel) + "-" + strconv.Itoa(i)
			ids := selectQuestions(pick)
			out := Grade(sid, ids, simulate(rng, trueLevel, sid, ids))
			switch diff := LevelIndex(out.Level) - trueLevel; {
			case diff == 0:
				hits++
			case diff == -1 || diff == 1:
				offByOne++
			default:
				offByMore++
			}
		}
		exact := float64(hits) / runs
		within := float64(hits+offByOne) / runs
		far := float64(offByMore) / runs

		if exact < 0.90 {
			t.Errorf("%s: точное попадание лишь в %.1f%% случаев", levelName, exact*100)
		}
		if within < 0.97 {
			t.Errorf("%s: попадание ±1 уровень лишь в %.1f%% случаев", levelName, within*100)
		}
		if far > 0.04 {
			t.Errorf("%s: промах на две ступени и больше в %.1f%% случаев", levelName, far*100)
		}
		t.Logf("%s: точно %.1f%%, ±1 уровень %.1f%%, грубый промах %.1f%%",
			levelName, exact*100, within*100, far*100)
	}
}

// Тыканье наугад не должно давать ничего выше A2.
func TestRandomGuesserGetsLowestLevel(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	pick := rand.New(rand.NewSource(99))
	for i := 0; i < 2000; i++ {
		sid := "guess-" + strconv.Itoa(i)
		ids := selectQuestions(pick)
		answers := make([]int, len(ids))
		for j := range answers {
			answers[j] = rng.Intn(4)
		}
		out := Grade(sid, ids, answers)
		if LevelIndex(out.Level) > LevelIndex("A2") {
			t.Fatalf("случайные ответы дали уровень %s (%d%%)", out.Level, out.Percent)
		}
	}
}

// Пустой бланк — это A1, а не ошибка.
func TestNoAnswersGivesA1(t *testing.T) {
	ids := SelectQuestions()
	out := Grade("empty", ids, make([]int, 0))
	if out.Level != "A1" || out.Correct != 0 {
		t.Fatalf("пустой бланк дал уровень %s с %d верными", out.Level, out.Correct)
	}
	if out.Answered != 0 {
		t.Fatalf("Answered=%d при пустом бланке", out.Answered)
	}
}

// Все ответы верны — только C2.
func TestPerfectRunGivesC2(t *testing.T) {
	ids := SelectQuestions()
	answers := make([]int, len(ids))
	for i, id := range ids {
		_, correct := ShuffleOptions("perfect", ByID[id])
		answers[i] = correct
	}
	out := Grade("perfect", ids, answers)
	if out.Level != "C2" {
		t.Fatalf("идеальный результат дал %s (%d%%)", out.Level, out.Percent)
	}
	if out.Correct != QuestionsPerTest {
		t.Fatalf("верных ответов %d из %d", out.Correct, QuestionsPerTest)
	}
}

// answersWithBands собирает бланк, в котором на каждой ступени ровно
// столько верных ответов, сколько указано в want.
func answersWithBands(sessionID string, ids []int, want map[string]int) []int {
	done := map[string]int{}
	answers := make([]int, len(ids))
	for i, id := range ids {
		question := ByID[id]
		_, correct := ShuffleOptions(sessionID, question)
		if done[question.Level] < want[question.Level] {
			answers[i] = correct
			done[question.Level]++
		} else {
			answers[i] = (correct + 1) % 4
		}
	}
	return answers
}

// Реальный случай из жизни: 23 верных из 40 при слабой нижней ступени.
// Раньше провал A1 обнулял всё и выдавал A1, хотя A2 и B1 закрыты.
func TestWeakBottomBandDoesNotSinkResult(t *testing.T) {
	const sid = "weak-bottom"
	ids := selectQuestions(rand.New(rand.NewSource(11)))

	out := Grade(sid, ids, answersWithBands(sid, ids, map[string]int{
		"A1": 2, "A2": 5, "B1": 5, "B2": 4, "C1": 4, "C2": 3,
	}))

	if out.Correct != 23 {
		t.Fatalf("ожидалось 23 верных, получено %d", out.Correct)
	}
	if out.Level != "B1" {
		t.Errorf("23 из 40 с закрытыми A2 и B1 дали уровень %s, ожидался B1", out.Level)
	}
	if LevelIndex(out.Level) < LevelIndex("A2") {
		t.Errorf("уровень %s ниже A2 — одна слабая ступень снова обнуляет результат", out.Level)
	}
}

// Сильный ученик, провалившийся на одной узкой теме посередине,
// не должен падать на две ступени.
func TestSingleWeakBandIsForgivenWhenRestIsStrong(t *testing.T) {
	const sid = "one-gap"
	ids := selectQuestions(rand.New(rand.NewSource(12)))

	out := Grade(sid, ids, answersWithBands(sid, ids, map[string]int{
		"A1": 6, "A2": 7, "B1": 7, "B2": 2, "C1": 7, "C2": 2,
	}))

	if LevelIndex(out.Level) < LevelIndex("C1") {
		t.Errorf("при полностью закрытых A1-B1 и C1 получен уровень %s, ожидался C1", out.Level)
	}
}

// Обратная проверка: одна удачно угаданная верхняя ступень не должна
// поднимать уровень, если ступень под ней провалена.
func TestLuckyTopBandDoesNotLiftLevel(t *testing.T) {
	const sid = "lucky-top"
	ids := selectQuestions(rand.New(rand.NewSource(13)))

	out := Grade(sid, ids, answersWithBands(sid, ids, map[string]int{
		"A1": 6, "A2": 7, "B1": 7, "B2": 1, "C1": 6, "C2": 0,
	}))

	if LevelIndex(out.Level) > LevelIndex("B2") {
		t.Errorf("угаданная ступень C1 при проваленной B2 дала уровень %s", out.Level)
	}
}

// Максимальная серия верных ответов подряд.
func TestBestStreak(t *testing.T) {
	ids := SelectQuestions()
	const sid = "streak"

	correctAt := func(i int) int {
		_, c := ShuffleOptions(sid, ByID[ids[i]])
		return c
	}
	wrongAt := func(i int) int { return (correctAt(i) + 1) % 4 }

	// Всё верно — серия во весь тест.
	all := make([]int, len(ids))
	for i := range ids {
		all[i] = correctAt(i)
	}
	if got := Grade(sid, ids, all).BestStreak; got != len(ids) {
		t.Errorf("идеальный прогон: серия %d, ожидалось %d", got, len(ids))
	}

	// Ни одного верного — серии нет.
	none := make([]int, len(ids))
	for i := range ids {
		none[i] = wrongAt(i)
	}
	if got := Grade(sid, ids, none).BestStreak; got != 0 {
		t.Errorf("ни одного верного: серия %d, ожидалось 0", got)
	}

	// Две серии: 5 верных, ошибка, 3 верных, остальное неверно.
	mixed := make([]int, len(ids))
	for i := range ids {
		switch {
		case i < 5, i >= 6 && i < 9:
			mixed[i] = correctAt(i)
		default:
			mixed[i] = wrongAt(i)
		}
	}
	if got := Grade(sid, ids, mixed).BestStreak; got != 5 {
		t.Errorf("серии 5 и 3: получено %d, ожидалось 5", got)
	}

	// Пропуск обрывает серию так же, как ошибка.
	skip := make([]int, len(ids))
	for i := range ids {
		skip[i] = correctAt(i)
	}
	skip[4] = -1
	if got := Grade(sid, ids, skip).BestStreak; got != len(ids)-5 {
		t.Errorf("пропуск на 5-м вопросе: серия %d, ожидалось %d", got, len(ids)-5)
	}
}

// Пропуск не должен штрафоваться сильнее, чем неверный ответ.
func TestSkippingIsNotPunishedMoreThanGuessing(t *testing.T) {
	ids := SelectQuestions()

	skipped := make([]int, len(ids))
	wrong := make([]int, len(ids))
	for i, id := range ids {
		_, correct := ShuffleOptions("skip", ByID[id])
		skipped[i] = -1
		wrong[i] = (correct + 1) % 4
	}

	if Grade("skip", ids, skipped).Percent < Grade("skip", ids, wrong).Percent {
		t.Fatal("пропуск вопросов оценивается хуже, чем заведомо неверные ответы")
	}
}
