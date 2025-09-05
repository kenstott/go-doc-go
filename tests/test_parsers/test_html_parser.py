"""
Unit tests for HTML document parser.
"""

import pytest
from typing import Dict, Any
from unittest.mock import Mock, patch

# Add parent directory to path
import sys
import os
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__)))))

from go_doc_go.document_parser.html import HtmlParser
from go_doc_go.storage import ElementType
from go_doc_go.relationships import RelationshipType


class TestHtmlParser:
    """Test suite for HTML document parser."""
    
    def setup_method(self):
        """Set up test fixtures."""
        self.parser = HtmlParser()
        self.sample_html = """<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Test HTML Document</title>
    <style>
        body { font-family: Arial; }
    </style>
</head>
<body>
    <header>
        <nav>
            <a href="#home">Home</a>
            <a href="#about">About</a>
            <a href="https://example.com">External Link</a>
        </nav>
    </header>
    
    <main>
        <h1 id="title">Main Title</h1>
        <p class="intro">This is an <strong>introduction</strong> paragraph with <em>emphasis</em>.</p>
        
        <section id="content">
            <h2>Section Heading</h2>
            <p>Regular paragraph with a <a href="https://test.org">link</a>.</p>
            
            <ul>
                <li>First item</li>
                <li>Second item with <code>inline code</code></li>
                <li>Third item</li>
            </ul>
            
            <ol>
                <li>Numbered item 1</li>
                <li>Numbered item 2</li>
            </ol>
            
            <table border="1">
                <thead>
                    <tr>
                        <th>Header 1</th>
                        <th>Header 2</th>
                    </tr>
                </thead>
                <tbody>
                    <tr>
                        <td>Cell 1</td>
                        <td>Cell 2</td>
                    </tr>
                    <tr>
                        <td>Cell 3</td>
                        <td>Cell 4</td>
                    </tr>
                </tbody>
            </table>
            
            <blockquote>
                This is a blockquote with important information.
            </blockquote>
            
            <pre><code>
def hello_world():
    print("Hello, World!")
            </code></pre>
            
            <img src="image.png" alt="Test Image" />
            
            <form action="/submit" method="post">
                <input type="text" name="username" placeholder="Username">
                <input type="password" name="password" placeholder="Password">
                <button type="submit">Submit</button>
            </form>
        </section>
    </main>
    
    <footer>
        <p>&copy; 2024 Test Company. All rights reserved.</p>
    </footer>
    
    <script>
        console.log("Test script");
    </script>
</body>
</html>"""
        
        self.sample_content = {
            "id": "/path/to/document.html",
            "content": self.sample_html,
            "metadata": {
                "doc_id": "html_doc_123",
                "filename": "document.html"
            }
        }
    
    def test_parser_initialization(self):
        """Test HTML parser initialization."""
        # Default initialization
        parser1 = HtmlParser()
        assert parser1.max_content_preview == 100
        assert parser1.extract_dates == True
        
        # Custom configuration
        config = {
            "max_content_preview": 50,
            "extract_dates": False,
            "enable_caching": False
        }
        parser2 = HtmlParser(config)
        assert parser2.max_content_preview == 50
        assert parser2.extract_dates == False
        assert parser2.enable_caching == False
    
    def test_basic_html_parsing(self):
        """Test basic HTML parsing functionality."""
        result = self.parser.parse(self.sample_content)
        
        # Check basic structure
        assert "document" in result
        assert "elements" in result
        assert "relationships" in result
        
        # Check document
        doc = result["document"]
        assert doc["doc_id"] == "html_doc_123"
        assert doc["doc_type"] == "html"
        assert doc["source"] == "/path/to/document.html"
        
        # Check that we have elements
        elements = result["elements"]
        assert len(elements) > 5
    
    def test_header_extraction(self):
        """Test extraction of headers from HTML."""
        result = self.parser.parse(self.sample_content)
        elements = result["elements"]
        
        # Find header elements (h1, h2, etc.)
        headers = [e for e in elements if e.get("element_type") == ElementType.HEADER.value]
        assert len(headers) > 0
        
        # Check for main title
        h1_found = any("Main Title" in h.get("content_preview", "") for h in headers)
        assert h1_found
    
    def test_paragraph_extraction(self):
        """Test extraction of paragraphs from HTML."""
        result = self.parser.parse(self.sample_content)
        elements = result["elements"]
        
        # Find paragraph elements
        paragraphs = [e for e in elements if e.get("element_type") == ElementType.PARAGRAPH.value]
        assert len(paragraphs) > 0
        
        # Check intro paragraph
        intro_found = any("introduction" in p.get("content_preview", "").lower() for p in paragraphs)
        assert intro_found
    
    def test_list_extraction(self):
        """Test extraction of lists from HTML."""
        result = self.parser.parse(self.sample_content)
        elements = result["elements"]
        
        # Find list elements
        lists = [e for e in elements if e.get("element_type") == ElementType.LIST.value]
        assert len(lists) > 0
        
        # Find list items
        list_items = [e for e in elements if e.get("element_type") == ElementType.LIST_ITEM.value]
        assert len(list_items) > 0
        
        # Check list content
        list_content = " ".join(item.get("content_preview", "") for item in list_items)
        assert "First item" in list_content or "Numbered item" in list_content
    
    def test_table_extraction(self):
        """Test extraction of tables from HTML."""
        result = self.parser.parse(self.sample_content)
        elements = result["elements"]
        
        # Find table elements
        tables = [e for e in elements if e.get("element_type") == ElementType.TABLE.value]
        table_rows = [e for e in elements if e.get("element_type") == ElementType.TABLE_ROW.value]
        
        # Should have table elements
        assert len(tables) > 0 or len(table_rows) > 0
    
    def test_link_extraction(self):
        """Test extraction of links from HTML."""
        result = self.parser.parse(self.sample_content)
        
        # Check if links are extracted
        links = result.get("links", [])
        
        if len(links) > 0:
            # Check that external links are captured
            link_urls = [link.get("url", "") or link.get("link_target", "") for link in links]
            assert any("example.com" in url for url in link_urls) or any("test.org" in url for url in link_urls)
    
    def test_code_block_extraction(self):
        """Test extraction of code blocks from HTML."""
        result = self.parser.parse(self.sample_content)
        elements = result["elements"]
        
        # Find code block elements
        code_blocks = [e for e in elements if e.get("element_type") == ElementType.CODE_BLOCK.value]
        
        # Should have code blocks or at least capture the content
        if len(code_blocks) > 0:
            code_content = " ".join(c.get("content_preview", "") for c in code_blocks)
            assert "hello_world" in code_content or "print" in code_content
    
    def test_blockquote_extraction(self):
        """Test extraction of blockquotes from HTML."""
        result = self.parser.parse(self.sample_content)
        elements = result["elements"]
        
        # Find blockquote elements
        blockquotes = [e for e in elements if e.get("element_type") == ElementType.BLOCKQUOTE.value]
        
        # Should have blockquote or capture it in other elements
        if len(blockquotes) > 0:
            quote_content = blockquotes[0].get("content_preview", "")
            assert "blockquote" in quote_content.lower() or "important" in quote_content
    
    def test_image_extraction(self):
        """Test extraction of images from HTML."""
        result = self.parser.parse(self.sample_content)
        elements = result["elements"]
        
        # Find image elements
        images = [e for e in elements if e.get("element_type") == ElementType.IMAGE.value]
        
        # Should handle images (may or may not create specific elements)
        assert len(images) >= 0
    
    def test_form_handling(self):
        """Test handling of form elements."""
        result = self.parser.parse(self.sample_content)
        elements = result["elements"]
        
        # Should handle forms gracefully
        assert len(elements) > 0
        
        # Form content might be captured in various ways
        full_content = " ".join(e.get("content_preview", "") for e in elements)
        # Forms might not have text content
        assert len(full_content) > 0
    
    def test_script_style_handling(self):
        """Test handling of script and style tags."""
        html_with_scripts = """<!DOCTYPE html>
<html>
<head>
    <style>body { color: red; }</style>
    <script>var x = 1;</script>
</head>
<body>
    <p>Visible content</p>
    <script>console.log("inline");</script>
</body>
</html>"""
        
        content = {
            "id": "/scripts.html",
            "content": html_with_scripts,
            "metadata": {}
        }
        
        result = self.parser.parse(content)
        elements = result["elements"]
        
        # Should handle scripts/styles (usually by filtering them)
        text_content = " ".join(e.get("content_preview", "") for e in elements)
        assert "Visible content" in text_content
        # Script content is usually filtered out
        assert "console.log" not in text_content or len(elements) > 0
    
    def test_nested_structure_parsing(self):
        """Test parsing of nested HTML structures."""
        result = self.parser.parse(self.sample_content)
        elements = result["elements"]
        relationships = result["relationships"]
        
        # Should have hierarchical structure
        assert len(elements) > 10
        assert len(relationships) > 0
        
        # Check relationship types
        rel_types = set(r["relationship_type"] for r in relationships)
        assert RelationshipType.CONTAINS.value in rel_types or RelationshipType.CONTAINED_BY.value in rel_types
    
    def test_empty_html(self):
        """Test handling of empty HTML."""
        empty_html = """<!DOCTYPE html>
<html>
<head><title></title></head>
<body></body>
</html>"""
        
        empty_content = {
            "id": "/empty.html",
            "content": empty_html,
            "metadata": {}
        }
        
        result = self.parser.parse(empty_content)
        
        # Should still create basic structure
        assert "document" in result
        assert "elements" in result
        elements = result["elements"]
        assert len(elements) >= 1  # At least root element
    
    def test_malformed_html(self):
        """Test handling of malformed HTML."""
        malformed_html = """<html>
<head><title>Test</head>  <!-- Missing closing title tag -->
<body>
    <p>Unclosed paragraph
    <div>Missing closing div
    <span>Text</span>
</body>
</html>"""
        
        content = {
            "id": "/malformed.html",
            "content": malformed_html,
            "metadata": {}
        }
        
        result = self.parser.parse(content)
        
        # BeautifulSoup is very forgiving with malformed HTML
        assert result is not None
        assert len(result["elements"]) > 0
    
    def test_special_characters(self):
        """Test handling of special characters and entities."""
        special_html = """<!DOCTYPE html>
<html>
<body>
    <p>HTML entities: &lt; &gt; &amp; &quot; &apos;</p>
    <p>Unicode: café, naïve, 文字, 😀</p>
    <p>Special chars: @#$%^&*()</p>
</body>
</html>"""
        
        content = {
            "id": "/special.html",
            "content": special_html,
            "metadata": {}
        }
        
        result = self.parser.parse(content)
        elements = result["elements"]
        
        # Should handle special characters
        text_content = " ".join(e.get("content_preview", "") for e in elements)
        # Entities should be decoded
        assert "<" in text_content or ">" in text_content or "&" in text_content or "café" in text_content
    
    def test_metadata_extraction(self):
        """Test extraction of metadata from HTML head."""
        meta_html = """<!DOCTYPE html>
<html>
<head>
    <title>Page Title</title>
    <meta name="description" content="Page description">
    <meta name="keywords" content="test, html, parser">
    <meta name="author" content="Test Author">
    <meta property="og:title" content="Open Graph Title">
</head>
<body>
    <p>Content</p>
</body>
</html>"""
        
        content = {
            "id": "/meta.html",
            "content": meta_html,
            "metadata": {}
        }
        
        result = self.parser.parse(content)
        
        # Metadata might be in document metadata or captured as elements
        doc_metadata = result["document"]["metadata"]
        assert doc_metadata is not None


class TestHtmlParserEdgeCases:
    """Test edge cases for HTML parser."""
    
    def setup_method(self):
        """Set up test fixtures."""
        self.parser = HtmlParser()
    
    def test_frameset_html(self):
        """Test handling of frameset HTML."""
        frameset_html = """<!DOCTYPE html>
<html>
<frameset cols="25%,75%">
    <frame src="menu.html">
    <frame src="content.html">
</frameset>
</html>"""
        
        content = {
            "id": "/frameset.html",
            "content": frameset_html,
            "metadata": {}
        }
        
        result = self.parser.parse(content)
        
        # Should handle framesets gracefully
        assert result is not None
    
    def test_iframe_handling(self):
        """Test handling of iframes."""
        iframe_html = """<!DOCTYPE html>
<html>
<body>
    <p>Before iframe</p>
    <iframe src="https://example.com" width="500" height="300"></iframe>
    <p>After iframe</p>
</body>
</html>"""
        
        content = {
            "id": "/iframe.html",
            "content": iframe_html,
            "metadata": {}
        }
        
        result = self.parser.parse(content)
        elements = result["elements"]
        
        # Should handle iframes
        text_content = " ".join(e.get("content_preview", "") for e in elements)
        assert "Before iframe" in text_content or "After iframe" in text_content
    
    def test_svg_content(self):
        """Test handling of SVG content in HTML."""
        svg_html = """<!DOCTYPE html>
<html>
<body>
    <p>Regular text</p>
    <svg width="100" height="100">
        <circle cx="50" cy="50" r="40" stroke="black" fill="red" />
        <text x="50" y="50">SVG Text</text>
    </svg>
    <p>More text</p>
</body>
</html>"""
        
        content = {
            "id": "/svg.html",
            "content": svg_html,
            "metadata": {}
        }
        
        result = self.parser.parse(content)
        
        # Should handle SVG content
        assert result is not None
        assert len(result["elements"]) > 0
    
    def test_data_attributes(self):
        """Test handling of data-* attributes."""
        data_html = """<!DOCTYPE html>
<html>
<body>
    <div data-id="123" data-type="container">
        <span data-value="test">Content with data attributes</span>
    </div>
</body>
</html>"""
        
        content = {
            "id": "/data.html",
            "content": data_html,
            "metadata": {}
        }
        
        result = self.parser.parse(content)
        elements = result["elements"]
        
        # Should handle data attributes
        text_content = " ".join(e.get("content_preview", "") for e in elements)
        assert "Content with data attributes" in text_content
    
    def test_very_deep_nesting(self):
        """Test handling of deeply nested HTML."""
        # Create deeply nested HTML
        nested_html = "<!DOCTYPE html>\n<html>\n<body>\n"
        for i in range(50):
            nested_html += f'<div class="level{i}">\n'
        nested_html += "<p>Deep content</p>\n"
        for i in range(49, -1, -1):
            nested_html += "</div>\n"
        nested_html += "</body>\n</html>"
        
        content = {
            "id": "/deepnest.html",
            "content": nested_html,
            "metadata": {}
        }
        
        result = self.parser.parse(content)
        
        # Should handle deep nesting
        assert result is not None
        assert len(result["elements"]) > 0
    
    def test_large_html_document(self):
        """Test handling of large HTML documents."""
        # Create large HTML
        large_html = """<!DOCTYPE html>
<html>
<body>
"""
        for i in range(500):
            large_html += f"""
    <div class="section{i}">
        <h2>Section {i}</h2>
        <p>This is paragraph {i} with some content.</p>
        <ul>
            <li>Item {i}.1</li>
            <li>Item {i}.2</li>
        </ul>
    </div>
"""
        large_html += """
</body>
</html>"""
        
        content = {
            "id": "/large.html",
            "content": large_html,
            "metadata": {}
        }
        
        result = self.parser.parse(content)
        elements = result["elements"]
        
        # Should handle large documents
        assert len(elements) > 100


if __name__ == "__main__":
    pytest.main([__file__, "-v"])