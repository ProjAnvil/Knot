package main

import (
	"fmt"
	"log"
	"os"
	"time"

	knot "github.com/ProjAnvil/knot/backend"
	"github.com/ProjAnvil/knot/backend/internal/config"
	"github.com/ProjAnvil/knot/backend/internal/database"
	"github.com/ProjAnvil/knot/backend/internal/embedded"
	"github.com/ProjAnvil/knot/backend/internal/handlers"
	"github.com/ProjAnvil/knot/backend/pkg/logger"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
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

	// BodyLimit is raised to 10 MB so multi-format import files fit within
	// Fiber's request body cap (the default is 4 MB).
	app := fiber.New(fiber.Config{AppName: "Knot", BodyLimit: 10 * 1024 * 1024})
	app.Use(recover.New())
	// Negotiate gzip/deflate/brotli. The API payloads are highly repetitive JSON
	// (the sidebar list is ~84 KB uncompressed, ~14 KB gzipped), so compression
	// is the single biggest win for clients on slow links.
	app.Use(compress.New(compress.Config{Level: compress.LevelDefault}))
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

	// Mount the embeddable engine under the default "/api" prefix. The host
	// keeps the health endpoint, the static frontend and the process
	// lifecycle; the engine's Close is a documented no-op, so it is not
	// deferred here.
	k, err := knot.New(knot.Config{DB: db, APIPrefix: "/api"})
	if err != nil {
		logger.Log.Fatal(fmt.Sprintf("Failed to initialize engine: %v", err))
	}
	k.Mount(app)

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
