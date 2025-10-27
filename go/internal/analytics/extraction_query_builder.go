package analytics

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ExtractionQueryBuilder builds SQL queries for ontology extraction in DuckDB
// Translates ontology schema rules into parameterized SQL queries
type ExtractionQueryBuilder struct {
	basePath string // Path to Parquet files
}

// NewExtractionQueryBuilder creates a new query builder
func NewExtractionQueryBuilder(basePath string) *ExtractionQueryBuilder {
	return &ExtractionQueryBuilder{
		basePath: basePath,
	}
}

// EntityExtractionQuery represents a compiled SQL query for entity extraction
type EntityExtractionQuery struct {
	SQL        string                 // Parameterized SQL query
	Parameters map[string]interface{} // Query parameters
}

// BuildEntityExtractionQuery builds a SQL query for extracting entities from a single ElementEntityMapping
//
// Strategy:
// 1. Filter to leaf elements (elements with embeddings)
// 2. Apply element_type filter
// 3. Apply extraction_rules (OR logic) - keyword, regex, text_similarity
// 4. Apply semantic_filter (AND logic) - validates matches against reference concepts
// 5. Return matched entities with metadata
func (b *ExtractionQueryBuilder) BuildEntityExtractionQuery(
	mappingJSON []byte, // Single ElementEntityMapping as JSON
	filters map[string]interface{}, // Query filters
	conceptEmbeddings map[string][]float64, // Pre-computed embeddings for semantic filtering
) (*EntityExtractionQuery, error) {

	// Parse ElementEntityMapping from JSON
	var mapping map[string]interface{}
	if err := json.Unmarshal(mappingJSON, &mapping); err != nil {
		return nil, fmt.Errorf("failed to parse entity mapping: %w", err)
	}

	entityType := mapping["entity_type"].(string)
	domain := mapping["domain"].(string)
	confidence := mapping["confidence"].(float64)
	extractionRules := mapping["extraction_rules"].([]interface{})
	// NOTE: element_types field is ignored - we evaluate against ALL leaf elements with embeddings

	// Build query parts
	params := make(map[string]interface{})
	params["entity_type"] = entityType
	params["domain"] = domain
	params["confidence"] = confidence

	// Build WHERE clauses for extraction rules (OR logic)
	// Filters are applied in cost order: pattern → proximity → instance_name → semantic
	var ruleClauses []string
	var extractionExprs []string           // Expressions to extract entity name

	for i, rule := range extractionRules {
		ruleMap := rule.(map[string]interface{})
		ruleType := ruleMap["type"].(string)

		var filterConditions []string  // AND-ed filter conditions
		var extractExpr string

		switch ruleType {
		case "content_extraction":
			// UNIFIED CONTENT EXTRACTION with optimized filter ordering
			// Apply filters in order of cost: cheapest first, semantic last

			// 1. Pattern filter (cheapest - regex on content)
			if pattern, ok := ruleMap["pattern"].(string); ok && pattern != "" {
				filterConditions = append(filterConditions, fmt.Sprintf("regexp_matches(content, '%s')", escapeSQL(pattern)))
			}

			// 2. Proximity filter (moderate cost - co-occurrence checking)
			if proximityFilter, ok := ruleMap["proximity_filter"].(map[string]interface{}); ok && proximityFilter != nil {
				proximityClause := b.buildProximityFilterClause(proximityFilter, i, params)
				if proximityClause != "" {
					filterConditions = append(filterConditions, proximityClause)
				}
			}

			// 3. Instance name extraction (REQUIRED - must succeed for entity to be included)
			instanceName, ok := ruleMap["instance_name"].(string)
			if !ok || instanceName == "" {
				// Skip rule if instance_name missing
				continue
			}
			// Check that instance_name regex can extract successfully (not NULL)
			filterConditions = append(filterConditions, fmt.Sprintf("regexp_extract(content, '%s', 1) IS NOT NULL", escapeSQL(instanceName)))
			extractExpr = fmt.Sprintf("regexp_extract(content, '%s', 1)", escapeSQL(instanceName))

			// 4. Semantic filter (most expensive - embedding similarity, applied LAST)
			if semanticFilter, ok := ruleMap["semantic_filter"].(map[string]interface{}); ok && semanticFilter != nil {
				semanticClause := b.buildSemanticFilterClauseInline(semanticFilter, i, params, conceptEmbeddings)
				if semanticClause != "" {
					filterConditions = append(filterConditions, semanticClause)
				}
			}

			// Combine all filters with AND
			if len(filterConditions) > 0 {
				ruleClauses = append(ruleClauses, "("+strings.Join(filterConditions, " AND ")+")")
				if extractExpr != "" {
					extractionExprs = append(extractionExprs, extractExpr)
				}
			}

		case "metadata_field":
			baseClause := b.buildMetadataClause(ruleMap, i, params)
			ruleClauses = append(ruleClauses, baseClause)

		case "jsonpath_query":
			baseClause := b.buildJSONPathClause(ruleMap, i, params)
			ruleClauses = append(ruleClauses, baseClause)
		}
	}

	// Combine extraction rules with OR
	extractionRulesClause := strings.Join(ruleClauses, " OR ")

	// Build entity_name expression
	// Try each extraction expression in order, fall back to content if none match
	// Strip newlines and extra whitespace from entity names
	entityNameExpr := "TRIM(REPLACE(REPLACE(content, CHR(10), ' '), CHR(13), ' '))"
	if len(extractionExprs) > 0 {
		// COALESCE tries each extraction in order, falls back to content if all NULL
		// Wrap in REPLACE to strip newlines (CHR(10) = \n, CHR(13) = \r)
		entityNameExpr = fmt.Sprintf("TRIM(REPLACE(REPLACE(COALESCE(%s, content), CHR(10), ' '), CHR(13), ' '))",
			strings.Join(extractionExprs, ", "))
	}

	// Build final SQL query
	// Semantic filters are now embedded within extraction rules, so no need for separate CTE
	// NOTE: No element_type filter - evaluate against ALL leaf elements with embeddings
	query := fmt.Sprintf(`
WITH
  -- Step 1: Join elements with embeddings to get text content and embedding vector
  -- ALL leaf elements are candidates (no element_type filter)
  leaf_elements AS (
    SELECT
      e.element_id,
      e.doc_id,
      e.source_name,
      e.element_type,
      emb.text as content,
      emb.embedding as embedding,
      e.content_preview,
      e.metadata,
      e.content_location
    FROM '%s/elements/**/*.parquet' e
    INNER JOIN '%s/embeddings/**/*.parquet' emb ON e.element_id = emb.element_id
    %s
  ),

  -- Step 2: Apply extraction rules with embedded semantic filters (OR logic)
  matched_elements AS (
    SELECT
      element_id,
      doc_id,
      source_name,
      element_type,
      content,
      content_preview
    FROM leaf_elements
    WHERE (%s)
  )

SELECT
  element_id,
  doc_id,
  source_name,
  '%s' as entity_type,
  '%s' as domain,
  %s as entity_name,
  CAST(%.2f AS DOUBLE) as confidence
FROM matched_elements
%s;
`,
		b.basePath,
		b.basePath,
		b.buildDocIDFilter(filters),
		extractionRulesClause,
		entityType,
		domain,
		entityNameExpr,
		confidence,
		b.buildLimitClause(filters),
	)

	return &EntityExtractionQuery{
		SQL:        query,
		Parameters: params,
	}, nil
}

// buildKeywordMatchClause builds SQL for keyword matching
// Returns: (whereClause, extractionExpr)
// - whereClause: SQL WHERE condition to match keywords
// - extractionExpr: SQL expression to extract the matched keyword
func (b *ExtractionQueryBuilder) buildKeywordMatchClause(rule map[string]interface{}, idx int, params map[string]interface{}) (string, string) {
	keywords := rule["keywords"].([]interface{})
	var conditions []string
	var caseConditions []string

	for i, kw := range keywords {
		paramName := fmt.Sprintf("keyword_%d_%d", idx, i)
		keyword := kw.(string)
		params[paramName] = keyword

		// WHERE condition for matching
		conditions = append(conditions, fmt.Sprintf("content LIKE '%%%s%%'", escapeSQL(keyword)))

		// CASE WHEN for extraction - returns the keyword that matched
		caseConditions = append(caseConditions,
			fmt.Sprintf("WHEN content LIKE '%%%s%%' THEN '%s'", escapeSQL(keyword), escapeSQL(keyword)))
	}

	whereClause := "(" + strings.Join(conditions, " OR ") + ")"

	// Build CASE expression to extract the first matching keyword
	extractionExpr := ""
	if len(caseConditions) > 0 {
		extractionExpr = fmt.Sprintf("CASE %s END", strings.Join(caseConditions, " "))
	}

	return whereClause, extractionExpr
}

// buildRegexClause builds SQL for regex pattern matching
func (b *ExtractionQueryBuilder) buildRegexClause(rule map[string]interface{}, idx int, params map[string]interface{}) string {
	pattern := rule["pattern"].(string)
	paramName := fmt.Sprintf("regex_%d", idx)
	params[paramName] = pattern
	// DuckDB regex function
	return fmt.Sprintf("regexp_matches(content, '%s')", escapeSQL(pattern))
}

// buildTextSimilarityClause builds SQL for semantic similarity using embeddings
// Returns both WHERE clause and extraction expression
func (b *ExtractionQueryBuilder) buildTextSimilarityClause(
	rule map[string]interface{},
	idx int,
	params map[string]interface{},
	conceptEmbeddings map[string][]float64,
) (string, string) {
	referenceText := rule["reference_text"].(string)
	threshold := rule["similarity_threshold"].(float64)

	// Get pre-computed embedding for reference text
	refEmbedding, ok := conceptEmbeddings[referenceText]
	if !ok {
		// If embedding not provided, skip this rule (can't execute without embedding)
		return "FALSE", ""
	}

	// Convert embedding to SQL array literal: [val1, val2, ...]
	embeddingSQL := embeddingArrayToSQL(refEmbedding)

	// Build WHERE clause subquery
	whereClause := fmt.Sprintf(`
		element_id IN (
			SELECT emb.element_id
			FROM '%s/embeddings/**/*.parquet' emb
			WHERE list_cosine_similarity(emb.embedding, %s::DOUBLE[]) > %.2f
		)
	`, b.basePath, embeddingSQL, threshold)

	// Build extraction expression - instance_name is required for text_similarity
	instanceName, ok := rule["instance_name"].(string)
	if !ok || instanceName == "" {
		// Missing required instance_name - skip this rule
		return "FALSE", ""
	}

	// Use regex extraction from instance_name field
	extractExpr := fmt.Sprintf("regexp_extract(content, '%s', 1)", escapeSQL(instanceName))

	return whereClause, extractExpr
}

// buildMetadataClause builds SQL for metadata field extraction
func (b *ExtractionQueryBuilder) buildMetadataClause(rule map[string]interface{}, idx int, params map[string]interface{}) string {
	fieldPath := rule["field_path"].(string)
	paramName := fmt.Sprintf("field_path_%d", idx)
	params[paramName] = fieldPath

	// Use DuckDB's json_extract_string for metadata querying
	return fmt.Sprintf("json_extract_string(metadata, '$.%s') IS NOT NULL", fieldPath)
}

// buildJSONPathClause builds SQL for JSONPath queries
// Reuses logic from internal/udml/query/duckdb.go:buildJSONPathPredicate
func (b *ExtractionQueryBuilder) buildJSONPathClause(rule map[string]interface{}, idx int, params map[string]interface{}) string {
	jsonPath := rule["jsonpath_expr"].(string)
	paramName := fmt.Sprintf("jsonpath_%d", idx)
	params[paramName] = jsonPath

	// DuckDB JSON access: json_extract(column, '$.path')
	jsonExpr := fmt.Sprintf("json_extract(metadata, '%s')", jsonPath)

	// Check if there's a comparison operator and value
	operator, hasOperator := rule["operator"].(string)
	value, hasValue := rule["value"]

	if !hasOperator || !hasValue {
		// No operator/value - just check if path exists and is not null
		return fmt.Sprintf("%s IS NOT NULL", jsonExpr)
	}

	// Format value for SQL
	formattedValue := formatValue(value)

	// Build comparison clause based on operator
	switch operator {
	case "=", "==", "equal":
		return fmt.Sprintf("%s = %s", jsonExpr, formattedValue)
	case "!=", "not_equal":
		return fmt.Sprintf("%s != %s", jsonExpr, formattedValue)
	case ">", "greater_than":
		return fmt.Sprintf("%s > %s", jsonExpr, formattedValue)
	case ">=", "greater_than_or_equal":
		return fmt.Sprintf("%s >= %s", jsonExpr, formattedValue)
	case "<", "less_than":
		return fmt.Sprintf("%s < %s", jsonExpr, formattedValue)
	case "<=", "less_than_or_equal":
		return fmt.Sprintf("%s <= %s", jsonExpr, formattedValue)
	default:
		// Unknown operator - just check if path exists
		return fmt.Sprintf("%s IS NOT NULL", jsonExpr)
	}
}

// formatValue formats a value for SQL (reused from duckdb.go)
func formatValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		return fmt.Sprintf("'%s'", strings.ReplaceAll(v, "'", "''"))
	case int, int64, float64:
		return fmt.Sprintf("%v", v)
	case bool:
		return fmt.Sprintf("%t", v)
	default:
		return fmt.Sprintf("'%v'", v)
	}
}

// buildSemanticFilterClauseInline builds an inline semantic filter for use within extraction rules
// Uses the embedding column from leaf_elements CTE
// Embeds the embedding vectors directly as SQL arrays
func (b *ExtractionQueryBuilder) buildSemanticFilterClauseInline(
	semanticFilter map[string]interface{},
	ruleIndex int,
	params map[string]interface{},
	conceptEmbeddings map[string][]float64,
) string {
	threshold := semanticFilter["similarity_threshold"].(float64)

	// Collect all reference texts from both possible formats
	var referenceTexts []string

	// Handle reference_text (single string)
	if refText, ok := semanticFilter["reference_text"].(string); ok && refText != "" {
		referenceTexts = append(referenceTexts, refText)
	}

	// Handle reference_concepts (array of strings)
	if refConcepts, ok := semanticFilter["reference_concepts"].([]interface{}); ok {
		for _, concept := range refConcepts {
			if conceptText, ok := concept.(string); ok {
				referenceTexts = append(referenceTexts, conceptText)
			}
		}
	}

	// Build SQL conditions for each reference text
	var conditions []string
	for _, conceptText := range referenceTexts {
		embedding, ok := conceptEmbeddings[conceptText]
		if !ok {
			continue // Skip if embedding not provided
		}

		// Convert embedding to SQL array literal
		// Format: [0.1, 0.2, 0.3, ...]
		embeddingStrs := make([]string, len(embedding))
		for i, val := range embedding {
			embeddingStrs[i] = fmt.Sprintf("%.6f", val)
		}
		embeddingSQL := fmt.Sprintf("[%s]", strings.Join(embeddingStrs, ", "))

		// Reference the embedding column from leaf_elements CTE
		// embedding column is already DOUBLE[] (list<element: double> in Parquet)
		// Literal array syntax: [val1, val2, ...]
		conditions = append(conditions, fmt.Sprintf(
			"list_cosine_similarity(embedding, %s::DOUBLE[]) > %.2f",
			embeddingSQL, threshold,
		))
	}

	if len(conditions) == 0 {
		return ""
	}

	// OR logic: element must match at least one reference concept
	return strings.Join(conditions, " OR ")
}

// buildSemanticFilterCTE builds the CTE for semantic filter validation
func (b *ExtractionQueryBuilder) buildSemanticFilterCTE(semanticFilterClause string) string {
	if semanticFilterClause == "" {
		return ""
	}

	return fmt.Sprintf(`,
  -- Step 3: Apply semantic filter (AND condition - validates matches)
  semantic_validated AS (
    SELECT me.*
    FROM matched_elements me
    JOIN '%s/embeddings/**/*.parquet' emb
      ON me.element_id = emb.element_id
    WHERE (%s)
  )`, b.basePath, semanticFilterClause)
}

// getFinalTable returns the table name to select from (with or without semantic filter)
func (b *ExtractionQueryBuilder) getFinalTable(semanticFilterClause string) string {
	if semanticFilterClause != "" {
		return "semantic_validated"
	}
	return "matched_elements"
}

// buildDocIDFilter builds optional doc_id filter
// Handles both single doc_id (string) and batch of doc_ids ([]string)
func (b *ExtractionQueryBuilder) buildDocIDFilter(filters map[string]interface{}) string {
	// Single doc_id
	if docID, ok := filters["doc_id"].(string); ok && docID != "" {
		return fmt.Sprintf("AND e.doc_id = '%s'", escapeSQL(docID))
	}

	// Batch of doc_ids
	if docIDs, ok := filters["doc_ids"].([]string); ok && len(docIDs) > 0 {
		quotedIDs := make([]string, len(docIDs))
		for i, id := range docIDs {
			quotedIDs[i] = fmt.Sprintf("'%s'", escapeSQL(id))
		}
		return fmt.Sprintf("AND e.doc_id IN (%s)", strings.Join(quotedIDs, ", "))
	}

	return ""
}

// buildLimitClause builds optional LIMIT for early stopping
func (b *ExtractionQueryBuilder) buildLimitClause(filters map[string]interface{}) string {
	if maxEntities, ok := filters["max_entities"].(int); ok && maxEntities > 0 {
		return fmt.Sprintf("LIMIT %d", maxEntities)
	}
	return ""
}

// embeddingArrayToSQL converts a float64 slice to SQL array literal format
func embeddingArrayToSQL(embedding []float64) string {
	embeddingStrs := make([]string, len(embedding))
	for i, val := range embedding {
		embeddingStrs[i] = fmt.Sprintf("%.6f", val)
	}
	return fmt.Sprintf("[%s]", strings.Join(embeddingStrs, ", "))
}

// buildProximityFilterClause builds SQL for proximity/co-occurrence filtering
// Returns WHERE clause that checks if entity appears near specified terms
func (b *ExtractionQueryBuilder) buildProximityFilterClause(
	proximityFilter map[string]interface{},
	idx int,
	params map[string]interface{},
) string {
	cooccurrenceTerms, ok := proximityFilter["cooccurrence_terms"].([]interface{})
	if !ok || len(cooccurrenceTerms) == 0 {
		return ""
	}

	// Get max_distance (default: 0 = same element)
	maxDistance := 0
	if dist, ok := proximityFilter["max_distance"].(float64); ok {
		maxDistance = int(dist)
	}

	// Get distance_unit (default: "element")
	distanceUnit := "element"
	if unit, ok := proximityFilter["distance_unit"].(string); ok {
		distanceUnit = unit
	}

	// Build conditions for co-occurrence terms
	var termConditions []string
	for i, term := range cooccurrenceTerms {
		termStr := term.(string)
		paramName := fmt.Sprintf("proximity_term_%d_%d", idx, i)
		params[paramName] = termStr

		if distanceUnit == "element" || maxDistance == 0 {
			// Simple co-occurrence: terms must appear somewhere in the same element
			termConditions = append(termConditions, fmt.Sprintf("content LIKE '%%%s%%'", escapeSQL(termStr)))
		} else if distanceUnit == "word" {
			// Word-based distance: use regex with word boundaries
			// This is a simplified implementation - DuckDB has limited regex distance support
			// For now, just check if terms appear in content (full word-distance requires UDF)
			termConditions = append(termConditions, fmt.Sprintf("regexp_matches(content, '\\b%s\\b')", escapeSQL(termStr)))
		} else if distanceUnit == "character" {
			// Character-based distance: similar limitation
			termConditions = append(termConditions, fmt.Sprintf("content LIKE '%%%s%%'", escapeSQL(termStr)))
		}
	}

	// All terms must appear (AND logic)
	if len(termConditions) > 0 {
		return "(" + strings.Join(termConditions, " AND ") + ")"
	}

	return ""
}

// escapeSQL escapes single quotes in SQL strings
func escapeSQL(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}
