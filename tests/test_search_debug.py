"""
Debug why semantic search returns no results despite successful join.
"""

import duckdb
import numpy as np

def debug_search():
    """Debug the search query step by step."""
    
    print("\n" + "="*80)
    print("SEMANTIC SEARCH DEBUGGING")
    print("="*80 + "\n")
    
    conn = duckdb.connect(':memory:')
    
    # Register Parquet files
    elements_path = 'data-lake/elements/**/*.parquet'
    embeddings_path = 'data-lake/embeddings/**/*.parquet'
    
    conn.execute(f"CREATE VIEW elements AS SELECT * FROM read_parquet('{elements_path}')")
    conn.execute(f"CREATE VIEW embeddings AS SELECT * FROM read_parquet('{embeddings_path}')")
    
    # Step 1: Check embedding dimensions
    print("1. CHECKING EMBEDDING DIMENSIONS:")
    
    dim_check = conn.execute("""
        SELECT 
            LENGTH(embedding) as dim,
            COUNT(*) as count
        FROM embeddings
        GROUP BY dim
    """).fetchall()
    
    for dim, count in dim_check:
        print(f"   Dimension {dim}: {count:,} embeddings")
    
    # Step 2: Check embedding magnitudes
    print("\n2. CHECKING EMBEDDING MAGNITUDES:")
    
    magnitude_check = conn.execute("""
        SELECT 
            MIN(sqrt(list_dot_product(embedding::DOUBLE[], embedding::DOUBLE[]))) as min_mag,
            MAX(sqrt(list_dot_product(embedding::DOUBLE[], embedding::DOUBLE[]))) as max_mag,
            AVG(sqrt(list_dot_product(embedding::DOUBLE[], embedding::DOUBLE[]))) as avg_mag,
            COUNT(CASE WHEN sqrt(list_dot_product(embedding::DOUBLE[], embedding::DOUBLE[])) = 0 THEN 1 END) as zero_count
        FROM embeddings
    """).fetchone()
    
    print(f"   Min magnitude: {magnitude_check[0]}")
    print(f"   Max magnitude: {magnitude_check[1]}")
    print(f"   Avg magnitude: {magnitude_check[2]}")
    print(f"   Zero magnitude count: {magnitude_check[3]}")
    
    # Step 3: Test with actual embedding
    print("\n3. TESTING WITH ACTUAL EMBEDDING:")
    
    # Get a real embedding to use as query
    real_embedding = conn.execute("""
        SELECT embedding 
        FROM embeddings 
        LIMIT 1
    """).fetchone()[0]
    
    print(f"   Using embedding with {len(real_embedding)} dimensions")
    
    # Convert to string for query
    query_vec_str = '[' + ','.join(map(str, real_embedding)) + ']'
    
    # Step 4: Test simplified similarity
    print("\n4. TESTING SIMPLIFIED SIMILARITY:")
    
    simple_test = conn.execute(f"""
        SELECT 
            e.element_id,
            e.element_type,
            e.content_preview,
            list_dot_product(emb.embedding::DOUBLE[], {query_vec_str}::DOUBLE[]) as dot_product,
            sqrt(list_dot_product(emb.embedding::DOUBLE[], emb.embedding::DOUBLE[])) as emb_mag,
            sqrt(list_dot_product({query_vec_str}::DOUBLE[], {query_vec_str}::DOUBLE[])) as query_mag
        FROM elements e
        JOIN embeddings emb ON e.element_id = emb.element_id
        LIMIT 5
    """).fetchall()
    
    print("   Sample calculations:")
    for elem_id, elem_type, preview, dot_prod, emb_mag, query_mag in simple_test:
        similarity = dot_prod / (emb_mag * query_mag) if (emb_mag * query_mag) > 0 else 0
        print(f"      {elem_id[:20]}... ({elem_type}):")
        print(f"         Dot product: {dot_prod}")
        print(f"         Emb magnitude: {emb_mag}")
        print(f"         Query magnitude: {query_mag}")
        print(f"         Similarity: {similarity}")
        print(f"         Preview: {preview[:50]}...")
    
    # Step 5: Test full search with debugging
    print("\n5. TESTING FULL SEARCH WITH DEBUGGING:")
    
    # Use minimum similarity of -1 to see all results
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
            emb_magnitude,
            sqrt(list_dot_product({query_vec_str}::DOUBLE[], {query_vec_str}::DOUBLE[])) as query_magnitude,
            list_dot_product(embedding::DOUBLE[], {query_vec_str}::DOUBLE[]) as dot_product,
            (
                list_dot_product(embedding::DOUBLE[], {query_vec_str}::DOUBLE[]) / 
                (emb_magnitude * sqrt(list_dot_product({query_vec_str}::DOUBLE[], {query_vec_str}::DOUBLE[])))
            ) as similarity
        FROM valid_embeddings
    )
    SELECT * FROM search_results
    WHERE similarity >= -1.0
    ORDER BY similarity DESC
    LIMIT 10
    """
    
    results = conn.execute(search_query).fetchall()
    
    print(f"   Search returned {len(results)} results")
    
    if results:
        print("\n   Top results:")
        for row in results[:5]:
            elem_id, elem_type, preview, emb_mag, query_mag, dot_prod, similarity = row
            print(f"      {elem_id[:20]}... ({elem_type}): similarity={similarity:.6f}")
            print(f"         Magnitudes: emb={emb_mag:.6f}, query={query_mag:.6f}")
            print(f"         Dot product: {dot_prod:.6f}")
            print(f"         {preview[:60]}...")
    else:
        # Check what's happening in the CTE stages
        print("\n   Checking CTE stages...")
        
        # Check valid_embeddings
        valid_count = conn.execute(f"""
            SELECT COUNT(*)
            FROM elements e
            JOIN embeddings emb ON e.element_id = emb.element_id
            WHERE sqrt(list_dot_product(emb.embedding::DOUBLE[], emb.embedding::DOUBLE[])) > 0.0
        """).fetchone()[0]
        
        print(f"   Valid embeddings count: {valid_count}")
        
        # Check if similarity calculation is producing NaN or NULL
        calc_check = conn.execute(f"""
            WITH valid_embeddings AS (
                SELECT 
                    e.element_id,
                    emb.embedding,
                    sqrt(list_dot_product(emb.embedding::DOUBLE[], emb.embedding::DOUBLE[])) as emb_magnitude
                FROM elements e
                JOIN embeddings emb ON e.element_id = emb.element_id
                WHERE sqrt(list_dot_product(emb.embedding::DOUBLE[], emb.embedding::DOUBLE[])) > 0.0
                LIMIT 10
            )
            SELECT 
                element_id,
                (
                    list_dot_product(embedding::DOUBLE[], {query_vec_str}::DOUBLE[]) / 
                    (emb_magnitude * sqrt(list_dot_product({query_vec_str}::DOUBLE[], {query_vec_str}::DOUBLE[])))
                ) as similarity,
                CASE 
                    WHEN isnan(
                        list_dot_product(embedding::DOUBLE[], {query_vec_str}::DOUBLE[]) / 
                        (emb_magnitude * sqrt(list_dot_product({query_vec_str}::DOUBLE[], {query_vec_str}::DOUBLE[])))
                    ) THEN 'NaN'
                    WHEN (
                        list_dot_product(embedding::DOUBLE[], {query_vec_str}::DOUBLE[]) / 
                        (emb_magnitude * sqrt(list_dot_product({query_vec_str}::DOUBLE[], {query_vec_str}::DOUBLE[])))
                    ) IS NULL THEN 'NULL'
                    ELSE 'OK'
                END as status
            FROM valid_embeddings
        """).fetchall()
        
        print("\n   Similarity calculation check:")
        for elem_id, similarity, status in calc_check:
            print(f"      {elem_id[:20]}...: similarity={similarity}, status={status}")
    
    # Step 6: Test text search for comparison
    print("\n6. TESTING TEXT SEARCH FOR COMPARISON:")
    
    text_results = conn.execute("""
        SELECT element_id, element_type, content_preview
        FROM elements
        WHERE content_preview ILIKE '%sales%'
        LIMIT 5
    """).fetchall()
    
    print(f"   Text search for 'sales' returned {len(text_results)} results")
    for elem_id, elem_type, preview in text_results[:3]:
        print(f"      {elem_id[:20]}... ({elem_type}): {preview[:60]}...")
    
    conn.close()
    
    print("\n" + "="*80)
    print("DEBUG COMPLETE")
    print("="*80)


if __name__ == "__main__":
    debug_search()