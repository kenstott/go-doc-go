"""
Backend health checking utilities for analytics registry.
"""

import logging
import os
import sqlite3
import tempfile
from pathlib import Path
from typing import Dict, Any, Optional

logger = logging.getLogger(__name__)


class BackendHealthChecker:
    """Check connectivity and health status of analytics backends."""
    
    @staticmethod
    def check_sqlite_backend(config: Dict[str, Any]) -> Dict[str, Any]:
        """
        Check SQLite backend connectivity.
        
        Args:
            config: Backend configuration
            
        Returns:
            Dict with status, latency, and details
        """
        import time
        
        try:
            start_time = time.time()
            db_path = config.get('path', ':memory:')
            
            if db_path == ':memory:':
                # In-memory database - always available
                conn = sqlite3.connect(db_path)
                conn.execute('SELECT 1').fetchone()
                conn.close()
                latency = (time.time() - start_time) * 1000
                
                return {
                    'status': 'active',
                    'latency_ms': round(latency, 2),
                    'message': 'In-memory database available',
                    'details': {'type': 'memory'}
                }
            else:
                # File-based database - check if file exists and is readable
                db_file = Path(db_path)
                
                if db_file.exists():
                    conn = sqlite3.connect(str(db_file))
                    conn.execute('SELECT 1').fetchone()
                    conn.close()
                    latency = (time.time() - start_time) * 1000
                    
                    file_size = db_file.stat().st_size
                    return {
                        'status': 'active',
                        'latency_ms': round(latency, 2),
                        'message': f'Database file accessible ({file_size} bytes)',
                        'details': {
                            'file_path': str(db_file),
                            'file_size_bytes': file_size,
                            'readable': True,
                            'writable': os.access(str(db_file), os.W_OK)
                        }
                    }
                else:
                    return {
                        'status': 'inactive',
                        'latency_ms': None,
                        'message': f'Database file not found: {db_path}',
                        'details': {'file_path': str(db_file), 'exists': False}
                    }
                    
        except Exception as e:
            return {
                'status': 'error',
                'latency_ms': None,
                'message': f'SQLite connection failed: {str(e)}',
                'details': {'error_type': type(e).__name__}
            }
    
    @staticmethod
    def check_parquet_backend(config: Dict[str, Any]) -> Dict[str, Any]:
        """
        Check Parquet backend (local filesystem or S3) connectivity.
        
        Args:
            config: Backend configuration
            
        Returns:
            Dict with status, latency, and details
        """
        import time
        
        try:
            start_time = time.time()
            base_path = config.get('base_path', './data-lake')
            
            if base_path.startswith('s3://'):
                # S3-based storage - check if credentials are available
                s3_config = config.get('s3', {})
                has_credentials = bool(
                    s3_config.get('access_key') and s3_config.get('secret_key') or
                    os.environ.get('AWS_ACCESS_KEY_ID') and os.environ.get('AWS_SECRET_ACCESS_KEY')
                )
                
                if has_credentials:
                    # Could add actual S3 connectivity check here
                    latency = (time.time() - start_time) * 1000
                    return {
                        'status': 'configured',  # Not fully active without actual S3 check
                        'latency_ms': round(latency, 2),
                        'message': f'S3 credentials configured for {base_path}',
                        'details': {
                            'storage_type': 's3',
                            'bucket': base_path.split('/')[2],
                            'has_credentials': True
                        }
                    }
                else:
                    return {
                        'status': 'inactive',
                        'latency_ms': None,
                        'message': 'S3 credentials not configured',
                        'details': {'storage_type': 's3', 'has_credentials': False}
                    }
            else:
                # Local filesystem storage
                data_path = Path(base_path)
                
                if data_path.exists():
                    # Check if we can read/write to the directory
                    latency = (time.time() - start_time) * 1000
                    
                    # Count parquet files
                    parquet_files = list(data_path.glob('**/*.parquet'))
                    
                    return {
                        'status': 'active',
                        'latency_ms': round(latency, 2),
                        'message': f'Local data lake accessible with {len(parquet_files)} files',
                        'details': {
                            'storage_type': 'local',
                            'directory_path': str(data_path),
                            'parquet_files': len(parquet_files),
                            'readable': os.access(str(data_path), os.R_OK),
                            'writable': os.access(str(data_path), os.W_OK)
                        }
                    }
                else:
                    return {
                        'status': 'inactive',
                        'latency_ms': None,
                        'message': f'Data lake directory not found: {base_path}',
                        'details': {'storage_type': 'local', 'directory_path': str(data_path)}
                    }
                    
        except Exception as e:
            return {
                'status': 'error',
                'latency_ms': None,
                'message': f'Parquet backend check failed: {str(e)}',
                'details': {'error_type': type(e).__name__}
            }
    
    @staticmethod
    def check_mongodb_backend(config: Dict[str, Any]) -> Dict[str, Any]:
        """
        Check MongoDB backend connectivity.
        
        Args:
            config: Backend configuration
            
        Returns:
            Dict with status, latency, and details
        """
        try:
            import pymongo
            import time
            
            start_time = time.time()
            uri = config.get('uri', 'mongodb://localhost:27017/')
            database = config.get('database', 'go_doc_go_analytics')
            
            # Try to connect with a short timeout
            client = pymongo.MongoClient(uri, serverSelectionTimeoutMS=2000)
            client.server_info()  # Force connection
            
            latency = (time.time() - start_time) * 1000
            
            # Get database stats
            db = client[database]
            stats = db.command('dbstats')
            
            return {
                'status': 'active',
                'latency_ms': round(latency, 2),
                'message': f'MongoDB connected to database: {database}',
                'details': {
                    'database': database,
                    'collections': stats.get('collections', 0),
                    'data_size_bytes': stats.get('dataSize', 0)
                }
            }
            
        except ImportError:
            return {
                'status': 'unavailable',
                'latency_ms': None,
                'message': 'pymongo package not installed',
                'details': {'missing_dependency': 'pymongo'}
            }
        except Exception as e:
            return {
                'status': 'inactive',
                'latency_ms': None,
                'message': f'MongoDB connection failed: {str(e)}',
                'details': {'error_type': type(e).__name__}
            }
    
    @staticmethod
    def check_elasticsearch_backend(config: Dict[str, Any]) -> Dict[str, Any]:
        """
        Check Elasticsearch backend connectivity.
        
        Args:
            config: Backend configuration
            
        Returns:
            Dict with status, latency, and details
        """
        try:
            import elasticsearch
            import time
            
            start_time = time.time()
            hosts = config.get('hosts', ['localhost:9200'])
            username = config.get('username')
            password = config.get('password')
            
            # Create client
            es_config = {'hosts': hosts, 'request_timeout': 2}
            if username and password:
                es_config['http_auth'] = (username, password)
            
            es = elasticsearch.Elasticsearch(**es_config)
            
            # Test connectivity
            info = es.info()
            latency = (time.time() - start_time) * 1000
            
            return {
                'status': 'active',
                'latency_ms': round(latency, 2),
                'message': f'Elasticsearch cluster: {info.get("cluster_name", "unknown")}',
                'details': {
                    'cluster_name': info.get('cluster_name'),
                    'version': info.get('version', {}).get('number'),
                    'nodes': len(hosts)
                }
            }
            
        except ImportError:
            return {
                'status': 'unavailable',
                'latency_ms': None,
                'message': 'elasticsearch package not installed',
                'details': {'missing_dependency': 'elasticsearch'}
            }
        except Exception as e:
            return {
                'status': 'inactive',
                'latency_ms': None,
                'message': f'Elasticsearch connection failed: {str(e)}',
                'details': {'error_type': type(e).__name__}
            }
    
    @staticmethod
    def check_neo4j_backend(config: Dict[str, Any]) -> Dict[str, Any]:
        """
        Check Neo4j backend connectivity.
        
        Args:
            config: Backend configuration
            
        Returns:
            Dict with status, latency, and details
        """
        try:
            from neo4j import GraphDatabase
            import time
            
            start_time = time.time()
            uri = config.get('uri', 'bolt://localhost:7687')
            username = config.get('username', 'neo4j')
            password = config.get('password')
            
            if not password:
                return {
                    'status': 'inactive',
                    'latency_ms': None,
                    'message': 'Neo4j password not configured',
                    'details': {'missing_config': 'password'}
                }
            
            # Try to connect
            driver = GraphDatabase.driver(uri, auth=(username, password))
            with driver.session() as session:
                result = session.run("RETURN 1 AS test")
                record = result.single()
                
            latency = (time.time() - start_time) * 1000
            driver.close()
            
            return {
                'status': 'active',
                'latency_ms': round(latency, 2),
                'message': 'Neo4j database connected successfully',
                'details': {
                    'uri': uri,
                    'username': username,
                    'connection_tested': True
                }
            }
            
        except ImportError:
            return {
                'status': 'unavailable',
                'latency_ms': None,
                'message': 'neo4j package not installed',
                'details': {'missing_dependency': 'neo4j'}
            }
        except Exception as e:
            return {
                'status': 'inactive',
                'latency_ms': None,
                'message': f'Neo4j connection failed: {str(e)}',
                'details': {'error_type': type(e).__name__}
            }
    
    @classmethod
    def check_backend_health(cls, backend_name: str, config: Dict[str, Any]) -> Dict[str, Any]:
        """
        Check the health of any analytics backend.
        
        Args:
            backend_name: Name of the backend
            config: Backend configuration
            
        Returns:
            Dict with comprehensive health status
        """
        backend_type = config.get('type', '').lower()
        
        # Map backend types to checker methods
        checkers = {
            'sqlite': cls.check_sqlite_backend,
            'parquet': cls.check_parquet_backend,
            'mongodb': cls.check_mongodb_backend,
            'mongo': cls.check_mongodb_backend,
            'elasticsearch': cls.check_elasticsearch_backend,
            'elastic': cls.check_elasticsearch_backend,
            'es': cls.check_elasticsearch_backend,
            'neo4j': cls.check_neo4j_backend,
        }
        
        checker = checkers.get(backend_type)
        
        if checker:
            try:
                health_status = checker(config)
                health_status['backend_name'] = backend_name
                health_status['backend_type'] = backend_type
                health_status['checked_at'] = time.time()
                return health_status
                
            except Exception as e:
                logger.error(f"Health check failed for {backend_name}: {e}")
                return {
                    'backend_name': backend_name,
                    'backend_type': backend_type,
                    'status': 'error',
                    'latency_ms': None,
                    'message': f'Health check error: {str(e)}',
                    'details': {'error_type': type(e).__name__},
                    'checked_at': time.time()
                }
        else:
            return {
                'backend_name': backend_name,
                'backend_type': backend_type,
                'status': 'unknown',
                'latency_ms': None,
                'message': f'No health checker available for type: {backend_type}',
                'details': {'supported_types': list(checkers.keys())},
                'checked_at': time.time()
            }