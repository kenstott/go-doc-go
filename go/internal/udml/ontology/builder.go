package ontology

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kennethstott/doculyzer-go-conversion/internal/analytics"
	"github.com/kennethstott/doculyzer-go-conversion/internal/embeddings"
	"github.com/kennethstott/doculyzer-go-conversion/internal/resolver"
	"github.com/kennethstott/doculyzer-go-conversion/internal/udml"
	"github.com/kennethstott/doculyzer-go-conversion/internal/udml/ontology/mcp"
	"github.com/kennethstott/doculyzer-go-conversion/internal/udml/sampler"
	"gopkg.in/yaml.v3"
)

//go:embed catalogs/**/*.yaml
var embeddedCatalogs embed.FS

// min returns the smaller of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// OntologyBuilder orchestrates the automatic ontology schema creation process
type OntologyBuilder struct {
	sampler          *sampler.Sampler
	llmClient        LLMClient
	config           BuilderConfig
	substantiveTypes []string                    // Element types discovered from corpus
	mcpServer        *mcp.OntologyCorpusExplorer // Optional: MCP server for LLM corpus exploration
	mcpStorage       analytics.Storage           // Storage backend for MCP server
}

// BuilderConfig configures the ontology building process
type BuilderConfig struct {
	// Sampling
	SampleSize          int     // Number of elements to sample
	StoragePath         string  // Path to UDML storage (format-agnostic)
	DiversityThreshold  float64 // Cosine similarity threshold for diversity filtering (0.0-1.0, lower = more diverse)

	// LLM
	LLMProvider   string // "anthropic", "openai", etc.
	LLMModel      string // Model name
	LLMAPIKey     string // API key
	LLMMaxTokens  int    // Max tokens for LLM responses

	// MCP Configuration (optional - enables interactive corpus exploration)
	EnableMCP      bool   // Enable MCP server for LLM tool calling
	EmbeddingModel string // Embedding model for semantic search (e.g., "all-MiniLM-L6-v2")

	// Analysis
	TopEntityCount      int     // Top N entities to analyze for aliases
	MinEntityFrequency  int     // Minimum frequency for entity consideration
	ConfidenceThreshold float64 // Minimum confidence for rules

	// Output
	SchemaName    string // Name for the generated schema
	SchemaVersion string // Version
	Domain        string // Domain name

	// Catalog Configuration
	ExternalCatalogPath string // Optional: path to external catalog directory to extend embedded catalogs
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
	MaxTokens    int
	Temperature  float64
	SystemPrompt string
}

// MCPToolDefinition describes an MCP tool to the LLM
type MCPToolDefinition struct {
	Name        string                    // Tool name (e.g., "search_corpus")
	Description string                    // What the tool does
	Parameters  map[string]ParameterDef   // Tool parameters
}

// ParameterDef describes a tool parameter
type ParameterDef struct {
	Type        string   // "string", "number", "boolean", "array"
	Description string   // Parameter description
	Required    bool     // Is this parameter required?
	Enum        []string // Allowed values (optional)
	Items       *ItemDef // For array types (optional)
}

// ItemDef describes array item types
type ItemDef struct {
	Type string // "string", "number", "boolean"
}

// MCPToolCall represents a tool call request from the LLM
type MCPToolCall struct {
	ID    string                 // Unique tool call ID
	Name  string                 // Tool name
	Input map[string]interface{} // Tool arguments
}

// MCPToolResult represents the result of a tool call
type MCPToolResult struct {
	ToolCallID string // ID matching the tool call
	Content    string // Tool result content (JSON or text)
	IsError    bool   // Whether this is an error result
}

// MCPCapableLLMClient extends LLMClient with MCP tool calling capabilities
// This interface enables the LLM to interactively explore the corpus via MCP tools
type MCPCapableLLMClient interface {
	LLMClient

	// CompleteWithTools sends a prompt to the LLM with MCP tool definitions
	// The LLM can choose to call tools during generation, and the implementation
	// handles the tool call execution loop, returning the final response
	CompleteWithTools(ctx context.Context, prompt string, tools []MCPToolDefinition, options LLMOptions) (string, error)
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
	// Load authoritative UDML element types from taxonomy
	substantiveTypes, err := loadUDMLElementTypes()
	if err != nil {
		return nil, fmt.Errorf("failed to load UDML element taxonomy: %w", err)
	}

	// Determine diversity threshold - use config value or default
	diversityThreshold := config.DiversityThreshold
	if diversityThreshold == 0.0 {
		diversityThreshold = 0.70 // Default: filter out samples with >70% similarity (moderate diversity, better coverage)
	}
	fmt.Printf("DEBUG: BuilderConfig.DiversityThreshold = %.6f, using = %.6f\n", config.DiversityThreshold, diversityThreshold)

	samplerConfig := sampler.SamplerConfig{
		ParquetPath:         config.StoragePath,
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
	if err := configureResolverFromStorage(samp, config.StoragePath); err != nil {
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
		config.LLMMaxTokens = 32000
	}
	if config.SchemaVersion == "" {
		config.SchemaVersion = "1.0.0"
	}
	if config.EmbeddingModel == "" {
		config.EmbeddingModel = "all-MiniLM-L6-v2" // Default embedding model
	}

	// Create MCP server if enabled
	var mcpServer *mcp.OntologyCorpusExplorer
	var mcpStorage analytics.Storage

	if config.EnableMCP {
		fmt.Println("✓ MCP enabled - creating corpus exploration server...")

		// Create analytics storage for MCP server (supports temporal filtering via filters parameter)
		storageConfig := map[string]interface{}{
			"path": config.StoragePath,
		}

		mcpStorage, err = analytics.NewHiveParquetStorage(storageConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create storage for MCP: %w", err)
		}

		// Create embedding generator for semantic search
		embConfig := embeddings.Config{
			Enabled:  true,
			Provider: "onnx",
			Model:    config.EmbeddingModel,
		}
		embGenerator, err := embeddings.CreateEmbeddingGenerator(embConfig)
		if err != nil {
			mcpStorage.Close()
			return nil, fmt.Errorf("failed to create embedding generator for MCP: %w", err)
		}

		// Create MCP server
		mcpServer, err = mcp.NewOntologyCorpusExplorer(mcpStorage, embGenerator)
		if err != nil {
			mcpStorage.Close()
			return nil, fmt.Errorf("failed to create MCP server: %w", err)
		}

		// Connect MCP server to LLM client if it supports tool calling
		if anthropicClient, ok := llmClient.(*AnthropicClient); ok {
			anthropicClient.SetMCPServer(mcpServer)
			fmt.Println("✓ MCP server connected to Anthropic client")
		} else {
			fmt.Printf("Warning: LLM client (%T) does not support MCP tool calling\n", llmClient)
		}

		fmt.Println("✓ MCP server ready with 6 corpus exploration tools")
	}

	return &OntologyBuilder{
		sampler:          samp,
		llmClient:        llmClient,
		config:           config,
		substantiveTypes: substantiveTypes,
		mcpServer:        mcpServer,
		mcpStorage:       mcpStorage,
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

	// Step 2: Define entity types and extraction rules (multi-step: universal + domain-specific)
	entityMappings, calls2, tokens2, err := b.defineEntityTypesMultiStep(ctx, samples, topEntities, domains)
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

	// Merge global catalog into schema (adds 37 global entity types + relationships)
	if err := b.mergeGlobalCatalog(schema); err != nil {
		return nil, llmCalls, totalTokens, fmt.Errorf("failed to merge global catalog: %w", err)
	}

	// Merge domain-specific catalogs for selected domains
	for _, domain := range domains {
		if err := b.mergeDomainCatalog(schema, domain.Name); err != nil {
			fmt.Printf("Warning: Failed to merge catalog for domain '%s': %v\n", domain.Name, err)
			// Don't fail - just warn and continue
		}
	}

	return schema, llmCalls, totalTokens, nil
}

// GlobalCatalogSchema represents the structure of a global catalog YAML file
type GlobalCatalogSchema struct {
	Domain       string                          `yaml:"domain"`
	Description  string                          `yaml:"description"`
	Subdomains   []string                        `yaml:"subdomains,omitempty"`
	EntityTypes  []GlobalEntityTypeTemplate      `yaml:"entity_types"`
	Relationships []GlobalRelationshipTemplate   `yaml:"relationships,omitempty"`
}

// GlobalEntityTypeTemplate represents an entity type in the global catalog
type GlobalEntityTypeTemplate struct {
	EntityType   string           `yaml:"entity_type"`
	ParentType   string           `yaml:"parent_type,omitempty"`
	WCategory    string           `yaml:"w_category,omitempty"`
	Domain       string           `yaml:"domain,omitempty"`
	Description  string           `yaml:"description"`
	ElementTypes []string         `yaml:"element_types,omitempty"`
	SampleRules  []ExtractionRule `yaml:"sample_rules,omitempty"`
}

// GlobalRelationshipTemplate represents a relationship template in the global catalog
type GlobalRelationshipTemplate struct {
	Name             string           `yaml:"name"`
	SourceType       string           `yaml:"source_type"`
	TargetType       string           `yaml:"target_type"`
	RelationshipType RelationshipType `yaml:"relationship_type"`
	Description      string           `yaml:"description,omitempty"`
}

// mergeGlobalCatalog merges global domain catalog into generated schema
// Adds 37 global entity types and global relationships to every schema
func (b *OntologyBuilder) mergeGlobalCatalog(schema *OntologySchema) error {
	var globalEntities []ElementEntityMappingConfig
	var globalRels []EntityRelationshipRule

	// Load from embedded catalogs
	fmt.Println("✓ Loading global catalog from embedded files...")
	err := fs.WalkDir(embeddedCatalogs, "catalogs/global", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".yaml") {
			return nil
		}

		// Read embedded file
		data, err := embeddedCatalogs.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read embedded global catalog %s: %w", path, err)
		}

		var catalog GlobalCatalogSchema
		if err := yaml.Unmarshal(data, &catalog); err != nil {
			return fmt.Errorf("failed to parse embedded global catalog %s: %w", path, err)
		}

		// Collect entity types
		for _, entityTemplate := range catalog.EntityTypes {
			globalEntities = append(globalEntities, ElementEntityMappingConfig{
				EntityType:      entityTemplate.EntityType,
				ParentType:      entityTemplate.ParentType,
				Domain:          entityTemplate.Domain,
				WCategory:       entityTemplate.WCategory,
				Description:     entityTemplate.Description,
				ElementTypes:    entityTemplate.ElementTypes,
				Confidence:      0.75, // Default confidence for global types
				ExtractionRules: entityTemplate.SampleRules,
			})
		}

		// Collect relationship rules
		for _, relTemplate := range catalog.Relationships {
			globalRels = append(globalRels, EntityRelationshipRule{
				Name:             relTemplate.Name,
				SourceEntityType: relTemplate.SourceType,
				TargetEntityType: relTemplate.TargetType,
				RelationshipType: relTemplate.RelationshipType,
				Description:      relTemplate.Description,
				Confidence:       0.70, // Default confidence for global relationship patterns
			})
		}

		fmt.Printf("  • Loaded global catalog: %s\n", path)
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to load embedded global catalogs: %w", err)
	}

	// Prepend global entities to schema (global entities come BEFORE domain-specific)
	schema.ElementEntityMappings = append(globalEntities, schema.ElementEntityMappings...)

	// Append global relationships to schema (after domain-specific relationships)
	schema.EntityRelationshipRules = append(schema.EntityRelationshipRules, globalRels...)

	// Add global domain to domains list (prepend so it appears first)
	globalDomain := Domain{
		Name:        "global",
		Description:  "Universal entity types and baseline patterns (37 types across 6 W's)",
		Owner:       "System",
	}
	schema.Domains = append([]Domain{globalDomain}, schema.Domains...)

	fmt.Printf("✓ Merged %d global entities and %d global relationships from embedded catalogs\n",
		len(globalEntities), len(globalRels))

	return nil
}

// mergeDomainCatalog merges a domain-specific catalog into the schema
func (b *OntologyBuilder) mergeDomainCatalog(schema *OntologySchema, domainName string) error {
	fmt.Printf("DEBUG: Attempting to merge catalog for domain '%s'\n", domainName)

	// Try to load from embedded catalogs first
	domainCatalog, err := b.loadDomainCatalogFromEmbedded(domainName)
	if err != nil {
		// Try external catalog path if specified
		if b.config.ExternalCatalogPath != "" {
			domainCatalog, err = b.loadDomainCatalogFromExternal(domainName, b.config.ExternalCatalogPath)
			if err != nil {
				return fmt.Errorf("domain catalog not found in embedded or external locations: %w", err)
			}
		} else {
			return fmt.Errorf("domain catalog not found in embedded catalogs: %w", err)
		}
	}

	// Merge entities from domain catalog
	var domainEntities []ElementEntityMappingConfig
	for _, entityTemplate := range domainCatalog.EntityTypes {
		domainEntities = append(domainEntities, ElementEntityMappingConfig{
			EntityType:      entityTemplate.EntityType,
			ParentType:      entityTemplate.ParentType,
			Domain:          entityTemplate.Domain,
			WCategory:       entityTemplate.WCategory,
			Description:     entityTemplate.Description,
			ElementTypes:    entityTemplate.ElementTypes,
			Confidence:      0.80, // Domain-specific entities get higher confidence than global (0.75)
			ExtractionRules: entityTemplate.SampleRules,
		})
	}

	// Merge relationships from domain catalog
	var domainRels []EntityRelationshipRule
	for _, relTemplate := range domainCatalog.Relationships {
		domainRels = append(domainRels, EntityRelationshipRule{
			Name:             relTemplate.Name,
			SourceEntityType: relTemplate.SourceType,
			TargetEntityType: relTemplate.TargetType,
			RelationshipType: relTemplate.RelationshipType,
			Description:      relTemplate.Description,
			Confidence:       0.75, // Domain-specific relationships
		})
	}

	// Append domain entities after LLM-generated but before they go through ComputeHierarchies
	schema.ElementEntityMappings = append(schema.ElementEntityMappings, domainEntities...)

	// Append domain relationships
	schema.EntityRelationshipRules = append(schema.EntityRelationshipRules, domainRels...)

	fmt.Printf("✓ Merged %d entities and %d relationships from '%s' domain catalog\n",
		len(domainEntities), len(domainRels), domainName)

	return nil
}

// loadDomainCatalogFromEmbedded loads a domain catalog from embedded files
func (b *OntologyBuilder) loadDomainCatalogFromEmbedded(domainName string) (*GlobalCatalogSchema, error) {
	// Search for domain catalog file in embedded filesystem
	var catalogPath string
	err := fs.WalkDir(embeddedCatalogs, "catalogs", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".yaml") {
			return nil
		}
		// Skip global catalogs
		if strings.Contains(path, "global/") {
			return nil
		}

		// Read and check domain name
		data, err := embeddedCatalogs.ReadFile(path)
		if err != nil {
			return nil // Skip on error
		}

		var catalog GlobalCatalogSchema
		if err := yaml.Unmarshal(data, &catalog); err != nil {
			return nil // Skip on parse error
		}

		if catalog.Domain == domainName {
			catalogPath = path
			return fs.SkipAll // Found it, stop walking
		}

		return nil
	})

	if err != nil && err != fs.SkipAll {
		return nil, err
	}

	if catalogPath == "" {
		return nil, fmt.Errorf("domain '%s' not found in embedded catalogs", domainName)
	}

	// Load the catalog
	data, err := embeddedCatalogs.ReadFile(catalogPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded catalog %s: %w", catalogPath, err)
	}

	var catalog GlobalCatalogSchema
	if err := yaml.Unmarshal(data, &catalog); err != nil {
		return nil, fmt.Errorf("failed to parse embedded catalog %s: %w", catalogPath, err)
	}

	fmt.Printf("  • Loaded embedded domain catalog: %s\n", catalogPath)
	return &catalog, nil
}

// loadDomainCatalogFromExternal loads a domain catalog from external filesystem
func (b *OntologyBuilder) loadDomainCatalogFromExternal(domainName string, externalPath string) (*GlobalCatalogSchema, error) {
	// Search for domain catalog file in external directory
	var catalogPath string
	err := filepath.Walk(externalPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue on error
		}
		if info.IsDir() || !strings.HasSuffix(path, ".yaml") {
			return nil
		}
		// Skip global catalogs
		if strings.Contains(path, "global/") {
			return nil
		}

		// Read and check domain name
		data, err := os.ReadFile(path)
		if err != nil {
			return nil // Skip on error
		}

		var catalog GlobalCatalogSchema
		if err := yaml.Unmarshal(data, &catalog); err != nil {
			return nil // Skip on parse error
		}

		if catalog.Domain == domainName {
			catalogPath = path
			return filepath.SkipAll // Found it, stop walking
		}

		return nil
	})

	if err != nil && err != filepath.SkipAll {
		return nil, err
	}

	if catalogPath == "" {
		return nil, fmt.Errorf("domain '%s' not found in external catalogs at %s", domainName, externalPath)
	}

	// Load the catalog
	data, err := os.ReadFile(catalogPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read external catalog %s: %w", catalogPath, err)
	}

	var catalog GlobalCatalogSchema
	if err := yaml.Unmarshal(data, &catalog); err != nil {
		return nil, fmt.Errorf("failed to parse external catalog %s: %w", catalogPath, err)
	}

	fmt.Printf("  • Loaded external domain catalog: %s\n", catalogPath)
	return &catalog, nil
}

// loadPredefinedDomains loads domain names and descriptions from catalog YAML files
// Returns error if catalog not found - no fallbacks
func loadPredefinedDomains() (map[string]string, error) {
	return loadPredefinedDomainsWithExternal("")
}

// loadPredefinedDomainsWithExternal loads domain catalogs from embedded files and optional external directory
func loadPredefinedDomainsWithExternal(externalCatalogPath string) (map[string]string, error) {
	fmt.Println("DEBUG: loadPredefinedDomainsWithExternal() called")
	domains := make(map[string]string) // map[domain_name]description

	// Load embedded catalogs (shipped with binary)
	fmt.Println("✓ Loading embedded domain catalogs...")
	err := fs.WalkDir(embeddedCatalogs, "catalogs", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".yaml") {
			return nil
		}

		// Skip global catalog (handled separately)
		if strings.Contains(path, "global/") {
			return nil
		}

		// Read embedded file
		data, err := embeddedCatalogs.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read embedded catalog %s: %w", path, err)
		}

		var catalog struct {
			Domain      string `yaml:"domain"`
			Description string `yaml:"description"`
		}

		if err := yaml.Unmarshal(data, &catalog); err != nil {
			return fmt.Errorf("failed to parse embedded catalog %s: %w", path, err)
		}

		if catalog.Domain != "" {
			domains[catalog.Domain] = catalog.Description
			fmt.Printf("  • Loaded embedded domain: %s\n", catalog.Domain)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to load embedded catalogs: %w", err)
	}

	embeddedCount := len(domains)
	fmt.Printf("✓ Loaded %d embedded domains\n", embeddedCount)

	// Load external catalogs (if specified)
	if externalCatalogPath != "" {
		fmt.Printf("✓ Loading external domain catalogs from: %s\n", externalCatalogPath)

		if _, err := os.Stat(externalCatalogPath); os.IsNotExist(err) {
			fmt.Printf("Warning: External catalog path does not exist: %s\n", externalCatalogPath)
		} else {
			err := filepath.Walk(externalCatalogPath, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if info.IsDir() || !strings.HasSuffix(path, ".yaml") {
					return nil
				}

				// Skip global catalog
				if strings.Contains(path, "global/") {
					return nil
				}

				// Read YAML file
				data, err := os.ReadFile(path)
				if err != nil {
					return fmt.Errorf("failed to read external catalog %s: %w", path, err)
				}

				var catalog struct {
					Domain      string `yaml:"domain"`
					Description string `yaml:"description"`
				}

				if err := yaml.Unmarshal(data, &catalog); err != nil {
					return fmt.Errorf("failed to parse external catalog %s: %w", path, err)
				}

				if catalog.Domain != "" {
					domains[catalog.Domain] = catalog.Description
					fmt.Printf("  • Loaded external domain: %s (overrides embedded if duplicate)\n", catalog.Domain)
				}

				return nil
			})

			if err != nil {
				return nil, fmt.Errorf("failed to load external catalogs: %w", err)
			}

			externalCount := len(domains) - embeddedCount
			if externalCount > 0 {
				fmt.Printf("✓ Loaded %d additional external domains\n", externalCount)
			}
		}
	}

	if len(domains) == 0 {
		return nil, fmt.Errorf("no domain catalogs found (embedded or external)")
	}

	fmt.Printf("✓ Total domains available: %d\n", len(domains))
	return domains, nil
}

// identifyDomains asks LLM to identify domains and key concepts
func (b *OntologyBuilder) identifyDomains(ctx context.Context, samples *sampler.SamplingResult) ([]Domain, []string, int, int, error) {
	fmt.Println("DEBUG: identifyDomains() called - starting domain identification")

	// Prepare sample text
	sampleTexts := b.prepareSampleTexts(samples.Samples, 20)

	// Load predefined domains from catalog (embedded + optional external)
	fmt.Println("DEBUG: About to call loadPredefinedDomainsWithExternal()")
	predefinedDomains, err := loadPredefinedDomainsWithExternal(b.config.ExternalCatalogPath)
	if err != nil {
		return nil, nil, 0, 0, fmt.Errorf("failed to load domain catalog: %w", err)
	}

	// Format predefined domains with descriptions for LLM
	var domainListFormatted strings.Builder
	for name, desc := range predefinedDomains {
		domainListFormatted.WriteString(fmt.Sprintf("  • **%s**: %s\n", name, desc))
	}

	// DEBUG: Log the domain list being sent to LLM
	fmt.Printf("DEBUG: Domain list for LLM (%d domains):\n%s\n", len(predefinedDomains), domainListFormatted.String())

	prompt := fmt.Sprintf(`You MUST select domains from this CLOSED LIST ONLY. You are PROHIBITED from creating new domain names.

## MANDATORY DOMAIN SELECTION - CLOSED LIST

**YOUR ONLY ALLOWED CHOICES:**

%s

## STRICT REQUIREMENTS - NO EXCEPTIONS

**YOU MUST:**
- ✓ Choose ONLY from the exact domain names listed above
- ✓ Use the EXACT spelling shown (e.g., "medical" NOT "medical_healthcare")
- ✓ Copy the domain name EXACTLY as written - no modifications
- ✓ Select 1-5 domains that best match the corpus content

**YOU ARE PROHIBITED FROM:**
- ✗ Creating ANY new domain names not in the list above
- ✗ Modifying domain names (no "medical_practice", "healthcare_delivery", etc.)
- ✗ Combining domains into compound names (no "medical_and_healthcare")
- ✗ Using variations, synonyms, or paraphrases of listed domains
- ✗ Adding prefixes, suffixes, or underscores to listed domains

**EXAMPLES OF PROHIBITED BEHAVIOR:**
- ✗ "medical_healthcare" → WRONG - Use "medical" OR "healthcare"
- ✗ "cognitive_psychology" → WRONG - Use "education" or create nothing
- ✗ "life_sciences_biology" → WRONG - Use "technical" or create nothing
- ✗ "financial_services" → WRONG - Use "financial" EXACTLY

## YOUR TASK

Analyze these document samples and select which domains from the CLOSED LIST above are present.

**CRITICAL: Focus on CONTENT TOPICS, not PRESENTATION FORMAT**

When analyzing the corpus, you MUST:
- ✓ Identify the SUBJECT MATTER being discussed (medicine, mining, finance, etc.)
- ✓ Focus on the substantive content and domain knowledge
- ✓ Look at what the documents are ABOUT

You MUST IGNORE presentation/wrapper metadata:
- ✗ Do NOT classify based on citation formats (ISBN, DOI, references)
- ✗ Do NOT classify based on document structure (TOC, headers, sections)
- ✗ Do NOT classify based on the publishing platform (Wikipedia, encyclopedia, journal)
- ✗ Do NOT classify based on bibliographic metadata (author names, publication dates)

**EXAMPLES:**
- A Wikipedia article about cancer → "medical" (NOT "library_science")
- An SEC filing about a mining company → "mining" (NOT "legal" or "library_science")
- A journal article about chemistry → "science_research" (NOT "library_science")

The domain should reflect the TOPIC/SUBJECT, not how the information is presented or organized.

Sample texts:
%s

## REQUIRED OUTPUT FORMAT

Return ONLY domains from the closed list above:

{
  "domains": [
    {
      "name": "medical",
      "description": "Medical content including clinical practice, medical history, and healthcare",
      "owner": "Medical Affairs Department",
      "key_concepts": ["surgery", "blood transfusion", "anatomy", "pharmacology"]
    }
  ],
  "overall_key_concepts": ["concept1", "concept2", ...]
}

**CRITICAL VALIDATION:**
Before returning your response, verify EVERY domain name appears EXACTLY in the closed list above. If a domain name is NOT in the list, you MUST remove it from your response.`, domainListFormatted.String(), sampleTexts)

	// Call LLM with or without MCP tools
	var response string
	llmOptions := LLMOptions{
		MaxTokens:   b.config.LLMMaxTokens,
		Temperature: 0.3,
		SystemPrompt: "You are an expert at identifying BUSINESS domains using data mesh principles. A domain is a BUSINESS CAPABILITY owned by a business team, NOT a data type, technical concern, or infrastructure component. Focus on business value and organizational ownership. Examples: Financial Management, Legal & Compliance, Healthcare Services, Sales & Marketing.",
	}

	if b.config.EnableMCP && b.mcpServer != nil {
		// Use MCP-capable client with tools
		mcpClient, ok := b.llmClient.(MCPCapableLLMClient)
		if !ok {
			fmt.Printf("Warning: MCP enabled but LLM client (%T) does not support tool calling\n", b.llmClient)
			response, err = b.llmClient.Complete(ctx, prompt, llmOptions)
		} else {
			tools := b.getMCPToolDefinitions()
			fmt.Printf("✓ Calling LLM with %d MCP tools for domain identification\n", len(tools))
			response, err = mcpClient.CompleteWithTools(ctx, prompt, tools, llmOptions)
		}
	} else {
		response, err = b.llmClient.Complete(ctx, prompt, llmOptions)
	}

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
func (b *OntologyBuilder) defineEntityTypes(ctx context.Context, samples *sampler.SamplingResult, topEntities []sampler.EntityFrequency, domains []Domain) ([]ElementEntityMappingConfig, int, int, error) {
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

	// Format element types CLOSED LIST
	elementTypeList := make([]string, len(b.substantiveTypes))
	for i, et := range b.substantiveTypes {
		elementTypeList[i] = fmt.Sprintf("  • **%s**", et)
	}

	// DEBUG: Log the element type list being sent to LLM
	fmt.Printf("DEBUG: Element type CLOSED LIST for LLM (%d types):\n%s\n", len(b.substantiveTypes), strings.Join(elementTypeList, "\n"))

	prompt := fmt.Sprintf(`You MUST use element types from this CLOSED LIST ONLY. You are PROHIBITED from creating new element type names.

## MANDATORY ELEMENT TYPE SELECTION - CLOSED LIST

**YOUR ONLY ALLOWED ELEMENT TYPES:**

%s

## STRICT REQUIREMENTS - NO EXCEPTIONS

**YOU MUST:**
- ✓ Choose ONLY from the exact element type names listed above
- ✓ Use the EXACT spelling shown (e.g., "header" NOT "heading")
- ✓ Copy the element type name EXACTLY as written - no modifications
- ✓ Only use element types that exist in the actual corpus

**YOU ARE PROHIBITED FROM:**
- ✗ Creating ANY new element type names not in the list above
- ✗ Modifying element type names (no "heading", "table_cell", etc.)
- ✗ Using synonyms or variations (e.g., "heading" instead of "header")
- ✗ Using "metadata" as an element type (it doesn't exist in this corpus)

**EXAMPLES OF PROHIBITED BEHAVIOR:**
- ✗ "heading" → WRONG - Use "header" EXACTLY
- ✗ "table_cell" → WRONG - This element type does NOT exist in corpus
- ✗ "metadata" → WRONG - This element type does NOT exist in corpus

**CRITICAL VALIDATION:**
Before returning your response, verify EVERY element_types array contains ONLY element types from the closed list above. If an element type is NOT in the list, you MUST remove it from your response.

================================================================================

## GLOBAL ENTITY TYPES

**IMPORTANT:**

Global entity types (person, organization, location, date, event, etc.) are automatically added to all generated schemas from the global domain catalog. You do NOT need to include them in your response.

**YOUR FOCUS:**
- Create domain-specific entity types relevant to the corpus content
- **LIBERALLY USE parent_type** to inherit from global types - this is STRONGLY ENCOURAGED
- Almost every entity type you create should extend a global type
- Use parent_type: "global.{type}" whenever a global type exists that relates to your entity

**WHEN TO USE parent_type (USE THIS LIBERALLY):**

✓ **Person-like entities** → parent_type: "global.person"
  - physician, patient, scientist, author, executive, employee, customer, etc.

✓ **Organization-like entities** → parent_type: "global.organization"
  - hospital, university, pharmaceutical_company, clinic, research_lab, etc.

✓ **Location-like entities** → parent_type: "global.location"
  - clinic, warehouse, datacenter, facility, office, branch, etc.

✓ **Time-like entities** → parent_type: "global.date" or "global.event"
  - deadline, milestone, appointment, transaction, incident, etc.

✓ **Document-like entities** → parent_type: "global.document"
  - report, patent, article, specification, contract, etc.

✓ **Product-like entities** → parent_type: "global.product"
  - medication, device, software, equipment, chemical, etc.

**EXAMPLES:**
- Create "physician" with parent_type: "global.person" for medical domain
- Create "pharmaceutical_company" with parent_type: "global.organization" for medical domain
- Create "hospital" with parent_type: "global.location" for medical domain
- Create "clinical_trial" with parent_type: "global.event" for medical domain
- Create "medication" with parent_type: "global.product" for medical domain

**WHY THIS MATTERS:**
- Inheritance allows specialized extraction rules while maintaining baseline patterns
- Enables cross-domain entity matching and relationship extraction
- Provides semantic hierarchy for better entity resolution

================================================================================

## ENTITY TYPE DESIGN - PREFER UNIVERSAL TYPES

**IMPORTANT**: When defining entity types, strongly prefer these UNIVERSAL entity types:

**UNIVERSAL ENTITY TYPES (USE THESE FIRST):**
- **person** - Individual people (names, roles, identities)
- **organization** - Companies, institutions, groups, agencies
- **location** - Geographic locations (cities, countries, addresses, regions)
- **date** - Temporal references (dates, times, periods, durations)
- **event** - Named events (meetings, conferences, incidents, milestones)
- **concept** - Abstract ideas, theories, principles, methodologies
- **product** - Products, services, brands, models
- **technology** - Technologies, tools, systems, platforms

**DESIGN GUIDELINES:**
1. **Start with universal types** - Ask yourself: Can this entity be classified as person, organization, location, date, event, concept, product, or technology?
2. **Only create domain-specific types when necessary** - Create custom entity types (e.g., "medical_procedure", "financial_metric") ONLY when the universal types don't capture the domain-specific semantics
3. **Use domain assignment for specialization** - Assign universal types to appropriate domains to capture context (e.g., "organization" in "medical" domain for hospitals, "organization" in "financial" domain for banks)
4. **Combine universal types with attributes** - Use the "attributes" field to add domain-specific details rather than creating new entity types

**CRITICAL: DO NOT USE STRUCTURAL ELEMENT TYPES AS ENTITY TYPES**

Structural/navigational element types describe document structure, NOT business entities. You MUST NOT create entity mappings for these types:

**ABSOLUTELY PROHIBITED ENTITY TYPE NAMES:**
- ✗ **hyperlink** - NEVER use "hyperlink" as entity_type (it's structural navigation)
- ✗ **heading** / **header** - Document structure (NOT entity types)
- ✗ **footer** - Document structure (NOT entity type)
- ✗ **reference** - Use "citation" or "bibliography_entry" instead if needed
- ✗ **bookmark** - Navigation aid (NOT entity type)
- ✗ **annotation** / **comment** - Metadata (NOT entity types)
- ✗ **tag** / **keyword** - Classification metadata (NOT entity types)
- ✗ **div** / **span** / **section** - HTML/layout containers (NOT entity types)
- ✗ **table** / **table_row** / **table_cell** / **list** / **list_item** - Structural containers (NOT entity types)
- ✗ **code_block** / **code_function** / **code_data** - Code structure (NOT entity types)

**WHY THIS MATTERS:**
- Element types (heading, hyperlink, div) describe document STRUCTURE
- Entity types (person, organization, concept) describe BUSINESS CONCEPTS
- NEVER use an element type name as an entity type name

**EXAMPLES OF VIOLATIONS (DO NOT DO THIS):**

Example 1: {"entity_type": "hyperlink", "element_types": ["div", "paragraph"]}
❌ WRONG - hyperlink is an element type, not an entity

Example 2: {"entity_type": "heading", "element_types": ["paragraph"]}
❌ WRONG - heading is an element type, not an entity

**CORRECT APPROACH:**

Example: {"entity_type": "url", "element_types": ["hyperlink", "paragraph"], "extraction_rules": [{"type": "regex_pattern", "pattern": "https?://..."}]}
✅ CORRECT - if you need to extract URLs/links as entities, hyperlink is used correctly here (as element_type)

- Extract entities FROM the content of structural elements
- Don't treat the structural element itself as an entity type
- Example: A hyperlink contains a URL → Extract as entity_type "url", NOT "hyperlink"
- Example: A heading contains "Dr. John Smith" → Extract as entity_type "person", NOT "heading"

**EXAMPLES OF GOOD DESIGN:**
✓ Use "person" assigned to "medical" domain for doctors, patients, medical staff
✓ Use "organization" assigned to "medical" domain for hospitals, clinics, pharmaceutical companies
✓ Use "concept" assigned to "medical" domain for diseases, conditions, treatments
✓ Use "product" assigned to "medical" domain for drugs, medical devices, equipment
✓ Use "event" assigned to "financial" domain for quarterly earnings, mergers, IPOs
✓ Use "location" with attributes for specific location types (warehouse, datacenter, store)

**EXAMPLES OF WHEN TO CREATE CUSTOM TYPES:**
✓ Create "medical_procedure" when "event" or "concept" don't capture the procedural nature
✓ Create "financial_metric" when "concept" doesn't capture the quantitative measurement aspect
✓ Create "legal_statute" when "concept" or "document" don't capture the regulatory nature
✓ Create "chemical_compound" when "product" or "concept" don't capture the scientific structure

**ENTITY TYPE SPECIFICITY RULES:**

Use the MOST SPECIFIC entity type that accurately describes the entity:
- ✓ Prefer concrete types: chemical_compound, research_methodology, publication, medication, technique, process
- ⚠ Use "concept" ONLY as last resort for genuinely abstract ideas/principles/theoretical frameworks
- ✗ Do NOT use "concept" for physical things, concrete processes, or identifiable objects

**Examples of GOOD "concept" usage (genuinely abstract):**
✓ "net zero" → concept (abstract goal/target)
✓ "circular economy" → concept (theoretical framework)
✓ "patient-centered care" → concept (abstract principle)
✓ "sustainability" → concept (abstract idea)

**Examples of BAD "concept" usage (be more specific):**
✗ "DNA barcoding" → research_methodology or technique (NOT concept)
✗ "calcium carbonate" → chemical_compound (NOT concept)
✗ "marble finishing" → process or technique (NOT concept)
✗ "Journal of Medicine" → publication (NOT concept)
✗ "aspirin" → medication or chemical_compound (NOT concept)
✗ "recycling" → process (NOT concept)

================================================================================

Analyze these document samples and define entity types to extract. For each entity type, discover multiple extraction patterns AND assign to the appropriate domain.

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

## EXTRACTION RULE TYPES

There are THREE extraction rule types:

1. **content_extraction** - Extract entities from document content using regex with optional filters
2. **metadata_field** - Extract from document metadata fields
3. **jsonpath_query** - Extract from JSON documents using JSONPath

---

### RULE TYPE 1: content_extraction

Extracts entities from document content using a **required instance_name regex** with optional pre-filters.

**REQUIRED FIELD:**
- instance_name (string, REQUIRED): Regex with named capture group (?P<name>...) that extracts entity text
  - Keywords → Use regex OR: (?P<name>Microsoft|MSFT|MS|Google|GOOG)
  - Patterns → Use regex: (?P<name>\\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\\.[A-Z|a-z]{2,}\\b) (email)

**OPTIONAL FILTERS** (applied BEFORE instance_name for performance):
- pattern (string): Cheap pre-filter regex - only run instance_name if this matches
- proximity_filter (object): Co-occurrence filter - require nearby keywords
- semantic_filter (object): Embedding similarity - validate entity context

**FILTER EXECUTION ORDER** (cheap to expensive):
1. pattern (if specified) - Fast regex pre-filter
2. proximity_filter (if specified) - Co-occurrence check
3. instance_name (REQUIRED) - Extract entity with named capture
4. semantic_filter (if specified) - Embedding similarity validation

**WHEN TO USE EACH FILTER:**

Use pattern when:
- instance_name is complex/expensive
- You want to pre-filter elements before extraction
- Example: pattern="\\b[A-Z]" before instance_name="(?P<name>[A-Z][a-z]+ (?:Inc|Corp|LLC))"

Use proximity_filter when:
- Entity needs contextual keywords nearby
- Example: Extract "aspirin" only when "medication", "drug", or "pharmaceutical" appear within 100 characters

Use semantic_filter when:
- instance_name regex is ambiguous (matches non-entities)
- Need to distinguish entity types semantically
- Example: Disambiguate person names from organization names (both match capitalized words)

## SEMANTIC FILTERING - VALIDATE ENTITY CONTEXT

**IMPORTANT**: When extraction patterns are ambiguous (e.g., broad regex matching capitalized words), use semantic filtering to validate entity matches based on element-level context.

**HOW IT WORKS:**
- Semantic filter validates at the ELEMENT LEVEL (entire element content, not word-level proximity)
- Uses cosine similarity between element embedding and reference concept embeddings
- Acts as AND condition: pattern MUST match AND semantic filter MUST pass
- Gracefully degrades: if embeddings unavailable, filter accepts all matches (pattern-only extraction)

**WHEN TO USE:**
- Ambiguous regex patterns (e.g., patterns matching two capitalized words can match both person names and title-case headings)
- Person vs organization disambiguation (both use similar patterns, anthropomorphism complicates semantics)
- Preventing false positives from formatted text (headings, titles, section names)
- Distinguishing entity types that share structural patterns but differ in semantic context

**STRUCTURE EXAMPLE:**
{
  "type": "content_extraction",
  "instance_name": "(?P<name>\\b[A-Z][a-z]+(?:\\s+[A-Z]\\.)?\\s+[A-Z][a-z]+\\b)",
  "semantic_filter": {
    "reference_concepts": [
      "individual person with biography or credentials",
      "author or creator attribution to individual",
      "personal pronouns (he, she, his, her) referencing the name"
    ],
    "similarity_threshold": 0.65
  }
}

**GUIDELINES:**
1. **Reference Concepts** - Abstract semantic concepts that indicate entity context:
   - For person: "biographical context", "professional role attribution", "personal pronouns"
   - For organization: "corporate actions (announced, acquired)", "business operations", "organizational structure"
   - For location: "geographic references", "spatial relationships", "addresses"
   - Be descriptive but concise (5-15 words per concept)
   - Provide 2-4 reference concepts for redundancy

2. **Similarity Threshold** - Cosine similarity (0.0-1.0):
   - 0.70-0.75: High confidence requirement (use for highly ambiguous patterns)
   - 0.65-0.70: Medium confidence (balanced - use for most cases)
   - 0.60-0.65: Low confidence (more permissive, use when pattern is already strong)
   - Lower threshold = more permissive (more false positives, fewer false negatives)
   - Higher threshold = more restrictive (fewer false positives, more false negatives)

3. **Multiple Rules with Confidence Tiers** - Combine structural signals with semantic filtering:
   - **HIGH CONFIDENCE**: Strong structural signals (titles, suffixes) → NO semantic filter needed
   - **MEDIUM CONFIDENCE**: Moderate patterns → semantic filter with 0.65-0.70 threshold
   - **LOW CONFIDENCE**: Ambiguous patterns → semantic filter with 0.70+ threshold

**EXAMPLE - PERSON EXTRACTION WITH SEMANTIC FILTERING:**
MAPPING 1 (High confidence - title prefix as strong signal):
- entity_type: "person", domain: "medical", confidence: 0.95
- element_types: ["paragraph", "div", "list_item", "table_cell"]
- Pattern: Regex matching Dr/Prof/Mr/Mrs/Ms + Name
- No semantic filter needed (title is strong structural signal)

MAPPING 2 (Lower confidence - ambiguous pattern needs semantic validation):
- entity_type: "person", domain: "medical", confidence: 0.75
- element_types: ["paragraph", "div", "list_item", "table_cell"]
- Pattern: Regex matching First Last or First M. Last
- semantic_filter with reference_concepts: ["individual person with biography", "author attribution", "personal pronouns referencing name"]
- similarity_threshold: 0.65

**EXAMPLE - ORGANIZATION EXTRACTION WITH AMBIGUITY:**
MAPPING 1 (High confidence - legal suffix as strong signal):
- entity_type: "organization", domain: "financial", confidence: 0.95
- element_types: ["paragraph", "div", "list_item", "table_cell"]
- Pattern: Regex matching Name + Inc/Corp/LLC/Ltd suffix
- No semantic filter needed (legal suffix is strong signal)

MAPPING 2 (Lower confidence - ambiguous two-word pattern):
- entity_type: "organization", domain: "financial", confidence: 0.75
- element_types: ["paragraph", "div", "list_item", "table_cell"]
- Pattern: Regex matching two capitalized words (Word Word)
- semantic_filter with reference_concepts: ["corporate actions (announced, acquired)", "business operations (revenue, products)", "organizational structure (headquarters, subsidiary)"]
- similarity_threshold: 0.70 (higher because very ambiguous)

**NOTES:**
- Semantic filter applies to content_extraction rules
- Element must match BOTH instance_name AND semantic context
- Only use semantic filtering when necessary - adds computational cost
- Exclude "header" from element_types to prevent title-case heading false positives
- Test with actual corpus data to tune similarity thresholds

## ENTITY NAME EXTRACTION - INSTANCE VS CATEGORY

**CRITICAL PRINCIPLE**: The extracted entity name must be a specific INSTANCE name, not a category descriptor or fragment.

**What to extract:**
- ✓ Complete instance names that uniquely identify a particular occurrence
- ✓ Proper nouns, identifiers, or distinctive names
- ✓ Full values that distinguish this entity from others of the same type

**What NOT to extract:**
- ✗ Category labels or type descriptors (e.g., "Dr.", "Hospital", "ISBN")
- ✗ Partial fragments of names (e.g., just a first name without last name)
- ✗ Generic descriptors (e.g., "syndrome", "disease", "city" by themselves)

**EXAMPLES ACROSS ENTITY TYPES:**

**Identifiers:**
- ✓ GOOD: "978-1-4939-8933-1" (the actual ISBN value)
- ✗ BAD: "ISBN" (category label, not instance)

**Person:**
- ✓ GOOD: "Jane Smith", "Dr. Robert Johnson"
- ✗ BAD: "Dr." or "Professor" (title fragment only)

**Organization:**
- ✓ GOOD: "Johns Hopkins Hospital", "Microsoft Corporation"
- ✗ BAD: "Hospital" or "University" (type descriptor only)

**Location:**
- ✓ GOOD: "London", "123 Main Street"
- ✗ BAD: "City" or "in" (category/preposition fragment)

**Medical Condition:**
- ✓ GOOD: "Type 1 Diabetes", "Parkinson's Disease"
- ✗ BAD: "syndrome" or "disease" (category fragment only)

**WHY THIS MATTERS:**
Your extraction rules should capture the COMPLETE identifying information, not just parts or labels.
If your regex pattern \\b(?:PMID|DOI|ISBN)\\s*:?\\s*[0-9\\-\\.]+\\b matches "ISBN 978-1-4939-8933-1",
the entity name should be "978-1-4939-8933-1" (the identifier value), NOT "ISBN" (the label).

**instance_name IS REQUIRED:**
All content_extraction rules MUST include an instance_name field with a (?P<name>...) capture group.

**REGEX PATTERNS FOR DIFFERENT NEEDS:**

1. **Keyword OR patterns** (for named entities with aliases):
   - Use regex OR with (?P<name>...) capture
   - Example: (?P<name>Microsoft|MSFT|MS|Google|GOOG|Alphabet)
   - Extracts: "Microsoft", "MSFT", etc.

2. **Structured patterns** (for formatted entities):
   - Use regex with named capture
   - Example: (?P<name>\\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\\.[A-Z|a-z]{2,}\\b) (email)
   - Example: (?P<name>\\b[A-Z][a-z]+ (?:Inc|Corp|LLC|Ltd)\\.?\\b) (company)
   - Extracts: "john@example.com", "Microsoft Corp", etc.

3. **Identifier extraction** (extract value, not label):
   - Use regex to extract specific part
   - Example: (?:PMID|DOI|ISBN)\\s*:?\\s*(?P<name>[0-9\\-\\.]+)
   - From text "ISBN: 978-1-4939-8933-1", extracts: "978-1-4939-8933-1" (NOT "ISBN")

4. **Full content capture** (for broad concepts):
   - Use (?P<name>.+) to capture entire matched text
   - Example: (?P<name>.+) with semantic_filter to validate element context
   - Extracts: Entire element content as entity name

5. **Bounded content** (prevent overly long names):
   - Use (?P<name>.{1,100}) to limit length
   - Prevents excessively long entity names
   - Try specific pattern first, fall back to truncated content

**EXAMPLES:**

Specific entity extraction:
{
  "type": "text_similarity",
  "reference_text": "medical specialty classification",
  "similarity_threshold": 0.7,
  "instance_name": "\\b(?P<name>\\w+(?:\\s+\\w+){0,2}(?:ology|iatry|medicine))\\b"
}
✓ Extracts specialty names like "cardiology", "internal medicine"

Full content capture:
{
  "type": "text_similarity",
  "reference_text": "therapeutic management strategy and treatment approach",
  "similarity_threshold": 0.7,
  "instance_name": "(?P<name>.+)"
}
✓ Uses entire matched text as entity name

Truncated content:
{
  "type": "text_similarity",
  "reference_text": "person with professional credentials",
  "similarity_threshold": 0.7,
  "instance_name": "\\b(?P<name>[A-Z][a-z]+(?:\\s+[A-Z][a-z]+){0,3})\\b|(?P<name>.{1,100})"
}
✓ Tries to extract person name, falls back to first 100 characters

### RULE TYPE 2: metadata_field

Extracts entities from document metadata fields (e.g., author, title, company).

**REQUIRED FIELD:**
- field_path (string): Dot-notation path to metadata field (e.g., "author.name", "company_info.ticker")

**EXAMPLE:**
{
  "type": "metadata_field",
  "field_path": "author.name"
}

### RULE TYPE 3: jsonpath_query

Extracts entities from JSON documents using JSONPath expressions.

**REQUIRED FIELD:**
- jsonpath_expr (string): JSONPath query (e.g., "$.items[*].price", "$.author.name")

**EXAMPLE:**
{
  "type": "jsonpath_query",
  "jsonpath_expr": "$.metadata.company_info.ticker"
}

---

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
        "type": "content_extraction",
        "instance_name": "(?P<name>Microsoft|MSFT|MS|Microsoft Corporation|Google|GOOG|Alphabet)"
      },
      {
        "type": "content_extraction",
        "instance_name": "(?P<name>\\b[A-Z][a-z]+ (?:Inc|Corp|LLC|Ltd)\\.?\\b)"
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
        "type": "content_extraction",
        "instance_name": "(?P<name>Microsoft|MSFT|MS|Microsoft Corporation|Google|GOOG|Alphabet)"
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
        "type": "content_extraction",
        "instance_name": "(?P<name>\\b[A-Za-z0-9._%%+-]+@[A-Za-z0-9.-]+\\.[A-Z|a-z]{2,}\\b)"
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
        "type": "content_extraction",
        "instance_name": "(?P<name>revenue|profit|EBITDA|earnings|sales)"
      }
    ]
  },
  {
    "entity_type": "financial_metric",
    "domain": "financial",
    "description": "Financial metrics from narrative contexts with validation",
    "element_types": ["paragraph"],
    "confidence": 0.75,
    "extraction_rules": [
      {
        "type": "content_extraction",
        "instance_name": "(?P<name>revenue|profit|EBITDA|earnings|sales)",
        "semantic_filter": {
          "reference_concepts": ["quarterly revenue reporting", "financial performance metrics", "earnings statements"],
          "similarity_threshold": 0.7
        }
      }
    ]
  },
  {
    "entity_type": "stock_ticker",
    "domain": "financial",
    "description": "Stock ticker symbols from JSON metadata",
    "element_types": ["paragraph", "table_cell"],
    "confidence": 0.95,
    "extraction_rules": [
      {
        "type": "jsonpath_query",
        "jsonpath_expr": "$.metadata.company_info.ticker"
      }
    ]
  }
]`, strings.Join(elementTypeList, "\n"), strings.Join(domainList, "\n"), strings.Join(entityList, "\n"), sampleTexts)

	// Call LLM with or without MCP tools
	var response string
	var err error
	llmOptions := LLMOptions{
		MaxTokens:   b.config.LLMMaxTokens,
		Temperature: 0.3,
		SystemPrompt: "You are an expert at entity extraction and ontology design.",
	}

	if b.config.EnableMCP && b.mcpServer != nil {
		// Use MCP-capable client with tools
		mcpClient, ok := b.llmClient.(MCPCapableLLMClient)
		if !ok {
			fmt.Printf("Warning: MCP enabled but LLM client (%T) does not support tool calling\n", b.llmClient)
			response, err = b.llmClient.Complete(ctx, prompt, llmOptions)
		} else {
			tools := b.getMCPToolDefinitions()
			fmt.Printf("✓ Calling LLM with %d MCP tools for entity type definition\n", len(tools))
			response, err = mcpClient.CompleteWithTools(ctx, prompt, tools, llmOptions)
		}
	} else {
		response, err = b.llmClient.Complete(ctx, prompt, llmOptions)
	}

	if err != nil {
		return nil, 1, 0, err
	}

	var mappings []ElementEntityMappingConfig
	if err := b.extractJSON(response, &mappings); err != nil {
		return nil, 1, len(response), err
	}

	return mappings, 1, len(response), nil
}

// defineEntityTypesMultiStep generates entity mappings for each domain in a single unified call
func (b *OntologyBuilder) defineEntityTypesMultiStep(ctx context.Context, samples *sampler.SamplingResult, topEntities []sampler.EntityFrequency, domains []Domain) ([]ElementEntityMappingConfig, int, int, error) {
	var allMappings []ElementEntityMappingConfig
	var totalLLMCalls int
	var totalTokens int

	fmt.Println("\n🔹 Generating entity types for each domain...")
	for _, domain := range domains {
		mappings, calls, tokens, err := b.generateAllEntitiesForDomain(ctx, samples, topEntities, domain)
		if err != nil {
			return nil, totalLLMCalls + calls, totalTokens + tokens, fmt.Errorf("failed to generate entities for domain %s: %w", domain.Name, err)
		}
		allMappings = append(allMappings, mappings...)
		totalLLMCalls += calls
		totalTokens += tokens
		fmt.Printf("  ✓ Generated %d entity mappings for domain '%s'\n", len(mappings), domain.Name)
	}

	fmt.Printf("\n  ✅ Total entity mappings generated: %d (from %d LLM calls)\n", len(allMappings), totalLLMCalls)
	return allMappings, totalLLMCalls, totalTokens, nil
}

// generateAllEntitiesForDomain generates all entity types for a domain in a single LLM call
func (b *OntologyBuilder) generateAllEntitiesForDomain(ctx context.Context, samples *sampler.SamplingResult, topEntities []sampler.EntityFrequency, domain Domain) ([]ElementEntityMappingConfig, int, int, error) {
	sampleTexts := b.prepareSampleTexts(samples.Samples, 10)

	// Format element types CLOSED LIST
	elementTypeList := make([]string, len(b.substantiveTypes))
	for i, et := range b.substantiveTypes {
		elementTypeList[i] = fmt.Sprintf("  • **%s**", et)
	}

	// Format top entities for context
	domainEntities := []string{}
	for i, e := range topEntities {
		if i >= 15 {
			break
		}
		domainEntities = append(domainEntities, fmt.Sprintf("%s (%d occurrences)", e.Entity, e.Count))
	}

	// Universal entity template descriptions
	templatesDesc := `
### PERSON
- Description: Individual people including names, roles, identities
- Suggested element types: paragraph, list_item, div, table_cell
- Example patterns:
  - Regex: \b([A-Z][a-z]+(?:\s+[A-Z]\.)?(?:\s+[A-Z][a-z]+)+)\b (capitalized names)
  - Keywords: Dr., Prof., CEO, President, Director
  - Proximity filter: Match person names only when near biographical keywords (born, died, founded, created)

### ORGANIZATION
- Description: Companies, institutions, groups, agencies, foundations
- Suggested element types: paragraph, list_item, div, table_cell
- Example patterns:
  - Regex: \b([A-Z][A-Za-z&\s]+(?:Inc|Corp|LLC|Ltd|Company|Foundation|Institute|University|Hospital|Bank|Group))\b
  - Keywords: company, corporation, organization, institution, foundation
  - Proximity filter: Match org names near keywords like founded, headquartered, acquired

### LOCATION
- Description: Geographic locations including cities, countries, addresses, regions
- Suggested element types: paragraph, list_item, div, table_cell
- Example patterns:
  - Regex: \b([A-Z][a-z]+(?:\s+[A-Z][a-z]+)*,?\s*(?:USA|UK|Canada|France|Germany|China|Japan)?)\b
  - Keywords: city, country, region, state, province, located in
  - Named entity recognition patterns for place names

### DATE
- Description: Temporal references including dates, times, periods, durations
- Suggested element types: paragraph, list_item, div, table_cell
- Example patterns:
  - Regex: \b((?:January|February|March|April|May|June|July|August|September|October|November|December)\s+\d{1,2},?\s+\d{4})\b
  - Regex: \b(\d{1,2}[/-]\d{1,2}[/-]\d{2,4})\b
  - Keywords: date, time, year, month, period
`

	prompt := fmt.Sprintf(`You MUST use element types from this CLOSED LIST ONLY:

## ALLOWED ELEMENT TYPES
%s

## TASK: Discover All Entity Types for Domain "%s"

**Domain Context:**
- Name: %s
- Description: %s
- Owner: %s

**Your Task:**
Analyze corpus samples and identify ALL relevant entities for this domain. This includes:

1. **Common entity types** (person, organization, location, date) - Adapt extraction rules to %s domain context
2. **Domain-specific entity types** - Specialized concepts unique to %s domain

**Common Entity Type Templates:**
%s

**Examples of Good Domain-Specific Types:**
- medical domain: "medical_procedure", "diagnosis", "medication", "symptom"
- banking domain: "financial_metric", "account_type", "transaction_type", "compliance_term"
- government domain: "legislation", "policy", "government_program", "regulation"
- technology domain: "programming_language", "framework", "protocol", "api_endpoint"

**Top Entities Found in Corpus:**
%s

**Domain Samples:**
%s

## OUTPUT REQUIREMENTS

Generate a JSON array of entity mappings. You MUST include:
- ALL 4 common types (person, organization, location, date) adapted to this domain
- 2-5 domain-specific entity types unique to %s domain

**JSON Format:**
[
  {
    "entity_type": "person",
    "domain": "%s",
    "description": "People entities in %s domain context",
    "element_types": ["paragraph", "div", "list_item"],
    "confidence": 0.85,
    "extraction_rules": [
      {
        "type": "content_extraction",
        "instance_name": "(?P<name>...pattern adapted for %s...)"
      }
    ]
  },
  {
    "entity_type": "medical_procedure",
    "domain": "%s",
    "description": "Medical procedures and interventions",
    "element_types": ["paragraph", "list_item"],
    "confidence": 0.80,
    "extraction_rules": [
      {
        "type": "content_extraction",
        "instance_name": "(?P<name>...domain-specific pattern...)",
        "proximity_filter": {
          "keywords": ["procedure", "treatment"],
          "max_distance": 100
        }
      }
    ]
  }
]

**CRITICAL REQUIREMENTS:**
- All mappings MUST be assigned to domain "%s"
- Use ONLY element types from the closed list above
- Include ALL 4 common types (person, organization, location, date)
- Include 2-5 domain-specific types
- Adapt all extraction rules to %s domain context`,
		strings.Join(elementTypeList, "\n"),
		domain.Name,
		domain.Name, domain.Description, domain.Owner,
		domain.Name, domain.Name,
		templatesDesc,
		strings.Join(domainEntities, "\n"),
		sampleTexts,
		domain.Name,
		domain.Name, domain.Name, domain.Name,
		domain.Name,
		domain.Name, domain.Name)

	llmOptions := LLMOptions{
		MaxTokens:   b.config.LLMMaxTokens,
		Temperature:  0.2,
		SystemPrompt: "You are an expert at discovering entity types and creating extraction rules adapted to domain-specific contexts.",
	}

	response, err := b.llmClient.Complete(ctx, prompt, llmOptions)
	if err != nil {
		return nil, 1, 0, err
	}

	var mappings []ElementEntityMappingConfig
	if err := b.extractJSON(response, &mappings); err != nil {
		return nil, 1, len(response), err
	}

	return mappings, 1, len(response), nil
}

// defineRelationshipTypes asks LLM to define relationship types
func (b *OntologyBuilder) defineRelationshipTypes(ctx context.Context, samples *sampler.SamplingResult, entityMappings []ElementEntityMappingConfig) ([]EntityRelationshipRule, int, int, error) {
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

For each relationship type, discover MULTIPLE extraction patterns (aim for 3-6 patterns per rule to maximize coverage):

1. **TEXT TEMPLATE PATTERNS** - Textual patterns with entity placeholders:
   - Example: "{person} is CEO of {organization}"
   - Example: "{organization} acquired {organization} for ${amount}"
   - Example: "{person} works at {organization} as {role}"
   - Use {entity_type} placeholders for entities
   - Pattern returns: TRUE if template matches, FALSE otherwise
   - **IMPORTANT**: Create MULTIPLE templates for different phrasings of the same relationship

2. **PROXIMITY PATTERNS** - Entities near each other with signal words:
   - Signal words indicating relationship (e.g., ["CEO", "president", "director"] for leadership)
   - Max distance between entities (in tokens)
   - Direction: "forward", "backward", "bidirectional"
   - Example: person within 10 tokens of organization with signals ["CEO", "president"]
   - Pattern returns: TRUE if entities + signals found within distance, FALSE otherwise
   - **IMPORTANT**: Include comprehensive lists of signal words (10-20 words per pattern)

3. **REGEX PATTERNS** - Complex patterns with named entity groups:
   - Use named groups: (?P<person>...) and (?P<organization>...)
   - Example: "(?P<person>[A-Z][a-z]+ [A-Z][a-z]+),\\s+CEO\\s+of\\s+(?P<org>[A-Z][^,]+)"
   - Use for highly structured text
   - Pattern returns: TRUE if regex matches, FALSE otherwise
   - **IMPORTANT**: Create multiple regex variants to handle formatting variations

4. **COOCCURRENCE PATTERNS** - Statistical co-occurrence:
   - Entities appear together in same context
   - Context window: "paragraph", "sentence", "element"
   - Required keywords to strengthen confidence
   - Pattern returns: TRUE if entities cooccur in context, FALSE otherwise
   - **IMPORTANT**: Include extensive keyword lists (15-30 keywords per pattern)

## MAXIMIZE RELATIONSHIP DISCOVERY

**CRITICAL INSTRUCTIONS**:
1. Generate AT LEAST 3-6 extraction patterns per relationship rule (more patterns = more matches)
2. For each entity type pair (e.g., person-organization), create 2-4 relationship rules with different confidence levels
3. Use COMPREHENSIVE signal word/keyword lists (aim for 10-30 words per proximity/cooccurrence pattern)
4. Consider synonyms, variations, and domain-specific terminology in all patterns
5. Use entity constraints (source_constraints/target_constraints) when relationships only apply to specific entity subtypes

## MULTIPLE RULES FOR SAME RELATIONSHIP

You SHOULD create MULTIPLE rules for the same relationship type with different confidence levels and pattern types:
- High confidence: Explicit template patterns (confidence: 0.95)
- Medium confidence: Regex and proximity patterns (confidence: 0.80-0.85)
- Lower confidence: Cooccurrence patterns with keywords (confidence: 0.70-0.75)

When multiple rules detect same relationship → HIGHEST confidence wins.

## ENTITY CONSTRAINTS (USE FREQUENTLY)

You should use entity constraints to create specialized relationship rules that only apply to specific entity subtypes:

- source_constraints (object, optional): Filters that source entities must satisfy
- target_constraints (object, optional): Filters that target entities must satisfy

**Constraint Fields** (all optional, applied in order):
1. pattern (string): Pre-filter regex - entity name must match (e.g., "\\b(Dr|Professor)\\b.*" for academics)
2. proximity_filter (object): Co-occurrence filter on entity context
   - required_keywords: Keywords that must appear in entity's context
   - window_size: Context window in tokens
3. instance_name (string): Named capture regex - entity name must match
4. semantic_filter (object): Embedding similarity on entity context
   - query: Semantic query describing the entity subtype
   - similarity_threshold: Minimum similarity (0.0-1.0)

**Example Use Cases**:
- Only link people with academic titles (Dr, Professor) to universities
- Only link Fortune 500 companies to specific business relationships
- Only link capital cities to geopolitical relationships
- Only link medical specialties with specific anatomical structures

**Example with Constraints**:
{
  "name": "academic_affiliated_with_university",
  "source_entity_type": "person",
  "target_entity_type": "organization",
  "relationship_type": "part_of",
  "description": "Academic personnel affiliated with university",
  "confidence": 0.90,
  "source_constraints": {
    "proximity_filter": {
      "required_keywords": ["Dr", "Professor", "PhD", "researcher", "faculty", "academic"],
      "window_size": 50
    }
  },
  "target_constraints": {
    "proximity_filter": {
      "required_keywords": ["University", "College", "Institute", "School"],
      "window_size": 30
    }
  },
  "extraction_patterns": [...]
}

Return JSON array (NO confidence in extraction_patterns, confidence at RULE level):
[
  {
    "name": "person_leads_organization_explicit",
    "source_entity_type": "person",
    "target_entity_type": "organization",
    "relationship_type": "part_of",
    "description": "Person in explicit leadership role at organization (high confidence)",
    "confidence": 0.95,
    "extraction_patterns": [
      {
        "type": "text_template",
        "template": "{person} is CEO of {organization}",
        "examples": ["John Smith is CEO of Acme Corp"]
      },
      {
        "type": "text_template",
        "template": "{person}, CEO of {organization}",
        "examples": ["John Smith, CEO of Acme Corp, announced..."]
      },
      {
        "type": "text_template",
        "template": "{person} serves as {role} at {organization}"
      },
      {
        "type": "regex",
        "pattern": "(?P<person>[A-Z][a-z]+ [A-Z][a-z]+),\\s+(?:CEO|President|Director)\\s+of\\s+(?P<organization>[A-Z][^,]+)"
      },
      {
        "type": "regex",
        "pattern": "(?P<organization>[A-Z][^,]+)'s\\s+(?:CEO|President|Director),?\\s+(?P<person>[A-Z][a-z]+ [A-Z][a-z]+)"
      }
    ]
  },
  {
    "name": "person_leads_organization_proximity",
    "source_entity_type": "person",
    "target_entity_type": "organization",
    "relationship_type": "part_of",
    "description": "Person in leadership role detected via proximity (medium confidence)",
    "confidence": 0.80,
    "extraction_patterns": [
      {
        "type": "proximity",
        "signal_words": ["CEO", "president", "director", "chief executive", "chief operating officer", "COO", "CFO", "chief financial officer", "CTO", "chief technology officer", "chairman", "chairwoman", "managing director", "executive director", "vice president", "VP"],
        "max_distance": 10,
        "direction": "bidirectional"
      },
      {
        "type": "proximity",
        "signal_words": ["leads", "manages", "heads", "oversees", "runs", "founded", "established", "created"],
        "max_distance": 8,
        "direction": "forward"
      }
    ]
  },
  {
    "name": "person_works_at_organization",
    "source_entity_type": "person",
    "target_entity_type": "organization",
    "relationship_type": "part_of",
    "description": "Person employed by organization (medium-low confidence)",
    "confidence": 0.70,
    "extraction_patterns": [
      {
        "type": "proximity",
        "signal_words": ["works at", "employed by", "employee of", "staff member", "team member", "works for", "hired by", "joined", "position at", "role at"],
        "max_distance": 12,
        "direction": "bidirectional"
      },
      {
        "type": "cooccurrence",
        "context_window": "paragraph",
        "required_keywords": ["works", "employed", "employee", "staff", "team", "position", "role", "job", "career", "hired", "joined", "working"]
      }
    ]
  },
  {
    "name": "organization_acquired_organization_explicit",
    "source_entity_type": "organization",
    "target_entity_type": "organization",
    "relationship_type": "related_to",
    "description": "Organization acquired another (high confidence)",
    "confidence": 0.95,
    "extraction_patterns": [
      {
        "type": "text_template",
        "template": "{organization} acquired {organization}"
      },
      {
        "type": "text_template",
        "template": "{organization} purchased {organization}"
      },
      {
        "type": "text_template",
        "template": "{organization} bought {organization}"
      },
      {
        "type": "regex",
        "pattern": "(?P<organization1>[A-Z][A-Za-z0-9 ]+)\\s+(?:acquired|purchased|bought)\\s+(?P<organization2>[A-Z][A-Za-z0-9 ]+)"
      }
    ]
  },
  {
    "name": "organization_acquired_organization_proximity",
    "source_entity_type": "organization",
    "target_entity_type": "organization",
    "relationship_type": "related_to",
    "description": "Potential acquisition detected via proximity (medium confidence)",
    "confidence": 0.80,
    "extraction_patterns": [
      {
        "type": "proximity",
        "signal_words": ["acquired", "purchased", "bought", "acquisition", "purchase", "takeover", "merger", "merged with", "acquired by", "buyout", "deal", "transaction"],
        "max_distance": 15,
        "direction": "bidirectional"
      },
      {
        "type": "cooccurrence",
        "context_window": "sentence",
        "required_keywords": ["acquired", "acquisition", "purchased", "purchase", "bought", "merger", "takeover", "deal", "buyout", "transaction", "agreement", "announced"]
      }
    ]
  }
]

**REMEMBER**:
- Generate 3-6 extraction patterns per rule for maximum coverage
- Create 2-4 relationship rules per entity type pair at different confidence levels
- Use comprehensive signal word lists (10-30 words)
- Use entity constraints when relationships apply only to specific subtypes`, strings.Join(entityTypes, ", "), sampleTexts)

	// Call LLM with or without MCP tools
	var response string
	var err error
	llmOptions := LLMOptions{
		MaxTokens:   b.config.LLMMaxTokens,
		Temperature: 0.3,
		SystemPrompt: "You are an expert at relationship extraction and pattern discovery. Analyze text samples to find patterns that signal relationships between entities.",
	}

	if b.config.EnableMCP && b.mcpServer != nil {
		// Use MCP-capable client with tools
		mcpClient, ok := b.llmClient.(MCPCapableLLMClient)
		if !ok {
			fmt.Printf("Warning: MCP enabled but LLM client (%T) does not support tool calling\n", b.llmClient)
			response, err = b.llmClient.Complete(ctx, prompt, llmOptions)
		} else {
			tools := b.getMCPToolDefinitions()
			fmt.Printf("✓ Calling LLM with %d MCP tools for relationship type discovery\n", len(tools))
			response, err = mcpClient.CompleteWithTools(ctx, prompt, tools, llmOptions)
		}
	} else {
		response, err = b.llmClient.Complete(ctx, prompt, llmOptions)
	}

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
	fmt.Printf("DEBUG extractJSON: Input response length: %d, starts with: %q\n", len(response), response[:min(50, len(response))])

	// Try to extract from ```json code block
	if start := strings.Index(response, "```json"); start != -1 {
		fmt.Printf("DEBUG: Found ```json at position %d\n", start)
		start += 7 // Skip past "```json"
		fmt.Printf("DEBUG: After skipping ```json, position=%d, char=%q\n", start, string(response[start]))
		// Skip any whitespace/newlines after the opening fence
		for start < len(response) && (response[start] == '\n' || response[start] == '\r' || response[start] == ' ' || response[start] == '\t') {
			start++
		}
		fmt.Printf("DEBUG: After skipping whitespace, position=%d, char=%q\n", start, string(response[start]))
		if end := strings.Index(response[start:], "```"); end != -1 {
			jsonStr = response[start : start+end]
			fmt.Printf("DEBUG: Extracted JSON from code block, length=%d\n", len(jsonStr))
		} else {
			// No closing fence - use rest of response after the opening fence
			fmt.Printf("DEBUG: Could not find closing ``` fence, using rest of response\n")
			jsonStr = response[start:]
		}
	} else if start := strings.Index(response, "```"); start != -1 {
		start += 3 // Skip past "```"
		// Skip any whitespace/newlines after the opening fence
		for start < len(response) && (response[start] == '\n' || response[start] == '\r' || response[start] == ' ' || response[start] == '\t') {
			start++
		}
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

	// Debug: show first 200 chars of extracted JSON
	preview := jsonStr
	if len(preview) > 200 {
		preview = preview[:200] + "..."
	}
	fmt.Printf("DEBUG extractJSON: extracted %d chars, first 200:\n%s\n", len(jsonStr), preview)

	if err := json.Unmarshal([]byte(jsonStr), target); err != nil {
		return fmt.Errorf("failed to parse LLM response as JSON: %w\nExtracted JSON (first 500 chars): %s", err, jsonStr[:min(500, len(jsonStr))])
	}

	return nil
}

func (b *OntologyBuilder) log(result *BuildResult, message string) {
	result.AnalysisLog = append(result.AnalysisLog, fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), message))
}

// getMCPToolDefinitions returns tool definitions for LLM if MCP is enabled
func (b *OntologyBuilder) getMCPToolDefinitions() []MCPToolDefinition {
	if !b.config.EnableMCP || b.mcpServer == nil {
		return nil
	}

	// Get raw tool definitions from MCP server
	rawTools := b.mcpServer.GetToolDefinitions()

	// Convert to MCPToolDefinition format
	tools := make([]MCPToolDefinition, 0, len(rawTools))
	for _, rawTool := range rawTools {
		toolMap, ok := rawTool.(map[string]interface{})
		if !ok {
			continue
		}

		name, _ := toolMap["name"].(string)
		description, _ := toolMap["description"].(string)
		paramsMap, _ := toolMap["parameters"].(map[string]interface{})

		// Convert parameters
		params := make(map[string]ParameterDef)
		for paramName, paramValue := range paramsMap {
			paramMap, ok := paramValue.(map[string]interface{})
			if !ok {
				continue
			}

			paramDef := ParameterDef{
				Type:        getString(paramMap, "type"),
				Description: getString(paramMap, "description"),
				Required:    getBool(paramMap, "required"),
			}

			// Handle enum
			if enumVal, ok := paramMap["enum"]; ok {
				if enumSlice, ok := enumVal.([]string); ok {
					paramDef.Enum = enumSlice
				} else if enumIface, ok := enumVal.([]interface{}); ok {
					paramDef.Enum = make([]string, len(enumIface))
					for i, v := range enumIface {
						paramDef.Enum[i], _ = v.(string)
					}
				}
			}

			// Handle items (for array types)
			if itemsVal, ok := paramMap["items"]; ok {
				if itemsMap, ok := itemsVal.(map[string]interface{}); ok {
					paramDef.Items = &ItemDef{
						Type: getString(itemsMap, "type"),
					}
				}
			}

			params[paramName] = paramDef
		}

		tools = append(tools, MCPToolDefinition{
			Name:        name,
			Description: description,
			Parameters:  params,
		})
	}

	return tools
}

// Helper functions for type assertions
func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func getBool(m map[string]interface{}, key string) bool {
	if val, ok := m[key]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return false
}

// Close closes the builder and releases resources
func (b *OntologyBuilder) Close() error {
	var errs []error

	// Close sampler
	if b.sampler != nil {
		if err := b.sampler.Close(); err != nil {
			errs = append(errs, fmt.Errorf("sampler close error: %w", err))
		}
	}

	// Close MCP storage backend
	if b.mcpStorage != nil {
		if err := b.mcpStorage.Close(); err != nil {
			errs = append(errs, fmt.Errorf("MCP storage close error: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("close errors: %v", errs)
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

// loadUDMLElementTypes loads the authoritative UDML element types from SchemaRegistry
func loadUDMLElementTypes() ([]string, error) {
	// Get element types from SchemaRegistry (single source of truth)
	registry := udml.NewSchemaRegistry()
	allTypes := registry.GetRegisteredTypes()

	// Filter out non-substantive types that shouldn't be used for entity extraction
	// These are container/metadata types that don't carry extractable entities
	excluded := map[string]bool{
		"image":      true, // Media - not text content
		"link":       true, // Navigation - not entity content
		"meta":       true, // Metadata - not content
		"property":   true, // Metadata field
		"attribute":  true, // Metadata field
	}

	var substantive []string
	for _, et := range allTypes {
		if !excluded[et] {
			substantive = append(substantive, et)
		}
	}

	fmt.Printf("✓ Loaded %d UDML element types from SchemaRegistry (%d total, %d substantive)\n",
		len(substantive), len(allTypes), len(substantive))

	return substantive, nil
}
