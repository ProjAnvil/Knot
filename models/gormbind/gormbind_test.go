package gormbind

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	models "github.com/ProjAnvil/knot/models"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// setupTestDB creates an in-memory SQLite database for testing.
// Every test gets a brand-new database instance, so tests cannot pollute
// each other's state.
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Enable foreign key constraints for SQLite
	db.Exec("PRAGMA foreign_keys = ON")

	// Migrate all models through the package's own canonical list
	if err := AutoMigrate(db); err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	return db
}

// TestMigrateModels verifies the canonical migration list: exact length,
// exact order (Group, API, Parameter) and pointer content.
func TestMigrateModels(t *testing.T) {
	got := MigrateModels()

	want := []any{&models.Group{}, &models.API{}, &models.Parameter{}}
	if len(got) != len(want) {
		t.Fatalf("Expected %d migrate models, got %d", len(want), len(got))
	}

	for i, w := range want {
		if got[i] == nil {
			t.Fatalf("MigrateModels()[%d] is nil, expected %T", i, w)
		}
		if reflect.TypeOf(got[i]) != reflect.TypeOf(w) {
			t.Errorf("MigrateModels()[%d] = %T, expected %T", i, got[i], w)
		}
	}
}

// TestAutoMigrate verifies AutoMigrate succeeds on a fresh in-memory
// database, creates all three tables, and is idempotent when run twice.
func TestAutoMigrate(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// First run must succeed
	if err := AutoMigrate(db); err != nil {
		t.Fatalf("AutoMigrate failed: %v", err)
	}

	// All model tables must exist afterwards
	for _, model := range MigrateModels() {
		if !db.Migrator().HasTable(model) {
			t.Errorf("Expected table for %T to exist after AutoMigrate", model)
		}
	}

	// Second run over the already-migrated schema must succeed (idempotent)
	if err := AutoMigrate(db); err != nil {
		t.Fatalf("Second AutoMigrate failed (expected idempotent): %v", err)
	}

	// A closed connection must surface an error instead of a silent success
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("Failed to get underlying sql.DB: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("Failed to close test database: %v", err)
	}
	if err := AutoMigrate(db); err == nil {
		t.Error("Expected error from AutoMigrate on a closed connection, got nil")
	}
}

// ---------------------------------------------------------------------------
// Behavioral tests re-homed from the former backend-internal model tests
// (rescued in the models-library Task 1 extraction). They exercise GORM
// tags, unique-name constraints, cascades, indexes and query operations
// against a real SQLite database.
// ---------------------------------------------------------------------------

// TestAPIGORMTags verifies GORM tags are correct using database constraints
func TestAPIGORMTags(t *testing.T) {
	db := setupTestDB(t)

	// Create a group first (required foreign key)
	group := models.Group{Name: "Test Group", Order: 1}
	db.Create(&group)

	// Test primary key and auto increment
	api := models.API{GroupID: group.ID, Name: "Test API", Endpoint: "/api/v1/test", Method: "GET", Type: "HTTP", Order: 1}
	result := db.Create(&api)
	if result.Error != nil {
		t.Fatalf("Failed to create API: %v", result.Error)
	}
	if api.ID == 0 {
		t.Error("Expected ID to be auto-incremented, got 0")
	}

	// Test foreign key constraint - invalid GroupID should fail
	// Note: GORM with SQLite may not enforce foreign keys by default
	// The important thing is that the model structure is correct
}

// TestAPIBelongsToGroup verifies the many-to-one relationship with Group
func TestAPIBelongsToGroup(t *testing.T) {
	db := setupTestDB(t)

	// Create a group
	group := models.Group{Name: "API Group", Order: 1}
	db.Create(&group)

	// Create associated API
	api := models.API{GroupID: group.ID, Name: "Test API", Endpoint: "/api/v1/test", Method: "GET", Type: "HTTP", Order: 1}
	db.Create(&api)

	// Query with preload
	var retrievedAPI models.API
	db.Preload("Group").First(&retrievedAPI, api.ID)

	if retrievedAPI.Group == nil {
		t.Fatal("Expected Group to be preloaded, got nil")
	}
	if retrievedAPI.Group.ID != group.ID {
		t.Errorf("Expected Group ID %d, got %d", group.ID, retrievedAPI.Group.ID)
	}
	if retrievedAPI.Group.Name != "API Group" {
		t.Errorf("Expected Group name 'API Group', got '%s'", retrievedAPI.Group.Name)
	}
}

// TestAPIHasManyParameters verifies the one-to-many relationship with Parameters
func TestAPIHasManyParameters(t *testing.T) {
	db := setupTestDB(t)

	// Create a group
	group := models.Group{Name: "API Group", Order: 1}
	db.Create(&group)

	// Create an API
	api := models.API{GroupID: group.ID, Name: "Test API", Endpoint: "/api/v1/test", Method: "POST", Type: "HTTP", Order: 1}
	db.Create(&api)

	// Create associated parameters
	parameters := []models.Parameter{
		{APIID: api.ID, Name: "param1", Type: "string", ParamType: "request", Order: 1},
		{APIID: api.ID, Name: "param2", Type: "number", ParamType: "request", Order: 2},
		{APIID: api.ID, Name: "response1", Type: "string", ParamType: "response", Order: 1},
	}
	for _, param := range parameters {
		db.Create(&param)
	}

	// Query with preload
	var retrievedAPI models.API
	db.Preload("Parameters").First(&retrievedAPI, api.ID)

	if len(retrievedAPI.Parameters) != 3 {
		t.Errorf("Expected 3 parameters, got %d", len(retrievedAPI.Parameters))
	}

	// Verify cascade delete
	db.Delete(&api)

	var paramCount int64
	db.Model(&models.Parameter{}).Where("api_id = ?", api.ID).Count(&paramCount)
	if paramCount != 0 {
		t.Errorf("Expected 0 parameters after cascade delete, got %d", paramCount)
	}
}

// TestAPIHTTPMethods verifies different HTTP method values
func TestAPIHTTPMethods(t *testing.T) {
	db := setupTestDB(t)

	group := models.Group{Name: "Test Group", Order: 1}
	db.Create(&group)

	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", ""}
	for _, method := range methods {
		api := models.API{
			GroupID:  group.ID,
			Name:     fmt.Sprintf("API %s", method),
			Endpoint: fmt.Sprintf("/api/v1/%d", len(method)),
			Method:   method,
			Type:     "HTTP",
			Order:    1,
		}
		result := db.Create(&api)
		if result.Error != nil {
			t.Errorf("Failed to create API with method '%s': %v", method, result.Error)
		}
	}
}

// TestAPITypes verifies different API type values
func TestAPITypes(t *testing.T) {
	db := setupTestDB(t)

	group := models.Group{Name: "Test Group", Order: 1}
	db.Create(&group)

	types := []string{"HTTP", "RPC", "GraphQL"}
	for _, apiType := range types {
		api := models.API{
			GroupID:  group.ID,
			Name:     fmt.Sprintf("API %s", apiType),
			Endpoint: fmt.Sprintf("/api/v1/%s", apiType),
			Method:   "POST",
			Type:     apiType,
			Order:    1,
		}
		result := db.Create(&api)
		if result.Error != nil {
			t.Errorf("Failed to create API with type '%s': %v", apiType, result.Error)
		}
	}
}

// TestAPIQueryOperations verifies basic database query operations
func TestAPIQueryOperations(t *testing.T) {
	db := setupTestDB(t)

	group := models.Group{Name: "Test Group", Order: 1}
	db.Create(&group)

	// Create test APIs
	apis := []models.API{
		{GroupID: group.ID, Name: "API A", Endpoint: "/api/v1/a", Method: "GET", Type: "HTTP", Order: 1},
		{GroupID: group.ID, Name: "API B", Endpoint: "/api/v1/b", Method: "POST", Type: "HTTP", Order: 2},
		{GroupID: group.ID, Name: "API C", Endpoint: "/api/v1/c", Method: "PUT", Type: "HTTP", Order: 3},
	}
	for _, api := range apis {
		db.Create(&api)
	}

	// Test FindAll
	var retrievedAPIs []models.API
	db.Find(&retrievedAPIs)
	if len(retrievedAPIs) != 3 {
		t.Errorf("Expected 3 APIs, got %d", len(retrievedAPIs))
	}

	// Test FindByID
	var singleAPI models.API
	db.First(&singleAPI, 2)
	if singleAPI.Name != "API B" {
		t.Errorf("Expected name 'API B', got '%s'", singleAPI.Name)
	}

	// Test Where
	var filteredAPIs []models.API
	db.Where("method = ?", "GET").Find(&filteredAPIs)
	if len(filteredAPIs) != 1 {
		t.Errorf("Expected 1 API with method GET, got %d", len(filteredAPIs))
	}

	// Test OrderBy - use column name with quotes to avoid SQL keyword issues
	var orderedAPIs []models.API
	db.Order("`order` asc").Find(&orderedAPIs)
	if orderedAPIs[0].Name != "API A" {
		t.Errorf("Expected first API to be 'API A', got '%s'", orderedAPIs[0].Name)
	}
}

// TestAPIUpdateOperations verifies update operations
func TestAPIUpdateOperations(t *testing.T) {
	db := setupTestDB(t)

	group := models.Group{Name: "Test Group", Order: 1}
	db.Create(&group)

	api := models.API{GroupID: group.ID, Name: "Original Name", Endpoint: "/api/v1/original", Method: "GET", Type: "HTTP", Order: 1}
	db.Create(&api)

	// Update single field
	db.Model(&api).Update("Name", "Updated Name")
	db.First(&api, api.ID)
	if api.Name != "Updated Name" {
		t.Errorf("Expected name 'Updated Name', got '%s'", api.Name)
	}

	// Update multiple fields
	newNote := "Updated note"
	db.Model(&api).Updates(map[string]interface{}{"Name": "New Name", "Endpoint": "/api/v1/new", "Note": &newNote})
	db.First(&api, api.ID)
	if api.Name != "New Name" || api.Endpoint != "/api/v1/new" {
		t.Errorf("Expected name 'New Name' and endpoint '/api/v1/new', got '%s' and '%s'", api.Name, api.Endpoint)
	}
}

// TestAPIDeleteOperations verifies delete operations
func TestAPIDeleteOperations(t *testing.T) {
	db := setupTestDB(t)

	group := models.Group{Name: "Test Group", Order: 1}
	db.Create(&group)

	api := models.API{GroupID: group.ID, Name: "To Delete", Endpoint: "/api/v1/delete", Method: "DELETE", Type: "HTTP", Order: 1}
	db.Create(&api)

	// Delete
	db.Delete(&api)

	var count int64
	db.Model(&models.API{}).Where("id = ?", api.ID).Count(&count)
	if count != 0 {
		t.Errorf("Expected 0 APIs after delete, got %d", count)
	}
}

// TestAPIWithAllHTTPMethods verifies all standard HTTP methods
func TestAPIWithAllHTTPMethods(t *testing.T) {
	db := setupTestDB(t)

	group := models.Group{Name: "Test Group", Order: 1}
	db.Create(&group)

	httpMethods := []struct {
		method string
		valid  bool
	}{
		{"GET", true},
		{"POST", true},
		{"PUT", true},
		{"DELETE", true},
		{"PATCH", true},
		{"HEAD", true},
		{"OPTIONS", true},
		{"TRACE", true},
		{"CONNECT", true},
		{"INVALID", true}, // GORM doesn't validate enum values by default
	}

	for _, hm := range httpMethods {
		api := models.API{
			GroupID:  group.ID,
			Name:     hm.method + " API",
			Endpoint: "/api/v1/test",
			Method:   hm.method,
			Type:     "HTTP",
			Order:    1,
		}
		result := db.Create(&api)
		if hm.valid && result.Error != nil {
			t.Errorf("Failed to create API with method '%s': %v", hm.method, result.Error)
		}
	}
}

// TestGroupGORMTags verifies GORM tags are correct using database constraints
func TestGroupGORMTags(t *testing.T) {
	db := setupTestDB(t)

	// Test primary key and auto increment
	group := models.Group{Name: "Test Group", Order: 1}
	result := db.Create(&group)
	if result.Error != nil {
		t.Fatalf("Failed to create group: %v", result.Error)
	}
	if group.ID == 0 {
		t.Error("Expected ID to be auto-incremented, got 0")
	}

	// Test unique constraint on Name
	duplicateGroup := models.Group{Name: "Test Group", Order: 2}
	result = db.Create(&duplicateGroup)
	if result.Error == nil {
		t.Error("Expected error for duplicate group name, got nil")
	}

	// Note: SQLite doesn't enforce empty string checks at the database level
	// unless a CHECK constraint is added. GORM doesn't add CHECK constraints automatically.
	// We verify that the unique constraint works, which is the main requirement.
}

// TestGroupHasManyAPIs verifies the one-to-many relationship with APIs
func TestGroupHasManyAPIs(t *testing.T) {
	db := setupTestDB(t)

	// Create a group
	group := models.Group{Name: "API Group", Order: 1}
	db.Create(&group)

	// Create associated APIs
	apis := []models.API{
		{GroupID: group.ID, Name: "API 1", Endpoint: "/api/v1/test1", Method: "GET", Type: "HTTP", Order: 1},
		{GroupID: group.ID, Name: "API 2", Endpoint: "/api/v1/test2", Method: "POST", Type: "HTTP", Order: 2},
	}
	for _, api := range apis {
		db.Create(&api)
	}

	// Query with preload
	var retrievedGroup models.Group
	db.Preload("APIs").First(&retrievedGroup, group.ID)

	if len(retrievedGroup.APIs) != 2 {
		t.Errorf("Expected 2 APIs, got %d", len(retrievedGroup.APIs))
	}

	// Verify cascade delete - GORM should handle this through the constraint tag
	db.Delete(&group)

	var apiCount int64
	db.Model(&models.API{}).Where("group_id = ?", group.ID).Count(&apiCount)
	if apiCount != 0 {
		t.Errorf("Expected 0 APIs after cascade delete, got %d", apiCount)
	}
}

// TestGroupQueryOperations verifies basic database query operations
func TestGroupQueryOperations(t *testing.T) {
	db := setupTestDB(t)

	// Create test groups
	groups := []models.Group{
		{Name: "Group A", Order: 1},
		{Name: "Group B", Order: 2},
		{Name: "Group C", Order: 3},
	}
	for _, group := range groups {
		db.Create(&group)
	}

	// Test FindAll
	var retrievedGroups []models.Group
	db.Find(&retrievedGroups)
	if len(retrievedGroups) != 3 {
		t.Errorf("Expected 3 groups, got %d", len(retrievedGroups))
	}

	// Test FindByID
	var singleGroup models.Group
	db.First(&singleGroup, 2)
	if singleGroup.Name != "Group B" {
		t.Errorf("Expected name 'Group B', got '%s'", singleGroup.Name)
	}

	// Test Where - use column name with quotes to avoid SQL keyword issues
	var filteredGroups []models.Group
	db.Where("`order` > ?", 1).Find(&filteredGroups)
	if len(filteredGroups) != 2 {
		t.Errorf("Expected 2 groups with order > 1, got %d", len(filteredGroups))
	}

	// Test OrderBy
	var orderedGroups []models.Group
	db.Order("`order` asc").Find(&orderedGroups)
	if orderedGroups[0].Name != "Group A" {
		t.Errorf("Expected first group to be 'Group A', got '%s'", orderedGroups[0].Name)
	}
}

// TestGroupUpdateOperations verifies update operations
func TestGroupUpdateOperations(t *testing.T) {
	db := setupTestDB(t)

	group := models.Group{Name: "Original Name", Order: 1}
	db.Create(&group)

	// Update single field
	db.Model(&group).Update("Name", "Updated Name")
	db.First(&group, group.ID)
	if group.Name != "Updated Name" {
		t.Errorf("Expected name 'Updated Name', got '%s'", group.Name)
	}

	// Update multiple fields
	db.Model(&group).Updates(map[string]interface{}{"Name": "New Name", "Order": 10})
	db.First(&group, group.ID)
	if group.Name != "New Name" || group.Order != 10 {
		t.Errorf("Expected name 'New Name' and order 10, got '%s' and %d", group.Name, group.Order)
	}
}

// TestGroupDeleteOperations verifies delete operations
func TestGroupDeleteOperations(t *testing.T) {
	db := setupTestDB(t)

	group := models.Group{Name: "To Delete", Order: 1}
	db.Create(&group)

	// Delete
	db.Delete(&group)

	var count int64
	db.Model(&models.Group{}).Where("id = ?", group.ID).Count(&count)
	if count != 0 {
		t.Errorf("Expected 0 groups after delete, got %d", count)
	}
}

// TestParameterGORMTags verifies GORM tags are correct using database constraints
func TestParameterGORMTags(t *testing.T) {
	db := setupTestDB(t)

	// Create a group and API first (required foreign key)
	group := models.Group{Name: "Test Group", Order: 1}
	db.Create(&group)

	api := models.API{GroupID: group.ID, Name: "Test API", Endpoint: "/api/v1/test", Method: "POST", Type: "HTTP", Order: 1}
	db.Create(&api)

	// Test primary key and auto increment
	param := models.Parameter{APIID: api.ID, Name: "testParam", Type: "string", ParamType: "request", Order: 1}
	result := db.Create(&param)
	if result.Error != nil {
		t.Fatalf("Failed to create parameter: %v", result.Error)
	}
	if param.ID == 0 {
		t.Error("Expected ID to be auto-incremented, got 0")
	}

	// Test foreign key constraint - invalid APIID should fail
	// Note: GORM with SQLite may not enforce foreign keys by default
	// The important thing is that the model structure is correct
}

// TestParameterBelongsToAPI verifies the many-to-one relationship with API
func TestParameterBelongsToAPI(t *testing.T) {
	db := setupTestDB(t)

	// Create a group
	group := models.Group{Name: "API Group", Order: 1}
	db.Create(&group)

	// Create an API
	api := models.API{GroupID: group.ID, Name: "Test API", Endpoint: "/api/v1/test", Method: "POST", Type: "HTTP", Order: 1}
	db.Create(&api)

	// Create a parameter
	param := models.Parameter{APIID: api.ID, Name: "testParam", Type: "string", ParamType: "request", Order: 1}
	db.Create(&param)

	// Query with preload
	var retrievedParam models.Parameter
	db.Preload("API").First(&retrievedParam, param.ID)

	if retrievedParam.API == nil {
		t.Fatal("Expected API to be preloaded, got nil")
	}
	if retrievedParam.API.ID != api.ID {
		t.Errorf("Expected API ID %d, got %d", api.ID, retrievedParam.API.ID)
	}
	if retrievedParam.API.Name != "Test API" {
		t.Errorf("Expected API name 'Test API', got '%s'", retrievedParam.API.Name)
	}
}

// TestParameterSelfReference verifies the self-referencing parent-child relationship
func TestParameterSelfReference(t *testing.T) {
	db := setupTestDB(t)

	// Create a group
	group := models.Group{Name: "API Group", Order: 1}
	db.Create(&group)

	// Create an API
	api := models.API{GroupID: group.ID, Name: "Test API", Endpoint: "/api/v1/test", Method: "POST", Type: "HTTP", Order: 1}
	db.Create(&api)

	// Create parent parameter
	parentParam := models.Parameter{APIID: api.ID, Name: "parentParam", Type: "object", ParamType: "request", Order: 1}
	db.Create(&parentParam)

	// Create child parameters
	childParam1 := models.Parameter{APIID: api.ID, ParentID: &parentParam.ID, Name: "childParam1", Type: "string", ParamType: "request", Order: 1}
	childParam2 := models.Parameter{APIID: api.ID, ParentID: &parentParam.ID, Name: "childParam2", Type: "number", ParamType: "request", Order: 2}
	db.Create(&childParam1)
	db.Create(&childParam2)

	// Query with preload
	var retrievedParam models.Parameter
	db.Preload("Children").First(&retrievedParam, parentParam.ID)

	if len(retrievedParam.Children) != 2 {
		t.Errorf("Expected 2 children, got %d", len(retrievedParam.Children))
	}

	// Verify child parameter's parent reference
	var retrievedChild models.Parameter
	db.Preload("Parent").First(&retrievedChild, childParam1.ID)
	if retrievedChild.Parent == nil {
		t.Fatal("Expected Parent to be preloaded, got nil")
	}
	if retrievedChild.Parent.ID != parentParam.ID {
		t.Errorf("Expected Parent ID %d, got %d", parentParam.ID, retrievedChild.Parent.ID)
	}
}

// TestParameterNestedStructure verifies nested parameter structures
func TestParameterNestedStructure(t *testing.T) {
	db := setupTestDB(t)

	// Create a group
	group := models.Group{Name: "API Group", Order: 1}
	db.Create(&group)

	// Create an API
	api := models.API{GroupID: group.ID, Name: "Test API", Endpoint: "/api/v1/test", Method: "POST", Type: "HTTP", Order: 1}
	db.Create(&api)

	// Create a nested structure: object -> nested object -> primitive
	objectParam := models.Parameter{APIID: api.ID, Name: "data", Type: "object", ParamType: "request", Order: 1}
	db.Create(&objectParam)

	nestedObjectParam := models.Parameter{APIID: api.ID, ParentID: &objectParam.ID, Name: "nested", Type: "object", ParamType: "request", Order: 1}
	db.Create(&nestedObjectParam)

	primitiveParam := models.Parameter{APIID: api.ID, ParentID: &nestedObjectParam.ID, Name: "value", Type: "string", ParamType: "request", Order: 1}
	db.Create(&primitiveParam)

	// Query the top-level object with nested children
	var retrievedParam models.Parameter
	db.Preload("Children").Preload("Children.Children").First(&retrievedParam, objectParam.ID)

	if len(retrievedParam.Children) != 1 {
		t.Errorf("Expected 1 child at first level, got %d", len(retrievedParam.Children))
	}

	if retrievedParam.Children[0].Name != "nested" {
		t.Errorf("Expected first child name 'nested', got '%s'", retrievedParam.Children[0].Name)
	}
}

// TestParameterTypes verifies different parameter type values
func TestParameterTypes(t *testing.T) {
	db := setupTestDB(t)

	group := models.Group{Name: "Test Group", Order: 1}
	db.Create(&group)

	api := models.API{GroupID: group.ID, Name: "Test API", Endpoint: "/api/v1/test", Method: "POST", Type: "HTTP", Order: 1}
	db.Create(&api)

	types := []string{"string", "number", "boolean", "array", "object"}
	for _, paramType := range types {
		param := models.Parameter{
			APIID:     api.ID,
			Name:      fmt.Sprintf("%sParam", paramType),
			Type:      paramType,
			ParamType: "request",
			Order:     1,
		}
		result := db.Create(&param)
		if result.Error != nil {
			t.Errorf("Failed to create parameter with type '%s': %v", paramType, result.Error)
		}
	}
}

// TestParameterParamTypes verifies request and response parameter types
func TestParameterParamTypes(t *testing.T) {
	db := setupTestDB(t)

	group := models.Group{Name: "Test Group", Order: 1}
	db.Create(&group)

	api := models.API{GroupID: group.ID, Name: "Test API", Endpoint: "/api/v1/test", Method: "POST", Type: "HTTP", Order: 1}
	db.Create(&api)

	paramTypes := []string{"request", "response"}
	for _, pt := range paramTypes {
		param := models.Parameter{
			APIID:     api.ID,
			Name:      fmt.Sprintf("%sParam", pt),
			Type:      "string",
			ParamType: pt,
			Order:     1,
		}
		result := db.Create(&param)
		if result.Error != nil {
			t.Errorf("Failed to create parameter with paramType '%s': %v", pt, result.Error)
		}
	}
}

// TestParameterQueryOperations verifies basic database query operations
func TestParameterQueryOperations(t *testing.T) {
	db := setupTestDB(t)

	group := models.Group{Name: "Test Group", Order: 1}
	db.Create(&group)

	api := models.API{GroupID: group.ID, Name: "Test API", Endpoint: "/api/v1/test", Method: "POST", Type: "HTTP", Order: 1}
	db.Create(&api)

	// Create test parameters
	params := []models.Parameter{
		{APIID: api.ID, Name: "param1", Type: "string", ParamType: "request", Order: 1},
		{APIID: api.ID, Name: "param2", Type: "number", ParamType: "request", Order: 2},
		{APIID: api.ID, Name: "param3", Type: "boolean", ParamType: "response", Order: 1},
	}
	for _, param := range params {
		db.Create(&param)
	}

	// Test FindAll
	var retrievedParams []models.Parameter
	db.Find(&retrievedParams)
	if len(retrievedParams) != 3 {
		t.Errorf("Expected 3 parameters, got %d", len(retrievedParams))
	}

	// Test FindByID
	var singleParam models.Parameter
	db.First(&singleParam, 2)
	if singleParam.Name != "param2" {
		t.Errorf("Expected name 'param2', got '%s'", singleParam.Name)
	}

	// Test Where
	var filteredParams []models.Parameter
	db.Where("param_type = ?", "request").Find(&filteredParams)
	if len(filteredParams) != 2 {
		t.Errorf("Expected 2 parameters with paramType 'request', got %d", len(filteredParams))
	}

	// Test OrderBy - use column name with quotes to avoid SQL keyword issues
	var orderedParams []models.Parameter
	db.Order("`order` asc").Find(&orderedParams)
	if orderedParams[0].Name != "param1" {
		t.Errorf("Expected first parameter to be 'param1', got '%s'", orderedParams[0].Name)
	}
}

// TestParameterUpdateOperations verifies update operations
func TestParameterUpdateOperations(t *testing.T) {
	db := setupTestDB(t)

	group := models.Group{Name: "Test Group", Order: 1}
	db.Create(&group)

	api := models.API{GroupID: group.ID, Name: "Test API", Endpoint: "/api/v1/test", Method: "POST", Type: "HTTP", Order: 1}
	db.Create(&api)

	param := models.Parameter{APIID: api.ID, Name: "Original Name", Type: "string", ParamType: "request", Order: 1}
	db.Create(&param)

	// Update single field
	db.Model(&param).Update("Name", "Updated Name")
	db.First(&param, param.ID)
	if param.Name != "Updated Name" {
		t.Errorf("Expected name 'Updated Name', got '%s'", param.Name)
	}

	// Update multiple fields
	newDescription := "Updated description"
	db.Model(&param).Updates(map[string]interface{}{"Name": "New Name", "Type": "number", "Description": &newDescription})
	db.First(&param, param.ID)
	if param.Name != "New Name" || param.Type != "number" {
		t.Errorf("Expected name 'New Name' and type 'number', got '%s' and '%s'", param.Name, param.Type)
	}
}

// TestParameterDeleteOperations verifies delete operations
func TestParameterDeleteOperations(t *testing.T) {
	db := setupTestDB(t)

	group := models.Group{Name: "Test Group", Order: 1}
	db.Create(&group)

	api := models.API{GroupID: group.ID, Name: "Test API", Endpoint: "/api/v1/test", Method: "POST", Type: "HTTP", Order: 1}
	db.Create(&api)

	param := models.Parameter{APIID: api.ID, Name: "To Delete", Type: "string", ParamType: "request", Order: 1}
	db.Create(&param)

	// Delete
	db.Delete(&param)

	var count int64
	db.Model(&models.Parameter{}).Where("id = ?", param.ID).Count(&count)
	if count != 0 {
		t.Errorf("Expected 0 parameters after delete, got %d", count)
	}
}

// TestParameterRequiredField verifies the Required field behavior
func TestParameterRequiredField(t *testing.T) {
	db := setupTestDB(t)

	group := models.Group{Name: "Test Group", Order: 1}
	db.Create(&group)

	api := models.API{GroupID: group.ID, Name: "Test API", Endpoint: "/api/v1/test", Method: "POST", Type: "HTTP", Order: 1}
	db.Create(&api)

	// Create required parameter
	requiredParam := models.Parameter{APIID: api.ID, Name: "requiredParam", Type: "string", Required: true, ParamType: "request", Order: 1}
	db.Create(&requiredParam)

	// Create optional parameter
	optionalParam := models.Parameter{APIID: api.ID, Name: "optionalParam", Type: "string", Required: false, ParamType: "request", Order: 2}
	db.Create(&optionalParam)

	// Query and verify
	var retrievedParams []models.Parameter
	db.Where("required = ?", true).Find(&retrievedParams)
	if len(retrievedParams) != 1 {
		t.Errorf("Expected 1 required parameter, got %d", len(retrievedParams))
	}
	if retrievedParams[0].Name != "requiredParam" {
		t.Errorf("Expected name 'requiredParam', got '%s'", retrievedParams[0].Name)
	}
}

// TestParameterWithChildrenSerialization verifies JSON serialization with children
func TestParameterWithChildrenSerialization(t *testing.T) {
	db := setupTestDB(t)

	group := models.Group{Name: "Test Group", Order: 1}
	db.Create(&group)

	api := models.API{GroupID: group.ID, Name: "Test API", Endpoint: "/api/v1/test", Method: "POST", Type: "HTTP", Order: 1}
	db.Create(&api)

	// Create parent parameter
	parentParam := models.Parameter{APIID: api.ID, Name: "parentParam", Type: "object", ParamType: "request", Order: 1}
	db.Create(&parentParam)

	// Create child parameter
	childParam := models.Parameter{APIID: api.ID, ParentID: &parentParam.ID, Name: "childParam", Type: "string", ParamType: "request", Order: 1}
	db.Create(&childParam)

	// Query with children preloaded
	var retrievedParam models.Parameter
	db.Preload("Children").First(&retrievedParam, parentParam.ID)

	data, err := json.Marshal(retrievedParam)
	if err != nil {
		t.Fatalf("Failed to marshal parameter to JSON: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Verify children array exists
	children, ok := result["children"].([]interface{})
	if !ok {
		t.Fatal("Expected children to be an array")
	}
	if len(children) != 1 {
		t.Errorf("Expected 1 child, got %d", len(children))
	}

	// Verify child properties
	child := children[0].(map[string]interface{})
	if child["name"] != "childParam" {
		t.Errorf("Expected child name 'childParam', got %v", child["name"])
	}
}

// TestParameterArrayAndObjectTypes verifies array and object parameter types with nesting
func TestParameterArrayAndObjectTypes(t *testing.T) {
	db := setupTestDB(t)

	group := models.Group{Name: "Test Group", Order: 1}
	db.Create(&group)

	api := models.API{GroupID: group.ID, Name: "Test API", Endpoint: "/api/v1/test", Method: "POST", Type: "HTTP", Order: 1}
	db.Create(&api)

	// Test array type with children (array items)
	arrayParam := models.Parameter{APIID: api.ID, Name: "items", Type: "array", ParamType: "request", Order: 1}
	db.Create(&arrayParam)

	arrayItemParam := models.Parameter{APIID: api.ID, ParentID: &arrayParam.ID, Name: "item", Type: "object", ParamType: "request", Order: 1}
	db.Create(&arrayItemParam)

	// Verify structure
	var retrievedArray models.Parameter
	db.Preload("Children").First(&retrievedArray, arrayParam.ID)

	if retrievedArray.Type != "array" {
		t.Errorf("Expected type 'array', got '%s'", retrievedArray.Type)
	}
	if len(retrievedArray.Children) != 1 {
		t.Errorf("Expected 1 child for array items, got %d", len(retrievedArray.Children))
	}
}
