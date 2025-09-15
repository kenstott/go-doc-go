#!/usr/bin/env python3
"""
Debug incremental processing with detailed logging
"""
import os
import sys
import tempfile
import time
import shutil
from pathlib import Path

# Create test data
test_dir = Path("./debug_detailed_data")
analytics_dir = Path("./debug_detailed_analytics") 
job_db = Path("./debug_detailed_job.db")

# Cleanup any existing data
for path in [test_dir, analytics_dir, job_db]:
    if path.exists():
        if path.is_dir():
            shutil.rmtree(path)
        else:
            path.unlink()

test_dir.mkdir()

# Create test file
test_file = test_dir / "test.txt" 
test_file.write_text("Test document content")
os.utime(test_file, (1600000000, 1600000000))

compound_doc_id = f"{test_file.absolute()}::{int(test_file.stat().st_mtime)}"
print(f"Expected compound doc_id: {compound_doc_id}")

# Import after setting up test data
sys.path.insert(0, 'src')
from go_doc_go.storage_adapters.analytics_parquet import ParquetAnalyticsStorage
from go_doc_go.processing.two_pass import TwoPassProcessor
from go_doc_go import Config

# Run 1: Initial processing
print(f"\n=== RUN 1: Initial Processing ===")

# Create config for run 1
config_data = {
    "content_sources": [
        {
            "name": "debug-test",
            "type": "file",
            "base_path": str(test_dir.absolute()),
            "file_pattern": "*.txt"
        }
    ],
    "storage": {
        "job": {"type": "sqlite", "path": str(job_db)},
        "analytics": {"type": "parquet", "path": str(analytics_dir)}
    },
    "processing": {"worker_mode": "two_pass"},
    "embedding": {"enabled": True},
    "relationship_detection": {"enabled": True}
}

# Create temporary config file
import tempfile
import yaml
with tempfile.NamedTemporaryFile(mode='w', suffix='.yaml', delete=False) as f:
    yaml.dump(config_data, f)
    config_file = f.name

try:
    config = Config(config_file)
    
    # Create two-pass processor
    processor = TwoPassProcessor(config)
    print(f"Processor run_id: {processor.run_id}")
    
    # Run processing
    result1 = processor.process_local()
    print(f"Run 1 result: {result1}")
    
    # Check what was stored
    print(f"\n=== Checking storage after Run 1 ===")
    analytics_storage = ParquetAnalyticsStorage({'path': str(analytics_dir)})
    
    if analytics_dir.exists():
        elements_dir = analytics_dir / "elements"
        if elements_dir.exists():
            print(f"Elements directory exists: {elements_dir}")
            
            # List all parquet files
            import glob
            elements_pattern = str(elements_dir / "**" / "*.parquet")
            parquet_files = glob.glob(elements_pattern, recursive=True)
            print(f"Parquet files found: {len(parquet_files)}")
            for f in parquet_files:
                print(f"  - {f}")
                
            if parquet_files:
                # Query directly
                elements = analytics_storage.get_document_elements(compound_doc_id)
                print(f"Direct query for compound_doc_id returned: {len(elements)} elements")
                for elem in elements:
                    print(f"  - {elem['element_id']} (doc_id: {elem['doc_id']})")
                
                # DEBUG: Show what doc_ids are actually stored
                print(f"DEBUG: Inspecting stored doc_ids...")
                import duckdb
                elements_pattern = str(elements_dir / "**" / "*.parquet")
                conn = duckdb.connect(':memory:')
                try:
                    conn.execute(f"CREATE VIEW elements AS SELECT * FROM read_parquet('{elements_pattern}', union_by_name=True)")
                    stored_doc_ids = conn.execute("SELECT DISTINCT doc_id FROM elements").fetchall()
                    print(f"Stored doc_ids: {[d[0] for d in stored_doc_ids]}")
                    print(f"Query doc_id:   {compound_doc_id}")
                    
                    for stored_id in stored_doc_ids:
                        stored = stored_id[0]
                        print(f"  Match: {stored == compound_doc_id}")
                        if stored != compound_doc_id:
                            print(f"  MISMATCH DETECTED!")
                            print(f"    Stored:  '{stored}' (len={len(stored)})")
                            print(f"    Query:   '{compound_doc_id}' (len={len(compound_doc_id)})")
                finally:
                    conn.close()
            else:
                print("No parquet files found!")
        else:
            print(f"Elements directory does not exist: {elements_dir}")
    else:
        print(f"Analytics directory does not exist: {analytics_dir}")
    
    # Wait a moment to ensure filesystem sync
    print(f"\nWaiting 2 seconds for filesystem sync...")
    time.sleep(2)
    
    # Run 2: Should skip documents
    print(f"\n=== RUN 2: Should Skip Documents ===")
    
    # Create new processor with same config (should get same run_id)
    processor2 = TwoPassProcessor(config)
    print(f"Processor2 run_id: {processor2.run_id}")
    print(f"Same run_id as Run 1: {processor2.run_id == processor.run_id}")
    
    # Before running, manually test the document check
    print(f"\n=== Manual document check before Run 2 ===")
    is_processed = processor2._document_already_processed(compound_doc_id)
    print(f"_document_already_processed({compound_doc_id}) = {is_processed}")
    
    # Run processing again
    result2 = processor2.process_local()
    print(f"Run 2 result: {result2}")
    
    # The key test: Run 2 should process 0 documents if incremental works
    run2_docs = result2.get('documents', -1)
    if run2_docs == 0:
        print(f"✅ SUCCESS: Incremental processing working - 0 documents processed in Run 2")
    else:
        print(f"❌ FAILURE: Incremental processing broken - {run2_docs} documents processed in Run 2")

finally:
    # Cleanup
    os.unlink(config_file)
    for path in [test_dir, analytics_dir, job_db]:
        if path.exists():
            if path.is_dir():
                shutil.rmtree(path)
            else:
                path.unlink()
    
    print(f"\nCleanup complete")