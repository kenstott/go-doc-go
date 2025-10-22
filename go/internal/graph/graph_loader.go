package graph

import (
	"fmt"

	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/simple"
)

// GonumGraph wraps a gonum directed graph with metadata
type GonumGraph struct {
	*simple.DirectedGraph
	NodeMap  map[int64]*GraphNode  // Maps gonum ID to our Node
	EdgeMap  map[int64]*GraphEdge  // Maps gonum ID to our Edge
	IDToNode map[string]int64      // Maps our Node ID to gonum ID
}

// GraphNode wraps our Node with gonum node interface
type GraphNode struct {
	Node   Node
	id     int64
}

// ID implements gonum graph.Node interface
func (n *GraphNode) ID() int64 {
	return n.id
}

// GraphEdge wraps our Edge with gonum edge interface
type GraphEdge struct {
	Edge   Edge
	id     int64
	from   int64
	to     int64
}

// From implements gonum graph.Edge interface
func (e *GraphEdge) From() graph.Node {
	return &GraphNode{id: e.from}
}

// To implements gonum graph.Edge interface
func (e *GraphEdge) To() graph.Node {
	return &GraphNode{id: e.to}
}

// ID implements gonum graph.Edge interface (for weighted/labeled edges)
func (e *GraphEdge) ID() int64 {
	return e.id
}

// LoadFromGraph converts our Graph to a gonum DirectedGraph
// This enables use of gonum's graph algorithms (PageRank, community detection, etc.)
func LoadFromGraph(g *Graph) (*GonumGraph, error) {
	if g == nil {
		return nil, fmt.Errorf("graph cannot be nil")
	}

	// Create gonum directed graph
	dg := simple.NewDirectedGraph()

	// Initialize wrapper
	gg := &GonumGraph{
		DirectedGraph: dg,
		NodeMap:       make(map[int64]*GraphNode),
		EdgeMap:       make(map[int64]*GraphEdge),
		IDToNode:      make(map[string]int64),
	}

	// Add nodes with sequential gonum IDs
	nextID := int64(0)
	for _, node := range g.Nodes {
		gn := &GraphNode{
			Node: node,
			id:   nextID,
		}
		dg.AddNode(gn)
		gg.NodeMap[nextID] = gn
		gg.IDToNode[node.ID] = nextID
		nextID++
	}

	// Add edges
	edgeID := int64(0)
	for _, edge := range g.Edges {
		fromID, ok := gg.IDToNode[edge.SourceID]
		if !ok {
			return nil, fmt.Errorf("edge source node not found: %s", edge.SourceID)
		}
		toID, ok := gg.IDToNode[edge.TargetID]
		if !ok {
			return nil, fmt.Errorf("edge target node not found: %s", edge.TargetID)
		}

		ge := &GraphEdge{
			Edge: edge,
			id:   edgeID,
			from: fromID,
			to:   toID,
		}

		// Add edge to gonum graph
		dg.SetEdge(simple.Edge{
			F: gg.NodeMap[fromID],
			T: gg.NodeMap[toID],
		})

		gg.EdgeMap[edgeID] = ge
		edgeID++
	}

	return gg, nil
}

// ToGraph converts a GonumGraph back to our Graph structure
func (gg *GonumGraph) ToGraph() *Graph {
	// Extract nodes
	nodes := make([]Node, 0, len(gg.NodeMap))
	for _, gn := range gg.NodeMap {
		nodes = append(nodes, gn.Node)
	}

	// Extract edges
	edges := make([]Edge, 0, len(gg.EdgeMap))
	for _, ge := range gg.EdgeMap {
		edges = append(edges, ge.Edge)
	}

	return &Graph{
		ID:       "",
		Name:     "",
		Version:  "",
		Nodes:    nodes,
		Edges:    edges,
		Metadata: make(map[string]interface{}),
	}
}

// GetNodeByID retrieves our Node by our original ID
func (gg *GonumGraph) GetNodeByID(id string) (*Node, bool) {
	gonumID, ok := gg.IDToNode[id]
	if !ok {
		return nil, false
	}
	gn, ok := gg.NodeMap[gonumID]
	if !ok {
		return nil, false
	}
	return &gn.Node, true
}

// GetEdgesBetween retrieves all edges between two nodes (by our IDs)
func (gg *GonumGraph) GetEdgesBetween(sourceID, targetID string) []Edge {
	fromGonumID, ok := gg.IDToNode[sourceID]
	if !ok {
		return nil
	}
	toGonumID, ok := gg.IDToNode[targetID]
	if !ok {
		return nil
	}

	// Check if edge exists in gonum graph
	if !gg.HasEdgeFromTo(fromGonumID, toGonumID) {
		return nil
	}

	// Find all matching edges in our EdgeMap
	var result []Edge
	for _, ge := range gg.EdgeMap {
		if ge.from == fromGonumID && ge.to == toGonumID {
			result = append(result, ge.Edge)
		}
	}
	return result
}

// NodeCount returns the number of nodes
func (gg *GonumGraph) NodeCount() int {
	return gg.Nodes().Len()
}

// EdgeCount returns the number of edges
func (gg *GonumGraph) EdgeCount() int {
	return len(gg.EdgeMap)
}
