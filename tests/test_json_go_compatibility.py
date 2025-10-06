"""
Compatibility tests for Go JSON parser implementation.

These tests ensure that the Go JSON parser produces compatible output
with the Python JSON parser implementation.
"""

import json
import os
import tempfile
import pytest
from pathlib import Path

from src.go_doc_go.document_parser.json import JSONParser
from src.go_doc_go.document_parser.json_go import GoJSONParser


class TestJSONGoCompatibility:
    """Test compatibility between Python and Go JSON parsers."""

    @pytest.fixture
    def python_parser(self):
        """Create Python JSON parser instance."""
        return JSONParser({})

    @pytest.fixture
    def go_parser(self):
        """Create Go JSON parser instance."""
        # Check if binary exists
        project_root = Path(__file__).parent.parent
        binary_path = project_root / "bin" / "jsonparser"

        if not binary_path.exists():
            pytest.skip("Go JSON parser binary not found")

        return GoJSONParser({})

    @pytest.fixture
    def sample_json_content(self):
        """Sample JSON content for testing."""
        return {
            "id": "test_doc",
            "content": json.dumps({
                "name": "John Doe",
                "age": 30,
                "city": "New York",
                "hobbies": ["reading", "gaming", "coding"],
                "address": {
                    "street": "123 Main St",
                    "zip": "10001"
                }
            }),
            "metadata": {
                "source": "test",
                "filename": "test.json"
            }
        }

    @pytest.fixture
    def complex_json_content(self):
        """Complex JSON content for testing."""
        return {
            "id": "complex_test",
            "content": json.dumps({
                "company": {
                    "name": "Tech Corp",
                    "employees": [
                        {
                            "id": 1,
                            "name": "Alice",
                            "skills": ["Python", "Go"],
                            "contact": {
                                "email": "alice@example.com"
                            }
                        },
                        {
                            "id": 2,
                            "name": "Bob",
                            "skills": ["JavaScript"],
                            "contact": {
                                "email": "bob@example.com"
                            }
                        }
                    ],
                    "website": "https://techcorp.com"
                }
            }),
            "metadata": {
                "source": "complex_test",
                "filename": "complex.json"
            }
        }

    def test_basic_json_parsing_compatibility(self, python_parser, go_parser, sample_json_content):
        """Test that both parsers produce compatible basic results."""
        # Parse with both parsers
        python_result = python_parser.parse(sample_json_content)
        go_result = go_parser.parse(sample_json_content)

        # Check document structure
        assert python_result["document"]["doc_id"] == go_result["document"]["doc_id"]
        assert python_result["document"]["doc_type"] == go_result["document"]["doc_type"]

        # Check element counts by type
        python_element_types = {}
        for element in python_result["elements"]:
            element_type = element["element_type"]
            python_element_types[element_type] = python_element_types.get(element_type, 0) + 1

        go_element_types = {}
        for element in go_result["elements"]:
            element_type = element["element_type"]
            go_element_types[element_type] = go_element_types.get(element_type, 0) + 1

        # Both should have same element type counts
        assert python_element_types == go_element_types

        # Check relationship counts (Go parser may be more efficient with unidirectional relationships)
        # Python parser creates bidirectional relationships, Go parser creates unidirectional
        go_rel_count = len(go_result["relationships"])
        python_rel_count = len(python_result["relationships"])

        # Go should have at least half the relationships (since Python may create bidirectional)
        assert go_rel_count > 0, "Go parser should create relationships"
        assert python_rel_count >= go_rel_count, "Python parser should have at least as many relationships"

        # In practice, Python creates 2x relationships (bidirectional)
        expected_ratio = python_rel_count / go_rel_count if go_rel_count > 0 else 1
        assert 1.5 <= expected_ratio <= 2.5, f"Relationship ratio should be ~2:1, got {expected_ratio}"

    def test_complex_json_parsing_compatibility(self, python_parser, go_parser, complex_json_content):
        """Test compatibility with complex nested JSON structures."""
        # Parse with both parsers
        python_result = python_parser.parse(complex_json_content)
        go_result = go_parser.parse(complex_json_content)

        # Check element counts
        assert len(python_result["elements"]) == len(go_result["elements"])

        # Check relationship counts with flexibility for bidirectional vs unidirectional
        go_rel_count = len(go_result["relationships"])
        python_rel_count = len(python_result["relationships"])
        assert go_rel_count > 0, "Go parser should create relationships"
        assert python_rel_count >= go_rel_count, "Python should have at least as many relationships"

        # Check for URL extraction
        python_has_links = len(python_result["links"]) > 0
        go_has_links = len(go_result["links"]) > 0

        # Both should extract the website URL
        assert python_has_links == go_has_links

        if python_has_links and go_has_links:
            # Check that both found the same URL
            python_urls = {link["link_target"] for link in python_result["links"]}
            go_urls = {link["link_target"] for link in go_result["links"]}
            assert python_urls == go_urls

    def test_element_structure_compatibility(self, python_parser, go_parser, sample_json_content):
        """Test that element structures are compatible."""
        # Parse with both parsers
        python_result = python_parser.parse(sample_json_content)
        go_result = go_parser.parse(sample_json_content)

        # Check that all elements have required fields
        required_fields = [
            "element_id", "doc_id", "element_type", "content_preview",
            "content_location", "content_hash", "element_order",
            "document_position", "metadata"
        ]

        for element in python_result["elements"]:
            for field in required_fields:
                assert field in element, f"Python parser missing field: {field}"

        for element in go_result["elements"]:
            for field in required_fields:
                assert field in element, f"Go parser missing field: {field}"

    def test_relationship_structure_compatibility(self, python_parser, go_parser, sample_json_content):
        """Test that relationship structures are compatible."""
        # Parse with both parsers
        python_result = python_parser.parse(sample_json_content)
        go_result = go_parser.parse(sample_json_content)

        # Check relationship structure
        required_rel_fields = [
            "relationship_id", "source_element_id", "target_element_id",
            "relationship_type", "confidence", "metadata"
        ]

        for rel in python_result["relationships"]:
            for field in required_rel_fields:
                assert field in rel, f"Python parser missing relationship field: {field}"

        for rel in go_result["relationships"]:
            for field in required_rel_fields:
                assert field in rel, f"Go parser missing relationship field: {field}"

        # Check that all relationships are "contains" type
        python_rel_types = {rel["relationship_type"] for rel in python_result["relationships"]}
        go_rel_types = {rel["relationship_type"] for rel in go_result["relationships"]}

        assert python_rel_types == go_rel_types
        assert "contains" in python_rel_types

    def test_json_path_compatibility(self, python_parser, go_parser, complex_json_content):
        """Test that JSON path generation is compatible."""
        # Parse with both parsers
        python_result = python_parser.parse(complex_json_content)
        go_result = go_parser.parse(complex_json_content)

        # Extract JSON paths from elements
        python_paths = set()
        for element in python_result["elements"]:
            if "json_path" in element.get("metadata", {}):
                python_paths.add(element["metadata"]["json_path"])

        go_paths = set()
        for element in go_result["elements"]:
            if "json_path" in element.get("metadata", {}):
                go_paths.add(element["metadata"]["json_path"])

        # Both should generate similar path structures
        assert len(python_paths) > 0, "Python parser should generate JSON paths"
        assert len(go_paths) > 0, "Go parser should generate JSON paths"

        # Check for common path patterns
        expected_patterns = ["$", "$.company", "$.company.employees", "$.company.employees[0]"]

        for pattern in expected_patterns:
            python_has_pattern = any(pattern in path for path in python_paths)
            go_has_pattern = any(pattern in path for path in go_paths)

            if python_has_pattern:
                assert go_has_pattern, f"Go parser missing path pattern: {pattern}"

    def test_empty_json_compatibility(self, python_parser, go_parser):
        """Test compatibility with empty JSON objects."""
        empty_content = {
            "id": "empty_test",
            "content": "{}",
            "metadata": {"source": "empty_test"}
        }

        # Parse with both parsers
        python_result = python_parser.parse(empty_content)
        go_result = go_parser.parse(empty_content)

        # Both should handle empty JSON gracefully
        assert len(python_result["elements"]) >= 1  # At least root element
        assert len(go_result["elements"]) >= 1  # At least root element

        # Check element types
        python_types = {elem["element_type"] for elem in python_result["elements"]}
        go_types = {elem["element_type"] for elem in go_result["elements"]}

        assert python_types == go_types

    def test_invalid_json_error_handling(self, python_parser, go_parser):
        """Test that both parsers handle invalid JSON similarly."""
        invalid_content = {
            "id": "invalid_test",
            "content": '{"invalid": json}',  # Missing quotes
            "metadata": {"source": "invalid_test"}
        }

        # Both should raise errors for invalid JSON
        with pytest.raises(Exception):
            python_parser.parse(invalid_content)

        with pytest.raises(Exception):
            go_parser.parse(invalid_content)

    def test_large_json_performance_compatibility(self, python_parser, go_parser):
        """Test performance characteristics with larger JSON."""
        # Create a larger JSON structure
        large_data = {
            "users": [
                {
                    "id": i,
                    "name": f"User {i}",
                    "data": {
                        "scores": [j for j in range(10)],
                        "metadata": {"active": True}
                    }
                }
                for i in range(50)  # 50 users with nested data
            ]
        }

        large_content = {
            "id": "large_test",
            "content": json.dumps(large_data),
            "metadata": {"source": "large_test"}
        }

        # Parse with both parsers (should complete without timeout)
        python_result = python_parser.parse(large_content)
        go_result = go_parser.parse(large_content)

        # Check that both parsed the data
        assert len(python_result["elements"]) > 100  # Should have many elements
        assert len(go_result["elements"]) > 100  # Should have many elements

        # Element counts should be similar
        element_count_diff = abs(len(python_result["elements"]) - len(go_result["elements"]))
        assert element_count_diff < 10, "Element counts should be very similar"

    def test_content_location_compatibility(self, python_parser, go_parser, sample_json_content):
        """Test that content location tracking is compatible."""
        # Parse with both parsers
        python_result = python_parser.parse(sample_json_content)
        go_result = go_parser.parse(sample_json_content)

        # Check content location structure
        for element in python_result["elements"]:
            assert "content_location" in element
            location = element["content_location"]
            assert "source" in location
            assert "type" in location

        for element in go_result["elements"]:
            assert "content_location" in element
            location = element["content_location"]
            assert "source" in location
            assert "type" in location

        # Check that location types match element types
        for element in python_result["elements"] + go_result["elements"]:
            location_type = element["content_location"]["type"]
            element_type = element["element_type"]

            # Location type should correspond to element type
            if element_type == "root":
                assert location_type == "root"
            elif element_type in ["json_object", "json_array", "json_field", "json_item"]:
                assert location_type == element_type

    def test_metadata_preservation_compatibility(self, python_parser, go_parser, sample_json_content):
        """Test that metadata is preserved consistently."""
        # Parse with both parsers
        python_result = python_parser.parse(sample_json_content)
        go_result = go_parser.parse(sample_json_content)

        # Check that original metadata is preserved in document
        original_metadata = sample_json_content["metadata"]

        python_doc_metadata = python_result["document"].get("metadata", {})
        go_doc_metadata = go_result["document"].get("metadata", {})

        # Both should preserve original metadata keys
        for key, value in original_metadata.items():
            assert key in python_doc_metadata
            assert key in go_doc_metadata

    @pytest.mark.skipif(
        not os.environ.get("RUN_PERFORMANCE_TESTS"),
        reason="Performance tests disabled by default"
    )
    def test_performance_comparison(self, python_parser, go_parser, complex_json_content):
        """Compare performance between Python and Go parsers."""
        import time

        # Time Python parser
        start_time = time.time()
        for _ in range(10):
            python_result = python_parser.parse(complex_json_content)
        python_time = time.time() - start_time

        # Time Go parser
        start_time = time.time()
        for _ in range(10):
            go_result = go_parser.parse(complex_json_content)
        go_time = time.time() - start_time

        print(f"Python parser time: {python_time:.3f}s")
        print(f"Go parser time: {go_time:.3f}s")
        print(f"Go speedup: {python_time / go_time:.2f}x")

        # Go parser should be at least as fast (allowing for subprocess overhead)
        # In practice, Go should be faster for larger documents
        assert go_time < python_time * 2.0, "Go parser should not be significantly slower"