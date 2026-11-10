package handlers

import (
	"encoding/json"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/ProjAnvil/knot/backend/pkg/response"
	models "github.com/ProjAnvil/knot/models"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// utoa formats a model ID for use in query strings and URLs.
func utoa(id uint) string {
	return strconv.FormatUint(uint64(id), 10)
}

// listFixture seeds two groups and four APIs whose intra-group order and
// group order are deliberately scrambled relative to the expected flat
// listing (group order first, then API order, then id):
//
//	Auth (order 1):     Sign Out (order 1), User Login (order 2)
//	Payments (order 2): Charge (order 1), Refund (order 2)
//
// Expected flat order: Sign Out, User Login, Charge, Refund.
type listFixture struct {
	auth     models.Group
	payments models.Group
	signOut  models.API
	login    models.API
	charge   models.API
	refund   models.API
}

// seedListFixture creates the standard ListAPIs test data in db.
func seedListFixture(t *testing.T, db *gorm.DB) listFixture {
	t.Helper()

	fx := listFixture{
		auth:     models.Group{Name: "Auth", Order: 1},
		payments: models.Group{Name: "Payments", Order: 2},
	}
	require.NoError(t, db.Create(&fx.auth).Error)
	require.NoError(t, db.Create(&fx.payments).Error)

	fx.signOut = models.API{GroupID: fx.auth.ID, Name: "Sign Out", Endpoint: "/AUTH/logout", Method: "POST", Type: "HTTP", Order: 1}
	fx.login = models.API{GroupID: fx.auth.ID, Name: "User Login", Endpoint: "/auth/login", Method: "POST", Type: "HTTP", Order: 2}
	fx.charge = models.API{GroupID: fx.payments.ID, Name: "Charge", Endpoint: "/payments/charge", Method: "POST", Type: "HTTP", Order: 1}
	fx.refund = models.API{GroupID: fx.payments.ID, Name: "Refund", Endpoint: "/payments/refund", Method: "GET", Type: "HTTP", Order: 2}
	require.NoError(t, db.Create(&fx.signOut).Error)
	require.NoError(t, db.Create(&fx.login).Error)
	require.NoError(t, db.Create(&fx.charge).Error)
	require.NoError(t, db.Create(&fx.refund).Error)

	return fx
}

// getListAPIs performs a GET /api/apis request (optionally with a raw query
// string) and decodes the standard response envelope.
func getListAPIs(t *testing.T, app *fiber.App, rawQuery string) (int, *response.Response) {
	t.Helper()

	target := "/api/apis"
	if rawQuery != "" {
		target += "?" + rawQuery
	}

	req := httptest.NewRequest("GET", target, nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	var body response.Response
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))

	return resp.StatusCode, &body
}

// listData extracts the ListAPIs data object from a response envelope.
func listData(t *testing.T, body *response.Response) map[string]interface{} {
	t.Helper()

	data, ok := body.Data.(map[string]interface{})
	require.True(t, ok, "response data should be the list envelope {items,total,page,pageSize}, got %T", body.Data)
	return data
}

// itemNames extracts the ordered API names from the items array.
func itemNames(t *testing.T, data map[string]interface{}) []string {
	t.Helper()

	items, ok := data["items"].([]interface{})
	require.True(t, ok, "items should be a JSON array, got %T", data["items"])

	names := make([]string, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]interface{})
		require.True(t, ok)
		names = append(names, m["name"].(string))
	}
	return names
}

// TestListAPIs tests the ListAPIs handler.
func TestListAPIs(t *testing.T) {
	t.Run("givenNoParams_whenListAPIs_thenReturnsDefaultsOrderedByGroupThenAPIOrder", func(t *testing.T) {
		// Arrange
		db := setupTestDB(t)
		app := setupAPIsTestApp(db)
		seedListFixture(t, db)

		// Act
		status, body := getListAPIs(t, app, "")

		// Assert
		assert.Equal(t, fiber.StatusOK, status)
		assert.True(t, body.Success)

		data := listData(t, body)
		// Defaults: page 1, pageSize 50.
		assert.Equal(t, float64(1), data["page"])
		assert.Equal(t, float64(50), data["pageSize"])
		assert.Equal(t, float64(4), data["total"])
		assert.Equal(t, []string{"Sign Out", "User Login", "Charge", "Refund"}, itemNames(t, data))

		// List items must not embed parameters (detail payload only) and
		// should carry the parent group for display.
		items := data["items"].([]interface{})
		for _, item := range items {
			m := item.(map[string]interface{})
			assert.NotContains(t, m, "parameters")
			group, ok := m["group"].(map[string]interface{})
			require.True(t, ok, "item should embed its group")
			assert.Contains(t, []interface{}{"Auth", "Payments"}, group["name"])
		}
	})

	t.Run("givenPaginationParams_whenListAPIs_thenReturnsRequestedPageSlice", func(t *testing.T) {
		// Arrange
		db := setupTestDB(t)
		app := setupAPIsTestApp(db)
		seedListFixture(t, db)

		// Act: second page of two items.
		status, body := getListAPIs(t, app, "page=2&pageSize=2")

		// Assert
		assert.Equal(t, fiber.StatusOK, status)
		data := listData(t, body)
		assert.Equal(t, float64(2), data["page"])
		assert.Equal(t, float64(2), data["pageSize"])
		// Total counts the whole filtered set, not the page.
		assert.Equal(t, float64(4), data["total"])
		assert.Equal(t, []string{"Charge", "Refund"}, itemNames(t, data))

		// Act: a ragged last page (4 items, page size 3, page 2).
		status, body = getListAPIs(t, app, "page=2&pageSize=3")

		// Assert
		assert.Equal(t, fiber.StatusOK, status)
		data = listData(t, body)
		assert.Equal(t, float64(4), data["total"])
		assert.Equal(t, []string{"Refund"}, itemNames(t, data))
	})

	t.Run("givenOutOfRangeParams_whenListAPIs_thenClampsValues", func(t *testing.T) {
		// Arrange
		db := setupTestDB(t)
		app := setupAPIsTestApp(db)
		seedListFixture(t, db)

		// Act / Assert: page clamps up to 1, pageSize clamps into 1..100.
		status, body := getListAPIs(t, app, "page=0&pageSize=0")
		assert.Equal(t, fiber.StatusOK, status)
		data := listData(t, body)
		assert.Equal(t, float64(1), data["page"])
		assert.Equal(t, float64(1), data["pageSize"])
		assert.Equal(t, float64(4), data["total"])
		assert.Equal(t, []string{"Sign Out"}, itemNames(t, data))

		status, body = getListAPIs(t, app, "page=-3&pageSize=500")
		assert.Equal(t, fiber.StatusOK, status)
		data = listData(t, body)
		assert.Equal(t, float64(1), data["page"])
		assert.Equal(t, float64(100), data["pageSize"])
		assert.Equal(t, 4, len(data["items"].([]interface{})))
	})

	t.Run("givenKeywordMatchingName_whenListAPIs_thenMatchesCaseInsensitively", func(t *testing.T) {
		// Arrange
		db := setupTestDB(t)
		app := setupAPIsTestApp(db)
		seedListFixture(t, db)

		// Act: keyword casing differs from the stored name "User Login".
		status, body := getListAPIs(t, app, "keyword=LOGIN")

		// Assert
		assert.Equal(t, fiber.StatusOK, status)
		data := listData(t, body)
		assert.Equal(t, float64(1), data["total"])
		assert.Equal(t, []string{"User Login"}, itemNames(t, data))
	})

	t.Run("givenKeywordMatchingEndpoint_whenListAPIs_thenMatchesCaseInsensitively", func(t *testing.T) {
		// Arrange
		db := setupTestDB(t)
		app := setupAPIsTestApp(db)
		seedListFixture(t, db)

		// Act: keyword matches the stored endpoint "/AUTH/logout" only after
		// lowercasing both sides.
		status, body := getListAPIs(t, app, "keyword=logout")

		// Assert
		assert.Equal(t, fiber.StatusOK, status)
		data := listData(t, body)
		assert.Equal(t, float64(1), data["total"])
		assert.Equal(t, []string{"Sign Out"}, itemNames(t, data))
	})

	t.Run("givenGroupIdFilter_whenListAPIs_thenReturnsOnlyThatGroup", func(t *testing.T) {
		// Arrange
		db := setupTestDB(t)
		app := setupAPIsTestApp(db)
		fx := seedListFixture(t, db)

		// Act
		status, body := getListAPIs(t, app, "groupId="+utoa(fx.payments.ID))

		// Assert
		assert.Equal(t, fiber.StatusOK, status)
		data := listData(t, body)
		assert.Equal(t, float64(2), data["total"])
		assert.Equal(t, []string{"Charge", "Refund"}, itemNames(t, data))
	})

	t.Run("givenMethodFilter_whenListAPIs_thenMatchesCaseInsensitively", func(t *testing.T) {
		// Arrange
		db := setupTestDB(t)
		app := setupAPIsTestApp(db)
		seedListFixture(t, db)

		// Act: lowercase input must match the stored "GET" exactly.
		status, body := getListAPIs(t, app, "method=get")

		// Assert
		assert.Equal(t, fiber.StatusOK, status)
		data := listData(t, body)
		assert.Equal(t, float64(1), data["total"])
		assert.Equal(t, []string{"Refund"}, itemNames(t, data))
	})

	t.Run("givenCombinedFilters_whenListAPIs_thenIntersectsAllFilters", func(t *testing.T) {
		// Arrange
		db := setupTestDB(t)
		app := setupAPIsTestApp(db)
		fx := seedListFixture(t, db)

		// Act: groupId + method + keyword together.
		status, body := getListAPIs(t, app,
			"groupId="+utoa(fx.payments.ID)+"&method=post&keyword=char")

		// Assert
		assert.Equal(t, fiber.StatusOK, status)
		data := listData(t, body)
		assert.Equal(t, float64(1), data["total"])
		assert.Equal(t, []string{"Charge"}, itemNames(t, data))

		// Act: keyword OR semantics across both columns inside one group
		// ("auth" hits neither Auth name but both Auth endpoints).
		status, body = getListAPIs(t, app,
			"groupId="+utoa(fx.auth.ID)+"&keyword=auth")

		// Assert: both Auth APIs match by endpoint; Payments APIs do not.
		assert.Equal(t, fiber.StatusOK, status)
		data = listData(t, body)
		assert.Equal(t, float64(2), data["total"])
		assert.Equal(t, []string{"Sign Out", "User Login"}, itemNames(t, data))

		// Act: a keyword whose endpoint matches live only OUTSIDE the
		// filtered group. If the OR expression were not parenthesized with
		// the groupId condition, the Payments APIs would leak in.
		status, body = getListAPIs(t, app,
			"groupId="+utoa(fx.auth.ID)+"&keyword=pay")

		// Assert
		assert.Equal(t, fiber.StatusOK, status)
		data = listData(t, body)
		assert.Equal(t, float64(0), data["total"])
		assert.Empty(t, data["items"].([]interface{}))
	})

	t.Run("givenNoMatches_whenListAPIs_thenReturnsEmptyItemsArrayAndZeroTotal", func(t *testing.T) {
		// Arrange
		db := setupTestDB(t)
		app := setupAPIsTestApp(db)
		seedListFixture(t, db)

		// Act
		status, body := getListAPIs(t, app, "keyword=no-such-api")

		// Assert
		assert.Equal(t, fiber.StatusOK, status)
		data := listData(t, body)
		assert.Equal(t, float64(0), data["total"])
		// items must serialize as [] — a JSON null would break clients.
		items, ok := data["items"].([]interface{})
		require.True(t, ok, "items must be a JSON array, not null")
		assert.Empty(t, items)
	})

	t.Run("givenInvalidPagination_whenListAPIs_thenReturnsBadRequest", func(t *testing.T) {
		// Arrange
		db := setupTestDB(t)
		app := setupAPIsTestApp(db)
		seedListFixture(t, db)

		for _, rawQuery := range []string{"page=abc", "pageSize=abc", "page=1.5"} {
			// Act
			status, body := getListAPIs(t, app, rawQuery)

			// Assert
			assert.Equal(t, fiber.StatusBadRequest, status, "query %q", rawQuery)
			assert.False(t, body.Success)
			assert.Contains(t, body.Error, "Invalid pagination parameters")
		}
	})

	t.Run("givenInvalidGroupId_whenListAPIs_thenReturnsBadRequest", func(t *testing.T) {
		// Arrange
		db := setupTestDB(t)
		app := setupAPIsTestApp(db)
		seedListFixture(t, db)

		// Act
		status, body := getListAPIs(t, app, "groupId=not-a-number")

		// Assert
		assert.Equal(t, fiber.StatusBadRequest, status)
		assert.False(t, body.Success)
		assert.Contains(t, body.Error, "Invalid group ID")
	})

	t.Run("givenBrokenDB_whenListAPIs_thenReturnsInternalError", func(t *testing.T) {
		// Arrange
		db := setupTestDB(t)
		app := setupAPIsTestApp(db)
		seedListFixture(t, db)

		sqlDB, err := db.DB()
		require.NoError(t, err)
		require.NoError(t, sqlDB.Close())

		// Act
		status, body := getListAPIs(t, app, "")

		// Assert: the count query is the first statement to touch the DB.
		assert.Equal(t, fiber.StatusInternalServerError, status)
		assert.False(t, body.Success)
		assert.Contains(t, body.Error, "Failed to count APIs")
	})

	t.Run("givenTablePrefix_whenListAPIs_thenJoinAndOrderSurvivePrefix", func(t *testing.T) {
		// Arrange: the prefix must be set BEFORE the DB is opened because
		// GORM parses and caches table names per *gorm.DB on first use.
		models.SetTablePrefix("knot_")
		t.Cleanup(func() { models.SetTablePrefix("") })

		db := setupTestDB(t)
		app := setupAPIsTestApp(db)
		fx := seedListFixture(t, db)

		// Sanity: only prefixed tables exist, so a raw "groups" join would
		// fail with "no such table".
		assert.True(t, db.Migrator().HasTable("knot_groups"))
		assert.True(t, db.Migrator().HasTable("knot_apis"))

		// Act: unfiltered list exercises the prefixed JOIN + ORDER BY.
		status, body := getListAPIs(t, app, "")

		// Assert
		assert.Equal(t, fiber.StatusOK, status)
		data := listData(t, body)
		assert.Equal(t, float64(4), data["total"])
		assert.Equal(t, []string{"Sign Out", "User Login", "Charge", "Refund"}, itemNames(t, data))

		// Act: filters must survive the prefix too.
		status, body = getListAPIs(t, app, "groupId="+utoa(fx.auth.ID)+"&keyword=login&method=post")
		assert.Equal(t, fiber.StatusOK, status)
		data = listData(t, body)
		assert.Equal(t, float64(1), data["total"])
		assert.Equal(t, []string{"User Login"}, itemNames(t, data))
	})
}

// TestListAndDetailCoexistence proves the static "/" list route and the
// "/:id" detail route resolve independently (static beats param in Fiber).
func TestListAndDetailCoexistence(t *testing.T) {
	t.Run("givenListAndDetailRoutes_whenBothRequested_thenEachReturnsItsShape", func(t *testing.T) {
		// Arrange
		db := setupTestDB(t)
		app := setupAPIsTestApp(db)
		fx := seedListFixture(t, db)

		// The detail API also owns a parameter, proving the detail route
		// still preloads parameters while the list route omits them.
		param := models.Parameter{APIID: fx.refund.ID, Name: "amount", Type: "number", ParamType: "request", Order: 1}
		require.NoError(t, db.Create(&param).Error)

		// Act: the flat list.
		status, body := getListAPIs(t, app, "")

		// Assert: list envelope, not a detail object.
		assert.Equal(t, fiber.StatusOK, status)
		data := listData(t, body)
		assert.Equal(t, float64(4), data["total"])
		items := data["items"].([]interface{})
		refundItem, ok := items[3].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "Refund", refundItem["name"])
		assert.NotContains(t, refundItem, "parameters")

		// Act: the detail for the same API id.
		req := httptest.NewRequest("GET", "/api/apis/"+utoa(fx.refund.ID), nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var detail response.Response
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&detail))
		assert.True(t, detail.Success)
		detailData, ok := detail.Data.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "Refund", detailData["name"])
		parameters, ok := detailData["parameters"].([]interface{})
		require.True(t, ok, "detail must preload parameters")
		assert.Len(t, parameters, 1)
	})
}
