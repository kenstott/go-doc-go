"""
SQLite job storage adapter for single-machine processing.
"""

import json
import logging
import sqlite3
import threading
from datetime import datetime, timedelta
from typing import Dict, Any, List, Optional
from pathlib import Path

from .base import JobStorage

logger = logging.getLogger(__name__)


class SQLiteJobStorage(JobStorage):
    """SQLite implementation of job storage for local processing."""
    
    def __init__(self, config: Dict[str, Any]):
        """
        Initialize SQLite job storage.
        
        Args:
            config: Configuration with path to database file
        """
        super().__init__(config)
        
        # Get database path
        self.db_path = config.get('path', 'jobs.db')
        
        # Ensure parent directory exists
        Path(self.db_path).parent.mkdir(parents=True, exist_ok=True)
        
        # Thread-local storage for connections
        self._local = threading.local()
        
        # Initialize database schema
        self.initialize()
    
    def _get_connection(self) -> sqlite3.Connection:
        """Get thread-local database connection."""
        if not hasattr(self._local, 'conn'):
            self._local.conn = sqlite3.connect(
                self.db_path,
                isolation_level='IMMEDIATE',  # For better concurrency
                timeout=30.0
            )
            self._local.conn.row_factory = sqlite3.Row
        return self._local.conn
    
    def initialize(self) -> None:
        """Initialize database schema."""
        conn = self._get_connection()
        cursor = conn.cursor()
        
        try:
            # Create job documents table
            cursor.execute("""
                CREATE TABLE IF NOT EXISTS job_documents (
                    doc_id TEXT PRIMARY KEY,
                    run_id TEXT NOT NULL,
                    status TEXT DEFAULT 'pending',
                    worker_id TEXT,
                    claimed_at TIMESTAMP,
                    completed_at TIMESTAMP,
                    error_message TEXT,
                    metadata TEXT,
                    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
                )
            """)
            
            # Create indexes
            cursor.execute("""
                CREATE INDEX IF NOT EXISTS idx_job_docs_status 
                ON job_documents(status, run_id)
            """)
            
            cursor.execute("""
                CREATE INDEX IF NOT EXISTS idx_job_docs_worker 
                ON job_documents(worker_id, status)
            """)
            
            # Create processing runs table
            cursor.execute("""
                CREATE TABLE IF NOT EXISTS processing_runs (
                    run_id TEXT PRIMARY KEY,
                    config TEXT,
                    status TEXT DEFAULT 'active',
                    total_documents INTEGER DEFAULT 0,
                    processed_documents INTEGER DEFAULT 0,
                    failed_documents INTEGER DEFAULT 0,
                    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
                )
            """)
            
            conn.commit()
            logger.info(f"SQLite job storage initialized: {self.db_path}")
            
        except Exception as e:
            conn.rollback()
            logger.error(f"Failed to initialize SQLite job storage: {e}")
            raise
    
    def add_document(self, doc_id: str, run_id: str, 
                    metadata: Optional[Dict[str, Any]] = None) -> bool:
        """Add document to job queue."""
        conn = self._get_connection()
        cursor = conn.cursor()
        
        try:
            metadata_json = json.dumps(metadata) if metadata else None
            
            cursor.execute("""
                INSERT OR IGNORE INTO job_documents 
                (doc_id, run_id, status, metadata)
                VALUES (?, ?, 'pending', ?)
            """, (doc_id, run_id, metadata_json))
            
            conn.commit()
            return cursor.rowcount > 0
            
        except Exception as e:
            conn.rollback()
            logger.error(f"Error adding document {doc_id}: {e}")
            return False
    
    def claim_document(self, doc_id: str, worker_id: str, 
                      timeout: int = 300) -> Optional[Dict[str, Any]]:
        """Claim a specific document for processing."""
        conn = self._get_connection()
        cursor = conn.cursor()
        
        try:
            # Try to claim the document atomically
            cursor.execute("""
                UPDATE job_documents
                SET status = 'processing',
                    worker_id = ?,
                    claimed_at = CURRENT_TIMESTAMP,
                    updated_at = CURRENT_TIMESTAMP
                WHERE doc_id = ?
                AND status = 'pending'
            """, (worker_id, doc_id))
            
            if cursor.rowcount == 0:
                return None
            
            # Get the claimed document
            cursor.execute("""
                SELECT doc_id, run_id, metadata
                FROM job_documents
                WHERE doc_id = ?
            """, (doc_id,))
            
            row = cursor.fetchone()
            conn.commit()
            
            if row:
                return {
                    'doc_id': row['doc_id'],
                    'run_id': row['run_id'],
                    'metadata': json.loads(row['metadata']) if row['metadata'] else {}
                }
            
            return None
            
        except Exception as e:
            conn.rollback()
            logger.error(f"Error claiming document {doc_id}: {e}")
            return None
    
    def claim_next_document(self, run_id: str, worker_id: str,
                          timeout: int = 300) -> Optional[Dict[str, Any]]:
        """Claim next available document from queue."""
        conn = self._get_connection()
        cursor = conn.cursor()
        
        try:
            # Find and claim next available document
            cursor.execute("""
                UPDATE job_documents
                SET status = 'processing',
                    worker_id = ?,
                    claimed_at = CURRENT_TIMESTAMP,
                    updated_at = CURRENT_TIMESTAMP
                WHERE doc_id IN (
                    SELECT doc_id FROM job_documents
                    WHERE run_id = ?
                    AND status = 'pending'
                    LIMIT 1
                )
            """, (worker_id, run_id))
            
            if cursor.rowcount == 0:
                # Check for expired claims
                expiry_time = datetime.now() - timedelta(seconds=timeout)
                cursor.execute("""
                    UPDATE job_documents
                    SET status = 'pending',
                        worker_id = NULL,
                        claimed_at = NULL,
                        updated_at = CURRENT_TIMESTAMP
                    WHERE status = 'processing'
                    AND claimed_at < ?
                    AND run_id = ?
                """, (expiry_time, run_id))
                
                if cursor.rowcount > 0:
                    conn.commit()
                    # Retry claiming
                    return self.claim_next_document(run_id, worker_id, timeout)
                
                return None
            
            # Get the claimed document
            cursor.execute("""
                SELECT doc_id, run_id, metadata
                FROM job_documents
                WHERE worker_id = ?
                AND status = 'processing'
                ORDER BY updated_at DESC
                LIMIT 1
            """, (worker_id,))
            
            row = cursor.fetchone()
            conn.commit()
            
            if row:
                return {
                    'doc_id': row['doc_id'],
                    'run_id': row['run_id'],
                    'metadata': json.loads(row['metadata']) if row['metadata'] else {}
                }
            
            return None
            
        except Exception as e:
            conn.rollback()
            logger.error(f"Error claiming next document: {e}")
            return None
    
    def update_status(self, doc_id: str, status: str, 
                     metadata: Optional[Dict] = None) -> bool:
        """Update document processing status."""
        conn = self._get_connection()
        cursor = conn.cursor()
        
        try:
            metadata_json = json.dumps(metadata) if metadata else None
            cursor.execute("""
                UPDATE job_documents
                SET status = ?,
                    metadata = COALESCE(?, metadata),
                    updated_at = CURRENT_TIMESTAMP
                WHERE doc_id = ?
            """, (status, metadata_json, doc_id))
            
            conn.commit()
            return cursor.rowcount > 0
            
        except Exception as e:
            conn.rollback()
            logger.error(f"Error updating status for {doc_id}: {e}")
            return False
    
    def mark_completed(self, doc_id: str, stats: Dict[str, Any]) -> bool:
        """Mark document as successfully processed."""
        conn = self._get_connection()
        cursor = conn.cursor()
        
        try:
            stats_json = json.dumps(stats)
            cursor.execute("""
                UPDATE job_documents
                SET status = 'completed',
                    metadata = ?,
                    completed_at = CURRENT_TIMESTAMP,
                    updated_at = CURRENT_TIMESTAMP
                WHERE doc_id = ?
            """, (stats_json, doc_id))
            
            conn.commit()
            return cursor.rowcount > 0
            
        except Exception as e:
            conn.rollback()
            logger.error(f"Error marking document {doc_id} as completed: {e}")
            return False
    
    def mark_failed(self, doc_id: str, error: str, 
                   retry: bool = True) -> bool:
        """Mark document as failed."""
        conn = self._get_connection()
        cursor = conn.cursor()
        
        try:
            new_status = 'pending' if retry else 'failed'
            cursor.execute("""
                UPDATE job_documents
                SET status = ?,
                    error_message = ?,
                    updated_at = CURRENT_TIMESTAMP
                WHERE doc_id = ?
            """, (new_status, error, doc_id))
            
            conn.commit()
            return cursor.rowcount > 0
            
        except Exception as e:
            conn.rollback()
            logger.error(f"Error marking document {doc_id} as failed: {e}")
            return False
    
    def release_document(self, doc_id: str, worker_id: str) -> bool:
        """Release claimed document back to queue."""
        conn = self._get_connection()
        cursor = conn.cursor()
        
        try:
            cursor.execute("""
                UPDATE job_documents
                SET status = 'pending',
                    worker_id = NULL,
                    claimed_at = NULL,
                    updated_at = CURRENT_TIMESTAMP
                WHERE doc_id = ?
                AND worker_id = ?
                AND status = 'processing'
            """, (doc_id, worker_id))
            
            conn.commit()
            return cursor.rowcount > 0
            
        except Exception as e:
            conn.rollback()
            logger.error(f"Error releasing document {doc_id}: {e}")
            return False
    
    def heartbeat(self, worker_id: str) -> bool:
        """Send worker heartbeat."""
        conn = self._get_connection()
        cursor = conn.cursor()
        
        try:
            # Update heartbeat for all documents claimed by this worker
            cursor.execute("""
                UPDATE job_documents
                SET updated_at = CURRENT_TIMESTAMP
                WHERE worker_id = ?
                AND status = 'processing'
            """, (worker_id,))
            
            conn.commit()
            return True
            
        except Exception as e:
            conn.rollback()
            logger.error(f"Error updating heartbeat for worker {worker_id}: {e}")
            return False
    
    def get_queue_status(self, run_id: str) -> Dict[str, int]:
        """Get current queue status."""
        conn = self._get_connection()
        cursor = conn.cursor()
        
        try:
            stats = {}
            
            # Count documents by status
            cursor.execute("""
                SELECT status, COUNT(*) as count
                FROM job_documents
                WHERE run_id = ?
                GROUP BY status
            """, (run_id,))
            
            for row in cursor.fetchall():
                stats[row['status']] = row['count']
            
            # Add total count
            stats['total'] = sum(stats.values())
            
            return stats
            
        except Exception as e:
            logger.error(f"Error getting queue status: {e}")
            return {}
    
    def cleanup(self, older_than: Optional[datetime] = None) -> int:
        """Clean up old job data."""
        conn = self._get_connection()
        cursor = conn.cursor()
        
        try:
            if older_than is None:
                # Default to cleaning up completed/failed jobs older than 7 days
                older_than = datetime.now() - timedelta(days=7)
            
            # Clean up completed and failed documents
            cursor.execute("""
                DELETE FROM job_documents
                WHERE status IN ('completed', 'failed')
                AND updated_at < ?
            """, (older_than,))
            
            deleted_docs = cursor.rowcount
            
            # Clean up old processing runs
            cursor.execute("""
                DELETE FROM processing_runs
                WHERE status = 'completed'
                AND updated_at < ?
            """, (older_than,))
            
            deleted_runs = cursor.rowcount
            
            conn.commit()
            logger.info(f"Cleaned up {deleted_docs} documents and {deleted_runs} runs")
            return deleted_docs
            
        except Exception as e:
            conn.rollback()
            logger.error(f"Error during cleanup: {e}")
            return 0
    
    def close(self) -> None:
        """Close database connection."""
        if hasattr(self._local, 'conn'):
            self._local.conn.close()
            delattr(self._local, 'conn')