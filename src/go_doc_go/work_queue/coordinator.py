"""
Elected leader worker for managing distributed document processing runs.
"""

import logging
import time
from typing import Dict, Any, List, Optional

from ..config import Config
from ..content_source.factory import get_content_source
from .work_queue import WorkQueue, RunCoordinator

logger = logging.getLogger(__name__)


class ElectedLeaderWorker:
    """
    Worker that can be elected as leader to handle document discovery,
    queue population, and post-processing while still processing documents.
    """
    
    def __init__(self, config: Config, worker_id: str):
        """
        Initialize elected leader worker.
        
        Args:
            config: Configuration object  
            worker_id: Unique worker ID
        """
        self.config = config
        self.worker_id = worker_id
        self.db = None
        self.work_queue = None
        self.run_coordinator = None
        self.is_leader = False
        
        logger.info(f"Initialized ElectedLeaderWorker: {self.worker_id}")
    
    def run_as_worker_with_leader_duties(self, source_configs: Optional[List[Dict]] = None,
                                        max_link_depth: Optional[int] = None) -> Dict[str, Any]:
        """
        Run as a worker that attempts to become leader for discovery and post-processing.
        
        Args:
            source_configs: Optional list of content source configs (overrides config)
            max_link_depth: Optional override for max link depth
            
        Returns:
            Run statistics and information
        """
        logger.info(f"Starting worker {self.worker_id} with leader election")
        
        # Initialize components
        self._initialize_components()
        
        # Get run ID from configuration
        generated_run_id = RunCoordinator.get_run_id_from_config(self.config.config)
        logger.info(f"Generated run ID: {generated_run_id}")

        # Ensure processing run exists
        run_info = self.run_coordinator.ensure_run_exists(generated_run_id, self.config.config)
        logger.info(f"Processing run initialized: {run_info}")

        # Use the actual run_id from run_info (in case of existing run with different ID)
        run_id = run_info['run_id']
        logger.info(f"Using actual run ID: {run_id}")

        # Register as worker
        self.run_coordinator.register_worker(run_id, self.worker_id, {
            'version': '1.0.0',
            'capabilities': {'leader_eligible': True}
        })

        # Attempt to become leader
        if self.run_coordinator.attempt_leader_election(run_id):
            logger.info(f"Worker {self.worker_id} elected as leader")
            self.is_leader = True
            
            # As leader, discover and queue documents from all sources
            sources_to_process = source_configs or self.config.get_content_sources()
            queuing_stats = self._discover_and_queue_documents(
                sources_to_process, run_id, max_link_depth
            )
            logger.info(f"Document discovery completed: {queuing_stats}")
        else:
            logger.info(f"Worker {self.worker_id} not elected as leader - will process documents from queue")
            queuing_stats = {"documents_queued": 0}
        
        # All workers (including leader) process documents from queue
        # Create and run document processor to consume the queued work
        processing_stats = self._process_queued_documents(run_id)
        logger.info(f"Document processing completed: {processing_stats}")
        
        # If we are the leader, we also handle "last man standing" duties
        if self.is_leader:
            # Wait for processing to complete
            completion_stats = self._wait_for_processing_completion(run_id)
            logger.info(f"Processing completion detected: {completion_stats}")
            
            # Perform post-processing
            post_processing_stats = self._perform_post_processing(run_id)
            logger.info(f"Post-processing completed: {post_processing_stats}")
            
            # Combine all statistics
            final_stats = {
                "run_id": run_id,
                "worker_id": self.worker_id,
                "is_leader": True,
                "documents_queued": queuing_stats["documents_queued"],
                "documents_processed": completion_stats["documents_processed"],
                "documents_failed": completion_stats["documents_failed"],
                "cross_document_relationships": post_processing_stats.get("relationships_created", 0),
                "total_runtime_seconds": completion_stats.get("total_runtime", 0)
            }
        else:
            # Non-leader workers just return basic stats
            final_stats = {
                "run_id": run_id,
                "worker_id": self.worker_id,
                "is_leader": False,
                "message": "Worker participating in distributed processing"
            }
        
        logger.info(f"Worker {self.worker_id} completed run participation: {final_stats}")
        return final_stats
    
    def _initialize_components(self):
        """Initialize database and coordination components."""
        logger.debug("Initializing worker components")

        # Initialize dual storage architecture - MANDATORY
        from ..storage_adapters.factory import StorageFactory
        job_storage, analytics_storage = StorageFactory.create_from_pipeline_config(
            self.config.config,
            registry=self.config.list_analytics_backends()
        )

        # Use job storage for work queue coordination
        self.db = job_storage
        self.analytics_storage = analytics_storage
        logger.debug(f"Dual storage initialized - Job: {type(job_storage).__name__}, Analytics: {type(analytics_storage).__name__}")

        # Initialize work queue and run coordinator with worker ID
        self.work_queue = WorkQueue(self.db, self.worker_id)
        self.run_coordinator = RunCoordinator(self.db, self.worker_id)

        logger.debug("Coordinator components initialized")
    
    def _get_analytics_embedding_generator(self):
        """
        Get the embedding generator configured for the analytics backend.

        Returns:
            Configured embedding generator for analytics backend
        """
        logger.info("[ENTRY] _get_analytics_embedding_generator")

        # Get analytics configuration from storage config
        analytics_config = self.config.config.get('storage', {}).get('analytics')
        logger.info(f"[_get_analytics_embedding_generator] analytics_config: {analytics_config}")

        if not analytics_config:
            raise ValueError("No analytics configuration found in storage config - required for embedding initialization")

        # Extract analytics backend name and get its embedding config
        analytics_backend_name = None
        embedding_config = None

        if isinstance(analytics_config, str):
            # Analytics specified as registry name
            analytics_backend_name = analytics_config
            logger.info(f"[_get_analytics_embedding_generator] Analytics is registry name: {analytics_backend_name}")
            # Get backend config from analytics registry
            backend_config = self.config.get_analytics_backend(analytics_backend_name)
            logger.info(f"[_get_analytics_embedding_generator] Backend config from registry: {backend_config}")
            if backend_config:
                embedding_config = backend_config.get('embedding', {})
                logger.info(f"[_get_analytics_embedding_generator] Embedding config from backend: {embedding_config}")
        elif isinstance(analytics_config, dict):
            # Analytics specified as direct config
            backend_type = analytics_config.get('type', '')
            # Map common backend types to registry names
            type_to_registry = {
                'parquet': 'parquet_duckdb',
                'elasticsearch': 'elasticsearch',
                'mongodb': 'mongodb'
            }
            analytics_backend_name = type_to_registry.get(backend_type)
            if analytics_backend_name:
                backend_config = self.config.get_analytics_backend(analytics_backend_name)
                if backend_config:
                    embedding_config = backend_config.get('embedding', {})
                else:
                    embedding_config = analytics_config.get('embedding', {})

        if not embedding_config:
            raise ValueError(f"No embedding configuration found for analytics backend: {analytics_backend_name}")

        # Create embedder using analytics registry configuration
        from ..embeddings.factory import get_embedder_from_analytics_registry

        try:
            logger.info(f"[_get_analytics_embedding_generator] Calling get_embedder_from_analytics_registry with backend_name={analytics_backend_name}, embedding_config={embedding_config}")
            result = get_embedder_from_analytics_registry(analytics_backend_name, embedding_config, self.config)
            logger.info(f"[EXIT] _get_analytics_embedding_generator: Successfully created embedder")
            return result
        except Exception as e:
            logger.error(f"[ERROR] _get_analytics_embedding_generator: Failed to create embedder from analytics registry: {e}")
            raise

    def _document_already_processed(self, compound_doc_id: str, run_id: str) -> bool:
        """
        Check if a document (with compound ID) has already been processed.
        
        Args:
            compound_doc_id: Document ID in format "filename::timestamp"  
            run_id: Processing run ID
            
        Returns:
            True if document has already been processed, False otherwise
        """
        try:
            # Use the already initialized analytics storage
            # Check if any elements exist for this document
            elements = self.analytics_storage.get_document_elements(compound_doc_id)
            
            # If elements exist, document has been processed
            is_processed = len(elements) > 0
            if is_processed:
                logger.debug(f"Document {compound_doc_id} already processed ({len(elements)} elements found)")
            
            return is_processed
            
        except Exception as e:
            logger.warning(f"Error checking if document {compound_doc_id} already processed: {e}")
            # If we can't check, assume it needs processing (safer)
            return False
    
    def _discover_and_queue_documents(self, source_configs: List[Dict], run_id: str,
                                     max_link_depth: Optional[int] = None) -> Dict[str, Any]:
        """
        Discover documents from all sources and add them to the work queue.
        
        Args:
            source_configs: List of content source configurations
            run_id: Processing run ID
            max_link_depth: Optional override for max link depth
            
        Returns:
            Queuing statistics
        """
        logger.info(f"Discovering documents from {len(source_configs)} sources")
        
        total_queued = 0
        source_stats = []
        
        for source_config in source_configs:
            source_name = source_config.get('name')
            source_type = source_config.get('type')
            
            logger.info(f"Processing source: {source_name} ({source_type})")
            
            # Override max_link_depth if specified
            if max_link_depth is not None:
                original_depth = source_config.get('max_link_depth', 1)
                source_config['max_link_depth'] = max_link_depth
                logger.debug(f"Overriding max_link_depth from {original_depth} to {max_link_depth}")
            
            try:
                # Create content source
                source = get_content_source(source_config)
                
                # List all documents
                documents = source.list_documents()
                logger.info(f"Found {len(documents)} documents in source {source_name}")
                
                # Add each document to queue (only if not already processed)
                queued_count = 0
                skipped_count = 0
                
                for doc in documents:
                    try:
                        # Generate compound document key: filename::timestamp
                        doc_metadata = doc.get('metadata', {})
                        last_modified = doc_metadata.get('last_modified')
                        
                        if last_modified is not None:
                            # Create compound key with timestamp
                            compound_doc_id = f"{doc['id']}::{int(last_modified)}"
                        else:
                            # Fallback to original doc_id if no timestamp available
                            compound_doc_id = doc['id']
                            logger.warning(f"No last_modified timestamp for {doc['id']}, using basic doc_id")
                        
                        # Check if this exact document version already exists in analytics storage
                        if self._document_already_processed(compound_doc_id, run_id):
                            logger.debug(f"Skipping already processed document: {compound_doc_id}")
                            skipped_count += 1
                            continue
                        
                        # Queue document with compound ID
                        queue_id = self.work_queue.add_document(
                            doc_id=compound_doc_id,
                            source_name=source_name,
                            run_id=run_id,
                            source_type='configured',
                            metadata={
                                'max_link_depth': source_config.get('max_link_depth', 1),
                                'source_config': source_config,
                                'original_doc_id': doc['id'],  # Preserve original for reference
                                'last_modified': last_modified
                            }
                        )
                        queued_count += 1
                        total_queued += 1
                        
                        logger.debug(f"Queued new/modified document {compound_doc_id} with queue_id {queue_id}")
                        
                    except Exception as e:
                        logger.error(f"Failed to queue document {doc['id']}: {str(e)}")
                
                source_stats.append({
                    "source_name": source_name,
                    "documents_found": len(documents),
                    "documents_queued": queued_count,
                    "documents_skipped": skipped_count
                })
                
                logger.info(f"Completed source {source_name}: {queued_count}/{len(documents)} queued, {skipped_count} skipped (already processed)")
                
            except Exception as e:
                logger.error(f"Error processing source {source_name}: {str(e)}")
                source_stats.append({
                    "source_name": source_name,
                    "documents_found": 0,
                    "documents_queued": 0,
                    "error": str(e)
                })
        
        queuing_stats = {
            "documents_queued": total_queued,
            "sources_processed": len(source_configs),
            "source_details": source_stats
        }
        
        logger.info(f"Document discovery completed: {total_queued} documents queued")
        return queuing_stats
    
    def _wait_for_processing_completion(self, run_id: str, 
                                      check_interval: int = 30,
                                      max_wait_time: int = 3600) -> Dict[str, Any]:
        """
        Wait for all workers to complete processing all documents.
        
        Args:
            run_id: Processing run ID
            check_interval: How often to check status (seconds)
            max_wait_time: Maximum time to wait (seconds)
            
        Returns:
            Completion statistics
        """
        logger.info(f"Waiting for processing completion of run {run_id}")
        
        start_time = time.time()
        last_status_time = 0
        
        while True:
            current_time = time.time()
            elapsed = current_time - start_time
            
            # Check for timeout
            if elapsed > max_wait_time:
                logger.error(f"Processing timeout after {elapsed} seconds")
                break
            
            # Get queue status
            queue_status = self.work_queue.get_queue_status(run_id)
            
            if not queue_status:
                logger.warning("No queue status available - assuming completion")
                break
            
            pending = queue_status.get('pending', 0)
            processing = queue_status.get('processing', 0)
            completed = queue_status.get('completed', 0)
            failed = queue_status.get('failed', 0)
            retry = queue_status.get('retry', 0)
            total = queue_status.get('total', 0)
            
            # Log status periodically
            if current_time - last_status_time >= 60:  # Every minute
                logger.info(
                    f"Queue status: {pending} pending, {processing} processing, "
                    f"{completed} completed, {failed} failed, {retry} retry "
                    f"(total: {total})"
                )
                last_status_time = current_time
            
            # Check if processing is complete
            active_work = pending + processing + retry
            if active_work == 0:
                logger.info("All documents processed - completion detected")
                break
            
            # Wait before next check
            time.sleep(check_interval)
        
        # Get final statistics
        final_status = self.work_queue.get_queue_status(run_id)
        completion_stats = {
            "documents_processed": final_status.get('completed', 0),
            "documents_failed": final_status.get('failed', 0),
            "total_runtime": time.time() - start_time
        }
        
        logger.info(f"Processing completion statistics: {completion_stats}")
        return completion_stats
    
    def _perform_post_processing(self, run_id: str) -> Dict[str, Any]:
        """
        Perform post-processing tasks after all documents are processed.
        
        Args:
            run_id: Processing run ID
            
        Returns:
            Post-processing statistics
        """
        logger.info(f"Starting post-processing for run {run_id}")
        
        post_processing_stats = {
            "relationships_created": 0
        }
        
        try:
            # Get all successfully processed documents for this run
            processed_docs = self._get_processed_documents(run_id)
            
            if not processed_docs:
                logger.warning("No processed documents found for cross-document relationships")
                return post_processing_stats
            
            logger.info(f"Found {len(processed_docs)} processed documents for post-processing")
            
            # Generate cross-document relationships if embedding is enabled
            if self.config.is_embedding_enabled():
                from ..main import _compute_cross_document_container_relationships
                
                processed_doc_ids = [doc['doc_id'] for doc in processed_docs]
                relationship_count = _compute_cross_document_container_relationships(
                    self.analytics_storage, processed_doc_ids, self.config
                )
                
                post_processing_stats["relationships_created"] = relationship_count
                logger.info(f"Created {relationship_count} cross-document relationships")
            else:
                logger.info("Embeddings not enabled - skipping cross-document relationships")
        
        except Exception as e:
            logger.error(f"Error during post-processing: {str(e)}")
            # Don't fail the entire run due to post-processing errors
        
        logger.info(f"Post-processing completed: {post_processing_stats}")
        return post_processing_stats
    
    def _get_processed_documents(self, run_id: str) -> List[Dict[str, Any]]:
        """
        Get list of successfully processed documents for the run.
        
        Args:
            run_id: Processing run ID
            
        Returns:
            List of processed document information with doc_id for each completed document
        """
        try:
            # Query the work queue for completed documents in this run
            queue_status_query = """
                SELECT DISTINCT doc_id, source_name, completed_at
                FROM document_queue 
                WHERE run_id = %s 
                AND status = 'completed'
                ORDER BY completed_at
            """
            
            completed_queue_items = self.db.execute(queue_status_query, (run_id,))
            
            # Convert queue items to document info format expected by post-processing
            # Query returns tuples with columns: doc_id, source_name, completed_at
            completed_docs = []
            for item in completed_queue_items:
                doc_info = {
                    'doc_id': item[0],  # doc_id
                    'source_name': item[1] if item[1] else 'unknown',  # source_name
                    'completed_at': item[2]  # completed_at
                }
                completed_docs.append(doc_info)
            
            logger.debug(f"Retrieved {len(completed_docs)} processed documents for run {run_id}")
            return completed_docs
            
        except Exception as e:
            logger.error(f"Error retrieving processed documents for run {run_id}: {str(e)}")
            return []
    
    def _process_queued_documents(self, run_id: str) -> Dict[str, Any]:
        """
        Process documents from the queue using the QueuedDocumentProcessor.
        
        Args:
            run_id: Processing run ID
            
        Returns:
            Processing statistics
        """
        logger.info(f"Worker {self.worker_id} starting queued document processing for run {run_id}")
        
        # Initialize embedding generator (if enabled) - use analytics backend config
        embedding_generator = None
        if self.config.is_embedding_enabled():
            try:
                logger.info("Starting embedding generator initialization...")
                embedding_generator = self._get_analytics_embedding_generator()
                logger.info(f"Embedding generator initialized for analytics backend")
            except Exception as e:
                logger.error(f"Failed to initialize embedding generator: {e}")
                logger.warning("Continuing without embeddings")
                embedding_generator = None
        
        # Initialize relationship detector
        logger.info("[_process_queued_documents] Starting relationship detector initialization...")
        ontology_manager = None
        if self.config.is_domain_detection_enabled():
            ontology_manager = self.config.get_ontology_manager()

        from ..relationships import create_relationship_detector
        relationship_detector = create_relationship_detector(
            self.config.get_relationship_detection_config(),
            embedding_generator,
            db=self.db,
            ontology_manager=ontology_manager
        )
        logger.info("[_process_queued_documents] Relationship detector initialized")

        # Initialize document processor with both storages
        logger.info("[_process_queued_documents] Starting document processor initialization...")
        from .document_processor import QueuedDocumentProcessor
        processor = QueuedDocumentProcessor(
            job_storage=self.db,
            analytics_storage=self.analytics_storage,
            work_queue=self.work_queue,
            relationship_detector=relationship_detector,
            embedding_generator=embedding_generator
        )
        logger.info("[_process_queued_documents] Document processor initialized")

        # Process documents from the queue
        logger.info(f"[_process_queued_documents] Starting to process documents for run {run_id}")
        processing_stats = processor.process_documents(run_id)
        logger.info(f"[_process_queued_documents] Finished processing documents: {processing_stats}")
        
        logger.info(
            f"Worker {self.worker_id} completed queued processing: "
            f"{processing_stats['documents_processed']} processed, {processing_stats['documents_failed']} failed"
        )
        
        return processing_stats