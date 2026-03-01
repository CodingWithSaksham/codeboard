package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/creative-computing-society/codeboard/internal/cache"
	"github.com/creative-computing-society/codeboard/internal/config"
	"github.com/creative-computing-society/codeboard/internal/db"
	authHandler "github.com/creative-computing-society/codeboard/internal/handlers/auth"
	"github.com/creative-computing-society/codeboard/internal/handlers/leaderboard"
	"github.com/creative-computing-society/codeboard/internal/middleware"
	"github.com/creative-computing-society/codeboard/internal/scheduler"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func main() {
	config.Load()
	db.Connect()
	db.Migrate()
	cache.Connect()

	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{"error": err.Error()})
		},
	})

	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*", // mirrors CORS_ORIGIN_ALLOW_ALL = True
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST",
	}))

	// ── Google OAuth routes ───────────────────────────────────────────────────
	// GET /auth/google/login   → redirect to Google consent screen
	// GET /auth/google/callback → exchange code, upsert user, return JWT
	auth := app.Group("/auth/google")
	auth.Get("/login", authHandler.GoogleLogin)
	auth.Get("/callback", authHandler.GoogleCallback)

	// ── Public leaderboard routes (no auth required) ──────────────────────────
	app.Get("/refresh_data/", leaderboard.DebugRefreshUserData)
	app.Get("/questions/all/", leaderboard.GetAllQuestions)
	app.Get("/daily/", leaderboard.DailyLeaderboard)
	app.Get("/weekly/", leaderboard.WeeklyLeaderboard)
	app.Get("/monthly/", leaderboard.MonthlyLeaderboard)

	// ── Protected routes (Bearer JWT required) ────────────────────────────────
	protected := app.Group("/", middleware.RequireAuth)
	protected.Post("/register/", leaderboard.Register)
	protected.Get("/user/profile/", leaderboard.Profile)
	protected.Get("/questions/today/", leaderboard.GetQuestionsForTheDay)

	// ── Cron scheduler (replaces Celery Beat) ────────────────────────────────
	scheduler.Start()
	defer scheduler.Stop()

	// ── Graceful shutdown ─────────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("Codeboard listening on :%s", config.C.Port)
		if err := app.Listen(":" + config.C.Port); err != nil {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-quit
	log.Println("Shutting down...")
	_ = app.Shutdown()
}
