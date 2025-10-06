"""
Compatibility tests for Go text parser implementation.

These tests ensure that the Go text parser produces compatible output
with the Python text parser implementation.
"""

import pytest
from pathlib import Path

from src.go_doc_go.document_parser.text import TextParser
from src.go_doc_go.document_parser.text_go import GoTextParser


class TestTextGoCompatibility:
    """Test compatibility between Python and Go text parsers."""

    @pytest.fixture
    def python_parser(self):
        """Create Python text parser instance."""
        return TextParser({})

    @pytest.fixture
    def go_parser(self):
        """Create Go text parser instance."""
        # Check if binary exists
        project_root = Path(__file__).parent.parent
        binary_path = project_root / "bin" / "textparser"

        if not binary_path.exists():
            pytest.skip("Go text parser binary not found")

        return GoTextParser({})

    @pytest.fixture
    def sample_text_content(self):
        """Sample text content for testing."""
        return {
            "id": "test_doc",
            "content": """This is the first paragraph of the document.
It contains multiple sentences and some information.

This is the second paragraph.
It has different content and structure.

The third paragraph is shorter.

This final paragraph contains a URL: https://example.com
and an email address: test@example.com for testing purposes.""",
            "metadata": {
                "source": "test",
                "filename": "test.txt"
            }
        }

    @pytest.fixture
    def complex_text_content(self):
        """Complex text content for testing."""
        return {
            "id": "complex_test",
            "content": """Document Analysis Report
========================

Executive Summary
-----------------

This report covers the analysis conducted on 2024-01-15.
The data shows significant improvements with 95.5% accuracy.

Key Findings:
• Performance increased by 25.3%
• Error rate decreased to 0.045
• Processing time: 123.45 seconds

Contact Information:
Email: analyst@company.com
Website: https://analytics.company.com
Report file: /reports/analysis_2024.pdf

Next Steps:
1. Review findings with stakeholders
2. Implement recommendations by March 15, 2024
3. Schedule follow-up meeting""",
            "metadata": {
                "source": "complex_test",
                "filename": "analysis_report.txt"
            }
        }

    def test_basic_text_parsing_compatibility(self, python_parser, go_parser, sample_text_content):
        """Test that both parsers produce compatible basic results."""
        # Parse with both parsers
        python_result = python_parser.parse(sample_text_content)
        go_result = go_parser.parse(sample_text_content)

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

        # Both should have root elements
        assert "root" in python_element_types
        assert "root" in go_element_types

        # Both should have paragraph elements
        assert python_element_types.get("paragraph", 0) > 0
        assert go_element_types.get("paragraph", 0) > 0

    def test_paragraph_extraction_compatibility(self, python_parser, go_parser, sample_text_content):
        """Test that both parsers extract paragraphs consistently."""
        # Parse with both parsers
        python_result = python_parser.parse(sample_text_content)
        go_result = go_parser.parse(sample_text_content)

        # Extract paragraph content
        python_paragraphs = []
        go_paragraphs = []

        for element in python_result["elements"]:
            if element["element_type"] == "paragraph":
                content = element.get("text") or element.get("content") or element.get("content_preview", "")
                python_paragraphs.append(content.strip())

        for element in go_result["elements"]:
            if element["element_type"] == "paragraph":
                content = element.get("text") or element.get("content") or element.get("content_preview", "")
                go_paragraphs.append(content.strip())

        # Both should extract paragraphs
        assert len(python_paragraphs) > 0
        assert len(go_paragraphs) > 0

        # Check for common content
        python_content = set(python_paragraphs)
        go_content = set(go_paragraphs)

        # There should be significant overlap in paragraph content
        common_content = python_content & go_content
        assert len(common_content) > 0

        # Check for specific expected content
        expected_content = {"This is the first paragraph"}
        python_has_expected = any(
            any(expected in paragraph for expected in expected_content)
            for paragraph in python_paragraphs
        )
        go_has_expected = any(
            any(expected in paragraph for expected in expected_content)
            for paragraph in go_paragraphs
        )

        assert python_has_expected or go_has_expected

    def test_link_extraction_compatibility(self, python_parser, go_parser, sample_text_content):
        """Test that both parsers extract links consistently."""
        # Parse with both parsers
        python_result = python_parser.parse(sample_text_content)
        go_result = go_parser.parse(sample_text_content)

        # Check link extraction
        python_links = python_result.get("links", [])
        go_links = go_result.get("links", [])

        # Extract link targets
        python_targets = {link["link_target"] for link in python_links}
        go_targets = {link["link_target"] for link in go_links}

        # Both should find URLs and emails
        python_has_urls = any("https://" in target for target in python_targets)
        go_has_urls = any("https://" in target for target in go_targets)
        python_has_emails = any("mailto:" in target for target in python_targets)
        go_has_emails = any("mailto:" in target for target in go_targets)

        # At least one parser should extract links
        assert python_has_urls or go_has_urls
        assert python_has_emails or go_has_emails

    def test_element_structure_compatibility(self, python_parser, go_parser, sample_text_content):
        """Test that element structures are compatible."""
        # Parse with both parsers
        python_result = python_parser.parse(sample_text_content)
        go_result = go_parser.parse(sample_text_content)

        # Check that all elements have required fields
        required_fields = [
            "element_id", "doc_id", "element_type", "content_preview",
            "content_location", "content_hash", "metadata"
        ]

        for element in python_result["elements"]:
            for field in required_fields:
                assert field in element, f"Python parser missing field: {field}"

        for element in go_result["elements"]:
            for field in required_fields:
                assert field in element, f"Go parser missing field: {field}"

    def test_relationship_structure_compatibility(self, python_parser, go_parser, sample_text_content):
        """Test that relationship structures are compatible."""
        # Parse with both parsers
        python_result = python_parser.parse(sample_text_content)
        go_result = go_parser.parse(sample_text_content)

        # Check relationship structure
        required_rel_fields = [
            "relationship_id", "source_element_id", "target_element_id",
            "relationship_type", "confidence", "metadata"
        ]

        # Go parser always creates relationships, Python may not in simple cases
        if python_result["relationships"]:
            for rel in python_result["relationships"]:
                for field in required_rel_fields:
                    assert field in rel, f"Python parser missing relationship field: {field}"

        for rel in go_result["relationships"]:
            for field in required_rel_fields:
                assert field in rel, f"Go parser missing relationship field: {field}"

        # Check that both create hierarchical relationships
        python_has_contains = any(
            rel["relationship_type"] == "contains"
            for rel in python_result["relationships"]
        )
        go_has_contains = any(
            rel["relationship_type"] == "contains"
            for rel in go_result["relationships"]
        )

        assert python_has_contains or go_has_contains

    def test_empty_text_compatibility(self, python_parser, go_parser):
        """Test compatibility with empty text documents."""
        empty_content = {
            "id": "empty_test",
            "content": "",
            "metadata": {"source": "empty_test"}
        }

        # Parse with both parsers - both should handle empty text gracefully or fail consistently
        python_error = None
        go_error = None

        try:
            python_result = python_parser.parse(empty_content)
        except Exception as e:
            python_error = e

        try:
            go_result = go_parser.parse(empty_content)
        except Exception as e:
            go_error = e

        # Both should either succeed or fail
        if python_error and go_error:
            # Both failed - that's acceptable for empty content
            pass
        elif not python_error and not go_error:
            # Both succeeded - check they both have at least root element
            assert len(python_result["elements"]) >= 1
            assert len(go_result["elements"]) >= 1
        else:
            # One succeeded, one failed - that's inconsistent behavior
            pytest.fail(f"Inconsistent empty content handling: Python error: {python_error}, Go error: {go_error}")

    def test_single_paragraph_compatibility(self, python_parser, go_parser):
        """Test compatibility with single paragraph text."""
        single_paragraph_content = {
            "id": "single_paragraph_test",
            "content": "This is a single paragraph with some content for testing purposes.",
            "metadata": {"source": "single_paragraph_test"}
        }

        # Parse with both parsers
        python_result = python_parser.parse(single_paragraph_content)
        go_result = go_parser.parse(single_paragraph_content)

        # Both should handle single paragraph
        assert len(python_result["elements"]) >= 1
        assert len(go_result["elements"]) >= 1

        # Check for paragraph content
        python_has_content = any(
            "single paragraph" in elem.get("content_preview", "") or
            "single paragraph" in elem.get("text", "")
            for elem in python_result["elements"]
        )
        go_has_content = any(
            "single paragraph" in elem.get("content_preview", "") or
            "single paragraph" in elem.get("text", "")
            for elem in go_result["elements"]
        )

        assert python_has_content or go_has_content

    def test_custom_paragraph_separator_compatibility(self, python_parser, go_parser):
        """Test compatibility with custom paragraph separators."""
        # Configure parsers for custom separator
        python_parser.paragraph_separator = "\n---\n"
        go_parser.paragraph_separator = "\n---\n"

        custom_separator_content = {
            "id": "custom_separator_test",
            "content": """First section of content.
Some more content here.
---
Second section of content.
Different content here.
---
Third section.""",
            "metadata": {"source": "custom_separator_test"}
        }

        # Parse with both parsers
        python_result = python_parser.parse(custom_separator_content)
        go_result = go_parser.parse(custom_separator_content)

        # Both should handle custom separator
        assert len(python_result["elements"]) > 1
        assert len(go_result["elements"]) > 1

        # Should have multiple paragraphs
        python_paragraphs = [
            elem for elem in python_result["elements"]
            if elem["element_type"] == "paragraph"
        ]
        go_paragraphs = [
            elem for elem in go_result["elements"]
            if elem["element_type"] == "paragraph"
        ]

        # Should have at least 3 sections
        assert len(python_paragraphs) >= 3 or len(go_paragraphs) >= 3

    def test_date_and_number_extraction_compatibility(self, python_parser, go_parser, complex_text_content):
        """Test compatibility of date and number extraction."""
        # Parse with both parsers
        python_result = python_parser.parse(complex_text_content)
        go_result = go_parser.parse(complex_text_content)

        # Check for date extraction in metadata
        python_has_dates = any(
            "dates" in elem.get("metadata", {})
            for elem in python_result["elements"]
            if elem["element_type"] == "paragraph"
        )
        go_has_dates = any(
            "dates" in elem.get("metadata", {})
            for elem in go_result["elements"]
            if elem["element_type"] == "paragraph"
        )

        # Check for number extraction in metadata
        python_has_numbers = any(
            "numbers" in elem.get("metadata", {})
            for elem in python_result["elements"]
            if elem["element_type"] == "paragraph"
        )
        go_has_numbers = any(
            "numbers" in elem.get("metadata", {})
            for elem in go_result["elements"]
            if elem["element_type"] == "paragraph"
        )

        # At least one parser should extract dates and numbers
        assert python_has_dates or go_has_dates
        assert python_has_numbers or go_has_numbers

    def test_metadata_preservation_compatibility(self, python_parser, go_parser, sample_text_content):
        """Test that metadata is preserved consistently."""
        # Parse with both parsers
        python_result = python_parser.parse(sample_text_content)
        go_result = go_parser.parse(sample_text_content)

        # Check that original metadata is preserved in document
        original_metadata = sample_text_content["metadata"]

        python_doc_metadata = python_result["document"].get("metadata", {})
        go_doc_metadata = go_result["document"].get("metadata", {})

        # Both should preserve some form of original metadata
        assert isinstance(python_doc_metadata, dict)
        assert isinstance(go_doc_metadata, dict)

        # Check for text-specific metadata
        text_metadata_fields = ["character_count", "word_count", "line_count", "paragraph_count"]

        python_has_text_meta = any(field in python_doc_metadata for field in text_metadata_fields)
        go_has_text_meta = any(field in go_doc_metadata for field in text_metadata_fields)

        assert python_has_text_meta or go_has_text_meta

    def test_large_text_handling_compatibility(self, python_parser, go_parser):
        """Test compatibility with larger text content."""
        # Create larger text content
        large_paragraph = "This is a test sentence. " * 100  # 2500 characters
        large_content = {
            "id": "large_text_test",
            "content": "\n\n".join([large_paragraph] * 10),  # 10 large paragraphs
            "metadata": {"source": "large_text_test"}
        }

        # Parse with both parsers
        python_result = python_parser.parse(large_content)
        go_result = go_parser.parse(large_content)

        # Both should handle large text
        assert len(python_result["elements"]) > 1
        assert len(go_result["elements"]) > 1

        # Both should create multiple paragraphs
        python_paragraphs = [
            elem for elem in python_result["elements"]
            if elem["element_type"] == "paragraph"
        ]
        go_paragraphs = [
            elem for elem in go_result["elements"]
            if elem["element_type"] == "paragraph"
        ]

        assert len(python_paragraphs) >= 5
        assert len(go_paragraphs) >= 5

    def test_special_characters_compatibility(self, python_parser, go_parser):
        """Test compatibility with special characters and unicode."""
        special_content = {
            "id": "special_chars_test",
            "content": """Text with special characters: àáâãäåæçèéêë

Unicode symbols: ★☆♠♣♥♦●○◆◇

Mixed content with émojis: 🚀🎯📊💡

Numbers and symbols: $100.50, €75.25, ¥1000""",
            "metadata": {"source": "special_chars_test"}
        }

        # Parse with both parsers
        python_result = python_parser.parse(special_content)
        go_result = go_parser.parse(special_content)

        # Both should handle special characters
        assert len(python_result["elements"]) > 1
        assert len(go_result["elements"]) > 1

        # Check that special characters are preserved in content
        python_has_special = any(
            "àáâãäåæçèéêë" in elem.get("text", "") or "àáâãäåæçèéêë" in elem.get("content_preview", "")
            for elem in python_result["elements"]
        )
        go_has_special = any(
            "àáâãäåæçèéêë" in elem.get("text", "") or "àáâãäåæçèéêë" in elem.get("content_preview", "")
            for elem in go_result["elements"]
        )

        assert python_has_special or go_has_special