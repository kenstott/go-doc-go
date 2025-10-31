package ontology

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	jsonpatch "github.com/evanphx/json-patch/v5"
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
	SampleSize         int     // Number of elements to sample
	StoragePath        string  // Path to UDML storage (format-agnostic)
	DiversityThreshold float64 // Cosine similarity threshold for diversity filtering (0.0-1.0, lower = more diverse)

	// LLM
	LLMProvider  string // "anthropic", "openai", etc.
	LLMModel     string // Model name
	LLMAPIKey    string // API key
	LLMMaxTokens int    // Max tokens for LLM responses

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

	// Relationship Generation Configuration
	MaxRelationshipsPerDomain int        // Maximum relationships per domain (prevents runaway generation, default: 1000000)
	InterDomainGroups         [][]string // Inter-domain relationship groups (each inner array is analyzed together, default: all domains in one group)

	// Debug Configuration
	DebugMode      bool   // Enable debug mode: preserve raw LLM responses for analysis
	DebugOutputDir string // Directory to store debug outputs (default: derived from output path)
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
	Prefill      string // Optional: Content to prefill in assistant's response (Anthropic-specific)
}

// MCPToolDefinition describes an MCP tool to the LLM
type MCPToolDefinition struct {
	Name        string                  // Tool name (e.g., "search_corpus")
	Description string                  // What the tool does
	Parameters  map[string]ParameterDef // Tool parameters
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
	Schema          *OntologySchema
	DraftSchema     *OntologySchema // Initial automatic draft
	Samples         *sampler.SamplingResult
	TopEntities     []sampler.EntityFrequency
	AnalysisLog     []string // Log of analysis steps
	UserRefinements []string // Log of user changes
	BuildTime       time.Duration
	LLMCallCount    int
	TotalLLMTokens  int
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
		ElementTypes:        substantiveTypes,   // Focus sampling on substantive content (excludes links, images, divs)
		IncludeMetadata:     false,              // Skip metadata (not present in older Parquet schemas)
		IncludeEmbedding:    true,               // Include embedding vectors for cosine similarity diversity
		PreferEmbeddingText: true,               // Use embeddings.text when available (richer context)
		ExcludeContainers:   true,               // Only sample leaf elements with actual content (fallback for non-embedding path)
		MinPerStratum:       5,                  // Ensure minimum representation from each substantive type
		DiversityThreshold:  diversityThreshold, // Cosine similarity threshold for diversity filtering
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
	if config.MaxRelationshipsPerDomain == 0 {
		config.MaxRelationshipsPerDomain = 1000000 // Default: 1M relationships per domain (prevents runaway generation)
	}
	if config.DebugOutputDir == "" && config.DebugMode {
		// Auto-derive debug output directory from storage path
		config.DebugOutputDir = filepath.Join(filepath.Dir(config.StoragePath), "ontology_debug")
	}

	// Enable LLM debug logging if debug mode is enabled
	if config.DebugMode && config.DebugOutputDir != "" {
		if anthropicClient, ok := llmClient.(*AnthropicClient); ok {
			anthropicClient.SetDebugOutputDir(config.DebugOutputDir)
			fmt.Printf("✓ LLM debug logging enabled: %s\n", config.DebugOutputDir)
		}
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
		Domain:                  primaryDomain, // Deprecated: set for backward compatibility
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
	Domain        string                       `yaml:"domain"`
	Description   string                       `yaml:"description"`
	Subdomains    []string                     `yaml:"subdomains,omitempty"`
	EntityTypes   []GlobalEntityTypeTemplate   `yaml:"entity_types"`
	Relationships []GlobalRelationshipTemplate `yaml:"relationships,omitempty"`
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
		Description: "Universal entity types and baseline patterns (37 types across 6 W's)",
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
	// Prepare sample text
	sampleTexts := b.prepareSampleTexts(samples.Samples, 20)

	// Load predefined domains from catalog (embedded + optional external)
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
		MaxTokens:    b.config.LLMMaxTokens,
		Temperature:  0.3,
		SystemPrompt: "You are an expert at identifying BUSINESS domains using data mesh principles.\n\nAlways provide your analysis as structured JSON. You respond exclusively with valid JSON objects, nothing else. No explanations, no preambles, no markdown code fences - only pure JSON.",
		Prefill:      "{",  // Force JSON object format, avoid preamble
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
		Domains            []Domain `json:"domains"`
		OverallKeyConcepts []string `json:"overall_key_concepts"`
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
  - Patterns → Use regex: (?P<name>\\b[A-Za-z0-9._%%+-]+@[A-Za-z0-9.-]+\\.[A-Z|a-z]{2,}\\b) (email)

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
   - Example: (?P<name>\\b[A-Za-z0-9._%%+-]+@[A-Za-z0-9.-]+\\.[A-Z|a-z]{2,}\\b) (email)
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
        "instance_name": "(?P<name>\\b[A-Za-z0-9._%%%%+-]+@[A-Za-z0-9.-]+\\.[A-Z|a-z]{2,}\\b)"
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
		MaxTokens:    b.config.LLMMaxTokens,
		Temperature:  0.3,
		SystemPrompt: "You are a JSON-only API. You respond exclusively with valid JSON arrays, nothing else. No explanations, no preambles, no markdown code fences - only pure JSON.",
		Prefill:      "[",  // Force JSON array format, avoid preamble
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

Generate a JSON array of entity mappings. Include entity types that are:
1. **Clearly observable** in the corpus samples with concrete extraction patterns
2. **High-value** for this domain based on frequency and importance
3. **Well-defined** with distinct boundaries from other entity types

**Entity Type Selection Guidelines:**
- Include common types (person, organization, location, date) ONLY if clearly present in corpus
- Add domain-specific types based on corpus analysis (no artificial limits)
- Quality over quantity: Better to have 5 well-defined types than 15 vague ones
- Each entity type must have at least 2 concrete extraction patterns from corpus samples
- Typical range: 5-15 entity types per domain (let corpus complexity guide you)

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
- Adapt all extraction rules to %s domain context

**ENTITY TYPE UNIQUENESS RULES:**
- Entity type names MUST be unique across ALL domains (global + domain-specific)
- PREFER reusing existing entity types from global domain when the concept is the same
- ONLY create domain-specific entity types when they provide clear additional value
- If creating a domain-specific variant, use qualified naming: {domain}_{entity_type}
  Example: If global "procedure" exists, use "medical_procedure" NOT "procedure"
- Set parent_type field to reference the parent entity (e.g., "parent_type": "global.procedure")
- DO NOT duplicate common entity types (person, organization, location, date, etc.) - these should only exist in global domain`,
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
		MaxTokens:    b.config.LLMMaxTokens,
		Temperature:  0.2,
		SystemPrompt: "You are a JSON-only API. You respond exclusively with valid JSON arrays, nothing else. No explanations, no preambles, no markdown code fences - only pure JSON.",
		Prefill:      "[",  // Force JSON array format, avoid preamble (no trailing whitespace)
	}

	response, err := b.llmClient.Complete(ctx, prompt, llmOptions)
	if err != nil {
		return nil, 1, 0, err
	}

	// Save debug response if debug mode enabled
	if b.config.DebugMode {
		if err := b.saveDebugResponse(fmt.Sprintf("entity_types_domain_%s", domain.Name), response); err != nil {
			// Log but don't fail on debug save errors
			fmt.Printf("Warning: Failed to save debug response for domain %s entity types: %v\n", domain.Name, err)
		}
	}

	var mappings []ElementEntityMappingConfig
	if err := b.extractJSON(response, &mappings); err != nil {
		return nil, 1, len(response), err
	}

	return mappings, 1, len(response), nil
}

// defineRelationshipTypes asks LLM to define relationship types
// ====================================================================
// DOMAIN-SCOPED RELATIONSHIP GENERATION HELPERS
// ====================================================================

// filterEntitiesByDomain returns entity mappings belonging to a specific domain
func filterEntitiesByDomain(allEntities []ElementEntityMappingConfig, domain string) []ElementEntityMappingConfig {
	var filtered []ElementEntityMappingConfig
	for _, e := range allEntities {
		if e.Domain == domain {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

// filterEntitiesToDomains returns entity mappings belonging to a set of domains
func filterEntitiesToDomains(allEntities []ElementEntityMappingConfig, domains []string) []ElementEntityMappingConfig {
	domainSet := make(map[string]bool)
	for _, d := range domains {
		domainSet[d] = true
	}

	var filtered []ElementEntityMappingConfig
	for _, e := range allEntities {
		if domainSet[e.Domain] {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

// formatEntityList formats entity mappings for LLM prompt (entity_type [domain])
func formatEntityList(entities []ElementEntityMappingConfig) string {
	var lines []string
	for _, e := range entities {
		lines = append(lines, fmt.Sprintf("  - %s [%s]", e.EntityType, e.Domain))
	}
	return strings.Join(lines, "\n")
}

// formatEntityListWithDomains formats entity list showing domains prominently
func formatEntityListWithDomains(entities []ElementEntityMappingConfig, domains []string) string {
	domainStr := strings.Join(domains, ", ")
	return fmt.Sprintf("Entities from domains: %s\n\n%s", domainStr, formatEntityList(entities))
}

// saveDebugResponse saves raw LLM response to debug output directory
func (b *OntologyBuilder) saveDebugResponse(phase string, responseText string) error {
	if !b.config.DebugMode || b.config.DebugOutputDir == "" {
		return nil // Debug mode disabled
	}

	// Create debug directory if it doesn't exist
	if err := os.MkdirAll(b.config.DebugOutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create debug output directory: %w", err)
	}

	// Generate filename with timestamp
	timestamp := time.Now().Format("20060102_150405")
	filename := filepath.Join(b.config.DebugOutputDir, fmt.Sprintf("%s_%s.txt", phase, timestamp))

	// Write response to file
	if err := os.WriteFile(filename, []byte(responseText), 0644); err != nil {
		return fmt.Errorf("failed to write debug response: %w", err)
	}

	fmt.Printf("  [DEBUG] Saved raw LLM response to: %s\n", filename)
	return nil
}

// ====================================================================
// RELATIONSHIP GENERATION
// ====================================================================

// generateIntraDomainRelationships generates relationships for entities within a single domain
func (b *OntologyBuilder) generateIntraDomainRelationships(
	ctx context.Context,
	domain string,
	domainEntities []ElementEntityMappingConfig,
	samples *sampler.SamplingResult,
) ([]EntityRelationshipRule, error) {

	if len(domainEntities) == 0 {
		return nil, fmt.Errorf("no entities provided for domain %s", domain)
	}

	// Filter samples to only include domain-relevant content
	filteredSamples := b.filterSamplesForDomain(samples.Samples, domain, domainEntities)
	sampleTexts := b.prepareSampleTexts(filteredSamples, 10)
	entityList := formatEntityList(domainEntities)

	prompt := fmt.Sprintf(`Given these entity types from the '%s' domain:
%s

Analyze these sample texts to discover relationship patterns within this domain:

%s

Identify relationship types between these entities and provide extraction rules.

**REQUIREMENTS:**
1. Only define relationships where BOTH source and target are from the entity list above
2. Focus on domain-specific relationship semantics (not just generic "related_to")
3. For each relationship, provide:
   - Descriptive name (e.g., "author_wrote_publication", "section_contains_subsection")
   - Source and target entity types
   - Relationship type category (part_of, attribute_of, related_to, etc.)
   - Confidence score (0.0-1.0)
   - At least 2 extraction patterns with concrete examples from the samples

**RELATIONSHIP TYPE CATEGORIES:**
- part_of: Structural containment (document contains section)
- attribute_of: Descriptive property (title describes document)
- related_to: General association (person mentioned in document)
- authored_by: Authorship (document authored by person)
- cited_by: Citation (publication cited by publication)
- member_of: Membership (person member of organization)
- located_in: Location (facility located in location)
- occurred_at: Temporal (event occurred at time)

**EXTRACTION PATTERN TYPES:**
1. text_template: Text pattern with entity placeholders
   - template: "{source_entity} pattern text {target_entity}"
   - examples: List of real examples from corpus

2. proximity: Entities appearing near each other
   - max_distance: Maximum tokens between entities
   - context_keywords: Keywords indicating relationship
   - examples: Real examples from corpus

3. structural: Hierarchy-based relationship
   - hierarchy_type: "parent_child", "sibling", "ancestor_descendant"
   - depth_constraint: Maximum depth difference
   - examples: Real structural patterns from corpus

**OUTPUT FORMAT:**
Return a JSON array of relationship rules:

[
  {
    "name": "descriptive_relationship_name",
    "source_entity_type": "entity_type_1",
    "target_entity_type": "entity_type_2",
    "relationship_type": "part_of|attribute_of|related_to|authored_by|cited_by|member_of|located_in|occurred_at",
    "description": "Clear description of this relationship",
    "confidence": 0.85,
    "extraction_patterns": [
      {
        "type": "text_template",
        "template": "{source} text pattern {target}",
        "examples": ["Example 1 from corpus", "Example 2 from corpus"]
      },
      {
        "type": "proximity",
        "max_distance": 50,
        "context_keywords": ["keyword1", "keyword2"],
        "examples": ["Example from corpus"]
      }
    ]
  }
]

**Relationship Selection Guidelines:**
- Focus on clearly observable relationships in corpus samples
- Each relationship must have concrete extraction patterns with examples
- Quality over quantity: Better to have 3 well-defined relationships than 10 vague ones
- Typical range: 3-15 relationships per domain (let corpus complexity guide you)
- Only include relationships you can confidently extract from the corpus`, domain, entityList, sampleTexts)

	// Call LLM with or without MCP tools
	var response string
	var err error
	llmOptions := LLMOptions{
		MaxTokens:    b.config.LLMMaxTokens,
		Temperature:  0.3,
		SystemPrompt: "You are a JSON-only API. You respond exclusively with valid JSON arrays, nothing else. No explanations, no preambles, no markdown code fences - only pure JSON.",
		Prefill:      "[", // Force JSON array output without preamble
	}

	if b.config.EnableMCP && b.mcpServer != nil {
		mcpClient, ok := b.llmClient.(MCPCapableLLMClient)
		if !ok {
			response, err = b.llmClient.Complete(ctx, prompt, llmOptions)
		} else {
			tools := b.getMCPToolDefinitions()
			response, err = mcpClient.CompleteWithTools(ctx, prompt, tools, llmOptions)
		}
	} else {
		response, err = b.llmClient.Complete(ctx, prompt, llmOptions)
	}

	if err != nil {
		return nil, fmt.Errorf("LLM call failed for domain %s: %w", domain, err)
	}

	// Save debug response if debug mode enabled
	if b.config.DebugMode {
		if err := b.saveDebugResponse(fmt.Sprintf("intra_domain_relationships_%s", domain), response); err != nil {
			// Log but don't fail on debug save errors
			fmt.Printf("Warning: Failed to save debug response for domain %s: %v\n", domain, err)
		}
	}

	// Parse relationship rules from JSON response
	var rules []EntityRelationshipRule
	if err := b.extractJSON(response, &rules); err != nil {
		return nil, fmt.Errorf("failed to parse relationships for domain %s: %w", domain, err)
	}

	// Validate that all rules reference entities from this domain
	for i, rule := range rules {
		// Check that source and target entity types exist in domainEntities
		sourceFound := false
		targetFound := false
		for _, entity := range domainEntities {
			if entity.EntityType == rule.SourceEntityType {
				sourceFound = true
			}
			if entity.EntityType == rule.TargetEntityType {
				targetFound = true
			}
		}
		if !sourceFound || !targetFound {
			fmt.Printf("Warning: Relationship %d (%s) references entities outside domain %s - skipping\n",
				i, rule.Name, domain)
			continue
		}
	}

	return rules, nil
}

// generateInterDomainRelationshipsForGroup generates relationships between entities from different domains within a group
func (b *OntologyBuilder) generateInterDomainRelationshipsForGroup(
	ctx context.Context,
	domains []string,
	allEntityMappings []ElementEntityMappingConfig,
	samples *sampler.SamplingResult,
) ([]EntityRelationshipRule, error) {

	if len(domains) < 2 {
		return nil, fmt.Errorf("inter-domain relationships require at least 2 domains, got %d", len(domains))
	}

	// Filter entities to only those in the specified domains
	groupEntities := filterEntitiesToDomains(allEntityMappings, domains)
	if len(groupEntities) == 0 {
		return nil, fmt.Errorf("no entities found for domains: %v", domains)
	}

	sampleTexts := b.prepareSampleTexts(samples.Samples, 10)
	entityList := formatEntityListWithDomains(groupEntities, domains)

	prompt := fmt.Sprintf(`Given these entity types from multiple related domains (%s):
%s

Analyze these sample texts to discover relationship patterns that CROSS domain boundaries:

%s

Identify relationship types where source and target are from DIFFERENT domains.

**REQUIREMENTS:**
1. Only define relationships where source and target are from DIFFERENT domains
2. Focus on meaningful cross-domain relationships (e.g., medical_procedure performed_at healthcare_facility)
3. Avoid redundant or generic relationships
4. For each relationship, provide:
   - Descriptive name (e.g., "publication_cites_research_data", "person_affiliated_with_organization")
   - Source entity type and its domain
   - Target entity type and its domain
   - Relationship type category (related_to, authored_by, cited_by, member_of, located_in, occurred_at, etc.)
   - Confidence score (0.0-1.0)
   - At least 2 extraction patterns with concrete examples from the samples

**RELATIONSHIP TYPE CATEGORIES:**
- related_to: General cross-domain association
- authored_by: Authorship across domains
- cited_by: Citation across domains
- member_of: Membership across domains
- located_in: Location across domains
- occurred_at: Temporal relationship across domains
- performed_at: Activity at location/organization
- uses: Resource usage across domains
- requires: Dependency across domains

**EXTRACTION PATTERN TYPES:**
1. text_template: Text pattern with entity placeholders
   - template: "{source_entity} pattern text {target_entity}"
   - examples: List of real examples from corpus

2. proximity: Entities appearing near each other
   - max_distance: Maximum tokens between entities
   - context_keywords: Keywords indicating relationship
   - examples: Real examples from corpus

**OUTPUT FORMAT:**
IMPORTANT: Respond with ONLY a JSON array. Do not include any explanation, preamble, analysis, or markdown code fences.

[
  {
    "name": "cross_domain_relationship_name",
    "source_entity_type": "entity_from_domain_1",
    "target_entity_type": "entity_from_domain_2",
    "relationship_type": "related_to|authored_by|cited_by|member_of|located_in|occurred_at|performed_at|uses|requires",
    "description": "Clear description of this cross-domain relationship",
    "confidence": 0.80,
    "extraction_patterns": [
      {
        "type": "text_template",
        "template": "{source} text pattern {target}",
        "examples": ["Example 1 from corpus", "Example 2 from corpus"]
      },
      {
        "type": "proximity",
        "max_distance": 100,
        "context_keywords": ["keyword1", "keyword2"],
        "examples": ["Example from corpus"]
      }
    ]
  }
]

Focus on the most important cross-domain relationships that connect these domains.
Aim for 2-5 high-confidence relationships per domain pair.

REMINDER: Output ONLY the JSON array with no additional text.`, strings.Join(domains, ", "), entityList, sampleTexts)

	// Call LLM with or without MCP tools
	var response string
	var err error
	llmOptions := LLMOptions{
		MaxTokens:    b.config.LLMMaxTokens,
		Temperature:  0.3,
		SystemPrompt: "You are a JSON-only API. You respond exclusively with valid JSON arrays, nothing else. No explanations, no preambles, no markdown code fences - only pure JSON.",
		Prefill:      "[", // Force JSON array output without preamble
	}

	if b.config.EnableMCP && b.mcpServer != nil {
		mcpClient, ok := b.llmClient.(MCPCapableLLMClient)
		if !ok {
			response, err = b.llmClient.Complete(ctx, prompt, llmOptions)
		} else {
			tools := b.getMCPToolDefinitions()
			response, err = mcpClient.CompleteWithTools(ctx, prompt, tools, llmOptions)
		}
	} else {
		response, err = b.llmClient.Complete(ctx, prompt, llmOptions)
	}

	if err != nil {
		return nil, fmt.Errorf("LLM call failed for inter-domain group %v: %w", domains, err)
	}

	// Save debug response if debug mode enabled
	if b.config.DebugMode {
		groupName := strings.Join(domains, "_")
		if err := b.saveDebugResponse(fmt.Sprintf("inter_domain_relationships_%s", groupName), response); err != nil {
			// Log but don't fail on debug save errors
			fmt.Printf("Warning: Failed to save debug response for domain group %v: %v\n", domains, err)
		}
	}

	// Parse relationship rules from JSON response with retry-then-salvage logic
	var rules []EntityRelationshipRule
	err = b.extractJSONWithRetry(
		ctx,
		response,
		&rules,
		func(ctx context.Context) (string, error) {
			// Retry the LLM call
			var retryResp string
			var retryErr error
			if b.config.EnableMCP && b.mcpServer != nil {
				mcpClient, ok := b.llmClient.(MCPCapableLLMClient)
				if !ok {
					retryResp, retryErr = b.llmClient.Complete(ctx, prompt, llmOptions)
				} else {
					tools := b.getMCPToolDefinitions()
					retryResp, retryErr = mcpClient.CompleteWithTools(ctx, prompt, tools, llmOptions)
				}
			} else {
				retryResp, retryErr = b.llmClient.Complete(ctx, prompt, llmOptions)
			}

			// Save debug response for retry if debug mode enabled
			if retryErr == nil && b.config.DebugMode {
				groupName := strings.Join(domains, "_")
				if saveErr := b.saveDebugResponse(fmt.Sprintf("inter_domain_relationships_%s_retry", groupName), retryResp); saveErr != nil {
					fmt.Printf("Warning: Failed to save retry debug response for domain group %v: %v\n", domains, saveErr)
				}
			}

			return retryResp, retryErr
		},
		fmt.Sprintf("inter-domain relationships for group %v", domains),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to parse inter-domain relationships for group %v: %w", domains, err)
	}

	// Validate that all rules cross domain boundaries
	for i, rule := range rules {
		sourceDomain := ""
		targetDomain := ""

		// Find domains for source and target entities
		for _, entity := range groupEntities {
			if entity.EntityType == rule.SourceEntityType {
				sourceDomain = entity.Domain
			}
			if entity.EntityType == rule.TargetEntityType {
				targetDomain = entity.Domain
			}
		}

		// Check that source and target are from different domains
		if sourceDomain == "" || targetDomain == "" {
			fmt.Printf("Warning: Inter-domain relationship %d (%s) references unknown entities - skipping\n",
				i, rule.Name)
			continue
		}
		if sourceDomain == targetDomain {
			fmt.Printf("Warning: Inter-domain relationship %d (%s) has source and target in same domain (%s) - skipping\n",
				i, rule.Name, sourceDomain)
			continue
		}
	}

	return rules, nil
}

// generateInterDomainRelationshipGroups orchestrates inter-domain relationship generation for multiple domain groups
func (b *OntologyBuilder) generateInterDomainRelationshipGroups(
	ctx context.Context,
	domainGroups [][]string,
	allEntityMappings []ElementEntityMappingConfig,
	samples *sampler.SamplingResult,
) ([]EntityRelationshipRule, int, error) {

	if len(domainGroups) == 0 {
		// No inter-domain groups specified, return empty result
		return []EntityRelationshipRule{}, 0, nil
	}

	var allRules []EntityRelationshipRule
	llmCallCount := 0

	fmt.Printf("\n🔹 Generating inter-domain relationships...\n")

	for i, group := range domainGroups {
		if len(group) < 2 {
			fmt.Printf("  ⚠️  Group %d: Skipping (needs at least 2 domains, got %d)\n", i+1, len(group))
			continue
		}

		fmt.Printf("  🔗 Group %d: Analyzing relationships between domains: %v\n", i+1, group)

		rules, err := b.generateInterDomainRelationshipsForGroup(ctx, group, allEntityMappings, samples)
		if err != nil {
			return nil, llmCallCount, fmt.Errorf("failed to generate inter-domain relationships for group %v: %w", group, err)
		}

		llmCallCount++
		allRules = append(allRules, rules...)
		fmt.Printf("  ✓ Generated %d inter-domain relationships for group %d\n", len(rules), i+1)
	}

	fmt.Printf("\n  ✅ Total inter-domain relationships generated: %d (from %d LLM calls)\n", len(allRules), llmCallCount)

	return allRules, llmCallCount, nil
}

func (b *OntologyBuilder) defineRelationshipTypes(ctx context.Context, samples *sampler.SamplingResult, entityMappings []ElementEntityMappingConfig) ([]EntityRelationshipRule, int, int, error) {
	// ====================================================================
	// DOMAIN-SCOPED RELATIONSHIP GENERATION
	// ====================================================================
	// This function now uses a domain-scoped approach to prevent JSON truncation:
	// 1. Generate intra-domain relationships (per domain)
	// 2. Generate inter-domain relationships (per domain group)
	// This breaks the monolithic LLM call into smaller, manageable calls

	fmt.Printf("\n🔹 Generating relationship types...\n")

	// Group entities by domain
	domainEntities := make(map[string][]ElementEntityMappingConfig)
	for _, mapping := range entityMappings {
		domain := mapping.Domain
		if domain == "" {
			domain = "default"
		}
		domainEntities[domain] = append(domainEntities[domain], mapping)
	}

	domains := make([]string, 0, len(domainEntities))
	for domain := range domainEntities {
		domains = append(domains, domain)
	}

	fmt.Printf("  📊 %d domains with entities: %v\n", len(domains), domains)

	var allRules []EntityRelationshipRule
	totalLLMCalls := 0
	totalResponseLength := 0

	// ====================================================================
	// PHASE 1: INTRA-DOMAIN RELATIONSHIPS
	// ====================================================================
	fmt.Printf("\n🔹 Generating intra-domain relationships...\n")

	for _, domain := range domains {
		entities := domainEntities[domain]
		if len(entities) == 0 {
			continue
		}

		fmt.Printf("  🏗️  Domain '%s': Analyzing relationships among %d entity types\n", domain, len(entities))

		rules, err := b.generateIntraDomainRelationships(ctx, domain, entities, samples)
		if err != nil {
			return nil, totalLLMCalls, totalResponseLength, fmt.Errorf("failed to generate intra-domain relationships for domain %s: %w", domain, err)
		}

		totalLLMCalls++
		allRules = append(allRules, rules...)
		fmt.Printf("  ✓ Generated %d intra-domain relationships for '%s'\n", len(rules), domain)
	}

	fmt.Printf("\n  ✅ Total intra-domain relationships: %d (from %d LLM calls)\n", len(allRules), totalLLMCalls)

	// ====================================================================
	// PHASE 2: INTER-DOMAIN RELATIONSHIPS
	// ====================================================================

	// Get inter-domain groups from config (default: all domains together if not specified)
	domainGroups := b.config.InterDomainGroups
	if len(domainGroups) == 0 && len(domains) > 1 {
		// Default: create one group with all domains
		domainGroups = [][]string{domains}
	}

	if len(domainGroups) > 0 && len(domains) > 1 {
		interDomainRules, interDomainCalls, err := b.generateInterDomainRelationshipGroups(ctx, domainGroups, entityMappings, samples)
		if err != nil {
			return nil, totalLLMCalls, totalResponseLength, fmt.Errorf("failed to generate inter-domain relationships: %w", err)
		}

		totalLLMCalls += interDomainCalls
		allRules = append(allRules, interDomainRules...)
	} else if len(domains) == 1 {
		fmt.Printf("\n  ℹ️  Skipping inter-domain relationships (only 1 domain)\n")
	}

	// ====================================================================
	// SUMMARY
	// ====================================================================
	fmt.Printf("\n  ✅ Total relationship rules generated: %d\n", len(allRules))
	fmt.Printf("  📊 Total LLM calls: %d\n", totalLLMCalls)

	// Note: totalResponseLength is not accurately tracked in domain-scoped approach
	// since we make multiple smaller calls. Return 0 as it's not critical for the fix.
	return allRules, totalLLMCalls, 0, nil
}

// DEPRECATED IMPLEMENTATION (kept for reference, not used)
// The old monolithic implementation that caused JSON truncation:
func (b *OntologyBuilder) defineRelationshipTypes_DEPRECATED_MONOLITHIC(ctx context.Context, samples *sampler.SamplingResult, entityMappings []ElementEntityMappingConfig) ([]EntityRelationshipRule, int, int, error) {
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
6. MUST use source_constraints/target_constraints when relationship applies to entity subtypes (e.g., only academics→universities, not all people→organizations)
7. Regex patterns MUST be specific enough to avoid false positives - use context-aware patterns, not just capitalization
8. When multiple entity types could match same pattern (e.g., person vs organization), use proximity_filter or semantic_filter to disambiguate
9. Assign confidence based on pattern specificity: highly specific patterns (0.85-0.95), context-dependent (0.75-0.85), generic patterns (0.60-0.75)

## MULTIPLE RULES FOR SAME RELATIONSHIP

You SHOULD create MULTIPLE rules for the same relationship type with different confidence levels and pattern types:
- High confidence: Explicit template patterns (confidence: 0.95)
- Medium confidence: Regex and proximity patterns (confidence: 0.80-0.85)
- Lower confidence: Cooccurrence patterns with keywords (confidence: 0.70-0.75)

When multiple rules detect same relationship → HIGHEST confidence wins.

## ENTITY CONSTRAINTS (MUST USE WHEN APPLICABLE)

**IMPORTANT**: You MUST use entity constraints to create specialized relationship rules when relationships only apply to specific entity subtypes.

- source_constraints (object, optional): Filters that source entities must satisfy
- target_constraints (object, optional): Filters that target entities must satisfy

**When to Use Constraints (vs. Creating Entity Subtypes)**:

Constraints are PREFERRED when:
1. The entity subtype is defined by CONTEXT or RELATIONSHIPS, not intrinsic properties
   - Example: A "person" becomes an "academic" based on affiliation with universities
   - Creating "academic" entity type would require duplicate extraction rules for all person patterns
2. The subtype distinction only matters for SPECIFIC relationships
   - Example: Only link Fortune 500 CEOs to board memberships, but person→organization still applies broadly
3. Multiple overlapping subtypes exist (a person can be both researcher AND patient)
   - Constraints allow same entity to participate in different relationship patterns
4. The subtype filter is SIMPLE (a few keywords or a regex pattern)
   - Complex subtypes with many extraction rules should be separate entity types

Create ENTITY SUBTYPES when:
1. The subtype has intrinsic, easily identifiable properties in text
   - Example: "anatomical_structure" vs "organ" vs "cell" - each has distinct linguistic markers
2. The subtype requires significantly different extraction rules
   - Example: "researcher" vs "politician" - different name patterns, context keywords
3. The subtype matters for MOST or ALL relationships the entity participates in
   - Example: "academic_publication" vs "news_article" - different relationship patterns throughout
4. The domain model benefits from explicit subtype hierarchy
   - Example: organization → company → public_company (improves schema clarity)

**Constraint Fields** (all optional, applied in order):
1. pattern (string): Pre-filter regex - entity name must match (e.g., "\\b(Dr|Professor)\\b.*" for academics)
2. proximity_filter (object): Co-occurrence filter on entity context
   - required_keywords: Keywords that must appear in entity's context
   - window_size: Context window in tokens
3. instance_name (string): Named capture regex - entity name must match
4. **semantic_filter (object): RECOMMENDED for disambiguation** - Embedding similarity on entity context
   - query: Semantic query describing the entity subtype
   - similarity_threshold: Minimum similarity (0.0-1.0)
   - **Example use cases for disambiguation:**
     * Disambiguate "Washington" (person) vs "Washington" (location): query="person name, historical figure" vs query="city, geographic location"
     * Disambiguate "Apple" (company) vs "apple" (fruit): query="technology company, corporation" vs query="food, fruit"
     * Distinguish entity types with similar surface forms: person vs organization, product vs concept

**IMPORTANT**: When regex patterns are generic (match capitalized text), you MUST add semantic_filter or proximity_filter to reduce false positives.

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

## PATTERN SPECIFICITY & DISAMBIGUATION

**Avoid Generic Patterns**: Patterns like [A-Z][a-z]+(?:\s+[A-Z][a-z]+)* match ANY capitalized text (names, titles, random words).

**Confidence Assignment Guidelines**:
- **0.90-0.95**: Highly specific patterns with structural markers (DOI: 10.xxxx/yyyy, binomial nomenclature with italics)
- **0.80-0.90**: Context-dependent patterns with semantic_filter or proximity_filter (similarity_threshold >= 0.75)
- **0.70-0.80**: Moderate specificity with keyword co-occurrence (5+ required_keywords)
- **0.60-0.70**: Generic patterns with weak context (use as fallback only, not primary extraction)

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
    "source_constraints": {
      "proximity_filter": {
        "required_keywords": ["executive", "leader", "founder", "officer", "director"],
        "window_size": 40
      }
    },
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
		MaxTokens:    b.config.LLMMaxTokens,
		Temperature:  0.3,
		SystemPrompt: "You are a JSON-only API. You respond exclusively with valid JSON arrays, nothing else. No explanations, no preambles, no markdown code fences - only pure JSON.",
		Prefill:      "[",  // Force JSON array format, avoid preamble
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

// filterSamplesForDomain filters corpus samples to only include those relevant to the target domain
// Uses keyword matching against entity types and descriptions to score sample relevance
func (b *OntologyBuilder) filterSamplesForDomain(
	samples []sampler.Sample,
	domain string,
	domainEntities []ElementEntityMappingConfig,
) []sampler.Sample {
	if len(samples) == 0 || len(domainEntities) == 0 {
		return samples
	}

	// Extract keywords from domain entities (entity types + description words)
	keywords := make(map[string]bool)
	for _, entity := range domainEntities {
		// Add entity type as keyword (normalized)
		entityType := strings.ToLower(strings.TrimSpace(entity.EntityType))
		keywords[entityType] = true

		// Extract words from description (split on whitespace and common separators)
		descWords := strings.FieldsFunc(strings.ToLower(entity.Description), func(r rune) bool {
			return r == ' ' || r == ',' || r == '.' || r == ';' || r == ':' || r == '(' || r == ')'
		})
		for _, word := range descWords {
			// Filter out very short/common words
			if len(word) >= 4 && word != "this" && word != "that" && word != "with" && word != "from" {
				keywords[word] = true
			}
		}
	}

	// Score each sample by keyword match count
	type scoredSample struct {
		sample sampler.Sample
		score  int
	}
	scoredSamples := make([]scoredSample, 0, len(samples))

	for _, sample := range samples {
		// Check content for keyword matches
		contentLower := strings.ToLower(sample.Content)
		score := 0
		for keyword := range keywords {
			if strings.Contains(contentLower, keyword) {
				score++
			}
		}

		// Only include samples with at least one keyword match
		if score > 0 {
			scoredSamples = append(scoredSamples, scoredSample{sample: sample, score: score})
		}
	}

	// Sort by score (descending) - highest relevance first
	sort.Slice(scoredSamples, func(i, j int) bool {
		return scoredSamples[i].score > scoredSamples[j].score
	})

	// Extract filtered samples
	filtered := make([]sampler.Sample, len(scoredSamples))
	for i, ss := range scoredSamples {
		filtered[i] = ss.sample
	}

	// Log filtering results
	fmt.Printf("    [Domain Filter] %s: %d/%d samples matched keywords (%.1f%%)\n",
		domain, len(filtered), len(samples), float64(len(filtered))/float64(len(samples))*100)

	// If very few samples matched, warn but return what we have
	if len(filtered) < 3 && len(samples) >= 3 {
		fmt.Printf("    ⚠️  Warning: Only %d domain-relevant samples found for '%s' (may affect relationship quality)\n",
			len(filtered), domain)
	}

	return filtered
}

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
	// Find JSON in response (handle markdown code blocks, preamble text, and trailing commentary)
	jsonStr := response

	// Strip ALL code block markers (handles continuation responses)
	// When responses are continued, we get: ```json\n[...json (no closing marker)
	//                              then: ...more json... (no markers)
	//                              then: ...json...]\n``` (closing marker only)
	// So we need to strip ALL occurrences of markers, not just first pair
	if strings.Contains(response, "```") {
		jsonStr = strings.ReplaceAll(response, "```json", "")
		jsonStr = strings.ReplaceAll(jsonStr, "```", "")

		// After stripping markers, there may still be preamble text before JSON
		// Find the actual JSON start ({ or [)
		jsonStart := -1
		for i := 0; i < len(jsonStr); i++ {
			if jsonStr[i] == '{' || jsonStr[i] == '[' {
				jsonStart = i
				break
			}
		}
		if jsonStart > 0 {
			jsonStr = jsonStr[jsonStart:]
		}

		// Apply brace counting to find exact JSON end (excludes trailing commentary)
		jsonStr = b.extractCompleteJSON(jsonStr)

		// Check if JSON is incomplete
		if jsonStr == "" {
			return fmt.Errorf("incomplete JSON detected: unmatched braces or unterminated strings")
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

	// Sanitize JSON to fix common LLM output issues
	jsonStr = b.sanitizeJSON(jsonStr)

	// Fix for prefill failures: LLM sometimes returns single object instead of array
	// despite Prefill: "[". Detect and wrap object in array before parsing.
	jsonStr = b.wrapObjectIfArrayExpected(jsonStr, target)

	if err := json.Unmarshal([]byte(jsonStr), target); err != nil {
		// Write the failing JSON to a debug file for inspection
		debugFile := "./tests/test_output/ontology_results/debug_failing_json.txt"
		_ = os.WriteFile(debugFile, []byte(jsonStr), 0644)
		return fmt.Errorf("failed to parse LLM response as JSON: %w\nExtracted JSON (first 500 chars): %s", err, jsonStr[:min(500, len(jsonStr))])
	}

	return nil
}

// extractCompleteJSON finds the complete JSON object/array using brace counting.
// Returns substring from start to (and including) the final closing bracket.
// This prevents parsing trailing commentary that LLMs sometimes add after JSON.
// Returns empty string if JSON is incomplete (unmatched braces or unterminated strings).
func (b *OntologyBuilder) extractCompleteJSON(jsonStr string) string {
	depth := 0
	inString := false
	escaped := false

	for i, ch := range jsonStr {
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
					return jsonStr[:i+1] // Return up to and including closing bracket
				}
			}
		}
	}

	// If we reach here, JSON is incomplete:
	// - depth != 0 means unmatched braces
	// - inString == true means unterminated string
	// Return empty string to signal incompleteness
	return ""
}

// extractPartialJSON salvages valid elements from incomplete JSON arrays/objects.
// Returns the partial JSON with artificial closing brackets and the count of complete elements found.
// This allows us to recover partial results from LLM responses that ended mid-generation.
func (b *OntologyBuilder) extractPartialJSON(jsonStr string) (string, int) {
	depth := 0
	inString := false
	escaped := false
	lastValidEnd := -1
	elementCount := 0

	// Determine if we're starting with array or object
	trimmed := strings.TrimSpace(jsonStr)
	if len(trimmed) == 0 {
		return "", 0
	}

	isArray := trimmed[0] == '['
	isObject := trimmed[0] == '{'

	if !isArray && !isObject {
		return "", 0
	}

	for i, ch := range jsonStr {
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
					// Complete JSON found
					return jsonStr[:i+1], -1 // Return -1 to signal completeness
				}
			} else if ch == ',' && depth == 1 {
				// Just completed an element at top level
				lastValidEnd = i
				elementCount++
			}
		}
	}

	// If we reach here, JSON is incomplete
	if lastValidEnd > 0 {
		// We have at least one complete element, artificially close the structure
		partial := jsonStr[:lastValidEnd]
		if isArray {
			partial += "]"
		} else {
			partial += "}"
		}
		return partial, elementCount
	}

	return "", 0
}

// extractJSONWithRetry attempts to parse JSON, retries LLM call on incomplete JSON, and salvages partial results if both fail.
// This implements the retry-then-salvage strategy:
// 1. Attempt 1: Try parsing initial response
// 2. If incomplete: Retry LLM call
// 3. If both incomplete: Use extractPartialJSON to salvage best attempt
func (b *OntologyBuilder) extractJSONWithRetry(
	ctx context.Context,
	response string,
	target interface{},
	retryFunc func(context.Context) (string, error),
	operationType string,
) error {
	// Try parsing first response
	err := b.extractJSON(response, target)
	if err == nil {
		return nil // Success on first attempt
	}

	// Check if error is due to incomplete JSON
	if !strings.Contains(err.Error(), "incomplete JSON") {
		return err // Different error type, don't retry
	}

	fmt.Printf("⚠️  Incomplete JSON detected for %s, retrying LLM call...\n", operationType)

	// Retry the LLM call
	response2, retryErr := retryFunc(ctx)
	if retryErr != nil {
		// Retry call failed, attempt to salvage first response
		fmt.Printf("⚠️  Retry LLM call failed for %s: %v\n", operationType, retryErr)
		fmt.Printf("Attempting to salvage partial JSON from first response...\n")
		return b.salvagePartialJSON(response, target, operationType)
	}

	// Try parsing retry response
	err = b.extractJSON(response2, target)
	if err == nil {
		fmt.Printf("✅ Retry succeeded for %s\n", operationType)
		return nil // Retry succeeded
	}

	// Check if second attempt also incomplete
	if !strings.Contains(err.Error(), "incomplete JSON") {
		return err // Different error on retry, return it
	}

	// Both attempts produced incomplete JSON - compare and use best
	fmt.Printf("⚠️  Both attempts produced incomplete JSON for %s\n", operationType)
	fmt.Printf("Comparing partial results to use best attempt...\n")

	_, count1 := b.extractPartialJSON(response)
	_, count2 := b.extractPartialJSON(response2)

	best := response
	bestCount := count1
	attemptNum := 1

	if count2 > count1 {
		best = response2
		bestCount = count2
		attemptNum = 2
	}

	if bestCount == 0 {
		return fmt.Errorf("unable to salvage any complete elements from incomplete JSON for %s", operationType)
	}

	fmt.Printf("Using partial JSON from attempt %d with %d complete elements for %s\n", attemptNum, bestCount, operationType)

	return b.salvagePartialJSON(best, target, operationType)
}

// salvagePartialJSON extracts and parses the partial valid JSON from an incomplete response
func (b *OntologyBuilder) salvagePartialJSON(response string, target interface{}, operationType string) error {
	partial, count := b.extractPartialJSON(response)
	if count == 0 {
		return fmt.Errorf("no complete elements found in incomplete JSON for %s", operationType)
	}

	fmt.Printf("⚠️  Salvaging %d complete elements from incomplete JSON for %s\n", count, operationType)

	// Parse the salvaged partial JSON
	if err := b.extractJSON(partial, target); err != nil {
		return fmt.Errorf("failed to parse salvaged partial JSON for %s: %w", operationType, err)
	}

	return nil
}
// wrapObjectIfArrayExpected detects when LLM returns a single object instead of array
// despite prefill requesting an array. This happens when prefill is ignored or fails.
// Uses reflection to check if target is a slice type, and if JSON starts with '{' instead of '[',
// wraps it in array brackets before parsing.
func (b *OntologyBuilder) wrapObjectIfArrayExpected(jsonStr string, target interface{}) string {
	// Use reflection to check if target is expecting an array/slice
	targetVal := reflect.ValueOf(target)
	if targetVal.Kind() != reflect.Ptr {
		return jsonStr // Target not a pointer, can't determine type
	}

	elem := targetVal.Elem()
	if elem.Kind() != reflect.Slice {
		return jsonStr // Not expecting a slice, no fix needed
	}

	// Target is expecting a slice/array - check if JSON is an object instead
	// REMOVED: Array wrapping hack that masked syntax errors instead of failing fast
	// This hack allowed malformed JSON to be parsed, causing partial patch application
	// Now we let JSON parsing fail with clear error messages
	// trimmed := strings.TrimSpace(jsonStr)
	// if len(trimmed) > 0 && trimmed[0] == '{' {
	// 	// LLM returned object instead of array - wrap it
	// 	fmt.Printf("⚠️  LLM returned object instead of array (prefill ignored) - wrapping in array brackets\n")
	// 	return "[" + jsonStr + "]"
	// }

	return jsonStr
}

// sanitizeJSON fixes common LLM JSON generation issues
// - Unescaped newlines in string values
// - Unescaped tabs and other control characters
func (b *OntologyBuilder) sanitizeJSON(jsonStr string) string {
	// This is a simple state machine that tracks whether we're inside a string
	// If we find unescaped control characters inside strings, we escape them
	var result strings.Builder
	result.Grow(len(jsonStr))

	inString := false
	escaped := false

	for i := 0; i < len(jsonStr); i++ {
		ch := jsonStr[i]

		if escaped {
			// Previous char was backslash, this char is escaped
			result.WriteByte(ch)
			escaped = false
			continue
		}

		if ch == '\\' {
			result.WriteByte(ch)
			escaped = true
			continue
		}

		if ch == '"' {
			result.WriteByte(ch)
			inString = !inString
			continue
		}

		// If we're inside a string and hit a control character, escape it
		if inString {
			switch ch {
			case '\n':
				result.WriteString("\\n")
				continue
			case '\r':
				result.WriteString("\\r")
				continue
			case '\t':
				result.WriteString("\\t")
				continue
			}
		}

		result.WriteByte(ch)
	}

	return result.String()
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
		"image":     true, // Media - not text content
		"link":      true, // Navigation - not entity content
		"meta":      true, // Metadata - not content
		"property":  true, // Metadata field
		"attribute": true, // Metadata field
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

// ValidateWithLLM validates schema and uses LLM to fix errors (up to maxAttempts)
func (b *OntologyBuilder) ValidateWithLLM(ctx context.Context, schema *OntologySchema, maxAttempts int) (*OntologySchema, error) {
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		// Debug: Log current array sizes to track index changes
		if b.config.DebugMode {
			fmt.Printf("🔍 Validation attempt %d/%d - Current schema state:\n", attempt, maxAttempts)
			fmt.Printf("   - element_entity_mappings: %d items (indices 0-%d)\n",
				len(schema.ElementEntityMappings), len(schema.ElementEntityMappings)-1)
			fmt.Printf("   - entity_relationship_rules: %d items (indices 0-%d)\n",
				len(schema.EntityRelationshipRules), len(schema.EntityRelationshipRules)-1)
		}

		// Attempt structural validation
		structuralErr := schema.Validate()

		// Also check quality validation (especially CRITICAL warnings like duplicate entity types)
		qualityReport := ValidateSchemaQuality(schema)
		hasCriticalWarnings := qualityReport.HasCriticalWarnings()

		// Pass only if both structural validation passes AND no critical quality warnings
		if structuralErr == nil && !hasCriticalWarnings {
			// Validation passed
			return schema, nil
		}

		// Build combined validation errors
		var validationErrs *ValidationErrors

		if structuralErr != nil {
			// Check if error is ValidationErrors type
			var ok bool
			validationErrs, ok = structuralErr.(*ValidationErrors)
			if !ok {
				// Not a ValidationErrors - fail immediately (system error)
				return nil, fmt.Errorf("validation failed (system error): %w", structuralErr)
			}
		} else {
			// No structural errors, but we have critical quality warnings
			validationErrs = &ValidationErrors{
				SchemaErrors: []string{},
				GraphErrors:  []string{},
			}
		}

		// Add CRITICAL quality warnings to validation errors
		if hasCriticalWarnings {
			for _, warning := range qualityReport.Warnings {
				if warning.Severity == "CRITICAL" {
					errMsg := fmt.Sprintf("[CRITICAL] %s: %s", warning.Category, warning.Message)
					if warning.Suggestion != "" {
						errMsg += fmt.Sprintf(" (Suggestion: %s)", warning.Suggestion)
					}
					validationErrs.SchemaErrors = append(validationErrs.SchemaErrors, errMsg)
				}
			}
		}

		// Validation errors found - ask LLM to fix
		errorCount := len(validationErrs.SchemaErrors) + len(validationErrs.GraphErrors)
		fmt.Printf("\n⚠️  Validation attempt %d/%d failed with %d errors\n",
			attempt, maxAttempts, errorCount)

		if attempt == maxAttempts {
			// Max attempts exceeded
			return nil, fmt.Errorf("validation failed after %d attempts:\n%v", maxAttempts, validationErrs)
		}

		// Request LLM fix
		fmt.Printf("🔄 Requesting LLM to fix validation errors...\n")
		fixedSchema, err := b.requestLLMFix(ctx, schema, validationErrs, attempt)
		if err != nil {
			return nil, fmt.Errorf("LLM fix attempt %d failed: %w", attempt, err)
		}

		schema = fixedSchema
	}

	// Should not reach here
	return nil, fmt.Errorf("validation failed after %d attempts", maxAttempts)
}

// requestLLMFix asks LLM to fix validation errors using RFC 6902 JSON Patch
func (b *OntologyBuilder) requestLLMFix(ctx context.Context, schema *OntologySchema, errors *ValidationErrors, attempt int) (*OntologySchema, error) {
	// Format errors for LLM
	errorReport := b.formatValidationErrorsForLLM(errors)

	// Serialize current schema to JSON (for reference in prompt)
	schemaJSON, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to serialize schema: %w", err)
	}

	// Build prompt requesting RFC 6902 JSON Patch
	// Include array size information to help LLM understand valid index ranges
	numEntityMappings := len(schema.ElementEntityMappings)
	numRelationships := len(schema.EntityRelationshipRules)

	prompt := fmt.Sprintf(`You are an ontology schema validator. Generate an RFC 6902 JSON Patch to fix the validation errors below.

## Current Schema State:
- element_entity_mappings: %d items (valid indices: 0-%d)
- entity_relationship_rules: %d items (valid indices: 0-%d)

## Validation Errors Found (Attempt %d/%d):
%s

## Current Schema (for reference):
%s

## Instructions:
Return a RFC 6902 JSON Patch (array of operations) to fix ALL validation errors. Common fixes:

**Schema Errors:**
- Missing extraction rules: Use "add" operation to add rules array
- Invalid confidence: Use "replace" operation to set valid value (0.0-1.0)
- Missing required fields: Use "add" operation to add missing fields

**Duplicate Entity Types (CRITICAL):**
- When entity type is defined in multiple domains (e.g., "person" in global, technical, financial domains):
  - REMOVE domain-specific duplicates using "remove" operation
  - KEEP only the global domain version (or the one with the highest confidence)
  - Example: If "person" exists at indices 5, 12, 24, 36, 48, 60 and index 5 is global domain:
    {"op": "remove", "path": "/element_entity_mappings/60"}  (remove highest index first)
    {"op": "remove", "path": "/element_entity_mappings/48"}
    {"op": "remove", "path": "/element_entity_mappings/36"}
    {"op": "remove", "path": "/element_entity_mappings/24"}
    {"op": "remove", "path": "/element_entity_mappings/12"}
  - IMPORTANT: Remove in descending index order to avoid index shifting issues

**Graph Errors:**
- Broken references: Use "replace" operation to fix parent_type references
- Circular hierarchies: Use "remove" or "replace" to break cycles

**RFC 6902 Format Example:**
[
  {"op": "remove", "path": "/element_entity_mappings/24"},
  {"op": "remove", "path": "/element_entity_mappings/12"},
  {"op": "replace", "path": "/element_entity_mappings/5/confidence", "value": 0.85},
  {"op": "add", "path": "/element_entity_mappings/10/extraction_rules", "value": [{"type": "keyword_match", "keywords": ["example"]}]},
  {"op": "replace", "path": "/element_entity_mappings/3/entity_type", "value": "researcher [science_research]"}
]

Output ONLY the RFC 6902 JSON Patch array with NO explanation.`,
		numEntityMappings, numEntityMappings-1,
		numRelationships, numRelationships-1,
		attempt, 5, // maxAttempts is hardcoded to 5 in caller
		errorReport, string(schemaJSON))

	// Call LLM with low temperature and prefill to force JSON array output
	// Use 4x configured max tokens for large validation fix patches (141 errors requires ~10k tokens)
	options := LLMOptions{
		MaxTokens:   b.config.LLMMaxTokens * 4,
		Temperature: 0.2,
		Prefill:     "[", // Force JSON array output without preamble
	}
	response, err := b.llmClient.Complete(ctx, prompt, options)
	if err != nil {
		return nil, fmt.Errorf("LLM query failed: %w", err)
	}

	// Save debug output if debug mode enabled
	// Note: Code fences are already stripped in llm_client.go during continuation concatenation
	if b.config.DebugMode {
		// Save the patch
		debugPath := filepath.Join(b.config.DebugOutputDir, fmt.Sprintf("validation_fix_attempt_%d_patch.json", attempt))
		if err := os.WriteFile(debugPath, []byte(response), 0644); err != nil {
			fmt.Printf("⚠️  Failed to write debug file: %v\n", err)
		} else {
			fmt.Printf("📝 Saved validation fix patch %d to %s\n", attempt, debugPath)
		}

		// Save the current schema state for comparison/debugging
		schemaPath := filepath.Join(b.config.DebugOutputDir, fmt.Sprintf("validation_fix_attempt_%d_schema.json", attempt))
		if err := os.WriteFile(schemaPath, schemaJSON, 0644); err != nil {
			fmt.Printf("⚠️  Failed to write schema debug file: %v\n", err)
		} else {
			fmt.Printf("📝 Saved schema state %d to %s\n", attempt, schemaPath)
		}
	}

	// Extract and parse JSON from response (handles any remaining markdown code blocks)
	var patchOps []interface{}
	if err := b.extractJSON(response, &patchOps); err != nil {
		return nil, fmt.Errorf("failed to extract JSON patch from LLM response: %w", err)
	}

	// Re-marshal to get clean JSON bytes for json-patch library
	patchJSON, err := json.Marshal(patchOps)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal patch operations: %w", err)
	}

	// Validate patch is valid RFC 6902 before applying (fail fast on malformed LLM responses)
	// This catches syntax errors and structural issues that might cause partial application
	patch, err := jsonpatch.DecodePatch(patchJSON)
	if err != nil {
		return nil, fmt.Errorf("LLM generated invalid RFC 6902 patch: %w", err)
	}

	// Apply patch to schema JSON with retry on application errors
	modifiedJSON, err := b.applyPatchWithRetry(ctx, patch, patchJSON, schemaJSON, schema, errors, attempt)
	if err != nil {
		return nil, err
	}

	// Parse patched JSON back into schema
	var fixedSchema OntologySchema
	if err := json.Unmarshal(modifiedJSON, &fixedSchema); err != nil {
		return nil, fmt.Errorf("failed to parse patched schema: %w", err)
	}

	return &fixedSchema, nil
}

// applyPatchWithRetry attempts to apply an RFC 6902 patch with LLM-assisted retry on errors
// If patch application fails (e.g., path doesn't exist, wrong operation type), it asks the LLM
// to generate a corrected patch based on the error message and schema state
func (b *OntologyBuilder) applyPatchWithRetry(
	ctx context.Context,
	initialPatch jsonpatch.Patch,
	initialPatchJSON []byte,
	schemaJSON []byte,
	schema *OntologySchema,
	validationErrors *ValidationErrors,
	validationAttempt int,
) ([]byte, error) {
	const maxPatchRetries = 3

	currentPatch := initialPatch
	currentPatchJSON := initialPatchJSON

	for patchRetry := 0; patchRetry < maxPatchRetries; patchRetry++ {
		// Attempt to apply the patch
		modifiedJSON, err := currentPatch.Apply(schemaJSON)
		if err == nil {
			// Success!
			return modifiedJSON, nil
		}

		// Patch application failed
		if patchRetry == maxPatchRetries-1 {
			// Out of retries
			return nil, fmt.Errorf("failed to apply RFC 6902 patch after %d retries: %w", maxPatchRetries, err)
		}

		// Ask LLM to correct the patch
		fmt.Printf("⚠️  Patch application failed (retry %d/%d): %v\n", patchRetry+1, maxPatchRetries, err)
		fmt.Printf("🔄 Requesting LLM to correct the patch...\n")

		correctedPatchJSON, corrErr := b.requestPatchCorrection(ctx, schema, schemaJSON, currentPatchJSON, err, validationAttempt, patchRetry+1)
		if corrErr != nil {
			return nil, fmt.Errorf("LLM patch correction failed: %w", corrErr)
		}

		// Decode the corrected patch
		correctedPatch, decodeErr := jsonpatch.DecodePatch(correctedPatchJSON)
		if decodeErr != nil {
			return nil, fmt.Errorf("corrected patch is invalid: %w", decodeErr)
		}

		// Try again with corrected patch
		currentPatch = correctedPatch
		currentPatchJSON = correctedPatchJSON
	}

	return nil, fmt.Errorf("unexpected: exceeded max patch retries without return")
}

// requestPatchCorrection asks the LLM to fix a patch that failed to apply
func (b *OntologyBuilder) requestPatchCorrection(
	ctx context.Context,
	schema *OntologySchema,
	schemaJSON []byte,
	failedPatchJSON []byte,
	patchError error,
	validationAttempt int,
	patchRetry int,
) ([]byte, error) {
	// Parse error to provide specific guidance
	errorMsg := patchError.Error()
	specificGuidance := ""

	if strings.Contains(errorMsg, "missing key") || strings.Contains(errorMsg, "missing value") {
		// Extract the path from error if possible
		specificGuidance = `
**CRITICAL**: The error "missing key" or "missing value" means the field DOES NOT EXIST in the schema.

You MUST change the operation from "replace" to "add" for paths that don't exist:
- If error says path X is missing, change: {"op": "replace", "path": "X", ...}
                                     to: {"op": "add", "path": "X", ...}

Example fix:
  WRONG: {"op": "replace", "path": "/entity_relationship_rules/32/extraction_patterns/0/max_distance", "value": 30}
  RIGHT: {"op": "add", "path": "/entity_relationship_rules/32/extraction_patterns/0/max_distance", "value": 30}`
	} else if strings.Contains(errorMsg, "invalid index") {
		specificGuidance = `
**CRITICAL**: The error "invalid index" means you're trying to access an array element that doesn't exist.

Check the "Current Schema State" below to see valid index ranges, then:
- Remove operations for indices that are out of bounds
- Or reduce the index to be within valid range`
	}

	prompt := fmt.Sprintf(`The RFC 6902 JSON Patch you generated failed to apply to the schema.

## Error Message:
%s
%s

## Current Schema State:
- element_entity_mappings: %d items (valid indices: 0-%d)
- entity_relationship_rules: %d items (valid indices: 0-%d)

## Your Failed Patch:
%s

## Current Schema (for reference):
%s

## Instructions:
Generate a CORRECTED RFC 6902 JSON Patch that fixes the specific error above.
- Review the error message and apply the specific guidance
- An error with the phrase "missing key" indicates a path does not exist. Use "add" for paths that do not exist.
- Use valid array indices only

Return ONLY the JSON patch array - no explanations.`,
		errorMsg,
		specificGuidance,
		len(schema.ElementEntityMappings), len(schema.ElementEntityMappings)-1,
		len(schema.EntityRelationshipRules), len(schema.EntityRelationshipRules)-1,
		string(failedPatchJSON),
		string(schemaJSON))

	options := LLMOptions{
		MaxTokens:   b.config.LLMMaxTokens * 4,
		Temperature: 0.1, // Lower temperature for correction
		Prefill:     "[",
	}

	response, err := b.llmClient.Complete(ctx, prompt, options)
	if err != nil {
		return nil, fmt.Errorf("LLM query failed: %w", err)
	}

	// Save debug output
	if b.config.DebugMode {
		debugPath := filepath.Join(b.config.DebugOutputDir,
			fmt.Sprintf("validation_fix_attempt_%d_patch_correction_%d.json", validationAttempt, patchRetry))
		if err := os.WriteFile(debugPath, []byte(response), 0644); err != nil {
			fmt.Printf("⚠️  Failed to write patch correction debug file: %v\n", err)
		} else {
			fmt.Printf("📝 Saved corrected patch %d.%d to %s\n", validationAttempt, patchRetry, debugPath)
		}
	}

	// Parse and validate the corrected patch
	var patchOps []interface{}
	if err := b.extractJSON(response, &patchOps); err != nil {
		return nil, fmt.Errorf("failed to extract JSON from corrected patch: %w", err)
	}

	correctedPatchJSON, err := json.Marshal(patchOps)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal corrected patch: %w", err)
	}

	return correctedPatchJSON, nil
}

// formatValidationErrorsForLLM formats validation errors for LLM prompt
// Extracts array indices from CRITICAL warnings to help LLM generate correct RFC 6902 patches
func (b *OntologyBuilder) formatValidationErrorsForLLM(errors *ValidationErrors) string {
	var result strings.Builder

	if len(errors.SchemaErrors) > 0 {
		result.WriteString("Schema Errors:\n")
		for i, err := range errors.SchemaErrors {
			result.WriteString(fmt.Sprintf("  %d. %s\n", i+1, err))

			// Parse if this is a CRITICAL duplicate_entity_type error with indices
			// Format: "[CRITICAL] duplicate_entity_type: Entity type 'X' defined N times..."
			if strings.Contains(err, "[CRITICAL] duplicate_entity_type:") {
				// Extract entity type and look it up in current quality validation
				// This will provide RFC 6902 patch instructions
				if strings.Contains(err, "Entity type '") {
					start := strings.Index(err, "Entity type '") + len("Entity type '")
					end := strings.Index(err[start:], "'")
					if end > 0 {
						// Note: The actual indices are already included in the Suggestion field
						// which was extracted from the ValidationWarning and added to the error string
						// So we don't need to extract them here - they're already in the error message
					}
				}
			}
		}
	}

	if len(errors.GraphErrors) > 0 {
		if len(errors.SchemaErrors) > 0 {
			result.WriteString("\n")
		}
		result.WriteString("Graph Errors:\n")
		for i, err := range errors.GraphErrors {
			result.WriteString(fmt.Sprintf("  %d. %s\n", i+1, err))
		}
	}

	return result.String()
}
