package handlers

import (
	"github.com/ProjAnvil/knot/backend/internal/models"
	"github.com/ProjAnvil/knot/backend/internal/services"
	"github.com/ProjAnvil/knot/backend/pkg/response"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// ExportAPIs exports selected APIs to HTML
func ExportAPIs(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var body struct {
			APIIDs []uint `json:"apiIds"`
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

		groupMap := make(map[uint]string)
		for _, g := range groups {
			groupMap[g.ID] = g.Name
		}

		// Fetch parameters for each API
		var apisWithParams []services.APIWithParams
		for _, api := range apis {
			var allParams []models.Parameter
			if err := db.Where("api_id = ?", api.ID).Order("`order` ASC").Find(&allParams).Error; err != nil {
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

			groupName := groupMap[api.GroupID]
			if groupName == "" {
				groupName = "Ungrouped"
			}

			apisWithParams = append(apisWithParams, services.APIWithParams{
				API:                api,
				GroupName:          groupName,
				RequestParameters:  requestParams,
				ResponseParameters: responseParams,
			})
		}

		// Get locale from cookie or default to "zh"
		locale := c.Cookies("locale", "zh")

		// Generate HTML
		html := services.GenerateHTML(apisWithParams, locale)

		// Set response headers
		c.Set("Content-Type", "text/html; charset=utf-8")
		c.Set("Content-Disposition", "attachment; filename=\"api-docs.html\"")

		return c.SendString(html)
	}
}
