"""
Tests for File adapter.
"""

import pytest
import os
import tempfile
import shutil
from unittest.mock import patch, MagicMock
from typing import Dict, Any

from go_doc_go.adapter.file import FileAdapter


class TestFileAdapterUnit:
    """Unit tests for FileAdapter without real files."""
    
    def test_adapter_initialization_with_config(self):
        """Test adapter initialization with custom configuration."""
        config = {
            "base_path": "/test/base",
            "follow_symlinks": False,
            "encoding_fallbacks": ['utf-8', 'latin-1']
        }
        
        adapter = FileAdapter(config)
        
        assert adapter.base_path == "/test/base"
        assert adapter.follow_symlinks is False
        assert adapter.encoding_fallbacks == ['utf-8', 'latin-1']
    
    def test_adapter_initialization_with_defaults(self):
        """Test adapter initialization with default values."""
        adapter = FileAdapter()
        
        assert adapter.base_path == ""
        assert adapter.follow_symlinks is True
        assert adapter.encoding_fallbacks == ['utf-8', 'latin-1', 'cp1252']
    
    def test_supports_location_absolute_path(self):
        """Test that adapter correctly identifies absolute file paths."""
        adapter = FileAdapter()
        
        with patch('os.path.exists') as mock_exists:
            mock_exists.return_value = True
            
            # Absolute paths
            assert adapter.supports_location({"source": "/absolute/path/file.txt"}) is True
            
            # Windows absolute paths
            if os.name == 'nt':
                assert adapter.supports_location({"source": "C:\\absolute\\path\\file.txt"}) is True
    
    def test_supports_location_file_uri(self):
        """Test that adapter correctly identifies file:// URIs."""
        adapter = FileAdapter()
        
        with patch('os.path.exists') as mock_exists:
            mock_exists.return_value = True
            
            # file:// URIs
            assert adapter.supports_location({"source": "file:///absolute/path/file.txt"}) is True
            assert adapter.supports_location({"source": "file://server/share/file.txt"}) is True
    
    def test_supports_location_relative_path(self):
        """Test that adapter handles relative paths correctly."""
        adapter = FileAdapter({"base_path": "/base"})
        
        with patch('os.path.exists') as mock_exists:
            mock_exists.return_value = True
            
            # Relative paths with base_path
            assert adapter.supports_location({"source": "relative/file.txt"}) is True
            
            mock_exists.assert_called_with("/base/relative/file.txt")
    
    def test_supports_location_invalid_sources(self):
        """Test that adapter rejects invalid sources."""
        adapter = FileAdapter()
        
        # Invalid sources
        assert adapter.supports_location({"source": ""}) is False
        assert adapter.supports_location({"source": "http://example.com"}) is False
        assert adapter.supports_location({"source": "s3://bucket/key"}) is False
        assert adapter.supports_location({}) is False  # No source key
    
    def test_is_binary_file_by_extension(self):
        """Test binary file detection by extension."""
        # Text extensions
        assert FileAdapter._is_binary_file("test.txt") is False
        assert FileAdapter._is_binary_file("test.md") is False
        assert FileAdapter._is_binary_file("test.json") is False
        assert FileAdapter._is_binary_file("test.py") is False
        assert FileAdapter._is_binary_file("test.html") is False
        
        # Binary extensions
        assert FileAdapter._is_binary_file("test.pdf") is True
        assert FileAdapter._is_binary_file("test.docx") is True
        assert FileAdapter._is_binary_file("test.png") is True
        assert FileAdapter._is_binary_file("test.exe") is True
    
    def test_is_binary_file_by_content(self):
        """Test binary file detection by content analysis."""
        with tempfile.NamedTemporaryFile(delete=False) as tmp_file:
            # Write binary content (contains null bytes)
            tmp_file.write(b'Hello\x00World\x00')
            tmp_file_path = tmp_file.name
        
        try:
            assert FileAdapter._is_binary_file(tmp_file_path) is True
        finally:
            os.unlink(tmp_file_path)
        
        with tempfile.NamedTemporaryFile(mode='w', delete=False, suffix='.unknown') as tmp_file:
            # Write text content
            tmp_file.write('Hello World, this is plain text')
            tmp_file_path = tmp_file.name
        
        try:
            assert FileAdapter._is_binary_file(tmp_file_path) is False
        finally:
            os.unlink(tmp_file_path)
    
    def test_parse_file_uri(self):
        """Test file URI parsing."""
        adapter = FileAdapter()
        
        # Standard file URI
        result = adapter.resolve_uri("file:///absolute/path/file.txt")
        assert result == {"source": "/absolute/path/file.txt"}
        
        # Network file URI
        result = adapter.resolve_uri("file://server/share/file.txt")
        expected_path = "server/share/file.txt"
        if os.name == 'nt':
            expected_path = "\\server\\share\\file.txt"
        assert result == {"source": os.path.normpath(expected_path)}
    
    def test_resolve_uri_relative_path(self):
        """Test resolving relative paths with base_path."""
        adapter = FileAdapter({"base_path": "/base/path"})
        
        result = adapter.resolve_uri("relative/file.txt")
        expected = os.path.abspath(os.path.join("/base/path", "relative/file.txt"))
        assert result == {"source": expected}
    
    def test_resolve_uri_absolute_path(self):
        """Test resolving absolute paths."""
        adapter = FileAdapter()
        
        result = adapter.resolve_uri("/absolute/path/file.txt")
        assert result == {"source": "/absolute/path/file.txt"}
    
    def test_get_content_file_not_found(self):
        """Test error handling for non-existent files."""
        adapter = FileAdapter()
        
        with pytest.raises(ValueError, match="File not found"):
            adapter.get_content({"source": "/nonexistent/file.txt"})
    
    def test_get_content_not_a_file(self):
        """Test error handling when source is not a file."""
        adapter = FileAdapter()
        
        with patch('os.path.exists', return_value=True), \
             patch('os.path.isfile', return_value=False):
            
            with pytest.raises(ValueError, match="Not a file"):
                adapter.get_content({"source": "/path/to/directory"})
    
    def test_get_content_symlink_disabled(self):
        """Test error handling for symlinks when following is disabled."""
        adapter = FileAdapter({"follow_symlinks": False})
        
        with patch('os.path.exists', return_value=True), \
             patch('os.path.isfile', return_value=True), \
             patch('os.path.islink', return_value=True):
            
            with pytest.raises(ValueError, match="symlink and following symlinks is disabled"):
                adapter.get_content({"source": "/path/to/symlink"})


class TestFileAdapterIntegration:
    """Integration tests for FileAdapter with real files."""
    
    @pytest.fixture
    def temp_dir(self):
        """Create a temporary directory for test files."""
        temp_dir = tempfile.mkdtemp()
        yield temp_dir
        shutil.rmtree(temp_dir, ignore_errors=True)
    
    @pytest.fixture
    def file_adapter(self, temp_dir):
        """Create FileAdapter configured for temp directory."""
        return FileAdapter({"base_path": temp_dir})
    
    def create_test_file(self, temp_dir: str, filename: str, content: str) -> str:
        """Create a test file with given content."""
        file_path = os.path.join(temp_dir, filename)
        os.makedirs(os.path.dirname(file_path), exist_ok=True)
        
        with open(file_path, 'w', encoding='utf-8') as f:
            f.write(content)
        
        return file_path
    
    def create_binary_file(self, temp_dir: str, filename: str, content: bytes) -> str:
        """Create a binary test file with given content."""
        file_path = os.path.join(temp_dir, filename)
        os.makedirs(os.path.dirname(file_path), exist_ok=True)
        
        with open(file_path, 'wb') as f:
            f.write(content)
        
        return file_path
    
    def test_get_content_text_file(self, file_adapter, temp_dir, sample_text_content):
        """Test getting content from a text file."""
        file_path = self.create_test_file(temp_dir, "test.md", sample_text_content)
        
        result = file_adapter.get_content({"source": file_path})
        
        assert result["content"] == sample_text_content
        assert result["content_type"] == "markdown"
        assert result["metadata"]["filename"] == "test.md"
        assert result["metadata"]["path"] == file_path
        assert result["metadata"]["extension"] == ".md"
        assert result["metadata"]["is_binary"] is False
        assert result["metadata"]["size"] == len(sample_text_content)
        assert "modified" in result["metadata"]
        assert "created" in result["metadata"]
    
    def test_get_content_binary_file(self, file_adapter, temp_dir):
        """Test getting content from a binary file."""
        binary_content = b'\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x10'
        file_path = self.create_binary_file(temp_dir, "test.png", binary_content)
        
        result = file_adapter.get_content({"source": file_path})
        
        assert result["content"] == binary_content
        assert result["content_type"] == "text"  # DocumentTypeDetector defaults to 'text' for unknown content
        assert result["metadata"]["filename"] == "test.png"
        assert result["metadata"]["is_binary"] is True
        assert result["metadata"]["size"] == len(binary_content)
    
    def test_get_content_with_encoding_fallback(self, file_adapter, temp_dir):
        """Test handling files with different encodings."""
        # Create file with latin-1 encoding
        content = "Café résumé naïve"
        file_path = os.path.join(temp_dir, "latin1.txt")
        
        with open(file_path, 'w', encoding='latin-1') as f:
            f.write(content)
        
        # Configure adapter with fallback encodings
        file_adapter.encoding_fallbacks = ['utf-8', 'latin-1']
        
        result = file_adapter.get_content({"source": file_path})
        
        assert result["content"] == content
        assert result["metadata"]["is_binary"] is False
    
    def test_get_content_encoding_fallback_to_binary(self, file_adapter, temp_dir):
        """Test fallback to binary when all encodings fail."""
        # Create file with problematic encoding
        file_path = os.path.join(temp_dir, "problematic.txt")
        with open(file_path, 'wb') as f:
            f.write(b'\x80\x81\x82\x83')  # Invalid UTF-8 sequences
        
        # Configure adapter with limited encodings
        file_adapter.encoding_fallbacks = ['utf-8']
        
        result = file_adapter.get_content({"source": file_path})
        
        # Should read as binary when all text encodings fail
        assert isinstance(result["content"], bytes)
        assert result["metadata"]["is_binary"] is True
    
    def test_get_binary_content(self, file_adapter, temp_dir):
        """Test getting binary content from file."""
        binary_content = b'\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x10'
        file_path = self.create_binary_file(temp_dir, "test.png", binary_content)
        
        result = file_adapter.get_binary_content({"source": file_path})
        
        assert result == binary_content
    
    def test_get_binary_content_from_text_file(self, file_adapter, temp_dir):
        """Test getting binary content from text file."""
        content = "Hello, World!"
        file_path = self.create_test_file(temp_dir, "test.txt", content)
        
        result = file_adapter.get_binary_content({"source": file_path})
        
        assert result == content.encode('utf-8')
    
    def test_get_metadata_without_content(self, file_adapter, temp_dir, sample_text_content):
        """Test getting metadata without reading full content."""
        file_path = self.create_test_file(temp_dir, "metadata-test.md", sample_text_content)
        
        metadata = file_adapter.get_metadata({"source": file_path})
        
        assert metadata["filename"] == "metadata-test.md"
        assert metadata["path"] == file_path
        assert metadata["extension"] == ".md"
        assert metadata["is_binary"] is False
        assert metadata["size"] == len(sample_text_content)
        assert "modified" in metadata
        assert "created" in metadata
        assert "mime_type" in metadata
    
    def test_get_metadata_binary_file(self, file_adapter, temp_dir):
        """Test getting metadata for binary file."""
        binary_content = b'\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR'
        file_path = self.create_binary_file(temp_dir, "test.png", binary_content)
        
        metadata = file_adapter.get_metadata({"source": file_path})
        
        assert metadata["filename"] == "test.png"
        assert metadata["is_binary"] is True
        assert metadata["size"] == len(binary_content)
        assert metadata["mime_type"] == "image/png"
    
    def test_supports_location_real_files(self, file_adapter, temp_dir):
        """Test location support check with real files."""
        # Create test file
        file_path = self.create_test_file(temp_dir, "exists.txt", "content")
        
        # Should support existing file
        assert file_adapter.supports_location({"source": file_path}) is True
        
        # Should not support non-existent file
        nonexistent = os.path.join(temp_dir, "nonexistent.txt")
        assert file_adapter.supports_location({"source": nonexistent}) is False
    
    def test_supports_location_with_base_path(self, file_adapter, temp_dir):
        """Test location support with base path resolution."""
        # Create test file
        self.create_test_file(temp_dir, "relative.txt", "content")
        
        # Should support relative path that exists in base_path
        assert file_adapter.supports_location({"source": "relative.txt"}) is True
        
        # Should not support relative path that doesn't exist
        assert file_adapter.supports_location({"source": "nonexistent.txt"}) is False
    
    def test_json_content_detection(self, file_adapter, temp_dir, sample_json_content):
        """Test JSON content type detection."""
        file_path = self.create_test_file(temp_dir, "test.json", sample_json_content)
        
        result = file_adapter.get_content({"source": file_path})
        
        assert result["content_type"] == "json"
        assert result["metadata"]["mime_type"] == "application/json"
    
    def test_csv_content_detection(self, file_adapter, temp_dir, sample_csv_content):
        """Test CSV content type detection."""
        file_path = self.create_test_file(temp_dir, "test.csv", sample_csv_content)
        
        result = file_adapter.get_content({"source": file_path})
        
        assert result["content_type"] == "csv"
        assert result["metadata"]["is_binary"] is False
    
    def test_html_content_detection(self, file_adapter, temp_dir):
        """Test HTML content type detection."""
        html_content = "<html><head><title>Test</title></head><body>Test</body></html>"
        file_path = self.create_test_file(temp_dir, "test.html", html_content)
        
        result = file_adapter.get_content({"source": file_path})
        
        assert result["content_type"] == "html"
        assert result["metadata"]["mime_type"] == "text/html"
    
    def test_unknown_extension_handling(self, file_adapter, temp_dir):
        """Test handling of files with unknown extensions."""
        content = "This is content in an unknown format"
        file_path = self.create_test_file(temp_dir, "test.unknown", content)
        
        result = file_adapter.get_content({"source": file_path})
        
        # Should still process as text but with detected type
        assert result["content"] == content
        assert result["metadata"]["is_binary"] is False
        assert result["metadata"]["extension"] == ".unknown"
    
    def test_file_uri_resolution_integration(self, file_adapter, temp_dir):
        """Test file URI resolution with real files."""
        file_path = self.create_test_file(temp_dir, "uri-test.txt", "URI content")
        file_uri = f"file://{file_path}"
        
        # Resolve URI
        location_data = file_adapter.resolve_uri(file_uri)
        
        # Should be able to get content using resolved location
        result = file_adapter.get_content(location_data)
        
        assert result["content"] == "URI content"
        assert result["metadata"]["filename"] == "uri-test.txt"
    
    def test_symlink_handling(self, file_adapter, temp_dir):
        """Test symlink handling when enabled."""
        if os.name == 'nt':
            pytest.skip("Symlink test not reliable on Windows")
        
        # Create original file
        original_path = self.create_test_file(temp_dir, "original.txt", "Original content")
        
        # Create symlink
        symlink_path = os.path.join(temp_dir, "symlink.txt")
        try:
            os.symlink(original_path, symlink_path)
        except OSError:
            pytest.skip("Cannot create symlinks on this system")
        
        # Should be able to read through symlink when following is enabled
        file_adapter.follow_symlinks = True
        result = file_adapter.get_content({"source": symlink_path})
        
        assert result["content"] == "Original content"
    
    def test_large_file_handling(self, file_adapter, temp_dir):
        """Test handling of larger files."""
        # Create a moderately large file
        large_content = "A" * 10000  # 10KB
        file_path = self.create_test_file(temp_dir, "large.txt", large_content)
        
        result = file_adapter.get_content({"source": file_path})
        
        assert result["content"] == large_content
        assert result["metadata"]["size"] == 10000
        assert result["metadata"]["is_binary"] is False
    
    def test_empty_file_handling(self, file_adapter, temp_dir):
        """Test handling of empty files."""
        file_path = self.create_test_file(temp_dir, "empty.txt", "")
        
        result = file_adapter.get_content({"source": file_path})
        
        assert result["content"] == ""
        assert result["metadata"]["size"] == 0
        assert result["metadata"]["is_binary"] is False
    
    def test_nested_directory_access(self, file_adapter, temp_dir):
        """Test accessing files in nested directories."""
        # Create nested directory structure
        nested_path = os.path.join(temp_dir, "level1", "level2")
        os.makedirs(nested_path, exist_ok=True)
        
        content = "Nested file content"
        file_path = os.path.join(nested_path, "nested.txt")
        with open(file_path, 'w') as f:
            f.write(content)
        
        result = file_adapter.get_content({"source": file_path})
        
        assert result["content"] == content
        assert result["metadata"]["filename"] == "nested.txt"
    
    def test_error_handling_read_permission(self, file_adapter, temp_dir):
        """Test error handling for files without read permission."""
        if os.name == 'nt':
            pytest.skip("Permission test not reliable on Windows")
        
        file_path = self.create_test_file(temp_dir, "no-read.txt", "content")
        
        # Remove read permission
        try:
            os.chmod(file_path, 0o000)
            
            with pytest.raises(ValueError, match="Error reading file"):
                file_adapter.get_content({"source": file_path})
        finally:
            # Restore permissions for cleanup
            os.chmod(file_path, 0o644)