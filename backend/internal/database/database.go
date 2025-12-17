package database

import (
	"fmt"
	"strings"

	"github.com/ProjAnvil/knot/backend/internal/config"
	"github.com/ProjAnvil/knot/backend/internal/models"
	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// InitDatabase initializes the database connection based on configuration
func InitDatabase(cfg *config.Config) (*gorm.DB, error) {
	var dialector gorm.Dialector

	switch strings.ToLower(cfg.DatabaseType) {
	case "postgres", "postgresql":
		if cfg.PostgresURL == "" {
			return nil, fmt.Errorf("PostgreSQL URL not configured. Please set postgresUrl in config")
		}
		dialector = postgres.Open(cfg.PostgresURL)
		fmt.Println("Using PostgreSQL database")

	case "mysql":
		if cfg.MySQLURL == "" {
			return nil, fmt.Errorf("MySQL URL not configured. Please set mysqlUrl in config")
		}
		dialector = mysql.Open(cfg.MySQLURL)
		fmt.Println("Using MySQL database")

	case "sqlite", "":
		if cfg.SQLitePath == "" {
			return nil, fmt.Errorf("SQLite path not configured")
		}
		dialector = sqlite.Open(cfg.SQLitePath)
		fmt.Printf("Using SQLite database: %s\n", cfg.SQLitePath)

	default:
		return nil, fmt.Errorf("unsupported database type: %s", cfg.DatabaseType)
	}

	// Configure GORM
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	}

	if cfg.EnableLogging {
		gormConfig.Logger = logger.Default.LogMode(logger.Info)
	}

	db, err := gorm.Open(dialector, gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Check if tables exist before running migration
	// This prevents migration issues with existing data
	var tableExists bool
	db.Raw("SELECT count(*) > 0 FROM sqlite_master WHERE type='table' AND name='groups'").Scan(&tableExists)

	if !tableExists {
		// Tables don't exist, run full migration
		if err := db.AutoMigrate(&models.Group{}, &models.API{}, &models.Parameter{}); err != nil {
			return nil, fmt.Errorf("failed to run auto-migration: %w", err)
		}
	} else {
		// Tables exist, only sync new columns/indexes without altering existing structure
		// Use Migrator to add missing columns only
		migrator := db.Migrator()

		// Ensure all tables exist
		if !migrator.HasTable(&models.Group{}) {
			if err := migrator.CreateTable(&models.Group{}); err != nil {
				return nil, fmt.Errorf("failed to create groups table: %w", err)
			}
		}
		if !migrator.HasTable(&models.API{}) {
			if err := migrator.CreateTable(&models.API{}); err != nil {
				return nil, fmt.Errorf("failed to create apis table: %w", err)
			}
		}
		if !migrator.HasTable(&models.Parameter{}) {
			if err := migrator.CreateTable(&models.Parameter{}); err != nil {
				return nil, fmt.Errorf("failed to create parameters table: %w", err)
			}
		}
	}

	fmt.Println("✓ Database initialized successfully")

	return db, nil
}
