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
        
        # Get embedding generator if available (for future semantic search in analytics backends)
        if self.config.get('embedding', {}).get('enabled', True):
            try:
                from go_doc_go.embeddings.fastembed import FastEmbedGenerator
                from go_doc_go import Config
                # Create a Config object if needed for FastEmbedGenerator
                config_obj = self._config_obj if self._config_obj else Config()
                self._embedder = FastEmbedGenerator(_config=config_obj)
                logger.debug("Embedding generator initialized for semantic search")
            except ImportError:
                logger.debug("FastEmbed not available, embeddings disabled")
                self._embedder = None
    
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
        
        # Create adapter based on backend type
        # For now, return a stub adapter for demonstration
        if backend_type == 'parquet':
            return ParquetDuckDBSearchAdapter(backend_config, self._embedder)
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
            
            # Add full content if requested and available
            if request.include_content and 'content' in result:
                hit.content = result.get('content', '')
            
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
            
            # Add full content if requested and available
            if request.include_content and 'content' in result:
                hit.content = result.get('content', '')
            
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
            if hit.element_type != filters['element_type']:
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
            # Note: Filters temporarily disabled as columns don't match Parquet schema
            results = self.storage.search_semantic(
                query_embedding=query_embedding,
                limit=limit,
                min_similarity=similarity_threshold,
                filters=None,  # Temporarily disabled - need column mapping
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