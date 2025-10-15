package embeddings

// EmbeddingGenerator defines the interface for generating embeddings
type EmbeddingGenerator interface {
	// Generate generates an embedding for a single text
	Generate(text string) ([]float64, error)

	// GenerateBatch generates embeddings for multiple texts
	GenerateBatch(texts []string) ([][]float64, error)

	// GetDimensions returns the number of dimensions in the embedding
	GetDimensions() int

	// GetModelName returns the name of the embedding model
	GetModelName() string

	// GetBatchSize returns the configured batch size for this generator
	GetBatchSize() int

	// Close closes the embedding generator and releases resources
	Close() error
}

// Config holds embedding configuration
type Config struct {
	Enabled          bool              `json:"enabled" toml:"enabled"`
	Provider         string            `json:"provider" toml:"provider"`
	Model            string            `json:"model" toml:"model"`
	ModelPath        string            `json:"model_path,omitempty" toml:"model_path,omitempty"` // Path to ONNX model directory
	Dimensions       int               `json:"dimensions" toml:"dimensions"`
	ChunkSize        int               `json:"chunk_size" toml:"chunk_size"`
	Overlap          int               `json:"overlap" toml:"overlap"`
	Contextual       bool              `json:"contextual" toml:"contextual"`
	PredecessorCount int               `json:"predecessor_count" toml:"predecessor_count"`
	SuccessorCount   int               `json:"successor_count" toml:"successor_count"`
	CacheDir         string            `json:"cache_dir,omitempty" toml:"cache_dir,omitempty"`
	PoolSize         int               `json:"pool_size,omitempty" toml:"pool_size,omitempty"` // ONNX session pool size (matches worker count)
	BatchSize        int               `json:"batch_size,omitempty" toml:"batch_size,omitempty"` // ONNX batch size for inference (default: 32)
	AdditionalConfig map[string]interface{} `json:"-" toml:"-"`
}
