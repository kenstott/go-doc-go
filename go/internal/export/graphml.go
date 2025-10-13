package export

import (
	"encoding/xml"
	"fmt"
	"io"

	"github.com/kennethstott/doculyzer-go-conversion/internal/graph"
)

// GraphMLExporter exports graphs to GraphML XML format
type GraphMLExporter struct {
	IncludeProvenance bool // Include UDML provenance
	IncludeConfidence bool // Include confidence scores
	IncludeDomains    bool // Include data mesh domains
	IncludeTimestamps bool // Include created/updated timestamps
}

// NewGraphMLExporter creates a new GraphML exporter with default settings
func NewGraphMLExporter() *GraphMLExporter {
	return &GraphMLExporter{
		IncludeProvenance: true,
		IncludeConfidence: true,
		IncludeDomains:    true,
		IncludeTimestamps: true,
	}
}

// Name returns the exporter identifier
func (e *GraphMLExporter) Name() string {
	return "graphml"
}

// Description returns a human-readable description
func (e *GraphMLExporter) Description() string {
	return "GraphML format - Universal graph interchange format (XML)"
}

// FileExtension returns the recommended file extension
func (e *GraphMLExporter) FileExtension() string {
	return ".graphml"
}

// SupportsFeature checks if the exporter supports a specific feature
func (e *GraphMLExporter) SupportsFeature(feature string) bool {
	switch feature {
	case FeatureProvenance, FeatureConfidence, FeatureDomains,
		FeatureTimestamps, FeatureProperties, FeatureMultiLabel, FeatureDirectedEdges:
		return true
	default:
		return false
	}
}

// Export exports a graph to GraphML format
func (e *GraphMLExporter) Export(g *graph.Graph, w io.Writer) error {
	// Validate graph
	if err := g.Validate(); err != nil {
		return fmt.Errorf("graph validation failed: %w", err)
	}

	// Build GraphML structure
	graphml := e.buildGraphML(g)

	// Marshal to XML with indentation
	encoder := xml.NewEncoder(w)
	encoder.Indent("", "  ")

	// Write XML header
	if _, err := w.Write([]byte(xml.Header)); err != nil {
		return fmt.Errorf("failed to write XML header: %w", err)
	}

	// Encode GraphML
	if err := encoder.Encode(graphml); err != nil {
		return fmt.Errorf("failed to encode GraphML: %w", err)
	}

	return nil
}

// GraphML XML structures
type GraphML struct {
	XMLName xml.Name `xml:"graphml"`
	XMLNS   string   `xml:"xmlns,attr"`
	XMLNSXSI string  `xml:"xmlns:xsi,attr"`
	SchemaLocation string `xml:"xsi:schemaLocation,attr"`
	Keys    []Key    `xml:"key"`
	Graph   GraphMLGraph `xml:"graph"`
}

type Key struct {
	ID       string `xml:"id,attr"`
	For      string `xml:"for,attr"` // "node", "edge", or "graph"
	AttrName string `xml:"attr.name,attr"`
	AttrType string `xml:"attr.type,attr"` // "string", "int", "long", "float", "double", "boolean"
}

type GraphMLGraph struct {
	ID          string `xml:"id,attr"`
	EdgeDefault string `xml:"edgedefault,attr"` // "directed" or "undirected"
	Data        []Data `xml:"data"`
	Nodes       []GraphMLNode `xml:"node"`
	Edges       []GraphMLEdge `xml:"edge"`
}

type GraphMLNode struct {
	ID   string `xml:"id,attr"`
	Data []Data `xml:"data"`
}

type GraphMLEdge struct {
	ID     string `xml:"id,attr"`
	Source string `xml:"source,attr"`
	Target string `xml:"target,attr"`
	Data   []Data `xml:"data"`
}

type Data struct {
	Key   string `xml:"key,attr"`
	Value string `xml:",chardata"`
}

// buildGraphML builds the GraphML structure from a graph
func (e *GraphMLExporter) buildGraphML(g *graph.Graph) GraphML {
	graphml := GraphML{
		XMLNS:   "http://graphml.graphdrawing.org/xmlns",
		XMLNSXSI: "http://www.w3.org/2001/XMLSchema-instance",
		SchemaLocation: "http://graphml.graphdrawing.org/xmlns http://graphml.graphdrawing.org/xmlns/1.0/graphml.xsd",
		Keys:    e.defineKeys(),
		Graph: GraphMLGraph{
			ID:          g.ID,
			EdgeDefault: "directed",
			Data:        e.buildGraphData(g),
			Nodes:       e.buildNodes(g.Nodes),
			Edges:       e.buildEdges(g.Edges),
		},
	}

	return graphml
}

// defineKeys defines the attribute keys for nodes and edges
func (e *GraphMLExporter) defineKeys() []Key {
	keys := []Key{
		// Graph keys
		{ID: "g_name", For: "graph", AttrName: "name", AttrType: "string"},
		{ID: "g_version", For: "graph", AttrName: "version", AttrType: "string"},

		// Node keys
		{ID: "n_labels", For: "node", AttrName: "labels", AttrType: "string"},
		{ID: "n_label", For: "node", AttrName: "label", AttrType: "string"}, // Primary label for visualization
	}

	if e.IncludeConfidence {
		keys = append(keys, Key{ID: "n_confidence", For: "node", AttrName: "confidence", AttrType: "double"})
		keys = append(keys, Key{ID: "e_confidence", For: "edge", AttrName: "confidence", AttrType: "double"})
	}

	if e.IncludeProvenance {
		keys = append(keys,
			Key{ID: "n_source_element_id", For: "node", AttrName: "sourceElementID", AttrType: "string"},
			Key{ID: "n_source_doc_id", For: "node", AttrName: "sourceDocID", AttrType: "string"},
			Key{ID: "e_source_element_id", For: "edge", AttrName: "sourceElementID", AttrType: "string"},
			Key{ID: "e_evidence", For: "edge", AttrName: "evidence", AttrType: "string"},
		)
	}

	if e.IncludeDomains {
		keys = append(keys,
			Key{ID: "n_domain", For: "node", AttrName: "domain", AttrType: "string"},
			Key{ID: "e_domain", For: "edge", AttrName: "domain", AttrType: "string"},
		)
	}

	if e.IncludeTimestamps {
		keys = append(keys,
			Key{ID: "n_created_at", For: "node", AttrName: "createdAt", AttrType: "string"},
			Key{ID: "n_updated_at", For: "node", AttrName: "updatedAt", AttrType: "string"},
			Key{ID: "e_created_at", For: "edge", AttrName: "createdAt", AttrType: "string"},
		)
	}

	// Edge type key
	keys = append(keys, Key{ID: "e_type", For: "edge", AttrName: "type", AttrType: "string"})

	return keys
}

// buildGraphData builds graph-level data attributes
func (e *GraphMLExporter) buildGraphData(g *graph.Graph) []Data {
	data := []Data{
		{Key: "g_name", Value: g.Name},
		{Key: "g_version", Value: g.Version},
	}
	return data
}

// buildNodes builds GraphML nodes from graph nodes
func (e *GraphMLExporter) buildNodes(nodes []graph.Node) []GraphMLNode {
	graphmlNodes := make([]GraphMLNode, 0, len(nodes))

	for _, node := range nodes {
		graphmlNode := GraphMLNode{
			ID:   node.ID,
			Data: e.buildNodeData(&node),
		}
		graphmlNodes = append(graphmlNodes, graphmlNode)
	}

	return graphmlNodes
}

// buildNodeData builds data attributes for a node
func (e *GraphMLExporter) buildNodeData(node *graph.Node) []Data {
	data := []Data{}

	// Add labels as comma-separated string
	if len(node.Labels) > 0 {
		labelsStr := ""
		for i, label := range node.Labels {
			if i > 0 {
				labelsStr += ","
			}
			labelsStr += label
		}
		data = append(data, Data{Key: "n_labels", Value: labelsStr})
		// Add primary label for visualization
		data = append(data, Data{Key: "n_label", Value: node.Labels[0]})
	}

	// Add confidence
	if e.IncludeConfidence && node.Confidence > 0 {
		data = append(data, Data{Key: "n_confidence", Value: fmt.Sprintf("%.4f", node.Confidence)})
	}

	// Add provenance
	if e.IncludeProvenance {
		if node.SourceElementID != "" {
			data = append(data, Data{Key: "n_source_element_id", Value: node.SourceElementID})
		}
		if node.SourceDocID != "" {
			data = append(data, Data{Key: "n_source_doc_id", Value: node.SourceDocID})
		}
	}

	// Add domain
	if e.IncludeDomains && node.Domain != "" {
		data = append(data, Data{Key: "n_domain", Value: node.Domain})
	}

	// Add timestamps
	if e.IncludeTimestamps {
		if !node.CreatedAt.IsZero() {
			data = append(data, Data{Key: "n_created_at", Value: node.CreatedAt.Format("2006-01-02T15:04:05Z07:00")})
		}
		if !node.UpdatedAt.IsZero() {
			data = append(data, Data{Key: "n_updated_at", Value: node.UpdatedAt.Format("2006-01-02T15:04:05Z07:00")})
		}
	}

	// Add properties as additional keys (dynamically)
	for key, value := range node.Properties {
		data = append(data, Data{Key: key, Value: fmt.Sprintf("%v", value)})
	}

	return data
}

// buildEdges builds GraphML edges from graph edges
func (e *GraphMLExporter) buildEdges(edges []graph.Edge) []GraphMLEdge {
	graphmlEdges := make([]GraphMLEdge, 0, len(edges))

	for _, edge := range edges {
		graphmlEdge := GraphMLEdge{
			ID:     edge.ID,
			Source: edge.SourceID,
			Target: edge.TargetID,
			Data:   e.buildEdgeData(&edge),
		}
		graphmlEdges = append(graphmlEdges, graphmlEdge)
	}

	return graphmlEdges
}

// buildEdgeData builds data attributes for an edge
func (e *GraphMLExporter) buildEdgeData(edge *graph.Edge) []Data {
	data := []Data{
		{Key: "e_type", Value: edge.Type},
	}

	// Add confidence
	if e.IncludeConfidence && edge.Confidence > 0 {
		data = append(data, Data{Key: "e_confidence", Value: fmt.Sprintf("%.4f", edge.Confidence)})
	}

	// Add provenance
	if e.IncludeProvenance {
		if edge.SourceElementID != "" {
			data = append(data, Data{Key: "e_source_element_id", Value: edge.SourceElementID})
		}
		if edge.Evidence != "" {
			data = append(data, Data{Key: "e_evidence", Value: edge.Evidence})
		}
	}

	// Add domain
	if e.IncludeDomains && edge.Domain != "" {
		data = append(data, Data{Key: "e_domain", Value: edge.Domain})
	}

	// Add timestamps
	if e.IncludeTimestamps && !edge.CreatedAt.IsZero() {
		data = append(data, Data{Key: "e_created_at", Value: edge.CreatedAt.Format("2006-01-02T15:04:05Z07:00")})
	}

	// Add properties
	for key, value := range edge.Properties {
		data = append(data, Data{Key: key, Value: fmt.Sprintf("%v", value)})
	}

	return data
}

// Register the GraphML exporter on package init
func init() {
	RegisterExporter(NewGraphMLExporter())
}
