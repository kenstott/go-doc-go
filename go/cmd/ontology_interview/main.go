package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/kennethstott/doculyzer-go-conversion/internal/udml/ontology"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	// Check for mode and option flags
	mode := ontology.ModeNewOntology
	existingSchemaPath := ""
	nonInteractive := false
	diversityThreshold := 0.0 // 0.0 means use default (0.85)
	argsOffset := 1

	// Parse flags
	for argsOffset < len(os.Args) && strings.HasPrefix(os.Args[argsOffset], "--") {
		switch os.Args[argsOffset] {
		case "--refine":
			mode = ontology.ModeRefineOntology
			if len(os.Args) < argsOffset+3 {
				fmt.Fprintf(os.Stderr, "Error: --refine requires <existing_schema> <parquet_path> [output_path]\n\n")
				printUsage()
				os.Exit(1)
			}
			existingSchemaPath = os.Args[argsOffset+1]
			argsOffset += 2
		case "--non-interactive":
			nonInteractive = true
			argsOffset++
		case "--diversity-threshold":
			if len(os.Args) < argsOffset+2 {
				fmt.Fprintf(os.Stderr, "Error: --diversity-threshold requires a value (0.0-1.0)\n\n")
				printUsage()
				os.Exit(1)
			}
			var err error
			diversityThreshold, err = strconv.ParseFloat(os.Args[argsOffset+1], 64)
			if err != nil || diversityThreshold < 0.0 || diversityThreshold > 1.0 {
				fmt.Fprintf(os.Stderr, "Error: --diversity-threshold must be a number between 0.0 and 1.0\n\n")
				printUsage()
				os.Exit(1)
			}
			argsOffset += 2
		default:
			fmt.Fprintf(os.Stderr, "Error: unknown flag %s\n\n", os.Args[argsOffset])
			printUsage()
			os.Exit(1)
		}
	}

	// Parse command line arguments
	if argsOffset >= len(os.Args) {
		fmt.Fprintf(os.Stderr, "Error: missing required argument <parquet_path>\n\n")
		printUsage()
		os.Exit(1)
	}

	parquetPath := os.Args[argsOffset]
	outputPath := "ontology.json"
	if len(os.Args) > argsOffset+1 {
		outputPath = os.Args[argsOffset+1]
	}

	// Check for API key
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		log.Fatal("Error: ANTHROPIC_API_KEY environment variable not set")
	}

	// Create builder config
	config := ontology.BuilderConfig{
		ParquetPath:        parquetPath,
		SampleSize:         1000,
		DiversityThreshold: diversityThreshold, // 0.0 uses default of 0.85 (filters out >85% similar samples)
		LLMProvider:        "anthropic",
		LLMModel:           "claude-sonnet-4-5-20250929",
		LLMAPIKey:          apiKey,
		LLMMaxTokens:       4096,
		SchemaName:         "InteractiveOntology",
		SchemaVersion:      "1.0.0",
	}

	// Create builder
	builder, err := ontology.NewOntologyBuilder(config)
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

	// Save schema to file
	schemaJSON, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal schema: %v", err)
	}

	if err := os.WriteFile(outputPath, schemaJSON, 0644); err != nil {
		log.Fatalf("Failed to write schema file: %v", err)
	}

	fmt.Printf("\n✓ Ontology schema saved to: %s\n", outputPath)
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `Interactive Ontology Builder

MODES:

1. NEW ONTOLOGY (default)
   Creates a new ontology from scratch with mandatory user approval.

   Usage:
     %s [--diversity-threshold <value>] [--non-interactive] <parquet_path> [output_path]

   Arguments:
     parquet_path   Path to UDML Parquet storage directory
     output_path    Output path for ontology schema (default: ontology.json)

   Options:
     --diversity-threshold <value>  Cosine similarity threshold for diversity filtering (0.0-1.0)
                                    Lower values = more diverse samples. Default: 0.85
     --non-interactive              Auto-approve all suggestions (for testing)

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
     %s --refine <existing_schema> <parquet_path> [output_path]

   Arguments:
     existing_schema  Path to existing ontology JSON file
     parquet_path     Path to UDML Parquet storage directory
     output_path      Output path for refined schema (default: ontology.json)

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
  %s ./output/udml.parquet
  %s ./corpus.parquet ./my-ontology.json

  # Refine existing ontology
  %s --refine ./current-ontology.json ./corpus.parquet ./improved-ontology.json

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
