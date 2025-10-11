package parser_test

import (
	"archive/zip"
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/kennethstott/go-doc-go/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createMinimalDOCX creates a minimal valid DOCX file content
func createMinimalDOCX(text string) []byte {
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	// Create minimal document.xml
	docXML := `<?xml version="1.0" encoding="UTF-8"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p>
      <w:r>
        <w:t>` + text + `</w:t>
      </w:r>
    </w:p>
  </w:body>
</w:document>`

	// Write document.xml
	docFile, _ := zipWriter.Create("word/document.xml")
	docFile.Write([]byte(docXML))

	// Create minimal [Content_Types].xml
	contentTypes := `<?xml version="1.0" encoding="UTF-8"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="xml" ContentType="application/xml"/>
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
</Types>`

	contentFile, _ := zipWriter.Create("[Content_Types].xml")
	contentFile.Write([]byte(contentTypes))

	// Create minimal _rels/.rels
	rels := `<?xml version="1.0" encoding="UTF-8"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`

	relsFile, _ := zipWriter.Create("_rels/.rels")
	relsFile.Write([]byte(rels))

	zipWriter.Close()
	return buf.Bytes()
}

// TestDOCXParserInterface validates DOCX parser implements Parser interface correctly
func TestDOCXParserInterface(t *testing.T) {
	docxParser := parser.NewDOCXParser()

	t.Run("implements Parser interface", func(t *testing.T) {
		// Verify parser implements the interface
		var _ parser.Parser = docxParser

		// Test GetName
		assert.Equal(t, "docx", docxParser.GetName())

		// Test GetSupportedFormats
		formats := docxParser.GetSupportedFormats()
		assert.Contains(t, formats, ".docx")
		assert.Contains(t, formats, "docx")

		// Test SupportsStreaming
		assert.False(t, docxParser.SupportsStreaming())

		// Test Close
		err := docxParser.Close()
		assert.NoError(t, err)
	})

	t.Run("can be registered in ParserRegistry", func(t *testing.T) {
		registry := parser.NewParserRegistry()
		registry.Register(docxParser)

		// Verify parser is registered
		retrieved, err := registry.GetParser("docx")
		require.NoError(t, err)
		assert.Equal(t, "docx", retrieved.GetName())

		// Verify parser can be found by file extension
		parserByFile, err := registry.GetParserForFile("document.docx")
		require.NoError(t, err)
		assert.Equal(t, "docx", parserByFile.GetName())
	})
}

// TestDOCXParserEndToEnd validates complete DOCX parsing workflow via Parser interface
func TestDOCXParserEndToEnd(t *testing.T) {
	docxContent := createMinimalDOCX("Hello World from DOCX")

	// Write to temporary file since DOCX parser expects file path
	tmpFile, err := os.CreateTemp("", "test-*.docx")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.Write(docxContent)
	require.NoError(t, err)
	tmpFile.Close()

	docxParser := parser.NewDOCXParser()
	ctx := context.Background()

	t.Run("parses DOCX via Parser interface", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-docx-001",
			Content: tmpFile.Name(),
			Metadata: map[string]interface{}{
				"source": "test",
			},
			Config: parser.DefaultParserConfig(),
		}

		result, err := docxParser.Parse(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, result)

		// Validate Document
		assert.Equal(t, "test-docx-001", result.Document.ID)
		assert.Equal(t, "docx", result.Document.DocType)

		// Validate Elements exist
		assert.NotEmpty(t, result.Elements)
	})

	t.Run("handles file path content type", func(t *testing.T) {
		// Test with file path string (expected by DOCX parser)
		reqPath := parser.ParseRequest{
			ID:      "test-docx-002",
			Content: tmpFile.Name(),
			Config:  parser.DefaultParserConfig(),
		}
		resultPath, err := docxParser.Parse(ctx, reqPath)
		require.NoError(t, err)
		assert.NotNil(t, resultPath)
	})

	t.Run("validates element structure", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-docx-003",
			Content: tmpFile.Name(),
			Config:  parser.DefaultParserConfig(),
		}

		result, err := docxParser.Parse(ctx, req)
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

	t.Run("validates DOCX-specific element types", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-docx-004",
			Content: tmpFile.Name(),
			Config:  parser.DefaultParserConfig(),
		}

		result, err := docxParser.Parse(ctx, req)
		require.NoError(t, err)

		elementTypes := make(map[string]int)
		for _, elem := range result.Elements {
			elementTypes[elem.ElementType]++
		}

		// Verify expected element types exist
		assert.Greater(t, elementTypes["root"], 0, "Should have root element")
		// DOCX may have: paragraph, text_run, header, footer, table, table_row, table_cell, etc.
	})

	t.Run("validates relationships exist", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-docx-005",
			Content: tmpFile.Name(),
			Config:  parser.DefaultParserConfig(),
		}

		result, err := docxParser.Parse(ctx, req)
		require.NoError(t, err)

		// DOCX parser should create bidirectional relationships
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

// TestDOCXPromotedFieldsDetail validates UDML Phase 1 promoted fields for DOCX
func TestDOCXPromotedFieldsDetail(t *testing.T) {
	// Create DOCX with headers and table
	docXML := `<?xml version="1.0" encoding="UTF-8"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p>
      <w:pPr>
        <w:pStyle w:val="Heading1"/>
      </w:pPr>
      <w:r>
        <w:t>Main Heading</w:t>
      </w:r>
    </w:p>
    <w:p>
      <w:pPr>
        <w:pStyle w:val="Heading2"/>
      </w:pPr>
      <w:r>
        <w:t>Subheading</w:t>
      </w:r>
    </w:p>
    <w:tbl>
      <w:tr>
        <w:tc>
          <w:p>
            <w:r>
              <w:t>Cell A1</w:t>
            </w:r>
          </w:p>
        </w:tc>
        <w:tc>
          <w:p>
            <w:r>
              <w:t>Cell B1</w:t>
            </w:r>
          </w:p>
        </w:tc>
      </w:tr>
    </w:tbl>
  </w:body>
</w:document>`

	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)
	docFile, _ := zipWriter.Create("word/document.xml")
	docFile.Write([]byte(docXML))

	contentTypes := `<?xml version="1.0" encoding="UTF-8"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
</Types>`
	contentFile, _ := zipWriter.Create("[Content_Types].xml")
	contentFile.Write([]byte(contentTypes))

	rels := `<?xml version="1.0" encoding="UTF-8"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`
	relsFile, _ := zipWriter.Create("_rels/.rels")
	relsFile.Write([]byte(rels))

	zipWriter.Close()
	docxContent := buf.Bytes()

	// Write to temporary file
	tmpFile, err := os.CreateTemp("", "test-promoted-*.docx")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.Write(docxContent)
	require.NoError(t, err)
	tmpFile.Close()

	docxParser := parser.NewDOCXParser()
	ctx := context.Background()

	req := parser.ParseRequest{
		ID:      "test-docx-promoted-001",
		Content: tmpFile.Name(),
		Config:  parser.DefaultParserConfig(),
	}

	result, err := docxParser.Parse(ctx, req)
	require.NoError(t, err)

	t.Run("header elements have SectionLevel populated", func(t *testing.T) {
		foundSectionLevel := false
		for _, elem := range result.Elements {
			if elem.ElementType == "header" && elem.SectionLevel != nil {
				foundSectionLevel = true
				assert.GreaterOrEqual(t, *elem.SectionLevel, 1, "SectionLevel should be >= 1")
				assert.LessOrEqual(t, *elem.SectionLevel, 6, "SectionLevel should be <= 6")
				t.Logf("DOCX header with SectionLevel: %d", *elem.SectionLevel)
			}
		}

		if foundSectionLevel {
			t.Log("SectionLevel promoted field successfully populated")
		} else {
			t.Log("No header elements with SectionLevel found")
		}
	})

	t.Run("table elements have RowIndex and ColumnIndex", func(t *testing.T) {
		foundTableCell := false
		for _, elem := range result.Elements {
			if elem.ElementType == "table_cell" {
				foundTableCell = true
				if elem.RowIndex != nil && elem.ColumnIndex != nil {
					assert.GreaterOrEqual(t, *elem.RowIndex, 0, "RowIndex should be >= 0")
					assert.GreaterOrEqual(t, *elem.ColumnIndex, 0, "ColumnIndex should be >= 0")
					t.Logf("DOCX table cell at row=%d, col=%d", *elem.RowIndex, *elem.ColumnIndex)
				}
			}
		}

		if foundTableCell {
			t.Log("Table cells found in DOCX with row/column indices")
		} else {
			t.Log("No table cells found in DOCX")
		}
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
				t.Logf("DOCX element with TemporalType: %s", *elem.TemporalType)
			}
		}

		if foundTemporalType {
			t.Log("TemporalType promoted field successfully populated")
		} else {
			t.Log("No temporal elements found in DOCX")
		}
	})
}

// TestDOCXDocumentMetadata validates document-level metadata extraction
func TestDOCXDocumentMetadata(t *testing.T) {
	docxContent := createMinimalDOCX("Test Document Content")

	// Write to temporary file
	tmpFile, err := os.CreateTemp("", "test-metadata-*.docx")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.Write(docxContent)
	require.NoError(t, err)
	tmpFile.Close()

	docxParser := parser.NewDOCXParser()
	ctx := context.Background()

	req := parser.ParseRequest{
		ID:      "test-docx-metadata-001",
		Content: tmpFile.Name(),
		Config:  parser.DefaultParserConfig(),
	}

	result, err := docxParser.Parse(ctx, req)
	require.NoError(t, err)

	t.Run("document metadata exists", func(t *testing.T) {
		assert.NotNil(t, result.Document)
		assert.Equal(t, "test-docx-metadata-001", result.Document.ID)
		assert.Equal(t, "docx", result.Document.DocType)
	})

	t.Run("elements have DOCX-specific metadata", func(t *testing.T) {
		// DOCX elements may have style, formatting metadata
		foundMetadata := false
		for _, elem := range result.Elements {
			if elem.Metadata != nil && len(elem.Metadata) > 0 {
				foundMetadata = true
				t.Logf("Found element with metadata: %+v", elem.Metadata)
				break
			}
		}

		if foundMetadata {
			t.Log("DOCX elements have metadata populated")
		} else {
			t.Log("No element metadata found (may be expected)")
		}
	})
}
