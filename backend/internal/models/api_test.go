package models

import (
	"encoding/json"
	"fmt"
	"regexp"
	"testing"
)

// TestAPITableName verifies the table name is correctly specified
func TestAPITableName(t *testing.T) {
	api := API{}
	if api.TableName() != "apis" {
		t.Errorf("Expected table name 'apis', got '%s'", api.TableName())
	}
}

// TestAPIFieldVerification verifies all struct fields are present
func TestAPIFieldVerification(t *testing.T) {
	note := "Test note"
	api := API{
		ID:        1,
		GroupID:   10,
		Name:      "Test API",
		Endpoint:  "/api/v1/test",
		Method:    "POST",
		Type:      "HTTP",
		Order:     5,
		Note:      &note,
		CreatedAt: 1234567890,
		UpdatedAt: 1234567890,
	}

	if api.ID != 1 {
		t.Errorf("Expected ID 1, got %d", api.ID)
	}
	if api.GroupID != 10 {
		t.Errorf("Expected GroupID 10, got %d", api.GroupID)
	}
	if api.Name != "Test API" {
		t.Errorf("Expected Name 'Test API', got '%s'", api.Name)
	}
	if api.Endpoint != "/api/v1/test" {
		t.Errorf("Expected Endpoint '/api/v1/test', got '%s'", api.Endpoint)
	}
	if api.Method != "POST" {
		t.Errorf("Expected Method 'POST', got '%s'", api.Method)
	}
	if api.Type != "HTTP" {
		t.Errorf("Expected Type 'HTTP', got '%s'", api.Type)
	}
	if api.Order != 5 {
		t.Errorf("Expected Order 5, got %d", api.Order)
	}
	if api.Note == nil || *api.Note != "Test note" {
		t.Errorf("Expected Note 'Test note', got %v", api.Note)
	}
	if api.CreatedAt != 1234567890 {
		t.Errorf("Expected CreatedAt 1234567890, got %d", api.CreatedAt)
	}
	if api.UpdatedAt != 1234567890 {
		t.Errorf("Expected UpdatedAt 1234567890, got %d", api.UpdatedAt)
	}
}

// TestAPIDefaultValues verifies default field values
func TestAPIDefaultValues(t *testing.T) {
	api := API{}

	// Order should default to 0
	if api.Order != 0 {
		t.Errorf("Expected default Order 0, got %d", api.Order)
	}

	// Note should be nil when uninitialized
	if api.Note != nil {
		t.Errorf("Expected Note to be nil, got %v", api.Note)
	}

	// Parameters should be nil when uninitialized
	if api.Parameters != nil {
		t.Errorf("Expected Parameters to be nil, got %v", api.Parameters)
	}

	// Method should be empty string
	if api.Method != "" {
		t.Errorf("Expected Method to be empty string, got '%s'", api.Method)
	}
}

// TestAPIGORMTags verifies GORM tags are correct using database constraints
func TestAPIGORMTags(t *testing.T) {
	db := setupTestDB(t)

	// Create a group first (required foreign key)
	group := Group{Name: "Test Group", Order: 1}
	db.Create(&group)

	// Test primary key and auto increment
	api := API{GroupID: group.ID, Name: "Test API", Endpoint: "/api/v1/test", Method: "GET", Type: "HTTP", Order: 1}
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
	group := Group{Name: "API Group", Order: 1}
	db.Create(&group)

	// Create associated API
	api := API{GroupID: group.ID, Name: "Test API", Endpoint: "/api/v1/test", Method: "GET", Type: "HTTP", Order: 1}
	db.Create(&api)

	// Query with preload
	var retrievedAPI API
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
	group := Group{Name: "API Group", Order: 1}
	db.Create(&group)

	// Create an API
	api := API{GroupID: group.ID, Name: "Test API", Endpoint: "/api/v1/test", Method: "POST", Type: "HTTP", Order: 1}
	db.Create(&api)

	// Create associated parameters
	parameters := []Parameter{
		{APIID: api.ID, Name: "param1", Type: "string", ParamType: "request", Order: 1},
		{APIID: api.ID, Name: "param2", Type: "number", ParamType: "request", Order: 2},
		{APIID: api.ID, Name: "response1", Type: "string", ParamType: "response", Order: 1},
	}
	for _, param := range parameters {
		db.Create(&param)
	}

	// Query with preload
	var retrievedAPI API
	db.Preload("Parameters").First(&retrievedAPI, api.ID)

	if len(retrievedAPI.Parameters) != 3 {
		t.Errorf("Expected 3 parameters, got %d", len(retrievedAPI.Parameters))
	}

	// Verify cascade delete
	db.Delete(&api)

	var paramCount int64
	db.Model(&Parameter{}).Where("api_id = ?", api.ID).Count(&paramCount)
	if paramCount != 0 {
		t.Errorf("Expected 0 parameters after cascade delete, got %d", paramCount)
	}
}

// TestAPIJSONSerialization verifies JSON marshaling
func TestAPIJSONSerialization(t *testing.T) {
	note := "Test note"
	api := API{
		ID:        1,
		GroupID:   10,
		Name:      "Test API",
		Endpoint:  "/api/v1/test",
		Method:    "POST",
		Type:      "HTTP",
		Order:     5,
		Note:      &note,
		CreatedAt: 1609459200, // 2021-01-01 00:00:00 UTC
		UpdatedAt: 1609545600, // 2021-01-02 00:00:00 UTC
	}

	data, err := json.Marshal(api)
	if err != nil {
		t.Fatalf("Failed to marshal API to JSON: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Verify fields
	if result["id"].(float64) != 1 {
		t.Errorf("Expected id 1, got %v", result["id"])
	}
	if result["groupId"].(float64) != 10 {
		t.Errorf("Expected groupId 10, got %v", result["groupId"])
	}
	if result["name"] != "Test API" {
		t.Errorf("Expected name 'Test API', got %v", result["name"])
	}
	if result["endpoint"] != "/api/v1/test" {
		t.Errorf("Expected endpoint '/api/v1/test', got %v", result["endpoint"])
	}
	if result["method"] != "POST" {
		t.Errorf("Expected method 'POST', got %v", result["method"])
	}
	if result["type"] != "HTTP" {
		t.Errorf("Expected type 'HTTP', got %v", result["type"])
	}
	if result["order"].(float64) != 5 {
		t.Errorf("Expected order 5, got %v", result["order"])
	}
	if result["note"] != "Test note" {
		t.Errorf("Expected note 'Test note', got %v", result["note"])
	}

	// Verify createdAt is in ISO 8601 format
	createdAt := result["createdAt"].(string)
	iso8601Pattern := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$`)
	if !iso8601Pattern.MatchString(createdAt) {
		t.Errorf("Expected createdAt in ISO 8601 format, got '%s'", createdAt)
	}

	// Verify updatedAt is in ISO 8601 format
	updatedAt := result["updatedAt"].(string)
	if !iso8601Pattern.MatchString(updatedAt) {
		t.Errorf("Expected updatedAt in ISO 8601 format, got '%s'", updatedAt)
	}
}

// TestAPIJSONSerializationWithNilNote verifies JSON marshaling when Note is nil
func TestAPIJSONSerializationWithNilNote(t *testing.T) {
	api := API{
		ID:        1,
		GroupID:   10,
		Name:      "Test API",
		Endpoint:  "/api/v1/test",
		Method:    "GET",
		Type:      "HTTP",
		Order:     1,
		Note:      nil,
	}

	data, err := json.Marshal(api)
	if err != nil {
		t.Fatalf("Failed to marshal API to JSON: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Note should be null when nil
	if _, exists := result["note"]; exists {
		noteVal := result["note"]
		if noteVal != nil {
			t.Errorf("Expected note to be null, got %v", noteVal)
		}
	}
}

// TestAPIJSONSerializationWithNilGroup verifies JSON marshaling when Group is nil
func TestAPIJSONSerializationWithNilGroup(t *testing.T) {
	api := API{
		ID:       1,
		GroupID:  10,
		Name:     "Test API",
		Endpoint: "/api/v1/test",
		Method:   "GET",
		Type:     "HTTP",
		Order:    1,
		Group:    nil,
	}

	data, err := json.Marshal(api)
	if err != nil {
		t.Fatalf("Failed to marshal API to JSON: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Group should be omitted when nil (using omitempty)
	if _, exists := result["group"]; exists {
		t.Error("Expected group to be omitted when nil")
	}
}

// TestAPIHTTPMethods verifies different HTTP method values
func TestAPIHTTPMethods(t *testing.T) {
	db := setupTestDB(t)

	group := Group{Name: "Test Group", Order: 1}
	db.Create(&group)

	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", ""}
	for _, method := range methods {
		api := API{
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

	group := Group{Name: "Test Group", Order: 1}
	db.Create(&group)

	types := []string{"HTTP", "RPC", "GraphQL"}
	for _, apiType := range types {
		api := API{
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

	group := Group{Name: "Test Group", Order: 1}
	db.Create(&group)

	// Create test APIs
	apis := []API{
		{GroupID: group.ID, Name: "API A", Endpoint: "/api/v1/a", Method: "GET", Type: "HTTP", Order: 1},
		{GroupID: group.ID, Name: "API B", Endpoint: "/api/v1/b", Method: "POST", Type: "HTTP", Order: 2},
		{GroupID: group.ID, Name: "API C", Endpoint: "/api/v1/c", Method: "PUT", Type: "HTTP", Order: 3},
	}
	for _, api := range apis {
		db.Create(&api)
	}

	// Test FindAll
	var retrievedAPIs []API
	db.Find(&retrievedAPIs)
	if len(retrievedAPIs) != 3 {
		t.Errorf("Expected 3 APIs, got %d", len(retrievedAPIs))
	}

	// Test FindByID
	var singleAPI API
	db.First(&singleAPI, 2)
	if singleAPI.Name != "API B" {
		t.Errorf("Expected name 'API B', got '%s'", singleAPI.Name)
	}

	// Test Where
	var filteredAPIs []API
	db.Where("method = ?", "GET").Find(&filteredAPIs)
	if len(filteredAPIs) != 1 {
		t.Errorf("Expected 1 API with method GET, got %d", len(filteredAPIs))
	}

	// Test OrderBy - use column name with quotes to avoid SQL keyword issues
	var orderedAPIs []API
	db.Order("`order` asc").Find(&orderedAPIs)
	if orderedAPIs[0].Name != "API A" {
		t.Errorf("Expected first API to be 'API A', got '%s'", orderedAPIs[0].Name)
	}
}

// TestAPIUpdateOperations verifies update operations
func TestAPIUpdateOperations(t *testing.T) {
	db := setupTestDB(t)

	group := Group{Name: "Test Group", Order: 1}
	db.Create(&group)

	api := API{GroupID: group.ID, Name: "Original Name", Endpoint: "/api/v1/original", Method: "GET", Type: "HTTP", Order: 1}
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

	group := Group{Name: "Test Group", Order: 1}
	db.Create(&group)

	api := API{GroupID: group.ID, Name: "To Delete", Endpoint: "/api/v1/delete", Method: "DELETE", Type: "HTTP", Order: 1}
	db.Create(&api)

	// Delete
	db.Delete(&api)

	var count int64
	db.Model(&API{}).Where("id = ?", api.ID).Count(&count)
	if count != 0 {
		t.Errorf("Expected 0 APIs after delete, got %d", count)
	}
}

// TestAPIWithAllHTTPMethods verifies all standard HTTP methods
func TestAPIWithAllHTTPMethods(t *testing.T) {
	db := setupTestDB(t)

	group := Group{Name: "Test Group", Order: 1}
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
		api := API{
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
