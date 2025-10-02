package analytics

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/apache/arrow/go/v18/arrow"
	"github.com/apache/arrow/go/v18/arrow/array"
	"github.com/apache/arrow/go/v18/arrow/memory"
	"github.com/apache/arrow/go/v18/parquet"
	"github.com/apache/arrow/go/v18/parquet/compress"
	"github.com/apache/arrow/go/v18/parquet/pqarrow"
)

// ParquetStorage implements Storage interface with native Go Parquet writing
type ParquetStorage struct {
	basePath     string
	partitioning []string // e.g., ["date", "source"]
	mu           sync.Mutex
	allocator    memory.Allocator
}

// NewParquetStorage creates a new Parquet storage backend
func NewParquetStorage(config map[string]interface{}) (*ParquetStorage, error) {
	// Extract configuration
	basePath, ok := config["path"].(string)
	if !ok || basePath == "" {
		return nil, fmt.Errorf("missing required 'path' in config")
	}

	// Extract partitioning configuration (default: date, source)
	var partitioning []string
	if partConfig, ok := config["partitioning"].([]interface{}); ok {
		for _, p := range partConfig {
			if pStr, ok := p.(string); ok {
				partitioning = append(partitioning, pStr)
			}
		}
	}
	if len(partitioning) == 0 {
		partitioning = []string{"date", "source"}
	}

	// Create base directory if it doesn't exist
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base path: %w", err)
	}

	log.Printf("========================================")
	log.Printf("ANALYTICS: Initialized native Go Parquet storage")
	log.Printf("  Path: %s", basePath)
	log.Printf("  Partitioning: %v", partitioning)
	log.Printf("  Storage type: Native Go (NO Python shims)")
	log.Printf("========================================")

	return &ParquetStorage{
		basePath:     basePath,
		partitioning: partitioning,
		allocator:    memory.NewGoAllocator(),
	}, nil
}

// AppendDocuments writes documents to Parquet files
func (s *ParquetStorage) AppendDocuments(documents []Document) error {
	if len(documents) == 0 {
		return nil
	}

	// Group documents by partition keys (no lock needed - read-only operation)
	partitioned := s.partitionDocuments(documents)

	for partKey, docs := range partitioned {
		if err := s.writeDocumentsToParquet(partKey, docs); err != nil {
			return fmt.Errorf("failed to write documents partition %s: %w", partKey, err)
		}
	}

	log.Printf("ANALYTICS: Wrote %d documents to Parquet (native Go)", len(documents))
	return nil
}

// AppendElements writes elements to Parquet files
func (s *ParquetStorage) AppendElements(elements []Element) error {
	if len(elements) == 0 {
		return nil
	}

	// Group elements by partition keys (no lock needed - read-only operation)
	partitioned := s.partitionElements(elements)

	for partKey, elems := range partitioned {
		if err := s.writeElementsToParquet(partKey, elems); err != nil {
			return fmt.Errorf("failed to write elements partition %s: %w", partKey, err)
		}
	}

	log.Printf("ANALYTICS: Wrote %d elements to Parquet (native Go)", len(elements))
	return nil
}

// AppendRelationships writes relationships to Parquet files
func (s *ParquetStorage) AppendRelationships(relationships []Relationship) error {
	if len(relationships) == 0 {
		return nil
	}

	// Group relationships by partition keys (no lock needed - read-only operation)
	partitioned := s.partitionRelationships(relationships)

	for partKey, rels := range partitioned {
		if err := s.writeRelationshipsToParquet(partKey, rels); err != nil {
			return fmt.Errorf("failed to write relationships partition %s: %w", partKey, err)
		}
	}

	log.Printf("ANALYTICS: Wrote %d relationships to Parquet (native Go)", len(relationships))
	return nil
}

// AppendEmbeddings writes embeddings to Parquet files
func (s *ParquetStorage) AppendEmbeddings(embeddings []Embedding) error {
	if len(embeddings) == 0 {
		return nil
	}

	// Group embeddings by partition keys (no lock needed - read-only operation)
	partitioned := s.partitionEmbeddings(embeddings)

	for partKey, embs := range partitioned {
		if err := s.writeEmbeddingsToParquet(partKey, embs); err != nil {
			return fmt.Errorf("failed to write embeddings partition %s: %w", partKey, err)
		}
	}

	log.Printf("ANALYTICS: Wrote %d embeddings to Parquet (native Go)", len(embeddings))
	return nil
}

// AppendLinks writes links to Parquet files
func (s *ParquetStorage) AppendLinks(links []Link) error {
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

	log.Printf("ANALYTICS: Wrote %d links to Parquet (native Go)", len(links))
	return nil
}

// Close closes the storage (no-op for Parquet)
func (s *ParquetStorage) Close() error {
	log.Println("Closing Parquet storage")
	return nil
}

// partitionDocuments groups documents by partition keys
func (s *ParquetStorage) partitionDocuments(documents []Document) map[string][]Document {
	partitioned := make(map[string][]Document)
	for _, doc := range documents {
		key := s.getPartitionKey(doc.ProcessedAt, doc.SourceName)
		partitioned[key] = append(partitioned[key], doc)
	}
	return partitioned
}

// partitionElements groups elements by partition keys
func (s *ParquetStorage) partitionElements(elements []Element) map[string][]Element {
	partitioned := make(map[string][]Element)
	for _, elem := range elements {
		// Use current time for partitioning (could be enhanced to use document processed time)
		key := s.getPartitionKey(time.Now(), elem.SourceName)
		partitioned[key] = append(partitioned[key], elem)
	}
	return partitioned
}

// partitionRelationships groups relationships by partition keys
func (s *ParquetStorage) partitionRelationships(relationships []Relationship) map[string][]Relationship {
	partitioned := make(map[string][]Relationship)
	for _, rel := range relationships {
		key := s.getPartitionKey(time.Now(), rel.SourceName)
		partitioned[key] = append(partitioned[key], rel)
	}
	return partitioned
}

// partitionEmbeddings groups embeddings by partition keys
func (s *ParquetStorage) partitionEmbeddings(embeddings []Embedding) map[string][]Embedding {
	partitioned := make(map[string][]Embedding)
	for _, emb := range embeddings {
		key := s.getPartitionKey(time.Now(), emb.SourceName)
		partitioned[key] = append(partitioned[key], emb)
	}
	return partitioned
}

// getPartitionKey generates a partition key based on date and source
func (s *ParquetStorage) getPartitionKey(timestamp time.Time, sourceName string) string {
	date := timestamp.Format("2006-01-02")
	return fmt.Sprintf("date=%s/source=%s", date, sourceName)
}

// getPartitionPath returns the full partition path for a given table and partition key
func (s *ParquetStorage) getPartitionPath(tableName, partKey string) string {
	return filepath.Join(s.basePath, tableName, partKey)
}

// writeDocumentsToParquet writes documents to a Parquet file
func (s *ParquetStorage) writeDocumentsToParquet(partKey string, documents []Document) error {
	// Create partition directory
	partPath := s.getPartitionPath("documents", partKey)
	if err := os.MkdirAll(partPath, 0755); err != nil {
		return fmt.Errorf("failed to create partition directory: %w", err)
	}

	// Generate unique filename matching Python format: {table_name}_{8_char_hex}.parquet
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

	// Create builders for each column
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

	// Write to Parquet file
	return s.writeRecordToFile(filepath, schema, record)
}

// writeElementsToParquet writes elements to a Parquet file
func (s *ParquetStorage) writeElementsToParquet(partKey string, elements []Element) error {
	// Create partition directory
	partPath := s.getPartitionPath("elements", partKey)
	if err := os.MkdirAll(partPath, 0755); err != nil {
		return fmt.Errorf("failed to create partition directory: %w", err)
	}

	// Generate unique filename matching Python format: {table_name}_{8_char_hex}.parquet
	filename := fmt.Sprintf("elements_%s.parquet", generateRandomHex(8))
	filepath := filepath.Join(partPath, filename)

	// Define schema - MUST match Python analytics schema exactly
	fields := []arrow.Field{
		{Name: "element_id", Type: arrow.BinaryTypes.String, Nullable: false},
		{Name: "doc_id", Type: arrow.BinaryTypes.String, Nullable: false},
		{Name: "source_name", Type: arrow.BinaryTypes.String, Nullable: false},
		{Name: "element_type", Type: arrow.BinaryTypes.String, Nullable: false},
		{Name: "element_category", Type: arrow.BinaryTypes.String, Nullable: false},
		{Name: "content", Type: arrow.BinaryTypes.String, Nullable: true},
		{Name: "content_preview", Type: arrow.BinaryTypes.String, Nullable: true},
		{Name: "content_hash", Type: arrow.BinaryTypes.String, Nullable: true},
		{Name: "parent_id", Type: arrow.BinaryTypes.String, Nullable: true},
		{Name: "element_order", Type: arrow.PrimitiveTypes.Float64, Nullable: true},
		{Name: "document_position", Type: arrow.PrimitiveTypes.Float64, Nullable: true},
	}
	schema := arrow.NewSchema(fields, nil)

	// Create builders
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

	// Append data
	for _, elem := range elements {
		elementIDBuilder.Append(elem.ElementID)
		docIDBuilder.Append(elem.DocID)
		sourceNameBuilder.Append(elem.SourceName)
		elementTypeBuilder.Append(elem.ElementType)
		elementCategoryBuilder.Append(elem.ElementCategory)

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
	}

	// Build record
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

// writeRelationshipsToParquet writes relationships to a Parquet file
func (s *ParquetStorage) writeRelationshipsToParquet(partKey string, relationships []Relationship) error {
	// Create partition directory
	partPath := s.getPartitionPath("relationships", partKey)
	if err := os.MkdirAll(partPath, 0755); err != nil {
		return fmt.Errorf("failed to create partition directory: %w", err)
	}

	// Generate unique filename matching Python format: {table_name}_{8_char_hex}.parquet
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
func (s *ParquetStorage) writeEmbeddingsToParquet(partKey string, embeddings []Embedding) error {
	// Create partition directory
	partPath := s.getPartitionPath("embeddings", partKey)
	if err := os.MkdirAll(partPath, 0755); err != nil {
		return fmt.Errorf("failed to create partition directory: %w", err)
	}

	// Generate unique filename matching Python format: {table_name}_{8_char_hex}.parquet
	filename := fmt.Sprintf("embeddings_%s.parquet", generateRandomHex(8))
	filepath := filepath.Join(partPath, filename)

	// Determine embedding dimension from first embedding
	embeddingDim := 0
	if len(embeddings) > 0 && len(embeddings[0].Embedding) > 0 {
		embeddingDim = len(embeddings[0].Embedding)
	}

	// Define schema (embedding as list of floats)
	fields := []arrow.Field{
		{Name: "element_id", Type: arrow.BinaryTypes.String, Nullable: false},
		{Name: "doc_id", Type: arrow.BinaryTypes.String, Nullable: false},
		{Name: "source_name", Type: arrow.BinaryTypes.String, Nullable: false},
		{Name: "text", Type: arrow.BinaryTypes.String, Nullable: true}, // Contextual text used for embedding
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
		textBuilder.Append(emb.Text)

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

	log.Printf("Writing %d embeddings (dimension: %d) to %s", len(embeddings), embeddingDim, filepath)
	return s.writeRecordToFile(filepath, schema, record)
}

// partitionLinks groups links by partition keys
func (s *ParquetStorage) partitionLinks(links []Link) map[string][]Link {
	partitioned := make(map[string][]Link)
	for _, link := range links {
		key := s.getPartitionKey(time.Now(), link.SourceName)
		partitioned[key] = append(partitioned[key], link)
	}
	return partitioned
}

// writeLinksToParquet writes links to a Parquet file
func (s *ParquetStorage) writeLinksToParquet(partKey string, links []Link) error {
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
func generateRandomHex(length int) string {
	bytes := make([]byte, length/2)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based if random fails
		return fmt.Sprintf("%08x", time.Now().UnixNano()&0xFFFFFFFF)[:length]
	}
	return hex.EncodeToString(bytes)[:length]
}

// writeRecordToFile writes an Arrow record to a Parquet file
func (s *ParquetStorage) writeRecordToFile(filepath string, schema *arrow.Schema, record arrow.Record) error {
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
