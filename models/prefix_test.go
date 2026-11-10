package models

import "testing"

// TestSetTablePrefix verifies the prefix-driven table naming contract: the
// default prefix is the empty string (v1.0 physical table names groups/apis/
// parameters), SetTablePrefix applies to every model's TableName, and
// restoring "" returns the unprefixed names.
func TestSetTablePrefix(t *testing.T) {
	// Default: no prefix, v1.0 table names.
	if TablePrefix != "" {
		t.Fatalf("Expected default TablePrefix to be empty, got %q", TablePrefix)
	}
	if got := (Group{}).TableName(); got != "groups" {
		t.Errorf("Expected Group table %q with default prefix, got %q", "groups", got)
	}
	if got := (API{}).TableName(); got != "apis" {
		t.Errorf("Expected API table %q with default prefix, got %q", "apis", got)
	}
	if got := (Parameter{}).TableName(); got != "parameters" {
		t.Errorf("Expected Parameter table %q with default prefix, got %q", "parameters", got)
	}

	// Always restore the zero prefix so later tests see v1.0 names.
	defer SetTablePrefix("")

	SetTablePrefix("knot_")
	if TablePrefix != "knot_" {
		t.Fatalf("Expected TablePrefix to be %q after SetTablePrefix, got %q", "knot_", TablePrefix)
	}
	if got := (Group{}).TableName(); got != "knot_groups" {
		t.Errorf("Expected Group table %q with prefix %q, got %q", "knot_groups", "knot_", got)
	}
	if got := (API{}).TableName(); got != "knot_apis" {
		t.Errorf("Expected API table %q with prefix %q, got %q", "knot_apis", "knot_", got)
	}
	if got := (Parameter{}).TableName(); got != "knot_parameters" {
		t.Errorf("Expected Parameter table %q with prefix %q, got %q", "knot_parameters", "knot_", got)
	}

	// Restoring the empty prefix must return the bare v1.0 names.
	SetTablePrefix("")
	if got := (Group{}).TableName(); got != "groups" {
		t.Errorf("Expected Group table %q after restoring empty prefix, got %q", "groups", got)
	}
}
