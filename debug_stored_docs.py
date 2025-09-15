#!/usr/bin/env python3
"""
Debug script to examine what document IDs are stored in analytics storage.
"""

import sys
sys.path.insert(0, 'src')

from go_doc_go import Config
from go_doc_go.storage_adapters.factory import StorageFactory
import logging

# Enable debug logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

def main():
    # Create analytics storage config directly (based on test setup)
    from go_doc_go.storage_adapters.analytics_parquet import ParquetAnalyticsStorage
    
    # Look for the parquet files created by the test
    import os
    from pathlib import Path
    
    # Check if we have analytics files from previous test runs
    test_data_dir = Path("test_data")
    if not test_data_dir.exists():
        print("ERROR: test_data directory not found. Run test_incremental_processing.py first.")
        return
        
    parquet_dir = test_data_dir / 'parquet'
    if not parquet_dir.exists():
        print("ERROR: parquet directory not found. Run test_incremental_processing.py first.")
        return
        
    analytics_config = {
        'base_path': str(parquet_dir),
        'partitioning': ['run_id', 'doc_type']
    }
    
    analytics_storage = ParquetAnalyticsStorage(analytics_config)
    analytics_storage.initialize()
    
    print("=== STORED DOCUMENTS IN ANALYTICS STORAGE ===")
    
    try:
        # Get all runs to see what's been processed
        runs = analytics_storage.list_runs()
        print(f"Found {len(runs)} runs: {runs}")
        
        # Try to query documents directly through DuckDB if possible
        if hasattr(analytics_storage, 'conn'):
            conn = analytics_storage.conn
            
            # Check if documents table exists and what's in it
            try:
                result = conn.execute("SELECT * FROM documents ORDER BY doc_id").fetchall()
                if result:
                    print(f"\nDOCUMENTS TABLE ({len(result)} rows):")
                    for row in result:
                        print(f"  doc_id: {row[0] if isinstance(row, tuple) else row}")
                else:
                    print("\nDOCUMENTS TABLE: Empty or not found")
            except Exception as e:
                print(f"\nError querying documents table: {e}")
            
            # Check elements table for doc_id values
            try:
                result = conn.execute("SELECT DISTINCT doc_id FROM elements ORDER BY doc_id").fetchall()
                if result:
                    print(f"\nELEMENTS TABLE - DISTINCT DOC_IDS ({len(result)} unique):")
                    for row in result:
                        doc_id = row[0] if isinstance(row, tuple) else row
                        print(f"  doc_id: {doc_id}")
                        
                        # Try to get elements for this doc_id to test the method
                        elements = analytics_storage.get_document_elements(doc_id)
                        print(f"    -> {len(elements)} elements found")
                else:
                    print("\nELEMENTS TABLE: No doc_id values found")
            except Exception as e:
                print(f"\nError querying elements table: {e}")
                
            # Show table schema
            try:
                print("\nELEMENTS TABLE SCHEMA:")
                result = conn.execute("DESCRIBE elements").fetchall()
                for row in result:
                    print(f"  {row}")
            except Exception as e:
                print(f"\nError getting elements schema: {e}")
                
        else:
            print("\nNo direct DuckDB connection available")
        
        # Test some specific compound doc_ids that should exist
        test_doc_ids = [
            "test_doc_00.txt::1640995200",
            "test_doc_01.txt::1640995201", 
            "test_doc_02.txt::1640995202",
            "test_doc_00.txt",
            "test_doc_01.txt",
            "test_doc_02.txt"
        ]
        
        print(f"\n=== TESTING SPECIFIC DOC_IDS ===")
        for doc_id in test_doc_ids:
            try:
                elements = analytics_storage.get_document_elements(doc_id)
                print(f"'{doc_id}': {len(elements)} elements")
            except Exception as e:
                print(f"'{doc_id}': ERROR - {e}")
                
    except Exception as e:
        logger.error(f"Error examining analytics storage: {e}")
        import traceback
        traceback.print_exc()

if __name__ == "__main__":
    main()