package ontology

import (
	"encoding/json"
	"fmt"
	"strings"
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
	RelationshipIsA          RelationshipType = "is_a"          // Inheritance/taxonomy
	RelationshipPartOf       RelationshipType = "part_of"       // Composition
	RelationshipRelatedTo    RelationshipType = "related_to"    // General association
	RelationshipLocatedIn    RelationshipType = "located_in"    // Spatial
	RelationshipOccurredAt   RelationshipType = "occurred_at"   // Temporal
	RelationshipCreatedBy    RelationshipType = "created_by"    // Authorship
	RelationshipMentions     RelationshipType = "mentions"      // Reference
	RelationshipDependsOn    RelationshipType = "depends_on"    // Dependency
	RelationshipImplements   RelationshipType = "implements"    // Technical
	RelationshipExtends      RelationshipType = "extends"       // Technical
	RelationshipContains     RelationshipType = "contains"      // Containment
	RelationshipReferencedBy RelationshipType = "referenced_by" // Back-reference
	RelationshipHasInstance  RelationshipType = "has_instance"  // Global entity has instance in domain
	RelationshipCustom       RelationshipType = "custom"        // User-defined
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

// CrossDomainMerging defines configuration for cross-domain entity merging
type CrossDomainMerging struct {
	Enabled             bool     `json:"enabled" yaml:"enabled"`                                   // Enable cross-domain entity merging
	SimilarityThreshold float64  `json:"similarity_threshold" yaml:"similarity_threshold"`         // Minimum similarity score (0.0-1.0) to merge entities across domains
	EvidenceTypes       []string `json:"evidence_types,omitempty" yaml:"evidence_types,omitempty"` // Types of evidence: "fuzzy_match", "semantic_similarity"
	MinDomains          int      `json:"min_domains,omitempty" yaml:"min_domains,omitempty"`       // Minimum number of domains entity must appear in (default: 2)
}

// OntologySchema defines the extraction rules for a domain
type OntologySchema struct {
	Name                    string                       `json:"name" yaml:"name"`                                                               // Schema name
	Version                 string                       `json:"version" yaml:"version"`                                                         // Schema version
	Description             string                       `json:"description" yaml:"description"`                                                 // Schema description
	LLMModel                string                       `json:"llm_model,omitempty" yaml:"llm_model,omitempty"`                                 // Default LLM model for all LLM operations (required for LLM features)
	LLMValidationModel      string                       `json:"llm_validation_model,omitempty" yaml:"llm_validation_model,omitempty"`           // LLM model for validation (optional, falls back to llm_model)
	Domain                  string                       `json:"domain" yaml:"domain"`                                                           // Primary domain (deprecated, use Domains)
	Domains                 []Domain                     `json:"domains" yaml:"domains"`                                                         // Domain registry (multi-domain support)
	DocumentTypes           []string                     `json:"document_types,omitempty" yaml:"document_types,omitempty"`                       // Applicable document types
	KeyConcepts             []string                     `json:"key_concepts,omitempty" yaml:"key_concepts,omitempty"`                           // Key concepts in domain
	Terms                   []Term                       `json:"terms,omitempty" yaml:"terms,omitempty"`                                         // Domain terms and synonyms
	ElementEntityMappings   []ElementEntityMappingConfig `json:"element_entity_mappings" yaml:"element_entity_mappings"`                         // Entity extraction rules (as-written, from YAML)
	EntityRelationshipRules []EntityRelationshipRule     `json:"entity_relationship_rules,omitempty" yaml:"entity_relationship_rules,omitempty"` // Relationship extraction rules
	DerivedEntities         []DerivedEntity              `json:"derived_entities,omitempty" yaml:"derived_entities,omitempty"`                   // Derived entity definitions
	CrossDomainMerging      *CrossDomainMerging          `json:"cross_domain_merging,omitempty" yaml:"cross_domain_merging,omitempty"`           // Cross-domain entity merging configuration
	Metadata                map[string]interface{}       `json:"metadata,omitempty" yaml:"metadata,omitempty"`                                   // Additional metadata
	CreatedAt               time.Time                    `json:"created_at" yaml:"created_at"`                                                   // Creation timestamp

	// Computed fields (populated by Validate(), not serialized)
	computedMappings []ElementEntityMapping `json:"-" yaml:"-"` // Runtime entity mappings (as-computed, with inherited/computed fields)
}

// Term defines a domain term and its synonyms
type Term struct {
	Term        string   `json:"term"`                  // Term name
	Synonyms    []string `json:"synonyms"`              // Synonyms
	Description string   `json:"description,omitempty"` // Term description
}

// ElementEntityMappingConfig is the YAML representation (as-written)
// This struct is used for loading from YAML files where some fields may be omitted
// and computed later (e.g., WCategory inherited from parent, Children computed from parent_type)
type ElementEntityMappingConfig struct {
	EntityType      string           `json:"entity_type" yaml:"entity_type"`                           // Entity type to extract
	ParentType      string           `json:"parent_type,omitempty" yaml:"parent_type,omitempty"`       // Parent entity type (e.g., "global.person")
	Children        []string         `json:"children,omitempty" yaml:"children,omitempty"`             // Child entity types (usually omitted, computed at runtime)
	Domain          string           `json:"domain" yaml:"domain"`                                     // Domain this entity belongs to (required)
	WCategory       string           `json:"w_category,omitempty" yaml:"w_category,omitempty"`         // 6 W's category (optional, inherited from parent if omitted)
	Description     string           `json:"description" yaml:"description"`                           // Description of entity type
	ElementTypes    []string         `json:"element_types,omitempty" yaml:"element_types,omitempty"`   // Simple element type filter
	ElementFilter   string           `json:"element_filter,omitempty" yaml:"element_filter,omitempty"` // JSONPath filter (advanced)
	Confidence      float64          `json:"confidence" yaml:"confidence"`                             // Context quality confidence (0.0-1.0)
	ExtractionRules []ExtractionRule `json:"extraction_rules" yaml:"extraction_rules"`                 // Rules for extraction (OR logic)
}

// ElementEntityMapping is the runtime representation (as-computed)
// This struct is used during execution where all computed fields are populated
// (e.g., WCategory always set, Children always computed)
type ElementEntityMapping struct {
	EntityType      string           `json:"entity_type"`      // Entity type to extract
	ParentType      string           `json:"parent_type"`      // Parent entity type (e.g., "global.person")
	Children        []string         `json:"children"`         // Child entity types (always computed)
	Domain          string           `json:"domain"`           // Domain this entity belongs to
	WCategory       string           `json:"w_category"`       // 6 W's category (always populated, inherited if needed)
	Description     string           `json:"description"`      // Description of entity type
	ElementTypes    []string         `json:"element_types"`    // Simple element type filter
	ElementFilter   string           `json:"element_filter"`   // JSONPath filter (advanced)
	Confidence      float64          `json:"confidence"`       // Context quality confidence (0.0-1.0)
	ExtractionRules []ExtractionRule `json:"extraction_rules"` // Rules for extraction (OR logic)
}

// ToElementEntityMapping converts the config representation to runtime representation
// This performs the initial transformation - fields like WCategory and Children will be
// fully populated by ComputeHierarchies() in Step 3
func (c *ElementEntityMappingConfig) ToElementEntityMapping() ElementEntityMapping {
	return ElementEntityMapping{
		EntityType:      c.EntityType,
		ParentType:      c.ParentType,
		Children:        c.Children, // Will be populated by ComputeHierarchies
		Domain:          c.Domain,
		WCategory:       c.WCategory, // Will be inherited by ComputeHierarchies if empty
		Description:     c.Description,
		ElementTypes:    c.ElementTypes,
		ElementFilter:   c.ElementFilter,
		Confidence:      c.Confidence,
		ExtractionRules: c.ExtractionRules,
	}
}

// GetComputedMappings returns the runtime entity mappings (as-computed)
// These are populated by Validate() and have all inherited/computed fields filled
func (s *OntologySchema) GetComputedMappings() []ElementEntityMapping {
	return s.computedMappings
}

// SemanticFilter validates entity matches using element-level semantic similarity
type SemanticFilter struct {
	ReferenceText       string   `json:"reference_text,omitempty" yaml:"reference_text,omitempty"`         // Single reference text (alternative to reference_concepts)
	ReferenceConcepts   []string `json:"reference_concepts,omitempty" yaml:"reference_concepts,omitempty"` // Concepts to compare against (alternative to reference_text)
	SimilarityThreshold float64  `json:"similarity_threshold" yaml:"similarity_threshold"`                 // Min similarity (0.0-1.0)
}

// ProximityFilter requires entity to appear near certain terms (co-occurrence/distance-based filtering)
type ProximityFilter struct {
	CooccurrenceTerms []string `json:"cooccurrence_terms" yaml:"cooccurrence_terms"`           // Terms that must appear in the same element
	MaxDistance       int      `json:"max_distance,omitempty" yaml:"max_distance,omitempty"`   // Maximum word/character distance (0 = same element, >0 = word distance)
	DistanceUnit      string   `json:"distance_unit,omitempty" yaml:"distance_unit,omitempty"` // Distance unit: "element" (default), "word", "character"
}

// DictionaryFilter validates entity matches using dictionary lookups (linguistic/semantic properties)
type DictionaryFilter struct {
	RequireUnknownWords        bool       `json:"require_unknown_words,omitempty" yaml:"require_unknown_words,omitempty"`               // At least one word NOT in dictionary (proper names)
	MaxKnownWordsRatio         float64    `json:"max_known_words_ratio,omitempty" yaml:"max_known_words_ratio,omitempty"`               // Max ratio of words in dictionary (0.0-1.0)
	RejectIfAllPOS             []string   `json:"reject_if_all_pos,omitempty" yaml:"reject_if_all_pos,omitempty"`                       // Reject if ALL words match these POS (e.g., ["noun"])
	RejectIfAllCategories      []string   `json:"reject_if_all_categories,omitempty" yaml:"reject_if_all_categories,omitempty"`         // Reject if ALL words match these categories (e.g., ["place"])
	RejectPOSCombinations      [][]string `json:"reject_pos_combinations,omitempty" yaml:"reject_pos_combinations,omitempty"`           // Reject specific POS sequences (e.g., [["noun","noun"]])
	RejectCategoryCombinations [][]string `json:"reject_category_combinations,omitempty" yaml:"reject_category_combinations,omitempty"` // Reject category sequences (e.g., [["place","noun"]])
}

// WordNetFilter is deprecated, use DictionaryFilter instead (kept for backward compatibility)
type WordNetFilter = DictionaryFilter

// LLMValidationPrompt defines LLM-based validation for filtering false positives
type LLMValidationPrompt struct {
	Prompt    string `json:"prompt" yaml:"prompt"`                             // Validation question (e.g., "Is this a valid person name?")
	BatchSize int    `json:"batch_size,omitempty" yaml:"batch_size,omitempty"` // Batch size for LLM API calls (default: 50)
}

// ExtractionRule defines a rule for extracting entities
//
// EXTRACTION METHODS (choose ONE):
//   - phrase_list: List of exact phrases to match (simple keyword extraction)
//   - instance_name: Regex with (?P<name>...) capture group to extract entity name
//
// UNIVERSAL ADDRESSING (optional):
//   - jsonpath: JSONPath expression to narrow scope before extraction
//
// PRE-FILTERS (all optional, applied in order, AND logic, fail-fast):
//  1. pattern: Regex pattern match (cheap - pre-filter on content before extraction)
//  2. proximity: Entity must appear near certain terms (moderate cost)
//  3. dictionary: Linguistic/dictionary validation (moderate cost)
//  4. semantic: Embedding similarity (expensive - applied last)
//  5. llm_validation: LLM-based validation (most expensive - applied during canonicalization)
type ExtractionRule struct {
	// Universal addressing (OPTIONAL)
	JSONPath string `json:"jsonpath,omitempty" yaml:"jsonpath,omitempty"` // JSONPath to narrow scope before extraction

	// Extraction methods (choose ONE)
	PhraseList   []string `json:"phrase_list,omitempty" yaml:"phrase_list,omitempty"`     // List of exact phrases to match
	InstanceName string   `json:"instance_name,omitempty" yaml:"instance_name,omitempty"` // Regex with (?P<name>...) capture group

	// Pre-filters (all OPTIONAL, AND logic, fail-fast)
	Pattern       string               `json:"pattern,omitempty" yaml:"pattern,omitempty"`               // Regex pattern pre-filter
	Proximity     *ProximityFilter     `json:"proximity,omitempty" yaml:"proximity,omitempty"`           // Co-occurrence filter
	Dictionary    *DictionaryFilter    `json:"dictionary,omitempty" yaml:"dictionary,omitempty"`         // Linguistic validation
	Semantic      *SemanticFilter      `json:"semantic,omitempty" yaml:"semantic,omitempty"`             // Embedding similarity
	LLMValidation *LLMValidationPrompt `json:"llm_validation,omitempty" yaml:"llm_validation,omitempty"` // LLM-based validation
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

// EntityConstraints defines optional filters that entities must satisfy
// Uses the same filter architecture as ExtractionRule to constrain which
// entities qualify as source/target nodes in a relationship
type EntityConstraints struct {
	InstanceName    string           `json:"instance_name,omitempty" yaml:"instance_name,omitempty"`       // Regex with named capture group - entity name must match
	Pattern         string           `json:"pattern,omitempty" yaml:"pattern,omitempty"`                   // Pre-filter regex for entity name
	ProximityFilter *ProximityFilter `json:"proximity_filter,omitempty" yaml:"proximity_filter,omitempty"` // Co-occurrence filter on entity context
	SemanticFilter  *SemanticFilter  `json:"semantic_filter,omitempty" yaml:"semantic_filter,omitempty"`   // Embedding similarity on entity context
}

// RelationshipExtractionPattern defines a pattern for extracting a relationship (binary match)
type RelationshipExtractionPattern struct {
	Type        RelationshipExtractionPatternType `json:"type" yaml:"type"`                                     // Pattern type
	Template    string                            `json:"template,omitempty" yaml:"template,omitempty"`         // Text template (e.g., "{person} is CEO of {organization}")
	Pattern     string                            `json:"pattern,omitempty" yaml:"pattern,omitempty"`           // Regex pattern with named groups
	SignalWords []string                          `json:"signal_words,omitempty" yaml:"signal_words,omitempty"` // Signal words indicating relationship
	MaxDistance int                               `json:"max_distance,omitempty" yaml:"max_distance,omitempty"` // Max tokens between entities (for proximity)
	Direction   string                            `json:"direction,omitempty" yaml:"direction,omitempty"`       // "forward", "backward", "bidirectional"
	Examples    []string                          `json:"examples,omitempty" yaml:"examples,omitempty"`         // Example texts matching this pattern
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
//
// INHERITANCE: Rules can extend parent rules to inherit extraction patterns and constraints.
// Example: "physician_works_at_hospital" extends "global.person_works_at_organization"
type EntityRelationshipRule struct {
	Name               string                          `json:"name" yaml:"name"`                                                   // Rule name
	ParentRelationship string                          `json:"parent_relationship,omitempty" yaml:"parent_relationship,omitempty"` // Parent rule to inherit from (e.g., "global.person_works_at_organization")
	SourceEntityType   string                          `json:"source_entity_type" yaml:"source_entity_type"`                       // Source entity type (must be subtype of parent if inheritance used)
	TargetEntityType   string                          `json:"target_entity_type" yaml:"target_entity_type"`                       // Target entity type (must be subtype of parent if inheritance used)
	RelationshipType   RelationshipType                `json:"relationship_type" yaml:"relationship_type"`                         // Relationship type (inherited from parent if not specified)
	Description        string                          `json:"description,omitempty" yaml:"description,omitempty"`                 // Rule description
	Confidence         float64                         `json:"confidence" yaml:"confidence"`                                       // Pattern reliability confidence (0.0-1.0, must be >= parent confidence)
	ExtractionPatterns []RelationshipExtractionPattern `json:"extraction_patterns" yaml:"extraction_patterns"`                     // Patterns for extracting relationship (APPENDED to parent patterns if inheritance used, OR logic)
	SourceConstraints  *EntityConstraints              `json:"source_constraints,omitempty" yaml:"source_constraints,omitempty"`   // Optional filters for source entity (ADDED to parent constraints if inheritance used, AND logic)
	TargetConstraints  *EntityConstraints              `json:"target_constraints,omitempty" yaml:"target_constraints,omitempty"`   // Optional filters for target entity (ADDED to parent constraints if inheritance used, AND logic)
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
	ID         string                 `json:"id"`         // Unique identifier
	Name       string                 `json:"name"`       // Entity name
	Type       EntityType             `json:"type"`       // Entity type
	Domain     string                 `json:"domain"`     // Domain ownership (inherited from mapping)
	Confidence float64                `json:"confidence"` // Extraction confidence (0-1)
	Attributes map[string]interface{} `json:"attributes"` // Additional properties
	ElementID  string                 `json:"element_id"` // Source UDML element
	Mentions   []Mention              `json:"mentions"`   // Where entity appears
	CreatedAt  time.Time              `json:"created_at"` // Extraction timestamp
	UpdatedAt  time.Time              `json:"updated_at"` // Last update timestamp
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
	ID         string                 `json:"id"`         // Unique identifier
	Type       RelationshipType       `json:"type"`       // Relationship type
	Domain     string                 `json:"domain"`     // Domain ownership (consumer domain, inherited from source entity being enriched)
	SourceID   string                 `json:"source_id"`  // Source entity ID
	TargetID   string                 `json:"target_id"`  // Target entity ID
	Confidence float64                `json:"confidence"` // Extraction confidence (0-1)
	Attributes map[string]interface{} `json:"attributes"` // Additional properties
	ElementID  string                 `json:"element_id"` // Source UDML element
	Evidence   string                 `json:"evidence"`   // Supporting text
	CreatedAt  time.Time              `json:"created_at"` // Extraction timestamp
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
	ID            string         `json:"id"`            // Unique identifier
	DocID         string         `json:"doc_id"`        // Source document ID
	Name          string         `json:"name"`          // Ontology name
	Description   string         `json:"description"`   // Ontology description
	Version       string         `json:"version"`       // Ontology version
	Entities      []Entity       `json:"entities"`      // Extracted entities
	Relationships []Relationship `json:"relationships"` // Extracted relationships
	Classes       []Class        `json:"classes"`       // Ontology classes
	Metadata      OntologyMeta   `json:"metadata"`      // Extraction metadata
	CreatedAt     time.Time      `json:"created_at"`    // Creation timestamp
	UpdatedAt     time.Time      `json:"updated_at"`    // Last update timestamp
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
	EntityCount       int                      `json:"entity_count"`
	RelationshipCount int                      `json:"relationship_count"`
	ClassCount        int                      `json:"class_count"`
	EntityTypes       map[EntityType]int       `json:"entity_types"`
	RelationshipTypes map[RelationshipType]int `json:"relationship_types"`
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
	errors := &ValidationErrors{}

	// Check required fields
	if s.Name == "" {
		errors.Add("schema", "schema name cannot be empty")
	}
	if s.Version == "" {
		errors.Add("schema", "schema version cannot be empty")
	}

	// Build domain registry map
	domainMap := make(map[string]bool)
	for _, domain := range s.Domains {
		if domain.Name == "" {
			errors.Add("schema", "domain has empty name")
		} else if domainMap[domain.Name] {
			errors.Add("schema", "duplicate domain name: "+domain.Name)
		} else {
			domainMap[domain.Name] = true
		}
	}

	// Check at least one entity mapping exists
	if len(s.ElementEntityMappings) == 0 {
		errors.Add("schema", "schema must have at least one entity mapping")
	}

	// Transform config mappings to runtime mappings
	s.computedMappings = make([]ElementEntityMapping, len(s.ElementEntityMappings))
	for i, configMapping := range s.ElementEntityMappings {
		s.computedMappings[i] = configMapping.ToElementEntityMapping()
	}

	// Validate entity mappings (BEFORE extraction rules check)
	for i, mapping := range s.ElementEntityMappings {
		if mapping.EntityType == "" {
			errors.Add("schema", fmt.Sprintf("entity mapping %d has empty entity_type", i))
		}
		if mapping.Domain == "" {
			errors.Add("schema", fmt.Sprintf("entity mapping %d (%s) has empty domain", i, mapping.EntityType))
		}
		// Validate domain exists in registry (if registry is populated)
		if len(s.Domains) > 0 && !domainMap[mapping.Domain] {
			errors.Add("schema", fmt.Sprintf("entity mapping %d (%s) references unknown domain: %s", i, mapping.EntityType, mapping.Domain))
		}
		if mapping.Confidence < 0.0 || mapping.Confidence > 1.0 {
			errors.Add("schema", fmt.Sprintf("entity mapping %d (%s) has invalid confidence: %.2f (must be 0.0-1.0)", i, mapping.EntityType, mapping.Confidence))
		}

		// Validate extraction rules (not checking if empty yet - that comes after hierarchy computation)
		for j, rule := range mapping.ExtractionRules {
			// At least one extraction method required
			hasExtractionMethod := len(rule.PhraseList) > 0 || rule.InstanceName != ""
			if !hasExtractionMethod {
				errors.Add("schema", fmt.Sprintf("entity mapping %d (%s), rule %d: must have phrase_list or instance_name", i, mapping.EntityType, j))
			}

			// If instance_name used, must have (?P<name>...) capture group
			if rule.InstanceName != "" {
				if !strings.Contains(rule.InstanceName, "(?P<name>") {
					errors.Add("schema", fmt.Sprintf("entity mapping %d (%s), rule %d: instance_name must contain (?P<name>...) capture group", i, mapping.EntityType, j))
				}
			}

			// Validate proximity filter if present
			if rule.Proximity != nil {
				if len(rule.Proximity.CooccurrenceTerms) == 0 {
					errors.Add("schema", fmt.Sprintf("entity mapping %d (%s), rule %d: proximity filter requires cooccurrence_terms", i, mapping.EntityType, j))
				}
				if rule.Proximity.DistanceUnit != "" {
					validUnits := map[string]bool{"element": true, "word": true, "character": true}
					if !validUnits[rule.Proximity.DistanceUnit] {
						errors.Add("schema", fmt.Sprintf("entity mapping %d (%s), rule %d: invalid proximity.distance_unit: %s (must be 'element', 'word', or 'character')", i, mapping.EntityType, j, rule.Proximity.DistanceUnit))
					}
				}
			}

			// Validate semantic filter if present
			if rule.Semantic != nil {
				if rule.Semantic.ReferenceText == "" && len(rule.Semantic.ReferenceConcepts) == 0 {
					errors.Add("schema", fmt.Sprintf("entity mapping %d (%s), rule %d: semantic filter requires reference_text or reference_concepts", i, mapping.EntityType, j))
				}
				if rule.Semantic.SimilarityThreshold < 0.0 || rule.Semantic.SimilarityThreshold > 1.0 {
					errors.Add("schema", fmt.Sprintf("entity mapping %d (%s), rule %d: invalid semantic.similarity_threshold: %.2f (must be 0.0-1.0)", i, mapping.EntityType, j, rule.Semantic.SimilarityThreshold))
				}
			}

			// Validate dictionary filter if present
			if rule.Dictionary != nil {
				// Dictionary filter validation (structure is self-validating)
			}

			// Validate LLM validation if present
			if rule.LLMValidation != nil {
				if rule.LLMValidation.Prompt == "" {
					errors.Add("schema", fmt.Sprintf("entity mapping %d (%s), rule %d: llm_validation requires prompt", i, mapping.EntityType, j))
				}
			}
		}
	}

	// Validate relationship rules
	for i, rule := range s.EntityRelationshipRules {
		if rule.Name == "" {
			errors.Add("schema", fmt.Sprintf("relationship rule %d has empty name", i))
		}
		if rule.SourceEntityType == "" {
			errors.Add("schema", fmt.Sprintf("relationship rule %d (%s) has empty source_entity_type", i, rule.Name))
		}
		if rule.TargetEntityType == "" {
			errors.Add("schema", fmt.Sprintf("relationship rule %d (%s) has empty target_entity_type", i, rule.Name))
		}
		if rule.Confidence < 0.0 || rule.Confidence > 1.0 {
			errors.Add("schema", fmt.Sprintf("relationship rule %d (%s) has invalid confidence: %.2f (must be 0.0-1.0)", i, rule.Name, rule.Confidence))
		}
		if len(rule.ExtractionPatterns) == 0 {
			errors.Add("schema", fmt.Sprintf("relationship rule %d (%s) has no extraction patterns", i, rule.Name))
		}
	}

	// Compute bidirectional hierarchies on runtime mappings
	if err := s.ComputeHierarchies(); err != nil {
		errors.Add("graph", fmt.Sprintf("hierarchy computation failed: %v", err))
	}

	// NOW validate extraction rules AFTER merging (should be inherited)
	for i, mapping := range s.computedMappings {
		if len(mapping.ExtractionRules) == 0 {
			errors.Add("schema", fmt.Sprintf("entity mapping %d (%s) has no extraction rules (even after parent inheritance)", i, mapping.EntityType))
		}
	}

	// Validate hierarchies
	if err := s.ValidateHierarchies(); err != nil {
		errors.Add("graph", fmt.Sprintf("hierarchy validation failed: %v", err))
	}

	// Return errors if any were collected
	if !errors.IsEmpty() {
		return errors
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

// ValidationErrors collects multiple validation errors
type ValidationErrors struct {
	SchemaErrors []string // Schema-level errors
	GraphErrors  []string // Graph/hierarchy errors
}

// Error implements the error interface
func (e *ValidationErrors) Error() string {
	var result strings.Builder
	result.WriteString("ontology validation errors:\n")

	if len(e.SchemaErrors) > 0 {
		result.WriteString("\nSchema Errors:\n")
		for i, err := range e.SchemaErrors {
			result.WriteString(fmt.Sprintf("  %d. %s\n", i+1, err))
		}
	}

	if len(e.GraphErrors) > 0 {
		result.WriteString("\nGraph Errors:\n")
		for i, err := range e.GraphErrors {
			result.WriteString(fmt.Sprintf("  %d. %s\n", i+1, err))
		}
	}

	return result.String()
}

// IsEmpty returns true if no errors collected
func (e *ValidationErrors) IsEmpty() bool {
	return len(e.SchemaErrors) == 0 && len(e.GraphErrors) == 0
}

// Add adds an error to the appropriate category
func (e *ValidationErrors) Add(category string, message string) {
	if category == "schema" {
		e.SchemaErrors = append(e.SchemaErrors, message)
	} else if category == "graph" {
		e.GraphErrors = append(e.GraphErrors, message)
	}
}

// ============================================================================
// DISTRIBUTED EXTRACTION - Task coordination for billion-scale extraction
// ============================================================================

// ExtractionTaskType represents the type of extraction task
type ExtractionTaskType string

const (
	TaskTypeEntityMapping    ExtractionTaskType = "entity_mapping"    // Entity extraction task
	TaskTypeRelationshipRule ExtractionTaskType = "relationship_rule" // Relationship extraction task
)

// ExtractionTaskStatus represents the status of an extraction task
type ExtractionTaskStatus string

const (
	TaskStatusPending   ExtractionTaskStatus = "pending"   // Waiting to be claimed
	TaskStatusClaimed   ExtractionTaskStatus = "claimed"   // Claimed by a worker
	TaskStatusCompleted ExtractionTaskStatus = "completed" // Successfully completed
	TaskStatusFailed    ExtractionTaskStatus = "failed"    // Failed with error
)

// ExtractionTask represents a unit of work for distributed extraction
type ExtractionTask struct {
	ID               string               // Unique task ID
	RunID            string               // Extraction run ID (for grouping)
	Type             ExtractionTaskType   // Task type (entity or relationship)
	EntityType       string               // For entity tasks: entity type to extract
	RelationshipType string               // For relationship tasks: relationship type to extract
	DocIDs           []string             // Batch of document IDs to process
	Status           ExtractionTaskStatus // Current task status
	ClaimedBy        string               // Worker ID that claimed the task
	ClaimedAt        *time.Time           // When task was claimed
	CompletedAt      *time.Time           // When task completed
	Error            string               // Error message (if failed)
	CreatedAt        time.Time            // Task creation time
}

// ExtractionTaskResult represents the result of a completed extraction task
type ExtractionTaskResult struct {
	EntitiesExtracted      int // Number of entities extracted
	RelationshipsExtracted int // Number of relationships extracted
	ElementsProcessed      int // Number of elements processed
}

// ComputeHierarchies fills missing parent_type and children relationships bidirectionally
// This operates on the computed runtime mappings (s.computedMappings), not the config mappings
func (s *OntologySchema) ComputeHierarchies() error {
	// Initialize computedMappings from ElementEntityMappings if empty (for standalone use)
	if len(s.computedMappings) == 0 && len(s.ElementEntityMappings) > 0 {
		s.computedMappings = make([]ElementEntityMapping, len(s.ElementEntityMappings))
		for i, config := range s.ElementEntityMappings {
			s.computedMappings[i] = ElementEntityMapping{
				EntityType:      config.EntityType,
				Domain:          config.Domain,
				Confidence:      config.Confidence,
				Description:     config.Description,
				ParentType:      config.ParentType,
				Children:        config.Children,
				ElementTypes:    config.ElementTypes,
				ElementFilter:   config.ElementFilter,
				ExtractionRules: config.ExtractionRules,
			}
		}
	}

	// Build entity map (first occurrence per qualified name) from runtime mappings
	entityMap := make(map[string]*ElementEntityMapping)
	for i := range s.computedMappings {
		mapping := &s.computedMappings[i]
		key := mapping.Domain + "." + mapping.EntityType

		if _, exists := entityMap[key]; !exists {
			entityMap[key] = mapping
		}
	}

	// Phase 1: Fill parent's children from child's parent_type
	for i := range s.computedMappings {
		mapping := &s.computedMappings[i]

		if mapping.ParentType != "" {
			parent := entityMap[mapping.ParentType]
			if parent != nil {
				childRef := mapping.Domain + "." + mapping.EntityType
				if !contains(parent.Children, childRef) {
					parent.Children = append(parent.Children, childRef)
				}
			}
		}
	}

	// Phase 2: Fill child's parent_type from parent's children
	for i := range s.computedMappings {
		mapping := &s.computedMappings[i]

		for _, childRef := range mapping.Children {
			child := entityMap[childRef]
			if child != nil && child.ParentType == "" {
				child.ParentType = mapping.Domain + "." + mapping.EntityType
			}
		}
	}

	// Phase 3: Inherit WCategory from parent if not set
	for i := range s.computedMappings {
		mapping := &s.computedMappings[i]

		if mapping.WCategory == "" && mapping.ParentType != "" {
			parent := entityMap[mapping.ParentType]
			if parent != nil && parent.WCategory != "" {
				mapping.WCategory = parent.WCategory
			}
		}
	}

	// Phase 4: Merge attributes from parent to child
	// Child values take precedence, but extraction rules are UNION
	for i := range s.computedMappings {
		mapping := &s.computedMappings[i]

		if mapping.ParentType != "" {
			parent := entityMap[mapping.ParentType]
			if parent != nil {
				// Union extraction rules (child + parent, no duplicates)
				mapping.ExtractionRules = unionExtractionRules(
					mapping.ExtractionRules,
					parent.ExtractionRules,
				)

				// Inherit element types if child doesn't specify
				if len(mapping.ElementTypes) == 0 && len(parent.ElementTypes) > 0 {
					mapping.ElementTypes = append([]string{}, parent.ElementTypes...)
				}

				// Inherit element filter if child doesn't specify
				if mapping.ElementFilter == "" && parent.ElementFilter != "" {
					mapping.ElementFilter = parent.ElementFilter
				}

				// Description and Confidence: child overrides, no inheritance
			}
		}
	}

	return nil
}

// ValidateHierarchies checks for broken references and circular hierarchies
func (s *OntologySchema) ValidateHierarchies() error {
	// Build entity existence map from runtime mappings
	entityMap := make(map[string]bool)
	for _, mapping := range s.computedMappings {
		key := mapping.Domain + "." + mapping.EntityType
		entityMap[key] = true
	}

	// Check for broken references
	for _, mapping := range s.computedMappings {
		qualifiedName := mapping.Domain + "." + mapping.EntityType

		// Check parent exists
		if mapping.ParentType != "" && !entityMap[mapping.ParentType] {
			return fmt.Errorf("entity %s references non-existent parent: %s",
				qualifiedName, mapping.ParentType)
		}

		// Check children exist
		for _, childRef := range mapping.Children {
			if !entityMap[childRef] {
				return fmt.Errorf("entity %s references non-existent child: %s",
					qualifiedName, childRef)
			}
		}
	}

	// Check for circular hierarchies
	for _, mapping := range s.computedMappings {
		qualifiedName := mapping.Domain + "." + mapping.EntityType
		if s.hasCycle(qualifiedName, make(map[string]bool)) {
			return fmt.Errorf("circular hierarchy detected involving: %s", qualifiedName)
		}
	}

	return nil
}

// hasCycle detects circular parent references using DFS
func (s *OntologySchema) hasCycle(entityName string, visited map[string]bool) bool {
	if visited[entityName] {
		return true // Cycle detected
	}

	// Find entity mapping in runtime mappings
	var parentType string
	for _, mapping := range s.computedMappings {
		if mapping.Domain+"."+mapping.EntityType == entityName {
			parentType = mapping.ParentType
			break
		}
	}

	if parentType == "" {
		return false // Reached top
	}

	visited[entityName] = true
	result := s.hasCycle(parentType, visited)
	delete(visited, entityName) // Backtrack

	return result
}

// unionExtractionRules returns unique union of child and parent extraction rules
// Child rules come first, then non-duplicate parent rules
func unionExtractionRules(child, parent []ExtractionRule) []ExtractionRule {
	if len(parent) == 0 {
		return child
	}
	if len(child) == 0 {
		return append([]ExtractionRule{}, parent...)
	}

	result := append([]ExtractionRule{}, child...)

	// Add parent rules that don't duplicate child rules
	for _, parentRule := range parent {
		isDuplicate := false
		for _, childRule := range child {
			if extractionRulesEqual(parentRule, childRule) {
				isDuplicate = true
				break
			}
		}
		if !isDuplicate {
			result = append(result, parentRule)
		}
	}

	return result
}

// extractionRulesEqual checks if two extraction rules are functionally equivalent
func extractionRulesEqual(a, b ExtractionRule) bool {
	// Simple equality check - compare JSONPath and phrase lists
	if a.JSONPath != b.JSONPath {
		return false
	}
	if a.InstanceName != b.InstanceName {
		return false
	}
	if !stringSlicesEqual(a.PhraseList, b.PhraseList) {
		return false
	}
	if a.Pattern != b.Pattern {
		return false
	}
	// Consider rules equal if these key fields match
	// (filters like Proximity, Dictionary, Semantic are not compared for simplicity)
	return true
}

// stringSlicesEqual checks if two string slices are equal
func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// contains checks if string slice contains value
func contains(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}
