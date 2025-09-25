"""
Simplified job control system for document processing coordination.

Eliminates run_id complexity - jobs are directly managed by config and analytics outputs.
Uses job control database only for document claiming, status tracking, and worker coordination.
"""

import json
import logging
import os
import sqlite3
import socket
import time
from abc import ABC, abstractmethod
from datetime import datetime, timedelta
from typing import Dict, Any, Optional, List
from pathlib import Path

logger = logging.getLogger(__name__)


class SimpleJobControlDB(ABC):
    """Abstract base class for simplified job control database backends."""

    @classmethod
    def create(cls, config) -> 'SimpleJobControlDB':
        """Factory method to create appropriate backend from config."""
        job_control_config = config.get_job_control_config()
        backend = job_control_config.get('backend', 'sqlite')

        if backend == 'sqlite':
            return SimpleSQLiteJobControlDB(config)
        elif backend == 'postgresql':
            return SimplePostgreSQLJobControlDB(config)
        elif backend == 'mysql':
            return SimpleMySQLJobControlDB(config)
        else:
            raise ValueError(f"Unsupported job control backend: {backend}")

    @abstractmethod
    def initialize_schema(self):
        """Create database schema if it doesn't exist."""
        pass

    @abstractmethod
    def enqueue_document(self, doc_id: str, source: str, metadata: Dict[str, Any]):
        """Add a document to the processing queue."""
        pass

    @abstractmethod
    def claim_next_document(self, worker_id: str) -> Optional[Dict[str, Any]]:
        """Atomically claim next available document for processing."""
        pass

    @abstractmethod
    def complete_document(self, doc_id: str, worker_id: str, success: bool, error_message: Optional[str] = None):
        """Mark document as completed (success or failure)."""
        pass

    @abstractmethod
    def release_document(self, doc_id: str, worker_id: str):
        """Release a claimed document back to the queue."""
        pass

    @abstractmethod
    def register_worker(self, worker_id: str, worker_info: Dict[str, Any]):
        """Register a worker in the system."""
        pass

    @abstractmethod
    def update_worker_heartbeat(self, worker_id: str):
        """Update worker heartbeat timestamp."""
        pass

    @abstractmethod
    def get_processing_status(self) -> Dict[str, Any]:
        """Get overall processing status."""
        pass

    @abstractmethod
    def cleanup_stale_claims(self, timeout_seconds: int = 300):
        """Release documents claimed by inactive workers."""
        pass

    @abstractmethod
    def is_document_queued(self, doc_id: str) -> bool:
        """Check if document is already in the queue."""
        pass

    @abstractmethod
    def elect_leader(self, worker_id: str, worker_info: Dict[str, Any]) -> bool:
        """Attempt to elect this worker as leader. Returns True if successful."""
        pass

    @abstractmethod
    def is_leader(self, worker_id: str) -> bool:
        """Check if the given worker is the current leader."""
        pass

    @abstractmethod
    def update_leader_heartbeat(self, worker_id: str):
        """Update leader heartbeat timestamp."""
        pass

    @abstractmethod
    def get_current_leader(self) -> Optional[Dict[str, Any]]:
        """Get information about the current leader, if any."""
        pass

    @abstractmethod
    def release_leadership(self, worker_id: str):
        """Release leadership role (for graceful shutdown)."""
        pass


class SimpleSQLiteJobControlDB(SimpleJobControlDB):
    """SQLite implementation of simplified job control database."""

    def __init__(self, config):
        self.config = config
        job_control_config = config.get_job_control_config()
        self.db_path = Path(job_control_config.get('path', './job_queue.db'))
        self.claim_timeout = job_control_config.get('claim_timeout', 300)
        self.heartbeat_interval = job_control_config.get('heartbeat_interval', 30)
        self.max_retries = job_control_config.get('max_retries', 3)

        # Ensure directory exists
        self.db_path.parent.mkdir(parents=True, exist_ok=True)

        self.initialize_schema()

    def _get_connection(self):
        """Get database connection with proper settings."""
        conn = sqlite3.connect(str(self.db_path), timeout=30.0)
        conn.row_factory = sqlite3.Row
        conn.execute("PRAGMA busy_timeout = 30000")  # 30 second timeout
        conn.execute("PRAGMA journal_mode = WAL")    # Better concurrency
        return conn

    def initialize_schema(self):
        """Create database schema if it doesn't exist."""
        schema_sql = """
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
            hostname TEXT,
            pid INTEGER,
            started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            last_heartbeat TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            status TEXT DEFAULT 'active',
            processed_count INTEGER DEFAULT 0,
            worker_info TEXT
        );

        CREATE TABLE IF NOT EXISTS leaders (
            id INTEGER PRIMARY KEY CHECK (id = 1),
            worker_id TEXT NOT NULL,
            elected_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            last_heartbeat TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            worker_info TEXT,
            FOREIGN KEY (worker_id) REFERENCES workers(worker_id)
        );

        CREATE INDEX IF NOT EXISTS idx_document_status ON document_queue(status);
        CREATE INDEX IF NOT EXISTS idx_document_claimed_at ON document_queue(claimed_at);
        CREATE INDEX IF NOT EXISTS idx_worker_heartbeat ON workers(last_heartbeat);
        """

        with self._get_connection() as conn:
            conn.executescript(schema_sql)
            conn.commit()

    def enqueue_document(self, doc_id: str, source: str, metadata: Dict[str, Any]):
        """Add a document to the processing queue."""
        with self._get_connection() as conn:
            conn.execute("""
                INSERT OR REPLACE INTO document_queue
                (doc_id, source, metadata, status, retry_count)
                VALUES (?, ?, ?, 'pending', 0)
            """, (doc_id, source, json.dumps(metadata)))
            conn.commit()

        logger.debug(f"Enqueued document {doc_id} from source {source}")

    def claim_next_document(self, worker_id: str) -> Optional[Dict[str, Any]]:
        """Atomically claim next available document for processing."""
        with self._get_connection() as conn:
            # Start transaction
            conn.execute("BEGIN IMMEDIATE")

            try:
                # Find next available document
                cursor = conn.execute("""
                    SELECT doc_id, source, metadata
                    FROM document_queue
                    WHERE status = 'pending'
                       AND retry_count < ?
                    ORDER BY created_at ASC
                    LIMIT 1
                """, (self.max_retries,))

                row = cursor.fetchone()
                if not row:
                    conn.rollback()
                    return None

                doc_id = row['doc_id']

                # Claim the document
                cursor = conn.execute("""
                    UPDATE document_queue
                    SET status = 'processing',
                        claimed_by = ?,
                        claimed_at = CURRENT_TIMESTAMP
                    WHERE doc_id = ? AND status = 'pending'
                """, (worker_id, doc_id))

                # Verify we actually claimed it
                if cursor.rowcount == 0:
                    conn.rollback()
                    return None

                conn.commit()

                logger.debug(f"Worker {worker_id} claimed document {doc_id}")

                return {
                    'doc_id': doc_id,
                    'source': row['source'],
                    'metadata': json.loads(row['metadata']) if row['metadata'] else {}
                }

            except Exception as e:
                conn.rollback()
                logger.error(f"Failed to claim document: {e}")
                return None

    def complete_document(self, doc_id: str, worker_id: str, success: bool, error_message: Optional[str] = None):
        """Mark document as completed (success or failure)."""
        status = 'completed' if success else 'failed'

        with self._get_connection() as conn:
            if success:
                conn.execute("""
                    UPDATE document_queue
                    SET status = ?, completed_at = CURRENT_TIMESTAMP, error_message = NULL
                    WHERE doc_id = ? AND claimed_by = ?
                """, (status, doc_id, worker_id))
            else:
                conn.execute("""
                    UPDATE document_queue
                    SET status = 'pending', claimed_by = NULL, claimed_at = NULL,
                        retry_count = retry_count + 1, error_message = ?
                    WHERE doc_id = ? AND claimed_by = ?
                """, (error_message, doc_id, worker_id))

            # Update worker processed count on success
            if success:
                conn.execute("""
                    UPDATE workers
                    SET processed_count = processed_count + 1,
                        last_heartbeat = CURRENT_TIMESTAMP
                    WHERE worker_id = ?
                """, (worker_id,))

            conn.commit()

        logger.debug(f"Document {doc_id} marked as {status} by worker {worker_id}")

    def release_document(self, doc_id: str, worker_id: str):
        """Release a claimed document back to the queue."""
        with self._get_connection() as conn:
            conn.execute("""
                UPDATE document_queue
                SET status = 'pending', claimed_by = NULL, claimed_at = NULL
                WHERE doc_id = ? AND claimed_by = ?
            """, (doc_id, worker_id))
            conn.commit()

        logger.debug(f"Document {doc_id} released by worker {worker_id}")

    def register_worker(self, worker_id: str, worker_info: Dict[str, Any]):
        """Register a worker in the system."""
        with self._get_connection() as conn:
            conn.execute("""
                INSERT OR REPLACE INTO workers
                (worker_id, hostname, pid, worker_info, last_heartbeat)
                VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
            """, (
                worker_id,
                socket.gethostname(),
                os.getpid(),
                json.dumps(worker_info)
            ))
            conn.commit()

        logger.info(f"Registered worker {worker_id}")

    def update_worker_heartbeat(self, worker_id: str):
        """Update worker heartbeat timestamp."""
        with self._get_connection() as conn:
            conn.execute("""
                UPDATE workers
                SET last_heartbeat = CURRENT_TIMESTAMP
                WHERE worker_id = ?
            """, (worker_id,))
            conn.commit()

    def get_processing_status(self) -> Dict[str, Any]:
        """Get overall processing status."""
        with self._get_connection() as conn:
            # Document counts by status
            cursor = conn.execute("""
                SELECT status, COUNT(*) as count
                FROM document_queue
                GROUP BY status
            """)
            doc_counts = {row['status']: row['count'] for row in cursor.fetchall()}

            # Active workers (heartbeat within 2x heartbeat interval)
            heartbeat_cutoff = datetime.now() - timedelta(seconds=self.heartbeat_interval * 2)
            cursor = conn.execute("""
                SELECT COUNT(*) as count
                FROM workers
                WHERE last_heartbeat > ?
            """, (heartbeat_cutoff,))
            active_workers = cursor.fetchone()['count']

            # Worker details
            cursor = conn.execute("""
                SELECT worker_id, hostname, pid, processed_count, last_heartbeat
                FROM workers
                WHERE last_heartbeat > ?
                ORDER BY last_heartbeat DESC
            """, (heartbeat_cutoff,))
            worker_details = [dict(row) for row in cursor.fetchall()]

            return {
                'document_counts': doc_counts,
                'active_workers': active_workers,
                'worker_details': worker_details,
                'timestamp': datetime.now().isoformat()
            }

    def cleanup_stale_claims(self, timeout_seconds: int = 300):
        """Release documents claimed by inactive workers."""
        cutoff_time = datetime.now() - timedelta(seconds=timeout_seconds)

        with self._get_connection() as conn:
            cursor = conn.execute("""
                UPDATE document_queue
                SET status = 'pending', claimed_by = NULL, claimed_at = NULL,
                    retry_count = retry_count + 1
                WHERE status = 'processing'
                  AND claimed_at < ?
            """, (cutoff_time,))

            released_count = cursor.rowcount
            conn.commit()

        if released_count > 0:
            logger.warning(f"Released {released_count} stale document claims")

        return released_count

    def is_document_queued(self, doc_id: str) -> bool:
        """Check if document is already in the queue."""
        with self._get_connection() as conn:
            cursor = conn.execute("""
                SELECT 1 FROM document_queue WHERE doc_id = ?
            """, (doc_id,))
            return cursor.fetchone() is not None

    def elect_leader(self, worker_id: str, worker_info: Dict[str, Any]) -> bool:
        """Attempt to elect this worker as leader. Returns True if successful."""
        with self._get_connection() as conn:
            try:
                conn.execute("BEGIN IMMEDIATE")

                # Check if there's already a leader with recent heartbeat
                cutoff_time = datetime.now() - timedelta(seconds=self.heartbeat_interval * 3)
                cursor = conn.execute("""
                    SELECT worker_id FROM leaders
                    WHERE last_heartbeat > ?
                """, (cutoff_time,))

                existing_leader = cursor.fetchone()
                if existing_leader:
                    conn.rollback()
                    return False

                # Attempt to become leader
                conn.execute("""
                    INSERT OR REPLACE INTO leaders (id, worker_id, worker_info)
                    VALUES (1, ?, ?)
                """, (worker_id, json.dumps(worker_info)))

                conn.commit()
                logger.info(f"Worker {worker_id} elected as leader")
                return True

            except Exception as e:
                conn.rollback()
                logger.error(f"Failed to elect leader: {e}")
                return False

    def is_leader(self, worker_id: str) -> bool:
        """Check if the given worker is the current leader."""
        with self._get_connection() as conn:
            cursor = conn.execute("""
                SELECT worker_id FROM leaders WHERE id = 1 AND worker_id = ?
            """, (worker_id,))
            return cursor.fetchone() is not None

    def update_leader_heartbeat(self, worker_id: str):
        """Update leader heartbeat timestamp."""
        with self._get_connection() as conn:
            conn.execute("""
                UPDATE leaders
                SET last_heartbeat = CURRENT_TIMESTAMP
                WHERE id = 1 AND worker_id = ?
            """, (worker_id,))
            conn.commit()

    def get_current_leader(self) -> Optional[Dict[str, Any]]:
        """Get information about the current leader, if any."""
        with self._get_connection() as conn:
            cursor = conn.execute("""
                SELECT worker_id, elected_at, last_heartbeat, worker_info
                FROM leaders WHERE id = 1
            """)

            row = cursor.fetchone()
            if not row:
                return None

            return {
                'worker_id': row['worker_id'],
                'elected_at': row['elected_at'],
                'last_heartbeat': row['last_heartbeat'],
                'worker_info': json.loads(row['worker_info']) if row['worker_info'] else {}
            }

    def release_leadership(self, worker_id: str):
        """Release leadership role (for graceful shutdown)."""
        with self._get_connection() as conn:
            conn.execute("""
                DELETE FROM leaders WHERE id = 1 AND worker_id = ?
            """, (worker_id,))
            conn.commit()
            logger.info(f"Worker {worker_id} released leadership")


class SimplePostgreSQLJobControlDB(SimpleJobControlDB):
    """PostgreSQL implementation - placeholder for future implementation."""

    def __init__(self, config):
        raise NotImplementedError("PostgreSQL backend not yet implemented")

    def initialize_schema(self):
        pass

    def enqueue_document(self, doc_id: str, source: str, metadata: Dict[str, Any]):
        pass

    def claim_next_document(self, worker_id: str) -> Optional[Dict[str, Any]]:
        pass

    def complete_document(self, doc_id: str, worker_id: str, success: bool, error_message: Optional[str] = None):
        pass

    def release_document(self, doc_id: str, worker_id: str):
        pass

    def register_worker(self, worker_id: str, worker_info: Dict[str, Any]):
        pass

    def update_worker_heartbeat(self, worker_id: str):
        pass

    def get_processing_status(self) -> Dict[str, Any]:
        pass

    def cleanup_stale_claims(self, timeout_seconds: int = 300):
        pass

    def is_document_queued(self, doc_id: str) -> bool:
        pass

    def elect_leader(self, worker_id: str, worker_info: Dict[str, Any]) -> bool:
        pass

    def is_leader(self, worker_id: str) -> bool:
        pass

    def update_leader_heartbeat(self, worker_id: str):
        pass

    def get_current_leader(self) -> Optional[Dict[str, Any]]:
        pass

    def release_leadership(self, worker_id: str):
        pass


class SimpleMySQLJobControlDB(SimpleJobControlDB):
    """MySQL implementation - placeholder for future implementation."""

    def __init__(self, config):
        raise NotImplementedError("MySQL backend not yet implemented")

    def initialize_schema(self):
        pass

    def enqueue_document(self, doc_id: str, source: str, metadata: Dict[str, Any]):
        pass

    def claim_next_document(self, worker_id: str) -> Optional[Dict[str, Any]]:
        pass

    def complete_document(self, doc_id: str, worker_id: str, success: bool, error_message: Optional[str] = None):
        pass

    def release_document(self, doc_id: str, worker_id: str):
        pass

    def register_worker(self, worker_id: str, worker_info: Dict[str, Any]):
        pass

    def update_worker_heartbeat(self, worker_id: str):
        pass

    def get_processing_status(self) -> Dict[str, Any]:
        pass

    def cleanup_stale_claims(self, timeout_seconds: int = 300):
        pass

    def is_document_queued(self, doc_id: str) -> bool:
        pass

    def elect_leader(self, worker_id: str, worker_info: Dict[str, Any]) -> bool:
        pass

    def is_leader(self, worker_id: str) -> bool:
        pass

    def update_leader_heartbeat(self, worker_id: str):
        pass

    def get_current_leader(self) -> Optional[Dict[str, Any]]:
        pass

    def release_leadership(self, worker_id: str):
        pass