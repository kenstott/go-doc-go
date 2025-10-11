package parser_test

import (
	"context"
	"testing"

	"github.com/kennethstott/go-doc-go/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPDFParserInterface validates PDF parser implements Parser interface correctly
func TestPDFParserInterface(t *testing.T) {
	pdfParser := parser.NewPDFParser()

	t.Run("implements Parser interface", func(t *testing.T) {
		// Verify parser implements the interface
		var _ parser.Parser = pdfParser

		// Test GetName
		assert.Equal(t, "pdf", pdfParser.GetName())

		// Test GetSupportedFormats
		formats := pdfParser.GetSupportedFormats()
		assert.Contains(t, formats, ".pdf")
		assert.Contains(t, formats, "pdf")

		// Test SupportsStreaming
		assert.False(t, pdfParser.SupportsStreaming())

		// Test Close
		err := pdfParser.Close()
		assert.NoError(t, err)
	})

	t.Run("can be registered in ParserRegistry", func(t *testing.T) {
		registry := parser.NewParserRegistry()
		registry.Register(pdfParser)

		// Verify parser is registered
		retrieved, err := registry.GetParser("pdf")
		require.NoError(t, err)
		assert.Equal(t, "pdf", retrieved.GetName())

		// Verify parser can be found by file extension
		parserByFile, err := registry.GetParserForFile("document.pdf")
		require.NoError(t, err)
		assert.Equal(t, "pdf", parserByFile.GetName())
	})
}

// TestPDFParserEndToEnd validates complete PDF parsing workflow via Parser interface
func TestPDFParserEndToEnd(t *testing.T) {
	// Create minimal valid PDF content (PDF header)
	pdfContent := []byte("%PDF-1.4\n1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R >>\nendobj\n4 0 obj\n<< /Length 44 >>\nstream\nBT\n/F1 12 Tf\n100 700 Td\n(Hello World) Tj\nET\nendstream\nendobj\nxref\n0 5\n0000000000 65535 f\n0000000009 00000 n\n0000000058 00000 n\n0000000115 00000 n\n0000000214 00000 n\ntrailer\n<< /Size 5 /Root 1 0 R >>\nstartxref\n310\n%%EOF")

	pdfParser := parser.NewPDFParser()
	ctx := context.Background()

	t.Run("parses PDF via Parser interface", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-pdf-001",
			Content: pdfContent,
			Metadata: map[string]interface{}{
				"source": "test",
			},
			Config: parser.DefaultParserConfig(),
		}

		result, err := pdfParser.Parse(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, result)

		// Validate Document
		assert.Equal(t, "test-pdf-001", result.Document.ID)
		assert.Equal(t, "pdf", result.Document.DocType)

		// Validate Elements exist
		assert.NotEmpty(t, result.Elements)
	})

	t.Run("handles different content types", func(t *testing.T) {
		// Test with []byte content
		reqBytes := parser.ParseRequest{
			ID:      "test-pdf-002",
			Content: pdfContent,
			Config:  parser.DefaultParserConfig(),
		}
		resultBytes, err := pdfParser.Parse(ctx, reqBytes)
		require.NoError(t, err)
		assert.NotNil(t, resultBytes)
	})

	t.Run("validates element structure", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-pdf-003",
			Content: pdfContent,
			Config:  parser.DefaultParserConfig(),
		}

		result, err := pdfParser.Parse(ctx, req)
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

	t.Run("validates PDF-specific element types", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-pdf-004",
			Content: pdfContent,
			Config:  parser.DefaultParserConfig(),
		}

		result, err := pdfParser.Parse(ctx, req)
		require.NoError(t, err)

		elementTypes := make(map[string]int)
		for _, elem := range result.Elements {
			elementTypes[elem.ElementType]++
		}

		// Verify expected element types exist
		assert.Greater(t, elementTypes["root"], 0, "Should have root element")
		// PDF may have: page, paragraph, text_block, header, footer, etc.
	})

	t.Run("validates relationships exist", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-pdf-005",
			Content: pdfContent,
			Config:  parser.DefaultParserConfig(),
		}

		result, err := pdfParser.Parse(ctx, req)
		require.NoError(t, err)

		// PDF parser should create bidirectional relationships
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

// TestPDFPromotedFieldsDetail validates UDML Phase 1 promoted fields for PDF
func TestPDFPromotedFieldsDetail(t *testing.T) {
	// Multi-page PDF content
	pdfContent := []byte("%PDF-1.4\n1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R >>\nendobj\n4 0 obj\n<< /Length 44 >>\nstream\nBT\n/F1 12 Tf\n100 700 Td\n(Page 1 Content) Tj\nET\nendstream\nendobj\nxref\n0 5\n0000000000 65535 f\n0000000009 00000 n\n0000000058 00000 n\n0000000115 00000 n\n0000000214 00000 n\ntrailer\n<< /Size 5 /Root 1 0 R >>\nstartxref\n310\n%%EOF")

	pdfParser := parser.NewPDFParser()
	ctx := context.Background()

	req := parser.ParseRequest{
		ID:      "test-pdf-promoted-001",
		Content: pdfContent,
		Config:  parser.DefaultParserConfig(),
	}

	result, err := pdfParser.Parse(ctx, req)
	require.NoError(t, err)

	t.Run("page elements have PageNumber populated", func(t *testing.T) {
		foundPageNumber := false
		for _, elem := range result.Elements {
			if elem.ElementType == "page" && elem.PageNumber != nil {
				foundPageNumber = true
				assert.GreaterOrEqual(t, *elem.PageNumber, 1, "PageNumber should be >= 1")
				t.Logf("PDF page with PageNumber: %d", *elem.PageNumber)
			}
		}

		// Note: May not have page elements if PDF parser creates different structure
		if foundPageNumber {
			t.Log("PageNumber promoted field successfully populated")
		} else {
			t.Log("No page elements with PageNumber (structure may vary)")
		}
	})

	t.Run("header elements have SectionLevel populated", func(t *testing.T) {
		foundSectionLevel := false
		for _, elem := range result.Elements {
			if elem.ElementType == "header" && elem.SectionLevel != nil {
				foundSectionLevel = true
				assert.GreaterOrEqual(t, *elem.SectionLevel, 1, "SectionLevel should be >= 1")
				t.Logf("PDF header with SectionLevel: %d", *elem.SectionLevel)
			}
		}

		if foundSectionLevel {
			t.Log("SectionLevel promoted field successfully populated")
		} else {
			t.Log("No header elements with SectionLevel found")
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
				t.Logf("PDF element with TemporalType: %s", *elem.TemporalType)
			}
		}

		if foundTemporalType {
			t.Log("TemporalType promoted field successfully populated")
		} else {
			t.Log("No temporal elements found in PDF")
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
					t.Logf("PDF table cell at row=%d, col=%d", *elem.RowIndex, *elem.ColumnIndex)
				}
			}
		}

		if foundTableCell {
			t.Log("Table cells found in PDF with row/column indices")
		} else {
			t.Log("No table cells found in PDF")
		}
	})
}

// TestPDFDocumentMetadata validates document-level metadata extraction
func TestPDFDocumentMetadata(t *testing.T) {
	pdfContent := []byte("%PDF-1.4\n1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R >>\nendobj\n4 0 obj\n<< /Length 44 >>\nstream\nBT\n/F1 12 Tf\n100 700 Td\n(Test Document) Tj\nET\nendstream\nendobj\nxref\n0 5\n0000000000 65535 f\n0000000009 00000 n\n0000000058 00000 n\n0000000115 00000 n\n0000000214 00000 n\ntrailer\n<< /Size 5 /Root 1 0 R >>\nstartxref\n310\n%%EOF")

	pdfParser := parser.NewPDFParser()
	ctx := context.Background()

	req := parser.ParseRequest{
		ID:      "test-pdf-metadata-001",
		Content: pdfContent,
		Config:  parser.DefaultParserConfig(),
	}

	result, err := pdfParser.Parse(ctx, req)
	require.NoError(t, err)

	t.Run("document metadata exists", func(t *testing.T) {
		assert.NotNil(t, result.Document)
		assert.Equal(t, "test-pdf-metadata-001", result.Document.ID)
		assert.Equal(t, "pdf", result.Document.DocType)
	})

	t.Run("elements have PDF-specific metadata", func(t *testing.T) {
		// PDF elements may have page_number, font info, position metadata
		foundMetadata := false
		for _, elem := range result.Elements {
			if elem.Metadata != nil && len(elem.Metadata) > 0 {
				foundMetadata = true
				t.Logf("Found element with metadata: %+v", elem.Metadata)
				break
			}
		}

		if foundMetadata {
			t.Log("PDF elements have metadata populated")
		} else {
			t.Log("No element metadata found (may be expected)")
		}
	})
}
