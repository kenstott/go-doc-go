"""
Abstract base class for simplified job control database backends.
"""

import logging
from abc import ABC, abstractmethod
from typing import Dict, Any, Optional, List

logger = logging.getLogger(__name__)


class SimpleJobControlDB(ABC):
    """Abstract base class for simplified job control database backends."""

    @classmethod
    def create(cls, config) -> 'SimpleJobControlDB':
        """Factory method to create appropriate backend from config."""
        from .sqlite import SimpleSQLiteJobControlDB
        from .sqlalchemy_impl import SQLAlchemyJobControlDB

        job_control_config = config.get_job_control_config()
        backend = job_control_config.get('backend', 'sqlite')

        if backend == 'sqlite':
            return SimpleSQLiteJobControlDB(config)
        elif backend in ['postgresql', 'mysql', 'mssql', 'oracle', 'sqlalchemy']:
            return SQLAlchemyJobControlDB(config)
        else:
            raise ValueError(f"Unsupported job control backend: {backend}. Supported: sqlite, postgresql, mysql, mssql, oracle, sqlalchemy")

    @abstractmethod
    def initialize_schema(self):
        """Create database schema if it doesn't exist."""
        pass

    @abstractmethod
    def enqueue_document(self, doc_id: str, source: str, metadata: Dict[str, Any]):
        """Add a document to the processing queue."""
        pass

    @abstractmethod
    def claim_next_document(self, worker_id: str) -> Optional[Dict[str, Any]]:
        """Atomically claim next available document for processing."""
        pass

    @abstractmethod
    def complete_document(self, doc_id: str, worker_id: str, success: bool, error_message: Optional[str] = None):
        """Mark a document as completed (successfully or failed)."""
        pass

    @abstractmethod
    def release_document(self, doc_id: str, worker_id: str):
        """Release a claimed document back to the queue."""
        pass

    @abstractmethod
    def register_worker(self, worker_id: str, worker_info: Dict[str, Any]):
        """Register a worker in the system."""
        pass

    @abstractmethod
    def update_worker_heartbeat(self, worker_id: str):
        """Update worker heartbeat timestamp."""
        pass

    @abstractmethod
    def get_processing_status(self) -> Dict[str, Any]:
        """Get overall processing status."""
        pass

    @abstractmethod
    def cleanup_stale_claims(self, timeout_seconds: int = 300):
        """Release documents claimed by inactive workers."""
        pass

    @abstractmethod
    def is_document_queued(self, doc_id: str) -> bool:
        """Check if document is already in the queue."""
        pass

    @abstractmethod
    def elect_leader(self, worker_id: str, worker_info: Dict[str, Any]) -> bool:
        """Attempt to elect this worker as leader. Returns True if successful."""
        pass

    @abstractmethod
    def is_leader(self, worker_id: str) -> bool:
        """Check if the given worker is the current leader."""
        pass

    @abstractmethod
    def update_leader_heartbeat(self, worker_id: str):
        """Update leader heartbeat timestamp."""
        pass

    @abstractmethod
    def get_current_leader(self) -> Optional[Dict[str, Any]]:
        """Get information about the current leader, if any."""
        pass

    @abstractmethod
    def release_leadership(self, worker_id: str):
        """Release leadership role (for graceful shutdown)."""
        pass

    @abstractmethod
    def store_document_metadata(self, doc_id: str, source: str,
                               last_modified: Optional[float] = None,
                               content_hash: Optional[str] = None,
                               file_size: Optional[int] = None,
                               processing_stats: Optional[Dict[str, Any]] = None):
        """Store document metadata for change tracking."""
        pass

    @abstractmethod
    def get_document_metadata(self, doc_id: str) -> Optional[Dict[str, Any]]:
        """Get stored document metadata."""
        pass

    @abstractmethod
    def has_document_changed(self, doc_id: str, source: str,
                            current_modified: Optional[float] = None,
                            current_hash: Optional[str] = None) -> bool:
        """Check if document has changed since last processing."""
        pass

    @abstractmethod
    def get_source_documents(self, source: str, pattern: Optional[str] = None) -> List[Dict[str, Any]]:
        """Get all document metadata for a specific source.

        Args:
            source: The content source name
            pattern: Optional SQL LIKE pattern for filtering doc_ids

        Returns:
            List of document metadata dictionaries
        """
        pass

    @abstractmethod
    def get_document_statistics(self) -> Dict[str, Any]:
        """Get overall document processing statistics.

        Returns:
            Dictionary with statistics including:
            - total_documents: Total number of documents
            - documents_by_source: Count per source
            - recently_processed: Documents processed in last hour
            - documents_with_changes: Documents with multiple versions
        """
        pass