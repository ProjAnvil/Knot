package models

import "slices"

// HTTPMethods lists the HTTP methods an API endpoint may declare.
var HTTPMethods = []string{"GET", "POST", "PUT", "DELETE", "PATCH"}

// APITypes lists the supported API types.
var APITypes = []string{"HTTP", "RPC"}

// ParamTypes lists the parameter categories (request or response body).
var ParamTypes = []string{"request", "response"}

// ParamValueTypes lists the allowed parameter value types.
var ParamValueTypes = []string{"string", "number", "boolean", "array", "object"}

// IsValidMethod reports whether m is one of the supported HTTP methods.
// Membership is case-sensitive.
func IsValidMethod(m string) bool {
	return slices.Contains(HTTPMethods, m)
}

// IsValidAPIType reports whether t is one of the supported API types.
// Membership is case-sensitive.
func IsValidAPIType(t string) bool {
	return slices.Contains(APITypes, t)
}

// IsValidParamType reports whether t is a valid parameter category.
// Membership is case-sensitive.
func IsValidParamType(t string) bool {
	return slices.Contains(ParamTypes, t)
}

// IsValidParamValueType reports whether t is a valid parameter value type.
// Membership is case-sensitive.
func IsValidParamValueType(t string) bool {
	return slices.Contains(ParamValueTypes, t)
}
