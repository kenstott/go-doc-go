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
		provider = "fastembed" // Default to Python shim
	}

	log.Printf("Creating embedding generator with provider: %s", provider)

	switch provider {
	case "onnx":
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

	case "fastembed":
		// Use Python shim (existing implementation)
		generator, err := NewPythonShimGenerator(config)
		if err != nil {
			return nil, fmt.Errorf("failed to create Python shim generator: %w", err)
		}

		log.Printf("Created Python shim embedding generator (model: %s)", config.Model)
		return generator, nil

	default:
		return nil, fmt.Errorf("unsupported embedding provider: %s", provider)
	}
}
