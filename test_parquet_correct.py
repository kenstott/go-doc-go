#!/usr/bin/env python3
"""Test Parquet Go parser with correct interface."""

import sys
import os
sys.path.insert(0, 'src')

from go_doc_go.document_parser.factory import create_parser

# Enable Go modules
os.environ["USE_GO_MODULES"] = "true"

# Test Parquet with correct interface
parquet_file = "tests/e2e/test_documents/techcorp_q4_2024_earnings.parquet"
if os.path.exists(parquet_file):
    try:
        parser = create_parser("parquet", {})
        result = parser.parse({
            "id": "test_parquet",
            "binary_path": parquet_file,  # Use binary_path instead of content
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
