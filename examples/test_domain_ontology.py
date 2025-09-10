#!/usr/bin/env python3
"""
Test script for domain ontology functionality.
"""
import sys
import os
from pathlib import Path

# Add parent directory to path
sys.path.insert(0, str(Path(__file__).parent.parent / 'src'))

from go_doc_go.domain import OntologyManager, OntologyEvaluator
from go_doc_go.storage.sqlite import SQLiteDocumentDatabase
from go_doc_go.embeddings.base import EmbeddingGenerator
from go_doc_go.relationships.domain import DomainRelationshipDetector


def create_test_documents():
    """Create sample automotive documentation."""
    documents = [
        {
            'doc_id': 'brake_manual_001',
            'doc_type': 'manual',
            'source': 'brake_service_manual.pdf',
            'metadata': {'title': 'Brake Service Manual'}
        },
        {
            'doc_id': 'safety_spec_001', 
            'doc_type': 'specification',
            'source': 'safety_requirements.pdf',
            'metadata': {'title': 'Safety Requirements Document'}
        }
    ]
    
    elements = [
        # Brake manual elements
        {
            'element_pk': 1,
            'element_id': 'brake_001',
            'doc_id': 'brake_manual_001',
            'element_type': 'heading',
            'content_preview': 'ABS System Maintenance',
            'document_position': 1
        },
        {
            'element_pk': 2,
            'element_id': 'brake_002',
            'doc_id': 'brake_manual_001',
            'element_type': 'paragraph',
            'content_preview': 'The anti-lock braking system prevents wheel lockup during emergency braking.',
            'document_position': 2
        },
        {
            'element_pk': 3,
            'element_id': 'brake_003',
            'doc_id': 'brake_manual_001',
            'element_type': 'paragraph',
            'content_preview': 'Regular maintenance of brake pads and rotors is essential for safety.',
            'document_position': 3
        },
        {
            'element_pk': 4,
            'element_id': 'brake_004',
            'doc_id': 'brake_manual_001',
            'element_type': 'list_item',
            'content_preview': 'Diagnostic code P0500 indicates a wheel speed sensor malfunction.',
            'document_position': 4
        },
        
        # Safety specification elements
        {
            'element_pk': 5,
            'element_id': 'safety_001',
            'doc_id': 'safety_spec_001',
            'element_type': 'heading',
            'content_preview': 'Braking System Safety Requirements',
            'document_position': 1
        },
        {
            'element_pk': 6,
            'element_id': 'safety_002',
            'doc_id': 'safety_spec_001',
            'element_type': 'paragraph',
            'content_preview': 'The brake system must comply with FMVSS 135 light vehicle brake standards.',
            'document_position': 2
        },
        {
            'element_pk': 7,
            'element_id': 'safety_003',
            'doc_id': 'safety_spec_001',
            'element_type': 'list_item',
            'content_preview': 'Test result: ABS engagement test passed - meets safety requirement.',
            'document_position': 3
        }
    ]
    
    return documents, elements


def main():
    """Test domain ontology functionality."""
    print("=" * 60)
    print("Domain Ontology Test")
    print("=" * 60)
    
    # Initialize database
    db_path = "test_domain.db"
    if os.path.exists(db_path):
        os.remove(db_path)
    
    db = SQLiteDocumentDatabase(db_path)
    db.initialize()
    
    # Create test data
    documents, elements = create_test_documents()
    
    # Store documents and elements
    print("\n1. Storing test documents and elements...")
    for doc in documents:
        # Store in database (simplified - normally would use proper storage methods)
        cursor = db.conn.cursor()
        cursor.execute("""
            INSERT INTO documents (doc_id, doc_type, source, metadata, created_at, updated_at)
            VALUES (?, ?, ?, ?, datetime('now'), datetime('now'))
        """, (doc['doc_id'], doc['doc_type'], doc['source'], 
              str(doc['metadata'])))
    
    for elem in elements:
        cursor = db.conn.cursor()
        cursor.execute("""
            INSERT INTO elements (element_pk, element_id, doc_id, element_type, 
                                content_preview, document_position)
            VALUES (?, ?, ?, ?, ?, ?)
        """, (elem['element_pk'], elem['element_id'], elem['doc_id'],
              elem['element_type'], elem['content_preview'], elem['document_position']))
    
    db.conn.commit()
    print(f"Stored {len(documents)} documents and {len(elements)} elements")
    
    # Load ontology
    print("\n2. Loading automotive ontology...")
    manager = OntologyManager()
    ontology_file = Path(__file__).parent / 'ontologies' / 'automotive.yaml'
    
    if not ontology_file.exists():
        print(f"Error: Ontology file not found at {ontology_file}")
        return
    
    ontology_name = manager.load_ontology(str(ontology_file))
    manager.activate_domain(ontology_name)
    print(f"Loaded and activated ontology: {ontology_name}")
    
    # Get the ontology
    ontology = manager.loader.get_ontology(ontology_name)
    print(f"  Terms: {len(ontology.terms)}")
    print(f"  Mapping rules: {len(ontology.element_mappings)}")
    print(f"  Relationship rules: {len(ontology.relationship_rules)}")
    
    # Create evaluator (without embeddings for simplicity)
    print("\n3. Mapping elements to domain terms...")
    evaluator = OntologyEvaluator(ontology, embedding_provider=None)
    
    # Map elements to terms
    all_mappings = []
    for elem in elements:
        # Prepare element for mapping
        element_data = {
            'element_pk': elem['element_pk'],
            'element_id': elem['element_id'],
            'element_type': elem['element_type'],
            'text': elem['content_preview'],
            'embedding': None  # No embeddings for this test
        }
        
        # Get mappings
        mappings = evaluator.map_element_to_terms(element_data)
        if mappings:
            for mapping in mappings:
                print(f"  {elem['element_id']}: {elem['content_preview'][:50]}...")
                print(f"    -> {mapping.term_id} (confidence: {mapping.confidence:.2f})")
                all_mappings.append(mapping)
    
    # Store mappings in database
    print("\n4. Storing term mappings in database...")
    from collections import defaultdict
    mappings_by_element = defaultdict(list)
    
    for mapping in all_mappings:
        mappings_by_element[mapping.element_pk].append(mapping.to_dict())
    
    for element_pk, element_mappings in mappings_by_element.items():
        db.store_element_term_mappings(element_pk, element_mappings)
    
    print(f"Stored {len(all_mappings)} term mappings")
    
    # Get statistics
    print("\n5. Term usage statistics:")
    stats = db.get_term_statistics(ontology_name)
    for term_key, term_stats in sorted(stats.items(), key=lambda x: x[1]['count'], reverse=True):
        print(f"  {term_key}: {term_stats['count']} elements (avg confidence: {term_stats['avg_confidence']:.2f})")
    
    # Test finding elements by term
    print("\n6. Finding elements by term:")
    for term in ['brake_system', 'safety_requirement', 'diagnostic_code']:
        elements_found = db.find_elements_by_term(term, ontology_name)
        print(f"  {term}: found {len(elements_found)} elements")
        for elem_pk, elem_id, confidence in elements_found[:3]:
            print(f"    - {elem_id} (confidence: {confidence:.2f})")
    
    # Clean up
    db.close()
    print("\n✅ Domain ontology test completed successfully!")


if __name__ == '__main__':
    main()