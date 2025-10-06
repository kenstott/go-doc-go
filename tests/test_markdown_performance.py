"""
Performance benchmarks for Python vs Go markdown parser implementations.

These tests measure parsing performance across different markdown sizes and complexities.
"""

import pytest
import time
from pathlib import Path
from typing import Dict, Any

from src.go_doc_go.document_parser.markdown import MarkdownParser
from src.go_doc_go.document_parser.markdown_go import GoMarkdownParser


@pytest.mark.performance
class TestMarkdownPerformance:
    """Performance benchmarks for markdown parsers."""

    @pytest.fixture
    def python_parser(self):
        """Create Python markdown parser instance."""
        return MarkdownParser({})

    @pytest.fixture
    def go_parser(self):
        """Create Go markdown parser instance."""
        # Check if binary exists
        project_root = Path(__file__).parent.parent
        binary_path = project_root / "bin" / "markdownparser"

        if not binary_path.exists():
            pytest.skip("Go markdown parser binary not found")

        return GoMarkdownParser({})

    @pytest.fixture
    def small_markdown_content(self):
        """Small markdown content for performance testing."""
        return {
            "id": "small_md_perf",
            "content": """# Small Document

This is a small markdown document for performance testing.

## Features

- Simple list item
- Another item

That's it!""",
            "metadata": {"source": "small_markdown_performance_test"}
        }

    @pytest.fixture
    def medium_markdown_content(self):
        """Medium markdown content for performance testing."""
        sections = []
        for i in range(20):
            section = f"""
## Section {i+1}

This is section {i+1} with some content including dates like 2024-{i+1:02d}-15,
numbers like {(i+1)*100}, URLs like https://section{i+1}.example.com,
and email addresses like section{i+1}@example.com.

### Subsection {i+1}.1

- Item 1 for section {i+1}
- Item 2 for section {i+1} with [link](https://link{i+1}.example.com)
- Item 3 for section {i+1}

```python
def function_for_section_{i+1}():
    print(f"This is code for section {i+1}")
    return {i+1} * 100
```

> This is a blockquote for section {i+1}
> with multiple lines of content.

| Column 1 | Column 2 | Column 3 |
|----------|----------|----------|
| Value {i+1}  | Data {i+1}   | Info {i+1}   |
"""
            sections.append(section)

        return {
            "id": "medium_md_perf",
            "content": "# Medium Document\n\n" + "\n".join(sections),
            "metadata": {"source": "medium_markdown_performance_test"}
        }

    @pytest.fixture
    def large_markdown_content(self):
        """Large markdown content for performance testing."""
        front_matter = """---
title: Large Performance Test Document
author: Performance Tester
created: 2024-01-15
tags: [performance, testing, markdown, large]
description: A large document for testing markdown parsing performance
version: 1.0
---

"""

        sections = []
        for i in range(100):
            section = f"""
# Chapter {i+1}: Advanced Topic {i+1}

This is chapter {i+1} covering advanced topics in performance testing.
Created on {2024}-{(i % 12) + 1:02d}-{(i % 28) + 1:02d}, this section contains
{(i+1)*1000} words of comprehensive content.

## {i+1}.1 Introduction

Performance testing is crucial for applications handling large markdown files.
Contact us at chapter{i+1}@performance.test or visit https://chapter{i+1}.performance.example.com
for more information about this topic.

### {i+1}.1.1 Key Concepts

The following concepts are important:

1. **Parsing Speed**: How quickly can we process {(i+1)*100} elements?
2. **Memory Usage**: Efficient handling of {(i+1)*50} MB of content
3. **Scalability**: Supporting up to {(i+1)*10000} concurrent operations

### {i+1}.1.2 Implementation Details

```python
class PerformanceTest{i+1}:
    def __init__(self, chapter_number={i+1}):
        self.chapter = chapter_number
        self.items = {(i+1)*10}
        self.created = "2024-{(i % 12) + 1:02d}-{(i % 28) + 1:02d}"

    def process_data(self):
        results = []
        for item in range(self.items):
            result = self.calculate_performance(item)
            results.append(result)
        return results

    def calculate_performance(self, item_id):
        # Complex calculation for item {i+1}
        return item_id * {i+1} * 1.{i:02d}
```

### {i+1}.1.3 Performance Metrics

> **Important Note**: Chapter {i+1} shows {(i+1)*25}% improvement
> over previous implementations when processing large datasets
> containing {(i+1)*500} elements with {(i+1)*1.5} complexity factor.

#### Benchmark Results for Chapter {i+1}

| Metric | Value | Unit | Improvement |
|--------|-------|------|-------------|
| Throughput | {(i+1)*1000} | ops/sec | +{i+1}% |
| Latency | {(i+1)*0.1:.1f} | ms | -{i+1}% |
| Memory | {(i+1)*10} | MB | -{i*2}% |
| CPU Usage | {(i+1)*2} | % | -{i}% |

## {i+1}.2 Advanced Features

Additional features include:

- **Feature A{i+1}**: Enhanced processing with {(i+1)*100} operations per second
- **Feature B{i+1}**: Improved memory usage saving {i+1} GB per hour
- **Feature C{i+1}**: Better error handling with {(i+1)*5} error types covered

### Code Examples for Chapter {i+1}

```javascript
const chapter{i+1} = {{
    id: {i+1},
    name: "Chapter {i+1}",
    performance: {(i+1)*100},
    features: [{i+1}, {i+2}, {i+3}],

    async processData(data) {{
        const results = await Promise.all(
            data.map(item => this.processItem(item, {i+1}))
        );
        return results.filter(r => r.score > {i+1});
    }}
}};
```

### {i+1}.2.1 Edge Cases

Special handling is required for:

1. Files larger than {(i+1)*100} MB
2. Documents with more than {(i+1)*1000} elements
3. Complex nesting deeper than {i+1} levels
4. Unicode content with {(i+1)*50} different character sets

> **Warning**: Chapter {i+1} processing may require up to {(i+1)*2} GB RAM
> for optimal performance when handling datasets exceeding {(i+1)*10000} records.

### {i+1}.2.2 Best Practices

For optimal results in chapter {i+1}:

- Allocate {(i+1)*512} MB memory pool
- Use chunk size of {(i+1)*64} KB for file processing
- Set timeout to {(i+1)*30} seconds for complex operations
- Enable caching for objects smaller than {(i+1)*4} MB

## {i+1}.3 Conclusion

Chapter {i+1} demonstrates significant improvements with {(i+1)*150}% better performance
compared to baseline measurements. The optimizations result in {i+1}x faster processing
while reducing memory footprint by {i*10}%.

For questions about chapter {i+1}, email support{i+1}@performance.test
or visit the documentation at https://docs.chapter{i+1}.performance.example.com.

---
"""
            sections.append(section)

        return {
            "id": "large_md_perf",
            "content": front_matter + "\n".join(sections),
            "metadata": {"source": "large_markdown_performance_test"}
        }

    def _benchmark_parser(self, parser, content: Dict[str, Any], iterations: int = 3) -> Dict[str, float]:
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
    def test_small_markdown_performance(self, python_parser, go_parser, small_markdown_content):
        """Compare performance on small markdown files."""
        print("\n=== Small Markdown Performance Test ===")

        # Benchmark Python parser
        python_stats = self._benchmark_parser(python_parser, small_markdown_content)
        print(f"Python parser - Avg: {python_stats['avg_time']:.4f}s, Min: {python_stats['min_time']:.4f}s, Max: {python_stats['max_time']:.4f}s")

        # Benchmark Go parser
        go_stats = self._benchmark_parser(go_parser, small_markdown_content)
        print(f"Go parser     - Avg: {go_stats['avg_time']:.4f}s, Min: {go_stats['min_time']:.4f}s, Max: {go_stats['max_time']:.4f}s")

        # Calculate performance ratio
        ratio = python_stats['avg_time'] / go_stats['avg_time']
        print(f"Performance ratio (Python/Go): {ratio:.2f}x")

        # Both should complete in reasonable time
        assert python_stats['avg_time'] < 5.0
        assert go_stats['avg_time'] < 5.0

    @pytest.mark.performance
    def test_medium_markdown_performance(self, python_parser, go_parser, medium_markdown_content):
        """Compare performance on medium markdown files."""
        print("\n=== Medium Markdown Performance Test ===")

        # Benchmark Python parser
        python_stats = self._benchmark_parser(python_parser, medium_markdown_content)
        print(f"Python parser - Avg: {python_stats['avg_time']:.4f}s, Min: {python_stats['min_time']:.4f}s, Max: {python_stats['max_time']:.4f}s")

        # Benchmark Go parser
        go_stats = self._benchmark_parser(go_parser, medium_markdown_content)
        print(f"Go parser     - Avg: {go_stats['avg_time']:.4f}s, Min: {go_stats['min_time']:.4f}s, Max: {go_stats['max_time']:.4f}s")

        # Calculate performance ratio
        ratio = python_stats['avg_time'] / go_stats['avg_time']
        print(f"Performance ratio (Python/Go): {ratio:.2f}x")

        # Both should complete in reasonable time
        assert python_stats['avg_time'] < 15.0
        assert go_stats['avg_time'] < 15.0

    @pytest.mark.performance
    def test_large_markdown_performance(self, python_parser, go_parser, large_markdown_content):
        """Compare performance on large markdown files."""
        print("\n=== Large Markdown Performance Test ===")

        # Use fewer iterations for large files
        iterations = 2

        # Benchmark Python parser
        python_stats = self._benchmark_parser(python_parser, large_markdown_content, iterations)
        print(f"Python parser - Avg: {python_stats['avg_time']:.4f}s, Min: {python_stats['min_time']:.4f}s, Max: {python_stats['max_time']:.4f}s")

        # Benchmark Go parser
        go_stats = self._benchmark_parser(go_parser, large_markdown_content, iterations)
        print(f"Go parser     - Avg: {go_stats['avg_time']:.4f}s, Min: {go_stats['min_time']:.4f}s, Max: {go_stats['max_time']:.4f}s")

        # Calculate performance ratio
        ratio = python_stats['avg_time'] / go_stats['avg_time']
        print(f"Performance ratio (Python/Go): {ratio:.2f}x")

        # Both should complete in reasonable time
        assert python_stats['avg_time'] < 60.0
        assert go_stats['avg_time'] < 60.0

    @pytest.mark.performance
    def test_front_matter_parsing_performance(self, python_parser, go_parser):
        """Compare performance of front matter parsing."""
        print("\n=== Front Matter Parsing Performance Test ===")

        # Create content with complex front matter
        front_matter_content = {
            "id": "front_matter_perf",
            "content": """---
title: Complex Front Matter Test
authors:
  - Primary Author
  - Secondary Author
  - Third Author
created: 2024-01-15T10:30:00Z
modified: 2024-01-20T15:45:30Z
tags:
  - performance
  - testing
  - markdown
  - front-matter
  - yaml
categories:
  - documentation
  - technical
status: published
version: 2.1.3
metadata:
  word_count: 1500
  reading_time: 8
  difficulty: intermediate
  topics:
    - parsing
    - performance
    - optimization
settings:
  enable_toc: true
  enable_comments: false
  enable_sharing: true
  template: technical
custom_fields:
  priority: high
  review_status: approved
  last_reviewer: technical-team
  review_date: 2024-01-18
---

# Main Content

This document tests front matter parsing performance with complex YAML structures.

The front matter above contains nested objects, arrays, and various data types
to stress test the YAML parsing capabilities of both implementations.

## Performance Considerations

When parsing complex front matter:

1. YAML parsing overhead
2. Object serialization costs
3. Memory allocation patterns
4. Type conversion efficiency

That's the end of the test document.
""" * 10,  # Repeat to make it larger
            "metadata": {"source": "front_matter_performance_test"}
        }

        # Benchmark Python parser
        python_stats = self._benchmark_parser(python_parser, front_matter_content)
        print(f"Python parser - Avg: {python_stats['avg_time']:.4f}s")

        # Benchmark Go parser
        go_stats = self._benchmark_parser(go_parser, front_matter_content)
        print(f"Go parser     - Avg: {go_stats['avg_time']:.4f}s")

        # Calculate performance ratio
        ratio = python_stats['avg_time'] / go_stats['avg_time']
        print(f"Performance ratio (Python/Go): {ratio:.2f}x")

    @pytest.mark.performance
    def test_code_block_parsing_performance(self, python_parser, go_parser):
        """Compare performance of code block parsing."""
        print("\n=== Code Block Parsing Performance Test ===")

        # Create content with many code blocks
        code_blocks = []
        for i in range(50):
            code_block = f"""
### Code Example {i+1}

```python
def function_{i+1}():
    \"\"\"This is function {i+1} for performance testing.\"\"\"
    data = []
    for j in range({(i+1)*10}):
        value = j * {i+1} + {i+1}
        data.append({{
            'id': j,
            'value': value,
            'processed': True,
            'timestamp': '2024-01-{(i % 28) + 1:02d}'
        }})
    return data

class TestClass{i+1}:
    def __init__(self):
        self.data = function_{i+1}()
        self.count = {(i+1)*100}

    def process(self):
        results = []
        for item in self.data:
            if item['value'] > {i+1}:
                results.append(item)
        return results
```

```javascript
const config{i+1} = {{
    id: {i+1},
    name: "Test Config {i+1}",
    values: [{', '.join(str(j) for j in range(i+1, i+11))}],

    process: function() {{
        return this.values.map(v => v * {i+1});
    }}
}};
```

```sql
SELECT
    id,
    name,
    value * {i+1} as calculated_value,
    created_date
FROM test_table_{i+1}
WHERE value > {i+1}
    AND created_date >= '2024-01-{(i % 28) + 1:02d}'
ORDER BY calculated_value DESC
LIMIT {(i+1)*10};
```
"""
            code_blocks.append(code_block)

        code_heavy_content = {
            "id": "code_block_perf",
            "content": "# Code Block Performance Test\n\n" + "\n".join(code_blocks),
            "metadata": {"source": "code_block_performance_test"}
        }

        # Benchmark Python parser
        python_stats = self._benchmark_parser(python_parser, code_heavy_content)
        print(f"Python parser - Avg: {python_stats['avg_time']:.4f}s")

        # Benchmark Go parser
        go_stats = self._benchmark_parser(go_parser, code_heavy_content)
        print(f"Go parser     - Avg: {go_stats['avg_time']:.4f}s")

        # Calculate performance ratio
        ratio = python_stats['avg_time'] / go_stats['avg_time']
        print(f"Performance ratio (Python/Go): {ratio:.2f}x")

    @pytest.mark.performance
    def test_link_heavy_performance(self, python_parser, go_parser):
        """Compare performance with link-heavy content."""
        print("\n=== Link-Heavy Content Performance Test ===")

        # Create content with many links
        link_sections = []
        for i in range(30):
            section = f"""
## Section {i+1} - Links and References

This section contains many links for testing:

- Website: [Site {i+1}](https://site{i+1}.example.com)
- Documentation: [Docs {i+1}](https://docs{i+1}.example.com/guide)
- Repository: [Repo {i+1}](https://github.com/user{i+1}/project{i+1})
- Issue Tracker: [Issues {i+1}](https://github.com/user{i+1}/project{i+1}/issues)
- Wiki: [[Internal Page {i+1}]]
- Email: Contact us at contact{i+1}@example{i+1}.com
- Support: Get help at support{i+1}@example{i+1}.org
- API: [API Docs {i+1}](https://api{i+1}.example.com/v{i+1}/docs)

Additional references:
- [External Link {i+1}A](https://external{i+1}a.example.org/path)
- [External Link {i+1}B](https://external{i+1}b.example.net/resource)
- [External Link {i+1}C](https://external{i+1}c.example.info/data)

Internal references:
- [[Page {i+1}A]]
- [[Page {i+1}B]]
- [[Page {i+1}C]]

Email contacts:
- admin{i+1}@example{i+1}.com
- user{i+1}@example{i+1}.net
- test{i+1}@example{i+1}.org
"""
            link_sections.append(section)

        link_heavy_content = {
            "id": "link_heavy_perf",
            "content": "# Link Heavy Performance Test\n\n" + "\n".join(link_sections),
            "metadata": {"source": "link_heavy_performance_test"}
        }

        # Benchmark Python parser
        python_stats = self._benchmark_parser(python_parser, link_heavy_content)
        print(f"Python parser - Avg: {python_stats['avg_time']:.4f}s")

        # Benchmark Go parser
        go_stats = self._benchmark_parser(go_parser, link_heavy_content)
        print(f"Go parser     - Avg: {go_stats['avg_time']:.4f}s")

        # Calculate performance ratio
        ratio = python_stats['avg_time'] / go_stats['avg_time']
        print(f"Performance ratio (Python/Go): {ratio:.2f}x")

    @pytest.mark.performance
    def test_table_parsing_performance(self, python_parser, go_parser):
        """Compare performance of table parsing."""
        print("\n=== Table Parsing Performance Test ===")

        # Create content with large tables
        tables = []
        for i in range(20):
            rows = []
            # Header
            rows.append(f"| ID | Name | Value {i+1} | Category | Status | Date | Score |")
            rows.append("|----|----|----|----|----|----|----| ")

            # Data rows
            for j in range(25):
                rows.append(f"| {j+1} | Item {j+1} | {(j+1)*(i+1)} | Cat{(j % 5)+1} | Active | 2024-{((j % 12)+1):02d}-{((j % 28)+1):02d} | {(j+1)*10} |")

            table_content = f"""
### Table {i+1}: Performance Data

This table contains performance metrics for test {i+1}:

{chr(10).join(rows)}

Analysis of table {i+1} shows improvement in processing speed.
"""
            tables.append(table_content)

        table_heavy_content = {
            "id": "table_heavy_perf",
            "content": "# Table Heavy Performance Test\n\n" + "\n".join(tables),
            "metadata": {"source": "table_heavy_performance_test"}
        }

        # Benchmark Python parser
        python_stats = self._benchmark_parser(python_parser, table_heavy_content)
        print(f"Python parser - Avg: {python_stats['avg_time']:.4f}s")

        # Benchmark Go parser
        go_stats = self._benchmark_parser(go_parser, table_heavy_content)
        print(f"Go parser     - Avg: {go_stats['avg_time']:.4f}s")

        # Calculate performance ratio
        ratio = python_stats['avg_time'] / go_stats['avg_time']
        print(f"Performance ratio (Python/Go): {ratio:.2f}x")

    @pytest.mark.performance
    def test_scalability_comparison(self, python_parser, go_parser):
        """Test scalability with varying content sizes."""
        print("\n=== Scalability Comparison ===")

        sizes = [5, 15, 30, 50]  # Number of sections
        python_times = []
        go_times = []

        for size in sizes:
            # Create content of varying size
            sections = []
            for i in range(size):
                section = f"""
## Section {i+1}

Content for section {i+1} with various elements:

- List item {i+1}A
- List item {i+1}B
- List item {i+1}C

```code
Example code {i+1}
```

> Quote {i+1}

[Link {i+1}](https://example{i+1}.com)
"""
                sections.append(section)

            content = {
                "id": f"scalability_test_{size}",
                "content": f"# Scalability Test {size}\n\n" + "\n".join(sections),
                "metadata": {"source": f"scalability_test_{size}"}
            }

            # Benchmark both parsers
            python_stats = self._benchmark_parser(python_parser, content, iterations=2)
            go_stats = self._benchmark_parser(go_parser, content, iterations=2)

            python_times.append(python_stats['avg_time'])
            go_times.append(go_stats['avg_time'])

            print(f"Size {size:2d} sections - Python: {python_stats['avg_time']:.4f}s, Go: {go_stats['avg_time']:.4f}s")

        # Print scalability summary
        print("\nScalability Summary:")
        for i, size in enumerate(sizes):
            ratio = python_times[i] / go_times[i] if go_times[i] > 0 else float('inf')
            print(f"  {size:2d} sections: {ratio:.2f}x (Python/Go)")

        # Check that performance scales reasonably
        for i in range(1, len(python_times)):
            # Time should increase with size, but not exponentially
            python_ratio = python_times[i] / python_times[i-1]
            go_ratio = go_times[i] / go_times[i-1]
            assert python_ratio < 10.0, f"Python performance degraded too much: {python_ratio:.2f}x"
            assert go_ratio < 10.0, f"Go performance degraded too much: {go_ratio:.2f}x"