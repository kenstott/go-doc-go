#!/usr/bin/env python3
"""
Test script to verify incremental pipeline processing.
"""

import sys
import time
import tempfile
import os
from pathlib import Path
sys.path.insert(0, str(Path(__file__).parent / "src"))

from go_doc_go import Config
from go_doc_go.pipeline.execution_engine import PipelineExecutionEngine
from go_doc_go.config_db import PipelineConfigDB, Pipeline
import yaml
import shutil

def create_test_files(test_dir: Path, file_count: int = 3):
    """Create test files with known modification times."""
    test_dir.mkdir(exist_ok=True)
    
    files_created = []
    base_time = 1700000000  # Fixed base timestamp
    
    for i in range(file_count):
        test_file = test_dir / f'test_doc_{i:02d}.txt'
        content = f"""Test Document {i}

This is test document number {i} for verifying incremental processing.
It contains sample content to ensure parsing works correctly.

Original creation time: {base_time + i}
"""
        test_file.write_text(content)
        
        # Set specific modification time
        mod_time = base_time + i
        os.utime(test_file, (mod_time, mod_time))
        
        files_created.append({
            'path': str(test_file),
            'mod_time': mod_time
        })
        
    return files_created

def modify_test_file(file_path: str, new_content: str):
    """Modify a test file and update its timestamp."""
    with open(file_path, 'w') as f:
        f.write(new_content)
    
    # Set new modification time (current time)
    new_time = int(time.time())
    os.utime(file_path, (new_time, new_time))
    return new_time

def create_test_pipeline(test_dir: Path):
    """Create a test pipeline configuration with dual storage."""
    pipeline_config = {
        'name': 'Incremental Processing Test',
        'content_sources': [
            {
                'name': 'test-files',
                'type': 'file',
                'base_path': str(test_dir),
                'file_pattern': '*.txt'
            }
        ],
        'storage': {
            # Dual storage architecture as required
            'job': {
                'type': 'sqlite',
                'path': './test_incremental_job.db'
            },
            'analytics': {
                'type': 'parquet', 
                'path': './test_incremental_analytics/'
            }
        },
        'embedding': {
            'enabled': True,
            'model': 'sentence-transformers/all-MiniLM-L6-v2'
        },
        'processing': {
            'mode': 'two_pass',
            'batch_size': 10
        }
    }
    
    return pipeline_config

def run_pipeline_and_check_results(engine, pipeline_id, test_name, expected_queued=None):
    """Run pipeline and return results."""
    print(f"\n=== {test_name} ===")
    
    execution = engine.execute_pipeline(pipeline_id)
    print(f"Execution started: {execution.run_id}")
    
    # Monitor execution briefly
    max_wait = 30  # seconds
    waited = 0
    
    while waited < max_wait:
        status = engine.get_execution_status(execution.run_id)
        if not status:
            break
            
        if status.get('status') in ['completed', 'failed', 'cancelled']:
            print(f"Execution {status.get('status')}: {execution.run_id}")
            
            # Get final progress
            progress = engine.progress_monitor.get_current_status(execution.run_id)
            if progress:
                stats = progress.get('stats', {})
                print(f"Documents processed: {stats.get('documents', 0)}")
                print(f"Elements created: {stats.get('elements', 0)}")
                print(f"Embeddings generated: {stats.get('embeddings_generated', 0)}")
            
            return status, progress
            
        time.sleep(1)
        waited += 1
    
    print(f"Execution timed out after {max_wait}s")
    return None, None

def main():
    """Main test function."""
    print("Testing Incremental Pipeline Processing")
    print("=" * 50)
    
    # Create test directory and files
    test_dir = Path('./test_incremental_data')
    files_created = create_test_files(test_dir, file_count=3)
    
    print(f"Created {len(files_created)} test files:")
    for f in files_created:
        print(f"  - {f['path']} (mod_time: {f['mod_time']})")
    
    try:
        # Initialize engine
        config = Config('./config.yaml')
        engine = PipelineExecutionEngine(config, db_path='test_incremental_pipeline.db')
        
        # Create pipeline in database
        pipeline_config = create_test_pipeline(test_dir)
        db = PipelineConfigDB('test_incremental_pipeline.db')
        
        pipeline = Pipeline(
            name=pipeline_config['name'],
            config_yaml=yaml.dump(pipeline_config),
            description='Test pipeline for incremental processing',
            is_active=True
        )
        
        pipeline = db.create_pipeline(pipeline)
        print(f"Created pipeline: {pipeline.name} (ID: {pipeline.id})")
        
        # Run 1: Initial processing (should process all 3 files)
        print(f"\n{'='*50}")
        print("RUN 1: Initial Processing - All files should be queued")
        print(f"{'='*50}")
        
        status1, progress1 = run_pipeline_and_check_results(
            engine, pipeline.id, "Initial Run", expected_queued=3
        )
        
        if not status1:
            print("❌ Initial run failed or timed out")
            return
            
        # Run 2: Immediate rerun (should skip all files)
        print(f"\n{'='*50}")
        print("RUN 2: Immediate Rerun - All files should be skipped")
        print(f"{'='*50}")
        
        status2, progress2 = run_pipeline_and_check_results(
            engine, pipeline.id, "Immediate Rerun", expected_queued=0
        )
        
        # Modify one file
        modified_file = files_created[1]['path']  # Modify second file
        new_content = f"""MODIFIED Test Document 1

This file has been MODIFIED to test incremental processing.
The content is now different and should trigger reprocessing.

Modified at: {int(time.time())}
"""
        
        new_mod_time = modify_test_file(modified_file, new_content)
        print(f"\n📝 Modified file: {modified_file}")
        print(f"   New modification time: {new_mod_time}")
        
        # Run 3: After modification (should process only 1 modified file)
        print(f"\n{'='*50}")
        print("RUN 3: After Modification - Only modified file should be queued")
        print(f"{'='*50}")
        
        status3, progress3 = run_pipeline_and_check_results(
            engine, pipeline.id, "After Modification", expected_queued=1
        )
        
        # Summary
        print(f"\n{'='*50}")
        print("INCREMENTAL PROCESSING TEST SUMMARY")
        print(f"{'='*50}")
        
        print(f"✅ Run 1 (Initial): {status1['status'] if status1 else 'FAILED'}")
        if progress1:
            stats1 = progress1.get('stats', {})
            print(f"   Documents: {stats1.get('documents', 0)}, Elements: {stats1.get('elements', 0)}")
            
        print(f"✅ Run 2 (Rerun): {status2['status'] if status2 else 'FAILED'}")  
        if progress2:
            stats2 = progress2.get('stats', {})
            print(f"   Documents: {stats2.get('documents', 0)}, Elements: {stats2.get('elements', 0)}")
            
        print(f"✅ Run 3 (Modified): {status3['status'] if status3 else 'FAILED'}")
        if progress3:
            stats3 = progress3.get('stats', {})
            print(f"   Documents: {stats3.get('documents', 0)}, Elements: {stats3.get('elements', 0)}")
            
        # Validate expectations
        success = True
        if progress1 and progress1.get('stats', {}).get('documents', 0) != 3:
            print("❌ Expected 3 documents processed in initial run")
            success = False
            
        if progress2 and progress2.get('stats', {}).get('documents', 0) != 0:
            print("❌ Expected 0 documents processed in immediate rerun")  
            success = False
            
        if progress3 and progress3.get('stats', {}).get('documents', 0) != 1:
            print("❌ Expected 1 document processed after modification")
            success = False
            
        if success:
            print("\n🎉 INCREMENTAL PROCESSING TEST PASSED!")
            print("✅ Files are only processed when new or modified")
            print("✅ Unchanged files are properly skipped")
        else:
            print("\n❌ INCREMENTAL PROCESSING TEST FAILED!")
            print("   Check implementation for issues")
            
    finally:
        # Cleanup
        cleanup_paths = [
            './test_incremental_data',
            './test_incremental_analytics',
            './test_incremental_job.db', 
            './test_incremental_pipeline.db'
        ]
        
        for path in cleanup_paths:
            if Path(path).exists():
                if Path(path).is_dir():
                    shutil.rmtree(path)
                else:
                    Path(path).unlink()
                print(f"Cleaned up: {path}")

if __name__ == '__main__':
    main()