package builder

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/kennethstott/go-doc-go/internal/udml/query"
)

// MockBackendForBuilder implements QueryBackend for testing
type MockBackendForBuilder struct {
	elements []map[string]interface{}
}

func (m *MockBackendForBuilder) GetName() string {
	return "mock"
}

func (m *MockBackendForBuilder) GetVersion() string {
	return "1.0.0"
}

func (m *MockBackendForBuilder) Initialize(ctx context.Context, config query.BackendConfig) error {
	return nil
}

func (m *MockBackendForBuilder) Translate(expr *query.Expression, opts query.TranslateOptions) (*query.NativeQuery, error) {
	return &query.NativeQuery{
		QueryString: "SELECT * FROM elements WHERE doc_id = ?",
	}, nil
}

func (m *MockBackendForBuilder) Execute(ctx context.Context, q *query.NativeQuery) (*query.QueryResult, error) {
	// Convert []map[string]interface{} to []query.Row
	rows := make([]query.Row, len(m.elements))
	for i, elem := range m.elements {
		rows[i] = query.Row(elem)
	}

	return &query.QueryResult{
		Rows:          rows,
		RowCount:      len(m.elements),
		ExecutionTime: time.Microsecond,
	}, nil
}

func (m *MockBackendForBuilder) Explain(ctx context.Context, q *query.NativeQuery) (*query.QueryPlan, error) {
	return &query.QueryPlan{BackendName: "mock"}, nil
}

func (m *MockBackendForBuilder) SupportsFeature(feature string) bool {
	return true
}

func (m *MockBackendForBuilder) Close() error {
	return nil
}

// Test Data Helpers

func createFlatElements() []map[string]interface{} {
	pageNum1 := int64(1)
	pageNum2 := int64(2)
	sectionLevel1 := int64(1)
	sectionLevel2 := int64(2)

	return []map[string]interface{}{
		{
			"element_id":        "root1",
			"doc_id":            "doc1",
			"source_name":       "test_doc",
			"version":           "v2.0.0",
			"element_type":      "document",
			"element_category":  "container",
			"content":           "Test Document",
			"content_preview":   "Test Document",
			"parent_id":         "",
			"element_order":     1.0,
			"document_position": 1.0,
		},
		{
			"element_id":        "header1",
			"doc_id":            "doc1",
			"source_name":       "test_doc",
			"version":           "v2.0.0",
			"element_type":      "header",
			"element_category":  "text",
			"content":           "Chapter 1",
			"content_preview":   "Chapter 1",
			"parent_id":         "root1",
			"element_order":     1.0,
			"document_position": 2.0,
			"page_number":       pageNum1,
			"section_level":     sectionLevel1,
		},
		{
			"element_id":        "para1",
			"doc_id":            "doc1",
			"source_name":       "test_doc",
			"version":           "v2.0.0",
			"element_type":      "paragraph",
			"element_category":  "text",
			"content":           "First paragraph in chapter 1",
			"content_preview":   "First paragraph in chapter 1",
			"parent_id":         "header1",
			"element_order":     1.0,
			"document_position": 3.0,
			"page_number":       pageNum1,
		},
		{
			"element_id":        "para2",
			"doc_id":            "doc1",
			"source_name":       "test_doc",
			"version":           "v2.0.0",
			"element_type":      "paragraph",
			"element_category":  "text",
			"content":           "Second paragraph in chapter 1",
			"content_preview":   "Second paragraph in chapter 1",
			"parent_id":         "header1",
			"element_order":     2.0,
			"document_position": 4.0,
			"page_number":       pageNum1,
		},
		{
			"element_id":        "header2",
			"doc_id":            "doc1",
			"source_name":       "test_doc",
			"version":           "v2.0.0",
			"element_type":      "header",
			"element_category":  "text",
			"content":           "Section 1.1",
			"content_preview":   "Section 1.1",
			"parent_id":         "header1",
			"element_order":     3.0,
			"document_position": 5.0,
			"page_number":       pageNum2,
			"section_level":     sectionLevel2,
		},
		{
			"element_id":        "para3",
			"doc_id":            "doc1",
			"source_name":       "test_doc",
			"version":           "v2.0.0",
			"element_type":      "paragraph",
			"element_category":  "text",
			"content":           "Paragraph in section 1.1",
			"content_preview":   "Paragraph in section 1.1",
			"parent_id":         "header2",
			"element_order":     1.0,
			"document_position": 6.0,
			"page_number":       pageNum2,
		},
	}
}

func createFlatElementsWithMetadata() []map[string]interface{} {
	elements := createFlatElements()

	// Add JSON overflow fields to some elements
	elements[1]["metadata"] = `{"font_size": 18, "font_family": "Arial"}`
	elements[2]["metadata"] = `{"font_size": 12, "styles": {"bold": false}}`
	elements[2]["content_location"] = `{"x": 100, "y": 200, "width": 400, "height": 50}`

	return elements
}

// Tests

func TestNewDocumentBuilder(t *testing.T) {
	backend := &MockBackendForBuilder{}
	options := DefaultBuildOptions()

	builder := NewDocumentBuilder(backend, options)

	if builder == nil {
		t.Fatal("Builder should not be nil")
	}
	if builder.backend == nil {
		t.Error("Backend not set correctly")
	}
	if !builder.options.IncludeContent {
		t.Error("Default options should include content")
	}
}

func TestDefaultBuildOptions(t *testing.T) {
	opts := DefaultBuildOptions()

	if !opts.IncludeContent {
		t.Error("Should include content by default")
	}
	if opts.MaxDepth != 0 {
		t.Error("Should have unlimited depth by default")
	}
	if !opts.SortChildren {
		t.Error("Should sort children by default")
	}
	if !opts.IncludeMetadata {
		t.Error("Should include metadata by default")
	}
}

func TestBuildDocumentFromElements_SimpleHierarchy(t *testing.T) {
	backend := &MockBackendForBuilder{}
	builder := NewDocumentBuilder(backend, DefaultBuildOptions())

	elements := createFlatElements()
	doc, err := builder.BuildDocumentFromElements("doc1", elements)
	if err != nil {
		t.Fatalf("Failed to build document: %v", err)
	}

	// Verify document
	if doc.DocID != "doc1" {
		t.Errorf("Expected doc_id 'doc1', got '%s'", doc.DocID)
	}
	if doc.SourceName != "test_doc" {
		t.Errorf("Expected source_name 'test_doc', got '%s'", doc.SourceName)
	}
	if doc.Version != "v2.0.0" {
		t.Errorf("Expected version 'v2.0.0', got '%s'", doc.Version)
	}

	// Should have 1 root element
	if len(doc.Elements) != 1 {
		t.Fatalf("Expected 1 root element, got %d", len(doc.Elements))
	}

	root := doc.Elements[0]
	if root.ElementID != "root1" {
		t.Errorf("Expected root element ID 'root1', got '%s'", root.ElementID)
	}

	// Root should have 2 children: header1 and header2 (both at same level)
	if len(root.Children) != 1 {
		t.Fatalf("Expected root to have 1 child (header1), got %d", len(root.Children))
	}

	header1 := root.Children[0]
	if header1.ElementID != "header1" {
		t.Errorf("Expected first child 'header1', got '%s'", header1.ElementID)
	}

	// header1 should have 3 children: para1, para2, header2
	if len(header1.Children) != 3 {
		t.Fatalf("Expected header1 to have 3 children, got %d", len(header1.Children))
	}

	// Verify children are sorted by element_order
	if header1.Children[0].ElementID != "para1" {
		t.Errorf("Expected first child 'para1', got '%s'", header1.Children[0].ElementID)
	}
	if header1.Children[1].ElementID != "para2" {
		t.Errorf("Expected second child 'para2', got '%s'", header1.Children[1].ElementID)
	}
	if header1.Children[2].ElementID != "header2" {
		t.Errorf("Expected third child 'header2', got '%s'", header1.Children[2].ElementID)
	}

	// header2 should have 1 child: para3
	header2 := header1.Children[2]
	if len(header2.Children) != 1 {
		t.Fatalf("Expected header2 to have 1 child, got %d", len(header2.Children))
	}
	if header2.Children[0].ElementID != "para3" {
		t.Errorf("Expected child 'para3', got '%s'", header2.Children[0].ElementID)
	}
}

func TestBuildDocumentFromElements_WithMetadata(t *testing.T) {
	backend := &MockBackendForBuilder{}
	builder := NewDocumentBuilder(backend, DefaultBuildOptions())

	elements := createFlatElementsWithMetadata()
	doc, err := builder.BuildDocumentFromElements("doc1", elements)
	if err != nil {
		t.Fatalf("Failed to build document: %v", err)
	}

	// Find header1 (should have metadata)
	header1 := doc.Elements[0].Children[0]
	if header1.Metadata == nil {
		t.Error("Expected metadata to be present")
	}

	fontSize, ok := header1.Metadata["font_size"].(float64)
	if !ok || fontSize != 18 {
		t.Errorf("Expected font_size=18, got %v", header1.Metadata["font_size"])
	}

	// Find para1 (should have metadata and content_location)
	para1 := header1.Children[0]
	if para1.Metadata == nil {
		t.Error("Expected metadata to be present on para1")
	}
	if para1.ContentLocation == nil {
		t.Error("Expected content_location to be present on para1")
	}

	x, ok := para1.ContentLocation["x"].(float64)
	if !ok || x != 100 {
		t.Errorf("Expected x=100, got %v", para1.ContentLocation["x"])
	}
}

func TestBuildDocumentFromElements_NoContent(t *testing.T) {
	backend := &MockBackendForBuilder{}
	options := DefaultBuildOptions()
	options.IncludeContent = false

	builder := NewDocumentBuilder(backend, options)

	elements := createFlatElements()
	doc, err := builder.BuildDocumentFromElements("doc1", elements)
	if err != nil {
		t.Fatalf("Failed to build document: %v", err)
	}

	// Content should be empty, but preview should exist
	root := doc.Elements[0]
	if root.Content != "" {
		t.Errorf("Expected no content, got '%s'", root.Content)
	}
	if root.ContentPreview == "" {
		t.Error("Expected content preview to exist")
	}
}

func TestBuildDocumentFromElements_MaxDepth(t *testing.T) {
	backend := &MockBackendForBuilder{}
	options := DefaultBuildOptions()
	options.MaxDepth = 2 // root + 1 level of children

	builder := NewDocumentBuilder(backend, options)

	elements := createFlatElements()
	doc, err := builder.BuildDocumentFromElements("doc1", elements)
	if err != nil {
		t.Fatalf("Failed to build document: %v", err)
	}

	// Root should have children
	root := doc.Elements[0]
	if len(root.Children) == 0 {
		t.Error("Expected root to have children at depth 2")
	}

	// But those children should not have children (depth exceeded)
	for _, child := range root.Children {
		if len(child.Children) > 0 {
			t.Error("Expected no grandchildren at max depth")
		}
	}
}

func TestBuildDocumentFromElements_NoSorting(t *testing.T) {
	backend := &MockBackendForBuilder{}
	options := DefaultBuildOptions()
	options.SortChildren = false

	builder := NewDocumentBuilder(backend, options)

	// Create elements in reverse order
	elements := []map[string]interface{}{
		{
			"element_id":        "root",
			"doc_id":            "doc1",
			"element_type":      "document",
			"parent_id":         "",
			"element_order":     1.0,
			"document_position": 1.0,
		},
		{
			"element_id":        "child2",
			"doc_id":            "doc1",
			"element_type":      "paragraph",
			"parent_id":         "root",
			"element_order":     2.0,
			"document_position": 3.0,
		},
		{
			"element_id":        "child1",
			"doc_id":            "doc1",
			"element_type":      "paragraph",
			"parent_id":         "root",
			"element_order":     1.0,
			"document_position": 2.0,
		},
	}

	doc, err := builder.BuildDocumentFromElements("doc1", elements)
	if err != nil {
		t.Fatalf("Failed to build document: %v", err)
	}

	// Children should NOT be sorted (insertion order)
	root := doc.Elements[0]
	if len(root.Children) != 2 {
		t.Fatalf("Expected 2 children, got %d", len(root.Children))
	}

	// First child should be child2 (insertion order, not sorted)
	if root.Children[0].ElementID != "child2" {
		t.Errorf("Expected first child 'child2' (unsorted), got '%s'", root.Children[0].ElementID)
	}
}

func TestDocument_ToJSON(t *testing.T) {
	backend := &MockBackendForBuilder{}
	builder := NewDocumentBuilder(backend, DefaultBuildOptions())

	elements := createFlatElements()
	doc, err := builder.BuildDocumentFromElements("doc1", elements)
	if err != nil {
		t.Fatalf("Failed to build document: %v", err)
	}

	jsonBytes, err := doc.ToJSON()
	if err != nil {
		t.Fatalf("Failed to serialize to JSON: %v", err)
	}

	// Parse back to verify structure
	var parsed map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if parsed["doc_id"] != "doc1" {
		t.Errorf("Expected doc_id 'doc1', got '%v'", parsed["doc_id"])
	}

	elementsRaw, ok := parsed["elements"]
	if !ok {
		t.Fatal("Expected elements field")
	}
	parsedElements, elemOk := elementsRaw.([]interface{})
	if !elemOk {
		t.Fatal("Expected elements array")
	}
	if len(parsedElements) != 1 {
		t.Errorf("Expected 1 root element, got %d", len(parsedElements))
	}
}

func TestDocument_ToJSONCompact(t *testing.T) {
	backend := &MockBackendForBuilder{}
	builder := NewDocumentBuilder(backend, DefaultBuildOptions())

	elements := []map[string]interface{}{
		{
			"element_id":        "root",
			"doc_id":            "doc1",
			"element_type":      "document",
			"parent_id":         "",
			"element_order":     1.0,
			"document_position": 1.0,
		},
	}

	doc, err := builder.BuildDocumentFromElements("doc1", elements)
	if err != nil {
		t.Fatalf("Failed to build document: %v", err)
	}

	compactJSON, err := doc.ToJSONCompact()
	if err != nil {
		t.Fatalf("Failed to serialize to compact JSON: %v", err)
	}

	prettyJSON, err := doc.ToJSON()
	if err != nil {
		t.Fatalf("Failed to serialize to pretty JSON: %v", err)
	}

	// Compact should be shorter
	if len(compactJSON) >= len(prettyJSON) {
		t.Error("Expected compact JSON to be shorter than pretty JSON")
	}
}

func TestDocument_GetElementCount(t *testing.T) {
	backend := &MockBackendForBuilder{}
	builder := NewDocumentBuilder(backend, DefaultBuildOptions())

	elements := createFlatElements()
	doc, err := builder.BuildDocumentFromElements("doc1", elements)
	if err != nil {
		t.Fatalf("Failed to build document: %v", err)
	}

	count := doc.GetElementCount()
	expectedCount := 6 // root + header1 + para1 + para2 + header2 + para3

	if count != expectedCount {
		t.Errorf("Expected %d elements, got %d", expectedCount, count)
	}
}

func TestDocument_GetMaxDepth(t *testing.T) {
	backend := &MockBackendForBuilder{}
	builder := NewDocumentBuilder(backend, DefaultBuildOptions())

	elements := createFlatElements()
	doc, err := builder.BuildDocumentFromElements("doc1", elements)
	if err != nil {
		t.Fatalf("Failed to build document: %v", err)
	}

	depth := doc.GetMaxDepth()
	// root -> header1 -> header2 -> para3 = depth 4
	expectedDepth := 4

	if depth != expectedDepth {
		t.Errorf("Expected depth %d, got %d", expectedDepth, depth)
	}
}

func TestBuildDocumentFromElements_OrphanElements(t *testing.T) {
	backend := &MockBackendForBuilder{}
	builder := NewDocumentBuilder(backend, DefaultBuildOptions())

	// Create elements where child references non-existent parent
	elements := []map[string]interface{}{
		{
			"element_id":        "root",
			"doc_id":            "doc1",
			"element_type":      "document",
			"parent_id":         "",
			"element_order":     1.0,
			"document_position": 1.0,
		},
		{
			"element_id":        "orphan",
			"doc_id":            "doc1",
			"element_type":      "paragraph",
			"parent_id":         "missing_parent",
			"element_order":     1.0,
			"document_position": 2.0,
		},
	}

	doc, err := builder.BuildDocumentFromElements("doc1", elements)
	if err != nil {
		t.Fatalf("Failed to build document: %v", err)
	}

	// Should have 2 root elements (root + orphan)
	if len(doc.Elements) != 2 {
		t.Errorf("Expected 2 root elements (including orphan), got %d", len(doc.Elements))
	}
}

func TestBuildDocument_Integration(t *testing.T) {
	// Mock backend with test data
	backend := &MockBackendForBuilder{
		elements: createFlatElements(),
	}

	builder := NewDocumentBuilder(backend, DefaultBuildOptions())

	ctx := context.Background()
	doc, err := builder.BuildDocument(ctx, "doc1")
	if err != nil {
		t.Fatalf("Failed to build document: %v", err)
	}

	if doc.DocID != "doc1" {
		t.Errorf("Expected doc_id 'doc1', got '%s'", doc.DocID)
	}

	if len(doc.Elements) == 0 {
		t.Error("Expected at least one root element")
	}
}

func TestBuildDocument_NotFound(t *testing.T) {
	// Mock backend with no elements
	backend := &MockBackendForBuilder{
		elements: []map[string]interface{}{},
	}

	builder := NewDocumentBuilder(backend, DefaultBuildOptions())

	ctx := context.Background()
	_, err := builder.BuildDocument(ctx, "nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent document")
	}
}

func TestHelperFunctions(t *testing.T) {
	t.Run("getStringField", func(t *testing.T) {
		row := map[string]interface{}{
			"str":     "value",
			"not_str": 123,
		}

		if val := getStringField(row, "str"); val != "value" {
			t.Errorf("Expected 'value', got '%s'", val)
		}
		if val := getStringField(row, "not_str"); val != "" {
			t.Errorf("Expected empty string for non-string, got '%s'", val)
		}
		if val := getStringField(row, "missing"); val != "" {
			t.Errorf("Expected empty string for missing field, got '%s'", val)
		}
	})

	t.Run("getFloatField", func(t *testing.T) {
		row := map[string]interface{}{
			"float64": float64(1.5),
			"float32": float32(2.5),
			"int":     42,
			"int64":   int64(99),
		}

		if val := getFloatField(row, "float64"); val != 1.5 {
			t.Errorf("Expected 1.5, got %f", val)
		}
		if val := getFloatField(row, "int"); val != 42.0 {
			t.Errorf("Expected 42.0, got %f", val)
		}
	})

	t.Run("getIntField", func(t *testing.T) {
		row := map[string]interface{}{
			"int":     42,
			"int64":   int64(99),
			"float64": float64(77),
		}

		val := getIntField(row, "int")
		if val == nil || *val != 42 {
			t.Errorf("Expected 42, got %v", val)
		}

		val = getIntField(row, "missing")
		if val != nil {
			t.Errorf("Expected nil for missing field, got %v", val)
		}
	})

	t.Run("getJSONField", func(t *testing.T) {
		row := map[string]interface{}{
			"json_map":    map[string]interface{}{"key": "value"},
			"json_string": `{"key":"value"}`,
			"invalid":     "not json",
		}

		val := getJSONField(row, "json_map")
		if val == nil || val["key"] != "value" {
			t.Errorf("Expected map with key=value, got %v", val)
		}

		val = getJSONField(row, "json_string")
		if val == nil || val["key"] != "value" {
			t.Errorf("Expected parsed JSON with key=value, got %v", val)
		}

		val = getJSONField(row, "invalid")
		if val != nil {
			t.Errorf("Expected nil for invalid JSON, got %v", val)
		}
	})
}
