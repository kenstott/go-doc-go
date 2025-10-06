"""
Performance benchmarks for JSON parser implementations.

These tests compare the performance of Python vs Go JSON parsers
to validate that the Go implementation provides performance benefits.
"""

import json
import time
import pytest
from pathlib import Path

from src.go_doc_go.document_parser.json import JSONParser
from src.go_doc_go.document_parser.json_go import GoJSONParser


@pytest.mark.performance
class TestJSONPerformance:
    """Performance benchmarks for JSON parsers."""

    @pytest.fixture
    def python_parser(self):
        """Create Python JSON parser instance."""
        return JSONParser({})

    @pytest.fixture
    def go_parser(self):
        """Create Go JSON parser instance."""
        # Check if binary exists
        project_root = Path(__file__).parent.parent
        binary_path = project_root / "bin" / "jsonparser"

        if not binary_path.exists():
            pytest.skip("Go JSON parser binary not found")

        return GoJSONParser({})

    @pytest.fixture
    def small_json_content(self):
        """Small JSON content for testing."""
        return {
            "id": "small_test",
            "content": json.dumps({
                "name": "Test Document",
                "version": "1.0",
                "metadata": {
                    "created": "2023-01-01",
                    "author": "Test Author"
                },
                "items": ["item1", "item2", "item3"]
            }),
            "metadata": {"source": "small_test"}
        }

    @pytest.fixture
    def medium_json_content(self):
        """Medium JSON content for testing."""
        # Create a more complex structure
        data = {
            "users": [
                {
                    "id": i,
                    "name": f"User {i}",
                    "email": f"user{i}@example.com",
                    "profile": {
                        "bio": f"This is user {i}'s biography",
                        "preferences": {
                            "theme": "dark" if i % 2 == 0 else "light",
                            "notifications": True,
                            "privacy": "public"
                        },
                        "tags": [f"tag{j}" for j in range(5)],
                        "scores": [j * 10 for j in range(10)]
                    },
                    "posts": [
                        {
                            "id": f"post_{i}_{j}",
                            "title": f"Post {j} by User {i}",
                            "content": f"This is the content of post {j} by user {i}",
                            "created": f"2023-{(j % 12) + 1:02d}-01",
                            "tags": [f"topic{k}" for k in range(3)],
                            "metrics": {
                                "views": j * 100,
                                "likes": j * 10,
                                "comments": j * 2
                            }
                        }
                        for j in range(5)
                    ]
                }
                for i in range(20)  # 20 users
            ],
            "metadata": {
                "total_users": 20,
                "created": "2023-01-01",
                "version": "2.0",
                "settings": {
                    "max_users": 1000,
                    "features": ["messaging", "posts", "profiles"],
                    "enabled": True
                }
            }
        }

        return {
            "id": "medium_test",
            "content": json.dumps(data),
            "metadata": {"source": "medium_test"}
        }

    @pytest.fixture
    def large_json_content(self):
        """Large JSON content for testing."""
        # Create an even larger structure
        data = {
            "organizations": [
                {
                    "id": org_id,
                    "name": f"Organization {org_id}",
                    "website": f"https://org{org_id}.example.com",
                    "departments": [
                        {
                            "id": dept_id,
                            "name": f"Department {dept_id}",
                            "employees": [
                                {
                                    "id": emp_id,
                                    "name": f"Employee {emp_id}",
                                    "email": f"employee{emp_id}@org{org_id}.com",
                                    "position": f"Position {emp_id % 5}",
                                    "skills": [f"skill{s}" for s in range(emp_id % 7 + 1)],
                                    "projects": [
                                        {
                                            "id": f"project_{org_id}_{dept_id}_{emp_id}_{p}",
                                            "name": f"Project {p}",
                                            "status": "active" if p % 2 == 0 else "completed",
                                            "budget": p * 10000,
                                            "timeline": {
                                                "start": f"2023-{(p % 12) + 1:02d}-01",
                                                "end": f"2023-{((p + 6) % 12) + 1:02d}-01"
                                            },
                                            "tasks": [
                                                {
                                                    "id": f"task_{p}_{t}",
                                                    "description": f"Task {t} for project {p}",
                                                    "completed": t % 3 == 0,
                                                    "priority": ["low", "medium", "high"][t % 3]
                                                }
                                                for t in range(5)
                                            ]
                                        }
                                        for p in range(3)
                                    ]
                                }
                                for emp_id in range(10)  # 10 employees per department
                            ]
                        }
                        for dept_id in range(5)  # 5 departments per organization
                    ]
                }
                for org_id in range(5)  # 5 organizations
            ]
        }

        return {
            "id": "large_test",
            "content": json.dumps(data),
            "metadata": {"source": "large_test"}
        }

    def measure_parsing_time(self, parser, content, iterations=5):
        """Measure parsing time for given content."""
        times = []
        for _ in range(iterations):
            start_time = time.time()
            result = parser.parse(content)
            end_time = time.time()
            times.append(end_time - start_time)
            # Verify parsing succeeded
            assert len(result["elements"]) > 0

        return {
            "min": min(times),
            "max": max(times),
            "avg": sum(times) / len(times),
            "times": times
        }

    @pytest.mark.performance
    def test_small_json_performance(self, python_parser, go_parser, small_json_content):
        """Compare performance on small JSON documents."""
        print("\n=== Small JSON Performance Test ===")

        # Measure Python parser
        python_stats = self.measure_parsing_time(python_parser, small_json_content)
        print(f"Python parser: {python_stats['avg']:.4f}s avg ({python_stats['min']:.4f}s - {python_stats['max']:.4f}s)")

        # Measure Go parser
        go_stats = self.measure_parsing_time(go_parser, small_json_content)
        print(f"Go parser: {go_stats['avg']:.4f}s avg ({go_stats['min']:.4f}s - {go_stats['max']:.4f}s)")

        # Calculate speedup
        speedup = python_stats['avg'] / go_stats['avg']
        print(f"Go speedup: {speedup:.2f}x")

        # Both should complete reasonably quickly
        assert python_stats['avg'] < 2.0, "Python parser should complete quickly"
        assert go_stats['avg'] < 2.0, "Go parser should complete quickly"

    @pytest.mark.performance
    def test_medium_json_performance(self, python_parser, go_parser, medium_json_content):
        """Compare performance on medium JSON documents."""
        print("\n=== Medium JSON Performance Test ===")

        # Measure Python parser
        python_stats = self.measure_parsing_time(python_parser, medium_json_content)
        print(f"Python parser: {python_stats['avg']:.4f}s avg ({python_stats['min']:.4f}s - {python_stats['max']:.4f}s)")

        # Measure Go parser
        go_stats = self.measure_parsing_time(go_parser, medium_json_content)
        print(f"Go parser: {go_stats['avg']:.4f}s avg ({go_stats['min']:.4f}s - {go_stats['max']:.4f}s)")

        # Calculate speedup
        speedup = python_stats['avg'] / go_stats['avg']
        print(f"Go speedup: {speedup:.2f}x")

        # Both should complete in reasonable time
        assert python_stats['avg'] < 5.0, "Python parser should complete in reasonable time"
        assert go_stats['avg'] < 5.0, "Go parser should complete in reasonable time"

        # Go should be faster or at least not significantly slower
        assert go_stats['avg'] <= python_stats['avg'] * 1.5, "Go parser should not be significantly slower"

    @pytest.mark.performance
    def test_large_json_performance(self, python_parser, go_parser, large_json_content):
        """Compare performance on large JSON documents."""
        print("\n=== Large JSON Performance Test ===")

        # Measure Python parser (fewer iterations for large content)
        python_stats = self.measure_parsing_time(python_parser, large_json_content, iterations=3)
        print(f"Python parser: {python_stats['avg']:.4f}s avg ({python_stats['min']:.4f}s - {python_stats['max']:.4f}s)")

        # Measure Go parser
        go_stats = self.measure_parsing_time(go_parser, large_json_content, iterations=3)
        print(f"Go parser: {go_stats['avg']:.4f}s avg ({go_stats['min']:.4f}s - {go_stats['max']:.4f}s)")

        # Calculate speedup
        speedup = python_stats['avg'] / go_stats['avg']
        print(f"Go speedup: {speedup:.2f}x")

        # Both should complete without timeout
        assert python_stats['avg'] < 30.0, "Python parser should complete without timeout"
        assert go_stats['avg'] < 30.0, "Go parser should complete without timeout"

        # For large documents, Go should show performance benefits
        # Allow some overhead for subprocess communication
        assert go_stats['avg'] <= python_stats['avg'] * 2.0, "Go parser should not be much slower than Python"

    @pytest.mark.performance
    def test_parsing_scalability(self, python_parser, go_parser, medium_json_content):
        """Test parsing scalability with multiple documents."""
        print("\n=== Scalability Test ===")

        # Test parsing multiple documents
        documents = []
        for i in range(10):
            doc = medium_json_content.copy()
            doc["id"] = f"scale_test_{i}"
            documents.append(doc)

        # Measure Python parser
        start_time = time.time()
        python_results = []
        for doc in documents:
            result = python_parser.parse(doc)
            python_results.append(result)
        python_time = time.time() - start_time

        # Measure Go parser
        start_time = time.time()
        go_results = []
        for doc in documents:
            result = go_parser.parse(doc)
            go_results.append(result)
        go_time = time.time() - start_time

        print(f"Python parser (10 docs): {python_time:.4f}s total, {python_time/10:.4f}s avg")
        print(f"Go parser (10 docs): {go_time:.4f}s total, {go_time/10:.4f}s avg")

        # Calculate speedup
        speedup = python_time / go_time
        print(f"Go speedup: {speedup:.2f}x")

        # Verify results
        assert len(python_results) == 10, "Should parse all Python documents"
        assert len(go_results) == 10, "Should parse all Go documents"

        # Check that all documents were parsed successfully
        for result in python_results + go_results:
            assert len(result["elements"]) > 0, "Each document should have elements"

        # Performance check
        assert python_time < 60.0, "Python batch processing should complete in reasonable time"
        assert go_time < 60.0, "Go batch processing should complete in reasonable time"

    @pytest.mark.performance
    def test_memory_efficiency(self, python_parser, go_parser, large_json_content):
        """Test memory efficiency of parsers."""
        print("\n=== Memory Efficiency Test ===")

        import psutil
        import os

        process = psutil.Process(os.getpid())

        # Measure Python parser memory usage
        initial_memory = process.memory_info().rss
        python_result = python_parser.parse(large_json_content)
        python_peak_memory = process.memory_info().rss
        python_memory_used = python_peak_memory - initial_memory

        print(f"Python parser memory usage: {python_memory_used / 1024 / 1024:.2f} MB")

        # Reset and measure Go parser memory usage
        initial_memory = process.memory_info().rss
        go_result = go_parser.parse(large_json_content)
        go_peak_memory = process.memory_info().rss
        go_memory_used = go_peak_memory - initial_memory

        print(f"Go parser memory usage: {go_memory_used / 1024 / 1024:.2f} MB")

        # Verify results are comparable
        assert len(python_result["elements"]) == len(go_result["elements"])

        # Memory usage should be reasonable
        max_reasonable_memory = 500 * 1024 * 1024  # 500MB
        assert python_memory_used < max_reasonable_memory, "Python parser memory usage should be reasonable"
        assert go_memory_used < max_reasonable_memory, "Go parser memory usage should be reasonable"

        print(f"Memory efficiency ratio: {python_memory_used / max(go_memory_used, 1):.2f}")

    @pytest.mark.performance
    def test_concurrent_parsing(self, python_parser, go_parser, medium_json_content):
        """Test concurrent parsing capabilities."""
        print("\n=== Concurrent Parsing Test ===")

        import threading
        import queue

        def parse_worker(parser, content_queue, result_queue, parser_name):
            """Worker function for concurrent parsing."""
            while True:
                try:
                    content = content_queue.get(timeout=1)
                    if content is None:
                        break

                    start_time = time.time()
                    result = parser.parse(content)
                    end_time = time.time()

                    result_queue.put({
                        'parser': parser_name,
                        'time': end_time - start_time,
                        'elements': len(result['elements'])
                    })
                    content_queue.task_done()
                except queue.Empty:
                    break

        # Prepare multiple documents
        documents = []
        for i in range(8):
            doc = medium_json_content.copy()
            doc["id"] = f"concurrent_test_{i}"
            documents.append(doc)

        # Test Python parser concurrency
        content_queue = queue.Queue()
        result_queue = queue.Queue()

        for doc in documents:
            content_queue.put(doc)

        python_start = time.time()
        python_threads = []
        for i in range(4):  # 4 concurrent threads
            thread = threading.Thread(
                target=parse_worker,
                args=(python_parser, content_queue, result_queue, "python")
            )
            thread.start()
            python_threads.append(thread)

        # Wait for completion
        content_queue.join()
        for thread in python_threads:
            thread.join()

        python_end = time.time()
        python_concurrent_time = python_end - python_start

        # Collect Python results
        python_results = []
        while not result_queue.empty():
            python_results.append(result_queue.get())

        # Test Go parser concurrency (sequential due to subprocess limitations)
        go_start = time.time()
        go_results = []
        for doc in documents:
            start_time = time.time()
            result = go_parser.parse(doc)
            end_time = time.time()
            go_results.append({
                'parser': 'go',
                'time': end_time - start_time,
                'elements': len(result['elements'])
            })
        go_end = time.time()
        go_sequential_time = go_end - go_start

        print(f"Python concurrent (4 threads): {python_concurrent_time:.4f}s")
        print(f"Go sequential: {go_sequential_time:.4f}s")

        # Verify results
        assert len(python_results) == 8, "Should process all documents with Python"
        assert len(go_results) == 8, "Should process all documents with Go"

        # Calculate average processing times
        python_avg = sum(r['time'] for r in python_results) / len(python_results)
        go_avg = sum(r['time'] for r in go_results) / len(go_results)

        print(f"Python avg per document: {python_avg:.4f}s")
        print(f"Go avg per document: {go_avg:.4f}s")

        # Both should complete in reasonable time
        assert python_concurrent_time < 30.0, "Python concurrent parsing should complete quickly"
        assert go_sequential_time < 30.0, "Go sequential parsing should complete quickly"