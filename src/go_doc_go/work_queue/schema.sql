-- Document Processing Queue Schema
-- This schema supports distributed document processing with config-based run coordination

-- Processing runs table - tracks coordinated processing batches
CREATE TABLE IF NOT EXISTS processing_runs (
    run_id VARCHAR(16) PRIMARY KEY,  -- Config hash (first 16 chars of SHA256)
    config_hash VARCHAR(64) NOT NULL,  -- Full SHA256 hash for verification
    config_snapshot JSONB NOT NULL,  -- Complete config at run start
    
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    -- Status values: 'active', 'processing_complete', 'post_processing', 'completed', 'failed', 'abandoned'
    
    -- Timing
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    first_worker_at TIMESTAMP,
    last_activity_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    processing_completed_at TIMESTAMP,
    post_processing_started_at TIMESTAMP,
    post_processing_completed_at TIMESTAMP,
    completed_at TIMESTAMP,
    
    -- Statistics
    worker_count INTEGER DEFAULT 0,
    documents_queued INTEGER DEFAULT 0,
    documents_processed INTEGER DEFAULT 0,
    documents_failed INTEGER DEFAULT 0,
    documents_retried INTEGER DEFAULT 0,
    
    -- Leader election
    leader_worker_id VARCHAR(100),
    leader_elected_at TIMESTAMP,
    leader_heartbeat TIMESTAMP,
    leader_lease_expires TIMESTAMP,
    
    -- Post-processing coordination (handled by leader)
    post_processor_worker_id VARCHAR(100),
    post_processing_lock_acquired_at TIMESTAMP,
    
    -- Metadata
    metadata JSONB,
    
    -- Ensure config hash uniqueness (though collision unlikely with SHA256)
    UNIQUE(config_hash)
);

-- Document queue table - the main work queue
CREATE TABLE IF NOT EXISTS document_queue (
    queue_id SERIAL PRIMARY KEY,
    
    -- Document identification
    doc_id VARCHAR(500) NOT NULL,
    source_name VARCHAR(255) NOT NULL,
    source_type VARCHAR(50) NOT NULL DEFAULT 'configured',
    -- source_type values: 'configured', 'linked', 'discovered'
    
    -- Run coordination
    run_id VARCHAR(16) NOT NULL REFERENCES processing_runs(run_id),
    
    -- Processing status
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    -- Status values: 'pending', 'processing', 'completed', 'failed', 'retry'
    
    -- Worker assignment
    worker_id VARCHAR(100),
    claimed_at TIMESTAMP,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    failed_at TIMESTAMP,
    
    -- Retry handling
    retry_count INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3,
    error_message TEXT,
    error_details JSONB,
    
    -- Link traversal
    parent_doc_id VARCHAR(500),  -- Document that linked to this one
    link_depth INTEGER DEFAULT 0,  -- How deep in the link chain
    max_link_depth INTEGER DEFAULT 3,  -- Max depth to traverse
    
    -- Change detection
    content_hash VARCHAR(64),
    last_modified TIMESTAMP,
    file_size BIGINT,
    
    -- Priority and scheduling
    priority INTEGER DEFAULT 0,  -- Higher number = higher priority
    scheduled_for TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Metadata
    metadata JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Ensure document uniqueness per run
    UNIQUE(run_id, doc_id, source_name),
    
    -- Indexes for performance (removed inline - will create separately)
    FOREIGN KEY (run_id) REFERENCES processing_runs(run_id)
);

-- Create indexes for document_queue performance
CREATE INDEX IF NOT EXISTS idx_queue_status_run ON document_queue (run_id, status, scheduled_for);
CREATE INDEX IF NOT EXISTS idx_queue_worker ON document_queue (worker_id, status);
CREATE INDEX IF NOT EXISTS idx_queue_parent ON document_queue (parent_doc_id);
CREATE INDEX IF NOT EXISTS idx_queue_priority ON document_queue (priority DESC, scheduled_for ASC);

-- Run workers table - tracks which workers are processing which run
CREATE TABLE IF NOT EXISTS run_workers (
    run_id VARCHAR(16) NOT NULL REFERENCES processing_runs(run_id),
    worker_id VARCHAR(100) NOT NULL,
    
    -- Worker lifecycle
    joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_heartbeat TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    left_at TIMESTAMP,
    
    -- Worker status
    status VARCHAR(20) DEFAULT 'active',
    -- Status values: 'active', 'idle', 'processing', 'stopped', 'failed'
    
    -- Statistics
    documents_claimed INTEGER DEFAULT 0,
    documents_processed INTEGER DEFAULT 0,
    documents_failed INTEGER DEFAULT 0,
    processing_time_seconds NUMERIC DEFAULT 0,
    
    -- Worker metadata
    hostname VARCHAR(255),
    process_id INTEGER,
    version VARCHAR(50),
    capabilities JSONB,  -- e.g., {"pdf": true, "ocr": false}
    
    PRIMARY KEY (run_id, worker_id)
);

-- Create index for run_workers performance
CREATE INDEX IF NOT EXISTS idx_worker_heartbeat ON run_workers (last_heartbeat DESC);

-- Document dependencies table - tracks linked document relationships
CREATE TABLE IF NOT EXISTS document_dependencies (
    parent_doc_id VARCHAR(500) NOT NULL,
    child_doc_id VARCHAR(500) NOT NULL,
    source_name VARCHAR(255) NOT NULL,
    run_id VARCHAR(16) NOT NULL REFERENCES processing_runs(run_id),
    
    -- Link metadata
    link_type VARCHAR(50),  -- 'explicit', 'discovered', 'inferred'
    link_depth INTEGER NOT NULL,
    discovered_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    discovered_by_worker VARCHAR(100),
    
    PRIMARY KEY (run_id, parent_doc_id, child_doc_id, source_name)
);

-- Create indexes for document_dependencies performance  
CREATE INDEX IF NOT EXISTS idx_deps_child ON document_dependencies (child_doc_id);
CREATE INDEX IF NOT EXISTS idx_deps_run ON document_dependencies (run_id);

-- Create update trigger for updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_document_queue_updated_at 
    BEFORE UPDATE ON document_queue 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

-- Create function to check for stale workers
CREATE OR REPLACE FUNCTION check_stale_workers(heartbeat_timeout_seconds INTEGER DEFAULT 300)
RETURNS TABLE(run_id VARCHAR, worker_id VARCHAR, last_heartbeat TIMESTAMP) AS $$
BEGIN
    RETURN QUERY
    SELECT rw.run_id, rw.worker_id, rw.last_heartbeat
    FROM run_workers rw
    WHERE rw.status = 'active'
      AND rw.last_heartbeat < CURRENT_TIMESTAMP - (heartbeat_timeout_seconds || ' seconds')::INTERVAL;
END;
$$ LANGUAGE plpgsql;

-- Create function to reclaim stale work
CREATE OR REPLACE FUNCTION reclaim_stale_work(timeout_seconds INTEGER DEFAULT 300)
RETURNS INTEGER AS $$
DECLARE
    reclaimed_count INTEGER;
BEGIN
    UPDATE document_queue
    SET status = 'pending',
        worker_id = NULL,
        claimed_at = NULL,
        retry_count = retry_count + 1
    WHERE status = 'processing'
      AND claimed_at < CURRENT_TIMESTAMP - (timeout_seconds || ' seconds')::INTERVAL;
    
    GET DIAGNOSTICS reclaimed_count = ROW_COUNT;
    RETURN reclaimed_count;
END;
$$ LANGUAGE plpgsql;

-- Create function to attempt leader election (atomic)
CREATE OR REPLACE FUNCTION attempt_leader_election(
    p_run_id VARCHAR(16),
    p_worker_id VARCHAR(100),
    p_lease_duration_seconds INTEGER DEFAULT 60
)
RETURNS BOOLEAN AS $$
DECLARE
    elected BOOLEAN := FALSE;
BEGIN
    -- Try to become leader if no current leader or lease expired
    UPDATE processing_runs
    SET leader_worker_id = p_worker_id,
        leader_elected_at = CURRENT_TIMESTAMP,
        leader_heartbeat = CURRENT_TIMESTAMP,
        leader_lease_expires = CURRENT_TIMESTAMP + (p_lease_duration_seconds || ' seconds')::INTERVAL
    WHERE run_id = p_run_id
      AND (
        leader_worker_id IS NULL OR 
        leader_lease_expires < CURRENT_TIMESTAMP
      );
    
    -- Check if we became leader
    IF FOUND THEN
        elected := TRUE;
    ELSE
        -- Check if we're already the leader and just need to renew lease
        UPDATE processing_runs
        SET leader_heartbeat = CURRENT_TIMESTAMP,
            leader_lease_expires = CURRENT_TIMESTAMP + (p_lease_duration_seconds || ' seconds')::INTERVAL
        WHERE run_id = p_run_id
          AND leader_worker_id = p_worker_id;
          
        IF FOUND THEN
            elected := TRUE;
        END IF;
    END IF;
    
    RETURN elected;
END;
$$ LANGUAGE plpgsql;