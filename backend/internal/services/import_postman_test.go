package services

import (
	"strings"
	"testing"
)

// TestParsePostmanCollection covers the Postman Collection v2.0/v2.1
// importer per spec §8.3: recursive folder flattening with "/" separators,
// both request.url shapes (object with raw, plain string), both
// request.description shapes (string, {content}), the x-knot-type RPC
// marker, raw JSON bodies and the lenient fallback paths (missing names,
// empty group names, unconvertible bodies).
func TestParsePostmanCollection(t *testing.T) {
	t.Run("givenFolderHierarchy_whenParse_thenGroupsFlattenedWithSlash", func(t *testing.T) {
		// Arrange - a folder holding a direct request plus a nested
		// subfolder, and a lone top-level request that belongs to the
		// collection itself
		data := []byte(`{
			"info": {"name": "My Collection", "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"},
			"item": [
				{
					"name": "Folder",
					"item": [
						{"name": "List users", "request": {"method": "get", "url": {"raw": "/users"}}},
						{"name": "Sub", "item": [
							{"name": "Deep purge", "request": {"method": "post", "url": {"raw": "/cache/purge"}}}
						]}
					]
				},
				{"name": "Lone request", "request": {"method": "GET", "url": {"raw": "/lone"}}}
			]
		}`)

		// Act
		plan, err := ParsePostmanCollection(data)

		// Assert - three groups in first-appearance order
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if plan == nil {
			t.Fatal("Expected a plan, got nil")
		}
		wantGroups := []string{"Folder", "Folder/Sub", "My Collection"}
		if len(plan.Groups) != len(wantGroups) {
			t.Fatalf("Expected %d groups, got %d: %+v", len(wantGroups), len(plan.Groups), plan.Groups)
		}
		for i, want := range wantGroups {
			if plan.Groups[i].Name != want {
				t.Errorf("Group %d: expected name %q, got %q", i, want, plan.Groups[i].Name)
			}
		}

		// Assert - each group holds exactly its own requests
		if got := plan.Groups[0].APIs; len(got) != 1 || got[0].Name != "List users" {
			t.Errorf("Expected [List users] in group \"Folder\", got %+v", got)
		}
		if got := plan.Groups[1].APIs; len(got) != 1 || got[0].Name != "Deep purge" {
			t.Errorf("Expected [Deep purge] in group \"Folder/Sub\", got %+v", got)
		}
		if got := plan.Groups[2].APIs; len(got) != 1 || got[0].Name != "Lone request" {
			t.Errorf("Expected [Lone request] in group \"My Collection\", got %+v", got)
		}

		// Assert - defaults: HTTP type, upper-cased method, endpoint from url.raw
		api := plan.Groups[0].APIs[0]
		if api.Type != "HTTP" {
			t.Errorf("Expected default type HTTP, got %q", api.Type)
		}
		if api.Method != "GET" {
			t.Errorf("Expected upper-cased method GET, got %q", api.Method)
		}
		if api.Endpoint != "/users" {
			t.Errorf("Expected endpoint from url.raw, got %q", api.Endpoint)
		}

		// Assert - top-level slices are non-nil for safe iteration
		if plan.Warnings == nil {
			t.Error("Expected non-nil Warnings, got nil")
		}
		if api.Request == nil || api.Response == nil {
			t.Error("Expected non-nil Request and Response slices, got nil")
		}
		if len(plan.Warnings) != 0 {
			t.Errorf("Expected no warnings, got %v", plan.Warnings)
		}
	})

	t.Run("givenURLAsPlainString_whenParse_thenEndpointVerbatim", func(t *testing.T) {
		// Arrange - v2.0-style plain string url carrying a Postman
		// variable placeholder that must pass through verbatim
		data := []byte(`{
			"info": {"name": "Collection"},
			"item": [{"name": "List things", "request": {"method": "GET", "url": "{{baseUrl}}/things"}}]
		}`)

		// Act
		plan, err := ParsePostmanCollection(data)

		// Assert
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if got := plan.Groups[0].APIs[0].Endpoint; got != "{{baseUrl}}/things" {
			t.Errorf("Expected plain-string url verbatim, got %q", got)
		}
	})

	t.Run("givenURLAsObject_whenParse_thenEndpointFromRawField", func(t *testing.T) {
		// Arrange - v2.1 object url form: raw wins over host/path segments
		data := []byte(`{
			"info": {"name": "Collection"},
			"item": [{"name": "Get user", "request": {"method": "GET",
				"url": {"raw": "/users/:id", "host": ["api", "example", "com"], "path": ["users", ":id"]}}}]
		}`)

		// Act
		plan, err := ParsePostmanCollection(data)

		// Assert
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if got := plan.Groups[0].APIs[0].Endpoint; got != "/users/:id" {
			t.Errorf("Expected endpoint from url.raw, got %q", got)
		}
	})

	t.Run("givenDescriptionVariants_whenParse_thenNoteFromStringOrContent", func(t *testing.T) {
		// Arrange - description as plain string, as {content: ...} object,
		// and absent
		data := []byte(`{
			"info": {"name": "Collection"},
			"item": [
				{"name": "String note", "request": {"method": "GET", "url": "/a", "description": "Requires admin role"}},
				{"name": "Object note", "request": {"method": "GET", "url": "/b",
					"description": {"content": "Object-form note", "type": "text"}}},
				{"name": "No note", "request": {"method": "GET", "url": "/c"}}
			]
		}`)

		// Act
		plan, err := ParsePostmanCollection(data)

		// Assert
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		apis := plan.Groups[0].APIs
		if len(apis) != 3 {
			t.Fatalf("Expected 3 APIs, got %d", len(apis))
		}
		if apis[0].Note == nil || *apis[0].Note != "Requires admin role" {
			t.Errorf("Expected string-form note, got %v", apis[0].Note)
		}
		if apis[1].Note == nil || *apis[1].Note != "Object-form note" {
			t.Errorf("Expected object-form note via content, got %v", apis[1].Note)
		}
		if apis[2].Note != nil {
			t.Errorf("Expected nil note when description absent, got %q", *apis[2].Note)
		}
	})

	t.Run("givenKnotRPCMarker_whenParse_thenTypeRPCAndEmptyMethod", func(t *testing.T) {
		// Arrange - one item carrying the custom x-knot-type rpc marker
		// next to a regular request
		data := []byte(`{
			"info": {"name": "Collection"},
			"item": [
				{"name": "Purge cache", "x-knot-type": "rpc",
					"request": {"method": "POST", "url": "/cache/purge"}},
				{"name": "Normal", "request": {"method": "post", "url": "/normal"}}
			]
		}`)

		// Act
		plan, err := ParsePostmanCollection(data)

		// Assert
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		apis := plan.Groups[0].APIs
		if apis[0].Type != "RPC" || apis[0].Method != "" {
			t.Errorf("Expected RPC type with empty method, got type %q method %q", apis[0].Type, apis[0].Method)
		}
		if apis[0].Endpoint != "/cache/purge" {
			t.Errorf("Expected endpoint preserved on RPC item, got %q", apis[0].Endpoint)
		}
		if apis[1].Type != "HTTP" || apis[1].Method != "POST" {
			t.Errorf("Expected HTTP type with POST method, got type %q method %q", apis[1].Type, apis[1].Method)
		}
	})

	t.Run("givenRawJSONBody_whenParse_thenRequestParamNodes", func(t *testing.T) {
		// Arrange - raw body with a JSON object mixing primitives and a
		// nested object; ParamNodesFromJSONValue sorts keys alphabetically
		data := []byte(`{
			"info": {"name": "Collection"},
			"item": [{"name": "Create user", "request": {"method": "POST", "url": "/users",
				"body": {"mode": "raw", "raw": "{\"count\": 3, \"user\": {\"name\": \"knot\", \"admin\": false}}",
					"options": {"raw": {"language": "json"}}}}}]
		}`)

		// Act
		plan, err := ParsePostmanCollection(data)

		// Assert - request nodes inferred from the example values
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		want := []ParamNode{
			{Name: "count", Type: "number", Order: 0, Children: []ParamNode{}},
			{Name: "user", Type: "object", Order: 1, Children: []ParamNode{
				{Name: "admin", Type: "boolean", Order: 0, Children: []ParamNode{}},
				{Name: "name", Type: "string", Order: 1, Children: []ParamNode{}},
			}},
		}
		api := plan.Groups[0].APIs[0]
		assertParamNodesEqual(t, api.Request, want, "request")
		if len(api.Response) != 0 {
			t.Errorf("Expected empty response params, got %+v", api.Response)
		}
		if len(plan.Warnings) != 0 {
			t.Errorf("Expected no warnings, got %v", plan.Warnings)
		}
	})

	t.Run("givenInvalidJSONBody_whenParse_thenWarningAndEmptyParams", func(t *testing.T) {
		// Arrange - raw body that is not valid JSON
		data := []byte(`{
			"info": {"name": "Collection"},
			"item": [{"name": "Broken body", "request": {"method": "POST", "url": "/users",
				"body": {"mode": "raw", "raw": "not json {"}}}]
		}`)

		// Act
		plan, err := ParsePostmanCollection(data)

		// Assert - API still imports, body conversion degrades to a warning
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		api := plan.Groups[0].APIs[0]
		if len(api.Request) != 0 {
			t.Errorf("Expected empty request params, got %+v", api.Request)
		}
		if len(plan.Warnings) != 1 || !strings.Contains(plan.Warnings[0], "Broken body") {
			t.Errorf("Expected one warning naming the item, got %v", plan.Warnings)
		}
	})

	t.Run("givenNonRawBodyMode_whenParse_thenBodyIgnored", func(t *testing.T) {
		// Arrange - formdata body: Knot documents bodies, not form fields
		data := []byte(`{
			"info": {"name": "Collection"},
			"item": [{"name": "Upload avatar", "request": {"method": "POST", "url": "/avatar",
				"body": {"mode": "formdata", "formdata": [{"key": "file", "type": "file"}]}}}]
		}`)

		// Act
		plan, err := ParsePostmanCollection(data)

		// Assert - silently ignored, no params, no warnings
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		api := plan.Groups[0].APIs[0]
		if len(api.Request) != 0 {
			t.Errorf("Expected empty request params, got %+v", api.Request)
		}
		if len(plan.Warnings) != 0 {
			t.Errorf("Expected no warnings, got %v", plan.Warnings)
		}
	})

	t.Run("givenMissingItemName_whenParse_thenFallbackWithWarning", func(t *testing.T) {
		// Arrange - one unnamed request with a usable endpoint and one
		// without any fallback source
		data := []byte(`{
			"info": {"name": "Collection"},
			"item": [
				{"request": {"method": "GET", "url": "/no-name"}},
				{"request": {"method": "GET"}}
			]
		}`)

		// Act
		plan, err := ParsePostmanCollection(data)

		// Assert - names fall back, never a hard error
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		apis := plan.Groups[0].APIs
		if len(apis) != 2 {
			t.Fatalf("Expected 2 APIs, got %d", len(apis))
		}
		if apis[0].Name != "/no-name" {
			t.Errorf("Expected endpoint fallback name, got %q", apis[0].Name)
		}
		if apis[1].Name != "Unnamed request" {
			t.Errorf("Expected \"Unnamed request\" fallback, got %q", apis[1].Name)
		}
		if len(plan.Warnings) != 2 {
			t.Errorf("Expected 2 warnings, got %v", plan.Warnings)
		}
	})

	t.Run("givenItemWithoutRequest_whenParse_thenSkippedWithWarning", func(t *testing.T) {
		// Arrange - an item that is neither folder nor usable request
		data := []byte(`{
			"info": {"name": "Collection"},
			"item": [{"name": "Just a name"}]
		}`)

		// Act
		plan, err := ParsePostmanCollection(data)

		// Assert - nothing importable, warning explains the skip
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if len(plan.Groups) != 0 {
			t.Errorf("Expected no groups, got %+v", plan.Groups)
		}
		if len(plan.Warnings) != 1 || !strings.Contains(plan.Warnings[0], "Just a name") {
			t.Errorf("Expected one warning naming the item, got %v", plan.Warnings)
		}
	})

	t.Run("givenNoCollectionName_whenParse_thenLoneRequestSkippedWithWarning", func(t *testing.T) {
		// Arrange - a lone top-level request but no info.name to group it
		data := []byte(`{
			"info": {"schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"},
			"item": [{"name": "Orphan", "request": {"method": "GET", "url": "/orphan"}}]
		}`)

		// Act
		plan, err := ParsePostmanCollection(data)

		// Assert - empty group name skips the group with a warning
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if len(plan.Groups) != 0 {
			t.Errorf("Expected no groups, got %+v", plan.Groups)
		}
		if len(plan.Warnings) != 1 {
			t.Errorf("Expected one skip warning, got %v", plan.Warnings)
		}
	})

	t.Run("givenEmptyCollection_whenParse_thenEmptyPlan", func(t *testing.T) {
		// Arrange - valid collection with no items at all
		data := []byte(`{"info": {"name": "Empty", "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"}}`)

		// Act
		plan, err := ParsePostmanCollection(data)

		// Assert - empty but non-nil plan slices
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if plan.Groups == nil || len(plan.Groups) != 0 {
			t.Errorf("Expected empty non-nil groups, got %+v", plan.Groups)
		}
		if plan.Warnings == nil || len(plan.Warnings) != 0 {
			t.Errorf("Expected empty non-nil warnings, got %+v", plan.Warnings)
		}
	})

	t.Run("givenInvalidJSON_whenParse_thenError", func(t *testing.T) {
		// Arrange - truncated JSON
		data := []byte(`{"info": `)

		// Act
		plan, err := ParsePostmanCollection(data)

		// Assert
		if err == nil {
			t.Fatal("Expected an error for invalid JSON, got nil")
		}
		if plan != nil {
			t.Errorf("Expected nil plan on error, got %+v", plan)
		}
		if !strings.Contains(err.Error(), "Postman") {
			t.Errorf("Expected error to mention the Postman file, got %q", err.Error())
		}
	})
}
