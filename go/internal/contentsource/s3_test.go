package contentsource

import (
	"testing"
)

func TestExtractBucketAndKey(t *testing.T) {
	tests := []struct {
		name       string
		s3URI      string
		wantBucket string
		wantKey    string
	}{
		{
			name:       "full S3 URI",
			s3URI:      "s3://my-bucket/path/to/file.txt",
			wantBucket: "my-bucket",
			wantKey:    "path/to/file.txt",
		},
		{
			name:       "key only",
			s3URI:      "path/to/file.txt",
			wantBucket: "",
			wantKey:    "path/to/file.txt",
		},
		{
			name:       "S3 URI with bucket only",
			s3URI:      "s3://my-bucket",
			wantBucket: "my-bucket",
			wantKey:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotBucket, gotKey := extractBucketAndKey(tt.s3URI)
			if gotBucket != tt.wantBucket {
				t.Errorf("extractBucketAndKey() bucket = %v, want %v", gotBucket, tt.wantBucket)
			}
			if gotKey != tt.wantKey {
				t.Errorf("extractBucketAndKey() key = %v, want %v", gotKey, tt.wantKey)
			}
		})
	}
}

func TestIsValidUTF8(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want bool
	}{
		{
			name: "valid ASCII",
			data: []byte("Hello, World!"),
			want: true,
		},
		{
			name: "valid UTF-8",
			data: []byte("Hello, 世界!"),
			want: true,
		},
		{
			name: "binary with null bytes",
			data: []byte{0x00, 0x01, 0x02},
			want: false,
		},
		{
			name: "empty",
			data: []byte{},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidUTF8(tt.data); got != tt.want {
				t.Errorf("isValidUTF8() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDetectDocType(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		contentType string
		isBinary    bool
		want        string
	}{
		{
			name:        "markdown file",
			key:         "document.md",
			contentType: "text/markdown",
			isBinary:    false,
			want:        "markdown",
		},
		{
			name:        "HTML file",
			key:         "page.html",
			contentType: "text/html",
			isBinary:    false,
			want:        "html",
		},
		{
			name:        "JSON file",
			key:         "data.json",
			contentType: "application/json",
			isBinary:    false,
			want:        "json",
		},
		{
			name:        "PDF file",
			key:         "document.pdf",
			contentType: "application/pdf",
			isBinary:    true,
			want:        "pdf",
		},
		{
			name:        "DOCX file",
			key:         "document.docx",
			contentType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
			isBinary:    true,
			want:        "docx",
		},
		{
			name:        "unknown binary",
			key:         "file.dat",
			contentType: "application/octet-stream",
			isBinary:    true,
			want:        "binary",
		},
		{
			name:        "text file",
			key:         "file.txt",
			contentType: "text/plain",
			isBinary:    false,
			want:        "text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := detectDocType(tt.key, tt.contentType, tt.isBinary); got != tt.want {
				t.Errorf("detectDocType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGuessDocType(t *testing.T) {
	tests := []struct {
		name      string
		extension string
		want      string
	}{
		{"markdown", "md", "markdown"},
		{"markdown long", "markdown", "markdown"},
		{"HTML", "html", "html"},
		{"HTML short", "htm", "html"},
		{"text", "txt", "text"},
		{"unknown", "xyz", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := guessDocType(tt.extension); got != tt.want {
				t.Errorf("guessDocType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewS3ContentSource(t *testing.T) {
	tests := []struct {
		name    string
		config  map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid config",
			config: map[string]interface{}{
				"name":        "test-s3-source",
				"bucket_name": "test-bucket",
				"region_name": "us-east-1",
			},
			wantErr: false,
		},
		{
			name: "missing bucket",
			config: map[string]interface{}{
				"name": "test-source",
			},
			wantErr: true,
		},
		{
			name: "with filters",
			config: map[string]interface{}{
				"name":               "test-source",
				"bucket_name":        "test-bucket",
				"include_extensions": []interface{}{"txt", "md"},
				"exclude_extensions": []interface{}{"tmp"},
				"include_patterns":   []interface{}{".*important.*"},
				"exclude_patterns":   []interface{}{".*temp.*"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewS3ContentSource(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewS3ContentSource() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestS3ContentSource_shouldIncludeObject(t *testing.T) {
	config := map[string]interface{}{
		"name":               "test-source",
		"bucket_name":        "test-bucket",
		"include_extensions": []interface{}{"txt", "md"},
		"exclude_prefixes":   []interface{}{"tmp/"},
	}

	source, err := NewS3ContentSource(config)
	if err != nil {
		t.Fatalf("NewS3ContentSource() error = %v", err)
	}

	tests := []struct {
		name string
		key  string
		want bool
	}{
		{"included extension", "file.txt", true},
		{"included extension 2", "file.md", true},
		{"excluded extension", "file.pdf", false},
		{"excluded prefix", "tmp/file.txt", false},
		{"normal file", "documents/file.txt", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := source.shouldIncludeObject(tt.key); got != tt.want {
				t.Errorf("shouldIncludeObject(%v) = %v, want %v", tt.key, got, tt.want)
			}
		})
	}
}

// Note: FetchDocument, ListDocuments, and HasChanged tests require actual S3 access
// or a mock S3 server (like localstack). These would be integration tests rather than
// unit tests. For a full test suite, you would want to:
// 1. Use testcontainers-go with localstack for integration tests
// 2. Or use AWS SDK's mock interfaces for unit testing the S3 client interactions
