package services

import (
	"strings"
	"time"
)

// KnotExportDocument is the top-level structure of a Knot JSON v1 export file.
type KnotExportDocument struct {
	Format     string            `json:"format"`
	Version    int               `json:"version"`
	ExportedAt string            `json:"exportedAt"`
	Groups     []KnotExportGroup `json:"groups"`
}

// KnotExportGroup is one API group within a Knot export document.
type KnotExportGroup struct {
	Name  string          `json:"name"`
	Order int             `json:"order"`
	APIs  []KnotExportAPI `json:"apis"`
}

// KnotExportAPI is one API entry within a group.
type KnotExportAPI struct {
	Name       string           `json:"name"`
	Endpoint   string           `json:"endpoint"`
	Method     string           `json:"method"`
	Type       string           `json:"type"`
	Order      int              `json:"order"`
	Note       *string          `json:"note"`
	Parameters KnotExportParams `json:"parameters"`
}

// KnotExportParams holds the request and response parameter trees of one API.
type KnotExportParams struct {
	Request  []ParamNode `json:"request"`
	Response []ParamNode `json:"response"`
}

// BuildKnotDocument assembles the Knot JSON v1 export document from APIs with
// their parameters. APIs are grouped by API.GroupID in first-appearance order
// (tracked with a seen-map over the input slice, never map iteration, so group
// order is deterministic); each group carries the GroupOrder of its APIs.
// Parameter rows are converted to nested ParamNode trees via TreeFromParams.
func BuildKnotDocument(apis []APIWithParams, now time.Time) *KnotExportDocument {
	groups := make([]KnotExportGroup, 0)
	groupIndexByGroupID := make(map[uint]int)

	for _, api := range apis {
		groupID := api.API.GroupID
		index, seen := groupIndexByGroupID[groupID]
		if !seen {
			index = len(groups)
			groupIndexByGroupID[groupID] = index
			groups = append(groups, KnotExportGroup{
				Name:  api.GroupName,
				Order: api.GroupOrder,
				APIs:  make([]KnotExportAPI, 0),
			})
		}

		groups[index].APIs = append(groups[index].APIs, KnotExportAPI{
			Name:     api.API.Name,
			Endpoint: api.API.Endpoint,
			Method:   strings.ToUpper(api.API.Method),
			Type:     api.API.Type,
			Order:    api.API.Order,
			Note:     api.API.Note,
			Parameters: KnotExportParams{
				Request:  TreeFromParams(api.RequestParameters),
				Response: TreeFromParams(api.ResponseParameters),
			},
		})
	}

	return &KnotExportDocument{
		Format:     "knot",
		Version:    1,
		ExportedAt: now.UTC().Format(time.RFC3339),
		Groups:     groups,
	}
}
