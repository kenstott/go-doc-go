"""Performance benchmark for Python vs Go XML parsers."""

import os
import time
from typing import Dict, Any

from src.go_doc_go.document_parser.factory import create_parser


def create_large_xml_content() -> Dict[str, Any]:
    """Create a large XML document for performance testing."""
    xml_content = '''<?xml version="1.0" encoding="UTF-8"?>
<catalog xmlns="http://example.com/catalog">'''
    
    # Generate many book entries
    for i in range(100):
        xml_content += f'''
    <book id="{i}" category="{'fiction' if i % 2 == 0 else 'non-fiction'}">
        <title>Book Title {i}</title>
        <authors>
            <author>
                <name>Author {i} Primary</name>
                <email>author{i}@example.com</email>
                <website>https://author{i}.example.com</website>
            </author>
            <author>
                <name>Author {i} Secondary</name>
                <email>author{i}b@example.com</email>
            </author>
        </authors>
        <publication>
            <year>{2020 + (i % 5)}</year>
            <publisher>Publisher {i % 10}</publisher>
            <isbn>978-{i:010d}</isbn>
            <website>https://publisher{i % 10}.example.com</website>
        </publication>
        <description>
            This is a detailed description of book {i}. It contains multiple sentences 
            to simulate real-world content. The book covers various topics including
            technology, science, and human behavior. Visit https://book{i}.example.com 
            for more information.
        </description>
        <metadata>
            <pages>{200 + (i * 10)}</pages>
            <language>English</language>
            <genre>{'Fiction' if i % 2 == 0 else 'Non-Fiction'}</genre>
            <price currency="USD">{9.99 + (i * 0.50)}</price>
        </metadata>
    </book>'''
    
    xml_content += '''
</catalog>'''
    
    return {
        'id': 'large_xml_performance_test',
        'content': xml_content,
        'metadata': {
            'source': 'performance_test',
            'filename': 'large_catalog.xml'
        }
    }


def benchmark_parser(parser_name: str, use_go: bool, content: Dict[str, Any], runs: int = 5) -> Dict[str, Any]:
    """Benchmark a parser with the given content."""
    os.environ['USE_GO_MODULES'] = 'true' if use_go else 'false'
    
    times = []
    results = []
    
    for i in range(runs):
        parser = create_parser('xml')
        
        start_time = time.time()
        result = parser.parse(content)
        end_time = time.time()
        
        parse_time = end_time - start_time
        times.append(parse_time)
        results.append(result)
    
    avg_time = sum(times) / len(times)
    min_time = min(times)
    max_time = max(times)
    
    # Get stats from first result
    first_result = results[0]
    
    return {
        'parser_name': parser_name,
        'parser_class': parser.__class__.__name__,
        'avg_time_ms': avg_time * 1000,
        'min_time_ms': min_time * 1000,
        'max_time_ms': max_time * 1000,
        'elements_parsed': len(first_result.get('elements', [])),
        'relationships_found': len(first_result.get('relationships', [])),
        'links_extracted': len(first_result.get('links', [])),
        'runs': runs
    }


def main():
    """Run performance benchmarks."""
    print("XML Parser Performance Benchmark")
    print("=" * 40)
    
    # Create test content
    print("Generating large XML content...")
    content = create_large_xml_content()
    content_size_kb = len(content['content']) / 1024
    print(f"XML content size: {content_size_kb:.1f} KB")
    print()
    
    # Benchmark Go parser
    print("Benchmarking Go XML parser...")
    go_results = benchmark_parser("Go XML Parser", True, content, runs=3)
    
    # Benchmark Python parser
    print("Benchmarking Python XML parser...")
    python_results = benchmark_parser("Python XML Parser", False, content, runs=3)
    
    # Print results
    print("\nPerformance Results:")
    print("-" * 50)
    
    print(f"Go XML Parser ({go_results['parser_class']}):")
    print(f"  Average time: {go_results['avg_time_ms']:.2f} ms")
    print(f"  Min time: {go_results['min_time_ms']:.2f} ms") 
    print(f"  Max time: {go_results['max_time_ms']:.2f} ms")
    print(f"  Elements parsed: {go_results['elements_parsed']}")
    print(f"  Relationships: {go_results['relationships_found']}")
    print(f"  Links extracted: {go_results['links_extracted']}")
    print()
    
    print(f"Python XML Parser ({python_results['parser_class']}):")
    print(f"  Average time: {python_results['avg_time_ms']:.2f} ms")
    print(f"  Min time: {python_results['min_time_ms']:.2f} ms")
    print(f"  Max time: {python_results['max_time_ms']:.2f} ms") 
    print(f"  Elements parsed: {python_results['elements_parsed']}")
    print(f"  Relationships: {python_results['relationships_found']}")
    print(f"  Links extracted: {python_results['links_extracted']}")
    print()
    
    # Calculate speedup
    if python_results['avg_time_ms'] > 0:
        speedup = python_results['avg_time_ms'] / go_results['avg_time_ms']
        print(f"Performance Summary:")
        print(f"  Go parser is {speedup:.2f}x faster than Python parser")
        print(f"  Time saved: {python_results['avg_time_ms'] - go_results['avg_time_ms']:.2f} ms")
        
        # Throughput calculations
        go_throughput = content_size_kb / (go_results['avg_time_ms'] / 1000)
        python_throughput = content_size_kb / (python_results['avg_time_ms'] / 1000)
        
        print(f"  Go throughput: {go_throughput:.1f} KB/s")
        print(f"  Python throughput: {python_throughput:.1f} KB/s")


if __name__ == '__main__':
    main()
