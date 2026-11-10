package services

import (
	"errors"
	"strings"
	"testing"
	"time"

	models "github.com/ProjAnvil/knot/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// exportBuildNow is the fixed instant used by the build tests so filenames
// and embedded timestamps are deterministic.
var exportBuildNow = time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)

// seedExportBuildFixture seeds one group with one API carrying one request
// and one response parameter, returning the API ID.
func seedExportBuildFixture(t *testing.T, db *gorm.DB) []uint {
	t.Helper()

	group := models.Group{Name: "Build Group", Order: 1}
	require.NoError(t, db.Create(&group).Error)

	api := models.API{
		GroupID: group.ID, Name: "Build API",
		Endpoint: "/build", Method: "GET", Type: "HTTP",
	}
	require.NoError(t, db.Create(&api).Error)

	require.NoError(t, db.Create(&models.Parameter{
		APIID: api.ID, Name: "q", Type: "string",
		ParamType: "request", Required: true, Order: 0,
	}).Error)
	require.NoError(t, db.Create(&models.Parameter{
		APIID: api.ID, Name: "out", Type: "string",
		ParamType: "response", Order: 0,
	}).Error)

	return []uint{api.ID}
}

// TestBuildExportDocument covers the shared export builder: format
// selection, filenames, content types, locale defaulting and the sentinel
// and database error paths.
func TestBuildExportDocument(t *testing.T) {
	t.Run("givenNoAPIIDs_whenBuild_thenErrNoAPIIDs", func(t *testing.T) {
		db := setupTestDB(t)

		payload, filename, contentType, err := BuildExportDocument(db, nil, "html", "zh", exportBuildNow)

		assert.ErrorIs(t, err, ErrNoAPIIDs)
		assert.Nil(t, payload)
		assert.Empty(t, filename)
		assert.Empty(t, contentType)
	})

	t.Run("givenHTMLFormat_whenBuild_thenHTMLDocument", func(t *testing.T) {
		db := setupTestDB(t)
		apiIDs := seedExportBuildFixture(t, db)

		payload, filename, contentType, err := BuildExportDocument(db, apiIDs, "html", "en", exportBuildNow)
		require.NoError(t, err)

		assert.Equal(t, "api-docs.html", filename)
		assert.Equal(t, "text/html; charset=utf-8", contentType)
		assert.Contains(t, string(payload), "<!DOCTYPE html>")
		assert.Contains(t, string(payload), "Build API")
	})

	t.Run("givenEmptyFormatAndLocale_whenBuild_thenHTMLDefaults", func(t *testing.T) {
		db := setupTestDB(t)
		apiIDs := seedExportBuildFixture(t, db)

		payloadByDefault, _, _, err := BuildExportDocument(db, apiIDs, "", "", exportBuildNow)
		require.NoError(t, err)
		payloadExplicit, _, _, err := BuildExportDocument(db, apiIDs, "html", "zh", exportBuildNow)
		require.NoError(t, err)

		// "" format behaves as "html" and "" locale defaults to "zh".
		assert.Equal(t, string(payloadExplicit), string(payloadByDefault))
		assert.Contains(t, string(payloadByDefault), "Build API")
	})

	t.Run("givenJSONFormats_whenBuild_thenStampedJSONDocuments", func(t *testing.T) {
		db := setupTestDB(t)
		apiIDs := seedExportBuildFixture(t, db)

		tests := []struct {
			format         string
			filenamePrefix string
			marker         string
		}{
			{format: "json", filenamePrefix: "knot-export-", marker: `"format":"knot"`},
			{format: "openapi", filenamePrefix: "openapi-", marker: `"openapi":"3.1.0"`},
			{format: "postman", filenamePrefix: "postman-", marker: "schema.getpostman.com"},
		}
		for _, tt := range tests {
			payload, filename, contentType, err := BuildExportDocument(db, apiIDs, tt.format, "zh", exportBuildNow)
			require.NoError(t, err)

			assert.Equal(t, tt.filenamePrefix+"1794312000.json", filename)
			assert.Equal(t, "application/json", contentType)
			assert.True(t, strings.HasPrefix(string(payload), "{"), "payload should be a JSON object")
			assert.Contains(t, string(payload), tt.marker)
		}
	})

	t.Run("givenUnknownFormat_whenBuild_thenErrUnsupportedExportFormat", func(t *testing.T) {
		db := setupTestDB(t)
		apiIDs := seedExportBuildFixture(t, db)

		payload, filename, contentType, err := BuildExportDocument(db, apiIDs, "yaml", "zh", exportBuildNow)

		assert.ErrorIs(t, err, ErrUnsupportedExportFormat)
		assert.Nil(t, payload)
		assert.Empty(t, filename)
		assert.Empty(t, contentType)
	})

	t.Run("givenBrokenDB_whenBuild_thenWrappedFetchError", func(t *testing.T) {
		db := setupTestDB(t)
		apiIDs := seedExportBuildFixture(t, db)
		sqlDB, err := db.DB()
		require.NoError(t, err)
		require.NoError(t, sqlDB.Close())

		_, _, _, err = BuildExportDocument(db, apiIDs, "json", "zh", exportBuildNow)
		require.Error(t, err)
		assert.False(t, errors.Is(err, ErrNoAPIIDs))
		assert.False(t, errors.Is(err, ErrUnsupportedExportFormat))
		assert.Contains(t, err.Error(), "failed to fetch")
	})

	t.Run("givenUnknownAPIIDs_whenBuild_thenEmptyDocumentNotError", func(t *testing.T) {
		db := setupTestDB(t)

		// IDs that match no rows are not an error: the builder renders an
		// empty document, mirroring the handler.
		payload, filename, _, err := BuildExportDocument(db, []uint{999}, "json", "zh", exportBuildNow)
		require.NoError(t, err)
		assert.Equal(t, "knot-export-1794312000.json", filename)
		assert.Contains(t, string(payload), `"format":"knot"`)
	})
}
