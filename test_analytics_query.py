#!/usr/bin/env python3
"""
Test the analytics storage query directly to debug the doc_id lookup issue
"""
import os
import sys
import tempfile
import shutil
from pathlib import Path
sys.path.insert(0, 'src')

from go_doc_go.storage_adapters.analytics_parquet import ParquetAnalyticsStorage

# Create test data and storage
test_dir = Path("./query_test_data")  
test_dir.mkdir(exist_ok=True)

analytics_dir = Path("./query_test_analytics")
analytics_dir.mkdir(exist_ok=True)

# Create test file
test_file = test_dir / "test.txt"
test_file.write_text("Test content for query debugging")
os.utime(test_file, (1600000000, 1600000000))

compound_doc_id = f"{test_file.absolute()}::{int(test_file.stat().st_mtime)}"
print(f"Testing compound doc_id: {compound_doc_id}")

# Create analytics storage
config = {'path': str(analytics_dir.absolute())}
analytics_storage = ParquetAnalyticsStorage(config)

# Create test document and elements data to store
test_document = {
    'doc_id': compound_doc_id,
    'doc_type': 'text', 
    'source': str(test_file),
    'metadata': {'size': 100},
    'created_at': '2025-09-14',
    'updated_at': '2025-09-14'
}

test_elements = [
    {
        'element_id': 'root_123',
        'doc_id': compound_doc_id,  # KEY: This should match what we query for
        'element_type': 'root',
        'content_preview': 'Root element',
        'metadata': {}
    },
    {
        'element_id': 'para_456', 
        'doc_id': compound_doc_id,  # KEY: This should match what we query for
        'element_type': 'paragraph',
        'content_preview': 'Test paragraph content',
        'metadata': {}
    }
]

print(f"\nStoring document with doc_id: {test_document['doc_id']}")
print(f"Storing {len(test_elements)} elements with doc_id: {test_elements[0]['doc_id']}")

# Store the data
run_id = "test_run_12345"
try:
    analytics_storage.append_documents([test_document], run_id)
    analytics_storage.append_elements(test_elements, run_id)
    print("✅ Data stored successfully")
except Exception as e:
    print(f"❌ Error storing data: {e}")
    sys.exit(1)

# Now test the query
print(f"\n=== Testing Query ===")
print(f"Querying for doc_id: {compound_doc_id}")

try:
    found_elements = analytics_storage.get_document_elements(compound_doc_id)
    print(f"✅ Query succeeded")
    print(f"Found {len(found_elements)} elements")
    
    if found_elements:
        print("Found elements:")
        for i, elem in enumerate(found_elements):
            print(f"  {i}: {elem.get('element_id')} (doc_id: {elem.get('doc_id')})")
    else:
        print("❌ No elements found - this is the bug!")
        
        # Let's check what doc_ids are actually stored
        print("\n=== Debugging: What doc_ids are actually stored? ===")
        import duckdb
        elements_path = os.path.join(analytics_dir, 'elements/**/*.parquet')
        
        conn = duckdb.connect(':memory:')
        try:
            conn.execute(f"CREATE VIEW elements AS SELECT * FROM read_parquet('{elements_path}', union_by_name=True)")
            stored_doc_ids = conn.execute("SELECT DISTINCT doc_id FROM elements").fetchall()
            print(f"Stored doc_ids ({len(stored_doc_ids)}):")
            for doc_id in stored_doc_ids:
                stored_id = doc_id[0]
                print(f"  - '{stored_id}'")
                print(f"    Length: {len(stored_id)}")
                print(f"    Equals query: {stored_id == compound_doc_id}")
                print(f"    Type: {type(stored_id)}")
                if stored_id != compound_doc_id:
                    print(f"    Difference found!")
                    # Show character-by-character comparison
                    for i, (c1, c2) in enumerate(zip(stored_id, compound_doc_id)):
                        if c1 != c2:
                            print(f"      Char {i}: stored='{c1}' (ord={ord(c1)}) vs query='{c2}' (ord={ord(c2)})")
                            break
        finally:
            conn.close()
        
except Exception as e:
    print(f"❌ Query failed: {e}")
    import traceback
    traceback.print_exc()

# Cleanup
shutil.rmtree(test_dir)
shutil.rmtree(analytics_dir)
print(f"\nCleanup complete")