"""
Direct ingestion bypassing complex imports.
"""

import json
import sqlite3
import os
import hashlib
import uuid
from bs4 import BeautifulSoup
import markdown

def generate_id(prefix=""):
    """Generate unique ID."""
    return f"{prefix}{uuid.uuid4().hex[:8]}"

def generate_hash(content):
    """Generate content hash."""
    return hashlib.md5(content.encode()).hexdigest()

def parse_markdown_direct(content, doc_id, source_path):
    """Parse markdown document directly with ordering."""
    
    # Convert markdown to HTML
    html_content = markdown.markdown(content, extensions=['extra', 'codehilite'])
    soup = BeautifulSoup(html_content, 'html.parser')
    
    elements = []
    relationships = []
    
    # Create root element
    root_id = generate_id("root_")
    elements.append({
        "element_id": root_id,
        "doc_id": doc_id,
        "element_type": "root",
        "parent_id": None,
        "element_order": 0,
        "document_position": 0,
        "content_preview": "",
        "content_location": json.dumps({
            "source": source_path,
            "type": "root"
        }),
        "content_hash": generate_hash(""),
        "metadata": {}
    })
    
    # Track positions
    current_parent = root_id
    parent_element_counts = {root_id: 0}
    global_position = 1
    section_stack = [{"id": root_id, "level": 0}]
    
    # Process HTML elements
    for tag in soup.children:
        if tag.name is None:
            continue
            
        if tag.name in ('h1', 'h2', 'h3', 'h4', 'h5', 'h6'):
            # Header element
            level = int(tag.name[1])
            
            # Adjust parent based on header level
            while section_stack[-1]["level"] >= level:
                section_stack.pop()
            
            current_parent = section_stack[-1]["id"]
            
            # Get element order
            if current_parent not in parent_element_counts:
                parent_element_counts[current_parent] = 0
            element_order = parent_element_counts[current_parent]
            parent_element_counts[current_parent] += 1
            
            # Create header element
            element_id = generate_id(f"h{level}_")
            header_text = tag.get_text().strip()
            
            elements.append({
                "element_id": element_id,
                "doc_id": doc_id,
                "element_type": "header",
                "parent_id": current_parent,
                "element_order": element_order,
                "document_position": global_position,
                "content_preview": header_text[:200],
                "content_location": json.dumps({
                    "source": source_path,
                    "type": "header",
                    "level": level
                }),
                "content_hash": generate_hash(header_text),
                "metadata": {"level": level}
            })
            
            # Create relationship
            relationships.append({
                "relationship_id": generate_id("rel_"),
                "source_id": current_parent,
                "target_id": element_id,
                "relationship_type": "contains",
                "metadata": {}
            })
            
            # Update stack
            section_stack.append({"id": element_id, "level": level})
            current_parent = element_id
            parent_element_counts[element_id] = 0
            
            global_position += 1
            
        elif tag.name == 'p':
            # Paragraph element
            para_text = tag.get_text().strip()
            
            if len(para_text) < 10:
                continue
                
            # Get element order
            if current_parent not in parent_element_counts:
                parent_element_counts[current_parent] = 0
            element_order = parent_element_counts[current_parent]
            parent_element_counts[current_parent] += 1
            
            element_id = generate_id("para_")
            
            elements.append({
                "element_id": element_id,
                "doc_id": doc_id,
                "element_type": "paragraph",
                "parent_id": current_parent,
                "element_order": element_order,
                "document_position": global_position,
                "content_preview": para_text[:200],
                "content_location": json.dumps({
                    "source": source_path,
                    "type": "paragraph"
                }),
                "content_hash": generate_hash(para_text),
                "metadata": {"length": len(para_text)}
            })
            
            # Create relationship
            relationships.append({
                "relationship_id": generate_id("rel_"),
                "source_id": current_parent,
                "target_id": element_id,
                "relationship_type": "contains",
                "metadata": {}
            })
            
            global_position += 1
    
    return elements, relationships


def ingest_test_document():
    """Ingest test document with ordering."""
    
    print("="*80)
    print("SIMPLE DOCUMENT INGESTION WITH ORDERING")
    print("="*80)
    
    # 1. Setup database
    db_path = "tests/data/document_db.sqlite"
    os.makedirs(os.path.dirname(db_path), exist_ok=True)
    
    conn = sqlite3.connect(db_path)
    cursor = conn.cursor()
    
    # 2. Create schema
    print("\n1. Creating database schema...")
    
    cursor.execute("DROP TABLE IF EXISTS relationships")
    cursor.execute("DROP TABLE IF EXISTS elements")  
    cursor.execute("DROP TABLE IF EXISTS documents")
    
    cursor.execute("""
        CREATE TABLE documents (
            doc_id TEXT PRIMARY KEY,
            doc_type TEXT,
            source TEXT,
            metadata TEXT,
            content_hash TEXT
        )
    """)
    
    cursor.execute("""
        CREATE TABLE elements (
            element_pk INTEGER PRIMARY KEY AUTOINCREMENT,
            element_id TEXT UNIQUE NOT NULL,
            doc_id TEXT REFERENCES documents(doc_id),
            element_type TEXT,
            parent_id TEXT,
            element_order INTEGER DEFAULT 0,
            document_position INTEGER DEFAULT 0,
            content_preview TEXT,
            content_location TEXT,
            content_hash TEXT,
            metadata TEXT
        )
    """)
    
    cursor.execute("""
        CREATE TABLE relationships (
            relationship_id TEXT PRIMARY KEY,
            source_id TEXT,
            target_id TEXT,
            relationship_type TEXT,
            metadata TEXT
        )
    """)
    
    print("   ✅ Schema created with ordering columns")
    
    # 3. Read and parse document
    print("\n2. Parsing markdown document...")
    
    test_file = "tests/assets/test_document_structure.md"
    with open(test_file, 'r') as f:
        content = f.read()
    
    doc_id = "test_doc_ordered_001"
    source_path = os.path.abspath(test_file)
    
    elements, relationships = parse_markdown_direct(content, doc_id, source_path)
    
    print(f"   ✅ Parsed {len(elements)} elements with ordering")
    
    # 4. Store in database
    print("\n3. Storing in database...")
    
    # Store document
    cursor.execute(
        "INSERT INTO documents (doc_id, doc_type, source, metadata, content_hash) VALUES (?, ?, ?, ?, ?)",
        (doc_id, "markdown", source_path, json.dumps({"title": "Project Management Guide"}), generate_hash(content))
    )
    
    # Store elements
    for element in elements:
        cursor.execute(
            """INSERT INTO elements 
            (element_id, doc_id, element_type, parent_id, element_order, document_position,
             content_preview, content_location, content_hash, metadata)
            VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)""",
            (
                element['element_id'],
                element['doc_id'], 
                element['element_type'],
                element.get('parent_id'),
                element['element_order'],
                element['document_position'],
                element['content_preview'],
                element['content_location'],
                element['content_hash'],
                json.dumps(element['metadata'])
            )
        )
    
    # Store relationships
    for rel in relationships:
        cursor.execute(
            "INSERT INTO relationships VALUES (?, ?, ?, ?, ?)",
            (rel['relationship_id'], rel['source_id'], rel['target_id'], 
             rel['relationship_type'], json.dumps(rel['metadata']))
        )
    
    conn.commit()
    print(f"   ✅ Stored {len(elements)} elements and {len(relationships)} relationships")
    
    # 5. Verify ordering
    print("\n4. Verifying element ordering...")
    
    cursor.execute("""
        SELECT document_position, element_order, element_type, content_preview
        FROM elements
        WHERE doc_id = ?
        ORDER BY document_position
        LIMIT 15
    """, (doc_id,))
    
    print(f"\n   {'Pos':<5} {'Ord':<5} {'Type':<10} {'Content':<50}")
    print("   " + "-"*70)
    
    for row in cursor.fetchall():
        pos, order, elem_type, preview = row
        preview = preview[:45] + "..." if len(preview) > 45 else preview
        print(f"   {pos:<5} {order:<5} {elem_type:<10} {preview:<50}")
    
    conn.close()
    
    print("\n" + "="*80)
    print("INGESTION COMPLETE - Ready for reconstruction test")
    print("="*80)


if __name__ == "__main__":
    ingest_test_document()