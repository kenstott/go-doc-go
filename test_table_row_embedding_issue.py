#!/usr/bin/env python3
"""
Test to demonstrate that table_row elements with the target text
are found but don't get embeddings due to being filtered as non-leaf elements.
"""

import sys
import os
import json
sys.path.insert(0, os.path.join(os.path.dirname(__file__), 'src'))

from go_doc_go.document_parser.html import HtmlParser
from go_doc_go.embeddings.contextual_embedding import ContextualEmbeddingGenerator
from go_doc_go.embeddings.fastembed import FastEmbedGenerator
from go_doc_go.config import Config

# Test HTML with a simple table containing the target text
TEST_HTML = """<!DOCTYPE html>
<html>
<head><title>Test Document</title></head>
<body>
<table>
    <tr>
        <td>Item 2.</td>
        <td>Unregistered Sales of Equity Securities and Use of Proceeds</td>
        <td>30</td>
    </tr>
    <tr>
        <td>Item 3.</td>
        <td>Other Information</td>
        <td>31</td>
    </tr>
</table>
</body>
</html>"""

def test_table_row_embedding():
    """Test that table_row elements get embeddings."""

    print("=" * 80)
    print("TEST: Table Row Embedding Issue")
    print("=" * 80)
    print()

    # Parse the HTML document
    print("1. Parsing HTML document...")
    parser = HtmlParser()
    result = parser.parse({
        'content': TEST_HTML,
        'id': 'test.html'
    })

    elements = result['elements']
    print(f"   Parsed {len(elements)} elements")

    # Find table elements
    tables = [e for e in elements if e.get('element_type') == 'table']
    table_rows = [e for e in elements if e.get('element_type') == 'table_row']
    table_cells = [e for e in elements if e.get('element_type') == 'table_cell']

    print(f"   - Tables: {len(tables)}")
    print(f"   - Table rows: {len(table_rows)}")
    print(f"   - Table cells: {len(table_cells)}")

    # Find the table_row with target text
    print("\n2. Finding table_row with target text...")
    target_row = None
    for row in table_rows:
        if "Unregistered Sales" in row.get('content_preview', ''):
            target_row = row
            print(f"   ✓ Found: {row['element_id']}")
            print(f"     Content: {row['content_preview']}")
            break

    if not target_row:
        print("   ✗ No table_row found with target text")
        return False

    # Check if the row has children
    row_id = target_row['element_id']
    row_children = [e for e in elements if e.get('parent_id') == row_id]
    print(f"\n3. Checking table_row structure...")
    print(f"   Row has {len(row_children)} children (cells)")
    for child in row_children[:3]:
        print(f"   - {child['element_type']}: {child.get('content_preview', '')[:50]}")

    # Generate embeddings
    print("\n4. Generating contextual embeddings...")
    config = Config()
    config.config = {
        'embedding': {
            'type': 'fastembed',
            'model_name': 'BAAI/bge-small-en-v1.5'
        }
    }

    base_generator = FastEmbedGenerator(config, model_name='BAAI/bge-small-en-v1.5')
    contextual_generator = ContextualEmbeddingGenerator(
        config,
        base_generator,
        predecessor_count=2,
        successor_count=2
    )

    embeddings = contextual_generator.generate_from_elements(elements)

    print(f"   Generated {len(embeddings)} embeddings")

    # Check which element types got embeddings
    embedded_types = {}
    for elem_id in embeddings:
        elem = next((e for e in elements if e['element_id'] == elem_id), None)
        if elem:
            elem_type = elem.get('element_type')
            embedded_types[elem_type] = embedded_types.get(elem_type, 0) + 1

    print(f"\n5. Elements with embeddings by type:")
    for elem_type, count in embedded_types.items():
        print(f"   - {elem_type}: {count}")

    # Check if our target row got an embedding
    print(f"\n6. Checking target table_row embedding...")
    if target_row['element_id'] in embeddings:
        print("   ✓ SUCCESS: Table row has embedding!")
        embedding_text = embeddings[target_row['element_id']].get('embedding_text', '')
        print(f"\n   Embedding text (first 500 chars):")
        print("   " + "-" * 40)
        print(f"   {embedding_text[:500]}")
        print("   " + "-" * 40)
        return True
    else:
        print("   ✗ FAILURE: Table row has NO embedding")
        print("\n   EXPLANATION:")
        print("   The table_row is being filtered out as a 'non-leaf' element")
        print("   because it has children (table_cell elements).")
        print("   This is a bug in contextual_embedding.py lines 661-664")
        print("   where non-leaf elements are skipped.")
        return False

if __name__ == "__main__":
    success = test_table_row_embedding()
    print()
    print("=" * 80)
    if success:
        print("✅ TEST PASSED")
    else:
        print("❌ TEST FAILED - Table rows with children don't get embeddings")
    print("=" * 80)
    sys.exit(0 if success else 1)