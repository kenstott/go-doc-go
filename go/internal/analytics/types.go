package analytics

import "time"

// Document represents a processed document for analytics
type Document struct {
	DocID             string    `json:"doc_id"`
	SourceName        string    `json:"source_name"`
	Title             string    `json:"title"`
	URL               string    `json:"url"`
	ContentType       string    `json:"content_type"`
	ProcessedAt       time.Time `json:"processed_at"`
	ElementCount      int       `json:"element_count"`
	RelationshipCount int       `json:"relationship_count"`
}

// Element represents a document element
type Element struct {
	ElementID         string                 `json:"element_id"`
	DocID             string                 `json:"doc_id"`
	SourceName        string                 `json:"source_name"`
	ElementType       string                 `json:"element_type"`
	Content           string                 `json:"content,omitempty"`           // Full content
	ContentPreview    string                 `json:"content_preview"`
	ContentLocation   map[string]interface{} `json:"content_location,omitempty"`  // Source location info
	ContentHash       string                 `json:"content_hash,omitempty"`      // Hash of content
	ParentID          string                 `json:"parent_id,omitempty"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
	ElementOrder      float64                `json:"element_order"`               // Order within parent
	DocumentPosition  float64                `json:"document_position"`           // Position in document
	TemporalMetadata  map[string]interface{} `json:"temporal_metadata,omitempty"` // Time-based metadata
}

// Relationship represents a relationship between elements
type Relationship struct {
	SourceElementID    string                 `json:"source_element_id"`
	TargetElementID    string                 `json:"target_element_id"`
	RelationshipType   string                 `json:"relationship_type"`
	DocID              string                 `json:"doc_id"`
	SourceName         string                 `json:"source_name"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
}

// Embedding represents an element embedding
type Embedding struct {
	ElementID  string    `json:"element_id"`
	DocID      string    `json:"doc_id"`
	SourceName string    `json:"source_name"`
	Embedding  []float64 `json:"embedding"`
	Text       string    `json:"text"` // Contextual text used for embedding (with parents/siblings)
}

// Storage defines the interface for analytics storage
type Storage interface {
	AppendDocuments(documents []Document) error
	AppendElements(elements []Element) error
	AppendRelationships(relationships []Relationship) error
	AppendEmbeddings(embeddings []Embedding) error
	Close() error
}
