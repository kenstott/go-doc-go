#!/usr/bin/env python3
"""
Test similarity search with detailed results display.
"""

import json
from go_doc_go.search_module import SearchEngine, SearchRequest

def format_json(obj):
    """Pretty format JSON for display."""
    return json.dumps(obj, indent=4, default=str)

def test_similarity_search():
    """Test similarity search and display detailed results."""
    
    print("=" * 80)
    print("SIMILARITY SEARCH TEST WITH DETAILED RESULTS")
    print("=" * 80)
    
    # Initialize search engine
    print("\n1. Initializing SearchEngine...")
    engine = SearchEngine()
    print("   ✅ SearchEngine initialized")
    
    # Test queries
    test_queries = [
        "revenue growth financial performance",
        "machine learning artificial intelligence",
        "customer acquisition marketing",
        "data analytics insights",
        "quarterly earnings report"
    ]
    
    for i, query in enumerate(test_queries, 1):
        print("\n" + "-" * 80)
        print(f"QUERY {i}: '{query}'")
        print("-" * 80)
        
        # Create search request
        request = SearchRequest(
            search_service="parquet_duckdb",
            similarity_query=query,
            limit=5,
            similarity_threshold=0.0,  # Show all results for testing
            include_content=True,
            include_metadata=True
        )
        
        print(f"\nSearch parameters:")
        print(f"  - Service: {request.search_service}")
        print(f"  - Query: {request.similarity_query}")
        print(f"  - Limit: {request.limit}")
        print(f"  - Threshold: {request.similarity_threshold}")
        print(f"  - Include content: {request.include_content}")
        print(f"  - Include metadata: {request.include_metadata}")
        
        # Execute search
        print("\nExecuting search...")
        try:
            response = engine.search(request)
            
            print(f"\n📊 RESULTS:")
            print(f"  Total hits: {response.total_hits}")
            print(f"  Query time: {response.took_ms}ms")
            
            if response.hits:
                print(f"\n  Top {len(response.hits)} results:")
                
                for j, hit in enumerate(response.hits, 1):
                    print(f"\n  ──────── Result {j} ────────")
                    print(f"    Score: {hit.score:.4f}")
                    print(f"    Element ID: {hit.element_id}")
                    print(f"    Document ID: {hit.doc_id}")
                    print(f"    Element Type: {hit.element_type}")
                    
                    if hit.content_preview:
                        print(f"    Content Preview: {hit.content_preview[:200]}...")
                    
                    if hit.content:
                        print(f"    Full Content Length: {len(hit.content)} chars")
                        print(f"    Content Sample: {hit.content[:300]}...")
                    
                    if hit.metadata:
                        print(f"    Metadata:")
                        if isinstance(hit.metadata, dict):
                            for key, value in hit.metadata.items():
                                print(f"      - {key}: {value}")
                        else:
                            print(f"      Raw: {hit.metadata}")
            else:
                print("\n  ⚠️  No results found")
                print("  This could mean:")
                print("    - No documents in the data-lake matching the query")
                print("    - Embeddings not yet generated for documents")
                print("    - Similarity threshold too high")
                
        except Exception as e:
            print(f"\n  ❌ Search failed: {e}")
            import traceback
            traceback.print_exc()
    
    print("\n" + "=" * 80)
    print("SEARCH TEST COMPLETED")
    print("=" * 80)

if __name__ == "__main__":
    test_similarity_search()