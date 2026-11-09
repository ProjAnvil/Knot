package models

import (
	"encoding/json"
	"testing"
)

// TestWireConstants locks the wire envelope identifiers: the format marker
// recognized by the import detector and the single supported version.
func TestWireConstants(t *testing.T) {
	if WireFormat != "knot" {
		t.Errorf("Expected WireFormat %q, got %q", "knot", WireFormat)
	}
	if WireVersion != 1 {
		t.Errorf("Expected WireVersion 1, got %d", WireVersion)
	}
}

// TestKnotExportDocumentMarshal verifies the JSON tags of the Knot v1 wire
// envelope at the type level: the document, group, API and parameter block
// keys, and the null encoding of a missing note.
func TestKnotExportDocumentMarshal(t *testing.T) {
	// Arrange - one group with one API whose note is set and both
	// parameter trees populated, plus a second API with a nil note.
	note := "Requires admin role"
	doc := KnotExportDocument{
		Format:     WireFormat,
		Version:    WireVersion,
		ExportedAt: "2026-11-10T12:30:00Z",
		Groups: []KnotExportGroup{
			{
				Name:  "Users",
				Order: 1,
				APIs: []KnotExportAPI{
					{
						Name:     "Create user",
						Endpoint: "/users",
						Method:   "POST",
						Type:     "HTTP",
						Order:    0,
						Note:     &note,
						Parameters: KnotExportParams{
							Request: []ParamNode{
								{Name: "name", Type: "string", Required: true, Order: 0, Children: []ParamNode{}},
							},
							Response: []ParamNode{},
						},
					},
					{
						Name:       "List users",
						Endpoint:   "/users",
						Method:     "GET",
						Type:       "HTTP",
						Order:      1,
						Parameters: KnotExportParams{Request: []ParamNode{}, Response: []ParamNode{}},
					},
				},
			},
		},
	}

	// Act
	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("Failed to marshal document: %v", err)
	}

	// Assert - the exact wire layout with keys in struct order; a nil note
	// marshals as an explicit null so absent vs empty stays distinguishable.
	want := `{"format":"knot","version":1,"exportedAt":"2026-11-10T12:30:00Z","groups":[{"name":"Users","order":1,"apis":[{"name":"Create user","endpoint":"/users","method":"POST","type":"HTTP","order":0,"note":"Requires admin role","parameters":{"request":[{"name":"name","type":"string","required":true,"description":null,"order":0,"children":[]}],"response":[]}},{"name":"List users","endpoint":"/users","method":"GET","type":"HTTP","order":1,"note":null,"parameters":{"request":[],"response":[]}}]}]}`
	if string(data) != want {
		t.Errorf("Marshaled wire document mismatch:\n got: %s\nwant: %s", data, want)
	}
}

// TestKnotExportDocumentUnmarshal verifies the envelope parses the keys it
// writes (round-trip at the type level, independent of any builder).
func TestKnotExportDocumentUnmarshal(t *testing.T) {
	// Arrange
	raw := `{"format":"knot","version":1,"exportedAt":"2026-11-10T12:30:00Z","groups":[{"name":"Users","order":1,"apis":[{"name":"Create user","endpoint":"/users","method":"POST","type":"HTTP","order":0,"note":"Requires admin role","parameters":{"request":[{"name":"name","type":"string","required":true,"order":0,"children":[]}],"response":[]}}]}]}`

	// Act
	var doc KnotExportDocument
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		t.Fatalf("Failed to unmarshal document: %v", err)
	}

	// Assert
	if doc.Format != WireFormat || doc.Version != WireVersion {
		t.Errorf("Expected envelope %q/v%d, got %q/v%d", WireFormat, WireVersion, doc.Format, doc.Version)
	}
	if len(doc.Groups) != 1 || doc.Groups[0].Name != "Users" || doc.Groups[0].Order != 1 {
		t.Fatalf("Unexpected groups: %+v", doc.Groups)
	}
	api := doc.Groups[0].APIs[0]
	if api.Note == nil || *api.Note != "Requires admin role" {
		t.Errorf("Expected note carried over as a non-nil pointer, got %v", api.Note)
	}
	if len(api.Parameters.Request) != 1 || api.Parameters.Request[0].Name != "name" {
		t.Errorf("Unexpected request tree: %+v", api.Parameters.Request)
	}
	if api.Parameters.Response == nil || len(api.Parameters.Response) != 0 {
		t.Errorf("Expected empty non-nil response slice, got %v", api.Parameters.Response)
	}
}
