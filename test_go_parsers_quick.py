#!/usr/bin/env python3
"""Quick test of Go parsers."""

import sys
import os
sys.path.insert(0, 'src')

from go_doc_go.document_parser.factory import create_parser

# Test files to check
test_files = [
    ("tests/assets/test_sample.csv", "csv"),
    ("tests/assets/test_sample.xml", "xml"), 
    ("tests/assets/test_sample.xlsx", "xlsx"),
    ("tests/assets/test_sample.docx", "docx"),
]

# Test with GO modules enabled
os.environ["USE_GO_MODULES"] = "true"

print("Testing Go parsers...")
for file_path, doc_type in test_files:
    if not os.path.exists(file_path):
        print(f"  ✗ {file_path} not found")
        continue
    
    try:
        with open(file_path, 'rb') as f:
            content = f.read()
        
        parser = create_parser(doc_type, {})
        result = parser.parse({
            "id": f"test_{doc_type}",
            "content": content,
            "metadata": {"filename": file_path}
        })
        
        if "document" in result and "elements" in result:
            print(f"  ✓ {doc_type}: {len(result['elements'])} elements")
        else:
            print(f"  ✗ {doc_type}: Invalid result structure")
    except Exception as e:
        print(f"  ✗ {doc_type}: {e}")

print("Done")
