"""
Database adapter module for the document pointer system.

This module provides an adapter to retrieve content from database sources.
"""

import logging
import re
import sqlite3
from typing import Dict, Any, Optional, Union

from .base import ContentSourceAdapter
from ..document_parser.document_type_detector import DocumentTypeDetector

logger = logging.getLogger(__name__)


class DatabaseAdapter(ContentSourceAdapter):
    """Adapter for database blob content."""

    def __init__(self, config: Optional[Dict[str, Any]] = None):
        """Initialize the database adapter."""
        super().__init__(config)
        self.config = config or {}
        self.connections = {}  # Cache for database connections

    def get_content(self, location_data: Dict[str, Any]) -> Dict[str, Any]:
        """
        Get content from a database.

        Args:
            location_data: Location data with database connection info

        Returns:
            Dictionary with content and metadata

        Raises:
            ValueError: If source is invalid or record not found
        """
        source = location_data.get("source", "")

        # Parse database connection info from source
        db_info = self._parse_db_source(source)
        if not db_info:
            raise ValueError(f"Invalid database source: {source}")

        # Get database connection
        conn = self._get_connection(db_info)

        # Extract content based on location data
        content = self._fetch_record(conn, db_info)
        if content is None:
            raise ValueError(f"Content not found for {source}")

        # Determine the content type
        content_type = DocumentTypeDetector.detect_from_content(
            content,
            metadata={
                "content_column": db_info.get("content_column", ""),
                "content_type": db_info.get("content_type", "")
            }
        )

        # Return content with metadata
        # Mask password in connection string for metadata
        masked_connection = self._mask_password(db_info["connection_id"])
        
        return {
            "content": content,
            "content_type": content_type,
            "metadata": {
                "database": masked_connection,
                "table": db_info["table"],
                "record_id": db_info["pk_value"],
                "content_column": db_info["content_column"]
            }
        }

    def supports_location(self, location_data: Dict[str, Any]) -> bool:
        """
        Check if this adapter supports the location.

        Args:
            location_data: Content location data

        Returns:
            True if supported, False otherwise
        """
        source = location_data.get("source", "")
        # Source must be a database URI
        return source.startswith("db://")

    def get_binary_content(self, location_data: Dict[str, Any]) -> bytes:
        """
        Get the containing document as a binary blob.

        Args:
            location_data: Content location data

        Returns:
            Document binary content

        Raises:
            ValueError: If document cannot be retrieved
        """
        source = location_data.get("source", "")

        # Parse database connection info from source
        db_info = self._parse_db_source(source)
        if not db_info:
            raise ValueError(f"Invalid database source: {source}")

        # Get database connection
        conn = self._get_connection(db_info)

        # Build query - modify to fetch binary content if available
        table = db_info["table"]
        pk_column = db_info["pk_column"]
        pk_value = db_info["pk_value"]
        content_column = db_info["content_column"]

        # Use appropriate query syntax and execution method based on connection type
        if isinstance(conn, sqlite3.Connection):
            query = f"SELECT {content_column} FROM {table} WHERE {pk_column} = ?"
            cursor = conn.execute(query, (pk_value,))
        else:
            # For PostgreSQL and other databases
            query = f"SELECT {content_column} FROM {table} WHERE {pk_column} = %s"
            cursor = conn.cursor()
            cursor.execute(query, (pk_value,))
        
        row = cursor.fetchone()
        
        if not isinstance(conn, sqlite3.Connection):
            cursor.close()

        if row is None:
            raise ValueError(f"Record not found: {pk_value}")

        content = row[0]

        # Handle different data types
        if isinstance(content, bytes):
            return content
        elif isinstance(content, memoryview):
            # PostgreSQL may return binary data as memoryview
            return bytes(content)
        elif hasattr(content, 'tobytes'):
            # Handle buffer-like objects
            return content.tobytes()
        else:
            # Otherwise convert string to bytes
            return content.encode('utf-8')

    @staticmethod
    def _mask_password(connection_string: str) -> str:
        """
        Mask password in connection string for display.
        
        Args:
            connection_string: Database connection string
            
        Returns:
            Connection string with password masked
        """
        import re
        
        # Pattern for user:password@host
        pattern = r'://([^:]+):([^@]+)@'
        match = re.search(pattern, connection_string)
        
        if match:
            user = match.group(1)
            # Replace password with asterisks
            masked = re.sub(pattern, f'://{user}:****@', connection_string)
            return masked
        
        return connection_string
    
    @staticmethod
    def _parse_db_source(source: str) -> Optional[Dict[str, str]]:
        """
        Parse database source URI.

        Format: db://<connection_id>/<table>/<pk_column>/<pk_value>/<content_column>[/<content_type>]
        
        For SQLite files with absolute paths, we need to be smarter about parsing.

        Returns:
            Dictionary with connection info or None if invalid
        """
        if not source.startswith("db://"):
            return None

        # Remove 'db://' prefix
        path = source[5:]

        # Split path
        parts = path.split('/')

        if len(parts) < 5:
            return None

        # Handle different types of connection IDs:
        # 1. Database URIs like postgresql://user:pass@host:port/db
        # 2. SQLite file paths (absolute or relative)  
        # 3. Simple identifiers
        
        connection_id = parts[0]
        remaining_parts = parts[1:]
        
        # Check if this is a database URI (postgresql://, mysql://, etc.)
        # Pattern: ['postgresql:', '', 'user:pass@host:port', 'database', 'table', 'pk', 'value', 'content']
        if connection_id.endswith(':') and len(parts) >= 8 and parts[1] == '':
            # This is a database URI like postgresql://user:pass@host:port/db
            # Reconstruct: postgresql://user:pass@host:port/database
            db_type = connection_id[:-1]  # Remove the ':'
            connection_id = f"{db_type}://{parts[2]}/{parts[3]}"
            remaining_parts = parts[4:]  # Start from table name
        
        # Handle SQLite file paths (absolute paths with slashes)
        elif (not connection_id and len(parts) > 5) or connection_id in ['var', 'tmp', 'Users'] or parts[0] == '':
            # This is likely an absolute path like /var/folders/.../file.db
            # Reconstruct the path until we find a .db file or have enough remaining parts
            path_parts = []
            table_index = None
            
            for i, part in enumerate(parts):
                if part.endswith('.db') or part.endswith('.sqlite'):
                    # Found the database file, everything after this should be table/pk/etc
                    path_parts.append(part)
                    if len(parts) - i - 1 >= 4:  # Need at least table/pk_col/pk_val/content_col
                        table_index = i + 1
                        break
                else:
                    path_parts.append(part)
            
            if table_index is not None:
                connection_id = '/'.join(path_parts)
                remaining_parts = parts[table_index:]
            else:
                # Fallback to original logic
                connection_id = parts[0]
                remaining_parts = parts[1:]
        
        # Now we should have: connection_id and remaining_parts with table/pk_column/pk_value/content_column[/content_type]
        if len(remaining_parts) < 4:
            return None

        # Extract content_type if provided
        content_type = None
        if len(remaining_parts) >= 5:
            content_type = remaining_parts[4]

        return {
            "connection_id": connection_id,
            "table": remaining_parts[0],
            "pk_column": remaining_parts[1],
            "pk_value": remaining_parts[2],
            "content_column": remaining_parts[3],
            "content_type": content_type
        }

    def _get_connection(self, db_info: Dict[str, str]) -> Any:
        """
        Get database connection.

        Args:
            db_info: Database connection info

        Returns:
            Database connection

        Raises:
            ValueError: If connection cannot be established
        """
        connection_id = db_info["connection_id"]

        # Check if connection already exists in cache
        if connection_id in self.connections:
            return self.connections[connection_id]

        # Handle different database types based on connection_id
        if connection_id.endswith('.db') or connection_id.endswith('.sqlite'):
            # Assume SQLite database
            try:
                conn = sqlite3.connect(connection_id)
                conn.row_factory = sqlite3.Row

                # Cache connection
                self.connections[connection_id] = conn
                return conn
            except Exception as e:
                raise ValueError(f"Error connecting to SQLite database {connection_id}: {str(e)}")
        elif connection_id.startswith('postgres://') or connection_id.startswith('postgresql://'):
            # PostgreSQL connection
            try:
                import psycopg2
                import psycopg2.extras

                conn = psycopg2.connect(connection_id)
                conn.cursor_factory = psycopg2.extras.DictCursor

                # Cache connection
                self.connections[connection_id] = conn
                return conn
            except ImportError:
                raise ValueError("psycopg2 is required for PostgreSQL connections")
            except Exception as e:
                raise ValueError(f"Error connecting to PostgreSQL database: {str(e)}")
        elif connection_id.startswith('mysql://'):
            # MySQL connection
            try:
                import mysql.connector

                # Parse connection string
                # Format: mysql://user:password@host:port/database
                conn_parts = re.match(r'mysql://([^:]+):([^@]+)@([^:]+):(\d+)/(.+)', connection_id)
                if not conn_parts:
                    raise ValueError(f"Invalid MySQL connection string: {connection_id}")

                user, password, host, port, database = conn_parts.groups()

                conn = mysql.connector.connect(
                    user=user,
                    password=password,
                    host=host,
                    port=int(port),
                    database=database
                )

                # Cache connection
                self.connections[connection_id] = conn
                return conn
            except ImportError:
                raise ValueError("mysql-connector-python is required for MySQL connections")
            except Exception as e:
                raise ValueError(f"Error connecting to MySQL database: {str(e)}")
        else:
            # Unknown database type
            raise ValueError(f"Unsupported database type for connection: {connection_id}")

    @staticmethod
    def _fetch_record(conn: Any, db_info: Dict[str, str]) -> Union[str, bytes]:
        """
        Fetch content from database.

        Args:
            conn: Database connection
            db_info: Database connection info

        Returns:
            Content as string or bytes

        Raises:
            ValueError: If record cannot be fetched
        """
        table = db_info["table"]
        pk_column = db_info["pk_column"]
        pk_value = db_info["pk_value"]
        content_column = db_info["content_column"]

        # Build query based on database type
        if isinstance(conn, sqlite3.Connection):
            # SQLite query
            query = f"SELECT {content_column} FROM {table} WHERE {pk_column} = ?"
            params = (pk_value,)
        else:
            # Generic SQL query with placeholder
            # This works for PostgreSQL, MySQL, etc.
            query = f"SELECT {content_column} FROM {table} WHERE {pk_column} = %s"
            params = (pk_value,)

        try:
            if isinstance(conn, sqlite3.Connection):
                cursor = conn.execute(query, params)
                row = cursor.fetchone()
            else:
                # For other database types
                cursor = conn.cursor()
                cursor.execute(query, params)
                row = cursor.fetchone()
                
                # Store column descriptions before closing cursor
                columns = None
                if cursor.description and not (isinstance(row, dict) or hasattr(row, 'keys')):
                    columns = [desc[0] for desc in cursor.description]
                
                cursor.close()

            if row is None:
                return None

            # Extract content from row
            if isinstance(conn, sqlite3.Connection):
                content = row[content_column]
            else:
                # For dict-like cursors
                if isinstance(row, dict) or hasattr(row, 'keys'):
                    content = row[content_column]
                else:
                    # For tuple-like cursors, use stored column info
                    if columns:
                        col_idx = columns.index(content_column)
                        content = row[col_idx]
                    else:
                        # Fallback: assume first column is the content
                        content = row[0]

            # Handle binary data
            if isinstance(content, bytes):
                # Try to decode as text if appropriate
                try:
                    # Check if this might be a text blob (e.g., HTML, markdown)
                    if content.startswith(b'<') or content.startswith(b'#'):
                        return content.decode('utf-8')
                    # Otherwise return as binary
                    return content
                except UnicodeDecodeError:
                    # Definitely binary data
                    return content

            return content

        except Exception as e:
            logger.error(f"Error fetching record: {str(e)}")
            raise ValueError(f"Error fetching record: {str(e)}")

    def cleanup(self):
        """Clean up database connections."""
        for conn_id, conn in self.connections.items():
            try:
                conn.close()
                logger.debug(f"Closed database connection: {conn_id}")
            except Exception as e:
                logger.warning(f"Error closing database connection {conn_id}: {str(e)}")

        # Clear connection cache
        self.connections = {}
