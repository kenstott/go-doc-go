package sampler

import (
	"context"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"time"

	"github.com/kennethstott/doculyzer-go-conversion/internal/udml/query"
)

// Sampler performs stratified sampling from UDML Parquet storage
type Sampler struct {
	backend query.QueryBackend
	config  SamplerConfig
}

// SamplerConfig defines sampling parameters
type SamplerConfig struct {
	ParquetPath      string            // Path to Parquet storage
	SampleSize       int               // Total number of elements to sample
	StratifyBy       string            // Field to stratify by (e.g., "element_type", "doc_id")
	MinPerStratum    int               // Minimum samples per stratum
	MaxTextLength    int               // Maximum text length per element
	ElementTypes     []string          // Filter by element types (empty = all)
	Metadata         map[string]string // Additional metadata filters
	RandomSeed       int64             // Random seed for reproducibility
	IncludeMetadata  bool              // Include element metadata in samples
	IncludeEmbedding bool              // Include embeddings if available
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
	Embedding      []float32              `json:"embedding,omitempty"`
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

	return &Sampler{
		backend: backend,
		config:  config,
	}, nil
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

	// Step 1: Get stratum counts
	stratumCounts, totalCount, err := s.getStratumCounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get stratum counts: %w", err)
	}
	result.TotalElements = totalCount

	// Step 2: Calculate sampling allocation per stratum
	allocation := s.calculateAllocation(stratumCounts, totalCount)

	// Step 3: Sample from each stratum
	for stratumValue, sampleCount := range allocation {
		samples, err := s.sampleFromStratum(ctx, stratumValue, sampleCount)
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

	result.SampledCount = len(result.Samples)
	result.SamplingTime = time.Since(startTime)

	return result, nil
}

// getStratumCounts returns the count of elements in each stratum
func (s *Sampler) getStratumCounts(ctx context.Context) (map[string]int64, int64, error) {
	// Build query to count by stratum
	expr := &query.Expression{
		QueryID: "stratum_counts",
		Select: []query.FieldSelection{
			{Field: s.config.StratifyBy, Alias: "stratum"},
			{Field: "COUNT(*)", Alias: "count"},
		},
		From:    "elements",
		GroupBy: []string{s.config.StratifyBy},
	}

	// Add filters
	if len(s.config.ElementTypes) > 0 {
		expr.Where = &query.Predicate{
			Type:     query.PredicateComparison,
			Field:    "element_type",
			Operator: query.OpIn,
			Value:    s.config.ElementTypes,
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
func (s *Sampler) sampleFromStratum(ctx context.Context, stratumValue string, sampleCount int) ([]Sample, error) {
	// Build select clause
	selectFields := []query.FieldSelection{
		{Field: "element_id"},
		{Field: "doc_id"},
		{Field: "element_type"},
		{Field: "content"},
		{Field: "content_preview"},
		{Field: "parent_id"},
		{Field: "element_order"},
	}

	if s.config.IncludeMetadata {
		selectFields = append(selectFields, query.FieldSelection{Field: "metadata"})
	}

	if s.config.IncludeEmbedding {
		selectFields = append(selectFields, query.FieldSelection{Field: "embedding"})
	}

	// Build query with random sampling
	expr := &query.Expression{
		QueryID: fmt.Sprintf("sample_%s", stratumValue),
		Select:  selectFields,
		From:    "elements",
		Where: &query.Predicate{
			Type:     query.PredicateComparison,
			Field:    s.config.StratifyBy,
			Operator: query.OpEqual,
			Value:    stratumValue,
		},
		OrderBy: []query.OrderByClause{
			{Field: "RANDOM()", Descending: false},
		},
		Limit: sampleCount,
	}

	// Add element type filter if specified
	if len(s.config.ElementTypes) > 0 && s.config.StratifyBy != "element_type" {
		expr.Where = &query.Predicate{
			Type: query.PredicateAnd,
			Children: []*query.Predicate{
				expr.Where,
				{
					Type:     query.PredicateComparison,
					Field:    "element_type",
					Operator: query.OpIn,
					Value:    s.config.ElementTypes,
				},
			},
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
		sample := Sample{
			ElementID:      s.extractString(row["element_id"]),
			DocID:          s.extractString(row["doc_id"]),
			ElementType:    s.extractString(row["element_type"]),
			Content:        s.extractString(row["content"]),
			ContentPreview: s.extractString(row["content_preview"]),
			ParentID:       s.extractString(row["parent_id"]),
			ElementOrder:   s.extractFloat64(row["element_order"]),
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

		// Extract embedding if included
		if s.config.IncludeEmbedding {
			if embedding, ok := row["embedding"].([]float32); ok {
				sample.Embedding = embedding
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

// Close closes the sampler and its backend
func (s *Sampler) Close() error {
	if s.backend != nil {
		return s.backend.Close()
	}
	return nil
}
