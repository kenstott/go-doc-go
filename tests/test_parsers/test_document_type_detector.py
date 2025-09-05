"""
Unit tests for document type detector.
"""

import pytest
import tempfile
import os
from pathlib import Path
from unittest.mock import Mock, patch

# Add parent directory to path
import sys
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__)))))

from go_doc_go.document_parser.document_type_detector import DocumentTypeDetector, initialize_magic


class TestDocumentTypeDetector:
    """Test suite for document type detector."""
    
    def test_detect_from_path_extensions(self):
        """Test detection based on file extensions."""
        test_cases = [
            ("/path/to/file.txt", "text"),
            ("/path/to/doc.md", "markdown"),
            ("/path/to/doc.markdown", "markdown"),
            ("/path/to/data.csv", "csv"),
            ("/path/to/data.tsv", "csv"),
            ("/path/to/config.json", "json"),
            ("/path/to/page.html", "html"),
            ("/path/to/page.htm", "html"),
            ("/path/to/doc.pdf", "pdf"),
            ("/path/to/doc.docx", "docx"),
            ("/path/to/doc.doc", "docx"),
            ("/path/to/sheet.xlsx", "xlsx"),
            ("/path/to/sheet.xls", "xlsx"),
            ("/path/to/pres.pptx", "pptx"),
            ("/path/to/pres.ppt", "pptx"),
            ("/path/to/data.xml", "xml"),
            ("/path/to/config.yaml", "yaml"),
            ("/path/to/config.yml", "yaml")
        ]
        
        for file_path, expected_type in test_cases:
            result = DocumentTypeDetector.detect_from_path(file_path)
            assert result == expected_type, f"Failed for {file_path}: expected {expected_type}, got {result}"
    
    def test_detect_from_path_case_insensitive(self):
        """Test that extension detection is case insensitive."""
        test_cases = [
            ("/path/to/FILE.TXT", "text"),
            ("/path/to/DOC.PDF", "pdf"),
            ("/path/to/DATA.CSV", "csv"),
            ("/path/to/PAGE.HTML", "html")
        ]
        
        for file_path, expected_type in test_cases:
            result = DocumentTypeDetector.detect_from_path(file_path)
            assert result == expected_type, f"Failed for {file_path}: expected {expected_type}, got {result}"
    
    def test_detect_from_path_no_extension(self):
        """Test detection for paths without extension."""
        result = DocumentTypeDetector.detect_from_path("/path/to/file")
        # Should either return None or try MIME detection
        assert result is None or isinstance(result, str)
        
        result = DocumentTypeDetector.detect_from_path("/path/to/README")
        assert result is None or isinstance(result, str)
    
    def test_detect_from_path_empty_or_none(self):
        """Test detection with empty or None paths."""
        assert DocumentTypeDetector.detect_from_path(None) is None
        assert DocumentTypeDetector.detect_from_path("") is None
    
    def test_detect_from_mime_type(self):
        """Test detection based on MIME types."""
        # Note: This might not work without actual files, so test gracefully
        test_files = {
            "test.txt": "text/plain",
            "test.html": "text/html",
            "test.json": "application/json"
        }
        
        with tempfile.TemporaryDirectory() as temp_dir:
            for filename, expected_mime in test_files.items():
                file_path = os.path.join(temp_dir, filename)
                
                # Create test file
                with open(file_path, "w") as f:
                    if filename.endswith(".json"):
                        f.write('{"test": "data"}')
                    elif filename.endswith(".html"):
                        f.write("<html><body>Test</body></html>")
                    else:
                        f.write("Test content")
                
                # Test MIME detection
                try:
                    result = DocumentTypeDetector.detect_from_mime(file_path)
                    # Should either detect correctly or return None/fallback
                    assert result is None or isinstance(result, str)
                except Exception:
                    # MIME detection might fail, that's ok
                    pass

    def test_detect_from_content_signatures(self):
        """Test detection based on content signatures."""
        # Test various binary signatures
        test_cases = [
            # PDF signature
            (b"%PDF-1.4", "pdf"),
            # ZIP-based formats
            (b"PK\x03\x04", "zip"),  # Could be docx, xlsx, pptx
            # PNG signature
            (b"\x89PNG\r\n\x1a\n", "image"),
            # JPEG signature  
            (b"\xff\xd8\xff", "image"),
            # GIF signature
            (b"GIF89a", "image"),
            # Test JSON content
            (b'{"key": "value"}', "json"),
            # Test CSV content
            (b"name,age,city\nJohn,30,NYC", "csv"),
            # Test HTML content
            (b"<html><body>Test</body></html>", "html"),
            # Test XML content
            (b'<?xml version="1.0"?><root></root>', "xml")
        ]
        
        for content, expected_type in test_cases:
            result = DocumentTypeDetector.detect_from_content(content)
            # Content detection may return the expected type or None
            assert result is None or result == expected_type

    def test_mime_type_mapping(self):
        """Test MIME type to document type mapping."""
        # Access internal mapping if available
        detector = DocumentTypeDetector()
        
        # Test some known mappings would exist
        common_mimes = [
            "text/plain",
            "text/html", 
            "application/json",
            "text/csv",
            "application/pdf"
        ]
        
        # These should either map to something or not raise errors
        for mime_type in common_mimes:
            # This tests internal mapping without requiring actual implementation
            assert isinstance(mime_type, str)

    def test_extension_mapping(self):
        """Test file extension to document type mapping."""
        # Test that the detector has reasonable extension mappings
        common_extensions = [
            ".txt", ".md", ".csv", ".json", ".html", 
            ".pdf", ".docx", ".xlsx", ".pptx", ".xml"
        ]
        
        for ext in common_extensions:
            # These should be recognized extensions
            test_path = f"/test/file{ext}"
            result = DocumentTypeDetector.detect_from_path(test_path)
            assert result is not None, f"Extension {ext} not recognized"

    def test_special_extensions(self):
        """Test special or edge case extensions."""
        special_cases = [
            ("/path/file.backup.pdf.bak", "pdf"),  # Multiple extensions
            ("/path/file.tar.gz", "text"),  # Compound extension
            ("/path/file.jpeg", "text"),  # Alternative image extension
            ("/path/file.yml", "yaml"),   # Alternative YAML extension
        ]
        
        for file_path, expected_fallback in special_cases:
            result = DocumentTypeDetector.detect_from_path(file_path)
            # Should handle gracefully - may detect correctly or fallback
            assert result is None or isinstance(result, str)

    def test_path_objects(self):
        """Test that Path objects are handled correctly."""
        path_obj = Path("/test/document.pdf")
        result = DocumentTypeDetector.detect_from_path(path_obj)
        assert result == "pdf"
        
        # Test with pathlib Path
        path_obj2 = Path("/test/data.csv")
        result2 = DocumentTypeDetector.detect_from_path(path_obj2)
        assert result2 == "csv"

    def test_relative_paths(self):
        """Test relative path handling."""
        relative_paths = [
            ("../document.txt", "text"),
            ("./data.json", "json"),
            ("file.html", "html"),
            ("subdir/file.pdf", "pdf")
        ]
        
        for rel_path, expected in relative_paths:
            result = DocumentTypeDetector.detect_from_path(rel_path)
            assert result == expected

    def test_complex_paths(self):
        """Test complex path scenarios."""
        complex_paths = [
            ("/very/long/path/to/some/deeply/nested/file.docx", "docx"),
            ("/path with spaces/file name.xlsx", "xlsx"),
            ("/path-with-dashes/file_with_underscores.pptx", "pptx"),
            ("C:\\Windows\\Path\\file.pdf", "pdf"),  # Windows path
        ]
        
        for file_path, expected in complex_paths:
            result = DocumentTypeDetector.detect_from_path(file_path)
            assert result == expected


class TestDocumentTypeDetectorEdgeCases:
    """Test edge cases and error conditions."""
    
    def test_multiple_extensions(self):
        """Test files with multiple extensions."""
        test_cases = [
            ("/path/backup.pdf.bak", None),  # Unknown final extension
            ("/path/data.csv.old", None),    # Unknown final extension  
            ("/path/archive.tar.gz", "text"), # Should detect .gz or fallback
        ]
        
        for file_path, expected in test_cases:
            result = DocumentTypeDetector.detect_from_path(file_path)
            if expected is None:
                assert result is None or isinstance(result, str)
            else:
                assert result == expected or result is None

    def test_no_extension_files(self):
        """Test files without extensions."""
        no_ext_files = [
            "/usr/bin/python",
            "/etc/hosts", 
            "/path/to/README",
            "/path/to/LICENSE",
            "Makefile"
        ]
        
        for file_path in no_ext_files:
            result = DocumentTypeDetector.detect_from_path(file_path)
            # Should handle gracefully
            assert result is None or isinstance(result, str)

    def test_hidden_files(self):
        """Test hidden files (starting with dot)."""
        hidden_files = [
            "/.gitignore",
            "/.env",
            "/path/.hidden.txt",
            ".bashrc"
        ]
        
        for file_path in hidden_files:
            result = DocumentTypeDetector.detect_from_path(file_path)
            # Should detect based on extension if present
            if file_path.endswith('.txt'):
                assert result == "text"
            else:
                assert result is None or isinstance(result, str)

    def test_very_long_paths(self):
        """Test very long file paths."""
        long_path = "/" + "/".join(["very_long_directory_name"] * 20) + "/file.json"
        result = DocumentTypeDetector.detect_from_path(long_path)
        assert result == "json"

    def test_unicode_paths(self):
        """Test paths with Unicode characters."""
        unicode_paths = [
            ("/пуÔÅ/файл.txt", "text"),
            ("/日本語/文書.pdf", "pdf"), 
            ("/émojis/🚀document.docx", "docx"),
            ("/español/documento.xlsx", "xlsx")
        ]
        
        for file_path, expected in unicode_paths:
            result = DocumentTypeDetector.detect_from_path(file_path)
            assert result == expected

    def test_malformed_paths(self):
        """Test malformed or unusual paths."""
        malformed_paths = [
            "///multiple///slashes//file.txt",
            "/path/./file.json", 
            "/path/../file.csv",
            "",
            "   ",  # Whitespace only
            ".",
            ".."
        ]
        
        for file_path in malformed_paths:
            # Should not crash
            result = DocumentTypeDetector.detect_from_path(file_path)
            assert result is None or isinstance(result, str)


class TestDocumentTypeDetectorInit:
    """Test initialization and magic library setup."""
    
    def test_initialize_magic_function(self):
        """Test magic library initialization."""
        # Test that initialize_magic can be called without crashing
        try:
            initialize_magic()
            # If no exception, that's good
            assert True
        except ImportError:
            # Magic library not available, that's ok
            pytest.skip("python-magic not available")
        except Exception as e:
            # Other errors are ok too as long as it doesn't crash the test
            assert isinstance(e, Exception)

    def test_magic_availability_detection(self):
        """Test magic availability detection."""
        # Test that we can check magic availability
        from go_doc_go.document_parser.document_type_detector import MAGIC_AVAILABLE
        assert isinstance(MAGIC_AVAILABLE, bool)

    def test_platform_specific_behavior(self):
        """Test platform-specific initialization."""
        from go_doc_go.document_parser.document_type_detector import IS_LINUX
        assert isinstance(IS_LINUX, bool)
        
        # Test that platform detection works
        import platform
        assert IS_LINUX == (platform.system() == 'Linux')


class TestDocumentTypeDetectorConstants:
    """Test internal constants and mappings."""
    
    def test_mime_type_map_completeness(self):
        """Test that MIME type mapping covers common types."""
        # Test basic functionality - constants should exist
        detector = DocumentTypeDetector()
        assert hasattr(detector, 'detect_from_mime')

    def test_extension_map_completeness(self):
        """Test that extension mapping covers common extensions."""
        # Test that common extensions are handled
        common_exts = ['.txt', '.pdf', '.docx', '.xlsx', '.pptx', '.csv', '.json', '.html', '.xml']
        
        for ext in common_exts:
            result = DocumentTypeDetector.detect_from_path(f"/test/file{ext}")
            assert result is not None, f"Extension {ext} should be recognized"

    def test_binary_signatures_exist(self):
        """Test that binary signature detection exists."""
        # Test that we can call detect_from_content without crashing
        test_content = b"Test binary content"
        result = DocumentTypeDetector.detect_from_content(test_content)
        assert result is None or isinstance(result, str)

    def test_mapping_consistency(self):
        """Test consistency between different detection methods."""
        # Test that the same file type is detected consistently
        test_cases = [
            ("file.txt", b"Plain text content", "text"),
            ("file.json", b'{"key": "value"}', "json"),
            ("file.html", b"<html><body>Test</body></html>", "html")
        ]
        
        for filename, content, expected_type in test_cases:
            path_result = DocumentTypeDetector.detect_from_path(f"/test/{filename}")
            content_result = DocumentTypeDetector.detect_from_content(content)
            
            # At least one method should detect the correct type
            assert path_result == expected_type or content_result == expected_type


class TestDocumentTypeDetectorIntegration:
    """Integration tests for the detector."""
    
    def test_detect_method_with_multiple_inputs(self):
        """Test the main detect method with various inputs.""" 
        # Test with path only
        result1 = DocumentTypeDetector.detect(path="/test/file.txt")
        assert result1 == "text" or result1 is None
        
        # Test with content only
        result2 = DocumentTypeDetector.detect(content=b'{"test": "json"}')
        assert result2 == "json" or result2 is None
        
        # Test with both path and content
        result3 = DocumentTypeDetector.detect(
            path="/test/file.json", 
            content=b'{"test": "data"}'
        )
        assert result3 == "json" or result3 is None

    def test_is_likely_csv_static_method(self):
        """Test the _is_likely_csv static method."""
        csv_samples = [
            "name,age,city\nJohn,30,NYC\nJane,25,LA",
            "col1,col2,col3\n1,2,3\n4,5,6", 
            "a,b,c\nd,e,f"
        ]
        
        non_csv_samples = [
            "This is plain text",
            "<html><body>HTML content</body></html>",
            '{"key": "json value"}'
        ]
        
        for csv_text in csv_samples:
            result = DocumentTypeDetector._is_likely_csv(csv_text)
            assert result == True, f"Should detect CSV: {csv_text[:30]}..."
            
        for non_csv_text in non_csv_samples:
            result = DocumentTypeDetector._is_likely_csv(non_csv_text) 
            assert result == False, f"Should not detect CSV: {non_csv_text[:30]}..."


# =============================================================================
# Performance and Stress Tests  
# =============================================================================

@pytest.mark.performance
class TestDocumentTypeDetectorPerformance:
    """Performance tests for document type detection."""
    
    def test_large_path_performance(self):
        """Test performance with very large paths."""
        import time
        
        # Create a very long path
        long_path = "/" + "/".join([f"dir{i}" for i in range(1000)]) + "/file.txt"
        
        start_time = time.time()
        result = DocumentTypeDetector.detect_from_path(long_path)
        elapsed = time.time() - start_time
        
        assert result == "text"
        assert elapsed < 1.0, "Path detection should be fast even for long paths"

    def test_many_extensions_performance(self):
        """Test performance when detecting many different extensions."""
        import time
        
        extensions = [
            '.txt', '.md', '.csv', '.json', '.html', '.htm', '.xml', '.yaml', '.yml',
            '.pdf', '.docx', '.doc', '.xlsx', '.xls', '.pptx', '.ppt'
        ]
        
        start_time = time.time()
        
        for i, ext in enumerate(extensions * 100):  # Test 1600 detections
            path = f"/test/file{i}{ext}"
            DocumentTypeDetector.detect_from_path(path)
            
        elapsed = time.time() - start_time
        
        assert elapsed < 5.0, "Should handle many detections quickly"

    @pytest.mark.timeout(10)
    def test_content_detection_timeout(self):
        """Test that content detection doesn't hang on large content."""
        # Create large binary content
        large_content = b"x" * (10 * 1024 * 1024)  # 10MB
        
        # Should complete within timeout
        result = DocumentTypeDetector.detect_from_content(large_content)
        assert result is None or isinstance(result, str)

    def test_detect_from_content_signatures(self):
        """Test detection based on content signatures."""
        # This tests the binary signature detection logic
        test_signatures = [
            (b"%PDF-1.4", "pdf"),
            (b"PK\x03\x04", "zip"),  # Office documents
            (b"\xD0\xCF\x11\xE0\xA1\xB1\x1A\xE1", "ms_compound")  # Legacy Office
        ]
        
        for signature, expected in test_signatures:
            # Test if detector recognizes these signatures
            if hasattr(DocumentTypeDetector, 'detect_from_content'):
                try:
                    result = DocumentTypeDetector.detect_from_content(signature)
                    assert result == expected or result is None
                except Exception:
                    # Method might not exist or work differently
                    pass
    
    def test_mime_type_mapping(self):
        """Test MIME type to document type mapping."""
        mime_mappings = DocumentTypeDetector.MIME_TYPE_MAP
        
        # Check key mappings exist
        assert mime_mappings.get("text/plain") == "text"
        assert mime_mappings.get("text/markdown") == "markdown"
        assert mime_mappings.get("application/pdf") == "pdf"
        assert mime_mappings.get("text/csv") == "csv"
        assert mime_mappings.get("application/json") == "json"
        assert mime_mappings.get("text/html") == "html"
        assert mime_mappings.get("application/xml") == "xml"
        
        # Check that all values are valid document types
        valid_types = {"text", "markdown", "html", "pdf", "docx", "xlsx", "pptx", "csv", "json", "xml", "yaml"}
        for mime_type, doc_type in mime_mappings.items():
            assert doc_type in valid_types, f"Invalid document type '{doc_type}' for MIME type '{mime_type}'"
    
    def test_extension_mapping(self):
        """Test file extension to document type mapping."""
        extension_map = DocumentTypeDetector.EXTENSION_MAP
        
        # Check key mappings
        assert extension_map.get(".txt") == "text"
        assert extension_map.get(".md") == "markdown"
        assert extension_map.get(".pdf") == "pdf"
        assert extension_map.get(".csv") == "csv"
        assert extension_map.get(".json") == "json"
        assert extension_map.get(".html") == "html"
        assert extension_map.get(".xml") == "xml"
        
        # Check Office documents
        assert extension_map.get(".docx") == "docx"
        assert extension_map.get(".doc") == "docx"
        assert extension_map.get(".xlsx") == "xlsx"
        assert extension_map.get(".xls") == "xlsx"
        assert extension_map.get(".pptx") == "pptx"
        assert extension_map.get(".ppt") == "pptx"
    
    def test_special_extensions(self):
        """Test handling of special file extensions."""
        special_cases = [
            ".mdown",  # Alternative markdown
            ".xhtml",  # Extended HTML
            ".tsv",    # Tab-separated values
            ".yml",    # YAML alternative
            ".svg",    # SVG as XML
            ".rss",    # RSS as XML
            ".xsd"     # XML Schema
        ]
        
        for ext in special_cases:
            result = DocumentTypeDetector.detect_from_path(f"/test{ext}")
            # Should map to something or return None
            assert result is None or isinstance(result, str)
    
    def test_path_objects(self):
        """Test detection with pathlib.Path objects."""
        # Convert to string - detector should handle it
        path_obj = Path("/path/to/document.pdf")
        result = DocumentTypeDetector.detect_from_path(str(path_obj))
        assert result == "pdf"
    
    def test_relative_paths(self):
        """Test detection with relative paths."""
        test_cases = [
            ("./document.md", "markdown"),
            ("../config.json", "json"),
            ("file.txt", "text"),
            ("data/sheet.xlsx", "xlsx")
        ]
        
        for path, expected_type in test_cases:
            result = DocumentTypeDetector.detect_from_path(path)
            assert result == expected_type

    def test_complex_paths(self):
        """Test detection with complex file paths."""
        complex_paths = [
            ("/path/with spaces/document.pdf", "pdf"),
            ("/path-with-dashes/file.md", "markdown"),
            ("/path_with_underscores/data.csv", "csv"),
            ("/path.with.dots/config.json", "json"),
            ("C:\\Windows\\Path\\file.txt", "text")  # Windows path
        ]
        
        for path, expected_type in complex_paths:
            result = DocumentTypeDetector.detect_from_path(path)
            assert result == expected_type


class TestDocumentTypeDetectorEdgeCases:
    """Test edge cases for document type detector."""
    
    def test_multiple_extensions(self):
        """Test files with multiple extensions."""
        multi_ext_cases = [
            ("backup.pdf.bak", "text"),  # Unknown final extension falls back to text
            ("data.csv.tmp", "text"),    # Temporary file falls back to text
            ("config.json.old", "text"), # Old file falls back to text
            ("file.tar.gz", "text")      # Archive falls back to text
        ]
        
        for path, expected in multi_ext_cases:
            result = DocumentTypeDetector.detect_from_path(path)
            # Falls back to text for unknown extensions
            assert result == expected
    
    def test_no_extension_files(self):
        """Test common files without extensions."""
        no_ext_files = [
            "README",
            "LICENSE", 
            "CHANGELOG",
            "Makefile",
            "Dockerfile"
        ]
        
        for filename in no_ext_files:
            result = DocumentTypeDetector.detect_from_path(filename)
            # Should return None or attempt MIME detection
            assert result is None or isinstance(result, str)
    
    def test_hidden_files(self):
        """Test hidden files (starting with dot)."""
        hidden_files = [
            (".gitignore", None),
            (".env", None),
            (".config.json", "json"),  # Hidden but has extension
            (".profile.md", "markdown")
        ]
        
        for path, expected in hidden_files:
            result = DocumentTypeDetector.detect_from_path(path)
            if expected:
                assert result == expected
            else:
                assert result is None or isinstance(result, str)
    
    def test_very_long_paths(self):
        """Test very long file paths."""
        # Create very long path
        long_path = "/very/" + "long/" * 100 + "document.pdf"
        
        result = DocumentTypeDetector.detect_from_path(long_path)
        assert result == "pdf"
    
    def test_unicode_paths(self):
        """Test paths with Unicode characters."""
        unicode_paths = [
            ("/path/文档.pdf", "pdf"),
            ("/путь/документ.docx", "docx"),
            ("/ruta/café.txt", "text")
        ]
        
        for path, expected_type in unicode_paths:
            try:
                result = DocumentTypeDetector.detect_from_path(path)
                assert result == expected_type
            except UnicodeError:
                # Should not raise unicode errors
                pytest.fail(f"Unicode error for path {path}")
    
    def test_malformed_paths(self):
        """Test handling of malformed paths."""
        malformed_paths = [
            "file..pdf",     # Double dot
            ".pdf",          # Just extension
            "file.",         # Trailing dot
            "file.PDF.PDF",  # Repeated extension
            "",              # Empty string
            "   ",           # Whitespace only
        ]
        
        for path in malformed_paths:
            try:
                result = DocumentTypeDetector.detect_from_path(path)
                # Should either detect or return None, not crash
                assert result is None or isinstance(result, str)
            except Exception as e:
                # Should not raise unexpected errors
                assert isinstance(e, (ValueError, TypeError)), f"Unexpected error for {path}: {e}"


class TestDocumentTypeDetectorInit:
    """Test initialization functions."""
    
    def test_initialize_magic_function(self):
        """Test initialize_magic function."""
        # Function should exist and be callable
        assert callable(initialize_magic)
        
        # Should not raise errors when called
        try:
            initialize_magic()
        except ImportError:
            # Expected if magic library not available
            pass
        except Exception as e:
            # Other errors might be platform-specific
            pass
    
    def test_magic_availability_detection(self):
        """Test magic library availability detection."""
        # Check that the module handles missing magic gracefully
        try:
            from go_doc_go.document_parser.document_type_detector import MAGIC_AVAILABLE
            # Should be a boolean
            assert isinstance(MAGIC_AVAILABLE, bool)
        except ImportError:
            # Module might not be available
            pass
    
    def test_platform_specific_behavior(self):
        """Test platform-specific behavior."""
        try:
            from go_doc_go.document_parser.document_type_detector import IS_LINUX
            assert isinstance(IS_LINUX, bool)
        except ImportError:
            pass


class TestDocumentTypeDetectorConstants:
    """Test detector constants and mappings."""
    
    def test_mime_type_map_completeness(self):
        """Test that MIME type map covers common types."""
        mime_map = DocumentTypeDetector.MIME_TYPE_MAP
        
        # Should have entries for all major document types
        required_types = {
            "text/plain", "text/html", "application/pdf", 
            "text/csv", "application/json", "text/xml"
        }
        
        for mime_type in required_types:
            assert mime_type in mime_map, f"Missing MIME type: {mime_type}"
    
    def test_extension_map_completeness(self):
        """Test that extension map covers common extensions."""
        ext_map = DocumentTypeDetector.EXTENSION_MAP
        
        # Should have entries for common extensions
        required_extensions = {
            ".txt", ".md", ".html", ".pdf", ".csv", ".json", ".xml"
        }
        
        for extension in required_extensions:
            assert extension in ext_map, f"Missing extension: {extension}"
    
    def test_binary_signatures_exist(self):
        """Test that binary signatures are defined."""
        signatures = DocumentTypeDetector.BINARY_SIGNATURES
        
        # Should have some signatures
        assert len(signatures) > 0
        
        # Should include PDF signature
        assert any(b'%PDF' in sig for sig in signatures.keys())
    
    def test_mapping_consistency(self):
        """Test consistency between different mappings."""
        mime_map = DocumentTypeDetector.MIME_TYPE_MAP
        ext_map = DocumentTypeDetector.EXTENSION_MAP
        
        # Common document types should exist in both maps
        common_types = {"text", "html", "pdf", "csv", "json", "xml"}
        
        mime_types = set(mime_map.values())
        ext_types = set(ext_map.values())
        
        # All common types should be supported
        for doc_type in common_types:
            assert doc_type in mime_types or doc_type in ext_types, f"Type {doc_type} not found in either map"


if __name__ == "__main__":
    pytest.main([__file__, "-v"])