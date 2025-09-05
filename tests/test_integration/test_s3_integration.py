"""
Integration tests for S3 adapter and content source with document processing.
"""

import pytest
import os
import sys
import json
import time
from typing import Dict, Any, List
from unittest.mock import patch

# Add test_adapters to path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..', 'test_adapters'))
from conftest import requires_boto3, requires_docker

from go_doc_go.adapter.s3 import S3Adapter
from go_doc_go.content_source.s3 import S3ContentSource
from go_doc_go.document_parser.factory import get_parser_for_content


@requires_boto3
@requires_docker
class TestS3Integration:
    """End-to-end integration tests for S3 with document processing."""
    
    @pytest.fixture
    def s3_components(self, s3_client, minio_config):
        """Create S3 adapter and content source for testing."""
        config = {
            "name": "test-integration",
            "bucket_name": minio_config["bucket_name"],
            "aws_access_key_id": minio_config["aws_access_key_id"],
            "aws_secret_access_key": minio_config["aws_secret_access_key"],
            "region_name": minio_config["region_name"],
            "endpoint_url": minio_config["endpoint_url"]
        }
        
        # Create adapter
        with patch.object(S3Adapter, '_get_s3_client') as mock_adapter_client:
            mock_adapter_client.return_value = s3_client
            adapter = S3Adapter(config)
        
        # Create content source
        with patch('go_doc_go.content_source.s3.boto3.client') as mock_source_client:
            mock_source_client.return_value = s3_client
            source = S3ContentSource(config)
            source.s3_client = s3_client
        
        return adapter, source
    
    def test_markdown_document_pipeline(self, s3_components, upload_test_files):
        """Test complete pipeline for markdown document."""
        adapter, source = s3_components
        
        markdown_content = """# Project Documentation

## Overview
This is a test project with [API documentation](./api-docs.md).

## Features
- Feature 1: Basic functionality
- Feature 2: Advanced features
- Feature 3: Integration with [external systems](https://example.com)

## Code Example
```python
def hello_world():
    print("Hello, World!")
```

## References
See also [user guide](./guides/user-guide.md) for more information.
"""
        
        # Upload markdown document
        s3_uri = upload_test_files("docs/project.md", markdown_content, "text/markdown")
        
        # Fetch document using content source
        doc_data = source.fetch_document(s3_uri)
        
        assert doc_data["doc_type"] == "markdown"
        assert doc_data["content"] == markdown_content
        assert doc_data["metadata"]["extension"] == "md"
        
        # Parse document
        parser = get_parser_for_content(doc_data)
        parsed = parser.parse(doc_data)
        
        # Verify parsing results
        assert "document" in parsed
        assert "elements" in parsed
        assert len(parsed["elements"]) > 0
        
        # Check for headers
        headers = [e for e in parsed["elements"] if e.get("element_type") == "header"]
        assert len(headers) >= 3  # Should have main header and section headers
        
        # Check for code block
        code_blocks = [e for e in parsed["elements"] if e.get("element_type") == "code_block"]
        assert len(code_blocks) >= 1
        assert "hello_world" in code_blocks[0].get("content_preview", "")
    
    def test_json_document_pipeline(self, s3_components, upload_test_files):
        """Test complete pipeline for JSON document."""
        adapter, source = s3_components
        
        json_data = {
            "project": {
                "name": "Test Project",
                "version": "1.0.0",
                "dependencies": [
                    {"name": "boto3", "version": "1.38.3"},
                    {"name": "pytest", "version": "8.3.5"}
                ],
                "config": {
                    "debug": True,
                    "timeout": 30,
                    "endpoints": {
                        "api": "https://api.example.com",
                        "auth": "https://auth.example.com"
                    }
                }
            },
            "metadata": {
                "created": "2024-01-01",
                "author": "Test User"
            }
        }
        
        json_content = json.dumps(json_data, indent=2)
        
        # Upload JSON document
        s3_uri = upload_test_files("config/project.json", json_content, "application/json")
        
        # Fetch document
        doc_data = source.fetch_document(s3_uri)
        
        assert doc_data["doc_type"] == "json"
        assert json.loads(doc_data["content"]) == json_data
        
        # Parse document
        parser = get_parser_for_content(doc_data)
        parsed = parser.parse(doc_data)
        
        # Verify JSON structure is parsed
        assert len(parsed["elements"]) > 0
        
        # Check for specific JSON paths in elements
        element_contents = [e.get("content_preview", "") for e in parsed["elements"]]
        joined_content = " ".join(element_contents)
        
        assert "Test Project" in joined_content
        assert "boto3" in joined_content
        assert "api.example.com" in joined_content
    
    def test_csv_document_pipeline(self, s3_components, upload_test_files, sample_csv_content):
        """Test complete pipeline for CSV document."""
        adapter, source = s3_components
        
        # Upload CSV document
        s3_uri = upload_test_files("data/employees.csv", sample_csv_content, "text/csv")
        
        # Fetch document
        doc_data = source.fetch_document(s3_uri)
        
        assert doc_data["doc_type"] == "csv"
        assert doc_data["content"] == sample_csv_content
        
        # Parse document
        parser = get_parser_for_content(doc_data)
        parsed = parser.parse(doc_data)
        
        # Verify CSV parsing
        assert "document" in parsed
        assert parsed["document"]["metadata"]["row_count"] == 5  # 5 data rows
        assert parsed["document"]["metadata"]["column_count"] == 4  # 4 columns
        
        # Check for table elements
        table_headers = [e for e in parsed["elements"] if e.get("element_type") == "table_header_row"]
        table_rows = [e for e in parsed["elements"] if e.get("element_type") == "table_row"]
        
        assert len(table_headers) == 1
        assert len(table_rows) == 5
        
        # Verify header content
        assert "Name,Age,Department,Salary" in table_headers[0].get("content_preview", "")
    
    def test_binary_file_handling(self, s3_components, upload_test_files):
        """Test handling of binary files."""
        adapter, source = s3_components
        
        # Create a simple PNG header (not a valid image, just for testing)
        png_header = b'\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x10\x00\x00\x00\x10'
        
        # Upload binary file
        s3_uri = upload_test_files("images/test.png", png_header, "image/png")
        
        # Fetch using adapter
        adapter_result = adapter.get_binary_content({"source": s3_uri})
        assert adapter_result == png_header
        
        # Fetch using content source
        source_result = source.fetch_document(s3_uri)
        assert source_result["metadata"]["is_binary"] is True
        assert source_result["content"] == ""  # Binary files have empty content
        assert source_result["binary_path"] is not None
        
        # Verify temp file contains correct data
        if source_result["binary_path"]:
            with open(source_result["binary_path"], 'rb') as f:
                assert f.read() == png_header
    
    def test_document_linking_and_traversal(self, s3_components, upload_test_files):
        """Test document linking and traversal."""
        adapter, source = s3_components
        
        # Create a network of linked documents
        index_content = """# Documentation Index

- [Getting Started](./getting-started.md)
- [API Reference](./api/reference.md)
- [Examples](./examples/index.md)
"""
        
        getting_started_content = """# Getting Started

Follow the [installation guide](./installation.md) to set up the project.
Then check out the [tutorials](./tutorials/basics.md).
"""
        
        api_reference_content = """# API Reference

## Endpoints
- `/api/v1/users` - User management
- `/api/v1/projects` - Project management

See [authentication](../auth/oauth.md) for details.
"""
        
        # Upload documents
        upload_test_files("docs/index.md", index_content, "text/markdown")
        upload_test_files("docs/getting-started.md", getting_started_content, "text/markdown")
        upload_test_files("docs/api/reference.md", api_reference_content, "text/markdown")
        
        # Configure source for link following
        source.local_link_mode = "relative"
        source.max_link_depth = 2
        
        # Follow links from index
        linked_docs = source.follow_links(index_content, "s3://test-bucket/docs/index.md")
        
        # Should find linked documents
        linked_keys = [doc["metadata"]["key"] for doc in linked_docs]
        assert "docs/getting-started.md" in linked_keys
        assert "docs/api/reference.md" in linked_keys
    
    def test_concurrent_document_access(self, s3_components, upload_test_files):
        """Test concurrent access to multiple documents."""
        adapter, source = s3_components
        
        # Upload multiple documents
        doc_uris = []
        for i in range(5):
            content = f"Document {i} content"
            uri = upload_test_files(f"concurrent/doc{i}.txt", content, "text/plain")
            doc_uris.append(uri)
        
        # Fetch all documents
        documents = []
        for uri in doc_uris:
            doc = source.fetch_document(uri)
            documents.append(doc)
        
        # Verify all documents were fetched correctly
        assert len(documents) == 5
        for i, doc in enumerate(documents):
            assert doc["content"] == f"Document {i} content"
            assert doc["metadata"]["key"] == f"concurrent/doc{i}.txt"
    
    def test_large_file_handling(self, s3_components, upload_test_files):
        """Test handling of large files."""
        adapter, source = s3_components
        
        # Create a large CSV file
        large_csv = "Col1,Col2,Col3,Col4,Col5\n"
        for i in range(10000):
            large_csv += f"Row{i},Value{i},Data{i},Info{i},Status{i}\n"
        
        # Upload large file
        s3_uri = upload_test_files("data/large.csv", large_csv, "text/csv")
        
        # Fetch document
        doc_data = source.fetch_document(s3_uri)
        
        assert doc_data["doc_type"] == "csv"
        assert len(doc_data["content"]) == len(large_csv)
        assert doc_data["metadata"]["size"] == len(large_csv.encode('utf-8'))
        
        # Parse with truncation
        parser = get_parser_for_content(doc_data)
        parser.max_rows = 1000  # Limit rows for testing
        parsed = parser.parse(doc_data)
        
        # Should handle truncation properly
        assert parsed["document"]["metadata"]["row_count"] <= 1000
    
    def test_document_metadata_preservation(self, s3_components, upload_test_files):
        """Test that document metadata is preserved through the pipeline."""
        adapter, source = s3_components
        
        content = "Test content with metadata"
        
        # Upload with specific metadata
        s3_uri = upload_test_files("metadata-test.txt", content, "text/plain")
        
        # Add custom metadata using S3 client
        source.s3_client.copy_object(
            Bucket="test-bucket",
            CopySource={"Bucket": "test-bucket", "Key": "metadata-test.txt"},
            Key="metadata-test.txt",
            Metadata={
                "author": "Test Author",
                "version": "1.0",
                "tags": "test,integration,s3"
            },
            MetadataDirective="REPLACE"
        )
        
        # Clear cache to force fresh fetch
        source.content_cache.clear()
        
        # Fetch document
        doc_data = source.fetch_document(s3_uri)
        
        # Check S3 metadata is preserved
        assert doc_data["metadata"]["bucket"] == "test-bucket"
        assert doc_data["metadata"]["key"] == "metadata-test.txt"
        assert doc_data["metadata"]["filename"] == "metadata-test.txt"
        assert doc_data["metadata"]["extension"] == "txt"
        
        # Custom metadata should be in user_metadata
        if "user_metadata" in doc_data["metadata"] and doc_data["metadata"]["user_metadata"]:
            assert doc_data["metadata"]["user_metadata"].get("author") == "Test Author"
            assert doc_data["metadata"]["user_metadata"].get("version") == "1.0"
    
    def test_error_recovery(self, s3_components):
        """Test error handling and recovery."""
        adapter, source = s3_components
        
        # Test non-existent file
        with pytest.raises(ValueError, match="Object not found"):
            source.fetch_document("s3://test-bucket/nonexistent.txt")
        
        # Test invalid S3 URI
        with pytest.raises(ValueError, match="Invalid S3 location"):
            adapter.get_content({"source": "http://not-s3.com/file.txt"})
        
        # Test with invalid bucket (should fail)
        with pytest.raises(ValueError):
            source.fetch_document("s3://invalid-bucket-name/file.txt")