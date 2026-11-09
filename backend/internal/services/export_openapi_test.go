package services

import (
	"encoding/json"
	"testing"

	"github.com/ProjAnvil/knot/backend/internal/models"
)

// TestNodesToJSONSchema covers converting ParamNode forests into JSON Schema
// objects: object/array/leaf type mapping, required name lists, the
// objectRequired gate, and description carrying.
func TestNodesToJSONSchema(t *testing.T) {
	filterDesc := "Filter criteria"
	nameDesc := "User name"
	nodes := []ParamNode{
		{
			Name:        "filter",
			Type:        "object",
			Description: &filterDesc,
			Order:       0,
			Children: []ParamNode{
				{Name: "name", Type: "string", Required: true, Description: &nameDesc, Order: 0},
				{Name: "page", Type: "number", Order: 1},
			},
		},
		{
			Name:  "tags",
			Type:  "array",
			Order: 1,
			Children: []ParamNode{
				{Name: "item", Type: "string", Order: 0},
			},
		},
		{Name: "active", Type: "boolean", Required: true, Order: 2},
	}

	t.Run("GivenMixedForest_whenToSchema_thenObjectPropertiesAndRequired", func(t *testing.T) {
		// Act
		schema := NodesToJSONSchema(nodes, true)

		// Assert - the forest forms an implicit object with the nodes as properties
		if schema["type"] != "object" {
			t.Fatalf("Expected type %q, got %v", "object", schema["type"])
		}
		properties, ok := schema["properties"].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected properties map, got %T", schema["properties"])
		}
		if len(properties) != 3 {
			t.Fatalf("Expected 3 properties, got %d", len(properties))
		}

		// Assert - object node: properties plus required listing only required children
		filter, ok := properties["filter"].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected filter schema map, got %T", properties["filter"])
		}
		if filter["type"] != "object" {
			t.Errorf("Expected filter type %q, got %v", "object", filter["type"])
		}
		if filter["description"] != filterDesc {
			t.Errorf("Expected filter description %q, got %v", filterDesc, filter["description"])
		}
		filterProperties, ok := filter["properties"].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected filter properties map, got %T", filter["properties"])
		}
		nameSchema, ok := filterProperties["name"].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected name schema map, got %T", filterProperties["name"])
		}
		if nameSchema["type"] != "string" {
			t.Errorf("Expected name type %q, got %v", "string", nameSchema["type"])
		}
		if nameSchema["description"] != nameDesc {
			t.Errorf("Expected name description %q, got %v", nameDesc, nameSchema["description"])
		}
		pageSchema, ok := filterProperties["page"].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected page schema map, got %T", filterProperties["page"])
		}
		if pageSchema["type"] != "number" {
			t.Errorf("Expected page type %q, got %v", "number", pageSchema["type"])
		}
		filterRequired, ok := filter["required"].([]string)
		if !ok {
			t.Fatalf("Expected filter required []string, got %T", filter["required"])
		}
		if len(filterRequired) != 1 || filterRequired[0] != "name" {
			t.Errorf("Expected filter required [%q], got %v", "name", filterRequired)
		}

		// Assert - array node: {type: array, items} with the leaf child schema
		tags, ok := properties["tags"].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected tags schema map, got %T", properties["tags"])
		}
		if tags["type"] != "array" {
			t.Errorf("Expected tags type %q, got %v", "array", tags["type"])
		}
		items, ok := tags["items"].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected tags items map, got %T", tags["items"])
		}
		if items["type"] != "string" {
			t.Errorf("Expected items type %q, got %v", "string", items["type"])
		}

		// Assert - leaf node maps directly
		active, ok := properties["active"].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected active schema map, got %T", properties["active"])
		}
		if active["type"] != "boolean" {
			t.Errorf("Expected active type %q, got %v", "boolean", active["type"])
		}

		// Assert - wrapper required lists only the required root names
		required, ok := schema["required"].([]string)
		if !ok {
			t.Fatalf("Expected required []string, got %T", schema["required"])
		}
		if len(required) != 1 || required[0] != "active" {
			t.Errorf("Expected required [%q], got %v", "active", required)
		}
	})

	t.Run("GivenObjectNotRequired_whenToSchema_thenRequiredOmitted", func(t *testing.T) {
		// Act
		schema := NodesToJSONSchema(nodes, false)

		// Assert - the required key is gated by objectRequired
		if _, exists := schema["required"]; exists {
			t.Error("Expected no required key when object is not required")
		}
	})

	t.Run("GivenNoRequiredNodes_whenToSchema_thenRequiredOmitted", func(t *testing.T) {
		// Arrange
		optional := []ParamNode{{Name: "q", Type: "string", Order: 0}}

		// Act
		schema := NodesToJSONSchema(optional, true)

		// Assert - no required member means no required array
		if _, exists := schema["required"]; exists {
			t.Error("Expected no required key when no node is required")
		}
	})
}

// TestBuildOpenAPIDocument covers building an OpenAPI 3.1 document from APIs
// with parameters: envelope, tags, path/method keying, operation fields,
// request/response schemas, and the serialized JSON shape.
func TestBuildOpenAPIDocument(t *testing.T) {
	// Arrange
	note := "Requires **admin** role"
	nameDesc := "User name"
	bodyID := uint(11)
	dataID := uint(21)
	apis := []APIWithParams{
		{
			API: models.API{
				ID:       1,
				GroupID:  100,
				Name:     "Create User!",
				Endpoint: "/users/{id}",
				Method:   "GET",
				Type:     "HTTP",
				Order:    0,
				Note:     &note,
			},
			GroupName:  "Users",
			GroupOrder: 1,
			RequestParameters: []models.Parameter{
				{ID: 11, APIID: 1, ParamType: "request", Name: "body", Type: "object", Required: true, Order: 0, ParentID: nil},
				{ID: 12, APIID: 1, ParamType: "request", Name: "name", Type: "string", Required: true, Description: &nameDesc, Order: 0, ParentID: &bodyID},
			},
			ResponseParameters: []models.Parameter{
				{ID: 21, APIID: 1, ParamType: "response", Name: "data", Type: "object", Required: true, Order: 0, ParentID: nil},
				{ID: 22, APIID: 1, ParamType: "response", Name: "id", Type: "number", Required: true, Order: 0, ParentID: &dataID},
			},
		},
	}

	t.Run("GivenAPIWithNoteAndParams_whenBuild_thenEnvelopePathOperation", func(t *testing.T) {
		// Act
		doc := BuildOpenAPIDocument(apis)

		// Assert - document envelope
		if doc == nil {
			t.Fatal("Expected a document, got nil")
		}
		if doc.OpenAPI != "3.1.0" {
			t.Errorf("Expected openapi %q, got %q", "3.1.0", doc.OpenAPI)
		}
		if doc.Info.Title != "Knot Export" {
			t.Errorf("Expected info.title %q, got %q", "Knot Export", doc.Info.Title)
		}
		if doc.Info.Version != "1.0.0" {
			t.Errorf("Expected info.version %q, got %q", "1.0.0", doc.Info.Version)
		}

		// Assert - one tag per group in first-appearance order
		if len(doc.Tags) != 1 || doc.Tags[0].Name != "Users" {
			t.Errorf("Expected tags [Users], got %+v", doc.Tags)
		}

		// Assert - path keyed by endpoint, operation keyed by lower-cased method
		item, ok := doc.Paths["/users/{id}"]
		if !ok || item == nil {
			t.Fatalf("Expected path item for /users/{id}, got %v", doc.Paths)
		}
		op := item.Get
		if op == nil {
			t.Fatal("Expected a get operation on /users/{id}")
		}
		if item.Post != nil {
			t.Error("Expected no post operation on /users/{id}")
		}

		// Assert - operation fields
		if op.Summary != "Create User!" {
			t.Errorf("Expected summary %q, got %q", "Create User!", op.Summary)
		}
		if op.Description != note {
			t.Errorf("Expected description %q, got %q", note, op.Description)
		}
		if op.OperationID != "create-user" {
			t.Errorf("Expected operationId %q, got %q", "create-user", op.OperationID)
		}
		if len(op.Tags) != 1 || op.Tags[0] != "Users" {
			t.Errorf("Expected operation tags [Users], got %v", op.Tags)
		}
		if op.XKnotType != "" {
			t.Errorf("Expected no x-knot-type on HTTP operation, got %q", op.XKnotType)
		}

		// Assert - requestBody required because a root request node is required
		if op.RequestBody == nil {
			t.Fatal("Expected a requestBody")
		}
		if !op.RequestBody.Required {
			t.Error("Expected requestBody.required true when a root request node is required")
		}
		requestContent, ok := op.RequestBody.Content["application/json"]
		if !ok {
			t.Fatal("Expected application/json request content")
		}
		requestSchema := requestContent.Schema
		if requestSchema["type"] != "object" {
			t.Errorf("Expected request schema type %q, got %v", "object", requestSchema["type"])
		}
		requestProperties, ok := requestSchema["properties"].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected request properties map, got %T", requestSchema["properties"])
		}
		body, ok := requestProperties["body"].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected body schema map, got %T", requestProperties["body"])
		}
		if body["type"] != "object" {
			t.Errorf("Expected body type %q, got %v", "object", body["type"])
		}
		bodyRequired, ok := body["required"].([]string)
		if !ok {
			t.Fatalf("Expected body required []string, got %T", body["required"])
		}
		if len(bodyRequired) != 1 || bodyRequired[0] != "name" {
			t.Errorf("Expected body required [%q], got %v", "name", bodyRequired)
		}
		bodyProperties, ok := body["properties"].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected body properties map, got %T", body["properties"])
		}
		nameSchema, ok := bodyProperties["name"].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected name schema map, got %T", bodyProperties["name"])
		}
		if nameSchema["description"] != nameDesc {
			t.Errorf("Expected name description %q, got %v", nameDesc, nameSchema["description"])
		}
		wrapperRequired, ok := requestSchema["required"].([]string)
		if !ok {
			t.Fatalf("Expected request schema required []string, got %T", requestSchema["required"])
		}
		if len(wrapperRequired) != 1 || wrapperRequired[0] != "body" {
			t.Errorf("Expected request schema required [%q], got %v", "body", wrapperRequired)
		}

		// Assert - response tree mapped to responses["200"]
		response, ok := op.Responses["200"]
		if !ok || response == nil {
			t.Fatalf("Expected a 200 response, got %v", op.Responses)
		}
		if response.Description != "OK" {
			t.Errorf("Expected response description %q, got %q", "OK", response.Description)
		}
		responseContent, ok := response.Content["application/json"]
		if !ok {
			t.Fatal("Expected application/json response content")
		}
		responseSchema := responseContent.Schema
		responseProperties, ok := responseSchema["properties"].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected response properties map, got %T", responseSchema["properties"])
		}
		data, ok := responseProperties["data"].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected data schema map, got %T", responseProperties["data"])
		}
		dataProperties, ok := data["properties"].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected data properties map, got %T", data["properties"])
		}
		idSchema, ok := dataProperties["id"].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected id schema map, got %T", dataProperties["id"])
		}
		if idSchema["type"] != "number" {
			t.Errorf("Expected id type %q, got %v", "number", idSchema["type"])
		}
	})

	t.Run("GivenBuiltDocument_whenMarshal_thenSpecJSONKeys", func(t *testing.T) {
		// Act
		doc := BuildOpenAPIDocument(apis)
		data, err := json.Marshal(doc)
		if err != nil {
			t.Fatalf("Failed to marshal document: %v", err)
		}
		var raw map[string]interface{}
		if err := json.Unmarshal(data, &raw); err != nil {
			t.Fatalf("Failed to unmarshal document: %v", err)
		}

		// Assert - envelope keys
		if raw["openapi"] != "3.1.0" {
			t.Errorf("Expected openapi %q, got %v", "3.1.0", raw["openapi"])
		}
		info, ok := raw["info"].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected info object, got %T", raw["info"])
		}
		if info["title"] != "Knot Export" || info["version"] != "1.0.0" {
			t.Errorf("Unexpected info object: %v", info)
		}

		// Assert - paths[path][method-lowercase] keying
		paths, ok := raw["paths"].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected paths object, got %T", raw["paths"])
		}
		item, ok := paths["/users/{id}"].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected path item object, got %T", paths["/users/{id}"])
		}
		op, ok := item["get"].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected get operation object, got %T", item["get"])
		}
		if op["operationId"] != "create-user" {
			t.Errorf("Expected operationId %q, got %v", "create-user", op["operationId"])
		}
		if op["summary"] != "Create User!" {
			t.Errorf("Expected summary %q, got %v", "Create User!", op["summary"])
		}
		if op["description"] != note {
			t.Errorf("Expected description %q, got %v", note, op["description"])
		}
		if _, hasMarker := op["x-knot-type"]; hasMarker {
			t.Error("Expected no x-knot-type key on HTTP operation")
		}
		responses, ok := op["responses"].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected responses object, got %T", op["responses"])
		}
		if _, has200 := responses["200"]; !has200 {
			t.Error("Expected a 200 response key")
		}
	})

	t.Run("GivenNoAPIs_whenBuild_thenEmptyPaths", func(t *testing.T) {
		// Act
		doc := BuildOpenAPIDocument(nil)

		// Assert - initialized empty paths and no tags
		if doc == nil {
			t.Fatal("Expected a document, got nil")
		}
		if doc.Paths == nil || len(doc.Paths) != 0 {
			t.Errorf("Expected initialized empty paths map, got %v", doc.Paths)
		}
		if len(doc.Tags) != 0 {
			t.Errorf("Expected no tags, got %+v", doc.Tags)
		}
	})
}

// TestBuildOpenAPIDocument_RPC covers exporting RPC APIs (empty or set
// method) as post operations carrying the x-knot-type marker.
func TestBuildOpenAPIDocument_RPC(t *testing.T) {
	t.Run("GivenRPCWithEmptyMethod_whenBuild_thenPostWithKnotMarker", func(t *testing.T) {
		// Arrange
		apis := []APIWithParams{
			{
				API:       models.API{ID: 1, GroupID: 100, Name: "Execute Job", Endpoint: "/jobs/execute", Method: "", Type: "RPC", Order: 0},
				GroupName: "Jobs",
			},
		}

		// Act
		doc := BuildOpenAPIDocument(apis)

		// Assert - keyed as post with the rpc marker
		item, ok := doc.Paths["/jobs/execute"]
		if !ok || item == nil {
			t.Fatalf("Expected path item for /jobs/execute, got %v", doc.Paths)
		}
		op := item.Post
		if op == nil {
			t.Fatal("Expected a post operation for the RPC API")
		}
		if item.Get != nil {
			t.Error("Expected no get operation for the RPC API")
		}
		if op.XKnotType != "rpc" {
			t.Errorf("Expected x-knot-type %q, got %q", "rpc", op.XKnotType)
		}
		if op.Summary != "Execute Job" {
			t.Errorf("Expected summary %q, got %q", "Execute Job", op.Summary)
		}
		if op.OperationID != "execute-job" {
			t.Errorf("Expected operationId %q, got %q", "execute-job", op.OperationID)
		}
		if op.RequestBody != nil {
			t.Error("Expected no requestBody when there are no request params")
		}
		if response, ok := op.Responses["200"]; !ok || response == nil {
			t.Fatalf("Expected a 200 response, got %v", op.Responses)
		} else if response.Content != nil {
			t.Error("Expected no response content when there are no response params")
		}

		// Assert - serialized form carries the marker for Knot's importer
		data, err := json.Marshal(doc)
		if err != nil {
			t.Fatalf("Failed to marshal document: %v", err)
		}
		var raw map[string]interface{}
		if err := json.Unmarshal(data, &raw); err != nil {
			t.Fatalf("Failed to unmarshal document: %v", err)
		}
		opJSON := raw["paths"].(map[string]interface{})["/jobs/execute"].(map[string]interface{})["post"].(map[string]interface{})
		if opJSON["x-knot-type"] != "rpc" {
			t.Errorf("Expected serialized x-knot-type %q, got %v", "rpc", opJSON["x-knot-type"])
		}
	})

	t.Run("GivenRPCWithMethod_whenBuild_thenStillPostWithMarker", func(t *testing.T) {
		// Arrange
		apis := []APIWithParams{
			{
				API:       models.API{ID: 1, GroupID: 100, Name: "Run Task", Endpoint: "/tasks/run", Method: "POST", Type: "RPC", Order: 0},
				GroupName: "Jobs",
			},
		}

		// Act
		doc := BuildOpenAPIDocument(apis)

		// Assert - RPC typing wins: single post operation with the marker
		item := doc.Paths["/tasks/run"]
		if item == nil {
			t.Fatal("Expected path item for /tasks/run")
		}
		if item.Post == nil || item.Post.XKnotType != "rpc" {
			t.Errorf("Expected post operation with x-knot-type rpc, got %+v", item.Post)
		}
		if item.Get != nil {
			t.Error("Expected no get operation for the RPC API")
		}
	})
}

// TestBuildOpenAPIDocument_Collision covers the merge policy when two
// selected APIs share endpoint and method: one operation, first
// summary/operationId wins, notes joined into the description in input order.
func TestBuildOpenAPIDocument_Collision(t *testing.T) {
	t.Run("GivenSameEndpointAndMethodTwice_whenBuild_thenMergedOperation", func(t *testing.T) {
		// Arrange
		firstNote := "First note"
		secondNote := "Second note"
		apis := []APIWithParams{
			{
				API:       models.API{ID: 1, GroupID: 100, Name: "Create User!", Endpoint: "/users", Method: "POST", Type: "HTTP", Order: 0, Note: &firstNote},
				GroupName: "Users",
			},
			{
				API:       models.API{ID: 2, GroupID: 100, Name: "Create User Again", Endpoint: "/users", Method: "POST", Type: "HTTP", Order: 1, Note: &secondNote},
				GroupName: "Users",
			},
		}

		// Act
		doc := BuildOpenAPIDocument(apis)

		// Assert - exactly one operation on the path
		item, ok := doc.Paths["/users"]
		if !ok || item == nil {
			t.Fatalf("Expected path item for /users, got %v", doc.Paths)
		}
		op := item.Post
		if op == nil {
			t.Fatal("Expected a post operation on /users")
		}
		if item.Get != nil || item.Put != nil || item.Delete != nil || item.Patch != nil {
			t.Error("Expected the merged collision to yield exactly one operation")
		}

		// Assert - first summary and operationId win
		if op.Summary != "Create User!" {
			t.Errorf("Expected first summary %q, got %q", "Create User!", op.Summary)
		}
		if op.OperationID != "create-user" {
			t.Errorf("Expected first operationId %q, got %q", "create-user", op.OperationID)
		}

		// Assert - notes joined into the description in input order
		wantDescription := firstNote + "\n\n---\n\n" + secondNote
		if op.Description != wantDescription {
			t.Errorf("Expected description %q, got %q", wantDescription, op.Description)
		}

		// Assert - duplicate group still yields a single tag
		if len(doc.Tags) != 1 || doc.Tags[0].Name != "Users" {
			t.Errorf("Expected tags [Users], got %+v", doc.Tags)
		}
	})

	t.Run("GivenFirstWithoutNote_whenBuild_thenDescriptionIsSecondNote", func(t *testing.T) {
		// Arrange
		secondNote := "Only note"
		apis := []APIWithParams{
			{
				API:       models.API{ID: 1, GroupID: 100, Name: "List Users", Endpoint: "/users", Method: "GET", Type: "HTTP", Order: 0},
				GroupName: "Users",
			},
			{
				API:       models.API{ID: 2, GroupID: 100, Name: "List Users Again", Endpoint: "/users", Method: "GET", Type: "HTTP", Order: 1, Note: &secondNote},
				GroupName: "Users",
			},
		}

		// Act
		doc := BuildOpenAPIDocument(apis)

		// Assert - description starts with the first note that exists
		op := doc.Paths["/users"].Get
		if op == nil {
			t.Fatal("Expected a get operation on /users")
		}
		if op.Description != secondNote {
			t.Errorf("Expected description %q, got %q", secondNote, op.Description)
		}
	})
}
