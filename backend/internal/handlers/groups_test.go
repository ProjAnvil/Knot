package handlers

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/ProjAnvil/knot/backend/internal/models"
	"github.com/ProjAnvil/knot/backend/pkg/response"
	"github.com/glebarez/sqlite"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&models.Group{}, &models.API{}, &models.Parameter{})
	require.NoError(t, err)

	return db
}

// setupTestApp creates a Fiber app with test routes
func setupTestApp(db *gorm.DB) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:               "Knot Test",
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

	// Setup routes - matching the actual server routes
	api := app.Group("/api")
	groups := api.Group("/groups")
	groups.Get("/", GetGroups(db))
	groups.Get("/with-apis", GetGroupsWithAPIs(db))
	groups.Post("/orders", UpdateGroupOrders(db))
	groups.Post("/", CreateGroup(db))
	groups.Patch("/:id", UpdateGroup(db))
	groups.Delete("/:id", DeleteGroup(db))

	return app
}

// TestGetGroups tests the GetGroups handler
func TestGetGroups(t *testing.T) {
	t.Run("givenEmptyDatabase_whenGetGroups_thenReturnsEmptyArray", func(t *testing.T) {
		// Arrange
		db := setupTestDB(t)
		app := setupTestApp(db)

		// Act
		req := httptest.NewRequest("GET", "/api/groups", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)

		// Assert
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var respBody response.Response
		err = json.NewDecoder(resp.Body).Decode(&respBody)
		require.NoError(t, err)

		assert.True(t, respBody.Success)
		assert.NotNil(t, respBody.Data)

		groups, ok := respBody.Data.([]interface{})
		assert.True(t, ok, "Data should be an array")
		assert.Empty(t, groups, "Should return empty array when no groups exist")
	})

	t.Run("givenMultipleGroups_whenGetGroups_thenReturnsGroupsOrderedByOrderField", func(t *testing.T) {
		// Arrange
		db := setupTestDB(t)
		app := setupTestApp(db)

		// Create groups with different orders
		group1 := models.Group{Name: "Group C", Order: 3}
		group2 := models.Group{Name: "Group A", Order: 1}
		group3 := models.Group{Name: "Group B", Order: 2}
		db.Create(&group1)
		db.Create(&group2)
		db.Create(&group3)

		// Act
		req := httptest.NewRequest("GET", "/api/groups", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)

		// Assert
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var respBody response.Response
		err = json.NewDecoder(resp.Body).Decode(&respBody)
		require.NoError(t, err)

		assert.True(t, respBody.Success)

		groups := respBody.Data.([]interface{})
		assert.Len(t, groups, 3)

		// Verify order - groups should be ordered by 'order' field
		firstGroup := groups[0].(map[string]interface{})
		secondGroup := groups[1].(map[string]interface{})
		thirdGroup := groups[2].(map[string]interface{})

		assert.Equal(t, "Group A", firstGroup["name"])
		assert.Equal(t, "Group B", secondGroup["name"])
		assert.Equal(t, "Group C", thirdGroup["name"])
	})
}

// TestGetGroupsWithAPIs tests the GetGroupsWithAPIs handler
func TestGetGroupsWithAPIs(t *testing.T) {
	t.Run("givenGroupsWithAPIs_whenGetGroupsWithAPIs_thenReturnsGroupsWithNestedAPIs", func(t *testing.T) {
		// Arrange
		db := setupTestDB(t)
		app := setupTestApp(db)

		// Create a group with APIs
		testGroup := models.Group{Name: "Test Group", Order: 1}
		db.Create(&testGroup)

		api1 := models.API{
			GroupID:  testGroup.ID,
			Name:     "API 1",
			Endpoint: "/api/one",
			Method:   "GET",
			Type:     "HTTP",
			Order:    1,
		}
		api2 := models.API{
			GroupID:  testGroup.ID,
			Name:     "API 2",
			Endpoint: "/api/two",
			Method:   "POST",
			Type:     "HTTP",
			Order:    2,
		}
		db.Create(&api1)
		db.Create(&api2)

		// Act
		req := httptest.NewRequest("GET", "/api/groups/with-apis", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)

		// Assert
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var respBody response.Response
		err = json.NewDecoder(resp.Body).Decode(&respBody)
		require.NoError(t, err)

		assert.True(t, respBody.Success)

		groups := respBody.Data.([]interface{})
		assert.Len(t, groups, 1)

		groupData := groups[0].(map[string]interface{})
		assert.Equal(t, "Test Group", groupData["name"])

		apis := groupData["apis"].([]interface{})
		assert.Len(t, apis, 2)
	})

	t.Run("givenEmptyDatabase_whenGetGroupsWithAPIs_thenReturnsEmptyArray", func(t *testing.T) {
		// Arrange
		db := setupTestDB(t)
		app := setupTestApp(db)

		// Act
		req := httptest.NewRequest("GET", "/api/groups/with-apis", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)

		// Assert
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var respBody response.Response
		err = json.NewDecoder(resp.Body).Decode(&respBody)
		require.NoError(t, err)

		assert.True(t, respBody.Success)
		groups := respBody.Data.([]interface{})
		assert.Empty(t, groups)
	})
}

// TestCreateGroup tests the CreateGroup handler
func TestCreateGroup(t *testing.T) {
	t.Run("givenValidGroupName_whenCreateGroup_thenGroupIsCreated", func(t *testing.T) {
		// Arrange
		db := setupTestDB(t)
		app := setupTestApp(db)

		body := map[string]string{"name": "New Group"}
		jsonBody, _ := json.Marshal(body)

		// Act
		req := httptest.NewRequest("POST", "/api/groups", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)

		// Assert
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var respBody response.Response
		err = json.NewDecoder(resp.Body).Decode(&respBody)
		require.NoError(t, err)

		assert.True(t, respBody.Success)

		groupData := respBody.Data.(map[string]interface{})
		assert.Equal(t, "New Group", groupData["name"])

		// Note: The response does not include 'order' field due to custom MarshalJSON in Group model
		// Verify in database instead
		var group models.Group
		db.First(&group, 1)
		assert.Equal(t, "New Group", group.Name)
		assert.Equal(t, 1, group.Order)
	})

	t.Run("givenEmptyGroupName_whenCreateGroup_thenReturnsBadRequest", func(t *testing.T) {
		// Arrange
		db := setupTestDB(t)
		app := setupTestApp(db)

		body := map[string]string{"name": ""}
		jsonBody, _ := json.Marshal(body)

		// Act
		req := httptest.NewRequest("POST", "/api/groups", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)

		// Assert
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		var respBody response.Response
		err = json.NewDecoder(resp.Body).Decode(&respBody)
		require.NoError(t, err)

		assert.False(t, respBody.Success)
		assert.Contains(t, respBody.Error, "Group name is required")
	})

	t.Run("givenInvalidRequestBody_whenCreateGroup_thenReturnsBadRequest", func(t *testing.T) {
		// Arrange
		db := setupTestDB(t)
		app := setupTestApp(db)

		// Act
		req := httptest.NewRequest("POST", "/api/groups", bytes.NewReader([]byte("invalid json")))
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

	t.Run("givenExistingGroup_whenCreateGroup_thenNewGroupHasOrderIncremented", func(t *testing.T) {
		// Arrange
		db := setupTestDB(t)
		app := setupTestApp(db)

		// Create first group
		existingGroup := models.Group{Name: "Existing Group", Order: 1}
		db.Create(&existingGroup)

		body := map[string]string{"name": "New Group"}
		jsonBody, _ := json.Marshal(body)

		// Act
		req := httptest.NewRequest("POST", "/api/groups", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)

		// Assert
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var respBody response.Response
		err = json.NewDecoder(resp.Body).Decode(&respBody)
		require.NoError(t, err)

		// Verify order was incremented in database
		var groups []models.Group
		db.Find(&groups)
		assert.Len(t, groups, 2)

		// Find the new group
		var newGroup models.Group
		db.Where("name = ?", "New Group").First(&newGroup)
		assert.Equal(t, 2, newGroup.Order)
	})
}

// TestUpdateGroup tests the UpdateGroup handler
func TestUpdateGroup(t *testing.T) {
	t.Run("givenValidGroupIDAndName_whenUpdateGroup_thenGroupIsUpdated", func(t *testing.T) {
		// Arrange
		db := setupTestDB(t)
		app := setupTestApp(db)

		group := models.Group{Name: "Original Name", Order: 1}
		db.Create(&group)

		body := map[string]string{"name": "Updated Name"}
		jsonBody, _ := json.Marshal(body)

		// Act
		req := httptest.NewRequest("PATCH", "/api/groups/1", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)

		// Assert
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var respBody response.Response
		err = json.NewDecoder(resp.Body).Decode(&respBody)
		require.NoError(t, err)

		assert.True(t, respBody.Success)

		groupData := respBody.Data.(map[string]interface{})
		assert.Equal(t, "Updated Name", groupData["name"])

		// Verify in database
		var updatedGroup models.Group
		db.First(&updatedGroup, 1)
		assert.Equal(t, "Updated Name", updatedGroup.Name)
	})

	t.Run("givenNonExistentGroupID_whenUpdateGroup_thenReturnsNotFound", func(t *testing.T) {
		// Arrange
		db := setupTestDB(t)
		app := setupTestApp(db)

		body := map[string]string{"name": "Updated Name"}
		jsonBody, _ := json.Marshal(body)

		// Act
		req := httptest.NewRequest("PATCH", "/api/groups/999", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)

		// Assert
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)

		var respBody response.Response
		err = json.NewDecoder(resp.Body).Decode(&respBody)
		require.NoError(t, err)

		assert.False(t, respBody.Success)
		assert.Contains(t, respBody.Error, "Group not found")
	})

	t.Run("givenInvalidGroupID_whenUpdateGroup_thenReturnsBadRequest", func(t *testing.T) {
		// Arrange
		db := setupTestDB(t)
		app := setupTestApp(db)

		body := map[string]string{"name": "Updated Name"}
		jsonBody, _ := json.Marshal(body)

		// Act
		req := httptest.NewRequest("PATCH", "/api/groups/invalid", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
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

	t.Run("givenEmptyGroupName_whenUpdateGroup_thenReturnsBadRequest", func(t *testing.T) {
		// Arrange
		db := setupTestDB(t)
		app := setupTestApp(db)

		group := models.Group{Name: "Original Name", Order: 1}
		db.Create(&group)

		body := map[string]string{"name": ""}
		jsonBody, _ := json.Marshal(body)

		// Act
		req := httptest.NewRequest("PATCH", "/api/groups/1", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)

		// Assert
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		var respBody response.Response
		err = json.NewDecoder(resp.Body).Decode(&respBody)
		require.NoError(t, err)

		assert.False(t, respBody.Success)
		assert.Contains(t, respBody.Error, "Group name is required")
	})
}

// TestUpdateGroupOrders tests the UpdateGroupOrders handler
func TestUpdateGroupOrders(t *testing.T) {
	t.Run("givenValidGroupOrders_whenUpdateGroupOrders_thenOrdersAreUpdated", func(t *testing.T) {
		// Arrange
		db := setupTestDB(t)
		app := setupTestApp(db)

		group1 := models.Group{Name: "Group 1", Order: 1}
		group2 := models.Group{Name: "Group 2", Order: 2}
		db.Create(&group1)
		db.Create(&group2)

		body := map[string]interface{}{
			"groupOrders": []map[string]interface{}{
				{"id": 1, "order": 2},
				{"id": 2, "order": 1},
			},
		}
		jsonBody, _ := json.Marshal(body)

		// Act
		req := httptest.NewRequest("POST", "/api/groups/orders", bytes.NewReader(jsonBody))
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
		var g1, g2 models.Group
		db.First(&g1, 1)
		db.First(&g2, 2)
		assert.Equal(t, 2, g1.Order)
		assert.Equal(t, 1, g2.Order)
	})

	t.Run("givenEmptyGroupOrders_whenUpdateGroupOrders_thenReturnsSuccess", func(t *testing.T) {
		// Arrange
		db := setupTestDB(t)
		app := setupTestApp(db)

		body := map[string]interface{}{
			"groupOrders": []map[string]interface{}{},
		}
		jsonBody, _ := json.Marshal(body)

		// Act
		req := httptest.NewRequest("POST", "/api/groups/orders", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)

		// Assert
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var respBody response.Response
		err = json.NewDecoder(resp.Body).Decode(&respBody)
		require.NoError(t, err)

		assert.True(t, respBody.Success)
	})

	t.Run("givenInvalidRequestBody_whenUpdateGroupOrders_thenReturnsBadRequest", func(t *testing.T) {
		// Arrange
		db := setupTestDB(t)
		app := setupTestApp(db)

		// Act
		req := httptest.NewRequest("POST", "/api/groups/orders", bytes.NewReader([]byte("invalid")))
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
}

// TestDeleteGroup tests the DeleteGroup handler
func TestDeleteGroup(t *testing.T) {
	t.Run("givenValidGroupID_whenDeleteGroup_thenGroupIsDeleted", func(t *testing.T) {
		// Arrange
		db := setupTestDB(t)
		app := setupTestApp(db)

		testGroup := models.Group{Name: "Test Group", Order: 1}
		db.Create(&testGroup)

		// Act
		req := httptest.NewRequest("DELETE", "/api/groups/1", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)

		// Assert
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var respBody response.Response
		err = json.NewDecoder(resp.Body).Decode(&respBody)
		require.NoError(t, err)

		assert.True(t, respBody.Success)

		// Verify deletion in database
		var group models.Group
		result := db.First(&group, 1)
		assert.Error(t, result.Error)
		// Note: Cascade delete behavior depends on database constraints
		// In-memory SQLite may not enforce foreign key constraints by default
	})

	t.Run("givenNonExistentGroupID_whenDeleteGroup_thenReturnsNotFound", func(t *testing.T) {
		// Arrange
		db := setupTestDB(t)
		app := setupTestApp(db)

		// Act
		req := httptest.NewRequest("DELETE", "/api/groups/999", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)

		// Assert
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)

		var respBody response.Response
		err = json.NewDecoder(resp.Body).Decode(&respBody)
		require.NoError(t, err)

		assert.False(t, respBody.Success)
		assert.Contains(t, respBody.Error, "Group not found")
	})

	t.Run("givenInvalidGroupID_whenDeleteGroup_thenReturnsBadRequest", func(t *testing.T) {
		// Arrange
		db := setupTestDB(t)
		app := setupTestApp(db)

		// Act
		req := httptest.NewRequest("DELETE", "/api/groups/invalid", nil)
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
