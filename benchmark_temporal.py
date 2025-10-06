#!/usr/bin/env python3
"""
Benchmark comparison between Python and Go temporal processing
"""

import time
import subprocess
import sys
from typing import List, Tuple
import json

# Add the src directory to path
sys.path.insert(0, 'src')

from go_doc_go.document_parser.temporal_semantics import detect_temporal_type, TemporalType
from go_doc_go.document_parser.temporal_metadata import generate_temporal_metadata
from go_doc_go.document_parser.temporal_normalization import normalize_temporal

def benchmark_python_temporal(test_data: List[str], iterations: int = 1000) -> Tuple[float, float, float]:
    """Benchmark Python temporal processing"""

    # Benchmark detect_temporal_type
    start = time.perf_counter()
    for _ in range(iterations):
        for value in test_data:
            detect_temporal_type(value)
    detect_time = time.perf_counter() - start

    # Benchmark generate_temporal_metadata
    start = time.perf_counter()
    for _ in range(iterations):
        for value in test_data:
            generate_temporal_metadata(value)
    metadata_time = time.perf_counter() - start

    # Benchmark normalize_temporal
    start = time.perf_counter()
    for _ in range(iterations):
        for value in test_data:
            temporal_type = detect_temporal_type(value)
            normalize_temporal(value, temporal_type)
    normalize_time = time.perf_counter() - start

    return detect_time, metadata_time, normalize_time


def benchmark_go_temporal() -> dict:
    """Run Go benchmarks and parse results"""

    # Run Go benchmarks
    result = subprocess.run(
        ['go', 'test', './go/internal/temporal', '-bench=.', '-benchtime=10s'],
        capture_output=True,
        text=True
    )

    # Parse benchmark results
    benchmarks = {}
    for line in result.stdout.split('\n'):
        if 'Benchmark' in line and 'ns/op' in line:
            parts = line.split()
            name = parts[0]
            ns_per_op = float(parts[2])
            benchmarks[name] = ns_per_op / 1e9  # Convert to seconds

    return benchmarks


def main():
    # Test data set
    test_data = [
        "2023-12-25",
        "December 25, 2023",
        "12/25/2023",
        "3:45pm",
        "15:30",
        "2023-12-25 14:30",
        "2023-12-25T14:30:00Z",
        "14:00-16:00",
        "2pm-4pm",
        "Q1 2024",
        "hello world",  # Non-temporal
        "123.45",       # Non-temporal
    ]

    iterations = 1000

    print("=" * 60)
    print("TEMPORAL PROCESSING BENCHMARK: Python vs Go")
    print("=" * 60)
    print(f"Test data items: {len(test_data)}")
    print(f"Iterations: {iterations}")
    print()

    # Benchmark Python
    print("Benchmarking Python temporal processing...")
    py_detect, py_metadata, py_normalize = benchmark_python_temporal(test_data, iterations)

    total_ops = len(test_data) * iterations

    print(f"Python Results:")
    print(f"  DetectTemporalType:  {py_detect:.3f}s total, {py_detect/total_ops*1e6:.2f} µs/op")
    print(f"  GenerateMetadata:    {py_metadata:.3f}s total, {py_metadata/total_ops*1e6:.2f} µs/op")
    print(f"  NormalizeTemporal:   {py_normalize:.3f}s total, {py_normalize/total_ops*1e6:.2f} µs/op")
    print(f"  Total time:          {py_detect + py_metadata + py_normalize:.3f}s")
    print()

    # Benchmark Go
    print("Running Go benchmarks...")
    go_benchmarks = benchmark_go_temporal()

    if go_benchmarks:
        print("Go Results (from benchmark suite):")
        for name, time_per_op in go_benchmarks.items():
            print(f"  {name}: {time_per_op*1e6:.2f} µs/op")
        print()

        # Calculate speedup if we have comparable benchmarks
        if 'BenchmarkDetectTemporalType' in go_benchmarks:
            go_detect_us = go_benchmarks['BenchmarkDetectTemporalType'] * 1e6
            py_detect_us = py_detect / total_ops * 1e6
            speedup = py_detect_us / go_detect_us
            print(f"DetectTemporalType speedup: {speedup:.1f}x faster in Go")

        if 'BenchmarkGenerateTemporalMetadata' in go_benchmarks:
            go_metadata_us = go_benchmarks['BenchmarkGenerateTemporalMetadata'] * 1e6
            py_metadata_us = py_metadata / total_ops * 1e6
            speedup = py_metadata_us / go_metadata_us
            print(f"GenerateMetadata speedup: {speedup:.1f}x faster in Go")

        if 'BenchmarkNormalizeTemporal' in go_benchmarks:
            go_normalize_us = go_benchmarks['BenchmarkNormalizeTemporal'] * 1e6
            py_normalize_us = py_normalize / total_ops * 1e6
            speedup = py_normalize_us / go_normalize_us
            print(f"NormalizeTemporal speedup: {speedup:.1f}x faster in Go")

    print()
    print("=" * 60)
    print("SUMMARY")
    print("=" * 60)

    # Average operations per second
    py_ops_per_sec = total_ops / (py_detect + py_metadata + py_normalize)
    print(f"Python: {py_ops_per_sec:,.0f} operations/second")

    # Estimate Go performance based on individual benchmarks
    if go_benchmarks:
        avg_go_time = sum(go_benchmarks.values()) / len(go_benchmarks)
        go_ops_per_sec = 1 / avg_go_time
        print(f"Go (estimated): {go_ops_per_sec:,.0f} operations/second")
        print(f"Overall speedup: ~{go_ops_per_sec/py_ops_per_sec:.1f}x")


if __name__ == "__main__":
    main()