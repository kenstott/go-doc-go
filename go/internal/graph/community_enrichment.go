package graph

import (
	"fmt"
	"time"
)

// AddCommunityNodes enriches a graph by adding explicit community nodes
// and BELONGS_TO edges from entity nodes to their communities.
// This implements "reification" - making implicit community structure explicit in the graph.
func AddCommunityNodes(g *Graph, communityResult *CommunityResult) *Graph {
	if g == nil || communityResult == nil {
		return g
	}

	// Create enriched graph by copying original nodes and edges
	enriched := &Graph{
		ID:       g.ID,
		Name:     g.Name,
		Version:  g.Version,
		Nodes:    make([]Node, len(g.Nodes)),
		Edges:    make([]Edge, len(g.Edges)),
		Metadata: make(map[string]interface{}),
	}

	// Copy existing nodes and edges
	copy(enriched.Nodes, g.Nodes)
	copy(enriched.Edges, g.Edges)

	// Copy existing metadata
	for k, v := range g.Metadata {
		enriched.Metadata[k] = v
	}

	// Create a node for each community
	for _, comm := range communityResult.Communities {
		commNode := Node{
			ID:     fmt.Sprintf("community_%s_%d", communityResult.Algorithm, comm.ID),
			Labels: []string{"Community"},
			Properties: map[string]interface{}{
				"community_id":  comm.ID,
				"algorithm":     string(communityResult.Algorithm),
				"size":          comm.Size,
				"member_count":  len(comm.NodeIDs),
				"modularity":    communityResult.Modularity,
				"member_ids":    comm.NodeIDs,
			},
			Domain:     "graph_analysis",
			Confidence: 1.0, // Community detection is deterministic
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}

		// Add community metadata from CommunityResult
		for k, v := range communityResult.Metadata {
			commNode.Properties[k] = v
		}

		enriched.Nodes = append(enriched.Nodes, commNode)

		// Create BELONGS_TO edges from each member to the community
		for _, nodeID := range comm.NodeIDs {
			edge := Edge{
				ID:       fmt.Sprintf("belongs_to_%s_comm_%d", nodeID, comm.ID),
				Type:     "BELONGS_TO",
				SourceID: nodeID,
				TargetID: commNode.ID,
				Properties: map[string]interface{}{
					"algorithm":    string(communityResult.Algorithm),
					"community_id": comm.ID,
				},
				Domain:     "graph_analysis",
				Confidence: 1.0,
				CreatedAt:  time.Now(),
			}
			enriched.Edges = append(enriched.Edges, edge)
		}
	}

	// Update graph metadata with community detection info
	enriched.Metadata["community_detection_algorithm"] = string(communityResult.Algorithm)
	enriched.Metadata["community_count"] = len(communityResult.Communities)
	enriched.Metadata["modularity"] = communityResult.Modularity
	enriched.Metadata["community_enriched_at"] = time.Now().Format(time.RFC3339)

	return enriched
}

// AddMultipleCommunityResults adds community nodes from multiple detection runs
// (e.g., both Louvain and Label Propagation results)
// Each algorithm's communities are kept separate with distinct node IDs
func AddMultipleCommunityResults(g *Graph, results []*CommunityResult) *Graph {
	enriched := g
	for _, result := range results {
		enriched = AddCommunityNodes(enriched, result)
	}
	return enriched
}

// GetCommunityNodesFromGraph extracts all community nodes from an enriched graph
func GetCommunityNodesFromGraph(g *Graph) []Node {
	var communityNodes []Node
	for _, node := range g.Nodes {
		// Check if node has "Community" label
		for _, label := range node.Labels {
			if label == "Community" {
				communityNodes = append(communityNodes, node)
				break
			}
		}
	}
	return communityNodes
}

// GetCommunityMembersFromGraph retrieves all nodes that belong to a specific community
func GetCommunityMembersFromGraph(g *Graph, communityNodeID string) []Node {
	// Find all BELONGS_TO edges pointing to this community
	var memberIDs []string
	for _, edge := range g.Edges {
		if edge.Type == "BELONGS_TO" && edge.TargetID == communityNodeID {
			memberIDs = append(memberIDs, edge.SourceID)
		}
	}

	// Find nodes with these IDs
	var members []Node
	for _, node := range g.Nodes {
		for _, memberID := range memberIDs {
			if node.ID == memberID {
				members = append(members, node)
				break
			}
		}
	}

	return members
}

// RemoveCommunityNodes removes all community nodes and BELONGS_TO edges
// This can be used to "flatten" an enriched graph back to original form
func RemoveCommunityNodes(g *Graph) *Graph {
	if g == nil {
		return nil
	}

	// Create new graph without community nodes
	cleaned := &Graph{
		ID:       g.ID,
		Name:     g.Name,
		Version:  g.Version,
		Nodes:    make([]Node, 0, len(g.Nodes)),
		Edges:    make([]Edge, 0, len(g.Edges)),
		Metadata: make(map[string]interface{}),
	}

	// Copy metadata (excluding community detection metadata)
	for k, v := range g.Metadata {
		if k != "community_detection_algorithm" &&
			k != "community_count" &&
			k != "modularity" &&
			k != "community_enriched_at" {
			cleaned.Metadata[k] = v
		}
	}

	// Build set of community node IDs for quick lookup
	communityNodeIDs := make(map[string]bool)
	for _, node := range g.Nodes {
		for _, label := range node.Labels {
			if label == "Community" {
				communityNodeIDs[node.ID] = true
				break
			}
		}
	}

	// Copy only non-community nodes
	for _, node := range g.Nodes {
		if !communityNodeIDs[node.ID] {
			cleaned.Nodes = append(cleaned.Nodes, node)
		}
	}

	// Copy only non-BELONGS_TO edges
	for _, edge := range g.Edges {
		if edge.Type != "BELONGS_TO" {
			cleaned.Edges = append(cleaned.Edges, edge)
		}
	}

	return cleaned
}

// CommunityEnrichmentStats provides statistics about community enrichment
type CommunityEnrichmentStats struct {
	OriginalNodeCount   int     // Nodes before enrichment
	EnrichedNodeCount   int     // Nodes after enrichment
	CommunityNodeCount  int     // Number of community nodes added
	BelongsToEdgeCount  int     // Number of BELONGS_TO edges added
	CommunityCount      int     // Number of communities detected
	Modularity          float64 // Modularity score
	Algorithm           string  // Algorithm used
	AvgCommunitySize    float64 // Average community size
}

// GetCommunityEnrichmentStats analyzes an enriched graph and returns statistics
func GetCommunityEnrichmentStats(original *Graph, enriched *Graph) *CommunityEnrichmentStats {
	stats := &CommunityEnrichmentStats{
		OriginalNodeCount: len(original.Nodes),
		EnrichedNodeCount: len(enriched.Nodes),
	}

	// Count community nodes
	communityNodes := GetCommunityNodesFromGraph(enriched)
	stats.CommunityNodeCount = len(communityNodes)

	// Count BELONGS_TO edges
	for _, edge := range enriched.Edges {
		if edge.Type == "BELONGS_TO" {
			stats.BelongsToEdgeCount++
		}
	}

	// Extract metadata
	if algo, ok := enriched.Metadata["community_detection_algorithm"].(string); ok {
		stats.Algorithm = algo
	}
	if count, ok := enriched.Metadata["community_count"].(int); ok {
		stats.CommunityCount = count
	}
	if mod, ok := enriched.Metadata["modularity"].(float64); ok {
		stats.Modularity = mod
	}

	// Calculate average community size
	if stats.CommunityCount > 0 {
		stats.AvgCommunitySize = float64(stats.OriginalNodeCount) / float64(stats.CommunityCount)
	}

	return stats
}
