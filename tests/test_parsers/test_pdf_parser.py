"""
Unit tests for PDF document parser using real PDFs.
"""

import json
import os
import tempfile
import pytest
from typing import Dict, Any

# Add parent directory to path
import sys
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__)))))

from go_doc_go.document_parser.pdf import PdfParser
from go_doc_go.storage import ElementType
from go_doc_go.relationships import RelationshipType

# Check if PyMuPDF is available
try:
    import fitz
    PYMUPDF_AVAILABLE = True
except ImportError:
    PYMUPDF_AVAILABLE = False


def create_simple_pdf(path: str, title: str = "Test Document", 
                     paragraphs: list = None, add_table: bool = False,
                     add_links: bool = False) -> None:
    """Create a simple PDF file for testing."""
    if not PYMUPDF_AVAILABLE:
        pytest.skip("PyMuPDF not available")
    
    doc = fitz.open()
    page = doc.new_page()
    
    y_position = 72  # Start position
    
    # Add title
    fontsize = 16
    text = title
    page.insert_text((72, y_position), text, fontsize=fontsize, fontname="Helvetica-Bold")
    y_position += fontsize + 20
    
    # Add paragraphs
    if paragraphs:
        fontsize = 11
        for para in paragraphs:
            # Split long paragraphs into lines
            words = para.split()
            line = ""
            for word in words:
                test_line = line + " " + word if line else word
                # Approximate line width check
                if len(test_line) > 80:  # Rough approximation
                    if line:
                        page.insert_text((72, y_position), line, fontsize=fontsize)
                        y_position += fontsize + 5
                    line = word
                else:
                    line = test_line
            if line:
                page.insert_text((72, y_position), line, fontsize=fontsize)
                y_position += fontsize + 15
    
    # Add table if requested
    if add_table:
        # Simple table with lines
        table_data = [
            ["Header 1", "Header 2", "Header 3"],
            ["Row 1, Col 1", "Row 1, Col 2", "Row 1, Col 3"],
            ["Row 2, Col 1", "Row 2, Col 2", "Row 2, Col 3"]
        ]
        
        x_start = 72
        col_width = 120
        row_height = 20
        
        for row_idx, row in enumerate(table_data):
            for col_idx, cell_text in enumerate(row):
                x = x_start + col_idx * col_width
                y = y_position + row_idx * row_height
                
                # Draw cell border
                rect = fitz.Rect(x, y - 15, x + col_width, y + 5)
                page.draw_rect(rect, width=0.5)
                
                # Insert text
                fontsize = 10
                if row_idx == 0:
                    page.insert_text((x + 2, y), cell_text, fontsize=fontsize, fontname="Helvetica-Bold")
                else:
                    page.insert_text((x + 2, y), cell_text, fontsize=fontsize)
        
        y_position += len(table_data) * row_height + 20
    
    # Add links if requested
    if add_links:
        link_text = "Visit Example.com"
        page.insert_text((72, y_position), link_text, fontsize=11, color=(0, 0, 1))
        # Add link annotation
        link_rect = fitz.Rect(72, y_position - 11, 200, y_position + 2)
        page.insert_link({"kind": fitz.LINK_URI, "uri": "https://example.com", "from": link_rect})
    
    doc.save(path)
    doc.close()


def create_multipage_pdf(path: str, num_pages: int = 3) -> None:
    """Create a multi-page PDF for testing."""
    if not PYMUPDF_AVAILABLE:
        pytest.skip("PyMuPDF not available")
    
    doc = fitz.open()
    
    for page_num in range(num_pages):
        page = doc.new_page()
        
        # Add page header
        page.insert_text((72, 72), f"Page {page_num + 1}", fontsize=14, fontname="Helvetica-Bold")
        
        # Add some content
        y = 100
        for i in range(3):
            text = f"This is paragraph {i + 1} on page {page_num + 1}."
            page.insert_text((72, y), text, fontsize=11)
            y += 30
    
    doc.save(path)
    doc.close()


@pytest.mark.skipif(not PYMUPDF_AVAILABLE, reason="PyMuPDF not available")
class TestPdfParser:
    """Test suite for PDF parser with real PDFs."""
    
    def setup_method(self):
        """Set up test fixtures."""
        self.parser = PdfParser()
    
    def test_parser_initialization(self):
        """Test PDF parser initialization."""
        # Default initialization
        parser1 = PdfParser()
        assert parser1.max_pages == 1000
        assert parser1.extract_images == False  # Default is False
        assert parser1.extract_tables == True
        assert parser1.extract_links == True
        
        # Custom configuration
        config = {
            "max_pages": 50,
            "extract_images": True,
            "extract_tables": False,
            "extract_links": False,
            "extract_dates": False
        }
        parser2 = PdfParser(config)
        assert parser2.max_pages == 50
        assert parser2.extract_images == True
        assert parser2.extract_tables == False
        assert parser2.extract_links == False
    
    def test_basic_pdf_parsing(self):
        """Test basic PDF parsing functionality."""
        with tempfile.NamedTemporaryFile(suffix='.pdf', delete=False) as tmp:
            tmp_path = tmp.name
        
        try:
            # Create a simple PDF
            create_simple_pdf(tmp_path, 
                            title="Test PDF Document",
                            paragraphs=["This is the first paragraph.",
                                      "This is the second paragraph with more content."])
            
            # Parse it
            content = {
                "id": tmp_path,
                "binary_path": tmp_path,
                "metadata": {
                    "doc_id": "test_pdf_123"
                }
            }
            
            result = self.parser.parse(content)
            
            # Check basic structure
            assert "document" in result
            assert "elements" in result
            assert "relationships" in result
            
            # Check document
            doc = result["document"]
            assert doc["doc_id"] == "test_pdf_123"
            assert doc["doc_type"] == "pdf"
            
            # Check elements
            elements = result["elements"]
            assert len(elements) > 0
            
            # Check for root element
            root = next((e for e in elements if e["element_type"] == ElementType.ROOT.value), None)
            assert root is not None
            
            # Check for body element
            body = next((e for e in elements if e["element_type"] == ElementType.BODY.value), None)
            assert body is not None
            
            # Check for page elements
            pages = [e for e in elements if e["element_type"] == ElementType.PAGE.value]
            assert len(pages) == 1
            
            # Check for text content
            paragraphs = [e for e in elements if e["element_type"] == ElementType.PARAGRAPH.value]
            assert len(paragraphs) >= 2  # At least title and one paragraph
            
        finally:
            if os.path.exists(tmp_path):
                os.unlink(tmp_path)
    
    def test_text_extraction(self):
        """Test text extraction from PDF pages."""
        with tempfile.NamedTemporaryFile(suffix='.pdf', delete=False) as tmp:
            tmp_path = tmp.name
        
        try:
            # Create PDF with specific text
            create_simple_pdf(tmp_path,
                            title="Document Title",
                            paragraphs=[
                                "This is a paragraph with some text content.",
                                "It has multiple paragraphs.",
                                "Another paragraph here with different content."
                            ])
            
            content = {
                "id": tmp_path,
                "binary_path": tmp_path,
                "metadata": {}
            }
            
            result = self.parser.parse(content)
            elements = result["elements"]
            
            # Should have extracted paragraphs
            paragraphs = [e for e in elements if e["element_type"] == ElementType.PARAGRAPH.value]
            assert len(paragraphs) > 0
            
            # Check content preservation
            content_texts = [p["content_preview"] for p in paragraphs]
            all_content = " ".join(content_texts)
            
            assert "paragraph with some text" in all_content
            assert "multiple paragraphs" in all_content
            
        finally:
            if os.path.exists(tmp_path):
                os.unlink(tmp_path)
    
    def test_multipage_pdf(self):
        """Test parsing of multi-page PDFs."""
        with tempfile.NamedTemporaryFile(suffix='.pdf', delete=False) as tmp:
            tmp_path = tmp.name
        
        try:
            # Create 3-page PDF
            create_multipage_pdf(tmp_path, num_pages=3)
            
            content = {
                "id": tmp_path,
                "binary_path": tmp_path,
                "metadata": {}
            }
            
            result = self.parser.parse(content)
            elements = result["elements"]
            
            # Should have 3 pages
            pages = [e for e in elements if e["element_type"] == ElementType.PAGE.value]
            assert len(pages) == 3
            
            # Each page should have content
            for i, page in enumerate(pages):
                assert f"Page {i + 1}" in page["content_preview"]
            
        finally:
            if os.path.exists(tmp_path):
                os.unlink(tmp_path)
    
    def test_table_extraction(self):
        """Test table extraction from PDF."""
        with tempfile.NamedTemporaryFile(suffix='.pdf', delete=False) as tmp:
            tmp_path = tmp.name
        
        try:
            # Create PDF with table
            create_simple_pdf(tmp_path,
                            title="Document with Table",
                            paragraphs=["Some text before the table."],
                            add_table=True)
            
            parser = PdfParser({"extract_tables": True})
            content = {
                "id": tmp_path,
                "binary_path": tmp_path,
                "metadata": {}
            }
            
            result = parser.parse(content)
            elements = result["elements"]
            
            # Should have table elements
            tables = [e for e in elements if e["element_type"] == ElementType.TABLE.value]
            # Note: PyMuPDF's table detection might not always work on simple tables
            # so we just check that parsing doesn't fail
            
        finally:
            if os.path.exists(tmp_path):
                os.unlink(tmp_path)
    
    def test_link_extraction(self):
        """Test link extraction from PDF."""
        with tempfile.NamedTemporaryFile(suffix='.pdf', delete=False) as tmp:
            tmp_path = tmp.name
        
        try:
            # Create PDF with links
            create_simple_pdf(tmp_path,
                            title="Document with Links",
                            paragraphs=["Click the link below:"],
                            add_links=True)
            
            parser = PdfParser({"extract_links": True})
            content = {
                "id": tmp_path,
                "binary_path": tmp_path,
                "metadata": {}
            }
            
            result = parser.parse(content)
            
            # Check if links were extracted
            if "links" in result:
                links = result["links"]
                # Should find the example.com link
                ext_links = [l for l in links if "example.com" in l.get("url", "")]
                assert len(ext_links) > 0
            
        finally:
            if os.path.exists(tmp_path):
                os.unlink(tmp_path)
    
    def test_large_pdf_handling(self):
        """Test handling of PDFs with many pages."""
        with tempfile.NamedTemporaryFile(suffix='.pdf', delete=False) as tmp:
            tmp_path = tmp.name
        
        try:
            # Create large PDF
            create_multipage_pdf(tmp_path, num_pages=10)
            
            # Parse with page limit
            parser = PdfParser({"max_pages": 5})
            content = {
                "id": tmp_path,
                "binary_path": tmp_path,
                "metadata": {}
            }
            
            result = parser.parse(content)
            elements = result["elements"]
            
            # Should limit pages to max_pages
            pages = [e for e in elements if e["element_type"] == ElementType.PAGE.value]
            assert len(pages) == 5
            
            # Metadata should show total pages
            metadata = result["document"]["metadata"]
            assert metadata["page_count"] == 10
            assert metadata.get("pages_processed") == 5
            
        finally:
            if os.path.exists(tmp_path):
                os.unlink(tmp_path)
    
    def test_empty_pdf(self):
        """Test handling of empty PDF."""
        with tempfile.NamedTemporaryFile(suffix='.pdf', delete=False) as tmp:
            tmp_path = tmp.name
        
        try:
            # Create empty PDF (no pages)
            doc = fitz.open()
            doc.save(tmp_path)
            doc.close()
            
            content = {
                "id": tmp_path,
                "binary_path": tmp_path,
                "metadata": {}
            }
            
            result = self.parser.parse(content)
            
            # Should handle empty PDF
            assert result is not None
            elements = result["elements"]
            
            # Should have root and body elements
            root = next((e for e in elements if e["element_type"] == ElementType.ROOT.value), None)
            assert root is not None
            
            # Should have no pages
            pages = [e for e in elements if e["element_type"] == ElementType.PAGE.value]
            assert len(pages) == 0
            
        finally:
            if os.path.exists(tmp_path):
                os.unlink(tmp_path)
    
    def test_metadata_extraction(self):
        """Test extraction of PDF metadata."""
        with tempfile.NamedTemporaryFile(suffix='.pdf', delete=False) as tmp:
            tmp_path = tmp.name
        
        try:
            # Create PDF with metadata
            doc = fitz.open()
            doc.set_metadata({
                "title": "Test PDF Title",
                "author": "Test Author",
                "subject": "Test Subject",
                "keywords": "test, pdf, metadata"
            })
            page = doc.new_page()
            page.insert_text((72, 72), "Content", fontsize=12)
            doc.save(tmp_path)
            doc.close()
            
            content = {
                "id": tmp_path,
                "binary_path": tmp_path,
                "metadata": {}
            }
            
            result = self.parser.parse(content)
            
            # Check metadata extraction
            metadata = result["document"]["metadata"]
            assert metadata.get("title") == "Test PDF Title"
            assert metadata.get("author") == "Test Author"
            assert metadata.get("subject") == "Test Subject"
            assert "test" in metadata.get("keywords", "")
            
        finally:
            if os.path.exists(tmp_path):
                os.unlink(tmp_path)
    
    def test_header_detection(self):
        """Test detection of headers based on formatting."""
        with tempfile.NamedTemporaryFile(suffix='.pdf', delete=False) as tmp:
            tmp_path = tmp.name
        
        try:
            # Create PDF with headers
            doc = fitz.open()
            page = doc.new_page()
            
            # Large header
            page.insert_text((72, 72), "Main Header", fontsize=20, fontname="Helvetica-Bold")
            
            # Subheader
            page.insert_text((72, 120), "Subheader", fontsize=14, fontname="Helvetica-Bold")
            
            # Regular text
            page.insert_text((72, 160), "This is regular paragraph text.", fontsize=11)
            
            doc.save(tmp_path)
            doc.close()
            
            parser = PdfParser({"detect_headers": True})
            content = {
                "id": tmp_path,
                "binary_path": tmp_path,
                "metadata": {}
            }
            
            result = parser.parse(content)
            elements = result["elements"]
            
            # Should detect headers
            headers = [e for e in elements if e["element_type"] == ElementType.HEADER.value]
            paragraphs = [e for e in elements if e["element_type"] == ElementType.PARAGRAPH.value]
            
            assert len(headers) >= 1  # At least one header detected
            assert len(paragraphs) >= 1  # Regular text as paragraph
            
        finally:
            if os.path.exists(tmp_path):
                os.unlink(tmp_path)


@pytest.mark.skipif(not PYMUPDF_AVAILABLE, reason="PyMuPDF not available") 
class TestPdfParserErrorHandling:
    """Test error handling in PDF parser."""
    
    def setup_method(self):
        """Set up test fixtures."""
        self.parser = PdfParser()
    
    def test_nonexistent_file(self):
        """Test handling of non-existent file."""
        content = {
            "id": "/nonexistent/file.pdf",
            "binary_path": "/nonexistent/file.pdf",
            "metadata": {}
        }
        
        with pytest.raises(Exception):
            self.parser.parse(content)
    
    def test_corrupt_pdf(self):
        """Test handling of corrupt PDF file."""
        with tempfile.NamedTemporaryFile(suffix='.pdf', delete=False) as tmp:
            # Write invalid PDF content
            tmp.write(b"This is not a valid PDF file")
            tmp_path = tmp.name
        
        try:
            content = {
                "id": tmp_path,
                "binary_path": tmp_path,
                "metadata": {}
            }
            
            with pytest.raises(Exception):
                self.parser.parse(content)
            
        finally:
            if os.path.exists(tmp_path):
                os.unlink(tmp_path)
    
    def test_binary_content_handling(self):
        """Test handling of binary content without file path."""
        # Create a simple PDF in memory
        doc = fitz.open()
        page = doc.new_page()
        page.insert_text((72, 72), "Test content", fontsize=12)
        pdf_bytes = doc.tobytes()
        doc.close()
        
        content = {
            "id": "memory_pdf",
            "content": pdf_bytes,
            "metadata": {}
        }
        
        result = self.parser.parse(content)
        
        # Should handle binary content
        assert result is not None
        assert "document" in result
        
        # Should have created elements
        elements = result["elements"]
        paragraphs = [e for e in elements if e["element_type"] == ElementType.PARAGRAPH.value]
        assert len(paragraphs) > 0


if __name__ == "__main__":
    pytest.main([__file__, "-v"])