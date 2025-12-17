package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/viper"
)

// Config holds the application configuration
type Config struct {
	DatabaseType  string `mapstructure:"databaseType"`
	SQLitePath    string `mapstructure:"sqlitePath"`
	PostgresURL   string `mapstructure:"postgresUrl"`
	MySQLURL      string `mapstructure:"mysqlUrl"`
	Port          int    `mapstructure:"port"`
	Host          string `mapstructure:"host"`
	EnableLogging bool   `mapstructure:"enableLogging"`
}

// GetUserDataDir returns the user data directory for Knot
// - Linux/macOS: ~/.knot
// - Windows: %LOCALAPPDATA%/knot
func GetUserDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		panic(fmt.Sprintf("failed to get user home directory: %v", err))
	}

	if runtime.GOOS == "windows" {
		// Windows: use LOCALAPPDATA
		appData := os.Getenv("LOCALAPPDATA")
		if appData == "" {
			appData = filepath.Join(home, "AppData", "Local")
		}
		return filepath.Join(appData, "knot")
	}

	// Linux/macOS: use ~/.knot
	return filepath.Join(home, ".knot")
}

// GetConfigPath returns the path to the config file
func GetConfigPath() string {
	return filepath.Join(GetUserDataDir(), "config.json")
}

// GetDefaultDBPath returns the default SQLite database path
func GetDefaultDBPath() string {
	return filepath.Join(GetUserDataDir(), "knot.db")
}

// GetPIDPath returns the path to the PID file
func GetPIDPath() string {
	return filepath.Join(GetUserDataDir(), "knot.pid")
}

// GetLogDir returns the log directory path
func GetLogDir() string {
	return filepath.Join(GetUserDataDir(), "log")
}

// GetLogPath returns the log file path
func GetLogPath() string {
	return filepath.Join(GetLogDir(), "knot.log")
}

// EnsureUserDataDir ensures the user data directory exists
func EnsureUserDataDir() error {
	userDir := GetUserDataDir()
	if err := os.MkdirAll(userDir, 0755); err != nil {
		return fmt.Errorf("failed to create user data directory: %w", err)
	}
	return nil
}

// LoadConfig loads configuration from file and defaults
func LoadConfig() (*Config, error) {
	// Ensure user data directory exists
	if err := EnsureUserDataDir(); err != nil {
		return nil, err
	}

	viper.SetConfigName("config")
	viper.SetConfigType("json")
	viper.AddConfigPath(GetUserDataDir())

	// Set defaults
	viper.SetDefault("databaseType", "sqlite")
	viper.SetDefault("sqlitePath", GetDefaultDBPath())
	viper.SetDefault("port", 3000)
	viper.SetDefault("host", "localhost")
	viper.SetDefault("enableLogging", false)

	// Try to read config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found, create default
			fmt.Println("⚠️  Config file not found, creating default config...")
			if err := InitConfig(); err != nil {
				return nil, err
			}
			// Reload config after creation
			if err := viper.ReadInConfig(); err != nil {
				return nil, fmt.Errorf("failed to read config after creation: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}

// InitConfig initializes the configuration file with defaults
func InitConfig() error {
	if err := EnsureUserDataDir(); err != nil {
		return err
	}

	configPath := GetConfigPath()

	// Check if config file already exists
	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("✓ Config file already exists: %s\n", configPath)
		return nil
	}

	// Create default config
	defaultConfig := Config{
		DatabaseType:  "sqlite",
		SQLitePath:    GetDefaultDBPath(),
		Port:          3000,
		Host:          "localhost",
		EnableLogging: false,
	}

	viper.Set("databaseType", defaultConfig.DatabaseType)
	viper.Set("sqlitePath", defaultConfig.SQLitePath)
	viper.Set("port", defaultConfig.Port)
	viper.Set("host", defaultConfig.Host)
	viper.Set("enableLogging", defaultConfig.EnableLogging)

	if err := viper.WriteConfigAs(configPath); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Printf("✓ Created config file: %s\n", configPath)
	return nil
}

// SaveConfig saves configuration to file
func SaveConfig(config *Config) error {
	if err := EnsureUserDataDir(); err != nil {
		return err
	}

	viper.Set("databaseType", config.DatabaseType)
	viper.Set("sqlitePath", config.SQLitePath)
	viper.Set("postgresUrl", config.PostgresURL)
	viper.Set("mysqlUrl", config.MySQLURL)
	viper.Set("port", config.Port)
	viper.Set("host", config.Host)
	viper.Set("enableLogging", config.EnableLogging)

	if err := viper.WriteConfig(); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// ShowConfig displays the current configuration
func ShowConfig() error {
	config, err := LoadConfig()
	if err != nil {
		return err
	}

	userDir := GetUserDataDir()

	fmt.Printf("\n📋 Knot Configuration\n\n")
	fmt.Printf("User Directory:  %s\n", userDir)
	fmt.Printf("Config File:     %s\n", GetConfigPath())
	fmt.Printf("\n")
	fmt.Printf("Database Type:   %s\n", config.DatabaseType)

	switch config.DatabaseType {
	case "sqlite":
		fmt.Printf("SQLite Path:     %s\n", config.SQLitePath)
	case "postgres", "postgresql":
		if config.PostgresURL != "" {
			fmt.Printf("PostgreSQL URL:  %s\n", config.PostgresURL)
		} else {
			fmt.Printf("PostgreSQL URL:  (not set)\n")
		}
	case "mysql":
		if config.MySQLURL != "" {
			fmt.Printf("MySQL URL:       %s\n", config.MySQLURL)
		} else {
			fmt.Printf("MySQL URL:       (not set)\n")
		}
	}

	fmt.Printf("Server Host:     %s\n", config.Host)
	fmt.Printf("Server Port:     %d\n", config.Port)
	fmt.Printf("Logging:         %v\n", config.EnableLogging)
	fmt.Printf("\n")

	return nil
}
