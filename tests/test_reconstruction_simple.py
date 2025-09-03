"""
Simple test for document reconstruction that directly uses the database.
This avoids complex import dependencies while testing the core functionality.
"""

import json
import sqlite3
import os
from typing import List, Dict, Any

def visualize_hierarchy(elements: List[Dict[str, Any]], indent: int = 0) -> str:
    """Create visual representation of element hierarchy."""
    output = []
    
    for elem in elements:
        # Create indent
        prefix = "  " * indent
        
        # Determine symbol based on type
        if elem['element_type'] == 'root':
            symbol = "📄"
        elif elem['element_type'] == 'header':
            level = json.loads(elem.get('metadata', '{}')).get('level', 1)
            symbols = {1: "H1", 2: "H2", 3: "H3", 4: "H4", 5: "H5", 6: "H6"}
            symbol = symbols.get(level, "H?")
        elif elem['element_type'] == 'paragraph':
            symbol = "¶"
        else:
            symbol = "•"
            
        # Format line
        preview = elem['content_preview'][:60] + "..." if len(elem['content_preview']) > 60 else elem['content_preview']
        line = f"{prefix}{symbol} [{elem['element_type']}] {preview}"
        
        # Add ordering info
        if elem.get('element_order') is not None:
            line += f" (order:{elem['element_order']}, pos:{elem.get('document_position', '?')})"
            
        output.append(line)
        
        # Process children
        if 'children' in elem:
            child_output = visualize_hierarchy(elem['children'], indent + 1)
            output.append(child_output)
            
    return "\n".join(output)


def test_reconstruction():
    """Test document reconstruction with ordering."""
    
    print("\n" + "="*80)
    print("DOCUMENT RECONSTRUCTION TEST - SIMPLE VERSION")
    print("="*80 + "\n")
    
    # Connect to SQLite database
    db_path = "tests/data/document_db.sqlite"
    
    if not os.path.exists(db_path):
        print(f"❌ Database not found at {db_path}")
        print("Please run document ingestion first.")
        return
        
    conn = sqlite3.connect(db_path)
    conn.row_factory = sqlite3.Row
    cursor = conn.cursor()
    
    try:
        # 1. Check if we have the ordering columns
        print("1. Checking database schema...")
        cursor.execute("PRAGMA table_info(elements)")
        columns = {row['name'] for row in cursor.fetchall()}
        
        has_ordering = 'element_order' in columns and 'document_position' in columns
        print(f"   Ordering columns present: {'✅ Yes' if has_ordering else '❌ No'}")
        
        if not has_ordering:
            print("\n⚠️  Warning: Ordering columns not found. Results may not be properly ordered.")
            print("   Please re-run ingestion with the updated schema.")
        
        # 2. Get a sample document
        print("\n2. Finding documents...")
        cursor.execute("SELECT doc_id, doc_type, source FROM documents LIMIT 5")
        docs = cursor.fetchall()
        
        if not docs:
            print("   No documents found in database.")
            return
            
        print(f"   Found {len(docs)} documents:")
        for doc in docs:
            print(f"     - {doc['doc_id']}: {doc['doc_type']} from {doc['source'][:50]}...")
            
        # Use first document
        test_doc_id = docs[0]['doc_id']
        print(f"\n3. Analyzing document: {test_doc_id}")
        
        # 3. Get all elements for this document
        if has_ordering:
            query = """
                SELECT element_pk, element_id, element_type, parent_id, 
                       content_preview, element_order, document_position, metadata
                FROM elements 
                WHERE doc_id = ?
                ORDER BY document_position, element_order
            """
        else:
            query = """
                SELECT element_pk, element_id, element_type, parent_id, 
                       content_preview, metadata
                FROM elements 
                WHERE doc_id = ?
            """
            
        cursor.execute(query, (test_doc_id,))
        elements = [dict(row) for row in cursor.fetchall()]
        
        print(f"   Found {len(elements)} elements")
        
        # 4. Build hierarchy
        print("\n4. Building document hierarchy...")
        
        # Create lookup maps
        element_by_id = {elem['element_id']: elem for elem in elements}
        children_by_parent = {}
        
        for elem in elements:
            parent_id = elem['parent_id']
            if parent_id not in children_by_parent:
                children_by_parent[parent_id] = []
            children_by_parent[parent_id].append(elem)
            
        # Sort children by order if available
        if has_ordering:
            for parent_id in children_by_parent:
                children_by_parent[parent_id].sort(
                    key=lambda x: (x.get('element_order', 0), x.get('document_position', 0))
                )
        
        # Build tree starting from root
        def build_tree(element_id):
            elem = element_by_id.get(element_id)
            if not elem:
                return None
                
            result = dict(elem)
            children = children_by_parent.get(element_id, [])
            if children:
                result['children'] = [build_tree(child['element_id']) for child in children]
                result['children'] = [c for c in result['children'] if c is not None]
                
            return result
            
        # Find root elements
        roots = [elem for elem in elements if elem['parent_id'] is None]
        hierarchy = [build_tree(root['element_id']) for root in roots]
        
        # 5. Display hierarchy
        print("\n5. DOCUMENT HIERARCHY VISUALIZATION:")
        print("-" * 60)
        print(visualize_hierarchy(hierarchy))
        
        # 6. Generate skeleton markdown
        print("\n6. RECONSTRUCTED DOCUMENT SKELETON:")
        print("-" * 60)
        
        def generate_skeleton(elements: List[Dict[str, Any]], level: int = 0) -> List[str]:
            """Generate markdown skeleton from elements."""
            output = []
            
            for elem in elements:
                if elem['element_type'] == 'header':
                    metadata = json.loads(elem.get('metadata', '{}'))
                    header_level = metadata.get('level', 1)
                    output.append(f"{'#' * header_level} {elem['content_preview']}")
                    output.append("")
                elif elem['element_type'] == 'paragraph':
                    # Show first line of paragraph
                    preview = elem['content_preview'].split('\n')[0]
                    if len(preview) > 100:
                        preview = preview[:100] + "..."
                    output.append(preview)
                    output.append("")
                elif elem['element_type'] == 'list_item':
                    indent = "  " * level
                    output.append(f"{indent}- {elem['content_preview']}")
                    
                # Process children
                if 'children' in elem:
                    child_level = level + 1 if elem['element_type'] == 'list' else level
                    child_output = generate_skeleton(elem['children'], child_level)
                    output.extend(child_output)
                    
            return output
            
        skeleton_lines = generate_skeleton(hierarchy)
        skeleton_text = "\n".join(skeleton_lines)
        print(skeleton_text[:2000])  # Show first 2000 chars
        
        if len(skeleton_text) > 2000:
            print(f"\n... ({len(skeleton_text) - 2000} more characters)")
            
        # 7. Analyze ordering
        if has_ordering:
            print("\n7. ORDERING ANALYSIS:")
            print("-" * 60)
            
            # Check if elements are in order
            positions = [elem.get('document_position', -1) for elem in elements if elem.get('document_position') is not None]
            
            if positions:
                is_ordered = all(positions[i] <= positions[i+1] for i in range(len(positions)-1))
                print(f"   Document positions in order: {'✅ Yes' if is_ordered else '❌ No'}")
                print(f"   Position range: {min(positions)} to {max(positions)}")
                
                # Check sibling ordering within each parent
                print("\n   Sibling ordering by parent:")
                for parent_id, children in children_by_parent.items():
                    if parent_id and len(children) > 1:
                        orders = [c.get('element_order', -1) for c in children]
                        parent_elem = element_by_id.get(parent_id, {})
                        parent_preview = parent_elem.get('content_preview', 'Unknown')[:30]
                        print(f"     Parent '{parent_preview}...': orders {orders}")
            else:
                print("   No ordering data available")
                
        # 8. Summary
        print("\n8. RECONSTRUCTION SUMMARY:")
        print("-" * 60)
        
        # Count element types
        type_counts = {}
        for elem in elements:
            elem_type = elem['element_type']
            type_counts[elem_type] = type_counts.get(elem_type, 0) + 1
            
        print("   Element types found:")
        for elem_type, count in sorted(type_counts.items()):
            print(f"     - {elem_type}: {count}")
            
        print(f"\n   Total elements: {len(elements)}")
        print(f"   Root elements: {len(roots)}")
        print(f"   Ordering available: {'✅ Yes' if has_ordering else '❌ No'}")
        
        if has_ordering and positions:
            print(f"   Order preserved: {'✅ Yes' if is_ordered else '❌ No'}")
            
    finally:
        conn.close()
        
    print("\n" + "="*80)
    print("TEST COMPLETED")
    print("="*80)


if __name__ == "__main__":
    test_reconstruction()