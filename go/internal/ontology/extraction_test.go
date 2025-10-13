package ontology_test

import (
	"context"
	"testing"
	"time"

	"github.com/kennethstott/doculyzer-go-conversion/internal/udml"
	"github.com/kennethstott/doculyzer-go-conversion/internal/udml/ontology"
)

// TestJSONPathExtraction tests advanced JSONPath query execution
func TestJSONPathExtraction(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		ontology      *ontology.Ontology
		document      *udml.Document
		expectEntities int
		expectRels     int
	}{
		{
			name:           "Simple entity extraction",
			ontology:       createSimpleOntology(),
			document:       createSimpleDocument(),
			expectEntities: 2,
			expectRels:     0,
		},
		{
			name:           "Regex-based extraction",
			ontology:       createRegexOntology(),
			document:       createRegexDocument(),
			expectEntities: 3,
			expectRels:     1,
		},
		{
			name:           "Nested property extraction",
			ontology:       createNestedOntology(),
			document:       createNestedDocument(),
			expectEntities: 4,
			expectRels:     2,
		},
		{
			name:           "Multi-domain extraction",
			ontology:       createMultiDomainOntology(),
			document:       createMultiDomainDocument(),
			expectEntities: 6,
			expectRels:     2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractor := ontology.NewExtractor(tt.ontology)
			result, err := extractor.Extract(ctx, tt.document)
			if err != nil {
				t.Fatalf("Extract failed: %v", err)
			}

			if len(result.Entities) != tt.expectEntities {
				t.Errorf("Expected %d entities, got %d", tt.expectEntities, len(result.Entities))
			}

			if len(result.Relationships) != tt.expectRels {
				t.Errorf("Expected %d relationships, got %d", tt.expectRels, len(result.Relationships))
			}

			// Validate entity structure
			for _, entity := range result.Entities {
				if entity.ID == "" {
					t.Error("Entity missing ID")
				}
				if entity.Type == "" {
					t.Error("Entity missing type")
				}
				if entity.Confidence <= 0 || entity.Confidence > 1 {
					t.Errorf("Invalid confidence: %f", entity.Confidence)
				}
			}

			// Validate relationship structure
			for _, rel := range result.Relationships {
				if rel.SourceID == "" {
					t.Error("Relationship missing source ID")
				}
				if rel.TargetID == "" {
					t.Error("Relationship missing target ID")
				}
				if rel.Type == "" {
					t.Error("Relationship missing type")
				}
			}
		})
	}
}

// TestConfidenceScoring tests confidence score calculation
func TestConfidenceScoring(t *testing.T) {
	ctx := context.Background()

	ont := &ontology.Ontology{
		Version: "1.0.0",
		Domains: []ontology.Domain{
			{
				Name: "test",
				Entities: []ontology.EntityDef{
					{
						Name: "HighConfidence",
						Queries: []ontology.QueryDef{
							{
								Path:       "$.elements[?(@.element_type=='section')].content",
								IDColumns:  []string{"content"},
								Confidence: 0.95,
							},
						},
					},
					{
						Name: "LowConfidence",
						Queries: []ontology.QueryDef{
							{
								Path:       "$.elements[?(@.element_type=='paragraph')].content",
								IDColumns:  []string{"content"},
								Confidence: 0.5,
							},
						},
					},
				},
			},
		},
	}

	doc := &udml.Document{
		DocumentID: "test-doc",
		Elements: []udml.Element{
			{
				ElementID:   "elem-1",
				ElementType: "section",
				Content:     "High confidence content",
				Confidence:  0.9,
			},
			{
				ElementID:   "elem-2",
				ElementType: "paragraph",
				Content:     "Low confidence content",
				Confidence:  0.6,
			},
		},
	}

	extractor := ontology.NewExtractor(ont)
	result, err := extractor.Extract(ctx, doc)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	// Verify confidence scores
	for _, entity := range result.Entities {
		switch entity.Type {
		case "HighConfidence":
			// Combined confidence: query (0.95) * element (0.9) = 0.855
			if entity.Confidence < 0.85 || entity.Confidence > 0.86 {
				t.Errorf("Expected high confidence ~0.855, got %f", entity.Confidence)
			}
		case "LowConfidence":
			// Combined confidence: query (0.5) * element (0.6) = 0.3
			if entity.Confidence < 0.29 || entity.Confidence > 0.31 {
				t.Errorf("Expected low confidence ~0.3, got %f", entity.Confidence)
			}
		}
	}
}

// TestDomainOwnership tests data mesh domain assignment
func TestDomainOwnership(t *testing.T) {
	ctx := context.Background()

	ont := &ontology.Ontology{
		Version: "1.0.0",
		Domains: []ontology.Domain{
			{
				Name:        "customers",
				Description: "Customer data domain",
				Entities: []ontology.EntityDef{
					{
						Name: "Customer",
						Queries: []ontology.QueryDef{
							{
								Path:      "$.elements[?(@.content =~ /Customer.*/)].content",
								IDColumns: []string{"content"},
							},
						},
					},
				},
			},
			{
				Name:        "products",
				Description: "Product data domain",
				Entities: []ontology.EntityDef{
					{
						Name: "Product",
						Queries: []ontology.QueryDef{
							{
								Path:      "$.elements[?(@.content =~ /Product.*/)].content",
								IDColumns: []string{"content"},
							},
						},
					},
				},
			},
		},
	}

	doc := &udml.Document{
		DocumentID: "test-doc",
		Elements: []udml.Element{
			{
				ElementID: "elem-1",
				Content:   "Customer: John Doe",
			},
			{
				ElementID: "elem-2",
				Content:   "Product: Widget XL",
			},
		},
	}

	extractor := ontology.NewExtractor(ont)
	result, err := extractor.Extract(ctx, doc)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	// Validate domain ownership
	domainCounts := make(map[string]int)
	for _, entity := range result.Entities {
		domainCounts[entity.Domain]++
	}

	if domainCounts["customers"] != 1 {
		t.Errorf("Expected 1 customer entity, got %d", domainCounts["customers"])
	}

	if domainCounts["products"] != 1 {
		t.Errorf("Expected 1 product entity, got %d", domainCounts["products"])
	}

	// Verify entities have correct domain
	for _, entity := range result.Entities {
		if entity.Domain == "" {
			t.Error("Entity missing domain assignment")
		}
		if entity.Type == "Customer" && entity.Domain != "customers" {
			t.Errorf("Customer entity has wrong domain: %s", entity.Domain)
		}
		if entity.Type == "Product" && entity.Domain != "products" {
			t.Errorf("Product entity has wrong domain: %s", entity.Domain)
		}
	}
}

// TestRelationshipExtraction tests relationship query execution
func TestRelationshipExtraction(t *testing.T) {
	ctx := context.Background()

	ont := &ontology.Ontology{
		Version: "1.0.0",
		Domains: []ontology.Domain{
			{
				Name: "sales",
				Entities: []ontology.EntityDef{
					{
						Name: "Customer",
						Queries: []ontology.QueryDef{
							{
								Path:      "$.elements[?(@.element_type=='customer')].properties.name",
								IDColumns: []string{"name"},
							},
						},
					},
					{
						Name: "Product",
						Queries: []ontology.QueryDef{
							{
								Path:      "$.elements[?(@.element_type=='product')].properties.name",
								IDColumns: []string{"name"},
							},
						},
					},
				},
				Relationships: []ontology.RelationshipDef{
					{
						Name:        "PURCHASED",
						Description: "Customer purchased product",
						SourceQuery: &ontology.QueryDef{
							Path:      "$.elements[?(@.element_type=='customer')].properties.name",
							IDColumns: []string{"name"},
						},
						TargetQuery: &ontology.QueryDef{
							Path:      "$.elements[?(@.element_type=='product')].properties.name",
							IDColumns: []string{"name"},
						},
						Confidence: 0.8,
					},
				},
			},
		},
	}

	doc := &udml.Document{
		DocumentID: "test-doc",
		Elements: []udml.Element{
			{
				ElementID:   "elem-1",
				ElementType: "customer",
				Properties: map[string]interface{}{
					"name": "John Doe",
				},
			},
			{
				ElementID:   "elem-2",
				ElementType: "product",
				Properties: map[string]interface{}{
					"name": "Widget XL",
				},
			},
		},
	}

	extractor := ontology.NewExtractor(ont)
	result, err := extractor.Extract(ctx, doc)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if len(result.Relationships) == 0 {
		t.Fatal("Expected relationships to be extracted")
	}

	// Validate relationship
	rel := result.Relationships[0]
	if rel.Type != "PURCHASED" {
		t.Errorf("Expected PURCHASED relationship, got %s", rel.Type)
	}
	if rel.Confidence != 0.8 {
		t.Errorf("Expected confidence 0.8, got %f", rel.Confidence)
	}
	if rel.Domain != "sales" {
		t.Errorf("Expected domain 'sales', got %s", rel.Domain)
	}
}

// TestProvenanceTracking tests UDML element provenance in extracted entities
func TestProvenanceTracking(t *testing.T) {
	ctx := context.Background()

	ont := createSimpleOntology()
	doc := &udml.Document{
		DocumentID: "provenance-test-doc",
		Metadata: udml.Metadata{
			Title: "Provenance Test",
		},
		Elements: []udml.Element{
			{
				ElementID:    "elem-123",
				ElementType:  "section",
				Content:      "Test entity content",
				ProviderName: "test-parser",
				Confidence:   0.95,
			},
		},
	}

	extractor := ontology.NewExtractor(ont)
	result, err := extractor.Extract(ctx, doc)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if len(result.Entities) == 0 {
		t.Fatal("Expected entities to be extracted")
	}

	// Verify provenance
	entity := result.Entities[0]
	if entity.Provenance.DocumentID != "provenance-test-doc" {
		t.Errorf("Wrong document ID in provenance: %s", entity.Provenance.DocumentID)
	}
	if entity.Provenance.ElementID != "elem-123" {
		t.Errorf("Wrong element ID in provenance: %s", entity.Provenance.ElementID)
	}
	if entity.Provenance.ParserName != "test-parser" {
		t.Errorf("Wrong parser name in provenance: %s", entity.Provenance.ParserName)
	}
}

// TestAttributeExtraction tests property extraction from UDML elements
func TestAttributeExtraction(t *testing.T) {
	ctx := context.Background()

	ont := &ontology.Ontology{
		Version: "1.0.0",
		Domains: []ontology.Domain{
			{
				Name: "test",
				Entities: []ontology.EntityDef{
					{
						Name: "Person",
						Queries: []ontology.QueryDef{
							{
								Path:      "$.elements[?(@.element_type=='person')]",
								IDColumns: []string{"properties.id"},
								Attributes: map[string]string{
									"name":  "$.properties.name",
									"age":   "$.properties.age",
									"email": "$.properties.email",
								},
							},
						},
					},
				},
			},
		},
	}

	doc := &udml.Document{
		DocumentID: "test-doc",
		Elements: []udml.Element{
			{
				ElementID:   "elem-1",
				ElementType: "person",
				Properties: map[string]interface{}{
					"id":    "person-001",
					"name":  "Alice Smith",
					"age":   30,
					"email": "alice@example.com",
				},
			},
		},
	}

	extractor := ontology.NewExtractor(ont)
	result, err := extractor.Extract(ctx, doc)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if len(result.Entities) != 1 {
		t.Fatalf("Expected 1 entity, got %d", len(result.Entities))
	}

	entity := result.Entities[0]
	attrs := entity.Attributes

	if attrs["name"] != "Alice Smith" {
		t.Errorf("Expected name 'Alice Smith', got %v", attrs["name"])
	}
	if attrs["age"] != 30 {
		t.Errorf("Expected age 30, got %v", attrs["age"])
	}
	if attrs["email"] != "alice@example.com" {
		t.Errorf("Expected email 'alice@example.com', got %v", attrs["email"])
	}
}

// Helper functions

func createSimpleOntology() *ontology.Ontology {
	return &ontology.Ontology{
		Version: "1.0.0",
		Domains: []ontology.Domain{
			{
				Name: "test",
				Entities: []ontology.EntityDef{
					{
						Name: "TestEntity",
						Queries: []ontology.QueryDef{
							{
								Path:       "$.elements[?(@.element_type=='section')].content",
								IDColumns:  []string{"content"},
								Confidence: 0.9,
							},
						},
					},
				},
			},
		},
	}
}

func createSimpleDocument() *udml.Document {
	return &udml.Document{
		DocumentID: "simple-doc",
		Elements: []udml.Element{
			{
				ElementID:   "elem-1",
				ElementType: "section",
				Content:     "Entity 1",
			},
			{
				ElementID:   "elem-2",
				ElementType: "section",
				Content:     "Entity 2",
			},
		},
	}
}

func createRegexOntology() *ontology.Ontology {
	return &ontology.Ontology{
		Version: "1.0.0",
		Domains: []ontology.Domain{
			{
				Name: "test",
				Entities: []ontology.EntityDef{
					{
						Name: "Customer",
						Queries: []ontology.QueryDef{
							{
								Path:      "$.elements[?(@.content =~ /Customer.*/)].content",
								IDColumns: []string{"content"},
							},
						},
					},
					{
						Name: "Product",
						Queries: []ontology.QueryDef{
							{
								Path:      "$.elements[?(@.content =~ /Product.*/)].content",
								IDColumns: []string{"content"},
							},
						},
					},
				},
				Relationships: []ontology.RelationshipDef{
					{
						Name: "RELATED",
						SourceQuery: &ontology.QueryDef{
							Path:      "$.elements[?(@.content =~ /Customer.*/)].content",
							IDColumns: []string{"content"},
						},
						TargetQuery: &ontology.QueryDef{
							Path:      "$.elements[?(@.content =~ /Product.*/)].content",
							IDColumns: []string{"content"},
						},
					},
				},
			},
		},
	}
}

func createRegexDocument() *udml.Document {
	return &udml.Document{
		DocumentID: "regex-doc",
		Elements: []udml.Element{
			{ElementID: "e1", Content: "Customer: John"},
			{ElementID: "e2", Content: "Customer: Jane"},
			{ElementID: "e3", Content: "Product: Widget"},
		},
	}
}

func createNestedOntology() *ontology.Ontology {
	return &ontology.Ontology{
		Version: "1.0.0",
		Domains: []ontology.Domain{
			{
				Name: "test",
				Entities: []ontology.EntityDef{
					{
						Name: "Order",
						Queries: []ontology.QueryDef{
							{
								Path:      "$.elements[?(@.properties.type=='order')].properties.id",
								IDColumns: []string{"id"},
							},
						},
					},
					{
						Name: "LineItem",
						Queries: []ontology.QueryDef{
							{
								Path:      "$.elements[?(@.properties.type=='line_item')].properties.id",
								IDColumns: []string{"id"},
							},
						},
					},
				},
				Relationships: []ontology.RelationshipDef{
					{
						Name: "CONTAINS",
						SourceQuery: &ontology.QueryDef{
							Path:      "$.elements[?(@.properties.type=='order')].properties.id",
							IDColumns: []string{"id"},
						},
						TargetQuery: &ontology.QueryDef{
							Path:      "$.elements[?(@.properties.type=='line_item')].properties.id",
							IDColumns: []string{"id"},
						},
					},
					{
						Name: "NEXT",
						SourceQuery: &ontology.QueryDef{
							Path:      "$.elements[?(@.properties.type=='line_item')].properties.id",
							IDColumns: []string{"id"},
						},
						TargetQuery: &ontology.QueryDef{
							Path:      "$.elements[?(@.properties.type=='line_item')].properties.id",
							IDColumns: []string{"id"},
						},
					},
				},
			},
		},
	}
}

func createNestedDocument() *udml.Document {
	return &udml.Document{
		DocumentID: "nested-doc",
		Elements: []udml.Element{
			{
				ElementID: "order-1",
				Properties: map[string]interface{}{
					"type": "order",
					"id":   "ORD-001",
				},
			},
			{
				ElementID: "line-1",
				Properties: map[string]interface{}{
					"type": "line_item",
					"id":   "LINE-001",
				},
			},
			{
				ElementID: "line-2",
				Properties: map[string]interface{}{
					"type": "line_item",
					"id":   "LINE-002",
				},
			},
			{
				ElementID: "line-3",
				Properties: map[string]interface{}{
					"type": "line_item",
					"id":   "LINE-003",
				},
			},
		},
	}
}

func createMultiDomainOntology() *ontology.Ontology {
	return &ontology.Ontology{
		Version: "1.0.0",
		Domains: []ontology.Domain{
			{
				Name: "sales",
				Entities: []ontology.EntityDef{
					{
						Name: "Customer",
						Queries: []ontology.QueryDef{
							{
								Path:      "$.elements[?(@.properties.domain=='sales')].properties.customer",
								IDColumns: []string{"customer"},
							},
						},
					},
					{
						Name: "Order",
						Queries: []ontology.QueryDef{
							{
								Path:      "$.elements[?(@.properties.domain=='sales')].properties.order",
								IDColumns: []string{"order"},
							},
						},
					},
				},
				Relationships: []ontology.RelationshipDef{
					{
						Name: "PLACED",
						SourceQuery: &ontology.QueryDef{
							Path:      "$.elements[?(@.properties.domain=='sales')].properties.customer",
							IDColumns: []string{"customer"},
						},
						TargetQuery: &ontology.QueryDef{
							Path:      "$.elements[?(@.properties.domain=='sales')].properties.order",
							IDColumns: []string{"order"},
						},
					},
				},
			},
			{
				Name: "inventory",
				Entities: []ontology.EntityDef{
					{
						Name: "Product",
						Queries: []ontology.QueryDef{
							{
								Path:      "$.elements[?(@.properties.domain=='inventory')].properties.product",
								IDColumns: []string{"product"},
							},
						},
					},
					{
						Name: "Warehouse",
						Queries: []ontology.QueryDef{
							{
								Path:      "$.elements[?(@.properties.domain=='inventory')].properties.warehouse",
								IDColumns: []string{"warehouse"},
							},
						},
					},
				},
				Relationships: []ontology.RelationshipDef{
					{
						Name: "STORED_IN",
						SourceQuery: &ontology.QueryDef{
							Path:      "$.elements[?(@.properties.domain=='inventory')].properties.product",
							IDColumns: []string{"product"},
						},
						TargetQuery: &ontology.QueryDef{
							Path:      "$.elements[?(@.properties.domain=='inventory')].properties.warehouse",
							IDColumns: []string{"warehouse"},
						},
					},
				},
			},
		},
	}
}

func createMultiDomainDocument() *udml.Document {
	return &udml.Document{
		DocumentID: "multi-domain-doc",
		Elements: []udml.Element{
			{
				ElementID: "e1",
				Properties: map[string]interface{}{
					"domain":   "sales",
					"customer": "CUST-001",
				},
			},
			{
				ElementID: "e2",
				Properties: map[string]interface{}{
					"domain":   "sales",
					"customer": "CUST-002",
				},
			},
			{
				ElementID: "e3",
				Properties: map[string]interface{}{
					"domain": "sales",
					"order":  "ORD-001",
				},
			},
			{
				ElementID: "e4",
				Properties: map[string]interface{}{
					"domain":  "inventory",
					"product": "PROD-001",
				},
			},
			{
				ElementID: "e5",
				Properties: map[string]interface{}{
					"domain":  "inventory",
					"product": "PROD-002",
				},
			},
			{
				ElementID: "e6",
				Properties: map[string]interface{}{
					"domain":    "inventory",
					"warehouse": "WH-001",
				},
			},
		},
	}
}

// Benchmark tests

func BenchmarkSimpleExtraction(b *testing.B) {
	ctx := context.Background()
	ont := createSimpleOntology()
	doc := createSimpleDocument()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		extractor := ontology.NewExtractor(ont)
		_, err := extractor.Extract(ctx, doc)
		if err != nil {
			b.Fatalf("Extract failed: %v", err)
		}
	}
}

func BenchmarkRegexExtraction(b *testing.B) {
	ctx := context.Background()
	ont := createRegexOntology()
	doc := createRegexDocument()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		extractor := ontology.NewExtractor(ont)
		_, err := extractor.Extract(ctx, doc)
		if err != nil {
			b.Fatalf("Extract failed: %v", err)
		}
	}
}

func BenchmarkMultiDomainExtraction(b *testing.B) {
	ctx := context.Background()
	ont := createMultiDomainOntology()
	doc := createMultiDomainDocument()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		extractor := ontology.NewExtractor(ont)
		_, err := extractor.Extract(ctx, doc)
		if err != nil {
			b.Fatalf("Extract failed: %v", err)
		}
	}
}
