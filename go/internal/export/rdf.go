package export

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/kennethstott/doculyzer-go-conversion/internal/graph"
)

// RDFExporter exports graphs to RDF Turtle format
type RDFExporter struct {
	Namespace string // Base namespace URI (default: http://udml.org/ontology#)
	IncludeProvenance bool // Include UDML provenance
	IncludeConfidence bool // Include confidence scores
	IncludeDomains    bool // Include data mesh domains
	IncludeTimestamps bool // Include created/updated timestamps
}

// NewRDFExporter creates a new RDF/Turtle exporter with default settings
func NewRDFExporter() *RDFExporter {
	return &RDFExporter{
		Namespace:         "http://udml.org/ontology#",
		IncludeProvenance: true,
		IncludeConfidence: true,
		IncludeDomains:    true,
		IncludeTimestamps: true,
	}
}

// Name returns the exporter identifier
func (e *RDFExporter) Name() string {
	return "rdf"
}

// Description returns a human-readable description
func (e *RDFExporter) Description() string {
	return "RDF Turtle format - W3C standard for semantic web"
}

// FileExtension returns the recommended file extension
func (e *RDFExporter) FileExtension() string {
	return ".ttl"
}

// SupportsFeature checks if the exporter supports a specific feature
func (e *RDFExporter) SupportsFeature(feature string) bool {
	switch feature {
	case FeatureProvenance, FeatureConfidence, FeatureDomains,
		FeatureTimestamps, FeatureProperties, FeatureMultiLabel, FeatureDirectedEdges:
		return true
	default:
		return false
	}
}

// Export exports a graph to RDF Turtle format
func (e *RDFExporter) Export(g *graph.Graph, w io.Writer) error {
	// Validate graph
	if err := g.Validate(); err != nil {
		return fmt.Errorf("graph validation failed: %w", err)
	}

	// Write prefixes
	if err := e.writePrefixes(w); err != nil {
		return fmt.Errorf("failed to write prefixes: %w", err)
	}

	// Write graph metadata
	if err := e.writeGraphMetadata(g, w); err != nil {
		return fmt.Errorf("failed to write graph metadata: %w", err)
	}

	// Write nodes
	if err := e.writeNodes(g, w); err != nil {
		return fmt.Errorf("failed to write nodes: %w", err)
	}

	// Write edges
	if err := e.writeEdges(g, w); err != nil {
		return fmt.Errorf("failed to write edges: %w", err)
	}

	return nil
}

// writePrefixes writes RDF namespace prefixes
func (e *RDFExporter) writePrefixes(w io.Writer) error {
	prefixes := []string{
		fmt.Sprintf("@prefix : <%s> .", e.Namespace),
		"@prefix rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#> .",
		"@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .",
		"@prefix xsd: <http://www.w3.org/2001/XMLSchema#> .",
		"@prefix udml: <http://udml.org/vocab#> .",
		"",
	}

	for _, prefix := range prefixes {
		if _, err := fmt.Fprintln(w, prefix); err != nil {
			return err
		}
	}
	return nil
}

// writeGraphMetadata writes graph-level metadata
func (e *RDFExporter) writeGraphMetadata(g *graph.Graph, w io.Writer) error {
	lines := []string{
		fmt.Sprintf("# Graph: %s", g.Name),
		fmt.Sprintf("# ID: %s", g.ID),
		fmt.Sprintf("# Version: %s", g.Version),
		fmt.Sprintf("# Nodes: %d", len(g.Nodes)),
		fmt.Sprintf("# Edges: %d", len(g.Edges)),
		"",
	}

	for _, line := range lines {
		if _, err := fmt.Fprintln(w, line); err != nil {
			return err
		}
	}

	// Write graph as a resource
	graphURI := e.sanitizeURI(fmt.Sprintf("graph/%s", g.ID))
	if _, err := fmt.Fprintf(w, ":%s\n", graphURI); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "  rdf:type udml:Graph ;\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "  rdfs:label \"%s\" ;\n", e.escapeString(g.Name)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "  udml:version \"%s\" ;\n", g.Version); err != nil {
		return err
	}
	if e.IncludeTimestamps {
		if _, err := fmt.Fprintf(w, "  udml:createdAt \"%s\"^^xsd:dateTime ;\n", g.CreatedAt.Format(time.RFC3339)); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "  udml:updatedAt \"%s\"^^xsd:dateTime ;\n", g.UpdatedAt.Format(time.RFC3339)); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(w, "  udml:nodeCount %d ;\n", len(g.Nodes)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "  udml:edgeCount %d .\n\n", len(g.Edges)); err != nil {
		return err
	}

	return nil
}

// writeNodes writes all nodes as RDF resources
func (e *RDFExporter) writeNodes(g *graph.Graph, w io.Writer) error {
	if _, err := fmt.Fprintln(w, "# Nodes"); err != nil {
		return err
	}

	for _, node := range g.Nodes {
		if err := e.writeNode(&node, w); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintln(w, ""); err != nil {
		return err
	}
	return nil
}

// writeNode writes a single node as an RDF resource
func (e *RDFExporter) writeNode(node *graph.Node, w io.Writer) error {
	nodeURI := e.sanitizeURI(fmt.Sprintf("node/%s", node.ID))

	if _, err := fmt.Fprintf(w, ":%s\n", nodeURI); err != nil {
		return err
	}

	// Write types (labels)
	for _, label := range node.Labels {
		labelType := e.sanitizeURI(label)
		if _, err := fmt.Fprintf(w, "  rdf:type :%s ;\n", labelType); err != nil {
			return err
		}
	}

	// Write confidence
	if e.IncludeConfidence && node.Confidence > 0 {
		if _, err := fmt.Fprintf(w, "  udml:confidence \"%.4f\"^^xsd:double ;\n", node.Confidence); err != nil {
			return err
		}
	}

	// Write provenance
	if e.IncludeProvenance {
		if node.SourceElementID != "" {
			if _, err := fmt.Fprintf(w, "  udml:sourceElementID \"%s\" ;\n", e.escapeString(node.SourceElementID)); err != nil {
				return err
			}
		}
		if node.SourceDocID != "" {
			if _, err := fmt.Fprintf(w, "  udml:sourceDocID \"%s\" ;\n", e.escapeString(node.SourceDocID)); err != nil {
				return err
			}
		}
	}

	// Write domain
	if e.IncludeDomains && node.Domain != "" {
		if _, err := fmt.Fprintf(w, "  udml:domain \"%s\" ;\n", e.escapeString(node.Domain)); err != nil {
			return err
		}
	}

	// Write timestamps
	if e.IncludeTimestamps && !node.CreatedAt.IsZero() {
		if _, err := fmt.Fprintf(w, "  udml:createdAt \"%s\"^^xsd:dateTime ;\n", node.CreatedAt.Format(time.RFC3339)); err != nil {
			return err
		}
		if !node.UpdatedAt.IsZero() {
			if _, err := fmt.Fprintf(w, "  udml:updatedAt \"%s\"^^xsd:dateTime ;\n", node.UpdatedAt.Format(time.RFC3339)); err != nil {
				return err
			}
		}
	}

	// Write properties
	if err := e.writeProperties(node.Properties, w); err != nil {
		return err
	}

	// Terminate with period
	if _, err := fmt.Fprintln(w, "."); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, ""); err != nil {
		return err
	}

	return nil
}

// writeEdges writes all edges as RDF relationships
func (e *RDFExporter) writeEdges(g *graph.Graph, w io.Writer) error {
	if _, err := fmt.Fprintln(w, "# Edges"); err != nil {
		return err
	}

	for _, edge := range g.Edges {
		if err := e.writeEdge(&edge, w); err != nil {
			return err
		}
	}

	return nil
}

// writeEdge writes a single edge as an RDF relationship
func (e *RDFExporter) writeEdge(edge *graph.Edge, w io.Writer) error {
	sourceURI := e.sanitizeURI(fmt.Sprintf("node/%s", edge.SourceID))
	targetURI := e.sanitizeURI(fmt.Sprintf("node/%s", edge.TargetID))
	relationshipType := e.sanitizeURI(edge.Type)

	// Simple relationship (direct predicate)
	if _, err := fmt.Fprintf(w, ":%s :%s :%s .\n", sourceURI, relationshipType, targetURI); err != nil {
		return err
	}

	// If edge has additional properties, confidence, or provenance, create reified statement
	hasMetadata := (e.IncludeConfidence && edge.Confidence > 0) ||
		(e.IncludeProvenance && (edge.SourceElementID != "" || edge.Evidence != "")) ||
		(e.IncludeDomains && edge.Domain != "") ||
		len(edge.Properties) > 0

	if hasMetadata {
		edgeURI := e.sanitizeURI(fmt.Sprintf("edge/%s", edge.ID))
		if _, err := fmt.Fprintf(w, ":%s\n", edgeURI); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "  rdf:type rdf:Statement ;\n"); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "  rdf:subject :%s ;\n", sourceURI); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "  rdf:predicate :%s ;\n", relationshipType); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "  rdf:object :%s ;\n", targetURI); err != nil {
			return err
		}

		// Write confidence
		if e.IncludeConfidence && edge.Confidence > 0 {
			if _, err := fmt.Fprintf(w, "  udml:confidence \"%.4f\"^^xsd:double ;\n", edge.Confidence); err != nil {
				return err
			}
		}

		// Write provenance
		if e.IncludeProvenance {
			if edge.SourceElementID != "" {
				if _, err := fmt.Fprintf(w, "  udml:sourceElementID \"%s\" ;\n", e.escapeString(edge.SourceElementID)); err != nil {
					return err
				}
			}
			if edge.Evidence != "" {
				if _, err := fmt.Fprintf(w, "  udml:evidence \"%s\" ;\n", e.escapeString(edge.Evidence)); err != nil {
					return err
				}
			}
		}

		// Write domain
		if e.IncludeDomains && edge.Domain != "" {
			if _, err := fmt.Fprintf(w, "  udml:domain \"%s\" ;\n", e.escapeString(edge.Domain)); err != nil {
				return err
			}
		}

		// Write properties
		if err := e.writeProperties(edge.Properties, w); err != nil {
			return err
		}

		if _, err := fmt.Fprintln(w, "."); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintln(w, ""); err != nil {
		return err
	}

	return nil
}

// writeProperties writes property map as RDF predicates
func (e *RDFExporter) writeProperties(props map[string]interface{}, w io.Writer) error {
	for key, value := range props {
		propName := e.sanitizeURI(key)
		propValue, propType := e.formatPropertyValue(value)

		if propType == "uri" {
			if _, err := fmt.Fprintf(w, "  :%s :%s ;\n", propName, propValue); err != nil {
				return err
			}
		} else if propType == "string" {
			if _, err := fmt.Fprintf(w, "  :%s \"%s\" ;\n", propName, e.escapeString(propValue)); err != nil {
				return err
			}
		} else {
			if _, err := fmt.Fprintf(w, "  :%s \"%s\"^^xsd:%s ;\n", propName, propValue, propType); err != nil {
				return err
			}
		}
	}
	return nil
}

// formatPropertyValue formats a property value for RDF output
func (e *RDFExporter) formatPropertyValue(value interface{}) (string, string) {
	switch v := value.(type) {
	case string:
		return v, "string"
	case int, int32, int64:
		return fmt.Sprintf("%d", v), "integer"
	case float32, float64:
		return fmt.Sprintf("%f", v), "double"
	case bool:
		return fmt.Sprintf("%t", v), "boolean"
	default:
		return fmt.Sprintf("%v", v), "string"
	}
}

// sanitizeURI sanitizes a string for use as an RDF URI local name
func (e *RDFExporter) sanitizeURI(s string) string {
	// Replace spaces and special characters with underscores
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "-", "_")
	s = strings.ReplaceAll(s, ".", "_")
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, ":", "_")
	// Remove any other non-alphanumeric characters
	var result strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// escapeString escapes special characters in RDF string literals
func (e *RDFExporter) escapeString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	s = strings.ReplaceAll(s, "\t", "\\t")
	return s
}

// Register the RDF exporter on package init
func init() {
	RegisterExporter(NewRDFExporter())
}
