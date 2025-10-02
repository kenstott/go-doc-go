package contentsource

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewFileContentSource(t *testing.T) {
	tests := []struct {
		name   string
		config map[string]interface{}
		want   *FileContentSource
	}{
		{
			name: "with full config",
			config: map[string]interface{}{
				"name":                "test-file-source",
				"base_path":           ".",
				"file_pattern":        "**/*.txt",
				"include_extensions":  []interface{}{"txt", "md"},
				"exclude_extensions":  []interface{}{"tmp"},
				"watch_for_changes":   false,
				"recursive":           false,
				"max_link_depth":      3,
			},
		},
		{
			name: "with defaults",
			config: map[string]interface{}{
				"name": "test-source",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source, err := NewFileContentSource(tt.config)
			if err != nil {
				t.Fatalf("NewFileContentSource() error = %v", err)
			}
			if source == nil {
				t.Fatal("NewFileContentSource() returned nil")
			}
		})
	}
}

func TestGetDocTypeAndMode(t *testing.T) {
	tests := []struct {
		ext          string
		wantDocType  string
		wantReadMode string
	}{
		{"md", "markdown", "text"},
		{"markdown", "markdown", "text"},
		{"txt", "text", "text"},
		{"html", "html", "text"},
		{"json", "json", "text"},
		{"pdf", "pdf", "binary"},
		{"docx", "docx", "binary"},
		{"xlsx", "xlsx", "binary"},
		{"png", "image", "binary"},
		{"unknown", "text", "text"},
	}

	for _, tt := range tests {
		t.Run(tt.ext, func(t *testing.T) {
			gotDocType, gotReadMode := getDocTypeAndMode(tt.ext)
			if gotDocType != tt.wantDocType {
				t.Errorf("getDocTypeAndMode(%s) docType = %v, want %v", tt.ext, gotDocType, tt.wantDocType)
			}
			if gotReadMode != tt.wantReadMode {
				t.Errorf("getDocTypeAndMode(%s) readMode = %v, want %v", tt.ext, gotReadMode, tt.wantReadMode)
			}
		})
	}
}

func TestFileContentSource_FetchDocument(t *testing.T) {
	// Create temp directory with test files
	tmpDir, err := os.MkdirTemp("", "file-source-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files
	textFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(textFile, []byte("Hello, World!"), 0644); err != nil {
		t.Fatal(err)
	}

	mdFile := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(mdFile, []byte("# Header\nContent"), 0644); err != nil {
		t.Fatal(err)
	}

	config := map[string]interface{}{
		"name":      "test-source",
		"base_path": tmpDir,
	}

	source, err := NewFileContentSource(config)
	if err != nil {
		t.Fatalf("NewFileContentSource() error = %v", err)
	}

	t.Run("fetch text file", func(t *testing.T) {
		doc, err := source.FetchDocument(textFile)
		if err != nil {
			t.Fatalf("FetchDocument() error = %v", err)
		}
		if doc.Content != "Hello, World!" {
			t.Errorf("FetchDocument() content = %v, want %v", doc.Content, "Hello, World!")
		}
		if doc.DocType != "text" {
			t.Errorf("FetchDocument() docType = %v, want %v", doc.DocType, "text")
		}
		if doc.BinaryPath != "" {
			t.Errorf("FetchDocument() binaryPath should be empty for text file")
		}
	})

	t.Run("fetch markdown file", func(t *testing.T) {
		doc, err := source.FetchDocument(mdFile)
		if err != nil {
			t.Fatalf("FetchDocument() error = %v", err)
		}
		if doc.DocType != "markdown" {
			t.Errorf("FetchDocument() docType = %v, want %v", doc.DocType, "markdown")
		}
	})

	t.Run("fetch nonexistent file", func(t *testing.T) {
		_, err := source.FetchDocument(filepath.Join(tmpDir, "nonexistent.txt"))
		if err == nil {
			t.Error("FetchDocument() expected error for nonexistent file")
		}
	})
}

func TestFileContentSource_ListDocuments(t *testing.T) {
	// Create temp directory with test files
	tmpDir, err := os.MkdirTemp("", "file-source-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files
	files := []string{"test1.txt", "test2.md", "test3.pdf", "excluded.tmp"}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(tmpDir, f), []byte("content"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Create subdirectory with files
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "nested.txt"), []byte("nested"), 0644); err != nil {
		t.Fatal(err)
	}

	t.Run("list all files", func(t *testing.T) {
		config := map[string]interface{}{
			"name":      "test-source",
			"base_path": tmpDir,
		}
		source, err := NewFileContentSource(config)
		if err != nil {
			t.Fatal(err)
		}

		docs, err := source.ListDocuments()
		if err != nil {
			t.Fatalf("ListDocuments() error = %v", err)
		}

		if len(docs) != 5 { // 4 in root + 1 in subdir
			t.Errorf("ListDocuments() found %d files, want 5", len(docs))
		}
	})

	t.Run("list with include extensions", func(t *testing.T) {
		config := map[string]interface{}{
			"name":               "test-source",
			"base_path":          tmpDir,
			"include_extensions": []interface{}{"txt"},
		}
		source, err := NewFileContentSource(config)
		if err != nil {
			t.Fatal(err)
		}

		docs, err := source.ListDocuments()
		if err != nil {
			t.Fatalf("ListDocuments() error = %v", err)
		}

		if len(docs) != 2 { // test1.txt and nested.txt
			t.Errorf("ListDocuments() found %d files, want 2", len(docs))
		}
	})

	t.Run("list with exclude extensions", func(t *testing.T) {
		config := map[string]interface{}{
			"name":               "test-source",
			"base_path":          tmpDir,
			"exclude_extensions": []interface{}{"tmp"},
		}
		source, err := NewFileContentSource(config)
		if err != nil {
			t.Fatal(err)
		}

		docs, err := source.ListDocuments()
		if err != nil {
			t.Fatalf("ListDocuments() error = %v", err)
		}

		if len(docs) != 4 { // All except excluded.tmp
			t.Errorf("ListDocuments() found %d files, want 4", len(docs))
		}
	})
}

func TestFileContentSource_HasChanged(t *testing.T) {
	// Create temp file
	tmpDir, err := os.MkdirTemp("", "file-source-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	t.Run("watch disabled always returns true", func(t *testing.T) {
		config := map[string]interface{}{
			"name":              "test-source",
			"base_path":         tmpDir,
			"watch_for_changes": false,
		}
		source, err := NewFileContentSource(config)
		if err != nil {
			t.Fatal(err)
		}

		changed, err := source.HasChanged(testFile, float64(0))
		if err != nil {
			t.Fatalf("HasChanged() error = %v", err)
		}
		if !changed {
			t.Error("HasChanged() should return true when watch disabled")
		}
	})

	t.Run("no previous timestamp returns true", func(t *testing.T) {
		config := map[string]interface{}{
			"name":      "test-source",
			"base_path": tmpDir,
		}
		source, err := NewFileContentSource(config)
		if err != nil {
			t.Fatal(err)
		}

		changed, err := source.HasChanged(testFile, nil)
		if err != nil {
			t.Fatalf("HasChanged() error = %v", err)
		}
		if !changed {
			t.Error("HasChanged() should return true with no previous timestamp")
		}
	})

	t.Run("detects unchanged file", func(t *testing.T) {
		config := map[string]interface{}{
			"name":      "test-source",
			"base_path": tmpDir,
		}
		source, err := NewFileContentSource(config)
		if err != nil {
			t.Fatal(err)
		}

		info, err := os.Stat(testFile)
		if err != nil {
			t.Fatal(err)
		}

		// Use current mod time as "last modified"
		changed, err := source.HasChanged(testFile, float64(info.ModTime().Unix()))
		if err != nil {
			t.Fatalf("HasChanged() error = %v", err)
		}
		if changed {
			t.Error("HasChanged() should return false for unchanged file")
		}
	})

	t.Run("detects changed file", func(t *testing.T) {
		config := map[string]interface{}{
			"name":      "test-source",
			"base_path": tmpDir,
		}
		source, err := NewFileContentSource(config)
		if err != nil {
			t.Fatal(err)
		}

		// Use old timestamp (definitely before file creation)
		changed, err := source.HasChanged(testFile, float64(0))
		if err != nil {
			t.Fatalf("HasChanged() error = %v", err)
		}
		if !changed {
			t.Error("HasChanged() should return true for modified file")
		}
	})
}
