"""
SQLite implementation of simplified job control database.
"""

import json
import logging
import random
import sqlite3
import time
import uuid
from datetime import datetime
from pathlib import Path
from typing import Dict, Any, Optional, List

from .base import SimpleJobControlDB, retry_on_db_contention

logger = logging.getLogger(__name__)


def retry_on_sqlite_lock(max_retries=5, initial_delay=0.1):
    """Retry decorator for SQLite database lock errors."""
    def decorator(func):
        def wrapper(*args, **kwargs):
            for attempt in range(max_retries):
                try:
                    return func(*args, **kwargs)
                except sqlite3.OperationalError as e:
                    if "database is locked" in str(e) and attempt < max_retries - 1:
                        delay = initial_delay * (2 ** attempt)  # Exponential backoff
                        jitter = random.uniform(0, 0.1)  # Add jitter
                        sleep_time = delay + jitter
                        logger.warning(f"Database locked, retrying in {sleep_time:.2f}s (attempt {attempt + 1}/{max_retries})")
                        time.sleep(sleep_time)
                        continue
                    raise
            return func(*args, **kwargs)
        return wrapper
    return decorator


class SimpleSQLiteJobControlDB(SimpleJobControlDB):
    """SQLite implementation of simplified job control database."""

    def __init__(self, config):
        self.config = config
        job_control_config = config.get_job_control_config()
        self.db_path = Path(job_control_config.get('path', './job_queue.db'))
        self.claim_timeout = job_control_config.get('claim_timeout', 300)
        self.heartbeat_interval = job_control_config.get('heartbeat_interval', 30)
        self.max_retries = job_control_config.get('max_retries', 3)

        # Retry configuration for initialization
        self.init_retries = job_control_config.get('init_retries', 5)
        self.init_retry_delay = job_control_config.get('init_retry_delay', 0.1)

        # Ensure directory exists
        self.db_path.parent.mkdir(parents=True, exist_ok=True)

        self.initialize_schema()

    def _is_retryable_error(self, error: Exception) -> bool:
        """SQLite-specific retryable error detection"""
        return (isinstance(error, sqlite3.OperationalError) and
                "database is locked" in str(error))

    @retry_on_sqlite_lock(max_retries=3, initial_delay=0.1)
    def _get_connection(self):
        """Get database connection with enhanced error handling."""
        conn = sqlite3.connect(str(self.db_path), timeout=30.0)
        conn.row_factory = sqlite3.Row
        conn.execute("PRAGMA busy_timeout = 30000")  # 30 second timeout
        conn.execute("PRAGMA journal_mode = WAL")    # Better concurrency
        return conn

    def _get_raw_connection(self):
        """Get raw database connection without retry decorators for schema initialization."""
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

        CREATE TABLE IF NOT EXISTS source_leaders (
            source_name TEXT PRIMARY KEY,
            worker_id TEXT NOT NULL,
            elected_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            last_heartbeat TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            last_discovery TIMESTAMP,
            worker_info TEXT,
            FOREIGN KEY (worker_id) REFERENCES workers(worker_id)
        );

        CREATE TABLE IF NOT EXISTS document_metadata (
            doc_id TEXT PRIMARY KEY,
            source TEXT NOT NULL,
            last_modified REAL,
            content_hash TEXT,
            file_size INTEGER,
            last_processed_at TIMESTAMP,
            processing_stats TEXT
        );

        CREATE INDEX IF NOT EXISTS idx_document_status ON document_queue(status);
        CREATE INDEX IF NOT EXISTS idx_document_claimed_at ON document_queue(claimed_at);
        CREATE INDEX IF NOT EXISTS idx_worker_heartbeat ON workers(last_heartbeat);
        CREATE INDEX IF NOT EXISTS idx_document_metadata_source ON document_metadata(source);
        CREATE INDEX IF NOT EXISTS idx_document_metadata_last_modified ON document_metadata(last_modified);
        CREATE INDEX IF NOT EXISTS idx_source_leaders_heartbeat ON source_leaders(last_heartbeat);
        CREATE INDEX IF NOT EXISTS idx_source_leaders_worker ON source_leaders(worker_id);
        """

        # Manual retry logic for schema initialization to avoid nested decorator issues
        for attempt in range(self.init_retries):
            conn = None
            try:
                conn = self._get_raw_connection()
                conn.executescript(schema_sql)
                conn.commit()

                # Verify the tables were created
                cursor = conn.execute("SELECT name FROM sqlite_master WHERE type='table' AND name='workers'")
                if not cursor.fetchone():
                    raise sqlite3.OperationalError("Failed to create workers table")

                logger.info(f"Successfully initialized database schema at {self.db_path}")
                conn.close()
                return

            except sqlite3.OperationalError as e:
                if conn:
                    conn.close()
                if "database is locked" in str(e) and attempt < self.init_retries - 1:
                    delay = self.init_retry_delay * (2 ** attempt)
                    jitter = random.uniform(0, 0.1)
                    sleep_time = delay + jitter
                    logger.warning(f"Database locked during schema init, retrying in {sleep_time:.2f}s (attempt {attempt + 1}/{self.init_retries})")
                    time.sleep(sleep_time)
                    continue
                else:
                    logger.error(f"Failed to initialize schema after {attempt + 1} attempts: {e}")
                    raise
            except Exception as e:
                if conn:
                    conn.close()
                logger.error(f"Unexpected error during schema initialization: {e}")
                raise

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
                # Find oldest available document
                cursor = conn.execute("""
                    SELECT doc_id, source, metadata
                    FROM document_queue
                    WHERE status = 'pending' AND retry_count < ?
                    ORDER BY created_at ASC
                    LIMIT 1
                """, (self.max_retries,))

                row = cursor.fetchone()
                if not row:
                    conn.rollback()
                    return None

                doc_id = row['doc_id']

                # Try to claim it
                result = conn.execute("""
                    UPDATE document_queue
                    SET status = 'processing',
                        claimed_by = ?,
                        claimed_at = CURRENT_TIMESTAMP
                    WHERE doc_id = ? AND status = 'pending'
                """, (worker_id, doc_id))

                if result.rowcount == 0:
                    # Someone else claimed it
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
                raise

    def complete_document(self, doc_id: str, worker_id: str, success: bool, error_message: Optional[str] = None):
        """Mark a document as completed (successfully or failed)."""
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
                    SET status = ?, retry_count = retry_count + 1, error_message = ?
                    WHERE doc_id = ? AND claimed_by = ?
                """, ('pending' if error_message else 'failed', error_message, doc_id, worker_id))

            # Update worker processed count if successful
            if success:
                conn.execute("""
                    UPDATE workers
                    SET processed_count = processed_count + 1
                    WHERE worker_id = ?
                """, (worker_id,))

            conn.commit()

        logger.debug(f"Worker {worker_id} completed document {doc_id} with status {status}")

    def release_document(self, doc_id: str, worker_id: str):
        """Release a claimed document back to the queue."""
        with self._get_connection() as conn:
            conn.execute("""
                UPDATE document_queue
                SET status = 'pending', claimed_by = NULL, claimed_at = NULL
                WHERE doc_id = ? AND claimed_by = ?
            """, (doc_id, worker_id))
            conn.commit()

        logger.debug(f"Worker {worker_id} released document {doc_id}")

    def register_worker(self, worker_id: str, worker_info: Dict[str, Any]):
        """Register a worker in the system."""
        try:
            with self._get_connection() as conn:
                conn.execute("""
                    INSERT OR REPLACE INTO workers
                    (worker_id, hostname, pid, worker_info, last_heartbeat)
                    VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
                """, (
                    worker_id,
                    worker_info.get('hostname', ''),
                    worker_info.get('pid', 0),
                    json.dumps(worker_info)
                ))
                conn.commit()
        except sqlite3.OperationalError as e:
            if "no such table: workers" in str(e):
                # Schema initialization may have failed, retry
                logger.warning("Workers table not found, re-initializing schema")
                self.initialize_schema()
                # Retry the operation
                with self._get_connection() as conn:
                    conn.execute("""
                        INSERT OR REPLACE INTO workers
                        (worker_id, hostname, pid, worker_info, last_heartbeat)
                        VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
                    """, (
                        worker_id,
                        worker_info.get('hostname', ''),
                        worker_info.get('pid', 0),
                        json.dumps(worker_info)
                    ))
                    conn.commit()
            else:
                raise

        logger.debug(f"Registered worker {worker_id}")

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
            cursor = conn.execute("""
                SELECT COUNT(*) as count
                FROM workers
                WHERE datetime(last_heartbeat) > datetime('now', '-{} seconds')
            """.format(self.heartbeat_interval * 2))

            active_workers = cursor.fetchone()['count']

            return {
                'documents': doc_counts,
                'active_workers': active_workers,
                'queue_status': 'idle' if doc_counts.get('pending', 0) == 0 and doc_counts.get('processing', 0) == 0 else 'active'
            }

    def cleanup_stale_claims(self, timeout_seconds: int = 300):
        """Release documents claimed by inactive workers."""
        with self._get_connection() as conn:
            result = conn.execute("""
                UPDATE document_queue
                SET status = 'pending', claimed_by = NULL, claimed_at = NULL
                WHERE status = 'processing'
                  AND claimed_at < datetime('now', '-{} seconds')
            """.format(timeout_seconds))

            count = result.rowcount
            conn.commit()

            if count > 0:
                logger.info(f"Released {count} stale document claims")

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

                # Check if leader already exists
                cursor = conn.execute("SELECT worker_id FROM leaders WHERE id = 1")
                existing_leader = cursor.fetchone()

                if existing_leader:
                    # Leader exists, check if it's stale
                    cursor = conn.execute("""
                        SELECT worker_id FROM leaders
                        WHERE id = 1 AND datetime(last_heartbeat) > datetime('now', '-{} seconds')
                    """.format(self.heartbeat_interval * 3))

                    if cursor.fetchone():
                        # Active leader exists
                        conn.rollback()
                        return False

                    # Stale leader, replace it
                    conn.execute("""
                        DELETE FROM leaders WHERE id = 1
                    """)

                # Insert as new leader
                conn.execute("""
                    INSERT INTO leaders (id, worker_id, worker_info)
                    VALUES (1, ?, ?)
                """, (worker_id, json.dumps(worker_info)))

                conn.commit()
                return True

            except Exception as e:
                conn.rollback()
                logger.debug(f"Leader election failed for {worker_id}: {e}")
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

    def store_document_metadata(self, doc_id: str, source: str,
                               last_modified: Optional[float] = None,
                               content_hash: Optional[str] = None,
                               file_size: Optional[int] = None,
                               processing_stats: Optional[Dict[str, Any]] = None):
        """Store document metadata for change tracking."""
        with self._get_connection() as conn:
            conn.execute("""
                INSERT OR REPLACE INTO document_metadata
                (doc_id, source, last_modified, content_hash, file_size,
                 last_processed_at, processing_stats)
                VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP, ?)
            """, (doc_id, source, last_modified, content_hash, file_size,
                  json.dumps(processing_stats) if processing_stats else None))
            conn.commit()

    def get_document_metadata(self, doc_id: str) -> Optional[Dict[str, Any]]:
        """Get stored document metadata."""
        with self._get_connection() as conn:
            cursor = conn.execute("""
                SELECT * FROM document_metadata WHERE doc_id = ?
            """, (doc_id,))

            row = cursor.fetchone()
            if not row:
                return None

            metadata = dict(row)
            if metadata.get('processing_stats'):
                metadata['processing_stats'] = json.loads(metadata['processing_stats'])

            return metadata

    def has_document_changed(self, doc_id: str, source: str,
                            current_modified: Optional[float] = None,
                            current_hash: Optional[str] = None) -> bool:
        """Check if document has changed since last processing."""
        stored_metadata = self.get_document_metadata(doc_id)

        if not stored_metadata:
            # No metadata stored - treat as changed (new document)
            return True

        # Check modification time if provided
        if current_modified is not None and stored_metadata.get('last_modified') is not None:
            if current_modified > stored_metadata['last_modified']:
                return True

        # Check content hash if provided
        if current_hash is not None and stored_metadata.get('content_hash') is not None:
            if current_hash != stored_metadata['content_hash']:
                return True

        # No change detected
        return False

    def get_source_documents(self, source: str, pattern: Optional[str] = None) -> List[Dict[str, Any]]:
        """Get all document metadata for a specific source.

        Args:
            source: The content source name
            pattern: Optional SQL LIKE pattern for filtering doc_ids

        Returns:
            List of document metadata dictionaries
        """
        with self._get_connection() as conn:
            if pattern:
                cursor = conn.execute("""
                    SELECT * FROM document_metadata
                    WHERE source = ? AND doc_id LIKE ?
                    ORDER BY last_processed_at DESC
                """, (source, pattern))
            else:
                cursor = conn.execute("""
                    SELECT * FROM document_metadata
                    WHERE source = ?
                    ORDER BY last_processed_at DESC
                """, (source,))

            documents = []
            for row in cursor.fetchall():
                metadata = dict(row)
                if metadata.get('processing_stats'):
                    metadata['processing_stats'] = json.loads(metadata['processing_stats'])
                documents.append(metadata)

            return documents

    def get_document_statistics(self) -> Dict[str, Any]:
        """Get overall document processing statistics.

        Returns:
            Dictionary with statistics including:
            - total_documents: Total number of documents
            - documents_by_source: Count per source
            - recently_processed: Documents processed in last hour
            - documents_with_changes: Documents with multiple versions
        """
        with self._get_connection() as conn:
            # Total documents
            total_docs = conn.execute(
                "SELECT COUNT(*) FROM document_metadata"
            ).fetchone()[0]

            # Documents by source
            cursor = conn.execute("""
                SELECT source, COUNT(*) as count
                FROM document_metadata
                GROUP BY source
            """)
            docs_by_source = dict(cursor.fetchall())

            # Recently processed (last hour)
            recent_count = conn.execute("""
                SELECT COUNT(*) FROM document_metadata
                WHERE last_processed_at > datetime('now', '-1 hour')
            """).fetchone()[0]

            # Documents with multiple processing (could indicate changes)
            # This would require a more complex query with history tracking
            # For now, we return 0 as we don't track full history
            docs_with_changes = 0

            return {
                'total_documents': total_docs,
                'documents_by_source': docs_by_source,
                'recently_processed': recent_count,
                'documents_with_changes': docs_with_changes
            }

    # Source-specific leadership methods

    @retry_on_db_contention(max_retries=5, initial_delay=0.1)
    def elect_source_leader(self, source_name: str, worker_id: str, worker_info: Dict[str, Any]) -> bool:
        """Elect leader for specific content source"""
        with self._get_connection() as conn:
            try:
                conn.execute("BEGIN IMMEDIATE")

                # Check for existing active leader
                cursor = conn.execute("""
                    SELECT worker_id FROM source_leaders
                    WHERE source_name = ? AND datetime(last_heartbeat) > datetime('now', '-{} seconds')
                """.format(self.heartbeat_interval * 3), (source_name,))

                if cursor.fetchone():
                    conn.rollback()
                    return False  # Active leader exists

                # Elect this worker as leader
                conn.execute("""
                    INSERT OR REPLACE INTO source_leaders
                    (source_name, worker_id, worker_info, elected_at, last_heartbeat)
                    VALUES (?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
                """, (source_name, worker_id, json.dumps(worker_info)))

                conn.commit()
                return True

            except Exception as e:
                conn.rollback()
                logger.debug(f"Source leader election failed for {worker_id}/{source_name}: {e}")
                return False

    @retry_on_db_contention(max_retries=3, initial_delay=0.05)
    def get_source_leader(self, source_name: str) -> Optional[Dict[str, Any]]:
        """Get current leader for specific source"""
        with self._get_connection() as conn:
            cursor = conn.execute("""
                SELECT worker_id, elected_at, last_heartbeat, last_discovery, worker_info
                FROM source_leaders
                WHERE source_name = ?
            """, (source_name,))

            row = cursor.fetchone()
            if not row:
                return None

            return {
                'source_name': source_name,
                'worker_id': row['worker_id'],
                'elected_at': row['elected_at'],
                'last_heartbeat': row['last_heartbeat'],
                'last_discovery': row['last_discovery'],
                'worker_info': json.loads(row['worker_info']) if row['worker_info'] else {}
            }

    @retry_on_db_contention(max_retries=3, initial_delay=0.05)
    def get_all_source_leaders(self) -> Dict[str, Dict[str, Any]]:
        """Get all source leadership information"""
        with self._get_connection() as conn:
            cursor = conn.execute("""
                SELECT source_name, worker_id, elected_at, last_heartbeat, last_discovery, worker_info
                FROM source_leaders
            """)

            leaders = {}
            for row in cursor.fetchall():
                leaders[row['source_name']] = {
                    'worker_id': row['worker_id'],
                    'elected_at': row['elected_at'],
                    'last_heartbeat': row['last_heartbeat'],
                    'last_discovery': row['last_discovery'],
                    'worker_info': json.loads(row['worker_info']) if row['worker_info'] else {}
                }

            return leaders

    @retry_on_db_contention(max_retries=3, initial_delay=0.05)
    def update_source_leader_heartbeat(self, source_name: str, worker_id: str):
        """Update heartbeat for source leader"""
        with self._get_connection() as conn:
            conn.execute("""
                UPDATE source_leaders
                SET last_heartbeat = CURRENT_TIMESTAMP
                WHERE source_name = ? AND worker_id = ?
            """, (source_name, worker_id))
            conn.commit()

    @retry_on_db_contention(max_retries=3, initial_delay=0.05)
    def release_source_leadership(self, source_name: str, worker_id: str):
        """Release leadership for specific source"""
        with self._get_connection() as conn:
            conn.execute("""
                DELETE FROM source_leaders
                WHERE source_name = ? AND worker_id = ?
            """, (source_name, worker_id))
            conn.commit()
            logger.info(f"Worker {worker_id} released leadership for source {source_name}")

    @retry_on_db_contention(max_retries=3, initial_delay=0.05)
    def get_worker_source_leaderships(self, worker_id: str) -> List[str]:
        """Get all sources this worker leads"""
        with self._get_connection() as conn:
            cursor = conn.execute("""
                SELECT source_name FROM source_leaders
                WHERE worker_id = ?
            """, (worker_id,))

            return [row['source_name'] for row in cursor.fetchall()]