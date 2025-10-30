package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/kennethstott/doculyzer-go-conversion/internal/analytics"
	"github.com/kennethstott/doculyzer-go-conversion/internal/embeddings"
	"github.com/kennethstott/doculyzer-go-conversion/internal/udml/ontology"
)

// globalStoragePath stores the storage path for use by worker functions
var globalStoragePath string

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

	// Set global storage path for worker functions
	globalStoragePath = storagePath

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
		log.Printf("🎯 This worker is the LEADER - creating entity extraction tasks")

		// Leader creates entity extraction tasks only (relationships come after consolidation)
		if err := createEntityTasks(jobControl, storage, schema, runID, *docBatchSize); err != nil {
			log.Fatalf("Failed to create entity extraction tasks: %v", err)
		}

		log.Printf("  ✓ Leader finished creating entity tasks\n")
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

		if err := storage.ConsolidateEntities(runID, "max_confidence", llmValidator, schema); err != nil {
			log.Fatalf("Failed to consolidate entities: %v", err)
		}
		log.Println("  ✓ Entity consolidation complete")

		// Now create relationship extraction tasks (working on canonical entities)
		log.Println("")
		log.Println("🔗 Leader creating relationship extraction tasks...")
		if err := createRelationshipTasks(jobControl, schema, runID); err != nil {
			log.Fatalf("Failed to create relationship extraction tasks: %v", err)
		}
		log.Println("  ✓ Relationship tasks created")
	}

	// Phase 4: Process relationship extraction tasks (all workers)
	if len(schema.EntityRelationshipRules) > 0 {
		log.Println("")
		log.Println("🔍 Processing relationship extraction tasks...")
		if err := processExtractionTasks(jobControl, storage, schema, runID, workerID, conceptEmbeddings); err != nil {
			log.Fatalf("Failed to process relationship tasks: %v", err)
		}
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

// generateUUID generates a simple UUID-like identifier
func generateUUID() string {
	return fmt.Sprintf("rel_%s", generateRandomHex(16))
}

// createExtractionTasks creates all extraction tasks for distributed processing
// The leader worker calls this once at the start of an extraction run
func createEntityTasks(
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

	return jobControl.CreateTasks(runID, tasks)
}

// createRelationshipTasks creates relationship extraction tasks (one per rule)
// These work on canonical entities, not documents, so no document batching needed
func createRelationshipTasks(
	jobControl ontology.ExtractionJobControl,
	schema ontology.OntologySchema,
	runID string,
) error {
	var tasks []ontology.ExtractionTask

	// Create ONE task per relationship rule (operates on canonical entities)
	for idx, rule := range schema.EntityRelationshipRules {
		relType := string(rule.RelationshipType)

		task := ontology.ExtractionTask{
			ID:               fmt.Sprintf("rel_%s_%d", relType, idx),
			RunID:            runID,
			Type:             ontology.TaskTypeRelationshipRule,
			RelationshipType: relType,
			DocIDs:           []string{}, // Not document-based - works on canonical entities
			Status:           ontology.TaskStatusPending,
			CreatedAt:        time.Now(),
		}
		tasks = append(tasks, task)
	}

	log.Printf("  Created %d relationship extraction tasks", len(tasks))

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
	// Find all mappings for this entity type (use computed runtime mappings)
	computedMappings := schema.GetComputedMappings()
	var mappings []*ontology.ElementEntityMapping
	for i := range computedMappings {
		if string(computedMappings[i].EntityType) == task.EntityType {
			mappings = append(mappings, &computedMappings[i])
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
// Extracts relationships by finding source/target entity pairs in the same document
func processRelationshipTask(
	storage analytics.Storage,
	schema ontology.OntologySchema,
	task *ontology.ExtractionTask,
	conceptEmbeddings map[string][]float64,
) error {
	// Find the rule for this relationship type
	var rule *ontology.EntityRelationshipRule
	for i := range schema.EntityRelationshipRules {
		if string(schema.EntityRelationshipRules[i].RelationshipType) == task.RelationshipType {
			rule = &schema.EntityRelationshipRules[i]
			break
		}
	}

	if rule == nil {
		return fmt.Errorf("no rule found for relationship type %s", task.RelationshipType)
	}

	log.Printf("  Extracting %s relationships (%s -> %s)",
		rule.RelationshipType, rule.SourceEntityType, rule.TargetEntityType)

	// Extract relationships using document-level co-occurrence
	// This is a proximity-based MVP: if both entities appear in same doc, create relationship
	relationships, err := extractRelationshipsByProximity(storage, task, rule)
	if err != nil {
		return fmt.Errorf("failed to extract relationships: %w", err)
	}

	log.Printf("    ✓ Extracted %d %s relationships", len(relationships), rule.RelationshipType)

	return nil
}

// extractRelationshipsByProximity extracts relationships by interpreting extraction patterns
func extractRelationshipsByProximity(
	storage analytics.Storage,
	task *ontology.ExtractionTask,
	rule *ontology.EntityRelationshipRule,
) ([]analytics.OntologyRelationship, error) {
	var allRelationships []analytics.OntologyRelationship

	// Process each extraction pattern
	for _, pattern := range rule.ExtractionPatterns {
		switch pattern.Type {
		case ontology.RelPatternCooccurrence:
			// Document-level co-occurrence: entities in same document
			rels, err := extractByCooccurrence(storage, task, rule)
			if err != nil {
				log.Printf("WARNING: Cooccurrence extraction failed: %v", err)
				continue
			}
			allRelationships = append(allRelationships, rels...)

		case ontology.RelPatternProximity:
			// Proximity-based: entities within max_distance with signal words
			rels, err := extractByProximity(storage, task, rule, &pattern)
			if err != nil {
				log.Printf("WARNING: Proximity extraction failed: %v", err)
				continue
			}
			allRelationships = append(allRelationships, rels...)

		case ontology.RelPatternTextTemplate:
			// Text template matching with entity placeholders
			rels, err := extractByTextTemplate(storage, task, rule, &pattern)
			if err != nil {
				log.Printf("WARNING: Text template extraction failed: %v", err)
				continue
			}
			allRelationships = append(allRelationships, rels...)

		case ontology.RelPatternRegex:
			// Regex with named capture groups
			rels, err := extractByRegex(storage, task, rule, &pattern)
			if err != nil {
				log.Printf("WARNING: Regex extraction failed: %v", err)
				continue
			}
			allRelationships = append(allRelationships, rels...)

		default:
			log.Printf("  SKIPPING unknown pattern type: %s", pattern.Type)
		}
	}

	// Store raw relationships (deduplication happens later in consolidation phase)
	if len(allRelationships) > 0 {
		if err := storage.(*analytics.HiveParquetStorage).AppendOntologyRelationships(allRelationships); err != nil {
			return nil, fmt.Errorf("failed to write raw relationships: %w", err)
		}
	}

	return allRelationships, nil
}

// extractByCooccurrence finds relationships when entities appear in same document
func extractByCooccurrence(
	storage analytics.Storage,
	task *ontology.ExtractionTask,
	rule *ontology.EntityRelationshipRule,
) ([]analytics.OntologyRelationship, error) {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		return nil, fmt.Errorf("failed to open DuckDB: %w", err)
	}
	defer db.Close()

	query := fmt.Sprintf(`
WITH source_canonical AS (
  SELECT representative_entity_id, entity_name, entity_type, domain, confidence
  FROM '%s/canonical_entities/run_id=%s/*.parquet'
  WHERE entity_type = '%s'
),
target_canonical AS (
  SELECT representative_entity_id, entity_name, entity_type, confidence
  FROM '%s/canonical_entities/run_id=%s/*.parquet'
  WHERE entity_type = '%s'
)
SELECT
  s.representative_entity_id, t.representative_entity_id, 'corpus', 'canonical', s.domain, 'canonical',
  s.entity_name, t.entity_name, (s.confidence + t.confidence) / 2.0
FROM source_canonical s
CROSS JOIN target_canonical t
WHERE s.representative_entity_id != t.representative_entity_id
`,
		globalStoragePath, task.RunID, rule.SourceEntityType,
		globalStoragePath, task.RunID, rule.TargetEntityType,
	)

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	var relationships []analytics.OntologyRelationship
	for rows.Next() {
		var sourceID, targetID, docID, sourceName, domain, elementID, sourceTxt, targetTxt string
		var avgConf float64

		if err := rows.Scan(&sourceID, &targetID, &docID, &sourceName, &domain, &elementID,
			&sourceTxt, &targetTxt, &avgConf); err != nil {
			continue
		}

		relationships = append(relationships, analytics.OntologyRelationship{
			RelationshipID:   generateUUID(),
			DocID:            docID,
			SourceName:       sourceName,
			SourceEntityID:   sourceID,
			TargetEntityID:   targetID,
			RelationshipType: string(rule.RelationshipType),
			Domain:           domain,
			Confidence:       rule.Confidence * avgConf,
			Evidence:         fmt.Sprintf("cooccurrence: %s with %s", sourceTxt, targetTxt),
			ElementID:        elementID,
			RunID:            task.RunID,
			ExtractedAt:      time.Now(),
		})
	}

	return relationships, rows.Err()
}

// extractByProximity finds relationships when entities are near each other with signal words
func extractByProximity(
	storage analytics.Storage,
	task *ontology.ExtractionTask,
	rule *ontology.EntityRelationshipRule,
	pattern *ontology.RelationshipExtractionPattern,
) ([]analytics.OntologyRelationship, error) {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		return nil, fmt.Errorf("failed to open DuckDB: %w", err)
	}
	defer db.Close()

	// Query canonical entities directly (no need for proximity since working at corpus level)
	query := fmt.Sprintf(`
WITH source_canonical AS (
  SELECT representative_entity_id, entity_name, entity_type, domain, confidence
  FROM '%s/canonical_entities/run_id=%s/*.parquet'
  WHERE entity_type = '%s'
),
target_canonical AS (
  SELECT representative_entity_id, entity_name, entity_type, confidence
  FROM '%s/canonical_entities/run_id=%s/*.parquet'
  WHERE entity_type = '%s'
)
SELECT
  s.representative_entity_id, t.representative_entity_id, 'corpus', 'canonical', s.domain, 'canonical',
  s.entity_name, t.entity_name, 'canonical entities', (s.confidence + t.confidence) / 2.0
FROM source_canonical s
CROSS JOIN target_canonical t
WHERE s.representative_entity_id != t.representative_entity_id
`,
		globalStoragePath, task.RunID, rule.SourceEntityType,
		globalStoragePath, task.RunID, rule.TargetEntityType,
	)

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	var relationships []analytics.OntologyRelationship
	for rows.Next() {
		var sourceID, targetID, docID, sourceName, domain, elementID, sourceTxt, targetTxt, text string
		var avgConf float64

		if err := rows.Scan(&sourceID, &targetID, &docID, &sourceName, &domain, &elementID,
			&sourceTxt, &targetTxt, &text, &avgConf); err != nil {
			continue
		}

		// Check proximity and signal words in text
		if checkProximity(text, sourceTxt, targetTxt, pattern.MaxDistance, pattern.SignalWords) {
			relationships = append(relationships, analytics.OntologyRelationship{
				RelationshipID:   generateUUID(),
				DocID:            docID,
				SourceName:       sourceName,
				SourceEntityID:   sourceID,
				TargetEntityID:   targetID,
				RelationshipType: string(rule.RelationshipType),
				Domain:           domain,
				Confidence:       rule.Confidence * avgConf,
				Evidence:         fmt.Sprintf("proximity: %s near %s", sourceTxt, targetTxt),
				ElementID:        elementID,
				RunID:            task.RunID,
				ExtractedAt:      time.Now(),
			})
		}
	}

	return relationships, rows.Err()
}

// checkProximity checks if two entities are within max_distance tokens with signal words
func checkProximity(text, entity1, entity2 string, maxDistance int, signalWords []string) bool {
	text = strings.ToLower(text)
	e1 := strings.ToLower(entity1)
	e2 := strings.ToLower(entity2)

	// Find positions of entity mentions
	idx1 := strings.Index(text, e1)
	idx2 := strings.Index(text, e2)

	if idx1 == -1 || idx2 == -1 {
		return false
	}

	// Ensure idx1 < idx2
	if idx1 > idx2 {
		idx1, idx2 = idx2, idx1
		e1, e2 = e2, e1
	}

	// Extract text between entities
	between := text[idx1+len(e1) : idx2]

	// Count tokens (split on whitespace)
	tokens := strings.Fields(between)
	if len(tokens) > maxDistance {
		return false
	}

	// Check if any signal word is present
	if len(signalWords) == 0 {
		return true // No signal words required
	}

	for _, signal := range signalWords {
		if strings.Contains(between, strings.ToLower(signal)) {
			return true
		}
	}

	return false
}

// consolidateRelationships deduplicates raw relationships by (source, target, type)
// keeping the highest confidence instance. Currently a placeholder - deduplication
// can be done at query time when reading relationships.
func consolidateRelationships(storage analytics.Storage, runID string) error {
	log.Println("  Raw relationships written - deduplication happens at query time")
	return nil
}

// extractByTextTemplate finds relationships by matching text templates with entity placeholders
func extractByTextTemplate(
	storage analytics.Storage,
	task *ontology.ExtractionTask,
	rule *ontology.EntityRelationshipRule,
	pattern *ontology.RelationshipExtractionPattern,
) ([]analytics.OntologyRelationship, error) {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		return nil, fmt.Errorf("failed to open DuckDB: %w", err)
	}
	defer db.Close()

	// Query embeddings for full contextual text (element + ancestor + sibling content)
	query := fmt.Sprintf(`
SELECT
  emb.element_id, emb.doc_id, emb.source_name, emb.text
FROM '%s/embeddings/**/*.parquet' emb
WHERE emb.doc_id IN (%s) AND emb.text IS NOT NULL
`,
		globalStoragePath, formatDocIDList(task.DocIDs),
	)

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	// Build regex from template by replacing {entity_type} with capture groups
	templateRegex := buildTemplateRegex(pattern.Template, rule.SourceEntityType, rule.TargetEntityType)
	re, err := regexp.Compile(templateRegex)
	if err != nil {
		return nil, fmt.Errorf("failed to compile template regex: %w", err)
	}

	var relationships []analytics.OntologyRelationship
	for rows.Next() {
		var elementID, docID, sourceName, text string
		if err := rows.Scan(&elementID, &docID, &sourceName, &text); err != nil {
			continue
		}

		// Find matches in text
		matches := re.FindAllStringSubmatch(text, -1)
		for _, match := range matches {
			if len(match) < 3 {
				continue
			}

			sourceEntity := strings.TrimSpace(match[1])
			targetEntity := strings.TrimSpace(match[2])

			// Look up entity IDs and confidence from extractions
			sourceID, sourceConf := lookupEntity(db, task.RunID, rule.SourceEntityType, sourceEntity, docID)
			targetID, targetConf := lookupEntity(db, task.RunID, rule.TargetEntityType, targetEntity, docID)

			if sourceID == "" || targetID == "" {
				continue
			}

			// Get domain from source entity
			domain := getDomainForEntity(db, task.RunID, sourceID)

			relationships = append(relationships, analytics.OntologyRelationship{
				RelationshipID:   generateUUID(),
				DocID:            docID,
				SourceName:       sourceName,
				SourceEntityID:   sourceID,
				TargetEntityID:   targetID,
				RelationshipType: string(rule.RelationshipType),
				Domain:           domain,
				Confidence:       rule.Confidence * ((sourceConf + targetConf) / 2.0),
				Evidence:         fmt.Sprintf("template: %s", match[0]),
				ElementID:        elementID,
				RunID:            task.RunID,
				ExtractedAt:      time.Now(),
			})
		}
	}

	return relationships, rows.Err()
}

// extractByRegex finds relationships using regex patterns with named capture groups
func extractByRegex(
	storage analytics.Storage,
	task *ontology.ExtractionTask,
	rule *ontology.EntityRelationshipRule,
	pattern *ontology.RelationshipExtractionPattern,
) ([]analytics.OntologyRelationship, error) {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		return nil, fmt.Errorf("failed to open DuckDB: %w", err)
	}
	defer db.Close()

	// Compile the regex pattern
	re, err := regexp.Compile(pattern.Pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to compile regex pattern: %w", err)
	}

	// Query embeddings for full contextual text (element + ancestor + sibling content)
	query := fmt.Sprintf(`
SELECT
  emb.element_id, emb.doc_id, emb.source_name, emb.text
FROM '%s/embeddings/**/*.parquet' emb
WHERE emb.doc_id IN (%s) AND emb.text IS NOT NULL
`,
		globalStoragePath, formatDocIDList(task.DocIDs),
	)

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	var relationships []analytics.OntologyRelationship
	for rows.Next() {
		var elementID, docID, sourceName, text string
		if err := rows.Scan(&elementID, &docID, &sourceName, &text); err != nil {
			continue
		}

		// Find regex matches
		matches := re.FindAllStringSubmatch(text, -1)
		names := re.SubexpNames()

		for _, match := range matches {
			// Extract named groups
			groups := make(map[string]string)
			for i, name := range names {
				if i > 0 && i < len(match) && name != "" {
					groups[name] = strings.TrimSpace(match[i])
				}
			}

			// Determine which group corresponds to source/target entity
			sourceEntity := groups[rule.SourceEntityType]
			targetEntity := groups[rule.TargetEntityType]

			if sourceEntity == "" || targetEntity == "" {
				continue
			}

			// Look up entity IDs
			sourceID, sourceConf := lookupEntity(db, task.RunID, rule.SourceEntityType, sourceEntity, docID)
			targetID, targetConf := lookupEntity(db, task.RunID, rule.TargetEntityType, targetEntity, docID)

			if sourceID == "" || targetID == "" {
				continue
			}

			domain := getDomainForEntity(db, task.RunID, sourceID)

			relationships = append(relationships, analytics.OntologyRelationship{
				RelationshipID:   generateUUID(),
				DocID:            docID,
				SourceName:       sourceName,
				SourceEntityID:   sourceID,
				TargetEntityID:   targetID,
				RelationshipType: string(rule.RelationshipType),
				Domain:           domain,
				Confidence:       rule.Confidence * ((sourceConf + targetConf) / 2.0),
				Evidence:         fmt.Sprintf("regex: %s", match[0]),
				ElementID:        elementID,
				RunID:            task.RunID,
				ExtractedAt:      time.Now(),
			})
		}
	}

	return relationships, rows.Err()
}

// buildTemplateRegex converts template like "{person}, {publication}" to regex
func buildTemplateRegex(template, sourceType, targetType string) string {
	// Escape special regex characters
	regex := regexp.QuoteMeta(template)

	// Replace {sourceType} with capture group
	sourcePattern := fmt.Sprintf("\\{%s\\}", sourceType)
	regex = strings.Replace(regex, sourcePattern, "([^,\"()]+)", 1)

	// Replace {targetType} with capture group
	targetPattern := fmt.Sprintf("\\{%s\\}", targetType)
	regex = strings.Replace(regex, targetPattern, "([^,\"()]+)", 1)

	// Replace other entity type placeholders with non-capturing groups
	regex = regexp.MustCompile(`\\{[^}]+\\}`).ReplaceAllString(regex, "(?:[^,\"()]+)")

	return regex
}

// lookupEntity finds entity ID and confidence by name in extractions
func lookupEntity(db *sql.DB, runID, entityType, entityName, docID string) (string, float64) {
	query := fmt.Sprintf(`
SELECT em.canonical_entity_id, e.confidence
FROM '%s/ontology_entities/run_id=%s/*.parquet' e
INNER JOIN '%s/entity_mentions/run_id=%s/*.parquet' em ON e.entity_id = em.raw_entity_id
WHERE e.entity_type = '%s' AND e.entity_name = '%s' AND e.doc_id = '%s'
LIMIT 1
`,
		globalStoragePath, runID,
		globalStoragePath, runID,
		strings.ReplaceAll(entityType, "'", "''"),
		strings.ReplaceAll(entityName, "'", "''"),
		strings.ReplaceAll(docID, "'", "''"),
	)

	var canonicalEntityID string
	var confidence float64
	err := db.QueryRow(query).Scan(&canonicalEntityID, &confidence)
	if err != nil {
		return "", 0.0
	}

	return canonicalEntityID, confidence
}

// getDomainForEntity retrieves domain for an entity
func getDomainForEntity(db *sql.DB, runID, entityID string) string {
	query := fmt.Sprintf(`
SELECT domain
FROM '%s/ontology_entities/run_id=%s/*.parquet'
WHERE entity_id = '%s'
LIMIT 1
`,
		globalStoragePath, runID,
		strings.ReplaceAll(entityID, "'", "''"),
	)

	var domain string
	db.QueryRow(query).Scan(&domain)
	return domain
}

// formatDocIDList formats document IDs for SQL IN clause
func formatDocIDList(docIDs []string) string {
	quoted := make([]string, len(docIDs))
	for i, id := range docIDs {
		// Escape single quotes in doc IDs
		escaped := strings.ReplaceAll(id, "'", "''")
		quoted[i] = fmt.Sprintf("'%s'", escaped)
	}
	return strings.Join(quoted, ", ")
}

// collectReferenceConcepts extracts all unique reference concepts from semantic filters in the schema
func collectReferenceConcepts(schema ontology.OntologySchema) []string {
	conceptsMap := make(map[string]bool)

	// Iterate through all entity mappings (use computed runtime mappings)
	computedMappings := schema.GetComputedMappings()
	for _, mapping := range computedMappings {
		// Check each extraction rule for semantic filter
		for _, rule := range mapping.ExtractionRules {
			if rule.Semantic != nil {
				// Handle reference_text (single string)
				if rule.Semantic.ReferenceText != "" {
					conceptsMap[rule.Semantic.ReferenceText] = true
				}

				// Handle reference_concepts (array)
				for _, concept := range rule.Semantic.ReferenceConcepts {
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
