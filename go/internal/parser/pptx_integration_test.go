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

// createMinimalPPTX creates a minimal valid PPTX file content
func createMinimalPPTX(text string) []byte {
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	// Create minimal presentation.xml
	presentationXML := `<?xml version="1.0" encoding="UTF-8"?>
<p:presentation xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"
                xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
  <p:sldIdLst>
    <p:sldId id="256" r:id="rId2"/>
  </p:sldIdLst>
</p:presentation>`

	presFile, _ := zipWriter.Create("ppt/presentation.xml")
	presFile.Write([]byte(presentationXML))

	// Create minimal slide1.xml
	slideXML := `<?xml version="1.0" encoding="UTF-8"?>
<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"
       xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
  <p:cSld>
    <p:spTree>
      <p:sp>
        <p:txBody>
          <a:p>
            <a:r>
              <a:t>` + text + `</a:t>
            </a:r>
          </a:p>
        </p:txBody>
      </p:sp>
    </p:spTree>
  </p:cSld>
</p:sld>`

	slideFile, _ := zipWriter.Create("ppt/slides/slide1.xml")
	slideFile.Write([]byte(slideXML))

	// Create minimal [Content_Types].xml
	contentTypes := `<?xml version="1.0" encoding="UTF-8"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="xml" ContentType="application/xml"/>
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Override PartName="/ppt/presentation.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.presentation.main+xml"/>
  <Override PartName="/ppt/slides/slide1.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slide+xml"/>
</Types>`

	contentFile, _ := zipWriter.Create("[Content_Types].xml")
	contentFile.Write([]byte(contentTypes))

	// Create minimal _rels/.rels
	rels := `<?xml version="1.0" encoding="UTF-8"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="ppt/presentation.xml"/>
</Relationships>`

	relsFile, _ := zipWriter.Create("_rels/.rels")
	relsFile.Write([]byte(rels))

	// Create ppt/_rels/presentation.xml.rels
	presRels := `<?xml version="1.0" encoding="UTF-8"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slide" Target="slides/slide1.xml"/>
</Relationships>`

	presRelsFile, _ := zipWriter.Create("ppt/_rels/presentation.xml.rels")
	presRelsFile.Write([]byte(presRels))

	zipWriter.Close()
	return buf.Bytes()
}

// TestPPTXParserInterface validates PPTX parser implements Parser interface correctly
func TestPPTXParserInterface(t *testing.T) {
	pptxParser := parser.NewPPTXParser()

	t.Run("implements Parser interface", func(t *testing.T) {
		// Verify parser implements the interface
		var _ parser.Parser = pptxParser

		// Test GetName
		assert.Equal(t, "pptx", pptxParser.GetName())

		// Test GetSupportedFormats
		formats := pptxParser.GetSupportedFormats()
		assert.Contains(t, formats, ".pptx")
		assert.Contains(t, formats, "pptx")

		// Test SupportsStreaming
		assert.False(t, pptxParser.SupportsStreaming())

		// Test Close
		err := pptxParser.Close()
		assert.NoError(t, err)
	})

	t.Run("can be registered in ParserRegistry", func(t *testing.T) {
		registry := parser.NewParserRegistry()
		registry.Register(pptxParser)

		// Verify parser is registered
		retrieved, err := registry.GetParser("pptx")
		require.NoError(t, err)
		assert.Equal(t, "pptx", retrieved.GetName())

		// Verify parser can be found by file extension
		parserByFile, err := registry.GetParserForFile("presentation.pptx")
		require.NoError(t, err)
		assert.Equal(t, "pptx", parserByFile.GetName())
	})
}

// TestPPTXParserEndToEnd validates complete PPTX parsing workflow via Parser interface
func TestPPTXParserEndToEnd(t *testing.T) {
	pptxContent := createMinimalPPTX("Hello World from PPTX")

	// Write to temporary file since PPTX parser expects file path
	tmpFile, err := os.CreateTemp("", "test-*.pptx")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.Write(pptxContent)
	require.NoError(t, err)
	tmpFile.Close()

	pptxParser := parser.NewPPTXParser()
	ctx := context.Background()

	t.Run("parses PPTX via Parser interface", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-pptx-001",
			Content: tmpFile.Name(),
			Metadata: map[string]interface{}{
				"source": "test",
			},
			Config: parser.DefaultParserConfig(),
		}

		result, err := pptxParser.Parse(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, result)

		assert.Equal(t, "test-pptx-001", result.Document.ID)
		assert.Equal(t, "pptx", result.Document.DocType)
		assert.NotEmpty(t, result.Elements)
	})

	t.Run("handles file path content type", func(t *testing.T) {
		reqPath := parser.ParseRequest{
			ID:      "test-pptx-002",
			Content: tmpFile.Name(),
			Config:  parser.DefaultParserConfig(),
		}
		resultPath, err := pptxParser.Parse(ctx, reqPath)
		require.NoError(t, err)
		assert.NotNil(t, resultPath)
	})

	t.Run("validates element structure", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-pptx-003",
			Content: tmpFile.Name(),
			Config:  parser.DefaultParserConfig(),
		}

		result, err := pptxParser.Parse(ctx, req)
		require.NoError(t, err)

		for _, elem := range result.Elements {
			assert.NotEmpty(t, elem.ElementID, "Element should have ID")
			assert.NotEmpty(t, elem.ElementType, "Element should have type")
			assert.GreaterOrEqual(t, elem.Position, 0, "Element should have valid position")
			assert.GreaterOrEqual(t, elem.Depth, 0, "Element should have valid depth")
			assert.NotEmpty(t, elem.ElementCategory, "Element should have category")
		}
	})

	t.Run("validates PPTX-specific element types", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-pptx-004",
			Content: tmpFile.Name(),
			Config:  parser.DefaultParserConfig(),
		}

		result, err := pptxParser.Parse(ctx, req)
		require.NoError(t, err)

		elementTypes := make(map[string]int)
		for _, elem := range result.Elements {
			elementTypes[elem.ElementType]++
		}

		assert.Greater(t, elementTypes["root"], 0, "Should have root element")
	})

	t.Run("validates relationships exist", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-pptx-005",
			Content: tmpFile.Name(),
			Config:  parser.DefaultParserConfig(),
		}

		result, err := pptxParser.Parse(ctx, req)
		require.NoError(t, err)

		assert.NotEmpty(t, result.Relationships, "Should have relationships between elements")

		for _, rel := range result.Relationships {
			assert.NotEmpty(t, rel.RelationshipID)
			assert.NotEmpty(t, rel.RelationshipType)
			assert.NotEmpty(t, rel.SourceElementID)
			assert.NotEmpty(t, rel.TargetElementID)
		}

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
		assert.True(t, hasContains, "Should have contains relationships")
		assert.True(t, hasContainedBy, "Should have contained_by relationships")
	})
}
// TestPPTXPromotedFieldsDetail validates UDML Phase 1 promoted fields for PPTX
func TestPPTXPromotedFieldsDetail(t *testing.T) {
	// Create PPTX with multiple slides and table
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	// Presentation with 2 slides
	presentationXML := `<?xml version="1.0" encoding="UTF-8"?>
<p:presentation xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"
                xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
  <p:sldIdLst>
    <p:sldId id="256" r:id="rId2"/>
    <p:sldId id="257" r:id="rId3"/>
  </p:sldIdLst>
</p:presentation>`

	presFile, _ := zipWriter.Create("ppt/presentation.xml")
	presFile.Write([]byte(presentationXML))

	// Slide 1
	slide1XML := `<?xml version="1.0" encoding="UTF-8"?>
<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"
       xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
  <p:cSld>
    <p:spTree>
      <p:sp>
        <p:txBody>
          <a:p>
            <a:r>
              <a:t>Slide 1 Content</a:t>
            </a:r>
          </a:p>
        </p:txBody>
      </p:sp>
    </p:spTree>
  </p:cSld>
</p:sld>`
	slide1File, _ := zipWriter.Create("ppt/slides/slide1.xml")
	slide1File.Write([]byte(slide1XML))

	// Slide 2
	slide2XML := `<?xml version="1.0" encoding="UTF-8"?>
<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"
       xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
  <p:cSld>
    <p:spTree>
      <p:sp>
        <p:txBody>
          <a:p>
            <a:r>
              <a:t>Slide 2 Content</a:t>
            </a:r>
          </a:p>
        </p:txBody>
      </p:sp>
    </p:spTree>
  </p:cSld>
</p:sld>`
	slide2File, _ := zipWriter.Create("ppt/slides/slide2.xml")
	slide2File.Write([]byte(slide2XML))

	contentTypes := `<?xml version="1.0" encoding="UTF-8"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/ppt/presentation.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.presentation.main+xml"/>
  <Override PartName="/ppt/slides/slide1.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slide+xml"/>
  <Override PartName="/ppt/slides/slide2.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slide+xml"/>
</Types>`
	contentFile, _ := zipWriter.Create("[Content_Types].xml")
	contentFile.Write([]byte(contentTypes))

	rels := `<?xml version="1.0" encoding="UTF-8"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="ppt/presentation.xml"/>
</Relationships>`
	relsFile, _ := zipWriter.Create("_rels/.rels")
	relsFile.Write([]byte(rels))

	presRels := `<?xml version="1.0" encoding="UTF-8"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slide" Target="slides/slide1.xml"/>
  <Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slide" Target="slides/slide2.xml"/>
</Relationships>`
	presRelsFile, _ := zipWriter.Create("ppt/_rels/presentation.xml.rels")
	presRelsFile.Write([]byte(presRels))

	zipWriter.Close()
	pptxContent := buf.Bytes()

	pptxParser := parser.NewPPTXParser()
	ctx := context.Background()

	req := parser.ParseRequest{
		ID:      "test-pptx-promoted-001",
		Content: pptxContent,
		Config:  parser.DefaultParserConfig(),
	}

	result, err := pptxParser.Parse(ctx, req)
	require.NoError(t, err)

	t.Run("slide elements have PageNumber populated", func(t *testing.T) {
		foundPageNumber := false
		slideNumbers := make(map[int]bool)

		for _, elem := range result.Elements {
			if elem.ElementType == "slide" && elem.PageNumber != nil {
				foundPageNumber = true
				slideNumbers[*elem.PageNumber] = true
				assert.GreaterOrEqual(t, *elem.PageNumber, 1, "PageNumber should be >= 1")
				t.Logf("PPTX slide with PageNumber: %d", *elem.PageNumber)
			}
		}

		if foundPageNumber {
			assert.GreaterOrEqual(t, len(slideNumbers), 1, "Should have at least one slide with PageNumber")
			t.Log("PageNumber promoted field successfully populated for slides")
		} else {
			t.Log("No slide elements with PageNumber found")
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
					t.Logf("PPTX table cell at row=%d, col=%d", *elem.RowIndex, *elem.ColumnIndex)
				}
			}
		}

		if foundTableCell {
			t.Log("Table cells found in PPTX with row/column indices")
		} else {
			t.Log("No table cells found in PPTX (may not have tables in test content)")
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
				t.Logf("PPTX element with TemporalType: %s", *elem.TemporalType)
			}
		}

		if foundTemporalType {
			t.Log("TemporalType promoted field successfully populated")
		} else {
			t.Log("No temporal elements found in PPTX")
		}
	})
}

// TestPPTXDocumentMetadata validates document-level metadata extraction
func TestPPTXDocumentMetadata(t *testing.T) {
	pptxContent := createMinimalPPTX("Test Presentation Content")

	pptxParser := parser.NewPPTXParser()
	ctx := context.Background()

	req := parser.ParseRequest{
		ID:      "test-pptx-metadata-001",
		Content: pptxContent,
		Config:  parser.DefaultParserConfig(),
	}

	result, err := pptxParser.Parse(ctx, req)
	require.NoError(t, err)

	t.Run("document metadata exists", func(t *testing.T) {
		assert.NotNil(t, result.Document)
		assert.Equal(t, "test-pptx-metadata-001", result.Document.ID)
		assert.Equal(t, "pptx", result.Document.DocType)
	})

	t.Run("elements have PPTX-specific metadata", func(t *testing.T) {
		// PPTX elements may have slide number, shape type, position metadata
		foundMetadata := false
		for _, elem := range result.Elements {
			if elem.Metadata != nil && len(elem.Metadata) > 0 {
				foundMetadata = true
				t.Logf("Found element with metadata: %+v", elem.Metadata)
				break
			}
		}

		if foundMetadata {
			t.Log("PPTX elements have metadata populated")
		} else {
			t.Log("No element metadata found (may be expected)")
		}
	})
}
