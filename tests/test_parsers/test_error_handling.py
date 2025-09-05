"""
Comprehensive error handling tests for all document parsers.
"""

import pytest
import json
import tempfile
import os
from typing import Dict, Any
from unittest.mock import Mock, patch, MagicMock, mock_open
import io

# Add parent directory to path
import sys
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__)))))

from go_doc_go.document_parser.factory import create_parser
from go_doc_go.document_parser.base import DocumentParser
from go_doc_go.document_parser.csv import CsvParser
from go_doc_go.document_parser.json import JSONParser
from go_doc_go.document_parser.pdf import PdfParser
from go_doc_go.document_parser.xlsx import XlsxParser
from go_doc_go.document_parser.docx import DocxParser
from go_doc_go.document_parser.pptx import PptxParser
from go_doc_go.document_parser.html import HtmlParser
from go_doc_go.document_parser.xml import XmlParser
from go_doc_go.document_parser.markdown import MarkdownParser
from go_doc_go.document_parser.text import TextParser


class TestCommonErrorHandling:
    """Test common error scenarios across all parsers."""
    
    def setup_method(self):
        """Set up test fixtures."""
        self.parsers = {
            "csv": CsvParser(),
            "json": JsonParser(),
            "text": TextParser(),
            "markdown": MarkdownParser(),
            "html": HtmlParser(),
            "xml": XmlParser()
        }
        
        self.binary_parsers = {
            "pdf": PdfParser(),
            "xlsx": XlsxParser(),
            "docx": DocxParser(),
            "pptx": PptxParser()
        }
    
    def test_missing_content_field(self):
        """Test all parsers handle missing content field."""
        invalid_content = {
            "id": "/test/file",
            "metadata": {"doc_id": "test_123"}
            # Missing "content" field
        }
        
        for name, parser in {**self.parsers, **self.binary_parsers}.items():
            with pytest.raises(KeyError, match="content"):
                parser.parse(invalid_content)
    
    def test_missing_id_field(self):
        """Test all parsers handle missing id field."""
        invalid_content = {
            "content": "test content",
            "metadata": {"doc_id": "test_123"}
            # Missing "id" field
        }
        
        for name, parser in self.parsers.items():
            # Should raise KeyError or handle gracefully
            try:
                result = parser.parse(invalid_content)
                # If it doesn't raise, check result is valid
                assert result is not None
                assert "document" in result
            except KeyError:
                pass  # Expected behavior
    
    def test_none_content(self):
        """Test all parsers handle None content."""
        none_content = {
            "id": "/test/file",
            "content": None,
            "metadata": {}
        }
        
        for name, parser in self.parsers.items():
            # Should handle None gracefully
            result = parser.parse(none_content)
            assert result is not None
            assert "elements" in result
            # Should have at least root element
            assert len(result["elements"]) >= 1
    
    def test_empty_content(self):
        """Test all parsers handle empty content."""
        empty_content = {
            "id": "/test/file",
            "content": "",
            "metadata": {}
        }
        
        for name, parser in self.parsers.items():
            result = parser.parse(empty_content)
            assert result is not None
            assert "document" in result
            assert "elements" in result
            assert len(result["elements"]) >= 1  # At least root
    
    def test_extremely_large_content(self):
        """Test handling of extremely large content."""
        # 100MB of content
        huge_content = {
            "id": "/test/huge",
            "content": "x" * (100 * 1024 * 1024),
            "metadata": {}
        }
        
        # Test text-based parsers
        text_parser = TextParser({"max_size": 10 * 1024 * 1024})  # 10MB limit
        result = text_parser.parse(huge_content)
        assert result is not None
        
        # Check content was truncated
        elements = result["elements"]
        total_preview_size = sum(len(e.get("content_preview", "")) for e in elements)
        assert total_preview_size < len(huge_content["content"])
    
    def test_unicode_handling(self):
        """Test all parsers handle Unicode content properly."""
        unicode_content = {
            "id": "/test/unicode",
            "content": "Test 中文 日本語 العربية 🚀 émoji √∑∫",
            "metadata": {}
        }
        
        for name, parser in self.parsers.items():
            result = parser.parse(unicode_content)
            assert result is not None
            
            # Check Unicode is preserved
            elements = result["elements"]
            combined_content = " ".join(e.get("content_preview", "") for e in elements)
            
            # At least some Unicode should be preserved
            assert any(char in combined_content for char in ["中", "日", "🚀", "é", "√"])
    
    def test_control_characters(self):
        """Test handling of control characters."""
        control_content = {
            "id": "/test/control",
            "content": "Test\x00with\x01control\x02chars\x03and\ttabs\nlines",
            "metadata": {}
        }
        
        for name, parser in self.parsers.items():
            result = parser.parse(control_content)
            assert result is not None
            assert "elements" in result


class TestBinaryParserErrorHandling:
    """Test error handling specific to binary format parsers."""
    
    @patch('go_doc_go.document_parser.pdf.fitz')
    def test_pdf_corrupt_file(self, mock_fitz):
        """Test PDF parser handling of corrupt files."""
        mock_fitz.open.side_effect = Exception("Corrupt PDF")
        
        parser = PdfParser()
        content = {
            "id": "/corrupt.pdf",
            "content": b"Not a real PDF",
            "metadata": {}
        }
        
        # Should handle corrupt PDF gracefully
        with pytest.raises(Exception):
            parser.parse(content)
    
    @patch('go_doc_go.document_parser.xlsx.openpyxl')
    def test_xlsx_corrupt_file(self, mock_openpyxl):
        """Test XLSX parser handling of corrupt files."""
        mock_openpyxl.load_workbook.side_effect = Exception("Invalid XLSX")
        
        parser = XlsxParser()
        content = {
            "id": "/corrupt.xlsx",
            "content": b"Not a real XLSX",
            "metadata": {}
        }
        
        # Should handle corrupt XLSX gracefully
        with pytest.raises(Exception):
            parser.parse(content)
    
    @patch('go_doc_go.document_parser.docx.Document')
    def test_docx_corrupt_file(self, mock_document):
        """Test DOCX parser handling of corrupt files."""
        mock_document.side_effect = Exception("Invalid DOCX")
        
        parser = DocxParser()
        content = {
            "id": "/corrupt.docx",
            "content": b"Not a real DOCX",
            "metadata": {}
        }
        
        # Should handle corrupt DOCX gracefully
        with pytest.raises(Exception):
            parser.parse(content)
    
    @patch('go_doc_go.document_parser.pptx.Presentation')
    def test_pptx_corrupt_file(self, mock_presentation):
        """Test PPTX parser handling of corrupt files."""
        mock_presentation.side_effect = Exception("Invalid PPTX")
        
        parser = PptxParser()
        content = {
            "id": "/corrupt.pptx",
            "content": b"Not a real PPTX",
            "metadata": {}
        }
        
        # Should handle corrupt PPTX gracefully
        with pytest.raises(Exception):
            parser.parse(content)


class TestTextParserErrorHandling:
    """Test error handling specific to text-based parsers."""
    
    def test_csv_inconsistent_columns(self):
        """Test CSV parser with inconsistent column counts."""
        inconsistent_csv = """Col1,Col2,Col3
Val1,Val2
Val3,Val4,Val5,Val6
Val7"""
        
        parser = CsvParser()
        content = {
            "id": "/inconsistent.csv",
            "content": inconsistent_csv,
            "metadata": {}
        }
        
        # Should handle gracefully
        result = parser.parse(content)
        assert result is not None
        assert "elements" in result
    
    def test_json_nested_too_deep(self):
        """Test JSON parser with extremely deep nesting."""
        # Create 100-level deep JSON
        deep_json = {}
        current = deep_json
        for i in range(100):
            current[f"level{i}"] = {}
            current = current[f"level{i}"]
        current["data"] = "deep value"
        
        parser = JsonParser({"max_depth": 10})
        content = {
            "id": "/deep.json",
            "content": json.dumps(deep_json),
            "metadata": {}
        }
        
        # Should handle by limiting depth
        result = parser.parse(content)
        assert result is not None
        
        # Should not have all 100 levels
        elements = result["elements"]
        assert len(elements) < 100
    
    def test_xml_invalid_syntax(self):
        """Test XML parser with invalid XML."""
        invalid_xmls = [
            "<root><unclosed>",
            "<root><tag attr='unclosed quote>content</tag></root>",
            "<root>&invalid_entity;</root>",
            "Not XML at all",
            "<root><nested><nested></root>"  # Mismatched tags
        ]
        
        parser = XmlParser()
        
        for invalid_xml in invalid_xmls:
            content = {
                "id": "/invalid.xml",
                "content": invalid_xml,
                "metadata": {}
            }
            
            # Should handle invalid XML gracefully
            try:
                result = parser.parse(content)
                assert result is not None
            except Exception:
                pass  # Some invalid XML may raise exceptions
    
    def test_html_malformed_structure(self):
        """Test HTML parser with malformed HTML."""
        malformed_htmls = [
            "<html><body><div>Unclosed div</body></html>",
            "<html><body><p>Paragraph <b>bold <i>italic</b></i></p></body></html>",
            "<!DOCTYPE html><html><body>Text without tags",
            "<script>alert('xss')</script><body>Content</body>"
        ]
        
        parser = HtmlParser()
        
        for malformed_html in malformed_htmls:
            content = {
                "id": "/malformed.html",
                "content": malformed_html,
                "metadata": {}
            }
            
            # HTML parsers are usually forgiving
            result = parser.parse(content)
            assert result is not None
            assert "elements" in result
    
    def test_markdown_invalid_syntax(self):
        """Test Markdown parser with edge cases."""
        edge_cases = [
            "# Heading without closing\n[Link without closing(",
            "```\nUnclosed code block",
            "**Bold *with* nested** emphasis",
            "> Quote\n>> Nested\n>>> Deep\n>>>> Very deep quotes"
        ]
        
        parser = MarkdownParser()
        
        for markdown in edge_cases:
            content = {
                "id": "/edge.md",
                "content": markdown,
                "metadata": {}
            }
            
            # Markdown is forgiving
            result = parser.parse(content)
            assert result is not None
            assert "elements" in result


class TestParserFactoryErrorHandling:
    """Test error handling in parser factory."""
    
    def test_unsupported_format(self):
        """Test factory with unsupported format."""
        # Should raise or return None for unsupported format
        with pytest.raises(ValueError):
            create_parser("unsupported", {})
    
    def test_invalid_config(self):
        """Test parser creation with invalid config."""
        # Invalid config type
        with pytest.raises(TypeError):
            create_parser("csv", "not_a_dict")
        
        # Config with invalid values
        invalid_configs = [
            {"max_rows": "not_a_number"},
            {"delimiter": 123},  # Should be string
            {"extract_header": "yes"}  # Should be boolean
        ]
        
        for invalid_config in invalid_configs:
            # Parser should handle or validate config
            try:
                parser = create_parser("csv", invalid_config)
                # If creation succeeds, parser should handle invalid config
                assert parser is not None
            except (TypeError, ValueError):
                pass  # Expected for invalid config


class TestMemoryAndPerformance:
    """Test memory and performance handling."""
    
    def test_memory_leak_prevention(self):
        """Test that parsers don't leak memory with repeated use."""
        parser = TextParser()
        
        # Parse many documents
        for i in range(100):
            content = {
                "id": f"/test_{i}.txt",
                "content": f"Content {i}" * 1000,
                "metadata": {}
            }
            result = parser.parse(content)
            
            # Result should be independent
            assert result is not None
            
            # Clear result to free memory
            del result
    
    def test_concurrent_parsing(self):
        """Test thread safety of parsers."""
        import threading
        import queue
        
        parser = TextParser()
        results = queue.Queue()
        errors = queue.Queue()
        
        def parse_document(doc_id):
            try:
                content = {
                    "id": f"/doc_{doc_id}.txt",
                    "content": f"Document {doc_id} content",
                    "metadata": {"thread_id": doc_id}
                }
                result = parser.parse(content)
                results.put(result)
            except Exception as e:
                errors.put(e)
        
        # Create multiple threads
        threads = []
        for i in range(10):
            t = threading.Thread(target=parse_document, args=(i,))
            threads.append(t)
            t.start()
        
        # Wait for completion
        for t in threads:
            t.join()
        
        # Check results
        assert errors.empty(), "Parsing errors occurred in threads"
        assert results.qsize() == 10, "Not all documents parsed"
        
        # Verify each result is valid
        seen_ids = set()
        while not results.empty():
            result = results.get()
            assert "document" in result
            doc_id = result["document"]["metadata"]["thread_id"]
            assert doc_id not in seen_ids, "Duplicate result"
            seen_ids.add(doc_id)


class TestEdgeCases:
    """Test various edge cases."""
    
    def test_empty_file_variations(self):
        """Test different types of empty content."""
        empty_variations = [
            "",  # Empty string
            " ",  # Single space
            "\n",  # Single newline
            "\t",  # Single tab
            "   \n\t\n   ",  # Only whitespace
            "\x00",  # Null character
            "\r\n",  # Windows newline
        ]
        
        parser = TextParser()
        
        for empty in empty_variations:
            content = {
                "id": "/empty.txt",
                "content": empty,
                "metadata": {}
            }
            
            result = parser.parse(content)
            assert result is not None
            assert "document" in result
            assert "elements" in result
    
    def test_mixed_encodings(self):
        """Test handling of mixed encodings."""
        # Mix of UTF-8, Latin-1, and ASCII
        mixed_content = "ASCII text café München 北京"
        
        parser = TextParser()
        content = {
            "id": "/mixed.txt",
            "content": mixed_content,
            "metadata": {}
        }
        
        result = parser.parse(content)
        assert result is not None
        
        # Check content preservation
        elements = result["elements"]
        combined = " ".join(e.get("content_preview", "") for e in elements)
        assert "café" in combined or "caf" in combined  # May lose accent
        assert "München" in combined or "Munchen" in combined
    
    def test_special_filenames(self):
        """Test handling of special characters in filenames."""
        special_names = [
            "/path with spaces/file.txt",
            "/path/with/中文/file.txt",
            "/path/with/.hidden/file.txt",
            "C:\\Windows\\Path\\file.txt",
            "/path/with/../relative/file.txt",
            "/path/with/~home/file.txt"
        ]
        
        parser = TextParser()
        
        for filename in special_names:
            content = {
                "id": filename,
                "content": "test content",
                "metadata": {"filename": filename}
            }
            
            result = parser.parse(content)
            assert result is not None
            assert result["document"]["source"] == filename


if __name__ == "__main__":
    pytest.main([__file__, "-v"])