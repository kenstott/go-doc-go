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

	// Close closes the embedding generator and releases resources
	Close() error
}

// Config holds embedding configuration
type Config struct {
	Enabled          bool              `json:"enabled" yaml:"enabled"`
	Provider         string            `json:"provider" yaml:"provider"`
	Model            string            `json:"model" yaml:"model"`
	ModelPath        string            `json:"model_path,omitempty" yaml:"model_path,omitempty"` // Path to ONNX model directory
	Dimensions       int               `json:"dimensions" yaml:"dimensions"`
	ChunkSize        int               `json:"chunk_size" yaml:"chunk_size"`
	Overlap          int               `json:"overlap" yaml:"overlap"`
	Contextual       bool              `json:"contextual" yaml:"contextual"`
	PredecessorCount int               `json:"predecessor_count" yaml:"predecessor_count"`
	SuccessorCount   int               `json:"successor_count" yaml:"successor_count"`
	CacheDir         string            `json:"cache_dir,omitempty" yaml:"cache_dir,omitempty"`
	AdditionalConfig map[string]interface{} `json:"-" yaml:"-"`
}
