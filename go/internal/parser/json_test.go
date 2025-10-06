package parser

import (
	"strings"
	"testing"
)

func TestNewJSONParser(t *testing.T) {
	parser := NewJSONParser()
	if parser == nil {
		t.Fatal("Failed to create parser")
	}
	if parser.MaxContentPreview != 100 {
		t.Error("MaxContentPreview not set correctly")
	}
	if !parser.IncludeFieldNames {
		t.Error("IncludeFieldNames not set correctly")
	}
	if parser.FlattenArrays {
		t.Error("FlattenArrays should be false by default")
	}
	if parser.MaxDepth != 10 {
		t.Error("MaxDepth not set correctly")
	}
}

func TestParseSimpleJSON(t *testing.T) {
	parser := NewJSONParser()

	request := JSONParseRequest{
		ID:      "test_doc",
		Content: `{"name": "John", "age": 30, "city": "New York"}`,
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
	if response.Document["doc_type"] != "json" {
		t.Error("Document type not set correctly")
	}

	// Should have at least root element
	if len(response.Elements) < 1 {
		t.Fatal("No elements found")
	}

	// Root element should be first
	root := response.Elements[0]
	if root.ElementType != JSONElementTypeRoot {
		t.Error("First element should be root")
	}
	if root.ElementID == "" {
		t.Error("Root element should have ID")
	}
}

func TestParseJSONObject(t *testing.T) {
	parser := NewJSONParser()

	jsonContent := `{
		"user": {
			"name": "Alice",
			"age": 25,
			"contact": {
				"email": "alice@example.com",
				"phone": "123-456-7890"
			}
		},
		"active": true
	}`

	request := JSONParseRequest{
		ID:      "test_object",
		Content: jsonContent,
	}

	response, err := parser.Parse(request)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Check for different element types
	elementTypes := make(map[JSONElementType]int)
	for _, element := range response.Elements {
		elementTypes[element.ElementType]++
	}

	// Should have various element types
	expectedTypes := []JSONElementType{
		JSONElementTypeRoot,
		JSONElementTypeObject,
		JSONElementTypeField,
	}

	for _, expectedType := range expectedTypes {
		if elementTypes[expectedType] == 0 {
			t.Errorf("Expected element type %s not found", expectedType)
		}
	}

	// Should have relationships
	if len(response.Relationships) == 0 {
		t.Error("Should have relationships")
	}
}

func TestParseJSONArray(t *testing.T) {
	parser := NewJSONParser()

	jsonContent := `{
		"users": [
			{"name": "Alice", "age": 25},
			{"name": "Bob", "age": 30},
			{"name": "Charlie", "age": 35}
		]
	}`

	request := JSONParseRequest{
		ID:      "test_array",
		Content: jsonContent,
	}

	response, err := parser.Parse(request)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Check for array-related element types
	elementTypes := make(map[JSONElementType]int)
	for _, element := range response.Elements {
		elementTypes[element.ElementType]++
	}

	// Should have array and item elements
	if elementTypes[JSONElementTypeArray] == 0 {
		t.Error("Expected array element type not found")
	}
	if elementTypes[JSONElementTypeItem] == 0 {
		t.Error("Expected item element type not found")
	}

	// Should have multiple items (3 users)
	if elementTypes[JSONElementTypeItem] < 3 {
		t.Errorf("Expected at least 3 item elements, got %d", elementTypes[JSONElementTypeItem])
	}
}

func TestParseComplexJSON(t *testing.T) {
	parser := NewJSONParser()

	complexJSON := `{
		"company": {
			"name": "Tech Corp",
			"employees": [
				{
					"id": 1,
					"name": "John Doe",
					"department": "Engineering",
					"skills": ["Go", "Python", "JavaScript"],
					"contact": {
						"email": "john@techcorp.com",
						"phone": "+1-555-0123"
					}
				},
				{
					"id": 2,
					"name": "Jane Smith",
					"department": "Design",
					"skills": ["Figma", "Photoshop"],
					"contact": {
						"email": "jane@techcorp.com",
						"phone": "+1-555-0124"
					}
				}
			],
			"founded": 2020,
			"website": "https://techcorp.com"
		}
	}`

	request := JSONParseRequest{
		ID:      "complex_test",
		Content: complexJSON,
	}

	response, err := parser.Parse(request)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should have many elements
	if len(response.Elements) < 10 {
		t.Errorf("Expected many elements, got %d", len(response.Elements))
	}

	// Should have relationships
	if len(response.Relationships) < 5 {
		t.Errorf("Expected many relationships, got %d", len(response.Relationships))
	}

	// Should find links
	if len(response.Links) == 0 {
		t.Error("Expected to find links in JSON content")
	}

	// Check that we have all element types
	elementTypes := make(map[JSONElementType]bool)
	for _, element := range response.Elements {
		elementTypes[element.ElementType] = true
	}

	expectedTypes := []JSONElementType{
		JSONElementTypeRoot,
		JSONElementTypeObject,
		JSONElementTypeArray,
		JSONElementTypeField,
		JSONElementTypeItem,
	}

	for _, expectedType := range expectedTypes {
		if !elementTypes[expectedType] {
			t.Errorf("Expected element type %s not found", expectedType)
		}
	}
}

func TestGetValueType(t *testing.T) {
	parser := NewJSONParser()

	tests := []struct {
		value    interface{}
		expected string
	}{
		{"hello", "string"},
		{42.0, "number"},
		{true, "boolean"},
		{map[string]interface{}{"key": "value"}, "object"},
		{[]interface{}{1, 2, 3}, "array"},
		{nil, "null"},
	}

	for _, test := range tests {
		result := parser.getValueType(test.value)
		if result != test.expected {
			t.Errorf("getValueType(%v) = %s, want %s", test.value, result, test.expected)
		}
	}
}

func TestGetObjectPreview(t *testing.T) {
	parser := NewJSONParser()

	// Test empty object
	emptyObj := make(map[string]interface{})
	preview := parser.getObjectPreview(emptyObj)
	if preview != "{}" {
		t.Errorf("Empty object preview = %s, want {}", preview)
	}

	// Test object with few fields
	obj := map[string]interface{}{
		"name": "John",
		"age":  30,
	}
	preview = parser.getObjectPreview(obj)
	if !strings.Contains(preview, "name: ...") || !strings.Contains(preview, "age: ...") {
		t.Errorf("Object preview missing expected content: %s", preview)
	}

	// Test object with many fields
	largeObj := map[string]interface{}{
		"field1": "value1",
		"field2": "value2",
		"field3": "value3",
		"field4": "value4",
		"field5": "value5",
	}
	preview = parser.getObjectPreview(largeObj)
	if !strings.Contains(preview, "...") {
		t.Errorf("Large object preview should contain truncation: %s", preview)
	}
}

func TestGetArrayPreview(t *testing.T) {
	parser := NewJSONParser()

	// Test empty array
	emptyArr := []interface{}{}
	preview := parser.getArrayPreview(emptyArr)
	if preview != "[]" {
		t.Errorf("Empty array preview = %s, want []", preview)
	}

	// Test small array
	smallArr := []interface{}{1, 2}
	preview = parser.getArrayPreview(smallArr)
	if !strings.Contains(preview, "...") {
		t.Errorf("Small array preview should contain placeholders: %s", preview)
	}

	// Test large array
	largeArr := []interface{}{1, 2, 3, 4, 5, 6}
	preview = parser.getArrayPreview(largeArr)
	if !strings.Contains(preview, "...") {
		t.Errorf("Large array preview should contain truncation: %s", preview)
	}
}

func TestJSONLinkExtraction(t *testing.T) {
	parser := NewJSONParser()

	jsonContent := `{
		"website": "https://example.com",
		"api": "https://api.example.com/v1",
		"contact": {
			"support": "https://support.example.com"
		},
		"links": [
			"https://docs.example.com",
			"https://blog.example.com"
		]
	}`

	request := JSONParseRequest{
		ID:      "test_links",
		Content: jsonContent,
	}

	response, err := parser.Parse(request)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should extract multiple URLs
	if len(response.Links) < 3 {
		t.Errorf("Expected at least 3 links, got %d", len(response.Links))
	}

	// Check link properties
	for _, link := range response.Links {
		if link.LinkType != "url" {
			t.Error("Link type should be 'url'")
		}
		if link.LinkTarget == "" {
			t.Error("Link target should not be empty")
		}
		if !strings.HasPrefix(link.LinkTarget, "https://") {
			t.Error("Link target should start with https://")
		}
	}
}

func TestJSONTruncateContent(t *testing.T) {
	parser := NewJSONParser()
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

func TestJSONGenerateHash(t *testing.T) {
	parser := NewJSONParser()

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

func TestJSONCreateContentLocation(t *testing.T) {
	parser := NewJSONParser()

	location := parser.createContentLocation("source.json", JSONElementTypeField, "$.user.name")

	if location["source"] != "source.json" {
		t.Error("Source not set correctly")
	}
	if location["type"] != "json_field" {
		t.Error("Type not set correctly")
	}
	if location["path"] != "$.user.name" {
		t.Error("Path not set correctly")
	}
}

func TestJSONElementSerialization(t *testing.T) {
	parser := NewJSONParser()

	request := JSONParseRequest{
		ID:      "test_json",
		Content: `{"name": "test", "value": 42}`,
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

	if !strings.Contains(jsonStr, "test_json") {
		t.Error("JSON should contain document ID")
	}

	// Test request from JSON
	requestJSON := `{"id":"test","content":"{\"key\":\"value\"}","metadata":{"key":"value"}}`
	var newRequest JSONParseRequest
	err = newRequest.FromJSON(requestJSON)
	if err != nil {
		t.Fatalf("FromJSON failed: %v", err)
	}

	if newRequest.ID != "test" {
		t.Error("ID not parsed correctly from JSON")
	}
	if newRequest.Content != `{"key":"value"}` {
		t.Error("Content not parsed correctly from JSON")
	}
}

func TestJSONRelationships(t *testing.T) {
	parser := NewJSONParser()

	jsonContent := `{
		"parent": {
			"child1": "value1",
			"child2": {
				"grandchild": "value2"
			}
		}
	}`

	request := JSONParseRequest{
		ID:      "test_relationships",
		Content: jsonContent,
	}

	response, err := parser.Parse(request)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should have relationships between parent and child elements
	if len(response.Relationships) == 0 {
		t.Error("Should have relationships")
	}

	// Check relationship properties
	for _, rel := range response.Relationships {
		if rel.RelationshipType != "contains" {
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

func TestEmptyJSON(t *testing.T) {
	parser := NewJSONParser()

	request := JSONParseRequest{
		ID:      "empty_test",
		Content: "{}",
	}

	response, err := parser.Parse(request)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should have root element and empty object
	if len(response.Elements) < 2 {
		t.Errorf("Expected at least 2 elements (root + object), got %d", len(response.Elements))
	}

	if response.Elements[0].ElementType != JSONElementTypeRoot {
		t.Error("Should have root element")
	}
}

func TestInvalidJSON(t *testing.T) {
	parser := NewJSONParser()

	request := JSONParseRequest{
		ID:      "invalid_test",
		Content: `{"invalid": json}`, // Missing quotes around json
	}

	_, err := parser.Parse(request)
	if err == nil {
		t.Error("Should fail for invalid JSON")
	}
}

func TestMaxDepth(t *testing.T) {
	parser := NewJSONParser()
	parser.MaxDepth = 2 // Set low depth limit

	// Create deeply nested JSON
	deepJSON := `{
		"level1": {
			"level2": {
				"level3": {
					"level4": "too deep"
				}
			}
		}
	}`

	request := JSONParseRequest{
		ID:      "depth_test",
		Content: deepJSON,
	}

	response, err := parser.Parse(request)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should complete without infinite recursion
	if len(response.Elements) == 0 {
		t.Error("Should have parsed some elements")
	}
}