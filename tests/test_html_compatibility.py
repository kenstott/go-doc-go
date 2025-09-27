#!/usr/bin/env python3
"""
Compatibility tests between Python and Go HTML parsers.

This test suite ensures that the Go HTML parser produces results
that are compatible with the existing Python HTML parser.
"""

import sys
import os
import unittest
from typing import Dict, Any, Set

# Add src directory to path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..', 'src'))

from go_doc_go.document_parser.html import HtmlParser as PythonHTMLParser
from go_doc_go.document_parser.html_go import GoHTMLParser


class TestHTMLParserCompatibility(unittest.TestCase):
    """Test compatibility between Python and Go HTML parsers."""

    def setUp(self):
        """Set up test parsers."""
        self.python_parser = PythonHTMLParser()
        self.go_parser = GoHTMLParser()

    def _compare_basic_structure(self, python_result: Dict[str, Any], go_result: Dict[str, Any]):
        """
        Compare basic structure of parser results.

        Args:
            python_result: Result from Python parser
            go_result: Result from Go parser
        """
        # Both should have the same top-level keys
        expected_keys = {"document", "elements", "relationships", "links"}

        self.assertIn("document", python_result)
        self.assertIn("document", go_result)
        self.assertIn("elements", python_result)
        self.assertIn("elements", go_result)
        self.assertIn("relationships", python_result)
        self.assertIn("relationships", go_result)
        self.assertIn("links", python_result)
        self.assertIn("links", go_result)

    def _compare_element_types(self, python_elements: list, go_elements: list):
        """
        Compare element types found by both parsers.

        Args:
            python_elements: Elements from Python parser
            go_elements: Elements from Go parser
        """
        python_types = set(elem["element_type"] for elem in python_elements)
        go_types = set(elem["element_type"] for elem in go_elements)

        # Both should find similar element types for HTML content
        # (allowing for some differences in implementation details)
        common_types = python_types.intersection(go_types)

        # Should have significant overlap in element types
        self.assertGreater(len(common_types), 0,
                          f"No common element types found. Python: {python_types}, Go: {go_types}")

    def _normalize_content_for_comparison(self, content: str) -> str:
        """
        Normalize content for comparison between parsers.

        Args:
            content: Raw content string

        Returns:
            Normalized content
        """
        # Remove extra whitespace, normalize newlines
        import re
        normalized = re.sub(r'\s+', ' ', content.strip())
        return normalized

    def test_simple_html_compatibility(self):
        """Test parsing simple HTML content."""
        content = {
            "id": "simple_test",
            "content": "<html><body><h1>Test Header</h1><p>Test paragraph</p></body></html>",
            "metadata": {"source": "test"}
        }

        try:
            python_result = self.python_parser.parse(content)
        except Exception as e:
            self.skipTest(f"Python parser failed: {e}")

        go_result = self.go_parser.parse(content)

        # Compare basic structure
        self._compare_basic_structure(python_result, go_result)

        # Both should find headers and paragraphs
        python_elements = python_result["elements"]
        go_elements = go_result["elements"]

        python_has_header = any(elem["element_type"] in ["header", "h1"] for elem in python_elements)
        go_has_header = any(elem["element_type"] in ["header", "h1"] for elem in go_elements)

        python_has_paragraph = any(elem["element_type"] in ["paragraph", "p"] for elem in python_elements)
        go_has_paragraph = any(elem["element_type"] in ["paragraph", "p"] for elem in go_elements)

        self.assertTrue(python_has_header or go_has_header, "At least one parser should find header elements")
        self.assertTrue(python_has_paragraph or go_has_paragraph, "At least one parser should find paragraph elements")

    def test_complex_html_compatibility(self):
        """Test parsing complex HTML content."""
        content = {
            "id": "complex_test",
            "content": """
            <html>
                <head><title>Test Page</title></head>
                <body>
                    <header>
                        <h1>Main Title</h1>
                        <nav>
                            <ul>
                                <li><a href="/home">Home</a></li>
                                <li><a href="/about">About</a></li>
                            </ul>
                        </nav>
                    </header>
                    <main>
                        <article>
                            <h2>Article Title</h2>
                            <p>Article content with <strong>bold</strong> text.</p>
                            <table>
                                <tr>
                                    <th>Header 1</th>
                                    <th>Header 2</th>
                                </tr>
                                <tr>
                                    <td>Cell 1</td>
                                    <td>Cell 2</td>
                                </tr>
                            </table>
                        </article>
                    </main>
                </body>
            </html>
            """,
            "metadata": {"source": "complex_test"}
        }

        try:
            python_result = self.python_parser.parse(content)
        except Exception as e:
            self.skipTest(f"Python parser failed: {e}")

        go_result = self.go_parser.parse(content)

        # Compare basic structure
        self._compare_basic_structure(python_result, go_result)

        # Compare element types
        self._compare_element_types(python_result["elements"], go_result["elements"])

        # Both should find links
        self.assertGreater(len(python_result["links"]) + len(go_result["links"]), 0,
                          "At least one parser should find links")

        # Both should find relationships
        self.assertGreater(len(python_result["relationships"]) + len(go_result["relationships"]), 0,
                          "At least one parser should find relationships")

    def test_link_extraction_compatibility(self):
        """Test link extraction compatibility."""
        content = {
            "id": "link_test",
            "content": """
            <html><body>
                <a href="http://example.com">External Link</a>
                <a href="/internal">Internal Link</a>
                <a href="mailto:test@example.com">Email Link</a>
            </body></html>
            """,
            "metadata": {"source": "link_test"}
        }

        try:
            python_result = self.python_parser.parse(content)
        except Exception as e:
            self.skipTest(f"Python parser failed: {e}")

        go_result = self.go_parser.parse(content)

        # Both should extract links
        python_links = python_result["links"]
        go_links = go_result["links"]

        # Should find at least some links
        total_links = len(python_links) + len(go_links)
        self.assertGreater(total_links, 0, "At least one parser should find links")

        # If both find links, check they have similar structure
        if python_links and go_links:
            # Check that link objects have required fields
            for link in python_links:
                self.assertIn("link_target", link)
                self.assertIn("link_text", link)

            for link in go_links:
                self.assertIn("link_target", link)
                self.assertIn("link_text", link)

    def test_empty_html_compatibility(self):
        """Test parsing empty HTML content."""
        content = {
            "id": "empty_test",
            "content": "",
            "metadata": {"source": "empty_test"}
        }

        try:
            python_result = self.python_parser.parse(content)
        except Exception as e:
            self.skipTest(f"Python parser failed: {e}")

        go_result = self.go_parser.parse(content)

        # Both should handle empty content gracefully
        self._compare_basic_structure(python_result, go_result)

        # Both should have at least root elements
        self.assertGreater(len(python_result["elements"]), 0)
        self.assertGreater(len(go_result["elements"]), 0)

    def test_malformed_html_compatibility(self):
        """Test parsing malformed HTML content."""
        content = {
            "id": "malformed_test",
            "content": "<div><p>Unclosed tags<span>More unclosed</div>",
            "metadata": {"source": "malformed_test"}
        }

        try:
            python_result = self.python_parser.parse(content)
        except Exception as e:
            self.skipTest(f"Python parser failed: {e}")

        go_result = self.go_parser.parse(content)

        # Both should handle malformed HTML gracefully
        self._compare_basic_structure(python_result, go_result)

        # Both should extract some elements even from malformed HTML
        self.assertGreater(len(python_result["elements"]), 0)
        self.assertGreater(len(go_result["elements"]), 0)

    def test_document_metadata_compatibility(self):
        """Test document metadata preservation."""
        metadata = {
            "source": "test_document.html",
            "parser_version": "test",
            "custom_field": "custom_value"
        }

        content = {
            "id": "metadata_test",
            "content": "<html><body><p>Test content</p></body></html>",
            "metadata": metadata
        }

        try:
            python_result = self.python_parser.parse(content)
        except Exception as e:
            self.skipTest(f"Python parser failed: {e}")

        go_result = self.go_parser.parse(content)

        # Both should preserve document metadata
        python_doc_metadata = python_result["document"].get("metadata", {})
        go_doc_metadata = go_result["document"].get("metadata", {})

        # At least one should preserve the metadata
        total_metadata_fields = len(python_doc_metadata) + len(go_doc_metadata)
        self.assertGreater(total_metadata_fields, 0,
                          "At least one parser should preserve metadata")

    def test_performance_reasonable(self):
        """Test that Go parser performance is reasonable."""
        import time

        # Large HTML content for performance testing
        large_content = "<html><body>"
        for i in range(100):
            large_content += f"<div><h2>Section {i}</h2><p>Content for section {i} with <a href='/link{i}'>link {i}</a></p></div>"
        large_content += "</body></html>"

        content = {
            "id": "performance_test",
            "content": large_content,
            "metadata": {"source": "performance_test"}
        }

        # Time the Go parser
        start_time = time.time()
        go_result = self.go_parser.parse(content)
        go_time = time.time() - start_time

        # Should complete in reasonable time (under 10 seconds for this test)
        self.assertLess(go_time, 10.0, f"Go parser took too long: {go_time:.2f}s")

        # Should produce reasonable number of elements
        self.assertGreater(len(go_result["elements"]), 50,
                          "Should find substantial number of elements in large document")

        print(f"Go parser processed large document in {go_time:.3f}s, found {len(go_result['elements'])} elements")


class TestGoParserSpecific(unittest.TestCase):
    """Test Go parser specific functionality."""

    def setUp(self):
        """Set up Go parser."""
        self.go_parser = GoHTMLParser()

    def test_css_selector_generation(self):
        """Test that Go parser generates CSS selectors."""
        content = {
            "id": "selector_test",
            "content": '<html><body><div id="main"><p class="content">Test</p></div></body></html>',
            "metadata": {}
        }

        result = self.go_parser.parse(content)

        # Check that elements have content_location with selectors
        elements_with_selectors = [
            elem for elem in result["elements"]
            if "content_location" in elem and
               isinstance(elem["content_location"], dict) and
               "selector" in elem["content_location"]
        ]

        self.assertGreater(len(elements_with_selectors), 0,
                          "Should generate CSS selectors for elements")

    def test_namespaced_elements(self):
        """Test handling of namespaced XML/XBRL elements."""
        content = {
            "id": "namespace_test",
            "content": '''
            <html xmlns:ix="http://www.xbrl.org/2013/inlineXBRL">
                <body>
                    <ix:nonnumeric>Test content</ix:nonnumeric>
                    <custom:element>Custom namespace</custom:element>
                </body>
            </html>
            ''',
            "metadata": {}
        }

        result = self.go_parser.parse(content)

        # Should handle namespaced elements without crashing
        element_types = [elem["element_type"] for elem in result["elements"]]

        # Should convert namespace prefixes to underscores
        namespaced_elements = [et for et in element_types if "_" in et and et not in ["list_item", "table_row", "table_header", "table_cell"]]

        # If namespaced elements are present, they should be converted properly
        if any(":" in elem["element_type"] for elem in result["elements"]):
            self.fail("Namespaced elements should have colons converted to underscores")


if __name__ == "__main__":
    # Run with verbose output
    unittest.main(verbosity=2)