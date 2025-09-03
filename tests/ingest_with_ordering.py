"""
Simple ingestion script to test document ordering.
This script ingests a markdown document with the new ordering fields.
"""

import json
import sqlite3
import os
import sys

# Add parent directory to path
parent_dir = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
sys.path.insert(0, parent_dir)
sys.path.insert(0, os.path.join(parent_dir, 'src'))

def ingest_test_document():
    """Ingest the test markdown document with ordering."""
    
    print("="*80)
    print("DOCUMENT INGESTION WITH ORDERING")
    print("="*80)
    
    # 1. Create/connect to database
    db_path = "tests/data/document_db.sqlite"
    os.makedirs(os.path.dirname(db_path), exist_ok=True)
    
    conn = sqlite3.connect(db_path)
    cursor = conn.cursor()
    
    # 2. Create tables with ordering columns
    print("\n1. Creating database schema with ordering columns...")
    
    # Drop existing tables
    cursor.execute("DROP TABLE IF EXISTS relationships")
    cursor.execute("DROP TABLE IF EXISTS elements")
    cursor.execute("DROP TABLE IF EXISTS documents")
    
    # Create documents table
    cursor.execute("""
        CREATE TABLE documents (
            doc_id TEXT PRIMARY KEY,
            doc_type TEXT,
            source TEXT,
            metadata TEXT,
            content_hash TEXT,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )
    """)
    
    # Create elements table with ordering columns
    cursor.execute("""
        CREATE TABLE elements (
            element_pk INTEGER PRIMARY KEY AUTOINCREMENT,
            element_id TEXT UNIQUE NOT NULL,
            doc_id TEXT REFERENCES documents(doc_id) ON DELETE CASCADE,
            element_type TEXT,
            parent_id TEXT REFERENCES elements(element_id),
            element_order INTEGER DEFAULT 0,
            document_position INTEGER DEFAULT 0,
            content_preview TEXT,
            content_location TEXT,
            content_hash TEXT,
            metadata TEXT
        )
    """)
    
    # Create relationships table
    cursor.execute("""
        CREATE TABLE relationships (
            relationship_id TEXT PRIMARY KEY,
            source_id TEXT,
            target_id TEXT,
            relationship_type TEXT,
            metadata TEXT
        )
    """)
    
    print("   ✅ Schema created with element_order and document_position columns")
    
    # 3. Parse the test document
    print("\n2. Parsing test markdown document...")
    
    # Import the markdown parser
    from go_doc_go.document_parser.markdown import MarkdownParser
    
    test_file = "tests/assets/test_document_structure.md"
    
    # Read the document
    with open(test_file, 'r', encoding='utf-8') as f:
        content = f.read()
    
    # Create parser and parse
    parser = MarkdownParser(config={'extract_dates': False})  # Disable date extraction to avoid dependency
    doc_content = {
        "id": os.path.abspath(test_file),
        "content": content,
        "metadata": {
            "doc_id": "test_markdown_001",
            "title": "Project Management Guide"
        }
    }
    
    parsed = parser.parse(doc_content)
    
    print(f"   ✅ Parsed {len(parsed['elements'])} elements")
    
    # 4. Store in database
    print("\n3. Storing document with ordering information...")
    
    # Store document
    cursor.execute(
        "INSERT INTO documents (doc_id, doc_type, source, metadata, content_hash) VALUES (?, ?, ?, ?, ?)",
        (
            parsed['document']['doc_id'],
            parsed['document']['doc_type'],
            parsed['document']['source'],
            json.dumps(parsed['document']['metadata']),
            parsed['document']['content_hash']
        )
    )
    
    # Store elements with ordering
    elements_with_order = 0
    for element in parsed['elements']:
        # Check if ordering fields are present
        has_order = 'element_order' in element and element['element_order'] is not None
        has_position = 'document_position' in element and element['document_position'] is not None
        
        if has_order and has_position:
            elements_with_order += 1
        
        cursor.execute(
            """
            INSERT INTO elements 
            (element_id, doc_id, element_type, parent_id, content_preview, 
             content_location, content_hash, metadata, element_order, document_position)
            VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
            """,
            (
                element['element_id'],
                element['doc_id'],
                element['element_type'],
                element.get('parent_id'),
                element.get('content_preview', ''),
                element.get('content_location', ''),
                element.get('content_hash', ''),
                json.dumps(element.get('metadata', {})),
                element.get('element_order', 0),
                element.get('document_position', 0)
            )
        )
    
    print(f"   ✅ Stored {len(parsed['elements'])} elements")
    print(f"   ✅ Elements with ordering: {elements_with_order}/{len(parsed['elements'])}")
    
    # Store relationships
    for relationship in parsed['relationships']:
        cursor.execute(
            "INSERT INTO relationships (relationship_id, source_id, target_id, relationship_type, metadata) VALUES (?, ?, ?, ?, ?)",
            (
                relationship['relationship_id'],
                relationship['source_id'],
                relationship['target_id'],
                relationship['relationship_type'],
                json.dumps(relationship.get('metadata', {}))
            )
        )
    
    print(f"   ✅ Stored {len(parsed['relationships'])} relationships")
    
    conn.commit()
    
    # 5. Verify ordering data
    print("\n4. Verifying ordering data...")
    
    cursor.execute("""
        SELECT element_type, element_order, document_position, content_preview
        FROM elements
        WHERE doc_id = ?
        ORDER BY document_position
        LIMIT 10
    """, (parsed['document']['doc_id'],))
    
    print("\n   First 10 elements by document position:")
    print(f"   {'Pos':<5} {'Order':<6} {'Type':<12} {'Preview':<40}")
    print("   " + "-"*65)
    
    for row in cursor.fetchall():
        elem_type, elem_order, doc_pos, preview = row
        preview_short = preview[:35] + "..." if len(preview) > 35 else preview
        print(f"   {doc_pos:<5} {elem_order:<6} {elem_type:<12} {preview_short:<40}")
    
    # Check parent-child ordering
    print("\n5. Checking sibling ordering...")
    
    cursor.execute("""
        SELECT parent_id, COUNT(*) as child_count
        FROM elements
        WHERE parent_id IS NOT NULL
        GROUP BY parent_id
        HAVING child_count > 1
        LIMIT 3
    """)
    
    parents_with_multiple_children = cursor.fetchall()
    
    for parent_id, child_count in parents_with_multiple_children:
        cursor.execute("""
            SELECT element_type, element_order, content_preview
            FROM elements
            WHERE parent_id = ?
            ORDER BY element_order
        """, (parent_id,))
        
        children = cursor.fetchall()
        print(f"\n   Parent '{parent_id[:20]}...' has {child_count} children:")
        for child_type, child_order, child_preview in children[:3]:
            preview_short = child_preview[:30] + "..." if len(child_preview) > 30 else child_preview
            print(f"     Order {child_order}: [{child_type}] {preview_short}")
    
    conn.close()
    
    print("\n" + "="*80)
    print("INGESTION COMPLETED SUCCESSFULLY")
    print("="*80)
    print("\nDatabase ready at:", db_path)
    print("Run 'python tests/test_reconstruction_simple.py' to test reconstruction")


if __name__ == "__main__":
    ingest_test_document()