#!/usr/bin/env python3
"""
Performance benchmark tests for Python vs Go HTML parsers.

This test suite measures and compares the performance characteristics
of the Python and Go HTML parser implementations.
"""

import sys
import os
import time
import unittest
import statistics
from typing import List, Dict, Any, Tuple

# Add src directory to path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..', 'src'))

from go_doc_go.document_parser.html import HtmlParser as PythonHTMLParser
from go_doc_go.document_parser.html_go import GoHTMLParser
import pytest


@pytest.mark.performance
class HTMLPerformanceBenchmark(unittest.TestCase):
    """Performance benchmark tests for HTML parsers."""

    def setUp(self):
        """Set up test parsers and benchmark data."""
        self.python_parser = PythonHTMLParser()
        self.go_parser = GoHTMLParser()

        # Generate test content of various sizes
        self.test_cases = self._generate_test_cases()

    def _generate_test_cases(self) -> List[Dict[str, Any]]:
        """
        Generate test cases with varying complexity and size.

        Returns:
            List of test case dictionaries
        """
        test_cases = []

        # Small document
        small_html = """
        <html>
            <head><title>Small Test</title></head>
            <body>
                <h1>Small Document</h1>
                <p>This is a small test document with <a href="/link">a link</a>.</p>
                <ul>
                    <li>Item 1</li>
                    <li>Item 2</li>
                    <li>Item 3</li>
                </ul>
            </body>
        </html>
        """

        test_cases.append({
            "name": "small",
            "size": "~500 bytes",
            "content": {
                "id": "small_test",
                "content": small_html,
                "metadata": {"source": "small_test.html"}
            }
        })

        # Medium document
        medium_html = "<html><head><title>Medium Test</title></head><body>"
        for i in range(50):
            medium_html += f"""
            <section>
                <h2>Section {i}</h2>
                <p>This is section {i} with some content and <a href="/section{i}">a link</a>.</p>
                <div class="content">
                    <p>More content in section {i} with additional text.</p>
                    <ul>
                        <li>Item {i}-1</li>
                        <li>Item {i}-2</li>
                        <li>Item {i}-3</li>
                    </ul>
                </div>
            </section>
            """
        medium_html += "</body></html>"

        test_cases.append({
            "name": "medium",
            "size": f"~{len(medium_html)} bytes",
            "content": {
                "id": "medium_test",
                "content": medium_html,
                "metadata": {"source": "medium_test.html"}
            }
        })

        # Large document
        large_html = "<html><head><title>Large Test</title></head><body>"
        for i in range(200):
            large_html += f"""
            <article id="article-{i}" class="article">
                <header>
                    <h1>Article {i} Title</h1>
                    <p class="meta">Published on 2024-01-{i % 28 + 1:02d}</p>
                </header>
                <section class="content">
                    <p>This is the introduction to article {i} with some meaningful content.</p>
                    <p>Article {i} continues with more detailed information and <a href="/article/{i}">internal links</a>.</p>
                    <table>
                        <tr>
                            <th>Column 1</th>
                            <th>Column 2</th>
                            <th>Column 3</th>
                        </tr>
                        <tr>
                            <td>Data {i}-1</td>
                            <td>Data {i}-2</td>
                            <td>Data {i}-3</td>
                        </tr>
                    </table>
                    <div class="sidebar">
                        <h3>Related Links</h3>
                        <ul>
                            <li><a href="/related/{i}/1">Related Article 1</a></li>
                            <li><a href="/related/{i}/2">Related Article 2</a></li>
                            <li><a href="/related/{i}/3">Related Article 3</a></li>
                        </ul>
                    </div>
                </section>
                <footer>
                    <p>Tags: <span class="tag">tag{i % 5}</span>, <span class="tag">category{i % 3}</span></p>
                </footer>
            </article>
            """
        large_html += "</body></html>"

        test_cases.append({
            "name": "large",
            "size": f"~{len(large_html)} bytes",
            "content": {
                "id": "large_test",
                "content": large_html,
                "metadata": {"source": "large_test.html"}
            }
        })

        # Complex document with nested structures
        complex_html = """
        <html>
            <head>
                <title>Complex Document</title>
                <meta charset="UTF-8">
                <meta name="viewport" content="width=device-width, initial-scale=1.0">
            </head>
            <body>
                <header class="main-header">
                    <nav class="primary-nav">
                        <ul class="nav-list">
        """

        for i in range(20):
            complex_html += f'<li><a href="/nav/{i}">Nav Item {i}</a></li>'

        complex_html += """
                        </ul>
                    </nav>
                </header>
                <main class="content-area">
        """

        for section in range(10):
            complex_html += f"""
            <section class="content-section" id="section-{section}">
                <h2>Complex Section {section}</h2>
                <div class="section-content">
            """

            for subsection in range(5):
                complex_html += f"""
                <div class="subsection" data-id="{section}-{subsection}">
                    <h3>Subsection {section}.{subsection}</h3>
                    <p>Content for subsection {section}.{subsection} with <strong>emphasis</strong> and <em>italics</em>.</p>
                    <blockquote cite="http://example.com/{section}/{subsection}">
                        This is a quote in subsection {section}.{subsection}.
                    </blockquote>
                    <div class="data-table">
                        <table>
                            <thead>
                                <tr>
                                    <th>ID</th>
                                    <th>Name</th>
                                    <th>Value</th>
                                    <th>Status</th>
                                </tr>
                            </thead>
                            <tbody>
                """

                for row in range(3):
                    complex_html += f"""
                    <tr class="data-row" data-row="{row}">
                        <td>{section}{subsection}{row}</td>
                        <td>Item {section}.{subsection}.{row}</td>
                        <td>{(section + subsection + row) * 100}</td>
                        <td><span class="status active">Active</span></td>
                    </tr>
                    """

                complex_html += """
                            </tbody>
                        </table>
                    </div>
                </div>
                """

            complex_html += """
                </div>
            </section>
            """

        complex_html += """
                </main>
                <aside class="sidebar">
                    <div class="widget">
                        <h3>Related Content</h3>
                        <ul class="related-list">
        """

        for i in range(15):
            complex_html += f'<li><a href="/related/{i}" class="related-link">Related Item {i}</a></li>'

        complex_html += """
                        </ul>
                    </div>
                </aside>
                <footer class="main-footer">
                    <div class="footer-content">
                        <p>&copy; 2024 Complex Document Example</p>
                    </div>
                </footer>
            </body>
        </html>
        """

        test_cases.append({
            "name": "complex",
            "size": f"~{len(complex_html)} bytes",
            "content": {
                "id": "complex_test",
                "content": complex_html,
                "metadata": {"source": "complex_test.html"}
            }
        })

        return test_cases

    def _benchmark_parser(self, parser, content: Dict[str, Any], iterations: int = 5) -> Tuple[List[float], Dict[str, Any]]:
        """
        Benchmark a parser with given content.

        Args:
            parser: Parser instance to benchmark
            content: Content to parse
            iterations: Number of iterations to run

        Returns:
            Tuple of (execution times, last result)
        """
        times = []
        result = None

        for _ in range(iterations):
            start_time = time.time()
            try:
                result = parser.parse(content)
                end_time = time.time()
                times.append(end_time - start_time)
            except Exception as e:
                # If Python parser fails, return empty times
                print(f"Parser failed: {e}")
                return [], {}

        return times, result

    def _analyze_results(self, python_times: List[float], go_times: List[float],
                        python_result: Dict[str, Any], go_result: Dict[str, Any]) -> Dict[str, Any]:
        """
        Analyze benchmark results.

        Args:
            python_times: Python parser execution times
            go_times: Go parser execution times
            python_result: Last result from Python parser
            go_result: Last result from Go parser

        Returns:
            Analysis dictionary
        """
        analysis = {}

        if python_times:
            analysis["python"] = {
                "mean_time": statistics.mean(python_times),
                "median_time": statistics.median(python_times),
                "min_time": min(python_times),
                "max_time": max(python_times),
                "std_dev": statistics.stdev(python_times) if len(python_times) > 1 else 0,
                "elements_found": len(python_result.get("elements", [])),
                "links_found": len(python_result.get("links", [])),
                "relationships_found": len(python_result.get("relationships", []))
            }
        else:
            analysis["python"] = {"failed": True}

        if go_times:
            analysis["go"] = {
                "mean_time": statistics.mean(go_times),
                "median_time": statistics.median(go_times),
                "min_time": min(go_times),
                "max_time": max(go_times),
                "std_dev": statistics.stdev(go_times) if len(go_times) > 1 else 0,
                "elements_found": len(go_result.get("elements", [])),
                "links_found": len(go_result.get("links", [])),
                "relationships_found": len(go_result.get("relationships", []))
            }
        else:
            analysis["go"] = {"failed": True}

        # Calculate performance improvement
        if python_times and go_times:
            python_mean = statistics.mean(python_times)
            go_mean = statistics.mean(go_times)

            if python_mean > 0:
                speedup = python_mean / go_mean
                improvement_pct = ((python_mean - go_mean) / python_mean) * 100
                analysis["performance"] = {
                    "speedup_factor": speedup,
                    "improvement_percentage": improvement_pct,
                    "faster_parser": "go" if go_mean < python_mean else "python"
                }

        return analysis

    def test_small_document_performance(self):
        """Test performance on small documents."""
        test_case = next(tc for tc in self.test_cases if tc["name"] == "small")
        content = test_case["content"]

        print(f"\n=== Small Document Benchmark ({test_case['size']}) ===")

        # Benchmark Python parser
        try:
            python_times, python_result = self._benchmark_parser(self.python_parser, content, iterations=10)
        except Exception:
            python_times, python_result = [], {}

        # Benchmark Go parser
        go_times, go_result = self._benchmark_parser(self.go_parser, content, iterations=10)

        # Analyze results
        analysis = self._analyze_results(python_times, go_times, python_result, go_result)

        self._print_analysis(analysis)

        # Assertions
        self.assertGreater(len(go_times), 0, "Go parser should complete successfully")
        self.assertGreater(len(go_result.get("elements", [])), 0, "Should find elements")

    def test_medium_document_performance(self):
        """Test performance on medium documents."""
        test_case = next(tc for tc in self.test_cases if tc["name"] == "medium")
        content = test_case["content"]

        print(f"\n=== Medium Document Benchmark ({test_case['size']}) ===")

        # Benchmark Python parser
        try:
            python_times, python_result = self._benchmark_parser(self.python_parser, content, iterations=5)
        except Exception:
            python_times, python_result = [], {}

        # Benchmark Go parser
        go_times, go_result = self._benchmark_parser(self.go_parser, content, iterations=5)

        # Analyze results
        analysis = self._analyze_results(python_times, go_times, python_result, go_result)

        self._print_analysis(analysis)

        # Assertions
        self.assertGreater(len(go_times), 0, "Go parser should complete successfully")
        self.assertGreater(len(go_result.get("elements", [])), 50, "Should find many elements")

    def test_large_document_performance(self):
        """Test performance on large documents."""
        test_case = next(tc for tc in self.test_cases if tc["name"] == "large")
        content = test_case["content"]

        print(f"\n=== Large Document Benchmark ({test_case['size']}) ===")

        # Benchmark Python parser (fewer iterations for large docs)
        try:
            python_times, python_result = self._benchmark_parser(self.python_parser, content, iterations=3)
        except Exception:
            python_times, python_result = [], {}

        # Benchmark Go parser
        go_times, go_result = self._benchmark_parser(self.go_parser, content, iterations=3)

        # Analyze results
        analysis = self._analyze_results(python_times, go_times, python_result, go_result)

        self._print_analysis(analysis)

        # Assertions
        self.assertGreater(len(go_times), 0, "Go parser should complete successfully")
        self.assertGreater(len(go_result.get("elements", [])), 200, "Should find many elements")

        # Performance assertion - Go should be reasonably fast
        if go_times:
            max_time = max(go_times)
            self.assertLess(max_time, 5.0, f"Go parser should complete large document in under 5s, took {max_time:.3f}s")

    def test_complex_document_performance(self):
        """Test performance on complex nested documents."""
        test_case = next(tc for tc in self.test_cases if tc["name"] == "complex")
        content = test_case["content"]

        print(f"\n=== Complex Document Benchmark ({test_case['size']}) ===")

        # Benchmark Python parser
        try:
            python_times, python_result = self._benchmark_parser(self.python_parser, content, iterations=3)
        except Exception:
            python_times, python_result = [], {}

        # Benchmark Go parser
        go_times, go_result = self._benchmark_parser(self.go_parser, content, iterations=3)

        # Analyze results
        analysis = self._analyze_results(python_times, go_times, python_result, go_result)

        self._print_analysis(analysis)

        # Assertions
        self.assertGreater(len(go_times), 0, "Go parser should complete successfully")
        self.assertGreater(len(go_result.get("elements", [])), 100, "Should find many elements")

    def _print_analysis(self, analysis: Dict[str, Any]):
        """Print performance analysis results."""

        if "python" in analysis and not analysis["python"].get("failed", False):
            python_stats = analysis["python"]
            print(f"Python Parser:")
            print(f"  Mean time: {python_stats['mean_time']:.3f}s")
            print(f"  Median time: {python_stats['median_time']:.3f}s")
            print(f"  Min time: {python_stats['min_time']:.3f}s")
            print(f"  Max time: {python_stats['max_time']:.3f}s")
            print(f"  Elements: {python_stats['elements_found']}")
            print(f"  Links: {python_stats['links_found']}")
            print(f"  Relationships: {python_stats['relationships_found']}")
        else:
            print("Python Parser: FAILED or SKIPPED")

        if "go" in analysis and not analysis["go"].get("failed", False):
            go_stats = analysis["go"]
            print(f"Go Parser:")
            print(f"  Mean time: {go_stats['mean_time']:.3f}s")
            print(f"  Median time: {go_stats['median_time']:.3f}s")
            print(f"  Min time: {go_stats['min_time']:.3f}s")
            print(f"  Max time: {go_stats['max_time']:.3f}s")
            print(f"  Elements: {go_stats['elements_found']}")
            print(f"  Links: {go_stats['links_found']}")
            print(f"  Relationships: {go_stats['relationships_found']}")
        else:
            print("Go Parser: FAILED")

        if "performance" in analysis:
            perf = analysis["performance"]
            print(f"Performance:")
            print(f"  Speedup: {perf['speedup_factor']:.2f}x")
            print(f"  Improvement: {perf['improvement_percentage']:.1f}%")
            print(f"  Faster: {perf['faster_parser'].upper()}")

    def test_memory_efficiency(self):
        """Test memory usage patterns (basic)."""
        # This is a basic test - full memory profiling would require additional tools
        import psutil
        import os

        process = psutil.Process(os.getpid())

        # Get baseline memory
        baseline_memory = process.memory_info().rss / 1024 / 1024  # MB

        # Test with large document
        test_case = next(tc for tc in self.test_cases if tc["name"] == "large")
        content = test_case["content"]

        # Measure Go parser memory usage
        memory_before = process.memory_info().rss / 1024 / 1024
        result = self.go_parser.parse(content)
        memory_after = process.memory_info().rss / 1024 / 1024

        memory_used = memory_after - memory_before

        print(f"\n=== Memory Usage Test ===")
        print(f"Baseline memory: {baseline_memory:.1f} MB")
        print(f"Memory before parsing: {memory_before:.1f} MB")
        print(f"Memory after parsing: {memory_after:.1f} MB")
        print(f"Memory used for parsing: {memory_used:.1f} MB")
        print(f"Elements parsed: {len(result.get('elements', []))}")

        # Memory should be reasonable (less than 100MB for this test)
        self.assertLess(memory_used, 100, f"Memory usage should be reasonable, used {memory_used:.1f} MB")


if __name__ == "__main__":
    # Run with verbose output
    unittest.main(verbosity=2)