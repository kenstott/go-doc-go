package udml_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/kennethstott/doculyzer-go-conversion/internal/graph"
	"github.com/kennethstott/doculyzer-go-conversion/internal/udml"
	"github.com/kennethstott/doculyzer-go-conversion/internal/udml/builder"
	"github.com/kennethstott/doculyzer-go-conversion/internal/udml/ontology"
	"github.com/kennethstott/doculyzer-go-conversion/internal/udml/query"
	"github.com/kennethstott/doculyzer-go-conversion/internal/udml/versioning"
)

// TestEndToEndPipeline tests the complete document → UDML → ontology → graph pipeline
func TestEndToEndPipeline(t *testing.T) {
	ctx := context.Background()

	// Step 1: Create sample UDML document
	doc := createSampleDocument(t)

	// Step 2: Query UDML elements
	backend := query.NewMemoryBackend()
	defer backend.Close()

	// Store document
	if err := backend.StoreDocument(ctx, doc); err != nil {
		t.Fatalf("Failed to store document: %v", err)
	}

	// Query elements
	q := &query.Query{
		Filters: []query.Filter{
			{Field: "element_type", Operator: query.OpEquals, Value: "section"},
		},
		Limit: 100,
	}

	results, err := backend.ExecuteQuery(ctx, q)
	if err != nil {
		t.Fatalf("Failed to execute query: %v", err)
	}

	if len(results) == 0 {
		t.Error("Expected query results, got none")
	}

	// Step 3: Extract ontology from UDML
	ont := createSampleOntology(t)
	extractor := ontology.NewExtractor(ont)

	extracted, err := extractor.Extract(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to extract ontology: %v", err)
	}

	if len(extracted.Entities) == 0 {
		t.Error("Expected extracted entities, got none")
	}

	if len(extracted.Relationships) == 0 {
		t.Error("Expected extracted relationships, got none")
	}

	// Step 4: Build knowledge graph
	graphBuilder := graph.NewBuilder()
	kg, err := graphBuilder.FromExtraction(extracted)
	if err != nil {
		t.Fatalf("Failed to build graph: %v", err)
	}

	if len(kg.Nodes) == 0 {
		t.Error("Expected graph nodes, got none")
	}

	if len(kg.Edges) == 0 {
		t.Error("Expected graph edges, got none")
	}

	// Validate graph structure
	validateGraph(t, kg)
}

// TestVersionMigrationWorkflow tests version upgrade scenarios
func TestVersionMigrationWorkflow(t *testing.T) {
	ctx := context.Background()

	// Create version manager
	versionBackend := versioning.NewMemoryBackend()
	vm := versioning.NewVersionManager(versionBackend)
	defer vm.Close()

	// Step 1: Create initial version
	v1 := &versioning.VersionMetadata{
		Version:         "1.0.0",
		ParserVersion:   "2.1.0",
		EmbeddingModel:  "text-embedding-ada-002",
		OntologyVersion: "1.0.0",
		ElementCount:    1000,
		DocumentCount:   10,
		Tags:            []string{"production"},
		CreatedAt:       time.Now().Add(-24 * time.Hour),
	}

	if err := vm.CreateVersion(ctx, v1); err != nil {
		t.Fatalf("Failed to create version 1.0.0: %v", err)
	}

	// Step 2: Create upgraded version
	v2 := &versioning.VersionMetadata{
		Version:         "2.0.0",
		ParserVersion:   "2.2.0",
		EmbeddingModel:  "text-embedding-3-small",
		OntologyVersion: "2.0.0",
		SchemaChanges: map[string]interface{}{
			"promoted_fields": []string{"speaker_id", "geo_location"},
		},
		ElementCount:  1500,
		DocumentCount: 15,
		Tags:          []string{"production", "latest"},
		CreatedAt:     time.Now(),
	}

	if err := vm.CreateVersion(ctx, v2); err != nil {
		t.Fatalf("Failed to create version 2.0.0: %v", err)
	}

	// Step 3: Create migration
	migration, err := vm.CreateMigration(ctx, "1.0.0", "2.0.0")
	if err != nil {
		t.Fatalf("Failed to create migration: %v", err)
	}

	// Step 4: Simulate migration progress
	migration.Status = versioning.MigrationRunning
	migration.TotalRecords = 1000
	migration.ProcessedRecords = 500

	if err := vm.UpdateMigration(ctx, migration); err != nil {
		t.Fatalf("Failed to update migration: %v", err)
	}

	// Step 5: Verify migration status
	retrieved, err := vm.GetMigration(ctx, "1.0.0", "2.0.0")
	if err != nil {
		t.Fatalf("Failed to retrieve migration: %v", err)
	}

	if retrieved.Status != versioning.MigrationRunning {
		t.Errorf("Expected migration status Running, got %s", retrieved.Status)
	}

	if retrieved.ProcessedRecords != 500 {
		t.Errorf("Expected 500 processed records, got %d", retrieved.ProcessedRecords)
	}

	// Step 6: Compare versions
	diff, err := vm.CompareVersions(ctx, "1.0.0", "2.0.0")
	if err != nil {
		t.Fatalf("Failed to compare versions: %v", err)
	}

	if diff.ElementsAdded != 500 {
		t.Errorf("Expected 500 elements added, got %d", diff.ElementsAdded)
	}

	if diff.DocumentsAdded != 5 {
		t.Errorf("Expected 5 documents added, got %d", diff.DocumentsAdded)
	}

	// Step 7: Deprecate old version
	if err := vm.DeprecateVersion(ctx, "1.0.0", "Superseded by 2.0.0"); err != nil {
		t.Fatalf("Failed to deprecate version: %v", err)
	}

	deprecated, err := vm.GetVersion(ctx, "1.0.0")
	if err != nil {
		t.Fatalf("Failed to get deprecated version: %v", err)
	}

	if !deprecated.Deprecated {
		t.Error("Expected version 1.0.0 to be deprecated")
	}
}

// TestMultiDomainOntology tests ontology with multiple data mesh domains
func TestMultiDomainOntology(t *testing.T) {
	ctx := context.Background()

	// Create ontology with multiple domains
	ont := &ontology.Ontology{
		Version: "1.0.0",
		Domains: []ontology.Domain{
			{
				Name:        "customer",
				Description: "Customer domain",
				Entities: []ontology.EntityDef{
					{
						Name:        "Customer",
						Description: "A customer entity",
						Queries: []ontology.QueryDef{
							{
								Path:       "$.elements[?(@.element_type=='section' && @.content =~ /Customer.*/)].content",
								IDColumns:  []string{"content"},
								Confidence: 0.9,
							},
						},
					},
				},
			},
			{
				Name:        "product",
				Description: "Product domain",
				Entities: []ontology.EntityDef{
					{
						Name:        "Product",
						Description: "A product entity",
						Queries: []ontology.QueryDef{
							{
								Path:       "$.elements[?(@.element_type=='section' && @.content =~ /Product.*/)].content",
								IDColumns:  []string{"content"},
								Confidence: 0.9,
							},
						},
					},
				},
			},
		},
	}

	// Create sample document
	doc := createSampleDocument(t)

	// Extract entities from each domain
	extractor := ontology.NewExtractor(ont)
	extracted, err := extractor.Extract(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to extract multi-domain ontology: %v", err)
	}

	// Validate domain ownership
	customerEntities := 0
	productEntities := 0

	for _, entity := range extracted.Entities {
		switch entity.Domain {
		case "customer":
			customerEntities++
		case "product":
			productEntities++
		}
	}

	if customerEntities == 0 {
		t.Error("Expected customer domain entities")
	}

	if productEntities == 0 {
		t.Error("Expected product domain entities")
	}

	t.Logf("Extracted %d customer entities and %d product entities",
		customerEntities, productEntities)
}

// TestPerformanceAtScale tests query performance with large datasets
func TestPerformanceAtScale(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	ctx := context.Background()
	backend := query.NewMemoryBackend()
	defer backend.Close()

	// Create document with many elements
	doc := createLargeDocument(t, 10000) // 10K elements

	startStore := time.Now()
	if err := backend.StoreDocument(ctx, doc); err != nil {
		t.Fatalf("Failed to store large document: %v", err)
	}
	storeDuration := time.Since(startStore)

	t.Logf("Stored 10K elements in %v", storeDuration)

	// Test query performance
	q := &query.Query{
		Filters: []query.Filter{
			{Field: "element_type", Operator: query.OpEquals, Value: "section"},
		},
		Limit: 100,
	}

	startQuery := time.Now()
	results, err := backend.ExecuteQuery(ctx, q)
	if err != nil {
		t.Fatalf("Failed to execute query: %v", err)
	}
	queryDuration := time.Since(startQuery)

	t.Logf("Queried %d results in %v", len(results), queryDuration)

	// Performance assertions
	if storeDuration > 5*time.Second {
		t.Errorf("Store operation too slow: %v (expected < 5s)", storeDuration)
	}

	if queryDuration > 100*time.Millisecond {
		t.Errorf("Query operation too slow: %v (expected < 100ms)", queryDuration)
	}
}

// Helper functions

func createSampleDocument(t *testing.T) *udml.Document {
	t.Helper()

	return &udml.Document{
		DocumentID: "test-doc-001",
		Metadata: udml.Metadata{
			Title:      "Test Document",
			SourceType: "test",
			CreatedAt:  time.Now(),
		},
		Elements: []udml.Element{
			{
				ElementID:   "elem-001",
				ElementType: "section",
				Content:     "Customer Information: John Doe is a premium customer.",
				Properties: map[string]interface{}{
					"page_number":   1,
					"section_level": 1,
				},
				ProviderName: "test-parser",
				Confidence:   0.95,
			},
			{
				ElementID:   "elem-002",
				ElementType: "section",
				Content:     "Product Details: Premium Widget XL is our flagship product.",
				Properties: map[string]interface{}{
					"page_number":   1,
					"section_level": 2,
				},
				ProviderName: "test-parser",
				Confidence:   0.90,
			},
			{
				ElementID:   "elem-003",
				ElementType: "table",
				Content:     "Order Summary Table",
				Properties: map[string]interface{}{
					"row_index":    0,
					"column_index": 0,
				},
				ProviderName: "test-parser",
				Confidence:   0.85,
			},
		},
		Relationships: []udml.Relationship{
			{
				SourceID:         "elem-001",
				TargetID:         "elem-002",
				RelationshipType: "contains",
				Confidence:       0.90,
			},
		},
	}
}

func createLargeDocument(t *testing.T, numElements int) *udml.Document {
	t.Helper()

	doc := &udml.Document{
		DocumentID: "large-doc-001",
		Metadata: udml.Metadata{
			Title:      "Large Test Document",
			SourceType: "test",
			CreatedAt:  time.Now(),
		},
		Elements: make([]udml.Element, numElements),
	}

	for i := 0; i < numElements; i++ {
		doc.Elements[i] = udml.Element{
			ElementID:   udml.NewElementID(),
			ElementType: "section",
			Content:     "Sample content for performance testing",
			Properties: map[string]interface{}{
				"page_number":   i / 100,
				"section_level": i % 5,
			},
			ProviderName: "test-parser",
			Confidence:   0.9,
		}
	}

	return doc
}

func createSampleOntology(t *testing.T) *ontology.Ontology {
	t.Helper()

	return &ontology.Ontology{
		Version:     "1.0.0",
		Description: "Test ontology for integration testing",
		Domains: []ontology.Domain{
			{
				Name:        "general",
				Description: "General domain for testing",
				Entities: []ontology.EntityDef{
					{
						Name:        "Customer",
						Description: "Customer entity",
						Queries: []ontology.QueryDef{
							{
								Path:       "$.elements[?(@.content =~ /Customer.*/)].content",
								IDColumns:  []string{"content"},
								Confidence: 0.9,
							},
						},
					},
					{
						Name:        "Product",
						Description: "Product entity",
						Queries: []ontology.QueryDef{
							{
								Path:       "$.elements[?(@.content =~ /Product.*/)].content",
								IDColumns:  []string{"content"},
								Confidence: 0.9,
							},
						},
					},
				},
				Relationships: []ontology.RelationshipDef{
					{
						Name:        "PURCHASED",
						Description: "Customer purchased product",
						SourceQuery: &ontology.QueryDef{
							Path:       "$.elements[?(@.content =~ /Customer.*/)].content",
							IDColumns:  []string{"content"},
							Confidence: 0.9,
						},
						TargetQuery: &ontology.QueryDef{
							Path:       "$.elements[?(@.content =~ /Product.*/)].content",
							IDColumns:  []string{"content"},
							Confidence: 0.9,
						},
					},
				},
			},
		},
	}
}

func validateGraph(t *testing.T, g *graph.Graph) {
	t.Helper()

	// Validate all nodes have IDs
	for _, node := range g.Nodes {
		if node.ID == "" {
			t.Error("Graph node missing ID")
		}
		if len(node.Labels) == 0 {
			t.Error("Graph node missing labels")
		}
	}

	// Validate all edges reference existing nodes
	nodeIDs := make(map[string]bool)
	for _, node := range g.Nodes {
		nodeIDs[node.ID] = true
	}

	for _, edge := range g.Edges {
		if edge.Source == "" || edge.Target == "" {
			t.Error("Graph edge missing source or target")
		}
		if !nodeIDs[edge.Source] {
			t.Errorf("Graph edge references non-existent source node: %s", edge.Source)
		}
		if !nodeIDs[edge.Target] {
			t.Errorf("Graph edge references non-existent target node: %s", edge.Target)
		}
		if edge.Type == "" {
			t.Error("Graph edge missing type")
		}
	}

	// Validate provenance
	for _, node := range g.Nodes {
		if prov := node.Properties["_provenance"]; prov != nil {
			provMap, ok := prov.(map[string]interface{})
			if !ok {
				t.Error("Invalid provenance format")
				continue
			}
			if _, hasDoc := provMap["document_id"]; !hasDoc {
				t.Error("Provenance missing document_id")
			}
			if _, hasElem := provMap["element_id"]; !hasElem {
				t.Error("Provenance missing element_id")
			}
		}
	}
}

// Benchmark tests

func BenchmarkEndToEndPipeline(b *testing.B) {
	ctx := context.Background()
	doc := createSampleDocument(&testing.T{})
	ont := createSampleOntology(&testing.T{})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Extract ontology
		extractor := ontology.NewExtractor(ont)
		extracted, err := extractor.Extract(ctx, doc)
		if err != nil {
			b.Fatalf("Extract failed: %v", err)
		}

		// Build graph
		graphBuilder := graph.NewBuilder()
		_, err = graphBuilder.FromExtraction(extracted)
		if err != nil {
			b.Fatalf("Graph build failed: %v", err)
		}
	}
}

func BenchmarkQueryExecution(b *testing.B) {
	ctx := context.Background()
	backend := query.NewMemoryBackend()
	defer backend.Close()

	doc := createLargeDocument(&testing.T{}, 1000)
	if err := backend.StoreDocument(ctx, doc); err != nil {
		b.Fatalf("Store failed: %v", err)
	}

	q := &query.Query{
		Filters: []query.Filter{
			{Field: "element_type", Operator: query.OpEquals, Value: "section"},
		},
		Limit: 100,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := backend.ExecuteQuery(ctx, q)
		if err != nil {
			b.Fatalf("Query failed: %v", err)
		}
	}
}

func BenchmarkOntologyExtraction(b *testing.B) {
	ctx := context.Background()
	doc := createSampleDocument(&testing.T{})
	ont := createSampleOntology(&testing.T{})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		extractor := ontology.NewExtractor(ont)
		_, err := extractor.Extract(ctx, doc)
		if err != nil {
			b.Fatalf("Extract failed: %v", err)
		}
	}
}

func BenchmarkGraphBuilding(b *testing.B) {
	ctx := context.Background()
	doc := createSampleDocument(&testing.T{})
	ont := createSampleOntology(&testing.T{})

	extractor := ontology.NewExtractor(ont)
	extracted, err := extractor.Extract(ctx, doc)
	if err != nil {
		b.Fatalf("Extract failed: %v", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		graphBuilder := graph.NewBuilder()
		_, err := graphBuilder.FromExtraction(extracted)
		if err != nil {
			b.Fatalf("Graph build failed: %v", err)
		}
	}
}

func BenchmarkJSONSerialization(b *testing.B) {
	doc := createLargeDocument(&testing.T{}, 1000)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(doc)
		if err != nil {
			b.Fatalf("Marshal failed: %v", err)
		}
	}
}
