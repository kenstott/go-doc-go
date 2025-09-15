"""
Parquet analytics storage adapter for OLAP operations.
"""

import json
import logging
import os
import uuid
from datetime import datetime
from typing import Dict, Any, List, Optional
from pathlib import Path

from .base import AnalyticsStorage

logger = logging.getLogger(__name__)

try:
    import pandas as pd
    import pyarrow as pa
    import pyarrow.parquet as pq
    PARQUET_AVAILABLE = True
except ImportError:
    PARQUET_AVAILABLE = False
    logger.warning("Parquet libraries not available. Install with: pip install pandas pyarrow")

try:
    import duckdb
    import numpy as np
    DUCKDB_AVAILABLE = True
except ImportError:
    DUCKDB_AVAILABLE = False
    logger.warning("DuckDB not available. Install with: pip install duckdb numpy")


class ParquetAnalyticsStorage(AnalyticsStorage):
    """Parquet implementation of analytics storage for data lake architectures."""
    
    def __init__(self, config: Dict[str, Any]):
        """
        Initialize Parquet analytics storage.
        
        Args:
            config: Configuration with path, partitioning scheme, etc.
        """
        super().__init__(config)
        
        if not PARQUET_AVAILABLE:
            raise ImportError("pandas and pyarrow required for Parquet analytics storage")
        
        self.base_path = config.get('path', './data-lake')
        self.s3_bucket = None
        self.s3_prefix = None
        
        # Parse S3 path if provided
        if self.base_path.startswith('s3://'):
            parts = self.base_path[5:].split('/', 1)
            self.s3_bucket = parts[0]
            self.s3_prefix = parts[1] if len(parts) > 1 else ''
            self.use_s3 = True
            
            # Initialize S3 filesystem
            try:
                import s3fs
                self.fs = s3fs.S3FileSystem()
                S3_AVAILABLE = True
            except ImportError:
                logger.warning("s3fs not available. Install with: pip install s3fs")
                # Fall back to local filesystem
                self.use_s3 = False
                self.fs = None
        else:
            self.use_s3 = False
            self.fs = None
        
        # Partitioning configuration
        self.partitioning = config.get('partitioning', ['year', 'month', 'day', 'run_id'])
        
        # Compression settings
        self.compression = config.get('compression', 'snappy')
        
        # Batch size for writing
        self.batch_size = config.get('batch_size', 10000)
        
        self.initialize()
    
    def initialize(self) -> None:
        """Initialize Parquet storage structure."""
        if not self.use_s3:
            # Create local directory structure
            base = Path(self.base_path)
            for data_type in ['documents', 'elements', 'embeddings', 'relationships', 'metrics']:
                (base / data_type).mkdir(parents=True, exist_ok=True)
            
            logger.info(f"Parquet analytics storage initialized at {self.base_path}")
        else:
            logger.info(f"Parquet analytics storage initialized at s3://{self.s3_bucket}/{self.s3_prefix}")
    
    def get_partition_path(self, run_id: str, data_type: str) -> str:
        """Generate partition path based on configuration."""
        now = datetime.now()
        partition_values = {
            'year': now.year,
            'month': f"{now.month:02d}",
            'day': f"{now.day:02d}",
            'hour': f"{now.hour:02d}",
            'run_id': run_id
        }
        
        # Build partition path
        partitions = []
        for part in self.partitioning:
            if part in partition_values:
                partitions.append(f"{part}={partition_values[part]}")
        
        if self.use_s3:
            return f"s3://{self.s3_bucket}/{self.s3_prefix}/{data_type}/{'/'.join(partitions)}"
        else:
            return os.path.join(self.base_path, data_type, *partitions)
    
    def _write_parquet_batch(self, df: pd.DataFrame, path: str, 
                            data_type: str) -> int:
        """Write a batch of data to Parquet file."""
        if df.empty:
            return 0
        
        # Generate unique filename
        filename = f"{data_type}_{uuid.uuid4().hex[:8]}.parquet"
        full_path = os.path.join(path, filename)
        
        # Ensure directory exists (for local filesystem)
        if not self.use_s3:
            Path(path).mkdir(parents=True, exist_ok=True)
        
        # Write Parquet file
        try:
            if self.use_s3 and self.fs:
                # Write to S3
                table = pa.Table.from_pandas(df)
                pq.write_table(
                    table,
                    full_path,
                    filesystem=self.fs,
                    compression=self.compression
                )
            else:
                # Write to local filesystem
                df.to_parquet(
                    full_path,
                    engine='pyarrow',
                    compression=self.compression,
                    index=False
                )
            
            logger.debug(f"Wrote {len(df)} {data_type} records to {full_path}")
            return len(df)
            
        except Exception as e:
            logger.error(f"Error writing Parquet file {full_path}: {e}")
            raise
    
    def append_documents(self, documents: List[Dict[str, Any]], 
                        run_id: str) -> int:
        """Append documents to Parquet storage."""
        if not documents:
            return 0
        
        path = self.get_partition_path(run_id, 'documents')
        
        # Process in batches
        total_written = 0
        for i in range(0, len(documents), self.batch_size):
            batch = documents[i:i + self.batch_size]
            
            # Prepare DataFrame
            df = pd.DataFrame(batch)
            
            # Add processing metadata
            df['_run_id'] = run_id
            df['_written_at'] = datetime.now().isoformat()
            
            # Handle nested JSON fields - standardize metadata to strings
            if 'metadata' in df.columns:
                def standardize_metadata(meta):
                    if meta is None or (isinstance(meta, float) and pd.isna(meta)):
                        return '{}'
                    elif isinstance(meta, str):
                        return meta  # Already a string
                    elif isinstance(meta, (dict, list)):
                        return json.dumps(meta)
                    else:
                        try:
                            return json.dumps(meta)
                        except (TypeError, ValueError):
                            return '{}'
                
                df['metadata'] = df['metadata'].apply(standardize_metadata)
            
            # Convert other dict/list columns to JSON strings
            for col in df.columns:
                if col != 'metadata' and df[col].dtype == 'object':
                    try:
                        if isinstance(df[col].iloc[0], (dict, list)):
                            df[col] = df[col].apply(json.dumps)
                    except (IndexError, TypeError):
                        pass
            
            written = self._write_parquet_batch(df, path, 'documents')
            total_written += written
        
        logger.info(f"Appended {total_written} documents to {path}")
        return total_written
    
    def append_elements(self, elements: List[Dict[str, Any]], 
                       run_id: str) -> int:
        """Append elements to Parquet storage."""
        if not elements:
            return 0
        
        path = self.get_partition_path(run_id, 'elements')
        
        # Process in batches
        total_written = 0
        for i in range(0, len(elements), self.batch_size):
            batch = elements[i:i + self.batch_size]
            
            # Prepare DataFrame
            df = pd.DataFrame(batch)
            
            # Add processing metadata
            df['_run_id'] = run_id
            df['_written_at'] = datetime.now().isoformat()
            
            # Handle nested JSON fields
            for col in df.columns:
                if df[col].dtype == 'object':
                    try:
                        if isinstance(df[col].iloc[0], (dict, list)):
                            df[col] = df[col].apply(json.dumps)
                    except (IndexError, TypeError):
                        pass
            
            written = self._write_parquet_batch(df, path, 'elements')
            total_written += written
        
        logger.info(f"Appended {total_written} elements to {path}")
        return total_written
    
    def append_embeddings(self, embeddings: List[Dict[str, Any]], 
                         run_id: str) -> int:
        """Append embeddings to Parquet storage."""
        if not embeddings:
            return 0
        
        path = self.get_partition_path(run_id, 'embeddings')
        
        # Process in batches
        total_written = 0
        for i in range(0, len(embeddings), self.batch_size):
            batch = embeddings[i:i + self.batch_size]
            
            # Prepare DataFrame
            df = pd.DataFrame(batch)
            
            # Add processing metadata
            df['_run_id'] = run_id
            df['_written_at'] = datetime.now().isoformat()
            
            # Convert embedding vectors to arrays if needed
            if 'embedding' in df.columns:
                # Ensure embeddings are stored as arrays
                df['embedding'] = df['embedding'].apply(
                    lambda x: x if isinstance(x, list) else list(x)
                )
            
            written = self._write_parquet_batch(df, path, 'embeddings')
            total_written += written
        
        logger.info(f"Appended {total_written} embeddings to {path}")
        return total_written
    
    def append_relationships(self, relationships: List[Dict[str, Any]], 
                           run_id: str) -> int:
        """Append relationships to Parquet storage."""
        if not relationships:
            return 0
        
        path = self.get_partition_path(run_id, 'relationships')
        
        # Process in batches
        total_written = 0
        for i in range(0, len(relationships), self.batch_size):
            batch = relationships[i:i + self.batch_size]
            
            # Prepare DataFrame
            df = pd.DataFrame(batch)
            
            # Standardize schema before writing
            # Rename target_reference to target_id if present
            if 'target_reference' in df.columns and 'target_id' not in df.columns:
                df = df.rename(columns={'target_reference': 'target_id'})
                logger.debug("Renamed target_reference to target_id for consistency")
            
            # Ensure required columns exist
            required_columns = ['relationship_id', 'source_id', 'target_id', 'relationship_type']
            missing_columns = [col for col in required_columns if col not in df.columns]
            if missing_columns:
                logger.warning(f"Missing required columns in relationships: {missing_columns}")
            
            # CRITICAL: Standardize metadata field to ensure consistent schema
            # Convert all metadata to JSON strings for consistent storage
            if 'metadata' in df.columns:
                def standardize_metadata(meta):
                    if meta is None or (isinstance(meta, float) and pd.isna(meta)):
                        return '{}'
                    elif isinstance(meta, str):
                        # Already a string, validate it's valid JSON
                        try:
                            json.loads(meta)
                            return meta
                        except (json.JSONDecodeError, TypeError):
                            return '{}'
                    elif isinstance(meta, dict):
                        # Convert dict to JSON string
                        return json.dumps(meta)
                    else:
                        # Convert other types to JSON string
                        try:
                            return json.dumps(meta)
                        except (TypeError, ValueError):
                            return '{}'
                
                df['metadata'] = df['metadata'].apply(standardize_metadata)
            else:
                # Add empty metadata if not present
                df['metadata'] = '{}'
            
            # Add processing metadata
            df['_run_id'] = run_id
            df['_written_at'] = datetime.now().isoformat()
            
            # Define standard column order
            column_order = [
                'relationship_id', 'source_id', 'target_id', 'relationship_type',
                'metadata', 'doc_id', '_run_id', '_written_at'
            ]
            
            # Reorder columns (only include columns that exist)
            existing_columns = [col for col in column_order if col in df.columns]
            # Add any extra columns not in the standard order
            extra_columns = [col for col in df.columns if col not in column_order]
            df = df[existing_columns + extra_columns]
            
            written = self._write_parquet_batch(df, path, 'relationships')
            total_written += written
        
        logger.info(f"Appended {total_written} relationships to {path}")
        return total_written
    
    def append_metrics(self, metrics: Dict[str, Any], 
                      run_id: str) -> bool:
        """Append processing metrics."""
        path = self.get_partition_path(run_id, 'metrics')
        
        # Add metadata
        metrics['_run_id'] = run_id
        metrics['_written_at'] = datetime.now().isoformat()
        
        # Create single-row DataFrame
        df = pd.DataFrame([metrics])
        
        written = self._write_parquet_batch(df, path, 'metrics')
        return written > 0
    
    def list_runs(self, start_date: Optional[datetime] = None,
                 end_date: Optional[datetime] = None) -> List[str]:
        """List processing runs in date range."""
        runs = set()
        
        # Scan partition directories
        if self.use_s3 and self.fs:
            # List S3 directories
            base = f"s3://{self.s3_bucket}/{self.s3_prefix}/metrics"
            try:
                for path in self.fs.glob(f"{base}/**/metrics_*.parquet"):
                    # Extract run_id from path
                    parts = path.split('/')
                    for part in parts:
                        if part.startswith('run_id='):
                            runs.add(part.split('=')[1])
            except Exception as e:
                logger.error(f"Error listing S3 runs: {e}")
        else:
            # List local directories
            base = Path(self.base_path) / 'metrics'
            if base.exists():
                for path in base.rglob('metrics_*.parquet'):
                    # Extract run_id from path
                    parts = str(path).split(os.sep)
                    for part in parts:
                        if part.startswith('run_id='):
                            runs.add(part.split('=')[1])
        
        return sorted(list(runs))
    
    def get_run_stats(self, run_id: str) -> Optional[Dict[str, Any]]:
        """Get statistics for a processing run."""
        # Read metrics for the run
        metrics_pattern = f"**/run_id={run_id}/metrics_*.parquet"
        
        try:
            if self.use_s3 and self.fs:
                base = f"s3://{self.s3_bucket}/{self.s3_prefix}/metrics"
                files = self.fs.glob(f"{base}/{metrics_pattern}")
                
                if files:
                    # Read first metrics file
                    df = pd.read_parquet(files[0], filesystem=self.fs)
                    if not df.empty:
                        return df.iloc[0].to_dict()
            else:
                base = Path(self.base_path) / 'metrics'
                files = list(base.glob(metrics_pattern))
                
                if files:
                    # Read first metrics file
                    df = pd.read_parquet(files[0])
                    if not df.empty:
                        return df.iloc[0].to_dict()
        
        except Exception as e:
            logger.error(f"Error reading run stats for {run_id}: {e}")
        
        return None
    
    def search_semantic(self, query_embedding: List[float], 
                       limit: int = 10,
                       min_similarity: float = 0.0,
                       filters: Optional[Dict[str, Any]] = None,
                       include_context: bool = True) -> List[Dict[str, Any]]:
        """
        Search for similar content using embeddings with DuckDB.
        """
        if not DUCKDB_AVAILABLE:
            raise ImportError("DuckDB required for search operations")
        
        conn = duckdb.connect(':memory:')
        try:
            # Register Parquet files as views
            elements_path = os.path.join(self.base_path, 'elements/**/*.parquet')
            embeddings_path = os.path.join(self.base_path, 'embeddings/**/*.parquet')
            
            conn.execute(f"CREATE VIEW elements AS SELECT * FROM read_parquet('{elements_path}')")
            conn.execute(f"CREATE VIEW embeddings AS SELECT * FROM read_parquet('{embeddings_path}')")
            
            # Convert query embedding to array string for DuckDB
            query_vec_str = '[' + ','.join(map(str, query_embedding)) + ']'
            
            # Build filter clause
            filter_clause = ""
            if filters:
                conditions = []
                for key, value in filters.items():
                    if isinstance(value, str):
                        conditions.append(f"e.{key} = '{value}'")
                    else:
                        conditions.append(f"e.{key} = {value}")
                if conditions:
                    filter_clause = "AND " + " AND ".join(conditions)
            
            # Semantic similarity search with cosine similarity
            # DuckDB doesn't have built-in cosine similarity, so we calculate it manually
            # FIXED: Filter out zero-magnitude embeddings to prevent division by zero and NaN results
            search_query = f"""
            WITH valid_embeddings AS (
                SELECT 
                    e.*,
                    emb.embedding,
                    sqrt(list_dot_product(emb.embedding::DOUBLE[], emb.embedding::DOUBLE[])) as emb_magnitude
                FROM elements e
                JOIN embeddings emb ON e.element_id = emb.element_id
                WHERE sqrt(list_dot_product(emb.embedding::DOUBLE[], emb.embedding::DOUBLE[])) > 0.0
                {filter_clause}
            ),
            search_results AS (
                SELECT 
                    *,
                    (
                        list_dot_product(embedding::DOUBLE[], {query_vec_str}::DOUBLE[]) / 
                        (emb_magnitude * sqrt(list_dot_product({query_vec_str}::DOUBLE[], {query_vec_str}::DOUBLE[])))
                    ) as similarity
                FROM valid_embeddings
            )
            SELECT * FROM search_results
            WHERE similarity >= {min_similarity}
            AND similarity IS NOT NULL
            AND NOT isnan(similarity)
            ORDER BY similarity DESC
            LIMIT {limit}
            """
            
            results = conn.execute(search_query).fetchall()
            column_names = [desc[0] for desc in conn.description]
            
            # Convert to list of dicts
            search_results = []
            for row in results:
                result = dict(zip(column_names, row))
                
                if include_context:
                    # Get context: parent, siblings, children
                    result['context'] = self._get_element_context(conn, result['element_id'], result.get('doc_id'))
                
                # Remove embedding from result to reduce size
                result.pop('embedding', None)
                search_results.append(result)
            
            return search_results
            
        finally:
            conn.close()
    
    def search_text(self, query: str,
                   limit: int = 10,
                   filters: Optional[Dict[str, Any]] = None) -> List[Dict[str, Any]]:
        """
        Full-text search across content using DuckDB.
        """
        if not DUCKDB_AVAILABLE:
            raise ImportError("DuckDB required for search operations")
        
        conn = duckdb.connect(':memory:')
        try:
            # Register Parquet files
            elements_path = os.path.join(self.base_path, 'elements/**/*.parquet')
            conn.execute(f"CREATE VIEW elements AS SELECT * FROM read_parquet('{elements_path}')")
            
            # Build filter clause
            filter_clause = ""
            if filters:
                conditions = []
                for key, value in filters.items():
                    if isinstance(value, str):
                        conditions.append(f"{key} = '{value}'")
                    else:
                        conditions.append(f"{key} = {value}")
                if conditions:
                    filter_clause = "WHERE " + " AND ".join(conditions)
            
            # Text search query
            search_query = f"""
            SELECT *
            FROM elements
            WHERE content_preview ILIKE '%{query}%'
            {filter_clause if not filter_clause else 'AND' + filter_clause[5:]}
            LIMIT {limit}
            """
            
            results = conn.execute(search_query).fetchall()
            column_names = [desc[0] for desc in conn.description]
            
            return [dict(zip(column_names, row)) for row in results]
            
        finally:
            conn.close()
    
    def search_structured(self, criteria: Dict[str, Any],
                         limit: int = 10) -> List[Dict[str, Any]]:
        """
        Structured search with multiple criteria using DuckDB.
        """
        if not DUCKDB_AVAILABLE:
            raise ImportError("DuckDB required for search operations")
        
        conn = duckdb.connect(':memory:')
        try:
            # Register Parquet files
            elements_path = os.path.join(self.base_path, 'elements/**/*.parquet')
            conn.execute(f"CREATE VIEW elements AS SELECT * FROM read_parquet('{elements_path}')")
            
            # Build WHERE clause from criteria
            conditions = []
            for key, value in criteria.items():
                if isinstance(value, dict):
                    # Handle operators like {'$gte': 100}
                    for op, val in value.items():
                        if op == '$gte':
                            conditions.append(f"{key} >= {val}")
                        elif op == '$lte':
                            conditions.append(f"{key} <= {val}")
                        elif op == '$gt':
                            conditions.append(f"{key} > {val}")
                        elif op == '$lt':
                            conditions.append(f"{key} < {val}")
                        elif op == '$ne':
                            conditions.append(f"{key} != '{val}'")
                        elif op == '$in':
                            values = ','.join([f"'{v}'" for v in val])
                            conditions.append(f"{key} IN ({values})")
                elif isinstance(value, str):
                    conditions.append(f"{key} = '{value}'")
                else:
                    conditions.append(f"{key} = {value}")
            
            where_clause = " AND ".join(conditions) if conditions else "1=1"
            
            search_query = f"""
            SELECT *
            FROM elements
            WHERE {where_clause}
            LIMIT {limit}
            """
            
            results = conn.execute(search_query).fetchall()
            column_names = [desc[0] for desc in conn.description]
            
            return [dict(zip(column_names, row)) for row in results]
            
        finally:
            conn.close()
    
    def get_element_by_id(self, element_id: str) -> Optional[Dict[str, Any]]:
        """
        Retrieve a specific element by ID using DuckDB.
        """
        if not DUCKDB_AVAILABLE:
            raise ImportError("DuckDB required for search operations")
        
        conn = duckdb.connect(':memory:')
        try:
            elements_path = os.path.join(self.base_path, 'elements/**/*.parquet')
            conn.execute(f"CREATE VIEW elements AS SELECT * FROM read_parquet('{elements_path}')")
            
            result = conn.execute(f"SELECT * FROM elements WHERE element_id = '{element_id}' LIMIT 1").fetchone()
            
            if result:
                column_names = [desc[0] for desc in conn.description]
                return dict(zip(column_names, result))
            return None
            
        finally:
            conn.close()
    
    def get_document_elements(self, doc_id: str) -> List[Dict[str, Any]]:
        """
        Get all elements for a document using DuckDB.
        """
        if not DUCKDB_AVAILABLE:
            raise ImportError("DuckDB required for search operations")
        
        conn = duckdb.connect(':memory:')
        try:
            elements_path = os.path.join(self.base_path, 'elements/**/*.parquet')
            # Use union_by_name=True to handle schema mismatches
            conn.execute(f"CREATE VIEW elements AS SELECT * FROM read_parquet('{elements_path}', union_by_name=True)")
            
            results = conn.execute(f"SELECT * FROM elements WHERE doc_id = '{doc_id}'").fetchall()
            column_names = [desc[0] for desc in conn.description]
            
            return [dict(zip(column_names, row)) for row in results]
            
        finally:
            conn.close()
    
    def _get_element_context(self, conn: Any, element_id: str, doc_id: Optional[str]) -> Dict[str, Any]:
        """
        Get surrounding context for an element.
        """
        context = {
            'parent': None,
            'siblings': [],
            'children': []
        }
        
        # Get the element itself first
        element = conn.execute(f"SELECT * FROM elements WHERE element_id = '{element_id}' LIMIT 1").fetchone()
        if not element:
            return context
        
        column_names = [desc[0] for desc in conn.description]
        element_dict = dict(zip(column_names, element))
        
        # Get parent if exists
        if element_dict.get('parent_id'):
            parent = conn.execute(f"SELECT * FROM elements WHERE element_id = '{element_dict['parent_id']}' LIMIT 1").fetchone()
            if parent:
                context['parent'] = dict(zip(column_names, parent))
        
        # Get siblings (same parent, different element)
        if element_dict.get('parent_id'):
            siblings = conn.execute(f"""
                SELECT * FROM elements 
                WHERE parent_id = '{element_dict['parent_id']}' 
                AND element_id != '{element_id}'
                LIMIT 5
            """).fetchall()
            context['siblings'] = [dict(zip(column_names, row)) for row in siblings]
        
        # Get children
        children = conn.execute(f"""
            SELECT * FROM elements 
            WHERE parent_id = '{element_id}'
            LIMIT 10
        """).fetchall()
        context['children'] = [dict(zip(column_names, row)) for row in children]
        
        return context
    
    def close(self) -> None:
        """Close storage connections."""
        # Parquet doesn't maintain persistent connections
        pass