package handlers

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ProjAnvil/knot/backend/internal/models"
	"github.com/ProjAnvil/knot/backend/pkg/response"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// setupImportTestApp creates a Fiber app with the import test route. The
// BodyLimit mirrors the production servers' 10 MB import payload cap (the
// export test app keeps its own 4 MB limit and is intentionally unchanged).
func setupImportTestApp(db *gorm.DB) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:               "Knot Import Test",
		DisableStartupMessage: true,
		BodyLimit:             10 * 1024 * 1024,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(response.Response{
				Success: false,
				Error:   err.Error(),
			})
		},
	})

	// Setup import route - matching the actual server routes
	api := app.Group("/api")
	api.Post("/import", ImportDocument(db))

	return app
}

// importResponseData mirrors the ImportResult payload served by the import
// endpoint for both dry runs and real imports.
type importResponseData struct {
	Format             string   `json:"format"`
	GroupsCreated      int      `json:"groupsCreated"`
	GroupsMerged       int      `json:"groupsMerged"`
	APIsImported       int      `json:"apisImported"`
	ParametersImported int      `json:"parametersImported"`
	Warnings           []string `json:"warnings"`
}

// knotImportFixture returns a minimal valid Knot JSON v1 export: one group
// with one API and three request parameters (one of them nested).
func knotImportFixture() []byte {
	return []byte(`{
		"format": "knot",
		"version": 1,
		"exportedAt": "2026-11-10T00:00:00Z",
		"groups": [
			{
				"name": "Imported Group",
				"order": 0,
				"apis": [
					{
						"name": "Create User",
						"endpoint": "/users",
						"method": "post",
						"type": "HTTP",
						"order": 0,
						"parameters": {
							"request": [
								{"name": "username", "type": "string", "required": true},
								{"name": "profile", "type": "object", "children": [
									{"name": "email", "type": "string"}
								]}
							],
							"response": []
						}
					}
				]
			}
		]
	}`)
}

// postImportRequest builds a multipart request with a "file" field (skipped
// when fileBytes is nil) and an optional "dryRun" form value.
func postImportRequest(t *testing.T, fileBytes []byte, dryRun string) *http.Request {
	t.Helper()

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	if fileBytes != nil {
		fw, err := w.CreateFormFile("file", "import.json")
		require.NoError(t, err)
		_, err = fw.Write(fileBytes)
		require.NoError(t, err)
	}
	if dryRun != "" {
		require.NoError(t, w.WriteField("dryRun", dryRun))
	}
	require.NoError(t, w.Close())

	req := httptest.NewRequest("POST", "/api/import", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	return req
}

// TestImportDocument tests the ImportDocument handler
func TestImportDocument(t *testing.T) {
	t.Run("givenKnotFileAndDryRun_whenImport_thenReturnsCountsWithoutWriting", func(t *testing.T) {
		// Arrange
		db := setupTestDB(t)
		app := setupImportTestApp(db)

		// Act
		resp, err := app.Test(postImportRequest(t, knotImportFixture(), "true"))
		require.NoError(t, err)

		// Assert
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var body struct {
			Success bool               `json:"success"`
			Data    importResponseData `json:"data"`
		}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))

		assert.True(t, body.Success)
		assert.Equal(t, "knot-json", body.Data.Format)
		assert.Equal(t, 1, body.Data.GroupsCreated)
		assert.Equal(t, 0, body.Data.GroupsMerged)
		assert.Equal(t, 1, body.Data.APIsImported)
		assert.Equal(t, 3, body.Data.ParametersImported)

		// A dry run must not write any rows.
		var groups, apis, params int64
		db.Model(&models.Group{}).Count(&groups)
		db.Model(&models.API{}).Count(&apis)
		db.Model(&models.Parameter{}).Count(&params)
		assert.Equal(t, int64(0), groups)
		assert.Equal(t, int64(0), apis)
		assert.Equal(t, int64(0), params)
	})

	t.Run("givenUpperCaseDryRun_whenImport_thenNoWrites", func(t *testing.T) {
		// Arrange - a non-lowercase truthy dryRun value must still be treated
		// as a preview request, not fall through to a real import.
		db := setupTestDB(t)
		app := setupImportTestApp(db)

		// Act
		resp, err := app.Test(postImportRequest(t, knotImportFixture(), "True"))
		require.NoError(t, err)

		// Assert
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var body struct {
			Success bool               `json:"success"`
			Data    importResponseData `json:"data"`
		}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))

		assert.True(t, body.Success)
		assert.Equal(t, 1, body.Data.GroupsCreated)
		assert.Equal(t, 1, body.Data.APIsImported)
		assert.Equal(t, 3, body.Data.ParametersImported)

		// The upper-case spelling must not write any rows.
		var groups, apis, params int64
		db.Model(&models.Group{}).Count(&groups)
		db.Model(&models.API{}).Count(&apis)
		db.Model(&models.Parameter{}).Count(&params)
		assert.Equal(t, int64(0), groups)
		assert.Equal(t, int64(0), apis)
		assert.Equal(t, int64(0), params)
	})

	t.Run("givenKnotFile_whenImport_thenPersistsRows", func(t *testing.T) {
		// Arrange
		db := setupTestDB(t)
		app := setupImportTestApp(db)

		// Act
		resp, err := app.Test(postImportRequest(t, knotImportFixture(), ""))
		require.NoError(t, err)

		// Assert
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var body struct {
			Success bool               `json:"success"`
			Data    importResponseData `json:"data"`
		}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))

		assert.True(t, body.Success)
		assert.Equal(t, "knot-json", body.Data.Format)
		assert.Equal(t, 1, body.Data.GroupsCreated)
		assert.Equal(t, 1, body.Data.APIsImported)
		assert.Equal(t, 3, body.Data.ParametersImported)

		var group models.Group
		require.NoError(t, db.Where("name = ?", "Imported Group").First(&group).Error)

		var api models.API
		require.NoError(t, db.Where("name = ?", "Create User").First(&api).Error)
		assert.Equal(t, group.ID, api.GroupID)
		assert.Equal(t, "/users", api.Endpoint)
		// Methods are upper-cased by the Knot parser.
		assert.Equal(t, "POST", api.Method)

		var paramCount int64
		db.Model(&models.Parameter{}).Where("api_id = ?", api.ID).Count(&paramCount)
		assert.Equal(t, int64(3), paramCount)
	})

	t.Run("givenMissingFileField_whenImport_thenReturnsBadRequest", func(t *testing.T) {
		// Arrange
		db := setupTestDB(t)
		app := setupImportTestApp(db)

		// Act - multipart body with only the dryRun field, no file
		resp, err := app.Test(postImportRequest(t, nil, "true"))
		require.NoError(t, err)

		// Assert
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		var respBody response.Response
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&respBody))

		assert.False(t, respBody.Success)
		assert.Contains(t, respBody.Error, "No file provided")
	})

	t.Run("givenGarbageBytes_whenImport_thenReturnsUnsupportedMediaType", func(t *testing.T) {
		// Arrange
		db := setupTestDB(t)
		app := setupImportTestApp(db)

		// Act
		resp, err := app.Test(postImportRequest(t, []byte("\x00\x01not json at all"), ""))
		require.NoError(t, err)

		// Assert
		assert.Equal(t, fiber.StatusUnsupportedMediaType, resp.StatusCode)

		var respBody response.Response
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&respBody))

		assert.False(t, respBody.Success)
		assert.NotEmpty(t, respBody.Error)
	})

	t.Run("givenSwagger20Document_whenImport_thenReturns415WithOpenAPI3Hint", func(t *testing.T) {
		// Arrange
		db := setupTestDB(t)
		app := setupImportTestApp(db)

		swaggerDoc := []byte(`{"swagger": "2.0", "info": {"title": "Legacy API", "version": "1.0"}, "paths": {}}`)

		// Act
		resp, err := app.Test(postImportRequest(t, swaggerDoc, ""))
		require.NoError(t, err)

		// Assert
		assert.Equal(t, fiber.StatusUnsupportedMediaType, resp.StatusCode)

		var respBody response.Response
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&respBody))

		assert.False(t, respBody.Success)
		assert.Contains(t, respBody.Error, "OpenAPI 3")
	})

	t.Run("givenOversizedFile_whenImport_thenReturnsRequestEntityTooLarge", func(t *testing.T) {
		// Arrange
		db := setupTestDB(t)
		app := setupImportTestApp(db)

		// 11 MB of payload, past the app's 10 MB BodyLimit.
		oversized := bytes.Repeat([]byte("a"), 11*1024*1024)

		// Act - disable the test timeout, the payload is large
		resp, err := app.Test(postImportRequest(t, oversized, ""), -1)

		// Assert
		// Fiber v2's app.Test surfaces fasthttp's body-limit rejection as an
		// error instead of the 413 response the server writes alongside it
		// (only fasthttp.ErrGetOnly is whitelisted through). For real clients
		// this exact rejection is what produces the 413, so accept either.
		if err != nil {
			assert.Contains(t, err.Error(), "body size exceeds the given limit")
			return
		}
		assert.Equal(t, fiber.StatusRequestEntityTooLarge, resp.StatusCode)
	})
}

// postmanImportFixture returns a minimal Postman v2.1 collection that
// DetectImportFormat classifies as postman: one group, one GET API.
func postmanImportFixture() []byte {
	return []byte(`{
		"info": {"name": "Collection", "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"},
		"item": [{"name": "List users", "request": {"method": "GET", "url": "/users"}}]
	}`)
}

// openAPIImportFixture returns a minimal OpenAPI 3.1 document that
// DetectImportFormat classifies as openapi: one tagged operation.
func openAPIImportFixture() []byte {
	return []byte(`{
		"openapi": "3.1.0",
		"info": {"title": "API", "version": "1.0.0"},
		"paths": {"/users": {"get": {"summary": "List users", "tags": ["Users"]}}}
	}`)
}

// TestImportDocument_FormatDispatchAndErrors covers the remaining dispatch
// and failure paths of the import handler: the postman and openapi parse
// branches, a detectable file whose parse fails (400) and an apply failure
// surfaced as 500.
func TestImportDocument_FormatDispatchAndErrors(t *testing.T) {
	t.Run("givenPostmanFile_whenImport_thenImportsCollection", func(t *testing.T) {
		// Arrange
		db := setupTestDB(t)
		app := setupImportTestApp(db)

		// Act - real import (no dryRun)
		resp, err := app.Test(postImportRequest(t, postmanImportFixture(), ""))
		require.NoError(t, err)

		// Assert
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var body struct {
			Success bool               `json:"success"`
			Data    importResponseData `json:"data"`
		}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		assert.True(t, body.Success)
		assert.Equal(t, "postman", body.Data.Format)
		assert.Equal(t, 1, body.Data.GroupsCreated)
		assert.Equal(t, 1, body.Data.APIsImported)

		// Assert - the collection is persisted with its API
		var groups []models.Group
		require.NoError(t, db.Find(&groups).Error)
		require.Len(t, groups, 1)
		var apis []models.API
		require.NoError(t, db.Where("group_id = ?", groups[0].ID).Find(&apis).Error)
		if assert.Len(t, apis, 1) {
			assert.Equal(t, "List users", apis[0].Name)
			assert.Equal(t, "/users", apis[0].Endpoint)
		}
	})

	t.Run("givenOpenAPIFileAndDryRun_whenImport_thenPreviewWithoutWriting", func(t *testing.T) {
		// Arrange
		db := setupTestDB(t)
		app := setupImportTestApp(db)

		// Act
		resp, err := app.Test(postImportRequest(t, openAPIImportFixture(), "true"))
		require.NoError(t, err)

		// Assert - preview counts are returned
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var body struct {
			Success bool               `json:"success"`
			Data    importResponseData `json:"data"`
		}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		assert.True(t, body.Success)
		assert.Equal(t, "openapi", body.Data.Format)
		assert.Equal(t, 1, body.Data.GroupsCreated)
		assert.Equal(t, 1, body.Data.APIsImported)

		// Assert - nothing was written
		var count int64
		db.Model(&models.API{}).Count(&count)
		assert.Equal(t, int64(0), count)
	})

	t.Run("givenKnotFileWithUnknownVersion_whenImport_thenReturnsBadRequest", func(t *testing.T) {
		// Arrange - the format marker detects as Knot but the version check
		// rejects the document during parsing
		db := setupTestDB(t)
		app := setupImportTestApp(db)

		// Act
		resp, err := app.Test(postImportRequest(t, []byte(`{"format": "knot", "version": 2, "groups": []}`), ""))
		require.NoError(t, err)

		// Assert
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		var respBody response.Response
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&respBody))
		assert.False(t, respBody.Success)
		assert.Contains(t, respBody.Error, "version")
	})

	t.Run("givenApplyFailure_whenImport_thenReturnsInternalServerError", func(t *testing.T) {
		// Arrange - a valid Knot upload against a database whose groups
		// table is missing, so the transactional apply fails
		db := setupTestDB(t)
		require.NoError(t, db.Migrator().DropTable(&models.Group{}))
		app := setupImportTestApp(db)

		// Act
		resp, err := app.Test(postImportRequest(t, knotImportFixture(), ""))
		require.NoError(t, err)

		// Assert
		assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)

		var respBody response.Response
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&respBody))
		assert.False(t, respBody.Success)
		assert.Contains(t, respBody.Error, "Failed to import document")
	})
}
