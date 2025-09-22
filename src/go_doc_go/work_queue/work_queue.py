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
    """Manages processing runs with elected leader coordination."""
    
    def __init__(self, db, worker_id: str):
        """
        Initialize run coordinator.
        
        Args:
            db: Database connection (must support transactions)
            worker_id: Unique worker identifier
        """
        self.db = db
        self.worker_id = worker_id
        self.leader_lease_duration = 60  # seconds
        self.is_leader_cache = False
        self.last_leader_check = None
    
    @staticmethod
    def get_run_id_from_config(config: Dict[str, Any]) -> str:
        """
        Generate deterministic run ID from configuration.

        Args:
            config: Configuration dictionary

        Returns:
            16-character run ID (first 16 chars of SHA256 hash)
        """
        import os

        # Extract only the parts that affect processing
        processing_config = {
            'pipeline_name': config.get('name', ''),  # Add pipeline name for easy reprocessing
            'content_sources': config.get('content_sources', []),
            'storage': config.get('storage', {}),
            'embedding': config.get('embedding', {}),
            'relationship_detection': config.get('relationship_detection', {}),
            # Don't include worker-specific settings like ports, log levels
        }

        # Add storage existence state to force reprocessing if storage is deleted
        storage_state = {}

        # Check main analytics storage (if directly configured)
        if 'storage' in config and 'analytics' in config['storage']:
            analytics = config['storage']['analytics']

            # Analytics can be a string (registry reference) or dict (direct config)
            if isinstance(analytics, str):
                # Analytics is a registry backend name - skip, will be checked below
                logger.debug(f"Analytics storage references registry: {analytics}")
            elif isinstance(analytics, dict):
                storage_type = analytics.get('type')

                if storage_type == 'parquet':
                    # Check if parquet base path exists and has actual data
                    base_path = analytics.get('base_path', './data-lake')
                    elements_path = os.path.join(base_path, 'elements')

                    # Normalized check: data exists only if directory has files
                    has_data = False
                    if os.path.exists(elements_path):
                        try:
                            has_data = bool(os.listdir(elements_path))
                        except OSError:
                            has_data = False

                    # Single state representing "analytics data exists"
                    storage_state['has_analytics_data'] = has_data
                    logger.debug(f"Analytics storage check - type: parquet, base_path: {base_path}, has_data: {has_data}")

                elif storage_type in ['postgresql', 'mongodb', 'elasticsearch', 'neo4j']:
                    # For databases, we track the type - actual data check would require connections
                    # Assume data exists if backend is configured (databases persist)
                    storage_state['has_analytics_data'] = True
                    storage_state['db_type'] = storage_type
                    logger.debug(f"Analytics storage check - type: {storage_type}, assumed has_data: True")

        # Check configured search backend (which determines analytics backend)
        if 'search' in config and 'default_backend' in config['search']:
            configured_backend = config['search']['default_backend']

            # If backend is parquet_duckdb, check if its storage has data
            if configured_backend == 'parquet_duckdb':
                # parquet_duckdb uses data-lake directory per analytics_registry.yaml
                base_path = './data-lake'
                elements_path = os.path.join(base_path, 'elements')

                # Normalized check: data exists only if directory has files
                has_data = False
                if os.path.exists(elements_path):
                    try:
                        has_data = bool(os.listdir(elements_path))
                    except OSError:
                        has_data = False

                # Override any previous state - this is the actual analytics backend being used
                storage_state['has_analytics_data'] = has_data
                logger.debug(f"Search backend check - backend: {configured_backend}, base_path: {base_path}, has_data: {has_data}")

        # Also check analytics registry backends
        if 'analytics_registry' in config:
            # Check each enabled analytics backend
            for backend_name, backend_config in config.get('analytics_registry', {}).items():
                if backend_config.get('enabled', False):
                    backend_type = backend_config.get('type')

                    if backend_type == 'parquet':
                        base_path = backend_config.get('base_path', './data-lake')
                        elements_path = os.path.join(base_path, 'elements')

                        # Normalized check: same logic as main storage
                        has_data = False
                        if os.path.exists(elements_path):
                            try:
                                has_data = bool(os.listdir(elements_path))
                            except OSError:
                                has_data = False

                        # If any enabled backend has data, set the flag
                        if has_data:
                            storage_state['has_analytics_data'] = True

                        logger.debug(f"Analytics registry '{backend_name}' check - type: parquet, base_path: {base_path}, has_data: {has_data}")

                    elif backend_type in ['postgresql', 'mongodb', 'elasticsearch', 'neo4j']:
                        # For database backends, assume data exists if configured
                        storage_state['has_analytics_data'] = True
                        logger.debug(f"Analytics registry '{backend_name}' check - type: {backend_type}, assumed has_data: True")

        # Note: We intentionally do NOT include storage_state in the hash
        # This ensures deterministic run_id generation based only on configuration
        # that affects how documents are processed, not on whether data exists
        # Storage state is still tracked separately for reprocessing decisions

        # Sort keys for deterministic hashing (excluding storage_state)
        config_str = json.dumps(processing_config, sort_keys=True)

        # Create hash - use first 16 chars for readability
        full_hash = hashlib.sha256(config_str.encode()).hexdigest()
        run_id = full_hash[:16]

        # Log run_id generation details
        logger.info(f"Generated run_id: {run_id}")
        logger.debug(f"Run ID generation - storage_state: {storage_state}")
        logger.debug(f"Run ID generation - config hash input: {config_str[:200]}...")

        return run_id
    
    def _check_analytics_storage_has_run(self, run_id: str, config: Dict[str, Any]) -> bool:
        """
        Check if the analytics storage has data for the given run_id.

        Args:
            run_id: Run ID to check
            config: Configuration to determine storage type

        Returns:
            True if analytics storage has data for this run, False otherwise
        """
        try:
            from ..storage_adapters.factory import StorageFactory

            # Get analytics storage configuration
            storage_config = config.get('storage', {})
            analytics_config = storage_config.get('analytics', {})

            # Create analytics storage instance
            analytics_storage = StorageFactory.create_analytics_storage(analytics_config)

            # Use the storage's has_run method to check if data exists
            if hasattr(analytics_storage, 'has_run'):
                return analytics_storage.has_run(run_id)
            else:
                # If storage doesn't implement has_run, assume data exists
                logger.debug(f"Analytics storage doesn't implement has_run method - assuming data exists")
                return True

        except Exception as e:
            logger.warning(f"Error checking analytics storage for run {run_id}: {e}")
            # On error, assume data exists to maintain backward compatibility
            return True

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
            # Try to get existing run by config_hash first
            existing = self.db.execute("""
                SELECT run_id, status, created_at, worker_count
                FROM processing_runs
                WHERE config_hash = %s
            """, (full_hash,))

            if existing:
                # Check if analytics storage actually has data for this run
                run_has_data = self._check_analytics_storage_has_run(existing['run_id'], config)

                if run_has_data:
                    # Update last activity
                    self.db.execute("""
                        UPDATE processing_runs
                        SET last_activity_at = CURRENT_TIMESTAMP
                        WHERE config_hash = %s
                    """, (full_hash,))
                    return existing
                else:
                    # Analytics storage missing - delete old run and all related data
                    logger.warning(f"Run {existing['run_id']} exists in DB but analytics storage is missing. Cleaning up and creating new run.")

                    # Delete in correct order to respect foreign key constraints
                    # First delete document dependencies
                    deleted_deps = self.db.execute("""
                        DELETE FROM document_dependencies WHERE run_id = %s
                    """, (existing['run_id'],))
                    if deleted_deps:
                        logger.debug(f"Deleted {deleted_deps} document dependencies for run {existing['run_id']}")

                    # Then delete queued documents
                    deleted_docs = self.db.execute("""
                        DELETE FROM document_queue WHERE run_id = %s
                    """, (existing['run_id'],))
                    if deleted_docs:
                        logger.debug(f"Deleted {deleted_docs} queued documents for run {existing['run_id']}")

                    # Delete worker registrations
                    deleted_workers = self.db.execute("""
                        DELETE FROM run_workers WHERE run_id = %s
                    """, (existing['run_id'],))
                    if deleted_workers:
                        logger.debug(f"Deleted {deleted_workers} worker registrations for run {existing['run_id']}")

                    # Finally delete the run itself
                    self.db.execute("""
                        DELETE FROM processing_runs WHERE config_hash = %s
                    """, (full_hash,))
                    logger.info(f"Cleaned up stale run {existing['run_id']} and all related data")

            # Create new run - ON CONFLICT handles both run_id and config_hash conflicts
            self.db.execute("""
                INSERT INTO processing_runs (
                    run_id, config_hash, config_snapshot, status
                ) VALUES (%s, %s, %s, 'active')
                ON CONFLICT (config_hash) DO UPDATE SET
                    last_activity_at = CURRENT_TIMESTAMP
            """, (run_id, full_hash, json.dumps(config)))

            # Query back the run (in case of conflict, we need the actual run_id)
            result = self.db.execute("""
                SELECT run_id, status, created_at, worker_count
                FROM processing_runs
                WHERE config_hash = %s
            """, (full_hash,))

            if result:
                logger.info(f"Processing run ensured: {result['run_id']}")
                return result
            else:
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
    
    def attempt_leader_election(self, run_id: str) -> bool:
        """
        Attempt to become the leader for a processing run.
        
        Args:
            run_id: Processing run ID
            
        Returns:
            True if this worker became or remains the leader
        """
        try:
            result = self.db.execute("""
                SELECT attempt_leader_election(%s, %s, %s) as elected
            """, (run_id, self.worker_id, self.leader_lease_duration))
            
            elected = result.get('elected', False) if result else False
            self.is_leader_cache = elected
            self.last_leader_check = time.time()
            
            if elected:
                logger.info(f"Worker {self.worker_id} elected as leader for run {run_id}")
            
            return elected
            
        except Exception as e:
            logger.error(f"Leader election failed for worker {self.worker_id}: {e}")
            self.is_leader_cache = False
            return False
    
    def is_leader(self, run_id: str) -> bool:
        """
        Check if this worker is currently the leader.
        
        Args:
            run_id: Processing run ID
            
        Returns:
            True if this worker is the current leader
        """
        # Check cache first (avoid frequent DB queries)
        if (self.last_leader_check and 
            time.time() - self.last_leader_check < 10):  # 10 second cache
            return self.is_leader_cache
        
        try:
            result = self.db.execute("""
                SELECT leader_worker_id, leader_lease_expires
                FROM processing_runs
                WHERE run_id = %s
            """, (run_id,))
            
            if result:
                leader_id = result.get('leader_worker_id')
                lease_expires = result.get('leader_lease_expires')
                
                # Check if we're the leader and lease hasn't expired
                is_leader = (leader_id == self.worker_id and 
                           lease_expires and lease_expires > datetime.now())
                
                self.is_leader_cache = is_leader
                self.last_leader_check = time.time()
                return is_leader
            
            self.is_leader_cache = False
            self.last_leader_check = time.time()
            return False
            
        except Exception as e:
            logger.error(f"Leader check failed for worker {self.worker_id}: {e}")
            return False
    
    def renew_leadership(self, run_id: str) -> bool:
        """
        Renew leadership lease if this worker is the leader.
        
        Args:
            run_id: Processing run ID
            
        Returns:
            True if lease was renewed successfully
        """
        if not self.is_leader_cache:
            return False
            
        try:
            # Attempt to renew (this also handles case where we lost leadership)
            return self.attempt_leader_election(run_id)
            
        except Exception as e:
            logger.error(f"Leadership renewal failed for worker {self.worker_id}: {e}")
            self.is_leader_cache = False
            return False


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
        self.max_retries = 3  # Default max retries for failed documents
    
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
        logger.debug(f"Worker {self.worker_id} attempting to claim document for run {run_id}")

        with self.db.transaction():
            # First, try to claim a new document
            logger.debug(f"Looking for pending documents in run {run_id}")
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
                logger.debug(f"No pending documents found for run {run_id}")
                # Check for retry documents that are ready to be processed
                doc = self.db.execute("""
                    SELECT queue_id, doc_id, source_name, source_type,
                           parent_doc_id, link_depth, metadata
                    FROM document_queue
                    WHERE run_id = %s
                      AND status = 'retry'
                      AND scheduled_for <= CURRENT_TIMESTAMP
                    ORDER BY priority DESC, link_depth ASC, created_at ASC
                    LIMIT 1
                    FOR UPDATE SKIP LOCKED
                """, (run_id,))

            if not doc:
                logger.debug(f"No retry documents ready for run {run_id}")
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
    
    def delete_documents_for_run(self, run_id: str) -> int:
        """
        Delete all documents for a run to allow re-enqueueing.

        Args:
            run_id: Processing run ID

        Returns:
            Number of documents deleted
        """
        with self.db.transaction():
            # Get count of documents to delete (for logging)
            count_result = self.db.execute("""
                SELECT COUNT(*) as count
                FROM document_queue
                WHERE run_id = %s
            """, (run_id,))

            count = count_result.get('count', 0) if count_result else 0

            # Delete all documents for this run
            self.db.execute("""
                DELETE FROM document_queue
                WHERE run_id = %s
            """, (run_id,))

            logger.info(f"Deleted {count} documents for run {run_id} to allow fresh discovery")
            return count

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