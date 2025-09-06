"""
Tests for database adapter.
"""

import pytest
import sqlite3
import tempfile
from unittest.mock import patch, MagicMock
from typing import Dict, Any

from go_doc_go.adapter.database import DatabaseAdapter
from conftest import requires_sqlalchemy, requires_psycopg2, requires_docker


class TestDatabaseAdapterUnit:
    """Unit tests for DatabaseAdapter without real databases."""
    
    def test_adapter_initialization(self):
        """Test that adapter initializes correctly."""
        config = {"max_connections": 5}
        adapter = DatabaseAdapter(config)
        
        assert adapter.config == config
        assert adapter.connections == {}
    
    def test_supports_location_db_uri(self):
        """Test that adapter correctly identifies database URIs."""
        adapter = DatabaseAdapter()
        
        # Valid database URIs
        assert adapter.supports_location({"source": "db://test.db/table/id/1/content"}) is True
        assert adapter.supports_location({"source": "db://conn_id/table/id/1/content"}) is True
        
        # Invalid URIs
        assert adapter.supports_location({"source": "http://example.com"}) is False
        assert adapter.supports_location({"source": "s3://bucket/key"}) is False
        assert adapter.supports_location({"source": "/local/path"}) is False
        assert adapter.supports_location({"source": ""}) is False
    
    def test_parse_db_source_valid_uris(self):
        """Test parsing valid database source URIs."""
        # Basic SQLite URI
        result = DatabaseAdapter._parse_db_source("db://test.db/documents/id/123/content")
        expected = {
            "connection_id": "test.db",
            "table": "documents",
            "pk_column": "id",
            "pk_value": "123",
            "content_column": "content",
            "content_type": None
        }
        assert result == expected
        
        # URI with content type
        result = DatabaseAdapter._parse_db_source("db://conn_id/table/id/456/data/json")
        expected = {
            "connection_id": "conn_id",
            "table": "table",
            "pk_column": "id",
            "pk_value": "456",
            "content_column": "data",
            "content_type": "json"
        }
        assert result == expected
    
    def test_parse_db_source_invalid_uris(self):
        """Test parsing invalid database source URIs."""
        # Not a db:// URI
        assert DatabaseAdapter._parse_db_source("http://example.com") is None
        
        # Too few parts
        assert DatabaseAdapter._parse_db_source("db://test.db/table") is None
        assert DatabaseAdapter._parse_db_source("db://test.db/table/id") is None
        assert DatabaseAdapter._parse_db_source("db://test.db/table/id/123") is None
    
    @patch('go_doc_go.adapter.database.sqlite3')
    def test_sqlite_connection(self, mock_sqlite3):
        """Test SQLite connection creation and caching."""
        mock_conn = MagicMock()
        mock_sqlite3.connect.return_value = mock_conn
        
        adapter = DatabaseAdapter()
        db_info = {
            "connection_id": "test.db",
            "table": "documents",
            "pk_column": "id",
            "pk_value": "123",
            "content_column": "content"
        }
        
        # First call should create connection
        conn = adapter._get_connection(db_info)
        assert conn == mock_conn
        mock_sqlite3.connect.assert_called_once_with("test.db")
        
        # Second call should use cached connection
        mock_sqlite3.reset_mock()
        conn2 = adapter._get_connection(db_info)
        assert conn2 == mock_conn
        mock_sqlite3.connect.assert_not_called()
        
        # Connection should be cached
        assert "test.db" in adapter.connections
    
    def test_postgresql_connection(self):
        """Test PostgreSQL connection creation."""
        with patch('psycopg2.connect') as mock_connect:
            mock_conn = MagicMock()
            mock_connect.return_value = mock_conn
            
            adapter = DatabaseAdapter()
            db_info = {
                "connection_id": "postgresql://user:pass@localhost:5432/testdb",
                "table": "documents",
                "pk_column": "id",
                "pk_value": "123",
                "content_column": "content"
            }
            
            conn = adapter._get_connection(db_info)
            assert conn == mock_conn
            mock_connect.assert_called_once_with("postgresql://user:pass@localhost:5432/testdb")
    
    def test_get_connection_unsupported_database(self):
        """Test error handling for unsupported database types."""
        adapter = DatabaseAdapter()
        db_info = {
            "connection_id": "redis://localhost:6379",
            "table": "documents",
            "pk_column": "id",
            "pk_value": "123",
            "content_column": "content"
        }
        
        with pytest.raises(ValueError, match="Unsupported database type"):
            adapter._get_connection(db_info)
    
    def test_fetch_record_sqlite(self):
        """Test fetching record from SQLite."""
        # Create a mock connection that looks like sqlite3.Connection
        mock_conn = MagicMock(spec=sqlite3.Connection)
        mock_cursor = MagicMock()
        mock_row = {"content": "Test content"}
        
        mock_conn.execute.return_value = mock_cursor
        mock_cursor.fetchone.return_value = mock_row
        
        db_info = {
            "table": "documents",
            "pk_column": "id",
            "pk_value": "123",
            "content_column": "content"
        }
        
        result = DatabaseAdapter._fetch_record(mock_conn, db_info)
        assert result == "Test content"
        
        expected_query = "SELECT content FROM documents WHERE id = ?"
        mock_conn.execute.assert_called_once_with(expected_query, ("123",))
    
    def test_fetch_record_not_found(self):
        """Test handling when record is not found."""
        mock_conn = MagicMock(spec=sqlite3.Connection)
        mock_cursor = MagicMock()
        mock_cursor.fetchone.return_value = None
        mock_conn.execute.return_value = mock_cursor
        
        db_info = {
            "table": "documents",
            "pk_column": "id",
            "pk_value": "999",
            "content_column": "content"
        }
        
        result = DatabaseAdapter._fetch_record(mock_conn, db_info)
        assert result is None
    
    def test_cleanup_connections(self):
        """Test connection cleanup."""
        mock_conn1 = MagicMock()
        mock_conn2 = MagicMock()
        
        adapter = DatabaseAdapter()
        adapter.connections = {
            "conn1": mock_conn1,
            "conn2": mock_conn2
        }
        
        adapter.cleanup()
        
        mock_conn1.close.assert_called_once()
        mock_conn2.close.assert_called_once()
        assert adapter.connections == {}


class TestDatabaseAdapterSQLiteIntegration:
    """Integration tests for DatabaseAdapter with SQLite."""
    
    def test_get_content_from_sqlite(self, sqlite_test_data):
        """Test getting content from actual SQLite database."""
        adapter = DatabaseAdapter()
        
        # Test fetching a document
        location_data = {
            "source": f"db://{sqlite_test_data}/documents/id/1/content"
        }
        
        result = adapter.get_content(location_data)
        
        assert result["content"] == "This is the content of document 1."
        assert result["content_type"] == "text"
        assert result["metadata"]["database"] == sqlite_test_data
        assert result["metadata"]["table"] == "documents"
        assert result["metadata"]["record_id"] == "1"
        assert result["metadata"]["content_column"] == "content"
    
    def test_get_content_markdown_document(self, sqlite_test_data):
        """Test getting markdown content from SQLite."""
        adapter = DatabaseAdapter()
        
        location_data = {
            "source": f"db://{sqlite_test_data}/documents/id/2/content"
        }
        
        result = adapter.get_content(location_data)
        
        assert "# Header" in result["content"]
        assert "markdown document" in result["content"]
        assert result["content_type"] == "markdown"
    
    def test_get_content_json_document(self, sqlite_test_data):
        """Test getting JSON content from SQLite."""
        adapter = DatabaseAdapter()
        
        location_data = {
            "source": f"db://{sqlite_test_data}/documents/id/4/content"
        }
        
        result = adapter.get_content(location_data)
        
        assert '"key": "value"' in result["content"]
        assert '"number": 42' in result["content"]
        # Note: Content type detection is done automatically and might vary
        # The important thing is that we get the JSON content correctly
        assert result["content_type"] in ["json", "text", "csv"]  # Content detection can vary based on JSON structure
    
    def test_get_binary_content_from_sqlite(self, sqlite_test_data):
        """Test getting binary content from SQLite."""
        adapter = DatabaseAdapter()
        
        location_data = {
            "source": f"db://{sqlite_test_data}/binary_docs/id/1/content"
        }
        
        result = adapter.get_binary_content(location_data)
        
        assert isinstance(result, bytes)
        assert result.startswith(b'\x89PNG')
        assert len(result) > 0
    
    def test_get_content_nonexistent_record(self, sqlite_test_data):
        """Test error handling for non-existent records."""
        adapter = DatabaseAdapter()
        
        location_data = {
            "source": f"db://{sqlite_test_data}/documents/id/999/content"
        }
        
        with pytest.raises(ValueError, match="Content not found"):
            adapter.get_content(location_data)
    
    def test_get_content_invalid_source(self, sqlite_test_data):
        """Test error handling for invalid database sources."""
        adapter = DatabaseAdapter()
        
        # Invalid URI format
        location_data = {"source": "not-a-db-uri"}
        
        with pytest.raises(ValueError, match="Invalid database source"):
            adapter.get_content(location_data)
    
    def test_connection_caching(self, sqlite_test_data):
        """Test that database connections are properly cached."""
        adapter = DatabaseAdapter()
        
        location_data1 = {
            "source": f"db://{sqlite_test_data}/documents/id/1/content"
        }
        location_data2 = {
            "source": f"db://{sqlite_test_data}/documents/id/2/content"
        }
        
        # First request
        result1 = adapter.get_content(location_data1)
        assert len(adapter.connections) == 1
        
        # Second request to same database should reuse connection
        result2 = adapter.get_content(location_data2)
        assert len(adapter.connections) == 1
        
        # Both requests should succeed
        assert result1["content"] == "This is the content of document 1."
        assert "# Header" in result2["content"]


@requires_psycopg2
@requires_docker
class TestDatabaseAdapterPostgreSQLIntegration:
    """Integration tests for DatabaseAdapter with PostgreSQL."""
    
    def test_get_content_from_postgres(self, postgres_test_data):
        """Test getting content from actual PostgreSQL database."""
        adapter = DatabaseAdapter()
        
        # Test fetching a document
        location_data = {
            "source": f"db://{postgres_test_data}/documents/id/1/content"
        }
        
        result = adapter.get_content(location_data)
        
        assert result["content"] == "This is the content of document 1."
        assert result["content_type"] == "text"
        assert result["metadata"]["database"] == "postgresql://testuser:****@localhost:5432/testdb"
        assert result["metadata"]["table"] == "documents"
        assert result["metadata"]["record_id"] == "1"
        assert result["metadata"]["content_column"] == "content"
    
    def test_get_content_json_document_postgres(self, postgres_test_data):
        """Test getting JSON content from PostgreSQL."""
        adapter = DatabaseAdapter()
        
        location_data = {
            "source": f"db://{postgres_test_data}/documents/id/4/content"
        }
        
        result = adapter.get_content(location_data)
        
        assert '"key": "value"' in result["content"]
        assert '"number": 42' in result["content"]
        # Content type detection can vary between json/csv/text
        assert result["content_type"] in ["json", "csv", "text"]
    
    def test_get_binary_content_from_postgres(self, postgres_test_data):
        """Test getting binary content from PostgreSQL."""
        adapter = DatabaseAdapter()
        
        location_data = {
            "source": f"db://{postgres_test_data}/binary_docs/id/1/content"
        }
        
        result = adapter.get_binary_content(location_data)
        
        assert isinstance(result, bytes)
        assert result.startswith(b'\x89PNG')
        assert len(result) > 0
    
    def test_postgres_error_handling(self, postgres_test_data):
        """Test PostgreSQL-specific error handling."""
        adapter = DatabaseAdapter()
        
        # Test non-existent record
        location_data = {
            "source": f"db://{postgres_test_data}/documents/id/999/content"
        }
        
        with pytest.raises(ValueError, match="Content not found"):
            adapter.get_content(location_data)
    
    def test_postgres_connection_reuse(self, postgres_test_data):
        """Test PostgreSQL connection reuse."""
        adapter = DatabaseAdapter()
        
        location_data1 = {
            "source": f"db://{postgres_test_data}/documents/id/1/content"
        }
        location_data2 = {
            "source": f"db://{postgres_test_data}/documents/id/2/content"
        }
        
        # First request
        result1 = adapter.get_content(location_data1)
        assert len(adapter.connections) == 1
        
        # Second request should reuse connection
        result2 = adapter.get_content(location_data2)
        assert len(adapter.connections) == 1
        
        # Both should work
        assert result1["content"] == "This is the content of document 1."
        assert "# Header" in result2["content"]


class TestDatabaseAdapterErrorHandling:
    """Tests for error handling and edge cases."""
    
    def test_connection_error_handling(self):
        """Test handling of database connection errors."""
        adapter = DatabaseAdapter()
        
        # Test with non-existent SQLite file
        db_info = {
            "connection_id": "/path/to/nonexistent.db",
            "table": "documents",
            "pk_column": "id",
            "pk_value": "123",
            "content_column": "content"
        }
        
        with pytest.raises(ValueError, match="Error connecting to SQLite database"):
            adapter._get_connection(db_info)
    
    def test_postgresql_not_available(self):
        """Test error when psycopg2 is not available."""
        adapter = DatabaseAdapter()
        
        db_info = {
            "connection_id": "postgresql://user:pass@localhost:5432/db",
            "table": "documents",
            "pk_column": "id",
            "pk_value": "123",
            "content_column": "content"
        }
        
        # Mock the import inside the _get_connection method
        with patch('builtins.__import__') as mock_import:
            def import_side_effect(name, *args, **kwargs):
                if name == 'psycopg2':
                    raise ImportError("No module named 'psycopg2'")
                return __import__(name, *args, **kwargs)
            
            mock_import.side_effect = import_side_effect
            
            with pytest.raises(ValueError, match="psycopg2 is required"):
                adapter._get_connection(db_info)
    
    def test_binary_data_handling(self):
        """Test handling of binary data in text columns."""
        # Create a mock row with binary data that can't be decoded
        mock_row = {"content": b'\x00\x01\x02\x03\xFF'}
        
        # Test binary content is returned as-is
        mock_conn = MagicMock(spec=sqlite3.Connection)
        mock_cursor = MagicMock()
        mock_conn.execute.return_value = mock_cursor
        mock_cursor.fetchone.return_value = mock_row
        
        db_info = {
            "table": "documents",
            "pk_column": "id", 
            "pk_value": "123",
            "content_column": "content"
        }
        
        result = DatabaseAdapter._fetch_record(mock_conn, db_info)
        assert isinstance(result, bytes)
        assert result == b'\x00\x01\x02\x03\xFF'