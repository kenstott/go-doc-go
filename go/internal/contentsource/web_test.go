package contentsource

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewWebContentSource(t *testing.T) {
	tests := []struct {
		name   string
		config map[string]interface{}
	}{
		{
			name: "with full config",
			config: map[string]interface{}{
				"name":             "test-web-source",
				"base_url":         "https://example.com",
				"url_list":         []interface{}{"https://example.com/page1", "https://example.com/page2"},
				"refresh_interval": 3600,
				"headers":          map[string]interface{}{"User-Agent": "Test-Agent"},
				"include_patterns": []interface{}{".*page.*"},
				"exclude_patterns": []interface{}{".*admin.*"},
				"max_link_depth":   3,
			},
		},
		{
			name: "with defaults",
			config: map[string]interface{}{
				"name": "test-source",
			},
		},
		{
			name: "with basic auth",
			config: map[string]interface{}{
				"name": "test-source",
				"authentication": map[string]interface{}{
					"type":     "basic",
					"username": "user",
					"password": "pass",
				},
			},
		},
		{
			name: "with bearer token",
			config: map[string]interface{}{
				"name": "test-source",
				"authentication": map[string]interface{}{
					"type":  "bearer",
					"token": "test-token",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source, err := NewWebContentSource(tt.config)
			if err != nil {
				t.Fatalf("NewWebContentSource() error = %v", err)
			}
			if source == nil {
				t.Fatal("NewWebContentSource() returned nil")
			}
		})
	}
}

func TestWebContentSource_FetchDocument(t *testing.T) {
	// Create test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/test.html" {
			w.Header().Set("Content-Type", "text/html")
			w.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
			fmt.Fprint(w, "<html><body>Test Content</body></html>")
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	config := map[string]interface{}{
		"name":     "test-source",
		"base_url": server.URL,
	}

	source, err := NewWebContentSource(config)
	if err != nil {
		t.Fatalf("NewWebContentSource() error = %v", err)
	}

	t.Run("fetch valid URL", func(t *testing.T) {
		doc, err := source.FetchDocument(server.URL + "/test.html")
		if err != nil {
			t.Fatalf("FetchDocument() error = %v", err)
		}
		if doc.Content != "<html><body>Test Content</body></html>" {
			t.Errorf("FetchDocument() content = %v, want %v", doc.Content, "<html><body>Test Content</body></html>")
		}
		if doc.DocType != "html" {
			t.Errorf("FetchDocument() docType = %v, want %v", doc.DocType, "html")
		}
	})

	t.Run("fetch invalid URL", func(t *testing.T) {
		_, err := source.FetchDocument(server.URL + "/notfound.html")
		if err == nil {
			t.Error("FetchDocument() expected error for 404")
		}
	})

	t.Run("cache hit", func(t *testing.T) {
		// First fetch
		doc1, err := source.FetchDocument(server.URL + "/test.html")
		if err != nil {
			t.Fatalf("FetchDocument() error = %v", err)
		}

		// Second fetch should hit cache
		doc2, err := source.FetchDocument(server.URL + "/test.html")
		if err != nil {
			t.Fatalf("FetchDocument() error = %v", err)
		}

		// Should be same instance from cache
		if doc1.ContentHash != doc2.ContentHash {
			t.Error("FetchDocument() cache not working")
		}
	})
}

func TestWebContentSource_ListDocuments(t *testing.T) {
	config := map[string]interface{}{
		"name":     "test-source",
		"base_url": "https://example.com",
		"url_list": []interface{}{
			"https://example.com/page1",
			"https://example.com/page2",
			"/relative/path",
		},
	}

	source, err := NewWebContentSource(config)
	if err != nil {
		t.Fatalf("NewWebContentSource() error = %v", err)
	}

	docs, err := source.ListDocuments()
	if err != nil {
		t.Fatalf("ListDocuments() error = %v", err)
	}

	if len(docs) != 3 {
		t.Errorf("ListDocuments() found %d docs, want 3", len(docs))
	}

	// Check that relative URL was resolved
	found := false
	for _, doc := range docs {
		if doc.ID == "https://example.com/relative/path" {
			found = true
			break
		}
	}
	if !found {
		t.Error("ListDocuments() did not resolve relative URL")
	}
}

func TestWebContentSource_HasChanged(t *testing.T) {
	lastModTime := time.Now().Add(-1 * time.Hour)

	// Create test server with Last-Modified support
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "HEAD" {
			// Check If-Modified-Since header
			if ims := r.Header.Get("If-Modified-Since"); ims != "" {
				imsTime, err := http.ParseTime(ims)
				if err == nil && !lastModTime.After(imsTime) {
					w.WriteHeader(http.StatusNotModified)
					return
				}
			}
			w.Header().Set("Last-Modified", lastModTime.UTC().Format(http.TimeFormat))
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	config := map[string]interface{}{
		"name": "test-source",
	}

	source, err := NewWebContentSource(config)
	if err != nil {
		t.Fatalf("NewWebContentSource() error = %v", err)
	}

	t.Run("no previous timestamp returns true", func(t *testing.T) {
		changed, err := source.HasChanged(server.URL, nil)
		if err != nil {
			t.Fatalf("HasChanged() error = %v", err)
		}
		if !changed {
			t.Error("HasChanged() should return true with no previous timestamp")
		}
	})

	t.Run("detects unchanged content", func(t *testing.T) {
		// Use time slightly after lastModTime - should return 304 Not Modified
		futureTime := lastModTime.Add(1 * time.Minute)
		changed, err := source.HasChanged(server.URL, float64(futureTime.Unix()))
		if err != nil {
			t.Fatalf("HasChanged() error = %v", err)
		}
		if changed {
			t.Error("HasChanged() should return false for unchanged content")
		}
	})

	t.Run("detects changed content", func(t *testing.T) {
		// Use old timestamp - content is newer
		oldTime := lastModTime.Add(-2 * time.Hour)
		changed, err := source.HasChanged(server.URL, float64(oldTime.Unix()))
		if err != nil {
			t.Fatalf("HasChanged() error = %v", err)
		}
		if !changed {
			t.Error("HasChanged() should return true for modified content")
		}
	})
}

func TestWebContentSource_ExtractLinks(t *testing.T) {
	server := httptest.NewServer(nil)
	defer server.Close()

	config := map[string]interface{}{
		"name": "test-source",
	}

	source, err := NewWebContentSource(config)
	if err != nil {
		t.Fatalf("NewWebContentSource() error = %v", err)
	}

	content := fmt.Sprintf(`
		<html>
		<body>
			<a href="%s/page1">Link 1</a>
			<a href="%s/page2">Link 2</a>
			<a href="https://external.com/page">External</a>
			<a href="#fragment">Fragment</a>
		</body>
		</html>
	`, server.URL, server.URL)

	links, err := source.ExtractLinks(content, server.URL)
	if err != nil {
		t.Fatalf("ExtractLinks() error = %v", err)
	}

	// Should extract 2 links (page1, page2) - excluding external and fragment
	if len(links) != 2 {
		t.Errorf("ExtractLinks() found %d links, want 2", len(links))
	}
}

func TestWebContentSource_Authentication(t *testing.T) {
	// Test server that checks for authentication
	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check basic auth
		user, pass, ok := r.BasicAuth()
		if !ok || user != "testuser" || pass != "testpass" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "Authenticated")
	}))
	defer authServer.Close()

	t.Run("basic auth", func(t *testing.T) {
		config := map[string]interface{}{
			"name": "test-source",
			"authentication": map[string]interface{}{
				"type":     "basic",
				"username": "testuser",
				"password": "testpass",
			},
		}

		source, err := NewWebContentSource(config)
		if err != nil {
			t.Fatalf("NewWebContentSource() error = %v", err)
		}

		doc, err := source.FetchDocument(authServer.URL)
		if err != nil {
			t.Fatalf("FetchDocument() error = %v", err)
		}

		if doc.Content != "Authenticated" {
			t.Errorf("FetchDocument() content = %v, want %v", doc.Content, "Authenticated")
		}
	})

	t.Run("bearer token", func(t *testing.T) {
		bearerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if auth != "Bearer test-token" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "Token Valid")
		}))
		defer bearerServer.Close()

		config := map[string]interface{}{
			"name": "test-source",
			"authentication": map[string]interface{}{
				"type":  "bearer",
				"token": "test-token",
			},
		}

		source, err := NewWebContentSource(config)
		if err != nil {
			t.Fatalf("NewWebContentSource() error = %v", err)
		}

		doc, err := source.FetchDocument(bearerServer.URL)
		if err != nil {
			t.Fatalf("FetchDocument() error = %v", err)
		}

		if doc.Content != "Token Valid" {
			t.Errorf("FetchDocument() content = %v, want %v", doc.Content, "Token Valid")
		}
	})
}
