#!/usr/bin/env python3
"""Simple test of Go parsers with small dataset."""

import sys
import os
import tempfile
import json
sys.path.insert(0, 'src')

from go_doc_go.document_parser.factory import create_parser

# Enable Go modules
os.environ["USE_GO_MODULES"] = "true"

# Test data
test_docs = [
    {
        "id": "csv_test",
        "doc_type": "csv",
        "content": "Name,Age,City\nJohn,30,NYC\nJane,25,LA"
    },
    {
        "id": "xml_test", 
        "doc_type": "xml",
        "content": '<?xml version="1.0"?><root><person><name>John</name></person></root>'
    }
]

print("Testing Go parsers with simple data...")
for doc in test_docs:
    try:
        parser = create_parser(doc["doc_type"], {})
        result = parser.parse({
            "id": doc["id"],
            "content": doc["content"],
            "metadata": {}
        })
        
        if "document" in result and "elements" in result:
            print(f"✓ {doc['doc_type']}: {len(result['elements'])} elements")
        else:
            print(f"✗ {doc['doc_type']}: Invalid result structure")
    except Exception as e:
        print(f"✗ {doc['doc_type']}: {e}")

print("Done")
