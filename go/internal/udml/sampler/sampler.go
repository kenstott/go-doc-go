package sampler

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strings"
	"time"

	"github.com/kennethstott/doculyzer-go-conversion/internal/resolver"
	"github.com/kennethstott/doculyzer-go-conversion/internal/udml/query"
)

// Sampler performs stratified sampling from UDML Parquet storage
type Sampler struct {
	backend  query.QueryBackend
	config   SamplerConfig
	resolver resolver.ContentResolver
}

// SamplerConfig defines sampling parameters
type SamplerConfig struct {
	ParquetPath         string            // Path to Parquet storage
	SampleSize          int               // Total number of elements to sample
	StratifyBy          string            // Field to stratify by (e.g., "element_type", "doc_id")
	MinPerStratum       int               // Minimum samples per stratum
	MaxTextLength       int               // Maximum text length per element
	ElementTypes        []string          // Filter by element types (empty = all)
	ExcludeContainers   bool              // Exclude container elements (table, list, div, nav, root, body)
	PreferEmbeddingText bool              // Use embeddings.text (with ancestor context) instead of content - automatically filters to leaf elements
	Metadata            map[string]string // Additional metadata filters
	RandomSeed          int64             // Random seed for reproducibility
	IncludeMetadata     bool              // Include element metadata in samples
	IncludeEmbedding    bool              // Include embedding vectors if available
	DiversityThreshold  float64           // Cosine similarity threshold for diversity filtering (0.0-1.0, default 0.85). Samples with similarity >= threshold are filtered out. Higher threshold = more diversity.
}

// Sample represents a sampled UDML element
type Sample struct {
	ElementID      string                 `json:"element_id"`
	DocID          string                 `json:"doc_id"`
	ElementType    string                 `json:"element_type"`
	Content        string                 `json:"content"`
	ContentPreview string                 `json:"content_preview"`
	ParentID       string                 `json:"parent_id,omitempty"`
	ElementOrder   float64                `json:"element_order"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	Embedding      []float64              `json:"embedding,omitempty"`      // Changed to float64 to match DuckDB DOUBLE[]
	EmbeddingText  string                 `json:"embedding_text,omitempty"` // Rich contextual text from embeddings table
}

// SamplingResult contains samples and statistics
type SamplingResult struct {
	Samples           []Sample                  `json:"samples"`
	TotalElements     int64                     `json:"total_elements"`
	SampledCount      int                       `json:"sampled_count"`
	StratumStats      map[string]StratumStats   `json:"stratum_stats"`
	EntityFrequencies map[string]int            `json:"entity_frequencies,omitempty"`
	SamplingTime      time.Duration             `json:"sampling_time"`
	Config            SamplerConfig             `json:"config"`
}

// StratumStats contains statistics for a single stratum
type StratumStats struct {
	StratumValue   string  `json:"stratum_value"`
	TotalCount     int64   `json:"total_count"`
	SampledCount   int     `json:"sampled_count"`
	SamplingRate   float64 `json:"sampling_rate"`
}

// NewSampler creates a new UDML sampler
func NewSampler(config SamplerConfig) (*Sampler, error) {
	// Create DuckDB backend
	backendConfig := query.BackendConfig{
		Type:        "duckdb",
		ParquetPath: config.ParquetPath,
	}

	backend, err := query.NewDuckDBBackend(backendConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create backend: %w", err)
	}

	// Initialize backend
	ctx := context.Background()
	if err := backend.Initialize(ctx, backendConfig); err != nil {
		return nil, fmt.Errorf("failed to initialize backend: %w", err)
	}

	// Set defaults
	if config.SampleSize == 0 {
		config.SampleSize = 1000
	}
	if config.MinPerStratum == 0 {
		config.MinPerStratum = 10
	}
	if config.MaxTextLength == 0 {
		config.MaxTextLength = 2000
	}
	if config.StratifyBy == "" {
		config.StratifyBy = "element_type"
	}
	if config.RandomSeed == 0 {
		config.RandomSeed = time.Now().UnixNano()
	}
	if config.DiversityThreshold == 0 {
		config.DiversityThreshold = 0.85 // Default to 0.85 - filters out samples with >85% similarity (near-duplicates)
	}

	return &Sampler{
		backend:  backend,
		config:   config,
		resolver: nil, // No resolver by default
	}, nil
}

// SetResolver sets the content resolver for resolving element content from source documents
func (s *Sampler) SetResolver(resolver resolver.ContentResolver) {
	s.resolver = resolver
}

// hasEmbeddingsTable checks if embeddings table exists in the Parquet storage
func (s *Sampler) hasEmbeddingsTable(ctx context.Context) bool {
	// Try to query embeddings table
	expr := &query.Expression{
		QueryID: "check_embeddings",
		Select: []query.FieldSelection{
			{Field: "COUNT(*)", Alias: "count"},
		},
		From:  "embeddings",
		Limit: 1,
	}

	nativeQuery, err := s.backend.Translate(expr, query.TranslateOptions{
		EnablePartitions: true,
		EnablePushdown:   false,
	})
	if err != nil {
		return false
	}

	_, err = s.backend.Execute(ctx, nativeQuery)
	return err == nil
}

// Sample performs stratified sampling from the UDML corpus
func (s *Sampler) Sample(ctx context.Context) (*SamplingResult, error) {
	startTime := time.Now()
	rand.Seed(s.config.RandomSeed)

	result := &SamplingResult{
		Samples:      []Sample{},
		StratumStats: make(map[string]StratumStats),
		Config:       s.config,
	}

	// Check if embeddings table exists and should be used
	useEmbeddings := s.config.PreferEmbeddingText && s.hasEmbeddingsTable(ctx)
	if useEmbeddings {
		fmt.Println("SAMPLER: Using embeddings table for sampling (INNER JOIN + embeddings.text)")
	} else if s.config.PreferEmbeddingText {
		fmt.Println("SAMPLER: Embeddings table not found - falling back to content resolver path")
	}

	// Step 1: Get stratum counts
	stratumCounts, totalCount, err := s.getStratumCounts(ctx, useEmbeddings)
	if err != nil {
		return nil, fmt.Errorf("failed to get stratum counts: %w", err)
	}
	result.TotalElements = totalCount

	// Step 2: Calculate sampling allocation per stratum
	allocation := s.calculateAllocation(stratumCounts, totalCount)

	// Step 3: Sample from each stratum
	for stratumValue, sampleCount := range allocation {
		samples, err := s.sampleFromStratum(ctx, stratumValue, sampleCount, useEmbeddings)
		if err != nil {
			return nil, fmt.Errorf("failed to sample from stratum %s: %w", stratumValue, err)
		}

		result.Samples = append(result.Samples, samples...)

		// Update stats
		result.StratumStats[stratumValue] = StratumStats{
			StratumValue: stratumValue,
			TotalCount:   stratumCounts[stratumValue],
			SampledCount: len(samples),
			SamplingRate: float64(len(samples)) / float64(stratumCounts[stratumValue]),
		}
	}

	result.SampledCount = len(result.Samples) // Set count BEFORE diversity filtering

	// Step 4: Apply cosine similarity diversity filter if using embeddings
	if useEmbeddings && s.config.IncludeEmbedding {
		beforeCount := len(result.Samples)
		result.Samples = s.applyCosineSimilarityDiversity(result.Samples, s.config.DiversityThreshold)
		fmt.Printf("SAMPLER: Applied cosine similarity diversity filter (threshold=%.2f) - kept %d/%d samples\n",
			s.config.DiversityThreshold, len(result.Samples), beforeCount)
		result.SampledCount = len(result.Samples)
	}

	result.SamplingTime = time.Since(startTime)

	return result, nil
}

// getStratumCounts returns the count of elements in each stratum
func (s *Sampler) getStratumCounts(ctx context.Context, useEmbeddings bool) (map[string]int64, int64, error) {
	var fromClause string
	if useEmbeddings {
		// INNER JOIN with embeddings - automatically filters to leaf elements
		fromClause = "elements INNER JOIN embeddings ON elements.element_id = embeddings.element_id"
	} else {
		fromClause = "elements"
	}

	// Build query to count by stratum
	expr := &query.Expression{
		QueryID: "stratum_counts",
		Select: []query.FieldSelection{
			{Field: "elements." + s.config.StratifyBy, Alias: "stratum"},
			{Field: "COUNT(*)", Alias: "count"},
		},
		From:    fromClause,
		GroupBy: []string{"elements." + s.config.StratifyBy},
	}

	// Add filters
	var predicates []*query.Predicate

	if len(s.config.ElementTypes) > 0 {
		predicates = append(predicates, &query.Predicate{
			Type:     query.PredicateComparison,
			Field:    "elements.element_type",
			Operator: query.OpIn,
			Value:    s.config.ElementTypes,
		})
	}

	// Path 1 (No embeddings): Filter elements with content_preview length > 0
	// Exclude container elements if requested
	if !useEmbeddings && s.config.ExcludeContainers {
		containerTypes := []string{"table", "list", "div", "nav", "root", "body", "figure"}
		predicates = append(predicates, &query.Predicate{
			Type:     query.PredicateComparison,
			Field:    "elements.element_type",
			Operator: query.OpNotIn,
			Value:    containerTypes,
		})
	}

	// Path 1: Also filter to elements with non-empty content_preview
	if !useEmbeddings {
		predicates = append(predicates, &query.Predicate{
			Type:     query.PredicateComparison,
			Field:    "LENGTH(elements.content_preview)",
			Operator: query.OpGreaterThan,
			Value:    0,
		})
	}

	// Combine predicates
	if len(predicates) > 0 {
		if len(predicates) == 1 {
			expr.Where = predicates[0]
		} else {
			expr.Where = &query.Predicate{
				Type:     query.PredicateAnd,
				Children: predicates,
			}
		}
	}

	// Translate and execute
	nativeQuery, err := s.backend.Translate(expr, query.TranslateOptions{
		EnablePartitions: true,
		EnablePushdown:   true,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to translate query: %w", err)
	}

	queryResult, err := s.backend.Execute(ctx, nativeQuery)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to execute query: %w", err)
	}

	// Parse results
	stratumCounts := make(map[string]int64)
	var totalCount int64

	for _, row := range queryResult.Rows {
		stratum := fmt.Sprintf("%v", row["stratum"])
		count := s.extractInt64(row["count"])
		stratumCounts[stratum] = count
		totalCount += count
	}

	return stratumCounts, totalCount, nil
}

// calculateAllocation calculates how many samples to take from each stratum
func (s *Sampler) calculateAllocation(stratumCounts map[string]int64, totalCount int64) map[string]int {
	allocation := make(map[string]int)

	// Proportional allocation
	for stratum, count := range stratumCounts {
		proportion := float64(count) / float64(totalCount)
		allocated := int(float64(s.config.SampleSize) * proportion)

		// Ensure minimum per stratum
		if allocated < s.config.MinPerStratum && count >= int64(s.config.MinPerStratum) {
			allocated = s.config.MinPerStratum
		}

		// Don't sample more than available
		if allocated > int(count) {
			allocated = int(count)
		}

		if allocated > 0 {
			allocation[stratum] = allocated
		}
	}

	return allocation
}

// sampleFromStratum samples elements from a specific stratum
func (s *Sampler) sampleFromStratum(ctx context.Context, stratumValue string, sampleCount int, useEmbeddings bool) ([]Sample, error) {
	var fromClause string
	var selectFields []query.FieldSelection

	if useEmbeddings {
		// Path 2: INNER JOIN with embeddings table
		fromClause = "elements INNER JOIN embeddings ON elements.element_id = embeddings.element_id"
		selectFields = []query.FieldSelection{
			{Field: "elements.element_id", Alias: "element_id"},
			{Field: "elements.doc_id", Alias: "doc_id"},
			{Field: "elements.element_type", Alias: "element_type"},
			{Field: "embeddings.text", Alias: "embedding_text"},     // Rich contextual text
			{Field: "elements.content_preview", Alias: "content_preview"},
			{Field: "elements.parent_id", Alias: "parent_id"},
			{Field: "elements.element_order", Alias: "element_order"},
		}

		if s.config.IncludeMetadata {
			selectFields = append(selectFields, query.FieldSelection{Field: "elements.metadata", Alias: "metadata"})
		}

		if s.config.IncludeEmbedding {
			selectFields = append(selectFields, query.FieldSelection{Field: "embeddings.embedding", Alias: "embedding"})
		}
	} else {
		// Path 1: Elements only with content resolver
		fromClause = "elements"
		selectFields = []query.FieldSelection{
			{Field: "element_id"},
			{Field: "doc_id"},
			{Field: "element_type"},
			{Field: "content"},
			{Field: "content_preview"},
			{Field: "content_location"}, // Need this for resolver
			{Field: "parent_id"},
			{Field: "element_order"},
		}

		if s.config.IncludeMetadata {
			selectFields = append(selectFields, query.FieldSelection{Field: "metadata"})
		}
	}

	// Build query with random sampling
	expr := &query.Expression{
		QueryID: fmt.Sprintf("sample_%s", stratumValue),
		Select:  selectFields,
		From:    fromClause,
		Where: &query.Predicate{
			Type:     query.PredicateComparison,
			Field:    "elements." + s.config.StratifyBy,
			Operator: query.OpEqual,
			Value:    stratumValue,
		},
		OrderBy: []query.OrderByClause{
			{Field: "RANDOM()", Descending: false},
		},
		Limit: sampleCount,
	}

	// Add additional filters
	var additionalPredicates []*query.Predicate

	if len(s.config.ElementTypes) > 0 && s.config.StratifyBy != "element_type" {
		additionalPredicates = append(additionalPredicates, &query.Predicate{
			Type:     query.PredicateComparison,
			Field:    "elements.element_type",
			Operator: query.OpIn,
			Value:    s.config.ElementTypes,
		})
	}

	// Path 1: Exclude container elements and filter on content_preview length
	if !useEmbeddings {
		if s.config.ExcludeContainers {
			containerTypes := []string{"table", "list", "div", "nav", "root", "body", "figure"}
			additionalPredicates = append(additionalPredicates, &query.Predicate{
				Type:     query.PredicateComparison,
				Field:    "elements.element_type",
				Operator: query.OpNotIn,
				Value:    containerTypes,
			})
		}

		// Filter to elements with non-empty content_preview
		additionalPredicates = append(additionalPredicates, &query.Predicate{
			Type:     query.PredicateComparison,
			Field:    "LENGTH(elements.content_preview)",
			Operator: query.OpGreaterThan,
			Value:    0,
		})
	}

	// Combine with existing where clause
	if len(additionalPredicates) > 0 {
		allPredicates := append([]*query.Predicate{expr.Where}, additionalPredicates...)
		expr.Where = &query.Predicate{
			Type:     query.PredicateAnd,
			Children: allPredicates,
		}
	}

	// Translate and execute
	nativeQuery, err := s.backend.Translate(expr, query.TranslateOptions{
		EnablePartitions: true,
		EnablePushdown:   true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to translate query: %w", err)
	}

	queryResult, err := s.backend.Execute(ctx, nativeQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}

	// Parse results into samples
	samples := make([]Sample, 0, len(queryResult.Rows))
	for _, row := range queryResult.Rows {
		var sample Sample

		if useEmbeddings {
			// Path 2: Use embeddings.text directly (rich contextual text)
			embeddingText := s.extractString(row["embedding_text"])

			sample = Sample{
				ElementID:      s.extractString(row["element_id"]),
				DocID:          s.extractString(row["doc_id"]),
				ElementType:    s.extractString(row["element_type"]),
				Content:        embeddingText, // Use rich embedding text as content
				EmbeddingText:  embeddingText,
				ContentPreview: s.extractString(row["content_preview"]),
				ParentID:       s.extractString(row["parent_id"]),
				ElementOrder:   s.extractFloat64(row["element_order"]),
			}

			// Extract embedding vector if requested
			if s.config.IncludeEmbedding {
				sample.Embedding = s.extractEmbedding(row["embedding"])
			}
		} else {
			// Path 1: Use 3-part waterfall for content resolution
			content := s.extractString(row["content"])
			contentPreview := s.extractString(row["content_preview"])
			contentLocation := s.extractJSONField(row["content_location"])

			resolvedContent := s.resolveElementContent(content, contentLocation, contentPreview)

			sample = Sample{
				ElementID:      s.extractString(row["element_id"]),
				DocID:          s.extractString(row["doc_id"]),
				ElementType:    s.extractString(row["element_type"]),
				Content:        resolvedContent,
				ContentPreview: contentPreview,
				ParentID:       s.extractString(row["parent_id"]),
				ElementOrder:   s.extractFloat64(row["element_order"]),
			}
		}

		// Truncate content if needed
		if s.config.MaxTextLength > 0 && len(sample.Content) > s.config.MaxTextLength {
			sample.Content = sample.Content[:s.config.MaxTextLength] + "..."
		}

		// Extract metadata if included
		if s.config.IncludeMetadata {
			if metadata, ok := row["metadata"].(map[string]interface{}); ok {
				sample.Metadata = metadata
			}
		}

		samples = append(samples, sample)
	}

	return samples, nil
}

// AnalyzeEntityFrequencies analyzes entity frequencies in the samples
func (s *Sampler) AnalyzeEntityFrequencies(samples []Sample) map[string]int {
	frequencies := make(map[string]int)

	for _, sample := range samples {
		// Simple word-based entity extraction for frequency analysis
		// This is a rough heuristic - proper entity extraction happens in the ontology builder
		words := s.extractCandidateEntities(sample.Content)
		for _, word := range words {
			frequencies[word]++
		}
	}

	return frequencies
}

// extractCandidateEntities extracts potential entity candidates from text
// This is a simple heuristic for frequency analysis
func (s *Sampler) extractCandidateEntities(text string) []string {
	var entities []string

	// Split into words
	words := strings.Fields(text)

	for i := 0; i < len(words); i++ {
		word := strings.Trim(words[i], ".,!?;:()")

		// Look for capitalized words (potential proper nouns)
		if len(word) > 2 && isCapitalized(word) {
			// Check for multi-word entities (e.g., "Microsoft Corporation")
			if i+1 < len(words) {
				nextWord := strings.Trim(words[i+1], ".,!?;:()")
				if isCapitalized(nextWord) {
					entity := word + " " + nextWord
					entities = append(entities, entity)
					i++ // Skip next word
					continue
				}
			}

			entities = append(entities, word)
		}
	}

	return entities
}

// GetTopEntities returns the top N most frequent entities
func (result *SamplingResult) GetTopEntities(n int) []EntityFrequency {
	var freqs []EntityFrequency
	for entity, count := range result.EntityFrequencies {
		freqs = append(freqs, EntityFrequency{
			Entity: entity,
			Count:  count,
		})
	}

	// Sort by count descending
	sort.Slice(freqs, func(i, j int) bool {
		return freqs[i].Count > freqs[j].Count
	})

	if n > 0 && len(freqs) > n {
		freqs = freqs[:n]
	}

	return freqs
}

// EntityFrequency represents an entity and its frequency
type EntityFrequency struct {
	Entity string `json:"entity"`
	Count  int    `json:"count"`
}

// resolveElementContent implements the 3-part waterfall for content resolution
func (s *Sampler) resolveElementContent(content string, contentLocation map[string]interface{}, contentPreview string) string {
	// 1. Try Content field first
	if content != "" {
		return content
	}

	// 2. Try content resolver with content_location
	if contentLocation != nil && s.resolver != nil {
		resolved, err := s.resolver.ResolveContent(contentLocation, true)
		if err == nil && resolved != "" {
			return resolved
		}
	}

	// 3. Fall back to ContentPreview
	return contentPreview
}

// Helper functions for type extraction

func (s *Sampler) extractString(val interface{}) string {
	if val == nil {
		return ""
	}
	if str, ok := val.(string); ok {
		return str
	}
	if bytes, ok := val.([]byte); ok {
		return string(bytes)
	}
	return fmt.Sprintf("%v", val)
}

func (s *Sampler) extractJSONField(val interface{}) map[string]interface{} {
	if val == nil {
		return nil
	}
	if m, ok := val.(map[string]interface{}); ok {
		return m
	}
	return nil
}

func (s *Sampler) extractInt64(val interface{}) int64 {
	if val == nil {
		return 0
	}
	switch v := val.(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case float64:
		return int64(v)
	default:
		return 0
	}
}

func (s *Sampler) extractFloat64(val interface{}) float64 {
	if val == nil {
		return 0.0
	}
	switch v := val.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	default:
		return 0.0
	}
}

func isCapitalized(word string) bool {
	if len(word) == 0 {
		return false
	}
	first := rune(word[0])
	return first >= 'A' && first <= 'Z'
}

func (s *Sampler) extractEmbedding(val interface{}) []float64 {
	if val == nil {
		return nil
	}

	// Try direct []float64 type assertion
	if emb, ok := val.([]float64); ok {
		return emb
	}

	// Try []interface{} (common from JSON/DuckDB)
	if arr, ok := val.([]interface{}); ok {
		result := make([]float64, len(arr))
		for i, v := range arr {
			switch num := v.(type) {
			case float64:
				result[i] = num
			case float32:
				result[i] = float64(num)
			case int:
				result[i] = float64(num)
			case int64:
				result[i] = float64(num)
			default:
				return nil // Unexpected type in array
			}
		}
		return result
	}

	return nil
}

// applyCosineSimilarityDiversity filters samples to ensure diversity using cosine similarity
// Keeps samples whose maximum similarity to existing samples is below the threshold
// This allows for diverse samples while avoiding near-duplicates
func (s *Sampler) applyCosineSimilarityDiversity(samples []Sample, threshold float64) []Sample {
	if len(samples) == 0 || threshold >= 1.0 {
		return samples
	}

	diverse := []Sample{samples[0]} // Always keep first sample

	for i := 1; i < len(samples); i++ {
		candidate := samples[i]
		if len(candidate.Embedding) == 0 {
			continue // Skip samples without embeddings
		}

		// Find the maximum similarity to any existing sample
		maxSimilarity := 0.0
		for _, existing := range diverse {
			if len(existing.Embedding) == 0 {
				continue
			}

			similarity := s.cosineSimilarity(candidate.Embedding, existing.Embedding)
			if similarity > maxSimilarity {
				maxSimilarity = similarity
			}
		}

		// Keep candidate if its maximum similarity is below threshold
		// This ensures we filter out near-duplicates but keep diverse samples
		if maxSimilarity < threshold {
			diverse = append(diverse, candidate)
		}
	}

	return diverse
}

// cosineSimilarity computes cosine similarity between two vectors
func (s *Sampler) cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0.0
	}

	var dotProduct, magA, magB float64
	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
		magA += a[i] * a[i]
		magB += b[i] * b[i]
	}

	if magA == 0 || magB == 0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(magA) * math.Sqrt(magB))
}

// Close closes the sampler and its backend
func (s *Sampler) Close() error {
	if s.backend != nil {
		return s.backend.Close()
	}
	return nil
}
