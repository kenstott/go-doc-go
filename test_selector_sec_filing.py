"""
Test enhanced selectors with actual SEC filing HTML structure.
"""

import sys
import os
sys.path.insert(0, os.path.join(os.path.dirname(__file__), 'src'))

from go_doc_go.document_parser.html import HtmlParser
from pathlib import Path
from collections import Counter

def test_sec_filing_selectors():
    """Test selectors on actual SEC filing."""

    # Check if the test file exists
    test_file = Path("/Users/kennethstott/PycharmProjects/doculyzer/tests/assets/aapl-20201226.htm")

    if not test_file.exists():
        print(f"Test file not found: {test_file}")
        print("Looking for other HTML test files...")

        # Look for alternative test files
        test_files = list(Path("/Users/kennethstott/PycharmProjects/doculyzer").glob("**/*.htm"))
        test_files.extend(list(Path("/Users/kennethstott/PycharmProjects/doculyzer").glob("**/*.html")))

        if test_files:
            print(f"Found {len(test_files)} HTML files. Using the first one.")
            test_file = test_files[0]
        else:
            print("No HTML files found. Creating a test case.")
            # Create test HTML with complex nested tables
            html_content = """
            <!DOCTYPE html>
            <html>
            <body>
                <table>
                    <tr>
                        <td>Parent Table Row 1</td>
                        <td>
                            <table>
                                <tr><td>Nested Table R1C1</td><td>Nested Table R1C2</td></tr>
                                <tr><td>Nested Table R2C1</td><td>Nested Table R2C2</td></tr>
                            </table>
                        </td>
                    </tr>
                    <tr>
                        <td>Parent Table Row 2</td>
                        <td>More content</td>
                    </tr>
                </table>

                <table>
                    <tr><td>Second Table R1</td></tr>
                    <tr><td>Second Table R2</td></tr>
                    <tr><td>Second Table R3</td></tr>
                </table>
            </body>
            </html>
            """
            test_file = None
    else:
        with open(test_file, 'r', encoding='utf-8') as f:
            html_content = f.read()

    print(f"Testing with: {test_file or 'Generated test HTML'}")
    print("=" * 80)

    # Parse the HTML
    parser = HtmlParser()

    # Parse document
    result = parser.parse({
        "id": "test_doc",
        "content": html_content,
        "metadata": {
            "source": str(test_file) if test_file else "test_html"
        }
    })

    # Analyze selectors from the elements
    print("\nAnalyzing generated selectors...")
    print("-" * 40)

    # Group elements by type
    elements_by_type = {}
    for element in result["elements"]:
        elem_type = element["element_type"]
        if elem_type not in elements_by_type:
            elements_by_type[elem_type] = []
        elements_by_type[elem_type].append(element)

    # Check table-related elements for unique selectors
    table_types = ["table", "table_row", "table_cell"]

    for elem_type in table_types:
        if elem_type in elements_by_type:
            elements = elements_by_type[elem_type]
            print(f"\n{elem_type.upper()} elements: {len(elements)}")

            # Extract selectors
            selectors = []
            for elem in elements[:10]:  # Show first 10 for brevity
                location = elem.get("content_location", {})
                if isinstance(location, dict):
                    selector = location.get("selector", "No selector")
                elif isinstance(location, str):
                    # Location might be a JSON string
                    import json
                    try:
                        location_dict = json.loads(location)
                        selector = location_dict.get("selector", "No selector")
                    except:
                        selector = "No selector"
                else:
                    selector = "No selector"
                selectors.append(selector)
                print(f"  {selector}")
                if elem.get("content_preview"):
                    print(f"    Preview: {elem['content_preview'][:60]}...")

            # Check for uniqueness
            all_selectors = []
            for elem in elements:
                location = elem.get("content_location", {})
                if isinstance(location, dict):
                    selector = location.get("selector", "")
                elif isinstance(location, str):
                    try:
                        location_dict = json.loads(location)
                        selector = location_dict.get("selector", "")
                    except:
                        selector = ""
                else:
                    selector = ""
                all_selectors.append(selector)
            selector_counts = Counter(all_selectors)
            duplicates = {k: v for k, v in selector_counts.items() if v > 1 and k}

            if duplicates:
                print(f"\n  ⚠️  Found duplicate selectors in {elem_type}:")
                for selector, count in list(duplicates.items())[:5]:
                    print(f"    '{selector}' appears {count} times")
            else:
                print(f"  ✓ All {len(all_selectors)} {elem_type} selectors are unique!")

    # Test specific problematic patterns
    print("\n" + "=" * 80)
    print("Testing specific patterns:")

    # Find all table rows and check their selectors
    table_rows = [e for e in result["elements"] if e["element_type"] == "table_row"]

    if table_rows:
        print(f"\nFound {len(table_rows)} table rows")

        # Group by parent table
        rows_by_table = {}
        for row in table_rows:
            location = row.get("content_location", {})
            if isinstance(location, dict):
                selector = location.get("selector", "")
            elif isinstance(location, str):
                try:
                    import json
                    location_dict = json.loads(location)
                    selector = location_dict.get("selector", "")
                except:
                    selector = ""
            else:
                selector = ""

            # Extract table part of selector
            if " > tr" in selector:
                table_part = selector.split(" > tr")[0]
                if table_part not in rows_by_table:
                    rows_by_table[table_part] = []
                rows_by_table[table_part].append(selector)

        print(f"Rows are distributed across {len(rows_by_table)} tables")

        for table_selector, row_selectors in list(rows_by_table.items())[:3]:  # Show first 3 tables
            print(f"\n  Table: {table_selector}")
            print(f"    Has {len(row_selectors)} rows")
            for i, row_sel in enumerate(row_selectors[:3]):  # Show first 3 rows
                print(f"      Row {i+1}: {row_sel}")

    print("\n" + "=" * 80)
    print("Summary:")
    print(f"  Total elements: {len(result['elements'])}")
    print(f"  Element types: {len(elements_by_type)}")

    # Calculate overall selector uniqueness
    all_selectors = []
    for elem in result["elements"]:
        location = elem.get("content_location", {})
        if isinstance(location, dict):
            selector = location.get("selector")
        elif isinstance(location, str):
            try:
                import json
                location_dict = json.loads(location)
                selector = location_dict.get("selector")
            except:
                selector = None
        else:
            selector = None

        if selector:
            all_selectors.append(selector)

    unique_selectors = set(all_selectors)
    if len(all_selectors) == len(unique_selectors):
        print(f"  ✓ All {len(all_selectors)} selectors are unique!")
    else:
        print(f"  ⚠️  {len(all_selectors)} total selectors, {len(unique_selectors)} unique")
        duplicates = Counter(all_selectors)
        most_common = duplicates.most_common(5)
        print("\n  Most duplicated selectors:")
        for selector, count in most_common:
            if count > 1:
                print(f"    '{selector}': {count} times")

if __name__ == "__main__":
    test_sec_filing_selectors()