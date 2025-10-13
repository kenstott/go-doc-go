package worker

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/kennethstott/doculyzer-go-conversion/internal/analytics"
	"github.com/kennethstott/doculyzer-go-conversion/internal/udml/ontology"
)

// OntologyExtractor performs entity and relationship extraction using ontology schemas
type OntologyExtractor struct {
	config   *OntologyConfig
	storages []analytics.Storage
	schemas  []*ontology.OntologySchema
}

// NewOntologyExtractor creates a new ontology extractor
func NewOntologyExtractor(config *OntologyConfig, storages []analytics.Storage) (*OntologyExtractor, error) {
	// Load ontology schemas from configured path
	schemas, err := loadOntologySchemas(config.SchemaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load ontology schemas: %w", err)
	}

	if len(schemas) == 0 {
		return nil, fmt.Errorf("no ontology schemas found in %s", config.SchemaPath)
	}

	log.Printf("ONTOLOGY: Loaded %d ontology schemas from %s", len(schemas), config.SchemaPath)
	for _, schema := range schemas {
		log.Printf("  - Domain: %s (%d entity mappings, %d relationship rules)",
			schema.Domain, len(schema.ElementEntityMappings), len(schema.EntityRelationshipRules))
	}

	return &OntologyExtractor{
		config:   config,
		storages: storages,
		schemas:  schemas,
	}, nil
}

// ExtractAndStore performs ontology extraction and stores results
func (e *OntologyExtractor) ExtractAndStore() error {
	if !e.config.Enabled {
		return fmt.Errorf("ontology extraction not enabled")
	}

	log.Println("========================================")
	log.Println("ONTOLOGY EXTRACTION: Starting entity and relationship extraction")
	log.Printf("  Schemas: %d domains", len(e.schemas))
	log.Printf("  Schema path: %s", e.config.SchemaPath)
	log.Println("========================================")

	startTime := time.Now()

	// Step 1: Query all elements from storage (build UDML-N in-memory layer)
	allElements, err := e.queryAllElements()
	if err != nil {
		return fmt.Errorf("failed to query elements: %w", err)
	}

	if len(allElements) == 0 {
		log.Println("ONTOLOGY EXTRACTION: No elements found, skipping")
		return nil
	}

	log.Printf("ONTOLOGY EXTRACTION: Found %d elements to analyze", len(allElements))

	// Step 2: Group elements by document for per-document extraction
	docElements := e.groupElementsByDocument(allElements)
	log.Printf("ONTOLOGY EXTRACTION: Processing %d documents", len(docElements))

	// Step 3: Extract entities and relationships per schema per document
	var allEntities []analytics.OntologyEntity
	var allRelationships []analytics.OntologyRelationship
	var allMentions []analytics.OntologyMention

	for docID, elements := range docElements {
		log.Printf("ONTOLOGY EXTRACTION: Processing document %s (%d elements)", docID, len(elements))

		// Apply each schema to this document
		for _, schema := range e.schemas {
			// Create extractor for this schema
			extractor := ontology.NewRuleBasedExtractor(schema)

			// Extract from elements
			ctx := context.Background()
			result, err := extractor.ExtractFromElements(ctx, docID, elements)
			if err != nil {
				log.Printf("WARNING: Extraction failed for document %s with schema %s: %v", docID, schema.Domain, err)
				continue
			}

			// Convert results to analytics types
			entities, relationships, mentions := e.convertToAnalyticsTypes(docID, schema.Domain, result, elements)
			allEntities = append(allEntities, entities...)
			allRelationships = append(allRelationships, relationships...)
			allMentions = append(allMentions, mentions...)

			log.Printf("  Schema %s: Found %d entities, %d relationships, %d mentions",
				schema.Domain, len(entities), len(relationships), len(mentions))
		}
	}

	log.Printf("ONTOLOGY EXTRACTION: Total extracted: %d entities, %d relationships, %d mentions",
		len(allEntities), len(allRelationships), len(allMentions))

	// Step 4: Store results in analytics storages
	if len(allEntities) > 0 || len(allRelationships) > 0 || len(allMentions) > 0 {
		for _, storage := range e.storages {
			if len(allEntities) > 0 {
				if err := storage.AppendOntologyEntities(allEntities); err != nil {
					log.Printf("Failed to store ontology entities: %v", err)
				}
			}
			if len(allRelationships) > 0 {
				if err := storage.AppendOntologyRelationships(allRelationships); err != nil {
					log.Printf("Failed to store ontology relationships: %v", err)
				}
			}
			if len(allMentions) > 0 {
				if err := storage.AppendOntologyMentions(allMentions); err != nil {
					log.Printf("Failed to store ontology mentions: %v", err)
				}
			}
		}
	}

	duration := time.Since(startTime)
	log.Println("========================================")
	log.Printf("ONTOLOGY EXTRACTION: Completed in %v", duration)
	log.Printf("  Documents processed: %d", len(docElements))
	log.Printf("  Entities extracted: %d", len(allEntities))
	log.Printf("  Relationships extracted: %d", len(allRelationships))
	log.Printf("  Mentions tracked: %d", len(allMentions))
	log.Printf("  Extraction rate: %.2f elements/sec", float64(len(allElements))/duration.Seconds())
	log.Println("========================================")

	return nil
}

// queryAllElements retrieves all elements from storage
func (e *OntologyExtractor) queryAllElements() ([]ontology.Element, error) {
	// Query all elements from the first storage (they all contain the same data)
	if len(e.storages) == 0 {
		return nil, fmt.Errorf("no storage configured for ontology extraction")
	}

	storage := e.storages[0]
	analyticsElements, err := storage.QueryElements(map[string]interface{}{})
	if err != nil {
		return nil, fmt.Errorf("failed to query elements from storage: %w", err)
	}

	// Convert analytics.Element to ontology.Element (minimal subset)
	ontologyElements := make([]ontology.Element, len(analyticsElements))
	for i, ae := range analyticsElements {
		// Ensure metadata has doc_id and source_name for grouping
		metadata := ae.Metadata
		if metadata == nil {
			metadata = make(map[string]interface{})
		}
		metadata["doc_id"] = ae.DocID
		metadata["source_name"] = ae.SourceName

		ontologyElements[i] = ontology.Element{
			ElementID:      ae.ElementID,
			ElementType:    ae.ElementType,
			Content:        ae.Content,
			ContentPreview: ae.ContentPreview,
			ParentID:       ae.ParentID,
			ElementOrder:   ae.ElementOrder,
			Metadata:       metadata,
		}
	}

	return ontologyElements, nil
}

// groupElementsByDocument groups elements by document ID
func (e *OntologyExtractor) groupElementsByDocument(elements []ontology.Element) map[string][]ontology.Element {
	docElements := make(map[string][]ontology.Element)
	for _, elem := range elements {
		// Extract doc_id from element metadata (populated by analytics conversion)
		docID := "unknown"
		if docIDVal, ok := elem.Metadata["doc_id"].(string); ok && docIDVal != "" {
			docID = docIDVal
		}
		docElements[docID] = append(docElements[docID], elem)
	}
	return docElements
}

// convertToAnalyticsTypes converts ontology extraction results to analytics types
func (e *OntologyExtractor) convertToAnalyticsTypes(
	docID string,
	domain string,
	result *ontology.Ontology,
	elements []ontology.Element,
) ([]analytics.OntologyEntity, []analytics.OntologyRelationship, []analytics.OntologyMention) {
	now := time.Now()

	// Get source name from first element (all elements in same doc have same source)
	sourceName := "unknown"
	if len(elements) > 0 {
		if sourceNameVal, ok := elements[0].Metadata["source_name"].(string); ok && sourceNameVal != "" {
			sourceName = sourceNameVal
		}
	}

	// Convert entities
	var entities []analytics.OntologyEntity
	for _, entity := range result.Entities {
		entities = append(entities, analytics.OntologyEntity{
			EntityID:    entity.ID,
			DocID:       docID,
			SourceName:  sourceName,
			EntityName:  entity.Name,
			EntityType:  string(entity.Type), // Convert EntityType to string
			Domain:      domain,
			Confidence:  entity.Confidence,
			Attributes:  entity.Attributes,
			ElementID:   entity.ElementID,
			ExtractedAt: now,
		})
	}

	// Convert relationships
	var relationships []analytics.OntologyRelationship
	for _, rel := range result.Relationships {
		relationships = append(relationships, analytics.OntologyRelationship{
			RelationshipID:   rel.ID,
			DocID:            docID,
			SourceName:       sourceName,
			SourceEntityID:   rel.SourceID,
			TargetEntityID:   rel.TargetID,
			RelationshipType: string(rel.Type), // Convert RelationshipType to string
			Domain:           domain,
			Confidence:       rel.Confidence,
			Evidence:         rel.Evidence,
			Attributes:       rel.Attributes,
			ElementID:        rel.ElementID,
			ExtractedAt:      now,
		})
	}

	// Convert mentions - mentions are within entities
	var mentions []analytics.OntologyMention
	mentionID := 0
	for _, entity := range result.Entities {
		for _, mention := range entity.Mentions {
			mentions = append(mentions, analytics.OntologyMention{
				MentionID:     fmt.Sprintf("mention_%s_%d", entity.ID, mentionID),
				EntityID:      entity.ID,
				DocID:         docID,
				SourceName:    sourceName,
				ElementID:     mention.ElementID,
				MentionText:   mention.Text,
				StartPosition: mention.StartPos,
				EndPosition:   mention.EndPos,
				ExtractedAt:   now,
			})
			mentionID++
		}
	}

	return entities, relationships, mentions
}

// loadOntologySchemas loads all ontology schemas from a directory
func loadOntologySchemas(schemaPath string) ([]*ontology.OntologySchema, error) {
	var schemas []*ontology.OntologySchema

	// Check if path exists
	if _, err := os.Stat(schemaPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("schema path does not exist: %s", schemaPath)
	}

	// Check if it's a file or directory
	info, err := os.Stat(schemaPath)
	if err != nil {
		return nil, err
	}

	if info.IsDir() {
		// Load all schema files from directory
		entries, err := os.ReadDir(schemaPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read schema directory: %w", err)
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			// Only load YAML and JSON files
			ext := filepath.Ext(entry.Name())
			if ext != ".yaml" && ext != ".yml" && ext != ".json" {
				continue
			}

			schemaFile := filepath.Join(schemaPath, entry.Name())
			schema, err := ontology.LoadSchema(schemaFile, ontology.FormatAuto)
			if err != nil {
				log.Printf("WARNING: Failed to load schema from %s: %v", schemaFile, err)
				continue
			}
			schemas = append(schemas, schema)
		}
	} else {
		// Load single schema file
		schema, err := ontology.LoadSchema(schemaPath, ontology.FormatAuto)
		if err != nil {
			return nil, fmt.Errorf("failed to load schema from %s: %w", schemaPath, err)
		}
		schemas = append(schemas, schema)
	}

	return schemas, nil
}
