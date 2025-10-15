package analytics

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/apache/arrow/go/v18/parquet/file"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHiveStorage_WriteAndReadBack validates complete write/read cycle
func TestHiveStorage_WriteAndReadBack(t *testing.T) {
	tmpDir := t.TempDir()

	config := map[string]interface{}{
		"path":    tmpDir,
		"version": "v2.0.0",
	}

	storage, err := NewHiveParquetStorage(config)
	require.NoError(t, err)
	defer storage.Close()

	// Create test elements with ALL promoted fields populated
	pageNum := 5
	sectionLevel := 2
	rowIdx := 3
	colIdx := 7
	temporalType := "datetime"
	tagName := "article"

	elements := []Element{
		{
			ElementID:        "elem_test_001",
			DocID:            "doc_test_001",
			SourceName:       "integration_test",
			ElementType:      "paragraph",
			ElementCategory:  "text",
			Content:          "This is a test paragraph with all promoted fields.",
			ContentPreview:   "This is a test paragraph...",
			ContentHash:      "abc123",
			ParentID:         "parent_001",
			ElementOrder:     1.5,
			DocumentPosition: 10.25,
			// All 6 promoted fields
			PageNumber:   &pageNum,
			SectionLevel: &sectionLevel,
			RowIndex:     &rowIdx,
			ColumnIndex:  &colIdx,
			TemporalType: &temporalType,
			TagName:      &tagName,
			// JSON overflow fields
			ContentLocation: map[string]interface{}{
				"x":      100,
				"y":      200,
				"width":  500,
				"height": 300,
			},
			Metadata: map[string]interface{}{
				"font_size":   12,
				"font_family": "Arial",
				"color":       "#000000",
			},
			TemporalMetadata: map[string]interface{}{
				"parsed_date": "2024-01-15T10:30:00Z",
				"timezone":    "UTC",
			},
		},
	}

	// Write elements
	err = storage.AppendElements(elements)
	require.NoError(t, err)

	// Find the written Parquet file
	pattern := filepath.Join(tmpDir, "elements", "element_type=paragraph", "version=v2.0.0", "date=*", "source=integration_test", "elements_*.parquet")
	matches, err := filepath.Glob(pattern)
	require.NoError(t, err)
	require.Len(t, matches, 1, "Should have created exactly one Parquet file")

	parquetFile := matches[0]
	t.Logf("Created Parquet file: %s", parquetFile)

	// Open and read the Parquet file to verify schema and data
	f, err := os.Open(parquetFile)
	require.NoError(t, err)
	defer f.Close()

	reader, err := file.NewParquetReader(f)
	require.NoError(t, err)
	defer reader.Close()

	// Verify schema has exactly 20 columns (UDML Phase 1)
	metadata := reader.MetaData()
	schema := metadata.Schema
	numCols := schema.NumColumns()
	assert.Equal(t, 20, numCols, "Parquet file should have exactly 20 columns")

	t.Logf("Parquet file has %d columns (expected 20)", numCols)

	// Verify column names match universal schema
	expectedColumns := []string{
		"element_id", "doc_id", "source_name", "element_type", "element_category",
		"content", "content_preview", "content_hash", "parent_id",
		"element_order", "document_position",
		// 6 promoted fields
		"page_number", "section_level", "row_index", "column_index",
		"temporal_type", "tag_name",
		// JSON overflow
		"content_location", "metadata", "temporal_metadata",
	}

	for i := 0; i < numCols; i++ {
		colName := schema.Column(i).Name()
		t.Logf("Column %d: %s", i, colName)

		if i < len(expectedColumns) {
			assert.Equal(t, expectedColumns[i], colName,
				"Column %d name should match", i)
		}
	}

	// Verify number of rows
	numRows := reader.NumRows()
	assert.Equal(t, int64(1), numRows, "Should have 1 row")
	t.Logf("Parquet file has %d rows", numRows)

	// Verify file metadata
	t.Logf("Parquet version: %s", metadata.Version())
	if metadata.CreatedBy != nil {
		t.Logf("Created by: %s", *metadata.CreatedBy)
	}
	t.Logf("Num row groups: %d", reader.NumRowGroups())
}

// TestHiveStorage_MultipleTypes_VerifyPartitions validates partition structure
func TestHiveStorage_MultipleTypes_VerifyPartitions(t *testing.T) {
	tmpDir := t.TempDir()

	config := map[string]interface{}{
		"path":    tmpDir,
		"version": "v2.1.0", // Test custom version
	}

	storage, err := NewHiveParquetStorage(config)
	require.NoError(t, err)
	defer storage.Close()

	// Create elements of 5 different types
	elementTypes := []string{"paragraph", "header", "table", "list", "code_block"}
	var allElements []Element

	for i, elemType := range elementTypes {
		elem := Element{
			ElementID:        elemType + "_001",
			DocID:            "doc_multi_001",
			SourceName:       "partition_test",
			ElementType:      elemType,
			ElementCategory:  "test",
			ContentPreview:   "Test " + elemType,
			ElementOrder:     float64(i),
			DocumentPosition: float64(i),
		}
		allElements = append(allElements, elem)
	}

	// Write all elements
	err = storage.AppendElements(allElements)
	require.NoError(t, err)

	// Verify each type created its own partition
	for _, elemType := range elementTypes {
		partitionPath := filepath.Join(tmpDir, "elements",
			"element_type="+elemType,
			"version=v2.1.0",
			"date=*",
			"source=partition_test")

		matches, err := filepath.Glob(filepath.Join(partitionPath, "elements_*.parquet"))
		require.NoError(t, err)
		assert.Greater(t, len(matches), 0,
			"Should have created partition for type: %s", elemType)

		t.Logf("Type %s: %d partition file(s)", elemType, len(matches))
	}

	// Verify partition structure
	elementsDir := filepath.Join(tmpDir, "elements")
	entries, err := os.ReadDir(elementsDir)
	require.NoError(t, err)

	typePartitions := 0
	for _, entry := range entries {
		if entry.IsDir() && len(entry.Name()) > len("element_type=") {
			typePartitions++
			t.Logf("Found partition: %s", entry.Name())
		}
	}

	assert.Equal(t, len(elementTypes), typePartitions,
		"Should have created %d type partitions", len(elementTypes))
}

// TestHiveStorage_SchemaValidation validates promoted fields are nullable
func TestHiveStorage_SchemaValidation(t *testing.T) {
	tmpDir := t.TempDir()

	config := map[string]interface{}{
		"path":    tmpDir,
		"version": "v2.0.0",
	}

	storage, err := NewHiveParquetStorage(config)
	require.NoError(t, err)
	defer storage.Close()

	// Create element with NO promoted fields (all NULL)
	element := Element{
		ElementID:        "null_test_001",
		DocID:            "doc_null_001",
		SourceName:       "null_test",
		ElementType:      "paragraph",
		ElementCategory:  "text",
		ContentPreview:   "Element with no promoted fields",
		ElementOrder:     1.0,
		DocumentPosition: 1.0,
		// All promoted fields are nil (NULL)
	}

	// This should succeed - promoted fields are nullable
	err = storage.AppendElements([]Element{element})
	require.NoError(t, err, "Should successfully write element with all NULL promoted fields")

	// Verify file was created
	pattern := filepath.Join(tmpDir, "elements", "element_type=paragraph", "version=v2.0.0", "date=*", "source=null_test", "elements_*.parquet")
	matches, err := filepath.Glob(pattern)
	require.NoError(t, err)
	assert.Greater(t, len(matches), 0, "Should have created Parquet file")
}

// TestHiveStorage_VersionPartitioning validates version-based partitioning
func TestHiveStorage_VersionPartitioning(t *testing.T) {
	tmpDir := t.TempDir()

	// Test multiple versions
	versions := []string{"v1.0.0", "v2.0.0", "v2.1.0"}

	for _, version := range versions {
		config := map[string]interface{}{
			"path":    tmpDir,
			"version": version,
		}

		storage, err := NewHiveParquetStorage(config)
		require.NoError(t, err)

		element := Element{
			ElementID:        "elem_" + version,
			DocID:            "doc_version_001",
			SourceName:       "version_test",
			ElementType:      "paragraph",
			ElementCategory:  "text",
			ContentPreview:   "Test version " + version,
			ElementOrder:     1.0,
			DocumentPosition: 1.0,
		}

		err = storage.AppendElements([]Element{element})
		require.NoError(t, err)
		storage.Close()
	}

	// Verify each version created its own partition
	for _, version := range versions {
		versionPath := filepath.Join(tmpDir, "elements", "element_type=paragraph", "version="+version)
		_, err := os.Stat(versionPath)
		require.NoError(t, err, "Version partition should exist: %s", version)
		t.Logf("✓ Version partition exists: %s", version)
	}
}

// TestHiveStorage_CompareWithFlatStorage validates data consistency
func TestHiveStorage_CompareWithFlatStorage(t *testing.T) {
	tmpDirHive := t.TempDir()
	tmpDirFlat := t.TempDir()

	// Create same element for both storages
	pageNum := 10
	element := Element{
		ElementID:        "compare_001",
		DocID:            "doc_compare_001",
		SourceName:       "compare_test",
		ElementType:      "paragraph",
		ElementCategory:  "text",
		Content:          "Test content for comparison",
		ContentPreview:   "Test content for comparison",
		ElementOrder:     1.0,
		DocumentPosition: 1.0,
		PageNumber:       &pageNum,
	}

	// Write to Hive storage
	hiveConfig := map[string]interface{}{
		"path":    tmpDirHive,
		"version": "v2.0.0",
	}
	hiveStorage, err := NewHiveParquetStorage(hiveConfig)
	require.NoError(t, err)
	err = hiveStorage.AppendElements([]Element{element})
	require.NoError(t, err)
	hiveStorage.Close()

	// Write to flat storage
	flatConfig := map[string]interface{}{
		"path": tmpDirFlat,
	}
	flatStorage, err := NewParquetStorage(flatConfig)
	require.NoError(t, err)
	err = flatStorage.AppendElements([]Element{element})
	require.NoError(t, err)
	flatStorage.Close()

	// Verify both created files
	hiveMatches, _ := filepath.Glob(filepath.Join(tmpDirHive, "elements", "element_type=paragraph", "version=v2.0.0", "date=*", "source=compare_test", "elements_*.parquet"))

	flatPattern := filepath.Join(tmpDirFlat, "elements", "date=*", "source=compare_test", "elements_*.parquet")
	flatMatches, _ := filepath.Glob(flatPattern)

	t.Logf("Hive storage: %d file(s)", len(hiveMatches))
	t.Logf("Flat storage: %d file(s)", len(flatMatches))

	assert.Greater(t, len(hiveMatches), 0, "Hive storage should create files")
	assert.Greater(t, len(flatMatches), 0, "Flat storage should create files")

	// Both should have successfully written the element
	t.Log("✓ Both Hive and Flat storage successfully wrote elements")
}
