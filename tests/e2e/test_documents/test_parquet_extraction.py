#!/usr/bin/env python3
"""Test parquet parsing and entity extraction."""

from go_doc_go.document_parser.parquet import ParquetParser
from go_doc_go.storage.sqlite import SQLiteDocumentDatabase
from go_doc_go.domain import OntologyManager
from go_doc_go.relationships.domain import DomainRelationshipDetector
import tempfile
import os
import json

# Create test database
db_path = tempfile.mktemp(suffix='.db')
db = SQLiteDocumentDatabase(db_path)
db.initialize()

# Parse the parquet file
parser = ParquetParser()
content = {
    'id': 'techcorp_earnings',
    'binary_path': 'techcorp_q4_2024_earnings.parquet',
    'metadata': {}
}

result = parser.parse(content)
print(f'✅ Parsed {len(result["elements"])} elements')

# Check metadata
print('\n📊 Checking element metadata:')
speaker_count = 0
for elem in result['elements']:
    meta = elem.get('metadata', {})
    if 'speaker' in meta:
        speaker_count += 1
        if speaker_count <= 3:
            print(f'  • {meta["speaker"]} ({meta.get("speaker_role", "Unknown")})')

print(f'\nTotal elements with speakers: {speaker_count}')

# Store in database
doc_data = result['document']
doc_data['source'] = 'techcorp_q4_2024_earnings.parquet'
doc_data['created_at'] = doc_data['updated_at'] = '2024-02-15'

cursor = db.conn.cursor()
cursor.execute('INSERT INTO documents (doc_id, doc_type, source, metadata, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)',
              (doc_data['doc_id'], doc_data['doc_type'], doc_data['source'], 
               json.dumps(doc_data['metadata']), doc_data['created_at'], doc_data['updated_at']))

# Store elements with proper metadata
element_pk = 1
for elem in result['elements']:
    cursor.execute(
        'INSERT INTO elements (element_pk, element_id, doc_id, element_type, parent_id, content_preview, metadata) VALUES (?, ?, ?, ?, ?, ?, ?)',
        (element_pk, elem['element_id'], doc_data['doc_id'], elem['element_type'], 
         elem.get('parent_id'), elem.get('content_preview', ''), json.dumps(elem.get('metadata', {})))
    )
    element_pk += 1

db.conn.commit()

# Now run entity extraction
print('\n🎯 Running entity extraction...')
manager = OntologyManager()
ontology_path = '../../../examples/ontologies/financial_markets.yaml'
ontology_name = manager.load_ontology(ontology_path)
manager.activate_domain(ontology_name)

# Get elements from DB to ensure proper structure
elements = db.get_document_elements(doc_data['doc_id'])

detector = DomainRelationshipDetector(db=db, ontology_manager=manager, embedding_generator=None, config={})
relationships = detector.detect_relationships(doc_data, elements)

# Check entities
entities = db.get_all_entities()
print(f'\n📈 Extracted {len(entities)} entities:')

# Group by type
by_type = {}
for entity in entities:
    entity_type = entity['entity_type']
    if entity_type not in by_type:
        by_type[entity_type] = []
    by_type[entity_type].append(entity['name'])

for entity_type, names in sorted(by_type.items()):
    print(f'  • {entity_type}: {names[:5]}...' if len(names) > 5 else f'  • {entity_type}: {names}')

# Check entity relationships  
entity_rels = db.get_all_entity_relationships()
print(f'\n🔗 Entity relationships: {len(entity_rels)}')
rel_types = {}
for rel in entity_rels:
    rel_type = rel['relationship_type']
    rel_types[rel_type] = rel_types.get(rel_type, 0) + 1

for rel_type, count in sorted(rel_types.items()):
    print(f'  • {rel_type}: {count}')

# Clean up
db.close()
os.remove(db_path)
print('\n✅ Test complete!')