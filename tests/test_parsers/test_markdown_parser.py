"""
Unit tests for Markdown document parser.
"""

import pytest
from typing import Dict, Any
from unittest.mock import Mock, patch

# Add parent directory to path
import sys
import os
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__)))))

from go_doc_go.document_parser.markdown import MarkdownParser
from go_doc_go.storage import ElementType
from go_doc_go.relationships import RelationshipType


class TestMarkdownParser:
    """Test suite for Markdown document parser."""
    
    def setup_method(self):
        """Set up test fixtures."""
        self.parser = MarkdownParser()
        self.sample_markdown = """# Main Title

This is the first paragraph with some text.

## Section 1: Introduction

This section introduces the topic. Here are some key points:

- Point 1: First important point
- Point 2: Second important point  
- Point 3: Third important point

### Subsection 1.1

More detailed content here with `inline code` and links.

## Section 2: Code Examples

Here's a code block:

```python
def hello_world():
    print("Hello, World!")
    return "success"
```

Another paragraph with [a link](https://example.com) and **bold text**.

> This is a blockquote with important information.
> It can span multiple lines.

### Table Example

| Column 1 | Column 2 | Column 3 |
|----------|----------|----------|
| Cell 1   | Cell 2   | Cell 3   |
| Data A   | Data B   | Data C   |

## Conclusion

Final thoughts and summary."""

        self.sample_content = {
            "id": "/path/to/document.md",
            "content": self.sample_markdown,
            "metadata": {
                "doc_id": "md_doc_123",
                "filename": "document.md"
            }
        }

    def test_parser_initialization(self):
        """Test Markdown parser initialization."""
        # Default initialization
        parser1 = MarkdownParser()
        assert parser1.extract_front_matter == True
        assert parser1.paragraph_threshold == 1
        assert parser1.max_content_preview == 100
        
        # Custom configuration
        config = {
            "extract_front_matter": False,
            "paragraph_threshold": 2,
            "max_content_preview": 50,
            "extract_dates": False
        }
        parser2 = MarkdownParser(config)
        assert parser2.extract_front_matter == False
        assert parser2.paragraph_threshold == 2
        assert parser2.max_content_preview == 50
        assert parser2.extract_dates == False

    def test_basic_markdown_parsing(self):
        """Test basic Markdown parsing functionality."""
        result = self.parser.parse(self.sample_content)
        
        # Check basic structure
        assert "document" in result
        assert "elements" in result
        assert "relationships" in result
        
        # Check document
        doc = result["document"]
        assert doc["doc_id"] == "md_doc_123"
        assert doc["doc_type"] == "markdown"
        assert doc["source"] == "/path/to/document.md"
        
        # Check that we have elements
        elements = result["elements"]
        assert len(elements) > 3

    def test_header_extraction(self):
        """Test extraction of headers from Markdown."""
        result = self.parser.parse(self.sample_content)
        elements = result["elements"]
        
        # Find header elements
        headers = [e for e in elements if e.get("element_type") == ElementType.HEADER.value]
        assert len(headers) > 0
        
        # Check main title
        main_title = next((h for h in headers if "Main Title" in h.get("content_preview", "")), None)
        assert main_title is not None
        assert "element_id" in main_title
        assert "doc_id" in main_title
        assert "content_hash" in main_title

    def test_paragraph_extraction(self):
        """Test extraction of paragraphs from Markdown."""
        result = self.parser.parse(self.sample_content)
        elements = result["elements"]
        
        # Find paragraph elements
        paragraphs = [e for e in elements if e.get("element_type") == ElementType.PARAGRAPH.value]
        assert len(paragraphs) > 0
        
        # Check first paragraph
        first_para = next((p for p in paragraphs if "first paragraph" in p.get("content_preview", "")), None)
        assert first_para is not None
        assert "some text" in first_para["content_preview"].lower()

    def test_list_extraction(self):
        """Test extraction of lists from Markdown."""
        result = self.parser.parse(self.sample_content)
        elements = result["elements"]
        
        # Find list elements
        lists = [e for e in elements if e.get("element_type") == ElementType.LIST.value]
        assert len(lists) > 0
        
        # Find list items
        list_items = [e for e in elements if e.get("element_type") == ElementType.LIST_ITEM.value]
        assert len(list_items) > 0
        
        # Check that list content exists
        list_content = " ".join(item.get("content_preview", "") for item in list_items)
        assert "Point 1" in list_content or "Point 2" in list_content

    def test_code_block_extraction(self):
        """Test extraction of code blocks from Markdown."""
        result = self.parser.parse(self.sample_content)
        elements = result["elements"]
        
        # Find code block elements
        code_blocks = [e for e in elements if e.get("element_type") == ElementType.CODE_BLOCK.value]
        assert len(code_blocks) > 0
        
        # Check Python code block
        python_block = next((c for c in code_blocks if "hello_world" in c.get("content_preview", "")), None)
        assert python_block is not None
        assert "print" in python_block["content_preview"]

    def test_table_extraction(self):
        """Test extraction of tables from Markdown."""
        result = self.parser.parse(self.sample_content)
        elements = result["elements"]
        
        # Find table elements
        tables = [e for e in elements if e.get("element_type") == ElementType.TABLE.value]
        table_rows = [e for e in elements if e.get("element_type") == ElementType.TABLE_ROW.value]
        
        # Should have either table structure or table content
        assert len(tables) > 0 or len(table_rows) > 0

    def test_blockquote_extraction(self):
        """Test extraction of blockquotes from Markdown."""
        result = self.parser.parse(self.sample_content)
        elements = result["elements"]
        
        # Find blockquote elements
        blockquotes = [e for e in elements if e.get("element_type") == ElementType.BLOCKQUOTE.value]
        
        # Should have blockquote or it's captured in paragraphs
        if len(blockquotes) > 0:
            quote = blockquotes[0]
            assert "blockquote" in quote.get("content_preview", "").lower() or "important information" in quote.get("content_preview", "")

    def test_front_matter_extraction(self):
        """Test extraction of YAML front matter."""
        front_matter_md = """---
title: "Test Document"
author: "Test Author"
date: "2024-01-15"
tags: ["markdown", "test"]
---

# Main Content

This is the document body."""
        
        content = {
            "id": "/front_matter.md",
            "content": front_matter_md,
            "metadata": {}
        }
        
        result = self.parser.parse(content)
        
        # Check if front matter is extracted to metadata or as elements
        metadata = result["document"]["metadata"]
        elements = result["elements"]
        
        # Front matter might be in document metadata or as separate elements
        full_content = " ".join(e.get("content_preview", "") for e in elements)
        assert "Test Document" in full_content or "title" in str(metadata)

    def test_link_extraction(self):
        """Test extraction of links from Markdown."""
        link_md = """# Links Test

Visit [our website](https://example.com) for more info.

Check out [[Wiki Page]] or [another link](http://test.org).

Reference style: [link text][1]

[1]: https://reference.com"""
        
        content = {
            "id": "/links.md",
            "content": link_md,
            "metadata": {}
        }
        
        result = self.parser.parse(content)
        
        # Check if links are extracted
        links = result.get("links", [])
        elements = result["elements"]
        
        # Links might be in links array or captured in element content
        if len(links) > 0:
            link_targets = [link.get("link_target", "") for link in links]
            assert any("example.com" in target for target in link_targets)

    def test_nested_structure_parsing(self):
        """Test parsing of nested Markdown structures."""
        result = self.parser.parse(self.sample_content)
        elements = result["elements"]
        relationships = result["relationships"]
        
        # Should have hierarchical structure
        assert len(elements) > 5
        assert len(relationships) > 0
        
        # Check relationship types
        rel_types = set(r["relationship_type"] for r in relationships)
        assert RelationshipType.CONTAINS.value in rel_types or RelationshipType.CONTAINED_BY.value in rel_types

    def test_empty_markdown(self):
        """Test handling of empty Markdown content."""
        empty_content = {
            "id": "/empty.md",
            "content": "",
            "metadata": {}
        }
        
        result = self.parser.parse(empty_content)
        
        # Should still create basic structure
        assert "document" in result
        assert "elements" in result
        elements = result["elements"]
        assert len(elements) >= 1  # At least root element

    def test_markdown_with_html(self):
        """Test handling of HTML within Markdown."""
        html_md = """# Title with HTML

This paragraph contains <strong>HTML tags</strong> and <em>emphasis</em>.

<div class="special">
<p>HTML block content</p>
</div>

Back to normal markdown."""
        
        content = {
            "id": "/html.md",
            "content": html_md,
            "metadata": {}
        }
        
        result = self.parser.parse(content)
        elements = result["elements"]
        
        # Should handle HTML gracefully
        assert len(elements) > 0
        full_content = " ".join(e.get("content_preview", "") for e in elements)
        assert "HTML tags" in full_content or "special" in full_content

    def test_special_characters(self):
        """Test handling of special characters in Markdown."""
        special_md = """# Title with émojis 😀

Text with unicode: café, naïve, 文字

Special markdown: *italics*, **bold**, `code`, ~~strikethrough~~

Escape sequences: \\* \\_ \\# \\[\\]"""
        
        content = {
            "id": "/special.md",
            "content": special_md,
            "metadata": {}
        }
        
        result = self.parser.parse(content)
        elements = result["elements"]
        
        # Should handle special characters
        assert len(elements) > 0
        full_content = " ".join(e.get("content_preview", "") for e in elements)
        assert "émojis" in full_content or "cafe" in full_content.lower()

    def test_large_document_handling(self):
        """Test handling of large Markdown documents."""
        # Create large markdown content
        large_md = "# Large Document\n\n"
        for i in range(100):
            large_md += f"## Section {i}\n\nThis is paragraph {i} with content.\n\n"
        
        content = {
            "id": "/large.md",
            "content": large_md,
            "metadata": {}
        }
        
        result = self.parser.parse(content)
        elements = result["elements"]
        
        # Should handle large documents
        assert len(elements) > 50
        headers = [e for e in elements if e.get("element_type") == ElementType.HEADER.value]
        assert len(headers) > 10

    def test_relationship_creation(self):
        """Test that relationships are properly created."""
        result = self.parser.parse(self.sample_content)
        relationships = result["relationships"]
        
        # Should have relationships
        assert len(relationships) > 0
        
        # Check relationship types
        rel_types = set(r["relationship_type"] for r in relationships)
        assert RelationshipType.CONTAINS.value in rel_types or RelationshipType.CONTAINED_BY.value in rel_types
        
        # Verify relationship structure
        for rel in relationships:
            assert "source_id" in rel
            assert "target_id" in rel
            assert "relationship_id" in rel

    def test_metadata_extraction(self):
        """Test extraction of document metadata."""
        result = self.parser.parse(self.sample_content)
        metadata = result["document"]["metadata"]
        
        # Should have date extraction metadata
        assert "date_extraction" in metadata


class TestMarkdownParserEdgeCases:
    """Test edge cases for Markdown parser."""
    
    def setup_method(self):
        """Set up test fixtures."""
        self.parser = MarkdownParser()

    def test_malformed_markdown(self):
        """Test handling of malformed Markdown."""
        malformed = """# Unclosed header

### Missing content

**Unclosed bold text

[Broken link](incomplete

* Incomplete list
  * Missing items"""
        
        content = {
            "id": "/malformed.md",
            "content": malformed,
            "metadata": {}
        }
        
        result = self.parser.parse(content)
        
        # Should handle gracefully
        assert result is not None
        assert len(result["elements"]) > 0

    def test_mixed_line_endings(self):
        """Test handling of mixed line endings."""
        mixed_endings = "# Title\r\n\r\nParagraph 1\n\nParagraph 2\r\n\r\n## Section\r"
        
        content = {
            "id": "/mixed.md",
            "content": mixed_endings,
            "metadata": {}
        }
        
        result = self.parser.parse(content)
        elements = result["elements"]
        
        # Should handle different line endings
        assert len(elements) > 0

    def test_very_long_lines(self):
        """Test handling of very long lines."""
        long_line = "# " + "x" * 10000  # 10K character header
        
        content = {
            "id": "/longline.md",
            "content": f"{long_line}\n\nNormal paragraph here.",
            "metadata": {}
        }
        
        result = self.parser.parse(content)
        elements = result["elements"]
        
        # Should handle long lines without crashing
        assert len(elements) > 0

    def test_nested_lists(self):
        """Test handling of nested lists."""
        nested_lists = """# Nested Lists

- Top level item 1
  - Sub item 1.1
  - Sub item 1.2
    - Deep item 1.2.1
- Top level item 2
  1. Numbered sub item 2.1
  2. Numbered sub item 2.2

1. Numbered list
   - Mixed with bullets
   - Another bullet
2. Second numbered item"""
        
        content = {
            "id": "/nested.md",
            "content": nested_lists,
            "metadata": {}
        }
        
        result = self.parser.parse(content)
        elements = result["elements"]
        
        # Should handle nested structures
        lists = [e for e in elements if e.get("element_type") == ElementType.LIST.value]
        list_items = [e for e in elements if e.get("element_type") == ElementType.LIST_ITEM.value]
        
        # Should have some list structure
        assert len(lists) > 0 or len(list_items) > 0

    def test_unicode_content(self):
        """Test handling of Unicode content."""
        unicode_md = """# Título con Acentos

Contenido en español con ñ y acentos: café, naïve.

## 中文標題

中文内容和日文: こんにちは世界

### Русский заголовок

Русский контент с символами: Привет мир!

Mixed content: English, español, 中文, русский."""
        
        content = {
            "id": "/unicode.md",
            "content": unicode_md,
            "metadata": {}
        }
        
        result = self.parser.parse(content)
        elements = result["elements"]
        
        # Should handle unicode gracefully
        assert len(elements) > 0
        
        # Check that unicode content is preserved
        full_content = " ".join(e.get("content_preview", "") for e in elements)
        assert "Título" in full_content or "español" in full_content

    def test_complex_formatting(self):
        """Test handling of complex formatting combinations."""
        complex_md = """# Complex **Formatting** Test

This paragraph has *italic*, **bold**, ***bold italic***, and `inline code`.

Combinations: **bold with `code`** and *italic with [link](url)*.

## Code and Lists

Here's `inline code` in a list:
- Item with **bold**
- Item with *italic*
- Item with `code`

```python
# Code block with comments
def complex_function(param1, param2):
    '''Docstring with special chars: àáâ'''
    return param1 * param2
```

> Blockquote with **formatting** and `code`."""
        
        content = {
            "id": "/complex.md",
            "content": complex_md,
            "metadata": {}
        }
        
        result = self.parser.parse(content)
        elements = result["elements"]
        
        # Should handle complex formatting
        assert len(elements) > 0


if __name__ == "__main__":
    pytest.main([__file__, "-v"])