#!/usr/bin/env python3
"""
Debug script to understand what elements are parsed from the iXBRL document.
"""

import sys
import os
from collections import Counter
sys.path.insert(0, os.path.join(os.path.dirname(__file__), 'src'))

from go_doc_go.document_parser.html import HtmlParser

html_file = "/Volumes/T9/govdata-cache/sec/sec-data/0000320193/000032019321000010/aapl-20201226.htm"

if not os.path.exists(html_file):
    print(f"ERROR: Test file not found: {html_file}")
    sys.exit(1)

# Read the HTML content
with open(html_file, 'r', encoding='utf-8', errors='ignore') as f:
    html_content = f.read()

print(f"Read {len(html_content)} bytes from file")

# Parse the HTML document
parser = HtmlParser()
result = parser.parse({
    'content': html_content,
    'id': html_file
})

elements = result['elements']
print(f"Parsed {len(elements)} elements\n")

# Count element types
element_types = Counter(e['element_type'] for e in elements)

print("Element type distribution:")
for elem_type, count in element_types.most_common(20):
    print(f"  {elem_type}: {count}")

print("\n\nSample non-table elements with content:")
non_table_with_content = [
    e for e in elements
    if e.get('element_type') not in ['table_cell', 'table_row', 'table', 'root', 'body']
    and e.get('content_preview')
    and len(e.get('content_preview', '')) > 20
]

for i, elem in enumerate(non_table_with_content[:10]):
    print(f"\n{i}. Type: {elem['element_type']}")
    print(f"   Content: {elem['content_preview'][:100]}...")
    print(f"   Full length: {len(elem.get('content_preview', ''))}")