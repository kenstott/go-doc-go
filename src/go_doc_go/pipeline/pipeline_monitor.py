"""
Unified pipeline monitoring system for real-time job and worker status tracking.
"""

import json
import logging
import os
import sqlite3
import threading
import time
from datetime import datetime, timedelta
from typing import Dict, Any, List, Optional, Tuple
from enum import Enum

logger = logging.getLogger(__name__)


class JobStatus(Enum):
    """Pipeline job status states."""
    PENDING = "pending"
    INITIALIZING = "initializing"
    RUNNING = "running"
    COMPLETED = "completed"
    FAILED = "failed"
    PAUSED = "paused"
    CANCELLED = "cancelled"


class JobPhase(Enum):
    """Pipeline execution phases."""
    SETUP = "setup"
    INGESTION = "ingestion"
    PARSING = "parsing"
    EXTRACTION = "extraction"
    EMBEDDING = "embedding"
    STORAGE = "storage"
    CLEANUP = "cleanup"


class WorkerStatus(Enum):
    """Worker status states."""
    IDLE = "idle"
    CLAIMING = "claiming"
    PROCESSING = "processing"
    FAILED = "failed"
    COMPLETED = "completed"
    TERMINATED = "terminated"


class HealthStatus(Enum):
    """Health status indicators."""
    HEALTHY = "healthy"
    WARNING = "warning"
    CRITICAL = "critical"
    STALE = "stale"
    UNKNOWN = "unknown"


class PipelineMonitor:
    """
    Unified monitoring system for pipeline executions.
    Provides real-time status tracking for jobs and workers.
    """

    def __init__(self, db_path: str = None):
        """
        Initialize the pipeline monitor.

        Args:
            db_path: Path to the monitoring database (defaults to PIPELINE_CONFIG_DB env var)
        """
        self.db_path = db_path or os.environ.get('PIPELINE_CONFIG_DB', 'pipeline_config.db')
        self._lock = threading.RLock()
        self._heartbeat_threads = {}

        # Initialize monitoring schema
        self._init_monitoring_schema()

        logger.info(f"Pipeline monitor initialized with database: {self.db_path}")

    def _init_monitoring_schema(self):
        """Initialize monitoring tables if they don't exist."""
        schema_path = os.path.join(os.path.dirname(__file__), '..', 'config_db', 'monitoring_schema.sql')

        if os.path.exists(schema_path):
            with open(schema_path, 'r') as f:
                schema_sql = f.read()

            with self._get_connection() as conn:
                conn.executescript(schema_sql)
                conn.commit()
                logger.info("Monitoring schema initialized")
        else:
            logger.warning(f"Monitoring schema file not found: {schema_path}")

    def _get_connection(self) -> sqlite3.Connection:
        """Get database connection with row factory and proper concurrency settings."""
        conn = sqlite3.connect(self.db_path, timeout=30.0)
        conn.row_factory = sqlite3.Row

        # Enable WAL mode for better concurrency
        conn.execute("PRAGMA journal_mode=WAL")
        conn.execute("PRAGMA synchronous=NORMAL")
        conn.execute("PRAGMA cache_size=-64000")  # 64MB cache
        conn.execute("PRAGMA temp_store=memory")
        conn.execute("PRAGMA mmap_size=268435456")  # 256MB mmap

        return conn

    # Job Status Management

    def create_job(self, run_id: str, pipeline_id: int, pipeline_name: str = None,
                   total_workers: int = 1, documents_total: int = 0,
                   metadata: Dict[str, Any] = None) -> bool:
        """
        Create a new job monitoring record.

        Args:
            run_id: Unique execution run ID
            pipeline_id: Pipeline ID
            pipeline_name: Optional pipeline name
            total_workers: Number of workers to deploy
            documents_total: Total documents to process
            metadata: Additional metadata

        Returns:
            True if created successfully
        """
        with self._lock:
            try:
                with self._get_connection() as conn:
                    cursor = conn.cursor()
                    cursor.execute("""
                        INSERT INTO pipeline_job_status
                        (run_id, pipeline_id, pipeline_name, status, total_workers,
                         documents_total, metadata)
                        VALUES (?, ?, ?, ?, ?, ?, ?)
                    """, (
                        run_id, pipeline_id, pipeline_name, JobStatus.PENDING.value,
                        total_workers, documents_total,
                        json.dumps(metadata) if metadata else None
                    ))
                    conn.commit()
                    logger.info(f"Created monitoring record for job {run_id}")
                    return True

            except sqlite3.IntegrityError:
                logger.warning(f"Job {run_id} already exists in monitoring")
                return False
            except Exception as e:
                logger.error(f"Failed to create job monitoring record: {e}")
                return False

    def update_job_status(self, run_id: str, status: JobStatus = None, phase: JobPhase = None,
                         documents_claimed: int = None, documents_processed: int = None,
                         documents_failed: int = None, active_workers: int = None,
                         health_status: HealthStatus = None, last_error: str = None,
                         cleanup_status: str = None, documents_cleaned: int = None,
                         elements_cleaned: int = None, **kwargs) -> bool:
        """
        Update job status and statistics.

        Args:
            run_id: Job run ID
            status: New job status
            phase: Current execution phase
            documents_claimed: Number of documents claimed
            documents_processed: Number of documents processed
            documents_failed: Number of documents that failed
            active_workers: Number of active workers
            health_status: Health status indicator
            last_error: Last error message
            **kwargs: Additional fields to update

        Returns:
            True if updated successfully
        """
        with self._lock:
            try:
                updates = []
                params = []

                # Build dynamic update query
                if status is not None:
                    updates.append("status = ?")
                    params.append(status.value if isinstance(status, JobStatus) else status)

                if phase is not None:
                    updates.append("phase = ?")
                    params.append(phase.value if isinstance(phase, JobPhase) else phase)

                if documents_claimed is not None:
                    updates.append("documents_claimed = ?")
                    params.append(documents_claimed)

                if documents_processed is not None:
                    updates.append("documents_processed = ?")
                    params.append(documents_processed)

                if documents_failed is not None:
                    updates.append("documents_failed = ?")
                    params.append(documents_failed)

                if active_workers is not None:
                    updates.append("active_workers = ?")
                    params.append(active_workers)

                if health_status is not None:
                    updates.append("health_status = ?")
                    params.append(health_status.value if isinstance(health_status, HealthStatus) else health_status)

                if last_error is not None:
                    updates.append("last_error = ?")
                    updates.append("error_count = error_count + 1")
                    params.append(last_error)

                if cleanup_status is not None:
                    updates.append("cleanup_status = ?")
                    params.append(cleanup_status)
                    if cleanup_status == 'in_progress':
                        updates.append("cleanup_started_at = CURRENT_TIMESTAMP")
                    elif cleanup_status in ['completed', 'failed']:
                        updates.append("cleanup_completed_at = CURRENT_TIMESTAMP")

                if documents_cleaned is not None:
                    updates.append("documents_cleaned = ?")
                    params.append(documents_cleaned)

                if elements_cleaned is not None:
                    updates.append("elements_cleaned = ?")
                    params.append(elements_cleaned)

                # Always update heartbeat
                updates.append("last_heartbeat = CURRENT_TIMESTAMP")

                # Handle completion
                if status == JobStatus.COMPLETED or status == JobStatus.FAILED:
                    updates.append("completed_at = CURRENT_TIMESTAMP")

                # Calculate progress percentage
                updates.append("""
                    progress_percentage = CASE
                        WHEN documents_total > 0
                        THEN CAST(documents_processed AS REAL) / documents_total * 100
                        ELSE 0
                    END
                """)

                if not updates:
                    return True  # Nothing to update

                params.append(run_id)

                with self._get_connection() as conn:
                    cursor = conn.cursor()
                    query = f"UPDATE pipeline_job_status SET {', '.join(updates)} WHERE run_id = ?"
                    cursor.execute(query, params)
                    conn.commit()

                    if cursor.rowcount > 0:
                        logger.debug(f"Updated job status for {run_id}")
                        return True
                    else:
                        logger.warning(f"No job found with run_id {run_id}")
                        return False

            except Exception as e:
                logger.error(f"Failed to update job status: {e}")
                return False

    def job_heartbeat(self, run_id: str) -> bool:
        """
        Send a heartbeat for a job.

        Args:
            run_id: Job run ID

        Returns:
            True if heartbeat sent successfully
        """
        with self._lock:
            try:
                with self._get_connection() as conn:
                    cursor = conn.cursor()
                    cursor.execute("""
                        UPDATE pipeline_job_status
                        SET last_heartbeat = CURRENT_TIMESTAMP
                        WHERE run_id = ?
                    """, (run_id,))
                    conn.commit()
                    return cursor.rowcount > 0

            except Exception as e:
                logger.error(f"Failed to send job heartbeat: {e}")
                return False

    # Worker Management

    def register_worker(self, worker_id: str, run_id: str, worker_index: int = None,
                       worker_type: str = "processor", hostname: str = None,
                       pid: int = None) -> bool:
        """
        Register a new worker for a job.

        Args:
            worker_id: Unique worker ID
            run_id: Job run ID
            worker_index: Worker index number
            worker_type: Type of worker
            hostname: Worker hostname
            pid: Worker process ID

        Returns:
            True if registered successfully
        """
        with self._lock:
            try:
                with self._get_connection() as conn:
                    cursor = conn.cursor()
                    cursor.execute("""
                        INSERT INTO pipeline_worker_status
                        (worker_id, run_id, worker_index, worker_type, hostname, pid, status)
                        VALUES (?, ?, ?, ?, ?, ?, ?)
                    """, (
                        worker_id, run_id, worker_index, worker_type, hostname, pid,
                        WorkerStatus.IDLE.value
                    ))
                    conn.commit()
                    logger.info(f"Registered worker {worker_id} for job {run_id}")
                    return True

            except sqlite3.IntegrityError:
                logger.warning(f"Worker {worker_id} already registered")
                return False
            except Exception as e:
                logger.error(f"Failed to register worker: {e}")
                return False

    def update_worker_status(self, worker_id: str, status: WorkerStatus = None,
                           current_document_id: str = None, documents_processed: int = None,
                           documents_failed: int = None, last_error: str = None,
                           memory_usage_mb: int = None, cpu_usage_percent: float = None) -> bool:
        """
        Update worker status and statistics.

        Args:
            worker_id: Worker ID
            status: Worker status
            current_document_id: ID of document being processed
            documents_processed: Number of documents processed
            documents_failed: Number of documents failed
            last_error: Last error message
            memory_usage_mb: Memory usage in MB
            cpu_usage_percent: CPU usage percentage

        Returns:
            True if updated successfully
        """
        with self._lock:
            try:
                updates = []
                params = []

                if status is not None:
                    updates.append("status = ?")
                    params.append(status.value if isinstance(status, WorkerStatus) else status)

                if current_document_id is not None:
                    updates.append("current_document_id = ?")
                    params.append(current_document_id)

                if documents_processed is not None:
                    updates.append("documents_processed = ?")
                    params.append(documents_processed)

                if documents_failed is not None:
                    updates.append("documents_failed = ?")
                    params.append(documents_failed)

                if last_error is not None:
                    updates.append("last_error = ?")
                    updates.append("last_error_time = CURRENT_TIMESTAMP")
                    updates.append("consecutive_failures = consecutive_failures + 1")
                    params.append(last_error)
                elif status == WorkerStatus.PROCESSING:
                    # Reset consecutive failures on successful processing
                    updates.append("consecutive_failures = 0")

                if memory_usage_mb is not None:
                    updates.append("memory_usage_mb = ?")
                    params.append(memory_usage_mb)

                if cpu_usage_percent is not None:
                    updates.append("cpu_usage_percent = ?")
                    params.append(cpu_usage_percent)

                # Always update heartbeat
                updates.append("last_heartbeat = CURRENT_TIMESTAMP")

                if not updates:
                    return True

                params.append(worker_id)

                with self._get_connection() as conn:
                    cursor = conn.cursor()
                    query = f"UPDATE pipeline_worker_status SET {', '.join(updates)} WHERE worker_id = ?"
                    cursor.execute(query, params)
                    conn.commit()
                    return cursor.rowcount > 0

            except Exception as e:
                logger.error(f"Failed to update worker status: {e}")
                return False

    def worker_heartbeat(self, worker_id: str, stats: Dict[str, Any] = None) -> bool:
        """
        Send worker heartbeat with optional statistics.

        Args:
            worker_id: Worker ID
            stats: Optional statistics to update

        Returns:
            True if heartbeat sent successfully
        """
        if stats:
            return self.update_worker_status(
                worker_id,
                documents_processed=stats.get('documents_processed'),
                documents_failed=stats.get('documents_failed'),
                memory_usage_mb=stats.get('memory_usage_mb'),
                cpu_usage_percent=stats.get('cpu_usage_percent')
            )
        else:
            with self._lock:
                try:
                    with self._get_connection() as conn:
                        cursor = conn.cursor()
                        cursor.execute("""
                            UPDATE pipeline_worker_status
                            SET last_heartbeat = CURRENT_TIMESTAMP
                            WHERE worker_id = ?
                        """, (worker_id,))
                        conn.commit()
                        return cursor.rowcount > 0

                except Exception as e:
                    logger.error(f"Failed to send worker heartbeat: {e}")
                    return False

    # Processing Events

    def log_processing_event(self, run_id: str, document_id: str, event_type: str,
                            worker_id: str = None, phase: str = None,
                            processing_time_ms: int = None, error_message: str = None,
                            metadata: Dict[str, Any] = None) -> bool:
        """
        Log a document processing event.

        Args:
            run_id: Job run ID
            document_id: Document ID
            event_type: Type of event (claimed, started, completed, failed, skipped, retried)
            worker_id: Worker ID that processed the document
            phase: Processing phase
            processing_time_ms: Processing time in milliseconds
            error_message: Error message if failed
            metadata: Additional event metadata

        Returns:
            True if logged successfully
        """
        try:
            with self._get_connection() as conn:
                cursor = conn.cursor()
                cursor.execute("""
                    INSERT INTO pipeline_processing_events
                    (run_id, worker_id, document_id, event_type, phase,
                     processing_time_ms, error_message, metadata)
                    VALUES (?, ?, ?, ?, ?, ?, ?, ?)
                """, (
                    run_id, worker_id, document_id, event_type, phase,
                    processing_time_ms, error_message,
                    json.dumps(metadata) if metadata else None
                ))
                conn.commit()
                return True

        except Exception as e:
            logger.error(f"Failed to log processing event: {e}")
            return False

    # Phase Checkpoints

    def checkpoint_phase(self, run_id: str, phase: JobPhase, status: str,
                        documents_in_phase: int = None, documents_completed: int = None,
                        error_message: str = None) -> bool:
        """
        Record a phase checkpoint.

        Args:
            run_id: Job run ID
            phase: Job phase
            status: Phase status (started, completed, failed, skipped)
            documents_in_phase: Number of documents in this phase
            documents_completed: Number of documents completed
            error_message: Error message if failed

        Returns:
            True if recorded successfully
        """
        try:
            phase_value = phase.value if isinstance(phase, JobPhase) else phase

            with self._get_connection() as conn:
                cursor = conn.cursor()

                if status == 'started':
                    # Insert or update start time
                    cursor.execute("""
                        INSERT INTO pipeline_phase_checkpoints
                        (run_id, phase, status, documents_in_phase, started_at)
                        VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
                        ON CONFLICT(run_id, phase) DO UPDATE SET
                            status = ?,
                            documents_in_phase = ?,
                            started_at = CURRENT_TIMESTAMP
                    """, (run_id, phase_value, status, documents_in_phase, status, documents_in_phase))

                elif status in ['completed', 'failed', 'skipped']:
                    # Update completion
                    cursor.execute("""
                        UPDATE pipeline_phase_checkpoints
                        SET status = ?,
                            documents_completed = ?,
                            completed_at = CURRENT_TIMESTAMP,
                            duration_ms = CAST((julianday(CURRENT_TIMESTAMP) - julianday(started_at)) * 24 * 60 * 60 * 1000 AS INTEGER),
                            error_message = ?
                        WHERE run_id = ? AND phase = ?
                    """, (status, documents_completed, error_message, run_id, phase_value))

                conn.commit()
                return True

        except Exception as e:
            logger.error(f"Failed to checkpoint phase: {e}")
            return False

    # Monitoring Queries

    def get_job_status(self, run_id: str) -> Optional[Dict[str, Any]]:
        """
        Get current status of a job.

        Args:
            run_id: Job run ID

        Returns:
            Job status dictionary or None if not found
        """
        try:
            with self._get_connection() as conn:
                cursor = conn.cursor()
                cursor.execute("""
                    SELECT * FROM pipeline_monitoring_dashboard
                    WHERE run_id = ?
                """, (run_id,))

                row = cursor.fetchone()
                if row:
                    return dict(row)
                return None

        except Exception as e:
            logger.error(f"Failed to get job status: {e}")
            return None

    def get_active_jobs(self, pipeline_id: int = None) -> List[Dict[str, Any]]:
        """
        Get all active jobs, optionally filtered by pipeline.

        Args:
            pipeline_id: Optional pipeline ID filter

        Returns:
            List of active job status dictionaries
        """
        try:
            with self._get_connection() as conn:
                cursor = conn.cursor()

                if pipeline_id:
                    cursor.execute("""
                        SELECT * FROM pipeline_monitoring_dashboard
                        WHERE status IN ('pending', 'initializing', 'running', 'paused')
                        AND pipeline_id = ?
                        ORDER BY started_at DESC
                    """, (pipeline_id,))
                else:
                    cursor.execute("""
                        SELECT * FROM pipeline_monitoring_dashboard
                        WHERE status IN ('pending', 'initializing', 'running', 'paused')
                        ORDER BY started_at DESC
                    """)

                return [dict(row) for row in cursor.fetchall()]

        except Exception as e:
            logger.error(f"Failed to get active jobs: {e}")
            return []

    def get_worker_status(self, run_id: str) -> List[Dict[str, Any]]:
        """
        Get status of all workers for a job.

        Args:
            run_id: Job run ID

        Returns:
            List of worker status dictionaries
        """
        try:
            with self._get_connection() as conn:
                cursor = conn.cursor()
                cursor.execute("""
                    SELECT * FROM worker_health_monitor
                    WHERE run_id = ?
                    ORDER BY worker_index
                """, (run_id,))

                return [dict(row) for row in cursor.fetchall()]

        except Exception as e:
            logger.error(f"Failed to get worker status: {e}")
            return []

    def get_job_health(self, run_id: str) -> Tuple[HealthStatus, str]:
        """
        Determine job health based on monitoring data.

        Args:
            run_id: Job run ID

        Returns:
            Tuple of (health status, reason)
        """
        try:
            job = self.get_job_status(run_id)
            if not job:
                return HealthStatus.UNKNOWN, "Job not found"

            # Check heartbeat staleness
            last_heartbeat = datetime.fromisoformat(job['last_heartbeat'])
            if (datetime.now() - last_heartbeat).total_seconds() > 300:  # 5 minutes
                return HealthStatus.STALE, "No heartbeat for over 5 minutes"

            # Check worker health
            workers = self.get_worker_status(run_id)
            if workers:
                failed_workers = sum(1 for w in workers if w['status'] == 'failed')
                if failed_workers > len(workers) * 0.5:
                    return HealthStatus.CRITICAL, f"{failed_workers}/{len(workers)} workers failed"
                elif failed_workers > 0:
                    return HealthStatus.WARNING, f"{failed_workers} worker(s) failed"

            # Check error rate
            if job.get('documents_total', 0) > 0:
                error_rate = job.get('documents_failed', 0) / job['documents_total']
                if error_rate > 0.1:
                    return HealthStatus.CRITICAL, f"High error rate: {error_rate:.1%}"
                elif error_rate > 0.05:
                    return HealthStatus.WARNING, f"Elevated error rate: {error_rate:.1%}"

            return HealthStatus.HEALTHY, "Job running normally"

        except Exception as e:
            logger.error(f"Failed to determine job health: {e}")
            return HealthStatus.UNKNOWN, str(e)

    def get_monitoring_summary(self) -> Dict[str, Any]:
        """
        Get comprehensive monitoring summary.

        Returns:
            Dictionary with monitoring statistics
        """
        try:
            with self._get_connection() as conn:
                cursor = conn.cursor()

                # Get active jobs summary
                cursor.execute("SELECT * FROM active_jobs_summary")
                summary = dict(cursor.fetchone()) if cursor.rowcount > 0 else {}

                # Get phase performance
                cursor.execute("SELECT * FROM phase_performance_analysis")
                summary['phase_performance'] = [dict(row) for row in cursor.fetchall()]

                # Get recent job history
                cursor.execute("""
                    SELECT * FROM historical_job_performance
                    ORDER BY execution_date DESC
                    LIMIT 7
                """)
                summary['recent_history'] = [dict(row) for row in cursor.fetchall()]

                return summary

        except Exception as e:
            logger.error(f"Failed to get monitoring summary: {e}")
            return {}

    def cleanup_old_records(self, days: int = 30) -> int:
        """
        Clean up old monitoring records.

        Args:
            days: Number of days to retain

        Returns:
            Number of records deleted
        """
        try:
            cutoff_date = datetime.now() - timedelta(days=days)

            with self._get_connection() as conn:
                cursor = conn.cursor()

                # Delete old job records (cascade will handle related tables)
                cursor.execute("""
                    DELETE FROM pipeline_job_status
                    WHERE started_at < ?
                """, (cutoff_date.isoformat(),))

                deleted = cursor.rowcount
                conn.commit()

                logger.info(f"Cleaned up {deleted} old monitoring records")
                return deleted

        except Exception as e:
            logger.error(f"Failed to cleanup old records: {e}")
            return 0

    def detect_and_fix_stale_jobs(self, stale_threshold_minutes: int = 30) -> int:
        """
        Detect stale jobs (running but no heartbeat) and mark them as failed.

        Args:
            stale_threshold_minutes: Jobs without heartbeat for this many minutes are considered stale

        Returns:
            Number of stale jobs fixed
        """
        try:
            stale_cutoff = datetime.now() - timedelta(minutes=stale_threshold_minutes)

            with self._get_connection() as conn:
                cursor = conn.cursor()

                # Find jobs that are marked as running but have stale heartbeats
                cursor.execute("""
                    SELECT run_id, pipeline_id, started_at, last_heartbeat
                    FROM pipeline_job_status
                    WHERE status = 'running'
                    AND (last_heartbeat IS NULL OR last_heartbeat < ?)
                    AND started_at < ?  -- Ensure job has been running for at least threshold time
                """, (stale_cutoff.isoformat(), stale_cutoff.isoformat()))

                stale_jobs = cursor.fetchall()

                if not stale_jobs:
                    return 0

                fixed_count = 0

                for job in stale_jobs:
                    run_id, pipeline_id, started_at, last_heartbeat = job

                    logger.warning(f"Detected stale job: {run_id} (pipeline {pipeline_id}), "
                                 f"started: {started_at}, last heartbeat: {last_heartbeat}")

                    # Update job status to failed
                    self.update_job_status(
                        run_id=run_id,
                        status=JobStatus.FAILED,
                        error_message=f"Job marked as failed due to staleness (no heartbeat for {stale_threshold_minutes}+ minutes)"
                    )

                    # Also update the pipeline execution tracker
                    try:
                        from ..config_db import PipelineConfigDB, PipelineExecutionTracker

                        # Get database instance
                        db_path = os.environ.get('PIPELINE_CONFIG_DB', 'pipeline_config.db')
                        db = PipelineConfigDB(db_path)
                        tracker = PipelineExecutionTracker(db)

                        # Update execution status in the main pipeline_executions table
                        cursor.execute("""
                            UPDATE pipeline_executions
                            SET status = 'failed',
                                completed_at = ?,
                                errors_count = errors_count + 1
                            WHERE run_id = ?
                        """, (datetime.now().isoformat(), run_id))

                    except Exception as e:
                        logger.error(f"Failed to update pipeline execution status for {run_id}: {e}")

                    fixed_count += 1

                conn.commit()

                if fixed_count > 0:
                    logger.info(f"Fixed {fixed_count} stale jobs")

                return fixed_count

        except Exception as e:
            logger.error(f"Failed to detect and fix stale jobs: {e}")
            return 0

    def migrate_executions_to_monitoring(self) -> int:
        """
        Migrate existing pipeline executions to the monitoring system.

        This creates monitoring records for executions that don't have them yet,
        allowing the monitoring system to track historical jobs.

        Returns:
            Number of executions migrated
        """
        try:
            with self._get_connection() as conn:
                cursor = conn.cursor()

                # Find executions that don't have monitoring records
                cursor.execute("""
                    SELECT
                        pe.run_id,
                        pe.pipeline_id,
                        p.name as pipeline_name,
                        pe.status,
                        pe.started_at,
                        pe.completed_at,
                        pe.documents_processed,
                        pe.documents_total,
                        pe.errors_count,
                        pe.warnings_count,
                        pe.worker_count
                    FROM pipeline_executions pe
                    LEFT JOIN pipelines p ON pe.pipeline_id = p.id
                    LEFT JOIN pipeline_job_status pjs ON pe.run_id = pjs.run_id
                    WHERE pjs.run_id IS NULL
                    ORDER BY pe.started_at DESC
                """)

                executions = cursor.fetchall()

                if not executions:
                    logger.info("No executions to migrate")
                    return 0

                migrated_count = 0

                for execution in executions:
                    run_id, pipeline_id, pipeline_name, status, started_at, completed_at, \
                    documents_processed, documents_total, errors_count, warnings_count, worker_count = execution

                    # Map old status to new status
                    if status == 'paused':
                        status = 'paused'
                    elif status == 'cancelled':
                        status = 'cancelled'
                    # Other statuses remain the same

                    # Calculate progress
                    progress_pct = 0.0
                    if documents_total and documents_total > 0:
                        progress_pct = (documents_processed / documents_total) * 100.0

                    # Determine health status
                    health_status = 'healthy'
                    if errors_count > 0:
                        health_status = 'warning'
                    if status == 'failed':
                        health_status = 'critical'
                    elif status == 'running' and started_at:
                        # Check if running job is stale (started more than 1 hour ago)
                        started_time = datetime.fromisoformat(started_at.replace('Z', '+00:00') if 'Z' in started_at else started_at)
                        if (datetime.now() - started_time).total_seconds() > 3600:  # 1 hour
                            health_status = 'stale'

                    # Insert into monitoring table
                    cursor.execute("""
                        INSERT INTO pipeline_job_status (
                            run_id, pipeline_id, pipeline_name, status,
                            total_workers, documents_total, documents_processed,
                            documents_failed, progress_percentage,
                            health_status, error_count, warning_count,
                            started_at, completed_at, last_heartbeat
                        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
                    """, (
                        run_id, pipeline_id, pipeline_name, status,
                        worker_count or 1, documents_total or 0, documents_processed or 0,
                        errors_count or 0, progress_pct,
                        health_status, errors_count or 0, warnings_count or 0,
                        started_at, completed_at,
                        completed_at or started_at  # Use completed_at as last_heartbeat, or started_at if still running
                    ))

                    migrated_count += 1

                conn.commit()
                logger.info(f"Migrated {migrated_count} executions to monitoring system")
                return migrated_count

        except Exception as e:
            logger.error(f"Failed to migrate executions to monitoring: {e}")
            return 0

    def start_automatic_heartbeat(self, run_id: str, worker_id: str = None,
                                 interval_seconds: int = 30) -> None:
        """
        Start automatic heartbeat thread for a job or worker.

        Args:
            run_id: Job run ID
            worker_id: Optional worker ID for worker heartbeat
            interval_seconds: Heartbeat interval in seconds
        """
        def heartbeat_loop():
            while run_id in self._heartbeat_threads:
                if worker_id:
                    self.worker_heartbeat(worker_id)
                else:
                    self.job_heartbeat(run_id)
                time.sleep(interval_seconds)

        if run_id not in self._heartbeat_threads:
            thread = threading.Thread(target=heartbeat_loop, daemon=True)
            thread.start()
            self._heartbeat_threads[run_id] = thread
            logger.info(f"Started automatic heartbeat for {run_id}")

    def stop_automatic_heartbeat(self, run_id: str) -> None:
        """
        Stop automatic heartbeat thread.

        Args:
            run_id: Job run ID
        """
        if run_id in self._heartbeat_threads:
            del self._heartbeat_threads[run_id]
            logger.info(f"Stopped automatic heartbeat for {run_id}")