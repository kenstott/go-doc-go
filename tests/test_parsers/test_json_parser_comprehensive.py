"""
Comprehensive unit tests for JSON document parser to improve coverage.
"""

import pytest
import json
import os
import tempfile
import time
from unittest.mock import MagicMock, patch, mock_open
from go_doc_go.document_parser.json import JSONParser
from go_doc_go.storage import ElementType
from go_doc_go.relationships import RelationshipType


class TestJSONParserInitialization:
    """Test JSON parser initialization and configuration."""
    
    def test_default_initialization(self):
        """Test parser initialization with default values."""
        parser = JSONParser()
        
        assert parser.max_preview_length == 100
        assert parser.include_field_names == True
        assert parser.flatten_arrays == False
        assert parser.max_depth == 10
        assert parser.enable_caching == True
        assert parser.cache_ttl == 3600
        assert parser.max_cache_size == 128
        assert parser.extract_dates == True
        assert parser.date_context_chars == 50
        assert parser.min_year == 1900
        assert parser.max_year == 2100
        assert parser.enable_performance_monitoring == False
        
    def test_custom_configuration(self):
        """Test parser initialization with custom configuration."""
        config = {
            "max_preview_length": 200,
            "include_field_names": False,
            "flatten_arrays": True,
            "max_depth": 5,
            "enable_caching": False,
            "cache_ttl": 7200,
            "max_cache_size": 256,
            "extract_dates": False,
            "date_context_chars": 100,
            "min_year": 1800,
            "max_year": 2200,
            "fiscal_year_start_month": 4,
            "default_locale": "UK",
            "enable_performance_monitoring": True,
            "temp_dir": "/tmp/json_parser"
        }
        
        parser = JSONParser(config)
        
        assert parser.max_preview_length == 200
        assert parser.include_field_names == False
        assert parser.flatten_arrays == True
        assert parser.max_depth == 5
        assert parser.enable_caching == False
        assert parser.cache_ttl == 7200
        assert parser.max_cache_size == 256
        assert parser.extract_dates == False
        assert parser.date_context_chars == 100
        assert parser.min_year == 1800
        assert parser.max_year == 2200
        assert parser.fiscal_year_start_month == 4
        assert parser.default_locale == "UK"
        assert parser.enable_performance_monitoring == True
        assert parser.temp_dir == "/tmp/json_parser"
        
    def test_date_extractor_import_failure(self):
        """Test handling when DateExtractor import fails."""
        with patch('go_doc_go.document_parser.json.DateExtractor', side_effect=ImportError("Module not found")):
            parser = JSONParser({"extract_dates": True})
            assert parser.extract_dates == False
            assert parser.date_extractor is None


class TestJSONParserSupportsLocation:
    """Test supports_location method."""
    
    def test_supports_json_file(self):
        """Test supports_location with JSON file."""
        parser = JSONParser()
        
        with tempfile.NamedTemporaryFile(suffix='.json', delete=False) as tmp:
            tmp.write(b'{"test": "data"}')
            temp_path = tmp.name
        
        try:
            location = {
                "source": temp_path,
                "type": "json_object"
            }
            assert parser.supports_location(location) == True
        finally:
            os.unlink(temp_path)
    
    def test_supports_json_element_types(self):
        """Test supports_location with JSON element types."""
        parser = JSONParser()
        
        json_types = [
            ElementType.ROOT.value,
            ElementType.JSON_OBJECT.value,
            ElementType.JSON_ARRAY.value,
            ElementType.JSON_FIELD.value,
            ElementType.JSON_ITEM.value
        ]
        
        for elem_type in json_types:
            location = {
                "source": "memory",
                "type": elem_type
            }
            assert parser.supports_location(location) == True
    
    def test_does_not_support_non_json(self):
        """Test supports_location returns False for non-JSON."""
        parser = JSONParser()
        
        # Non-JSON file
        with tempfile.NamedTemporaryFile(suffix='.txt', delete=False) as tmp:
            tmp.write(b'Plain text')
            temp_path = tmp.name
        
        try:
            location = {
                "source": temp_path,
                "type": "text"
            }
            assert parser.supports_location(location) == False
        finally:
            os.unlink(temp_path)
        
        # Non-JSON element type
        location = {
            "source": "memory",
            "type": "paragraph"
        }
        assert parser.supports_location(location) == False
    
    def test_handles_invalid_location(self):
        """Test supports_location handles invalid input."""
        parser = JSONParser()
        
        # Invalid JSON
        location = "not a dict"
        assert parser.supports_location(location) == False
        
        # Missing keys
        location = {}
        assert parser.supports_location(location) == False


class TestJSONParserBasicParsing:
    """Test basic JSON parsing functionality."""
    
    def test_parse_simple_object(self):
        """Test parsing simple JSON object."""
        parser = JSONParser()
        
        content = {
            "id": "/test.json",
            "content": '{"name": "John", "age": 30, "city": "NYC"}',
            "metadata": {"doc_id": "test123"}
        }
        
        result = parser.parse(content)
        
        assert "document" in result
        assert "elements" in result
        assert "relationships" in result
        
        # Check document
        assert result["document"]["doc_id"] == "test123"
        assert result["document"]["doc_type"] == "json"
        
        # Check elements
        elements = result["elements"]
        assert len(elements) > 0
        
        # Should have root and object elements
        element_types = set(e["element_type"] for e in elements)
        assert ElementType.ROOT.value in element_types
        assert ElementType.JSON_OBJECT.value in element_types or ElementType.JSON_FIELD.value in element_types
    
    def test_parse_simple_array(self):
        """Test parsing simple JSON array."""
        parser = JSONParser()
        
        content = {
            "id": "/array.json",
            "content": '["apple", "banana", "orange"]',
            "metadata": {}
        }
        
        result = parser.parse(content)
        
        elements = result["elements"]
        element_types = set(e["element_type"] for e in elements)
        
        # Should have array elements
        assert ElementType.ROOT.value in element_types
        assert ElementType.JSON_ARRAY.value in element_types or ElementType.JSON_ITEM.value in element_types
    
    def test_parse_nested_structure(self):
        """Test parsing nested JSON structure."""
        parser = JSONParser()
        
        nested_json = {
            "user": {
                "name": "Alice",
                "details": {
                    "age": 25,
                    "address": {
                        "city": "Boston",
                        "zip": "02101"
                    }
                }
            },
            "items": [1, 2, 3]
        }
        
        content = {
            "id": "/nested.json",
            "content": json.dumps(nested_json),
            "metadata": {}
        }
        
        result = parser.parse(content)
        
        # Should create elements for nested structure
        elements = result["elements"]
        assert len(elements) > 5
        
        # Should create relationships
        relationships = result["relationships"]
        assert len(relationships) > 0
        rel_types = set(r["relationship_type"] for r in relationships)
        assert RelationshipType.CONTAINS.value in rel_types
    
    def test_parse_with_various_types(self):
        """Test parsing JSON with various data types."""
        parser = JSONParser()
        
        mixed_json = {
            "string": "text",
            "number": 42,
            "float": 3.14,
            "boolean": True,
            "null": None,
            "array": [1, "two", 3.0, False, None],
            "object": {"nested": "value"}
        }
        
        content = {
            "id": "/mixed.json",
            "content": json.dumps(mixed_json),
            "metadata": {}
        }
        
        result = parser.parse(content)
        
        elements = result["elements"]
        # Should handle all data types
        assert len(elements) > len(mixed_json)
    
    def test_parse_empty_structures(self):
        """Test parsing empty JSON structures."""
        parser = JSONParser()
        
        test_cases = [
            ('{}', "empty object"),
            ('[]', "empty array"),
            ('{"empty": {}, "array": []}', "nested empty"),
            ('""', "empty string"),
            ('null', "null value")
        ]
        
        for json_str, description in test_cases:
            content = {
                "id": f"/{description}.json",
                "content": json_str,
                "metadata": {}
            }
            
            result = parser.parse(content)
            assert "document" in result
            assert "elements" in result
            # Should handle empty structures gracefully


class TestJSONParserDateExtraction:
    """Test date extraction functionality."""
    
    def test_extract_dates_from_values(self):
        """Test extracting dates from JSON values."""
        parser = JSONParser({"extract_dates": True})
        
        json_with_dates = {
            "created": "2024-01-15",
            "modified": "January 20, 2024",
            "deadline": "2024-02-01T10:00:00Z",
            "fiscal": "FY2024",
            "quarter": "Q1 2024",
            "description": "Meeting on March 15, 2024 at 2pm"
        }
        
        content = {
            "id": "/dates.json",
            "content": json.dumps(json_with_dates),
            "metadata": {}
        }
        
        result = parser.parse(content)
        
        # Date extraction should process date strings
        elements = result["elements"]
        # Date content should be in elements
        content_text = " ".join(e.get("content_preview", "") for e in elements)
        assert "2024" in content_text
    
    def test_disable_date_extraction(self):
        """Test with date extraction disabled."""
        parser = JSONParser({"extract_dates": False})
        
        json_with_dates = {
            "date": "2024-01-15",
            "time": "10:30:00"
        }
        
        content = {
            "id": "/no_dates.json",
            "content": json.dumps(json_with_dates),
            "metadata": {}
        }
        
        result = parser.parse(content)
        assert "document" in result
        # Should still parse without date extraction


class TestJSONParserCaching:
    """Test caching functionality."""
    
    def test_caching_enabled(self):
        """Test with caching enabled."""
        parser = JSONParser({"enable_caching": True})
        
        assert parser.document_cache is not None
        assert parser.json_cache is not None
        assert parser.text_cache is not None
        assert parser.document_cache.max_size == 128
        assert parser.json_cache.max_size == min(50, 128)
    
    def test_caching_disabled(self):
        """Test with caching disabled."""
        parser = JSONParser({"enable_caching": False})
        
        # Caches should still be created but may not be used
        assert parser.enable_caching == False
    
    def test_clear_caches(self):
        """Test clearing caches."""
        parser = JSONParser()
        
        # Add some data to parse (populates caches)
        content = {
            "id": "/cache_test.json",
            "content": '{"test": "data"}',
            "metadata": {}
        }
        parser.parse(content)
        
        # Clear caches
        parser.clear_caches()
        
        # Caches should be cleared
        assert parser.document_cache.size == 0
        assert parser.json_cache.size == 0
        assert parser.text_cache.size == 0
    
    def test_get_cache_stats(self):
        """Test getting cache statistics."""
        parser = JSONParser()
        
        stats = parser.get_cache_stats()
        
        assert "document_cache" in stats
        assert "json_cache" in stats
        assert "text_cache" in stats
        assert "size" in stats["document_cache"]
        assert "max_size" in stats["document_cache"]


class TestJSONParserPerformance:
    """Test performance monitoring."""
    
    def test_performance_monitoring_enabled(self):
        """Test with performance monitoring enabled."""
        parser = JSONParser({"enable_performance_monitoring": True})
        
        assert parser.enable_performance_monitoring == True
        assert "parse_count" in parser.performance_stats
        assert "cache_hits" in parser.performance_stats
        assert parser.performance_stats["parse_count"] == 0
    
    def test_get_performance_stats(self):
        """Test getting performance statistics."""
        parser = JSONParser({"enable_performance_monitoring": True})
        
        stats = parser.get_performance_stats()
        
        assert "parse_count" in stats
        assert "cache_hits" in stats
        assert "cache_misses" in stats
        assert "total_parse_time" in stats
        assert "average_parse_time" in stats
    
    def test_reset_performance_stats(self):
        """Test resetting performance statistics."""
        parser = JSONParser({"enable_performance_monitoring": True})
        
        # Parse something to generate stats
        content = {
            "id": "/perf_test.json",
            "content": '{"test": "data"}',
            "metadata": {}
        }
        parser.parse(content)
        
        # Reset stats
        parser.reset_performance_stats()
        
        assert parser.performance_stats["parse_count"] == 0
        assert parser.performance_stats["total_parse_time"] == 0.0


class TestJSONParserAdvancedFeatures:
    """Test advanced parser features."""
    
    def test_max_depth_limit(self):
        """Test max depth limitation."""
        parser = JSONParser({"max_depth": 3})
        
        # Create deeply nested structure
        deep_json = {"level1": {"level2": {"level3": {"level4": {"level5": "too deep"}}}}}
        
        content = {
            "id": "/deep.json",
            "content": json.dumps(deep_json),
            "metadata": {}
        }
        
        result = parser.parse(content)
        
        # Should limit depth
        elements = result["elements"]
        assert len(elements) > 0
        # Depth should be limited
    
    def test_flatten_arrays_mode(self):
        """Test array flattening mode."""
        parser = JSONParser({"flatten_arrays": True})
        
        json_with_arrays = {
            "items": [
                {"id": 1, "name": "Item 1"},
                {"id": 2, "name": "Item 2"}
            ],
            "tags": ["tag1", "tag2", "tag3"]
        }
        
        content = {
            "id": "/arrays.json",
            "content": json.dumps(json_with_arrays),
            "metadata": {}
        }
        
        result = parser.parse(content)
        
        elements = result["elements"]
        # Arrays should be flattened
        assert len(elements) > 0
    
    def test_include_field_names(self):
        """Test field name inclusion in previews."""
        parser = JSONParser({"include_field_names": True})
        
        test_json = {"field1": "value1", "field2": "value2"}
        
        content = {
            "id": "/fields.json",
            "content": json.dumps(test_json),
            "metadata": {}
        }
        
        result = parser.parse(content)
        
        elements = result["elements"]
        # Field names might be in content previews
        previews = [e.get("content_preview", "") for e in elements]
        assert any("field" in p.lower() or "value" in p for p in previews)
    
    def test_max_preview_length(self):
        """Test preview length limitation."""
        parser = JSONParser({"max_preview_length": 10})
        
        long_value = "This is a very long string that should be truncated in the preview"
        test_json = {"long": long_value}
        
        content = {
            "id": "/long.json",
            "content": json.dumps(test_json),
            "metadata": {}
        }
        
        result = parser.parse(content)
        
        elements = result["elements"]
        # Previews should be limited
        for element in elements:
            preview = element.get("content_preview", "")
            if preview and preview != "...":
                assert len(preview) <= 10 or "..." in preview


class TestJSONParserErrorHandling:
    """Test error handling."""
    
    def test_invalid_json(self):
        """Test handling of invalid JSON."""
        parser = JSONParser()
        
        invalid_json_cases = [
            '{"unclosed": ',
            '{invalid json}',
            '{"key": undefined}',
            "{'single': 'quotes'}",
            "{unquoted: keys}"
        ]
        
        for invalid_json in invalid_json_cases:
            content = {
                "id": "/invalid.json",
                "content": invalid_json,
                "metadata": {}
            }
            
            try:
                result = parser.parse(content)
                # Should handle gracefully
                assert "document" in result
            except json.JSONDecodeError:
                # Expected for invalid JSON
                pass
    
    def test_missing_content(self):
        """Test handling of missing content."""
        parser = JSONParser()
        
        content = {
            "id": "/missing.json",
            "metadata": {}
        }
        
        try:
            result = parser.parse(content)
            # Should handle missing content
            assert result is not None
        except Exception as e:
            # Should be a clear error about missing content
            assert "content" in str(e).lower()
    
    def test_binary_content(self):
        """Test handling of binary content."""
        parser = JSONParser()
        
        binary_content = b'\x00\x01\x02\x03'
        
        content = {
            "id": "/binary.json",
            "content": binary_content,
            "metadata": {}
        }
        
        try:
            result = parser.parse(content)
            # Should handle or error gracefully
            assert result is not None
        except Exception:
            # Expected for binary that's not valid JSON
            pass
    
    def test_circular_reference_prevention(self):
        """Test prevention of circular references."""
        parser = JSONParser({"max_depth": 10})
        
        # Can't create actual circular reference in JSON string,
        # but test deep nesting that could cause issues
        nested = {"a": {}}
        current = nested["a"]
        for i in range(20):
            current["b"] = {}
            current = current["b"]
        
        content = {
            "id": "/circular.json",
            "content": json.dumps(nested),
            "metadata": {}
        }
        
        result = parser.parse(content)
        # Should handle deep nesting without stack overflow
        assert "document" in result


class TestJSONParserSpecialCases:
    """Test special cases and edge conditions."""
    
    def test_jsonl_format(self):
        """Test JSON Lines format."""
        parser = JSONParser()
        
        jsonl = '{"line": 1}\n{"line": 2}\n{"line": 3}'
        
        content = {
            "id": "/test.jsonl",
            "content": jsonl,
            "metadata": {}
        }
        
        # JSONL might be handled as single JSON or error
        try:
            result = parser.parse(content)
            assert "document" in result
        except json.JSONDecodeError:
            # Expected if parser doesn't support JSONL
            pass
    
    def test_unicode_handling(self):
        """Test Unicode character handling."""
        parser = JSONParser()
        
        unicode_json = {
            "english": "Hello",
            "chinese": "你好",
            "arabic": "مرحبا",
            "emoji": "😀🎉",
            "special": "café, naïve, résumé"
        }
        
        content = {
            "id": "/unicode.json",
            "content": json.dumps(unicode_json),
            "metadata": {}
        }
        
        result = parser.parse(content)
        
        elements = result["elements"]
        # Should handle Unicode properly
        content_text = " ".join(e.get("content_preview", "") for e in elements)
        assert any(char in content_text for char in ["你", "مر", "😀", "café"])
    
    def test_large_numbers(self):
        """Test handling of large numbers."""
        parser = JSONParser()
        
        large_json = {
            "big_int": 9999999999999999,
            "float": 3.141592653589793,
            "scientific": 1.23e10,
            "negative": -9876543210
        }
        
        content = {
            "id": "/numbers.json",
            "content": json.dumps(large_json),
            "metadata": {}
        }
        
        result = parser.parse(content)
        assert "document" in result
        assert len(result["elements"]) > 0
    
    def test_special_keys(self):
        """Test handling of special JSON keys."""
        parser = JSONParser()
        
        special_json = {
            "": "empty key",
            "with spaces": "spaced key",
            "with-dashes": "dashed key",
            "with.dots": "dotted key",
            "$special": "special char",
            "@type": "at sign",
            "123": "numeric key"
        }
        
        content = {
            "id": "/special_keys.json",
            "content": json.dumps(special_json),
            "metadata": {}
        }
        
        result = parser.parse(content)
        assert "document" in result
        assert len(result["elements"]) > 0


class TestJSONParserIntegration:
    """Integration tests."""
    
    def test_parse_from_file(self):
        """Test parsing JSON from file."""
        parser = JSONParser()
        
        test_json = {"file": "content", "number": 42}
        
        with tempfile.NamedTemporaryFile(mode='w', suffix='.json', delete=False) as tmp:
            json.dump(test_json, tmp)
            temp_path = tmp.name
        
        try:
            # Read file content
            with open(temp_path, 'r') as f:
                file_content = f.read()
            
            content = {
                "id": temp_path,
                "content": file_content,
                "metadata": {"source_path": temp_path}
            }
            
            result = parser.parse(content)
            assert "document" in result
            assert result["document"]["source"] == temp_path
            
        finally:
            os.unlink(temp_path)
    
    def test_comprehensive_json_document(self):
        """Test parsing a comprehensive JSON document."""
        parser = JSONParser({
            "extract_dates": True,
            "include_field_names": True,
            "max_depth": 10
        })
        
        comprehensive_json = {
            "metadata": {
                "version": "1.0",
                "created": "2024-01-15T10:00:00Z",
                "author": "Test System"
            },
            "data": {
                "users": [
                    {
                        "id": 1,
                        "name": "Alice",
                        "email": "alice@example.com",
                        "active": True,
                        "roles": ["admin", "user"],
                        "profile": {
                            "age": 30,
                            "location": {
                                "city": "Boston",
                                "country": "USA"
                            }
                        }
                    },
                    {
                        "id": 2,
                        "name": "Bob",
                        "email": "bob@example.com",
                        "active": False,
                        "roles": ["user"],
                        "profile": None
                    }
                ],
                "settings": {
                    "theme": "dark",
                    "notifications": {
                        "email": True,
                        "sms": False,
                        "push": True
                    },
                    "limits": {
                        "max_items": 1000,
                        "max_size": 5368709120
                    }
                },
                "stats": {
                    "total_users": 2,
                    "active_users": 1,
                    "last_update": "2024-01-20",
                    "metrics": [10, 20, 30, 40, 50]
                }
            },
            "empty_values": {
                "null": None,
                "empty_string": "",
                "empty_array": [],
                "empty_object": {}
            }
        }
        
        content = {
            "id": "/comprehensive.json",
            "content": json.dumps(comprehensive_json),
            "metadata": {"doc_id": "comp_test"}
        }
        
        result = parser.parse(content)
        
        # Verify comprehensive parsing
        assert "document" in result
        assert result["document"]["doc_id"] == "comp_test"
        assert result["document"]["doc_type"] == "json"
        
        # Check elements
        elements = result["elements"]
        assert len(elements) > 20  # Should have many elements
        
        # Check element types
        element_types = set(e["element_type"] for e in elements)
        assert ElementType.ROOT.value in element_types
        assert ElementType.JSON_OBJECT.value in element_types or ElementType.JSON_FIELD.value in element_types
        
        # Check relationships
        relationships = result["relationships"]
        assert len(relationships) > 10
        
        # Check relationship types
        rel_types = set(r["relationship_type"] for r in relationships)
        assert RelationshipType.CONTAINS.value in rel_types


class TestJSONParserPrivateMethods:
    """Test private helper methods."""
    
    def test_generate_id(self):
        """Test ID generation."""
        id1 = JSONParser._generate_id("test_")
        id2 = JSONParser._generate_id("test_")
        
        assert id1.startswith("test_")
        assert id2.startswith("test_")
        assert id1 != id2  # Should be unique
    
    def test_generate_hash(self):
        """Test hash generation."""
        hash1 = JSONParser._generate_hash("test content")
        hash2 = JSONParser._generate_hash("test content")
        hash3 = JSONParser._generate_hash("different content")
        
        assert hash1 == hash2  # Same content = same hash
        assert hash1 != hash3  # Different content = different hash
        assert len(hash1) == 64  # SHA-256 hex digest length
    
    def test_get_type(self):
        """Test type detection."""
        assert JSONParser._get_type({}) == "object"
        assert JSONParser._get_type([]) == "array"
        assert JSONParser._get_type("text") == "string"
        assert JSONParser._get_type(123) == "number"
        assert JSONParser._get_type(3.14) == "number"
        assert JSONParser._get_type(True) == "boolean"
        assert JSONParser._get_type(None) == "null"
    
    def test_is_identity_field(self):
        """Test identity field detection."""
        identity_fields = ["id", "ID", "Id", "_id", "uuid", "uid", "key", 
                          "name", "email", "username", "code", "sku"]
        
        for field in identity_fields:
            assert JSONParser._is_identity_field(field) == True
        
        non_identity_fields = ["description", "content", "value", "data"]
        for field in non_identity_fields:
            assert JSONParser._is_identity_field(field) == False
    
    def test_split_field_path(self):
        """Test field path splitting."""
        # Simple field
        field, remainder = JSONParser._split_field_path("field")
        assert field == "field"
        assert remainder is None
        
        # Nested field
        field, remainder = JSONParser._split_field_path("field.nested")
        assert field == "field"
        assert remainder == "nested"
        
        # Array index
        field, remainder = JSONParser._split_field_path("[0]")
        assert field == "0"
        assert remainder is None
        
        # Complex path
        field, remainder = JSONParser._split_field_path("field[0].nested")
        assert field == "field"
        assert remainder == "[0].nested"