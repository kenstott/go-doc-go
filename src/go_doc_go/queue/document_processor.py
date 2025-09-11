"""
Queue-enabled document processor for distributed document processing.
"""

import logging
import time
from typing import Dict, Any, Optional, List

from ..document_parser.factory import get_parser_for_content
from ..embeddings import EmbeddingGenerator
from ..relationships import RelationshipDetector
from .work_queue import WorkQueue
from .dead_letter import DeadLetterQueue

logger = logging.getLogger(__name__)


class QueuedDocumentProcessor:
    """
    Document processor that claims work from a queue and processes documents
    in a distributed manner with link discovery capabilities.
    """
    
    def __init__(self, db, work_queue: WorkQueue, relationship_detector: RelationshipDetector,
                 embedding_generator: Optional[EmbeddingGenerator] = None):
        """
        Initialize the queued document processor.
        
        Args:
            db: Database connection for storing processed documents
            work_queue: WorkQueue instance for claiming and managing work
            relationship_detector: Detector for document relationships
            embedding_generator: Optional embedding generator
        """
        self.db = db
        self.work_queue = work_queue
        self.relationship_detector = relationship_detector
        self.embedding_generator = embedding_generator
        self.worker_id = work_queue.worker_id
        self.dead_letter_queue = DeadLetterQueue(db)
        
        logger.info(f"Initialized QueuedDocumentProcessor for worker {self.worker_id}")
    
    def process_documents(self, run_id: str, max_documents: Optional[int] = None) -> Dict[str, Any]:
        """
        Process documents from the queue until no more work is available.
        
        Args:
            run_id: Processing run ID
            max_documents: Optional limit on number of documents to process
            
        Returns:
            Processing statistics
        """
        stats = {
            "documents_processed": 0,
            "documents_failed": 0,
            "elements_created": 0,
            "relationships_created": 0,
            "links_discovered": 0
        }
        
        logger.info(f"Worker {self.worker_id} starting document processing for run {run_id}")
        
        documents_processed = 0
        while max_documents is None or documents_processed < max_documents:
            # Claim next document from queue
            claimed_doc = self.work_queue.claim_next_document(run_id)
            
            if not claimed_doc:
                logger.debug(f"No more work available for worker {self.worker_id}")
                break
            
            doc_id = claimed_doc['doc_id']
            queue_id = claimed_doc['queue_id']
            source_name = claimed_doc['source_name']
            
            logger.info(f"Worker {self.worker_id} processing document: {doc_id} (queue_id: {queue_id})")
            
            try:
                # Process the document
                processing_result = self._process_single_document(
                    doc_id, source_name, claimed_doc, run_id
                )
                
                # Update statistics
                stats["documents_processed"] += 1
                stats["elements_created"] += processing_result.get("elements_created", 0)
                stats["relationships_created"] += processing_result.get("relationships_created", 0)
                stats["links_discovered"] += processing_result.get("links_discovered", 0)
                
                # Mark as completed in queue
                content_hash = processing_result.get("content_hash")
                file_size = processing_result.get("file_size")
                self.work_queue.mark_completed(queue_id, content_hash, file_size)
                
                logger.info(f"Worker {self.worker_id} completed document: {doc_id}")
                
            except Exception as e:
                logger.error(f"Worker {self.worker_id} failed to process document {doc_id}: {str(e)}")
                
                # Mark as failed in queue
                error_details = {
                    "error_type": type(e).__name__,
                    "worker_id": self.worker_id,
                    "timestamp": time.time()
                }
                
                # Check if this document should go to dead letter queue
                # Move to dead letter if max retries exceeded or critical error
                retry_count = claimed_doc.get('retry_count', 0)
                max_retries = self.work_queue.max_retries
                
                if retry_count >= max_retries or self._is_critical_error(e):
                    logger.warning(f"Moving document {doc_id} to dead letter queue after {retry_count} retries")
                    self.dead_letter_queue.move_to_dead_letter(
                        queue_id=queue_id,
                        error_message=str(e),
                        error_details=error_details
                    )
                else:
                    # Mark for retry
                    self.work_queue.mark_failed(queue_id, str(e), error_details)
                
                stats["documents_failed"] += 1
            
            documents_processed += 1
            
            # Send heartbeat periodically
            if documents_processed % 10 == 0:
                self.work_queue.heartbeat(run_id)
        
        logger.info(
            f"Worker {self.worker_id} completed processing: "
            f"{stats['documents_processed']} processed, {stats['documents_failed']} failed"
        )
        
        return stats
    
    def _process_single_document(self, doc_id: str, source_name: str, 
                                claimed_doc: Dict[str, Any], run_id: str) -> Dict[str, Any]:
        """
        Process a single claimed document.
        
        Args:
            doc_id: Document ID
            source_name: Name of the content source
            claimed_doc: Claimed document info from queue
            run_id: Processing run ID
            
        Returns:
            Processing results with statistics
        """
        # Get content source to fetch document
        from ..content_source.factory import get_content_source_by_name
        
        try:
            content_source = get_content_source_by_name(source_name)
        except Exception as e:
            raise RuntimeError(f"Failed to get content source '{source_name}': {str(e)}")
        
        # Fetch document content
        try:
            doc_content = content_source.fetch_document(doc_id)
        except Exception as e:
            raise RuntimeError(f"Failed to fetch document content: {str(e)}")
        
        # Check if document is unchanged (skip if so)
        last_processed_info = self.db.get_last_processed_info(doc_id)
        if last_processed_info:
            try:
                # Check modification time
                if not content_source.has_changed(doc_id, last_processed_info.get("last_modified")):
                    logger.info(f"Document unchanged since last processing: {doc_id}")
                    return {
                        "elements_created": 0,
                        "relationships_created": 0, 
                        "links_discovered": 0,
                        "content_hash": last_processed_info.get("content_hash"),
                        "file_size": last_processed_info.get("file_size")
                    }
                
                # Check content hash if available
                if ("content_hash" in last_processed_info and 
                    doc_content.get("content_hash") == last_processed_info["content_hash"]):
                    logger.info(f"Document content unchanged (verified by hash): {doc_id}")
                    return {
                        "elements_created": 0,
                        "relationships_created": 0,
                        "links_discovered": 0,
                        "content_hash": doc_content.get("content_hash"),
                        "file_size": doc_content.get("file_size")
                    }
            except Exception as e:
                logger.warning(f"Error checking if document changed: {str(e)}")
                # Continue with processing
        
        # Create parser and parse document
        parser = get_parser_for_content(doc_content)
        parsed_doc = parser.parse(doc_content)
        
        logger.debug(f"Parsed document {doc_id}: {len(parsed_doc.get('elements', []))} elements")
        
        # Detect relationships
        links = parsed_doc.get('links', [])
        relationships = parsed_doc.get('relationships', [])
        element_dates = parsed_doc.get('element_dates', [])
        
        relationships.extend(self.relationship_detector.detect_relationships(
            parsed_doc['document'],
            parsed_doc['elements'],
            links
        ))
        
        # Store document in database
        self.db.store_document(
            parsed_doc['document'], 
            parsed_doc['elements'], 
            relationships, 
            element_dates
        )
        
        # Update processing history
        content_hash = doc_content.get("content_hash", "")
        if content_hash:
            self.db.update_processing_history(doc_id, content_hash)
        
        # Generate embeddings if enabled
        embeddings_created = 0
        if self.embedding_generator:
            embeddings = self.embedding_generator.generate_from_elements(parsed_doc['elements'], self.db)
            
            # Store embeddings
            for element_id, embedding in embeddings.items():
                self.db.store_embedding(element_id, embedding)
            
            embeddings_created = len(embeddings)
            logger.debug(f"Generated {embeddings_created} embeddings for document {doc_id}")
        
        # Discover and queue linked documents
        links_added = self._discover_and_queue_links(
            content_source, doc_content, doc_id, run_id, claimed_doc
        )
        
        return {
            "elements_created": len(parsed_doc['elements']),
            "relationships_created": len(relationships),
            "links_discovered": links_added,
            "embeddings_created": embeddings_created,
            "content_hash": doc_content.get("content_hash"),
            "file_size": doc_content.get("file_size", 0)
        }
    
    def _discover_and_queue_links(self, content_source, doc_content: Dict[str, Any], 
                                 doc_id: str, run_id: str, claimed_doc: Dict[str, Any]) -> int:
        """
        Discover linked documents and add them to the queue.
        
        Args:
            content_source: Content source for following links
            doc_content: Document content
            doc_id: Current document ID
            run_id: Processing run ID
            claimed_doc: Claimed document information
            
        Returns:
            Number of linked documents added to queue
        """
        current_depth = claimed_doc.get('link_depth', 0)
        max_depth = claimed_doc.get('metadata', {}).get('max_link_depth', 1)
        
        if current_depth >= max_depth:
            logger.debug(f"Not following links - at max depth {current_depth}/{max_depth}")
            return 0
        
        try:
            # Use content source's follow_links method
            linked_docs = content_source.follow_links(
                doc_content.get('content', ''),
                doc_id,
                current_depth,
                set()  # Global visited tracking handled by queue
            )
            
            links_added = 0
            for linked_doc in linked_docs:
                linked_id = linked_doc['id']
                
                # Add to queue with increased depth
                success = self.work_queue.add_linked_document(
                    parent_doc_id=doc_id,
                    child_doc_id=linked_id,
                    source_name=claimed_doc['source_name'],
                    run_id=run_id,
                    link_depth=current_depth + 1
                )
                
                if success:
                    links_added += 1
                    logger.debug(f"Added linked document to queue: {linked_id} (depth: {current_depth + 1})")
            
            logger.info(f"Discovered and queued {links_added} linked documents from {doc_id}")
            return links_added
            
        except Exception as e:
            logger.warning(f"Error discovering links from document {doc_id}: {str(e)}")
            return 0
    
    def _is_critical_error(self, exception: Exception) -> bool:
        """
        Determine if an error is critical and should immediately go to dead letter queue.
        
        Args:
            exception: The exception that occurred
            
        Returns:
            True if this is a critical error that shouldn't be retried
        """
        critical_error_types = [
            # File format errors - unlikely to be fixed by retry
            "UnsupportedFileFormatError",
            "InvalidDocumentFormatError",
            "CorruptedFileError",
            
            # Parser-specific errors
            "ParserConfigurationError",
            "UnsupportedDocumentTypeError",
            
            # Authentication/Permission errors
            "PermissionError", 
            "AuthenticationError",
            "AccessDeniedError",
            
            # Configuration errors
            "ConfigurationError",
            "InvalidConfigError"
        ]
        
        error_type = type(exception).__name__
        if error_type in critical_error_types:
            return True
        
        # Check for specific error messages that indicate critical issues
        error_msg = str(exception).lower()
        critical_messages = [
            "permission denied",
            "access denied", 
            "authentication failed",
            "invalid format",
            "corrupted file",
            "unsupported format"
        ]
        
        return any(msg in error_msg for msg in critical_messages)