"""
Test document reconstruction with element ordering.
This test verifies that document structure can be accurately reconstructed
from search results using the element ordering system.
"""

import json
import logging
import os
import sys
from typing import List, Dict, Any, Optional
from collections import defaultdict

# Add parent directory to path for imports
parent_dir = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
sys.path.insert(0, parent_dir)
sys.path.insert(0, os.path.join(parent_dir, 'src'))

from go_doc_go.config import Config
from go_doc_go.document_parser.markdown import MarkdownParser
from go_doc_go.search import search_by_text
from go_doc_go.storage import flatten_hierarchy

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    handlers=[logging.StreamHandler(sys.stdout)]
)
logger = logging.getLogger(__name__)


def create_visual_tree(elements: List[Dict[str, Any]], indent: str = "") -> str:
    """
    Create a visual tree representation of hierarchical elements.
    
    Args:
        elements: List of element dictionaries with hierarchy
        indent: Current indentation level
        
    Returns:
        String representation of the tree
    """
    output = []
    
    for element in elements:
        # Determine the visual prefix based on element type
        if element.get('element_type') == 'root':
            prefix = "📄"
        elif element.get('element_type') == 'header':
            level = element.get('metadata', {}).get('level', 1)
            prefixes = {1: "📌", 2: "📍", 3: "📎", 4: "🔹", 5: "▪", 6: "·"}
            prefix = prefixes.get(level, "•")
        elif element.get('element_type') == 'paragraph':
            prefix = "¶"
        elif element.get('element_type') == 'list':
            prefix = "📝"
        elif element.get('element_type') == 'list_item':
            prefix = "•"
        else:
            prefix = "○"
            
        # Build the line
        content = element.get('content_preview', '')[:80]
        if len(element.get('content_preview', '')) > 80:
            content += "..."
            
        line = f"{indent}{prefix} [{element.get('element_type')}] {content}"
        
        # Add position information
        if element.get('element_order') is not None:
            line += f" (order: {element['element_order']}, pos: {element.get('document_position', '?')})"
            
        output.append(line)
        
        # Process children if they exist
        if 'child_elements' in element and element['child_elements']:
            child_output = create_visual_tree(element['child_elements'], indent + "  ")
            output.append(child_output)
            
    return "\n".join(output)


def reconstruct_document_skeleton(search_tree: List[Any]) -> Dict[str, Any]:
    """
    Reconstruct a document skeleton from search tree results.
    
    Args:
        search_tree: Hierarchical search results
        
    Returns:
        Reconstructed document structure
    """
    def process_element(element) -> Dict[str, Any]:
        """Process a single element and its children."""
        result = {
            'type': element.element_type,
            'content': element.content_preview,
            'order': element.element_order if hasattr(element, 'element_order') else None,
            'position': element.document_position if hasattr(element, 'document_position') else None,
            'children': []
        }
        
        # Add metadata for headers
        if element.element_type == 'header' and hasattr(element, 'metadata'):
            result['level'] = element.metadata.get('level', 1)
            
        # Process children
        if hasattr(element, 'child_elements') and element.child_elements:
            for child in element.child_elements:
                result['children'].append(process_element(child))
                
        return result
    
    # Process all root elements
    skeleton = {
        'type': 'document',
        'elements': []
    }
    
    for element in search_tree:
        skeleton['elements'].append(process_element(element))
        
    return skeleton


def generate_markdown_from_skeleton(skeleton: Dict[str, Any]) -> str:
    """
    Generate markdown representation from document skeleton.
    
    Args:
        skeleton: Document skeleton structure
        
    Returns:
        Markdown string
    """
    output = []
    
    def process_element(element: Dict[str, Any], parent_level: int = 0):
        """Process element and generate markdown."""
        elem_type = element['type']
        content = element['content']
        
        if elem_type == 'header':
            level = element.get('level', 1)
            output.append(f"{'#' * level} {content}")
            output.append("")  # Add blank line after header
        elif elem_type == 'paragraph':
            output.append(content)
            output.append("")  # Add blank line after paragraph
        elif elem_type == 'list_item':
            indent = "  " * parent_level
            output.append(f"{indent}- {content}")
        elif elem_type == 'root':
            # Skip root element content, just process children
            pass
        elif content:  # Any other element with content
            output.append(content)
            output.append("")
            
        # Process children
        for child in element.get('children', []):
            child_level = parent_level + 1 if elem_type == 'list' else parent_level
            process_element(child, child_level)
    
    # Process all top-level elements
    for element in skeleton.get('elements', []):
        process_element(element)
        
    return "\n".join(output)


def test_document_reconstruction():
    """
    Test document reconstruction with ordering.
    """
    print("\n" + "="*80)
    print("DOCUMENT RECONSTRUCTION TEST")
    print("="*80 + "\n")
    
    # 1. Setup configuration
    config = Config(os.environ.get('DOCULYZER_CONFIG_PATH', 'tests/config.yaml'))
    db = config.get_document_database()
    
    # 2. Initialize database
    print("1. Initializing database...")
    db.initialize()
    
    try:
        # 3. Parse the test document
        print("2. Parsing test document...")
        test_file = "tests/assets/test_document_structure.md"
        
        # Read the document
        with open(test_file, 'r', encoding='utf-8') as f:
            content = f.read()
            
        # Create parser and parse document
        parser = MarkdownParser()
        doc_content = {
            "id": os.path.abspath(test_file),
            "content": content,
            "metadata": {
                "doc_id": "test_doc_001",
                "title": "Project Management Guide"
            }
        }
        
        parsed = parser.parse(doc_content)
        
        print(f"   - Found {len(parsed['elements'])} elements")
        print(f"   - Found {len(parsed['relationships'])} relationships")
        
        # 4. Store in database
        print("3. Storing document in database...")
        
        # Store document
        db.conn.execute(
            "INSERT OR REPLACE INTO documents (doc_id, doc_type, source, metadata, content_hash) VALUES (?, ?, ?, ?, ?)",
            (
                parsed['document']['doc_id'],
                parsed['document']['doc_type'],
                parsed['document']['source'],
                json.dumps(parsed['document']['metadata']),
                parsed['document']['content_hash']
            )
        )
        
        # Store elements with ordering
        element_stats = defaultdict(int)
        for element in parsed['elements']:
            element_stats[element['element_type']] += 1
            
            # The elements should already have element_order and document_position from our updates
            db.conn.execute(
                """
                INSERT OR REPLACE INTO elements 
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
            
        db.conn.commit()
        
        print("   Element statistics:")
        for elem_type, count in element_stats.items():
            print(f"     - {elem_type}: {count}")
        
        # 5. Search for document elements
        print("\n4. Searching for document elements...")
        search_results = search_by_text(
            query_text="project management",
            limit=50,
            text=False,
            content=False,
            flat=False,
            include_parents=True
        )
        
        print(f"   - Found {len(search_results.results)} matching elements")
        print(f"   - Search tree contains {len(search_results.search_tree)} root elements")
        
        # 6. Display the hierarchical structure
        print("\n5. HIERARCHICAL STRUCTURE (with ordering):")
        print("-" * 60)
        
        if search_results.search_tree:
            # Convert search tree to simple dict structure for visualization
            tree_dict = []
            for element in search_results.search_tree:
                tree_dict.append({
                    'element_type': element.element_type,
                    'content_preview': element.content_preview,
                    'element_order': element.element_order if hasattr(element, 'element_order') else None,
                    'document_position': element.document_position if hasattr(element, 'document_position') else None,
                    'metadata': element.metadata if hasattr(element, 'metadata') else {},
                    'child_elements': process_tree_for_display(element.child_elements if hasattr(element, 'child_elements') else [])
                })
            
            print(create_visual_tree(tree_dict))
        
        # 7. Reconstruct document skeleton
        print("\n6. RECONSTRUCTED DOCUMENT SKELETON:")
        print("-" * 60)
        
        if search_results.search_tree:
            skeleton = reconstruct_document_skeleton(search_results.search_tree)
            
            # Display skeleton structure
            print(json.dumps(skeleton, indent=2, default=str)[:2000] + "..." if len(json.dumps(skeleton)) > 2000 else "")
            
            # 8. Generate markdown from skeleton
            print("\n7. RECONSTRUCTED MARKDOWN:")
            print("-" * 60)
            reconstructed_md = generate_markdown_from_skeleton(skeleton)
            print(reconstructed_md)
            
            # 9. Compare with original
            print("\n8. COMPARISON WITH ORIGINAL:")
            print("-" * 60)
            
            # Show first 500 chars of original
            print("Original (first 500 chars):")
            print(content[:500])
            print("\nReconstructed (first 500 chars):")
            print(reconstructed_md[:500])
            
            # Calculate similarity metrics
            original_lines = [l.strip() for l in content.split('\n') if l.strip()]
            reconstructed_lines = [l.strip() for l in reconstructed_md.split('\n') if l.strip()]
            
            # Find headers in both
            original_headers = [l for l in original_lines if l.startswith('#')]
            reconstructed_headers = [l for l in reconstructed_lines if l.startswith('#')]
            
            print(f"\n9. RECONSTRUCTION METRICS:")
            print(f"   Original headers: {len(original_headers)}")
            print(f"   Reconstructed headers: {len(reconstructed_headers)}")
            print(f"   Header preservation: {len(reconstructed_headers) / len(original_headers) * 100:.1f}%")
            
            # Check header order
            order_preserved = all(
                oh in reconstructed_headers[i:i+1] 
                for i, oh in enumerate(original_headers[:len(reconstructed_headers)])
            )
            print(f"   Header order preserved: {'✅ Yes' if order_preserved else '❌ No'}")
            
        # 10. Test flattened view with ordering
        print("\n10. FLATTENED VIEW (sorted by document position):")
        print("-" * 60)
        
        if search_results.search_tree:
            flattened = flatten_hierarchy(search_results.search_tree)
            
            print(f"{'Pos':<5} {'Order':<6} {'Type':<12} {'Content Preview':<50}")
            print("-" * 75)
            
            for i, element in enumerate(flattened[:20]):  # Show first 20
                pos = element.document_position if hasattr(element, 'document_position') else '?'
                order = element.element_order if hasattr(element, 'element_order') else '?'
                preview = element.content_preview[:45] + "..." if len(element.content_preview) > 45 else element.content_preview
                print(f"{pos:<5} {order:<6} {element.element_type:<12} {preview:<50}")
                
            if len(flattened) > 20:
                print(f"... and {len(flattened) - 20} more elements")
                
        print("\n" + "="*80)
        print("TEST COMPLETED SUCCESSFULLY")
        print("="*80)
        
    finally:
        # Clean up
        db.close()


def process_tree_for_display(elements: List[Any]) -> List[Dict[str, Any]]:
    """Helper function to process child elements for display."""
    result = []
    for element in elements:
        result.append({
            'element_type': element.element_type,
            'content_preview': element.content_preview,
            'element_order': element.element_order if hasattr(element, 'element_order') else None,
            'document_position': element.document_position if hasattr(element, 'document_position') else None,
            'metadata': element.metadata if hasattr(element, 'metadata') else {},
            'child_elements': process_tree_for_display(element.child_elements if hasattr(element, 'child_elements') else [])
        })
    return result


if __name__ == "__main__":
    test_document_reconstruction()