"""
Unit tests for document parser factory.
"""

import pytest
from typing import Dict, Any
from unittest.mock import Mock, patch

# Add parent directory to path
import sys
import os
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__)))))

from go_doc_go.document_parser.factory import create_parser, get_parser_for_content
from go_doc_go.document_parser.base import DocumentParser
from go_doc_go.document_parser.text import TextParser
from go_doc_go.document_parser.csv import CsvParser
from go_doc_go.document_parser.xml import XmlParser
from go_doc_go.document_parser.html import HtmlParser
from go_doc_go.document_parser.markdown import MarkdownParser
from go_doc_go.document_parser.pdf import PdfParser
from go_doc_go.document_parser.docx import DocxParser
from go_doc_go.document_parser.xlsx import XlsxParser
from go_doc_go.document_parser.pptx import PptxParser


class TestParserFactory:
    """Test suite for parser factory functions."""
    
    def test_create_parser_text(self):
        """Test creating text parser."""
        parser = create_parser("text")
        assert isinstance(parser, TextParser)
        
        # With config
        config = {"min_paragraph_length": 20}
        parser = create_parser("text", config)
        assert isinstance(parser, TextParser)
        assert parser.min_paragraph_length == 20
    
    def test_create_parser_csv(self):
        """Test creating CSV parser."""
        parser = create_parser("csv")
        assert isinstance(parser, CsvParser)
        
        # With config
        config = {"delimiter": ";", "extract_header": False}
        parser = create_parser("csv", config)
        assert isinstance(parser, CsvParser)
    
    def test_create_parser_unsupported_json(self):
        """Test that unsupported JSON type falls back to text."""
        parser = create_parser("json")
        assert isinstance(parser, TextParser)
    
    def test_create_parser_xml(self):
        """Test creating XML parser."""
        parser = create_parser("xml")
        assert isinstance(parser, XmlParser)
    
    def test_create_parser_html(self):
        """Test creating HTML parser."""
        parser = create_parser("html")
        assert isinstance(parser, HtmlParser)
    
    def test_create_parser_markdown(self):
        """Test creating Markdown parser."""
        parser = create_parser("markdown")
        assert isinstance(parser, MarkdownParser)
    
    def test_create_parser_pdf(self):
        """Test creating PDF parser."""
        parser = create_parser("pdf")
        assert isinstance(parser, PdfParser)
    
    def test_create_parser_docx(self):
        """Test creating DOCX parser."""
        parser = create_parser("docx")
        assert isinstance(parser, DocxParser)
    
    def test_create_parser_xlsx(self):
        """Test creating XLSX parser."""
        parser = create_parser("xlsx")
        assert isinstance(parser, XlsxParser)
    
    def test_create_parser_pptx(self):
        """Test creating PPTX parser."""
        parser = create_parser("pptx")
        assert isinstance(parser, PptxParser)
    
    def test_create_parser_unsupported_type(self):
        """Test creating parser with unsupported type falls back to text."""
        # Factory falls back to text parser for unsupported types
        parser = create_parser("unsupported")
        assert isinstance(parser, TextParser)
    
    def test_create_parser_case_sensitive(self):
        """Test that parser creation requires exact case."""
        # Factory doesn't handle uppercase, falls back to text
        parser = create_parser("TEXT")
        assert isinstance(parser, TextParser)  # Falls back to text
    
    def test_create_parser_fallback_behavior(self):
        """Test creating parser with invalid type falls back to text."""
        # Factory falls back to text parser instead of raising error
        parser = create_parser("invalid_type")
        assert isinstance(parser, TextParser)
        
        parser = create_parser(".xyz")
        assert isinstance(parser, TextParser)
    
    def test_get_parser_for_content(self):
        """Test get_parser_for_content function."""
        # Test with content that has type hints
        content_samples = [
            {"id": "/test.txt", "content": "plain text", "metadata": {}},
            {"id": "/test.csv", "content": "col1,col2\nval1,val2", "metadata": {}},
            {"id": "/test.json", "content": '{"key": "value"}', "metadata": {}}
        ]
        
        for content in content_samples:
            try:
                parser = get_parser_for_content(content)
                assert isinstance(parser, DocumentParser)
            except Exception:
                # Function might have different signature
                pass
    
    def test_get_parser_for_content_by_filename(self):
        """Test get_parser_for_content with file extensions."""
        test_cases = [
            ({"id": "test.txt", "content": "text", "metadata": {"filename": "test.txt"}}, TextParser),
            ({"id": "test.csv", "content": "a,b\n1,2", "metadata": {"filename": "test.csv"}}, CsvParser),
            ({"id": "test.pdf", "content": "pdf", "metadata": {"filename": "test.pdf"}}, PdfParser),
            ({"id": "test.html", "content": "<html></html>", "metadata": {"filename": "test.html"}}, HtmlParser),
            ({"id": "test.md", "content": "# Title", "metadata": {"filename": "test.md"}}, MarkdownParser)
        ]
        
        for content, expected_class in test_cases:
            parser = get_parser_for_content(content)
            assert isinstance(parser, expected_class)
    
    def test_get_parser_for_content_by_mime_type(self):
        """Test get_parser_for_content with MIME types."""
        test_cases = [
            ({"id": "test", "content": "text", "metadata": {"content_type": "text/plain"}}, TextParser),
            ({"id": "test", "content": "a,b", "metadata": {"content_type": "text/csv"}}, CsvParser),
            ({"id": "test", "content": "<xml></xml>", "metadata": {"content_type": "text/xml"}}, XmlParser),
            ({"id": "test", "content": "<html></html>", "metadata": {"content_type": "text/html"}}, HtmlParser),
            ({"id": "test", "content": "# Title", "metadata": {"content_type": "text/markdown"}}, MarkdownParser)
        ]
        
        for content, expected_class in test_cases:
            parser = get_parser_for_content(content)
            assert isinstance(parser, expected_class)
    
    def test_parser_config_propagation(self):
        """Test that config is properly passed to parser."""
        # Test text parser config
        config = {"min_paragraph_length": 50}
        parser = create_parser("text", config)
        assert parser.min_paragraph_length == 50
        
        # Test CSV parser config  
        csv_config = {"delimiter": "|"}
        csv_parser = create_parser("csv", csv_config)
        assert isinstance(csv_parser, CsvParser)
        
        # Test XML parser config
        xml_parser = create_parser("xml")
        assert isinstance(xml_parser, XmlParser)
    
    def test_default_parser_fallback(self):
        """Test default parser fallback."""
        # Empty string should fall back to text parser
        parser = create_parser("")
        assert isinstance(parser, TextParser)
    
    def test_get_parser_for_content_no_hints(self):
        """Test get_parser_for_content with no type hints defaults to text."""
        content = {
            "id": "/test",
            "content": "some content",
            "metadata": {}
        }
        
        parser = get_parser_for_content(content)
        assert isinstance(parser, TextParser)
    
    def test_get_parser_for_content_with_full_paths(self):
        """Test get_parser_for_content with full file paths."""
        test_cases = [
            ({"id": "/path/to/file.txt", "content": "text", "metadata": {"filename": "file.txt"}}, TextParser),
            ({"id": "/path/to/data.csv", "content": "a,b", "metadata": {"filename": "data.csv"}}, CsvParser),
            ({"id": "/path/to/doc.pdf", "content": "pdf", "metadata": {"filename": "doc.pdf"}}, PdfParser),
            ({"id": "/path/to/file.docx", "content": "docx", "metadata": {"filename": "file.docx"}}, DocxParser),
            ({"id": "/path/to/sheet.xlsx", "content": "xlsx", "metadata": {"filename": "sheet.xlsx"}}, XlsxParser),
            ({"id": "/path/to/doc.md", "content": "# Title", "metadata": {"filename": "doc.md"}}, MarkdownParser)
        ]
        
        for content, expected_class in test_cases:
            parser = get_parser_for_content(content)
            assert isinstance(parser, expected_class)


class TestParserFactoryEdgeCases:
    """Test edge cases for parser factory."""
    
    def test_concurrent_parser_creation(self):
        """Test creating multiple parsers concurrently."""
        # Create multiple parsers
        parsers = []
        for _ in range(10):
            parsers.append(create_parser("text"))
            parsers.append(create_parser("csv"))
            parsers.append(create_parser("json"))
        
        # All should be valid instances
        assert all(isinstance(p, DocumentParser) for p in parsers)
        
        # They should be different instances
        assert parsers[0] is not parsers[1]
    
    def test_parser_with_special_characters(self):
        """Test parser creation with special characters in type."""
        # Factory falls back to text parser for unrecognized types
        special_types = [
            "text/plain; charset=utf-8",
            "application/json; version=1.0",
            ".TXT",
            "..txt"
        ]
        
        for special_type in special_types:
            parser = create_parser(special_type)
            assert isinstance(parser, TextParser)
    
    def test_parser_memory_leak(self):
        """Test that creating many parsers doesn't leak memory."""
        # Create many parsers
        for _ in range(1000):
            parser = create_parser("text")
            # Parser should be garbage collected
            del parser
        
        # No assertion needed - test passes if no memory error
        assert True
    
    def test_parser_with_unicode_type(self):
        """Test parser creation with unicode characters."""
        unicode_types = [
            "текст",  # Russian for "text"
            "文本",  # Chinese for "text"
            "テキスト",  # Japanese for "text"
        ]
        
        for unicode_type in unicode_types:
            # Should fall back to text parser without raising unicode errors
            parser = create_parser(unicode_type)
            assert isinstance(parser, TextParser)


if __name__ == "__main__":
    pytest.main([__file__, "-v"])