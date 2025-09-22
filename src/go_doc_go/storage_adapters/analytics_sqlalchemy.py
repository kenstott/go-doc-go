"""
SQLAlchemy analytics storage adapter for OLAP operations.
Supports any SQLAlchemy-compatible database (PostgreSQL, MySQL, SQLite, etc.).
"""

import json
import logging
from datetime import datetime
from typing import Dict, Any, List, Optional

from .base import AnalyticsStorage

logger = logging.getLogger(__name__)

try:
    from sqlalchemy import create_engine, Column, String, Integer, DateTime, Text, JSON, Float, Index, MetaData, Table
    from sqlalchemy.ext.declarative import declarative_base
    from sqlalchemy.orm import sessionmaker
    from sqlalchemy.dialects.postgresql import ARRAY
    SQLALCHEMY_AVAILABLE = True
except ImportError:
    SQLALCHEMY_AVAILABLE = False
    logger.warning("SQLAlchemy not available. Install with: pip install sqlalchemy")


class SQLAlchemyAnalyticsStorage(AnalyticsStorage):
    """SQLAlchemy implementation of analytics storage for any SQL database."""
    
    def __init__(self, config: Dict[str, Any]):
        """
        Initialize SQLAlchemy analytics storage.
        
        Args:
            config: Configuration with uri or connection parameters
        """
        super().__init__(config)
        
        if not SQLALCHEMY_AVAILABLE:
            raise ImportError("sqlalchemy required for SQLAlchemy analytics storage")
        
        # Connection settings
        self.uri = config.get('uri')
        if not self.uri:
            # Build URI from components
            dialect = config.get('dialect', 'postgresql')
            driver = config.get('driver', '')
            if driver:
                dialect = f"{dialect}+{driver}"
            
            username = config.get('username', '')
            password = config.get('password', '')
            host = config.get('host', 'localhost')
            port = config.get('port', '')
            database = config.get('database', 'analytics')
            
            if username and password:
                self.uri = f"{dialect}://{username}:{password}@{host}"
            elif username:
                self.uri = f"{dialect}://{username}@{host}"
            else:
                self.uri = f"{dialect}://{host}"
            
            if port:
                self.uri += f":{port}"
            self.uri += f"/{database}"
        
        # Table prefix
        self.table_prefix = config.get('table_prefix', 'analytics')
        
        # Engine settings
        self.pool_size = config.get('pool_size', 10)
        self.max_overflow = config.get('max_overflow', 20)
        
        self.engine = None
        self.metadata = None
        self.tables = {}
        self.Session = None
        
        self.initialize()
    
    def initialize(self) -> None:
        """Initialize SQLAlchemy connection and tables."""
        try:
            # Create engine
            self.engine = create_engine(
                self.uri,
                pool_size=self.pool_size,
                max_overflow=self.max_overflow,
                echo=False
            )
            
            # Create metadata
            self.metadata = MetaData()
            
            # Define tables
            self._define_tables()
            
            # Create tables if they don't exist
            self.metadata.create_all(self.engine)
            
            # Create session factory
            self.Session = sessionmaker(bind=self.engine)
            
            # Test connection
            with self.engine.connect() as conn:
                conn.execute("SELECT 1")
            
            logger.info(f"SQLAlchemy analytics storage initialized: {self.uri}")
            
        except Exception as e:
            logger.error(f"Failed to initialize SQLAlchemy analytics storage: {e}")
            raise
    
    def _define_tables(self):
        """Define database tables."""
        # Documents table
        self.tables['documents'] = Table(
            f'{self.table_prefix}_documents',
            self.metadata,
            Column('id', Integer, primary_key=True, autoincrement=True),
            Column('doc_id', String(255), index=True),
            Column('doc_type', String(100)),
            Column('source', Text),
            Column('content', Text),
            Column('metadata', JSON),
            Column('run_id', String(100), index=True),
            Column('written_at', DateTime, default=datetime.now),
            Column('year', Integer),
            Column('month', Integer),
            Column('day', Integer),
            Index(f'idx_{self.table_prefix}_docs_partition', 'run_id', 'year', 'month', 'day')
        )
        
        # Elements table
        self.tables['elements'] = Table(
            f'{self.table_prefix}_elements',
            self.metadata,
            Column('id', Integer, primary_key=True, autoincrement=True),
            Column('element_id', String(255), index=True),
            Column('element_type', String(100)),
            Column('doc_id', String(255), index=True),
            Column('parent_id', String(255)),
            Column('content_preview', Text),
            Column('full_content', Text),
            Column('metadata', JSON),
            Column('run_id', String(100), index=True),
            Column('written_at', DateTime, default=datetime.now),
            Column('year', Integer),
            Column('month', Integer),
            Column('day', Integer),
            Index(f'idx_{self.table_prefix}_elems_partition', 'run_id', 'year', 'month', 'day')
        )
        
        # Embeddings table
        self.tables['embeddings'] = Table(
            f'{self.table_prefix}_embeddings',
            self.metadata,
            Column('id', Integer, primary_key=True, autoincrement=True),
            Column('element_id', String(255), index=True),
            Column('embedding', Text),  # JSON array or base64 encoded
            Column('embedding_model', String(100)),
            Column('dimensions', Integer),
            Column('run_id', String(100), index=True),
            Column('written_at', DateTime, default=datetime.now),
            Index(f'idx_{self.table_prefix}_embs_element', 'element_id', 'run_id')
        )
        
        # Relationships table
        self.tables['relationships'] = Table(
            f'{self.table_prefix}_relationships',
            self.metadata,
            Column('id', Integer, primary_key=True, autoincrement=True),
            Column('source_id', String(255), index=True),
            Column('target_id', String(255), index=True),
            Column('relationship_type', String(100)),
            Column('metadata', JSON),
            Column('run_id', String(100), index=True),
            Column('written_at', DateTime, default=datetime.now),
            Index(f'idx_{self.table_prefix}_rels_source_target', 'source_id', 'target_id', 'run_id')
        )
        
        # Metrics table
        self.tables['metrics'] = Table(
            f'{self.table_prefix}_metrics',
            self.metadata,
            Column('id', Integer, primary_key=True, autoincrement=True),
            Column('run_id', String(100), index=True),
            Column('documents_processed', Integer),
            Column('elements_created', Integer),
            Column('relationships_created', Integer),
            Column('embeddings_generated', Integer),
            Column('processing_time', Float),
            Column('errors', Integer),
            Column('metadata', JSON),
            Column('written_at', DateTime, default=datetime.now),
            Index(f'idx_{self.table_prefix}_metrics_run', 'run_id', 'written_at')
        )
    
    def _add_partition_columns(self, data: Dict[str, Any], run_id: str) -> Dict[str, Any]:
        """Add partition columns to data."""
        now = datetime.now()
        data['run_id'] = run_id
        data['written_at'] = now
        
        if 'year' in self.partitioning:
            data['year'] = now.year
        if 'month' in self.partitioning:
            data['month'] = now.month
        if 'day' in self.partitioning:
            data['day'] = now.day
        
        return data
    
    def append_documents(self, documents: List[Dict[str, Any]], 
                        run_id: str) -> int:
        """Append documents to SQL database."""
        if not documents:
            return 0
        
        session = self.Session()
        try:
            table = self.tables['documents']
            
            # Prepare documents for insert
            docs_to_insert = []
            for doc in documents:
                doc_data = self._add_partition_columns(doc.copy(), run_id)
                
                # Convert complex types to JSON strings
                if 'metadata' in doc_data and isinstance(doc_data['metadata'], dict):
                    doc_data['metadata'] = json.dumps(doc_data['metadata'])
                
                docs_to_insert.append(doc_data)
            
            # Bulk insert
            result = session.execute(table.insert(), docs_to_insert)
            session.commit()
            
            inserted = result.rowcount
            logger.info(f"Appended {inserted} documents to SQLAlchemy storage")
            return inserted
            
        except Exception as e:
            session.rollback()
            logger.error(f"Error appending documents to SQLAlchemy storage: {e}")
            raise
        finally:
            session.close()
    
    def append_elements(self, elements: List[Dict[str, Any]], 
                       run_id: str) -> int:
        """Append elements to SQL database."""
        if not elements:
            return 0
        
        session = self.Session()
        try:
            table = self.tables['elements']
            
            # Prepare elements for insert
            elems_to_insert = []
            for elem in elements:
                elem_data = self._add_partition_columns(elem.copy(), run_id)
                
                # Convert complex types to JSON strings
                if 'metadata' in elem_data and isinstance(elem_data['metadata'], dict):
                    elem_data['metadata'] = json.dumps(elem_data['metadata'])
                
                elems_to_insert.append(elem_data)
            
            # Bulk insert
            result = session.execute(table.insert(), elems_to_insert)
            session.commit()
            
            inserted = result.rowcount
            logger.info(f"Appended {inserted} elements to SQLAlchemy storage")
            return inserted
            
        except Exception as e:
            session.rollback()
            logger.error(f"Error appending elements to SQLAlchemy storage: {e}")
            raise
        finally:
            session.close()
    
    def append_embeddings(self, embeddings: List[Dict[str, Any]], 
                         run_id: str) -> int:
        """Append embeddings to SQL database."""
        if not embeddings:
            return 0
        
        session = self.Session()
        try:
            table = self.tables['embeddings']
            
            # Prepare embeddings for insert
            embs_to_insert = []
            for emb in embeddings:
                emb_data = self._add_partition_columns(emb.copy(), run_id)
                
                # Convert embedding vector to JSON string
                if 'embedding' in emb_data:
                    vector = emb_data['embedding']
                    if isinstance(vector, list):
                        emb_data['embedding'] = json.dumps(vector)
                        emb_data['dimensions'] = len(vector)
                
                embs_to_insert.append(emb_data)
            
            # Bulk insert
            result = session.execute(table.insert(), embs_to_insert)
            session.commit()
            
            inserted = result.rowcount
            logger.info(f"Appended {inserted} embeddings to SQLAlchemy storage")
            return inserted
            
        except Exception as e:
            session.rollback()
            logger.error(f"Error appending embeddings to SQLAlchemy storage: {e}")
            raise
        finally:
            session.close()
    
    def append_relationships(self, relationships: List[Dict[str, Any]], 
                           run_id: str) -> int:
        """Append relationships to SQL database."""
        if not relationships:
            return 0
        
        session = self.Session()
        try:
            table = self.tables['relationships']
            
            # Prepare relationships for insert
            rels_to_insert = []
            for rel in relationships:
                rel_data = self._add_partition_columns(rel.copy(), run_id)
                
                # Convert metadata to JSON string
                if 'metadata' in rel_data and isinstance(rel_data['metadata'], dict):
                    rel_data['metadata'] = json.dumps(rel_data['metadata'])
                
                rels_to_insert.append(rel_data)
            
            # Bulk insert
            result = session.execute(table.insert(), rels_to_insert)
            session.commit()
            
            inserted = result.rowcount
            logger.info(f"Appended {inserted} relationships to SQLAlchemy storage")
            return inserted
            
        except Exception as e:
            session.rollback()
            logger.error(f"Error appending relationships to SQLAlchemy storage: {e}")
            raise
        finally:
            session.close()
    
    def append_metrics(self, metrics: Dict[str, Any], 
                      run_id: str) -> bool:
        """Append processing metrics."""
        session = self.Session()
        try:
            table = self.tables['metrics']
            
            # Prepare metrics for insert
            metrics_data = self._add_partition_columns(metrics.copy(), run_id)
            
            # Convert metadata to JSON string if present
            if 'metadata' in metrics_data and isinstance(metrics_data['metadata'], dict):
                metrics_data['metadata'] = json.dumps(metrics_data['metadata'])
            
            # Insert
            session.execute(table.insert(), metrics_data)
            session.commit()
            
            return True
            
        except Exception as e:
            session.rollback()
            logger.error(f"Error appending metrics to SQLAlchemy storage: {e}")
            return False
        finally:
            session.close()
    
    def get_partition_path(self, run_id: str, data_type: str) -> str:
        """Get logical partition path."""
        return f"{self.table_prefix}_{data_type}/{run_id}"
    
    def list_runs(self, start_date: Optional[datetime] = None,
                 end_date: Optional[datetime] = None) -> List[str]:
        """List processing runs in date range."""
        session = self.Session()
        try:
            table = self.tables['metrics']
            query = session.query(table.c.run_id).distinct()
            
            if start_date:
                query = query.filter(table.c.written_at >= start_date)
            if end_date:
                query = query.filter(table.c.written_at <= end_date)
            
            runs = [row[0] for row in query.all()]
            return sorted(runs)
            
        finally:
            session.close()
    
    def get_run_stats(self, run_id: str) -> Optional[Dict[str, Any]]:
        """Get statistics for a processing run."""
        session = self.Session()
        try:
            # Get latest metrics
            metrics_table = self.tables['metrics']
            metrics_row = session.query(metrics_table).filter(
                metrics_table.c.run_id == run_id
            ).order_by(metrics_table.c.written_at.desc()).first()
            
            if not metrics_row:
                return None
            
            stats = dict(metrics_row)
            
            # Add counts from other tables
            for data_type in ['documents', 'elements', 'relationships', 'embeddings']:
                table = self.tables[data_type]
                count = session.query(table).filter(table.c.run_id == run_id).count()
                stats[f'{data_type}_count'] = count
            
            return stats
            
        finally:
            session.close()
    
    def close(self) -> None:
        """Close SQLAlchemy connections."""
        if self.engine:
            self.engine.dispose()
            self.engine = None

    # TODO: Implement sampling methods for MCP database sampling
    def sample_elements(self, filters: Optional[Dict] = None, limit: int = 100,
                       stratify_by: Optional[str] = None,
                       random_seed: Optional[int] = None) -> List[Dict[str, Any]]:
        """
        Sample elements from SQL database with flexible filtering.

        TODO: Implement SQL sampling queries
        - Use TABLESAMPLE or ORDER BY RANDOM() for sampling
        - Implement stratified sampling with window functions
        - Apply filters with WHERE clauses
        """
        raise NotImplementedError("SQLAlchemy sampling methods not yet implemented")

    def get_corpus_stats(self, filters: Optional[Dict] = None) -> Dict[str, Any]:
        """
        Get comprehensive statistics about the database corpus.

        TODO: Implement SQL aggregations for statistics
        - Count rows in each table
        - Get distribution with GROUP BY
        - Calculate statistics with aggregate functions
        """
        raise NotImplementedError("SQLAlchemy corpus stats not yet implemented")

    def sample_documents(self, filters: Optional[Dict] = None, limit: int = 50,
                        random_seed: Optional[int] = None) -> List[Dict[str, Any]]:
        """
        Sample documents from SQL database.

        TODO: Implement document sampling
        - Query documents table with filters
        - Use TABLESAMPLE or ORDER BY RANDOM()
        """
        raise NotImplementedError("SQLAlchemy document sampling not yet implemented")

    def execute_custom_query(self, query: str,
                            params: Optional[List] = None) -> List[Dict[str, Any]]:
        """
        Execute custom SQL query.

        TODO: Implement safe execution of SQL queries
        - Validate query is read-only (SELECT only)
        - Execute with parameters
        - Return results as list of dicts
        """
        raise NotImplementedError("SQLAlchemy custom query execution not yet implemented")