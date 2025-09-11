"""
Work queue implementation for distributed document processing.
"""

import hashlib
import json
import logging
import platform
import socket
import time
import uuid
from datetime import datetime, timedelta
from typing import Dict, Any, Optional, List, Tuple

logger = logging.getLogger(__name__)


class RunCoordinator:
    """Manages processing runs based on configuration hash."""
    
    def __init__(self, db):
        """
        Initialize run coordinator.
        
        Args:
            db: Database connection (must support transactions)
        """
        self.db = db
    
    @staticmethod
    def get_run_id_from_config(config: Dict[str, Any]) -> str:
        """
        Generate deterministic run ID from configuration.
        
        Args:
            config: Configuration dictionary
            
        Returns:
            16-character run ID (first 16 chars of SHA256 hash)
        """
        # Extract only the parts that affect processing
        processing_config = {
            'content_sources': config.get('content_sources', []),
            'storage': config.get('storage', {}),
            'embedding': config.get('embedding', {}),
            'relationship_detection': config.get('relationship_detection', {}),
            # Don't include worker-specific settings like ports, log levels
        }
        
        # Sort keys for deterministic hashing
        config_str = json.dumps(processing_config, sort_keys=True)
        
        # Create hash - use first 16 chars for readability
        full_hash = hashlib.sha256(config_str.encode()).hexdigest()
        return full_hash[:16]
    
    def ensure_run_exists(self, run_id: str, config: Dict[str, Any]) -> Dict[str, Any]:
        """
        Ensure a processing run exists, creating if necessary.
        
        Args:
            run_id: Run ID (from get_run_id_from_config)
            config: Full configuration
            
        Returns:
            Run information dictionary
        """
        config_str = json.dumps(config, sort_keys=True)
        full_hash = hashlib.sha256(config_str.encode()).hexdigest()
        
        with self.db.transaction():
            # Try to get existing run
            existing = self.db.execute("""
                SELECT run_id, status, created_at, worker_count
                FROM processing_runs
                WHERE run_id = %s
            """, (run_id,))
            
            if existing:
                # Update last activity
                self.db.execute("""
                    UPDATE processing_runs
                    SET last_activity_at = CURRENT_TIMESTAMP
                    WHERE run_id = %s
                """, (run_id,))
                return existing
            
            # Create new run
            self.db.execute("""
                INSERT INTO processing_runs (
                    run_id, config_hash, config_snapshot, status
                ) VALUES (%s, %s, %s, 'active')
                ON CONFLICT (run_id) DO NOTHING
            """, (run_id, full_hash, json.dumps(config)))
            
            logger.info(f"Created new processing run: {run_id}")
            
            return {
                'run_id': run_id,
                'status': 'active',
                'created_at': datetime.now(),
                'worker_count': 0
            }
    
    def register_worker(self, run_id: str, worker_id: str, 
                       metadata: Optional[Dict] = None) -> None:
        """
        Register a worker for a processing run.
        
        Args:
            run_id: Run ID
            worker_id: Worker ID
            metadata: Optional worker metadata
        """
        hostname = socket.gethostname()
        process_id = platform.os.getpid() if hasattr(platform.os, 'getpid') else None
        
        with self.db.transaction():
            # Register worker
            self.db.execute("""
                INSERT INTO run_workers (
                    run_id, worker_id, hostname, process_id, 
                    version, capabilities
                ) VALUES (%s, %s, %s, %s, %s, %s)
                ON CONFLICT (run_id, worker_id) 
                DO UPDATE SET 
                    last_heartbeat = CURRENT_TIMESTAMP,
                    status = 'active'
            """, (
                run_id, worker_id, hostname, process_id,
                metadata.get('version') if metadata else None,
                json.dumps(metadata.get('capabilities', {})) if metadata else None
            ))
            
            # Update worker count
            self.db.execute("""
                UPDATE processing_runs
                SET worker_count = (
                    SELECT COUNT(DISTINCT worker_id)
                    FROM run_workers
                    WHERE run_workers.run_id = processing_runs.run_id
                      AND status = 'active'
                )
                WHERE run_id = %s
            """, (run_id,))


class WorkQueue:
    """Document work queue with atomic operations."""
    
    def __init__(self, db, worker_id: str):
        """
        Initialize work queue.
        
        Args:
            db: Database connection
            worker_id: Unique worker identifier
        """
        self.db = db
        self.worker_id = worker_id
        self.heartbeat_interval = 30  # seconds
        self.claim_timeout = 300  # 5 minutes
    
    def add_document(self, doc_id: str, source_name: str, run_id: str,
                    source_type: str = 'configured',
                    parent_doc_id: Optional[str] = None,
                    link_depth: int = 0,
                    metadata: Optional[Dict] = None) -> int:
        """
        Add a document to the processing queue.
        
        Args:
            doc_id: Document identifier
            source_name: Source name
            run_id: Processing run ID
            source_type: Type of source ('configured', 'linked', 'discovered')
            parent_doc_id: Parent document ID if this is a linked document
            link_depth: Depth in link chain
            metadata: Optional metadata
            
        Returns:
            Queue ID of the added document
        """
        with self.db.transaction():
            result = self.db.execute("""
                INSERT INTO document_queue (
                    doc_id, source_name, source_type, run_id,
                    parent_doc_id, link_depth, metadata
                ) VALUES (%s, %s, %s, %s, %s, %s, %s)
                ON CONFLICT (run_id, doc_id, source_name) 
                DO UPDATE SET
                    updated_at = CURRENT_TIMESTAMP,
                    link_depth = LEAST(document_queue.link_depth, EXCLUDED.link_depth)
                RETURNING queue_id
            """, (
                doc_id, source_name, source_type, run_id,
                parent_doc_id, link_depth,
                json.dumps(metadata) if metadata else None
            ))
            
            # Update run statistics
            self.db.execute("""
                UPDATE processing_runs
                SET documents_queued = documents_queued + 1,
                    last_activity_at = CURRENT_TIMESTAMP
                WHERE run_id = %s
            """, (run_id,))
            
            return result['queue_id']
    
    def claim_next_document(self, run_id: str) -> Optional[Dict[str, Any]]:
        """
        Atomically claim the next available document for processing.
        
        Uses PostgreSQL's FOR UPDATE SKIP LOCKED to ensure only one
        worker can claim each document.
        
        Args:
            run_id: Processing run ID
            
        Returns:
            Document information or None if no work available
        """
        with self.db.transaction():
            # First, try to claim a new document
            doc = self.db.execute("""
                SELECT queue_id, doc_id, source_name, source_type,
                       parent_doc_id, link_depth, metadata
                FROM document_queue
                WHERE run_id = %s
                  AND status = 'pending'
                  AND scheduled_for <= CURRENT_TIMESTAMP
                ORDER BY priority DESC, link_depth ASC, created_at ASC
                LIMIT 1
                FOR UPDATE SKIP LOCKED
            """, (run_id,))
            
            if not doc:
                # Check for stale claims (worker died)
                doc = self.db.execute("""
                    SELECT queue_id, doc_id, source_name, source_type,
                           parent_doc_id, link_depth, metadata
                    FROM document_queue
                    WHERE run_id = %s
                      AND status = 'processing'
                      AND claimed_at < CURRENT_TIMESTAMP - INTERVAL '%s seconds'
                    ORDER BY priority DESC, link_depth ASC, created_at ASC
                    LIMIT 1
                    FOR UPDATE SKIP LOCKED
                """, (run_id, self.claim_timeout))
            
            if doc:
                # Claim the document
                self.db.execute("""
                    UPDATE document_queue
                    SET status = 'processing',
                        worker_id = %s,
                        claimed_at = CURRENT_TIMESTAMP,
                        started_at = CURRENT_TIMESTAMP
                    WHERE queue_id = %s
                """, (self.worker_id, doc['queue_id']))
                
                # Update worker statistics
                self.db.execute("""
                    UPDATE run_workers
                    SET documents_claimed = documents_claimed + 1,
                        last_heartbeat = CURRENT_TIMESTAMP,
                        status = 'processing'
                    WHERE run_id = %s AND worker_id = %s
                """, (run_id, self.worker_id))
                
                logger.debug(f"Worker {self.worker_id} claimed document {doc['doc_id']}")
                return doc
        
        return None
    
    def mark_completed(self, queue_id: int, content_hash: Optional[str] = None,
                      file_size: Optional[int] = None) -> None:
        """
        Mark a document as successfully processed.
        
        Args:
            queue_id: Queue ID of the document
            content_hash: Optional content hash for change detection
            file_size: Optional file size
        """
        with self.db.transaction():
            # Get run_id for statistics update
            result = self.db.execute("""
                UPDATE document_queue
                SET status = 'completed',
                    completed_at = CURRENT_TIMESTAMP,
                    content_hash = COALESCE(%s, content_hash),
                    file_size = COALESCE(%s, file_size)
                WHERE queue_id = %s AND worker_id = %s
                RETURNING run_id
            """, (content_hash, file_size, queue_id, self.worker_id))
            
            if result:
                run_id = result['run_id']
                
                # Update run statistics
                self.db.execute("""
                    UPDATE processing_runs
                    SET documents_processed = documents_processed + 1,
                        last_activity_at = CURRENT_TIMESTAMP
                    WHERE run_id = %s
                """, (run_id,))
                
                # Update worker statistics
                self.db.execute("""
                    UPDATE run_workers
                    SET documents_processed = documents_processed + 1,
                        last_heartbeat = CURRENT_TIMESTAMP
                    WHERE run_id = %s AND worker_id = %s
                """, (run_id, self.worker_id))
                
                logger.debug(f"Document {queue_id} marked as completed")
    
    def mark_failed(self, queue_id: int, error_message: str,
                   error_details: Optional[Dict] = None) -> None:
        """
        Mark a document as failed and schedule retry if applicable.
        
        Args:
            queue_id: Queue ID of the document
            error_message: Error message
            error_details: Optional detailed error information
        """
        with self.db.transaction():
            # Get current retry count and run_id
            result = self.db.execute("""
                SELECT retry_count, max_retries, run_id
                FROM document_queue
                WHERE queue_id = %s
            """, (queue_id,))
            
            if not result:
                logger.warning(f"Document {queue_id} not found")
                return
            
            run_id = result['run_id']
            
            if result['retry_count'] < result['max_retries']:
                # Schedule retry with exponential backoff
                retry_delay = 2 ** result['retry_count'] * 60  # 1, 2, 4 minutes
                
                self.db.execute("""
                    UPDATE document_queue
                    SET status = 'retry',
                        worker_id = NULL,
                        retry_count = retry_count + 1,
                        scheduled_for = CURRENT_TIMESTAMP + INTERVAL '%s seconds',
                        error_message = %s,
                        error_details = %s,
                        failed_at = CURRENT_TIMESTAMP
                    WHERE queue_id = %s
                """, (retry_delay, error_message, 
                     json.dumps(error_details) if error_details else None,
                     queue_id))
                
                logger.info(f"Document {queue_id} scheduled for retry in {retry_delay} seconds")
                
                # Update retry statistics
                self.db.execute("""
                    UPDATE processing_runs
                    SET documents_retried = documents_retried + 1
                    WHERE run_id = %s
                """, (run_id,))
            else:
                # Max retries exceeded
                self.db.execute("""
                    UPDATE document_queue
                    SET status = 'failed',
                        error_message = %s,
                        error_details = %s,
                        failed_at = CURRENT_TIMESTAMP
                    WHERE queue_id = %s
                """, (error_message,
                     json.dumps(error_details) if error_details else None,
                     queue_id))
                
                logger.error(f"Document {queue_id} failed after {result['retry_count']} retries")
                
                # Update failure statistics
                self.db.execute("""
                    UPDATE processing_runs
                    SET documents_failed = documents_failed + 1
                    WHERE run_id = %s
                """, (run_id,))
            
            # Update worker statistics
            self.db.execute("""
                UPDATE run_workers
                SET documents_failed = documents_failed + 1,
                    last_heartbeat = CURRENT_TIMESTAMP
                WHERE run_id = %s AND worker_id = %s
            """, (run_id, self.worker_id))
    
    def add_linked_document(self, parent_doc_id: str, child_doc_id: str,
                          source_name: str, run_id: str,
                          link_depth: int) -> bool:
        """
        Add a discovered linked document to the queue.
        
        Args:
            parent_doc_id: Parent document ID
            child_doc_id: Child document ID
            source_name: Source name
            run_id: Processing run ID
            link_depth: Depth in link chain
            
        Returns:
            True if document was added, False if already exists
        """
        try:
            # Record dependency
            self.db.execute("""
                INSERT INTO document_dependencies (
                    parent_doc_id, child_doc_id, source_name, run_id,
                    link_type, link_depth, discovered_by_worker
                ) VALUES (%s, %s, %s, %s, 'discovered', %s, %s)
                ON CONFLICT DO NOTHING
            """, (parent_doc_id, child_doc_id, source_name, run_id,
                 link_depth, self.worker_id))
            
            # Add to queue
            queue_id = self.add_document(
                child_doc_id, source_name, run_id,
                source_type='linked',
                parent_doc_id=parent_doc_id,
                link_depth=link_depth
            )
            
            logger.info(f"Added linked document {child_doc_id} at depth {link_depth}")
            return True
            
        except Exception as e:
            logger.debug(f"Document {child_doc_id} already in queue or error: {e}")
            return False
    
    def get_queue_status(self, run_id: str) -> Dict[str, Any]:
        """
        Get current queue status for a run.
        
        Args:
            run_id: Processing run ID
            
        Returns:
            Queue status information
        """
        return self.db.execute("""
            SELECT 
                COUNT(*) FILTER (WHERE status = 'pending') as pending,
                COUNT(*) FILTER (WHERE status = 'processing') as processing,
                COUNT(*) FILTER (WHERE status = 'completed') as completed,
                COUNT(*) FILTER (WHERE status = 'failed') as failed,
                COUNT(*) FILTER (WHERE status = 'retry') as retry,
                COUNT(*) as total
            FROM document_queue
            WHERE run_id = %s
        """, (run_id,))
    
    def heartbeat(self, run_id: str) -> None:
        """
        Send worker heartbeat.
        
        Args:
            run_id: Processing run ID
        """
        self.db.execute("""
            UPDATE run_workers
            SET last_heartbeat = CURRENT_TIMESTAMP
            WHERE run_id = %s AND worker_id = %s
        """, (run_id, self.worker_id))