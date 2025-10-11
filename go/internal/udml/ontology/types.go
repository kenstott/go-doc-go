package ontology

import (
	"encoding/json"
	"time"
)

// EntityType represents the type of an extracted entity
type EntityType string

const (
	EntityTypePerson       EntityType = "person"
	EntityTypeOrganization EntityType = "organization"
	EntityTypeLocation     EntityType = "location"
	EntityTypeDate         EntityType = "date"
	EntityTypeEvent        EntityType = "event"
	EntityTypeConcept      EntityType = "concept"
	EntityTypeProduct      EntityType = "product"
	EntityTypeTechnology   EntityType = "technology"
	EntityTypeCustom       EntityType = "custom"
)

// RelationshipType represents the type of relationship between entities
type RelationshipType string

const (
	RelationshipIsA          RelationshipType = "is_a"           // Inheritance/taxonomy
	RelationshipPartOf       RelationshipType = "part_of"        // Composition
	RelationshipRelatedTo    RelationshipType = "related_to"     // General association
	RelationshipLocatedIn    RelationshipType = "located_in"     // Spatial
	RelationshipOccurredAt   RelationshipType = "occurred_at"    // Temporal
	RelationshipCreatedBy    RelationshipType = "created_by"     // Authorship
	RelationshipMentions     RelationshipType = "mentions"       // Reference
	RelationshipDependsOn    RelationshipType = "depends_on"     // Dependency
	RelationshipImplements   RelationshipType = "implements"     // Technical
	RelationshipExtends      RelationshipType = "extends"        // Technical
	RelationshipContains     RelationshipType = "contains"       // Containment
	RelationshipReferencedBy RelationshipType = "referenced_by" // Back-reference
	RelationshipCustom       RelationshipType = "custom"         // User-defined
)

// ============================================================================
// ONTOLOGY SCHEMA (EXTRACTION RULES) - Input to extraction
// ============================================================================

// OntologySchema defines the extraction rules for a domain
type OntologySchema struct {
	Name                    string                    `json:"name"`                      // Schema name
	Version                 string                    `json:"version"`                   // Schema version
	Description             string                    `json:"description"`               // Schema description
	Domain                  string                    `json:"domain"`                    // Domain (e.g., "financial", "legal")
	DocumentTypes           []string                  `json:"document_types"`            // Applicable document types
	KeyConcepts             []string                  `json:"key_concepts"`              // Key concepts in domain
	Terms                   []Term                    `json:"terms,omitempty"`           // Domain terms and synonyms
	ElementEntityMappings   []ElementEntityMapping    `json:"element_entity_mappings"`   // Entity extraction rules
	EntityRelationshipRules []EntityRelationshipRule  `json:"entity_relationship_rules"` // Relationship extraction rules
	DerivedEntities         []DerivedEntity           `json:"derived_entities,omitempty"` // Derived entity definitions
	Metadata                map[string]interface{}    `json:"metadata,omitempty"`        // Additional metadata
	CreatedAt               time.Time                 `json:"created_at"`                // Creation timestamp
}

// Term defines a domain term and its synonyms
type Term struct {
	Term        string   `json:"term"`                  // Term name
	Synonyms    []string `json:"synonyms"`              // Synonyms
	Description string   `json:"description,omitempty"` // Term description
}

// ElementEntityMapping defines how to extract an entity type from element types
type ElementEntityMapping struct {
	EntityType      string           `json:"entity_type"`   // Entity type to extract
	Description     string           `json:"description"`   // Description of entity type
	ElementTypes    []string         `json:"element_types"` // UDML element types to process
	ExtractionRules []ExtractionRule `json:"extraction_rules"` // Rules for extraction
}

// ExtractionRuleType defines the type of extraction rule
type ExtractionRuleType string

const (
	RuleTypeMetadata   ExtractionRuleType = "metadata_field"   // Extract from element metadata
	RuleTypeRegex      ExtractionRuleType = "regex_pattern"    // Extract using regex pattern
	RuleTypeKeyword    ExtractionRuleType = "keyword_match"    // Extract by keyword matching
	RuleTypeSimilarity ExtractionRuleType = "text_similarity"  // Extract based on text similarity
	RuleTypeJSONPath   ExtractionRuleType = "jsonpath_query"   // Extract using JSONPath expressions
)

// ExtractionRule defines a rule for extracting entities
type ExtractionRule struct {
	Type                ExtractionRuleType `json:"type"`                            // Rule type
	FieldPath           string             `json:"field_path,omitempty"`            // Metadata field path (for metadata_field)
	Pattern             string             `json:"pattern,omitempty"`               // Regex pattern (for regex_pattern)
	Keywords            []string           `json:"keywords,omitempty"`              // Keywords to match (for keyword_match)
	ReferenceText       string             `json:"reference_text,omitempty"`        // Reference text for similarity (for text_similarity)
	SimilarityThreshold float64            `json:"similarity_threshold,omitempty"`  // Minimum similarity score (for text_similarity)
	JSONPathExpr        string             `json:"jsonpath_expr,omitempty"`         // JSONPath expression (for jsonpath_query)
	Confidence          float64            `json:"confidence"`                      // Confidence score for this rule
}

// EntityRelationshipRule defines how to extract relationships between entity types
type EntityRelationshipRule struct {
	Name                string           `json:"name"`                        // Rule name
	SourceEntityType    string           `json:"source_entity_type"`          // Source entity type
	TargetEntityType    string           `json:"target_entity_type"`          // Target entity type
	RelationshipType    RelationshipType `json:"relationship_type"`           // Relationship type
	Description         string           `json:"description,omitempty"`       // Rule description
	ConfidenceThreshold float64          `json:"confidence_threshold"`        // Minimum confidence to extract
}

// DerivedEntity defines an entity derived from combinations of other entities
type DerivedEntity struct {
	Name            string   `json:"name"`             // Derived entity name
	Description     string   `json:"description"`      // Description
	SourceEntities  []string `json:"source_entities"`  // Source entity types
	AggregationType string   `json:"aggregation_type"` // How to combine (e.g., "COMBINATION")
}

// ============================================================================
// EXTRACTED INSTANCES - Output from extraction
// ============================================================================

// Entity represents a named entity extracted from a document
type Entity struct {
	ID         string                 `json:"id"`          // Unique identifier
	Name       string                 `json:"name"`        // Entity name
	Type       EntityType             `json:"type"`        // Entity type
	Confidence float64                `json:"confidence"`  // Extraction confidence (0-1)
	Attributes map[string]interface{} `json:"attributes"`  // Additional properties
	ElementID  string                 `json:"element_id"`  // Source UDML element
	Mentions   []Mention              `json:"mentions"`    // Where entity appears
	CreatedAt  time.Time              `json:"created_at"`  // Extraction timestamp
	UpdatedAt  time.Time              `json:"updated_at"`  // Last update timestamp
}

// Mention represents a specific occurrence of an entity in text
type Mention struct {
	ElementID string `json:"element_id"` // UDML element where mentioned
	Text      string `json:"text"`       // Actual text mention
	StartPos  int    `json:"start_pos"`  // Character offset start
	EndPos    int    `json:"end_pos"`    // Character offset end
}

// Relationship represents a relationship between two entities
type Relationship struct {
	ID         string                 `json:"id"`          // Unique identifier
	Type       RelationshipType       `json:"type"`        // Relationship type
	SourceID   string                 `json:"source_id"`   // Source entity ID
	TargetID   string                 `json:"target_id"`   // Target entity ID
	Confidence float64                `json:"confidence"`  // Extraction confidence (0-1)
	Attributes map[string]interface{} `json:"attributes"`  // Additional properties
	ElementID  string                 `json:"element_id"`  // Source UDML element
	Evidence   string                 `json:"evidence"`    // Supporting text
	CreatedAt  time.Time              `json:"created_at"`  // Extraction timestamp
}

// Class represents an ontology class (concept definition)
type Class struct {
	ID          string                 `json:"id"`          // Unique identifier
	Name        string                 `json:"name"`        // Class name
	Description string                 `json:"description"` // Class description
	ParentID    string                 `json:"parent_id"`   // Parent class (for hierarchy)
	Properties  []Property             `json:"properties"`  // Class properties
	Attributes  map[string]interface{} `json:"attributes"`  // Additional metadata
	CreatedAt   time.Time              `json:"created_at"`  // Creation timestamp
}

// Property represents a property of an ontology class
type Property struct {
	Name        string      `json:"name"`        // Property name
	Type        string      `json:"type"`        // Data type
	Description string      `json:"description"` // Property description
	Required    bool        `json:"required"`    // Whether required
	DefaultVal  interface{} `json:"default"`     // Default value
}

// Ontology represents a complete ontology extracted from a document
type Ontology struct {
	ID            string         `json:"id"`             // Unique identifier
	DocID         string         `json:"doc_id"`         // Source document ID
	Name          string         `json:"name"`           // Ontology name
	Description   string         `json:"description"`    // Ontology description
	Version       string         `json:"version"`        // Ontology version
	Entities      []Entity       `json:"entities"`       // Extracted entities
	Relationships []Relationship `json:"relationships"`  // Extracted relationships
	Classes       []Class        `json:"classes"`        // Ontology classes
	Metadata      OntologyMeta   `json:"metadata"`       // Extraction metadata
	CreatedAt     time.Time      `json:"created_at"`     // Creation timestamp
	UpdatedAt     time.Time      `json:"updated_at"`     // Last update timestamp
}

// OntologyMeta contains metadata about the ontology extraction process
type OntologyMeta struct {
	ExtractorType    string                 `json:"extractor_type"`    // LLM provider used
	ExtractorVersion string                 `json:"extractor_version"` // Extractor version
	ExtractionTime   time.Duration          `json:"extraction_time"`   // Time taken
	ElementCount     int                    `json:"element_count"`     // Source elements processed
	Confidence       float64                `json:"confidence"`        // Overall confidence
	ModelName        string                 `json:"model_name"`        // LLM model name
	PromptTokens     int                    `json:"prompt_tokens"`     // Token usage
	CompletionTokens int                    `json:"completion_tokens"` // Token usage
	CustomFields     map[string]interface{} `json:"custom_fields"`     // Extensibility
}

// ToJSON serializes the ontology to pretty-printed JSON
func (o *Ontology) ToJSON() ([]byte, error) {
	return json.MarshalIndent(o, "", "  ")
}

// ToJSONCompact serializes the ontology to compact JSON
func (o *Ontology) ToJSONCompact() ([]byte, error) {
	return json.Marshal(o)
}

// GetEntityByID retrieves an entity by its ID
func (o *Ontology) GetEntityByID(id string) *Entity {
	for i := range o.Entities {
		if o.Entities[i].ID == id {
			return &o.Entities[i]
		}
	}
	return nil
}

// GetEntityByName retrieves an entity by its name
func (o *Ontology) GetEntityByName(name string) *Entity {
	for i := range o.Entities {
		if o.Entities[i].Name == name {
			return &o.Entities[i]
		}
	}
	return nil
}

// GetEntitiesByType retrieves all entities of a specific type
func (o *Ontology) GetEntitiesByType(entityType EntityType) []Entity {
	var result []Entity
	for _, entity := range o.Entities {
		if entity.Type == entityType {
			result = append(result, entity)
		}
	}
	return result
}

// GetRelationshipsBySource retrieves all relationships from a source entity
func (o *Ontology) GetRelationshipsBySource(sourceID string) []Relationship {
	var result []Relationship
	for _, rel := range o.Relationships {
		if rel.SourceID == sourceID {
			result = append(result, rel)
		}
	}
	return result
}

// GetRelationshipsByTarget retrieves all relationships to a target entity
func (o *Ontology) GetRelationshipsByTarget(targetID string) []Relationship {
	var result []Relationship
	for _, rel := range o.Relationships {
		if rel.TargetID == targetID {
			result = append(result, rel)
		}
	}
	return result
}

// GetRelationshipsByType retrieves all relationships of a specific type
func (o *Ontology) GetRelationshipsByType(relType RelationshipType) []Relationship {
	var result []Relationship
	for _, rel := range o.Relationships {
		if rel.Type == relType {
			result = append(result, rel)
		}
	}
	return result
}

// GetClassByID retrieves a class by its ID
func (o *Ontology) GetClassByID(id string) *Class {
	for i := range o.Classes {
		if o.Classes[i].ID == id {
			return &o.Classes[i]
		}
	}
	return nil
}

// GetStats returns statistics about the ontology
func (o *Ontology) GetStats() OntologyStats {
	stats := OntologyStats{
		EntityCount:       len(o.Entities),
		RelationshipCount: len(o.Relationships),
		ClassCount:        len(o.Classes),
		EntityTypes:       make(map[EntityType]int),
		RelationshipTypes: make(map[RelationshipType]int),
	}

	for _, entity := range o.Entities {
		stats.EntityTypes[entity.Type]++
	}

	for _, rel := range o.Relationships {
		stats.RelationshipTypes[rel.Type]++
	}

	return stats
}

// OntologyStats contains statistics about an ontology
type OntologyStats struct {
	EntityCount       int                          `json:"entity_count"`
	RelationshipCount int                          `json:"relationship_count"`
	ClassCount        int                          `json:"class_count"`
	EntityTypes       map[EntityType]int           `json:"entity_types"`
	RelationshipTypes map[RelationshipType]int     `json:"relationship_types"`
}

// Validate checks if the ontology is valid
func (o *Ontology) Validate() error {
	// Check entity IDs are unique
	entityIDs := make(map[string]bool)
	for _, entity := range o.Entities {
		if entity.ID == "" {
			return NewValidationError("entity has empty ID")
		}
		if entityIDs[entity.ID] {
			return NewValidationError("duplicate entity ID: " + entity.ID)
		}
		entityIDs[entity.ID] = true
	}

	// Check relationship references valid entities
	for _, rel := range o.Relationships {
		if rel.SourceID == "" || rel.TargetID == "" {
			return NewValidationError("relationship has empty source or target ID")
		}
		if !entityIDs[rel.SourceID] {
			return NewValidationError("relationship references unknown source entity: " + rel.SourceID)
		}
		if !entityIDs[rel.TargetID] {
			return NewValidationError("relationship references unknown target entity: " + rel.TargetID)
		}
	}

	// Check class IDs are unique
	classIDs := make(map[string]bool)
	for _, class := range o.Classes {
		if class.ID == "" {
			return NewValidationError("class has empty ID")
		}
		if classIDs[class.ID] {
			return NewValidationError("duplicate class ID: " + class.ID)
		}
		classIDs[class.ID] = true
	}

	return nil
}

// ValidationError represents an ontology validation error
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return "ontology validation error: " + e.Message
}

// NewValidationError creates a new validation error
func NewValidationError(message string) error {
	return &ValidationError{Message: message}
}
