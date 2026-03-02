package services

import (
	"encoding/json"
	"testing"

	"github.com/ProjAnvil/knot/backend/internal/models"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Migrate the schema
	err = db.AutoMigrate(&models.Parameter{})
	if err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	return db
}

// TestParameterService_UpdateParameters_GivenValidParameters_whenUpdateParameters_thenParametersSaved
func TestParameterService_UpdateParameters_GivenValidParameters_whenUpdateParameters_thenParametersSaved(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	service := NewParameterService(db)

	apiID := uint(1)
	paramType := "request"

	parameters := json.RawMessage(`[
		{
			"name": "username",
			"type": "string",
			"description": "User's username",
			"required": true
		},
		{
			"name": "password",
			"type": "string",
			"description": "User's password",
			"required": true
		}
	]`)

	// Act
	count, err := service.UpdateParameters(apiID, paramType, parameters)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected 2 parameters inserted, got: %d", count)
	}

	var params []models.Parameter
	db.Find(&params)

	if len(params) != 2 {
		t.Errorf("Expected 2 parameters in database, got: %d", len(params))
	}

	if params[0].Name != "username" || params[0].Type != "string" {
		t.Errorf("First parameter not saved correctly")
	}

	if params[0].Required != true {
		t.Errorf("First parameter required flag not saved correctly")
	}

	if params[0].Description == nil || *params[0].Description != "User's username" {
		t.Errorf("First parameter description not saved correctly")
	}
}

// TestParameterService_UpdateParameters_GivenNestedParameters_whenUpdateParameters_thenNestedParametersSaved
func TestParameterService_UpdateParameters_GivenNestedParameters_whenUpdateParameters_thenNestedParametersSaved(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	service := NewParameterService(db)

	apiID := uint(1)
	paramType := "request"

	parameters := json.RawMessage(`[
		{
			"name": "user",
			"type": "object",
			"description": "User information",
			"required": true,
			"children": [
				{
					"name": "name",
					"type": "string",
					"description": "User's full name",
					"required": true
				},
				{
					"name": "age",
					"type": "number",
					"description": "User's age",
					"required": false
				}
			]
		}
	]`)

	// Act
	count, err := service.UpdateParameters(apiID, paramType, parameters)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if count != 3 {
		t.Errorf("Expected 3 parameters inserted (1 parent + 2 children), got: %d", count)
	}

	var rootParams []models.Parameter
	db.Where("parent_id IS NULL").Find(&rootParams)

	if len(rootParams) != 1 {
		t.Errorf("Expected 1 root parameter, got: %d", len(rootParams))
	}

	if rootParams[0].Name != "user" {
		t.Errorf("Root parameter name should be 'user', got: %s", rootParams[0].Name)
	}

	var childParams []models.Parameter
	db.Where("parent_id = ?", rootParams[0].ID).Find(&childParams)

	if len(childParams) != 2 {
		t.Errorf("Expected 2 child parameters, got: %d", len(childParams))
	}

	names := make(map[string]bool)
	for _, child := range childParams {
		names[child.Name] = true
	}

	if !names["name"] || !names["age"] {
		t.Errorf("Child parameters not saved correctly")
	}
}

// TestParameterService_UpdateParameters_GivenEmptyArray_whenUpdateParameters_thenNoError
func TestParameterService_UpdateParameters_GivenEmptyArray_whenUpdateParameters_thenNoError(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	service := NewParameterService(db)

	apiID := uint(1)
	paramType := "request"

	// First insert some parameters
	parameters := json.RawMessage(`[
		{
			"name": "oldParam",
			"type": "string",
			"description": "Old parameter",
			"required": true
		}
	]`)
	_, _ = service.UpdateParameters(apiID, paramType, parameters)

	// Act - Update with empty array
	emptyParams := json.RawMessage(`[]`)
	count, err := service.UpdateParameters(apiID, paramType, emptyParams)

	// Assert
	if err != nil {
		t.Errorf("Expected no error when updating with empty array, got: %v", err)
	}

	if count != 0 {
		t.Errorf("Expected 0 count when updating with empty array, got: %d", count)
	}

	var params []models.Parameter
	db.Find(&params)

	if len(params) != 0 {
		t.Errorf("Expected 0 parameters in database after empty update, got: %d", len(params))
	}
}

// TestParameterService_UpdateParameters_GivenInvalidJSON_whenUpdateParameters_thenError
func TestParameterService_UpdateParameters_GivenInvalidJSON_whenUpdateParameters_thenError(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	service := NewParameterService(db)

	apiID := uint(1)
	paramType := "request"

	parameters := json.RawMessage(`invalid json`)

	// Act
	count, err := service.UpdateParameters(apiID, paramType, parameters)

	// Assert
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}

	if count != 0 {
		t.Errorf("Expected 0 count for invalid JSON, got: %d", count)
	}
}

// TestParameterService_UpdateParameters_GivenReplaceParameters_whenUpdateParameters_thenOldParametersDeleted
func TestParameterService_UpdateParameters_GivenReplaceParameters_whenUpdateParameters_thenOldParametersDeleted(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	service := NewParameterService(db)

	apiID := uint(1)
	paramType := "request"

	// First insert some parameters
	parameters := json.RawMessage(`[
		{
			"name": "oldParam",
			"type": "string",
			"description": "Old parameter",
			"required": true
		},
		{
			"name": "anotherOld",
			"type": "number",
			"description": "Another old parameter",
			"required": false
		}
	]`)
	_, _ = service.UpdateParameters(apiID, paramType, parameters)

	// Act - Replace with new parameters
	newParameters := json.RawMessage(`[
		{
			"name": "newParam",
			"type": "string",
			"description": "New parameter",
			"required": true
		}
	]`)
	count, err := service.UpdateParameters(apiID, paramType, newParameters)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 parameter, got: %d", count)
	}

	var params []models.Parameter
	db.Find(&params)

	if len(params) != 1 {
		t.Errorf("Expected 1 parameter in database after replacement, got: %d", len(params))
	}

	if params[0].Name != "newParam" {
		t.Errorf("Expected 'newParam', got: %s", params[0].Name)
	}
}

// TestParameterService_UpdateParameters_GivenDifferentParamTypes_whenUpdateParameters_thenEachTypeIndependent
func TestParameterService_UpdateParameters_GivenDifferentParamTypes_whenUpdateParameters_thenEachTypeIndependent(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	service := NewParameterService(db)

	apiID := uint(1)

	// Insert request parameters
	requestParams := json.RawMessage(`[
		{
			"name": "requestParam",
			"type": "string",
			"description": "Request parameter",
			"required": true
		}
	]`)
	_, _ = service.UpdateParameters(apiID, "request", requestParams)

	// Insert response parameters
	responseParams := json.RawMessage(`[
		{
			"name": "responseParam",
			"type": "string",
			"description": "Response parameter",
			"required": false
		}
	]`)
	_, _ = service.UpdateParameters(apiID, "response", responseParams)

	// Act - Update only request parameters
	newRequestParams := json.RawMessage(`[
		{
			"name": "newRequestParam",
			"type": "number",
			"description": "New request parameter",
			"required": true
		}
	]`)
	_, _ = service.UpdateParameters(apiID, "request", newRequestParams)

	// Assert - Response parameters should still exist
	var responseParamsResult []models.Parameter
	db.Where("param_type = ?", "response").Find(&responseParamsResult)

	if len(responseParamsResult) != 1 {
		t.Errorf("Expected 1 response parameter, got: %d", len(responseParamsResult))
	}

	if responseParamsResult[0].Name != "responseParam" {
		t.Errorf("Response parameter should not be affected, got: %s", responseParamsResult[0].Name)
	}

	// Request parameters should be replaced
	var requestParamsResult []models.Parameter
	db.Where("param_type = ?", "request").Find(&requestParamsResult)

	if len(requestParamsResult) != 1 {
		t.Errorf("Expected 1 request parameter, got: %d", len(requestParamsResult))
	}

	if requestParamsResult[0].Name != "newRequestParam" {
		t.Errorf("Request parameter should be replaced, got: %s", requestParamsResult[0].Name)
	}
}

// TestParameterService_UpdateParameters_GivenDeepNesting_whenUpdateParameters_thenAllLevelsSaved
func TestParameterService_UpdateParameters_GivenDeepNesting_whenUpdateParameters_thenAllLevelsSaved(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	service := NewParameterService(db)

	apiID := uint(1)
	paramType := "request"

	// Deep nesting: object -> object -> array -> string
	parameters := json.RawMessage(`[
		{
			"name": "level1",
			"type": "object",
			"children": [
				{
					"name": "level2",
					"type": "object",
					"children": [
						{
							"name": "level3",
							"type": "array",
							"children": [
								{
									"name": "level4",
									"type": "string",
									"description": "Deep nested string"
								}
							]
						}
					]
				}
			]
		}
	]`)

	// Act
	count, err := service.UpdateParameters(apiID, paramType, parameters)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if count != 4 {
		t.Errorf("Expected 4 parameters at all levels, got: %d", count)
	}

	var allParams []models.Parameter
	db.Find(&allParams)

	if len(allParams) != 4 {
		t.Errorf("Expected 4 parameters in database, got: %d", len(allParams))
	}

	// Verify the hierarchy
	var level1 models.Parameter
	db.Where("name = ? AND parent_id IS NULL", "level1").First(&level1)

	if level1.ID == 0 {
		t.Fatal("Level 1 parameter not found")
	}

	var level2 models.Parameter
	db.Where("name = ? AND parent_id = ?", "level2", level1.ID).First(&level2)

	if level2.ID == 0 {
		t.Fatal("Level 2 parameter not found")
	}

	var level3 models.Parameter
	db.Where("name = ? AND parent_id = ?", "level3", level2.ID).First(&level3)

	if level3.ID == 0 {
		t.Fatal("Level 3 parameter not found")
	}

	var level4 models.Parameter
	db.Where("name = ? AND parent_id = ?", "level4", level3.ID).First(&level4)

	if level4.ID == 0 {
		t.Fatal("Level 4 parameter not found")
	}
}

// TestParameterService_UpdateParameters_GivenArrayWithMultipleItems_whenUpdateParameters_thenChildrenSaved
func TestParameterService_UpdateParameters_GivenArrayWithMultipleItems_whenUpdateParameters_thenChildrenSaved(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	service := NewParameterService(db)

	apiID := uint(1)
	paramType := "response"

	parameters := json.RawMessage(`[
		{
			"name": "items",
			"type": "array",
			"description": "List of items",
			"children": [
				{
					"name": "itemId",
					"type": "string",
					"description": "Item ID"
				},
				{
					"name": "itemName",
					"type": "string",
					"description": "Item name"
				},
				{
					"name": "quantity",
					"type": "number",
					"description": "Item quantity"
				}
			]
		}
	]`)

	// Act
	count, err := service.UpdateParameters(apiID, paramType, parameters)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if count != 4 {
		t.Errorf("Expected 4 parameters (1 array + 3 children), got: %d", count)
	}

	var arrayParam models.Parameter
	db.Where("name = ? AND type = ?", "items", "array").First(&arrayParam)

	if arrayParam.ID == 0 {
		t.Fatal("Array parameter not found")
	}

	var children []models.Parameter
	db.Where("parent_id = ?", arrayParam.ID).Find(&children)

	if len(children) != 3 {
		t.Errorf("Expected 3 children for array parameter, got: %d", len(children))
	}

	expectedNames := []string{"itemId", "itemName", "quantity"}
	for i, child := range children {
		if child.Name != expectedNames[i] {
			t.Errorf("Child %d name incorrect: expected %s, got %s", i, expectedNames[i], child.Name)
		}
	}
}

// TestParameterService_UpdateParameters_GivenParametersWithoutDescription_whenUpdateParameters_thenDescriptionNil
func TestParameterService_UpdateParameters_GivenParametersWithoutDescription_whenUpdateParameters_thenDescriptionNil(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	service := NewParameterService(db)

	apiID := uint(1)
	paramType := "request"

	parameters := json.RawMessage(`[
		{
			"name": "param1",
			"type": "string",
			"required": true
		},
		{
			"name": "param2",
			"type": "number",
			"required": false
		}
	]`)

	// Act
	count, err := service.UpdateParameters(apiID, paramType, parameters)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected 2 parameters, got: %d", count)
	}

	var params []models.Parameter
	db.Find(&params)

	for _, param := range params {
		if param.Description != nil {
			t.Errorf("Description should be nil when not provided, got: %v", param.Description)
		}
	}
}

// TestParameterService_UpdateParametersFromJSON_GivenValidJSON_whenConvert_thenParametersCreated
func TestParameterService_UpdateParametersFromJSON_GivenValidJSON_whenConvert_thenParametersCreated(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	service := NewParameterService(db)

	apiID := uint(1)
	paramType := "request"

	jsonData := map[string]interface{}{
		"username": "john_doe",
		"age":       float64(30),
		"active":    true,
	}

	// Act
	count, err := service.UpdateParametersFromJSON(apiID, paramType, jsonData)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if count != 3 {
		t.Errorf("Expected 3 parameters, got: %d", count)
	}

	var params []models.Parameter
	db.Find(&params)

	if len(params) != 3 {
		t.Errorf("Expected 3 parameters in database, got: %d", len(params))
	}

	// Verify types
	paramMap := make(map[string]models.Parameter)
	for _, param := range params {
		paramMap[param.Name] = param
	}

	if paramMap["username"].Type != "string" {
		t.Errorf("Expected username type to be 'string', got: %s", paramMap["username"].Type)
	}

	if paramMap["age"].Type != "number" {
		t.Errorf("Expected age type to be 'number', got: %s", paramMap["age"].Type)
	}

	if paramMap["active"].Type != "boolean" {
		t.Errorf("Expected active type to be 'boolean', got: %s", paramMap["active"].Type)
	}
}

// TestParameterService_UpdateParametersFromJSON_GivenNestedObject_whenConvert_thenNestedParametersCreated
func TestParameterService_UpdateParametersFromJSON_GivenNestedObject_whenConvert_thenNestedParametersCreated(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	service := NewParameterService(db)

	apiID := uint(1)
	paramType := "request"

	jsonData := map[string]interface{}{
		"user": map[string]interface{}{
			"name": "John Doe",
			"age":  float64(30),
		},
	}

	// Act
	count, err := service.UpdateParametersFromJSON(apiID, paramType, jsonData)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 root parameter, got: %d", count)
	}

	var userParam models.Parameter
	db.Where("name = ? AND parent_id IS NULL", "user").First(&userParam)

	if userParam.ID == 0 {
		t.Fatal("User parameter not found")
	}

	if userParam.Type != "object" {
		t.Errorf("Expected user type to be 'object', got: %s", userParam.Type)
	}

	var children []models.Parameter
	db.Where("parent_id = ?", userParam.ID).Find(&children)

	if len(children) != 2 {
		t.Errorf("Expected 2 child parameters, got: %d", len(children))
	}
}

// TestParameterService_UpdateParametersFromJSON_GivenArray_whenConvert_thenArrayParameterCreated
func TestParameterService_UpdateParametersFromJSON_GivenArray_whenConvert_thenArrayParameterCreated(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	service := NewParameterService(db)

	apiID := uint(1)
	paramType := "response"

	jsonData := map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{
				"id":   "1",
				"name": "Item 1",
			},
		},
	}

	// Act
	count, err := service.UpdateParametersFromJSON(apiID, paramType, jsonData)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 root parameter, got: %d", count)
	}

	var itemsParam models.Parameter
	db.Where("name = ?", "items").First(&itemsParam)

	if itemsParam.ID == 0 {
		t.Fatal("Items parameter not found")
	}

	if itemsParam.Type != "array" {
		t.Errorf("Expected items type to be 'array', got: %s", itemsParam.Type)
	}

	var children []models.Parameter
	db.Where("parent_id = ?", itemsParam.ID).Find(&children)

	if len(children) != 2 {
		t.Errorf("Expected 2 child parameters (from array item), got: %d", len(children))
	}
}

// TestParameterService_UpdateParametersFromJSON_GivenExistingParams_whenConvert_thenPreservesRequiredAndDescription
func TestParameterService_UpdateParametersFromJSON_GivenExistingParams_whenConvert_thenPreservesRequiredAndDescription(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	service := NewParameterService(db)

	apiID := uint(1)
	paramType := "request"

	// Create existing parameters with required flags and descriptions
	desc1 := "User's username"
	desc2 := "User's age"

	existingParams := []models.Parameter{
		{
			APIID:       apiID,
			Name:        "username",
			Type:        "string",
			Required:    true,
			Description: &desc1,
			ParamType:   paramType,
			Order:       0,
		},
		{
			APIID:       apiID,
			Name:        "age",
			Type:        "number",
			Required:    false,
			Description: &desc2,
			ParamType:   paramType,
			Order:       1,
		},
	}

	for _, param := range existingParams {
		db.Create(&param)
	}

	// New JSON data with same keys
	jsonData := map[string]interface{}{
		"username": "john_doe",
		"age":       float64(30),
	}

	// Act
	count, err := service.UpdateParametersFromJSON(apiID, paramType, jsonData)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected 2 parameters, got: %d", count)
	}

	var params []models.Parameter
	db.Find(&params)

	// Verify required flags and descriptions are preserved
	paramMap := make(map[string]models.Parameter)
	for _, param := range params {
		paramMap[param.Name] = param
	}

	if !paramMap["username"].Required {
		t.Errorf("Expected username required to be preserved as true")
	}

	if paramMap["username"].Description == nil || *paramMap["username"].Description != "User's username" {
		t.Errorf("Expected username description to be preserved")
	}

	if paramMap["age"].Required {
		t.Errorf("Expected age required to be preserved as false")
	}
}

// TestParameterService_UpdateParameters_GivenParameterOrder_whenUpdateParameters_thenOrderPreserved
func TestParameterService_UpdateParameters_GivenParameterOrder_whenUpdateParameters_thenOrderPreserved(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	service := NewParameterService(db)

	apiID := uint(1)
	paramType := "request"

	parameters := json.RawMessage(`[
		{
			"name": "first",
			"type": "string",
			"description": "First parameter",
			"required": true
		},
		{
			"name": "second",
			"type": "string",
			"description": "Second parameter",
			"required": true
		},
		{
			"name": "third",
			"type": "string",
			"description": "Third parameter",
			"required": true
		}
	]`)

	// Act
	count, err := service.UpdateParameters(apiID, paramType, parameters)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if count != 3 {
		t.Errorf("Expected 3 parameters, got: %d", count)
	}

	var params []models.Parameter
	db.Order("`order` ASC").Find(&params)

	if len(params) != 3 {
		t.Fatalf("Expected 3 parameters in database, got: %d", len(params))
	}

	if params[0].Name != "first" || params[0].Order != 0 {
		t.Errorf("First parameter order incorrect: name=%s, order=%d", params[0].Name, params[0].Order)
	}

	if params[1].Name != "second" || params[1].Order != 1 {
		t.Errorf("Second parameter order incorrect: name=%s, order=%d", params[1].Name, params[1].Order)
	}

	if params[2].Name != "third" || params[2].Order != 2 {
		t.Errorf("Third parameter order incorrect: name=%s, order=%d", params[2].Name, params[2].Order)
	}
}
