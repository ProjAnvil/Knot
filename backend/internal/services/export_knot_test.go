package services

import (
	"testing"
	"time"

	"github.com/ProjAnvil/knot/backend/internal/models"
)

// TestBuildKnotDocument covers building the Knot JSON v1 export document
// from APIs with parameters.
func TestBuildKnotDocument(t *testing.T) {
	// Fixed wall-clock time in a non-UTC zone so the UTC conversion is verified.
	now := time.Date(2026, 11, 10, 20, 30, 0, 0, time.FixedZone("UTC+8", 8*60*60))

	t.Run("GivenGroupedAPIs_whenBuild_thenDocumentMetaGroupsAndParams", func(t *testing.T) {
		// Arrange
		note := "Requires admin role"
		filterID := uint(11)
		dataID := uint(21)
		apis := []APIWithParams{
			{
				API: models.API{
					ID:       1,
					GroupID:  100,
					Name:     "List users",
					Endpoint: "/users",
					Method:   "get",
					Type:     "HTTP",
					Order:    0,
				},
				GroupName:  "Users",
				GroupOrder: 1,
				RequestParameters: []models.Parameter{
					{ID: 11, Name: "filter", Type: "object", Order: 0, ParentID: nil},
					{ID: 12, Name: "name", Type: "string", Required: true, Order: 0, ParentID: &filterID},
				},
				ResponseParameters: []models.Parameter{},
			},
			{
				API: models.API{
					ID:       2,
					GroupID:  100,
					Name:     "Create user",
					Endpoint: "/users",
					Method:   "post",
					Type:     "HTTP",
					Order:    1,
					Note:     &note,
				},
				GroupName:  "Users",
				GroupOrder: 1,
			},
			{
				API: models.API{
					ID:       3,
					GroupID:  200,
					Name:     "Delete item",
					Endpoint: "/items/{id}",
					Method:   "delete",
					Type:     "HTTP",
					Order:    0,
				},
				GroupName:  "Items",
				GroupOrder: 2,
				ResponseParameters: []models.Parameter{
					{ID: 21, Name: "data", Type: "object", Order: 0, ParentID: nil},
					{ID: 22, Name: "item", Type: "object", Order: 0, ParentID: &dataID},
				},
			},
		}

		// Act
		doc := BuildKnotDocument(apis, now)

		// Assert - document metadata
		if doc == nil {
			t.Fatal("Expected a document, got nil")
		}
		if doc.Format != "knot" {
			t.Errorf("Expected format %q, got %q", "knot", doc.Format)
		}
		if doc.Version != 1 {
			t.Errorf("Expected version 1, got %d", doc.Version)
		}
		if want := now.UTC().Format(time.RFC3339); doc.ExportedAt != want {
			t.Errorf("Expected exportedAt %q, got %q", want, doc.ExportedAt)
		}

		// Assert - groups in first-appearance order with names and orders
		if len(doc.Groups) != 2 {
			t.Fatalf("Expected 2 groups, got %d", len(doc.Groups))
		}
		if doc.Groups[0].Name != "Users" || doc.Groups[0].Order != 1 {
			t.Errorf("Expected first group Users/1, got %q/%d", doc.Groups[0].Name, doc.Groups[0].Order)
		}
		if doc.Groups[1].Name != "Items" || doc.Groups[1].Order != 2 {
			t.Errorf("Expected second group Items/2, got %q/%d", doc.Groups[1].Name, doc.Groups[1].Order)
		}

		// Assert - API membership per group
		if len(doc.Groups[0].APIs) != 2 {
			t.Fatalf("Expected 2 APIs in Users group, got %d", len(doc.Groups[0].APIs))
		}
		if len(doc.Groups[1].APIs) != 1 {
			t.Fatalf("Expected 1 API in Items group, got %d", len(doc.Groups[1].APIs))
		}

		// Assert - API fields, upper-cased method, nil note preserved
		first := doc.Groups[0].APIs[0]
		if first.Name != "List users" || first.Endpoint != "/users" || first.Type != "HTTP" || first.Order != 0 {
			t.Errorf("Unexpected API fields: %+v", first)
		}
		if first.Method != "GET" {
			t.Errorf("Expected upper-cased method %q, got %q", "GET", first.Method)
		}
		if first.Note != nil {
			t.Errorf("Expected nil note to stay nil, got %q", *first.Note)
		}
		second := doc.Groups[0].APIs[1]
		if second.Method != "POST" {
			t.Errorf("Expected upper-cased method %q, got %q", "POST", second.Method)
		}
		if second.Note == nil || *second.Note != note {
			t.Errorf("Expected note %q carried over, got %v", note, second.Note)
		}

		// Assert - nested request parameters
		if len(first.Parameters.Request) != 1 {
			t.Fatalf("Expected 1 request root node, got %d", len(first.Parameters.Request))
		}
		filter := first.Parameters.Request[0]
		if filter.Name != "filter" || filter.Type != "object" {
			t.Errorf("Unexpected request root node: %+v", filter)
		}
		if len(filter.Children) != 1 || filter.Children[0].Name != "name" {
			t.Errorf("Expected request[0] children nesting with name, got %+v", filter.Children)
		}
		if !filter.Children[0].Required {
			t.Error("Expected nested name node to be required")
		}
		// Empty response parameters become an empty slice, not nil
		if first.Parameters.Response == nil || len(first.Parameters.Response) != 0 {
			t.Errorf("Expected empty response parameters slice, got %v", first.Parameters.Response)
		}

		// Assert - nested response parameters
		third := doc.Groups[1].APIs[0]
		if third.Method != "DELETE" {
			t.Errorf("Expected upper-cased method %q, got %q", "DELETE", third.Method)
		}
		if len(third.Parameters.Response) != 1 {
			t.Fatalf("Expected 1 response root node, got %d", len(third.Parameters.Response))
		}
		data := third.Parameters.Response[0]
		if data.Name != "data" {
			t.Errorf("Expected response root node data, got %q", data.Name)
		}
		if len(data.Children) != 1 || data.Children[0].Name != "item" {
			t.Errorf("Expected response[0] children nesting with item, got %+v", data.Children)
		}
		// No request parameters yield an empty slice, not nil
		if third.Parameters.Request == nil || len(third.Parameters.Request) != 0 {
			t.Errorf("Expected empty request parameters slice, got %v", third.Parameters.Request)
		}
	})

	t.Run("GivenNoAPIs_whenBuild_thenNoGroups", func(t *testing.T) {
		// Act
		doc := BuildKnotDocument(nil, now)

		// Assert - empty groups are omitted from the document
		if doc == nil {
			t.Fatal("Expected a document, got nil")
		}
		if doc.Groups == nil {
			t.Error("Expected initialized empty groups slice, got nil")
		}
		if len(doc.Groups) != 0 {
			t.Errorf("Expected 0 groups for empty input, got %d", len(doc.Groups))
		}
	})
}
