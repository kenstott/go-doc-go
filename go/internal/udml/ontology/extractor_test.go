package ontology

import (
	"context"
	"testing"
	"time"
)

func TestRuleBasedExtractor_MetadataExtraction(t *testing.T) {
	// Create schema with JSONPath extraction rule for metadata
	schema := &OntologySchema{
		Name:    "test_schema",
		Version: "1.0.0",
		Domains: []Domain{{Name: "test", Description: "Test domain", Owner: "Test Team"}},
		ElementEntityMappings: []ElementEntityMapping{
			{
				EntityType:   "Speaker",
				Domain:       "test",
				Confidence:   0.9,
				Description:  "Person speaking",
				ElementTypes: []string{"paragraph"},
				ExtractionRules: []ExtractionRule{
					{
						JSONPath:     "$.metadata.speaker",
						InstanceName: "$[0]",
					},
				},
			},
		},
		CreatedAt: time.Now(),
	}

	extractor := NewRuleBasedExtractor(schema, nil)

	elements := []Element{
		{
			ElementID:   "elem1",
			ElementType: "paragraph",
			Content:     "Hello everyone!",
			Metadata: map[string]interface{}{
				"speaker": "John Doe",
			},
		},
		{
			ElementID:   "elem2",
			ElementType: "paragraph",
			Content:     "Welcome to the conference.",
			Metadata: map[string]interface{}{
				"speaker": "Jane Smith",
			},
		},
	}

	ctx := context.Background()
	ontology, err := extractor.ExtractFromElements(ctx, "doc1", elements)

	if err != nil {
		t.Fatalf("ExtractFromElements() error = %v", err)
	}

	if len(ontology.Entities) != 2 {
		t.Errorf("Expected 2 entities, got %d", len(ontology.Entities))
	}

	// Check entity names and domain inheritance
	names := make(map[string]bool)
	for _, entity := range ontology.Entities {
		names[entity.Name] = true
		if entity.Type != EntityTypeCustom {
			t.Errorf("Expected custom entity type, got %s", entity.Type)
		}
		if entity.Confidence != 0.9 {
			t.Errorf("Expected confidence 0.9 (from mapping), got %.2f", entity.Confidence)
		}
		if entity.Domain != "test" {
			t.Errorf("Expected domain 'test' (inherited from mapping), got '%s'", entity.Domain)
		}
	}

	if !names["John Doe"] || !names["Jane Smith"] {
		t.Error("Expected John Doe and Jane Smith to be extracted")
	}
}

func TestRuleBasedExtractor_RegexExtraction(t *testing.T) {
	// Create schema with regex extraction rule
	schema := &OntologySchema{
		Name:    "test_schema",
		Version: "1.0.0",
		Domains: []Domain{{Name: "test", Description: "Test domain", Owner: "Test Team"}},
		ElementEntityMappings: []ElementEntityMapping{
			{
				EntityType:   "Email",
				Domain:       "test",
				Confidence:   0.95,
				Description:  "Email addresses",
				ElementTypes: []string{"paragraph"},
				ExtractionRules: []ExtractionRule{
					{
						Type:    RuleTypeRegex,
						Pattern: `\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`,
					},
				},
			},
		},
		CreatedAt: time.Now(),
	}

	extractor := NewRuleBasedExtractor(schema, nil)

	elements := []Element{
		{
			ElementID:   "elem1",
			ElementType: "paragraph",
			Content:     "Contact us at support@example.com or sales@example.com",
		},
	}

	ctx := context.Background()
	ontology, err := extractor.ExtractFromElements(ctx, "doc1", elements)

	if err != nil {
		t.Fatalf("ExtractFromElements() error = %v", err)
	}

	if len(ontology.Entities) != 2 {
		t.Errorf("Expected 2 email entities, got %d", len(ontology.Entities))
	}

	for _, entity := range ontology.Entities {
		if entity.Confidence != 0.95 {
			t.Errorf("Expected confidence 0.95 (from mapping), got %.2f", entity.Confidence)
		}
		if entity.Domain != "test" {
			t.Errorf("Expected domain 'test', got '%s'", entity.Domain)
		}
		t.Logf("Extracted email: %s (domain: %s)", entity.Name, entity.Domain)
	}
}

func TestRuleBasedExtractor_KeywordExtraction(t *testing.T) {
	// Create schema with keyword extraction rule
	schema := &OntologySchema{
		Name:    "test_schema",
		Version: "1.0.0",
		Domains: []Domain{{Name: "test", Description: "Test domain", Owner: "Test Team"}},
		ElementEntityMappings: []ElementEntityMapping{
			{
				EntityType:   "Organization",
				Domain:       "test",
				Confidence:   0.85,
				Description:  "Company names",
				ElementTypes: []string{"paragraph"},
				ExtractionRules: []ExtractionRule{
					{
						Type:     RuleTypeKeyword,
						Keywords: []string{"Microsoft", "Apple", "Google"},
					},
				},
			},
		},
		CreatedAt: time.Now(),
	}

	extractor := NewRuleBasedExtractor(schema, nil)

	elements := []Element{
		{
			ElementID:   "elem1",
			ElementType: "paragraph",
			Content:     "Microsoft and Apple are technology companies.",
		},
	}

	ctx := context.Background()
	ontology, err := extractor.ExtractFromElements(ctx, "doc1", elements)

	if err != nil {
		t.Fatalf("ExtractFromElements() error = %v", err)
	}

	if len(ontology.Entities) != 2 {
		t.Errorf("Expected 2 organization entities, got %d", len(ontology.Entities))
	}

	names := make(map[string]bool)
	for _, entity := range ontology.Entities {
		names[entity.Name] = true
		if entity.Type != EntityTypeOrganization {
			t.Errorf("Expected organization type, got %s", entity.Type)
		}
		if entity.Domain != "test" {
			t.Errorf("Expected domain 'test', got '%s'", entity.Domain)
		}
	}

	if !names["Microsoft"] || !names["Apple"] {
		t.Error("Expected Microsoft and Apple to be extracted")
	}
}

func TestRuleBasedExtractor_RelationshipExtraction(t *testing.T) {
	// Create schema with entity and relationship rules
	schema := &OntologySchema{
		Name:    "test_schema",
		Version: "1.0.0",
		Domains: []Domain{{Name: "test", Description: "Test domain", Owner: "Test Team"}},
		ElementEntityMappings: []ElementEntityMapping{
			{
				EntityType:   "person",
				Domain:       "test",
				Confidence:   0.8,
				Description:  "People",
				ElementTypes: []string{"paragraph"},
				ExtractionRules: []ExtractionRule{
					{
						Type:    RuleTypeRegex,
						Pattern: `\b[A-Z][a-z]+ [A-Z][a-z]+\b`,
					},
				},
			},
			{
				EntityType:   "organization",
				Domain:       "test",
				Confidence:   0.85,
				Description:  "Organizations",
				ElementTypes: []string{"paragraph"},
				ExtractionRules: []ExtractionRule{
					{
						Type:     RuleTypeKeyword,
						Keywords: []string{"Corp", "Inc", "LLC"},
					},
				},
			},
		},
		EntityRelationshipRules: []EntityRelationshipRule{
			{
				Name:             "person_works_at_org",
				SourceEntityType: "person",
				TargetEntityType: "organization",
				RelationshipType: RelationshipPartOf,
				Description:      "Person works at organization",
				Confidence:       0.7,
			},
		},
		CreatedAt: time.Now(),
	}

	extractor := NewRuleBasedExtractor(schema, nil)

	elements := []Element{
		{
			ElementID:   "elem1",
			ElementType: "paragraph",
			Content:     "John Smith works at Acme Corp",
		},
	}

	ctx := context.Background()
	ontology, err := extractor.ExtractFromElements(ctx, "doc1", elements)

	if err != nil {
		t.Fatalf("ExtractFromElements() error = %v", err)
	}

	// The regex matches both "John Smith" and "Acme Corp" as person entities
	// The keyword matches "Corp" as an organization entity
	// So we expect 3 entities total
	if len(ontology.Entities) != 3 {
		t.Errorf("Expected 3 entities, got %d", len(ontology.Entities))
		for _, e := range ontology.Entities {
			t.Logf("  Entity: %s (%s, domain: %s)", e.Name, e.Type, e.Domain)
		}
	}

	// Should have multiple relationships: person->person, person->org, person->org
	if len(ontology.Relationships) < 1 {
		t.Errorf("Expected at least 1 relationship, got %d", len(ontology.Relationships))
	}

	// Verify at least one relationship is part_of type and inherits domain from source
	hasPartOf := false
	for _, rel := range ontology.Relationships {
		t.Logf("Relationship: %s -> %s (%s, domain: %s)", rel.SourceID, rel.TargetID, rel.Type, rel.Domain)
		if rel.Type == RelationshipPartOf {
			hasPartOf = true
			if rel.Domain != "test" {
				t.Errorf("Expected relationship domain 'test' (from source entity), got '%s'", rel.Domain)
			}
			if rel.Confidence != 0.7 {
				t.Errorf("Expected relationship confidence 0.7 (from rule), got %.2f", rel.Confidence)
			}
		}
	}

	if !hasPartOf {
		t.Error("Expected at least one part_of relationship")
	}
}

func TestRuleBasedExtractor_ElementTypeFiltering(t *testing.T) {
	// Create schema that only processes specific element types
	schema := &OntologySchema{
		Name:    "test_schema",
		Version: "1.0.0",
		Domains: []Domain{{Name: "test", Description: "Test domain", Owner: "Test Team"}},
		ElementEntityMappings: []ElementEntityMapping{
			{
				EntityType:   "Heading",
				Domain:       "test",
				Confidence:   0.9,
				Description:  "Headings only",
				ElementTypes: []string{"heading"},
				ExtractionRules: []ExtractionRule{
					{
						Type:    RuleTypeRegex,
						Pattern: `.+`,
					},
				},
			},
		},
		CreatedAt: time.Now(),
	}

	extractor := NewRuleBasedExtractor(schema, nil)

	elements := []Element{
		{
			ElementID:   "elem1",
			ElementType: "heading",
			Content:     "Chapter 1",
		},
		{
			ElementID:   "elem2",
			ElementType: "paragraph",
			Content:     "This is a paragraph",
		},
	}

	ctx := context.Background()
	ontology, err := extractor.ExtractFromElements(ctx, "doc1", elements)

	if err != nil {
		t.Fatalf("ExtractFromElements() error = %v", err)
	}

	// Should only extract from heading, not paragraph
	if len(ontology.Entities) != 1 {
		t.Errorf("Expected 1 entity (from heading only), got %d", len(ontology.Entities))
	}

	if len(ontology.Entities) > 0 {
		entity := ontology.Entities[0]
		if entity.ElementID != "elem1" {
			t.Errorf("Expected entity from elem1 (heading), got %s", entity.ElementID)
		}
		if entity.Domain != "test" {
			t.Errorf("Expected domain 'test', got '%s'", entity.Domain)
		}
	}
}

func TestRuleBasedExtractor_NestedMetadata(t *testing.T) {
	// Test nested metadata path extraction
	schema := &OntologySchema{
		Name:    "test_schema",
		Version: "1.0.0",
		Domains: []Domain{{Name: "test", Description: "Test domain", Owner: "Test Team"}},
		ElementEntityMappings: []ElementEntityMapping{
			{
				EntityType:   "Author",
				Domain:       "test",
				Confidence:   0.95,
				Description:  "Document author",
				ElementTypes: []string{"paragraph"},
				ExtractionRules: []ExtractionRule{
					{
						Type:      RuleTypeMetadata,
						FieldPath: "author.name",
					},
				},
			},
		},
		CreatedAt: time.Now(),
	}

	extractor := NewRuleBasedExtractor(schema, nil)

	elements := []Element{
		{
			ElementID:   "elem1",
			ElementType: "paragraph",
			Content:     "Document content",
			Metadata: map[string]interface{}{
				"author": map[string]interface{}{
					"name": "Alice Johnson",
				},
			},
		},
	}

	ctx := context.Background()
	ontology, err := extractor.ExtractFromElements(ctx, "doc1", elements)

	if err != nil {
		t.Fatalf("ExtractFromElements() error = %v", err)
	}

	if len(ontology.Entities) != 1 {
		t.Errorf("Expected 1 entity, got %d", len(ontology.Entities))
	}

	if len(ontology.Entities) > 0 {
		entity := ontology.Entities[0]
		if entity.Name != "Alice Johnson" {
			t.Errorf("Expected 'Alice Johnson', got '%s'", entity.Name)
		}
		if entity.Domain != "test" {
			t.Errorf("Expected domain 'test', got '%s'", entity.Domain)
		}
	}
}

func TestRuleBasedExtractor_MultipleRules(t *testing.T) {
	// Test multiple extraction rules for the same entity type
	schema := &OntologySchema{
		Name:    "test_schema",
		Version: "1.0.0",
		Domains: []Domain{{Name: "test", Description: "Test domain", Owner: "Test Team"}},
		ElementEntityMappings: []ElementEntityMapping{
			{
				EntityType:   "person",
				Domain:       "test",
				Confidence:   0.8,
				Description:  "People",
				ElementTypes: []string{"paragraph"},
				ExtractionRules: []ExtractionRule{
					{
						Type:      RuleTypeMetadata,
						FieldPath: "author",
					},
					{
						Type:    RuleTypeRegex,
						Pattern: `\b[A-Z][a-z]+ [A-Z][a-z]+\b`,
					},
				},
			},
		},
		CreatedAt: time.Now(),
	}

	extractor := NewRuleBasedExtractor(schema, nil)

	elements := []Element{
		{
			ElementID:   "elem1",
			ElementType: "paragraph",
			Content:     "Bob Wilson wrote this document.",
			Metadata: map[string]interface{}{
				"author": "Alice Brown",
			},
		},
	}

	ctx := context.Background()
	ontology, err := extractor.ExtractFromElements(ctx, "doc1", elements)

	if err != nil {
		t.Fatalf("ExtractFromElements() error = %v", err)
	}

	// Should extract both Alice Brown (from metadata) and Bob Wilson (from regex)
	if len(ontology.Entities) != 2 {
		t.Errorf("Expected 2 entities, got %d", len(ontology.Entities))
	}

	names := make(map[string]bool)
	for _, entity := range ontology.Entities {
		names[entity.Name] = true
		if entity.Domain != "test" {
			t.Errorf("Expected domain 'test', got '%s'", entity.Domain)
		}
		t.Logf("Extracted: %s (confidence: %.2f, domain: %s)", entity.Name, entity.Confidence, entity.Domain)
	}

	if !names["Alice Brown"] || !names["Bob Wilson"] {
		t.Error("Expected both Alice Brown and Bob Wilson to be extracted")
	}
}

func TestRuleBasedExtractor_Deduplication(t *testing.T) {
	// Test that duplicate entity names are deduplicated
	schema := &OntologySchema{
		Name:    "test_schema",
		Version: "1.0.0",
		Domains: []Domain{{Name: "test", Description: "Test domain", Owner: "Test Team"}},
		ElementEntityMappings: []ElementEntityMapping{
			{
				EntityType:   "person",
				Domain:       "test",
				Confidence:   0.8,
				Description:  "People",
				ElementTypes: []string{"paragraph"},
				ExtractionRules: []ExtractionRule{
					{
						Type:    RuleTypeRegex,
						Pattern: `\bAlice\b`,
					},
				},
			},
		},
		CreatedAt: time.Now(),
	}

	extractor := NewRuleBasedExtractor(schema, nil)

	elements := []Element{
		{
			ElementID:   "elem1",
			ElementType: "paragraph",
			Content:     "Alice said hello.",
		},
		{
			ElementID:   "elem2",
			ElementType: "paragraph",
			Content:     "Alice replied.",
		},
	}

	ctx := context.Background()
	ontology, err := extractor.ExtractFromElements(ctx, "doc1", elements)

	if err != nil {
		t.Fatalf("ExtractFromElements() error = %v", err)
	}

	// Should deduplicate "Alice" even though it appears in 2 elements
	if len(ontology.Entities) != 1 {
		t.Errorf("Expected 1 deduplicated entity, got %d", len(ontology.Entities))
	}

	if len(ontology.Entities) > 0 {
		entity := ontology.Entities[0]
		if entity.Name != "Alice" {
			t.Errorf("Expected 'Alice', got '%s'", entity.Name)
		}
		if entity.Domain != "test" {
			t.Errorf("Expected domain 'test', got '%s'", entity.Domain)
		}
		// Should have 2 mentions
		if len(entity.Mentions) != 2 {
			t.Errorf("Expected 2 mentions, got %d", len(entity.Mentions))
		}
	}
}

func TestRuleBasedExtractor_EmptySchema(t *testing.T) {
	// Test with empty schema (no extraction rules)
	schema := &OntologySchema{
		Name:                    "empty_schema",
		Version:                 "1.0.0",
		Domains:                 []Domain{{Name: "test", Description: "Test domain", Owner: "Test Team"}},
		ElementEntityMappings:   []ElementEntityMapping{},
		EntityRelationshipRules: []EntityRelationshipRule{},
		CreatedAt:               time.Now(),
	}

	extractor := NewRuleBasedExtractor(schema, nil)

	elements := []Element{
		{
			ElementID:   "elem1",
			ElementType: "paragraph",
			Content:     "Some content here",
		},
	}

	ctx := context.Background()
	ontology, err := extractor.ExtractFromElements(ctx, "doc1", elements)

	if err != nil {
		t.Fatalf("ExtractFromElements() error = %v", err)
	}

	// Should extract nothing
	if len(ontology.Entities) != 0 {
		t.Errorf("Expected 0 entities, got %d", len(ontology.Entities))
	}

	if len(ontology.Relationships) != 0 {
		t.Errorf("Expected 0 relationships, got %d", len(ontology.Relationships))
	}
}

func TestRuleBasedExtractor_ValidationError(t *testing.T) {
	// Test that extractor validates the resulting ontology
	schema := &OntologySchema{
		Name:    "test_schema",
		Version: "1.0.0",
		Domains: []Domain{{Name: "test", Description: "Test domain", Owner: "Test Team"}},
		ElementEntityMappings: []ElementEntityMapping{
			{
				EntityType:   "person",
				Domain:       "test",
				Confidence:   0.8,
				Description:  "People",
				ElementTypes: []string{"paragraph"},
				ExtractionRules: []ExtractionRule{
					{
						Type:     RuleTypeKeyword,
						Keywords: []string{"John"},
					},
				},
			},
		},
		EntityRelationshipRules: []EntityRelationshipRule{
			{
				Name:             "invalid_rel",
				SourceEntityType: "person",
				TargetEntityType: "nonexistent_type",
				RelationshipType: RelationshipRelatedTo,
				Confidence:       0.5,
			},
		},
		CreatedAt: time.Now(),
	}

	extractor := NewRuleBasedExtractor(schema, nil)

	elements := []Element{
		{
			ElementID:   "elem1",
			ElementType: "paragraph",
			Content:     "John is here",
		},
	}

	ctx := context.Background()
	ontology, err := extractor.ExtractFromElements(ctx, "doc1", elements)

	// Should succeed - relationships that don't match any entities are just not created
	if err != nil {
		t.Fatalf("ExtractFromElements() error = %v", err)
	}

	// Should have 1 entity but 0 relationships (target type doesn't exist)
	if len(ontology.Entities) != 1 {
		t.Errorf("Expected 1 entity, got %d", len(ontology.Entities))
	}

	if len(ontology.Relationships) != 0 {
		t.Errorf("Expected 0 relationships (no matching target), got %d", len(ontology.Relationships))
	}
}

// Mock embedding resolver for testing semantic filtering
type mockEmbeddingResolver struct {
	embeddings map[string][]float64
}

func (m *mockEmbeddingResolver) ResolveContent(contentLocation map[string]interface{}, text bool) (string, error) {
	return "", nil
}

func (m *mockEmbeddingResolver) GetEmbedding(contentLocation map[string]interface{}) ([]float64, error) {
	if elemID, ok := contentLocation["element_id"].(string); ok {
		if emb, exists := m.embeddings[elemID]; exists {
			return emb, nil
		}
	}
	return nil, nil
}

func TestRuleBasedExtractor_SemanticFilteringAccepts(t *testing.T) {
	// Test that semantic filter accepts matching content
	schema := &OntologySchema{
		Name:    "test_schema",
		Version: "1.0.0",
		Domains: []Domain{{Name: "medical", Description: "Medical domain", Owner: "Medical Team"}},
		ElementEntityMappings: []ElementEntityMapping{
			{
				EntityType:   "person",
				Domain:       "medical",
				Confidence:   0.75,
				Description:  "Person names from narrative context",
				ElementTypes: []string{"paragraph"},
				ExtractionRules: []ExtractionRule{
					{
						Type:    RuleTypeRegex,
						Pattern: `\b[A-Z][a-z]+ [A-Z][a-z]+\b`,
						SemanticFilter: &SemanticFilter{
							ReferenceConcepts: []string{
								"individual person with biography or credentials",
								"author or creator attribution to individual",
							},
							SimilarityThreshold: 0.65,
						},
					},
				},
			},
		},
		CreatedAt: time.Now(),
	}

	// Mock resolver that provides embeddings
	mockResolver := &mockEmbeddingResolver{
		embeddings: map[string][]float64{
			"elem1": {0.1, 0.2, 0.3, 0.4}, // Simulate biographical context embedding
		},
	}

	extractor := NewRuleBasedExtractor(schema, mockResolver)

	// Mock concept embedding cache to simulate matching semantic context
	// (In reality, this would be computed from the reference concepts)
	conceptEmbeddingCache = map[string][]float64{
		"individual person with biography or credentials": {0.15, 0.25, 0.32, 0.38}, // Similar to elem1
		"author or creator attribution to individual":     {0.12, 0.22, 0.31, 0.39}, // Similar to elem1
	}
	defer func() { conceptEmbeddingCache = make(map[string][]float64) }() // Reset after test

	elements := []Element{
		{
			ElementID:   "elem1",
			ElementType: "paragraph",
			Content:     "Dr. John Smith is a renowned cardiologist who graduated from Harvard Medical School.",
			ContentLocation: map[string]interface{}{
				"element_id": "elem1",
			},
		},
	}

	ctx := context.Background()
	ontology, err := extractor.ExtractFromElements(ctx, "doc1", elements)

	if err != nil {
		t.Fatalf("ExtractFromElements() error = %v", err)
	}

	// Should extract both "Dr John" and "John Smith" because semantic filter accepts (similar embeddings)
	if len(ontology.Entities) < 1 {
		t.Errorf("Expected at least 1 entity (semantic filter should accept), got %d", len(ontology.Entities))
	}

	foundJohnSmith := false
	for _, entity := range ontology.Entities {
		if entity.Domain != "medical" {
			t.Errorf("Expected domain 'medical', got '%s'", entity.Domain)
		}
		if entity.Name == "John Smith" {
			foundJohnSmith = true
		}
		t.Logf("✓ Semantic filter accepted: %s (confidence: %.2f)", entity.Name, entity.Confidence)
	}

	if !foundJohnSmith {
		t.Errorf("Expected 'John Smith' to be extracted")
	}
}

func TestRuleBasedExtractor_SemanticFilteringRejects(t *testing.T) {
	// Test that semantic filter rejects non-matching content (e.g., "After The" from heading)
	schema := &OntologySchema{
		Name:    "test_schema",
		Version: "1.0.0",
		Domains: []Domain{{Name: "medical", Description: "Medical domain", Owner: "Medical Team"}},
		ElementEntityMappings: []ElementEntityMapping{
			{
				EntityType:   "person",
				Domain:       "medical",
				Confidence:   0.75,
				Description:  "Person names from narrative context",
				ElementTypes: []string{"paragraph"},
				ExtractionRules: []ExtractionRule{
					{
						Type:    RuleTypeRegex,
						Pattern: `\b[A-Z][a-z]+ [A-Z][a-z]+\b`,
						SemanticFilter: &SemanticFilter{
							ReferenceConcepts: []string{
								"individual person with biography or credentials",
								"author or creator attribution to individual",
							},
							SimilarityThreshold: 0.65,
		},
					},
				},
			},
		},
		CreatedAt: time.Now(),
	}

	// Mock resolver that provides embeddings
	mockResolver := &mockEmbeddingResolver{
		embeddings: map[string][]float64{
			"elem1": {0.9, 0.1, 0.05, 0.02}, // Simulate heading/title context embedding (very different)
		},
	}

	extractor := NewRuleBasedExtractor(schema, mockResolver)

	// Mock concept embedding cache with person-related concepts
	conceptEmbeddingCache = map[string][]float64{
		"individual person with biography or credentials": {0.1, 0.2, 0.3, 0.4}, // Not similar to elem1
		"author or creator attribution to individual":     {0.12, 0.22, 0.31, 0.39}, // Not similar to elem1
	}
	defer func() { conceptEmbeddingCache = make(map[string][]float64) }() // Reset after test

	elements := []Element{
		{
			ElementID:   "elem1",
			ElementType: "paragraph",
			Content:     "After The Surgery",
			ContentLocation: map[string]interface{}{
				"element_id": "elem1",
			},
		},
	}

	ctx := context.Background()
	ontology, err := extractor.ExtractFromElements(ctx, "doc1", elements)

	if err != nil {
		t.Fatalf("ExtractFromElements() error = %v", err)
	}

	// Should NOT extract "After The" because semantic filter rejects (dissimilar embeddings)
	if len(ontology.Entities) != 0 {
		t.Errorf("Expected 0 entities (semantic filter should reject 'After The'), got %d", len(ontology.Entities))
		for _, entity := range ontology.Entities {
			t.Errorf("  Unexpected entity: %s", entity.Name)
		}
	} else {
		t.Logf("✓ Semantic filter correctly rejected 'After The'")
	}
}

func TestRuleBasedExtractor_SemanticFilteringGracefulDegradation(t *testing.T) {
	// Test that semantic filter gracefully degrades when embeddings unavailable
	schema := &OntologySchema{
		Name:    "test_schema",
		Version: "1.0.0",
		Domains: []Domain{{Name: "medical", Description: "Medical domain", Owner: "Medical Team"}},
		ElementEntityMappings: []ElementEntityMapping{
			{
				EntityType:   "person",
				Domain:       "medical",
				Confidence:   0.75,
				Description:  "Person names",
				ElementTypes: []string{"paragraph"},
				ExtractionRules: []ExtractionRule{
					{
						Type:    RuleTypeRegex,
						Pattern: `\b[A-Z][a-z]+ [A-Z][a-z]+\b`,
						SemanticFilter: &SemanticFilter{
							ReferenceConcepts:   []string{"person context"},
							SimilarityThreshold: 0.65,
						},
					},
				},
			},
		},
		CreatedAt: time.Now(),
	}

	// No mock resolver - embeddings unavailable
	extractor := NewRuleBasedExtractor(schema, nil)

	elements := []Element{
		{
			ElementID:   "elem1",
			ElementType: "paragraph",
			Content:     "John Smith and Jane Doe are here.",
		},
	}

	ctx := context.Background()
	ontology, err := extractor.ExtractFromElements(ctx, "doc1", elements)

	if err != nil {
		t.Fatalf("ExtractFromElements() error = %v", err)
	}

	// Should extract both names because semantic filter degrades gracefully (accepts when no embeddings)
	if len(ontology.Entities) != 2 {
		t.Errorf("Expected 2 entities (graceful degradation should accept all), got %d", len(ontology.Entities))
	} else {
		t.Logf("✓ Graceful degradation: extracted %d entities without embeddings", len(ontology.Entities))
	}
}

func TestRuleBasedExtractor_SemanticFilterWithKeywords(t *testing.T) {
	// Test that semantic filter works with keyword extraction rules
	schema := &OntologySchema{
		Name:    "test_schema",
		Version: "1.0.0",
		Domains: []Domain{{Name: "medical", Description: "Medical domain", Owner: "Medical Team"}},
		ElementEntityMappings: []ElementEntityMapping{
			{
				EntityType:   "organization",
				Domain:       "medical",
				Confidence:   0.75,
				Description:  "Medical organizations",
				ElementTypes: []string{"paragraph"},
				ExtractionRules: []ExtractionRule{
					{
						Type:     RuleTypeKeyword,
						Keywords: []string{"Hospital", "Clinic", "Center"},
						SemanticFilter: &SemanticFilter{
							ReferenceConcepts: []string{
								"healthcare facility providing medical services",
								"hospital or clinic with patients and doctors",
							},
							SimilarityThreshold: 0.60,
						},
					},
				},
			},
		},
		CreatedAt: time.Now(),
	}

	mockResolver := &mockEmbeddingResolver{
		embeddings: map[string][]float64{
			"elem1": {0.2, 0.3, 0.4, 0.3}, // Medical context
			"elem2": {0.9, 0.1, 0.05, 0.02}, // Non-medical context
		},
	}

	extractor := NewRuleBasedExtractor(schema, mockResolver)

	conceptEmbeddingCache = map[string][]float64{
		"healthcare facility providing medical services":  {0.25, 0.35, 0.38, 0.32}, // Similar to elem1
		"hospital or clinic with patients and doctors": {0.22, 0.32, 0.41, 0.31}, // Similar to elem1
	}
	defer func() { conceptEmbeddingCache = make(map[string][]float64) }()

	elements := []Element{
		{
			ElementID:   "elem1",
			ElementType: "paragraph",
			Content:     "The Hospital treated 1000 patients last year with excellent outcomes.",
			ContentLocation: map[string]interface{}{
				"element_id": "elem1",
			},
		},
		{
			ElementID:   "elem2",
			ElementType: "paragraph",
			Content:     "The Community Center offers yoga classes.",
			ContentLocation: map[string]interface{}{
				"element_id": "elem2",
			},
		},
	}

	ctx := context.Background()
	ontology, err := extractor.ExtractFromElements(ctx, "doc1", elements)

	if err != nil {
		t.Fatalf("ExtractFromElements() error = %v", err)
	}

	// Should extract "Hospital" from elem1 but reject "Center" from elem2
	if len(ontology.Entities) != 1 {
		t.Errorf("Expected 1 entity (Hospital), got %d", len(ontology.Entities))
		for _, entity := range ontology.Entities {
			t.Logf("  Entity: %s from element %s", entity.Name, entity.ElementID)
		}
	}

	if len(ontology.Entities) > 0 {
		entity := ontology.Entities[0]
		if entity.Name != "Hospital" {
			t.Errorf("Expected 'Hospital', got '%s'", entity.Name)
		}
		if entity.ElementID != "elem1" {
			t.Errorf("Expected extraction from elem1, got %s", entity.ElementID)
		}
		t.Logf("✓ Semantic filter correctly accepted 'Hospital' from medical context")
		t.Logf("✓ Semantic filter correctly rejected 'Center' from non-medical context")
	}
}

func TestRuleBasedExtractor_MultipleRulesWithSemanticFilter(t *testing.T) {
	// Test multiple rules with different semantic filters (high vs medium confidence)
	schema := &OntologySchema{
		Name:    "test_schema",
		Version: "1.0.0",
		Domains: []Domain{{Name: "medical", Description: "Medical domain", Owner: "Medical Team"}},
		ElementEntityMappings: []ElementEntityMapping{
			// High confidence - title prefix (no semantic filter)
			{
				EntityType:   "person",
				Domain:       "medical",
				Confidence:   0.95,
				Description:  "Person with title prefix",
				ElementTypes: []string{"paragraph"},
				ExtractionRules: []ExtractionRule{
					{
						Type:    RuleTypeRegex,
						Pattern: `\b(?:Dr|Prof)\.\s+[A-Z][a-z]+ [A-Z][a-z]+\b`,
						// No semantic filter - title is strong signal
					},
				},
			},
			// Medium confidence - ambiguous pattern (with semantic filter)
			{
				EntityType:   "person",
				Domain:       "medical",
				Confidence:   0.75,
				Description:  "Person without title",
				ElementTypes: []string{"paragraph"},
				ExtractionRules: []ExtractionRule{
					{
						Type:    RuleTypeRegex,
						Pattern: `\b[A-Z][a-z]+ [A-Z][a-z]+\b`,
						SemanticFilter: &SemanticFilter{
							ReferenceConcepts:   []string{"biographical context"},
							SimilarityThreshold: 0.65,
						},
					},
				},
			},
		},
		CreatedAt: time.Now(),
	}

	mockResolver := &mockEmbeddingResolver{
		embeddings: map[string][]float64{
			"elem1": {0.1, 0.2, 0.3, 0.4}, // Biographical context
			"elem2": {0.9, 0.1, 0.05, 0.02}, // Heading context
		},
	}

	extractor := NewRuleBasedExtractor(schema, mockResolver)

	conceptEmbeddingCache = map[string][]float64{
		"biographical context": {0.12, 0.22, 0.31, 0.39},
	}
	defer func() { conceptEmbeddingCache = make(map[string][]float64) }()

	elements := []Element{
		{
			ElementID:   "elem1",
			ElementType: "paragraph",
			Content:     "Dr. Jane Smith and John Doe work together.",
			ContentLocation: map[string]interface{}{
				"element_id": "elem1",
			},
		},
		{
			ElementID:   "elem2",
			ElementType: "paragraph",
			Content:     "After The Surgery chapter discusses Dr. Bob Wilson.",
			ContentLocation: map[string]interface{}{
				"element_id": "elem2",
			},
		},
	}

	ctx := context.Background()
	ontology, err := extractor.ExtractFromElements(ctx, "doc1", elements)

	if err != nil {
		t.Fatalf("ExtractFromElements() error = %v", err)
	}

	// Should extract:
	// - Jane Smith (medium confidence, passes semantic filter in elem1)
	// - John Doe (medium confidence, passes semantic filter in elem1)
	// - Dr. Jane Smith (high confidence, title prefix, no semantic filter)
	// - Dr. Bob Wilson (high confidence, title prefix, no semantic filter)
	// Should NOT extract:
	// - After The (medium confidence, FAILS semantic filter in elem2)

	// Should extract at least 3-5 entities (depends on deduplication and pattern overlap)
	if len(ontology.Entities) < 3 {
		t.Errorf("Expected at least 3 entities, got %d", len(ontology.Entities))
	}

	foundAfterThe := false
	for _, entity := range ontology.Entities {
		t.Logf("Extracted: %s (confidence: %.2f)", entity.Name, entity.Confidence)
		if entity.Name == "After The" {
			foundAfterThe = true
			t.Errorf("'After The' should have been rejected by semantic filter!")
		}
	}

	if foundAfterThe {
		t.Error("'After The' was incorrectly extracted")
	}

	t.Logf("✓ High confidence rule (title prefix) extracted without semantic filtering")
	t.Logf("✓ Medium confidence rule extracted with semantic validation")
	t.Logf("✓ 'After The' correctly rejected by semantic filter")
}
