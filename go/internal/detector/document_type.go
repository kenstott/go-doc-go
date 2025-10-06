package detector

import (
	"encoding/json"
	"mime"
	"path/filepath"
	"regexp"
	"strings"
)

// DocumentTypeDetector provides document type detection functionality
type DocumentTypeDetector struct {
	mimeTypeMap   map[string]string
	extensionMap  map[string]string
	binarySignatures map[string]string
}

// DetectionRequest represents a request for document type detection
type DetectionRequest struct {
	Path     string            `json:"path,omitempty"`
	Content  string            `json:"content,omitempty"`  // base64 encoded for binary content
	Metadata map[string]string `json:"metadata,omitempty"`
}

// DetectionResponse represents the response from document type detection
type DetectionResponse struct {
	DocumentType string `json:"document_type"`
	Method       string `json:"method"` // "extension", "mime", "content", "signature"
}

// NewDocumentTypeDetector creates a new document type detector
func NewDocumentTypeDetector() *DocumentTypeDetector {
	return &DocumentTypeDetector{
		mimeTypeMap:   createMimeTypeMap(),
		extensionMap:  createExtensionMap(),
		binarySignatures: createBinarySignatures(),
	}
}

// createMimeTypeMap creates the MIME type to document type mapping
func createMimeTypeMap() map[string]string {
	return map[string]string{
		"text/markdown":                     "markdown",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document": "docx",
		"application/msword":                "docx",
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":       "xlsx",
		"application/vnd.ms-excel":          "xlsx",
		"application/vnd.openxmlformats-officedocument.presentationml.presentation": "pptx",
		"application/vnd.ms-powerpoint":     "pptx",
		"application/pdf":                   "pdf",
		"text/html":                         "html",
		"application/xhtml+xml":             "html",
		"text/plain":                        "text",
		"text/csv":                          "csv",
		"text/tab-separated-values":         "csv",
		"application/csv":                   "csv",
		"application/json":                  "json",
		"application/xml":                   "xml",
		"text/xml":                          "xml",
		"application/x-yaml":                "yaml",
		"text/yaml":                         "yaml",
		"application/yaml":                  "yaml",
		"image/svg+xml":                     "xml",
		"application/rdf+xml":               "xml",
		"application/rss+xml":               "xml",
		"application/xslt+xml":              "xml",
		"application/wsdl+xml":              "xml",
	}
}

// createExtensionMap creates the file extension to document type mapping
func createExtensionMap() map[string]string {
	return map[string]string{
		".md":       "markdown",
		".markdown": "markdown",
		".mdown":    "markdown",
		".docx":     "docx",
		".doc":      "docx",
		".xlsx":     "xlsx",
		".xls":      "xlsx",
		".pptx":     "pptx",
		".ppt":      "pptx",
		".pdf":      "pdf",
		".html":     "html",
		".htm":      "html",
		".xhtml":    "html",
		".txt":      "text",
		".text":     "text",
		".csv":      "csv",
		".tsv":      "csv",
		".json":     "json",
		".xml":      "xml",
		".xsd":      "xml",
		".rdf":      "xml",
		".rss":      "xml",
		".svg":      "xml",
		".wsdl":     "xml",
		".xslt":     "xml",
		".yaml":     "yaml",
		".yml":      "yaml",
	}
}

// createBinarySignatures creates the binary signature mapping
func createBinarySignatures() map[string]string {
	return map[string]string{
		"%PDF":         "pdf",
		"PK\x03\x04":   "zip", // ZIP files (could be docx, xlsx, pptx)
		"\xD0\xCF\x11\xE0\xA1\xB1\x1A\xE1": "ms_compound", // MS Compound File
	}
}

// DetectFromPath detects document type from file path
func (d *DocumentTypeDetector) DetectFromPath(path string) (string, string) {
	if path == "" {
		return "text", "default"
	}

	// Get file extension
	ext := strings.ToLower(filepath.Ext(path))

	// Check extension mapping
	if docType, exists := d.extensionMap[ext]; exists {
		return docType, "extension"
	}

	// Fallback to MIME type detection
	docType, method := d.DetectFromMime(path)
	return docType, method
}

// DetectFromMime detects document type from MIME type
func (d *DocumentTypeDetector) DetectFromMime(path string) (string, string) {
	// Get MIME type using Go's standard library
	mimeType := mime.TypeByExtension(filepath.Ext(path))

	// Remove charset parameter if present
	if semicolon := strings.Index(mimeType, ";"); semicolon != -1 {
		mimeType = mimeType[:semicolon]
	}

	// Check MIME type mapping
	if docType, exists := d.mimeTypeMap[mimeType]; exists {
		return docType, "mime"
	}

	// Special case for markdown files (Go's mime package doesn't have markdown)
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".md" || ext == ".markdown" {
		return "markdown", "mime"
	}

	// Default to text
	return "text", "default"
}

// DetectFromContent detects document type by inspecting content
func (d *DocumentTypeDetector) DetectFromContent(content []byte, metadata map[string]string) (string, string) {
	// Check metadata hints first if provided
	if metadata != nil {
		// Check explicit content type hint
		if contentType, exists := metadata["content_type"]; exists {
			if docType, mappingExists := d.mimeTypeMap[contentType]; mappingExists {
				return docType, "metadata"
			}
		}

		// Check column name hint for database content
		if contentColumn, exists := metadata["content_column"]; exists {
			if strings.HasSuffix(contentColumn, "_html") {
				return "html", "metadata"
			} else if strings.HasSuffix(contentColumn, "_md") || strings.HasSuffix(contentColumn, "_markdown") {
				return "markdown", "metadata"
			} else if strings.HasSuffix(contentColumn, "_json") {
				return "json", "metadata"
			} else if strings.HasSuffix(contentColumn, "_xml") {
				return "xml", "metadata"
			} else if strings.HasSuffix(contentColumn, "_csv") {
				return "csv", "metadata"
			}
		}
	}

	// Check binary signatures first
	contentStr := string(content)
	for signature, docType := range d.binarySignatures {
		if strings.HasPrefix(contentStr, signature) {
			// Special handling for ZIP-based Office formats
			if docType == "zip" {
				// Look for Office XML signatures in the first 4000 bytes
				contentStart := content
				if len(content) > 4000 {
					contentStart = content[:4000]
				}
				contentStartStr := string(contentStart)

				if strings.Contains(contentStartStr, "word/") {
					return "docx", "signature"
				} else if strings.Contains(contentStartStr, "xl/") {
					return "xlsx", "signature"
				} else if strings.Contains(contentStartStr, "ppt/") {
					return "pptx", "signature"
				}
			}
			return docType, "signature"
		}
	}

	// Try to decode as UTF-8 for text analysis
	if !isValidUTF8(content) {
		return "binary", "content"
	}

	// Text-based content detection
	text := string(content)

	// Check for JSON first (before CSV to avoid false positives)
	trimmed := strings.TrimSpace(text)
	if (strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}")) ||
		(strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]")) {
		// Try to parse as JSON
		var js interface{}
		if json.Unmarshal([]byte(text), &js) == nil {
			return "json", "content"
		}
	}

	// Check for CSV format
	if d.isLikelyCSV(text) {
		return "csv", "content"
	}

	// Check for markdown headers
	if matched, _ := regexp.MatchString(`(?m)^#{1,6}\s+`, text); matched {
		return "markdown", "content"
	}

	// Check for HTML
	if matched, _ := regexp.MatchString(`(?i)<!DOCTYPE html>|<html|<body|<div|<span|<p>`, text); matched {
		return "html", "content"
	}

	// Check for XML
	if strings.HasPrefix(trimmed, "<") && strings.HasSuffix(trimmed, ">") {
		if matched, _ := regexp.MatchString(`<\?xml|<[a-zA-Z]+>[^<>]*</[a-zA-Z]+>`, text); matched {
			return "xml", "content"
		}
	}

	// Default to text for string content
	return "text", "content"
}

// Detect detects document type using all available methods
func (d *DocumentTypeDetector) Detect(path string, content []byte, metadata map[string]string) (string, string) {
	// Try path-based detection first
	if path != "" {
		docType, method := d.DetectFromPath(path)
		if docType != "text" || method != "default" {
			return docType, method
		}
	}

	// Then try content-based detection with metadata hints
	if content != nil && len(content) > 0 {
		docType, method := d.DetectFromContent(content, metadata)
		if docType != "text" || method != "content" {
			return docType, method
		}
	}

	// Default to text
	return "text", "default"
}

// isLikelyCSV detects if text content is likely a CSV file
func (d *DocumentTypeDetector) isLikelyCSV(text string) bool {
	// Quick check for empty content
	if strings.TrimSpace(text) == "" {
		return false
	}

	// Get first few lines
	lines := strings.Split(text, "\n")
	if len(lines) > 5 {
		lines = lines[:5]
	}
	if len(lines) == 0 {
		return false
	}

	// Check if consistent delimiters exist
	potentialDelimiters := []string{",", "\t", ";", "|"}

	// Count delimiters in each line
	delimiterCounts := make(map[string]int)
	for _, delimiter := range potentialDelimiters {
		counts := make([]int, 0, len(lines))
		for _, line := range lines {
			count := strings.Count(line, delimiter)
			counts = append(counts, count)
		}

		// If delimiter appears consistently and at least once per line
		allHaveDelimiter := true
		minCount, maxCount := counts[0], counts[0]
		for _, count := range counts {
			if count == 0 {
				allHaveDelimiter = false
				break
			}
			if count < minCount {
				minCount = count
			}
			if count > maxCount {
				maxCount = count
			}
		}

		if allHaveDelimiter && maxCount-minCount <= 1 {
			totalCount := 0
			for _, count := range counts {
				totalCount += count
			}
			delimiterCounts[delimiter] = totalCount
		}
	}

	// If we found consistent delimiters
	if len(delimiterCounts) > 0 {
		// Choose the most frequent delimiter
		mostFrequent := ""
		maxCount := 0
		for delimiter, count := range delimiterCounts {
			if count > maxCount {
				maxCount = count
				mostFrequent = delimiter
			}
		}

		// Verify most lines have approximately same number of fields
		fieldsPerLine := make([]int, 0, len(lines))
		for _, line := range lines {
			fieldsPerLine = append(fieldsPerLine, len(strings.Split(line, mostFrequent)))
		}

		avgFields := 0.0
		for _, fields := range fieldsPerLine {
			avgFields += float64(fields)
		}
		avgFields /= float64(len(fieldsPerLine))

		// Check if field count is consistent (within 1 of average)
		for _, fields := range fieldsPerLine {
			if float64(fields) < avgFields-1 || float64(fields) > avgFields+1 {
				return false
			}
		}
		return true
	}

	return false
}

// isValidUTF8 checks if the byte slice is valid UTF-8
func isValidUTF8(data []byte) bool {
	// Check for null bytes which indicate binary content
	for _, b := range data {
		if b == 0 {
			return false
		}
	}

	// Simple check - if we can convert to string without replacement characters
	str := string(data)
	return !strings.Contains(str, "\uFFFD")
}

// ToJSON converts the detector result to JSON format
func (r *DetectionResponse) ToJSON() (string, error) {
	data, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// FromJSON creates a DetectionRequest from JSON
func (r *DetectionRequest) FromJSON(jsonStr string) error {
	return json.Unmarshal([]byte(jsonStr), r)
}