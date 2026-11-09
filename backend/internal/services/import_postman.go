package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	models "github.com/ProjAnvil/knot/models"
)

// postmanParser carries the accumulator state while walking a Postman
// collection: the plan under construction plus a group-name → index map so
// folders that resolve to the same path merge into a single group.
type postmanParser struct {
	plan           *models.ImportPlan
	groupIndex     map[string]int
	unnamedDropped int
}

// ParsePostmanCollection parses a Postman Collection v2.0/v2.1 document
// into a models.ImportPlan per spec §8.3. item[] is walked recursively: an item
// with a nested item array is a folder whose group name flattens the
// nesting with "/" separators (top-level folders use their own name),
// while a lone top-level request lands in a group named after info.name.
// request.url and request.description each accept both the v2.1 object
// form and the plain-string (v2.0) form. An item carrying the custom
// property "x-knot-type": "rpc" imports as Type "RPC" with an empty
// method, mirroring the Postman exporter. A raw request body that parses
// as a JSON object is converted to request ParamNodes; unconvertible
// bodies record a warning instead of failing. A UTF-8 BOM is tolerated so
// any file accepted by DetectImportFormat parses.
func ParsePostmanCollection(data []byte) (*models.ImportPlan, error) {
	data = bytes.TrimPrefix(data, utf8BOM)

	var root map[string]interface{}
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("invalid Postman collection file: %w", err)
	}

	parser := &postmanParser{
		plan:       &models.ImportPlan{Groups: []models.ImportGroup{}, Warnings: []string{}},
		groupIndex: make(map[string]int),
	}
	collectionName := postmanStringValue(postmanMapValue(root["info"])["name"])
	parser.walkItems(postmanArrayValue(root["item"]), "", collectionName)

	if parser.unnamedDropped > 0 {
		parser.plan.Warnings = append(parser.plan.Warnings,
			fmt.Sprintf("skipped %d request(s) whose group name resolved to empty: the collection has no info.name and no folder name applies", parser.unnamedDropped))
	}
	return parser.plan, nil
}

// walkItems processes sibling items. Direct request children are added to
// requestGroup; child folders recurse with a path derived from folderPath
// (empty at the collection root, so top-level folders use their own name
// and deeper nesting extends the "/" chain). Groups are created lazily by
// addRequest, so folders containing only subfolders contribute no empty
// group.
func (p *postmanParser) walkItems(items []interface{}, folderPath, requestGroup string) {
	for _, rawItem := range items {
		item, ok := rawItem.(map[string]interface{})
		if !ok {
			// Schema-violating entry that is neither folder nor request; skip it.
			continue
		}
		if children := postmanArrayValue(item["item"]); children != nil {
			childPath := joinGroupPath(folderPath, postmanStringValue(item["name"]))
			p.walkItems(children, childPath, childPath)
		} else {
			p.addRequest(item, requestGroup)
		}
	}
}

// addRequest converts one request item into a models.ImportAPI appended to the
// group named groupName. A request whose group name resolves to empty is
// dropped with a warning (never an error); a missing item name falls back
// to the endpoint, then to "Unnamed request", with a warning; an item
// without a request object is skipped with a warning.
func (p *postmanParser) addRequest(item map[string]interface{}, groupName string) {
	if groupName == "" {
		p.unnamedDropped++
		return
	}

	request := postmanMapValue(item["request"])
	if request == nil {
		p.plan.Warnings = append(p.plan.Warnings,
			fmt.Sprintf("skipped Postman item %q: it has no request object", postmanStringValue(item["name"])))
		return
	}

	name := postmanStringValue(item["name"])
	endpoint := postmanEndpoint(request["url"])
	if name == "" {
		if endpoint == "" {
			name = "Unnamed request"
		} else {
			name = endpoint
		}
		p.plan.Warnings = append(p.plan.Warnings,
			fmt.Sprintf("group %q: request without a name imported as %q", groupName, name))
	}

	api := models.ImportAPI{
		Name:     name,
		Endpoint: endpoint,
		Request:  []models.ParamNode{},
		Response: []models.ParamNode{},
	}
	if postmanStringValue(item["x-knot-type"]) == "rpc" {
		api.Type = "RPC"
	} else {
		api.Type = "HTTP"
		api.Method = strings.ToUpper(postmanStringValue(request["method"]))
	}
	if note := postmanDescription(request["description"]); note != "" {
		api.Note = &note
	}
	api.Request = p.postmanRequestBody(request["body"], name)

	index, seen := p.groupIndex[groupName]
	if !seen {
		index = len(p.plan.Groups)
		p.groupIndex[groupName] = index
		p.plan.Groups = append(p.plan.Groups, models.ImportGroup{Name: groupName, APIs: []models.ImportAPI{}})
	}
	p.plan.Groups[index].APIs = append(p.plan.Groups[index].APIs, api)
}

// postmanRequestBody converts a raw-mode JSON body to request ParamNodes.
// Other body modes (formdata, urlencoded, file, graphql) are ignored per
// spec §8.3. A raw body that does not parse as a JSON object records a
// warning naming the item and yields no nodes.
func (p *postmanParser) postmanRequestBody(body interface{}, itemName string) []models.ParamNode {
	bodyMap, ok := body.(map[string]interface{})
	if !ok || postmanStringValue(bodyMap["mode"]) != "raw" {
		return []models.ParamNode{}
	}

	raw := postmanStringValue(bodyMap["raw"])
	var decoded interface{}
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		p.plan.Warnings = append(p.plan.Warnings,
			fmt.Sprintf("request %q: raw body is not valid JSON, request parameters skipped", itemName))
		return []models.ParamNode{}
	}
	if _, isObject := decoded.(map[string]interface{}); !isObject {
		p.plan.Warnings = append(p.plan.Warnings,
			fmt.Sprintf("request %q: raw body is not a JSON object, request parameters skipped", itemName))
		return []models.ParamNode{}
	}
	return models.ParamNodesFromJSONValue(decoded)
}

// postmanEndpoint extracts the endpoint from request.url, which Postman
// v2.1 writes as an object (endpoint taken from its raw field) and v2.0
// as a plain string. {{variable}} placeholders pass through verbatim.
func postmanEndpoint(rawURL interface{}) string {
	if s, ok := rawURL.(string); ok {
		return s
	}
	if m, ok := rawURL.(map[string]interface{}); ok {
		return postmanStringValue(m["raw"])
	}
	return ""
}

// postmanDescription extracts the note from request.description, which may
// be a plain string or a {content: "..."} object. An empty result is
// treated as no note by the caller.
func postmanDescription(desc interface{}) string {
	if s, ok := desc.(string); ok {
		return s
	}
	if m, ok := desc.(map[string]interface{}); ok {
		return postmanStringValue(m["content"])
	}
	return ""
}

// postmanStringValue returns v as a string, or "" for any other shape.
func postmanStringValue(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// postmanMapValue returns v as a JSON object, or nil for any other shape.
func postmanMapValue(v interface{}) map[string]interface{} {
	if m, ok := v.(map[string]interface{}); ok {
		return m
	}
	return nil
}

// postmanArrayValue returns v as a JSON array, or nil for any other shape.
func postmanArrayValue(v interface{}) []interface{} {
	if a, ok := v.([]interface{}); ok {
		return a
	}
	return nil
}

// joinGroupPath extends a folder path chain with one more folder name,
// skipping empty components so unnamed folders degrade gracefully.
func joinGroupPath(parent, name string) string {
	switch {
	case parent == "":
		return name
	case name == "":
		return parent
	default:
		return parent + "/" + name
	}
}
