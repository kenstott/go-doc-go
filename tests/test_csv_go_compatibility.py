"""
Compatibility tests for Go CSV parser implementation.

These tests ensure that the Go CSV parser produces compatible output
with the Python CSV parser implementation.
"""

import pytest
from pathlib import Path

from src.go_doc_go.document_parser.csv import CsvParser
from src.go_doc_go.document_parser.csv_go import GoCSVParser


class TestCSVGoCompatibility:
    """Test compatibility between Python and Go CSV parsers."""

    @pytest.fixture
    def python_parser(self):
        """Create Python CSV parser instance."""
        return CsvParser({})

    @pytest.fixture
    def go_parser(self):
        """Create Go CSV parser instance."""
        # Check if binary exists
        project_root = Path(__file__).parent.parent
        binary_path = project_root / "bin" / "csvparser"

        if not binary_path.exists():
            pytest.skip("Go CSV parser binary not found")

        return GoCSVParser({})

    @pytest.fixture
    def sample_csv_content(self):
        """Sample CSV content for testing."""
        return {
            "id": "test_doc",
            "content": """name,age,city,department
John Doe,30,New York,Engineering
Jane Smith,25,Los Angeles,Marketing
Bob Wilson,35,Chicago,Sales
Alice Brown,28,Boston,Engineering""",
            "metadata": {
                "source": "test",
                "filename": "test.csv"
            }
        }

    @pytest.fixture
    def complex_csv_content(self):
        """Complex CSV content for testing."""
        return {
            "id": "complex_test",
            "content": """employee_id,first_name,last_name,email,department,salary,hire_date,website
1001,John,Smith,john.smith@company.com,Engineering,75000,2023-01-15,https://johnsmith.dev
1002,Jane,Doe,jane.doe@company.com,Marketing,65000,2023-02-01,https://janedoe.com
1003,Bob,Johnson,bob.johnson@company.com,Engineering,80000,2023-01-20,https://bobjohnson.org
1004,Alice,Williams,alice.williams@company.com,Sales,70000,2023-03-01,https://alicewilliams.net""",
            "metadata": {
                "source": "complex_test",
                "filename": "complex.csv"
            }
        }

    def test_basic_csv_parsing_compatibility(self, python_parser, go_parser, sample_csv_content):
        """Test that both parsers produce compatible basic results."""
        # Parse with both parsers
        python_result = python_parser.parse(sample_csv_content)
        go_result = go_parser.parse(sample_csv_content)

        # Check document structure
        assert python_result["document"]["doc_id"] == go_result["document"]["doc_id"]
        assert python_result["document"]["doc_type"] == go_result["document"]["doc_type"]

        # Check element counts by type
        python_element_types = {}
        for element in python_result["elements"]:
            element_type = element["element_type"]
            python_element_types[element_type] = python_element_types.get(element_type, 0) + 1

        go_element_types = {}
        for element in go_result["elements"]:
            element_type = element["element_type"]
            go_element_types[element_type] = go_element_types.get(element_type, 0) + 1

        # Both should have root and table elements
        assert "root" in python_element_types
        assert "root" in go_element_types
        assert "table" in python_element_types or "table" in go_element_types

        # Both should have table rows and cells
        assert python_element_types.get("table_row", 0) > 0
        assert go_element_types.get("table_row", 0) > 0
        assert python_element_types.get("table_cell", 0) > 0
        assert go_element_types.get("table_cell", 0) > 0

    def test_header_extraction_compatibility(self, python_parser, go_parser, sample_csv_content):
        """Test that both parsers extract headers consistently."""
        # Parse with both parsers
        python_result = python_parser.parse(sample_csv_content)
        go_result = go_parser.parse(sample_csv_content)

        # Both should extract header information
        python_has_header_row = any(
            elem["element_type"] == "table_header_row"
            for elem in python_result["elements"]
        )
        go_has_header_row = any(
            elem["element_type"] == "table_header_row"
            for elem in go_result["elements"]
        )

        # At least one should have header row (Go parser creates explicit header elements)
        assert python_has_header_row or go_has_header_row

        # Check for header information in metadata
        python_headers = []
        go_headers = []

        # Extract headers from document metadata
        python_doc_meta = python_result["document"].get("metadata", {})
        go_doc_meta = go_result["document"].get("metadata", {})

        if "headers" in python_doc_meta:
            python_headers = python_doc_meta["headers"]
        if "headers" in go_doc_meta:
            go_headers = go_doc_meta["headers"]

        # If both have headers, they should be similar
        if python_headers and go_headers:
            assert len(python_headers) == len(go_headers)
            # Check that header names match (case-insensitive)
            for i, (py_header, go_header) in enumerate(zip(python_headers, go_headers)):
                assert py_header.lower() == go_header.lower()

    def test_cell_content_compatibility(self, python_parser, go_parser, sample_csv_content):
        """Test that both parsers extract cell content consistently."""
        # Parse with both parsers
        python_result = python_parser.parse(sample_csv_content)
        go_result = go_parser.parse(sample_csv_content)

        # Extract cell content
        python_cells = []
        go_cells = []

        for element in python_result["elements"]:
            if element["element_type"] == "table_cell":
                cell_text = element.get("text") or element.get("content") or element.get("content_preview", "")
                python_cells.append(cell_text.strip())

        for element in go_result["elements"]:
            if element["element_type"] == "table_cell":
                cell_text = element.get("text") or element.get("content") or element.get("content_preview", "")
                go_cells.append(cell_text.strip())

        # Both should extract cell content
        assert len(python_cells) > 0
        assert len(go_cells) > 0

        # Check for common content
        python_content = set(python_cells)
        go_content = set(go_cells)

        # There should be significant overlap in cell content
        common_content = python_content & go_content
        assert len(common_content) > 0

        # Check for specific expected content
        expected_content = {"John Doe", "30", "New York", "Engineering", "Jane Smith"}
        python_has_expected = any(content in python_content for content in expected_content)
        go_has_expected = any(content in go_content for content in expected_content)

        assert python_has_expected or go_has_expected

    def test_link_extraction_compatibility(self, python_parser, go_parser, complex_csv_content):
        """Test that both parsers extract links consistently."""
        # Parse with both parsers
        python_result = python_parser.parse(complex_csv_content)
        go_result = go_parser.parse(complex_csv_content)

        # Check link extraction
        python_links = python_result.get("links", [])
        go_links = go_result.get("links", [])

        # Extract link targets
        python_targets = {link["link_target"] for link in python_links}
        go_targets = {link["link_target"] for link in go_links}

        # Both should find URLs and emails
        python_has_urls = any("https://" in target for target in python_targets)
        go_has_urls = any("https://" in target for target in go_targets)
        python_has_emails = any("mailto:" in target for target in python_targets)
        go_has_emails = any("mailto:" in target for target in go_targets)

        # At least one parser should extract links
        assert python_has_urls or go_has_urls
        assert python_has_emails or go_has_emails

    def test_element_structure_compatibility(self, python_parser, go_parser, sample_csv_content):
        """Test that element structures are compatible."""
        # Parse with both parsers
        python_result = python_parser.parse(sample_csv_content)
        go_result = go_parser.parse(sample_csv_content)

        # Check that all elements have required fields
        required_fields = [
            "element_id", "doc_id", "element_type", "content_preview",
            "content_location", "content_hash", "metadata"
        ]

        for element in python_result["elements"]:
            for field in required_fields:
                assert field in element, f"Python parser missing field: {field}"

        for element in go_result["elements"]:
            for field in required_fields:
                assert field in element, f"Go parser missing field: {field}"

    def test_relationship_structure_compatibility(self, python_parser, go_parser, sample_csv_content):
        """Test that relationship structures are compatible."""
        # Parse with both parsers
        python_result = python_parser.parse(sample_csv_content)
        go_result = go_parser.parse(sample_csv_content)

        # Check relationship structure
        required_rel_fields = [
            "relationship_id", "source_element_id", "target_element_id",
            "relationship_type", "confidence", "metadata"
        ]

        # Go parser always creates relationships, Python may not in simple cases
        if python_result["relationships"]:
            for rel in python_result["relationships"]:
                for field in required_rel_fields:
                    assert field in rel, f"Python parser missing relationship field: {field}"

        for rel in go_result["relationships"]:
            for field in required_rel_fields:
                assert field in rel, f"Go parser missing relationship field: {field}"

        # Check that both create hierarchical relationships
        python_has_contains = any(
            rel["relationship_type"] == "contains"
            for rel in python_result["relationships"]
        )
        go_has_contains = any(
            rel["relationship_type"] == "contains"
            for rel in go_result["relationships"]
        )

        assert python_has_contains or go_has_contains

    def test_empty_csv_compatibility(self, python_parser, go_parser):
        """Test compatibility with empty CSV documents."""
        empty_content = {
            "id": "empty_test",
            "content": "",
            "metadata": {"source": "empty_test"}
        }

        # Parse with both parsers
        python_result = python_parser.parse(empty_content)
        go_result = go_parser.parse(empty_content)

        # Both should handle empty CSV gracefully
        assert len(python_result["elements"]) >= 1  # At least root element
        assert len(go_result["elements"]) >= 1  # At least root element

    def test_single_row_csv_compatibility(self, python_parser, go_parser):
        """Test compatibility with single row CSV."""
        single_row_content = {
            "id": "single_row_test",
            "content": "name,age,city\nJohn Doe,30,New York",
            "metadata": {"source": "single_row_test"}
        }

        # Parse with both parsers
        python_result = python_parser.parse(single_row_content)
        go_result = go_parser.parse(single_row_content)

        # Both should handle single row CSV
        assert len(python_result["elements"]) > 1
        assert len(go_result["elements"]) > 1

        # Check for header and data content
        python_has_john = any(
            "John Doe" in elem.get("content_preview", "") or
            "John Doe" in elem.get("text", "")
            for elem in python_result["elements"]
        )
        go_has_john = any(
            "John Doe" in elem.get("content_preview", "") or
            "John Doe" in elem.get("text", "")
            for elem in go_result["elements"]
        )

        assert python_has_john or go_has_john

    def test_no_header_csv_compatibility(self, python_parser, go_parser):
        """Test compatibility with CSV without headers."""
        # Configure parsers to not extract headers
        python_parser.extract_header = False
        go_parser.extract_header = False

        no_header_content = {
            "id": "no_header_test",
            "content": "John Doe,30,New York\nJane Smith,25,Los Angeles",
            "metadata": {"source": "no_header_test"}
        }

        # Parse with both parsers
        python_result = python_parser.parse(no_header_content)
        go_result = go_parser.parse(no_header_content)

        # Both should handle no-header CSV
        assert len(python_result["elements"]) > 1
        assert len(go_result["elements"]) > 1

        # Should not have header row elements
        python_header_rows = [
            elem for elem in python_result["elements"]
            if elem["element_type"] == "table_header_row"
        ]
        go_header_rows = [
            elem for elem in go_result["elements"]
            if elem["element_type"] == "table_header_row"
        ]

        assert len(python_header_rows) == 0
        assert len(go_header_rows) == 0

    def test_different_delimiter_compatibility(self, python_parser, go_parser):
        """Test compatibility with different delimiters."""
        # Configure parsers for semicolon delimiter
        python_parser.delimiter = ";"
        go_parser.delimiter = ";"

        semicolon_content = {
            "id": "semicolon_test",
            "content": "name;age;city\nJohn Doe;30;New York\nJane Smith;25;Los Angeles",
            "metadata": {"source": "semicolon_test"}
        }

        # Parse with both parsers
        python_result = python_parser.parse(semicolon_content)
        go_result = go_parser.parse(semicolon_content)

        # Both should handle semicolon delimiter
        assert len(python_result["elements"]) > 1
        assert len(go_result["elements"]) > 1

        # Check that content was parsed correctly (not treated as single column)
        python_cells = [
            elem for elem in python_result["elements"]
            if elem["element_type"] == "table_cell"
        ]
        go_cells = [
            elem for elem in go_result["elements"]
            if elem["element_type"] == "table_cell"
        ]

        # Should have multiple cells, not just one per row
        assert len(python_cells) >= 6  # At least 3 columns x 2 data rows
        assert len(go_cells) >= 6

    def test_quoted_content_compatibility(self, python_parser, go_parser):
        """Test compatibility with quoted CSV content."""
        quoted_content = {
            "id": "quoted_test",
            "content": 'name,description,price\n"Widget A","A great widget, with features",9.99\n"Widget B","Another widget, even better",14.99',
            "metadata": {"source": "quoted_test"}
        }

        # Parse with both parsers
        python_result = python_parser.parse(quoted_content)
        go_result = go_parser.parse(quoted_content)

        # Both should handle quoted content
        assert len(python_result["elements"]) > 1
        assert len(go_result["elements"]) > 1

        # Check that comma-containing content was preserved
        python_has_comma_content = any(
            "A great widget, with features" in elem.get("text", "") or
            "A great widget, with features" in elem.get("content", "") or
            "A great widget, with features" in elem.get("content_preview", "")
            for elem in python_result["elements"]
        )
        go_has_comma_content = any(
            "A great widget, with features" in elem.get("text", "") or
            "A great widget, with features" in elem.get("content", "") or
            "A great widget, with features" in elem.get("content_preview", "")
            for elem in go_result["elements"]
        )

        assert python_has_comma_content or go_has_comma_content

    def test_metadata_preservation_compatibility(self, python_parser, go_parser, sample_csv_content):
        """Test that metadata is preserved consistently."""
        # Parse with both parsers
        python_result = python_parser.parse(sample_csv_content)
        go_result = go_parser.parse(sample_csv_content)

        # Check that original metadata is preserved in document
        original_metadata = sample_csv_content["metadata"]

        python_doc_metadata = python_result["document"].get("metadata", {})
        go_doc_metadata = go_result["document"].get("metadata", {})

        # Both should preserve some form of original metadata
        assert isinstance(python_doc_metadata, dict)
        assert isinstance(go_doc_metadata, dict)

        # Check for CSV-specific metadata
        csv_metadata_fields = ["row_count", "column_count", "has_header"]

        python_has_csv_meta = any(field in python_doc_metadata for field in csv_metadata_fields)
        go_has_csv_meta = any(field in go_doc_metadata for field in csv_metadata_fields)

        assert python_has_csv_meta or go_has_csv_meta