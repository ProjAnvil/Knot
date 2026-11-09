package models

import (
	"encoding/json"
	"regexp"
	"testing"
)

// TestParameterTableName verifies the table name is correctly specified
func TestParameterTableName(t *testing.T) {
	param := Parameter{}
	if param.TableName() != "parameters" {
		t.Errorf("Expected table name 'parameters', got '%s'", param.TableName())
	}
}

// TestParameterFieldVerification verifies all struct fields are present
func TestParameterFieldVerification(t *testing.T) {
	description := "Test description"
	var parentID uint = 100

	param := Parameter{
		ID:          1,
		APIID:       10,
		ParentID:    &parentID,
		Name:        "testParam",
		Type:        "string",
		Description: &description,
		Required:    true,
		ParamType:   "request",
		Order:       5,
		CreatedAt:   1234567890,
		UpdatedAt:   1234567890,
	}

	if param.ID != 1 {
		t.Errorf("Expected ID 1, got %d", param.ID)
	}
	if param.APIID != 10 {
		t.Errorf("Expected APIID 10, got %d", param.APIID)
	}
	if param.ParentID == nil || *param.ParentID != 100 {
		t.Errorf("Expected ParentID 100, got %v", param.ParentID)
	}
	if param.Name != "testParam" {
		t.Errorf("Expected Name 'testParam', got '%s'", param.Name)
	}
	if param.Type != "string" {
		t.Errorf("Expected Type 'string', got '%s'", param.Type)
	}
	if param.Description == nil || *param.Description != "Test description" {
		t.Errorf("Expected Description 'Test description', got %v", param.Description)
	}
	if !param.Required {
		t.Error("Expected Required to be true, got false")
	}
	if param.ParamType != "request" {
		t.Errorf("Expected ParamType 'request', got '%s'", param.ParamType)
	}
	if param.Order != 5 {
		t.Errorf("Expected Order 5, got %d", param.Order)
	}
	if param.CreatedAt != 1234567890 {
		t.Errorf("Expected CreatedAt 1234567890, got %d", param.CreatedAt)
	}
	if param.UpdatedAt != 1234567890 {
		t.Errorf("Expected UpdatedAt 1234567890, got %d", param.UpdatedAt)
	}
}

// TestParameterDefaultValues verifies default field values
func TestParameterDefaultValues(t *testing.T) {
	param := Parameter{}

	// Required should default to false
	if param.Required != false {
		t.Errorf("Expected default Required false, got %v", param.Required)
	}

	// ParentID should be nil when uninitialized
	if param.ParentID != nil {
		t.Errorf("Expected ParentID to be nil, got %v", param.ParentID)
	}

	// Description should be nil when uninitialized
	if param.Description != nil {
		t.Errorf("Expected Description to be nil, got %v", param.Description)
	}

	// Children should be empty when uninitialized
	if param.Children != nil {
		t.Errorf("Expected Children to be nil, got %v", param.Children)
	}
}

// TestParameterJSONSerialization verifies JSON marshaling
func TestParameterJSONSerialization(t *testing.T) {
	description := "Test description"
	param := Parameter{
		ID:          1,
		APIID:       10,
		Name:        "testParam",
		Type:        "string",
		Description: &description,
		Required:    true,
		ParamType:   "request",
		Order:       5,
		CreatedAt:   1609459200, // 2021-01-01 00:00:00 UTC
		UpdatedAt:   1609545600, // 2021-01-02 00:00:00 UTC
	}

	data, err := json.Marshal(param)
	if err != nil {
		t.Fatalf("Failed to marshal parameter to JSON: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Verify fields
	if result["id"].(float64) != 1 {
		t.Errorf("Expected id 1, got %v", result["id"])
	}
	if result["apiId"].(float64) != 10 {
		t.Errorf("Expected apiId 10, got %v", result["apiId"])
	}
	if result["name"] != "testParam" {
		t.Errorf("Expected name 'testParam', got %v", result["name"])
	}
	if result["type"] != "string" {
		t.Errorf("Expected type 'string', got %v", result["type"])
	}
	if result["description"] != "Test description" {
		t.Errorf("Expected description 'Test description', got %v", result["description"])
	}
	if result["required"] != true {
		t.Errorf("Expected required true, got %v", result["required"])
	}
	if result["paramType"] != "request" {
		t.Errorf("Expected paramType 'request', got %v", result["paramType"])
	}
	if result["order"].(float64) != 5 {
		t.Errorf("Expected order 5, got %v", result["order"])
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

// TestParameterJSONSerializationWithNilFields verifies JSON marshaling with nil fields
func TestParameterJSONSerializationWithNilFields(t *testing.T) {
	param := Parameter{
		ID:          1,
		APIID:       10,
		Name:        "testParam",
		Type:        "string",
		ParentID:    nil,
		Description: nil,
		Required:    false,
		ParamType:   "request",
		Order:       1,
	}

	data, err := json.Marshal(param)
	if err != nil {
		t.Fatalf("Failed to marshal parameter to JSON: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// ParentID should be null when nil
	if result["parentId"] != nil {
		t.Errorf("Expected parentId to be null, got %v", result["parentId"])
	}

	// Description should be null when nil
	if result["description"] != nil {
		t.Errorf("Expected description to be null, got %v", result["description"])
	}
}
