package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	models "github.com/ProjAnvil/knot/models"
)

// ParseKnotJSON parses a Knot JSON v1 export (as produced by
// BuildKnotDocument) into a models.ImportPlan. It is the direct inverse of the
// export builder per spec §8.3: unknown fields are ignored, methods are
// upper-cased, and type defaults to "HTTP" when absent. An unknown type
// value is normalized to "HTTP" with a recorded warning rather than failing.
// A UTF-8 BOM is tolerated so any file accepted by DetectImportFormat parses.
func ParseKnotJSON(data []byte) (*models.ImportPlan, error) {
	data = bytes.TrimPrefix(data, utf8BOM)

	var doc models.KnotExportDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("invalid Knot JSON file: %w", err)
	}
	if doc.Version != 1 {
		return nil, fmt.Errorf("unsupported Knot export version %d: only version 1 can be imported", doc.Version)
	}

	plan := &models.ImportPlan{
		Groups:   make([]models.ImportGroup, 0, len(doc.Groups)),
		Warnings: []string{},
	}
	for i, group := range doc.Groups {
		if group.Name == "" {
			return nil, fmt.Errorf("invalid Knot JSON file: group at index %d has an empty name", i)
		}
		importedGroup := models.ImportGroup{
			Name: group.Name,
			APIs: make([]models.ImportAPI, 0, len(group.APIs)),
		}
		for j, api := range group.APIs {
			if api.Name == "" {
				return nil, fmt.Errorf("invalid Knot JSON file: API at index %d in group %q has an empty name", j, group.Name)
			}
			importedAPI := models.ImportAPI{
				Name:     api.Name,
				Endpoint: api.Endpoint,
				Method:   strings.ToUpper(api.Method),
				Type:     normalizeKnotAPIType(api.Type),
				Note:     api.Note,
				Request:  nonNilParamNodes(api.Parameters.Request),
				Response: nonNilParamNodes(api.Parameters.Response),
			}
			if api.Type != "" && api.Type != "HTTP" && api.Type != "RPC" {
				warning := fmt.Sprintf("group %q API %q: unknown type %q normalized to \"HTTP\"", group.Name, api.Name, api.Type)
				plan.Warnings = append(plan.Warnings, warning)
			}
			importedGroup.APIs = append(importedGroup.APIs, importedAPI)
		}
		plan.Groups = append(plan.Groups, importedGroup)
	}
	return plan, nil
}

// normalizeKnotAPIType maps a knot export type field to "HTTP" or "RPC".
// An absent value defaults to "HTTP" per spec §8.3; any other value also
// degrades to "HTTP" (the caller records a warning for that case).
func normalizeKnotAPIType(apiType string) string {
	switch apiType {
	case "RPC":
		return "RPC"
	default:
		return "HTTP"
	}
}

// nonNilParamNodes returns the input slice, replacing nil with an empty
// slice so downstream consumers can iterate safely (the same convention
// models.TreeFromParams follows on the export side).
func nonNilParamNodes(nodes []models.ParamNode) []models.ParamNode {
	if nodes == nil {
		return []models.ParamNode{}
	}
	return nodes
}
