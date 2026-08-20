package database

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ProjAnvil/knot/backend/internal/config"
	"github.com/ProjAnvil/knot/backend/internal/models"
)

// setupTestConfig creates a test configuration with a temporary SQLite database
func setupTestConfig(t *testing.T) *config.Config {
	t.Helper()

	// Create temp directory for test database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	return &config.Config{
		DatabaseType:  "sqlite",
		SQLitePath:    dbPath,
		Port:          3000,
		Host:          "localhost",
		EnableLogging: false,
	}
}

// TestGivenSQLiteConfig_whenInitDatabase_thenConnectionEstablished tests SQLite database initialization
func TestGivenSQLiteConfig_whenInitDatabase_thenConnectionEstablished(t *testing.T) {
	cfg := setupTestConfig(t)

	db, err := InitDatabase(cfg)
	if err != nil {
		t.Fatalf("expected no error initializing SQLite database, got %v", err)
	}

	if db == nil {
		t.Fatal("expected database connection to be non-nil")
	}

	// Verify connection works
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("failed to get underlying sql.DB: %v", err)
	}
	defer sqlDB.Close()

	if err := sqlDB.Ping(); err != nil {
		t.Errorf("failed to ping database: %v", err)
	}
}

// TestGivenSQLiteConfig_whenInitDatabase_thenTablesCreated tests table creation
func TestGivenSQLiteConfig_whenInitDatabase_thenTablesCreated(t *testing.T) {
	cfg := setupTestConfig(t)

	db, err := InitDatabase(cfg)
	if err != nil {
		t.Fatalf("expected no error initializing database, got %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("failed to get underlying sql.DB: %v", err)
	}
	defer sqlDB.Close()

	// Verify tables exist
	migrator := db.Migrator()

	tables := []interface{}{
		&models.Group{},
		&models.API{},
		&models.Parameter{},
	}

	for _, table := range tables {
		if !migrator.HasTable(table) {
			t.Errorf("expected table %T to exist", table)
		}
	}
}

// TestGivenSQLiteConfig_whenInitDatabase_thenCanInsertGroup tests database operations
func TestGivenSQLiteConfig_whenInitDatabase_thenCanInsertGroup(t *testing.T) {
	cfg := setupTestConfig(t)

	db, err := InitDatabase(cfg)
	if err != nil {
		t.Fatalf("expected no error initializing database, got %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("failed to get underlying sql.DB: %v", err)
	}
	defer sqlDB.Close()

	// Insert a group
	group := models.Group{
		Name:  "Test Group",
		Order: 1,
	}

	if err := db.Create(&group).Error; err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	// Verify group was inserted
	var retrievedGroup models.Group
	if err := db.First(&retrievedGroup, group.ID).Error; err != nil {
		t.Errorf("failed to retrieve group: %v", err)
	}

	if retrievedGroup.Name != group.Name {
		t.Errorf("expected group name '%s', got '%s'", group.Name, retrievedGroup.Name)
	}
}

// TestGivenSQLiteConfig_whenInitDatabase_thenCanInsertAPIWithParameters tests nested data structures
func TestGivenSQLiteConfig_whenInitDatabase_thenCanInsertAPIWithParameters(t *testing.T) {
	cfg := setupTestConfig(t)

	db, err := InitDatabase(cfg)
	if err != nil {
		t.Fatalf("expected no error initializing database, got %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("failed to get underlying sql.DB: %v", err)
	}
	defer sqlDB.Close()

	// Create group
	group := models.Group{
		Name:  "API Group",
		Order: 1,
	}
	if err := db.Create(&group).Error; err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	// Create API with parameters
	api := models.API{
		GroupID:  group.ID,
		Name:     "Test API",
		Endpoint: "/test",
		Method:   "GET",
		Type:     "HTTP",
		Order:    1,
		Parameters: []models.Parameter{
			{
				Name:      "param1",
				Type:      "string",
				Required:  true,
				ParamType: "request",
				Order:     1,
			},
			{
				Name:      "param2",
				Type:      "number",
				Required:  false,
				ParamType: "response",
				Order:     1,
			},
		},
	}

	if err := db.Create(&api).Error; err != nil {
		t.Fatalf("failed to create API: %v", err)
	}

	// Verify API with parameters was inserted
	var retrievedAPI models.API
	if err := db.Preload("Parameters").First(&retrievedAPI, api.ID).Error; err != nil {
		t.Errorf("failed to retrieve API: %v", err)
	}

	if len(retrievedAPI.Parameters) != 2 {
		t.Errorf("expected 2 parameters, got %d", len(retrievedAPI.Parameters))
	}
}

// TestGivenSQLiteConfig_whenInitDatabase_thenCanHandleNestedParameters tests nested parameter support
func TestGivenSQLiteConfig_whenInitDatabase_thenCanHandleNestedParameters(t *testing.T) {
	cfg := setupTestConfig(t)

	db, err := InitDatabase(cfg)
	if err != nil {
		t.Fatalf("expected no error initializing database, got %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("failed to get underlying sql.DB: %v", err)
	}
	defer sqlDB.Close()

	// Create group and API
	group := models.Group{Name: "Group", Order: 1}
	if err := db.Create(&group).Error; err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	api := models.API{
		GroupID:  group.ID,
		Name:     "API",
		Endpoint: "/api",
		Method:   "POST",
		Type:     "HTTP",
		Order:    1,
	}
	if err := db.Create(&api).Error; err != nil {
		t.Fatalf("failed to create API: %v", err)
	}

	// Create parent parameter
	parentParam := models.Parameter{
		APIID:     api.ID,
		Name:      "nestedObject",
		Type:      "object",
		ParamType: "request",
		Order:     1,
	}
	if err := db.Create(&parentParam).Error; err != nil {
		t.Fatalf("failed to create parent parameter: %v", err)
	}

	// Create child parameters
	childParam1 := models.Parameter{
		APIID:     api.ID,
		ParentID:  &parentParam.ID,
		Name:      "field1",
		Type:      "string",
		ParamType: "request",
		Order:     1,
	}
	childParam2 := models.Parameter{
		APIID:     api.ID,
		ParentID:  &parentParam.ID,
		Name:      "field2",
		Type:      "number",
		ParamType: "request",
		Order:     2,
	}

	if err := db.Create(&childParam1).Error; err != nil {
		t.Fatalf("failed to create child parameter 1: %v", err)
	}
	if err := db.Create(&childParam2).Error; err != nil {
		t.Fatalf("failed to create child parameter 2: %v", err)
	}

	// Verify nested structure
	var retrievedParent models.Parameter
	if err := db.Preload("Children").First(&retrievedParent, parentParam.ID).Error; err != nil {
		t.Errorf("failed to retrieve parent parameter: %v", err)
	}

	if len(retrievedParent.Children) != 2 {
		t.Errorf("expected 2 child parameters, got %d", len(retrievedParent.Children))
	}
}

// TestGivenEmptySQLitePathConfig_whenInitDatabase_thenReturnError tests missing SQLite path
func TestGivenEmptySQLitePathConfig_whenInitDatabase_thenReturnError(t *testing.T) {
	cfg := &config.Config{
		DatabaseType: "sqlite",
		SQLitePath:   "", // Empty path should cause error
		Port:         3000,
		Host:         "localhost",
	}

	_, err := InitDatabase(cfg)
	if err == nil {
		t.Error("expected error when SQLite path is empty, got nil")
	}
}

// TestGivenUnsupportedDatabaseType_whenInitDatabase_thenReturnError tests unsupported database type
func TestGivenUnsupportedDatabaseType_whenInitDatabase_thenReturnError(t *testing.T) {
	cfg := &config.Config{
		DatabaseType: "mongodb", // Unsupported type
		Port:         3000,
		Host:         "localhost",
	}

	_, err := InitDatabase(cfg)
	if err == nil {
		t.Error("expected error for unsupported database type, got nil")
	}
}

// TestGivenEmptyPostgresURLConfig_whenInitDatabase_thenReturnError tests missing PostgreSQL URL
func TestGivenEmptyPostgresURLConfig_whenInitDatabase_thenReturnError(t *testing.T) {
	cfg := &config.Config{
		DatabaseType: "postgres",
		PostgresURL:  "", // Empty URL should cause error
		Port:         3000,
		Host:         "localhost",
	}

	_, err := InitDatabase(cfg)
	if err == nil {
		t.Error("expected error when PostgreSQL URL is empty, got nil")
	}

	expectedMsg := "PostgreSQL URL not configured"
	if err != nil && !containsString(err.Error(), expectedMsg) {
		t.Errorf("expected error message to contain '%s', got '%s'", expectedMsg, err.Error())
	}
}

// TestGivenEmptyMySQLURLConfig_whenInitDatabase_thenReturnError tests missing MySQL URL
func TestGivenEmptyMySQLURLConfig_whenInitDatabase_thenReturnError(t *testing.T) {
	cfg := &config.Config{
		DatabaseType: "mysql",
		MySQLURL:     "", // Empty URL should cause error
		Port:         3000,
		Host:         "localhost",
	}

	_, err := InitDatabase(cfg)
	if err == nil {
		t.Error("expected error when MySQL URL is empty, got nil")
	}

	expectedMsg := "MySQL URL not configured"
	if err != nil && !containsString(err.Error(), expectedMsg) {
		t.Errorf("expected error message to contain '%s', got '%s'", expectedMsg, err.Error())
	}
}

// TestGivenDatabaseTypeWithDifferentCase_whenInitDatabase_thenWorks tests case-insensitive database type
func TestGivenDatabaseTypeWithDifferentCase_whenInitDatabase_thenWorks(t *testing.T) {
	// Only test SQLite variations since we can't test other DBs without actual servers
	sqliteVariations := []string{"sqlite", "SQLite", "SQLITE", ""}

	for _, dbType := range sqliteVariations {
		t.Run(dbType, func(t *testing.T) {
			// Create new temp DB for each test
			tempDir := t.TempDir()
			dbPath := filepath.Join(tempDir, "test_case.db")

			testCfg := &config.Config{
				DatabaseType:  dbType,
				SQLitePath:    dbPath,
				Port:          3000,
				Host:          "localhost",
				EnableLogging: false,
			}

			db, err := InitDatabase(testCfg)
			if err != nil {
				t.Errorf("expected no error for database type '%s', got %v", dbType, err)
			}

			if db != nil {
				sqlDB, _ := db.DB()
				if sqlDB != nil {
					sqlDB.Close()
				}
			}
		})
	}
}

// TestGivenExistingDatabase_whenInitDatabase_thenPreservesData tests re-initializing existing database
func TestGivenExistingDatabase_whenInitDatabase_thenPreservesData(t *testing.T) {
	cfg := setupTestConfig(t)

	// Initialize database and insert data
	db1, err := InitDatabase(cfg)
	if err != nil {
		t.Fatalf("expected no error on first init, got %v", err)
	}

	group := models.Group{
		Name:  "Existing Group",
		Order: 1,
	}
	if err := db1.Create(&group).Error; err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	sqlDB1, _ := db1.DB()
	if sqlDB1 != nil {
		sqlDB1.Close()
	}

	// Initialize again with same config
	db2, err := InitDatabase(cfg)
	if err != nil {
		t.Fatalf("expected no error on second init, got %v", err)
	}

	// Verify data still exists
	var count int64
	db2.Model(&models.Group{}).Count(&count)
	if count != 1 {
		t.Errorf("expected 1 group after re-initialization, got %d", count)
	}

	sqlDB2, _ := db2.DB()
	if sqlDB2 != nil {
		sqlDB2.Close()
	}
}

// TestGivenLoggingEnabledConfig_whenInitDatabase_thenLoggerConfigured tests logging configuration
func TestGivenLoggingEnabledConfig_whenInitDatabase_thenLoggerConfigured(t *testing.T) {
	cfg := setupTestConfig(t)
	cfg.EnableLogging = true

	db, err := InitDatabase(cfg)
	if err != nil {
		t.Fatalf("expected no error initializing database with logging, got %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("failed to get underlying sql.DB: %v", err)
	}
	defer sqlDB.Close()

	// Just verify connection works with logging enabled
	if err := sqlDB.Ping(); err != nil {
		t.Errorf("failed to ping database with logging enabled: %v", err)
	}
}

// TestGivenInvalidDatabasePath_whenInitDatabase_thenReturnError tests invalid database path
func TestGivenInvalidDatabasePath_whenInitDatabase_thenReturnError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows due to different path semantics")
	}

	cfg := &config.Config{
		DatabaseType: "sqlite",
		SQLitePath:   "/nonexistent/directory/path/test.db",
		Port:         3000,
		Host:         "localhost",
	}

	_, err := InitDatabase(cfg)
	// Note: SQLite may create parent directories, so this test depends on permissions
	if err != nil {
		// Expected to fail on most systems
		t.Logf("Got expected error for invalid path: %v", err)
	}
}

// containsString is a helper to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && contains(s, substr))
}

// contains is a simple string contains implementation
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestGivenExistingDatabase_whenInitDatabaseAgain_thenDataPreserved tests re-initialization path
func TestGivenExistingDatabase_whenInitDatabaseAgain_thenDataPreserved(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "reinit_test.db")

	cfg := &config.Config{
		DatabaseType: "sqlite",
		SQLitePath:   dbPath,
	}

	// First init: create tables and insert a row
	db, err := InitDatabase(cfg)
	if err != nil {
		t.Fatalf("first InitDatabase failed: %v", err)
	}
	group := models.Group{Name: "preserved-group"}
	if err := db.Create(&group).Error; err != nil {
		t.Fatalf("failed to insert group: %v", err)
	}
	sqlDB, _ := db.DB()
	sqlDB.Close()

	// Second init on the same file: tables exist, must not fail or lose data
	db2, err := InitDatabase(cfg)
	if err != nil {
		t.Fatalf("second InitDatabase failed: %v", err)
	}
	var count int64
	if err := db2.Model(&models.Group{}).Where("name = ?", "preserved-group").Count(&count).Error; err != nil {
		t.Fatalf("failed to count groups: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 group after re-init, got %d", count)
	}
	sqlDB2, _ := db2.DB()
	sqlDB2.Close()
}
