#!/usr/bin/env python3
"""
End-to-End Test for Domain Entity Extraction Pipeline

This test demonstrates the complete flow:
1. Document ingestion from file system
2. Document parsing into elements and relationships (universal doc)
3. Domain entity extraction using financial ontology
4. Entity relationship creation
5. Export to Neo4j
6. Update handling (re-ingestion and entity updates)
"""

import os
import sys
import json
import shutil
import tempfile
from pathlib import Path
from datetime import datetime

# Add src to path
sys.path.insert(0, str(Path(__file__).parent.parent.parent / 'src'))

from go_doc_go import Config
from go_doc_go.main import ingest_documents
from go_doc_go.storage.sqlite import SQLiteDocumentDatabase
from go_doc_go.domain import OntologyManager
from go_doc_go.relationships.domain import DomainRelationshipDetector

# Import Neo4j exporter
sys.path.insert(0, str(Path(__file__).parent.parent / 'integration'))
from neo4j_exporter import Neo4jExporter


def setup_test_environment():
    """Set up test directories and files."""
    # Create test directories
    test_dir = Path(__file__).parent
    test_docs_dir = test_dir / 'test_documents'
    test_docs_dir.mkdir(exist_ok=True)
    
    # Ensure sample document exists
    sample_doc = test_docs_dir / 'techcorp_q4_2024_earnings.parquet'
    if not sample_doc.exists():
        print(f"❌ Sample document not found: {sample_doc}")
        print("   Please create the sample earnings call parquet file first")
        # Try to create it if it doesn't exist
        create_script = test_docs_dir / 'create_earnings_parquet.py'
        if create_script.exists():
            import subprocess
            print("   Creating parquet file...")
            subprocess.run([sys.executable, str(create_script)], cwd=test_docs_dir)
        if not sample_doc.exists():
            return False
    
    return True


def run_e2e_pipeline(config_path: str, clear_db: bool = True):
    """Run the complete end-to-end pipeline."""
    print("\n" + "="*80)
    print("🚀 RUNNING END-TO-END ENTITY EXTRACTION PIPELINE")
    print("="*80)
    
    # Load configuration
    print("\n📋 Loading configuration...")
    config = Config(config_path)
    
    # Clear database if requested
    storage_config = config.get('storage', {})
    db_path_str = storage_config.get('path', './test_e2e.db')
    
    if clear_db and db_path_str != ':memory:':
        db_path = Path(db_path_str)
        if db_path.exists():
            print(f"🗑️  Clearing existing database: {db_path}")
            if db_path.is_dir():
                shutil.rmtree(db_path)
            else:
                os.remove(db_path)
    
    # Initialize database
    print("💾 Initializing database...")
    config.initialize_database()
    db = SQLiteDocumentDatabase(db_path_str)
    db.initialize()  # Make sure to initialize the connection
    
    # Phase 1: Document Ingestion
    print("\n📄 PHASE 1: Document Ingestion")
    print("-" * 40)
    
    # Get content sources from config
    content_sources = config.content_sources
    print(f"   Content sources: {len(content_sources)}")
    for source in content_sources:
        print(f"   - {source['name']}: {source['type']} from {source.get('base_path', 'N/A')}")
    
    # Ingest documents
    result = ingest_documents(config)
    print(f"\n   ✅ Ingestion Results:")
    print(f"      Documents: {result.get('documents', 0)}")
    print(f"      Elements: {result.get('elements', 0)}")
    print(f"      Relationships: {result.get('relationships', 0)}")
    
    # Phase 2: Show Document Structure (Universal Doc)
    print("\n🏗️  PHASE 2: Document Structure (Universal Doc)")
    print("-" * 40)
    
    # Get all documents
    documents = db.find_documents()
    for doc in documents:
        print(f"\n   Document: {doc['doc_id']}")
        print(f"   Type: {doc['doc_type']}")
        print(f"   Source: {doc['source']}")
        
        # Get elements for this document
        elements = db.get_document_elements(doc['doc_id'])
        print(f"   Elements: {len(elements)}")
        
        # Show element types
        element_types = {}
        for elem in elements:
            elem_type = elem['element_type']
            element_types[elem_type] = element_types.get(elem_type, 0) + 1
        
        for elem_type, count in sorted(element_types.items()):
            print(f"      - {elem_type}: {count}")
    
    # Phase 3: Domain Entity Extraction
    print("\n🎯 PHASE 3: Domain Entity Extraction")
    print("-" * 40)
    
    # Load domain ontology
    manager = OntologyManager()
    ontology_path = Path(__file__).parent.parent.parent / 'examples' / 'ontologies' / 'financial_markets.yaml'
    ontology_name = manager.load_ontology(str(ontology_path))
    manager.activate_domain(ontology_name)
    print(f"   Loaded ontology: {ontology_name}")
    
    # Run domain entity extraction
    detector = DomainRelationshipDetector(
        db=db,
        ontology_manager=manager,
        embedding_generator=None,
        config=config.relationship_detection
    )
    
    # Process each document
    for doc in documents:
        elements = db.get_document_elements(doc['doc_id'])
        relationships = detector.detect_relationships(doc, elements)
        print(f"\n   Document: {doc['doc_id']}")
        print(f"   Domain relationships detected: {len(relationships)}")
    
    # Show extracted entities
    entities = db.get_all_entities()
    print(f"\n   📊 Extracted Entities: {len(entities)}")
    
    # Group by type
    entities_by_type = {}
    for entity in entities:
        entity_type = entity['entity_type']
        if entity_type not in entities_by_type:
            entities_by_type[entity_type] = []
        entities_by_type[entity_type].append(entity)
    
    for entity_type, type_entities in sorted(entities_by_type.items()):
        print(f"\n   {entity_type.upper()} ({len(type_entities)}):")
        for entity in type_entities[:3]:  # Show first 3 of each type
            print(f"      - {entity['name']} [{entity['entity_id']}]")
        if len(type_entities) > 3:
            print(f"      ... and {len(type_entities) - 3} more")
    
    # Show entity relationships
    entity_rels = db.get_all_entity_relationships()
    print(f"\n   🔗 Entity Relationships: {len(entity_rels)}")
    
    # Group by relationship type
    rels_by_type = {}
    for rel in entity_rels:
        rel_type = rel['relationship_type']
        rels_by_type[rel_type] = rels_by_type.get(rel_type, 0) + 1
    
    for rel_type, count in sorted(rels_by_type.items()):
        print(f"      - {rel_type}: {count}")
    
    # Phase 4: Neo4j Export
    print("\n🌐 PHASE 4: Neo4j Export")
    print("-" * 40)
    
    try:
        # Create Neo4j exporter
        neo4j_config = config.get('neo4j', {})
        if neo4j_config.get('enabled', False):
            # Neo4jExporter expects specific connection parameters
            neo4j_params = {
                'uri': neo4j_config.get('uri', 'bolt://localhost:7687'),
                'username': neo4j_config.get('username', 'neo4j'),
                'password': neo4j_config.get('password', 'password'),
                'database': neo4j_config.get('database', 'neo4j')
            }
            exporter = Neo4jExporter(neo4j_params)
            
            # Clear graph if requested
            if neo4j_config.get('clear_before_export', False):
                print("   Clearing Neo4j graph...")
                exporter.clear_graph()
            
            # Export entities
            print("   Exporting entities...")
            entity_count = exporter.export_entities(entities)
            print(f"   ✅ Exported {entity_count} entities")
            
            # Export entity relationships
            print("   Exporting entity relationships...")
            rel_count = exporter.export_entity_relationships(entity_rels)
            print(f"   ✅ Exported {rel_count} entity relationships")
            
            # Export element-entity mappings (DERIVED_FROM)
            # Get all unique element PKs from entities' source_elements
            mappings = []
            unique_element_pks = set()
            
            # First get all elements for all documents to get their PKs
            for doc in documents:
                elements = db.get_document_elements(doc['doc_id'])
                for elem in elements:
                    if 'element_pk' in elem:
                        elem_mappings = db.get_element_entity_mappings(elem['element_pk'])
                        mappings.extend(elem_mappings)
            
            print("   Exporting DERIVED_FROM relationships...")
            mapping_count = exporter.export_element_entity_mappings(mappings)
            print(f"   ✅ Exported {mapping_count} DERIVED_FROM relationships")
            
            print(f"\n   🎉 Neo4j export complete!")
            print(f"   View at: http://localhost:7474/browser/")
            print(f"   Credentials: neo4j / {neo4j_config.get('password', 'password')}")
            
            exporter.close()
        else:
            print("   Neo4j export disabled in configuration")
            
    except Exception as e:
        print(f"   ❌ Neo4j export failed: {e}")
    
    return {
        'documents': len(documents),
        'elements': sum(len(db.get_document_elements(d['doc_id'])) for d in documents),
        'entities': len(entities),
        'entity_relationships': len(entity_rels),
        'db': db
    }


def test_update_scenario(config_path: str):
    """Test entity update scenario - what happens when documents are re-ingested."""
    print("\n" + "="*80)
    print("🔄 TESTING UPDATE SCENARIO")
    print("="*80)
    
    # First run - initial ingestion
    print("\n1️⃣  Initial Ingestion:")
    initial_results = run_e2e_pipeline(config_path, clear_db=True)
    
    # Modify the document (simulate an update)
    test_docs_dir = Path(__file__).parent / 'test_documents'
    sample_doc = test_docs_dir / 'techcorp_q4_2024_earnings.md'
    
    print("\n2️⃣  Modifying document (adding analyst)...")
    
    # Read original content
    with open(sample_doc, 'r') as f:
        content = f.read()
    
    # Add a new analyst question at the end of Q&A
    additional_qa = """
**Lisa Thompson, Analyst - Barclays:** How are you thinking about international expansion, particularly in the APAC region?

**Sarah Chen, CEO:** Great question, Lisa. APAC represents a significant growth opportunity for us. We're planning to establish local partnerships in key markets like Japan and Singapore in 2025, with a focus on adapting our solutions to local regulatory requirements.
"""
    
    # Insert before closing remarks
    modified_content = content.replace("## Closing Remarks", additional_qa + "\n## Closing Remarks")
    
    # Save modified document
    with open(sample_doc, 'w') as f:
        f.write(modified_content)
    
    # Re-run ingestion
    print("\n3️⃣  Re-ingesting modified document:")
    config = Config(config_path)
    config.initialize_database()
    
    # Re-ingest (this should update existing entities and add new ones)
    result = ingest_documents(config)
    print(f"   Re-ingestion complete: {result}")
    
    # Check what changed
    storage_config = config.get('storage', {})
    db_path_str = storage_config.get('path', './test_e2e.db')
    db = SQLiteDocumentDatabase(db_path_str)
    db.initialize()  # Make sure to initialize the connection
    
    # Get entities after update
    entities_after = db.get_all_entities()
    entities_after_by_type = {}
    for entity in entities_after:
        entity_type = entity['entity_type']
        if entity_type not in entities_after_by_type:
            entities_after_by_type[entity_type] = []
        entities_after_by_type[entity_type].append(entity['name'])
    
    print("\n4️⃣  Comparison:")
    print(f"   Entities before: {initial_results['entities']}")
    print(f"   Entities after: {len(entities_after)}")
    
    # Check for new analyst
    if 'speaker' in entities_after_by_type:
        speakers = entities_after_by_type['speaker']
        print(f"\n   Speakers after update: {speakers}")
        if 'Lisa Thompson' in speakers:
            print("   ✅ New analyst entity detected: Lisa Thompson")
        else:
            print("   ⚠️  New analyst not found in entities")
    
    # Restore original document
    with open(sample_doc, 'w') as f:
        f.write(content)
    
    return entities_after


def main():
    """Main entry point for E2E test."""
    # Setup test environment
    if not setup_test_environment():
        sys.exit(1)
    
    # Config path
    config_path = Path(__file__).parent / 'config_e2e_test.yaml'
    
    # Run main pipeline
    results = run_e2e_pipeline(str(config_path), clear_db=True)
    
    # Test update scenario
    print("\n" + "="*80)
    print("Do you want to test the UPDATE scenario? (y/n)")
    
    # Auto-run for testing
    test_update_scenario(str(config_path))
    
    print("\n" + "="*80)
    print("✅ END-TO-END TEST COMPLETE")
    print("="*80)
    print("\nSummary:")
    print(f"  - Documents processed: {results['documents']}")
    print(f"  - Elements extracted: {results['elements']}")
    print(f"  - Entities created: {results['entities']}")
    print(f"  - Entity relationships: {results['entity_relationships']}")
    print("\nThe complete pipeline from document ingestion to Neo4j export is working!")
    
    # Cleanup
    db_path = Path('./test_e2e.db')
    if db_path.exists():
        print(f"\n🧹 Cleaning up test database: {db_path}")
        os.remove(db_path)


if __name__ == "__main__":
    main()