#!/usr/bin/env python3
"""
Debug incremental processing - simplified version to capture the issue
"""
import os
import shutil
import tempfile
from pathlib import Path

# Create test data
test_dir = Path("./debug_test_data")
test_dir.mkdir(exist_ok=True)

# Create a simple test file
test_file = test_dir / "test.txt"
test_file.write_text("Test document content for debugging")

# Set file modification time to a known value 
os.utime(test_file, (1600000000, 1600000000))  # Both access and modify time

print(f"Created test file: {test_file}")
print(f"File mod time: {test_file.stat().st_mtime}")

# Test document ID that should be generated
expected_doc_id = f"{test_file.absolute()}::{int(test_file.stat().st_mtime)}"
print(f"Expected compound doc_id: {expected_doc_id}")

# Run the processing once
print("\n=== Running initial processing ===")
import subprocess
import sys

# Set up environment for test
env = os.environ.copy()
env['PYTHONPATH'] = 'src'

result = subprocess.run([
    sys.executable, '-c', f'''
import sys
sys.path.insert(0, "src")
from go_doc_go import Config
from go_doc_go.main import ingest_documents

config_data = {{
    "content_sources": [{{
        "name": "debug-test",
        "type": "file",
        "base_path": "{test_dir.absolute()}",
        "file_pattern": "*.txt"
    }}],
    "storage": {{
        "job": {{"backend": "sqlite", "path": "./debug_job.db"}},
        "analytics": {{"backend": "parquet", "path": "./debug_analytics/"}}
    }},
    "processing": {{"worker_mode": "two_pass"}},
    "embedding": {{"enabled": True}},
    "relationship_detection": {{"enabled": True}}
}}

config = Config(config_data)
result = ingest_documents(config)
print(f"Processing result: {{result}}")
'''
], capture_output=True, text=True, env=env)

print(f"Return code: {result.returncode}")
if result.stdout:
    print(f"STDOUT: {result.stdout}")
if result.stderr:
    print(f"STDERR: {result.stderr}")

# Now check what was stored
print("\n=== Checking stored data ===")
analytics_path = "./debug_analytics"
if os.path.exists(analytics_path):
    print("Analytics directory exists")
    
    # Check elements
    elements_path = f"{analytics_path}/elements"
    if os.path.exists(elements_path):
        print("Elements directory exists")
        
        # Inspect with DuckDB
        try:
            import duckdb
            conn = duckdb.connect()
            conn.execute(f"CREATE VIEW elements AS SELECT * FROM read_parquet('{elements_path}/**/*.parquet', union_by_name=True)")
            doc_ids = conn.execute("SELECT DISTINCT doc_id FROM elements").fetchall()
            print(f"Stored doc_ids in elements:")
            for doc_id in doc_ids:
                print(f"  - {doc_id[0]}")
            conn.close()
        except Exception as e:
            print(f"Error reading elements: {e}")
    else:
        print("No elements directory found")
else:
    print("No analytics directory found")

# Run again to test incremental processing
print(f"\n=== Running again to test incremental processing ===")
print(f"Should find existing document: {expected_doc_id}")

result = subprocess.run([
    sys.executable, '-c', f'''
import sys
sys.path.insert(0, "src")
from go_doc_go import Config
from go_doc_go.main import ingest_documents

config_data = {{
    "content_sources": [{{
        "name": "debug-test",
        "type": "file", 
        "base_path": "{test_dir.absolute()}",
        "file_pattern": "*.txt"
    }}],
    "storage": {{
        "job": {{"backend": "sqlite", "path": "./debug_job.db"}},
        "analytics": {{"backend": "parquet", "path": "./debug_analytics/"}}
    }},
    "processing": {{"worker_mode": "two_pass"}},
    "embedding": {{"enabled": True}},
    "relationship_detection": {{"enabled": True}}
}}

config = Config(config_data)
result = ingest_documents(config)
print(f"Second processing result: {{result}}")
'''
], capture_output=True, text=True, env=env)

print(f"Return code: {result.returncode}")
if result.stdout:
    print(f"STDOUT: {result.stdout}")
if result.stderr:
    print(f"STDERR: {result.stderr}")

# Cleanup
if os.path.exists("debug_job.db"):
    os.remove("debug_job.db")
if os.path.exists("debug_analytics"):
    shutil.rmtree("debug_analytics")
if os.path.exists(test_dir):
    shutil.rmtree(test_dir)

print("\nCleanup complete")