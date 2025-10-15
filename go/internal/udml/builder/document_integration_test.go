package builder

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/kennethstott/doculyzer-go-conversion/internal/analytics"
	"github.com/kennethstott/doculyzer-go-conversion/internal/udml/query"
)

// TestDocumentBuilder_EndToEnd tests document building with real Parquet storage and DuckDB
func TestDocumentBuilder_EndToEnd(t *testing.T) {
	// Create temp directory for test data
	tmpDir, err := os.MkdirTemp("", "builder_integration_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Step 1: Write hierarchical test data to Parquet
	if err := writeHierarchicalTestData(tmpDir); err != nil {
		t.Fatalf("Failed to write test data: %v", err)
	}

	// Step 2: Initialize DuckDB backend
	backend, err := initializeTestBackend(tmpDir)
	if err != nil {
		t.Fatalf("Failed to initialize backend: %v", err)
	}
	defer backend.Close()

	// Step 3: Create document builder
	builder := NewDocumentBuilder(backend, DefaultBuildOptions())

	// Step 4: Build document
	ctx := context.Background()
	doc, err := builder.BuildDocument(ctx, "doc1")
	if err != nil {
		t.Fatalf("Failed to build document: %v", err)
	}

	// Verify document structure
	if doc.DocID != "doc1" {
		t.Errorf("Expected doc_id 'doc1', got '%s'", doc.DocID)
	}

	if len(doc.Elements) != 1 {
		t.Fatalf("Expected 1 root element, got %d", len(doc.Elements))
	}

	root := doc.Elements[0]
	if root.ElementType != "document" {
		t.Errorf("Expected root type 'document', got '%s'", root.ElementType)
	}

	// Verify hierarchy is built correctly
	if len(root.Children) == 0 {
		t.Error("Expected root to have children")
	}

	t.Logf("Successfully built document with %d total elements", doc.GetElementCount())
	t.Logf("Document max depth: %d", doc.GetMaxDepth())
}

// TestDocumentBuilder_MultipleDocuments tests building multiple documents
func TestDocumentBuilder_MultipleDocuments(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "builder_multi_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Write data for two documents
	if err := writeMultiDocumentTestData(tmpDir); err != nil {
		t.Fatalf("Failed to write test data: %v", err)
	}

	backend, err := initializeTestBackend(tmpDir)
	if err != nil {
		t.Fatalf("Failed to initialize backend: %v", err)
	}
	defer backend.Close()

	builder := NewDocumentBuilder(backend, DefaultBuildOptions())
	ctx := context.Background()

	// Build doc1
	doc1, err := builder.BuildDocument(ctx, "doc1")
	if err != nil {
		t.Fatalf("Failed to build doc1: %v", err)
	}

	// Build doc2
	doc2, err := builder.BuildDocument(ctx, "doc2")
	if err != nil {
		t.Fatalf("Failed to build doc2: %v", err)
	}

	// Verify both documents are distinct
	if doc1.DocID == doc2.DocID {
		t.Error("Documents should have different IDs")
	}

	t.Logf("doc1: %d elements, max depth %d", doc1.GetElementCount(), doc1.GetMaxDepth())
	t.Logf("doc2: %d elements, max depth %d", doc2.GetElementCount(), doc2.GetMaxDepth())
}

// TestDocumentBuilder_WithFilters tests building with element type filters
func TestDocumentBuilder_WithFilters(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "builder_filters_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := writeHierarchicalTestData(tmpDir); err != nil {
		t.Fatalf("Failed to write test data: %v", err)
	}

	backend, err := initializeTestBackend(tmpDir)
	if err != nil {
		t.Fatalf("Failed to initialize backend: %v", err)
	}
	defer backend.Close()

	// Build with filter: only paragraphs and headers
	options := DefaultBuildOptions()
	options.Filter = query.NewOr(
		query.NewComparison("element_type", query.OpEqual, "paragraph"),
		query.NewComparison("element_type", query.OpEqual, "header"),
	)

	builder := NewDocumentBuilder(backend, options)
	ctx := context.Background()

	doc, err := builder.BuildDocument(ctx, "doc1")
	if err != nil {
		t.Fatalf("Failed to build filtered document: %v", err)
	}

	// Verify only paragraphs and headers
	verifyElementTypes(t, doc.Elements, []string{"paragraph", "header", "document"})

	t.Logf("Filtered document: %d elements", doc.GetElementCount())
}

// TestDocumentBuilder_JSONSerialization tests JSON output
func TestDocumentBuilder_JSONSerialization(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "builder_json_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := writeHierarchicalTestData(tmpDir); err != nil {
		t.Fatalf("Failed to write test data: %v", err)
	}

	backend, err := initializeTestBackend(tmpDir)
	if err != nil {
		t.Fatalf("Failed to initialize backend: %v", err)
	}
	defer backend.Close()

	builder := NewDocumentBuilder(backend, DefaultBuildOptions())
	ctx := context.Background()

	doc, err := builder.BuildDocument(ctx, "doc1")
	if err != nil {
		t.Fatalf("Failed to build document: %v", err)
	}

	// Test pretty JSON
	prettyJSON, err := doc.ToJSON()
	if err != nil {
		t.Fatalf("Failed to serialize to pretty JSON: %v", err)
	}

	// Test compact JSON
	compactJSON, err := doc.ToJSONCompact()
	if err != nil {
		t.Fatalf("Failed to serialize to compact JSON: %v", err)
	}

	// Verify JSON is valid
	var parsed map[string]interface{}
	if err := json.Unmarshal(prettyJSON, &parsed); err != nil {
		t.Fatalf("Invalid pretty JSON: %v", err)
	}

	if err := json.Unmarshal(compactJSON, &parsed); err != nil {
		t.Fatalf("Invalid compact JSON: %v", err)
	}

	t.Logf("Pretty JSON size: %d bytes", len(prettyJSON))
	t.Logf("Compact JSON size: %d bytes", len(compactJSON))

	if len(compactJSON) >= len(prettyJSON) {
		t.Error("Expected compact JSON to be smaller")
	}
}

// TestDocumentBuilder_MaxDepth tests depth limiting
func TestDocumentBuilder_MaxDepth(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "builder_depth_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := writeDeepHierarchyData(tmpDir); err != nil {
		t.Fatalf("Failed to write test data: %v", err)
	}

	backend, err := initializeTestBackend(tmpDir)
	if err != nil {
		t.Fatalf("Failed to initialize backend: %v", err)
	}
	defer backend.Close()

	// Build with max depth = 3
	options := DefaultBuildOptions()
	options.MaxDepth = 3

	builder := NewDocumentBuilder(backend, options)
	ctx := context.Background()

	doc, err := builder.BuildDocument(ctx, "deep_doc")
	if err != nil {
		t.Fatalf("Failed to build document: %v", err)
	}

	actualDepth := doc.GetMaxDepth()
	if actualDepth > 3 {
		t.Errorf("Expected max depth <= 3, got %d", actualDepth)
	}

	t.Logf("Limited depth to %d (actual: %d)", options.MaxDepth, actualDepth)
}

// Test Helper Functions

func writeHierarchicalTestData(tmpDir string) error {
	storage, err := analytics.NewHiveParquetStorage(map[string]interface{}{
		"path":    tmpDir,
		"version": "v2.0.0",
	})
	if err != nil {
		return err
	}
	defer storage.Close()

	pageNum1 := 1
	sectionLevel1 := 1

	elements := []analytics.Element{
		{
			ElementID:        "root1",
			DocID:            "doc1",
			SourceName:       "test_doc",
			ElementType:      "document",
			ElementCategory:  "container",
			Content:          "Test Document",
			ContentPreview:   "Test Document",
			ElementOrder:     1.0,
			DocumentPosition: 1.0,
		},
		{
			ElementID:        "header1",
			DocID:            "doc1",
			SourceName:       "test_doc",
			ElementType:      "header",
			ElementCategory:  "text",
			Content:          "Chapter 1",
			ContentPreview:   "Chapter 1",
			ParentID:         "root1",
			ElementOrder:     1.0,
			DocumentPosition: 2.0,
			PageNumber:       &pageNum1,
			SectionLevel:     &sectionLevel1,
		},
		{
			ElementID:        "para1",
			DocID:            "doc1",
			SourceName:       "test_doc",
			ElementType:      "paragraph",
			ElementCategory:  "text",
			Content:          "First paragraph",
			ContentPreview:   "First paragraph",
			ParentID:         "header1",
			ElementOrder:     1.0,
			DocumentPosition: 3.0,
			PageNumber:       &pageNum1,
		},
		{
			ElementID:        "para2",
			DocID:            "doc1",
			SourceName:       "test_doc",
			ElementType:      "paragraph",
			ElementCategory:  "text",
			Content:          "Second paragraph",
			ContentPreview:   "Second paragraph",
			ParentID:         "header1",
			ElementOrder:     2.0,
			DocumentPosition: 4.0,
			PageNumber:       &pageNum1,
		},
	}

	return storage.AppendElements(elements)
}

func writeMultiDocumentTestData(tmpDir string) error {
	storage, err := analytics.NewHiveParquetStorage(map[string]interface{}{
		"path":    tmpDir,
		"version": "v2.0.0",
	})
	if err != nil {
		return err
	}
	defer storage.Close()

	elements := []analytics.Element{
		// doc1
		{
			ElementID:        "root1",
			DocID:            "doc1",
			SourceName:       "doc1",
			ElementType:      "document",
			Content:          "Document 1",
			ContentPreview:   "Document 1",
			ElementOrder:     1.0,
			DocumentPosition: 1.0,
		},
		{
			ElementID:        "para1",
			DocID:            "doc1",
			SourceName:       "doc1",
			ElementType:      "paragraph",
			Content:          "Content for doc1",
			ContentPreview:   "Content for doc1",
			ParentID:         "root1",
			ElementOrder:     1.0,
			DocumentPosition: 2.0,
		},
		// doc2
		{
			ElementID:        "root2",
			DocID:            "doc2",
			SourceName:       "doc2",
			ElementType:      "document",
			Content:          "Document 2",
			ContentPreview:   "Document 2",
			ElementOrder:     1.0,
			DocumentPosition: 1.0,
		},
		{
			ElementID:        "para2",
			DocID:            "doc2",
			SourceName:       "doc2",
			ElementType:      "paragraph",
			Content:          "Content for doc2",
			ContentPreview:   "Content for doc2",
			ParentID:         "root2",
			ElementOrder:     1.0,
			DocumentPosition: 2.0,
		},
	}

	return storage.AppendElements(elements)
}

func writeDeepHierarchyData(tmpDir string) error {
	storage, err := analytics.NewHiveParquetStorage(map[string]interface{}{
		"path":    tmpDir,
		"version": "v2.0.0",
	})
	if err != nil {
		return err
	}
	defer storage.Close()

	// Create a deep hierarchy: root -> ch1 -> ch2 -> ch3 -> ch4 -> ch5
	elements := []analytics.Element{
		{
			ElementID:        "root",
			DocID:            "deep_doc",
			SourceName:       "deep",
			ElementType:      "document",
			Content:          "Root",
			ContentPreview:   "Root",
			ElementOrder:     1.0,
			DocumentPosition: 1.0,
		},
		{
			ElementID:        "ch1",
			DocID:            "deep_doc",
			SourceName:       "deep",
			ElementType:      "header",
			Content:          "Level 1",
			ContentPreview:   "Level 1",
			ParentID:         "root",
			ElementOrder:     1.0,
			DocumentPosition: 2.0,
		},
		{
			ElementID:        "ch2",
			DocID:            "deep_doc",
			SourceName:       "deep",
			ElementType:      "header",
			Content:          "Level 2",
			ContentPreview:   "Level 2",
			ParentID:         "ch1",
			ElementOrder:     1.0,
			DocumentPosition: 3.0,
		},
		{
			ElementID:        "ch3",
			DocID:            "deep_doc",
			SourceName:       "deep",
			ElementType:      "header",
			Content:          "Level 3",
			ContentPreview:   "Level 3",
			ParentID:         "ch2",
			ElementOrder:     1.0,
			DocumentPosition: 4.0,
		},
		{
			ElementID:        "ch4",
			DocID:            "deep_doc",
			SourceName:       "deep",
			ElementType:      "header",
			Content:          "Level 4",
			ContentPreview:   "Level 4",
			ParentID:         "ch3",
			ElementOrder:     1.0,
			DocumentPosition: 5.0,
		},
		{
			ElementID:        "ch5",
			DocID:            "deep_doc",
			SourceName:       "deep",
			ElementType:      "paragraph",
			Content:          "Level 5",
			ContentPreview:   "Level 5",
			ParentID:         "ch4",
			ElementOrder:     1.0,
			DocumentPosition: 6.0,
		},
	}

	return storage.AppendElements(elements)
}

func initializeTestBackend(tmpDir string) (query.QueryBackend, error) {
	config := query.BackendConfig{
		Type:        "duckdb",
		ParquetPath: tmpDir,
		Timeout:     30 * time.Second,
	}

	backend, err := query.GetOrCreateBackend(config)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	if err := backend.Initialize(ctx, config); err != nil {
		return nil, err
	}

	return backend, nil
}

func verifyElementTypes(t *testing.T, elements []*Element, allowedTypes []string) {
	allowed := make(map[string]bool)
	for _, typ := range allowedTypes {
		allowed[typ] = true
	}

	var check func([]*Element)
	check = func(elems []*Element) {
		for _, elem := range elems {
			if !allowed[elem.ElementType] {
				t.Errorf("Unexpected element type: %s (allowed: %v)", elem.ElementType, allowedTypes)
			}
			if len(elem.Children) > 0 {
				check(elem.Children)
			}
		}
	}

	check(elements)
}
