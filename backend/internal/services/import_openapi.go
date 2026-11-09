package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// openapiMethods lists the HTTP method keys of an OpenAPI Path Item Object
// in the fixed order the importer walks them, so the APIs of one document
// always import in the same sequence (Go map iteration is randomized and
// decoded JSON maps carry no key order).
var openapiMethods = []string{"get", "post", "put", "delete", "patch", "head", "options", "trace"}

const (
	// openapiMaxSchemaDepth caps recursive JSON Schema descent per spec
	// §8.3; anything nested deeper is skipped with a recorded warning.
	openapiMaxSchemaDepth = 32
	// openapiSchemaRefPrefix is the only $ref form the importer resolves:
	// local pointers into components.schemas.
	openapiSchemaRefPrefix = "#/components/schemas/"
	// openapiDefaultGroup names the group of operations without tags.
	openapiDefaultGroup = "Imported"
)

// openapiParser carries importer state: the plan under construction, the
// decoded document (the resolution root for $ref) and a group-name → index
// map so operations sharing a tag merge into a single group.
type openapiParser struct {
	plan       *ImportPlan
	doc        map[string]interface{}
	groupIndex map[string]int
}

// ParseOpenAPIDocument parses an OpenAPI 3.x document into an ImportPlan
// per spec §8.3. Path items are walked in sorted path order and each
// operation key (get/post/put/delete/patch/head/options/trace) in the
// fixed method order; other path-item keys (parameters, summary, ...) and
// root-level servers are ignored, as are schemas not reachable from any
// path. The operation name falls back summary → operationId →
// "METHOD /path"; the group is the first tag, else "Imported". Request and
// response ParamNodes come from the application/json schema of requestBody
// and of the first informative response ("200", then an any-case "2XX",
// then "default"), inverting NodesToJSONSchema. An operation marked
// "x-knot-type": "rpc" imports as Type "RPC" with an empty method,
// round-tripping the OpenAPI exporter. $ref values are resolved against
// the document with a visited-set cycle guard and a depth cap; offending
// subtrees are skipped with recorded warnings. A UTF-8 BOM is tolerated so
// any file accepted by DetectImportFormat parses.
func ParseOpenAPIDocument(data []byte) (*ImportPlan, error) {
	data = bytes.TrimPrefix(data, utf8BOM)

	var doc map[string]interface{}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("invalid OpenAPI file: %w", err)
	}

	parser := &openapiParser{
		plan:       &ImportPlan{Groups: []ImportGroup{}, Warnings: []string{}},
		doc:        doc,
		groupIndex: make(map[string]int),
	}
	parser.walkPaths(openapiMapValue(doc["paths"]))
	return parser.plan, nil
}

// walkPaths visits path items in sorted key order and each operation key
// in the fixed method order, keeping the resulting plan deterministic.
func (p *openapiParser) walkPaths(paths map[string]interface{}) {
	for _, path := range openapiSortedKeys(paths) {
		item := openapiMapValue(paths[path])
		if item == nil {
			continue
		}
		for _, method := range openapiMethods {
			if op := openapiMapValue(item[method]); op != nil {
				p.addOperation(path, method, op)
			}
		}
	}
}

// addOperation converts one operation object into an ImportAPI appended to
// the group named by its first tag (groups are created lazily, so plan
// group order is first-appearance order over the sorted paths).
func (p *openapiParser) addOperation(path, method string, op map[string]interface{}) {
	upperMethod := strings.ToUpper(method)
	name := openapiStringValue(op["summary"])
	if name == "" {
		name = openapiStringValue(op["operationId"])
	}
	if name == "" {
		name = upperMethod + " " + path
	}

	api := ImportAPI{
		Name:     name,
		Endpoint: path,
		Request:  []ParamNode{},
		Response: []ParamNode{},
	}
	if openapiStringValue(op["x-knot-type"]) == "rpc" {
		api.Type = "RPC"
	} else {
		api.Type = "HTTP"
		api.Method = upperMethod
	}
	if description := openapiStringValue(op["description"]); description != "" {
		api.Note = &description
	}
	api.Request = p.schemaNodes(openapiRequestBodySchema(op["requestBody"]))
	api.Response = p.schemaNodes(openapiResponseSchema(op["responses"]))

	group := openapiOperationGroup(op["tags"])
	index, seen := p.groupIndex[group]
	if !seen {
		index = len(p.plan.Groups)
		p.groupIndex[group] = index
		p.plan.Groups = append(p.plan.Groups, ImportGroup{Name: group, APIs: []ImportAPI{}})
	}
	p.plan.Groups[index].APIs = append(p.plan.Groups[index].APIs, api)
}

// schemaNodes converts one operation-level JSON Schema into ParamNodes,
// starting a fresh visited set for the $ref cycle guard; a nil schema
// yields an empty (non-nil) slice.
func (p *openapiParser) schemaNodes(schema map[string]interface{}) []ParamNode {
	if schema == nil {
		return []ParamNode{}
	}
	return jsonSchemaToNodes(schema, p.doc, map[string]bool{}, 0, &p.plan.Warnings)
}

// openapiOperationGroup returns the operation's group name: its first
// non-empty tag, or "Imported" when untagged.
func openapiOperationGroup(raw interface{}) string {
	for _, tag := range openapiArrayValue(raw) {
		if name := openapiStringValue(tag); name != "" {
			return name
		}
	}
	return openapiDefaultGroup
}

// openapiRequestBodySchema extracts the application/json schema of a
// requestBody object, or nil when absent.
func openapiRequestBodySchema(raw interface{}) map[string]interface{} {
	body := openapiMapValue(raw)
	if body == nil {
		return nil
	}
	return openapiContentJSONSchema(body["content"])
}

// openapiResponseSchema selects the response schema per spec §8.3: "200",
// then an any-case "2XX" key, then "default" — the first candidate that
// actually carries an application/json schema wins; nil when none does.
func openapiResponseSchema(raw interface{}) map[string]interface{} {
	responses := openapiMapValue(raw)
	if responses == nil {
		return nil
	}
	if schema := openapiResponseEntrySchema(responses["200"]); schema != nil {
		return schema
	}
	for _, key := range openapiSortedKeys(responses) {
		if strings.EqualFold(key, "2XX") {
			if schema := openapiResponseEntrySchema(responses[key]); schema != nil {
				return schema
			}
		}
	}
	return openapiResponseEntrySchema(responses["default"])
}

// openapiResponseEntrySchema returns the application/json schema of one
// response entry, or nil when the entry or its JSON schema is absent.
func openapiResponseEntrySchema(raw interface{}) map[string]interface{} {
	response := openapiMapValue(raw)
	if response == nil {
		return nil
	}
	return openapiContentJSONSchema(response["content"])
}

// openapiContentJSONSchema extracts the schema of the application/json
// media type from an OpenAPI content map, or nil when absent. Other media
// types are ignored per spec §8.3.
func openapiContentJSONSchema(raw interface{}) map[string]interface{} {
	content := openapiMapValue(raw)
	if content == nil {
		return nil
	}
	media := openapiMapValue(content["application/json"])
	if media == nil {
		return nil
	}
	return openapiMapValue(media["schema"])
}

// JSONSchemaToNodes is the inverse of NodesToJSONSchema: it converts a
// JSON Schema (an object schema whose properties are the nodes, or a $ref
// to one) back into the ParamNode forest it describes. Property names are
// sorted for deterministic node order and Order values, the schema's
// "required" array marks required nodes, nested object schemas nest as
// children, array "items" become the array node's children, leaf types map
// directly and "description" is carried on every node that has one. $ref
// values are resolved against root with the passed visited set as cycle
// guard; subtrees behind cyclic, external, unresolvable or too-deeply
// nested references are skipped (the warning-recording variant used by
// ParseOpenAPIDocument reports them; this entry point drops them).
func JSONSchemaToNodes(schema map[string]interface{}, root map[string]interface{}, visited map[string]bool, depth int) []ParamNode {
	if visited == nil {
		visited = map[string]bool{}
	}
	var warnings []string
	return jsonSchemaToNodes(schema, root, visited, depth, &warnings)
}

// jsonSchemaToNodes is the warning-recording core of JSONSchemaToNodes: it
// converts an object schema (or a $ref to one) at the given depth into the
// nodes of its properties.
func jsonSchemaToNodes(schema map[string]interface{}, root map[string]interface{}, visited map[string]bool, depth int, warnings *[]string) []ParamNode {
	if schema == nil {
		return []ParamNode{}
	}
	if depth > openapiMaxSchemaDepth {
		openapiWarnDepth(warnings)
		return []ParamNode{}
	}
	return derefOpenAPISchema(schema, root, visited, warnings, func(s map[string]interface{}) []ParamNode {
		properties := openapiMapValue(s["properties"])
		if properties == nil {
			return []ParamNode{}
		}
		required := openapiStringSet(s["required"])
		nodes := make([]ParamNode, 0, len(properties))
		for i, key := range openapiSortedKeys(properties) {
			if node := openapiPropertyNode(key, properties[key], required[key], i, root, visited, depth+1, warnings); node != nil {
				nodes = append(nodes, *node)
			}
		}
		return nodes
	})
}

// openapiPropertyNode converts one property's schema into a ParamNode, or
// nil when the whole subtree must be skipped (depth cap or a skipped $ref).
func openapiPropertyNode(name string, schema interface{}, required bool, order int, root map[string]interface{}, visited map[string]bool, depth int, warnings *[]string) *ParamNode {
	if depth > openapiMaxSchemaDepth {
		openapiWarnDepth(warnings)
		return nil
	}
	schemaMap := openapiMapValue(schema)
	if schemaMap == nil {
		// Malformed property schema: keep the field visible as a leaf.
		return &ParamNode{Name: name, Type: "string", Required: required, Order: order, Children: []ParamNode{}}
	}
	return derefOpenAPISchema(schemaMap, root, visited, warnings, func(s map[string]interface{}) *ParamNode {
		node := &ParamNode{
			Name:     name,
			Type:     openapiSchemaNodeType(s),
			Required: required,
			Order:    order,
			Children: []ParamNode{},
		}
		if description := openapiStringValue(s["description"]); description != "" {
			node.Description = &description
		}
		switch node.Type {
		case "object":
			node.Children = jsonSchemaToNodes(s, root, visited, depth, warnings)
		case "array":
			node.Children = openapiArrayItemNodes(s["items"], root, visited, depth+1, warnings)
		}
		return node
	})
}

// openapiArrayItemNodes builds the children of an array node from its
// items schema: an object items schema contributes one child per property
// (mirroring how NodesToJSONSchema packs element fields into an object
// element schema); an absent or empty items schema contributes none; any
// other shape degrades to a single "item" child.
func openapiArrayItemNodes(items interface{}, root map[string]interface{}, visited map[string]bool, depth int, warnings *[]string) []ParamNode {
	itemsSchema := openapiMapValue(items)
	if len(itemsSchema) == 0 {
		return []ParamNode{}
	}
	return derefOpenAPISchema(itemsSchema, root, visited, warnings, func(s map[string]interface{}) []ParamNode {
		if openapiSchemaNodeType(s) == "object" {
			return jsonSchemaToNodes(s, root, visited, depth, warnings)
		}
		node := openapiPropertyNode("item", s, false, 0, root, visited, depth, warnings)
		if node == nil {
			return []ParamNode{}
		}
		return []ParamNode{*node}
	})
}

// derefOpenAPISchema resolves schema when it is a $ref and hands the
// effective schema to process; ref-to-ref chains are followed
// transitively. Each referenced name stays marked on the DFS path (removed
// afterwards), so cyclic references are detected while repeated sibling
// references remain legal. A ref that must be skipped — external or
// otherwise unsupported, unknown in components.schemas, or cyclic —
// records a warning and yields the zero value without calling process,
// which is what skips the offending subtree.
func derefOpenAPISchema[T any](schema map[string]interface{}, root map[string]interface{}, visited map[string]bool, warnings *[]string, process func(map[string]interface{}) T) T {
	var zero T
	ref, isRef := openapiRefString(schema)
	if !isRef {
		return process(schema)
	}
	if !strings.HasPrefix(ref, openapiSchemaRefPrefix) {
		*warnings = append(*warnings, fmt.Sprintf("skipped $ref %q: only local %q references are supported", ref, openapiSchemaRefPrefix))
		return zero
	}
	name := strings.TrimPrefix(ref, openapiSchemaRefPrefix)
	if visited[name] {
		*warnings = append(*warnings, fmt.Sprintf("skipped cyclic $ref %q", ref))
		return zero
	}
	target := openapiSchemaByName(root, name)
	if target == nil {
		*warnings = append(*warnings, fmt.Sprintf("skipped $ref %q: components.schemas has no %q entry", ref, name))
		return zero
	}
	visited[name] = true
	defer delete(visited, name)
	if _, isRef := openapiRefString(target); isRef {
		return derefOpenAPISchema(target, root, visited, warnings, process)
	}
	return process(target)
}

// openapiWarnDepth records the depth-cap warning.
func openapiWarnDepth(warnings *[]string) {
	*warnings = append(*warnings, fmt.Sprintf("skipped JSON Schema subtree nested deeper than %d levels", openapiMaxSchemaDepth))
}

// openapiSchemaNodeType maps a schema to its ParamNode type: recognized
// JSON Schema types pass through, with "integer" narrowing to "number" (the
// only numeric type ParamNode has); a schema without "type" but with
// "properties" reads as an object; anything else degrades to "string",
// mirroring the exporter's unknown-type degradation.
func openapiSchemaNodeType(schema map[string]interface{}) string {
	switch t := openapiStringValue(schema["type"]); t {
	case "string", "number", "boolean", "object", "array":
		return t
	case "integer":
		return "number"
	}
	if openapiMapValue(schema["properties"]) != nil {
		return "object"
	}
	return "string"
}

// openapiRefString returns the schema's $ref string when present.
func openapiRefString(schema map[string]interface{}) (string, bool) {
	if ref, ok := schema["$ref"].(string); ok && ref != "" {
		return ref, true
	}
	return "", false
}

// openapiSchemaByName looks up components.schemas[name] in the document.
func openapiSchemaByName(root map[string]interface{}, name string) map[string]interface{} {
	components := openapiMapValue(root["components"])
	schemas := openapiMapValue(components["schemas"])
	return openapiMapValue(schemas[name])
}

// openapiStringSet reads a JSON array of strings into a set.
func openapiStringSet(raw interface{}) map[string]bool {
	set := make(map[string]bool)
	for _, v := range openapiArrayValue(raw) {
		if s, ok := v.(string); ok {
			set[s] = true
		}
	}
	return set
}

// openapiSortedKeys returns the map's keys in sorted order, for
// deterministic iteration.
func openapiSortedKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// openapiStringValue returns v as a string, or "" for any other shape.
func openapiStringValue(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// openapiMapValue returns v as a JSON object, or nil for any other shape.
func openapiMapValue(v interface{}) map[string]interface{} {
	if m, ok := v.(map[string]interface{}); ok {
		return m
	}
	return nil
}

// openapiArrayValue returns v as a JSON array, or nil for any other shape.
func openapiArrayValue(v interface{}) []interface{} {
	if a, ok := v.([]interface{}); ok {
		return a
	}
	return nil
}
