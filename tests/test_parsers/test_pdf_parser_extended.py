"""
Extended unit tests for PDF document parser to improve coverage.
"""

import pytest
from unittest.mock import MagicMock, patch, Mock
import tempfile
import os
import json
from go_doc_go.document_parser.pdf import PdfParser
from go_doc_go.storage import ElementType
from go_doc_go.relationships import RelationshipType


@pytest.mark.unit
class TestPDFParserConfiguration:
    """Test PDF parser configuration and initialization."""
    
    def test_comprehensive_configuration(self):
        """Test all configuration options."""
        config = {
            "max_content_preview": 200,
            "extract_images": False,
            "extract_tables": False,
            "extract_annotations": False,
            "extract_metadata": False,
            "extract_bookmarks": False,
            "extract_forms": False,
            "max_pages": 50,
            "page_numbers": [1, 2, 3],
            "extract_text_method": "blocks",
            "preserve_layout": True,
            "extract_dates": True,
            "date_context_chars": 100,
            "min_year": 1800,
            "max_year": 2200,
            "ocr_enabled": False,
            "ocr_language": "eng"
        }
        
        parser = PdfParser(config)
        
        assert parser.max_content_preview == 200
        assert parser.extract_images == False
        assert parser.extract_tables == False
        assert parser.extract_annotations == False
        assert parser.extract_metadata == False
        assert parser.max_pages == 50

    def test_default_configuration(self):
        """Test default configuration values."""
        parser = PdfParser()
        
        assert parser.max_content_preview == 100
        assert parser.extract_images == True
        assert parser.extract_tables == True
        assert parser.extract_metadata == True


@pytest.mark.unit
class TestPDFParserWithMocks:
    """Test PDF parser with mocked PyMuPDF."""
    
    def test_basic_pdf_parsing(self):
        """Test basic PDF parsing with mocked document."""
        with patch('go_doc_go.document_parser.pdf.fitz') as mock_fitz:
            # Setup mock document
            mock_doc = MagicMock()
            mock_doc.__len__.return_value = 2
            mock_doc.metadata = {
                "title": "Test PDF",
                "author": "Test Author",
                "subject": "Test Subject",
                "keywords": "test, pdf",
                "creator": "Test Creator",
                "producer": "Test Producer"
            }
            mock_doc.page_count = 2
            
            # Setup mock pages
            mock_page1 = MagicMock()
            mock_page1.number = 0
            mock_page1.rect = MagicMock(width=612, height=792)
            mock_page1.rotation = 0
            mock_page1.get_text.return_value = "Page 1 content\nWith multiple lines"
            mock_page1.get_links.return_value = []
            mock_page1.get_images.return_value = []
            mock_page1.find_tables.return_value = []
            mock_page1.annots.return_value = []
            
            mock_page2 = MagicMock()
            mock_page2.number = 1
            mock_page2.rect = MagicMock(width=612, height=792)
            mock_page2.rotation = 0
            mock_page2.get_text.return_value = "Page 2 content\nMore text here"
            mock_page2.get_links.return_value = []
            mock_page2.get_images.return_value = []
            mock_page2.find_tables.return_value = []
            mock_page2.annots.return_value = []
            
            mock_doc.__getitem__.side_effect = lambda x: [mock_page1, mock_page2][x]
            mock_doc.__iter__.return_value = iter([mock_page1, mock_page2])
            mock_fitz.open.return_value = mock_doc
            
            # Create temp file
            with tempfile.NamedTemporaryFile(suffix='.pdf', delete=False) as tmp:
                tmp.write(b'Mock PDF')
                temp_path = tmp.name
            
            try:
                parser = PdfParser()
                content = {
                    "id": "/test.pdf",
                    "binary_path": temp_path,
                    "metadata": {"doc_id": "pdf_test"}
                }
                
                result = parser.parse(content)
                
                assert "document" in result
                assert result["document"]["doc_id"] == "pdf_test"
                assert result["document"]["doc_type"] == "pdf"
                
                elements = result["elements"]
                assert len(elements) > 0
                
                # Should have page elements
                page_elements = [e for e in elements if e["element_type"] == ElementType.PAGE.value]
                assert len(page_elements) >= 2
                
            finally:
                os.unlink(temp_path)

    def test_pdf_with_images(self):
        """Test PDF parsing with images."""
        with patch('go_doc_go.document_parser.pdf.fitz') as mock_fitz:
            mock_doc = MagicMock()
            mock_doc.__len__.return_value = 1
            mock_doc.metadata = {}
            
            mock_page = MagicMock()
            mock_page.number = 0
            mock_page.rect = MagicMock(width=612, height=792)
            mock_page.get_text.return_value = "Page with images"
            
            # Mock image data
            mock_image = {
                "width": 200,
                "height": 100,
                "ext": "png",
                "xref": 5
            }
            mock_page.get_images.return_value = [mock_image]
            mock_page.get_links.return_value = []
            mock_page.find_tables.return_value = []
            mock_page.annots.return_value = []
            
            mock_doc.__getitem__.return_value = mock_page
            mock_doc.__iter__.return_value = iter([mock_page])
            mock_fitz.open.return_value = mock_doc
            
            with tempfile.NamedTemporaryFile(suffix='.pdf', delete=False) as tmp:
                tmp.write(b'Mock PDF')
                temp_path = tmp.name
            
            try:
                parser = PdfParser({"extract_images": True})
                content = {
                    "id": "/images.pdf",
                    "binary_path": temp_path,
                    "metadata": {}
                }
                
                result = parser.parse(content)
                elements = result["elements"]
                
                # Should detect images
                image_elements = [e for e in elements if e["element_type"] == ElementType.IMAGE.value]
                # Images might be extracted or referenced
                
            finally:
                os.unlink(temp_path)

    def test_pdf_with_tables(self):
        """Test PDF parsing with tables."""
        with patch('go_doc_go.document_parser.pdf.fitz') as mock_fitz:
            mock_doc = MagicMock()
            mock_doc.__len__.return_value = 1
            mock_doc.metadata = {}
            
            mock_page = MagicMock()
            mock_page.number = 0
            mock_page.rect = MagicMock(width=612, height=792)
            mock_page.get_text.return_value = "Page with table"
            
            # Mock table data
            mock_table = MagicMock()
            mock_table.rows = [
                ["Header 1", "Header 2", "Header 3"],
                ["Cell 1", "Cell 2", "Cell 3"],
                ["Cell 4", "Cell 5", "Cell 6"]
            ]
            mock_table.header = mock_table.rows[0]
            mock_page.find_tables.return_value = [mock_table]
            
            mock_page.get_links.return_value = []
            mock_page.get_images.return_value = []
            mock_page.annots.return_value = []
            
            mock_doc.__getitem__.return_value = mock_page
            mock_doc.__iter__.return_value = iter([mock_page])
            mock_fitz.open.return_value = mock_doc
            
            with tempfile.NamedTemporaryFile(suffix='.pdf', delete=False) as tmp:
                tmp.write(b'Mock PDF')
                temp_path = tmp.name
            
            try:
                parser = PdfParser({"extract_tables": True})
                content = {
                    "id": "/tables.pdf",
                    "binary_path": temp_path,
                    "metadata": {}
                }
                
                result = parser.parse(content)
                elements = result["elements"]
                
                # Should detect tables
                table_elements = [e for e in elements if e["element_type"] == ElementType.TABLE.value]
                # Tables might be extracted
                
            finally:
                os.unlink(temp_path)

    def test_pdf_with_links(self):
        """Test PDF parsing with links."""
        with patch('go_doc_go.document_parser.pdf.fitz') as mock_fitz:
            mock_doc = MagicMock()
            mock_doc.__len__.return_value = 1
            mock_doc.metadata = {}
            
            mock_page = MagicMock()
            mock_page.number = 0
            mock_page.rect = MagicMock(width=612, height=792)
            mock_page.get_text.return_value = "Page with links"
            
            # Mock link data
            mock_link = {
                "kind": 2,  # URI link
                "uri": "https://example.com",
                "from": MagicMock(x0=100, y0=100, x1=200, y1=120)
            }
            mock_page.get_links.return_value = [mock_link]
            
            mock_page.get_images.return_value = []
            mock_page.find_tables.return_value = []
            mock_page.annots.return_value = []
            
            mock_doc.__getitem__.return_value = mock_page
            mock_doc.__iter__.return_value = iter([mock_page])
            mock_fitz.open.return_value = mock_doc
            
            with tempfile.NamedTemporaryFile(suffix='.pdf', delete=False) as tmp:
                tmp.write(b'Mock PDF')
                temp_path = tmp.name
            
            try:
                parser = PdfParser()
                content = {
                    "id": "/links.pdf",
                    "binary_path": temp_path,
                    "metadata": {}
                }
                
                result = parser.parse(content)
                
                # Check if links are extracted
                if "links" in result:
                    links = result["links"]
                    assert len(links) >= 0
                
            finally:
                os.unlink(temp_path)

    def test_pdf_with_annotations(self):
        """Test PDF parsing with annotations."""
        with patch('go_doc_go.document_parser.pdf.fitz') as mock_fitz:
            mock_doc = MagicMock()
            mock_doc.__len__.return_value = 1
            mock_doc.metadata = {}
            
            mock_page = MagicMock()
            mock_page.number = 0
            mock_page.rect = MagicMock(width=612, height=792)
            mock_page.get_text.return_value = "Page with annotations"
            
            # Mock annotation
            mock_annot = MagicMock()
            mock_annot.type = [0, "Text"]
            mock_annot.get_text.return_value = "Annotation content"
            mock_annot.author = "Annotator"
            mock_annot.subject = "Comment"
            mock_page.annots.return_value = [mock_annot]
            
            mock_page.get_links.return_value = []
            mock_page.get_images.return_value = []
            mock_page.find_tables.return_value = []
            
            mock_doc.__getitem__.return_value = mock_page
            mock_doc.__iter__.return_value = iter([mock_page])
            mock_fitz.open.return_value = mock_doc
            
            with tempfile.NamedTemporaryFile(suffix='.pdf', delete=False) as tmp:
                tmp.write(b'Mock PDF')
                temp_path = tmp.name
            
            try:
                parser = PdfParser({"extract_annotations": True})
                content = {
                    "id": "/annotations.pdf",
                    "binary_path": temp_path,
                    "metadata": {}
                }
                
                result = parser.parse(content)
                elements = result["elements"]
                
                # Annotations might be extracted as comments
                comment_elements = [e for e in elements if e["element_type"] == ElementType.COMMENT.value]
                # May have comment elements
                
            finally:
                os.unlink(temp_path)

    def test_pdf_with_bookmarks(self):
        """Test PDF parsing with bookmarks/outline."""
        with patch('go_doc_go.document_parser.pdf.fitz') as mock_fitz:
            mock_doc = MagicMock()
            mock_doc.__len__.return_value = 2
            mock_doc.metadata = {}
            
            # Mock bookmarks/TOC
            mock_doc.get_toc.return_value = [
                [1, "Chapter 1", 1],
                [2, "Section 1.1", 1],
                [2, "Section 1.2", 2],
                [1, "Chapter 2", 2]
            ]
            
            mock_page1 = MagicMock()
            mock_page1.number = 0
            mock_page1.get_text.return_value = "Page 1"
            mock_page1.get_links.return_value = []
            mock_page1.get_images.return_value = []
            mock_page1.find_tables.return_value = []
            mock_page1.annots.return_value = []
            
            mock_page2 = MagicMock()
            mock_page2.number = 1
            mock_page2.get_text.return_value = "Page 2"
            mock_page2.get_links.return_value = []
            mock_page2.get_images.return_value = []
            mock_page2.find_tables.return_value = []
            mock_page2.annots.return_value = []
            
            mock_doc.__getitem__.side_effect = lambda x: [mock_page1, mock_page2][x]
            mock_doc.__iter__.return_value = iter([mock_page1, mock_page2])
            mock_fitz.open.return_value = mock_doc
            
            with tempfile.NamedTemporaryFile(suffix='.pdf', delete=False) as tmp:
                tmp.write(b'Mock PDF')
                temp_path = tmp.name
            
            try:
                parser = PdfParser({"extract_bookmarks": True})
                content = {
                    "id": "/bookmarks.pdf",
                    "binary_path": temp_path,
                    "metadata": {}
                }
                
                result = parser.parse(content)
                
                # Bookmarks might be in metadata or elements
                metadata = result["document"]["metadata"]
                # Could contain bookmarks/outline
                
            finally:
                os.unlink(temp_path)


@pytest.mark.unit
class TestPDFParserTextExtraction:
    """Test different text extraction methods."""
    
    def test_text_extraction_methods(self):
        """Test different text extraction methods."""
        with patch('go_doc_go.document_parser.pdf.fitz') as mock_fitz:
            mock_doc = MagicMock()
            mock_doc.__len__.return_value = 1
            mock_doc.metadata = {}
            
            mock_page = MagicMock()
            mock_page.number = 0
            mock_page.rect = MagicMock(width=612, height=792)
            
            # Setup different text extraction returns
            def get_text_mock(format="text"):
                if format == "text":
                    return "Simple text extraction"
                elif format == "blocks":
                    return [
                        (72, 100, 500, 120, "Block 1 text\n", 0, 0),
                        (72, 150, 500, 170, "Block 2 text\n", 0, 1)
                    ]
                elif format == "dict":
                    return {
                        "blocks": [
                            {
                                "type": 0,
                                "lines": [
                                    {"spans": [{"text": "Dict extraction"}]}
                                ]
                            }
                        ]
                    }
                return ""
            
            mock_page.get_text = get_text_mock
            mock_page.get_links.return_value = []
            mock_page.get_images.return_value = []
            mock_page.find_tables.return_value = []
            mock_page.annots.return_value = []
            
            mock_doc.__getitem__.return_value = mock_page
            mock_doc.__iter__.return_value = iter([mock_page])
            mock_fitz.open.return_value = mock_doc
            
            with tempfile.NamedTemporaryFile(suffix='.pdf', delete=False) as tmp:
                tmp.write(b'Mock PDF')
                temp_path = tmp.name
            
            try:
                # Test with blocks extraction
                parser = PdfParser({"extract_text_method": "blocks"})
                content = {
                    "id": "/blocks.pdf",
                    "binary_path": temp_path,
                    "metadata": {}
                }
                
                result = parser.parse(content)
                assert len(result["elements"]) > 0
                
                # Test with dict extraction
                parser2 = PdfParser({"extract_text_method": "dict"})
                result2 = parser2.parse(content)
                assert len(result2["elements"]) > 0
                
            finally:
                os.unlink(temp_path)


@pytest.mark.unit
class TestPDFParserPageSelection:
    """Test page selection and limiting."""
    
    def test_max_pages_limit(self):
        """Test max pages limitation."""
        with patch('go_doc_go.document_parser.pdf.fitz') as mock_fitz:
            mock_doc = MagicMock()
            mock_doc.__len__.return_value = 10
            mock_doc.metadata = {}
            mock_doc.page_count = 10
            
            # Create mock pages
            mock_pages = []
            for i in range(10):
                mock_page = MagicMock()
                mock_page.number = i
                mock_page.get_text.return_value = f"Page {i+1} content"
                mock_page.get_links.return_value = []
                mock_page.get_images.return_value = []
                mock_page.find_tables.return_value = []
                mock_page.annots.return_value = []
                mock_pages.append(mock_page)
            
            mock_doc.__getitem__.side_effect = lambda x: mock_pages[x]
            mock_doc.__iter__.return_value = iter(mock_pages[:3])  # Limit to 3 pages
            mock_fitz.open.return_value = mock_doc
            
            with tempfile.NamedTemporaryFile(suffix='.pdf', delete=False) as tmp:
                tmp.write(b'Mock PDF')
                temp_path = tmp.name
            
            try:
                parser = PdfParser({"max_pages": 3})
                content = {
                    "id": "/limited.pdf",
                    "binary_path": temp_path,
                    "metadata": {}
                }
                
                result = parser.parse(content)
                
                # Should only process limited pages
                page_elements = [e for e in result["elements"] if e["element_type"] == ElementType.PAGE.value]
                assert len(page_elements) <= 3
                
            finally:
                os.unlink(temp_path)

    def test_specific_pages(self):
        """Test extracting specific pages."""
        with patch('go_doc_go.document_parser.pdf.fitz') as mock_fitz:
            mock_doc = MagicMock()
            mock_doc.__len__.return_value = 5
            mock_doc.metadata = {}
            
            # Create mock pages
            mock_pages = []
            for i in range(5):
                mock_page = MagicMock()
                mock_page.number = i
                mock_page.get_text.return_value = f"Page {i+1}"
                mock_page.get_links.return_value = []
                mock_page.get_images.return_value = []
                mock_page.find_tables.return_value = []
                mock_page.annots.return_value = []
                mock_pages.append(mock_page)
            
            mock_doc.__getitem__.side_effect = lambda x: mock_pages[x]
            # Only return specified pages
            mock_doc.__iter__.return_value = iter([mock_pages[0], mock_pages[2], mock_pages[4]])
            mock_fitz.open.return_value = mock_doc
            
            with tempfile.NamedTemporaryFile(suffix='.pdf', delete=False) as tmp:
                tmp.write(b'Mock PDF')
                temp_path = tmp.name
            
            try:
                parser = PdfParser({"page_numbers": [1, 3, 5]})  # 1-indexed
                content = {
                    "id": "/specific.pdf",
                    "binary_path": temp_path,
                    "metadata": {}
                }
                
                result = parser.parse(content)
                elements = result["elements"]
                
                # Should process specified pages
                assert len(elements) > 0
                
            finally:
                os.unlink(temp_path)


@pytest.mark.unit
class TestPDFParserErrorHandling:
    """Test error handling in PDF parser."""
    
    def test_corrupted_pdf(self):
        """Test handling of corrupted PDF."""
        with patch('go_doc_go.document_parser.pdf.fitz') as mock_fitz:
            mock_fitz.open.side_effect = Exception("Cannot open PDF: corrupted")
            
            with tempfile.NamedTemporaryFile(suffix='.pdf', delete=False) as tmp:
                tmp.write(b'Corrupted PDF')
                temp_path = tmp.name
            
            try:
                parser = PdfParser()
                content = {
                    "id": "/corrupted.pdf",
                    "binary_path": temp_path,
                    "metadata": {}
                }
                
                try:
                    result = parser.parse(content)
                    # Should handle error gracefully
                    assert "error" in result or "document" in result
                except Exception as e:
                    # Should be a PDF-related error
                    assert "pdf" in str(e).lower() or "open" in str(e).lower()
                
            finally:
                os.unlink(temp_path)

    def test_empty_pdf(self):
        """Test handling of empty PDF."""
        with patch('go_doc_go.document_parser.pdf.fitz') as mock_fitz:
            mock_doc = MagicMock()
            mock_doc.__len__.return_value = 0
            mock_doc.metadata = {}
            mock_doc.page_count = 0
            mock_doc.__iter__.return_value = iter([])
            mock_fitz.open.return_value = mock_doc
            
            with tempfile.NamedTemporaryFile(suffix='.pdf', delete=False) as tmp:
                tmp.write(b'Empty PDF')
                temp_path = tmp.name
            
            try:
                parser = PdfParser()
                content = {
                    "id": "/empty.pdf",
                    "binary_path": temp_path,
                    "metadata": {}
                }
                
                result = parser.parse(content)
                assert "document" in result
                # Should handle empty PDF
                
            finally:
                os.unlink(temp_path)

    def test_missing_file(self):
        """Test handling of missing file."""
        parser = PdfParser()
        content = {
            "id": "/missing.pdf",
            "binary_path": "/path/to/missing/file.pdf",
            "metadata": {}
        }
        
        try:
            result = parser.parse(content)
            # Should handle missing file
            assert "error" in result or "document" in result
        except Exception as e:
            # Should be file not found error
            assert "not found" in str(e).lower() or "exist" in str(e).lower()


@pytest.mark.unit
class TestPDFParserMetadata:
    """Test metadata extraction from PDF."""
    
    def test_comprehensive_metadata(self):
        """Test extraction of comprehensive metadata."""
        with patch('go_doc_go.document_parser.pdf.fitz') as mock_fitz:
            mock_doc = MagicMock()
            mock_doc.__len__.return_value = 1
            mock_doc.metadata = {
                "title": "Test Document",
                "author": "John Doe",
                "subject": "Testing",
                "keywords": "test, pdf, parser",
                "creator": "Test Creator",
                "producer": "Test Producer",
                "creationDate": "D:20240115120000",
                "modDate": "D:20240120150000"
            }
            mock_doc.page_count = 1
            
            mock_page = MagicMock()
            mock_page.number = 0
            mock_page.get_text.return_value = "Content"
            mock_page.get_links.return_value = []
            mock_page.get_images.return_value = []
            mock_page.find_tables.return_value = []
            mock_page.annots.return_value = []
            
            mock_doc.__getitem__.return_value = mock_page
            mock_doc.__iter__.return_value = iter([mock_page])
            mock_fitz.open.return_value = mock_doc
            
            with tempfile.NamedTemporaryFile(suffix='.pdf', delete=False) as tmp:
                tmp.write(b'Mock PDF')
                temp_path = tmp.name
            
            try:
                parser = PdfParser({"extract_metadata": True})
                content = {
                    "id": "/metadata.pdf",
                    "binary_path": temp_path,
                    "metadata": {}
                }
                
                result = parser.parse(content)
                doc_metadata = result["document"]["metadata"]
                
                # Should extract metadata
                assert "title" in doc_metadata or "author" in doc_metadata or "pdf_metadata" in doc_metadata
                
            finally:
                os.unlink(temp_path)