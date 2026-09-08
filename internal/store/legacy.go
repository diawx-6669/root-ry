package store

// Перевод старых эмодзи-идентификаторов на новые.
//
// До перехода на SVG аватарка и значок хранились в базе как эмодзи:
// "🐱", "🏆". Картинку рисовал шрифт операционной системы, поэтому одна
// и та же награда выглядела по-разному в Windows, на телефоне и на
// проекторе, а кое-где вырождалась в пустой квадрат. Теперь и то и
// другое — короткий идентификатор ("cat", "trophy"), которому
// соответствует файл в static/img/.
//
// У аккаунтов, заведённых раньше, в базе остались эмодзи. Переписывать
// их миграцией было бы неудобно: пользователи заходят по одному, а
// колонки — JSONB. Вместо этого значения приводятся к новому виду при
// чтении пользователя и сохраняются в новом виде при первой же записи.
// Незнакомое значение остаётся как есть: чужой идентификатор лучше
// потерянной награды.

// legacyAvatars — эмодзи аватарок до перехода на SVG.
var legacyAvatars = map[string]string{
	"🐱": "cat", "🐶": "dog", "🦊": "fox", "🐼": "panda", "🐨": "koala",
	"🐻": "bear", "🐸": "frog", "🦁": "lion", "🐯": "tiger",
	"🦄": "unicorn", "🐉": "dragon", "🦋": "butterfly", "🦚": "peacock",
	"🦜": "parrot", "🦩": "flamingo", "🐬": "dolphin",
	"🧙": "wizard", "🧛": "vampire", "🦸": "hero", "🧝": "elf", "🧜": "mermaid",
	"👑": "crown", "🌟": "star", "💫": "comet",
	"🌈": "rainbow",
}

// legacyBadges — эмодзи значков и знаков уровня КСПОЯ.
var legacyBadges = map[string]string{
	"📚": "book", "✏️": "pencil", "✏": "pencil", "📝": "notepad", "🎒": "backpack",
	"⭐": "star", "🔥": "fire", "💡": "bulb",
	"🏆": "trophy", "💎": "diamond", "👑": "crown", "🌳": "tree",
	"🔰": "rank_a1", "📗": "rank_a2", "📘": "rank_b1",
	"🏅": "rank_b2", "🔮": "rank_c1", "🦉": "rank_c2",
	// До миграции 006 знаком C2 была радуга. В списке аватарок «🌈» —
	// мифическая радуга, а в списке значков — именно уровень: колонки
	// разные, поэтому и таблицы перевода разные.
	"🌈": "rank_c2",
	// «🎓» выдавался демо-аккаунту при первом запуске и своей пары в
	// новом каталоге не имеет — ближе всего значок за пройденное дерево.
	"🎓": "tree",
}

// NormalizeAvatar приводит идентификатор аватарки к новому виду.
func NormalizeAvatar(v string) string {
	if id, ok := legacyAvatars[v]; ok {
		return id
	}
	return v
}

// NormalizeBadge приводит идентификатор значка к новому виду.
func NormalizeBadge(v string) string {
	if id, ok := legacyBadges[v]; ok {
		return id
	}
	return v
}

// normalizeList переводит список и убирает дубли, которые могли
// возникнуть при слиянии («⭐» и «star» рядом дали бы две одинаковые
// звезды в профиле).
func normalizeList(items []string, conv func(string) string) []string {
	if items == nil {
		return nil
	}
	seen := make(map[string]bool, len(items))
	out := make([]string, 0, len(items))
	for _, it := range items {
		id := conv(it)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

// NormalizeAvatars — список аватарок в новом виде.
func NormalizeAvatars(items []string) []string { return normalizeList(items, NormalizeAvatar) }

// NormalizeBadges — список значков в новом виде.
func NormalizeBadges(items []string) []string { return normalizeList(items, NormalizeBadge) }
