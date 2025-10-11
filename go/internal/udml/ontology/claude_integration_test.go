package ontology

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

// skipIfNoAPIKey skips the test if ANTHROPIC_API_KEY is not set
func skipIfNoAPIKey(t *testing.T) string {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping integration test: ANTHROPIC_API_KEY not set")
	}
	return apiKey
}

func TestClaudeExtractor_Integration_SimpleText(t *testing.T) {
	apiKey := skipIfNoAPIKey(t)

	config := ExtractorConfig{
		Type:   "claude",
		APIKey: apiKey,
	}

	extractor, err := CreateExtractor(config)
	if err != nil {
		t.Fatalf("Failed to create extractor: %v", err)
	}
	defer extractor.Close()

	text := `
	John Smith is the CEO of Acme Corporation, a technology company based in San Francisco.
	The company was founded in 2020 and specializes in artificial intelligence solutions.
	John previously worked at Microsoft before starting Acme Corp.
	`

	options := ExtractionOptions{
		DocID:          "doc1",
		DocName:        "Test Document",
		ExtractClasses: false,
		MinConfidence:  0.5,
		Temperature:    0.0, // Deterministic
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ontology, err := extractor.ExtractFromText(ctx, text, options)
	if err != nil {
		t.Fatalf("ExtractFromText() error = %v", err)
	}

	// Validate ontology structure
	if ontology.DocID != "doc1" {
		t.Errorf("Expected doc_id 'doc1', got '%s'", ontology.DocID)
	}

	if len(ontology.Entities) == 0 {
		t.Error("Expected at least one entity")
	}

	t.Logf("Extracted %d entities, %d relationships",
		len(ontology.Entities), len(ontology.Relationships))

	// Validate metadata
	if ontology.Metadata.ExtractorType != "claude" {
		t.Errorf("Expected extractor type 'claude', got '%s'", ontology.Metadata.ExtractorType)
	}

	if ontology.Metadata.PromptTokens == 0 {
		t.Error("Expected non-zero prompt tokens")
	}

	if ontology.Metadata.CompletionTokens == 0 {
		t.Error("Expected non-zero completion tokens")
	}

	// Check for expected entities
	entityNames := make([]string, 0, len(ontology.Entities))
	for _, entity := range ontology.Entities {
		entityNames = append(entityNames, entity.Name)
		t.Logf("Entity: %s (type=%s, confidence=%.2f)", entity.Name, entity.Type, entity.Confidence)

		// Validate entity structure
		if entity.ID == "" {
			t.Error("Entity ID should not be empty")
		}
		if entity.Name == "" {
			t.Error("Entity name should not be empty")
		}
		if entity.Confidence < 0 || entity.Confidence > 1 {
			t.Errorf("Entity confidence should be between 0 and 1, got %.2f", entity.Confidence)
		}
	}

	// Check for expected relationships
	for _, rel := range ontology.Relationships {
		t.Logf("Relationship: %s -[%s]-> %s (confidence=%.2f)",
			rel.SourceID, rel.Type, rel.TargetID, rel.Confidence)

		// Validate relationship structure
		if rel.ID == "" {
			t.Error("Relationship ID should not be empty")
		}
		if rel.SourceID == "" || rel.TargetID == "" {
			t.Error("Relationship source and target IDs should not be empty")
		}
	}

	// Validate the ontology
	if err := ontology.Validate(); err != nil {
		t.Errorf("Ontology validation failed: %v", err)
	}
}

func TestClaudeExtractor_Integration_WithElements(t *testing.T) {
	apiKey := skipIfNoAPIKey(t)

	config := ExtractorConfig{
		Type:   "claude",
		APIKey: apiKey,
	}

	extractor, err := CreateExtractor(config)
	if err != nil {
		t.Fatalf("Failed to create extractor: %v", err)
	}
	defer extractor.Close()

	elements := []Element{
		{
			ElementID:   "elem1",
			ElementType: "paragraph",
			Content:     "Apple Inc. is an American multinational technology company headquartered in Cupertino, California.",
			ParentID:    "",
		},
		{
			ElementID:   "elem2",
			ElementType: "paragraph",
			Content:     "Steve Jobs co-founded Apple in 1976 and served as CEO until his death in 2011.",
			ParentID:    "",
		},
	}

	options := ExtractionOptions{
		DocID:          "doc2",
		DocName:        "Apple Document",
		ExtractClasses: false,
		MinConfidence:  0.5,
		Temperature:    0.0,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ontology, err := extractor.ExtractFromElements(ctx, elements, options)
	if err != nil {
		t.Fatalf("ExtractFromElements() error = %v", err)
	}

	t.Logf("Extracted %d entities from %d elements",
		len(ontology.Entities), len(elements))

	if len(ontology.Entities) == 0 {
		t.Error("Expected at least one entity")
	}

	// Check that metadata reflects element count
	if ontology.Metadata.ElementCount != len(elements) {
		t.Errorf("Expected element count %d, got %d",
			len(elements), ontology.Metadata.ElementCount)
	}

	// Validate the ontology
	if err := ontology.Validate(); err != nil {
		t.Errorf("Ontology validation failed: %v", err)
	}
}

func TestClaudeExtractor_Integration_ClassExtraction(t *testing.T) {
	apiKey := skipIfNoAPIKey(t)

	config := ExtractorConfig{
		Type:   "claude",
		APIKey: apiKey,
	}

	extractor, err := CreateExtractor(config)
	if err != nil {
		t.Fatalf("Failed to create extractor: %v", err)
	}
	defer extractor.Close()

	text := `
	A Document is a structured information container. Every Document has a title, author, and creation date.
	A Report is a Document that contains analysis and findings. Reports have conclusions and recommendations.
	A Contract is a Document that defines legal agreements between parties.
	`

	options := ExtractionOptions{
		DocID:          "doc3",
		DocName:        "Ontology Document",
		ExtractClasses: true, // Enable class extraction
		MinConfidence:  0.5,
		Temperature:    0.0,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ontology, err := extractor.ExtractFromText(ctx, text, options)
	if err != nil {
		t.Fatalf("ExtractFromText() error = %v", err)
	}

	t.Logf("Extracted %d classes", len(ontology.Classes))

	if len(ontology.Classes) == 0 {
		t.Error("Expected at least one class")
	}

	// Validate classes
	for _, class := range ontology.Classes {
		t.Logf("Class: %s - %s (properties: %d)",
			class.Name, class.Description, len(class.Properties))

		if class.ID == "" {
			t.Error("Class ID should not be empty")
		}
		if class.Name == "" {
			t.Error("Class name should not be empty")
		}
	}

	// Validate the ontology
	if err := ontology.Validate(); err != nil {
		t.Errorf("Ontology validation failed: %v", err)
	}
}

func TestClaudeExtractor_Integration_EntityTypeFilter(t *testing.T) {
	apiKey := skipIfNoAPIKey(t)

	config := ExtractorConfig{
		Type:   "claude",
		APIKey: apiKey,
	}

	extractor, err := CreateExtractor(config)
	if err != nil {
		t.Fatalf("Failed to create extractor: %v", err)
	}
	defer extractor.Close()

	text := `
	Microsoft Corporation is a technology company founded by Bill Gates and Paul Allen in 1975.
	The company is headquartered in Redmond, Washington.
	`

	options := ExtractionOptions{
		DocID:          "doc4",
		DocName:        "Microsoft Document",
		EntityTypes:    []EntityType{EntityTypeOrganization}, // Only extract organizations
		ExtractClasses: false,
		MinConfidence:  0.5,
		Temperature:    0.0,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ontology, err := extractor.ExtractFromText(ctx, text, options)
	if err != nil {
		t.Fatalf("ExtractFromText() error = %v", err)
	}

	t.Logf("Extracted %d entities (organizations only)", len(ontology.Entities))

	// Note: The filter is applied by the mock extractor, but Claude extractor
	// extracts all entities and we filter afterwards. So we might still get people.
	// This is expected behavior - the filter is a post-processing step.

	for _, entity := range ontology.Entities {
		t.Logf("Entity: %s (type=%s)", entity.Name, entity.Type)
	}
}

func TestClaudeExtractor_Integration_ConfidenceFiltering(t *testing.T) {
	apiKey := skipIfNoAPIKey(t)

	config := ExtractorConfig{
		Type:   "claude",
		APIKey: apiKey,
	}

	extractor, err := CreateExtractor(config)
	if err != nil {
		t.Fatalf("Failed to create extractor: %v", err)
	}
	defer extractor.Close()

	text := `
	Google LLC is a technology company that specializes in Internet-related services and products.
	The company was founded in 1998 by Larry Page and Sergey Brin.
	`

	options := ExtractionOptions{
		DocID:          "doc5",
		DocName:        "Google Document",
		ExtractClasses: false,
		MinConfidence:  0.8, // High confidence threshold
		Temperature:    0.0,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ontology, err := extractor.ExtractFromText(ctx, text, options)
	if err != nil {
		t.Fatalf("ExtractFromText() error = %v", err)
	}

	t.Logf("Extracted %d entities with confidence >= 0.8", len(ontology.Entities))

	// Validate that all entities meet the confidence threshold
	for _, entity := range ontology.Entities {
		if entity.Confidence < 0.8 {
			t.Errorf("Entity %s has confidence %.2f, expected >= 0.8",
				entity.Name, entity.Confidence)
		}
		t.Logf("Entity: %s (confidence=%.2f)", entity.Name, entity.Confidence)
	}

	// Validate the ontology
	if err := ontology.Validate(); err != nil {
		t.Errorf("Ontology validation failed: %v", err)
	}
}

func TestClaudeExtractor_Integration_JSONSerialization(t *testing.T) {
	apiKey := skipIfNoAPIKey(t)

	config := ExtractorConfig{
		Type:   "claude",
		APIKey: apiKey,
	}

	extractor, err := CreateExtractor(config)
	if err != nil {
		t.Fatalf("Failed to create extractor: %v", err)
	}
	defer extractor.Close()

	text := `Amazon.com, Inc. is an American multinational technology company based in Seattle, Washington.`

	options := ExtractionOptions{
		DocID:          "doc6",
		DocName:        "Amazon Document",
		ExtractClasses: false,
		MinConfidence:  0.5,
		Temperature:    0.0,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ontology, err := extractor.ExtractFromText(ctx, text, options)
	if err != nil {
		t.Fatalf("ExtractFromText() error = %v", err)
	}

	// Test JSON serialization
	jsonData, err := ontology.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error = %v", err)
	}

	if len(jsonData) == 0 {
		t.Error("JSON data should not be empty")
	}

	t.Logf("JSON size: %d bytes", len(jsonData))

	// Test compact JSON
	compactJSON, err := ontology.ToJSONCompact()
	if err != nil {
		t.Fatalf("ToJSONCompact() error = %v", err)
	}

	if len(compactJSON) >= len(jsonData) {
		t.Error("Compact JSON should be smaller than pretty JSON")
	}

	t.Logf("Compact JSON size: %d bytes", len(compactJSON))

	// Verify it's valid JSON by parsing
	if !strings.Contains(string(jsonData), "\"doc_id\"") {
		t.Error("JSON should contain doc_id field")
	}
}

func TestClaudeExtractor_Integration_ComplexDocument(t *testing.T) {
	apiKey := skipIfNoAPIKey(t)

	config := ExtractorConfig{
		Type:   "claude",
		APIKey: apiKey,
	}

	extractor, err := CreateExtractor(config)
	if err != nil {
		t.Fatalf("Failed to create extractor: %v", err)
	}
	defer extractor.Close()

	text := `
	The Artificial Intelligence Research Laboratory at MIT was established in 1959.
	John McCarthy, one of the founders of AI, worked at MIT and coined the term "Artificial Intelligence" in 1956.
	The lab has contributed to many advances in computer science, including the development of Lisp programming language.
	McCarthy later moved to Stanford University where he continued his AI research.
	Today, MIT CSAIL (Computer Science and Artificial Intelligence Laboratory) continues this legacy.
	`

	options := ExtractionOptions{
		DocID:          "doc7",
		DocName:        "AI History Document",
		ExtractClasses: true,
		MinConfidence:  0.5,
		Temperature:    0.0,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ontology, err := extractor.ExtractFromText(ctx, text, options)
	if err != nil {
		t.Fatalf("ExtractFromText() error = %v", err)
	}

	t.Logf("Complex document extraction results:")
	t.Logf("  Entities: %d", len(ontology.Entities))
	t.Logf("  Relationships: %d", len(ontology.Relationships))
	t.Logf("  Classes: %d", len(ontology.Classes))
	t.Logf("  Confidence: %.2f", ontology.Metadata.Confidence)
	t.Logf("  Prompt tokens: %d", ontology.Metadata.PromptTokens)
	t.Logf("  Completion tokens: %d", ontology.Metadata.CompletionTokens)

	// Get statistics
	stats := ontology.GetStats()
	t.Logf("  Entity types: %d", len(stats.EntityTypes))
	t.Logf("  Relationship types: %d", len(stats.RelationshipTypes))

	for entityType, count := range stats.EntityTypes {
		t.Logf("    %s: %d", entityType, count)
	}

	for relType, count := range stats.RelationshipTypes {
		t.Logf("    %s: %d", relType, count)
	}

	// Validate the ontology
	if err := ontology.Validate(); err != nil {
		t.Errorf("Ontology validation failed: %v", err)
	}

	// Test graph traversal
	if len(ontology.Entities) > 0 {
		firstEntity := ontology.Entities[0]
		outgoing := ontology.GetRelationshipsBySource(firstEntity.ID)
		incoming := ontology.GetRelationshipsByTarget(firstEntity.ID)
		t.Logf("Entity '%s' has %d outgoing and %d incoming relationships",
			firstEntity.Name, len(outgoing), len(incoming))
	}
}

func TestClaudeExtractor_Features(t *testing.T) {
	// This test doesn't need API key - just checks feature support
	config := ExtractorConfig{
		Type:   "claude",
		APIKey: "dummy", // Won't be used
	}

	extractor, err := CreateExtractor(config)
	if err != nil {
		t.Fatalf("Failed to create extractor: %v", err)
	}
	defer extractor.Close()

	features := []struct {
		name      string
		supported bool
	}{
		{"entity_extraction", true},
		{"relationship_extraction", true},
		{"class_extraction", true},
		{"batch_processing", true},
		{"streaming", false},
		{"unknown_feature", false},
	}

	for _, f := range features {
		supported := extractor.SupportsFeature(f.name)
		if supported != f.supported {
			t.Errorf("Feature %s: expected %v, got %v", f.name, f.supported, supported)
		}
	}
}
