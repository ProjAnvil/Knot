package models

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"
)

// TestGroupTableName verifies the table name is correctly specified
func TestGroupTableName(t *testing.T) {
	group := Group{}
	if group.TableName() != "groups" {
		t.Errorf("Expected table name 'groups', got '%s'", group.TableName())
	}
}

// TestGroupFieldVerification verifies all struct fields are present
func TestGroupFieldVerification(t *testing.T) {
	group := Group{
		ID:        1,
		Name:      "Test Group",
		Order:     10,
		CreatedAt: 1234567890,
		UpdatedAt: 1234567890,
	}

	if group.ID != 1 {
		t.Errorf("Expected ID 1, got %d", group.ID)
	}
	if group.Name != "Test Group" {
		t.Errorf("Expected Name 'Test Group', got '%s'", group.Name)
	}
	if group.Order != 10 {
		t.Errorf("Expected Order 10, got %d", group.Order)
	}
	if group.CreatedAt != 1234567890 {
		t.Errorf("Expected CreatedAt 1234567890, got %d", group.CreatedAt)
	}
	if group.UpdatedAt != 1234567890 {
		t.Errorf("Expected UpdatedAt 1234567890, got %d", group.UpdatedAt)
	}
}

// TestGroupDefaultValues verifies default field values
func TestGroupDefaultValues(t *testing.T) {
	group := Group{}

	// Order should default to 0
	if group.Order != 0 {
		t.Errorf("Expected default Order 0, got %d", group.Order)
	}

	// APIs should be nil when uninitialized
	if group.APIs != nil {
		t.Errorf("Expected APIs to be nil, got %v", group.APIs)
	}
}

// TestGroupJSONSerialization verifies JSON marshaling
// Note: The custom MarshalJSON includes the Order field
func TestGroupJSONSerialization(t *testing.T) {
	group := Group{
		ID:        1,
		Name:      "Test Group",
		Order:     5,
		CreatedAt: 1609459200, // 2021-01-01 00:00:00 UTC
		UpdatedAt: 1609545600, // 2021-01-02 00:00:00 UTC
	}

	data, err := json.Marshal(group)
	if err != nil {
		t.Fatalf("Failed to marshal group to JSON: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Verify fields - handle potential nil values safely
	if id, ok := result["id"].(float64); !ok || id != 1 {
		t.Errorf("Expected id 1, got %v", result["id"])
	}
	if result["name"] != "Test Group" {
		t.Errorf("Expected name 'Test Group', got %v", result["name"])
	}

	// Order is emitted by the custom MarshalJSON
	if order, ok := result["order"].(float64); !ok || order != 5 {
		t.Errorf("Expected order 5, got %v", result["order"])
	}

	// Verify createdAt is in ISO 8601 format
	createdAt, ok := result["createdAt"].(string)
	if !ok {
		t.Fatal("Expected createdAt to be a string")
	}
	iso8601Pattern := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$`)
	if !iso8601Pattern.MatchString(createdAt) {
		t.Errorf("Expected createdAt in ISO 8601 format, got '%s'", createdAt)
	}

	// Verify updatedAt is in ISO 8601 format
	updatedAt, ok := result["updatedAt"].(string)
	if !ok {
		t.Fatal("Expected updatedAt to be a string")
	}
	if !iso8601Pattern.MatchString(updatedAt) {
		t.Errorf("Expected updatedAt in ISO 8601 format, got '%s'", updatedAt)
	}
}

// TestGroupJSONSerializationIncludesOrder verifies the custom MarshalJSON
// emits the Order field and pins the serialized key order:
// name, then order, then apis.
func TestGroupJSONSerializationIncludesOrder(t *testing.T) {
	group := Group{Name: "x", Order: 7}

	data, err := json.Marshal(group)
	if err != nil {
		t.Fatalf("Failed to marshal group to JSON: %v", err)
	}

	raw := string(data)
	if !strings.Contains(raw, `"order":7`) {
		t.Errorf("Expected JSON to contain \"order\":7, got %s", raw)
	}

	// Assert key positions in the raw JSON bytes: name < order < apis
	nameIdx := strings.Index(raw, `"name"`)
	orderIdx := strings.Index(raw, `"order"`)
	apisIdx := strings.Index(raw, `"apis"`)
	if nameIdx == -1 || orderIdx == -1 || apisIdx == -1 {
		t.Fatalf("Expected keys name, order, apis all present, got %s", raw)
	}
	if !(nameIdx < orderIdx && orderIdx < apisIdx) {
		t.Errorf("Expected key order name < order < apis, got indices name=%d, order=%d, apis=%d in %s",
			nameIdx, orderIdx, apisIdx, raw)
	}
}

// TestGroupJSONSerializationWithNilAPIs verifies JSON marshaling when APIs is nil
func TestGroupJSONSerializationWithNilAPIs(t *testing.T) {
	group := Group{
		ID:    1,
		Name:  "Test Group",
		Order: 5,
		APIs:  nil, // Explicitly nil
	}

	data, err := json.Marshal(group)
	if err != nil {
		t.Fatalf("Failed to marshal group to JSON: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// APIs should be an empty array, not null
	apis, ok := result["apis"].([]interface{})
	if !ok {
		t.Error("Expected apis to be an array")
	}
	if apis == nil || len(apis) != 0 {
		t.Errorf("Expected empty array for apis, got %v", apis)
	}
}

// TestGroupJSONSerializationWithEmptyAPIs verifies JSON marshaling when APIs is empty
func TestGroupJSONSerializationWithEmptyAPIs(t *testing.T) {
	group := Group{
		ID:    1,
		Name:  "Test Group",
		Order: 5,
		APIs:  []API{},
	}

	data, err := json.Marshal(group)
	if err != nil {
		t.Fatalf("Failed to marshal group to JSON: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	apis, ok := result["apis"].([]interface{})
	if !ok {
		t.Error("Expected apis to be an array")
	}
	if len(apis) != 0 {
		t.Errorf("Expected empty array for apis, got %d items", len(apis))
	}
}
