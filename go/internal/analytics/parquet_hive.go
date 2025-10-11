package analytics

import (
	"crypto/rand"
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
	"github.com/kennethstott/go-doc-go/internal/udml"
)

// HiveParquetStorage implements Storage interface with Hive-partitioned structure
// Partition scheme: element_type=X/version=Y/date=Z/source=W/
// This enables query engines to skip irrelevant partitions for 60-1000x faster queries
type HiveParquetStorage struct {
	basePath       string
	version        string // UDML schema version (e.g., "v2.0.0")
	schemaRegistry *udml.SchemaRegistry
	mu             sync.Mutex
	allocator      memory.Allocator
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

	log.Printf("========================================")
	log.Printf("ANALYTICS: Initialized Hive-partitioned Parquet storage")
	log.Printf("  Path: %s", basePath)
	log.Printf("  Version: %s", version)
	log.Printf("  Partitioning: element_type -> version -> date -> source")
	log.Printf("  Schema: UDML Phase 1 (20 fields with 6 promoted query-optimized fields)")
	log.Printf("========================================")

	return &HiveParquetStorage{
		basePath:       basePath,
		version:        version,
		schemaRegistry: udml.NewSchemaRegistry(),
		allocator:      memory.NewGoAllocator(),
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

// getPartitionKey generates a partition key based on date and source (for non-elements tables)
func (s *HiveParquetStorage) getPartitionKey(timestamp time.Time, sourceName string) string {
	date := timestamp.Format("2006-01-02")
	return fmt.Sprintf("date=%s/source=%s", date, sourceName)
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
