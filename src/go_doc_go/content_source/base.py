import hashlib
from abc import ABC, abstractmethod
from typing import Dict, Any, List, Optional


class ContentSource(ABC):
    """Abstract base class for content sources."""

    def __init__(self, config: Dict[str, Any]):
        """
        Initialize the content source with configuration.

        Args:
            config: Configuration dictionary for the content source
        """
        self.config = config
        self.name = config.get("name", "unnamed-source")
        self.max_link_depth = config.get("max_link_depth", 1)
        self.discovery_interval = config.get("discovery_interval", 300)  # 5 minutes default

    @abstractmethod
    def fetch_document(self, source_id: str) -> Dict[str, Any]:
        """
        Fetch document content from the source.

        Args:
            source_id: Identifier for the document in this source

        Returns:
            Dictionary containing document content and metadata
        """
        pass

    @abstractmethod
    def list_documents(self) -> List[Dict[str, Any]]:
        """
        List available documents in this source.

        Returns:
            List of document identifiers and metadata
        """
        pass

    @abstractmethod
    def has_changed(self, source_id: str, last_modified: Optional[float] = None) -> bool:
        """
        Check if a document has changed since last processing.

        Args:
            source_id: Identifier for the document
            last_modified: Timestamp of last known modification

        Returns:
            True if document has changed, False otherwise
        """
        pass

    def supports_continuous_discovery(self) -> bool:
        """
        Check if this content source supports continuous discovery.

        Returns:
            True if source can discover new documents over time, False otherwise
        """
        return False

    def discover_new_documents(self, last_discovery_time: Optional[float] = None) -> List[Dict[str, Any]]:
        """
        Discover new documents that have appeared since last discovery.

        Args:
            last_discovery_time: Unix timestamp of last discovery (None for first discovery)

        Returns:
            List of newly discovered documents
        """
        # Default implementation just returns all documents
        # Override in subclasses for incremental discovery
        return self.list_documents()

    def follow_links(self, content: str, source_id: str, current_depth: int = 0, global_visited_docs=None) -> List[
        Dict[str, Any]]:
        """
        Extract and follow links in document content with global visited tracking.

        Args:
            content: Document content
            source_id: Identifier for the source document
            current_depth: Current depth of link following
            global_visited_docs: Global set of all visited document IDs

        Returns:
            List of linked documents
        """
        import logging
        logger = logging.getLogger(__name__)

        if current_depth >= self.max_link_depth:
            logger.debug(f"Max link depth {self.max_link_depth} reached for {source_id}")
            return []

        # Initialize global visited set if not provided
        if global_visited_docs is None:
            global_visited_docs = set()

        # Add current document to global visited set
        global_visited_docs.add(source_id)

        # Default implementation does nothing
        logger.debug(f"Base follow_links called, no implementation for source type: {self.__class__.__name__}")
        return []

    @staticmethod
    def get_content_hash(content: str) -> str:
        """
        Generate a hash of content for change detection.

        Args:
            content: Document content

        Returns:
            MD5 hash of content
        """
        return hashlib.md5(content.encode('utf-8')).hexdigest()
