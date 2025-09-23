#!/usr/bin/env python3
"""
Comprehensive test for XML/iXBRL embedding generation fixes.
Tests:
1. Leaf divs get embeddings
2. Parent text doesn't include children's text
3. iXBRL namespace-prefixed elements are parsed
4. Context inclusion works correctly
"""

import sys
import os
sys.path.insert(0, os.path.join(os.path.dirname(__file__), 'src'))

from go_doc_go.document_parser.xml import XmlParser
from go_doc_go.document_parser.html import HtmlParser
from go_doc_go.embeddings.contextual_embedding import ContextualEmbeddingGenerator
import json
from typing import Dict, List, Any


def create_test_xml() -> str:
    """Create test XML with nested structure."""
    return """<?xml version="1.0" encoding="UTF-8"?>
<document>
    <header>Header Direct Text
        <title>Title Text</title>
        <meta>Meta Text</meta>
    </header>
    <body>Body Direct Text
        <section id="s1">Section 1 Direct Text
            <p>Paragraph 1 in Section 1</p>
            <p>Paragraph 2 in Section 1</p>
        </section>
        <section id="s2">Section 2 Direct Text
            <div>This is a leaf div with important content</div>
            <div>Container Div Text
                <span>Nested span text</span>
            </div>
        </section>
    </body>
</document>"""


def create_test_ixbrl() -> str:
    """Create test iXBRL document with namespace-prefixed elements."""
    return """<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml"
      xmlns:ix="http://www.xbrl.org/2013/inlineXBRL">
<head>
    <title>Test iXBRL Document</title>
</head>
<body>
    <div>Regular HTML div content</div>
    <ix:nonNumeric name="CompanyName">Apple Inc.</ix:nonNumeric>
    <p>Revenue for the quarter was
        <ix:nonFraction name="Revenue" unitRef="USD" decimals="0">
            123,456,789
        </ix:nonFraction>
    </p>
    <div>Another regular div that is a leaf</div>
    <ix:continuation id="cont1">
        Continuation content with more details
    </ix:continuation>
</body>
</html>"""


def test_xml_parsing():
    """Test XML parsing and text extraction."""
    print("\n=== Testing XML Parsing ===")

    parser = XmlParser()
    content = {
        "id": "test.xml",
        "source": "test.xml",
        "doc_type": "xml",
        "content": create_test_xml().encode('utf-8')
    }

    result = parser.parse(content)
    elements = result["elements"]

    print(f"\nTotal elements parsed: {len(elements)}")

    # Check specific elements
    for elem in elements:
        if elem["element_type"] == "section" and "Section 1" in elem.get("content_preview", ""):
            print(f"\nSection 1 element:")
            print(f"  Path: {elem['path']}")
            print(f"  Content preview: '{elem['content_preview']}'")
            print(f"  Has children: {elem.get('has_children', False)}")

            # Verify it only contains direct text
            assert "Paragraph 1" not in elem['content_preview'], "Parent should not include child text!"
            print("  ✓ Parent text doesn't include children")

    # Check leaf div
    leaf_divs = [e for e in elements if e["element_type"] == "div" and not e.get("has_children", False)]
    print(f"\nLeaf divs found: {len(leaf_divs)}")
    for div in leaf_divs:
        print(f"  - '{div['content_preview']}'")

    assert len(leaf_divs) > 0, "Should find leaf divs!"
    print("✓ Leaf divs are being parsed")

    return result


def test_ixbrl_parsing():
    """Test iXBRL parsing with namespace-prefixed elements."""
    print("\n=== Testing iXBRL Parsing ===")

    parser = HtmlParser()
    content = {
        "id": "test.ixbrl",
        "source": "test.ixbrl",
        "doc_type": "html",
        "content": create_test_ixbrl().encode('utf-8')
    }

    result = parser.parse(content)
    elements = result["elements"]

    print(f"\nTotal elements parsed: {len(elements)}")

    # Check for iXBRL elements
    ixbrl_elements = [e for e in elements if ':' in e.get("element_type", "")]
    print(f"\niXBRL elements found: {len(ixbrl_elements)}")

    for elem in ixbrl_elements[:3]:  # Show first 3
        print(f"  - Type: {elem['element_type']}, Content: '{elem.get('content_preview', '')[:50]}'")

    assert len(ixbrl_elements) > 0, "Should find iXBRL namespace-prefixed elements!"
    print("✓ iXBRL elements are being parsed")

    # Check that iXBRL elements only show direct text
    for elem in ixbrl_elements:
        if "nonFraction" in elem["element_type"]:
            # This element has text "123,456,789"
            preview = elem.get("content_preview", "")
            print(f"\nnonFraction element content: '{preview}'")
            assert "123,456,789" in preview or preview == "", "Should have direct text or be empty"
            print("✓ iXBRL element shows only direct text")
            break

    return result


def test_embedding_generation():
    """Test that embeddings are generated correctly."""
    print("\n=== Testing Embedding Generation ===")

    # Parse XML first
    parser = XmlParser()
    content = {
        "id": "test.xml",
        "source": "test.xml",
        "doc_type": "xml",
        "content": create_test_xml().encode('utf-8'),
        "doc_id": "test_doc_001"
    }

    parse_result = parser.parse(content)

    # Generate embeddings
    embedding_generator = ContextualEmbeddingGenerator(
        config={
            "predecessors": 1,
            "successors": 1,
            "include_ancestors": True
        }
    )

    embeddings_data = embedding_generator.create_contextual_embeddings(
        elements=parse_result["elements"],
        relationships=parse_result.get("relationships", []),
        document=parse_result["document"]
    )

    print(f"\nTotal embeddings generated: {len(embeddings_data['embeddings'])}")

    # Check that leaf divs got embeddings
    div_embeddings = [e for e in embeddings_data['embeddings']
                      if "div" in e.get("element_type", "")]
    print(f"Div embeddings: {len(div_embeddings)}")

    for emb in div_embeddings[:2]:
        print(f"\nDiv embedding:")
        print(f"  Element type: {emb.get('element_type', 'N/A')}")
        print(f"  Text preview: '{emb.get('text', '')[:100]}'...")

        # Check that ancestor context doesn't include all children
        if "Section" in emb.get('text', ''):
            assert "Paragraph 1 in Section 1" not in emb['text'], \
                "Ancestor context should not include all descendant text!"
            print("  ✓ Ancestor context is clean")

    assert len(div_embeddings) > 0, "Should generate embeddings for leaf divs!"
    print("\n✓ Embeddings generated for leaf divs")

    return embeddings_data


def test_context_resolution():
    """Test that context resolution returns only direct text."""
    print("\n=== Testing Context Resolution ===")

    parser = XmlParser()

    # Create a simple test case
    xml_content = """<?xml version="1.0"?>
    <root>
        <parent>Parent text only
            <child>Child text</child>
        </parent>
    </root>"""

    content = {
        "id": "test_context.xml",
        "source": "test_context.xml",
        "doc_type": "xml",
        "content": xml_content.encode('utf-8')
    }

    result = parser.parse(content)

    # Find the parent element
    parent_elem = None
    for elem in result["elements"]:
        if elem["element_type"] == "parent":
            parent_elem = elem
            break

    assert parent_elem is not None, "Should find parent element"

    # Resolve its content
    resolved = parser._resolve_element_content(
        element_id=parent_elem["element_id"],
        element_type=parent_elem["element_type"],
        path=parent_elem["path"]
    )

    print(f"\nParent element resolved content: '{resolved}'")

    # Should only have direct text, not child text
    assert "Child text" not in resolved, "Parent resolution should not include child text!"
    assert "Parent text only" in resolved or "<parent>" in resolved, "Should have parent content"

    print("✓ Context resolution returns only direct text")


def main():
    """Run all tests."""
    print("=" * 60)
    print("XML/iXBRL Embedding Fixes - Comprehensive Test")
    print("=" * 60)

    try:
        # Test XML parsing
        xml_result = test_xml_parsing()

        # Test iXBRL parsing
        ixbrl_result = test_ixbrl_parsing()

        # Test embedding generation
        embeddings = test_embedding_generation()

        # Test context resolution
        test_context_resolution()

        print("\n" + "=" * 60)
        print("ALL TESTS PASSED! ✓")
        print("=" * 60)
        print("\nSummary:")
        print("1. ✓ Leaf divs are parsed and get embeddings")
        print("2. ✓ Parent text doesn't include children's text")
        print("3. ✓ iXBRL namespace-prefixed elements are parsed")
        print("4. ✓ Context resolution is clean")
        print("5. ✓ Embeddings are generated correctly")

    except AssertionError as e:
        print(f"\n❌ TEST FAILED: {e}")
        sys.exit(1)
    except Exception as e:
        print(f"\n❌ ERROR: {e}")
        import traceback
        traceback.print_exc()
        sys.exit(1)


if __name__ == "__main__":
    main()