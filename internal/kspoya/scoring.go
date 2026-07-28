package kspoya

import (
	crand "crypto/rand"
	"encoding/binary"
	"hash/fnv"
	"math/rand"
	"sort"
	"strconv"
)

// QuestionsPerTest — сколько вопросов получает ученик из банка в 120 штук.
const QuestionsPerTest = 40

// TestMinutes — сколько минут даётся на тест.
const TestMinutes = 40

// Quota — сколько вопросов каждого уровня попадает в один тест.
// Сумма = QuestionsPerTest.
//
// Перекос в сторону сложных: лёгких вопросов немного, потому что они почти
// никого не различают, а основную работу по определению уровня делают
// ступени B2 и C1.
var Quota = map[string]int{
	"A1": 3,
	"A2": 5,
	"B1": 7,
	"B2": 9,
	"C1": 9,
	"C2": 7,
}

// LevelThreshold — минимальный балл для уровня.
type LevelThreshold struct {
	Level    string `json:"level"`
	MinScore int    `json:"min_score"`
	MaxScore int    `json:"max_score"`
}

// LevelThresholds — шкала «балл → уровень», как в IELTS: сколько вопросов
// решил, такой уровень и получил. Никаких скрытых весов и коэффициентов.
//
// Диапазоны заданы вручную. Они хорошо ложатся на ожидаемый результат:
// ученик уровня X уверенно решает всё до X включительно и угадывает то,
// что выше, набирая примерно 9 / 13 / 18 / 25 / 32 / 38 баллов для A1...C2.
//
// Порядок — от высокого уровня к низкому, Grade берёт первый подходящий.
var LevelThresholds = []LevelThreshold{
	{Level: "C2", MinScore: 38, MaxScore: 40},
	{Level: "C1", MinScore: 32, MaxScore: 37},
	{Level: "B2", MinScore: 24, MaxScore: 31},
	{Level: "B1", MinScore: 18, MaxScore: 23},
	{Level: "A2", MinScore: 10, MaxScore: 17},
	{Level: "A1", MinScore: 0, MaxScore: 9},
}

// Reward — награда за подтверждённый уровень.
type Reward struct {
	XP    int
	Coins int
	Badge string
	Label string
}

// Rewards — награды и значки по уровням. Значки подобраны так, чтобы не
// пересекаться с эмодзи из кейсов магазина: их нельзя получить иначе как тестом.
var Rewards = map[string]Reward{
	"A1": {XP: 0, Coins: 0, Badge: "🔰", Label: "A1 — Начальный"},
	"A2": {XP: 300, Coins: 80, Badge: "📗", Label: "A2 — Элементарный"},
	"B1": {XP: 750, Coins: 200, Badge: "📘", Label: "B1 — Средний"},
	"B2": {XP: 1000, Coins: 300, Badge: "🏅", Label: "B2 — Выше среднего"},
	"C1": {XP: 1500, Coins: 400, Badge: "🔮", Label: "C1 — Продвинутый"},
	"C2": {XP: 2000, Coins: 500, Badge: "🌈", Label: "C2 — Мастерство"},
}

// LevelIndex возвращает порядковый номер уровня (A1 = 0 ... C2 = 5).
func LevelIndex(level string) int {
	for i, l := range Levels {
		if l == level {
			return i
		}
	}
	return 0
}

// LevelForScore возвращает уровень по числу верных ответов.
func LevelForScore(correct int) string {
	for _, t := range LevelThresholds {
		if correct >= t.MinScore {
			return t.Level
		}
	}
	return "A1"
}

// newRand создаёт генератор, засеянный из crypto/rand, — чтобы наборы вопросов
// нельзя было предсказать по времени запроса.
func newRand() *rand.Rand {
	var b [8]byte
	if _, err := crand.Read(b[:]); err != nil {
		// crypto/rand недоступен — тест всё равно должен работать.
		return rand.New(rand.NewSource(1))
	}
	return rand.New(rand.NewSource(int64(binary.LittleEndian.Uint64(b[:]))))
}

// SelectQuestions отбирает 40 вопросов по квоте из каждого уровня сложности,
// затем перемешивает итоговый список, чтобы вопросы не шли от простых к сложным.
func SelectQuestions() []int {
	return selectQuestions(newRand())
}

// selectQuestions вынесен отдельно, чтобы тесты могли подать управляемый
// генератор и получать воспроизводимые наборы.
func selectQuestions(rng *rand.Rand) []int {
	byLevel := make(map[string][]int, len(Levels))
	for _, question := range Bank {
		byLevel[question.Level] = append(byLevel[question.Level], question.ID)
	}

	var selected []int
	for _, level := range Levels {
		pool := append([]int(nil), byLevel[level]...)
		sort.Ints(pool)
		rng.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })

		want := Quota[level]
		if want > len(pool) {
			want = len(pool)
		}
		selected = append(selected, pool[:want]...)
	}

	rng.Shuffle(len(selected), func(i, j int) { selected[i], selected[j] = selected[j], selected[i] })
	return selected
}

// ShuffleOptions переставляет варианты ответа вопроса и возвращает новый индекс
// правильного варианта.
//
// Порядок зависит от пары (сессия, вопрос) и потому одинаков при выдаче
// вопроса и при проверке ответа — хранить перестановку в базе не нужно.
// Благодаря перемешиванию правильный ответ равномерно распределён по позициям,
// и стратегия «всегда жать один и тот же вариант» не работает.
func ShuffleOptions(sessionID string, question Question) (options []string, correct int) {
	h := fnv.New64a()
	h.Write([]byte(sessionID))
	h.Write([]byte{0})
	h.Write([]byte(strconv.Itoa(question.ID)))
	rng := rand.New(rand.NewSource(int64(h.Sum64())))

	perm := rng.Perm(len(question.Options))
	options = make([]string, len(question.Options))
	for newIdx, oldIdx := range perm {
		options[newIdx] = question.Options[oldIdx]
		if oldIdx == question.Correct {
			correct = newIdx
		}
	}
	return options, correct
}

// Outcome — результат проверки одной попытки.
type Outcome struct {
	Correct    int             // сколько верных ответов из 40
	Total      int             // сколько вопросов было задано
	Answered   int             // на сколько вопросов ученик вообще ответил
	BestStreak int             // самая длинная серия верных ответов подряд
	Percent    int             // доля верных ответов, 0..100
	Level      string          // итоговый уровень A1..C2
	ByLevel    map[string]Band // разбивка по уровням сложности
	ByTopic    map[string]Band // разбивка по разделам грамматики
}

// Band — сколько верных ответов из скольких в одной группе.
type Band struct {
	Correct int `json:"correct"`
	Total   int `json:"total"`
}

// Grade проверяет ответы и определяет уровень.
//
// sessionID нужен, чтобы восстановить тот же порядок вариантов ответа,
// в каком вопрос был показан ученику;
// questionIDs — вопросы в том порядке, в каком они были выданы;
// answers[i] — выбранный вариант для questionIDs[i], либо -1, если ответа нет.
//
// Уровень определяется только числом верных ответов по шкале LevelThresholds.
// Разбивка по ступеням и темам считается для аналитики и на уровень не влияет.
func Grade(sessionID string, questionIDs []int, answers []int) Outcome {
	out := Outcome{
		Total:   len(questionIDs),
		ByLevel: map[string]Band{},
		ByTopic: map[string]Band{},
	}

	streak := 0
	for i, id := range questionIDs {
		question, ok := ByID[id]
		if !ok {
			continue
		}

		levelBand := out.ByLevel[question.Level]
		topicBand := out.ByTopic[question.Topic]
		levelBand.Total++
		topicBand.Total++

		// Восстанавливаем ту же перестановку вариантов, что видел ученик.
		_, correctIdx := ShuffleOptions(sessionID, question)

		answer := -1
		if i < len(answers) {
			answer = answers[i]
		}

		switch {
		case answer == correctIdx:
			out.Correct++
			out.Answered++
			levelBand.Correct++
			topicBand.Correct++

			// Серия считается по порядку выдачи вопросов; пропуск и
			// неверный ответ одинаково её обрывают.
			streak++
			if streak > out.BestStreak {
				out.BestStreak = streak
			}
		case answer >= 0 && answer < len(question.Options):
			out.Answered++
			streak = 0
		default:
			streak = 0
		}

		out.ByLevel[question.Level] = levelBand
		out.ByTopic[question.Topic] = topicBand
	}

	if out.Total > 0 {
		out.Percent = int(float64(out.Correct) / float64(out.Total) * 100)
	}
	out.Level = LevelForScore(out.Correct)
	return out
}
