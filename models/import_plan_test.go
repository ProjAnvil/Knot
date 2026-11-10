package models

import (
	"encoding/json"
	"testing"
)

// TestImportResultJSONTags verifies that models.ImportResult marshals to the
// exact byte shape the v1.0 import handler served: same field order, same
// JSON keys (format/groupsCreated/groupsMerged/apisImported/
// parametersImported/warnings). Relocating the type from
// backend/internal/services must not change one byte of the wire format.
func TestImportResultJSONTags(t *testing.T) {
	// Populated result: field order and keys must match the v1.0 response
	// byte for byte (encoding/json emits fields in declaration order).
	result := ImportResult{
		Format:             "knot-json",
		GroupsCreated:      1,
		GroupsMerged:       2,
		APIsImported:       3,
		ParametersImported: 4,
		Warnings:           []string{"unsupported field skipped", "duplicate name renamed"},
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal ImportResult: %v", err)
	}

	want := `{"format":"knot-json","groupsCreated":1,"groupsMerged":2,"apisImported":3,"parametersImported":4,"warnings":["unsupported field skipped","duplicate name renamed"]}`
	if string(data) != want {
		t.Errorf("ImportResult JSON mismatch:\n got: %s\nwant: %s", data, want)
	}

	// Zero value keeps the v1.0 shape too: zero counters and a null warnings
	// slice (a nil []string marshals as null, exactly like the old struct).
	zero, err := json.Marshal(ImportResult{})
	if err != nil {
		t.Fatalf("Failed to marshal zero ImportResult: %v", err)
	}
	wantZero := `{"format":"","groupsCreated":0,"groupsMerged":0,"apisImported":0,"parametersImported":0,"warnings":null}`
	if string(zero) != wantZero {
		t.Errorf("Zero ImportResult JSON mismatch:\n got: %s\nwant: %s", zero, wantZero)
	}
}
