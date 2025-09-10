"""
Domain-based relationship detector using ontology rules.
"""
import logging
from typing import List, Dict, Any, Optional, Tuple
from collections import defaultdict

from ..storage.base import DocumentDatabase
from ..domain import (
    OntologyManager, 
    OntologyEvaluator,
    ElementTermMapping,
    DomainRelationship
)
from ..embeddings.base import EmbeddingGenerator
from .base import RelationshipDetector

logger = logging.getLogger(__name__)


class DomainRelationshipDetector(RelationshipDetector):
    """
    Detects relationships between elements based on domain ontology rules.
    
    This detector:
    1. Maps elements to domain terms using ontology rules
    2. Discovers relationships between mapped elements
    3. Stores both mappings and relationships in the database
    """
    
    def __init__(self, 
                 db: DocumentDatabase,
                 ontology_manager: OntologyManager,
                 embedding_generator: Optional[EmbeddingGenerator] = None,
                 config: Optional[Dict[str, Any]] = None):
        """
        Initialize the domain relationship detector.
        
        Args:
            db: Document database instance
            ontology_manager: Manager for domain ontologies
            embedding_generator: Optional embedding generator for semantic matching
            config: Optional configuration dictionary
        """
        self.db = db
        self.config = config or {}
        self.ontology_manager = ontology_manager
        self.embedding_generator = embedding_generator
        
        # Configuration
        self.batch_size = self.config.get('batch_size', 100)
        self.min_mapping_confidence = self.config.get('min_mapping_confidence', 0.5)
        self.min_relationship_confidence = self.config.get('min_relationship_confidence', 0.6)
        
        # Cache for evaluators
        self._evaluators: Dict[str, OntologyEvaluator] = {}
    
    def detect_relationships(self, document: Dict[str, Any],
                            elements: List[Dict[str, Any]],
                            links: List[Dict[str, Any]] = None) -> List[Dict[str, Any]]:
        """
        Detect domain-based relationships for all elements in a document.
        
        Args:
            document: Document metadata
            elements: Document elements
            links: Optional list of links extracted by the parser
            
        Returns:
            List of discovered relationships
        """
        if not elements:
            doc_id = document.get('doc_id', 'unknown')
            logger.warning(f"No elements found for document {doc_id}")
            return []
        
        # Get active ontologies
        ontologies = self.ontology_manager.get_active_ontologies()
        if not ontologies:
            logger.info("No active domain ontologies")
            return []
        
        all_relationships = []
        
        for ontology in ontologies:
            logger.info(f"Processing ontology: {ontology.name} v{ontology.version}")
            
            # Get or create evaluator for this ontology
            evaluator = self._get_evaluator(ontology)
            
            # Phase 1: Map elements to terms
            element_mappings = self._map_elements_to_terms(elements, evaluator)
            
            if not element_mappings:
                logger.info(f"No elements mapped to terms in {ontology.name}")
                continue
            
            # Store mappings in database
            self._store_mappings(element_mappings)
            
            # Phase 1.5: Extract entities from elements and document
            # Create a pseudo-element for the document itself for document-level entity extraction
            doc_element = {
                'element_id': f"doc_{document.get('doc_id', 'unknown')}",
                'element_type': 'document',
                'doc_id': document.get('doc_id'),
                'metadata': document.get('metadata', {}),
                'content_preview': document.get('source', '')
            }
            elements_with_doc = [doc_element] + elements
            entities = self._extract_entities(elements_with_doc, ontology)
            if entities:
                logger.info(f"Extracted {len(entities)} entities in {ontology.name}")
                
                # Phase 1.6: Create domain relationships between entities BEFORE storing
                # (because _store_entities modifies the entities list by popping source_elements)
                entity_relationships = self._create_entity_relationships(entities, ontology)
                if entity_relationships:
                    logger.info(f"Created {len(entity_relationships)} entity-to-entity relationships in {ontology.name}")
                
                # Now store entities (this will pop source_elements)
                self._store_entities(entities, elements, ontology)
                
                # Store entity relationships after entities are stored
                if entity_relationships:
                    self._store_entity_relationships(entity_relationships, ontology)
            
            # Phase 2: Discover relationships between mapped elements
            relationships = self._discover_relationships(
                element_mappings, elements, evaluator
            )
            
            if relationships:
                logger.info(f"Found {len(relationships)} relationships in {ontology.name}")
                all_relationships.extend(relationships)
        
        return all_relationships
    
    def _get_evaluator(self, ontology) -> OntologyEvaluator:
        """Get or create an evaluator for an ontology."""
        if ontology.name not in self._evaluators:
            self._evaluators[ontology.name] = OntologyEvaluator(
                ontology, 
                self.embedding_generator
            )
        return self._evaluators[ontology.name]
    
    def _map_elements_to_terms(self, elements: List[Dict[str, Any]], 
                               evaluator: OntologyEvaluator) -> List[ElementTermMapping]:
        """
        Map elements to domain terms using ontology rules.
        
        Args:
            elements: List of element dictionaries
            evaluator: Ontology evaluator
            
        Returns:
            List of element-term mappings
        """
        all_mappings = []
        
        for element in elements:
            # Prepare element data for evaluation
            element_data = self._prepare_element_for_mapping(element)
            
            # Get mappings for this element
            mappings = evaluator.map_element_to_terms(element_data)
            
            # Filter by minimum confidence
            filtered_mappings = [
                m for m in mappings 
                if m.confidence >= self.min_mapping_confidence
            ]
            
            if filtered_mappings:
                logger.debug(f"Element {element['element_id']} mapped to {len(filtered_mappings)} terms")
                all_mappings.extend(filtered_mappings)
        
        return all_mappings
    
    def _prepare_element_for_mapping(self, element: Dict[str, Any]) -> Dict[str, Any]:
        """
        Prepare element data for ontology mapping.
        
        Args:
            element: Raw element from database
            
        Returns:
            Element data prepared for evaluation
        """
        # Get element text content
        text = element.get('content_preview', '')
        
        # Get element embedding if available
        embedding = None
        if self.embedding_generator and element.get('element_id'):
            # Try to get existing embedding from database
            embedding = self.db.get_embedding(element['element_id'])
        
        return {
            'element_pk': element['element_pk'],
            'element_id': element['element_id'],
            'element_type': element.get('element_type', ''),
            'text': text,
            'embedding': embedding,
            'document_position': element.get('document_position', 0),
            'parent_id': element.get('parent_id')
        }
    
    def _store_mappings(self, mappings: List[ElementTermMapping]) -> None:
        """Store element-term mappings in the database."""
        # Group mappings by element_pk for efficient storage
        mappings_by_element = defaultdict(list)
        
        for mapping in mappings:
            mappings_by_element[mapping.element_pk].append(mapping.to_dict())
        
        # Store mappings for each element
        for element_pk, element_mappings in mappings_by_element.items():
            try:
                self.db.store_element_term_mappings(element_pk, element_mappings)
            except Exception as e:
                logger.error(f"Failed to store mappings for element {element_pk}: {e}")
    
    def _discover_relationships(self, 
                               mappings: List[ElementTermMapping],
                               elements: List[Dict[str, Any]],
                               evaluator: OntologyEvaluator) -> List[Dict[str, Any]]:
        """
        Discover relationships between elements based on their term mappings.
        
        Args:
            mappings: Element-term mappings
            elements: All elements (for lookup)
            evaluator: Ontology evaluator
            
        Returns:
            List of discovered relationships
        """
        # Create element lookup dictionary
        element_lookup = {e['element_id']: e for e in elements}
        
        # Group elements by their mapped terms
        elements_by_term = defaultdict(list)
        for mapping in mappings:
            element_id = mapping.element_id
            if element_id in element_lookup:
                elements_by_term[mapping.term_id].append(
                    (element_lookup[element_id], mapping)
                )
        
        # Discover relationships using ontology rules
        domain_relationships = evaluator.discover_relationships(
            elements_by_term, element_lookup
        )
        
        # Convert to storage format and filter by confidence
        storage_relationships = []
        for rel in domain_relationships:
            if rel.confidence >= self.min_relationship_confidence:
                storage_relationships.append(rel.to_dict())
        
        return storage_relationships
    
    def detect_cross_document_relationships(self, 
                                          doc_ids: List[str]) -> List[Dict[str, Any]]:
        """
        Detect domain-based relationships across multiple documents.
        
        Args:
            doc_ids: List of document identifiers
            
        Returns:
            List of cross-document relationships
        """
        all_elements = []
        for doc_id in doc_ids:
            elements = self.db.get_elements_by_document(doc_id)
            all_elements.extend(elements)
        
        if not all_elements:
            logger.warning("No elements found for cross-document analysis")
            return []
        
        # Get active ontologies
        ontologies = self.ontology_manager.get_active_ontologies()
        if not ontologies:
            logger.info("No active domain ontologies")
            return []
        
        all_relationships = []
        
        for ontology in ontologies:
            logger.info(f"Cross-document processing with ontology: {ontology.name}")
            
            evaluator = self._get_evaluator(ontology)
            
            # Map all elements to terms
            element_mappings = self._map_elements_to_terms(all_elements, evaluator)
            
            if not element_mappings:
                continue
            
            # Store mappings
            self._store_mappings(element_mappings)
            
            # Discover relationships (including cross-document)
            relationships = self._discover_relationships(
                element_mappings, all_elements, evaluator
            )
            
            # Filter for cross-document relationships
            cross_doc_rels = []
            for rel in relationships:
                source_doc = self._get_element_document(rel['source_id'], all_elements)
                target_doc = self._get_element_document(rel['target_reference'], all_elements)
                
                if source_doc and target_doc and source_doc != target_doc:
                    rel['metadata'] = rel.get('metadata', {})
                    rel['metadata']['cross_document'] = True
                    rel['metadata']['source_doc'] = source_doc
                    rel['metadata']['target_doc'] = target_doc
                    cross_doc_rels.append(rel)
            
            if cross_doc_rels:
                logger.info(f"Found {len(cross_doc_rels)} cross-document relationships in {ontology.name}")
                all_relationships.extend(cross_doc_rels)
        
        return all_relationships
    
    def _get_element_document(self, element_id: str, 
                             elements: List[Dict[str, Any]]) -> Optional[str]:
        """Get the document ID for an element."""
        for element in elements:
            if element['element_id'] == element_id:
                return element.get('doc_id')
        return None
    
    def get_term_usage_report(self, domain: Optional[str] = None) -> Dict[str, Any]:
        """
        Generate a report of term usage across documents.
        
        Args:
            domain: Optional domain to filter by
            
        Returns:
            Report dictionary with statistics
        """
        stats = self.db.get_term_statistics(domain)
        
        # Organize report
        report = {
            'total_terms': len(stats),
            'total_mappings': sum(s['count'] for s in stats.values()),
            'average_confidence': 0.0,
            'terms': []
        }
        
        if stats:
            total_confidence = sum(s['avg_confidence'] * s['count'] for s in stats.values())
            total_count = sum(s['count'] for s in stats.values())
            if total_count > 0:
                report['average_confidence'] = total_confidence / total_count
            
            # Sort terms by usage count
            sorted_terms = sorted(
                stats.items(), 
                key=lambda x: x[1]['count'], 
                reverse=True
            )
            
            for term_key, term_stats in sorted_terms:
                report['terms'].append({
                    'term': term_key,
                    'count': term_stats['count'],
                    'avg_confidence': term_stats['avg_confidence'],
                    'confidence_range': (
                        term_stats['min_confidence'],
                        term_stats['max_confidence']
                    ),
                    'domain': term_stats.get('domain')
                })
        
        return report
    
    def _extract_entities(self, elements: List[Dict[str, Any]], ontology) -> List[Dict[str, Any]]:
        """
        Extract entities from elements based on ontology derived entity rules.
        
        Args:
            elements: List of element dictionaries
            ontology: Domain ontology
            
        Returns:
            List of extracted entities
        """
        if not ontology.derived_entity_rules:
            return []
        
        entities = {}  # For deduplication
        
        for rule in ontology.derived_entity_rules:
            for element in elements:
                # Check if element type matches rule
                if not self._element_matches_rule(element, rule):
                    continue
                
                # Extract entity from element metadata
                entity_data = self._extract_entity_from_element(element, rule)
                if entity_data:
                    # Add doc_id to entity for same_document matching
                    entity_data['doc_id'] = element.get('doc_id')
                    
                    # Use deduplication key to identify unique entities
                    dedup_key = self._get_deduplication_key(entity_data, rule)
                    if dedup_key not in entities:
                        entities[dedup_key] = entity_data
                        entities[dedup_key]['source_elements'] = []
                    
                    entities[dedup_key]['source_elements'].append(element['element_id'])
        
        return list(entities.values())
    
    def _create_entity_relationships(self, entities: List[Dict[str, Any]], ontology) -> List[Dict[str, Any]]:
        """Create domain relationships between entities based on entity relationship rules."""
        if not ontology.entity_relationship_rules:
            return []
        
        relationships = []
        
        # Group entities by type for efficient lookup
        entities_by_type = {}
        for entity in entities:
            entity_type = entity['entity_type']
            if entity_type not in entities_by_type:
                entities_by_type[entity_type] = []
            entities_by_type[entity_type].append(entity)
        
        # Process each entity relationship rule
        for rule in ontology.entity_relationship_rules:
            source_entities = entities_by_type.get(rule.source_entity_type, [])
            target_entities = entities_by_type.get(rule.target_entity_type, [])
            
            if not source_entities or not target_entities:
                continue
            
            # Find matching entity pairs
            for source_entity in source_entities:
                for target_entity in target_entities:
                    if self._entities_match_criteria(source_entity, target_entity, rule):
                        # Create entity-to-entity relationship
                        relationship = {
                            'source_entity_id': source_entity['entity_id'],
                            'target_entity_id': target_entity['entity_id'],
                            'relationship_type': rule.relationship_type,
                            'confidence': rule.confidence,
                            'domain': ontology.name,
                            'metadata': {
                                'rule_name': rule.name,
                                'rule_description': rule.description,
                                'source_entity_type': rule.source_entity_type,
                                'target_entity_type': rule.target_entity_type
                            }
                        }
                        relationships.append(relationship)
        
        return relationships
    
    def _entities_match_criteria(self, source_entity: Dict[str, Any], target_entity: Dict[str, Any], rule) -> bool:
        """Check if two entities match the relationship rule criteria."""
        criteria = rule.matching_criteria
        matches = []
        
        # Check if entities are from the same document
        if criteria.same_document:
            source_doc_id = source_entity.get('doc_id')
            target_doc_id = target_entity.get('doc_id')
            matches.append(source_doc_id and target_doc_id and source_doc_id == target_doc_id)
        
        # Check if entities are derived from the same element
        if criteria.same_source_element:
            source_elements = set(source_entity.get('source_elements', []))
            target_elements = set(target_entity.get('source_elements', []))
            matches.append(bool(source_elements.intersection(target_elements)))
        
        # Check metadata matching
        if criteria.metadata_match:
            metadata_matches = False
            for match_rule in criteria.metadata_match:
                source_field = match_rule.get('source_field')
                target_field = match_rule.get('target_field')
                target_value_required = match_rule.get('target_value')  # Specific value to match
                
                # Handle matching against a specific value (e.g., role name must be "CEO")
                if target_value_required and target_field:
                    target_value = self._get_entity_field_value(target_entity, target_field)
                    if target_value and target_value.lower() == target_value_required.lower():
                        metadata_matches = True
                        break
                # Handle field-to-field matching
                elif source_field and target_field:
                    source_value = self._get_entity_field_value(source_entity, source_field)
                    target_value = self._get_entity_field_value(target_entity, target_field)
                    
                    if source_value and target_value:
                        if source_value.lower() == target_value.lower():
                            metadata_matches = True
                            break
            matches.append(metadata_matches)
        
        # Return True only if all specified criteria match (AND logic)
        return all(matches) if matches else False
    
    def _get_entity_field_value(self, entity: Dict[str, Any], field_name: str) -> Optional[str]:
        """Get field value from entity, checking both top-level fields and attributes."""
        # First check top-level entity fields
        if field_name in entity:
            return str(entity[field_name])
        
        # Then check attributes
        attributes = entity.get('attributes', {})
        if isinstance(attributes, str):
            # Handle JSON string attributes
            try:
                import json
                attributes = json.loads(attributes)
            except (json.JSONDecodeError, ValueError):
                attributes = {}
        
        if isinstance(attributes, dict) and field_name in attributes:
            return str(attributes[field_name])
            
        return None
    
    def _element_matches_rule(self, element: Dict[str, Any], rule) -> bool:
        """Check if element matches the derived entity rule."""
        element_type = element.get('element_type', '')
        
        # Check if element type matches rule's source types
        if '*' in rule.source_element_types:
            return True
        
        return element_type in rule.source_element_types
    
    def _extract_entity_from_element(self, element: Dict[str, Any], rule) -> Optional[Dict[str, Any]]:
        """Extract entity data from element based on rule."""
        metadata = element.get('metadata', {})
        
        # Extract values from specified metadata fields
        entity_attributes = {}
        entity_name = None
        name_parts = []  # Collect parts for composite name
        
        for field in rule.metadata_fields:
            if field in metadata and metadata[field]:
                value = metadata[field]
                
                # Collect all non-empty values for potential composite name
                name_parts.append(str(value))
                
                # Store as attribute
                entity_attributes[field] = value
        
        # Build entity name - use all parts if multiple fields extracted
        if name_parts:
            # Check if rule wants composite name (when using 'name' deduplication and multiple fields)
            if len(name_parts) > 1 and rule.deduplication_key == 'name':
                # Combine all parts for composite name
                entity_name = ' '.join(name_parts)
            else:
                # Use first field value
                entity_name = name_parts[0]
        
        # If no name found and rule allows using element_id as fallback
        if not entity_name:
            # Check if rule's deduplication key is 'element_id' - means it wants to use element_id
            if rule.deduplication_key == 'element_id':
                entity_name = element.get('element_id', 'unknown')
            else:
                return None
        
        # Extract content if requested by rule
        if getattr(rule, 'extract_content', False):
            content_field = getattr(rule, 'content_field', 'content_preview')
            if content_field in element and element[content_field]:
                entity_attributes['content'] = element[content_field]
                # For comments, also store the element_id for uniqueness
                entity_attributes['element_id'] = element.get('element_id')
                entity_attributes['document_position'] = element.get('document_position')
        
        # Create entity data - prepare format dictionary with all possible fields
        format_dict = {
            'entity_type': rule.entity_type,
            'name': self._slugify(entity_name),
            'element_id': element.get('element_id', 'unknown')
        }
        
        # Add all entity attributes to format dict (for templates like {company}_{quarter})
        for key, value in entity_attributes.items():
            format_dict[key] = self._slugify(str(value)) if value else 'unknown'
        
        entity_id = rule.id_template.format(**format_dict)
        
        return {
            'entity_id': entity_id,
            'entity_type': rule.entity_type,
            'name': entity_name,
            'domain': None,  # Will be set when stored
            'attributes': entity_attributes,
            'rule': rule
        }
    
    def _get_deduplication_key(self, entity_data: Dict[str, Any], rule) -> str:
        """Generate deduplication key for entity."""
        if rule.deduplication_key in entity_data['attributes']:
            key_value = entity_data['attributes'][rule.deduplication_key]
            return f"{entity_data['entity_type']}:{self._slugify(str(key_value))}"
        else:
            return f"{entity_data['entity_type']}:{self._slugify(entity_data['name'])}"
    
    def _slugify(self, text: str) -> str:
        """Convert text to slug format."""
        import re
        # Convert to lowercase, replace spaces and special chars with underscore
        slug = re.sub(r'[^\w\s-]', '', str(text).lower())
        slug = re.sub(r'[\s_-]+', '_', slug)
        return slug.strip('_')
    
    def _store_entities(self, entities: List[Dict[str, Any]], 
                       elements: List[Dict[str, Any]], 
                       ontology) -> None:
        """Store entities and their relationships to elements with incremental update support."""
        # Create element lookup
        elements_by_id = {e['element_id']: e for e in elements}
        
        # Get document ID for fetching existing entities
        doc_id = elements[0]['doc_id'] if elements else None
        existing_entities = {}
        if doc_id:
            # Fetch existing entities for this document
            try:
                existing_entities_list = self.db.get_entities_for_document(doc_id)
                existing_entities = {e['entity_id']: e for e in existing_entities_list}
            except Exception as e:
                logger.warning(f"Could not fetch existing entities for document {doc_id}: {e}")
        
        for entity_data in entities:
            # Set domain
            entity_data['domain'] = ontology.name
            rule = entity_data.pop('rule')  # Remove rule from entity data
            source_element_ids = entity_data.pop('source_elements', [])
            entity_id = entity_data['entity_id']
            
            # Generate embedding for entity if generator is available
            if self.embedding_generator:
                # Create text representation for embedding
                entity_text = f"{entity_data.get('entity_type', '')}: {entity_data.get('name', '')}"
                if entity_data.get('attributes'):
                    # Add key attributes to text
                    attrs = entity_data['attributes']
                    if isinstance(attrs, dict):
                        for key, value in attrs.items():
                            if value and key not in ['element_id', 'doc_id']:
                                entity_text += f" {key}={value}"
                
                try:
                    embedding = self.embedding_generator.generate(entity_text)
                    entity_data['embedding'] = embedding
                    entity_data['embedding_model'] = getattr(self.embedding_generator, 'model_name', 'unknown')
                except Exception as e:
                    logger.warning(f"Failed to generate embedding for entity {entity_id}: {e}")
            
            try:
                # Check if entity exists and needs update
                existing_entity = existing_entities.get(entity_id)
                
                if existing_entity:
                    # Entity exists - check if it needs updating
                    if self._entity_needs_update(existing_entity, entity_data):
                        entity_pk = existing_entity['entity_pk']
                        success = self.db.update_entity(entity_pk, entity_data)
                        if success:
                            logger.debug(f"Updated entity {entity_id}")
                        else:
                            logger.warning(f"Failed to update entity {entity_id}")
                    else:
                        entity_pk = existing_entity['entity_pk']
                        logger.debug(f"Entity {entity_id} unchanged")
                else:
                    # New entity - store it
                    entity_pk = self.db.store_entity(entity_data)
                    logger.debug(f"Created new entity {entity_id}")
                
                # Update element-entity mappings
                # First, get existing mappings for this entity
                existing_mappings = set()
                if existing_entity:
                    # We'd need a method to get existing mappings - for now, recreate them
                    # In a full implementation, we'd fetch existing mappings and compare
                    # For now, delete old mappings and create new ones
                    self.db.delete_element_entity_mappings(entity_pk=entity_pk)
                
                # Create element-entity mappings
                for element_id in source_element_ids:
                    if element_id in elements_by_id:
                        element = elements_by_id[element_id]
                        element_pk = self._get_element_pk(element)
                        
                        if element_pk:
                            # DERIVED_FROM relationship (always created)
                            self.db.store_element_entity_mapping({
                                'element_pk': element_pk,
                                'entity_pk': entity_pk,
                                'relationship_type': 'DERIVED_FROM',
                                'domain': ontology.name,
                                'confidence': 1.0,
                                'extraction_method': 'metadata_extraction',
                                'metadata': {}
                            })
                            
                            # Domain-specific relationship (if configured)
                            if rule.create_relationships:
                                self.db.store_element_entity_mapping({
                                    'element_pk': element_pk,
                                    'entity_pk': entity_pk,
                                    'relationship_type': rule.relationship_type,
                                    'domain': ontology.name,
                                    'confidence': 1.0,
                                    'extraction_method': 'metadata_extraction',
                                    'metadata': {}
                                })
                
            except Exception as e:
                logger.error(f"Failed to store/update entity {entity_id}: {e}")
        
        # Handle deleted entities (entities that existed before but are not in current extraction)
        if existing_entities:
            current_entity_ids = {e['entity_id'] for e in entities}
            deleted_entity_ids = set(existing_entities.keys()) - current_entity_ids
            
            for entity_id in deleted_entity_ids:
                entity_pk = existing_entities[entity_id]['entity_pk']
                try:
                    self.db.delete_entity(entity_pk)
                    logger.info(f"Deleted entity {entity_id} (no longer extracted)")
                except Exception as e:
                    logger.error(f"Failed to delete entity {entity_id}: {e}")
    
    def _entity_needs_update(self, existing_entity: Dict[str, Any], new_entity: Dict[str, Any]) -> bool:
        """
        Compare existing entity with new entity data to determine if update is needed.
        
        Args:
            existing_entity: Current entity from database
            new_entity: New entity data from extraction
            
        Returns:
            True if entity needs updating, False otherwise
        """
        # Compare key fields that would indicate a change
        fields_to_compare = ['name', 'entity_type', 'domain']
        
        for field in fields_to_compare:
            if existing_entity.get(field) != new_entity.get(field):
                logger.debug(f"Entity {new_entity.get('entity_id')} field '{field}' changed: "
                           f"'{existing_entity.get(field)}' -> '{new_entity.get(field)}'")
                return True
        
        # Compare attributes (deep comparison)
        existing_attrs = existing_entity.get('attributes', {})
        new_attrs = new_entity.get('attributes', {})
        
        # Handle different attribute formats (could be JSON string or dict)
        if isinstance(existing_attrs, str):
            try:
                import json
                existing_attrs = json.loads(existing_attrs)
            except:
                pass
        
        if existing_attrs != new_attrs:
            logger.debug(f"Entity {new_entity.get('entity_id')} attributes changed")
            return True
        
        # Check if embedding needs regeneration (if model changed)
        if self.embedding_generator:
            existing_model = existing_entity.get('embedding_model')
            new_model = getattr(self.embedding_generator, 'model_name', 'unknown')
            if existing_model != new_model:
                logger.debug(f"Entity {new_entity.get('entity_id')} embedding model changed: "
                           f"'{existing_model}' -> '{new_model}'")
                return True
        
        return False
    
    def _store_entity_relationships(self, relationships: List[Dict[str, Any]], ontology) -> None:
        """Store entity-to-entity relationships in the database with incremental update support."""
        # First, get all existing relationships for entities in this domain
        entity_pks_involved = set()
        entity_id_to_pk = {}
        
        # Build mapping of entity_id to entity_pk for all relationships
        for rel in relationships:
            source_entity = self.db.get_entity(rel['source_entity_id'])
            target_entity = self.db.get_entity(rel['target_entity_id'])
            
            if source_entity:
                entity_id_to_pk[rel['source_entity_id']] = source_entity['entity_pk']
                entity_pks_involved.add(source_entity['entity_pk'])
            if target_entity:
                entity_id_to_pk[rel['target_entity_id']] = target_entity['entity_pk']
                entity_pks_involved.add(target_entity['entity_pk'])
        
        # Get existing relationships for these entities
        existing_relationships = {}
        for entity_pk in entity_pks_involved:
            try:
                entity_rels = self.db.get_entity_relationships(entity_pk)
                for er in entity_rels:
                    # Create a key for the relationship
                    rel_key = (er['source_entity_pk'], er['target_entity_pk'], er['relationship_type'])
                    existing_relationships[rel_key] = er
            except Exception as e:
                logger.warning(f"Could not fetch existing relationships for entity {entity_pk}: {e}")
        
        # Track which relationships we've seen in current extraction
        seen_relationships = set()
        
        for rel in relationships:
            try:
                source_pk = entity_id_to_pk.get(rel['source_entity_id'])
                target_pk = entity_id_to_pk.get(rel['target_entity_id'])
                
                if not source_pk or not target_pk:
                    logger.warning(f"Could not find entities for relationship: {rel['source_entity_id']} -> {rel['target_entity_id']}")
                    continue
                
                rel_key = (source_pk, target_pk, rel['relationship_type'])
                seen_relationships.add(rel_key)
                
                existing_rel = existing_relationships.get(rel_key)
                
                if existing_rel:
                    # Relationship exists - check if it needs updating
                    if (existing_rel.get('confidence') != rel['confidence'] or
                        existing_rel.get('domain') != rel['domain'] or
                        existing_rel.get('metadata') != rel.get('metadata', {})):
                        
                        # Update existing relationship
                        rel_id = existing_rel.get('relationship_id')
                        if rel_id:
                            success = self.db.update_entity_relationship(rel_id, {
                                'source_entity_pk': source_pk,
                                'target_entity_pk': target_pk,
                                'relationship_type': rel['relationship_type'],
                                'confidence': rel['confidence'],
                                'domain': rel['domain'],
                                'metadata': rel.get('metadata', {})
                            })
                            if success:
                                logger.debug(f"Updated entity relationship {rel['source_entity_id']} -> {rel['target_entity_id']}")
                else:
                    # New relationship - store it
                    self.db.store_entity_relationship({
                        'source_entity_pk': source_pk,
                        'target_entity_pk': target_pk,
                        'relationship_type': rel['relationship_type'],
                        'confidence': rel['confidence'],
                        'domain': rel['domain'],
                        'metadata': rel.get('metadata', {})
                    })
                    logger.debug(f"Created entity relationship {rel['source_entity_id']} -> {rel['target_entity_id']}")
                
            except Exception as e:
                logger.error(f"Failed to store/update entity relationship {rel['source_entity_id']} -> {rel['target_entity_id']}: {e}")
        
        # Delete relationships that no longer exist
        for rel_key, existing_rel in existing_relationships.items():
            if rel_key not in seen_relationships and existing_rel.get('domain') == ontology.name:
                # This relationship existed before but not in current extraction
                try:
                    # Delete by source and target
                    count = self.db.delete_entity_relationships(
                        source_entity_pk=rel_key[0],
                        target_entity_pk=rel_key[1]
                    )
                    if count > 0:
                        logger.info(f"Deleted entity relationship (no longer extracted): "
                                  f"{rel_key[0]} -> {rel_key[1]} ({rel_key[2]})")
                except Exception as e:
                    logger.error(f"Failed to delete entity relationship: {e}")
    
    def _get_element_pk(self, element: Dict[str, Any]) -> Optional[int]:
        """Get element primary key from database."""
        try:
            # Try to get from element dict first
            if 'element_pk' in element:
                return element['element_pk']
            
            # Look up by element_id
            element_id = element.get('element_id')
            if element_id:
                stored_element = self.db.get_element_by_id(element_id)
                if stored_element:
                    return stored_element.get('element_pk')
            
            return None
        except Exception as e:
            logger.error(f"Failed to get element_pk for {element.get('element_id')}: {e}")
            return None