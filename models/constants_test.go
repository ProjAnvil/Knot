package models

import "testing"

// TestIsValidMethod verifies HTTP method membership (case-sensitive).
func TestIsValidMethod(t *testing.T) {
	tests := []struct {
		name   string
		method string
		want   bool
	}{
		{"GET is valid", "GET", true},
		{"POST is valid", "POST", true},
		{"PUT is valid", "PUT", true},
		{"DELETE is valid", "DELETE", true},
		{"PATCH is valid", "PATCH", true},
		{"lowercase get is rejected", "get", false},
		{"HEAD is not in the allowed set", "HEAD", false},
		{"OPTIONS is not in the allowed set", "OPTIONS", false},
		{"empty string is rejected", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidMethod(tt.method); got != tt.want {
				t.Errorf("IsValidMethod(%q) = %v, want %v", tt.method, got, tt.want)
			}
		})
	}
}

// TestIsValidAPIType verifies API type membership (case-sensitive).
func TestIsValidAPIType(t *testing.T) {
	tests := []struct {
		name    string
		apiType string
		want    bool
	}{
		{"HTTP is valid", "HTTP", true},
		{"RPC is valid", "RPC", true},
		{"lowercase http is rejected", "http", false},
		{"GraphQL is not in the allowed set", "GraphQL", false},
		{"empty string is rejected", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidAPIType(tt.apiType); got != tt.want {
				t.Errorf("IsValidAPIType(%q) = %v, want %v", tt.apiType, got, tt.want)
			}
		})
	}
}

// TestIsValidParamType verifies parameter category membership (case-sensitive).
func TestIsValidParamType(t *testing.T) {
	tests := []struct {
		name      string
		paramType string
		want      bool
	}{
		{"request is valid", "request", true},
		{"response is valid", "response", true},
		{"Request is rejected (case-sensitive)", "Request", false},
		{"header is not in the allowed set", "header", false},
		{"empty string is rejected", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidParamType(tt.paramType); got != tt.want {
				t.Errorf("IsValidParamType(%q) = %v, want %v", tt.paramType, got, tt.want)
			}
		})
	}
}

// TestIsValidParamValueType verifies parameter value type membership (case-sensitive).
func TestIsValidParamValueType(t *testing.T) {
	tests := []struct {
		name     string
		valueTyp string
		want     bool
	}{
		{"string is valid", "string", true},
		{"number is valid", "number", true},
		{"boolean is valid", "boolean", true},
		{"array is valid", "array", true},
		{"object is valid", "object", true},
		{"integer is not in the allowed set", "integer", false},
		{"String is rejected (case-sensitive)", "String", false},
		{"empty string is rejected", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidParamValueType(tt.valueTyp); got != tt.want {
				t.Errorf("IsValidParamValueType(%q) = %v, want %v", tt.valueTyp, got, tt.want)
			}
		})
	}
}

// TestConstantValues locks the exported constant slice contents.
func TestConstantValues(t *testing.T) {
	assertSlice := func(name string, got, want []string) {
		t.Helper()
		if len(got) != len(want) {
			t.Errorf("%s = %v, want %v", name, got, want)
			return
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("%s[%d] = %q, want %q", name, i, got[i], want[i])
			}
		}
	}

	assertSlice("HTTPMethods", HTTPMethods, []string{"GET", "POST", "PUT", "DELETE", "PATCH"})
	assertSlice("APITypes", APITypes, []string{"HTTP", "RPC"})
	assertSlice("ParamTypes", ParamTypes, []string{"request", "response"})
	assertSlice("ParamValueTypes", ParamValueTypes, []string{"string", "number", "boolean", "array", "object"})
}
