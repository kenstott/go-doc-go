package query

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/kennethstott/doculyzer-go-conversion/internal/analytics"
)

// TestJSONPath_IntegrationWithDuckDB tests JSONPath queries against real Parquet files
func TestJSONPath_IntegrationWithDuckDB(t *testing.T) {
	// Create temp directory for Parquet files
	tmpDir, err := os.MkdirTemp("", "jsonpath_integration_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Write test data with JSON overflow fields
	if err := writeTestDataWithJSON(tmpDir); err != nil {
		t.Fatalf("Failed to write test data: %v", err)
	}

	// Initialize DuckDB backend
	backend, err := initTestBackend(tmpDir)
	if err != nil {
		t.Fatalf("Failed to initialize backend: %v", err)
	}
	defer backend.Close()

	ctx := context.Background()

	t.Run("QueryMetadataFontSize", func(t *testing.T) {
		// Query: metadata.font_size > 12
		pred := &Predicate{
			Type:     PredicateJSONPath,
			Field:    "metadata",
			JSONPath: "$.font_size",
			Operator: OpGreaterThan,
			Value:    float64(12),
		}

		expr := NewExpression("elements").
			SelectFields("element_id", "element_type", "metadata").
			Filter(pred)

		query, err := backend.Translate(expr, TranslateOptions{EnablePartitions: true})
		if err != nil {
			t.Fatalf("Failed to translate: %v", err)
		}

		result, err := backend.Execute(ctx, query)
		if err != nil {
			t.Fatalf("Failed to execute: %v", err)
		}

		// Should find elements with font_size > 12
		if result.RowCount == 0 {
			t.Error("Expected to find elements with font_size > 12")
		}

		// Verify metadata field contains expected values
		for _, row := range result.Rows {
			metadataJSON, ok := row["metadata"].(string)
			if !ok {
				continue
			}

			var metadata map[string]interface{}
			if err := json.Unmarshal([]byte(metadataJSON), &metadata); err != nil {
				continue
			}

			if fontSize, ok := metadata["font_size"].(float64); ok {
				if fontSize <= 12 {
					t.Errorf("Font size should be > 12, got %.1f", fontSize)
				}
			}
		}
	})

	t.Run("QueryContentLocationX", func(t *testing.T) {
		// Query: content_location.x >= 100
		pred := &Predicate{
			Type:     PredicateJSONPath,
			Field:    "content_location",
			JSONPath: "$.x",
			Operator: OpGreaterThanOrEqual,
			Value:    float64(100),
		}

		expr := NewExpression("elements").
			SelectFields("element_id", "content_location").
			Filter(pred)

		query, err := backend.Translate(expr, TranslateOptions{EnablePartitions: true})
		if err != nil {
			t.Fatalf("Failed to translate: %v", err)
		}

		result, err := backend.Execute(ctx, query)
		if err != nil {
			t.Fatalf("Failed to execute: %v", err)
		}

		if result.RowCount == 0 {
			t.Error("Expected to find elements with x >= 100")
		}

		// Verify content_location values
		for _, row := range result.Rows {
			locationJSON, ok := row["content_location"].(string)
			if !ok {
				continue
			}

			var location map[string]interface{}
			if err := json.Unmarshal([]byte(locationJSON), &location); err != nil {
				continue
			}

			if x, ok := location["x"].(float64); ok {
				if x < 100 {
					t.Errorf("X coordinate should be >= 100, got %.1f", x)
				}
			}
		}
	})

	t.Run("QueryNestedMetadata", func(t *testing.T) {
		// Query: metadata.styles.bold = true
		pred := &Predicate{
			Type:     PredicateJSONPath,
			Field:    "metadata",
			JSONPath: "$.styles.bold",
			Operator: OpEqual,
			Value:    true,
		}

		expr := NewExpression("elements").
			SelectFields("element_id", "element_type", "metadata").
			Filter(pred)

		query, err := backend.Translate(expr, TranslateOptions{EnablePartitions: true})
		if err != nil {
			t.Fatalf("Failed to translate: %v", err)
		}

		result, err := backend.Execute(ctx, query)
		if err != nil {
			t.Fatalf("Failed to execute: %v", err)
		}

		// Should find elements with bold style
		if result.RowCount == 0 {
			t.Error("Expected to find elements with bold=true")
		}

		// Verify nested metadata
		for _, row := range result.Rows {
			metadataJSON, ok := row["metadata"].(string)
			if !ok {
				continue
			}

			var metadata map[string]interface{}
			if err := json.Unmarshal([]byte(metadataJSON), &metadata); err != nil {
				continue
			}

			styles, ok := metadata["styles"].(map[string]interface{})
			if !ok {
				t.Error("Expected styles object in metadata")
				continue
			}

			if bold, ok := styles["bold"].(bool); !ok || !bold {
				t.Error("Expected bold=true in styles")
			}
		}
	})

	t.Run("QueryContentLocationBounds", func(t *testing.T) {
		// Complex query: x >= 50 AND x <= 150 AND y >= 100
		pred := NewAnd(
			NewAnd(
				&Predicate{
					Type:     PredicateJSONPath,
					Field:    "content_location",
					JSONPath: "$.x",
					Operator: OpGreaterThanOrEqual,
					Value:    float64(50),
				},
				&Predicate{
					Type:     PredicateJSONPath,
					Field:    "content_location",
					JSONPath: "$.x",
					Operator: OpLessThanOrEqual,
					Value:    float64(150),
				},
			),
			&Predicate{
				Type:     PredicateJSONPath,
				Field:    "content_location",
				JSONPath: "$.y",
				Operator: OpGreaterThanOrEqual,
				Value:    float64(100),
			},
		)

		expr := NewExpression("elements").
			SelectFields("element_id", "content_location").
			Filter(pred)

		query, err := backend.Translate(expr, TranslateOptions{EnablePartitions: true})
		if err != nil {
			t.Fatalf("Failed to translate: %v", err)
		}

		result, err := backend.Execute(ctx, query)
		if err != nil {
			t.Fatalf("Failed to execute: %v", err)
		}

		// Verify bounds
		for _, row := range result.Rows {
			locationJSON, ok := row["content_location"].(string)
			if !ok {
				continue
			}

			var location map[string]interface{}
			if err := json.Unmarshal([]byte(locationJSON), &location); err != nil {
				continue
			}

			x, xOk := location["x"].(float64)
			y, yOk := location["y"].(float64)

			if xOk && (x < 50 || x > 150) {
				t.Errorf("X should be in [50, 150], got %.1f", x)
			}
			if yOk && y < 100 {
				t.Errorf("Y should be >= 100, got %.1f", y)
			}
		}
	})

	t.Run("CombineJSONPathWithRegularFilters", func(t *testing.T) {
		// Query: element_type = 'paragraph' AND metadata.font_size > 10
		pred := NewAnd(
			NewComparison("element_type", OpEqual, "paragraph"),
			&Predicate{
				Type:     PredicateJSONPath,
				Field:    "metadata",
				JSONPath: "$.font_size",
				Operator: OpGreaterThan,
				Value:    float64(10),
			},
		)

		expr := NewExpression("elements").
			SelectFields("element_id", "element_type", "metadata").
			Filter(pred)

		query, err := backend.Translate(expr, TranslateOptions{EnablePartitions: true})
		if err != nil {
			t.Fatalf("Failed to translate: %v", err)
		}

		result, err := backend.Execute(ctx, query)
		if err != nil {
			t.Fatalf("Failed to execute: %v", err)
		}

		// All results must be paragraphs
		for _, row := range result.Rows {
			if row["element_type"] != "paragraph" {
				t.Errorf("Expected paragraph, got %v", row["element_type"])
			}

			// And have font_size > 10
			metadataJSON, ok := row["metadata"].(string)
			if !ok {
				continue
			}

			var metadata map[string]interface{}
			if err := json.Unmarshal([]byte(metadataJSON), &metadata); err != nil {
				continue
			}

			if fontSize, ok := metadata["font_size"].(float64); ok {
				if fontSize <= 10 {
					t.Errorf("Font size should be > 10, got %.1f", fontSize)
				}
			}
		}
	})
}

// writeTestDataWithJSON writes test UDML elements with JSON overflow fields
func writeTestDataWithJSON(tmpDir string) error {
	hiveConfig := map[string]interface{}{
		"path":    tmpDir,
		"version": "v2.0.0",
	}

	storage, err := analytics.NewHiveParquetStorage(hiveConfig)
	if err != nil {
		return err
	}
	defer storage.Close()

	pageNum1 := 1
	pageNum2 := 2

	// Create test elements with various JSON metadata and content_location
	elements := []analytics.Element{
		// Paragraph with small font
		{
			ElementID:       "para1",
			DocID:           "doc1",
			SourceName:      "test_doc",
			ElementType:     "paragraph",
			ElementCategory: "text",
			Content:         "Small font paragraph",
			ContentPreview:  "Small font paragraph",
			ElementOrder:    1.0,
			PageNumber:      &pageNum1,
			Metadata: map[string]interface{}{
				"font_size":   10,
				"font_family": "Arial",
				"styles": map[string]interface{}{
					"bold": false,
				},
			},
			ContentLocation: map[string]interface{}{
				"x":      50,
				"y":      100,
				"width":  400,
				"height": 20,
			},
		},
		// Paragraph with large font
		{
			ElementID:       "para2",
			DocID:           "doc1",
			SourceName:      "test_doc",
			ElementType:     "paragraph",
			ElementCategory: "text",
			Content:         "Large font paragraph",
			ContentPreview:  "Large font paragraph",
			ElementOrder:    2.0,
			PageNumber:      &pageNum1,
			Metadata: map[string]interface{}{
				"font_size":   14,
				"font_family": "Helvetica",
				"styles": map[string]interface{}{
					"bold": true,
				},
			},
			ContentLocation: map[string]interface{}{
				"x":      100,
				"y":      150,
				"width":  500,
				"height": 25,
			},
		},
		// Header with extra large font
		{
			ElementID:       "header1",
			DocID:           "doc1",
			SourceName:      "test_doc",
			ElementType:     "header",
			ElementCategory: "text",
			Content:         "Main Header",
			ContentPreview:  "Main Header",
			ElementOrder:    3.0,
			PageNumber:      &pageNum1,
			Metadata: map[string]interface{}{
				"font_size":   18,
				"font_family": "Times",
				"styles": map[string]interface{}{
					"bold":   true,
					"italic": false,
				},
			},
			ContentLocation: map[string]interface{}{
				"x":      75,
				"y":      50,
				"width":  600,
				"height": 30,
			},
		},
		// Table with positioning
		{
			ElementID:       "table1",
			DocID:           "doc1",
			SourceName:      "test_doc",
			ElementType:     "table",
			ElementCategory: "structure",
			Content:         "Data Table",
			ContentPreview:  "Data Table",
			ElementOrder:    4.0,
			PageNumber:      &pageNum2,
			Metadata: map[string]interface{}{
				"rows":    5,
				"columns": 3,
				"styles": map[string]interface{}{
					"bordered": true,
				},
			},
			ContentLocation: map[string]interface{}{
				"x":      200,
				"y":      300,
				"width":  700,
				"height": 200,
			},
		},
		// Code block with special formatting
		{
			ElementID:       "code1",
			DocID:           "doc1",
			SourceName:      "test_doc",
			ElementType:     "code_block",
			ElementCategory: "code",
			Content:         "func main() { fmt.Println(\"Hello\") }",
			ContentPreview:  "func main() { fmt.Println(\"Hello\") }",
			ElementOrder:    5.0,
			PageNumber:      &pageNum2,
			Metadata: map[string]interface{}{
				"language":  "go",
				"font_size": 11,
				"styles": map[string]interface{}{
					"monospace": true,
				},
			},
			ContentLocation: map[string]interface{}{
				"x":      120,
				"y":      200,
				"width":  600,
				"height": 80,
			},
		},
	}

	return storage.AppendElements(elements)
}

// initTestBackend initializes a DuckDB backend for testing
func initTestBackend(tmpDir string) (QueryBackend, error) {
	config := BackendConfig{
		Type:        "duckdb",
		ParquetPath: tmpDir,
		Timeout:     30 * time.Second,
	}

	backend, err := GetOrCreateBackend(config)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	if err := backend.Initialize(ctx, config); err != nil {
		return nil, err
	}

	return backend, nil
}
