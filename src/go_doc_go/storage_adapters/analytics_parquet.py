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
from ..storage.element_relationship import ElementRelationship

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
            config: Configuration with path/base_path, partitioning scheme, etc.
        """
        super().__init__(config)

        if not PARQUET_AVAILABLE:
            raise ImportError("pandas and pyarrow required for Parquet analytics storage")

        # Handle both 'path' and 'base_path' field names
        path = config.get('base_path') or config.get('path')
        if not path:
            raise ValueError("Parquet analytics requires 'base_path' or 'path' configuration")

        self.base_path = path
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
        self.partitioning = config.get('partitioning', ['date', 'source'])
        
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
    
    def get_partition_path(self, source_name: str, data_type: str) -> str:
        """Generate partition path based on configuration."""
        now = datetime.now()
        partition_values = {
            'date': now.strftime('%Y-%m-%d'),
            'year': now.year,
            'month': f"{now.month:02d}",
            'day': f"{now.day:02d}",
            'hour': f"{now.hour:02d}",
            'source': source_name
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

        # Extract source name from first document, fallback to "unknown"
        source_name = documents[0].get('source', 'unknown') if documents else 'unknown'
        path = self.get_partition_path(source_name, 'documents')
        
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

        # Extract source name from first element's doc metadata, fallback to "unknown"
        source_name = "unknown"
        if elements:
            # Try to get source from element metadata or use fallback
            first_elem = elements[0]
            metadata = first_elem.get('metadata', {})
            if isinstance(metadata, str):
                try:
                    metadata = json.loads(metadata)
                except:
                    metadata = {}
            source_name = metadata.get('source', 'unknown')

        path = self.get_partition_path(source_name, 'elements')

        # Define consistent schema for elements
        # All elements MUST have these columns for consistent parquet schema
        required_columns = {
            'element_id': None,
            'doc_id': None,
            'element_type': None,
            'parent_id': None,
            'content': None,  # Full content (optional, based on store_full_content flag)
            'content_preview': '',
            'content_location': None,
            'content_hash': None,
            'metadata': None,
            'element_order': None,  # Ensure element_order is always present
            'document_position': None,  # Ensure document_position is always present
            'temporal_metadata': None  # Nullable JSON field for temporal elements
        }

        # Process in batches
        total_written = 0
        for i in range(0, len(elements), self.batch_size):
            batch = elements[i:i + self.batch_size]

            # Ensure all elements have required columns
            normalized_batch = []
            for element in batch:
                normalized = {**required_columns, **element}  # Defaults then actual values
                normalized_batch.append(normalized)

            # Prepare DataFrame with consistent columns
            df = pd.DataFrame(normalized_batch)

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

        # Extract source name from first embedding's metadata, fallback to "unknown"
        source_name = "unknown"
        if embeddings:
            first_embedding = embeddings[0]
            # Embeddings might have element_metadata that contains source info
            metadata = first_embedding.get('metadata', {})
            if isinstance(metadata, str):
                try:
                    metadata = json.loads(metadata)
                except:
                    metadata = {}
            source_name = metadata.get('source', 'unknown')

        path = self.get_partition_path(source_name, 'embeddings')
        
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

        # Extract source name from first relationship's metadata, fallback to "unknown"
        source_name = "unknown"
        if relationships:
            first_rel = relationships[0]
            metadata = first_rel.get('metadata', {})
            if isinstance(metadata, str):
                try:
                    metadata = json.loads(metadata)
                except:
                    metadata = {}
            source_name = metadata.get('source', 'unknown')

        path = self.get_partition_path(source_name, 'relationships')
        
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
        # Extract source name from metrics, fallback to "unknown"
        source_name = metrics.get('source', 'unknown')

        path = self.get_partition_path(source_name, 'metrics')
        
        # Add metadata
        metrics['_run_id'] = run_id
        metrics['_written_at'] = datetime.now().isoformat()
        
        # Create single-row DataFrame
        df = pd.DataFrame([metrics])
        
        written = self._write_parquet_batch(df, path, 'metrics')
        return written > 0
    
    def has_run(self, run_id: str) -> bool:
        """
        Check if storage has data for the given run_id.

        Args:
            run_id: Run ID to check

        Returns:
            True if storage has data for this run, False otherwise
        """
        from pathlib import Path

        # Check for elements data (most fundamental)
        elements_path = Path(self.base_path) / 'elements'

        if elements_path.exists():
            # Look for this run_id in the directory structure
            for path in elements_path.rglob(f'run_id={run_id}'):
                if path.is_dir():
                    # Check if directory has any parquet files
                    parquet_files = list(path.glob('*.parquet'))
                    if parquet_files:
                        logger.debug(f"Found {len(parquet_files)} parquet files for run {run_id}")
                        return True

        logger.debug(f"No parquet files found for run {run_id}")
        return False

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
            documents_path = os.path.join(self.base_path, 'documents/**/*.parquet')

            conn.execute(f"CREATE VIEW elements AS SELECT * FROM read_parquet('{elements_path}', union_by_name=true)")
            conn.execute(f"CREATE VIEW embeddings AS SELECT * FROM read_parquet('{embeddings_path}', union_by_name=true)")
            conn.execute(f"CREATE VIEW documents AS SELECT * FROM read_parquet('{documents_path}', union_by_name=true)")

            # Convert query embedding to array string for DuckDB
            query_vec_str = '[' + ','.join(map(str, query_embedding)) + ']'

            # Build filter clause
            filter_clause = ""
            parent_chain_filter = ""
            needs_doc_join = False
            if filters:
                conditions = []
                for key, value in filters.items():
                    if key == 'element_type':
                        # Skip element_type - will be handled via parent chain CTE
                        continue
                    elif key == 'doc_type':
                        # doc_type is in documents table, not elements
                        needs_doc_join = True
                        if isinstance(value, list):
                            # Handle list of doc_types
                            types_str = "', '".join(value)
                            conditions.append(f"d.doc_type IN ('{types_str}')")
                        elif isinstance(value, str):
                            conditions.append(f"d.doc_type = '{value}'")
                        else:
                            conditions.append(f"d.doc_type = {value}")
                    elif isinstance(value, str):
                        conditions.append(f"e.{key} = '{value}'")
                    else:
                        conditions.append(f"e.{key} = {value}")
                if conditions:
                    filter_clause = "AND " + " AND ".join(conditions)
            
            # Build parent chain CTE if element_type filter is present
            parent_chain_cte = ""
            if filters and 'element_type' in filters:
                if isinstance(filters['element_type'], list):
                    element_types = filters['element_type']
                else:
                    element_types = [filters['element_type']] if filters['element_type'] else []

                if element_types:
                    types_str = "', '".join(element_types)
                    parent_chain_cte = f"""WITH RECURSIVE parent_chain AS (
                -- Recursive CTE to check element and all its parents
                SELECT
                    element_id,
                    element_type,
                    parent_id,
                    element_type IN ('{types_str}') as has_type,
                    0 as level
                FROM elements

                UNION ALL

                SELECT
                    pc.element_id,
                    e.element_type,
                    e.parent_id,
                    pc.has_type OR e.element_type IN ('{types_str}') as has_type,
                    pc.level + 1
                FROM parent_chain pc
                JOIN elements e ON pc.parent_id = e.element_id
                WHERE pc.level < 10  -- Limit recursion depth
                    AND NOT pc.has_type  -- Stop if we already found a match
            ), """
                    parent_chain_filter = " AND e.element_id IN (SELECT DISTINCT element_id FROM parent_chain WHERE has_type)"

            # Semantic similarity search with cosine similarity
            # DuckDB doesn't have built-in cosine similarity, so we calculate it manually
            # FIXED: Filter out zero-magnitude embeddings to prevent division by zero and NaN results
            # Build CTE chain properly
            if parent_chain_cte:
                # parent_chain_cte already includes "WITH" and trailing comma
                # Check if we need to join with documents table
                from_clause = "FROM elements e"
                if needs_doc_join:
                    from_clause += " JOIN documents d ON e.doc_id = d.doc_id"

                search_query = f"""
            {parent_chain_cte}
            valid_embeddings AS (
                SELECT
                    e.*,
                    emb.embedding,
                    emb.embedding_text,
                    sqrt(list_dot_product(emb.embedding::DOUBLE[], emb.embedding::DOUBLE[])) as emb_magnitude
                {from_clause}
                JOIN embeddings emb ON e.element_id = emb.element_id
                WHERE sqrt(list_dot_product(emb.embedding::DOUBLE[], emb.embedding::DOUBLE[])) > 0.0
                {filter_clause}
                {parent_chain_filter}
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
            else:
                # No parent_chain_cte, start with WITH
                # Check if we need to join with documents table
                from_clause = "FROM elements e"
                if needs_doc_join:
                    from_clause += " JOIN documents d ON e.doc_id = d.doc_id"

                search_query = f"""
            WITH valid_embeddings AS (
                SELECT
                    e.*,
                    emb.embedding,
                    emb.embedding_text,
                    sqrt(list_dot_product(emb.embedding::DOUBLE[], emb.embedding::DOUBLE[])) as emb_magnitude
                {from_clause}
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

            # Log query details for debugging (can be enabled when needed)
            if logger.isEnabledFor(logging.DEBUG):
                logger.debug(f"Semantic search - filters: {filters}, min_similarity: {min_similarity}")

            results = conn.execute(search_query).fetchall()
            column_names = [desc[0] for desc in conn.description]
            if logger.isEnabledFor(logging.DEBUG):
                logger.debug(f"Search query returned {len(results)} results")
            
            # Convert to list of dicts
            search_results = []
            for row in results:
                result = dict(zip(column_names, row))
                
                if include_context:
                    # Get context: parent, siblings, children
                    result['context'] = self._get_element_context(conn, result['element_id'], result.get('doc_id'))
                
                # Remove embedding vector from result to reduce size (keep embedding_text)
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
            conn.execute(f"CREATE VIEW elements AS SELECT * FROM read_parquet('{elements_path}', union_by_name=true)")
            
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
            conn.execute(f"CREATE VIEW elements AS SELECT * FROM read_parquet('{elements_path}', union_by_name=true)")
            
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
        Joins embeddings (for embedding_text) with elements (for metadata).
        """
        if not DUCKDB_AVAILABLE:
            raise ImportError("DuckDB required for search operations")

        conn = duckdb.connect(':memory:')
        try:
            embeddings_path = os.path.join(self.base_path, 'embeddings/**/*.parquet')
            elements_path = os.path.join(self.base_path, 'elements/**/*.parquet')

            try:
                # Create views for both tables
                conn.execute(f"CREATE VIEW embeddings AS SELECT * FROM read_parquet('{embeddings_path}', union_by_name=true)")
                conn.execute(f"CREATE VIEW elements AS SELECT * FROM read_parquet('{elements_path}', union_by_name=true)")

                # First check if the content column exists
                # Try to get column information
                try:
                    # Test if content column exists
                    test_query = "SELECT content FROM elements LIMIT 0"
                    conn.execute(test_query).fetchone()
                    has_content_column = True
                except:
                    has_content_column = False

                # Build query based on available columns
                if has_content_column:
                    query = """
                    SELECT
                        e.element_id,
                        e.doc_id,
                        e.element_type,
                        e.content,  -- Full content if stored
                        emb.embedding_text,  -- Text used for embedding
                        e.content_preview,
                        e.content_location,
                        e.metadata
                    FROM elements e
                    LEFT JOIN embeddings emb ON e.element_id = emb.element_id
                    WHERE e.element_id = ?
                    LIMIT 1
                    """
                else:
                    # Fallback query without content column for older data
                    query = """
                    SELECT
                        e.element_id,
                        e.doc_id,
                        e.element_type,
                        NULL as content,  -- No content column in older data
                        emb.embedding_text,  -- Text used for embedding
                        e.content_preview,
                        e.content_location,
                        e.metadata
                    FROM elements e
                    LEFT JOIN embeddings emb ON e.element_id = emb.element_id
                    WHERE e.element_id = ?
                    LIMIT 1
                    """

                result = conn.execute(query, [element_id]).fetchone()

                if result:
                    return {
                        'element_id': result[0],
                        'doc_id': result[1],
                        'element_type': result[2],
                        'content': result[3],  # Full content (may be None)
                        'embedding_text': result[4],  # Text used for embedding
                        'content_preview': result[5],
                        'content_location': result[6],
                        'metadata': result[7]
                    }
                return None

            except Exception as e:
                # Handle the case where no parquet files exist yet (first run)
                if "No files found that match the pattern" in str(e):
                    # This is expected on first run - no documents have been processed yet
                    return None
                else:
                    # Re-raise other errors
                    raise

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
            
            try:
                # Use union_by_name=True to handle schema mismatches
                conn.execute(f"CREATE VIEW elements AS SELECT * FROM read_parquet('{elements_path}', union_by_name=True)")
                
                results = conn.execute(f"SELECT * FROM elements WHERE doc_id = '{doc_id}'").fetchall()
                column_names = [desc[0] for desc in conn.description]
                
                return [dict(zip(column_names, row)) for row in results]
                
            except Exception as e:
                # Handle the case where no parquet files exist yet (first run)
                if "No files found that match the pattern" in str(e):
                    # This is expected on first run - no documents have been processed yet
                    return []
                else:
                    # Re-raise other errors
                    raise
            
        finally:
            conn.close()
    
    def sample_elements(self,
                       filters: Optional[Dict[str, Any]] = None,
                       limit: int = 100,
                       stratify_by: Optional[str] = None,
                       random_seed: Optional[int] = None,
                       run_id: Optional[str] = None) -> List[Dict[str, Any]]:
        """
        Sample elements from the parquet data lake.

        Args:
            filters: Column filters to apply
            limit: Maximum number of elements
            stratify_by: Column to stratify sampling by
            random_seed: Random seed for reproducibility
            run_id: Filter to specific run_id

        Returns:
            List of element dictionaries
        """
        if not DUCKDB_AVAILABLE:
            raise ImportError("DuckDB required for sampling operations")

        conn = duckdb.connect(':memory:')
        try:
            # Register parquet files
            elements_path = os.path.join(self.base_path, 'elements/**/*.parquet')
            documents_path = os.path.join(self.base_path, 'documents/**/*.parquet')

            conn.execute(f"CREATE VIEW elements AS SELECT * FROM read_parquet('{elements_path}', hive_partitioning=true, union_by_name=true)")
            conn.execute(f"CREATE VIEW documents AS SELECT * FROM read_parquet('{documents_path}', hive_partitioning=true, union_by_name=true)")

            # Create enriched view
            conn.execute("""
                CREATE VIEW element_document_enriched AS
                SELECT
                    e.*,
                    d.source as doc_source,
                    d.doc_type,
                    json_extract_string(e.metadata, '$.element_name') as structural_name,
                    json_extract_string(e.metadata, '$.path') as structural_path,
                    CASE
                        WHEN e.element_type = 'xml_element' THEN 'xml'
                        WHEN e.element_type = 'json_field' THEN 'json'
                        WHEN e.element_type = 'csv_cell' THEN 'csv'
                        ELSE 'other'
                    END as format_type
                FROM elements e
                LEFT JOIN documents d ON e.doc_id = d.doc_id
            """)

            # Build WHERE clause
            where_conditions = []
            params = []

            # Add run_id filter if provided
            if run_id:
                where_conditions.append(f"run_id = '{run_id}'")

            if filters:
                for key, value in filters.items():
                    if isinstance(value, list):
                        values_str = "', '".join(str(v) for v in value)
                        where_conditions.append(f"{key} IN ('{values_str}')")
                    elif isinstance(value, str) and '*' in value:
                        where_conditions.append(f"{key} LIKE '{value.replace('*', '%')}'")
                    else:
                        where_conditions.append(f"{key} = '{value}'")

            where_clause = " AND ".join(where_conditions) if where_conditions else "TRUE"

            # Build sampling query
            if stratify_by:
                # Set seed if provided
                if random_seed:
                    conn.execute(f"SELECT SETSEED({random_seed / 10000.0})")  # Normalize seed to 0-1 range

                # Stratified sampling
                query = f"""
                WITH stratified AS (
                    SELECT *,
                        ROW_NUMBER() OVER (
                            PARTITION BY {stratify_by}
                            ORDER BY RANDOM()
                        ) as rn
                    FROM element_document_enriched
                    WHERE {where_clause}
                ),
                strata_counts AS (
                    SELECT COUNT(DISTINCT {stratify_by}) as num_strata
                    FROM element_document_enriched
                    WHERE {where_clause}
                )
                SELECT stratified.* FROM stratified, strata_counts
                WHERE rn <= GREATEST(1, CAST({limit} AS FLOAT) / NULLIF(num_strata, 0))
                LIMIT {limit}
                """
            else:
                # Set seed if provided
                if random_seed:
                    conn.execute(f"SELECT SETSEED({random_seed / 10000.0})")  # Normalize seed to 0-1 range

                # Simple random sampling
                query = f"""
                SELECT * FROM element_document_enriched
                WHERE {where_clause}
                ORDER BY RANDOM()
                LIMIT {limit}
                """

            results = conn.execute(query).fetchall()
            column_names = [desc[0] for desc in conn.description]

            return [dict(zip(column_names, row)) for row in results]

        finally:
            conn.close()

    def get_corpus_stats(self, filters: Optional[Dict[str, Any]] = None, run_id: Optional[str] = None) -> Dict[str, Any]:
        """
        Get statistics about the corpus.

        Args:
            filters: Optional filters to apply
            run_id: Filter to specific run_id

        Returns:
            Dictionary with corpus statistics
        """
        if not DUCKDB_AVAILABLE:
            raise ImportError("DuckDB required for statistics operations")

        conn = duckdb.connect(':memory:')
        try:
            # Register parquet files
            elements_path = os.path.join(self.base_path, 'elements/**/*.parquet')
            documents_path = os.path.join(self.base_path, 'documents/**/*.parquet')
            relationships_path = os.path.join(self.base_path, 'relationships/**/*.parquet')

            conn.execute(f"CREATE VIEW elements AS SELECT * FROM read_parquet('{elements_path}', hive_partitioning=true, union_by_name=true)")
            conn.execute(f"CREATE VIEW documents AS SELECT * FROM read_parquet('{documents_path}', hive_partitioning=true, union_by_name=true)")
            conn.execute(f"CREATE VIEW relationships AS SELECT * FROM read_parquet('{relationships_path}', hive_partitioning=true, union_by_name=true)")

            # Build WHERE clause for corpus stats
            where_conditions = []

            # Add run_id filter if provided
            if run_id:
                where_conditions.append(f"run_id = '{run_id}'")

            if filters:
                for key, value in filters.items():
                    if isinstance(value, list):
                        values_str = "', '".join(str(v) for v in value)
                        where_conditions.append(f"{key} IN ('{values_str}')")
                    elif isinstance(value, str) and '*' in value:
                        where_conditions.append(f"{key} LIKE '{value.replace('*', '%')}'")
                    else:
                        where_conditions.append(f"{key} = '{value}'")

            where_clause = " AND ".join(where_conditions) if where_conditions else "TRUE"

            # Get statistics
            stats_query = f"""
            SELECT
                COUNT(*) as total_elements,
                COUNT(DISTINCT doc_id) as total_documents,
                COUNT(DISTINCT element_type) as distinct_element_types
            FROM elements
            WHERE {where_clause}
            """

            stats = conn.execute(stats_query).fetchone()
            column_names = [desc[0] for desc in conn.description]
            result = dict(zip(column_names, stats))

            # Get element type distribution
            dist_query = f"""
            SELECT element_type, COUNT(*) as count
            FROM elements
            WHERE {where_clause}
            GROUP BY element_type
            ORDER BY count DESC
            """

            dist_results = conn.execute(dist_query).fetchall()
            result['element_type_distribution'] = {row[0]: row[1] for row in dist_results}

            # Get relationships count
            rel_count = conn.execute("SELECT COUNT(*) FROM relationships").fetchone()[0]
            result['total_relationships'] = rel_count

            return result

        finally:
            conn.close()

    def sample_documents(self,
                        filters: Optional[Dict[str, Any]] = None,
                        limit: int = 50,
                        random_seed: Optional[int] = None,
                        run_id: Optional[str] = None) -> List[Dict[str, Any]]:
        """
        Sample documents from the parquet data lake.

        Args:
            filters: Column filters to apply
            limit: Maximum number of documents
            random_seed: Random seed for reproducibility
            run_id: Filter to specific run_id

        Returns:
            List of document dictionaries
        """
        if not DUCKDB_AVAILABLE:
            raise ImportError("DuckDB required for sampling operations")

        conn = duckdb.connect(':memory:')
        try:
            # Register parquet files
            documents_path = os.path.join(self.base_path, 'documents/**/*.parquet')
            conn.execute(f"CREATE VIEW documents AS SELECT * FROM read_parquet('{documents_path}', hive_partitioning=true, union_by_name=true)")

            # Build WHERE clause for document sampling
            where_conditions = []

            # Add run_id filter if provided
            if run_id:
                where_conditions.append(f"run_id = '{run_id}'")

            if filters:
                for key, value in filters.items():
                    if isinstance(value, list):
                        values_str = "', '".join(str(v) for v in value)
                        where_conditions.append(f"{key} IN ('{values_str}')")
                    elif isinstance(value, str) and '*' in value:
                        where_conditions.append(f"{key} LIKE '{value.replace('*', '%')}'")
                    else:
                        where_conditions.append(f"{key} = '{value}'")

            where_clause = " AND ".join(where_conditions) if where_conditions else "TRUE"

            # Set seed if provided
            if random_seed:
                conn.execute(f"SELECT SETSEED({random_seed / 10000.0})")  # Normalize seed to 0-1 range

            # Sample documents
            query = f"""
            SELECT * FROM documents
            WHERE {where_clause}
            ORDER BY RANDOM()
            LIMIT {limit}
            """

            results = conn.execute(query).fetchall()
            column_names = [desc[0] for desc in conn.description]

            return [dict(zip(column_names, row)) for row in results]

        finally:
            conn.close()

    def execute_custom_query(self, query: str, params: Optional[List] = None, run_id: Optional[str] = None) -> List[Dict[str, Any]]:
        """
        Execute a custom DuckDB query on the parquet data.

        Args:
            query: DuckDB SQL query
            params: Optional query parameters
            run_id: Filter to specific run_id (note: user must include run_id filter manually in query)

        Returns:
            List of result dictionaries
        """
        if not DUCKDB_AVAILABLE:
            raise ImportError("DuckDB required for custom queries")

        # Safety check - only allow SELECT queries
        if not query.strip().upper().startswith('SELECT'):
            raise ValueError("Only SELECT queries are allowed")

        conn = duckdb.connect(':memory:')
        try:
            # Register all parquet files as views
            elements_path = os.path.join(self.base_path, 'elements/**/*.parquet')
            documents_path = os.path.join(self.base_path, 'documents/**/*.parquet')
            relationships_path = os.path.join(self.base_path, 'relationships/**/*.parquet')
            embeddings_path = os.path.join(self.base_path, 'embeddings/**/*.parquet')

            conn.execute(f"CREATE VIEW elements AS SELECT * FROM read_parquet('{elements_path}', hive_partitioning=true, union_by_name=true)")
            conn.execute(f"CREATE VIEW documents AS SELECT * FROM read_parquet('{documents_path}', hive_partitioning=true, union_by_name=true)")
            conn.execute(f"CREATE VIEW relationships AS SELECT * FROM read_parquet('{relationships_path}', hive_partitioning=true, union_by_name=true)")
            conn.execute(f"CREATE VIEW embeddings AS SELECT * FROM read_parquet('{embeddings_path}', hive_partitioning=true)")

            # Execute query
            if params:
                results = conn.execute(query, params).fetchall()
            else:
                results = conn.execute(query).fetchall()

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
    
    def get_outgoing_relationships(self, element_id: str) -> List[ElementRelationship]:
        """
        Find all relationships where the specified element_id is the source.

        Args:
            element_id: The ID of the source element

        Returns:
            List of ElementRelationship objects where this element is the source
        """
        if not DUCKDB_AVAILABLE:
            raise ImportError("DuckDB required for relationship operations")
            
        conn = duckdb.connect(':memory:')
        try:
            # Set up Parquet file paths
            relationships_path = os.path.join(self.base_path, 'relationships/**/*.parquet')
            elements_path = os.path.join(self.base_path, 'elements/**/*.parquet')
            
            try:
                # Create views for relationships and elements
                conn.execute(f"CREATE VIEW relationships AS SELECT * FROM read_parquet('{relationships_path}', union_by_name=True)")
                conn.execute(f"CREATE VIEW elements AS SELECT * FROM read_parquet('{elements_path}', union_by_name=True)")
                
                # Query for outgoing relationships with target element details
                # Filter by element_id directly since parquet doesn't have element_pk
                results = conn.execute(f"""
                    SELECT
                        r.relationship_id,
                        r.source_id,
                        r.relationship_type,
                        r.target_id as target_reference,
                        r.doc_id,
                        r.metadata,
                        NULL as source_element_pk,
                        t.element_type as target_element_type,
                        t.content_preview as target_content_preview,
                        NULL as target_element_pk
                    FROM relationships r
                    LEFT JOIN elements t ON r.target_id = t.element_id
                    WHERE r.source_id = '{element_id}'
                """).fetchall()
                
                column_names = [desc[0] for desc in conn.description]
                relationship_dicts = [dict(zip(column_names, row)) for row in results]
                
                # Convert to ElementRelationship objects
                element_relationships = []
                for rel_dict in relationship_dicts:
                    # Parse metadata if it's a JSON string
                    metadata = rel_dict.get('metadata', {})
                    if isinstance(metadata, str) and metadata:
                        try:
                            import json
                            metadata = json.loads(metadata)
                        except json.JSONDecodeError:
                            metadata = {}
                    elif metadata is None:
                        metadata = {}
                    
                    element_rel = ElementRelationship(
                        relationship_id=rel_dict.get('relationship_id', ''),
                        source_id=rel_dict.get('source_id', ''),
                        source_element_pk=rel_dict.get('source_element_pk'),
                        source_element_type=None,  # We know this is the source but don't have type here
                        relationship_type=rel_dict.get('relationship_type', ''),
                        target_reference=rel_dict.get('target_reference') or '',  # Default to empty string if None
                        target_element_pk=rel_dict.get('target_element_pk'),
                        target_element_type=rel_dict.get('target_element_type'),
                        target_content_preview=rel_dict.get('target_content_preview'),
                        doc_id=rel_dict.get('doc_id'),
                        metadata=metadata,
                        is_source=True  # This element is always the source for outgoing relationships
                    )
                    element_relationships.append(element_rel)
                
                return element_relationships
                
            except Exception as e:
                if "No files found that match the pattern" in str(e):
                    # No relationship data exists yet
                    return []
                else:
                    raise
                    
        finally:
            conn.close()
    
    def cleanup_run(self, run_id: str) -> Dict[str, int]:
        """
        Clean up all parquet files for a specific processing run.

        Since parquet files are immutable, we delete entire directories
        organized by run_id rather than attempting record-level deletion.

        Args:
            run_id: Processing run identifier to clean up

        Returns:
            Dictionary with cleanup statistics
        """
        import shutil
        from pathlib import Path

        stats = {
            'files_removed': 0,
            'directories_removed': 0,
            'total_bytes_freed': 0,
            'storage_type': 'parquet'
        }

        if self.use_s3:
            logger.warning("S3 cleanup not yet implemented")
            return stats

        base_path = Path(self.base_path)

        # Data types that may contain run_id partitions
        data_types = ['documents', 'elements', 'embeddings', 'relationships', 'metrics']

        for data_type in data_types:
            data_dir = base_path / data_type
            if not data_dir.exists():
                continue

            # Find all directories with this run_id
            # Pattern: data_type/year=2024/month=01/day=15/run_id={run_id}/
            for run_id_dir in data_dir.rglob(f'run_id={run_id}'):
                if run_id_dir.is_dir():
                    # Calculate size before deletion
                    dir_size = sum(f.stat().st_size for f in run_id_dir.rglob('*.parquet') if f.is_file())
                    file_count = len(list(run_id_dir.rglob('*.parquet')))

                    logger.info(f"Removing run_id directory: {run_id_dir} ({file_count} files, {dir_size} bytes)")

                    try:
                        shutil.rmtree(run_id_dir)
                        stats['directories_removed'] += 1
                        stats['files_removed'] += file_count
                        stats['total_bytes_freed'] += dir_size
                    except Exception as e:
                        logger.error(f"Failed to remove directory {run_id_dir}: {e}")

        # Clean up empty parent directories after removing run_id directories
        for data_type in data_types:
            data_dir = base_path / data_type
            if data_dir.exists():
                # Remove empty year/month/day directories that may be left behind
                for year_dir in data_dir.iterdir():
                    if year_dir.is_dir() and year_dir.name.startswith('year='):
                        for month_dir in year_dir.iterdir():
                            if month_dir.is_dir() and month_dir.name.startswith('month='):
                                for day_dir in month_dir.iterdir():
                                    if day_dir.is_dir() and day_dir.name.startswith('day='):
                                        # Remove day directory if empty
                                        if not any(day_dir.iterdir()):
                                            try:
                                                day_dir.rmdir()
                                                logger.debug(f"Removed empty day directory: {day_dir}")
                                            except OSError:
                                                pass
                                # Remove month directory if empty
                                if not any(month_dir.iterdir()):
                                    try:
                                        month_dir.rmdir()
                                        logger.debug(f"Removed empty month directory: {month_dir}")
                                    except OSError:
                                        pass
                        # Remove year directory if empty
                        if not any(year_dir.iterdir()):
                            try:
                                year_dir.rmdir()
                                logger.debug(f"Removed empty year directory: {year_dir}")
                            except OSError:
                                pass

        # Add implementation details for UI
        if stats['files_removed'] > 0:
            stats['implementation_details'] = f"deleting {stats['files_removed']} parquet files in {stats['directories_removed']} directories"
        else:
            stats['implementation_details'] = "no parquet files found to delete"

        logger.info(f"Cleanup completed for run {run_id}: {stats}")
        return stats

    def close(self) -> None:
        """Close storage connections."""
        # Parquet doesn't maintain persistent connections
        pass