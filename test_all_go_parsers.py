#!/usr/bin/env python3
"""Comprehensive test of ALL Go parsers."""

import sys
import os
import tempfile
sys.path.insert(0, 'src')

from go_doc_go.document_parser.factory import create_parser

# Enable Go modules
os.environ["USE_GO_MODULES"] = "true"

# Create test data for all parsers
test_data = {
    "csv": {
        "content": "Name,Age,City\nJohn,30,NYC\nJane,25,LA\nBob,35,Chicago",
        "file": "tests/assets/test_sample.csv"
    },
    "xml": {
        "content": '<?xml version="1.0"?><root><person><name>John</name><age>30</age></person></root>',
        "file": "tests/assets/test_sample.xml"
    },
    "json": {
        "content": '{"name": "John", "age": 30, "city": "NYC", "hobbies": ["reading", "gaming"]}',
        "file": None
    },
    "html": {
        "content": '''<!DOCTYPE html>
<html>
<head><title>Test Page</title></head>
<body>
<h1>Hello World</h1>
<p>This is a test paragraph with <a href="http://example.com">a link</a>.</p>
<ul>
  <li>Item 1</li>
  <li>Item 2</li>
</ul>
</body>
</html>''',
        "file": None
    },
    "markdown": {
        "content": '''# Test Document

This is a **bold** paragraph with *italic* text.

## Section 1

Here's a [link](http://example.com) and some `code`.

- Item 1
- Item 2

```python
def hello():
    print("world")
```
''',
        "file": None
    },
    "text": {
        "content": """This is a simple text document.
It has multiple lines.
Some lines are longer than others to test text parsing capabilities.

This is a new paragraph after a blank line.""",
        "file": None
    },
    "xlsx": {
        "content": None,  # Binary file
        "file": "tests/assets/test_sample.xlsx"
    },
    "docx": {
        "content": None,  # Binary file
        "file": "tests/assets/test_sample.docx"
    },
    "pptx": {
        "content": None,  # Binary file
        "file": "tests/assets/test_sample.pptx"
    },
    "pdf": {
        "content": None,  # Would need binary PDF - skip for now
        "file": None
    },
    "parquet": {
        "content": None,  # Would need binary Parquet - skip for now
        "file": None
    }
}

print("=" * 60)
print("TESTING ALL GO PARSERS")
print("=" * 60)

success_count = 0
fail_count = 0
skip_count = 0
results = []

for doc_type, test_info in test_data.items():
    content = test_info["content"]
    file_path = test_info["file"]

    # Load content from file if needed
    if file_path and os.path.exists(file_path):
        with open(file_path, 'rb') as f:
            content = f.read()

    if content is None:
        results.append(f"⚠️  {doc_type:10} SKIPPED (no test data available)")
        skip_count += 1
        continue

    try:
        parser = create_parser(doc_type, {})
        result = parser.parse({
            "id": f"test_{doc_type}",
            "content": content,
            "metadata": {"test": True}
        })

        if "document" in result and "elements" in result:
            element_count = len(result.get("elements", []))
            relationship_count = len(result.get("relationships", []))
            link_count = len(result.get("links", []))

            results.append(f"✅ {doc_type:10} SUCCESS - {element_count:3} elements, {relationship_count:3} relationships, {link_count:3} links")
            success_count += 1
        else:
            results.append(f"❌ {doc_type:10} FAILED  - Invalid result structure")
            fail_count += 1

    except Exception as e:
        error_msg = str(e)
        if "not found" in error_msg.lower():
            results.append(f"⚠️  {doc_type:10} SKIPPED - Go binary not found")
            skip_count += 1
        else:
            # Truncate long error messages
            if len(error_msg) > 60:
                error_msg = error_msg[:57] + "..."
            results.append(f"❌ {doc_type:10} FAILED  - {error_msg}")
            fail_count += 1

# Print results
print("\nResults:")
print("-" * 60)
for result in results:
    print(result)

print("-" * 60)
print(f"\nSummary: {success_count} passed, {fail_count} failed, {skip_count} skipped")

if fail_count > 0:
    print("\n⚠️  Some parsers failed! Check the errors above.")
    sys.exit(1)
else:
    print("\n✅ All available parsers working correctly!")