package mcp

import (
	"context"
	"fmt"
	"log"

	"github.com/kennethstott/doculyzer-go-conversion/internal/analytics"
)

// Implementation functions - all now delegate to analytics.Storage interface

func (e *OntologyCorpusExplorer) executeSemanticSearch(ctx context.Context, query string, elementTypes []string, limit int, threshold float64) ([]analytics.SearchResult, error) {
	log.Printf("[MCP:search_corpus:semantic] query=%q, element_types=%v, limit=%d, threshold=%.2f", query, elementTypes, limit, threshold)

	// Generate embedding for query
	queryEmb, err := e.embGenerator.Generate(query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Build filters for Storage query
	filters := make(map[string]interface{})
	filters["latest_only"] = true // Enable temporal filtering by default

	if len(elementTypes) > 0 {
		// Note: Storage interface doesn't support multiple element_type filters yet
		// For now, we'll need to filter results in-memory
		// TODO: Extend Storage interface to support element_type IN (...) queries
	}

	// Use Storage interface for semantic search
	results, err := e.storage.SearchSemanticSimilarity(queryEmb, filters, threshold, limit)
	if err != nil {
		return nil, fmt.Errorf("semantic search failed: %w", err)
	}

	// Filter by element type if specified (in-memory filtering for now)
	if len(elementTypes) > 0 {
		filtered := []analytics.SearchResult{}
		typeMap := make(map[string]bool)
		for _, t := range elementTypes {
			typeMap[t] = true
		}
		for _, result := range results {
			if typeMap[result.Element.ElementType] {
				filtered = append(filtered, result)
			}
		}
		return filtered, nil
	}

	return results, nil
}

func (e *OntologyCorpusExplorer) executeKeywordSearch(ctx context.Context, query string, elementTypes []string, limit int) ([]analytics.SearchResult, error) {
	log.Printf("[MCP:search_corpus:keyword] query=%q, element_types=%v, limit=%d", query, elementTypes, limit)

	// Build filters for Storage query
	filters := make(map[string]interface{})
	filters["latest_only"] = true // Enable temporal filtering by default

	// Use Storage interface for keyword search
	results, err := e.storage.SearchByKeyword(query, filters, limit)
	if err != nil {
		return nil, fmt.Errorf("keyword search failed: %w", err)
	}

	// Filter by element type if specified (in-memory filtering for now)
	if len(elementTypes) > 0 {
		filtered := []analytics.SearchResult{}
		typeMap := make(map[string]bool)
		for _, t := range elementTypes {
			typeMap[t] = true
		}
		for _, result := range results {
			if typeMap[result.Element.ElementType] {
				filtered = append(filtered, result)
			}
		}
		return filtered, nil
	}

	return results, nil
}

func (e *OntologyCorpusExplorer) executeRegexSearch(ctx context.Context, pattern string, elementTypes []string, limit int) ([]analytics.SearchResult, error) {
	log.Printf("[MCP:search_corpus:regex] pattern=%q, element_types=%v, limit=%d", pattern, elementTypes, limit)

	// Build filters for Storage query
	filters := make(map[string]interface{})
	filters["latest_only"] = true // Enable temporal filtering by default

	// Use Storage interface for regex search
	results, err := e.storage.SearchByRegex(pattern, filters, limit)
	if err != nil {
		return nil, fmt.Errorf("regex search failed: %w", err)
	}

	// Filter by element type if specified (in-memory filtering for now)
	if len(elementTypes) > 0 {
		filtered := []analytics.SearchResult{}
		typeMap := make(map[string]bool)
		for _, t := range elementTypes {
			typeMap[t] = true
		}
		for _, result := range results {
			if typeMap[result.Element.ElementType] {
				filtered = append(filtered, result)
			}
		}
		return filtered, nil
	}

	return results, nil
}

func (e *OntologyCorpusExplorer) analyzePattern(ctx context.Context, pattern string, elementTypes []string, maxExamples int) (*analytics.PatternStats, error) {
	log.Printf("[MCP:analyze_patterns] pattern=%q, element_types=%v, max_examples=%d", pattern, elementTypes, maxExamples)

	// Build filters for Storage query
	filters := make(map[string]interface{})
	filters["latest_only"] = true // Enable temporal filtering by default

	// Use Storage interface for pattern analysis
	result, err := e.storage.AnalyzePattern(pattern, filters, maxExamples)
	if err != nil {
		return nil, fmt.Errorf("pattern analysis failed: %w", err)
	}

	return result, nil
}

func (e *OntologyCorpusExplorer) computeFrequencies(ctx context.Context, terms []string, caseSensitive bool, elementTypes []string) ([]analytics.TermFrequency, error) {
	log.Printf("[MCP:compute_frequencies] terms=%v, case_sensitive=%v, element_types=%v", terms, caseSensitive, elementTypes)

	// OPTIMIZATION: Use semantic similarity to pre-filter to relevant elements
	// This avoids scanning the entire corpus for term frequencies

	// Step 1: Create semantic query from terms
	semanticQuery := ""
	for i, term := range terms {
		if i > 0 {
			semanticQuery += " "
		}
		semanticQuery += term
	}

	// Step 2: Use semantic search to find top 500-1000 relevant leaf elements
	leafElementTypes := []string{"paragraph", "text", "table_cell", "list_item", "code_block"}
	if len(elementTypes) > 0 {
		// Use provided element types (should already be leaf types)
		leafElementTypes = elementTypes
	}

	log.Printf("[MCP:compute_frequencies] Using semantic pre-filter: query=%q, leaf_types=%v", semanticQuery, leafElementTypes)

	relevantElements, err := e.executeSemanticSearch(ctx, semanticQuery, leafElementTypes, 1000, 0.5)
	if err != nil {
		return nil, fmt.Errorf("semantic pre-filtering failed: %w", err)
	}

	log.Printf("[MCP:compute_frequencies] Found %d semantically relevant elements (filtered from full corpus)", len(relevantElements))

	// Step 3: Extract element IDs for filtering
	elementIDs := make([]string, len(relevantElements))
	for i, elem := range relevantElements {
		elementIDs[i] = elem.Element.ElementID
	}

	// Step 4: Build filters for Storage query with element ID restriction
	filters := make(map[string]interface{})
	filters["latest_only"] = true
	filters["element_ids"] = elementIDs // Only count within semantically relevant elements

	// Step 5: Use Storage interface for frequency computation on filtered set
	results, err := e.storage.ComputeTermFrequencies(terms, caseSensitive, filters)
	if err != nil {
		return nil, fmt.Errorf("frequency computation failed: %w", err)
	}

	return results, nil
}

func (e *OntologyCorpusExplorer) findCooccurrences(ctx context.Context, entity1, entity2, contextWindow string, maxExamples int) (*analytics.CooccurrenceResult, error) {
	log.Printf("[MCP:find_cooccurrences] entity1=%q, entity2=%q, context_window=%s, max_examples=%d", entity1, entity2, contextWindow, maxExamples)

	// Build filters for Storage query
	filters := make(map[string]interface{})
	filters["latest_only"] = true // Enable temporal filtering by default

	// Use Storage interface for co-occurrence analysis
	result, err := e.storage.FindCooccurrences(entity1, entity2, contextWindow, filters, maxExamples)
	if err != nil {
		return nil, fmt.Errorf("cooccurrence search failed: %w", err)
	}

	return result, nil
}

func (e *OntologyCorpusExplorer) getElementContext(ctx context.Context, elementID string, contextDepth int, includeSiblings, includeChildren bool) (*analytics.ElementContext, error) {
	log.Printf("[MCP:get_element_context] element_id=%s, context_depth=%d, include_siblings=%v, include_children=%v", elementID, contextDepth, includeSiblings, includeChildren)

	// Build filters for Storage query
	filters := make(map[string]interface{})
	filters["latest_only"] = true // Enable temporal filtering by default

	// Use Storage interface for element context retrieval
	result, err := e.storage.GetElementContext(elementID, filters, contextDepth, includeSiblings, includeChildren)
	if err != nil {
		return nil, fmt.Errorf("failed to get element context: %w", err)
	}

	return result, nil
}

func (e *OntologyCorpusExplorer) aggregateStatistics(ctx context.Context, metrics []string, elementTypes []string) (*analytics.CorpusStats, error) {
	log.Printf("[MCP:aggregate_statistics] metrics=%v, element_types=%v", metrics, elementTypes)

	// Build filters for Storage query
	filters := make(map[string]interface{})
	filters["latest_only"] = true // Enable temporal filtering by default

	// Use Storage interface for aggregate statistics
	result, err := e.storage.AggregateStatistics(metrics, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to compute statistics: %w", err)
	}

	return result, nil
}
