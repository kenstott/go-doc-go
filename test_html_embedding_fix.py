#!/usr/bin/env python3
"""
Focused test for HTML element contextual embedding.
Verifies that HTML text nodes get proper context without raw document content.
"""

import sys
import os
import json
import tempfile
sys.path.insert(0, os.path.join(os.path.dirname(__file__), 'src'))

from go_doc_go.document_parser.html import HtmlParser
from go_doc_go.embeddings.contextual_embedding import ContextualEmbeddingGenerator
from go_doc_go.embeddings.fastembed import FastEmbedGenerator
from go_doc_go.config import Config
from go_doc_go.storage import ElementType

# Test HTML document - Apple financial report sample
TEST_HTML = """<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml">
<head>
    <title>Apple Inc. Form 10-Q</title>
    <meta http-equiv="Content-Type" content="text/html; charset=UTF-8"/>
</head>
<body>
    <div class="document">
        <div class="cover-page">
            <h1>United States Securities and Exchange Commission</h1>
            <h2>Form 10-Q</h2>
            <h3>Quarterly Report</h3>
        </div>

        <div class="company-info">
            <p><strong>Company Name:</strong> Apple Inc.</p>
            <p><strong>Trading Symbol:</strong> AAPL</p>
            <p><strong>CIK:</strong> 0000320193</p>
            <p><strong>Fiscal Year End:</strong> September 25</p>
        </div>

        <div class="financial-data">
            <h2>Financial Highlights</h2>
            <div class="revenue-section">
                <h3>Revenue Summary</h3>
                <table class="financial-table">
                    <thead>
                        <tr>
                            <th>Category</th>
                            <th>Q1 2021</th>
                            <th>Q1 2020</th>
                        </tr>
                    </thead>
                    <tbody>
                        <tr>
                            <td>Products Revenue</td>
                            <td>$95,678 million</td>
                            <td>$79,123 million</td>
                        </tr>
                        <tr>
                            <td>Services Revenue</td>
                            <td>$15,761 million</td>
                            <td>$12,715 million</td>
                        </tr>
                        <tr class="total-row">
                            <td><strong>Total Revenue</strong></td>
                            <td><strong>$111,439 million</strong></td>
                            <td><strong>$91,838 million</strong></td>
                        </tr>
                    </tbody>
                </table>
            </div>

            <div class="earnings-section">
                <h3>Earnings Information</h3>
                <p><strong>Period:</strong> Three months ended December 26, 2020</p>
                <p><strong>Net Income:</strong> $28,755 million</p>
                <p><strong>Earnings per Share (Diluted):</strong> $1.68</p>
                <p>The Company recorded its highest quarterly revenue ever.</p>
            </div>
        </div>

        <div class="risk-factors">
            <h2>Risk Factors</h2>
            <ul>
                <li>Global economic conditions could materially adversely affect the Company</li>
                <li>The Company faces intense competition</li>
                <li>The Company depends on component availability</li>
            </ul>
        </div>

        <div class="signatures">
            <p class="signature-line">
                <span class="signature-name">/s/ Luca Maestri</span><br/>
                <span class="signature-title">Senior Vice President, Chief Financial Officer</span><br/>
                <span class="signature-date">January 27, 2021</span>
            </p>
        </div>
    </div>
</body>
</html>"""


def test_html_embedding():
    """Test that HTML text nodes get proper contextual embeddings without raw document content."""

    print("=" * 80)
    print("TEST: HTML Element Contextual Embedding")
    print("=" * 80)
    print()

    # Create a temporary HTML file
    with tempfile.NamedTemporaryFile(mode='w', suffix='.html', delete=False) as f:
        f.write(TEST_HTML)
        html_file = f.name

    try:
        # Parse the HTML document
        print("1. Parsing HTML document...")
        parser = HtmlParser()
        result = parser.parse({
            'content': TEST_HTML,
            'id': html_file
        })

        elements = result['elements']
        print(f"   Parsed {len(elements)} elements")

        # Debug: Look for specific text nodes
        # In HTML parser, text is embedded within elements like <p>, <td>, etc.
        print("\n2. Looking for elements containing specific text:")
        target_texts = ["$28,755 million", "28,755 million", "28755", "Net Income"]
        text_node = None

        # Find elements containing our target text
        for elem in elements:
            element_type = elem.get('element_type', '')
            # Look in paragraph, table cell, header, and list item elements
            if element_type in [ElementType.PARAGRAPH.value, ElementType.TABLE_CELL.value,
                               ElementType.HEADER.value, ElementType.LIST_ITEM.value,
                               'td', 'p', 'h1', 'h2', 'h3', 'strong']:
                content = str(elem.get('content_preview', ''))
                for target in target_texts:
                    if target in content:
                        print(f"\n   Found '{elem['content_preview'][:50]}' in {element_type} element")
                        # Check if this is the Net Income value
                        if 'Net Income' in content or '$28,755' in content:
                            text_node = elem
                            break
                if text_node:
                    break

        # If we didn't find Net Income, look for Trading Symbol
        if not text_node:
            print("   Net Income not found, looking for Trading Symbol...")
            for elem in elements:
                element_type = elem.get('element_type', '')
                if element_type in [ElementType.PARAGRAPH.value, ElementType.TABLE_CELL.value, 'p', 'td']:
                    content = str(elem.get('content_preview', ''))
                    if "Trading Symbol: AAPL" in content or "AAPL" in content:
                        if len(content) < 100:  # Reasonably short element
                            text_node = elem
                            break

        # If still not found, look for earnings per share
        if not text_node:
            print("   Looking for earnings per share value...")
            for elem in elements:
                element_type = elem.get('element_type', '')
                if element_type in [ElementType.PARAGRAPH.value, ElementType.TABLE_CELL.value, 'p', 'td']:
                    content = str(elem.get('content_preview', ''))
                    if "$1.68" in content or "1.68" in content:
                        if "Earnings" in content or "Share" in content:
                            text_node = elem
                            break

        if not text_node:
            print("ERROR: Could not find a suitable element for testing!")
            print("Available elements with text:")
            text_elements = [e for e in elements if e.get('element_type') in
                           [ElementType.PARAGRAPH.value, ElementType.TABLE_CELL.value,
                            ElementType.HEADER.value, 'p', 'td', 'h1', 'h2', 'h3']]
            for i, elem in enumerate(text_elements[:10]):
                print(f"   {i} ({elem['element_type']}): {elem.get('content_preview', '')[:60]}")
            return False

        print(f"   Found text node: {text_node['element_id']}")
        print(f"   Content: '{text_node['content_preview']}'")
        location = json.loads(text_node.get('content_location', '{}'))
        print(f"   Path: {location.get('path', 'unknown')}")
        print()

        # Generate contextual embeddings
        print("3. Generating contextual embeddings...")
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

        # Check the text node's embedding
        if text_node['element_id'] not in embeddings:
            print("ERROR: Text node has no embedding!")
            return False

        embedding_data = embeddings[text_node['element_id']]
        embedding_text = embedding_data.get('embedding_text', '')

        print("4. Analyzing embedding_text...")
        print("-" * 40)
        print("Full embedding_text:")
        print("Length:", len(embedding_text), "characters")
        print("Content:")
        print(embedding_text)
        print("-" * 40)
        print()

        # Run tests
        print("5. Running verification tests...")

        # Test 1: No raw HTML document
        if "<!DOCTYPE html" in embedding_text or "<html" in embedding_text:
            print("   ❌ FAIL: Raw HTML document found in embedding_text!")
            print("   The embedding contains the raw HTML")
            return False
        else:
            print("   ✓ PASS: No raw HTML document in embedding_text")

        # Test 2: No duplicate of the main text
        target_text = text_node['content_preview'].strip()
        occurrences = embedding_text.count(target_text)
        if occurrences > 1:
            print(f"   ❌ FAIL: Text '{target_text}' appears {occurrences} times (should be 1)")
            # Show where duplicates appear
            lines = embedding_text.split('\n')
            for i, line in enumerate(lines):
                if target_text in line:
                    print(f"      Line {i}: {line[:100]}")
            return False
        else:
            print(f"   ✓ PASS: Text '{target_text}' appears exactly once")

        # Test 3: Contains expected context elements
        # Different expected context depending on which text node we found
        if "AAPL" in target_text:
            expected_context = ['Trading Symbol', 'Company', 'Apple']
        elif "$28,755" in target_text or "28,755" in target_text:
            expected_context = ['Net Income', 'Earnings', 'million']
        elif "$1.68" in target_text or "1.68" in target_text:
            expected_context = ['Earnings', 'Share', 'Diluted']
        else:
            expected_context = ['Financial', 'Revenue', 'Apple']

        found_context = []
        for ctx in expected_context:
            if ctx in embedding_text:
                found_context.append(ctx)

        if len(found_context) > 0:
            print(f"   ✓ PASS: Found expected context elements: {found_context}")
        else:
            print(f"   ⚠ WARNING: No expected context elements found")
            print(f"   Expected: {expected_context}")

        # Test 4: Reasonable size
        if len(embedding_text) < 10:
            print(f"   ❌ FAIL: Embedding text too short ({len(embedding_text)} chars)")
            return False
        elif len(embedding_text) > 10000:
            print(f"   ❌ FAIL: Embedding text too long ({len(embedding_text)} chars)")
            return False
        else:
            print(f"   ✓ PASS: Embedding text has reasonable size ({len(embedding_text)} chars)")

        # Test 5: No error messages
        if "Element not found" in embedding_text or "Error:" in embedding_text:
            print("   ❌ FAIL: Error messages found in embedding_text!")
            return False
        else:
            print("   ✓ PASS: No error messages in embedding_text")

        # Test 6: No HTML tags in embedding (should be text only)
        if "<td>" in embedding_text or "<div>" in embedding_text or "<p>" in embedding_text:
            print("   ❌ FAIL: HTML tags found in embedding_text!")
            print("   Embedding should contain text content only, not HTML markup")
            return False
        else:
            print("   ✓ PASS: No HTML tags in embedding_text")

        print()
        print("=" * 80)
        print("✅ ALL TESTS PASSED - HTML embeddings are working correctly!")
        print("=" * 80)
        return True

    except Exception as e:
        print(f"ERROR: Test failed with exception: {e}")
        import traceback
        traceback.print_exc()
        return False

    finally:
        # Clean up temp file
        if os.path.exists(html_file):
            os.unlink(html_file)


if __name__ == "__main__":
    success = test_html_embedding()
    sys.exit(0 if success else 1)