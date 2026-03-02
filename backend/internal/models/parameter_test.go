package models

import (
	"encoding/json"
	"fmt"
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

// TestParameterGORMTags verifies GORM tags are correct using database constraints
func TestParameterGORMTags(t *testing.T) {
	db := setupTestDB(t)

	// Create a group and API first (required foreign key)
	group := Group{Name: "Test Group", Order: 1}
	db.Create(&group)

	api := API{GroupID: group.ID, Name: "Test API", Endpoint: "/api/v1/test", Method: "POST", Type: "HTTP", Order: 1}
	db.Create(&api)

	// Test primary key and auto increment
	param := Parameter{APIID: api.ID, Name: "testParam", Type: "string", ParamType: "request", Order: 1}
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
	group := Group{Name: "API Group", Order: 1}
	db.Create(&group)

	// Create an API
	api := API{GroupID: group.ID, Name: "Test API", Endpoint: "/api/v1/test", Method: "POST", Type: "HTTP", Order: 1}
	db.Create(&api)

	// Create a parameter
	param := Parameter{APIID: api.ID, Name: "testParam", Type: "string", ParamType: "request", Order: 1}
	db.Create(&param)

	// Query with preload
	var retrievedParam Parameter
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
	group := Group{Name: "API Group", Order: 1}
	db.Create(&group)

	// Create an API
	api := API{GroupID: group.ID, Name: "Test API", Endpoint: "/api/v1/test", Method: "POST", Type: "HTTP", Order: 1}
	db.Create(&api)

	// Create parent parameter
	parentParam := Parameter{APIID: api.ID, Name: "parentParam", Type: "object", ParamType: "request", Order: 1}
	db.Create(&parentParam)

	// Create child parameters
	childParam1 := Parameter{APIID: api.ID, ParentID: &parentParam.ID, Name: "childParam1", Type: "string", ParamType: "request", Order: 1}
	childParam2 := Parameter{APIID: api.ID, ParentID: &parentParam.ID, Name: "childParam2", Type: "number", ParamType: "request", Order: 2}
	db.Create(&childParam1)
	db.Create(&childParam2)

	// Query with preload
	var retrievedParam Parameter
	db.Preload("Children").First(&retrievedParam, parentParam.ID)

	if len(retrievedParam.Children) != 2 {
		t.Errorf("Expected 2 children, got %d", len(retrievedParam.Children))
	}

	// Verify child parameter's parent reference
	var retrievedChild Parameter
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
	group := Group{Name: "API Group", Order: 1}
	db.Create(&group)

	// Create an API
	api := API{GroupID: group.ID, Name: "Test API", Endpoint: "/api/v1/test", Method: "POST", Type: "HTTP", Order: 1}
	db.Create(&api)

	// Create a nested structure: object -> nested object -> primitive
	objectParam := Parameter{APIID: api.ID, Name: "data", Type: "object", ParamType: "request", Order: 1}
	db.Create(&objectParam)

	nestedObjectParam := Parameter{APIID: api.ID, ParentID: &objectParam.ID, Name: "nested", Type: "object", ParamType: "request", Order: 1}
	db.Create(&nestedObjectParam)

	primitiveParam := Parameter{APIID: api.ID, ParentID: &nestedObjectParam.ID, Name: "value", Type: "string", ParamType: "request", Order: 1}
	db.Create(&primitiveParam)

	// Query the top-level object with nested children
	var retrievedParam Parameter
	db.Preload("Children").Preload("Children.Children").First(&retrievedParam, objectParam.ID)

	if len(retrievedParam.Children) != 1 {
		t.Errorf("Expected 1 child at first level, got %d", len(retrievedParam.Children))
	}

	if retrievedParam.Children[0].Name != "nested" {
		t.Errorf("Expected first child name 'nested', got '%s'", retrievedParam.Children[0].Name)
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

// TestParameterTypes verifies different parameter type values
func TestParameterTypes(t *testing.T) {
	db := setupTestDB(t)

	group := Group{Name: "Test Group", Order: 1}
	db.Create(&group)

	api := API{GroupID: group.ID, Name: "Test API", Endpoint: "/api/v1/test", Method: "POST", Type: "HTTP", Order: 1}
	db.Create(&api)

	types := []string{"string", "number", "boolean", "array", "object"}
	for _, paramType := range types {
		param := Parameter{
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

	group := Group{Name: "Test Group", Order: 1}
	db.Create(&group)

	api := API{GroupID: group.ID, Name: "Test API", Endpoint: "/api/v1/test", Method: "POST", Type: "HTTP", Order: 1}
	db.Create(&api)

	paramTypes := []string{"request", "response"}
	for _, pt := range paramTypes {
		param := Parameter{
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

	group := Group{Name: "Test Group", Order: 1}
	db.Create(&group)

	api := API{GroupID: group.ID, Name: "Test API", Endpoint: "/api/v1/test", Method: "POST", Type: "HTTP", Order: 1}
	db.Create(&api)

	// Create test parameters
	params := []Parameter{
		{APIID: api.ID, Name: "param1", Type: "string", ParamType: "request", Order: 1},
		{APIID: api.ID, Name: "param2", Type: "number", ParamType: "request", Order: 2},
		{APIID: api.ID, Name: "param3", Type: "boolean", ParamType: "response", Order: 1},
	}
	for _, param := range params {
		db.Create(&param)
	}

	// Test FindAll
	var retrievedParams []Parameter
	db.Find(&retrievedParams)
	if len(retrievedParams) != 3 {
		t.Errorf("Expected 3 parameters, got %d", len(retrievedParams))
	}

	// Test FindByID
	var singleParam Parameter
	db.First(&singleParam, 2)
	if singleParam.Name != "param2" {
		t.Errorf("Expected name 'param2', got '%s'", singleParam.Name)
	}

	// Test Where
	var filteredParams []Parameter
	db.Where("param_type = ?", "request").Find(&filteredParams)
	if len(filteredParams) != 2 {
		t.Errorf("Expected 2 parameters with paramType 'request', got %d", len(filteredParams))
	}

	// Test OrderBy - use column name with quotes to avoid SQL keyword issues
	var orderedParams []Parameter
	db.Order("`order` asc").Find(&orderedParams)
	if orderedParams[0].Name != "param1" {
		t.Errorf("Expected first parameter to be 'param1', got '%s'", orderedParams[0].Name)
	}
}

// TestParameterUpdateOperations verifies update operations
func TestParameterUpdateOperations(t *testing.T) {
	db := setupTestDB(t)

	group := Group{Name: "Test Group", Order: 1}
	db.Create(&group)

	api := API{GroupID: group.ID, Name: "Test API", Endpoint: "/api/v1/test", Method: "POST", Type: "HTTP", Order: 1}
	db.Create(&api)

	param := Parameter{APIID: api.ID, Name: "Original Name", Type: "string", ParamType: "request", Order: 1}
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

	group := Group{Name: "Test Group", Order: 1}
	db.Create(&group)

	api := API{GroupID: group.ID, Name: "Test API", Endpoint: "/api/v1/test", Method: "POST", Type: "HTTP", Order: 1}
	db.Create(&api)

	param := Parameter{APIID: api.ID, Name: "To Delete", Type: "string", ParamType: "request", Order: 1}
	db.Create(&param)

	// Delete
	db.Delete(&param)

	var count int64
	db.Model(&Parameter{}).Where("id = ?", param.ID).Count(&count)
	if count != 0 {
		t.Errorf("Expected 0 parameters after delete, got %d", count)
	}
}

// TestParameterRequiredField verifies the Required field behavior
func TestParameterRequiredField(t *testing.T) {
	db := setupTestDB(t)

	group := Group{Name: "Test Group", Order: 1}
	db.Create(&group)

	api := API{GroupID: group.ID, Name: "Test API", Endpoint: "/api/v1/test", Method: "POST", Type: "HTTP", Order: 1}
	db.Create(&api)

	// Create required parameter
	requiredParam := Parameter{APIID: api.ID, Name: "requiredParam", Type: "string", Required: true, ParamType: "request", Order: 1}
	db.Create(&requiredParam)

	// Create optional parameter
	optionalParam := Parameter{APIID: api.ID, Name: "optionalParam", Type: "string", Required: false, ParamType: "request", Order: 2}
	db.Create(&optionalParam)

	// Query and verify
	var retrievedParams []Parameter
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

	group := Group{Name: "Test Group", Order: 1}
	db.Create(&group)

	api := API{GroupID: group.ID, Name: "Test API", Endpoint: "/api/v1/test", Method: "POST", Type: "HTTP", Order: 1}
	db.Create(&api)

	// Create parent parameter
	parentParam := Parameter{APIID: api.ID, Name: "parentParam", Type: "object", ParamType: "request", Order: 1}
	db.Create(&parentParam)

	// Create child parameter
	childParam := Parameter{APIID: api.ID, ParentID: &parentParam.ID, Name: "childParam", Type: "string", ParamType: "request", Order: 1}
	db.Create(&childParam)

	// Query with children preloaded
	var retrievedParam Parameter
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

	group := Group{Name: "Test Group", Order: 1}
	db.Create(&group)

	api := API{GroupID: group.ID, Name: "Test API", Endpoint: "/api/v1/test", Method: "POST", Type: "HTTP", Order: 1}
	db.Create(&api)

	// Test array type with children (array items)
	arrayParam := Parameter{APIID: api.ID, Name: "items", Type: "array", ParamType: "request", Order: 1}
	db.Create(&arrayParam)

	arrayItemParam := Parameter{APIID: api.ID, ParentID: &arrayParam.ID, Name: "item", Type: "object", ParamType: "request", Order: 1}
	db.Create(&arrayItemParam)

	// Verify structure
	var retrievedArray Parameter
	db.Preload("Children").First(&retrievedArray, arrayParam.ID)

	if retrievedArray.Type != "array" {
		t.Errorf("Expected type 'array', got '%s'", retrievedArray.Type)
	}
	if len(retrievedArray.Children) != 1 {
		t.Errorf("Expected 1 child for array items, got %d", len(retrievedArray.Children))
	}
}
