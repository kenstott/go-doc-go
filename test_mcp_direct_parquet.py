#!/usr/bin/env python3
"""
Test MCP functionality directly against the parquet data lake.
This bypasses the pipeline configuration and tests the core functionality.
"""

import sys
import os
sys.path.insert(0, os.path.join(os.path.dirname(__file__), 'src'))

from go_doc_go.storage_adapters.analytics_parquet import ParquetAnalyticsStorage

def test_mcp_operations():
    """Test MCP-like operations directly on parquet data."""

    print("=" * 60)
    print("TESTING MCP OPERATIONS ON PARQUET DATA LAKE")
    print("=" * 60)
    print()

    # Create analytics backend for main data-lake
    config = {
        'type': 'parquet',
        'path': './data-lake'  # Main data lake with actual data
    }

    print(f"Using data lake at: {config['path']}")
    backend = ParquetAnalyticsStorage(config)
    print()

    # Test 1: Sample elements (MCP operation)
    print("1. MCP Operation: sample_elements")
    print("-" * 40)
    try:
        elements = backend.sample_elements(limit=5, random_seed=42)
        print(f"✓ Success! Sampled {len(elements)} elements")

        for i, elem in enumerate(elements[:3], 1):
            print(f"\n  Element {i}:")
            print(f"    Type: {elem.get('element_type', 'N/A')}")
            print(f"    ID: {elem.get('element_id', 'N/A')[:40]}...")
            content = elem.get('content_preview', '')
            if content:
                print(f"    Content: {content[:60]}...")
    except Exception as e:
        print(f"✗ Failed: {e}")
    print()

    # Test 2: Get corpus statistics (MCP operation)
    print("2. MCP Operation: get_corpus_stats")
    print("-" * 40)
    try:
        stats = backend.get_corpus_stats()
        print("✓ Success! Retrieved corpus statistics:")
        print(f"  - Total elements: {stats.get('total_elements', 0):,}")
        print(f"  - Total documents: {stats.get('total_documents', 0):,}")
        print(f"  - Element types: {stats.get('distinct_element_types', 0)}")
        print(f"  - Total relationships: {stats.get('total_relationships', 0):,}")

        dist = stats.get('element_type_distribution', {})
        if dist:
            print("\n  Element type distribution (top 5):")
            for etype, count in list(dist.items())[:5]:
                pct = (count / stats.get('total_elements', 1) * 100)
                print(f"    - {etype}: {count:,} ({pct:.1f}%)")
    except Exception as e:
        print(f"✗ Failed: {e}")
    print()

    # Test 3: Filtered sampling (MCP operation)
    print("3. MCP Operation: sample_elements with filters")
    print("-" * 40)
    try:
        xml_elements = backend.sample_elements(
            filters={'element_type': 'xml_element'},
            limit=3,
            random_seed=42
        )
        print(f"✓ Success! Sampled {len(xml_elements)} XML elements")

        for elem in xml_elements:
            name = elem.get('structural_name', 'N/A')
            path = elem.get('structural_path', 'N/A')
            print(f"  - {name} at {path}")
    except Exception as e:
        print(f"✗ Failed: {e}")
    print()

    # Test 4: Sample documents (MCP operation)
    print("4. MCP Operation: sample_documents")
    print("-" * 40)
    try:
        documents = backend.sample_documents(limit=3, random_seed=42)
        print(f"✓ Success! Sampled {len(documents)} documents")

        for doc in documents:
            doc_type = doc.get('doc_type', 'N/A')
            source = doc.get('source', 'N/A')
            print(f"  - Type: {doc_type}")
            print(f"    Source: {source[:80]}...")
    except Exception as e:
        print(f"✗ Failed: {e}")
    print()

    # Test 5: Custom query (MCP operation)
    print("5. MCP Operation: execute_custom_query")
    print("-" * 40)
    try:
        query = """
        SELECT element_type, COUNT(*) as count
        FROM elements
        GROUP BY element_type
        ORDER BY count DESC
        LIMIT 5
        """

        results = backend.execute_custom_query(query)
        print(f"✓ Success! Query returned {len(results)} rows")

        for row in results:
            etype = row.get('element_type', 'N/A')
            count = row.get('count', 0)
            print(f"  - {etype}: {count:,}")
    except Exception as e:
        print(f"✗ Failed: {e}")
    print()

    # Test 6: Stratified sampling (MCP operation)
    print("6. MCP Operation: stratified sampling")
    print("-" * 40)
    try:
        stratified = backend.sample_elements(
            limit=10,
            stratify_by='element_type',
            random_seed=42
        )
        print(f"✓ Success! Stratified sample of {len(stratified)} elements")

        # Count distribution
        type_counts = {}
        for elem in stratified:
            etype = elem.get('element_type', 'unknown')
            type_counts[etype] = type_counts.get(etype, 0) + 1

        print("  Distribution in stratified sample:")
        for etype, count in sorted(type_counts.items()):
            print(f"    - {etype}: {count}")
    except Exception as e:
        print(f"✗ Failed: {e}")
    print()

    print("=" * 60)
    print("MCP OPERATIONS TEST COMPLETE")
    print("=" * 60)
    print()
    print("These operations demonstrate the MCP server functionality:")
    print("- sample_elements: Random or filtered element sampling")
    print("- get_corpus_stats: Comprehensive corpus statistics")
    print("- sample_documents: Document sampling")
    print("- execute_custom_query: Custom DuckDB queries")
    print("- stratified sampling: Balanced sampling across categories")
    print()
    print("The MCP server wraps these operations and exposes them")
    print("via a standard protocol for AI assistants to use.")


if __name__ == "__main__":
    test_mcp_operations()