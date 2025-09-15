#!/usr/bin/env python3
"""
Debug what gets passed to parser in real pipeline
"""
import os
import sys
import shutil
import tempfile
import yaml
from pathlib import Path

# Create test environment
test_dir = Path("./debug_parser_input_data")
analytics_dir = Path("./debug_parser_input_analytics") 
job_db = Path("./debug_parser_input_job.db")

# Cleanup
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

# Import after test setup
sys.path.insert(0, 'src')
from go_doc_go.document_parser.text import TextParser

# Monkey patch the text parser to debug what it receives
original_parse = TextParser.parse

def debug_parse(self, doc_content):
    print(f"\n=== DEBUG: TextParser.parse() called ===")
    print(f"doc_content type: {type(doc_content)}")
    print(f"doc_content keys: {list(doc_content.keys())}")
    print(f"doc_content['id']: {doc_content.get('id', 'MISSING')}")
    print(f"doc_content['metadata']: {doc_content.get('metadata', 'MISSING')}")
    if 'metadata' in doc_content and 'doc_id' in doc_content['metadata']:
        print(f"doc_content['metadata']['doc_id']: {doc_content['metadata']['doc_id']}")
    
    # Call the original parse method
    result = original_parse(self, doc_content)
    
    print(f"Generated doc_id in result: {result['document']['doc_id']}")
    print(f"=== END DEBUG ===\n")
    
    return result

# Apply the monkey patch
TextParser.parse = debug_parse

# Now run the pipeline
from go_doc_go import Config
from go_doc_go.processing.two_pass import TwoPassProcessor

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

with tempfile.NamedTemporaryFile(mode='w', suffix='.yaml', delete=False) as f:
    yaml.dump(config_data, f)
    config_file = f.name

try:
    config = Config(config_file)
    processor = TwoPassProcessor(config)
    
    print(f"Starting processing...")
    result = processor.process_local()
    print(f"Processing result: {result}")
    
finally:
    # Cleanup
    os.unlink(config_file)
    for path in [test_dir, analytics_dir, job_db]:
        if path.exists():
            if path.is_dir():
                shutil.rmtree(path)
            else:
                path.unlink()

print(f"Debug complete")