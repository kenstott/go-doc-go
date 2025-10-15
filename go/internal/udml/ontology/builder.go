package ontology

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kennethstott/doculyzer-go-conversion/internal/analytics"
	"github.com/kennethstott/doculyzer-go-conversion/internal/resolver"
	"github.com/kennethstott/doculyzer-go-conversion/internal/udml/sampler"
	"gopkg.in/yaml.v3"
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
	SampleSize          int     // Number of elements to sample
	ParquetPath         string  // Path to UDML Parquet storage
	DiversityThreshold  float64 // Cosine similarity threshold for diversity filtering (0.0-1.0, lower = more diverse)

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
	// For ontology building, focus on substantive element types with rich content
	substantiveTypes := []string{"paragraph", "header", "list_item", "table_cell", "caption"}

	// Determine diversity threshold - use config value or default
	diversityThreshold := config.DiversityThreshold
	if diversityThreshold == 0.0 {
		diversityThreshold = 0.0001 // Very low threshold to effectively disable diversity filtering
	}
	fmt.Printf("DEBUG: BuilderConfig.DiversityThreshold = %.6f, using = %.6f\n", config.DiversityThreshold, diversityThreshold)

	samplerConfig := sampler.SamplerConfig{
		ParquetPath:         config.ParquetPath,
		SampleSize:          config.SampleSize,
		MaxTextLength:       2000,
		ElementTypes:        substantiveTypes,    // Focus sampling on substantive content (excludes links, images, divs)
		IncludeMetadata:     false,               // Skip metadata (not present in older Parquet schemas)
		IncludeEmbedding:    true,                // Include embedding vectors for cosine similarity diversity
		PreferEmbeddingText: true,                // Use embeddings.text when available (richer context)
		ExcludeContainers:   true,                // Only sample leaf elements with actual content (fallback for non-embedding path)
		MinPerStratum:       5,                   // Ensure minimum representation from each substantive type
		DiversityThreshold:  diversityThreshold,  // Cosine similarity threshold for diversity filtering
	}

	samp, err := sampler.NewSampler(samplerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create sampler: %w", err)
	}

	// Configure sampler with resolver from analytics storage
	// This enables content resolution from content_location pointers
	if err := configureResolverFromStorage(samp, config.ParquetPath); err != nil {
		// Log warning but don't fail - sampler can work without resolver (falls back to content_preview)
		fmt.Printf("Warning: Failed to configure content resolver: %v\n", err)
		fmt.Println("  Sampler will fall back to content_preview (100 chars)")
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

	// Step 2: Define entity types and extraction rules (with domain assignment)
	entityMappings, calls2, tokens2, err := b.defineEntityTypes(ctx, samples, topEntities, domains)
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

	// Determine primary domain for description
	primaryDomain := "multi-domain"
	if len(domains) == 1 {
		primaryDomain = domains[0].Name
	} else if len(domains) > 1 {
		domainNames := make([]string, len(domains))
		for i, d := range domains {
			domainNames[i] = d.Name
		}
		primaryDomain = strings.Join(domainNames, "/")
	}

	// Assemble schema
	schema := &OntologySchema{
		Name:                    b.config.SchemaName,
		Version:                 b.config.SchemaVersion,
		Description:             fmt.Sprintf("Automatically generated ontology for %s domain(s)", primaryDomain),
		Domain:                  primaryDomain,          // Deprecated: set for backward compatibility
		Domains:                 domains,
		KeyConcepts:             keyConcepts,
		ElementEntityMappings:   entityMappings,
		EntityRelationshipRules: relationshipRules,
		CreatedAt:               time.Now(),
	}

	return schema, llmCalls, totalTokens, nil
}

// loadPredefinedDomains loads domain names from catalog YAML files
func loadPredefinedDomains() []string {
	catalogPaths := []string{}

	// Check environment variable first - allows user to specify catalog location
	if envPath := os.Getenv("ONTOLOGY_CATALOG_PATH"); envPath != "" {
		catalogPaths = append(catalogPaths, envPath)
	}

	// Fallback to relative paths for local development
	catalogPaths = append(catalogPaths,
		"./examples/ontologies",
		"../examples/ontologies",
		"../../examples/ontologies",
	)

	var catalogPath string
	for _, path := range catalogPaths {
		if _, err := os.Stat(path); err == nil {
			catalogPath = path
			break
		}
	}

	if catalogPath == "" {
		// Fallback to hardcoded list if catalog not found
		fmt.Println("⚠️  Domain catalogs not found - using fallback list")
		fmt.Println("   Set ONTOLOGY_CATALOG_PATH environment variable to use rich domain catalogs")
		return []string{
			"compliance", "financial", "human_resources", "legal", "logistics", "manufacturing",
			"marketing", "retail", "sales", "supply_chain", "banking", "entertainment_media",
			"healthcare", "insurance", "leisure", "education", "medical", "technical",
			"nonprofit", "religion",
		}
	}

	fmt.Printf("✓ Loading domain catalogs from: %s\n", catalogPath)
	domains := make([]string, 0, 20)

	// Walk through all subdirectories and find YAML files
	filepath.Walk(catalogPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".yaml") {
			return nil
		}

		// Read YAML file to extract domain name
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		var catalog struct {
			Domain string `yaml:"domain"`
		}

		if err := yaml.Unmarshal(data, &catalog); err != nil {
			return nil
		}

		if catalog.Domain != "" {
			domains = append(domains, catalog.Domain)
		}

		return nil
	})

	return domains
}

// identifyDomains asks LLM to identify domains and key concepts
func (b *OntologyBuilder) identifyDomains(ctx context.Context, samples *sampler.SamplingResult) ([]Domain, []string, int, int, error) {
	// Prepare sample text
	sampleTexts := b.prepareSampleTexts(samples.Samples, 20)

	// Load predefined domains from catalog
	predefinedDomains := loadPredefinedDomains()
	domainListFormatted := strings.Join(predefinedDomains, ", ")

	prompt := fmt.Sprintf(`Analyze the following document samples and identify ALL distinct domains present in the corpus.

## DOMAIN IDENTIFICATION PRINCIPLES

A **domain** represents a SUBJECT AREA or ORGANIZATIONAL FUNCTION with:
- **Clear ownership boundary** (which team/department/discipline owns this?)
- **Distinct vocabulary** (specialized terminology)
- **Logical cohesion** (concepts naturally group together)

## PREDEFINED DOMAINS

**IMPORTANT**: Try to use these PRE-DEFINED domains first: **%s**

Only create new domains if the corpus content clearly doesn't fit any predefined domain.

**Domain categories include**:
- **Business Functions**: compliance, financial, human_resources, legal, logistics, manufacturing, marketing, retail, sales, supply_chain
- **Industry Sectors**: banking, entertainment_media, healthcare, insurance, leisure
- **Academic Disciplines**: education, medical, technical
- **Organizational Types**: nonprofit, religion

## INVALID "DOMAINS" (DO NOT use these)

**DO NOT** classify these as domains:
- ❌ Data types (temporal data, spatial data, binary data)
- ❌ Technical concerns (authentication, logging, caching)
- ❌ Document formats (JSON, XML, CSV, markdown)
- ❌ Infrastructure (storage, networking, compute)
- ❌ Generic attributes (dates, timestamps, IDs, names)

**Examples of WRONG domains**:
- ❌ "temporal_event_tracking" → This is a data type, not a domain
- ❌ "document_metadata" → This is technical metadata, not a subject area
- ❌ "authentication_system" → This is infrastructure, not a domain

## MULTI-DOMAIN DETECTION

Look for MULTIPLE domains. Common patterns:
- **financial** + **legal** in annual reports
- **medical** + **insurance** in healthcare records
- **retail** + **logistics** in e-commerce
- **manufacturing** + **logistics** in supply chain

For each domain, identify:
1. **Name**: Use predefined domain name when applicable (e.g., "financial", "legal", "medical")
2. **Description**: What this domain covers (1-2 sentences)
3. **Owner**: Team/department/discipline owner
4. **Key Concepts**: 3-5 domain-specific concepts

Sample texts:
%s

Return JSON format:
{
  "domains": [
    {
      "name": "financial",
      "description": "Financial performance, metrics, and reporting data",
      "owner": "CFO Office",
      "key_concepts": ["revenue", "profit", "EBITDA", "cash flow"]
    },
    {
      "name": "legal",
      "description": "Legal entities, compliance, and regulatory information",
      "owner": "Legal Department",
      "key_concepts": ["entity", "jurisdiction", "compliance", "regulation"]
    }
  ],
  "overall_key_concepts": ["concept1", "concept2", ...]
}

**SELECTION PROCESS**:
1. **First**, review the PRE-DEFINED domains above
2. **Match** corpus content to existing domains where possible
3. **Only create new domains** if content truly doesn't fit any pre-defined domain
4. Use consistent naming from the pre-defined list (e.g., "financial" not "finance", "legal" not "legal_affairs")

**IMPORTANT**:
- Identify ALL distinct domains (typically 1-5 domains per corpus)
- If truly single-domain, return ONE domain
- Multi-domain is common for enterprise documents (reports, filings, documentation)
- ALWAYS prefer pre-defined domains over creating new ones`, domainListFormatted, sampleTexts)

	response, err := b.llmClient.Complete(ctx, prompt, LLMOptions{
		MaxTokens:   2000,
		Temperature: 0.3,
		SystemPrompt: "You are an expert at identifying BUSINESS domains using data mesh principles. A domain is a BUSINESS CAPABILITY owned by a business team, NOT a data type, technical concern, or infrastructure component. Focus on business value and organizational ownership. Examples: Financial Management, Legal & Compliance, Healthcare Services, Sales & Marketing.",
	})
	if err != nil {
		return nil, nil, 1, 0, err
	}

	// Parse response
	var result struct {
		Domains             []Domain `json:"domains"`
		OverallKeyConcepts  []string `json:"overall_key_concepts"`
	}

	if err := b.extractJSON(response, &result); err != nil {
		return nil, nil, 1, len(response), err
	}

	return result.Domains, result.OverallKeyConcepts, 1, len(response), nil
}

// defineEntityTypes asks LLM to define entity types and extraction rules
func (b *OntologyBuilder) defineEntityTypes(ctx context.Context, samples *sampler.SamplingResult, topEntities []sampler.EntityFrequency, domains []Domain) ([]ElementEntityMapping, int, int, error) {
	sampleTexts := b.prepareSampleTexts(samples.Samples, 15)

	// Format top entities
	entityList := make([]string, 0, len(topEntities))
	for i, e := range topEntities {
		if i >= 20 { // Limit to top 20
			break
		}
		entityList = append(entityList, fmt.Sprintf("%s (%d occurrences)", e.Entity, e.Count))
	}

	// Format domains
	domainList := make([]string, len(domains))
	for i, d := range domains {
		domainList[i] = fmt.Sprintf("- **%s**: %s (Owner: %s)", d.Name, d.Description, d.Owner)
	}

	prompt := fmt.Sprintf(`Analyze these document samples and define entity types to extract. For each entity type, discover multiple extraction patterns AND assign to the appropriate domain.

## DISCOVERED DOMAINS

%s

## DOMAIN ASSIGNMENT

**CRITICAL**: Every entity mapping MUST be assigned to ONE domain:
- Choose the domain where this entity type NATURALLY BELONGS
- Consider: Which organizational owner manages this data?
- Examples:
  * "revenue", "EBITDA" → financial domain (CFO owns)
  * "registered_entity", "jurisdiction" → legal domain (Legal owns)
  * "product_code", "SKU" → product domain (Product team owns)

Top entities found:
%s

Sample texts:
%s

## CONFIDENCE MODEL

**IMPORTANT**: Confidence represents CONTEXT QUALITY (not pattern matching certainty):
- 0.95 = Structured context (tables, forms, metadata) - highly reliable extraction context
- 0.85 = Semi-structured context (lists, headings) - reliable context
- 0.75 = Narrative context (paragraphs, sentences) - less predictable context
- 0.65 = Unstructured context (mixed/unknown) - least reliable context

All extraction patterns are BINARY (TRUE/FALSE match). Confidence is assigned at the mapping level based on WHERE entities are found, not HOW they match.

## EXTRACTION PATTERNS

For each entity type, analyze the samples to discover MULTIPLE extraction patterns:

1. **KEYWORD PATTERNS** - For named entities:
   - Automatically include aliases for top entities
   - Example: Microsoft → ["Microsoft", "MSFT", "MS", "Microsoft Corporation"]
   - Use for: Company names, person names, product names
   - Pattern returns: TRUE if keyword found, FALSE otherwise

2. **REGEX PATTERNS** - For structured/formatted entities:
   - Look for repeating formats in samples
   - Examples:
     * Email: \b[A-Za-z0-9._%%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b
     * Phone: \b\d{3}[-.]?\d{3}[-.]?\d{4}\b
     * Stock ticker: \b[A-Z]{2,5}\b
     * Product code: \b[A-Z]{2,3}-\d{4,6}\b
     * Date: \b\d{1,2}/\d{1,2}/\d{4}\b
   - Use for: IDs, codes, contact info, dates, measurements
   - Pattern returns: TRUE if regex matches, FALSE otherwise

3. **TEXT SIMILARITY PATTERNS** - For context-based entities:
   - Find typical phrases/contexts where entities appear
   - Example reference texts:
     * "the CEO stated that" → executive mentions
     * "quarterly revenue of" → financial metrics
     * "located in the city of" → geographic locations
   - Include reference_text and similarity_threshold (0.6-0.8)
   - Use for: Contextual entity recognition
   - Pattern returns: TRUE if similarity >= threshold, FALSE otherwise

4. **METADATA FIELD PATTERNS** - For structured metadata:
   - Extract from document metadata fields
   - Example: "author.name", "company_info.ticker"
   - Use for: Structured document properties
   - Pattern returns: TRUE if field exists, FALSE otherwise

## MULTIPLE MAPPINGS FOR SAME ENTITY TYPE

You can create MULTIPLE mappings for the same entity_type with different confidence levels:
- One mapping for table contexts (confidence: 0.95)
- Another mapping for paragraph contexts (confidence: 0.75)

When multiple mappings extract same entity → HIGHEST confidence wins.

Return JSON array with DOMAIN FIELD (NO confidence field in extraction_rules):
[
  {
    "entity_type": "organization",
    "domain": "legal",
    "description": "Companies and organizations from structured contexts",
    "element_types": ["table_cell", "heading"],
    "confidence": 0.95,
    "extraction_rules": [
      {
        "type": "keyword_match",
        "keywords": ["Microsoft", "MSFT", "MS", "Microsoft Corporation"]
      },
      {
        "type": "regex_pattern",
        "pattern": "\\b[A-Z][a-z]+ (Inc|Corp|LLC|Ltd)\\.?\\b"
      }
    ]
  },
  {
    "entity_type": "organization",
    "domain": "legal",
    "description": "Companies and organizations from narrative text",
    "element_types": ["paragraph"],
    "confidence": 0.75,
    "extraction_rules": [
      {
        "type": "keyword_match",
        "keywords": ["Microsoft", "MSFT", "MS", "Microsoft Corporation"]
      }
    ]
  },
  {
    "entity_type": "email",
    "domain": "legal",
    "description": "Email addresses for legal entities",
    "element_types": ["paragraph", "table_cell"],
    "confidence": 0.95,
    "extraction_rules": [
      {
        "type": "regex_pattern",
        "pattern": "\\b[A-Za-z0-9._%%+-]+@[A-Za-z0-9.-]+\\.[A-Z|a-z]{2,}\\b"
      }
    ]
  },
  {
    "entity_type": "financial_metric",
    "domain": "financial",
    "description": "Financial metrics from structured data",
    "element_types": ["table_cell"],
    "confidence": 0.95,
    "extraction_rules": [
      {
        "type": "keyword_match",
        "keywords": ["revenue", "profit", "EBITDA", "earnings"]
      }
    ]
  },
  {
    "entity_type": "financial_metric",
    "domain": "financial",
    "description": "Financial metrics from narrative contexts",
    "element_types": ["paragraph"],
    "confidence": 0.75,
    "extraction_rules": [
      {
        "type": "text_similarity",
        "reference_text": "quarterly revenue of $500 million",
        "similarity_threshold": 0.7
      }
    ]
  }
]`, strings.Join(domainList, "\n"), strings.Join(entityList, "\n"), sampleTexts)

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

Analyze these sample texts to discover relationship patterns between entities. For each relationship, identify the PATTERNS that signal it in text.

Sample texts:
%s

## CONFIDENCE MODEL

**IMPORTANT**: Confidence represents PATTERN RELIABILITY (how confident we are the pattern indicates this relationship):
- 0.95 = Explicit, unambiguous pattern (e.g., "{person} is CEO of {organization}")
- 0.85 = Strong signal pattern (e.g., regex with structure, clear templates)
- 0.75 = Moderate signal pattern (e.g., proximity with signal words)
- 0.65 = Weak signal pattern (e.g., cooccurrence without clear signals)

All extraction patterns are BINARY (TRUE/FALSE match). Confidence is assigned at the RULE level (not pattern level) based on pattern reliability.

## EXTRACTION PATTERNS

For each relationship type, discover MULTIPLE extraction patterns:

1. **TEXT TEMPLATE PATTERNS** - Textual patterns with entity placeholders:
   - Example: "{person} is CEO of {organization}"
   - Example: "{organization} acquired {organization} for ${amount}"
   - Example: "{person} works at {organization} as {role}"
   - Use {entity_type} placeholders for entities
   - Pattern returns: TRUE if template matches, FALSE otherwise

2. **PROXIMITY PATTERNS** - Entities near each other with signal words:
   - Signal words indicating relationship (e.g., ["CEO", "president", "director"] for leadership)
   - Max distance between entities (in tokens)
   - Direction: "forward", "backward", "bidirectional"
   - Example: person within 10 tokens of organization with signals ["CEO", "president"]
   - Pattern returns: TRUE if entities + signals found within distance, FALSE otherwise

3. **REGEX PATTERNS** - Complex patterns with named entity groups:
   - Use named groups: (?P<person>...) and (?P<organization>...)
   - Example: "(?P<person>[A-Z][a-z]+ [A-Z][a-z]+),\\s+CEO\\s+of\\s+(?P<org>[A-Z][^,]+)"
   - Use for highly structured text
   - Pattern returns: TRUE if regex matches, FALSE otherwise

4. **COOCCURRENCE PATTERNS** - Statistical co-occurrence:
   - Entities appear together in same context
   - Context window: "paragraph", "sentence", "element"
   - Required keywords to strengthen confidence
   - Pattern returns: TRUE if entities cooccur in context, FALSE otherwise

## MULTIPLE RULES FOR SAME RELATIONSHIP

You can create MULTIPLE rules for the same relationship name with different confidence levels:
- Explicit template pattern (confidence: 0.95)
- Proximity pattern (confidence: 0.75)

When multiple rules detect same relationship → HIGHEST confidence wins.

Return JSON array (NO confidence in extraction_patterns, confidence at RULE level):
[
  {
    "name": "person_leads_organization",
    "source_entity_type": "person",
    "target_entity_type": "organization",
    "relationship_type": "part_of",
    "description": "Person in leadership role at organization",
    "confidence": 0.95,
    "extraction_patterns": [
      {
        "type": "text_template",
        "template": "{person} is CEO of {organization}",
        "examples": ["John Smith is CEO of Acme Corp", "Jane Doe is CEO of TechCo"]
      },
      {
        "type": "text_template",
        "template": "{person}, CEO of {organization}",
        "examples": ["John Smith, CEO of Acme Corp, announced..."]
      },
      {
        "type": "regex",
        "pattern": "(?P<person>[A-Z][a-z]+ [A-Z][a-z]+),\\s+(?:CEO|President)\\s+of\\s+(?P<organization>[A-Z][^,]+)"
      }
    ]
  },
  {
    "name": "person_near_organization",
    "source_entity_type": "person",
    "target_entity_type": "organization",
    "relationship_type": "related_to",
    "description": "Person associated with organization (lower confidence)",
    "confidence": 0.75,
    "extraction_patterns": [
      {
        "type": "proximity",
        "signal_words": ["CEO", "president", "director", "chief executive", "works at", "employed by"],
        "max_distance": 10,
        "direction": "bidirectional"
      }
    ]
  },
  {
    "name": "company_acquired_company",
    "source_entity_type": "organization",
    "target_entity_type": "organization",
    "relationship_type": "related_to",
    "description": "Organization acquired another organization",
    "confidence": 0.95,
    "extraction_patterns": [
      {
        "type": "text_template",
        "template": "{organization} acquired {organization}"
      },
      {
        "type": "text_template",
        "template": "{organization} purchased {organization}"
      }
    ]
  },
  {
    "name": "company_near_company_acquisition",
    "source_entity_type": "organization",
    "target_entity_type": "organization",
    "relationship_type": "related_to",
    "description": "Potential acquisition relationship (lower confidence)",
    "confidence": 0.80,
    "extraction_patterns": [
      {
        "type": "proximity",
        "signal_words": ["acquired", "purchased", "bought", "acquisition"],
        "max_distance": 15,
        "direction": "forward"
      }
    ]
  }
]`, strings.Join(entityTypes, ", "), sampleTexts)

	response, err := b.llmClient.Complete(ctx, prompt, LLMOptions{
		MaxTokens:   4000,
		Temperature: 0.3,
		SystemPrompt: "You are an expert at relationship extraction and pattern discovery. Analyze text samples to find patterns that signal relationships between entities.",
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
	// Find JSON in response (handle markdown code blocks and preamble text)
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
	} else {
		// No code fence - try to find valid JSON by searching for { or [ and attempting to parse
		// We may encounter false positives like [header] in text, so we try each candidate
		for startIdx := 0; startIdx < len(response); startIdx++ {
			// Find next { or [ character
			jsonStart := -1
			for i := startIdx; i < len(response); i++ {
				if response[i] == '{' || response[i] == '[' {
					jsonStart = i
					break
				}
			}

			if jsonStart == -1 {
				break // No more candidates
			}

			// Find the matching closing brace/bracket
			candidateStr := response[jsonStart:]

			// Attempt to parse incrementally to find the complete JSON
			// This handles cases where there's text after the JSON
			depth := 0
			inString := false
			escaped := false
			jsonEnd := -1

			for i, ch := range candidateStr {
				if escaped {
					escaped = false
					continue
				}

				if ch == '\\' {
					escaped = true
					continue
				}

				if ch == '"' {
					inString = !inString
					continue
				}

				if !inString {
					if ch == '{' || ch == '[' {
						depth++
					} else if ch == '}' || ch == ']' {
						depth--
						if depth == 0 {
							jsonEnd = i + 1
							break
						}
					}
				}
			}

			if jsonEnd != -1 {
				candidate := candidateStr[:jsonEnd]
				// Try to parse this candidate - if it works, we found valid JSON
				var test interface{}
				if err := json.Unmarshal([]byte(candidate), &test); err == nil {
					jsonStr = candidate
					break // Found valid JSON!
				}
				// This candidate didn't parse, try next { or [
				startIdx = jsonStart + 1
			} else {
				// Couldn't find closing brace, try next { or [
				startIdx = jsonStart + 1
			}
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

// configureResolverFromStorage opens the analytics storage and configures the sampler's content resolver
func configureResolverFromStorage(samp *sampler.Sampler, parquetPath string) error {
	// Create analytics storage to get the resolver
	storageConfig := map[string]interface{}{
		"path": parquetPath,
	}

	storage, err := analytics.NewHiveParquetStorage(storageConfig)
	if err != nil {
		return fmt.Errorf("failed to create analytics storage: %w", err)
	}
	defer storage.Close()

	// Get resolver from storage
	resolverInterface := storage.GetContentResolver()
	if resolverInterface == nil {
		return fmt.Errorf("storage did not provide a content resolver")
	}

	// Type assert to ContentResolver
	contentResolver, ok := resolverInterface.(resolver.ContentResolver)
	if !ok {
		return fmt.Errorf("storage resolver has wrong type: %T", resolverInterface)
	}

	// Configure sampler with the resolver
	samp.SetResolver(contentResolver)

	return nil
}
