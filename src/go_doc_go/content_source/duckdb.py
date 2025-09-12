"""
DuckDB Content Source for hive-partitioned parquet files.

This module provides access to documents from DuckDB-queryable parquet datasets,
particularly useful for hive-partitioned datasets like SEC filings.
"""

import hashlib
import json
import logging
import os
import re
from typing import Dict, Any, List, Optional, TYPE_CHECKING

# Import types for type checking only - these won't be imported at runtime
if TYPE_CHECKING:
    import duckdb

from .base import ContentSource

logger = logging.getLogger(__name__)

# Define global flags for availability - these will be set at runtime
DUCKDB_AVAILABLE = False

# Try to import DuckDB conditionally
try:
    import duckdb
    DUCKDB_AVAILABLE = True
except ImportError:
    logger.warning("DuckDB not available. Install with 'pip install duckdb'.")


class DuckDBContentSource(ContentSource):
    """Content source for DuckDB-queryable parquet files with hive partitioning support."""

    def __init__(self, config: Dict[str, Any]):
        """
        Initialize the DuckDB content source.

        Args:
            config: Configuration dictionary with:
                - database_path: Path to parquet dataset or DuckDB database file
                - queries: List of query configurations, each with:
                    - name: Query identifier
                    - sql: SQL query to execute
                    - id_columns: List of columns to use for document ID (or doc_id_columns for compatibility)
                    - content_column: Column containing the document content
                    - metadata_columns: Optional list of columns to include in metadata (defaults to all non-content columns)
                    - doc_type: Optional document type (default: "text")
                - connection_config: Optional DuckDB connection settings
                - enable_hive_partitioning: Enable hive-style partitioning (default: True)
        """
        if not DUCKDB_AVAILABLE:
            raise ImportError("DuckDB is required for DuckDBContentSource but not available")

        super().__init__(config)
        self.database_path = config.get("database_path")
        self.queries = config.get("queries", [])
        self.connection_config = config.get("connection_config", {})
        self.enable_hive_partitioning = config.get("enable_hive_partitioning", True)
        
        # Connection will be created per operation for thread safety
        self._connection: Optional['duckdb.DuckDBPyConnection'] = None
        
        if not self.database_path:
            raise ValueError("database_path is required for DuckDBContentSource")
        
        if not self.queries:
            raise ValueError("At least one query must be configured")
        
        # Validate query configurations
        self._validate_queries()
        
        logger.debug(f"Initialized DuckDB content source with database path: {self.database_path}")

    def _validate_queries(self):
        """Validate query configurations."""
        for i, query in enumerate(self.queries):
            if "name" not in query:
                raise ValueError(f"Query {i} missing required 'name' field")
            if "sql" not in query:
                raise ValueError(f"Query '{query['name']}' missing required 'sql' field")
            
            # Support both id_columns and doc_id_columns for backward compatibility
            if "id_columns" not in query and "doc_id_columns" not in query:
                raise ValueError(f"Query '{query['name']}' missing required 'id_columns' or 'doc_id_columns' field")
            
            # Normalize to id_columns internally
            if "doc_id_columns" in query and "id_columns" not in query:
                query["id_columns"] = query["doc_id_columns"]
            
            if "content_column" not in query:
                raise ValueError(f"Query '{query['name']}' missing required 'content_column' field")

    def _get_connection(self) -> 'duckdb.DuckDBPyConnection':
        """Get a DuckDB connection, creating if necessary."""
        # Create a new connection each time for thread safety
        if os.path.isfile(self.database_path):
            # It's a DuckDB database file
            conn = duckdb.connect(self.database_path)
        else:
            # It's a directory path - use in-memory database
            conn = duckdb.connect(":memory:")
        
        # Configure connection
        if self.enable_hive_partitioning:
            conn.execute("SET enable_object_cache=true")
            conn.execute("SET threads=4")  # Reasonable default for parquet reading
            
        # Apply any additional connection config
        for key, value in self.connection_config.items():
            conn.execute(f"SET {key}={value}")
            
        return conn

    def _extract_hive_partitions(self, file_path: str) -> Dict[str, str]:
        """
        Extract hive partition information from file path.
        
        Args:
            file_path: Path containing hive partitions like 'cik=123/filing_type=10K/year=2023'
            
        Returns:
            Dictionary of partition key-value pairs
        """
        partitions = {}
        
        # Match hive partition pattern: key=value
        partition_pattern = r'([^/=]+)=([^/]+)'
        matches = re.findall(partition_pattern, file_path)
        
        for key, value in matches:
            partitions[key] = value
            
        return partitions

    def _build_document_id(self, query_name: str, row_data: Dict[str, Any], 
                          partitions: Dict[str, str]) -> str:
        """
        Build a unique document ID from query name, row data, and partitions.
        
        Args:
            query_name: Name of the query that generated this document
            row_data: Row data from query result
            partitions: Hive partition data
            
        Returns:
            Unique document identifier
        """
        query_config = next(q for q in self.queries if q["name"] == query_name)
        # Use id_columns (normalized in validation)
        id_columns = query_config.get("id_columns", query_config.get("doc_id_columns", []))
        
        # Collect ID components from row data and partitions
        id_components = [query_name]
        
        for col in id_columns:
            if col in row_data:
                id_components.append(f"{col}={row_data[col]}")
            elif col in partitions:
                id_components.append(f"{col}={partitions[col]}")
            else:
                logger.warning(f"ID column '{col}' not found in row data or partitions for query '{query_name}'")
                
        return "/".join(id_components)

    def fetch_document(self, source_id: str) -> Dict[str, Any]:
        """
        Fetch document content by re-executing the appropriate query with filters.

        Args:
            source_id: Document identifier created by _build_document_id

        Returns:
            Dictionary containing document content and metadata

        Raises:
            ValueError: If document not found or source_id is invalid
        """
        # Parse source_id to extract query name and filters
        parts = source_id.split("/")
        if len(parts) < 2:
            raise ValueError(f"Invalid source_id format: {source_id}")
            
        query_name = parts[0]
        
        # Find the query configuration
        query_config = None
        for q in self.queries:
            if q["name"] == query_name:
                query_config = q
                break
                
        if not query_config:
            raise ValueError(f"Unknown query name in source_id: {query_name}")
        
        # Extract filter conditions from source_id
        filters = {}
        for part in parts[1:]:
            if "=" in part:
                key, value = part.split("=", 1)
                filters[key] = value
        
        # Build filtered query
        base_sql = query_config["sql"]
        where_conditions = []
        
        for key, value in filters.items():
            # Handle different data types appropriately
            if value.isdigit():
                where_conditions.append(f"{key} = {value}")
            else:
                where_conditions.append(f"{key} = '{value}'")
        
        if where_conditions:
            # Add WHERE clause or extend existing one
            if "WHERE" in base_sql.upper():
                filtered_sql = f"SELECT * FROM ({base_sql}) sub WHERE {' AND '.join(where_conditions)}"
            else:
                filtered_sql = f"{base_sql} WHERE {' AND '.join(where_conditions)}"
        else:
            filtered_sql = base_sql
            
        try:
            conn = self._get_connection()
            result = conn.execute(filtered_sql).fetchall()
            columns = [desc[0] for desc in conn.description]
            conn.close()
            
            if not result:
                raise ValueError(f"Document not found: {source_id}")
                
            if len(result) > 1:
                logger.warning(f"Multiple rows found for {source_id}, using first row")
                
            # Convert result to dictionary
            row_data = dict(zip(columns, result[0]))
            
            # Extract content
            content_column = query_config["content_column"]
            if content_column not in row_data:
                raise ValueError(f"Content column '{content_column}' not found in query result")
                
            content = str(row_data[content_column] or "")
            
            # Build metadata based on metadata_columns if specified
            metadata = {}
            metadata_columns = query_config.get("metadata_columns")
            
            if metadata_columns:
                # Only include specified metadata columns
                for col in metadata_columns:
                    if col in row_data and row_data[col] is not None:
                        metadata[col] = row_data[col]
            else:
                # Include all non-content columns as metadata (backward compatibility)
                for key, value in row_data.items():
                    if key != content_column and value is not None:
                        metadata[key] = value
            
            # Extract and add hive partitions from any path-like columns
            partitions = {}
            for key, value in row_data.items():
                if isinstance(value, str) and "/" in value:
                    file_partitions = self._extract_hive_partitions(value)
                    partitions.update(file_partitions)
            
            # Add partitions to metadata
            metadata.update(partitions)
            
            # Add system metadata
            metadata["query_name"] = query_name
            metadata["source_type"] = "duckdb"
            
            return {
                "id": source_id,
                "content": content,
                "metadata": metadata,
                "content_hash": self.get_content_hash(content),
                "doc_type": query_config.get("doc_type", "text")
            }
            
        except Exception as e:
            logger.error(f"Error fetching document {source_id}: {str(e)}")
            raise

    def list_documents(self) -> List[Dict[str, Any]]:
        """
        List available documents by executing all configured queries.

        Returns:
            List of document identifiers and metadata
        """
        results = []
        
        for query_config in self.queries:
            query_name = query_config["name"]
            sql = query_config["sql"]
            content_column = query_config["content_column"]
            doc_type = query_config.get("doc_type", "text")
            
            try:
                conn = self._get_connection()
                query_results = conn.execute(sql).fetchall()
                columns = [desc[0] for desc in conn.description]
                conn.close()
                
                logger.debug(f"Query '{query_name}' returned {len(query_results)} rows")
                
                for row in query_results:
                    row_data = dict(zip(columns, row))
                    
                    # Skip rows with empty content
                    if not row_data.get(content_column):
                        continue
                    
                    # Extract hive partitions from any path-like columns
                    partitions = {}
                    for key, value in row_data.items():
                        if isinstance(value, str) and "/" in value:
                            file_partitions = self._extract_hive_partitions(value)
                            partitions.update(file_partitions)
                    
                    # Build document ID
                    doc_id = self._build_document_id(query_name, row_data, partitions)
                    
                    # Build metadata based on metadata_columns if specified
                    metadata = {}
                    metadata_columns = query_config.get("metadata_columns")
                    
                    if metadata_columns:
                        # Only include specified metadata columns
                        for col in metadata_columns:
                            if col in row_data and row_data[col] is not None:
                                metadata[col] = row_data[col]
                    else:
                        # Include all non-content columns as metadata (backward compatibility)
                        for key, value in row_data.items():
                            if key != content_column and value is not None:
                                metadata[key] = value
                    
                    # Add partitions and system metadata
                    metadata.update(partitions)
                    metadata["query_name"] = query_name
                    metadata["source_type"] = "duckdb"
                    
                    # Add content size info
                    content_size = len(str(row_data[content_column] or ""))
                    metadata["content_size"] = content_size
                    
                    results.append({
                        "id": doc_id,
                        "metadata": metadata,
                        "doc_type": doc_type
                    })
                    
            except Exception as e:
                logger.error(f"Error executing query '{query_name}': {str(e)}")
                # Continue with other queries
                continue
        
        logger.info(f"Listed {len(results)} documents from {len(self.queries)} queries")
        return results

    def has_changed(self, source_id: str, last_modified: Optional[float] = None) -> bool:
        """
        Check if document has changed.
        
        For DuckDB/parquet sources, we check file modification times if available,
        otherwise assume content may have changed.

        Args:
            source_id: Document identifier
            last_modified: Previous modification timestamp

        Returns:
            True if document may have changed, False otherwise
        """
        # Parse source_id to get query name
        parts = source_id.split("/")
        if len(parts) < 1:
            return True
            
        query_name = parts[0]
        
        # Find the query configuration
        query_config = None
        for q in self.queries:
            if q["name"] == query_name:
                query_config = q
                break
                
        if not query_config:
            return True
        
        # For file-based sources, check if any relevant files have changed
        if os.path.isdir(self.database_path):
            try:
                # Get the most recent modification time from the dataset directory
                latest_mtime = 0
                for root, dirs, files in os.walk(self.database_path):
                    for file in files:
                        if file.endswith('.parquet'):
                            file_path = os.path.join(root, file)
                            mtime = os.path.getmtime(file_path)
                            latest_mtime = max(latest_mtime, mtime)
                
                if last_modified is None:
                    return True
                    
                return latest_mtime > last_modified
                
            except Exception as e:
                logger.warning(f"Error checking file modification times: {str(e)}")
                return True
        
        # For database files or other cases, assume changed if we can't determine
        return True

    def get_query_info(self) -> List[Dict[str, Any]]:
        """
        Get information about configured queries.
        
        Returns:
            List of query information dictionaries
        """
        return [
            {
                "name": q["name"],
                "sql": q["sql"],
                "doc_id_columns": q["doc_id_columns"],
                "content_column": q["content_column"],
                "doc_type": q.get("doc_type", "text"),
                "description": q.get("description", "")
            }
            for q in self.queries
        ]

    def test_connection(self) -> bool:
        """
        Test the DuckDB connection and basic functionality.
        
        Returns:
            True if connection works, False otherwise
        """
        try:
            conn = self._get_connection()
            
            # Test basic query
            result = conn.execute("SELECT 1 as test").fetchone()
            conn.close()
            
            return result[0] == 1
            
        except Exception as e:
            logger.error(f"Connection test failed: {str(e)}")
            return False