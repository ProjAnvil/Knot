package knot

import (
	"errors"
	"testing"

	models "github.com/ProjAnvil/knot/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseImportDocument pins the import dispatch contract of the
// unexported parseImportDocument helper: every detected format reaches its
// importer, parse failures come back wrapped in ErrImportParse, and the
// defensive unknown-format branch (unreachable through Import because
// DetectImportFormat only returns known formats) reports
// ErrUnsupportedFormat.
func TestParseImportDocument(t *testing.T) {
	t.Run("givenKnotFormat_whenParse_thenPlanBuilt", func(t *testing.T) {
		data := []byte(`{"format":"knot","version":1,"groups":[{"name":"G","apis":[{"name":"A","endpoint":"/a","method":"get"}]}]}`)

		plan, err := parseImportDocument(models.ImportFormatKnot, data)
		require.NoError(t, err)
		require.Len(t, plan.Groups, 1)
		assert.Equal(t, "G", plan.Groups[0].Name)
	})

	t.Run("givenPostmanFormat_whenParse_thenPlanBuilt", func(t *testing.T) {
		data := []byte(`{"info":{"name":"C","schema":"https://schema.getpostman.com/json/collection/v2.1.0/collection.json"},"item":[]}`)

		plan, err := parseImportDocument(models.ImportFormatPostman, data)
		require.NoError(t, err)
		require.NotNil(t, plan)
	})

	t.Run("givenOpenAPIFormat_whenParse_thenPlanBuilt", func(t *testing.T) {
		data := []byte(`{"openapi":"3.1.0","info":{"title":"T","version":"1.0.0"},"paths":{}}`)

		plan, err := parseImportDocument(models.ImportFormatOpenAPI, data)
		require.NoError(t, err)
		require.NotNil(t, plan)
	})

	t.Run("givenParseFailure_whenParse_thenErrImportParse", func(t *testing.T) {
		// The knot signature is present but version 2 is rejected by the
		// importer itself.
		_, err := parseImportDocument(models.ImportFormatKnot, []byte(`{"format":"knot","version":2}`))
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrImportParse)
	})

	t.Run("givenUnknownFormat_whenParse_thenErrUnsupportedFormat", func(t *testing.T) {
		_, err := parseImportDocument(models.ImportFormat("bogus"), []byte(`{}`))
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrUnsupportedFormat)
		assert.False(t, errors.Is(err, ErrImportParse))
	})
}
