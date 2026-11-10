package services

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"
	"time"

	models "github.com/ProjAnvil/knot/models"
)

// uuidV4Pattern matches an RFC 4122 version 4 UUID in its canonical
// 8-4-4-4-12 lower-case hex form.
var uuidV4Pattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

// marshalExampleJSON marshals an example value with the same formatting
// the builder uses, so tests compare against verbatim builder output.
func marshalExampleJSON(t *testing.T, v interface{}) string {
	t.Helper()
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal example value: %v", err)
	}
	return string(raw)
}

// TestBuildPostmanCollection covers building a Postman Collection v2.1
// document from APIs with parameters.
func TestBuildPostmanCollection(t *testing.T) {
	// Fixed wall-clock time in a non-UTC zone so the UTC conversion is verified.
	now := time.Date(2026, 11, 10, 20, 30, 0, 0, time.FixedZone("UTC+8", 8*60*60))

	t.Run("GivenGroupedAPIs_whenBuild_thenFolderPerGroupAndRequestItems", func(t *testing.T) {
		// Arrange
		note := "Requires **admin** role"
		filterID := uint(11)
		requestParams := []models.Parameter{
			{ID: 11, Name: "filter", Type: "object", Order: 0, ParentID: nil},
			{ID: 12, Name: "name", Type: "string", Required: true, Order: 0, ParentID: &filterID},
		}
		responseParams := []models.Parameter{
			{ID: 31, Name: "success", Type: "boolean", Order: 0, ParentID: nil},
		}
		apis := []APIWithParams{
			{
				API: models.API{
					ID:       1,
					GroupID:  100,
					Name:     "List users",
					Endpoint: "/users",
					Method:   "get",
					Type:     "HTTP",
					Note:     &note,
				},
				GroupName:         "Users",
				GroupOrder:        1,
				RequestParameters: requestParams,
			},
			{
				API: models.API{
					ID:       3,
					GroupID:  200,
					Name:     "Search items",
					Endpoint: "/items.search",
					Method:   "",
					Type:     "RPC",
				},
				GroupName:          "Items",
				GroupOrder:         2,
				ResponseParameters: responseParams,
			},
		}

		// Act
		collection := BuildPostmanCollection(apis, now)

		// Assert - info block
		if collection == nil {
			t.Fatal("Expected a collection, got nil")
		}
		if collection.Info.Name != "Knot Export" {
			t.Errorf("Expected info name %q, got %q", "Knot Export", collection.Info.Name)
		}
		if want := "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"; collection.Info.Schema != want {
			t.Errorf("Expected info schema %q, got %q", want, collection.Info.Schema)
		}
		if !uuidV4Pattern.MatchString(collection.Info.PostmanID) {
			t.Errorf("Expected _postman_id to match the UUID v4 pattern, got %q", collection.Info.PostmanID)
		}
		if want := "Exported from Knot at " + now.UTC().Format(time.RFC3339); collection.Info.Description != want {
			t.Errorf("Expected info description %q, got %q", want, collection.Info.Description)
		}

		// Assert - one folder per group in first-appearance order
		if len(collection.Item) != 2 {
			t.Fatalf("Expected 2 folder items, got %d", len(collection.Item))
		}
		users, items := collection.Item[0], collection.Item[1]
		if users.Name != "Users" || items.Name != "Items" {
			t.Errorf("Expected folders Users,Items for this input order, got %q,%q", users.Name, items.Name)
		}
		if len(users.Item) != 1 || len(items.Item) != 1 {
			t.Fatalf("Expected 1 leaf per folder, got %d and %d", len(users.Item), len(items.Item))
		}

		// Assert - HTTP leaf: upper-cased method, raw URL, empty header array,
		// note as description, raw JSON body from request params, no response
		leaf := users.Item[0]
		if leaf.Name != "List users" {
			t.Errorf("Expected leaf name %q, got %q", "List users", leaf.Name)
		}
		if leaf.XKnotType != "" {
			t.Errorf("Expected no x-knot-type on HTTP item, got %q", leaf.XKnotType)
		}
		if leaf.Request == nil {
			t.Fatal("Expected a request on the leaf item, got nil")
		}
		if leaf.Request.Method != "GET" {
			t.Errorf("Expected upper-cased method %q, got %q", "GET", leaf.Request.Method)
		}
		if leaf.Request.URL.Raw != "/users" {
			t.Errorf("Expected url.raw %q, got %q", "/users", leaf.Request.URL.Raw)
		}
		if leaf.Request.Header == nil || len(leaf.Request.Header) != 0 {
			t.Errorf("Expected empty non-nil header array, got %v", leaf.Request.Header)
		}
		if leaf.Request.Description != note {
			t.Errorf("Expected request description %q, got %q", note, leaf.Request.Description)
		}
		if leaf.Request.Body == nil {
			t.Fatal("Expected a body from request params, got nil")
		}
		if leaf.Request.Body.Mode != "raw" {
			t.Errorf("Expected body mode %q, got %q", "raw", leaf.Request.Body.Mode)
		}
		wantBody := marshalExampleJSON(t, GenerateExampleJSON(BuildParameterTree(requestParams)))
		if leaf.Request.Body.Raw != wantBody {
			t.Errorf("Expected body raw %q, got %q", wantBody, leaf.Request.Body.Raw)
		}
		if language := leaf.Request.Body.Options.Raw.Language; language != "json" {
			t.Errorf("Expected body options.raw.language %q, got %q", "json", language)
		}
		if leaf.Response != nil {
			t.Errorf("Expected response omitted without response params, got %v", leaf.Response)
		}

		// Assert - RPC leaf: POST method, x-knot-type marker, body omitted,
		// example response from response params
		rpcLeaf := items.Item[0]
		if rpcLeaf.XKnotType != "rpc" {
			t.Errorf("Expected x-knot-type %q on RPC item, got %q", "rpc", rpcLeaf.XKnotType)
		}
		if rpcLeaf.Request == nil {
			t.Fatal("Expected a request on the RPC item, got nil")
		}
		if rpcLeaf.Request.Method != "POST" {
			t.Errorf("Expected RPC method %q, got %q", "POST", rpcLeaf.Request.Method)
		}
		if rpcLeaf.Request.Body != nil {
			t.Errorf("Expected body omitted without request params, got %v", rpcLeaf.Request.Body)
		}
		if len(rpcLeaf.Response) != 1 {
			t.Fatalf("Expected 1 example response, got %d", len(rpcLeaf.Response))
		}
		if name := rpcLeaf.Response[0].Name; name != "Example 200" {
			t.Errorf("Expected response name %q, got %q", "Example 200", name)
		}
		wantResponse := marshalExampleJSON(t, GenerateExampleJSON(BuildParameterTree(responseParams)))
		if rpcLeaf.Response[0].Body != wantResponse {
			t.Errorf("Expected response body %q, got %q", wantResponse, rpcLeaf.Response[0].Body)
		}
	})

	t.Run("GivenGroupedAPIs_whenMarshal_thenPostmanKeysAndOmissions", func(t *testing.T) {
		// Arrange - one HTTP API with request params, one RPC API with response
		// params, so every optional key has both a present and an absent case
		note := "note"
		apis := []APIWithParams{
			{
				API: models.API{
					ID: 1, GroupID: 100, Name: "List users",
					Endpoint: "/users", Method: "get", Type: "HTTP", Note: &note,
				},
				GroupName: "Users",
				RequestParameters: []models.Parameter{
					{ID: 11, Name: "name", Type: "string", Order: 0, ParentID: nil},
				},
			},
			{
				API: models.API{
					ID: 2, GroupID: 100, Name: "Search items",
					Endpoint: "/items.search", Method: "", Type: "RPC",
				},
				GroupName: "Users",
				ResponseParameters: []models.Parameter{
					{ID: 31, Name: "success", Type: "boolean", Order: 0, ParentID: nil},
				},
			},
		}

		// Act
		raw, err := json.Marshal(BuildPostmanCollection(apis, now))
		if err != nil {
			t.Fatalf("Failed to marshal collection: %v", err)
		}
		var doc map[string]interface{}
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatalf("Failed to unmarshal collection: %v", err)
		}

		// Assert - info keys under their Postman names
		info, ok := doc["info"].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected info object, got %T", doc["info"])
		}
		for _, key := range []string{"name", "_postman_id", "schema"} {
			if _, has := info[key]; !has {
				t.Errorf("Expected info key %q in serialized collection", key)
			}
		}

		// Assert - folder and leaf structure
		topItems, ok := doc["item"].([]interface{})
		if !ok || len(topItems) != 1 {
			t.Fatalf("Expected 1 top-level folder, got %v", doc["item"])
		}
		folder, _ := topItems[0].(map[string]interface{})
		if _, has := folder["request"]; has {
			t.Error("Expected no request key on folder items")
		}
		leaves, _ := folder["item"].([]interface{})
		if len(leaves) != 2 {
			t.Fatalf("Expected 2 leaves in folder, got %d", len(leaves))
		}
		httpLeaf, _ := leaves[0].(map[string]interface{})
		rpcLeaf, _ := leaves[1].(map[string]interface{})

		// Assert - header serializes as an empty array, not null
		httpRequest, _ := httpLeaf["request"].(map[string]interface{})
		header, ok := httpRequest["header"].([]interface{})
		if !ok || len(header) != 0 {
			t.Errorf("Expected request.header to serialize as [], got %v", httpRequest["header"])
		}

		// Assert - body present with raw mode on the HTTP leaf
		body, _ := httpRequest["body"].(map[string]interface{})
		if body["mode"] != "raw" {
			t.Errorf("Expected serialized body mode raw, got %v", body["mode"])
		}
		if _, err := json.Marshal(body["raw"]); err != nil {
			t.Errorf("Expected body raw to be a JSON string: %v", err)
		}

		// Assert - x-knot-type only on the RPC item; response only there too
		if _, has := httpLeaf["x-knot-type"]; has {
			t.Error("Expected no x-knot-type key on HTTP item")
		}
		if rpcLeaf["x-knot-type"] != "rpc" {
			t.Errorf("Expected serialized x-knot-type rpc, got %v", rpcLeaf["x-knot-type"])
		}
		rpcRequest, _ := rpcLeaf["request"].(map[string]interface{})
		if _, has := rpcRequest["body"]; has {
			t.Error("Expected no body key on RPC request without request params")
		}
		responses, _ := rpcLeaf["response"].([]interface{})
		if len(responses) != 1 {
			t.Fatalf("Expected 1 serialized response, got %d", len(responses))
		}
		response, _ := responses[0].(map[string]interface{})
		if response["name"] != "Example 200" {
			t.Errorf("Expected serialized response name Example 200, got %v", response["name"])
		}
		if !strings.Contains(response["body"].(string), `"success"`) {
			t.Errorf("Expected serialized response body from response params, got %v", response["body"])
		}
		if _, has := httpLeaf["response"]; has {
			t.Error("Expected no response key on leaf without response params")
		}
	})

	t.Run("GivenNoAPIs_whenBuild_thenNoItems", func(t *testing.T) {
		// Act
		collection := BuildPostmanCollection(nil, now)

		// Assert - empty item array, info still fully populated
		if collection == nil {
			t.Fatal("Expected a collection, got nil")
		}
		if collection.Item == nil || len(collection.Item) != 0 {
			t.Errorf("Expected initialized empty item slice, got %v", collection.Item)
		}
		if !uuidV4Pattern.MatchString(collection.Info.PostmanID) {
			t.Errorf("Expected _postman_id to match the UUID v4 pattern, got %q", collection.Info.PostmanID)
		}
	})
}

// TestNewPostmanID covers the deterministic instant-derived generator used
// for info._postman_id: same instant → same well-formed UUID, different
// instants → different UUIDs.
func TestNewPostmanID(t *testing.T) {
	t.Run("GivenSameInstant_whenGenerateTwice_thenIdenticalValidV4", func(t *testing.T) {
		now := time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC)
		first := newPostmanID(now)
		second := newPostmanID(now)

		if first != second {
			t.Fatalf("Expected a deterministic ID for the same instant, got %q and %q", first, second)
		}
		if !uuidV4Pattern.MatchString(first) {
			t.Fatalf("UUID %q does not match the v4 pattern (version/variant bits wrong?)", first)
		}
	})

	t.Run("GivenDifferentSeconds_whenGenerate_thenAllValidV4AndUnique", func(t *testing.T) {
		base := time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC)
		seen := make(map[string]bool, 100)
		for i := 0; i < 100; i++ {
			id := newPostmanID(base.Add(time.Duration(i) * time.Second))
			if !uuidV4Pattern.MatchString(id) {
				t.Fatalf("UUID %q does not match the v4 pattern (version/variant bits wrong?)", id)
			}
			if seen[id] {
				t.Fatalf("UUID %q generated twice at second %d", id, i)
			}
			seen[id] = true
		}
		if len(seen) != 100 {
			t.Errorf("Expected 100 unique UUIDs, got %d", len(seen))
		}
	})
}
