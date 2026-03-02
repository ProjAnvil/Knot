package services

import (
	"strings"
	"testing"

	"github.com/ProjAnvil/knot/backend/internal/models"
)

// TestExportService_BuildParameterTree_GivenEmptyList_whenBuildTree_thenEmptyResult
func TestExportService_BuildParameterTree_GivenEmptyList_whenBuildTree_thenEmptyResult(t *testing.T) {
	// Arrange
	params := []models.Parameter{}

	// Act
	result := BuildParameterTree(params)

	// Assert
	if len(result) != 0 {
		t.Errorf("Expected empty result for empty input, got %d items", len(result))
	}
}

// TestExportService_BuildParameterTree_GivenFlatParameters_whenBuildTree_thenRootsReturned
func TestExportService_BuildParameterTree_GivenFlatParameters_whenBuildTree_thenRootsReturned(t *testing.T) {
	// Arrange
	params := []models.Parameter{
		{ID: 1, Name: "param1", Type: "string", Required: true, ParentID: nil},
		{ID: 2, Name: "param2", Type: "number", Required: false, ParentID: nil},
		{ID: 3, Name: "param3", Type: "boolean", Required: true, ParentID: nil},
	}

	// Act
	result := BuildParameterTree(params)

	// Assert
	if len(result) != 3 {
		t.Errorf("Expected 3 root parameters, got %d", len(result))
	}

	for i, param := range result {
		if param.ParentID != nil {
			t.Errorf("Parameter %d should have nil ParentID, got %v", i, param.ParentID)
		}
		if len(param.Children) != 0 {
			t.Errorf("Parameter %d should have no children, got %d", i, len(param.Children))
		}
	}
}

// TestExportService_BuildParameterTree_GivenNestedParameters_whenBuildTree_thenTreeBuiltCorrectly
func TestExportService_BuildParameterTree_GivenNestedParameters_whenBuildTree_thenTreeBuiltCorrectly(t *testing.T) {
	// Arrange
	parentID := uint(1)
	params := []models.Parameter{
		{ID: 1, Name: "user", Type: "object", Required: true, ParentID: nil},
		{ID: 2, Name: "name", Type: "string", Required: true, ParentID: &parentID},
		{ID: 3, Name: "age", Type: "number", Required: false, ParentID: &parentID},
	}

	// Act
	result := BuildParameterTree(params)

	// Assert
	if len(result) != 1 {
		t.Fatalf("Expected 1 root parameter, got %d", len(result))
	}

	root := result[0]
	if root.Name != "user" {
		t.Errorf("Expected root name 'user', got '%s'", root.Name)
	}

	if len(root.Children) != 2 {
		t.Fatalf("Expected 2 children, got %d", len(root.Children))
	}

	if root.Children[0].Name != "name" {
		t.Errorf("Expected first child name 'name', got '%s'", root.Children[0].Name)
	}

	if root.Children[1].Name != "age" {
		t.Errorf("Expected second child name 'age', got '%s'", root.Children[1].Name)
	}
}

// TestExportService_BuildParameterTree_GivenMultipleNestedParams_whenBuildTree_thenAllChildrenBuilt
func TestExportService_BuildParameterTree_GivenMultipleNestedParams_whenBuildTree_thenAllChildrenBuilt(t *testing.T) {
	// Arrange
	// Test with multiple root parameters, each with their own children
	parentID1 := uint(1)
	parentID2 := uint(4)

	params := []models.Parameter{
		{ID: 1, Name: "user", Type: "object", ParentID: nil},
		{ID: 2, Name: "name", Type: "string", ParentID: &parentID1},
		{ID: 3, Name: "age", Type: "number", ParentID: &parentID1},
		{ID: 4, Name: "address", Type: "object", ParentID: nil},
		{ID: 5, Name: "street", Type: "string", ParentID: &parentID2},
		{ID: 6, Name: "city", Type: "string", ParentID: &parentID2},
	}

	// Act
	result := BuildParameterTree(params)

	// Assert
	if len(result) != 2 {
		t.Fatalf("Expected 2 root parameters, got %d", len(result))
	}

	// Find "user" root
	var userRoot *models.Parameter
	for i := range result {
		if result[i].Name == "user" {
			userRoot = &result[i]
			break
		}
	}

	if userRoot == nil {
		t.Fatal("user root not found in result")
	}

	if len(userRoot.Children) != 2 {
		t.Errorf("Expected user root to have 2 children, got %d", len(userRoot.Children))
	}

	// Find "address" root
	var addressRoot *models.Parameter
	for i := range result {
		if result[i].Name == "address" {
			addressRoot = &result[i]
			break
		}
	}

	if addressRoot == nil {
		t.Fatal("address root not found in result")
	}

	if len(addressRoot.Children) != 2 {
		t.Errorf("Expected address root to have 2 children, got %d", len(addressRoot.Children))
	}
}

// TestExportService_BuildParameterTree_GivenMixedRootsAndChildren_whenBuildTree_thenRootsOnly
func TestExportService_BuildParameterTree_GivenMixedRootsAndChildren_whenBuildTree_thenRootsOnly(t *testing.T) {
	// Arrange
	parentID := uint(1)
	params := []models.Parameter{
		{ID: 1, Name: "root1", Type: "object", ParentID: nil},
		{ID: 2, Name: "child1", Type: "string", ParentID: &parentID},
		{ID: 3, Name: "root2", Type: "string", ParentID: nil},
		{ID: 4, Name: "child2", Type: "number", ParentID: &parentID},
	}

	// Act
	result := BuildParameterTree(params)

	// Assert
	if len(result) != 2 {
		t.Fatalf("Expected 2 root parameters, got %d", len(result))
	}

	// Verify root1 has children
	var root1 *models.Parameter
	for i := range result {
		if result[i].Name == "root1" {
			root1 = &result[i]
			break
		}
	}

	if root1 == nil {
		t.Fatal("root1 not found in result")
	}

	if len(root1.Children) != 2 {
		t.Errorf("Expected root1 to have 2 children, got %d", len(root1.Children))
	}

	// Verify root2 has no children
	var root2 *models.Parameter
	for i := range result {
		if result[i].Name == "root2" {
			root2 = &result[i]
			break
		}
	}

	if root2 == nil {
		t.Fatal("root2 not found in result")
	}

	if len(root2.Children) != 0 {
		t.Errorf("Expected root2 to have 0 children, got %d", len(root2.Children))
	}
}

// TestExportService_GenerateParameterHTML_GivenEmptyList_whenGenerate_thenNoParamsMessage
func TestExportService_GenerateParameterHTML_GivenEmptyList_whenGenerate_thenNoParamsMessage(t *testing.T) {
	// Arrange
	params := []models.Parameter{}

	// Act
	result := GenerateParameterHTML(params, 0)

	// Assert
	expected := `<p class="text-muted">No parameters</p>`
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

// TestExportService_GenerateParameterHTML_GivenFlatParams_whenGenerate_thenTableGenerated
func TestExportService_GenerateParameterHTML_GivenFlatParams_whenGenerate_thenTableGenerated(t *testing.T) {
	// Arrange
	desc := "Test description"
	params := []models.Parameter{
		{ID: 1, Name: "username", Type: "string", Required: true, Description: &desc, ParentID: nil, Children: []models.Parameter{}},
		{ID: 2, Name: "age", Type: "number", Required: false, ParentID: nil, Children: []models.Parameter{}},
	}

	// Act
	result := GenerateParameterHTML(params, 0)

	// Assert
	if !strings.Contains(result, `<table class="param-table">`) {
		t.Error("Expected result to contain table tag")
	}

	if !strings.Contains(result, "username") {
		t.Error("Expected result to contain 'username'")
	}

	if !strings.Contains(result, "age") {
		t.Error("Expected result to contain 'age'")
	}

	if !strings.Contains(result, "badge-required") {
		t.Error("Expected result to contain required badge for username")
	}

	if !strings.Contains(result, "badge-optional") {
		t.Error("Expected result to contain optional badge for age")
	}

	if !strings.Contains(result, "type-string") {
		t.Error("Expected result to contain string type badge")
	}

	if !strings.Contains(result, "type-number") {
		t.Error("Expected result to contain number type badge")
	}

	if !strings.Contains(result, "Test description") {
		t.Error("Expected result to contain description")
	}
}

// TestExportService_GenerateParameterHTML_GivenNestedParams_whenGenerate_thenIndentedChildren
func TestExportService_GenerateParameterHTML_GivenNestedParams_whenGenerate_thenIndentedChildren(t *testing.T) {
	// Arrange
	parentID := uint(1)
	params := []models.Parameter{
		{
			ID: 1, Name: "user", Type: "object", Required: true, ParentID: nil,
			Children: []models.Parameter{
				{ID: 2, Name: "name", Type: "string", Required: true, ParentID: &parentID, Children: []models.Parameter{}},
				{ID: 3, Name: "age", Type: "number", Required: false, ParentID: &parentID, Children: []models.Parameter{}},
			},
		},
	}

	// Act
	result := GenerateParameterHTML(params, 0)

	// Assert
	if !strings.Contains(result, "user") {
		t.Error("Expected result to contain 'user'")
	}

	if !strings.Contains(result, "name") {
		t.Error("Expected result to contain child 'name'")
	}

	if !strings.Contains(result, "age") {
		t.Error("Expected result to contain child 'age'")
	}

	// Check for indentation (non-breaking spaces)
	if !strings.Contains(result, "&nbsp;") {
		t.Error("Expected result to contain indentation for nested parameters")
	}

	// Check for tree prefix
	if !strings.Contains(result, "└─") {
		t.Error("Expected result to contain tree prefix for nested parameters")
	}
}

// TestExportService_GenerateParameterHTML_GivenNoDescription_whenGenerate_thenDashShown
func TestExportService_GenerateParameterHTML_GivenNoDescription_whenGenerate_thenDashShown(t *testing.T) {
	// Arrange
	params := []models.Parameter{
		{ID: 1, Name: "param", Type: "string", Required: true, Description: nil, ParentID: nil, Children: []models.Parameter{}},
	}

	// Act
	result := GenerateParameterHTML(params, 0)

	// Assert
	if !strings.Contains(result, ">-<") {
		t.Error("Expected result to contain '-' for missing description")
	}
}

// TestExportService_GenerateExampleJSON_GivenStringType_whenGenerate_thenStringValue
func TestExportService_GenerateExampleJSON_GivenStringType_whenGenerate_thenStringValue(t *testing.T) {
	// Arrange
	params := []models.Parameter{
		{ID: 1, Name: "username", Type: "string", Description: nil, ParentID: nil, Children: []models.Parameter{}},
	}

	// Act
	result := GenerateExampleJSON(params)

	// Assert
	if len(result) != 1 {
		t.Fatalf("Expected 1 key in result, got %d", len(result))
	}

	value, ok := result["username"]
	if !ok {
		t.Fatal("Expected 'username' key in result")
	}

	strValue, ok := value.(string)
	if !ok {
		t.Fatalf("Expected string value, got %T", value)
	}

	if strValue != "string" {
		t.Errorf("Expected 'string' value, got '%s'", strValue)
	}
}

// TestExportService_GenerateExampleJSON_GivenStringWithDescription_whenGenerate_thenDescriptionAsValue
func TestExportService_GenerateExampleJSON_GivenStringWithDescription_whenGenerate_thenDescriptionAsValue(t *testing.T) {
	// Arrange
	desc := "john_doe"
	params := []models.Parameter{
		{ID: 1, Name: "username", Type: "string", Description: &desc, ParentID: nil, Children: []models.Parameter{}},
	}

	// Act
	result := GenerateExampleJSON(params)

	// Assert
	value, ok := result["username"]
	if !ok {
		t.Fatal("Expected 'username' key in result")
	}

	if value != "john_doe" {
		t.Errorf("Expected description 'john_doe' as value, got '%v'", value)
	}
}

// TestExportService_GenerateExampleJSON_GivenNumberType_whenGenerate_thenZeroValue
func TestExportService_GenerateExampleJSON_GivenNumberType_whenGenerate_thenZeroValue(t *testing.T) {
	// Arrange
	params := []models.Parameter{
		{ID: 1, Name: "age", Type: "number", Description: nil, ParentID: nil, Children: []models.Parameter{}},
	}

	// Act
	result := GenerateExampleJSON(params)

	// Assert
	value, ok := result["age"]
	if !ok {
		t.Fatal("Expected 'age' key in result")
	}

	// The value can be either int or float64 depending on implementation
	numValueAsInt, isInt := value.(int)
	numValueAsFloat, isFloat := value.(float64)

	if !isInt && !isFloat {
		t.Fatalf("Expected numeric value, got %T", value)
	}

	if isInt && numValueAsInt != 0 {
		t.Errorf("Expected 0 value, got %v", numValueAsInt)
	}

	if isFloat && numValueAsFloat != 0 {
		t.Errorf("Expected 0 value, got %v", numValueAsFloat)
	}
}

// TestExportService_GenerateExampleJSON_GivenBooleanType_whenGenerate_thenFalseValue
func TestExportService_GenerateExampleJSON_GivenBooleanType_whenGenerate_thenFalseValue(t *testing.T) {
	// Arrange
	params := []models.Parameter{
		{ID: 1, Name: "active", Type: "boolean", Description: nil, ParentID: nil, Children: []models.Parameter{}},
	}

	// Act
	result := GenerateExampleJSON(params)

	// Assert
	value, ok := result["active"]
	if !ok {
		t.Fatal("Expected 'active' key in result")
	}

	boolValue, ok := value.(bool)
	if !ok {
		t.Fatalf("Expected bool value, got %T", value)
	}

	if boolValue != false {
		t.Errorf("Expected false value, got %v", boolValue)
	}
}

// TestExportService_GenerateExampleJSON_GivenEmptyObject_whenGenerate_thenEmptyObject
func TestExportService_GenerateExampleJSON_GivenEmptyObject_whenGenerate_thenEmptyObject(t *testing.T) {
	// Arrange
	params := []models.Parameter{
		{ID: 1, Name: "metadata", Type: "object", Description: nil, ParentID: nil, Children: []models.Parameter{}},
	}

	// Act
	result := GenerateExampleJSON(params)

	// Assert
	value, ok := result["metadata"]
	if !ok {
		t.Fatal("Expected 'metadata' key in result")
	}

	objValue, ok := value.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map value, got %T", value)
	}

	if len(objValue) != 0 {
		t.Errorf("Expected empty object, got %d keys", len(objValue))
	}
}

// TestExportService_GenerateExampleJSON_GivenNestedObject_whenGenerate_thenNestedJSON
func TestExportService_GenerateExampleJSON_GivenNestedObject_whenGenerate_thenNestedJSON(t *testing.T) {
	// Arrange
	parentID := uint(1)
	params := []models.Parameter{
		{
			ID: 1, Name: "user", Type: "object", Description: nil, ParentID: nil,
			Children: []models.Parameter{
				{ID: 2, Name: "name", Type: "string", Description: nil, ParentID: &parentID, Children: []models.Parameter{}},
				{ID: 3, Name: "age", Type: "number", Description: nil, ParentID: &parentID, Children: []models.Parameter{}},
			},
		},
	}

	// Act
	result := GenerateExampleJSON(params)

	// Assert
	userValue, ok := result["user"]
	if !ok {
		t.Fatal("Expected 'user' key in result")
	}

	userObj, ok := userValue.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected user to be a map, got %T", userValue)
	}

	if len(userObj) != 2 {
		t.Errorf("Expected user object to have 2 keys, got %d", len(userObj))
	}

	if _, ok := userObj["name"]; !ok {
		t.Error("Expected 'name' key in user object")
	}

	if _, ok := userObj["age"]; !ok {
		t.Error("Expected 'age' key in user object")
	}
}

// TestExportService_GenerateExampleJSON_GivenEmptyArray_whenGenerate_thenEmptyArray
func TestExportService_GenerateExampleJSON_GivenEmptyArray_whenGenerate_thenEmptyArray(t *testing.T) {
	// Arrange
	params := []models.Parameter{
		{ID: 1, Name: "items", Type: "array", Description: nil, ParentID: nil, Children: []models.Parameter{}},
	}

	// Act
	result := GenerateExampleJSON(params)

	// Assert
	value, ok := result["items"]
	if !ok {
		t.Fatal("Expected 'items' key in result")
	}

	arrValue, ok := value.([]interface{})
	if !ok {
		t.Fatalf("Expected array value, got %T", value)
	}

	if len(arrValue) != 0 {
		t.Errorf("Expected empty array, got %d items", len(arrValue))
	}
}

// TestExportService_GenerateExampleJSON_GivenArrayWithPrimitiveChild_whenGenerate_thenArrayOfOneItem
func TestExportService_GenerateExampleJSON_GivenArrayWithPrimitiveChild_whenGenerate_thenArrayOfOneItem(t *testing.T) {
	// Arrange
	parentID := uint(1)
	params := []models.Parameter{
		{
			ID: 1, Name: "tags", Type: "array", Description: nil, ParentID: nil,
			Children: []models.Parameter{
				{ID: 2, Name: "tag", Type: "string", Description: nil, ParentID: &parentID, Children: []models.Parameter{}},
			},
		},
	}

	// Act
	result := GenerateExampleJSON(params)

	// Assert
	value, ok := result["tags"]
	if !ok {
		t.Fatal("Expected 'tags' key in result")
	}

	arrValue, ok := value.([]interface{})
	if !ok {
		t.Fatalf("Expected array value, got %T", value)
	}

	if len(arrValue) != 1 {
		t.Fatalf("Expected array with 1 item, got %d", len(arrValue))
	}

	if arrValue[0] != "string" {
		t.Errorf("Expected array item to be 'string', got %v", arrValue[0])
	}
}

// TestExportService_GenerateExampleJSON_GivenArrayOfObjects_whenGenerate_thenArrayOfObjectExamples
func TestExportService_GenerateExampleJSON_GivenArrayOfObjects_whenGenerate_thenArrayOfObjectExamples(t *testing.T) {
	// Arrange
	parentID := uint(1)
	params := []models.Parameter{
		{
			ID: 1, Name: "users", Type: "array", Description: nil, ParentID: nil,
			Children: []models.Parameter{
				{ID: 2, Name: "name", Type: "string", Description: nil, ParentID: &parentID, Children: []models.Parameter{}},
				{ID: 3, Name: "age", Type: "number", Description: nil, ParentID: &parentID, Children: []models.Parameter{}},
			},
		},
	}

	// Act
	result := GenerateExampleJSON(params)

	// Assert
	value, ok := result["users"]
	if !ok {
		t.Fatal("Expected 'users' key in result")
	}

	arrValue, ok := value.([]interface{})
	if !ok {
		t.Fatalf("Expected array value, got %T", value)
	}

	if len(arrValue) != 1 {
		t.Fatalf("Expected array with 1 item (object example), got %d", len(arrValue))
	}

	objValue, ok := arrValue[0].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected object in array, got %T", arrValue[0])
	}

	if len(objValue) != 2 {
		t.Errorf("Expected object with 2 keys, got %d", len(objValue))
	}
}

// TestExportService_GenerateHTML_GivenEnglishLocale_whenGenerate_thenEnglishText
func TestExportService_GenerateHTML_GivenEnglishLocale_whenGenerate_thenEnglishText(t *testing.T) {
	// Arrange
	desc := "Test description"
	desc2 := "Response description"

	apis := []APIWithParams{
		{
			API: models.API{
				ID:        1,
				GroupID:   1,
				Name:      "Get User",
				Endpoint:  "/api/users/{id}",
				Method:    "GET",
				Type:      "HTTP",
				Note:      &desc,
			},
			GroupName: "User Management",
			RequestParameters: []models.Parameter{
				{ID: 1, Name: "userId", Type: "number", Required: true, Description: &desc, ParentID: nil, Children: []models.Parameter{}},
			},
			ResponseParameters: []models.Parameter{
				{ID: 2, Name: "userName", Type: "string", Required: true, Description: &desc2, ParentID: nil, Children: []models.Parameter{}},
			},
		},
	}

	// Act
	result := GenerateHTML(apis, "en")

	// Assert
	if !strings.Contains(result, "API Documentation") {
		t.Error("Expected English title 'API Documentation'")
	}

	if !strings.Contains(result, "Request Parameters") {
		t.Error("Expected English 'Request Parameters' text")
	}

	if !strings.Contains(result, "Response Parameters") {
		t.Error("Expected English 'Response Parameters' text")
	}

	// Examples are only shown when there are parameters (len(JSON) > 0)
	if !strings.Contains(result, "Request Example") {
		t.Error("Expected English 'Request Example' text")
	}

	if !strings.Contains(result, "Response Example") {
		t.Error("Expected English 'Response Example' text")
	}

	if !strings.Contains(result, "Table of Contents") {
		t.Error("Expected English 'Table of Contents' text")
	}

	if !strings.Contains(result, "Generated at") {
		t.Error("Expected English 'Generated at' text")
	}
}

// TestExportService_GenerateHTML_GivenChineseLocale_whenGenerate_thenChineseText
func TestExportService_GenerateHTML_GivenChineseLocale_whenGenerate_thenChineseText(t *testing.T) {
	// Arrange
	desc := "测试描述"
	desc2 := "响应描述"

	apis := []APIWithParams{
		{
			API: models.API{
				ID:        1,
				GroupID:   1,
				Name:      "获取用户",
				Endpoint:  "/api/users/{id}",
				Method:    "GET",
				Type:      "HTTP",
				Note:      &desc,
			},
			GroupName: "用户管理",
			RequestParameters: []models.Parameter{
				{ID: 1, Name: "userId", Type: "number", Required: true, Description: &desc, ParentID: nil, Children: []models.Parameter{}},
			},
			ResponseParameters: []models.Parameter{
				{ID: 2, Name: "userName", Type: "string", Required: true, Description: &desc2, ParentID: nil, Children: []models.Parameter{}},
			},
		},
	}

	// Act
	result := GenerateHTML(apis, "zh")

	// Assert
	if !strings.Contains(result, "API 文档") {
		t.Error("Expected Chinese title 'API 文档'")
	}

	if !strings.Contains(result, "请求参数") {
		t.Error("Expected Chinese '请求参数' text")
	}

	if !strings.Contains(result, "响应参数") {
		t.Error("Expected Chinese '响应参数' text")
	}

	// Examples are only shown when there are parameters
	if !strings.Contains(result, "请求示例") {
		t.Error("Expected Chinese '请求示例' text")
	}

	if !strings.Contains(result, "响应示例") {
		t.Error("Expected Chinese '响应示例' text")
	}

	if !strings.Contains(result, "目录") {
		t.Error("Expected Chinese '目录' text")
	}

	if !strings.Contains(result, "生成时间") {
		t.Error("Expected Chinese '生成时间' text")
	}
}

// TestExportService_GenerateHTML_GivenMultipleGroups_whenGenerate_thenSidebarGroups
func TestExportService_GenerateHTML_GivenMultipleGroups_whenGenerate_thenSidebarGroups(t *testing.T) {
	// Arrange
	apis := []APIWithParams{
		{
			API: models.API{ID: 1, GroupID: 1, Name: "API 1", Endpoint: "/api/1", Method: "GET", Type: "HTTP"},
			GroupName:          "Group A",
			RequestParameters:  []models.Parameter{},
			ResponseParameters: []models.Parameter{},
		},
		{
			API: models.API{ID: 2, GroupID: 2, Name: "API 2", Endpoint: "/api/2", Method: "POST", Type: "HTTP"},
			GroupName:          "Group B",
			RequestParameters:  []models.Parameter{},
			ResponseParameters: []models.Parameter{},
		},
		{
			API: models.API{ID: 3, GroupID: 1, Name: "API 3", Endpoint: "/api/3", Method: "PUT", Type: "HTTP"},
			GroupName:          "Group A",
			RequestParameters:  []models.Parameter{},
			ResponseParameters: []models.Parameter{},
		},
	}

	// Act
	result := GenerateHTML(apis, "en")

	// Assert
	// Should contain sidebar groups
	if !strings.Contains(result, "sidebar-group") {
		t.Error("Expected sidebar-group class")
	}

	// Should contain group names
	if !strings.Contains(result, "Group A") {
		t.Error("Expected 'Group A' in sidebar")
	}

	if !strings.Contains(result, "Group B") {
		t.Error("Expected 'Group B' in sidebar")
	}

	// Should contain API links
	if !strings.Contains(result, "href=\"#api-0\"") {
		t.Error("Expected API link for api-0")
	}

	if !strings.Contains(result, "href=\"#api-1\"") {
		t.Error("Expected API link for api-1")
	}

	if !strings.Contains(result, "href=\"#api-2\"") {
		t.Error("Expected API link for api-2")
	}
}

// TestExportService_GenerateHTML_GivenAPIWithParams_whenGenerate_thenCompleteHTMLDocument
func TestExportService_GenerateHTML_GivenAPIWithParams_whenGenerate_thenCompleteHTMLDocument(t *testing.T) {
	// Arrange
	reqDesc := "Request username"
	resDesc := "Response user ID"

	apis := []APIWithParams{
		{
			API: models.API{
				ID:       1,
				GroupID:  1,
				Name:     "Create User",
				Endpoint: "/api/users",
				Method:   "POST",
				Type:     "HTTP",
			},
			GroupName: "Users",
			RequestParameters: []models.Parameter{
				{
					ID: 1, Name: "username", Type: "string", Required: true, Description: &reqDesc, ParentID: nil,
					Children: []models.Parameter{},
				},
			},
			ResponseParameters: []models.Parameter{
				{
					ID: 2, Name: "userId", Type: "number", Required: true, Description: &resDesc, ParentID: nil,
					Children: []models.Parameter{},
				},
			},
		},
	}

	// Act
	result := GenerateHTML(apis, "en")

	// Assert
	// Check for DOCTYPE and HTML structure
	if !strings.Contains(result, "<!DOCTYPE html>") {
		t.Error("Expected DOCTYPE declaration")
	}

	if !strings.Contains(result, "<html lang=\"en\">") {
		t.Error("Expected HTML tag with lang attribute")
	}

	if !strings.Contains(result, "<head>") {
		t.Error("Expected head tag")
	}

	if !strings.Contains(result, "<body>") {
		t.Error("Expected body tag")
	}

	// Check for sidebar
	if !strings.Contains(result, "class=\"sidebar\"") {
		t.Error("Expected sidebar div")
	}

	// Check for main content
	if !strings.Contains(result, "class=\"main-content\"") {
		t.Error("Expected main-content div")
	}

	// Check for API section
	if !strings.Contains(result, "id=\"api-0\"") {
		t.Error("Expected API section with id")
	}

	if !strings.Contains(result, "Create User") {
		t.Error("Expected API name in HTML")
	}

	if !strings.Contains(result, "POST") {
		t.Error("Expected HTTP method in HTML")
	}

	if !strings.Contains(result, "/api/users") {
		t.Error("Expected endpoint in HTML")
	}

	// Check for badge class based on method
	if !strings.Contains(result, "badge-post") {
		t.Error("Expected POST badge class")
	}

	// Check for parameter tables
	if !strings.Contains(result, "param-table") {
		t.Error("Expected parameter table class")
	}

	// Check for parameter values
	if !strings.Contains(result, "username") {
		t.Error("Expected 'username' parameter in HTML")
	}

	if !strings.Contains(result, "userId") {
		t.Error("Expected 'userId' parameter in HTML")
	}

	// Check for JSON examples
	if !strings.Contains(result, "<pre><code>") {
		t.Error("Expected JSON code blocks")
	}
}

// TestExportService_GenerateHTML_GivenAPIWithNestedParamsInRequest_whenGenerate_thenNestedParamsInHTML
func TestExportService_GenerateHTML_GivenAPIWithNestedParamsInRequest_whenGenerate_thenNestedParamsInHTML(t *testing.T) {
	// Arrange
	// Use flat parameter list with ParentID relationships, BuildParameterTree will build the tree
	parentID := uint(1)
	reqDesc := "User object"
	resDesc := "User ID response"

	apis := []APIWithParams{
		{
			API: models.API{
				ID:       1,
				GroupID:  1,
				Name:     "Create User",
				Endpoint: "/api/users",
				Method:   "POST",
				Type:     "HTTP",
			},
			GroupName: "Users",
			// Flat list with ParentID references - GenerateHTML will call BuildParameterTree
			RequestParameters: []models.Parameter{
				{ID: 1, Name: "user", Type: "object", Required: true, Description: &reqDesc, ParentID: nil, Children: []models.Parameter{}},
				{ID: 2, Name: "name", Type: "string", Required: true, ParentID: &parentID, Children: []models.Parameter{}},
				{ID: 3, Name: "age", Type: "number", Required: false, ParentID: &parentID, Children: []models.Parameter{}},
			},
			ResponseParameters: []models.Parameter{
				{ID: 4, Name: "userId", Type: "number", Required: true, Description: &resDesc, ParentID: nil, Children: []models.Parameter{}},
			},
		},
	}

	// Act
	result := GenerateHTML(apis, "en")

	// Assert
	// The request parameters should be built into a tree by GenerateHTML's BuildParameterTree call
	// and GenerateParameterHTML should show nested children with indentation
	// Check for nested parameter indicators
	if !strings.Contains(result, "&nbsp;") {
		t.Error("Expected indentation for nested parameters")
	}

	if !strings.Contains(result, "└─") {
		t.Error("Expected tree prefix for nested parameters")
	}

	// Check that both parent and children are in the result
	if !strings.Contains(result, "user") {
		t.Error("Expected parent parameter 'user' in HTML")
	}

	if !strings.Contains(result, "name") {
		t.Error("Expected child parameter 'name' in HTML")
	}

	if !strings.Contains(result, "age") {
		t.Error("Expected child parameter 'age' in HTML")
	}
}

// TestExportService_GenerateHTML_GivenAPIWithArrayParams_whenGenerate_thenArrayExampleInJSON
func TestExportService_GenerateHTML_GivenAPIWithArrayParams_whenGenerate_thenArrayExampleInJSON(t *testing.T) {
	// Arrange
	parentID := uint(1)
	resDesc := "User list"
	reqDesc := "Limit parameter"

	apis := []APIWithParams{
		{
			API: models.API{
				ID:       1,
				GroupID:  1,
				Name:     "List Users",
				Endpoint: "/api/users",
				Method:   "GET",
				Type:     "HTTP",
			},
			GroupName: "Users",
			RequestParameters: []models.Parameter{
				{ID: 1, Name: "limit", Type: "number", Required: false, Description: &reqDesc, ParentID: nil, Children: []models.Parameter{}},
			},
			// Flat list with ParentID references for array children
			ResponseParameters: []models.Parameter{
				{ID: 2, Name: "users", Type: "array", Required: true, Description: &resDesc, ParentID: nil, Children: []models.Parameter{}},
				{ID: 3, Name: "id", Type: "number", Required: true, ParentID: &parentID, Children: []models.Parameter{}},
				{ID: 4, Name: "name", Type: "string", Required: true, ParentID: &parentID, Children: []models.Parameter{}},
			},
		},
	}

	// Act
	result := GenerateHTML(apis, "en")

	// Assert
	// Check for array in JSON example
	if !strings.Contains(result, "[") {
		t.Error("Expected array syntax in JSON example")
	}

	// Check that children are in the result
	if !strings.Contains(result, "users") {
		t.Error("Expected array parameter 'users' in HTML")
	}

	if !strings.Contains(result, "id") {
		t.Error("Expected child parameter 'id' in HTML")
	}

	if !strings.Contains(result, "name") {
		t.Error("Expected child parameter 'name' in HTML")
	}
}

// TestExportService_BuildParameterTree_GivenOriginalParamsUnchanged_whenBuildTree_thenOriginalNotModified
func TestExportService_BuildParameterTree_GivenOriginalParamsUnchanged_whenBuildTree_thenOriginalNotModified(t *testing.T) {
	// Arrange
	parentID := uint(1)
	originalParams := []models.Parameter{
		{ID: 1, Name: "root", Type: "object", ParentID: nil, Children: []models.Parameter{}},
		{ID: 2, Name: "child", Type: "string", ParentID: &parentID, Children: []models.Parameter{}},
	}

	// Store original state
	originalRootChildrenLen := len(originalParams[0].Children)

	// Act
	result := BuildParameterTree(originalParams)

	// Assert - Original should not be modified
	if len(originalParams[0].Children) != originalRootChildrenLen {
		t.Errorf("Original parameters were modified: expected children length %d, got %d",
			originalRootChildrenLen, len(originalParams[0].Children))
	}

	// Result should have children
	if len(result[0].Children) != 1 {
		t.Errorf("Expected result to have 1 child, got %d", len(result[0].Children))
	}
}

// TestExportService_GenerateHTML_GivenAllHTTPMethods_whenGenerate_thenAllBadgesPresent
func TestExportService_GenerateHTML_GivenAllHTTPMethods_whenGenerate_thenAllBadgesPresent(t *testing.T) {
	// Arrange
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}
	var apis []APIWithParams

	for i, method := range methods {
		apis = append(apis, APIWithParams{
			API: models.API{
				ID:       uint(i + 1),
				GroupID:  1,
				Name:     method + " API",
				Endpoint: "/api/" + method,
				Method:   method,
				Type:     "HTTP",
			},
			GroupName:          "Test",
			RequestParameters:  []models.Parameter{},
			ResponseParameters: []models.Parameter{},
		})
	}

	// Act
	result := GenerateHTML(apis, "en")

	// Assert
	expectedBadges := []string{"badge-get", "badge-post", "badge-put", "badge-delete", "badge-patch"}
	for _, badge := range expectedBadges {
		if !strings.Contains(result, badge) {
			t.Errorf("Expected badge class '%s' in HTML", badge)
		}
	}
}

// TestExportService_GenerateHTML_VerifyValidHTMLStructure
func TestExportService_GenerateHTML_VerifyValidHTMLStructure(t *testing.T) {
	// Arrange
	apis := []APIWithParams{
		{
			API:                models.API{ID: 1, GroupID: 1, Name: "Test", Endpoint: "/test", Method: "GET", Type: "HTTP"},
			GroupName:          "Test Group",
			RequestParameters:  []models.Parameter{},
			ResponseParameters: []models.Parameter{},
		},
	}

	// Act
	result := GenerateHTML(apis, "en")

	// Assert - Verify basic HTML structure
	requiredTags := []string{
		"<!DOCTYPE html>",
		"<html",
		"</html>",
		"<head>",
		"</head>",
		"<body>",
		"</body>",
		"<meta charset=\"UTF-8\">",
		"<title>",
		"<style>",
		"</style>",
		"<script>",
		"</script>",
	}

	for _, tag := range requiredTags {
		if !strings.Contains(result, tag) {
			t.Errorf("Required tag or content missing: %s", tag)
		}
	}
}

// TestExportService_GenerateExampleJSON_GivenComplexNestedStructure_whenGenerate_thenCorrectStructure
func TestExportService_GenerateExampleJSON_GivenComplexNestedStructure_whenGenerate_thenCorrectStructure(t *testing.T) {
	// Arrange
	// Structure: data -> items (array) -> {id, name, tags (array) -> tag}
	level1ID := uint(1)
	level2ID := uint(2)
	level3ID := uint(3)

	params := []models.Parameter{
		{
			ID: 1, Name: "data", Type: "object", ParentID: nil,
			Children: []models.Parameter{
				{
					ID: 2, Name: "items", Type: "array", ParentID: &level1ID,
					Children: []models.Parameter{
						{ID: 3, Name: "id", Type: "number", ParentID: &level2ID, Children: []models.Parameter{}},
						{ID: 4, Name: "name", Type: "string", ParentID: &level2ID, Children: []models.Parameter{}},
						{
							ID: 5, Name: "tags", Type: "array", ParentID: &level2ID,
							Children: []models.Parameter{
								{ID: 6, Name: "tag", Type: "string", ParentID: &level3ID, Children: []models.Parameter{}},
							},
						},
					},
				},
			},
		},
	}

	// Act
	result := GenerateExampleJSON(params)

	// Assert
	data, ok := result["data"]
	if !ok {
		t.Fatal("Expected 'data' key")
	}

	dataObj, ok := data.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected data to be object, got %T", data)
	}

	items, ok := dataObj["items"]
	if !ok {
		t.Fatal("Expected 'items' key in data")
	}

	itemsArr, ok := items.([]interface{})
	if !ok {
		t.Fatalf("Expected items to be array, got %T", items)
	}

	if len(itemsArr) != 1 {
		t.Fatalf("Expected 1 item in array, got %d", len(itemsArr))
	}

	itemObj, ok := itemsArr[0].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected item to be object, got %T", itemsArr[0])
	}

	// Check for id, name, tags
	if _, ok := itemObj["id"]; !ok {
		t.Error("Expected 'id' in item")
	}

	if _, ok := itemObj["name"]; !ok {
		t.Error("Expected 'name' in item")
	}

	if _, ok := itemObj["tags"]; !ok {
		t.Error("Expected 'tags' in item")
	}

	// Verify tags is array
	tagsArr, ok := itemObj["tags"].([]interface{})
	if !ok {
		t.Fatalf("Expected tags to be array, got %T", itemObj["tags"])
	}

	if len(tagsArr) != 1 {
		t.Errorf("Expected 1 tag in tags array, got %d", len(tagsArr))
	}
}

// TestExportService_GenerateParameterHTML_GivenAllTypes_whenGenerate_thenAllBadgesPresent
func TestExportService_GenerateParameterHTML_GivenAllTypes_whenGenerate_thenAllBadgesPresent(t *testing.T) {
	// Arrange
	params := []models.Parameter{
		{ID: 1, Name: "strParam", Type: "string", Required: true, ParentID: nil, Children: []models.Parameter{}},
		{ID: 2, Name: "numParam", Type: "number", Required: true, ParentID: nil, Children: []models.Parameter{}},
		{ID: 3, Name: "boolParam", Type: "boolean", Required: true, ParentID: nil, Children: []models.Parameter{}},
		{ID: 4, Name: "arrayParam", Type: "array", Required: true, ParentID: nil, Children: []models.Parameter{}},
		{ID: 5, Name: "objectParam", Type: "object", Required: true, ParentID: nil, Children: []models.Parameter{}},
	}

	// Act
	result := GenerateParameterHTML(params, 0)

	// Assert
	expectedBadges := []string{"type-string", "type-number", "type-boolean", "type-array", "type-object"}
	for _, badge := range expectedBadges {
		if !strings.Contains(result, badge) {
			t.Errorf("Expected type badge '%s' in HTML", badge)
		}
	}
}
