"""
Elasticsearch analytics storage adapter for OLAP operations.
"""

import json
import logging
from datetime import datetime
from typing import Dict, Any, List, Optional
import hashlib

from .base import AnalyticsStorage

logger = logging.getLogger(__name__)

try:
    from elasticsearch import Elasticsearch, helpers
    from elasticsearch.exceptions import BulkIndexError
    ELASTICSEARCH_AVAILABLE = True
except ImportError:
    ELASTICSEARCH_AVAILABLE = False
    logger.warning("Elasticsearch not available. Install with: pip install elasticsearch")


class ElasticsearchAnalyticsStorage(AnalyticsStorage):
    """Elasticsearch implementation of analytics storage."""
    
    def __init__(self, config: Dict[str, Any]):
        """
        Initialize Elasticsearch analytics storage.
        
        Args:
            config: Configuration with hosts, index_prefix, etc.
        """
        super().__init__(config)
        
        if not ELASTICSEARCH_AVAILABLE:
            raise ImportError("elasticsearch required for Elasticsearch analytics storage")
        
        # Connection settings
        self.hosts = config.get('hosts', ['localhost:9200'])
        self.index_prefix = config.get('index_prefix', 'godocgo')
        self.username = config.get('username')
        self.password = config.get('password')
        self.api_key = config.get('api_key')
        self.cloud_id = config.get('cloud_id')
        
        # Index settings
        self.shards = config.get('shards', 2)
        self.replicas = config.get('replicas', 1)
        self.refresh_interval = config.get('refresh_interval', '1s')
        
        self.client = None
        self.initialize()
    
    def initialize(self) -> None:
        """Initialize Elasticsearch connection and indices."""
        try:
            # Build connection parameters
            es_params = {}
            
            if self.cloud_id:
                es_params['cloud_id'] = self.cloud_id
            else:
                es_params['hosts'] = self.hosts
            
            if self.api_key:
                es_params['api_key'] = self.api_key
            elif self.username and self.password:
                es_params['basic_auth'] = (self.username, self.password)
            
            self.client = Elasticsearch(**es_params)
            
            # Test connection
            info = self.client.info()
            logger.info(f"Connected to Elasticsearch {info['version']['number']}")
            
            # Create index templates
            self._create_index_templates()
            
        except Exception as e:
            logger.error(f"Failed to initialize Elasticsearch analytics storage: {e}")
            raise
    
    def _create_index_templates(self):
        """Create index templates for different data types."""
        base_settings = {
            "number_of_shards": self.shards,
            "number_of_replicas": self.replicas,
            "refresh_interval": self.refresh_interval
        }
        
        # Document index template
        self.client.indices.put_index_template(
            name=f"{self.index_prefix}-documents",
            body={
                "index_patterns": [f"{self.index_prefix}-documents-*"],
                "template": {
                    "settings": base_settings,
                    "mappings": {
                        "properties": {
                            "doc_id": {"type": "keyword"},
                            "doc_type": {"type": "keyword"},
                            "source": {"type": "keyword"},
                            "content": {"type": "text"},
                            "metadata": {"type": "object"},
                            "_run_id": {"type": "keyword"},
                            "_written_at": {"type": "date"}
                        }
                    }
                }
            }
        )
        
        # Elements index template
        self.client.indices.put_index_template(
            name=f"{self.index_prefix}-elements",
            body={
                "index_patterns": [f"{self.index_prefix}-elements-*"],
                "template": {
                    "settings": base_settings,
                    "mappings": {
                        "properties": {
                            "element_id": {"type": "keyword"},
                            "element_type": {"type": "keyword"},
                            "doc_id": {"type": "keyword"},
                            "content_preview": {"type": "text"},
                            "full_content": {"type": "text"},
                            "metadata": {"type": "object"},
                            "_run_id": {"type": "keyword"},
                            "_written_at": {"type": "date"}
                        }
                    }
                }
            }
        )
        
        # Embeddings index template with dense vector
        self.client.indices.put_index_template(
            name=f"{self.index_prefix}-embeddings",
            body={
                "index_patterns": [f"{self.index_prefix}-embeddings-*"],
                "template": {
                    "settings": base_settings,
                    "mappings": {
                        "properties": {
                            "element_pk": {"type": "keyword"},
                            "element_id": {"type": "keyword"},
                            "embedding": {
                                "type": "dense_vector",
                                "dims": 384,  # Default, should be configurable
                                "index": True,
                                "similarity": "cosine"
                            },
                            "model": {"type": "keyword"},
                            "_run_id": {"type": "keyword"},
                            "_written_at": {"type": "date"}
                        }
                    }
                }
            }
        )
        
        # Relationships index template
        self.client.indices.put_index_template(
            name=f"{self.index_prefix}-relationships",
            body={
                "index_patterns": [f"{self.index_prefix}-relationships-*"],
                "template": {
                    "settings": base_settings,
                    "mappings": {
                        "properties": {
                            "source_id": {"type": "keyword"},
                            "target_id": {"type": "keyword"},
                            "relationship_type": {"type": "keyword"},
                            "metadata": {"type": "object"},
                            "_run_id": {"type": "keyword"},
                            "_written_at": {"type": "date"}
                        }
                    }
                }
            }
        )
    
    def _get_index_name(self, data_type: str, run_id: str) -> str:
        """Get index name for data type and run."""
        now = datetime.now()
        date_suffix = now.strftime("%Y%m%d")
        return f"{self.index_prefix}-{data_type}-{date_suffix}"
    
    def _bulk_index(self, index: str, documents: List[Dict[str, Any]]) -> int:
        """Bulk index documents to Elasticsearch."""
        if not documents:
            return 0
        
        # Prepare bulk actions
        actions = []
        for doc in documents:
            # Generate document ID from content hash
            doc_str = json.dumps(doc, sort_keys=True)
            doc_id = hashlib.md5(doc_str.encode()).hexdigest()
            
            actions.append({
                "_index": index,
                "_id": doc_id,
                "_source": doc
            })
        
        try:
            success, failed = helpers.bulk(
                self.client,
                actions,
                raise_on_error=False,
                raise_on_exception=False
            )
            
            if failed:
                logger.warning(f"Failed to index {len(failed)} documents")
            
            logger.info(f"Indexed {success} documents to {index}")
            return success
            
        except BulkIndexError as e:
            # Extract successful count from error
            success = len(documents) - len(e.errors)
            logger.warning(f"Partial bulk index: {success} succeeded, {len(e.errors)} failed")
            return success
        except Exception as e:
            logger.error(f"Error bulk indexing to Elasticsearch: {e}")
            raise
    
    def append_documents(self, documents: List[Dict[str, Any]], 
                        run_id: str) -> int:
        """Append documents to Elasticsearch."""
        if not documents:
            return 0
        
        index = self._get_index_name('documents', run_id)
        
        # Add metadata
        for doc in documents:
            doc['_run_id'] = run_id
            doc['_written_at'] = datetime.now().isoformat()
        
        return self._bulk_index(index, documents)
    
    def append_elements(self, elements: List[Dict[str, Any]], 
                       run_id: str) -> int:
        """Append elements to Elasticsearch."""
        if not elements:
            return 0
        
        index = self._get_index_name('elements', run_id)
        
        # Add metadata
        for elem in elements:
            elem['_run_id'] = run_id
            elem['_written_at'] = datetime.now().isoformat()
        
        return self._bulk_index(index, elements)
    
    def append_embeddings(self, embeddings: List[Dict[str, Any]], 
                         run_id: str) -> int:
        """Append embeddings to Elasticsearch."""
        if not embeddings:
            return 0
        
        index = self._get_index_name('embeddings', run_id)
        
        # Add metadata and ensure embedding is a list
        for emb in embeddings:
            emb['_run_id'] = run_id
            emb['_written_at'] = datetime.now().isoformat()
            
            # Ensure embedding is a list (not numpy array)
            if 'embedding' in emb and not isinstance(emb['embedding'], list):
                emb['embedding'] = list(emb['embedding'])
        
        return self._bulk_index(index, embeddings)
    
    def append_relationships(self, relationships: List[Dict[str, Any]], 
                           run_id: str) -> int:
        """Append relationships to Elasticsearch."""
        if not relationships:
            return 0
        
        index = self._get_index_name('relationships', run_id)
        
        # Add metadata
        for rel in relationships:
            rel['_run_id'] = run_id
            rel['_written_at'] = datetime.now().isoformat()
        
        return self._bulk_index(index, relationships)
    
    def append_metrics(self, metrics: Dict[str, Any], 
                      run_id: str) -> bool:
        """Append processing metrics."""
        index = self._get_index_name('metrics', run_id)
        
        # Add metadata
        metrics['_run_id'] = run_id
        metrics['_written_at'] = datetime.now().isoformat()
        
        try:
            result = self.client.index(
                index=index,
                document=metrics
            )
            return result['result'] in ['created', 'updated']
            
        except Exception as e:
            logger.error(f"Error appending metrics to Elasticsearch: {e}")
            return False
    
    def get_partition_path(self, run_id: str, data_type: str) -> str:
        """Get logical partition path (index name)."""
        return self._get_index_name(data_type, run_id)
    
    def list_runs(self, start_date: Optional[datetime] = None,
                 end_date: Optional[datetime] = None) -> List[str]:
        """List processing runs in date range."""
        # Search for unique run IDs in metrics indices
        query = {"match_all": {}}
        
        if start_date or end_date:
            range_filter = {}
            if start_date:
                range_filter['gte'] = start_date.isoformat()
            if end_date:
                range_filter['lte'] = end_date.isoformat()
            
            query = {
                "range": {
                    "_written_at": range_filter
                }
            }
        
        try:
            response = self.client.search(
                index=f"{self.index_prefix}-metrics-*",
                body={
                    "query": query,
                    "aggs": {
                        "unique_runs": {
                            "terms": {
                                "field": "_run_id",
                                "size": 10000
                            }
                        }
                    },
                    "size": 0
                }
            )
            
            runs = []
            if 'aggregations' in response:
                buckets = response['aggregations']['unique_runs']['buckets']
                runs = [bucket['key'] for bucket in buckets]
            
            return sorted(runs)
            
        except Exception as e:
            logger.error(f"Error listing runs from Elasticsearch: {e}")
            return []
    
    def get_run_stats(self, run_id: str) -> Optional[Dict[str, Any]]:
        """Get statistics for a processing run."""
        try:
            # Get latest metrics
            response = self.client.search(
                index=f"{self.index_prefix}-metrics-*",
                body={
                    "query": {
                        "term": {"_run_id": run_id}
                    },
                    "sort": [{"_written_at": "desc"}],
                    "size": 1
                }
            )
            
            if response['hits']['total']['value'] > 0:
                stats = response['hits']['hits'][0]['_source']
                
                # Add document counts
                for data_type in ['documents', 'elements', 'relationships', 'embeddings']:
                    count_response = self.client.count(
                        index=f"{self.index_prefix}-{data_type}-*",
                        body={
                            "query": {"term": {"_run_id": run_id}}
                        }
                    )
                    stats[f"{data_type}_count"] = count_response['count']
                
                return stats
            
        except Exception as e:
            logger.error(f"Error getting run stats from Elasticsearch: {e}")
        
        return None
    
    def close(self) -> None:
        """Close Elasticsearch connection."""
        if self.client:
            self.client.close()
            self.client = None