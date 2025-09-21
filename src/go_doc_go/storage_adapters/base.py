"""
Base classes and interfaces for storage adapters.
"""

import logging
from abc import ABC, abstractmethod
from datetime import datetime
from typing import Dict, Any, List, Optional, Tuple
from dataclasses import dataclass

logger = logging.getLogger(__name__)


@dataclass
class StorageConfig:
    """Configuration for a storage backend."""
    backend: str
    config: Dict[str, Any]
    ttl: Optional[int] = None  # Time to live in seconds for job storage
    partitioning: Optional[List[str]] = None  # Partitioning scheme for analytics


class JobStorage(ABC):
    """
    Interface for job coordination storage (OLTP).
    
    Used for transient, mutable state during document processing:
    - Document queue status
    - Worker claims and heartbeats  
    - Processing state transitions
    - Retry tracking
    """
    
    def __init__(self, config: Dict[str, Any]):
        """Initialize job storage with configuration."""
        self.config = config
        self.ttl = config.get('ttl')
    
    @abstractmethod
    def initialize(self) -> None:
        """Initialize storage backend and create necessary schema."""
        pass
    
    @abstractmethod
    def claim_document(self, doc_id: str, worker_id: str, 
                      timeout: int = 300) -> Optional[Dict[str, Any]]:
        """
        Atomically claim a document for processing.
        
        Args:
            doc_id: Document identifier
            worker_id: Worker claiming the document
            timeout: Claim timeout in seconds
            
        Returns:
            Document metadata if claimed, None if already claimed
        """
        pass
    
    @abstractmethod
    def update_status(self, doc_id: str, status: str, 
                     metadata: Optional[Dict] = None) -> bool:
        """
        Update document processing status.
        
        Args:
            doc_id: Document identifier
            status: New status (pending, processing, completed, failed)
            metadata: Optional metadata to store
            
        Returns:
            True if updated successfully
        """
        pass
    
    @abstractmethod
    def mark_completed(self, doc_id: str, stats: Dict[str, Any]) -> bool:
        """
        Mark document as successfully processed.
        
        Args:
            doc_id: Document identifier
            stats: Processing statistics
            
        Returns:
            True if marked successfully
        """
        pass
    
    @abstractmethod
    def mark_failed(self, doc_id: str, error: str, 
                   retry: bool = True) -> bool:
        """
        Mark document as failed.
        
        Args:
            doc_id: Document identifier
            error: Error message
            retry: Whether to schedule for retry
            
        Returns:
            True if marked successfully
        """
        pass
    
    @abstractmethod
    def get_queue_status(self, run_id: str) -> Dict[str, int]:
        """
        Get current queue status.
        
        Args:
            run_id: Processing run identifier
            
        Returns:
            Dictionary with counts by status
        """
        pass
    
    @abstractmethod
    def heartbeat(self, worker_id: str) -> bool:
        """
        Send worker heartbeat.
        
        Args:
            worker_id: Worker identifier
            
        Returns:
            True if heartbeat recorded
        """
        pass
    
    @abstractmethod
    def cleanup(self, older_than: Optional[datetime] = None) -> int:
        """
        Clean up old job data.
        
        Args:
            older_than: Clean up data older than this timestamp
            
        Returns:
            Number of records cleaned up
        """
        pass
    
    @abstractmethod
    def close(self) -> None:
        """Close storage connections."""
        pass


class AnalyticsStorage(ABC):
    """
    Interface for analytical storage (OLAP).
    
    Used for permanent, append-only archive of processing results:
    - Parsed documents
    - Extracted elements with embeddings
    - Detected relationships
    - Processing metrics
    """
    
    def __init__(self, config: Dict[str, Any]):
        """Initialize analytics storage with configuration."""
        self.config = config
        self.partitioning = config.get('partitioning', [])
    
    @abstractmethod
    def initialize(self) -> None:
        """Initialize storage backend."""
        pass
    
    @abstractmethod
    def append_documents(self, documents: List[Dict[str, Any]], 
                        run_id: str) -> int:
        """
        Append documents to analytical storage.
        
        Args:
            documents: List of document records
            run_id: Processing run identifier
            
        Returns:
            Number of documents written
        """
        pass
    
    @abstractmethod
    def append_elements(self, elements: List[Dict[str, Any]], 
                       run_id: str) -> int:
        """
        Append elements to analytical storage.
        
        Args:
            elements: List of element records
            run_id: Processing run identifier
            
        Returns:
            Number of elements written
        """
        pass
    
    @abstractmethod
    def append_embeddings(self, embeddings: List[Dict[str, Any]], 
                         run_id: str) -> int:
        """
        Append embeddings to analytical storage.
        
        Args:
            embeddings: List of embedding records
            run_id: Processing run identifier
            
        Returns:
            Number of embeddings written
        """
        pass
    
    @abstractmethod
    def append_relationships(self, relationships: List[Dict[str, Any]], 
                           run_id: str) -> int:
        """
        Append relationships to analytical storage.
        
        Args:
            relationships: List of relationship records
            run_id: Processing run identifier
            
        Returns:
            Number of relationships written
        """
        pass
    
    def has_run(self, run_id: str) -> bool:
        """
        Check if storage has data for the given run_id.

        Args:
            run_id: Run ID to check

        Returns:
            True if storage has data for this run, False otherwise
        """
        # Default implementation checks if we can list any data for this run
        try:
            # Try to get elements for this run
            results = self.search_text(
                query="",  # Empty query
                filters={'_run_id': run_id},
                limit=1
            )
            return len(results) > 0
        except Exception:
            # If search fails, assume no data
            return False

    @abstractmethod
    def append_metrics(self, metrics: Dict[str, Any],
                      run_id: str) -> bool:
        """
        Append processing metrics.
        
        Args:
            metrics: Processing metrics
            run_id: Processing run identifier
            
        Returns:
            True if written successfully
        """
        pass
    
    @abstractmethod
    def get_partition_path(self, run_id: str, 
                          data_type: str) -> str:
        """
        Get partition path for data.
        
        Args:
            run_id: Processing run identifier
            data_type: Type of data (documents, elements, etc.)
            
        Returns:
            Partition path string
        """
        pass
    
    @abstractmethod
    def list_runs(self, start_date: Optional[datetime] = None,
                 end_date: Optional[datetime] = None) -> List[str]:
        """
        List processing runs in date range.
        
        Args:
            start_date: Start of date range
            end_date: End of date range
            
        Returns:
            List of run identifiers
        """
        pass
    
    @abstractmethod
    def get_run_stats(self, run_id: str) -> Optional[Dict[str, Any]]:
        """
        Get statistics for a processing run.
        
        Args:
            run_id: Processing run identifier
            
        Returns:
            Run statistics or None if not found
        """
        pass
    
    @abstractmethod
    def search_semantic(self, query_embedding: List[float], 
                       limit: int = 10,
                       min_similarity: float = 0.0,
                       filters: Optional[Dict[str, Any]] = None,
                       include_context: bool = True) -> List[Dict[str, Any]]:
        """
        Search for similar content using embeddings.
        
        Args:
            query_embedding: Query vector
            limit: Maximum results to return
            min_similarity: Minimum similarity threshold
            filters: Optional filters (element_type, doc_id, etc.)
            include_context: Include surrounding context (parent, siblings, children)
            
        Returns:
            List of matching elements with similarity scores and optional context
        """
        pass
    
    @abstractmethod
    def search_text(self, query: str,
                   limit: int = 10,
                   filters: Optional[Dict[str, Any]] = None) -> List[Dict[str, Any]]:
        """
        Full-text search across content.
        
        Args:
            query: Search query string
            limit: Maximum results to return
            filters: Optional filters
            
        Returns:
            List of matching elements
        """
        pass
    
    @abstractmethod
    def search_structured(self, criteria: Dict[str, Any],
                         limit: int = 10) -> List[Dict[str, Any]]:
        """
        Structured search with multiple criteria.
        
        Args:
            criteria: Search criteria (field conditions)
            limit: Maximum results to return
            
        Returns:
            List of matching elements
        """
        pass
    
    @abstractmethod
    def get_element_by_id(self, element_id: str) -> Optional[Dict[str, Any]]:
        """
        Retrieve a specific element by ID.
        
        Args:
            element_id: Element identifier
            
        Returns:
            Element data or None if not found
        """
        pass
    
    @abstractmethod
    def get_document_elements(self, doc_id: str) -> List[Dict[str, Any]]:
        """
        Get all elements for a document.
        
        Args:
            doc_id: Document identifier
            
        Returns:
            List of elements
        """
        pass
    
    @abstractmethod
    def close(self) -> None:
        """Close storage connections."""
        pass