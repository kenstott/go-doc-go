#!/usr/bin/env python3
"""
Test if there's a path resolution issue in analytics storage
"""
import os
import sys
sys.path.insert(0, 'src')
import tempfile
import shutil
from pathlib import Path
from go_doc_go.storage_adapters.analytics_parquet import ParquetAnalyticsStorage

# Test with relative path like the pipeline uses
rel_path = "./test_relative_analytics"
if os.path.exists(rel_path):
    shutil.rmtree(rel_path)

print(f"Testing with relative path: {rel_path}")
print(f"Absolute version: {Path(rel_path).absolute()}")

# Create analytics storage with relative path
config = {'path': rel_path}
analytics_storage = ParquetAnalyticsStorage(config)

# Create test data
compound_doc_id = "/absolute/path/test.txt::1600000000"
run_id = "test_run_12345"

test_document = {
    'doc_id': compound_doc_id,
    'doc_type': 'text', 
    'source': '/absolute/path/test.txt',
    'metadata': {},
    'created_at': '2025-09-14',
    'updated_at': '2025-09-14'
}

test_elements = [
    {
        'element_id': 'root_123',
        'doc_id': compound_doc_id,
        'element_type': 'root', 
        'content_preview': 'Root element',
        'metadata': {}
    }
]

# Store the data
print(f"\nStoring data with relative path analytics storage...")
try:
    analytics_storage.append_documents([test_document], run_id)
    analytics_storage.append_elements(test_elements, run_id)
    print("✅ Data stored successfully")
except Exception as e:
    print(f"❌ Error storing data: {e}")
    exit(1)

# Check what was actually created
print(f"\n=== Directory structure ===")
for root, dirs, files in os.walk(rel_path):
    level = root.replace(rel_path, '').count(os.sep)
    indent = ' ' * 2 * level
    print(f'{indent}{os.path.basename(root)}/')
    subindent = ' ' * 2 * (level + 1)
    for file in files:
        if file.endswith('.parquet'):
            print(f'{subindent}{file}')

# Now test querying
print(f"\n=== Testing query with relative path ===")
try:
    found_elements = analytics_storage.get_document_elements(compound_doc_id)
    print(f"Query result: {len(found_elements)} elements found")
    
    if found_elements:
        print("✅ Query successful with relative path")
        for elem in found_elements:
            print(f"  - {elem['element_id']} (doc_id: {elem['doc_id']})")
    else:
        print("❌ Query failed - no elements found")
        
        # Debug: check if files exist
        elements_path = os.path.join(rel_path, 'elements/**/*.parquet')
        print(f"Elements path pattern: {elements_path}")
        
        import glob
        matching_files = glob.glob(elements_path, recursive=True)
        print(f"Matching files: {len(matching_files)}")
        for f in matching_files:
            print(f"  - {f}")
            
        if matching_files:
            # Try to query directly with DuckDB
            import duckdb
            conn = duckdb.connect(':memory:')
            try:
                conn.execute(f"CREATE VIEW elements AS SELECT * FROM read_parquet('{elements_path}', union_by_name=True)")
                all_docs = conn.execute("SELECT DISTINCT doc_id FROM elements").fetchall()
                print(f"All doc_ids in storage: {[d[0] for d in all_docs]}")
                
                specific_query = conn.execute(f"SELECT COUNT(*) FROM elements WHERE doc_id = '{compound_doc_id}'").fetchone()
                print(f"Direct query result: {specific_query[0]} elements")
            finally:
                conn.close()
        
except Exception as e:
    print(f"❌ Query error: {e}")
    import traceback
    traceback.print_exc()

# Cleanup
shutil.rmtree(rel_path)
print(f"\nCleanup complete")