package export

import (
	"fmt"
	"io"
	"strings"

	"github.com/kennethstott/doculyzer-go-conversion/internal/graph"
)

// CypherExporter exports graphs to Neo4j Cypher script format
type CypherExporter struct {
	IncludeProvenance bool // Include UDML provenance
	IncludeConfidence bool // Include confidence scores
	IncludeDomains    bool // Include data mesh domains
	IncludeTimestamps bool // Include created/updated timestamps
	BatchSize         int  // Number of nodes/edges per transaction
	UseMerge          bool // Use MERGE instead of CREATE (prevents duplicates)
}

// NewCypherExporter creates a new Cypher exporter with default settings
func NewCypherExporter() *CypherExporter {
	return &CypherExporter{
		IncludeProvenance: true,
		IncludeConfidence: true,
		IncludeDomains:    true,
		IncludeTimestamps: true,
		BatchSize:         1000,
		UseMerge:          true,
	}
}

// Name returns the exporter identifier
func (e *CypherExporter) Name() string {
	return "cypher"
}

// Description returns a human-readable description
func (e *CypherExporter) Description() string {
	return "Cypher script format - Neo4j graph database import"
}

// FileExtension returns the recommended file extension
func (e *CypherExporter) FileExtension() string {
	return ".cypher"
}

// SupportsFeature checks if the exporter supports a specific feature
func (e *CypherExporter) SupportsFeature(feature string) bool {
	switch feature {
	case FeatureProvenance, FeatureConfidence, FeatureDomains,
		FeatureTimestamps, FeatureProperties, FeatureMultiLabel, FeatureDirectedEdges:
		return true
	default:
		return false
	}
}

// Export exports a graph to Cypher script format
func (e *CypherExporter) Export(g *graph.Graph, w io.Writer) error {
	// Validate graph
	if err := g.Validate(); err != nil {
		return fmt.Errorf("graph validation failed: %w", err)
	}

	// Write header comments
	if err := e.writeHeader(g, w); err != nil {
		return err
	}

	// Write constraint creation statements
	if err := e.writeConstraints(g, w); err != nil {
		return err
	}

	// Write index creation statements
	if err := e.writeIndexes(g, w); err != nil {
		return err
	}

	// Write nodes in batches
	if err := e.writeNodes(g, w); err != nil {
		return err
	}

	// Write edges in batches
	if err := e.writeEdges(g, w); err != nil {
		return err
	}

	return nil
}

// writeHeader writes script header with metadata
func (e *CypherExporter) writeHeader(g *graph.Graph, w io.Writer) error {
	header := fmt.Sprintf(`// Neo4j Cypher Import Script
// Graph: %s
// ID: %s
// Version: %s
// Nodes: %d
// Edges: %d
// Generated: %s

`, g.Name, g.ID, g.Version, len(g.Nodes), len(g.Edges), g.UpdatedAt.Format("2006-01-02 15:04:05"))

	_, err := w.Write([]byte(header))
	return err
}

// writeConstraints writes constraint creation statements
func (e *CypherExporter) writeConstraints(g *graph.Graph, w io.Writer) error {
	// Collect unique labels
	labelSet := make(map[string]bool)
	for _, node := range g.Nodes {
		for _, label := range node.Labels {
			labelSet[label] = true
		}
	}

	if len(labelSet) == 0 {
		return nil
	}

	if _, err := w.Write([]byte("// Create constraints\n")); err != nil {
		return err
	}

	// Create uniqueness constraints on node IDs for each label
	for label := range labelSet {
		constraint := fmt.Sprintf("CREATE CONSTRAINT IF NOT EXISTS FOR (n:%s) REQUIRE n.id IS UNIQUE;\n",
			e.sanitizeLabel(label))
		if _, err := w.Write([]byte(constraint)); err != nil {
			return err
		}
	}

	if _, err := w.Write([]byte("\n")); err != nil {
		return err
	}

	return nil
}

// writeIndexes writes index creation statements
func (e *CypherExporter) writeIndexes(g *graph.Graph, w io.Writer) error {
	// Collect unique labels
	labelSet := make(map[string]bool)
	for _, node := range g.Nodes {
		for _, label := range node.Labels {
			labelSet[label] = true
		}
	}

	if len(labelSet) == 0 {
		return nil
	}

	if _, err := w.Write([]byte("// Create indexes\n")); err != nil {
		return err
	}

	// Create indexes on commonly queried properties
	for label := range labelSet {
		if e.IncludeConfidence {
			index := fmt.Sprintf("CREATE INDEX IF NOT EXISTS FOR (n:%s) ON (n.confidence);\n",
				e.sanitizeLabel(label))
			if _, err := w.Write([]byte(index)); err != nil {
				return err
			}
		}
		if e.IncludeDomains {
			index := fmt.Sprintf("CREATE INDEX IF NOT EXISTS FOR (n:%s) ON (n.domain);\n",
				e.sanitizeLabel(label))
			if _, err := w.Write([]byte(index)); err != nil {
				return err
			}
		}
	}

	if _, err := w.Write([]byte("\n")); err != nil {
		return err
	}

	return nil
}

// writeNodes writes node creation statements in batches
func (e *CypherExporter) writeNodes(g *graph.Graph, w io.Writer) error {
	if _, err := w.Write([]byte("// Create nodes\n")); err != nil {
		return err
	}

	for i := 0; i < len(g.Nodes); i += e.BatchSize {
		end := i + e.BatchSize
		if end > len(g.Nodes) {
			end = len(g.Nodes)
		}

		// Start transaction
		if _, err := w.Write([]byte(":begin\n")); err != nil {
			return err
		}

		// Write nodes in batch
		for j := i; j < end; j++ {
			if err := e.writeNode(&g.Nodes[j], w); err != nil {
				return err
			}
		}

		// Commit transaction
		if _, err := w.Write([]byte(":commit\n\n")); err != nil {
			return err
		}
	}

	return nil
}

// writeNode writes a single node creation statement
func (e *CypherExporter) writeNode(node *graph.Node, w io.Writer) error {
	// Build labels string
	labelsStr := ""
	for _, label := range node.Labels {
		labelsStr += ":" + e.sanitizeLabel(label)
	}
	if labelsStr == "" {
		labelsStr = ":Node" // Default label if none specified
	}

	// Build properties map
	props := map[string]interface{}{
		"id": node.ID,
	}

	// Add confidence
	if e.IncludeConfidence && node.Confidence > 0 {
		props["confidence"] = node.Confidence
	}

	// Add provenance
	if e.IncludeProvenance {
		if node.SourceElementID != "" {
			props["sourceElementID"] = node.SourceElementID
		}
		if node.SourceDocID != "" {
			props["sourceDocID"] = node.SourceDocID
		}
	}

	// Add domain
	if e.IncludeDomains && node.Domain != "" {
		props["domain"] = node.Domain
	}

	// Add timestamps
	if e.IncludeTimestamps {
		if !node.CreatedAt.IsZero() {
			props["createdAt"] = node.CreatedAt.Format("2006-01-02T15:04:05Z07:00")
		}
		if !node.UpdatedAt.IsZero() {
			props["updatedAt"] = node.UpdatedAt.Format("2006-01-02T15:04:05Z07:00")
		}
	}

	// Add custom properties
	for key, value := range node.Properties {
		props[key] = value
	}

	// Build Cypher statement
	operation := "CREATE"
	if e.UseMerge {
		operation = "MERGE"
	}

	propsStr := e.formatProperties(props)
	cypher := fmt.Sprintf("%s (n%s %s);\n", operation, labelsStr, propsStr)

	_, err := w.Write([]byte(cypher))
	return err
}

// writeEdges writes edge creation statements in batches
func (e *CypherExporter) writeEdges(g *graph.Graph, w io.Writer) error {
	if _, err := w.Write([]byte("// Create relationships\n")); err != nil {
		return err
	}

	for i := 0; i < len(g.Edges); i += e.BatchSize {
		end := i + e.BatchSize
		if end > len(g.Edges) {
			end = len(g.Edges)
		}

		// Start transaction
		if _, err := w.Write([]byte(":begin\n")); err != nil {
			return err
		}

		// Write edges in batch
		for j := i; j < end; j++ {
			if err := e.writeEdge(&g.Edges[j], w); err != nil {
				return err
			}
		}

		// Commit transaction
		if _, err := w.Write([]byte(":commit\n\n")); err != nil {
			return err
		}
	}

	return nil
}

// writeEdge writes a single edge creation statement
func (e *CypherExporter) writeEdge(edge *graph.Edge, w io.Writer) error {
	// Build properties map
	props := map[string]interface{}{}

	// Add confidence
	if e.IncludeConfidence && edge.Confidence > 0 {
		props["confidence"] = edge.Confidence
	}

	// Add provenance
	if e.IncludeProvenance {
		if edge.SourceElementID != "" {
			props["sourceElementID"] = edge.SourceElementID
		}
		if edge.Evidence != "" {
			props["evidence"] = edge.Evidence
		}
	}

	// Add domain
	if e.IncludeDomains && edge.Domain != "" {
		props["domain"] = edge.Domain
	}

	// Add timestamps
	if e.IncludeTimestamps && !edge.CreatedAt.IsZero() {
		props["createdAt"] = edge.CreatedAt.Format("2006-01-02T15:04:05Z07:00")
	}

	// Add custom properties
	for key, value := range edge.Properties {
		props[key] = value
	}

	// Build Cypher statement
	relType := e.sanitizeLabel(edge.Type)
	propsStr := ""
	if len(props) > 0 {
		propsStr = " " + e.formatProperties(props)
	}

	cypher := fmt.Sprintf("MATCH (a {id: %s}), (b {id: %s})\n",
		e.formatValue(edge.SourceID),
		e.formatValue(edge.TargetID))

	operation := "CREATE"
	if e.UseMerge {
		operation = "MERGE"
	}

	cypher += fmt.Sprintf("%s (a)-[:%s%s]->(b);\n", operation, relType, propsStr)

	_, err := w.Write([]byte(cypher))
	return err
}

// formatProperties formats a property map as a Cypher property map string
func (e *CypherExporter) formatProperties(props map[string]interface{}) string {
	if len(props) == 0 {
		return "{}"
	}

	parts := make([]string, 0, len(props))
	for key, value := range props {
		parts = append(parts, fmt.Sprintf("%s: %s", e.sanitizePropertyName(key), e.formatValue(value)))
	}

	return "{" + strings.Join(parts, ", ") + "}"
}

// formatValue formats a value for Cypher
func (e *CypherExporter) formatValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		return fmt.Sprintf("\"%s\"", e.escapeString(v))
	case int, int32, int64:
		return fmt.Sprintf("%d", v)
	case float32, float64:
		return fmt.Sprintf("%f", v)
	case bool:
		return fmt.Sprintf("%t", v)
	default:
		return fmt.Sprintf("\"%v\"", v)
	}
}

// sanitizeLabel sanitizes a label for Cypher (removes invalid characters)
func (e *CypherExporter) sanitizeLabel(label string) string {
	// Replace spaces and special characters with underscores
	label = strings.ReplaceAll(label, " ", "_")
	label = strings.ReplaceAll(label, "-", "_")
	label = strings.ReplaceAll(label, ".", "_")
	label = strings.ReplaceAll(label, "/", "_")

	// Remove any other non-alphanumeric characters
	var result strings.Builder
	for _, r := range label {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			result.WriteRune(r)
		}
	}

	sanitized := result.String()
	if sanitized == "" {
		return "Node"
	}

	// Ensure starts with letter
	if sanitized[0] >= '0' && sanitized[0] <= '9' {
		sanitized = "N" + sanitized
	}

	return sanitized
}

// sanitizePropertyName sanitizes a property name for Cypher
func (e *CypherExporter) sanitizePropertyName(name string) string {
	// Use backticks to allow any property name
	return "`" + strings.ReplaceAll(name, "`", "\\`") + "`"
}

// escapeString escapes special characters in Cypher strings
func (e *CypherExporter) escapeString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	s = strings.ReplaceAll(s, "\t", "\\t")
	return s
}

// Register the Cypher exporter on package init
func init() {
	RegisterExporter(NewCypherExporter())
}
