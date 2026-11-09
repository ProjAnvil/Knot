package handlers

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/ProjAnvil/knot/backend/internal/services"
	"github.com/ProjAnvil/knot/backend/pkg/response"
	models "github.com/ProjAnvil/knot/models"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ExportAPIs exports selected APIs in the requested format:
// "html" (default), "json" (Knot JSON v1), "openapi" or "postman"
func ExportAPIs(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var body struct {
			APIIDs []uint `json:"apiIds"`
			Format string `json:"format"`
		}

		if err := c.BodyParser(&body); err != nil {
			return response.BadRequest(c, "Invalid request body")
		}

		if len(body.APIIDs) == 0 {
			return response.BadRequest(c, "No API IDs provided")
		}

		// Fetch APIs with their parameters and group information
		var apis []models.API
		if err := db.Where("id IN ?", body.APIIDs).Find(&apis).Error; err != nil {
			return response.InternalError(c, "Failed to fetch APIs")
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
				return response.InternalError(c, "Failed to fetch groups")
			}
		}

		groupMap := make(map[uint]models.Group)
		for _, g := range groups {
			groupMap[g.ID] = g
		}

		// Fetch parameters for each API
		var apisWithParams []services.APIWithParams
		for _, api := range apis {
			var allParams []models.Parameter
			if err := db.Where("api_id = ?", api.ID).Order(clause.OrderByColumn{Column: clause.Column{Name: "order"}}).Find(&allParams).Error; err != nil {
				return response.InternalError(c, "Failed to fetch parameters")
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

			apisWithParams = append(apisWithParams, services.APIWithParams{
				API:                api,
				GroupName:          groupName,
				GroupOrder:         group.Order,
				RequestParameters:  requestParams,
				ResponseParameters: responseParams,
			})
		}

		now := time.Now()

		switch body.Format {
		case "", "html":
			// Get locale from cookie or default to "zh"
			locale := c.Cookies("locale", "zh")

			// Generate HTML
			html := services.GenerateHTML(apisWithParams, locale)

			// Set response headers
			c.Set("Content-Type", "text/html; charset=utf-8")
			c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", "api-docs.html"))

			return c.SendString(html)
		case "json":
			data, err := json.Marshal(services.BuildKnotDocument(apisWithParams, now))
			if err != nil {
				return response.InternalError(c, "Failed to encode export document")
			}

			filename := fmt.Sprintf("knot-export-%d.json", now.Unix())
			c.Set("Content-Type", "application/json")
			c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))

			return c.Send(data)
		case "openapi":
			data, err := json.Marshal(services.BuildOpenAPIDocument(apisWithParams))
			if err != nil {
				return response.InternalError(c, "Failed to encode export document")
			}

			filename := fmt.Sprintf("openapi-%d.json", now.Unix())
			c.Set("Content-Type", "application/json")
			c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))

			return c.Send(data)
		case "postman":
			data, err := json.Marshal(services.BuildPostmanCollection(apisWithParams, now))
			if err != nil {
				return response.InternalError(c, "Failed to encode export document")
			}

			filename := fmt.Sprintf("postman-%d.json", now.Unix())
			c.Set("Content-Type", "application/json")
			c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))

			return c.Send(data)
		default:
			return response.BadRequest(c, "Unsupported export format")
		}
	}
}
