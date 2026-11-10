package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	models "github.com/ProjAnvil/knot/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Export input-validation sentinels. They let the HTTP handler and the
// embeddable engine facade (package knot) map the same failures to their
// respective error surfaces without either side string-matching messages.
var (
	// ErrNoAPIIDs is returned when an export request carries no API IDs.
	ErrNoAPIIDs = errors.New("no API IDs provided")
	// ErrUnsupportedExportFormat is returned for an unknown export format
	// identifier; "" and "html" are the HTML default.
	ErrUnsupportedExportFormat = errors.New("unsupported export format")
)

// defaultExportLocale matches the handler's cookie default ("zh").
const defaultExportLocale = "zh"

// BuildExportDocument fetches the selected APIs with their groups and
// parameters, renders them in the requested format, and returns the payload
// together with the download filename and Content-Type. It is the single
// source of export truth shared by the HTTP handler (handlers/export.go) and
// the embeddable engine facade (backend/knot.go); both surfaces therefore
// produce byte-identical output for the same inputs and instant.
//
// format follows the handler contract: "" and "html" render the standalone
// HTML document; "json", "openapi" and "postman" render the respective JSON
// documents. locale selects the HTML template language and defaults to "zh"
// when empty. now stamps time-derived fields (exportedAt, descriptions) and
// the JSON filenames, all with second granularity.
func BuildExportDocument(db *gorm.DB, apiIDs []uint, format, locale string, now time.Time) ([]byte, string, string, error) {
	if len(apiIDs) == 0 {
		return nil, "", "", ErrNoAPIIDs
	}

	// Fetch APIs with their parameters and group information
	var apis []models.API
	if err := db.Where("id IN ?", apiIDs).Find(&apis).Error; err != nil {
		return nil, "", "", fmt.Errorf("failed to fetch APIs: %w", err)
	}

	// Fetch all groups that these APIs belong to
	groupIDs := make([]uint, 0)
	groupIDMap := make(map[uint]bool)
	for _, api := range apis {
		if !groupIDMap[api.GroupID] {
			groupIDs = append(groupIDs, api.GroupID)
			groupIDMap[api.GroupID] = true
		}
	}

	var groups []models.Group
	if len(groupIDs) > 0 {
		if err := db.Where("id IN ?", groupIDs).Find(&groups).Error; err != nil {
			return nil, "", "", fmt.Errorf("failed to fetch groups: %w", err)
		}
	}

	groupMap := make(map[uint]models.Group)
	for _, g := range groups {
		groupMap[g.ID] = g
	}

	// Fetch parameters for each API
	var apisWithParams []APIWithParams
	for _, api := range apis {
		var allParams []models.Parameter
		if err := db.Where("api_id = ?", api.ID).Order(clause.OrderByColumn{Column: clause.Column{Name: "order"}}).Find(&allParams).Error; err != nil {
			return nil, "", "", fmt.Errorf("failed to fetch parameters: %w", err)
		}

		requestParams := make([]models.Parameter, 0)
		responseParams := make([]models.Parameter, 0)

		for _, p := range allParams {
			if p.ParamType == "request" {
				requestParams = append(requestParams, p)
			} else if p.ParamType == "response" {
				responseParams = append(responseParams, p)
			}
		}

		group := groupMap[api.GroupID]
		groupName := group.Name
		if groupName == "" {
			groupName = "Ungrouped"
		}

		apisWithParams = append(apisWithParams, APIWithParams{
			API:                api,
			GroupName:          groupName,
			GroupOrder:         group.Order,
			RequestParameters:  requestParams,
			ResponseParameters: responseParams,
		})
	}

	if locale == "" {
		locale = defaultExportLocale
	}

	switch format {
	case "", "html":
		html := GenerateHTML(apisWithParams, locale)
		return []byte(html), "api-docs.html", "text/html; charset=utf-8", nil
	case "json", "openapi", "postman":
		var doc any
		var filenamePrefix string
		switch format {
		case "json":
			doc = BuildKnotDocument(apisWithParams, now)
			filenamePrefix = "knot-export-"
		case "openapi":
			doc = BuildOpenAPIDocument(apisWithParams)
			filenamePrefix = "openapi-"
		default: // postman
			doc = BuildPostmanCollection(apisWithParams, now)
			filenamePrefix = "postman-"
		}

		data, err := json.Marshal(doc)
		if err != nil {
			return nil, "", "", fmt.Errorf("failed to encode export document: %w", err)
		}
		return data, fmt.Sprintf("%s%d.json", filenamePrefix, now.Unix()), "application/json", nil
	default:
		return nil, "", "", ErrUnsupportedExportFormat
	}
}
