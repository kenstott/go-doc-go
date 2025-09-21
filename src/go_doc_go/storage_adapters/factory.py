"""
Storage adapter factory for creating job and analytics storage instances.
"""

import logging
from typing import Dict, Any, Optional

from .base import JobStorage, AnalyticsStorage

# Job storage implementations
from .job_postgresql import PostgreSQLJobStorage
from .job_redis import RedisJobStorage
# SQLite uses the queue adapter directly - no JobStorage wrapper needed
from ..queue.sqlite_adapter import SQLiteQueueAdapter, create_sqlite_queue_adapter

# Analytics storage implementations
from .analytics_parquet import ParquetAnalyticsStorage
from .analytics_mongodb import MongoDBAnalyticsStorage
from .analytics_elasticsearch import ElasticsearchAnalyticsStorage
from .analytics_solr import SolrAnalyticsStorage
from .analytics_neo4j import Neo4jAnalyticsStorage
from .analytics_sqlalchemy import SQLAlchemyAnalyticsStorage

logger = logging.getLogger(__name__)


class StorageFactory:
    """Factory for creating storage adapter instances."""
    
    # Registry of job storage implementations
    # Note: SQLite returns SQLiteQueueAdapter directly (no JobStorage wrapper)
    JOB_STORAGE_TYPES = {
        'postgresql': PostgreSQLJobStorage,
        'postgres': PostgreSQLJobStorage,
        'redis': RedisJobStorage,
        'sqlite': 'sqlite_queue_adapter'  # Special case - uses queue adapter directly
    }
    
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
    def create_job_storage(cls, config: Dict[str, Any]) -> JobStorage:
        """
        Create a job storage instance from configuration.
        
        Args:
            config: Storage configuration with 'type' and backend-specific settings
            
        Returns:
            JobStorage instance (or SQLiteQueueAdapter for SQLite)
            
        Raises:
            ValueError: If storage type is unknown or configuration is invalid
        """
        storage_type = config.get('type', '').lower()
        
        if not storage_type:
            raise ValueError("Job storage configuration missing 'type' field")
        
        if storage_type not in cls.JOB_STORAGE_TYPES:
            available = ', '.join(cls.JOB_STORAGE_TYPES.keys())
            raise ValueError(
                f"Unknown job storage type: {storage_type}. "
                f"Available types: {available}"
            )
        
        storage_class = cls.JOB_STORAGE_TYPES[storage_type]
        
        # Special handling for SQLite - use queue adapter directly
        if storage_type == 'sqlite':
            try:
                # SQLite uses the queue adapter directly (no JobStorage wrapper needed)
                storage = create_sqlite_queue_adapter(config)
                logger.info(f"Created SQLite queue adapter (direct usage, no JobStorage wrapper)")
                return storage
            except Exception as e:
                logger.error(f"Failed to create SQLite queue adapter: {e}")
                raise
        
        # Regular JobStorage implementations
        try:
            storage = storage_class(config)
            logger.info(f"Created job storage: {storage_type}")
            return storage
            
        except Exception as e:
            logger.error(f"Failed to create job storage {storage_type}: {e}")
            raise
    
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
    def create_dual_storage(cls, job_config: Dict[str, Any], 
                          analytics_config: Dict[str, Any]) -> tuple[JobStorage, AnalyticsStorage]:
        """
        Create both job and analytics storage instances.
        
        Args:
            job_config: Job storage configuration
            analytics_config: Analytics storage configuration
            
        Returns:
            Tuple of (JobStorage, AnalyticsStorage)
        """
        job_storage = cls.create_job_storage(job_config)
        analytics_storage = cls.create_analytics_storage(analytics_config)
        
        return job_storage, analytics_storage
    
    @classmethod
    def create_from_pipeline_config(cls, config: Dict[str, Any], registry: Optional[Dict[str, Dict[str, Any]]] = None) -> tuple[JobStorage, AnalyticsStorage]:
        """
        Create storage instances from pipeline configuration.
        
        Expected configuration structure:
        {
            "storage": {
                "job": {
                    "type": "postgresql",
                    "host": "localhost",
                    ...
                },
                "analytics": {
                    "type": "parquet",
                    "base_path": "s3://analytics-bucket/",
                    ...
                }
                # OR using registry name:
                "analytics": "parquet_lake"  # Name from analytics registry
            }
        }
        
        Args:
            config: Pipeline configuration dictionary
            registry: Optional analytics registry for name lookups
            
        Returns:
            Tuple of (JobStorage, AnalyticsStorage)
        """
        storage_config = config.get('storage', {})
        
        # Get job storage config (required)
        job_config = storage_config.get('job')
        if not job_config:
            raise ValueError("Pipeline configuration missing 'storage.job' section")
        
        # Get analytics storage config (required)
        analytics_config = storage_config.get('analytics')
        if not analytics_config:
            raise ValueError("Pipeline configuration missing 'storage.analytics' section")
        
        # If analytics is a string, look it up in the registry
        if isinstance(analytics_config, str):
            if not registry:
                raise ValueError(f"Analytics backend '{analytics_config}' specified but no registry provided")
            
            backend_name = analytics_config
            analytics_config = registry.get(backend_name)
            if not analytics_config:
                available = ', '.join(registry.keys()) if registry else 'none'
                raise ValueError(
                    f"Analytics backend '{backend_name}' not found in registry. "
                    f"Available backends: {available}"
                )
            
            logger.info(f"Using analytics backend '{backend_name}' from registry")
        
        return cls.create_dual_storage(job_config, analytics_config)
    
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
    def get_supported_types(cls) -> Dict[str, list]:
        """
        Get lists of supported storage types.
        
        Returns:
            Dictionary with 'job' and 'analytics' lists
        """
        return {
            'job': list(cls.JOB_STORAGE_TYPES.keys()),
            'analytics': list(cls.ANALYTICS_STORAGE_TYPES.keys())
        }