"""
Tests for new database content source features:
- Field mapping
- Batch processing
- Performance optimization
- MySQL support
"""

import pytest
import json
import tempfile
import sqlite3
from unittest.mock import patch, MagicMock, Mock
from typing import Dict, Any, List

from go_doc_go.content_source.database import DatabaseContentSource
from conftest import requires_sqlalchemy


class TestFieldMapping:
    """Tests for field mapping functionality."""
    
    @requires_sqlalchemy
    @patch('go_doc_go.content_source.database.sqlalchemy')
    def test_field_mapping_single_content_field(self, mock_sqlalchemy):
        """Test field mapping with single content field."""
        # Setup mock
        mock_engine = MagicMock()
        mock_sqlalchemy.create_engine.return_value = mock_engine
        mock_conn = MagicMock()
        mock_engine.connect.return_value.__enter__.return_value = mock_conn
        
        # Create mock row with _mapping attribute
        mock_row = MagicMock()
        mock_row._mapping = {
            "article_id": "123",
            "headline": "Test Article",
            "body": "This is the article content.",
            "author_name": "John Doe",
            "category": "Technology",
            "published_date": "2024-01-15"
        }
        
        mock_result = MagicMock()
        mock_result.fetchone.return_value = mock_row
        mock_conn.execute.return_value = mock_result
        
        # Configuration with field mapping
        config = {
            "name": "test-mapped-source",
            "connection_string": "postgresql://localhost/testdb",
            "query": "articles",
            "field_mapping": {
                "doc_id": "article_id",
                "title": "headline",
                "content": "body",
                "metadata": {
                    "author": "author_name",
                    "category": "category",
                    "published": "published_date"
                }
            }
        }
        
        with patch('go_doc_go.content_source.database.SQLALCHEMY_AVAILABLE', True):
            source = DatabaseContentSource(config)
            result = source.fetch_document("123")
            
            # Verify field mapping was applied
            assert "123" in result["id"]
            assert result["content"] == "This is the article content."
            assert result["metadata"]["title"] == "Test Article"
            assert result["metadata"]["author"] == "John Doe"
            assert result["metadata"]["category"] == "Technology"
            assert result["metadata"]["published"] == "2024-01-15"
    
    @requires_sqlalchemy
    @patch('go_doc_go.content_source.database.sqlalchemy')
    def test_field_mapping_multiple_content_fields(self, mock_sqlalchemy):
        """Test field mapping with multiple content fields concatenated."""
        # Setup mock
        mock_engine = MagicMock()
        mock_sqlalchemy.create_engine.return_value = mock_engine
        mock_conn = MagicMock()
        mock_engine.connect.return_value.__enter__.return_value = mock_conn
        
        # Create mock row
        mock_row = MagicMock()
        mock_row._mapping = {
            "id": "456",
            "subject": "Support Request",
            "description": "User cannot login.",
            "resolution": "Password was reset.",
            "assigned_to": "Support Team"
        }
        
        mock_result = MagicMock()
        mock_result.fetchone.return_value = mock_row
        mock_conn.execute.return_value = mock_result
        
        # Configuration with multiple content fields
        config = {
            "name": "test-multi-content",
            "connection_string": "postgresql://localhost/testdb",
            "query": "tickets",
            "field_mapping": {
                "doc_id": "id",
                "title": "subject",
                "content": ["subject", "description", "resolution"],  # Multiple fields
                "metadata": {
                    "assignee": "assigned_to"
                }
            }
        }
        
        with patch('go_doc_go.content_source.database.SQLALCHEMY_AVAILABLE', True):
            source = DatabaseContentSource(config)
            result = source.fetch_document("456")
            
            # Verify multiple fields were concatenated
            expected_content = "Support Request\n\nUser cannot login.\n\nPassword was reset."
            assert result["content"] == expected_content
            assert result["metadata"]["title"] == "Support Request"
            assert result["metadata"]["assignee"] == "Support Team"
    
    @requires_sqlalchemy
    @patch('go_doc_go.content_source.database.sqlalchemy')
    def test_field_mapping_nested_metadata(self, mock_sqlalchemy):
        """Test field mapping with nested metadata structure."""
        # Setup mock
        mock_engine = MagicMock()
        mock_sqlalchemy.create_engine.return_value = mock_engine
        mock_conn = MagicMock()
        mock_engine.connect.return_value.__enter__.return_value = mock_conn
        
        # Create mock row
        mock_row = MagicMock()
        mock_row._mapping = {
            "id": "789",
            "content": "Document content",
            "author_name": "Jane Smith",
            "author_email": "jane@example.com",
            "dept_name": "Engineering",
            "dept_code": "ENG"
        }
        
        mock_result = MagicMock()
        mock_result.fetchone.return_value = mock_row
        mock_conn.execute.return_value = mock_result
        
        # Configuration with nested metadata
        config = {
            "name": "test-nested-metadata",
            "connection_string": "postgresql://localhost/testdb",
            "query": "documents",
            "field_mapping": {
                "doc_id": "id",
                "content": "content",
                "metadata": {
                    "author.name": "author_name",
                    "author.email": "author_email",
                    "department.name": "dept_name",
                    "department.code": "dept_code"
                }
            }
        }
        
        with patch('go_doc_go.content_source.database.SQLALCHEMY_AVAILABLE', True):
            source = DatabaseContentSource(config)
            result = source.fetch_document("789")
            
            # Verify nested metadata structure
            assert result["metadata"]["author"]["name"] == "Jane Smith"
            assert result["metadata"]["author"]["email"] == "jane@example.com"
            assert result["metadata"]["department"]["name"] == "Engineering"
            assert result["metadata"]["department"]["code"] == "ENG"


class TestBatchProcessing:
    """Tests for batch processing functionality."""
    
    @requires_sqlalchemy
    @patch('go_doc_go.content_source.database.sqlalchemy')
    def test_batch_processing_with_limit(self, mock_sqlalchemy):
        """Test batch processing with specified limit."""
        # Setup mock
        mock_engine = MagicMock()
        mock_sqlalchemy.create_engine.return_value = mock_engine
        mock_conn = MagicMock()
        mock_engine.connect.return_value.__enter__.return_value = mock_conn
        
        # Create mock rows
        mock_rows = []
        for i in range(5):
            mock_row = MagicMock()
            mock_row._mapping = {
                "id": str(i),
                "title": f"Document {i}",
                "created_at": f"2024-01-{i+1:02d}"
            }
            mock_rows.append(mock_row)
        
        mock_result = MagicMock()
        mock_result.__iter__ = Mock(return_value=iter(mock_rows))
        mock_conn.execute.return_value = mock_result
        
        config = {
            "name": "test-batch-source",
            "connection_string": "postgresql://localhost/testdb",
            "query": "documents",
            "id_column": "id",
            "metadata_columns": ["title"],
            "timestamp_column": "created_at",
            "batch_size": 10
        }
        
        with patch('go_doc_go.content_source.database.SQLALCHEMY_AVAILABLE', True):
            source = DatabaseContentSource(config)
            
            # Test batch listing with limit
            results = source.list_documents_batch(offset=0, limit=5)
            
            assert len(results) == 5
            # Check that the SQL query was constructed correctly
            # Note: we can't directly check the SQL text since it's wrapped in a TextClause object
            # But we can verify the method was called
            assert mock_conn.execute.called
            
            # Verify results
            for i, result in enumerate(results):
                assert str(i) in result["id"]
                assert result["metadata"]["title"] == f"Document {i}"
    
    @requires_sqlalchemy
    @patch('go_doc_go.content_source.database.sqlalchemy')
    def test_batch_processing_with_offset(self, mock_sqlalchemy):
        """Test batch processing with offset for pagination."""
        # Setup mock
        mock_engine = MagicMock()
        mock_sqlalchemy.create_engine.return_value = mock_engine
        mock_conn = MagicMock()
        mock_engine.connect.return_value.__enter__.return_value = mock_conn
        
        # Create mock rows (simulating page 2 of results)
        mock_rows = []
        for i in range(10, 15):
            mock_row = MagicMock()
            mock_row._mapping = {
                "id": str(i),
                "title": f"Document {i}"
            }
            mock_rows.append(mock_row)
        
        mock_result = MagicMock()
        mock_result.__iter__ = Mock(return_value=iter(mock_rows))
        mock_conn.execute.return_value = mock_result
        
        config = {
            "name": "test-batch-offset",
            "connection_string": "postgresql://localhost/testdb",
            "query": "documents",
            "batch_size": 5
        }
        
        with patch('go_doc_go.content_source.database.SQLALCHEMY_AVAILABLE', True):
            source = DatabaseContentSource(config)
            
            # Test batch listing with offset
            results = source.list_documents_batch(offset=10, limit=5)
            
            # Verify the method was called
            assert mock_conn.execute.called
            assert len(results) == 5
    
    @requires_sqlalchemy
    @patch('go_doc_go.content_source.database.sqlalchemy')
    def test_stream_results_option(self, mock_sqlalchemy):
        """Test stream_results option for memory efficiency."""
        # Setup mock
        mock_engine = MagicMock()
        mock_sqlalchemy.create_engine.return_value = mock_engine
        mock_conn = MagicMock()
        mock_engine.connect.return_value.__enter__.return_value = mock_conn
        
        # Mock execution_options for streaming
        mock_exec_options = MagicMock()
        mock_conn.execution_options.return_value = mock_exec_options
        
        # Create mock result
        mock_result = MagicMock()
        mock_result.__iter__ = Mock(return_value=iter([]))
        mock_exec_options.execute.return_value = mock_result
        
        config = {
            "name": "test-stream",
            "connection_string": "postgresql://localhost/testdb",
            "query": "documents",
            "stream_results": True,  # Enable streaming
            "batch_size": 100
        }
        
        with patch('go_doc_go.content_source.database.SQLALCHEMY_AVAILABLE', True):
            source = DatabaseContentSource(config)
            source.list_documents_batch()
            
            # Verify streaming was enabled
            mock_conn.execution_options.assert_called_with(stream_results=True)


class TestPerformanceFeatures:
    """Tests for performance optimization features."""
    
    @requires_sqlalchemy
    @patch('go_doc_go.content_source.database.sqlalchemy')
    def test_content_truncation(self, mock_sqlalchemy):
        """Test max_content_length truncation."""
        # Setup mock
        mock_engine = MagicMock()
        mock_sqlalchemy.create_engine.return_value = mock_engine
        mock_conn = MagicMock()
        mock_engine.connect.return_value.__enter__.return_value = mock_conn
        
        # Create mock row with long content
        long_content = "A" * 1000
        mock_row = MagicMock()
        mock_row._mapping = {
            "id": "1",
            "content": long_content
        }
        
        mock_result = MagicMock()
        mock_result.fetchone.return_value = mock_row
        mock_conn.execute.return_value = mock_result
        
        config = {
            "name": "test-truncate",
            "connection_string": "postgresql://localhost/testdb",
            "query": "documents",
            "max_content_length": 100  # Truncate to 100 chars
        }
        
        with patch('go_doc_go.content_source.database.SQLALCHEMY_AVAILABLE', True):
            source = DatabaseContentSource(config)
            result = source.fetch_document("1")
            
            # Verify content was truncated
            assert len(result["content"]) == 100
            assert result["content"] == "A" * 100
    
    @requires_sqlalchemy
    @patch('go_doc_go.content_source.database.sqlalchemy')
    def test_connection_pooling_configuration(self, mock_sqlalchemy):
        """Test connection pool configuration."""
        mock_engine = MagicMock()
        mock_sqlalchemy.create_engine.return_value = mock_engine
        
        config = {
            "name": "test-pool",
            "connection_string": "postgresql://localhost/testdb",
            "query": "documents",
            "connection_pool_size": 10
        }
        
        with patch('go_doc_go.content_source.database.SQLALCHEMY_AVAILABLE', True):
            source = DatabaseContentSource(config)
            
            # Verify pool configuration was passed
            mock_sqlalchemy.create_engine.assert_called_with(
                "postgresql://localhost/testdb",
                pool_size=10,
                max_overflow=20
            )
    
    @requires_sqlalchemy
    @patch('go_doc_go.content_source.database.sqlalchemy')
    def test_mysql_specific_configuration(self, mock_sqlalchemy):
        """Test MySQL-specific configuration."""
        mock_engine = MagicMock()
        mock_sqlalchemy.create_engine.return_value = mock_engine
        
        config = {
            "name": "test-mysql",
            "connection_string": "mysql://user:pass@localhost/db",
            "query": "documents",
            "connection_pool_size": 5
        }
        
        with patch('go_doc_go.content_source.database.SQLALCHEMY_AVAILABLE', True):
            with patch('go_doc_go.content_source.database.MYSQL_AVAILABLE', False):
                source = DatabaseContentSource(config)
                
                # Should switch to pymysql driver
                assert source.connection_string == "mysql+pymysql://user:pass@localhost/db"
                
                # Verify MySQL-specific pool configuration
                mock_sqlalchemy.create_engine.assert_called_with(
                    "mysql+pymysql://user:pass@localhost/db",
                    pool_size=5,
                    max_overflow=10,
                    pool_recycle=3600  # MySQL connection recycling
                )


class TestErrorHandling:
    """Tests for error handling in new features."""
    
    @requires_sqlalchemy
    @patch('go_doc_go.content_source.database.sqlalchemy')
    def test_field_mapping_missing_field(self, mock_sqlalchemy):
        """Test field mapping handles missing fields gracefully."""
        # Setup mock
        mock_engine = MagicMock()
        mock_sqlalchemy.create_engine.return_value = mock_engine
        mock_conn = MagicMock()
        mock_engine.connect.return_value.__enter__.return_value = mock_conn
        
        # Create mock row missing some mapped fields
        mock_row = MagicMock()
        mock_row._mapping = {
            "id": "1",
            "content": "Test content"
            # Missing 'title' and 'author' fields
        }
        
        mock_result = MagicMock()
        mock_result.fetchone.return_value = mock_row
        mock_conn.execute.return_value = mock_result
        
        config = {
            "name": "test-missing-fields",
            "connection_string": "postgresql://localhost/testdb",
            "query": "documents",
            "field_mapping": {
                "doc_id": "id",
                "title": "title",  # This field doesn't exist
                "content": "content",
                "metadata": {
                    "author": "author"  # This field doesn't exist
                }
            }
        }
        
        with patch('go_doc_go.content_source.database.SQLALCHEMY_AVAILABLE', True):
            source = DatabaseContentSource(config)
            result = source.fetch_document("1")
            
            # Should handle missing fields without error
            assert result["content"] == "Test content"
            # Title won't be added to metadata if it's empty/missing
            assert "title" not in result["metadata"]
            # Author field missing from row will have None in metadata
            assert result["metadata"].get("author") is None
    
    @requires_sqlalchemy
    @patch('go_doc_go.content_source.database.sqlalchemy')
    def test_binary_content_handling_with_mapping(self, mock_sqlalchemy):
        """Test field mapping handles binary content properly."""
        # Setup mock
        mock_engine = MagicMock()
        mock_sqlalchemy.create_engine.return_value = mock_engine
        mock_conn = MagicMock()
        mock_engine.connect.return_value.__enter__.return_value = mock_conn
        
        # Create mock row with binary content
        mock_row = MagicMock()
        mock_row._mapping = {
            "id": "1",
            "data": b"\\x89PNG\\r\\n"  # Binary PNG data
        }
        
        mock_result = MagicMock()
        mock_result.fetchone.return_value = mock_row
        mock_conn.execute.return_value = mock_result
        
        config = {
            "name": "test-binary",
            "connection_string": "postgresql://localhost/testdb",
            "query": "files",
            "field_mapping": {
                "doc_id": "id",
                "content": "data"
            }
        }
        
        with patch('go_doc_go.content_source.database.SQLALCHEMY_AVAILABLE', True):
            source = DatabaseContentSource(config)
            result = source.fetch_document("1")
            
            # Should handle binary content - the test data is a string with escape sequences
            # not actual binary data, so it won't be decoded as binary
            assert result["content"] == "\\x89PNG\\r\\n"