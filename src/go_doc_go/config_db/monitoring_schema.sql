-- Pipeline Monitoring Schema
-- Unified tables for real-time pipeline job monitoring and status tracking

-- Main job status table - single source of truth for pipeline execution status
CREATE TABLE IF NOT EXISTS pipeline_job_status (
    run_id TEXT PRIMARY KEY,
    pipeline_id INTEGER NOT NULL,
    pipeline_name TEXT,
    status TEXT NOT NULL CHECK (status IN ('pending', 'initializing', 'running', 'completed', 'failed', 'paused', 'cancelled', 'cancelled_cleanup')),
    phase TEXT CHECK (phase IN ('setup', 'ingestion', 'parsing', 'extraction', 'embedding', 'storage', 'cleanup')),

    -- Worker statistics
    total_workers INTEGER DEFAULT 1,
    active_workers INTEGER DEFAULT 0,
    failed_workers INTEGER DEFAULT 0,

    -- Document processing statistics
    documents_total INTEGER DEFAULT 0,
    documents_claimed INTEGER DEFAULT 0,
    documents_processed INTEGER DEFAULT 0,
    documents_failed INTEGER DEFAULT 0,
    documents_skipped INTEGER DEFAULT 0,

    -- Performance metrics
    avg_processing_time_ms INTEGER,
    total_processing_time_ms INTEGER,
    queue_wait_time_ms INTEGER,

    -- Progress tracking
    progress_percentage REAL DEFAULT 0.0,
    estimated_completion_time TIMESTAMP,

    -- Health and heartbeat
    health_status TEXT DEFAULT 'healthy' CHECK (health_status IN ('healthy', 'warning', 'critical', 'stale', 'unknown')),
    last_heartbeat TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_error TEXT,
    error_count INTEGER DEFAULT 0,
    warning_count INTEGER DEFAULT 0,

    -- Cleanup tracking
    cleanup_status TEXT DEFAULT 'none' CHECK (cleanup_status IN ('none', 'pending', 'in_progress', 'completed', 'failed')),
    cleanup_started_at TIMESTAMP,
    cleanup_completed_at TIMESTAMP,
    documents_cleaned INTEGER DEFAULT 0,
    elements_cleaned INTEGER DEFAULT 0,

    -- Timestamps
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    -- Additional metadata as JSON
    metadata TEXT,  -- JSON metadata

    FOREIGN KEY (pipeline_id) REFERENCES pipelines(id) ON DELETE CASCADE
);

-- Worker-level status tracking
CREATE TABLE IF NOT EXISTS pipeline_worker_status (
    worker_id TEXT PRIMARY KEY,
    run_id TEXT NOT NULL,
    worker_index INTEGER,
    worker_type TEXT DEFAULT 'processor',  -- 'processor', 'coordinator', 'monitor'
    hostname TEXT,
    pid INTEGER,

    -- Status tracking
    status TEXT NOT NULL CHECK (status IN ('idle', 'claiming', 'processing', 'failed', 'completed', 'terminated')),
    current_document_id TEXT,
    current_document_type TEXT,
    current_phase TEXT,

    -- Performance metrics
    documents_processed INTEGER DEFAULT 0,
    documents_failed INTEGER DEFAULT 0,
    documents_skipped INTEGER DEFAULT 0,
    total_processing_time_ms INTEGER DEFAULT 0,
    avg_processing_time_ms INTEGER,

    -- Resource usage
    memory_usage_mb INTEGER,
    cpu_usage_percent REAL,

    -- Health tracking
    consecutive_failures INTEGER DEFAULT 0,
    last_error TEXT,
    last_error_time TIMESTAMP,

    -- Heartbeat and timestamps
    last_heartbeat TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,

    FOREIGN KEY (run_id) REFERENCES pipeline_job_status(run_id) ON DELETE CASCADE
);

-- Document processing events for detailed tracking
CREATE TABLE IF NOT EXISTS pipeline_processing_events (
    event_id INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id TEXT NOT NULL,
    worker_id TEXT,
    document_id TEXT NOT NULL,
    event_type TEXT NOT NULL CHECK (event_type IN ('claimed', 'started', 'completed', 'failed', 'skipped', 'retried')),
    phase TEXT,
    processing_time_ms INTEGER,
    error_message TEXT,
    metadata TEXT,  -- JSON with additional event data
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (run_id) REFERENCES pipeline_job_status(run_id) ON DELETE CASCADE,
    FOREIGN KEY (worker_id) REFERENCES pipeline_worker_status(worker_id) ON DELETE CASCADE
);

-- Monitoring checkpoints for phase transitions
CREATE TABLE IF NOT EXISTS pipeline_phase_checkpoints (
    checkpoint_id INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id TEXT NOT NULL,
    phase TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('started', 'completed', 'failed', 'skipped')),
    documents_in_phase INTEGER,
    documents_completed INTEGER,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    duration_ms INTEGER,
    error_message TEXT,

    FOREIGN KEY (run_id) REFERENCES pipeline_job_status(run_id) ON DELETE CASCADE,
    UNIQUE(run_id, phase)
);

-- Performance metrics aggregation table
CREATE TABLE IF NOT EXISTS pipeline_performance_metrics (
    metric_id INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id TEXT NOT NULL,
    metric_name TEXT NOT NULL,
    metric_value REAL NOT NULL,
    metric_unit TEXT,
    aggregation_type TEXT DEFAULT 'current',  -- 'current', 'avg', 'sum', 'min', 'max'
    window_start TIMESTAMP,
    window_end TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (run_id) REFERENCES pipeline_job_status(run_id) ON DELETE CASCADE
);

-- Create indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_job_status_pipeline_id ON pipeline_job_status(pipeline_id);
CREATE INDEX IF NOT EXISTS idx_job_status_status ON pipeline_job_status(status);
CREATE INDEX IF NOT EXISTS idx_job_status_started_at ON pipeline_job_status(started_at DESC);
CREATE INDEX IF NOT EXISTS idx_job_status_health ON pipeline_job_status(health_status);
CREATE INDEX IF NOT EXISTS idx_job_status_last_heartbeat ON pipeline_job_status(last_heartbeat DESC);

CREATE INDEX IF NOT EXISTS idx_worker_status_run_id ON pipeline_worker_status(run_id);
CREATE INDEX IF NOT EXISTS idx_worker_status_status ON pipeline_worker_status(status);
CREATE INDEX IF NOT EXISTS idx_worker_status_heartbeat ON pipeline_worker_status(last_heartbeat DESC);

CREATE INDEX IF NOT EXISTS idx_processing_events_run_id ON pipeline_processing_events(run_id);
CREATE INDEX IF NOT EXISTS idx_processing_events_document_id ON pipeline_processing_events(document_id);
CREATE INDEX IF NOT EXISTS idx_processing_events_created_at ON pipeline_processing_events(created_at DESC);

CREATE INDEX IF NOT EXISTS idx_phase_checkpoints_run_id ON pipeline_phase_checkpoints(run_id);
CREATE INDEX IF NOT EXISTS idx_performance_metrics_run_id ON pipeline_performance_metrics(run_id);

-- Triggers for automatic timestamp updates
CREATE TRIGGER IF NOT EXISTS update_job_status_timestamp
    AFTER UPDATE ON pipeline_job_status
    FOR EACH ROW
    BEGIN
        UPDATE pipeline_job_status SET updated_at = CURRENT_TIMESTAMP WHERE run_id = NEW.run_id;
    END;

-- Views for monitoring dashboard

-- Real-time pipeline monitoring view
CREATE VIEW IF NOT EXISTS pipeline_monitoring_dashboard AS
SELECT
    pjs.run_id,
    pjs.pipeline_id,
    pjs.pipeline_name,
    pjs.status,
    pjs.phase,
    pjs.health_status,

    -- Progress calculation
    pjs.progress_percentage,
    CASE
        WHEN pjs.documents_total > 0
        THEN ROUND((CAST(pjs.documents_processed AS REAL) / pjs.documents_total) * 100, 2)
        ELSE 0
    END as calculated_progress_pct,

    -- Document statistics
    pjs.documents_total,
    pjs.documents_claimed,
    pjs.documents_processed,
    pjs.documents_failed,
    pjs.documents_skipped,
    (pjs.documents_total - pjs.documents_processed - pjs.documents_failed - pjs.documents_skipped) as documents_remaining,

    -- Worker statistics
    pjs.total_workers,
    pjs.active_workers,
    pjs.failed_workers,
    (pjs.total_workers - pjs.active_workers - pjs.failed_workers) as idle_workers,

    -- Performance metrics
    pjs.avg_processing_time_ms,
    CASE
        WHEN pjs.avg_processing_time_ms > 0 AND pjs.documents_remaining > 0
        THEN datetime(pjs.started_at, '+' || (pjs.avg_processing_time_ms * pjs.documents_remaining / 1000 / MAX(pjs.active_workers, 1)) || ' seconds')
        ELSE pjs.estimated_completion_time
    END as calculated_eta,

    -- Timing
    pjs.started_at,
    pjs.completed_at,
    pjs.last_heartbeat,
    CASE
        WHEN pjs.completed_at IS NOT NULL
        THEN CAST((julianday(pjs.completed_at) - julianday(pjs.started_at)) * 24 * 60 AS INTEGER)
        ELSE CAST((julianday(CURRENT_TIMESTAMP) - julianday(pjs.started_at)) * 24 * 60 AS INTEGER)
    END as duration_minutes,

    -- Health check
    CASE
        WHEN pjs.last_heartbeat < datetime('now', '-5 minutes') THEN 'stale'
        WHEN pjs.failed_workers > pjs.total_workers * 0.5 THEN 'critical'
        WHEN pjs.failed_workers > 0 OR pjs.error_count > 10 THEN 'warning'
        ELSE 'healthy'
    END as calculated_health,

    pjs.error_count,
    pjs.warning_count,
    pjs.last_error

FROM pipeline_job_status pjs
ORDER BY pjs.started_at DESC;

-- Worker health monitoring view
CREATE VIEW IF NOT EXISTS worker_health_monitor AS
SELECT
    pws.worker_id,
    pws.run_id,
    pws.worker_index,
    pws.status,
    pws.hostname,
    pws.current_document_id,
    pws.documents_processed,
    pws.documents_failed,
    pws.avg_processing_time_ms,
    pws.last_heartbeat,

    -- Health indicators
    CASE
        WHEN pws.last_heartbeat < datetime('now', '-2 minutes') THEN 'stale'
        WHEN pws.consecutive_failures > 5 THEN 'critical'
        WHEN pws.consecutive_failures > 0 THEN 'warning'
        ELSE 'healthy'
    END as health_status,

    CAST((julianday(CURRENT_TIMESTAMP) - julianday(pws.last_heartbeat)) * 24 * 60 * 60 AS INTEGER) as seconds_since_heartbeat,

    -- Performance metrics
    CASE
        WHEN pws.documents_processed > 0
        THEN ROUND(CAST(pws.documents_failed AS REAL) / pws.documents_processed * 100, 2)
        ELSE 0
    END as failure_rate_pct,

    pws.memory_usage_mb,
    pws.cpu_usage_percent

FROM pipeline_worker_status pws
ORDER BY pws.run_id, pws.worker_index;

-- Active jobs summary view
CREATE VIEW IF NOT EXISTS active_jobs_summary AS
SELECT
    COUNT(*) as total_active_jobs,
    COUNT(CASE WHEN status = 'running' THEN 1 END) as running_jobs,
    COUNT(CASE WHEN status = 'pending' THEN 1 END) as pending_jobs,
    COUNT(CASE WHEN status = 'paused' THEN 1 END) as paused_jobs,
    SUM(total_workers) as total_workers_deployed,
    SUM(active_workers) as total_active_workers,
    SUM(documents_processed) as total_documents_processed,
    SUM(documents_failed) as total_documents_failed,
    AVG(progress_percentage) as avg_progress_percentage,
    MIN(started_at) as oldest_job_started,
    MAX(last_heartbeat) as most_recent_heartbeat
FROM pipeline_job_status
WHERE status IN ('pending', 'initializing', 'running', 'paused');

-- Phase performance view
CREATE VIEW IF NOT EXISTS phase_performance_analysis AS
SELECT
    ppc.phase,
    COUNT(DISTINCT ppc.run_id) as total_runs,
    AVG(ppc.duration_ms) as avg_duration_ms,
    MIN(ppc.duration_ms) as min_duration_ms,
    MAX(ppc.duration_ms) as max_duration_ms,
    SUM(CASE WHEN ppc.status = 'completed' THEN 1 ELSE 0 END) as successful_phases,
    SUM(CASE WHEN ppc.status = 'failed' THEN 1 ELSE 0 END) as failed_phases,
    AVG(CAST(ppc.documents_completed AS REAL) / NULLIF(ppc.documents_in_phase, 0) * 100) as avg_completion_rate
FROM pipeline_phase_checkpoints ppc
GROUP BY ppc.phase
ORDER BY
    CASE ppc.phase
        WHEN 'setup' THEN 1
        WHEN 'ingestion' THEN 2
        WHEN 'parsing' THEN 3
        WHEN 'extraction' THEN 4
        WHEN 'embedding' THEN 5
        WHEN 'storage' THEN 6
        WHEN 'cleanup' THEN 7
        ELSE 8
    END;

-- Historical job performance view
CREATE VIEW IF NOT EXISTS historical_job_performance AS
SELECT
    DATE(pjs.started_at) as execution_date,
    COUNT(*) as total_jobs,
    COUNT(CASE WHEN pjs.status = 'completed' THEN 1 END) as successful_jobs,
    COUNT(CASE WHEN pjs.status = 'failed' THEN 1 END) as failed_jobs,
    AVG(CAST(pjs.documents_processed AS REAL) / NULLIF(pjs.documents_total, 0) * 100) as avg_completion_rate,
    AVG(pjs.avg_processing_time_ms) as avg_processing_time,
    SUM(pjs.documents_processed) as total_documents_processed,
    SUM(pjs.documents_failed) as total_documents_failed
FROM pipeline_job_status pjs
WHERE pjs.started_at >= datetime('now', '-30 days')
GROUP BY DATE(pjs.started_at)
ORDER BY execution_date DESC;

-- Add current run tracking to pipelines table (if not exists)
-- This tracks which run_id is considered "current" for queries
-- SQLite doesn't support ALTER TABLE ADD COLUMN IF NOT EXISTS, so we use a workaround
CREATE TABLE IF NOT EXISTS _pipeline_monitoring_migration (
    migration_id TEXT PRIMARY KEY,
    applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Track applied migrations
INSERT OR IGNORE INTO _pipeline_monitoring_migration (migration_id) VALUES ('add_current_run_tracking');