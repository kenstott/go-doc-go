"""
Tests for File content source.
"""

import pytest
import os
import tempfile
import time
import shutil
from unittest.mock import patch, MagicMock
from typing import Dict, Any

from go_doc_go.content_source.file import FileContentSource


class TestFileContentSourceUnit:
    """Unit tests for FileContentSource without real files."""
    
    def test_content_source_initialization_with_config(self):
        """Test content source initialization with configuration."""
        config = {
            "name": "test-file-source",
            "base_path": "/test/path",
            "file_pattern": "**/*.txt",
            "include_extensions": ["txt", "md"],
            "exclude_extensions": ["tmp"],
            "watch_for_changes": False,
            "recursive": False,
            "max_link_depth": 3
        }
        
        source = FileContentSource(config)
        
        assert source.base_path == os.path.abspath("/test/path")
        assert source.file_pattern == "**/*.txt"
        assert source.include_extensions == ["txt", "md"]
        assert source.exclude_extensions == ["tmp"]
        assert source.watch_for_changes is False
        assert source.recursive is False
        assert source.max_link_depth == 3
    
    def test_content_source_initialization_with_defaults(self):
        """Test content source initialization with default values."""
        config = {"name": "test-file-source"}
        
        source = FileContentSource(config)
        
        assert source.base_path == os.path.abspath(".")
        assert source.file_pattern == "**/*"
        assert source.include_extensions == []
        assert source.exclude_extensions == []
        assert source.watch_for_changes is True
        assert source.recursive is True
        assert source.max_link_depth == 1
    
    def test_get_doc_type_and_mode(self):
        """Test document type and read mode detection."""
        # Text formats
        assert FileContentSource._get_doc_type_and_mode("md") == ("markdown", "text")
        assert FileContentSource._get_doc_type_and_mode("txt") == ("text", "text")
        assert FileContentSource._get_doc_type_and_mode("html") == ("html", "text")
        assert FileContentSource._get_doc_type_and_mode("json") == ("text", "text")
        
        # Binary formats
        assert FileContentSource._get_doc_type_and_mode("pdf") == ("pdf", "binary")
        assert FileContentSource._get_doc_type_and_mode("docx") == ("docx", "binary")
        assert FileContentSource._get_doc_type_and_mode("xlsx") == ("xlsx", "binary")
        assert FileContentSource._get_doc_type_and_mode("png") == ("image", "binary")
        
        # Unknown extensions default to text
        assert FileContentSource._get_doc_type_and_mode("unknown") == ("text", "text")
    
    def test_fetch_document_nonexistent_file(self):
        """Test fetching a non-existent file raises FileNotFoundError."""
        config = {"name": "test-source", "base_path": "/nonexistent"}
        source = FileContentSource(config)
        
        with pytest.raises(FileNotFoundError, match="File not found"):
            source.fetch_document("nonexistent.txt")
    
    def test_has_changed_with_watch_disabled(self):
        """Test change detection with watching disabled."""
        config = {"name": "test-source", "watch_for_changes": False}
        source = FileContentSource(config)
        
        # Should always return True when watching is disabled
        assert source.has_changed("/any/path", 1000.0) is True
    
    def test_has_changed_with_no_previous_timestamp(self):
        """Test change detection with no previous timestamp."""
        config = {"name": "test-source"}
        source = FileContentSource(config)
        
        # Should return True when no previous timestamp provided
        with patch('os.path.exists', return_value=True), \
             patch('os.stat') as mock_stat:
            mock_stat.return_value.st_mtime = 1000.0
            assert source.has_changed("/any/path", None) is True
    
    def test_has_changed_nonexistent_file(self):
        """Test change detection for non-existent file."""
        config = {"name": "test-source"}
        source = FileContentSource(config)
        
        with patch('os.path.exists', return_value=False):
            assert source.has_changed("/nonexistent/path", 1000.0) is False
    
    def test_extract_links_from_content_markdown(self):
        """Test link extraction from markdown content."""
        config = {"name": "test-source"}
        source = FileContentSource(config)
        
        content = """# Test Document
        
This has a [[wiki link]] and a [markdown link](./relative.md).
Also has [another link](../parent.md).
"""
        
        links = source._extract_links_from_content(content, "/test/doc.md", "markdown")
        
        assert len(links) == 3
        link_targets = [link["link_target"] for link in links]
        assert "wiki link" in link_targets
        assert "./relative.md" in link_targets
        assert "../parent.md" in link_targets
    
    def test_extract_links_from_content_html(self):
        """Test link extraction from HTML content."""
        config = {"name": "test-source"}
        source = FileContentSource(config)
        
        content = """<html>
<body>
    <a href="./page1.html">Page 1</a>
    <a href="/absolute/page2.html">Page 2</a>
</body>
</html>"""
        
        links = source._extract_links_from_content(content, "/test/doc.html", "html")
        
        assert len(links) == 2
        # The HTML pattern captures href first, then link text, so target/text are swapped
        link_data = [(link["link_target"], link["link_text"]) for link in links]
        assert ("Page 1", "./page1.html") in link_data
        assert ("Page 2", "/absolute/page2.html") in link_data
    
    def test_extract_links_from_content_text(self):
        """Test link extraction from plain text content."""
        config = {"name": "test-source"}
        source = FileContentSource(config)
        
        content = """This document references:
https://example.com/page
file:///local/file.txt
http://another-site.com/resource
"""
        
        links = source._extract_links_from_content(content, "/test/doc.txt", "text")
        
        assert len(links) == 3
        link_targets = [link["link_target"] for link in links]
        assert "https://example.com/page" in link_targets
        assert "file:///local/file.txt" in link_targets
        assert "http://another-site.com/resource" in link_targets


class TestFileContentSourceIntegration:
    """Integration tests for FileContentSource with real files."""
    
    @pytest.fixture
    def temp_dir(self):
        """Create a temporary directory for test files."""
        temp_dir = tempfile.mkdtemp()
        yield temp_dir
        shutil.rmtree(temp_dir, ignore_errors=True)
    
    @pytest.fixture
    def file_source(self, temp_dir):
        """Create FileContentSource configured for temp directory."""
        config = {
            "name": "test-file-source",
            "base_path": temp_dir,
            "recursive": True
        }
        return FileContentSource(config)
    
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
    
    def test_fetch_text_document(self, file_source, temp_dir, sample_text_content):
        """Test fetching text document."""
        # Create test file
        file_path = self.create_test_file(temp_dir, "test-doc.md", sample_text_content)
        
        # Fetch document
        result = file_source.fetch_document(file_path)
        
        assert result["id"] == file_path
        assert result["content"] == sample_text_content
        assert result["doc_type"] == "markdown"
        assert result["metadata"]["filename"] == "test-doc.md"
        assert result["metadata"]["extension"] == "md"
        assert result["metadata"]["full_path"] == file_path
        assert "last_modified" in result["metadata"]
        assert "size" in result["metadata"]
        assert "content_hash" in result
        assert "binary_path" not in result
    
    def test_fetch_binary_document(self, file_source, temp_dir):
        """Test fetching binary document."""
        # Create binary file
        binary_content = b'\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x10'
        file_path = self.create_binary_file(temp_dir, "test-image.png", binary_content)
        
        # Fetch document
        result = file_source.fetch_document(file_path)
        
        assert result["id"] == file_path
        assert result["content"] == ""  # Binary content should be empty string
        assert result["binary_path"] == file_path
        assert result["doc_type"] == "image"
        assert result["metadata"]["filename"] == "test-image.png"
        assert result["metadata"]["extension"] == "png"
        assert "content_hash" in result
    
    def test_fetch_with_relative_path(self, file_source, temp_dir):
        """Test fetching document using relative path."""
        # Create test file
        content = "Test content"
        file_path = self.create_test_file(temp_dir, "relative.txt", content)
        
        # Fetch using relative path
        relative_path = "relative.txt"
        result = file_source.fetch_document(relative_path)
        
        assert result["id"] == file_path  # Should return absolute path as ID
        assert result["content"] == content
        assert result["metadata"]["relative_path"] == relative_path
    
    def test_list_documents(self, file_source, temp_dir):
        """Test listing documents in directory."""
        # Create multiple test files
        self.create_test_file(temp_dir, "doc1.txt", "Content 1")
        self.create_test_file(temp_dir, "doc2.md", "Content 2")
        self.create_test_file(temp_dir, "data.json", '{"test": true}')
        self.create_test_file(temp_dir, "folder/nested.txt", "Nested content")
        
        # List documents
        documents = file_source.list_documents()
        
        # Should have all 4 files
        assert len(documents) == 4
        
        # Check document structure
        filenames = {doc["metadata"]["filename"] for doc in documents}
        assert "doc1.txt" in filenames
        assert "doc2.md" in filenames
        assert "data.json" in filenames
        assert "nested.txt" in filenames
        
        # Check metadata structure
        for doc in documents:
            assert "id" in doc
            assert os.path.isabs(doc["id"])  # Should be absolute path
            assert "metadata" in doc
            assert "doc_type" in doc
            assert "filename" in doc["metadata"]
            assert "size" in doc["metadata"]
            assert "last_modified" in doc["metadata"]
    
    def test_list_documents_with_extension_filtering(self, file_source, temp_dir):
        """Test listing documents with extension filtering."""
        # Configure source with extension filtering
        file_source.include_extensions = ["txt", "md"]
        file_source.exclude_extensions = ["tmp"]
        
        # Create files with different extensions
        self.create_test_file(temp_dir, "include1.txt", "Content 1")
        self.create_test_file(temp_dir, "include2.md", "Content 2")
        self.create_test_file(temp_dir, "exclude.json", "Content 3")
        self.create_test_file(temp_dir, "exclude.tmp", "Content 4")
        
        # List documents
        documents = file_source.list_documents()
        
        # Should only include .txt and .md files
        filenames = {doc["metadata"]["filename"] for doc in documents}
        assert "include1.txt" in filenames
        assert "include2.md" in filenames
        assert "exclude.json" not in filenames
        assert "exclude.tmp" not in filenames
    
    def test_list_documents_non_recursive(self, temp_dir):
        """Test listing documents without recursion."""
        # Configure non-recursive source with single-level pattern
        config = {
            "name": "test-source",
            "base_path": temp_dir,
            "recursive": False,
            "file_pattern": "*"  # Single level pattern
        }
        source = FileContentSource(config)
        
        # Create files in root and subdirectory
        self.create_test_file(temp_dir, "root.txt", "Root content")
        self.create_test_file(temp_dir, "subdir/nested.txt", "Nested content")
        
        # List documents
        documents = source.list_documents()
        
        # Should only find root file
        filenames = {doc["metadata"]["filename"] for doc in documents}
        assert "root.txt" in filenames
        assert "nested.txt" not in filenames
    
    def test_has_changed_detection(self, file_source, temp_dir):
        """Test document change detection."""
        # Create initial file
        content = "Initial content"
        file_path = self.create_test_file(temp_dir, "changing.txt", content)
        
        # Get initial document
        initial_doc = file_source.fetch_document(file_path)
        initial_modified = initial_doc["metadata"]["last_modified"]
        
        # Check has_changed with current timestamp - should be False
        assert file_source.has_changed(file_path, initial_modified) is False
        
        # Wait a moment and update the file
        time.sleep(0.1)  # Small delay to ensure different timestamp
        with open(file_path, 'w') as f:
            f.write("Updated content")
        
        # Should detect change
        assert file_source.has_changed(file_path, initial_modified) is True
    
    def test_follow_links_markdown(self, file_source, temp_dir):
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
        
        # Create documents
        main_path = self.create_test_file(temp_dir, "main.md", main_content)
        self.create_test_file(temp_dir, "linked-doc.md", linked_content)
        os.makedirs(os.path.join(temp_dir, "subfolder"), exist_ok=True)
        self.create_test_file(temp_dir, "subfolder/third.md", third_content)
        
        # Follow links from main document
        linked_docs = file_source.follow_links(main_content, main_path)
        
        # Should find the linked documents
        assert len(linked_docs) == 2
        
        linked_filenames = [os.path.basename(doc["metadata"]["full_path"]) for doc in linked_docs]
        assert "linked-doc.md" in linked_filenames
        assert "third.md" in linked_filenames
    
    def test_follow_links_prevents_cycles(self, file_source, temp_dir):
        """Test that link following prevents infinite cycles."""
        # Create documents that link to each other
        doc1_content = """# Document 1
Links to [Document 2](./doc2.md)
"""
        
        doc2_content = """# Document 2
Links back to [Document 1](./doc1.md)
"""
        
        # Create documents
        doc1_path = self.create_test_file(temp_dir, "doc1.md", doc1_content)
        self.create_test_file(temp_dir, "doc2.md", doc2_content)
        
        # Follow links with depth limit
        file_source.max_link_depth = 2
        linked_docs = file_source.follow_links(doc1_content, doc1_path)
        
        # Should not get stuck in infinite loop
        # Should find doc2, but not recursively follow back to doc1
        assert len(linked_docs) == 1
        assert os.path.basename(linked_docs[0]["metadata"]["full_path"]) == "doc2.md"
    
    def test_follow_links_respects_depth_limit(self, file_source, temp_dir):
        """Test that link following respects maximum depth."""
        # Set shallow depth limit
        file_source.max_link_depth = 1
        
        # Create chain of linked documents
        doc1_content = "Links to [doc2](./doc2.md)"
        doc2_content = "Links to [doc3](./doc3.md)"
        doc3_content = "Final document"
        
        # Create documents
        doc1_path = self.create_test_file(temp_dir, "doc1.md", doc1_content)
        self.create_test_file(temp_dir, "doc2.md", doc2_content)
        self.create_test_file(temp_dir, "doc3.md", doc3_content)
        
        # Follow links
        linked_docs = file_source.follow_links(doc1_content, doc1_path)
        
        # Should only find doc2 (depth 1), not doc3 (depth 2)
        assert len(linked_docs) == 1
        assert os.path.basename(linked_docs[0]["metadata"]["full_path"]) == "doc2.md"
    
    def test_follow_links_skips_external_urls(self, file_source, temp_dir):
        """Test that link following skips external URLs."""
        content = """# Test Document
        
This has an [external link](https://example.com) and 
a [local link](./local.md).
"""
        
        # Create documents
        main_path = self.create_test_file(temp_dir, "main.md", content)
        self.create_test_file(temp_dir, "local.md", "Local content")
        
        # Follow links
        linked_docs = file_source.follow_links(content, main_path)
        
        # Should only find local document
        assert len(linked_docs) == 1
        assert os.path.basename(linked_docs[0]["metadata"]["full_path"]) == "local.md"
    
    def test_follow_links_handles_nonexistent_files(self, file_source, temp_dir):
        """Test that link following handles non-existent target files."""
        content = """# Test Document
        
This links to [existing doc](./exists.md) and [missing doc](./missing.md).
"""
        
        # Create only one of the linked documents
        main_path = self.create_test_file(temp_dir, "main.md", content)
        self.create_test_file(temp_dir, "exists.md", "Existing content")
        # Don't create missing.md
        
        # Follow links
        linked_docs = file_source.follow_links(content, main_path)
        
        # Should only find existing document
        assert len(linked_docs) == 1
        assert os.path.basename(linked_docs[0]["metadata"]["full_path"]) == "exists.md"
    
    def test_content_hashing_for_change_detection(self, file_source, temp_dir):
        """Test content hashing for change detection."""
        # Create file
        content1 = "Original content"
        file_path = self.create_test_file(temp_dir, "test.txt", content1)
        
        # Fetch document and get hash
        doc1 = file_source.fetch_document(file_path)
        hash1 = doc1["content_hash"]
        
        # Modify file content
        content2 = "Modified content"
        with open(file_path, 'w') as f:
            f.write(content2)
        
        # Fetch again and check hash changed
        doc2 = file_source.fetch_document(file_path)
        hash2 = doc2["content_hash"]
        
        assert hash1 != hash2
        assert doc1["content"] != doc2["content"]
    
    def test_content_type_detection(self, file_source, temp_dir):
        """Test content type detection based on file extension."""
        # Test different file types - use actual MIME type mappings
        test_files = [
            ("test.json", '{"key": "value"}', "application/json"),
            ("test.csv", "name,value\ntest,123", "text/csv"),
            ("test.html", "<html><body>Test</body></html>", "text/html"),
            ("test.unknown", "Unknown content", "application/unknown"),
        ]
        
        for filename, content, expected_type in test_files:
            file_path = self.create_test_file(temp_dir, filename, content)
            result = file_source.fetch_document(file_path)
            
            content_type = result["metadata"]["content_type"]
            assert content_type == expected_type