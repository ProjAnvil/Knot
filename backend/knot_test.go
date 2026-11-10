package knot_test

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	knot "github.com/ProjAnvil/knot/backend"
	"github.com/ProjAnvil/knot/backend/internal/handlers"
	models "github.com/ProjAnvil/knot/models"
	"github.com/glebarez/sqlite"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// openTestDB opens a fresh in-memory SQLite database without migrating it:
// New owns migration, so the tests must observe the tables it creates.
func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	return db
}

// closeUnderTestDB closes the underlying sql.DB pool so subsequent queries
// fail; used to exercise the error paths of New/Export/Import.
func closeUnderTestDB(t *testing.T, db *gorm.DB) {
	t.Helper()
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())
}

// mustNew builds an engine via knot.New and fails the test on error.
func mustNew(t *testing.T, cfg knot.Config) *knot.Knot {
	t.Helper()
	k, err := knot.New(cfg)
	require.NoError(t, err)
	return k
}

// waitForFreshSecond sleeps until just past a whole-second boundary so two
// rapid calls (facade + handler) observe the same wall-clock second. The
// export payload and filename embed time.Now() with second granularity
// (RFC3339 / Unix), which makes byte parity deterministic.
func waitForFreshSecond(t *testing.T) {
	t.Helper()
	now := time.Now()
	until := now.Truncate(time.Second).Add(time.Second + 50*time.Millisecond)
	time.Sleep(time.Until(until))
}

// filenameFromDisposition extracts the quoted filename from a
// Content-Disposition header such as `attachment; filename="api-docs.html"`.
func filenameFromDisposition(t *testing.T, disposition string) string {
	t.Helper()
	require.Contains(t, disposition, "filename=")
	filename := strings.TrimSpace(strings.TrimPrefix(disposition, "attachment; filename="))
	return strings.Trim(filename, `"`)
}

// seedExportFixture seeds one group and two APIs (one with request and
// response parameters, one nested parameter child) and returns the API IDs.
func seedExportFixture(t *testing.T, db *gorm.DB) []uint {
	t.Helper()

	group := models.Group{Name: "Parity Group", Order: 3}
	require.NoError(t, db.Create(&group).Error)

	api1 := models.API{
		GroupID:  group.ID,
		Name:     "Create User",
		Endpoint: "/users",
		Method:   "POST",
		Type:     "HTTP",
	}
	require.NoError(t, db.Create(&api1).Error)

	reqParam := models.Parameter{
		APIID: api1.ID, Name: "username", Type: "string",
		ParamType: "request", Required: true, Order: 0,
	}
	require.NoError(t, db.Create(&reqParam).Error)

	nested := models.Parameter{
		APIID: api1.ID, ParentID: &reqParam.ID, Name: "profile", Type: "object",
		ParamType: "request", Order: 1,
	}
	require.NoError(t, db.Create(&nested).Error)

	respParam := models.Parameter{
		APIID: api1.ID, Name: "result", Type: "string",
		ParamType: "response", Order: 0,
	}
	require.NoError(t, db.Create(&respParam).Error)

	api2 := models.API{
		GroupID:  group.ID,
		Name:     "Get User",
		Endpoint: "/users/:id",
		Method:   "GET",
		Type:     "HTTP",
	}
	require.NoError(t, db.Create(&api2).Error)

	return []uint{api1.ID, api2.ID}
}

// knotImportFixture is a minimal valid Knot JSON v1 document: one group with
// one API and three request parameters (one nested).
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

// postImportRequest builds a multipart import request with a "file" field
// and a dryRun form value, targeting the given URL.
func postImportRequest(t *testing.T, url, dryRun string) *http.Request {
	t.Helper()

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, err := w.CreateFormFile("file", "import.json")
	require.NoError(t, err)
	_, err = fw.Write(knotImportFixture())
	require.NoError(t, err)
	require.NoError(t, w.WriteField("dryRun", dryRun))
	require.NoError(t, w.Close())

	req := httptest.NewRequest("POST", url, &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	return req
}

// postExportRequest builds a JSON export request targeting the given URL.
func postExportRequest(t *testing.T, url string, apiIDs []uint, format string) *http.Request {
	t.Helper()
	body := map[string]interface{}{"apiIds": apiIDs}
	if format != "" {
		body["format"] = format
	}
	jsonBody, err := json.Marshal(body)
	require.NoError(t, err)
	req := httptest.NewRequest("POST", url, bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	return req
}

// TestNew covers engine construction: nil-DB validation, migration of the
// canonical table set, custom table prefixes, prefix non-clobbering and
// migration failure reporting.
func TestNew(t *testing.T) {
	t.Run("givenNilDB_whenNew_thenReturnsError", func(t *testing.T) {
		k, err := knot.New(knot.Config{DB: nil})
		require.Error(t, err)
		assert.Nil(t, k)
		assert.Contains(t, err.Error(), "DB")
	})

	t.Run("givenValidDB_whenNew_thenMigratesAllTables", func(t *testing.T) {
		db := openTestDB(t)
		k := mustNew(t, knot.Config{DB: db})

		assert.NotNil(t, k)
		for _, table := range []string{"groups", "apis", "parameters"} {
			assert.True(t, db.Migrator().HasTable(table), "table %s should exist after New", table)
		}
	})

	t.Run("givenTablePrefix_whenNew_thenCreatesPrefixedTablesOnly", func(t *testing.T) {
		t.Cleanup(func() { models.SetTablePrefix("") })

		db := openTestDB(t)
		mustNew(t, knot.Config{DB: db, TablePrefix: "knot_"})

		for _, table := range []string{"knot_groups", "knot_apis", "knot_parameters"} {
			assert.True(t, db.Migrator().HasTable(table), "prefixed table %s should exist", table)
		}
		for _, table := range []string{"groups", "apis", "parameters"} {
			assert.False(t, db.Migrator().HasTable(table), "bare table %s should not exist", table)
		}
	})

	t.Run("givenEmptyTablePrefix_whenNew_thenKeepsHostSetPrefix", func(t *testing.T) {
		models.SetTablePrefix("host_")
		t.Cleanup(func() { models.SetTablePrefix("") })

		db := openTestDB(t)
		mustNew(t, knot.Config{DB: db})

		assert.Equal(t, "host_", models.TablePrefix, "New must not clobber a host-set prefix")
	})

	t.Run("givenBrokenDB_whenNew_thenReturnsWrappedError", func(t *testing.T) {
		db := openTestDB(t)
		closeUnderTestDB(t, db)

		k, err := knot.New(knot.Config{DB: db})
		require.Error(t, err)
		assert.Nil(t, k)
		assert.Contains(t, err.Error(), "migrate")
	})
}

// TestMountPrefixAndIdempotence covers route mounting: a custom API prefix
// isolates the engine from host routes, the default prefix stays "/api", and
// a double Mount must not duplicate or panic.
func TestMountPrefixAndIdempotence(t *testing.T) {
	t.Run("givenCustomPrefix_whenMount_thenRoutesReachableUnderPrefixOnly", func(t *testing.T) {
		db := openTestDB(t)
		k := mustNew(t, knot.Config{DB: db, APIPrefix: "/knot/api"})

		app := fiber.New(fiber.Config{DisableStartupMessage: true})
		k.Mount(app)

		req := httptest.NewRequest("GET", "/knot/api/groups", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		// The engine must not claim the unprefixed host namespace.
		req = httptest.NewRequest("GET", "/api/groups", nil)
		resp, err = app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})

	t.Run("givenDefaultConfig_whenMount_thenRoutesUnderAPIPrefix", func(t *testing.T) {
		db := openTestDB(t)
		k := mustNew(t, knot.Config{DB: db})

		app := fiber.New(fiber.Config{DisableStartupMessage: true})
		k.Mount(app)

		req := httptest.NewRequest("GET", "/api/groups", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})

	t.Run("givenDoubleMount_whenMount_thenSameRouteSetNoPanic", func(t *testing.T) {
		db := openTestDB(t)
		k := mustNew(t, knot.Config{DB: db, APIPrefix: "/knot/api"})

		app := fiber.New(fiber.Config{DisableStartupMessage: true})
		k.Mount(app)
		routesAfterFirst := countRoutes(app)

		require.NotPanics(t, func() { k.Mount(app) })
		assert.Equal(t, routesAfterFirst, countRoutes(app), "double Mount must not register duplicates")

		// The mounted routes still serve exactly one matching handler.
		req := httptest.NewRequest("GET", "/knot/api/groups", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})
}

// countRoutes totals the registered routes across all method stacks.
func countRoutes(app *fiber.App) int {
	count := 0
	for _, stack := range app.Stack() {
		count += len(stack)
	}
	return count
}

// TestMountPerApp covers per-app mount semantics: every distinct app the
// engine is mounted on receives the full route set exactly once, while a
// repeated Mount of the same app stays a no-op even under concurrency.
func TestMountPerApp(t *testing.T) {
	t.Run("givenSecondApp_whenMount_thenBothAppsServeFullRouteSet", func(t *testing.T) {
		db := openTestDB(t)
		k := mustNew(t, knot.Config{DB: db})

		appA := fiber.New(fiber.Config{DisableStartupMessage: true})
		appB := fiber.New(fiber.Config{DisableStartupMessage: true})
		k.Mount(appA)
		k.Mount(appB)

		// Both apps must serve the engine routes, not just the first one.
		for i, app := range []*fiber.App{appA, appB} {
			req := httptest.NewRequest("GET", "/api/groups", nil)
			resp, err := app.Test(req)
			require.NoError(t, err, "app %d should serve GET /api/groups", i)
			assert.Equal(t, fiber.StatusOK, resp.StatusCode, "app %d should serve GET /api/groups with 200", i)
		}

		// And each app carries the identical route set.
		assert.Equal(t, countRoutes(appA), countRoutes(appB),
			"both apps must carry the same route set")
	})

	t.Run("givenSameAppTwice_whenMount_thenRouteCountUnchanged", func(t *testing.T) {
		db := openTestDB(t)
		k := mustNew(t, knot.Config{DB: db})

		app := fiber.New(fiber.Config{DisableStartupMessage: true})
		k.Mount(app)
		routesAfterFirst := countRoutes(app)

		k.Mount(app)
		assert.Equal(t, routesAfterFirst, countRoutes(app),
			"a second Mount of the same app must not register duplicates")
	})

	t.Run("givenConcurrentMountsOfSameApp_whenMount_thenSingleRouteSet", func(t *testing.T) {
		db := openTestDB(t)
		k := mustNew(t, knot.Config{DB: db})

		app := fiber.New(fiber.Config{DisableStartupMessage: true})
		var wg sync.WaitGroup
		for i := 0; i < 8; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				k.Mount(app)
			}()
		}
		wg.Wait()

		// A sequential single Mount on a fresh app defines the expected count.
		reference := fiber.New(fiber.Config{DisableStartupMessage: true})
		k.Mount(reference)
		assert.Equal(t, countRoutes(reference), countRoutes(app),
			"concurrent Mounts of the same app must register exactly one route set")
	})
}

// TestExportParity asserts the facade Export returns byte-identical payloads
// and filenames to the HTTP export handler for every supported format.
func TestExportParity(t *testing.T) {
	db := openTestDB(t)
	k := mustNew(t, knot.Config{DB: db})
	apiIDs := seedExportFixture(t, db)

	// The handler side runs through the production route registration so the
	// parity is against the real HTTP surface, not a test-only replica.
	handlerApp := fiber.New(fiber.Config{DisableStartupMessage: true})
	handlers.SetupRoutes(handlerApp, db)

	tests := []struct {
		name           string
		format         string
		locale         string
		cookieLocale   string
		filenamePrefix string
		contentType    string
	}{
		{name: "emptyFormatDefaultsToHTML", format: "", locale: "", cookieLocale: "", filenamePrefix: "api-docs.html", contentType: "text/html; charset=utf-8"},
		{name: "html", format: "html", locale: "zh", cookieLocale: "zh", filenamePrefix: "api-docs.html", contentType: "text/html; charset=utf-8"},
		{name: "htmlEnglishLocale", format: "html", locale: "en", cookieLocale: "en", filenamePrefix: "api-docs.html", contentType: "text/html; charset=utf-8"},
		{name: "json", format: "json", locale: "zh", cookieLocale: "zh", filenamePrefix: "knot-export-", contentType: "application/json"},
		{name: "openapi", format: "openapi", locale: "zh", cookieLocale: "zh", filenamePrefix: "openapi-", contentType: "application/json"},
		{name: "postman", format: "postman", locale: "zh", cookieLocale: "zh", filenamePrefix: "postman-", contentType: "application/json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Both surfaces stamp the wall clock with second granularity, so
			// each attempt starts just after a second boundary. If the pair
			// straddles a boundary anyway, retry in a fresh second; a real
			// parity regression fails every attempt.
			var facadeBytes, handlerBytes []byte
			var facadeFilename, handlerFilename string
			for attempt := 0; attempt < 3; attempt++ {
				waitForFreshSecond(t)

				var err error
				facadeBytes, facadeFilename, err = k.Export(apiIDs, tt.format, tt.locale)
				require.NoError(t, err)

				req := postExportRequest(t, "/api/export", apiIDs, tt.format)
				if tt.cookieLocale != "" {
					req.Header.Set("Cookie", "locale="+tt.cookieLocale)
				}
				resp, err := handlerApp.Test(req)
				require.NoError(t, err)
				require.Equal(t, fiber.StatusOK, resp.StatusCode)
				assert.Equal(t, tt.contentType, resp.Header.Get("Content-Type"))

				handlerBytes, err = io.ReadAll(resp.Body)
				require.NoError(t, err)
				handlerFilename = filenameFromDisposition(t, resp.Header.Get("Content-Disposition"))

				if bytes.Equal(handlerBytes, facadeBytes) && handlerFilename == facadeFilename {
					break
				}
			}

			assert.NotNil(t, facadeBytes)
			assert.NotEmpty(t, facadeBytes)
			assert.True(t, bytes.Equal(handlerBytes, facadeBytes), "facade export bytes must equal handler response body")
			assert.Equal(t, handlerFilename, facadeFilename, "facade filename must equal handler filename")
			assert.True(t, strings.HasPrefix(facadeFilename, tt.filenamePrefix), "filename %q should start with %q", facadeFilename, tt.filenamePrefix)
		})
	}
}

// TestImportParity asserts the facade Import returns the same counts as the
// HTTP import handler on a dry run, and that a real import persists rows.
func TestImportParity(t *testing.T) {
	t.Run("givenKnotFixture_whenImportDryRun_thenCountsMatchHandler", func(t *testing.T) {
		db := openTestDB(t)
		k := mustNew(t, knot.Config{DB: db})

		handlerApp := fiber.New(fiber.Config{DisableStartupMessage: true})
		handlers.SetupRoutes(handlerApp, db)

		resp, err := handlerApp.Test(postImportRequest(t, "/api/import", "true"))
		require.NoError(t, err)
		require.Equal(t, fiber.StatusOK, resp.StatusCode)

		var body struct {
			Success bool               `json:"success"`
			Data    importResponseData `json:"data"`
		}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		require.True(t, body.Success)

		result, err := k.Import(knotImportFixture(), true)
		require.NoError(t, err)

		assert.Equal(t, body.Data.Format, result.Format)
		assert.Equal(t, body.Data.GroupsCreated, result.GroupsCreated)
		assert.Equal(t, body.Data.GroupsMerged, result.GroupsMerged)
		assert.Equal(t, body.Data.APIsImported, result.APIsImported)
		assert.Equal(t, body.Data.ParametersImported, result.ParametersImported)
		assert.Equal(t, body.Data.Warnings, result.Warnings)

		// A dry run must leave the database untouched on both paths.
		var groupCount, apiCount int64
		require.NoError(t, db.Model(&models.Group{}).Count(&groupCount).Error)
		require.NoError(t, db.Model(&models.API{}).Count(&apiCount).Error)
		assert.Equal(t, int64(0), groupCount)
		assert.Equal(t, int64(0), apiCount)
	})

	t.Run("givenKnotFixture_whenImportForReal_thenPersistsRows", func(t *testing.T) {
		db := openTestDB(t)
		k := mustNew(t, knot.Config{DB: db})

		result, err := k.Import(knotImportFixture(), false)
		require.NoError(t, err)

		assert.Equal(t, "knot-json", result.Format)
		assert.Equal(t, 1, result.GroupsCreated)
		assert.Equal(t, 0, result.GroupsMerged)
		assert.Equal(t, 1, result.APIsImported)
		assert.Equal(t, 3, result.ParametersImported)

		var groups []models.Group
		require.NoError(t, db.Find(&groups).Error)
		require.Len(t, groups, 1)
		assert.Equal(t, "Imported Group", groups[0].Name)

		var apis []models.API
		require.NoError(t, db.Find(&apis).Error)
		require.Len(t, apis, 1)
		assert.Equal(t, "Create User", apis[0].Name)

		var params []models.Parameter
		require.NoError(t, db.Find(&params).Error)
		assert.Len(t, params, 3)
	})
}

// TestExportErrors pins the facade export error contract: distinct sentinel
// errors the host can map to HTTP statuses.
func TestExportErrors(t *testing.T) {
	t.Run("givenNoAPIIDs_whenExport_thenErrNoAPIIDs", func(t *testing.T) {
		db := openTestDB(t)
		k := mustNew(t, knot.Config{DB: db})

		payload, filename, err := k.Export(nil, "html", "zh")
		assert.ErrorIs(t, err, knot.ErrNoAPIIDs)
		assert.Nil(t, payload)
		assert.Empty(t, filename)
	})

	t.Run("givenUnknownFormat_whenExport_thenErrUnsupportedFormat", func(t *testing.T) {
		db := openTestDB(t)
		k := mustNew(t, knot.Config{DB: db})
		apiIDs := seedExportFixture(t, db)

		payload, filename, err := k.Export(apiIDs, "yaml", "zh")
		assert.ErrorIs(t, err, knot.ErrUnsupportedFormat)
		assert.Nil(t, payload)
		assert.Empty(t, filename)
	})

	t.Run("givenBrokenDB_whenExport_thenErrExportFailed", func(t *testing.T) {
		db := openTestDB(t)
		k := mustNew(t, knot.Config{DB: db})
		apiIDs := seedExportFixture(t, db)
		closeUnderTestDB(t, db)

		payload, filename, err := k.Export(apiIDs, "json", "zh")
		assert.ErrorIs(t, err, knot.ErrExportFailed)
		assert.Nil(t, payload)
		assert.Empty(t, filename)
	})
}

// TestImportErrors pins the facade import error contract: unsupported input
// maps to a 415-mappable sentinel, parse and apply failures are distinct.
func TestImportErrors(t *testing.T) {
	t.Run("givenNonJSONInput_whenImport_thenErrUnsupportedFormat", func(t *testing.T) {
		db := openTestDB(t)
		k := mustNew(t, knot.Config{DB: db})

		result, err := k.Import([]byte("definitely not json"), true)
		assert.ErrorIs(t, err, knot.ErrUnsupportedFormat)
		assert.Nil(t, result)
	})

	t.Run("givenUndetectableInput_whenImport_thenErrUnsupportedFormat", func(t *testing.T) {
		db := openTestDB(t)
		k := mustNew(t, knot.Config{DB: db})

		// Valid JSON without any known format signature is a 415 case.
		result, err := k.Import([]byte(`{"hello": "world"}`), true)
		assert.ErrorIs(t, err, knot.ErrUnsupportedFormat)
		assert.Nil(t, result)
	})

	t.Run("givenBadKnotVersion_whenImport_thenErrImportParse", func(t *testing.T) {
		db := openTestDB(t)
		k := mustNew(t, knot.Config{DB: db})

		// Detect accepts the knot signature but the parser rejects version 2.
		result, err := k.Import([]byte(`{"format": "knot", "version": 2}`), true)
		assert.ErrorIs(t, err, knot.ErrImportParse)
		assert.Nil(t, result)
	})

	t.Run("givenBrokenDB_whenImport_thenErrImportApply", func(t *testing.T) {
		db := openTestDB(t)
		k := mustNew(t, knot.Config{DB: db})
		closeUnderTestDB(t, db)

		result, err := k.Import(knotImportFixture(), false)
		assert.ErrorIs(t, err, knot.ErrImportApply)
		assert.Nil(t, result)
	})
}

// TestClose verifies the documented no-op Close.
func TestClose(t *testing.T) {
	db := openTestDB(t)
	k := mustNew(t, knot.Config{DB: db})
	assert.NoError(t, k.Close())
}
