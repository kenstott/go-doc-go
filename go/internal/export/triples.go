package export

import (
	"encoding/csv"
	"fmt"
	"io"
	"strings"

	"github.com/kennethstott/doculyzer-go-conversion/internal/graph"
)

// TriplesExporter exports graphs to RDF triple CSV format (subject, predicate, object)
type TriplesExporter struct {
	IncludeProvenance bool // Include UDML provenance as additional triples
	IncludeConfidence bool // Include confidence scores as additional triples
	IncludeDomains    bool // Include data mesh domains as additional triples
	IncludeTimestamps bool // Include created/updated timestamps as additional triples
	IncludeHeader     bool // Include CSV header row
	URIPrefix         string // URI prefix for nodes (default: urn:udml:node:)
}

// NewTriplesExporter creates a new triples exporter with default settings
func NewTriplesExporter() *TriplesExporter {
	return &TriplesExporter{
		IncludeProvenance: true,
		IncludeConfidence: true,
		IncludeDomains:    true,
		IncludeTimestamps: true,
		IncludeHeader:     true,
		URIPrefix:         "urn:udml:node:",
	}
}

// Name returns the exporter identifier
func (e *TriplesExporter) Name() string {
	return "triples"
}

// Description returns a human-readable description
func (e *TriplesExporter) Description() string {
	return "RDF Triples CSV format - Subject, Predicate, Object"
}

// FileExtension returns the recommended file extension
func (e *TriplesExporter) FileExtension() string {
	return ".csv"
}

// SupportsFeature checks if the exporter supports a specific feature
func (e *TriplesExporter) SupportsFeature(feature string) bool {
	switch feature {
	case FeatureProvenance, FeatureConfidence, FeatureDomains,
		FeatureTimestamps, FeatureProperties, FeatureMultiLabel, FeatureDirectedEdges:
		return true
	default:
		return false
	}
}

// Export exports a graph to RDF triple CSV format
func (e *TriplesExporter) Export(g *graph.Graph, w io.Writer) error {
	// Validate graph
	if err := g.Validate(); err != nil {
		return fmt.Errorf("graph validation failed: %w", err)
	}

	// Create CSV writer
	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()

	// Write header
	if e.IncludeHeader {
		if err := csvWriter.Write([]string{"subject", "predicate", "object", "object_type", "graph_id"}); err != nil {
			return fmt.Errorf("failed to write header: %w", err)
		}
	}

	// Write graph metadata triples
	if err := e.writeGraphMetadataTriples(g, csvWriter); err != nil {
		return err
	}

	// Write node triples
	if err := e.writeNodeTriples(g, csvWriter); err != nil {
		return err
	}

	// Write edge triples
	if err := e.writeEdgeTriples(g, csvWriter); err != nil {
		return err
	}

	return nil
}

// writeGraphMetadataTriples writes graph-level metadata as triples
func (e *TriplesExporter) writeGraphMetadataTriples(g *graph.Graph, w *csv.Writer) error {
	graphURI := fmt.Sprintf("urn:udml:graph:%s", g.ID)

	triples := [][]string{
		{graphURI, "rdf:type", "udml:Graph", "uri", g.ID},
		{graphURI, "udml:name", g.Name, "literal", g.ID},
		{graphURI, "udml:version", g.Version, "literal", g.ID},
		{graphURI, "udml:nodeCount", fmt.Sprintf("%d", len(g.Nodes)), "literal", g.ID},
		{graphURI, "udml:edgeCount", fmt.Sprintf("%d", len(g.Edges)), "literal", g.ID},
	}

	if e.IncludeTimestamps {
		triples = append(triples,
			[]string{graphURI, "udml:createdAt", g.CreatedAt.Format("2006-01-02T15:04:05Z07:00"), "literal", g.ID},
			[]string{graphURI, "udml:updatedAt", g.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"), "literal", g.ID},
		)
	}

	for _, triple := range triples {
		if err := w.Write(triple); err != nil {
			return fmt.Errorf("failed to write graph metadata triple: %w", err)
		}
	}

	return nil
}

// writeNodeTriples writes all node triples
func (e *TriplesExporter) writeNodeTriples(g *graph.Graph, w *csv.Writer) error {
	for _, node := range g.Nodes {
		if err := e.writeNodeTriple(&node, g.ID, w); err != nil {
			return err
		}
	}
	return nil
}

// writeNodeTriple writes triples for a single node
func (e *TriplesExporter) writeNodeTriple(node *graph.Node, graphID string, w *csv.Writer) error {
	nodeURI := e.getNodeURI(node.ID)

	triples := [][]string{}

	// Write type triples (labels)
	for _, label := range node.Labels {
		triples = append(triples, []string{nodeURI, "rdf:type", label, "uri", graphID})
	}

	// Write confidence
	if e.IncludeConfidence && node.Confidence > 0 {
		triples = append(triples, []string{
			nodeURI, "udml:confidence", fmt.Sprintf("%.4f", node.Confidence), "literal", graphID,
		})
	}

	// Write provenance
	if e.IncludeProvenance {
		if node.SourceElementID != "" {
			triples = append(triples, []string{
				nodeURI, "udml:sourceElementID", node.SourceElementID, "literal", graphID,
			})
		}
		if node.SourceDocID != "" {
			triples = append(triples, []string{
				nodeURI, "udml:sourceDocID", node.SourceDocID, "literal", graphID,
			})
		}
	}

	// Write domain
	if e.IncludeDomains && node.Domain != "" {
		triples = append(triples, []string{
			nodeURI, "udml:domain", node.Domain, "literal", graphID,
		})
	}

	// Write timestamps
	if e.IncludeTimestamps {
		if !node.CreatedAt.IsZero() {
			triples = append(triples, []string{
				nodeURI, "udml:createdAt", node.CreatedAt.Format("2006-01-02T15:04:05Z07:00"), "literal", graphID,
			})
		}
		if !node.UpdatedAt.IsZero() {
			triples = append(triples, []string{
				nodeURI, "udml:updatedAt", node.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"), "literal", graphID,
			})
		}
	}

	// Write properties
	for key, value := range node.Properties {
		objectType := "literal"
		objectValue := fmt.Sprintf("%v", value)

		// Check if value is a reference to another node
		if strings.HasPrefix(objectValue, "urn:udml:node:") {
			objectType = "uri"
		}

		triples = append(triples, []string{
			nodeURI, e.sanitizePredicate(key), objectValue, objectType, graphID,
		})
	}

	// Write all triples
	for _, triple := range triples {
		if err := w.Write(triple); err != nil {
			return fmt.Errorf("failed to write node triple: %w", err)
		}
	}

	return nil
}

// writeEdgeTriples writes all edge triples
func (e *TriplesExporter) writeEdgeTriples(g *graph.Graph, w *csv.Writer) error {
	for _, edge := range g.Edges {
		if err := e.writeEdgeTriple(&edge, g.ID, w); err != nil {
			return err
		}
	}
	return nil
}

// writeEdgeTriple writes triples for a single edge
func (e *TriplesExporter) writeEdgeTriple(edge *graph.Edge, graphID string, w *csv.Writer) error {
	sourceURI := e.getNodeURI(edge.SourceID)
	targetURI := e.getNodeURI(edge.TargetID)
	edgeURI := fmt.Sprintf("urn:udml:edge:%s", edge.ID)

	triples := [][]string{
		// Main relationship triple
		{sourceURI, edge.Type, targetURI, "uri", graphID},
		// Edge as a resource (for metadata)
		{edgeURI, "rdf:type", "rdf:Statement", "uri", graphID},
		{edgeURI, "rdf:subject", sourceURI, "uri", graphID},
		{edgeURI, "rdf:predicate", edge.Type, "uri", graphID},
		{edgeURI, "rdf:object", targetURI, "uri", graphID},
	}

	// Write confidence
	if e.IncludeConfidence && edge.Confidence > 0 {
		triples = append(triples, []string{
			edgeURI, "udml:confidence", fmt.Sprintf("%.4f", edge.Confidence), "literal", graphID,
		})
	}

	// Write provenance
	if e.IncludeProvenance {
		if edge.SourceElementID != "" {
			triples = append(triples, []string{
				edgeURI, "udml:sourceElementID", edge.SourceElementID, "literal", graphID,
			})
		}
		if edge.Evidence != "" {
			triples = append(triples, []string{
				edgeURI, "udml:evidence", edge.Evidence, "literal", graphID,
			})
		}
	}

	// Write domain
	if e.IncludeDomains && edge.Domain != "" {
		triples = append(triples, []string{
			edgeURI, "udml:domain", edge.Domain, "literal", graphID,
		})
	}

	// Write timestamps
	if e.IncludeTimestamps && !edge.CreatedAt.IsZero() {
		triples = append(triples, []string{
			edgeURI, "udml:createdAt", edge.CreatedAt.Format("2006-01-02T15:04:05Z07:00"), "literal", graphID,
		})
	}

	// Write properties
	for key, value := range edge.Properties {
		objectType := "literal"
		objectValue := fmt.Sprintf("%v", value)

		// Check if value is a reference to another node
		if strings.HasPrefix(objectValue, "urn:udml:node:") {
			objectType = "uri"
		}

		triples = append(triples, []string{
			edgeURI, e.sanitizePredicate(key), objectValue, objectType, graphID,
		})
	}

	// Write all triples
	for _, triple := range triples {
		if err := w.Write(triple); err != nil {
			return fmt.Errorf("failed to write edge triple: %w", err)
		}
	}

	return nil
}

// getNodeURI returns the URI for a node ID
func (e *TriplesExporter) getNodeURI(nodeID string) string {
	return e.URIPrefix + nodeID
}

// sanitizePredicate sanitizes a predicate name for RDF
func (e *TriplesExporter) sanitizePredicate(pred string) string {
	// Replace spaces and special characters with underscores
	pred = strings.ReplaceAll(pred, " ", "_")
	pred = strings.ReplaceAll(pred, "-", "_")
	pred = strings.ReplaceAll(pred, ".", "_")
	pred = strings.ReplaceAll(pred, "/", "_")
	pred = strings.ReplaceAll(pred, ":", "_")

	// Remove any other non-alphanumeric characters
	var result strings.Builder
	for _, r := range pred {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			result.WriteRune(r)
		}
	}

	sanitized := result.String()
	if sanitized == "" {
		return "property"
	}
	return sanitized
}

// Register the triples exporter on package init
func init() {
	RegisterExporter(NewTriplesExporter())
}
