package ontology

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteJobControl provides SQLite-based job control for distributed extraction
// Uses WAL mode for multi-process safety
type SQLiteJobControl struct {
	db *sql.DB
}

// NewSQLiteJobControl creates a new SQLite-based job control
func NewSQLiteJobControl(dbPath string) (*SQLiteJobControl, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite database: %w", err)
	}

	// Enable WAL mode for multi-process safety
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	// Create schema
	if err := createSQLiteSchema(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	return &SQLiteJobControl{db: db}, nil
}

// createSQLiteSchema creates the database schema
func createSQLiteSchema(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS run_leaders (
		run_id TEXT PRIMARY KEY,
		worker_id TEXT NOT NULL,
		created_at TIMESTAMP NOT NULL
	);

	CREATE TABLE IF NOT EXISTS extraction_tasks (
		task_id TEXT PRIMARY KEY,
		run_id TEXT NOT NULL,
		type TEXT NOT NULL,
		entity_type TEXT,
		relationship_type TEXT,
		doc_ids TEXT,
		status TEXT NOT NULL,
		claimed_by TEXT,
		claimed_at TIMESTAMP,
		completed_at TIMESTAMP,
		error TEXT,
		created_at TIMESTAMP NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_run_status ON extraction_tasks(run_id, status);
	CREATE INDEX IF NOT EXISTS idx_run_type ON extraction_tasks(run_id, type);
	`

	_, err := db.Exec(schema)
	return err
}

// ClaimLeaderRole atomically claims the leader role for a run
// Returns true if this worker became the leader, false if another worker already claimed it
func (s *SQLiteJobControl) ClaimLeaderRole(runID string, workerID string) (bool, error) {
	result, err := s.db.Exec(`
		INSERT INTO run_leaders (run_id, worker_id, created_at)
		VALUES (?, ?, ?)
		ON CONFLICT (run_id) DO NOTHING
	`, runID, workerID, time.Now())

	if err != nil {
		return false, fmt.Errorf("failed to claim leader role: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	return rowsAffected > 0, nil // true = I'm the leader
}

// CreateTasks creates new extraction tasks
func (s *SQLiteJobControl) CreateTasks(runID string, tasks []ExtractionTask) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO extraction_tasks
		(task_id, run_id, type, entity_type, relationship_type, doc_ids, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, task := range tasks {
		docIDsJSON, _ := json.Marshal(task.DocIDs)
		_, err := stmt.Exec(
			task.ID,
			runID,
			task.Type,
			task.EntityType,
			task.RelationshipType,
			string(docIDsJSON),
			TaskStatusPending,
			time.Now(),
		)
		if err != nil {
			return fmt.Errorf("failed to insert task %s: %w", task.ID, err)
		}
	}

	return tx.Commit()
}

// ClaimTask atomically claims the next available task for a worker
func (s *SQLiteJobControl) ClaimTask(runID string, taskType ExtractionTaskType, workerID string) (*ExtractionTask, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Find first pending task
	var taskID, docIDsJSON string
	var entityType, relType sql.NullString

	err = tx.QueryRow(`
		SELECT task_id, entity_type, relationship_type, doc_ids
		FROM extraction_tasks
		WHERE run_id = ? AND type = ? AND status = ?
		LIMIT 1
	`, runID, taskType, TaskStatusPending).Scan(&taskID, &entityType, &relType, &docIDsJSON)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("no pending tasks of type %s for run %s", taskType, runID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query task: %w", err)
	}

	// Claim the task
	now := time.Now()
	_, err = tx.Exec(`
		UPDATE extraction_tasks
		SET status = ?, claimed_by = ?, claimed_at = ?
		WHERE task_id = ?
	`, TaskStatusClaimed, workerID, now, taskID)

	if err != nil {
		return nil, fmt.Errorf("failed to claim task: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Parse doc_ids
	var docIDs []string
	json.Unmarshal([]byte(docIDsJSON), &docIDs)

	return &ExtractionTask{
		ID:               taskID,
		RunID:            runID,
		Type:             taskType,
		EntityType:       entityType.String,
		RelationshipType: relType.String,
		DocIDs:           docIDs,
		Status:           TaskStatusClaimed,
		ClaimedBy:        workerID,
		ClaimedAt:        &now,
	}, nil
}

// CompleteTask marks a task as completed
func (s *SQLiteJobControl) CompleteTask(taskID string, result ExtractionTaskResult) error {
	now := time.Now()
	_, err := s.db.Exec(`
		UPDATE extraction_tasks
		SET status = ?, completed_at = ?
		WHERE task_id = ?
	`, TaskStatusCompleted, now, taskID)

	if err != nil {
		return fmt.Errorf("failed to complete task %s: %w", taskID, err)
	}

	return nil
}

// FailTask marks a task as failed
func (s *SQLiteJobControl) FailTask(taskID string, errorMsg string) error {
	_, err := s.db.Exec(`
		UPDATE extraction_tasks
		SET status = ?, error = ?
		WHERE task_id = ?
	`, TaskStatusFailed, errorMsg, taskID)

	if err != nil {
		return fmt.Errorf("failed to fail task %s: %w", taskID, err)
	}

	return nil
}

// ReleaseTask releases a claimed task back to pending state
func (s *SQLiteJobControl) ReleaseTask(taskID string) error {
	_, err := s.db.Exec(`
		UPDATE extraction_tasks
		SET status = ?, claimed_by = NULL, claimed_at = NULL
		WHERE task_id = ?
	`, TaskStatusPending, taskID)

	if err != nil {
		return fmt.Errorf("failed to release task %s: %w", taskID, err)
	}

	return nil
}

// GetTaskStatus returns count of tasks by status for a run
func (s *SQLiteJobControl) GetTaskStatus(runID string) (map[ExtractionTaskStatus]int, error) {
	rows, err := s.db.Query(`
		SELECT status, COUNT(*)
		FROM extraction_tasks
		WHERE run_id = ?
		GROUP BY status
	`, runID)

	if err != nil {
		return nil, fmt.Errorf("failed to query task status: %w", err)
	}
	defer rows.Close()

	counts := make(map[ExtractionTaskStatus]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("failed to scan status: %w", err)
		}
		counts[ExtractionTaskStatus(status)] = count
	}

	return counts, nil
}

// GetPendingTaskCount returns the number of pending tasks for a specific type
func (s *SQLiteJobControl) GetPendingTaskCount(runID string, taskType ExtractionTaskType) (int, error) {
	var count int
	err := s.db.QueryRow(`
		SELECT COUNT(*)
		FROM extraction_tasks
		WHERE run_id = ? AND type = ? AND status = ?
	`, runID, taskType, TaskStatusPending).Scan(&count)

	if err != nil {
		return 0, fmt.Errorf("failed to query pending task count: %w", err)
	}

	return count, nil
}

// IsPhaseComplete checks if all tasks of a specific type are completed
func (s *SQLiteJobControl) IsPhaseComplete(runID string, taskType ExtractionTaskType) (bool, error) {
	var incompleteCount int
	err := s.db.QueryRow(`
		SELECT COUNT(*)
		FROM extraction_tasks
		WHERE run_id = ? AND type = ? AND status NOT IN (?, ?)
	`, runID, taskType, TaskStatusCompleted, TaskStatusFailed).Scan(&incompleteCount)

	if err != nil {
		return false, fmt.Errorf("failed to check phase completion: %w", err)
	}

	return incompleteCount == 0, nil
}

// CleanupRun removes all tasks and leader records for a run
func (s *SQLiteJobControl) CleanupRun(runID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Remove tasks
	_, err = tx.Exec("DELETE FROM extraction_tasks WHERE run_id = ?", runID)
	if err != nil {
		return fmt.Errorf("failed to delete tasks: %w", err)
	}

	// Remove leader record
	_, err = tx.Exec("DELETE FROM run_leaders WHERE run_id = ?", runID)
	if err != nil {
		return fmt.Errorf("failed to delete leader record: %w", err)
	}

	return tx.Commit()
}

// Close closes the database connection
func (s *SQLiteJobControl) Close() error {
	return s.db.Close()
}
