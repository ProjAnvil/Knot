package models

import (
	"encoding/json"
	"testing"
)

// TestTreeFromParams covers converting flat ordered parameter rows into a nested ParamNode tree.
func TestTreeFromParams(t *testing.T) {
	t.Run("GivenEmptyList_whenConvert_thenEmptyResult", func(t *testing.T) {
		// Arrange
		params := []Parameter{}

		// Act
		result := TreeFromParams(params)

		// Assert
		if len(result) != 0 {
			t.Errorf("Expected empty result for empty input, got %d items", len(result))
		}
	})

	t.Run("GivenFlatRows_whenConvert_thenRootsOnly", func(t *testing.T) {
		// Arrange
		params := []Parameter{
			{ID: 1, Name: "page", Type: "number", Required: false, Order: 0, ParentID: nil},
			{ID: 2, Name: "keyword", Type: "string", Required: true, Order: 1, ParentID: nil},
		}

		// Act
		result := TreeFromParams(params)

		// Assert
		if len(result) != 2 {
			t.Fatalf("Expected 2 root nodes, got %d", len(result))
		}
		if result[0].Name != "page" || result[1].Name != "keyword" {
			t.Errorf("Expected root order preserved (page, keyword), got (%s, %s)", result[0].Name, result[1].Name)
		}
		for i, node := range result {
			if len(node.Children) != 0 {
				t.Errorf("Node %d should have no children, got %d", i, len(node.Children))
			}
		}
	})

	t.Run("GivenParentIDLinks_whenConvert_thenNestedChildren", func(t *testing.T) {
		// Arrange
		userID := uint(1)
		addressID := uint(4)
		desc := "User container"
		params := []Parameter{
			{ID: 1, Name: "user", Type: "object", Required: true, Description: &desc, Order: 0, ParentID: nil},
			{ID: 2, Name: "name", Type: "string", Required: true, Order: 0, ParentID: &userID},
			{ID: 3, Name: "age", Type: "number", Required: false, Order: 1, ParentID: &userID},
			{ID: 4, Name: "address", Type: "object", Required: false, Order: 2, ParentID: &userID},
			{ID: 5, Name: "city", Type: "string", Required: false, Order: 0, ParentID: &addressID},
		}

		// Act
		result := TreeFromParams(params)

		// Assert
		if len(result) != 1 {
			t.Fatalf("Expected 1 root node, got %d", len(result))
		}
		root := result[0]
		if root.Name != "user" || root.Type != "object" || !root.Required {
			t.Errorf("Unexpected root node: %+v", root)
		}
		if root.Description == nil || *root.Description != "User container" {
			t.Errorf("Expected description carried over, got %v", root.Description)
		}
		if len(root.Children) != 3 {
			t.Fatalf("Expected 3 children under user, got %d", len(root.Children))
		}
		// Child order must follow the input row order
		if root.Children[0].Name != "name" || root.Children[1].Name != "age" || root.Children[2].Name != "address" {
			t.Errorf("Expected child order (name, age, address), got (%s, %s, %s)",
				root.Children[0].Name, root.Children[1].Name, root.Children[2].Name)
		}
		address := root.Children[2]
		if len(address.Children) != 1 || address.Children[0].Name != "city" {
			t.Errorf("Expected nested address/city structure, got %+v", address.Children)
		}
		// Order field must be carried over
		if root.Children[1].Order != 1 {
			t.Errorf("Expected order field 1 carried over for age, got %d", root.Children[1].Order)
		}
	})

	t.Run("GivenMultipleRootsWithChildren_whenConvert_thenEachTreeBuilt", func(t *testing.T) {
		// Arrange
		filterID := uint(2)
		params := []Parameter{
			{ID: 1, Name: "sort", Type: "string", Order: 0, ParentID: nil},
			{ID: 2, Name: "filter", Type: "object", Order: 1, ParentID: nil},
			{ID: 3, Name: "field", Type: "string", Order: 0, ParentID: &filterID},
		}

		// Act
		result := TreeFromParams(params)

		// Assert
		if len(result) != 2 {
			t.Fatalf("Expected 2 root nodes, got %d", len(result))
		}
		if result[0].Name != "sort" || len(result[0].Children) != 0 {
			t.Errorf("Unexpected first root: %+v", result[0])
		}
		if result[1].Name != "filter" || len(result[1].Children) != 1 || result[1].Children[0].Name != "field" {
			t.Errorf("Unexpected second root: %+v", result[1])
		}
	})
}

// TestParamNodesFromJSONValue covers best-effort conversion of decoded JSON values into ParamNodes.
func TestParamNodesFromJSONValue(t *testing.T) {
	t.Run("GivenDecodedJSON_whenConvert_thenTypedNodesWithSortedKeys", func(t *testing.T) {
		// Arrange
		raw := `{
			"active": true,
			"age": 42,
			"name": "john",
			"tags": ["a", "b"],
			"profile": {"city": "SF", "zip": 94105}
		}`
		var v interface{}
		if err := json.Unmarshal([]byte(raw), &v); err != nil {
			t.Fatalf("Failed to unmarshal test JSON: %v", err)
		}

		// Act
		result := ParamNodesFromJSONValue(v)

		// Assert
		if len(result) != 5 {
			t.Fatalf("Expected 5 nodes, got %d", len(result))
		}
		// Map keys must be sorted for deterministic order
		expectedOrder := []string{"active", "age", "name", "profile", "tags"}
		for i, want := range expectedOrder {
			if result[i].Name != want {
				t.Fatalf("Expected node %d to be %q, got %q", i, want, result[i].Name)
			}
			if result[i].Order != i {
				t.Errorf("Expected node %q order %d, got %d", want, i, result[i].Order)
			}
		}
		byName := make(map[string]ParamNode, len(result))
		for _, node := range result {
			byName[node.Name] = node
		}
		if byName["name"].Type != "string" {
			t.Errorf("Expected string type for name, got %q", byName["name"].Type)
		}
		if byName["age"].Type != "number" {
			t.Errorf("Expected number type for age, got %q", byName["age"].Type)
		}
		if byName["active"].Type != "boolean" {
			t.Errorf("Expected boolean type for active, got %q", byName["active"].Type)
		}
		if byName["tags"].Type != "array" {
			t.Errorf("Expected array type for tags, got %q", byName["tags"].Type)
		}
		if byName["profile"].Type != "object" {
			t.Errorf("Expected object type for profile, got %q", byName["profile"].Type)
		}
	})

	t.Run("GivenNestedObjects_whenConvert_thenObjectHasChildren", func(t *testing.T) {
		// Arrange
		raw := `{"user": {"name": "john", "address": {"city": "SF"}}}`
		var v interface{}
		if err := json.Unmarshal([]byte(raw), &v); err != nil {
			t.Fatalf("Failed to unmarshal test JSON: %v", err)
		}

		// Act
		result := ParamNodesFromJSONValue(v)

		// Assert
		if len(result) != 1 {
			t.Fatalf("Expected 1 node, got %d", len(result))
		}
		user := result[0]
		if user.Type != "object" {
			t.Fatalf("Expected object type, got %q", user.Type)
		}
		if len(user.Children) != 2 {
			t.Fatalf("Expected 2 children under user, got %d", len(user.Children))
		}
		if user.Children[0].Name != "address" || user.Children[0].Type != "object" {
			t.Errorf("Expected first sorted child address/object, got %q/%q", user.Children[0].Name, user.Children[0].Type)
		}
		if len(user.Children[0].Children) != 1 || user.Children[0].Children[0].Name != "city" {
			t.Errorf("Expected nested address/city, got %+v", user.Children[0].Children)
		}
		if user.Children[1].Name != "name" || user.Children[1].Type != "string" {
			t.Errorf("Expected second sorted child name/string, got %q/%q", user.Children[1].Name, user.Children[1].Type)
		}
	})

	t.Run("GivenArrays_whenConvert_thenElementShapeAsChildren", func(t *testing.T) {
		// Arrange
		raw := `{
			"ids": [1, 2, 3],
			"users": [{"id": 1, "name": "john"}],
			"empty": []
		}`
		var v interface{}
		if err := json.Unmarshal([]byte(raw), &v); err != nil {
			t.Fatalf("Failed to unmarshal test JSON: %v", err)
		}

		// Act
		result := ParamNodesFromJSONValue(v)

		// Assert
		byName := make(map[string]ParamNode, len(result))
		for _, node := range result {
			byName[node.Name] = node
		}
		ids := byName["ids"]
		if ids.Type != "array" || len(ids.Children) != 1 || ids.Children[0].Type != "number" {
			t.Errorf("Expected ids array with number item child, got %+v", ids)
		}
		users := byName["users"]
		if users.Type != "array" || len(users.Children) != 2 {
			t.Fatalf("Expected users array with 2 element children, got %+v", users)
		}
		if users.Children[0].Name != "id" || users.Children[0].Type != "number" {
			t.Errorf("Expected id/number child, got %q/%q", users.Children[0].Name, users.Children[0].Type)
		}
		if users.Children[1].Name != "name" || users.Children[1].Type != "string" {
			t.Errorf("Expected name/string child, got %q/%q", users.Children[1].Name, users.Children[1].Type)
		}
		empty := byName["empty"]
		if empty.Type != "array" || len(empty.Children) != 0 {
			t.Errorf("Expected empty array without children, got %+v", empty)
		}
	})

	t.Run("GivenNonObjectValue_whenConvert_thenEmptyResult", func(t *testing.T) {
		// Arrange - a top-level JSON body must be an object to yield parameters
		var v interface{} = "not an object"

		// Act
		result := ParamNodesFromJSONValue(v)

		// Assert
		if len(result) != 0 {
			t.Errorf("Expected empty result for non-object input, got %d items", len(result))
		}
	})
}

// TestCountNodes covers counting all nodes in a ParamNode tree.
func TestCountNodes(t *testing.T) {
	t.Run("GivenEmptyTree_whenCount_thenZero", func(t *testing.T) {
		// Act
		count := CountNodes(nil)

		// Assert
		if count != 0 {
			t.Errorf("Expected 0 for empty tree, got %d", count)
		}
	})

	t.Run("GivenNestedTree_whenCount_thenAllDescendantsCounted", func(t *testing.T) {
		// Arrange
		// 3 roots: a (1), b with 2 children (3), c with a child that has a child (3) = 7
		nodes := []ParamNode{
			{Name: "a", Type: "string"},
			{Name: "b", Type: "object", Children: []ParamNode{
				{Name: "b1", Type: "string"},
				{Name: "b2", Type: "number"},
			}},
			{Name: "c", Type: "object", Children: []ParamNode{
				{Name: "c1", Type: "object", Children: []ParamNode{
					{Name: "c1x", Type: "boolean"},
				}},
			}},
		}

		// Act
		count := CountNodes(nodes)

		// Assert
		if count != 7 {
			t.Errorf("Expected 7 nodes in tree, got %d", count)
		}
	})
}
