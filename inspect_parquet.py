#!/usr/bin/env python3
"""
Direct inspection of parquet files to see what doc_ids are stored.
"""

import sys
sys.path.insert(0, 'src')

import duckdb
import os
from pathlib import Path

def main():
    data_lake = Path("./data-lake")
    elements_dir = data_lake / "elements"
    
    if not elements_dir.exists():
        print(f"Elements directory does not exist: {elements_dir}")
        return
    
    # Find elements parquet files for our specific run
    run_id = "d751713988987e9331980363e24189ce"
    elements_files = list(elements_dir.rglob(f"**/run_id={run_id}/*.parquet"))
    
    print(f"Found {len(elements_files)} elements files for run_id {run_id}:")
    for f in elements_files[:5]:
        rel_path = f.relative_to(data_lake)
        size = f.stat().st_size
        print(f"  {rel_path} ({size} bytes)")
    
    if elements_files:
        # Use DuckDB to directly query a specific parquet file
        conn = duckdb.connect(':memory:')
        
        # Test one file at a time to avoid schema issues
        for elements_file in elements_files[:3]:  # Test first 3 files
            print(f"\n=== INSPECTING: {elements_file.name} ===")
            try:
                # Query the specific file
                conn.execute(f"CREATE OR REPLACE VIEW test_elements AS SELECT * FROM read_parquet('{elements_file}')")
                
                # Get all doc_ids from this file
                results = conn.execute("SELECT DISTINCT doc_id FROM test_elements").fetchall()
                print(f"Document IDs in {elements_file.name}:")
                for row in results:
                    doc_id = row[0]
                    print(f"  '{doc_id}'")
                
                # Get a sample element to see the structure
                sample = conn.execute("SELECT * FROM test_elements LIMIT 1").fetchall()
                if sample:
                    columns = [desc[0] for desc in conn.description]
                    print(f"Sample element structure (columns: {len(columns)}):")
                    print(f"  Columns: {columns}")
                    element = dict(zip(columns, sample[0]))
                    for key, value in element.items():
                        if key in ['content_location', 'metadata']:
                            # Show full content for these important fields
                            print(f"    {key}: {repr(value)}")
                        elif isinstance(value, str) and len(value) > 50:
                            value = value[:47] + "..."
                            print(f"    {key}: {repr(value)}")
                        else:
                            print(f"    {key}: {repr(value)}")
                
            except Exception as e:
                print(f"Error querying {elements_file.name}: {e}")
        
        conn.close()

if __name__ == "__main__":
    main()