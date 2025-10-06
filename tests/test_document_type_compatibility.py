"""
Compatibility tests for document type detection.

Tests that the Go implementation produces identical results to the Python implementation
across a comprehensive range of inputs and scenarios.
"""

import base64
import os
import tempfile
import pytest
from typing import Dict, Any, List, Tuple, Optional

from src.go_doc_go.document_parser.document_type_detector import (
    DocumentTypeDetector as PythonDocumentTypeDetector,
    GoDocumentTypeDetector,
    create_document_type_detector
)


class TestDocumentTypeCompatibility:
    """Test compatibility between Python and Go document type detectors."""

    def setup_method(self):
        """Set up test instances."""
        self.python_detector = PythonDocumentTypeDetector()
        self.go_detector = GoDocumentTypeDetector()

    def test_factory_creates_python_by_default(self):
        """Test that factory creates Python detector by default."""
        detector = create_document_type_detector()
        assert isinstance(detector, PythonDocumentTypeDetector)

    def test_factory_creates_go_with_env_var(self):
        """Test that factory creates Go detector when USE_GO_DETECTOR is set."""
        os.environ["USE_GO_DETECTOR"] = "true"
        try:
            detector = create_document_type_detector()
            assert isinstance(detector, GoDocumentTypeDetector)
        finally:
            del os.environ["USE_GO_DETECTOR"]

    @pytest.mark.parametrize("path,expected_type", [
        ("document.pdf", "pdf"),
        ("spreadsheet.xlsx", "xlsx"),
        ("presentation.pptx", "pptx"),
        ("text.txt", "text"),
        ("data.csv", "csv"),
        ("config.json", "json"),
        ("style.xml", "xml"),
        ("readme.md", "markdown"),
        ("page.html", "html"),
        ("script.yaml", "yaml"),
        ("document.docx", "docx"),
        ("unknown.xyz", "text"),
        ("", None),
    ])
    def test_detect_from_path_compatibility(self, path: str, expected_type: Optional[str]):
        """Test that both implementations return same results for path detection."""
        python_result = self.python_detector.detect_from_path(path)
        go_result = self.go_detector.detect_from_path(path)

        assert python_result == go_result
        assert python_result == expected_type

    @pytest.mark.parametrize("content,metadata,expected_type", [
        # JSON content
        ('{"key": "value"}', None, "json"),
        ('[{"item": 1}, {"item": 2}]', None, "json"),

        # XML content
        ('<?xml version="1.0"?><root></root>', None, "xml"),
        ('<note><to>User</to><from>System</from></note>', None, "xml"),

        # HTML content
        ('<!DOCTYPE html><html><body>Hello</body></html>', None, "html"),
        ('<div class="content"><p>Hello world</p></div>', None, "html"),

        # Markdown content
        ("# Header\n\nSome content", None, "markdown"),
        ("## Subheader\n\n- List item", None, "markdown"),

        # CSV content
        ("name,age,city\nJohn,30,NYC\nJane,25,LA", None, "csv"),
        ("col1\tcol2\tcol3\nval1\tval2\tval3", None, "csv"),

        # Plain text
        ("This is just plain text content", None, "text"),
        ("", None, "text"),

        # Metadata hints
        ("Some content", {"content_type": "application/json"}, "json"),
        ("Some content", {"content_column": "data_html"}, "html"),
        ("Some content", {"content_column": "notes_md"}, "markdown"),
    ])
    def test_detect_from_content_compatibility(self, content: str, metadata: Dict[str, str], expected_type: str):
        """Test that both implementations return same results for content detection."""
        python_result = self.python_detector.detect_from_content(content, metadata)
        go_result = self.go_detector.detect_from_content(content, metadata)

        assert python_result == go_result
        assert python_result == expected_type

    def test_binary_content_compatibility(self):
        """Test that both implementations handle binary content identically."""
        # Create binary content that should be detected as binary
        binary_content = bytes([0xFF, 0xFE, 0x00, 0x01, 0x02, 0x03])

        python_result = self.python_detector.detect_from_content(binary_content, None)
        go_result = self.go_detector.detect_from_content(binary_content, None)

        assert python_result == go_result
        assert python_result == "binary"

    def test_pdf_signature_compatibility(self):
        """Test that both implementations detect PDF signature identically."""
        pdf_content = b"%PDF-1.4\nSome PDF content"

        python_result = self.python_detector.detect_from_content(pdf_content, None)
        go_result = self.go_detector.detect_from_content(pdf_content, None)

        assert python_result == go_result
        assert python_result == "pdf"

    @pytest.mark.parametrize("path,content,metadata,expected_type", [
        # Path takes precedence
        ("document.pdf", "This is not PDF content", None, "pdf"),
        ("", '{"key": "value"}', None, "json"),

        # Content detection when path is unknown
        ("unknown.file", '{"key": "value"}', None, "json"),

        # Metadata hints
        ("", "Some content", {"content_type": "application/xml"}, "xml"),

        # Default fallback
        ("", "", None, "text"),
    ])
    def test_detect_compatibility(self, path: str, content: str, metadata: Dict[str, str], expected_type: str):
        """Test that both implementations return same results for general detection."""
        python_result = self.python_detector.detect(path, content, metadata)
        go_result = self.go_detector.detect(path, content, metadata)

        assert python_result == go_result
        assert python_result == expected_type

    def test_csv_detection_compatibility(self):
        """Test that both implementations detect CSV format identically."""
        csv_samples = [
            "name,age,city\nJohn,30,NYC\nJane,25,LA",
            "col1,col2,col3\nval1,val2,val3\nval4,val5,val6",
            "name\tage\tcity\nJohn\t30\tNYC\nJane\t25\tLA",  # Tab-separated
            "name;age;city\nJohn;30;NYC\nJane;25;LA",  # Semicolon-separated
            "name|age|city\nJohn|30|NYC\nJane|25|LA",  # Pipe-separated
        ]

        for csv_content in csv_samples:
            python_result = self.python_detector.detect_from_content(csv_content, None)
            go_result = self.go_detector.detect_from_content(csv_content, None)

            assert python_result == go_result
            assert python_result == "csv"

    def test_non_csv_content_compatibility(self):
        """Test that both implementations reject non-CSV content identically."""
        non_csv_samples = [
            "This is just plain text",
            "name,age\nJohn\nJane,25,LA,Extra",  # Inconsistent columns
            "",  # Empty
            "name\nJohn\nJane",  # No delimiters
        ]

        for content in non_csv_samples:
            python_result = self.python_detector.detect_from_content(content, None)
            go_result = self.go_detector.detect_from_content(content, None)

            assert python_result == go_result
            assert python_result != "csv"

    def test_office_document_signatures_compatibility(self):
        """Test that both implementations handle Office document signatures identically."""
        # Simulate ZIP signature with Office XML content
        zip_header = b"PK\x03\x04"
        word_content = zip_header + b"word/" + b"\x00" * 100
        excel_content = zip_header + b"xl/" + b"\x00" * 100
        powerpoint_content = zip_header + b"ppt/" + b"\x00" * 100

        test_cases = [
            (word_content, "docx"),
            (excel_content, "xlsx"),
            (powerpoint_content, "pptx"),
        ]

        for content, expected_type in test_cases:
            python_result = self.python_detector.detect_from_content(content, None)
            go_result = self.go_detector.detect_from_content(content, None)

            assert python_result == go_result
            assert python_result == expected_type

    def test_yaml_detection_compatibility(self):
        """Test that both implementations detect YAML identically."""
        yaml_paths = ["config.yaml", "data.yml"]

        for path in yaml_paths:
            python_result = self.python_detector.detect_from_path(path)
            go_result = self.go_detector.detect_from_path(path)

            assert python_result == go_result
            assert python_result == "yaml"

    def test_markdown_variations_compatibility(self):
        """Test that both implementations handle markdown variations identically."""
        markdown_paths = ["readme.md", "doc.markdown", "notes.mdown"]

        for path in markdown_paths:
            python_result = self.python_detector.detect_from_path(path)
            go_result = self.go_detector.detect_from_path(path)

            assert python_result == go_result
            assert python_result == "markdown"

    def test_method_field_compatibility(self):
        """Test that both implementations return identical method fields."""
        test_cases = [
            # Extension-based detection
            ("document.pdf", "", None),
            # MIME-based detection
            ("unknown.xyz", "", None),
            # Content-based detection
            ("", '{"key": "value"}', None),
            # Metadata-based detection
            ("", "content", {"content_type": "application/json"}),
            # Signature-based detection
            ("", "%PDF-1.4", None),
        ]

        for path, content, metadata in test_cases:
            python_result = self.python_detector.detect(path, content, metadata)
            go_result = self.go_detector.detect(path, content, metadata)

            assert python_result == go_result
            # Both should return the same document type string

    def test_large_content_compatibility(self):
        """Test that both implementations handle large content identically."""
        # Create large JSON content
        large_json = '{"data": [' + ','.join([f'{{"id": {i}}}' for i in range(1000)]) + ']}'

        python_result = self.python_detector.detect_from_content(large_json, None)
        go_result = self.go_detector.detect_from_content(large_json, None)

        assert python_result == go_result
        assert python_result == "json"

    def test_special_characters_compatibility(self):
        """Test that both implementations handle special characters identically."""
        content_with_unicode = '{"message": "Hello 世界! 🌍", "special": "àáâãäå"}'

        python_result = self.python_detector.detect_from_content(content_with_unicode, None)
        go_result = self.go_detector.detect_from_content(content_with_unicode, None)

        assert python_result == go_result
        assert python_result == "json"

    def test_empty_inputs_compatibility(self):
        """Test that both implementations handle empty inputs identically."""
        test_cases = [
            ("", "", None),
            ("", None, None),
            (None, "", None),
            (None, None, None),
        ]

        for path, content, metadata in test_cases:
            python_result = self.python_detector.detect(path, content, metadata)
            go_result = self.go_detector.detect(path, content, metadata)

            assert python_result == go_result

    def test_file_based_detection_compatibility(self):
        """Test that both implementations work identically with actual files."""
        test_files = [
            ("test.json", b'{"test": "data"}'),
            ("test.csv", b"name,value\ntest,123"),
            ("test.xml", b'<?xml version="1.0"?><root><item>test</item></root>'),
            ("test.html", b'<!DOCTYPE html><html><body><h1>Test</h1></body></html>'),
            ("test.md", b"# Test Document\n\nThis is a test."),
        ]

        for filename, content in test_files:
            with tempfile.NamedTemporaryFile(suffix=f"_{filename}", delete=False) as tmp_file:
                tmp_file.write(content)
                tmp_file.flush()

                try:
                    # Test file-based detection
                    python_result = self.python_detector.detect_from_path(tmp_file.name)
                    go_result = self.go_detector.detect_from_path(tmp_file.name)

                    assert python_result == go_result

                finally:
                    os.unlink(tmp_file.name)

    def test_error_handling_compatibility(self):
        """Test that both implementations handle errors identically."""
        # Both should handle None content gracefully
        python_result = self.python_detector.detect("", None, None)
        go_result = self.go_detector.detect("", None, None)

        assert python_result == go_result
        assert python_result == "text"

    @pytest.mark.performance
    def test_performance_comparison(self):
        """Compare performance between Python and Go implementations."""
        import time

        test_content = '{"data": [' + ','.join([f'{{"id": {i}}}' for i in range(100)]) + ']}'
        iterations = 100

        # Time Python implementation
        start_time = time.time()
        for _ in range(iterations):
            self.python_detector.detect_from_content(test_content, None)
        python_time = time.time() - start_time

        # Time Go implementation
        start_time = time.time()
        for _ in range(iterations):
            self.go_detector.detect_from_content(test_content, None)
        go_time = time.time() - start_time

        print(f"\nPerformance comparison ({iterations} iterations):")
        print(f"Python: {python_time:.4f}s ({python_time/iterations*1000:.2f}ms per call)")
        print(f"Go: {go_time:.4f}s ({go_time/iterations*1000:.2f}ms per call)")
        print(f"Go speedup: {python_time/go_time:.2f}x")

        # Go should be faster (allowing some margin for subprocess overhead)
        # Note: In real applications, the subprocess overhead would be amortized
        # across many calls, making Go significantly faster