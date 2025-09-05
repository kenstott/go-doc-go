"""
Tests for S3 adapter.
"""

import pytest
import json
from unittest.mock import patch, MagicMock
from typing import Dict, Any

from go_doc_go.adapter.s3 import S3Adapter
from .conftest import requires_boto3, requires_docker


class TestS3AdapterUnit:
    """Unit tests for S3Adapter without real S3/Minio."""
    
    def test_adapter_initialization_without_boto3(self):
        """Test that adapter raises error when boto3 is not available."""
        with patch('go_doc_go.adapter.s3.BOTO3_AVAILABLE', False):
            with pytest.raises(ImportError, match="boto3 is required"):
                S3Adapter()
    
    @patch('go_doc_go.adapter.s3.boto3')
    def test_adapter_initialization_with_config(self, mock_boto3):
        """Test adapter initialization with custom configuration."""
        config = {
            "region": "us-west-2",
            "aws_access_key_id": "test-key",
            "aws_secret_access_key": "test-secret",
            "use_credentials": True
        }
        
        with patch('go_doc_go.adapter.s3.BOTO3_AVAILABLE', True):
            adapter = S3Adapter(config)
            
            assert adapter.default_region == "us-west-2"
            assert adapter.use_credentials is True
            assert "aws_access_key_id" in adapter.credentials
    
    def test_supports_location_s3_uri(self):
        """Test that adapter correctly identifies S3 URIs."""
        with patch('go_doc_go.adapter.s3.BOTO3_AVAILABLE', True):
            adapter = S3Adapter()
            
            # Valid S3 URIs
            assert adapter.supports_location({"source": "s3://bucket/key"}) is True
            assert adapter.supports_location({"source": "s3://bucket/path/to/file.txt"}) is True
            
            # Invalid URIs
            assert adapter.supports_location({"source": "http://example.com"}) is False
            assert adapter.supports_location({"source": "/local/path"}) is False
            assert adapter.supports_location({"source": ""}) is False
    
    def test_parse_s3_uri(self):
        """Test S3 URI parsing."""
        with patch('go_doc_go.adapter.s3.BOTO3_AVAILABLE', True):
            adapter = S3Adapter()
            
            # Standard S3 URI
            bucket, key, region = adapter._parse_s3_uri("s3://my-bucket/path/to/file.txt")
            assert bucket == "my-bucket"
            assert key == "path/to/file.txt"
            assert region is None
            
            # S3 URI with region
            adapter.default_region = "us-east-1"
            bucket, key, region = adapter._parse_s3_uri("s3://my-bucket/file.txt")
            assert bucket == "my-bucket"
            assert key == "file.txt"
            assert region == "us-east-1"
    
    def test_is_text_content(self):
        """Test MIME type detection for text content."""
        # Text types
        assert S3Adapter._is_text_content("text/plain") is True
        assert S3Adapter._is_text_content("text/html") is True
        assert S3Adapter._is_text_content("application/json") is True
        assert S3Adapter._is_text_content("application/xml") is True
        assert S3Adapter._is_text_content("application/yaml") is True
        assert S3Adapter._is_text_content("application/javascript") is True
        assert S3Adapter._is_text_content("application/csv") is True
        
        # Binary types
        assert S3Adapter._is_text_content("image/png") is False
        assert S3Adapter._is_text_content("application/pdf") is False
        assert S3Adapter._is_text_content("application/octet-stream") is False
        assert S3Adapter._is_text_content("video/mp4") is False
    
    @patch('go_doc_go.adapter.s3.boto3')
    def test_get_content_text_file(self, mock_boto3):
        """Test getting text content from S3."""
        # Setup mock
        mock_client = MagicMock()
        mock_boto3.client.return_value = mock_client
        
        # Mock head_object response
        mock_client.head_object.return_value = {
            'ContentType': 'text/plain',
            'ContentLength': 100,
            'LastModified': MagicMock(),
            'ETag': '"abc123"',
            'StorageClass': 'STANDARD',
            'Metadata': {'custom': 'value'}
        }
        
        # Mock get_object response
        mock_body = MagicMock()
        mock_body.read.return_value = b"Hello, World!"
        mock_client.get_object.return_value = {'Body': mock_body}
        
        with patch('go_doc_go.adapter.s3.BOTO3_AVAILABLE', True):
            adapter = S3Adapter()
            result = adapter.get_content({"source": "s3://test-bucket/test.txt"})
            
            assert result["content"] == "Hello, World!"
            assert result["content_type"] == "text"
            assert result["metadata"]["bucket"] == "test-bucket"
            assert result["metadata"]["key"] == "test.txt"
            assert result["metadata"]["content_type"] == "text/plain"
            assert result["metadata"]["is_binary"] is False


@requires_boto3
@requires_docker
class TestS3AdapterIntegration:
    """Integration tests for S3Adapter with Minio."""
    
    def test_get_content_from_minio(self, s3_client, minio_config, upload_test_files):
        """Test getting content from actual Minio instance."""
        # Upload test file
        content = "This is test content from Minio"
        s3_uri = upload_test_files("test-file.txt", content, "text/plain")
        
        # Create adapter with Minio config
        adapter_config = {
            "aws_access_key_id": minio_config["aws_access_key_id"],
            "aws_secret_access_key": minio_config["aws_secret_access_key"],
            "region": minio_config["region_name"]
        }
        
        # Patch the S3 client creation to use our endpoint
        with patch.object(S3Adapter, '_get_s3_client') as mock_get_client:
            mock_get_client.return_value = s3_client
            
            adapter = S3Adapter(adapter_config)
            result = adapter.get_content({"source": s3_uri})
            
            assert result["content"] == content
            assert result["metadata"]["bucket"] == minio_config["bucket_name"]
            assert result["metadata"]["key"] == "test-file.txt"
            assert result["metadata"]["is_binary"] is False
    
    def test_get_binary_content_from_minio(self, s3_client, minio_config, upload_test_files):
        """Test getting binary content from Minio."""
        # Upload binary file
        binary_content = b'\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR'
        s3_uri = upload_test_files("test-image.png", binary_content, "image/png")
        
        adapter_config = {
            "aws_access_key_id": minio_config["aws_access_key_id"],
            "aws_secret_access_key": minio_config["aws_secret_access_key"],
            "region": minio_config["region_name"]
        }
        
        with patch.object(S3Adapter, '_get_s3_client') as mock_get_client:
            mock_get_client.return_value = s3_client
            
            adapter = S3Adapter(adapter_config)
            result = adapter.get_binary_content({"source": s3_uri})
            
            assert result == binary_content
    
    def test_get_metadata_from_minio(self, s3_client, minio_config, upload_test_files):
        """Test getting metadata without downloading content."""
        # Upload test file
        content = "Test content for metadata"
        s3_uri = upload_test_files("metadata-test.md", content, "text/markdown")
        
        adapter_config = {
            "aws_access_key_id": minio_config["aws_access_key_id"],
            "aws_secret_access_key": minio_config["aws_secret_access_key"],
            "region": minio_config["region_name"]
        }
        
        with patch.object(S3Adapter, '_get_s3_client') as mock_get_client:
            mock_get_client.return_value = s3_client
            
            adapter = S3Adapter(adapter_config)
            metadata = adapter.get_metadata({"source": s3_uri})
            
            assert metadata["bucket"] == minio_config["bucket_name"]
            assert metadata["key"] == "metadata-test.md"
            assert metadata["content_type"] == "text/markdown"
            assert metadata["size"] == len(content)
            assert metadata["is_binary"] is False
            assert metadata["filename"] == "metadata-test.md"
    
    def test_get_content_with_different_encodings(self, s3_client, minio_config, upload_test_files):
        """Test handling different text encodings."""
        # UTF-8 content
        utf8_content = "Hello 世界 🌍"
        s3_uri = upload_test_files("utf8-test.txt", utf8_content.encode('utf-8'), "text/plain")
        
        adapter_config = {
            "aws_access_key_id": minio_config["aws_access_key_id"],
            "aws_secret_access_key": minio_config["aws_secret_access_key"],
            "region": minio_config["region_name"]
        }
        
        with patch.object(S3Adapter, '_get_s3_client') as mock_get_client:
            mock_get_client.return_value = s3_client
            
            adapter = S3Adapter(adapter_config)
            result = adapter.get_content({"source": s3_uri})
            
            assert result["content"] == utf8_content
            assert result["metadata"]["is_binary"] is False
    
    def test_error_handling_nonexistent_file(self, s3_client, minio_config):
        """Test error handling for non-existent files."""
        adapter_config = {
            "aws_access_key_id": minio_config["aws_access_key_id"],
            "aws_secret_access_key": minio_config["aws_secret_access_key"],
            "region": minio_config["region_name"]
        }
        
        with patch.object(S3Adapter, '_get_s3_client') as mock_get_client:
            mock_get_client.return_value = s3_client
            
            adapter = S3Adapter(adapter_config)
            
            with pytest.raises(ValueError, match="Object not found"):
                adapter.get_content({"source": f"s3://{minio_config['bucket_name']}/nonexistent.txt"})
    
    def test_resolve_uri(self, s3_client, minio_config):
        """Test URI resolution."""
        adapter_config = {
            "aws_access_key_id": minio_config["aws_access_key_id"],
            "aws_secret_access_key": minio_config["aws_secret_access_key"],
            "region": minio_config["region_name"]
        }
        
        adapter = S3Adapter(adapter_config)
        
        # Valid S3 URI
        result = adapter.resolve_uri("s3://bucket/path/to/file.txt")
        assert result == {"source": "s3://bucket/path/to/file.txt"}
        
        # Invalid URI
        with pytest.raises(ValueError, match="Not an S3 URI"):
            adapter.resolve_uri("http://example.com/file.txt")
    
    def test_json_content_parsing(self, s3_client, minio_config, upload_test_files, sample_json_content):
        """Test getting JSON content."""
        s3_uri = upload_test_files("test.json", sample_json_content, "application/json")
        
        adapter_config = {
            "aws_access_key_id": minio_config["aws_access_key_id"],
            "aws_secret_access_key": minio_config["aws_secret_access_key"],
            "region": minio_config["region_name"]
        }
        
        with patch.object(S3Adapter, '_get_s3_client') as mock_get_client:
            mock_get_client.return_value = s3_client
            
            adapter = S3Adapter(adapter_config)
            result = adapter.get_content({"source": s3_uri})
            
            assert result["content"] == sample_json_content
            assert result["content_type"] == "json"
            assert result["metadata"]["content_type"] == "application/json"
            
            # Verify JSON is valid
            parsed = json.loads(result["content"])
            assert parsed["title"] == "Sample JSON Document"
    
    def test_csv_content_handling(self, s3_client, minio_config, upload_test_files, sample_csv_content):
        """Test getting CSV content."""
        s3_uri = upload_test_files("test.csv", sample_csv_content, "text/csv")
        
        adapter_config = {
            "aws_access_key_id": minio_config["aws_access_key_id"],
            "aws_secret_access_key": minio_config["aws_secret_access_key"],
            "region": minio_config["region_name"]
        }
        
        with patch.object(S3Adapter, '_get_s3_client') as mock_get_client:
            mock_get_client.return_value = s3_client
            
            adapter = S3Adapter(adapter_config)
            result = adapter.get_content({"source": s3_uri})
            
            assert result["content"] == sample_csv_content
            assert result["content_type"] == "csv"
            assert result["metadata"]["is_binary"] is False
            
            # Verify CSV has expected structure
            lines = result["content"].split('\n')
            assert len(lines) == 6  # Header + 5 data rows
            assert "Name,Age,Department,Salary" in lines[0]