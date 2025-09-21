"""
Simplified search module with a single, unified search function.
"""

from typing import Dict, List, Any, Optional, Union
from dataclasses import dataclass, field
import json
import logging
from abc import ABC, abstractmethod

logger = logging.getLogger(__name__)


@dataclass
class SearchRequest:
    """Unified search request structure."""
    search_service: str  # Required: name of registered analytics backend
    similarity_query: str
    limit: int = 10
    offset: int = 0
    filters: Dict[str, Any] = field(default_factory=dict)
    similarity_threshold: float = 0.7
    include_content: bool = False
    include_metadata: bool = True


@dataclass
class SearchHit:
    """Individual search result."""
    element_id: str
    doc_id: str
    score: float
    content_preview: str
    element_type: str
    metadata: Dict[str, Any] = field(default_factory=dict)
    content: Optional[str] = None


@dataclass
class SearchResponse:
    """Search response with results and metadata."""
    similarity_query: str
    hits: List[SearchHit]
    total_hits: int
    took_ms: int
    filters_applied: Dict[str, Any]


class SearchEngine:
    """
    Main search engine class with a single, unified search interface.
    """
    
    def __init__(self, config: Optional[Dict[str, Any]] = None):
        """
        Initialize the search engine.

        Args:
            config: Optional configuration dictionary
        """
        self.config = config or {}
        self._embedder = None
        self._config_obj = None  # Config object for analytics backend access
        self._analytics_adapters = {}  # Cache for analytics backend adapters
        self._content_resolver = None  # Content resolver for full content retrieval
        self._initialize_components()
    
    def _initialize_components(self):
        """Initialize analytics backend registry and optional embedding components."""
        # Initialize Config object for analytics backend access
        try:
            from go_doc_go import Config
            self._config_obj = Config()
            logger.debug(f"Loaded {len(self._config_obj.list_analytics_backends())} analytics backends")
        except Exception as e:
            logger.warning(f"Could not load Config for analytics backends: {e}")
            self._config_obj = None
        
        # Backend-specific embedders will be created on-demand using analytics registry
        self._backend_embedders = {}  # Cache for backend-specific embedders
        logger.debug("SearchEngine initialized with analytics registry-based embedding configuration")
    
    def search(self, request: Union[SearchRequest, Dict[str, Any]]) -> SearchResponse:
        """
        Perform a unified search across the document database.
        
        This is the single entry point for all search operations. It handles:
        - Semantic/vector search using embeddings
        - Filtering by document properties, dates, types, etc.
        - Content retrieval and enrichment
        - Result ranking and pagination
        
        Args:
            request: SearchRequest object or dictionary with search parameters
        
        Returns:
            SearchResponse with ranked results and metadata
        
        Example:
            >>> engine = SearchEngine()
            >>> response = engine.search(SearchRequest(
            ...     search_service="parquet_duckdb",
            ...     similarity_query="machine learning algorithms",
            ...     limit=10,
            ...     filters={"doc_type": "pdf", "date_after": "2024-01-01"}
            ... ))
            >>> for hit in response.hits:
            ...     print(f"{hit.score:.2f}: {hit.content_preview}")
        """
        import time
        start_time = time.time()
        
        # Convert dict to SearchRequest if needed
        if isinstance(request, dict):
            request = SearchRequest(**request)
        
        # Validate request
        if not request.similarity_query:
            raise ValueError("Search similarity_query cannot be empty")
        
        # Validate search_service against registered analytics backends
        self._validate_search_service(request.search_service)
        
        # Perform the search
        hits = self._execute_search(request)
        
        # Calculate timing
        took_ms = int((time.time() - start_time) * 1000)
        
        # Build response
        return SearchResponse(
            similarity_query=request.similarity_query,
            hits=hits[:request.limit],
            total_hits=len(hits),
            took_ms=took_ms,
            filters_applied=request.filters
        )
    
    def _validate_search_service(self, search_service: str):
        """
        Validate search_service against registered analytics backends.
        
        Args:
            search_service: Name of the analytics backend to validate
        
        Raises:
            ValueError: If search_service is not a registered analytics backend
        """
        if not self._config_obj:
            logger.warning("Config not available, skipping search_service validation")
            return
        
        available_backends = self._config_obj.list_analytics_backends()
        if search_service not in available_backends:
            raise ValueError(
                f"Unknown search_service '{search_service}'. "
                f"Available backends: {', '.join(available_backends.keys())}"
            )
    
    def _get_or_create_adapter(self, search_service: str):
        """
        Get or create an analytics backend adapter for the given service.
        
        Args:
            search_service: Name of the analytics backend
        
        Returns:
            Analytics backend adapter instance or None if creation fails
        """
        # Check cache first
        if search_service in self._analytics_adapters:
            return self._analytics_adapters[search_service]
        
        # Create new adapter based on backend type
        adapter = self._create_adapter(search_service)
        if adapter:
            self._analytics_adapters[search_service] = adapter
        
        return adapter
    
    def _get_embedder_for_backend(self, backend_name: str):
        """
        Get or create embedder for specific analytics backend using analytics registry.
        
        Args:
            backend_name: Name of the analytics backend
        
        Returns:
            EmbeddingGenerator instance or None if creation fails
        """
        # Check cache first
        if backend_name in self._backend_embedders:
            return self._backend_embedders[backend_name]
        
        if not self._config_obj:
            logger.error("Config not available, cannot create embedder")
            return None
        
        # Get backend configuration from analytics registry
        backends = self._config_obj.list_analytics_backends()
        backend_config = backends.get(backend_name)
        
        if not backend_config:
            logger.error(f"Backend configuration not found for: {backend_name}")
            return None
        
        # Get embedding configuration from backend
        embedding_config = backend_config.get('embedding', {})
        
        if not embedding_config:
            logger.warning(f"No embedding configuration found for backend: {backend_name}")
            return None
        
        # Create embedder using analytics registry configuration
        try:
            from .embeddings.factory import get_embedder_from_analytics_registry
            embedder = get_embedder_from_analytics_registry(backend_name, embedding_config, self._config_obj)
            
            if embedder:
                self._backend_embedders[backend_name] = embedder
                logger.info(f"Created embedder for backend '{backend_name}' using model: {embedding_config.get('model', 'unknown')}")
            
            return embedder
            
        except ImportError as e:
            logger.error(f"Failed to import embedding factory: {e}")
            return None
        except Exception as e:
            logger.error(f"Failed to create embedder for backend '{backend_name}': {e}")
            return None
    
    def _create_adapter(self, search_service: str):
        """
        Create a new analytics backend adapter.
        
        Args:
            search_service: Name of the analytics backend
        
        Returns:
            Analytics backend adapter instance or None if creation fails
        """
        if not self._config_obj:
            logger.error("Config not available, cannot create adapter")
            return None
        
        backends = self._config_obj.list_analytics_backends()
        backend_config = backends.get(search_service)
        
        if not backend_config:
            logger.error(f"Backend configuration not found for: {search_service}")
            return None
        
        backend_type = backend_config.get('type')
        
        # Get backend-specific embedder
        backend_embedder = self._get_embedder_for_backend(search_service)
        
        if not backend_embedder:
            logger.error(f"Failed to create embedder for backend: {search_service}")
            return None
        
        # Create adapter based on backend type
        if backend_type == 'parquet':
            return ParquetDuckDBSearchAdapter(backend_config, backend_embedder)
        elif backend_type == 'elasticsearch':
            # Future: return ElasticsearchAdapter(backend_config)
            logger.info(f"Elasticsearch adapter not yet implemented")
            return None
        elif backend_type == 'mongodb':
            # Future: return MongoDBAdapter(backend_config)
            logger.info(f"MongoDB adapter not yet implemented")
            return None
        else:
            logger.warning(f"Unknown backend type: {backend_type}")
            return None
    
    def _create_hit_from_adapter_result(self, result: Dict[str, Any], request: SearchRequest) -> Optional[SearchHit]:
        """
        Create a SearchHit from an analytics backend adapter result.
        
        Args:
            result: Result from analytics backend adapter
            request: Original search request
        
        Returns:
            SearchHit object or None if invalid
        """
        try:
            # Build metadata from result
            metadata = result.get('metadata', {})
            
            # Parse metadata if it's a JSON string
            if isinstance(metadata, str):
                try:
                    metadata = json.loads(metadata)
                except:
                    metadata = {}
            
            # Add content_location to metadata if available
            if 'content_location' in result and result['content_location']:
                metadata['content_location'] = result['content_location']
            
            # Analytics backends should return results in a standard format
            hit = SearchHit(
                element_id=result.get('element_id', ''),
                doc_id=result.get('doc_id', ''),
                score=float(result.get('score', 0.0)),
                content_preview=result.get('content_preview', '')[:200],
                element_type=result.get('element_type', 'unknown'),
                metadata=metadata
            )
            
            # Add full content if requested
            if request.include_content:
                if 'content' in result:
                    # Content already available in result
                    hit.content = result.get('content', '')
                else:
                    # Try to resolve content from content_location
                    content_location = metadata.get('content_location')
                    if content_location:
                        hit.content = self._resolve_full_content(content_location)
                    else:
                        # No content available, use preview as fallback
                        hit.content = hit.content_preview
            
            return hit
            
        except Exception as e:
            logger.debug(f"Could not create hit from adapter result: {e}")
            return None
    
    def _execute_search(self, request: SearchRequest) -> List[SearchHit]:
        """
        Execute the actual search operation by routing to the appropriate analytics backend.
        
        Args:
            request: Validated search request
        
        Returns:
            List of search hits, ranked by relevance
        """
        hits = []
        
        # Get the analytics backend adapter for this search_service
        adapter = self._get_or_create_adapter(request.search_service)
        if not adapter:
            logger.error(f"Could not create adapter for search_service: {request.search_service}")
            return hits
        
        try:
            # Execute search through the analytics backend adapter
            results = adapter.search(
                query_text=request.similarity_query,
                limit=request.limit,
                offset=request.offset,
                filters=request.filters,
                similarity_threshold=request.similarity_threshold,
                include_content=request.include_content,
                include_metadata=request.include_metadata
            )
            
            # Convert adapter results to SearchHit objects
            for result in results:
                hit = self._create_hit_from_adapter_result(result, request)
                if hit:
                    hits.append(hit)
                    
        except NotImplementedError:
            logger.warning(f"Search not yet implemented for backend: {request.search_service}")
            # Return empty results for unimplemented backends
            pass
        except Exception as e:
            logger.error(f"Search execution failed for {request.search_service}: {e}")
            # Return empty results on error
            pass
        
        # Results should already be sorted by the backend, but ensure it
        hits.sort(key=lambda h: h.score, reverse=True)
        
        return hits
    
    def _create_hit_from_result(self, result: Dict[str, Any], request: SearchRequest) -> Optional[SearchHit]:
        """
        Create a SearchHit from a raw database result.

        Args:
            result: Raw result from database
            request: Original search request

        Returns:
            SearchHit object or None if invalid
        """
        try:
            # Extract basic information from result
            # The result structure depends on the search implementation
            element_id = result.get('element_id', '')
            doc_id = result.get('doc_id', '')
            score = result.get('similarity', result.get('score', 0.0))

            # Build the hit with available information
            hit = SearchHit(
                element_id=element_id or f"element_{result.get('element_pk', 'unknown')}",
                doc_id=doc_id or 'unknown',
                score=float(score),
                content_preview=result.get('content_preview', result.get('text', ''))[:200],
                element_type=result.get('element_type', 'unknown'),
                metadata=result.get('metadata', {})
            )

            # Add full content if requested
            if request.include_content:
                if 'content' in result:
                    # Content already available in result
                    hit.content = result.get('content', '')
                else:
                    # Try to resolve content from content_location
                    content_location = hit.metadata.get('content_location')
                    if content_location:
                        hit.content = self._resolve_full_content(content_location)
                    else:
                        # No content available, use preview as fallback
                        hit.content = hit.content_preview

            return hit

        except Exception as e:
            logger.debug(f"Could not create hit from result: {e}")
            return None
    
    def _apply_filters(self, hit: SearchHit, filters: Dict[str, Any]) -> bool:
        """
        Apply filters to a search hit.
        
        Args:
            hit: Search hit to filter
            filters: Filter criteria
        
        Returns:
            True if hit passes all filters, False otherwise
        """
        if not filters:
            return True
        
        # Document type filter
        if 'doc_type' in filters:
            if hit.metadata.get('document_type') != filters['doc_type']:
                return False
        
        # Element type filter
        if 'element_type' in filters:
            element_types = filters['element_type']
            # Handle both single string and list of strings
            if isinstance(element_types, str):
                if hit.element_type != element_types:
                    return False
            elif isinstance(element_types, list):
                if hit.element_type not in element_types:
                    return False
        
        # Date filters
        if 'date_after' in filters:
            # Would need to parse and compare dates
            pass
        
        if 'date_before' in filters:
            # Would need to parse and compare dates
            pass
        
        # Custom metadata filters
        if 'metadata' in filters:
            element_meta = hit.metadata.get('element_metadata', {})
            for key, value in filters['metadata'].items():
                if element_meta.get(key) != value:
                    return False
        
        return True

    def _resolve_full_content(self, content_location: str) -> Optional[str]:
        """
        Resolve full content from content_location using content resolver.

        Args:
            content_location: JSON string or dict containing content location info

        Returns:
            Full content string or None if resolution fails
        """
        try:
            # Parse content_location if it's a JSON string
            if isinstance(content_location, str):
                try:
                    content_location = json.loads(content_location)
                except:
                    logger.warning(f"Could not parse content_location: {content_location}")
                    return None

            # Initialize content resolver if not already done
            if not self._content_resolver:
                try:
                    from go_doc_go.adapter.factory import create_content_resolver
                    from go_doc_go import Config
                    config_obj = Config()
                    self._content_resolver = create_content_resolver(config_obj)
                    logger.info("Created content resolver for full content retrieval")
                except Exception as e:
                    logger.error(f"Failed to create content resolver: {e}")
                    return None

            # Try to resolve content using the resolver
            if self._content_resolver:
                try:
                    # Content resolver expects content_location as a string
                    location_str = json.dumps(content_location) if isinstance(content_location, dict) else content_location
                    full_content = self._content_resolver.resolve_content(location_str, text=True)
                    logger.debug(f"Successfully resolved content from location: {content_location.get('source', 'unknown')}")
                    return full_content
                except Exception as e:
                    logger.warning(f"Failed to resolve content from location: {e}")
                    return None

        except Exception as e:
            logger.error(f"Error in content resolution: {e}")

        return None


# Convenience function for simple searches
def search(search_service: str,
           similarity_query: str, 
           limit: int = 10,
           filters: Optional[Dict[str, Any]] = None,
           config: Optional[Dict[str, Any]] = None) -> SearchResponse:
    """
    Convenience function for performing a simple search.
    
    Args:
        search_service: Name of the registered analytics backend
        similarity_query: Search query string for similarity matching
        limit: Maximum number of results
        filters: Optional filter criteria
        config: Optional configuration
    
    Returns:
        SearchResponse with results
    
    Example:
        >>> results = search("parquet_duckdb", "machine learning", limit=5)
        >>> print(f"Found {results.total_hits} results")
    """
    engine = SearchEngine(config)
    request = SearchRequest(
        search_service=search_service,
        similarity_query=similarity_query,
        limit=limit,
        filters=filters or {}
    )
    return engine.search(request)


class AnalyticsSearchAdapter(ABC):
    """
    Base class for analytics backend search adapters.
    Each analytics backend (parquet, elasticsearch, mongodb, etc.) 
    implements this interface to provide search capabilities.
    """
    
    def __init__(self, config: Dict[str, Any], embedder=None):
        """
        Initialize the adapter.
        
        Args:
            config: Backend configuration from analytics registry
            embedder: Optional embedding generator for semantic search
        """
        self.config = config
        self.embedder = embedder
        self.backend_type = config.get('type', 'unknown')
        self.enabled = config.get('enabled', False)
    
    @abstractmethod
    def search(self, query_text: str, limit: int = 10, offset: int = 0,
               filters: Dict[str, Any] = None, similarity_threshold: float = 0.7,
               include_content: bool = False, include_metadata: bool = True) -> List[Dict[str, Any]]:
        """
        Execute a search query against the analytics backend.
        
        Args:
            query_text: The search query text
            limit: Maximum number of results
            offset: Number of results to skip
            filters: Optional filter criteria
            similarity_threshold: Minimum similarity score
            include_content: Whether to include full content
            include_metadata: Whether to include metadata
        
        Returns:
            List of search results in standard format
        """
        pass
    
    @abstractmethod
    def is_available(self) -> bool:
        """
        Check if the backend is available and ready for queries.
        
        Returns:
            True if backend is available, False otherwise
        """
        pass


class ParquetDuckDBSearchAdapter(AnalyticsSearchAdapter):
    """
    Search adapter for Parquet files using DuckDB.
    
    Uses the existing ParquetAnalyticsStorage for actual DuckDB operations,
    adding only the embedding generation layer for similarity search.
    """
    
    def __init__(self, config: Dict[str, Any], embedder=None):
        """Initialize the Parquet/DuckDB adapter."""
        super().__init__(config, embedder)
        
        # Use existing ParquetAnalyticsStorage instead of reimplementing
        try:
            from go_doc_go.storage_adapters.analytics_parquet import ParquetAnalyticsStorage
            # Pass the config with correct path key for ParquetAnalyticsStorage
            storage_config = {
                'path': config.get('base_path', './data-lake'),
                **config  # Include any other config options
            }
            self.storage = ParquetAnalyticsStorage(storage_config)
            logger.info(f"Initialized ParquetDuckDBSearchAdapter with base_path: {storage_config['path']}")
        except ImportError as e:
            logger.error(f"Failed to import ParquetAnalyticsStorage: {e}")
            self.storage = None
    
    def search(self, query_text: str, limit: int = 10, offset: int = 0,
               filters: Dict[str, Any] = None, similarity_threshold: float = 0.7,
               include_content: bool = False, include_metadata: bool = True) -> List[Dict[str, Any]]:
        """
        Execute similarity search against Parquet files using DuckDB.
        
        Generates embeddings for the query text and uses the existing
        ParquetAnalyticsStorage.search_semantic method for actual search.
        """
        logger.info(f"ParquetDuckDBSearchAdapter.search called with query: {query_text}")
        
        # Check if storage is available
        if not self.storage:
            logger.error("ParquetAnalyticsStorage not available")
            return []
        
        # Generate embedding for query text
        if not self.embedder:
            logger.error("Embedder required for similarity search but not available")
            return []
        
        try:
            # Generate query embedding
            query_embedding = self.embedder.generate(query_text)
            logger.debug(f"Generated embedding of dimension {len(query_embedding)} for query")
            
            # Use existing search_semantic method from ParquetAnalyticsStorage
            results = self.storage.search_semantic(
                query_embedding=query_embedding,
                limit=limit,
                min_similarity=similarity_threshold,
                filters=filters,  # Pass through filters including run_id
                include_context=include_metadata
            )
            
            logger.info(f"Found {len(results)} results from ParquetAnalyticsStorage")
            
            # Transform results to match expected format
            # The search_semantic method returns results with 'similarity' field
            # We need to rename it to 'score' for consistency with the interface
            for result in results:
                if 'similarity' in result:
                    result['score'] = result.pop('similarity')
            
            return results
            
        except Exception as e:
            logger.error(f"Error during similarity search: {e}")
            return []
    
    def is_available(self) -> bool:
        """
        Check if DuckDB and Parquet files are available.
        """
        # Check if the storage adapter is available
        return self.storage is not None and self.embedder is not None