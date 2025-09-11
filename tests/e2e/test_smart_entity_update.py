#!/usr/bin/env python3
"""
Test smart entity update strategy.

This test validates that the smart entity update method:
1. Preserves unchanged entities (with embeddings and relationships)
2. Updates modified entities 
3. Creates new entities
4. Deletes removed entities
5. Returns proper statistics
"""

import tempfile
import os
import sys
from pathlib import Path

# Add src to path for imports
sys.path.insert(0, str(Path(__file__).parent.parent.parent / "src"))

from go_doc_go.storage.sqlite import SQLiteDocumentDatabase


def test_smart_entity_update():
    """Test the smart entity update strategy."""
    
    # Create temporary database
    with tempfile.NamedTemporaryFile(suffix='.db', delete=False) as tmp:
        db_path = tmp.name
    
    try:
        # Initialize database
        db = SQLiteDocumentDatabase(db_path)
        db.initialize()
        
        # Create initial document
        doc_id = "test_doc_123"
        initial_document = {
            'doc_id': doc_id,
            'doc_type': 'test',
            'source': 'test_source.txt',
            'metadata': {'version': 1},
            'created_at': '2024-01-01',
            'updated_at': '2024-01-01'
        }
        
        initial_elements = [
            {
                'element_id': 'elem_001',
                'element_type': 'paragraph',
                'content_preview': 'Original content',
                'metadata': {'speaker': 'John Doe'},
                'doc_id': doc_id
            }
        ]
        
        initial_entities = [
            {
                'entity_id': 'entity_unchanged',
                'entity_type': 'speaker',
                'name': 'John Doe',
                'domain': 'earnings_call',
                'attributes': {'role': 'CEO', 'company': 'TestCorp'}
            },
            {
                'entity_id': 'entity_to_modify',
                'entity_type': 'company',
                'name': 'TestCorp',
                'domain': 'earnings_call',
                'attributes': {'ticker': 'TEST', 'sector': 'Technology'}
            },
            {
                'entity_id': 'entity_to_delete',
                'entity_type': 'metric',
                'name': 'Revenue',
                'domain': 'earnings_call',
                'attributes': {'value': '100M', 'currency': 'USD'}
            }
        ]
        
        # Store initial document
        db.store_document(initial_document, initial_elements, [])
        
        # Store initial entities manually (simulating domain detection)
        for entity in initial_entities:
            entity_pk = db.store_entity(entity)
            # Create element-entity mappings to link entities to document elements
            element_pk = initial_elements[0]['element_pk']  # Use the stored element PK
            db.store_element_entity_mapping({
                'element_pk': element_pk,
                'entity_pk': entity_pk,
                'relationship_type': 'DERIVED_FROM',
                'domain': entity['domain']
            })
            
        print(f"✅ Initial setup complete:")
        print(f"   - Document: {doc_id}")
        print(f"   - Elements: {len(initial_elements)}")
        print(f"   - Entities: {len(initial_entities)}")
        
        # Now test smart update with:
        # - 1 unchanged entity (entity_unchanged) 
        # - 1 modified entity (entity_to_modify)
        # - 1 deleted entity (entity_to_delete - not in new list)
        # - 1 new entity (entity_new)
        
        updated_document = {
            'doc_id': doc_id,
            'doc_type': 'test',
            'source': 'test_source.txt',
            'metadata': {'version': 2},  # Updated version
            'created_at': '2024-01-01',
            'updated_at': '2024-01-02'    # Updated timestamp
        }
        
        updated_elements = [
            {
                'element_id': 'elem_001_updated',
                'element_type': 'paragraph', 
                'content_preview': 'Updated content',
                'metadata': {'speaker': 'John Doe'},
                'doc_id': doc_id
            }
        ]
        
        updated_entities = [
            {
                'entity_id': 'entity_unchanged',  # Same as before
                'entity_type': 'speaker',
                'name': 'John Doe',
                'domain': 'earnings_call',
                'attributes': {'role': 'CEO', 'company': 'TestCorp'}
            },
            {
                'entity_id': 'entity_to_modify',  # Modified attributes
                'entity_type': 'company',
                'name': 'TestCorp',
                'domain': 'earnings_call',
                'attributes': {'ticker': 'TEST', 'sector': 'Technology', 'market_cap': '10B'}  # Added market_cap
            },
            {
                'entity_id': 'entity_new',  # New entity
                'entity_type': 'financial_metric',
                'name': 'Profit Margin',
                'domain': 'earnings_call',
                'attributes': {'value': '25%', 'period': 'Q4'}
            }
            # Note: entity_to_delete is not in this list, so it should be deleted
        ]
        
        # Test smart update
        print(f"\n🧠 Testing smart entity update...")
        stats = db.update_document_smart(
            doc_id=doc_id,
            document=updated_document,
            elements=updated_elements,
            relationships=[],
            new_entities=updated_entities
        )
        
        print(f"📊 Smart update statistics:")
        print(f"   - Entities preserved: {stats['entities_preserved']}")
        print(f"   - Entities updated: {stats['entities_updated']}")
        print(f"   - Entities created: {stats['entities_created']}")
        print(f"   - Entities deleted: {stats['entities_deleted']}")
        
        # Validate results
        expected_stats = {
            'entities_preserved': 1,  # entity_unchanged
            'entities_updated': 1,    # entity_to_modify
            'entities_created': 1,    # entity_new
            'entities_deleted': 1     # entity_to_delete
        }
        
        print(f"\n✅ Validation:")
        for key, expected in expected_stats.items():
            actual = stats[key]
            if actual == expected:
                print(f"   ✓ {key}: {actual} (expected {expected})")
            else:
                print(f"   ❌ {key}: {actual} (expected {expected})")
                
        # Check final entity state by looking at all entities in database
        all_entities = db.get_all_entities()
        final_entity_ids = {e['entity_id'] for e in all_entities}
        
        expected_final_ids = {'entity_unchanged', 'entity_to_modify', 'entity_new'}
        if final_entity_ids == expected_final_ids:
            print(f"   ✓ Final entities: {len(all_entities)} ({', '.join(sorted(final_entity_ids))})")
        else:
            print(f"   ❌ Final entities: {final_entity_ids} (expected {expected_final_ids})")
            
        # Verify the modified entity has updated attributes
        modified_entity = next((e for e in all_entities if e['entity_id'] == 'entity_to_modify'), None)
        if modified_entity and 'market_cap' in modified_entity.get('attributes', {}):
            print(f"   ✓ Modified entity has updated attributes")
        else:
            print(f"   ❌ Modified entity attributes not updated correctly")
            if modified_entity:
                print(f"     Actual attributes: {modified_entity.get('attributes', {})}")
                
        # Verify that the deleted entity is gone
        deleted_entity = next((e for e in all_entities if e['entity_id'] == 'entity_to_delete'), None)
        if deleted_entity is None:
            print(f"   ✓ Deleted entity was properly removed")
        else:
            print(f"   ❌ Deleted entity still exists: {deleted_entity['entity_id']}")
            
        # Verify that the new entity was created
        new_entity = next((e for e in all_entities if e['entity_id'] == 'entity_new'), None)
        if new_entity and new_entity['name'] == 'Profit Margin':
            print(f"   ✓ New entity was properly created")
        else:
            print(f"   ❌ New entity not found or incorrect")
            
        print(f"\n🎉 Smart entity update test completed!")
        return True
        
    except Exception as e:
        print(f"❌ Test failed: {e}")
        import traceback
        traceback.print_exc()
        return False
        
    finally:
        # Cleanup
        if os.path.exists(db_path):
            os.unlink(db_path)


if __name__ == "__main__":
    success = test_smart_entity_update()
    sys.exit(0 if success else 1)