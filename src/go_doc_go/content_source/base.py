import hashlib
from abc import ABC, abstractmethod
from typing import Dict, Any, List, Optional, TYPE_CHECKING

if TYPE_CHECKING:
    from go_doc_go.shared.simple_job_control import SimpleJobControl


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
        self.job_control = None  # Will be injected by worker for asynchronous link queuing

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

    def set_job_control(self, job_control: 'SimpleJobControl'):
        """
        Inject job control for asynchronous link queuing.

        Args:
            job_control: Job control instance for queuing discovered links
        """
        self.job_control = job_control

    def queue_discovered_link(self, link_url: str, source_id: str, depth: int) -> bool:
        """
        Queue a discovered link for asynchronous processing by other workers.

        Args:
            link_url: URL of discovered link
            source_id: Source document that contained the link
            depth: Current link depth

        Returns:
            True if link was queued, False if not (no job control or depth exceeded)
        """
        # Check if we've reached max depth BEFORE queuing
        if not self.job_control or (depth + 1) > self.max_link_depth:
            import logging
            logger = logging.getLogger(__name__)
            if (depth + 1) > self.max_link_depth:
                logger.debug(f"Not queuing {link_url} - would exceed max_link_depth {self.max_link_depth} (current depth: {depth})")
            return False

        try:
            # Queue the link as a new document with proper depth tracking
            metadata = {
                "url": link_url,
                "parent_url": source_id,
                "discovery_depth": depth + 1,
                "max_link_depth": self.max_link_depth,  # Pass max depth to queued document
                "source_name": self.name
            }

            self.job_control.enqueue_document(link_url, self.name, metadata)

            import logging
            logger = logging.getLogger(__name__)
            logger.debug(f"Queued link {link_url} at depth {depth + 1} (max: {self.max_link_depth})")
            return True

        except Exception as e:
            import logging
            logger = logging.getLogger(__name__)
            logger.warning(f"Failed to queue discovered link {link_url}: {str(e)}")
            return False

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
