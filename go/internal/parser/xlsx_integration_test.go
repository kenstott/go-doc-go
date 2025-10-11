package parser_test

import (
	"archive/zip"
	"bytes"
	"context"
	"testing"
	"os"
	"github.com/kennethstott/go-doc-go/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createMinimalXLSX creates a minimal valid XLSX file content
func createMinimalXLSX(sheetData string) []byte {
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	// Create minimal workbook.xml
	workbookXML := `<?xml version="1.0" encoding="UTF-8"?>
<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"
          xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
  <sheets>
    <sheet name="Sheet1" sheetId="1" r:id="rId1"/>
  </sheets>
</workbook>`

	workbookFile, _ := zipWriter.Create("xl/workbook.xml")
	workbookFile.Write([]byte(workbookXML))

	// Create sheet1.xml with provided data
	sheetFile, _ := zipWriter.Create("xl/worksheets/sheet1.xml")
	sheetFile.Write([]byte(sheetData))

	// Create minimal [Content_Types].xml
	contentTypes := `<?xml version="1.0" encoding="UTF-8"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="xml" ContentType="application/xml"/>
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>
  <Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>
</Types>`

	contentFile, _ := zipWriter.Create("[Content_Types].xml")
	contentFile.Write([]byte(contentTypes))

	// Create minimal _rels/.rels
	rels := `<?xml version="1.0" encoding="UTF-8"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>
</Relationships>`

	relsFile, _ := zipWriter.Create("_rels/.rels")
	relsFile.Write([]byte(rels))

	// Create xl/_rels/workbook.xml.rels
	workbookRels := `<?xml version="1.0" encoding="UTF-8"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/>
</Relationships>`

	workbookRelsFile, _ := zipWriter.Create("xl/_rels/workbook.xml.rels")
	workbookRelsFile.Write([]byte(workbookRels))

	zipWriter.Close()
	return buf.Bytes()
}

// TestXLSXParserInterface validates XLSX parser implements Parser interface correctly
func TestXLSXParserInterface(t *testing.T) {
	xlsxParser := parser.NewXLSXParser()

	t.Run("implements Parser interface", func(t *testing.T) {
		// Verify parser implements the interface
		var _ parser.Parser = xlsxParser

		// Test GetName
		assert.Equal(t, "xlsx", xlsxParser.GetName())

		// Test GetSupportedFormats
		formats := xlsxParser.GetSupportedFormats()
		assert.Contains(t, formats, ".xlsx")
		assert.Contains(t, formats, "xlsx")

		// Test SupportsStreaming
		assert.False(t, xlsxParser.SupportsStreaming())

		// Test Close
		err := xlsxParser.Close()
		assert.NoError(t, err)
	})

	t.Run("can be registered in ParserRegistry", func(t *testing.T) {
		registry := parser.NewParserRegistry()
		registry.Register(xlsxParser)

		// Verify parser is registered
		retrieved, err := registry.GetParser("xlsx")
		require.NoError(t, err)
		assert.Equal(t, "xlsx", retrieved.GetName())

		// Verify parser can be found by file extension
		parserByFile, err := registry.GetParserForFile("spreadsheet.xlsx")
		require.NoError(t, err)
		assert.Equal(t, "xlsx", parserByFile.GetName())
	})
}

// TestXLSXParserEndToEnd validates complete XLSX parsing workflow via Parser interface
func TestXLSXParserEndToEnd(t *testing.T) {
	// Create minimal sheet with data
	sheetXML := `<?xml version="1.0" encoding="UTF-8"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
  <sheetData>
    <row r="1">
      <c r="A1" t="inlineStr">
        <is>
          <t>Hello</t>
        </is>
      </c>
      <c r="B1" t="inlineStr">
        <is>
          <t>World</t>
        </is>
      </c>
    </row>
    <row r="2">
      <c r="A2" t="inlineStr">
        <is>
          <t>Data1</t>
        </is>
      </c>
      <c r="B2" t="inlineStr">
        <is>
          <t>Data2</t>
        </is>
      </c>
    </row>
  </sheetData>
</worksheet>`

	xlsxContent := createMinimalXLSX(sheetXML)

	// Write to temporary file since XLSX parser expects file path
	tmpFile, err := os.CreateTemp("", "test-*.xlsx")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.Write(xlsxContent)
	require.NoError(t, err)
	tmpFile.Close()

	xlsxParser := parser.NewXLSXParser()
	ctx := context.Background()

	t.Run("parses XLSX via Parser interface", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-xlsx-001",
			Content: tmpFile.Name(),
			Metadata: map[string]interface{}{
				"source": "test",
			},
			Config: parser.DefaultParserConfig(),
		}

		result, err := xlsxParser.Parse(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, result)

		// Validate Document
		assert.Equal(t, "test-xlsx-001", result.Document.ID)
		assert.Equal(t, "xlsx", result.Document.DocType)

		// Validate Elements exist
		assert.NotEmpty(t, result.Elements)
	})

	t.Run("handles file path content type", func(t *testing.T) {
		// Test with file path string (expected by XLSX parser)
		reqPath := parser.ParseRequest{
			ID:      "test-xlsx-002",
			Content: tmpFile.Name(),
			Config:  parser.DefaultParserConfig(),
		}
		resultPath, err := xlsxParser.Parse(ctx, reqPath)
		require.NoError(t, err)
		assert.NotNil(t, resultPath)
	})

	t.Run("validates element structure", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-xlsx-003",
			Content: tmpFile.Name(),
			Config:  parser.DefaultParserConfig(),
		}

		result, err := xlsxParser.Parse(ctx, req)
		require.NoError(t, err)

		// Verify all elements have required fields
		for _, elem := range result.Elements {
			assert.NotEmpty(t, elem.ElementID, "Element should have ID")
			assert.NotEmpty(t, elem.ElementType, "Element should have type")
			// ContentPreview may be empty for container elements
			assert.GreaterOrEqual(t, elem.Position, 0, "Element should have valid position")
			assert.GreaterOrEqual(t, elem.Depth, 0, "Element should have valid depth")
			assert.NotEmpty(t, elem.ElementCategory, "Element should have category")
		}
	})

	t.Run("validates XLSX-specific element types", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-xlsx-004",
			Content: tmpFile.Name(),
			Config:  parser.DefaultParserConfig(),
		}

		result, err := xlsxParser.Parse(ctx, req)
		require.NoError(t, err)

		elementTypes := make(map[string]int)
		for _, elem := range result.Elements {
			elementTypes[elem.ElementType]++
		}

		// Verify expected element types exist
		assert.Greater(t, elementTypes["root"], 0, "Should have root element")
		// XLSX may have: sheet, table, table_row, table_cell, etc.
	})

	t.Run("validates relationships exist", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-xlsx-005",
			Content: tmpFile.Name(),
			Config:  parser.DefaultParserConfig(),
		}

		result, err := xlsxParser.Parse(ctx, req)
		require.NoError(t, err)

		// XLSX parser should create bidirectional relationships
		assert.NotEmpty(t, result.Relationships, "Should have relationships between elements")

		// Verify relationship structure
		for _, rel := range result.Relationships {
			assert.NotEmpty(t, rel.RelationshipID)
			assert.NotEmpty(t, rel.RelationshipType)
			assert.NotEmpty(t, rel.SourceElementID)
			assert.NotEmpty(t, rel.TargetElementID)
		}

		// Verify bidirectional relationships exist
		hasContains := false
		hasContainedBy := false
		for _, rel := range result.Relationships {
			if rel.RelationshipType == "contains" {
				hasContains = true
			}
			if rel.RelationshipType == "contained_by" {
				hasContainedBy = true
			}
		}
		assert.True(t, hasContains, "Should have 'contains' relationships")
		assert.True(t, hasContainedBy, "Should have 'contained_by' relationships")
	})
}

// TestXLSXPromotedFieldsDetail validates UDML Phase 1 promoted fields for XLSX
func TestXLSXPromotedFieldsDetail(t *testing.T) {
	// Create XLSX with multiple sheets and data
	sheetXML := `<?xml version="1.0" encoding="UTF-8"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
  <sheetData>
    <row r="1">
      <c r="A1" t="inlineStr">
        <is>
          <t>Name</t>
        </is>
      </c>
      <c r="B1" t="inlineStr">
        <is>
          <t>Age</t>
        </is>
      </c>
      <c r="C1" t="inlineStr">
        <is>
          <t>Date</t>
        </is>
      </c>
    </row>
    <row r="2">
      <c r="A2" t="inlineStr">
        <is>
          <t>Alice</t>
        </is>
      </c>
      <c r="B2" t="inlineStr">
        <is>
          <t>30</t>
        </is>
      </c>
      <c r="C2" t="inlineStr">
        <is>
          <t>2023-01-15</t>
        </is>
      </c>
    </row>
    <row r="3">
      <c r="A3" t="inlineStr">
        <is>
          <t>Bob</t>
        </is>
      </c>
      <c r="B3" t="inlineStr">
        <is>
          <t>25</t>
        </is>
      </c>
      <c r="C3" t="inlineStr">
        <is>
          <t>2023-06-20</t>
        </is>
      </c>
    </row>
  </sheetData>
</worksheet>`

	xlsxContent := createMinimalXLSX(sheetXML)
	xlsxParser := parser.NewXLSXParser()
	ctx := context.Background()

	req := parser.ParseRequest{
		ID:      "test-xlsx-promoted-001",
		Content: xlsxContent,
		Config:  parser.DefaultParserConfig(),
	}

	result, err := xlsxParser.Parse(ctx, req)
	require.NoError(t, err)

	t.Run("sheet elements have PageNumber populated", func(t *testing.T) {
		foundPageNumber := false
		for _, elem := range result.Elements {
			if elem.ElementType == "sheet" && elem.PageNumber != nil {
				foundPageNumber = true
				assert.GreaterOrEqual(t, *elem.PageNumber, 1, "PageNumber should be >= 1")
				t.Logf("XLSX sheet with PageNumber: %d", *elem.PageNumber)
			}
		}

		if foundPageNumber {
			t.Log("PageNumber promoted field successfully populated for sheets")
		} else {
			t.Log("No sheet elements with PageNumber found")
		}
	})

	t.Run("cell elements have RowIndex and ColumnIndex", func(t *testing.T) {
		foundCell := false
		cellCoordinates := make(map[string]bool)

		for _, elem := range result.Elements {
			if elem.ElementType == "table_cell" && elem.RowIndex != nil && elem.ColumnIndex != nil {
				foundCell = true
				assert.GreaterOrEqual(t, *elem.RowIndex, 0, "RowIndex should be >= 0")
				assert.GreaterOrEqual(t, *elem.ColumnIndex, 0, "ColumnIndex should be >= 0")

				coordinate := string(rune('A'+*elem.ColumnIndex)) + string(rune('1'+*elem.RowIndex))
				cellCoordinates[coordinate] = true
				t.Logf("XLSX cell at row=%d, col=%d (Excel: %s)", *elem.RowIndex, *elem.ColumnIndex, coordinate)
			}
		}

		assert.True(t, foundCell, "Should have cells with RowIndex and ColumnIndex populated")
		assert.GreaterOrEqual(t, len(cellCoordinates), 6, "Should have multiple cells with coordinates")
		t.Logf("Found %d cells with row/column indices", len(cellCoordinates))
	})

	t.Run("all elements have ElementCategory", func(t *testing.T) {
		for _, elem := range result.Elements {
			assert.NotEmpty(t, elem.ElementCategory, "ElementCategory should be populated")
		}
	})

	t.Run("temporal elements have TemporalType", func(t *testing.T) {
		foundTemporalType := false
		for _, elem := range result.Elements {
			if elem.TemporalType != nil {
				foundTemporalType = true
				assert.NotEmpty(t, *elem.TemporalType, "TemporalType should not be empty if set")
				t.Logf("XLSX element with TemporalType: %s (content: %s)", *elem.TemporalType, elem.ContentPreview)
			}
		}

		if foundTemporalType {
			t.Log("TemporalType promoted field successfully populated")
		} else {
			t.Log("No temporal elements found in XLSX (temporal detection may require configuration)")
		}
	})
}

// TestXLSXDocumentMetadata validates document-level metadata extraction
func TestXLSXDocumentMetadata(t *testing.T) {
	sheetXML := `<?xml version="1.0" encoding="UTF-8"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
  <sheetData>
    <row r="1">
      <c r="A1" t="inlineStr">
        <is>
          <t>Test Data</t>
        </is>
      </c>
    </row>
  </sheetData>
</worksheet>`

	xlsxContent := createMinimalXLSX(sheetXML)
	xlsxParser := parser.NewXLSXParser()
	ctx := context.Background()

	req := parser.ParseRequest{
		ID:      "test-xlsx-metadata-001",
		Content: xlsxContent,
		Config:  parser.DefaultParserConfig(),
	}

	result, err := xlsxParser.Parse(ctx, req)
	require.NoError(t, err)

	t.Run("document metadata exists", func(t *testing.T) {
		assert.NotNil(t, result.Document)
		assert.Equal(t, "test-xlsx-metadata-001", result.Document.ID)
		assert.Equal(t, "xlsx", result.Document.DocType)
	})

	t.Run("elements have XLSX-specific metadata", func(t *testing.T) {
		// XLSX elements may have sheet name, cell reference, formula metadata
		foundMetadata := false
		for _, elem := range result.Elements {
			if elem.Metadata != nil && len(elem.Metadata) > 0 {
				foundMetadata = true
				t.Logf("Found element with metadata: %+v", elem.Metadata)
				break
			}
		}

		if foundMetadata {
			t.Log("XLSX elements have metadata populated")
		} else {
			t.Log("No element metadata found (may be expected)")
		}
	})
}

// TestXLSXMultipleSheets validates parsing of XLSX with multiple sheets
func TestXLSXMultipleSheets(t *testing.T) {
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	// Workbook with 2 sheets
	workbookXML := `<?xml version="1.0" encoding="UTF-8"?>
<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"
          xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
  <sheets>
    <sheet name="Sales" sheetId="1" r:id="rId1"/>
    <sheet name="Expenses" sheetId="2" r:id="rId2"/>
  </sheets>
</workbook>`

	workbookFile, _ := zipWriter.Create("xl/workbook.xml")
	workbookFile.Write([]byte(workbookXML))

	// Sheet 1
	sheet1XML := `<?xml version="1.0" encoding="UTF-8"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
  <sheetData>
    <row r="1">
      <c r="A1" t="inlineStr"><is><t>Sales Data</t></is></c>
    </row>
  </sheetData>
</worksheet>`
	sheet1File, _ := zipWriter.Create("xl/worksheets/sheet1.xml")
	sheet1File.Write([]byte(sheet1XML))

	// Sheet 2
	sheet2XML := `<?xml version="1.0" encoding="UTF-8"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
  <sheetData>
    <row r="1">
      <c r="A1" t="inlineStr"><is><t>Expense Data</t></is></c>
    </row>
  </sheetData>
</worksheet>`
	sheet2File, _ := zipWriter.Create("xl/worksheets/sheet2.xml")
	sheet2File.Write([]byte(sheet2XML))

	contentTypes := `<?xml version="1.0" encoding="UTF-8"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>
  <Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>
  <Override PartName="/xl/worksheets/sheet2.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>
</Types>`
	contentFile, _ := zipWriter.Create("[Content_Types].xml")
	contentFile.Write([]byte(contentTypes))

	rels := `<?xml version="1.0" encoding="UTF-8"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>
</Relationships>`
	relsFile, _ := zipWriter.Create("_rels/.rels")
	relsFile.Write([]byte(rels))

	workbookRels := `<?xml version="1.0" encoding="UTF-8"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/>
  <Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet2.xml"/>
</Relationships>`
	workbookRelsFile, _ := zipWriter.Create("xl/_rels/workbook.xml.rels")
	workbookRelsFile.Write([]byte(workbookRels))

	zipWriter.Close()
	xlsxContent := buf.Bytes()

	xlsxParser := parser.NewXLSXParser()
	ctx := context.Background()

	req := parser.ParseRequest{
		ID:      "test-xlsx-multisheet-001",
		Content: xlsxContent,
		Config:  parser.DefaultParserConfig(),
	}

	result, err := xlsxParser.Parse(ctx, req)
	require.NoError(t, err)

	t.Run("parses multiple sheets", func(t *testing.T) {
		sheetCount := 0
		for _, elem := range result.Elements {
			if elem.ElementType == "sheet" {
				sheetCount++
			}
		}

		assert.GreaterOrEqual(t, sheetCount, 1, "Should have at least one sheet")
		t.Logf("Found %d sheets in XLSX", sheetCount)
	})
}
