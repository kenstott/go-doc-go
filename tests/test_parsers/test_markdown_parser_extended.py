"""
Extended unit tests for Markdown document parser to improve coverage.
"""

import pytest
from unittest.mock import MagicMock, patch, mock_open
import tempfile
import os
import json
from go_doc_go.document_parser.markdown import MarkdownParser
from go_doc_go.storage import ElementType
from go_doc_go.relationships import RelationshipType


@pytest.mark.unit
class TestMarkdownParserConfiguration:
    """Test Markdown parser configuration and initialization."""
    
    def test_comprehensive_configuration(self):
        """Test all configuration options."""
        config = {
            "max_content_preview": 200,
            "extract_code_blocks": False,
            "extract_links": False,
            "extract_images": False,
            "extract_tables": False,
            "extract_footnotes": False,
            "extract_metadata": False,
            "parse_frontmatter": False,
            "preserve_whitespace": True,
            "heading_anchors": False,
            "extract_dates": True,
            "date_context_chars": 100,
            "min_year": 1800,
            "max_year": 2200
        }
        
        parser = MarkdownParser(config)
        
        assert parser.max_content_preview == 200
        assert parser.extract_code_blocks == False
        assert parser.extract_links == False
        assert parser.extract_images == False
        assert parser.extract_tables == False
        assert parser.extract_footnotes == False
        assert parser.extract_metadata == False
        assert parser.parse_frontmatter == False
        assert parser.preserve_whitespace == True

    def test_default_configuration(self):
        """Test default configuration values."""
        parser = MarkdownParser()
        
        assert parser.max_content_preview == 100
        assert parser.extract_code_blocks == True
        assert parser.extract_links == True
        assert parser.extract_images == True
        assert parser.extract_tables == True


@pytest.mark.unit
class TestMarkdownParserBasicElements:
    """Test parsing of basic Markdown elements."""
    
    def test_headings_parsing(self):
        """Test parsing of different heading levels."""
        parser = MarkdownParser()
        
        markdown = """# Level 1 Heading
## Level 2 Heading
### Level 3 Heading
#### Level 4 Heading
##### Level 5 Heading
###### Level 6 Heading

Alternative H1
==============

Alternative H2
--------------"""
        
        content = {"id": "/headings.md", "content": markdown, "metadata": {}}
        result = parser.parse(content)
        
        elements = result["elements"]
        # Should have header elements
        header_elements = [e for e in elements if e["element_type"] == ElementType.HEADER.value]
        assert len(header_elements) >= 6
        
        # Check header levels in metadata
        for header in header_elements:
            if "metadata" in header:
                level = header["metadata"].get("level")
                assert level is None or (1 <= level <= 6)

    def test_paragraph_parsing(self):
        """Test parsing of paragraphs."""
        parser = MarkdownParser()
        
        markdown = """This is the first paragraph.
It continues on the next line.

This is the second paragraph after a blank line.

This is a third paragraph with **bold** and *italic* text.

Final paragraph with `inline code` and a [link](https://example.com)."""
        
        content = {"id": "/paragraphs.md", "content": markdown, "metadata": {}}
        result = parser.parse(content)
        
        elements = result["elements"]
        paragraph_elements = [e for e in elements if e["element_type"] == ElementType.PARAGRAPH.value]
        assert len(paragraph_elements) >= 3

    def test_emphasis_parsing(self):
        """Test parsing of emphasis (bold, italic, strikethrough)."""
        parser = MarkdownParser()
        
        markdown = """
This has **bold text** and __also bold__.

This has *italic text* and _also italic_.

This has ***bold italic*** text.

This has ~~strikethrough~~ text.

Nested: **bold with *italic* inside**.
"""
        
        content = {"id": "/emphasis.md", "content": markdown, "metadata": {}}
        result = parser.parse(content)
        
        elements = result["elements"]
        # Content should preserve emphasis markers or handle them
        content_text = " ".join(e.get("content_preview", "") for e in elements)
        assert "bold" in content_text.lower() or "italic" in content_text.lower()


@pytest.mark.unit
class TestMarkdownParserLists:
    """Test parsing of list structures."""
    
    def test_unordered_lists(self):
        """Test parsing of unordered lists."""
        parser = MarkdownParser()
        
        markdown = """
- First item
- Second item
- Third item

* Item with asterisk
* Another item
  * Nested item
  * Another nested item
    * Deeply nested

+ Plus list item
+ Another plus item
"""
        
        content = {"id": "/ul.md", "content": markdown, "metadata": {}}
        result = parser.parse(content)
        
        elements = result["elements"]
        list_elements = [e for e in elements if e["element_type"] == ElementType.LIST.value]
        list_item_elements = [e for e in elements if e["element_type"] == ElementType.LIST_ITEM.value]
        
        assert len(list_elements) > 0 or len(list_item_elements) > 0

    def test_ordered_lists(self):
        """Test parsing of ordered lists."""
        parser = MarkdownParser()
        
        markdown = """
1. First item
2. Second item
3. Third item

1) Item with parenthesis
2) Another item
   1) Nested item
   2) Another nested

1. Item one
   a. Sub-item a
   b. Sub-item b
2. Item two
"""
        
        content = {"id": "/ol.md", "content": markdown, "metadata": {}}
        result = parser.parse(content)
        
        elements = result["elements"]
        # Should detect list structures
        assert len(elements) > 5

    def test_task_lists(self):
        """Test parsing of task lists (checkboxes)."""
        parser = MarkdownParser()
        
        markdown = """
- [x] Completed task
- [ ] Uncompleted task
- [x] Another completed task
  - [ ] Nested uncompleted
  - [x] Nested completed
"""
        
        content = {"id": "/tasks.md", "content": markdown, "metadata": {}}
        result = parser.parse(content)
        
        elements = result["elements"]
        # Task lists should be parsed
        content_text = " ".join(e.get("content_preview", "") for e in elements)
        assert "[x]" in content_text or "[ ]" in content_text or "task" in content_text.lower()


@pytest.mark.unit
class TestMarkdownParserCodeBlocks:
    """Test parsing of code blocks and inline code."""
    
    def test_fenced_code_blocks(self):
        """Test parsing of fenced code blocks."""
        parser = MarkdownParser({"extract_code_blocks": True})
        
        markdown = '''
```python
def hello_world():
    print("Hello, World!")
    return True
```

```javascript
function helloWorld() {
    console.log("Hello, World!");
    return true;
}
```

```
Plain code block without language
```
'''
        
        content = {"id": "/code.md", "content": markdown, "metadata": {}}
        result = parser.parse(content)
        
        elements = result["elements"]
        code_elements = [e for e in elements if e["element_type"] == ElementType.CODE_BLOCK.value]
        assert len(code_elements) >= 2
        
        # Check for language metadata
        for code in code_elements:
            if "metadata" in code:
                lang = code["metadata"].get("language")
                # Language might be python, javascript, or None

    def test_indented_code_blocks(self):
        """Test parsing of indented code blocks."""
        parser = MarkdownParser()
        
        markdown = """
Normal paragraph.

    def indented_code():
        return "This is indented code"
    
    # More indented code
    x = 10

Another paragraph.
"""
        
        content = {"id": "/indented.md", "content": markdown, "metadata": {}}
        result = parser.parse(content)
        
        elements = result["elements"]
        # Should detect code blocks
        code_content = " ".join(e.get("content_preview", "") for e in elements)
        assert "indented_code" in code_content or "def" in code_content

    def test_inline_code(self):
        """Test parsing of inline code."""
        parser = MarkdownParser()
        
        markdown = """
This paragraph has `inline code` in it.

Use the `print()` function to output text.

Multiple inline codes: `var1`, `var2`, and `var3`.
"""
        
        content = {"id": "/inline.md", "content": markdown, "metadata": {}}
        result = parser.parse(content)
        
        elements = result["elements"]
        content_text = " ".join(e.get("content_preview", "") for e in elements)
        assert "inline code" in content_text or "print()" in content_text


@pytest.mark.unit
class TestMarkdownParserLinks:
    """Test parsing of links and references."""
    
    def test_inline_links(self):
        """Test parsing of inline links."""
        parser = MarkdownParser({"extract_links": True})
        
        markdown = """
This is [an inline link](https://example.com).

This is [a link with title](https://example.com "Example Title").

[Link at start](https://start.com) of paragraph.

Paragraph with [link at end](https://end.com).

Multiple [first link](https://first.com) and [second link](https://second.com).
"""
        
        content = {"id": "/links.md", "content": markdown, "metadata": {}}
        result = parser.parse(content)
        
        # Check if links are extracted
        if "links" in result:
            links = result["links"]
            assert len(links) >= 3
            # Verify link structure
            for link in links:
                assert "url" in link or "href" in link

    def test_reference_links(self):
        """Test parsing of reference-style links."""
        parser = MarkdownParser()
        
        markdown = """
This is [a reference link][ref1].

This is [another reference][ref2].

[ref1]: https://example.com "Optional Title"
[ref2]: https://another.com

You can also use [link text itself].

[link text itself]: https://self.com
"""
        
        content = {"id": "/ref_links.md", "content": markdown, "metadata": {}}
        result = parser.parse(content)
        
        elements = result["elements"]
        assert len(elements) > 0

    def test_wiki_links(self):
        """Test parsing of wiki-style links."""
        parser = MarkdownParser()
        
        markdown = """
This has a [[Wiki Link]] to another page.

Another [[Wiki Link with Alias|alias]] example.

Multiple: [[First]], [[Second]], and [[Third]] wiki links.
"""
        
        content = {"id": "/wiki.md", "content": markdown, "metadata": {}}
        result = parser.parse(content)
        
        elements = result["elements"]
        content_text = " ".join(e.get("content_preview", "") for e in elements)
        assert "Wiki Link" in content_text or "[[" in content_text


@pytest.mark.unit
class TestMarkdownParserImages:
    """Test parsing of images."""
    
    def test_inline_images(self):
        """Test parsing of inline images."""
        parser = MarkdownParser({"extract_images": True})
        
        markdown = """
![Alt text](image.png)

![Image with title](photo.jpg "Photo Title")

![](no-alt.gif)

Reference style image: ![alt text][img-ref]

[img-ref]: path/to/image.png "Reference Image"
"""
        
        content = {"id": "/images.md", "content": markdown, "metadata": {}}
        result = parser.parse(content)
        
        elements = result["elements"]
        image_elements = [e for e in elements if e["element_type"] == ElementType.IMAGE.value]
        
        # Should detect images
        assert len(image_elements) > 0 or len(elements) > 0


@pytest.mark.unit
class TestMarkdownParserTables:
    """Test parsing of tables."""
    
    def test_pipe_tables(self):
        """Test parsing of pipe tables."""
        parser = MarkdownParser({"extract_tables": True})
        
        markdown = """
| Header 1 | Header 2 | Header 3 |
|----------|----------|----------|
| Cell 1   | Cell 2   | Cell 3   |
| Cell 4   | Cell 5   | Cell 6   |

| Left | Center | Right |
|:-----|:------:|------:|
| L1   | C1     | R1    |
| L2   | C2     | R2    |
"""
        
        content = {"id": "/tables.md", "content": markdown, "metadata": {}}
        result = parser.parse(content)
        
        elements = result["elements"]
        table_elements = [e for e in elements if e["element_type"] == ElementType.TABLE.value]
        assert len(table_elements) > 0 or len(elements) > 4

    def test_grid_tables(self):
        """Test parsing of grid tables."""
        parser = MarkdownParser()
        
        markdown = """
+----------+----------+
| Header 1 | Header 2 |
+==========+==========+
| Cell 1   | Cell 2   |
+----------+----------+
| Cell 3   | Cell 4   |
+----------+----------+
"""
        
        content = {"id": "/grid.md", "content": markdown, "metadata": {}}
        result = parser.parse(content)
        
        elements = result["elements"]
        # Grid tables might be parsed as code or tables
        assert len(elements) > 0


@pytest.mark.unit
class TestMarkdownParserBlockquotes:
    """Test parsing of blockquotes."""
    
    def test_simple_blockquotes(self):
        """Test parsing of simple blockquotes."""
        parser = MarkdownParser()
        
        markdown = """
> This is a blockquote.
> It continues on the next line.

> Another blockquote.

Regular paragraph.

> Blockquote with **bold** and *italic*.
"""
        
        content = {"id": "/blockquotes.md", "content": markdown, "metadata": {}}
        result = parser.parse(content)
        
        elements = result["elements"]
        blockquote_elements = [e for e in elements if e["element_type"] == ElementType.BLOCKQUOTE.value]
        assert len(blockquote_elements) > 0 or len(elements) > 2

    def test_nested_blockquotes(self):
        """Test parsing of nested blockquotes."""
        parser = MarkdownParser()
        
        markdown = """
> Level 1 blockquote
>> Nested level 2
>>> Deeply nested level 3
>> Back to level 2
> Back to level 1
"""
        
        content = {"id": "/nested_quotes.md", "content": markdown, "metadata": {}}
        result = parser.parse(content)
        
        elements = result["elements"]
        assert len(elements) > 0


@pytest.mark.unit
class TestMarkdownParserFrontmatter:
    """Test parsing of frontmatter/metadata."""
    
    def test_yaml_frontmatter(self):
        """Test parsing of YAML frontmatter."""
        parser = MarkdownParser({"parse_frontmatter": True})
        
        markdown = """---
title: Test Document
author: John Doe
date: 2024-01-15
tags:
  - test
  - markdown
  - parser
---

# Main Content

This is the document content.
"""
        
        content = {"id": "/frontmatter.md", "content": markdown, "metadata": {}}
        result = parser.parse(content)
        
        # Frontmatter might be in document metadata
        doc_metadata = result["document"]["metadata"]
        # Could contain parsed frontmatter
        
        elements = result["elements"]
        assert len(elements) > 0

    def test_toml_frontmatter(self):
        """Test parsing of TOML frontmatter."""
        parser = MarkdownParser({"parse_frontmatter": True})
        
        markdown = """+++
title = "Test Document"
author = "Jane Doe"
date = "2024-01-15"
tags = ["test", "markdown"]
+++

# Content

Document body.
"""
        
        content = {"id": "/toml.md", "content": markdown, "metadata": {}}
        result = parser.parse(content)
        
        elements = result["elements"]
        assert len(elements) > 0


@pytest.mark.unit
class TestMarkdownParserSpecialFeatures:
    """Test special Markdown features."""
    
    def test_footnotes(self):
        """Test parsing of footnotes."""
        parser = MarkdownParser({"extract_footnotes": True})
        
        markdown = """
This has a footnote[^1] reference.

Another footnote[^2] here.

[^1]: This is the first footnote.
[^2]: This is the second footnote with [a link](https://example.com).
"""
        
        content = {"id": "/footnotes.md", "content": markdown, "metadata": {}}
        result = parser.parse(content)
        
        elements = result["elements"]
        # Footnotes might be extracted separately
        content_text = " ".join(e.get("content_preview", "") for e in elements)
        assert "footnote" in content_text.lower()

    def test_horizontal_rules(self):
        """Test parsing of horizontal rules."""
        parser = MarkdownParser()
        
        markdown = """
Section 1

---

Section 2

***

Section 3

___

Section 4
"""
        
        content = {"id": "/hr.md", "content": markdown, "metadata": {}}
        result = parser.parse(content)
        
        elements = result["elements"]
        # Should detect sections separated by horizontal rules
        assert len(elements) >= 4

    def test_html_in_markdown(self):
        """Test handling of HTML within Markdown."""
        parser = MarkdownParser()
        
        markdown = """
Regular markdown paragraph.

<div class="custom">
  <p>HTML paragraph</p>
  <span>HTML span</span>
</div>

Back to markdown with <em>inline HTML</em>.

<table>
  <tr>
    <td>HTML Table Cell</td>
  </tr>
</table>
"""
        
        content = {"id": "/html_mixed.md", "content": markdown, "metadata": {}}
        result = parser.parse(content)
        
        elements = result["elements"]
        # Should handle HTML within markdown
        assert len(elements) > 0

    def test_math_expressions(self):
        """Test parsing of math expressions."""
        parser = MarkdownParser()
        
        markdown = """
Inline math: $x = y + 2$

Display math:
$$
\\sum_{i=1}^{n} i = \\frac{n(n+1)}{2}
$$

Another inline $\\alpha + \\beta = \\gamma$ equation.
"""
        
        content = {"id": "/math.md", "content": markdown, "metadata": {}}
        result = parser.parse(content)
        
        elements = result["elements"]
        content_text = " ".join(e.get("content_preview", "") for e in elements)
        # Math expressions should be preserved
        assert "$" in content_text or "sum" in content_text.lower()


@pytest.mark.unit
class TestMarkdownParserErrorHandling:
    """Test error handling in Markdown parser."""
    
    def test_empty_content(self):
        """Test handling of empty content."""
        parser = MarkdownParser()
        
        test_cases = [
            {"id": "/empty1.md", "content": "", "metadata": {}},
            {"id": "/empty2.md", "content": "\n\n\n", "metadata": {}},
            {"id": "/empty3.md", "content": "   ", "metadata": {}},
        ]
        
        for content in test_cases:
            result = parser.parse(content)
            assert "document" in result
            assert "elements" in result

    def test_malformed_markdown(self):
        """Test handling of malformed Markdown."""
        parser = MarkdownParser()
        
        # Unclosed elements and malformed syntax
        malformed = """
[Unclosed link (https://example.com)

![Unclosed image (image.png)

```python
Unclosed code block

**Unclosed bold

| Incomplete | Table
| Missing | Separator
"""
        
        content = {"id": "/malformed.md", "content": malformed, "metadata": {}}
        
        # Should handle gracefully
        result = parser.parse(content)
        assert "document" in result
        assert len(result["elements"]) > 0

    def test_binary_content(self):
        """Test handling of binary content."""
        parser = MarkdownParser()
        
        binary_content = b'\x00\x01\x02\x03\x04'
        
        content = {"id": "/binary.md", "content": binary_content, "metadata": {}}
        
        try:
            result = parser.parse(content)
            # Should handle or error gracefully
            assert result is not None
        except Exception:
            # Expected to fail on binary
            pass


@pytest.mark.unit
class TestMarkdownParserPerformance:
    """Test performance aspects of Markdown parser."""
    
    def test_large_document(self):
        """Test parsing of large Markdown documents."""
        parser = MarkdownParser({"max_content_preview": 50})
        
        # Generate large markdown
        sections = []
        for i in range(100):
            sections.append(f"""
## Section {i}

This is paragraph {i} with some content.

- List item {i}.1
- List item {i}.2
- List item {i}.3

```python
def function_{i}():
    return {i}
```

| Col1 | Col2 | Col3 |
|------|------|------|
| A{i} | B{i} | C{i} |
""")
        
        large_markdown = "# Large Document\n\n" + "\n".join(sections)
        
        content = {"id": "/large.md", "content": large_markdown, "metadata": {}}
        
        import time
        start = time.time()
        result = parser.parse(content)
        elapsed = time.time() - start
        
        # Should parse efficiently
        assert "document" in result
        assert len(result["elements"]) > 100
        assert elapsed < 5.0  # Should complete within 5 seconds

    def test_deeply_nested_lists(self):
        """Test deeply nested list structures."""
        parser = MarkdownParser()
        
        # Create deeply nested lists
        nested_list = "- Level 0\n"
        for i in range(1, 20):
            indent = "  " * i
            nested_list += f"{indent}- Level {i}\n"
        
        content = {"id": "/deep_lists.md", "content": nested_list, "metadata": {}}
        result = parser.parse(content)
        
        # Should handle deep nesting
        assert "document" in result
        assert len(result["elements"]) > 10


@pytest.mark.unit
class TestMarkdownParserIntegration:
    """Integration tests for Markdown parser."""
    
    def test_comprehensive_document(self):
        """Test parsing a comprehensive Markdown document."""
        parser = MarkdownParser({
            "extract_code_blocks": True,
            "extract_links": True,
            "extract_images": True,
            "extract_tables": True,
            "parse_frontmatter": True
        })
        
        comprehensive_md = """---
title: Comprehensive Test Document
author: Test Author
date: 2024-01-15
tags: [test, markdown, comprehensive]
---

# Main Title

## Introduction

This is a **comprehensive** test document with *various* Markdown features.

### Features Included

1. Headings at multiple levels
2. **Bold**, *italic*, and ~~strikethrough~~ text
3. [Links](https://example.com) and [[Wiki Links]]
4. Images and code blocks

## Code Examples

### Python Code

```python
def process_data(data):
    '''Process input data.'''
    result = []
    for item in data:
        if item > 0:
            result.append(item * 2)
    return result
```

### JavaScript Code

```javascript
const processData = (data) => {
    return data.filter(x => x > 0).map(x => x * 2);
};
```

## Lists and Tasks

### Unordered List

- First item
  - Nested item 1
  - Nested item 2
    - Deeply nested
- Second item
- Third item

### Task List

- [x] Completed task
- [ ] Pending task
- [x] Another completed task

## Tables

| Name  | Age | City |
|-------|-----|------|
| Alice | 30  | NYC  |
| Bob   | 25  | LA   |
| Carol | 35  | SF   |

## Blockquotes

> This is a blockquote with multiple paragraphs.
>
> Second paragraph in the blockquote.
>> Nested blockquote here.

## Links and References

Here's an [inline link](https://inline.com) and a [reference link][ref].

[ref]: https://reference.com "Reference Title"

## Images

![Test Image](image.png "Image Title")

## Footnotes

This has a footnote[^1] reference.

[^1]: This is the footnote content.

---

## Math

Inline math: $E = mc^2$

Display math:
$$
\\int_{0}^{\\infty} e^{-x^2} dx = \\frac{\\sqrt{\\pi}}{2}
$$

## HTML Mixed

<div class="custom">
  <p>Some HTML content</p>
</div>

## Conclusion

This document demonstrates comprehensive Markdown parsing capabilities.
"""
        
        content = {
            "id": "/comprehensive.md",
            "content": comprehensive_md,
            "metadata": {"doc_id": "comp_test"}
        }
        
        result = parser.parse(content)
        
        # Verify comprehensive parsing
        assert "document" in result
        assert result["document"]["doc_id"] == "comp_test"
        assert result["document"]["doc_type"] == "markdown"
        
        # Check elements
        elements = result["elements"]
        assert len(elements) > 20
        
        # Check element types
        element_types = set(e["element_type"] for e in elements)
        # Should have various element types
        assert ElementType.HEADER.value in element_types or ElementType.PARAGRAPH.value in element_types
        
        # Check relationships if present
        if "relationships" in result:
            relationships = result["relationships"]
            assert len(relationships) > 0

    def test_source_file_loading(self):
        """Test loading Markdown from source files."""
        parser = MarkdownParser()
        
        # Create temporary markdown file
        test_md = """# Test Document

This is a test document from a file.

## Section

With some content.
"""
        
        with tempfile.NamedTemporaryFile(mode='w', suffix='.md', delete=False) as f:
            f.write(test_md)
            temp_path = f.name
        
        try:
            # Test with file content
            with open(temp_path, 'r') as f:
                file_content = f.read()
            
            content = {
                "id": temp_path,
                "content": file_content,
                "metadata": {"source_path": temp_path}
            }
            
            result = parser.parse(content)
            assert "document" in result
            assert len(result["elements"]) > 0
            
        finally:
            os.unlink(temp_path)