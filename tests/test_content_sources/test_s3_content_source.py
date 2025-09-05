"""
Tests for S3 content source.
"""

import pytest
import os
import time
from unittest.mock import patch, MagicMock, Mock
from typing import Dict, Any, List

# Import from parent directory's conftest
import sys
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..', 'test_adapters'))
from conftest import requires_boto3, requires_docker

from go_doc_go.content_source.s3 import S3ContentSource


class TestS3ContentSourceUnit:
    """Unit tests for S3ContentSource without real S3/Minio."""
    
    def test_content_source_initialization_without_boto3(self):
        """Test that content source raises error when boto3 is not available."""
        config = {
            "name": "test-s3-source",
            "bucket_name": "test-bucket"
        }
        
        with patch('go_doc_go.content_source.s3.BOTO3_AVAILABLE', False):
            with pytest.raises(ImportError, match="boto3 is required"):
                S3ContentSource(config)
    
    @patch('go_doc_go.content_source.s3.boto3')
    def test_content_source_initialization_with_config(self, mock_boto3):
        """Test content source initialization with configuration."""
        mock_client = MagicMock()
        mock_boto3.client.return_value = mock_client
        mock_client.list_buckets.return_value = {'Buckets': []}
        
        config = {
            "name": "test-s3-source",
            "bucket_name": "test-bucket",
            "prefix": "documents/",
            "region_name": "us-west-2",
            "aws_access_key_id": "test-key",
            "aws_secret_access_key": "test-secret",
            "endpoint_url": "http://localhost:9000",
            "include_extensions": [".txt", ".md"],
            "exclude_extensions": [".tmp"],
            "recursive": True,
            "max_depth": 5
        }
        
        with patch('go_doc_go.content_source.s3.BOTO3_AVAILABLE', True):
            source = S3ContentSource(config)
            
            assert source.bucket_name == "test-bucket"
            assert source.prefix == "documents/"
            assert source.region_name == "us-west-2"
            assert source.endpoint_url == "http://localhost:9000"
            assert source.include_extensions == [".txt", ".md"]
            assert source.exclude_extensions == [".tmp"]
            assert source.recursive is True
            assert source.max_depth == 5
    
    def test_extract_bucket_and_key(self):
        """Test extraction of bucket and key from S3 URI."""
        # Full S3 URI
        bucket, key = S3ContentSource._extract_bucket_and_key("s3://my-bucket/path/to/file.txt")
        assert bucket == "my-bucket"
        assert key == "path/to/file.txt"
        
        # Just key (no s3:// prefix)
        bucket, key = S3ContentSource._extract_bucket_and_key("path/to/file.txt")
        assert bucket is None
        assert key == "path/to/file.txt"
        
        # S3 URI with no key
        bucket, key = S3ContentSource._extract_bucket_and_key("s3://my-bucket/")
        assert bucket == "my-bucket"
        assert key == ""
    
    def test_should_include_object(self):
        """Test object filtering logic."""
        config = {
            "name": "test-source",
            "bucket_name": "test",
            "include_extensions": ["txt", "md"],
            "exclude_extensions": ["tmp", "bak"],
            "include_prefixes": ["docs/", "data/"],
            "exclude_prefixes": ["temp/", "backup/"],
            "include_patterns": [r".*report.*"],
            "exclude_patterns": [r".*draft.*"]
        }
        
        with patch('go_doc_go.content_source.s3.boto3'):
            with patch('go_doc_go.content_source.s3.BOTO3_AVAILABLE', True):
                with patch.object(S3ContentSource, '_initialize_s3_client', return_value=MagicMock()):
                    source = S3ContentSource(config)
                    
                    # Test extension filtering
                    assert source._should_include_object("file.txt") is False  # Doesn't match prefix
                    assert source._should_include_object("docs/file.txt") is False  # Doesn't match pattern
                    assert source._should_include_object("docs/report.txt") is True
                    assert source._should_include_object("docs/report.tmp") is False  # Excluded extension
                    assert source._should_include_object("temp/report.txt") is False  # Excluded prefix
                    assert source._should_include_object("docs/draft_report.txt") is False  # Excluded pattern
    
    @patch('go_doc_go.content_source.s3.boto3')
    def test_get_safe_connection_string(self, mock_boto3):
        """Test safe connection string generation."""
        mock_client = MagicMock()
        mock_boto3.client.return_value = mock_client
        mock_client.list_buckets.return_value = {'Buckets': []}
        
        config = {
            "name": "test-source",
            "bucket_name": "my-bucket",
            "region_name": "us-east-1",
            "aws_secret_access_key": "secret-key-should-not-appear"
        }
        
        with patch('go_doc_go.content_source.s3.BOTO3_AVAILABLE', True):
            source = S3ContentSource(config)
            conn_str = source.get_safe_connection_string()
            
            assert "my-bucket" in conn_str
            assert "secret-key-should-not-appear" not in conn_str
            assert "us-east-1" in conn_str
    
    @patch('go_doc_go.content_source.s3.boto3')
    def test_has_changed_with_cache(self, mock_boto3):
        """Test change detection with cache."""
        mock_client = MagicMock()
        mock_boto3.client.return_value = mock_client
        mock_client.list_buckets.return_value = {'Buckets': []}
        
        config = {
            "name": "test-source",
            "bucket_name": "test-bucket"
        }
        
        with patch('go_doc_go.content_source.s3.BOTO3_AVAILABLE', True):
            source = S3ContentSource(config)
            source.s3_client = mock_client
            
            # Add to cache
            source.content_cache["test-bucket/file.txt"] = {
                "metadata": {"last_modified": 1000.0}
            }
            
            # Test with older timestamp - should return False (not changed)
            assert source.has_changed("s3://test-bucket/file.txt", 1500.0) is False
            
            # Test with newer timestamp in cache - should return True (changed)
            source.content_cache["test-bucket/file.txt"]["metadata"]["last_modified"] = 2000.0
            assert source.has_changed("s3://test-bucket/file.txt", 1500.0) is True


@requires_boto3
@requires_docker
class TestS3ContentSourceIntegration:
    """Integration tests for S3ContentSource with Minio."""
    
    @pytest.fixture
    def s3_source(self, s3_client, minio_config):
        """Create S3ContentSource configured for Minio."""
        config = {
            "name": "test-minio-source",
            "bucket_name": minio_config["bucket_name"],
            "aws_access_key_id": minio_config["aws_access_key_id"],
            "aws_secret_access_key": minio_config["aws_secret_access_key"],
            "region_name": minio_config["region_name"],
            "endpoint_url": minio_config["endpoint_url"]
        }
        
        with patch('go_doc_go.content_source.s3.boto3.client') as mock_client:
            mock_client.return_value = s3_client
            source = S3ContentSource(config)
            source.s3_client = s3_client
            return source
    
    def test_fetch_document_from_minio(self, s3_source, upload_test_files, sample_text_content):
        """Test fetching document from Minio."""
        # Upload test document
        s3_uri = upload_test_files("test-doc.md", sample_text_content, "text/markdown")
        
        # Fetch document
        result = s3_source.fetch_document(s3_uri)
        
        assert result["id"] == s3_uri
        assert result["content"] == sample_text_content
        assert result["doc_type"] == "markdown"
        assert result["metadata"]["bucket"] == "test-bucket"
        assert result["metadata"]["key"] == "test-doc.md"
        assert result["metadata"]["filename"] == "test-doc.md"
        assert result["metadata"]["extension"] == "md"
        assert result["metadata"]["is_binary"] is False
    
    def test_fetch_binary_document(self, s3_source, upload_test_files, temp_binary_file):
        """Test fetching binary document from Minio."""
        # Read binary content
        with open(temp_binary_file, 'rb') as f:
            binary_content = f.read()
        
        # Upload binary file
        s3_uri = upload_test_files("test-image.png", binary_content, "image/png")
        
        # Fetch document
        result = s3_source.fetch_document(s3_uri)
        
        assert result["id"] == s3_uri
        assert result["content"] == ""  # Binary content should be empty string
        assert result["binary_path"] is not None  # Should have temp file path
        assert result["metadata"]["is_binary"] is True
        assert result["metadata"]["content_type"] == "image/png"
        
        # Verify temp file exists and contains correct data
        if result["binary_path"]:
            assert os.path.exists(result["binary_path"])
            with open(result["binary_path"], 'rb') as f:
                assert f.read() == binary_content
    
    def test_list_documents(self, s3_source, upload_test_files):
        """Test listing documents in S3/Minio."""
        # Upload multiple test files
        upload_test_files("doc1.txt", "Content 1", "text/plain")
        upload_test_files("doc2.md", "Content 2", "text/markdown")
        upload_test_files("data.json", '{"test": true}', "application/json")
        upload_test_files("folder/nested.txt", "Nested content", "text/plain")
        
        # List documents
        documents = s3_source.list_documents()
        
        # Should have at least the 4 files we uploaded
        assert len(documents) >= 4
        
        # Check document structure
        doc_keys = {doc["metadata"]["key"] for doc in documents}
        assert "doc1.txt" in doc_keys
        assert "doc2.md" in doc_keys
        assert "data.json" in doc_keys
        assert "folder/nested.txt" in doc_keys
        
        # Check metadata
        for doc in documents:
            assert "id" in doc
            assert doc["id"].startswith("s3://")
            assert "metadata" in doc
            assert "key" in doc["metadata"]
            assert "bucket" in doc["metadata"]
            assert "size" in doc["metadata"]
    
    def test_list_documents_with_prefix(self, s3_source, upload_test_files):
        """Test listing documents with prefix filtering."""
        # Upload files with different prefixes
        upload_test_files("docs/file1.txt", "Content 1", "text/plain")
        upload_test_files("docs/file2.txt", "Content 2", "text/plain")
        upload_test_files("data/file3.txt", "Content 3", "text/plain")
        upload_test_files("other.txt", "Other content", "text/plain")
        
        # Configure source with prefix
        s3_source.prefix = "docs/"
        
        # List documents
        documents = s3_source.list_documents()
        
        # Should only get docs/ files
        keys = [doc["metadata"]["key"] for doc in documents]
        docs_files = [k for k in keys if k.startswith("docs/")]
        assert len(docs_files) >= 2
        
        # Should not include files outside prefix
        assert not any(k.startswith("data/") for k in keys)
    
    def test_has_changed_detection(self, s3_source, s3_client, upload_test_files):
        """Test document change detection."""
        # Upload initial file
        s3_uri = upload_test_files("changing-doc.txt", "Initial content", "text/plain")
        
        # Get initial modification time
        initial_doc = s3_source.fetch_document(s3_uri)
        initial_modified = initial_doc["metadata"]["last_modified"]
        
        # Check has_changed with current timestamp - should be False
        assert s3_source.has_changed(s3_uri, initial_modified) is False
        
        # Wait a moment and update the file
        time.sleep(1)
        s3_client.put_object(
            Bucket="test-bucket",
            Key="changing-doc.txt",
            Body=b"Updated content",
            ContentType="text/plain"
        )
        
        # Clear cache to force API check
        s3_source.content_cache.clear()
        
        # Should detect change
        assert s3_source.has_changed(s3_uri, initial_modified) is True
    
    def test_follow_links_in_markdown(self, s3_source, upload_test_files):
        """Test following links in markdown documents."""
        # Create linked documents
        main_content = """# Main Document
        
This document links to [another document](./linked-doc.md).
And also to [a third document](./subfolder/third.md).
"""
        
        linked_content = """# Linked Document
        
This is the linked document.
"""
        
        third_content = """# Third Document
        
This is the third document.
"""
        
        # Upload documents
        upload_test_files("main.md", main_content, "text/markdown")
        upload_test_files("linked-doc.md", linked_content, "text/markdown")
        upload_test_files("subfolder/third.md", third_content, "text/markdown")
        
        # Configure link following
        s3_source.local_link_mode = "relative"
        s3_source.max_link_depth = 2
        
        # Follow links from main document
        linked_docs = s3_source.follow_links(main_content, "s3://test-bucket/main.md")
        
        # Should find the linked documents
        assert len(linked_docs) == 2
        
        linked_keys = [doc["metadata"]["key"] for doc in linked_docs]
        assert "linked-doc.md" in linked_keys
        assert "subfolder/third.md" in linked_keys
    
    def test_content_caching(self, s3_source, upload_test_files):
        """Test content caching mechanism."""
        # Upload test file
        s3_uri = upload_test_files("cached-doc.txt", "Cached content", "text/plain")
        
        # Clear cache
        s3_source.content_cache.clear()
        
        # First fetch - should hit S3
        result1 = s3_source.fetch_document(s3_uri)
        assert result1["content"] == "Cached content"
        
        # Check cache was populated
        cache_key = "test-bucket/cached-doc.txt"
        assert cache_key in s3_source.content_cache
        
        # Mock S3 client to verify cache is used
        original_client = s3_source.s3_client
        mock_client = MagicMock()
        s3_source.s3_client = mock_client
        
        # Second fetch - should use cache
        result2 = s3_source.fetch_document(s3_uri)
        assert result2["content"] == "Cached content"
        
        # S3 client should not have been called
        mock_client.head_object.assert_not_called()
        mock_client.get_object.assert_not_called()
        
        # Restore original client
        s3_source.s3_client = original_client
    
    def test_json_document_parsing(self, s3_source, upload_test_files, sample_json_content):
        """Test JSON document detection and parsing."""
        s3_uri = upload_test_files("test-data.json", sample_json_content, "application/json")
        
        result = s3_source.fetch_document(s3_uri)
        
        assert result["doc_type"] == "json"
        assert result["content"] == sample_json_content
        assert result["metadata"]["extension"] == "json"
        assert result["metadata"]["content_type"] == "application/json"
    
    def test_csv_document_handling(self, s3_source, upload_test_files, sample_csv_content):
        """Test CSV document handling."""
        s3_uri = upload_test_files("test-data.csv", sample_csv_content, "text/csv")
        
        result = s3_source.fetch_document(s3_uri)
        
        assert result["doc_type"] == "csv"
        assert result["content"] == sample_csv_content
        assert result["metadata"]["extension"] == "csv"
        assert not result["metadata"]["is_binary"]