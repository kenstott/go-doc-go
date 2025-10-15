package export

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"strings"
	"testing"
	"time"

	"github.com/kennethstott/doculyzer-go-conversion/internal/graph"
)

// createTestGraph creates a test graph for export testing
func createTestGraph() *graph.Graph {
	now := time.Now()

	return &graph.Graph{
		ID:      "test-graph-1",
		Name:    "Test Knowledge Graph",
		Version: "1.0.0",
		Nodes: []graph.Node{
			{
				ID:              "node1",
				Labels:          []string{"Person", "Employee"},
				Properties:      map[string]interface{}{"name": "John Doe", "age": 30},
				SourceElementID: "elem-1",
				SourceDocID:     "doc-1",
				Domain:          "hr",
				Confidence:      0.95,
				CreatedAt:       now,
				UpdatedAt:       now,
			},
			{
				ID:              "node2",
				Labels:          []string{"Organization"},
				Properties:      map[string]interface{}{"name": "Acme Corp", "industry": "Technology"},
				SourceElementID: "elem-2",
				SourceDocID:     "doc-1",
				Domain:          "org",
				Confidence:      0.90,
				CreatedAt:       now,
				UpdatedAt:       now,
			},
			{
				ID:              "node3",
				Labels:          []string{"Location"},
				Properties:      map[string]interface{}{"city": "San Francisco", "country": "USA"},
				SourceElementID: "elem-3",
				SourceDocID:     "doc-1",
				Domain:          "geo",
				Confidence:      0.85,
				CreatedAt:       now,
				UpdatedAt:       now,
			},
		},
		Edges: []graph.Edge{
			{
				ID:              "edge1",
				Type:            "WORKS_FOR",
				SourceID:        "node1",
				TargetID:        "node2",
				Properties:      map[string]interface{}{"since": "2020-01-01", "role": "Engineer"},
				SourceElementID: "elem-4",
				Domain:          "hr",
				Confidence:      0.92,
				Evidence:        "Employment record",
				CreatedAt:       now,
			},
			{
				ID:              "edge2",
				Type:            "LOCATED_IN",
				SourceID:        "node2",
				TargetID:        "node3",
				Properties:      map[string]interface{}{"headquarters": true},
				SourceElementID: "elem-5",
				Domain:          "org",
				Confidence:      0.88,
				Evidence:        "Address information",
				CreatedAt:       now,
			},
		},
		Metadata:  map[string]interface{}{"source": "test"},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// Test Exporter Registry
func TestExporterRegistry(t *testing.T) {
	registry := NewExporterRegistry()

	// Test registration
	exporter := NewRDFExporter()
	err := registry.Register(exporter)
	if err != nil {
		t.Fatalf("Failed to register exporter: %v", err)
	}

	// Test duplicate registration
	err = registry.Register(exporter)
	if err == nil {
		t.Error("Expected error for duplicate registration")
	}

	// Test retrieval
	retrieved, err := registry.Get("rdf")
	if err != nil {
		t.Fatalf("Failed to get exporter: %v", err)
	}
	if retrieved.Name() != "rdf" {
		t.Errorf("Expected exporter name 'rdf', got '%s'", retrieved.Name())
	}

	// Test non-existent exporter
	_, err = registry.Get("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent exporter")
	}

	// Test list exporters
	exporters := registry.ListExporters()
	if len(exporters) != 1 {
		t.Errorf("Expected 1 exporter, got %d", len(exporters))
	}
}

// Test Global Registry
func TestGlobalRegistry(t *testing.T) {
	// All exporters should be auto-registered via init()
	exporters := ListExporters()

	expectedExporters := []string{"rdf", "jsonld", "graphml", "cypher", "triples"}
	if len(exporters) != len(expectedExporters) {
		t.Errorf("Expected %d exporters, got %d: %v", len(expectedExporters), len(exporters), exporters)
	}

	// Test each exporter can be retrieved
	for _, name := range expectedExporters {
		exporter, err := GetExporter(name)
		if err != nil {
			t.Errorf("Failed to get exporter '%s': %v", name, err)
		}
		if exporter.Name() != name {
			t.Errorf("Expected name '%s', got '%s'", name, exporter.Name())
		}
	}
}

// Test RDF Exporter
func TestRDFExporter(t *testing.T) {
	g := createTestGraph()
	exporter := NewRDFExporter()

	var buf bytes.Buffer
	err := exporter.Export(g, &buf)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	output := buf.String()

	// Check for prefixes
	if !strings.Contains(output, "@prefix") {
		t.Error("Output missing RDF prefixes")
	}

	// Check for nodes
	if !strings.Contains(output, "node1") || !strings.Contains(output, "node2") {
		t.Error("Output missing nodes")
	}

	// Check for edges
	if !strings.Contains(output, "WORKS_FOR") || !strings.Contains(output, "LOCATED_IN") {
		t.Error("Output missing edges")
	}

	// Check for labels
	if !strings.Contains(output, "Person") || !strings.Contains(output, "Organization") {
		t.Error("Output missing labels")
	}

	// Check for confidence
	if !strings.Contains(output, "confidence") {
		t.Error("Output missing confidence scores")
	}

	// Check for provenance
	if !strings.Contains(output, "sourceElementID") {
		t.Error("Output missing provenance")
	}
}

// Test JSON-LD Exporter
func TestJSONLDExporter(t *testing.T) {
	g := createTestGraph()
	exporter := NewJSONLDExporter()

	var buf bytes.Buffer
	err := exporter.Export(g, &buf)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Parse JSON to validate structure
	var doc map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatalf("Failed to parse JSON-LD: %v", err)
	}

	// Check context
	if _, ok := doc["@context"]; !ok {
		t.Error("Missing @context")
	}

	// Check graph metadata
	if doc["name"] != "Test Knowledge Graph" {
		t.Error("Missing or incorrect graph name")
	}

	// Check graph array
	graphArray, ok := doc["@graph"].([]interface{})
	if !ok {
		t.Fatal("Missing or invalid @graph")
	}

	// Should have 3 nodes + 2 edges = 5 entries (edges include reified statements)
	if len(graphArray) < 3 {
		t.Errorf("Expected at least 3 entries in @graph, got %d", len(graphArray))
	}
}

// Test GraphML Exporter
func TestGraphMLExporter(t *testing.T) {
	g := createTestGraph()
	exporter := NewGraphMLExporter()

	var buf bytes.Buffer
	err := exporter.Export(g, &buf)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Parse XML to validate structure
	var graphml GraphML
	if err := xml.Unmarshal(buf.Bytes(), &graphml); err != nil {
		t.Fatalf("Failed to parse GraphML: %v", err)
	}

	// Check namespace
	if graphml.XMLNS != "http://graphml.graphdrawing.org/xmlns" {
		t.Error("Incorrect GraphML namespace")
	}

	// Check keys are defined
	if len(graphml.Keys) == 0 {
		t.Error("No keys defined")
	}

	// Check nodes
	if len(graphml.Graph.Nodes) != 3 {
		t.Errorf("Expected 3 nodes, got %d", len(graphml.Graph.Nodes))
	}

	// Check edges
	if len(graphml.Graph.Edges) != 2 {
		t.Errorf("Expected 2 edges, got %d", len(graphml.Graph.Edges))
	}

	// Validate node has data
	if len(graphml.Graph.Nodes[0].Data) == 0 {
		t.Error("Node has no data attributes")
	}

	// Validate edge has source and target
	if graphml.Graph.Edges[0].Source == "" || graphml.Graph.Edges[0].Target == "" {
		t.Error("Edge missing source or target")
	}
}

// Test Cypher Exporter
func TestCypherExporter(t *testing.T) {
	g := createTestGraph()
	exporter := NewCypherExporter()

	var buf bytes.Buffer
	err := exporter.Export(g, &buf)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	output := buf.String()

	// Check for header comments
	if !strings.Contains(output, "// Neo4j Cypher Import Script") {
		t.Error("Missing header comments")
	}

	// Check for constraints
	if !strings.Contains(output, "CREATE CONSTRAINT") {
		t.Error("Missing constraint statements")
	}

	// Check for indexes
	if !strings.Contains(output, "CREATE INDEX") {
		t.Error("Missing index statements")
	}

	// Check for node creation
	if !strings.Contains(output, "MERGE (n:Person") || !strings.Contains(output, "MERGE (n:Organization") {
		t.Error("Missing node creation statements")
	}

	// Check for relationship creation
	if !strings.Contains(output, "WORKS_FOR") || !strings.Contains(output, "LOCATED_IN") {
		t.Error("Missing relationship creation statements")
	}

	// Check for transactions
	if !strings.Contains(output, ":begin") || !strings.Contains(output, ":commit") {
		t.Error("Missing transaction markers")
	}

	// Check for properties
	if !strings.Contains(output, "confidence") {
		t.Error("Missing property attributes")
	}
}

// Test Triples Exporter
func TestTriplesExporter(t *testing.T) {
	g := createTestGraph()
	exporter := NewTriplesExporter()

	var buf bytes.Buffer
	err := exporter.Export(g, &buf)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Parse CSV
	reader := csv.NewReader(&buf)
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to parse CSV: %v", err)
	}

	// Check header
	if len(records) < 1 {
		t.Fatal("No records in output")
	}
	header := records[0]
	expectedHeader := []string{"subject", "predicate", "object", "object_type", "graph_id"}
	if len(header) != len(expectedHeader) {
		t.Errorf("Expected %d columns, got %d", len(expectedHeader), len(header))
	}

	// Check we have multiple triples (graph metadata + nodes + edges)
	if len(records) < 10 {
		t.Errorf("Expected at least 10 triples, got %d", len(records))
	}

	// Check for node URIs
	foundNodeURI := false
	for _, record := range records[1:] {
		if strings.Contains(record[0], "urn:udml:node:") {
			foundNodeURI = true
			break
		}
	}
	if !foundNodeURI {
		t.Error("No node URIs found in triples")
	}

	// Check for edge types
	foundEdgeType := false
	for _, record := range records[1:] {
		if record[1] == "WORKS_FOR" || record[1] == "LOCATED_IN" {
			foundEdgeType = true
			break
		}
	}
	if !foundEdgeType {
		t.Error("No edge types found in triples")
	}
}

// Test Exporter Features
func TestExporterFeatures(t *testing.T) {
	exporters := []Exporter{
		NewRDFExporter(),
		NewJSONLDExporter(),
		NewGraphMLExporter(),
		NewCypherExporter(),
		NewTriplesExporter(),
	}

	for _, exporter := range exporters {
		t.Run(exporter.Name(), func(t *testing.T) {
			// All exporters should support these core features
			coreFeatures := []string{
				FeatureProvenance,
				FeatureConfidence,
				FeatureDomains,
				FeatureProperties,
				FeatureDirectedEdges,
			}

			for _, feature := range coreFeatures {
				if !exporter.SupportsFeature(feature) {
					t.Errorf("Exporter %s should support feature %s", exporter.Name(), feature)
				}
			}

			// Check unsupported feature returns false
			if exporter.SupportsFeature("unsupported_feature_xyz") {
				t.Errorf("Exporter %s should not support 'unsupported_feature_xyz'", exporter.Name())
			}
		})
	}
}

// Test Invalid Graph Validation
func TestExporterValidation(t *testing.T) {
	exporters := []Exporter{
		NewRDFExporter(),
		NewJSONLDExporter(),
		NewGraphMLExporter(),
		NewCypherExporter(),
		NewTriplesExporter(),
	}

	// Create invalid graph (edge references non-existent node)
	invalidGraph := &graph.Graph{
		ID:      "invalid-graph",
		Name:    "Invalid Graph",
		Version: "1.0.0",
		Nodes: []graph.Node{
			{ID: "node1", Labels: []string{"Test"}},
		},
		Edges: []graph.Edge{
			{ID: "edge1", Type: "TEST", SourceID: "node1", TargetID: "nonexistent"},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	for _, exporter := range exporters {
		t.Run(exporter.Name(), func(t *testing.T) {
			var buf bytes.Buffer
			err := exporter.Export(invalidGraph, &buf)
			if err == nil {
				t.Errorf("Exporter %s should reject invalid graph", exporter.Name())
			}
		})
	}
}

// Test Empty Graph Export
func TestEmptyGraphExport(t *testing.T) {
	emptyGraph := &graph.Graph{
		ID:        "empty-graph",
		Name:      "Empty Graph",
		Version:   "1.0.0",
		Nodes:     []graph.Node{},
		Edges:     []graph.Edge{},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	exporters := []Exporter{
		NewRDFExporter(),
		NewJSONLDExporter(),
		NewGraphMLExporter(),
		NewCypherExporter(),
		NewTriplesExporter(),
	}

	for _, exporter := range exporters {
		t.Run(exporter.Name(), func(t *testing.T) {
			var buf bytes.Buffer
			err := exporter.Export(emptyGraph, &buf)
			if err != nil {
				t.Errorf("Exporter %s failed on empty graph: %v", exporter.Name(), err)
			}

			if buf.Len() == 0 {
				t.Errorf("Exporter %s produced no output for empty graph", exporter.Name())
			}
		})
	}
}

// Test File Extensions
func TestFileExtensions(t *testing.T) {
	tests := []struct {
		name      string
		exporter  Exporter
		extension string
	}{
		{"RDF", NewRDFExporter(), ".ttl"},
		{"JSON-LD", NewJSONLDExporter(), ".jsonld"},
		{"GraphML", NewGraphMLExporter(), ".graphml"},
		{"Cypher", NewCypherExporter(), ".cypher"},
		{"Triples", NewTriplesExporter(), ".csv"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.exporter.FileExtension() != tt.extension {
				t.Errorf("Expected extension %s, got %s", tt.extension, tt.exporter.FileExtension())
			}
		})
	}
}

// Test Exporter Descriptions
func TestExporterDescriptions(t *testing.T) {
	exporters := []Exporter{
		NewRDFExporter(),
		NewJSONLDExporter(),
		NewGraphMLExporter(),
		NewCypherExporter(),
		NewTriplesExporter(),
	}

	for _, exporter := range exporters {
		t.Run(exporter.Name(), func(t *testing.T) {
			desc := exporter.Description()
			if desc == "" {
				t.Errorf("Exporter %s has empty description", exporter.Name())
			}
			if len(desc) < 10 {
				t.Errorf("Exporter %s has suspiciously short description: %s", exporter.Name(), desc)
			}
		})
	}
}

// Benchmark exporters
func BenchmarkRDFExporter(b *testing.B) {
	g := createTestGraph()
	exporter := NewRDFExporter()
	var buf bytes.Buffer

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		exporter.Export(g, &buf)
	}
}

func BenchmarkJSONLDExporter(b *testing.B) {
	g := createTestGraph()
	exporter := NewJSONLDExporter()
	var buf bytes.Buffer

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		exporter.Export(g, &buf)
	}
}

func BenchmarkGraphMLExporter(b *testing.B) {
	g := createTestGraph()
	exporter := NewGraphMLExporter()
	var buf bytes.Buffer

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		exporter.Export(g, &buf)
	}
}

func BenchmarkCypherExporter(b *testing.B) {
	g := createTestGraph()
	exporter := NewCypherExporter()
	var buf bytes.Buffer

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		exporter.Export(g, &buf)
	}
}

func BenchmarkTriplesExporter(b *testing.B) {
	g := createTestGraph()
	exporter := NewTriplesExporter()
	var buf bytes.Buffer

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		exporter.Export(g, &buf)
	}
}
