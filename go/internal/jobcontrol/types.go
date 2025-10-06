package jobcontrol

import (
	"time"
)

// DocumentStatus represents the status of a document in the queue
type DocumentStatus string

const (
	StatusPending    DocumentStatus = "pending"
	StatusProcessing DocumentStatus = "processing"
	StatusCompleted  DocumentStatus = "completed"
	StatusFailed     DocumentStatus = "failed"
)

// DocumentInfo represents a document in the job queue
type DocumentInfo struct {
	DocID       string                 `json:"doc_id"`
	Source      string                 `json:"source"`
	Metadata    map[string]interface{} `json:"metadata"`
	Status      DocumentStatus         `json:"status"`
	ClaimedBy   string                 `json:"claimed_by,omitempty"`
	ClaimedAt   *time.Time             `json:"claimed_at,omitempty"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	RetryCount  int                    `json:"retry_count"`
	ErrorMsg    string                 `json:"error_message,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
}

// WorkerInfo represents a registered worker
type WorkerInfo struct {
	WorkerID     string                 `json:"worker_id"`
	Info         map[string]interface{} `json:"info"`
	LastHeartbeat time.Time             `json:"last_heartbeat"`
	RegisteredAt time.Time              `json:"registered_at"`
}

// LeaderInfo represents the current leader
type LeaderInfo struct {
	WorkerID     string                 `json:"worker_id"`
	Info         map[string]interface{} `json:"info"`
	ElectedAt    time.Time              `json:"elected_at"`
	LastHeartbeat time.Time             `json:"last_heartbeat"`
}

// ProcessingStatus represents overall queue status
type ProcessingStatus struct {
	QueueStatus string         `json:"queue_status"`
	Documents   map[string]int `json:"documents"`
	Workers     map[string]int `json:"workers"`
	Leader      *LeaderInfo    `json:"leader,omitempty"`
}

// DocumentMetadata represents stored metadata for change tracking
type DocumentMetadata struct {
	DocID            string                 `json:"doc_id"`
	Source           string                 `json:"source"`
	LastModified     *time.Time             `json:"last_modified,omitempty"`
	ContentHash      string                 `json:"content_hash,omitempty"`
	FileSize         *int64                 `json:"file_size,omitempty"`
	ProcessingStats  map[string]interface{} `json:"processing_stats,omitempty"`
	LastProcessedAt  time.Time              `json:"last_processed_at"`
}
