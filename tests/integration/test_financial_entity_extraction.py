#!/usr/bin/env python3
"""
Integration test for financial entity extraction and Neo4j knowledge graph creation.
This test simulates earnings call transcript paragraphs and demonstrates the full pipeline.
"""
import os
import sys
import logging
import tempfile
from pathlib import Path
from typing import List, Dict, Any
import json
from datetime import datetime, timedelta
import random

# Add src to path
sys.path.insert(0, str(Path(__file__).parent.parent.parent / 'src'))

from go_doc_go import Config
from go_doc_go.domain import OntologyManager, OntologyEvaluator
from go_doc_go.storage.sqlite import SQLiteDocumentDatabase
from go_doc_go.relationships.domain import DomainRelationshipDetector
from neo4j_exporter import Neo4jExporter, Neo4jConfig

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


class EarningsCallSimulator:
    """Simulates earnings call transcript paragraphs similar to what would be in a database."""
    
    @staticmethod
    def generate_earnings_call_paragraphs() -> List[Dict[str, Any]]:
        """
        Generate simulated earnings call transcript paragraphs.
        These represent what would be pulled from the database content source.
        """
        # Multiple companies and quarters for rich relationship detection
        companies = [
            ("TechCorp", "TECH", "Sarah Chen", "Michael Rodriguez"),
            ("CloudGiant", "CLDG", "James Wilson", "Emily Davis"),
            ("DataFlow Inc", "DFLW", "Robert Kim", "Lisa Thompson"),
            ("AI Innovations", "AIIN", "David Lee", "Jennifer Martinez"),
            ("ServerMax", "SMAX", "Mark Johnson", "Susan White")
        ]
        
        quarters = ["Q1 2024", "Q2 2024", "Q3 2024", "Q4 2024"]
        
        paragraphs = []
        element_id = 1
        
        for company, ticker, ceo, cfo in companies:
            for quarter in quarters:
                doc_id = f"{ticker.lower()}_{quarter.lower().replace(' ', '_')}_earnings"
                
                # Opening remarks
                paragraphs.append({
                    'element_pk': element_id,
                    'element_id': f'elem_{element_id:06d}',
                    'doc_id': doc_id,
                    'element_type': 'paragraph',
                    'content_preview': f"Good afternoon, and welcome to {company}'s (NASDAQ: {ticker}) {quarter} earnings conference call. "
                                     f"I'm pleased to be joined by {ceo}, our Chief Executive Officer, and {cfo}, our Chief Financial Officer.",
                    'document_position': 1,
                    'metadata': {'speaker': 'IR', 'section': 'opening'}
                })
                element_id += 1
                
                # CEO commentary on revenue
                revenue = random.uniform(1.5, 5.0)
                growth = random.uniform(10, 35)
                paragraphs.append({
                    'element_pk': element_id,
                    'element_id': f'elem_{element_id:06d}',
                    'doc_id': doc_id,
                    'element_type': 'paragraph',
                    'content_preview': f"Thank you for joining us today. {company} delivered strong results in {quarter} with revenue of "
                                     f"${revenue:.1f} billion, representing {growth:.0f}% year-over-year growth. This performance exceeded "
                                     f"our guidance range and demonstrates the strength of our business model.",
                    'document_position': 2,
                    'metadata': {'speaker': ceo, 'section': 'ceo_remarks', 'contains_metrics': True}
                })
                element_id += 1
                
                # Competition and market dynamics
                competitors = [c for c, _, _, _ in companies if c != company]
                main_competitor = random.choice(competitors)
                paragraphs.append({
                    'element_pk': element_id,
                    'element_id': f'elem_{element_id:06d}',
                    'doc_id': doc_id,
                    'element_type': 'paragraph',
                    'content_preview': f"In terms of competitive dynamics, we continue to see pressure from {main_competitor} in the enterprise segment. "
                                     f"However, our differentiated AI-powered solutions and superior customer service have enabled us to maintain "
                                     f"and even expand our market share. We added over 200 new enterprise customers this quarter.",
                    'document_position': 3,
                    'metadata': {'speaker': ceo, 'section': 'competition'}
                })
                element_id += 1
                
                # Partnerships and acquisitions
                partner = random.choice([c for c, _, _, _ in companies if c != company and c != main_competitor])
                paragraphs.append({
                    'element_pk': element_id,
                    'element_id': f'elem_{element_id:06d}',
                    'doc_id': doc_id,
                    'element_type': 'paragraph',
                    'content_preview': f"Our strategic partnership with {partner} announced last quarter is already showing results. "
                                     f"The integration of their technology into our platform has improved customer engagement metrics by 30%. "
                                     f"We're also exploring additional M&A opportunities in the AI and cloud infrastructure space.",
                    'document_position': 4,
                    'metadata': {'speaker': ceo, 'section': 'partnerships'}
                })
                element_id += 1
                
                # CFO financial details
                operating_margin = random.uniform(20, 35)
                eps = random.uniform(2.0, 4.5)
                paragraphs.append({
                    'element_pk': element_id,
                    'element_id': f'elem_{element_id:06d}',
                    'doc_id': doc_id,
                    'element_type': 'paragraph',
                    'content_preview': f"Looking at our financial performance in detail, gross margin expanded to 72%, up 200 basis points year-over-year. "
                                     f"Operating margin came in at {operating_margin:.1f}%, reflecting our disciplined cost management. "
                                     f"Earnings per share was ${eps:.2f}, representing 25% growth compared to the same quarter last year.",
                    'document_position': 5,
                    'metadata': {'speaker': cfo, 'section': 'financial_details', 'contains_metrics': True}
                })
                element_id += 1
                
                # Guidance
                next_quarter_revenue = revenue * random.uniform(1.02, 1.08)
                paragraphs.append({
                    'element_pk': element_id,
                    'element_id': f'elem_{element_id:06d}',
                    'doc_id': doc_id,
                    'element_type': 'paragraph',
                    'content_preview': f"For the next quarter, we're providing guidance of ${next_quarter_revenue:.1f} to ${next_quarter_revenue + 0.1:.1f} billion in revenue. "
                                     f"We expect continued momentum in our cloud services division and anticipate operating margins to remain in the 28-30% range. "
                                     f"This outlook reflects our confidence in the business despite macroeconomic headwinds.",
                    'document_position': 6,
                    'metadata': {'speaker': cfo, 'section': 'guidance', 'forward_looking': True}
                })
                element_id += 1
                
                # Analyst question about competition
                paragraphs.append({
                    'element_pk': element_id,
                    'element_id': f'elem_{element_id:06d}',
                    'doc_id': doc_id,
                    'element_type': 'paragraph',
                    'content_preview': f"Analyst from Goldman Sachs: Can you provide more color on the competitive dynamics with {main_competitor}? "
                                     f"How are you differentiating your offerings, and what's your strategy for maintaining market share?",
                    'document_position': 7,
                    'metadata': {'speaker': 'analyst_gs', 'section': 'qa', 'question': True}
                })
                element_id += 1
                
                # CEO response
                paragraphs.append({
                    'element_pk': element_id,
                    'element_id': f'elem_{element_id:06d}',
                    'doc_id': doc_id,
                    'element_type': 'paragraph',
                    'content_preview': f"{ceo}: Great question. While {main_competitor} competes on price, we're winning on innovation and customer success. "
                                     f"Our AI capabilities are 18 months ahead according to Gartner, and our Net Promoter Score of 72 compares to their 45. "
                                     f"We're not engaging in a price war; instead, we're focused on delivering superior value to our enterprise customers.",
                    'document_position': 8,
                    'metadata': {'speaker': ceo, 'section': 'qa', 'response': True}
                })
                element_id += 1
                
                # Risk discussion
                paragraphs.append({
                    'element_pk': element_id,
                    'element_id': f'elem_{element_id:06d}',
                    'doc_id': doc_id,
                    'element_type': 'paragraph',
                    'content_preview': f"While we're optimistic about our growth prospects, we're monitoring several headwinds including supply chain constraints, "
                                     f"foreign exchange volatility, and potential regulatory changes in Europe. These factors could impact our international expansion plans "
                                     f"and may create margin pressure in the coming quarters.",
                    'document_position': 9,
                    'metadata': {'speaker': cfo, 'section': 'risks'}
                })
                element_id += 1
                
                # Opportunity discussion
                paragraphs.append({
                    'element_pk': element_id,
                    'element_id': f'elem_{element_id:06d}',
                    'doc_id': doc_id,
                    'element_type': 'paragraph',
                    'content_preview': f"On the opportunity side, the enterprise AI market represents a $500 billion total addressable market by 2030. "
                                     f"Our early investments in quantum computing and edge infrastructure position us well to capture this opportunity. "
                                     f"We're particularly excited about our new blockchain platform launching next quarter.",
                    'document_position': 10,
                    'metadata': {'speaker': ceo, 'section': 'opportunities'}
                })
                element_id += 1
        
        return paragraphs
    
    @staticmethod
    def generate_documents(paragraphs: List[Dict[str, Any]]) -> List[Dict[str, Any]]:
        """Generate document records for the paragraphs."""
        docs = {}
        for para in paragraphs:
            doc_id = para['doc_id']
            if doc_id not in docs:
                # Extract company and quarter from doc_id
                parts = doc_id.split('_')
                ticker = parts[0].upper()
                quarter = f"{parts[1].upper()} {parts[2]}"
                
                # Extract year from quarter string
                year = '2024'  # Default year
                quarter_only = quarter
                if ' ' in quarter:
                    parts = quarter.split()
                    quarter_only = parts[0]
                    if len(parts) > 1:
                        year = parts[1]
                
                docs[doc_id] = {
                    'doc_id': doc_id,
                    'doc_type': 'earnings_call',
                    'source': f'{ticker}_{quarter}_transcript.txt',
                    'metadata': {
                        'title': f'{ticker} {quarter} Earnings Call',
                        'date': datetime.now().isoformat(),
                        'ticker': ticker,
                        'company': ticker,  # Add company field
                        'quarter': quarter_only,
                        'year': year,
                        'document_type': 'earnings_call_transcript'
                    },
                    'created_at': datetime.now().isoformat(),
                    'updated_at': datetime.now().isoformat()
                }
        
        return list(docs.values())


def setup_test_database(db_path: str) -> SQLiteDocumentDatabase:
    """Set up test database with schema."""
    db = SQLiteDocumentDatabase(db_path)
    db.initialize()
    
    # Ensure we have the element_term_mappings table
    cursor = db.conn.cursor()
    cursor.execute("""
        CREATE TABLE IF NOT EXISTS element_term_mappings (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            element_pk INTEGER NOT NULL,
            term_id TEXT NOT NULL,
            domain TEXT NOT NULL,
            confidence REAL NOT NULL,
            rule_type TEXT,
            extracted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            metadata TEXT,
            FOREIGN KEY (element_pk) REFERENCES elements(element_pk),
            UNIQUE(element_pk, term_id, domain)
        )
    """)
    
    cursor.execute("""
        CREATE INDEX IF NOT EXISTS idx_etm_element_pk ON element_term_mappings(element_pk)
    """)
    
    cursor.execute("""
        CREATE INDEX IF NOT EXISTS idx_etm_term_domain ON element_term_mappings(term_id, domain)
    """)
    
    db.conn.commit()
    return db


def main():
    """Run the complete entity extraction to Neo4j pipeline test."""
    print("=" * 80)
    print("Financial Entity Extraction and Knowledge Graph Test")
    print("=" * 80)
    
    # Step 1: Generate simulated earnings call data
    print("\n1. Generating simulated earnings call transcript paragraphs...")
    simulator = EarningsCallSimulator()
    paragraphs = simulator.generate_earnings_call_paragraphs()
    documents = simulator.generate_documents(paragraphs)
    print(f"   Generated {len(paragraphs)} paragraphs from {len(documents)} earnings calls")
    
    # Step 2: Set up test database
    print("\n2. Setting up test database...")
    db_path = tempfile.mktemp(suffix='.db')
    db = setup_test_database(db_path)
    
    # Insert documents
    for doc in documents:
        cursor = db.conn.cursor()
        cursor.execute("""
            INSERT INTO documents (doc_id, doc_type, source, metadata, created_at, updated_at)
            VALUES (?, ?, ?, ?, ?, ?)
        """, (doc['doc_id'], doc['doc_type'], doc['source'], 
              json.dumps(doc['metadata']), doc['created_at'], doc['updated_at']))
    
    # Insert elements (paragraphs)
    for para in paragraphs:
        cursor = db.conn.cursor()
        cursor.execute("""
            INSERT INTO elements (element_pk, element_id, doc_id, element_type, 
                                content_preview, document_position, metadata)
            VALUES (?, ?, ?, ?, ?, ?, ?)
        """, (para['element_pk'], para['element_id'], para['doc_id'],
              para['element_type'], para['content_preview'], 
              para['document_position'], json.dumps(para.get('metadata', {}))))
    
    db.conn.commit()
    print(f"   Inserted {len(documents)} documents and {len(paragraphs)} elements")
    
    # Step 3: Load financial markets ontology
    print("\n3. Loading financial markets ontology...")
    manager = OntologyManager()
    ontology_file = Path(__file__).parent.parent.parent / 'examples' / 'ontologies' / 'financial_markets.yaml'
    
    if not ontology_file.exists():
        print(f"Error: Ontology file not found at {ontology_file}")
        return
    
    # Load the raw YAML for neo4j_export config
    import yaml
    with open(ontology_file, 'r') as f:
        ontology_raw = yaml.safe_load(f)
    
    ontology_name = manager.load_ontology(str(ontology_file))
    manager.activate_domain(ontology_name)
    ontology = manager.loader.get_ontology(ontology_name)
    print(f"   Loaded ontology: {ontology_name}")
    print(f"   Terms: {len(ontology.terms)}")
    print(f"   Mapping rules: {len(ontology.element_mappings)}")
    print(f"   Relationship rules: {len(ontology.relationship_rules)}")
    
    # Step 4: Run entity extraction
    print("\n4. Extracting entities using domain ontology...")
    detector = DomainRelationshipDetector(
        db=db,
        ontology_manager=manager,
        embedding_generator=None,  # No embeddings for this test
        config={'min_mapping_confidence': 0.60, 'min_relationship_confidence': 0.60}
    )
    
    all_relationships = []
    all_mappings = []
    
    for doc in documents[:5]:  # Process first 5 documents for demo
        doc_elements = [p for p in paragraphs if p['doc_id'] == doc['doc_id']]
        print(f"   Processing {doc['doc_id']} with {len(doc_elements)} elements...")
        
        relationships = detector.detect_relationships(doc, doc_elements)
        all_relationships.extend(relationships)
        
        # Get the mappings that were created
        for elem in doc_elements:
            elem_mappings = db.get_element_term_mappings(elem['element_pk'])
            all_mappings.extend(elem_mappings)
    
    print(f"   Extracted {len(all_mappings)} entity mappings")
    print(f"   Discovered {len(all_relationships)} relationships")
    
    # Show sample extractions
    if all_mappings:
        print("\n   Sample entity extractions:")
        term_counts = {}
        for mapping in all_mappings:
            term_id = mapping.get('term_id', 'unknown')
            term_counts[term_id] = term_counts.get(term_id, 0) + 1
        
        for term, count in sorted(term_counts.items(), key=lambda x: x[1], reverse=True)[:10]:
            print(f"      {term}: {count} occurrences")
    
    # Step 5: Export to Neo4j
    print("\n5. Exporting to Neo4j knowledge graph...")
    print("   Ensuring Neo4j is running (docker-compose up -d)...")
    
    neo4j_config = Neo4jConfig(
        uri="bolt://localhost:7687",
        username="neo4j",
        password="go-doc-go123"
    )
    
    try:
        with Neo4jExporter(neo4j_config) as exporter:
            # Clear existing data
            exporter.clear_graph()
            exporter.create_indexes()
            
            # Get domain configuration for export from raw YAML
            domain_config = ontology_raw.get('domain', {})
            
            # Export documents with domain config
            doc_count = exporter.export_documents(documents[:5], domain_config)
            print(f"   Exported {doc_count} documents")
            
            # Export elements with domain config
            export_elements = [p for p in paragraphs if p['doc_id'] in [d['doc_id'] for d in documents[:5]]]
            elem_count = exporter.export_elements(export_elements, domain_config)
            print(f"   Exported {elem_count} elements")
            
            # Export terms from ontology
            term_count = exporter.export_terms(ontology.terms, ontology_name)
            print(f"   Exported {term_count} terms")
            
            # Export mappings
            mapping_export = []
            for mapping in all_mappings:
                # Find the element_id for this mapping
                elem_id = next((p['element_id'] for p in paragraphs 
                              if p['element_pk'] == mapping.get('element_pk')), None)
                if elem_id:
                    mapping_export.append({
                        'element_id': elem_id,
                        'term_id': f"{ontology_name}:{mapping.get('term_id')}",
                        'confidence': mapping.get('confidence', 0.0),
                        'rule_type': mapping.get('rule_type', 'unknown')
                    })
            
            map_count = exporter.export_element_term_mappings(mapping_export)
            print(f"   Exported {map_count} element-term mappings")
            
            # Export relationships
            rel_export = []
            for rel in all_relationships:
                rel_export.append({
                    'source_id': rel.get('source_id'),
                    'target_reference': rel.get('target_reference'),
                    'relationship_type': rel.get('relationship_type'),
                    'confidence': rel.get('confidence', 0.0),
                    'rule_id': rel.get('metadata', {}).get('rule_id', ''),
                    'metadata': rel.get('metadata', {})
                })
            
            rel_count = exporter.export_domain_relationships(rel_export)
            print(f"   Exported {rel_count} domain relationships")
            
            # Create term hierarchy
            # Pass the ontology object directly, the exporter will handle it
            ontology_dict = {
                'domain': {'name': ontology_name},
                'relationship_rules': ontology.relationship_rules  # Pass objects directly
            }
            hier_count = exporter.create_term_hierarchy(ontology_dict)
            print(f"   Created {hier_count} term hierarchy relationships")
            
            # Get statistics
            print("\n6. Graph Statistics:")
            stats = exporter.get_graph_statistics()
            print(f"   Total nodes: {stats.get('documents', 0) + stats.get('elements', 0) + stats.get('terms', 0)}")
            print(f"   - Documents: {stats.get('documents', 0)}")
            print(f"   - Elements: {stats.get('elements', 0)}")
            print(f"   - Terms: {stats.get('terms', 0)}")
            print(f"   Total relationships: {sum([stats.get(k, 0) for k in stats if k.endswith('relationships') or k.endswith('_to_doc') or k.endswith('_to_term')])}")
            print(f"   Average mappings per element: {stats.get('avg_mappings_per_element', 0):.2f}")
            
            if stats.get('top_terms'):
                print("\n   Top extracted terms:")
                for term, count in stats['top_terms'][:5]:
                    print(f"      {term}: {count} occurrences")
            
            print("\n✅ Successfully created financial knowledge graph in Neo4j!")
            print("\n   To visualize the graph:")
            print("   1. Open Neo4j Browser at http://localhost:7474")
            print("   2. Login with neo4j/go-doc-go123")
            print("   3. Run queries like:")
            print("      - MATCH (n) RETURN n LIMIT 100")
            print("      - MATCH (c:Term {label:'Company'})-[r]-(e:Element) RETURN c, r, e LIMIT 50")
            print("      - MATCH path = (t1:Term)-[*..3]-(t2:Term) RETURN path LIMIT 20")
            
    except Exception as e:
        print(f"\n❌ Neo4j export failed: {e}")
        print("   Make sure Neo4j is running:")
        print("   cd tests/docker && docker-compose up -d")
    
    finally:
        # Clean up
        db.close()
        if os.path.exists(db_path):
            os.remove(db_path)


if __name__ == '__main__':
    main()