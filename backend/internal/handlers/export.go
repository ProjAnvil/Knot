package handlers

import (
	"errors"
	"fmt"
	"time"

	"github.com/ProjAnvil/knot/backend/internal/services"
	"github.com/ProjAnvil/knot/backend/pkg/response"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// ExportAPIs exports selected APIs in the requested format:
// "html" (default), "json" (Knot JSON v1), "openapi" or "postman".
// The document itself is built by services.BuildExportDocument, the shared
// source also used by the embeddable engine facade, so both surfaces serve
// byte-identical payloads and filenames.
func ExportAPIs(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var body struct {
			APIIDs []uint `json:"apiIds"`
			Format string `json:"format"`
		}

		if err := c.BodyParser(&body); err != nil {
			return response.BadRequest(c, "Invalid request body")
		}

		payload, filename, contentType, err := services.BuildExportDocument(
			db, body.APIIDs, body.Format, c.Cookies("locale", "zh"), time.Now())
		if err != nil {
			switch {
			case errors.Is(err, services.ErrNoAPIIDs):
				return response.BadRequest(c, "No API IDs provided")
			case errors.Is(err, services.ErrUnsupportedExportFormat):
				return response.BadRequest(c, "Unsupported export format")
			default:
				return response.InternalError(c, "Failed to export APIs")
			}
		}

		c.Set("Content-Type", contentType)
		c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))

		return c.Send(payload)
	}
}
