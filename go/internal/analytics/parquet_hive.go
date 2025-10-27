package analytics

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/apache/arrow/go/v18/arrow"
	"github.com/apache/arrow/go/v18/arrow/array"
	"github.com/apache/arrow/go/v18/arrow/memory"
	"github.com/apache/arrow/go/v18/parquet"
	"github.com/apache/arrow/go/v18/parquet/compress"
	"github.com/apache/arrow/go/v18/parquet/pqarrow"
	"github.com/kennethstott/doculyzer-go-conversion/internal/parser"
	"github.com/kennethstott/doculyzer-go-conversion/internal/resolver"
	"github.com/kennethstott/doculyzer-go-conversion/internal/udml"
	_ "github.com/marcboeker/go-duckdb"
	"reflect"
)

// Local interfaces to avoid circular dependency with ontology package
// These match the interfaces defined in ontology package

// llmClientInterface defines the LLM client interface for validation
type llmClientInterface interface {
	Complete(ctx context.Context, prompt string, options llmOptionsInterface) (string, error)
}

// llmOptionsInterface defines LLM completion options
type llmOptionsInterface interface {
	GetMaxTokens() int
	GetTemperature() float64
	GetSystemPrompt() string
}

// llmValidatorInterface defines the validator interface
type llmValidatorInterface interface {
	BatchValidate(ctx context.Context, entities []entityToValidate) ([]bool, error)
}

// entityToValidate represents an entity for validation
type entityToValidate struct {
	EntityName string
	Prompt     string // Deprecated - validation templates use entity_type instead
	EntityType string // Entity type for template lookup
}

// HiveParquetStorage implements Storage interface with Hive-partitioned structure
// Partition scheme: element_type=X/version=Y/date=Z/source=W/
// This enables query engines to skip irrelevant partitions for 60-1000x faster queries
type HiveParquetStorage struct {
	basePath        string
	version         string // UDML schema version (e.g., "v2.0.0")
	schemaRegistry  *udml.SchemaRegistry
	contentResolver resolver.ContentResolver
	mu              sync.Mutex
	allocator       memory.Allocator
}

// NewHiveParquetStorage creates a new Hive-partitioned Parquet storage backend
func NewHiveParquetStorage(config map[string]interface{}) (*HiveParquetStorage, error) {
	// Extract configuration
	basePath, ok := config["path"].(string)
	if !ok || basePath == "" {
		return nil, fmt.Errorf("missing required 'path' in config")
	}

	// Extract version (default: v2.0.0 for UDML Phase 1)
	version, ok := config["version"].(string)
	if !ok || version == "" {
		version = "v2.0.0"
	}

	// Create base directory if it doesn't exist
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base path: %w", err)
	}

	// Create parsers for content resolution
	htmlParser := parser.NewHTMLParser()
	xmlParser := parser.NewXMLParser()
	jsonParser := parser.NewJSONParser()

	// Create ContentResolver with HTML/XML/JSON parsers
	parserResolvers := map[string]resolver.ParserResolver{
		"html": htmlParser,
		"xml":  xmlParser,
		"json": jsonParser,
	}
	contentResolver := resolver.NewContentResolver(parserResolvers)

	log.Printf("========================================")
	log.Printf("ANALYTICS: Initialized Hive-partitioned Parquet storage")
	log.Printf("  Path: %s", basePath)
	log.Printf("  Version: %s", version)
	log.Printf("  Partitioning: element_type -> version -> date -> source")
	log.Printf("  Schema: UDML Phase 1 (20 fields with 6 promoted query-optimized fields)")
	log.Printf("  Content Resolver: HTML/XML/JSON parsers configured")
	log.Printf("========================================")

	return &HiveParquetStorage{
		basePath:        basePath,
		version:         version,
		schemaRegistry:  udml.NewSchemaRegistry(),
		contentResolver: contentResolver,
		allocator:       memory.NewGoAllocator(),
	}, nil
}

// AppendDocuments writes documents to Parquet files
func (s *HiveParquetStorage) AppendDocuments(documents []Document) error {
	if len(documents) == 0 {
		return nil
	}

	// Group documents by partition keys
	partitioned := s.partitionDocuments(documents)

	for partKey, docs := range partitioned {
		if err := s.writeDocumentsToParquet(partKey, docs); err != nil {
			return fmt.Errorf("failed to write documents partition %s: %w", partKey, err)
		}
	}

	log.Printf("ANALYTICS: Wrote %d documents to Hive-partitioned Parquet", len(documents))
	return nil
}

// AppendElements writes elements to Hive-partitioned Parquet files
// Elements are first grouped by element_type for optimal query performance
func (s *HiveParquetStorage) AppendElements(elements []Element) error {
	if len(elements) == 0 {
		return nil
	}

	// Group by element type first (primary partition key)
	byType := make(map[string][]Element)
	for _, elem := range elements {
		byType[elem.ElementType] = append(byType[elem.ElementType], elem)
	}

	// Write each type to its Hive partition
	for elemType, typeElements := range byType {
		// Further partition by date and source
		partitioned := s.partitionElements(typeElements)

		for partKey, elems := range partitioned {
			if err := s.writeElementsToHivePartition(elemType, partKey, elems); err != nil {
				return fmt.Errorf("failed to write elements partition %s/%s: %w", elemType, partKey, err)
			}
		}
	}

	log.Printf("ANALYTICS: Wrote %d elements to Hive-partitioned Parquet (%d types)", len(elements), len(byType))
	return nil
}

// QueryElements queries elements from Hive-partitioned Parquet files using DuckDB
func (s *HiveParquetStorage) QueryElements(filters map[string]interface{}) ([]Element, error) {
	// Open DuckDB connection (in-memory)
	db, err := sql.Open("duckdb", "")
	if err != nil {
		return nil, fmt.Errorf("failed to open DuckDB connection: %w", err)
	}
	defer db.Close()

	// Build query - select only core fields (no promoted fields - they were never implemented in writer)
	query := fmt.Sprintf(`SELECT
		element_id, doc_id, source_name, element_type, element_category,
		content, content_preview, content_hash, parent_id,
		element_order, document_position, content_location, metadata
		FROM '%s/elements/**/*.parquet'`, s.basePath)

	// Build WHERE clauses from filters
	var whereClauses []string
	if source, ok := filters["source_name"].(string); ok && source != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("source_name = '%s'", source))
	}
	if docID, ok := filters["doc_id"].(string); ok && docID != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("doc_id = '%s'", docID))
	}
	if elementType, ok := filters["element_type"].(string); ok && elementType != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("element_type = '%s'", elementType))
	}
	if elementCategory, ok := filters["element_category"].(string); ok && elementCategory != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("element_category = '%s'", elementCategory))
	}

	// Filter to only leaf elements (elements with embeddings)
	if hasEmbedding, ok := filters["has_embedding"].(bool); ok && hasEmbedding {
		whereClauses = append(whereClauses, fmt.Sprintf(
			"element_id IN (SELECT DISTINCT element_id FROM '%s/embeddings/**/*.parquet')",
			s.basePath))
	}

	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	// Add LIMIT if specified - critical for not loading billions into memory
	if limit, ok := filters["limit"].(int); ok && limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	// Execute query
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query elements: %w", err)
	}
	defer rows.Close()

	// Parse results (only 13 core fields - promoted fields not in actual Parquet files)
	var elements []Element
	for rows.Next() {
		var elem Element
		var content sql.NullString
		var contentPreview sql.NullString
		var metadataJSON sql.NullString
		var contentLocationJSON sql.NullString
		var contentHash sql.NullString
		var parentID sql.NullString

		err := rows.Scan(
			&elem.ElementID, &elem.DocID, &elem.SourceName, &elem.ElementType, &elem.ElementCategory,
			&content, &contentPreview, &contentHash, &parentID,
			&elem.ElementOrder, &elem.DocumentPosition,
			&contentLocationJSON, &metadataJSON,
		)
		if err != nil {
			log.Printf("Failed to scan element row: %v", err)
			continue
		}

		// Parse nullable string fields
		if content.Valid {
			elem.Content = content.String
		}
		if contentPreview.Valid {
			elem.ContentPreview = contentPreview.String
		}
		if contentHash.Valid {
			elem.ContentHash = contentHash.String
		}
		if parentID.Valid {
			elem.ParentID = parentID.String
		}

		// Parse JSON fields
		if metadataJSON.Valid && metadataJSON.String != "" {
			if err := json.Unmarshal([]byte(metadataJSON.String), &elem.Metadata); err != nil {
				log.Printf("Failed to unmarshal metadata for element %s: %v", elem.ElementID, err)
			}
		}
		if contentLocationJSON.Valid && contentLocationJSON.String != "" {
			if err := json.Unmarshal([]byte(contentLocationJSON.String), &elem.ContentLocation); err != nil {
				log.Printf("Failed to unmarshal content_location for element %s: %v", elem.ElementID, err)
			}
		}

		elements = append(elements, elem)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating element rows: %w", err)
	}

	log.Printf("ANALYTICS: Queried %d elements from Hive-partitioned Parquet", len(elements))
	return elements, nil
}

// GetDistinctElementTypes queries distinct element types from Parquet files
// This is used by the ontology builder to discover actual element types in the corpus
func (s *HiveParquetStorage) GetDistinctElementTypes() ([]string, error) {
	// Open DuckDB connection (in-memory)
	db, err := sql.Open("duckdb", "")
	if err != nil {
		return nil, fmt.Errorf("failed to open DuckDB connection: %w", err)
	}
	defer db.Close()

	// Query distinct element types
	query := fmt.Sprintf("SELECT DISTINCT element_type FROM '%s/elements/**/*.parquet' ORDER BY element_type", s.basePath)

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query distinct element types: %w", err)
	}
	defer rows.Close()

	// Collect element types
	var elementTypes []string
	for rows.Next() {
		var elementType string
		if err := rows.Scan(&elementType); err != nil {
			log.Printf("Failed to scan element type row: %v", err)
			continue
		}
		elementTypes = append(elementTypes, elementType)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating element type rows: %w", err)
	}

	return elementTypes, nil
}

// AppendRelationships writes relationships to Parquet files
func (s *HiveParquetStorage) AppendRelationships(relationships []Relationship) error {
	if len(relationships) == 0 {
		return nil
	}

	// Group relationships by partition keys
	partitioned := s.partitionRelationships(relationships)

	for partKey, rels := range partitioned {
		if err := s.writeRelationshipsToParquet(partKey, rels); err != nil {
			return fmt.Errorf("failed to write relationships partition %s: %w", partKey, err)
		}
	}

	log.Printf("ANALYTICS: Wrote %d relationships to Hive-partitioned Parquet", len(relationships))
	return nil
}

// AppendEmbeddings writes embeddings to Parquet files
func (s *HiveParquetStorage) AppendEmbeddings(embeddings []Embedding) error {
	if len(embeddings) == 0 {
		return nil
	}

	// Group embeddings by partition keys
	partitioned := s.partitionEmbeddings(embeddings)

	for partKey, embs := range partitioned {
		if err := s.writeEmbeddingsToParquet(partKey, embs); err != nil {
			return fmt.Errorf("failed to write embeddings partition %s: %w", partKey, err)
		}
	}

	log.Printf("ANALYTICS: Wrote %d embeddings to Hive-partitioned Parquet", len(embeddings))
	return nil
}

// AppendLinks writes links to Parquet files
func (s *HiveParquetStorage) AppendLinks(links []Link) error {
	if len(links) == 0 {
		return nil
	}

	// Group links by partition keys
	partitioned := s.partitionLinks(links)

	for partKey, lnks := range partitioned {
		if err := s.writeLinksToParquet(partKey, lnks); err != nil {
			return fmt.Errorf("failed to write links partition %s: %w", partKey, err)
		}
	}

	log.Printf("ANALYTICS: Wrote %d links to Hive-partitioned Parquet", len(links))
	return nil
}

// QueryEmbeddings queries embeddings from Hive-partitioned Parquet files using DuckDB
func (s *HiveParquetStorage) QueryEmbeddings(filters map[string]interface{}) ([]Embedding, error) {
	// HiveParquetStorage uses the same DuckDB query approach as ParquetStorage
	// The query engine handles the Hive partitioning transparently

	// Open DuckDB connection (in-memory)
	db, err := sql.Open("duckdb", "")
	if err != nil {
		return nil, fmt.Errorf("failed to open DuckDB connection: %w", err)
	}
	defer db.Close()

	// Build query based on filters
	query := fmt.Sprintf("SELECT element_id, doc_id, source_name, embedding, text FROM '%s/embeddings/**/*.parquet'",
		s.basePath)

	var whereClauses []string
	if source, ok := filters["source_name"].(string); ok && source != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("source_name = '%s'", source))
	}
	if docID, ok := filters["doc_id"].(string); ok && docID != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("doc_id = '%s'", docID))
	}
	if elementID, ok := filters["element_id"].(string); ok && elementID != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("element_id = '%s'", elementID))
	}

	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	// Execute query
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query embeddings: %w", err)
	}
	defer rows.Close()

	// Parse results
	var embeddings []Embedding
	for rows.Next() {
		var emb Embedding
		var embeddingRaw interface{} // DuckDB returns arrays as []interface{}

		err := rows.Scan(&emb.ElementID, &emb.DocID, &emb.SourceName, &embeddingRaw, &emb.Text)
		if err != nil {
			log.Printf("Failed to scan embedding row: %v", err)
			continue
		}

		// Convert embedding from DuckDB array type to []float64
		// DuckDB returns arrays as []interface{} where each element is float64
		if embeddingArray, ok := embeddingRaw.([]interface{}); ok {
			emb.Embedding = make([]float64, len(embeddingArray))
			for i, val := range embeddingArray {
				if floatVal, ok := val.(float64); ok {
					emb.Embedding[i] = floatVal
				} else {
					log.Printf("WARNING: Embedding element %d for %s is not float64: %T", i, emb.ElementID, val)
				}
			}
		} else {
			log.Printf("Failed to parse embedding for element %s: expected []interface{}, got %T", emb.ElementID, embeddingRaw)
			continue
		}

		embeddings = append(embeddings, emb)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	log.Printf("ANALYTICS: Queried %d embeddings from Hive-partitioned Parquet", len(embeddings))
	return embeddings, nil
}

// ============================================================================
// UDML-O: Ontology Instance Layer Methods
// ============================================================================

// AppendOntologyEntities writes ontology entities to Parquet files
// Partitioning: source=X/domain=Y/run_id=Z/
func (s *HiveParquetStorage) AppendOntologyEntities(entities []OntologyEntity) error {
	if len(entities) == 0 {
		return nil
	}

	// Group entities by partition keys (source, domain, run_id)
	partitioned := s.partitionOntologyEntities(entities)

	for partKey, ents := range partitioned {
		if err := s.writeOntologyEntitiesToParquet(partKey, ents); err != nil {
			return fmt.Errorf("failed to write ontology entities partition %s: %w", partKey, err)
		}
	}

	log.Printf("ANALYTICS: Wrote %d ontology entities to Hive-partitioned Parquet", len(entities))
	return nil
}

// AppendOntologyRelationships writes ontology relationships to Parquet files
// Partitioning: source=X/domain=Y/run_id=Z/
func (s *HiveParquetStorage) AppendOntologyRelationships(relationships []OntologyRelationship) error {
	if len(relationships) == 0 {
		return nil
	}

	// Group relationships by partition keys (source, domain, run_id)
	partitioned := s.partitionOntologyRelationships(relationships)

	for partKey, rels := range partitioned {
		if err := s.writeOntologyRelationshipsToParquet(partKey, rels); err != nil {
			return fmt.Errorf("failed to write ontology relationships partition %s: %w", partKey, err)
		}
	}

	log.Printf("ANALYTICS: Wrote %d ontology relationships to Hive-partitioned Parquet", len(relationships))
	return nil
}

// AppendOntologyMentions writes ontology mentions to Parquet files
// Partitioning: source=X/domain=Y/date=Z/
func (s *HiveParquetStorage) AppendOntologyMentions(mentions []OntologyMention) error {
	if len(mentions) == 0 {
		return nil
	}

	// Group mentions by partition keys
	partitioned := s.partitionOntologyMentions(mentions)

	for partKey, mns := range partitioned {
		if err := s.writeOntologyMentionsToParquet(partKey, mns); err != nil {
			return fmt.Errorf("failed to write ontology mentions partition %s: %w", partKey, err)
		}
	}

	log.Printf("ANALYTICS: Wrote %d ontology mentions to Hive-partitioned Parquet", len(mentions))
	return nil
}

// QueryOntologyEntities queries ontology entities from Hive-partitioned Parquet files
func (s *HiveParquetStorage) QueryOntologyEntities(filters map[string]interface{}) ([]OntologyEntity, error) {
	// Open DuckDB connection (in-memory)
	db, err := sql.Open("duckdb", "")
	if err != nil {
		return nil, fmt.Errorf("failed to open DuckDB connection: %w", err)
	}
	defer db.Close()

	// Build query based on filters
	query := fmt.Sprintf("SELECT entity_id, doc_id, source_name, entity_name, entity_type, domain, confidence, attributes, element_id, extracted_at FROM '%s/ontology_entities/**/*.parquet'",
		s.basePath)

	var whereClauses []string
	if source, ok := filters["source_name"].(string); ok && source != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("source_name = '%s'", source))
	}
	if docID, ok := filters["doc_id"].(string); ok && docID != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("doc_id = '%s'", docID))
	}
	if domain, ok := filters["domain"].(string); ok && domain != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("domain = '%s'", domain))
	}
	if entityType, ok := filters["entity_type"].(string); ok && entityType != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("entity_type = '%s'", entityType))
	}

	// Support nested attribute filtering (e.g., attributes.run_id)
	// DuckDB supports JSON path queries on JSON strings
	if runID, ok := filters["attributes.run_id"].(string); ok && runID != "" {
		// Use DuckDB's JSON string extraction: json_extract_string(attributes, '$.run_id')
		whereClauses = append(whereClauses, fmt.Sprintf("json_extract_string(attributes, '$.run_id') = '%s'", runID))
	}

	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	// Execute query
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query ontology entities: %w", err)
	}
	defer rows.Close()

	// Parse results
	var entities []OntologyEntity
	for rows.Next() {
		var entity OntologyEntity
		var attributesJSON string
		var extractedAtStr string

		err := rows.Scan(&entity.EntityID, &entity.DocID, &entity.SourceName, &entity.EntityName,
			&entity.EntityType, &entity.Domain, &entity.Confidence, &attributesJSON, &entity.ElementID, &extractedAtStr)
		if err != nil {
			log.Printf("Failed to scan ontology entity row: %v", err)
			continue
		}

		// Parse attributes from JSON
		if attributesJSON != "" {
			if err := json.Unmarshal([]byte(attributesJSON), &entity.Attributes); err != nil {
				log.Printf("Failed to unmarshal attributes for entity %s: %v", entity.EntityID, err)
			}
		}

		// Parse timestamp
		if entity.ExtractedAt, err = time.Parse(time.RFC3339, extractedAtStr); err != nil {
			entity.ExtractedAt = time.Now()
		}

		entities = append(entities, entity)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	log.Printf("ANALYTICS: Queried %d ontology entities from Hive-partitioned Parquet", len(entities))
	return entities, nil
}

// QueryOntologyRelationships queries ontology relationships from Hive-partitioned Parquet files
func (s *HiveParquetStorage) QueryOntologyRelationships(filters map[string]interface{}) ([]OntologyRelationship, error) {
	// Open DuckDB connection (in-memory)
	db, err := sql.Open("duckdb", "")
	if err != nil {
		return nil, fmt.Errorf("failed to open DuckDB connection: %w", err)
	}
	defer db.Close()

	// Build query based on filters
	query := fmt.Sprintf("SELECT relationship_id, doc_id, source_name, source_entity_id, target_entity_id, relationship_type, domain, confidence, evidence, attributes, element_id, extracted_at FROM '%s/ontology_relationships/**/*.parquet'",
		s.basePath)

	var whereClauses []string
	if source, ok := filters["source_name"].(string); ok && source != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("source_name = '%s'", source))
	}
	if docID, ok := filters["doc_id"].(string); ok && docID != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("doc_id = '%s'", docID))
	}
	if domain, ok := filters["domain"].(string); ok && domain != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("domain = '%s'", domain))
	}
	if relType, ok := filters["relationship_type"].(string); ok && relType != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("relationship_type = '%s'", relType))
	}

	// Support nested attribute filtering (e.g., attributes.run_id)
	// DuckDB supports JSON path queries on JSON strings
	if runID, ok := filters["attributes.run_id"].(string); ok && runID != "" {
		// Use DuckDB's JSON string extraction: json_extract_string(attributes, '$.run_id')
		whereClauses = append(whereClauses, fmt.Sprintf("json_extract_string(attributes, '$.run_id') = '%s'", runID))
	}

	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	// Execute query
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query ontology relationships: %w", err)
	}
	defer rows.Close()

	// Parse results
	var relationships []OntologyRelationship
	for rows.Next() {
		var rel OntologyRelationship
		var attributesJSON string
		var evidenceStr sql.NullString
		var extractedAtStr string

		err := rows.Scan(&rel.RelationshipID, &rel.DocID, &rel.SourceName, &rel.SourceEntityID,
			&rel.TargetEntityID, &rel.RelationshipType, &rel.Domain, &rel.Confidence,
			&evidenceStr, &attributesJSON, &rel.ElementID, &extractedAtStr)
		if err != nil {
			log.Printf("Failed to scan ontology relationship row: %v", err)
			continue
		}

		// Parse evidence (nullable)
		if evidenceStr.Valid {
			rel.Evidence = evidenceStr.String
		}

		// Parse attributes from JSON
		if attributesJSON != "" {
			if err := json.Unmarshal([]byte(attributesJSON), &rel.Attributes); err != nil {
				log.Printf("Failed to unmarshal attributes for relationship %s: %v", rel.RelationshipID, err)
			}
		}

		// Parse timestamp
		if rel.ExtractedAt, err = time.Parse(time.RFC3339, extractedAtStr); err != nil {
			rel.ExtractedAt = time.Now()
		}

		relationships = append(relationships, rel)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	log.Printf("ANALYTICS: Queried %d ontology relationships from Hive-partitioned Parquet", len(relationships))
	return relationships, nil
}

// GetContentResolver returns the content resolver for this storage backend
// This allows samplers and query engines to resolve element content from content_location pointers
func (s *HiveParquetStorage) GetContentResolver() interface{} {
	return s.contentResolver
}

// ============================================================================
// Corpus Exploration Methods - for MCP server and interactive tools
// HiveParquetStorage uses the same DuckDB-based implementation as ParquetStorage
// ============================================================================

// SearchSemanticSimilarity performs semantic similarity search using vector embeddings
func (s *HiveParquetStorage) SearchSemanticSimilarity(queryVector []float64, filters map[string]interface{}, threshold float64, limit int) ([]SearchResult, error) {
	return searchSemanticSimilarityImpl(s.basePath, queryVector, filters, threshold, limit)
}

// SearchByRegex performs regex pattern matching on element content
func (s *HiveParquetStorage) SearchByRegex(pattern string, filters map[string]interface{}, limit int) ([]SearchResult, error) {
	return searchByRegexImpl(s.basePath, pattern, filters, limit)
}

// SearchByKeyword performs keyword search on element content
func (s *HiveParquetStorage) SearchByKeyword(keyword string, filters map[string]interface{}, limit int) ([]SearchResult, error) {
	return searchByKeywordImpl(s.basePath, keyword, filters, limit)
}

// AnalyzePattern analyzes a regex pattern across the corpus
func (s *HiveParquetStorage) AnalyzePattern(pattern string, filters map[string]interface{}, maxExamples int) (*PatternStats, error) {
	return analyzePatternImpl(s.basePath, pattern, filters, maxExamples)
}

// ComputeTermFrequencies computes frequency statistics for given terms
func (s *HiveParquetStorage) ComputeTermFrequencies(terms []string, caseSensitive bool, filters map[string]interface{}) ([]TermFrequency, error) {
	return computeTermFrequenciesImpl(s.basePath, terms, caseSensitive, filters)
}

// FindCooccurrences finds co-occurrences of two entities within a context window
func (s *HiveParquetStorage) FindCooccurrences(entity1, entity2 string, contextWindow string, filters map[string]interface{}, maxExamples int) (*CooccurrenceResult, error) {
	return findCooccurrencesImpl(s.basePath, entity1, entity2, contextWindow, filters, maxExamples)
}

// GetElementContext retrieves an element with its hierarchical context
func (s *HiveParquetStorage) GetElementContext(elementID string, filters map[string]interface{}, contextDepth int, includeSiblings, includeChildren bool) (*ElementContext, error) {
	return getElementContextImpl(s.basePath, elementID, filters, contextDepth, includeSiblings, includeChildren)
}

// AggregateStatistics computes aggregate statistics about the corpus
func (s *HiveParquetStorage) AggregateStatistics(metrics []string, filters map[string]interface{}) (*CorpusStats, error) {
	return aggregateStatisticsImpl(s.basePath, metrics, filters)
}

// ============================================================================
// SQL-Based Ontology Extraction - Scalable to billions of elements
// ============================================================================

// GetAllDocIDs returns all unique document IDs in the corpus
// Used for distributed extraction task batching
func (s *HiveParquetStorage) GetAllDocIDs(filters map[string]interface{}) ([]string, error) {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		return nil, fmt.Errorf("failed to open DuckDB: %w", err)
	}
	defer db.Close()

	// Build query
	query := fmt.Sprintf(`
		SELECT DISTINCT doc_id
		FROM '%s/elements/**/*.parquet'
		ORDER BY doc_id
	`, s.basePath)

	// Execute query
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query doc IDs: %w", err)
	}
	defer rows.Close()

	// Collect results
	var docIDs []string
	for rows.Next() {
		var docID string
		if err := rows.Scan(&docID); err != nil {
			return nil, fmt.Errorf("failed to scan doc ID: %w", err)
		}
		docIDs = append(docIDs, docID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating doc IDs: %w", err)
	}

	return docIDs, nil
}

// ExtractAndStoreEntities executes entity extraction for a batch of documents
// and writes results directly to Parquet (no memory accumulation).
// This is the distributed extraction method used by workers.
func (s *HiveParquetStorage) ExtractAndStoreEntities(
	runID string,
	entityType string,
	docIDs []string,
	mappingJSON []byte,
	filters map[string]interface{},
	conceptEmbeddings map[string][]float64,
) (int, error) {
	log.Printf("Extracting entity type '%s' for %d documents", entityType, len(docIDs))

	// Parse mapping to check for LLM validation prompts
	// We need to import the ontology package types, but to avoid circular dependency
	// we'll define a minimal struct here just for parsing the validation prompt
	type llmPrompt struct {
		Prompt    string `json:"prompt"`
		BatchSize int    `json:"batch_size,omitempty"`
	}
	type extractionRule struct {
		LLMFalsePositiveTest *llmPrompt `json:"llm_false_positive_test,omitempty"`
	}
	type elementMapping struct {
		ExtractionRules []extractionRule `json:"extraction_rules"`
	}

	var mapping elementMapping
	var llmValidationPrompt *llmPrompt
	if err := json.Unmarshal(mappingJSON, &mapping); err != nil {
		log.Printf("WARNING: Failed to parse mapping JSON for LLM validation check: %v", err)
	} else {
		// Check if any extraction rule has LLM validation
		for _, rule := range mapping.ExtractionRules {
			if rule.LLMFalsePositiveTest != nil {
				llmValidationPrompt = rule.LLMFalsePositiveTest
				log.Printf("  LLM validation ENABLED for entity type '%s': %s", entityType, llmValidationPrompt.Prompt)
				break
			}
		}
	}

	// Build SQL query
	builder := NewExtractionQueryBuilder(s.basePath)

	// Add doc_ids filter
	filters["doc_ids"] = docIDs

	query, err := builder.BuildEntityExtractionQuery(mappingJSON, filters, conceptEmbeddings)
	if err != nil {
		return 0, fmt.Errorf("failed to build query: %w", err)
	}

	// DEBUG: Log the generated SQL query (only for person entities to reduce noise)
	if entityType == "person" {
		log.Printf("DEBUG: Generated SQL query for entity type 'person':\n%s", query.SQL)
	}

	// Execute query and stream results
	db, err := sql.Open("duckdb", "")
	if err != nil {
		return 0, fmt.Errorf("failed to open DuckDB: %w", err)
	}
	defer db.Close()

	rows, err := db.Query(query.SQL)
	if err != nil {
		return 0, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	// Write raw entity records directly (no deduplication)
	// Each SQL result becomes one entity record
	entities := make([]OntologyEntity, 0, 100)

	for rows.Next() {
		var elementID, docID, sourceName, domain, entityName string
		var confidence float64

		if err := rows.Scan(&elementID, &docID, &sourceName, &entityType, &domain, &entityName, &confidence); err != nil {
			return 0, fmt.Errorf("failed to scan row: %w", err)
		}

		// Create raw entity record with attributes
		attributes := make(map[string]interface{})

		// Store LLM validation prompt if configured for this mapping
		if llmValidationPrompt != nil {
			attributes["llm_validation_prompt"] = llmValidationPrompt.Prompt
			if llmValidationPrompt.BatchSize > 0 {
				attributes["llm_validation_batch_size"] = llmValidationPrompt.BatchSize
			}
		}

		entity := OntologyEntity{
			EntityID:    generateRandomHex(16), // Unique ID per extraction
			DocID:       docID,
			SourceName:  sourceName,
			EntityName:  entityName,
			EntityType:  entityType,
			Domain:      domain,
			Confidence:  confidence,
			Attributes:  attributes,
			ElementID:   elementID,
			RunID:       runID,
			ExtractedAt: time.Now(),
		}
		entities = append(entities, entity)
	}

	// Check for row iteration errors
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("error iterating rows: %w", err)
	}

	// Write raw entities to Parquet with Hive partitioning
	if len(entities) > 0 {
		partKey := fmt.Sprintf("run_id=%s", runID)
		if err := s.writeOntologyEntitiesToParquet(partKey, entities); err != nil {
			return 0, fmt.Errorf("failed to write entities: %w", err)
		}
	}

	entityCount := len(entities)
	if llmValidationPrompt != nil {
		log.Printf("  ✓ Completed extraction for entity type '%s': %d raw entities (LLM validation will be applied during consolidation)",
			entityType, entityCount)
	} else {
		log.Printf("  ✓ Completed extraction for entity type '%s': %d raw entities",
			entityType, entityCount)
	}
	return entityCount, nil
}

// Helper functions for entity ID generation and doc_id extraction
// These are used by ExtractAndStoreEntities

func generateEntityID() string {
	return fmt.Sprintf("entity_%s", generateRandomHex(16))
}

func generateRandomHex(n int) string {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based random if crypto/rand fails
		return fmt.Sprintf("%x", time.Now().UnixNano())[:n]
	}
	return hex.EncodeToString(bytes)[:n]
}

// generateStableID generates a deterministic hash from a dedup key
// This ensures the same entity name always gets the same ID
func generateStableID(dedupKey string) string {
	h := sha256.New()
	h.Write([]byte(dedupKey))
	hash := h.Sum(nil)
	return hex.EncodeToString(hash)[:16]
}

func extractDocIDFromElementID(elementID string) string {
	// Split by underscores
	parts := strings.Split(elementID, "_")
	if len(parts) < 2 {
		// If format is unexpected, return the whole elementID
		return elementID
	}

	// Element ID format: <doc_id>_<element_type>_<hash>
	// doc_id itself may contain underscores, so we need to find where element_type starts
	// Common element types: paragraph, text, list_item, table, title, heading
	elementTypes := []string{"paragraph", "text", "list_item", "list", "table", "title", "heading", "hyperlink", "diagram", "image"}

	// Scan from right to left to find element type
	for i := len(parts) - 2; i >= 1; i-- {
		for _, elemType := range elementTypes {
			if parts[i] == elemType {
				// Found element type at position i
				// doc_id is everything before position i
				return strings.Join(parts[:i], "_")
			}
		}
	}

	// Fallback: assume last 2 parts are element_type and hash
	// Return everything except last 2 parts
	if len(parts) > 2 {
		return strings.Join(parts[:len(parts)-2], "_")
	}

	return elementID
}

// Cross-domain entity merging data structures

// DomainEntityInstance represents a canonical entity from a specific domain
type DomainEntityInstance struct {
	EntityID     string   // Canonical entity ID from domain
	Domain       string   // Domain name
	EntityName   string   // Entity name
	Confidence   float64  // Confidence score
	MentionCount int      // Number of mentions
	Contexts     []string // Sample contexts (up to 3)
}

// CrossDomainEntityCandidate represents entities with same name across multiple domains
type CrossDomainEntityCandidate struct {
	EntityType string                 // Entity type (e.g., "person", "organization")
	EntityName string                 // Normalized entity name
	Instances  []DomainEntityInstance // Instances from different domains
}

// CrossDomainMergeJudgment represents LLM's decision on whether to merge cross-domain entities
type CrossDomainMergeJudgment struct {
	EntityType  string  // Entity type
	EntityName  string  // Entity name
	ShouldMerge bool    // Whether to create global entity
	Confidence  float64 // LLM confidence (0.0-1.0)
	Reasoning   string  // LLM explanation
}

// findCrossDomainCandidates identifies entities appearing in multiple domains
func (s *HiveParquetStorage) findCrossDomainCandidates(runID string, minDomains int) ([]CrossDomainEntityCandidate, error) {
	// Open DuckDB connection
	db, err := sql.Open("duckdb", "")
	if err != nil {
		return nil, fmt.Errorf("failed to open DuckDB: %w", err)
	}
	defer db.Close()

	// Query to find entities appearing in multiple domains
	query := fmt.Sprintf(`
WITH canonical_with_contexts AS (
  SELECT
    ce.entity_type,
    ce.entity_name,
    ce.domain,
    ce.entity_id,
    ce.confidence,
    ce.mention_count,
    LIST(em.context ORDER BY em.confidence DESC LIMIT 3) as contexts
  FROM '%s/canonical_entities/run_id=%s/*.parquet' ce
  LEFT JOIN '%s/entity_mentions/run_id=%s/*.parquet' em
    ON ce.representative_entity_id = em.entity_id
  GROUP BY ce.entity_type, ce.entity_name, ce.domain, ce.entity_id, ce.confidence, ce.mention_count
),
cross_domain_groups AS (
  SELECT
    entity_type,
    LOWER(TRIM(entity_name)) as normalized_name,
    COUNT(DISTINCT domain) as domain_count,
    LIST(STRUCT_PACK(
      entity_id := entity_id,
      domain := domain,
      entity_name := entity_name,
      confidence := confidence,
      mention_count := mention_count,
      contexts := contexts
    )) as instances
  FROM canonical_with_contexts
  GROUP BY entity_type, LOWER(TRIM(entity_name))
  HAVING COUNT(DISTINCT domain) >= %d
)
SELECT
  entity_type,
  normalized_name,
  domain_count,
  instances
FROM cross_domain_groups
ORDER BY entity_type, normalized_name;
	`, s.basePath, runID, s.basePath, runID, minDomains)

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query cross-domain candidates: %w", err)
	}
	defer rows.Close()

	var candidates []CrossDomainEntityCandidate
	for rows.Next() {
		var entityType, normalizedName string
		var domainCount int
		var instancesJSON string

		if err := rows.Scan(&entityType, &normalizedName, &domainCount, &instancesJSON); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Parse instances JSON
		var rawInstances []map[string]interface{}
		if err := json.Unmarshal([]byte(instancesJSON), &rawInstances); err != nil {
			return nil, fmt.Errorf("failed to parse instances JSON: %w", err)
		}

		// Convert to DomainEntityInstance structs
		var instances []DomainEntityInstance
		for _, raw := range rawInstances {
			instance := DomainEntityInstance{
				EntityID:     raw["entity_id"].(string),
				Domain:       raw["domain"].(string),
				EntityName:   raw["entity_name"].(string),
				Confidence:   raw["confidence"].(float64),
				MentionCount: int(raw["mention_count"].(float64)),
			}

			// Parse contexts array
			if contextsRaw, ok := raw["contexts"].([]interface{}); ok {
				for _, ctx := range contextsRaw {
					if ctxStr, ok := ctx.(string); ok {
						instance.Contexts = append(instance.Contexts, ctxStr)
					}
				}
			}

			instances = append(instances, instance)
		}

		candidates = append(candidates, CrossDomainEntityCandidate{
			EntityType: entityType,
			EntityName: normalizedName,
			Instances:  instances,
		})
	}

	return candidates, nil
}

// llmBatchJudgeCrossDomainMerges uses LLM to determine which cross-domain entities should be merged
func (s *HiveParquetStorage) llmBatchJudgeCrossDomainMerges(ctx context.Context, candidates []CrossDomainEntityCandidate, llmValidator interface{}, threshold float64) ([]CrossDomainMergeJudgment, error) {
	if len(candidates) == 0 {
		return nil, nil
	}

	// Use reflection to call LLM Complete method
	validatorVal := reflect.ValueOf(llmValidator)
	completeMethod := validatorVal.MethodByName("Complete")
	if !completeMethod.IsValid() {
		return nil, fmt.Errorf("LLM validator does not have Complete method")
	}

	// Process in batches of 50
	batchSize := 50
	var allJudgments []CrossDomainMergeJudgment

	for i := 0; i < len(candidates); i += batchSize {
		end := i + batchSize
		if end > len(candidates) {
			end = len(candidates)
		}
		batch := candidates[i:end]

		// Build prompt for this batch
		prompt := s.buildCrossDomainMergePrompt(batch)

		// Call LLM Complete method via reflection
		// Complete(ctx, prompt, options) (string, error)
		callArgs := []reflect.Value{
			reflect.ValueOf(ctx),
			reflect.ValueOf(prompt),
			reflect.ValueOf(llmOptions{
				maxTokens:    4000,
				temperature:  0.0,
				systemPrompt: "You are an expert at entity resolution and determining if entities from different domains represent the same real-world entity.",
			}),
		}

		results := completeMethod.Call(callArgs)
		if len(results) != 2 {
			return nil, fmt.Errorf("unexpected number of return values from Complete method")
		}

		// Check for error
		if !results[1].IsNil() {
			return nil, fmt.Errorf("LLM call failed: %v", results[1].Interface())
		}

		// Parse response
		response := results[0].String()
		judgments, err := s.parseCrossDomainMergeResponse(response, batch)
		if err != nil {
			return nil, fmt.Errorf("failed to parse LLM response: %w", err)
		}

		// Filter by confidence threshold
		for _, judgment := range judgments {
			if judgment.Confidence >= threshold {
				allJudgments = append(allJudgments, judgment)
			}
		}

		// Rate limiting - small delay between batches
		if end < len(candidates) {
			time.Sleep(100 * time.Millisecond)
		}
	}

	return allJudgments, nil
}

// buildCrossDomainMergePrompt creates the LLM prompt for cross-domain entity merging
func (s *HiveParquetStorage) buildCrossDomainMergePrompt(candidates []CrossDomainEntityCandidate) string {
	var sb strings.Builder

	sb.WriteString("# Cross-Domain Entity Merging Task\n\n")
	sb.WriteString("Analyze the following entities that appear in multiple domains and determine if they represent the same real-world entity.\n\n")
	sb.WriteString("For each entity, provide:\n")
	sb.WriteString("- should_merge: true/false\n")
	sb.WriteString("- confidence: 0.0-1.0 (how confident you are)\n")
	sb.WriteString("- reasoning: brief explanation\n\n")
	sb.WriteString("## Entities to Analyze\n\n")

	for i, candidate := range candidates {
		sb.WriteString(fmt.Sprintf("### Entity %d\n", i+1))
		sb.WriteString(fmt.Sprintf("- Type: %s\n", candidate.EntityType))
		sb.WriteString(fmt.Sprintf("- Name: %s\n", candidate.EntityName))
		sb.WriteString(fmt.Sprintf("- Appears in %d domains:\n", len(candidate.Instances)))

		for _, instance := range candidate.Instances {
			sb.WriteString(fmt.Sprintf("  - Domain: %s (confidence: %.2f, mentions: %d)\n",
				instance.Domain, instance.Confidence, instance.MentionCount))
			if len(instance.Contexts) > 0 {
				sb.WriteString("    Sample contexts:\n")
				for _, ctx := range instance.Contexts {
					sb.WriteString(fmt.Sprintf("    - %s\n", ctx))
				}
			}
		}
		sb.WriteString("\n")
	}

	sb.WriteString("\n## Response Format\n\n")
	sb.WriteString("Return a JSON array with one judgment per entity:\n")
	sb.WriteString("```json\n")
	sb.WriteString("[\n")
	sb.WriteString("  {\n")
	sb.WriteString("    \"entity_type\": \"person\",\n")
	sb.WriteString("    \"entity_name\": \"john smith\",\n")
	sb.WriteString("    \"should_merge\": true,\n")
	sb.WriteString("    \"confidence\": 0.95,\n")
	sb.WriteString("    \"reasoning\": \"Same person mentioned in medical and education contexts with consistent titles\"\n")
	sb.WriteString("  }\n")
	sb.WriteString("]\n")
	sb.WriteString("```\n")

	return sb.String()
}

// createCrossDomainGlobalEntities creates global entities for cross-domain merges
func (s *HiveParquetStorage) createCrossDomainGlobalEntities(ctx context.Context, runID string, schema interface{}, llmValidator interface{}) error {
	// Use reflection to get CrossDomainMerging config from schema
	schemaVal := reflect.ValueOf(schema)
	if schemaVal.Kind() == reflect.Ptr {
		schemaVal = schemaVal.Elem()
	}

	crossDomainField := schemaVal.FieldByName("CrossDomainMerging")
	if !crossDomainField.IsValid() || crossDomainField.IsNil() {
		log.Printf("  Cross-domain merging not configured - skipping")
		return nil
	}

	// Extract configuration values
	crossDomainConfig := crossDomainField.Elem()
	enabled := crossDomainConfig.FieldByName("Enabled").Bool()
	if !enabled {
		log.Printf("  Cross-domain merging disabled - skipping")
		return nil
	}

	threshold := crossDomainConfig.FieldByName("SimilarityThreshold").Float()
	minDomainsField := crossDomainConfig.FieldByName("MinDomains")
	minDomains := 2 // default
	if minDomainsField.IsValid() && minDomainsField.Int() > 0 {
		minDomains = int(minDomainsField.Int())
	}

	log.Printf("========================================")
	log.Printf("CROSS-DOMAIN ENTITY MERGING")
	log.Printf("========================================")
	log.Printf("  Run ID: %s", runID)
	log.Printf("  Min domains: %d", minDomains)
	log.Printf("  Similarity threshold: %.2f", threshold)
	log.Printf("========================================\n")

	// Find cross-domain candidates
	log.Printf("  Finding entities appearing in multiple domains...")
	candidates, err := s.findCrossDomainCandidates(runID, minDomains)
	if err != nil {
		return fmt.Errorf("failed to find cross-domain candidates: %w", err)
	}

	if len(candidates) == 0 {
		log.Printf("  ✓ No cross-domain candidates found")
		log.Printf("========================================\n")
		return nil
	}

	log.Printf("  ✓ Found %d cross-domain candidate entities", len(candidates))

	// LLM judgment on whether to merge
	if llmValidator == nil {
		log.Printf("  WARNING: LLM validator not available - skipping cross-domain merging")
		log.Printf("========================================\n")
		return nil
	}

	log.Printf("  Calling LLM to judge cross-domain merges...")
	judgments, err := s.llmBatchJudgeCrossDomainMerges(ctx, candidates, llmValidator, threshold)
	if err != nil {
		log.Printf("  WARNING: LLM judgment failed: %v - skipping cross-domain merging", err)
		log.Printf("========================================\n")
		return nil
	}

	log.Printf("  ✓ LLM approved %d merges (threshold: %.2f)", len(judgments), threshold)

	if len(judgments) == 0 {
		log.Printf("  ✓ No entities meet merging criteria")
		log.Printf("========================================\n")
		return nil
	}

	// Create global entities
	log.Printf("  Creating global entities and has_instance relationships...")

	var globalEntities []CanonicalEntity
	var hasInstanceRelationships []OntologyRelationship

	for _, judgment := range judgments {
		// Find the candidate for this judgment
		var candidate *CrossDomainEntityCandidate
		for i := range candidates {
			if candidates[i].EntityType == judgment.EntityType &&
				candidates[i].EntityName == judgment.EntityName {
				candidate = &candidates[i]
				break
			}
		}

		if candidate == nil {
			continue
		}

		// Generate global entity ID
		dedupKey := fmt.Sprintf("%s.global.%s", judgment.EntityType, strings.ToLower(strings.TrimSpace(judgment.EntityName)))
		globalEntityID := generateStableID(dedupKey)

		// Calculate total mention count across domains
		totalMentions := 0
		for _, instance := range candidate.Instances {
			totalMentions += instance.MentionCount
		}

		// Create global canonical entity
		globalEntity := CanonicalEntity{
			EntityID:               globalEntityID,
			RepresentativeEntityID: candidate.Instances[0].EntityID, // Use first instance as representative
			EntityName:             judgment.EntityName,
			EntityType:             judgment.EntityType,
			Domain:                 "global",
			Confidence:             judgment.Confidence,
			MentionCount:           totalMentions,
			Strategy:               "cross_domain_merge",
			Attributes: map[string]interface{}{
				"llm_merge_confidence": judgment.Confidence,
				"llm_merge_reasoning":  judgment.Reasoning,
				"source_domains":       len(candidate.Instances),
			},
			RunID:     runID,
			CreatedAt: time.Now(),
		}
		globalEntities = append(globalEntities, globalEntity)

		// Create has_instance relationships for each domain instance
		for _, instance := range candidate.Instances {
			relationshipID := fmt.Sprintf("rel_%s", generateRandomHex(16))
			relationship := OntologyRelationship{
				RelationshipID:   relationshipID,
				SourceEntityID:   globalEntityID,
				TargetEntityID:   instance.EntityID,
				RelationshipType: "has_instance",
				Confidence:       judgment.Confidence,
				Evidence:         fmt.Sprintf("Global entity has instance in %s domain", instance.Domain),
				Attributes: map[string]interface{}{
					"domain":                   instance.Domain,
					"instance_confidence":      instance.Confidence,
					"instance_mention_count":   instance.MentionCount,
					"cross_domain_merge":       true,
					"llm_merge_confidence":     judgment.Confidence,
				},
				RunID:       runID,
				ExtractedAt: time.Now(),
			}
			hasInstanceRelationships = append(hasInstanceRelationships, relationship)
		}
	}

	// Write global entities to Parquet (domain=global partition)
	if len(globalEntities) > 0 {
		if err := s.writeGlobalEntitiesToParquet(runID, globalEntities); err != nil {
			return fmt.Errorf("failed to write global entities: %w", err)
		}
		log.Printf("  ✓ Wrote %d global entities to domain=global partition", len(globalEntities))
	}

	// Write has_instance relationships to Parquet
	if len(hasInstanceRelationships) > 0 {
		if err := s.writeOntologyRelationshipsToParquet(runID, hasInstanceRelationships); err != nil {
			return fmt.Errorf("failed to write has_instance relationships: %w", err)
		}
		log.Printf("  ✓ Wrote %d has_instance relationships", len(hasInstanceRelationships))
	}

	log.Printf("  ✓ Cross-domain merging complete: %d global entities created", len(globalEntities))
	log.Printf("========================================\n")
	return nil
}

// writeGlobalEntitiesToParquet writes global entities to the domain=global partition
func (s *HiveParquetStorage) writeGlobalEntitiesToParquet(runID string, entities []CanonicalEntity) error {
	// Override domain to "global" for all entities
	for i := range entities {
		entities[i].Domain = "global"
	}

	// Use existing writeCanonicalEntitiesToParquet - it will use the domain field for partitioning
	return s.writeCanonicalEntitiesToParquet(runID, entities)
}

// parseCrossDomainMergeResponse parses LLM JSON response into judgments
func (s *HiveParquetStorage) parseCrossDomainMergeResponse(response string, candidates []CrossDomainEntityCandidate) ([]CrossDomainMergeJudgment, error) {
	// Extract JSON from response (may be wrapped in markdown code blocks)
	jsonStart := strings.Index(response, "[")
	jsonEnd := strings.LastIndex(response, "]")
	if jsonStart == -1 || jsonEnd == -1 {
		return nil, fmt.Errorf("no JSON array found in response")
	}

	jsonStr := response[jsonStart : jsonEnd+1]

	var rawJudgments []struct {
		EntityType  string  `json:"entity_type"`
		EntityName  string  `json:"entity_name"`
		ShouldMerge bool    `json:"should_merge"`
		Confidence  float64 `json:"confidence"`
		Reasoning   string  `json:"reasoning"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &rawJudgments); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	var judgments []CrossDomainMergeJudgment
	for _, raw := range rawJudgments {
		if raw.ShouldMerge {
			judgments = append(judgments, CrossDomainMergeJudgment{
				EntityType:  raw.EntityType,
				EntityName:  raw.EntityName,
				ShouldMerge: raw.ShouldMerge,
				Confidence:  raw.Confidence,
				Reasoning:   raw.Reasoning,
			})
		}
	}

	return judgments, nil
}

// ConsolidateEntities performs global entity resolution on raw extractions
// Creates canonical entities by deduplicating raw extractions
// llmValidator should be an *ontology.LLMValidator for validation, or nil to disable
// schema should be an *ontology.OntologySchema for cross-domain merging configuration, or nil to disable
func (s *HiveParquetStorage) ConsolidateEntities(runID string, strategy string, llmValidator interface{}, schema interface{}) error {
	log.Printf("========================================")
	log.Printf("ENTITY CONSOLIDATION")
	log.Printf("========================================")
	log.Printf("  Run ID: %s", runID)
	log.Printf("  Strategy: %s", strategy)
	log.Printf("  Storage: %s", s.basePath)

	// LLM validation status
	if llmValidator != nil {
		log.Printf("  LLM Validation: ENABLED")
	} else {
		log.Printf("  LLM Validation: DISABLED")
	}
	log.Printf("========================================\n")

	// Open DuckDB connection
	db, err := sql.Open("duckdb", "")
	if err != nil {
		return fmt.Errorf("failed to open DuckDB: %w", err)
	}
	defer db.Close()

	// Check which entity types have LLM validation templates available
	// Use reflection to avoid circular dependency with ontology package
	// Map of entity_type -> validation prompt (string is kept for backward compatibility,
	// but actual prompts are generated from templates in llm_validator.go)
	entityTypeValidation := make(map[string]string)

	if llmValidator != nil {
		// Get available entity types using reflection
		// llmValidator is an LLMValidator from ontology package
		// We need to call GetAvailableEntityTypes() method
		validatorVal := reflect.ValueOf(llmValidator)
		getTypesMethod := validatorVal.MethodByName("GetAvailableEntityTypes")

		if getTypesMethod.IsValid() {
			// Call GetAvailableEntityTypes()
			callResults := getTypesMethod.Call([]reflect.Value{})
			if len(callResults) == 1 {
				// Extract []string result
				typesVal := callResults[0]
				if typesVal.Kind() == reflect.Slice {
					for i := 0; i < typesVal.Len(); i++ {
						entityType := typesVal.Index(i).String()
						// Mark this entity type as having validation available
						// The actual prompt is generated from templates, not stored here
						entityTypeValidation[entityType] = "template-based"
						log.Printf("  Entity type '%s' has LLM validation template", entityType)
					}
				}
			}
		} else {
			log.Printf("  WARNING: LLM validator does not have GetAvailableEntityTypes method - LLM validation disabled")
		}
	}

	// Build DuckDB query to consolidate raw entities
	// Filter out empty/whitespace entity names - these are invalid extractions
	// Select representative entity (highest confidence) for each canonical
	query := fmt.Sprintf(`
WITH raw_entities AS (
  SELECT
    entity_id,
    entity_name,
    entity_type,
    domain,
    confidence
  FROM '%s/ontology_entities/run_id=%s/*.parquet'
  WHERE entity_name IS NOT NULL
    AND TRIM(entity_name) != ''
),
canonical AS (
  SELECT
    entity_type || '.' || LOWER(TRIM(entity_name)) as dedup_key,
    LOWER(TRIM(entity_name)) as entity_name,
    entity_type,
    FIRST(domain) as domain,
    MAX(confidence) as confidence,
    COUNT(*) as mention_count,
    ARG_MAX(entity_id, confidence) as representative_entity_id
  FROM raw_entities
  GROUP BY entity_type, LOWER(TRIM(entity_name))
)
SELECT
  dedup_key,
  entity_name,
  entity_type,
  domain,
  confidence,
  mention_count,
  representative_entity_id
FROM canonical
ORDER BY entity_type, entity_name;
	`, s.basePath, runID)

	log.Println("Executing consolidation query...")
	rows, err := db.Query(query)
	if err != nil {
		return fmt.Errorf("failed to execute consolidation query: %w", err)
	}
	defer rows.Close()

	// Collect canonical entities grouped by entity type for validation
	entitiesByType := make(map[string][]CanonicalEntity)

	for rows.Next() {
		var dedupKey, entityName, entityType, domain, representativeEntityID string
		var confidence float64
		var mentionCount int

		if err := rows.Scan(&dedupKey, &entityName, &entityType, &domain, &confidence, &mentionCount, &representativeEntityID); err != nil {
			return fmt.Errorf("failed to scan row: %w", err)
		}

		// Generate stable entity ID from dedup key
		entityID := generateStableID(dedupKey)

		canonical := CanonicalEntity{
			EntityID:               entityID,
			RepresentativeEntityID: representativeEntityID,
			EntityName:             entityName,
			EntityType:             entityType,
			Domain:                 domain,
			Confidence:             confidence,
			MentionCount:           mentionCount,
			Strategy:               strategy,
			Attributes:             make(map[string]interface{}),
			RunID:                  runID,
			CreatedAt:              time.Now(),
		}

		entitiesByType[entityType] = append(entitiesByType[entityType], canonical)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating rows: %w", err)
	}

	totalEntities := 0
	for _, entities := range entitiesByType {
		totalEntities += len(entities)
	}
	log.Printf("  ✓ Resolved %d canonical entities from raw extractions", totalEntities)

	// Apply LLM validation to entity types that have validation prompts
	finalEntities := make([]CanonicalEntity, 0, totalEntities)

	for entityType, entities := range entitiesByType {
		validationPrompt, hasValidation := entityTypeValidation[entityType]

		if !hasValidation || llmValidator == nil {
			// No validation needed - include all entities
			finalEntities = append(finalEntities, entities...)
			log.Printf("  ✓ Entity type '%s': %d entities (no LLM validation)", entityType, len(entities))
			continue
		}

		// Apply LLM validation
		log.Printf("  Applying LLM validation to entity type '%s' (%d entities)...", entityType, len(entities))

		// Build validation entities
		validationEntities := make([]entityToValidate, len(entities))
		for i, entity := range entities {
			validationEntities[i] = entityToValidate{
				EntityName: entity.EntityName,
				Prompt:     validationPrompt,
				EntityType: entityType,
			}
		}

		// Call LLM validation in batches (batch size 50)
		// batchValidateLLM uses reflection to call the Complete method
		validationResults, err := s.batchValidateLLM(context.Background(), llmValidator, validationEntities, 50)
		if err != nil {
			log.Printf("  WARNING: LLM validation failed: %v - including all entities (permissive)", err)
			finalEntities = append(finalEntities, entities...)
			continue
		}

		// Filter entities based on validation results
		acceptedCount := 0
		rejectedCount := 0
		for i, isValid := range validationResults {
			if isValid {
				// Store validation info in attributes
				entities[i].Attributes["llm_validation_prompt"] = validationPrompt
				entities[i].Attributes["llm_validation_result"] = "accepted"
				finalEntities = append(finalEntities, entities[i])
				acceptedCount++
			} else {
				rejectedCount++
				log.Printf("    LLM rejected entity: '%s' (type=%s)", entities[i].EntityName, entityType)
			}
		}

		log.Printf("  ✓ LLM validation complete: %d accepted, %d rejected", acceptedCount, rejectedCount)
	}

	// Write canonical entities to Parquet
	if len(finalEntities) > 0 {
		if err := s.writeCanonicalEntitiesToParquet(runID, finalEntities); err != nil {
			return fmt.Errorf("failed to write canonical entities: %w", err)
		}
	}

	// Write entity mentions mapping to Parquet
	if err := s.writeEntityMentionsToParquet(runID); err != nil {
		return fmt.Errorf("failed to write entity mentions: %w", err)
	}

	log.Printf("  ✓ Consolidation complete: %d final canonical entities", len(finalEntities))
	log.Printf("========================================\n")

	// Perform cross-domain entity merging if configured
	if schema != nil {
		if err := s.createCrossDomainGlobalEntities(context.Background(), runID, schema, llmValidator); err != nil {
			return fmt.Errorf("failed to create cross-domain global entities: %w", err)
		}
	}

	return nil
}

// llmOptions implements llmOptionsInterface
type llmOptions struct {
	maxTokens    int
	temperature  float64
	systemPrompt string
}

func (o llmOptions) GetMaxTokens() int      { return o.maxTokens }
func (o llmOptions) GetTemperature() float64 { return o.temperature }
func (o llmOptions) GetSystemPrompt() string { return o.systemPrompt }

// batchValidateLLM validates entities using LLM in batches via ValidateInBatches reflection call
// Returns []bool in same order as input entities
// On error, returns all false (strict - reject entities)
func (s *HiveParquetStorage) batchValidateLLM(ctx context.Context, validator interface{}, entities []entityToValidate, batchSize int) ([]bool, error) {
	if len(entities) == 0 {
		return []bool{}, nil
	}

	// Convert []entityToValidate to the format expected by LLMValidator.ValidateInBatches
	// The ontology package defines: type EntityToValidate struct { EntityName string; Prompt string; EntityType string }
	// We need to create a slice of these structs dynamically using reflection

	validatorVal := reflect.ValueOf(validator)
	validateMethod := validatorVal.MethodByName("ValidateInBatches")

	if !validateMethod.IsValid() {
		// Strict error handling - on failure, reject all entities
		log.Printf("WARNING: Validator does not have ValidateInBatches method - rejecting all entities (strict)")
		results := make([]bool, len(entities))
		// results are already false (zero value)
		return results, nil
	}

	// Get the method signature to determine the EntityToValidate type
	validateMethodType := validateMethod.Type()
	if validateMethodType.NumIn() != 3 {
		log.Printf("WARNING: ValidateInBatches has unexpected signature - rejecting all entities (strict)")
		results := make([]bool, len(entities))
		return results, nil
	}

	// Get the slice element type (EntityToValidate)
	entitiesParamType := validateMethodType.In(1) // 2nd parameter (0=ctx, 1=entities, 2=batchSize)
	if entitiesParamType.Kind() != reflect.Slice {
		log.Printf("WARNING: ValidateInBatches entities parameter is not a slice - rejecting all entities (strict)")
		results := make([]bool, len(entities))
		return results, nil
	}
	entityType := entitiesParamType.Elem()

	// Create a slice of EntityToValidate structs using reflection
	entitiesSlice := reflect.MakeSlice(entitiesParamType, len(entities), len(entities))
	for i, entity := range entities {
		entityStruct := reflect.New(entityType).Elem()

		// Set fields: EntityName, Prompt, EntityType
		entityNameField := entityStruct.FieldByName("EntityName")
		if entityNameField.IsValid() && entityNameField.CanSet() {
			entityNameField.SetString(entity.EntityName)
		}

		promptField := entityStruct.FieldByName("Prompt")
		if promptField.IsValid() && promptField.CanSet() {
			promptField.SetString(entity.Prompt)
		}

		entityTypeField := entityStruct.FieldByName("EntityType")
		if entityTypeField.IsValid() && entityTypeField.CanSet() {
			entityTypeField.SetString(entity.EntityType)
		}

		entitiesSlice.Index(i).Set(entityStruct)
	}

	// Call ValidateInBatches(ctx context.Context, entities []EntityToValidate, batchSize int) ([]bool, error)
	args := []reflect.Value{
		reflect.ValueOf(ctx),
		entitiesSlice,
		reflect.ValueOf(batchSize),
	}

	callResults := validateMethod.Call(args)
	if len(callResults) != 2 {
		log.Printf("WARNING: ValidateInBatches returned unexpected number of values - rejecting all entities (strict)")
		results := make([]bool, len(entities))
		return results, nil
	}

	// Extract results and error
	var results []bool
	var err error

	if callResults[0].IsValid() && callResults[0].CanInterface() {
		if resultsInterface := callResults[0].Interface(); resultsInterface != nil {
			results = resultsInterface.([]bool)
		}
	}

	if !callResults[1].IsNil() {
		err = callResults[1].Interface().(error)
	}

	if err != nil {
		// Strict error handling - on failure, reject all entities
		log.Printf("WARNING: LLM validation failed: %v - rejecting all entities (strict)", err)
		results = make([]bool, len(entities))
		return results, nil
	}

	// Verify results length matches
	if len(results) != len(entities) {
		log.Printf("WARNING: ValidateInBatches returned %d results but expected %d - rejecting all entities (strict)",
			len(results), len(entities))
		results = make([]bool, len(entities))
		return results, nil
	}

	return results, nil
}

// writeCanonicalEntitiesToParquet writes canonical entities to Parquet storage
func (s *HiveParquetStorage) writeCanonicalEntitiesToParquet(runID string, entities []CanonicalEntity) error {
	if len(entities) == 0 {
		return nil
	}

	// Create output directory with run_id partition
	outputDir := filepath.Join(s.basePath, "canonical_entities", fmt.Sprintf("run_id=%s", runID))
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Generate unique filename
	filename := fmt.Sprintf("canonical_entities_%s.parquet", generateRandomHex(8))
	outputPath := filepath.Join(outputDir, filename)

	// Open DuckDB connection
	db, err := sql.Open("duckdb", "")
	if err != nil {
		return fmt.Errorf("failed to open DuckDB: %w", err)
	}
	defer db.Close()

	// Use DuckDB to write Parquet
	placeholders := make([]string, len(entities))
	for i := range entities {
		placeholders[i] = fmt.Sprintf(
			"('%s', '%s', '%s', '%s', '%s', %.2f, %d, '%s', '%s', '%s')",
			escapeSQL(entities[i].EntityID),
			escapeSQL(entities[i].RepresentativeEntityID),
			escapeSQL(entities[i].EntityName),
			escapeSQL(entities[i].EntityType),
			escapeSQL(entities[i].Domain),
			entities[i].Confidence,
			entities[i].MentionCount,
			escapeSQL(entities[i].Strategy),
			runID,
			entities[i].CreatedAt.Format(time.RFC3339),
		)
	}

	query := fmt.Sprintf(`
		COPY (
			SELECT
				entity_id,
				representative_entity_id,
				entity_name,
				entity_type,
				domain,
				confidence,
				mention_count,
				strategy,
				run_id,
				CAST(created_at AS TIMESTAMP) as created_at
			FROM (VALUES %s) AS t(entity_id, representative_entity_id, entity_name, entity_type, domain, confidence, mention_count, strategy, run_id, created_at)
		) TO '%s' (FORMAT PARQUET, COMPRESSION ZSTD);
	`, strings.Join(placeholders, ", "), outputPath)

	if _, err := db.Exec(query); err != nil {
		return fmt.Errorf("failed to write canonical entities to Parquet: %w", err)
	}

	log.Printf("  Wrote %d canonical entities to: %s", len(entities), outputPath)
	return nil
}

// writeEntityMentionsToParquet writes entity mentions mapping to Parquet storage
// Creates mappings from canonical entity IDs to all raw entity IDs (mentions)
func (s *HiveParquetStorage) writeEntityMentionsToParquet(runID string) error {
	// Create output directory with run_id partition
	outputDir := filepath.Join(s.basePath, "entity_mentions", fmt.Sprintf("run_id=%s", runID))
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Generate unique filename
	filename := fmt.Sprintf("entity_mentions_%s.parquet", generateRandomHex(8))
	outputPath := filepath.Join(outputDir, filename)

	// Open DuckDB connection
	db, err := sql.Open("duckdb", "")
	if err != nil {
		return fmt.Errorf("failed to open DuckDB: %w", err)
	}
	defer db.Close()

	// Query to generate entity mentions from raw entities
	// Maps each raw entity to its canonical entity using the dedup key
	query := fmt.Sprintf(`
		COPY (
			WITH raw_entities AS (
				SELECT
					entity_id as raw_entity_id,
					entity_name,
					entity_type
				FROM '%s/ontology_entities/run_id=%s/*.parquet'
				WHERE entity_name IS NOT NULL
					AND TRIM(entity_name) != ''
			),
			canonical_mapping AS (
				SELECT
					entity_type || '.' || LOWER(TRIM(entity_name)) as dedup_key,
					raw_entity_id
				FROM raw_entities
			)
			SELECT
				'%s' || to_hex(sha256(dedup_key)) as canonical_entity_id,
				raw_entity_id,
				'%s' as run_id,
				CURRENT_TIMESTAMP as mapped_at
			FROM canonical_mapping
			ORDER BY canonical_entity_id, raw_entity_id
		) TO '%s' (FORMAT PARQUET, COMPRESSION ZSTD);
	`, s.basePath, runID, "ont_canonical_", runID, outputPath)

	if _, err := db.Exec(query); err != nil {
		return fmt.Errorf("failed to write entity mentions to Parquet: %w", err)
	}

	// Count mentions written
	var mentionCount int64
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*) FROM '%s/ontology_entities/run_id=%s/*.parquet'
		WHERE entity_name IS NOT NULL AND TRIM(entity_name) != ''
	`, s.basePath, runID)

	if err := db.QueryRow(countQuery).Scan(&mentionCount); err != nil {
		log.Printf("  WARNING: Failed to count entity mentions: %v", err)
	} else {
		log.Printf("  Wrote %d entity mentions to: %s", mentionCount, outputPath)
	}

	return nil
}

// ExtractEntitiesSQL executes ontology entity extraction entirely within DuckDB
// This method processes elements in-database using SQL queries, avoiding the need
// to load all elements into Go memory. Scales to billions of elements.
func (s *HiveParquetStorage) ExtractEntitiesSQL(
	schemaJSON []byte,
	filters map[string]interface{},
	conceptEmbeddings map[string][]float64,
) ([]byte, error) {
	log.Printf("========================================")
	log.Printf("SQL-BASED ENTITY EXTRACTION")
	log.Printf("========================================")
	log.Printf("  Storage: %s", s.basePath)
	log.Printf("  Filters: %v", filters)
	log.Printf("  Concept embeddings: %d", len(conceptEmbeddings))
	log.Printf("========================================")

	// Parse ontology schema
	var schema map[string]interface{}
	if err := json.Unmarshal(schemaJSON, &schema); err != nil {
		return nil, fmt.Errorf("failed to parse ontology schema: %w", err)
	}

	// Get element_entity_mappings from schema
	mappings, ok := schema["element_entity_mappings"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("schema missing element_entity_mappings")
	}

	log.Printf("  Entity mappings to process: %d", len(mappings))

	// Create query builder
	builder := NewExtractionQueryBuilder(s.basePath)

	// Extract entities for each mapping
	var allEntities []map[string]interface{}

	for idx, mappingInterface := range mappings {
		mappingMap := mappingInterface.(map[string]interface{})
		entityType := mappingMap["entity_type"].(string)

		log.Printf("  [%d/%d] Processing entity type: %s", idx+1, len(mappings), entityType)

		// Convert mapping to JSON
		mappingJSON, err := json.Marshal(mappingMap)
		if err != nil {
			log.Printf("    ERROR: Failed to marshal mapping: %v", err)
			continue
		}

		// Build SQL query for this mapping
		query, err := builder.BuildEntityExtractionQuery(mappingJSON, filters, conceptEmbeddings)
		if err != nil {
			log.Printf("    ERROR: Failed to build query: %v", err)
			continue
		}

		// Log the SQL query for debugging
		if idx == 0 {
			log.Printf("    DEBUG: Generated SQL query:\n%s\n", query.SQL)
		}

		// Execute query in DuckDB
		entities, err := s.executeEntityExtractionQuery(query)
		if err != nil {
			log.Printf("    ERROR: Failed to execute query: %v", err)
			continue
		}

		log.Printf("    ✓ Extracted %d entities", len(entities))
		allEntities = append(allEntities, entities...)
	}

	log.Printf("========================================")
	log.Printf("  TOTAL ENTITIES EXTRACTED: %d", len(allEntities))
	log.Printf("========================================")

	// Convert to JSON
	result, err := json.Marshal(allEntities)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal results: %w", err)
	}

	return result, nil
}

// executeEntityExtractionQuery executes a single entity extraction SQL query
func (s *HiveParquetStorage) executeEntityExtractionQuery(query *EntityExtractionQuery) ([]map[string]interface{}, error) {
	// Open DuckDB connection
	db, err := sql.Open("duckdb", "")
	if err != nil {
		return nil, fmt.Errorf("failed to open DuckDB connection: %w", err)
	}
	defer db.Close()

	// Execute query
	rows, err := db.Query(query.SQL)
	if err != nil {
		return nil, fmt.Errorf("failed to execute extraction query: %w\nSQL: %s", err, query.SQL)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	// Parse results
	var entities []map[string]interface{}
	for rows.Next() {
		// Create slice of interface{} to hold column values
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		// Scan row
		if err := rows.Scan(valuePtrs...); err != nil {
			log.Printf("WARNING: Failed to scan row: %v", err)
			continue
		}

		// Build entity map
		entity := make(map[string]interface{})
		for i, col := range columns {
			entity[col] = values[i]
		}

		entities = append(entities, entity)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return entities, nil
}

// ExtractRelationshipsSQL executes relationship extraction within DuckDB
// TODO: Implement relationship extraction (similar pattern to entity extraction)
func (s *HiveParquetStorage) ExtractRelationshipsSQL(
	schemaJSON []byte,
	entitiesJSON []byte,
	filters map[string]interface{},
) ([]byte, error) {
	// Placeholder - will implement after entity extraction is validated
	log.Printf("ExtractRelationshipsSQL: Not yet implemented")
	return []byte("[]"), nil
}

// Close closes the storage (no-op for Parquet)
func (s *HiveParquetStorage) Close() error {
	log.Println("Closing Hive-partitioned Parquet storage")
	return nil
}

// partitionDocuments groups documents by partition keys (date, source)
func (s *HiveParquetStorage) partitionDocuments(documents []Document) map[string][]Document {
	partitioned := make(map[string][]Document)
	for _, doc := range documents {
		key := s.getPartitionKey(doc.ProcessedAt, doc.SourceName)
		partitioned[key] = append(partitioned[key], doc)
	}
	return partitioned
}

// partitionElements groups elements by partition keys (date, source)
func (s *HiveParquetStorage) partitionElements(elements []Element) map[string][]Element {
	partitioned := make(map[string][]Element)
	for _, elem := range elements {
		key := s.getPartitionKey(time.Now(), elem.SourceName)
		partitioned[key] = append(partitioned[key], elem)
	}
	return partitioned
}

// partitionRelationships groups relationships by partition keys
func (s *HiveParquetStorage) partitionRelationships(relationships []Relationship) map[string][]Relationship {
	partitioned := make(map[string][]Relationship)
	for _, rel := range relationships {
		key := s.getPartitionKey(time.Now(), rel.SourceName)
		partitioned[key] = append(partitioned[key], rel)
	}
	return partitioned
}

// partitionEmbeddings groups embeddings by partition keys
func (s *HiveParquetStorage) partitionEmbeddings(embeddings []Embedding) map[string][]Embedding {
	partitioned := make(map[string][]Embedding)
	for _, emb := range embeddings {
		key := s.getPartitionKey(time.Now(), emb.SourceName)
		partitioned[key] = append(partitioned[key], emb)
	}
	return partitioned
}

// partitionLinks groups links by partition keys
func (s *HiveParquetStorage) partitionLinks(links []Link) map[string][]Link {
	partitioned := make(map[string][]Link)
	for _, link := range links {
		key := s.getPartitionKey(time.Now(), link.SourceName)
		partitioned[key] = append(partitioned[key], link)
	}
	return partitioned
}

// partitionOntologyEntities groups ontology entities by partition keys (source, domain, run_id)
func (s *HiveParquetStorage) partitionOntologyEntities(entities []OntologyEntity) map[string][]OntologyEntity {
	partitioned := make(map[string][]OntologyEntity)
	for _, entity := range entities {
		key := s.getOntologyPartitionKey(entity.SourceName, entity.Domain, entity.RunID)
		partitioned[key] = append(partitioned[key], entity)
	}
	return partitioned
}

// partitionOntologyRelationships groups ontology relationships by partition keys (source, domain, run_id)
func (s *HiveParquetStorage) partitionOntologyRelationships(relationships []OntologyRelationship) map[string][]OntologyRelationship {
	partitioned := make(map[string][]OntologyRelationship)
	for _, rel := range relationships {
		key := s.getOntologyPartitionKey(rel.SourceName, rel.Domain, rel.RunID)
		partitioned[key] = append(partitioned[key], rel)
	}
	return partitioned
}

// partitionOntologyMentions groups ontology mentions by partition keys
func (s *HiveParquetStorage) partitionOntologyMentions(mentions []OntologyMention) map[string][]OntologyMention {
	partitioned := make(map[string][]OntologyMention)
	for _, mention := range mentions {
		// Mentions inherit partition from their entity's source/domain
		key := s.getPartitionKey(time.Now(), mention.SourceName)
		partitioned[key] = append(partitioned[key], mention)
	}
	return partitioned
}

// getPartitionKey generates a partition key based on date and source (for non-elements tables)
func (s *HiveParquetStorage) getPartitionKey(timestamp time.Time, sourceName string) string {
	date := timestamp.Format("2006-01-02")
	return fmt.Sprintf("date=%s/source=%s", date, sourceName)
}

// getOntologyPartitionKey generates partition key for ontology data (source, domain, run_id)
func (s *HiveParquetStorage) getOntologyPartitionKey(sourceName, domain, runID string) string {
	return fmt.Sprintf("source=%s/domain=%s/run_id=%s", sourceName, domain, runID)
}

// getHivePartitionPath returns the full Hive partition path for elements
// Format: basePath/elements/element_type=X/version=Y/date=Z/source=W/
func (s *HiveParquetStorage) getHivePartitionPath(elemType string, date, source string) string {
	return filepath.Join(
		s.basePath,
		"elements",
		fmt.Sprintf("element_type=%s", elemType),
		fmt.Sprintf("version=%s", s.version),
		fmt.Sprintf("date=%s", date),
		fmt.Sprintf("source=%s", source),
	)
}

// getPartitionPath returns the full partition path for non-elements tables
func (s *HiveParquetStorage) getPartitionPath(tableName, partKey string) string {
	return filepath.Join(s.basePath, tableName, partKey)
}

// writeElementsToHivePartition writes elements to a Hive-partitioned Parquet file
func (s *HiveParquetStorage) writeElementsToHivePartition(elemType, partKey string, elements []Element) error {
	// Parse partition key to extract date and source
	parts := strings.Split(partKey, "/")
	date := strings.TrimPrefix(parts[0], "date=")
	source := strings.TrimPrefix(parts[1], "source=")

	// Build Hive partition path
	partPath := s.getHivePartitionPath(elemType, date, source)
	if err := os.MkdirAll(partPath, 0755); err != nil {
		return fmt.Errorf("failed to create partition directory: %w", err)
	}

	// Generate unique filename
	filename := fmt.Sprintf("elements_%s.parquet", generateRandomHex(8))
	filepath := filepath.Join(partPath, filename)

	// Get type-specific schema from registry (universal schema with 20 fields)
	schema := s.schemaRegistry.GetSchema(elemType)

	// Write elements using the universal schema
	return s.writeElementsWithSchema(filepath, schema, elements)
}

// writeElementsWithSchema writes elements to Parquet using the provided Arrow schema
func (s *HiveParquetStorage) writeElementsWithSchema(filepath string, schema *arrow.Schema, elements []Element) error {
	// Create builders for all 20 fields in the universal schema
	elementIDBuilder := array.NewStringBuilder(s.allocator)
	docIDBuilder := array.NewStringBuilder(s.allocator)
	sourceNameBuilder := array.NewStringBuilder(s.allocator)
	elementTypeBuilder := array.NewStringBuilder(s.allocator)
	elementCategoryBuilder := array.NewStringBuilder(s.allocator)
	contentBuilder := array.NewStringBuilder(s.allocator)
	contentPreviewBuilder := array.NewStringBuilder(s.allocator)
	contentHashBuilder := array.NewStringBuilder(s.allocator)
	parentIDBuilder := array.NewStringBuilder(s.allocator)
	elementOrderBuilder := array.NewFloat64Builder(s.allocator)
	documentPositionBuilder := array.NewFloat64Builder(s.allocator)

	// UDML Phase 1: 6 promoted fields (nullable)
	pageNumberBuilder := array.NewInt64Builder(s.allocator)
	sectionLevelBuilder := array.NewInt64Builder(s.allocator)
	rowIndexBuilder := array.NewInt64Builder(s.allocator)
	columnIndexBuilder := array.NewInt64Builder(s.allocator)
	temporalTypeBuilder := array.NewStringBuilder(s.allocator)
	tagNameBuilder := array.NewStringBuilder(s.allocator)

	// JSON overflow fields
	contentLocationBuilder := array.NewStringBuilder(s.allocator)
	metadataBuilder := array.NewStringBuilder(s.allocator)
	temporalMetadataBuilder := array.NewStringBuilder(s.allocator)

	defer elementIDBuilder.Release()
	defer docIDBuilder.Release()
	defer sourceNameBuilder.Release()
	defer elementTypeBuilder.Release()
	defer elementCategoryBuilder.Release()
	defer contentBuilder.Release()
	defer contentPreviewBuilder.Release()
	defer contentHashBuilder.Release()
	defer parentIDBuilder.Release()
	defer elementOrderBuilder.Release()
	defer documentPositionBuilder.Release()
	defer pageNumberBuilder.Release()
	defer sectionLevelBuilder.Release()
	defer rowIndexBuilder.Release()
	defer columnIndexBuilder.Release()
	defer temporalTypeBuilder.Release()
	defer tagNameBuilder.Release()
	defer contentLocationBuilder.Release()
	defer metadataBuilder.Release()
	defer temporalMetadataBuilder.Release()

	// Append data
	for _, elem := range elements {
		// Core fields (required)
		elementIDBuilder.Append(elem.ElementID)
		docIDBuilder.Append(elem.DocID)
		sourceNameBuilder.Append(elem.SourceName)
		elementTypeBuilder.Append(elem.ElementType)
		elementCategoryBuilder.Append(elem.ElementCategory)

		// Nullable core fields
		if elem.Content != "" {
			contentBuilder.Append(elem.Content)
		} else {
			contentBuilder.AppendNull()
		}

		contentPreviewBuilder.Append(elem.ContentPreview)

		if elem.ContentHash != "" {
			contentHashBuilder.Append(elem.ContentHash)
		} else {
			contentHashBuilder.AppendNull()
		}

		if elem.ParentID != "" {
			parentIDBuilder.Append(elem.ParentID)
		} else {
			parentIDBuilder.AppendNull()
		}

		elementOrderBuilder.Append(elem.ElementOrder)
		documentPositionBuilder.Append(elem.DocumentPosition)

		// UDML Phase 1: 6 promoted fields (all nullable)
		if elem.PageNumber != nil {
			pageNumberBuilder.Append(int64(*elem.PageNumber))
		} else {
			pageNumberBuilder.AppendNull()
		}

		if elem.SectionLevel != nil {
			sectionLevelBuilder.Append(int64(*elem.SectionLevel))
		} else {
			sectionLevelBuilder.AppendNull()
		}

		if elem.RowIndex != nil {
			rowIndexBuilder.Append(int64(*elem.RowIndex))
		} else {
			rowIndexBuilder.AppendNull()
		}

		if elem.ColumnIndex != nil {
			columnIndexBuilder.Append(int64(*elem.ColumnIndex))
		} else {
			columnIndexBuilder.AppendNull()
		}

		if elem.TemporalType != nil {
			temporalTypeBuilder.Append(*elem.TemporalType)
		} else {
			temporalTypeBuilder.AppendNull()
		}

		if elem.TagName != nil {
			tagNameBuilder.Append(*elem.TagName)
		} else {
			tagNameBuilder.AppendNull()
		}

		// JSON overflow fields (serialize as JSON strings)
		if elem.ContentLocation != nil {
			contentLocationJSON, err := json.Marshal(elem.ContentLocation)
			if err != nil {
				log.Printf("WARNING: Failed to marshal content_location for element %s: %v", elem.ElementID, err)
				contentLocationBuilder.AppendNull()
			} else {
				contentLocationBuilder.Append(string(contentLocationJSON))
			}
		} else {
			contentLocationBuilder.AppendNull()
		}

		if elem.Metadata != nil {
			metadataJSON, err := json.Marshal(elem.Metadata)
			if err != nil {
				log.Printf("WARNING: Failed to marshal metadata for element %s: %v", elem.ElementID, err)
				metadataBuilder.AppendNull()
			} else {
				metadataBuilder.Append(string(metadataJSON))
			}
		} else {
			metadataBuilder.AppendNull()
		}

		if elem.TemporalMetadata != nil {
			temporalMetadataJSON, err := json.Marshal(elem.TemporalMetadata)
			if err != nil {
				log.Printf("WARNING: Failed to marshal temporal_metadata for element %s: %v", elem.ElementID, err)
				temporalMetadataBuilder.AppendNull()
			} else {
				temporalMetadataBuilder.Append(string(temporalMetadataJSON))
			}
		} else {
			temporalMetadataBuilder.AppendNull()
		}
	}

	// Build record with all 20 columns in schema order
	columns := []arrow.Array{
		elementIDBuilder.NewArray(),
		docIDBuilder.NewArray(),
		sourceNameBuilder.NewArray(),
		elementTypeBuilder.NewArray(),
		elementCategoryBuilder.NewArray(),
		contentBuilder.NewArray(),
		contentPreviewBuilder.NewArray(),
		contentHashBuilder.NewArray(),
		parentIDBuilder.NewArray(),
		elementOrderBuilder.NewArray(),
		documentPositionBuilder.NewArray(),
		// 6 promoted fields
		pageNumberBuilder.NewArray(),
		sectionLevelBuilder.NewArray(),
		rowIndexBuilder.NewArray(),
		columnIndexBuilder.NewArray(),
		temporalTypeBuilder.NewArray(),
		tagNameBuilder.NewArray(),
		// JSON overflow
		contentLocationBuilder.NewArray(),
		metadataBuilder.NewArray(),
		temporalMetadataBuilder.NewArray(),
	}
	defer func() {
		for _, col := range columns {
			col.Release()
		}
	}()

	record := array.NewRecord(schema, columns, int64(len(elements)))
	defer record.Release()

	return s.writeRecordToFile(filepath, schema, record)
}

// writeDocumentsToParquet writes documents to a Parquet file
func (s *HiveParquetStorage) writeDocumentsToParquet(partKey string, documents []Document) error {
	// Create partition directory
	partPath := s.getPartitionPath("documents", partKey)
	if err := os.MkdirAll(partPath, 0755); err != nil {
		return fmt.Errorf("failed to create partition directory: %w", err)
	}

	// Generate unique filename
	filename := fmt.Sprintf("documents_%s.parquet", generateRandomHex(8))
	filepath := filepath.Join(partPath, filename)

	// Define schema
	fields := []arrow.Field{
		{Name: "doc_id", Type: arrow.BinaryTypes.String, Nullable: false},
		{Name: "source_name", Type: arrow.BinaryTypes.String, Nullable: false},
		{Name: "title", Type: arrow.BinaryTypes.String, Nullable: true},
		{Name: "url", Type: arrow.BinaryTypes.String, Nullable: true},
		{Name: "content_type", Type: arrow.BinaryTypes.String, Nullable: true},
		{Name: "processed_at", Type: &arrow.TimestampType{Unit: arrow.Microsecond, TimeZone: "UTC"}, Nullable: false},
		{Name: "element_count", Type: arrow.PrimitiveTypes.Int64, Nullable: false},
		{Name: "relationship_count", Type: arrow.PrimitiveTypes.Int64, Nullable: false},
	}
	schema := arrow.NewSchema(fields, nil)

	// Create builders
	docIDBuilder := array.NewStringBuilder(s.allocator)
	sourceNameBuilder := array.NewStringBuilder(s.allocator)
	titleBuilder := array.NewStringBuilder(s.allocator)
	urlBuilder := array.NewStringBuilder(s.allocator)
	contentTypeBuilder := array.NewStringBuilder(s.allocator)
	processedAtBuilder := array.NewTimestampBuilder(s.allocator, &arrow.TimestampType{Unit: arrow.Microsecond, TimeZone: "UTC"})
	elementCountBuilder := array.NewInt64Builder(s.allocator)
	relationshipCountBuilder := array.NewInt64Builder(s.allocator)

	defer docIDBuilder.Release()
	defer sourceNameBuilder.Release()
	defer titleBuilder.Release()
	defer urlBuilder.Release()
	defer contentTypeBuilder.Release()
	defer processedAtBuilder.Release()
	defer elementCountBuilder.Release()
	defer relationshipCountBuilder.Release()

	// Append data
	for _, doc := range documents {
		docIDBuilder.Append(doc.DocID)
		sourceNameBuilder.Append(doc.SourceName)
		titleBuilder.Append(doc.Title)
		urlBuilder.Append(doc.URL)
		contentTypeBuilder.Append(doc.ContentType)
		processedAtBuilder.Append(arrow.Timestamp(doc.ProcessedAt.UnixMicro()))
		elementCountBuilder.Append(int64(doc.ElementCount))
		relationshipCountBuilder.Append(int64(doc.RelationshipCount))
	}

	// Build record
	columns := []arrow.Array{
		docIDBuilder.NewArray(),
		sourceNameBuilder.NewArray(),
		titleBuilder.NewArray(),
		urlBuilder.NewArray(),
		contentTypeBuilder.NewArray(),
		processedAtBuilder.NewArray(),
		elementCountBuilder.NewArray(),
		relationshipCountBuilder.NewArray(),
	}
	defer func() {
		for _, col := range columns {
			col.Release()
		}
	}()

	record := array.NewRecord(schema, columns, int64(len(documents)))
	defer record.Release()

	return s.writeRecordToFile(filepath, schema, record)
}

// writeRelationshipsToParquet writes relationships to a Parquet file
func (s *HiveParquetStorage) writeRelationshipsToParquet(partKey string, relationships []Relationship) error {
	// Create partition directory
	partPath := s.getPartitionPath("relationships", partKey)
	if err := os.MkdirAll(partPath, 0755); err != nil {
		return fmt.Errorf("failed to create partition directory: %w", err)
	}

	// Generate unique filename
	filename := fmt.Sprintf("relationships_%s.parquet", generateRandomHex(8))
	filepath := filepath.Join(partPath, filename)

	// Define schema
	fields := []arrow.Field{
		{Name: "source_element_id", Type: arrow.BinaryTypes.String, Nullable: false},
		{Name: "target_element_id", Type: arrow.BinaryTypes.String, Nullable: false},
		{Name: "relationship_type", Type: arrow.BinaryTypes.String, Nullable: false},
		{Name: "doc_id", Type: arrow.BinaryTypes.String, Nullable: false},
		{Name: "source_name", Type: arrow.BinaryTypes.String, Nullable: false},
	}
	schema := arrow.NewSchema(fields, nil)

	// Create builders
	sourceElemBuilder := array.NewStringBuilder(s.allocator)
	targetElemBuilder := array.NewStringBuilder(s.allocator)
	relTypeBuilder := array.NewStringBuilder(s.allocator)
	docIDBuilder := array.NewStringBuilder(s.allocator)
	sourceNameBuilder := array.NewStringBuilder(s.allocator)

	defer sourceElemBuilder.Release()
	defer targetElemBuilder.Release()
	defer relTypeBuilder.Release()
	defer docIDBuilder.Release()
	defer sourceNameBuilder.Release()

	// Append data
	for _, rel := range relationships {
		sourceElemBuilder.Append(rel.SourceElementID)
		targetElemBuilder.Append(rel.TargetElementID)
		relTypeBuilder.Append(rel.RelationshipType)
		docIDBuilder.Append(rel.DocID)
		sourceNameBuilder.Append(rel.SourceName)
	}

	// Build record
	columns := []arrow.Array{
		sourceElemBuilder.NewArray(),
		targetElemBuilder.NewArray(),
		relTypeBuilder.NewArray(),
		docIDBuilder.NewArray(),
		sourceNameBuilder.NewArray(),
	}
	defer func() {
		for _, col := range columns {
			col.Release()
		}
	}()

	record := array.NewRecord(schema, columns, int64(len(relationships)))
	defer record.Release()

	return s.writeRecordToFile(filepath, schema, record)
}

// writeEmbeddingsToParquet writes embeddings to a Parquet file
func (s *HiveParquetStorage) writeEmbeddingsToParquet(partKey string, embeddings []Embedding) error {
	// Create partition directory
	partPath := s.getPartitionPath("embeddings", partKey)
	if err := os.MkdirAll(partPath, 0755); err != nil {
		return fmt.Errorf("failed to create partition directory: %w", err)
	}

	// Generate unique filename
	filename := fmt.Sprintf("embeddings_%s.parquet", generateRandomHex(8))
	filepath := filepath.Join(partPath, filename)

	// Define schema
	fields := []arrow.Field{
		{Name: "element_id", Type: arrow.BinaryTypes.String, Nullable: false},
		{Name: "doc_id", Type: arrow.BinaryTypes.String, Nullable: false},
		{Name: "source_name", Type: arrow.BinaryTypes.String, Nullable: false},
		{Name: "text", Type: arrow.BinaryTypes.String, Nullable: true},
		{Name: "embedding", Type: arrow.ListOf(arrow.PrimitiveTypes.Float64), Nullable: true},
	}
	schema := arrow.NewSchema(fields, nil)

	// Create builders
	elementIDBuilder := array.NewStringBuilder(s.allocator)
	docIDBuilder := array.NewStringBuilder(s.allocator)
	sourceNameBuilder := array.NewStringBuilder(s.allocator)
	textBuilder := array.NewStringBuilder(s.allocator)
	embeddingBuilder := array.NewListBuilder(s.allocator, arrow.PrimitiveTypes.Float64)
	valueBuilder := embeddingBuilder.ValueBuilder().(*array.Float64Builder)

	defer elementIDBuilder.Release()
	defer docIDBuilder.Release()
	defer sourceNameBuilder.Release()
	defer textBuilder.Release()
	defer embeddingBuilder.Release()

	// Append data
	for _, emb := range embeddings {
		elementIDBuilder.Append(emb.ElementID)
		docIDBuilder.Append(emb.DocID)
		sourceNameBuilder.Append(emb.SourceName)
		textBuilder.Append(sanitizeUTF8(emb.Text))

		// Append embedding vector
		embeddingBuilder.Append(true)
		for _, val := range emb.Embedding {
			valueBuilder.Append(val)
		}
	}

	// Build record
	columns := []arrow.Array{
		elementIDBuilder.NewArray(),
		docIDBuilder.NewArray(),
		sourceNameBuilder.NewArray(),
		textBuilder.NewArray(),
		embeddingBuilder.NewArray(),
	}
	defer func() {
		for _, col := range columns {
			col.Release()
		}
	}()

	record := array.NewRecord(schema, columns, int64(len(embeddings)))
	defer record.Release()

	return s.writeRecordToFile(filepath, schema, record)
}

// writeLinksToParquet writes links to a Parquet file
func (s *HiveParquetStorage) writeLinksToParquet(partKey string, links []Link) error {
	// Create partition directory
	partPath := s.getPartitionPath("links", partKey)
	if err := os.MkdirAll(partPath, 0755); err != nil {
		return fmt.Errorf("failed to create partition directory: %w", err)
	}

	// Generate unique filename
	filename := fmt.Sprintf("links_%s.parquet", generateRandomHex(8))
	filepath := filepath.Join(partPath, filename)

	// Define schema
	fields := []arrow.Field{
		{Name: "link_id", Type: arrow.BinaryTypes.String, Nullable: false},
		{Name: "source_element_id", Type: arrow.BinaryTypes.String, Nullable: false},
		{Name: "doc_id", Type: arrow.BinaryTypes.String, Nullable: false},
		{Name: "source_name", Type: arrow.BinaryTypes.String, Nullable: false},
		{Name: "link_type", Type: arrow.BinaryTypes.String, Nullable: false},
		{Name: "link_target", Type: arrow.BinaryTypes.String, Nullable: false},
		{Name: "link_text", Type: arrow.BinaryTypes.String, Nullable: true},
	}
	schema := arrow.NewSchema(fields, nil)

	// Create builders
	linkIDBuilder := array.NewStringBuilder(s.allocator)
	sourceElementIDBuilder := array.NewStringBuilder(s.allocator)
	docIDBuilder := array.NewStringBuilder(s.allocator)
	sourceNameBuilder := array.NewStringBuilder(s.allocator)
	linkTypeBuilder := array.NewStringBuilder(s.allocator)
	linkTargetBuilder := array.NewStringBuilder(s.allocator)
	linkTextBuilder := array.NewStringBuilder(s.allocator)

	defer linkIDBuilder.Release()
	defer sourceElementIDBuilder.Release()
	defer docIDBuilder.Release()
	defer sourceNameBuilder.Release()
	defer linkTypeBuilder.Release()
	defer linkTargetBuilder.Release()
	defer linkTextBuilder.Release()

	// Append data
	for _, link := range links {
		linkIDBuilder.Append(link.LinkID)
		sourceElementIDBuilder.Append(link.SourceElementID)
		docIDBuilder.Append(link.DocID)
		sourceNameBuilder.Append(link.SourceName)
		linkTypeBuilder.Append(link.LinkType)
		linkTargetBuilder.Append(link.LinkTarget)
		linkTextBuilder.Append(link.LinkText)
	}

	// Build record
	columns := []arrow.Array{
		linkIDBuilder.NewArray(),
		sourceElementIDBuilder.NewArray(),
		docIDBuilder.NewArray(),
		sourceNameBuilder.NewArray(),
		linkTypeBuilder.NewArray(),
		linkTargetBuilder.NewArray(),
		linkTextBuilder.NewArray(),
	}
	defer func() {
		for _, col := range columns {
			col.Release()
		}
	}()

	record := array.NewRecord(schema, columns, int64(len(links)))
	defer record.Release()

	return s.writeRecordToFile(filepath, schema, record)
}

// generateRandomHex generates a random hex string of specified length
func generateHiveRandomHex(length int) string {
	bytes := make([]byte, length/2)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based if random fails
		return fmt.Sprintf("%08x", time.Now().UnixNano()&0xFFFFFFFF)[:length]
	}
	return hex.EncodeToString(bytes)[:length]
}

// sanitizeHiveUTF8 removes invalid UTF-8 sequences from a string
func sanitizeHiveUTF8(s string) string {
	if utf8.ValidString(s) {
		return s
	}
	// Replace invalid UTF-8 sequences with replacement character
	return strings.ToValidUTF8(s, "�")
}

// writeRecordToFile writes an Arrow record to a Parquet file
func (s *HiveParquetStorage) writeRecordToFile(filepath string, schema *arrow.Schema, record arrow.Record) error {
	// Create output file
	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Create Parquet writer with compression
	props := parquet.NewWriterProperties(
		parquet.WithCompression(compress.Codecs.Snappy),
	)

	writer, err := pqarrow.NewFileWriter(schema, file, props, pqarrow.DefaultWriterProps())
	if err != nil {
		return fmt.Errorf("failed to create parquet writer: %w", err)
	}
	defer writer.Close()

	// Write record
	if err := writer.Write(record); err != nil {
		return fmt.Errorf("failed to write record: %w", err)
	}

	return nil
}
