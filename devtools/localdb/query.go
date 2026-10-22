package main

import (
	"database/sql"
	"fmt"
	"strings"
)

// runQuery выполняет один SQL-запрос на уже поднятой локальной базе и печатает
// результат таблицей.
//
// Сборка embedded-postgres приезжает без psql, а заглядывать в базу во время
// разработки нужно постоянно: проверить, что миграция накатилась, посмотреть,
// что записал /api/attempt. Ради этого держать отдельный SQL-клиент незачем.
//
//	cd devtools/localdb && go run . -q "SELECT * FROM topic_mastery"
func runQuery(query string) error {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	rows, err := db.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return err
	}

	// Собираем всё в память: это инструмент разработчика, а не выгрузка.
	var table [][]string
	for rows.Next() {
		cells := make([]any, len(cols))
		for i := range cells {
			cells[i] = new(sql.NullString)
		}
		if err := rows.Scan(cells...); err != nil {
			return err
		}
		line := make([]string, len(cols))
		for i, c := range cells {
			if v := c.(*sql.NullString); v.Valid {
				line[i] = v.String
			} else {
				line[i] = "NULL"
			}
		}
		table = append(table, line)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	width := make([]int, len(cols))
	for i, c := range cols {
		width[i] = len(c)
	}
	for _, line := range table {
		for i, cell := range line {
			if n := len([]rune(cell)); n > width[i] {
				width[i] = n
			}
		}
	}

	printRow := func(cells []string) {
		parts := make([]string, len(cells))
		for i, cell := range cells {
			parts[i] = cell + strings.Repeat(" ", width[i]-len([]rune(cell)))
		}
		fmt.Println(" " + strings.Join(parts, " | "))
	}

	printRow(cols)
	seps := make([]string, len(cols))
	for i := range seps {
		seps[i] = strings.Repeat("-", width[i])
	}
	printRow(seps)
	for _, line := range table {
		printRow(line)
	}
	fmt.Printf("\n(строк: %d)\n", len(table))
	return nil
}
