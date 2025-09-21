"""
Test to verify that search now works with correct embedding model.
"""

import os
import sys
from pathlib import Path

# Add parent directory to path for imports
parent_dir = Path(__file__).parent.parent.absolute()
sys.path.insert(0, str(parent_dir))
sys.path.insert(0, str(parent_dir / 'src'))

from go_doc_go.search_module import SearchEngine, SearchRequest


def test_search_fixed():
    """Test that search returns results with the correct embedding model."""
    
    print("\n" + "="*80)
    print("TESTING FIXED SEARCH ENGINE")
    print("="*80 + "\n")
    
    # Initialize search engine (should now use correct model)
    search_engine = SearchEngine()
    
    # Test with simple queries as requested
    test_queries = [
        "sales",
        "revenue",
        "customer",
        "financial",
        "microsoft"
    ]
    
    for query in test_queries:
        print(f"\nTesting query: '{query}'")
        print("-" * 40)
        
        try:
            request = SearchRequest(
                search_service="parquet_duckdb",
                similarity_query=query,
                limit=5,
                similarity_threshold=0.3,  # Lower threshold to get more results
                include_content=True,
                include_metadata=True
            )
            
            response = search_engine.search(request)
            
            print(f"Results: {response.total_hits} hits in {response.took_ms}ms")
            
            if response.hits:
                print("\nTop 3 results:")
                for i, hit in enumerate(response.hits[:3], 1):
                    print(f"\n  {i}. Score: {hit.score:.4f}")
                    print(f"     Type: {hit.element_type}")
                    print(f"     Preview: {hit.content_preview[:100]}...")
                    if hit.metadata:
                        print(f"     Metadata keys: {list(hit.metadata.keys())}")
            else:
                print("  No results found")
                
        except Exception as e:
            print(f"  Error: {e}")
            import traceback
            traceback.print_exc()
    
    print("\n" + "="*80)
    print("TEST COMPLETE")
    print("="*80)


if __name__ == "__main__":
    test_search_fixed()