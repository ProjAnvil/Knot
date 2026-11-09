package models

import (
	"encoding/json"
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
		ID:       1,
		GroupID:  10,
		Name:     "Test API",
		Endpoint: "/api/v1/test",
		Method:   "GET",
		Type:     "HTTP",
		Order:    1,
		Note:     nil,
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
