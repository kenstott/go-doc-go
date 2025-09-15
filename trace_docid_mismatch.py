#!/usr/bin/env python3
"""
Trace the exact doc_id mismatch issue by manually testing the parser
"""
import os
import sys
sys.path.insert(0, 'src')
from pathlib import Path
from go_doc_go.document_parser.factory import get_parser_for_content
from go_doc_go.document_parser.text import TextParser

# Create test file
test_dir = Path("./trace_test_data")
test_dir.mkdir(exist_ok=True)

test_file = test_dir / "test.txt" 
test_file.write_text("Test document content for tracing")

# Set known modification time
os.utime(test_file, (1600000000, 1600000000))

print(f"Created test file: {test_file}")
print(f"File mod time: {test_file.stat().st_mtime}")

# Test what the two-pass processor would generate as compound doc_id
compound_doc_id = f"{test_file.absolute()}::{int(test_file.stat().st_mtime)}"
print(f"Expected compound doc_id: {compound_doc_id}")

# Test what the parser actually uses when given this compound doc_id
print("\n=== Testing Text Parser ===")
parser = TextParser()

# Create the content structure that two-pass processor would pass
doc_content = {
    'id': compound_doc_id,  # This is the compound ID
    'content': test_file.read_text(),
    'metadata': {
        'doc_type': 'text',
        'source': str(test_file),
        'size': test_file.stat().st_size,
        'modified': test_file.stat().st_mtime
    }
}

print(f"doc_content['id']: {doc_content['id']}")
print(f"doc_content['metadata']: {doc_content['metadata']}")

# Parse and see what doc_id gets used
result = parser.parse(doc_content)
actual_doc_id = result['document']['doc_id']

print(f"\nActual doc_id in parsed document: {actual_doc_id}")
print(f"Compound doc_id matches: {actual_doc_id == compound_doc_id}")

if actual_doc_id != compound_doc_id:
    print(f"❌ MISMATCH FOUND!")
    print(f"  Expected: {compound_doc_id}")
    print(f"  Actual:   {actual_doc_id}")
    print(f"  This explains why incremental processing fails!")
else:
    print(f"✅ Doc IDs match - parser fix is working")

# Check elements too
elements = result['elements']
print(f"\nElements in parsed result: {len(elements)}")
for i, elem in enumerate(elements[:3]):
    print(f"  Element {i}: {elem['element_id']} (doc_id: {elem.get('doc_id', 'MISSING')})")

# Cleanup
import shutil
shutil.rmtree(test_dir)