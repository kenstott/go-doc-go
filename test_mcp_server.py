#!/usr/bin/env python3
"""
Test script for MCP passthrough server.
"""

import sys
import os
sys.path.insert(0, os.path.join(os.path.dirname(__file__), 'src'))

from go_doc_go.mcp.mcp_passthrough_server import MCPPassthroughServer

def main():
    # Create MCP server instance
    mcp = MCPPassthroughServer(primary_server_url="http://localhost:5002")

    print("=" * 60)
    print("TESTING MCP PASSTHROUGH SERVER")
    print("=" * 60)
    print()

    # Test 1: Sample elements
    print("1. Testing sample_elements...")
    try:
        result = mcp.sample_elements(limit=5)
        if result.get('success'):
            print(f"✓ Success! Sampled {result.get('count', 0)} elements")
            elements = result.get('elements', [])
            for i, elem in enumerate(elements[:3], 1):
                print(f"  {i}. {elem.get('element_type', 'N/A')}")
        else:
            print(f"✗ Error: {result.get('error')}")
    except Exception as e:
        print(f"✗ Failed: {e}")
    print()

    # Test 2: Get corpus stats
    print("2. Testing get_corpus_stats...")
    try:
        result = mcp.get_corpus_stats()
        if result.get('success'):
            stats = result.get('statistics', {})
            print(f"✓ Success! Corpus statistics:")
            print(f"  - Total elements: {stats.get('total_elements', 0)}")
            print(f"  - Total documents: {stats.get('total_documents', 0)}")
            print(f"  - Element types: {stats.get('distinct_element_types', 0)}")
        else:
            print(f"✗ Error: {result.get('error')}")
    except Exception as e:
        print(f"✗ Failed: {e}")
    print()

    # Test 3: Sample with filters
    print("3. Testing filtered sampling (XML elements only)...")
    try:
        result = mcp.sample_elements(
            filters={'element_type': 'xml_element'},
            limit=3
        )
        if result.get('success'):
            print(f"✓ Success! Sampled {result.get('count', 0)} XML elements")
            elements = result.get('elements', [])
            for elem in elements:
                name = elem.get('structural_name', 'N/A')
                print(f"  - {name}")
        else:
            print(f"✗ Error: {result.get('error')}")
    except Exception as e:
        print(f"✗ Failed: {e}")
    print()

    # Test 4: Sample documents
    print("4. Testing sample_documents...")
    try:
        result = mcp.sample_documents(limit=3)
        if result.get('success'):
            print(f"✓ Success! Sampled {result.get('count', 0)} documents")
            documents = result.get('documents', [])
            for doc in documents:
                doc_type = doc.get('doc_type', 'N/A')
                print(f"  - {doc_type}")
        else:
            print(f"✗ Error: {result.get('error')}")
    except Exception as e:
        print(f"✗ Failed: {e}")
    print()

    # Test 5: Execute custom query
    print("5. Testing custom query...")
    try:
        query = """
        SELECT element_type, COUNT(*) as count
        FROM elements
        GROUP BY element_type
        ORDER BY count DESC
        LIMIT 5
        """
        result = mcp.execute_custom_query(query)
        if result.get('success'):
            print(f"✓ Success! Query returned {result.get('count', 0)} rows")
            for row in result.get('results', [])[:3]:
                print(f"  - {row.get('element_type', 'N/A')}: {row.get('count', 0)}")
        else:
            print(f"✗ Error: {result.get('error')}")
    except Exception as e:
        print(f"✗ Failed: {e}")
    print()

    print("=" * 60)
    print("MCP SERVER TESTING COMPLETE")
    print("=" * 60)


if __name__ == "__main__":
    # Make sure the primary server is running
    import requests
    try:
        response = requests.get("http://localhost:5002/health", timeout=2)
        if response.status_code != 200:
            print("⚠️  Warning: Primary server health check failed")
            print("Make sure the server is running on port 5002")
    except:
        print("❌ Error: Cannot connect to primary server at http://localhost:5002")
        print("Please start the server first with: cd src && python -m go_doc_go.server")
        sys.exit(1)

    main()