package contentsource

import (
	"crypto/sha256"
	"fmt"
)

// DocumentContent represents content fetched from a source
type DocumentContent struct {
	ID          string                 `json:"id"`
	Content     string                 `json:"content"`
	BinaryPath  string                 `json:"binary_path,omitempty"`
	Metadata    map[string]interface{} `json:"metadata"`
	ContentHash string                 `json:"content_hash,omitempty"`
	DocType     string                 `json:"doc_type,omitempty"`
}

// DocumentInfo represents metadata about a document
type DocumentInfo struct {
	ID       string                 `json:"id"`
	Metadata map[string]interface{} `json:"metadata"`
	DocType  string                 `json:"doc_type,omitempty"`
}

// ContentSource defines the interface for content sources
type ContentSource interface {
	FetchDocument(sourceID string) (*DocumentContent, error)
	ListDocuments() ([]DocumentInfo, error)
	HasChanged(sourceID string, lastModified interface{}) (bool, error)
}

// GetContentHash generates a SHA256 hash of the content for change detection
func GetContentHash(content string) string {
	hash := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", hash)
}
