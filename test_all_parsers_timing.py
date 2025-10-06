#!/usr/bin/env python3
"""
Comprehensive timing test for all document parsers.
Compares Python and Go implementations where available.
"""

import time
import json
import os
from pathlib import Path
from typing import Dict, Any, Tuple
import pandas as pd

# Add src to path
import sys
sys.path.insert(0, str(Path(__file__).parent / 'src'))

from src.go_doc_go.document_parser.factory import create_parser


def time_parser(doc_type: str, content: Dict[str, Any], config: Dict[str, Any] = None) -> Tuple[float, int, str]:
    """Time a single parser execution."""
    try:
        parser = create_parser(doc_type, config)
        parser_name = parser.__class__.__name__

        start = time.time()
        result = parser.parse(content)
        elapsed = time.time() - start

        element_count = len(result.get('elements', []))
        return elapsed, element_count, parser_name
    except Exception as e:
        print(f"  Error: {e}")
        return -1, 0, "Error"


def test_all_parsers():
    """Test all parsers with both Python and Go implementations."""

    # Test configurations with actual available files
    tests = [
        {
            'name': 'Markdown',
            'doc_type': 'markdown',
            'file': 'tests/assets/introduction.md',
            'content_type': 'text',
        },
        {
            'name': 'HTML',
            'doc_type': 'html',
            'file': 'tests/assets/aapl-20201226.htm',
            'content_type': 'text',
        },
        {
            'name': 'JSON',
            'doc_type': 'json',
            'file': 'tests/assets/web/data.json',
            'content_type': 'text',
        },
        {
            'name': 'XML',
            'doc_type': 'xml',
            'file': 'tests/assets/sample-xml-file.xml',
            'content_type': 'text',
        },
        {
            'name': 'CSV',
            'doc_type': 'csv',
            'file': 'tests/assets/Accounts_History.csv',
            'content_type': 'text',
        },
        {
            'name': 'Text',
            'doc_type': 'text',
            'file': 'tests/assets/technical-details.md',  # Use as text
            'content_type': 'text',
        },
        {
            'name': 'PDF',
            'doc_type': 'pdf',
            'file': 'tests/assets/crazyones-pdfa.pdf',
            'content_type': 'binary',
        },
        {
            'name': 'XLSX',
            'doc_type': 'xlsx',
            'file': 'tests/assets/CopilotAnswers-20240924-154049.xlsx',
            'content_type': 'binary',
        },
        {
            'name': 'DOCX',
            'doc_type': 'docx',
            'file': 'tests/assets/test_sample.docx',
            'content_type': 'binary',
        },
        {
            'name': 'PPTX',
            'doc_type': 'pptx',
            'file': 'tests/assets/test_sample.pptx',
            'content_type': 'binary',
        },
        {
            'name': 'Parquet',
            'doc_type': 'parquet',
            'file': 'tests/e2e/test_documents/techcorp_q4_2024_earnings.parquet',
            'content_type': 'binary',
            'config': {
                'text_column': 'paragraph_text',
                'group_by_column': 'section_type',
                'metadata_columns': ['company', 'ticker']
            }
        }
    ]

    results = []
    print("\n" + "="*80)
    print("COMPREHENSIVE PARSER TIMING TEST")
    print("="*80)

    for test in tests:
        doc_type = test['doc_type']
        file_path = test['file']

        # Check if file exists
        if not Path(file_path).exists():
            print(f"\n{test['name']} Parser:")
            print(f"  ⚠️  File not found: {file_path}")
            continue

        print(f"\n{test['name']} Parser ({doc_type}):")
        print(f"  File: {file_path}")

        # Prepare content
        if test['content_type'] == 'text':
            with open(file_path, 'r', encoding='utf-8') as f:
                file_content = f.read()
            content = {
                'id': f'test_{doc_type}',
                'content': file_content,
                'metadata': {'source': file_path}
            }
        else:  # binary
            content = {
                'id': f'test_{doc_type}',
                'binary_path': file_path,
                'metadata': {'source': file_path}
            }

        config = test.get('config', {})

        # Test Python implementation
        os.environ['USE_GO_MODULES'] = 'false'
        print("  Python Implementation:")
        py_time, py_elements, py_parser = time_parser(doc_type, content, config)
        if py_time > 0:
            print(f"    Parser: {py_parser}")
            print(f"    Time: {py_time*1000:.2f}ms")
            print(f"    Elements: {py_elements}")
            results.append({
                'Document': test['name'],
                'Implementation': 'Python',
                'Parser': py_parser,
                'Time (ms)': round(py_time * 1000, 2),
                'Elements': py_elements
            })

        # Test Go implementation
        os.environ['USE_GO_MODULES'] = 'true'
        print("  Go Implementation:")
        go_time, go_elements, go_parser = time_parser(doc_type, content, config)
        if go_time > 0:
            print(f"    Parser: {go_parser}")
            print(f"    Time: {go_time*1000:.2f}ms")
            print(f"    Elements: {go_elements}")
            results.append({
                'Document': test['name'],
                'Implementation': 'Go',
                'Parser': go_parser,
                'Time (ms)': round(go_time * 1000, 2),
                'Elements': go_elements
            })

            # Calculate speedup
            if py_time > 0:
                speedup = py_time / go_time
                print(f"  📊 Speedup: {speedup:.2f}x")
                if speedup > 1:
                    print(f"     Go is {speedup:.2f}x faster")
                else:
                    print(f"     Python is {1/speedup:.2f}x faster")

    # Summary table
    print("\n" + "="*80)
    print("SUMMARY")
    print("="*80)

    if results:
        df = pd.DataFrame(results)

        # Pivot table for comparison
        pivot = df.pivot_table(
            values='Time (ms)',
            index='Document',
            columns='Implementation',
            aggfunc='first'
        )

        # Calculate speedup
        if 'Go' in pivot.columns and 'Python' in pivot.columns:
            pivot['Speedup'] = (pivot['Python'] / pivot['Go']).round(2)
            pivot['Faster'] = pivot.apply(
                lambda row: f"Go {row['Speedup']:.1f}x" if row['Speedup'] > 1 else f"Python {1/row['Speedup']:.1f}x",
                axis=1
            )

        print(pivot.to_string())

        # Overall statistics
        print("\n" + "-"*80)
        print("OVERALL STATISTICS:")

        go_results = df[df['Implementation'] == 'Go']['Time (ms)']
        py_results = df[df['Implementation'] == 'Python']['Time (ms)']

        if len(go_results) > 0:
            print(f"  Go parsers average: {go_results.mean():.2f}ms")
            print(f"  Go parsers total: {go_results.sum():.2f}ms")

        if len(py_results) > 0:
            print(f"  Python parsers average: {py_results.mean():.2f}ms")
            print(f"  Python parsers total: {py_results.sum():.2f}ms")

        if len(go_results) > 0 and len(py_results) > 0:
            overall_speedup = py_results.sum() / go_results.sum()
            print(f"\n  🚀 Overall Go speedup: {overall_speedup:.2f}x")


if __name__ == "__main__":
    test_all_parsers()