#!/usr/bin/env python3
"""
Quick script to inspect what's stored in the test incremental analytics parquet files
"""
import duckdb
import os

analytics_path = "./test_incremental_analytics"

if not os.path.exists(analytics_path):
    print(f"Analytics directory does not exist: {analytics_path}")
    exit(1)

elements_path = f"{analytics_path}/elements"
documents_path = f"{analytics_path}/documents" 

if not os.path.exists(elements_path):
    print(f"Elements directory does not exist: {elements_path}")
    exit(1)

if not os.path.exists(documents_path):
    print(f"Documents directory does not exist: {documents_path}")
    exit(1)

# Check what's in the directories
print("=== Elements directory structure ===")
for root, dirs, files in os.walk(elements_path):
    level = root.replace(elements_path, '').count(os.sep)
    indent = ' ' * 2 * level
    print(f'{indent}{os.path.basename(root)}/')
    subindent = ' ' * 2 * (level + 1)
    for file in files:
        if file.endswith('.parquet'):
            print(f'{subindent}{file}')

print("\n=== Documents directory structure ===")
for root, dirs, files in os.walk(documents_path):
    level = root.replace(documents_path, '').count(os.sep)
    indent = ' ' * 2 * level
    print(f'{indent}{os.path.basename(root)}/')
    subindent = ' ' * 2 * (level + 1)
    for file in files:
        if file.endswith('.parquet'):
            print(f'{subindent}{file}')

# Connect to DuckDB and inspect the data
conn = duckdb.connect()

try:
    # Read elements data
    print("\n=== Elements data ===")
    conn.execute(f"CREATE VIEW elements AS SELECT * FROM read_parquet('{elements_path}/**/*.parquet', union_by_name=True)")
    result = conn.execute("SELECT doc_id, element_id, element_type FROM elements ORDER BY doc_id LIMIT 10").fetchall()
    
    print("doc_id | element_id | element_type")
    print("-" * 80)
    for row in result:
        print(f"{row[0]} | {row[1]} | {row[2]}")
    
    # Get unique doc_ids
    doc_ids = conn.execute("SELECT DISTINCT doc_id FROM elements ORDER BY doc_id").fetchall()
    print(f"\nUnique doc_ids in elements ({len(doc_ids)} total):")
    for doc_id in doc_ids:
        print(f"  - {doc_id[0]}")
        
except Exception as e:
    print(f"Error reading elements: {e}")

try:
    # Read documents data
    print("\n=== Documents data ===")
    conn.execute(f"CREATE VIEW documents AS SELECT * FROM read_parquet('{documents_path}/**/*.parquet', union_by_name=True)")
    result = conn.execute("SELECT doc_id, doc_type, source FROM documents ORDER BY doc_id LIMIT 10").fetchall()
    
    print("doc_id | doc_type | source")
    print("-" * 120)
    for row in result:
        print(f"{row[0]} | {row[1]} | {row[2]}")
    
    # Get unique doc_ids
    doc_ids = conn.execute("SELECT DISTINCT doc_id FROM documents ORDER BY doc_id").fetchall()
    print(f"\nUnique doc_ids in documents ({len(doc_ids)} total):")
    for doc_id in doc_ids:
        print(f"  - {doc_id[0]}")
        
except Exception as e:
    print(f"Error reading documents: {e}")

conn.close()