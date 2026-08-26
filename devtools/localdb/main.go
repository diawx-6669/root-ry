// Команда localdb поднимает локальный PostgreSQL для разработки.
//
// Прод живёт в Supabase, но держать разработку на общей базе нельзя: любой
// эксперимент с миграцией виден всем, а без интернета проект вообще не
// запускается. Эта команда скачивает настоящий PostgreSQL (один раз, в кэш),
// поднимает его на отдельном порту и накатывает все миграции проекта.
//
// Это отдельный модуль (devtools/localdb/go.mod), чтобы зависимость на
// embedded-postgres не попадала в go.mod самого приложения.
//
// Запуск:
//
//	cd devtools/localdb && go run .
//
// Дальше в другом терминале:
//
//	DATABASE_URL="postgres://rootry:rootry@localhost:5433/rootry?sslmode=disable" go run .
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"sort"
	"syscall"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	_ "github.com/lib/pq"
)

const (
	dbPort = 5433
	dbUser = "rootry"
	dbPass = "rootry"
	dbName = "rootry"
)

// dsn — строка подключения, которую нужно передать приложению.
var dsn = fmt.Sprintf("postgres://%s:%s@localhost:%d/%s?sslmode=disable",
	dbUser, dbPass, dbPort, dbName)

func main() {
	query := flag.String("q", "", "выполнить SQL на уже поднятой базе и выйти")
	reset := flag.Bool("reset", false, "снести данные и накатить схему заново")
	flag.Parse()

	if *query != "" {
		if err := runQuery(*query); err != nil {
			log.Fatalf("запрос не выполнился: %v", err)
		}
		return
	}

	root, err := repoRoot()
	if err != nil {
		log.Fatalf("не найден корень репозитория: %v", err)
	}

	cache := dataDir(root)
	log.Println("каталог базы:", cache)
	pg := embeddedpostgres.NewDatabase(
		embeddedpostgres.DefaultConfig().
			Username(dbUser).
			Password(dbPass).
			Database(dbName).
			Port(dbPort).
			// Windows-инсталляция initdb по умолчанию берёт системную локаль
			// (Russian_Kazakhstan.1251) и вместе с ней кодировку WIN1251, в
			// которую не влезают UTF-8 символы из комментариев к миграциям.
			// Прод в Supabase живёт на UTF8 — локальная база должна совпадать.
			Locale("C").
			Encoding("UTF8").
			RuntimePath(filepath.Join(cache, "runtime")).
			DataPath(filepath.Join(cache, "data")).
			BinariesPath(filepath.Join(cache, "bin")).
			StartTimeout(3 * time.Minute),
	)

	if portInUse(dbPort) {
		log.Println("порт", dbPort, "уже занят — где-то работает другая копия базы.")
		log.Println("Останови её (Ctrl+C в том терминале) или прибей процесс:")
		log.Fatalln("  Get-Process postgres | Stop-Process -Force")
	}

	if *reset {
		// Снести данные — единственный способ применить правку в уже
		// накатанной миграции: все файлы идемпотентны и второй раз
		// изменённый CREATE TABLE не выполнится.
		data := filepath.Join(cache, "data")
		log.Println("сношу данные базы:", data)
		if err := os.RemoveAll(data); err != nil {
			log.Fatalf("не удаляется каталог данных: %v", err)
		}
	}

	log.Println("поднимаю PostgreSQL на порту", dbPort, "(первый запуск скачивает бинарники)")
	if err := pg.Start(); err != nil {
		log.Fatalf("не стартует PostgreSQL: %v", err)
	}
	defer func() {
		log.Println("останавливаю PostgreSQL")
		_ = pg.Stop()
	}()

	if err := applyMigrations(root); err != nil {
		log.Printf("миграции не накатились: %v", err)
		return
	}

	fmt.Printf("\n  База готова. Запускай приложение так:\n\n    DATABASE_URL=%q go run .\n\n  Ctrl+C — остановить базу.\n\n", dsn)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
}

// dataDir выбирает каталог для бинарников и данных PostgreSQL.
//
// Держать их рядом с проектом было бы удобнее, но initdb ломается, если в
// пути есть неASCII-символы: он записывает путь в служебные таблицы в
// системной кодировке, а база у нас в UTF8, и инициализация падает с
// «invalid byte sequence for encoding UTF8». Путь вида
// C:\Users\Админ\ркнп 2026 — ровно этот случай.
//
// Поэтому на Windows по умолчанию уходим в C:\Users\Public: он гарантированно
// ASCII и доступен на запись без прав администратора. Переопределяется
// переменной ROOTRY_PGDATA.
func dataDir(root string) string {
	if custom := os.Getenv("ROOTRY_PGDATA"); custom != "" {
		return custom
	}
	local := filepath.Join(root, "devtools", "localdb", ".pgdata")
	if isASCII(local) {
		return local
	}
	if runtime.GOOS == "windows" {
		return filepath.Join(`C:\Users\Public`, "rootry-localdb")
	}
	return filepath.Join(os.TempDir(), "rootry-localdb")
}

// isASCII сообщает, состоит ли путь только из символов, которые initdb
// переживёт при любой системной кодировке.
func isASCII(s string) bool {
	for _, r := range s {
		if r > 127 {
			return false
		}
	}
	return true
}

// repoRoot поднимается вверх от рабочего каталога, пока не увидит schema.sql.
func repoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for i := 0; i < 6; i++ {
		if _, err := os.Stat(filepath.Join(dir, "schema.sql")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("schema.sql не найден выше %s", dir)
}

// applyMigrations накатывает базовую схему, затем миграции по возрастанию имени.
//
// Порядок важен: schema.sql создаёт users, на который ссылаются все остальные
// таблицы. Все файлы идемпотентны (IF NOT EXISTS), так что повторный запуск
// на уже готовой базе ничего не ломает.
func applyMigrations(root string) error {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	files := []string{filepath.Join(root, "schema.sql")}

	legacy, _ := filepath.Glob(filepath.Join(root, "supabase_migration_*.sql"))
	sort.Strings(legacy)
	for _, f := range legacy {
		// supabase_migration_kspoя.sql (с кириллической «я») — мёртвый файл:
		// ни одна таблица оттуда в коде не используется, а сам он ещё и
		// нерабочий (ALTER TABLE badges стоит раньше CREATE TABLE badges).
		if filepath.Base(f) == "supabase_migration_kspoя.sql" {
			continue
		}
		files = append(files, f)
	}

	next, _ := filepath.Glob(filepath.Join(root, "migrations", "*.sql"))
	sort.Strings(next)
	files = append(files, next...)

	for _, f := range files {
		src, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("%s: %w", filepath.Base(f), err)
		}
		if _, err := db.Exec(string(src)); err != nil {
			return fmt.Errorf("%s: %w", filepath.Base(f), err)
		}
		log.Println("накатил", filepath.Base(f))
	}
	return nil
}

// portInUse проверяет, не занят ли порт базы.
//
// Без этой проверки сценарий выглядел так: вторая копия не стартует из-за
// занятого порта, но -reset к этому моменту уже снёс каталог данных, и
// работающий сервер остаётся с удалённой под ним базой. Чинится только
// ручным прибиванием процесса.
func portInUse(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), time.Second)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}
