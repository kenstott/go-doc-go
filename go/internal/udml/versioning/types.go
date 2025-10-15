package versioning

import (
	"time"
)

// VersionMetadata describes a UDML version
type VersionMetadata struct {
	Version         string                 // Semantic version (e.g., "1.0.0", "1.1.0")
	CreatedAt       time.Time              // When this version was created
	ParserVersion   string                 // Version of parsers used
	EmbeddingModel  string                 // Embedding model version (e.g., "text-embedding-ada-002")
	OntologyVersion string                 // Ontology schema version
	SchemaChanges   map[string]interface{} // Structured schema change log
	Notes           string                 // Human-readable notes
	Tags            []string               // Version tags (e.g., "production", "beta", "archived")

	// Statistics
	ElementCount    int64 // Total elements in this version
	DocumentCount   int64 // Total documents in this version
	SizeBytes       int64 // Total storage size

	// Audit trail
	CreatedBy       string // User or system that created this version
	Deprecated      bool   // Whether this version is deprecated
	DeprecatedAt    *time.Time // When version was deprecated
}

// VersionDiff represents differences between two versions
type VersionDiff struct {
	FromVersion string
	ToVersion   string
	ComparedAt  time.Time

	// Schema changes
	SchemaChanges SchemaChanges

	// Data changes
	ElementsAdded   int64
	ElementsRemoved int64
	ElementsChanged int64

	// Document changes
	DocumentsAdded   int64
	DocumentsRemoved int64
	DocumentsChanged int64

	// Detailed changes
	FieldChanges map[string]FieldChange // Field-level changes
	TypeChanges  map[string]TypeChange  // Element type changes

	// Size impact
	SizeDelta int64 // Change in storage size (bytes)
}

// SchemaChanges represents schema-level changes
type SchemaChanges struct {
	FieldsAdded   []string
	FieldsRemoved []string
	FieldsChanged []FieldSchemaChange
	TypesAdded    []string // New element types
	TypesRemoved  []string // Removed element types
}

// FieldSchemaChange represents a change to a field schema
type FieldSchemaChange struct {
	FieldName    string
	OldType      string
	NewType      string
	OldNullable  bool
	NewNullable  bool
	ChangeReason string
}

// FieldChange represents changes to a specific field's data
type FieldChange struct {
	FieldName       string
	ValuesAdded     int64 // Number of new non-null values
	ValuesRemoved   int64 // Number of values that became null
	ValuesChanged   int64 // Number of changed values
	DistinctBefore  int64 // Distinct values before
	DistinctAfter   int64 // Distinct values after
	NullCountBefore int64
	NullCountAfter  int64
}

// TypeChange represents changes to elements of a specific type
type TypeChange struct {
	ElementType  string
	CountBefore  int64
	CountAfter   int64
	CountAdded   int64
	CountRemoved int64
	CountChanged int64
}

// VersionQuery represents a query against a specific version
type VersionQuery struct {
	Version   string            // Target version
	AsOfTime  *time.Time        // Time-travel: query version as of this time
	Tags      []string          // Query versions with these tags
	Filters   map[string]string // Additional filters
}

// VersionStats represents aggregate statistics for a version
type VersionStats struct {
	Version       string
	ElementCounts map[string]int64 // Count by element_type
	DocumentCounts map[string]int64 // Count by source_name
	TotalElements int64
	TotalDocuments int64
	SizeBytes     int64
	ComputedAt    time.Time
}

// MigrationRecord tracks version migrations
type MigrationRecord struct {
	FromVersion string
	ToVersion   string
	StartedAt   time.Time
	CompletedAt *time.Time
	Status      MigrationStatus
	ErrorMsg    string
	Rollback    bool // Whether this is a rollback migration

	// Progress tracking
	TotalRecords     int64
	ProcessedRecords int64
	FailedRecords    int64
}

// MigrationStatus represents migration state
type MigrationStatus string

const (
	MigrationPending   MigrationStatus = "pending"
	MigrationRunning   MigrationStatus = "running"
	MigrationCompleted MigrationStatus = "completed"
	MigrationFailed    MigrationStatus = "failed"
	MigrationRolledBack MigrationStatus = "rolled_back"
)
