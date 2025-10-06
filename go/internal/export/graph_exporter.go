package export

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/apache/arrow/go/v18/parquet/file"
	"github.com/kennethstott/go-doc-go/internal/analytics"
)

// GraphExporter exports analytics data from Parquet to Neo4j
type GraphExporter struct {
	sourceStorage analytics.Storage // Source storage (Parquet)
	targetStorage analytics.Storage // Target storage (Neo4j)
	sourcePath    string
	batchSize     int
}

// Config for graph exporter
type Config struct {
	SourceType       string                 // "parquet"
	SourcePath       string                 // Path to parquet files
	TargetType       string                 // "neo4j"
	TargetConnection map[string]interface{} // Neo4j connection config
	BatchSize        int                    // Batch size for export
}

// NewGraphExporter creates a new graph exporter
func NewGraphExporter(config Config) (*GraphExporter, error) {
	// Create source storage (Parquet)
	sourceConfig := map[string]interface{}{
		"type": config.SourceType,
		"path": config.SourcePath,
	}
	sourceStorage, err := analytics.NewStorage(sourceConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create source storage: %w", err)
	}

	// Create target storage (Neo4j)
	targetConfig := config.TargetConnection
	targetConfig["type"] = config.TargetType
	targetStorage, err := analytics.NewStorage(targetConfig)
	if err != nil {
		sourceStorage.Close()
		return nil, fmt.Errorf("failed to create target storage: %w", err)
	}

	batchSize := config.BatchSize
	if batchSize == 0 {
		batchSize = 1000
	}

	return &GraphExporter{
		sourceStorage: sourceStorage,
		targetStorage: targetStorage,
		sourcePath:    config.SourcePath,
		batchSize:     batchSize,
	}, nil
}

// ExportResult contains the result of an export operation
type ExportResult struct {
	Success       bool
	DocumentCount int
	ElementCount  int
	RelationshipCount int
	EmbeddingCount int
	Error         string
	ExportRunID   string
}

// ExportToNeo4j exports all analytics data to Neo4j
func (e *GraphExporter) ExportToNeo4j() ExportResult {
	result := ExportResult{
		Success:     true,
		ExportRunID: fmt.Sprintf("export_%s", time.Now().Format("20060102_150405")),
	}

	log.Println("========================================")
	log.Println("STARTING NEO4J GRAPH EXPORT")
	log.Println("========================================")

	// Export documents
	log.Println("Exporting documents...")
	docCount, err := e.exportDocuments()
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("failed to export documents: %v", err)
		log.Printf("ERROR: %s", result.Error)
		return result
	}
	result.DocumentCount = docCount
	log.Printf("✓ Exported %d documents", docCount)

	// Export elements
	log.Println("Exporting elements...")
	elemCount, err := e.exportElements()
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("failed to export elements: %v", err)
		log.Printf("ERROR: %s", result.Error)
		return result
	}
	result.ElementCount = elemCount
	log.Printf("✓ Exported %d elements", elemCount)

	// Export relationships
	log.Println("Exporting relationships...")
	relCount, err := e.exportRelationships()
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("failed to export relationships: %v", err)
		log.Printf("ERROR: %s", result.Error)
		return result
	}
	result.RelationshipCount = relCount
	log.Printf("✓ Exported %d relationships", relCount)

	// Export embeddings
	log.Println("Exporting embeddings...")
	embCount, err := e.exportEmbeddings()
	if err != nil {
		// Embeddings are optional, log warning but don't fail
		log.Printf("Warning: failed to export embeddings: %v", err)
	} else {
		result.EmbeddingCount = embCount
		log.Printf("✓ Exported %d embeddings", embCount)
	}

	log.Println("========================================")
	log.Printf("NEO4J EXPORT COMPLETED SUCCESSFULLY")
	log.Printf("  Documents: %d", result.DocumentCount)
	log.Printf("  Elements: %d", result.ElementCount)
	log.Printf("  Relationships: %d", result.RelationshipCount)
	log.Printf("  Embeddings: %d", result.EmbeddingCount)
	log.Println("========================================")

	return result
}

// exportDocuments exports documents from Parquet to Neo4j
func (e *GraphExporter) exportDocuments() (int, error) {
	documents, err := e.readDocumentsFromParquet()
	if err != nil {
		return 0, fmt.Errorf("failed to read documents: %w", err)
	}

	if len(documents) == 0 {
		log.Println("No documents found to export")
		return 0, nil
	}

	// Export in batches
	totalExported := 0
	for i := 0; i < len(documents); i += e.batchSize {
		end := i + e.batchSize
		if end > len(documents) {
			end = len(documents)
		}
		batch := documents[i:end]

		if err := e.targetStorage.AppendDocuments(batch); err != nil {
			return totalExported, fmt.Errorf("failed to export document batch: %w", err)
		}

		totalExported += len(batch)
		log.Printf("Exported documents: %d/%d", totalExported, len(documents))
	}

	return totalExported, nil
}

// exportElements exports elements from Parquet to Neo4j
func (e *GraphExporter) exportElements() (int, error) {
	elements, err := e.readElementsFromParquet()
	if err != nil {
		return 0, fmt.Errorf("failed to read elements: %w", err)
	}

	if len(elements) == 0 {
		log.Println("No elements found to export")
		return 0, nil
	}

	// Export in batches
	totalExported := 0
	for i := 0; i < len(elements); i += e.batchSize {
		end := i + e.batchSize
		if end > len(elements) {
			end = len(elements)
		}
		batch := elements[i:end]

		if err := e.targetStorage.AppendElements(batch); err != nil {
			return totalExported, fmt.Errorf("failed to export element batch: %w", err)
		}

		totalExported += len(batch)
		log.Printf("Exported elements: %d/%d", totalExported, len(elements))
	}

	return totalExported, nil
}

// exportRelationships exports relationships from Parquet to Neo4j
func (e *GraphExporter) exportRelationships() (int, error) {
	relationships, err := e.readRelationshipsFromParquet()
	if err != nil {
		return 0, fmt.Errorf("failed to read relationships: %w", err)
	}

	if len(relationships) == 0 {
		log.Println("No relationships found to export")
		return 0, nil
	}

	// Export in batches
	totalExported := 0
	for i := 0; i < len(relationships); i += e.batchSize {
		end := i + e.batchSize
		if end > len(relationships) {
			end = len(relationships)
		}
		batch := relationships[i:end]

		if err := e.targetStorage.AppendRelationships(batch); err != nil {
			return totalExported, fmt.Errorf("failed to export relationship batch: %w", err)
		}

		totalExported += len(batch)
		log.Printf("Exported relationships: %d/%d", totalExported, len(relationships))
	}

	return totalExported, nil
}

// exportEmbeddings exports embeddings from Parquet to Neo4j
func (e *GraphExporter) exportEmbeddings() (int, error) {
	embeddings, err := e.readEmbeddingsFromParquet()
	if err != nil {
		return 0, fmt.Errorf("failed to read embeddings: %w", err)
	}

	if len(embeddings) == 0 {
		log.Println("No embeddings found to export")
		return 0, nil
	}

	// Export in batches
	totalExported := 0
	for i := 0; i < len(embeddings); i += e.batchSize {
		end := i + e.batchSize
		if end > len(embeddings) {
			end = len(embeddings)
		}
		batch := embeddings[i:end]

		if err := e.targetStorage.AppendEmbeddings(batch); err != nil {
			return totalExported, fmt.Errorf("failed to export embedding batch: %w", err)
		}

		totalExported += len(batch)
		log.Printf("Exported embeddings: %d/%d", totalExported, len(embeddings))
	}

	return totalExported, nil
}

// readDocumentsFromParquet reads all documents from Parquet files
func (e *GraphExporter) readDocumentsFromParquet() ([]analytics.Document, error) {
	var documents []analytics.Document

	// Find all document parquet files
	err := filepath.Walk(e.sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip if not a parquet file or not a documents file
		if info.IsDir() || !strings.HasSuffix(path, ".parquet") {
			return nil
		}

		if !strings.Contains(path, "documents") {
			return nil
		}

		// Read parquet file
		docs, err := readDocumentsParquet(path)
		if err != nil {
			log.Printf("Warning: failed to read documents from %s: %v", path, err)
			return nil // Continue walking
		}

		documents = append(documents, docs...)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk source path: %w", err)
	}

	return documents, nil
}

// readElementsFromParquet reads all elements from Parquet files
func (e *GraphExporter) readElementsFromParquet() ([]analytics.Element, error) {
	var elements []analytics.Element

	// Find all element parquet files
	err := filepath.Walk(e.sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip if not a parquet file or not an elements file
		if info.IsDir() || !strings.HasSuffix(path, ".parquet") {
			return nil
		}

		if !strings.Contains(path, "elements") {
			return nil
		}

		// Read parquet file
		elems, err := readElementsParquet(path)
		if err != nil {
			log.Printf("Warning: failed to read elements from %s: %v", path, err)
			return nil // Continue walking
		}

		elements = append(elements, elems...)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk source path: %w", err)
	}

	return elements, nil
}

// readRelationshipsFromParquet reads all relationships from Parquet files
func (e *GraphExporter) readRelationshipsFromParquet() ([]analytics.Relationship, error) {
	var relationships []analytics.Relationship

	// Find all relationship parquet files
	err := filepath.Walk(e.sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip if not a parquet file or not a relationships file
		if info.IsDir() || !strings.HasSuffix(path, ".parquet") {
			return nil
		}

		if !strings.Contains(path, "relationships") {
			return nil
		}

		// Read parquet file
		rels, err := readRelationshipsParquet(path)
		if err != nil {
			log.Printf("Warning: failed to read relationships from %s: %v", path, err)
			return nil // Continue walking
		}

		relationships = append(relationships, rels...)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk source path: %w", err)
	}

	return relationships, nil
}

// readEmbeddingsFromParquet reads all embeddings from Parquet files
func (e *GraphExporter) readEmbeddingsFromParquet() ([]analytics.Embedding, error) {
	var embeddings []analytics.Embedding

	// Find all embedding parquet files
	err := filepath.Walk(e.sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip if not a parquet file or not an embeddings file
		if info.IsDir() || !strings.HasSuffix(path, ".parquet") {
			return nil
		}

		if !strings.Contains(path, "embeddings") {
			return nil
		}

		// Read parquet file
		embs, err := readEmbeddingsParquet(path)
		if err != nil {
			log.Printf("Warning: failed to read embeddings from %s: %v", path, err)
			return nil // Continue walking
		}

		embeddings = append(embeddings, embs...)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk source path: %w", err)
	}

	return embeddings, nil
}

// Close closes both source and target storage
func (e *GraphExporter) Close() error {
	var errs []error

	if err := e.sourceStorage.Close(); err != nil {
		errs = append(errs, fmt.Errorf("failed to close source storage: %w", err))
	}

	if err := e.targetStorage.Close(); err != nil {
		errs = append(errs, fmt.Errorf("failed to close target storage: %w", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("close errors: %v", errs)
	}

	return nil
}

// Helper functions to read Parquet files (simplified - just returns empty slices for now)
// TODO: Implement actual Parquet reading using Arrow

func readDocumentsParquet(path string) ([]analytics.Document, error) {
	// Open parquet file
	pf, err := file.OpenParquetFile(path, false)
	if err != nil {
		return nil, fmt.Errorf("failed to open parquet file: %w", err)
	}
	defer pf.Close()

	// TODO: Implement proper Parquet reading
	// For now, return empty to avoid compile errors
	log.Printf("TODO: Read documents from %s", path)
	return []analytics.Document{}, nil
}

func readElementsParquet(path string) ([]analytics.Element, error) {
	pf, err := file.OpenParquetFile(path, false)
	if err != nil {
		return nil, fmt.Errorf("failed to open parquet file: %w", err)
	}
	defer pf.Close()

	// TODO: Implement proper Parquet reading
	log.Printf("TODO: Read elements from %s", path)
	return []analytics.Element{}, nil
}

func readRelationshipsParquet(path string) ([]analytics.Relationship, error) {
	pf, err := file.OpenParquetFile(path, false)
	if err != nil {
		return nil, fmt.Errorf("failed to open parquet file: %w", err)
	}
	defer pf.Close()

	// TODO: Implement proper Parquet reading
	log.Printf("TODO: Read relationships from %s", path)
	return []analytics.Relationship{}, nil
}

func readEmbeddingsParquet(path string) ([]analytics.Embedding, error) {
	pf, err := file.OpenParquetFile(path, false)
	if err != nil {
		return nil, fmt.Errorf("failed to open parquet file: %w", err)
	}
	defer pf.Close()

	// TODO: Implement proper Parquet reading
	log.Printf("TODO: Read embeddings from %s", path)
	return []analytics.Embedding{}, nil
}
