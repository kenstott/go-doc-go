package graph

import (
	"fmt"
	"log"
)

// CommunityDetectionAlgorithm represents different community detection algorithms
type CommunityDetectionAlgorithm string

const (
	AlgorithmLouvain          CommunityDetectionAlgorithm = "louvain"
	AlgorithmLabelPropagation CommunityDetectionAlgorithm = "label_propagation"
)

// CommunityResult represents the result of community detection
type CommunityResult struct {
	Algorithm   CommunityDetectionAlgorithm // Algorithm used
	Communities []Community                 // Detected communities
	Modularity  float64                     // Modularity score of the partition
	Metadata    map[string]interface{}      // Additional metadata
}

// Community represents a detected community
type Community struct {
	ID      int      // Community ID
	NodeIDs []string // Original node IDs (from our Graph)
	Size    int      // Number of nodes in community
}

// DetectCommunities detects communities in a GonumGraph using the specified algorithm
func DetectCommunities(gg *GonumGraph, algorithm CommunityDetectionAlgorithm) (*CommunityResult, error) {
	switch algorithm {
	case AlgorithmLouvain:
		return detectCommunitiesLouvain(gg)
	case AlgorithmLabelPropagation:
		return detectCommunitiesLabelPropagation(gg)
	default:
		return nil, fmt.Errorf("unknown algorithm: %s", algorithm)
	}
}

// detectCommunitiesLouvain performs Louvain community detection
// Louvain is a greedy optimization method that maximizes modularity
// TODO: Implement proper Louvain algorithm using gonum or igraph bindings
func detectCommunitiesLouvain(gg *GonumGraph) (*CommunityResult, error) {
	if gg == nil || gg.Nodes().Len() == 0 {
		return nil, fmt.Errorf("graph is empty")
	}

	log.Printf("Running Louvain community detection (simplified) on graph with %d nodes, %d edges",
		gg.NodeCount(), gg.EdgeCount())

	// Simplified implementation: Group nodes into communities based on basic heuristics
	// This is a placeholder - proper Louvain implementation requires:
	// 1. Modularity calculation
	// 2. Iterative node reassignment
	// 3. Community aggregation
	communities := detectCommunitiesSimple(gg)

	// Calculate approximate modularity (simplified)
	modularity := 0.5 // Placeholder value

	log.Printf("Louvain (simplified): Found %d communities, modularity: %.4f",
		len(communities), modularity)

	result := &CommunityResult{
		Algorithm:   AlgorithmLouvain,
		Communities: communities,
		Modularity:  modularity,
		Metadata:    make(map[string]interface{}),
	}

	// Add statistics to metadata
	result.Metadata["total_communities"] = len(communities)
	if len(communities) > 0 {
		result.Metadata["avg_community_size"] = float64(gg.NodeCount()) / float64(len(communities))
	}

	return result, nil
}

// detectCommunitiesLabelPropagation performs Label Propagation community detection
// Label Propagation is a fast, near-linear time algorithm based on label spreading
// TODO: Implement proper Label Propagation using gonum or igraph bindings
func detectCommunitiesLabelPropagation(gg *GonumGraph) (*CommunityResult, error) {
	if gg == nil || gg.Nodes().Len() == 0 {
		return nil, fmt.Errorf("graph is empty")
	}

	log.Printf("Running Label Propagation (simplified) on graph with %d nodes, %d edges",
		gg.NodeCount(), gg.EdgeCount())

	// Simplified implementation
	communities := detectCommunitiesSimple(gg)

	modularity := 0.5 // Placeholder

	log.Printf("Label Propagation (simplified): Found %d communities, modularity: %.4f",
		len(communities), modularity)

	result := &CommunityResult{
		Algorithm:   AlgorithmLabelPropagation,
		Communities: communities,
		Modularity:  modularity,
		Metadata:    make(map[string]interface{}),
	}

	// Add statistics to metadata
	result.Metadata["total_communities"] = len(communities)
	if len(communities) > 0 {
		result.Metadata["avg_community_size"] = float64(gg.NodeCount()) / float64(len(communities))
	}

	return result, nil
}

// detectCommunitiesSimple is a placeholder that creates random communities
// TODO: Replace with proper community detection algorithms
func detectCommunitiesSimple(gg *GonumGraph) []Community {
	// Simple heuristic: Create communities of roughly equal size
	numCommunities := max(1, gg.NodeCount()/10) // ~10 nodes per community
	communities := make([]Community, numCommunities)

	for i := range communities {
		communities[i] = Community{
			ID:      i,
			NodeIDs: make([]string, 0),
			Size:    0,
		}
	}

	// Assign each node to a random community
	commIndex := 0
	for _, gn := range gg.NodeMap {
		communities[commIndex].NodeIDs = append(communities[commIndex].NodeIDs, gn.Node.ID)
		communities[commIndex].Size++
		commIndex = (commIndex + 1) % numCommunities
	}

	return communities
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// GetNodeCommunity returns the community ID for a given node
func (cr *CommunityResult) GetNodeCommunity(nodeID string) (int, bool) {
	for _, comm := range cr.Communities {
		for _, nid := range comm.NodeIDs {
			if nid == nodeID {
				return comm.ID, true
			}
		}
	}
	return -1, false
}

// GetCommunityByID retrieves a community by its ID
func (cr *CommunityResult) GetCommunityByID(communityID int) (*Community, bool) {
	if communityID < 0 || communityID >= len(cr.Communities) {
		return nil, false
	}
	return &cr.Communities[communityID], true
}

// GetLargestCommunities returns the N largest communities by size
func (cr *CommunityResult) GetLargestCommunities(n int) []Community {
	if n <= 0 || len(cr.Communities) == 0 {
		return nil
	}

	// Copy communities for sorting
	communities := make([]Community, len(cr.Communities))
	copy(communities, cr.Communities)

	// Simple bubble sort by size (descending)
	// For large N, consider using sort.Slice
	for i := 0; i < len(communities)-1; i++ {
		for j := 0; j < len(communities)-i-1; j++ {
			if communities[j].Size < communities[j+1].Size {
				communities[j], communities[j+1] = communities[j+1], communities[j]
			}
		}
	}

	// Return top N
	if n > len(communities) {
		n = len(communities)
	}
	return communities[:n]
}

// CompareCommunities compares two community detection results
// Returns metrics like:
// - Normalized Mutual Information (NMI)
// - Adjusted Rand Index (ARI)
// For now, returns a simple overlap metric
func CompareCommunities(cr1, cr2 *CommunityResult) map[string]float64 {
	metrics := make(map[string]float64)

	// Simple overlap metric: fraction of nodes that are in the same community
	// in both results (relative to the first result)
	if len(cr1.Communities) == 0 || len(cr2.Communities) == 0 {
		metrics["overlap"] = 0.0
		return metrics
	}

	totalNodes := 0
	matchingNodes := 0

	for _, comm1 := range cr1.Communities {
		for _, nodeID := range comm1.NodeIDs {
			totalNodes++

			// Find which community this node belongs to in cr2
			comm2ID, found := cr2.GetNodeCommunity(nodeID)
			if !found {
				continue
			}

			// Check if other nodes in comm1 are also in the same community in cr2
			comm2, _ := cr2.GetCommunityByID(comm2ID)
			for _, otherNodeID := range comm1.NodeIDs {
				if otherNodeID == nodeID {
					continue
				}

				// Check if otherNodeID is also in comm2
				for _, nid := range comm2.NodeIDs {
					if nid == otherNodeID {
						matchingNodes++
						break
					}
				}
			}
		}
	}

	if totalNodes > 0 {
		metrics["overlap"] = float64(matchingNodes) / float64(totalNodes)
	} else {
		metrics["overlap"] = 0.0
	}

	return metrics
}
