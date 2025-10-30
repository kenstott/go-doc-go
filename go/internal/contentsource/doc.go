// Package contentsource provides content ingestion from various sources.
//
// The contentsource package implements the ContentSource interface for fetching
// documents from different storage systems and protocols. Content sources populate
// the job queue with documents to be processed by workers.
//
// # Supported Content Sources
//
// **File System**:
//   - Local directories and files
//   - Glob pattern matching (**, *, ?)
//   - Recursive directory traversal
//   - Change detection via file modification time
//
// **Web/HTTP**:
//   - HTTP/HTTPS URLs
//   - HTML pages
//   - Hyperlink discovery and crawling
//   - User-agent and headers configuration
//   - Rate limiting support
//
// **AWS S3**:
//   - S3 buckets and objects
//   - Prefix-based filtering
//   - AWS credential support (IAM, env vars, profiles)
//   - Large file handling
//
// # Usage
//
// Create a content source from configuration:
//
//	// File source
//	fileSource, err := contentsource.NewFileSource(map[string]interface{}{
//		"name":         "local_docs",
//		"type":         "file",
//		"base_path":    "./documents",
//		"file_pattern": "**/*.pdf",
//	})
//
//	// Web source
//	webSource, err := contentsource.NewWebSource(map[string]interface{}{
//		"name":      "company_docs",
//		"type":      "web",
//		"base_url":  "https://docs.example.com",
//		"url_list":  []string{"https://docs.example.com/api"},
//	})
//
//	// S3 source
//	s3Source, err := contentsource.NewS3Source(map[string]interface{}{
//		"name":     "data_lake",
//		"type":     "s3",
//		"bucket":   "my-bucket",
//		"prefix":   "documents/",
//		"region":   "us-east-1",
//	})
//
// # ContentSource Interface
//
// All content sources implement three core methods:
//
//	type ContentSource interface {
//		// FetchDocument retrieves document content by ID
//		FetchDocument(sourceID string) (*DocumentContent, error)
//
//		// ListDocuments returns metadata for all documents
//		ListDocuments() ([]DocumentInfo, error)
//
//		// HasChanged detects if document changed since last fetch
//		HasChanged(sourceID string, lastModified interface{}) (bool, error)
//	}
//
// # Document Fetching
//
// Fetch document content:
//
//	content, err := source.FetchDocument("doc-123")
//	if err != nil {
//		log.Fatalf("Failed to fetch: %v", err)
//	}
//
//	fmt.Printf("Content: %s\n", content.Content)
//	fmt.Printf("Binary path: %s\n", content.BinaryPath)
//	fmt.Printf("Metadata: %+v\n", content.Metadata)
//
// DocumentContent includes:
//   - ID: Unique document identifier
//   - Content: Text content (for text-based formats)
//   - BinaryPath: File path (for binary formats like PDF, DOCX)
//   - Metadata: Source-specific metadata (URL, file path, S3 key, etc.)
//   - ContentHash: SHA256 hash for change detection
//   - DocType: Detected document type (pdf, html, docx, etc.)
//
// # Document Listing
//
// List all available documents:
//
//	docs, err := source.ListDocuments()
//	for _, doc := range docs {
//		fmt.Printf("ID: %s, Type: %s\n", doc.ID, doc.DocType)
//	}
//
// This is used to populate the job queue initially.
//
// # Change Detection
//
// Detect document changes for incremental processing:
//
//	lastMod := time.Now().Add(-24 * time.Hour)
//	changed, err := source.HasChanged("doc-123", lastMod)
//	if changed {
//		// Reprocess the document
//		content, _ := source.FetchDocument("doc-123")
//	}
//
// Change detection strategies:
//   - File: Modification time comparison
//   - Web: ETag or Last-Modified headers
//   - S3: Object LastModified timestamp
//
// # File Source Configuration
//
//	[[content_sources]]
//	name = "local_documents"
//	type = "file"
//	base_path = "./documents"
//	file_pattern = "**/*.{pdf,docx,xlsx}"  # Glob pattern
//
//	[content_sources.discovery]
//	enabled = true
//	max_depth = 2
//
// **File Pattern Syntax**:
//   - `*.pdf`: All PDFs in base_path
//   - `**/*.pdf`: All PDFs recursively
//   - `docs/**/*.{pdf,docx}`: PDFs and DOCX files in docs/
//
// # Web Source Configuration
//
//	[[content_sources]]
//	name = "company_website"
//	type = "web"
//	base_url = "https://example.com"
//	url_list = [
//		"https://example.com/docs",
//		"https://example.com/api"
//	]
//
//	[content_sources.discovery]
//	enabled = true
//	max_depth = 3
//	include_patterns = ["^https://example\\.com/"]
//	exclude_patterns = ["/archive/", "/old/"]
//
//	[content_sources.discovery.hyperlinks]
//	enabled = true
//	skip_external_domains = true
//
// **URL Processing**:
//   - Absolute URLs used as-is
//   - Relative URLs resolved against base_url
//   - Fragment identifiers (#section) removed
//   - Query parameters preserved
//
// # S3 Source Configuration
//
//	[[content_sources]]
//	name = "data_lake"
//	type = "s3"
//	bucket = "my-data-bucket"
//	prefix = "documents/"
//	region = "us-east-1"
//
//	# Authentication (optional, uses AWS SDK default chain)
//	access_key_id = "AKIA..."
//	secret_access_key = "..."
//	session_token = "..."  # For temporary credentials
//
// **AWS Credential Chain**:
//   1. Configuration (access_key_id, secret_access_key)
//   2. Environment variables (AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY)
//   3. AWS credentials file (~/.aws/credentials)
//   4. IAM role (for EC2/ECS/Lambda)
//
// # Factory Pattern
//
// Create content sources dynamically:
//
//	import "github.com/kennethstott/doculyzer-go-conversion/internal/contentsource"
//
//	config := map[string]interface{}{
//		"name": "my_source",
//		"type": "file",  // or "web", "s3"
//		// ... type-specific config
//	}
//
//	source, err := contentsource.NewContentSource(config)
//	if err != nil {
//		log.Fatal(err)
//	}
//
// The factory inspects the "type" field and creates the appropriate implementation.
//
// # Document IDs
//
// Document IDs are source-specific:
//
//   - File: Relative file path (e.g., "docs/guide.pdf")
//   - Web: Full URL (e.g., "https://example.com/page.html")
//   - S3: S3 key (e.g., "documents/2024/report.pdf")
//
// IDs must be stable - the same document should always have the same ID.
//
// # Metadata
//
// Each content source provides source-specific metadata:
//
// **File Source**:
//   - file_path: Absolute file path
//   - file_name: Base filename
//   - file_size: Size in bytes
//   - last_modified: Modification timestamp
//   - content_type: MIME type (detected)
//
// **Web Source**:
//   - url: Full URL
//   - title: HTML <title> (if HTML)
//   - content_type: Content-Type header
//   - last_modified: Last-Modified header
//   - etag: ETag header
//
// **S3 Source**:
//   - s3_bucket: Bucket name
//   - s3_key: Object key
//   - s3_size: Object size in bytes
//   - last_modified: LastModified timestamp
//   - content_type: ContentType metadata
//   - etag: ETag
//
// # Binary vs. Text Content
//
// Content sources handle both text and binary documents:
//
// **Text Content** (HTML, Markdown, JSON, CSV):
//   - Content field populated with text
//   - BinaryPath empty
//   - Parser receives content directly
//
// **Binary Content** (PDF, DOCX, XLSX, images):
//   - Content field empty
//   - BinaryPath populated with file path
//   - Parser reads from BinaryPath
//
// For web sources, binary content is downloaded to a temporary file.
//
// # Error Handling
//
// Content sources return errors for:
//
//   - Network failures (web, S3)
//   - File not found (file, S3)
//   - Permission denied (file, S3)
//   - Invalid URLs (web)
//   - AWS credential errors (S3)
//
// Errors are wrapped with context using fmt.Errorf with %w.
//
// # Performance
//
// **File Source**:
//   - Listing: O(N) directory traversal
//   - Fetching: O(1) file read
//   - Memory: Minimal (streams large files)
//
// **Web Source**:
//   - Listing: Depends on initial URL count
//   - Fetching: Network latency (100ms - 5s)
//   - Memory: Downloads stored temporarily
//   - Rate limiting: Configurable delay between requests
//
// **S3 Source**:
//   - Listing: O(N) with pagination (1000 objects/request)
//   - Fetching: Network latency (100ms - 2s)
//   - Memory: Streams large objects
//
// # Discovery Integration
//
// Content sources integrate with the discovery system for crawling:
//
// **Hyperlinks** (Web/File):
//   - Parser extracts hyperlink elements
//   - Worker resolves relative URLs
//   - Worker applies include/exclude patterns
//   - Worker queues discovered URLs
//
// **Code Dependencies** (File):
//   - Parser extracts import statements
//   - Worker resolves import paths to file paths
//   - Worker applies package filters
//   - Worker queues discovered files
//
// See docs/features/discovery/ for details.
//
// # Concurrency
//
// Content sources are thread-safe for concurrent reads:
//
//	var wg sync.WaitGroup
//	for _, docID := range documentIDs {
//		wg.Add(1)
//		go func(id string) {
//			defer wg.Done()
//			content, _ := source.FetchDocument(id)
//			// Process content
//		}(docID)
//	}
//	wg.Wait()
//
// # See Also
//
//   - Worker: go/internal/worker/ (document processing orchestration)
//   - Job Control: go/internal/jobcontrol/ (document queue management)
//   - Discovery: docs/features/discovery/ (crawling configuration)
//   - Configuration: docs/configuration/sources.md
//   - Examples: examples/distributed-workers/
package contentsource
