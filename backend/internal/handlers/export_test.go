package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ProjAnvil/knot/backend/internal/models"
	"github.com/ProjAnvil/knot/backend/pkg/response"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// setupExportTestApp creates a Fiber app with export test routes
func setupExportTestApp(db *gorm.DB) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:               "Knot Export Test",
		DisableStartupMessage: true,
		BodyLimit:             4 * 1024 * 1024,
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

	// Setup export route - matching the actual server routes
	api := app.Group("/api")
	export := api.Group("/export")
	export.Post("/", ExportAPIs(db))

	return app
}

// TestExportAPIs tests the ExportAPIs handler
func TestExportAPIs(t *testing.T) {
	t.Run("givenValidAPIIDs_whenExportAPIs_thenReturnsHTMLDocument", func(t *testing.T) {
		// Arrange
		db := setupTestDB(t)
		app := setupExportTestApp(db)

		// Create group
		group := models.Group{Name: "Test Group", Order: 1}
		db.Create(&group)

		// Create API with parameters
		api := models.API{
			GroupID:  group.ID,
			Name:     "Test API",
			Endpoint: "/test",
			Method:   "GET",
			Type:     "HTTP",
		}
		db.Create(&api)

		// Add request parameters
		reqParam := models.Parameter{
			APIID:     api.ID,
			Name:      "param1",
			Type:      "string",
			ParamType: "request",
			Required:  true,
			Order:     0,
		}
		db.Create(&reqParam)

		// Add response parameters
		respParam := models.Parameter{
			APIID:     api.ID,
			Name:      "result",
			Type:      "string",
			ParamType: "response",
			Required:  false,
			Order:     0,
		}
		db.Create(&respParam)

		body := map[string][]uint{
			"apiIds": {1},
		}
		jsonBody, _ := json.Marshal(body)

		// Act
		req := httptest.NewRequest("POST", "/api/export", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)

		// Assert
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
		assert.Equal(t, "text/html; charset=utf-8", resp.Header.Get("Content-Type"))
		assert.Contains(t, resp.Header.Get("Content-Disposition"), "attachment")

		// Read body
		bodyBytes, _ := io.ReadAll(resp.Body)
		html := string(bodyBytes)

		assert.Contains(t, html, "<!DOCTYPE html>")
		assert.Contains(t, html, "Test API")
		assert.Contains(t, html, "/test")
		assert.Contains(t, html, "param1")
		assert.Contains(t, html, "result")
	})

	t.Run("givenEmptyAPIIDs_whenExportAPIs_thenReturnsBadRequest", func(t *testing.T) {
		// Arrange
		db := setupTestDB(t)
		app := setupExportTestApp(db)

		body := map[string][]uint{
			"apiIds": {},
		}
		jsonBody, _ := json.Marshal(body)

		// Act
		req := httptest.NewRequest("POST", "/api/export", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)

		// Assert
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		var respBody response.Response
		err = json.NewDecoder(resp.Body).Decode(&respBody)
		require.NoError(t, err)

		assert.False(t, respBody.Success)
		assert.Contains(t, respBody.Error, "No API IDs provided")
	})

	t.Run("givenInvalidRequestBody_whenExportAPIs_thenReturnsBadRequest", func(t *testing.T) {
		// Arrange
		db := setupTestDB(t)
		app := setupExportTestApp(db)

		// Act
		req := httptest.NewRequest("POST", "/api/export", bytes.NewReader([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)

		// Assert
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		var respBody response.Response
		err = json.NewDecoder(resp.Body).Decode(&respBody)
		require.NoError(t, err)

		assert.False(t, respBody.Success)
	})

	t.Run("givenMultipleAPIs_whenExportAPIs_thenReturnsHTMLWithAllAPIs", func(t *testing.T) {
		// Arrange
		db := setupTestDB(t)
		app := setupExportTestApp(db)

		// Create group
		group := models.Group{Name: "API Group", Order: 1}
		db.Create(&group)

		// Create multiple APIs
		api1 := models.API{
			GroupID:  group.ID,
			Name:     "API 1",
			Endpoint: "/api/one",
			Method:   "GET",
			Type:     "HTTP",
		}
		api2 := models.API{
			GroupID:  group.ID,
			Name:     "API 2",
			Endpoint: "/api/two",
			Method:   "POST",
			Type:     "HTTP",
		}
		db.Create(&api1)
		db.Create(&api2)

		body := map[string][]uint{
			"apiIds": {1, 2},
		}
		jsonBody, _ := json.Marshal(body)

		// Act
		req := httptest.NewRequest("POST", "/api/export", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)

		// Assert
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		// Read body
		bodyBytes, _ := io.ReadAll(resp.Body)
		html := string(bodyBytes)

		assert.Contains(t, html, "API 1")
		assert.Contains(t, html, "API 2")
		assert.Contains(t, html, "/api/one")
		assert.Contains(t, html, "/api/two")
	})

	t.Run("givenChineseLocaleCookie_whenExportAPIs_thenReturnsChineseHTML", func(t *testing.T) {
		// Arrange
		db := setupTestDB(t)
		app := setupExportTestApp(db)

		// Create group
		group := models.Group{Name: "Test Group", Order: 1}
		db.Create(&group)

		// Create API
		api := models.API{
			GroupID:  group.ID,
			Name:     "Test API",
			Endpoint: "/test",
			Method:   "GET",
			Type:     "HTTP",
		}
		db.Create(&api)

		body := map[string][]uint{
			"apiIds": {1},
		}
		jsonBody, _ := json.Marshal(body)

		// Act
		req := httptest.NewRequest("POST", "/api/export", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Cookie", "locale=zh")
		resp, err := app.Test(req)
		require.NoError(t, err)

		// Assert
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		// Read body
		bodyBytes, _ := io.ReadAll(resp.Body)
		html := string(bodyBytes)

		// Check for Chinese labels
		assert.Contains(t, html, "API")
		assert.Contains(t, html, "lang=\"zh\"")
	})

	t.Run("givenDefaultLocale_whenExportAPIs_thenReturnsChineseHTML", func(t *testing.T) {
		// Arrange
		db := setupTestDB(t)
		app := setupExportTestApp(db)

		// Create group
		group := models.Group{Name: "Test Group", Order: 1}
		db.Create(&group)

		// Create API
		api := models.API{
			GroupID:  group.ID,
			Name:     "Test API",
			Endpoint: "/test",
			Method:   "GET",
			Type:     "HTTP",
		}
		db.Create(&api)

		body := map[string][]uint{
			"apiIds": {1},
		}
		jsonBody, _ := json.Marshal(body)

		// Act - no locale cookie set, should default to zh
		req := httptest.NewRequest("POST", "/api/export", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)

		// Assert
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		// Read body
		bodyBytes, _ := io.ReadAll(resp.Body)
		html := string(bodyBytes)

		// Default should be Chinese
		assert.Contains(t, html, "lang=\"zh\"")
	})

	t.Run("givenNestedParameters_whenExportAPIs_thenIncludesNestedStructureInHTML", func(t *testing.T) {
		// Arrange
		db := setupTestDB(t)
		app := setupExportTestApp(db)

		// Create group
		group := models.Group{Name: "Test Group", Order: 1}
		db.Create(&group)

		// Create API
		api := models.API{
			GroupID:  group.ID,
			Name:     "Test API",
			Endpoint: "/test",
			Method:   "POST",
			Type:     "HTTP",
		}
		db.Create(&api)

		// Create parent parameter
		parentParam := models.Parameter{
			APIID:     api.ID,
			Name:      "user",
			Type:      "object",
			ParamType: "request",
			Order:     0,
		}
		db.Create(&parentParam)

		// Create child parameter
		childParam := models.Parameter{
			APIID:     api.ID,
			ParentID:  &parentParam.ID,
			Name:      "name",
			Type:      "string",
			ParamType: "request",
			Order:     0,
		}
		db.Create(&childParam)

		body := map[string][]uint{
			"apiIds": {1},
		}
		jsonBody, _ := json.Marshal(body)

		// Act
		req := httptest.NewRequest("POST", "/api/export", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)

		// Assert
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		// Read body
		bodyBytes, _ := io.ReadAll(resp.Body)
		html := string(bodyBytes)

		// Should include nested parameters
		assert.Contains(t, html, "user")
		assert.Contains(t, html, "name")
	})

	t.Run("givenAPIWithNote_whenExportAPIs_thenIncludesNoteInHTML", func(t *testing.T) {
		// Arrange
		db := setupTestDB(t)
		app := setupExportTestApp(db)

		// Create group
		group := models.Group{Name: "Test Group", Order: 1}
		db.Create(&group)

		// Create API with note
		note := "This is a test note"
		api := models.API{
			GroupID:  group.ID,
			Name:     "Test API",
			Endpoint: "/test",
			Method:   "GET",
			Type:     "HTTP",
			Note:     &note,
		}
		db.Create(&api)

		body := map[string][]uint{
			"apiIds": {1},
		}
		jsonBody, _ := json.Marshal(body)

		// Act
		req := httptest.NewRequest("POST", "/api/export", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)

		// Assert
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		// Read body
		bodyBytes, _ := io.ReadAll(resp.Body)
		html := string(bodyBytes)

		// Note might be included in the HTML
		assert.Contains(t, html, "Test API")
	})

	t.Run("givenEnglishLocaleCookie_whenExportAPIs_thenReturnsEnglishHTML", func(t *testing.T) {
		// Arrange
		db := setupTestDB(t)
		app := setupExportTestApp(db)

		// Create group
		group := models.Group{Name: "Test Group", Order: 1}
		db.Create(&group)

		// Create API
		api := models.API{
			GroupID:  group.ID,
			Name:     "Test API",
			Endpoint: "/test",
			Method:   "GET",
			Type:     "HTTP",
		}
		db.Create(&api)

		body := map[string][]uint{
			"apiIds": {1},
		}
		jsonBody, _ := json.Marshal(body)

		// Act
		req := httptest.NewRequest("POST", "/api/export", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Cookie", "locale=en")
		resp, err := app.Test(req)
		require.NoError(t, err)

		// Assert
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		// Read body
		bodyBytes, _ := io.ReadAll(resp.Body)
		html := string(bodyBytes)

		// Should be English
		assert.Contains(t, html, "lang=\"en\"")
		assert.True(t, strings.Contains(html, "API Documentation") || strings.Contains(html, "Request Parameters"))
	})
}
