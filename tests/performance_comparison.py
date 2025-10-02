#!/usr/bin/env python3
"""
Performance comparison test between Python and Go workers.
Runs each worker for a fixed time and collects metrics.
"""

import os
import sys
import time
import subprocess
import yaml
import shutil
import signal
from pathlib import Path
from datetime import datetime

def setup_test_environment(test_name):
    """Create isolated test environment."""
    test_dir = Path(__file__).parent / "test_output" / test_name

    # Clean up existing
    if test_dir.exists():
        shutil.rmtree(test_dir, ignore_errors=True)

    test_dir.mkdir(parents=True, exist_ok=True)
    (test_dir / "logs").mkdir(parents=True)

    # Load base config
    config_path = Path(__file__).parent / "config.sqlite.yaml"
    with open(config_path) as f:
        config = yaml.safe_load(f)

    # Set paths
    job_db_path = test_dir / "job_queue.db"
    config['processing']['job_control']['path'] = str(job_db_path)

    analytics_path = test_dir / "analytics-output"
    config['analytics']['outputs'][0]['path'] = str(analytics_path)
    config['logging']['file'] = str(test_dir / "logs" / "worker.log")

    assets_dir = Path(__file__).parent / "assets"
    config['content_sources'][0]['base_path'] = str(assets_dir)

    # Write config
    isolated_config = test_dir / "worker_config.yaml"
    with open(isolated_config, 'w') as f:
        yaml.dump(config, f, default_flow_style=False)

    return test_dir, isolated_config

def run_python_worker(config_path, duration_seconds, num_workers=1):
    """Run Python worker for specified duration."""
    print(f"\n{'='*60}")
    print(f"PYTHON WORKER TEST ({num_workers} processes)")
    print(f"{'='*60}")
    print(f"Duration: {duration_seconds} seconds")
    print(f"Config: {config_path}")

    cmd = [
        "python", "-m", "go_doc_go.cli.worker",
        "--config", str(config_path),
        "--max-documents", "10000",
        "--workers", str(num_workers),
    ]

    print(f"Command: {' '.join(cmd)}")

    start_time = time.time()

    env = os.environ.copy()
    env['USE_GO_MODULES'] = 'false'
    env['PYTHONPATH'] = 'src'
    env['NUM_WORKERS'] = str(num_workers)

    process = subprocess.Popen(
        cmd,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
        text=True,
        env=env,
        cwd=Path(__file__).parent.parent
    )

    print(f"Started Python worker (PID: {process.pid}) at {datetime.now()}")

    # Let it run for the specified duration
    try:
        time.sleep(duration_seconds)
    except KeyboardInterrupt:
        print("\nInterrupted by user")

    # Terminate gracefully
    print(f"Sending SIGTERM to worker...")
    process.terminate()

    try:
        stdout, _ = process.communicate(timeout=10)
    except subprocess.TimeoutExpired:
        print("Worker didn't stop gracefully, sending SIGKILL...")
        process.kill()
        stdout, _ = process.communicate()

    end_time = time.time()
    elapsed = end_time - start_time

    print(f"Worker stopped. Elapsed time: {elapsed:.2f} seconds")

    return elapsed, stdout

def run_go_worker(config_path, duration_seconds, num_workers=10):
    """Run Go worker for specified duration."""
    print(f"\n{'='*60}")
    print(f"GO WORKER TEST ({num_workers} goroutines)")
    print(f"{'='*60}")
    print(f"Duration: {duration_seconds} seconds")
    print(f"Config: {config_path}")

    cmd = [
        "python", "-m", "go_doc_go.cli.worker",
        "--config", str(config_path),
        "--max-documents", "10000",
        "--workers", str(num_workers),
    ]

    print(f"Command: {' '.join(cmd)}")

    start_time = time.time()

    env = os.environ.copy()
    env['USE_GO_MODULES'] = 'true'
    env['PYTHONPATH'] = 'src'

    process = subprocess.Popen(
        cmd,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
        text=True,
        env=env,
        cwd=Path(__file__).parent.parent
    )

    print(f"Started Go worker (PID: {process.pid}) at {datetime.now()}")

    # Let it run for the specified duration
    try:
        time.sleep(duration_seconds)
    except KeyboardInterrupt:
        print("\nInterrupted by user")

    # Terminate gracefully
    print(f"Sending SIGTERM to worker...")
    process.terminate()

    try:
        stdout, _ = process.communicate(timeout=10)
    except subprocess.TimeoutExpired:
        print("Worker didn't stop gracefully, sending SIGKILL...")
        process.kill()
        stdout, _ = process.communicate()

    end_time = time.time()
    elapsed = end_time - start_time

    print(f"Worker stopped. Elapsed time: {elapsed:.2f} seconds")

    return elapsed, stdout

def collect_metrics(test_dir):
    """Collect performance metrics from test output."""
    import sqlite3

    metrics = {
        'documents_discovered': 0,
        'documents_pending': 0,
        'documents_completed': 0,
        'documents_failed': 0,
        'total_embeddings': 0,
        'total_elements': 0,
        'total_relationships': 0,
    }

    # Get document counts from job queue DB
    job_db = test_dir / "job_queue.db"
    if job_db.exists():
        try:
            conn = sqlite3.connect(str(job_db))
            cursor = conn.cursor()

            # Total documents
            cursor.execute("SELECT COUNT(*) FROM documents")
            metrics['documents_discovered'] = cursor.fetchone()[0]

            # By status
            cursor.execute("SELECT status, COUNT(*) FROM documents GROUP BY status")
            for status, count in cursor.fetchall():
                if status == 'pending':
                    metrics['documents_pending'] = count
                elif status == 'completed':
                    metrics['documents_completed'] = count
                elif status == 'failed':
                    metrics['documents_failed'] = count

            conn.close()
        except Exception as e:
            print(f"Warning: Could not read job queue DB: {e}")

    # Count parquet records using duckdb
    analytics_dir = test_dir / "analytics-output"
    if analytics_dir.exists():
        try:
            # Count embeddings
            embeddings_pattern = str(analytics_dir / "embeddings" / "**" / "*.parquet")
            result = subprocess.run(
                ["duckdb", "-c", f"SELECT COUNT(*) FROM read_parquet('{embeddings_pattern}')"],
                capture_output=True,
                text=True,
                timeout=5
            )
            if result.returncode == 0:
                lines = result.stdout.strip().split('\n')
                for line in lines:
                    if line.strip().isdigit():
                        metrics['total_embeddings'] = int(line.strip())
                        break

            # Count elements
            elements_pattern = str(analytics_dir / "elements" / "**" / "*.parquet")
            result = subprocess.run(
                ["duckdb", "-c", f"SELECT COUNT(*) FROM read_parquet('{elements_pattern}')"],
                capture_output=True,
                text=True,
                timeout=5
            )
            if result.returncode == 0:
                lines = result.stdout.strip().split('\n')
                for line in lines:
                    if line.strip().isdigit():
                        metrics['total_elements'] = int(line.strip())
                        break

            # Count relationships
            relationships_pattern = str(analytics_dir / "relationships" / "**" / "*.parquet")
            result = subprocess.run(
                ["duckdb", "-c", f"SELECT COUNT(*) FROM read_parquet('{relationships_pattern}')"],
                capture_output=True,
                text=True,
                timeout=5
            )
            if result.returncode == 0:
                lines = result.stdout.strip().split('\n')
                for line in lines:
                    if line.strip().isdigit():
                        metrics['total_relationships'] = int(line.strip())
                        break

        except Exception as e:
            print(f"Warning: Could not read analytics data: {e}")

    return metrics

def main():
    """Run performance comparison."""
    test_duration = 600  # 10 minutes per test
    num_workers = 10

    print("="*60)
    print("PYTHON vs GO WORKER PERFORMANCE COMPARISON")
    print("="*60)
    print(f"Test duration: {test_duration} seconds (10 minutes)")
    print(f"Number of workers: {num_workers}")
    print(f"  - Python: {num_workers} independent processes")
    print(f"  - Go: {num_workers} goroutines in single process")
    print()

    # Test 1: Python worker with multiple processes
    print("\n" + "="*60)
    print(f"TEST 1: PYTHON WORKER ({num_workers} processes)")
    print("="*60)

    python_test_dir, python_config = setup_test_environment("python_worker_perf")
    python_elapsed, python_output = run_python_worker(python_config, test_duration, num_workers=num_workers)
    python_metrics = collect_metrics(python_test_dir)

    print("\nPython worker metrics:")
    for key, value in python_metrics.items():
        print(f"  {key}: {value}")

    # Test 2: Go worker with goroutines
    print("\n" + "="*60)
    print(f"TEST 2: GO WORKER ({num_workers} goroutines)")
    print("="*60)

    go_test_dir, go_config = setup_test_environment("go_worker_perf")
    go_elapsed, go_output = run_go_worker(go_config, test_duration, num_workers=num_workers)
    go_metrics = collect_metrics(go_test_dir)

    print("\nGo worker metrics:")
    for key, value in go_metrics.items():
        print(f"  {key}: {value}")

    # Compare results
    print("\n" + "="*60)
    print("PERFORMANCE COMPARISON")
    print("="*60)

    print(f"\nDocuments Completed:")
    print(f"  Python: {python_metrics['documents_completed']}")
    print(f"  Go:     {go_metrics['documents_completed']}")
    if python_metrics['documents_completed'] > 0:
        speedup = go_metrics['documents_completed'] / python_metrics['documents_completed']
        print(f"  Speedup: {speedup:.2f}x")

    print(f"\nEmbeddings Generated:")
    print(f"  Python: {python_metrics['total_embeddings']}")
    print(f"  Go:     {go_metrics['total_embeddings']}")
    if python_metrics['total_embeddings'] > 0:
        speedup = go_metrics['total_embeddings'] / python_metrics['total_embeddings']
        print(f"  Speedup: {speedup:.2f}x")

    print(f"\nElements Extracted:")
    print(f"  Python: {python_metrics['total_elements']}")
    print(f"  Go:     {go_metrics['total_elements']}")
    if python_metrics['total_elements'] > 0:
        speedup = go_metrics['total_elements'] / python_metrics['total_elements']
        print(f"  Speedup: {speedup:.2f}x")

    print(f"\nRelationships Detected:")
    print(f"  Python: {python_metrics['total_relationships']}")
    print(f"  Go:     {go_metrics['total_relationships']}")
    if python_metrics['total_relationships'] > 0:
        speedup = go_metrics['total_relationships'] / python_metrics['total_relationships']
        print(f"  Speedup: {speedup:.2f}x")

    # Calculate throughput
    if python_elapsed > 0 and go_elapsed > 0:
        print(f"\nThroughput (documents/second):")
        python_throughput = python_metrics['documents_completed'] / python_elapsed
        go_throughput = go_metrics['documents_completed'] / go_elapsed
        print(f"  Python: {python_throughput:.2f} docs/sec")
        print(f"  Go:     {go_throughput:.2f} docs/sec")

        print(f"\nThroughput (embeddings/second):")
        python_embed_throughput = python_metrics['total_embeddings'] / python_elapsed
        go_embed_throughput = go_metrics['total_embeddings'] / go_elapsed
        print(f"  Python: {python_embed_throughput:.2f} embeddings/sec")
        print(f"  Go:     {go_embed_throughput:.2f} embeddings/sec")

    print("\n" + "="*60)
    print(f"Test results saved to:")
    print(f"  Python: {python_test_dir}")
    print(f"  Go:     {go_test_dir}")
    print("="*60)

if __name__ == "__main__":
    main()
