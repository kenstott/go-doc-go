package jobcontrol

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
)

// PostgreSQLJobControl implements JobControl using PostgreSQL
type PostgreSQLJobControl struct {
	db                *sql.DB
	claimTimeout      int // seconds
	heartbeatInterval int // seconds
	maxRetries        int
}

// NewPostgreSQLJobControl creates a new PostgreSQL job control database
func NewPostgreSQLJobControl(config Config) (*PostgreSQLJobControl, error) {
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

	// Build connection string from config.Path (assuming it contains the DSN)
	// Example: "postgres://user:password@localhost/dbname?sslmode=disable"
	log.Printf("DEBUG: PostgreSQL connection string: %s", config.Path)
	db, err := sql.Open("postgres", config.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(time.Hour)

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	jc := &PostgreSQLJobControl{
		db:                db,
		claimTimeout:      config.ClaimTimeout,
		heartbeatInterval: config.HeartbeatInterval,
		maxRetries:        config.MaxRetries,
	}

	// Initialize schema
	if err := jc.initializeSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return jc, nil
}

// initializeSchema creates database tables if they don't exist
func (jc *PostgreSQLJobControl) initializeSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS document_queue (
		doc_id TEXT PRIMARY KEY,
		source TEXT NOT NULL,
		metadata JSONB,
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
		info JSONB,
		last_heartbeat TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		registered_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS leader (
		worker_id TEXT PRIMARY KEY,
		info JSONB,
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
		file_size BIGINT,
		processing_stats JSONB,
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
func (jc *PostgreSQLJobControl) EnqueueDocument(docID, source string, metadata map[string]interface{}) error {
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// First check if document exists and what status it has
	var existingStatus string
	err = jc.db.QueryRow(`SELECT status FROM document_queue WHERE doc_id = $1`, docID).Scan(&existingStatus)

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
		VALUES ($1, $2, $3, 'pending', CURRENT_TIMESTAMP)
		ON CONFLICT(doc_id) DO UPDATE SET
			source = EXCLUDED.source,
			metadata = EXCLUDED.metadata,
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
func (jc *PostgreSQLJobControl) ClaimNextDocument(workerID string) (*DocumentInfo, error) {
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
		WHERE status = 'processing' AND claimed_at < $1
	`, staleThreshold)
	if err != nil {
		return nil, fmt.Errorf("failed to cleanup stale claims: %w", err)
	}

	// Find and claim next document using FOR UPDATE SKIP LOCKED for better concurrency
	query := `
		UPDATE document_queue
		SET status = 'processing', claimed_by = $1, claimed_at = $2
		WHERE doc_id = (
			SELECT doc_id FROM document_queue
			WHERE status = 'pending' AND retry_count < $3
			ORDER BY created_at ASC
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		)
		RETURNING doc_id, source, metadata, retry_count, created_at
	`

	now := time.Now()
	var docID, source, metadataJSON string
	var retryCount int
	var createdAt time.Time

	err = tx.QueryRow(query, workerID, now, jc.maxRetries).Scan(&docID, &source, &metadataJSON, &retryCount, &createdAt)
	if err == sql.ErrNoRows {
		return nil, nil // No documents available
	}
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
func (jc *PostgreSQLJobControl) CompleteDocument(docID, workerID string, success bool, errorMsg string) error {
	now := time.Now()

	if success {
		_, err := jc.db.Exec(`
			UPDATE document_queue
			SET status = 'completed', completed_at = $1
			WHERE doc_id = $2 AND claimed_by = $3
		`, now, docID, workerID)
		return err
	}

	// Failed - increment retry count
	_, err := jc.db.Exec(`
		UPDATE document_queue
		SET status = 'failed', retry_count = retry_count + 1, error_message = $1, completed_at = $2
		WHERE doc_id = $3 AND claimed_by = $4
	`, errorMsg, now, docID, workerID)
	return err
}

// IsDocumentQueued checks if a document is already in the queue
func (jc *PostgreSQLJobControl) IsDocumentQueued(docID string) (bool, error) {
	var count int
	err := jc.db.QueryRow(`
		SELECT COUNT(*) FROM document_queue
		WHERE doc_id = $1 AND status IN ('pending', 'processing', 'completed')
	`, docID).Scan(&count)
	return count > 0, err
}

// RegisterWorker registers a worker in the system
func (jc *PostgreSQLJobControl) RegisterWorker(workerID string, info map[string]interface{}) error {
	infoJSON, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("failed to marshal worker info: %w", err)
	}

	_, err = jc.db.Exec(`
		INSERT INTO workers (worker_id, info, last_heartbeat, registered_at)
		VALUES ($1, $2, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT(worker_id) DO UPDATE SET
			info = EXCLUDED.info,
			last_heartbeat = CURRENT_TIMESTAMP
	`, workerID, string(infoJSON))

	return err
}

// UpdateWorkerHeartbeat updates the worker's last heartbeat time
func (jc *PostgreSQLJobControl) UpdateWorkerHeartbeat(workerID string) error {
	_, err := jc.db.Exec(`
		UPDATE workers SET last_heartbeat = CURRENT_TIMESTAMP
		WHERE worker_id = $1
	`, workerID)
	return err
}

// GetActiveWorkers returns list of active workers
func (jc *PostgreSQLJobControl) GetActiveWorkers() ([]*WorkerInfo, error) {
	timeout := time.Now().Add(-time.Duration(jc.heartbeatInterval*3) * time.Second)

	rows, err := jc.db.Query(`
		SELECT worker_id, info, last_heartbeat, registered_at
		FROM workers
		WHERE last_heartbeat > $1
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
func (jc *PostgreSQLJobControl) ElectLeader(workerID string, info map[string]interface{}) (bool, error) {
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
			VALUES ($1, $2, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
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
			SET worker_id = $1, info = $2, elected_at = CURRENT_TIMESTAMP, last_heartbeat = CURRENT_TIMESTAMP
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
func (jc *PostgreSQLJobControl) GetCurrentLeader() (*LeaderInfo, error) {
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
func (jc *PostgreSQLJobControl) UpdateLeaderHeartbeat(workerID string) error {
	_, err := jc.db.Exec(`
		UPDATE leader
		SET last_heartbeat = CURRENT_TIMESTAMP
		WHERE worker_id = $1
	`, workerID)
	return err
}

// ReleaseLeadership releases leadership for this worker
func (jc *PostgreSQLJobControl) ReleaseLeadership(workerID string) error {
	_, err := jc.db.Exec(`
		DELETE FROM leader WHERE worker_id = $1
	`, workerID)
	return err
}

// ElectSourceLeader attempts to become leader for a specific content source
func (jc *PostgreSQLJobControl) ElectSourceLeader(sourceName, workerID, info string) (bool, error) {
	// First, clean up stale leaders (heartbeat older than claim timeout)
	staleThreshold := time.Now().Add(-time.Duration(jc.claimTimeout) * time.Second)
	result, err := jc.db.Exec(`
		DELETE FROM source_leaders
		WHERE source_name = $1
		AND last_heartbeat < $2
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
		VALUES ($1, $2, $3, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT(source_name) DO NOTHING
	`, sourceName, workerID, info)

	if err != nil {
		log.Printf("ERROR: Failed to insert source leader for %s: %v", sourceName, err)
		return false, err
	}

	// Check if we are the leader for this source
	var currentLeaderID string
	err = jc.db.QueryRow(`
		SELECT worker_id FROM source_leaders WHERE source_name = $1
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
func (jc *PostgreSQLJobControl) GetSourceLeader(sourceName string) (string, error) {
	var leaderID string
	err := jc.db.QueryRow(`
		SELECT worker_id FROM source_leaders WHERE source_name = $1
	`, sourceName).Scan(&leaderID)

	if err == sql.ErrNoRows {
		return "", nil // No leader for this source
	}
	return leaderID, err
}

// ReleaseSourceLeadership releases leadership for this worker for a specific source
func (jc *PostgreSQLJobControl) ReleaseSourceLeadership(sourceName, workerID string) error {
	_, err := jc.db.Exec(`
		DELETE FROM source_leaders WHERE source_name = $1 AND worker_id = $2
	`, sourceName, workerID)
	return err
}

// UpdateSourceLeaderHeartbeat updates the heartbeat timestamp for source leader
func (jc *PostgreSQLJobControl) UpdateSourceLeaderHeartbeat(sourceName, workerID string) error {
	_, err := jc.db.Exec(`
		UPDATE source_leaders
		SET last_heartbeat = CURRENT_TIMESTAMP
		WHERE source_name = $1 AND worker_id = $2
	`, sourceName, workerID)
	return err
}

// HasDocumentChanged checks if a document has changed since last processing
func (jc *PostgreSQLJobControl) HasDocumentChanged(docID, source string, currentModified *time.Time, currentHash string) (bool, error) {
	var lastModified *time.Time
	var storedHash string

	err := jc.db.QueryRow(`
		SELECT last_modified, content_hash
		FROM document_metadata
		WHERE doc_id = $1
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
func (jc *PostgreSQLJobControl) StoreDocumentMetadata(docID, source string, lastModified *time.Time, contentHash string, fileSize *int64, processingStats map[string]interface{}) error {
	statsJSON, err := json.Marshal(processingStats)
	if err != nil {
		return fmt.Errorf("failed to marshal processing stats: %w", err)
	}

	_, err = jc.db.Exec(`
		INSERT INTO document_metadata (doc_id, source, last_modified, content_hash, file_size, processing_stats, last_processed_at)
		VALUES ($1, $2, $3, $4, $5, $6, CURRENT_TIMESTAMP)
		ON CONFLICT(doc_id) DO UPDATE SET
			source = EXCLUDED.source,
			last_modified = EXCLUDED.last_modified,
			content_hash = EXCLUDED.content_hash,
			file_size = EXCLUDED.file_size,
			processing_stats = EXCLUDED.processing_stats,
			last_processed_at = CURRENT_TIMESTAMP
	`, docID, source, lastModified, contentHash, fileSize, string(statsJSON))

	return err
}

// GetDocumentMetadata retrieves stored metadata for a document
func (jc *PostgreSQLJobControl) GetDocumentMetadata(docID string) (*DocumentMetadata, error) {
	var source, contentHash string
	var statsJSON string
	var lastModified *time.Time
	var fileSize *int64
	var lastProcessedAt time.Time

	err := jc.db.QueryRow(`
		SELECT source, last_modified, content_hash, file_size, processing_stats, last_processed_at
		FROM document_metadata
		WHERE doc_id = $1
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
func (jc *PostgreSQLJobControl) GetProcessingStatus() (*ProcessingStatus, error) {
	// Count documents by status
	var pending, processing, completed, failed int
	err := jc.db.QueryRow(`
		SELECT
			SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END)::int,
			SUM(CASE WHEN status = 'processing' THEN 1 ELSE 0 END)::int,
			SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END)::int,
			SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END)::int
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
func (jc *PostgreSQLJobControl) CleanupStaleWorkers(timeout int) (int, error) {
	threshold := time.Now().Add(-time.Duration(timeout) * time.Second)
	result, err := jc.db.Exec(`
		DELETE FROM workers WHERE last_heartbeat < $1
	`, threshold)
	if err != nil {
		return 0, err
	}

	count, err := result.RowsAffected()
	return int(count), err
}

// CleanupStaleLeader removes stale leader
func (jc *PostgreSQLJobControl) CleanupStaleLeader(timeout int) (bool, error) {
	threshold := time.Now().Add(-time.Duration(timeout) * time.Second)
	result, err := jc.db.Exec(`
		DELETE FROM leader WHERE last_heartbeat < $1
	`, threshold)
	if err != nil {
		return false, err
	}

	count, err := result.RowsAffected()
	return count > 0, err
}

// Close closes the database connection
func (jc *PostgreSQLJobControl) Close() error {
	return jc.db.Close()
}
