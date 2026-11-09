package services

import (
	"strings"
	"time"

	models "github.com/ProjAnvil/knot/models"
)

// BuildKnotDocument assembles the Knot JSON v1 export document from APIs with
// their parameters. APIs are grouped by API.GroupID in first-appearance order
// (tracked with a seen-map over the input slice, never map iteration, so group
// order is deterministic); each group carries the GroupOrder of its APIs.
// Parameter rows are converted to nested models.ParamNode trees via models.TreeFromParams.
func BuildKnotDocument(apis []APIWithParams, now time.Time) *models.KnotExportDocument {
	groups := make([]models.KnotExportGroup, 0)
	groupIndexByGroupID := make(map[uint]int)

	for _, api := range apis {
		groupID := api.API.GroupID
		index, seen := groupIndexByGroupID[groupID]
		if !seen {
			index = len(groups)
			groupIndexByGroupID[groupID] = index
			groups = append(groups, models.KnotExportGroup{
				Name:  api.GroupName,
				Order: api.GroupOrder,
				APIs:  make([]models.KnotExportAPI, 0),
			})
		}

		groups[index].APIs = append(groups[index].APIs, models.KnotExportAPI{
			Name:     api.API.Name,
			Endpoint: api.API.Endpoint,
			Method:   strings.ToUpper(api.API.Method),
			Type:     api.API.Type,
			Order:    api.API.Order,
			Note:     api.API.Note,
			Parameters: models.KnotExportParams{
				Request:  models.TreeFromParams(api.RequestParameters),
				Response: models.TreeFromParams(api.ResponseParameters),
			},
		})
	}

	return &models.KnotExportDocument{
		Format:     models.WireFormat,
		Version:    models.WireVersion,
		ExportedAt: now.UTC().Format(time.RFC3339),
		Groups:     groups,
	}
}
