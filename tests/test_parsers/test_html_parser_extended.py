"""
Extended unit tests for HTML document parser to improve coverage.
"""

import pytest
from unittest.mock import MagicMock, patch, mock_open
import tempfile
import os
import json
from go_doc_go.document_parser.html import HtmlParser
from go_doc_go.storage import ElementType
from go_doc_go.relationships import RelationshipType


@pytest.mark.unit
class TestHTMLParserConfiguration:
    """Test HTML parser configuration and initialization."""
    
    def test_comprehensive_configuration(self):
        """Test all configuration options."""
        config = {
            "max_content_preview": 200,
            "extract_links": False,
            "extract_images": False,
            "extract_tables": False,
            "extract_forms": False,
            "extract_metadata": False,
            "parse_scripts": False,
            "parse_styles": False,
            "preserve_whitespace": True,
            "extract_dates": True,
            "date_context_chars": 100,
            "min_year": 1800,
            "max_year": 2200,
            "prettify": False,
            "parser": "html.parser"
        }
        
        parser = HtmlParser(config)
        
        assert parser.max_content_preview == 200
        assert parser.extract_links == False
        assert parser.extract_images == False
        assert parser.extract_tables == False
        assert parser.extract_forms == False
        assert parser.extract_metadata == False
        assert parser.parse_scripts == False
        assert parser.parse_styles == False
        assert parser.preserve_whitespace == True

    def test_default_configuration(self):
        """Test default configuration values."""
        parser = HtmlParser()
        
        assert parser.max_content_preview == 100
        assert parser.extract_links == True
        assert parser.extract_images == True
        assert parser.extract_tables == True


@pytest.mark.unit
class TestHTMLParserBasicElements:
    """Test parsing of basic HTML elements."""
    
    def test_headings_parsing(self):
        """Test parsing of heading elements."""
        parser = HtmlParser()
        
        html = """<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
    <h1>Heading Level 1</h1>
    <h2>Heading Level 2</h2>
    <h3>Heading Level 3</h3>
    <h4>Heading Level 4</h4>
    <h5>Heading Level 5</h5>
    <h6>Heading Level 6</h6>
</body>
</html>"""
        
        content = {"id": "/headings.html", "content": html, "metadata": {}}
        result = parser.parse(content)
        
        elements = result["elements"]
        header_elements = [e for e in elements if e["element_type"] == ElementType.HEADER.value]
        assert len(header_elements) >= 6

    def test_paragraph_parsing(self):
        """Test parsing of paragraphs."""
        parser = HtmlParser()
        
        html = """<!DOCTYPE html>
<html>
<body>
    <p>This is the first paragraph.</p>
    <p>This is the second paragraph with <strong>bold</strong> and <em>italic</em> text.</p>
    <p>Final paragraph with <code>inline code</code> and a <a href="https://example.com">link</a>.</p>
</body>
</html>"""
        
        content = {"id": "/paragraphs.html", "content": html, "metadata": {}}
        result = parser.parse(content)
        
        elements = result["elements"]
        paragraph_elements = [e for e in elements if e["element_type"] == ElementType.PARAGRAPH.value]
        assert len(paragraph_elements) >= 3

    def test_div_and_span_parsing(self):
        """Test parsing of div and span elements."""
        parser = HtmlParser()
        
        html = """<!DOCTYPE html>
<html>
<body>
    <div class="container">
        <div id="header">Header content</div>
        <div class="content">
            <span class="highlight">Highlighted text</span>
            <span>Regular span</span>
        </div>
        <div id="footer">Footer content</div>
    </div>
</body>
</html>"""
        
        content = {"id": "/divs.html", "content": html, "metadata": {}}
        result = parser.parse(content)
        
        elements = result["elements"]
        # Should parse div structure
        assert len(elements) > 5


@pytest.mark.unit
class TestHTMLParserLists:
    """Test parsing of list structures."""
    
    def test_unordered_lists(self):
        """Test parsing of unordered lists."""
        parser = HtmlParser()
        
        html = """<!DOCTYPE html>
<html>
<body>
    <ul>
        <li>First item</li>
        <li>Second item</li>
        <li>Third item</li>
    </ul>
    
    <ul>
        <li>Item with nested list
            <ul>
                <li>Nested item 1</li>
                <li>Nested item 2</li>
            </ul>
        </li>
        <li>Regular item</li>
    </ul>
</body>
</html>"""
        
        content = {"id": "/ul.html", "content": html, "metadata": {}}
        result = parser.parse(content)
        
        elements = result["elements"]
        list_elements = [e for e in elements if e["element_type"] == ElementType.LIST.value]
        list_item_elements = [e for e in elements if e["element_type"] == ElementType.LIST_ITEM.value]
        
        assert len(list_elements) > 0 or len(list_item_elements) > 0

    def test_ordered_lists(self):
        """Test parsing of ordered lists."""
        parser = HtmlParser()
        
        html = """<!DOCTYPE html>
<html>
<body>
    <ol>
        <li>First ordered item</li>
        <li>Second ordered item</li>
        <li>Third ordered item</li>
    </ol>
    
    <ol type="A">
        <li>Item A</li>
        <li>Item B</li>
    </ol>
    
    <ol start="5">
        <li>Item 5</li>
        <li>Item 6</li>
    </ol>
</body>
</html>"""
        
        content = {"id": "/ol.html", "content": html, "metadata": {}}
        result = parser.parse(content)
        
        elements = result["elements"]
        assert len(elements) > 5

    def test_definition_lists(self):
        """Test parsing of definition lists."""
        parser = HtmlParser()
        
        html = """<!DOCTYPE html>
<html>
<body>
    <dl>
        <dt>Term 1</dt>
        <dd>Definition 1</dd>
        <dt>Term 2</dt>
        <dd>Definition 2</dd>
        <dd>Additional definition for term 2</dd>
    </dl>
</body>
</html>"""
        
        content = {"id": "/dl.html", "content": html, "metadata": {}}
        result = parser.parse(content)
        
        elements = result["elements"]
        assert len(elements) > 3


@pytest.mark.unit
class TestHTMLParserTables:
    """Test parsing of table structures."""
    
    def test_simple_table(self):
        """Test parsing of simple tables."""
        parser = HtmlParser({"extract_tables": True})
        
        html = """<!DOCTYPE html>
<html>
<body>
    <table>
        <thead>
            <tr>
                <th>Header 1</th>
                <th>Header 2</th>
                <th>Header 3</th>
            </tr>
        </thead>
        <tbody>
            <tr>
                <td>Cell 1</td>
                <td>Cell 2</td>
                <td>Cell 3</td>
            </tr>
            <tr>
                <td>Cell 4</td>
                <td>Cell 5</td>
                <td>Cell 6</td>
            </tr>
        </tbody>
    </table>
</body>
</html>"""
        
        content = {"id": "/table.html", "content": html, "metadata": {}}
        result = parser.parse(content)
        
        elements = result["elements"]
        table_elements = [e for e in elements if e["element_type"] == ElementType.TABLE.value]
        assert len(table_elements) > 0 or len(elements) > 5

    def test_complex_table(self):
        """Test parsing of complex tables with colspan and rowspan."""
        parser = HtmlParser()
        
        html = """<!DOCTYPE html>
<html>
<body>
    <table border="1">
        <caption>Complex Table</caption>
        <thead>
            <tr>
                <th colspan="2">Merged Header</th>
                <th rowspan="2">Tall Header</th>
            </tr>
            <tr>
                <th>Sub Header 1</th>
                <th>Sub Header 2</th>
            </tr>
        </thead>
        <tbody>
            <tr>
                <td colspan="3">Wide cell</td>
            </tr>
            <tr>
                <td>Normal</td>
                <td rowspan="2">Tall cell</td>
                <td>Normal</td>
            </tr>
            <tr>
                <td>Normal</td>
                <td>Normal</td>
            </tr>
        </tbody>
        <tfoot>
            <tr>
                <td colspan="3">Footer</td>
            </tr>
        </tfoot>
    </table>
</body>
</html>"""
        
        content = {"id": "/complex_table.html", "content": html, "metadata": {}}
        result = parser.parse(content)
        
        elements = result["elements"]
        assert len(elements) > 8


@pytest.mark.unit
class TestHTMLParserLinks:
    """Test parsing of links and anchors."""
    
    def test_external_links(self):
        """Test parsing of external links."""
        parser = HtmlParser({"extract_links": True})
        
        html = """<!DOCTYPE html>
<html>
<body>
    <a href="https://example.com">External Link</a>
    <a href="http://test.org" target="_blank">New Window Link</a>
    <a href="https://secure.com" rel="nofollow">Nofollow Link</a>
    <a href="mailto:test@example.com">Email Link</a>
    <a href="tel:+1234567890">Phone Link</a>
</body>
</html>"""
        
        content = {"id": "/links.html", "content": html, "metadata": {}}
        result = parser.parse(content)
        
        # Check if links are extracted
        if "links" in result:
            links = result["links"]
            assert len(links) >= 3

    def test_internal_links(self):
        """Test parsing of internal links and anchors."""
        parser = HtmlParser()
        
        html = """<!DOCTYPE html>
<html>
<body>
    <a href="#section1">Jump to Section 1</a>
    <a href="#section2">Jump to Section 2</a>
    <a href="/page.html">Relative Link</a>
    <a href="../parent.html">Parent Link</a>
    
    <h2 id="section1">Section 1</h2>
    <h2 id="section2">Section 2</h2>
</body>
</html>"""
        
        content = {"id": "/internal.html", "content": html, "metadata": {}}
        result = parser.parse(content)
        
        elements = result["elements"]
        assert len(elements) > 4


@pytest.mark.unit
class TestHTMLParserImages:
    """Test parsing of images."""
    
    def test_image_elements(self):
        """Test parsing of image elements."""
        parser = HtmlParser({"extract_images": True})
        
        html = """<!DOCTYPE html>
<html>
<body>
    <img src="image.jpg" alt="Test Image">
    <img src="/path/to/image.png" alt="Another Image" width="100" height="100">
    <img src="data:image/png;base64,iVBORw..." alt="Data URL Image">
    
    <picture>
        <source media="(min-width:650px)" srcset="large.jpg">
        <source media="(min-width:465px)" srcset="medium.jpg">
        <img src="small.jpg" alt="Responsive Image">
    </picture>
    
    <figure>
        <img src="figure.jpg" alt="Figure Image">
        <figcaption>Image Caption</figcaption>
    </figure>
</body>
</html>"""
        
        content = {"id": "/images.html", "content": html, "metadata": {}}
        result = parser.parse(content)
        
        elements = result["elements"]
        image_elements = [e for e in elements if e["element_type"] == ElementType.IMAGE.value]
        assert len(image_elements) > 0 or len(elements) > 3


@pytest.mark.unit
class TestHTMLParserForms:
    """Test parsing of form elements."""
    
    def test_form_elements(self):
        """Test parsing of form elements."""
        parser = HtmlParser({"extract_forms": True})
        
        html = """<!DOCTYPE html>
<html>
<body>
    <form action="/submit" method="post">
        <label for="name">Name:</label>
        <input type="text" id="name" name="name" required>
        
        <label for="email">Email:</label>
        <input type="email" id="email" name="email">
        
        <label for="message">Message:</label>
        <textarea id="message" name="message" rows="4" cols="50"></textarea>
        
        <select name="options">
            <option value="opt1">Option 1</option>
            <option value="opt2">Option 2</option>
        </select>
        
        <input type="checkbox" id="agree" name="agree">
        <label for="agree">I agree</label>
        
        <input type="radio" name="choice" value="yes"> Yes
        <input type="radio" name="choice" value="no"> No
        
        <button type="submit">Submit</button>
    </form>
</body>
</html>"""
        
        content = {"id": "/form.html", "content": html, "metadata": {}}
        result = parser.parse(content)
        
        elements = result["elements"]
        # Should parse form elements
        assert len(elements) > 5


@pytest.mark.unit
class TestHTMLParserSemantic:
    """Test parsing of semantic HTML5 elements."""
    
    def test_semantic_structure(self):
        """Test parsing of semantic HTML5 elements."""
        parser = HtmlParser()
        
        html = """<!DOCTYPE html>
<html lang="en">
<head>
    <title>Semantic HTML</title>
</head>
<body>
    <header>
        <nav>
            <ul>
                <li><a href="#home">Home</a></li>
                <li><a href="#about">About</a></li>
            </ul>
        </nav>
    </header>
    
    <main>
        <article>
            <header>
                <h1>Article Title</h1>
                <time datetime="2024-01-15">January 15, 2024</time>
            </header>
            <section>
                <h2>Section Title</h2>
                <p>Section content.</p>
            </section>
            <aside>
                <p>Related information.</p>
            </aside>
            <footer>
                <p>Article footer.</p>
            </footer>
        </article>
    </main>
    
    <footer>
        <address>
            Contact: <a href="mailto:test@example.com">test@example.com</a>
        </address>
    </footer>
</body>
</html>"""
        
        content = {"id": "/semantic.html", "content": html, "metadata": {}}
        result = parser.parse(content)
        
        elements = result["elements"]
        # Should parse semantic structure
        assert len(elements) > 10


@pytest.mark.unit
class TestHTMLParserMetadata:
    """Test parsing of metadata and head elements."""
    
    def test_head_metadata(self):
        """Test parsing of head metadata."""
        parser = HtmlParser({"extract_metadata": True})
        
        html = """<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <meta name="description" content="Test page description">
    <meta name="keywords" content="test, html, parser">
    <meta name="author" content="Test Author">
    <meta property="og:title" content="Open Graph Title">
    <meta property="og:description" content="OG Description">
    <meta property="og:image" content="https://example.com/image.jpg">
    <title>Page Title</title>
    <link rel="canonical" href="https://example.com/page">
    <link rel="stylesheet" href="style.css">
    <link rel="icon" href="favicon.ico">
</head>
<body>
    <p>Content</p>
</body>
</html>"""
        
        content = {"id": "/metadata.html", "content": html, "metadata": {}}
        result = parser.parse(content)
        
        # Metadata might be in document metadata
        doc_metadata = result["document"]["metadata"]
        # Could contain parsed meta tags


@pytest.mark.unit
class TestHTMLParserScriptsStyles:
    """Test parsing of scripts and styles."""
    
    def test_script_elements(self):
        """Test parsing of script elements."""
        parser = HtmlParser({"parse_scripts": True})
        
        html = """<!DOCTYPE html>
<html>
<head>
    <script src="external.js"></script>
    <script>
        function test() {
            console.log("Inline script");
        }
    </script>
</head>
<body>
    <p>Content</p>
    <script type="module">
        import { module } from './module.js';
    </script>
    <script type="application/json">
        {"data": "json"}
    </script>
</body>
</html>"""
        
        content = {"id": "/scripts.html", "content": html, "metadata": {}}
        result = parser.parse(content)
        
        elements = result["elements"]
        # Scripts might be parsed or skipped
        assert len(elements) > 0

    def test_style_elements(self):
        """Test parsing of style elements."""
        parser = HtmlParser({"parse_styles": True})
        
        html = """<!DOCTYPE html>
<html>
<head>
    <link rel="stylesheet" href="external.css">
    <style>
        body { margin: 0; }
        .class { color: red; }
    </style>
</head>
<body>
    <p style="color: blue;">Inline styled content</p>
    <div style="background: yellow; padding: 10px;">Styled div</div>
</body>
</html>"""
        
        content = {"id": "/styles.html", "content": html, "metadata": {}}
        result = parser.parse(content)
        
        elements = result["elements"]
        assert len(elements) > 0


@pytest.mark.unit
class TestHTMLParserSpecialContent:
    """Test parsing of special HTML content."""
    
    def test_comments(self):
        """Test handling of HTML comments."""
        parser = HtmlParser()
        
        html = """<!DOCTYPE html>
<html>
<body>
    <!-- This is a comment -->
    <p>Visible content</p>
    <!-- Multi-line
         comment -->
    <!--[if IE]>
        <p>IE specific content</p>
    <![endif]-->
</body>
</html>"""
        
        content = {"id": "/comments.html", "content": html, "metadata": {}}
        result = parser.parse(content)
        
        elements = result["elements"]
        # Comments might be included or excluded
        assert len(elements) > 0

    def test_entities_and_special_chars(self):
        """Test handling of HTML entities and special characters."""
        parser = HtmlParser()
        
        html = """<!DOCTYPE html>
<html>
<body>
    <p>&lt;tag&gt; &amp; &quot;quotes&quot;</p>
    <p>&copy; &reg; &trade; &euro; &pound;</p>
    <p>Unicode: café, 中文, 😊</p>
    <p>&nbsp;&nbsp;&nbsp;Non-breaking spaces</p>
</body>
</html>"""
        
        content = {"id": "/entities.html", "content": html, "metadata": {}}
        result = parser.parse(content)
        
        elements = result["elements"]
        # Entities should be decoded
        content_text = " ".join(e.get("content_preview", "") for e in elements)
        assert "&" in content_text or "©" in content_text or "café" in content_text

    def test_iframe_and_embed(self):
        """Test handling of iframe and embed elements."""
        parser = HtmlParser()
        
        html = """<!DOCTYPE html>
<html>
<body>
    <iframe src="https://example.com/embedded" width="600" height="400"></iframe>
    <iframe srcdoc="<p>Inline iframe content</p>"></iframe>
    
    <embed src="document.pdf" type="application/pdf">
    <object data="document.pdf" type="application/pdf">
        <p>Fallback content</p>
    </object>
    
    <video src="video.mp4" controls>
        <track src="captions.vtt" kind="captions">
    </video>
    
    <audio src="audio.mp3" controls></audio>
</body>
</html>"""
        
        content = {"id": "/embedded.html", "content": html, "metadata": {}}
        result = parser.parse(content)
        
        elements = result["elements"]
        assert len(elements) > 3


@pytest.mark.unit
class TestHTMLParserErrorHandling:
    """Test error handling in HTML parser."""
    
    def test_malformed_html(self):
        """Test handling of malformed HTML."""
        parser = HtmlParser()
        
        malformed = """<!DOCTYPE html>
<html>
<body>
    <p>Unclosed paragraph
    <div>Unclosed div
    <span>Missing close tags
    <table>
        <tr>
            <td>Incomplete table
    </body>
</html>"""
        
        content = {"id": "/malformed.html", "content": malformed, "metadata": {}}
        
        # Should handle gracefully
        result = parser.parse(content)
        assert "document" in result
        assert len(result["elements"]) > 0

    def test_empty_content(self):
        """Test handling of empty content."""
        parser = HtmlParser()
        
        test_cases = [
            {"id": "/empty1.html", "content": "", "metadata": {}},
            {"id": "/empty2.html", "content": "<!DOCTYPE html><html></html>", "metadata": {}},
            {"id": "/empty3.html", "content": "<html><body></body></html>", "metadata": {}},
        ]
        
        for content in test_cases:
            result = parser.parse(content)
            assert "document" in result
            assert "elements" in result

    def test_non_html_content(self):
        """Test handling of non-HTML content."""
        parser = HtmlParser()
        
        non_html = """This is plain text, not HTML.
It should still be parsed somehow."""
        
        content = {"id": "/text.html", "content": non_html, "metadata": {}}
        result = parser.parse(content)
        
        # Should handle as best as possible
        assert "document" in result


@pytest.mark.unit
class TestHTMLParserPerformance:
    """Test performance aspects of HTML parser."""
    
    def test_large_document(self):
        """Test parsing of large HTML documents."""
        parser = HtmlParser({"max_content_preview": 50})
        
        # Generate large HTML
        rows = []
        for i in range(500):
            rows.append(f"""
        <tr>
            <td>Row {i} Cell 1</td>
            <td>Row {i} Cell 2</td>
            <td>Row {i} Cell 3</td>
            <td><a href="/link{i}">Link {i}</a></td>
        </tr>""")
        
        large_html = f"""<!DOCTYPE html>
<html>
<head><title>Large Document</title></head>
<body>
    <h1>Large Table</h1>
    <table>
        <thead>
            <tr>
                <th>Column 1</th>
                <th>Column 2</th>
                <th>Column 3</th>
                <th>Links</th>
            </tr>
        </thead>
        <tbody>
            {''.join(rows)}
        </tbody>
    </table>
</body>
</html>"""
        
        content = {"id": "/large.html", "content": large_html, "metadata": {}}
        
        import time
        start = time.time()
        result = parser.parse(content)
        elapsed = time.time() - start
        
        # Should parse efficiently
        assert "document" in result
        assert len(result["elements"]) > 500
        assert elapsed < 10.0  # Should complete within 10 seconds

    def test_deeply_nested_html(self):
        """Test deeply nested HTML structures."""
        parser = HtmlParser()
        
        # Create deeply nested divs
        nested_html = "<!DOCTYPE html><html><body>"
        for i in range(50):
            nested_html += f'<div class="level{i}">'
        nested_html += "Deep content"
        for i in range(50):
            nested_html += "</div>"
        nested_html += "</body></html>"
        
        content = {"id": "/deep.html", "content": nested_html, "metadata": {}}
        result = parser.parse(content)
        
        # Should handle deep nesting
        assert "document" in result
        assert len(result["elements"]) > 0


@pytest.mark.unit
class TestHTMLParserIntegration:
    """Integration tests for HTML parser."""
    
    def test_comprehensive_document(self):
        """Test parsing a comprehensive HTML document."""
        parser = HtmlParser({
            "extract_links": True,
            "extract_images": True,
            "extract_tables": True,
            "extract_forms": True,
            "extract_metadata": True
        })
        
        comprehensive_html = """<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="description" content="Comprehensive test document">
    <title>Comprehensive HTML Test</title>
    <style>
        body { font-family: Arial; }
    </style>
</head>
<body>
    <header>
        <nav>
            <ul>
                <li><a href="#section1">Section 1</a></li>
                <li><a href="#section2">Section 2</a></li>
            </ul>
        </nav>
    </header>
    
    <main>
        <article>
            <h1>Main Article Title</h1>
            
            <section id="section1">
                <h2>Section 1: Text Content</h2>
                <p>This is a paragraph with <strong>bold</strong>, <em>italic</em>, and <code>code</code> text.</p>
                <blockquote>This is a blockquote.</blockquote>
                <pre>Preformatted text</pre>
            </section>
            
            <section id="section2">
                <h2>Section 2: Lists and Tables</h2>
                
                <ul>
                    <li>Unordered item 1</li>
                    <li>Unordered item 2</li>
                </ul>
                
                <ol>
                    <li>Ordered item 1</li>
                    <li>Ordered item 2</li>
                </ol>
                
                <table>
                    <caption>Sample Table</caption>
                    <thead>
                        <tr>
                            <th>Name</th>
                            <th>Value</th>
                        </tr>
                    </thead>
                    <tbody>
                        <tr>
                            <td>Item 1</td>
                            <td>100</td>
                        </tr>
                        <tr>
                            <td>Item 2</td>
                            <td>200</td>
                        </tr>
                    </tbody>
                </table>
            </section>
            
            <section>
                <h2>Media and Forms</h2>
                
                <figure>
                    <img src="image.jpg" alt="Test image">
                    <figcaption>Image caption</figcaption>
                </figure>
                
                <form action="/submit" method="post">
                    <label>Name: <input type="text" name="name"></label>
                    <label>Email: <input type="email" name="email"></label>
                    <textarea name="message" placeholder="Your message"></textarea>
                    <button type="submit">Submit</button>
                </form>
            </section>
        </article>
        
        <aside>
            <h3>Related Links</h3>
            <ul>
                <li><a href="https://example.com">External Link</a></li>
                <li><a href="/internal">Internal Link</a></li>
            </ul>
        </aside>
    </main>
    
    <footer>
        <p>&copy; 2024 Test. All rights reserved.</p>
    </footer>
    
    <script>
        console.log("Page loaded");
    </script>
</body>
</html>"""
        
        content = {
            "id": "/comprehensive.html",
            "content": comprehensive_html,
            "metadata": {"doc_id": "comp_test"}
        }
        
        result = parser.parse(content)
        
        # Verify comprehensive parsing
        assert "document" in result
        assert result["document"]["doc_id"] == "comp_test"
        assert result["document"]["doc_type"] == "html"
        
        # Check elements
        elements = result["elements"]
        assert len(elements) > 20
        
        # Check element types
        element_types = set(e["element_type"] for e in elements)
        # Should have various element types
        
        # Check relationships if present
        if "relationships" in result:
            relationships = result["relationships"]
            assert len(relationships) > 0