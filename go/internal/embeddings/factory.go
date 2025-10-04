package embeddings

import (
	"fmt"
	"log"
	"path/filepath"
)

// CreateEmbeddingGenerator creates an embedding generator based on configuration
func CreateEmbeddingGenerator(config Config) (EmbeddingGenerator, error) {
	provider := config.Provider
	if provider == "" {
		provider = "onnx" // Default to native ONNX
	}

	log.Printf("Creating embedding generator with provider: %s", provider)

	// Only ONNX provider is supported in Go worker
	if provider != "onnx" {
		return nil, fmt.Errorf("Go worker only supports 'onnx' provider (requested: %s)", provider)
	}

	// Use native Go ONNX Runtime
	modelDir := config.ModelPath
	if modelDir == "" {
		// Default model path
		modelDir = filepath.Join("go", "models", "all-MiniLM-L6-v2")
	}

	generator, err := NewOnnxEmbeddingGenerator(config, modelDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create ONNX generator: %w", err)
	}

	log.Printf("Created native ONNX embedding generator (model: %s)", modelDir)
	return generator, nil
}
