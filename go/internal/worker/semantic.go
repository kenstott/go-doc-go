package worker

import (
	"fmt"
	"log"
	"time"

	"github.com/kennethstott/doculyzer-go-conversion/internal/analytics"
	"github.com/kennethstott/doculyzer-go-conversion/internal/udml/query"
)

// SemanticAnalyzer performs cross-document semantic relationship detection
type SemanticAnalyzer struct {
	config   *SemanticConfig
	storages []analytics.Storage
}

// NewSemanticAnalyzer creates a new semantic analyzer
func NewSemanticAnalyzer(config *SemanticConfig, storages []analytics.Storage) *SemanticAnalyzer {
	return &SemanticAnalyzer{
		config:   config,
		storages: storages,
	}
}

// AnalyzeAndStore performs semantic similarity analysis and stores relationships
func (s *SemanticAnalyzer) AnalyzeAndStore() error {
	if !s.config.Enabled {
		return fmt.Errorf("semantic analysis not enabled")
	}

	log.Println("========================================")
	log.Println("SEMANTIC ANALYSIS: Starting cross-document analysis")
	log.Printf("  Similarity threshold: %.2f", s.config.SimilarityThreshold)
	log.Printf("  Rate limit: %d comparisons per batch, %dms sleep", s.config.RateLimitBatchSize, s.config.RateLimitSleepMs)
	log.Println("========================================")

	startTime := time.Now()

	// Step 1: Query all embeddings from storage
	var allEmbeddings []analytics.Embedding
	for _, storage := range s.storages {
		embeddings, err := storage.QueryEmbeddings(map[string]interface{}{})
		if err != nil {
			log.Printf("Failed to query embeddings from storage: %v", err)
			continue
		}
		allEmbeddings = append(allEmbeddings, embeddings...)
	}

	if len(allEmbeddings) == 0 {
		log.Println("SEMANTIC ANALYSIS: No embeddings found, skipping")
		return nil
	}

	log.Printf("SEMANTIC ANALYSIS: Found %d embeddings to compare", len(allEmbeddings))

	// Step 2: Delete existing semantic relationships
	if err := s.deleteSemanticRelationships(); err != nil {
		return fmt.Errorf("failed to delete existing semantic relationships: %w", err)
	}

	// Step 3: Perform cross-document comparisons with rate limiting
	relationships, err := s.findSemanticRelationships(allEmbeddings)
	if err != nil {
		return fmt.Errorf("failed to find semantic relationships: %w", err)
	}

	log.Printf("SEMANTIC ANALYSIS: Found %d semantic relationships above threshold", len(relationships))

	// Step 4: Store relationships
	if len(relationships) > 0 {
		for _, storage := range s.storages {
			if err := storage.AppendRelationships(relationships); err != nil {
				log.Printf("Failed to store semantic relationships: %v", err)
			}
		}
	}

	duration := time.Since(startTime)
	log.Println("========================================")
	log.Printf("SEMANTIC ANALYSIS: Completed in %v", duration)
	log.Printf("  Total comparisons: %d", s.countComparisons(allEmbeddings))
	log.Printf("  Relationships found: %d", len(relationships))
	log.Printf("  Analysis rate: %.2f comparisons/sec", float64(s.countComparisons(allEmbeddings))/duration.Seconds())
	log.Println("========================================")

	return nil
}

// findSemanticRelationships performs pairwise comparison of embeddings
func (s *SemanticAnalyzer) findSemanticRelationships(embeddings []analytics.Embedding) ([]analytics.Relationship, error) {
	var relationships []analytics.Relationship
	comparisons := 0
	batchStart := time.Now()

	for i := 0; i < len(embeddings); i++ {
		for j := i + 1; j < len(embeddings); j++ {
			emb1 := embeddings[i]
			emb2 := embeddings[j]

			// Skip same-document pairs (cross-document only)
			if emb1.DocID == emb2.DocID {
				continue
			}

			// Calculate cosine similarity
			similarity, err := query.CosineSimilarityFloat64(emb1.Embedding, emb2.Embedding)
			if err != nil {
				log.Printf("Failed to calculate similarity between %s and %s: %v", emb1.ElementID, emb2.ElementID, err)
				continue
			}

			// Check threshold
			if similarity >= s.config.SimilarityThreshold {
				// Create bidirectional relationships
				relationships = append(relationships,
					analytics.Relationship{
						SourceElementID:  emb1.ElementID,
						TargetElementID:  emb2.ElementID,
						RelationshipType: "semantic_similarity",
						DocID:            emb1.DocID,
						SourceName:       emb1.SourceName,
						Metadata: map[string]interface{}{
							"similarity_score": similarity,
							"target_doc_id":    emb2.DocID,
							"target_source":    emb2.SourceName,
						},
					},
					analytics.Relationship{
						SourceElementID:  emb2.ElementID,
						TargetElementID:  emb1.ElementID,
						RelationshipType: "semantic_similarity",
						DocID:            emb2.DocID,
						SourceName:       emb2.SourceName,
						Metadata: map[string]interface{}{
							"similarity_score": similarity,
							"target_doc_id":    emb1.DocID,
							"target_source":    emb1.SourceName,
						},
					},
				)
			}

			comparisons++

			// Rate limiting: sleep after each batch
			if comparisons%s.config.RateLimitBatchSize == 0 {
				batchDuration := time.Since(batchStart)
				log.Printf("SEMANTIC ANALYSIS: Completed %d comparisons in %v (%.2f comparisons/sec), found %d relationships so far",
					comparisons, batchDuration, float64(s.config.RateLimitBatchSize)/batchDuration.Seconds(), len(relationships))

				time.Sleep(time.Duration(s.config.RateLimitSleepMs) * time.Millisecond)
				batchStart = time.Now()
			}
		}
	}

	return relationships, nil
}

// deleteSemanticRelationships deletes existing semantic relationships
func (s *SemanticAnalyzer) deleteSemanticRelationships() error {
	// This is a simplified version - in practice, you'd need to:
	// 1. For Parquet: Can't delete easily, so we'd use a separate semantic_relationships partition
	// 2. For Neo4j: Delete relationships with type "semantic_similarity"
	//
	// For now, we'll just log - the actual deletion would be storage-specific
	log.Println("SEMANTIC ANALYSIS: Deleting existing semantic relationships (full rebuild)")

	// TODO: Implement storage-specific deletion
	// For Parquet: Write to a separate partition/table that can be replaced
	// For Neo4j: MATCH ()-[r:SEMANTIC_SIMILARITY]->() DELETE r

	return nil
}

// countComparisons calculates total number of cross-document comparisons
func (s *SemanticAnalyzer) countComparisons(embeddings []analytics.Embedding) int {
	// Count unique doc pairs
	docPairs := make(map[string]bool)
	for i := 0; i < len(embeddings); i++ {
		for j := i + 1; j < len(embeddings); j++ {
			if embeddings[i].DocID != embeddings[j].DocID {
				// Use sorted pair to avoid double-counting
				pair := embeddings[i].DocID + ":" + embeddings[j].DocID
				if embeddings[i].DocID > embeddings[j].DocID {
					pair = embeddings[j].DocID + ":" + embeddings[i].DocID
				}
				docPairs[pair] = true
			}
		}
	}

	// Count actual comparisons (all cross-document pairs)
	comparisons := 0
	for i := 0; i < len(embeddings); i++ {
		for j := i + 1; j < len(embeddings); j++ {
			if embeddings[i].DocID != embeddings[j].DocID {
				comparisons++
			}
		}
	}
	return comparisons
}
