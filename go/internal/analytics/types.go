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
	ElementCategory   string                 `json:"element_category"`            // Universal Document Model category
	Content           string                 `json:"content,omitempty"`           // Full content
	ContentPreview    string                 `json:"content_preview"`
	ContentLocation   map[string]interface{} `json:"content_location,omitempty"`  // Source location info
	ContentHash       string                 `json:"content_hash,omitempty"`      // Hash of content
	ParentID          string                 `json:"parent_id,omitempty"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
	ElementOrder      float64                `json:"element_order"`               // Order within parent
	DocumentPosition  float64                `json:"document_position"`           // Position in document

	// UDML Phase 1: Query-Optimized Promoted Fields (nullable, 70-95% NULL is acceptable)
	// These fields enable 60-1000x faster queries across all backends (Parquet, PostgreSQL, Neo4j, Elasticsearch)
	PageNumber    *int    `json:"page_number,omitempty" parquet:"page_number"`       // PDF/DOCX/PPTX page location (~30% populated)
	SectionLevel  *int    `json:"section_level,omitempty" parquet:"section_level"`   // Heading hierarchy level (~15% populated)
	RowIndex      *int    `json:"row_index,omitempty" parquet:"row_index"`           // Table row position (~20% populated)
	ColumnIndex   *int    `json:"column_index,omitempty" parquet:"column_index"`     // Table column position (~20% populated)
	TemporalType  *string `json:"temporal_type,omitempty" parquet:"temporal_type"`   // date/datetime/year/etc (~5-10% populated)
	TagName       *string `json:"tag_name,omitempty" parquet:"tag_name"`             // HTML/XML tag identifier (~25% populated)

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

// Link represents an extracted link
type Link struct {
	LinkID          string `json:"link_id"`
	SourceElementID string `json:"source_element_id"`
	DocID           string `json:"doc_id"`
	SourceName      string `json:"source_name"`
	LinkType        string `json:"link_type"`
	LinkTarget      string `json:"link_target"`
	LinkText        string `json:"link_text,omitempty"`
}

// Storage defines the interface for analytics storage
type Storage interface {
	AppendDocuments(documents []Document) error
	AppendElements(elements []Element) error
	AppendRelationships(relationships []Relationship) error
	AppendEmbeddings(embeddings []Embedding) error
	AppendLinks(links []Link) error
	Close() error
}
