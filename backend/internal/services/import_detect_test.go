package services

import (
	"strings"
	"testing"

	models "github.com/ProjAnvil/knot/models"
)

// TestDetectImportFormat covers spec §8.2 format detection: disjoint root
// signatures for knot-json / postman / openapi 3, the Swagger 2.0 rejection
// hint, UTF-8 BOM tolerance (Review Focus 1), and invalid-input errors.
func TestDetectImportFormat(t *testing.T) {
	t.Run("GivenKnotFile_whenDetect_thenKnotFormat", func(t *testing.T) {
		// Arrange
		data := []byte(`{"format":"knot","version":1,"exportedAt":"2026-11-10T12:00:00Z","groups":[]}`)

		// Act
		format, err := DetectImportFormat(data)

		// Assert
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if format != models.ImportFormatKnot {
			t.Errorf("Expected format %q, got %q", models.ImportFormatKnot, format)
		}
	})

	t.Run("GivenPostmanV21Schema_whenDetect_thenPostmanFormat", func(t *testing.T) {
		// Arrange
		data := []byte(`{"info":{"name":"Demo","schema":"https://schema.getpostman.com/json/collection/v2.1.0/collection.json"},"item":[]}`)

		// Act
		format, err := DetectImportFormat(data)

		// Assert
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if format != models.ImportFormatPostman {
			t.Errorf("Expected format %q, got %q", models.ImportFormatPostman, format)
		}
	})

	t.Run("GivenPostmanV20Schema_whenDetect_thenPostmanFormat", func(t *testing.T) {
		// Arrange
		data := []byte(`{"info":{"_postman_id":"abc","name":"Demo","schema":"https://schema.getpostman.com/json/collection/v2.0.0/collection.json"},"item":[]}`)

		// Act
		format, err := DetectImportFormat(data)

		// Assert
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if format != models.ImportFormatPostman {
			t.Errorf("Expected format %q, got %q", models.ImportFormatPostman, format)
		}
	})

	t.Run("GivenOpenAPI30_whenDetect_thenOpenAPIFormat", func(t *testing.T) {
		// Arrange
		data := []byte(`{"openapi":"3.0.3","info":{"title":"Demo","version":"1.0.0"},"paths":{}}`)

		// Act
		format, err := DetectImportFormat(data)

		// Assert
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if format != models.ImportFormatOpenAPI {
			t.Errorf("Expected format %q, got %q", models.ImportFormatOpenAPI, format)
		}
	})

	t.Run("GivenOpenAPI31_whenDetect_thenOpenAPIFormat", func(t *testing.T) {
		// Arrange
		data := []byte(`{"openapi":"3.1.0","info":{"title":"Demo","version":"1.0.0"},"paths":{}}`)

		// Act
		format, err := DetectImportFormat(data)

		// Assert
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if format != models.ImportFormatOpenAPI {
			t.Errorf("Expected format %q, got %q", models.ImportFormatOpenAPI, format)
		}
	})

	t.Run("GivenSwagger20_whenDetect_thenErrorMentionsOpenAPI3", func(t *testing.T) {
		// Arrange
		data := []byte(`{"swagger":"2.0","info":{"title":"Demo","version":"1.0.0"},"paths":{}}`)

		// Act
		format, err := DetectImportFormat(data)

		// Assert - rejected with a hint naming OpenAPI 3 as the requirement
		if err == nil {
			t.Fatal("Expected an error for a Swagger 2.0 file, got nil")
		}
		if format != "" {
			t.Errorf("Expected empty format on error, got %q", format)
		}
		if !strings.Contains(err.Error(), "OpenAPI 3") {
			t.Errorf("Expected error to mention \"OpenAPI 3\", got %q", err.Error())
		}
	})

	t.Run("GivenOpenAPIMajorVersion4_whenDetect_thenError", func(t *testing.T) {
		// Arrange - only major version 3 is accepted
		data := []byte(`{"openapi":"4.0.0","info":{"title":"Demo","version":"1.0.0"},"paths":{}}`)

		// Act
		format, err := DetectImportFormat(data)

		// Assert
		if err == nil {
			t.Fatal("Expected an error for an OpenAPI 4 document, got nil")
		}
		if format != "" {
			t.Errorf("Expected empty format on error, got %q", format)
		}
	})

	t.Run("GivenUnknownJSON_whenDetect_thenError", func(t *testing.T) {
		// Arrange - valid JSON but no known signature
		data := []byte(`{"hello":"world"}`)

		// Act
		format, err := DetectImportFormat(data)

		// Assert
		if err == nil {
			t.Fatal("Expected an error for an unrecognized document, got nil")
		}
		if format != "" {
			t.Errorf("Expected empty format on error, got %q", format)
		}
	})

	t.Run("GivenNotJSON_whenDetect_thenError", func(t *testing.T) {
		// Arrange
		data := []byte("this is definitely not json")

		// Act
		format, err := DetectImportFormat(data)

		// Assert
		if err == nil {
			t.Fatal("Expected an error for non-JSON input, got nil")
		}
		if format != "" {
			t.Errorf("Expected empty format on error, got %q", format)
		}
	})

	t.Run("GivenBOMPrefixedKnotFile_whenDetect_thenKnotFormat", func(t *testing.T) {
		// Arrange - Review Focus 1: files edited on Windows often carry a UTF-8 BOM
		data := append([]byte{0xEF, 0xBB, 0xBF}, []byte(`{"format":"knot","version":1,"groups":[]}`)...)

		// Act
		format, err := DetectImportFormat(data)

		// Assert
		if err != nil {
			t.Fatalf("Expected no error for BOM-prefixed knot file, got %v", err)
		}
		if format != models.ImportFormatKnot {
			t.Errorf("Expected format %q, got %q", models.ImportFormatKnot, format)
		}
	})

	t.Run("GivenEmptyInput_whenDetect_thenError", func(t *testing.T) {
		// Arrange
		data := []byte{}

		// Act
		format, err := DetectImportFormat(data)

		// Assert
		if err == nil {
			t.Fatal("Expected an error for empty input, got nil")
		}
		if format != "" {
			t.Errorf("Expected empty format on error, got %q", format)
		}
	})
}
