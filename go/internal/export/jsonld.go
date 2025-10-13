package export

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/kennethstott/doculyzer-go-conversion/internal/graph"
)

// JSONLDExporter exports graphs to JSON-LD format
type JSONLDExporter struct {
	Context           string // JSON-LD context URI
	IncludeProvenance bool   // Include UDML provenance
	IncludeConfidence bool   // Include confidence scores
	IncludeDomains    bool   // Include data mesh domains
	IncludeTimestamps bool   // Include created/updated timestamps
	PrettyPrint       bool   // Pretty-print JSON output
}

// NewJSONLDExporter creates a new JSON-LD exporter with default settings
func NewJSONLDExporter() *JSONLDExporter {
	return &JSONLDExporter{
		Context:           "http://udml.org/context.jsonld",
		IncludeProvenance: true,
		IncludeConfidence: true,
		IncludeDomains:    true,
		IncludeTimestamps: true,
		PrettyPrint:       true,
	}
}

// Name returns the exporter identifier
func (e *JSONLDExporter) Name() string {
	return "jsonld"
}

// Description returns a human-readable description
func (e *JSONLDExporter) Description() string {
	return "JSON-LD format - W3C standard for linked data"
}

// FileExtension returns the recommended file extension
func (e *JSONLDExporter) FileExtension() string {
	return ".jsonld"
}

// SupportsFeature checks if the exporter supports a specific feature
func (e *JSONLDExporter) SupportsFeature(feature string) bool {
	switch feature {
	case FeatureProvenance, FeatureConfidence, FeatureDomains,
		FeatureTimestamps, FeatureProperties, FeatureMultiLabel, FeatureDirectedEdges:
		return true
	default:
		return false
	}
}

// Export exports a graph to JSON-LD format
func (e *JSONLDExporter) Export(g *graph.Graph, w io.Writer) error {
	// Validate graph
	if err := g.Validate(); err != nil {
		return fmt.Errorf("graph validation failed: %w", err)
	}

	// Build JSON-LD document
	doc := e.buildJSONLDDocument(g)

	// Marshal to JSON
	var data []byte
	var err error
	if e.PrettyPrint {
		data, err = json.MarshalIndent(doc, "", "  ")
	} else {
		data, err = json.Marshal(doc)
	}
	if err != nil {
		return fmt.Errorf("failed to marshal JSON-LD: %w", err)
	}

	// Write to output
	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("failed to write JSON-LD: %w", err)
	}

	return nil
}

// buildJSONLDDocument builds the JSON-LD document structure
func (e *JSONLDExporter) buildJSONLDDocument(g *graph.Graph) map[string]interface{} {
	doc := map[string]interface{}{
		"@context": map[string]interface{}{
			"@vocab":    "http://udml.org/vocab#",
			"rdf":       "http://www.w3.org/1999/02/22-rdf-syntax-ns#",
			"rdfs":      "http://www.w3.org/2000/01/rdf-schema#",
			"xsd":       "http://www.w3.org/2001/XMLSchema#",
			"udml":      "http://udml.org/vocab#",
			"id":        "@id",
			"type":      "@type",
			"graph":     "@graph",
		},
		"@type":   "udml:Graph",
		"@id":     fmt.Sprintf("urn:udml:graph:%s", g.ID),
		"name":    g.Name,
		"version": g.Version,
	}

	// Add graph metadata
	if e.IncludeTimestamps {
		doc["createdAt"] = g.CreatedAt.Format("2006-01-02T15:04:05Z07:00")
		doc["updatedAt"] = g.UpdatedAt.Format("2006-01-02T15:04:05Z07:00")
	}

	// Add nodes and edges
	graphArray := make([]map[string]interface{}, 0, len(g.Nodes)+len(g.Edges))

	// Add nodes
	for _, node := range g.Nodes {
		nodeData := e.buildNodeData(&node)
		graphArray = append(graphArray, nodeData)
	}

	// Add edges
	for _, edge := range g.Edges {
		edgeData := e.buildEdgeData(&edge)
		graphArray = append(graphArray, edgeData)
	}

	doc["@graph"] = graphArray

	// Add graph statistics
	doc["nodeCount"] = len(g.Nodes)
	doc["edgeCount"] = len(g.Edges)

	return doc
}

// buildNodeData builds JSON-LD data for a node
func (e *JSONLDExporter) buildNodeData(node *graph.Node) map[string]interface{} {
	nodeData := map[string]interface{}{
		"@id": fmt.Sprintf("urn:udml:node:%s", node.ID),
	}

	// Add types (labels)
	if len(node.Labels) > 0 {
		if len(node.Labels) == 1 {
			nodeData["@type"] = node.Labels[0]
		} else {
			nodeData["@type"] = node.Labels
		}
	}

	// Add confidence
	if e.IncludeConfidence && node.Confidence > 0 {
		nodeData["confidence"] = node.Confidence
	}

	// Add provenance
	if e.IncludeProvenance {
		if node.SourceElementID != "" {
			nodeData["sourceElementID"] = node.SourceElementID
		}
		if node.SourceDocID != "" {
			nodeData["sourceDocID"] = node.SourceDocID
		}
	}

	// Add domain
	if e.IncludeDomains && node.Domain != "" {
		nodeData["domain"] = node.Domain
	}

	// Add timestamps
	if e.IncludeTimestamps {
		if !node.CreatedAt.IsZero() {
			nodeData["createdAt"] = node.CreatedAt.Format("2006-01-02T15:04:05Z07:00")
		}
		if !node.UpdatedAt.IsZero() {
			nodeData["updatedAt"] = node.UpdatedAt.Format("2006-01-02T15:04:05Z07:00")
		}
	}

	// Add properties
	for key, value := range node.Properties {
		nodeData[key] = value
	}

	return nodeData
}

// buildEdgeData builds JSON-LD data for an edge
func (e *JSONLDExporter) buildEdgeData(edge *graph.Edge) map[string]interface{} {
	edgeData := map[string]interface{}{
		"@id":     fmt.Sprintf("urn:udml:edge:%s", edge.ID),
		"@type":   "rdf:Statement",
		"subject": map[string]interface{}{
			"@id": fmt.Sprintf("urn:udml:node:%s", edge.SourceID),
		},
		"predicate": edge.Type,
		"object": map[string]interface{}{
			"@id": fmt.Sprintf("urn:udml:node:%s", edge.TargetID),
		},
	}

	// Add confidence
	if e.IncludeConfidence && edge.Confidence > 0 {
		edgeData["confidence"] = edge.Confidence
	}

	// Add provenance
	if e.IncludeProvenance {
		if edge.SourceElementID != "" {
			edgeData["sourceElementID"] = edge.SourceElementID
		}
		if edge.Evidence != "" {
			edgeData["evidence"] = edge.Evidence
		}
	}

	// Add domain
	if e.IncludeDomains && edge.Domain != "" {
		edgeData["domain"] = edge.Domain
	}

	// Add timestamps
	if e.IncludeTimestamps && !edge.CreatedAt.IsZero() {
		edgeData["createdAt"] = edge.CreatedAt.Format("2006-01-02T15:04:05Z07:00")
	}

	// Add properties
	for key, value := range edge.Properties {
		edgeData[key] = value
	}

	return edgeData
}

// Register the JSON-LD exporter on package init
func init() {
	RegisterExporter(NewJSONLDExporter())
}
