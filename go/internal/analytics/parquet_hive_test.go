package analytics

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewHiveParquetStorage validates Hive storage initialization
func TestNewHiveParquetStorage(t *testing.T) {
	tmpDir := t.TempDir()

	config := map[string]interface{}{
		"path":    tmpDir,
		"version": "v2.0.0",
	}

	storage, err := NewHiveParquetStorage(config)
	require.NoError(t, err)
	require.NotNil(t, storage)

	assert.Equal(t, tmpDir, storage.basePath)
	assert.Equal(t, "v2.0.0", storage.version)
	assert.NotNil(t, storage.schemaRegistry)
	assert.NotNil(t, storage.allocator)

	err = storage.Close()
	assert.NoError(t, err)
}

// TestHiveParquetStorage_AppendElements validates Hive-partitioned element writing
func TestHiveParquetStorage_AppendElements(t *testing.T) {
	tmpDir := t.TempDir()

	config := map[string]interface{}{
		"path":    tmpDir,
		"version": "v2.0.0",
	}

	storage, err := NewHiveParquetStorage(config)
	require.NoError(t, err)
	defer storage.Close()

	// Create test elements with different types and promoted fields
	pageNum1 := 1
	sectionLevel1 := 1
	rowIdx1 := 0
	colIdx1 := 0
	temporalType1 := "date"
	tagName1 := "h1"

	elements := []Element{
		{
			ElementID:       "elem1",
			DocID:           "doc1",
			SourceName:      "test_source",
			ElementType:     "paragraph",
			ElementCategory: "text",
			Content:         "Test paragraph",
			ContentPreview:  "Test paragraph",
			ElementOrder:    1.0,
			DocumentPosition: 1.0,
		},
		{
			ElementID:        "elem2",
			DocID:            "doc1",
			SourceName:       "test_source",
			ElementType:      "header",
			ElementCategory:  "text",
			Content:          "Test header",
			ContentPreview:   "Test header",
			ElementOrder:     2.0,
			DocumentPosition: 2.0,
			PageNumber:       &pageNum1,
			SectionLevel:     &sectionLevel1,
			TagName:          &tagName1,
		},
		{
			ElementID:        "elem3",
			DocID:            "doc1",
			SourceName:       "test_source",
			ElementType:      "table_cell",
			ElementCategory:  "structure",
			Content:          "Test cell",
			ContentPreview:   "Test cell",
			ElementOrder:     3.0,
			DocumentPosition: 3.0,
			RowIndex:         &rowIdx1,
			ColumnIndex:      &colIdx1,
		},
		{
			ElementID:        "elem4",
			DocID:            "doc1",
			SourceName:       "test_source",
			ElementType:      "paragraph",
			ElementCategory:  "text",
			Content:          "Date: 2024-01-15",
			ContentPreview:   "Date: 2024-01-15",
			ElementOrder:     4.0,
			DocumentPosition: 4.0,
			TemporalType:     &temporalType1,
		},
	}

	// Write elements
	err = storage.AppendElements(elements)
	require.NoError(t, err)

	// Verify Hive partition structure was created
	// Elements should be in: elements/element_type=X/version=v2.0.0/date=YYYY-MM-DD/source=test_source/

	// Check paragraph partition
	paragraphPattern := filepath.Join(tmpDir, "elements", "element_type=paragraph", "version=v2.0.0", "date=*", "source=test_source", "elements_*.parquet")
	matches, err := filepath.Glob(paragraphPattern)
	require.NoError(t, err)
	assert.Greater(t, len(matches), 0, "Should have created paragraph partition files")

	// Check header partition
	headerPattern := filepath.Join(tmpDir, "elements", "element_type=header", "version=v2.0.0", "date=*", "source=test_source", "elements_*.parquet")
	matches, err = filepath.Glob(headerPattern)
	require.NoError(t, err)
	assert.Greater(t, len(matches), 0, "Should have created header partition files")

	// Check table_cell partition
	tableCellPattern := filepath.Join(tmpDir, "elements", "element_type=table_cell", "version=v2.0.0", "date=*", "source=test_source", "elements_*.parquet")
	matches, err = filepath.Glob(tableCellPattern)
	require.NoError(t, err)
	assert.Greater(t, len(matches), 0, "Should have created table_cell partition files")
}

// TestHiveParquetStorage_AppendDocuments validates document writing
func TestHiveParquetStorage_AppendDocuments(t *testing.T) {
	tmpDir := t.TempDir()

	config := map[string]interface{}{
		"path":    tmpDir,
		"version": "v2.0.0",
	}

	storage, err := NewHiveParquetStorage(config)
	require.NoError(t, err)
	defer storage.Close()

	// Create test documents
	documents := []Document{
		{
			DocID:             "doc1",
			SourceName:        "test_source",
			Title:             "Test Document",
			URL:               "http://example.com/doc1",
			ContentType:       "text/html",
			ElementCount:      10,
			RelationshipCount: 5,
		},
	}

	// Write documents
	err = storage.AppendDocuments(documents)
	require.NoError(t, err)

	// Verify documents were written with date/source partitioning
	pattern := filepath.Join(tmpDir, "documents", "date=*", "source=test_source", "documents_*.parquet")
	matches, err := filepath.Glob(pattern)
	require.NoError(t, err)
	assert.Greater(t, len(matches), 0, "Should have created document partition files")
}

// TestHiveParquetStorage_AppendRelationships validates relationship writing
func TestHiveParquetStorage_AppendRelationships(t *testing.T) {
	tmpDir := t.TempDir()

	config := map[string]interface{}{
		"path":    tmpDir,
		"version": "v2.0.0",
	}

	storage, err := NewHiveParquetStorage(config)
	require.NoError(t, err)
	defer storage.Close()

	// Create test relationships
	relationships := []Relationship{
		{
			SourceElementID:  "elem1",
			TargetElementID:  "elem2",
			RelationshipType: "parent_of",
			DocID:            "doc1",
			SourceName:       "test_source",
		},
	}

	// Write relationships
	err = storage.AppendRelationships(relationships)
	require.NoError(t, err)

	// Verify relationships were written
	pattern := filepath.Join(tmpDir, "relationships", "date=*", "source=test_source", "relationships_*.parquet")
	matches, err := filepath.Glob(pattern)
	require.NoError(t, err)
	assert.Greater(t, len(matches), 0, "Should have created relationship partition files")
}

// TestHivePartitionPath validates Hive partition path generation
func TestHivePartitionPath(t *testing.T) {
	tmpDir := t.TempDir()

	config := map[string]interface{}{
		"path":    tmpDir,
		"version": "v2.0.0",
	}

	storage, err := NewHiveParquetStorage(config)
	require.NoError(t, err)
	defer storage.Close()

	// Test Hive partition path for elements
	path := storage.getHivePartitionPath("paragraph", "2024-01-15", "test_source")

	expectedPath := filepath.Join(tmpDir, "elements", "element_type=paragraph", "version=v2.0.0", "date=2024-01-15", "source=test_source")
	assert.Equal(t, expectedPath, path)
}

// TestFactoryHiveBackend validates factory creates Hive storage
func TestFactoryHiveBackend(t *testing.T) {
	tmpDir := t.TempDir()

	config := map[string]interface{}{
		"type":    "hive",
		"path":    tmpDir,
		"version": "v2.0.0",
	}

	storage, err := NewStorage(config)
	require.NoError(t, err)
	require.NotNil(t, storage)

	// Verify it's a HiveParquetStorage instance
	hiveStorage, ok := storage.(*HiveParquetStorage)
	assert.True(t, ok, "Should be HiveParquetStorage instance")
	assert.Equal(t, "v2.0.0", hiveStorage.version)

	err = storage.Close()
	assert.NoError(t, err)
}

// TestHiveStorage_DefaultVersion validates default version assignment
func TestHiveStorage_DefaultVersion(t *testing.T) {
	tmpDir := t.TempDir()

	config := map[string]interface{}{
		"path": tmpDir,
		// No version specified - should default to v2.0.0
	}

	storage, err := NewHiveParquetStorage(config)
	require.NoError(t, err)
	defer storage.Close()

	assert.Equal(t, "v2.0.0", storage.version)
}

// TestHiveStorage_MultipleElementTypes validates grouping by element type
func TestHiveStorage_MultipleElementTypes(t *testing.T) {
	tmpDir := t.TempDir()

	config := map[string]interface{}{
		"path":    tmpDir,
		"version": "v2.0.0",
	}

	storage, err := NewHiveParquetStorage(config)
	require.NoError(t, err)
	defer storage.Close()

	// Create elements of multiple types
	elements := []Element{
		{
			ElementID:        "p1",
			DocID:            "doc1",
			SourceName:       "source1",
			ElementType:      "paragraph",
			ElementCategory:  "text",
			ContentPreview:   "Paragraph 1",
			ElementOrder:     1.0,
			DocumentPosition: 1.0,
		},
		{
			ElementID:        "p2",
			DocID:            "doc1",
			SourceName:       "source1",
			ElementType:      "paragraph",
			ElementCategory:  "text",
			ContentPreview:   "Paragraph 2",
			ElementOrder:     2.0,
			DocumentPosition: 2.0,
		},
		{
			ElementID:        "h1",
			DocID:            "doc1",
			SourceName:       "source1",
			ElementType:      "header",
			ElementCategory:  "text",
			ContentPreview:   "Header 1",
			ElementOrder:     3.0,
			DocumentPosition: 3.0,
		},
		{
			ElementID:        "t1",
			DocID:            "doc1",
			SourceName:       "source1",
			ElementType:      "table",
			ElementCategory:  "structure",
			ContentPreview:   "Table 1",
			ElementOrder:     4.0,
			DocumentPosition: 4.0,
		},
	}

	err = storage.AppendElements(elements)
	require.NoError(t, err)

	// Verify separate partitions were created for each type
	elementTypes := []string{"paragraph", "header", "table"}
	for _, elemType := range elementTypes {
		pattern := filepath.Join(tmpDir, "elements", fmt.Sprintf("element_type=%s", elemType), "version=v2.0.0", "date=*", "source=source1", "elements_*.parquet")
		matches, err := filepath.Glob(pattern)
		require.NoError(t, err)
		assert.Greater(t, len(matches), 0, "Should have created %s partition", elemType)
	}
}
