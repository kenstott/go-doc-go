"""
Processing coordinator for managing distributed document processing runs.
"""

import logging
import time
from typing import Dict, Any, List, Optional

from ..config import Config
from ..content_source.factory import get_content_source
from .work_queue import WorkQueue, RunCoordinator

logger = logging.getLogger(__name__)


class ProcessingCoordinator:
    """
    Coordinates document processing runs by discovering documents,
    populating the work queue, and managing post-processing.
    """
    
    def __init__(self, config: Config, coordinator_id: Optional[str] = None):
        """
        Initialize processing coordinator.
        
        Args:
            config: Configuration object  
            coordinator_id: Optional coordinator ID
        """
        self.config = config
        self.coordinator_id = coordinator_id or "coordinator"
        self.db = None
        self.work_queue = None
        self.run_coordinator = None
        
        logger.info(f"Initialized ProcessingCoordinator: {self.coordinator_id}")
    
    def coordinate_processing_run(self, source_configs: Optional[List[Dict]] = None,
                                 max_link_depth: Optional[int] = None) -> Dict[str, Any]:
        """
        Coordinate a complete distributed processing run.
        
        Args:
            source_configs: Optional list of content source configs (overrides config)
            max_link_depth: Optional override for max link depth
            
        Returns:
            Run statistics and information
        """
        logger.info(f"Starting coordinated processing run with coordinator {self.coordinator_id}")
        
        # Initialize components
        self._initialize_components()
        
        # Get run ID from configuration
        run_id = RunCoordinator.get_run_id_from_config(self.config.config)
        logger.info(f"Processing run ID: {run_id}")
        
        # Ensure processing run exists
        run_info = self.run_coordinator.ensure_run_exists(run_id, self.config.config)
        logger.info(f"Processing run initialized: {run_info}")
        
        # Discover and queue documents from all sources
        sources_to_process = source_configs or self.config.get_content_sources()
        queuing_stats = self._discover_and_queue_documents(
            sources_to_process, run_id, max_link_depth
        )
        
        logger.info(f"Document discovery completed: {queuing_stats}")
        
        # Wait for processing to complete
        completion_stats = self._wait_for_processing_completion(run_id)
        
        logger.info(f"Processing completion detected: {completion_stats}")
        
        # Perform post-processing
        post_processing_stats = self._perform_post_processing(run_id)
        
        logger.info(f"Post-processing completed: {post_processing_stats}")
        
        # Combine all statistics
        final_stats = {
            "run_id": run_id,
            "coordinator_id": self.coordinator_id,
            "documents_queued": queuing_stats["documents_queued"],
            "documents_processed": completion_stats["documents_processed"],
            "documents_failed": completion_stats["documents_failed"],
            "cross_document_relationships": post_processing_stats.get("relationships_created", 0),
            "total_runtime_seconds": completion_stats.get("total_runtime", 0)
        }
        
        logger.info(f"Coordinated processing run completed: {final_stats}")
        return final_stats
    
    def _initialize_components(self):
        """Initialize database and coordination components."""
        logger.debug("Initializing coordinator components")
        
        # Initialize database
        self.db = self.config.get_document_database()
        logger.debug(f"Database initialized: {type(self.db).__name__}")
        
        # Initialize work queue and run coordinator
        self.work_queue = WorkQueue(self.db, self.coordinator_id)
        self.run_coordinator = RunCoordinator(self.db)
        
        logger.debug("Coordinator components initialized")
    
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
                
                # Add each document to queue
                queued_count = 0
                for doc in documents:
                    try:
                        queue_id = self.work_queue.add_document(
                            doc_id=doc['id'],
                            source_name=source_name,
                            run_id=run_id,
                            source_type='configured',
                            metadata={
                                'max_link_depth': source_config.get('max_link_depth', 1),
                                'source_config': source_config
                            }
                        )
                        queued_count += 1
                        total_queued += 1
                        
                        logger.debug(f"Queued document {doc['id']} with queue_id {queue_id}")
                        
                    except Exception as e:
                        logger.error(f"Failed to queue document {doc['id']}: {str(e)}")
                
                source_stats.append({
                    "source_name": source_name,
                    "documents_found": len(documents),
                    "documents_queued": queued_count
                })
                
                logger.info(f"Completed source {source_name}: {queued_count}/{len(documents)} queued")
                
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
                    self.db, processed_doc_ids, self.config
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
            completed_docs = []
            for item in completed_queue_items:
                doc_info = {
                    'doc_id': item['doc_id'],
                    'source_name': item.get('source_name', 'unknown'),
                    'completed_at': item.get('completed_at')
                }
                completed_docs.append(doc_info)
            
            logger.debug(f"Retrieved {len(completed_docs)} processed documents for run {run_id}")
            return completed_docs
            
        except Exception as e:
            logger.error(f"Error retrieving processed documents for run {run_id}: {str(e)}")
            return []