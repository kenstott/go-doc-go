package export_test

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/kennethstott/doculyzer-go-conversion/internal/export"
	"github.com/kennethstott/doculyzer-go-conversion/internal/graph"
	"github.com/kennethstott/doculyzer-go-conversion/internal/udml"
	"github.com/kennethstott/doculyzer-go-conversion/internal/udml/ontology"
)

// TestMultiFormatExportPipeline tests complete pipeline from UDML → Graph → All Export Formats
func TestMultiFormatExportPipeline(t *testing.T) {
	ctx := context.Background()

	// Step 1: Create sample UDML document
	doc := createSampleUDMLDocument()

	// Step 2: Extract ontology
	ont := createSampleOntology()
	extractor := ontology.NewExtractor(ont)
	extracted, err := extractor.Extract(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to extract ontology: %v", err)
	}

	// Step 3: Build knowledge graph
	builder := graph.NewBuilder()
	kg, err := builder.FromExtraction(extracted)
	if err != nil {
		t.Fatalf("Failed to build graph: %v", err)
	}

	// Step 4: Test all export formats
	formats := []struct {
		name     string
		exporter export.Exporter
		validate func(*testing.T, []byte)
	}{
		{
			name:     "RDF/Turtle",
			exporter: export.NewRDFExporter(),
			validate: validateRDF,
		},
		{
			name:     "JSON-LD",
			exporter: export.NewJSONLDExporter(),
			validate: validateJSONLD,
		},
		{
			name:     "GraphML",
			exporter: export.NewGraphMLExporter(),
			validate: validateGraphML,
		},
		{
			name:     "Cypher",
			exporter: export.NewCypherExporter(),
			validate: validateCypher,
		},
		{
			name:     "RDF Triples",
			exporter: export.NewTriplesExporter(),
			validate: validateTriples,
		},
	}

	for _, format := range formats {
		t.Run(format.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := format.exporter.Export(kg, &buf); err != nil {
				t.Fatalf("Export to %s failed: %v", format.name, err)
			}

			data := buf.Bytes()
			if len(data) == 0 {
				t.Errorf("%s export produced no output", format.name)
			}

			// Validate format-specific output
			format.validate(t, data)

			t.Logf("%s export: %d bytes", format.name, len(data))
		})
	}
}

// TestProvenancePreservation tests that UDML provenance is preserved through export
func TestProvenancePreservation(t *testing.T) {
	ctx := context.Background()

	// Create UDML document with specific provenance
	doc := &udml.Document{
		DocumentID: "provenance-doc-001",
		Metadata: udml.Metadata{
			Title:      "Provenance Test",
			SourceType: "test",
		},
		Elements: []udml.Element{
			{
				ElementID:    "elem-prov-001",
				ElementType:  "section",
				Content:      "Customer: Alice with provenance",
				ProviderName: "provenance-parser",
				Confidence:   0.95,
			},
		},
	}

	// Extract with ontology
	ont := &ontology.Ontology{
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
				},
			},
		},
	}

	extractor := ontology.NewExtractor(ont)
	extracted, err := extractor.Extract(ctx, doc)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	// Build graph
	builder := graph.NewBuilder()
	kg, err := builder.FromExtraction(extracted)
	if err != nil {
		t.Fatalf("Graph build failed: %v", err)
	}

	// Test provenance in each format
	formats := []struct {
		name             string
		exporter         export.Exporter
		checkProvenance func(*testing.T, []byte)
	}{
		{
			name:            "RDF/Turtle",
			exporter:        export.NewRDFExporter(),
			checkProvenance: checkRDFProvenance,
		},
		{
			name:            "JSON-LD",
			exporter:        export.NewJSONLDExporter(),
			checkProvenance: checkJSONLDProvenance,
		},
		{
			name:            "GraphML",
			exporter:        export.NewGraphMLExporter(),
			checkProvenance: checkGraphMLProvenance,
		},
		{
			name:            "Cypher",
			exporter:        export.NewCypherExporter(),
			checkProvenance: checkCypherProvenance,
		},
	}

	for _, format := range formats {
		t.Run(format.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := format.exporter.Export(kg, &buf); err != nil {
				t.Fatalf("Export failed: %v", err)
			}

			format.checkProvenance(t, buf.Bytes())
		})
	}
}

// TestConfidenceScoreExport tests that confidence scores are exported correctly
func TestConfidenceScoreExport(t *testing.T) {
	// Create graph with specific confidence scores
	kg := &graph.Graph{
		Nodes: []graph.Node{
			{
				ID:     "node-1",
				Labels: []string{"Entity"},
				Properties: map[string]interface{}{
					"name":       "High Confidence Entity",
					"confidence": 0.95,
				},
			},
			{
				ID:     "node-2",
				Labels: []string{"Entity"},
				Properties: map[string]interface{}{
					"name":       "Low Confidence Entity",
					"confidence": 0.6,
				},
			},
		},
		Edges: []graph.Edge{
			{
				Source: "node-1",
				Target: "node-2",
				Type:   "RELATED",
				Properties: map[string]interface{}{
					"confidence": 0.8,
				},
			},
		},
	}

	// Test confidence in each format
	exporters := []export.Exporter{
		export.NewRDFExporter(),
		export.NewJSONLDExporter(),
		export.NewGraphMLExporter(),
		export.NewCypherExporter(),
		export.NewTriplesExporter(),
	}

	for _, exp := range exporters {
		t.Run(exp.Name(), func(t *testing.T) {
			var buf bytes.Buffer
			if err := exp.Export(kg, &buf); err != nil {
				t.Fatalf("Export failed: %v", err)
			}

			output := buf.String()

			// Check that confidence values are present in output
			if !strings.Contains(output, "0.95") && !strings.Contains(output, "95") {
				t.Errorf("%s export missing high confidence score", exp.Name())
			}

			t.Logf("%s successfully exported confidence scores", exp.Name())
		})
	}
}

// TestMultiLabelNodeExport tests nodes with multiple labels
func TestMultiLabelNodeExport(t *testing.T) {
	kg := &graph.Graph{
		Nodes: []graph.Node{
			{
				ID:     "multi-label-node",
				Labels: []string{"Entity", "Customer", "PremiumCustomer"},
				Properties: map[string]interface{}{
					"name": "Multi-Label Test",
				},
			},
		},
	}

	formats := []struct {
		name          string
		exporter      export.Exporter
		supportsMulti bool
	}{
		{"RDF/Turtle", export.NewRDFExporter(), true},
		{"JSON-LD", export.NewJSONLDExporter(), true},
		{"GraphML", export.NewGraphMLExporter(), true},
		{"Cypher", export.NewCypherExporter(), true},
		{"RDF Triples", export.NewTriplesExporter(), true},
	}

	for _, format := range formats {
		t.Run(format.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := format.exporter.Export(kg, &buf); err != nil {
				t.Fatalf("Export failed: %v", err)
			}

			if !format.supportsMulti {
				t.Skip("Format doesn't support multi-label")
				return
			}

			output := buf.String()

			// Check that all labels are present
			hasEntity := strings.Contains(output, "Entity")
			hasCustomer := strings.Contains(output, "Customer")
			hasPremium := strings.Contains(output, "Premium")

			if !hasEntity || !hasCustomer || !hasPremium {
				t.Errorf("%s missing some labels: Entity=%v, Customer=%v, Premium=%v",
					format.name, hasEntity, hasCustomer, hasPremium)
			}

			t.Logf("%s successfully exported multi-label node", format.name)
		})
	}
}

// TestLargeGraphExport tests export performance with large graphs
func TestLargeGraphExport(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large graph test in short mode")
	}

	// Create large graph (1000 nodes, 2000 edges)
	kg := createLargeGraph(1000, 2000)

	exporters := []export.Exporter{
		export.NewRDFExporter(),
		export.NewJSONLDExporter(),
		export.NewGraphMLExporter(),
		export.NewCypherExporter(),
		export.NewTriplesExporter(),
	}

	for _, exp := range exporters {
		t.Run(exp.Name(), func(t *testing.T) {
			var buf bytes.Buffer
			if err := exp.Export(kg, &buf); err != nil {
				t.Fatalf("Large graph export failed: %v", err)
			}

			size := buf.Len()
			t.Logf("%s exported 1000 nodes + 2000 edges = %d bytes (%.2f KB)",
				exp.Name(), size, float64(size)/1024)

			if size == 0 {
				t.Error("Export produced no output")
			}
		})
	}
}

// TestEmptyGraphHandling tests export of empty graphs
func TestEmptyGraphHandling(t *testing.T) {
	kg := &graph.Graph{
		Nodes: []graph.Node{},
		Edges: []graph.Edge{},
	}

	exporters := []export.Exporter{
		export.NewRDFExporter(),
		export.NewJSONLDExporter(),
		export.NewGraphMLExporter(),
		export.NewCypherExporter(),
		export.NewTriplesExporter(),
	}

	for _, exp := range exporters {
		t.Run(exp.Name(), func(t *testing.T) {
			var buf bytes.Buffer
			err := exp.Export(kg, &buf)

			// Some exporters may return error for empty graph, others may produce minimal output
			if err != nil {
				t.Logf("%s correctly handles empty graph with error: %v", exp.Name(), err)
			} else {
				t.Logf("%s produced minimal output for empty graph: %d bytes", exp.Name(), buf.Len())
			}
		})
	}
}

// TestDomainSeparation tests data mesh domain preservation in exports
func TestDomainSeparation(t *testing.T) {
	kg := &graph.Graph{
		Nodes: []graph.Node{
			{
				ID:     "customer-1",
				Labels: []string{"Customer"},
				Properties: map[string]interface{}{
					"name":   "Alice",
					"domain": "customers",
				},
			},
			{
				ID:     "product-1",
				Labels: []string{"Product"},
				Properties: map[string]interface{}{
					"name":   "Widget",
					"domain": "products",
				},
			},
		},
		Edges: []graph.Edge{
			{
				Source: "customer-1",
				Target: "product-1",
				Type:   "PURCHASED",
				Properties: map[string]interface{}{
					"domain": "sales",
				},
			},
		},
	}

	exporters := []export.Exporter{
		export.NewRDFExporter(),
		export.NewJSONLDExporter(),
		export.NewGraphMLExporter(),
		export.NewCypherExporter(),
		export.NewTriplesExporter(),
	}

	for _, exp := range exporters {
		t.Run(exp.Name(), func(t *testing.T) {
			var buf bytes.Buffer
			if err := exp.Export(kg, &buf); err != nil {
				t.Fatalf("Export failed: %v", err)
			}

			output := buf.String()

			// Verify domain properties are exported
			if !strings.Contains(output, "customers") {
				t.Errorf("%s missing 'customers' domain", exp.Name())
			}
			if !strings.Contains(output, "products") {
				t.Errorf("%s missing 'products' domain", exp.Name())
			}
			if !strings.Contains(output, "sales") {
				t.Errorf("%s missing 'sales' domain", exp.Name())
			}

			t.Logf("%s successfully exported domain information", exp.Name())
		})
	}
}

// Helper functions

func createSampleUDMLDocument() *udml.Document {
	return &udml.Document{
		DocumentID: "export-test-doc",
		Metadata: udml.Metadata{
			Title:      "Export Test Document",
			SourceType: "test",
		},
		Elements: []udml.Element{
			{
				ElementID:   "elem-1",
				ElementType: "section",
				Content:     "Customer: Alice Smith",
			},
			{
				ElementID:   "elem-2",
				ElementType: "section",
				Content:     "Product: Premium Widget",
			},
		},
	}
}

func createSampleOntology() *ontology.Ontology {
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
			},
		},
	}
}

func createLargeGraph(numNodes, numEdges int) *graph.Graph {
	kg := &graph.Graph{
		Nodes: make([]graph.Node, numNodes),
		Edges: make([]graph.Edge, numEdges),
	}

	// Create nodes
	for i := 0; i < numNodes; i++ {
		kg.Nodes[i] = graph.Node{
			ID:     graph.NewNodeID(),
			Labels: []string{"Entity"},
			Properties: map[string]interface{}{
				"name":  "Entity " + string(rune(i)),
				"index": i,
			},
		}
	}

	// Create edges
	for i := 0; i < numEdges; i++ {
		source := i % numNodes
		target := (i + 1) % numNodes
		kg.Edges[i] = graph.Edge{
			Source: kg.Nodes[source].ID,
			Target: kg.Nodes[target].ID,
			Type:   "RELATED",
			Properties: map[string]interface{}{
				"weight": 1.0,
			},
		}
	}

	return kg
}

// Validation functions

func validateRDF(t *testing.T, data []byte) {
	t.Helper()
	output := string(data)

	// RDF/Turtle should have @prefix declarations
	if !strings.Contains(output, "@prefix") {
		t.Error("RDF export missing @prefix declarations")
	}

	// Should have rdf:type predicates
	if !strings.Contains(output, "rdf:type") {
		t.Error("RDF export missing rdf:type predicates")
	}
}

func validateJSONLD(t *testing.T, data []byte) {
	t.Helper()

	// Should be valid JSON
	var jsonData map[string]interface{}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		t.Errorf("JSON-LD export is not valid JSON: %v", err)
	}

	// Should have @context
	if _, hasContext := jsonData["@context"]; !hasContext {
		t.Error("JSON-LD export missing @context")
	}
}

func validateGraphML(t *testing.T, data []byte) {
	t.Helper()
	output := string(data)

	// GraphML should have XML declaration
	if !strings.Contains(output, "<?xml") {
		t.Error("GraphML export missing XML declaration")
	}

	// Should have graphml root element
	if !strings.Contains(output, "<graphml") {
		t.Error("GraphML export missing <graphml> root")
	}

	// Should have graph element
	if !strings.Contains(output, "<graph") {
		t.Error("GraphML export missing <graph> element")
	}
}

func validateCypher(t *testing.T, data []byte) {
	t.Helper()
	output := string(data)

	// Cypher should have CREATE statements
	if !strings.Contains(output, "CREATE") {
		t.Error("Cypher export missing CREATE statements")
	}

	// Should create nodes
	if !strings.Contains(output, "(") && !strings.Contains(output, ")") {
		t.Error("Cypher export missing node syntax")
	}
}

func validateTriples(t *testing.T, data []byte) {
	t.Helper()
	output := string(data)

	// Triples format should be CSV-like
	lines := strings.Split(output, "\n")
	if len(lines) < 2 {
		t.Error("Triples export has too few lines")
	}

	// Should have subject, predicate, object columns
	if len(lines) > 0 && !strings.Contains(lines[0], ",") {
		t.Error("Triples export doesn't appear to be CSV format")
	}
}

func checkRDFProvenance(t *testing.T, data []byte) {
	t.Helper()
	output := string(data)

	// Check for provenance predicates
	if !strings.Contains(output, "provenance-doc-001") {
		t.Error("RDF export missing document_id provenance")
	}
	if !strings.Contains(output, "elem-prov-001") {
		t.Error("RDF export missing element_id provenance")
	}
}

func checkJSONLDProvenance(t *testing.T, data []byte) {
	t.Helper()
	output := string(data)

	// Check for provenance in JSON
	if !strings.Contains(output, "provenance-doc-001") {
		t.Error("JSON-LD export missing document_id provenance")
	}
	if !strings.Contains(output, "elem-prov-001") {
		t.Error("JSON-LD export missing element_id provenance")
	}
}

func checkGraphMLProvenance(t *testing.T, data []byte) {
	t.Helper()
	output := string(data)

	// Check for provenance in GraphML data elements
	if !strings.Contains(output, "provenance-doc-001") {
		t.Error("GraphML export missing document_id provenance")
	}
	if !strings.Contains(output, "elem-prov-001") {
		t.Error("GraphML export missing element_id provenance")
	}
}

func checkCypherProvenance(t *testing.T, data []byte) {
	t.Helper()
	output := string(data)

	// Check for provenance in node properties
	if !strings.Contains(output, "provenance-doc-001") {
		t.Error("Cypher export missing document_id provenance")
	}
	if !strings.Contains(output, "elem-prov-001") {
		t.Error("Cypher export missing element_id provenance")
	}
}

// Benchmark tests

func BenchmarkMultiFormatExport(b *testing.B) {
	kg := createLargeGraph(100, 200)

	exporters := []export.Exporter{
		export.NewRDFExporter(),
		export.NewJSONLDExporter(),
		export.NewGraphMLExporter(),
		export.NewCypherExporter(),
		export.NewTriplesExporter(),
	}

	for _, exp := range exporters {
		b.Run(exp.Name(), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				var buf bytes.Buffer
				if err := exp.Export(kg, &buf); err != nil {
					b.Fatalf("Export failed: %v", err)
				}
			}
		})
	}
}
