package kspoya

import (
	"math/rand"
	"os"
	"testing"
)

// Набор должен давать РОВНО заданное число верных ответов: балл
// известен заранее, ещё до проверки.
func TestPrefillAnswersHitTarget(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	ids := selectQuestions(rng)

	for _, target := range []int{0, 1, 20, 25, 30, len(ids)} {
		answers := PrefillAnswers("demo-session", ids, target, rand.New(rand.NewSource(int64(target))))
		if len(answers) != len(ids) {
			t.Fatalf("target %d: длина ответов %d, ожидалось %d", target, len(answers), len(ids))
		}
		out := Grade("demo-session", ids, answers)
		if out.Correct != target {
			t.Errorf("target %d: получилось %d верных", target, out.Correct)
		}
		// Пропусков быть не должно: пустой ответ в разборе выглядит как
		// брошенный тест, а нужен тест с ошибками.
		if out.Answered != len(ids) {
			t.Errorf("target %d: отвечено %d из %d", target, out.Answered, len(ids))
		}
	}
}

// Цель всегда попадает в объявленный диапазон 20–30.
func TestPrefillTargetInRange(t *testing.T) {
	for i := 0; i < 200; i++ {
		got := PrefillTarget(QuestionsPerTest)
		if got < PrefillMinScore || got > PrefillMaxScore {
			t.Fatalf("цель %d вне диапазона %d..%d", got, PrefillMinScore, PrefillMaxScore)
		}
	}
}

// Балл из целевого диапазона обязан давать B1 или B2 — уровень, на
// котором в карте пробелов есть и провалы, и удержанные разделы.
func TestPrefillRangeGivesMiddleLevel(t *testing.T) {
	for score := PrefillMinScore; score <= PrefillMaxScore; score++ {
		level := LevelForScore(score)
		if level != "B1" && level != "B2" {
			t.Errorf("балл %d даёт уровень %s, ожидались B1/B2", score, level)
		}
	}
}

// Без переменной окружения подстановки не существует.
func TestPrefillDisabledByDefault(t *testing.T) {
	t.Setenv("KSPOYA_PREFILL", "")
	if PrefillEnabled() {
		t.Error("подстановка включена без KSPOYA_PREFILL")
	}
	t.Setenv("KSPOYA_PREFILL", "1")
	if !PrefillEnabled() {
		t.Error("KSPOYA_PREFILL=1 не включает подстановку")
	}
	t.Setenv("KSPOYA_PREFILL", "0")
	if PrefillEnabled() {
		t.Error("KSPOYA_PREFILL=0 включает подстановку")
	}
	_ = os.Unsetenv("KSPOYA_PREFILL")
}
