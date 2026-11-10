package services

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	models "github.com/ProjAnvil/knot/models"
)

// postmanSchemaURL is the schema URL identifying a Postman Collection
// v2.1.0 document.
const postmanSchemaURL = "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"

// PostmanCollection is the root object of a Postman Collection v2.1
// document built by Knot.
type PostmanCollection struct {
	Info PostmanInfo   `json:"info"`
	Item []PostmanItem `json:"item"`
}

// PostmanInfo identifies the collection and the schema it follows.
type PostmanInfo struct {
	Name        string `json:"name"`
	PostmanID   string `json:"_postman_id"`
	Description string `json:"description,omitempty"`
	Schema      string `json:"schema"`
}

// PostmanItem is either a folder (one API group, only Name and Item set)
// or a request leaf (Name, Request and, when the API has response
// parameters, saved example responses).
type PostmanItem struct {
	Name      string            `json:"name"`
	XKnotType string            `json:"x-knot-type,omitempty"`
	Item      []PostmanItem     `json:"item,omitempty"`
	Request   *PostmanRequest   `json:"request,omitempty"`
	Response  []PostmanResponse `json:"response,omitempty"`
}

// PostmanRequest describes how to send one request. Header is always an
// empty array: Knot documents bodies, not headers.
type PostmanRequest struct {
	Method      string          `json:"method"`
	Header      []PostmanHeader `json:"header"`
	URL         PostmanURL      `json:"url"`
	Description string          `json:"description,omitempty"`
	Body        *PostmanBody    `json:"body,omitempty"`
}

// PostmanHeader is one header entry; exported collections carry none.
type PostmanHeader struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// PostmanURL holds the raw request URL.
type PostmanURL struct {
	Raw string `json:"raw"`
}

// PostmanBody carries a raw request body plus its language hint.
type PostmanBody struct {
	Mode    string             `json:"mode"`
	Raw     string             `json:"raw"`
	Options PostmanBodyOptions `json:"options"`
}

// PostmanBodyOptions hints at the raw body language.
type PostmanBodyOptions struct {
	Raw PostmanRawOptions `json:"raw"`
}

// PostmanRawOptions names the language of a raw body.
type PostmanRawOptions struct {
	Language string `json:"language"`
}

// PostmanResponse is one saved example response on a request item.
type PostmanResponse struct {
	Name string `json:"name"`
	Body string `json:"body"`
}

// BuildPostmanCollection assembles a Postman Collection v2.1 document
// from APIs with their parameters. Each API group becomes one folder
// item, created on the group's first appearance (tracked with a seen-map
// over the input slice, never map iteration, so folder order is
// deterministic); groups with zero selected APIs never appear because
// folders are only created for selected APIs. Request and response
// example bodies reuse the GenerateExampleJSON tree walker and are
// omitted when the API has no parameters on that side. RPC-like APIs are
// exported as POST requests carrying the x-knot-type marker so Knot's
// importer can restore their classification.
func BuildPostmanCollection(apis []APIWithParams, now time.Time) *PostmanCollection {
	items := make([]PostmanItem, 0)
	folderIndexByGroupID := make(map[uint]int)

	for _, api := range apis {
		groupID := api.API.GroupID
		index, seen := folderIndexByGroupID[groupID]
		if !seen {
			index = len(items)
			folderIndexByGroupID[groupID] = index
			items = append(items, PostmanItem{
				Name: api.GroupName,
				Item: make([]PostmanItem, 0),
			})
		}

		items[index].Item = append(items[index].Item, newPostmanRequestItem(api))
	}

	return &PostmanCollection{
		Info: PostmanInfo{
			Name:        "Knot Export",
			PostmanID:   newPostmanID(now),
			Description: "Exported from Knot at " + now.UTC().Format(time.RFC3339),
			Schema:      postmanSchemaURL,
		},
		Item: items,
	}
}

// newPostmanRequestItem builds the leaf item for one API: the method
// upper-cased (RPC-like APIs always POST), the endpoint as the raw URL,
// the note as the request description, and example bodies from the
// request and response parameter trees.
func newPostmanRequestItem(api APIWithParams) PostmanItem {
	item := PostmanItem{Name: api.API.Name}

	method := strings.ToUpper(api.API.Method)
	if exportsAsRPC(api.API) {
		method = "POST"
		item.XKnotType = "rpc"
	}

	request := &PostmanRequest{
		Method:      method,
		Header:      []PostmanHeader{},
		URL:         PostmanURL{Raw: api.API.Endpoint},
		Description: noteText(api.API.Note),
	}
	if body := exampleBody(api.RequestParameters); body != "" {
		request.Body = &PostmanBody{
			Mode:    "raw",
			Raw:     body,
			Options: PostmanBodyOptions{Raw: PostmanRawOptions{Language: "json"}},
		}
	}
	item.Request = request

	if body := exampleBody(api.ResponseParameters); body != "" {
		item.Response = []PostmanResponse{{Name: "Example 200", Body: body}}
	}

	return item
}

// exampleBody renders the GenerateExampleJSON example of a parameter set
// as indented JSON, or "" when there are no parameters. Mirrors the HTML
// exporter, which feeds BuildParameterTree into GenerateExampleJSON.
func exampleBody(params []models.Parameter) string {
	example := GenerateExampleJSON(BuildParameterTree(params))
	if len(example) == 0 {
		return ""
	}
	raw, err := json.MarshalIndent(example, "", "  ")
	if err != nil {
		return ""
	}
	return string(raw)
}

// newPostmanID derives info._postman_id deterministically from the export
// instant (unix seconds). A stable ID per instant makes two exports of the
// same data in the same second byte-identical — required by the facade and
// HTTP handler parity contract, which share BuildExportDocument — while
// different seconds still yield different IDs. The value keeps the RFC 4122
// version-4 and variant bits and the canonical 8-4-4-4-12 hex shape, so it
// remains a well-formed UUID for Postman.
func newPostmanID(now time.Time) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("knot-postman-%d", now.Unix())))
	b := sum[:16]
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // RFC 4122 variant

	var dst [36]byte
	hex.Encode(dst[0:8], b[0:4])
	dst[8] = '-'
	hex.Encode(dst[9:13], b[4:6])
	dst[13] = '-'
	hex.Encode(dst[14:18], b[6:8])
	dst[18] = '-'
	hex.Encode(dst[19:23], b[8:10])
	dst[23] = '-'
	hex.Encode(dst[24:36], b[10:16])
	return string(dst[:])
}
