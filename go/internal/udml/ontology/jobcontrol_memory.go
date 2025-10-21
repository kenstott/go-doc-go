package ontology

import (
	"fmt"
	"sync"
	"time"
)

// MemoryExtractionJobControl provides an in-memory implementation of ExtractionJobControl
// Useful for testing and single-process distributed extraction
// NOTE: Actual extraction results (entities/relationships) are written directly to Storage
type MemoryExtractionJobControl struct {
	tasks   map[string]*ExtractionTask        // taskID -> task
	results map[string]*ExtractionTaskResult  // taskID -> result (just metrics)
	leaders map[string]string                 // runID -> workerID (leader for that run)
	mu      sync.RWMutex
}

// NewMemoryExtractionJobControl creates a new in-memory job control
func NewMemoryExtractionJobControl() *MemoryExtractionJobControl {
	return &MemoryExtractionJobControl{
		tasks:   make(map[string]*ExtractionTask),
		results: make(map[string]*ExtractionTaskResult),
		leaders: make(map[string]string),
	}
}

// ClaimLeaderRole atomically claims the leader role for a run
// Returns true if this worker became the leader, false if another worker already claimed it
func (m *MemoryExtractionJobControl) ClaimLeaderRole(runID string, workerID string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if leader already exists for this run
	if existingLeader, exists := m.leaders[runID]; exists {
		// Leader already claimed
		return existingLeader == workerID, nil
	}

	// Claim leadership
	m.leaders[runID] = workerID
	return true, nil
}

// CreateTasks creates new extraction tasks
func (m *MemoryExtractionJobControl) CreateTasks(runID string, tasks []ExtractionTask) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, task := range tasks {
		// Ensure task has correct runID
		task.RunID = runID
		if task.CreatedAt.IsZero() {
			task.CreatedAt = time.Now()
		}
		if task.Status == "" {
			task.Status = TaskStatusPending
		}

		m.tasks[task.ID] = &task
	}

	return nil
}

// ClaimTask atomically claims the next available task for a worker
func (m *MemoryExtractionJobControl) ClaimTask(runID string, taskType ExtractionTaskType, workerID string) (*ExtractionTask, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Find first pending task of the requested type
	for _, task := range m.tasks {
		if task.RunID == runID &&
		   task.Type == taskType &&
		   task.Status == TaskStatusPending {

			// Claim the task
			now := time.Now()
			task.Status = TaskStatusClaimed
			task.ClaimedBy = workerID
			task.ClaimedAt = &now

			// Return a copy
			taskCopy := *task
			return &taskCopy, nil
		}
	}

	// No pending tasks found
	return nil, fmt.Errorf("no pending tasks of type %s for run %s", taskType, runID)
}

// CompleteTask marks a task as completed and stores the result metrics
func (m *MemoryExtractionJobControl) CompleteTask(taskID string, result ExtractionTaskResult) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, exists := m.tasks[taskID]
	if !exists {
		return fmt.Errorf("task %s not found", taskID)
	}

	// Update task status
	now := time.Now()
	task.Status = TaskStatusCompleted
	task.CompletedAt = &now

	// Store result metrics (just store it directly)
	m.results[taskID] = &result

	return nil
}

// FailTask marks a task as failed
func (m *MemoryExtractionJobControl) FailTask(taskID string, errorMsg string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, exists := m.tasks[taskID]
	if !exists {
		return fmt.Errorf("task %s not found", taskID)
	}

	task.Status = TaskStatusFailed
	task.Error = errorMsg

	return nil
}

// ReleaseTask releases a claimed task back to pending state
func (m *MemoryExtractionJobControl) ReleaseTask(taskID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, exists := m.tasks[taskID]
	if !exists {
		return fmt.Errorf("task %s not found", taskID)
	}

	task.Status = TaskStatusPending
	task.ClaimedBy = ""
	task.ClaimedAt = nil

	return nil
}

// GetTaskStatus returns count of tasks by status for a run
func (m *MemoryExtractionJobControl) GetTaskStatus(runID string) (map[ExtractionTaskStatus]int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	counts := make(map[ExtractionTaskStatus]int)
	for _, task := range m.tasks {
		if task.RunID == runID {
			counts[task.Status]++
		}
	}

	return counts, nil
}

// GetPendingTaskCount returns the number of pending tasks for a specific type
func (m *MemoryExtractionJobControl) GetPendingTaskCount(runID string, taskType ExtractionTaskType) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, task := range m.tasks {
		if task.RunID == runID &&
		   task.Type == taskType &&
		   task.Status == TaskStatusPending {
			count++
		}
	}

	return count, nil
}

// IsPhaseComplete checks if all tasks of a specific type are completed
func (m *MemoryExtractionJobControl) IsPhaseComplete(runID string, taskType ExtractionTaskType) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, task := range m.tasks {
		if task.RunID == runID && task.Type == taskType {
			if task.Status != TaskStatusCompleted && task.Status != TaskStatusFailed {
				return false, nil
			}
		}
	}

	return true, nil
}

// CleanupRun removes all tasks and results for a run
func (m *MemoryExtractionJobControl) CleanupRun(runID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Remove tasks
	for taskID, task := range m.tasks {
		if task.RunID == runID {
			delete(m.tasks, taskID)
			delete(m.results, taskID)
		}
	}

	return nil
}
