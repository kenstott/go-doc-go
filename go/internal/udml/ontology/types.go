package ontology

import (
	"encoding/json"
	"fmt"
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

// Domain represents a domain of ownership in the data mesh
type Domain struct {
	Name        string `json:"name" yaml:"name"`                                   // Domain name
	Description string `json:"description,omitempty" yaml:"description,omitempty"` // Domain description
	Owner       string `json:"owner,omitempty" yaml:"owner,omitempty"`             // Domain owner (team/person)
}

// OntologySchema defines the extraction rules for a domain
type OntologySchema struct {
	Name                    string                    `json:"name" yaml:"name"`                                                     // Schema name
	Version                 string                    `json:"version" yaml:"version"`                                               // Schema version
	Description             string                    `json:"description" yaml:"description"`                                       // Schema description
	Domain                  string                    `json:"domain" yaml:"domain"`                                                 // Primary domain (deprecated, use Domains)
	Domains                 []Domain                  `json:"domains" yaml:"domains"`                                               // Domain registry (multi-domain support)
	DocumentTypes           []string                  `json:"document_types,omitempty" yaml:"document_types,omitempty"`             // Applicable document types
	KeyConcepts             []string                  `json:"key_concepts,omitempty" yaml:"key_concepts,omitempty"`                 // Key concepts in domain
	Terms                   []Term                    `json:"terms,omitempty" yaml:"terms,omitempty"`                               // Domain terms and synonyms
	ElementEntityMappings   []ElementEntityMapping    `json:"element_entity_mappings" yaml:"element_entity_mappings"`               // Entity extraction rules
	EntityRelationshipRules []EntityRelationshipRule  `json:"entity_relationship_rules,omitempty" yaml:"entity_relationship_rules,omitempty"` // Relationship extraction rules
	DerivedEntities         []DerivedEntity           `json:"derived_entities,omitempty" yaml:"derived_entities,omitempty"`         // Derived entity definitions
	Metadata                map[string]interface{}    `json:"metadata,omitempty" yaml:"metadata,omitempty"`                         // Additional metadata
	CreatedAt               time.Time                 `json:"created_at" yaml:"created_at"`                                         // Creation timestamp
}

// Term defines a domain term and its synonyms
type Term struct {
	Term        string   `json:"term"`                  // Term name
	Synonyms    []string `json:"synonyms"`              // Synonyms
	Description string   `json:"description,omitempty"` // Term description
}

// ElementEntityMapping defines how to extract an entity type from element types
type ElementEntityMapping struct {
	EntityType      string           `json:"entity_type" yaml:"entity_type"`             // Entity type to extract
	Domain          string           `json:"domain" yaml:"domain"`                       // Domain this entity belongs to (required)
	Description     string           `json:"description" yaml:"description"`             // Description of entity type
	ElementTypes    []string         `json:"element_types,omitempty" yaml:"element_types,omitempty"` // Simple element type filter
	ElementFilter   string           `json:"element_filter,omitempty" yaml:"element_filter,omitempty"` // JSONPath filter (advanced)
	Confidence      float64          `json:"confidence" yaml:"confidence"`               // Context quality confidence (0.0-1.0)
	ExtractionRules []ExtractionRule `json:"extraction_rules" yaml:"extraction_rules"`   // Rules for extraction (OR logic)
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

// ExtractionRule defines a rule for extracting entities (binary match)
type ExtractionRule struct {
	Type                ExtractionRuleType `json:"type" yaml:"type"`                                              // Rule type
	FieldPath           string             `json:"field_path,omitempty" yaml:"field_path,omitempty"`              // Metadata field path (for metadata_field)
	Pattern             string             `json:"pattern,omitempty" yaml:"pattern,omitempty"`                    // Regex pattern (for regex_pattern)
	Keywords            []string           `json:"keywords,omitempty" yaml:"keywords,omitempty"`                  // Keywords to match (for keyword_match)
	ReferenceText       string             `json:"reference_text,omitempty" yaml:"reference_text,omitempty"`      // Reference text for similarity (for text_similarity)
	SimilarityThreshold float64            `json:"similarity_threshold,omitempty" yaml:"similarity_threshold,omitempty"` // Minimum similarity score (for text_similarity)
	JSONPathExpr        string             `json:"jsonpath_expr,omitempty" yaml:"jsonpath_expr,omitempty"`        // JSONPath expression (for jsonpath_query)
}

// RelationshipExtractionPatternType defines types of relationship extraction patterns
type RelationshipExtractionPatternType string

const (
	RelPatternTextTemplate RelationshipExtractionPatternType = "text_template" // Text pattern with entity placeholders
	RelPatternProximity    RelationshipExtractionPatternType = "proximity"     // Entities near each other with signal words
	RelPatternRegex        RelationshipExtractionPatternType = "regex"         // Regex with named groups for entities
	RelPatternDependency   RelationshipExtractionPatternType = "dependency"    // Grammatical dependency pattern
	RelPatternCooccurrence RelationshipExtractionPatternType = "cooccurrence"  // Statistical co-occurrence
)

// RelationshipExtractionPattern defines a pattern for extracting a relationship (binary match)
type RelationshipExtractionPattern struct {
	Type        RelationshipExtractionPatternType `json:"type" yaml:"type"`                               // Pattern type
	Template    string                            `json:"template,omitempty" yaml:"template,omitempty"`   // Text template (e.g., "{person} is CEO of {organization}")
	Pattern     string                            `json:"pattern,omitempty" yaml:"pattern,omitempty"`     // Regex pattern with named groups
	SignalWords []string                          `json:"signal_words,omitempty" yaml:"signal_words,omitempty"` // Signal words indicating relationship
	MaxDistance int                               `json:"max_distance,omitempty" yaml:"max_distance,omitempty"` // Max tokens between entities (for proximity)
	Direction   string                            `json:"direction,omitempty" yaml:"direction,omitempty"` // "forward", "backward", "bidirectional"
	Examples    []string                          `json:"examples,omitempty" yaml:"examples,omitempty"`   // Example texts matching this pattern
}

// EntityRelationshipRule defines how to extract relationships between entity types
//
// Relationship semantic: source_entity --[ENRICHED_BY]--> target_entity
//   - SourceEntityType: Entity being enriched (consumer's entity)
//   - TargetEntityType: Entity providing enrichment (producer's entity)
//   - Domain ownership: Inherited from source entity (consumer domain)
//
// The domain that NEEDS the enrichment owns the relationship. Source/target are
// graph structure terms (for Neo4j/RDF export). In data governance contexts, this
// pattern aligns with consumer ownership of integration logic and producer ownership
// of data access approvals.
type EntityRelationshipRule struct {
	Name               string                          `json:"name" yaml:"name"`                                       // Rule name
	SourceEntityType   string                          `json:"source_entity_type" yaml:"source_entity_type"`           // Source entity type
	TargetEntityType   string                          `json:"target_entity_type" yaml:"target_entity_type"`           // Target entity type
	RelationshipType   RelationshipType                `json:"relationship_type" yaml:"relationship_type"`             // Relationship type
	Description        string                          `json:"description,omitempty" yaml:"description,omitempty"`     // Rule description
	Confidence         float64                         `json:"confidence" yaml:"confidence"`                           // Pattern reliability confidence (0.0-1.0)
	ExtractionPatterns []RelationshipExtractionPattern `json:"extraction_patterns" yaml:"extraction_patterns"`         // Patterns for extracting relationship (OR logic)
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
	Domain     string                 `json:"domain"`      // Domain ownership (inherited from mapping)
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
	Domain     string                 `json:"domain"`      // Domain ownership (consumer domain, inherited from source entity being enriched)
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

// Validate checks if the ontology schema is valid
func (s *OntologySchema) Validate() error {
	// Check required fields
	if s.Name == "" {
		return NewValidationError("schema name cannot be empty")
	}
	if s.Version == "" {
		return NewValidationError("schema version cannot be empty")
	}

	// Build domain registry map
	domainMap := make(map[string]bool)
	for _, domain := range s.Domains {
		if domain.Name == "" {
			return NewValidationError("domain has empty name")
		}
		if domainMap[domain.Name] {
			return NewValidationError("duplicate domain name: " + domain.Name)
		}
		domainMap[domain.Name] = true
	}

	// Check at least one entity mapping exists
	if len(s.ElementEntityMappings) == 0 {
		return NewValidationError("schema must have at least one entity mapping")
	}

	// Validate entity mappings
	for i, mapping := range s.ElementEntityMappings {
		if mapping.EntityType == "" {
			return NewValidationError(fmt.Sprintf("entity mapping %d has empty entity_type", i))
		}
		if mapping.Domain == "" {
			return NewValidationError(fmt.Sprintf("entity mapping %d (%s) has empty domain", i, mapping.EntityType))
		}
		// Validate domain exists in registry (if registry is populated)
		if len(s.Domains) > 0 && !domainMap[mapping.Domain] {
			return NewValidationError(fmt.Sprintf("entity mapping %d (%s) references unknown domain: %s", i, mapping.EntityType, mapping.Domain))
		}
		if mapping.Confidence < 0.0 || mapping.Confidence > 1.0 {
			return NewValidationError(fmt.Sprintf("entity mapping %d (%s) has invalid confidence: %.2f (must be 0.0-1.0)", i, mapping.EntityType, mapping.Confidence))
		}
		if len(mapping.ExtractionRules) == 0 {
			return NewValidationError(fmt.Sprintf("entity mapping %d (%s) has no extraction rules", i, mapping.EntityType))
		}

		// Validate extraction rules
		for j, rule := range mapping.ExtractionRules {
			switch rule.Type {
			case RuleTypeMetadata:
				if rule.FieldPath == "" {
					return NewValidationError(fmt.Sprintf("entity mapping %d (%s), rule %d: metadata_field requires field_path", i, mapping.EntityType, j))
				}
			case RuleTypeRegex:
				if rule.Pattern == "" {
					return NewValidationError(fmt.Sprintf("entity mapping %d (%s), rule %d: regex_pattern requires pattern", i, mapping.EntityType, j))
				}
			case RuleTypeKeyword:
				if len(rule.Keywords) == 0 {
					return NewValidationError(fmt.Sprintf("entity mapping %d (%s), rule %d: keyword_match requires keywords", i, mapping.EntityType, j))
				}
			case RuleTypeSimilarity:
				if rule.ReferenceText == "" {
					return NewValidationError(fmt.Sprintf("entity mapping %d (%s), rule %d: text_similarity requires reference_text", i, mapping.EntityType, j))
				}
				if rule.SimilarityThreshold < 0.0 || rule.SimilarityThreshold > 1.0 {
					return NewValidationError(fmt.Sprintf("entity mapping %d (%s), rule %d: invalid similarity_threshold: %.2f (must be 0.0-1.0)", i, mapping.EntityType, j, rule.SimilarityThreshold))
				}
			case RuleTypeJSONPath:
				if rule.JSONPathExpr == "" {
					return NewValidationError(fmt.Sprintf("entity mapping %d (%s), rule %d: jsonpath_query requires jsonpath_expr", i, mapping.EntityType, j))
				}
			default:
				return NewValidationError(fmt.Sprintf("entity mapping %d (%s), rule %d: unknown rule type: %s", i, mapping.EntityType, j, rule.Type))
			}
		}
	}

	// Validate relationship rules
	for i, rule := range s.EntityRelationshipRules {
		if rule.Name == "" {
			return NewValidationError(fmt.Sprintf("relationship rule %d has empty name", i))
		}
		if rule.SourceEntityType == "" {
			return NewValidationError(fmt.Sprintf("relationship rule %d (%s) has empty source_entity_type", i, rule.Name))
		}
		if rule.TargetEntityType == "" {
			return NewValidationError(fmt.Sprintf("relationship rule %d (%s) has empty target_entity_type", i, rule.Name))
		}
		if rule.Confidence < 0.0 || rule.Confidence > 1.0 {
			return NewValidationError(fmt.Sprintf("relationship rule %d (%s) has invalid confidence: %.2f (must be 0.0-1.0)", i, rule.Name, rule.Confidence))
		}
		if len(rule.ExtractionPatterns) == 0 {
			return NewValidationError(fmt.Sprintf("relationship rule %d (%s) has no extraction patterns", i, rule.Name))
		}
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
