package export

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/kennethstott/doculyzer-go-conversion/internal/graph"
)

// RDFFormat specifies the RDF serialization format
type RDFFormat string

const (
	FormatNTriples RDFFormat = "n-triples" // N-Triples format (.nt)
	FormatTurtle   RDFFormat = "turtle"    // Turtle format (.ttl)
	FormatRDFXML   RDFFormat = "rdf-xml"   // RDF/XML format (.rdf)
)

// RDFExporter converts graphs to RDF format for SPARQL querying
type RDFExporter struct {
	baseURI     string      // Base URI for RDF resources
	format      RDFFormat   // Output format
	prefixes    map[string]string // Namespace prefixes
	includeTime bool        // Include timestamps as RDF properties
}

// RDFExportOptions configures RDF export behavior
type RDFExportOptions struct {
	BaseURI     string            // Base URI (default: http://example.org/graph/)
	Format      RDFFormat         // Output format (default: Turtle)
	Prefixes    map[string]string // Custom namespace prefixes
	IncludeTime bool              // Include createdAt/updatedAt (default: true)
}

// DefaultRDFExportOptions returns sensible defaults
func DefaultRDFExportOptions() RDFExportOptions {
	return RDFExportOptions{
		BaseURI:     "http://example.org/graph/",
		Format:      FormatTurtle,
		Prefixes:    defaultPrefixes(),
		IncludeTime: true,
	}
}

// defaultPrefixes returns standard RDF namespace prefixes
func defaultPrefixes() map[string]string {
	return map[string]string{
		"rdf":  "http://www.w3.org/1999/02/22-rdf-syntax-ns#",
		"rdfs": "http://www.w3.org/2000/01/rdf-schema#",
		"xsd":  "http://www.w3.org/2001/XMLSchema#",
		"kg":   "http://example.org/graph/",
	}
}

// NewRDFExporter creates a new RDF exporter with options
func NewRDFExporter(opts RDFExportOptions) *RDFExporter {
	if opts.BaseURI == "" {
		opts.BaseURI = "http://example.org/graph/"
	}
	if opts.Format == "" {
		opts.Format = FormatTurtle
	}
	if opts.Prefixes == nil {
		opts.Prefixes = defaultPrefixes()
	}

	return &RDFExporter{
		baseURI:     opts.BaseURI,
		format:      opts.Format,
		prefixes:    opts.Prefixes,
		includeTime: opts.IncludeTime,
	}
}

// ExportGraph converts a graph to RDF and writes to the writer
func (e *RDFExporter) ExportGraph(g *graph.Graph, w io.Writer) error {
	switch e.format {
	case FormatNTriples:
		return e.exportNTriples(g, w)
	case FormatTurtle:
		return e.exportTurtle(g, w)
	case FormatRDFXML:
		return e.exportRDFXML(g, w)
	default:
		return fmt.Errorf("unsupported RDF format: %s", e.format)
	}
}

// exportNTriples exports graph in N-Triples format
func (e *RDFExporter) exportNTriples(g *graph.Graph, w io.Writer) error {
	// N-Triples: Simple line-based format
	// <subject> <predicate> <object> .

	// Export nodes
	for _, node := range g.Nodes {
		triples := e.nodeToNTriples(node)
		for _, triple := range triples {
			if _, err := fmt.Fprintln(w, triple); err != nil {
				return err
			}
		}
	}

	// Export edges
	for _, edge := range g.Edges {
		triples := e.edgeToNTriples(edge)
		for _, triple := range triples {
			if _, err := fmt.Fprintln(w, triple); err != nil {
				return err
			}
		}
	}

	return nil
}

// exportTurtle exports graph in Turtle format
func (e *RDFExporter) exportTurtle(g *graph.Graph, w io.Writer) error {
	// Write prefixes
	if err := e.writeTurtlePrefixes(w); err != nil {
		return err
	}

	// Write graph metadata
	if _, err := fmt.Fprintf(w, "\n# Graph: %s (version %s)\n", g.Name, g.Version); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "# Nodes: %d, Edges: %d\n\n", len(g.Nodes), len(g.Edges)); err != nil {
		return err
	}

	// Export nodes
	if _, err := fmt.Fprintln(w, "# Nodes"); err != nil {
		return err
	}
	for _, node := range g.Nodes {
		if err := e.nodeToTurtle(node, w); err != nil {
			return err
		}
	}

	// Export edges
	if _, err := fmt.Fprintln(w, "\n# Edges"); err != nil {
		return err
	}
	for _, edge := range g.Edges {
		if err := e.edgeToTurtle(edge, w); err != nil {
			return err
		}
	}

	return nil
}

// exportRDFXML exports graph in RDF/XML format
func (e *RDFExporter) exportRDFXML(g *graph.Graph, w io.Writer) error {
	// Write XML header
	if _, err := fmt.Fprintln(w, `<?xml version="1.0" encoding="UTF-8"?>`); err != nil {
		return err
	}

	// Write RDF root element with namespaces
	if _, err := fmt.Fprint(w, `<rdf:RDF`); err != nil {
		return err
	}
	for prefix, uri := range e.prefixes {
		if _, err := fmt.Fprintf(w, `
    xmlns:%s="%s"`, prefix, uri); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(w, ">"); err != nil {
		return err
	}

	// Export nodes as RDF resources
	for _, node := range g.Nodes {
		if err := e.nodeToRDFXML(node, w); err != nil {
			return err
		}
	}

	// Export edges as RDF statements
	for _, edge := range g.Edges {
		if err := e.edgeToRDFXML(edge, w); err != nil {
			return err
		}
	}

	// Close RDF root
	if _, err := fmt.Fprintln(w, "</rdf:RDF>"); err != nil {
		return err
	}

	return nil
}

// writeTurtlePrefixes writes namespace prefixes for Turtle format
func (e *RDFExporter) writeTurtlePrefixes(w io.Writer) error {
	// Sort prefixes for consistent output
	var prefixList []string
	for prefix := range e.prefixes {
		prefixList = append(prefixList, prefix)
	}
	sort.Strings(prefixList)

	for _, prefix := range prefixList {
		uri := e.prefixes[prefix]
		if _, err := fmt.Fprintf(w, "@prefix %s: <%s> .\n", prefix, uri); err != nil {
			return err
		}
	}

	return nil
}

// nodeToNTriples converts a node to N-Triples statements
func (e *RDFExporter) nodeToNTriples(node graph.Node) []string {
	var triples []string
	nodeURI := e.nodeURI(node.ID)

	// rdf:type for each label
	for _, label := range node.Labels {
		triples = append(triples, fmt.Sprintf("<%s> <http://www.w3.org/1999/02/22-rdf-syntax-ns#type> <%s> .",
			nodeURI, e.baseURI+"class/"+sanitizeURI(label)))
	}

	// Node properties
	for key, value := range node.Properties {
		triple := e.propertyToNTriple(nodeURI, key, value)
		if triple != "" {
			triples = append(triples, triple)
		}
	}

	// Domain
	if node.Domain != "" {
		triples = append(triples, fmt.Sprintf("<%s> <%sdomain> \"%s\" .",
			nodeURI, e.baseURI, escapeString(node.Domain)))
	}

	// Confidence
	triples = append(triples, fmt.Sprintf("<%s> <%sconfidence> \"%f\"^^<http://www.w3.org/2001/XMLSchema#decimal> .",
		nodeURI, e.baseURI, node.Confidence))

	// Timestamps
	if e.includeTime {
		if !node.CreatedAt.IsZero() {
			triples = append(triples, fmt.Sprintf("<%s> <%screatedAt> \"%s\"^^<http://www.w3.org/2001/XMLSchema#dateTime> .",
				nodeURI, e.baseURI, node.CreatedAt.Format(time.RFC3339)))
		}
		if !node.UpdatedAt.IsZero() {
			triples = append(triples, fmt.Sprintf("<%s> <%supdatedAt> \"%s\"^^<http://www.w3.org/2001/XMLSchema#dateTime> .",
				nodeURI, e.baseURI, node.UpdatedAt.Format(time.RFC3339)))
		}
	}

	return triples
}

// edgeToNTriples converts an edge to N-Triples statements
func (e *RDFExporter) edgeToNTriples(edge graph.Edge) []string {
	var triples []string
	sourceURI := e.nodeURI(edge.SourceID)
	targetURI := e.nodeURI(edge.TargetID)
	edgeURI := e.edgeURI(edge.ID)

	// Main relationship triple
	predicateURI := e.baseURI + "relation/" + sanitizeURI(edge.Type)
	triples = append(triples, fmt.Sprintf("<%s> <%s> <%s> .",
		sourceURI, predicateURI, targetURI))

	// Reified edge for properties (as a resource)
	triples = append(triples, fmt.Sprintf("<%s> <http://www.w3.org/1999/02/22-rdf-syntax-ns#type> <http://www.w3.org/1999/02/22-rdf-syntax-ns#Statement> .",
		edgeURI))
	triples = append(triples, fmt.Sprintf("<%s> <http://www.w3.org/1999/02/22-rdf-syntax-ns#subject> <%s> .",
		edgeURI, sourceURI))
	triples = append(triples, fmt.Sprintf("<%s> <http://www.w3.org/1999/02/22-rdf-syntax-ns#predicate> <%s> .",
		edgeURI, predicateURI))
	triples = append(triples, fmt.Sprintf("<%s> <http://www.w3.org/1999/02/22-rdf-syntax-ns#object> <%s> .",
		edgeURI, targetURI))

	// Edge properties
	for key, value := range edge.Properties {
		triple := e.propertyToNTriple(edgeURI, key, value)
		if triple != "" {
			triples = append(triples, triple)
		}
	}

	// Edge confidence
	triples = append(triples, fmt.Sprintf("<%s> <%sconfidence> \"%f\"^^<http://www.w3.org/2001/XMLSchema#decimal> .",
		edgeURI, e.baseURI, edge.Confidence))

	return triples
}

// nodeToTurtle writes a node in Turtle format
func (e *RDFExporter) nodeToTurtle(node graph.Node, w io.Writer) error {
	nodeID := "kg:" + sanitizeURI(node.ID)

	// Start node description
	if _, err := fmt.Fprintf(w, "\n%s\n", nodeID); err != nil {
		return err
	}

	// rdf:type for labels
	for i, label := range node.Labels {
		prefix := "    a"
		if i > 0 {
			prefix = "      "
		}
		if _, err := fmt.Fprintf(w, "%s kg:%s", prefix, sanitizeURI(label)); err != nil {
			return err
		}
		if i < len(node.Labels)-1 || len(node.Properties) > 0 || node.Domain != "" || e.includeTime {
			if _, err := fmt.Fprintln(w, " ;"); err != nil {
				return err
			}
		} else {
			if _, err := fmt.Fprintln(w, " ."); err != nil {
				return err
			}
		}
	}

	// Properties
	propCount := 0
	totalProps := len(node.Properties)
	if node.Domain != "" {
		totalProps++
	}
	totalProps++ // confidence
	if e.includeTime && !node.CreatedAt.IsZero() {
		totalProps++
	}
	if e.includeTime && !node.UpdatedAt.IsZero() {
		totalProps++
	}

	for key, value := range node.Properties {
		propCount++
		if err := e.propertyToTurtle(w, key, value, propCount < totalProps); err != nil {
			return err
		}
	}

	// Domain
	if node.Domain != "" {
		propCount++
		terminator := ";"
		if propCount >= totalProps {
			terminator = "."
		}
		if _, err := fmt.Fprintf(w, "    kg:domain \"%s\" %s\n", escapeString(node.Domain), terminator); err != nil {
			return err
		}
	}

	// Confidence
	propCount++
	terminator := ";"
	if propCount >= totalProps {
		terminator = "."
	}
	if _, err := fmt.Fprintf(w, "    kg:confidence %f %s\n", node.Confidence, terminator); err != nil {
		return err
	}

	// Timestamps
	if e.includeTime && !node.CreatedAt.IsZero() {
		propCount++
		terminator := ";"
		if propCount >= totalProps {
			terminator = "."
		}
		if _, err := fmt.Fprintf(w, "    kg:createdAt \"%s\"^^xsd:dateTime %s\n",
			node.CreatedAt.Format(time.RFC3339), terminator); err != nil {
			return err
		}
	}

	if e.includeTime && !node.UpdatedAt.IsZero() {
		if _, err := fmt.Fprintf(w, "    kg:updatedAt \"%s\"^^xsd:dateTime .\n",
			node.UpdatedAt.Format(time.RFC3339)); err != nil {
			return err
		}
	}

	return nil
}

// edgeToTurtle writes an edge in Turtle format
func (e *RDFExporter) edgeToTurtle(edge graph.Edge, w io.Writer) error {
	sourceID := "kg:" + sanitizeURI(edge.SourceID)
	targetID := "kg:" + sanitizeURI(edge.TargetID)
	edgeID := "kg:edge_" + sanitizeURI(edge.ID)
	relationType := "kg:" + sanitizeURI(edge.Type)

	// Direct relationship
	if _, err := fmt.Fprintf(w, "\n%s %s %s .\n", sourceID, relationType, targetID); err != nil {
		return err
	}

	// Reified edge for properties
	if len(edge.Properties) > 0 || edge.Confidence > 0 {
		if _, err := fmt.Fprintf(w, "\n%s\n", edgeID); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w, "    a rdf:Statement ;"); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "    rdf:subject %s ;\n", sourceID); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "    rdf:predicate %s ;\n", relationType); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "    rdf:object %s ;\n", targetID); err != nil {
			return err
		}

		// Edge properties
		propCount := 0
		totalProps := len(edge.Properties) + 1 // +1 for confidence
		for key, value := range edge.Properties {
			propCount++
			if err := e.propertyToTurtle(w, key, value, propCount < totalProps); err != nil {
				return err
			}
		}

		// Confidence
		if _, err := fmt.Fprintf(w, "    kg:confidence %f .\n", edge.Confidence); err != nil {
			return err
		}
	}

	return nil
}

// nodeToRDFXML writes a node in RDF/XML format
func (e *RDFExporter) nodeToRDFXML(node graph.Node, w io.Writer) error {
	// Primary label as rdf:type
	primaryLabel := "Entity"
	if len(node.Labels) > 0 {
		primaryLabel = node.Labels[0]
	}

	if _, err := fmt.Fprintf(w, "\n  <kg:%s rdf:about=\"%s\">\n",
		sanitizeXML(primaryLabel), e.nodeURI(node.ID)); err != nil {
		return err
	}

	// Additional labels as rdf:type
	for i := 1; i < len(node.Labels); i++ {
		if _, err := fmt.Fprintf(w, "    <rdf:type rdf:resource=\"%sclass/%s\"/>\n",
			e.baseURI, sanitizeURI(node.Labels[i])); err != nil {
			return err
		}
	}

	// Properties
	for key, value := range node.Properties {
		if err := e.propertyToRDFXML(w, key, value); err != nil {
			return err
		}
	}

	// Domain
	if node.Domain != "" {
		if _, err := fmt.Fprintf(w, "    <kg:domain>%s</kg:domain>\n", escapeXML(node.Domain)); err != nil {
			return err
		}
	}

	// Confidence
	if _, err := fmt.Fprintf(w, "    <kg:confidence rdf:datatype=\"http://www.w3.org/2001/XMLSchema#decimal\">%f</kg:confidence>\n",
		node.Confidence); err != nil {
		return err
	}

	// Timestamps
	if e.includeTime && !node.CreatedAt.IsZero() {
		if _, err := fmt.Fprintf(w, "    <kg:createdAt rdf:datatype=\"http://www.w3.org/2001/XMLSchema#dateTime\">%s</kg:createdAt>\n",
			node.CreatedAt.Format(time.RFC3339)); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintf(w, "  </kg:%s>\n", sanitizeXML(primaryLabel)); err != nil {
		return err
	}

	return nil
}

// edgeToRDFXML writes an edge in RDF/XML format
func (e *RDFExporter) edgeToRDFXML(edge graph.Edge, w io.Writer) error {
	// Reified statement for edge with properties
	if _, err := fmt.Fprintf(w, "\n  <rdf:Statement rdf:about=\"%s\">\n", e.edgeURI(edge.ID)); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "    <rdf:subject rdf:resource=\"%s\"/>\n", e.nodeURI(edge.SourceID)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "    <rdf:predicate rdf:resource=\"%srelation/%s\"/>\n",
		e.baseURI, sanitizeURI(edge.Type)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "    <rdf:object rdf:resource=\"%s\"/>\n", e.nodeURI(edge.TargetID)); err != nil {
		return err
	}

	// Edge properties
	for key, value := range edge.Properties {
		if err := e.propertyToRDFXML(w, key, value); err != nil {
			return err
		}
	}

	// Confidence
	if _, err := fmt.Fprintf(w, "    <kg:confidence rdf:datatype=\"http://www.w3.org/2001/XMLSchema#decimal\">%f</kg:confidence>\n",
		edge.Confidence); err != nil {
		return err
	}

	if _, err := fmt.Fprintln(w, "  </rdf:Statement>"); err != nil {
		return err
	}

	return nil
}

// propertyToNTriple converts a property to an N-Triple statement
func (e *RDFExporter) propertyToNTriple(subjectURI, key string, value interface{}) string {
	predicateURI := e.baseURI + sanitizeURI(key)

	switch v := value.(type) {
	case string:
		return fmt.Sprintf("<%s> <%s> \"%s\" .", subjectURI, predicateURI, escapeString(v))
	case int, int64:
		return fmt.Sprintf("<%s> <%s> \"%d\"^^<http://www.w3.org/2001/XMLSchema#integer> .",
			subjectURI, predicateURI, v)
	case float64, float32:
		return fmt.Sprintf("<%s> <%s> \"%f\"^^<http://www.w3.org/2001/XMLSchema#decimal> .",
			subjectURI, predicateURI, v)
	case bool:
		return fmt.Sprintf("<%s> <%s> \"%t\"^^<http://www.w3.org/2001/XMLSchema#boolean> .",
			subjectURI, predicateURI, v)
	default:
		// Fallback to string representation
		return fmt.Sprintf("<%s> <%s> \"%v\" .", subjectURI, predicateURI, escapeString(fmt.Sprintf("%v", v)))
	}
}

// propertyToTurtle writes a property in Turtle format
func (e *RDFExporter) propertyToTurtle(w io.Writer, key string, value interface{}, hasMore bool) error {
	terminator := ";"
	if !hasMore {
		terminator = "."
	}

	predicateID := "kg:" + sanitizeURI(key)

	switch v := value.(type) {
	case string:
		_, err := fmt.Fprintf(w, "    %s \"%s\" %s\n", predicateID, escapeString(v), terminator)
		return err
	case int, int64:
		_, err := fmt.Fprintf(w, "    %s %d %s\n", predicateID, v, terminator)
		return err
	case float64, float32:
		_, err := fmt.Fprintf(w, "    %s %f %s\n", predicateID, v, terminator)
		return err
	case bool:
		_, err := fmt.Fprintf(w, "    %s %t %s\n", predicateID, v, terminator)
		return err
	default:
		_, err := fmt.Fprintf(w, "    %s \"%v\" %s\n", predicateID, escapeString(fmt.Sprintf("%v", v)), terminator)
		return err
	}
}

// propertyToRDFXML writes a property in RDF/XML format
func (e *RDFExporter) propertyToRDFXML(w io.Writer, key string, value interface{}) error {
	switch v := value.(type) {
	case string:
		_, err := fmt.Fprintf(w, "    <kg:%s>%s</kg:%s>\n", sanitizeXML(key), escapeXML(v), sanitizeXML(key))
		return err
	case int, int64:
		_, err := fmt.Fprintf(w, "    <kg:%s rdf:datatype=\"http://www.w3.org/2001/XMLSchema#integer\">%d</kg:%s>\n",
			sanitizeXML(key), v, sanitizeXML(key))
		return err
	case float64, float32:
		_, err := fmt.Fprintf(w, "    <kg:%s rdf:datatype=\"http://www.w3.org/2001/XMLSchema#decimal\">%f</kg:%s>\n",
			sanitizeXML(key), v, sanitizeXML(key))
		return err
	case bool:
		_, err := fmt.Fprintf(w, "    <kg:%s rdf:datatype=\"http://www.w3.org/2001/XMLSchema#boolean\">%t</kg:%s>\n",
			sanitizeXML(key), v, sanitizeXML(key))
		return err
	default:
		_, err := fmt.Fprintf(w, "    <kg:%s>%s</kg:%s>\n",
			sanitizeXML(key), escapeXML(fmt.Sprintf("%v", v)), sanitizeXML(key))
		return err
	}
}

// nodeURI creates a URI for a node
func (e *RDFExporter) nodeURI(nodeID string) string {
	return e.baseURI + "node/" + sanitizeURI(nodeID)
}

// edgeURI creates a URI for an edge
func (e *RDFExporter) edgeURI(edgeID string) string {
	return e.baseURI + "edge/" + sanitizeURI(edgeID)
}

// sanitizeURI makes a string safe for use in URIs
func sanitizeURI(s string) string {
	// Replace spaces and special characters with underscores
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, ":", "_")
	s = strings.ReplaceAll(s, "#", "_")
	s = strings.ReplaceAll(s, "?", "_")
	s = strings.ReplaceAll(s, "&", "_")
	s = strings.ReplaceAll(s, "=", "_")
	return s
}

// sanitizeXML makes a string safe for use in XML element names
func sanitizeXML(s string) string {
	// Replace spaces and special characters for XML element names
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, ":", "_")
	s = strings.ReplaceAll(s, "<", "_")
	s = strings.ReplaceAll(s, ">", "_")
	s = strings.ReplaceAll(s, "&", "_")
	return s
}

// escapeString escapes special characters in RDF strings
func escapeString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	s = strings.ReplaceAll(s, "\t", "\\t")
	return s
}

// escapeXML escapes special characters for XML content
func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}
