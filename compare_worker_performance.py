#!/usr/bin/env python3
"""
Compare worker CLI performance with and without Go modules.
Runs the test multiple times and reports statistics.
"""

import subprocess
import time
import statistics
import os
import json
from pathlib import Path

def run_worker_test(use_go_modules: bool, verbose: bool = False) -> float:
    """Run worker CLI test and return execution time."""
    env = os.environ.copy()
    env['USE_GO_MODULES'] = 'true' if use_go_modules else 'false'

    cmd = [
        'python', '-m', 'pytest',
        'test_worker_cli.py::TestWorkerCLI::test_worker_cli_process_documents',
        '-v' if verbose else '-q',
        '--tb=short'
    ]

    start_time = time.time()
    result = subprocess.run(
        cmd,
        capture_output=True,
        text=True,
        env=env
    )
    elapsed = time.time() - start_time

    if result.returncode != 0:
        print(f"Test failed: {result.stderr}")
        return -1

    return elapsed

def main():
    print("="*80)
    print("WORKER CLI PERFORMANCE COMPARISON")
    print("="*80)
    print("\nNote: Each test processes all documents in tests/assets/")
    print("This takes approximately 6-7 minutes per run.")

    # Skip warmup for long-running tests
    print("\nSkipping warmup due to test duration...")

    # Number of test runs - these take 6-7 minutes each
    num_runs = 1  # Just one run each since they're long

    print(f"\nRunning {num_runs} tests for each configuration...\n")

    # Test without Go modules
    print("Testing WITHOUT Go modules (Python parsers):")
    python_times = []
    for i in range(num_runs):
        print(f"  Run {i+1}/{num_runs}...", end=" ", flush=True)
        elapsed = run_worker_test(False, False)
        if elapsed > 0:
            python_times.append(elapsed)
            print(f"{elapsed:.2f}s")
        else:
            print("FAILED")

    # Test with Go modules
    print("\nTesting WITH Go modules (Go parsers):")
    go_times = []
    for i in range(num_runs):
        print(f"  Run {i+1}/{num_runs}...", end=" ", flush=True)
        elapsed = run_worker_test(True, False)
        if elapsed > 0:
            go_times.append(elapsed)
            print(f"{elapsed:.2f}s")
        else:
            print("FAILED")

    # Calculate statistics
    print("\n" + "="*80)
    print("RESULTS")
    print("="*80)

    if python_times:
        print(f"\nPython Parsers (USE_GO_MODULES=false):")
        print(f"  Runs:    {python_times}")
        print(f"  Average: {statistics.mean(python_times):.2f}s")
        print(f"  Min:     {min(python_times):.2f}s")
        print(f"  Max:     {max(python_times):.2f}s")
        if len(python_times) > 1:
            print(f"  StdDev:  {statistics.stdev(python_times):.2f}s")

    if go_times:
        print(f"\nGo Parsers (USE_GO_MODULES=true):")
        print(f"  Runs:    {go_times}")
        print(f"  Average: {statistics.mean(go_times):.2f}s")
        print(f"  Min:     {min(go_times):.2f}s")
        print(f"  Max:     {max(go_times):.2f}s")
        if len(go_times) > 1:
            print(f"  StdDev:  {statistics.stdev(go_times):.2f}s")

    if python_times and go_times:
        avg_python = statistics.mean(python_times)
        avg_go = statistics.mean(go_times)
        speedup = avg_python / avg_go

        print("\n" + "-"*80)
        print("PERFORMANCE COMPARISON:")
        print(f"  Average Python: {avg_python:.2f}s")
        print(f"  Average Go:     {avg_go:.2f}s")
        print(f"  Speedup:        {speedup:.2f}x")

        if speedup > 1:
            print(f"\n  🚀 Go parsers are {speedup:.2f}x faster!")
            time_saved = avg_python - avg_go
            percent_faster = ((avg_python - avg_go) / avg_python) * 100
            print(f"     Time saved: {time_saved:.2f}s ({percent_faster:.1f}% faster)")
        else:
            print(f"\n  Python parsers are {1/speedup:.2f}x faster")

    # Analyze what documents were processed
    print("\n" + "-"*80)
    print("DOCUMENT PROCESSING DETAILS:")

    # Check the test output for document counts
    test_output_dir = Path("tests/test_output/worker_cli_test")
    if test_output_dir.exists():
        # Count files in analytics output
        analytics_dir = test_output_dir / "analytics-output"
        if analytics_dir.exists():
            parquet_files = list(analytics_dir.rglob("*.parquet"))
            print(f"  Output parquet files: {len(parquet_files)}")

        # Check source documents
        assets_dir = Path("tests/assets")
        if assets_dir.exists():
            doc_types = {}
            for file in assets_dir.iterdir():
                if file.is_file():
                    ext = file.suffix.lower()
                    doc_types[ext] = doc_types.get(ext, 0) + 1

            print(f"  Source documents by type:")
            for ext, count in sorted(doc_types.items()):
                print(f"    {ext}: {count} files")
            print(f"  Total documents: {sum(doc_types.values())}")

if __name__ == "__main__":
    os.chdir("tests")  # Change to tests directory
    main()