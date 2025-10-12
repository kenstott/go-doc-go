package ontology

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/kennethstott/go-doc-go/internal/udml/sampler"
)

// OntologyBuilder orchestrates the automatic ontology schema creation process
type OntologyBuilder struct {
	sampler   *sampler.Sampler
	llmClient LLMClient
	config    BuilderConfig
}

// BuilderConfig configures the ontology building process
type BuilderConfig struct {
	// Sampling
	SampleSize    int    // Number of elements to sample
	ParquetPath   string // Path to UDML Parquet storage

	// LLM
	LLMProvider   string // "anthropic", "openai", etc.
	LLMModel      string // Model name
	LLMAPIKey     string // API key
	LLMMaxTokens  int    // Max tokens for LLM responses

	// Analysis
	TopEntityCount      int     // Top N entities to analyze for aliases
	MinEntityFrequency  int     // Minimum frequency for entity consideration
	ConfidenceThreshold float64 // Minimum confidence for rules

	// Output
	SchemaName    string // Name for the generated schema
	SchemaVersion string // Version
	Domain        string // Domain name
}

// LLMClient defines the interface for LLM providers
type LLMClient interface {
	// Complete sends a prompt to the LLM and returns the response
	Complete(ctx context.Context, prompt string, options LLMOptions) (string, error)

	// GetProvider returns the provider name
	GetProvider() string

	// GetModel returns the model name
	GetModel() string
}

// LLMOptions configures an LLM request
type LLMOptions struct {
	MaxTokens   int
	Temperature float64
	SystemPrompt string
}

// BuildResult contains the output of the build process
type BuildResult struct {
	Schema           *OntologySchema
	DraftSchema      *OntologySchema // Initial automatic draft
	Samples          *sampler.SamplingResult
	TopEntities      []sampler.EntityFrequency
	AnalysisLog      []string // Log of analysis steps
	UserRefinements  []string // Log of user changes
	BuildTime        time.Duration
	LLMCallCount     int
	TotalLLMTokens   int
}

// NewOntologyBuilder creates a new ontology builder
func NewOntologyBuilder(config BuilderConfig) (*OntologyBuilder, error) {
	// Create sampler
	samplerConfig := sampler.SamplerConfig{
		ParquetPath:      config.ParquetPath,
		SampleSize:       config.SampleSize,
		MaxTextLength:    2000,
		IncludeMetadata:  true,
		IncludeEmbedding: false,
	}

	samp, err := sampler.NewSampler(samplerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create sampler: %w", err)
	}

	// Create LLM client
	var llmClient LLMClient
	switch config.LLMProvider {
	case "anthropic":
		llmClient = NewAnthropicClient(config.LLMAPIKey, config.LLMModel)
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", config.LLMProvider)
	}

	// Set defaults
	if config.TopEntityCount == 0 {
		config.TopEntityCount = 50
	}
	if config.MinEntityFrequency == 0 {
		config.MinEntityFrequency = 5
	}
	if config.ConfidenceThreshold == 0 {
		config.ConfidenceThreshold = 0.7
	}
	if config.LLMMaxTokens == 0 {
		config.LLMMaxTokens = 4096
	}
	if config.SchemaVersion == "" {
		config.SchemaVersion = "1.0.0"
	}

	return &OntologyBuilder{
		sampler:   samp,
		llmClient: llmClient,
		config:    config,
	}, nil
}

// Build executes the automatic ontology building process
func (b *OntologyBuilder) Build(ctx context.Context) (*BuildResult, error) {
	startTime := time.Now()
	result := &BuildResult{
		AnalysisLog: []string{},
	}

	// Phase 1: Sample corpus
	b.log(result, "Phase 1: Sampling UDML corpus...")
	samples, err := b.sampler.Sample(ctx)
	if err != nil {
		return nil, fmt.Errorf("sampling failed: %w", err)
	}
	result.Samples = samples
	b.log(result, fmt.Sprintf("Sampled %d elements from %d total", samples.SampledCount, samples.TotalElements))

	// Phase 2: Analyze entity frequencies
	b.log(result, "Phase 2: Analyzing entity frequencies...")
	samples.EntityFrequencies = b.sampler.AnalyzeEntityFrequencies(samples.Samples)
	result.TopEntities = samples.GetTopEntities(b.config.TopEntityCount)
	b.log(result, fmt.Sprintf("Found %d unique entities, top %d selected", len(samples.EntityFrequencies), len(result.TopEntities)))

	// Phase 3: Generate draft schema using LLM
	b.log(result, "Phase 3: Generating draft ontology schema with LLM...")
	draftSchema, llmCalls, tokens, err := b.generateDraftSchema(ctx, samples, result.TopEntities)
	if err != nil {
		return nil, fmt.Errorf("draft generation failed: %w", err)
	}
	result.DraftSchema = draftSchema
	result.LLMCallCount += llmCalls
	result.TotalLLMTokens += tokens
	b.log(result, fmt.Sprintf("Generated draft schema with %d entity types, %d relationship types",
		len(draftSchema.ElementEntityMappings), len(draftSchema.EntityRelationshipRules)))

	// Phase 4: Schema is ready (refinement happens externally via CLI)
	result.Schema = draftSchema
	result.BuildTime = time.Since(startTime)

	b.log(result, fmt.Sprintf("Build complete in %v", result.BuildTime))
	return result, nil
}

// generateDraftSchema uses LLM to create initial schema
func (b *OntologyBuilder) generateDraftSchema(ctx context.Context, samples *sampler.SamplingResult, topEntities []sampler.EntityFrequency) (*OntologySchema, int, int, error) {
	llmCalls := 0
	totalTokens := 0

	// Step 1: Identify domains and key concepts
	domains, keyConcepts, calls1, tokens1, err := b.identifyDomains(ctx, samples)
	if err != nil {
		return nil, llmCalls, totalTokens, err
	}
	llmCalls += calls1
	totalTokens += tokens1

	// Step 2: Define entity types and extraction rules
	entityMappings, calls2, tokens2, err := b.defineEntityTypes(ctx, samples, topEntities)
	if err != nil {
		return nil, llmCalls, totalTokens, err
	}
	llmCalls += calls2
	totalTokens += tokens2

	// Step 3: Define relationship types
	relationshipRules, calls3, tokens3, err := b.defineRelationshipTypes(ctx, samples, entityMappings)
	if err != nil {
		return nil, llmCalls, totalTokens, err
	}
	llmCalls += calls3
	totalTokens += tokens3

	// Assemble schema
	schema := &OntologySchema{
		Name:                    b.config.SchemaName,
		Version:                 b.config.SchemaVersion,
		Description:             fmt.Sprintf("Automatically generated ontology for %s domain", domains),
		Domain:                  domains,
		KeyConcepts:             keyConcepts,
		ElementEntityMappings:   entityMappings,
		EntityRelationshipRules: relationshipRules,
		CreatedAt:               time.Now(),
	}

	return schema, llmCalls, totalTokens, nil
}

// identifyDomains asks LLM to identify domains and key concepts
func (b *OntologyBuilder) identifyDomains(ctx context.Context, samples *sampler.SamplingResult) (string, []string, int, int, error) {
	// Prepare sample text
	sampleTexts := b.prepareSampleTexts(samples.Samples, 20)

	prompt := fmt.Sprintf(`Analyze the following document samples and identify:
1. The primary DOMAIN (e.g., "financial", "technical", "medical", "legal")
2. KEY CONCEPTS that appear frequently (5-10 concepts)

Sample texts:
%s

Return your analysis in JSON format:
{
  "domain": "domain name",
  "key_concepts": ["concept1", "concept2", ...]
}`, sampleTexts)

	response, err := b.llmClient.Complete(ctx, prompt, LLMOptions{
		MaxTokens:   1000,
		Temperature: 0.3,
		SystemPrompt: "You are an expert at analyzing document corpora and identifying domains and key concepts.",
	})
	if err != nil {
		return "", nil, 1, 0, err
	}

	// Parse response
	var result struct {
		Domain      string   `json:"domain"`
		KeyConcepts []string `json:"key_concepts"`
	}

	if err := b.extractJSON(response, &result); err != nil {
		return "", nil, 1, len(response), err
	}

	return result.Domain, result.KeyConcepts, 1, len(response), nil
}

// defineEntityTypes asks LLM to define entity types and extraction rules
func (b *OntologyBuilder) defineEntityTypes(ctx context.Context, samples *sampler.SamplingResult, topEntities []sampler.EntityFrequency) ([]ElementEntityMapping, int, int, error) {
	sampleTexts := b.prepareSampleTexts(samples.Samples, 15)

	// Format top entities
	entityList := make([]string, 0, len(topEntities))
	for i, e := range topEntities {
		if i >= 20 { // Limit to top 20
			break
		}
		entityList = append(entityList, fmt.Sprintf("%s (%d occurrences)", e.Entity, e.Count))
	}

	prompt := fmt.Sprintf(`Analyze these document samples and define entity types to extract.

Top entities found:
%s

Sample texts:
%s

For each entity type, provide:
1. entity_type: Type name (e.g., "organization", "person", "product")
2. description: What this type represents
3. element_types: Which element types to process (e.g., ["paragraph", "heading"])
4. extraction_rules: Array of rules with:
   - type: "keyword_match", "regex_pattern", or "metadata_field"
   - keywords: Array of keywords/aliases (for keyword_match)
   - pattern: Regex pattern (for regex_pattern)
   - field_path: Metadata path (for metadata_field)
   - confidence: Confidence score (0.0-1.0)

For top entities, automatically include aliases. For example:
- Microsoft → ["Microsoft", "MSFT", "MS", "Microsoft Corporation"]
- Apple → ["Apple", "Apple Inc", "AAPL", "Apple Computer"]

Return JSON array of entity mappings:
[
  {
    "entity_type": "organization",
    "description": "Companies and organizations",
    "element_types": ["paragraph"],
    "extraction_rules": [
      {
        "type": "keyword_match",
        "keywords": ["Microsoft", "MSFT", "Microsoft Corporation"],
        "confidence": 0.9
      }
    ]
  }
]`, strings.Join(entityList, "\n"), sampleTexts)

	response, err := b.llmClient.Complete(ctx, prompt, LLMOptions{
		MaxTokens:   3000,
		Temperature: 0.3,
		SystemPrompt: "You are an expert at entity extraction and ontology design.",
	})
	if err != nil {
		return nil, 1, 0, err
	}

	var mappings []ElementEntityMapping
	if err := b.extractJSON(response, &mappings); err != nil {
		return nil, 1, len(response), err
	}

	return mappings, 1, len(response), nil
}

// defineRelationshipTypes asks LLM to define relationship types
func (b *OntologyBuilder) defineRelationshipTypes(ctx context.Context, samples *sampler.SamplingResult, entityMappings []ElementEntityMapping) ([]EntityRelationshipRule, int, int, error) {
	sampleTexts := b.prepareSampleTexts(samples.Samples, 10)

	// Format entity types
	entityTypes := make([]string, len(entityMappings))
	for i, m := range entityMappings {
		entityTypes[i] = m.EntityType
	}

	prompt := fmt.Sprintf(`Given these entity types: %s

Analyze these sample texts and define relationship types between entities:
%s

For each relationship type, provide:
1. name: Relationship name
2. source_entity_type: Source entity type
3. target_entity_type: Target entity type
4. relationship_type: One of [is_a, part_of, related_to, located_in, occurred_at, created_by, mentions, depends_on]
5. description: What this relationship represents
6. confidence_threshold: Minimum confidence (0.0-1.0)

Return JSON array:
[
  {
    "name": "person_works_at_org",
    "source_entity_type": "person",
    "target_entity_type": "organization",
    "relationship_type": "part_of",
    "description": "Person employed by organization",
    "confidence_threshold": 0.7
  }
]`, strings.Join(entityTypes, ", "), sampleTexts)

	response, err := b.llmClient.Complete(ctx, prompt, LLMOptions{
		MaxTokens:   2000,
		Temperature: 0.3,
		SystemPrompt: "You are an expert at identifying relationships in text.",
	})
	if err != nil {
		return nil, 1, 0, err
	}

	var rules []EntityRelationshipRule
	if err := b.extractJSON(response, &rules); err != nil {
		return nil, 1, len(response), err
	}

	return rules, 1, len(response), nil
}

// Helper functions

func (b *OntologyBuilder) prepareSampleTexts(samples []sampler.Sample, maxSamples int) string {
	var texts []string
	for i, sample := range samples {
		if i >= maxSamples {
			break
		}
		texts = append(texts, fmt.Sprintf("[%s] %s", sample.ElementType, sample.Content))
	}
	return strings.Join(texts, "\n\n")
}

func (b *OntologyBuilder) extractJSON(response string, target interface{}) error {
	// Find JSON in response (handle markdown code blocks)
	jsonStr := response

	// Try to extract from ```json code block
	if start := strings.Index(response, "```json"); start != -1 {
		start += 7
		if end := strings.Index(response[start:], "```"); end != -1 {
			jsonStr = response[start : start+end]
		}
	} else if start := strings.Index(response, "```"); start != -1 {
		start += 3
		if end := strings.Index(response[start:], "```"); end != -1 {
			jsonStr = response[start : start+end]
		}
	}

	jsonStr = strings.TrimSpace(jsonStr)

	if err := json.Unmarshal([]byte(jsonStr), target); err != nil {
		return fmt.Errorf("failed to parse LLM response as JSON: %w\nResponse: %s", err, response)
	}

	return nil
}

func (b *OntologyBuilder) log(result *BuildResult, message string) {
	result.AnalysisLog = append(result.AnalysisLog, fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), message))
}

// Close closes the builder and releases resources
func (b *OntologyBuilder) Close() error {
	if b.sampler != nil {
		return b.sampler.Close()
	}
	return nil
}
