package store

import (
	crand "crypto/rand"
	"encoding/binary"
	"log"
	"math/rand"
)

// RarityOrder — порядок редкостей от частой к редкой.
var RarityOrder = []string{"common", "rare", "epic", "legendary", "mythic"}

// CaseChances — шансы выпадения по типам кейсов, в процентах.
//
// Сумма каждой таблицы обязана равняться 100: проверка расчёта идёт
// нарастающим итогом, и «лишние» проценты просто съедают редкости в конце
// списка. Раньше у эпического кейса сумма была 105 — из-за этого легендарка
// выпадала в 5% случаев вместо обещанных 9%, а мифическая аватарка не
// выпадала вообще. Лишние 5% сняты с обычной редкости, чтобы обещания
// по ценным предметам остались нетронутыми.
var CaseChances = map[string]map[string]float64{
	"common":    {"common": 70, "rare": 18, "epic": 10, "legendary": 2, "mythic": 0},
	"rare":      {"common": 40, "rare": 42.5, "epic": 12.5, "legendary": 4.5, "mythic": 0.5},
	"epic":      {"common": 15, "rare": 52.5, "epic": 22.5, "legendary": 9, "mythic": 1},
	"legendary": {"common": 0, "rare": 20, "epic": 40, "legendary": 35, "mythic": 5},
	// Бесплатный кейс, доступный раз в сутки.
	"free": {"common": 70, "rare": 18, "epic": 10, "legendary": 1.9, "mythic": 0.1},
}

// BadgeCaseChances — шансы значкового кейса. Мифических значков нет.
var BadgeCaseChances = map[string]float64{
	"common": 40, "rare": 30, "epic": 20, "legendary": 10, "mythic": 0,
}

// AvatarPool — аватарки по редкости.
var AvatarPool = map[string][]string{
	"common":    {"🐱", "🐶", "🦊", "🐼", "🐨", "🦁", "🐯", "🐻", "🐸"},
	"rare":      {"🦄", "🐉", "🦋", "🦚", "🦜", "🦩", "🐬"},
	"epic":      {"🧙", "🧛", "🧜", "🧝", "🦸"},
	"legendary": {"👑", "🌟", "💫"},
	"mythic":    {"🌈"},
}

// BadgePool — значки по редкости.
var BadgePool = map[string][]string{
	"common":    {"📚", "✏️", "📝", "🎒"},
	"rare":      {"⭐", "🔥", "💡"},
	"epic":      {"🏆", "💎"},
	"legendary": {"👑"},
}

// FreeCaseAvailable сообщает, доступен ли бесплатный кейс сегодня.
func (s *Store) FreeCaseAvailable(username string) bool {
	var available bool
	err := s.db.QueryRow(`
		SELECT last_free_case IS DISTINCT FROM (now() AT TIME ZONE $2)::date
		FROM users WHERE username = $1`, username, localZone).Scan(&available)
	if err != nil {
		log.Printf("FreeCaseAvailable(%s): %v", username, err)
		return false
	}
	return available
}

// ClaimFreeCase отмечает бесплатный кейс за сегодня использованным.
// Возвращает false, если сегодня его уже открывали.
//
// Отметка ставится одним UPDATE с условием: два одновременных запроса
// не смогут открыть кейс дважды за день.
func (s *Store) ClaimFreeCase(username string) bool {
	res, err := s.db.Exec(`
		UPDATE users SET last_free_case = (now() AT TIME ZONE $2)::date
		WHERE username = $1
		  AND last_free_case IS DISTINCT FROM (now() AT TIME ZONE $2)::date`,
		username, localZone)
	if err != nil {
		log.Printf("ClaimFreeCase(%s): %v", username, err)
		return false
	}
	n, err := res.RowsAffected()
	return err == nil && n == 1
}

// caseRand — источник случайности для розыгрышей.
//
// Раньше и редкость, и конкретный предмет выводились из одного значения
// time.Now().UnixNano(): результаты были связаны между собой и в принципе
// предсказуемы по времени запроса. Теперь генератор засевается из
// crypto/rand, а редкость и предмет тянутся отдельными бросками.
func caseRand() *rand.Rand {
	var b [8]byte
	if _, err := crand.Read(b[:]); err != nil {
		return rand.New(rand.NewSource(1))
	}
	return rand.New(rand.NewSource(int64(binary.LittleEndian.Uint64(b[:]))))
}

// rollRarity выбирает редкость по таблице шансов.
func rollRarity(rng *rand.Rand, chances map[string]float64) string {
	roll := rng.Float64() * 100
	cum := 0.0
	for _, rarity := range RarityOrder {
		cum += chances[rarity]
		if roll < cum {
			return rarity
		}
	}
	// Сюда попадаем только если сумма таблицы меньше 100.
	return RarityOrder[0]
}

// RollCase разыгрывает предмет из кейса.
// Возвращает предмет, его редкость и признак дубликата.
func RollCase(caseType string, userAvatars, userBadges []string, isBadgeCase bool) (string, string, bool) {
	rng := caseRand()

	if isBadgeCase {
		rarity := rollRarity(rng, BadgeCaseChances)
		pool := BadgePool[rarity]
		if len(pool) == 0 {
			pool = BadgePool["common"]
			rarity = "common"
		}
		item := pool[rng.Intn(len(pool))]
		return item, rarity, hasBadge(userBadges, item)
	}

	chances := CaseChances[caseType]
	if chances == nil {
		chances = CaseChances["common"]
	}
	rarity := rollRarity(rng, chances)
	pool := AvatarPool[rarity]
	if len(pool) == 0 {
		pool = AvatarPool["common"]
		rarity = "common"
	}
	item := pool[rng.Intn(len(pool))]
	return item, rarity, hasAvatar(userAvatars, item)
}
