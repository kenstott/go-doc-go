package graph

import (
	"fmt"
	"time"
)

// Builder helps construct graphs with automatic deduplication
type Builder struct {
	id        string
	name      string
	version   string
	nodes     map[string]*Node // Deduplicate by ID
	edges     map[string]*Edge // Deduplicate by source+target+type
	metadata  map[string]interface{}
	createdAt time.Time
}

// NewBuilder creates a new graph builder
func NewBuilder(id, name, version string) *Builder {
	return &Builder{
		id:        id,
		name:      name,
		version:   version,
		nodes:     make(map[string]*Node),
		edges:     make(map[string]*Edge),
		metadata:  make(map[string]interface{}),
		createdAt: time.Now(),
	}
}

// AddNode adds a node to the graph (deduplicates by ID)
// If a node with the same ID exists, keeps the one with higher confidence
func (b *Builder) AddNode(node Node) {
	if existing, ok := b.nodes[node.ID]; ok {
		// Node already exists - keep the one with higher confidence
		if node.Confidence > existing.Confidence {
			b.nodes[node.ID] = &node
		}
	} else {
		b.nodes[node.ID] = &node
	}
}

// AddNodes adds multiple nodes to the graph
func (b *Builder) AddNodes(nodes []Node) {
	for _, node := range nodes {
		b.AddNode(node)
	}
}

// AddEdge adds an edge to the graph (deduplicates by source+target+type)
// If an edge with the same key exists, keeps the one with higher confidence
func (b *Builder) AddEdge(edge Edge) {
	key := edgeKey(edge)
	if existing, ok := b.edges[key]; ok {
		// Edge already exists - keep the one with higher confidence
		if edge.Confidence > existing.Confidence {
			b.edges[key] = &edge
		}
	} else {
		b.edges[key] = &edge
	}
}

// AddEdges adds multiple edges to the graph
func (b *Builder) AddEdges(edges []Edge) {
	for _, edge := range edges {
		b.AddEdge(edge)
	}
}

// GetNode retrieves a node by ID
func (b *Builder) GetNode(id string) *Node {
	return b.nodes[id]
}

// HasNode checks if a node exists
func (b *Builder) HasNode(id string) bool {
	_, exists := b.nodes[id]
	return exists
}

// RemoveNode removes a node and all connected edges
func (b *Builder) RemoveNode(id string) {
	delete(b.nodes, id)

	// Remove all edges connected to this node
	toRemove := []string{}
	for key, edge := range b.edges {
		if edge.SourceID == id || edge.TargetID == id {
			toRemove = append(toRemove, key)
		}
	}
	for _, key := range toRemove {
		delete(b.edges, key)
	}
}

// RemoveEdge removes an edge
func (b *Builder) RemoveEdge(sourceID, targetID, edgeType string) {
	key := fmt.Sprintf("%s->%s:%s", sourceID, targetID, edgeType)
	delete(b.edges, key)
}

// SetMetadata sets a metadata field
func (b *Builder) SetMetadata(key string, value interface{}) {
	b.metadata[key] = value
}

// GetMetadata retrieves a metadata field
func (b *Builder) GetMetadata(key string) interface{} {
	return b.metadata[key]
}

// NodeCount returns the number of nodes
func (b *Builder) NodeCount() int {
	return len(b.nodes)
}

// EdgeCount returns the number of edges
func (b *Builder) EdgeCount() int {
	return len(b.edges)
}

// Clear removes all nodes and edges
func (b *Builder) Clear() {
	b.nodes = make(map[string]*Node)
	b.edges = make(map[string]*Edge)
}

// Build constructs the final Graph
func (b *Builder) Build() *Graph {
	// Convert maps to slices
	nodes := make([]Node, 0, len(b.nodes))
	for _, node := range b.nodes {
		nodes = append(nodes, *node)
	}

	edges := make([]Edge, 0, len(b.edges))
	for _, edge := range b.edges {
		edges = append(edges, *edge)
	}

	return &Graph{
		ID:        b.id,
		Name:      b.name,
		Version:   b.version,
		Nodes:     nodes,
		Edges:     edges,
		Metadata:  b.metadata,
		CreatedAt: b.createdAt,
		UpdatedAt: time.Now(),
	}
}

// BuildAndValidate constructs the graph and validates it
func (b *Builder) BuildAndValidate() (*Graph, error) {
	graph := b.Build()
	if err := graph.Validate(); err != nil {
		return nil, err
	}
	return graph, nil
}

// edgeKey generates a unique key for an edge
func edgeKey(edge Edge) string {
	return fmt.Sprintf("%s->%s:%s", edge.SourceID, edge.TargetID, edge.Type)
}

// MergeNode merges properties from a new node into an existing node
// Updates confidence to the maximum of the two
// Merges properties (new values override existing)
func (b *Builder) MergeNode(id string, updates Node) {
	existing, ok := b.nodes[id]
	if !ok {
		// Node doesn't exist, just add it
		b.AddNode(updates)
		return
	}

	// Merge properties
	if existing.Properties == nil {
		existing.Properties = make(map[string]interface{})
	}
	for key, value := range updates.Properties {
		existing.Properties[key] = value
	}

	// Merge labels (deduplicate)
	labelSet := make(map[string]bool)
	for _, label := range existing.Labels {
		labelSet[label] = true
	}
	for _, label := range updates.Labels {
		labelSet[label] = true
	}
	labels := make([]string, 0, len(labelSet))
	for label := range labelSet {
		labels = append(labels, label)
	}
	existing.Labels = labels

	// Update confidence to max
	if updates.Confidence > existing.Confidence {
		existing.Confidence = updates.Confidence
	}

	// Update provenance if provided
	if updates.SourceElementID != "" {
		existing.SourceElementID = updates.SourceElementID
	}
	if updates.SourceDocID != "" {
		existing.SourceDocID = updates.SourceDocID
	}
	if updates.Domain != "" {
		existing.Domain = updates.Domain
	}

	existing.UpdatedAt = time.Now()
}

// FilterNodesByLabel returns all nodes with a specific label
func (b *Builder) FilterNodesByLabel(label string) []*Node {
	var result []*Node
	for _, node := range b.nodes {
		for _, l := range node.Labels {
			if l == label {
				result = append(result, node)
				break
			}
		}
	}
	return result
}

// FilterNodesByDomain returns all nodes in a specific domain
func (b *Builder) FilterNodesByDomain(domain string) []*Node {
	var result []*Node
	for _, node := range b.nodes {
		if node.Domain == domain {
			result = append(result, node)
		}
	}
	return result
}

// FilterEdgesByType returns all edges of a specific type
func (b *Builder) FilterEdgesByType(edgeType string) []*Edge {
	var result []*Edge
	for _, edge := range b.edges {
		if edge.Type == edgeType {
			result = append(result, edge)
		}
	}
	return result
}

// Clone creates a copy of the builder
func (b *Builder) Clone() *Builder {
	clone := NewBuilder(b.id, b.name, b.version)
	clone.createdAt = b.createdAt

	// Copy nodes
	for id, node := range b.nodes {
		nodeCopy := *node
		clone.nodes[id] = &nodeCopy
	}

	// Copy edges
	for key, edge := range b.edges {
		edgeCopy := *edge
		clone.edges[key] = &edgeCopy
	}

	// Copy metadata
	for key, value := range b.metadata {
		clone.metadata[key] = value
	}

	return clone
}