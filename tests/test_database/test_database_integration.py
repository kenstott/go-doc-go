"""
Integration tests for database adapter and content source with document processing.
"""

import pytest
import json
import tempfile
import sqlite3
from typing import Dict, Any, List
from unittest.mock import patch

from go_doc_go.adapter.database import DatabaseAdapter
from go_doc_go.content_source.database import DatabaseContentSource
from go_doc_go.document_parser.factory import get_parser_for_content
from conftest import requires_sqlalchemy, requires_psycopg2, requires_docker


class TestDatabaseIntegrationSQLite:
    """Integration tests for database components with SQLite."""
    
    def test_adapter_and_content_source_consistency(self, sqlite_test_data, sample_database_configs):
        """Test that adapter and content source return consistent data."""
        # Test with DatabaseAdapter
        adapter = DatabaseAdapter()
        adapter_location = {
            "source": f"db://{sqlite_test_data}/documents/id/1/content"
        }
        adapter_result = adapter.get_content(adapter_location)
        
        # Test with DatabaseContentSource (SQLAlchemy-based)
        config = sample_database_configs["sqlite_blob"].copy()
        config["connection_string"] = f"sqlite:///{sqlite_test_data}"
        
        if requires_sqlalchemy:
            with patch('go_doc_go.content_source.database.SQLALCHEMY_AVAILABLE', True):
                from sqlalchemy import create_engine
                
                # Temporarily set the connection string to None to avoid engine creation in __init__
                original_conn_string = config["connection_string"]
                config["connection_string"] = None
                
                source = DatabaseContentSource(config)
                
                # Now set the real engine
                engine = create_engine(f"sqlite:///{sqlite_test_data}")
                source.engine = engine
                source.connection_string = original_conn_string
                source_result = source.fetch_document("1")
                
                # Both should return the same content
                assert adapter_result["content"] == source_result["content"]
                assert adapter_result["content_type"] == "text"  # Adapter determines this
    
    def test_markdown_document_pipeline_sqlite(self, sqlite_test_data):
        """Test complete pipeline for markdown document with SQLite."""
        adapter = DatabaseAdapter()
        
        # Test markdown document (id=2)
        location_data = {
            "source": f"db://{sqlite_test_data}/documents/id/2/content"
        }
        
        # Fetch document using adapter
        doc_data = adapter.get_content(location_data)
        
        assert "# Header" in doc_data["content"]
        assert doc_data["content_type"] == "markdown"
        
        # Simulate document structure for parser
        parser_input = {
            "id": location_data["source"],
            "content": doc_data["content"],
            "doc_type": doc_data["content_type"],
            "metadata": doc_data["metadata"]
        }
        
        # Parse document
        parser = get_parser_for_content(parser_input)
        parsed = parser.parse(parser_input)
        
        # Verify parsing results
        assert "document" in parsed
        assert "elements" in parsed
        assert len(parsed["elements"]) > 0
        
        # Check for headers
        headers = [e for e in parsed["elements"] if e.get("element_type") == "header"]
        assert len(headers) >= 1  # Should have at least the main header
    
    def test_json_document_pipeline_sqlite(self, sqlite_test_data):
        """Test complete pipeline for JSON document with SQLite."""
        adapter = DatabaseAdapter()
        
        # Test JSON document (id=4)
        location_data = {
            "source": f"db://{sqlite_test_data}/documents/id/4/content"
        }
        
        # Fetch document
        doc_data = adapter.get_content(location_data)
        
        # Content type detection may vary, but content should be correct
        assert doc_data["content_type"] in ["json", "text", "csv"]  # Allow flexible content type detection
        assert '"key": "value"' in doc_data["content"]
        
        # Verify JSON is valid regardless of detected content type
        json_data = json.loads(doc_data["content"])
        assert json_data["key"] == "value"
        assert json_data["number"] == 42
        
        # Simulate document structure for parser
        parser_input = {
            "id": location_data["source"],
            "content": doc_data["content"],
            "doc_type": doc_data["content_type"],
            "metadata": doc_data["metadata"]
        }
        
        # Parse document
        parser = get_parser_for_content(parser_input)
        parsed = parser.parse(parser_input)
        
        # Verify JSON structure is parsed
        assert len(parsed["elements"]) > 0
        
        # Check for specific JSON paths in elements
        element_contents = [e.get("content_preview", "") for e in parsed["elements"]]
        joined_content = " ".join(element_contents)
        
        assert "value" in joined_content
        assert "42" in joined_content
    
    def test_csv_document_pipeline_sqlite(self, sqlite_test_data):
        """Test complete pipeline for CSV document with SQLite."""
        adapter = DatabaseAdapter()
        
        # Test CSV document (id=5)
        location_data = {
            "source": f"db://{sqlite_test_data}/documents/id/5/content"
        }
        
        # Fetch document
        doc_data = adapter.get_content(location_data)
        
        assert doc_data["content_type"] == "csv"
        assert "Name,Age,City" in doc_data["content"]
        
        # Simulate document structure for parser
        parser_input = {
            "id": location_data["source"],
            "content": doc_data["content"],
            "doc_type": doc_data["content_type"],
            "metadata": doc_data["metadata"]
        }
        
        # Parse document
        parser = get_parser_for_content(parser_input)
        parsed = parser.parse(parser_input)
        
        # Verify CSV parsing
        assert "document" in parsed
        assert parsed["document"]["metadata"]["row_count"] == 4  # 1 header + 3 data rows
        assert parsed["document"]["metadata"]["column_count"] == 3  # 3 columns
        
        # Check for table elements
        table_headers = [e for e in parsed["elements"] if e.get("element_type") == "table_header_row"]
        table_rows = [e for e in parsed["elements"] if e.get("element_type") == "table_row"]
        
        assert len(table_headers) == 1
        assert len(table_rows) == 3
    
    @requires_sqlalchemy
    def test_json_mode_document_pipeline_sqlite(self, sqlite_test_data, sample_database_configs):
        """Test JSON mode document processing with SQLite."""
        config = sample_database_configs["sqlite_json"].copy()
        config["connection_string"] = f"sqlite:///{sqlite_test_data}"
        
        with patch('go_doc_go.content_source.database.SQLALCHEMY_AVAILABLE', True):
            from sqlalchemy import create_engine
            
            # Temporarily set the connection string to None to avoid engine creation in __init__
            original_conn_string = config["connection_string"]
            config["connection_string"] = None
            
            source = DatabaseContentSource(config)
            
            # Now set the real engine
            engine = create_engine(f"sqlite:///{sqlite_test_data}")
            source.engine = engine
            source.connection_string = original_conn_string
            
            # Fetch JSON document
            doc_data = source.fetch_document("1")
            
            assert doc_data["content_type"] == "application/json"
            
            # Parse the JSON content
            json_content = json.loads(doc_data["content"])
            assert json_content["name"] == "Record 1"
            assert json_content["status"] == "active"
            
            # Simulate document structure for parser
            parser_input = {
                "id": doc_data["id"],
                "content": doc_data["content"],
                "doc_type": "json",
                "metadata": doc_data["metadata"]
            }
            
            # Parse document
            parser = get_parser_for_content(parser_input)
            parsed = parser.parse(parser_input)
            
            # Should parse as JSON document
            assert len(parsed["elements"]) > 0
            
            # Find elements containing our test data
            element_contents = [e.get("content_preview", "") for e in parsed["elements"]]
            joined_content = " ".join(element_contents)
            
            assert "Record 1" in joined_content
            assert "active" in joined_content
    
    def test_binary_content_handling_sqlite(self, sqlite_test_data):
        """Test handling of binary files with SQLite."""
        adapter = DatabaseAdapter()
        
        # Test binary document (PNG in binary_docs table)
        location_data = {
            "source": f"db://{sqlite_test_data}/binary_docs/id/1/content"
        }
        
        # Fetch binary content
        binary_result = adapter.get_binary_content(location_data)
        
        assert isinstance(binary_result, bytes)
        assert binary_result.startswith(b'\x89PNG')
        assert len(binary_result) > 0
    
    def test_error_handling_and_recovery_sqlite(self, sqlite_test_data):
        """Test error handling and recovery with SQLite."""
        adapter = DatabaseAdapter()
        
        # Test non-existent record
        location_data = {
            "source": f"db://{sqlite_test_data}/documents/id/999/content"
        }
        
        with pytest.raises(ValueError, match="Content not found"):
            adapter.get_content(location_data)
        
        # Test invalid table
        location_data = {
            "source": f"db://{sqlite_test_data}/nonexistent_table/id/1/content"
        }
        
        with pytest.raises(ValueError, match="Error fetching record"):
            adapter.get_content(location_data)
    
    def test_metadata_preservation_sqlite(self, sqlite_test_data):
        """Test that metadata is preserved through the pipeline."""
        adapter = DatabaseAdapter()
        
        location_data = {
            "source": f"db://{sqlite_test_data}/documents/id/1/content"
        }
        
        result = adapter.get_content(location_data)
        
        # Check that database metadata is preserved
        assert result["metadata"]["database"] == sqlite_test_data
        assert result["metadata"]["table"] == "documents"
        assert result["metadata"]["record_id"] == "1"
        assert result["metadata"]["content_column"] == "content"


class TestDatabaseIntegrationPerformance:
    """Performance and scalability tests for database integration."""
    
    def test_connection_caching_efficiency(self, sqlite_test_data):
        """Test that database connections are efficiently cached."""
        adapter = DatabaseAdapter()
        
        # Fetch same document multiple times
        location_data = {
            "source": f"db://{sqlite_test_data}/documents/id/1/content"
        }
        
        for _ in range(10):
            result = adapter.get_content(location_data)
            assert result["content"] == "This is the content of document 1."
        
        # Should only have one cached connection
        assert len(adapter.connections) == 1
    
    @requires_sqlalchemy
    def test_large_json_document_handling(self, sqlite_test_data, sample_database_configs):
        """Test handling of large JSON documents."""
        # Create a large JSON document in SQLite
        conn = sqlite3.connect(sqlite_test_data)
        cursor = conn.cursor()
        
        # Create large JSON content
        large_json_data = {
            "data": [{"id": i, "value": f"item_{i}", "metadata": {"index": i}} for i in range(1000)]
        }
        large_json_str = json.dumps(large_json_data)
        
        # Use current timestamp format like the fixtures
        from datetime import datetime, timedelta
        current_time = datetime.now()
        timestamp_str = (current_time - timedelta(minutes=2)).strftime("%Y-%m-%d %H:%M:%S")
        
        cursor.execute("""
            INSERT INTO json_records (id, name, description, data_field, status, config, tags, timestamp)
            VALUES (?, ?, ?, ?, ?, ?, ?, ?)
        """, (100, "Large Document", "Large JSON test document", large_json_str, "active", 
             '{"type": "large"}', "large,test", timestamp_str))
        
        conn.commit()
        conn.close()
        
        # Test with DatabaseContentSource
        config = sample_database_configs["sqlite_json"].copy()
        config["connection_string"] = f"sqlite:///{sqlite_test_data}"
        
        with patch('go_doc_go.content_source.database.SQLALCHEMY_AVAILABLE', True):
            from sqlalchemy import create_engine
            
            # Temporarily set the connection string to None to avoid engine creation in __init__
            original_conn_string = config["connection_string"]
            config["connection_string"] = None
            
            source = DatabaseContentSource(config)
            
            # Now set the real engine
            engine = create_engine(f"sqlite:///{sqlite_test_data}")
            source.engine = engine
            source.connection_string = original_conn_string
            
            # Should handle large document without issues
            result = source.fetch_document("100")
            
            # Verify large content is handled properly
            assert "Large Document" in result["content"]
            assert len(result["content"]) > 10000  # Should be a large document
            
            # Should still be valid JSON
            json_content = json.loads(result["content"])
            assert "data_field" in json_content
            assert len(json_content["data_field"]) > 5000  # Large JSON string
    
    def test_cleanup_functionality(self, sqlite_test_data):
        """Test database connection cleanup."""
        adapter = DatabaseAdapter()
        
        # Create multiple connections
        for i in range(1, 4):
            location_data = {
                "source": f"db://{sqlite_test_data}/documents/id/{i}/content"
            }
            adapter.get_content(location_data)
        
        # Should have cached connections
        assert len(adapter.connections) >= 1
        
        # Cleanup should close all connections
        adapter.cleanup()
        assert len(adapter.connections) == 0