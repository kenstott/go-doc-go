package parser

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"time"
)

// Universal Document Model Types
// These types are shared across ALL parsers to maintain consistency

// Document represents the top-level document metadata
type Document struct {
	ID       string                 `json:"id"`
	DocType  string                 `json:"doc_type"`
	Title    string                 `json:"title,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// Element represents a universal document element that works across all formats
type Element struct {
	ElementID       string                 `json:"element_id"`
	ElementType     string                 `json:"element_type"`
	Content         string                 `json:"content,omitempty"`
	ContentPreview  string                 `json:"content_preview"`
	ParentID        string                 `json:"parent_id,omitempty"`
	Position        int                    `json:"position"`
	Depth           int                    `json:"depth"`
	ContentLocation map[string]interface{} `json:"content_location,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// Relationship represents a relationship between elements
type Relationship struct {
	RelationshipID   string                 `json:"relationship_id"`
	RelationshipType string                 `json:"relationship_type"`
	SourceElementID  string                 `json:"source_element_id"`
	TargetElementID  string                 `json:"target_element_id"`
	Confidence       float64                `json:"confidence,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// Link represents an extracted link (URL, email, file reference, etc.)
type Link struct {
	LinkID          string `json:"link_id"`
	SourceElementID string `json:"source_element_id"`
	LinkType        string `json:"link_type"`
	LinkTarget      string `json:"link_target"`
	LinkText        string `json:"link_text,omitempty"`
	Context         string `json:"context,omitempty"`
}

// ParseResult is the universal output format for all parsers
type ParseResult struct {
	Document      Document       `json:"document"`
	Elements      []Element      `json:"elements"`
	Relationships []Relationship `json:"relationships"`
	Links         []Link         `json:"links,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}


// Common relationship types
const (
	RelationshipContains    = "contains"
	RelationshipContainedBy = "contained_by"
	RelationshipReferences  = "references"
	RelationshipReferencedBy = "referenced_by"
	RelationshipNext        = "next"
	RelationshipPrevious    = "previous"
	RelationshipLinksTo     = "links_to"
)

// Common link types
const (
	LinkTypeURL      = "url"
	LinkTypeEmail    = "email"
	LinkTypeFile     = "file"
	LinkTypeInternal = "internal"
	LinkTypeCitation = "citation"
	LinkTypeFootnote = "footnote"
)

// Helper functions

// generateID generates a unique ID with an optional prefix
func generateID(prefix string) string {
	b := make([]byte, 8)
	rand.Read(b)
	timestamp := time.Now().UnixNano() / 1000000 // milliseconds
	return fmt.Sprintf("%s_%d_%x", prefix, timestamp, b)
}

// ToJSON converts ParseResult to JSON string
func (r *ParseResult) ToJSON() (string, error) {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}