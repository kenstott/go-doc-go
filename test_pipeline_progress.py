#!/usr/bin/env python3
"""
Test script to verify pipeline execution progress monitoring.
"""

import sys
import time
import json
from pathlib import Path
sys.path.insert(0, str(Path(__file__).parent / "src"))

from go_doc_go import Config
from go_doc_go.pipeline.execution_engine import PipelineExecutionEngine
from go_doc_go.config_db import PipelineConfigDB
import tempfile
import yaml

def create_test_pipeline():
    """Create a test pipeline configuration with proper dual storage."""
    pipeline_config = {
        'name': 'Test Progress Pipeline',
        'content_sources': [
            {
                'name': 'test-files',
                'type': 'file',
                'base_path': './test_data',
                'file_pattern': '*.txt'
            }
        ],
        'storage': {
            # Dual storage architecture as required
            'job': {
                'type': 'sqlite',
                'path': './test_pipeline_job.db'
            },
            'analytics': {
                'type': 'parquet',
                'path': './test_pipeline_analytics/'
            }
        },
        'embedding': {
            'enabled': True,
            'model': 'sentence-transformers/all-MiniLM-L6-v2'
        },
        'processing': {
            'worker_mode': 'auto',  # Local processing for testing
            'batch_size': 10
        }
    }
    
    # Create test data directory with sample files
    test_dir = Path('./test_data')
    test_dir.mkdir(exist_ok=True)
    
    # Create sample test files
    for i in range(5):
        test_file = test_dir / f'test_doc_{i:02d}.txt'
        test_file.write_text(f"""
Test Document {i}

This is a test document for verifying progress monitoring.
It contains multiple paragraphs to ensure parsing works correctly.

Section 1: Introduction
This section introduces the test content.

Section 2: Details
This section provides more detailed information about the test.

Section 3: Conclusion
This concludes the test document number {i}.
""")
    
    return pipeline_config

def monitor_execution(engine, run_id):
    """Monitor execution progress in real-time."""
    print(f"\nMonitoring execution: {run_id}")
    print("-" * 60)
    
    last_status = None
    while True:
        status = engine.get_execution_status(run_id)
        if not status:
            print("Execution not found")
            break
        
        # Get current progress from progress monitor
        progress = engine.progress_monitor.get_current_status(run_id)
        
        if progress and progress != last_status:
            print(f"\nStatus: {progress.get('status', 'unknown')}")
            
            stats = progress.get('stats', {})
            if stats:
                # Debug: show all stats keys
                print(f"Stats keys: {list(stats.keys())}")
                print(f"Documents Total: {stats.get('documents_total', 'NOT SET')}")
                print(f"Documents Parsed: {stats.get('documents_parsed', 0)}")
                print(f"Documents Embedded: {stats.get('documents_embedded', 0)}")
                print(f"Parsing Complete: {stats.get('parsing_complete', False)}")
                print(f"Embedding Complete: {stats.get('embedding_complete', False)}")
                print(f"Elements: {stats.get('elements', 0)}")
                print(f"Relationships: {stats.get('relationships', 0)}")
            
            last_status = progress
        
        # Check if execution is complete
        if status.get('status') in ['completed', 'failed', 'cancelled']:
            print(f"\n=== Execution {status.get('status').upper()} ===")
            if status.get('status') == 'completed':
                print(f"Final stats: {json.dumps(stats, indent=2)}")
            break
        
        time.sleep(0.5)  # Poll every 500ms
    
    # Get execution logs
    logs = engine.get_execution_logs(run_id)
    if logs.get('logs'):
        print("\n=== Recent Log Entries ===")
        for log_entry in logs['logs'][-10:]:  # Last 10 entries
            print(f"[{log_entry['level']}] {log_entry['message']}")

def main():
    """Main test function."""
    print("Testing Pipeline Progress Monitoring")
    print("=" * 60)
    
    # Create test pipeline
    pipeline_config = create_test_pipeline()
    
    # Initialize engine
    config = Config('./config.yaml')
    engine = PipelineExecutionEngine(config, db_path='test_pipeline.db')
    
    # Create pipeline in database
    from go_doc_go.config_db import Pipeline
    db = PipelineConfigDB('test_pipeline.db')
    
    # Create Pipeline object
    pipeline = Pipeline(
        name=pipeline_config['name'],
        config_yaml=yaml.dump(pipeline_config),
        description='Test pipeline for progress monitoring',
        is_active=True
    )
    
    # Save to database
    pipeline = db.create_pipeline(pipeline)
    
    print(f"Created pipeline: {pipeline.name} (ID: {pipeline.id})")
    
    # Execute pipeline
    print("\nStarting pipeline execution...")
    execution = engine.execute_pipeline(
        pipeline_id=pipeline.id,
        execution_params={
            'worker_count': 1,
            'documents_total': 5  # We know we have 5 test documents
        }
    )
    
    print(f"Execution started: {execution.run_id}")
    
    # Monitor execution
    monitor_execution(engine, execution.run_id)
    
    # Clean up
    import shutil
    if Path('./test_data').exists():
        shutil.rmtree('./test_data')
    if Path('./test_pipeline_analytics').exists():
        shutil.rmtree('./test_pipeline_analytics')
    for db_file in ['test_pipeline.db', 'test_pipeline_job.db']:
        if Path(f'./{db_file}').exists():
            Path(f'./{db_file}').unlink()
    
    print("\nTest complete!")

if __name__ == '__main__':
    main()