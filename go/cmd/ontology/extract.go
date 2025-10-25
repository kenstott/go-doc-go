package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/kennethstott/doculyzer-go-conversion/internal/analytics"
	"github.com/kennethstott/doculyzer-go-conversion/internal/embeddings"
	"github.com/kennethstott/doculyzer-go-conversion/internal/udml/ontology"
)

// runExtract executes the ontology extract subcommand
func runExtract(args []string) {
	// Create new flag set for this subcommand
	fs := flag.NewFlagSet("extract", flag.ExitOnError)

	// Parse command-line flags
	configPath := fs.String("config", "", "Path to configuration file (required)")
	schemaPath := fs.String("schema", "", "Path to ontology schema JSON (required)")
	jobDBPath := fs.String("job-db", "", "Path to SQLite job control database (optional, in-memory if not provided)")
	docBatchSize := fs.Int("doc-batch-size", 500, "Number of documents per extraction task")

	fs.Parse(args[1:])

	// Validate required flags
	if *configPath == "" || *schemaPath == "" {
		fmt.Fprintln(os.Stderr, "Error: --config and --schema are required")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Usage:")
		fs.PrintDefaults()
		os.Exit(1)
	}

	// Load configuration
	config, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Extract storage path from config
	storagePath, err := getStoragePath(config)
	if err != nil {
		log.Fatalf("Failed to get storage path from config: %v", err)
	}

	// Generate unique run ID and worker ID
	runID := fmt.Sprintf("extraction_%d", time.Now().Unix())
	workerID := fmt.Sprintf("worker_%s", generateRandomHex(8))

	log.Printf("========================================")
	log.Printf("DISTRIBUTED ONTOLOGY EXTRACTION")
	log.Printf("========================================")
	log.Printf("  Run ID: %s", runID)
	log.Printf("  Worker ID: %s", workerID)
	log.Printf("  Config: %s", *configPath)
	log.Printf("  Schema: %s", *schemaPath)
	log.Printf("  Storage: %s", storagePath)
	log.Printf("  Doc batch size: %d", *docBatchSize)
	if *jobDBPath != "" {
		log.Printf("  Job DB: %s", *jobDBPath)
	} else {
		log.Printf("  Job DB: in-memory (single worker mode)")
	}
	log.Printf("========================================\n")

	// Load ontology schema
	log.Println("📖 Loading ontology schema...")
	schemaData, err := os.ReadFile(*schemaPath)
	if err != nil {
		log.Fatalf("Failed to read schema file: %v", err)
	}

	var schema ontology.OntologySchema
	if err = json.Unmarshal(schemaData, &schema); err != nil {
		log.Fatalf("Failed to parse schema JSON: %v", err)
	}

	log.Printf("  ✓ Loaded schema: %s (version %s)", schema.Name, schema.Version)
	log.Printf("    - Domains: %d", len(schema.Domains))
	log.Printf("    - Entity mappings: %d", len(schema.ElementEntityMappings))
	log.Printf("    - Relationship rules: %d\n", len(schema.EntityRelationshipRules))

	// Initialize storage
	log.Println("📊 Initializing storage...")
	storage, err := analytics.NewHiveParquetStorage(map[string]interface{}{
		"path":    storagePath,
		"version": "v2.0.0",
	})
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	defer storage.Close()

	// Initialize job control
	var jobControl ontology.ExtractionJobControl
	if *jobDBPath != "" {
		// SQLite-based job control for multi-process coordination
		jobControl, err = ontology.NewSQLiteJobControl(*jobDBPath)
		if err != nil {
			log.Fatalf("Failed to initialize job control: %v", err)
		}
		defer jobControl.(*ontology.SQLiteJobControl).Close()
	} else {
		// In-memory job control for single worker mode
		jobControl = ontology.NewMemoryExtractionJobControl()
	}

	// Attempt to claim leader role (keep isLeader in scope for later use)
	var isLeader bool
	isLeader, err = jobControl.ClaimLeaderRole(runID, workerID)
	if err != nil {
		log.Fatalf("Failed to claim leader role: %v", err)
	}

	if isLeader {
		log.Printf("🎯 This worker is the LEADER - creating extraction tasks")

		// Leader creates all extraction tasks
		if err := createExtractionTasks(jobControl, storage, schema, runID, *docBatchSize); err != nil {
			log.Fatalf("Failed to create extraction tasks: %v", err)
		}

		log.Printf("  ✓ Leader finished creating tasks\n")
	} else {
		log.Printf("⚙️ This worker is a FOLLOWER - waiting for tasks\n")
	}

	// Collect all reference concepts from semantic filters
	log.Println("📝 Collecting reference concepts from schema...")
	referenceConcepts := collectReferenceConcepts(schema)
	log.Printf("  Found %d unique reference concepts requiring embeddings\n", len(referenceConcepts))

	// Generate embeddings for reference concepts
	conceptEmbeddings := make(map[string][]float64)
	if len(referenceConcepts) > 0 {
		log.Println("🔢 Generating embeddings for reference concepts...")
		var err error
		conceptEmbeddings, err = generateConceptEmbeddings(referenceConcepts)
		if err != nil {
			log.Printf("⚠️  WARNING: Failed to generate concept embeddings: %v", err)
			log.Println("   Semantic filtering will be disabled")
		} else {
			log.Printf("  ✓ Generated %d concept embeddings for semantic filtering\n", len(conceptEmbeddings))
		}
	}

	// All workers (including leader) process tasks
	log.Println("🔍 Starting distributed extraction...")
	if err := processExtractionTasks(jobControl, storage, schema, runID, workerID, conceptEmbeddings); err != nil {
		log.Fatalf("Failed to process extraction tasks: %v", err)
	}

	// Phase 3: Entity Consolidation (leader only)
	if isLeader {
		log.Println("")
		log.Println("🔧 Leader starting entity consolidation...")

		// Initialize LLM validator for false positive validation (optional)
		var llmValidator interface{}
		apiKey := os.Getenv("ANTHROPIC_API_KEY")

		// Determine which model to use for validation
		validationModel := schema.LLMValidationModel
		if validationModel == "" {
			validationModel = schema.LLMModel // Fall back to general llm_model
		}

		// LLM validation requires both API key AND model configuration
		if apiKey != "" && validationModel != "" {
			llmClient := ontology.NewAnthropicClient(apiKey, validationModel)
			llmValidator = ontology.NewLLMValidator(llmClient)
			log.Printf("  ✓ LLM validation enabled (model: %s)", llmClient.GetModel())
		} else {
			if apiKey == "" {
				log.Println("  ⚠ LLM validation disabled (ANTHROPIC_API_KEY not set)")
			} else {
				log.Println("  ⚠ LLM validation disabled (llm_model not configured in schema)")
			}
		}

		if err := storage.ConsolidateEntities(runID, "max_confidence", llmValidator); err != nil {
			log.Fatalf("Failed to consolidate entities: %v", err)
		}
		log.Println("  ✓ Entity consolidation complete")
	}

	log.Println("")
	log.Println("========================================")
	log.Println("EXTRACTION COMPLETE")
	log.Println("========================================")
	log.Printf("Worker %s finished", workerID)
	log.Printf("Run ID: %s", runID)
	log.Println("")
	log.Println("Results written to storage:")
	log.Printf("  - Raw entities: %s/ontology_entities/run_id=%s/", storagePath, runID)
	if isLeader {
		log.Printf("  - Canonical entities: %s/canonical_entities/run_id=%s/", storagePath, runID)
	}
	log.Printf("  - Relationships: %s/ontology_relationships/run_id=%s/", storagePath, runID)
	log.Println("========================================")
}

// generateRandomHex generates a random hex string of length n
func generateRandomHex(n int) string {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based random if crypto/rand fails
		return fmt.Sprintf("%x", time.Now().UnixNano())[:n]
	}
	return hex.EncodeToString(bytes)[:n]
}

// createExtractionTasks creates all extraction tasks for distributed processing
// The leader worker calls this once at the start of an extraction run
func createExtractionTasks(
	jobControl ontology.ExtractionJobControl,
	storage analytics.Storage,
	schema ontology.OntologySchema,
	runID string,
	docBatchSize int,
) error {
	// Get all document IDs from storage
	docIDs, err := storage.GetAllDocIDs(make(map[string]interface{}))
	if err != nil {
		return fmt.Errorf("failed to get doc IDs: %w", err)
	}

	log.Printf("  Found %d documents in corpus", len(docIDs))

	var tasks []ontology.ExtractionTask
	taskCounter := 0

	// Collect unique entity types from mappings
	entityTypes := make(map[string]bool)
	for _, mapping := range schema.ElementEntityMappings {
		entityTypes[string(mapping.EntityType)] = true
	}

	// Create entity extraction tasks - one per (entity_type, doc_batch)
	for entityType := range entityTypes {
		// Split documents into batches
		for i := 0; i < len(docIDs); i += docBatchSize {
			end := i + docBatchSize
			if end > len(docIDs) {
				end = len(docIDs)
			}
			docBatch := docIDs[i:end]

			task := ontology.ExtractionTask{
				ID:         fmt.Sprintf("entity_%s_%d", entityType, taskCounter),
				RunID:      runID,
				Type:       ontology.TaskTypeEntityMapping,
				EntityType: entityType,
				DocIDs:     docBatch,
				Status:     ontology.TaskStatusPending,
				CreatedAt:  time.Now(),
			}
			tasks = append(tasks, task)
			taskCounter++
		}
	}

	log.Printf("  Created %d entity extraction tasks", len(tasks))

	// Create relationship extraction tasks
	relationshipTasks := 0
	for _, rule := range schema.EntityRelationshipRules {
		relType := string(rule.RelationshipType)

		for i := 0; i < len(docIDs); i += docBatchSize {
			end := i + docBatchSize
			if end > len(docIDs) {
				end = len(docIDs)
			}
			docBatch := docIDs[i:end]

			task := ontology.ExtractionTask{
				ID:               fmt.Sprintf("rel_%s_%d", relType, taskCounter),
				RunID:            runID,
				Type:             ontology.TaskTypeRelationshipRule,
				RelationshipType: relType,
				DocIDs:           docBatch,
				Status:           ontology.TaskStatusPending,
				CreatedAt:        time.Now(),
			}
			tasks = append(tasks, task)
			taskCounter++
			relationshipTasks++
		}
	}

	log.Printf("  Created %d relationship extraction tasks", relationshipTasks)
	log.Printf("  Total tasks: %d", len(tasks))

	return jobControl.CreateTasks(runID, tasks)
}

// processExtractionTasks is the main worker loop
// Claims and processes tasks until all tasks are complete or idle timeout is reached
func processExtractionTasks(
	jobControl ontology.ExtractionJobControl,
	storage analytics.Storage,
	schema ontology.OntologySchema,
	runID string,
	workerID string,
	conceptEmbeddings map[string][]float64,
) error {
	const idleTimeout = 2 * time.Minute
	lastActivityTime := time.Now()

	log.Printf("Worker %s starting task processing loop", workerID)

	for {
		// Check idle timeout
		if time.Since(lastActivityTime) > idleTimeout {
			log.Printf("Worker %s idle timeout reached, terminating", workerID)
			break
		}

		// Phase 1: Process entity extraction tasks first
		entityComplete, err := jobControl.IsPhaseComplete(runID, ontology.TaskTypeEntityMapping)
		if err != nil {
			return fmt.Errorf("failed to check entity phase completion: %w", err)
		}

		if !entityComplete {
			// Try to claim an entity task
			task, err := jobControl.ClaimTask(runID, ontology.TaskTypeEntityMapping, workerID)
			if err != nil {
				// No pending tasks - wait and retry
				time.Sleep(1 * time.Second)
				continue
			}

			log.Printf("Worker %s claimed entity task: %s", workerID, task.ID)
			lastActivityTime = time.Now()

			// Process the entity task
			if err := processEntityTask(storage, schema, task, conceptEmbeddings); err != nil {
				log.Printf("Worker %s failed entity task %s: %v", workerID, task.ID, err)
				jobControl.FailTask(task.ID, err.Error())
				continue
			}

			// Mark task as completed
			result := ontology.ExtractionTaskResult{
				EntitiesExtracted: 0, // TODO: track actual count
			}
			if err := jobControl.CompleteTask(task.ID, result); err != nil {
				log.Printf("Worker %s failed to mark task complete: %v", workerID, err)
			}
			log.Printf("Worker %s completed entity task: %s", workerID, task.ID)
			continue
		}

		// Phase 2: Process relationship extraction tasks (after entities complete)
		relComplete, err := jobControl.IsPhaseComplete(runID, ontology.TaskTypeRelationshipRule)
		if err != nil {
			return fmt.Errorf("failed to check relationship phase completion: %w", err)
		}

		if !relComplete {
			// Try to claim a relationship task
			task, err := jobControl.ClaimTask(runID, ontology.TaskTypeRelationshipRule, workerID)
			if err != nil {
				// No pending tasks - wait and retry
				time.Sleep(1 * time.Second)
				continue
			}

			log.Printf("Worker %s claimed relationship task: %s", workerID, task.ID)
			lastActivityTime = time.Now()

			// Process the relationship task
			if err := processRelationshipTask(storage, schema, task, conceptEmbeddings); err != nil {
				log.Printf("Worker %s failed relationship task %s: %v", workerID, task.ID, err)
				jobControl.FailTask(task.ID, err.Error())
				continue
			}

			// Mark task as completed
			result := ontology.ExtractionTaskResult{
				RelationshipsExtracted: 0, // TODO: track actual count
			}
			if err := jobControl.CompleteTask(task.ID, result); err != nil {
				log.Printf("Worker %s failed to mark task complete: %v", workerID, err)
			}
			log.Printf("Worker %s completed relationship task: %s", workerID, task.ID)
			continue
		}

		// Both phases complete
		log.Printf("Worker %s: All tasks complete", workerID)
		break
	}

	return nil
}

// processEntityTask processes a single entity extraction task
// Runs ALL mappings for the entity type to extract maximum coverage
// Deduplication/consolidation happens later in the canonicalization phase
func processEntityTask(
	storage analytics.Storage,
	schema ontology.OntologySchema,
	task *ontology.ExtractionTask,
	conceptEmbeddings map[string][]float64,
) error {
	// Find all mappings for this entity type
	var mappings []*ontology.ElementEntityMapping
	for i := range schema.ElementEntityMappings {
		if string(schema.ElementEntityMappings[i].EntityType) == task.EntityType {
			mappings = append(mappings, &schema.ElementEntityMappings[i])
		}
	}

	if len(mappings) == 0 {
		return fmt.Errorf("no mappings found for entity type %s", task.EntityType)
	}

	// Sort by confidence descending (highest confidence first - for logging clarity)
	sort.Slice(mappings, func(i, j int) bool {
		return mappings[i].Confidence > mappings[j].Confidence
	})

	log.Printf("  Running %d mappings for entity type %s...", len(mappings), task.EntityType)

	totalExtracted := 0
	successfulMappings := 0

	// Run ALL mappings - no short-circuiting
	for idx, mapping := range mappings {
		log.Printf("  Running mapping %d/%d: confidence=%.2f", idx+1, len(mappings), mapping.Confidence)

		// Marshal mapping to JSON
		mappingJSON, err := json.Marshal(mapping)
		if err != nil {
			log.Printf("  ⚠️  Failed to marshal mapping: %v", err)
			continue
		}

		// Call ExtractAndStoreEntities
		filters := make(map[string]interface{})
		entityCount, err := storage.ExtractAndStoreEntities(
			task.RunID,
			task.EntityType,
			task.DocIDs,
			mappingJSON,
			filters,
			conceptEmbeddings,
		)

		if err != nil {
			log.Printf("  ⚠️  Mapping failed with error: %v", err)
			continue
		}

		if entityCount > 0 {
			log.Printf("  ✓ Mapping extracted %d entities at confidence %.2f", entityCount, mapping.Confidence)
			totalExtracted += entityCount
			successfulMappings++
		} else {
			log.Printf("  ℹ️  Mapping extracted 0 entities at confidence %.2f", mapping.Confidence)
		}
	}

	if totalExtracted == 0 {
		log.Printf("  ⚠️  All %d mappings extracted 0 entities for entity type %s", len(mappings), task.EntityType)
		return nil // Not an error - just no matches
	}

	log.Printf("  ✓ Completed entity type %s: %d total raw entities from %d/%d mappings",
		task.EntityType, totalExtracted, successfulMappings, len(mappings))
	return nil
}

// processRelationshipTask processes a single relationship extraction task
func processRelationshipTask(
	storage analytics.Storage,
	schema ontology.OntologySchema,
	task *ontology.ExtractionTask,
	conceptEmbeddings map[string][]float64,
) error {
	// TODO: Implement relationship extraction
	// For now, just return nil as a stub
	log.Printf("Relationship extraction not yet implemented for task: %s", task.ID)
	return nil
}

// collectReferenceConcepts extracts all unique reference concepts from semantic filters in the schema
func collectReferenceConcepts(schema ontology.OntologySchema) []string {
	conceptsMap := make(map[string]bool)

	// Iterate through all entity mappings
	for _, mapping := range schema.ElementEntityMappings {
		// Check each extraction rule for semantic_filter
		for _, rule := range mapping.ExtractionRules {
			if rule.SemanticFilter != nil {
				// Handle reference_text (single string)
				if rule.SemanticFilter.ReferenceText != "" {
					conceptsMap[rule.SemanticFilter.ReferenceText] = true
				}

				// Handle reference_concepts (array)
				for _, concept := range rule.SemanticFilter.ReferenceConcepts {
					conceptsMap[concept] = true
				}
			}
		}
	}

	// Convert map to slice
	concepts := make([]string, 0, len(conceptsMap))
	for concept := range conceptsMap {
		concepts = append(concepts, concept)
	}

	return concepts
}

// generateConceptEmbeddings generates embeddings for reference concepts using the embedding model
func generateConceptEmbeddings(concepts []string) (map[string][]float64, error) {
	if len(concepts) == 0 {
		return make(map[string][]float64), nil
	}

	// Create embedding generator with default ONNX model
	config := embeddings.Config{
		Provider:   "onnx",
		ModelPath:  filepath.Join("go", "models", "all-MiniLM-L6-v2"),
		Dimensions: 384,
		BatchSize:  32,
	}

	generator, err := embeddings.CreateEmbeddingGenerator(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding generator: %w", err)
	}
	defer generator.Close()

	// Generate embeddings in batch
	embeddings, err := generator.GenerateBatch(concepts)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embeddings: %w", err)
	}

	// Create map from concepts to embeddings
	result := make(map[string][]float64, len(concepts))
	for i, concept := range concepts {
		result[concept] = embeddings[i]
	}

	return result, nil
}