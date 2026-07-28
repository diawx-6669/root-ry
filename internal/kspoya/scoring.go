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

// TestDuration — сколько минут даётся на тест.
const TestMinutes = 40

// Quota — сколько вопросов каждого уровня попадает в один тест.
// Сумма = QuestionsPerTest. В каждой группе минимум 6 вопросов, иначе
// точность оценки уровня внутри группы была бы слишком грубой.
var Quota = map[string]int{
	"A1": 6,
	"A2": 7,
	"B1": 7,
	"B2": 7,
	"C1": 7,
	"C2": 6,
}

// weight — «вес» вопроса при подсчёте итогового индекса уровня.
// Чем сложнее вопрос, тем дороже правильный ответ.
var weight = map[string]int{
	"A1": 1, "A2": 2, "B1": 3, "B2": 4, "C1": 5, "C2": 6,
}

// thresholds — минимальный индекс (в процентах) для каждого уровня.
//
// Пороги выведены из принципа: ученик, который уверенно решает всё до уровня X
// включительно и угадывает наугад всё, что выше, должен получить ровно X.
// При квоте 6/7/7/7/7/6 максимальный вес теста равен 140, а накопленный вес
// «полного владения» по уровням составляет 6 / 20 / 41 / 69 / 104 / 140 баллов.
// С учётом того, что живой ученик ошибается примерно в 5% «своих» вопросов,
// ожидаемый индекс для уровней равен 4 / 14 / 28 / 47 / 71 / 95 процентов.
// Порог для каждого уровня стоит посередине между соседними ожиданиями —
// так случайный разброс реже перекидывает результат на соседнюю ступень.
var thresholds = []struct {
	level  string
	minPct int
}{
	{"C2", 83},
	{"C1", 59},
	{"B2", 37},
	{"B1", 21},
	{"A2", 9},
	{"A1", 0},
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

// newRand создаёт генератор, засеянный из crypto/rand, — чтобы наборы вопросов
// нельзя было предсказать по времени запроса.
func newRand() *rand.Rand {
	var b [8]byte
	if _, err := crand.Read(b[:]); err != nil {
		// crypto/rand недоступен — используем нулевой сид, тест всё равно
		// останется работоспособным.
		return rand.New(rand.NewSource(1))
	}
	return rand.New(rand.NewSource(int64(binary.LittleEndian.Uint64(b[:]))))
}

// SelectQuestions отбирает 40 вопросов: по квоте из каждого уровня сложности,
// затем перемешивает итоговый список, чтобы вопросы не шли от простых к сложным.
// Возвращает идентификаторы вопросов в том порядке, в каком их увидит ученик.
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
// Благодаря перемешиванию правильный ответ равномерно распределён по позициям
// A/B/C/D, и стратегия «всегда жать один и тот же вариант» не работает.
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
	Correct  int             // сколько верных ответов из 40
	Total    int             // сколько вопросов было задано
	Answered int             // на сколько вопросов ученик вообще ответил
	Percent  int             // индекс уровня, 0..100
	Level    string          // итоговый уровень A1..C2
	Capped   bool            // уровень был ограничен «потолком»
	ByLevel  map[string]Band // разбивка по уровням сложности
	ByTopic  map[string]Band // разбивка по разделам грамматики
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
// Подсчёт с поправкой на угадывание: правильный ответ приносит полный вес
// вопроса, неправильный отнимает треть веса (вариантов 4, вероятность
// случайного попадания 1/4), пропуск не штрафуется. При таком счёте
// математическое ожидание для того, кто отвечает наугад, равно нулю, поэтому
// «натыкать» высокий уровень случайными кликами невозможно.
func Grade(sessionID string, questionIDs []int, answers []int) Outcome {
	out := Outcome{
		Total:   len(questionIDs),
		ByLevel: map[string]Band{},
		ByTopic: map[string]Band{},
	}

	earned := 0.0
	maxScore := 0.0

	for i, id := range questionIDs {
		question, ok := ByID[id]
		if !ok {
			continue
		}
		w := float64(weight[question.Level])
		maxScore += w

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
			earned += w
			levelBand.Correct++
			topicBand.Correct++
		case answer >= 0 && answer < len(question.Options):
			out.Answered++
			earned -= w / 3 // поправка на угадывание
		}

		out.ByLevel[question.Level] = levelBand
		out.ByTopic[question.Topic] = topicBand
	}

	if maxScore <= 0 {
		out.Level = "A1"
		return out
	}
	if earned < 0 {
		earned = 0
	}
	out.Percent = int(earned / maxScore * 100)

	// ── Определение уровня ───────────────────────────────────────────────
	//
	// Основной критерий — «лестница освоенных групп»: уровень засчитывается,
	// если ученик решил не меньше 60% вопросов этой группы.
	//
	// Одна провальная группа ниже прощается, но только начиная с B2: сильный
	// ученик может споткнуться на узкой теме, и это не должно отбрасывать его
	// на две ступени. А вот заявить B1, провалив при этом основы, нельзя —
	// иначе редкая удача на средней группе давала бы завышенный уровень.
	//
	// Такой критерий гораздо устойчивее к угадыванию, чем общий процент:
	// чтобы случайно «набрать» группу из 7 вопросов, нужно угадать 5 из них,
	// а это происходит примерно в одном случае из восьмидесяти.
	const bandPass = 0.6
	const forgiveFrom = 3 // B2 и выше

	ladder := 0
	failures := 0
	for i, level := range Levels {
		band := out.ByLevel[level]
		if band.Total == 0 {
			continue
		}
		if float64(band.Correct)/float64(band.Total) >= bandPass {
			allowed := 0
			if i >= forgiveFrom {
				allowed = 1
			}
			if failures <= allowed {
				ladder = i
			}
		} else {
			failures++
			if failures > 1 {
				break
			}
		}
	}

	// Страховка: уровень не может превышать оценку по взвешенному индексу
	// больше чем на одну ступень. Отсекает случай, когда группы «пройдены»
	// удачей, а тест в целом решён плохо.
	byIndex := 0
	for _, t := range thresholds {
		if out.Percent >= t.minPct {
			byIndex = LevelIndex(t.level)
			break
		}
	}
	ceiling := byIndex + 1
	if ceiling > len(Levels)-1 {
		ceiling = len(Levels) - 1
	}
	if ladder > ceiling {
		ladder = ceiling
		out.Capped = true
	}

	out.Level = Levels[ladder]
	return out
}
