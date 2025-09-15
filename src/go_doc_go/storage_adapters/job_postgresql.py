"""
PostgreSQL job storage adapter for OLTP operations.
"""

import json
import logging
import psycopg2
from psycopg2.extras import RealDictCursor
from datetime import datetime, timedelta
from typing import Dict, Any, List, Optional
from contextlib import contextmanager

from .base import JobStorage

logger = logging.getLogger(__name__)


class PostgreSQLJobStorage(JobStorage):
    """PostgreSQL implementation of job coordination storage."""
    
    def __init__(self, config: Dict[str, Any]):
        """
        Initialize PostgreSQL job storage.
        
        Args:
            config: Configuration with 'uri' or connection parameters
        """
        super().__init__(config)
        self.uri = config.get('uri')
        self.conn_params = {
            'host': config.get('host', 'localhost'),
            'port': config.get('port', 5432),
            'database': config.get('database', 'job_coordination'),
            'user': config.get('user', 'postgres'),
            'password': config.get('password')
        }
        self.conn = None
        self.initialize()
    
    @contextmanager
    def get_cursor(self):
        """Get a database cursor with automatic cleanup."""
        cursor = self.conn.cursor(cursor_factory=RealDictCursor)
        try:
            yield cursor
            self.conn.commit()
        except Exception:
            self.conn.rollback()
            raise
        finally:
            cursor.close()
    
    def initialize(self) -> None:
        """Initialize PostgreSQL connection and schema."""
        try:
            if self.uri:
                self.conn = psycopg2.connect(self.uri)
            else:
                self.conn = psycopg2.connect(**self.conn_params)
            
            # Create job coordination tables
            with self.get_cursor() as cursor:
                # Document queue table
                cursor.execute("""
                    CREATE TABLE IF NOT EXISTS job_documents (
                        doc_id VARCHAR(255) PRIMARY KEY,
                        run_id VARCHAR(255) NOT NULL,
                        status VARCHAR(50) DEFAULT 'pending',
                        worker_id VARCHAR(255),
                        claimed_at TIMESTAMP,
                        started_at TIMESTAMP,
                        completed_at TIMESTAMP,
                        failed_at TIMESTAMP,
                        retry_count INTEGER DEFAULT 0,
                        error_message TEXT,
                        metadata JSONB,
                        stats JSONB,
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
                
                # Worker heartbeats table
                cursor.execute("""
                    CREATE TABLE IF NOT EXISTS job_workers (
                        worker_id VARCHAR(255) PRIMARY KEY,
                        last_heartbeat TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                        status VARCHAR(50) DEFAULT 'active',
                        metadata JSONB
                    )
                """)
                
                logger.info("PostgreSQL job storage initialized")
                
        except Exception as e:
            logger.error(f"Failed to initialize PostgreSQL job storage: {e}")
            raise
    
    def claim_document(self, doc_id: str, worker_id: str, 
                      timeout: int = 300) -> Optional[Dict[str, Any]]:
        """Atomically claim a document for processing."""
        with self.get_cursor() as cursor:
            # Try to claim using SELECT FOR UPDATE SKIP LOCKED
            cursor.execute("""
                SELECT doc_id, run_id, metadata
                FROM job_documents
                WHERE doc_id = %s
                  AND status = 'pending'
                FOR UPDATE SKIP LOCKED
            """, (doc_id,))
            
            doc = cursor.fetchone()
            if doc:
                # Claim the document
                cursor.execute("""
                    UPDATE job_documents
                    SET status = 'processing',
                        worker_id = %s,
                        claimed_at = CURRENT_TIMESTAMP,
                        started_at = CURRENT_TIMESTAMP,
                        updated_at = CURRENT_TIMESTAMP
                    WHERE doc_id = %s
                """, (worker_id, doc_id))
                
                return dict(doc)
            
            # Check for stale claims
            cursor.execute("""
                SELECT doc_id, run_id, metadata
                FROM job_documents
                WHERE doc_id = %s
                  AND status = 'processing'
                  AND claimed_at < CURRENT_TIMESTAMP - INTERVAL '%s seconds'
                FOR UPDATE SKIP LOCKED
            """, (doc_id, timeout))
            
            doc = cursor.fetchone()
            if doc:
                # Reclaim stale document
                cursor.execute("""
                    UPDATE job_documents
                    SET worker_id = %s,
                        claimed_at = CURRENT_TIMESTAMP,
                        retry_count = retry_count + 1,
                        updated_at = CURRENT_TIMESTAMP
                    WHERE doc_id = %s
                """, (worker_id, doc_id))
                
                return dict(doc)
        
        return None
    
    def update_status(self, doc_id: str, status: str, 
                     metadata: Optional[Dict] = None) -> bool:
        """Update document processing status."""
        with self.get_cursor() as cursor:
            if metadata:
                cursor.execute("""
                    UPDATE job_documents
                    SET status = %s,
                        metadata = %s,
                        updated_at = CURRENT_TIMESTAMP
                    WHERE doc_id = %s
                """, (status, json.dumps(metadata), doc_id))
            else:
                cursor.execute("""
                    UPDATE job_documents
                    SET status = %s,
                        updated_at = CURRENT_TIMESTAMP
                    WHERE doc_id = %s
                """, (status, doc_id))
            
            return cursor.rowcount > 0
    
    def mark_completed(self, doc_id: str, stats: Dict[str, Any]) -> bool:
        """Mark document as successfully processed."""
        with self.get_cursor() as cursor:
            cursor.execute("""
                UPDATE job_documents
                SET status = 'completed',
                    completed_at = CURRENT_TIMESTAMP,
                    stats = %s,
                    updated_at = CURRENT_TIMESTAMP
                WHERE doc_id = %s
            """, (json.dumps(stats), doc_id))
            
            return cursor.rowcount > 0
    
    def mark_failed(self, doc_id: str, error: str, 
                   retry: bool = True) -> bool:
        """Mark document as failed."""
        with self.get_cursor() as cursor:
            if retry:
                # Check retry count
                cursor.execute("""
                    SELECT retry_count FROM job_documents
                    WHERE doc_id = %s
                """, (doc_id,))
                
                result = cursor.fetchone()
                if result and result['retry_count'] < 3:
                    # Schedule for retry
                    cursor.execute("""
                        UPDATE job_documents
                        SET status = 'pending',
                            worker_id = NULL,
                            retry_count = retry_count + 1,
                            error_message = %s,
                            updated_at = CURRENT_TIMESTAMP
                        WHERE doc_id = %s
                    """, (error, doc_id))
                else:
                    # Max retries exceeded
                    cursor.execute("""
                        UPDATE job_documents
                        SET status = 'failed',
                            failed_at = CURRENT_TIMESTAMP,
                            error_message = %s,
                            updated_at = CURRENT_TIMESTAMP
                        WHERE doc_id = %s
                    """, (error, doc_id))
            else:
                # No retry
                cursor.execute("""
                    UPDATE job_documents
                    SET status = 'failed',
                        failed_at = CURRENT_TIMESTAMP,
                        error_message = %s,
                        updated_at = CURRENT_TIMESTAMP
                    WHERE doc_id = %s
                """, (error, doc_id))
            
            return cursor.rowcount > 0
    
    def get_queue_status(self, run_id: str) -> Dict[str, int]:
        """Get current queue status."""
        with self.get_cursor() as cursor:
            cursor.execute("""
                SELECT 
                    COUNT(*) FILTER (WHERE status = 'pending') as pending,
                    COUNT(*) FILTER (WHERE status = 'processing') as processing,
                    COUNT(*) FILTER (WHERE status = 'completed') as completed,
                    COUNT(*) FILTER (WHERE status = 'failed') as failed,
                    COUNT(*) as total
                FROM job_documents
                WHERE run_id = %s
            """, (run_id,))
            
            result = cursor.fetchone()
            return dict(result) if result else {}
    
    def heartbeat(self, worker_id: str) -> bool:
        """Send worker heartbeat."""
        with self.get_cursor() as cursor:
            cursor.execute("""
                INSERT INTO job_workers (worker_id, last_heartbeat)
                VALUES (%s, CURRENT_TIMESTAMP)
                ON CONFLICT (worker_id)
                DO UPDATE SET last_heartbeat = CURRENT_TIMESTAMP
            """, (worker_id,))
            
            return cursor.rowcount > 0
    
    def cleanup(self, older_than: Optional[datetime] = None) -> int:
        """Clean up old job data."""
        if not older_than:
            # Default to cleaning up data older than TTL
            if self.ttl:
                older_than = datetime.now() - timedelta(seconds=self.ttl)
            else:
                older_than = datetime.now() - timedelta(days=1)
        
        with self.get_cursor() as cursor:
            cursor.execute("""
                DELETE FROM job_documents
                WHERE (status = 'completed' OR status = 'failed')
                  AND updated_at < %s
            """, (older_than,))
            
            deleted = cursor.rowcount
            
            # Clean up old worker records
            cursor.execute("""
                DELETE FROM job_workers
                WHERE last_heartbeat < %s
            """, (older_than,))
            
            logger.info(f"Cleaned up {deleted} old job records")
            return deleted
    
    def close(self) -> None:
        """Close PostgreSQL connection."""
        if self.conn:
            self.conn.close()
            self.conn = None