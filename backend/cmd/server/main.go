package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ProjAnvil/knot/backend/internal/config"
	"github.com/ProjAnvil/knot/backend/internal/database"
	"github.com/ProjAnvil/knot/backend/internal/embedded"
	"github.com/ProjAnvil/knot/backend/internal/handlers"
	"github.com/ProjAnvil/knot/backend/pkg/logger"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if err := logger.InitLogger(cfg); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	logger.Log.Info("Starting Knot server...")

	db, err := database.InitDatabase(cfg)
	if err != nil {
		logger.Log.Fatal(fmt.Sprintf("Failed to initialize database: %v", err))
	}

	app := fiber.New(fiber.Config{AppName: "Knot"})
	app.Use(recover.New())
	app.Use(cors.New())

	// Health check with database status
	app.Get("/api/health", func(c *fiber.Ctx) error {
		dbStatus := "ok"
		dbError := ""
		sqlDB, err := db.DB()
		if err != nil {
			dbStatus = "error"
			dbError = err.Error()
		} else if err := sqlDB.Ping(); err != nil {
			dbStatus = "error"
			dbError = err.Error()
		}

		resp := fiber.Map{
			"status":    "ok",
			"runtime":   "go",
			"timestamp": time.Now().Format(time.RFC3339),
			"database": fiber.Map{
				"status": dbStatus,
				"type":   cfg.DatabaseType,
			},
		}
		if dbError != "" {
			resp["database"].(fiber.Map)["error"] = dbError
		}
		return c.JSON(resp)
	})

	handlers.SetupRoutes(app, db)

	// Static file serving for frontend (only if embedded)
	if embedded.HasFrontend() {
		frontendFS, err := embedded.GetFrontendFS()
		if err != nil {
			logger.Log.Warn("Failed to get embedded frontend filesystem")
		} else {
			fmt.Printf("📁 Serving embedded frontend\n")
			app.Use(handlers.ServeEmbeddedFiles(frontendFS))
		}
	} else {
		fmt.Printf("ℹ️  No embedded frontend (API-only mode)\n")
		fmt.Printf("   This is normal for knot-server. Use CLI to start with frontend.\n")
	}

	port := cfg.Port
	if envPort := os.Getenv("PORT"); envPort != "" {
		fmt.Sscanf(envPort, "%d", &port)
	}
	host := cfg.Host
	if envHost := os.Getenv("HOST"); envHost != "" {
		host = envHost
	}

	addr := fmt.Sprintf("%s:%d", host, port)
	logger.Log.Info(fmt.Sprintf("Server starting on %s", addr))
	fmt.Printf("🚀 Server running on http://%s\n", addr)

	if err := app.Listen(addr); err != nil {
		logger.Log.Fatal(fmt.Sprintf("Failed to start server: %v", err))
	}
}
