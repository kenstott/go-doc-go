package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kennethstott/doculyzer-go-conversion/internal/embeddings"
	"github.com/kennethstott/doculyzer-go-conversion/internal/udml/query"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// OntologyCorpusExplorer provides MCP tools for exploring UDML corpus during ontology refinement
type OntologyCorpusExplorer struct {
	querier      *query.Querier
	embGenerator embeddings.EmbeddingGenerator
}

// NewOntologyCorpusExplorer creates a new MCP server for corpus exploration
func NewOntologyCorpusExplorer(querier *query.Querier, embGenerator embeddings.EmbeddingGenerator) *OntologyCorpusExplorer {
	return &OntologyCorpusExplorer{
		querier:      querier,
		embGenerator: embGenerator,
	}
}

// CreateMCPServer creates and configures the MCP server with all tools
func (e *OntologyCorpusExplorer) CreateMCPServer() *server.MCPServer {
	s := server.NewMCPServer(
		"Ontology Corpus Explorer",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	// Tool 1: search_corpus
	e.addSearchCorpusTool(s)

	// Tool 2: analyze_patterns
	e.addAnalyzePatternsTool(s)

	// Tool 3: compute_frequencies
	e.addComputeFrequenciesTool(s)

	// Tool 4: find_cooccurrences
	e.addFindCooccurrencesTool(s)

	// Tool 5: get_element_context
	e.addGetElementContextTool(s)

	// Tool 6: aggregate_statistics
	e.addAggregateStatisticsTool(s)

	return s
}

// Tool 1: search_corpus - Semantic/keyword/regex search across corpus
func (e *OntologyCorpusExplorer) addSearchCorpusTool(s *server.MCPServer) {
	searchTool := mcp.NewTool("search_corpus",
		mcp.WithDescription("Search the UDML corpus using semantic similarity, keywords, or regex patterns. Returns matching elements with context."),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("Search query (text for semantic search, keywords, or regex pattern)"),
		),
		mcp.WithString("search_type",
			mcp.Required(),
			mcp.Description("Search type: 'semantic', 'keyword', or 'regex'"),
			mcp.Enum("semantic", "keyword", "regex"),
		),
		mcp.WithArray("element_types",
			mcp.Description("Filter by element types (paragraph, table_cell, heading, etc.)"),
			mcp.Items(mcp.ItemsString()),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of results (default: 10)"),
		),
		mcp.WithNumber("similarity_threshold",
			mcp.Description("Minimum similarity score for semantic search (default: 0.7)"),
		),
	)

	s.AddTool(searchTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return e.handleSearchCorpus(ctx, request)
	})
}

// Tool 2: analyze_patterns - Analyze regex pattern matches across corpus
func (e *OntologyCorpusExplorer) addAnalyzePatternsTool(s *server.MCPServer) {
	patternTool := mcp.NewTool("analyze_patterns",
		mcp.WithDescription("Analyze how a regex pattern matches across the corpus. Returns match examples, frequency, and element type distribution."),
		mcp.WithString("pattern",
			mcp.Required(),
			mcp.Description("Regex pattern to analyze"),
		),
		mcp.WithArray("element_types",
			mcp.Description("Filter by element types"),
			mcp.Items(mcp.ItemsString()),
		),
		mcp.WithNumber("max_examples",
			mcp.Description("Maximum examples to return (default: 20)"),
		),
	)

	s.AddTool(patternTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return e.handleAnalyzePatterns(ctx, request)
	})
}

// Tool 3: compute_frequencies - Compute term/entity frequencies
func (e *OntologyCorpusExplorer) addComputeFrequenciesTool(s *server.MCPServer) {
	freqTool := mcp.NewTool("compute_frequencies",
		mcp.WithDescription("Compute frequency counts for terms or entities across the corpus. Useful for understanding what concepts are most prominent."),
		mcp.WithArray("terms",
			mcp.Required(),
			mcp.Description("List of terms/entities to count"),
			mcp.Items(mcp.ItemsString()),
		),
		mcp.WithBoolean("case_sensitive",
			mcp.Description("Whether search is case-sensitive (default: false)"),
		),
		mcp.WithArray("element_types",
			mcp.Description("Filter by element types"),
			mcp.Items(mcp.ItemsString()),
		),
	)

	s.AddTool(freqTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return e.handleComputeFrequencies(ctx, request)
	})
}

// Tool 4: find_cooccurrences - Find entity co-occurrences
func (e *OntologyCorpusExplorer) addFindCooccurrencesTool(s *server.MCPServer) {
	cooccurTool := mcp.NewTool("find_cooccurrences",
		mcp.WithDescription("Find how often two entities appear together in the same context. Useful for discovering relationships."),
		mcp.WithString("entity1",
			mcp.Required(),
			mcp.Description("First entity/term to search for"),
		),
		mcp.WithString("entity2",
			mcp.Required(),
			mcp.Description("Second entity/term to search for"),
		),
		mcp.WithString("context_window",
			mcp.Description("Context window: 'element', 'paragraph', or 'document' (default: 'element')"),
			mcp.Enum("element", "paragraph", "document"),
		),
		mcp.WithNumber("max_examples",
			mcp.Description("Maximum examples to return (default: 10)"),
		),
	)

	s.AddTool(cooccurTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return e.handleFindCooccurrences(ctx, request)
	})
}

// Tool 5: get_element_context - Get surrounding context for elements
func (e *OntologyCorpusExplorer) addGetElementContextTool(s *server.MCPServer) {
	contextTool := mcp.NewTool("get_element_context",
		mcp.WithDescription("Retrieve surrounding context for specific elements. Shows parent/sibling/child elements to understand document structure."),
		mcp.WithString("element_id",
			mcp.Required(),
			mcp.Description("Element ID to get context for"),
		),
		mcp.WithNumber("context_depth",
			mcp.Description("Number of parent levels to traverse (default: 2)"),
		),
		mcp.WithBoolean("include_siblings",
			mcp.Description("Include sibling elements (default: true)"),
		),
		mcp.WithBoolean("include_children",
			mcp.Description("Include child elements (default: true)"),
		),
	)

	s.AddTool(contextTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return e.handleGetElementContext(ctx, request)
	})
}

// Tool 6: aggregate_statistics - Aggregate corpus statistics
func (e *OntologyCorpusExplorer) addAggregateStatisticsTool(s *server.MCPServer) {
	statsTool := mcp.NewTool("aggregate_statistics",
		mcp.WithDescription("Compute aggregate statistics about the corpus: element type distribution, document counts, average lengths, etc."),
		mcp.WithArray("metrics",
			mcp.Description("Metrics to compute: 'element_type_distribution', 'document_count', 'avg_content_length', 'entity_type_distribution'"),
			mcp.Items(mcp.ItemsString()),
		),
		mcp.WithArray("element_types",
			mcp.Description("Filter by element types"),
			mcp.Items(mcp.ItemsString()),
		),
	)

	s.AddTool(statsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return e.handleAggregateStatistics(ctx, request)
	})
}

// Handler implementations

func (e *OntologyCorpusExplorer) handleSearchCorpus(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Parse arguments
	args := request.Params.Arguments
	query, ok := args["query"].(string)
	if !ok {
		return nil, fmt.Errorf("query must be a string")
	}

	searchType, ok := args["search_type"].(string)
	if !ok {
		return nil, fmt.Errorf("search_type must be a string")
	}

	limit := 10
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	var elementTypes []string
	if et, ok := args["element_types"].([]interface{}); ok {
		for _, t := range et {
			if str, ok := t.(string); ok {
				elementTypes = append(elementTypes, str)
			}
		}
	}

	// Execute search based on type
	var results interface{}
	var err error

	switch searchType {
	case "semantic":
		similarityThreshold := 0.7
		if st, ok := args["similarity_threshold"].(float64); ok {
			similarityThreshold = st
		}
		results, err = e.executeSemanticSearch(ctx, query, elementTypes, limit, similarityThreshold)

	case "keyword":
		results, err = e.executeKeywordSearch(ctx, query, elementTypes, limit)

	case "regex":
		results, err = e.executeRegexSearch(ctx, query, elementTypes, limit)

	default:
		return nil, fmt.Errorf("unsupported search_type: %s", searchType)
	}

	if err != nil {
		return nil, err
	}

	// Format results
	content, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to format results: %w", err)
	}

	return mcp.NewToolResultText(string(content)), nil
}

func (e *OntologyCorpusExplorer) handleAnalyzePatterns(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.Params.Arguments
	pattern, ok := args["pattern"].(string)
	if !ok {
		return nil, fmt.Errorf("pattern must be a string")
	}

	maxExamples := 20
	if me, ok := args["max_examples"].(float64); ok {
		maxExamples = int(me)
	}

	var elementTypes []string
	if et, ok := args["element_types"].([]interface{}); ok {
		for _, t := range et {
			if str, ok := t.(string); ok {
				elementTypes = append(elementTypes, str)
			}
		}
	}

	results, err := e.analyzePattern(ctx, pattern, elementTypes, maxExamples)
	if err != nil {
		return nil, err
	}

	content, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to format results: %w", err)
	}

	return mcp.NewToolResultText(string(content)), nil
}

func (e *OntologyCorpusExplorer) handleComputeFrequencies(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.Params.Arguments

	var terms []string
	if t, ok := args["terms"].([]interface{}); ok {
		for _, term := range t {
			if str, ok := term.(string); ok {
				terms = append(terms, str)
			}
		}
	} else {
		return nil, fmt.Errorf("terms must be an array of strings")
	}

	caseSensitive := false
	if cs, ok := args["case_sensitive"].(bool); ok {
		caseSensitive = cs
	}

	var elementTypes []string
	if et, ok := args["element_types"].([]interface{}); ok {
		for _, t := range et {
			if str, ok := t.(string); ok {
				elementTypes = append(elementTypes, str)
			}
		}
	}

	results, err := e.computeFrequencies(ctx, terms, caseSensitive, elementTypes)
	if err != nil {
		return nil, err
	}

	content, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to format results: %w", err)
	}

	return mcp.NewToolResultText(string(content)), nil
}

func (e *OntologyCorpusExplorer) handleFindCooccurrences(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.Params.Arguments

	entity1, ok := args["entity1"].(string)
	if !ok {
		return nil, fmt.Errorf("entity1 must be a string")
	}

	entity2, ok := args["entity2"].(string)
	if !ok {
		return nil, fmt.Errorf("entity2 must be a string")
	}

	contextWindow := "element"
	if cw, ok := args["context_window"].(string); ok {
		contextWindow = cw
	}

	maxExamples := 10
	if me, ok := args["max_examples"].(float64); ok {
		maxExamples = int(me)
	}

	results, err := e.findCooccurrences(ctx, entity1, entity2, contextWindow, maxExamples)
	if err != nil {
		return nil, err
	}

	content, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to format results: %w", err)
	}

	return mcp.NewToolResultText(string(content)), nil
}

func (e *OntologyCorpusExplorer) handleGetElementContext(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.Params.Arguments

	elementID, ok := args["element_id"].(string)
	if !ok {
		return nil, fmt.Errorf("element_id must be a string")
	}

	contextDepth := 2
	if cd, ok := args["context_depth"].(float64); ok {
		contextDepth = int(cd)
	}

	includeSiblings := true
	if is, ok := args["include_siblings"].(bool); ok {
		includeSiblings = is
	}

	includeChildren := true
	if ic, ok := args["include_children"].(bool); ok {
		includeChildren = ic
	}

	results, err := e.getElementContext(ctx, elementID, contextDepth, includeSiblings, includeChildren)
	if err != nil {
		return nil, err
	}

	content, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to format results: %w", err)
	}

	return mcp.NewToolResultText(string(content)), nil
}

func (e *OntologyCorpusExplorer) handleAggregateStatistics(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.Params.Arguments

	var metrics []string
	if m, ok := args["metrics"].([]interface{}); ok {
		for _, metric := range m {
			if str, ok := metric.(string); ok {
				metrics = append(metrics, str)
			}
		}
	}

	// Default metrics if none specified
	if len(metrics) == 0 {
		metrics = []string{"element_type_distribution", "document_count", "avg_content_length"}
	}

	var elementTypes []string
	if et, ok := args["element_types"].([]interface{}); ok {
		for _, t := range et {
			if str, ok := t.(string); ok {
				elementTypes = append(elementTypes, str)
			}
		}
	}

	results, err := e.aggregateStatistics(ctx, metrics, elementTypes)
	if err != nil {
		return nil, err
	}

	content, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to format results: %w", err)
	}

	return mcp.NewToolResultText(string(content)), nil
}
