package ontology

import (
	"time"
)

// ExtractionTaskType represents the type of extraction task
type ExtractionTaskType string

const (
	TaskTypeEntityMapping      ExtractionTaskType = "entity_mapping"
	TaskTypeRelationshipRule   ExtractionTaskType = "relationship_rule"
)

// ExtractionTaskStatus represents the status of an extraction task
type ExtractionTaskStatus string

const (
	TaskStatusPending    ExtractionTaskStatus = "pending"
	TaskStatusProcessing ExtractionTaskStatus = "processing"
	TaskStatusCompleted  ExtractionTaskStatus = "completed"
	TaskStatusFailed     ExtractionTaskStatus = "failed"
)

// ExtractionTask represents a unit of work for distributed extraction
type ExtractionTask struct {
	TaskID      string               `json:"task_id"`       // Unique task identifier
	RunID       string               `json:"run_id"`        // Extraction run identifier
	Type        ExtractionTaskType   `json:"type"`          // Task type (entity_mapping or relationship_rule)
	MappingID   string               `json:"mapping_id"`    // Entity mapping ID (for entity tasks)
	RuleID      string               `json:"rule_id"`       // Relationship rule ID (for relationship tasks)
	Status      ExtractionTaskStatus `json:"status"`        // Task status
	ClaimedBy   string               `json:"claimed_by"`    // Worker ID that claimed this task
	ClaimedAt   *time.Time           `json:"claimed_at"`    // When task was claimed
	CompletedAt *time.Time           `json:"completed_at"`  // When task completed
	RetryCount  int                  `json:"retry_count"`   // Number of retry attempts
	ErrorMsg    string               `json:"error_message"` // Error message if failed
	CreatedAt   time.Time            `json:"created_at"`    // Task creation time
}

// ExtractionTaskResult represents the completion metrics of an extraction task
// (actual entities/relationships are written directly to Storage)
type ExtractionTaskResult struct {
	TaskID       string         `json:"task_id"`       // Associated task ID
	RunID        string         `json:"run_id"`        // Extraction run identifier
	Type         ExtractionTaskType `json:"type"`      // Result type
	MappingID    string         `json:"mapping_id"`    // Entity mapping ID (for entity results)
	RuleID       string         `json:"rule_id"`       // Relationship rule ID (for relationship results)
	EntityCount  int            `json:"entity_count"`  // Number of entities extracted
	RelationshipCount int       `json:"relationship_count"` // Number of relationships extracted
	CreatedAt    time.Time      `json:"created_at"`    // Result creation time
}

// ExtractionJobControl defines the interface for managing extraction tasks
type ExtractionJobControl interface {
	// Task management
	CreateTasks(runID string, tasks []ExtractionTask) error
	ClaimTask(runID string, taskType ExtractionTaskType, workerID string) (*ExtractionTask, error)
	CompleteTask(taskID string, result ExtractionTaskResult) error
	FailTask(taskID string, errorMsg string) error
	ReleaseTask(taskID string) error

	// Status queries
	GetTaskStatus(runID string) (map[ExtractionTaskStatus]int, error)
	GetPendingTaskCount(runID string, taskType ExtractionTaskType) (int, error)
	IsPhaseComplete(runID string, taskType ExtractionTaskType) (bool, error)

	// Cleanup
	CleanupRun(runID string) error
}

