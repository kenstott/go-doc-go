#!/usr/bin/env python3
"""
Test sampling endpoints with test-updated1 pipeline.
"""

import requests
import json

# Configuration
SERVER_URL = "http://localhost:5002"
BASE_URL = f"{SERVER_URL}/api/sampling"
PIPELINE_NAME = "test-updated1"


def test_corpus_stats():
    """Test the corpus stats endpoint."""
    print(f"Testing corpus stats for pipeline: {PIPELINE_NAME}")

    try:
        response = requests.post(
            f"{BASE_URL}/corpus-stats",
            json={"pipeline_name": PIPELINE_NAME},
            timeout=10
        )

        print(f"Status: {response.status_code}")

        if response.status_code == 200:
            data = response.json()
            if data.get('success'):
                stats = data.get('statistics', {})
                print("✓ Success! Corpus statistics:")
                print(f"  Total elements: {stats.get('total_elements', 0)}")
                print(f"  Total documents: {stats.get('total_documents', 0)}")
                print(f"  Element types: {stats.get('distinct_element_types', 0)}")

                dist = stats.get('element_type_distribution', {})
                if dist:
                    print("  Distribution:")
                    for etype, count in list(dist.items())[:5]:
                        print(f"    - {etype}: {count}")
            else:
                print(f"✗ Error: {data.get('error')}")
        else:
            print(f"✗ HTTP {response.status_code}: {response.text[:200]}")

    except Exception as e:
        print(f"✗ Request failed: {e}")

    print()


def test_element_sampling():
    """Test the element sampling endpoint."""
    print(f"Testing element sampling for pipeline: {PIPELINE_NAME}")

    try:
        response = requests.post(
            f"{BASE_URL}/elements",
            json={
                "pipeline_name": PIPELINE_NAME,
                "limit": 5
            },
            timeout=10
        )

        print(f"Status: {response.status_code}")

        if response.status_code == 200:
            data = response.json()
            if data.get('success'):
                elements = data.get('elements', [])
                print(f"✓ Success! Sampled {len(elements)} elements")

                for i, elem in enumerate(elements[:3], 1):
                    print(f"  {i}. {elem.get('element_type', 'N/A')} - {elem.get('element_id', 'N/A')[:20]}...")
                    preview = elem.get('content_preview', '')
                    if preview:
                        print(f"     Content: {preview[:50]}...")
            else:
                print(f"✗ Error: {data.get('error')}")
        else:
            print(f"✗ HTTP {response.status_code}: {response.text[:200]}")

    except Exception as e:
        print(f"✗ Request failed: {e}")

    print()


def test_custom_query():
    """Test custom query execution."""
    print(f"Testing custom query for pipeline: {PIPELINE_NAME}")

    try:
        query = """
        SELECT element_type, COUNT(*) as count
        FROM elements
        GROUP BY element_type
        ORDER BY count DESC
        LIMIT 5
        """

        response = requests.post(
            f"{BASE_URL}/custom-query",
            json={
                "pipeline_name": PIPELINE_NAME,
                "query": query
            },
            timeout=10
        )

        print(f"Status: {response.status_code}")

        if response.status_code == 200:
            data = response.json()
            if data.get('success'):
                results = data.get('results', [])
                print(f"✓ Success! Query returned {len(results)} rows")

                for row in results:
                    print(f"  {row.get('element_type', 'N/A')}: {row.get('count', 0)}")
            else:
                print(f"✗ Error: {data.get('error')}")
        else:
            print(f"✗ HTTP {response.status_code}: {response.text[:200]}")

    except Exception as e:
        print(f"✗ Request failed: {e}")

    print()


def test_schema_info():
    """Test schema information retrieval."""
    print(f"Testing schema info for pipeline: {PIPELINE_NAME}")

    try:
        response = requests.get(
            f"{BASE_URL}/schema",
            params={"pipeline_name": PIPELINE_NAME},
            timeout=10
        )

        print(f"Status: {response.status_code}")

        if response.status_code == 200:
            data = response.json()
            if data.get('success'):
                columns = data.get('columns', [])
                print(f"✓ Success! Schema has {len(columns)} columns")

                print("  Sample columns:")
                for col in columns[:5]:
                    print(f"    - {col.get('name')}: {col.get('type')}")

                examples = data.get('examples', {})
                if examples:
                    print("  Example values:")
                    for col_name, values in examples.items():
                        if values:
                            print(f"    {col_name}:")
                            for val in values[:3]:
                                print(f"      - {val.get('value')}: {val.get('count')}")
            else:
                print(f"✗ Error: {data.get('error')}")
        else:
            print(f"✗ HTTP {response.status_code}: {response.text[:200]}")

    except Exception as e:
        print(f"✗ Request failed: {e}")

    print()


def main():
    """Run all tests."""
    print("=" * 60)
    print(f"TESTING SAMPLING ENDPOINTS WITH PIPELINE: {PIPELINE_NAME}")
    print("=" * 60)
    print()

    # Check server health first
    try:
        response = requests.get(f"{SERVER_URL}/health", timeout=5)
        if response.status_code != 200:
            print("⚠️  Warning: Server health check failed")
    except:
        print("❌ Error: Cannot connect to server at", SERVER_URL)
        print("Make sure the server is running")
        return

    # Run tests
    test_corpus_stats()
    test_element_sampling()
    test_custom_query()
    test_schema_info()

    print("=" * 60)
    print("TESTING COMPLETE")
    print("=" * 60)


if __name__ == "__main__":
    main()