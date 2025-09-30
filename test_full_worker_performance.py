#!/usr/bin/env python3
"""
Full worker performance test - processes ALL documents in tests/assets.
Compares Python vs Go parser performance.
"""

import subprocess
import time
import os
import shutil
from pathlib import Path
import tempfile
import json

def setup_test_config(test_dir: Path, use_go: bool) -> Path:
    """Create test configuration file."""
    config = {
        "content_source": {
            "type": "file",
            "base_path": "tests/assets",
            "file_pattern": "**/*",
            "recursive": True,
            "watch_for_changes": False
        },
        "document_database": {
            "type": "sqlite",
            "connection_string": str(test_dir / "documents.db")
        },
        "analytics_output": {
            "parquet_dir": str(test_dir / "analytics-output")
        },
        "job_control": {
            "type": "sqlite",
            "connection_string": str(test_dir / "job_queue.db"),
            "batch_size": 100
        }
    }

    config_path = test_dir / "config.json"
    with open(config_path, 'w') as f:
        json.dump(config, f, indent=2)

    return config_path

def count_documents() -> int:
    """Count total documents in tests/assets."""
    assets_dir = Path("tests/assets")
    count = 0
    for file in assets_dir.rglob("*"):
        if file.is_file():
            count += 1
    return count

def run_worker_test(use_go: bool, max_docs: int = None) -> tuple:
    """Run worker and return timing and results."""

    # Create temporary test directory
    test_dir = Path(tempfile.mkdtemp(prefix="worker_test_"))

    try:
        # Setup configuration
        config_path = setup_test_config(test_dir, use_go)

        # Set environment
        env = os.environ.copy()
        env['USE_GO_MODULES'] = 'true' if use_go else 'false'
        env['PYTHONPATH'] = 'src'

        # Build command
        cmd = [
            "python", "-m", "go_doc_go.cli.worker",
            "--config", str(config_path),
        ]

        if max_docs:
            cmd.extend(["--max-documents", str(max_docs)])

        print(f"  Running worker with {'Go' if use_go else 'Python'} parsers...")
        if max_docs:
            print(f"  Processing up to {max_docs} documents")
        else:
            print(f"  Processing all documents until queue is empty")

        # Run worker
        start_time = time.time()
        result = subprocess.run(
            cmd,
            capture_output=True,
            text=True,
            env=env,
            timeout=600  # 10 minute timeout
        )
        elapsed = time.time() - start_time

        # Count processed documents
        analytics_dir = test_dir / "analytics-output"
        processed_docs = 0
        if analytics_dir.exists():
            # Count unique documents in the parquet files
            doc_parquet = list(analytics_dir.rglob("documents*.parquet"))
            processed_docs = len(doc_parquet)

        return elapsed, processed_docs, result.returncode == 0

    except subprocess.TimeoutExpired:
        elapsed = time.time() - start_time
        return elapsed, -1, False

    finally:
        # Cleanup
        if test_dir.exists():
            shutil.rmtree(test_dir)

def main():
    print("="*80)
    print("FULL WORKER PERFORMANCE TEST")
    print("="*80)

    # Count total documents
    total_docs = count_documents()
    print(f"\nTotal documents in tests/assets: {total_docs}")
    print()

    # Test with limited documents first (warmup)
    print("WARMUP TEST (5 documents):")
    print("-"*40)

    py_time, py_docs, py_success = run_worker_test(False, max_docs=5)
    print(f"  Python: {py_time:.2f}s, processed {py_docs} docs, success={py_success}")

    go_time, go_docs, go_success = run_worker_test(True, max_docs=5)
    print(f"  Go:     {go_time:.2f}s, processed {go_docs} docs, success={go_success}")

    if py_time > 0 and go_time > 0:
        speedup = py_time / go_time
        print(f"  Speedup: {speedup:.2f}x")

    # Full test
    print(f"\nFULL TEST (ALL {total_docs} documents):")
    print("-"*40)
    print("This will take several minutes...\n")

    # Python parsers
    print("Testing with Python parsers...")
    py_time_full, py_docs_full, py_success_full = run_worker_test(False, max_docs=None)
    print(f"  Time: {py_time_full:.2f}s")
    print(f"  Documents processed: {py_docs_full}")
    print(f"  Success: {py_success_full}")

    # Go parsers
    print("\nTesting with Go parsers...")
    go_time_full, go_docs_full, go_success_full = run_worker_test(True, max_docs=None)
    print(f"  Time: {go_time_full:.2f}s")
    print(f"  Documents processed: {go_docs_full}")
    print(f"  Success: {go_success_full}")

    # Summary
    print("\n" + "="*80)
    print("SUMMARY")
    print("="*80)

    print("\nWarmup (5 documents):")
    print(f"  Python: {py_time:.2f}s")
    print(f"  Go:     {go_time:.2f}s")
    if py_time > 0 and go_time > 0:
        speedup = py_time / go_time
        print(f"  Speedup: {speedup:.2f}x")

    print(f"\nFull Test ({total_docs} documents):")
    print(f"  Python: {py_time_full:.2f}s")
    print(f"  Go:     {go_time_full:.2f}s")
    if py_time_full > 0 and go_time_full > 0:
        speedup_full = py_time_full / go_time_full
        print(f"  Speedup: {speedup_full:.2f}x")

        time_saved = py_time_full - go_time_full
        percent_faster = ((py_time_full - go_time_full) / py_time_full) * 100

        if speedup_full > 1:
            print(f"\n🚀 Go parsers are {speedup_full:.2f}x faster!")
            print(f"   Time saved: {time_saved:.2f}s ({percent_faster:.1f}% faster)")
        else:
            print(f"\nPython parsers are {1/speedup_full:.2f}x faster")

    # Per-document metrics
    if py_docs_full > 0 and go_docs_full > 0:
        print(f"\nPer-document processing time:")
        print(f"  Python: {py_time_full/py_docs_full*1000:.2f}ms per document")
        print(f"  Go:     {go_time_full/go_docs_full*1000:.2f}ms per document")

if __name__ == "__main__":
    main()