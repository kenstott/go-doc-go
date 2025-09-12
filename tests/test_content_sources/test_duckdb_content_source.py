"""
Tests for DuckDB Content Source.
"""

import os
import tempfile
import json
import pytest
from pathlib import Path
from unittest.mock import patch, MagicMock

try:
    import duckdb
    DUCKDB_AVAILABLE = True
except ImportError:
    DUCKDB_AVAILABLE = False

from go_doc_go.content_source.duckdb import DuckDBContentSource


@pytest.fixture
def sample_parquet_dir():
    """Create a temporary directory with sample parquet structure."""
    with tempfile.TemporaryDirectory() as temp_dir:
        # Create hive-partitioned structure
        # cik=123/filing_type=10K/year=2023/mda.parquet
        parquet_path = Path(temp_dir) / "cik=0000123456" / "filing_type=10K" / "year=2023"
        parquet_path.mkdir(parents=True, exist_ok=True)
        
        yield temp_dir


@pytest.fixture
def basic_config(sample_parquet_dir):
    """Basic DuckDB configuration for testing."""
    return {
        "name": "test-duckdb",
        "type": "duckdb",
        "database_path": sample_parquet_dir,
        "queries": [
            {
                "name": "test-query",
                "sql": "SELECT 'Test content' as text, 'AAPL' as ticker, '10K' as filing_type, 2023 as year",
                "id_columns": ["ticker", "filing_type", "year"],
                "content_column": "text",
                "metadata_columns": ["ticker", "filing_type", "year"],
                "doc_type": "text"
            }
        ]
    }


class TestDuckDBContentSource:
    """Unit tests for DuckDB content source."""

    def test_initialization_without_duckdb(self):
        """Test that initialization fails gracefully without DuckDB."""
        with patch('go_doc_go.content_source.duckdb.DUCKDB_AVAILABLE', False):
            with pytest.raises(ImportError, match="DuckDB is required"):
                DuckDBContentSource({
                    "database_path": "/tmp",
                    "queries": [{"name": "test", "sql": "SELECT 1", "id_columns": ["id"], "content_column": "content"}]
                })

    @pytest.mark.skipif(not DUCKDB_AVAILABLE, reason="DuckDB not available")
    def test_initialization_missing_database_path(self):
        """Test initialization fails without database_path."""
        with pytest.raises(ValueError, match="database_path is required"):
            DuckDBContentSource({
                "queries": [{"name": "test", "sql": "SELECT 1", "id_columns": ["id"], "content_column": "content"}]
            })

    @pytest.mark.skipif(not DUCKDB_AVAILABLE, reason="DuckDB not available")
    def test_initialization_missing_queries(self):
        """Test initialization fails without queries."""
        with pytest.raises(ValueError, match="At least one query must be configured"):
            DuckDBContentSource({"database_path": "/tmp"})

    @pytest.mark.skipif(not DUCKDB_AVAILABLE, reason="DuckDB not available")
    def test_initialization_invalid_queries(self):
        """Test initialization fails with invalid query configurations."""
        # Missing 'name'
        with pytest.raises(ValueError, match="missing required 'name' field"):
            DuckDBContentSource({
                "database_path": "/tmp",
                "queries": [{"sql": "SELECT 1", "id_columns": ["id"], "content_column": "content"}]
            })

        # Missing 'sql'
        with pytest.raises(ValueError, match="missing required 'sql' field"):
            DuckDBContentSource({
                "database_path": "/tmp",
                "queries": [{"name": "test", "doc_id_columns": ["id"], "content_column": "content"}]
            })

        # Missing 'id_columns'
        with pytest.raises(ValueError, match="missing required 'id_columns' or 'doc_id_columns' field"):
            DuckDBContentSource({
                "database_path": "/tmp",
                "queries": [{"name": "test", "sql": "SELECT 1", "content_column": "content"}]
            })

        # Missing 'content_column'
        with pytest.raises(ValueError, match="missing required 'content_column' field"):
            DuckDBContentSource({
                "database_path": "/tmp",
                "queries": [{"name": "test", "sql": "SELECT 1", "id_columns": ["id"]}]
            })

    @pytest.mark.skipif(not DUCKDB_AVAILABLE, reason="DuckDB not available")
    def test_successful_initialization(self, basic_config):
        """Test successful initialization with valid configuration."""
        source = DuckDBContentSource(basic_config)
        assert source.database_path == basic_config["database_path"]
        assert len(source.queries) == 1
        assert source.queries[0]["name"] == "test-query"

    @pytest.mark.skipif(not DUCKDB_AVAILABLE, reason="DuckDB not available")
    def test_extract_hive_partitions(self, basic_config):
        """Test hive partition extraction from file paths."""
        source = DuckDBContentSource(basic_config)
        
        # Test basic hive partitions
        partitions = source._extract_hive_partitions("cik=123456/filing_type=10K/year=2023/mda.parquet")
        expected = {"cik": "123456", "filing_type": "10K", "year": "2023"}
        assert partitions == expected
        
        # Test with no partitions
        partitions = source._extract_hive_partitions("simple/path/file.parquet")
        assert partitions == {}
        
        # Test with mixed separators
        partitions = source._extract_hive_partitions("company=AAPL/quarter=Q1/year=2023/earnings.parquet")
        expected = {"company": "AAPL", "quarter": "Q1", "year": "2023"}
        assert partitions == expected

    @pytest.mark.skipif(not DUCKDB_AVAILABLE, reason="DuckDB not available")
    def test_build_document_id(self, basic_config):
        """Test document ID building."""
        source = DuckDBContentSource(basic_config)
        
        row_data = {"ticker": "AAPL", "filing_type": "10K", "year": 2023}
        partitions = {"cik": "123456"}
        
        doc_id = source._build_document_id("test-query", row_data, partitions)
        assert doc_id == "test-query/ticker=AAPL/filing_type=10K/year=2023"

    @pytest.mark.skipif(not DUCKDB_AVAILABLE, reason="DuckDB not available")
    def test_connection_test(self, basic_config):
        """Test connection functionality."""
        source = DuckDBContentSource(basic_config)
        
        # Test connection should work with in-memory database
        assert source.test_connection() is True

    @pytest.mark.skipif(not DUCKDB_AVAILABLE, reason="DuckDB not available")
    def test_get_query_info(self, basic_config):
        """Test query information retrieval."""
        source = DuckDBContentSource(basic_config)
        
        query_info = source.get_query_info()
        assert len(query_info) == 1
        assert query_info[0]["name"] == "test-query"
        assert "sql" in query_info[0]
        # The source now normalizes to id_columns internally
        assert "id_columns" in query_info[0] or "doc_id_columns" in query_info[0]
        columns = query_info[0].get("id_columns", query_info[0].get("doc_id_columns", []))
        assert columns == ["ticker", "filing_type", "year"]


@pytest.mark.skipif(not DUCKDB_AVAILABLE, reason="DuckDB not available")
class TestDuckDBContentSourceIntegration:
    """Integration tests that actually execute DuckDB queries."""

    def test_list_documents_simple_query(self, basic_config):
        """Test listing documents with a simple in-memory query."""
        source = DuckDBContentSource(basic_config)
        
        documents = source.list_documents()
        assert len(documents) == 1
        
        doc = documents[0]
        assert doc["id"] == "test-query/ticker=AAPL/filing_type=10K/year=2023"
        assert doc["doc_type"] == "text"
        assert doc["metadata"]["ticker"] == "AAPL"
        assert doc["metadata"]["filing_type"] == "10K"
        assert doc["metadata"]["year"] == 2023
        assert doc["metadata"]["query_name"] == "test-query"

    def test_fetch_document_simple_query(self, basic_config):
        """Test fetching a document with a simple query."""
        source = DuckDBContentSource(basic_config)
        
        # First list to get the document ID
        documents = source.list_documents()
        doc_id = documents[0]["id"]
        
        # Now fetch the specific document
        document = source.fetch_document(doc_id)
        
        assert document["id"] == doc_id
        assert document["content"] == "Test content"
        assert document["metadata"]["ticker"] == "AAPL"
        assert document["metadata"]["filing_type"] == "10K"
        assert document["metadata"]["year"] == 2023
        assert "content_hash" in document

    def test_fetch_nonexistent_document(self, basic_config):
        """Test fetching a non-existent document raises ValueError."""
        source = DuckDBContentSource(basic_config)
        
        with pytest.raises(ValueError, match="Document not found"):
            source.fetch_document("test-query/ticker=INVALID/filing_type=10K/year=2023")

    def test_has_changed_directory_based(self, basic_config):
        """Test change detection for directory-based sources."""
        source = DuckDBContentSource(basic_config)
        
        # Should return True for new documents (no last_modified)
        assert source.has_changed("test-query/ticker=AAPL/filing_type=10K/year=2023") is True
        
        # Should return False when no parquet files exist and we have a recent timestamp
        import time
        last_modified = time.time()
        # The temp directory has no parquet files, so latest_mtime will be 0
        # which is less than the current timestamp
        assert source.has_changed("test-query/ticker=AAPL/filing_type=10K/year=2023", last_modified) is False


    def test_backward_compatibility_doc_id_columns(self):
        """Test that doc_id_columns still works for backward compatibility."""
        config = {
            "name": "test-duckdb",
            "type": "duckdb",
            "database_path": tempfile.mkdtemp(),
            "queries": [
                {
                    "name": "test-query",
                    "sql": "SELECT 'Test' as content, 1 as id",
                    "doc_id_columns": ["id"],  # Using old field name
                    "content_column": "content",
                    "doc_type": "text"
                }
            ]
        }
        
        # Should work without errors
        source = DuckDBContentSource(config)
        assert source.queries[0]["id_columns"] == ["id"]
        
        documents = source.list_documents()
        assert len(documents) == 1


class TestDuckDBContentSourceErrors:
    """Test error handling in DuckDB content source."""

    @pytest.mark.skipif(not DUCKDB_AVAILABLE, reason="DuckDB not available")
    def test_invalid_source_id_format(self, basic_config):
        """Test handling of invalid source_id format."""
        source = DuckDBContentSource(basic_config)
        
        with pytest.raises(ValueError, match="Invalid source_id format"):
            source.fetch_document("invalid-format")

    @pytest.mark.skipif(not DUCKDB_AVAILABLE, reason="DuckDB not available")  
    def test_unknown_query_name(self, basic_config):
        """Test handling of unknown query name in source_id."""
        source = DuckDBContentSource(basic_config)
        
        with pytest.raises(ValueError, match="Unknown query name"):
            source.fetch_document("unknown-query/ticker=AAPL")