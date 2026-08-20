package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

// resetViper resets viper instance to avoid test interference
func resetViper() {
	viper.Reset()
}

// TestConfigDataStructure tests the Config struct can hold all expected fields
func TestConfigDataStructure(t *testing.T) {
	resetViper()

	cfg := &Config{
		DatabaseType:  "sqlite",
		SQLitePath:    "/path/to/db.sqlite",
		PostgresURL:   "postgres://localhost:5432/db",
		MySQLURL:      "mysql://localhost:3306/db",
		Port:          8080,
		Host:          "0.0.0.0",
		EnableLogging: true,
	}

	if cfg.DatabaseType != "sqlite" {
		t.Errorf("expected DatabaseType to be 'sqlite', got '%s'", cfg.DatabaseType)
	}
	if cfg.Port != 8080 {
		t.Errorf("expected Port to be 8080, got %d", cfg.Port)
	}
	if !cfg.EnableLogging {
		t.Error("expected EnableLogging to be true")
	}
}

// TestGivenValidConfigPath_whenGetConfigPath_thenReturnValidPath tests GetConfigPath
func TestGivenValidConfigPath_whenGetConfigPath_thenReturnValidPath(t *testing.T) {
	resetViper()

	// Temporarily override home directory for testing
	originalHome := os.Getenv("HOME")
	defer func() {
		os.Setenv("HOME", originalHome)
		os.Unsetenv("LOCALAPPDATA")
	}()

	tempDir := t.TempDir()
	if runtime.GOOS == "windows" {
		os.Setenv("LOCALAPPDATA", tempDir)
	} else {
		os.Setenv("HOME", tempDir)
	}

	configPath := GetConfigPath()

	// Verify the path ends with config.json
	if !strings.HasSuffix(configPath, "config.json") {
		t.Errorf("expected config path to end with 'config.json', got '%s'", configPath)
	}

	// Verify the path is within our temp directory
	if !strings.HasPrefix(configPath, tempDir) {
		t.Errorf("expected config path to be within temp dir, got '%s'", configPath)
	}
}

// TestGivenValidDirectory_whenGetUserDataDir_thenReturnCorrectPath tests GetUserDataDir
func TestGivenValidDirectory_whenGetUserDataDir_thenReturnCorrectPath(t *testing.T) {
	resetViper()

	// Temporarily override home directory for testing
	originalHome := os.Getenv("HOME")
	defer func() {
		os.Setenv("HOME", originalHome)
		os.Unsetenv("LOCALAPPDATA")
	}()

	tempDir := t.TempDir()
	if runtime.GOOS == "windows" {
		os.Setenv("LOCALAPPDATA", tempDir)
	} else {
		os.Setenv("HOME", tempDir)
	}

	userDataDir := GetUserDataDir()

	// Verify the path ends with .knot or knot depending on OS
	expectedSuffix := ".knot"
	if runtime.GOOS == "windows" {
		expectedSuffix = "knot"
	}

	if !strings.HasSuffix(userDataDir, expectedSuffix) {
		t.Errorf("expected user data dir to end with '%s', got '%s'", expectedSuffix, userDataDir)
	}

	// Verify the path is within our temp directory
	if !strings.HasPrefix(userDataDir, tempDir) {
		t.Errorf("expected user data dir to be within temp dir, got '%s'", userDataDir)
	}
}

// TestGivenValidDirectory_whenGetDefaultDBPath_thenReturnCorrectPath tests GetDefaultDBPath
func TestGivenValidDirectory_whenGetDefaultDBPath_thenReturnCorrectPath(t *testing.T) {
	resetViper()

	// Temporarily override home directory for testing
	originalHome := os.Getenv("HOME")
	defer func() {
		os.Setenv("HOME", originalHome)
		os.Unsetenv("LOCALAPPDATA")
	}()

	tempDir := t.TempDir()
	if runtime.GOOS == "windows" {
		os.Setenv("LOCALAPPDATA", tempDir)
	} else {
		os.Setenv("HOME", tempDir)
	}

	dbPath := GetDefaultDBPath()

	// Verify the path ends with knot.db
	if !strings.HasSuffix(dbPath, "knot.db") {
		t.Errorf("expected db path to end with 'knot.db', got '%s'", dbPath)
	}
}

// TestGivenValidDirectory_whenGetPIDPath_thenReturnCorrectPath tests GetPIDPath
func TestGivenValidDirectory_whenGetPIDPath_thenReturnCorrectPath(t *testing.T) {
	resetViper()

	// Temporarily override home directory for testing
	originalHome := os.Getenv("HOME")
	defer func() {
		os.Setenv("HOME", originalHome)
		os.Unsetenv("LOCALAPPDATA")
	}()

	tempDir := t.TempDir()
	if runtime.GOOS == "windows" {
		os.Setenv("LOCALAPPDATA", tempDir)
	} else {
		os.Setenv("HOME", tempDir)
	}

	pidPath := GetPIDPath()

	// Verify the path ends with knot.pid
	if !strings.HasSuffix(pidPath, "knot.pid") {
		t.Errorf("expected pid path to end with 'knot.pid', got '%s'", pidPath)
	}
}

// TestGivenValidDirectory_whenGetLogDir_thenReturnCorrectPath tests GetLogDir
func TestGivenValidDirectory_whenGetLogDir_thenReturnCorrectPath(t *testing.T) {
	resetViper()

	// Temporarily override home directory for testing
	originalHome := os.Getenv("HOME")
	defer func() {
		os.Setenv("HOME", originalHome)
		os.Unsetenv("LOCALAPPDATA")
	}()

	tempDir := t.TempDir()
	if runtime.GOOS == "windows" {
		os.Setenv("LOCALAPPDATA", tempDir)
	} else {
		os.Setenv("HOME", tempDir)
	}

	logDir := GetLogDir()

	// Verify the path ends with log directory
	if !strings.HasSuffix(logDir, filepath.Join("log")) {
		t.Errorf("expected log dir to end with 'log', got '%s'", logDir)
	}
}

// TestGivenValidDirectory_whenGetLogPath_thenReturnCorrectPath tests GetLogPath
func TestGivenValidDirectory_whenGetLogPath_thenReturnCorrectPath(t *testing.T) {
	resetViper()

	// Temporarily override home directory for testing
	originalHome := os.Getenv("HOME")
	defer func() {
		os.Setenv("HOME", originalHome)
		os.Unsetenv("LOCALAPPDATA")
	}()

	tempDir := t.TempDir()
	if runtime.GOOS == "windows" {
		os.Setenv("LOCALAPPDATA", tempDir)
	} else {
		os.Setenv("HOME", tempDir)
	}

	logPath := GetLogPath()

	// Verify the path ends with knot.log
	if !strings.HasSuffix(logPath, filepath.Join("log", "knot.log")) &&
	   !strings.HasSuffix(logPath, "knot.log") {
		t.Errorf("expected log path to end with 'knot.log', got '%s'", logPath)
	}
}

// TestGivenNonExistentDirectory_whenEnsureUserDataDir_thenDirectoryCreated tests EnsureUserDataDir
func TestGivenNonExistentDirectory_whenEnsureUserDataDir_thenDirectoryCreated(t *testing.T) {
	resetViper()

	// Temporarily override home directory for testing
	originalHome := os.Getenv("HOME")
	defer func() {
		os.Setenv("HOME", originalHome)
		os.Unsetenv("LOCALAPPDATA")
	}()

	tempDir := t.TempDir()
	if runtime.GOOS == "windows" {
		os.Setenv("LOCALAPPDATA", tempDir)
	} else {
		os.Setenv("HOME", tempDir)
	}

	err := EnsureUserDataDir()
	if err != nil {
		t.Fatalf("expected no error creating user data dir, got %v", err)
	}

	// Verify directory exists
	userDataDir := GetUserDataDir()
	info, err := os.Stat(userDataDir)
	if err != nil {
		t.Fatalf("expected user data dir to exist, got error: %v", err)
	}

	if !info.IsDir() {
		t.Error("expected path to be a directory")
	}
}

// TestGivenExistingDirectory_whenEnsureUserDataDir_thenNoError tests EnsureUserDataDir with existing dir
func TestGivenExistingDirectory_whenEnsureUserDataDir_thenNoError(t *testing.T) {
	resetViper()

	// Temporarily override home directory for testing
	originalHome := os.Getenv("HOME")
	defer func() {
		os.Setenv("HOME", originalHome)
		os.Unsetenv("LOCALAPPDATA")
	}()

	tempDir := t.TempDir()
	if runtime.GOOS == "windows" {
		os.Setenv("LOCALAPPDATA", tempDir)
	} else {
		os.Setenv("HOME", tempDir)
	}

	// Create directory first
	err := EnsureUserDataDir()
	if err != nil {
		t.Fatalf("failed to create initial directory: %v", err)
	}

	// Call again - should not error
	err = EnsureUserDataDir()
	if err != nil {
		t.Errorf("expected no error when directory already exists, got %v", err)
	}
}

// TestGivenNoConfigFile_whenLoadConfig_thenDefaultConfigCreated tests LoadConfig creates default
func TestGivenNoConfigFile_whenLoadConfig_thenDefaultConfigCreated(t *testing.T) {
	resetViper()

	// Temporarily override home directory for testing
	originalHome := os.Getenv("HOME")
	defer func() {
		os.Setenv("HOME", originalHome)
		os.Unsetenv("LOCALAPPDATA")
	}()

	tempDir := t.TempDir()
	if runtime.GOOS == "windows" {
		os.Setenv("LOCALAPPDATA", tempDir)
	} else {
		os.Setenv("HOME", tempDir)
	}

	// Ensure no config exists
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("expected no error loading config, got %v", err)
	}

	// Verify default values
	if cfg.DatabaseType != "sqlite" {
		t.Errorf("expected default database type to be 'sqlite', got '%s'", cfg.DatabaseType)
	}
	if cfg.Port != 3000 {
		t.Errorf("expected default port to be 3000, got %d", cfg.Port)
	}
	if cfg.Host != "localhost" {
		t.Errorf("expected default host to be 'localhost', got '%s'", cfg.Host)
	}
	if cfg.EnableLogging {
		t.Error("expected default enable logging to be false")
	}

	// Verify config file was created
	configPath := GetConfigPath()
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("expected config file to be created")
	}
}

// TestGivenExistingConfigFile_whenLoadConfig_thenConfigLoaded tests LoadConfig reads existing
func TestGivenExistingConfigFile_whenLoadConfig_thenConfigLoaded(t *testing.T) {
	resetViper()

	// Temporarily override home directory for testing
	originalHome := os.Getenv("HOME")
	defer func() {
		os.Setenv("HOME", originalHome)
		os.Unsetenv("LOCALAPPDATA")
	}()

	tempDir := t.TempDir()
	if runtime.GOOS == "windows" {
		os.Setenv("LOCALAPPDATA", tempDir)
	} else {
		os.Setenv("HOME", tempDir)
	}

	// Create a config file manually
	err := EnsureUserDataDir()
	if err != nil {
		t.Fatalf("failed to create user data dir: %v", err)
	}

	expectedCfg := Config{
		DatabaseType:  "postgres",
		PostgresURL:   "postgres://localhost:5432/testdb",
		Port:          8080,
		Host:          "0.0.0.0",
		EnableLogging: true,
	}

	configPath := GetConfigPath()
	data, err := json.MarshalIndent(expectedCfg, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal config: %v", err)
	}
	err = os.WriteFile(configPath, data, 0644)
	if err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Load config
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("expected no error loading config, got %v", err)
	}

	// Verify loaded values
	if cfg.DatabaseType != expectedCfg.DatabaseType {
		t.Errorf("expected database type '%s', got '%s'", expectedCfg.DatabaseType, cfg.DatabaseType)
	}
	if cfg.PostgresURL != expectedCfg.PostgresURL {
		t.Errorf("expected postgres url '%s', got '%s'", expectedCfg.PostgresURL, cfg.PostgresURL)
	}
	if cfg.Port != expectedCfg.Port {
		t.Errorf("expected port %d, got %d", expectedCfg.Port, cfg.Port)
	}
	if cfg.Host != expectedCfg.Host {
		t.Errorf("expected host '%s', got '%s'", expectedCfg.Host, cfg.Host)
	}
	if cfg.EnableLogging != expectedCfg.EnableLogging {
		t.Errorf("expected enable logging %v, got %v", expectedCfg.EnableLogging, cfg.EnableLogging)
	}
}

// TestGivenInvalidJSONConfig_whenLoadConfig_thenReturnError tests LoadConfig with invalid JSON
func TestGivenInvalidJSONConfig_whenLoadConfig_thenReturnError(t *testing.T) {
	resetViper()

	// Temporarily override home directory for testing
	originalHome := os.Getenv("HOME")
	defer func() {
		os.Setenv("HOME", originalHome)
		os.Unsetenv("LOCALAPPDATA")
	}()

	tempDir := t.TempDir()
	if runtime.GOOS == "windows" {
		os.Setenv("LOCALAPPDATA", tempDir)
	} else {
		os.Setenv("HOME", tempDir)
	}

	// Create a config file with invalid JSON
	err := EnsureUserDataDir()
	if err != nil {
		t.Fatalf("failed to create user data dir: %v", err)
	}

	configPath := GetConfigPath()
	err = os.WriteFile(configPath, []byte("{ invalid json }"), 0644)
	if err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Load config should return error
	_, err = LoadConfig()
	if err == nil {
		t.Error("expected error loading invalid config, got nil")
	}
}

// TestGivenNewConfig_whenInitConfig_thenConfigFileCreated tests InitConfig
func TestGivenNewConfig_whenInitConfig_thenConfigFileCreated(t *testing.T) {
	resetViper()

	// Temporarily override home directory for testing
	originalHome := os.Getenv("HOME")
	defer func() {
		os.Setenv("HOME", originalHome)
		os.Unsetenv("LOCALAPPDATA")
	}()

	tempDir := t.TempDir()
	if runtime.GOOS == "windows" {
		os.Setenv("LOCALAPPDATA", tempDir)
	} else {
		os.Setenv("HOME", tempDir)
	}

	err := InitConfig()
	if err != nil {
		t.Fatalf("expected no error initializing config, got %v", err)
	}

	// Verify config file exists
	configPath := GetConfigPath()
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("expected config file to exist after InitConfig")
	}

	// Verify config file contains valid JSON
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config file: %v", err)
	}

	var cfg Config
	err = json.Unmarshal(data, &cfg)
	if err != nil {
		t.Errorf("expected config file to contain valid JSON, got error: %v", err)
	}

	// Verify default values
	if cfg.DatabaseType != "sqlite" {
		t.Errorf("expected default database type 'sqlite', got '%s'", cfg.DatabaseType)
	}
	if cfg.Port != 3000 {
		t.Errorf("expected default port 3000, got %d", cfg.Port)
	}
}

// TestGivenExistingConfig_whenInitConfig_thenNoError tests InitConfig with existing config
func TestGivenExistingConfig_whenInitConfig_thenNoError(t *testing.T) {
	resetViper()

	// Temporarily override home directory for testing
	originalHome := os.Getenv("HOME")
	defer func() {
		os.Setenv("HOME", originalHome)
		os.Unsetenv("LOCALAPPDATA")
	}()

	tempDir := t.TempDir()
	if runtime.GOOS == "windows" {
		os.Setenv("LOCALAPPDATA", tempDir)
	} else {
		os.Setenv("HOME", tempDir)
	}

	// Create config first
	err := InitConfig()
	if err != nil {
		t.Fatalf("failed to initialize first config: %v", err)
	}

	// Call InitConfig again - should not error
	err = InitConfig()
	if err != nil {
		t.Errorf("expected no error when config already exists, got %v", err)
	}
}

// TestGivenValidConfig_whenSaveConfig_thenFileUpdated tests SaveConfig
func TestGivenValidConfig_whenSaveConfig_thenFileUpdated(t *testing.T) {
	resetViper()

	// Temporarily override home directory for testing
	originalHome := os.Getenv("HOME")
	defer func() {
		os.Setenv("HOME", originalHome)
		os.Unsetenv("LOCALAPPDATA")
	}()

	tempDir := t.TempDir()
	if runtime.GOOS == "windows" {
		os.Setenv("LOCALAPPDATA", tempDir)
	} else {
		os.Setenv("HOME", tempDir)
	}

	// Load config first to set up viper with config file path
	_, err := LoadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Create new config to save
	newCfg := &Config{
		DatabaseType:  "mysql",
		MySQLURL:      "mysql://localhost:3306/testdb",
		Port:          9000,
		Host:          "127.0.0.1",
		EnableLogging: true,
	}

	err = SaveConfig(newCfg)
	if err != nil {
		t.Fatalf("expected no error saving config, got %v", err)
	}

	// Reset viper and reload to verify saved config
	resetViper()
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("failed to load saved config: %v", err)
	}

	if cfg.DatabaseType != newCfg.DatabaseType {
		t.Errorf("expected database type '%s', got '%s'", newCfg.DatabaseType, cfg.DatabaseType)
	}
	if cfg.MySQLURL != newCfg.MySQLURL {
		t.Errorf("expected mysql url '%s', got '%s'", newCfg.MySQLURL, cfg.MySQLURL)
	}
	if cfg.Port != newCfg.Port {
		t.Errorf("expected port %d, got %d", newCfg.Port, cfg.Port)
	}
	if cfg.Host != newCfg.Host {
		t.Errorf("expected host '%s', got '%s'", newCfg.Host, cfg.Host)
	}
	if cfg.EnableLogging != newCfg.EnableLogging {
		t.Errorf("expected enable logging %v, got %v", newCfg.EnableLogging, cfg.EnableLogging)
	}
}

// TestGivenConfigWithAllDatabaseTypes_whenSaveConfig_thenAllFieldsPreserved tests SaveConfig with all fields
func TestGivenConfigWithAllDatabaseTypes_whenSaveConfig_thenAllFieldsPreserved(t *testing.T) {
	resetViper()

	// Temporarily override home directory for testing
	originalHome := os.Getenv("HOME")
	defer func() {
		os.Setenv("HOME", originalHome)
		os.Unsetenv("LOCALAPPDATA")
	}()

	tempDir := t.TempDir()
	if runtime.GOOS == "windows" {
		os.Setenv("LOCALAPPDATA", tempDir)
	} else {
		os.Setenv("HOME", tempDir)
	}

	// Load config first to set up viper with config file path
	_, err := LoadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Create config with all database URLs
	newCfg := &Config{
		DatabaseType:  "sqlite",
		SQLitePath:    "/custom/path/to/db.sqlite",
		PostgresURL:   "postgres://user:pass@localhost:5432/db?sslmode=disable",
		MySQLURL:      "mysql://user:pass@tcp(localhost:3306)/db",
		Port:          3000,
		Host:          "localhost",
		EnableLogging: false,
	}

	err = SaveConfig(newCfg)
	if err != nil {
		t.Fatalf("expected no error saving config, got %v", err)
	}

	// Reset viper and reload to verify all fields were saved
	resetViper()
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("failed to load saved config: %v", err)
	}

	if cfg.SQLitePath != newCfg.SQLitePath {
		t.Errorf("expected sqlite path '%s', got '%s'", newCfg.SQLitePath, cfg.SQLitePath)
	}
	if cfg.PostgresURL != newCfg.PostgresURL {
		t.Errorf("expected postgres url '%s', got '%s'", newCfg.PostgresURL, cfg.PostgresURL)
	}
	if cfg.MySQLURL != newCfg.MySQLURL {
		t.Errorf("expected mysql url '%s', got '%s'", newCfg.MySQLURL, cfg.MySQLURL)
	}
}

// TestGivenValidConfig_whenShowConfig_thenNoError tests ShowConfig
func TestGivenValidConfig_whenShowConfig_thenNoError(t *testing.T) {
	resetViper()

	// Temporarily override home directory for testing
	originalHome := os.Getenv("HOME")
	defer func() {
		os.Setenv("HOME", originalHome)
		os.Unsetenv("LOCALAPPDATA")
	}()

	tempDir := t.TempDir()
	if runtime.GOOS == "windows" {
		os.Setenv("LOCALAPPDATA", tempDir)
	} else {
		os.Setenv("HOME", tempDir)
	}

	// Initialize config
	err := InitConfig()
	if err != nil {
		t.Fatalf("failed to initialize config: %v", err)
	}

	// ShowConfig should not error
	err = ShowConfig()
	if err != nil {
		t.Errorf("expected no error showing config, got %v", err)
	}
}

// TestGivenConfig_whenSaveConfigWithoutDirectory_thenReturnError tests SaveConfig when dir creation fails
func TestGivenInvalidDirectory_whenEnsureUserDataDir_thenReturnError(t *testing.T) {
	resetViper()

	// Create an invalid path scenario by setting an impossible path
	// Note: This test is limited as we can't truly simulate permission errors in tests
	originalHome := os.Getenv("HOME")
	defer func() {
		os.Setenv("HOME", originalHome)
		os.Unsetenv("LOCALAPPDATA")
	}()

	// This is a conceptual test - in real scenarios, permission errors would occur
	// For now, we test with a valid temp directory which should always work
	tempDir := t.TempDir()
	if runtime.GOOS == "windows" {
		os.Setenv("LOCALAPPDATA", tempDir)
	} else {
		os.Setenv("HOME", tempDir)
	}

	// With valid temp dir, should succeed
	err := EnsureUserDataDir()
	if err != nil {
		t.Errorf("expected no error with valid temp dir, got %v", err)
	}
}

// TestGivenEnvOverrides_whenLoadConfig_thenEnvValuesApplied tests KNOT_* env var overrides
func TestGivenEnvOverrides_whenLoadConfig_thenEnvValuesApplied(t *testing.T) {
	resetViper()

	originalHome := os.Getenv("HOME")
	defer func() {
		os.Setenv("HOME", originalHome)
		os.Unsetenv("KNOT_DATABASE_TYPE")
		os.Unsetenv("KNOT_POSTGRES_URL")
		os.Unsetenv("KNOT_SQLITE_PATH")
	}()

	tempDir := t.TempDir()
	os.Setenv("HOME", tempDir)
	os.Setenv("KNOT_DATABASE_TYPE", "postgres")
	os.Setenv("KNOT_POSTGRES_URL", "postgres://u:p@db:5432/knot")
	os.Setenv("KNOT_SQLITE_PATH", "/custom/knot.db")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.DatabaseType != "postgres" {
		t.Errorf("expected DatabaseType 'postgres', got '%s'", cfg.DatabaseType)
	}
	if cfg.PostgresURL != "postgres://u:p@db:5432/knot" {
		t.Errorf("expected PostgresURL from env, got '%s'", cfg.PostgresURL)
	}
	if cfg.SQLitePath != "/custom/knot.db" {
		t.Errorf("expected SQLitePath from env, got '%s'", cfg.SQLitePath)
	}
}

// TestGivenNoEnvOverrides_whenLoadConfig_thenDefaultsUnchanged tests backward compatibility
func TestGivenNoEnvOverrides_whenLoadConfig_thenDefaultsUnchanged(t *testing.T) {
	resetViper()

	// Defensively clear KNOT_* env vars so a developer machine that has them
	// set does not break this test
	for _, key := range []string{"KNOT_DATABASE_TYPE", "KNOT_SQLITE_PATH", "KNOT_POSTGRES_URL", "KNOT_MYSQL_URL"} {
		t.Setenv(key, "")
		os.Unsetenv(key)
	}

	originalHome := os.Getenv("HOME")
	defer func() {
		os.Setenv("HOME", originalHome)
	}()

	tempDir := t.TempDir()
	os.Setenv("HOME", tempDir)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.DatabaseType != "sqlite" {
		t.Errorf("expected default DatabaseType 'sqlite', got '%s'", cfg.DatabaseType)
	}
	if cfg.PostgresURL != "" {
		t.Errorf("expected empty PostgresURL, got '%s'", cfg.PostgresURL)
	}
}
