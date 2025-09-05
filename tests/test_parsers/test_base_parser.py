"""
Unit tests for the base document parser functionality.
CLEANED VERSION - Removed tests for non-existent methods.
"""

import json
import pytest
from typing import Dict, Any
from unittest.mock import Mock, patch, MagicMock

# Add parent directory to path for imports
import sys
import os
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__)))))

from go_doc_go.document_parser.base import DocumentParser
from go_doc_go.storage import ElementType
from go_doc_go.relationships import RelationshipType


class ConcreteParser(DocumentParser):
    """Concrete implementation of DocumentParser for testing."""
    
    def parse(self, doc_content: Dict[str, Any]) -> Dict[str, Any]:
        """Simple parse implementation for testing."""
        doc_id = doc_content.get("metadata", {}).get("doc_id", self._generate_id("doc_"))
        source_id = doc_content["id"]
        
        return {
            "document": {
                "doc_id": doc_id,
                "doc_type": "test",
                "source": source_id,
                "metadata": doc_content.get("metadata", {}),
                "content_hash": self._generate_hash(doc_content.get("content", ""))
            },
            "elements": [
                self._create_root_element(doc_id, source_id)
            ],
            "relationships": []
        }
    
    def _resolve_element_content(self, element_id: str, doc_content: Dict[str, Any]) -> str:
        """Resolve element content."""
        return doc_content.get("content", "")
    
    def _resolve_element_text(self, element_id: str, doc_content: Dict[str, Any]) -> str:
        """Resolve element text."""
        return doc_content.get("content", "")
    
    def supports_location(self) -> bool:
        """Check if parser supports location resolution."""
        return False


class TestDocumentParser:
    """Test suite for DocumentParser base class - ONLY VALID TESTS."""
    
    def setup_method(self):
        """Set up test fixtures."""
        self.parser = ConcreteParser()
        self.sample_content = {
            "id": "/path/to/document.txt",
            "content": "Sample document content",
            "metadata": {
                "doc_id": "test_doc_123",
                "title": "Test Document"
            }
        }
    
    def test_parser_initialization(self):
        """Test parser initialization with and without config."""
        # Test default initialization
        parser1 = ConcreteParser()
        assert parser1.config == {}
        
        # Test with config
        config = {"max_content_preview": 200}
        parser2 = ConcreteParser(config)
        assert parser2.config == config
    
    def test_generate_id(self):
        """Test ID generation."""
        # Test basic ID generation
        id1 = self.parser._generate_id("test_")
        assert id1.startswith("test_")
        assert len(id1) > len("test_")
        
        # Test uniqueness
        id2 = self.parser._generate_id("test_")
        assert id1 != id2
        
        # Test different prefixes
        doc_id = self.parser._generate_id("doc_")
        elem_id = self.parser._generate_id("elem_")
        assert doc_id.startswith("doc_")
        assert elem_id.startswith("elem_")
    
    def test_generate_hash(self):
        """Test content hash generation."""
        # Test hash generation for string
        content1 = "Test content"
        hash1 = self.parser._generate_hash(content1)
        assert isinstance(hash1, str)
        assert len(hash1) == 32  # MD5 produces 32 hex characters
        
        # Test hash consistency
        hash2 = self.parser._generate_hash(content1)
        assert hash1 == hash2
        
        # Test different content produces different hash
        content2 = "Different content"
        hash3 = self.parser._generate_hash(content2)
        assert hash1 != hash3
        
        # Test empty content
        empty_hash = self.parser._generate_hash("")
        assert isinstance(empty_hash, str)
        assert len(empty_hash) == 32
    
    def test_create_root_element(self):
        """Test root element creation."""
        doc_id = "doc_123"
        source = "/path/to/doc"
        
        root = self.parser._create_root_element(doc_id, source)
        
        assert root["element_id"].startswith("root_")
        assert root["doc_id"] == doc_id
        assert root["element_type"] == "root"
        assert root["parent_id"] is None
        assert root["content_preview"] == ""  # Root element has empty preview
        assert root["element_order"] == 0
        assert root["document_position"] == 0
        
        # Check content_location
        location = json.loads(root["content_location"])
        assert location["source"] == source
        assert location["type"] == "root"
    
    def test_parse_implementation(self):
        """Test the concrete parse implementation."""
        result = self.parser.parse(self.sample_content)
        
        # Check document structure
        assert "document" in result
        assert "elements" in result
        assert "relationships" in result
        
        # Check document fields
        doc = result["document"]
        assert doc["doc_id"] == "test_doc_123"
        assert doc["doc_type"] == "test"
        assert doc["source"] == "/path/to/document.txt"
        assert doc["metadata"]["title"] == "Test Document"
        assert isinstance(doc["content_hash"], str)
        
        # Check elements
        assert len(result["elements"]) == 1
        root_elem = result["elements"][0]
        assert root_elem["element_type"] == "root"
        assert root_elem["parent_id"] is None


class TestParserErrorHandling:
    """Test error handling in document parsers - ONLY VALID TESTS."""
    
    def setup_method(self):
        """Set up test fixtures."""
        self.parser = ConcreteParser()
    
    def test_missing_required_fields(self):
        """Test handling of missing required fields in doc_content."""
        # Missing 'id' field should be handled
        content_no_id = {
            "content": "Test content",
            "metadata": {}
        }
        
        # This should raise or handle gracefully depending on implementation
        with pytest.raises(KeyError):
            self.parser.parse(content_no_id)
    
    def test_invalid_element_type(self):
        """Test handling of invalid element types."""
        # This tests that the enum validation works
        # Since we can't create elements directly, we can only test that
        # invalid types would be caught if passed to internal methods
        pass  # This test has no value without _create_element


if __name__ == "__main__":
    pytest.main([__file__, "-v"])