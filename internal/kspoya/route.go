package kspoya

import (
	"sort"
	"strings"

	"rootry/internal/topics"
)

// Маршрут после диагностики.
//
// Раньше КСПОЯ заканчивался уровнем: «у тебя B1» — и всё. Уровень без
// назначения бесполезен, это градусник без лечения. Здесь результат теста
// превращается в список тем, с которых ученику стоит начать.
//
// Метки вопросов в банке устроены как «Раздел · Подтема», например
// «Орфография · Н и НН». Отсюда две дороги к теме дерева:
//
//	точная — подтема совпадает с названием темы («Н и НН» → orf-n-nn);
//	грубая — по разделу, когда точного совпадения нет.
//
// Точных совпадений сейчас 21 из 80 подтем. Остальное честнее сводить к
// разделу, чем угадывать: сорок вопросов физически не могут надёжно
// продиагностировать 74 отдельные темы, и делать вид, что могут, —
// значит обманывать ученика точностью, которой нет.

// SectionToTree — соответствие разделов банка разделам дерева.
//
// Задано таблицей, а не выводится из строк: названия близки, но не
// совпадают («Графика» против «2. ГРАФИКА И АЛФАВИТ»), и любое умное
// сопоставление здесь сломается на первой же правке названия.
var SectionToTree = map[string]string{
	"Фонетика":         "1. ФОНЕТИКА",
	"Графика":          "2. ГРАФИКА И АЛФАВИТ",
	"Орфография":       "3. ОРФОГРАФИЯ",
	"Морфемика":        "4. МОРФЕМИКА",
	"Словообразование": "5. СЛОВООБРАЗОВАНИЕ",
	"Лексикология":     "6. ЛЕКСИКОЛОГИЯ",
	"Морфология":       "7. МОРФОЛОГИЯ",
	"Синтаксис":        "8. СИНТАКСИС",
	"Пунктуация":       "9. ПУНКТУАЦИЯ",
	"Стилистика":       "10. СТИЛИСТИКА",
	"Культура речи":    "11. КУЛЬТУРА РЕЧИ",
	"Текстоведение":    "12. ТЕКСТОВЕДЕНИЕ",
}

// WeakAccuracy — доля верных ответов, ниже которой раздел считается
// провальным.
//
// Две трети, а не половина: вариантов ответа шесть, случайным тыком
// набирается около 17%, и половина — это ещё не «в целом разобрался».
const WeakAccuracy = 0.67

// MinWeakQuestions — сколько вопросов по разделу нужно, чтобы называть его
// проваленным.
//
// В тесте сорок вопросов на дюжину разделов, и по редким разделам их
// достаётся один-два. Один неверный ответ — это не «раздел провален», это
// шум: ученик мог не дочитать вопрос. Без этого порога раздел с
// единственным вопросом стабильно оказывался первым в карте пробелов и
// тянул за собой весь маршрут.
//
// Раздел с одним вопросом всё равно показывается в карте — просто не
// помечается провальным и не определяет, что учить.
const MinWeakQuestions = 2

// RouteLimit — сколько тем попадает в маршрут.
//
// Семь: список, который реально начинают проходить. Двадцать тем в
// маршруте читаются как приговор и закрываются, не начавшись.
const RouteLimit = 7

// SectionGap — как ученик справился с одним разделом.
type SectionGap struct {
	Section  string  `json:"section"`  // раздел дерева
	Label    string  `json:"label"`    // название раздела в банке
	Correct  int     `json:"correct"`  // верных ответов
	Total    int     `json:"total"`    // вопросов по разделу
	Accuracy float64 `json:"accuracy"` // 0..1
	Weak     bool    `json:"weak"`     // раздел провален
}

// RouteTopic — тема в персональном маршруте.
type RouteTopic struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Section string `json:"section"`
	XP      int    `json:"xp"`
	// Reason объясняет, почему тема здесь. Без объяснения маршрут выглядит
	// как случайный список, и ему не верят.
	Reason string `json:"reason"`
	// Precise = тема опознана точно, по названию подтемы в вопросе.
	// Иначе она попала в маршрут как часть провального раздела.
	Precise bool `json:"precise"`
}

// Diagnosis — что делать после теста.
type Diagnosis struct {
	Gaps  []SectionGap `json:"gaps"`
	Route []RouteTopic `json:"route"`
}

// titleIndex — названия тем дерева в нижнем регистре.
var titleIndex = func() map[string]topics.Topic {
	m := make(map[string]topics.Topic, len(topics.All))
	for _, t := range topics.All {
		m[normalizeTitle(t.Title)] = t
	}
	return m
}()

// order — позиция темы в реестре. Реестр идёт в порядке разделов дерева,
// от фонетики к текстоведению, и этот порядок и есть учебная
// последовательность: орфографию не объяснить, не разобравшись со звуками.
var order = func() map[string]int {
	m := make(map[string]int, len(topics.All))
	for i, t := range topics.All {
		m[t.ID] = i
	}
	return m
}()

// sectionOrder — позиция раздела в дереве, по первой теме раздела.
// Нужен отдельно от order: там ключ — идентификатор темы, и передавать
// туда название раздела значило бы всегда получать ноль.
var sectionOrder = func() map[string]int {
	m := map[string]int{}
	for i, t := range topics.All {
		if _, seen := m[t.Section]; !seen {
			m[t.Section] = i
		}
	}
	return m
}()

func normalizeTitle(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// splitLabel разбирает метку вопроса на раздел и подтему.
func splitLabel(label string) (section, tail string) {
	parts := strings.SplitN(label, " · ", 2)
	section = strings.TrimSpace(parts[0])
	if len(parts) > 1 {
		tail = strings.TrimSpace(parts[1])
	}
	return section, tail
}

// Diagnose строит карту пробелов и маршрут по результатам теста.
//
// known — темы, которые ученик уже освоил: они в маршрут не попадают,
// сколько бы вопросов по ним ни было провалено. Если тема освоена, а
// вопросы по разделу провалены, дело в других темах раздела.
func Diagnose(byTopic map[string]Band, known map[string]bool) Diagnosis {
	sections := map[string]*SectionGap{}
	// preciseMiss — темы, опознанные точно и при этом проваленные.
	preciseMiss := map[string]string{}

	for label, band := range byTopic {
		if band.Total == 0 {
			continue
		}
		sectionLabel, tail := splitLabel(label)
		treeSection, mapped := SectionToTree[sectionLabel]
		if !mapped {
			// Раздела нет в дереве — молча пропускаем, а не роняем разбор.
			// Тест на соответствие таблицы банку ловит это раньше прода.
			continue
		}

		gap := sections[treeSection]
		if gap == nil {
			gap = &SectionGap{Section: treeSection, Label: sectionLabel}
			sections[treeSection] = gap
		}
		gap.Correct += band.Correct
		gap.Total += band.Total

		if band.Correct < band.Total {
			if t, ok := titleIndex[normalizeTitle(tail)]; ok {
				preciseMiss[t.ID] = tail
			}
		}
	}

	gaps := make([]SectionGap, 0, len(sections))
	for _, g := range sections {
		g.Accuracy = float64(g.Correct) / float64(g.Total)
		g.Weak = g.Accuracy < WeakAccuracy && g.Total >= MinWeakQuestions
		gaps = append(gaps, *g)
	}
	// Слабое — первым: карта пробелов читается сверху вниз.
	//
	// При равной доле выше идёт раздел, по которому вопросов было больше:
	// ноль из пяти — это вывод, ноль из одного — совпадение.
	sort.Slice(gaps, func(i, j int) bool {
		if gaps[i].Weak != gaps[j].Weak {
			return gaps[i].Weak
		}
		if gaps[i].Accuracy != gaps[j].Accuracy {
			return gaps[i].Accuracy < gaps[j].Accuracy
		}
		if gaps[i].Total != gaps[j].Total {
			return gaps[i].Total > gaps[j].Total
		}
		return sectionOrder[gaps[i].Section] < sectionOrder[gaps[j].Section]
	})

	return Diagnosis{Gaps: gaps, Route: buildRoute(gaps, preciseMiss, known)}
}

func buildRoute(gaps []SectionGap, preciseMiss map[string]string, known map[string]bool) []RouteTopic {
	picked := map[string]bool{}
	route := []RouteTopic{}

	add := func(t topics.Topic, reason string, precise bool) bool {
		if picked[t.ID] || known[t.ID] {
			return false
		}
		picked[t.ID] = true
		route = append(route, RouteTopic{
			ID: t.ID, Title: t.Title, Section: t.Section,
			XP: t.XP, Reason: reason, Precise: precise,
		})
		return len(route) >= RouteLimit
	}

	// Сначала темы, опознанные точно: по ним есть прямое доказательство
	// пробела, а не догадка по разделу.
	precise := make([]topics.Topic, 0, len(preciseMiss))
	for id := range preciseMiss {
		if t, ok := topics.Get(id); ok {
			precise = append(precise, t)
		}
	}
	sort.Slice(precise, func(i, j int) bool { return order[precise[i].ID] < order[precise[j].ID] })
	for _, t := range precise {
		if add(t, "ошибка в вопросе по этой теме", true) {
			return route
		}
	}

	// Затем — темы из проваленных разделов, в порядке дерева.
	for _, gap := range gaps {
		if !gap.Weak {
			continue
		}
		reason := "раздел «" + gap.Label + "» просел: " +
			itoa(gap.Correct) + " из " + itoa(gap.Total)
		for _, t := range topics.All {
			if t.Section != gap.Section {
				continue
			}
			if add(t, reason, false) {
				return route
			}
		}
	}
	return route
}

// itoa — маленькая замена strconv ради одной строки в причине.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
