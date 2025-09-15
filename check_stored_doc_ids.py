#!/usr/bin/env python3
"""
Simple test to see what doc_ids are actually stored in analytics storage.
"""

import sys
sys.path.insert(0, 'src')

import os
import time
import tempfile
from pathlib import Path
import shutil

def main():
    # Create a simple test to see what gets stored
    from go_doc_go.processing.two_pass import TwoPassProcessor
    from go_doc_go.storage_adapters.factory import StorageFactory
    
    # Create temporary directories
    test_dir = Path("simple_test_data")
    analytics_dir = Path("simple_test_analytics")
    job_db = Path("simple_test_job.db")
    
    # Clean up
    for path in [test_dir, analytics_dir, job_db]:
        if path.exists():
            if path.is_dir():
                shutil.rmtree(path)
            else:
                path.unlink()
    
    # Create one test file
    test_dir.mkdir()
    test_file = test_dir / "test.txt"
    test_file.write_text("Test document content")
    
    # Set specific modification time
    test_mod_time = 1600000000
    os.utime(test_file, (test_mod_time, test_mod_time))
    
    print(f"Created test file: {test_file}")
    print(f"Modification time: {test_mod_time}")
    
    # Create processor config
    config_dict = {
        'storage': {
            'job': {
                'type': 'sqlite',
                'path': str(job_db)
            },
            'analytics': {
                'type': 'parquet',
                'base_path': str(analytics_dir),
                'partitioning': ['run_id', 'doc_type']
            }
        },
        'embedding': {
            'enabled': False
        }
    }
    
    # Create source config
    source_configs = [{
        'name': 'test-files',
        'type': 'file',
        'base_path': str(test_dir),
        'file_pattern': '*.txt'
    }]
    
    # Create temporary config file
    import tempfile
    import yaml
    temp_config_fd, temp_config_path = tempfile.mkstemp(suffix='.yaml')
    try:
        with os.fdopen(temp_config_fd, 'w') as f:
            yaml.dump(config_dict, f)
        
        # Create processor with Config object
        from go_doc_go import Config
        config = Config(temp_config_path)
        processor = TwoPassProcessor(config)
    
        print("\n=== PROCESSING DOCUMENT ===")
        result = processor.process_local(source_configs)
        print(f"Processing result: {result}")
        
        # Get analytics storage from processor for direct testing
        analytics_storage = processor.analytics_storage
        
        # Test both original and compound doc_ids (use absolute paths)
        original_doc_id = str(test_file.absolute())
        compound_doc_id = f"{original_doc_id}::{test_mod_time}"
        
        print("\n=== CHECKING ANALYTICS STORAGE ===")
        print(f"Expected analytics dir: {analytics_dir}")
        
        # Check both configured and default locations for parquet files
        possible_dirs = [analytics_dir, Path("./data-lake")]
        
        for check_dir in possible_dirs:
            print(f"\nChecking directory: {check_dir}")
            if check_dir.exists():
                parquet_files = list(check_dir.rglob("*.parquet"))
                print(f"  Found {len(parquet_files)} parquet files:")
                for pf in parquet_files:
                    rel_path = pf.relative_to(check_dir)
                    size = pf.stat().st_size
                    print(f"    {rel_path} ({size} bytes)")
            else:
                print(f"  Directory does not exist")
        
        # Always test doc_ids regardless of file location
        print("\n=== CHECKING DOC_IDS IN ELEMENTS ===")
        test_doc_ids = [original_doc_id, compound_doc_id]
        
        for doc_id in test_doc_ids:
            try:
                elements = analytics_storage.get_document_elements(doc_id)
                print(f"Doc ID: '{doc_id}' -> {len(elements)} elements")
                if elements:
                    for elem in elements[:2]:  # Show first 2 elements
                        elem_id = elem.get('element_id', 'NO_ID')
                        elem_doc_id = elem.get('doc_id', 'NO_DOC_ID')
                        preview = elem.get('content_preview', 'NO_PREVIEW')[:50]
                        print(f"  Element: {elem_id}, doc_id: {elem_doc_id}, preview: {preview}")
            except Exception as e:
                print(f"Doc ID: '{doc_id}' -> ERROR: {e}")
        
        # Don't clean up - preserve for inspection
        print(f"\n=== FILES PRESERVED ===")
        print(f"Test data: {test_dir}")
        print(f"Expected analytics: {analytics_dir}")  
        print(f"Actual analytics: ./data-lake")
        print(f"Job DB: {job_db}")
        
        print("\n=== EXPECTED vs ACTUAL ===")
        print(f"Expected compound doc_id: {compound_doc_id}")
        print("Check above to see what doc_id is actually stored in elements.")
        
    finally:
        # Clean up temp config file
        if os.path.exists(temp_config_path):
            os.unlink(temp_config_path)

if __name__ == "__main__":
    main()