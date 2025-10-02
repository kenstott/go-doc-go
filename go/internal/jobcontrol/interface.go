package jobcontrol

import "time"

// JobControl defines the interface for job control database operations
type JobControl interface {
	// Document Queue Operations
	EnqueueDocument(docID, source string, metadata map[string]interface{}) error
	ClaimNextDocument(workerID string) (*DocumentInfo, error)
	CompleteDocument(docID, workerID string, success bool, errorMsg string) error
	IsDocumentQueued(docID string) (bool, error)

	// Worker Management
	RegisterWorker(workerID string, info map[string]interface{}) error
	UpdateWorkerHeartbeat(workerID string) error
	GetActiveWorkers() ([]*WorkerInfo, error)

	// Leader Election
	ElectLeader(workerID string, info map[string]interface{}) (bool, error)
	GetCurrentLeader() (*LeaderInfo, error)
	UpdateLeaderHeartbeat(workerID string) error
	ReleaseLeadership(workerID string) error

	// Per-Source Leader Election
	ElectSourceLeader(sourceName, workerID, info string) (bool, error)
	GetSourceLeader(sourceName string) (string, error)
	UpdateSourceLeaderHeartbeat(sourceName, workerID string) error
	ReleaseSourceLeadership(sourceName, workerID string) error

	// Change Tracking
	HasDocumentChanged(docID, source string, currentModified *time.Time, currentHash string) (bool, error)
	StoreDocumentMetadata(docID, source string, lastModified *time.Time, contentHash string, fileSize *int64, processingStats map[string]interface{}) error
	GetDocumentMetadata(docID string) (*DocumentMetadata, error)

	// Status & Monitoring
	GetProcessingStatus() (*ProcessingStatus, error)
	CleanupStaleWorkers(timeout int) (int, error)
	CleanupStaleLeader(timeout int) (bool, error)

	// Lifecycle
	Close() error
}
