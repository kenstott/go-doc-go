#!/usr/bin/env python3
"""
Test script for the Go PDF parser integration.
"""

import json
import os
import sys
from pathlib import Path

# Add src directory to path
sys.path.insert(0, str(Path(__file__).parent / "src"))

from go_doc_go.document_parser.factory import create_parser


def test_pdf_parser():
    """Test the PDF parser with a sample PDF file."""

    # Enable Go modules
    os.environ["USE_GO_MODULES"] = "true"

    # Look for a test PDF file
    test_files = [
        Path("tests/assets/departments.pdf"),
        Path("tests/assets/crazyones-pdfa.pdf"),
        Path("tests/assets/sample.pdf"),
        Path("tests/assets/test.pdf"),
        Path("tests/assets/document.pdf"),
    ]

    test_file = None
    for f in test_files:
        if f.exists():
            test_file = f
            break

    if not test_file:
        print("No test PDF file found. Creating a simple test...")
        # Test with minimal configuration
        parser = create_parser("pdf")
        print(f"Parser type: {parser.__class__.__name__}")

        # Test with dummy content
        content = {
            "id": "test_pdf",
            "content": "dummy_path.pdf",  # This will fail but shows the parser is created
            "metadata": {"test": True}
        }

        result = parser.parse(content)
        print(f"Parse result keys: {result.keys()}")
        print(f"Document: {result.get('document', {})}")
        print(f"Elements count: {len(result.get('elements', []))}")
        print(f"Relationships count: {len(result.get('relationships', []))}")
        return

    print(f"Testing with file: {test_file}")

    # Create parser
    parser = create_parser("pdf", {
        "max_pages": 5,
        "extract_metadata": True,
        "extract_links": True,
    })

    print(f"Parser type: {parser.__class__.__name__}")

    # Parse the PDF
    content = {
        "id": test_file.stem,
        "content": str(test_file),
        "metadata": {
            "filename": test_file.name,
            "source": "test"
        }
    }

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
        print("First 3 elements:")
        for i, elem in enumerate(result['elements'][:3]):
            print(f"  {i+1}. [{elem['element_type']}] {elem.get('content_preview', '')[:50]}...")

    print(f"\nRelationships: {len(result['relationships'])}")

    if result.get('links'):
        print(f"Links found: {len(result['links'])}")
        for link in result['links'][:3]:
            print(f"  - [{link['link_type']}] {link['link_target']}")


if __name__ == "__main__":
    test_pdf_parser()