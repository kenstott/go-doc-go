// +build integration

package contentsource

import (
	"path/filepath"
	"testing"
)

func TestFileContentSource_Integration(t *testing.T) {
	// Use actual test assets directory
	assetsDir, err := filepath.Abs("../../tests/assets")
	if err != nil {
		t.Fatalf("Failed to get assets path: %v", err)
	}

	config := map[string]interface{}{
		"name":      "test-assets",
		"type":      "file",
		"base_path": assetsDir,
	}

	source, err := NewContentSource(config)
	if err != nil {
		t.Fatalf("NewContentSource() error = %v", err)
	}

	// List documents
	docs, err := source.ListDocuments()
	if err != nil {
		t.Fatalf("ListDocuments() error = %v", err)
	}

	t.Logf("Found %d documents in test assets", len(docs))

	if len(docs) == 0 {
		t.Error("Expected at least one document in test assets")
	}

	// Fetch first document
	if len(docs) > 0 {
		doc, err := source.FetchDocument(docs[0].ID)
		if err != nil {
			t.Fatalf("FetchDocument() error = %v", err)
		}

		t.Logf("Fetched document: %s, Type: %s", doc.ID, doc.DocType)

		if doc.ID == "" {
			t.Error("Document ID is empty")
		}
		if doc.DocType == "" {
			t.Error("Document type is empty")
		}
		if doc.BinaryPath == "" && doc.Content == "" {
			t.Error("Both BinaryPath and Content are empty")
		}
	}
}
