"""
Storage adapter factory for creating analytics storage instances.
"""

import logging
from typing import Dict, Any

from .base import AnalyticsStorage

# Analytics storage implementations
from .analytics_parquet import ParquetAnalyticsStorage
from .analytics_mongodb import MongoDBAnalyticsStorage
from .analytics_elasticsearch import ElasticsearchAnalyticsStorage
from .analytics_solr import SolrAnalyticsStorage
from .analytics_neo4j import Neo4jAnalyticsStorage
from .analytics_sqlalchemy import SQLAlchemyAnalyticsStorage

logger = logging.getLogger(__name__)


class StorageFactory:
    """Factory for creating analytics storage instances."""

    # Registry of analytics storage implementations
    ANALYTICS_STORAGE_TYPES = {
        'parquet': ParquetAnalyticsStorage,
        'mongodb': MongoDBAnalyticsStorage,
        'mongo': MongoDBAnalyticsStorage,
        'elasticsearch': ElasticsearchAnalyticsStorage,
        'elastic': ElasticsearchAnalyticsStorage,
        'es': ElasticsearchAnalyticsStorage,
        'solr': SolrAnalyticsStorage,
        'neo4j': Neo4jAnalyticsStorage,
        'sqlalchemy': SQLAlchemyAnalyticsStorage,
        'sql': SQLAlchemyAnalyticsStorage
    }
    
    
    @classmethod
    def create_analytics_storage(cls, config: Dict[str, Any]) -> AnalyticsStorage:
        """
        Create an analytics storage instance from configuration.
        
        Args:
            config: Storage configuration with 'type' and backend-specific settings
            
        Returns:
            AnalyticsStorage instance
            
        Raises:
            ValueError: If storage type is unknown or configuration is invalid
        """
        storage_type = config.get('type', '').lower()
        
        if not storage_type:
            raise ValueError("Analytics storage configuration missing 'type' field")
        
        if storage_type not in cls.ANALYTICS_STORAGE_TYPES:
            available = ', '.join(cls.ANALYTICS_STORAGE_TYPES.keys())
            raise ValueError(
                f"Unknown analytics storage type: {storage_type}. "
                f"Available types: {available}"
            )
        
        storage_class = cls.ANALYTICS_STORAGE_TYPES[storage_type]
        
        try:
            storage = storage_class(config)
            logger.info(f"Created analytics storage: {storage_type}")
            return storage
            
        except Exception as e:
            logger.error(f"Failed to create analytics storage {storage_type}: {e}")
            raise
    
    
    
    @classmethod
    def create_analytics_from_registry(cls, backend_name: str, registry: Dict[str, Dict[str, Any]]) -> AnalyticsStorage:
        """
        Create an analytics storage instance from a registry name.
        
        Args:
            backend_name: Name of the backend in the registry
            registry: Analytics registry dictionary
            
        Returns:
            AnalyticsStorage instance
            
        Raises:
            ValueError: If backend not found in registry
        """
        if backend_name not in registry:
            available = ', '.join(registry.keys())
            raise ValueError(
                f"Analytics backend '{backend_name}' not found in registry. "
                f"Available backends: {available}"
            )
        
        backend_config = registry[backend_name]
        return cls.create_analytics_storage(backend_config)
    
    @classmethod
    def get_supported_types(cls) -> list:
        """
        Get list of supported analytics storage types.

        Returns:
            List of supported analytics storage types
        """
        return list(cls.ANALYTICS_STORAGE_TYPES.keys())