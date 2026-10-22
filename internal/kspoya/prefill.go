package kspoya

import (
	"math/rand"
	"os"
	"strconv"
	"strings"
)

// ═══════════════════════════════════════════════════════════════════
//  Служебная подстановка ответов.
//
//  Собирает набор ответов на выданную попытку с заранее заданным числом
//  верных. Нужна там, где важен не процесс решения, а то, что система
//  выдаёт после него: уровень, карта пробелов, маршрут и разбор.
//  Такой набор нельзя посчитать в браузере — правильные варианты до
//  завершения теста туда не уходят.
//
//  По умолчанию выключено. Включается переменной окружения
//  KSPOYA_PREFILL=1; без неё обработчик отвечает 404.
// ═══════════════════════════════════════════════════════════════════

// Целевой диапазон верных ответов. 20–30 из 40 — это B1/B2: уровень,
// на котором в карте пробелов есть и провалы, и удержанные разделы,
// то есть видно, что диагностика вообще что-то различает.
const (
	PrefillMinScore = 20
	PrefillMaxScore = 30
)

// PrefillEnabled сообщает, включена ли служебная подстановка.
func PrefillEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("KSPOYA_PREFILL"))) {
	case "1", "true", "yes", "on", "да":
		return true
	}
	return false
}

// prefillBounds читает границы целевого балла: KSPOYA_PREFILL_MIN / _MAX,
// если их задали, иначе константы выше. Значения вне 0..QuestionsPerTest
// и перевёрнутый диапазон игнорируются — опечатка в переменной окружения
// не должна ронять обработчик.
func prefillBounds(total int) (int, int) {
	min, max := PrefillMinScore, PrefillMaxScore
	if v, err := strconv.Atoi(strings.TrimSpace(os.Getenv("KSPOYA_PREFILL_MIN"))); err == nil {
		min = v
	}
	if v, err := strconv.Atoi(strings.TrimSpace(os.Getenv("KSPOYA_PREFILL_MAX"))); err == nil {
		max = v
	}
	if min < 0 {
		min = 0
	}
	if max > total {
		max = total
	}
	if min > max {
		min, max = max, min
	}
	if min > total {
		min = total
	}
	return min, max
}

// PrefillAnswers собирает ответы на выданные вопросы так, чтобы верных
// оказалось ровно target штук, а остальные были неверными (но всё же
// выбранными — пропусков нет, иначе разбор выглядел бы как брошенный
// тест, а не как тест с ошибками).
//
// Верные ответы распределяются по тесту случайно, а не идут подряд:
// иначе карта пробелов показала бы ровно те разделы, что стоят в
// конце набора, — то есть артефакт порядка, а не диагностику.
func PrefillAnswers(sessionID string, questionIDs []int, target int, rng *rand.Rand) []int {
	if rng == nil {
		rng = newRand()
	}
	total := len(questionIDs)
	if target < 0 {
		target = 0
	}
	if target > total {
		target = total
	}

	// Какие позиции сделать верными.
	positions := rng.Perm(total)
	right := make(map[int]bool, target)
	for _, p := range positions[:target] {
		right[p] = true
	}

	answers := make([]int, total)
	for i, id := range questionIDs {
		question, ok := ByID[id]
		if !ok {
			answers[i] = -1
			continue
		}
		options, correct := ShuffleOptions(sessionID, question)
		if right[i] {
			answers[i] = correct
			continue
		}
		// Неверный вариант: любой, кроме правильного.
		wrong := rng.Intn(len(options) - 1)
		if wrong >= correct {
			wrong++
		}
		answers[i] = wrong
	}
	return answers
}

// PrefillTarget выбирает случайное число верных ответов в целевом диапазоне.
func PrefillTarget(total int) int {
	min, max := prefillBounds(total)
	if max <= min {
		return min
	}
	return min + newRand().Intn(max-min+1)
}
