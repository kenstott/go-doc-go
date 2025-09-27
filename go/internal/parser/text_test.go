package parser

import (
	"testing"
)

func TestTextParser_BasicParsing(t *testing.T) {
	parser := NewTextParser()

	request := TextParseRequest{
		ID: "test_doc",
		Content: `This is the first paragraph.
This is still the first paragraph.

This is the second paragraph.
It has multiple lines too.

This is the third paragraph.`,
		Metadata: map[string]interface{}{
			"source": "test",
		},
	}

	response, err := parser.Parse(request)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Check document structure
	if response.Document["doc_id"] != "test_doc" {
		t.Errorf("Expected doc_id 'test_doc', got %v", response.Document["doc_id"])
	}
	if response.Document["doc_type"] != "text" {
		t.Errorf("Expected doc_type 'text', got %v", response.Document["doc_type"])
	}

	// Should have 4 elements: 1 root + 3 paragraphs
	if len(response.Elements) != 4 {
		t.Errorf("Expected 4 elements, got %d", len(response.Elements))
	}

	// Check root element
	rootElement := response.Elements[0]
	if rootElement.ElementType != TextElementTypeRoot {
		t.Errorf("Expected root element type, got %v", rootElement.ElementType)
	}
	if rootElement.ParentID != "" {
		t.Errorf("Root element should not have parent, got %v", rootElement.ParentID)
	}

	// Check paragraph elements
	for i := 1; i < len(response.Elements); i++ {
		element := response.Elements[i]
		if element.ElementType != TextElementTypeParagraph {
			t.Errorf("Expected paragraph element type, got %v", element.ElementType)
		}
		if element.ParentID != rootElement.ElementID {
			t.Errorf("Paragraph should have root as parent, got %v", element.ParentID)
		}
	}

	// Should have 3 relationships (root contains each paragraph)
	if len(response.Relationships) != 3 {
		t.Errorf("Expected 3 relationships, got %d", len(response.Relationships))
	}

	// Check relationships
	for _, rel := range response.Relationships {
		if rel.SourceElementID != rootElement.ElementID {
			t.Errorf("Relationship source should be root element")
		}
		if rel.RelationshipType != "contains" {
			t.Errorf("Expected 'contains' relationship, got %v", rel.RelationshipType)
		}
		if rel.Confidence != 1.0 {
			t.Errorf("Expected confidence 1.0, got %v", rel.Confidence)
		}
	}
}

func TestTextParser_EmptyContent(t *testing.T) {
	parser := NewTextParser()

	request := TextParseRequest{
		ID:      "empty_doc",
		Content: "",
		Metadata: map[string]interface{}{
			"source": "test",
		},
	}

	_, err := parser.Parse(request)
	if err == nil {
		t.Error("Expected error for empty content")
	}
}

func TestTextParser_NoDocumentID(t *testing.T) {
	parser := NewTextParser()

	request := TextParseRequest{
		ID:      "",
		Content: "Some content",
		Metadata: map[string]interface{}{
			"source": "test",
		},
	}

	_, err := parser.Parse(request)
	if err == nil {
		t.Error("Expected error for missing document ID")
	}
}

func TestTextParser_SingleParagraph(t *testing.T) {
	parser := NewTextParser()

	request := TextParseRequest{
		ID:      "single_para",
		Content: "This is a single paragraph with some content.",
		Metadata: map[string]interface{}{
			"source": "test",
		},
	}

	response, err := parser.Parse(request)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should have 2 elements: 1 root + 1 paragraph
	if len(response.Elements) != 2 {
		t.Errorf("Expected 2 elements, got %d", len(response.Elements))
	}

	// Should have 1 relationship
	if len(response.Relationships) != 1 {
		t.Errorf("Expected 1 relationship, got %d", len(response.Relationships))
	}
}

func TestTextParser_LinkExtraction(t *testing.T) {
	parser := NewTextParser()
	parser.EnableLinkExtraction = true

	request := TextParseRequest{
		ID: "link_test",
		Content: `Visit our website at https://example.com for more info.
Contact us at support@example.com for help.
Check the file at /path/to/document.pdf for details.`,
		Metadata: map[string]interface{}{
			"source": "test",
		},
	}

	response, err := parser.Parse(request)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should extract 3 links: URL, email, file path
	if len(response.Links) < 3 {
		t.Errorf("Expected at least 3 links, got %d", len(response.Links))
	}

	// Check link types
	linkTypes := make(map[string]bool)
	for _, link := range response.Links {
		linkTypes[link.LinkType] = true
	}

	if !linkTypes["url"] {
		t.Error("Expected to find URL link")
	}
	if !linkTypes["email"] {
		t.Error("Expected to find email link")
	}
	if !linkTypes["file"] {
		t.Error("Expected to find file link")
	}
}

func TestTextParser_DateExtraction(t *testing.T) {
	parser := NewTextParser()
	parser.ExtractDates = true

	request := TextParseRequest{
		ID: "date_test",
		Content: `The meeting is scheduled for 2024-01-15.
Please submit by 12/31/2023.
The event was held on January 5, 2024.`,
		Metadata: map[string]interface{}{
			"source": "test",
		},
	}

	response, err := parser.Parse(request)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Check that paragraphs have date metadata
	foundDates := false
	for _, element := range response.Elements {
		if element.ElementType == TextElementTypeParagraph {
			if dates, exists := element.Metadata["dates"]; exists {
				if dateSlice, ok := dates.([]string); ok && len(dateSlice) > 0 {
					foundDates = true
					break
				}
			}
		}
	}

	if !foundDates {
		t.Error("Expected to find dates in paragraph metadata")
	}
}

func TestTextParser_NumberExtraction(t *testing.T) {
	parser := NewTextParser()
	parser.ExtractNumbers = true

	request := TextParseRequest{
		ID: "number_test",
		Content: `The price is 25.99 dollars.
We have 150 items in stock.
The ratio is 3.14159.`,
		Metadata: map[string]interface{}{
			"source": "test",
		},
	}

	response, err := parser.Parse(request)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Check that paragraphs have number metadata
	foundNumbers := false
	for _, element := range response.Elements {
		if element.ElementType == TextElementTypeParagraph {
			if numbers, exists := element.Metadata["numbers"]; exists {
				if numberSlice, ok := numbers.([]interface{}); ok && len(numberSlice) > 0 {
					foundNumbers = true
					break
				}
			}
		}
	}

	if !foundNumbers {
		t.Error("Expected to find numbers in paragraph metadata")
	}
}

func TestTextParser_CustomParagraphSeparator(t *testing.T) {
	parser := NewTextParser()
	parser.ParagraphSeparator = "\n---\n"

	request := TextParseRequest{
		ID: "custom_separator",
		Content: `First section of text.
More content here.
---
Second section of text.
Different content here.
---
Third section.`,
		Metadata: map[string]interface{}{
			"source": "test",
		},
	}

	response, err := parser.Parse(request)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should have 4 elements: 1 root + 3 sections
	if len(response.Elements) != 4 {
		t.Errorf("Expected 4 elements, got %d", len(response.Elements))
	}
}

func TestTextParser_MinParagraphLength(t *testing.T) {
	parser := NewTextParser()
	parser.MinParagraphLength = 20

	request := TextParseRequest{
		ID: "min_length_test",
		Content: `Short.

This is a longer paragraph that meets the minimum length requirement.

Too short.

This is another paragraph that is long enough to be included in the results.`,
		Metadata: map[string]interface{}{
			"source": "test",
		},
	}

	response, err := parser.Parse(request)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should have 3 elements: 1 root + 2 long paragraphs (short ones filtered out)
	if len(response.Elements) != 3 {
		t.Errorf("Expected 3 elements, got %d", len(response.Elements))
	}
}

func TestTextParser_MaxElements(t *testing.T) {
	parser := NewTextParser()
	parser.MaxElements = 3

	request := TextParseRequest{
		ID: "max_elements_test",
		Content: `First paragraph.

Second paragraph.

Third paragraph.

Fourth paragraph.

Fifth paragraph.`,
		Metadata: map[string]interface{}{
			"source": "test",
		},
	}

	response, err := parser.Parse(request)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should be limited to 3 elements total
	if len(response.Elements) > 3 {
		t.Errorf("Expected at most 3 elements, got %d", len(response.Elements))
	}
}

func TestTextParser_ContentPreview(t *testing.T) {
	parser := NewTextParser()
	parser.MaxContentPreview = 20

	longContent := "This is a very long paragraph that should be truncated in the content preview because it exceeds the maximum length."

	request := TextParseRequest{
		ID:      "preview_test",
		Content: longContent,
		Metadata: map[string]interface{}{
			"source": "test",
		},
	}

	response, err := parser.Parse(request)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Check that content preview is truncated
	for _, element := range response.Elements {
		if len(element.ContentPreview) > 23 { // 20 + "..."
			t.Errorf("Content preview too long: %d chars", len(element.ContentPreview))
		}
		if element.ContentPreview != element.Text && !strings.HasSuffix(element.ContentPreview, "...") {
			t.Error("Truncated content should end with '...'")
		}
	}
}

func TestTextParser_DocumentMetadata(t *testing.T) {
	parser := NewTextParser()

	request := TextParseRequest{
		ID: "metadata_test",
		Content: `First paragraph with some words.

Second paragraph with more words and content.

Third paragraph.`,
		Metadata: map[string]interface{}{
			"source":   "test_source",
			"filename": "test.txt",
		},
	}

	response, err := parser.Parse(request)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Check document metadata
	metadata, ok := response.Document["metadata"].(map[string]interface{})
	if !ok {
		t.Fatal("Document metadata should be a map")
	}

	// Check required metadata fields
	if metadata["character_count"] == nil {
		t.Error("Expected character_count in metadata")
	}
	if metadata["word_count"] == nil {
		t.Error("Expected word_count in metadata")
	}
	if metadata["line_count"] == nil {
		t.Error("Expected line_count in metadata")
	}
	if metadata["paragraph_count"] == nil {
		t.Error("Expected paragraph_count in metadata")
	}
	if metadata["source"] != "test_source" {
		t.Errorf("Expected source 'test_source', got %v", metadata["source"])
	}
	if metadata["filename"] != "test.txt" {
		t.Errorf("Expected filename 'test.txt', got %v", metadata["filename"])
	}
}

func TestTextParser_ToJSON(t *testing.T) {
	parser := NewTextParser()

	request := TextParseRequest{
		ID:      "json_test",
		Content: "Simple test content for JSON serialization.",
		Metadata: map[string]interface{}{
			"source": "test",
		},
	}

	response, err := parser.Parse(request)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Test JSON conversion
	jsonStr, err := response.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	if jsonStr == "" {
		t.Error("JSON output should not be empty")
	}

	// Should be valid JSON (basic check)
	if !strings.Contains(jsonStr, "\"document\"") {
		t.Error("JSON should contain document field")
	}
	if !strings.Contains(jsonStr, "\"elements\"") {
		t.Error("JSON should contain elements field")
	}
}