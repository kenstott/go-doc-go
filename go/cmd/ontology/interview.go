package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/kennethstott/doculyzer-go-conversion/internal/udml/ontology"
	"gopkg.in/yaml.v3"
)

// runInterview executes the ontology interview subcommand
func runInterview(args []string) {
	// Prepend command name for compatibility with existing logic
	os.Args = append([]string{"ontology interview"}, args[1:]...)

	if len(os.Args) < 2 {
		printInterviewUsage()
		os.Exit(1)
	}

	// Check for mode and option flags
	mode := ontology.ModeNewOntology
	existingSchemaPath := ""
	configPath := ""
	nonInteractive := false
	diversityThreshold := 0.0 // 0.0 means use default (0.85)
	enableMCP := false
	embeddingModel := "all-MiniLM-L6-v2" // Default embedding model
	outputPath := "ontology.json"
	argsOffset := 1

	// Parse flags
	for argsOffset < len(os.Args) && strings.HasPrefix(os.Args[argsOffset], "--") {
		switch os.Args[argsOffset] {
		case "--config":
			if len(os.Args) < argsOffset+2 {
				fmt.Fprintf(os.Stderr, "Error: --config requires a path to configuration file\n\n")
				printInterviewUsage()
				os.Exit(1)
			}
			configPath = os.Args[argsOffset+1]
			argsOffset += 2
		case "--refine":
			mode = ontology.ModeRefineOntology
			if len(os.Args) < argsOffset+2 {
				fmt.Fprintf(os.Stderr, "Error: --refine requires <existing_schema>\n\n")
				printInterviewUsage()
				os.Exit(1)
			}
			existingSchemaPath = os.Args[argsOffset+1]
			argsOffset += 2
		case "--output":
			if len(os.Args) < argsOffset+2 {
				fmt.Fprintf(os.Stderr, "Error: --output requires a path\n\n")
				printInterviewUsage()
				os.Exit(1)
			}
			outputPath = os.Args[argsOffset+1]
			argsOffset += 2
		case "--non-interactive":
			nonInteractive = true
			argsOffset++
		case "--diversity-threshold":
			if len(os.Args) < argsOffset+2 {
				fmt.Fprintf(os.Stderr, "Error: --diversity-threshold requires a value (0.0-1.0)\n\n")
				printInterviewUsage()
				os.Exit(1)
			}
			var err error
			diversityThreshold, err = strconv.ParseFloat(os.Args[argsOffset+1], 64)
			if err != nil || diversityThreshold < 0.0 || diversityThreshold > 1.0 {
				fmt.Fprintf(os.Stderr, "Error: --diversity-threshold must be a number between 0.0 and 1.0\n\n")
				printInterviewUsage()
				os.Exit(1)
			}
			argsOffset += 2
		case "--enable-mcp":
			enableMCP = true
			argsOffset++
		case "--embedding-model":
			if len(os.Args) < argsOffset+2 {
				fmt.Fprintf(os.Stderr, "Error: --embedding-model requires a model name\n\n")
				printInterviewUsage()
				os.Exit(1)
			}
			embeddingModel = os.Args[argsOffset+1]
			argsOffset += 2
		default:
			fmt.Fprintf(os.Stderr, "Error: unknown flag %s\n\n", os.Args[argsOffset])
			printInterviewUsage()
			os.Exit(1)
		}
	}

	// Validate required flags
	if configPath == "" {
		fmt.Fprintf(os.Stderr, "Error: --config is required\n\n")
		printInterviewUsage()
		os.Exit(1)
	}

	// Load configuration
	config, err := loadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Extract storage path from config
	storagePath, err := getStoragePath(config)
	if err != nil {
		log.Fatalf("Failed to get storage path from config: %v", err)
	}

	// Check for API key
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		log.Fatal("Error: ANTHROPIC_API_KEY environment variable not set")
	}

	// Create builder config
	builderConfig := ontology.BuilderConfig{
		StoragePath:        storagePath,
		SampleSize:         1000,
		DiversityThreshold: diversityThreshold, // 0.0 uses default of 0.85 (filters out >85% similar samples)
		LLMProvider:        "anthropic",
		LLMModel:           "claude-sonnet-4-5-20250929",
		LLMAPIKey:          apiKey,
		LLMMaxTokens:       4096,
		SchemaName:         "InteractiveOntology",
		SchemaVersion:      "1.0.0",
		EnableMCP:          enableMCP,
		EmbeddingModel:     embeddingModel,
	}

	// Create builder
	builder, err := ontology.NewOntologyBuilder(builderConfig)
	if err != nil {
		log.Fatalf("Failed to create builder: %v", err)
	}
	defer builder.Close()

	// Create interview builder with mode
	interview := ontology.NewInterviewBuilderV2(builder, mode)
	interview.SetNonInteractive(nonInteractive)

	ctx := context.Background()
	var schema *ontology.OntologySchema

	if mode == ontology.ModeRefineOntology {
		// Load existing schema
		existingData, err := os.ReadFile(existingSchemaPath)
		if err != nil {
			log.Fatalf("Failed to read existing schema: %v", err)
		}

		var existingSchema ontology.OntologySchema
		if err := json.Unmarshal(existingData, &existingSchema); err != nil {
			log.Fatalf("Failed to parse existing schema: %v", err)
		}

		// Start refinement interview
		schema, err = interview.StartRefinementInterview(ctx, &existingSchema)
		if err != nil {
			log.Fatalf("Refinement interview failed: %v", err)
		}
	} else {
		// Start new ontology interview
		schema, err = interview.StartInterview(ctx)
		if err != nil {
			log.Fatalf("Interview failed: %v", err)
		}
	}

	// Validate schema (computes hierarchies and fills parent_type/children/w_category fields)
	fmt.Println("\n🔄 Validating and finalizing schema...")
	if err := schema.Validate(); err != nil {
		log.Fatalf("Schema validation failed: %v", err)
	}
	fmt.Println("✅ Schema validation passed (hierarchies computed)")

	// Save schema to file (YAML or JSON based on extension)
	ext := filepath.Ext(outputPath)
	var schemaBytes []byte
	var format string

	if ext == ".yaml" || ext == ".yml" {
		schemaBytes, err = yaml.Marshal(schema)
		format = "YAML"
	} else {
		schemaBytes, err = json.MarshalIndent(schema, "", "  ")
		format = "JSON"
	}

	if err != nil {
		log.Fatalf("Failed to marshal schema to %s: %v", format, err)
	}

	if err := os.WriteFile(outputPath, schemaBytes, 0644); err != nil {
		log.Fatalf("Failed to write schema file: %v", err)
	}

	fmt.Printf("\n✓ Ontology schema saved to: %s (%s format)\n", outputPath, format)
}

func printInterviewUsage() {
	fmt.Fprintf(os.Stderr, `Interactive Ontology Builder

MODES:

1. NEW ONTOLOGY (default)
   Creates a new ontology from scratch with mandatory user approval.

   Usage:
     %s --config <config_file> [options]

   Required:
     --config <path>  Path to configuration file (same config used by worker)

   Options:
     --output <path>                Output path for ontology schema (default: ontology.json)
     --diversity-threshold <value>  Cosine similarity threshold for diversity filtering (0.0-1.0)
                                    Lower values = more diverse samples. Default: 0.85
     --non-interactive              Auto-approve all suggestions (for testing)
     --enable-mcp                   Enable MCP corpus exploration tools for LLM
     --embedding-model <model>      Embedding model for semantic search (default: all-MiniLM-L6-v2)

   Process (3 phases):
     Phase 1: Domain Selection
       - LLM analyzes corpus and suggests domains
       - You confirm or adjust domain selection

     Phase 2: Draft Generation
       - LLM automatically generates entity types and relationships
       - Based on corpus analysis and confirmed domains

     Phase 3: Review & Confirmation (MANDATORY)
       - Review complete schema
       - Options: approve, refine, or reject
       - No schema generated without your approval

2. REFINEMENT MODE
   Refines an existing ontology by comparing it against the corpus.

   Usage:
     %s --config <config_file> --refine <existing_schema> [--output <output_path>]

   Required:
     --config <path>  Path to configuration file
     --refine <path>  Path to existing ontology JSON file to refine

   Optional:
     --output <path>  Output path for refined schema (default: ontology.json)

   Process (3 phases):
     Phase 1: Analysis
       - LLM compares existing schema vs corpus
       - Identifies gaps, issues, and improvements

     Phase 2: Suggested Changes
       - LLM proposes specific changes
       - You approve/reject each suggestion individually

     Phase 3: Review & Confirmation (MANDATORY)
       - Review refined schema
       - Options: approve, further refine, or reject

Environment:
  ANTHROPIC_API_KEY   Required. Your Anthropic API key

Examples:
  # Create new ontology
  export ANTHROPIC_API_KEY=your_key_here
  %s --config config.toml
  %s --config config.toml --output my-ontology.json

  # Refine existing ontology
  %s --config config.toml --refine current-ontology.json --output improved-ontology.json

Features:
  ✓ LLM-guided conversation
  ✓ Mandatory user approval (no auto-apply)
  ✓ Domain-driven design with ownership
  ✓ Iterative refinement
  ✓ Two workflows: create new or refine existing

The interview typically takes 5-10 minutes and results in a high-quality,
customized ontology schema that YOU control.
`, os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0])
}
