package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/kennethstott/doculyzer-go-conversion/internal/analytics"
	"github.com/kennethstott/doculyzer-go-conversion/internal/graph"
	"github.com/kennethstott/doculyzer-go-conversion/internal/udml/ontology"
)

// runAnalyzeGraph executes the ontology analyze-graph subcommand
func runAnalyzeGraph(args []string) {
	// Create new flag set for this subcommand
	fs := flag.NewFlagSet("analyze-graph", flag.ExitOnError)

	// Parse command-line flags
	parquetPath := fs.String("parquet", "", "Path to UDML Parquet storage (required)")
	runID := fs.String("run-id", "", "Run ID for extraction results (required)")
	outputGOB := fs.String("output-gob", "", "Output path for graph in GOB format (optional)")
	outputJSON := fs.String("output-json", "", "Output path for graph metadata in JSON format (optional)")
	algorithm := fs.String("algorithm", "louvain", "Community detection algorithm: louvain, label_propagation")
	enrichCommunities := fs.Bool("enrich-communities", true, "Add explicit community nodes to graph")
	statsOnly := fs.Bool("stats-only", false, "Only print statistics, don't write output files")

	fs.Parse(args[1:])

	// Validate required flags
	if *parquetPath == "" || *runID == "" {
		fmt.Fprintln(os.Stderr, "Error: --parquet and --run-id are required")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Usage:")
		fs.PrintDefaults()
		os.Exit(1)
	}

	log.Printf("========================================")
	log.Printf("KNOWLEDGE GRAPH ANALYSIS")
	log.Printf("========================================")
	log.Printf("  Parquet: %s", *parquetPath)
	log.Printf("  Run ID: %s", *runID)
	log.Printf("  Algorithm: %s", *algorithm)
	log.Printf("  Enrich communities: %v", *enrichCommunities)
	log.Printf("========================================\n")

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

	// Load ontology from Parquet
	log.Println("📖 Loading ontology from Parquet...")
	ont, err := loadOntologyFromParquet(storage, *runID)
	if err != nil {
		log.Fatalf("Failed to load ontology: %v", err)
	}

	log.Printf("  ✓ Loaded ontology:")
	log.Printf("    - Entities: %d", len(ont.Entities))
	log.Printf("    - Relationships: %d\n", len(ont.Relationships))

	// Convert to graph
	log.Println("🔄 Converting ontology to knowledge graph...")
	converter := graph.NewOntologyConverter()
	converter.IncludeProvenance = true
	converter.IncludeMentions = true

	kg, err := converter.ConvertToGraph(ont)
	if err != nil {
		log.Fatalf("Failed to convert to graph: %v", err)
	}

	log.Printf("  ✓ Knowledge graph created:")
	log.Printf("    - Nodes: %d", len(kg.Nodes))
	log.Printf("    - Edges: %d\n", len(kg.Edges))

	// Convert to gonum graph for analysis
	log.Println("🔍 Preparing graph for analysis...")
	gonumGraph, err := graph.LoadFromGraph(kg)
	if err != nil {
		log.Fatalf("Failed to load graph: %v", err)
	}

	// Run community detection
	log.Printf("🧩 Running community detection (%s)...", *algorithm)
	var communityAlg graph.CommunityDetectionAlgorithm
	switch *algorithm {
	case "louvain":
		communityAlg = graph.AlgorithmLouvain
	case "label_propagation":
		communityAlg = graph.AlgorithmLabelPropagation
	default:
		log.Fatalf("Unknown algorithm: %s (use 'louvain' or 'label_propagation')", *algorithm)
	}

	communities, err := graph.DetectCommunities(gonumGraph, communityAlg)
	if err != nil {
		log.Fatalf("Community detection failed: %v", err)
	}

	log.Printf("  ✓ Community detection complete:")
	log.Printf("    - Communities found: %d", len(communities.Communities))
	log.Printf("    - Modularity: %.4f", communities.Modularity)

	// Print top communities
	if len(communities.Communities) > 0 {
		log.Println("")
		log.Println("  Top communities by size:")
		topCommunities := communities.GetLargestCommunities(5)
		for i, comm := range topCommunities {
			log.Printf("    %d. Community %d: %d nodes", i+1, comm.ID, comm.Size)
		}
	}

	// Enrich graph with community nodes if requested
	var enrichedGraph *graph.Graph
	if *enrichCommunities {
		log.Println("")
		log.Println("🌟 Enriching graph with community nodes...")
		enrichedGraph = graph.AddCommunityNodes(kg, communities)

		// Get enrichment stats
		stats := graph.GetCommunityEnrichmentStats(kg, enrichedGraph)
		log.Printf("  ✓ Graph enrichment complete:")
		log.Printf("    - Original nodes: %d", stats.OriginalNodeCount)
		log.Printf("    - Enriched nodes: %d", stats.EnrichedNodeCount)
		log.Printf("    - Community nodes added: %d", stats.CommunityNodeCount)
		log.Printf("    - BELONGS_TO edges added: %d", stats.BelongsToEdgeCount)
		log.Printf("    - Avg community size: %.1f nodes", stats.AvgCommunitySize)
	} else {
		enrichedGraph = kg
	}

	// Create query engine for analysis
	log.Println("")
	log.Println("🔎 Analyzing graph structure...")
	queryEngine := graph.NewGraphQueryEngine(enrichedGraph)

	// Count nodes by label
	entityNodes := queryEngine.FindNodesByLabel("Entity", graph.DefaultQueryOptions())
	log.Printf("  Entity nodes: %d", len(entityNodes))

	if *enrichCommunities {
		communityNodes := queryEngine.FindNodesByLabel("Community", graph.DefaultQueryOptions())
		log.Printf("  Community nodes: %d", len(communityNodes))
	}

	// Count nodes by domain
	domains := make(map[string]int)
	for _, node := range enrichedGraph.Nodes {
		domains[node.Domain]++
	}
	log.Println("")
	log.Println("  Nodes by domain:")
	for domain, count := range domains {
		log.Printf("    - %s: %d", domain, count)
	}

	// Count edges by type
	edgeTypes := make(map[string]int)
	for _, edge := range enrichedGraph.Edges {
		edgeTypes[edge.Type]++
	}
	log.Println("")
	log.Println("  Edges by type:")
	for edgeType, count := range edgeTypes {
		log.Printf("    - %s: %d", edgeType, count)
	}

	// If stats-only mode, we're done
	if *statsOnly {
		log.Println("")
		log.Println("========================================")
		log.Println("ANALYSIS COMPLETE (stats-only mode)")
		log.Println("========================================")
		return
	}

	// Convert community results to ontology entities/relationships and store in Parquet
	log.Println("")
	log.Println("💾 Storing community structure in Parquet...")
	communityEntities, communityRelationships, err := graph.ConvertCommunityGraphToOntology(
		kg, // Use original graph (before enrichment)
		communities,
		*runID,
	)
	if err != nil {
		log.Fatalf("Failed to convert communities to ontology: %v", err)
	}

	log.Printf("  Converted to ontology format:")
	log.Printf("    - Community entities: %d", len(communityEntities))
	log.Printf("    - Community relationships: %d", len(communityRelationships))

	// Convert to analytics types
	analyticsEntities := make([]analytics.OntologyEntity, len(communityEntities))
	for i, entity := range communityEntities {
		analyticsEntities[i] = analytics.OntologyEntity{
			EntityID:    entity.ID,
			EntityName:  entity.Name,
			EntityType:  string(entity.Type),
			Domain:      entity.Domain,
			Confidence:  entity.Confidence,
			Attributes:  entity.Attributes,
			ElementID:   entity.ElementID,
			RunID:       *runID,
			ExtractedAt: entity.CreatedAt,
		}
	}

	analyticsRelationships := make([]analytics.OntologyRelationship, len(communityRelationships))
	for i, rel := range communityRelationships {
		analyticsRelationships[i] = analytics.OntologyRelationship{
			RelationshipID:   rel.ID,
			SourceEntityID:   rel.SourceID,
			TargetEntityID:   rel.TargetID,
			RelationshipType: string(rel.Type),
			Domain:           rel.Domain,
			Confidence:       rel.Confidence,
			Evidence:         rel.Evidence,
			Attributes:       rel.Attributes,
			ElementID:        rel.ElementID,
			RunID:            *runID,
			ExtractedAt:      rel.CreatedAt,
		}
	}

	// Batch write to Parquet
	if err := storage.AppendOntologyEntities(analyticsEntities); err != nil {
		log.Fatalf("Failed to append community entities: %v", err)
	}

	if err := storage.AppendOntologyRelationships(analyticsRelationships); err != nil {
		log.Fatalf("Failed to append community relationships: %v", err)
	}

	log.Println("  ✓ Community structure stored in Parquet")

	// Save graph in GOB format if requested (optional export)
	if *outputGOB != "" {
		log.Println("")
		log.Printf("💾 Exporting graph to GOB format: %s", *outputGOB)
		if err := graph.SaveGraph(enrichedGraph, *outputGOB); err != nil {
			log.Fatalf("Failed to save graph: %v", err)
		}
		log.Println("  ✓ Graph exported to GOB successfully")
	}

	// Save graph metadata in JSON format if requested
	if *outputJSON != "" {
		log.Println("")
		log.Printf("💾 Saving graph metadata to JSON: %s", *outputJSON)

		// Create metadata structure
		metadata := map[string]interface{}{
			"id":              enrichedGraph.ID,
			"name":            enrichedGraph.Name,
			"version":         enrichedGraph.Version,
			"node_count":      len(enrichedGraph.Nodes),
			"edge_count":      len(enrichedGraph.Edges),
			"domains":         domains,
			"edge_types":      edgeTypes,
			"run_id":          *runID,
			"community_count": len(communities.Communities),
			"modularity":      communities.Modularity,
			"algorithm":       string(communities.Algorithm),
			"metadata":        enrichedGraph.Metadata,
		}

		// Write JSON
		jsonData, err := json.MarshalIndent(metadata, "", "  ")
		if err != nil {
			log.Fatalf("Failed to marshal metadata: %v", err)
		}

		if err := os.WriteFile(*outputJSON, jsonData, 0644); err != nil {
			log.Fatalf("Failed to write JSON: %v", err)
		}
		log.Println("  ✓ Metadata saved successfully")
	}

	log.Println("")
	log.Println("========================================")
	log.Println("GRAPH ANALYSIS COMPLETE")
	log.Println("========================================")
	if *outputGOB != "" {
		log.Printf("  Graph (GOB): %s", *outputGOB)
	}
	if *outputJSON != "" {
		log.Printf("  Metadata (JSON): %s", *outputJSON)
	}
	log.Println("========================================")
}

// loadOntologyFromParquet loads an ontology from Parquet storage
func loadOntologyFromParquet(storage analytics.Storage, runID string) (*ontology.Ontology, error) {
	// Query for canonical entities
	filters := map[string]interface{}{
		"run_id": runID,
	}

	// Get entities - QueryOntologyEntities returns raw extraction entities
	// We need to convert analytics.OntologyEntity to ontology.Entity
	rawEntities, err := storage.QueryOntologyEntities(filters)
	if err != nil {
		return nil, fmt.Errorf("failed to query entities: %w", err)
	}

	// Convert raw entities to ontology entities
	entities := make([]ontology.Entity, 0, len(rawEntities))
	for _, raw := range rawEntities {
		entities = append(entities, ontology.Entity{
			ID:         raw.EntityID,
			Name:       raw.EntityName,
			Type:       ontology.EntityType(raw.EntityType),
			Domain:     raw.Domain,
			Confidence: raw.Confidence,
			Attributes: raw.Attributes,
			ElementID:  raw.ElementID,
			CreatedAt:  raw.ExtractedAt,
			UpdatedAt:  raw.ExtractedAt,
		})
	}

	// Get relationships
	rawRelationships, err := storage.QueryOntologyRelationships(filters)
	if err != nil {
		return nil, fmt.Errorf("failed to query relationships: %w", err)
	}

	// Convert raw relationships to ontology relationships
	relationships := make([]ontology.Relationship, 0, len(rawRelationships))
	for _, raw := range rawRelationships {
		relationships = append(relationships, ontology.Relationship{
			ID:         raw.RelationshipID,
			Type:       ontology.RelationshipType(raw.RelationshipType),
			Domain:     raw.Domain,
			SourceID:   raw.SourceEntityID,
			TargetID:   raw.TargetEntityID,
			Confidence: raw.Confidence,
			Attributes: raw.Attributes,
			ElementID:  raw.ElementID,
			Evidence:   raw.Evidence,
			CreatedAt:  raw.ExtractedAt,
		})
	}

	// Create ontology
	ont := &ontology.Ontology{
		ID:            "graph_analysis_" + runID,
		Name:          "Knowledge Graph from " + runID,
		Version:       "1.0.0",
		Entities:      entities,
		Relationships: relationships,
		Metadata:      ontology.OntologyMeta{},
	}

	return ont, nil
}
