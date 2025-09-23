"""
Test script to verify that the enhanced _add_selectors method generates unique selectors.
"""

import sys
import os
sys.path.insert(0, os.path.join(os.path.dirname(__file__), 'src'))

from go_doc_go.document_parser.html import HtmlParser
from bs4 import BeautifulSoup

def test_selector_uniqueness():
    """Test that selectors are unique for different elements."""

    # Create a complex HTML structure with multiple similar elements
    html_content = """
    <!DOCTYPE html>
    <html>
    <body>
        <div class="container">
            <table>
                <tr><td>Cell 1-1</td><td>Cell 1-2</td></tr>
                <tr><td>Cell 2-1</td><td>Cell 2-2</td></tr>
                <tr><td>Cell 3-1</td><td>Cell 3-2</td></tr>
            </table>

            <table class="data-table">
                <tr><td>Data 1-1</td><td>Data 1-2</td></tr>
                <tr><td>Data 2-1</td><td>Data 2-2</td></tr>
            </table>

            <div>
                <p>Paragraph 1</p>
                <p>Paragraph 2</p>
                <p>Paragraph 3</p>
            </div>

            <ul>
                <li>Item 1</li>
                <li>Item 2</li>
                <li>Item 3</li>
            </ul>
        </div>

        <div id="unique-div">
            <table>
                <tr><td>Unique Cell 1</td></tr>
                <tr><td>Unique Cell 2</td></tr>
            </table>
        </div>
    </body>
    </html>
    """

    parser = HtmlParser()
    soup = BeautifulSoup(html_content, 'html.parser')

    # Add selectors to all elements
    parser._add_selectors(soup)

    # Collect all selectors
    selectors = []

    def collect_selectors(element):
        if hasattr(element, 'name') and element.name:
            selector = element.get('_selector')
            if selector:
                selectors.append((selector, element.name, element.get_text(strip=True)[:50]))
            for child in element.children:
                collect_selectors(child)

    collect_selectors(soup)

    print("Generated Selectors:")
    print("=" * 80)

    # Group by element type for better readability
    from collections import defaultdict
    by_type = defaultdict(list)

    for selector, tag, text in selectors:
        by_type[tag].append((selector, text))

    for tag in sorted(by_type.keys()):
        print(f"\n{tag.upper()} elements:")
        for selector, text in by_type[tag]:
            print(f"  {selector}")
            if text:
                print(f"    Text: {text}")

    # Check for uniqueness
    print("\n" + "=" * 80)
    print("Uniqueness Check:")

    selector_list = [s[0] for s in selectors]
    unique_selectors = set(selector_list)

    if len(selector_list) == len(unique_selectors):
        print(f"✓ All {len(selector_list)} selectors are unique!")
    else:
        print(f"✗ Found duplicates: {len(selector_list)} total, {len(unique_selectors)} unique")

        # Find and display duplicates
        from collections import Counter
        counter = Counter(selector_list)
        duplicates = {k: v for k, v in counter.items() if v > 1}

        if duplicates:
            print("\nDuplicate selectors:")
            for selector, count in duplicates.items():
                print(f"  {selector}: appears {count} times")

    # Test specific problematic selectors
    print("\n" + "=" * 80)
    print("Testing Specific Selectors:")

    # Test that table rows have unique selectors
    table_rows = soup.find_all('tr')
    tr_selectors = [tr.get('_selector', '') for tr in table_rows]

    print(f"\nTable row selectors ({len(table_rows)} rows):")
    for i, selector in enumerate(tr_selectors):
        row_text = table_rows[i].get_text(strip=True)[:50]
        print(f"  Row {i+1}: {selector}")
        print(f"    Content: {row_text}")

    # Verify each selector returns exactly one element
    print("\n" + "=" * 80)
    print("Selector Resolution Test:")

    errors = []
    for selector, tag, text in selectors[:10]:  # Test first 10 for brevity
        try:
            matches = soup.select(selector)
            if len(matches) == 0:
                errors.append(f"Selector '{selector}' returned 0 matches")
            elif len(matches) > 1:
                errors.append(f"Selector '{selector}' returned {len(matches)} matches (expected 1)")
            else:
                print(f"✓ {selector}: OK")
        except Exception as e:
            errors.append(f"Selector '{selector}' failed: {str(e)}")

    if errors:
        print("\nErrors found:")
        for error in errors:
            print(f"  ✗ {error}")
    else:
        print("\n✓ All tested selectors resolve to exactly one element!")

if __name__ == "__main__":
    test_selector_uniqueness()