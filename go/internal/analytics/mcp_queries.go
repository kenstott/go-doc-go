package analytics

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	_ "github.com/marcboeker/go-duckdb"
)

// This file contains shared MCP query implementations that can be used by both
// ParquetStorage and HiveParquetStorage. Both storage backends use DuckDB for
// queries, so the implementation is identical except for the basePath.

// ============================================================================
// Shared implementation functions for MCP queries
// ============================================================================

// searchSemanticSimilarityImpl performs semantic similarity search
func searchSemanticSimilarityImpl(basePath string, queryVector []float64, filters map[string]interface{}, threshold float64, limit int) ([]SearchResult, error) {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		return nil, fmt.Errorf("failed to open DuckDB connection: %w", err)
	}
	defer db.Close()

	// Build temporal CTE if needed
	elementsQuery := buildElementsCTE(basePath, filters)

	// Build query with cosine similarity
	query := fmt.Sprintf(`
		WITH elements_filtered AS (%s),
		embeddings_joined AS (
			SELECT
				e.element_id, e.doc_id, e.source_name, e.element_type, e.element_category,
				e.content, e.content_preview, e.content_hash, e.parent_id,
				e.element_order, e.document_position, e.content_location,
				emb.embedding
			FROM elements_filtered e
			JOIN '%s/embeddings/**/*.parquet' emb ON e.element_id = emb.element_id
		)
		SELECT
			element_id, doc_id, source_name, element_type, element_category,
			content, content_preview, content_hash, parent_id,
			element_order, document_position, content_location,
			list_cosine_similarity(embedding, ?::DOUBLE[]) as similarity
		FROM embeddings_joined
		WHERE list_cosine_similarity(embedding, ?::DOUBLE[]) >= ?
		ORDER BY similarity DESC
		LIMIT ?
	`, elementsQuery, basePath)

	rows, err := db.Query(query, queryVector, queryVector, threshold, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to execute semantic similarity query: %w", err)
	}
	defer rows.Close()

	return scanSearchResults(rows, true)
}

// searchByRegexImpl performs regex pattern matching
func searchByRegexImpl(basePath string, pattern string, filters map[string]interface{}, limit int) ([]SearchResult, error) {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		return nil, fmt.Errorf("failed to open DuckDB connection: %w", err)
	}
	defer db.Close()

	elementsQuery := buildElementsCTE(basePath, filters)

	query := fmt.Sprintf(`
		WITH elements_filtered AS (%s)
		SELECT
			element_id, doc_id, source_name, element_type, element_category,
			content, content_preview, content_hash, parent_id,
			element_order, document_position, content_location,
			length(regexp_matches(content, ?)) as match_count
		FROM elements_filtered
		WHERE regexp_matches(content, ?) IS NOT NULL
		ORDER BY match_count DESC
		LIMIT ?
	`, elementsQuery)

	rows, err := db.Query(query, pattern, pattern, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to execute regex search: %w", err)
	}
	defer rows.Close()

	return scanSearchResults(rows, false)
}

// searchByKeywordImpl performs keyword search
func searchByKeywordImpl(basePath string, keyword string, filters map[string]interface{}, limit int) ([]SearchResult, error) {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		return nil, fmt.Errorf("failed to open DuckDB connection: %w", err)
	}
	defer db.Close()

	elementsQuery := buildElementsCTE(basePath, filters)

	query := fmt.Sprintf(`
		WITH elements_filtered AS (%s)
		SELECT
			element_id, doc_id, source_name, element_type, element_category,
			content, content_preview, content_hash, parent_id,
			element_order, document_position, content_location
		FROM elements_filtered
		WHERE lower(content) LIKE lower(?)
		LIMIT ?
	`, elementsQuery)

	rows, err := db.Query(query, "%"+keyword+"%", limit)
	if err != nil {
		return nil, fmt.Errorf("failed to execute keyword search: %w", err)
	}
	defer rows.Close()

	return scanSearchResults(rows, false)
}

// analyzePatternImpl analyzes a regex pattern
func analyzePatternImpl(basePath string, pattern string, filters map[string]interface{}, maxExamples int) (*PatternStats, error) {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		return nil, fmt.Errorf("failed to open DuckDB connection: %w", err)
	}
	defer db.Close()

	elementsQuery := buildElementsCTE(basePath, filters)

	// Get statistics
	statsQuery := fmt.Sprintf(`
		WITH elements_filtered AS (%s),
		matches AS (
			SELECT
				element_type,
				doc_id,
				length(regexp_matches(content, ?)) as match_count
			FROM elements_filtered
			WHERE regexp_matches(content, ?) IS NOT NULL
		)
		SELECT
			COUNT(*) as total_matches,
			COUNT(DISTINCT doc_id) as document_count
		FROM matches
	`, elementsQuery)

	var totalMatches, documentCount int
	err = db.QueryRow(statsQuery, pattern, pattern).Scan(&totalMatches, &documentCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get pattern statistics: %w", err)
	}

	// Get element type distribution
	distribQuery := fmt.Sprintf(`
		WITH elements_filtered AS (%s)
		SELECT
			element_type,
			COUNT(*) as count
		FROM elements_filtered
		WHERE regexp_matches(content, ?) IS NOT NULL
		GROUP BY element_type
	`, elementsQuery)

	distribRows, err := db.Query(distribQuery, pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to get element type distribution: %w", err)
	}
	defer distribRows.Close()

	distrib := make(map[string]int)
	for distribRows.Next() {
		var elementType string
		var count int
		if err := distribRows.Scan(&elementType, &count); err != nil {
			continue
		}
		distrib[elementType] = count
	}

	// Get examples
	examplesQuery := fmt.Sprintf(`
		WITH elements_filtered AS (%s)
		SELECT
			element_id, doc_id, source_name, element_type, element_category,
			content, content_preview, content_hash, parent_id,
			element_order, document_position, content_location
		FROM elements_filtered
		WHERE regexp_matches(content, ?) IS NOT NULL
		LIMIT ?
	`, elementsQuery)

	exampleRows, err := db.Query(examplesQuery, pattern, maxExamples)
	if err != nil {
		return nil, fmt.Errorf("failed to get pattern examples: %w", err)
	}
	defer exampleRows.Close()

	examples := scanElements(exampleRows)

	return &PatternStats{
		Pattern:            pattern,
		TotalMatches:       totalMatches,
		DocumentCount:      documentCount,
		ElementTypeDistrib: distrib,
		Examples:           examples,
	}, nil
}

// computeTermFrequenciesImpl computes term frequency statistics
func computeTermFrequenciesImpl(basePath string, terms []string, caseSensitive bool, filters map[string]interface{}) ([]TermFrequency, error) {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		return nil, fmt.Errorf("failed to open DuckDB connection: %w", err)
	}
	defer db.Close()

	elementsQuery := buildElementsCTE(basePath, filters)

	var results []TermFrequency
	for _, term := range terms {
		var matchPattern string
		if caseSensitive {
			matchPattern = "%" + term + "%"
		} else {
			matchPattern = "%" + strings.ToLower(term) + "%"
		}

		var query string
		if caseSensitive {
			query = fmt.Sprintf(`
				WITH elements_filtered AS (%s)
				SELECT
					COUNT(*) as frequency,
					COUNT(DISTINCT doc_id) as document_count
				FROM elements_filtered
				WHERE content LIKE ?
			`, elementsQuery)
		} else {
			query = fmt.Sprintf(`
				WITH elements_filtered AS (%s)
				SELECT
					COUNT(*) as frequency,
					COUNT(DISTINCT doc_id) as document_count
				FROM elements_filtered
				WHERE lower(content) LIKE ?
			`, elementsQuery)
		}

		var frequency, documentCount int
		err = db.QueryRow(query, matchPattern).Scan(&frequency, &documentCount)
		if err != nil {
			log.Printf("Failed to compute frequency for term '%s': %v", term, err)
			continue
		}

		// Get element type distribution
		var distribQuery string
		if caseSensitive {
			distribQuery = fmt.Sprintf(`
				WITH elements_filtered AS (%s)
				SELECT element_type, COUNT(*) as count
				FROM elements_filtered
				WHERE content LIKE ?
				GROUP BY element_type
			`, elementsQuery)
		} else {
			distribQuery = fmt.Sprintf(`
				WITH elements_filtered AS (%s)
				SELECT element_type, COUNT(*) as count
				FROM elements_filtered
				WHERE lower(content) LIKE ?
				GROUP BY element_type
			`, elementsQuery)
		}

		distribRows, err := db.Query(distribQuery, matchPattern)
		if err != nil {
			log.Printf("Failed to get element type distribution for term '%s': %v", term, err)
			continue
		}

		distrib := make(map[string]int)
		for distribRows.Next() {
			var elementType string
			var count int
			if err := distribRows.Scan(&elementType, &count); err != nil {
				continue
			}
			distrib[elementType] = count
		}
		distribRows.Close()

		results = append(results, TermFrequency{
			Term:               term,
			Frequency:          frequency,
			DocumentCount:      documentCount,
			ElementTypeDistrib: distrib,
		})
	}

	return results, nil
}

// findCooccurrencesImpl finds co-occurrences of entities
func findCooccurrencesImpl(basePath string, entity1, entity2 string, contextWindow string, filters map[string]interface{}, maxExamples int) (*CooccurrenceResult, error) {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		return nil, fmt.Errorf("failed to open DuckDB connection: %w", err)
	}
	defer db.Close()

	elementsQuery := buildElementsCTE(basePath, filters)

	var query string
	switch contextWindow {
	case "element":
		query = fmt.Sprintf(`
			WITH elements_filtered AS (%s)
			SELECT
				element_id, doc_id, source_name, element_type, element_category,
				content, content_preview, content_hash, parent_id,
				element_order, document_position, content_location
			FROM elements_filtered
			WHERE lower(content) LIKE lower(?)
			  AND lower(content) LIKE lower(?)
			LIMIT ?
		`, elementsQuery)
	case "document":
		query = fmt.Sprintf(`
			WITH elements_filtered AS (%s),
			docs_with_both AS (
				SELECT DISTINCT doc_id
				FROM elements_filtered
				WHERE lower(content) LIKE lower(?)
				INTERSECT
				SELECT DISTINCT doc_id
				FROM elements_filtered
				WHERE lower(content) LIKE lower(?)
			)
			SELECT
				e.element_id, e.doc_id, e.source_name, e.element_type, e.element_category,
				e.content, e.content_preview, e.content_hash, e.parent_id,
				e.element_order, e.document_position, e.content_location
			FROM elements_filtered e
			JOIN docs_with_both d ON e.doc_id = d.doc_id
			WHERE lower(e.content) LIKE lower(?)
			   OR lower(e.content) LIKE lower(?)
			LIMIT ?
		`, elementsQuery)
	default:
		return nil, fmt.Errorf("unsupported context window: %s", contextWindow)
	}

	var rows *sql.Rows
	if contextWindow == "element" {
		rows, err = db.Query(query, "%"+entity1+"%", "%"+entity2+"%", maxExamples)
	} else {
		rows, err = db.Query(query, "%"+entity1+"%", "%"+entity2+"%", "%"+entity1+"%", "%"+entity2+"%", maxExamples)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to find co-occurrences: %w", err)
	}
	defer rows.Close()

	examples := scanElements(rows)

	return &CooccurrenceResult{
		Entity1:       entity1,
		Entity2:       entity2,
		CooccurCount:  len(examples),
		ContextWindow: contextWindow,
		Examples:      examples,
	}, nil
}

// getElementContextImpl retrieves element with hierarchical context
func getElementContextImpl(basePath string, elementID string, filters map[string]interface{}, contextDepth int, includeSiblings, includeChildren bool) (*ElementContext, error) {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		return nil, fmt.Errorf("failed to open DuckDB connection: %w", err)
	}
	defer db.Close()

	elementsQuery := buildElementsCTE(basePath, filters)

	// Get the target element
	targetQuery := fmt.Sprintf(`
		WITH elements_filtered AS (%s)
		SELECT
			element_id, doc_id, source_name, element_type, element_category,
			content, content_preview, content_hash, parent_id,
			element_order, document_position, content_location
		FROM elements_filtered
		WHERE element_id = ?
	`, elementsQuery)

	var targetElement Element
	var contentLocationJSON sql.NullString
	err = db.QueryRow(targetQuery, elementID).Scan(
		&targetElement.ElementID, &targetElement.DocID, &targetElement.SourceName,
		&targetElement.ElementType, &targetElement.ElementCategory,
		&targetElement.Content, &targetElement.ContentPreview, &targetElement.ContentHash,
		&targetElement.ParentID, &targetElement.ElementOrder, &targetElement.DocumentPosition,
		&contentLocationJSON,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to find element: %w", err)
	}

	if contentLocationJSON.Valid && contentLocationJSON.String != "" {
		if err := json.Unmarshal([]byte(contentLocationJSON.String), &targetElement.ContentLocation); err != nil {
			log.Printf("Failed to unmarshal content_location: %v", err)
		}
	}

	context := &ElementContext{Element: targetElement}

	// Get parents (recursive up to contextDepth)
	if targetElement.ParentID != "" && contextDepth > 0 {
		parents, err := getParentChain(db, basePath, elementsQuery, targetElement.ParentID, contextDepth)
		if err == nil {
			context.Parents = parents
		}
	}

	// Get siblings if requested
	if includeSiblings && targetElement.ParentID != "" {
		siblingsQuery := fmt.Sprintf(`
			WITH elements_filtered AS (%s)
			SELECT
				element_id, doc_id, source_name, element_type, element_category,
				content, content_preview, content_hash, parent_id,
				element_order, document_position, content_location
			FROM elements_filtered
			WHERE parent_id = ? AND element_id != ?
			ORDER BY element_order
		`, elementsQuery)

		siblingsRows, err := db.Query(siblingsQuery, targetElement.ParentID, elementID)
		if err == nil {
			defer siblingsRows.Close()
			context.Siblings = scanElements(siblingsRows)
		}
	}

	// Get children if requested
	if includeChildren {
		childrenQuery := fmt.Sprintf(`
			WITH elements_filtered AS (%s)
			SELECT
				element_id, doc_id, source_name, element_type, element_category,
				content, content_preview, content_hash, parent_id,
				element_order, document_position, content_location
			FROM elements_filtered
			WHERE parent_id = ?
			ORDER BY element_order
		`, elementsQuery)

		childrenRows, err := db.Query(childrenQuery, elementID)
		if err == nil {
			defer childrenRows.Close()
			context.Children = scanElements(childrenRows)
		}
	}

	return context, nil
}

// aggregateStatisticsImpl computes aggregate statistics
func aggregateStatisticsImpl(basePath string, metrics []string, filters map[string]interface{}) (*CorpusStats, error) {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		return nil, fmt.Errorf("failed to open DuckDB connection: %w", err)
	}
	defer db.Close()

	elementsQuery := buildElementsCTE(basePath, filters)

	stats := &CorpusStats{}

	for _, metric := range metrics {
		switch metric {
		case "element_type_distribution":
			query := fmt.Sprintf(`
				WITH elements_filtered AS (%s)
				SELECT element_type, COUNT(*) as count
				FROM elements_filtered
				GROUP BY element_type
			`, elementsQuery)

			rows, err := db.Query(query)
			if err != nil {
				log.Printf("Failed to compute element_type_distribution: %v", err)
				continue
			}

			distrib := make(map[string]int)
			for rows.Next() {
				var elementType string
				var count int
				if err := rows.Scan(&elementType, &count); err != nil {
					continue
				}
				distrib[elementType] = count
			}
			rows.Close()
			stats.ElementTypeDistribution = distrib

		case "document_count":
			query := fmt.Sprintf(`
				WITH elements_filtered AS (%s)
				SELECT COUNT(DISTINCT doc_id) FROM elements_filtered
			`, elementsQuery)

			var count int
			if err := db.QueryRow(query).Scan(&count); err != nil {
				log.Printf("Failed to compute document_count: %v", err)
			} else {
				stats.DocumentCount = count
			}

		case "total_elements":
			query := fmt.Sprintf(`
				WITH elements_filtered AS (%s)
				SELECT COUNT(*) FROM elements_filtered
			`, elementsQuery)

			var count int
			if err := db.QueryRow(query).Scan(&count); err != nil {
				log.Printf("Failed to compute total_elements: %v", err)
			} else {
				stats.TotalElements = count
			}

		case "avg_content_length":
			query := fmt.Sprintf(`
				WITH elements_filtered AS (%s)
				SELECT AVG(length(content)) FROM elements_filtered WHERE content IS NOT NULL
			`, elementsQuery)

			var avgLength float64
			if err := db.QueryRow(query).Scan(&avgLength); err != nil {
				log.Printf("Failed to compute avg_content_length: %v", err)
			} else {
				stats.AvgContentLength = avgLength
			}
		}
	}

	return stats, nil
}

// ============================================================================
// Helper functions
// ============================================================================

// buildElementsCTE builds the elements CTE with temporal filtering
func buildElementsCTE(basePath string, filters map[string]interface{}) string {
	// Extract temporal filter parameters
	latestOnly := false
	if val, ok := filters["latest_only"].(bool); ok {
		latestOnly = val
	}
	asOfDate := ""
	if val, ok := filters["as_of_date"].(string); ok {
		asOfDate = val
	}

	if latestOnly {
		// Use window function to deduplicate by doc_id
		query := fmt.Sprintf(`
			SELECT
				element_id, doc_id, source_name, element_type, element_category,
				content, content_preview, content_hash, parent_id,
				element_order, document_position, content_location
			FROM (
				SELECT
					element_id, doc_id, source_name, element_type, element_category,
					content, content_preview, content_hash, parent_id,
					element_order, document_position, content_location,
					ROW_NUMBER() OVER (
						PARTITION BY doc_id
						ORDER BY regexp_extract(filename, 'date=(\d{4}-\d{2}-\d{2})', 1) DESC
					) as row_num
				FROM read_parquet('%s/elements/**/*.parquet', filename=true)
				WHERE 1=1`, basePath)

		if asOfDate != "" {
			query += fmt.Sprintf(" AND regexp_extract(filename, 'date=(\\d{4}-\\d{2}-\\d{2})', 1) <= '%s'", asOfDate)
		}

		// Add standard filters
		if source, ok := filters["source_name"].(string); ok && source != "" {
			query += fmt.Sprintf(" AND source_name = '%s'", source)
		}
		if docID, ok := filters["doc_id"].(string); ok && docID != "" {
			query += fmt.Sprintf(" AND doc_id = '%s'", docID)
		}
		if elementType, ok := filters["element_type"].(string); ok && elementType != "" {
			query += fmt.Sprintf(" AND element_type = '%s'", elementType)
		}
		if elementCategory, ok := filters["element_category"].(string); ok && elementCategory != "" {
			query += fmt.Sprintf(" AND element_category = '%s'", elementCategory)
		}

		query += "\n) WHERE row_num = 1"
		return query
	}

	// Standard query without deduplication
	query := fmt.Sprintf(`
		SELECT
			element_id, doc_id, source_name, element_type, element_category,
			content, content_preview, content_hash, parent_id,
			element_order, document_position, content_location
		FROM '%s/elements/**/*.parquet'
		WHERE 1=1`, basePath)

	// Add standard filters
	if source, ok := filters["source_name"].(string); ok && source != "" {
		query += fmt.Sprintf(" AND source_name = '%s'", source)
	}
	if docID, ok := filters["doc_id"].(string); ok && docID != "" {
		query += fmt.Sprintf(" AND doc_id = '%s'", docID)
	}
	if elementType, ok := filters["element_type"].(string); ok && elementType != "" {
		query += fmt.Sprintf(" AND element_type = '%s'", elementType)
	}
	if elementCategory, ok := filters["element_category"].(string); ok && elementCategory != "" {
		query += fmt.Sprintf(" AND element_category = '%s'", elementCategory)
	}

	return query
}

// scanSearchResults scans search results with optional score/match count
func scanSearchResults(rows *sql.Rows, withScore bool) ([]SearchResult, error) {
	var results []SearchResult
	for rows.Next() {
		var elem Element
		var contentLocationJSON sql.NullString
		var scoreOrMatchCount interface{}

		var scanArgs []interface{}
		if withScore {
			var score float64
			scoreOrMatchCount = &score
			scanArgs = []interface{}{
				&elem.ElementID, &elem.DocID, &elem.SourceName, &elem.ElementType, &elem.ElementCategory,
				&elem.Content, &elem.ContentPreview, &elem.ContentHash, &elem.ParentID,
				&elem.ElementOrder, &elem.DocumentPosition, &contentLocationJSON,
				scoreOrMatchCount,
			}
		} else {
			// Check if we have a match_count column (regex search)
			cols, _ := rows.Columns()
			if len(cols) > 12 {
				var matchCount int
				scoreOrMatchCount = &matchCount
				scanArgs = []interface{}{
					&elem.ElementID, &elem.DocID, &elem.SourceName, &elem.ElementType, &elem.ElementCategory,
					&elem.Content, &elem.ContentPreview, &elem.ContentHash, &elem.ParentID,
					&elem.ElementOrder, &elem.DocumentPosition, &contentLocationJSON,
					scoreOrMatchCount,
				}
			} else {
				scanArgs = []interface{}{
					&elem.ElementID, &elem.DocID, &elem.SourceName, &elem.ElementType, &elem.ElementCategory,
					&elem.Content, &elem.ContentPreview, &elem.ContentHash, &elem.ParentID,
					&elem.ElementOrder, &elem.DocumentPosition, &contentLocationJSON,
				}
			}
		}

		if err := rows.Scan(scanArgs...); err != nil {
			log.Printf("Failed to scan search result: %v", err)
			continue
		}

		if contentLocationJSON.Valid && contentLocationJSON.String != "" {
			if err := json.Unmarshal([]byte(contentLocationJSON.String), &elem.ContentLocation); err != nil {
				log.Printf("Failed to unmarshal content_location: %v", err)
			}
		}

		result := SearchResult{Element: elem}
		if withScore {
			result.Score = *scoreOrMatchCount.(*float64)
		} else if scoreOrMatchCount != nil {
			result.MatchCount = *scoreOrMatchCount.(*int)
		}

		results = append(results, result)
	}

	return results, nil
}

// scanElements is a helper to scan element rows
func scanElements(rows *sql.Rows) []Element {
	var elements []Element
	for rows.Next() {
		var elem Element
		var contentLocationJSON sql.NullString

		err := rows.Scan(
			&elem.ElementID, &elem.DocID, &elem.SourceName,
			&elem.ElementType, &elem.ElementCategory,
			&elem.Content, &elem.ContentPreview, &elem.ContentHash,
			&elem.ParentID, &elem.ElementOrder, &elem.DocumentPosition,
			&contentLocationJSON,
		)
		if err != nil {
			log.Printf("Failed to scan element: %v", err)
			continue
		}

		if contentLocationJSON.Valid && contentLocationJSON.String != "" {
			if err := json.Unmarshal([]byte(contentLocationJSON.String), &elem.ContentLocation); err != nil {
				log.Printf("Failed to unmarshal content_location: %v", err)
			}
		}

		elements = append(elements, elem)
	}
	return elements
}

// getParentChain recursively retrieves parent elements
func getParentChain(db *sql.DB, basePath string, elementsQuery string, parentID string, maxDepth int) ([]Element, error) {
	var parents []Element
	currentParentID := parentID

	for i := 0; i < maxDepth && currentParentID != ""; i++ {
		query := fmt.Sprintf(`
			WITH elements_filtered AS (%s)
			SELECT
				element_id, doc_id, source_name, element_type, element_category,
				content, content_preview, content_hash, parent_id,
				element_order, document_position, content_location
			FROM elements_filtered
			WHERE element_id = ?
		`, elementsQuery)

		var parent Element
		var contentLocationJSON sql.NullString
		err := db.QueryRow(query, currentParentID).Scan(
			&parent.ElementID, &parent.DocID, &parent.SourceName,
			&parent.ElementType, &parent.ElementCategory,
			&parent.Content, &parent.ContentPreview, &parent.ContentHash,
			&parent.ParentID, &parent.ElementOrder, &parent.DocumentPosition,
			&contentLocationJSON,
		)
		if err != nil {
			break
		}

		if contentLocationJSON.Valid && contentLocationJSON.String != "" {
			if err := json.Unmarshal([]byte(contentLocationJSON.String), &parent.ContentLocation); err != nil {
				log.Printf("Failed to unmarshal content_location: %v", err)
			}
		}

		parents = append(parents, parent)
		currentParentID = parent.ParentID
	}

	return parents, nil
}
