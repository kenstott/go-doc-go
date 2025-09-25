"""
Configurable job control database for CLI coordination.
Supports multiple database backends (SQLite, PostgreSQL, MySQL).
"""

import hashlib
import json
import logging
import os
import sqlite3
import socket
import platform
import time
from abc import ABC, abstractmethod
from datetime import datetime, timedelta
from typing import Dict, Any, Optional, List, Tuple
from pathlib import Path

logger = logging.getLogger(__name__)


class JobControlDB(ABC):
    """Abstract base class for job control database backends."""

    @classmethod
    def create(cls, config) -> 'JobControlDB':
        """Factory method to create appropriate backend from config."""
        job_control_config = config.get_job_control_config()
        backend = job_control_config.get('backend', 'sqlite')

        if backend == 'sqlite':
            return SQLiteJobControlDB(config)
        elif backend == 'postgresql':
            return PostgreSQLJobControlDB(config)
        elif backend == 'mysql':
            return MySQLJobControlDB(config)
        else:
            raise ValueError(f"Unsupported job control backend: {backend}")

    @abstractmethod
    def initialize_schema(self):
        """Create database schema if it doesn't exist."""
        pass

    @abstractmethod
    def create_processing_run(self, config: Dict[str, Any]) -> str:
        """Create a processing run from config, return run_id."""
        pass

    @abstractmethod
    def get_run_status(self, run_id: str) -> Optional[Dict[str, Any]]:
        """Get status of a processing run."""
        pass

    @abstractmethod
    def list_active_runs(self) -> List[Dict[str, Any]]:
        """List all active processing runs."""
        pass

    @abstractmethod
    def enqueue_documents(self, run_id: str, documents: List[Dict[str, Any]]):
        """Add documents to the processing queue."""
        pass

    @abstractmethod
    def claim_document(self, run_id: str, worker_id: str) -> Optional[Dict[str, Any]]:
        """Atomically claim a document for processing."""
        pass

    @abstractmethod
    def update_document_status(self, queue_id: int, status: str, **kwargs):
        """Update document processing status."""
        pass

    @abstractmethod
    def register_worker(self, run_id: str, worker_id: str, **metadata) -> bool:
        """Register a worker for a processing run."""
        pass

    @abstractmethod
    def update_worker_heartbeat(self, run_id: str, worker_id: str) -> bool:
        """Update worker heartbeat."""
        pass

    @abstractmethod
    def cleanup_stale_workers(self, timeout_seconds: int = 300) -> int:
        """Clean up workers with stale heartbeats."""
        pass


class SQLiteJobControlDB(JobControlDB):
    """SQLite implementation of job control database."""

    def __init__(self, config):
        self.config = config
        job_control_config = config.get_job_control_config()
        self.db_path = job_control_config.get('path', './job_queue.db')
        self.claim_timeout = job_control_config.get('claim_timeout', 300)
        self.max_retries = job_control_config.get('max_retries', 3)

        # Ensure directory exists
        os.makedirs(os.path.dirname(os.path.abspath(self.db_path)), exist_ok=True)

        # Initialize schema
        self.initialize_schema()

    def _get_connection(self):
        """Get database connection with proper settings."""
        conn = sqlite3.connect(self.db_path, timeout=30.0)
        conn.row_factory = sqlite3.Row
        conn.execute("PRAGMA foreign_keys = ON")
        conn.execute("PRAGMA journal_mode = WAL")
        return conn

    def initialize_schema(self):
        """Create SQLite schema for job control."""
        schema_sql = """
        -- Processing runs table
        CREATE TABLE IF NOT EXISTS processing_runs (
            run_id TEXT PRIMARY KEY,
            config_hash TEXT NOT NULL UNIQUE,
            config_snapshot TEXT NOT NULL,
            status TEXT NOT NULL DEFAULT 'active',
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            first_worker_at TIMESTAMP,
            last_activity_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            processing_completed_at TIMESTAMP,
            completed_at TIMESTAMP,
            worker_count INTEGER DEFAULT 0,
            documents_queued INTEGER DEFAULT 0,
            documents_processed INTEGER DEFAULT 0,
            documents_failed INTEGER DEFAULT 0,
            leader_worker_id TEXT,
            leader_elected_at TIMESTAMP,
            leader_heartbeat TIMESTAMP,
            metadata TEXT
        );

        -- Document queue table
        CREATE TABLE IF NOT EXISTS document_queue (
            queue_id INTEGER PRIMARY KEY AUTOINCREMENT,
            doc_id TEXT NOT NULL,
            source_name TEXT NOT NULL,
            source_type TEXT NOT NULL DEFAULT 'configured',
            run_id TEXT NOT NULL,
            status TEXT NOT NULL DEFAULT 'pending',
            worker_id TEXT,
            claimed_at TIMESTAMP,
            started_at TIMESTAMP,
            completed_at TIMESTAMP,
            failed_at TIMESTAMP,
            retry_count INTEGER DEFAULT 0,
            max_retries INTEGER DEFAULT 3,
            error_message TEXT,
            error_details TEXT,
            parent_doc_id TEXT,
            link_depth INTEGER DEFAULT 0,
            max_link_depth INTEGER DEFAULT 3,
            content_hash TEXT,
            last_modified TIMESTAMP,
            file_size INTEGER,
            priority INTEGER DEFAULT 0,
            scheduled_for TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            metadata TEXT,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            FOREIGN KEY (run_id) REFERENCES processing_runs(run_id),
            UNIQUE(run_id, doc_id, source_name)
        );

        -- Run workers table
        CREATE TABLE IF NOT EXISTS run_workers (
            run_id TEXT NOT NULL,
            worker_id TEXT NOT NULL,
            joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            last_heartbeat TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            left_at TIMESTAMP,
            status TEXT DEFAULT 'active',
            documents_claimed INTEGER DEFAULT 0,
            documents_processed INTEGER DEFAULT 0,
            documents_failed INTEGER DEFAULT 0,
            processing_time_seconds REAL DEFAULT 0,
            hostname TEXT,
            process_id INTEGER,
            version TEXT,
            capabilities TEXT,
            PRIMARY KEY (run_id, worker_id),
            FOREIGN KEY (run_id) REFERENCES processing_runs(run_id)
        );

        -- Create indexes
        CREATE INDEX IF NOT EXISTS idx_queue_status_run ON document_queue (run_id, status, scheduled_for);
        CREATE INDEX IF NOT EXISTS idx_queue_worker ON document_queue (worker_id, status);
        CREATE INDEX IF NOT EXISTS idx_worker_heartbeat ON run_workers (last_heartbeat DESC);

        -- Create update trigger
        CREATE TRIGGER IF NOT EXISTS update_document_queue_updated_at
            AFTER UPDATE ON document_queue
            FOR EACH ROW
            BEGIN
                UPDATE document_queue SET updated_at = CURRENT_TIMESTAMP WHERE queue_id = NEW.queue_id;
            END;
        """

        with self._get_connection() as conn:
            conn.executescript(schema_sql)
            conn.commit()

    def _generate_run_id(self, config: Dict[str, Any]) -> str:
        """Generate deterministic run ID from config."""
        # Extract processing-relevant config
        processing_config = {
            'content_sources': config.get('content_sources', []),
            'storage': config.get('storage', {}),
            'embedding': config.get('embedding', {}),
            'relationship_detection': config.get('relationship_detection', {}),
        }

        config_json = json.dumps(processing_config, sort_keys=True)
        config_hash = hashlib.sha256(config_json.encode()).hexdigest()
        return config_hash[:16]  # First 16 chars for run_id

    def create_processing_run(self, config: Dict[str, Any]) -> str:
        """Create or get existing processing run."""
        run_id = self._generate_run_id(config)
        config_json = json.dumps(config, sort_keys=True)
        config_hash = hashlib.sha256(config_json.encode()).hexdigest()

        with self._get_connection() as conn:
            # Check if run already exists
            cursor = conn.execute(
                "SELECT run_id FROM processing_runs WHERE run_id = ?",
                (run_id,)
            )
            existing = cursor.fetchone()

            if not existing:
                # Create new run
                conn.execute("""
                    INSERT INTO processing_runs (
                        run_id, config_hash, config_snapshot, status
                    ) VALUES (?, ?, ?, ?)
                """, (run_id, config_hash, config_json, 'active'))
                logger.info(f"Created new processing run: {run_id}")
            else:
                logger.info(f"Using existing processing run: {run_id}")

            conn.commit()
            return run_id

    def get_run_status(self, run_id: str) -> Optional[Dict[str, Any]]:
        """Get comprehensive run status."""
        with self._get_connection() as conn:
            # Get run details
            cursor = conn.execute("""
                SELECT * FROM processing_runs WHERE run_id = ?
            """, (run_id,))
            run_data = cursor.fetchone()

            if not run_data:
                return None

            # Get document statistics
            cursor = conn.execute("""
                SELECT
                    status,
                    COUNT(*) as count,
                    AVG(CASE
                        WHEN completed_at IS NOT NULL AND started_at IS NOT NULL
                        THEN (julianday(completed_at) - julianday(started_at)) * 86400
                        ELSE NULL
                    END) as avg_processing_time
                FROM document_queue
                WHERE run_id = ?
                GROUP BY status
            """, (run_id,))
            doc_stats = {row['status']: {
                'count': row['count'],
                'avg_processing_time': row['avg_processing_time']
            } for row in cursor.fetchall()}

            # Get worker information
            cursor = conn.execute("""
                SELECT * FROM run_workers
                WHERE run_id = ? AND status = 'active'
                ORDER BY last_heartbeat DESC
            """, (run_id,))
            workers = [dict(row) for row in cursor.fetchall()]

            return {
                'run_id': run_data['run_id'],
                'status': run_data['status'],
                'created_at': run_data['created_at'],
                'documents_queued': run_data['documents_queued'],
                'documents_processed': run_data['documents_processed'],
                'documents_failed': run_data['documents_failed'],
                'worker_count': len(workers),
                'document_stats': doc_stats,
                'workers': workers
            }

    def list_active_runs(self) -> List[Dict[str, Any]]:
        """List all active processing runs."""
        with self._get_connection() as conn:
            cursor = conn.execute("""
                SELECT run_id, status, created_at, last_activity_at,
                       documents_queued, documents_processed, documents_failed,
                       worker_count
                FROM processing_runs
                WHERE status IN ('active', 'processing_complete')
                ORDER BY last_activity_at DESC
            """)
            return [dict(row) for row in cursor.fetchall()]

    def enqueue_documents(self, run_id: str, documents: List[Dict[str, Any]]):
        """Add documents to processing queue."""
        with self._get_connection() as conn:
            for doc in documents:
                conn.execute("""
                    INSERT OR IGNORE INTO document_queue (
                        run_id, doc_id, source_name, source_type,
                        metadata, priority, scheduled_for
                    ) VALUES (?, ?, ?, ?, ?, ?, ?)
                """, (
                    run_id,
                    doc['doc_id'],
                    doc['source_name'],
                    doc.get('source_type', 'configured'),
                    json.dumps(doc.get('metadata', {})),
                    doc.get('priority', 0),
                    doc.get('scheduled_for', datetime.now())
                ))

            # Update documents_queued count
            conn.execute("""
                UPDATE processing_runs
                SET documents_queued = (
                    SELECT COUNT(*) FROM document_queue WHERE run_id = ?
                ),
                last_activity_at = CURRENT_TIMESTAMP
                WHERE run_id = ?
            """, (run_id, run_id))

            conn.commit()
            logger.info(f"Enqueued {len(documents)} documents for run {run_id}")

    def claim_document(self, run_id: str, worker_id: str) -> Optional[Dict[str, Any]]:
        """Atomically claim next available document."""
        with self._get_connection() as conn:
            # Start transaction for atomic claim
            cursor = conn.execute("""
                SELECT queue_id, doc_id, source_name, metadata
                FROM document_queue
                WHERE run_id = ?
                  AND status = 'pending'
                  AND (scheduled_for IS NULL OR scheduled_for <= CURRENT_TIMESTAMP)
                ORDER BY priority DESC, scheduled_for ASC
                LIMIT 1
            """, (run_id,))

            doc = cursor.fetchone()
            if not doc:
                return None

            # Claim the document
            queue_id = doc['queue_id']
            conn.execute("""
                UPDATE document_queue
                SET status = 'processing',
                    worker_id = ?,
                    claimed_at = CURRENT_TIMESTAMP,
                    started_at = CURRENT_TIMESTAMP
                WHERE queue_id = ?
            """, (worker_id, queue_id))

            # Update worker stats
            conn.execute("""
                UPDATE run_workers
                SET documents_claimed = documents_claimed + 1,
                    last_heartbeat = CURRENT_TIMESTAMP
                WHERE run_id = ? AND worker_id = ?
            """, (run_id, worker_id))

            conn.commit()

            return {
                'queue_id': queue_id,
                'doc_id': doc['doc_id'],
                'source_name': doc['source_name'],
                'metadata': json.loads(doc['metadata'] or '{}')
            }

    def update_document_status(self, queue_id: int, status: str, **kwargs):
        """Update document processing status."""
        with self._get_connection() as conn:
            updates = ['status = ?']
            params = [status]

            if status == 'completed':
                updates.append('completed_at = CURRENT_TIMESTAMP')
            elif status == 'failed':
                updates.append('failed_at = CURRENT_TIMESTAMP')
                if 'error_message' in kwargs:
                    updates.append('error_message = ?')
                    params.append(kwargs['error_message'])
                if 'error_details' in kwargs:
                    updates.append('error_details = ?')
                    params.append(json.dumps(kwargs['error_details']))

            params.append(queue_id)

            conn.execute(f"""
                UPDATE document_queue
                SET {', '.join(updates)}
                WHERE queue_id = ?
            """, params)

            conn.commit()

    def register_worker(self, run_id: str, worker_id: str, **metadata) -> bool:
        """Register worker for processing run."""
        with self._get_connection() as conn:
            conn.execute("""
                INSERT OR REPLACE INTO run_workers (
                    run_id, worker_id, hostname, process_id, capabilities,
                    joined_at, last_heartbeat, status
                ) VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'active')
            """, (
                run_id, worker_id,
                metadata.get('hostname', socket.gethostname()),
                metadata.get('process_id', os.getpid()),
                json.dumps(metadata.get('capabilities', {}))
            ))

            # Update worker count
            conn.execute("""
                UPDATE processing_runs
                SET worker_count = (
                    SELECT COUNT(*) FROM run_workers
                    WHERE run_id = ? AND status = 'active'
                ),
                first_worker_at = COALESCE(first_worker_at, CURRENT_TIMESTAMP),
                last_activity_at = CURRENT_TIMESTAMP
                WHERE run_id = ?
            """, (run_id, run_id))

            conn.commit()
            return True

    def update_worker_heartbeat(self, run_id: str, worker_id: str) -> bool:
        """Update worker heartbeat."""
        with self._get_connection() as conn:
            cursor = conn.execute("""
                UPDATE run_workers
                SET last_heartbeat = CURRENT_TIMESTAMP
                WHERE run_id = ? AND worker_id = ?
            """, (run_id, worker_id))

            conn.commit()
            return cursor.rowcount > 0

    def cleanup_stale_workers(self, timeout_seconds: int = 300) -> int:
        """Clean up stale workers and reclaim their documents."""
        with self._get_connection() as conn:
            cutoff_time = datetime.now() - timedelta(seconds=timeout_seconds)

            # Find stale workers
            cursor = conn.execute("""
                SELECT run_id, worker_id FROM run_workers
                WHERE status = 'active'
                  AND last_heartbeat < ?
            """, (cutoff_time,))
            stale_workers = cursor.fetchall()

            if not stale_workers:
                return 0

            # Mark workers as failed
            worker_ids = [(row['run_id'], row['worker_id']) for row in stale_workers]
            for run_id, worker_id in worker_ids:
                conn.execute("""
                    UPDATE run_workers
                    SET status = 'failed', left_at = CURRENT_TIMESTAMP
                    WHERE run_id = ? AND worker_id = ?
                """, (run_id, worker_id))

            # Reclaim their documents
            reclaimed = 0
            for run_id, worker_id in worker_ids:
                cursor = conn.execute("""
                    UPDATE document_queue
                    SET status = 'pending',
                        worker_id = NULL,
                        claimed_at = NULL,
                        retry_count = retry_count + 1
                    WHERE run_id = ? AND worker_id = ? AND status = 'processing'
                """, (run_id, worker_id))
                reclaimed += cursor.rowcount

            conn.commit()
            logger.warning(f"Cleaned up {len(stale_workers)} stale workers, reclaimed {reclaimed} documents")
            return len(stale_workers)


class PostgreSQLJobControlDB(JobControlDB):
    """PostgreSQL implementation - placeholder for future implementation."""

    def __init__(self, config):
        self.config = config
        # TODO: Implement PostgreSQL backend
        raise NotImplementedError("PostgreSQL job control backend not yet implemented")

    def initialize_schema(self):
        raise NotImplementedError()

    def create_processing_run(self, config: Dict[str, Any]) -> str:
        raise NotImplementedError()

    def get_run_status(self, run_id: str) -> Optional[Dict[str, Any]]:
        raise NotImplementedError()

    def list_active_runs(self) -> List[Dict[str, Any]]:
        raise NotImplementedError()

    def enqueue_documents(self, run_id: str, documents: List[Dict[str, Any]]):
        raise NotImplementedError()

    def claim_document(self, run_id: str, worker_id: str) -> Optional[Dict[str, Any]]:
        raise NotImplementedError()

    def update_document_status(self, queue_id: int, status: str, **kwargs):
        raise NotImplementedError()

    def register_worker(self, run_id: str, worker_id: str, **metadata) -> bool:
        raise NotImplementedError()

    def update_worker_heartbeat(self, run_id: str, worker_id: str) -> bool:
        raise NotImplementedError()

    def cleanup_stale_workers(self, timeout_seconds: int = 300) -> int:
        raise NotImplementedError()


class MySQLJobControlDB(JobControlDB):
    """MySQL implementation - placeholder for future implementation."""

    def __init__(self, config):
        self.config = config
        # TODO: Implement MySQL backend
        raise NotImplementedError("MySQL job control backend not yet implemented")

    def initialize_schema(self):
        raise NotImplementedError()

    def create_processing_run(self, config: Dict[str, Any]) -> str:
        raise NotImplementedError()

    def get_run_status(self, run_id: str) -> Optional[Dict[str, Any]]:
        raise NotImplementedError()

    def list_active_runs(self) -> List[Dict[str, Any]]:
        raise NotImplementedError()

    def enqueue_documents(self, run_id: str, documents: List[Dict[str, Any]]):
        raise NotImplementedError()

    def claim_document(self, run_id: str, worker_id: str) -> Optional[Dict[str, Any]]:
        raise NotImplementedError()

    def update_document_status(self, queue_id: int, status: str, **kwargs):
        raise NotImplementedError()

    def register_worker(self, run_id: str, worker_id: str, **metadata) -> bool:
        raise NotImplementedError()

    def update_worker_heartbeat(self, run_id: str, worker_id: str) -> bool:
        raise NotImplementedError()

    def cleanup_stale_workers(self, timeout_seconds: int = 300) -> int:
        raise NotImplementedError()