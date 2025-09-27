#!/usr/bin/env python3
"""
Test script for the Go XLSX parser integration.
"""

import json
import os
import sys
from pathlib import Path

# Add src directory to path
sys.path.insert(0, str(Path(__file__).parent / "src"))

from go_doc_go.document_parser.factory import create_parser


def test_xlsx_parser():
    """Test the XLSX parser with Go implementation."""

    # Enable Go modules
    os.environ["USE_GO_MODULES"] = "true"

    # Look for a test XLSX file
    test_files = [
        Path("tests/assets/test_sample.xlsx"),
        Path("tests/assets/departments.xlsx"),
        Path("tests/assets/sample.xlsx"),
        Path("tests/assets/test.xlsx"),
        Path("tests/assets/data.xlsx"),
        Path("tests/assets/workbook.xlsx"),
    ]

    test_file = None
    for f in test_files:
        if f.exists():
            test_file = f
            break

    if not test_file:
        print("No test XLSX file found. Creating a simple test...")
        # Test with minimal configuration
        parser = create_parser("xlsx")
        print(f"Parser type: {parser.__class__.__name__}")

        # Test with dummy content - this will fail but shows the parser is created
        try:
            content = {
                "id": "test_xlsx",
                "content": "dummy_path.xlsx",  # This will fail but shows the parser is created
                "metadata": {"test": True}
            }

            result = parser.parse(content)
            print(f"Parse result keys: {result.keys()}")
            print(f"Document: {result.get('document', {})}")
            print(f"Elements count: {len(result.get('elements', []))}")
            print(f"Relationships count: {len(result.get('relationships', []))}")
        except Exception as e:
            print(f"Expected error (no real file): {e}")
            print("Parser is correctly instantiated and available")
        return

    print(f"Testing with file: {test_file}")

    # Create parser
    parser = create_parser("xlsx", {
        "max_rows": 100,
        "max_cols": 20,
        "detect_tables": True,
        "extract_comments": True,
        "extract_formulas": True,
        "extract_links": True,
    })

    print(f"Parser type: {parser.__class__.__name__}")

    # Parse the XLSX (Go parser expects file path in content field)
    content = {
        "id": test_file.stem,
        "content": str(test_file),
        "metadata": {
            "filename": test_file.name,
            "source": "test"
        }
    }

    try:
        result = parser.parse(content)

        # Display results
        print("\n=== Parse Results ===")
        print(f"Document ID: {result['document']['id']}")
        print(f"Document Type: {result['document']['doc_type']}")

        if result['document'].get('title'):
            print(f"Title: {result['document']['title']}")

        if result['document'].get('metadata'):
            print(f"Metadata: {json.dumps(result['document']['metadata'], indent=2)}")

        print(f"\nElements: {len(result['elements'])}")
        if result['elements']:
            print("First 5 elements:")
            for i, elem in enumerate(result['elements'][:5]):
                location_info = ""
                if elem.get('content_location'):
                    loc = elem['content_location']
                    if 'sheet_name' in loc:
                        location_info = f" (Sheet: {loc['sheet_name']}"
                        if 'cell_range' in loc:
                            location_info += f", Range: {loc['cell_range']}"
                        location_info += ")"

                preview = elem.get('content_preview', '')[:60]
                print(f"  {i+1}. [{elem['element_type']}] {preview}...{location_info}")

        print(f"\nRelationships: {len(result['relationships'])}")

        if result.get('links'):
            print(f"Links found: {len(result['links'])}")
            for link in result['links'][:3]:
                print(f"  - [{link['link_type']}] {link['link_target']}")

        # Show table information
        tables = [e for e in result['elements'] if e['element_type'] == 'table']
        if tables:
            print(f"\nTables detected: {len(tables)}")
            for i, table in enumerate(tables[:3]):
                print(f"  Table {i+1}: {table['content_preview']}")
                if table.get('content_location'):
                    loc = table['content_location']
                    print(f"    Location: Sheet '{loc.get('sheet_name')}', Range: {loc.get('cell_range')}")

    except Exception as e:
        print(f"Error during parsing: {e}")
        import traceback
        traceback.print_exc()


def test_python_vs_go_performance():
    """Compare Python vs Go XLSX parser performance."""

    test_files = [
        Path("tests/assets/test_sample.xlsx"),
        Path("tests/assets/departments.xlsx"),
        Path("tests/assets/sample.xlsx"),
        Path("tests/assets/test.xlsx"),
    ]

    test_file = None
    for f in test_files:
        if f.exists():
            test_file = f
            break

    if not test_file:
        print("No test file found for performance comparison")
        return

    print(f"\n=== Performance Comparison with {test_file.name} ===")

    # Content for Go parser (expects file path in content)
    go_content = {
        "id": test_file.stem,
        "content": str(test_file),
        "metadata": {"filename": test_file.name}
    }

    # Content for Python parser (expects binary_path)
    py_content = {
        "id": test_file.stem,
        "binary_path": str(test_file),
        "metadata": {"filename": test_file.name}
    }

    # Test Go parser
    print("\nTesting Go XLSX parser...")
    os.environ["USE_GO_MODULES"] = "true"
    import time

    try:
        go_parser = create_parser("xlsx")
        print(f"Go Parser: {go_parser.__class__.__name__}")

        start = time.time()
        go_result = go_parser.parse(go_content)
        go_time = time.time() - start

        print(f"Go parsing: {go_time*1000:.2f}ms")
        print(f"Elements parsed: {len(go_result['elements'])}")
        print(f"Relationships: {len(go_result['relationships'])}")
        print(f"Links found: {len(go_result.get('links', []))}")

    except Exception as e:
        print(f"Go parser error: {e}")

    # Test Python parser
    print("\nTesting Python XLSX parser...")
    os.environ["USE_GO_MODULES"] = "false"

    try:
        py_parser = create_parser("xlsx")
        print(f"Python Parser: {py_parser.__class__.__name__}")

        start = time.time()
        py_result = py_parser.parse(py_content)
        py_time = time.time() - start

        print(f"Python parsing: {py_time*1000:.2f}ms")
        print(f"Elements parsed: {len(py_result['elements'])}")
        print(f"Relationships: {len(py_result['relationships'])}")
        print(f"Links found: {len(py_result.get('links', []))}")

        if 'go_time' in locals():
            speedup = py_time / go_time
            print(f"\nSpeedup: {speedup:.2f}x (Go is {speedup:.2f}x faster)")

    except Exception as e:
        print(f"Python parser error: {e}")


if __name__ == "__main__":
    test_xlsx_parser()
    test_python_vs_go_performance()