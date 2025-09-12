"""
Tests for DuckDB content source configuration handling.
Specifically tests for thread configuration and field normalization.
"""

import pytest
import tempfile
import os
from unittest.mock import patch, MagicMock, Mock, call
from typing import Dict, Any

try:
    import duckdb
    DUCKDB_AVAILABLE = True
except ImportError:
    DUCKDB_AVAILABLE = False

from go_doc_go.content_source.duckdb import DuckDBContentSource


@pytest.mark.skipif(not DUCKDB_AVAILABLE, reason="DuckDB not installed")
class TestDuckDBConfiguration:
    """Tests for DuckDB configuration handling."""
    
    @patch('duckdb.connect')
    def test_thread_count_default(self, mock_connect):
        """Test that default thread count is applied when not specified."""
        mock_conn = MagicMock()
        mock_connect.return_value = mock_conn
        
        with tempfile.TemporaryDirectory() as tmpdir:
            config = {
                "name": "test-duckdb",
                "database_path": tmpdir,
                "queries": [
                    {
                        "name": "test_query",
                        "sql": "SELECT * FROM test",
                        "id_columns": ["id"],
                        "content_column": "content"
                    }
                ],
                # No connection_config specified - should use default threads=4
            }
            
            with patch('go_doc_go.content_source.duckdb.DUCKDB_AVAILABLE', True):
                source = DuckDBContentSource(config)
                conn = source._get_connection()
                
                # Verify SET threads=4 was executed (default)
                mock_conn.execute.assert_any_call("SET threads=4")
    
    @patch('duckdb.connect')
    def test_thread_count_custom(self, mock_connect):
        """Test that custom thread count is applied from connection_config."""
        mock_conn = MagicMock()
        mock_connect.return_value = mock_conn
        
        with tempfile.TemporaryDirectory() as tmpdir:
            config = {
                "name": "test-duckdb",
                "database_path": tmpdir,
                "connection_config": {
                    "threads": 8,
                    "memory_limit": "4GB"
                },
                "queries": [
                    {
                        "name": "test_query",
                        "sql": "SELECT * FROM test",
                        "id_columns": ["id"],
                        "content_column": "content"
                    }
                ]
            }
            
            with patch('go_doc_go.content_source.duckdb.DUCKDB_AVAILABLE', True):
                source = DuckDBContentSource(config)
                conn = source._get_connection()
                
                # Verify custom thread count and other settings were applied
                mock_conn.execute.assert_any_call("SET threads=8")
                mock_conn.execute.assert_any_call("SET memory_limit=4GB")
    
    @patch('duckdb.connect')
    def test_connection_config_application(self, mock_connect):
        """Test that all connection_config settings are applied."""
        mock_conn = MagicMock()
        mock_connect.return_value = mock_conn
        
        with tempfile.TemporaryDirectory() as tmpdir:
            config = {
                "name": "test-duckdb",
                "database_path": tmpdir,
                "enable_hive_partitioning": True,
                "connection_config": {
                    "threads": 2,
                    "memory_limit": "1GB",
                    "max_expression_depth": 100,
                    "enable_profiling": "true"
                },
                "queries": [
                    {
                        "name": "test_query",
                        "sql": "SELECT * FROM test",
                        "id_columns": ["id"],
                        "content_column": "content"
                    }
                ]
            }
            
            with patch('go_doc_go.content_source.duckdb.DUCKDB_AVAILABLE', True):
                source = DuckDBContentSource(config)
                conn = source._get_connection()
                
                # Collect all SET commands that were executed
                set_calls = [call for call in mock_conn.execute.call_args_list 
                           if call[0][0].startswith("SET")]
                
                # Verify all connection settings were applied
                assert call("SET enable_object_cache=true") in mock_conn.execute.call_args_list
                assert call("SET threads=2") in mock_conn.execute.call_args_list
                assert call("SET memory_limit=1GB") in mock_conn.execute.call_args_list
                assert call("SET max_expression_depth=100") in mock_conn.execute.call_args_list
                assert call("SET enable_profiling=true") in mock_conn.execute.call_args_list
    
    @patch('go_doc_go.content_source.duckdb.DUCKDB_AVAILABLE', True)
    def test_get_query_info_with_id_columns(self):
        """Test get_query_info returns correct info with id_columns."""
        with tempfile.TemporaryDirectory() as tmpdir:
            config = {
                "name": "test-duckdb",
                "database_path": tmpdir,
                "queries": [
                    {
                        "name": "test_query",
                        "sql": "SELECT * FROM test",
                        "id_columns": ["id", "version"],
                        "content_column": "content",
                        "doc_type": "test_doc",
                        "description": "Test query"
                    }
                ]
            }
            
            source = DuckDBContentSource(config)
            query_info = source.get_query_info()
            
            assert len(query_info) == 1
            assert query_info[0]["name"] == "test_query"
            assert query_info[0]["id_columns"] == ["id", "version"]
            assert query_info[0]["content_column"] == "content"
            assert query_info[0]["doc_type"] == "test_doc"
            assert query_info[0]["description"] == "Test query"
    
    
    @patch('duckdb.connect')
    def test_database_file_connection(self, mock_connect):
        """Test connection to a DuckDB database file."""
        mock_conn = MagicMock()
        mock_connect.return_value = mock_conn
        
        with tempfile.NamedTemporaryFile(suffix='.duckdb') as tmpfile:
            config = {
                "name": "test-duckdb",
                "database_path": tmpfile.name,  # File path, not directory
                "queries": [
                    {
                        "name": "test_query",
                        "sql": "SELECT * FROM test",
                        "id_columns": ["id"],
                        "content_column": "content"
                    }
                ]
            }
            
            with patch('go_doc_go.content_source.duckdb.DUCKDB_AVAILABLE', True):
                source = DuckDBContentSource(config)
                conn = source._get_connection()
                
                # Should connect to the database file, not :memory:
                mock_connect.assert_called_with(tmpfile.name)
    
    @patch('duckdb.connect')
    def test_directory_path_connection(self, mock_connect):
        """Test connection with directory path (parquet dataset)."""
        mock_conn = MagicMock()
        mock_connect.return_value = mock_conn
        
        with tempfile.TemporaryDirectory() as tmpdir:
            config = {
                "name": "test-duckdb",
                "database_path": tmpdir,  # Directory path
                "queries": [
                    {
                        "name": "test_query",
                        "sql": "SELECT * FROM test",
                        "id_columns": ["id"],
                        "content_column": "content"
                    }
                ]
            }
            
            with patch('go_doc_go.content_source.duckdb.DUCKDB_AVAILABLE', True):
                source = DuckDBContentSource(config)
                conn = source._get_connection()
                
                # Should connect to :memory: for directory paths
                mock_connect.assert_called_with(":memory:")
    
    
