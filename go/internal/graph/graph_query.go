package graph

import (
	"fmt"
	"strings"
)

// GraphQueryEngine provides a native Go query API for graphs
// Avoids need for external query languages (Cypher, SPARQL, etc.)
type GraphQueryEngine struct {
	graph *Graph
}

// NewGraphQueryEngine creates a new query engine for a graph
func NewGraphQueryEngine(g *Graph) *GraphQueryEngine {
	return &GraphQueryEngine{graph: g}
}

// NodeFilter defines a filter function for nodes
type NodeFilter func(Node) bool

// EdgeFilter defines a filter function for edges
type EdgeFilter func(Edge) bool

// QueryOptions configures query behavior
type QueryOptions struct {
	Limit  int  // Maximum results to return (0 = no limit)
	Offset int  // Number of results to skip
	Sort   bool // Whether to sort results
}

// DefaultQueryOptions returns sensible defaults
func DefaultQueryOptions() QueryOptions {
	return QueryOptions{
		Limit:  0,
		Offset: 0,
		Sort:   false,
	}
}

// FindNodes returns all nodes matching the filter
func (qe *GraphQueryEngine) FindNodes(filter NodeFilter, opts QueryOptions) []Node {
	var results []Node

	for _, node := range qe.graph.Nodes {
		if filter(node) {
			results = append(results, node)
		}
	}

	// Apply offset and limit
	return applyPagination(results, opts.Offset, opts.Limit)
}

// FindNodeByID finds a node by its ID
func (qe *GraphQueryEngine) FindNodeByID(id string) (*Node, bool) {
	for _, node := range qe.graph.Nodes {
		if node.ID == id {
			return &node, true
		}
	}
	return nil, false
}

// FindNodesByLabel finds all nodes with a specific label
func (qe *GraphQueryEngine) FindNodesByLabel(label string, opts QueryOptions) []Node {
	return qe.FindNodes(func(n Node) bool {
		return hasLabel(n, label)
	}, opts)
}

// FindNodesByProperty finds nodes with a specific property value
func (qe *GraphQueryEngine) FindNodesByProperty(key string, value interface{}, opts QueryOptions) []Node {
	return qe.FindNodes(func(n Node) bool {
		if propValue, exists := n.Properties[key]; exists {
			return propValue == value
		}
		return false
	}, opts)
}

// FindNodesByDomain finds nodes in a specific domain
func (qe *GraphQueryEngine) FindNodesByDomain(domain string, opts QueryOptions) []Node {
	return qe.FindNodes(func(n Node) bool {
		return n.Domain == domain
	}, opts)
}

// FindNodesByConfidence finds nodes with confidence >= threshold
func (qe *GraphQueryEngine) FindNodesByConfidence(minConfidence float64, opts QueryOptions) []Node {
	return qe.FindNodes(func(n Node) bool {
		return n.Confidence >= minConfidence
	}, opts)
}

// FindEdges returns all edges matching the filter
func (qe *GraphQueryEngine) FindEdges(filter EdgeFilter, opts QueryOptions) []Edge {
	var results []Edge

	for _, edge := range qe.graph.Edges {
		if filter(edge) {
			results = append(results, edge)
		}
	}

	return applyPaginationEdges(results, opts.Offset, opts.Limit)
}

// FindEdgeByID finds an edge by its ID
func (qe *GraphQueryEngine) FindEdgeByID(id string) (*Edge, bool) {
	for _, edge := range qe.graph.Edges {
		if edge.ID == id {
			return &edge, true
		}
	}
	return nil, false
}

// FindEdgesByType finds all edges of a specific type
func (qe *GraphQueryEngine) FindEdgesByType(edgeType string, opts QueryOptions) []Edge {
	return qe.FindEdges(func(e Edge) bool {
		return e.Type == edgeType
	}, opts)
}

// FindEdgesBetween finds all edges from source to target
func (qe *GraphQueryEngine) FindEdgesBetween(sourceID, targetID string, opts QueryOptions) []Edge {
	return qe.FindEdges(func(e Edge) bool {
		return e.SourceID == sourceID && e.TargetID == targetID
	}, opts)
}

// FindEdgesFrom finds all edges originating from a node
func (qe *GraphQueryEngine) FindEdgesFrom(sourceID string, opts QueryOptions) []Edge {
	return qe.FindEdges(func(e Edge) bool {
		return e.SourceID == sourceID
	}, opts)
}

// FindEdgesTo finds all edges pointing to a node
func (qe *GraphQueryEngine) FindEdgesTo(targetID string, opts QueryOptions) []Edge {
	return qe.FindEdges(func(e Edge) bool {
		return e.TargetID == targetID
	}, opts)
}

// GetNeighbors returns all nodes connected to the given node
func (qe *GraphQueryEngine) GetNeighbors(nodeID string, direction NeighborDirection, opts QueryOptions) []Node {
	var neighborIDs []string

	switch direction {
	case OutgoingNeighbors:
		// Nodes this node points to
		for _, edge := range qe.graph.Edges {
			if edge.SourceID == nodeID {
				neighborIDs = append(neighborIDs, edge.TargetID)
			}
		}
	case IncomingNeighbors:
		// Nodes that point to this node
		for _, edge := range qe.graph.Edges {
			if edge.TargetID == nodeID {
				neighborIDs = append(neighborIDs, edge.SourceID)
			}
		}
	case BothNeighbors:
		// All connected nodes
		for _, edge := range qe.graph.Edges {
			if edge.SourceID == nodeID {
				neighborIDs = append(neighborIDs, edge.TargetID)
			} else if edge.TargetID == nodeID {
				neighborIDs = append(neighborIDs, edge.SourceID)
			}
		}
	}

	// Deduplicate neighbor IDs
	uniqueIDs := make(map[string]bool)
	for _, id := range neighborIDs {
		uniqueIDs[id] = true
	}

	// Find neighbor nodes
	var neighbors []Node
	for id := range uniqueIDs {
		if node, found := qe.FindNodeByID(id); found {
			neighbors = append(neighbors, *node)
		}
	}

	return applyPagination(neighbors, opts.Offset, opts.Limit)
}

// NeighborDirection specifies which neighbors to retrieve
type NeighborDirection int

const (
	OutgoingNeighbors NeighborDirection = iota // Nodes this node points to
	IncomingNeighbors                          // Nodes that point to this node
	BothNeighbors                              // All connected nodes
)

// Path represents a path through the graph
type Path struct {
	Nodes []Node // Ordered list of nodes in the path
	Edges []Edge // Ordered list of edges connecting the nodes
	Cost  float64 // Total cost/weight of the path
}

// FindShortestPath finds the shortest path between two nodes using BFS
// Returns nil if no path exists
func (qe *GraphQueryEngine) FindShortestPath(startID, endID string, maxDepth int) *Path {
	if maxDepth == 0 {
		maxDepth = 100 // Default max depth
	}

	// BFS to find shortest path
	type queueItem struct {
		nodeID string
		path   []string
		edges  []string
		depth  int
	}

	visited := make(map[string]bool)
	queue := []queueItem{{nodeID: startID, path: []string{startID}, edges: []string{}, depth: 0}}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if current.nodeID == endID {
			// Found the destination - construct path
			return qe.constructPath(current.path, current.edges)
		}

		if current.depth >= maxDepth {
			continue
		}

		if visited[current.nodeID] {
			continue
		}
		visited[current.nodeID] = true

		// Explore outgoing edges
		for _, edge := range qe.graph.Edges {
			if edge.SourceID == current.nodeID {
				targetID := edge.TargetID
				if !visited[targetID] {
					newPath := append([]string{}, current.path...)
					newPath = append(newPath, targetID)
					newEdges := append([]string{}, current.edges...)
					newEdges = append(newEdges, edge.ID)

					queue = append(queue, queueItem{
						nodeID: targetID,
						path:   newPath,
						edges:  newEdges,
						depth:  current.depth + 1,
					})
				}
			}
		}
	}

	return nil // No path found
}

// constructPath builds a Path from node IDs and edge IDs
func (qe *GraphQueryEngine) constructPath(nodeIDs []string, edgeIDs []string) *Path {
	path := &Path{
		Nodes: make([]Node, 0, len(nodeIDs)),
		Edges: make([]Edge, 0, len(edgeIDs)),
		Cost:  0,
	}

	// Collect nodes
	for _, nodeID := range nodeIDs {
		if node, found := qe.FindNodeByID(nodeID); found {
			path.Nodes = append(path.Nodes, *node)
		}
	}

	// Collect edges
	for _, edgeID := range edgeIDs {
		if edge, found := qe.FindEdgeByID(edgeID); found {
			path.Edges = append(path.Edges, *edge)
			// Use confidence as cost (higher confidence = lower cost)
			path.Cost += (1.0 - edge.Confidence)
		}
	}

	return path
}

// GetSubgraph returns a subgraph containing only nodes matching the filter
// and edges between those nodes
func (qe *GraphQueryEngine) GetSubgraph(nodeFilter NodeFilter) *Graph {
	// Find matching nodes
	matchingNodes := qe.FindNodes(nodeFilter, DefaultQueryOptions())

	// Build node ID set for quick lookup
	nodeIDSet := make(map[string]bool)
	for _, node := range matchingNodes {
		nodeIDSet[node.ID] = true
	}

	// Find edges between matching nodes
	var matchingEdges []Edge
	for _, edge := range qe.graph.Edges {
		if nodeIDSet[edge.SourceID] && nodeIDSet[edge.TargetID] {
			matchingEdges = append(matchingEdges, edge)
		}
	}

	// Create subgraph
	subgraph := &Graph{
		ID:       fmt.Sprintf("%s_subgraph", qe.graph.ID),
		Name:     fmt.Sprintf("%s Subgraph", qe.graph.Name),
		Version:  qe.graph.Version,
		Nodes:    matchingNodes,
		Edges:    matchingEdges,
		Metadata: make(map[string]interface{}),
	}

	// Copy relevant metadata
	for k, v := range qe.graph.Metadata {
		subgraph.Metadata[k] = v
	}
	subgraph.Metadata["is_subgraph"] = true
	subgraph.Metadata["parent_graph_id"] = qe.graph.ID

	return subgraph
}

// CountNodes counts nodes matching the filter
func (qe *GraphQueryEngine) CountNodes(filter NodeFilter) int {
	count := 0
	for _, node := range qe.graph.Nodes {
		if filter(node) {
			count++
		}
	}
	return count
}

// CountEdges counts edges matching the filter
func (qe *GraphQueryEngine) CountEdges(filter EdgeFilter) int {
	count := 0
	for _, edge := range qe.graph.Edges {
		if filter(edge) {
			count++
		}
	}
	return count
}

// GetGraph returns the underlying graph
func (qe *GraphQueryEngine) GetGraph() *Graph {
	return qe.graph
}

// Helper functions

// hasLabel checks if a node has a specific label
func hasLabel(node Node, label string) bool {
	for _, l := range node.Labels {
		if strings.EqualFold(l, label) {
			return true
		}
	}
	return false
}

// applyPagination applies offset and limit to node results
func applyPagination(nodes []Node, offset, limit int) []Node {
	// Apply offset
	if offset >= len(nodes) {
		return []Node{}
	}
	nodes = nodes[offset:]

	// Apply limit
	if limit > 0 && limit < len(nodes) {
		nodes = nodes[:limit]
	}

	return nodes
}

// applyPaginationEdges applies offset and limit to edge results
func applyPaginationEdges(edges []Edge, offset, limit int) []Edge {
	// Apply offset
	if offset >= len(edges) {
		return []Edge{}
	}
	edges = edges[offset:]

	// Apply limit
	if limit > 0 && limit < len(edges) {
		edges = edges[:limit]
	}

	return edges
}

// Common filter constructors for convenience

// AndNodeFilters combines multiple node filters with AND logic
func AndNodeFilters(filters ...NodeFilter) NodeFilter {
	return func(n Node) bool {
		for _, f := range filters {
			if !f(n) {
				return false
			}
		}
		return true
	}
}

// OrNodeFilters combines multiple node filters with OR logic
func OrNodeFilters(filters ...NodeFilter) NodeFilter {
	return func(n Node) bool {
		for _, f := range filters {
			if f(n) {
				return true
			}
		}
		return false
	}
}

// NotNodeFilter negates a node filter
func NotNodeFilter(filter NodeFilter) NodeFilter {
	return func(n Node) bool {
		return !filter(n)
	}
}

// HasLabelFilter creates a filter that checks for a specific label
func HasLabelFilter(label string) NodeFilter {
	return func(n Node) bool {
		return hasLabel(n, label)
	}
}

// HasPropertyFilter creates a filter that checks if a property exists
func HasPropertyFilter(key string) NodeFilter {
	return func(n Node) bool {
		_, exists := n.Properties[key]
		return exists
	}
}

// PropertyEqualsFilter creates a filter for exact property match
func PropertyEqualsFilter(key string, value interface{}) NodeFilter {
	return func(n Node) bool {
		if propValue, exists := n.Properties[key]; exists {
			return propValue == value
		}
		return false
	}
}

// ConfidenceAboveFilter creates a filter for minimum confidence
func ConfidenceAboveFilter(minConfidence float64) NodeFilter {
	return func(n Node) bool {
		return n.Confidence >= minConfidence
	}
}

// DomainFilter creates a filter for a specific domain
func DomainFilter(domain string) NodeFilter {
	return func(n Node) bool {
		return n.Domain == domain
	}
}

// AndEdgeFilters combines multiple edge filters with AND logic
func AndEdgeFilters(filters ...EdgeFilter) EdgeFilter {
	return func(e Edge) bool {
		for _, f := range filters {
			if !f(e) {
				return false
			}
		}
		return true
	}
}

// EdgeTypeFilter creates a filter for a specific edge type
func EdgeTypeFilter(edgeType string) EdgeFilter {
	return func(e Edge) bool {
		return e.Type == edgeType
	}
}

// EdgeConfidenceAboveFilter creates a filter for minimum edge confidence
func EdgeConfidenceAboveFilter(minConfidence float64) EdgeFilter {
	return func(e Edge) bool {
		return e.Confidence >= minConfidence
	}
}
