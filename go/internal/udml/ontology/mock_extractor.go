package ontology

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"
)

// MockExtractor is a mock implementation of OntologyExtractor for testing
type MockExtractor struct {
	name    string
	version string
}

// init registers the mock extractor
func init() {
	Register("mock", NewMockExtractor)
}

// NewMockExtractor creates a new mock extractor
func NewMockExtractor(config ExtractorConfig) (OntologyExtractor, error) {
	return &MockExtractor{
		name:    "mock",
		version: "1.0.0",
	}, nil
}

// GetName returns the extractor identifier
func (m *MockExtractor) GetName() string {
	return m.name
}

// GetVersion returns the extractor version
func (m *MockExtractor) GetVersion() string {
	return m.version
}

// ExtractFromText extracts ontology from text using simple pattern matching
func (m *MockExtractor) ExtractFromText(ctx context.Context, text string, options ExtractionOptions) (*Ontology, error) {
	ontology := &Ontology{
		ID:            generateID("ont"),
		DocID:         options.DocID,
		Name:          options.DocName,
		Description:   "Mock ontology extracted from text",
		Version:       "1.0.0",
		Entities:      []Entity{},
		Relationships: []Relationship{},
		Classes:       []Class{},
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Extract simple entities based on capitalization patterns
	entities := m.extractSimpleEntities(text, options)
	ontology.Entities = entities

	// Extract simple relationships based on common patterns
	relationships := m.extractSimpleRelationships(entities, text, options)
	ontology.Relationships = relationships

	// Extract simple classes if enabled
	if options.ExtractClasses {
		classes := m.extractSimpleClasses(text, options)
		ontology.Classes = classes
	}

	// Set metadata
	ontology.Metadata = OntologyMeta{
		ExtractorType:    "mock",
		ExtractorVersion: m.version,
		ElementCount:     1,
		Confidence:       0.5,
		ModelName:        "mock-model",
		PromptTokens:     len(text) / 4, // Rough approximation
		CompletionTokens: 100,
		CustomFields:     make(map[string]interface{}),
	}

	return ontology, nil
}

// ExtractFromElements extracts ontology from structured elements
func (m *MockExtractor) ExtractFromElements(ctx context.Context, elements []Element, options ExtractionOptions) (*Ontology, error) {
	// Combine all element content
	var sb strings.Builder
	for _, elem := range elements {
		sb.WriteString(elem.Content)
		sb.WriteString("\n\n")
	}

	text := sb.String()
	ontology, err := m.ExtractFromText(ctx, text, options)
	if err != nil {
		return nil, err
	}

	// Link entities to elements
	for i := range ontology.Entities {
		if i < len(elements) {
			ontology.Entities[i].ElementID = elements[i].ElementID
		}
	}

	ontology.Metadata.ElementCount = len(elements)
	return ontology, nil
}

// SupportsFeature checks if the extractor supports a specific feature
func (m *MockExtractor) SupportsFeature(feature string) bool {
	supported := map[string]bool{
		"entity_extraction":       true,
		"relationship_extraction": true,
		"class_extraction":        true,
		"batch_processing":        true,
		"streaming":               false,
	}
	return supported[feature]
}

// Close cleans up resources
func (m *MockExtractor) Close() error {
	return nil
}

// extractSimpleEntities extracts entities using simple pattern matching
func (m *MockExtractor) extractSimpleEntities(text string, options ExtractionOptions) []Entity {
	entities := []Entity{}
	words := strings.Fields(text)

	seen := make(map[string]bool)
	entityCount := 0

	for _, word := range words {
		// Simple heuristic: capitalized words might be entities
		if len(word) < 2 {
			continue
		}

		// Remove punctuation
		word = strings.Trim(word, ".,;:!?\"'()[]{}")

		// Check if starts with capital letter
		if word[0] >= 'A' && word[0] <= 'Z' {
			// Skip if already seen or too many entities
			if seen[word] || (options.MaxEntities > 0 && entityCount >= options.MaxEntities) {
				continue
			}

			// Determine entity type based on simple heuristics
			entityType := m.guessEntityType(word, text)

			// Check if type is in filter
			if len(options.EntityTypes) > 0 {
				typeAllowed := false
				for _, t := range options.EntityTypes {
					if t == entityType {
						typeAllowed = true
						break
					}
				}
				if !typeAllowed {
					continue
				}
			}

			entity := Entity{
				ID:         generateID("ent"),
				Name:       word,
				Type:       entityType,
				Confidence: 0.7, // Mock confidence
				Attributes: make(map[string]interface{}),
				Mentions:   []Mention{},
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			}

			entities = append(entities, entity)
			seen[word] = true
			entityCount++
		}
	}

	return entities
}

// extractSimpleRelationships extracts relationships using pattern matching
func (m *MockExtractor) extractSimpleRelationships(entities []Entity, text string, options ExtractionOptions) []Relationship {
	relationships := []Relationship{}

	if len(entities) < 2 {
		return relationships
	}

	// Simple pattern: entities that appear near each other might be related
	relationshipCount := 0
	for i, e1 := range entities {
		if options.MaxRelationships > 0 && relationshipCount >= options.MaxRelationships {
			break
		}

		for j := i + 1; j < len(entities); j++ {
			e2 := entities[j]

			// Check if both entities appear in text
			if !strings.Contains(text, e1.Name) || !strings.Contains(text, e2.Name) {
				continue
			}

			// Create a simple relationship
			relType := RelationshipRelatedTo // Default type
			if e1.Type == EntityTypePerson && e2.Type == EntityTypeOrganization {
				relType = RelationshipPartOf
			} else if e1.Type == EntityTypeEvent && e2.Type == EntityTypeLocation {
				relType = RelationshipLocatedIn
			}

			// Check if type is in filter
			if len(options.RelationshipTypes) > 0 {
				typeAllowed := false
				for _, t := range options.RelationshipTypes {
					if t == relType {
						typeAllowed = true
						break
					}
				}
				if !typeAllowed {
					continue
				}
			}

			rel := Relationship{
				ID:         generateID("rel"),
				Type:       relType,
				SourceID:   e1.ID,
				TargetID:   e2.ID,
				Confidence: 0.6, // Mock confidence
				Attributes: make(map[string]interface{}),
				Evidence:   fmt.Sprintf("Both %s and %s appear in the text", e1.Name, e2.Name),
				CreatedAt:  time.Now(),
			}

			relationships = append(relationships, rel)
			relationshipCount++

			if options.MaxRelationships > 0 && relationshipCount >= options.MaxRelationships {
				break
			}
		}
	}

	return relationships
}

// extractSimpleClasses extracts ontology classes using simple heuristics
func (m *MockExtractor) extractSimpleClasses(text string, options ExtractionOptions) []Class {
	classes := []Class{}

	// Simple heuristic: look for definitions or "is a" patterns
	sentences := strings.Split(text, ".")
	seen := make(map[string]bool)

	for _, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if len(sentence) < 10 {
			continue
		}

		// Look for "is a" or "is an" patterns
		if strings.Contains(strings.ToLower(sentence), " is a ") ||
			strings.Contains(strings.ToLower(sentence), " is an ") {

			words := strings.Fields(sentence)
			if len(words) > 2 {
				// Extract potential class name (first capitalized word)
				for _, word := range words {
					word = strings.Trim(word, ".,;:!?\"'()[]{}")
					if len(word) > 2 && word[0] >= 'A' && word[0] <= 'Z' && !seen[word] {
						class := Class{
							ID:          generateID("cls"),
							Name:        word,
							Description: sentence,
							Properties:  []Property{},
							Attributes:  make(map[string]interface{}),
							CreatedAt:   time.Now(),
						}
						classes = append(classes, class)
						seen[word] = true
						break
					}
				}
			}
		}
	}

	return classes
}

// guessEntityType guesses the entity type based on simple heuristics
func (m *MockExtractor) guessEntityType(word string, context string) EntityType {
	lowerWord := strings.ToLower(word)

	// Check for common organizational suffixes
	if strings.HasSuffix(lowerWord, "corp") ||
		strings.HasSuffix(lowerWord, "inc") ||
		strings.HasSuffix(lowerWord, "ltd") ||
		strings.HasSuffix(lowerWord, "llc") {
		return EntityTypeOrganization
	}

	// Check for common personal titles in context
	contextLower := strings.ToLower(context)
	if strings.Contains(contextLower, "mr. "+lowerWord) ||
		strings.Contains(contextLower, "mrs. "+lowerWord) ||
		strings.Contains(contextLower, "dr. "+lowerWord) {
		return EntityTypePerson
	}

	// Check for date patterns
	if len(word) == 4 {
		// Might be a year
		return EntityTypeDate
	}

	// Check for location indicators
	if strings.HasSuffix(lowerWord, "land") ||
		strings.HasSuffix(lowerWord, "ville") ||
		strings.HasSuffix(lowerWord, "city") ||
		strings.HasSuffix(lowerWord, "ton") {
		return EntityTypeLocation
	}

	// Default to concept
	return EntityTypeConcept
}

// idCounter is a global counter for generating unique IDs
var idCounter uint64

// generateID generates a simple unique ID
func generateID(prefix string) string {
	count := atomic.AddUint64(&idCounter, 1)
	return fmt.Sprintf("%s_%d_%d", prefix, time.Now().UnixNano()/1000000, count)
}
