package parser

import (
	"strings"
	"testing"
)

func TestMarkdownParser_BasicParsing(t *testing.T) {
	parser := NewMarkdownParser()

	request := MarkdownParseRequest{
		ID: "test_doc",
		Content: `# Header 1

This is a paragraph with some content.

## Header 2

Another paragraph here.

- List item 1
- List item 2

> This is a blockquote

` + "```go" + `
fmt.Println("Hello, World!")
` + "```",
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
	if response.Document["doc_type"] != "markdown" {
		t.Errorf("Expected doc_type 'markdown', got %v", response.Document["doc_type"])
	}

	// Should have multiple elements: root + headers + paragraphs + list + blockquote + code
	if len(response.Elements) < 6 {
		t.Errorf("Expected at least 6 elements, got %d", len(response.Elements))
	}

	// Check root element
	rootElement := response.Elements[0]
	if rootElement.ElementType != MarkdownElementTypeRoot {
		t.Errorf("Expected root element type, got %v", rootElement.ElementType)
	}
	if rootElement.ParentID != "" {
		t.Errorf("Root element should not have parent, got %v", rootElement.ParentID)
	}

	// Check for different element types
	elementTypes := make(map[MarkdownElementType]int)
	for _, element := range response.Elements {
		elementTypes[element.ElementType]++
	}

	if elementTypes[MarkdownElementTypeHeader] < 2 {
		t.Errorf("Expected at least 2 headers, got %d", elementTypes[MarkdownElementTypeHeader])
	}
	if elementTypes[MarkdownElementTypeParagraph] < 2 {
		t.Errorf("Expected at least 2 paragraphs, got %d", elementTypes[MarkdownElementTypeParagraph])
	}
	if elementTypes[MarkdownElementTypeList] < 1 {
		t.Errorf("Expected at least 1 list, got %d", elementTypes[MarkdownElementTypeList])
	}
	if elementTypes[MarkdownElementTypeBlockquote] < 1 {
		t.Errorf("Expected at least 1 blockquote, got %d", elementTypes[MarkdownElementTypeBlockquote])
	}
	if elementTypes[MarkdownElementTypeCodeBlock] < 1 {
		t.Errorf("Expected at least 1 code block, got %d", elementTypes[MarkdownElementTypeCodeBlock])
	}

	// Check relationships
	if len(response.Relationships) == 0 {
		t.Error("Expected relationships between elements")
	}

	// All non-root elements should have relationships to root
	for _, rel := range response.Relationships {
		if rel.SourceElementID != rootElement.ElementID {
			t.Errorf("Expected relationship source to be root element")
		}
		if rel.RelationshipType != "contains" {
			t.Errorf("Expected 'contains' relationship, got %v", rel.RelationshipType)
		}
	}
}

func TestMarkdownParser_FrontMatter(t *testing.T) {
	parser := NewMarkdownParser()

	request := MarkdownParseRequest{
		ID: "front_matter_test",
		Content: `---
title: Test Document
author: Test Author
date: 2024-01-15
tags: [test, markdown]
---

# Main Content

This is the main content after front matter.`,
		Metadata: map[string]interface{}{
			"source": "test",
		},
	}

	response, err := parser.Parse(request)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Check document metadata includes front matter
	metadata, ok := response.Document["metadata"].(map[string]interface{})
	if !ok {
		t.Fatal("Document metadata should be a map")
	}

	if !metadata["has_front_matter"].(bool) {
		t.Error("Expected has_front_matter to be true")
	}

	frontMatter, exists := metadata["front_matter"]
	if !exists {
		t.Error("Expected front_matter in document metadata")
	}

	frontMatterMap, ok := frontMatter.(map[string]interface{})
	if !ok {
		t.Fatal("Front matter should be a map")
	}

	if frontMatterMap["title"] != "Test Document" {
		t.Errorf("Expected title 'Test Document', got %v", frontMatterMap["title"])
	}

	// Check for front matter element
	hasFrontMatterElement := false
	for _, element := range response.Elements {
		if element.ElementType == MarkdownElementTypeFrontMatter {
			hasFrontMatterElement = true
			// Check metadata includes parsed YAML
			if element.Metadata["title"] != "Test Document" {
				t.Errorf("Expected title in front matter element metadata")
			}
			break
		}
	}

	if !hasFrontMatterElement {
		t.Error("Expected front matter element")
	}
}

func TestMarkdownParser_NoFrontMatter(t *testing.T) {
	parser := NewMarkdownParser()

	request := MarkdownParseRequest{
		ID: "no_front_matter_test",
		Content: `# Regular Markdown

No front matter here, just regular content.`,
		Metadata: map[string]interface{}{
			"source": "test",
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

	if metadata["has_front_matter"].(bool) {
		t.Error("Expected has_front_matter to be false")
	}

	// Should not have front matter element
	for _, element := range response.Elements {
		if element.ElementType == MarkdownElementTypeFrontMatter {
			t.Error("Should not have front matter element")
		}
	}
}

func TestMarkdownParser_Headers(t *testing.T) {
	parser := NewMarkdownParser()

	request := MarkdownParseRequest{
		ID: "headers_test",
		Content: `# Level 1 Header
## Level 2 Header
### Level 3 Header
#### Level 4 Header
##### Level 5 Header
###### Level 6 Header`,
		Metadata: map[string]interface{}{
			"source": "test",
		},
	}

	response, err := parser.Parse(request)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Find header elements
	headers := []MarkdownElement{}
	for _, element := range response.Elements {
		if element.ElementType == MarkdownElementTypeHeader {
			headers = append(headers, element)
		}
	}

	if len(headers) != 6 {
		t.Errorf("Expected 6 headers, got %d", len(headers))
	}

	// Check header levels
	expectedLevels := []int{1, 2, 3, 4, 5, 6}
	for i, header := range headers {
		level, ok := header.Metadata["level"].(int)
		if !ok {
			t.Fatal("Header should have level metadata")
		}
		if level != expectedLevels[i] {
			t.Errorf("Expected header level %d, got %d", expectedLevels[i], level)
		}
	}
}

func TestMarkdownParser_CodeBlocks(t *testing.T) {
	parser := NewMarkdownParser()

	request := MarkdownParseRequest{
		ID: "code_test",
		Content: "```go\nfmt.Println(\"Hello\")\n```\n\n```python\nprint('Hello')\n```\n\n```\nNo language specified\n```",
		Metadata: map[string]interface{}{
			"source": "test",
		},
	}

	response, err := parser.Parse(request)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Find code block elements
	codeBlocks := []MarkdownElement{}
	for _, element := range response.Elements {
		if element.ElementType == MarkdownElementTypeCodeBlock {
			codeBlocks = append(codeBlocks, element)
		}
	}

	if len(codeBlocks) != 3 {
		t.Errorf("Expected 3 code blocks, got %d", len(codeBlocks))
	}

	// Check languages
	expectedLanguages := []string{"go", "python", ""}
	for i, codeBlock := range codeBlocks {
		language, ok := codeBlock.Metadata["language"].(string)
		if !ok {
			t.Fatal("Code block should have language metadata")
		}
		if language != expectedLanguages[i] {
			t.Errorf("Expected language '%s', got '%s'", expectedLanguages[i], language)
		}
	}

	// Check content
	if !strings.Contains(codeBlocks[0].Text, "fmt.Println") {
		t.Error("Expected Go code in first code block")
	}
	if !strings.Contains(codeBlocks[1].Text, "print") {
		t.Error("Expected Python code in second code block")
	}
}

func TestMarkdownParser_Lists(t *testing.T) {
	parser := NewMarkdownParser()

	request := MarkdownParseRequest{
		ID: "list_test",
		Content: `- Item 1
- Item 2
- Item 3

1. Numbered item 1
2. Numbered item 2
3. Numbered item 3`,
		Metadata: map[string]interface{}{
			"source": "test",
		},
	}

	response, err := parser.Parse(request)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Find list elements
	lists := []MarkdownElement{}
	for _, element := range response.Elements {
		if element.ElementType == MarkdownElementTypeList {
			lists = append(lists, element)
		}
	}

	if len(lists) != 2 {
		t.Errorf("Expected 2 lists, got %d", len(lists))
	}

	// Check item counts
	for _, list := range lists {
		itemCount, ok := list.Metadata["item_count"].(int)
		if !ok {
			t.Fatal("List should have item_count metadata")
		}
		if itemCount != 3 {
			t.Errorf("Expected 3 items per list, got %d", itemCount)
		}
	}
}

func TestMarkdownParser_Blockquotes(t *testing.T) {
	parser := NewMarkdownParser()

	request := MarkdownParseRequest{
		ID: "blockquote_test",
		Content: `> This is a blockquote
> with multiple lines
> of content.

> Another blockquote`,
		Metadata: map[string]interface{}{
			"source": "test",
		},
	}

	response, err := parser.Parse(request)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Find blockquote elements
	blockquotes := []MarkdownElement{}
	for _, element := range response.Elements {
		if element.ElementType == MarkdownElementTypeBlockquote {
			blockquotes = append(blockquotes, element)
		}
	}

	if len(blockquotes) != 2 {
		// Debug output
		t.Logf("All elements:")
		for i, elem := range response.Elements {
			t.Logf("  %d: Type=%s, Content preview: %s", i, elem.ElementType, elem.ContentPreview)
		}
		t.Errorf("Expected 2 blockquotes, got %d", len(blockquotes))
	}

	// Check content (should have > markers removed)
	firstQuote := blockquotes[0].Text
	if strings.Contains(firstQuote, ">") {
		t.Error("Blockquote content should not contain > markers")
	}
	if !strings.Contains(firstQuote, "multiple lines") {
		t.Error("Expected blockquote content to span multiple lines")
	}
}

func TestMarkdownParser_LinkExtraction(t *testing.T) {
	parser := NewMarkdownParser()

	request := MarkdownParseRequest{
		ID: "link_test",
		Content: `# Header with [Link](https://example.com)

Paragraph with [another link](https://test.org) and email@example.com.

Also has wiki link [[Internal Page]] and plain URL https://plain.example.com.`,
		Metadata: map[string]interface{}{
			"source": "test",
		},
	}

	response, err := parser.Parse(request)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should extract multiple types of links
	if len(response.Links) < 4 {
		t.Errorf("Expected at least 4 links, got %d", len(response.Links))
	}

	// Check link types
	linkTypes := make(map[string]int)
	for _, link := range response.Links {
		linkTypes[link.LinkType]++
	}

	if linkTypes["url"] < 1 {
		t.Error("Expected at least one URL link")
	}
	if linkTypes["email"] < 1 {
		t.Error("Expected at least one email link")
	}
	if linkTypes["wiki"] < 1 {
		t.Error("Expected at least one wiki link")
	}

	// Check link targets
	hasExample := false
	hasEmail := false
	hasWiki := false
	for _, link := range response.Links {
		if strings.Contains(link.LinkTarget, "example.com") {
			hasExample = true
		}
		if strings.Contains(link.LinkTarget, "mailto:") {
			hasEmail = true
		}
		if link.LinkTarget == "Internal Page" {
			hasWiki = true
		}
	}

	if !hasExample {
		t.Error("Expected link to example.com")
	}
	if !hasEmail {
		t.Error("Expected mailto link")
	}
	if !hasWiki {
		t.Error("Expected wiki link")
	}
}

func TestMarkdownParser_Tables(t *testing.T) {
	parser := NewMarkdownParser()

	request := MarkdownParseRequest{
		ID: "table_test",
		Content: `| Header 1 | Header 2 | Header 3 |
|----------|----------|----------|
| Cell 1   | Cell 2   | Cell 3   |
| Cell 4   | Cell 5   | Cell 6   |`,
		Metadata: map[string]interface{}{
			"source": "test",
		},
	}

	response, err := parser.Parse(request)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Find table elements
	tables := []MarkdownElement{}
	for _, element := range response.Elements {
		if element.ElementType == MarkdownElementTypeTable {
			tables = append(tables, element)
		}
	}

	if len(tables) != 1 {
		t.Errorf("Expected 1 table, got %d", len(tables))
	}

	table := tables[0]
	rowCount, ok := table.Metadata["row_count"].(int)
	if !ok {
		t.Fatal("Table should have row_count metadata")
	}
	if rowCount != 2 { // Data rows (excluding header separator)
		t.Errorf("Expected 2 data rows, got %d", rowCount)
	}

	colCount, ok := table.Metadata["column_count"].(int)
	if !ok {
		t.Fatal("Table should have column_count metadata")
	}
	if colCount != 3 { // 3 actual columns
		t.Errorf("Expected 3 columns, got %d", colCount)
	}
}

func TestMarkdownParser_EmptyContent(t *testing.T) {
	parser := NewMarkdownParser()

	request := MarkdownParseRequest{
		ID:      "empty_test",
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

func TestMarkdownParser_NoDocumentID(t *testing.T) {
	parser := NewMarkdownParser()

	request := MarkdownParseRequest{
		ID:      "",
		Content: "# Test",
		Metadata: map[string]interface{}{
			"source": "test",
		},
	}

	_, err := parser.Parse(request)
	if err == nil {
		t.Error("Expected error for missing document ID")
	}
}

func TestMarkdownParser_DocumentMetadata(t *testing.T) {
	parser := NewMarkdownParser()

	request := MarkdownParseRequest{
		ID: "metadata_test",
		Content: `# Header

Paragraph content.

- List item

` + "```" + `
Code block
` + "```",
		Metadata: map[string]interface{}{
			"source":   "test_source",
			"filename": "test.md",
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
	if metadata["header_count"] == nil {
		t.Error("Expected header_count in metadata")
	}
	if metadata["code_block_count"] == nil {
		t.Error("Expected code_block_count in metadata")
	}
	if metadata["list_item_count"] == nil {
		t.Error("Expected list_item_count in metadata")
	}

	if metadata["source"] != "test_source" {
		t.Errorf("Expected source 'test_source', got %v", metadata["source"])
	}
	if metadata["filename"] != "test.md" {
		t.Errorf("Expected filename 'test.md', got %v", metadata["filename"])
	}
	if metadata["content_type"] != "text/markdown" {
		t.Errorf("Expected content_type 'text/markdown', got %v", metadata["content_type"])
	}
}

func TestMarkdownParser_ToJSON(t *testing.T) {
	parser := NewMarkdownParser()

	request := MarkdownParseRequest{
		ID:      "json_test",
		Content: "# Test\n\nParagraph content.",
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