"""
MongoDB analytics storage adapter for OLAP operations.
"""

import json
import logging
from datetime import datetime
from typing import Dict, Any, List, Optional

from .base import AnalyticsStorage

logger = logging.getLogger(__name__)

try:
    from pymongo import MongoClient, ASCENDING
    from pymongo.errors import BulkWriteError
    MONGODB_AVAILABLE = True
except ImportError:
    MONGODB_AVAILABLE = False
    logger.warning("MongoDB not available. Install with: pip install pymongo")


class MongoDBAnalyticsStorage(AnalyticsStorage):
    """MongoDB implementation of analytics storage."""
    
    def __init__(self, config: Dict[str, Any]):
        """
        Initialize MongoDB analytics storage.
        
        Args:
            config: Configuration with uri, database, etc.
        """
        super().__init__(config)
        
        if not MONGODB_AVAILABLE:
            raise ImportError("pymongo required for MongoDB analytics storage")
        
        self.uri = config.get('uri', 'mongodb://localhost:27017/')
        self.database_name = config.get('database', 'analytics')
        self.client = None
        self.db = None
        
        # Collection names
        self.collections = {
            'documents': config.get('documents_collection', 'documents'),
            'elements': config.get('elements_collection', 'elements'),
            'embeddings': config.get('embeddings_collection', 'embeddings'),
            'relationships': config.get('relationships_collection', 'relationships'),
            'metrics': config.get('metrics_collection', 'run_metrics')
        }
        
        self.initialize()
    
    def initialize(self) -> None:
        """Initialize MongoDB connection and indexes."""
        try:
            self.client = MongoClient(self.uri)
            self.db = self.client[self.database_name]
            
            # Create indexes for efficient queries
            for collection_name in self.collections.values():
                collection = self.db[collection_name]
                
                # Index on run_id and timestamp
                collection.create_index([
                    ('_run_id', ASCENDING),
                    ('_written_at', ASCENDING)
                ])
                
                # Compound index for partitioning simulation
                if self.partitioning:
                    index_fields = []
                    for field in self.partitioning:
                        if field == 'run_id':
                            index_fields.append(('_run_id', ASCENDING))
                        elif field in ['year', 'month', 'day']:
                            index_fields.append((f'_{field}', ASCENDING))
                    
                    if index_fields:
                        collection.create_index(index_fields)
            
            # Test connection
            self.client.server_info()
            logger.info(f"MongoDB analytics storage initialized: {self.database_name}")
            
        except Exception as e:
            logger.error(f"Failed to initialize MongoDB analytics storage: {e}")
            raise
    
    def _add_partition_fields(self, doc: Dict[str, Any], run_id: str) -> Dict[str, Any]:
        """Add partition fields to document for query optimization."""
        now = datetime.now()
        doc['_run_id'] = run_id
        doc['_written_at'] = now
        
        # Add partition fields
        if 'year' in self.partitioning:
            doc['_year'] = now.year
        if 'month' in self.partitioning:
            doc['_month'] = now.month
        if 'day' in self.partitioning:
            doc['_day'] = now.day
        if 'hour' in self.partitioning:
            doc['_hour'] = now.hour
        
        return doc
    
    def append_documents(self, documents: List[Dict[str, Any]], 
                        run_id: str) -> int:
        """Append documents to MongoDB."""
        if not documents:
            return 0
        
        collection = self.db[self.collections['documents']]
        
        # Add partition fields
        docs_to_insert = [
            self._add_partition_fields(doc.copy(), run_id)
            for doc in documents
        ]
        
        try:
            result = collection.insert_many(docs_to_insert, ordered=False)
            inserted = len(result.inserted_ids)
            logger.info(f"Appended {inserted} documents to MongoDB")
            return inserted
            
        except BulkWriteError as e:
            # Some documents may have been inserted
            inserted = e.details.get('nInserted', 0)
            logger.warning(f"Partial insert: {inserted} documents inserted, errors: {e.details.get('writeErrors', [])}")
            return inserted
        except Exception as e:
            logger.error(f"Error appending documents to MongoDB: {e}")
            raise
    
    def append_elements(self, elements: List[Dict[str, Any]], 
                       run_id: str) -> int:
        """Append elements to MongoDB."""
        if not elements:
            return 0
        
        collection = self.db[self.collections['elements']]
        
        # Add partition fields
        elements_to_insert = [
            self._add_partition_fields(elem.copy(), run_id)
            for elem in elements
        ]
        
        try:
            result = collection.insert_many(elements_to_insert, ordered=False)
            inserted = len(result.inserted_ids)
            logger.info(f"Appended {inserted} elements to MongoDB")
            return inserted
            
        except BulkWriteError as e:
            inserted = e.details.get('nInserted', 0)
            logger.warning(f"Partial insert: {inserted} elements inserted")
            return inserted
        except Exception as e:
            logger.error(f"Error appending elements to MongoDB: {e}")
            raise
    
    def append_embeddings(self, embeddings: List[Dict[str, Any]], 
                         run_id: str) -> int:
        """Append embeddings to MongoDB."""
        if not embeddings:
            return 0
        
        collection = self.db[self.collections['embeddings']]
        
        # Add partition fields
        embeddings_to_insert = [
            self._add_partition_fields(emb.copy(), run_id)
            for emb in embeddings
        ]
        
        try:
            result = collection.insert_many(embeddings_to_insert, ordered=False)
            inserted = len(result.inserted_ids)
            logger.info(f"Appended {inserted} embeddings to MongoDB")
            return inserted
            
        except BulkWriteError as e:
            inserted = e.details.get('nInserted', 0)
            logger.warning(f"Partial insert: {inserted} embeddings inserted")
            return inserted
        except Exception as e:
            logger.error(f"Error appending embeddings to MongoDB: {e}")
            raise
    
    def append_relationships(self, relationships: List[Dict[str, Any]], 
                           run_id: str) -> int:
        """Append relationships to MongoDB."""
        if not relationships:
            return 0
        
        collection = self.db[self.collections['relationships']]
        
        # Add partition fields
        rels_to_insert = [
            self._add_partition_fields(rel.copy(), run_id)
            for rel in relationships
        ]
        
        try:
            result = collection.insert_many(rels_to_insert, ordered=False)
            inserted = len(result.inserted_ids)
            logger.info(f"Appended {inserted} relationships to MongoDB")
            return inserted
            
        except BulkWriteError as e:
            inserted = e.details.get('nInserted', 0)
            logger.warning(f"Partial insert: {inserted} relationships inserted")
            return inserted
        except Exception as e:
            logger.error(f"Error appending relationships to MongoDB: {e}")
            raise
    
    def append_metrics(self, metrics: Dict[str, Any], 
                      run_id: str) -> bool:
        """Append processing metrics."""
        collection = self.db[self.collections['metrics']]
        
        # Add partition fields
        metrics_doc = self._add_partition_fields(metrics.copy(), run_id)
        
        try:
            result = collection.insert_one(metrics_doc)
            return result.acknowledged
            
        except Exception as e:
            logger.error(f"Error appending metrics to MongoDB: {e}")
            return False
    
    def get_partition_path(self, run_id: str, data_type: str) -> str:
        """Get logical partition path (for compatibility)."""
        # MongoDB doesn't use file paths, return logical path
        now = datetime.now()
        parts = []
        
        for field in self.partitioning:
            if field == 'year':
                parts.append(f"year={now.year}")
            elif field == 'month':
                parts.append(f"month={now.month:02d}")
            elif field == 'day':
                parts.append(f"day={now.day:02d}")
            elif field == 'run_id':
                parts.append(f"run_id={run_id}")
        
        return f"{self.database_name}.{self.collections.get(data_type, data_type)}/{'/'.join(parts)}"
    
    def list_runs(self, start_date: Optional[datetime] = None,
                 end_date: Optional[datetime] = None) -> List[str]:
        """List processing runs in date range."""
        collection = self.db[self.collections['metrics']]
        
        # Build query
        query = {}
        if start_date or end_date:
            date_filter = {}
            if start_date:
                date_filter['$gte'] = start_date
            if end_date:
                date_filter['$lte'] = end_date
            query['_written_at'] = date_filter
        
        # Get distinct run IDs
        runs = collection.distinct('_run_id', query)
        return sorted(runs)
    
    def get_run_stats(self, run_id: str) -> Optional[Dict[str, Any]]:
        """Get statistics for a processing run."""
        metrics_collection = self.db[self.collections['metrics']]
        
        # Get latest metrics for run
        metrics = metrics_collection.find_one(
            {'_run_id': run_id},
            sort=[('_written_at', -1)]
        )
        
        if metrics:
            # Remove MongoDB's _id field
            metrics.pop('_id', None)
            
            # Add document counts
            stats = dict(metrics)
            stats['document_count'] = self.db[self.collections['documents']].count_documents({'_run_id': run_id})
            stats['element_count'] = self.db[self.collections['elements']].count_documents({'_run_id': run_id})
            stats['relationship_count'] = self.db[self.collections['relationships']].count_documents({'_run_id': run_id})
            stats['embedding_count'] = self.db[self.collections['embeddings']].count_documents({'_run_id': run_id})
            
            return stats
        
        return None
    
    def close(self) -> None:
        """Close MongoDB connection."""
        if self.client:
            self.client.close()
            self.client = None
            self.db = None