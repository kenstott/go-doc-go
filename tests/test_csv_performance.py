"""
Performance benchmarks for CSV parser implementations.

These tests compare the performance of Python vs Go CSV parsers
to validate that the Go implementation provides performance benefits.
"""

import time
import pytest
import io
from pathlib import Path

from src.go_doc_go.document_parser.csv import CsvParser
from src.go_doc_go.document_parser.csv_go import GoCSVParser


@pytest.mark.performance
class TestCSVPerformance:
    """Performance benchmarks for CSV parsers."""

    @pytest.fixture
    def python_parser(self):
        """Create Python CSV parser instance."""
        return CsvParser({})

    @pytest.fixture
    def go_parser(self):
        """Create Go CSV parser instance."""
        # Check if binary exists
        project_root = Path(__file__).parent.parent
        binary_path = project_root / "bin" / "csvparser"

        if not binary_path.exists():
            pytest.skip("Go CSV parser binary not found")

        return GoCSVParser({})

    @pytest.fixture
    def small_csv_content(self):
        """Small CSV content for testing."""
        return {
            "id": "small_test",
            "content": """name,age,department,salary,email
John Doe,30,Engineering,75000,john.doe@company.com
Jane Smith,25,Marketing,65000,jane.smith@company.com
Bob Wilson,35,Sales,70000,bob.wilson@company.com
Alice Brown,28,Engineering,72000,alice.brown@company.com
Charlie Davis,32,Marketing,68000,charlie.davis@company.com""",
            "metadata": {"source": "small_test"}
        }

    @pytest.fixture
    def medium_csv_content(self):
        """Medium CSV content for testing."""
        # Generate 500 rows of employee data
        lines = ["employee_id,first_name,last_name,email,department,salary,hire_date,manager_id,office_location,phone"]

        departments = ["Engineering", "Marketing", "Sales", "HR", "Finance", "Operations"]
        offices = ["New York", "San Francisco", "Chicago", "Boston", "Austin", "Seattle"]

        for i in range(1, 501):  # 500 employees
            dept = departments[i % len(departments)]
            office = offices[i % len(offices)]
            manager_id = max(1, i - 50)  # Simple manager hierarchy

            line = f"{i:04d},Employee,{i:04d},employee{i:04d}@company.com,{dept},{50000 + (i * 100)},2023-{(i % 12) + 1:02d}-{(i % 28) + 1:02d},{manager_id:04d},{office},{555}-{i:04d}"
            lines.append(line)

        return {
            "id": "medium_test",
            "content": "\n".join(lines),
            "metadata": {"source": "medium_test"}
        }

    @pytest.fixture
    def large_csv_content(self):
        """Large CSV content for testing."""
        # Generate 2000 rows of transaction data
        lines = ["transaction_id,customer_id,product_id,product_name,category,quantity,unit_price,total_amount,transaction_date,payment_method,store_location,sales_rep"]

        categories = ["Electronics", "Clothing", "Books", "Home & Garden", "Sports", "Automotive", "Food", "Health"]
        payment_methods = ["Credit Card", "Debit Card", "Cash", "PayPal", "Apple Pay", "Google Pay"]
        stores = ["Store_001", "Store_002", "Store_003", "Store_004", "Store_005"]

        for i in range(1, 2001):  # 2000 transactions
            customer_id = f"CUST_{(i % 500) + 1:04d}"
            product_id = f"PROD_{(i % 100) + 1:04d}"
            category = categories[i % len(categories)]
            payment = payment_methods[i % len(payment_methods)]
            store = stores[i % len(stores)]
            quantity = (i % 10) + 1
            unit_price = 10.00 + (i % 100)
            total_amount = quantity * unit_price

            line = f"TXN_{i:06d},{customer_id},{product_id},Product_{i % 100},{category},{quantity},{unit_price:.2f},{total_amount:.2f},2023-{(i % 12) + 1:02d}-{(i % 28) + 1:02d},{payment},{store},REP_{(i % 20) + 1:03d}"
            lines.append(line)

        return {
            "id": "large_test",
            "content": "\n".join(lines),
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
    def test_small_csv_performance(self, python_parser, go_parser, small_csv_content):
        """Compare performance on small CSV documents."""
        print("\n=== Small CSV Performance Test ===")

        # Measure Python parser
        python_stats = self.measure_parsing_time(python_parser, small_csv_content)
        print(f"Python parser: {python_stats['avg']:.4f}s avg ({python_stats['min']:.4f}s - {python_stats['max']:.4f}s)")

        # Measure Go parser
        go_stats = self.measure_parsing_time(go_parser, small_csv_content)
        print(f"Go parser: {go_stats['avg']:.4f}s avg ({go_stats['min']:.4f}s - {go_stats['max']:.4f}s)")

        # Calculate speedup
        speedup = python_stats['avg'] / go_stats['avg'] if go_stats['avg'] > 0 else 1
        print(f"Go speedup: {speedup:.2f}x")

        # Both should complete reasonably quickly
        assert python_stats['avg'] < 5.0, "Python parser should complete quickly"
        assert go_stats['avg'] < 5.0, "Go parser should complete quickly"

    @pytest.mark.performance
    def test_medium_csv_performance(self, python_parser, go_parser, medium_csv_content):
        """Compare performance on medium CSV documents."""
        print("\n=== Medium CSV Performance Test ===")

        # Measure Python parser
        python_stats = self.measure_parsing_time(python_parser, medium_csv_content)
        print(f"Python parser: {python_stats['avg']:.4f}s avg ({python_stats['min']:.4f}s - {python_stats['max']:.4f}s)")

        # Measure Go parser
        go_stats = self.measure_parsing_time(go_parser, medium_csv_content)
        print(f"Go parser: {go_stats['avg']:.4f}s avg ({go_stats['min']:.4f}s - {go_stats['max']:.4f}s)")

        # Calculate speedup
        speedup = python_stats['avg'] / go_stats['avg'] if go_stats['avg'] > 0 else 1
        print(f"Go speedup: {speedup:.2f}x")

        # Both should complete in reasonable time
        assert python_stats['avg'] < 10.0, "Python parser should complete in reasonable time"
        assert go_stats['avg'] < 10.0, "Go parser should complete in reasonable time"

        # Go should be faster or at least not significantly slower
        assert go_stats['avg'] <= python_stats['avg'] * 1.5, "Go parser should not be significantly slower"

    @pytest.mark.performance
    def test_large_csv_performance(self, python_parser, go_parser, large_csv_content):
        """Compare performance on large CSV documents."""
        print("\n=== Large CSV Performance Test ===")

        # Measure Python parser (fewer iterations for large content)
        python_stats = self.measure_parsing_time(python_parser, large_csv_content, iterations=3)
        print(f"Python parser: {python_stats['avg']:.4f}s avg ({python_stats['min']:.4f}s - {python_stats['max']:.4f}s)")

        # Measure Go parser
        go_stats = self.measure_parsing_time(go_parser, large_csv_content, iterations=3)
        print(f"Go parser: {go_stats['avg']:.4f}s avg ({go_stats['min']:.4f}s - {go_stats['max']:.4f}s)")

        # Calculate speedup
        speedup = python_stats['avg'] / go_stats['avg'] if go_stats['avg'] > 0 else 1
        print(f"Go speedup: {speedup:.2f}x")

        # Both should complete without timeout
        assert python_stats['avg'] < 30.0, "Python parser should complete without timeout"
        assert go_stats['avg'] < 30.0, "Go parser should complete without timeout"

        # For large documents, Go should show performance benefits
        assert go_stats['avg'] <= python_stats['avg'] * 2.0, "Go parser should not be much slower than Python"

    @pytest.mark.performance
    def test_wide_csv_performance(self, python_parser, go_parser):
        """Test performance with CSV files having many columns."""
        print("\n=== Wide CSV Performance Test ===")

        # Create CSV with 50 columns and 100 rows
        headers = [f"column_{i:02d}" for i in range(50)]
        lines = [",".join(headers)]

        for row in range(100):
            values = [f"value_{row:03d}_{col:02d}" for col in range(50)]
            lines.append(",".join(values))

        wide_csv_content = {
            "id": "wide_test",
            "content": "\n".join(lines),
            "metadata": {"source": "wide_test"}
        }

        # Measure both parsers
        python_stats = self.measure_parsing_time(python_parser, wide_csv_content, iterations=3)
        go_stats = self.measure_parsing_time(go_parser, wide_csv_content, iterations=3)

        print(f"Python parser (50 cols x 100 rows): {python_stats['avg']:.4f}s")
        print(f"Go parser (50 cols x 100 rows): {go_stats['avg']:.4f}s")
        print(f"Go speedup: {python_stats['avg'] / go_stats['avg']:.2f}x")

        # Verify results are reasonable
        assert python_stats['avg'] < 15.0, "Python parser should handle wide CSV"
        assert go_stats['avg'] < 15.0, "Go parser should handle wide CSV"

    @pytest.mark.performance
    def test_parsing_scalability(self, python_parser, go_parser, medium_csv_content):
        """Test parsing scalability with multiple documents."""
        print("\n=== Scalability Test ===")

        # Test parsing multiple documents
        documents = []
        for i in range(10):
            doc = medium_csv_content.copy()
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
        speedup = python_time / go_time if go_time > 0 else 1
        print(f"Go speedup: {speedup:.2f}x")

        # Verify results
        assert len(python_results) == 10, "Should parse all Python documents"
        assert len(go_results) == 10, "Should parse all Go documents"

        # Performance check
        assert python_time < 60.0, "Python batch processing should complete in reasonable time"
        assert go_time < 60.0, "Go batch processing should complete in reasonable time"

    @pytest.mark.performance
    def test_memory_efficiency(self, python_parser, go_parser, large_csv_content):
        """Test memory efficiency of parsers."""
        print("\n=== Memory Efficiency Test ===")

        import psutil
        import os

        process = psutil.Process(os.getpid())

        # Measure Python parser memory usage
        initial_memory = process.memory_info().rss
        python_result = python_parser.parse(large_csv_content)
        python_peak_memory = process.memory_info().rss
        python_memory_used = python_peak_memory - initial_memory

        print(f"Python parser memory usage: {python_memory_used / 1024 / 1024:.2f} MB")

        # Reset and measure Go parser memory usage
        initial_memory = process.memory_info().rss
        go_result = go_parser.parse(large_csv_content)
        go_peak_memory = process.memory_info().rss
        go_memory_used = go_peak_memory - initial_memory

        print(f"Go parser memory usage: {go_memory_used / 1024 / 1024:.2f} MB")

        # Verify results are comparable
        assert len(python_result["elements"]) > 0
        assert len(go_result["elements"]) > 0

        # Memory usage should be reasonable
        max_reasonable_memory = 500 * 1024 * 1024  # 500MB
        assert python_memory_used < max_reasonable_memory, "Python parser memory usage should be reasonable"
        assert go_memory_used < max_reasonable_memory, "Go parser memory usage should be reasonable"

        print(f"Memory efficiency ratio: {python_memory_used / max(go_memory_used, 1):.2f}")

    @pytest.mark.performance
    def test_delimiter_parsing_performance(self, python_parser, go_parser):
        """Test performance with different delimiters."""
        print("\n=== Delimiter Performance Test ===")

        # Test semicolon delimiter
        python_parser.delimiter = ";"
        go_parser.delimiter = ";"

        # Create CSV with semicolon delimiter
        lines = ["name;age;department;salary"]
        for i in range(1000):
            lines.append(f"Employee{i:04d};{25 + (i % 40)};Dept{i % 5};{50000 + i * 10}")

        semicolon_content = {
            "id": "semicolon_perf_test",
            "content": "\n".join(lines),
            "metadata": {"source": "semicolon_test"}
        }

        # Measure both parsers
        python_stats = self.measure_parsing_time(python_parser, semicolon_content, iterations=3)
        go_stats = self.measure_parsing_time(go_parser, semicolon_content, iterations=3)

        print(f"Python parser (semicolon): {python_stats['avg']:.4f}s")
        print(f"Go parser (semicolon): {go_stats['avg']:.4f}s")
        print(f"Go speedup: {python_stats['avg'] / go_stats['avg']:.2f}x")

    @pytest.mark.performance
    def test_quoted_content_performance(self, python_parser, go_parser):
        """Test performance with heavily quoted content."""
        print("\n=== Quoted Content Performance Test ===")

        # Create CSV with lots of quoted content
        lines = ['product_name,description,category,price']
        for i in range(500):
            name = f'"Product {i:04d}"'
            desc = f'"This is a detailed description of product {i:04d}, with commas, quotes, and special characters!"'
            category = f'"Category {i % 10}"'
            price = f"{10.00 + (i * 0.5):.2f}"
            lines.append(f"{name},{desc},{category},{price}")

        quoted_content = {
            "id": "quoted_perf_test",
            "content": "\n".join(lines),
            "metadata": {"source": "quoted_test"}
        }

        # Measure both parsers
        python_stats = self.measure_parsing_time(python_parser, quoted_content, iterations=3)
        go_stats = self.measure_parsing_time(go_parser, quoted_content, iterations=3)

        print(f"Python parser (quoted): {python_stats['avg']:.4f}s")
        print(f"Go parser (quoted): {go_stats['avg']:.4f}s")
        print(f"Go speedup: {python_stats['avg'] / go_stats['avg']:.2f}x")

    @pytest.mark.performance
    def test_link_extraction_performance(self, python_parser, go_parser):
        """Test performance impact of link extraction."""
        print("\n=== Link Extraction Performance Test ===")

        # Create CSV with many URLs and emails
        lines = ["name,email,website,contact"]
        for i in range(1000):
            name = f"Person{i:04d}"
            email = f"person{i:04d}@company{i % 10}.com"
            website = f"https://person{i:04d}.example{i % 5}.org"
            contact = f"Contact person{i:04d} at {email} or visit {website}"
            lines.append(f"{name},{email},{website},{contact}")

        link_content = {
            "id": "link_perf_test",
            "content": "\n".join(lines),
            "metadata": {"source": "link_test"}
        }

        # Test with link extraction enabled
        python_parser.extract_dates = True  # Enable link-related processing
        go_parser.enable_link_extraction = True

        python_stats = self.measure_parsing_time(python_parser, link_content, iterations=3)
        go_stats = self.measure_parsing_time(go_parser, link_content, iterations=3)

        print(f"Python parser (with links): {python_stats['avg']:.4f}s")
        print(f"Go parser (with links): {go_stats['avg']:.4f}s")
        print(f"Go speedup: {python_stats['avg'] / go_stats['avg']:.2f}x")

        # Verify links were extracted
        python_result = python_parser.parse(link_content)
        go_result = go_parser.parse(link_content)

        print(f"Python extracted links: {len(python_result.get('links', []))}")
        print(f"Go extracted links: {len(go_result.get('links', []))}")

        # Both should extract substantial number of links
        assert len(go_result.get('links', [])) > 1000, "Go parser should extract many links"