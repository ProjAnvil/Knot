package handlers

import (
	"io"

	"github.com/ProjAnvil/knot/backend/internal/services"
	"github.com/ProjAnvil/knot/backend/pkg/response"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// ImportDocument handles the upload and import of an API documentation file
// per spec §8.1. The request must be multipart/form-data with a required
// "file" field and an optional "dryRun" form value ("true" previews the
// import without writing anything). The format is auto-detected from the
// file bytes (Knot JSON, Postman Collection v2.0/v2.1, OpenAPI 3.x); the
// parsed plan is applied transactionally and the resulting counts and
// warnings are returned.
func ImportDocument(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		fileHeader, err := c.FormFile("file")
		if err != nil {
			return response.BadRequest(c, "No file provided")
		}

		f, err := fileHeader.Open()
		if err != nil {
			return response.BadRequest(c, "Failed to read uploaded file")
		}
		defer f.Close()

		data, err := io.ReadAll(f)
		if err != nil {
			return response.BadRequest(c, "Failed to read uploaded file")
		}

		format, err := services.DetectImportFormat(data)
		if err != nil {
			return response.Error(c, fiber.StatusUnsupportedMediaType, err.Error())
		}

		var plan *services.ImportPlan
		switch format {
		case services.ImportFormatKnot:
			plan, err = services.ParseKnotJSON(data)
		case services.ImportFormatPostman:
			plan, err = services.ParsePostmanCollection(data)
		case services.ImportFormatOpenAPI:
			plan, err = services.ParseOpenAPIDocument(data)
		default:
			return response.Error(c, fiber.StatusUnsupportedMediaType, "Unsupported file format")
		}
		if err != nil {
			return response.BadRequest(c, err.Error())
		}

		dryRun := c.FormValue("dryRun") == "true"

		result, err := services.ApplyImport(db, format, plan, dryRun)
		if err != nil {
			return response.InternalError(c, "Failed to import document")
		}

		return response.Success(c, result)
	}
}
