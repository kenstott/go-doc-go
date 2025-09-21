"""
Test to diagnose why embedding-element join returns no results.
This directly tests the SQL query used in ParquetAnalyticsStorage.search_semantic.
"""

import duckdb
import os
import numpy as np

def test_embedding_element_join():
    """Test the exact join used in search_semantic method."""
    
    print("\n" + "="*80)
    print("EMBEDDING-ELEMENT JOIN DIAGNOSTIC")
    print("="*80 + "\n")
    
    conn = duckdb.connect(':memory:')
    
    # Register Parquet files
    elements_path = 'data-lake/elements/**/*.parquet'
    embeddings_path = 'data-lake/embeddings/**/*.parquet'
    
    print("1. REGISTERING PARQUET FILES:")
    print(f"   Elements: {elements_path}")
    print(f"   Embeddings: {embeddings_path}")
    
    conn.execute(f"CREATE VIEW elements AS SELECT * FROM read_parquet('{elements_path}')")
    conn.execute(f"CREATE VIEW embeddings AS SELECT * FROM read_parquet('{embeddings_path}')")
    
    # Check schemas
    print("\n2. CHECKING SCHEMAS:")
    
    print("\n   Elements schema:")
    elements_schema = conn.execute("DESCRIBE elements").fetchall()
    for col in elements_schema[:10]:  # Show first 10 columns
        print(f"      {col[0]}: {col[1]}")
    
    print("\n   Embeddings schema:")
    embeddings_schema = conn.execute("DESCRIBE embeddings").fetchall()
    for col in embeddings_schema:
        print(f"      {col[0]}: {col[1]}")
    
    # Count records
    print("\n3. RECORD COUNTS:")
    
    elements_count = conn.execute("SELECT COUNT(*) FROM elements").fetchone()[0]
    print(f"   Total elements: {elements_count:,}")
    
    embeddings_count = conn.execute("SELECT COUNT(*) FROM embeddings").fetchone()[0]
    print(f"   Total embeddings: {embeddings_count:,}")
    
    # Sample some element IDs
    print("\n4. SAMPLE ELEMENT IDs:")
    sample_elements = conn.execute("""
        SELECT element_id, element_type, content_preview
        FROM elements 
        WHERE content_preview LIKE '%sales%' 
        LIMIT 5
    """).fetchall()
    
    if sample_elements:
        print("   Elements containing 'sales':")
        for elem_id, elem_type, preview in sample_elements:
            print(f"      {elem_id[:20]}... ({elem_type}): {preview[:50]}...")
    else:
        print("   No elements found containing 'sales'")
    
    # Sample some embeddings
    print("\n5. SAMPLE EMBEDDING IDs:")
    sample_embeddings = conn.execute("""
        SELECT element_id, model 
        FROM embeddings 
        LIMIT 5
    """).fetchall()
    
    for elem_id, model in sample_embeddings:
        print(f"      {elem_id[:20]}... (model: {model})")
    
    # Test the JOIN
    print("\n6. TESTING THE JOIN:")
    
    # Simple join test
    join_result = conn.execute("""
        SELECT COUNT(*) as joined_count
        FROM elements e
        JOIN embeddings emb ON e.element_id = emb.element_id
    """).fetchone()[0]
    
    print(f"   Joined records: {join_result:,}")
    
    if join_result == 0:
        print("\n   ❌ JOIN RETURNS NO RESULTS!")
        
        # Investigate why
        print("\n   Investigating mismatch...")
        
        # Check if any element_ids match
        matching_ids = conn.execute("""
            SELECT COUNT(*) 
            FROM (
                SELECT DISTINCT element_id FROM elements
                INTERSECT
                SELECT DISTINCT element_id FROM embeddings
            ) t
        """).fetchone()[0]
        
        print(f"   Matching element_ids: {matching_ids}")
        
        # Compare ID formats
        print("\n   Sample ID formats:")
        
        elem_ids = conn.execute("""
            SELECT element_id 
            FROM elements 
            LIMIT 3
        """).fetchall()
        
        print("   From elements table:")
        for (elem_id,) in elem_ids:
            print(f"      '{elem_id}' (length: {len(elem_id)})")
        
        emb_ids = conn.execute("""
            SELECT element_id 
            FROM embeddings 
            LIMIT 3
        """).fetchall()
        
        print("   From embeddings table:")
        for (elem_id,) in emb_ids:
            print(f"      '{elem_id}' (length: {len(elem_id)})")
            
        # Check for common prefixes
        print("\n   Checking ID patterns:")
        
        elem_prefix = conn.execute("""
            SELECT SUBSTRING(element_id, 1, 10) as prefix, COUNT(*) as count
            FROM elements
            GROUP BY prefix
            ORDER BY count DESC
            LIMIT 3
        """).fetchall()
        
        print("   Element ID prefixes:")
        for prefix, count in elem_prefix:
            print(f"      '{prefix}': {count} elements")
        
        emb_prefix = conn.execute("""
            SELECT SUBSTRING(element_id, 1, 10) as prefix, COUNT(*) as count
            FROM embeddings
            GROUP BY prefix
            ORDER BY count DESC
            LIMIT 3
        """).fetchall()
        
        print("   Embedding ID prefixes:")
        for prefix, count in emb_prefix:
            print(f"      '{prefix}': {count} embeddings")
    
    else:
        print(f"   ✅ Join successful with {join_result:,} results")
        
        # Test with actual search
        print("\n7. TESTING SEMANTIC SEARCH:")
        
        # Create a dummy query embedding
        query_embedding = [0.1] * 384  # Assuming 384 dimensions
        query_vec_str = '[' + ','.join(map(str, query_embedding)) + ']'
        
        # Test the full search query
        search_query = f"""
        WITH valid_embeddings AS (
            SELECT 
                e.*,
                emb.embedding,
                sqrt(list_dot_product(emb.embedding::DOUBLE[], emb.embedding::DOUBLE[])) as emb_magnitude
            FROM elements e
            JOIN embeddings emb ON e.element_id = emb.element_id
            WHERE sqrt(list_dot_product(emb.embedding::DOUBLE[], emb.embedding::DOUBLE[])) > 0.0
        ),
        search_results AS (
            SELECT 
                element_id,
                element_type,
                content_preview,
                (
                    list_dot_product(embedding::DOUBLE[], {query_vec_str}::DOUBLE[]) / 
                    (emb_magnitude * sqrt(list_dot_product({query_vec_str}::DOUBLE[], {query_vec_str}::DOUBLE[])))
                ) as similarity
            FROM valid_embeddings
        )
        SELECT * FROM search_results
        WHERE similarity >= 0.0
        AND similarity IS NOT NULL
        AND NOT isnan(similarity)
        ORDER BY similarity DESC
        LIMIT 5
        """
        
        try:
            results = conn.execute(search_query).fetchall()
            print(f"   Search returned {len(results)} results")
            
            if results:
                for elem_id, elem_type, preview, similarity in results[:3]:
                    print(f"      {elem_id[:20]}... ({elem_type}): similarity={similarity:.4f}")
                    print(f"         {preview[:60]}...")
        except Exception as e:
            print(f"   Search query failed: {e}")
    
    conn.close()
    
    print("\n" + "="*80)
    print("DIAGNOSTIC COMPLETE")
    print("="*80)


if __name__ == "__main__":
    test_embedding_element_join()