package store

import (
	"math"
	"math/rand"
	"testing"
)

// Главная проверка: сумма шансов в каждой таблице обязана быть ровно 100.
// Если сумма больше, «лишние» проценты съедают редкости в конце списка,
// и самый ценный предмет становится недостижимым. Именно так эпический
// кейс никогда не выдавал мифическую аватарку.
func TestChanceTablesSumTo100(t *testing.T) {
	tables := map[string]map[string]float64{
		"кейс common":    CaseChances["common"],
		"кейс rare":      CaseChances["rare"],
		"кейс epic":      CaseChances["epic"],
		"кейс legendary": CaseChances["legendary"],
		"кейс free":      CaseChances["free"],
		"кейс значков":   BadgeCaseChances,
	}

	for name, table := range tables {
		sum := 0.0
		for _, rarity := range RarityOrder {
			sum += table[rarity]
		}
		if math.Abs(sum-100) > 1e-9 {
			t.Errorf("%s: сумма шансов %.2f%%, должно быть ровно 100%%", name, sum)
		}
		for rarity, chance := range table {
			if chance < 0 {
				t.Errorf("%s: отрицательный шанс у %s", name, rarity)
			}
		}
		for rarity := range table {
			known := false
			for _, r := range RarityOrder {
				if r == rarity {
					known = true
				}
			}
			if !known {
				t.Errorf("%s: неизвестная редкость %q", name, rarity)
			}
		}
	}
}

// Бесплатный кейс должен раздавать ровно то, что обещано.
func TestFreeCaseChances(t *testing.T) {
	want := map[string]float64{
		"common": 70, "rare": 18, "epic": 10, "legendary": 1.9, "mythic": 0.1,
	}
	for rarity, chance := range want {
		if got := CaseChances["free"][rarity]; math.Abs(got-chance) > 1e-9 {
			t.Errorf("бесплатный кейс, %s: шанс %.2f, ожидалось %.2f", rarity, got, chance)
		}
	}
}

// У каждой достижимой редкости должен быть хотя бы один предмет,
// иначе розыгрыш молча подменит её на обычную.
func TestPoolsCoverReachableRarities(t *testing.T) {
	for caseType, table := range CaseChances {
		for _, rarity := range RarityOrder {
			if table[rarity] > 0 && len(AvatarPool[rarity]) == 0 {
				t.Errorf("кейс %s: редкость %s достижима (%.2f%%), но аватарок в ней нет",
					caseType, rarity, table[rarity])
			}
		}
	}
	for _, rarity := range RarityOrder {
		if BadgeCaseChances[rarity] > 0 && len(BadgePool[rarity]) == 0 {
			t.Errorf("кейс значков: редкость %s достижима, но значков в ней нет", rarity)
		}
	}
}

// Фактическая частота выпадения должна сходиться с таблицей.
// Проверяем на большом числе бросков с фиксированным генератором.
func TestRollRarityMatchesTable(t *testing.T) {
	const rolls = 200000
	const tolerance = 0.5 // процентных пункта

	for caseType, table := range CaseChances {
		rng := rand.New(rand.NewSource(2024))
		counts := map[string]int{}
		for i := 0; i < rolls; i++ {
			counts[rollRarity(rng, table)]++
		}
		for _, rarity := range RarityOrder {
			want := table[rarity]
			got := float64(counts[rarity]) / rolls * 100
			if math.Abs(got-want) > tolerance {
				t.Errorf("кейс %s, %s: выпадает %.2f%%, в таблице %.2f%%",
					caseType, rarity, got, want)
			}
		}
		t.Logf("кейс %-9s common %.1f%% rare %.1f%% epic %.1f%% legendary %.2f%% mythic %.2f%%",
			caseType,
			float64(counts["common"])/rolls*100, float64(counts["rare"])/rolls*100,
			float64(counts["epic"])/rolls*100, float64(counts["legendary"])/rolls*100,
			float64(counts["mythic"])/rolls*100)
	}
}

// Редкость с нулевым шансом выпадать не должна вовсе.
func TestZeroChanceNeverDrops(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	table := CaseChances["common"] // mythic: 0
	for i := 0; i < 100000; i++ {
		if rollRarity(rng, table) == "mythic" {
			t.Fatal("из обычного кейса выпала мифическая редкость с шансом 0%")
		}
	}
}

// RollCase должен возвращать предмет из пула той редкости, которую выдал,
// и корректно отмечать дубликат.
func TestRollCaseReturnsConsistentItem(t *testing.T) {
	for i := 0; i < 2000; i++ {
		item, rarity, dup := RollCase("epic", []string{"🐱"}, nil, false)

		pool := AvatarPool[rarity]
		found := false
		for _, p := range pool {
			if p == item {
				found = true
			}
		}
		if !found {
			t.Fatalf("предмет %q не принадлежит редкости %q", item, rarity)
		}
		if want := item == "🐱"; dup != want {
			t.Fatalf("предмет %q: дубликат=%v, ожидалось %v", item, dup, want)
		}
	}
}

// Значковый кейс тянет из пула значков, а не аватарок.
func TestBadgeCaseUsesBadgePool(t *testing.T) {
	for i := 0; i < 2000; i++ {
		item, rarity, _ := RollCase("badge", nil, []string{"📚"}, true)
		if rarity == "mythic" {
			t.Fatal("у значков нет мифической редкости")
		}
		found := false
		for _, p := range BadgePool[rarity] {
			if p == item {
				found = true
			}
		}
		if !found {
			t.Fatalf("значок %q не принадлежит редкости %q", item, rarity)
		}
	}
}
