package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kennethstott/doculyzer-go-conversion/internal/analytics"
	"github.com/kennethstott/doculyzer-go-conversion/internal/graph"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// GraphQueryExplorer provides MCP tools for querying and analyzing knowledge graphs
// Built on top of the graph.GraphQueryEngine for native Go graph operations
type GraphQueryExplorer struct {
	storage analytics.Storage // Access to Parquet storage for loading graphs
}

// NewGraphQueryExplorer creates a new MCP server for graph exploration
func NewGraphQueryExplorer(storage analytics.Storage) (*GraphQueryExplorer, error) {
	if storage == nil {
		return nil, fmt.Errorf("storage cannot be nil")
	}

	return &GraphQueryExplorer{
		storage: storage,
	}, nil
}

// CreateMCPServer creates and configures the MCP server with all graph tools
func (e *GraphQueryExplorer) CreateMCPServer() *server.MCPServer {
	s := server.NewMCPServer(
		"Knowledge Graph Explorer",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	// Tool 1: query_graph_nodes
	e.addQueryGraphNodesTool(s)

	// Tool 2: query_graph_edges
	e.addQueryGraphEdgesTool(s)

	// Tool 3: find_shortest_path
	e.addFindShortestPathTool(s)

	// Tool 4: get_neighbors
	e.addGetNeighborsTool(s)

	// Tool 5: get_subgraph
	e.addGetSubgraphTool(s)

	// Tool 6: analyze_communities
	e.addAnalyzeCommunitiesTool(s)

	// Tool 7: graph_statistics
	e.addGraphStatisticsTool(s)

	return s
}

// Tool 1: query_graph_nodes - Query nodes by various criteria
func (e *GraphQueryExplorer) addQueryGraphNodesTool(s *server.MCPServer) {
	queryNodesTool := mcp.NewTool("query_graph_nodes",
		mcp.WithDescription("Query nodes in the knowledge graph by label, property, domain, or confidence. Returns matching nodes with their properties."),
		mcp.WithString("run_id",
			mcp.Required(),
			mcp.Description("Run ID for the extraction (identifies the graph to query)"),
		),
		mcp.WithString("label",
			mcp.Description("Filter by node label (e.g., 'Entity', 'Community')"),
		),
		mcp.WithString("domain",
			mcp.Description("Filter by domain"),
		),
		mcp.WithNumber("min_confidence",
			mcp.Description("Minimum confidence threshold (0.0 to 1.0)"),
		),
		mcp.WithString("property_key",
			mcp.Description("Property key to filter by"),
		),
		mcp.WithString("property_value",
			mcp.Description("Property value to match (requires property_key)"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of results (default: 10)"),
		),
	)

	s.AddTool(queryNodesTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return e.handleQueryGraphNodes(ctx, request)
	})
}

// Tool 2: query_graph_edges - Query edges by type, source, target
func (e *GraphQueryExplorer) addQueryGraphEdgesTool(s *server.MCPServer) {
	queryEdgesTool := mcp.NewTool("query_graph_edges",
		mcp.WithDescription("Query edges in the knowledge graph by type, source node, or target node. Returns matching edges with their properties."),
		mcp.WithString("run_id",
			mcp.Required(),
			mcp.Description("Run ID for the extraction (identifies the graph to query)"),
		),
		mcp.WithString("edge_type",
			mcp.Description("Filter by edge type (e.g., 'BELONGS_TO', 'MENTIONED_BY')"),
		),
		mcp.WithString("source_id",
			mcp.Description("Filter by source node ID"),
		),
		mcp.WithString("target_id",
			mcp.Description("Filter by target node ID"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of results (default: 10)"),
		),
	)

	s.AddTool(queryEdgesTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return e.handleQueryGraphEdges(ctx, request)
	})
}

// Tool 3: find_shortest_path - Find shortest path between two nodes
func (e *GraphQueryExplorer) addFindShortestPathTool(s *server.MCPServer) {
	pathTool := mcp.NewTool("find_shortest_path",
		mcp.WithDescription("Find the shortest path between two nodes in the knowledge graph using BFS. Returns the path as a sequence of nodes and edges."),
		mcp.WithString("run_id",
			mcp.Required(),
			mcp.Description("Run ID for the extraction (identifies the graph to query)"),
		),
		mcp.WithString("start_node_id",
			mcp.Required(),
			mcp.Description("Starting node ID"),
		),
		mcp.WithString("end_node_id",
			mcp.Required(),
			mcp.Description("Ending node ID"),
		),
		mcp.WithNumber("max_depth",
			mcp.Description("Maximum path depth to search (default: 100)"),
		),
	)

	s.AddTool(pathTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return e.handleFindShortestPath(ctx, request)
	})
}

// Tool 4: get_neighbors - Get neighboring nodes
func (e *GraphQueryExplorer) addGetNeighborsTool(s *server.MCPServer) {
	neighborsTool := mcp.NewTool("get_neighbors",
		mcp.WithDescription("Get all neighboring nodes connected to a given node. Can filter by direction (incoming, outgoing, or both)."),
		mcp.WithString("run_id",
			mcp.Required(),
			mcp.Description("Run ID for the extraction (identifies the graph to query)"),
		),
		mcp.WithString("node_id",
			mcp.Required(),
			mcp.Description("Node ID to get neighbors for"),
		),
		mcp.WithString("direction",
			mcp.Description("Direction: 'incoming', 'outgoing', or 'both' (default: 'both')"),
			mcp.Enum("incoming", "outgoing", "both"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of neighbors to return (default: 10)"),
		),
	)

	s.AddTool(neighborsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return e.handleGetNeighbors(ctx, request)
	})
}

// Tool 5: get_subgraph - Extract a subgraph
func (e *GraphQueryExplorer) addGetSubgraphTool(s *server.MCPServer) {
	subgraphTool := mcp.NewTool("get_subgraph",
		mcp.WithDescription("Extract a subgraph containing only nodes matching specific criteria. Returns nodes and edges between them."),
		mcp.WithString("run_id",
			mcp.Required(),
			mcp.Description("Run ID for the extraction (identifies the graph to query)"),
		),
		mcp.WithString("label",
			mcp.Description("Filter nodes by label"),
		),
		mcp.WithString("domain",
			mcp.Description("Filter nodes by domain"),
		),
		mcp.WithNumber("min_confidence",
			mcp.Description("Minimum confidence threshold (0.0 to 1.0)"),
		),
	)

	s.AddTool(subgraphTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return e.handleGetSubgraph(ctx, request)
	})
}

// Tool 6: analyze_communities - Analyze community structure
func (e *GraphQueryExplorer) addAnalyzeCommunitiesTool(s *server.MCPServer) {
	communityTool := mcp.NewTool("analyze_communities",
		mcp.WithDescription("Detect and analyze communities in the knowledge graph using Louvain or Label Propagation algorithms. Returns community statistics and membership."),
		mcp.WithString("run_id",
			mcp.Required(),
			mcp.Description("Run ID for the extraction (identifies the graph to query)"),
		),
		mcp.WithString("algorithm",
			mcp.Description("Community detection algorithm: 'louvain' or 'label_propagation' (default: 'louvain')"),
			mcp.Enum("louvain", "label_propagation"),
		),
		mcp.WithNumber("top_n",
			mcp.Description("Number of top communities to return (default: 10)"),
		),
	)

	s.AddTool(communityTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return e.handleAnalyzeCommunities(ctx, request)
	})
}

// Tool 7: graph_statistics - Get graph statistics
func (e *GraphQueryExplorer) addGraphStatisticsTool(s *server.MCPServer) {
	statsTool := mcp.NewTool("graph_statistics",
		mcp.WithDescription("Compute statistics about the knowledge graph: node counts, edge counts, domain distribution, edge type distribution, etc."),
		mcp.WithString("run_id",
			mcp.Required(),
			mcp.Description("Run ID for the extraction (identifies the graph to query)"),
		),
		mcp.WithArray("metrics",
			mcp.Description("Metrics to compute: 'node_count', 'edge_count', 'domain_distribution', 'edge_type_distribution', 'label_distribution'"),
			mcp.WithStringItems(),
		),
	)

	s.AddTool(statsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return e.handleGraphStatistics(ctx, request)
	})
}

// Handler implementations

func (e *GraphQueryExplorer) handleQueryGraphNodes(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid arguments type")
	}

	runID, ok := args["run_id"].(string)
	if !ok {
		return nil, fmt.Errorf("run_id must be a string")
	}

	limit := 10
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	// Load graph from Parquet
	kg, err := e.loadGraphFromParquet(ctx, runID)
	if err != nil {
		return nil, err
	}

	// Create query engine
	qe := graph.NewGraphQueryEngine(kg)

	// Build filter based on arguments
	var filters []graph.NodeFilter

	if label, ok := args["label"].(string); ok {
		filters = append(filters, graph.HasLabelFilter(label))
	}

	if domain, ok := args["domain"].(string); ok {
		filters = append(filters, graph.DomainFilter(domain))
	}

	if minConf, ok := args["min_confidence"].(float64); ok {
		filters = append(filters, graph.ConfidenceAboveFilter(minConf))
	}

	if propKey, ok := args["property_key"].(string); ok {
		if propValue, ok := args["property_value"].(string); ok {
			filters = append(filters, graph.PropertyEqualsFilter(propKey, propValue))
		} else {
			filters = append(filters, graph.HasPropertyFilter(propKey))
		}
	}

	// Combine filters
	var finalFilter graph.NodeFilter
	if len(filters) == 0 {
		finalFilter = func(n graph.Node) bool { return true }
	} else if len(filters) == 1 {
		finalFilter = filters[0]
	} else {
		finalFilter = graph.AndNodeFilters(filters...)
	}

	// Execute query
	opts := graph.QueryOptions{Limit: limit}
	results := qe.FindNodes(finalFilter, opts)

	// Format results
	content, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to format results: %w", err)
	}

	return mcp.NewToolResultText(string(content)), nil
}

func (e *GraphQueryExplorer) handleQueryGraphEdges(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid arguments type")
	}

	runID, ok := args["run_id"].(string)
	if !ok {
		return nil, fmt.Errorf("run_id must be a string")
	}

	limit := 10
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	// Load graph
	kg, err := e.loadGraphFromParquet(ctx, runID)
	if err != nil {
		return nil, err
	}

	qe := graph.NewGraphQueryEngine(kg)
	opts := graph.QueryOptions{Limit: limit}

	var results []graph.Edge

	// Query based on provided filters
	if edgeType, ok := args["edge_type"].(string); ok {
		results = qe.FindEdgesByType(edgeType, opts)
	} else if sourceID, ok := args["source_id"].(string); ok {
		if targetID, ok := args["target_id"].(string); ok {
			results = qe.FindEdgesBetween(sourceID, targetID, opts)
		} else {
			results = qe.FindEdgesFrom(sourceID, opts)
		}
	} else if targetID, ok := args["target_id"].(string); ok {
		results = qe.FindEdgesTo(targetID, opts)
	} else {
		// Return all edges (limited)
		results = qe.FindEdges(func(e graph.Edge) bool { return true }, opts)
	}

	content, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to format results: %w", err)
	}

	return mcp.NewToolResultText(string(content)), nil
}

func (e *GraphQueryExplorer) handleFindShortestPath(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid arguments type")
	}

	runID, ok := args["run_id"].(string)
	if !ok {
		return nil, fmt.Errorf("run_id must be a string")
	}

	startID, ok := args["start_node_id"].(string)
	if !ok {
		return nil, fmt.Errorf("start_node_id must be a string")
	}

	endID, ok := args["end_node_id"].(string)
	if !ok {
		return nil, fmt.Errorf("end_node_id must be a string")
	}

	maxDepth := 100
	if md, ok := args["max_depth"].(float64); ok {
		maxDepth = int(md)
	}

	// Load graph
	kg, err := e.loadGraphFromParquet(ctx, runID)
	if err != nil {
		return nil, err
	}

	qe := graph.NewGraphQueryEngine(kg)
	path := qe.FindShortestPath(startID, endID, maxDepth)

	if path == nil {
		return mcp.NewToolResultText("No path found between the specified nodes"), nil
	}

	content, err := json.MarshalIndent(path, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to format results: %w", err)
	}

	return mcp.NewToolResultText(string(content)), nil
}

func (e *GraphQueryExplorer) handleGetNeighbors(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid arguments type")
	}

	runID, ok := args["run_id"].(string)
	if !ok {
		return nil, fmt.Errorf("run_id must be a string")
	}

	nodeID, ok := args["node_id"].(string)
	if !ok {
		return nil, fmt.Errorf("node_id must be a string")
	}

	direction := "both"
	if d, ok := args["direction"].(string); ok {
		direction = d
	}

	limit := 10
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	// Load graph
	kg, err := e.loadGraphFromParquet(ctx, runID)
	if err != nil {
		return nil, err
	}

	qe := graph.NewGraphQueryEngine(kg)

	var neighborDirection graph.NeighborDirection
	switch direction {
	case "incoming":
		neighborDirection = graph.IncomingNeighbors
	case "outgoing":
		neighborDirection = graph.OutgoingNeighbors
	default:
		neighborDirection = graph.BothNeighbors
	}

	opts := graph.QueryOptions{Limit: limit}
	neighbors := qe.GetNeighbors(nodeID, neighborDirection, opts)

	content, err := json.MarshalIndent(neighbors, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to format results: %w", err)
	}

	return mcp.NewToolResultText(string(content)), nil
}

func (e *GraphQueryExplorer) handleGetSubgraph(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid arguments type")
	}

	runID, ok := args["run_id"].(string)
	if !ok {
		return nil, fmt.Errorf("run_id must be a string")
	}

	// Load graph
	kg, err := e.loadGraphFromParquet(ctx, runID)
	if err != nil {
		return nil, err
	}

	qe := graph.NewGraphQueryEngine(kg)

	// Build filter
	var filters []graph.NodeFilter

	if label, ok := args["label"].(string); ok {
		filters = append(filters, graph.HasLabelFilter(label))
	}

	if domain, ok := args["domain"].(string); ok {
		filters = append(filters, graph.DomainFilter(domain))
	}

	if minConf, ok := args["min_confidence"].(float64); ok {
		filters = append(filters, graph.ConfidenceAboveFilter(minConf))
	}

	var finalFilter graph.NodeFilter
	if len(filters) == 0 {
		finalFilter = func(n graph.Node) bool { return true }
	} else if len(filters) == 1 {
		finalFilter = filters[0]
	} else {
		finalFilter = graph.AndNodeFilters(filters...)
	}

	subgraph := qe.GetSubgraph(finalFilter)

	// Format results (simplified version - just counts)
	result := map[string]interface{}{
		"id":         subgraph.ID,
		"name":       subgraph.Name,
		"node_count": len(subgraph.Nodes),
		"edge_count": len(subgraph.Edges),
		"metadata":   subgraph.Metadata,
	}

	content, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to format results: %w", err)
	}

	return mcp.NewToolResultText(string(content)), nil
}

func (e *GraphQueryExplorer) handleAnalyzeCommunities(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid arguments type")
	}

	runID, ok := args["run_id"].(string)
	if !ok {
		return nil, fmt.Errorf("run_id must be a string")
	}

	algorithm := "louvain"
	if a, ok := args["algorithm"].(string); ok {
		algorithm = a
	}

	topN := 10
	if tn, ok := args["top_n"].(float64); ok {
		topN = int(tn)
	}

	// Load graph
	kg, err := e.loadGraphFromParquet(ctx, runID)
	if err != nil {
		return nil, err
	}

	// Convert to gonum graph for analysis
	gonumGraph, err := graph.LoadFromGraph(kg)
	if err != nil {
		return nil, fmt.Errorf("failed to load graph for analysis: %w", err)
	}

	// Run community detection
	var communityAlg graph.CommunityDetectionAlgorithm
	if algorithm == "label_propagation" {
		communityAlg = graph.AlgorithmLabelPropagation
	} else {
		communityAlg = graph.AlgorithmLouvain
	}

	communities, err := graph.DetectCommunities(gonumGraph, communityAlg)
	if err != nil {
		return nil, fmt.Errorf("community detection failed: %w", err)
	}

	// Get top N communities
	topCommunities := communities.GetLargestCommunities(topN)

	result := map[string]interface{}{
		"algorithm":        string(communities.Algorithm),
		"total_communities": len(communities.Communities),
		"modularity":       communities.Modularity,
		"top_communities":  topCommunities,
	}

	content, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to format results: %w", err)
	}

	return mcp.NewToolResultText(string(content)), nil
}

func (e *GraphQueryExplorer) handleGraphStatistics(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid arguments type")
	}

	runID, ok := args["run_id"].(string)
	if !ok {
		return nil, fmt.Errorf("run_id must be a string")
	}

	// Load graph
	kg, err := e.loadGraphFromParquet(ctx, runID)
	if err != nil {
		return nil, err
	}

	// Collect requested metrics
	var metrics []string
	if m, ok := args["metrics"].([]interface{}); ok {
		for _, metric := range m {
			if str, ok := metric.(string); ok {
				metrics = append(metrics, str)
			}
		}
	}

	// Default metrics
	if len(metrics) == 0 {
		metrics = []string{"node_count", "edge_count", "domain_distribution", "edge_type_distribution"}
	}

	result := make(map[string]interface{})

	for _, metric := range metrics {
		switch metric {
		case "node_count":
			result["node_count"] = len(kg.Nodes)

		case "edge_count":
			result["edge_count"] = len(kg.Edges)

		case "domain_distribution":
			domains := make(map[string]int)
			for _, node := range kg.Nodes {
				domains[node.Domain]++
			}
			result["domain_distribution"] = domains

		case "edge_type_distribution":
			edgeTypes := make(map[string]int)
			for _, edge := range kg.Edges {
				edgeTypes[edge.Type]++
			}
			result["edge_type_distribution"] = edgeTypes

		case "label_distribution":
			labels := make(map[string]int)
			for _, node := range kg.Nodes {
				for _, label := range node.Labels {
					labels[label]++
				}
			}
			result["label_distribution"] = labels
		}
	}

	content, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to format results: %w", err)
	}

	return mcp.NewToolResultText(string(content)), nil
}

// Helper: Load graph from Parquet storage
func (e *GraphQueryExplorer) loadGraphFromParquet(ctx context.Context, runID string) (*graph.Graph, error) {
	// Query for entities
	filters := map[string]interface{}{
		"run_id": runID,
	}

	rawEntities, err := e.storage.QueryOntologyEntities(filters)
	if err != nil {
		return nil, fmt.Errorf("failed to query entities: %w", err)
	}

	// Query for relationships
	rawRelationships, err := e.storage.QueryOntologyRelationships(filters)
	if err != nil {
		return nil, fmt.Errorf("failed to query relationships: %w", err)
	}

	// Convert to ontology format
	// Note: This duplicates code from analyze_graph.go - could be extracted to a helper
	// But for now, we'll inline it here to keep the MCP tools self-contained

	// For simplicity, we'll create a minimal graph directly
	// In production, this should use the full ontology conversion pipeline

	converter := graph.NewOntologyConverter()
	converter.IncludeProvenance = true
	converter.IncludeMentions = false // Keep it simple for MCP queries

	// Create a minimal ontology structure
	// This is a simplified version - production code should use the full pipeline
	kg := &graph.Graph{
		ID:       "mcp_query_" + runID,
		Name:     "Knowledge Graph from " + runID,
		Version:  "1.0.0",
		Nodes:    make([]graph.Node, 0),
		Edges:    make([]graph.Edge, 0),
		Metadata: make(map[string]interface{}),
	}

	// Convert entities to nodes
	for _, raw := range rawEntities {
		node := graph.Node{
			ID:         raw.EntityID,
			Labels:     []string{"Entity"},
			Properties: raw.Attributes,
			Domain:     raw.Domain,
			Confidence: raw.Confidence,
			CreatedAt:  raw.ExtractedAt,
			UpdatedAt:  raw.ExtractedAt,
		}
		kg.Nodes = append(kg.Nodes, node)
	}

	// Convert relationships to edges
	for _, raw := range rawRelationships {
		edge := graph.Edge{
			ID:         raw.RelationshipID,
			Type:       raw.RelationshipType,
			SourceID:   raw.SourceEntityID,
			TargetID:   raw.TargetEntityID,
			Properties: raw.Attributes,
			Domain:     raw.Domain,
			Confidence: raw.Confidence,
			CreatedAt:  raw.ExtractedAt,
		}
		kg.Edges = append(kg.Edges, edge)
	}

	return kg, nil
}
