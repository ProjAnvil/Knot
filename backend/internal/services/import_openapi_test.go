package services

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	models "github.com/ProjAnvil/knot/models"
)

// TestParseOpenAPIDocument covers the OpenAPI 3.x importer per spec §8.3:
// paths×methods iteration in deterministic order, the name fallback chain,
// tag-based grouping with the "Imported" default, request/response JSON
// Schema conversion (the NodesToJSONSchema inverse), response candidate
// preference (200, then any-case 2XX, then default), $ref resolution with
// cycle guard and depth cap, external-ref skipping, the x-knot-type RPC
// marker, unreachable component schemas, and a full round-trip against the
// real OpenAPI export builder.
func TestParseOpenAPIDocument(t *testing.T) {
	t.Run("givenMultipleMethodsAndPaths_whenParse_thenAPIsInSortedPathAndMethodOrder", func(t *testing.T) {
		// Arrange - two paths sharing one tag; the path item also carries a
		// non-operation key ("parameters") and the root declares servers,
		// both of which must be ignored
		data := []byte(`{
			"openapi": "3.1.0",
			"info": {"title": "API", "version": "1.0.0"},
			"servers": [{"url": "https://api.example.com"}],
			"paths": {
				"/users": {
					"parameters": [{"name": "q", "in": "query"}],
					"summary": "User operations",
					"get": {"summary": "List users", "tags": ["Users"]},
					"post": {"summary": "Create user", "tags": ["Users"]}
				},
				"/admin": {
					"get": {"summary": "Admin panel", "tags": ["Users"]},
					"head": {"summary": "Health", "tags": ["Users"]}
				}
			}
		}`)

		// Act
		plan, err := ParseOpenAPIDocument(data)

		// Assert - one group, four APIs ordered by sorted path then fixed
		// method order (get, post, put, delete, patch, head, options, trace)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if plan == nil {
			t.Fatal("Expected a plan, got nil")
		}
		if len(plan.Warnings) != 0 {
			t.Errorf("Expected no warnings, got %v", plan.Warnings)
		}
		if len(plan.Groups) != 1 || plan.Groups[0].Name != "Users" {
			t.Fatalf("Expected single group \"Users\", got %+v", plan.Groups)
		}
		apis := plan.Groups[0].APIs
		want := []struct{ endpoint, method, name string }{
			{"/admin", "GET", "Admin panel"},
			{"/admin", "HEAD", "Health"},
			{"/users", "GET", "List users"},
			{"/users", "POST", "Create user"},
		}
		if len(apis) != len(want) {
			t.Fatalf("Expected %d APIs, got %d: %+v", len(want), len(apis), apis)
		}
		for i, w := range want {
			if apis[i].Endpoint != w.endpoint || apis[i].Method != w.method || apis[i].Name != w.name {
				t.Errorf("API %d: expected %s %s %q, got %s %s %q",
					i, w.method, w.endpoint, w.name, apis[i].Method, apis[i].Endpoint, apis[i].Name)
			}
			if apis[i].Type != "HTTP" {
				t.Errorf("API %d: expected type HTTP, got %q", i, apis[i].Type)
			}
		}
	})

	t.Run("givenNameFallbackChain_whenParse_thenSummaryThenOperationIdThenDerived", func(t *testing.T) {
		// Arrange - one operation per fallback level
		data := []byte(`{
			"openapi": "3.1.0",
			"info": {"title": "API", "version": "1.0.0"},
			"paths": {
				"/a": {"get": {"summary": "From summary", "operationId": "from-op-id"}},
				"/b": {"get": {"operationId": "from-op-id"}},
				"/c": {"delete": {}}
			}
		}`)

		// Act
		plan, err := ParseOpenAPIDocument(data)

		// Assert
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		apis := allPlanAPIs(plan)
		wantNames := []string{"From summary", "from-op-id", "DELETE /c"}
		if len(apis) != len(wantNames) {
			t.Fatalf("Expected %d APIs, got %d", len(wantNames), len(apis))
		}
		for i, want := range wantNames {
			if apis[i].Name != want {
				t.Errorf("API %d: expected name %q, got %q", i, want, apis[i].Name)
			}
		}
	})

	t.Run("givenTags_whenParse_thenFirstTagGroupElseImported", func(t *testing.T) {
		// Arrange - a tagged operation (first tag wins) and an untagged one
		data := []byte(`{
			"openapi": "3.1.0",
			"info": {"title": "API", "version": "1.0.0"},
			"paths": {
				"/a": {"get": {"summary": "Tagged", "tags": ["Users", "Extra"]}},
				"/b": {"get": {"summary": "Untagged"}}
			}
		}`)

		// Act
		plan, err := ParseOpenAPIDocument(data)

		// Assert - groups in first-appearance order
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		wantGroups := []string{"Users", "Imported"}
		if len(plan.Groups) != len(wantGroups) {
			t.Fatalf("Expected groups %v, got %+v", wantGroups, plan.Groups)
		}
		for i, want := range wantGroups {
			if plan.Groups[i].Name != want {
				t.Errorf("Group %d: expected %q, got %q", i, want, plan.Groups[i].Name)
			}
		}
		for _, group := range plan.Groups {
			if len(group.APIs) != 1 {
				t.Errorf("Group %q: expected 1 API, got %d", group.Name, len(group.APIs))
			}
		}
	})

	t.Run("givenJSONSchemaProperties_whenParse_thenRequestNodesWithTypeRequiredDescription", func(t *testing.T) {
		// Arrange - one operation exercising every schema shape; a second
		// operation carries only a non-JSON media type, which is ignored
		data := []byte(`{
			"openapi": "3.1.0",
			"info": {"title": "API", "version": "1.0.0"},
			"paths": {
				"/users": {"post": {"summary": "Create user", "tags": ["Users"], "requestBody": {
					"content": {"application/json": {"schema": {
						"type": "object",
						"properties": {
							"active": {"type": "boolean"},
							"address": {"type": "object", "properties": {"city": {"type": "string"}}, "required": ["city"]},
							"age": {"type": "number"},
							"name": {"type": "string", "description": "User name"},
							"tags": {"type": "array", "items": {"type": "string"}}
						},
						"required": ["name", "address"]
					}}}
				}}},
				"/legacy": {"post": {"summary": "Legacy", "requestBody": {
					"content": {"application/xml": {"schema": {"type": "object", "properties": {"x": {"type": "string"}}}}}
				}}}
			}
		}`)

		// Act
		plan, err := ParseOpenAPIDocument(data)

		// Assert - properties become nodes in sorted-key order with Order
		// as the sorted index and "required" membership as the flag
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if len(plan.Warnings) != 0 {
			t.Errorf("Expected no warnings, got %v", plan.Warnings)
		}
		request := findPlanAPI(t, plan, "/users").Request
		if len(request) != 5 {
			t.Fatalf("Expected 5 request nodes, got %d: %+v", len(request), request)
		}
		assertNode(t, request[0], "active", "boolean", false, 0, nil)
		assertNode(t, request[1], "address", "object", true, 1, nil)
		if len(request[1].Children) != 1 {
			t.Fatalf("Expected address to have 1 child, got %+v", request[1].Children)
		}
		assertNode(t, request[1].Children[0], "city", "string", true, 0, nil)
		assertNode(t, request[2], "age", "number", false, 2, nil)
		assertNode(t, request[3], "name", "string", true, 3, strPtr("User name"))
		assertNode(t, request[4], "tags", "array", false, 4, nil)
		if len(request[4].Children) != 1 {
			t.Fatalf("Expected tags to have 1 item child, got %+v", request[4].Children)
		}
		assertNode(t, request[4].Children[0], "item", "string", false, 0, nil)

		// Assert - non-JSON media types yield no request nodes
		if got := findPlanAPI(t, plan, "/legacy").Request; len(got) != 0 {
			t.Errorf("Expected no request nodes for an XML-only body, got %+v", got)
		}
	})

	t.Run("givenResponseCandidates_whenParse_then200ThenAnyCase2xxThenDefault", func(t *testing.T) {
		// Arrange - /a: 200 beats default; /b: 2xx (lower-case) beats
		// default; /c: only a non-selected status; /d: a 200 without a JSON
		// schema falls through to default
		data := []byte(`{
			"openapi": "3.1.0",
			"info": {"title": "API", "version": "1.0.0"},
			"paths": {
				"/a": {"get": {"responses": {
					"200": {"description": "OK", "content": {"application/json": {"schema": {"type": "object", "properties": {"from200": {"type": "string"}}}}}},
					"default": {"description": "err", "content": {"application/json": {"schema": {"type": "object", "properties": {"fromDefault": {"type": "string"}}}}}}
				}}},
				"/b": {"get": {"responses": {
					"2xx": {"description": "ok", "content": {"application/json": {"schema": {"type": "object", "properties": {"from2xx": {"type": "string"}}}}}},
					"default": {"description": "err", "content": {"application/json": {"schema": {"type": "object", "properties": {"fromDefault2": {"type": "string"}}}}}}
				}}},
				"/c": {"get": {"responses": {
					"404": {"description": "no", "content": {"application/json": {"schema": {"type": "object", "properties": {"notSelected": {"type": "string"}}}}}}
				}}},
				"/d": {"get": {"responses": {
					"200": {"description": "no content here"},
					"default": {"description": "err", "content": {"application/json": {"schema": {"type": "object", "properties": {"fromDefaultD": {"type": "string"}}}}}}
				}}}
			}
		}`)

		// Act
		plan, err := ParseOpenAPIDocument(data)

		// Assert
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		cases := []struct {
			endpoint string
			want     []string
		}{
			{"/a", []string{"from200"}},
			{"/b", []string{"from2xx"}},
			{"/c", nil},
			{"/d", []string{"fromDefaultD"}},
		}
		for _, c := range cases {
			response := findPlanAPI(t, plan, c.endpoint).Response
			if len(response) != len(c.want) {
				t.Fatalf("%s: expected %d response nodes, got %d: %+v", c.endpoint, len(c.want), len(response), response)
			}
			for i, want := range c.want {
				if response[i].Name != want {
					t.Errorf("%s: expected response node %q, got %q", c.endpoint, want, response[i].Name)
				}
			}
		}
	})

	t.Run("givenLocalRef_whenParse_thenResolvedAgainstComponents", func(t *testing.T) {
		// Arrange - the request schema is a $ref whose property nests a
		// second $ref; the response schema references the nested schema too
		data := []byte(`{
			"openapi": "3.1.0",
			"info": {"title": "API", "version": "1.0.0"},
			"paths": {
				"/users": {"post": {"summary": "Create user", "requestBody": {
					"content": {"application/json": {"schema": {"$ref": "#/components/schemas/User"}}}
				}, "responses": {
					"200": {"description": "OK", "content": {"application/json": {"schema": {"$ref": "#/components/schemas/Profile"}}}}
				}}}
			},
			"components": {"schemas": {
				"User": {"type": "object", "properties": {
					"id": {"type": "number"},
					"profile": {"$ref": "#/components/schemas/Profile"}
				}, "required": ["id"]},
				"Profile": {"type": "object", "properties": {"bio": {"type": "string"}}, "required": ["bio"]}
			}}
		}`)

		// Act
		plan, err := ParseOpenAPIDocument(data)

		// Assert - both refs resolve without warnings
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if len(plan.Warnings) != 0 {
			t.Errorf("Expected no warnings, got %v", plan.Warnings)
		}
		request := findPlanAPI(t, plan, "/users").Request
		if len(request) != 2 {
			t.Fatalf("Expected 2 request nodes, got %+v", request)
		}
		assertNode(t, request[0], "id", "number", true, 0, nil)
		assertNode(t, request[1], "profile", "object", false, 1, nil)
		if len(request[1].Children) != 1 {
			t.Fatalf("Expected profile to have 1 child, got %+v", request[1].Children)
		}
		assertNode(t, request[1].Children[0], "bio", "string", true, 0, nil)

		response := findPlanAPI(t, plan, "/users").Response
		if len(response) != 1 {
			t.Fatalf("Expected 1 response node, got %+v", response)
		}
		assertNode(t, response[0], "bio", "string", true, 0, nil)
	})

	t.Run("givenCyclicRefs_whenParse_thenTerminatesWithWarningAndSubtreeSkipped", func(t *testing.T) {
		// Arrange - A references B and B references A (Review Focus 3):
		// parsing must terminate, record a cyclic-ref warning and skip the
		// offending subtree while keeping the non-cyclic siblings
		data := []byte(`{
			"openapi": "3.1.0",
			"info": {"title": "API", "version": "1.0.0"},
			"paths": {
				"/tree": {"post": {"summary": "Mutual", "requestBody": {
					"content": {"application/json": {"schema": {"$ref": "#/components/schemas/A"}}}
				}}}
			},
			"components": {"schemas": {
				"A": {"type": "object", "properties": {
					"aName": {"type": "string"},
					"b": {"$ref": "#/components/schemas/B"}
				}},
				"B": {"type": "object", "properties": {
					"a": {"$ref": "#/components/schemas/A"},
					"bName": {"type": "string"}
				}}
			}}
		}`)

		// Act - terminating at all is part of what is under test
		plan, err := ParseOpenAPIDocument(data)

		// Assert
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if len(plan.Warnings) != 1 {
			t.Fatalf("Expected exactly 1 warning, got %v", plan.Warnings)
		}
		if !strings.Contains(plan.Warnings[0], "cyclic") || !strings.Contains(plan.Warnings[0], "#/components/schemas/A") {
			t.Errorf("Expected a cyclic warning naming the A ref, got %q", plan.Warnings[0])
		}

		// aName survives; b survives with B's non-cyclic child only (the
		// back-reference to A is the skipped subtree). Surviving nodes keep
		// their sorted-key enumeration index as Order, so gaps where
		// subtrees were skipped remain.
		request := findPlanAPI(t, plan, "/tree").Request
		if len(request) != 2 {
			t.Fatalf("Expected 2 request nodes, got %+v", request)
		}
		assertNode(t, request[0], "aName", "string", false, 0, nil)
		assertNode(t, request[1], "b", "object", false, 1, nil)
		if len(request[1].Children) != 1 {
			t.Fatalf("Expected b to keep only its non-cyclic child, got %+v", request[1].Children)
		}
		assertNode(t, request[1].Children[0], "bName", "string", false, 1, nil)
	})

	t.Run("givenSchemaDeeperThan32Levels_whenParse_thenWarningAndSubtreeSkipped", func(t *testing.T) {
		// Arrange - 60 nested object schemas; the cap allows 32 levels
		schema := `{"type": "string"}`
		for i := 1; i < 60; i++ {
			schema = fmt.Sprintf(`{"type": "object", "properties": {"child": %s}}`, schema)
		}
		data := []byte(fmt.Sprintf(`{
			"openapi": "3.1.0",
			"info": {"title": "API", "version": "1.0.0"},
			"paths": {
				"/deep": {"post": {"summary": "Deep", "requestBody": {
					"content": {"application/json": {"schema": %s}}
				}}}
			}
		}`, schema))

		// Act
		plan, err := ParseOpenAPIDocument(data)

		// Assert - exactly one depth warning and a chain of exactly 32
		// nested nodes (levels 1..32 of the property chain)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if len(plan.Warnings) != 1 || !strings.Contains(plan.Warnings[0], "32") {
			t.Fatalf("Expected one depth warning mentioning 32, got %v", plan.Warnings)
		}
		request := findPlanAPI(t, plan, "/deep").Request
		if len(request) != 1 {
			t.Fatalf("Expected 1 request root, got %+v", request)
		}
		depth := 1
		node := request[0]
		for len(node.Children) > 0 {
			node = node.Children[0]
			depth++
		}
		if depth != 32 {
			t.Errorf("Expected a 32-level chain before the skip, got %d levels", depth)
		}
	})

	t.Run("givenChainedRefs_whenParse_thenResolvedTransitively", func(t *testing.T) {
		// Arrange - A points at B which points at the real schema (refs
		// replace transitively); a second chain loops C→D→C
		data := []byte(`{
			"openapi": "3.1.0",
			"info": {"title": "API", "version": "1.0.0"},
			"paths": {
				"/chain": {"post": {"summary": "Chain", "requestBody": {
					"content": {"application/json": {"schema": {"$ref": "#/components/schemas/A"}}}
				}}},
				"/loop": {"post": {"summary": "Loop", "requestBody": {
					"content": {"application/json": {"schema": {"$ref": "#/components/schemas/C"}}}
				}}}
			},
			"components": {"schemas": {
				"A": {"$ref": "#/components/schemas/B"},
				"B": {"type": "object", "properties": {"chainField": {"type": "string"}}},
				"C": {"$ref": "#/components/schemas/D"},
				"D": {"$ref": "#/components/schemas/C"}
			}}
		}`)

		// Act
		plan, err := ParseOpenAPIDocument(data)

		// Assert - the chain resolves to B's properties without warnings;
		// the loop terminates with a cyclic warning and no nodes
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		request := findPlanAPI(t, plan, "/chain").Request
		if len(request) != 1 || request[0].Name != "chainField" {
			t.Fatalf("Expected request [chainField] through the ref chain, got %+v", request)
		}
		loop := findPlanAPI(t, plan, "/loop").Request
		if len(loop) != 0 {
			t.Errorf("Expected no nodes from the cyclic chain, got %+v", loop)
		}
		if len(plan.Warnings) != 1 || !strings.Contains(plan.Warnings[0], "cyclic") {
			t.Fatalf("Expected one cyclic warning for the chain loop, got %v", plan.Warnings)
		}
	})

	t.Run("givenExternalRef_whenParse_thenWarningAndSubtreeSkipped", func(t *testing.T) {
		// Arrange - one property references an external document; its
		// sibling is a plain schema
		data := []byte(`{
			"openapi": "3.1.0",
			"info": {"title": "API", "version": "1.0.0"},
			"paths": {
				"/users": {"post": {"summary": "Create user", "requestBody": {
					"content": {"application/json": {"schema": {
						"type": "object",
						"properties": {
							"ext": {"$ref": "https://schemas.example.com/user.json"},
							"keep": {"type": "string"}
						}
					}}}
				}}}
			}
		}`)

		// Act
		plan, err := ParseOpenAPIDocument(data)

		// Assert - the external-ref subtree is skipped with a warning
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if len(plan.Warnings) != 1 || !strings.Contains(plan.Warnings[0], "https://schemas.example.com/user.json") {
			t.Fatalf("Expected one warning naming the external ref, got %v", plan.Warnings)
		}
		request := findPlanAPI(t, plan, "/users").Request
		if len(request) != 1 {
			t.Fatalf("Expected only the sibling node, got %+v", request)
		}
		// keep holds its sorted-key index ("ext" < "keep"), leaving a gap
		assertNode(t, request[0], "keep", "string", false, 1, nil)
	})

	t.Run("givenKnotRPCMarker_whenParse_thenTypeRPCAndEmptyMethod", func(t *testing.T) {
		// Arrange - an operation carrying Knot's RPC round-trip marker next
		// to a regular one
		data := []byte(`{
			"openapi": "3.1.0",
			"info": {"title": "API", "version": "1.0.0"},
			"paths": {
				"/cache": {
					"post": {"x-knot-type": "rpc", "operationId": "purge-cache"},
					"get": {"operationId": "cache-status"}
				}
			}
		}`)

		// Act
		plan, err := ParseOpenAPIDocument(data)

		// Assert - get lands before post in the fixed method order
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		apis := plan.Groups[0].APIs
		if len(apis) != 2 {
			t.Fatalf("Expected 2 APIs, got %+v", apis)
		}
		if apis[0].Name != "cache-status" || apis[0].Type != "HTTP" || apis[0].Method != "GET" {
			t.Errorf("Expected HTTP/GET cache-status, got %+v", apis[0])
		}
		if apis[1].Name != "purge-cache" || apis[1].Type != "RPC" || apis[1].Method != "" {
			t.Errorf("Expected RPC with empty method purge-cache, got %+v", apis[1])
		}
	})

	t.Run("givenUnreachableComponentSchema_whenParse_thenNotImported", func(t *testing.T) {
		// Arrange - components hold one referenced and one orphaned schema
		data := []byte(`{
			"openapi": "3.1.0",
			"info": {"title": "API", "version": "1.0.0"},
			"paths": {
				"/a": {"get": {"summary": "A", "responses": {
					"200": {"description": "OK", "content": {"application/json": {"schema": {"$ref": "#/components/schemas/Used"}}}}
				}}}
			},
			"components": {"schemas": {
				"Used": {"type": "object", "properties": {"usedField": {"type": "string"}}},
				"Orphan": {"type": "object", "properties": {"orphanField": {"type": "string"}}}
			}}
		}`)

		// Act
		plan, err := ParseOpenAPIDocument(data)

		// Assert - only the reachable schema contributes nodes; the orphan
		// is silently ignored (no warning, no nodes)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if len(plan.Warnings) != 0 {
			t.Errorf("Expected no warnings, got %v", plan.Warnings)
		}
		if len(plan.Groups) != 1 || len(plan.Groups[0].APIs) != 1 {
			t.Fatalf("Expected one imported API, got %+v", plan.Groups)
		}
		for _, node := range allPlanAPIs(plan) {
			for _, root := range node.Request {
				if root.Name == "orphanField" {
					t.Error("Expected the orphaned schema to stay unimported")
				}
			}
			for _, root := range node.Response {
				if root.Name == "orphanField" {
					t.Error("Expected the orphaned schema to stay unimported")
				}
			}
		}
		response := plan.Groups[0].APIs[0].Response
		if len(response) != 1 || response[0].Name != "usedField" {
			t.Errorf("Expected response [usedField], got %+v", response)
		}
	})

	t.Run("givenBuiltDocument_whenMarshalAndParse_thenPlanMatches", func(t *testing.T) {
		// Arrange - in-memory APIs with nested params, notes and an RPC
		// entry, exported through the real builder. Param names are chosen
		// so their insertion order matches the importer's sorted-key order.
		// Group order follows first appearance over sorted paths
		// (/cache/purge < /items/{id} < /users), so Items precedes Users.
		note := "Requires admin role"
		nameDesc := "exact match"
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
					{ID: 12, APIID: 1, Name: "name", Type: "string", Required: true, Order: 0, ParentID: &filterID, Description: &nameDesc},
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
					{ID: 32, APIID: 3, Name: "count", Type: "number", Order: 0, ParentID: &dataID},
					{ID: 33, APIID: 3, Name: "label", Type: "string", Required: true, Order: 1, ParentID: &dataID},
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
		doc := BuildOpenAPIDocument(apis)
		data, err := json.Marshal(doc)
		if err != nil {
			t.Fatalf("Failed to marshal fixture document: %v", err)
		}

		// Act
		plan, err := ParseOpenAPIDocument(data)

		// Assert - parse succeeds without warnings
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if len(plan.Warnings) != 0 {
			t.Errorf("Expected no warnings, got %v", plan.Warnings)
		}

		// Assert - group order follows sorted-path first appearance
		wantGroups := []struct {
			name string
			apis []string
		}{
			{"Items", []string{"Purge cache", "Delete item"}},
			{"Users", []string{"List users", "Create user"}},
		}
		if len(plan.Groups) != len(wantGroups) {
			t.Fatalf("Expected %d groups, got %d: %+v", len(wantGroups), len(plan.Groups), plan.Groups)
		}
		for i, want := range wantGroups {
			got := plan.Groups[i]
			if got.Name != want.name {
				t.Errorf("Group %d: expected name %q, got %q", i, want.name, got.Name)
			}
			if len(got.APIs) != len(want.apis) {
				t.Fatalf("Group %q: expected %d APIs, got %d", want.name, len(want.apis), len(got.APIs))
			}
			for j, wantName := range want.apis {
				if got.APIs[j].Name != wantName {
					t.Errorf("Group %q API %d: expected name %q, got %q", want.name, j, wantName, got.APIs[j].Name)
				}
			}
		}

		// Assert - each API round-trips its fields and parameter trees
		// (mapped back to the fixture order used above)
		purge := plan.Groups[0].APIs[0]
		if purge.Type != "RPC" || purge.Method != "" || purge.Endpoint != "/cache/purge" {
			t.Errorf("Expected RPC Purge cache with empty method, got %+v", purge)
		}
		del := plan.Groups[0].APIs[1]
		if del.Type != "HTTP" || del.Method != "DELETE" {
			t.Errorf("Expected HTTP DELETE, got %+v", del)
		}
		assertParamNodesEqual(t, del.Response, models.TreeFromParams(apis[2].ResponseParameters), "response")

		list := plan.Groups[1].APIs[0]
		if list.Type != "HTTP" || list.Method != "GET" {
			t.Errorf("Expected HTTP GET, got %+v", list)
		}
		assertParamNodesEqual(t, list.Request, models.TreeFromParams(apis[0].RequestParameters), "request")
		if list.Note != nil {
			t.Errorf("Expected no note for List users, got %q", *list.Note)
		}

		create := plan.Groups[1].APIs[1]
		if create.Note == nil || *create.Note != note {
			t.Errorf("Expected note %q, got %v", note, create.Note)
		}
	})

	t.Run("givenNoPaths_whenParse_thenEmptyPlan", func(t *testing.T) {
		// Arrange - a valid document without a paths object
		data := []byte(`{"openapi": "3.1.0", "info": {"title": "API", "version": "1.0.0"}}`)

		// Act
		plan, err := ParseOpenAPIDocument(data)

		// Assert - empty but non-nil slices
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if plan == nil || plan.Groups == nil || plan.Warnings == nil {
			t.Fatalf("Expected non-nil plan slices, got %+v", plan)
		}
		if len(plan.Groups) != 0 || len(plan.Warnings) != 0 {
			t.Errorf("Expected an empty plan, got %+v", plan)
		}
	})

	t.Run("givenInvalidJSON_whenParse_thenError", func(t *testing.T) {
		// Arrange
		data := []byte(`{"openapi": "3.1.0", "paths":`)

		// Act
		plan, err := ParseOpenAPIDocument(data)

		// Assert
		if err == nil {
			t.Fatal("Expected an error, got nil")
		}
		if plan != nil {
			t.Errorf("Expected nil plan on error, got %+v", plan)
		}
		if !strings.Contains(err.Error(), "OpenAPI") {
			t.Errorf("Expected error to mention OpenAPI, got %q", err.Error())
		}
	})
}

// findPlanAPI returns the first API whose endpoint matches, failing the
// test when none exists.
func findPlanAPI(t *testing.T, plan *models.ImportPlan, endpoint string) models.ImportAPI {
	t.Helper()
	for _, api := range allPlanAPIs(plan) {
		if api.Endpoint == endpoint {
			return api
		}
	}
	t.Fatalf("No API with endpoint %q in plan: %+v", endpoint, plan.Groups)
	return models.ImportAPI{}
}

// allPlanAPIs flattens a plan into its APIs, group by group.
func allPlanAPIs(plan *models.ImportPlan) []models.ImportAPI {
	var apis []models.ImportAPI
	for _, group := range plan.Groups {
		apis = append(apis, group.APIs...)
	}
	return apis
}

// assertNode checks the scalar fields of one models.ParamNode.
func assertNode(t *testing.T, node models.ParamNode, name, nodeType string, required bool, order int, description *string) {
	t.Helper()
	if node.Name != name {
		t.Errorf("Expected node name %q, got %q", name, node.Name)
	}
	if node.Type != nodeType {
		t.Errorf("Node %q: expected type %q, got %q", name, nodeType, node.Type)
	}
	if node.Required != required {
		t.Errorf("Node %q: expected required %v, got %v", name, required, node.Required)
	}
	if node.Order != order {
		t.Errorf("Node %q: expected order %d, got %d", name, order, node.Order)
	}
	if (node.Description == nil) != (description == nil) {
		t.Errorf("Node %q: description nil-ness mismatch (want nil: %v)", name, description == nil)
	} else if description != nil && *node.Description != *description {
		t.Errorf("Node %q: expected description %q, got %q", name, *description, *node.Description)
	}
}

// strPtr is a test convenience for building description pointers.
func strPtr(s string) *string {
	return &s
}

// TestJSONSchemaToNodes covers the direct conversion entry point: it
// initializes the visited set when nil, and handles nil schemas, the depth
// cap and property-less schemas without panicking (warnings are dropped by
// this variant since it has no plan to record them on).
func TestJSONSchemaToNodes(t *testing.T) {
	t.Run("givenObjectSchemaAndNilVisited_whenConvert_thenSortedNodes", func(t *testing.T) {
		// Arrange - map iteration order is random, so sorted-key order is
		// what makes the result deterministic
		schema := map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"b": map[string]interface{}{"type": "string"},
				"a": map[string]interface{}{"type": "number"},
			},
		}

		// Act - a nil visited set is initialized internally
		nodes := JSONSchemaToNodes(schema, map[string]interface{}{}, nil, 0)

		// Assert
		if len(nodes) != 2 {
			t.Fatalf("Expected 2 nodes, got %+v", nodes)
		}
		assertNode(t, nodes[0], "a", "number", false, 0, nil)
		assertNode(t, nodes[1], "b", "string", false, 1, nil)
	})

	t.Run("givenNilSchema_whenConvert_thenNoNodes", func(t *testing.T) {
		// Act
		nodes := JSONSchemaToNodes(nil, nil, nil, 0)

		// Assert
		if len(nodes) != 0 {
			t.Errorf("Expected no nodes for a nil schema, got %+v", nodes)
		}
	})

	t.Run("givenSchemaWithoutProperties_whenConvert_thenNoNodes", func(t *testing.T) {
		// Arrange
		schema := map[string]interface{}{"type": "object"}

		// Act
		nodes := JSONSchemaToNodes(schema, nil, nil, 0)

		// Assert
		if len(nodes) != 0 {
			t.Errorf("Expected no nodes without properties, got %+v", nodes)
		}
	})

	t.Run("givenDepthBeyondCap_whenConvert_thenSubtreeSkipped", func(t *testing.T) {
		// Arrange - starting past the depth cap skips the whole subtree
		schema := map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{"a": map[string]interface{}{"type": "string"}},
		}

		// Act
		nodes := JSONSchemaToNodes(schema, nil, nil, openapiMaxSchemaDepth+1)

		// Assert
		if len(nodes) != 0 {
			t.Errorf("Expected the too-deep subtree to be skipped, got %+v", nodes)
		}
	})
}

// TestParseOpenAPIDocument_SchemaShapes covers the remaining schema
// conversion shapes: array items schemas (object elements and empty items),
// the type mapping ("integer" narrows to "number") and the type inference
// fallbacks (properties without a type read as object, unknown types
// degrade to string) and malformed path entries.
func TestParseOpenAPIDocument_SchemaShapes(t *testing.T) {
	// Arrange - one operation whose request schema exercises every shape,
	// plus a path entry that is not an object at all
	data := []byte(`{
		"openapi": "3.1.0",
		"info": {"title": "API", "version": "1.0.0"},
		"paths": {
			"/bad": "not-a-path-item",
			"/things": {"post": {"summary": "Create thing", "requestBody": {
				"content": {"application/json": {"schema": {
					"type": "object",
					"properties": {
						"badges": {"type": "array", "items": {"type": "object", "properties": {
							"label": {"type": "string"},
							"level": {"type": "number"}
						}, "required": ["label"]}},
						"count": {"type": "integer"},
						"flags": {"type": "array", "items": {}},
						"meta": {"properties": {"k": {"type": "string"}}},
						"weird": {"type": "opaque"}
					}
				}}}
			}}}
		}
	}`)

	// Act
	plan, err := ParseOpenAPIDocument(data)

	// Assert - the malformed path entry is skipped silently
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(plan.Warnings) != 0 {
		t.Errorf("Expected no warnings, got %v", plan.Warnings)
	}
	if apis := allPlanAPIs(plan); len(apis) != 1 {
		t.Fatalf("Expected only the /things operation, got %+v", apis)
	}

	// Assert - request nodes in sorted-key order
	request := findPlanAPI(t, plan, "/things").Request
	if len(request) != 5 {
		t.Fatalf("Expected 5 request nodes, got %+v", request)
	}
	// Array with an object items schema: one child per element field, sorted.
	assertNode(t, request[0], "badges", "array", false, 0, nil)
	if len(request[0].Children) != 2 {
		t.Fatalf("Expected badges to carry the element fields, got %+v", request[0].Children)
	}
	assertNode(t, request[0].Children[0], "label", "string", true, 0, nil)
	assertNode(t, request[0].Children[1], "level", "number", false, 1, nil)
	// A JSON Schema "integer" property imports as the numeric models.ParamNode type.
	assertNode(t, request[1], "count", "number", false, 1, nil)
	// An empty items schema contributes no children.
	assertNode(t, request[2], "flags", "array", false, 2, nil)
	if len(request[2].Children) != 0 {
		t.Errorf("Expected no children for empty items, got %+v", request[2].Children)
	}
	// A schema without "type" but with "properties" reads as an object.
	assertNode(t, request[3], "meta", "object", false, 3, nil)
	if len(request[3].Children) != 1 {
		t.Fatalf("Expected meta to have 1 child, got %+v", request[3].Children)
	}
	assertNode(t, request[3].Children[0], "k", "string", false, 0, nil)
	// Unknown schema types degrade to string leaves.
	assertNode(t, request[4], "weird", "string", false, 4, nil)
}
