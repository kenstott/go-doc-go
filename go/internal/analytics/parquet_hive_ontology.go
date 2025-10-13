package analytics

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/apache/arrow/go/v18/arrow"
	"github.com/apache/arrow/go/v18/arrow/array"
)

// writeOntologyEntitiesToParquet writes ontology entities to a Parquet file
func (s *HiveParquetStorage) writeOntologyEntitiesToParquet(partKey string, entities []OntologyEntity) error {
	// Create partition directory
	partPath := s.getPartitionPath("ontology_entities", partKey)
	if err := os.MkdirAll(partPath, 0755); err != nil {
		return fmt.Errorf("failed to create partition directory: %w", err)
	}

	// Generate unique filename
	filename := fmt.Sprintf("ontology_entities_%s.parquet", generateRandomHex(8))
	filepath := filepath.Join(partPath, filename)

	// Define schema
	fields := []arrow.Field{
		{Name: "entity_id", Type: arrow.BinaryTypes.String, Nullable: false},
		{Name: "doc_id", Type: arrow.BinaryTypes.String, Nullable: false},
		{Name: "source_name", Type: arrow.BinaryTypes.String, Nullable: false},
		{Name: "entity_name", Type: arrow.BinaryTypes.String, Nullable: false},
		{Name: "entity_type", Type: arrow.BinaryTypes.String, Nullable: false},
		{Name: "domain", Type: arrow.BinaryTypes.String, Nullable: false},
		{Name: "confidence", Type: arrow.PrimitiveTypes.Float64, Nullable: false},
		{Name: "attributes", Type: arrow.BinaryTypes.String, Nullable: true}, // JSON
		{Name: "element_id", Type: arrow.BinaryTypes.String, Nullable: false},
		{Name: "extracted_at", Type: &arrow.TimestampType{Unit: arrow.Microsecond, TimeZone: "UTC"}, Nullable: false},
	}
	schema := arrow.NewSchema(fields, nil)

	// Create builders
	entityIDBuilder := array.NewStringBuilder(s.allocator)
	docIDBuilder := array.NewStringBuilder(s.allocator)
	sourceNameBuilder := array.NewStringBuilder(s.allocator)
	entityNameBuilder := array.NewStringBuilder(s.allocator)
	entityTypeBuilder := array.NewStringBuilder(s.allocator)
	domainBuilder := array.NewStringBuilder(s.allocator)
	confidenceBuilder := array.NewFloat64Builder(s.allocator)
	attributesBuilder := array.NewStringBuilder(s.allocator)
	elementIDBuilder := array.NewStringBuilder(s.allocator)
	extractedAtBuilder := array.NewTimestampBuilder(s.allocator, &arrow.TimestampType{Unit: arrow.Microsecond, TimeZone: "UTC"})

	defer entityIDBuilder.Release()
	defer docIDBuilder.Release()
	defer sourceNameBuilder.Release()
	defer entityNameBuilder.Release()
	defer entityTypeBuilder.Release()
	defer domainBuilder.Release()
	defer confidenceBuilder.Release()
	defer attributesBuilder.Release()
	defer elementIDBuilder.Release()
	defer extractedAtBuilder.Release()

	// Append data
	for _, entity := range entities {
		entityIDBuilder.Append(entity.EntityID)
		docIDBuilder.Append(entity.DocID)
		sourceNameBuilder.Append(entity.SourceName)
		entityNameBuilder.Append(entity.EntityName)
		entityTypeBuilder.Append(entity.EntityType)
		domainBuilder.Append(entity.Domain)
		confidenceBuilder.Append(entity.Confidence)

		// Serialize attributes as JSON
		if entity.Attributes != nil && len(entity.Attributes) > 0 {
			attributesJSON, err := json.Marshal(entity.Attributes)
			if err != nil {
				log.Printf("WARNING: Failed to marshal attributes for entity %s: %v", entity.EntityID, err)
				attributesBuilder.AppendNull()
			} else {
				attributesBuilder.Append(string(attributesJSON))
			}
		} else {
			attributesBuilder.AppendNull()
		}

		elementIDBuilder.Append(entity.ElementID)
		extractedAtBuilder.Append(arrow.Timestamp(entity.ExtractedAt.UnixMicro()))
	}

	// Build record
	columns := []arrow.Array{
		entityIDBuilder.NewArray(),
		docIDBuilder.NewArray(),
		sourceNameBuilder.NewArray(),
		entityNameBuilder.NewArray(),
		entityTypeBuilder.NewArray(),
		domainBuilder.NewArray(),
		confidenceBuilder.NewArray(),
		attributesBuilder.NewArray(),
		elementIDBuilder.NewArray(),
		extractedAtBuilder.NewArray(),
	}
	defer func() {
		for _, col := range columns {
			col.Release()
		}
	}()

	record := array.NewRecord(schema, columns, int64(len(entities)))
	defer record.Release()

	return s.writeRecordToFile(filepath, schema, record)
}

// writeOntologyRelationshipsToParquet writes ontology relationships to a Parquet file
func (s *HiveParquetStorage) writeOntologyRelationshipsToParquet(partKey string, relationships []OntologyRelationship) error {
	// Create partition directory
	partPath := s.getPartitionPath("ontology_relationships", partKey)
	if err := os.MkdirAll(partPath, 0755); err != nil {
		return fmt.Errorf("failed to create partition directory: %w", err)
	}

	// Generate unique filename
	filename := fmt.Sprintf("ontology_relationships_%s.parquet", generateRandomHex(8))
	filepath := filepath.Join(partPath, filename)

	// Define schema
	fields := []arrow.Field{
		{Name: "relationship_id", Type: arrow.BinaryTypes.String, Nullable: false},
		{Name: "doc_id", Type: arrow.BinaryTypes.String, Nullable: false},
		{Name: "source_name", Type: arrow.BinaryTypes.String, Nullable: false},
		{Name: "source_entity_id", Type: arrow.BinaryTypes.String, Nullable: false},
		{Name: "target_entity_id", Type: arrow.BinaryTypes.String, Nullable: false},
		{Name: "relationship_type", Type: arrow.BinaryTypes.String, Nullable: false},
		{Name: "domain", Type: arrow.BinaryTypes.String, Nullable: false},
		{Name: "confidence", Type: arrow.PrimitiveTypes.Float64, Nullable: false},
		{Name: "evidence", Type: arrow.BinaryTypes.String, Nullable: true},
		{Name: "attributes", Type: arrow.BinaryTypes.String, Nullable: true}, // JSON
		{Name: "element_id", Type: arrow.BinaryTypes.String, Nullable: false},
		{Name: "extracted_at", Type: &arrow.TimestampType{Unit: arrow.Microsecond, TimeZone: "UTC"}, Nullable: false},
	}
	schema := arrow.NewSchema(fields, nil)

	// Create builders
	relationshipIDBuilder := array.NewStringBuilder(s.allocator)
	docIDBuilder := array.NewStringBuilder(s.allocator)
	sourceNameBuilder := array.NewStringBuilder(s.allocator)
	sourceEntityIDBuilder := array.NewStringBuilder(s.allocator)
	targetEntityIDBuilder := array.NewStringBuilder(s.allocator)
	relationshipTypeBuilder := array.NewStringBuilder(s.allocator)
	domainBuilder := array.NewStringBuilder(s.allocator)
	confidenceBuilder := array.NewFloat64Builder(s.allocator)
	evidenceBuilder := array.NewStringBuilder(s.allocator)
	attributesBuilder := array.NewStringBuilder(s.allocator)
	elementIDBuilder := array.NewStringBuilder(s.allocator)
	extractedAtBuilder := array.NewTimestampBuilder(s.allocator, &arrow.TimestampType{Unit: arrow.Microsecond, TimeZone: "UTC"})

	defer relationshipIDBuilder.Release()
	defer docIDBuilder.Release()
	defer sourceNameBuilder.Release()
	defer sourceEntityIDBuilder.Release()
	defer targetEntityIDBuilder.Release()
	defer relationshipTypeBuilder.Release()
	defer domainBuilder.Release()
	defer confidenceBuilder.Release()
	defer evidenceBuilder.Release()
	defer attributesBuilder.Release()
	defer elementIDBuilder.Release()
	defer extractedAtBuilder.Release()

	// Append data
	for _, rel := range relationships {
		relationshipIDBuilder.Append(rel.RelationshipID)
		docIDBuilder.Append(rel.DocID)
		sourceNameBuilder.Append(rel.SourceName)
		sourceEntityIDBuilder.Append(rel.SourceEntityID)
		targetEntityIDBuilder.Append(rel.TargetEntityID)
		relationshipTypeBuilder.Append(rel.RelationshipType)
		domainBuilder.Append(rel.Domain)
		confidenceBuilder.Append(rel.Confidence)

		if rel.Evidence != "" {
			evidenceBuilder.Append(rel.Evidence)
		} else {
			evidenceBuilder.AppendNull()
		}

		// Serialize attributes as JSON
		if rel.Attributes != nil && len(rel.Attributes) > 0 {
			attributesJSON, err := json.Marshal(rel.Attributes)
			if err != nil {
				log.Printf("WARNING: Failed to marshal attributes for relationship %s: %v", rel.RelationshipID, err)
				attributesBuilder.AppendNull()
			} else {
				attributesBuilder.Append(string(attributesJSON))
			}
		} else {
			attributesBuilder.AppendNull()
		}

		elementIDBuilder.Append(rel.ElementID)
		extractedAtBuilder.Append(arrow.Timestamp(rel.ExtractedAt.UnixMicro()))
	}

	// Build record
	columns := []arrow.Array{
		relationshipIDBuilder.NewArray(),
		docIDBuilder.NewArray(),
		sourceNameBuilder.NewArray(),
		sourceEntityIDBuilder.NewArray(),
		targetEntityIDBuilder.NewArray(),
		relationshipTypeBuilder.NewArray(),
		domainBuilder.NewArray(),
		confidenceBuilder.NewArray(),
		evidenceBuilder.NewArray(),
		attributesBuilder.NewArray(),
		elementIDBuilder.NewArray(),
		extractedAtBuilder.NewArray(),
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

// writeOntologyMentionsToParquet writes ontology mentions to a Parquet file
func (s *HiveParquetStorage) writeOntologyMentionsToParquet(partKey string, mentions []OntologyMention) error {
	// Create partition directory
	partPath := s.getPartitionPath("ontology_mentions", partKey)
	if err := os.MkdirAll(partPath, 0755); err != nil {
		return fmt.Errorf("failed to create partition directory: %w", err)
	}

	// Generate unique filename
	filename := fmt.Sprintf("ontology_mentions_%s.parquet", generateRandomHex(8))
	filepath := filepath.Join(partPath, filename)

	// Define schema
	fields := []arrow.Field{
		{Name: "mention_id", Type: arrow.BinaryTypes.String, Nullable: false},
		{Name: "entity_id", Type: arrow.BinaryTypes.String, Nullable: false},
		{Name: "doc_id", Type: arrow.BinaryTypes.String, Nullable: false},
		{Name: "source_name", Type: arrow.BinaryTypes.String, Nullable: false},
		{Name: "element_id", Type: arrow.BinaryTypes.String, Nullable: false},
		{Name: "mention_text", Type: arrow.BinaryTypes.String, Nullable: false},
		{Name: "start_position", Type: arrow.PrimitiveTypes.Int64, Nullable: false},
		{Name: "end_position", Type: arrow.PrimitiveTypes.Int64, Nullable: false},
		{Name: "extracted_at", Type: &arrow.TimestampType{Unit: arrow.Microsecond, TimeZone: "UTC"}, Nullable: false},
	}
	schema := arrow.NewSchema(fields, nil)

	// Create builders
	mentionIDBuilder := array.NewStringBuilder(s.allocator)
	entityIDBuilder := array.NewStringBuilder(s.allocator)
	docIDBuilder := array.NewStringBuilder(s.allocator)
	sourceNameBuilder := array.NewStringBuilder(s.allocator)
	elementIDBuilder := array.NewStringBuilder(s.allocator)
	mentionTextBuilder := array.NewStringBuilder(s.allocator)
	startPositionBuilder := array.NewInt64Builder(s.allocator)
	endPositionBuilder := array.NewInt64Builder(s.allocator)
	extractedAtBuilder := array.NewTimestampBuilder(s.allocator, &arrow.TimestampType{Unit: arrow.Microsecond, TimeZone: "UTC"})

	defer mentionIDBuilder.Release()
	defer entityIDBuilder.Release()
	defer docIDBuilder.Release()
	defer sourceNameBuilder.Release()
	defer elementIDBuilder.Release()
	defer mentionTextBuilder.Release()
	defer startPositionBuilder.Release()
	defer endPositionBuilder.Release()
	defer extractedAtBuilder.Release()

	// Append data
	for _, mention := range mentions {
		mentionIDBuilder.Append(mention.MentionID)
		entityIDBuilder.Append(mention.EntityID)
		docIDBuilder.Append(mention.DocID)
		sourceNameBuilder.Append(mention.SourceName)
		elementIDBuilder.Append(mention.ElementID)
		mentionTextBuilder.Append(sanitizeUTF8(mention.MentionText))
		startPositionBuilder.Append(int64(mention.StartPosition))
		endPositionBuilder.Append(int64(mention.EndPosition))
		extractedAtBuilder.Append(arrow.Timestamp(mention.ExtractedAt.UnixMicro()))
	}

	// Build record
	columns := []arrow.Array{
		mentionIDBuilder.NewArray(),
		entityIDBuilder.NewArray(),
		docIDBuilder.NewArray(),
		sourceNameBuilder.NewArray(),
		elementIDBuilder.NewArray(),
		mentionTextBuilder.NewArray(),
		startPositionBuilder.NewArray(),
		endPositionBuilder.NewArray(),
		extractedAtBuilder.NewArray(),
	}
	defer func() {
		for _, col := range columns {
			col.Release()
		}
	}()

	record := array.NewRecord(schema, columns, int64(len(mentions)))
	defer record.Release()

	return s.writeRecordToFile(filepath, schema, record)
}
