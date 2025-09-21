#!/usr/bin/env python3
"""
Direct test of parquet files with DuckDB to debug search issues.
"""

import duckdb
import os
from pathlib import Path

def test_parquet_data():
    """Test if we can query parquet files directly."""
    
    print("🔍 TESTING PARQUET DATA DIRECTLY WITH DUCKDB")
    print("=" * 60)
    
    # Connect to DuckDB
    conn = duckdb.connect(':memory:')
    
    # Check embeddings
    embeddings_pattern = 'data-lake/embeddings/**/*.parquet'
    print(f"\n1. Checking embeddings at: {embeddings_pattern}")
    
    try:
        # Count total embeddings
        result = conn.execute(f"""
            SELECT COUNT(*) as count 
            FROM read_parquet('{embeddings_pattern}')
        """).fetchone()
        print(f"   Total embeddings: {result[0]}")
        
        # Check schema
        schema = conn.execute(f"""
            DESCRIBE SELECT * 
            FROM read_parquet('{embeddings_pattern}')
            LIMIT 1
        """).fetchall()
        print("\n   Schema:")
        for col in schema:
            print(f"     - {col[0]}: {col[1]}")
        
        # Sample a few embeddings
        samples = conn.execute(f"""
            SELECT element_id, model,
                   list_sum(list_transform(embedding, x -> x * x)) as magnitude_sq,
                   array_slice(embedding, 1, 5) as first_5_values
            FROM read_parquet('{embeddings_pattern}')
            LIMIT 5
        """).fetchall()
        
        print("\n   Sample embeddings:")
        for sample in samples:
            print(f"     - Element: {sample[0][:30]}...")
            print(f"       Model: {sample[1]}")
            print(f"       Magnitude²: {sample[2]:.6f}")
            print(f"       First 5 values: {sample[3]}")
        
        # Search for content with "sales" or similar terms
        print("\n2. Searching for elements with 'sales' related content:")
        
        # First check elements
        elements_pattern = 'data-lake/elements/**/*.parquet'
        sales_elements = conn.execute(f"""
            SELECT element_id, element_type, content_preview
            FROM read_parquet('{elements_pattern}')
            WHERE lower(content_preview) LIKE '%sale%'
               OR lower(content_preview) LIKE '%revenue%'
               OR lower(content_preview) LIKE '%market%'
            LIMIT 5
        """).fetchall()
        
        if sales_elements:
            print(f"   Found {len(sales_elements)} elements with sales-related content:")
            for elem in sales_elements:
                print(f"     - {elem[1]}: {elem[2][:60]}...")
                
            # Check if these have embeddings
            elem_ids = [elem[0] for elem in sales_elements]
            elem_ids_str = "', '".join(elem_ids)
            
            embeddings_for_elements = conn.execute(f"""
                SELECT element_id
                FROM read_parquet('{embeddings_pattern}')
                WHERE element_id IN ('{elem_ids_str}')
            """).fetchall()
            
            print(f"\n   Embeddings found for {len(embeddings_for_elements)}/{len(sales_elements)} elements")
        else:
            print("   No elements found with sales-related content")
        
        # Check embedding statistics
        print("\n3. Embedding statistics:")
        stats = conn.execute(f"""
            SELECT 
                COUNT(*) as total,
                COUNT(DISTINCT element_id) as unique_elements,
                COUNT(DISTINCT model) as unique_models,
                AVG(list_sum(list_transform(embedding, x -> x * x))) as avg_magnitude_sq,
                MIN(list_sum(list_transform(embedding, x -> x * x))) as min_magnitude_sq,
                MAX(list_sum(list_transform(embedding, x -> x * x))) as max_magnitude_sq
            FROM read_parquet('{embeddings_pattern}')
        """).fetchone()
        
        print(f"   Total embeddings: {stats[0]}")
        print(f"   Unique elements: {stats[1]}")
        print(f"   Unique models: {stats[2]}")
        print(f"   Avg magnitude²: {stats[3]:.6f}")
        print(f"   Min magnitude²: {stats[4]:.6f}")
        print(f"   Max magnitude²: {stats[5]:.6f}")
        
        # Check for zero or near-zero embeddings
        zero_embeddings = conn.execute(f"""
            SELECT COUNT(*) as count
            FROM read_parquet('{embeddings_pattern}')
            WHERE list_sum(list_transform(embedding, x -> x * x)) < 0.001
        """).fetchone()
        
        if zero_embeddings[0] > 0:
            print(f"\n   ⚠️  WARNING: {zero_embeddings[0]} embeddings have near-zero magnitude!")
        
    except Exception as e:
        print(f"   ❌ Error querying parquet files: {e}")
        import traceback
        traceback.print_exc()
    
    finally:
        conn.close()
    
    print("\n" + "=" * 60)


if __name__ == "__main__":
    test_parquet_data()