// Package export provides graph export to various formats (RDF, JSON-LD, GraphML, Cypher).
//
// The export package implements the Exporter interface for converting internal graph
// representations to standard knowledge graph and semantic web formats.
//
// # Supported Export Formats
//
// **RDF (Resource Description Framework)**:
//   - Turtle (.ttl) syntax
//   - W3C standard for knowledge graphs
//   - Supports UDML ontology, provenance, domains
//   - Triple-based representation (subject-predicate-object)
//
// **JSON-LD (JSON Linked Data)**:
//   - JSON format with @context
//   - W3C standard for semantic web
//   - Human-readable and machine-processable
//   - Supports UDML ontology, confidence scores, timestamps
//
// **GraphML**:
//   - XML-based graph format
//   - Supports node/edge properties
//   - Compatible with graph visualization tools (Gephi, Cytoscape)
//   - Schema definition in XML
//
// **Cypher**:
//   - Neo4j query language format
//   - CREATE statements for nodes and relationships
//   - Directly executable in Neo4j
//   - Optimized for graph database ingestion
//
// **Triples (N-Triples)**:
//   - Simple line-based RDF format
//   - Subject Predicate Object format
//   - Machine-processable
//   - Used for large-scale data exchange
//
// # Usage
//
// Export a graph to different formats:
//
//	import (
//		"github.com/kennethstott/doculyzer-go-conversion/internal/export"
//		"github.com/kennethstott/doculyzer-go-conversion/internal/graph"
//	)
//
//	// Create graph from UDML data
//	g := graph.NewGraph()
//	// ... populate graph
//
//	// Export to RDF/Turtle
//	rdfExporter := export.NewRDFExporter()
//	file, _ := os.Create("output.ttl")
//	rdfExporter.Export(g, file)
//
//	// Export to JSON-LD
//	jsonldExporter := export.NewJSONLDExporter()
//	file, _ := os.Create("output.jsonld")
//	jsonldExporter.Export(g, file)
//
//	// Export to GraphML
//	graphmlExporter := export.NewGraphMLExporter()
//	file, _ := os.Create("output.graphml")
//	graphmlExporter.Export(g, file)
//
//	// Export to Cypher
//	cypherExporter := export.NewCypherExporter()
//	file, _ := os.Create("output.cypher")
//	cypherExporter.Export(g, file)
//
// # Exporter Registry
//
// Use the registry pattern to manage exporters:
//
//	registry := export.NewExporterRegistry()
//	registry.Register(export.NewRDFExporter())
//	registry.Register(export.NewJSONLDExporter())
//	registry.Register(export.NewGraphMLExporter())
//
//	// Get exporter by name
//	exporter, _ := registry.Get("rdf")
//	exporter.Export(graph, writer)
//
//	// List available exporters
//	names := registry.ListExporters()
//
// # Export Features
//
// Different exporters support different features:
//
//	if exporter.SupportsFeature(export.FeatureProvenance) {
//		// Exporter can track UDML element IDs
//	}
//	if exporter.SupportsFeature(export.FeatureConfidence) {
//		// Exporter can include confidence scores
//	}
//
// **Standard Features**:
//   - FeatureProvenance: UDML element tracking
//   - FeatureConfidence: Confidence scores
//   - FeatureDomains: Data mesh domains
//   - FeatureTimestamps: Created/updated timestamps
//   - FeatureProperties: Arbitrary property bags
//   - FeatureMultiLabel: Multiple labels per node
//   - FeatureDirectedEdges: Directed relationships
//   - FeatureUndirectedEdges: Undirected relationships
//
// # Graph Model
//
// Exporters consume the internal graph representation:
//
//	type Graph struct {
//		Nodes []Node          // Graph nodes (entities, elements)
//		Edges []Edge          // Graph edges (relationships)
//	}
//
//	type Node struct {
//		ID         string                 // Unique node ID
//		Labels     []string               // Node types/labels
//		Properties map[string]interface{} // Node properties
//	}
//
//	type Edge struct {
//		Source     string                 // Source node ID
//		Target     string                 // Target node ID
//		Type       string                 // Relationship type
//		Properties map[string]interface{} // Edge properties
//	}
//
// See go/internal/graph/ for details.
//
// # RDF Export Example
//
// RDF/Turtle output:
//
//	@prefix udml: <http://doculyzer.org/udml/> .
//	@prefix entity: <http://doculyzer.org/entity/> .
//	@prefix domain: <http://doculyzer.org/domain/> .
//
//	entity:person_123 a udml:Person ;
//		udml:name "John Doe" ;
//		udml:domain domain:healthcare ;
//		udml:confidence 0.95 ;
//		udml:extractedAt "2024-01-15T10:30:00Z" .
//
//	entity:org_456 a udml:Organization ;
//		udml:name "Acme Corp" .
//
//	entity:person_123 udml:worksAt entity:org_456 .
//
// # JSON-LD Export Example
//
// JSON-LD output:
//
//	{
//		"@context": {
//			"udml": "http://doculyzer.org/udml/",
//			"entity": "http://doculyzer.org/entity/"
//		},
//		"@graph": [
//			{
//				"@id": "entity:person_123",
//				"@type": "udml:Person",
//				"udml:name": "John Doe",
//				"udml:domain": "domain:healthcare",
//				"udml:confidence": 0.95,
//				"udml:extractedAt": "2024-01-15T10:30:00Z"
//			}
//		]
//	}
//
// # GraphML Export Example
//
// GraphML output:
//
//	<?xml version="1.0" encoding="UTF-8"?>
//	<graphml xmlns="http://graphml.graphdrawing.org/xmlns">
//		<key id="name" for="node" attr.name="name" attr.type="string"/>
//		<key id="confidence" for="node" attr.name="confidence" attr.type="double"/>
//		<graph id="G" edgedefault="directed">
//			<node id="person_123">
//				<data key="name">John Doe</data>
//				<data key="confidence">0.95</data>
//			</node>
//			<edge source="person_123" target="org_456" label="worksAt"/>
//		</graph>
//	</graphml>
//
// # Cypher Export Example
//
// Cypher output (Neo4j):
//
//	CREATE (:Person {id: "person_123", name: "John Doe", confidence: 0.95});
//	CREATE (:Organization {id: "org_456", name: "Acme Corp"});
//	MATCH (p:Person {id: "person_123"}), (o:Organization {id: "org_456"})
//	CREATE (p)-[:WORKS_AT]->(o);
//
// # Performance
//
// Export performance depends on graph size:
//
//   - Small graph (< 10K nodes): < 1 second
//   - Medium graph (10K-100K nodes): 1-10 seconds
//   - Large graph (> 100K nodes): 10+ seconds
//
// All exporters stream output, so memory usage is O(1) regardless of graph size.
//
// # Error Handling
//
// Exporters return errors for:
//
//   - I/O failures (file write errors)
//   - Invalid graph structure (missing node IDs)
//   - Unsupported property types
//   - Format-specific validation errors
//
// Errors are wrapped with context using fmt.Errorf with %w.
//
// # See Also
//
//   - Graph Building: go/internal/graph/ (graph construction from UDML)
//   - Analytics Storage: go/internal/analytics/ (data source for graphs)
//   - Neo4j Integration: docs/operations/neo4j.md
//   - Examples: examples/neo4j-knowledge-graph/
package export
