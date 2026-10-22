package kspoya

import (
	"testing"

	"rootry/internal/topics"
)

// TestSectionTableCoversBank — самый ценный тест файла.
//
// Если в банк добавят вопрос с новым разделом и забудут строку в
// SectionToTree, диагностика молча проигнорирует эти вопросы: ученик
// провалит раздел, а в маршруте его не будет. Молча — худший способ
// ломаться, поэтому ловим здесь.
func TestSectionTableCoversBank(t *testing.T) {
	missing := map[string]int{}
	for _, q := range Bank {
		section, _ := splitLabel(q.Topic)
		if _, ok := SectionToTree[section]; !ok {
			missing[section]++
		}
	}
	for section, n := range missing {
		t.Errorf("раздел %q встречается в банке %d раз, но его нет в SectionToTree", section, n)
	}
}

// TestSectionTablePointsAtRealSections — обратная проверка: таблица не
// должна ссылаться на разделы, которых в дереве нет.
func TestSectionTablePointsAtRealSections(t *testing.T) {
	real := map[string]bool{}
	for _, topic := range topics.All {
		real[topic.Section] = true
	}
	for label, section := range SectionToTree {
		if !real[section] {
			t.Errorf("раздел банка %q ведёт на %q, которого нет в дереве", label, section)
		}
	}
}

// TestBankCoversTree сообщает о разделах дерева, по которым в банке нет
// ни одного вопроса: диагностика такие разделы не проверяет вообще.
//
// Тест намеренно не падает, а пишет в лог. Это дыра в содержании банка, а
// не ошибка кода; падать ему стоило бы, только когда банк объявят полным.
func TestBankCoversTree(t *testing.T) {
	covered := map[string]bool{}
	for _, q := range Bank {
		section, _ := splitLabel(q.Topic)
		if tree, ok := SectionToTree[section]; ok {
			covered[tree] = true
		}
	}
	reported := map[string]bool{}
	for _, topic := range topics.All {
		if covered[topic.Section] || reported[topic.Section] {
			continue
		}
		reported[topic.Section] = true
		t.Logf("в банке КСПОЯ нет ни одного вопроса по разделу %q", topic.Section)
	}
}

// gapFor находит разбор по разделу дерева.
func gapFor(d Diagnosis, section string) (SectionGap, bool) {
	for _, g := range d.Gaps {
		if g.Section == section {
			return g, true
		}
	}
	return SectionGap{}, false
}

func hasTopic(route []RouteTopic, id string) bool {
	for _, r := range route {
		if r.ID == id {
			return true
		}
	}
	return false
}

// TestDiagnoseMarksWeakSections — провальный раздел помечается, сильный нет.
func TestDiagnoseMarksWeakSections(t *testing.T) {
	d := Diagnose(map[string]Band{
		"Орфография · Н и НН":     {Correct: 0, Total: 3},
		"Орфография · НЕ и НИ":    {Correct: 1, Total: 3},
		"Лексикология · Синонимы": {Correct: 3, Total: 3},
		"Лексикология · Антонимы": {Correct: 2, Total: 2},
	}, nil)

	orf, ok := gapFor(d, "3. ОРФОГРАФИЯ")
	if !ok {
		t.Fatal("орфография не попала в карту пробелов")
	}
	if !orf.Weak {
		t.Errorf("орфография с 1 из 6 не помечена как провальная: accuracy=%.2f", orf.Accuracy)
	}
	if orf.Correct != 1 || orf.Total != 6 {
		t.Errorf("суммы по разделу: %d из %d, ожидалось 1 из 6", orf.Correct, orf.Total)
	}

	lex, ok := gapFor(d, "6. ЛЕКСИКОЛОГИЯ")
	if !ok {
		t.Fatal("лексикология не попала в карту пробелов")
	}
	if lex.Weak {
		t.Error("лексикология с 5 из 5 помечена как провальная")
	}

	// Слабое идёт первым: карта читается сверху вниз.
	if d.Gaps[0].Section != "3. ОРФОГРАФИЯ" {
		t.Errorf("первым в карте %q, ожидалась орфография", d.Gaps[0].Section)
	}
}

// TestRoutePrefersPreciseHits — тема, название которой прямо совпало с
// проваленным вопросом, должна попасть в маршрут обязательно: по ней есть
// доказательство пробела, а не догадка по разделу.
func TestRoutePrefersPreciseHits(t *testing.T) {
	d := Diagnose(map[string]Band{
		"Орфография · Н и НН":  {Correct: 0, Total: 2},
		"Орфография · Тире":    {Correct: 2, Total: 2},
		"Пунктуация · Запятая": {Correct: 0, Total: 4},
	}, nil)

	if !hasTopic(d.Route, "orf-n-nn") {
		t.Errorf("тема «Н и НН» не попала в маршрут: %+v", d.Route)
	}
	for _, r := range d.Route {
		if r.ID == "orf-n-nn" {
			if !r.Precise {
				t.Error("«Н и НН» помечена как неточное попадание")
			}
			if r.Reason == "" {
				t.Error("у темы в маршруте нет объяснения, почему она здесь")
			}
		}
	}
}

// TestRouteSkipsKnownTopics — освоенные темы в маршрут не идут. Если тема
// освоена, а раздел провален, дело в других темах раздела.
func TestRouteSkipsKnownTopics(t *testing.T) {
	byTopic := map[string]Band{
		"Орфография · Н и НН": {Correct: 0, Total: 3},
	}

	before := Diagnose(byTopic, nil)
	if !hasTopic(before.Route, "orf-n-nn") {
		t.Fatal("тема не попала в маршрут даже без списка освоенных")
	}

	after := Diagnose(byTopic, map[string]bool{"orf-n-nn": true})
	if hasTopic(after.Route, "orf-n-nn") {
		t.Error("освоенная тема всё равно попала в маршрут")
	}
}

// TestRouteIsCappedAndOrdered — маршрут не длиннее лимита и идёт в порядке
// дерева: орфографию не объяснить, не разобравшись со звуками.
func TestRouteIsCappedAndOrdered(t *testing.T) {
	// Провалено вообще всё.
	byTopic := map[string]Band{}
	for label := range SectionToTree {
		byTopic[label+" · разное"] = Band{Correct: 0, Total: 3}
	}

	d := Diagnose(byTopic, nil)

	if len(d.Route) > RouteLimit {
		t.Errorf("в маршруте %d тем, лимит %d", len(d.Route), RouteLimit)
	}
	if len(d.Route) == 0 {
		t.Fatal("маршрут пуст, хотя провалены все разделы")
	}
	for i := 1; i < len(d.Route); i++ {
		if order[d.Route[i-1].ID] > order[d.Route[i].ID] {
			t.Errorf("маршрут идёт не в порядке дерева: %q стоит перед %q",
				d.Route[i-1].Title, d.Route[i].Title)
			break
		}
	}
}

// TestDiagnoseHandlesEmptyInput — пустой результат не должен ронять разбор
// и не должен выдумывать маршрут.
func TestDiagnoseHandlesEmptyInput(t *testing.T) {
	d := Diagnose(nil, nil)
	if len(d.Gaps) != 0 {
		t.Errorf("на пустом входе получено %d пробелов", len(d.Gaps))
	}
	if len(d.Route) != 0 {
		t.Errorf("на пустом входе построен маршрут из %d тем", len(d.Route))
	}

	d = Diagnose(map[string]Band{"Орфография · Н и НН": {Correct: 0, Total: 0}}, nil)
	if len(d.Gaps) != 0 {
		t.Error("раздел без заданных вопросов попал в карту пробелов")
	}
}

// TestPerfectScoreGivesNoRoute — ученику, решившему всё, учить нечего.
func TestPerfectScoreGivesNoRoute(t *testing.T) {
	d := Diagnose(map[string]Band{
		"Орфография · Н и НН":     {Correct: 3, Total: 3},
		"Лексикология · Синонимы": {Correct: 2, Total: 2},
	}, nil)

	if len(d.Route) != 0 {
		t.Errorf("при полном результате построен маршрут: %+v", d.Route)
	}
	for _, g := range d.Gaps {
		if g.Weak {
			t.Errorf("раздел %q помечен провальным при 100%%", g.Section)
		}
	}
}

// TestSingleQuestionSectionIsNotWeak — раздел с единственным вопросом не
// должен объявляться проваленным. Иначе одна невнимательность делает
// «Словообразование: 0 из 1» первой строкой карты и тащит за собой
// весь маршрут.
func TestSingleQuestionSectionIsNotWeak(t *testing.T) {
	d := Diagnose(map[string]Band{
		"Словообразование · Способы": {Correct: 0, Total: 1},
		"Орфография · Н и НН":        {Correct: 1, Total: 6},
	}, nil)

	slv, ok := gapFor(d, "5. СЛОВООБРАЗОВАНИЕ")
	if !ok {
		t.Fatal("раздел с одним вопросом пропал из карты — он должен быть виден")
	}
	if slv.Weak {
		t.Error("раздел с единственным вопросом помечен проваленным")
	}

	orf, _ := gapFor(d, "3. ОРФОГРАФИЯ")
	if !orf.Weak {
		t.Error("орфография с 1 из 6 не помечена проваленной")
	}

	// Провальный раздел идёт первым, несмотря на более высокую долю.
	if d.Gaps[0].Section != "3. ОРФОГРАФИЯ" {
		t.Errorf("первым в карте %q, ожидалась орфография", d.Gaps[0].Section)
	}
	// И маршрут строится по нему, а не по случайному одиночному вопросу.
	if len(d.Route) == 0 || d.Route[0].ID != "orf-n-nn" {
		t.Errorf("маршрут начинается не с «Н и НН»: %+v", d.Route)
	}
}
