package ontology

import (
	"context"
	"fmt"
	"sort"

	"github.com/kennethstott/doculyzer-go-conversion/internal/embeddings"
	"github.com/kennethstott/doculyzer-go-conversion/internal/udml/query"
	"github.com/kennethstott/doculyzer-go-conversion/internal/udml/sampler"
)

// DomainCatalogProvider is an interface for accessing domain catalogs
// This avoids import cycle with the catalogs package
type DomainCatalogProvider interface {
	ListDomains() []string
	GetDomainDescription(domain string) (description string, subdomains []string, terms []string, entities []string, exists bool)
}

// DomainSimilarity represents similarity between corpus and domain
type DomainSimilarity struct {
	Domain      string   `json:"domain"`
	Description string   `json:"description"`
	Similarity  float64  `json:"similarity"`
	Subdomains  []string `json:"subdomains,omitempty"`
}

// DomainSimilarityResult contains domain similarity analysis results
type DomainSimilarityResult struct {
	CorpusEmbedding    []float64          `json:"-"` // Corpus average embedding
	DomainSimilarities []DomainSimilarity `json:"domain_similarities"`
	TopDomains         []DomainSimilarity `json:"top_domains"` // Top N by similarity
	AnalysisMethod     string             `json:"analysis_method"`
}

// ComputeDomainSimilarities analyzes corpus samples and computes similarity to all domain catalogs
// This provides the initial domain suggestion before LLM refinement
func (b *OntologyBuilder) ComputeDomainSimilarities(ctx context.Context, samples *sampler.SamplingResult, catalogProvider DomainCatalogProvider, embGen embeddings.EmbeddingGenerator, topN int) (*DomainSimilarityResult, error) {
	// Step 1: Compute corpus embedding (average of sample embeddings)
	corpusEmbedding, err := b.computeCorpusEmbedding(ctx, samples, embGen)
	if err != nil {
		return nil, fmt.Errorf("failed to compute corpus embedding: %w", err)
	}

	// Step 2: Compute domain embeddings and similarities
	domainSimilarities := []DomainSimilarity{}

	for _, domainName := range catalogProvider.ListDomains() {
		description, subdomains, terms, entities, exists := catalogProvider.GetDomainDescription(domainName)
		if !exists {
			continue
		}

		// Generate domain description embedding
		domainDesc := b.generateDomainDescriptionFromComponents(description, terms, entities)
		domainEmbedding, err := embGen.Generate(domainDesc)
		if err != nil {
			return nil, fmt.Errorf("failed to generate domain embedding for %s: %w", domainName, err)
		}

		// Compute cosine similarity
		similarity, err := query.CosineSimilarityFloat64(corpusEmbedding, domainEmbedding)
		if err != nil {
			return nil, fmt.Errorf("failed to compute similarity for %s: %w", domainName, err)
		}

		domainSimilarities = append(domainSimilarities, DomainSimilarity{
			Domain:      domainName,
			Description: description,
			Similarity:  similarity,
			Subdomains:  subdomains,
		})
	}

	// Step 3: Sort by similarity (descending)
	sort.Slice(domainSimilarities, func(i, j int) bool {
		return domainSimilarities[i].Similarity > domainSimilarities[j].Similarity
	})

	// Step 4: Extract top N
	topDomains := domainSimilarities
	if topN > 0 && topN < len(domainSimilarities) {
		topDomains = domainSimilarities[:topN]
	}

	return &DomainSimilarityResult{
		CorpusEmbedding:    corpusEmbedding,
		DomainSimilarities: domainSimilarities,
		TopDomains:         topDomains,
		AnalysisMethod:     "cosine_similarity",
	}, nil
}

// computeCorpusEmbedding generates an average embedding representing the corpus
func (b *OntologyBuilder) computeCorpusEmbedding(ctx context.Context, samples *sampler.SamplingResult, embGen embeddings.EmbeddingGenerator) ([]float64, error) {
	// Strategy: Generate embeddings for sample texts and average them
	maxSamplesForEmbedding := 50 // Limit for performance
	sampleTexts := []string{}

	for i, sample := range samples.Samples {
		if i >= maxSamplesForEmbedding {
			break
		}
		// Truncate long content
		content := sample.Content
		if len(content) > 500 {
			content = content[:500]
		}
		sampleTexts = append(sampleTexts, content)
	}

	if len(sampleTexts) == 0 {
		return nil, fmt.Errorf("no samples available for embedding")
	}

	// Generate embeddings
	embeddings, err := embGen.GenerateBatch(sampleTexts)
	if err != nil {
		return nil, fmt.Errorf("failed to generate sample embeddings: %w", err)
	}

	// Compute average embedding
	dimensions := embGen.GetDimensions()
	avgEmbedding := make([]float64, dimensions)

	for _, emb := range embeddings {
		for i := 0; i < dimensions; i++ {
			avgEmbedding[i] += emb[i]
		}
	}

	// Normalize by count
	count := float64(len(embeddings))
	for i := 0; i < dimensions; i++ {
		avgEmbedding[i] /= count
	}

	return avgEmbedding, nil
}

// generateDomainDescriptionFromComponents creates a representative text description for a domain
func (b *OntologyBuilder) generateDomainDescriptionFromComponents(description string, terms []string, entities []string) string {
	// Combine description, terms, and entity types into a representative text
	fullDesc := description

	if len(terms) > 0 {
		fullDesc += ". Common terms: " + joinStrings(terms, ", ")
	}
	if len(entities) > 0 {
		fullDesc += ". Entity types: " + joinStrings(entities, ", ")
	}

	return fullDesc
}

// Helper function to join strings with limit
func joinStrings(strs []string, sep string) string {
	maxLen := 20 // Limit number of terms/entities
	if len(strs) > maxLen {
		strs = strs[:maxLen]
	}

	result := ""
	for i, s := range strs {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}
