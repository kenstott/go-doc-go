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

// ============================================================================
// UDML-O: Ontology Instance Layer - Entity instances and relationships extracted via ontology schemas
// ============================================================================

// OntologyEntity represents an extracted entity from ontology analysis
type OntologyEntity struct {
	EntityID    string                 `json:"entity_id"`    // Unique entity ID
	DocID       string                 `json:"doc_id"`       // Source document
	SourceName  string                 `json:"source_name"`  // Source name
	EntityName  string                 `json:"entity_name"`  // Entity name
	EntityType  string                 `json:"entity_type"`  // person/organization/location/etc
	Domain      string                 `json:"domain"`       // Domain ownership (data mesh)
	Confidence  float64                `json:"confidence"`   // Extraction confidence (0.0-1.0)
	Attributes  map[string]interface{} `json:"attributes"`   // Entity properties
	ElementID   string                 `json:"element_id"`   // Source UDML element
	ExtractedAt time.Time              `json:"extracted_at"` // Extraction timestamp
}

// OntologyRelationship represents a semantic relationship between entities
type OntologyRelationship struct {
	RelationshipID   string                 `json:"relationship_id"`   // Unique relationship ID
	DocID            string                 `json:"doc_id"`            // Source document
	SourceName       string                 `json:"source_name"`       // Source name
	SourceEntityID   string                 `json:"source_entity_id"`  // Source entity ID
	TargetEntityID   string                 `json:"target_entity_id"`  // Target entity ID
	RelationshipType string                 `json:"relationship_type"` // is_a/part_of/mentions/etc
	Domain           string                 `json:"domain"`            // Consumer domain (owns the enrichment)
	Confidence       float64                `json:"confidence"`        // Pattern reliability (0.0-1.0)
	Evidence         string                 `json:"evidence"`          // Supporting text
	Attributes       map[string]interface{} `json:"attributes"`        // Relationship properties
	ElementID        string                 `json:"element_id"`        // Source UDML element
	ExtractedAt      time.Time              `json:"extracted_at"`      // Extraction timestamp
}

// OntologyMention represents where an entity is mentioned in the document
type OntologyMention struct {
	MentionID     string    `json:"mention_id"`     // Unique mention ID
	EntityID      string    `json:"entity_id"`      // Entity being mentioned
	DocID         string    `json:"doc_id"`         // Source document
	SourceName    string    `json:"source_name"`    // Source name
	ElementID     string    `json:"element_id"`     // UDML element where mentioned
	MentionText   string    `json:"mention_text"`   // Actual text of mention
	StartPosition int       `json:"start_position"` // Character offset start
	EndPosition   int       `json:"end_position"`   // Character offset end
	ExtractedAt   time.Time `json:"extracted_at"`   // Extraction timestamp
}

// Storage defines the interface for analytics storage
type Storage interface {
	// UDML-D: Documents
	AppendDocuments(documents []Document) error

	// UDML-E: Elements
	AppendElements(elements []Element) error
	QueryElements(filters map[string]interface{}) ([]Element, error)

	// UDML-R: Structural and semantic relationships between elements
	AppendRelationships(relationships []Relationship) error

	// UDML-V: Vector embeddings
	AppendEmbeddings(embeddings []Embedding) error
	QueryEmbeddings(filters map[string]interface{}) ([]Embedding, error)

	// UDML-L: Links
	AppendLinks(links []Link) error

	// UDML-O: Ontology instances - entity instances and relationships extracted via ontology extraction rules
	AppendOntologyEntities(entities []OntologyEntity) error
	AppendOntologyRelationships(relationships []OntologyRelationship) error
	AppendOntologyMentions(mentions []OntologyMention) error
	QueryOntologyEntities(filters map[string]interface{}) ([]OntologyEntity, error)

	Close() error
}
