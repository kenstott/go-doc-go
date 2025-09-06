"""
Tests for database content source (SQLAlchemy-based).
"""

import pytest
import json
import time
from unittest.mock import patch, MagicMock
from typing import Dict, Any

from go_doc_go.content_source.database import DatabaseContentSource
from conftest import requires_sqlalchemy, requires_psycopg2, requires_docker


class TestDatabaseContentSourceUnit:
    """Unit tests for DatabaseContentSource without real databases."""
    
    def test_content_source_initialization_without_sqlalchemy(self):
        """Test that content source raises error when SQLAlchemy is not available."""
        config = {
            "name": "test-db-source",
            "connection_string": "sqlite:///test.db",
            "query": "documents",
            "id_column": "id",
            "content_column": "content"
        }
        
        with patch('go_doc_go.content_source.database.SQLALCHEMY_AVAILABLE', False):
            with pytest.raises(ImportError, match="SQLAlchemy is required"):
                DatabaseContentSource(config)
    
    @requires_sqlalchemy
    @patch('go_doc_go.content_source.database.sqlalchemy')
    def test_content_source_initialization_with_config(self, mock_sqlalchemy):
        """Test content source initialization with configuration."""
        mock_engine = MagicMock()
        mock_sqlalchemy.create_engine.return_value = mock_engine
        
        config = {
            "name": "test-db-source",
            "connection_string": "postgresql://user:pass@localhost:5432/testdb",
            "query": "documents",
            "id_column": "id",
            "content_column": "content",
            "metadata_columns": ["title", "doc_type"],
            "timestamp_column": "created_at",
            "json_mode": False
        }
        
        with patch('go_doc_go.content_source.database.SQLALCHEMY_AVAILABLE', True):
            source = DatabaseContentSource(config)
            
            assert source.connection_string == config["connection_string"]
            assert source.query == config["query"]
            assert source.id_column == config["id_column"]
            assert source.content_column == config["content_column"]
            assert source.metadata_columns == config["metadata_columns"]
            assert source.timestamp_column == config["timestamp_column"]
            assert source.json_mode == config["json_mode"]
            assert source.engine == mock_engine
    
    @requires_sqlalchemy
    @patch('go_doc_go.content_source.database.sqlalchemy')
    def test_json_mode_initialization(self, mock_sqlalchemy):
        """Test initialization with JSON mode enabled."""
        mock_engine = MagicMock()
        mock_sqlalchemy.create_engine.return_value = mock_engine
        
        config = {
            "name": "test-db-json-source",
            "connection_string": "sqlite:///test.db",
            "query": "json_records",
            "id_column": "id",
            "json_mode": True,
            "json_columns": ["name", "data", "config"],
            "json_include_metadata": True
        }
        
        with patch('go_doc_go.content_source.database.SQLALCHEMY_AVAILABLE', True):
            source = DatabaseContentSource(config)
            
            assert source.json_mode is True
            assert source.json_columns == config["json_columns"]
            assert source.json_include_metadata is True
    
    def test_get_safe_connection_string(self):
        """Test safe connection string generation."""
        config = {
            "name": "test-source",
            "connection_string": "postgresql://testuser:secretpass@localhost:5432/testdb",
            "query": "documents",
            "id_column": "id",
            "content_column": "content"
        }
        
        with patch('go_doc_go.content_source.database.SQLALCHEMY_AVAILABLE', True):
            with patch('go_doc_go.content_source.database.sqlalchemy.create_engine'):
                source = DatabaseContentSource(config)
                safe_string = source.get_safe_connection_string()
                
                assert "testuser" in safe_string
                assert "secretpass" not in safe_string
                assert "****" in safe_string
                assert "localhost" in safe_string
                assert "testdb" in safe_string


@requires_sqlalchemy
class TestDatabaseContentSourceSQLiteIntegration:
    """Integration tests for DatabaseContentSource with SQLite."""
    
    def test_fetch_blob_document_from_sqlite(self, sqlite_engine, sample_database_configs):
        """Test fetching blob document from SQLite using SQLAlchemy."""
        config = sample_database_configs["sqlite_blob"].copy()
        config["connection_string"] = str(sqlite_engine.url)
        
        with patch('go_doc_go.content_source.database.sqlalchemy.create_engine') as mock_create:
            mock_create.return_value = sqlite_engine
            source = DatabaseContentSource(config)
            source.engine = sqlite_engine  # Override engine
            
            # Fetch document
            result = source.fetch_document("1")
            
            assert result["content"] == "This is the content of document 1."
            assert "id" in result
            assert result["id"].startswith("db://")
            assert "content_hash" in result
            assert result["metadata"]["title"] == "Sample Document 1"
            assert result["metadata"]["doc_type"] == "markdown"
    
    def test_fetch_json_document_from_sqlite(self, sqlite_engine, sample_database_configs):
        """Test fetching JSON document from SQLite."""
        config = sample_database_configs["sqlite_json"].copy()
        config["connection_string"] = str(sqlite_engine.url)
        
        with patch('go_doc_go.content_source.database.sqlalchemy.create_engine') as mock_create:
            mock_create.return_value = sqlite_engine
            source = DatabaseContentSource(config)
            source.engine = sqlite_engine
            
            # Fetch JSON document
            result = source.fetch_document("1")
            
            # Verify JSON content
            json_content = json.loads(result["content"])
            assert json_content["name"] == "Record 1"
            assert json_content["description"] == "First test record"
            assert json_content["data_field"] == "data_value_1"
            assert json_content["status"] == "active"
            
            assert result["content_type"] == "application/json"
            assert "id" in result
            assert result["id"].endswith("/json")
    
    def test_list_documents_sqlite(self, sqlite_engine, sample_database_configs):
        """Test listing documents from SQLite."""
        config = sample_database_configs["sqlite_blob"].copy()
        config["connection_string"] = str(sqlite_engine.url)
        
        with patch('go_doc_go.content_source.database.sqlalchemy.create_engine') as mock_create:
            mock_create.return_value = sqlite_engine
            source = DatabaseContentSource(config)
            source.engine = sqlite_engine
            
            # List documents
            documents = source.list_documents()
            
            assert len(documents) == 5  # Should have 5 test documents
            
            # Check document structure
            for doc in documents:
                assert "id" in doc
                assert doc["id"].startswith("db://")
                assert "metadata" in doc
                
                if "title" in doc["metadata"]:
                    assert doc["metadata"]["title"] is not None
    
    def test_has_changed_detection_sqlite(self, sqlite_engine, sample_database_configs):
        """Test change detection for SQLite documents."""
        config = sample_database_configs["sqlite_blob"].copy()
        config["connection_string"] = str(sqlite_engine.url)
        
        with patch('go_doc_go.content_source.database.sqlalchemy.create_engine') as mock_create:
            mock_create.return_value = sqlite_engine
            source = DatabaseContentSource(config)
            source.engine = sqlite_engine
            
            # Get current timestamp for comparison
            current_time = time.time()
            
            # Test with old timestamp - should indicate change
            db_source = f"db://test/documents/id/1/content"
            has_changed = source.has_changed(db_source, current_time - 3600)  # 1 hour ago
            assert has_changed is True
            
            # Test with future timestamp - should indicate no change
            has_changed = source.has_changed(db_source, current_time + 3600)  # 1 hour in future
            assert has_changed is False
    
    def test_fetch_document_not_found_sqlite(self, sqlite_engine, sample_database_configs):
        """Test error handling when document is not found."""
        config = sample_database_configs["sqlite_blob"].copy()
        config["connection_string"] = str(sqlite_engine.url)
        
        with patch('go_doc_go.content_source.database.sqlalchemy.create_engine') as mock_create:
            mock_create.return_value = sqlite_engine
            source = DatabaseContentSource(config)
            source.engine = sqlite_engine
            
            with pytest.raises(ValueError, match="Document not found"):
                source.fetch_document("999")
    
    def test_binary_content_handling_sqlite(self, sqlite_engine, sample_database_configs):
        """Test handling of binary content in SQLite."""
        # Use a custom configuration for binary docs table
        config = {
            "name": "test-binary-source",
            "connection_string": str(sqlite_engine.url),
            "query": "binary_docs",
            "id_column": "id",
            "content_column": "content",
            "metadata_columns": ["filename", "content_type", "size"]
        }
        
        with patch('go_doc_go.content_source.database.sqlalchemy.create_engine') as mock_create:
            mock_create.return_value = sqlite_engine
            source = DatabaseContentSource(config)
            source.engine = sqlite_engine
            
            # Fetch binary document - should handle gracefully
            result = source.fetch_document("1")
            
            # Binary content should be handled properly
            assert "content" in result
            assert result["metadata"]["filename"] == "test.png"
            assert result["metadata"]["content_type"] == "image/png"


class TestDatabaseContentSourceErrorHandling:
    """Tests for error handling and edge cases."""
    
    @requires_sqlalchemy
    def test_database_not_configured_error(self):
        """Test error when database engine creation fails."""
        config = {
            "name": "test-source",
            "connection_string": "invalid://connection/string",
            "query": "documents",
            "id_column": "id",
            "content_column": "content"
        }
        
        with patch('go_doc_go.content_source.database.SQLALCHEMY_AVAILABLE', True):
            with patch('go_doc_go.content_source.database.sqlalchemy.create_engine') as mock_create:
                mock_create.side_effect = Exception("Connection failed")
                
                with pytest.raises(Exception, match="Connection failed"):
                    source = DatabaseContentSource(config)
    
    @requires_sqlalchemy
    def test_fetch_document_no_engine(self, sample_database_configs):
        """Test fetch_document when engine is not configured."""
        config = sample_database_configs["sqlite_blob"]
        
        with patch('go_doc_go.content_source.database.SQLALCHEMY_AVAILABLE', True):
            with patch('go_doc_go.content_source.database.sqlalchemy.create_engine'):
                source = DatabaseContentSource(config)
                source.engine = None  # Simulate failed initialization
                
                with pytest.raises(ValueError, match="Database not configured"):
                    source.fetch_document("1")
    
    @requires_sqlalchemy
    def test_invalid_json_serialization_handling(self, sqlite_engine):
        """Test handling of objects that can't be JSON serialized."""
        config = {
            "name": "test-source",
            "connection_string": str(sqlite_engine.url),
            "query": "documents", 
            "id_column": "id",
            "json_mode": True,
            "json_columns": ["id", "title", "created_at"]  # created_at might not be JSON serializable
        }
        
        with patch('go_doc_go.content_source.database.sqlalchemy.create_engine') as mock_create:
            mock_create.return_value = sqlite_engine
            source = DatabaseContentSource(config)
            source.engine = sqlite_engine
            
            # Should handle non-serializable objects gracefully
            result = source.fetch_document("1")
            json_content = json.loads(result["content"])
            
            # created_at should be converted to string
            assert "created_at" in json_content
            assert isinstance(json_content["created_at"], str)
    
    @requires_sqlalchemy 
    def test_malformed_connection_string_safety(self):
        """Test safe handling of malformed connection strings."""
        config = {
            "name": "test-source",
            "connection_string": "not-a-valid-connection-string",
            "query": "documents",
            "id_column": "id",
            "content_column": "content"
        }
        
        with patch('go_doc_go.content_source.database.SQLALCHEMY_AVAILABLE', True):
            with patch('go_doc_go.content_source.database.sqlalchemy.create_engine'):
                source = DatabaseContentSource(config)
                safe_string = source.get_safe_connection_string()
                
                # Should not crash and should not expose sensitive data
                assert isinstance(safe_string, str)
                assert safe_string != config["connection_string"]