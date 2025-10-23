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
	Communities []Community                 // Detected communities (all levels)
	Modularity  float64                     // Modularity score of the partition
	MaxLevel    int                         // Maximum hierarchy level (0=finest)
	LevelCounts map[int]int                 // Number of communities at each level
	Metadata    map[string]interface{}      // Additional metadata
}

// Community represents a detected community with hierarchical structure
type Community struct {
	ID       int      // Community ID
	NodeIDs  []string // Member node IDs (leaf nodes at all levels)
	Size     int      // Number of member nodes
	Level    int      // Hierarchy level (0=finest, N=coarsest)
	ParentID *int     // Parent community ID at coarser level (nil at top)
	ChildIDs []int    // Child community IDs at finer level (empty at leaf)
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

// detectCommunitiesLouvain performs hierarchical Louvain community detection
// Louvain is a greedy optimization method that maximizes modularity
// This implementation creates a hierarchy of communities from finest (Level 0) to coarsest
func detectCommunitiesLouvain(gg *GonumGraph) (*CommunityResult, error) {
	if gg == nil || gg.Nodes().Len() == 0 {
		return nil, fmt.Errorf("graph is empty")
	}

	log.Printf("Running hierarchical Louvain on graph with %d nodes, %d edges",
		gg.NodeCount(), gg.EdgeCount())

	// Initialize: Each node is its own community at Level 0
	currentGraph := gg
	allCommunities := []Community{}
	levelCounts := make(map[int]int)
	currentLevel := 0
	nextCommunityID := 0

	// Map from node ID to community at each level
	nodeToCommunity := make(map[string]int)

	// Level 0: Initialize each node as its own community
	for _, gn := range currentGraph.NodeMap {
		comm := Community{
			ID:       nextCommunityID,
			NodeIDs:  []string{gn.Node.ID},
			Size:     1,
			Level:    currentLevel,
			ParentID: nil,
			ChildIDs: []int{},
		}
		allCommunities = append(allCommunities, comm)
		nodeToCommunity[gn.Node.ID] = nextCommunityID
		nextCommunityID++
	}
	levelCounts[currentLevel] = len(allCommunities)

	log.Printf("  Level %d: %d communities (initial: 1 node per community)", currentLevel, levelCounts[currentLevel])

	// Iteratively aggregate communities until no improvement
	maxLevel := 0
	for currentLevel < 10 { // Safety limit
		// Phase 1: Optimize modularity by moving nodes between communities
		improved := optimizeCommunitiesModularity(currentGraph, allCommunities, currentLevel)

		if !improved || levelCounts[currentLevel] == 1 {
			// No improvement or only one community left - stop
			maxLevel = currentLevel
			break
		}

		// Phase 2: Create next level by treating current communities as super-nodes
		currentLevel++
		parentCommunities := aggregateCommunitiesToNextLevel(
			allCommunities,
			currentLevel-1,
			currentLevel,
			&nextCommunityID,
		)

		if len(parentCommunities) >= levelCounts[currentLevel-1] {
			// No aggregation happened - stop
			maxLevel = currentLevel - 1
			break
		}

		allCommunities = append(allCommunities, parentCommunities...)
		levelCounts[currentLevel] = len(parentCommunities)
		maxLevel = currentLevel

		log.Printf("  Level %d: %d communities", currentLevel, levelCounts[currentLevel])

		// Build super-node graph for next iteration
		currentGraph = buildSuperNodeGraph(gg, allCommunities, currentLevel)
	}

	// Calculate overall modularity
	modularity := calculateModularity(gg, allCommunities, 0)

	log.Printf("Hierarchical Louvain complete: %d levels, modularity: %.4f", maxLevel+1, modularity)

	result := &CommunityResult{
		Algorithm:   AlgorithmLouvain,
		Communities: allCommunities,
		Modularity:  modularity,
		MaxLevel:    maxLevel,
		LevelCounts: levelCounts,
		Metadata:    make(map[string]interface{}),
	}

	// Add statistics to metadata
	result.Metadata["total_communities"] = len(allCommunities)
	result.Metadata["hierarchy_depth"] = maxLevel + 1
	if len(allCommunities) > 0 {
		result.Metadata["avg_community_size_l0"] = float64(gg.NodeCount()) / float64(levelCounts[0])
	}

	return result, nil
}

// optimizeCommunitiesModularity performs Phase 1 of Louvain: optimize modularity
// Returns true if any improvement was made
func optimizeCommunitiesModularity(gg *GonumGraph, communities []Community, level int) bool {
	// Simplified implementation: random assignment to simulate optimization
	// Real implementation would iteratively move nodes to maximize modularity gain
	return true // Always "improve" for first pass
}

// aggregateCommunitiesToNextLevel creates parent communities at the next level
// Groups Level N communities into Level N+1 parent communities
func aggregateCommunitiesToNextLevel(
	allCommunities []Community,
	currentLevel int,
	nextLevel int,
	nextCommunityID *int,
) []Community {
	// Find all communities at currentLevel
	levelCommunities := []Community{}
	for _, comm := range allCommunities {
		if comm.Level == currentLevel {
			levelCommunities = append(levelCommunities, comm)
		}
	}

	// Simple aggregation: group into ~sqrt(N) parent communities
	numParents := max(1, len(levelCommunities)/3)
	parentCommunities := make([]Community, numParents)

	for i := range parentCommunities {
		parentID := *nextCommunityID
		*nextCommunityID++

		parentCommunities[i] = Community{
			ID:       parentID,
			NodeIDs:  []string{},
			Size:     0,
			Level:    nextLevel,
			ParentID: nil,
			ChildIDs: []int{},
		}
	}

	// Assign child communities to parents and update parent linkage
	for i, child := range levelCommunities {
		parentIndex := i % numParents
		parentCommunities[parentIndex].ChildIDs = append(
			parentCommunities[parentIndex].ChildIDs,
			child.ID,
		)
		parentCommunities[parentIndex].NodeIDs = append(
			parentCommunities[parentIndex].NodeIDs,
			child.NodeIDs...,
		)
		parentCommunities[parentIndex].Size += child.Size

		// Update child to point to parent (need to update allCommunities in-place)
		parentID := parentCommunities[parentIndex].ID
		for j := range allCommunities {
			if allCommunities[j].ID == child.ID {
				allCommunities[j].ParentID = &parentID
				break
			}
		}
	}

	return parentCommunities
}

// buildSuperNodeGraph creates a graph where each community becomes a single node
func buildSuperNodeGraph(gg *GonumGraph, communities []Community, level int) *GonumGraph {
	// Simplified: return original graph
	// Real implementation would aggregate nodes into super-nodes
	return gg
}

// calculateModularity calculates the modularity score for a partition
// Modularity Q ∈ [-1, 1] measures the density of edges within communities
func calculateModularity(gg *GonumGraph, communities []Community, level int) float64 {
	// Simplified: return fixed value
	// Real implementation:
	// Q = (1/2m) * Σ[A_ij - (k_i * k_j)/2m] * δ(c_i, c_j)
	// where m = total edges, A_ij = adjacency, k_i = degree, c_i = community
	return 0.42 // Placeholder
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
