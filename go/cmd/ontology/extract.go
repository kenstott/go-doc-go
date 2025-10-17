package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/kennethstott/doculyzer-go-conversion/internal/analytics"
	"github.com/kennethstott/doculyzer-go-conversion/internal/udml/ontology"
)

// runExtract executes the ontology extract subcommand
func runExtract(args []string) {
	// Create new flag set for this subcommand
	fs := flag.NewFlagSet("extract", flag.ExitOnError)

	// Parse command-line flags
	schemaPath := fs.String("schema", "", "Path to ontology schema JSON (required)")
	parquetPath := fs.String("parquet", "", "Path to UDML Parquet storage (required)")
	outputPath := fs.String("output", "", "Output path for extracted entities/relationships JSON (optional, stdout if not provided)")
	docID := fs.String("doc-id", "", "Filter by document ID (optional)")
	maxElements := fs.Int("max-elements", 0, "Maximum number of elements to process (0 = all)")
	distributed := fs.Bool("distributed", false, "Enable distributed extraction mode")
	workers := fs.Int("workers", 4, "Number of workers for distributed extraction (default: 4)")

	fs.Parse(args[1:])

	// Validate required flags
	if *schemaPath == "" || *parquetPath == "" {
		fmt.Fprintln(os.Stderr, "Error: --schema and --parquet are required")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Usage:")
		fs.PrintDefaults()
		os.Exit(1)
	}

	log.Printf("========================================")
	log.Printf("ONTOLOGY ENTITY EXTRACTION")
	log.Printf("========================================")
	log.Printf("  Schema: %s", *schemaPath)
	log.Printf("  Parquet: %s", *parquetPath)
	if *docID != "" {
		log.Printf("  Doc ID filter: %s", *docID)
	}
	if *maxElements > 0 {
		log.Printf("  Max elements: %d", *maxElements)
	}
	if *distributed {
		log.Printf("  Mode: DISTRIBUTED (%d workers)", *workers)
	} else {
		log.Printf("  Mode: Single-worker")
	}
	log.Printf("========================================\n")

	// Load ontology schema
	log.Println("📖 Loading ontology schema...")
	schemaData, err := os.ReadFile(*schemaPath)
	if err != nil {
		log.Fatalf("Failed to read schema file: %v", err)
	}

	var schema ontology.OntologySchema
	if err := json.Unmarshal(schemaData, &schema); err != nil {
		log.Fatalf("Failed to parse schema JSON: %v", err)
	}

	log.Printf("  ✓ Loaded schema: %s (version %s)\n", schema.Name, schema.Version)
	log.Printf("    - Domains: %d", len(schema.Domains))
	log.Printf("    - Entity mappings: %d", len(schema.ElementEntityMappings))
	log.Printf("    - Relationship rules: %d\n", len(schema.EntityRelationshipRules))

	// Initialize Parquet storage
	log.Println("📊 Initializing Parquet storage...")
	storage, err := analytics.NewHiveParquetStorage(map[string]interface{}{
		"path":    *parquetPath,
		"version": "v2.0.0",
	})
	if err != nil {
		log.Fatalf("Failed to initialize Parquet storage: %v", err)
	}
	defer storage.Close()

	// Query elements from Parquet
	log.Println("🔍 Querying UDML elements from Parquet...")
	filters := make(map[string]interface{})
	if *docID != "" {
		filters["doc_id"] = *docID
	}

	elements, err := storage.QueryElements(filters)
	if err != nil {
		log.Fatalf("Failed to query elements: %v", err)
	}

	// Limit elements if specified
	if *maxElements > 0 && len(elements) > *maxElements {
		elements = elements[:*maxElements]
		log.Printf("  ✓ Limited to %d elements (from %d total)\n", *maxElements, len(elements))
	} else {
		log.Printf("  ✓ Loaded %d elements\n", len(elements))
	}

	// Convert analytics.Element to ontology.Element
	log.Println("🔄 Converting elements to ontology format...")
	ontologyElements := make([]ontology.Element, len(elements))
	for i, elem := range elements {
		ontologyElements[i] = ontology.Element{
			ElementID:       elem.ElementID,
			ElementType:     elem.ElementType,
			Content:         elem.Content,
			ContentPreview:  elem.ContentPreview,
			ContentLocation: elem.ContentLocation,
			ParentID:        elem.ParentID,
			ElementOrder:    elem.ElementOrder,
			Metadata:        elem.Metadata,
		}
	}

	// Get ContentResolver from storage
	var contentResolver ontology.ContentResolver
	if resolverInterface := storage.GetContentResolver(); resolverInterface != nil {
		if cr, ok := resolverInterface.(ontology.ContentResolver); ok {
			contentResolver = cr
		}
	}

	// Use first doc_id or "all" if multiple docs
	targetDocID := "all"
	targetSourceName := "extracted"
	if *docID != "" {
		targetDocID = *docID
	} else if len(elements) > 0 {
		targetDocID = elements[0].DocID
		targetSourceName = elements[0].SourceName
	}

	ctx := context.Background()
	var result *ontology.Ontology

	// Choose extraction mode based on flag
	if *distributed {
		// DISTRIBUTED MODE: Use coordinator with multiple workers
		log.Printf("🤖 Extracting entities and relationships (DISTRIBUTED MODE - %d workers)...", *workers)

		// Create job control (in-memory for now)
		jobControl := ontology.NewMemoryExtractionJobControl()

		// Create coordinator
		coordinator := ontology.NewExtractionCoordinator(
			&schema,
			ontologyElements,
			targetDocID,
			targetSourceName,
			jobControl,
			storage,
			contentResolver,
		)

		// Run distributed extraction
		var err error
		result, err = coordinator.RunDistributedExtraction(ctx, *workers)
		if err != nil {
			log.Fatalf("Failed to extract entities (distributed): %v", err)
		}
	} else {
		// SINGLE-WORKER MODE: Use traditional extractor
		log.Println("🤖 Extracting entities and relationships (SINGLE-WORKER MODE)...")
		extractor := ontology.NewRuleBasedExtractor(&schema, contentResolver)

		var err error
		result, err = extractor.ExtractFromElements(ctx, targetDocID, ontologyElements)
		if err != nil {
			log.Fatalf("Failed to extract entities: %v", err)
		}
	}

	log.Printf("  ✓ Extracted %d entities\n", len(result.Entities))
	log.Printf("  ✓ Extracted %d relationships\n", len(result.Relationships))
	log.Printf("  ✓ Extraction time: %v\n", result.Metadata.ExtractionTime)
	log.Printf("  ✓ Overall confidence: %.2f\n\n", result.Metadata.Confidence)

	// Output results
	log.Println("📝 Outputting results...")

	// Create summary structure
	output := map[string]interface{}{
		"ontology_id":   result.ID,
		"doc_id":        result.DocID,
		"schema_name":   result.Name,
		"schema_version": result.Version,
		"metadata": map[string]interface{}{
			"extractor_type":    result.Metadata.ExtractorType,
			"extractor_version": result.Metadata.ExtractorVersion,
			"extraction_time":   result.Metadata.ExtractionTime.String(),
			"element_count":     result.Metadata.ElementCount,
			"confidence":        result.Metadata.Confidence,
		},
		"entities":      result.Entities,
		"relationships": result.Relationships,
		"statistics": map[string]interface{}{
			"total_entities":      len(result.Entities),
			"total_relationships": len(result.Relationships),
			"entities_by_type":    countByType(result.Entities),
			"entities_by_domain":  countByDomain(result.Entities),
		},
	}

	outputJSON, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal output: %v", err)
	}

	if *outputPath != "" {
		// Write to file
		if err := os.WriteFile(*outputPath, outputJSON, 0644); err != nil {
			log.Fatalf("Failed to write output file: %v", err)
		}
		log.Printf("  ✓ Saved extraction results to: %s\n", *outputPath)
	} else {
		// Write to stdout
		fmt.Println(string(outputJSON))
	}

	// Print summary to stderr (so it doesn't interfere with JSON output)
	log.Println("")
	log.Println("========================================")
	log.Println("EXTRACTION COMPLETE")
	log.Println("========================================")
	log.Printf("Entities extracted: %d", len(result.Entities))
	log.Printf("Relationships extracted: %d", len(result.Relationships))
	log.Println("")
	log.Println("Top Entity Types:")
	typeCounts := countByType(result.Entities)
	for entityType, count := range typeCounts {
		log.Printf("  - %s: %d", entityType, count)
	}
	log.Println("")
	log.Println("Domains:")
	domainCounts := countByDomain(result.Entities)
	for domain, count := range domainCounts {
		log.Printf("  - %s: %d entities", domain, count)
	}
	log.Println("========================================")
}

func countByType(entities []ontology.Entity) map[string]int {
	counts := make(map[string]int)
	for _, entity := range entities {
		counts[string(entity.Type)]++
	}
	return counts
}

func countByDomain(entities []ontology.Entity) map[string]int {
	counts := make(map[string]int)
	for _, entity := range entities {
		counts[string(entity.Domain)]++
	}
	return counts
}