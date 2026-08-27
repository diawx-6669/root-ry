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
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "rootry_dev_secret_change_in_prod"
		log.Println("WARNING: JWT_SECRET not set, using default dev secret")
	}
	middleware.SetJWTSecret(jwtSecret)

	s := store.New()
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
	// Модель знаний: журнал ответов, состояние тем, план повторения,
	// тетрадь ошибок.
	mux.HandleFunc("/api/attempt", middleware.AuthMiddleware(h.Attempt))
	mux.HandleFunc("/api/progress", middleware.AuthMiddleware(h.Progress))
	mux.HandleFunc("/api/review/plan", middleware.AuthMiddleware(h.ReviewPlan))
	mux.HandleFunc("/api/mistakes", middleware.AuthMiddleware(h.Mistakes))
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
	// Классы и кабинет учителя.
	mux.HandleFunc("/api/class/create", middleware.AuthMiddleware(h.ClassCreate))
	mux.HandleFunc("/api/class/join", middleware.AuthMiddleware(h.ClassJoin))
	mux.HandleFunc("/api/class/my", middleware.AuthMiddleware(h.ClassMy))
	mux.HandleFunc("/api/class/heatmap", middleware.AuthMiddleware(h.ClassHeatmap))
	mux.HandleFunc("/api/class/export", middleware.AuthMiddleware(h.ClassExport))
	mux.HandleFunc("/api/admin/teacher", middleware.AdminMiddleware(h.AdminSetTeacher))
	// Исследовательский контур: сводка, выгрузка, распределение по группам.
	mux.HandleFunc("/api/admin/research/summary", middleware.AdminMiddleware(h.ResearchSummary))
	mux.HandleFunc("/api/admin/research/export", middleware.AdminMiddleware(h.ResearchExport))
	mux.HandleFunc("/api/admin/research/group", middleware.AdminMiddleware(h.ResearchSetGroup))
	mux.HandleFunc("/api/research/consent", middleware.AuthMiddleware(h.ResearchConsent))
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
