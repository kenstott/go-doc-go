"""
Simple test to verify parser functionality without complex imports.
"""

import sys
import os
import json
import hashlib

# Add src to path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..', '..', 'src'))

# Import only the specific classes we need
from go_doc_go.document_parser.base import DocumentParser
from go_doc_go.storage import ElementType
from go_doc_go.relationships import RelationshipType


class ConcreteParser(DocumentParser):
    """Concrete implementation for testing."""
    
    def parse(self, doc_content):
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


def test_parser_basic():
    """Test basic parser functionality."""
    parser = ConcreteParser()
    
    # Test ID generation
    id1 = parser._generate_id("test_")
    assert id1.startswith("test_")
    assert len(id1) > len("test_")
    
    id2 = parser._generate_id("test_")
    assert id1 != id2
    
    print("✓ ID generation works")
    
    # Test hash generation
    hash1 = parser._generate_hash("test content")
    assert isinstance(hash1, str)
    assert len(hash1) == 64
    
    hash2 = parser._generate_hash("test content")
    assert hash1 == hash2
    
    hash3 = parser._generate_hash("different content")
    assert hash1 != hash3
    
    print("✓ Hash generation works")
    
    # Test element creation
    element = parser._create_element(
        doc_id="doc_123",
        parent_id="parent_456",
        element_type=ElementType.PARAGRAPH,
        content="Test paragraph content",
        source="/test/source"
    )
    
    assert element["doc_id"] == "doc_123"
    assert element["parent_id"] == "parent_456"
    assert element["element_type"] == ElementType.PARAGRAPH.value
    assert element["content_preview"] == "Test paragraph content"
    
    print("✓ Element creation works")
    
    # Test document parsing
    doc_content = {
        "id": "/test/document.txt",
        "content": "Sample content",
        "metadata": {
            "doc_id": "test_doc",
            "title": "Test Document"
        }
    }
    
    result = parser.parse(doc_content)
    
    assert result["document"]["doc_id"] == "test_doc"
    assert result["document"]["doc_type"] == "test"
    assert result["document"]["source"] == "/test/document.txt"
    assert len(result["elements"]) == 1
    
    print("✓ Document parsing works")
    
    print("\n✅ All basic parser tests passed!")


if __name__ == "__main__":
    test_parser_basic()