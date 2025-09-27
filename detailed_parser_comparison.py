#!/usr/bin/env python3
"""
Detailed comparison of Go vs Python XLSX parser outputs.
"""

import json
import os
import sys
from pathlib import Path
from collections import Counter

# Add src directory to path
sys.path.insert(0, str(Path(__file__).parent / "src"))

from go_doc_go.document_parser.factory import create_parser


def compare_parsers():
    """Compare Go and Python parser outputs in detail."""

    test_file = Path("tests/assets/test_sample.xlsx")
    if not test_file.exists():
        print("Test file not found")
        return

    print(f"Detailed Parser Comparison: {test_file.name}")
    print("=" * 60)

    # Test Go parser
    print("\n1. GO PARSER ANALYSIS")
    print("-" * 30)
    os.environ["USE_GO_MODULES"] = "true"

    go_content = {
        "id": test_file.stem,
        "content": str(test_file),
        "metadata": {"filename": test_file.name}
    }

    go_parser = create_parser("xlsx")
    go_result = go_parser.parse(go_content)

    print(f"Parser: {go_parser.__class__.__name__}")
    print(f"Elements: {len(go_result['elements'])}")
    print(f"Relationships: {len(go_result['relationships'])}")

    # Analyze element types
    go_element_types = Counter(elem['element_type'] for elem in go_result['elements'])
    print("\nElement types (Go):")
    for elem_type, count in sorted(go_element_types.items()):
        print(f"  {elem_type}: {count}")

    # Test Python parser
    print("\n2. PYTHON PARSER ANALYSIS")
    print("-" * 30)
    os.environ["USE_GO_MODULES"] = "false"

    py_content = {
        "id": test_file.stem,
        "binary_path": str(test_file),
        "metadata": {"filename": test_file.name}
    }

    py_parser = create_parser("xlsx")
    py_result = py_parser.parse(py_content)

    print(f"Parser: {py_parser.__class__.__name__}")
    print(f"Elements: {len(py_result['elements'])}")
    print(f"Relationships: {len(py_result['relationships'])}")

    # Analyze element types
    py_element_types = Counter(elem['element_type'] for elem in py_result['elements'])
    print("\nElement types (Python):")
    for elem_type, count in sorted(py_element_types.items()):
        print(f"  {elem_type}: {count}")

    # Compare element types
    print("\n3. ELEMENT TYPE COMPARISON")
    print("-" * 30)
    all_types = set(go_element_types.keys()) | set(py_element_types.keys())

    print(f"{'Element Type':<20} {'Go':<8} {'Python':<8} {'Diff':<8}")
    print("-" * 50)
    for elem_type in sorted(all_types):
        go_count = go_element_types.get(elem_type, 0)
        py_count = py_element_types.get(elem_type, 0)
        diff = py_count - go_count
        diff_str = f"{diff:+d}" if diff != 0 else "0"
        print(f"{elem_type:<20} {go_count:<8} {py_count:<8} {diff_str:<8}")

    # Show sample elements from each parser
    print("\n4. SAMPLE ELEMENTS")
    print("-" * 30)

    print("\nGo Parser Sample Elements:")
    for i, elem in enumerate(go_result['elements'][:10]):
        content = elem.get('content_preview', '')[:50]
        location = ""
        if elem.get('content_location'):
            try:
                loc_data = json.loads(elem['content_location']) if isinstance(elem['content_location'], str) else elem['content_location']
                if 'sheet_name' in loc_data:
                    location = f" (Sheet: {loc_data['sheet_name']})"
            except:
                pass
        print(f"  {i+1:2d}. [{elem['element_type']}] {content}...{location}")

    print("\nPython Parser Sample Elements:")
    for i, elem in enumerate(py_result['elements'][:10]):
        content = elem.get('content_preview', '')[:50]
        location = ""
        if elem.get('content_location'):
            try:
                loc_data = json.loads(elem['content_location']) if isinstance(elem['content_location'], str) else elem['content_location']
                if 'sheet_name' in loc_data:
                    location = f" (Sheet: {loc_data['sheet_name']})"
                elif 'sheet' in loc_data:
                    location = f" (Sheet: {loc_data['sheet']})"
            except:
                pass
        print(f"  {i+1:2d}. [{elem['element_type']}] {content}...{location}")

    # Check for unique elements in each parser
    print("\n5. UNIQUE ELEMENT ANALYSIS")
    print("-" * 30)

    go_only = set(go_element_types.keys()) - set(py_element_types.keys())
    py_only = set(py_element_types.keys()) - set(go_element_types.keys())

    if go_only:
        print(f"Elements only in Go parser: {', '.join(sorted(go_only))}")
    if py_only:
        print(f"Elements only in Python parser: {', '.join(sorted(py_only))}")

    if not go_only and not py_only:
        print("Both parsers create the same element types (just different counts)")


if __name__ == "__main__":
    compare_parsers()