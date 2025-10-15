package query

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/kennethstott/doculyzer-go-conversion/internal/analytics"
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

// TestDuckDBIntegration_SimilaritySearch validates vector similarity search
func TestDuckDBIntegration_SimilaritySearch(t *testing.T) {
	tmpDir := t.TempDir()

	// Step 1: Write test data with embeddings
	hiveConfig := map[string]interface{}{
		"path":    tmpDir,
		"version": "v2.0.0",
	}
	storage, err := analytics.NewHiveParquetStorage(hiveConfig)
	require.NoError(t, err)
	defer storage.Close()

	// Create elements with varying similarity to query vector [1, 0, 0]
	// Embedding format: stored in metadata as JSON array
	elements := []analytics.Element{
		{
			ElementID:        "elem1",
			DocID:            "doc1",
			SourceName:       "test_source",
			ElementType:      "paragraph",
			ElementCategory:  "text",
			Content:          "Very similar document",
			ContentPreview:   "Very similar document",
			ElementOrder:     1.0,
			DocumentPosition: 1.0,
			// Embedding: [1, 0, 0] - identical to query (similarity = 1.0)
			Metadata: map[string]interface{}{
				"embedding": []float64{1.0, 0.0, 0.0},
			},
		},
		{
			ElementID:        "elem2",
			DocID:            "doc1",
			SourceName:       "test_source",
			ElementType:      "paragraph",
			ElementCategory:  "text",
			Content:          "Similar document",
			ContentPreview:   "Similar document",
			ElementOrder:     2.0,
			DocumentPosition: 2.0,
			// Embedding: [0.9, 0.1, 0] - very similar (similarity ~0.9945)
			Metadata: map[string]interface{}{
				"embedding": []float64{0.9, 0.1, 0.0},
			},
		},
		{
			ElementID:        "elem3",
			DocID:            "doc1",
			SourceName:       "test_source",
			ElementType:      "paragraph",
			ElementCategory:  "text",
			Content:          "Somewhat different document",
			ContentPreview:   "Somewhat different document",
			ElementOrder:     3.0,
			DocumentPosition: 3.0,
			// Embedding: [0.5, 0.5, 0] - moderately similar (similarity ~0.707)
			Metadata: map[string]interface{}{
				"embedding": []float64{0.5, 0.5, 0.0},
			},
		},
		{
			ElementID:        "elem4",
			DocID:            "doc1",
			SourceName:       "test_source",
			ElementType:      "paragraph",
			ElementCategory:  "text",
			Content:          "Unrelated document",
			ContentPreview:   "Unrelated document",
			ElementOrder:     4.0,
			DocumentPosition: 4.0,
			// Embedding: [0, 1, 0] - orthogonal (similarity = 0.0)
			Metadata: map[string]interface{}{
				"embedding": []float64{0.0, 1.0, 0.0},
			},
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

	// Step 3: Query with similarity predicate (threshold = 0.8)
	queryVector := []float32{1.0, 0.0, 0.0}
	expr := NewExpression("elements").
		SelectAll().
		Filter(NewSimilarity("metadata", queryVector, 0.8, 0))

	query, err := backend.Translate(expr, TranslateOptions{
		EnablePartitions: true,
	})
	require.NoError(t, err)

	result, err := backend.Execute(ctx, query)
	require.NoError(t, err)

	// Step 4: Apply similarity filter (in-memory)
	// Note: DuckDB returns candidates (WHERE metadata IS NOT NULL)
	// We need to extract embeddings and apply cosine similarity
	filtered, err := ApplySimilarityFilterFromMetadata(result, "metadata", queryVector, 0.8, 0)
	require.NoError(t, err)

	// Step 5: Verify results
	// Should return elem1 (1.0) and elem2 (~0.9945) - both above threshold 0.8
	assert.GreaterOrEqual(t, filtered.RowCount, 2, "Should have at least 2 results with similarity >= 0.8")
	assert.LessOrEqual(t, filtered.RowCount, 3, "Should have at most 3 results")

	// Verify results are sorted by similarity (descending)
	if filtered.RowCount >= 2 {
		sim1 := filtered.Rows[0]["_similarity"].(float64)
		sim2 := filtered.Rows[1]["_similarity"].(float64)
		assert.GreaterOrEqual(t, sim1, sim2, "Results should be sorted by similarity (descending)")
	}

	// First result should be elem1 (highest similarity)
	assert.Equal(t, "elem1", filtered.Rows[0]["element_id"])
	assert.Greater(t, filtered.Rows[0]["_similarity"].(float64), 0.99, "elem1 should have similarity > 0.99")

	t.Logf("Found %d similar documents (threshold 0.8)", filtered.RowCount)
	for i, row := range filtered.Rows {
		t.Logf("  %d. %s: similarity = %.4f", i+1, row["element_id"], row["_similarity"])
	}
}

// TestDuckDBIntegration_SimilaritySearchTopK validates TopK limiting
func TestDuckDBIntegration_SimilaritySearchTopK(t *testing.T) {
	tmpDir := t.TempDir()

	// Write test data with 5 elements with varying similarity
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
			ContentPreview:   "Doc 1",
			ElementOrder:     1.0,
			DocumentPosition: 1.0,
			Metadata:         map[string]interface{}{"embedding": []float64{1.0, 0.0, 0.0}},
		},
		{
			ElementID:        "elem2",
			DocID:            "doc1",
			SourceName:       "test_source",
			ElementType:      "paragraph",
			ElementCategory:  "text",
			ContentPreview:   "Doc 2",
			ElementOrder:     2.0,
			DocumentPosition: 2.0,
			Metadata:         map[string]interface{}{"embedding": []float64{0.9, 0.1, 0.0}},
		},
		{
			ElementID:        "elem3",
			DocID:            "doc1",
			SourceName:       "test_source",
			ElementType:      "paragraph",
			ElementCategory:  "text",
			ContentPreview:   "Doc 3",
			ElementOrder:     3.0,
			DocumentPosition: 3.0,
			Metadata:         map[string]interface{}{"embedding": []float64{0.8, 0.2, 0.0}},
		},
		{
			ElementID:        "elem4",
			DocID:            "doc1",
			SourceName:       "test_source",
			ElementType:      "paragraph",
			ElementCategory:  "text",
			ContentPreview:   "Doc 4",
			ElementOrder:     4.0,
			DocumentPosition: 4.0,
			Metadata:         map[string]interface{}{"embedding": []float64{0.7, 0.3, 0.0}},
		},
		{
			ElementID:        "elem5",
			DocID:            "doc1",
			SourceName:       "test_source",
			ElementType:      "paragraph",
			ElementCategory:  "text",
			ContentPreview:   "Doc 5",
			ElementOrder:     5.0,
			DocumentPosition: 5.0,
			Metadata:         map[string]interface{}{"embedding": []float64{0.6, 0.4, 0.0}},
		},
	}

	err = storage.AppendElements(elements)
	require.NoError(t, err)

	// Create DuckDB backend
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

	// Query with TopK = 3
	queryVector := []float32{1.0, 0.0, 0.0}
	expr := NewExpression("elements").
		SelectAll().
		Filter(NewSimilarity("metadata", queryVector, 0.0, 3)) // No threshold, TopK = 3

	query, err := backend.Translate(expr, TranslateOptions{
		EnablePartitions: true,
	})
	require.NoError(t, err)

	result, err := backend.Execute(ctx, query)
	require.NoError(t, err)

	// Apply similarity filter with TopK
	filtered, err := ApplySimilarityFilterFromMetadata(result, "metadata", queryVector, 0.0, 3)
	require.NoError(t, err)

	// Should return exactly 3 results
	assert.Equal(t, 3, filtered.RowCount, "TopK=3 should return exactly 3 results")

	// Verify top 3 are elem1, elem2, elem3
	assert.Equal(t, "elem1", filtered.Rows[0]["element_id"])
	assert.Equal(t, "elem2", filtered.Rows[1]["element_id"])
	assert.Equal(t, "elem3", filtered.Rows[2]["element_id"])

	t.Logf("TopK=3 results:")
	for i, row := range filtered.Rows {
		t.Logf("  %d. %s: similarity = %.4f", i+1, row["element_id"], row["_similarity"])
	}
}

// ApplySimilarityFilterFromMetadata is a helper for integration tests that extracts embeddings from metadata
func ApplySimilarityFilterFromMetadata(results *QueryResult, metadataField string, queryVector []float32, threshold float64, topK int) (*QueryResult, error) {
	// Create a similarity predicate
	pred := NewSimilarity(metadataField, queryVector, threshold, topK)

	// Extract embeddings from metadata JSON and create rows with direct embedding fields
	rowsWithEmbeddings := make([]Row, 0, len(results.Rows))
	for _, row := range results.Rows {
		// Copy row
		newRow := make(Row)
		for k, v := range row {
			newRow[k] = v
		}

		// Extract embedding from metadata (could be JSON string or map)
		if metadataVal, ok := row[metadataField]; ok {
			var metadata map[string]interface{}

			// Handle different formats
			switch val := metadataVal.(type) {
			case string:
				// Parse JSON string (common when reading from Parquet)
				var parsed map[string]interface{}
				if err := json.Unmarshal([]byte(val), &parsed); err == nil {
					metadata = parsed
				}
			case []byte:
				// Parse JSON bytes
				var parsed map[string]interface{}
				if err := json.Unmarshal(val, &parsed); err == nil {
					metadata = parsed
				}
			case map[string]interface{}:
				// Already a map
				metadata = val
			}

			// Extract embedding from parsed metadata
			if metadata != nil {
				if embedding, ok := metadata["embedding"]; ok {
					// Convert embedding to []float64
					switch emb := embedding.(type) {
					case []float64:
						newRow[metadataField] = emb
					case []interface{}:
						converted := make([]float64, len(emb))
						for i, v := range emb {
							if f, ok := v.(float64); ok {
								converted[i] = f
							}
						}
						newRow[metadataField] = converted
					}
				}
			}
		}

		rowsWithEmbeddings = append(rowsWithEmbeddings, newRow)
	}

	// Create temporary result with extracted embeddings
	tempResult := &QueryResult{
		Rows:          rowsWithEmbeddings,
		RowCount:      len(rowsWithEmbeddings),
		Columns:       results.Columns,
		ExecutionTime: results.ExecutionTime,
		BackendType:   results.BackendType,
		QueryID:       results.QueryID,
		Stats:         results.Stats,
	}

	// Apply similarity filter
	return ApplySimilarityFilter(tempResult, pred)
}
