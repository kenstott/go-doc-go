#!/usr/bin/env python3
"""
Test sampling directly using the parquet analytics backend.
"""

import sys
import os
sys.path.insert(0, os.path.join(os.path.dirname(__file__), 'src'))

from go_doc_go.storage_adapters.analytics_parquet import ParquetAnalyticsStorage

def main():
    print("=" * 60)
    print("DIRECT PARQUET SAMPLING TEST")
    print("=" * 60)
    print()

    # Create analytics backend directly
    config = {
        'type': 'parquet',
        'path': './data-lake/test-updated1'
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

        dist = stats.get('element_type_distribution', {})
        if dist:
            print("✓ Distribution:")
            for etype, count in list(dist.items())[:3]:
                print(f"    - {etype}: {count}")
    except Exception as e:
        print(f"✗ Error: {e}")

    # Test element sampling
    print("\n2. Sampling elements...")
    try:
        elements = backend.sample_elements(limit=5)
        print(f"✓ Sampled {len(elements)} elements")

        for i, elem in enumerate(elements[:3], 1):
            print(f"  {i}. {elem.get('element_type', 'N/A')} - ID: {elem.get('element_id', 'N/A')[:30]}...")
            content = elem.get('content_preview', '')
            if content:
                print(f"     Content: {content[:60]}...")
    except Exception as e:
        print(f"✗ Error: {e}")

    # Test filtered sampling
    print("\n3. Sampling XML elements only...")
    try:
        xml_elements = backend.sample_elements(
            filters={'element_type': 'xml_element'},
            limit=3
        )
        print(f"✓ Sampled {len(xml_elements)} XML elements")

        for elem in xml_elements:
            name = elem.get('structural_name', 'N/A')
            print(f"  - {name}: {elem.get('content_preview', '')[:50]}...")
    except Exception as e:
        print(f"✗ Error: {e}")

    # Test document sampling
    print("\n4. Sampling documents...")
    try:
        documents = backend.sample_documents(limit=3)
        print(f"✓ Sampled {len(documents)} documents")

        for doc in documents:
            print(f"  - {doc.get('doc_type', 'N/A')}: {doc.get('source', 'N/A')[:60]}...")
    except Exception as e:
        print(f"✗ Error: {e}")

    # Test custom query
    print("\n5. Running custom query...")
    try:
        query = """
        SELECT element_type, COUNT(*) as count
        FROM elements
        GROUP BY element_type
        ORDER BY count DESC
        LIMIT 5
        """

        results = backend.execute_custom_query(query)
        print(f"✓ Query returned {len(results)} rows")

        for row in results:
            print(f"  - {row.get('element_type', 'N/A')}: {row.get('count', 0)}")
    except Exception as e:
        print(f"✗ Error: {e}")

    print("\n" + "=" * 60)
    print("TEST COMPLETE")
    print("=" * 60)


if __name__ == "__main__":
    main()