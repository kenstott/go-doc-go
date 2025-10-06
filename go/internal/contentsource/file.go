package contentsource

import (
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// FileContentSource implements ContentSource for local file systems
type FileContentSource struct {
	name              string
	basePath          string
	filePattern       string
	includeExtensions []string
	excludeExtensions []string
	watchForChanges   bool
	recursive         bool
	maxLinkDepth      int
	linkPatterns      map[string][]*regexp.Regexp
}

// FileConfig holds configuration for FileContentSource
type FileConfig struct {
	Name              string   `json:"name"`
	BasePath          string   `json:"base_path"`
	FilePattern       string   `json:"file_pattern"`
	IncludeExtensions []string `json:"include_extensions"`
	ExcludeExtensions []string `json:"exclude_extensions"`
	WatchForChanges   bool     `json:"watch_for_changes"`
	Recursive         bool     `json:"recursive"`
	MaxLinkDepth      int      `json:"max_link_depth"`
}

// NewFileContentSource creates a new file content source
func NewFileContentSource(config map[string]interface{}) (*FileContentSource, error) {
	// Extract configuration
	name, _ := config["name"].(string)
	if name == "" {
		name = "unnamed-file-source"
	}

	basePath, _ := config["base_path"].(string)
	if basePath == "" {
		basePath = "."
	}
	// Expand and make absolute
	basePath = os.ExpandEnv(basePath)
	if strings.HasPrefix(basePath, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			basePath = filepath.Join(home, basePath[1:])
		}
	}
	absPath, err := filepath.Abs(basePath)
	if err != nil {
		return nil, fmt.Errorf("invalid base_path: %w", err)
	}

	filePattern, _ := config["file_pattern"].(string)
	if filePattern == "" {
		filePattern = "**/*"
	}

	watchForChanges := true
	if val, ok := config["watch_for_changes"].(bool); ok {
		watchForChanges = val
	}

	recursive := true
	if val, ok := config["recursive"].(bool); ok {
		recursive = val
	}

	maxLinkDepth := 1
	if val, ok := config["max_link_depth"].(float64); ok {
		maxLinkDepth = int(val)
	} else if val, ok := config["max_link_depth"].(int); ok {
		maxLinkDepth = val
	}

	// Extract extension lists
	var includeExts, excludeExts []string
	if val, ok := config["include_extensions"].([]interface{}); ok {
		for _, ext := range val {
			if s, ok := ext.(string); ok {
				includeExts = append(includeExts, s)
			}
		}
	}
	if val, ok := config["exclude_extensions"].([]interface{}); ok {
		for _, ext := range val {
			if s, ok := ext.(string); ok {
				excludeExts = append(excludeExts, s)
			}
		}
	}

	// Initialize link extraction patterns
	linkPatterns := map[string][]*regexp.Regexp{
		"markdown": {
			regexp.MustCompile(`\[\[(.*?)\]\]`),                 // Wiki-style links [[Page]]
			regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`),       // Markdown links [text](url)
		},
		"html": {
			regexp.MustCompile(`<a\s+href=["'](.*?)["'][^>]*>(.*?)</a>`), // HTML links
		},
		"text": {
			regexp.MustCompile(`https?://[^\s<>)"]+`), // Plain URLs
			regexp.MustCompile(`file://[^\s<>)"]+`),   // file:// URLs
		},
		"xml": {
			regexp.MustCompile(`<link\s+.*?href=["'](.*?)["'].*?>`), // XML link elements
			regexp.MustCompile(`xlink:href=["'](.*?)["']`),          // XLink references
		},
	}

	return &FileContentSource{
		name:              name,
		basePath:          absPath,
		filePattern:       filePattern,
		includeExtensions: includeExts,
		excludeExtensions: excludeExts,
		watchForChanges:   watchForChanges,
		recursive:         recursive,
		maxLinkDepth:      maxLinkDepth,
		linkPatterns:      linkPatterns,
	}, nil
}

// FetchDocument fetches a document from the file system
func (f *FileContentSource) FetchDocument(sourceID string) (*DocumentContent, error) {
	// Determine if sourceID is absolute or relative to base_path
	var filePath string
	if filepath.IsAbs(sourceID) {
		filePath = sourceID
	} else {
		filePath = filepath.Join(f.basePath, sourceID)
	}

	// Check if file exists
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("file not found: %w", err)
	}

	// Get file details
	fileName := filepath.Base(filePath)
	fileExt := strings.TrimPrefix(filepath.Ext(fileName), ".")

	// Determine document type and read mode
	docType, readMode := getDocTypeAndMode(fileExt)

	// Read file content
	var content string
	var binaryPath string

	if readMode == "text" {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("error reading file: %w", err)
		}
		content = string(data)
	} else {
		// Binary mode - set binary_path instead of reading content
		binaryPath = filePath
	}

	// Get relative path
	relPath, err := filepath.Rel(f.basePath, filePath)
	if err != nil {
		relPath = filePath
	}

	// Determine content type
	contentType := mime.TypeByExtension("." + fileExt)
	if contentType == "" {
		contentType = fmt.Sprintf("application/%s", fileExt)
	}

	// Create metadata
	metadata := map[string]interface{}{
		"filename":      fileName,
		"last_modified": float64(info.ModTime().Unix()),
		"size":          info.Size(),
		"full_path":     filePath,
		"relative_path": relPath,
		"extension":     fileExt,
		"content_type":  contentType,
	}

	// Generate content hash
	var contentHash string
	if readMode == "text" {
		contentHash = GetContentHash(content)
	} else {
		// For binary files, use file path and mod time as hash
		contentHash = GetContentHash(fmt.Sprintf("%s:%d", filePath, info.ModTime().Unix()))
	}

	return &DocumentContent{
		ID:          filePath,
		Content:     content,
		BinaryPath:  binaryPath,
		Metadata:    metadata,
		ContentHash: contentHash,
		DocType:     docType,
	}, nil
}

// ListDocuments lists available documents in the file system
func (f *FileContentSource) ListDocuments() ([]DocumentInfo, error) {
	var results []DocumentInfo

	// Build glob pattern
	pattern := filepath.Join(f.basePath, f.filePattern)

	// Find files using doublestar for ** support
	files, err := doublestar.FilepathGlob(pattern)
	if err != nil {
		return nil, fmt.Errorf("glob pattern error: %w", err)
	}

	for _, filePath := range files {
		// Skip directories
		info, err := os.Stat(filePath)
		if err != nil || info.IsDir() {
			continue
		}

		fileName := filepath.Base(filePath)
		fileExt := strings.TrimPrefix(filepath.Ext(fileName), ".")

		// Apply extension filters
		if len(f.excludeExtensions) > 0 && contains(f.excludeExtensions, fileExt) {
			continue
		}
		if len(f.includeExtensions) > 0 && !contains(f.includeExtensions, fileExt) {
			continue
		}

		// Get relative path
		relPath, err := filepath.Rel(f.basePath, filePath)
		if err != nil {
			relPath = filePath
		}

		// Determine doc type
		docType, _ := getDocTypeAndMode(fileExt)

		// Determine content type
		contentType := mime.TypeByExtension("." + fileExt)
		if contentType == "" {
			contentType = fmt.Sprintf("application/%s", fileExt)
		}

		metadata := map[string]interface{}{
			"filename":      fileName,
			"last_modified": float64(info.ModTime().Unix()),
			"size":          info.Size(),
			"full_path":     filePath,
			"relative_path": relPath,
			"extension":     fileExt,
			"content_type":  contentType,
		}

		results = append(results, DocumentInfo{
			ID:       filePath,
			Metadata: metadata,
			DocType:  docType,
		})
	}

	return results, nil
}

// HasChanged checks if a file has changed based on modification time
func (f *FileContentSource) HasChanged(sourceID string, lastModified interface{}) (bool, error) {
	// If watching for changes is disabled, always return true
	if !f.watchForChanges {
		return true, nil
	}

	// Check if file exists
	info, err := os.Stat(sourceID)
	if err != nil {
		return false, fmt.Errorf("file not found: %w", err)
	}

	// If no previous modification time, consider it changed
	if lastModified == nil {
		return true, nil
	}

	// Parse lastModified to float64 (Unix timestamp)
	var lastModTime float64
	switch v := lastModified.(type) {
	case float64:
		lastModTime = v
	case int64:
		lastModTime = float64(v)
	case int:
		lastModTime = float64(v)
	default:
		return true, nil // Can't determine, assume changed
	}

	currentModTime := float64(info.ModTime().Unix())
	return currentModTime > lastModTime, nil
}

// getDocTypeAndMode determines document type and read mode based on file extension
func getDocTypeAndMode(extension string) (string, string) {
	extensionMap := map[string]struct {
		docType  string
		readMode string
	}{
		// Text-based formats
		"md":       {"markdown", "text"},
		"markdown": {"markdown", "text"},
		"txt":      {"text", "text"},
		"html":     {"html", "text"},
		"htm":      {"html", "text"},
		"css":      {"text", "text"},
		"js":       {"text", "text"},
		"json":     {"json", "text"},
		"xml":      {"xml", "text"},
		"yaml":     {"text", "text"},
		"yml":      {"text", "text"},
		"csv":      {"csv", "text"},

		// Binary formats
		"pdf":     {"pdf", "binary"},
		"docx":    {"docx", "binary"},
		"doc":     {"doc", "binary"},
		"pptx":    {"pptx", "binary"},
		"ppt":     {"ppt", "binary"},
		"xlsx":    {"xlsx", "binary"},
		"xls":     {"xls", "binary"},
		"parquet": {"parquet", "binary"},
		"png":     {"image", "binary"},
		"jpg":     {"image", "binary"},
		"jpeg":    {"image", "binary"},
		"gif":     {"image", "binary"},
		"zip":     {"binary", "binary"},
	}

	if mapping, ok := extensionMap[strings.ToLower(extension)]; ok {
		return mapping.docType, mapping.readMode
	}
	return "text", "text" // Default to text
}

// contains checks if a string slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
