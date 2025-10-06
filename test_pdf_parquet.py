#!/usr/bin/env python3
"""Test PDF and Parquet Go parsers."""

import sys
import os
sys.path.insert(0, 'src')

from go_doc_go.document_parser.factory import create_parser

# Enable Go modules
os.environ["USE_GO_MODULES"] = "true"

print("Testing PDF and Parquet Go parsers...")
print("=" * 60)

# Test PDF
pdf_file = "tests/assets/departments.pdf"
if os.path.exists(pdf_file):
    try:
        with open(pdf_file, 'rb') as f:
            pdf_content = f.read()
        
        parser = create_parser("pdf", {})
        result = parser.parse({
            "id": "test_pdf",
            "content": pdf_content,
            "metadata": {"filename": pdf_file}
        })
        
        if "document" in result and "elements" in result:
            print(f"✅ PDF: {len(result['elements'])} elements, {len(result['relationships'])} relationships")
        else:
            print(f"❌ PDF: Invalid result structure")
    except Exception as e:
        print(f"❌ PDF: {e}")
else:
    print(f"⚠️  PDF: Test file not found: {pdf_file}")

# Test Parquet
parquet_file = "tests/e2e/test_documents/techcorp_q4_2024_earnings.parquet"
if os.path.exists(parquet_file):
    try:
        with open(parquet_file, 'rb') as f:
            parquet_content = f.read()
        
        parser = create_parser("parquet", {})
        result = parser.parse({
            "id": "test_parquet",
            "content": parquet_content,
            "metadata": {"filename": parquet_file}
        })
        
        if "document" in result and "elements" in result:
            print(f"✅ Parquet: {len(result['elements'])} elements, {len(result['relationships'])} relationships")
        else:
            print(f"❌ Parquet: Invalid result structure")
    except Exception as e:
        print(f"❌ Parquet: {e}")
else:
    print(f"⚠️  Parquet: Test file not found: {parquet_file}")

print("=" * 60)
