"""
Apache Solr analytics storage adapter for OLAP operations.
"""

import json
import logging
from datetime import datetime
from typing import Dict, Any, List, Optional
import hashlib

from .base import AnalyticsStorage

logger = logging.getLogger(__name__)

try:
    import pysolr
    SOLR_AVAILABLE = True
except ImportError:
    SOLR_AVAILABLE = False
    logger.warning("pysolr not available. Install with: pip install pysolr")


class SolrAnalyticsStorage(AnalyticsStorage):
    """Apache Solr implementation of analytics storage."""
    
    def __init__(self, config: Dict[str, Any]):
        """
        Initialize Solr analytics storage.
        
        Args:
            config: Configuration with url, collections, etc.
        """
        super().__init__(config)
        
        if not SOLR_AVAILABLE:
            raise ImportError("pysolr required for Solr analytics storage")
        
        # Connection settings
        self.base_url = config.get('url', 'http://localhost:8983/solr')
        self.timeout = config.get('timeout', 10)
        self.auth = None
        
        if config.get('username') and config.get('password'):
            self.auth = (config['username'], config['password'])
        
        # Collection names
        self.collections = {
            'documents': config.get('documents_collection', 'documents'),
            'elements': config.get('elements_collection', 'elements'),
            'embeddings': config.get('embeddings_collection', 'embeddings'),
            'relationships': config.get('relationships_collection', 'relationships'),
            'metrics': config.get('metrics_collection', 'metrics')
        }
        
        # Solr clients for each collection
        self.clients = {}
        
        self.initialize()
    
    def initialize(self) -> None:
        """Initialize Solr connections."""
        try:
            for data_type, collection_name in self.collections.items():
                url = f"{self.base_url}/{collection_name}"
                
                if self.auth:
                    self.clients[data_type] = pysolr.Solr(
                        url,
                        timeout=self.timeout,
                        auth=self.auth
                    )
                else:
                    self.clients[data_type] = pysolr.Solr(
                        url,
                        timeout=self.timeout
                    )
                
                # Test connection
                self.clients[data_type].ping()
            
            logger.info(f"Solr analytics storage initialized: {self.base_url}")
            
        except Exception as e:
            logger.error(f"Failed to initialize Solr analytics storage: {e}")
            raise
    
    def _prepare_document(self, doc: Dict[str, Any], run_id: str) -> Dict[str, Any]:
        """Prepare document for Solr indexing."""
        # Make a copy to avoid modifying original
        solr_doc = doc.copy()
        
        # Add metadata fields
        solr_doc['_run_id_s'] = run_id  # _s suffix for string field
        solr_doc['_written_at_dt'] = datetime.now().isoformat() + 'Z'  # _dt for datetime
        
        # Add partition fields for faceting
        now = datetime.now()
        if 'year' in self.partitioning:
            solr_doc['_year_i'] = now.year  # _i for integer
        if 'month' in self.partitioning:
            solr_doc['_month_i'] = now.month
        if 'day' in self.partitioning:
            solr_doc['_day_i'] = now.day
        if 'hour' in self.partitioning:
            solr_doc['_hour_i'] = now.hour
        
        # Handle nested objects by flattening or converting to JSON
        for key, value in list(solr_doc.items()):
            if isinstance(value, (dict, list)):
                # Convert complex types to JSON strings
                solr_doc[f"{key}_json"] = json.dumps(value)
                del solr_doc[key]
            elif isinstance(value, datetime):
                # Convert datetime to Solr format
                solr_doc[key] = value.isoformat() + 'Z'
        
        # Generate unique ID if not present
        if 'id' not in solr_doc:
            doc_str = json.dumps(solr_doc, sort_keys=True)
            solr_doc['id'] = hashlib.md5(doc_str.encode()).hexdigest()
        
        return solr_doc
    
    def append_documents(self, documents: List[Dict[str, Any]], 
                        run_id: str) -> int:
        """Append documents to Solr."""
        if not documents:
            return 0
        
        client = self.clients['documents']
        
        # Prepare documents for Solr
        solr_docs = [
            self._prepare_document(doc, run_id)
            for doc in documents
        ]
        
        try:
            # Add documents and commit
            client.add(solr_docs, commit=True)
            logger.info(f"Appended {len(solr_docs)} documents to Solr")
            return len(solr_docs)
            
        except Exception as e:
            logger.error(f"Error appending documents to Solr: {e}")
            raise
    
    def append_elements(self, elements: List[Dict[str, Any]], 
                       run_id: str) -> int:
        """Append elements to Solr."""
        if not elements:
            return 0
        
        client = self.clients['elements']
        
        # Prepare elements for Solr
        solr_docs = [
            self._prepare_document(elem, run_id)
            for elem in elements
        ]
        
        try:
            client.add(solr_docs, commit=True)
            logger.info(f"Appended {len(solr_docs)} elements to Solr")
            return len(solr_docs)
            
        except Exception as e:
            logger.error(f"Error appending elements to Solr: {e}")
            raise
    
    def append_embeddings(self, embeddings: List[Dict[str, Any]], 
                         run_id: str) -> int:
        """Append embeddings to Solr."""
        if not embeddings:
            return 0
        
        client = self.clients['embeddings']
        
        # Prepare embeddings for Solr
        solr_docs = []
        for emb in embeddings:
            solr_doc = self._prepare_document(emb, run_id)
            
            # Convert embedding vector to comma-separated string or multiple fields
            if 'embedding' in emb:
                vector = emb['embedding']
                if isinstance(vector, list):
                    # Store as comma-separated string for now
                    # In production, you'd use Solr's dense vector field type
                    solr_doc['embedding_vector'] = ','.join(map(str, vector))
                    del solr_doc['embedding']
            
            solr_docs.append(solr_doc)
        
        try:
            client.add(solr_docs, commit=True)
            logger.info(f"Appended {len(solr_docs)} embeddings to Solr")
            return len(solr_docs)
            
        except Exception as e:
            logger.error(f"Error appending embeddings to Solr: {e}")
            raise
    
    def append_relationships(self, relationships: List[Dict[str, Any]], 
                           run_id: str) -> int:
        """Append relationships to Solr."""
        if not relationships:
            return 0
        
        client = self.clients['relationships']
        
        # Prepare relationships for Solr
        solr_docs = [
            self._prepare_document(rel, run_id)
            for rel in relationships
        ]
        
        try:
            client.add(solr_docs, commit=True)
            logger.info(f"Appended {len(solr_docs)} relationships to Solr")
            return len(solr_docs)
            
        except Exception as e:
            logger.error(f"Error appending relationships to Solr: {e}")
            raise
    
    def append_metrics(self, metrics: Dict[str, Any], 
                      run_id: str) -> bool:
        """Append processing metrics."""
        client = self.clients['metrics']
        
        # Prepare metrics document
        solr_doc = self._prepare_document(metrics, run_id)
        
        try:
            client.add([solr_doc], commit=True)
            return True
            
        except Exception as e:
            logger.error(f"Error appending metrics to Solr: {e}")
            return False
    
    def get_partition_path(self, run_id: str, data_type: str) -> str:
        """Get logical partition path (collection name)."""
        return f"{self.collections[data_type]}/{run_id}"
    
    def list_runs(self, start_date: Optional[datetime] = None,
                 end_date: Optional[datetime] = None) -> List[str]:
        """List processing runs in date range."""
        client = self.clients['metrics']
        
        # Build query
        query = "*:*"
        fq = []
        
        if start_date or end_date:
            date_range = []
            if start_date:
                date_range.append(start_date.isoformat() + 'Z')
            else:
                date_range.append('*')
            
            date_range.append('TO')
            
            if end_date:
                date_range.append(end_date.isoformat() + 'Z')
            else:
                date_range.append('*')
            
            fq.append(f"_written_at_dt:[{' '.join(date_range)}]")
        
        try:
            # Use faceting to get unique run IDs
            results = client.search(
                query,
                fq=fq,
                facet='on',
                **{'facet.field': '_run_id_s', 'facet.limit': -1, 'rows': 0}
            )
            
            runs = []
            if 'facet_counts' in results.raw_response:
                facet_fields = results.raw_response['facet_counts']['facet_fields']
                if '_run_id_s' in facet_fields:
                    # Facet results come as [value, count, value, count, ...]
                    run_facets = facet_fields['_run_id_s']
                    runs = [run_facets[i] for i in range(0, len(run_facets), 2)]
            
            return sorted(runs)
            
        except Exception as e:
            logger.error(f"Error listing runs from Solr: {e}")
            return []
    
    def get_run_stats(self, run_id: str) -> Optional[Dict[str, Any]]:
        """Get statistics for a processing run."""
        try:
            # Get latest metrics
            client = self.clients['metrics']
            results = client.search(
                f"_run_id_s:{run_id}",
                sort="_written_at_dt desc",
                rows=1
            )
            
            if results:
                stats = dict(results.docs[0])
                
                # Add document counts from each collection
                for data_type, collection_client in self.clients.items():
                    if data_type != 'metrics':
                        count_results = collection_client.search(
                            f"_run_id_s:{run_id}",
                            rows=0
                        )
                        stats[f"{data_type}_count"] = count_results.hits
                
                return stats
            
        except Exception as e:
            logger.error(f"Error getting run stats from Solr: {e}")
        
        return None
    
    def close(self) -> None:
        """Close Solr connections."""
        # pysolr doesn't maintain persistent connections
        self.clients.clear()

    # TODO: Implement sampling methods for MCP database sampling
    def sample_elements(self, filters: Optional[Dict] = None, limit: int = 100,
                       stratify_by: Optional[str] = None,
                       random_seed: Optional[int] = None) -> List[Dict[str, Any]]:
        """
        Sample elements from Solr with flexible filtering.

        TODO: Implement Solr sampling
        - Use random field and sort for sampling
        - Implement stratified sampling with faceting
        - Apply filters with fq parameter
        """
        raise NotImplementedError("Solr sampling methods not yet implemented")

    def get_corpus_stats(self, filters: Optional[Dict] = None) -> Dict[str, Any]:
        """
        Get comprehensive statistics about the Solr corpus.

        TODO: Implement faceting and stats for corpus statistics
        - Use facet queries for counts
        - Use stats component for numeric fields
        - Use terms component for distributions
        """
        raise NotImplementedError("Solr corpus stats not yet implemented")

    def sample_documents(self, filters: Optional[Dict] = None, limit: int = 50,
                        random_seed: Optional[int] = None) -> List[Dict[str, Any]]:
        """
        Sample documents from Solr.

        TODO: Implement document sampling
        - Use random_* fields for sampling
        - Apply document type filters
        """
        raise NotImplementedError("Solr document sampling not yet implemented")

    def execute_custom_query(self, query: str,
                            params: Optional[List] = None) -> List[Dict[str, Any]]:
        """
        Execute custom Solr query.

        TODO: Implement safe execution of Solr queries
        - Parse query parameters
        - Validate safety (no updates/deletes)
        - Execute and return results
        """
        raise NotImplementedError("Solr custom query execution not yet implemented")