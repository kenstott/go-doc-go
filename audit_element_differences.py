#!/usr/bin/env python3
"""
Detailed audit of element differences between Go and Python XLSX parsers.
"""

import json
import os
import sys
from pathlib import Path
from collections import defaultdict

# Add src directory to path
sys.path.insert(0, str(Path(__file__).parent / "src"))

from go_doc_go.document_parser.factory import create_parser


def audit_elements():
    """Audit all elements created by both parsers in detail."""

    test_file = Path("tests/assets/test_sample.xlsx")
    if not test_file.exists():
        print("Test file not found")
        return

    print(f"DETAILED ELEMENT AUDIT: {test_file.name}")
    print("=" * 80)

    # Test Go parser
    print("\n🔧 GO PARSER ELEMENTS")
    print("=" * 40)
    os.environ["USE_GO_MODULES"] = "true"

    go_content = {
        "id": test_file.stem,
        "content": str(test_file),
        "metadata": {"filename": test_file.name}
    }

    go_parser = create_parser("xlsx")
    go_result = go_parser.parse(go_content)

    print(f"Total Elements: {len(go_result['elements'])}")
    print(f"Total Relationships: {len(go_result['relationships'])}")

    print("\nAll Go Parser Elements:")
    go_elements_by_type = defaultdict(list)
    for i, elem in enumerate(go_result['elements']):
        elem_type = elem['element_type']
        go_elements_by_type[elem_type].append(elem)

        # Get location info
        location = ""
        if elem.get('content_location'):
            loc_data = elem['content_location']
            if isinstance(loc_data, dict):
                if 'sheet_name' in loc_data:
                    location = f" [Sheet: {loc_data['sheet_name']}]"
                if 'cell' in loc_data:
                    location += f" [Cell: {loc_data['cell']}]"
            elif isinstance(loc_data, str):
                try:
                    loc_data = json.loads(loc_data)
                    if 'sheet_name' in loc_data:
                        location = f" [Sheet: {loc_data['sheet_name']}]"
                except:
                    pass

        content = elem.get('content_preview', '')[:60]
        print(f"  {i+1:2d}. [{elem_type:15}] {content}...{location}")

    # Test Python parser
    print(f"\n🐍 PYTHON PARSER ELEMENTS")
    print("=" * 40)
    os.environ["USE_GO_MODULES"] = "false"

    py_content = {
        "id": test_file.stem,
        "binary_path": str(test_file),
        "metadata": {"filename": test_file.name}
    }

    py_parser = create_parser("xlsx")
    py_result = py_parser.parse(py_content)

    print(f"Total Elements: {len(py_result['elements'])}")
    print(f"Total Relationships: {len(py_result['relationships'])}")

    print("\nAll Python Parser Elements:")
    py_elements_by_type = defaultdict(list)
    for i, elem in enumerate(py_result['elements']):
        elem_type = elem['element_type']
        py_elements_by_type[elem_type].append(elem)

        # Get location info
        location = ""
        if elem.get('content_location'):
            loc_data = elem['content_location']
            if isinstance(loc_data, str):
                try:
                    loc_data = json.loads(loc_data)
                except:
                    pass

            if isinstance(loc_data, dict):
                if 'sheet_name' in loc_data:
                    location = f" [Sheet: {loc_data['sheet_name']}]"
                elif 'sheet' in loc_data:
                    location = f" [Sheet: {loc_data['sheet']}]"
                if 'cell_ref' in loc_data:
                    location += f" [Cell: {loc_data['cell_ref']}]"

        content = elem.get('content_preview', '')[:60]
        print(f"  {i+1:2d}. [{elem_type:15}] {content}...{location}")

    # Compare by element type
    print(f"\n📊 DETAILED TYPE-BY-TYPE COMPARISON")
    print("=" * 60)

    all_types = set(go_elements_by_type.keys()) | set(py_elements_by_type.keys())

    for elem_type in sorted(all_types):
        go_count = len(go_elements_by_type.get(elem_type, []))
        py_count = len(py_elements_by_type.get(elem_type, []))
        diff = py_count - go_count

        print(f"\n{elem_type.upper()} ELEMENTS:")
        print(f"  Go: {go_count}, Python: {py_count}, Diff: {diff:+d}")

        if diff != 0:
            print("  Go elements:")
            for elem in go_elements_by_type.get(elem_type, []):
                content = elem.get('content_preview', '')[:40]
                print(f"    - {content}")

            print("  Python elements:")
            for elem in py_elements_by_type.get(elem_type, []):
                content = elem.get('content_preview', '')[:40]
                print(f"    - {content}")

    # Check for structural differences
    print(f"\n🔍 STRUCTURAL ANALYSIS")
    print("=" * 40)

    # Count elements by sheet
    go_by_sheet = defaultdict(int)
    py_by_sheet = defaultdict(int)

    for elem in go_result['elements']:
        if elem.get('content_location'):
            loc_data = elem['content_location']
            if isinstance(loc_data, dict) and 'sheet_name' in loc_data:
                go_by_sheet[loc_data['sheet_name']] += 1

    for elem in py_result['elements']:
        if elem.get('content_location'):
            loc_data = elem['content_location']
            if isinstance(loc_data, str):
                try:
                    loc_data = json.loads(loc_data)
                except:
                    continue
            if isinstance(loc_data, dict):
                sheet = loc_data.get('sheet_name') or loc_data.get('sheet')
                if sheet:
                    py_by_sheet[sheet] += 1

    print("Elements by sheet:")
    all_sheets = set(go_by_sheet.keys()) | set(py_by_sheet.keys())
    for sheet in sorted(all_sheets):
        go_count = go_by_sheet.get(sheet, 0)
        py_count = py_by_sheet.get(sheet, 0)
        diff = py_count - go_count
        print(f"  {sheet}: Go={go_count}, Python={py_count}, Diff={diff:+d}")


if __name__ == "__main__":
    audit_elements()