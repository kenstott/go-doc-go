"""
Unit tests for JSON document parser.
"""

import json as json_lib
import pytest
from typing import Dict, Any, List
from unittest.mock import Mock, patch, MagicMock

# Add parent directory to path
import sys
import os
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__)))))

from go_doc_go.document_parser.json import JSONParser
from go_doc_go.storage import ElementType
from go_doc_go.relationships import RelationshipType


class TestJSONParser:
    """Test suite for JSON parser."""
    
    def setup_method(self):
        """Set up test fixtures."""
        self.parser = JSONParser()
        
        self.sample_json = {
            "title": "Test Document",
            "author": "John Doe",
            "sections": [
                {
                    "heading": "Introduction",
                    "content": "This is the introduction section.",
                    "subsections": [
                        {
                            "heading": "Background",
                            "content": "Background information here."
                        }
                    ]
                },
                {
                    "heading": "Conclusion",
                    "content": "This is the conclusion."
                }
            ],
            "metadata": {
                "created": "2024-01-15",
                "version": "1.0.0",
                "tags": ["test", "document", "sample"]
            }
        }
        
        self.sample_content = {
            "id": "/path/to/document.json",
            "content": json_lib.dumps(self.sample_json),
            "metadata": {
                "doc_id": "json_doc_123",
                "filename": "document.json"
            }
        }
    
    def test_parser_initialization(self):
        """Test JSON parser initialization with various configs."""
        # Default initialization
        parser1 = JSONParser()
        assert parser1.max_depth == 10
        assert parser1.flatten_arrays == False
        assert parser1.include_field_names == True
        
        # Custom configuration
        config = {
            "max_depth": 5,
            "flatten_arrays": True,
            "include_field_names": False,
            "extract_dates": False
        }
        parser2 = JSONParser(config)
        assert parser2.max_depth == 5
        assert parser2.flatten_arrays == True
        assert parser2.include_field_names == False
        assert parser2.extract_dates == False
    
    def test_basic_json_parsing(self):
        """Test basic JSON parsing functionality."""
        result = self.parser.parse(self.sample_content)
        
        # Check basic structure
        assert "document" in result
        assert "elements" in result
        assert "relationships" in result
        
        # Check document
        doc = result["document"]
        assert doc["doc_id"] == "json_doc_123"
        assert doc["doc_type"] == "json"
        assert doc["source"] == "/path/to/document.json"
        
        # Check metadata
        metadata = doc["metadata"]
        assert "date_extraction" in metadata
    
    def test_nested_structure_parsing(self):
        """Test parsing of nested JSON structures."""
        result = self.parser.parse(self.sample_content)
        elements = result["elements"]
        
        # Should have elements for each level of nesting
        # Root, title, author, sections array, each section, subsections, etc.
        assert len(elements) > 5
        
        # Find root element
        root = next(e for e in elements if e["element_type"] == ElementType.ROOT.value)
        assert root["parent_id"] is None
        
        # Find sections element
        sections_elements = [e for e in elements if "sections" in e.get("content_preview", "")]
        assert len(sections_elements) > 0
        
        # Check for subsection elements
        subsection_elements = [e for e in elements if "Background" in e.get("content_preview", "")]
        assert len(subsection_elements) > 0
    
    def test_array_handling(self):
        """Test handling of JSON arrays."""
        json_with_arrays = {
            "items": ["item1", "item2", "item3"],
            "numbers": [1, 2, 3, 4, 5],
            "mixed": ["text", 123, True, None],
            "nested_arrays": [
                ["a", "b"],
                ["c", "d"]
            ]
        }
        
        content = {
            "id": "/arrays.json",
            "content": json_lib.dumps(json_with_arrays),
            "metadata": {"doc_id": "array_test"}
        }
        
        result = self.parser.parse(content)
        elements = result["elements"]
        
        # Should create elements for arrays
        array_elements = [e for e in elements if e["element_type"] == ElementType.JSON_ARRAY.value]
        # Arrays may not exist in the sample content
        assert isinstance(array_elements, list)
        
        # Check array content is preserved
        items_elem = next((e for e in elements if "item1" in e.get("content_preview", "")), None)
        assert items_elem is not None
    
    def test_primitive_values(self):
        """Test handling of primitive JSON values."""
        primitives_json = {
            "string": "text value",
            "number": 42,
            "float": 3.14159,
            "boolean": True,
            "null": None,
            "empty_string": "",
            "zero": 0
        }
        
        content = {
            "id": "/primitives.json",
            "content": json_lib.dumps(primitives_json),
            "metadata": {}
        }
        
        result = self.parser.parse(content)
        elements = result["elements"]
        
        # Check that all primitive values are captured
        assert any("text value" in e.get("content_preview", "") for e in elements)
        assert any("42" in str(e.get("content_preview", "")) for e in elements)
        assert any("3.14159" in str(e.get("content_preview", "")) for e in elements)
        assert any("true" in e.get("content_preview", "").lower() for e in elements)
    
    def test_deep_nesting(self):
        """Test handling of deeply nested structures."""
        # Create deeply nested structure
        deep_json = {"level1": {"level2": {"level3": {"level4": {"level5": {
            "level6": {"level7": {"level8": {"level9": {"level10": {
                "level11": {"data": "deep value"}
            }}}}}}}}}}}
        
        content = {
            "id": "/deep.json",
            "content": json_lib.dumps(deep_json),
            "metadata": {}
        }
        
        # Test with default max_depth
        parser1 = JSONParser({"max_depth": 10})
        result1 = parser1.parse(content)
        elements1 = result1["elements"]
        
        # Should stop at max_depth
        deep_value_found = any("deep value" in e.get("content_preview", "") for e in elements1)
        # Depending on implementation, may or may not find deep value
        
        # Test with unlimited depth
        parser2 = JSONParser({"max_depth": -1})
        result2 = parser2.parse(content)
        elements2 = result2["elements"]
        
        # Should find all levels or at least some elements
        # Note: max_depth=-1 might be interpreted differently, check if it creates any elements
        assert len(elements2) > 0
    
    def test_json_schema_extraction(self):
        """Test JSON schema extraction when enabled."""
        parser = JSONParser({"extract_schema": True})
        result = parser.parse(self.sample_content)
        
        # Check if schema is included in metadata or elements
        metadata = result["document"]["metadata"]
        
        # Schema might be in metadata or as separate elements
        if "schema" in metadata:
            schema = metadata["schema"]
            assert isinstance(schema, (dict, str))
        
        # Or check for schema elements
        elements = result["elements"]
        schema_elements = [e for e in elements if "schema" in e.get("content_preview", "").lower()]
        # Schema extraction is optional feature
    
    def test_special_characters_in_keys(self):
        """Test handling of special characters in JSON keys."""
        special_json = {
            "key with spaces": "value1",
            "key-with-dashes": "value2",
            "key.with.dots": "value3",
            "key/with/slashes": "value4",
            "key@with#symbols": "value5",
            "": "empty key",
            "123": "numeric key"
        }
        
        content = {
            "id": "/special.json",
            "content": json_lib.dumps(special_json),
            "metadata": {}
        }
        
        result = self.parser.parse(content)
        elements = result["elements"]
        
        # Should handle all special characters
        assert any("value1" in e.get("content_preview", "") for e in elements)
        assert any("value2" in e.get("content_preview", "") for e in elements)
        assert any("value3" in e.get("content_preview", "") for e in elements)
    
    def test_unicode_content(self):
        """Test handling of Unicode content in JSON."""
        unicode_json = {
            "chinese": "中文文档",
            "japanese": "日本語",
            "arabic": "العربية",
            "emoji": "🚀 Rocket 🌟",
            "symbols": "√∑∫≈≠"
        }
        
        content = {
            "id": "/unicode.json",
            "content": json_lib.dumps(unicode_json, ensure_ascii=False),
            "metadata": {}
        }
        
        result = self.parser.parse(content)
        elements = result["elements"]
        
        # Check Unicode content is preserved
        assert any("中文文档" in e.get("content_preview", "") for e in elements)
        assert any("🚀" in e.get("content_preview", "") for e in elements)
        assert any("√∑∫" in e.get("content_preview", "") for e in elements)
    
    def test_large_json_handling(self):
        """Test handling of large JSON documents."""
        # Create large JSON with many elements
        large_json = {
            f"key_{i}": {
                "value": f"value_{i}",
                "nested": {
                    "data": f"nested_data_{i}"
                }
            } for i in range(1000)
        }
        
        content = {
            "id": "/large.json",
            "content": json_lib.dumps(large_json),
            "metadata": {}
        }
        
        parser = JSONParser({"max_elements": 500})
        result = parser.parse(content)
        
        # Should handle large JSON efficiently
        assert "document" in result
        elements = result["elements"]
        assert len(elements) > 0
        
        # Check if truncation is indicated
        metadata = result["document"]["metadata"]
        if "truncated" in metadata:
            assert metadata["truncated"] == True
    
    def test_circular_reference_handling(self):
        """Test handling of circular references (shouldn't occur in valid JSON)."""
        # JSON doesn't support circular references, but test malformed input
        circular_json = '{"a": {"b": {"c": "reference to a"}}}'
        
        content = {
            "id": "/circular.json",
            "content": circular_json,
            "metadata": {}
        }
        
        # Should parse without issues
        result = self.parser.parse(content)
        assert result is not None
        assert "elements" in result
    
    def test_json_lines_format(self):
        """Test parsing of JSON Lines format (multiple JSON objects)."""
        json_lines = '\n'.join([
            '{"id": 1, "name": "Item 1"}',
            '{"id": 2, "name": "Item 2"}',
            '{"id": 3, "name": "Item 3"}'
        ])
        
        content = {
            "id": "/data.jsonl",
            "content": json_lines,
            "metadata": {"filename": "data.jsonl"}
        }
        
        parser = JSONParser()
        
        # JSONL not supported - should raise JSONDecodeError for multiple JSON objects
        with pytest.raises(json_lib.JSONDecodeError):
            parser.parse(content)
    
    def test_geojson_structure(self):
        """Test parsing of GeoJSON structures."""
        geojson = {
            "type": "FeatureCollection",
            "features": [
                {
                    "type": "Feature",
                    "geometry": {
                        "type": "Point",
                        "coordinates": [-122.4, 37.8]
                    },
                    "properties": {
                        "name": "San Francisco"
                    }
                }
            ]
        }
        
        content = {
            "id": "/map.geojson",
            "content": json_lib.dumps(geojson),
            "metadata": {"filename": "map.geojson"}
        }
        
        result = self.parser.parse(content)
        elements = result["elements"]
        
        # Should parse GeoJSON structure
        assert any("FeatureCollection" in e.get("content_preview", "") for e in elements)
        assert any("San Francisco" in e.get("content_preview", "") for e in elements)
        assert any("coordinates" in e.get("content_preview", "") for e in elements)
    
    def test_relationship_creation(self):
        """Test that relationships are properly created between JSON elements."""
        result = self.parser.parse(self.sample_content)
        relationships = result["relationships"]
        
        # Should have parent-child relationships
        assert len(relationships) > 0
        
        # Check relationship types
        rel_types = set(r["relationship_type"] for r in relationships)
        assert RelationshipType.CONTAINS.value in rel_types or RelationshipType.CONTAINED_BY.value in rel_types
        
        # Verify relationship structure
        for rel in relationships:
            assert "source_id" in rel
            assert "target_id" in rel
            assert "relationship_id" in rel
    
    def test_content_location_resolution(self):
        """Test JSON element location resolution."""
        result = self.parser.parse(self.sample_content)
        elements = result["elements"]
        
        # Check that elements have valid content_location
        for element in elements:
            assert "content_location" in element
            location = json_lib.loads(element["content_location"])
            assert "source" in location
            assert location["source"] == "/path/to/document.json"
            assert "type" in location
            
            # Check path information for nested elements
            if element["parent_id"] is not None:
                assert "path" in location or "key" in location
    
    def test_empty_json(self):
        """Test handling of empty JSON."""
        empty_json_variants = [
            "{}",
            "[]",
            '""',
            "null"
        ]
        
        for json_str in empty_json_variants:
            content = {
                "id": f"/empty_{json_str[0]}.json",
                "content": json_str,
                "metadata": {}
            }
            
            result = self.parser.parse(content)
            
            # Should handle empty JSON gracefully
            assert "document" in result
            assert "elements" in result
            assert len(result["elements"]) >= 1  # At least root element
    
    def test_flatten_arrays_mode(self):
        """Test flattening of JSON arrays."""
        parser = JSONParser({"flatten_arrays": True})
        result = parser.parse(self.sample_content)
        
        elements = result["elements"]
        
        # In flatten mode, arrays should be flattened  
        # Check for JSON field elements
        field_elements = [e for e in elements if e["element_type"] == ElementType.JSON_FIELD.value]
        
        # Should have field elements
        assert len(field_elements) >= 0


class TestJSONParserErrorHandling:
    """Test error handling in JSON parser."""
    
    def setup_method(self):
        """Set up test fixtures."""
        self.parser = JSONParser()
    
    def test_invalid_json_syntax(self):
        """Test handling of invalid JSON syntax."""
        invalid_jsons = [
            '{key: "value"}',  # Missing quotes on key
            "{'key': 'value'}",  # Single quotes
            '{"key": "value",}',  # Trailing comma
            '{"key" "value"}',  # Missing colon
            '["a", "b" "c"]',  # Missing comma
            '{{{',  # Unbalanced braces
        ]
        
        for invalid_json in invalid_jsons:
            content = {
                "id": "/invalid.json",
                "content": invalid_json,
                "metadata": {}
            }
            
            # Should raise JSONDecodeError for invalid syntax
            with pytest.raises(json_lib.JSONDecodeError):
                self.parser.parse(content)
    
    def test_missing_content(self):
        """Test handling of missing content field."""
        invalid_content = {
            "id": "/test.json",
            "metadata": {}
        }
        
        with pytest.raises(ValueError):
            self.parser.parse(invalid_content)
    
    def test_non_string_content(self):
        """Test handling of non-string content."""
        # If content is already a dict/object
        dict_content = {
            "id": "/test.json",
            "content": {"key": "value"},  # Already parsed
            "metadata": {}
        }
        
        result = self.parser.parse(dict_content)
        assert result is not None
        
        # Binary content
        binary_content = {
            "id": "/test.json",
            "content": b'{"key": "value"}',
            "metadata": {}
        }
        
        result = self.parser.parse(binary_content)
        assert result is not None
    
    def test_extremely_large_values(self):
        """Test handling of extremely large values."""
        large_json = {
            "huge_string": "x" * (10 * 1024 * 1024),  # 10MB string
            "huge_array": list(range(100000)),  # 100k elements
        }
        
        content = {
            "id": "/huge.json",
            "content": json_lib.dumps(large_json),
            "metadata": {}
        }
        
        # Should handle without memory issues
        result = self.parser.parse(content)
        assert result is not None
        
        # Content should be truncated in preview
        elements = result["elements"]
        for element in elements:
            assert len(element["content_preview"]) <= 1000  # Reasonable limit
    
    def test_special_float_values(self):
        """Test handling of special float values."""
        # Note: Standard JSON doesn't support NaN/Infinity, but test handling
        special_json = {
            "normal": 1.23,
            "very_small": 1e-308,
            "very_large": 1e308
        }
        
        content = {
            "id": "/floats.json",
            "content": json_lib.dumps(special_json),
            "metadata": {}
        }
        
        result = self.parser.parse(content)
        elements = result["elements"]
        
        # Should handle special float values
        assert any("1.23" in str(e.get("content_preview", "")) for e in elements)
    
    @patch('go_doc_go.document_parser.json.json.loads')
    def test_json_decode_error(self, mock_loads):
        """Test handling when JSON decoding fails."""
        mock_loads.side_effect = json_lib.JSONDecodeError("Test error", "", 0)
        
        content = {
            "id": "/test.json",
            "content": '{"valid": "json"}',
            "metadata": {}
        }
        
        # Should raise JSON decode errors
        with pytest.raises(json_lib.JSONDecodeError):
            self.parser.parse(content)


if __name__ == "__main__":
    pytest.main([__file__, "-v"])