// Package timeutil держит единственное определение «сегодня» для всего проекта.
//
// Раньше день считался в двух местах по-разному: серия входов — по Asia/Almaty
// прямо в SQL, а дейлики и ежедневный бонус — по UTC сервера. Из-за этого
// квесты обнулялись в 05:00 по местному времени, а серия — в полночь.
package timeutil

import (
	"time"

	// tzdata вшивает базу часовых поясов в бинарник: в контейнере Railway
	// системной базы может не быть, и LoadLocation вернул бы ошибку.
	_ "time/tzdata"
)

// Zone — часовой пояс учеников. Тот же литерал используется в SQL-запросах.
const Zone = "Asia/Almaty"

var loc = mustLoad()

func mustLoad() *time.Location {
	l, err := time.LoadLocation(Zone)
	if err != nil {
		// Крайний случай: работаем по фиксированному смещению +05:00.
		return time.FixedZone("ALMT", 5*60*60)
	}
	return l
}

// Location возвращает часовой пояс проекта.
func Location() *time.Location { return loc }

// Now — текущее время в местном поясе.
func Now() time.Time { return time.Now().In(loc) }

// Today — сегодняшняя дата в формате YYYY-MM-DD.
func Today() string { return Now().Format("2006-01-02") }
