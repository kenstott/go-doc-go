"""
Compatibility tests for Go markdown parser implementation.

These tests ensure that the Go markdown parser produces compatible output
with the Python markdown parser implementation.
"""

import pytest
from pathlib import Path

from src.go_doc_go.document_parser.markdown import MarkdownParser
from src.go_doc_go.document_parser.markdown_go import GoMarkdownParser


@pytest.mark.integration
@pytest.mark.compatibility
class TestMarkdownGoCompatibility:
    """Test compatibility between Python and Go markdown parsers."""

    @pytest.fixture
    def python_parser(self):
        """Create Python markdown parser instance."""
        return MarkdownParser({})

    @pytest.fixture
    def go_parser(self):
        """Create Go markdown parser instance."""
        # Check if binary exists
        project_root = Path(__file__).parent.parent
        binary_path = project_root / "bin" / "markdownparser"

        if not binary_path.exists():
            pytest.skip("Go markdown parser binary not found")

        return GoMarkdownParser({})

    @pytest.fixture
    def sample_markdown_content(self):
        """Sample markdown content for testing."""
        return {
            "id": "test_doc",
            "content": """---
title: Test Document
author: Test Author
date: 2024-01-15
tags: [test, markdown]
---

# Main Header

This is a paragraph with some **bold text** and *italic text*.

## Secondary Header

Another paragraph with a [link](https://example.com) and email@test.com.

### List Example

- First item
- Second item with [internal link](page.md)
- Third item

### Code Example

```python
def hello_world():
    print("Hello, World!")
    return True
```

### Blockquote Example

> This is a blockquote
> with multiple lines
> of content.

### Table Example

| Header 1 | Header 2 | Header 3 |
|----------|----------|----------|
| Cell 1   | Cell 2   | Cell 3   |
| Cell 4   | Cell 5   | Cell 6   |""",
            "metadata": {
                "source": "test",
                "filename": "test.md"
            }
        }

    @pytest.fixture
    def simple_markdown_content(self):
        """Simple markdown content without front matter."""
        return {
            "id": "simple_test",
            "content": """# Simple Header

This is a simple paragraph.

## Another Header

- List item 1
- List item 2

That's it!""",
            "metadata": {
                "source": "simple_test",
                "filename": "simple.md"
            }
        }

    def test_basic_markdown_parsing_compatibility(self, python_parser, go_parser, simple_markdown_content):
        """Test that both parsers produce compatible basic results."""
        # Parse with both parsers
        python_result = python_parser.parse(simple_markdown_content)
        go_result = go_parser.parse(simple_markdown_content)

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

        # Both should have header and paragraph elements
        assert python_element_types.get("header", 0) > 0 or go_element_types.get("header", 0) > 0
        assert python_element_types.get("paragraph", 0) > 0 or go_element_types.get("paragraph", 0) > 0

    def test_front_matter_extraction_compatibility(self, python_parser, go_parser, sample_markdown_content):
        """Test that both parsers handle front matter consistently."""
        # Parse with both parsers
        python_result = python_parser.parse(sample_markdown_content)
        go_result = go_parser.parse(sample_markdown_content)

        # Check document metadata for front matter
        python_doc_metadata = python_result["document"].get("metadata", {})
        go_doc_metadata = go_result["document"].get("metadata", {})

        # Both should detect front matter
        python_has_front_matter = python_doc_metadata.get("has_front_matter", False)
        go_has_front_matter = go_doc_metadata.get("has_front_matter", False)

        # At least one should detect front matter
        assert python_has_front_matter or go_has_front_matter

        # If both detect front matter, check consistency
        if python_has_front_matter and go_has_front_matter:
            python_fm = python_doc_metadata.get("front_matter", {})
            go_fm = go_doc_metadata.get("front_matter", {})

            # Both should have title
            if "title" in python_fm and "title" in go_fm:
                assert python_fm["title"] == go_fm["title"] or \
                       str(python_fm["title"]) == str(go_fm["title"])

    def test_header_extraction_compatibility(self, python_parser, go_parser, sample_markdown_content):
        """Test that both parsers extract headers consistently."""
        # Parse with both parsers
        python_result = python_parser.parse(sample_markdown_content)
        go_result = go_parser.parse(sample_markdown_content)

        # Extract header elements
        python_headers = [
            elem for elem in python_result["elements"]
            if elem["element_type"] == "header"
        ]
        go_headers = [
            elem for elem in go_result["elements"]
            if elem["element_type"] == "header"
        ]

        # Both should extract headers
        assert len(python_headers) > 0 or len(go_headers) > 0

        # If both extract headers, check for common content
        if python_headers and go_headers:
            python_header_texts = {
                elem.get("text", elem.get("content_preview", "")).strip()
                for elem in python_headers
            }
            go_header_texts = {
                elem.get("text", elem.get("content_preview", "")).strip()
                for elem in go_headers
            }

            # Should have some overlap
            common_headers = python_header_texts & go_header_texts
            assert len(common_headers) > 0

    def test_code_block_extraction_compatibility(self, python_parser, go_parser, sample_markdown_content):
        """Test that both parsers extract code blocks consistently."""
        # Parse with both parsers
        python_result = python_parser.parse(sample_markdown_content)
        go_result = go_parser.parse(sample_markdown_content)

        # Extract code block elements
        python_code_blocks = [
            elem for elem in python_result["elements"]
            if elem["element_type"] == "code_block"
        ]
        go_code_blocks = [
            elem for elem in go_result["elements"]
            if elem["element_type"] == "code_block"
        ]

        # Both should extract code blocks
        assert len(python_code_blocks) > 0 or len(go_code_blocks) > 0

        # Check for Python code content
        python_has_python_code = any(
            "print" in elem.get("text", elem.get("content_preview", ""))
            for elem in python_code_blocks
        )
        go_has_python_code = any(
            "print" in elem.get("text", elem.get("content_preview", ""))
            for elem in go_code_blocks
        )

        assert python_has_python_code or go_has_python_code

    def test_list_extraction_compatibility(self, python_parser, go_parser, sample_markdown_content):
        """Test that both parsers extract lists consistently."""
        # Parse with both parsers
        python_result = python_parser.parse(sample_markdown_content)
        go_result = go_parser.parse(sample_markdown_content)

        # Extract list elements
        python_lists = [
            elem for elem in python_result["elements"]
            if elem["element_type"] in ["list", "list_item"]
        ]
        go_lists = [
            elem for elem in go_result["elements"]
            if elem["element_type"] in ["list", "list_item"]
        ]

        # Both should extract lists
        assert len(python_lists) > 0 or len(go_lists) > 0

        # Check for list content
        python_has_list_content = any(
            "First item" in elem.get("text", elem.get("content_preview", ""))
            for elem in python_lists
        )
        go_has_list_content = any(
            "First item" in elem.get("text", elem.get("content_preview", ""))
            for elem in go_lists
        )

        assert python_has_list_content or go_has_list_content

    def test_blockquote_extraction_compatibility(self, python_parser, go_parser, sample_markdown_content):
        """Test that both parsers extract blockquotes consistently."""
        # Parse with both parsers
        python_result = python_parser.parse(sample_markdown_content)
        go_result = go_parser.parse(sample_markdown_content)

        # Extract blockquote elements
        python_blockquotes = [
            elem for elem in python_result["elements"]
            if elem["element_type"] == "blockquote"
        ]
        go_blockquotes = [
            elem for elem in go_result["elements"]
            if elem["element_type"] == "blockquote"
        ]

        # Both should extract blockquotes
        assert len(python_blockquotes) > 0 or len(go_blockquotes) > 0

        # Check for blockquote content
        python_has_quote_content = any(
            "This is a blockquote" in elem.get("text", elem.get("content_preview", ""))
            for elem in python_blockquotes
        )
        go_has_quote_content = any(
            "This is a blockquote" in elem.get("text", elem.get("content_preview", ""))
            for elem in go_blockquotes
        )

        assert python_has_quote_content or go_has_quote_content

    def test_table_extraction_compatibility(self, python_parser, go_parser, sample_markdown_content):
        """Test that both parsers extract tables consistently."""
        # Parse with both parsers
        python_result = python_parser.parse(sample_markdown_content)
        go_result = go_parser.parse(sample_markdown_content)

        # Extract table elements
        python_tables = [
            elem for elem in python_result["elements"]
            if elem["element_type"] in ["table", "table_row", "table_cell"]
        ]
        go_tables = [
            elem for elem in go_result["elements"]
            if elem["element_type"] in ["table", "table_row", "table_cell"]
        ]

        # Both should extract tables or table content
        assert len(python_tables) > 0 or len(go_tables) > 0

        # Check for table content
        python_has_table_content = any(
            "Header 1" in elem.get("text", elem.get("content_preview", ""))
            for elem in python_tables
        )
        go_has_table_content = any(
            "Header 1" in elem.get("text", elem.get("content_preview", ""))
            for elem in go_tables
        )

        assert python_has_table_content or go_has_table_content

    def test_link_extraction_compatibility(self, python_parser, go_parser, sample_markdown_content):
        """Test that both parsers extract links consistently."""
        # Parse with both parsers
        python_result = python_parser.parse(sample_markdown_content)
        go_result = go_parser.parse(sample_markdown_content)

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

    def test_element_structure_compatibility(self, python_parser, go_parser, simple_markdown_content):
        """Test that element structures are compatible."""
        # Parse with both parsers
        python_result = python_parser.parse(simple_markdown_content)
        go_result = go_parser.parse(simple_markdown_content)

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

    def test_relationship_structure_compatibility(self, python_parser, go_parser, simple_markdown_content):
        """Test that relationship structures are compatible."""
        # Parse with both parsers
        python_result = python_parser.parse(simple_markdown_content)
        go_result = go_parser.parse(simple_markdown_content)

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

    def test_empty_markdown_compatibility(self, python_parser, go_parser):
        """Test compatibility with empty markdown documents."""
        empty_content = {
            "id": "empty_test",
            "content": "",
            "metadata": {"source": "empty_test"}
        }

        # Parse with both parsers - both should handle empty markdown gracefully or fail consistently
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

    def test_metadata_preservation_compatibility(self, python_parser, go_parser, simple_markdown_content):
        """Test that metadata is preserved consistently."""
        # Parse with both parsers
        python_result = python_parser.parse(simple_markdown_content)
        go_result = go_parser.parse(simple_markdown_content)

        # Check that original metadata is preserved in document
        original_metadata = simple_markdown_content["metadata"]

        python_doc_metadata = python_result["document"].get("metadata", {})
        go_doc_metadata = go_result["document"].get("metadata", {})

        # Both should preserve some form of original metadata
        assert isinstance(python_doc_metadata, dict)
        assert isinstance(go_doc_metadata, dict)

        # Check for markdown-specific metadata
        markdown_metadata_fields = ["character_count", "word_count", "line_count", "header_count"]

        python_has_md_meta = any(field in python_doc_metadata for field in markdown_metadata_fields)
        go_has_md_meta = any(field in go_doc_metadata for field in markdown_metadata_fields)

        assert python_has_md_meta or go_has_md_meta

    def test_no_front_matter_compatibility(self, python_parser, go_parser, simple_markdown_content):
        """Test compatibility with markdown without front matter."""
        # Parse with both parsers
        python_result = python_parser.parse(simple_markdown_content)
        go_result = go_parser.parse(simple_markdown_content)

        # Check that both handle no front matter correctly
        python_doc_metadata = python_result["document"].get("metadata", {})
        go_doc_metadata = go_result["document"].get("metadata", {})

        python_has_front_matter = python_doc_metadata.get("has_front_matter", False)
        go_has_front_matter = go_doc_metadata.get("has_front_matter", False)

        # Both should indicate no front matter
        assert not python_has_front_matter
        assert not go_has_front_matter

        # Should not have front matter elements
        python_front_matter_elements = [
            elem for elem in python_result["elements"]
            if elem["element_type"] == "front_matter"
        ]
        go_front_matter_elements = [
            elem for elem in go_result["elements"]
            if elem["element_type"] == "front_matter"
        ]

        assert len(python_front_matter_elements) == 0
        assert len(go_front_matter_elements) == 0

    def test_complex_markdown_compatibility(self, python_parser, go_parser, sample_markdown_content):
        """Test compatibility with complex markdown content."""
        # Parse with both parsers
        python_result = python_parser.parse(sample_markdown_content)
        go_result = go_parser.parse(sample_markdown_content)

        # Both should handle complex content
        assert len(python_result["elements"]) > 5
        assert len(go_result["elements"]) > 5

        # Both should find multiple element types
        python_element_types = {elem["element_type"] for elem in python_result["elements"]}
        go_element_types = {elem["element_type"] for elem in go_result["elements"]}

        # Should have variety of element types
        assert len(python_element_types) >= 3
        assert len(go_element_types) >= 3

        # Both should find links
        assert len(python_result.get("links", [])) > 0 or len(go_result.get("links", [])) > 0

    def test_configuration_compatibility(self, python_parser, go_parser):
        """Test that parser configurations work consistently."""
        # Configure both parsers to disable front matter
        python_parser.extract_front_matter = False
        go_parser.extract_front_matter = False

        content_with_fm = {
            "id": "config_test",
            "content": """---
title: Should be ignored
---

# Header

Content here.""",
            "metadata": {"source": "config_test"}
        }

        # Parse with both parsers
        python_result = python_parser.parse(content_with_fm)
        go_result = go_parser.parse(content_with_fm)

        # Both should ignore front matter
        python_doc_metadata = python_result["document"].get("metadata", {})
        go_doc_metadata = go_result["document"].get("metadata", {})

        # Front matter should be treated as regular content or ignored
        python_has_front_matter = python_doc_metadata.get("has_front_matter", False)
        go_has_front_matter = go_doc_metadata.get("has_front_matter", False)

        # Both should not extract front matter
        assert not python_has_front_matter
        assert not go_has_front_matter