package parser

import (
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func TestNewHTMLParser(t *testing.T) {
	parser := NewHTMLParser()
	if parser == nil {
		t.Fatal("Failed to create parser")
	}
	if parser.MaxContentPreview != 100 {
		t.Error("MaxContentPreview not set correctly")
	}
	if !parser.ExtractDates {
		t.Error("ExtractDates not set correctly")
	}
	if !parser.EnableCaching {
		t.Error("EnableCaching not set correctly")
	}
}

func TestParseSimpleHTML(t *testing.T) {
	parser := NewHTMLParser()

	request := ParseRequest{
		ID:      "test_doc",
		Content: "<html><body><h1>Test Header</h1><p>Test paragraph</p></body></html>",
		Metadata: map[string]interface{}{
			"source": "test",
		},
	}

	response, err := parser.Parse("test-doc", htmlContent)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Check document
	if result.Document."doc_id"] != "test_doc" {
		t.Error("Document ID not set correctly")
	}
	if result.Document."doc_type"] != "html" {
		t.Error("Document type not set correctly")
	}

	// Should have at least root element
	if len(response.Elements) < 1 {
		t.Fatal("No elements found")
	}

	// Root element should be first
	root := response.Elements[0]
	if root.ElementType != HTMLElementTypeRoot {
		t.Error("First element should be root")
	}
	if root.ElementID == "" {
		t.Error("Root element should have ID")
	}
}

func TestParseHTMLElements(t *testing.T) {
	parser := NewHTMLParser()

	html := `<html>
		<body>
			<h1>Main Header</h1>
			<h2>Sub Header</h2>
			<p>This is a paragraph with some text.</p>
			<ul>
				<li>First item</li>
				<li>Second item</li>
			</ul>
			<table>
				<tr>
					<th>Header 1</th>
					<th>Header 2</th>
				</tr>
				<tr>
					<td>Cell 1</td>
					<td>Cell 2</td>
				</tr>
			</table>
		</body>
	</html>`

	request := ParseRequest{
		ID:      "test_elements",
		Content: html,
	}

	response, err := parser.Parse("test-doc", htmlContent)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Check for different element types
	elementTypes := make(map[HTMLElementType]int)
	for _, element := range response.Elements {
		elementTypes[element.ElementType]++
	}

	// Should have various element types
	expectedTypes := []HTMLElementType{
		HTMLElementTypeRoot,
		HTMLElementTypeHeader,
		HTMLElementTypeParagraph,
		HTMLElementTypeList,
		HTMLElementTypeListItem,
		HTMLElementTypeTable,
		HTMLElementTypeTableRow,
		HTMLElementTypeTableHeader,
		HTMLElementTypeTableCell,
	}

	for _, expectedType := range expectedTypes {
		if elementTypes[expectedType] == 0 {
			t.Errorf("Expected element type %s not found", expectedType)
		}
	}
}

func TestGetElementType(t *testing.T) {
	parser := NewHTMLParser()

	tests := []struct {
		tagName  string
		expected HTMLElementType
	}{
		{"h1", HTMLElementTypeHeader},
		{"h2", HTMLElementTypeHeader},
		{"h6", HTMLElementTypeHeader},
		{"p", HTMLElementTypeParagraph},
		{"ul", HTMLElementTypeList},
		{"ol", HTMLElementTypeList},
		{"li", HTMLElementTypeListItem},
		{"table", HTMLElementTypeTable},
		{"tr", HTMLElementTypeTableRow},
		{"th", HTMLElementTypeTableHeader},
		{"td", HTMLElementTypeTableCell},
		{"img", HTMLElementTypeImage},
		{"pre", HTMLElementTypeCodeBlock},
		{"code", HTMLElementTypeCodeBlock},
		{"blockquote", HTMLElementTypeBlockquote},
		{"div", HTMLElementTypeDiv},
		{"article", HTMLElementTypeArticle},
		{"section", HTMLElementTypeSection},
		{"span", HTMLElementTypeSpan},
		{"body", HTMLElementTypeBody},
	}

	for _, test := range tests {
		result := parser.getElementType(test.tagName)
		if result != test.expected {
			t.Errorf("getElementType(%s) = %s, want %s", test.tagName, result, test.expected)
		}
	}
}

func TestGetElementTypeNamespaced(t *testing.T) {
	parser := NewHTMLParser()

	// Test namespace-prefixed elements (XBRL, iXBRL, etc.)
	tests := []struct {
		tagName  string
		expected HTMLElementType
	}{
		{"ix:nonnumeric", HTMLElementType("ix_nonnumeric")},
		{"xbrl:context", HTMLElementType("xbrl_context")},
		{"custom:element", HTMLElementType("custom_element")},
	}

	for _, test := range tests {
		result := parser.getElementType(test.tagName)
		if result != test.expected {
			t.Errorf("getElementType(%s) = %s, want %s", test.tagName, result, test.expected)
		}
	}
}

func TestShouldSkipElement(t *testing.T) {
	parser := NewHTMLParser()

	skipTags := []string{"script", "style", "noscript", "meta", "link", "title"}
	keepTags := []string{"div", "p", "h1", "table", "span"}

	for _, tag := range skipTags {
		if !parser.shouldSkipElement(tag) {
			t.Errorf("shouldSkipElement(%s) should return true", tag)
		}
	}

	for _, tag := range keepTags {
		if parser.shouldSkipElement(tag) {
			t.Errorf("shouldSkipElement(%s) should return false", tag)
		}
	}
}

func TestExtractTextContent(t *testing.T) {
	parser := NewHTMLParser()

	html := "<div>Hello <span>world</span> test</div>"
	doc, err := parseHTML(html)
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}

	text := parser.extractTextContent(doc)
	if !strings.Contains(text, "Hello") || !strings.Contains(text, "world") || !strings.Contains(text, "test") {
		t.Errorf("extractTextContent failed, got: %s", text)
	}
}

func TestExtractLinks(t *testing.T) {
	parser := NewHTMLParser()

	html := `<html><body>
		<a href="http://example.com">Example Link</a>
		<a href="/internal/page">Internal Link</a>
		<a href="mailto:test@example.com">Email Link</a>
	</body></html>`

	request := ParseRequest{
		ID:      "test_links",
		Content: html,
	}

	response, err := parser.Parse("test-doc", htmlContent)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(response.Links) != 3 {
		t.Errorf("Expected 3 links, got %d", len(response.Links))
	}

	// Check link properties
	for _, link := range response.Links {
		if link.LinkType != "html" {
			t.Error("Link type should be 'html'")
		}
		if link.LinkTarget == "" {
			t.Error("Link target should not be empty")
		}
	}
}

func TestTruncateContent(t *testing.T) {
	parser := NewHTMLParser()
	parser.MaxContentPreview = 10

	tests := []struct {
		input    string
		expected string
	}{
		{"short", "short"},
		{"this is a long text that should be truncated", "this is..."},
		{"exactly10c", "exactly10c"},
		{"", ""},
	}

	for _, test := range tests {
		result := parser.truncateContent(test.input)
		if result != test.expected {
			t.Errorf("truncateContent(%s) = %s, want %s", test.input, result, test.expected)
		}
	}
}

func TestGenerateHash(t *testing.T) {
	parser := NewHTMLParser()

	content1 := "test content"
	content2 := "different content"
	content3 := "test content" // same as content1

	hash1 := parser.generateHash(content1)
	hash2 := parser.generateHash(content2)
	hash3 := parser.generateHash(content3)

	if hash1 == hash2 {
		t.Error("Different content should have different hashes")
	}
	if hash1 != hash3 {
		t.Error("Same content should have same hash")
	}
	if len(hash1) != 32 {
		t.Error("MD5 hash should be 32 characters")
	}
}

func TestCreateContentLocation(t *testing.T) {
	parser := NewHTMLParser()

	location := parser.createContentLocation("source.html", string(HTMLElementTypeParagraph), "p:nth-of-type(1)")

	if location["source"] != "source.html" {
		t.Error("Source not set correctly")
	}
	if location["type"] != "paragraph" {
		t.Error("Type not set correctly")
	}
	if location["selector"] != "p:nth-of-type(1)" {
		t.Error("Selector not set correctly")
	}
}

func TestJSONSerialization(t *testing.T) {
	parser := NewHTMLParser()

	request := ParseRequest{
		ID:      "test_json",
		Content: "<html><body><p>Test</p></body></html>",
		Metadata: map[string]interface{}{
			"source": "test",
		},
	}

	response, err := parser.Parse("test-doc", htmlContent)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Test response to JSON
	jsonStr, err := response.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	if !strings.Contains(jsonStr, "test_json") {
		t.Error("JSON should contain document ID")
	}

	// Test request from JSON
	requestJSON := `{"id":"test","content":"<p>test</p>","metadata":{"key":"value"}}`
	var newRequest ParseRequest
	err = newRequest.FromJSON(requestJSON)
	if err != nil {
		t.Fatalf("FromJSON failed: %v", err)
	}

	if newRequest.ID != "test" {
		t.Error("ID not parsed correctly from JSON")
	}
	if newRequest.Content != "<p>test</p>" {
		t.Error("Content not parsed correctly from JSON")
	}
}

func TestRelationships(t *testing.T) {
	parser := NewHTMLParser()

	html := `<html><body>
		<div>
			<p>Paragraph 1</p>
			<p>Paragraph 2</p>
		</div>
	</body></html>`

	request := ParseRequest{
		ID:      "test_relationships",
		Content: html,
	}

	response, err := parser.Parse("test-doc", htmlContent)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should have relationships between parent and child elements
	if len(response.Relationships) == 0 {
		t.Error("Should have relationships")
	}

	// Check relationship properties
	for _, rel := range response.Relationships {
		if rel.RelationshipType != RelationshipContains {
			t.Error("Expected 'contains' relationship")
		}
		if rel.Confidence != 1.0 {
			t.Error("Expected confidence of 1.0")
		}
		if rel.SourceElementID == "" || rel.TargetElementID == "" {
			t.Error("Relationship should have source and target IDs")
		}
	}
}

func TestComplexHTML(t *testing.T) {
	parser := NewHTMLParser()

	html := `<!DOCTYPE html>
	<html lang="en">
	<head>
		<title>Test Page</title>
		<meta charset="UTF-8">
	</head>
	<body>
		<header>
			<h1 id="main-title" class="title primary">Main Title</h1>
			<nav>
				<ul>
					<li><a href="/home">Home</a></li>
					<li><a href="/about">About</a></li>
				</ul>
			</nav>
		</header>
		<main>
			<article class="content">
				<h2>Article Title</h2>
				<p class="intro">Introduction paragraph with <strong>bold</strong> text.</p>
				<section>
					<h3>Section Title</h3>
					<p>Section content with <a href="http://example.com">external link</a>.</p>
					<blockquote cite="http://example.com">
						This is a blockquote with citation.
					</blockquote>
				</section>
			</article>
		</main>
		<footer>
			<p>&copy; 2024 Test Company</p>
		</footer>
	</body>
	</html>`

	request := ParseRequest{
		ID:      "complex_test",
		Content: html,
	}

	response, err := parser.Parse("test-doc", htmlContent)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should have many elements
	if len(response.Elements) < 10 {
		t.Errorf("Expected many elements, got %d", len(response.Elements))
	}

	// Should have links
	if len(response.Links) < 3 {
		t.Errorf("Expected at least 3 links, got %d", len(response.Links))
	}

	// Should have relationships
	if len(response.Relationships) < 5 {
		t.Errorf("Expected many relationships, got %d", len(response.Relationships))
	}

	// Check that we have various element types
	elementTypes := make(map[HTMLElementType]bool)
	for _, element := range response.Elements {
		elementTypes[element.ElementType] = true
	}

	expectedTypes := []HTMLElementType{
		HTMLElementTypeRoot,
		HTMLElementTypeHeader,
		HTMLElementTypeParagraph,
		HTMLElementTypeList,
		HTMLElementTypeListItem,
		HTMLElementTypeBlockquote,
	}

	for _, expectedType := range expectedTypes {
		if !elementTypes[expectedType] {
			t.Errorf("Expected element type %s not found", expectedType)
		}
	}
}

func TestEmptyHTML(t *testing.T) {
	parser := NewHTMLParser()

	request := ParseRequest{
		ID:      "empty_test",
		Content: "",
	}

	response, err := parser.Parse("test-doc", htmlContent)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should have root element plus auto-generated html, head, body
	if len(response.Elements) < 1 {
		t.Error("Should have at least root element")
	}

	if response.Elements[0].ElementType != HTMLElementTypeRoot {
		t.Error("First element should be root")
	}

	// HTML parser automatically creates html/head/body structure even for empty content
	if len(response.Elements) > 1 {
		// Verify we have html and body elements
		hasHTML := false
		hasBody := false
		for _, elem := range response.Elements[1:] {
			if elem.ElementType == HTMLElementType("html") {
				hasHTML = true
			}
			if elem.ElementType == HTMLElementTypeBody {
				hasBody = true
			}
		}
		if !hasHTML || !hasBody {
			t.Error("Expected auto-generated html and body elements")
		}
	}
}

func TestInvalidHTML(t *testing.T) {
	parser := NewHTMLParser()

	// Malformed HTML should still be parseable (Go's html parser is forgiving)
	request := ParseRequest{
		ID:      "invalid_test",
		Content: "<div><p>Unclosed tags</div>",
	}

	response, err := parser.Parse("test-doc", htmlContent)
	if err != nil {
		t.Fatalf("Parse should handle malformed HTML: %v", err)
	}

	// Should have at least root element
	if len(response.Elements) < 1 {
		t.Error("Should have at least root element")
	}
}

// Helper function to parse HTML for testing
func parseHTML(htmlStr string) (*html.Node, error) {
	return html.Parse(strings.NewReader(htmlStr))
}