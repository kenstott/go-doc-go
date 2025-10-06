package parser

import (
	"path/filepath"
	"testing"
)

func TestDocxParser_Creation(t *testing.T) {
	parser := NewDocxParser()
	if parser == nil {
		t.Fatal("Failed to create DOCX parser")
	}
	if parser.MaxContentPreview != 100 {
		t.Errorf("Expected MaxContentPreview 100, got %d", parser.MaxContentPreview)
	}
	if !parser.ExtractHeadersFooters {
		t.Error("ExtractHeadersFooters should be true by default")
	}
	if !parser.ExtractComments {
		t.Error("ExtractComments should be true by default")
	}
	if !parser.ExtractStyles {
		t.Error("ExtractStyles should be true by default")
	}
	if parser.MaxImageSize != 1024*1024 {
		t.Errorf("Expected MaxImageSize 1MB, got %d", parser.MaxImageSize)
	}
}

func TestDocxParser_BasicParsing(t *testing.T) {
	parser := NewDocxParser()

	// Get path to test asset
	testAssets := "../../../tests/assets"
	testFile := filepath.Join(testAssets, "test_sample.docx")

	request := DocxParseRequest{
		ID:       "test_doc",
		Content:  testFile,
		Metadata: map[string]interface{}{"filename": "test_sample.docx"},
	}

	response, err := parser.Parse(request)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Verify document structure
	if response.Document["doc_id"] != "test_doc" {
		t.Errorf("Expected doc_id 'test_doc', got %v", response.Document["doc_id"])
	}
	if response.Document["doc_type"] != "docx" {
		t.Errorf("Expected doc_type 'docx', got %v", response.Document["doc_type"])
	}

	// Should have root element
	if len(response.Elements) == 0 {
		t.Fatal("Expected at least root element")
	}

	// Check root element
	rootElement := response.Elements[0]
	if rootElement.ElementType != "root" {
		t.Errorf("Expected root element type, got %v", rootElement.ElementType)
	}
	if rootElement.ParentID != "" {
		t.Errorf("Root element should not have parent, got %v", rootElement.ParentID)
	}
	if rootElement.ElementID == "" {
		t.Error("Root element should have ID")
	}

	// Should have headers, footers, body, and comments containers
	hasHeaders := false
	hasFooters := false
	hasBody := false
	hasComments := false

	for _, elem := range response.Elements {
		switch elem.ElementType {
		case "headers":
			hasHeaders = true
		case "footers":
			hasFooters = true
		case "body":
			hasBody = true
		case "comments":
			hasComments = true
		}
	}

	if !hasHeaders {
		t.Error("Expected headers container element")
	}
	if !hasFooters {
		t.Error("Expected footers container element")
	}
	if !hasBody {
		t.Error("Expected body element")
	}
	if !hasComments {
		t.Error("Expected comments container element")
	}

	// Should have relationships
	if len(response.Relationships) == 0 {
		t.Error("Expected relationships between elements")
	}

	// Verify relationship structure
	for _, rel := range response.Relationships {
		if rel.RelationshipID == "" {
			t.Error("Relationship should have ID")
		}
		if rel.SourceElementID == "" {
			t.Error("Relationship should have source")
		}
		if rel.TargetElementID == "" {
			t.Error("Relationship should have target")
		}
		if rel.RelationshipType == "" {
			t.Error("Relationship should have type")
		}
		if rel.Confidence != 1.0 {
			t.Errorf("Expected confidence 1.0, got %f", rel.Confidence)
		}
	}
}

func TestDocxParser_HeaderDetection(t *testing.T) {
	parser := NewDocxParser()

	testAssets := "../../../tests/assets"
	testFile := filepath.Join(testAssets, "test_sample.docx")

	request := DocxParseRequest{
		ID:       "header_test",
		Content:  testFile,
		Metadata: map[string]interface{}{"filename": "test_sample.docx"},
	}

	response, err := parser.Parse(request)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Count headers at different levels
	headerCount := 0
	headerLevels := make(map[int]int)

	for _, elem := range response.Elements {
		if elem.ElementType == "header" {
			headerCount++
			if level, ok := elem.Metadata["level"].(int); ok {
				headerLevels[level]++
			}
		}
	}

	if headerCount == 0 {
		t.Error("Expected at least one header element")
	}

	// Headers should have valid levels (1-6)
	for level := range headerLevels {
		if level < 1 || level > 6 {
			t.Errorf("Invalid header level: %d", level)
		}
	}

	// Log header distribution for debugging
	t.Logf("Found %d headers with distribution: %v", headerCount, headerLevels)
}

func TestDocxParser_TableParsing(t *testing.T) {
	parser := NewDocxParser()

	testAssets := "../../../tests/assets"
	testFile := filepath.Join(testAssets, "test_sample.docx")

	request := DocxParseRequest{
		ID:       "table_test",
		Content:  testFile,
		Metadata: map[string]interface{}{"filename": "test_sample.docx"},
	}

	response, err := parser.Parse(request)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Count table-related elements
	tableCount := 0
	rowCount := 0
	cellCount := 0
	headerCellCount := 0

	for _, elem := range response.Elements {
		switch elem.ElementType {
		case "table":
			tableCount++
			// Verify table metadata
			if rows, ok := elem.Metadata["rows"].(int); ok {
				if rows <= 0 {
					t.Errorf("Table should have positive row count, got %d", rows)
				}
			}
			if cols, ok := elem.Metadata["columns"].(int); ok {
				if cols <= 0 {
					t.Errorf("Table should have positive column count, got %d", cols)
				}
			}
		case "table_row":
			rowCount++
		case "table_cell":
			cellCount++
		case "table_header":
			headerCellCount++
		}
	}

	if tableCount == 0 {
		t.Log("No tables found in test document (may be expected)")
	} else {
		t.Logf("Found %d tables, %d rows, %d cells (%d headers)",
			tableCount, rowCount, cellCount, headerCellCount)

		// If we have tables, we should have rows and cells
		if rowCount == 0 {
			t.Error("Tables should have rows")
		}
		if cellCount == 0 && headerCellCount == 0 {
			t.Error("Tables should have cells")
		}
	}
}

func TestDocxParser_MergedCells(t *testing.T) {
	parser := NewDocxParser()

	testAssets := "../../../tests/assets"
	testFile := filepath.Join(testAssets, "test_merged_cells.docx")

	request := DocxParseRequest{
		ID:       "merged_cells_test",
		Content:  testFile,
		Metadata: map[string]interface{}{"filename": "test_merged_cells.docx"},
	}

	response, err := parser.Parse(request)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Look for merged cells
	mergedCellCount := 0
	hasColspan := false
	hasRowspan := false

	for _, elem := range response.Elements {
		if elem.ElementType == "merged_cell" {
			mergedCellCount++

			// Check for colspan
			if colspan, ok := elem.Metadata["colspan"].(int); ok && colspan > 1 {
				hasColspan = true
				t.Logf("Found cell with colspan=%d", colspan)
			}

			// Check for rowspan
			if rowspan, ok := elem.Metadata["rowspan"].(int); ok && rowspan > 1 {
				hasRowspan = true
				t.Logf("Found cell with rowspan=%d", rowspan)
			}
		}
	}

	t.Logf("Found %d merged cells (colspan=%v, rowspan=%v)",
		mergedCellCount, hasColspan, hasRowspan)

	// The test file should have merged cells
	if mergedCellCount == 0 {
		t.Log("Warning: No merged cells found in test_merged_cells.docx")
	}
}

func TestDocxParser_LinkExtraction(t *testing.T) {
	parser := NewDocxParser()

	testAssets := "../../../tests/assets"
	testFile := filepath.Join(testAssets, "test_sample.docx")

	request := DocxParseRequest{
		ID:       "link_test",
		Content:  testFile,
		Metadata: map[string]interface{}{"filename": "test_sample.docx"},
	}

	response, err := parser.Parse(request)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Check if links were extracted
	if len(response.Links) > 0 {
		t.Logf("Found %d links", len(response.Links))

		for i, link := range response.Links {
			if link.SourceID == "" {
				t.Errorf("Link %d should have source ID", i)
			}
			if link.LinkTarget == "" {
				t.Errorf("Link %d should have target", i)
			}
			if link.LinkType == "" {
				t.Errorf("Link %d should have type", i)
			}
			t.Logf("Link %d: %s -> %s (type: %s)", i, link.LinkText, link.LinkTarget, link.LinkType)
		}
	} else {
		t.Log("No links found in document (may be expected)")
	}
}

func TestDocxParser_DocumentMetadata(t *testing.T) {
	parser := NewDocxParser()

	testAssets := "../../../tests/assets"
	testFile := filepath.Join(testAssets, "test_sample.docx")

	request := DocxParseRequest{
		ID:       "metadata_test",
		Content:  testFile,
		Metadata: map[string]interface{}{"filename": "test_sample.docx"},
	}

	response, err := parser.Parse(request)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Check document metadata
	metadata, ok := response.Document["metadata"].(map[string]interface{})
	if !ok {
		t.Fatal("Document should have metadata map")
	}

	// Check for extracted statistics
	if paragraphCount, ok := metadata["paragraph_count"].(int); ok {
		if paragraphCount < 0 {
			t.Error("Paragraph count should be non-negative")
		}
		t.Logf("Paragraph count: %d", paragraphCount)
	}

	if tableCount, ok := metadata["table_count"].(int); ok {
		if tableCount < 0 {
			t.Error("Table count should be non-negative")
		}
		t.Logf("Table count: %d", tableCount)
	}

	if wordCount, ok := metadata["word_count"].(int); ok {
		if wordCount < 0 {
			t.Error("Word count should be non-negative")
		}
		t.Logf("Word count: %d", wordCount)
	}

	if pageCount, ok := metadata["page_count"].(int); ok {
		if pageCount <= 0 {
			t.Error("Page count should be positive")
		}
		t.Logf("Estimated page count: %d", pageCount)
	}

	// Check content hash
	if contentHash, ok := response.Document["content_hash"].(string); ok {
		if contentHash == "" {
			t.Error("Content hash should not be empty")
		}
		if len(contentHash) != 32 { // MD5 hash is 32 hex characters
			t.Errorf("Expected 32-character MD5 hash, got %d characters", len(contentHash))
		}
	}
}

func TestDocxParser_ContentPreviewTruncation(t *testing.T) {
	parser := NewDocxParser()
	parser.MaxContentPreview = 50 // Set smaller limit for testing

	testAssets := "../../../tests/assets"
	testFile := filepath.Join(testAssets, "test_sample.docx")

	request := DocxParseRequest{
		ID:       "truncation_test",
		Content:  testFile,
		Metadata: map[string]interface{}{"filename": "test_sample.docx"},
	}

	response, err := parser.Parse(request)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Check that content previews are truncated
	for _, elem := range response.Elements {
		if len(elem.ContentPreview) > parser.MaxContentPreview {
			t.Errorf("Content preview exceeds max length: %d > %d",
				len(elem.ContentPreview), parser.MaxContentPreview)
		}
	}
}

func TestDocxParser_ElementOrdering(t *testing.T) {
	parser := NewDocxParser()

	testAssets := "../../../tests/assets"
	testFile := filepath.Join(testAssets, "test_sample.docx")

	request := DocxParseRequest{
		ID:       "ordering_test",
		Content:  testFile,
		Metadata: map[string]interface{}{"filename": "test_sample.docx"},
	}

	response, err := parser.Parse(request)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Verify element ordering is sequential
	seenOrders := make(map[int]bool)
	for _, elem := range response.Elements {
		if elem.ElementOrder < 0 {
			t.Errorf("Element order should be non-negative, got %d", elem.ElementOrder)
		}
		if elem.DocumentOrder < 0 {
			t.Errorf("Document order should be non-negative, got %d", elem.DocumentOrder)
		}
		if seenOrders[elem.ElementOrder] {
			t.Errorf("Duplicate element order: %d", elem.ElementOrder)
		}
		seenOrders[elem.ElementOrder] = true
	}

	// Root should be first (order 0)
	if len(response.Elements) > 0 {
		if response.Elements[0].ElementOrder != 0 {
			t.Errorf("Root element should have order 0, got %d", response.Elements[0].ElementOrder)
		}
	}
}

func TestDocxParser_ToParseResult(t *testing.T) {
	parser := NewDocxParser()

	testAssets := "../../../tests/assets"
	testFile := filepath.Join(testAssets, "test_sample.docx")

	request := DocxParseRequest{
		ID:       "parse_result_test",
		Content:  testFile,
		Metadata: map[string]interface{}{"filename": "test_sample.docx"},
	}

	response, err := parser.Parse(request)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Convert to standard ParseResult format
	result := response.ToParseResult()

	if result.Document.ID != "parse_result_test" {
		t.Errorf("Expected document ID 'parse_result_test', got %v", result.Document.ID)
	}
	if result.Document.DocType != "docx" {
		t.Errorf("Expected doc type 'docx', got %v", result.Document.DocType)
	}

	// Should have same number of elements, relationships, and links
	if len(result.Elements) != len(response.Elements) {
		t.Errorf("Element count mismatch: %d vs %d", len(result.Elements), len(response.Elements))
	}
	if len(result.Relationships) != len(response.Relationships) {
		t.Errorf("Relationship count mismatch: %d vs %d", len(result.Relationships), len(response.Relationships))
	}
	if len(result.Links) != len(response.Links) {
		t.Errorf("Link count mismatch: %d vs %d", len(result.Links), len(response.Links))
	}

	// Verify element conversion
	for i, elem := range result.Elements {
		if elem.ElementID != response.Elements[i].ElementID {
			t.Errorf("Element ID mismatch at index %d", i)
		}
		if elem.ElementType != response.Elements[i].ElementType {
			t.Errorf("Element type mismatch at index %d", i)
		}
	}
}

func TestDocxParser_RealWorldDocument(t *testing.T) {
	parser := NewDocxParser()

	testAssets := "../../../tests/assets"
	testFile := filepath.Join(testAssets, "Cash Management Research Report.docx")

	request := DocxParseRequest{
		ID:       "real_world_test",
		Content:  testFile,
		Metadata: map[string]interface{}{"filename": "Cash Management Research Report.docx"},
	}

	response, err := parser.Parse(request)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Real-world document should have substantial content
	if len(response.Elements) < 10 {
		t.Errorf("Expected more elements in real-world document, got %d", len(response.Elements))
	}

	// Count different element types
	elementTypes := make(map[string]int)
	for _, elem := range response.Elements {
		elementTypes[elem.ElementType]++
	}

	t.Logf("Real-world document analysis:")
	t.Logf("  Total elements: %d", len(response.Elements))
	t.Logf("  Element types: %v", elementTypes)
	t.Logf("  Relationships: %d", len(response.Relationships))
	t.Logf("  Links: %d", len(response.Links))

	// Should have paragraphs
	if elementTypes["paragraph"] == 0 {
		t.Error("Expected paragraphs in real-world document")
	}
}
