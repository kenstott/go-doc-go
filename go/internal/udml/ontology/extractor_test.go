package ontology

import (
	"context"
	"testing"
	"time"
)

func TestRuleBasedExtractor_MetadataExtraction(t *testing.T) {
	// Create schema with metadata extraction rule
	schema := &OntologySchema{
		Name:    "test_schema",
		Version: "1.0.0",
		Domain:  "test",
		ElementEntityMappings: []ElementEntityMapping{
			{
				EntityType:   "Speaker",
				Description:  "Person speaking",
				ElementTypes: []string{"paragraph"},
				ExtractionRules: []ExtractionRule{
					{
						Type:       RuleTypeMetadata,
						FieldPath:  "speaker",
						Confidence: 0.9,
					},
				},
			},
		},
		CreatedAt: time.Now(),
	}

	extractor := NewRuleBasedExtractor(schema)

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

	// Check entity names
	names := make(map[string]bool)
	for _, entity := range ontology.Entities {
		names[entity.Name] = true
		if entity.Type != EntityTypeCustom {
			t.Errorf("Expected custom entity type, got %s", entity.Type)
		}
		if entity.Confidence != 0.9 {
			t.Errorf("Expected confidence 0.9, got %.2f", entity.Confidence)
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
		Domain:  "test",
		ElementEntityMappings: []ElementEntityMapping{
			{
				EntityType:   "Email",
				Description:  "Email addresses",
				ElementTypes: []string{"paragraph"},
				ExtractionRules: []ExtractionRule{
					{
						Type:       RuleTypeRegex,
						Pattern:    `\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`,
						Confidence: 0.95,
					},
				},
			},
		},
		CreatedAt: time.Now(),
	}

	extractor := NewRuleBasedExtractor(schema)

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
			t.Errorf("Expected confidence 0.95, got %.2f", entity.Confidence)
		}
		t.Logf("Extracted email: %s", entity.Name)
	}
}

func TestRuleBasedExtractor_KeywordExtraction(t *testing.T) {
	// Create schema with keyword extraction rule
	schema := &OntologySchema{
		Name:    "test_schema",
		Version: "1.0.0",
		Domain:  "test",
		ElementEntityMappings: []ElementEntityMapping{
			{
				EntityType:   "Organization",
				Description:  "Company names",
				ElementTypes: []string{"paragraph"},
				ExtractionRules: []ExtractionRule{
					{
						Type:       RuleTypeKeyword,
						Keywords:   []string{"Microsoft", "Apple", "Google"},
						Confidence: 0.85,
					},
				},
			},
		},
		CreatedAt: time.Now(),
	}

	extractor := NewRuleBasedExtractor(schema)

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
		Domain:  "test",
		ElementEntityMappings: []ElementEntityMapping{
			{
				EntityType:   "person",
				Description:  "People",
				ElementTypes: []string{"paragraph"},
				ExtractionRules: []ExtractionRule{
					{
						Type:       RuleTypeRegex,
						Pattern:    `\b[A-Z][a-z]+ [A-Z][a-z]+\b`,
						Confidence: 0.8,
					},
				},
			},
			{
				EntityType:   "organization",
				Description:  "Organizations",
				ElementTypes: []string{"paragraph"},
				ExtractionRules: []ExtractionRule{
					{
						Type:       RuleTypeKeyword,
						Keywords:   []string{"Corp", "Inc", "LLC"},
						Confidence: 0.85,
					},
				},
			},
		},
		EntityRelationshipRules: []EntityRelationshipRule{
			{
				Name:                "person_works_at_org",
				SourceEntityType:    "person",
				TargetEntityType:    "organization",
				RelationshipType:    RelationshipPartOf,
				Description:         "Person works at organization",
				ConfidenceThreshold: 0.7,
			},
		},
		CreatedAt: time.Now(),
	}

	extractor := NewRuleBasedExtractor(schema)

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
			t.Logf("  Entity: %s (%s)", e.Name, e.Type)
		}
	}

	// Should have multiple relationships: person->person, person->org, person->org
	if len(ontology.Relationships) < 1 {
		t.Errorf("Expected at least 1 relationship, got %d", len(ontology.Relationships))
	}

	// Verify at least one relationship is part_of type
	hasPartOf := false
	for _, rel := range ontology.Relationships {
		t.Logf("Relationship: %s -> %s (%s)", rel.SourceID, rel.TargetID, rel.Type)
		if rel.Type == RelationshipPartOf {
			hasPartOf = true
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
		Domain:  "test",
		ElementEntityMappings: []ElementEntityMapping{
			{
				EntityType:   "Heading",
				Description:  "Headings only",
				ElementTypes: []string{"heading"},
				ExtractionRules: []ExtractionRule{
					{
						Type:       RuleTypeRegex,
						Pattern:    `.+`,
						Confidence: 0.9,
					},
				},
			},
		},
		CreatedAt: time.Now(),
	}

	extractor := NewRuleBasedExtractor(schema)

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
	}
}

func TestRuleBasedExtractor_NestedMetadata(t *testing.T) {
	// Test nested metadata path extraction
	schema := &OntologySchema{
		Name:    "test_schema",
		Version: "1.0.0",
		Domain:  "test",
		ElementEntityMappings: []ElementEntityMapping{
			{
				EntityType:   "Author",
				Description:  "Document author",
				ElementTypes: []string{"paragraph"},
				ExtractionRules: []ExtractionRule{
					{
						Type:       RuleTypeMetadata,
						FieldPath:  "author.name",
						Confidence: 0.95,
					},
				},
			},
		},
		CreatedAt: time.Now(),
	}

	extractor := NewRuleBasedExtractor(schema)

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
	}
}

func TestRuleBasedExtractor_MultipleRules(t *testing.T) {
	// Test multiple extraction rules for the same entity type
	schema := &OntologySchema{
		Name:    "test_schema",
		Version: "1.0.0",
		Domain:  "test",
		ElementEntityMappings: []ElementEntityMapping{
			{
				EntityType:   "person",
				Description:  "People",
				ElementTypes: []string{"paragraph"},
				ExtractionRules: []ExtractionRule{
					{
						Type:       RuleTypeMetadata,
						FieldPath:  "author",
						Confidence: 0.95,
					},
					{
						Type:       RuleTypeRegex,
						Pattern:    `\b[A-Z][a-z]+ [A-Z][a-z]+\b`,
						Confidence: 0.8,
					},
				},
			},
		},
		CreatedAt: time.Now(),
	}

	extractor := NewRuleBasedExtractor(schema)

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
		t.Logf("Extracted: %s (confidence: %.2f)", entity.Name, entity.Confidence)
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
		Domain:  "test",
		ElementEntityMappings: []ElementEntityMapping{
			{
				EntityType:   "person",
				Description:  "People",
				ElementTypes: []string{"paragraph"},
				ExtractionRules: []ExtractionRule{
					{
						Type:       RuleTypeRegex,
						Pattern:    `\bAlice\b`,
						Confidence: 0.8,
					},
				},
			},
		},
		CreatedAt: time.Now(),
	}

	extractor := NewRuleBasedExtractor(schema)

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
		Domain:                  "test",
		ElementEntityMappings:   []ElementEntityMapping{},
		EntityRelationshipRules: []EntityRelationshipRule{},
		CreatedAt:               time.Now(),
	}

	extractor := NewRuleBasedExtractor(schema)

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
		Domain:  "test",
		ElementEntityMappings: []ElementEntityMapping{
			{
				EntityType:   "person",
				Description:  "People",
				ElementTypes: []string{"paragraph"},
				ExtractionRules: []ExtractionRule{
					{
						Type:       RuleTypeKeyword,
						Keywords:   []string{"John"},
						Confidence: 0.8,
					},
				},
			},
		},
		EntityRelationshipRules: []EntityRelationshipRule{
			{
				Name:                "invalid_rel",
				SourceEntityType:    "person",
				TargetEntityType:    "nonexistent_type",
				RelationshipType:    RelationshipRelatedTo,
				ConfidenceThreshold: 0.5,
			},
		},
		CreatedAt: time.Now(),
	}

	extractor := NewRuleBasedExtractor(schema)

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
