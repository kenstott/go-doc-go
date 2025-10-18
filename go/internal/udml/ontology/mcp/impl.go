package mcp

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
)

// Search result types
type SearchResult struct {
	ElementID      string            `json:"element_id"`
	DocumentID     string            `json:"document_id"`
	ElementType    string            `json:"element_type"`
	ContentPreview string            `json:"content_preview"`
	Similarity     float64           `json:"similarity,omitempty"`
	MatchCount     int               `json:"match_count,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

type PatternAnalysis struct {
	Pattern            string            `json:"pattern"`
	TotalMatches       int               `json:"total_matches"`
	DocumentCount      int               `json:"document_count"`
	ElementTypeDistrib map[string]int    `json:"element_type_distribution"`
	Examples           []PatternExample  `json:"examples"`
}

type PatternExample struct {
	Match          string `json:"match"`
	Context        string `json:"context"`
	ElementType    string `json:"element_type"`
	DocumentID     string `json:"document_id"`
}

type FrequencyResult struct {
	Term           string         `json:"term"`
	Frequency      int            `json:"frequency"`
	DocumentCount  int            `json:"document_count"`
	ElementTypeDistrib map[string]int `json:"element_type_distribution"`
}

type CooccurrenceResult struct {
	Entity1        string                 `json:"entity1"`
	Entity2        string                 `json:"entity2"`
	CooccurCount   int                    `json:"cooccurrence_count"`
	ContextWindow  string                 `json:"context_window"`
	Examples       []CooccurrenceExample  `json:"examples"`
}

type CooccurrenceExample struct {
	DocumentID  string `json:"document_id"`
	ElementType string `json:"element_type"`
	Context     string `json:"context"`
}

type ElementContext struct {
	Element       map[string]interface{}   `json:"element"`
	Parents       []map[string]interface{} `json:"parents,omitempty"`
	Siblings      []map[string]interface{} `json:"siblings,omitempty"`
	Children      []map[string]interface{} `json:"children,omitempty"`
}

type Statistics struct {
	ElementTypeDistribution map[string]int    `json:"element_type_distribution,omitempty"`
	DocumentCount           int               `json:"document_count,omitempty"`
	AvgContentLength        float64           `json:"avg_content_length,omitempty"`
	EntityTypeDistribution  map[string]int    `json:"entity_type_distribution,omitempty"`
}

// Implementation functions

func (e *OntologyCorpusExplorer) executeSemanticSearch(ctx context.Context, query string, elementTypes []string, limit int, threshold float64) ([]SearchResult, error) {
	log.Printf("[MCP:search_corpus:semantic] query=%q, element_types=%v, limit=%d, threshold=%.2f", query, elementTypes, limit, threshold)

	// Generate embedding for query
	queryEmb, err := e.embGenerator.Generate(query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Build DuckDB query for similarity search
	var typeFilter string
	if len(elementTypes) > 0 {
		typeFilter = fmt.Sprintf("AND element_type IN (%s)", joinQuoted(elementTypes))
	}

	duckdbQuery := fmt.Sprintf(`
		SELECT
			element_id,
			document_id,
			element_type,
			content_preview,
			list_cosine_similarity(embedding, $1::FLOAT[]) as similarity
		FROM udml_elements
		WHERE embedding IS NOT NULL
		%s
		AND list_cosine_similarity(embedding, $1::FLOAT[]) >= $2
		ORDER BY similarity DESC
		LIMIT $3
	`, typeFilter)

	rows, err := e.queryRaw(ctx, duckdbQuery, queryEmb, threshold, limit)
	if err != nil {
		return nil, fmt.Errorf("semantic search failed: %w", err)
	}
	defer rows.Close()

	results := []SearchResult{}
	for rows.Next() {
		var r SearchResult
		if err := rows.Scan(&r.ElementID, &r.DocumentID, &r.ElementType, &r.ContentPreview, &r.Similarity); err != nil {
			return nil, err
		}
		results = append(results, r)
	}

	return results, nil
}

func (e *OntologyCorpusExplorer) executeKeywordSearch(ctx context.Context, query string, elementTypes []string, limit int) ([]SearchResult, error) {
	log.Printf("[MCP:search_corpus:keyword] query=%q, element_types=%v, limit=%d", query, elementTypes, limit)

	var typeFilter string
	if len(elementTypes) > 0 {
		typeFilter = fmt.Sprintf("AND element_type IN (%s)", joinQuoted(elementTypes))
	}

	duckdbQuery := fmt.Sprintf(`
		SELECT
			element_id,
			document_id,
			element_type,
			content_preview,
			(LENGTH(content) - LENGTH(REPLACE(LOWER(content), LOWER($1), ''))) / LENGTH($1) as match_count
		FROM udml_elements
		WHERE LOWER(content) LIKE LOWER('%%' || $1 || '%%')
		%s
		ORDER BY match_count DESC
		LIMIT $2
	`, typeFilter)

	rows, err := e.queryRaw(ctx, duckdbQuery, query, limit)
	if err != nil {
		return nil, fmt.Errorf("keyword search failed: %w", err)
	}
	defer rows.Close()

	results := []SearchResult{}
	for rows.Next() {
		var r SearchResult
		if err := rows.Scan(&r.ElementID, &r.DocumentID, &r.ElementType, &r.ContentPreview, &r.MatchCount); err != nil {
			return nil, err
		}
		results = append(results, r)
	}

	return results, nil
}

func (e *OntologyCorpusExplorer) executeRegexSearch(ctx context.Context, pattern string, elementTypes []string, limit int) ([]SearchResult, error) {
	log.Printf("[MCP:search_corpus:regex] pattern=%q, element_types=%v, limit=%d", pattern, elementTypes, limit)

	// Validate regex pattern
	_, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %w", err)
	}

	var typeFilter string
	if len(elementTypes) > 0 {
		typeFilter = fmt.Sprintf("AND element_type IN (%s)", joinQuoted(elementTypes))
	}

	duckdbQuery := fmt.Sprintf(`
		SELECT
			element_id,
			document_id,
			element_type,
			content_preview
		FROM udml_elements
		WHERE regexp_matches(content, $1)
		%s
		LIMIT $2
	`, typeFilter)

	rows, err := e.queryRaw(ctx, duckdbQuery, pattern, limit)
	if err != nil {
		return nil, fmt.Errorf("regex search failed: %w", err)
	}
	defer rows.Close()

	results := []SearchResult{}
	for rows.Next() {
		var r SearchResult
		if err := rows.Scan(&r.ElementID, &r.DocumentID, &r.ElementType, &r.ContentPreview); err != nil {
			return nil, err
		}
		results = append(results, r)
	}

	return results, nil
}

func (e *OntologyCorpusExplorer) analyzePattern(ctx context.Context, pattern string, elementTypes []string, maxExamples int) (*PatternAnalysis, error) {
	log.Printf("[MCP:analyze_patterns] pattern=%q, element_types=%v, max_examples=%d", pattern, elementTypes, maxExamples)

	// Validate regex
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %w", err)
	}

	var typeFilter string
	if len(elementTypes) > 0 {
		typeFilter = fmt.Sprintf("AND element_type IN (%s)", joinQuoted(elementTypes))
	}

	// Get total matches and distribution
	duckdbQuery := fmt.Sprintf(`
		SELECT
			COUNT(*) as total_matches,
			COUNT(DISTINCT document_id) as document_count,
			element_type,
			COUNT(*) as type_count
		FROM udml_elements
		WHERE regexp_matches(content, $1)
		%s
		GROUP BY element_type
	`, typeFilter)

	rows, err := e.queryRaw(ctx, duckdbQuery, pattern)
	if err != nil {
		return nil, fmt.Errorf("pattern analysis failed: %w", err)
	}
	defer rows.Close()

	result := &PatternAnalysis{
		Pattern:            pattern,
		ElementTypeDistrib: make(map[string]int),
		Examples:           []PatternExample{},
	}

	for rows.Next() {
		var totalMatches, docCount, typeCount int
		var elementType string
		if err := rows.Scan(&totalMatches, &docCount, &elementType, &typeCount); err != nil {
			return nil, err
		}
		result.TotalMatches += totalMatches
		result.DocumentCount = docCount
		result.ElementTypeDistrib[elementType] = typeCount
	}

	// Get examples
	exampleQuery := fmt.Sprintf(`
		SELECT
			content,
			element_type,
			document_id
		FROM udml_elements
		WHERE regexp_matches(content, $1)
		%s
		LIMIT $2
	`, typeFilter)

	exampleRows, err := e.queryRaw(ctx, exampleQuery, pattern, maxExamples)
	if err != nil {
		return nil, fmt.Errorf("failed to get examples: %w", err)
	}
	defer exampleRows.Close()

	for exampleRows.Next() {
		var content, elementType, documentID string
		if err := exampleRows.Scan(&content, &elementType, &documentID); err != nil {
			return nil, err
		}

		// Extract match
		matches := re.FindStringSubmatch(content)
		match := ""
		if len(matches) > 0 {
			match = matches[0]
		}

		result.Examples = append(result.Examples, PatternExample{
			Match:       match,
			Context:     truncate(content, 200),
			ElementType: elementType,
			DocumentID:  documentID,
		})
	}

	return result, nil
}

func (e *OntologyCorpusExplorer) computeFrequencies(ctx context.Context, terms []string, caseSensitive bool, elementTypes []string) ([]FrequencyResult, error) {
	log.Printf("[MCP:compute_frequencies] terms=%v, case_sensitive=%v, element_types=%v", terms, caseSensitive, elementTypes)

	results := []FrequencyResult{}

	for _, term := range terms {
		var typeFilter string
		if len(elementTypes) > 0 {
			typeFilter = fmt.Sprintf("AND element_type IN (%s)", joinQuoted(elementTypes))
		}

		comparison := "LOWER(content) LIKE LOWER"
		if caseSensitive {
			comparison = "content LIKE"
		}

		duckdbQuery := fmt.Sprintf(`
			SELECT
				COUNT(*) as frequency,
				COUNT(DISTINCT document_id) as document_count,
				element_type,
				COUNT(*) as type_count
			FROM udml_elements
			WHERE %s('%%' || $1 || '%%')
			%s
			GROUP BY element_type
		`, comparison, typeFilter)

		rows, err := e.queryRaw(ctx, duckdbQuery, term)
		if err != nil {
			return nil, fmt.Errorf("frequency computation failed for %s: %w", term, err)
		}

		result := FrequencyResult{
			Term:               term,
			ElementTypeDistrib: make(map[string]int),
		}

		for rows.Next() {
			var freq, docCount, typeCount int
			var elementType string
			if err := rows.Scan(&freq, &docCount, &elementType, &typeCount); err != nil {
				rows.Close()
				return nil, err
			}
			result.Frequency += freq
			result.DocumentCount = docCount
			result.ElementTypeDistrib[elementType] = typeCount
		}
		rows.Close()

		results = append(results, result)
	}

	return results, nil
}

func (e *OntologyCorpusExplorer) findCooccurrences(ctx context.Context, entity1, entity2, contextWindow string, maxExamples int) (*CooccurrenceResult, error) {
	log.Printf("[MCP:find_cooccurrences] entity1=%q, entity2=%q, context_window=%s, max_examples=%d", entity1, entity2, contextWindow, maxExamples)

	// Build query based on context window
	var joinCondition string
	switch contextWindow {
	case "element":
		joinCondition = "WHERE LOWER(e.content) LIKE LOWER('%' || $1 || '%') AND LOWER(e.content) LIKE LOWER('%' || $2 || '%')"
	case "paragraph":
		joinCondition = `
			WHERE e.element_id IN (
				SELECT element_id FROM udml_elements
				WHERE element_type = 'paragraph'
				AND (LOWER(content) LIKE LOWER('%' || $1 || '%') OR LOWER(content) LIKE LOWER('%' || $2 || '%'))
			)
		`
	case "document":
		joinCondition = `
			WHERE e.document_id IN (
				SELECT DISTINCT document_id FROM udml_elements
				WHERE LOWER(content) LIKE LOWER('%' || $1 || '%')
			)
			AND e.document_id IN (
				SELECT DISTINCT document_id FROM udml_elements
				WHERE LOWER(content) LIKE LOWER('%' || $2 || '%')
			)
		`
	default:
		joinCondition = "WHERE LOWER(e.content) LIKE LOWER('%' || $1 || '%') AND LOWER(e.content) LIKE LOWER('%' || $2 || '%')"
	}

	duckdbQuery := fmt.Sprintf(`
		SELECT
			COUNT(*) as cooccur_count,
			e.document_id,
			e.element_type,
			e.content
		FROM udml_elements e
		%s
		GROUP BY e.document_id, e.element_type, e.content
		LIMIT $3
	`, joinCondition)

	rows, err := e.queryRaw(ctx, duckdbQuery, entity1, entity2, maxExamples)
	if err != nil {
		return nil, fmt.Errorf("cooccurrence search failed: %w", err)
	}
	defer rows.Close()

	result := &CooccurrenceResult{
		Entity1:       entity1,
		Entity2:       entity2,
		ContextWindow: contextWindow,
		Examples:      []CooccurrenceExample{},
	}

	for rows.Next() {
		var count int
		var documentID, elementType, content string
		if err := rows.Scan(&count, &documentID, &elementType, &content); err != nil {
			return nil, err
		}
		result.CooccurCount += count
		result.Examples = append(result.Examples, CooccurrenceExample{
			DocumentID:  documentID,
			ElementType: elementType,
			Context:     truncate(content, 300),
		})
	}

	return result, nil
}

func (e *OntologyCorpusExplorer) getElementContext(ctx context.Context, elementID string, contextDepth int, includeSiblings, includeChildren bool) (*ElementContext, error) {
	log.Printf("[MCP:get_element_context] element_id=%s, context_depth=%d, include_siblings=%v, include_children=%v", elementID, contextDepth, includeSiblings, includeChildren)

	// Get the element itself
	elemQuery := `
		SELECT element_id, document_id, element_type, content_preview, parent_id, metadata
		FROM udml_elements
		WHERE element_id = $1
	`

	rows, err := e.queryRaw(ctx, elemQuery, elementID)
	if err != nil {
		return nil, fmt.Errorf("failed to get element: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, fmt.Errorf("element not found: %s", elementID)
	}

	result := &ElementContext{
		Element:  make(map[string]interface{}),
		Parents:  []map[string]interface{}{},
		Siblings: []map[string]interface{}{},
		Children: []map[string]interface{}{},
	}

	var parentID *string
	var elemData struct {
		ID, DocID, ElemType, Preview string
		Metadata                      string
	}

	if err := rows.Scan(&elemData.ID, &elemData.DocID, &elemData.ElemType, &elemData.Preview, &parentID, &elemData.Metadata); err != nil {
		return nil, err
	}

	result.Element["element_id"] = elemData.ID
	result.Element["document_id"] = elemData.DocID
	result.Element["element_type"] = elemData.ElemType
	result.Element["content_preview"] = elemData.Preview

	// Get parents
	if parentID != nil && contextDepth > 0 {
		// Recursively get parents up to contextDepth
		currentParentID := *parentID
		for depth := 0; depth < contextDepth; depth++ {
			parentQuery := `
				SELECT element_id, element_type, content_preview, parent_id
				FROM udml_elements
				WHERE element_id = $1
			`
			parentRows, err := e.queryRaw(ctx, parentQuery, currentParentID)
			if err != nil {
				break
			}

			if !parentRows.Next() {
				parentRows.Close()
				break
			}

			var nextParentID *string
			parentElem := make(map[string]interface{})
			var pID, pType, pPreview string
			if err := parentRows.Scan(&pID, &pType, &pPreview, &nextParentID); err != nil {
				parentRows.Close()
				break
			}

			parentElem["element_id"] = pID
			parentElem["element_type"] = pType
			parentElem["content_preview"] = pPreview
			result.Parents = append(result.Parents, parentElem)

			parentRows.Close()

			if nextParentID == nil {
				break
			}
			currentParentID = *nextParentID
		}
	}

	// Get siblings
	if includeSiblings && parentID != nil {
		siblingQuery := `
			SELECT element_id, element_type, content_preview
			FROM udml_elements
			WHERE parent_id = $1 AND element_id != $2
			LIMIT 10
		`
		siblingRows, err := e.queryRaw(ctx, siblingQuery, *parentID, elementID)
		if err == nil {
			for siblingRows.Next() {
				sibling := make(map[string]interface{})
				var sID, sType, sPreview string
				if err := siblingRows.Scan(&sID, &sType, &sPreview); err != nil {
					break
				}
				sibling["element_id"] = sID
				sibling["element_type"] = sType
				sibling["content_preview"] = sPreview
				result.Siblings = append(result.Siblings, sibling)
			}
			siblingRows.Close()
		}
	}

	// Get children
	if includeChildren {
		childQuery := `
			SELECT element_id, element_type, content_preview
			FROM udml_elements
			WHERE parent_id = $1
			LIMIT 20
		`
		childRows, err := e.queryRaw(ctx, childQuery, elementID)
		if err == nil {
			for childRows.Next() {
				child := make(map[string]interface{})
				var cID, cType, cPreview string
				if err := childRows.Scan(&cID, &cType, &cPreview); err != nil {
					break
				}
				child["element_id"] = cID
				child["element_type"] = cType
				child["content_preview"] = cPreview
				result.Children = append(result.Children, child)
			}
			childRows.Close()
		}
	}

	return result, nil
}

func (e *OntologyCorpusExplorer) aggregateStatistics(ctx context.Context, metrics []string, elementTypes []string) (*Statistics, error) {
	log.Printf("[MCP:aggregate_statistics] metrics=%v, element_types=%v", metrics, elementTypes)

	result := &Statistics{
		ElementTypeDistribution: make(map[string]int),
		EntityTypeDistribution:  make(map[string]int),
	}

	var typeFilter string
	if len(elementTypes) > 0 {
		typeFilter = fmt.Sprintf("WHERE element_type IN (%s)", joinQuoted(elementTypes))
	}

	for _, metric := range metrics {
		switch metric {
		case "element_type_distribution":
			query := fmt.Sprintf(`
				SELECT element_type, COUNT(*) as count
				FROM udml_elements
				%s
				GROUP BY element_type
			`, typeFilter)

			rows, err := e.queryRaw(ctx, query)
			if err != nil {
				return nil, fmt.Errorf("failed to compute element_type_distribution: %w", err)
			}

			for rows.Next() {
				var elemType string
				var count int
				if err := rows.Scan(&elemType, &count); err != nil {
					rows.Close()
					return nil, err
				}
				result.ElementTypeDistribution[elemType] = count
			}
			rows.Close()

		case "document_count":
			query := fmt.Sprintf(`
				SELECT COUNT(DISTINCT document_id) as doc_count
				FROM udml_elements
				%s
			`, typeFilter)

			rows, err := e.queryRaw(ctx, query)
			if err != nil {
				return nil, fmt.Errorf("failed to compute document_count: %w", err)
			}

			if rows.Next() {
				if err := rows.Scan(&result.DocumentCount); err != nil {
					rows.Close()
					return nil, err
				}
			}
			rows.Close()

		case "avg_content_length":
			query := fmt.Sprintf(`
				SELECT AVG(LENGTH(content)) as avg_length
				FROM udml_elements
				%s
			`, typeFilter)

			rows, err := e.queryRaw(ctx, query)
			if err != nil {
				return nil, fmt.Errorf("failed to compute avg_content_length: %w", err)
			}

			if rows.Next() {
				if err := rows.Scan(&result.AvgContentLength); err != nil {
					rows.Close()
					return nil, err
				}
			}
			rows.Close()

		case "entity_type_distribution":
			query := fmt.Sprintf(`
				SELECT entity_type, COUNT(*) as count
				FROM udml_entities
				%s
				GROUP BY entity_type
			`, strings.Replace(typeFilter, "element_type", "e.element_type", -1))

			rows, err := e.queryRaw(ctx, query)
			if err != nil {
				// Entities table might not exist or be populated yet
				continue
			}

			for rows.Next() {
				var entityType string
				var count int
				if err := rows.Scan(&entityType, &count); err != nil {
					rows.Close()
					continue
				}
				result.EntityTypeDistribution[entityType] = count
			}
			rows.Close()
		}
	}

	return result, nil
}

// Helper functions

func joinQuoted(strs []string) string {
	quoted := make([]string, len(strs))
	for i, s := range strs {
		quoted[i] = "'" + s + "'"
	}
	return strings.Join(quoted, ", ")
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
