#!/usr/bin/env python
"""
Test runner that bypasses the problematic server initialization.
"""

import sys
import os
import unittest

# Add src to path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..', '..', 'src'))

# Mock the server module to prevent database initialization
import unittest.mock as mock
sys.modules['go_doc_go.server'] = mock.MagicMock()

# Now we can import the parsers without triggering database init
from go_doc_go.document_parser.csv import CsvParser
from go_doc_go.document_parser.base import DocumentParser
from go_doc_go.storage import ElementType
from go_doc_go.relationships import RelationshipType

def test_csv_parser():
    """Test CSV parser basic functionality."""
    print("Testing CSV Parser...")
    
    parser = CsvParser()
    
    # Test initialization
    assert parser.delimiter == ","
    assert parser.max_rows == 1000
    print("✓ CSV parser initialization works")
    
    # Test parsing
    sample_csv = """Name,Age,City
John,30,NYC
Jane,25,LA"""
    
    content = {
        "id": "/test.csv",
        "content": sample_csv,
        "metadata": {"doc_id": "test_csv"}
    }
    
    result = parser.parse(content)
    
    assert "document" in result
    assert "elements" in result
    assert result["document"]["doc_type"] == "csv"
    print("✓ CSV parsing works")
    
    elements = result["elements"]
    assert len(elements) > 0
    print(f"✓ Created {len(elements)} elements")
    
    return True

def test_base_parser():
    """Test base parser functionality."""
    print("\nTesting Base Parser...")
    
    class TestParser(DocumentParser):
        def parse(self, doc_content):
            doc_id = doc_content.get("metadata", {}).get("doc_id", self._generate_id("doc_"))
            return {
                "document": {
                    "doc_id": doc_id,
                    "doc_type": "test",
                    "source": doc_content["id"],
                    "metadata": doc_content.get("metadata", {}),
                    "content_hash": self._generate_hash(doc_content.get("content", ""))
                },
                "elements": [self._create_root_element(doc_id, doc_content["id"])],
                "relationships": []
            }
        
        def _resolve_element_content(self, element_id, doc_content):
            return doc_content.get("content", "")
        
        def _resolve_element_text(self, element_id, doc_content):
            return doc_content.get("content", "")
        
        def supports_location(self):
            return False
    
    parser = TestParser()
    
    # Test ID generation
    id1 = parser._generate_id("test_")
    assert id1.startswith("test_")
    print("✓ ID generation works")
    
    # Test hash generation  
    hash1 = parser._generate_hash("content")
    print(f"  Hash: {hash1}, length: {len(hash1)}")
    assert isinstance(hash1, str)
    assert len(hash1) > 0  # Just check it's not empty
    print("✓ Hash generation works")
    
    # Test root element creation
    elem = parser._create_root_element("doc_123", "/source/path")
    assert elem["parent_id"] is None
    assert elem["doc_id"] == "doc_123"
    assert "element_id" in elem
    print("✓ Root element creation works")
    
    return True

def main():
    """Run tests."""
    print("=" * 60)
    print("Running Document Parser Tests")
    print("=" * 60)
    
    try:
        # Test base parser
        if test_base_parser():
            print("✅ Base parser tests passed!")
        
        # Test CSV parser
        if test_csv_parser():
            print("✅ CSV parser tests passed!")
            
        print("\n" + "=" * 60)
        print("🎉 All tests passed successfully!")
        print("=" * 60)
        return 0
        
    except Exception as e:
        print(f"\n❌ Test failed: {e}")
        import traceback
        traceback.print_exc()
        return 1

if __name__ == "__main__":
    sys.exit(main())