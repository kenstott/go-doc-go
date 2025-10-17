package jobcontrol

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteJobControl implements JobControl using SQLite
type SQLiteJobControl struct {
	db               *sql.DB
	dbPath           string
	claimTimeout     int // seconds
	heartbeatInterval int // seconds
	maxRetries       int
}

// Config holds configuration for job control
type Config struct {
	Backend           string // "sqlite" or "postgres"
	Path              string // For SQLite: file path; For PostgreSQL: connection string (DSN)
	ClaimTimeout      int
	HeartbeatInterval int
	MaxRetries        int
}

// NewSQLiteJobControl creates a new SQLite job control database
func NewSQLiteJobControl(config Config) (*SQLiteJobControl, error) {
	// Set defaults
	if config.ClaimTimeout == 0 {
		config.ClaimTimeout = 300 // 5 minutes
	}
	if config.HeartbeatInterval == 0 {
		config.HeartbeatInterval = 30
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}

	// Ensure directory exists (but not the database file itself)
	dir := filepath.Dir(config.Path)
	if dir != "." && dir != "/" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory: %w", err)
		}
	}

	// Remove if path exists as directory (common error)
	if info, err := os.Stat(config.Path); err == nil && info.IsDir() {
		return nil, fmt.Errorf("database path exists as directory: %s", config.Path)
	}

	// Open database
	db, err := sql.Open("sqlite3", config.Path+"?_busy_timeout=30000&_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	jc := &SQLiteJobControl{
		db:               db,
		dbPath:           config.Path,
		claimTimeout:     config.ClaimTimeout,
		heartbeatInterval: config.HeartbeatInterval,
		maxRetries:       config.MaxRetries,
	}

	// Initialize schema
	if err := jc.initializeSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return jc, nil
}

// initializeSchema creates database tables if they don't exist
func (jc *SQLiteJobControl) initializeSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS document_queue (
		doc_id TEXT PRIMARY KEY,
		source TEXT NOT NULL,
		metadata TEXT,
		status TEXT DEFAULT 'pending',
		claimed_by TEXT,
		claimed_at TIMESTAMP,
		completed_at TIMESTAMP,
		retry_count INTEGER DEFAULT 0,
		error_message TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS workers (
		worker_id TEXT PRIMARY KEY,
		info TEXT,
		last_heartbeat TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		registered_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS leader (
		worker_id TEXT PRIMARY KEY,
		info TEXT,
		elected_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		last_heartbeat TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS source_leaders (
		source_name TEXT PRIMARY KEY,
		worker_id TEXT NOT NULL,
		info TEXT,
		elected_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		last_heartbeat TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS document_metadata (
		doc_id TEXT PRIMARY KEY,
		source TEXT NOT NULL,
		last_modified TIMESTAMP,
		content_hash TEXT,
		file_size INTEGER,
		processing_stats TEXT,
		last_processed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_queue_status ON document_queue(status);
	CREATE INDEX IF NOT EXISTS idx_queue_claimed ON document_queue(claimed_by, status);
	CREATE INDEX IF NOT EXISTS idx_metadata_source ON document_metadata(source);
	`

	_, err := jc.db.Exec(schema)
	return err
}

// EnqueueDocument adds a document to the processing queue
// Only resets documents that are in 'failed' status or don't exist
// NEVER resets completed, pending, or processing documents
func (jc *SQLiteJobControl) EnqueueDocument(docID, source string, metadata map[string]interface{}) error {
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// First check if document exists and what status it has
	var existingStatus string
	err = jc.db.QueryRow(`SELECT status FROM document_queue WHERE doc_id = ?`, docID).Scan(&existingStatus)

	if err == nil {
		// Document exists - only re-enqueue if it's failed
		if existingStatus == "completed" || existingStatus == "pending" || existingStatus == "processing" {
			// Don't re-enqueue - document is already completed or being processed
			return nil
		}
		// Status is 'failed' - allow re-queue by continuing
	} else if err != sql.ErrNoRows {
		return fmt.Errorf("failed to check document status: %w", err)
	}
	// If sql.ErrNoRows, document doesn't exist - proceed to insert

	query := `
		INSERT INTO document_queue (doc_id, source, metadata, status, created_at)
		VALUES (?, ?, ?, 'pending', CURRENT_TIMESTAMP)
		ON CONFLICT(doc_id) DO UPDATE SET
			source = excluded.source,
			metadata = excluded.metadata,
			status = 'pending',
			claimed_by = NULL,
			claimed_at = NULL,
			retry_count = 0,
			error_message = NULL
		WHERE document_queue.status = 'failed'
	`

	result, err := jc.db.Exec(query, docID, source, string(metadataJSON))
	if err != nil {
		return fmt.Errorf("failed to enqueue document: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		log.Printf("Enqueued document: %s from source: %s", docID, source)
	}
	return nil
}

// ClaimNextDocument atomically claims the next available document
func (jc *SQLiteJobControl) ClaimNextDocument(workerID string) (*DocumentInfo, error) {
	tx, err := jc.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Cleanup stale claims
	staleThreshold := time.Now().Add(-time.Duration(jc.claimTimeout) * time.Second)
	_, err = tx.Exec(`
		UPDATE document_queue
		SET status = 'pending', claimed_by = NULL, claimed_at = NULL
		WHERE status = 'processing' AND claimed_at < ?
	`, staleThreshold)
	if err != nil {
		return nil, fmt.Errorf("failed to cleanup stale claims: %w", err)
	}

	// Find and claim next document using random sampling for diversity
	query := `
		SELECT doc_id, source, metadata, retry_count, created_at
		FROM document_queue
		WHERE status = 'pending' AND retry_count < ?
		ORDER BY RANDOM()
		LIMIT 1
	`

	var docID, source, metadataJSON string
	var retryCount int
	var createdAt time.Time

	err = tx.QueryRow(query, jc.maxRetries).Scan(&docID, &source, &metadataJSON, &retryCount, &createdAt)
	if err == sql.ErrNoRows {
		return nil, nil // No documents available
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find document: %w", err)
	}

	// Claim the document
	now := time.Now()
	_, err = tx.Exec(`
		UPDATE document_queue
		SET status = 'processing', claimed_by = ?, claimed_at = ?
		WHERE doc_id = ? AND status = 'pending'
	`, workerID, now, docID)
	if err != nil {
		return nil, fmt.Errorf("failed to claim document: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Parse metadata
	var metadata map[string]interface{}
	if err := json.Unmarshal([]byte(metadataJSON), &metadata); err != nil {
		log.Printf("Warning: failed to unmarshal metadata for %s: %v", docID, err)
		metadata = make(map[string]interface{})
	}

	return &DocumentInfo{
		DocID:      docID,
		Source:     source,
		Metadata:   metadata,
		Status:     StatusProcessing,
		ClaimedBy:  workerID,
		ClaimedAt:  &now,
		RetryCount: retryCount,
		CreatedAt:  createdAt,
	}, nil
}

// CompleteDocument marks a document as completed or failed
func (jc *SQLiteJobControl) CompleteDocument(docID, workerID string, success bool, errorMsg string) error {
	now := time.Now()

	if success {
		_, err := jc.db.Exec(`
			UPDATE document_queue
			SET status = 'completed', completed_at = ?
			WHERE doc_id = ? AND claimed_by = ?
		`, now, docID, workerID)
		return err
	}

	// Failed - increment retry count
	_, err := jc.db.Exec(`
		UPDATE document_queue
		SET status = 'failed', retry_count = retry_count + 1, error_message = ?, completed_at = ?
		WHERE doc_id = ? AND claimed_by = ?
	`, errorMsg, now, docID, workerID)
	return err
}

// IsDocumentQueued checks if a document is already in the queue
// Returns true for pending, processing, or completed documents
// Only returns false for documents that don't exist or have failed
func (jc *SQLiteJobControl) IsDocumentQueued(docID string) (bool, error) {
	var count int
	err := jc.db.QueryRow(`
		SELECT COUNT(*) FROM document_queue
		WHERE doc_id = ? AND status IN ('pending', 'processing', 'completed')
	`, docID).Scan(&count)
	return count > 0, err
}

// RegisterWorker registers a worker in the system
func (jc *SQLiteJobControl) RegisterWorker(workerID string, info map[string]interface{}) error {
	infoJSON, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("failed to marshal worker info: %w", err)
	}

	_, err = jc.db.Exec(`
		INSERT INTO workers (worker_id, info, last_heartbeat, registered_at)
		VALUES (?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT(worker_id) DO UPDATE SET
			info = excluded.info,
			last_heartbeat = CURRENT_TIMESTAMP
	`, workerID, string(infoJSON))

	return err
}

// UpdateWorkerHeartbeat updates the worker's last heartbeat time
func (jc *SQLiteJobControl) UpdateWorkerHeartbeat(workerID string) error {
	_, err := jc.db.Exec(`
		UPDATE workers SET last_heartbeat = CURRENT_TIMESTAMP
		WHERE worker_id = ?
	`, workerID)
	return err
}

// UpdateDocumentClaimHeartbeat updates the document's claimed_at timestamp to prevent claim timeout
// This is called periodically during document processing to keep the claim alive
func (jc *SQLiteJobControl) UpdateDocumentClaimHeartbeat(docID, workerID string) error {
	_, err := jc.db.Exec(`
		UPDATE document_queue
		SET claimed_at = CURRENT_TIMESTAMP
		WHERE doc_id = ? AND claimed_by = ? AND status = 'processing'
	`, docID, workerID)
	return err
}

// GetActiveWorkers returns list of active workers
func (jc *SQLiteJobControl) GetActiveWorkers() ([]*WorkerInfo, error) {
	timeout := time.Now().Add(-time.Duration(jc.heartbeatInterval*3) * time.Second)

	rows, err := jc.db.Query(`
		SELECT worker_id, info, last_heartbeat, registered_at
		FROM workers
		WHERE last_heartbeat > ?
	`, timeout)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workers []*WorkerInfo
	for rows.Next() {
		var workerID, infoJSON string
		var lastHeartbeat, registeredAt time.Time

		if err := rows.Scan(&workerID, &infoJSON, &lastHeartbeat, &registeredAt); err != nil {
			return nil, err
		}

		var info map[string]interface{}
		if err := json.Unmarshal([]byte(infoJSON), &info); err != nil {
			log.Printf("Warning: failed to unmarshal worker info: %v", err)
			info = make(map[string]interface{})
		}

		workers = append(workers, &WorkerInfo{
			WorkerID:      workerID,
			Info:          info,
			LastHeartbeat: lastHeartbeat,
			RegisteredAt:  registeredAt,
		})
	}

	return workers, nil
}

// ElectLeader attempts to elect this worker as leader
func (jc *SQLiteJobControl) ElectLeader(workerID string, info map[string]interface{}) (bool, error) {
	infoJSON, err := json.Marshal(info)
	if err != nil {
		return false, fmt.Errorf("failed to marshal leader info: %w", err)
	}

	tx, err := jc.db.Begin()
	if err != nil {
		return false, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Check if there's a current leader
	var currentLeader string
	var lastHeartbeat time.Time
	err = tx.QueryRow(`SELECT worker_id, last_heartbeat FROM leader LIMIT 1`).Scan(&currentLeader, &lastHeartbeat)

	if err == sql.ErrNoRows {
		// No leader exists, claim leadership
		_, err := tx.Exec(`
			INSERT INTO leader (worker_id, info, elected_at, last_heartbeat)
			VALUES (?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		`, workerID, string(infoJSON))
		if err != nil {
			return false, err
		}

		if err := tx.Commit(); err != nil {
			return false, err
		}
		return true, nil
	}

	if err != nil {
		return false, err
	}

	// Check if current leader is stale
	timeout := time.Now().Add(-time.Duration(jc.heartbeatInterval*3) * time.Second)
	if lastHeartbeat.Before(timeout) {
		// Current leader is stale, take over
		_, err := tx.Exec(`
			UPDATE leader
			SET worker_id = ?, info = ?, elected_at = CURRENT_TIMESTAMP, last_heartbeat = CURRENT_TIMESTAMP
		`, workerID, string(infoJSON))
		if err != nil {
			return false, err
		}

		if err := tx.Commit(); err != nil {
			return false, err
		}
		return true, nil
	}

	// Leader exists and is active
	return false, nil
}

// GetCurrentLeader returns the current leader information
func (jc *SQLiteJobControl) GetCurrentLeader() (*LeaderInfo, error) {
	var workerID, infoJSON string
	var electedAt, lastHeartbeat time.Time

	err := jc.db.QueryRow(`
		SELECT worker_id, info, elected_at, last_heartbeat
		FROM leader
		LIMIT 1
	`).Scan(&workerID, &infoJSON, &electedAt, &lastHeartbeat)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Check if leader is stale
	timeout := time.Now().Add(-time.Duration(jc.heartbeatInterval*3) * time.Second)
	if lastHeartbeat.Before(timeout) {
		return nil, nil // Leader is stale
	}

	var info map[string]interface{}
	if err := json.Unmarshal([]byte(infoJSON), &info); err != nil {
		log.Printf("Warning: failed to unmarshal leader info: %v", err)
		info = make(map[string]interface{})
	}

	return &LeaderInfo{
		WorkerID:      workerID,
		Info:          info,
		ElectedAt:     electedAt,
		LastHeartbeat: lastHeartbeat,
	}, nil
}

// UpdateLeaderHeartbeat updates the leader's heartbeat
func (jc *SQLiteJobControl) UpdateLeaderHeartbeat(workerID string) error {
	_, err := jc.db.Exec(`
		UPDATE leader
		SET last_heartbeat = CURRENT_TIMESTAMP
		WHERE worker_id = ?
	`, workerID)
	return err
}

// ReleaseLeadership releases leadership for this worker
func (jc *SQLiteJobControl) ReleaseLeadership(workerID string) error {
	_, err := jc.db.Exec(`
		DELETE FROM leader WHERE worker_id = ?
	`, workerID)
	return err
}

// ElectSourceLeader attempts to become leader for a specific content source
// Returns true if successfully elected, false if another worker is already leader
func (jc *SQLiteJobControl) ElectSourceLeader(sourceName, workerID, info string) (bool, error) {
	// First, clean up stale leaders (heartbeat older than claim timeout)
	// This allows new workers to take over from crashed/stopped workers
	staleThreshold := time.Now().Add(-time.Duration(jc.claimTimeout) * time.Second)
	result, err := jc.db.Exec(`
		DELETE FROM source_leaders
		WHERE source_name = ?
		AND last_heartbeat < ?
	`, sourceName, staleThreshold)

	if err != nil {
		return false, fmt.Errorf("failed to cleanup stale source leaders: %w", err)
	}

	rowsDeleted, _ := result.RowsAffected()
	if rowsDeleted > 0 {
		log.Printf("Cleaned up %d stale source leaders for %s", rowsDeleted, sourceName)
	}

	// Try to insert this worker as leader for this source
	_, err = jc.db.Exec(`
		INSERT INTO source_leaders (source_name, worker_id, info, elected_at, last_heartbeat)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT(source_name) DO NOTHING
	`, sourceName, workerID, info)

	if err != nil {
		log.Printf("ERROR: Failed to insert source leader for %s: %v", sourceName, err)
		return false, err
	}

	// Check if we are the leader for this source
	var currentLeaderID string
	err = jc.db.QueryRow(`
		SELECT worker_id FROM source_leaders WHERE source_name = ?
	`, sourceName).Scan(&currentLeaderID)

	if err != nil {
		log.Printf("ERROR: Failed to query source leader for %s: %v", sourceName, err)
		return false, err
	}

	isLeader := currentLeaderID == workerID
	log.Printf("Source leader election for %s: current=%s, candidate=%s, elected=%v", sourceName, currentLeaderID, workerID, isLeader)
	return isLeader, nil
}

// GetSourceLeader returns the current leader for a specific content source
func (jc *SQLiteJobControl) GetSourceLeader(sourceName string) (string, error) {
	var leaderID string
	err := jc.db.QueryRow(`
		SELECT worker_id FROM source_leaders WHERE source_name = ?
	`, sourceName).Scan(&leaderID)

	if err == sql.ErrNoRows {
		return "", nil // No leader for this source
	}
	return leaderID, err
}

// ReleaseSourceLeadership releases leadership for this worker for a specific source
func (jc *SQLiteJobControl) ReleaseSourceLeadership(sourceName, workerID string) error {
	_, err := jc.db.Exec(`
		DELETE FROM source_leaders WHERE source_name = ? AND worker_id = ?
	`, sourceName, workerID)
	return err
}

// UpdateSourceLeaderHeartbeat updates the heartbeat timestamp for source leader
func (jc *SQLiteJobControl) UpdateSourceLeaderHeartbeat(sourceName, workerID string) error {
	_, err := jc.db.Exec(`
		UPDATE source_leaders
		SET last_heartbeat = CURRENT_TIMESTAMP
		WHERE source_name = ? AND worker_id = ?
	`, sourceName, workerID)
	return err
}

// HasDocumentChanged checks if a document has changed since last processing
func (jc *SQLiteJobControl) HasDocumentChanged(docID, source string, currentModified *time.Time, currentHash string) (bool, error) {
	var lastModified *time.Time
	var storedHash string

	err := jc.db.QueryRow(`
		SELECT last_modified, content_hash
		FROM document_metadata
		WHERE doc_id = ?
	`, docID).Scan(&lastModified, &storedHash)

	if err == sql.ErrNoRows {
		// Document never processed before
		return true, nil
	}
	if err != nil {
		return false, err
	}

	// Check hash if available
	if currentHash != "" && storedHash != "" {
		return currentHash != storedHash, nil
	}

	// Check modification time
	if currentModified != nil && lastModified != nil {
		return currentModified.After(*lastModified), nil
	}

	// Can't determine, assume changed
	return true, nil
}

// StoreDocumentMetadata stores metadata for change tracking
func (jc *SQLiteJobControl) StoreDocumentMetadata(docID, source string, lastModified *time.Time, contentHash string, fileSize *int64, processingStats map[string]interface{}) error {
	statsJSON, err := json.Marshal(processingStats)
	if err != nil {
		return fmt.Errorf("failed to marshal processing stats: %w", err)
	}

	_, err = jc.db.Exec(`
		INSERT INTO document_metadata (doc_id, source, last_modified, content_hash, file_size, processing_stats, last_processed_at)
		VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(doc_id) DO UPDATE SET
			source = excluded.source,
			last_modified = excluded.last_modified,
			content_hash = excluded.content_hash,
			file_size = excluded.file_size,
			processing_stats = excluded.processing_stats,
			last_processed_at = CURRENT_TIMESTAMP
	`, docID, source, lastModified, contentHash, fileSize, string(statsJSON))

	return err
}

// GetDocumentMetadata retrieves stored metadata for a document
func (jc *SQLiteJobControl) GetDocumentMetadata(docID string) (*DocumentMetadata, error) {
	var source, contentHash string
	var statsJSON string
	var lastModified *time.Time
	var fileSize *int64
	var lastProcessedAt time.Time

	err := jc.db.QueryRow(`
		SELECT source, last_modified, content_hash, file_size, processing_stats, last_processed_at
		FROM document_metadata
		WHERE doc_id = ?
	`, docID).Scan(&source, &lastModified, &contentHash, &fileSize, &statsJSON, &lastProcessedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var stats map[string]interface{}
	if err := json.Unmarshal([]byte(statsJSON), &stats); err != nil {
		log.Printf("Warning: failed to unmarshal processing stats: %v", err)
		stats = make(map[string]interface{})
	}

	return &DocumentMetadata{
		DocID:           docID,
		Source:          source,
		LastModified:    lastModified,
		ContentHash:     contentHash,
		FileSize:        fileSize,
		ProcessingStats: stats,
		LastProcessedAt: lastProcessedAt,
	}, nil
}

// GetProcessingStatus returns overall processing status
func (jc *SQLiteJobControl) GetProcessingStatus() (*ProcessingStatus, error) {
	// Count documents by status
	var pending, processing, completed, failed int
	err := jc.db.QueryRow(`
		SELECT
			SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END),
			SUM(CASE WHEN status = 'processing' THEN 1 ELSE 0 END),
			SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END),
			SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END)
		FROM document_queue
	`).Scan(&pending, &processing, &completed, &failed)
	if err != nil {
		return nil, err
	}

	// Count workers
	workers, err := jc.GetActiveWorkers()
	if err != nil {
		return nil, err
	}

	// Get leader
	leader, err := jc.GetCurrentLeader()
	if err != nil {
		return nil, err
	}

	queueStatus := "active"
	if pending == 0 && processing == 0 {
		queueStatus = "empty"
	}

	return &ProcessingStatus{
		QueueStatus: queueStatus,
		Documents: map[string]int{
			"pending":    pending,
			"processing": processing,
			"completed":  completed,
			"failed":     failed,
		},
		Workers: map[string]int{
			"active": len(workers),
		},
		Leader: leader,
	}, nil
}

// CleanupStaleWorkers removes workers that haven't sent heartbeat
func (jc *SQLiteJobControl) CleanupStaleWorkers(timeout int) (int, error) {
	threshold := time.Now().Add(-time.Duration(timeout) * time.Second)
	result, err := jc.db.Exec(`
		DELETE FROM workers WHERE last_heartbeat < ?
	`, threshold)
	if err != nil {
		return 0, err
	}

	count, err := result.RowsAffected()
	return int(count), err
}

// CleanupStaleLeader removes stale leader
func (jc *SQLiteJobControl) CleanupStaleLeader(timeout int) (bool, error) {
	threshold := time.Now().Add(-time.Duration(timeout) * time.Second)
	result, err := jc.db.Exec(`
		DELETE FROM leader WHERE last_heartbeat < ?
	`, threshold)
	if err != nil {
		return false, err
	}

	count, err := result.RowsAffected()
	return count > 0, err
}

// Close closes the database connection
func (jc *SQLiteJobControl) Close() error {
	return jc.db.Close()
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
