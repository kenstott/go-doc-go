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

// ============================================================================
// Corpus Exploration Types - for MCP server and interactive exploration
// ============================================================================

// SearchResult represents a search result with optional similarity/match score
type SearchResult struct {
	Element    Element `json:"element"`
	Score      float64 `json:"score,omitempty"`       // Similarity or relevance score
	MatchCount int     `json:"match_count,omitempty"` // Number of matches (for keyword search)
}

// PatternStats represents statistics about regex pattern matches
type PatternStats struct {
	Pattern            string         `json:"pattern"`
	TotalMatches       int            `json:"total_matches"`
	DocumentCount      int            `json:"document_count"`
	ElementTypeDistrib map[string]int `json:"element_type_distribution"`
	Examples           []Element      `json:"examples"` // Sample matching elements
}

// TermFrequency represents frequency statistics for a term
type TermFrequency struct {
	Term               string         `json:"term"`
	Frequency          int            `json:"frequency"`           // Total occurrences
	DocumentCount      int            `json:"document_count"`      // Unique documents
	ElementTypeDistrib map[string]int `json:"element_type_distribution"`
}

// CooccurrenceResult represents co-occurrence analysis between two terms
type CooccurrenceResult struct {
	Entity1       string    `json:"entity1"`
	Entity2       string    `json:"entity2"`
	CooccurCount  int       `json:"cooccurrence_count"`
	ContextWindow string    `json:"context_window"` // "element", "paragraph", "document"
	Examples      []Element `json:"examples"`       // Sample co-occurrences
}

// ElementContext represents an element with its hierarchical context
type ElementContext struct {
	Element  Element   `json:"element"`
	Parents  []Element `json:"parents,omitempty"`
	Siblings []Element `json:"siblings,omitempty"`
	Children []Element `json:"children,omitempty"`
}

// CorpusStats represents aggregate statistics about the corpus
type CorpusStats struct {
	ElementTypeDistribution map[string]int `json:"element_type_distribution,omitempty"`
	DocumentCount           int            `json:"document_count,omitempty"`
	TotalElements           int            `json:"total_elements,omitempty"`
	AvgContentLength        float64        `json:"avg_content_length,omitempty"`
}

// Storage defines the interface for analytics storage
type Storage interface {
	// UDML-D: Documents
	AppendDocuments(documents []Document) error

	// UDML-E: Elements
	AppendElements(elements []Element) error
	// QueryElements retrieves elements based on filters.
	// Standard filters:
	//   - "source_name" (string): Filter by source name
	//   - "doc_id" (string): Filter by document ID
	//   - "element_type" (string): Filter by element type
	//   - "element_category" (string): Filter by element category
	// Temporal filters (for versioned/partitioned storage):
	//   - "latest_only" (bool): If true, deduplicate by doc_id to return only latest version (default: false)
	//   - "as_of_date" (string): Return corpus state as of this date, format "YYYY-MM-DD" (optional)
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
	QueryOntologyRelationships(filters map[string]interface{}) ([]OntologyRelationship, error)

	// Content resolution for samplers/query engines
	GetContentResolver() interface{}

	// ========================================================================
	// Corpus Exploration Methods - for MCP server and interactive tools
	// All methods support temporal filtering through the filters parameter
	// ========================================================================

	// SearchSemanticSimilarity performs semantic similarity search using vector embeddings
	// filters: standard filters (source_name, doc_id, element_type, latest_only, as_of_date)
	SearchSemanticSimilarity(queryVector []float64, filters map[string]interface{}, threshold float64, limit int) ([]SearchResult, error)

	// SearchByRegex performs regex pattern matching on element content
	// filters: standard filters (source_name, doc_id, element_type, latest_only, as_of_date)
	SearchByRegex(pattern string, filters map[string]interface{}, limit int) ([]SearchResult, error)

	// SearchByKeyword performs keyword search on element content (case-insensitive substring match)
	// filters: standard filters (source_name, doc_id, element_type, latest_only, as_of_date)
	SearchByKeyword(keyword string, filters map[string]interface{}, limit int) ([]SearchResult, error)

	// AnalyzePattern analyzes a regex pattern across the corpus and returns statistics
	// filters: standard filters (source_name, doc_id, element_type, latest_only, as_of_date)
	AnalyzePattern(pattern string, filters map[string]interface{}, maxExamples int) (*PatternStats, error)

	// ComputeTermFrequencies computes frequency statistics for given terms
	// filters: standard filters (source_name, doc_id, element_type, latest_only, as_of_date)
	ComputeTermFrequencies(terms []string, caseSensitive bool, filters map[string]interface{}) ([]TermFrequency, error)

	// FindCooccurrences finds co-occurrences of two entities within a context window
	// contextWindow: "element", "paragraph", "document"
	// filters: standard filters (source_name, doc_id, element_type, latest_only, as_of_date)
	FindCooccurrences(entity1, entity2 string, contextWindow string, filters map[string]interface{}, maxExamples int) (*CooccurrenceResult, error)

	// GetElementContext retrieves an element with its hierarchical context (parents, siblings, children)
	// filters: standard filters (source_name, doc_id, element_type, latest_only, as_of_date)
	GetElementContext(elementID string, filters map[string]interface{}, contextDepth int, includeSiblings, includeChildren bool) (*ElementContext, error)

	// AggregateStatistics computes aggregate statistics about the corpus
	// metrics: list of metrics to compute (e.g., "element_type_distribution", "document_count", "total_elements", "avg_content_length")
	// filters: standard filters (source_name, doc_id, element_type, latest_only, as_of_date)
	AggregateStatistics(metrics []string, filters map[string]interface{}) (*CorpusStats, error)

	Close() error
}
