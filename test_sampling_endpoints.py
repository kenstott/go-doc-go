#!/usr/bin/env python3
"""
Test script for sampling endpoints.
"""

import requests
import json
import sys

# Server URL
SERVER_URL = "http://localhost:5002"
BASE_URL = f"{SERVER_URL}/api/sampling"


def test_corpus_stats():
    """Test the corpus stats endpoint."""
    print("Testing POST /api/sampling/corpus-stats...")

    try:
        response = requests.post(
            f"{BASE_URL}/corpus-stats",
            json={},
            timeout=10
        )

        if response.status_code == 200:
            data = response.json()
            print(f"✓ Success: Got corpus stats")
            print(f"  Total elements: {data.get('statistics', {}).get('total_elements', 'N/A')}")
            print(f"  Total documents: {data.get('statistics', {}).get('total_documents', 'N/A')}")
            return True
        else:
            print(f"✗ Failed with status {response.status_code}: {response.text}")
            return False

    except Exception as e:
        print(f"✗ Error: {e}")
        return False


def test_element_sampling():
    """Test the element sampling endpoint."""
    print("\nTesting POST /api/sampling/elements...")

    try:
        # Test with basic sampling
        payload = {
            "limit": 10,
            "include_document_attrs": True
        }

        response = requests.post(
            f"{BASE_URL}/elements",
            json=payload,
            timeout=10
        )

        if response.status_code == 200:
            data = response.json()
            print(f"✓ Success: Sampled {data.get('count', 0)} elements")

            # Test with filters
            print("\n  Testing with filters...")
            payload = {
                "filters": {"element_type": "xml_element"},
                "limit": 5
            }

            response = requests.post(
                f"{BASE_URL}/elements",
                json=payload,
                timeout=10
            )

            if response.status_code == 200:
                data = response.json()
                print(f"  ✓ Filtered sampling: Got {data.get('count', 0)} XML elements")
                return True
            else:
                print(f"  ✗ Filtered sampling failed: {response.status_code}")
                return False
        else:
            print(f"✗ Failed with status {response.status_code}: {response.text}")
            return False

    except Exception as e:
        print(f"✗ Error: {e}")
        return False


def test_document_sampling():
    """Test the document sampling endpoint."""
    print("\nTesting POST /api/sampling/documents...")

    try:
        payload = {
            "limit": 5
        }

        response = requests.post(
            f"{BASE_URL}/documents",
            json=payload,
            timeout=10
        )

        if response.status_code == 200:
            data = response.json()
            print(f"✓ Success: Sampled {data.get('count', 0)} documents")
            return True
        else:
            print(f"✗ Failed with status {response.status_code}: {response.text}")
            return False

    except Exception as e:
        print(f"✗ Error: {e}")
        return False


def test_schema_info():
    """Test the schema info endpoint."""
    print("\nTesting GET /api/sampling/schema...")

    try:
        response = requests.get(
            f"{BASE_URL}/schema",
            timeout=10
        )

        if response.status_code == 200:
            data = response.json()
            columns = data.get('columns', [])
            print(f"✓ Success: Got schema with {len(columns)} columns")

            # Show some example columns
            if columns:
                print("  Sample columns:")
                for col in columns[:5]:
                    print(f"    - {col.get('name')} ({col.get('type')})")

            return True
        else:
            print(f"✗ Failed with status {response.status_code}: {response.text}")
            return False

    except Exception as e:
        print(f"✗ Error: {e}")
        return False


def test_custom_query():
    """Test the custom query endpoint."""
    print("\nTesting POST /api/sampling/custom-query...")

    try:
        # Simple SELECT query
        payload = {
            "query": "SELECT COUNT(*) as total FROM element_document_enriched",
            "params": []
        }

        response = requests.post(
            f"{BASE_URL}/custom-query",
            json=payload,
            timeout=10
        )

        if response.status_code == 200:
            data = response.json()
            results = data.get('results', [])
            print(f"✓ Success: Query returned {len(results)} rows")
            if results:
                print(f"  Total count: {results[0].get('total', 'N/A')}")

            # Test query rejection for non-SELECT
            print("\n  Testing query safety...")
            payload = {
                "query": "DROP TABLE test",
                "params": []
            }

            response = requests.post(
                f"{BASE_URL}/custom-query",
                json=payload,
                timeout=10
            )

            if response.status_code == 403:
                print("  ✓ Correctly rejected non-SELECT query")
                return True
            else:
                print(f"  ✗ Should have rejected DROP query but got: {response.status_code}")
                return False
        else:
            print(f"✗ Failed with status {response.status_code}: {response.text}")
            return False

    except Exception as e:
        print(f"✗ Error: {e}")
        return False


def test_ontology_sampling():
    """Test the ontology sampling endpoint."""
    print("\nTesting POST /api/sampling/ontology-sample...")

    try:
        payload = {
            "domain_keywords": ["owner", "transaction", "date"],
            "max_elements": 50,
            "include_stats": True
        }

        response = requests.post(
            f"{BASE_URL}/ontology-sample",
            json=payload,
            timeout=30
        )

        if response.status_code == 200:
            data = response.json()
            print(f"✓ Success: Ontology sampling completed")
            print(f"  Sample count: {data.get('sample_count', 0)}")

            patterns = data.get('patterns', {})
            if patterns:
                structural_names = patterns.get('structural_names', {})
                print(f"  Found {len(structural_names)} unique structural names")

                # Show top patterns
                if structural_names:
                    print("  Top structural names:")
                    for name, count in list(structural_names.items())[:5]:
                        print(f"    - {name}: {count}")

            return True
        else:
            print(f"✗ Failed with status {response.status_code}: {response.text}")
            return False

    except Exception as e:
        print(f"✗ Error: {e}")
        return False


def main():
    """Run all tests."""
    print("=" * 60)
    print("SAMPLING ENDPOINTS TEST SUITE")
    print("=" * 60)
    print(f"Testing server at: {SERVER_URL}")
    print()

    # Check if server is running
    try:
        response = requests.get(f"{SERVER_URL}/health", timeout=5)
        if response.status_code != 200:
            print("⚠️  Warning: Server health check failed")
    except:
        print("❌ Error: Cannot connect to server at", SERVER_URL)
        print("Make sure the server is running: python src/go_doc_go/server.py")
        return 1

    tests = [
        test_corpus_stats,
        test_element_sampling,
        test_document_sampling,
        test_schema_info,
        test_custom_query,
        test_ontology_sampling
    ]

    passed = 0
    failed = 0

    for test in tests:
        if test():
            passed += 1
        else:
            failed += 1

    print()
    print("=" * 60)
    print(f"RESULTS: {passed} passed, {failed} failed")
    print("=" * 60)

    return 0 if failed == 0 else 1


if __name__ == "__main__":
    sys.exit(main())