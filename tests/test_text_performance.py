"""
Performance benchmarks for Python vs Go text parser implementations.

These tests measure parsing performance across different text sizes and complexities.
"""

import pytest
import time
from pathlib import Path
from typing import Dict, Any

from src.go_doc_go.document_parser.text import TextParser
from src.go_doc_go.document_parser.text_go import GoTextParser


class TestTextPerformance:
    """Performance benchmarks for text parsers."""

    @pytest.fixture
    def python_parser(self):
        """Create Python text parser instance."""
        return TextParser({})

    @pytest.fixture
    def go_parser(self):
        """Create Go text parser instance."""
        # Check if binary exists
        project_root = Path(__file__).parent.parent
        binary_path = project_root / "bin" / "textparser"

        if not binary_path.exists():
            pytest.skip("Go text parser binary not found")

        return GoTextParser({})

    @pytest.fixture
    def small_text_content(self):
        """Small text content for performance testing."""
        paragraph = "This is a test paragraph with some content. " * 10  # ~450 chars
        return {
            "id": "small_text_perf",
            "content": "\n\n".join([paragraph] * 5),  # 5 paragraphs
            "metadata": {"source": "small_text_performance_test"}
        }

    @pytest.fixture
    def medium_text_content(self):
        """Medium text content for performance testing."""
        paragraph = "This is a longer test paragraph with more detailed content including dates like 2024-01-15, numbers like 123.45, URLs like https://example.com, and email addresses like test@example.com. " * 5  # ~650 chars
        return {
            "id": "medium_text_perf",
            "content": "\n\n".join([paragraph] * 50),  # 50 paragraphs
            "metadata": {"source": "medium_text_performance_test"}
        }

    @pytest.fixture
    def large_text_content(self):
        """Large text content for performance testing."""
        paragraph = "This is a comprehensive test paragraph containing various elements for performance evaluation. It includes dates such as January 1, 2024, and 2024-12-31, numerical values like 99.99% accuracy, 1,234,567 items processed, financial figures $10,000.50 and €5,432.10, web links https://performance.test.com and https://benchmark.example.org, email contacts performance@test.com and benchmark@example.org, file references /data/performance/results.txt and C:\\Performance\\Benchmarks\\data.csv, and various other textual content designed to stress test the parsing capabilities. " * 3  # ~1200 chars
        return {
            "id": "large_text_perf",
            "content": "\n\n".join([paragraph] * 200),  # 200 paragraphs
            "metadata": {"source": "large_text_performance_test"}
        }

    def _benchmark_parser(self, parser, content: Dict[str, Any], iterations: int = 5) -> Dict[str, float]:
        """Benchmark a parser with given content."""
        times = []

        for i in range(iterations):
            start_time = time.time()
            result = parser.parse(content)
            end_time = time.time()

            parse_time = end_time - start_time
            times.append(parse_time)

            # Validate result structure
            assert "document" in result
            assert "elements" in result
            assert "relationships" in result
            assert len(result["elements"]) > 0

        return {
            "min_time": min(times),
            "max_time": max(times),
            "avg_time": sum(times) / len(times),
            "times": times
        }

    @pytest.mark.performance
    def test_small_text_performance(self, python_parser, go_parser, small_text_content):
        """Compare performance on small text files."""
        print("\n=== Small Text Performance Test ===")

        # Benchmark Python parser
        python_stats = self._benchmark_parser(python_parser, small_text_content)
        print(f"Python parser - Avg: {python_stats['avg_time']:.4f}s, Min: {python_stats['min_time']:.4f}s, Max: {python_stats['max_time']:.4f}s")

        # Benchmark Go parser
        go_stats = self._benchmark_parser(go_parser, small_text_content)
        print(f"Go parser     - Avg: {go_stats['avg_time']:.4f}s, Min: {go_stats['min_time']:.4f}s, Max: {go_stats['max_time']:.4f}s")

        # Calculate performance ratio
        ratio = python_stats['avg_time'] / go_stats['avg_time']
        print(f"Performance ratio (Python/Go): {ratio:.2f}x")

        # Both should complete in reasonable time
        assert python_stats['avg_time'] < 5.0  # Should complete in under 5 seconds
        assert go_stats['avg_time'] < 5.0

    @pytest.mark.performance
    def test_medium_text_performance(self, python_parser, go_parser, medium_text_content):
        """Compare performance on medium text files."""
        print("\n=== Medium Text Performance Test ===")

        # Benchmark Python parser
        python_stats = self._benchmark_parser(python_parser, medium_text_content)
        print(f"Python parser - Avg: {python_stats['avg_time']:.4f}s, Min: {python_stats['min_time']:.4f}s, Max: {python_stats['max_time']:.4f}s")

        # Benchmark Go parser
        go_stats = self._benchmark_parser(go_parser, medium_text_content)
        print(f"Go parser     - Avg: {go_stats['avg_time']:.4f}s, Min: {go_stats['min_time']:.4f}s, Max: {go_stats['max_time']:.4f}s")

        # Calculate performance ratio
        ratio = python_stats['avg_time'] / go_stats['avg_time']
        print(f"Performance ratio (Python/Go): {ratio:.2f}x")

        # Both should complete in reasonable time
        assert python_stats['avg_time'] < 10.0  # Should complete in under 10 seconds
        assert go_stats['avg_time'] < 10.0

    @pytest.mark.performance
    def test_large_text_performance(self, python_parser, go_parser, large_text_content):
        """Compare performance on large text files."""
        print("\n=== Large Text Performance Test ===")

        # Use fewer iterations for large files
        iterations = 3

        # Benchmark Python parser
        python_stats = self._benchmark_parser(python_parser, large_text_content, iterations)
        print(f"Python parser - Avg: {python_stats['avg_time']:.4f}s, Min: {python_stats['min_time']:.4f}s, Max: {python_stats['max_time']:.4f}s")

        # Benchmark Go parser
        go_stats = self._benchmark_parser(go_parser, large_text_content, iterations)
        print(f"Go parser     - Avg: {go_stats['avg_time']:.4f}s, Min: {go_stats['min_time']:.4f}s, Max: {go_stats['max_time']:.4f}s")

        # Calculate performance ratio
        ratio = python_stats['avg_time'] / go_stats['avg_time']
        print(f"Performance ratio (Python/Go): {ratio:.2f}x")

        # Both should complete in reasonable time
        assert python_stats['avg_time'] < 30.0  # Should complete in under 30 seconds
        assert go_stats['avg_time'] < 30.0

    @pytest.mark.performance
    def test_link_extraction_performance(self, python_parser, go_parser):
        """Compare performance of link extraction."""
        print("\n=== Link Extraction Performance Test ===")

        # Create content rich in links
        link_rich_content = {
            "id": "link_perf_test",
            "content": """
Website links: https://example.com, https://test.org, https://benchmark.net
Email contacts: test@example.com, support@test.org, admin@benchmark.net
File paths: /path/to/file.txt, /data/results.csv, /tmp/benchmark.log
More websites: https://performance.example.com, https://speed.test.org
Additional emails: performance@example.com, speed@test.org, benchmark@example.com
More file paths: C:\\Data\\Files\\test.docx, /usr/local/bin/parser, /home/user/documents/report.pdf

""" * 50,  # Repeat 50 times for more links
            "metadata": {"source": "link_extraction_performance_test"}
        }

        # Benchmark Python parser
        python_stats = self._benchmark_parser(python_parser, link_rich_content)
        print(f"Python parser - Avg: {python_stats['avg_time']:.4f}s")

        # Benchmark Go parser
        go_stats = self._benchmark_parser(go_parser, link_rich_content)
        print(f"Go parser     - Avg: {go_stats['avg_time']:.4f}s")

        # Calculate performance ratio
        ratio = python_stats['avg_time'] / go_stats['avg_time']
        print(f"Performance ratio (Python/Go): {ratio:.2f}x")

    @pytest.mark.performance
    def test_date_number_extraction_performance(self, python_parser, go_parser):
        """Compare performance of date and number extraction."""
        print("\n=== Date/Number Extraction Performance Test ===")

        # Create content rich in dates and numbers
        date_number_content = {
            "id": "date_number_perf_test",
            "content": """
Dates: 2024-01-15, 2024-02-28, 2024-12-31, January 1, 2024, March 15, 2024, December 25, 2024
Numbers: 123, 456.78, 99.99%, 1,234,567, 0.045, 3.14159, 100.0, 42
Financial: $1,234.56, €987.65, ¥10,000, £543.21, $99.99, €75.50
Percentages: 95.5%, 12.3%, 0.05%, 100%, 150.75%, 33.33%
Measurements: 123.45 cm, 67.89 kg, 98.6°F, 37.0°C, 1.75 m, 85.5 mph

""" * 100,  # Repeat 100 times for more data
            "metadata": {"source": "date_number_extraction_performance_test"}
        }

        # Benchmark Python parser
        python_stats = self._benchmark_parser(python_parser, date_number_content)
        print(f"Python parser - Avg: {python_stats['avg_time']:.4f}s")

        # Benchmark Go parser
        go_stats = self._benchmark_parser(go_parser, date_number_content)
        print(f"Go parser     - Avg: {go_stats['avg_time']:.4f}s")

        # Calculate performance ratio
        ratio = python_stats['avg_time'] / go_stats['avg_time']
        print(f"Performance ratio (Python/Go): {ratio:.2f}x")

    @pytest.mark.performance
    def test_paragraph_separation_performance(self, python_parser, go_parser):
        """Compare performance with different paragraph separators."""
        print("\n=== Paragraph Separation Performance Test ===")

        # Configure both parsers with custom separator
        python_parser.paragraph_separator = "\n---\n"
        go_parser.paragraph_separator = "\n---\n"

        # Create content with custom separators
        separator_content = {
            "id": "separator_perf_test",
            "content": "Paragraph content with various text and elements. " * 20 + "\n---\n" * 200,  # Many sections
            "metadata": {"source": "separator_performance_test"}
        }

        # Benchmark Python parser
        python_stats = self._benchmark_parser(python_parser, separator_content)
        print(f"Python parser - Avg: {python_stats['avg_time']:.4f}s")

        # Benchmark Go parser
        go_stats = self._benchmark_parser(go_parser, separator_content)
        print(f"Go parser     - Avg: {go_stats['avg_time']:.4f}s")

        # Calculate performance ratio
        ratio = python_stats['avg_time'] / go_stats['avg_time']
        print(f"Performance ratio (Python/Go): {ratio:.2f}x")

    @pytest.mark.performance
    def test_unicode_performance(self, python_parser, go_parser):
        """Compare performance with unicode-heavy content."""
        print("\n=== Unicode Performance Test ===")

        # Create unicode-rich content
        unicode_content = {
            "id": "unicode_perf_test",
            "content": """
Unicode text: αβγδεζηθικλμνξοπρστυφχψω ΑΒΓΔΕΖΗΘΙΚΛΜΝΞΟΠΡΣΤΥΦΧΨΩ
Chinese: 你好世界 这是一个测试 中文内容处理
Japanese: こんにちは世界 これはテストです 日本語の内容
Korean: 안녕하세요 세계 이것은 테스트입니다 한국어 내용
Arabic: مرحبا بالعالم هذا اختبار المحتوى العربي
Russian: Привет мир это тест русского контента
Emojis: 🚀🎯📊💡🔍📈📉💻🌟⭐️🎉🎊🎁🎈
Symbols: ★☆♠♣♥♦●○◆◇□■△▲▽▼◄►▲▼
Math: ∑∏∫∮∇∂∆√∞≈≠≤≥±×÷∝∴∵∈∉⊂⊃∪∩

""" * 50,  # Repeat for more unicode content
            "metadata": {"source": "unicode_performance_test"}
        }

        # Benchmark Python parser
        python_stats = self._benchmark_parser(python_parser, unicode_content)
        print(f"Python parser - Avg: {python_stats['avg_time']:.4f}s")

        # Benchmark Go parser
        go_stats = self._benchmark_parser(go_parser, unicode_content)
        print(f"Go parser     - Avg: {go_stats['avg_time']:.4f}s")

        # Calculate performance ratio
        ratio = python_stats['avg_time'] / go_stats['avg_time']
        print(f"Performance ratio (Python/Go): {ratio:.2f}x")

    @pytest.mark.performance
    def test_memory_usage_estimation(self, python_parser, go_parser, large_text_content):
        """Estimate memory usage patterns (basic test)."""
        print("\n=== Memory Usage Estimation ===")

        # Parse with Python
        python_result = python_parser.parse(large_text_content)
        python_elements = len(python_result["elements"])
        python_relationships = len(python_result["relationships"])
        python_links = len(python_result.get("links", []))

        print(f"Python parser results:")
        print(f"  Elements: {python_elements}")
        print(f"  Relationships: {python_relationships}")
        print(f"  Links: {python_links}")

        # Parse with Go
        go_result = go_parser.parse(large_text_content)
        go_elements = len(go_result["elements"])
        go_relationships = len(go_result["relationships"])
        go_links = len(go_result.get("links", []))

        print(f"Go parser results:")
        print(f"  Elements: {go_elements}")
        print(f"  Relationships: {go_relationships}")
        print(f"  Links: {go_links}")

        # Basic memory estimation (very rough)
        python_estimated_memory = (python_elements * 500 + python_relationships * 200 + python_links * 100)
        go_estimated_memory = (go_elements * 500 + go_relationships * 200 + go_links * 100)

        print(f"Estimated memory usage:")
        print(f"  Python: ~{python_estimated_memory / 1024:.1f} KB")
        print(f"  Go: ~{go_estimated_memory / 1024:.1f} KB")

    @pytest.mark.performance
    def test_scalability_comparison(self, python_parser, go_parser):
        """Test scalability with varying content sizes."""
        print("\n=== Scalability Comparison ===")

        sizes = [10, 50, 100, 200]  # Number of paragraphs
        python_times = []
        go_times = []

        for size in sizes:
            # Create content of varying size
            paragraph = "This is a test paragraph with some content. " * 10
            content = {
                "id": f"scalability_test_{size}",
                "content": "\n\n".join([paragraph] * size),
                "metadata": {"source": f"scalability_test_{size}"}
            }

            # Benchmark both parsers
            python_stats = self._benchmark_parser(python_parser, content, iterations=3)
            go_stats = self._benchmark_parser(go_parser, content, iterations=3)

            python_times.append(python_stats['avg_time'])
            go_times.append(go_stats['avg_time'])

            print(f"Size {size:3d} paragraphs - Python: {python_stats['avg_time']:.4f}s, Go: {go_stats['avg_time']:.4f}s")

        # Print scalability summary
        print("\nScalability Summary:")
        for i, size in enumerate(sizes):
            ratio = python_times[i] / go_times[i] if go_times[i] > 0 else float('inf')
            print(f"  {size:3d} paragraphs: {ratio:.2f}x (Python/Go)")

        # Check that performance scales reasonably
        for i in range(1, len(python_times)):
            # Time should increase with size, but not exponentially
            python_ratio = python_times[i] / python_times[i-1]
            go_ratio = go_times[i] / go_times[i-1]
            assert python_ratio < 10.0, f"Python performance degraded too much: {python_ratio:.2f}x"
            assert go_ratio < 10.0, f"Go performance degraded too much: {go_ratio:.2f}x"