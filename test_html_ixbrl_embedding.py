#!/usr/bin/env python3
"""
Test for HTML/iXBRL document contextual embedding using actual SEC filing.
Verifies that HTML elements get proper context without raw document content.
"""

import sys
import os
import json
sys.path.insert(0, os.path.join(os.path.dirname(__file__), 'src'))

from go_doc_go.document_parser.html import HtmlParser
from go_doc_go.embeddings.contextual_embedding import ContextualEmbeddingGenerator
from go_doc_go.embeddings.fastembed import FastEmbedGenerator
from go_doc_go.config import Config
from go_doc_go.storage import ElementType


def test_html_ixbrl_embedding():
    """Test HTML/iXBRL document embeddings using actual SEC filing."""

    print("=" * 80)
    print("TEST: HTML/iXBRL Document Contextual Embedding")
    print("=" * 80)
    print()

    # Use the actual HTML file
    html_file = "./tests/assets/aapl-20201226.htm"

    if not os.path.exists(html_file):
        print(f"ERROR: Test file not found: {html_file}")
        return False

    try:
        # Read the HTML content
        print("1. Reading HTML/iXBRL document...")
        with open(html_file, 'r', encoding='utf-8', errors='ignore') as f:
            html_content = f.read()

        print(f"   Read {len(html_content)} bytes from file")

        # Parse the HTML document
        print("\n2. Parsing HTML document...")
        parser = HtmlParser()
        result = parser.parse({
            'content': html_content,
            'id': html_file
        })

        elements = result['elements']
        print(f"   Parsed {len(elements)} elements")

        # Look for the specific text in table_row elements
        print("\n3. Looking for specific element:")
        target_text = "Unregistered Sales of Equity Securities and Use of Proceeds"
        text_node = None

        # Look for table_row elements containing the target text
        print(f"   Searching for table_row containing: '{target_text}'")
        table_rows = [e for e in elements if e.get('element_type') == 'table_row']
        print(f"   Found {len(table_rows)} table_row elements to search")

        for elem in table_rows:
            content = elem.get('content_preview', '')
            # Check if the target text is in this row (exact or partial match)
            if target_text in content or "Unregistered Sales" in content or "Equity Securities" in content:
                text_node = elem
                print(f"   ✓ Found target text in table_row: {elem['element_id']}")
                print(f"      Match type: {'exact' if target_text in content else 'partial'}")
                break

        if not text_node:
            print("ERROR: Could not find a suitable element for testing!")
            # Show some sample elements
            print("\nSample elements found:")
            sample_elements = [e for e in elements if e.get('content_preview')][:5]
            for i, elem in enumerate(sample_elements):
                print(f"   {i} ({elem['element_type']}): {elem.get('content_preview', '')[:60]}...")
            return False

        print(f"\n   Selected element: {text_node['element_id']}")
        print(f"   Type: {text_node['element_type']}")
        print(f"   Content preview: '{text_node['content_preview']}'")
        print(f"   Content length: {len(text_node['content_preview'])} characters")
        print()

        # Generate contextual embeddings
        print("4. Generating contextual embeddings...")
        config = Config()
        config.config = {
            'embedding': {
                'type': 'fastembed',
                'model_name': 'BAAI/bge-small-en-v1.5'
            }
        }

        # Create embedding generator
        base_generator = FastEmbedGenerator(config, model_name='BAAI/bge-small-en-v1.5')

        # ContextualEmbeddingGenerator will create its own resolver internally
        contextual_generator = ContextualEmbeddingGenerator(
            config,
            base_generator,
            predecessor_count=2,
            successor_count=2
        )

        # Generate embeddings
        embeddings = contextual_generator.generate_from_elements(elements)

        # Debug: Show which elements got embeddings
        print(f"\n   Generated embeddings for {len(embeddings)} elements")

        # Check which table_rows got embeddings
        table_rows_with_embeddings = [
            elem_id for elem_id in embeddings
            if any(e['element_id'] == elem_id and e.get('element_type') == 'table_row' for e in elements)
        ]
        print(f"   Table rows with embeddings: {len(table_rows_with_embeddings)}")

        # Show a sample of elements that got embeddings
        print("\n   Sample of elements with embeddings:")
        for i, (elem_id, emb_data) in enumerate(list(embeddings.items())[:5]):
            elem = next((e for e in elements if e['element_id'] == elem_id), None)
            if elem:
                print(f"      - {elem['element_type']}: {elem.get('content_preview', '')[:50]}...")

        # Check the text node's embedding
        if text_node['element_id'] not in embeddings:
            print(f"\nERROR: Selected element (table_row) has no embedding!")
            print(f"   Element ID: {text_node['element_id']}")
            print(f"   Content: {text_node['content_preview'][:100]}")

            # Check if it's being filtered out for some reason
            print("\n   Debugging why this element was skipped:")
            print(f"   - Element type: {text_node.get('element_type')}")
            print(f"   - Content length: {len(text_node.get('content_preview', ''))}")

            # Show nearby table_rows that might have embeddings
            print("\n   Looking for nearby table_rows with embeddings...")
            for elem in elements:
                if elem.get('element_type') == 'table_row' and elem['element_id'] in embeddings:
                    content = elem.get('content_preview', '')
                    if 'Item' in content or 'Securities' in content:
                        print(f"      Found: {content[:80]}...")
                        break

            return False

        embedding_data = embeddings[text_node['element_id']]
        embedding_text = embedding_data.get('embedding_text', '')

        print("\n5. Analyzing contextual embedding:")
        print("=" * 60)
        print("FULL CONTEXTUAL EMBEDDING TEXT:")
        print("-" * 60)
        print(embedding_text)
        print("-" * 60)
        print(f"\nEmbedding statistics:")
        print(f"   Total length: {len(embedding_text)} characters")
        print(f"   Number of lines: {len(embedding_text.splitlines())}")

        # Check if target text is in embedding
        if "Unregistered Sales" in embedding_text:
            print("   ✓ Target text found in embedding")
        else:
            print("   ⚠ Target text NOT found in embedding")
        print()

        # Run tests
        print("6. Running verification tests...")

        # Test 1: No raw HTML/XML document
        if "<?xml version" in embedding_text or "<!DOCTYPE" in embedding_text or "<html" in embedding_text:
            print("   ❌ FAIL: Raw document markup found in embedding_text!")
            return False
        else:
            print("   ✓ PASS: No raw document markup in embedding_text")

        # Test 2: Target text appears in embedding
        if "Unregistered Sales" in embedding_text or "Equity Securities" in embedding_text:
            print("   ✓ PASS: Target text found in embedding")
        else:
            print("   ❌ FAIL: Target text not found in embedding!")
            return False

        # Test 2b: Check for duplication
        target_phrase = "Unregistered Sales of Equity Securities"
        if target_phrase in text_node['content_preview']:
            occurrences = embedding_text.count(target_phrase)
            if occurrences > 1:
                print(f"   ⚠ WARNING: Target phrase appears {occurrences} times (possible duplication)")
            else:
                print(f"   ✓ PASS: No duplication of target phrase")

        # Test 3: Reasonable size
        if len(embedding_text) < 10:
            print(f"   ❌ FAIL: Embedding text too short ({len(embedding_text)} chars)")
            return False
        elif len(embedding_text) > 50000:
            print(f"   ❌ FAIL: Embedding text too long ({len(embedding_text)} chars)")
            return False
        else:
            print(f"   ✓ PASS: Embedding text has reasonable size ({len(embedding_text)} chars)")

        # Test 4: No error messages
        if "Element not found" in embedding_text or "Error:" in embedding_text:
            print("   ❌ FAIL: Error messages found in embedding_text!")
            return False
        else:
            print("   ✓ PASS: No error messages in embedding_text")

        # Test 5: No excessive HTML tags (should be mostly text)
        html_tag_count = embedding_text.count('<') + embedding_text.count('>')
        if html_tag_count > 20:
            print(f"   ❌ FAIL: Too many HTML tags found ({html_tag_count} angle brackets)")
            return False
        else:
            print("   ✓ PASS: Minimal HTML tags in embedding_text")

        # Test 6: Check for iXBRL namespace pollution
        if 'xmlns:' in embedding_text or 'xbrli:' in embedding_text:
            print("   ❌ FAIL: XML namespace declarations found in embedding_text!")
            return False
        else:
            print("   ✓ PASS: No XML namespace pollution")

        print()
        print("=" * 80)
        print("✅ ALL TESTS PASSED - HTML/iXBRL embeddings are working correctly!")
        print("=" * 80)
        return True

    except Exception as e:
        print(f"ERROR: Test failed with exception: {e}")
        import traceback
        traceback.print_exc()
        return False


if __name__ == "__main__":
    success = test_html_ixbrl_embedding()
    sys.exit(0 if success else 1)
