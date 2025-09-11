"""
Document worker implementation for distributed processing.
"""

import logging
import signal
import threading
import time
import uuid
from typing import Dict, Any, Optional

from ..config import Config
from ..embeddings import get_embedding_generator
from ..relationships import create_relationship_detector
from .document_processor import QueuedDocumentProcessor
from .work_queue import WorkQueue, RunCoordinator

logger = logging.getLogger(__name__)


class DocumentWorker:
    """
    Standalone document worker that processes documents from a work queue.
    Handles worker lifecycle, heartbeats, and graceful shutdown.
    """
    
    def __init__(self, config: Config, worker_id: Optional[str] = None):
        """
        Initialize document worker.
        
        Args:
            config: Configuration object
            worker_id: Optional worker ID (generates UUID if not provided)
        """
        self.config = config
        self.worker_id = worker_id or f"worker_{uuid.uuid4().hex[:8]}"
        self.running = False
        self.shutdown_requested = False
        
        # Initialize components
        self.db = None
        self.work_queue = None
        self.processor = None
        self.heartbeat_thread = None
        
        # Statistics
        self.stats = {
            "documents_processed": 0,
            "documents_failed": 0,
            "elements_created": 0,
            "relationships_created": 0,
            "links_discovered": 0,
            "start_time": None,
            "end_time": None
        }
        
        logger.info(f"Initialized DocumentWorker: {self.worker_id}")
    
    def start(self) -> Dict[str, Any]:
        """
        Start the worker and begin processing documents.
        
        Returns:
            Processing statistics
        """
        logger.info(f"Starting worker {self.worker_id}")
        
        # Set up signal handlers for graceful shutdown
        signal.signal(signal.SIGINT, self._signal_handler)
        signal.signal(signal.SIGTERM, self._signal_handler)
        
        self.stats["start_time"] = time.time()
        self.running = True
        
        try:
            # Initialize database and components
            self._initialize_components()
            
            # Get processing run ID from configuration
            run_id = RunCoordinator.get_run_id_from_config(self.config.config)
            
            # Register worker with coordinator
            coordinator = RunCoordinator(self.db)
            coordinator.register_worker(run_id, self.worker_id, {
                "version": "1.0.0",  # Could get from package metadata
                "capabilities": ["document_parsing", "link_discovery", "embedding_generation"]
            })
            
            # Start heartbeat thread
            self._start_heartbeat_thread(run_id)
            
            # Process documents until shutdown or no more work
            logger.info(f"Worker {self.worker_id} beginning document processing for run {run_id}")
            
            processing_stats = self.processor.process_documents(run_id)
            
            # Update overall statistics
            self.stats.update(processing_stats)
            
            logger.info(f"Worker {self.worker_id} completed processing")
            
        except Exception as e:
            logger.error(f"Worker {self.worker_id} encountered error: {str(e)}")
            raise
        finally:
            self._cleanup()
        
        self.stats["end_time"] = time.time()
        return self.stats
    
    def stop(self):
        """Request graceful shutdown of the worker."""
        logger.info(f"Shutdown requested for worker {self.worker_id}")
        self.shutdown_requested = True
        self.running = False
    
    def _initialize_components(self):
        """Initialize database, work queue, and processor components."""
        logger.debug(f"Initializing components for worker {self.worker_id}")
        
        # Initialize database
        self.db = self.config.get_document_database()
        logger.debug(f"Database initialized: {type(self.db).__name__}")
        
        # Initialize work queue
        self.work_queue = WorkQueue(self.db, self.worker_id)
        logger.debug(f"Work queue initialized for worker {self.worker_id}")
        
        # Initialize embedding generator (if enabled)
        embedding_generator = None
        if self.config.is_embedding_enabled():
            embedding_generator = get_embedding_generator(self.config)
            logger.info(f"Embedding generator initialized: {self.config.get_embedding_model()}")
        
        # Initialize relationship detector
        ontology_manager = None
        if self.config.is_domain_detection_enabled():
            ontology_manager = self.config.get_ontology_manager()
        
        relationship_detector = create_relationship_detector(
            self.config.get_relationship_detection_config(),
            embedding_generator,
            db=self.db,
            ontology_manager=ontology_manager
        )
        logger.debug("Relationship detector initialized")
        
        # Initialize document processor
        self.processor = QueuedDocumentProcessor(
            db=self.db,
            work_queue=self.work_queue,
            relationship_detector=relationship_detector,
            embedding_generator=embedding_generator
        )
        logger.debug("Document processor initialized")
    
    def _start_heartbeat_thread(self, run_id: str):
        """Start background thread for sending heartbeats."""
        heartbeat_interval = self.work_queue.heartbeat_interval
        
        def heartbeat_loop():
            while self.running and not self.shutdown_requested:
                try:
                    self.work_queue.heartbeat(run_id)
                    logger.debug(f"Worker {self.worker_id} sent heartbeat")
                except Exception as e:
                    logger.error(f"Heartbeat error for worker {self.worker_id}: {str(e)}")
                
                # Wait for next heartbeat interval
                for _ in range(heartbeat_interval):
                    if not self.running or self.shutdown_requested:
                        break
                    time.sleep(1)
        
        self.heartbeat_thread = threading.Thread(target=heartbeat_loop, daemon=True)
        self.heartbeat_thread.start()
        logger.debug(f"Heartbeat thread started for worker {self.worker_id}")
    
    def _signal_handler(self, signum, frame):
        """Handle shutdown signals gracefully."""
        signal_name = signal.Signals(signum).name
        logger.info(f"Worker {self.worker_id} received {signal_name} signal")
        self.stop()
    
    def _cleanup(self):
        """Clean up resources and connections."""
        logger.debug(f"Cleaning up worker {self.worker_id}")
        
        self.running = False
        
        # Wait for heartbeat thread to finish
        if self.heartbeat_thread and self.heartbeat_thread.is_alive():
            self.heartbeat_thread.join(timeout=5)
        
        # Close database connection
        if self.db:
            try:
                self.db.close()
            except Exception as e:
                logger.warning(f"Error closing database connection: {str(e)}")
        
        logger.debug(f"Cleanup completed for worker {self.worker_id}")


class WorkerManager:
    """
    Manager for running multiple document workers in a single process.
    """
    
    def __init__(self, config: Config, num_workers: int = 1):
        """
        Initialize worker manager.
        
        Args:
            config: Configuration object
            num_workers: Number of worker threads to run
        """
        self.config = config
        self.num_workers = num_workers
        self.workers = []
        self.worker_threads = []
        
        logger.info(f"Initialized WorkerManager with {num_workers} workers")
    
    def start_all(self) -> Dict[str, Any]:
        """
        Start all workers and wait for completion.
        
        Returns:
            Combined processing statistics
        """
        logger.info(f"Starting {self.num_workers} workers")
        
        # Create and start worker threads
        for i in range(self.num_workers):
            worker = DocumentWorker(self.config, f"worker_{i+1}")
            self.workers.append(worker)
            
            thread = threading.Thread(target=worker.start, name=f"Worker-{i+1}")
            self.worker_threads.append(thread)
            thread.start()
        
        # Wait for all workers to complete
        combined_stats = {
            "total_workers": self.num_workers,
            "documents_processed": 0,
            "documents_failed": 0,
            "elements_created": 0,
            "relationships_created": 0,
            "links_discovered": 0
        }
        
        for thread in self.worker_threads:
            thread.join()
        
        # Combine statistics from all workers
        for worker in self.workers:
            for key in ["documents_processed", "documents_failed", "elements_created", 
                       "relationships_created", "links_discovered"]:
                combined_stats[key] += worker.stats.get(key, 0)
        
        logger.info(
            f"All workers completed. Processed {combined_stats['documents_processed']} documents "
            f"with {combined_stats['documents_failed']} failures"
        )
        
        return combined_stats
    
    def stop_all(self):
        """Stop all workers gracefully."""
        logger.info("Stopping all workers")
        
        for worker in self.workers:
            worker.stop()
        
        # Wait for threads to finish
        for thread in self.worker_threads:
            thread.join(timeout=30)  # 30 second timeout
        
        logger.info("All workers stopped")