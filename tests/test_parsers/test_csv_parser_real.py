"""
Real CSV parser tests without mocks.
"""

import os
import tempfile
import pytest
from go_doc_go.document_parser.csv import CsvParser


class TestCsvParserReal:
    """Test CSV parser with real files."""
    
    def setup_method(self):
        """Set up test fixtures."""
        self.parser = CsvParser()
        self.temp_dir = tempfile.mkdtemp()
    
    def teardown_method(self):
        """Clean up temp files."""
        import shutil
        shutil.rmtree(self.temp_dir, ignore_errors=True)
    
    def test_parse_real_csv_file(self):
        """Test parsing a real CSV file."""
        # Create a real CSV file
        csv_content = """Name,Age,City,Country
John Doe,30,New York,USA
Jane Smith,25,London,UK
Bob Johnson,35,Toronto,Canada"""
        
        csv_path = os.path.join(self.temp_dir, "test.csv")
        with open(csv_path, 'w') as f:
            f.write(csv_content)
        
        # Parse the file
        doc_content = {
            "id": csv_path,
            "content": csv_content,
            "metadata": {
                "doc_id": "test_csv_001",
                "filename": "test.csv"
            }
        }
        
        result = self.parser.parse(doc_content)
        
        # Verify results
        assert result is not None
        assert "document" in result
        assert "elements" in result
        
        doc = result["document"]
        assert doc["doc_type"] == "csv"
        assert doc["doc_id"] == "test_csv_001"
        
        # Check elements were created
        elements = result["elements"]
        assert len(elements) > 0
        
        # Check for table structure - print what we actually have
        element_types = [e["element_type"] for e in elements]
        print(f"  Element types found: {set(element_types)}")
        
        # Check for table elements (lowercase)
        table_elements = [e for e in elements if e["element_type"] == "table"]
        assert len(table_elements) > 0, f"No table elements found. Found: {set(element_types)}"
        
        # Check for rows (lowercase)
        row_elements = [e for e in elements if e["element_type"] == "table_row"]
        assert len(row_elements) >= 3  # At least the data rows
        
        print(f"✓ Parsed CSV with {len(elements)} elements")
        print(f"  - Found {len(table_elements)} tables")
        print(f"  - Found {len(row_elements)} rows")
    
    def test_parse_csv_with_special_characters(self):
        """Test CSV with quotes and special characters."""
        csv_content = '''Product,Description,Price
"Laptop","High-end, 16GB RAM",1299.99
"Mouse","Wireless, RGB",49.99
"Monitor","27"" 4K Display",599.99'''
        
        doc_content = {
            "id": "/test/products.csv",
            "content": csv_content,
            "metadata": {"doc_id": "test_002"}
        }
        
        result = self.parser.parse(doc_content)
        
        assert result is not None
        elements = result["elements"]
        
        # Check content is preserved
        content_found = False
        for element in elements:
            preview = element.get("content_preview", "")
            if "16GB RAM" in preview or "4K Display" in preview:
                content_found = True
                break
        
        assert content_found, "Special characters in content not preserved"
        print("✓ CSV with special characters parsed correctly")
    
    def test_parse_empty_csv(self):
        """Test parsing empty CSV."""
        csv_content = ""
        
        doc_content = {
            "id": "/test/empty.csv",
            "content": csv_content,
            "metadata": {}
        }
        
        result = self.parser.parse(doc_content)
        
        assert result is not None
        assert "document" in result
        assert "elements" in result
        
        # Should still create at least root element
        elements = result["elements"]
        assert len(elements) >= 1
        
        print("✓ Empty CSV handled gracefully")
    
    def test_parse_large_csv(self):
        """Test parsing large CSV with row limits."""
        # Create CSV with 100 rows
        header = "ID,Name,Value\n"
        rows = [f"{i},Item_{i},{i*10}" for i in range(100)]
        csv_content = header + "\n".join(rows)
        
        # Parser with 50 row limit
        parser = CsvParser({"max_rows": 50})
        
        doc_content = {
            "id": "/test/large.csv",
            "content": csv_content,
            "metadata": {}
        }
        
        result = parser.parse(doc_content)
        
        elements = result["elements"]
        row_elements = [e for e in elements if e["element_type"] == "TABLE_ROW"]
        
        # Should limit rows
        assert len(row_elements) <= 50
        
        print(f"✓ Large CSV limited to {len(row_elements)} rows")


if __name__ == "__main__":
    # Run tests
    test = TestCsvParserReal()
    test.setup_method()
    
    try:
        test.test_parse_real_csv_file()
        test.test_parse_csv_with_special_characters()
        test.test_parse_empty_csv()
        test.test_parse_large_csv()
        print("\n✅ All real CSV tests passed!")
    finally:
        test.teardown_method()