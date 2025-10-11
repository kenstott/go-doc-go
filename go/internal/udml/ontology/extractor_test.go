package ontology

import (
	"context"
	"testing"
	"time"
)

func TestExtractorRegistry_Register(t *testing.T) {
	registry := &ExtractorRegistry{
		extractors: make(map[string]ExtractorFactory),
	}

	factory := func(config ExtractorConfig) (OntologyExtractor, error) {
		return &MockExtractor{}, nil
	}

	registry.Register("test", factory)

	if len(registry.extractors) != 1 {
		t.Errorf("Expected 1 extractor, got %d", len(registry.extractors))
	}
}

func TestExtractorRegistry_GetExtractor(t *testing.T) {
	registry := &ExtractorRegistry{
		extractors: make(map[string]ExtractorFactory),
	}

	factory := func(config ExtractorConfig) (OntologyExtractor, error) {
		return &MockExtractor{}, nil
	}

	registry.Register("test", factory)

	retrieved, exists := registry.GetExtractor("test")
	if !exists {
		t.Error("Expected extractor to exist")
	}

	if retrieved == nil {
		t.Error("Retrieved extractor factory is nil")
	}

	_, notExists := registry.GetExtractor("nonexistent")
	if notExists {
		t.Error("Expected false for nonexistent extractor")
	}
}

func TestExtractorRegistry_GetRegisteredExtractors(t *testing.T) {
	registry := &ExtractorRegistry{
		extractors: make(map[string]ExtractorFactory),
	}

	factory := func(config ExtractorConfig) (OntologyExtractor, error) {
		return &MockExtractor{}, nil
	}

	registry.Register("mock", factory)
	registry.Register("test", factory)

	names := registry.GetRegisteredExtractors()
	if len(names) != 2 {
		t.Errorf("Expected 2 extractors, got %d", len(names))
	}
}

func TestExtractorRegistry_CreateExtractor(t *testing.T) {
	registry := &ExtractorRegistry{
		extractors: make(map[string]ExtractorFactory),
	}

	factory := func(config ExtractorConfig) (OntologyExtractor, error) {
		return &MockExtractor{name: config.Type}, nil
	}

	registry.Register("mock", factory)

	config := ExtractorConfig{
		Type: "mock",
	}

	extractor, err := registry.CreateExtractor(config)
	if err != nil {
		t.Fatalf("CreateExtractor() error = %v", err)
	}

	if extractor.GetName() != "mock" {
		t.Errorf("Expected name 'mock', got '%s'", extractor.GetName())
	}
}

func TestExtractorRegistry_CreateExtractorUnknown(t *testing.T) {
	registry := &ExtractorRegistry{
		extractors: make(map[string]ExtractorFactory),
	}

	config := ExtractorConfig{
		Type: "unknown",
	}

	_, err := registry.CreateExtractor(config)
	if err == nil {
		t.Error("Expected error for unknown extractor type")
	}
}

func TestDefaultExtractionOptions(t *testing.T) {
	opts := DefaultExtractionOptions()

	if opts.MaxEntities != 0 {
		t.Errorf("Expected MaxEntities 0, got %d", opts.MaxEntities)
	}

	if opts.MaxRelationships != 0 {
		t.Errorf("Expected MaxRelationships 0, got %d", opts.MaxRelationships)
	}

	if !opts.ExtractClasses {
		t.Error("Expected ExtractClasses to be true")
	}

	if opts.MinConfidence != 0.5 {
		t.Errorf("Expected MinConfidence 0.5, got %f", opts.MinConfidence)
	}

	if opts.ContextWindow != 4000 {
		t.Errorf("Expected ContextWindow 4000, got %d", opts.ContextWindow)
	}

	if opts.BatchSize != 10 {
		t.Errorf("Expected BatchSize 10, got %d", opts.BatchSize)
	}

	if opts.Temperature != 0.0 {
		t.Errorf("Expected Temperature 0.0, got %f", opts.Temperature)
	}
}

func TestMockExtractor_GetName(t *testing.T) {
	extractor := &MockExtractor{name: "mock"}

	if extractor.GetName() != "mock" {
		t.Errorf("Expected name 'mock', got '%s'", extractor.GetName())
	}
}

func TestMockExtractor_GetVersion(t *testing.T) {
	extractor := &MockExtractor{version: "1.0.0"}

	if extractor.GetVersion() != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", extractor.GetVersion())
	}
}

func TestMockExtractor_ExtractFromText(t *testing.T) {
	extractor := &MockExtractor{
		name:    "mock",
		version: "1.0.0",
	}

	text := "John Doe works at Acme Corporation in New York. The Company was founded in 2020."

	opts := DefaultExtractionOptions()
	opts.DocID = "doc1"
	opts.DocName = "Test Document"

	ctx := context.Background()
	ontology, err := extractor.ExtractFromText(ctx, text, opts)

	if err != nil {
		t.Fatalf("ExtractFromText() error = %v", err)
	}

	if ontology.DocID != "doc1" {
		t.Errorf("Expected doc_id 'doc1', got '%s'", ontology.DocID)
	}

	if len(ontology.Entities) == 0 {
		t.Error("Expected at least some entities to be extracted")
	}

	t.Logf("Extracted %d entities, %d relationships, %d classes",
		len(ontology.Entities), len(ontology.Relationships), len(ontology.Classes))
}

func TestMockExtractor_ExtractFromElements(t *testing.T) {
	extractor := &MockExtractor{
		name:    "mock",
		version: "1.0.0",
	}

	elements := []Element{
		{
			ElementID:   "elem1",
			ElementType: "paragraph",
			Content:     "Apple Inc. announced new products. Tim Cook presented the keynote.",
		},
		{
			ElementID:   "elem2",
			ElementType: "paragraph",
			Content:     "The Event took place in California.",
		},
	}

	opts := DefaultExtractionOptions()
	opts.DocID = "doc1"

	ctx := context.Background()
	ontology, err := extractor.ExtractFromElements(ctx, elements, opts)

	if err != nil {
		t.Fatalf("ExtractFromElements() error = %v", err)
	}

	if ontology.Metadata.ElementCount != 2 {
		t.Errorf("Expected element count 2, got %d", ontology.Metadata.ElementCount)
	}

	if len(ontology.Entities) == 0 {
		t.Error("Expected at least some entities to be extracted")
	}

	t.Logf("Extracted %d entities from %d elements",
		len(ontology.Entities), len(elements))
}

func TestMockExtractor_SupportsFeature(t *testing.T) {
	extractor := &MockExtractor{}

	features := map[string]bool{
		"entity_extraction":       true,
		"relationship_extraction": true,
		"class_extraction":        true,
		"batch_processing":        true,
		"streaming":               false,
		"unknown_feature":         false,
	}

	for feature, expected := range features {
		result := extractor.SupportsFeature(feature)
		if result != expected {
			t.Errorf("SupportsFeature(%s) = %v, expected %v", feature, result, expected)
		}
	}
}

func TestMockExtractor_Close(t *testing.T) {
	extractor := &MockExtractor{}

	err := extractor.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestMockExtractor_ExtractionOptions(t *testing.T) {
	extractor := &MockExtractor{
		name:    "mock",
		version: "1.0.0",
	}

	text := "Alice works with Bob at Corporation. Charlie is in London. David founded Company in 2020."

	t.Run("max entities limit", func(t *testing.T) {
		opts := DefaultExtractionOptions()
		opts.DocID = "doc1"
		opts.MaxEntities = 2

		ctx := context.Background()
		ontology, _ := extractor.ExtractFromText(ctx, text, opts)

		if len(ontology.Entities) > 2 {
			t.Errorf("Expected at most 2 entities, got %d", len(ontology.Entities))
		}
	})

	t.Run("entity type filter", func(t *testing.T) {
		opts := DefaultExtractionOptions()
		opts.DocID = "doc1"
		opts.EntityTypes = []EntityType{EntityTypePerson}

		ctx := context.Background()
		ontology, _ := extractor.ExtractFromText(ctx, text, opts)

		for _, entity := range ontology.Entities {
			if entity.Type != EntityTypePerson {
				t.Errorf("Expected only person entities, got %s", entity.Type)
			}
		}
	})

	t.Run("max relationships limit", func(t *testing.T) {
		opts := DefaultExtractionOptions()
		opts.DocID = "doc1"
		opts.MaxRelationships = 1

		ctx := context.Background()
		ontology, _ := extractor.ExtractFromText(ctx, text, opts)

		if len(ontology.Relationships) > 1 {
			t.Errorf("Expected at most 1 relationship, got %d", len(ontology.Relationships))
		}
	})

	t.Run("extract classes disabled", func(t *testing.T) {
		opts := DefaultExtractionOptions()
		opts.DocID = "doc1"
		opts.ExtractClasses = false

		ctx := context.Background()
		ontology, _ := extractor.ExtractFromText(ctx, text, opts)

		if len(ontology.Classes) > 0 {
			t.Errorf("Expected no classes when disabled, got %d", len(ontology.Classes))
		}
	})
}

func TestMergeOntologies_Empty(t *testing.T) {
	_, err := MergeOntologies()
	if err == nil {
		t.Error("Expected error when merging no ontologies")
	}
}

func TestMergeOntologies_Single(t *testing.T) {
	ont1 := &Ontology{
		ID:            "ont1",
		Entities:      []Entity{{ID: "ent1", Name: "Entity1"}},
		Relationships: []Relationship{},
		Classes:       []Class{},
	}

	merged, err := MergeOntologies(ont1)
	if err != nil {
		t.Fatalf("MergeOntologies() error = %v", err)
	}

	if merged.ID != ont1.ID {
		t.Error("Expected merged ontology to have same ID as input")
	}
}

func TestMergeOntologies_Multiple(t *testing.T) {
	ont1 := &Ontology{
		ID:       "ont1",
		DocID:    "doc1",
		Name:     "Ontology 1",
		Entities: []Entity{{ID: "ent1", Name: "Entity1"}},
		Relationships: []Relationship{
			{ID: "rel1", SourceID: "ent1", TargetID: "ent2"},
		},
		Classes:   []Class{{ID: "cls1", Name: "Class1"}},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Metadata: OntologyMeta{
			ElementCount:     5,
			PromptTokens:     100,
			CompletionTokens: 50,
		},
	}

	ont2 := &Ontology{
		ID:       "ont2",
		DocID:    "doc1",
		Entities: []Entity{{ID: "ent2", Name: "Entity2"}},
		Relationships: []Relationship{
			{ID: "rel2", SourceID: "ent2", TargetID: "ent3"},
		},
		Classes:   []Class{{ID: "cls2", Name: "Class2"}},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Metadata: OntologyMeta{
			ElementCount:     3,
			PromptTokens:     80,
			CompletionTokens: 40,
		},
	}

	merged, err := MergeOntologies(ont1, ont2)
	if err != nil {
		t.Fatalf("MergeOntologies() error = %v", err)
	}

	if len(merged.Entities) != 2 {
		t.Errorf("Expected 2 entities, got %d", len(merged.Entities))
	}

	if len(merged.Relationships) != 2 {
		t.Errorf("Expected 2 relationships, got %d", len(merged.Relationships))
	}

	if len(merged.Classes) != 2 {
		t.Errorf("Expected 2 classes, got %d", len(merged.Classes))
	}

	if merged.Metadata.ElementCount != 8 {
		t.Errorf("Expected merged element count 8, got %d", merged.Metadata.ElementCount)
	}

	if merged.Metadata.PromptTokens != 180 {
		t.Errorf("Expected merged prompt tokens 180, got %d", merged.Metadata.PromptTokens)
	}
}

func TestMergeOntologies_Deduplication(t *testing.T) {
	ont1 := &Ontology{
		ID:       "ont1",
		Entities: []Entity{{ID: "ent1", Name: "Entity1"}},
	}

	ont2 := &Ontology{
		ID: "ont2",
		Entities: []Entity{
			{ID: "ent1", Name: "Entity1"}, // Duplicate
			{ID: "ent2", Name: "Entity2"},
		},
	}

	merged, err := MergeOntologies(ont1, ont2)
	if err != nil {
		t.Fatalf("MergeOntologies() error = %v", err)
	}

	// Should deduplicate ent1
	if len(merged.Entities) != 2 {
		t.Errorf("Expected 2 entities after deduplication, got %d", len(merged.Entities))
	}
}

func TestFilterOntology(t *testing.T) {
	ont := &Ontology{
		ID:    "ont1",
		DocID: "doc1",
		Entities: []Entity{
			{ID: "ent1", Name: "John", Type: EntityTypePerson, Confidence: 0.9},
			{ID: "ent2", Name: "Acme", Type: EntityTypeOrganization, Confidence: 0.8},
			{ID: "ent3", Name: "London", Type: EntityTypeLocation, Confidence: 0.6},
		},
		Relationships: []Relationship{
			{ID: "rel1", SourceID: "ent1", TargetID: "ent2", Type: RelationshipPartOf, Confidence: 0.85},
			{ID: "rel2", SourceID: "ent2", TargetID: "ent3", Type: RelationshipLocatedIn, Confidence: 0.7},
		},
		Classes: []Class{
			{ID: "cls1", Name: "Person"},
		},
	}

	t.Run("filter by confidence", func(t *testing.T) {
		filter := OntologyFilter{
			MinConfidence: 0.7,
		}

		filtered := FilterOntology(ont, filter)

		// Should filter out ent3 (0.6 confidence)
		if len(filtered.Entities) != 2 {
			t.Errorf("Expected 2 entities with confidence >= 0.7, got %d", len(filtered.Entities))
		}

		// rel2 should also be filtered out because its target (ent3) was removed
		if len(filtered.Relationships) != 1 {
			t.Errorf("Expected 1 relationship, got %d", len(filtered.Relationships))
		}
	})

	t.Run("filter by entity type", func(t *testing.T) {
		filter := OntologyFilter{
			EntityTypes: []EntityType{EntityTypePerson, EntityTypeOrganization},
		}

		filtered := FilterOntology(ont, filter)

		// Should include ent1 and ent2, exclude ent3
		if len(filtered.Entities) != 2 {
			t.Errorf("Expected 2 entities, got %d", len(filtered.Entities))
		}

		for _, entity := range filtered.Entities {
			if entity.Type == EntityTypeLocation {
				t.Error("Location entity should be filtered out")
			}
		}
	})

	t.Run("filter by relationship type", func(t *testing.T) {
		filter := OntologyFilter{
			RelationshipTypes: []RelationshipType{RelationshipPartOf},
		}

		filtered := FilterOntology(ont, filter)

		// Should include rel1, exclude rel2
		if len(filtered.Relationships) != 1 {
			t.Errorf("Expected 1 relationship, got %d", len(filtered.Relationships))
		}

		if filtered.Relationships[0].Type != RelationshipPartOf {
			t.Error("Expected only part_of relationships")
		}
	})
}

func TestOntologyFilter_AcceptEntity(t *testing.T) {
	filter := OntologyFilter{
		MinConfidence: 0.7,
		EntityTypes:   []EntityType{EntityTypePerson},
	}

	tests := []struct {
		name     string
		entity   Entity
		expected bool
	}{
		{
			name:     "accepts valid entity",
			entity:   Entity{Type: EntityTypePerson, Confidence: 0.9},
			expected: true,
		},
		{
			name:     "rejects low confidence",
			entity:   Entity{Type: EntityTypePerson, Confidence: 0.5},
			expected: false,
		},
		{
			name:     "rejects wrong type",
			entity:   Entity{Type: EntityTypeOrganization, Confidence: 0.9},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filter.AcceptEntity(tt.entity)
			if result != tt.expected {
				t.Errorf("AcceptEntity() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestOntologyFilter_AcceptRelationship(t *testing.T) {
	filter := OntologyFilter{
		MinConfidence:     0.7,
		RelationshipTypes: []RelationshipType{RelationshipPartOf},
	}

	tests := []struct {
		name     string
		rel      Relationship
		expected bool
	}{
		{
			name:     "accepts valid relationship",
			rel:      Relationship{Type: RelationshipPartOf, Confidence: 0.9},
			expected: true,
		},
		{
			name:     "rejects low confidence",
			rel:      Relationship{Type: RelationshipPartOf, Confidence: 0.5},
			expected: false,
		},
		{
			name:     "rejects wrong type",
			rel:      Relationship{Type: RelationshipIsA, Confidence: 0.9},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filter.AcceptRelationship(tt.rel)
			if result != tt.expected {
				t.Errorf("AcceptRelationship() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestGlobalRegistry(t *testing.T) {
	// Test that mock extractor was registered via init()
	names := globalRegistry.GetRegisteredExtractors()

	found := false
	for _, name := range names {
		if name == "mock" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Mock extractor should be registered in global registry")
	}

	// Test CreateExtractor with global registry
	config := ExtractorConfig{
		Type: "mock",
	}

	extractor, err := CreateExtractor(config)
	if err != nil {
		t.Fatalf("CreateExtractor() error = %v", err)
	}

	if extractor.GetName() != "mock" {
		t.Errorf("Expected mock extractor, got %s", extractor.GetName())
	}
}
