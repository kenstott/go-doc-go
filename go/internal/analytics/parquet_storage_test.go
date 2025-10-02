package analytics

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParquetStorage_AppendDocuments(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()

	// Create storage
	config := map[string]interface{}{
		"path":         tempDir,
		"type":         "parquet",
		"partitioning": []interface{}{"date", "source"},
	}

	storage, err := NewParquetStorage(config)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	// Create test documents
	docs := []Document{
		{
			DocID:             "doc1",
			SourceName:        "test-source",
			Title:             "Test Document 1",
			URL:               "http://example.com/doc1",
			ContentType:       "text/html",
			ProcessedAt:       time.Now(),
			ElementCount:      10,
			RelationshipCount: 5,
		},
		{
			DocID:             "doc2",
			SourceName:        "test-source",
			Title:             "Test Document 2",
			URL:               "http://example.com/doc2",
			ContentType:       "text/html",
			ProcessedAt:       time.Now(),
			ElementCount:      15,
			RelationshipCount: 8,
		},
	}

	// Write documents
	if err := storage.AppendDocuments(docs); err != nil {
		t.Fatalf("Failed to append documents: %v", err)
	}

	// Verify files were created
	date := time.Now().Format("2006-01-02")
	expectedPath := filepath.Join(tempDir, "documents", "date="+date, "source=test-source")

	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Fatalf("Expected directory not created: %s", expectedPath)
	}

	// Check for parquet files
	matches, err := filepath.Glob(filepath.Join(expectedPath, "*.parquet"))
	if err != nil {
		t.Fatalf("Failed to glob parquet files: %v", err)
	}
	if len(matches) == 0 {
		t.Fatal("No parquet files created")
	}

	t.Logf("Successfully created %d parquet file(s) at %s", len(matches), expectedPath)
}

func TestParquetStorage_AppendElements(t *testing.T) {
	tempDir := t.TempDir()

	config := map[string]interface{}{
		"path":         tempDir,
		"type":         "parquet",
		"partitioning": []interface{}{"date", "source"},
	}

	storage, err := NewParquetStorage(config)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	// Create test elements
	elements := []Element{
		{
			ElementID:      "elem1",
			DocID:          "doc1",
			SourceName:     "test-source",
			ElementType:    "paragraph",
			ContentPreview: "This is a test paragraph",
			ParentID:       "",
		},
		{
			ElementID:      "elem2",
			DocID:          "doc1",
			SourceName:     "test-source",
			ElementType:    "heading",
			ContentPreview: "Test Heading",
			ParentID:       "elem1",
		},
	}

	// Write elements
	if err := storage.AppendElements(elements); err != nil {
		t.Fatalf("Failed to append elements: %v", err)
	}

	// Verify files were created
	date := time.Now().Format("2006-01-02")
	expectedPath := filepath.Join(tempDir, "elements", "date="+date, "source=test-source")

	matches, err := filepath.Glob(filepath.Join(expectedPath, "*.parquet"))
	if err != nil {
		t.Fatalf("Failed to glob parquet files: %v", err)
	}
	if len(matches) == 0 {
		t.Fatal("No parquet files created")
	}

	t.Logf("Successfully created %d element parquet file(s)", len(matches))
}

func TestParquetStorage_AppendEmbeddings(t *testing.T) {
	tempDir := t.TempDir()

	config := map[string]interface{}{
		"path":         tempDir,
		"type":         "parquet",
		"partitioning": []interface{}{"date", "source"},
	}

	storage, err := NewParquetStorage(config)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	// Create test embeddings (384 dimensions like all-MiniLM-L6-v2)
	embedding := make([]float64, 384)
	for i := range embedding {
		embedding[i] = float64(i) * 0.01
	}

	embeddings := []Embedding{
		{
			ElementID:  "elem1",
			DocID:      "doc1",
			SourceName: "test-source",
			Embedding:  embedding,
			Text:       "Test text for embedding",
		},
	}

	// Write embeddings
	if err := storage.AppendEmbeddings(embeddings); err != nil {
		t.Fatalf("Failed to append embeddings: %v", err)
	}

	// Verify files were created
	date := time.Now().Format("2006-01-02")
	expectedPath := filepath.Join(tempDir, "embeddings", "date="+date, "source=test-source")

	matches, err := filepath.Glob(filepath.Join(expectedPath, "*.parquet"))
	if err != nil {
		t.Fatalf("Failed to glob parquet files: %v", err)
	}
	if len(matches) == 0 {
		t.Fatal("No parquet files created")
	}

	t.Logf("Successfully created %d embedding parquet file(s)", len(matches))
}

func TestParquetStorage_Factory(t *testing.T) {
	tempDir := t.TempDir()

	config := map[string]interface{}{
		"path": tempDir,
		"type": "parquet",
	}

	storage, err := NewStorage(config)
	if err != nil {
		t.Fatalf("Failed to create storage via factory: %v", err)
	}
	defer storage.Close()

	// Verify it's the right type
	if _, ok := storage.(*ParquetStorage); !ok {
		t.Fatalf("Expected *ParquetStorage, got %T", storage)
	}

	t.Log("Factory successfully created ParquetStorage")
}
