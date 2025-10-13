package graph

import (
	"encoding/json"
	"fmt"
	"time"
)

// Graph represents a format-agnostic knowledge graph
// This is the universal representation that can be exported to RDF, Neo4j, GraphML, etc.
type Graph struct {
	ID        string            `json:"id"`         // Unique graph identifier
	Name      string            `json:"name"`       // Graph name
	Version   string            `json:"version"`    // Graph version
	Nodes     []Node            `json:"nodes"`      // All nodes in the graph
	Edges     []Edge            `json:"edges"`      // All edges in the graph
	Metadata  map[string]interface{} `json:"metadata"` // Additional metadata
	CreatedAt time.Time         `json:"created_at"` // Creation timestamp
	UpdatedAt time.Time         `json:"updated_at"` // Last update timestamp
}

// Node represents an entity in the knowledge graph
type Node struct {
	ID         string                 `json:"id"`           // Unique node identifier
	Labels     []string               `json:"labels"`       // Node labels/types (e.g., ["Person", "Entity"])
	Properties map[string]interface{} `json:"properties"`   // Node properties

	// UDML provenance - tracks where this node came from
	SourceElementID string  `json:"source_element_id,omitempty"` // UDML element ID
	SourceDocID     string  `json:"source_doc_id,omitempty"`     // UDML document ID
	Domain          string  `json:"domain,omitempty"`            // Data mesh domain
	Confidence      float64 `json:"confidence"`                  // Extraction confidence (0.0-1.0)

	// Graph metadata
	CreatedAt time.Time `json:"created_at"` // Creation timestamp
	UpdatedAt time.Time `json:"updated_at"` // Last update timestamp
}

// Edge represents a relationship in the knowledge graph
type Edge struct {
	ID         string                 `json:"id"`         // Unique edge identifier
	Type       string                 `json:"type"`       // Relationship type (e.g., "WORKS_FOR", "LOCATED_IN")
	SourceID   string                 `json:"source_id"`  // Source node ID
	TargetID   string                 `json:"target_id"`  // Target node ID
	Properties map[string]interface{} `json:"properties"` // Edge properties

	// UDML provenance
	SourceElementID string  `json:"source_element_id,omitempty"` // UDML element ID
	Domain          string  `json:"domain,omitempty"`            // Data mesh domain
	Confidence      float64 `json:"confidence"`                  // Extraction confidence (0.0-1.0)
	Evidence        string  `json:"evidence,omitempty"`          // Supporting evidence text

	// Graph metadata
	CreatedAt time.Time `json:"created_at"` // Creation timestamp
}

// GraphStats contains statistics about a graph
type GraphStats struct {
	NodeCount       int                 `json:"node_count"`
	EdgeCount       int                 `json:"edge_count"`
	LabelCounts     map[string]int      `json:"label_counts"`     // Count by label
	EdgeTypeCounts  map[string]int      `json:"edge_type_counts"` // Count by edge type
	DomainCounts    map[string]int      `json:"domain_counts"`    // Count by domain
	AvgConfidence   float64             `json:"avg_confidence"`
	ConnectedComponents int             `json:"connected_components"`
	IsolatedNodes   int                 `json:"isolated_nodes"` // Nodes with no edges
}

// ToJSON serializes the graph to pretty-printed JSON
func (g *Graph) ToJSON() ([]byte, error) {
	return json.MarshalIndent(g, "", "  ")
}

// ToJSONCompact serializes the graph to compact JSON
func (g *Graph) ToJSONCompact() ([]byte, error) {
	return json.Marshal(g)
}

// GetNodeByID retrieves a node by its ID
func (g *Graph) GetNodeByID(id string) *Node {
	for i := range g.Nodes {
		if g.Nodes[i].ID == id {
			return &g.Nodes[i]
		}
	}
	return nil
}

// GetEdgesBySource retrieves all edges from a source node
func (g *Graph) GetEdgesBySource(sourceID string) []Edge {
	var result []Edge
	for _, edge := range g.Edges {
		if edge.SourceID == sourceID {
			result = append(result, edge)
		}
	}
	return result
}

// GetEdgesByTarget retrieves all edges to a target node
func (g *Graph) GetEdgesByTarget(targetID string) []Edge {
	var result []Edge
	for _, edge := range g.Edges {
		if edge.TargetID == targetID {
			result = append(result, edge)
		}
	}
	return result
}

// GetEdgesByType retrieves all edges of a specific type
func (g *Graph) GetEdgesByType(edgeType string) []Edge {
	var result []Edge
	for _, edge := range g.Edges {
		if edge.Type == edgeType {
			result = append(result, edge)
		}
	}
	return result
}

// GetNodesByLabel retrieves all nodes with a specific label
func (g *Graph) GetNodesByLabel(label string) []Node {
	var result []Node
	for _, node := range g.Nodes {
		for _, l := range node.Labels {
			if l == label {
				result = append(result, node)
				break
			}
		}
	}
	return result
}

// GetNodesByDomain retrieves all nodes in a specific domain
func (g *Graph) GetNodesByDomain(domain string) []Node {
	var result []Node
	for _, node := range g.Nodes {
		if node.Domain == domain {
			result = append(result, node)
		}
	}
	return result
}

// GetStats calculates and returns graph statistics
func (g *Graph) GetStats() GraphStats {
	stats := GraphStats{
		NodeCount:      len(g.Nodes),
		EdgeCount:      len(g.Edges),
		LabelCounts:    make(map[string]int),
		EdgeTypeCounts: make(map[string]int),
		DomainCounts:   make(map[string]int),
	}

	// Count labels
	for _, node := range g.Nodes {
		for _, label := range node.Labels {
			stats.LabelCounts[label]++
		}
		if node.Domain != "" {
			stats.DomainCounts[node.Domain]++
		}
	}

	// Count edge types
	for _, edge := range g.Edges {
		stats.EdgeTypeCounts[edge.Type]++
	}

	// Calculate average confidence
	totalConfidence := 0.0
	confidenceCount := 0
	for _, node := range g.Nodes {
		totalConfidence += node.Confidence
		confidenceCount++
	}
	for _, edge := range g.Edges {
		totalConfidence += edge.Confidence
		confidenceCount++
	}
	if confidenceCount > 0 {
		stats.AvgConfidence = totalConfidence / float64(confidenceCount)
	}

	// Count isolated nodes (nodes with no edges)
	nodeHasEdge := make(map[string]bool)
	for _, edge := range g.Edges {
		nodeHasEdge[edge.SourceID] = true
		nodeHasEdge[edge.TargetID] = true
	}
	for _, node := range g.Nodes {
		if !nodeHasEdge[node.ID] {
			stats.IsolatedNodes++
		}
	}

	// Calculate connected components (simplified)
	stats.ConnectedComponents = g.countConnectedComponents()

	return stats
}

// countConnectedComponents counts connected components using DFS
func (g *Graph) countConnectedComponents() int {
	if len(g.Nodes) == 0 {
		return 0
	}

	// Build adjacency list
	adjacency := make(map[string][]string)
	for _, node := range g.Nodes {
		adjacency[node.ID] = []string{}
	}
	for _, edge := range g.Edges {
		adjacency[edge.SourceID] = append(adjacency[edge.SourceID], edge.TargetID)
		adjacency[edge.TargetID] = append(adjacency[edge.TargetID], edge.SourceID)
	}

	// DFS to find components
	visited := make(map[string]bool)
	components := 0

	var dfs func(string)
	dfs = func(nodeID string) {
		visited[nodeID] = true
		for _, neighbor := range adjacency[nodeID] {
			if !visited[neighbor] {
				dfs(neighbor)
			}
		}
	}

	for _, node := range g.Nodes {
		if !visited[node.ID] {
			components++
			dfs(node.ID)
		}
	}

	return components
}

// Validate checks if the graph is valid
func (g *Graph) Validate() error {
	// Check node IDs are unique
	nodeIDs := make(map[string]bool)
	for _, node := range g.Nodes {
		if node.ID == "" {
			return NewValidationError("node has empty ID")
		}
		if nodeIDs[node.ID] {
			return NewValidationError("duplicate node ID: " + node.ID)
		}
		nodeIDs[node.ID] = true

		// Validate confidence range
		if node.Confidence < 0.0 || node.Confidence > 1.0 {
			return NewValidationError(fmt.Sprintf("node %s has invalid confidence: %.2f", node.ID, node.Confidence))
		}
	}

	// Check edge references valid nodes
	for _, edge := range g.Edges {
		if edge.ID == "" {
			return NewValidationError("edge has empty ID")
		}
		if edge.SourceID == "" || edge.TargetID == "" {
			return NewValidationError("edge has empty source or target ID")
		}
		if !nodeIDs[edge.SourceID] {
			return NewValidationError("edge references unknown source node: " + edge.SourceID)
		}
		if !nodeIDs[edge.TargetID] {
			return NewValidationError("edge references unknown target node: " + edge.TargetID)
		}

		// Validate confidence range
		if edge.Confidence < 0.0 || edge.Confidence > 1.0 {
			return NewValidationError(fmt.Sprintf("edge %s has invalid confidence: %.2f", edge.ID, edge.Confidence))
		}
	}

	return nil
}

// FilterByConfidence returns a new graph with only nodes and edges above a confidence threshold
func (g *Graph) FilterByConfidence(minConfidence float64) *Graph {
	// Filter nodes
	var nodes []Node
	keepNodeIDs := make(map[string]bool)
	for _, node := range g.Nodes {
		if node.Confidence >= minConfidence {
			nodes = append(nodes, node)
			keepNodeIDs[node.ID] = true
		}
	}

	// Filter edges (must have high confidence AND reference kept nodes)
	var edges []Edge
	for _, edge := range g.Edges {
		if edge.Confidence >= minConfidence &&
			keepNodeIDs[edge.SourceID] &&
			keepNodeIDs[edge.TargetID] {
			edges = append(edges, edge)
		}
	}

	return &Graph{
		ID:        g.ID + "_filtered",
		Name:      g.Name + " (filtered)",
		Version:   g.Version,
		Nodes:     nodes,
		Edges:     edges,
		Metadata:  g.Metadata,
		CreatedAt: g.CreatedAt,
		UpdatedAt: time.Now(),
	}
}

// Merge combines two graphs into a single graph
func (g *Graph) Merge(other *Graph) *Graph {
	merged := &Graph{
		ID:        g.ID + "_merged",
		Name:      g.Name + " + " + other.Name,
		Version:   g.Version,
		Nodes:     make([]Node, 0, len(g.Nodes)+len(other.Nodes)),
		Edges:     make([]Edge, 0, len(g.Edges)+len(other.Edges)),
		Metadata:  make(map[string]interface{}),
		CreatedAt: g.CreatedAt,
		UpdatedAt: time.Now(),
	}

	// Merge nodes (deduplicate by ID)
	nodeMap := make(map[string]Node)
	for _, node := range g.Nodes {
		nodeMap[node.ID] = node
	}
	for _, node := range other.Nodes {
		if existing, ok := nodeMap[node.ID]; ok {
			// Keep node with higher confidence
			if node.Confidence > existing.Confidence {
				nodeMap[node.ID] = node
			}
		} else {
			nodeMap[node.ID] = node
		}
	}
	for _, node := range nodeMap {
		merged.Nodes = append(merged.Nodes, node)
	}

	// Merge edges (deduplicate by source+target+type)
	edgeKey := func(e Edge) string {
		return fmt.Sprintf("%s->%s:%s", e.SourceID, e.TargetID, e.Type)
	}
	edgeMap := make(map[string]Edge)
	for _, edge := range g.Edges {
		edgeMap[edgeKey(edge)] = edge
	}
	for _, edge := range other.Edges {
		key := edgeKey(edge)
		if existing, ok := edgeMap[key]; ok {
			// Keep edge with higher confidence
			if edge.Confidence > existing.Confidence {
				edgeMap[key] = edge
			}
		} else {
			edgeMap[key] = edge
		}
	}
	for _, edge := range edgeMap {
		merged.Edges = append(merged.Edges, edge)
	}

	return merged
}

// ValidationError represents a graph validation error
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return "graph validation error: " + e.Message
}

// NewValidationError creates a new validation error
func NewValidationError(message string) error {
	return &ValidationError{Message: message}
}