package query

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/kennethstott/go-doc-go/internal/analytics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDuckDBIntegration_SimpleQuery validates end-to-end query flow
func TestDuckDBIntegration_SimpleQuery(t *testing.T) {
	tmpDir := t.TempDir()

	// Step 1: Write test data using HiveParquetStorage
	hiveConfig := map[string]interface{}{
		"path":    tmpDir,
		"version": "v2.0.0",
	}
	storage, err := analytics.NewHiveParquetStorage(hiveConfig)
	require.NoError(t, err)
	defer storage.Close()

	pageNum := 1
	elements := []analytics.Element{
		{
			ElementID:        "elem1",
			DocID:            "doc1",
			SourceName:       "test_source",
			ElementType:      "paragraph",
			ElementCategory:  "text",
			Content:          "First paragraph content",
			ContentPreview:   "First paragraph content",
			ElementOrder:     1.0,
			DocumentPosition: 1.0,
			PageNumber:       &pageNum,
		},
		{
			ElementID:        "elem2",
			DocID:            "doc1",
			SourceName:       "test_source",
			ElementType:      "paragraph",
			ElementCategory:  "text",
			Content:          "Second paragraph content",
			ContentPreview:   "Second paragraph content",
			ElementOrder:     2.0,
			DocumentPosition: 2.0,
		},
	}

	err = storage.AppendElements(elements)
	require.NoError(t, err)

	// Step 2: Create DuckDB backend
	duckdbConfig := BackendConfig{
		Type:        "duckdb",
		ParquetPath: tmpDir,
		Timeout:     30 * time.Second,
	}

	backend, err := NewDuckDBBackend(duckdbConfig)
	require.NoError(t, err)
	defer backend.Close()

	ctx := context.Background()
	err = backend.Initialize(ctx, duckdbConfig)
	require.NoError(t, err)

	// Step 3: Query all elements
	expr := NewExpression("elements").SelectAll()

	query, err := backend.Translate(expr, TranslateOptions{
		EnablePartitions: true,
	})
	require.NoError(t, err)

	result, err := backend.Execute(ctx, query)
	require.NoError(t, err)

	// Step 4: Verify results
	assert.Equal(t, "duckdb", result.BackendType)
	assert.Equal(t, 2, result.RowCount)
	assert.NotEmpty(t, result.Columns)

	// Verify we got both elements
	foundElem1 := false
	foundElem2 := false
	for _, row := range result.Rows {
		if row["element_id"] == "elem1" {
			foundElem1 = true
			assert.Equal(t, "First paragraph content", row["content"])
		}
		if row["element_id"] == "elem2" {
			foundElem2 = true
			assert.Equal(t, "Second paragraph content", row["content"])
		}
	}
	assert.True(t, foundElem1, "Should find elem1")
	assert.True(t, foundElem2, "Should find elem2")
}

// TestDuckDBIntegration_FilteredQuery validates WHERE clause filtering
func TestDuckDBIntegration_FilteredQuery(t *testing.T) {
	tmpDir := t.TempDir()

	// Write test data with different types
	hiveConfig := map[string]interface{}{
		"path":    tmpDir,
		"version": "v2.0.0",
	}
	storage, err := analytics.NewHiveParquetStorage(hiveConfig)
	require.NoError(t, err)
	defer storage.Close()

	pageNum1 := 1
	pageNum2 := 10
	elements := []analytics.Element{
		{
			ElementID:        "elem1",
			DocID:            "doc1",
			SourceName:       "test_source",
			ElementType:      "paragraph",
			ElementCategory:  "text",
			ContentPreview:   "Paragraph 1",
			ElementOrder:     1.0,
			DocumentPosition: 1.0,
			PageNumber:       &pageNum1,
		},
		{
			ElementID:        "elem2",
			DocID:            "doc1",
			SourceName:       "test_source",
			ElementType:      "header",
			ElementCategory:  "text",
			ContentPreview:   "Header 1",
			ElementOrder:     2.0,
			DocumentPosition: 2.0,
			PageNumber:       &pageNum2,
		},
		{
			ElementID:        "elem3",
			DocID:            "doc1",
			SourceName:       "test_source",
			ElementType:      "paragraph",
			ElementCategory:  "text",
			ContentPreview:   "Paragraph 2",
			ElementOrder:     3.0,
			DocumentPosition: 3.0,
			PageNumber:       &pageNum2,
		},
	}

	err = storage.AppendElements(elements)
	require.NoError(t, err)

	// Query only paragraphs
	duckdbConfig := BackendConfig{
		Type:        "duckdb",
		ParquetPath: tmpDir,
	}

	backend, err := NewDuckDBBackend(duckdbConfig)
	require.NoError(t, err)
	defer backend.Close()

	ctx := context.Background()
	err = backend.Initialize(ctx, duckdbConfig)
	require.NoError(t, err)

	expr := NewExpression("elements").
		SelectAll().
		Filter(NewComparison("element_type", OpEqual, "paragraph"))

	query, err := backend.Translate(expr, TranslateOptions{
		EnablePartitions: true,
	})
	require.NoError(t, err)

	result, err := backend.Execute(ctx, query)
	require.NoError(t, err)

	// Should only get paragraphs
	assert.Equal(t, 2, result.RowCount)
	for _, row := range result.Rows {
		assert.Equal(t, "paragraph", row["element_type"])
	}
}

// TestDuckDBIntegration_ComplexQuery validates complex filtering
func TestDuckDBIntegration_ComplexQuery(t *testing.T) {
	tmpDir := t.TempDir()

	// Write test data
	hiveConfig := map[string]interface{}{
		"path":    tmpDir,
		"version": "v2.0.0",
	}
	storage, err := analytics.NewHiveParquetStorage(hiveConfig)
	require.NoError(t, err)
	defer storage.Close()

	pageNum1 := 1
	pageNum5 := 5
	pageNum10 := 10
	elements := []analytics.Element{
		{
			ElementID:        "elem1",
			DocID:            "doc1",
			SourceName:       "test_source",
			ElementType:      "paragraph",
			ElementCategory:  "text",
			ContentPreview:   "Page 1 paragraph",
			ElementOrder:     1.0,
			DocumentPosition: 1.0,
			PageNumber:       &pageNum1,
		},
		{
			ElementID:        "elem2",
			DocID:            "doc1",
			SourceName:       "test_source",
			ElementType:      "paragraph",
			ElementCategory:  "text",
			ContentPreview:   "Page 5 paragraph",
			ElementOrder:     2.0,
			DocumentPosition: 2.0,
			PageNumber:       &pageNum5,
		},
		{
			ElementID:        "elem3",
			DocID:            "doc1",
			SourceName:       "test_source",
			ElementType:      "header",
			ElementCategory:  "text",
			ContentPreview:   "Page 10 header",
			ElementOrder:     3.0,
			DocumentPosition: 3.0,
			PageNumber:       &pageNum10,
		},
	}

	err = storage.AppendElements(elements)
	require.NoError(t, err)

	// Query: element_type = 'paragraph' AND page_number > 3
	duckdbConfig := BackendConfig{
		Type:        "duckdb",
		ParquetPath: tmpDir,
	}

	backend, err := NewDuckDBBackend(duckdbConfig)
	require.NoError(t, err)
	defer backend.Close()

	ctx := context.Background()
	err = backend.Initialize(ctx, duckdbConfig)
	require.NoError(t, err)

	expr := NewExpression("elements").
		SelectAll().
		Filter(NewAnd(
			NewComparison("element_type", OpEqual, "paragraph"),
			NewComparison("page_number", OpGreaterThan, 3),
		))

	query, err := backend.Translate(expr, TranslateOptions{
		EnablePartitions: true,
	})
	require.NoError(t, err)

	result, err := backend.Execute(ctx, query)
	require.NoError(t, err)

	// Should only get elem2 (paragraph on page 5)
	assert.Equal(t, 1, result.RowCount)
	assert.Equal(t, "elem2", result.Rows[0]["element_id"])
}

// TestDuckDBIntegration_OrderByLimit validates ordering and limiting
func TestDuckDBIntegration_OrderByLimit(t *testing.T) {
	tmpDir := t.TempDir()

	// Write test data
	hiveConfig := map[string]interface{}{
		"path":    tmpDir,
		"version": "v2.0.0",
	}
	storage, err := analytics.NewHiveParquetStorage(hiveConfig)
	require.NoError(t, err)
	defer storage.Close()

	elements := []analytics.Element{
		{
			ElementID:        "elem1",
			DocID:            "doc1",
			SourceName:       "test_source",
			ElementType:      "paragraph",
			ElementCategory:  "text",
			ContentPreview:   "Element 1",
			ElementOrder:     3.0,
			DocumentPosition: 3.0,
		},
		{
			ElementID:        "elem2",
			DocID:            "doc1",
			SourceName:       "test_source",
			ElementType:      "paragraph",
			ElementCategory:  "text",
			ContentPreview:   "Element 2",
			ElementOrder:     1.0,
			DocumentPosition: 1.0,
		},
		{
			ElementID:        "elem3",
			DocID:            "doc1",
			SourceName:       "test_source",
			ElementType:      "paragraph",
			ElementCategory:  "text",
			ContentPreview:   "Element 3",
			ElementOrder:     2.0,
			DocumentPosition: 2.0,
		},
	}

	err = storage.AppendElements(elements)
	require.NoError(t, err)

	// Query with ORDER BY and LIMIT
	duckdbConfig := BackendConfig{
		Type:        "duckdb",
		ParquetPath: tmpDir,
	}

	backend, err := NewDuckDBBackend(duckdbConfig)
	require.NoError(t, err)
	defer backend.Close()

	ctx := context.Background()
	err = backend.Initialize(ctx, duckdbConfig)
	require.NoError(t, err)

	expr := NewExpression("elements").
		SelectAll().
		Order("element_order", false).
		LimitResults(2, 0)

	query, err := backend.Translate(expr, TranslateOptions{
		EnablePartitions: true,
	})
	require.NoError(t, err)

	result, err := backend.Execute(ctx, query)
	require.NoError(t, err)

	// Should get first 2 elements ordered by element_order
	assert.Equal(t, 2, result.RowCount)
	assert.Equal(t, "elem2", result.Rows[0]["element_id"]) // order 1.0
	assert.Equal(t, "elem3", result.Rows[1]["element_id"]) // order 2.0
}

// TestDuckDBIntegration_MultipleTypes validates querying across multiple element types
func TestDuckDBIntegration_MultipleTypes(t *testing.T) {
	tmpDir := t.TempDir()

	// Write elements of multiple types
	hiveConfig := map[string]interface{}{
		"path":    tmpDir,
		"version": "v2.0.0",
	}
	storage, err := analytics.NewHiveParquetStorage(hiveConfig)
	require.NoError(t, err)
	defer storage.Close()

	elements := []analytics.Element{
		{
			ElementID:        "p1",
			DocID:            "doc1",
			SourceName:       "test_source",
			ElementType:      "paragraph",
			ElementCategory:  "text",
			ContentPreview:   "Paragraph",
			ElementOrder:     1.0,
			DocumentPosition: 1.0,
		},
		{
			ElementID:        "h1",
			DocID:            "doc1",
			SourceName:       "test_source",
			ElementType:      "header",
			ElementCategory:  "text",
			ContentPreview:   "Header",
			ElementOrder:     2.0,
			DocumentPosition: 2.0,
		},
		{
			ElementID:        "t1",
			DocID:            "doc1",
			SourceName:       "test_source",
			ElementType:      "table",
			ElementCategory:  "structure",
			ContentPreview:   "Table",
			ElementOrder:     3.0,
			DocumentPosition: 3.0,
		},
	}

	err = storage.AppendElements(elements)
	require.NoError(t, err)

	// Query all types
	duckdbConfig := BackendConfig{
		Type:        "duckdb",
		ParquetPath: tmpDir,
	}

	backend, err := NewDuckDBBackend(duckdbConfig)
	require.NoError(t, err)
	defer backend.Close()

	ctx := context.Background()
	err = backend.Initialize(ctx, duckdbConfig)
	require.NoError(t, err)

	expr := NewExpression("elements").SelectAll()

	query, err := backend.Translate(expr, TranslateOptions{
		EnablePartitions: true,
	})
	require.NoError(t, err)

	result, err := backend.Execute(ctx, query)
	require.NoError(t, err)

	// Should get all 3 elements
	assert.Equal(t, 3, result.RowCount)

	types := make(map[string]bool)
	for _, row := range result.Rows {
		types[row["element_type"].(string)] = true
	}
	assert.True(t, types["paragraph"])
	assert.True(t, types["header"])
	assert.True(t, types["table"])
}

// TestDuckDBIntegration_ExplainQuery validates query plan generation
func TestDuckDBIntegration_ExplainQuery(t *testing.T) {
	tmpDir := t.TempDir()

	// Write some test data
	hiveConfig := map[string]interface{}{
		"path":    tmpDir,
		"version": "v2.0.0",
	}
	storage, err := analytics.NewHiveParquetStorage(hiveConfig)
	require.NoError(t, err)
	defer storage.Close()

	elements := []analytics.Element{
		{
			ElementID:        "elem1",
			DocID:            "doc1",
			SourceName:       "test_source",
			ElementType:      "paragraph",
			ElementCategory:  "text",
			ContentPreview:   "Test",
			ElementOrder:     1.0,
			DocumentPosition: 1.0,
		},
	}

	err = storage.AppendElements(elements)
	require.NoError(t, err)

	// Get query plan
	duckdbConfig := BackendConfig{
		Type:        "duckdb",
		ParquetPath: tmpDir,
	}

	backend, err := NewDuckDBBackend(duckdbConfig)
	require.NoError(t, err)
	defer backend.Close()

	ctx := context.Background()
	err = backend.Initialize(ctx, duckdbConfig)
	require.NoError(t, err)

	expr := NewExpression("elements").
		SelectAll().
		Filter(NewComparison("element_type", OpEqual, "paragraph"))

	query, err := backend.Translate(expr, TranslateOptions{
		EnablePartitions: true,
	})
	require.NoError(t, err)

	plan, err := backend.Explain(ctx, query)
	require.NoError(t, err)

	assert.Equal(t, "duckdb", plan.BackendName)
	assert.NotEmpty(t, plan.RawPlan)
	assert.NotEmpty(t, plan.Steps)

	t.Logf("Query plan:\n%s", plan.RawPlan)
}

// TestDuckDBIntegration_RealParquetPath validates using actual file paths
func TestDuckDBIntegration_RealParquetPath(t *testing.T) {
	tmpDir := t.TempDir()

	// Write some test data
	hiveConfig := map[string]interface{}{
		"path":    tmpDir,
		"version": "v2.0.0",
	}
	storage, err := analytics.NewHiveParquetStorage(hiveConfig)
	require.NoError(t, err)

	elements := []analytics.Element{
		{
			ElementID:        "elem1",
			DocID:            "doc1",
			SourceName:       "test_source",
			ElementType:      "paragraph",
			ElementCategory:  "text",
			ContentPreview:   "Test paragraph",
			ElementOrder:     1.0,
			DocumentPosition: 1.0,
		},
	}

	err = storage.AppendElements(elements)
	require.NoError(t, err)

	// Close storage to flush all data
	err = storage.Close()
	require.NoError(t, err)

	// Verify partition structure was created
	partitionPattern := filepath.Join(tmpDir, "elements", "element_type=paragraph", "version=v2.0.0", "date=*", "source=test_source", "*.parquet")
	matches, err := filepath.Glob(partitionPattern)
	require.NoError(t, err)
	require.Greater(t, len(matches), 0, "Should have created Parquet file in Hive partition structure")

	t.Logf("Created Parquet file: %s", matches[0])

	// Query using DuckDB
	duckdbConfig := BackendConfig{
		Type:        "duckdb",
		ParquetPath: tmpDir,
	}

	backend, err := NewDuckDBBackend(duckdbConfig)
	require.NoError(t, err)
	defer backend.Close()

	ctx := context.Background()
	err = backend.Initialize(ctx, duckdbConfig)
	require.NoError(t, err)

	expr := NewExpression("elements").SelectAll()

	query, err := backend.Translate(expr, TranslateOptions{
		EnablePartitions: true,
	})
	require.NoError(t, err)

	t.Logf("Query: %s", query.QueryString)

	result, err := backend.Execute(ctx, query)
	require.NoError(t, err)

	assert.Equal(t, 1, result.RowCount)
	assert.Equal(t, "elem1", result.Rows[0]["element_id"])
}
