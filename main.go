package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os"

	"rootry/internal/handlers"
	"rootry/internal/middleware"
	"rootry/internal/store"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		// Раньше подставлялся секрет из исходников. Зная его, любой мог
		// подписать себе токен администратора. Теперь без переменной
		// окружения секрет случайный: после перезапуска все просто войдут
		// заново, но подделать токен нельзя.
		var b [32]byte
		if _, err := rand.Read(b[:]); err != nil {
			log.Fatalf("JWT_SECRET: %v", err)
		}
		jwtSecret = hex.EncodeToString(b[:])
		log.Println("WARNING: JWT_SECRET not set, using a random secret (sessions reset on restart)")
	}
	middleware.SetJWTSecret(jwtSecret)

	s := store.New()
	middleware.SetAdminChecker(s.IsAdmin)
	h := handlers.New(s)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/register", h.Register)
	mux.HandleFunc("/api/login", h.Login)
	// Рейтинг закрыт авторизацией: раньше он был публичным и отдавал логины,
	// ники, баланс и серии вообще всех зарегистрированных учеников.
	mux.HandleFunc("/api/leaderboard", middleware.AuthMiddleware(h.Leaderboard))
	mux.HandleFunc("/api/me", middleware.AuthMiddleware(h.Me))
	mux.HandleFunc("/api/profile/favorites", middleware.AuthMiddleware(h.UpdateFavorites))
	mux.HandleFunc("/api/topics", middleware.AuthMiddleware(h.Topics))
	mux.HandleFunc("/api/promo", middleware.AuthMiddleware(h.RedeemPromo))
	mux.HandleFunc("/api/game/submit", middleware.AuthMiddleware(h.GameSubmit))
	mux.HandleFunc("/api/topic/complete", middleware.AuthMiddleware(h.TopicComplete))
	mux.HandleFunc("/api/kspoya/start", middleware.AuthMiddleware(h.KspoyaStart))
	mux.HandleFunc("/api/kspoya/submit", middleware.AuthMiddleware(h.KspoyaSubmit))
	mux.HandleFunc("/api/kspoya/abort", middleware.AuthMiddleware(h.KspoyaAbort))
	mux.HandleFunc("/api/kspoya/status", middleware.AuthMiddleware(h.KspoyaStatus))
	mux.HandleFunc("/api/kspoya/history", middleware.AuthMiddleware(h.KspoyaHistory))
	mux.HandleFunc("/api/kspoya/attempt", middleware.AuthMiddleware(h.KspoyaAttempt))
	mux.HandleFunc("/api/kspoya/leaderboard", middleware.AuthMiddleware(h.KspoyaLeaderboard))
	mux.HandleFunc("/api/profile/avatar", middleware.AuthMiddleware(h.UpdateAvatar))
	mux.HandleFunc("/api/case/open", middleware.AuthMiddleware(h.CaseOpen))
	mux.HandleFunc("/api/profile/nickname", middleware.AuthMiddleware(h.UpdateNickname))
	mux.HandleFunc("/api/daily/claim", middleware.AuthMiddleware(h.DailyClaim))
	mux.HandleFunc("/api/admin/users", middleware.AdminMiddleware(h.AdminUsers))
	mux.HandleFunc("/api/admin/stats", middleware.AdminMiddleware(h.AdminStats))
	mux.Handle("/", http.FileServer(http.Dir("./static")))

	handler := middleware.CORS(mux)
	fmt.Printf("\n╔══════════════════════════════════════╗\n")
	fmt.Printf("║  🎓 RootRy запущен на порту %s      ║\n", port)
	fmt.Printf("║  Demo: demo / demo123                ║\n")
	fmt.Printf("╚══════════════════════════════════════╝\n\n")
	log.Fatal(http.ListenAndServe(":"+port, handler))
}
