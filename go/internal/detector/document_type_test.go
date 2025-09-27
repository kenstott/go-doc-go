package detector

import (
	"testing"
)

func TestNewDocumentTypeDetector(t *testing.T) {
	detector := NewDocumentTypeDetector()
	if detector == nil {
		t.Fatal("Failed to create detector")
	}
	if detector.mimeTypeMap == nil {
		t.Error("MIME type map not initialized")
	}
	if detector.extensionMap == nil {
		t.Error("Extension map not initialized")
	}
	if detector.binarySignatures == nil {
		t.Error("Binary signatures not initialized")
	}
}

func TestDetectFromPath(t *testing.T) {
	detector := NewDocumentTypeDetector()

	tests := []struct {
		path     string
		expected string
		method   string
	}{
		{"document.pdf", "pdf", "extension"},
		{"spreadsheet.xlsx", "xlsx", "extension"},
		{"presentation.pptx", "pptx", "extension"},
		{"text.txt", "text", "extension"},
		{"data.csv", "csv", "extension"},
		{"config.json", "json", "extension"},
		{"style.xml", "xml", "extension"},
		{"readme.md", "markdown", "extension"},
		{"page.html", "html", "extension"},
		{"unknown.xyz", "text", "default"},
		{"", "text", "default"},
	}

	for _, test := range tests {
		docType, method := detector.DetectFromPath(test.path)
		if docType != test.expected {
			t.Errorf("DetectFromPath(%s) = %s, want %s", test.path, docType, test.expected)
		}
		if method != test.method {
			t.Errorf("DetectFromPath(%s) method = %s, want %s", test.path, method, test.method)
		}
	}
}

func TestDetectFromMime(t *testing.T) {
	detector := NewDocumentTypeDetector()

	tests := []struct {
		path     string
		expected string
	}{
		{"document.pdf", "pdf"},
		{"data.csv", "csv"},
		{"config.json", "json"},
		{"style.xml", "xml"},
		{"readme.md", "markdown"},
		{"page.html", "html"},
		{"unknown.xyz", "text"},
	}

	for _, test := range tests {
		docType, _ := detector.DetectFromMime(test.path)
		if docType != test.expected {
			t.Errorf("DetectFromMime(%s) = %s, want %s", test.path, docType, test.expected)
		}
	}
}

func TestDetectFromContent(t *testing.T) {
	detector := NewDocumentTypeDetector()

	tests := []struct {
		content  string
		metadata map[string]string
		expected string
		method   string
	}{
		// JSON content
		{`{"key": "value"}`, nil, "json", "content"},
		{`[{"item": 1}, {"item": 2}]`, nil, "json", "content"},

		// XML content
		{`<?xml version="1.0"?><root></root>`, nil, "xml", "content"},
		{`<note><to>User</to><from>System</from></note>`, nil, "xml", "content"},

		// HTML content
		{`<!DOCTYPE html><html><body>Hello</body></html>`, nil, "html", "content"},
		{`<div class="content"><p>Hello world</p></div>`, nil, "html", "content"},

		// Markdown content
		{"# Header\n\nSome content", nil, "markdown", "content"},
		{"## Subheader\n\n- List item", nil, "markdown", "content"},

		// CSV content
		{"name,age,city\nJohn,30,NYC\nJane,25,LA", nil, "csv", "content"},
		{"col1\tcol2\tcol3\nval1\tval2\tval3", nil, "csv", "content"},

		// PDF binary signature
		{"%PDF-1.4\nSome PDF content", nil, "pdf", "signature"},

		// Plain text
		{"This is just plain text content", nil, "text", "content"},
		{"", nil, "text", "content"},

		// Metadata hints
		{"Some content", map[string]string{"content_type": "application/json"}, "json", "metadata"},
		{"Some content", map[string]string{"content_column": "data_html"}, "html", "metadata"},
		{"Some content", map[string]string{"content_column": "notes_md"}, "markdown", "metadata"},
	}

	for _, test := range tests {
		docType, method := detector.DetectFromContent([]byte(test.content), test.metadata)
		if docType != test.expected {
			t.Errorf("DetectFromContent(%q, %v) = %s, want %s", test.content, test.metadata, docType, test.expected)
		}
		if method != test.method {
			t.Errorf("DetectFromContent(%q, %v) method = %s, want %s", test.content, test.metadata, method, test.method)
		}
	}
}

func TestIsLikelyCSV(t *testing.T) {
	detector := NewDocumentTypeDetector()

	tests := []struct {
		content  string
		expected bool
	}{
		// Valid CSV
		{"name,age,city\nJohn,30,NYC\nJane,25,LA", true},
		{"col1,col2,col3\nval1,val2,val3\nval4,val5,val6", true},

		// Tab-separated
		{"name\tage\tcity\nJohn\t30\tNYC\nJane\t25\tLA", true},

		// Semicolon-separated
		{"name;age;city\nJohn;30;NYC\nJane;25;LA", true},

		// Pipe-separated
		{"name|age|city\nJohn|30|NYC\nJane|25|LA", true},

		// Not CSV
		{"This is just plain text", false},
		{"name,age\nJohn\nJane,25,LA,Extra", false}, // Inconsistent columns
		{"", false}, // Empty
		{"name\nJohn\nJane", false}, // No delimiters
	}

	for _, test := range tests {
		result := detector.isLikelyCSV(test.content)
		if result != test.expected {
			t.Errorf("isLikelyCSV(%q) = %v, want %v", test.content, result, test.expected)
		}
	}
}

func TestDetect(t *testing.T) {
	detector := NewDocumentTypeDetector()

	tests := []struct {
		path     string
		content  string
		metadata map[string]string
		expected string
	}{
		// Path takes precedence
		{"document.pdf", "This is not PDF content", nil, "pdf"},
		{"", `{"key": "value"}`, nil, "json"},

		// Content detection when path is unknown
		{"unknown.file", `{"key": "value"}`, nil, "json"},

		// Metadata hints
		{"", "Some content", map[string]string{"content_type": "application/xml"}, "xml"},

		// Default fallback
		{"", "", nil, "text"},
	}

	for _, test := range tests {
		var content []byte
		if test.content != "" {
			content = []byte(test.content)
		}

		docType, _ := detector.Detect(test.path, content, test.metadata)
		if docType != test.expected {
			t.Errorf("Detect(%s, %q, %v) = %s, want %s", test.path, test.content, test.metadata, docType, test.expected)
		}
	}
}

func TestDetectionResponseJSON(t *testing.T) {
	response := &DetectionResponse{
		DocumentType: "pdf",
		Method:       "extension",
	}

	jsonStr, err := response.ToJSON()
	if err != nil {
		t.Fatalf("Failed to convert to JSON: %v", err)
	}

	expected := `{"document_type":"pdf","method":"extension"}`
	if jsonStr != expected {
		t.Errorf("ToJSON() = %s, want %s", jsonStr, expected)
	}
}

func TestDetectionRequestFromJSON(t *testing.T) {
	jsonStr := `{"path":"test.pdf","content":"base64content","metadata":{"key":"value"}}`

	var request DetectionRequest
	err := request.FromJSON(jsonStr)
	if err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if request.Path != "test.pdf" {
		t.Errorf("Path = %s, want test.pdf", request.Path)
	}
	if request.Content != "base64content" {
		t.Errorf("Content = %s, want base64content", request.Content)
	}
	if request.Metadata["key"] != "value" {
		t.Errorf("Metadata[key] = %s, want value", request.Metadata["key"])
	}
}

func TestBinaryContent(t *testing.T) {
	detector := NewDocumentTypeDetector()

	// Test with binary content that can't be decoded as UTF-8
	binaryContent := []byte{0xFF, 0xFE, 0x00, 0x01, 0x02, 0x03}
	docType, method := detector.DetectFromContent(binaryContent, nil)

	if docType != "binary" {
		t.Errorf("Binary content detection = %s, want binary", docType)
	}
	if method != "content" {
		t.Errorf("Binary content method = %s, want content", method)
	}
}