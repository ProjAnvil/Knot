package services

import (
	"strings"
	"unicode"

	models "github.com/ProjAnvil/knot/models"
)

// OpenAPIDocument is the root object of an OpenAPI 3.1 document built by
// Knot. Paths is a map because path keys are arbitrary endpoint strings.
type OpenAPIDocument struct {
	OpenAPI string                      `json:"openapi"`
	Info    OpenAPIInfo                 `json:"info"`
	Tags    []OpenAPITag                `json:"tags,omitempty"`
	Paths   map[string]*OpenAPIPathItem `json:"paths"`
}

// OpenAPIInfo identifies the exporting tool and the document version.
type OpenAPIInfo struct {
	Title   string `json:"title"`
	Version string `json:"version"`
}

// OpenAPITag declares one API group as an OpenAPI tag.
type OpenAPITag struct {
	Name string `json:"name"`
}

// OpenAPIPathItem holds the operations of one endpoint. Each field
// serializes as its lower-cased HTTP method key.
type OpenAPIPathItem struct {
	Get    *OpenAPIOperation `json:"get,omitempty"`
	Post   *OpenAPIOperation `json:"post,omitempty"`
	Put    *OpenAPIOperation `json:"put,omitempty"`
	Delete *OpenAPIOperation `json:"delete,omitempty"`
	Patch  *OpenAPIOperation `json:"patch,omitempty"`
}

// OpenAPIOperation describes one endpoint+method pair.
type OpenAPIOperation struct {
	Summary     string                      `json:"summary"`
	Description string                      `json:"description,omitempty"`
	OperationID string                      `json:"operationId"`
	Tags        []string                    `json:"tags,omitempty"`
	XKnotType   string                      `json:"x-knot-type,omitempty"`
	RequestBody *OpenAPIRequestBody         `json:"requestBody,omitempty"`
	Responses   map[string]*OpenAPIResponse `json:"responses,omitempty"`
}

// OpenAPIRequestBody carries the request body schema and whether a body
// must be sent.
type OpenAPIRequestBody struct {
	Required bool                       `json:"required"`
	Content  map[string]*OpenAPIContent `json:"content"`
}

// OpenAPIResponse describes one HTTP response.
type OpenAPIResponse struct {
	Description string                     `json:"description"`
	Content     map[string]*OpenAPIContent `json:"content,omitempty"`
}

// OpenAPIContent wraps one JSON Schema under a media type key.
type OpenAPIContent struct {
	Schema map[string]interface{} `json:"schema"`
}

// BuildOpenAPIDocument assembles an OpenAPI 3.1 document from APIs with
// their parameters. Tags are declared per group in first-appearance order
// (tracked with a seen-map over the input slice, never map iteration, so
// tag order is deterministic). Operations are keyed by endpoint path and
// lower-cased HTTP method; when the same endpoint+method is selected twice
// the operations merge: the first occurrence wins for summary/operationId
// and subsequent notes are appended to the description joined by a
// thematic break, in input order.
func BuildOpenAPIDocument(apis []APIWithParams) *OpenAPIDocument {
	doc := &OpenAPIDocument{
		OpenAPI: "3.1.0",
		Info:    OpenAPIInfo{Title: "Knot Export", Version: "1.0.0"},
		Paths:   make(map[string]*OpenAPIPathItem),
	}

	tagSeen := make(map[string]bool)
	for _, api := range apis {
		// Declare the group tag on its first appearance.
		if !tagSeen[api.GroupName] {
			tagSeen[api.GroupName] = true
			doc.Tags = append(doc.Tags, OpenAPITag{Name: api.GroupName})
		}

		item, ok := doc.Paths[api.API.Endpoint]
		if !ok {
			item = &OpenAPIPathItem{}
			doc.Paths[api.API.Endpoint] = item
		}

		slot := item.operationSlot(operationVerb(api.API))
		if *slot == nil {
			*slot = newOpenAPIOperation(api)
			continue
		}

		// Collision: keep the first summary/operationId/tags and merge the
		// note into the description in input order.
		if note := noteText(api.API.Note); note != "" {
			if (*slot).Description == "" {
				(*slot).Description = note
			} else {
				(*slot).Description += "\n\n---\n\n" + note
			}
		}
	}

	return doc
}

// newOpenAPIOperation builds the operation for one API: summary and
// operationId from the API name, the group as the only tag, the request
// tree as requestBody (required iff any root request node is required) and
// the response tree as the 200 response. RPC-like APIs carry the
// x-knot-type marker so Knot's importer can restore their classification.
func newOpenAPIOperation(api APIWithParams) *OpenAPIOperation {
	op := &OpenAPIOperation{
		Summary:     api.API.Name,
		Description: noteText(api.API.Note),
		OperationID: slugifyOperationID(api.API.Name),
		Tags:        []string{api.GroupName},
		Responses:   map[string]*OpenAPIResponse{"200": {Description: "OK"}},
	}
	if exportsAsRPC(api.API) {
		op.XKnotType = "rpc"
	}

	requestTree := models.TreeFromParams(api.RequestParameters)
	if len(requestTree) > 0 {
		required := anyNodeRequired(requestTree)
		op.RequestBody = &OpenAPIRequestBody{
			Required: required,
			Content:  jsonContent(NodesToJSONSchema(requestTree, required)),
		}
	}

	responseTree := models.TreeFromParams(api.ResponseParameters)
	if len(responseTree) > 0 {
		response := op.Responses["200"]
		response.Content = jsonContent(NodesToJSONSchema(responseTree, anyNodeRequired(responseTree)))
	}

	return op
}

// operationVerb maps an API to its OpenAPI operation key. RPC-like APIs
// are keyed as post (consumers that ignore vendor extensions still see a
// valid document); other methods are lower-cased.
func operationVerb(api models.API) string {
	if exportsAsRPC(api) {
		return "post"
	}
	return strings.ToLower(api.Method)
}

// exportsAsRPC reports whether an API must be exported as an RPC
// operation: RPC-typed APIs (which may carry an empty HTTP method), plus
// any API whose method is missing, since there is no verb to key it under.
func exportsAsRPC(api models.API) bool {
	return api.Type == "RPC" || api.Method == ""
}

// operationSlot returns the address of the field holding the operation for
// a lower-cased verb, so callers can both read and assign it. Verbs outside
// Knot's method domain (get/put/delete/patch) land in the post slot so
// unrecognized methods are still exported.
func (p *OpenAPIPathItem) operationSlot(verb string) **OpenAPIOperation {
	switch verb {
	case "get":
		return &p.Get
	case "put":
		return &p.Put
	case "delete":
		return &p.Delete
	case "patch":
		return &p.Patch
	default:
		return &p.Post
	}
}

// noteText returns the note text of an API, or "" when there is none.
func noteText(note *string) string {
	if note == nil || *note == "" {
		return ""
	}
	return *note
}

// jsonContent wraps one JSON Schema in an application/json content map.
func jsonContent(schema map[string]interface{}) map[string]*OpenAPIContent {
	return map[string]*OpenAPIContent{"application/json": {Schema: schema}}
}

// anyNodeRequired reports whether any node in the forest is required.
func anyNodeRequired(nodes []models.ParamNode) bool {
	for _, node := range nodes {
		if node.Required {
			return true
		}
	}
	return false
}

// slugifyOperationID converts an API name into an operationId slug:
// lower-cased, runs of non-alphanumeric characters collapsed to a single
// hyphen, surrounding hyphens dropped ("Create User!" -> "create-user").
func slugifyOperationID(name string) string {
	var slug strings.Builder
	hyphenPending := false
	for _, r := range strings.ToLower(name) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if hyphenPending && slug.Len() > 0 {
				slug.WriteByte('-')
			}
			hyphenPending = false
			slug.WriteRune(r)
			continue
		}
		hyphenPending = true
	}
	return slug.String()
}

// NodesToJSONSchema converts a models.ParamNode forest into the JSON Schema of
// the value the forest describes: an implicit object whose properties are
// the nodes, mirroring GenerateExampleJSON, where each root node becomes
// one key of the request/response body. objectRequired reports whether
// that object requires its members (the requestBody-level required): when
// true and at least one node is required, the schema carries a "required"
// array listing their names; otherwise the key is omitted.
//
// Nested nodes map per the export spec: object -> {type, properties,
// required: [names of required children]}; array -> {type, items}; leaf
// types (string|number|boolean) map directly; description is carried on
// every node that has one.
func NodesToJSONSchema(nodes []models.ParamNode, objectRequired bool) map[string]interface{} {
	schema := map[string]interface{}{
		"type":       "object",
		"properties": propertiesOfNodes(nodes),
	}
	if objectRequired {
		if required := requiredNodeNames(nodes); len(required) > 0 {
			schema["required"] = required
		}
	}
	return schema
}

// propertiesOfNodes maps each node to its schema, keyed by node name.
func propertiesOfNodes(nodes []models.ParamNode) map[string]interface{} {
	properties := make(map[string]interface{}, len(nodes))
	for _, node := range nodes {
		properties[node.Name] = paramNodeToSchema(node)
	}
	return properties
}

// requiredNodeNames lists the names of required nodes in input order.
func requiredNodeNames(nodes []models.ParamNode) []string {
	var required []string
	for _, node := range nodes {
		if node.Required {
			required = append(required, node.Name)
		}
	}
	return required
}

// objectSchemaOfNodes builds the schema of an object whose properties are
// the given nodes, always emitting "required" when any node is required
// (nested object schemas are not gated by objectRequired).
func objectSchemaOfNodes(nodes []models.ParamNode) map[string]interface{} {
	schema := map[string]interface{}{
		"type":       "object",
		"properties": propertiesOfNodes(nodes),
	}
	if required := requiredNodeNames(nodes); len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

// paramNodeToSchema builds the JSON Schema of a single node.
func paramNodeToSchema(node models.ParamNode) map[string]interface{} {
	schemaType := node.Type
	if !isLeafType(schemaType) && schemaType != "object" && schemaType != "array" {
		// Unknown types degrade to string, mirroring models.ParamNodesFromJSONValue.
		schemaType = "string"
	}

	schema := map[string]interface{}{"type": schemaType}
	if node.Description != nil && *node.Description != "" {
		schema["description"] = *node.Description
	}

	switch schemaType {
	case "object":
		schema["properties"] = propertiesOfNodes(node.Children)
		if required := requiredNodeNames(node.Children); len(required) > 0 {
			schema["required"] = required
		}
	case "array":
		schema["items"] = arrayItemsSchema(node.Children)
	}
	return schema
}

// arrayItemsSchema builds the element schema of an array node, mirroring
// GenerateExampleJSON's example values: no children -> a generic schema
// that allows anything; a single leaf child -> that leaf's schema;
// otherwise the children form the properties of an object element.
func arrayItemsSchema(children []models.ParamNode) map[string]interface{} {
	if len(children) == 0 {
		return map[string]interface{}{}
	}
	if len(children) == 1 && isLeafType(children[0].Type) {
		return paramNodeToSchema(children[0])
	}
	return objectSchemaOfNodes(children)
}

// isLeafType reports whether a parameter type maps to a JSON Schema leaf.
func isLeafType(paramType string) bool {
	switch paramType {
	case "string", "number", "boolean":
		return true
	}
	return false
}
