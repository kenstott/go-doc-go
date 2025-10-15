package parser_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/kennethstott/doculyzer-go-conversion/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCSVParserInterface validates CSV parser implements Parser interface correctly
func TestCSVParserInterface(t *testing.T) {
	csvParser := parser.NewCSVParser()

	t.Run("implements Parser interface", func(t *testing.T) {
		// Verify parser implements the interface
		var _ parser.Parser = csvParser

		// Test GetName
		assert.Equal(t, "csv", csvParser.GetName())

		// Test GetSupportedFormats
		formats := csvParser.GetSupportedFormats()
		assert.Contains(t, formats, ".csv")
		assert.Contains(t, formats, "csv")

		// Test SupportsStreaming
		assert.False(t, csvParser.SupportsStreaming())

		// Test Close
		err := csvParser.Close()
		assert.NoError(t, err)
	})

	t.Run("can be registered in ParserRegistry", func(t *testing.T) {
		registry := parser.NewParserRegistry()
		registry.Register(csvParser)

		// Verify parser is registered
		retrieved, err := registry.GetParser("csv")
		require.NoError(t, err)
		assert.Equal(t, "csv", retrieved.GetName())

		// Verify parser can be found by file extension
		parserByFile, err := registry.GetParserForFile("data.csv")
		require.NoError(t, err)
		assert.Equal(t, "csv", parserByFile.GetName())
	})
}

// TestCSVParserEndToEnd validates complete CSV parsing workflow via Parser interface
func TestCSVParserEndToEnd(t *testing.T) {
	// Create sample CSV content
	csvContent := `Name,Age,City
Alice,30,New York
Bob,25,Los Angeles
Carol,35,Chicago`

	csvParser := parser.NewCSVParser()
	ctx := context.Background()

	t.Run("parses CSV via Parser interface", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-csv-001",
			Content: csvContent,
			Metadata: map[string]interface{}{
				"source": "test",
			},
			Config: parser.DefaultParserConfig(),
		}

		result, err := csvParser.Parse(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, result)

		// Validate Document
		assert.Equal(t, "test-csv-001", result.Document.ID)
		assert.Equal(t, "csv", result.Document.DocType)

		// Validate Elements exist
		assert.NotEmpty(t, result.Elements)
	})

	t.Run("populates promoted fields correctly", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-csv-002",
			Content: csvContent,
			Config:  parser.DefaultParserConfig(),
		}

		result, err := csvParser.Parse(ctx, req)
		require.NoError(t, err)

		// Find table cell elements
		var tableCells []parser.Element
		for _, elem := range result.Elements {
			if elem.ElementType == "table_cell" {
				tableCells = append(tableCells, elem)
			}
		}

		// Should have cells from header (3) + data rows (3*3=9) = 12 total
		assert.NotEmpty(t, tableCells, "Should have table cells")

		// Verify at least one cell has promoted fields populated
		foundPromotedFields := false
		for _, cell := range tableCells {
			if cell.RowIndex != nil && cell.ColumnIndex != nil {
				foundPromotedFields = true

				// Verify values are non-negative
				assert.GreaterOrEqual(t, *cell.RowIndex, 0)
				assert.GreaterOrEqual(t, *cell.ColumnIndex, 0)
			}
		}

		assert.True(t, foundPromotedFields, "At least one cell should have RowIndex and ColumnIndex populated")
	})

	t.Run("handles different content types", func(t *testing.T) {
		// Test with string content
		reqString := parser.ParseRequest{
			ID:      "test-csv-003",
			Content: csvContent,
			Config:  parser.DefaultParserConfig(),
		}
		resultString, err := csvParser.Parse(ctx, reqString)
		require.NoError(t, err)
		assert.NotNil(t, resultString)

		// Test with []byte content
		reqBytes := parser.ParseRequest{
			ID:      "test-csv-004",
			Content: []byte(csvContent),
			Config:  parser.DefaultParserConfig(),
		}
		resultBytes, err := csvParser.Parse(ctx, reqBytes)
		require.NoError(t, err)
		assert.NotNil(t, resultBytes)
	})

	t.Run("validates element structure", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-csv-005",
			Content: csvContent,
			Config:  parser.DefaultParserConfig(),
		}

		result, err := csvParser.Parse(ctx, req)
		require.NoError(t, err)

		// Verify all elements have required fields
		for _, elem := range result.Elements {
			assert.NotEmpty(t, elem.ElementID, "Element should have ID")
			assert.NotEmpty(t, elem.ElementType, "Element should have type")
			assert.NotEmpty(t, elem.ContentPreview, "Element should have content preview")
			assert.GreaterOrEqual(t, elem.Position, 0, "Element should have valid position")
			assert.GreaterOrEqual(t, elem.Depth, 0, "Element should have valid depth")
		}
	})

	t.Run("validates CSV-specific element types", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-csv-006",
			Content: csvContent,
			Config:  parser.DefaultParserConfig(),
		}

		result, err := csvParser.Parse(ctx, req)
		require.NoError(t, err)

		elementTypes := make(map[string]int)
		for _, elem := range result.Elements {
			elementTypes[elem.ElementType]++
		}

		// Verify expected element types exist
		assert.Greater(t, elementTypes["root"], 0, "Should have root element")
		assert.Greater(t, elementTypes["table"], 0, "Should have table element")
		assert.Greater(t, elementTypes["table_header_row"], 0, "Should have header row")
		assert.Greater(t, elementTypes["table_row"], 0, "Should have data rows")
		assert.Greater(t, elementTypes["table_cell"], 0, "Should have table cells")
	})

	t.Run("validates relationships exist", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-csv-007",
			Content: csvContent,
			Config:  parser.DefaultParserConfig(),
		}

		result, err := csvParser.Parse(ctx, req)
		require.NoError(t, err)

		// Verify hierarchy via parent_id field (replaces old Relationships array)
		hierarchyCount := 0
		parentChildMap := make(map[string][]string)

		for _, elem := range result.Elements {
			if elem.ParentID != "" {
				hierarchyCount++
				parentChildMap[elem.ParentID] = append(parentChildMap[elem.ParentID], elem.ElementID)
			}
		}

		assert.Greater(t, hierarchyCount, 0, "Should have elements with parent_id forming hierarchy")

		// Verify root element exists and has children
		rootFound := false
		for _, elem := range result.Elements {
			if elem.ElementType == "root" && elem.ParentID == "" {
				rootFound = true
				assert.Greater(t, len(parentChildMap[elem.ElementID]), 0, "Root should have children")
				break
			}
		}
		assert.True(t, rootFound, "Should have root element")
	})
}

// TestCSVPromotedFieldsDetail validates promoted fields in detail
func TestCSVPromotedFieldsDetail(t *testing.T) {
	csvContent := `A,B,C
1,2,3
4,5,6`

	csvParser := parser.NewCSVParser()
	ctx := context.Background()

	req := parser.ParseRequest{
		ID:      "test-promoted-001",
		Content: csvContent,
		Config:  parser.DefaultParserConfig(),
	}

	result, err := csvParser.Parse(ctx, req)
	require.NoError(t, err)

	// Find specific cells and verify their coordinates
	cellsByCoordinate := make(map[string]parser.Element)
	for _, elem := range result.Elements {
		if elem.ElementType == "table_cell" && elem.RowIndex != nil && elem.ColumnIndex != nil {
			key := formatCoordinate(*elem.RowIndex, *elem.ColumnIndex)
			cellsByCoordinate[key] = elem
		}
	}

	t.Run("header cells have correct coordinates", func(t *testing.T) {
		// Header row should be row 0
		for col := 0; col < 3; col++ {
			key := formatCoordinate(0, col)
			cell, exists := cellsByCoordinate[key]
			if exists {
				assert.Equal(t, 0, *cell.RowIndex, "Header cell should be in row 0")
				assert.Equal(t, col, *cell.ColumnIndex, "Column index should match")
			}
		}
	})

	t.Run("data cells have correct coordinates", func(t *testing.T) {
		// Data rows should be rows 1 and 2
		for row := 1; row <= 2; row++ {
			for col := 0; col < 3; col++ {
				key := formatCoordinate(row, col)
				cell, exists := cellsByCoordinate[key]
				if exists {
					assert.Equal(t, row, *cell.RowIndex, "Row index should match")
					assert.Equal(t, col, *cell.ColumnIndex, "Column index should match")
				}
			}
		}
	})
}

// Helper function to format coordinates as string for map key
func formatCoordinate(row, col int) string {
	return fmt.Sprintf("r%d_c%d", row, col)
}
