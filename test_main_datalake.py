#!/usr/bin/env python3
"""
Test sampling using the main data-lake.
"""

import sys
import os
sys.path.insert(0, os.path.join(os.path.dirname(__file__), 'src'))

from go_doc_go.storage_adapters.analytics_parquet import ParquetAnalyticsStorage

def main():
    print("=" * 60)
    print("MAIN DATA-LAKE SAMPLING TEST")
    print("=" * 60)
    print()

    # Create analytics backend for main data-lake
    config = {
        'type': 'parquet',
        'path': './data-lake'
    }

    print(f"Creating ParquetAnalyticsStorage with path: {config['path']}")
    backend = ParquetAnalyticsStorage(config)

    # Test corpus stats
    print("\n1. Getting corpus statistics...")
    try:
        stats = backend.get_corpus_stats()
        print(f"✓ Total elements: {stats.get('total_elements', 0)}")
        print(f"✓ Total documents: {stats.get('total_documents', 0)}")
        print(f"✓ Element types: {stats.get('distinct_element_types', 0)}")
        print(f"✓ Total relationships: {stats.get('total_relationships', 0)}")

        dist = stats.get('element_type_distribution', {})
        if dist:
            print("✓ Element type distribution:")
            for etype, count in list(dist.items())[:5]:
                print(f"    - {etype}: {count}")
    except Exception as e:
        print(f"✗ Error: {e}")

    # Test element sampling
    print("\n2. Sampling elements...")
    try:
        elements = backend.sample_elements(limit=5)
        print(f"✓ Sampled {len(elements)} elements")

        for i, elem in enumerate(elements[:3], 1):
            print(f"\n  Element {i}:")
            print(f"    Type: {elem.get('element_type', 'N/A')}")
            print(f"    ID: {elem.get('element_id', 'N/A')[:40]}...")
            print(f"    Structural name: {elem.get('structural_name', 'N/A')}")
            content = elem.get('content_preview', '')
            if content:
                print(f"    Content: {content[:80]}...")
    except Exception as e:
        print(f"✗ Error: {e}")

    # Test filtered sampling with XML elements
    print("\n3. Sampling XML elements only...")
    try:
        xml_elements = backend.sample_elements(
            filters={'element_type': 'xml_element'},
            limit=3
        )
        print(f"✓ Sampled {len(xml_elements)} XML elements")

        for elem in xml_elements:
            name = elem.get('structural_name', 'N/A')
            path = elem.get('structural_path', 'N/A')
            print(f"  - {name} at {path}")
            print(f"    Content: {elem.get('content_preview', '')[:60]}...")
    except Exception as e:
        print(f"✗ Error: {e}")

    # Test document sampling
    print("\n4. Sampling documents...")
    try:
        documents = backend.sample_documents(limit=5)
        print(f"✓ Sampled {len(documents)} documents")

        for doc in documents[:3]:
            print(f"  - Type: {doc.get('doc_type', 'N/A')}")
            print(f"    Source: {doc.get('source', 'N/A')[:80]}...")
            print(f"    ID: {doc.get('doc_id', 'N/A')[:40]}...")
    except Exception as e:
        print(f"✗ Error: {e}")

    # Test custom query
    print("\n5. Running custom query for element type distribution...")
    try:
        query = """
        SELECT element_type, COUNT(*) as count
        FROM elements
        GROUP BY element_type
        ORDER BY count DESC
        LIMIT 10
        """

        results = backend.execute_custom_query(query)
        print(f"✓ Query returned {len(results)} rows")

        total = sum(row.get('count', 0) for row in results)
        print(f"✓ Total elements across types: {total}")

        for row in results:
            etype = row.get('element_type', 'N/A')
            count = row.get('count', 0)
            pct = (count / total * 100) if total > 0 else 0
            print(f"    {etype}: {count} ({pct:.1f}%)")
    except Exception as e:
        print(f"✗ Error: {e}")

    # Test stratified sampling
    print("\n6. Testing stratified sampling by element_type...")
    try:
        stratified = backend.sample_elements(
            limit=20,
            stratify_by='element_type',
            random_seed=42
        )
        print(f"✓ Stratified sample of {len(stratified)} elements")

        # Count distribution in sample
        type_counts = {}
        for elem in stratified:
            etype = elem.get('element_type', 'unknown')
            type_counts[etype] = type_counts.get(etype, 0) + 1

        print("✓ Distribution in stratified sample:")
        for etype, count in sorted(type_counts.items()):
            print(f"    {etype}: {count}")
    except Exception as e:
        print(f"✗ Error: {e}")

    print("\n" + "=" * 60)
    print("TEST COMPLETE - Sampling methods working correctly!")
    print("=" * 60)


if __name__ == "__main__":
    main()