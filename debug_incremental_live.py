#!/usr/bin/env python3
"""
Live debugging script for incremental processing - shows what's in storage between runs.
"""

import sys
sys.path.insert(0, 'src')

import os
import time
from pathlib import Path
from go_doc_go import Config
from go_doc_go.pipeline.execution_engine import PipelineExecutionEngine
from go_doc_go.config_db.database import PipelineConfigDB

def create_test_files(test_dir: Path, file_count: int = 3):
    """Create test files with known modification times."""
    test_dir.mkdir(parents=True, exist_ok=True)
    
    # Use fixed timestamps for consistent testing
    base_time = 1700000000  # Fixed timestamp
    
    for i in range(file_count):
        test_file = test_dir / f'test_doc_{i:02d}.txt'
        content = f"""Test Document {i}

This is test document number {i} created for incremental processing testing.
It contains some sample text that can be parsed and processed.
The document has multiple lines to create interesting elements.

Created at: {time.ctime(base_time + i)}
Document ID: {i}
"""
        test_file.write_text(content)
        
        # Set modification time
        mod_time = base_time + i
        os.utime(test_file, (mod_time, mod_time))
        print(f"Created: {test_file} (mod_time: {mod_time})")

def check_analytics_storage(analytics_dir: Path):
    """Check what's in analytics storage."""
    print(f"\n=== ANALYTICS STORAGE CHECK: {analytics_dir} ===")
    
    if not analytics_dir.exists():
        print("Analytics directory doesn't exist yet")
        return
        
    # Find all parquet files
    parquet_files = list(analytics_dir.rglob("*.parquet"))
    print(f"Found {len(parquet_files)} parquet files:")
    for f in parquet_files:
        rel_path = f.relative_to(analytics_dir)
        size = f.stat().st_size
        print(f"  {rel_path} ({size} bytes)")
        
    # Try to query documents if we can
    try:
        from go_doc_go.storage_adapters.analytics_parquet import ParquetAnalyticsStorage
        
        analytics_config = {
            'base_path': str(analytics_dir),
            'partitioning': ['run_id', 'doc_type']
        }
        
        storage = ParquetAnalyticsStorage(analytics_config)
        storage.initialize()
        
        # Test specific compound doc_ids
        test_doc_ids = [
            f"{test_dir}/test_doc_00.txt::1700000000",
            f"{test_dir}/test_doc_01.txt::1700000001", 
            f"{test_dir}/test_doc_02.txt::1700000002"
        ]
        
        print("\nTesting document existence:")
        for doc_id in test_doc_ids:
            try:
                elements = storage.get_document_elements(doc_id)
                print(f"  '{doc_id}': {len(elements)} elements")
                if elements:
                    print(f"    First element: {elements[0].get('element_id', 'NO_ID')}")
            except Exception as e:
                print(f"  '{doc_id}': ERROR - {e}")
                
    except Exception as e:
        print(f"Error querying analytics storage: {e}")

def main():
    # Setup paths
    base_dir = Path.cwd()
    test_dir = base_dir / "debug_incremental_data"
    analytics_dir = base_dir / "debug_incremental_analytics" 
    job_db_path = base_dir / "debug_incremental_job.db"
    pipeline_db_path = base_dir / "debug_incremental_pipeline.db"
    
    # Clean up any existing files
    import shutil
    for path in [test_dir, analytics_dir, job_db_path, pipeline_db_path]:
        if path.exists():
            if path.is_dir():
                shutil.rmtree(path)
            else:
                path.unlink()
    
    print("=== LIVE INCREMENTAL PROCESSING DEBUG ===")
    
    # Create test files
    create_test_files(test_dir, 3)
    
    # Create pipeline configuration
    config_db = PipelineConfigDB(str(pipeline_db_path))
    
    from go_doc_go.config_db.models import Pipeline
    import yaml
    
    pipeline_config = {
        "processing_mode": "two_pass",
        "content_sources": [{
            "name": "debug-files",
            "type": "file",
            "base_path": str(test_dir),
            "file_pattern": "*.txt"
        }],
        "storage": {
            "job": {
                "type": "sqlite",
                "path": str(job_db_path)
            },
            "analytics": {
                "type": "parquet",
                "base_path": str(analytics_dir),
                "partitioning": ["run_id", "doc_type"]
            }
        },
        "embedding": {
            "enabled": False
        }
    }
    
    pipeline = Pipeline(
        name="Debug Incremental Test",
        description="Debug incremental processing",
        config_yaml=yaml.dump(pipeline_config)
    )
    
    created_pipeline = config_db.create_pipeline(pipeline)
    pipeline_id = created_pipeline.id
    print(f"\nCreated pipeline: {pipeline.name} (ID: {pipeline_id})")
    
    # Initialize execution engine
    engine = PipelineExecutionEngine(config_db)
    
    # RUN 1: Initial processing
    print(f"\n{'='*50}")
    print("RUN 1: Initial Processing")
    print(f"{'='*50}")
    
    result1 = engine.execute_pipeline(pipeline_id)
    engine.wait_for_completion(result1['run_id'])
    stats1 = engine.get_execution_stats(result1['run_id'])
    print(f"RUN 1 COMPLETE: {stats1}")
    
    # Check analytics storage after run 1
    check_analytics_storage(analytics_dir)
    
    # RUN 2: Immediate rerun (should skip all)
    print(f"\n{'='*50}")
    print("RUN 2: Immediate Rerun - Should skip all")
    print(f"{'='*50}")
    
    result2 = engine.execute_pipeline(pipeline_id)
    engine.wait_for_completion(result2['run_id'])
    stats2 = engine.get_execution_stats(result2['run_id'])
    print(f"RUN 2 COMPLETE: {stats2}")
    
    # Check analytics storage after run 2
    check_analytics_storage(analytics_dir)
    
    # Modify one file
    print(f"\n📝 MODIFYING FILE...")
    test_file_01 = test_dir / "test_doc_01.txt"
    new_content = test_file_01.read_text() + "\n\nMODIFIED CONTENT ADDED!"
    test_file_01.write_text(new_content)
    new_mod_time = int(time.time())
    os.utime(test_file_01, (new_mod_time, new_mod_time))
    print(f"Modified: {test_file_01} (new mod_time: {new_mod_time})")
    
    # RUN 3: After modification (should process only modified file)
    print(f"\n{'='*50}")
    print("RUN 3: After Modification - Should process only 1 file")
    print(f"{'='*50}")
    
    result3 = engine.execute_pipeline(pipeline_id)
    engine.wait_for_completion(result3['run_id'])
    stats3 = engine.get_execution_stats(result3['run_id'])
    print(f"RUN 3 COMPLETE: {stats3}")
    
    # Check analytics storage after run 3
    check_analytics_storage(analytics_dir)
    
    # Summary
    print(f"\n{'='*50}")
    print("SUMMARY")
    print(f"{'='*50}")
    print(f"RUN 1 (Initial): {stats1.get('documents', 0)} documents")
    print(f"RUN 2 (Rerun): {stats2.get('documents', 0)} documents")
    print(f"RUN 3 (Modified): {stats3.get('documents', 0)} documents")
    
    expected_docs = [3, 0, 1]  # Expected pattern
    actual_docs = [stats1.get('documents', 0), stats2.get('documents', 0), stats3.get('documents', 0)]
    
    if actual_docs == expected_docs:
        print("✅ INCREMENTAL PROCESSING WORKING!")
    else:
        print("❌ INCREMENTAL PROCESSING FAILED!")
        print(f"Expected: {expected_docs}")
        print(f"Actual:   {actual_docs}")
    
    print(f"\nFiles preserved for debugging:")
    print(f"  Data: {test_dir}")
    print(f"  Analytics: {analytics_dir}")
    print(f"  Job DB: {job_db_path}")
    print(f"  Pipeline DB: {pipeline_db_path}")

if __name__ == "__main__":
    main()