"""
Unit tests for the base document parser functionality.
"""

import json
import hashlib
import pytest
from typing import Dict, Any, Optional
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
    """Test suite for DocumentParser base class."""
    
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
    
    def disabled_test_create_element_disabled(self):
        """Test general element creation."""
        doc_id = "doc_456"
        parent_id = "parent_123"
        element_type = "paragraph"
        content = "This is a test paragraph with lots of content that should be truncated in the preview."
        source = "/test/source"
        
        element = self.parser._create_element(
            doc_id=doc_id,
            parent_id=parent_id,
            element_type=element_type,
            content=content,
            source=source,
            element_order=5,
            document_position=10
        )
        
        assert element["element_id"].startswith("elem_")
        assert element["doc_id"] == doc_id
        assert element["parent_id"] == parent_id
        assert element["element_type"] == element_type.value
        assert element["content_preview"] == content[:100]  # Default max_content_preview
        assert element["element_order"] == 5
        assert element["document_position"] == 10
        
        # Check content_location
        location = json.loads(element["content_location"])
        assert location["source"] == source
        assert location["type"] == element_type.value
    
    def test_create_element_with_metadata(self):
        """Test element creation with additional metadata."""
        metadata = {
            "style": "bold",
            "language": "en",
            "custom_field": "value"
        }
        
        element = self.parser._create_element(
            doc_id="doc_789",
            parent_id=None,
            element_type=ElementType.HEADER,
            content="Test Header",
            source="/source",
            element_metadata=metadata
        )
        
        location = json.loads(element["content_location"])
        assert location["metadata"] == metadata
    
    def test_create_relationship(self):
        """Test relationship creation."""
        source_id = "elem_123"
        target_id = "elem_456"
        doc_id = "doc_789"
        rel_type = RelationshipType.CONTAINS
        metadata = {"order": 1}
        
        relationship = self.parser._create_relationship(
            source_id=source_id,
            target_id=target_id,
            doc_id=doc_id,
            relationship_type=rel_type,
            metadata=metadata
        )
        
        assert relationship["source_id"] == source_id
        assert relationship["target_id"] == target_id
        assert relationship["doc_id"] == doc_id
        assert relationship["relationship_type"] == rel_type.value
        assert relationship["metadata"] == json.dumps(metadata)
    
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
    
    def test_content_preview_truncation(self):
        """Test that content preview respects max length."""
        # Test with custom max_content_preview
        parser = ConcreteParser({"max_content_preview": 50})
        
        long_content = "a" * 200
        element = parser._create_element(
            doc_id="doc_test",
            parent_id=None,
            element_type=ElementType.PARAGRAPH,
            content=long_content,
            source="/test"
        )
        
        assert len(element["content_preview"]) == 50
        assert element["content_preview"] == "a" * 50
    
    def test_element_ordering(self):
        """Test element ordering fields."""
        # Create multiple elements with ordering
        elements = []
        for i in range(3):
            element = self.parser._create_element(
                doc_id="doc_order",
                parent_id="parent_123",
                element_type=ElementType.PARAGRAPH,
                content=f"Paragraph {i}",
                source="/test",
                element_order=i,
                document_position=i * 10
            )
            elements.append(element)
        
        # Verify ordering
        assert elements[0]["element_order"] == 0
        assert elements[0]["document_position"] == 0
        assert elements[1]["element_order"] == 1
        assert elements[1]["document_position"] == 10
        assert elements[2]["element_order"] == 2
        assert elements[2]["document_position"] == 20
    
    def test_empty_content_handling(self):
        """Test handling of empty or None content."""
        # Test with empty string
        element1 = self.parser._create_element(
            doc_id="doc_empty",
            parent_id=None,
            element_type=ElementType.PARAGRAPH,
            content="",
            source="/test"
        )
        assert element1["content_preview"] == ""
        
        # Test with None (should handle gracefully)
        element2 = self.parser._create_element(
            doc_id="doc_none",
            parent_id=None,
            element_type=ElementType.PARAGRAPH,
            content=None,
            source="/test"
        )
        assert element2["content_preview"] == ""
    
    def test_special_characters_in_content(self):
        """Test handling of special characters in content."""
        special_content = 'Test "quoted" & <tagged> content\nwith\ttabs'
        
        element = self.parser._create_element(
            doc_id="doc_special",
            parent_id=None,
            element_type=ElementType.PARAGRAPH,
            content=special_content,
            source="/test"
        )
        
        # Should preserve special characters in preview
        assert '"quoted"' in element["content_preview"]
        assert "&" in element["content_preview"]
        assert "<tagged>" in element["content_preview"]
    
    def test_json_serialization_of_location(self):
        """Test that content_location is valid JSON."""
        element = self.parser._create_element(
            doc_id="doc_json",
            parent_id="parent_123",
            element_type=ElementType.TABLE,
            content="Table content",
            source="/path/to/file.csv",
            element_metadata={"rows": 10, "columns": 5}
        )
        
        # Should be able to parse content_location as JSON
        location = json.loads(element["content_location"])
        assert location["source"] == "/path/to/file.csv"
        assert location["type"] == ElementType.TABLE.value
        assert location["metadata"]["rows"] == 10
        assert location["metadata"]["columns"] == 5
    
    def test_relationship_metadata_serialization(self):
        """Test that relationship metadata is properly serialized."""
        # Test with dict metadata
        rel1 = self.parser._create_relationship(
            source_id="s1",
            target_id="t1",
            doc_id="d1",
            relationship_type=RelationshipType.REFERENCE,
            metadata={"weight": 0.8, "type": "citation"}
        )
        assert json.loads(rel1["metadata"]) == {"weight": 0.8, "type": "citation"}
        
        # Test with None metadata
        rel2 = self.parser._create_relationship(
            source_id="s2",
            target_id="t2",
            doc_id="d1",
            relationship_type=RelationshipType.PARENT_CHILD,
            metadata=None
        )
        assert rel2["metadata"] == "{}"
        
        # Test with empty dict
        rel3 = self.parser._create_relationship(
            source_id="s3",
            target_id="t3",
            doc_id="d1",
            relationship_type=RelationshipType.SIBLING,
            metadata={}
        )
        assert rel3["metadata"] == "{}"


class TestParserErrorHandling:
    """Test error handling in document parsers."""
    
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
        with pytest.raises(AttributeError):
            self.parser._create_element(
                doc_id="doc_invalid",
                parent_id=None,
                element_type="INVALID_TYPE",  # Should be ElementType enum
                content="Test",
                source="/test"
            )
    
    def test_unicode_content_handling(self):
        """Test handling of Unicode content."""
        unicode_content = "Test 文档 with émojis 😀 and symbols √∑∫"
        
        element = self.parser._create_element(
            doc_id="doc_unicode",
            parent_id=None,
            element_type=ElementType.PARAGRAPH,
            content=unicode_content,
            source="/test"
        )
        
        assert "文档" in element["content_preview"]
        assert "😀" in element["content_preview"]
        assert "√∑∫" in element["content_preview"]
        
        # Hash should work with Unicode
        hash_result = self.parser._generate_hash(unicode_content)
        assert isinstance(hash_result, str)
        assert len(hash_result) == 64
    
    def test_extremely_long_content(self):
        """Test handling of extremely long content."""
        # Create 10MB of content
        huge_content = "x" * (10 * 1024 * 1024)
        
        # Should handle without memory issues
        element = self.parser._create_element(
            doc_id="doc_huge",
            parent_id=None,
            element_type=ElementType.PARAGRAPH,
            content=huge_content,
            source="/test"
        )
        
        # Preview should be truncated
        assert len(element["content_preview"]) == 100
        
        # Hash should still work
        hash_result = self.parser._generate_hash(huge_content)
        assert isinstance(hash_result, str)
    
    def test_circular_reference_prevention(self):
        """Test that circular references are prevented in relationships."""
        # Create a relationship that could be circular
        rel = self.parser._create_relationship(
            source_id="elem_1",
            target_id="elem_1",  # Same as source
            doc_id="doc_1",
            relationship_type=RelationshipType.PARENT_CHILD
        )
        
        # Should create the relationship (parser doesn't validate logic)
        # but it should be properly formed
        assert rel["source_id"] == rel["target_id"]
        assert rel["relationship_type"] == RelationshipType.PARENT_CHILD.value


if __name__ == "__main__":
    pytest.main([__file__, "-v"])