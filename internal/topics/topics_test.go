package topics

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// indexEntry вытаскивает из static/lessons/index.js идентификатор урока
// и его награду в XP.
var indexEntry = regexp.MustCompile(`\{\s*id:'([a-z0-9-]+)'.*?xp:(\d+)\s*\}`)

// TestRegistryMatchesLessonIndex ловит главный способ сломать уроки:
// добавить тему в один список и забыть про другой.
//
// Реестр в Go — источник правды для награды и для проверки в
// /api/topic/complete. Индекс в JS решает, какие узлы дерева кликабельны.
// Разъедутся — ученик откроет урок, а сервер ответит «Неизвестная тема».
func TestRegistryMatchesLessonIndex(t *testing.T) {
	path := filepath.Join("..", "..", "static", "lessons", "index.js")
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("не читается %s: %v", path, err)
	}

	matches := indexEntry.FindAllStringSubmatch(string(src), -1)
	if len(matches) == 0 {
		t.Fatal("в index.js не найдено ни одной записи — проверь формат файла")
	}

	jsXP := make(map[string]string, len(matches))
	for _, m := range matches {
		if _, dup := jsXP[m[1]]; dup {
			t.Errorf("id %q встречается в index.js дважды", m[1])
		}
		jsXP[m[1]] = m[2]
	}

	if len(jsXP) != len(All) {
		t.Errorf("тем в реестре %d, уроков в index.js %d", len(All), len(jsXP))
	}

	for _, topic := range All {
		xp, ok := jsXP[topic.ID]
		if !ok {
			t.Errorf("тема %q есть в реестре, но её нет в index.js", topic.ID)
			continue
		}
		if want := itoa(topic.XP); xp != want {
			t.Errorf("тема %q: XP в реестре %s, в index.js %s", topic.ID, want, xp)
		}
		delete(jsXP, topic.ID)
	}
	for id := range jsXP {
		t.Errorf("урок %q есть в index.js, но его нет в реестре тем", id)
	}
}

// TestGetKnownAndUnknown проверяет саму проверку идентификатора: именно она
// закрывает бесконечный фарм XP через /api/topic/complete.
func TestGetKnownAndUnknown(t *testing.T) {
	if _, ok := Get("fon-zvuki"); !ok {
		t.Error("известная тема fon-zvuki не найдена в реестре")
	}
	for _, bad := range []string{"", "случайная строка", "FON-ZVUKI", "fon-zvuki "} {
		if _, ok := Get(bad); ok {
			t.Errorf("реестр принял неизвестную тему %q", bad)
		}
	}
}

// TestRewardsArePositive: тема без награды бессмысленна, а слишком щедрая
// ломает баланс относительно КСПОЯ (2000 XP за высший уровень).
func TestRewardsArePositive(t *testing.T) {
	seen := make(map[string]bool, len(All))
	for _, topic := range All {
		if seen[topic.ID] {
			t.Errorf("дубликат id в реестре: %q", topic.ID)
		}
		seen[topic.ID] = true

		if topic.XP <= 0 || topic.Coins <= 0 {
			t.Errorf("тема %q: награда должна быть положительной (xp=%d, coins=%d)",
				topic.ID, topic.XP, topic.Coins)
		}
		if topic.XP > 200 {
			t.Errorf("тема %q: %d XP — слишком много для одного урока", topic.ID, topic.XP)
		}
		if topic.Title == "" || topic.Section == "" {
			t.Errorf("тема %q: пустое название или раздел", topic.ID)
		}
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [12]byte
	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[pos:])
}
