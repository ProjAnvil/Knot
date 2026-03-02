package handlers

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/ProjAnvil/knot/backend/internal/models"
	"github.com/ProjAnvil/knot/backend/pkg/response"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// setupAPIsTestApp creates a Fiber app with API test routes
func setupAPIsTestApp(db *gorm.DB) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:               "Knot APIs Test",
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

	// Setup API routes - matching the actual server routes
	api := app.Group("/api")
	apis := api.Group("/apis")
	apis.Get("/:id", GetAPI(db))
	apis.Get("/group/:groupId", GetAPIsByGroup(db))
	apis.Post("/", CreateAPI(db))
	apis.Patch("/:id", UpdateAPI(db))
	apis.Patch("/:id/note", UpdateAPINote(db))
	apis.Post("/orders", UpdateAPIOrders(db))
	apis.Delete("/:id", DeleteAPI(db))
	apis.Put("/:id/parameters", UpdateParameters(db))
	apis.Post("/:id/parameters/from-json", UpdateParametersFromJSON(db))

	return app
}

// TestGetAPI tests the GetAPI handler
func TestGetAPI(t *testing.T) {
	t.Run("givenValidAPIID_whenGetAPI_thenReturnsAPIWithParameters", func(t *testing.T) {
		// Arrange
		db := setupTestDB(t)
		app := setupAPIsTestApp(db)

		group := models.Group{Name: "Test Group", Order: 1}
		db.Create(&group)

		testAPI := models.API{
			GroupID:  group.ID,
			Name:     "Test API",
			Endpoint: "/test",
			Method:   "GET",
			Type:     "HTTP",
		}
		db.Create(&testAPI)

		// Add parameters
		param1 := models.Parameter{
			APIID:     testAPI.ID,
			Name:      "param1",
			Type:      "string",
			ParamType: "request",
			Order:     0,
		}
		param2 := models.Parameter{
			APIID:     testAPI.ID,
			Name:      "param2",
			Type:      "number",
			ParamType: "response",
			Order:     1,
		}
		db.Create(&param1)
		db.Create(&param2)

		// Act
		req := httptest.NewRequest("GET", "/api/apis/1", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)

		// Assert
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var respBody response.Response
		err = json.NewDecoder(resp.Body).Decode(&respBody)
		require.NoError(t, err)

		assert.True(t, respBody.Success)

		apiData := respBody.Data.(map[string]interface{})
		assert.Equal(t, "Test API", apiData["name"])
		assert.Equal(t, "/test", apiData["endpoint"])
		assert.Equal(t, "GET", apiData["method"])

		parameters := apiData["parameters"].([]interface{})
		assert.Len(t, parameters, 2)
	})

	t.Run("givenNonExistentAPIID_whenGetAPI_thenReturnsNotFound", func(t *testing.T) {
		// Arrange
		db := setupTestDB(t)
		app := setupAPIsTestApp(db)

		// Act
		req := httptest.NewRequest("GET", "/api/apis/999", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)

		// Assert
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)

		var respBody response.Response
		err = json.NewDecoder(resp.Body).Decode(&respBody)
		require.NoError(t, err)

		assert.False(t, respBody.Success)
		assert.Contains(t, respBody.Error, "API not found")
	})

	t.Run("givenInvalidAPIID_whenGetAPI_thenReturnsBadRequest", func(t *testing.T) {
		// Arrange
		db := setupTestDB(t)
		app := setupAPIsTestApp(db)

		// Act
		req := httptest.NewRequest("GET", "/api/apis/invalid", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)

		// Assert
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		var respBody response.Response
		err = json.NewDecoder(resp.Body).Decode(&respBody)
		require.NoError(t, err)

		assert.False(t, respBody.Success)
		assert.Contains(t, respBody.Error, "Invalid API ID")
	})
}

// TestGetAPIsByGroup tests the GetAPIsByGroup handler
func TestGetAPIsByGroup(t *testing.T) {
	t.Run("givenValidGroupID_whenGetAPIsByGroup_thenReturnsAPIsInGroup", func(t *testing.T) {
		// Arrange
		db := setupTestDB(t)
		app := setupAPIsTestApp(db)

		group := models.Group{Name: "Test Group", Order: 1}
		db.Create(&group)

		api1 := models.API{
			GroupID:  group.ID,
			Name:     "API 1",
			Endpoint: "/api/one",
			Method:   "GET",
			Type:     "HTTP",
			Order:    1,
		}
		api2 := models.API{
			GroupID:  group.ID,
			Name:     "API 2",
			Endpoint: "/api/two",
			Method:   "POST",
			Type:     "HTTP",
			Order:    2,
		}
		db.Create(&api1)
		db.Create(&api2)

		// Act
		req := httptest.NewRequest("GET", "/api/apis/group/1", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)

		// Assert
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var respBody response.Response
		err = json.NewDecoder(resp.Body).Decode(&respBody)
		require.NoError(t, err)

		assert.True(t, respBody.Success)

		apis := respBody.Data.([]interface{})
		assert.Len(t, apis, 2)
	})

	t.Run("givenEmptyGroup_whenGetAPIsByGroup_thenReturnsEmptyArray", func(t *testing.T) {
		// Arrange
		db := setupTestDB(t)
		app := setupAPIsTestApp(db)

		group := models.Group{Name: "Test Group", Order: 1}
		db.Create(&group)

		// Act
		req := httptest.NewRequest("GET", "/api/apis/group/1", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)

		// Assert
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var respBody response.Response
		err = json.NewDecoder(resp.Body).Decode(&respBody)
		require.NoError(t, err)

		assert.True(t, respBody.Success)

		apis := respBody.Data.([]interface{})
		assert.Empty(t, apis)
	})

	t.Run("givenInvalidGroupID_whenGetAPIsByGroup_thenReturnsBadRequest", func(t *testing.T) {
		// Arrange
		db := setupTestDB(t)
		app := setupAPIsTestApp(db)

		// Act
		req := httptest.NewRequest("GET", "/api/apis/group/invalid", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)

		// Assert
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		var respBody response.Response
		err = json.NewDecoder(resp.Body).Decode(&respBody)
		require.NoError(t, err)

		assert.False(t, respBody.Success)
		assert.Contains(t, respBody.Error, "Invalid group ID")
	})
}

// TestCreateAPI tests the CreateAPI handler
func TestCreateAPI(t *testing.T) {
	t.Run("givenValidAPIData_whenCreateAPI_thenAPIIsCreated", func(t *testing.T) {
		// Arrange
		db := setupTestDB(t)
		app := setupAPIsTestApp(db)

		group := models.Group{Name: "Test Group", Order: 1}
		db.Create(&group)

		note := "This is a test note"
		body := map[string]interface{}{
			"groupId":  1,
			"name":     "New API",
			"endpoint": "/new-api",
			"method":   "POST",
			"type":     "HTTP",
			"note":     &note,
		}
		jsonBody, _ := json.Marshal(body)

		// Act
		req := httptest.NewRequest("POST", "/api/apis", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)

		// Assert
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var respBody response.Response
		err = json.NewDecoder(resp.Body).Decode(&respBody)
		require.NoError(t, err)

		assert.True(t, respBody.Success)

		apiData := respBody.Data.(map[string]interface{})
		assert.Equal(t, "New API", apiData["name"])
		assert.Equal(t, "/new-api", apiData["endpoint"])
		assert.Equal(t, "POST", apiData["method"])
		assert.Equal(t, "HTTP", apiData["type"])
		assert.Equal(t, "This is a test note", apiData["note"])
		assert.Equal(t, float64(1), apiData["order"])
	})

	t.Run("givenMissingRequiredFields_whenCreateAPI_thenReturnsBadRequest", func(t *testing.T) {
		// Arrange
		db := setupTestDB(t)
		app := setupAPIsTestApp(db)

		body := map[string]interface{}{
			"name": "New API",
			// Missing groupId, endpoint, type
		}
		jsonBody, _ := json.Marshal(body)

		// Act
		req := httptest.NewRequest("POST", "/api/apis", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)

		// Assert
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		var respBody response.Response
		err = json.NewDecoder(resp.Body).Decode(&respBody)
		require.NoError(t, err)

		assert.False(t, respBody.Success)
		assert.Contains(t, respBody.Error, "Missing required fields")
	})

	t.Run("givenHTTPTypeWithoutMethod_whenCreateAPI_thenReturnsBadRequest", func(t *testing.T) {
		// Arrange
		db := setupTestDB(t)
		app := setupAPIsTestApp(db)

		group := models.Group{Name: "Test Group", Order: 1}
		db.Create(&group)

		body := map[string]interface{}{
			"groupId":  1,
			"name":     "New API",
			"endpoint": "/new-api",
			"type":     "HTTP",
			// Missing method
		}
		jsonBody, _ := json.Marshal(body)

		// Act
		req := httptest.NewRequest("POST", "/api/apis", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)

		// Assert
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		var respBody response.Response
		err = json.NewDecoder(resp.Body).Decode(&respBody)
		require.NoError(t, err)

		assert.False(t, respBody.Success)
		assert.Contains(t, respBody.Error, "Method is required for HTTP APIs")
	})

	t.Run("givenExistingAPIsInGroup_whenCreateAPI_thenNewAPIHasOrderIncremented", func(t *testing.T) {
		// Arrange
		db := setupTestDB(t)
		app := setupAPIsTestApp(db)

		group := models.Group{Name: "Test Group", Order: 1}
		db.Create(&group)

		existingAPI := models.API{
			GroupID:  group.ID,
			Name:     "Existing API",
			Endpoint: "/existing",
			Method:   "GET",
			Type:     "HTTP",
			Order:    1,
		}
		db.Create(&existingAPI)

		body := map[string]interface{}{
			"groupId":  1,
			"name":     "New API",
			"endpoint": "/new-api",
			"method":   "GET",
			"type":     "HTTP",
		}
		jsonBody, _ := json.Marshal(body)

		// Act
		req := httptest.NewRequest("POST", "/api/apis", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)

		// Assert
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var respBody response.Response
		err = json.NewDecoder(resp.Body).Decode(&respBody)
		require.NoError(t, err)

		apiData := respBody.Data.(map[string]interface{})
		assert.Equal(t, float64(2), apiData["order"])
	})
}

// TestUpdateAPI tests the UpdateAPI handler
func TestUpdateAPI(t *testing.T) {
	t.Run("givenValidAPIIDAndData_whenUpdateAPI_thenAPIIsUpdated", func(t *testing.T) {
		// Arrange
		db := setupTestDB(t)
		app := setupAPIsTestApp(db)

		group := models.Group{Name: "Test Group", Order: 1}
		db.Create(&group)

		testAPI := models.API{
			GroupID:  group.ID,
			Name:     "Original Name",
			Endpoint: "/original",
			Method:   "GET",
			Type:     "HTTP",
		}
		db.Create(&testAPI)

		newName := "Updated Name"
		newEndpoint := "/updated"
		body := map[string]interface{}{
			"name":     &newName,
			"endpoint": &newEndpoint,
		}
		jsonBody, _ := json.Marshal(body)

		// Act
		req := httptest.NewRequest("PATCH", "/api/apis/1", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)

		// Assert
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var respBody response.Response
		err = json.NewDecoder(resp.Body).Decode(&respBody)
		require.NoError(t, err)

		assert.True(t, respBody.Success)

		apiData := respBody.Data.(map[string]interface{})
		assert.Equal(t, "Updated Name", apiData["name"])
		assert.Equal(t, "/updated", apiData["endpoint"])
	})

	t.Run("givenPartialUpdate_whenUpdateAPI_thenOnlyProvidedFieldsAreUpdated", func(t *testing.T) {
		// Arrange
		db := setupTestDB(t)
		app := setupAPIsTestApp(db)

		group := models.Group{Name: "Test Group", Order: 1}
		db.Create(&group)

		testAPI := models.API{
			GroupID:  group.ID,
			Name:     "Original Name",
			Endpoint: "/original",
			Method:   "GET",
			Type:     "HTTP",
		}
		db.Create(&testAPI)

		newName := "Updated Name"
		body := map[string]interface{}{
			"name": &newName,
			// Only updating name
		}
		jsonBody, _ := json.Marshal(body)

		// Act
		req := httptest.NewRequest("PATCH", "/api/apis/1", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)

		// Assert
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var respBody response.Response
		err = json.NewDecoder(resp.Body).Decode(&respBody)
		require.NoError(t, err)

		apiData := respBody.Data.(map[string]interface{})
		assert.Equal(t, "Updated Name", apiData["name"])
		assert.Equal(t, "/original", apiData["endpoint"]) // Unchanged
	})

	t.Run("givenNonExistentAPIID_whenUpdateAPI_thenReturnsNotFound", func(t *testing.T) {
		// Arrange
		db := setupTestDB(t)
		app := setupAPIsTestApp(db)

		newName := "Updated Name"
		body := map[string]interface{}{
			"name": &newName,
		}
		jsonBody, _ := json.Marshal(body)

		// Act
		req := httptest.NewRequest("PATCH", "/api/apis/999", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)

		// Assert
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)

		var respBody response.Response
		err = json.NewDecoder(resp.Body).Decode(&respBody)
		require.NoError(t, err)

		assert.False(t, respBody.Success)
		assert.Contains(t, respBody.Error, "API not found")
	})
}

// TestUpdateAPINote tests the UpdateAPINote handler
func TestUpdateAPINote(t *testing.T) {
	t.Run("givenValidNote_whenUpdateAPINote_thenNoteIsUpdated", func(t *testing.T) {
		// Arrange
		db := setupTestDB(t)
		app := setupAPIsTestApp(db)

		group := models.Group{Name: "Test Group", Order: 1}
		db.Create(&group)

		testAPI := models.API{
			GroupID:  group.ID,
			Name:     "Test API",
			Endpoint: "/test",
			Method:   "GET",
			Type:     "HTTP",
		}
		db.Create(&testAPI)

		note := "This is a test note"
		body := map[string]*string{
			"note": &note,
		}
		jsonBody, _ := json.Marshal(body)

		// Act
		req := httptest.NewRequest("PATCH", "/api/apis/1/note", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)

		// Assert
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var respBody response.Response
		err = json.NewDecoder(resp.Body).Decode(&respBody)
		require.NoError(t, err)

		assert.True(t, respBody.Success)

		apiData := respBody.Data.(map[string]interface{})
		assert.Equal(t, "This is a test note", apiData["note"])
	})

	t.Run("givenNilNote_whenUpdateAPINote_thenNoteIsCleared", func(t *testing.T) {
		// Arrange
		db := setupTestDB(t)
		app := setupAPIsTestApp(db)

		group := models.Group{Name: "Test Group", Order: 1}
		db.Create(&group)

		note := "Existing note"
		testAPI := models.API{
			GroupID:  group.ID,
			Name:     "Test API",
			Endpoint: "/test",
			Method:   "GET",
			Type:     "HTTP",
			Note:     &note,
		}
		db.Create(&testAPI)

		body := map[string]*string{
			"note": nil,
		}
		jsonBody, _ := json.Marshal(body)

		// Act
		req := httptest.NewRequest("PATCH", "/api/apis/1/note", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)

		// Assert
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var respBody response.Response
		err = json.NewDecoder(resp.Body).Decode(&respBody)
		require.NoError(t, err)

		assert.True(t, respBody.Success)

		// Verify note is nil
		var updatedAPI models.API
		db.First(&updatedAPI, 1)
		assert.Nil(t, updatedAPI.Note)
	})
}

// TestUpdateAPIOrders tests the UpdateAPIOrders handler
func TestUpdateAPIOrders(t *testing.T) {
	t.Run("givenValidAPIOrders_whenUpdateAPIOrders_thenOrdersAreUpdated", func(t *testing.T) {
		// Arrange
		db := setupTestDB(t)
		app := setupAPIsTestApp(db)

		group := models.Group{Name: "Test Group", Order: 1}
		db.Create(&group)

		api1 := models.API{
			GroupID:  group.ID,
			Name:     "API 1",
			Endpoint: "/api/1",
			Method:   "GET",
			Type:     "HTTP",
			Order:    1,
		}
		api2 := models.API{
			GroupID:  group.ID,
			Name:     "API 2",
			Endpoint: "/api/2",
			Method:   "GET",
			Type:     "HTTP",
			Order:    2,
		}
		db.Create(&api1)
		db.Create(&api2)

		body := map[string]interface{}{
			"apiOrders": []map[string]interface{}{
				{"id": 1, "order": 2},
				{"id": 2, "order": 1},
			},
		}
		jsonBody, _ := json.Marshal(body)

		// Act
		req := httptest.NewRequest("POST", "/api/apis/orders", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)

		// Assert
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var respBody response.Response
		err = json.NewDecoder(resp.Body).Decode(&respBody)
		require.NoError(t, err)

		assert.True(t, respBody.Success)

		// Verify in database
		var a1, a2 models.API
		db.First(&a1, 1)
		db.First(&a2, 2)
		assert.Equal(t, 2, a1.Order)
		assert.Equal(t, 1, a2.Order)
	})
}

// TestDeleteAPI tests the DeleteAPI handler
func TestDeleteAPI(t *testing.T) {
	t.Run("givenValidAPIID_whenDeleteAPI_thenAPIIsDeleted", func(t *testing.T) {
		// Arrange
		db := setupTestDB(t)
		app := setupAPIsTestApp(db)

		group := models.Group{Name: "Test Group", Order: 1}
		db.Create(&group)

		testAPI := models.API{
			GroupID:  group.ID,
			Name:     "Test API",
			Endpoint: "/test",
			Method:   "GET",
			Type:     "HTTP",
		}
		db.Create(&testAPI)

		// Act
		req := httptest.NewRequest("DELETE", "/api/apis/1", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)

		// Assert
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var respBody response.Response
		err = json.NewDecoder(resp.Body).Decode(&respBody)
		require.NoError(t, err)

		assert.True(t, respBody.Success)

		// Verify deletion in database
		var api models.API
		result := db.First(&api, 1)
		assert.Error(t, result.Error)
		assert.Equal(t, gorm.ErrRecordNotFound, result.Error)
		// Note: Cascade delete behavior depends on database constraints
	})

	t.Run("givenNonExistentAPIID_whenDeleteAPI_thenReturnsNotFound", func(t *testing.T) {
		// Arrange
		db := setupTestDB(t)
		app := setupAPIsTestApp(db)

		// Act
		req := httptest.NewRequest("DELETE", "/api/apis/999", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)

		// Assert
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)

		var respBody response.Response
		err = json.NewDecoder(resp.Body).Decode(&respBody)
		require.NoError(t, err)

		assert.False(t, respBody.Success)
		assert.Contains(t, respBody.Error, "API not found")
	})
}

// TestUpdateParameters tests the UpdateParameters handler
func TestUpdateParameters(t *testing.T) {
	t.Run("givenValidParameters_whenUpdateParameters_thenParametersAreUpdated", func(t *testing.T) {
		// Arrange
		db := setupTestDB(t)
		app := setupAPIsTestApp(db)

		group := models.Group{Name: "Test Group", Order: 1}
		db.Create(&group)

		testAPI := models.API{
			GroupID:  group.ID,
			Name:     "Test API",
			Endpoint: "/test",
			Method:   "GET",
			Type:     "HTTP",
		}
		db.Create(&testAPI)

		// Create existing parameters
		param1 := models.Parameter{
			APIID:     testAPI.ID,
			Name:      "oldParam",
			Type:      "string",
			ParamType: "request",
			Order:     0,
		}
		db.Create(&param1)

		parameters := []map[string]interface{}{
			{
				"name":        "newParam",
				"type":        "string",
				"description": "A new parameter",
				"required":    true,
			},
		}
		body := map[string]interface{}{
			"paramType":  "request",
			"parameters": parameters,
		}
		jsonBody, _ := json.Marshal(body)

		// Act
		req := httptest.NewRequest("PUT", "/api/apis/1/parameters", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)

		// Assert
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var respBody response.Response
		err = json.NewDecoder(resp.Body).Decode(&respBody)
		require.NoError(t, err)

		assert.True(t, respBody.Success)

		// Verify parameters in database
		var params []models.Parameter
		db.Where("api_id = ? AND param_type = ?", 1, "request").Find(&params)
		assert.Len(t, params, 1)
		assert.Equal(t, "newParam", params[0].Name)
	})

	t.Run("givenInvalidParamType_whenUpdateParameters_thenReturnsBadRequest", func(t *testing.T) {
		// Arrange
		db := setupTestDB(t)
		app := setupAPIsTestApp(db)

		group := models.Group{Name: "Test Group", Order: 1}
		db.Create(&group)

		testAPI := models.API{
			GroupID:  group.ID,
			Name:     "Test API",
			Endpoint: "/test",
			Method:   "GET",
			Type:     "HTTP",
		}
		db.Create(&testAPI)

		body := map[string]interface{}{
			"paramType":  "invalid",
			"parameters": []map[string]interface{}{},
		}
		jsonBody, _ := json.Marshal(body)

		// Act
		req := httptest.NewRequest("PUT", "/api/apis/1/parameters", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)

		// Assert
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		var respBody response.Response
		err = json.NewDecoder(resp.Body).Decode(&respBody)
		require.NoError(t, err)

		assert.False(t, respBody.Success)
		assert.Contains(t, respBody.Error, "Invalid paramType")
	})
}

// TestUpdateParametersFromJSON tests the UpdateParametersFromJSON handler
func TestUpdateParametersFromJSON(t *testing.T) {
	t.Run("givenValidJSON_whenUpdateParametersFromJSON_thenParametersAreCreated", func(t *testing.T) {
		// Arrange
		db := setupTestDB(t)
		app := setupAPIsTestApp(db)

		group := models.Group{Name: "Test Group", Order: 1}
		db.Create(&group)

		testAPI := models.API{
			GroupID:  group.ID,
			Name:     "Test API",
			Endpoint: "/test",
			Method:   "GET",
			Type:     "HTTP",
		}
		db.Create(&testAPI)

		jsonData := map[string]interface{}{
			"username": "john_doe",
			"age":      30,
			"active":   true,
		}
		body := map[string]interface{}{
			"paramType": "request",
			"json":      jsonData,
		}
		jsonBody, _ := json.Marshal(body)

		// Act
		req := httptest.NewRequest("POST", "/api/apis/1/parameters/from-json", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)

		// Assert
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var respBody response.Response
		err = json.NewDecoder(resp.Body).Decode(&respBody)
		require.NoError(t, err)

		assert.True(t, respBody.Success)

		// Verify parameters in database
		var params []models.Parameter
		db.Where("api_id = ? AND param_type = ?", 1, "request").Find(&params)
		assert.Len(t, params, 3)

		// Check parameter types
		paramMap := make(map[string]models.Parameter)
		for _, p := range params {
			paramMap[p.Name] = p
		}
		assert.Equal(t, "string", paramMap["username"].Type)
		assert.Equal(t, "number", paramMap["age"].Type)
		assert.Equal(t, "boolean", paramMap["active"].Type)
	})

	t.Run("givenNestedJSON_whenUpdateParametersFromJSON_thenNestedParametersAreCreated", func(t *testing.T) {
		// Arrange
		db := setupTestDB(t)
		app := setupAPIsTestApp(db)

		group := models.Group{Name: "Test Group", Order: 1}
		db.Create(&group)

		testAPI := models.API{
			GroupID:  group.ID,
			Name:     "Test API",
			Endpoint: "/test",
			Method:   "POST",
			Type:     "HTTP",
		}
		db.Create(&testAPI)

		jsonData := map[string]interface{}{
			"user": map[string]interface{}{
				"name": "John",
				"age":  30,
			},
		}
		body := map[string]interface{}{
			"paramType": "request",
			"json":      jsonData,
		}
		jsonBody, _ := json.Marshal(body)

		// Act
		req := httptest.NewRequest("POST", "/api/apis/1/parameters/from-json", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)

		// Assert
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var respBody response.Response
		err = json.NewDecoder(resp.Body).Decode(&respBody)
		require.NoError(t, err)

		assert.True(t, respBody.Success)

		// Verify parent parameter
		var userParam models.Parameter
		db.Where("api_id = ? AND param_type = ? AND name = ?", 1, "request", "user").First(&userParam)
		assert.Equal(t, "object", userParam.Type)

		// Verify child parameters
		var childParams []models.Parameter
		db.Where("parent_id = ?", userParam.ID).Find(&childParams)
		assert.Len(t, childParams, 2)
	})

	t.Run("givenInvalidParamType_whenUpdateParametersFromJSON_thenReturnsBadRequest", func(t *testing.T) {
		// Arrange
		db := setupTestDB(t)
		app := setupAPIsTestApp(db)

		group := models.Group{Name: "Test Group", Order: 1}
		db.Create(&group)

		testAPI := models.API{
			GroupID:  group.ID,
			Name:     "Test API",
			Endpoint: "/test",
			Method:   "GET",
			Type:     "HTTP",
		}
		db.Create(&testAPI)

		body := map[string]interface{}{
			"paramType": "invalid",
			"json":      map[string]interface{}{},
		}
		jsonBody, _ := json.Marshal(body)

		// Act
		req := httptest.NewRequest("POST", "/api/apis/1/parameters/from-json", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)

		// Assert
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		var respBody response.Response
		err = json.NewDecoder(resp.Body).Decode(&respBody)
		require.NoError(t, err)

		assert.False(t, respBody.Success)
		assert.Contains(t, respBody.Error, "Invalid paramType")
	})
}
