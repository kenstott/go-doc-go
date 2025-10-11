package ontology

import (
	"context"
	"fmt"
	"sync"
)

// OntologyExtractor defines the interface for extracting ontologies from documents
type OntologyExtractor interface {
	// GetName returns the extractor identifier
	GetName() string

	// GetVersion returns the extractor version
	GetVersion() string

	// ExtractFromText extracts entities, relationships, and classes from text
	ExtractFromText(ctx context.Context, text string, options ExtractionOptions) (*Ontology, error)

	// ExtractFromElements extracts ontology from structured UDML elements
	ExtractFromElements(ctx context.Context, elements []Element, options ExtractionOptions) (*Ontology, error)

	// SupportsFeature checks if the extractor supports a specific feature
	SupportsFeature(feature string) bool

	// Close cleans up any resources used by the extractor
	Close() error
}

// Element represents a minimal UDML element for extraction
// We use a simplified version to avoid circular dependencies
type Element struct {
	ElementID       string                 `json:"element_id"`
	ElementType     string                 `json:"element_type"`
	Content         string                 `json:"content"`
	ContentPreview  string                 `json:"content_preview"`
	ParentID        string                 `json:"parent_id"`
	ElementOrder    float64                `json:"element_order"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// ExtractionOptions controls the ontology extraction process
type ExtractionOptions struct {
	// DocID is the document identifier
	DocID string

	// DocName is the document name/title
	DocName string

	// MaxEntities limits the number of entities to extract (0 = no limit)
	MaxEntities int

	// MaxRelationships limits the number of relationships to extract (0 = no limit)
	MaxRelationships int

	// EntityTypes filters which entity types to extract (nil = all types)
	EntityTypes []EntityType

	// RelationshipTypes filters which relationship types to extract (nil = all types)
	RelationshipTypes []RelationshipType

	// ExtractClasses enables ontology class extraction
	ExtractClasses bool

	// MinConfidence filters extractions below this confidence threshold (0-1)
	MinConfidence float64

	// ContextWindow sets the context window size for extraction (in characters)
	ContextWindow int

	// BatchSize controls how many elements to process in one LLM call
	BatchSize int

	// Temperature controls LLM creativity (0 = deterministic, 1 = creative)
	Temperature float64

	// CustomPrompt allows overriding the default extraction prompt
	CustomPrompt string

	// CustomFields allows passing provider-specific options
	CustomFields map[string]interface{}
}

// DefaultExtractionOptions returns default extraction options
func DefaultExtractionOptions() ExtractionOptions {
	return ExtractionOptions{
		MaxEntities:       0,    // No limit
		MaxRelationships:  0,    // No limit
		EntityTypes:       nil,  // All types
		RelationshipTypes: nil,  // All types
		ExtractClasses:    true, // Extract classes by default
		MinConfidence:     0.5,  // 50% minimum confidence
		ContextWindow:     4000, // 4000 characters context
		BatchSize:         10,   // 10 elements per batch
		Temperature:       0.0,  // Deterministic
		CustomFields:      make(map[string]interface{}),
	}
}

// ExtractorFactory is a function that creates an OntologyExtractor
type ExtractorFactory func(config ExtractorConfig) (OntologyExtractor, error)

// ExtractorConfig contains configuration for creating an extractor
type ExtractorConfig struct {
	Type         string                 // Extractor type (e.g., "claude", "openai", "mock")
	APIKey       string                 // API key for the LLM provider
	ModelName    string                 // Model name to use
	BaseURL      string                 // Base URL for API (optional)
	Timeout      int                    // Request timeout in seconds
	MaxRetries   int                    // Maximum retry attempts
	CustomFields map[string]interface{} // Provider-specific config
}

// ExtractorRegistry manages available ontology extractors
type ExtractorRegistry struct {
	mu         sync.RWMutex
	extractors map[string]ExtractorFactory
}

var (
	// globalRegistry is the global extractor registry
	globalRegistry = &ExtractorRegistry{
		extractors: make(map[string]ExtractorFactory),
	}
)

// Register registers an extractor factory in the global registry
func Register(name string, factory ExtractorFactory) {
	globalRegistry.Register(name, factory)
}

// Register registers an extractor factory
func (r *ExtractorRegistry) Register(name string, factory ExtractorFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.extractors[name] = factory
}

// GetExtractor retrieves an extractor factory by name
func (r *ExtractorRegistry) GetExtractor(name string) (ExtractorFactory, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	factory, exists := r.extractors[name]
	return factory, exists
}

// GetRegisteredExtractors returns all registered extractor names
func (r *ExtractorRegistry) GetRegisteredExtractors() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.extractors))
	for name := range r.extractors {
		names = append(names, name)
	}
	return names
}

// CreateExtractor creates an extractor instance from the global registry
func CreateExtractor(config ExtractorConfig) (OntologyExtractor, error) {
	return globalRegistry.CreateExtractor(config)
}

// CreateExtractor creates an extractor instance
func (r *ExtractorRegistry) CreateExtractor(config ExtractorConfig) (OntologyExtractor, error) {
	factory, exists := r.GetExtractor(config.Type)
	if !exists {
		return nil, fmt.Errorf("unknown extractor type: %s", config.Type)
	}
	return factory(config)
}

// MergeOntologies combines multiple ontologies into one
// Handles deduplication and relationship merging
func MergeOntologies(ontologies ...*Ontology) (*Ontology, error) {
	if len(ontologies) == 0 {
		return nil, fmt.Errorf("no ontologies to merge")
	}

	if len(ontologies) == 1 {
		return ontologies[0], nil
	}

	merged := &Ontology{
		ID:            ontologies[0].ID,
		DocID:         ontologies[0].DocID,
		Name:          ontologies[0].Name,
		Description:   ontologies[0].Description,
		Version:       ontologies[0].Version,
		Entities:      []Entity{},
		Relationships: []Relationship{},
		Classes:       []Class{},
		CreatedAt:     ontologies[0].CreatedAt,
		UpdatedAt:     ontologies[0].UpdatedAt,
	}

	// Track seen IDs for deduplication
	entityIDs := make(map[string]bool)
	relationshipIDs := make(map[string]bool)
	classIDs := make(map[string]bool)

	// Merge entities
	for _, ont := range ontologies {
		for _, entity := range ont.Entities {
			if !entityIDs[entity.ID] {
				merged.Entities = append(merged.Entities, entity)
				entityIDs[entity.ID] = true
			}
		}
	}

	// Merge relationships
	for _, ont := range ontologies {
		for _, rel := range ont.Relationships {
			if !relationshipIDs[rel.ID] {
				merged.Relationships = append(merged.Relationships, rel)
				relationshipIDs[rel.ID] = true
			}
		}
	}

	// Merge classes
	for _, ont := range ontologies {
		for _, class := range ont.Classes {
			if !classIDs[class.ID] {
				merged.Classes = append(merged.Classes, class)
				classIDs[class.ID] = true
			}
		}
	}

	// Update metadata
	totalElements := 0
	totalPromptTokens := 0
	totalCompletionTokens := 0
	for _, ont := range ontologies {
		totalElements += ont.Metadata.ElementCount
		totalPromptTokens += ont.Metadata.PromptTokens
		totalCompletionTokens += ont.Metadata.CompletionTokens
	}

	merged.Metadata = OntologyMeta{
		ExtractorType:    ontologies[0].Metadata.ExtractorType,
		ExtractorVersion: ontologies[0].Metadata.ExtractorVersion,
		ElementCount:     totalElements,
		PromptTokens:     totalPromptTokens,
		CompletionTokens: totalCompletionTokens,
		CustomFields:     make(map[string]interface{}),
	}

	return merged, nil
}

// FilterOntology filters an ontology based on criteria
func FilterOntology(ont *Ontology, filter OntologyFilter) *Ontology {
	filtered := &Ontology{
		ID:            ont.ID,
		DocID:         ont.DocID,
		Name:          ont.Name,
		Description:   ont.Description,
		Version:       ont.Version,
		Entities:      []Entity{},
		Relationships: []Relationship{},
		Classes:       ont.Classes, // Classes are not filtered
		Metadata:      ont.Metadata,
		CreatedAt:     ont.CreatedAt,
		UpdatedAt:     ont.UpdatedAt,
	}

	// Filter entities
	for _, entity := range ont.Entities {
		if filter.AcceptEntity(entity) {
			filtered.Entities = append(filtered.Entities, entity)
		}
	}

	// Build entity ID set for relationship filtering
	entityIDs := make(map[string]bool)
	for _, entity := range filtered.Entities {
		entityIDs[entity.ID] = true
	}

	// Filter relationships (only keep if both source and target are in filtered entities)
	for _, rel := range ont.Relationships {
		if entityIDs[rel.SourceID] && entityIDs[rel.TargetID] && filter.AcceptRelationship(rel) {
			filtered.Relationships = append(filtered.Relationships, rel)
		}
	}

	return filtered
}

// OntologyFilter defines filtering criteria for ontologies
type OntologyFilter struct {
	MinConfidence     float64
	EntityTypes       []EntityType
	RelationshipTypes []RelationshipType
	ElementIDs        []string
}

// AcceptEntity checks if an entity passes the filter
func (f *OntologyFilter) AcceptEntity(entity Entity) bool {
	// Check confidence
	if entity.Confidence < f.MinConfidence {
		return false
	}

	// Check entity type
	if len(f.EntityTypes) > 0 {
		found := false
		for _, t := range f.EntityTypes {
			if entity.Type == t {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check element ID
	if len(f.ElementIDs) > 0 {
		found := false
		for _, id := range f.ElementIDs {
			if entity.ElementID == id {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// AcceptRelationship checks if a relationship passes the filter
func (f *OntologyFilter) AcceptRelationship(rel Relationship) bool {
	// Check confidence
	if rel.Confidence < f.MinConfidence {
		return false
	}

	// Check relationship type
	if len(f.RelationshipTypes) > 0 {
		found := false
		for _, t := range f.RelationshipTypes {
			if rel.Type == t {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}
