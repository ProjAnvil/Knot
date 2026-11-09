package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	models "github.com/ProjAnvil/knot/models"
)

// utf8BOM is the byte-order mark some editors (typically on Windows) prepend
// to UTF-8 files. It must be stripped before JSON probing or unmarshalling.
var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// DetectImportFormat inspects raw file bytes and decides which importer
// should handle them, per spec §8.2. A UTF-8 BOM is stripped before probing,
// then the document is unmarshalled into a raw root object and matched
// against the (disjoint) format signatures in order:
//
//  1. root "format" == "knot"                          → knot-json
//  2. root "info.schema" contains schema.getpostman.com → postman (v2.0 and v2.1)
//  3. root "openapi" with major version "3"             → openapi
//  4. root "swagger" present                            → error naming OpenAPI 3
//
// Non-JSON, empty, or unrecognized input is an error. All returned errors
// carry user-facing messages suitable for a 415-style response.
func DetectImportFormat(data []byte) (models.ImportFormat, error) {
	data = bytes.TrimPrefix(data, utf8BOM)
	if len(bytes.TrimSpace(data)) == 0 {
		return "", errors.New("import file is empty")
	}

	var root map[string]json.RawMessage
	if err := json.Unmarshal(data, &root); err != nil {
		return "", fmt.Errorf("import file is not valid JSON: %w", err)
	}

	// Signature 1: Knot's own export marker.
	if raw, ok := root["format"]; ok {
		var format string
		if err := json.Unmarshal(raw, &format); err == nil && format == "knot" {
			return models.ImportFormatKnot, nil
		}
	}

	// Signature 2: Postman collection schema URL (v2.0 and v2.1 both match).
	if raw, ok := root["info"]; ok {
		var info struct {
			Schema string `json:"schema"`
		}
		if err := json.Unmarshal(raw, &info); err == nil && strings.Contains(info.Schema, "schema.getpostman.com") {
			return models.ImportFormatPostman, nil
		}
	}

	// Signature 3: OpenAPI document whose major version is 3.
	if raw, ok := root["openapi"]; ok {
		var version string
		if err := json.Unmarshal(raw, &version); err == nil {
			if major, _, _ := strings.Cut(version, "."); major == "3" {
				return models.ImportFormatOpenAPI, nil
			}
		}
	}

	// Signature 4: Swagger 2.0 gets a dedicated hint instead of a generic error.
	if _, ok := root["swagger"]; ok {
		return "", errors.New("unsupported format: Swagger 2.0 file detected, OpenAPI 3.x is required; convert the file to OpenAPI 3 and try again")
	}

	return "", errors.New("unsupported file format: expected a Knot JSON export, Postman Collection v2.0/v2.1, or OpenAPI 3.x document")
}
