package ontology

import (
	"context"
	"log"
	"sync"

	"github.com/kennethstott/doculyzer-go-conversion/internal/analytics"
)

// EmbeddingResolver provides on-demand lookup of element embeddings with LRU caching
type EmbeddingResolver struct {
	storage      analytics.Storage
	cache        map[string][]float64 // element_id -> embedding vector (LRU cache)
	mu           sync.RWMutex
	maxCacheSize int
}

// NewEmbeddingResolver creates a new embedding resolver with on-demand loading from Storage
// Uses LRU cache to avoid loading millions of embeddings into memory at once
func NewEmbeddingResolver(ctx context.Context, storage analytics.Storage) (*EmbeddingResolver, error) {
	resolver := &EmbeddingResolver{
		storage:      storage,
		cache:        make(map[string][]float64),
		maxCacheSize: 10000, // Cache up to 10k embeddings (configurable)
	}

	log.Printf("✓ Embedding resolver initialized with on-demand loading (cache size: %d)", resolver.maxCacheSize)

	return resolver, nil
}

// GetEmbedding retrieves the embedding vector for an element_id
// First checks cache, then queries storage on cache miss
func (r *EmbeddingResolver) GetEmbedding(elementID string) ([]float64, bool) {
	// Check cache first (read lock)
	r.mu.RLock()
	if embedding, exists := r.cache[elementID]; exists {
		r.mu.RUnlock()
		return embedding, true
	}
	r.mu.RUnlock()

	// Cache miss - query from storage (write lock for cache update)
	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-check cache in case another goroutine loaded it
	if embedding, exists := r.cache[elementID]; exists {
		return embedding, true
	}

	// Query from storage - QueryEmbeddings returns analytics.Embedding structs
	// which already have the embedding vectors properly parsed
	embeddings, err := r.storage.QueryEmbeddings(map[string]interface{}{
		"element_id": elementID,
	})
	if err != nil {
		log.Printf("DEBUG: Failed to query embedding for element %s: %v", elementID, err)
		return nil, false
	}

	if len(embeddings) == 0 {
		// Element has no embedding
		return nil, false
	}

	// Extract the embedding vector from the analytics.Embedding struct
	embeddingVector := embeddings[0].Embedding
	if len(embeddingVector) == 0 {
		log.Printf("DEBUG: Element %s has empty embedding vector", elementID)
		return nil, false
	}

	// Cache the embedding (simple eviction: clear cache if full)
	if len(r.cache) >= r.maxCacheSize {
		// Simple cache eviction: clear entire cache when full
		// TODO: Implement proper LRU eviction if needed
		log.Printf("DEBUG: Embedding cache full (%d entries), clearing cache", len(r.cache))
		r.cache = make(map[string][]float64)
	}

	r.cache[elementID] = embeddingVector
	return embeddingVector, true
}

// Count returns the number of embeddings currently in cache
func (r *EmbeddingResolver) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.cache)
}
