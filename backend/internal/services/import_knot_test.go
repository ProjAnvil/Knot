package services

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	models "github.com/ProjAnvil/knot/models"
)

// TestParseKnotJSON_RoundTrip parses a document built by the real
// BuildKnotDocument builder (not hand-written JSON) and checks the resulting
// models.ImportPlan is its semantic inverse: groups, API fields, note pointers and
// full parameter trees survive the marshal/parse cycle.
func TestParseKnotJSON_RoundTrip(t *testing.T) {
	t.Run("GivenBuiltDocument_whenMarshalAndParse_thenPlanMatches", func(t *testing.T) {
		// Arrange - in-memory APIs with nested params, notes and an RPC entry
		now := time.Date(2026, 11, 10, 20, 30, 0, 0, time.UTC)
		note := "Requires admin role"
		filterID := uint(11)
		dataID := uint(31)
		apis := []APIWithParams{
			{
				API: models.API{
					ID: 1, GroupID: 100, Name: "List users", Endpoint: "/users",
					Method: "get", Type: "HTTP", Order: 0,
				},
				GroupName: "Users", GroupOrder: 1,
				RequestParameters: []models.Parameter{
					{ID: 11, APIID: 1, Name: "filter", Type: "object", Order: 0, ParentID: nil},
					{ID: 12, APIID: 1, Name: "name", Type: "string", Required: true, Order: 0, ParentID: &filterID},
				},
			},
			{
				API: models.API{
					ID: 2, GroupID: 100, Name: "Create user", Endpoint: "/users",
					Method: "post", Type: "HTTP", Order: 1, Note: &note,
				},
				GroupName: "Users", GroupOrder: 1,
			},
			{
				API: models.API{
					ID: 3, GroupID: 200, Name: "Delete item", Endpoint: "/items/{id}",
					Method: "delete", Type: "HTTP", Order: 0,
				},
				GroupName: "Items", GroupOrder: 2,
				ResponseParameters: []models.Parameter{
					{ID: 31, APIID: 3, Name: "data", Type: "object", Order: 0, ParentID: nil},
					{ID: 32, APIID: 3, Name: "item", Type: "object", Order: 0, ParentID: &dataID},
				},
			},
			{
				API: models.API{
					ID: 4, GroupID: 200, Name: "Purge cache", Endpoint: "/cache/purge",
					Method: "", Type: "RPC", Order: 1,
				},
				GroupName: "Items", GroupOrder: 2,
			},
		}
		doc := BuildKnotDocument(apis, now)
		data, err := json.Marshal(doc)
		if err != nil {
			t.Fatalf("Failed to marshal fixture document: %v", err)
		}

		// Act
		plan, err := ParseKnotJSON(data)

		// Assert - parse succeeds without warnings
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if plan == nil {
			t.Fatal("Expected a plan, got nil")
		}
		if len(plan.Warnings) != 0 {
			t.Errorf("Expected no warnings, got %v", plan.Warnings)
		}

		// Assert - groups match the built document in order
		if len(plan.Groups) != len(doc.Groups) {
			t.Fatalf("Expected %d groups, got %d", len(doc.Groups), len(plan.Groups))
		}
		for i := range doc.Groups {
			wantGroup, gotGroup := doc.Groups[i], plan.Groups[i]
			if gotGroup.Name != wantGroup.Name {
				t.Errorf("Group %d: expected name %q, got %q", i, wantGroup.Name, gotGroup.Name)
			}
			if len(gotGroup.APIs) != len(wantGroup.APIs) {
				t.Fatalf("Group %q: expected %d APIs, got %d", wantGroup.Name, len(wantGroup.APIs), len(gotGroup.APIs))
			}

			for j := range wantGroup.APIs {
				wantAPI, gotAPI := wantGroup.APIs[j], gotGroup.APIs[j]
				if gotAPI.Name != wantAPI.Name {
					t.Errorf("Group %q API %d: expected name %q, got %q", wantGroup.Name, j, wantAPI.Name, gotAPI.Name)
				}
				if gotAPI.Endpoint != wantAPI.Endpoint {
					t.Errorf("Group %q API %q: expected endpoint %q, got %q", wantGroup.Name, wantAPI.Name, wantAPI.Endpoint, gotAPI.Endpoint)
				}
				// The builder already upper-cases; parsing must preserve that.
				if gotAPI.Method != wantAPI.Method {
					t.Errorf("Group %q API %q: expected method %q, got %q", wantGroup.Name, wantAPI.Name, wantAPI.Method, gotAPI.Method)
				}
				if gotAPI.Type != wantAPI.Type {
					t.Errorf("Group %q API %q: expected type %q, got %q", wantGroup.Name, wantAPI.Name, wantAPI.Type, gotAPI.Type)
				}
				// Note pointer values: nil stays nil, non-nil carries the value.
				if (gotAPI.Note == nil) != (wantAPI.Note == nil) {
					t.Errorf("Group %q API %q: note nil-ness mismatch (want nil: %v)", wantGroup.Name, wantAPI.Name, wantAPI.Note == nil)
				} else if wantAPI.Note != nil && *gotAPI.Note != *wantAPI.Note {
					t.Errorf("Group %q API %q: expected note %q, got %q", wantGroup.Name, wantAPI.Name, *wantAPI.Note, *gotAPI.Note)
				}
				assertParamNodesEqual(t, gotAPI.Request, wantAPI.Parameters.Request, "request")
				assertParamNodesEqual(t, gotAPI.Response, wantAPI.Parameters.Response, "response")
			}
		}

		// Assert - spot-check the semantic essentials the loop above compared
		if plan.Groups[0].Name != "Users" || plan.Groups[1].Name != "Items" {
			t.Errorf("Expected group names Users/Items, got %q/%q", plan.Groups[0].Name, plan.Groups[1].Name)
		}
		if got := plan.Groups[0].APIs[0].Method; got != "GET" {
			t.Errorf("Expected upper-cased method GET, got %q", got)
		}
		if got := plan.Groups[0].APIs[1].Note; got == nil || *got != note {
			t.Errorf("Expected note %q, got %v", note, got)
		}
		if got := plan.Groups[1].APIs[1]; got.Type != "RPC" || got.Method != "" {
			t.Errorf("Expected RPC type with empty method, got type %q method %q", got.Type, got.Method)
		}
	})

	t.Run("GivenLowercaseMethodAndAbsentType_whenParse_thenNormalized", func(t *testing.T) {
		// Arrange - hand-written file with unknown extra fields (ignored),
		// a lower-case method and no type field at all
		data := []byte(`{
			"format": "knot", "version": 1, "exportedAt": "2026-11-10T12:00:00Z", "futureField": true,
			"groups": [{
				"name": "Users", "order": 1, "extra": "ignored",
				"apis": [{
					"name": "List users", "endpoint": "/users", "method": "post", "order": 0,
					"unknownApiField": 42,
					"parameters": {"request": [{"name": "q", "type": "string", "required": true, "order": 0, "children": []}], "response": []}
				}]
			}]
		}`)

		// Act
		plan, err := ParseKnotJSON(data)

		// Assert
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		api := plan.Groups[0].APIs[0]
		if api.Method != "POST" {
			t.Errorf("Expected upper-cased method POST, got %q", api.Method)
		}
		if api.Type != "HTTP" {
			t.Errorf("Expected default type HTTP, got %q", api.Type)
		}
		if len(plan.Warnings) != 0 {
			t.Errorf("Expected no warnings, got %v", plan.Warnings)
		}
	})

	t.Run("GivenUnknownType_whenParse_thenNormalizedToHTTPWithWarning", func(t *testing.T) {
		// Arrange
		data := []byte(`{
			"format": "knot", "version": 1,
			"groups": [{"name": "Users", "apis": [{"name": "List users", "endpoint": "/users", "method": "GET", "type": "grpc"}]}]
		}`)

		// Act
		plan, err := ParseKnotJSON(data)

		// Assert - unknown type does not fail; it normalizes with a warning
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if got := plan.Groups[0].APIs[0].Type; got != "HTTP" {
			t.Errorf("Expected normalized type HTTP, got %q", got)
		}
		if len(plan.Warnings) != 1 || !strings.Contains(plan.Warnings[0], "grpc") {
			t.Errorf("Expected one warning mentioning \"grpc\", got %v", plan.Warnings)
		}
	})

	t.Run("GivenWrongVersion_whenParse_thenError", func(t *testing.T) {
		// Arrange
		data := []byte(`{"format": "knot", "version": 2, "groups": []}`)

		// Act
		plan, err := ParseKnotJSON(data)

		// Assert
		if err == nil {
			t.Fatal("Expected an error for version 2, got nil")
		}
		if plan != nil {
			t.Errorf("Expected nil plan on error, got %+v", plan)
		}
		if !strings.Contains(err.Error(), "version") {
			t.Errorf("Expected error to mention the version, got %q", err.Error())
		}
	})

	t.Run("GivenEmptyGroupName_whenParse_thenError", func(t *testing.T) {
		// Arrange
		data := []byte(`{"format": "knot", "version": 1, "groups": [{"name": "", "apis": []}]}`)

		// Act
		plan, err := ParseKnotJSON(data)

		// Assert
		if err == nil {
			t.Fatal("Expected an error for an empty group name, got nil")
		}
		if plan != nil {
			t.Errorf("Expected nil plan on error, got %+v", plan)
		}
	})

	t.Run("GivenEmptyAPIName_whenParse_thenError", func(t *testing.T) {
		// Arrange
		data := []byte(`{"format": "knot", "version": 1, "groups": [{"name": "Users", "apis": [{"name": "", "endpoint": "/users", "method": "GET"}]}]}`)

		// Act
		plan, err := ParseKnotJSON(data)

		// Assert
		if err == nil {
			t.Fatal("Expected an error for an empty API name, got nil")
		}
		if plan != nil {
			t.Errorf("Expected nil plan on error, got %+v", plan)
		}
	})
}

// assertParamNodesEqual recursively compares two parameter trees node by
// node: name, type, required flag, order, description pointer values and
// children.
func assertParamNodesEqual(t *testing.T, got, want []models.ParamNode, path string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: expected %d nodes, got %d", path, len(want), len(got))
	}
	for i := range want {
		gotNode, wantNode := got[i], want[i]
		childPath := path + "[" + wantNode.Name + "].children"
		if gotNode.Name != wantNode.Name {
			t.Errorf("%s[%d]: expected name %q, got %q", path, i, wantNode.Name, gotNode.Name)
		}
		if gotNode.Type != wantNode.Type {
			t.Errorf("%s[%d] %q: expected type %q, got %q", path, i, wantNode.Name, wantNode.Type, gotNode.Type)
		}
		if gotNode.Required != wantNode.Required {
			t.Errorf("%s[%d] %q: expected required %v, got %v", path, i, wantNode.Name, wantNode.Required, gotNode.Required)
		}
		if gotNode.Order != wantNode.Order {
			t.Errorf("%s[%d] %q: expected order %d, got %d", path, i, wantNode.Name, wantNode.Order, gotNode.Order)
		}
		if (gotNode.Description == nil) != (wantNode.Description == nil) {
			t.Errorf("%s[%d] %q: description nil-ness mismatch (want nil: %v)", path, i, wantNode.Name, wantNode.Description == nil)
		} else if wantNode.Description != nil && *gotNode.Description != *wantNode.Description {
			t.Errorf("%s[%d] %q: expected description %q, got %q", path, i, wantNode.Name, *wantNode.Description, *gotNode.Description)
		}
		assertParamNodesEqual(t, gotNode.Children, wantNode.Children, childPath)
	}
}
