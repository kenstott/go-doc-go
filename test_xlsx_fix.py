#!/usr/bin/env python3
import sys
sys.path.insert(0, 'src')

from go_doc_go.document_parser.xlsx_go import GoXLSXParser

parser = GoXLSXParser()
print(f"Parser created - binary: {parser.binary_path}")

result = parser.parse({
    'id': 'test',
    'content': '/Users/kennethstott/PycharmProjects/doculyzer-go-conversion/tests/assets/test_sample.xlsx'
})

print(f"Elements: {len(result.get('elements', []))}")
print(f"Has error: {'error' in result.get('document', {}).get('metadata', {})}")

if result.get('elements'):
    print("SUCCESS - Parser working correctly")
else:
    print(f"WARNING - No elements parsed")
    if 'error' in result.get('document', {}).get('metadata', {}):
        print(f"Error: {result['document']['metadata']['error']}")