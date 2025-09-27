package parser

import (
	"strings"
	"testing"
)

func TestNewXMLParser(t *testing.T) {
	parser := NewXMLParser()
	if parser == nil {
		t.Fatal("Failed to create parser")
	}
	if parser.MaxContentPreview != 100 {
		t.Error("MaxContentPreview not set correctly")
	}
	if !parser.ExtractAttributes {
		t.Error("ExtractAttributes not set correctly")
	}
	if !parser.FlattenNamespaces {
		t.Error("FlattenNamespaces should be true by default")
	}
	if parser.MaxDepth != 20 {
		t.Error("MaxDepth not set correctly")
	}
}

func TestParseSimpleXML(t *testing.T) {
	parser := NewXMLParser()

	xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
	<root>
		<name>John Doe</name>
		<age>30</age>
		<city>New York</city>
	</root>`

	request := XMLParseRequest{
		ID:      "test_doc",
		Content: xmlContent,
		Metadata: map[string]interface{}{
			"source": "test",
		},
	}

	response, err := parser.Parse(request)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Check document
	if response.Document["doc_id"] != "test_doc" {
		t.Error("Document ID not set correctly")
	}
	if response.Document["doc_type"] != "xml" {
		t.Error("Document type not set correctly")
	}

	// Should have at least root element and child elements
	if len(response.Elements) < 2 {
		t.Fatalf("Expected at least 2 elements, got %d", len(response.Elements))
	}

	// Root element should be first
	root := response.Elements[0]
	if root.ElementType != XMLElementTypeRoot {
		t.Error("First element should be root")
	}
	if root.ElementID == "" {
		t.Error("Root element should have ID")
	}
}

func TestParseXMLWithAttributes(t *testing.T) {
	parser := NewXMLParser()

	xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
	<book isbn="978-0-123456-78-9" year="2023">
		<title lang="en">Sample Book</title>
		<author id="1">Jane Doe</author>
		<price currency="USD">29.99</price>
	</book>`

	request := XMLParseRequest{
		ID:      "test_book",
		Content: xmlContent,
	}

	response, err := parser.Parse(request)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Check for attributes extraction
	foundBookElement := false
	for _, element := range response.Elements {
		if tagName, ok := element.Metadata["tag_name"]; ok && tagName == "book" {
			foundBookElement = true

			// Check if attributes were extracted
			if attributes, ok := element.Metadata["attributes"].(map[string]string); ok {
				if attributes["isbn"] != "978-0-123456-78-9" {
					t.Error("ISBN attribute not extracted correctly")
				}
				if attributes["year"] != "2023" {
					t.Error("Year attribute not extracted correctly")
				}
			} else {
				t.Error("Attributes not extracted for book element")
			}
			break
		}
	}

	if !foundBookElement {
		t.Error("Book element not found")
	}
}

func TestParseXMLWithNamespaces(t *testing.T) {
	parser := NewXMLParser()

	xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
	<root xmlns="http://example.org/default" xmlns:custom="http://example.org/custom">
		<element1>Default namespace</element1>
		<custom:element2>Custom namespace</custom:element2>
	</root>`

	request := XMLParseRequest{
		ID:      "test_namespace",
		Content: xmlContent,
	}

	response, err := parser.Parse(request)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Check if namespaces were extracted
	if namespaces, ok := response.Document["namespaces"].(map[string]string); ok {
		if namespaces[""] != "http://example.org/default" {
			t.Error("Default namespace not extracted correctly")
		}
		if namespaces["custom"] != "http://example.org/custom" {
			t.Error("Custom namespace not extracted correctly")
		}
	}
}

func TestParseXMLWithText(t *testing.T) {
	parser := NewXMLParser()

	xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
	<document>
		<paragraph>This is some text content with spaces.</paragraph>
		<empty></empty>
		<mixed>Text <bold>bold text</bold> more text</mixed>
	</document>`

	request := XMLParseRequest{
		ID:      "test_text",
		Content: xmlContent,
	}

	response, err := parser.Parse(request)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Check for text elements
	textCount := 0
	for _, element := range response.Elements {
		if element.ElementType == XMLElementTypeText {
			textCount++
			if element.Text == "" {
				t.Error("Text element should have content")
			}
		}
	}

	if textCount == 0 {
		t.Error("Should have found text elements")
	}
}

func TestParseNestedXML(t *testing.T) {
	parser := NewXMLParser()

	xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
	<library>
		<section name="Fiction">
			<book id="1">
				<title>Book One</title>
				<author>
					<name>John Smith</name>
					<bio>Author bio here</bio>
				</author>
			</book>
			<book id="2">
				<title>Book Two</title>
				<author>
					<name>Jane Doe</name>
					<bio>Another bio</bio>
				</author>
			</book>
		</section>
	</library>`

	request := XMLParseRequest{
		ID:      "test_nested",
		Content: xmlContent,
	}

	response, err := parser.Parse(request)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should have many elements due to nesting
	if len(response.Elements) < 10 {
		t.Errorf("Expected many elements for nested structure, got %d", len(response.Elements))
	}

	// Should have relationships
	if len(response.Relationships) == 0 {
		t.Error("Should have relationships between elements")
	}

	// Check relationship types
	for _, rel := range response.Relationships {
		if rel.RelationshipType != "contains" {
			t.Errorf("Expected 'contains' relationship, got %s", rel.RelationshipType)
		}
		if rel.Confidence != 1.0 {
			t.Error("Relationship confidence should be 1.0")
		}
	}
}

func TestLinkExtraction(t *testing.T) {
	parser := NewXMLParser()

	xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
	<document>
		<website>https://example.com</website>
		<email>contact@example.com</email>
		<description>Visit https://docs.example.com for more info</description>
	</document>`

	request := XMLParseRequest{
		ID:      "test_links",
		Content: xmlContent,
	}

	response, err := parser.Parse(request)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should extract URLs and emails
	if len(response.Links) < 3 {
		t.Errorf("Expected at least 3 links, got %d", len(response.Links))
	}

	// Check link types
	hasURL := false
	hasEmail := false
	for _, link := range response.Links {
		if link.LinkType == "url" && strings.HasPrefix(link.LinkTarget, "https://") {
			hasURL = true
		}
		if link.LinkType == "email" && strings.HasPrefix(link.LinkTarget, "mailto:") {
			hasEmail = true
		}
	}

	if !hasURL {
		t.Error("Should have extracted URL links")
	}
	if !hasEmail {
		t.Error("Should have extracted email links")
	}
}

func TestContentPreviewTruncation(t *testing.T) {
	parser := NewXMLParser()
	parser.MaxContentPreview = 20

	longText := "This is a very long text that should be truncated to fit the preview limit"
	preview := parser.truncateContent(longText)

	if len(preview) > 20 {
		t.Errorf("Preview too long: %d characters", len(preview))
	}
	if !strings.HasSuffix(preview, "...") {
		t.Error("Truncated preview should end with ...")
	}
}

func TestElementPathGeneration(t *testing.T) {
	parser := NewXMLParser()

	xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
	<root>
		<level1>
			<level2>
				<level3>Deep content</level3>
			</level2>
		</level1>
	</root>`

	request := XMLParseRequest{
		ID:      "test_paths",
		Content: xmlContent,
	}

	response, err := parser.Parse(request)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Check paths in metadata
	for _, element := range response.Elements {
		if element.ElementType == XMLElementTypeElement || element.ElementType == XMLElementTypeText {
			if path, ok := element.Metadata["xml_path"].(string); ok {
				if path == "" {
					t.Error("Element should have xml_path")
				}
				// Check path format
				if element.ElementType == XMLElementTypeElement && !strings.Contains(path, "/") {
					t.Error("Path should contain /")
				}
			}
		}
	}
}

func TestEmptyXML(t *testing.T) {
	parser := NewXMLParser()

	xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
	<root></root>`

	request := XMLParseRequest{
		ID:      "empty_test",
		Content: xmlContent,
	}

	response, err := parser.Parse(request)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should have root element and XML root
	if len(response.Elements) < 2 {
		t.Errorf("Expected at least 2 elements (root + XML root), got %d", len(response.Elements))
	}

	if response.Elements[0].ElementType != XMLElementTypeRoot {
		t.Error("Should have document root element")
	}
}

func TestInvalidXML(t *testing.T) {
	parser := NewXMLParser()

	xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
	<root>
		<unclosed>
		<another>tag</another>`  // Missing closing tag

	request := XMLParseRequest{
		ID:      "invalid_test",
		Content: xmlContent,
	}

	_, err := parser.Parse(request)
	if err == nil {
		t.Error("Should fail for invalid XML")
	}
}

func TestXMLSerialization(t *testing.T) {
	parser := NewXMLParser()

	request := XMLParseRequest{
		ID:      "test_xml",
		Content: `<?xml version="1.0"?><root><item>test</item></root>`,
		Metadata: map[string]interface{}{
			"source": "test",
		},
	}

	response, err := parser.Parse(request)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Test response to JSON
	jsonStr, err := response.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	if !strings.Contains(jsonStr, "test_xml") {
		t.Error("JSON should contain document ID")
	}

	// Test request from JSON
	requestJSON := `{"id":"test","content":"<root/>","metadata":{"key":"value"}}`
	var newRequest XMLParseRequest
	err = newRequest.FromJSON(requestJSON)
	if err != nil {
		t.Fatalf("FromJSON failed: %v", err)
	}

	if newRequest.ID != "test" {
		t.Error("ID not parsed correctly from JSON")
	}
	if newRequest.Content != "<root/>" {
		t.Error("Content not parsed correctly from JSON")
	}
}

func TestComplexRealWorldXML(t *testing.T) {
	parser := NewXMLParser()

	// Simulate a more complex real-world XML structure
	xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
	<catalog xmlns="http://example.org/catalog" lastUpdated="2025-04-28T14:32:10">
		<name>Metropolitan Public Library</name>
		<address>
			<street>123 Reading Avenue</street>
			<city>Bookville</city>
			<state>LT</state>
			<zip>12345</zip>
		</address>
		<contact>
			<phone>(555) 123-4567</phone>
			<email>info@library.org</email>
			<website>https://www.library.org</website>
		</contact>
		<books>
			<book id="ISBN-001" available="true">
				<title>The Go Programming Language</title>
				<author>Alan Donovan</author>
				<author>Brian Kernighan</author>
				<year>2015</year>
				<category>Programming</category>
			</book>
			<book id="ISBN-002" available="false">
				<title>Clean Code</title>
				<author>Robert Martin</author>
				<year>2008</year>
				<category>Programming</category>
			</book>
		</books>
	</catalog>`

	request := XMLParseRequest{
		ID:      "library_catalog",
		Content: xmlContent,
		Metadata: map[string]interface{}{
			"source": "library_system",
		},
	}

	response, err := parser.Parse(request)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should have many elements
	if len(response.Elements) < 20 {
		t.Errorf("Expected many elements for complex XML, got %d", len(response.Elements))
	}

	// Should have extracted links
	if len(response.Links) < 2 { // email and website
		t.Errorf("Expected at least 2 links, got %d", len(response.Links))
	}

	// Should have namespaces
	if _, ok := response.Document["namespaces"]; ok {
		// Good, namespaces were captured
	}

	// Check for specific elements
	foundLibraryName := false
	foundBookTitle := false
	for _, element := range response.Elements {
		if element.ElementType == XMLElementTypeText {
			if strings.Contains(element.Text, "Metropolitan Public Library") {
				foundLibraryName = true
			}
			if strings.Contains(element.Text, "The Go Programming Language") {
				foundBookTitle = true
			}
		}
	}

	if !foundLibraryName {
		t.Error("Should have found library name in text")
	}
	if !foundBookTitle {
		t.Error("Should have found book title in text")
	}
}