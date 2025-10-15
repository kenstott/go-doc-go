package query

import (
	"context"
	"time"
)

// QueryBackend is the interface that all query engines must implement.
// This allows swappable backends (DuckDB, PostgreSQL, Neo4j, etc.) without changing core logic.
type QueryBackend interface {
	// GetName returns the backend identifier (e.g., "duckdb", "postgresql", "neo4j")
	GetName() string

	// GetVersion returns the backend version information
	GetVersion() string

	// Initialize prepares the backend for queries (e.g., mount Parquet files, connect to DB)
	Initialize(ctx context.Context, config BackendConfig) error

	// Translate converts a universal Expression into a backend-specific NativeQuery
	Translate(expr *Expression, opts TranslateOptions) (*NativeQuery, error)

	// Execute runs a native query and returns results
	Execute(ctx context.Context, query *NativeQuery) (*QueryResult, error)

	// SupportsFeature checks if the backend supports a specific feature
	// Features: "jsonpath", "similarity", "regex", "full_text_search", "graph_traversal"
	SupportsFeature(feature string) bool

	// Explain returns the query execution plan (for optimization/debugging)
	Explain(ctx context.Context, query *NativeQuery) (*QueryPlan, error)

	// Close releases backend resources
	Close() error
}

// BackendConfig holds backend-specific configuration
type BackendConfig struct {
	// Common fields
	Type    string                 // Backend type: "duckdb", "postgresql", etc.
	Timeout time.Duration          // Query timeout
	Config  map[string]interface{} // Backend-specific config

	// Parquet-related (for DuckDB, Arrow, etc.)
	ParquetPath string // Path to Hive-partitioned Parquet files

	// Database-related (for PostgreSQL, Neo4j, etc.)
	ConnectionString string
	MaxConnections   int

	// Features
	EnableSimilarity bool // Enable semantic similarity queries
	EmbeddingModel   string
}

// TranslateOptions controls how Expressions are translated to native queries
type TranslateOptions struct {
	// Query optimization
	EnablePushdown   bool // Push predicates down to storage layer
	EnablePartitions bool // Use partition pruning

	// Result control
	Limit  int
	Offset int

	// Debugging
	Verbose bool // Include translation metadata
}

// QueryPlan represents the execution plan for a query
type QueryPlan struct {
	BackendName string                 // Which backend generated this plan
	RawPlan     string                 // Backend-specific plan representation
	Steps       []QueryPlanStep        // Parsed plan steps
	Metadata    map[string]interface{} // Backend-specific metadata
	EstimatedCost float64              // Estimated query cost
}

// QueryPlanStep represents one step in the execution plan
type QueryPlanStep struct {
	Operator    string  // e.g., "SCAN", "FILTER", "JOIN", "AGGREGATE"
	Description string
	EstimatedRows int
	EstimatedCost float64
}

// BackendCapabilities describes what a backend supports
type BackendCapabilities struct {
	SupportsJSONPath        bool
	SupportsSimilarity      bool
	SupportsRegex           bool
	SupportsFullTextSearch  bool
	SupportsGraphTraversal  bool
	SupportsPartitionPruning bool
	SupportsVectorized      bool
	SupportsParallel        bool
}

// GetCapabilities returns the backend's capabilities
func (bc *BackendCapabilities) GetCapabilities() BackendCapabilities {
	return *bc
}
