package main

import (
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
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "rootry_data.json"
	}
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "rootry_dev_secret_change_in_prod"
		log.Println("WARNING: JWT_SECRET not set, using default dev secret")
	}
	middleware.SetJWTSecret(jwtSecret)

	s := store.New(dbPath)
	h := handlers.New(s)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/register", h.Register)
	mux.HandleFunc("/api/login", h.Login)
	mux.HandleFunc("/api/leaderboard", h.Leaderboard)
	mux.HandleFunc("/api/me", middleware.AuthMiddleware(h.Me))
	mux.HandleFunc("/api/promo", middleware.AuthMiddleware(h.RedeemPromo))
	mux.HandleFunc("/api/game/submit", middleware.AuthMiddleware(h.GameSubmit))
	mux.HandleFunc("/api/topic/complete", middleware.AuthMiddleware(h.TopicComplete))
	mux.HandleFunc("/api/kspoya/start", middleware.AuthMiddleware(h.KspoyaStart))
	mux.HandleFunc("/api/kspoya/submit", middleware.AuthMiddleware(h.KspoyaSubmit))
	mux.HandleFunc("/api/kspoya/abort", middleware.AuthMiddleware(h.KspoyaAbort))
	mux.HandleFunc("/api/kspoya/status", middleware.AuthMiddleware(h.KspoyaStatus))
	mux.HandleFunc("/api/kspoya/history", middleware.AuthMiddleware(h.KspoyaHistory))
	mux.HandleFunc("/api/kspoya/attempt", middleware.AuthMiddleware(h.KspoyaAttempt))
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
