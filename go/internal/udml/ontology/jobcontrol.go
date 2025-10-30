package ontology

// ExtractionJobControl defines the interface for managing extraction tasks
type ExtractionJobControl interface {
	// Leader election - atomically claim leadership for task creation
	ClaimLeaderRole(runID string, workerID string) (bool, error)

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
