"""
Entity and relationship extraction using ontology rules.
"""

import json
import logging
import re
from collections import defaultdict
from typing import Dict, List, Any, Optional, Tuple, Set
from dataclasses import dataclass
from datetime import datetime

from go_doc_go.domain.ontology import DomainOntology, RuleType, RelationshipDirection

# Optional embedding support
try:
    from go_doc_go.embedding.client import EmbeddingClient
except ImportError:
    EmbeddingClient = None

logger = logging.getLogger(__name__)


@dataclass
class ExtractedEntity:
    """Represents an extracted entity."""
    entity_id: str
    entity_type: str
    term_id: str
    source_element_id: str
    content: str
    confidence: float
    metadata: Dict[str, Any]
    doc_id: str
    source_name: str
    extracted_at: str


@dataclass
class ExtractedRelationship:
    """Represents an extracted relationship."""
    relationship_id: str
    relationship_type: str
    source_entity_id: str
    target_entity_id: str
    confidence: float
    metadata: Dict[str, Any]
    doc_id: str
    source_name: str
    extracted_at: str


class OntologyEntityExtractor:
    """Extract entities and relationships using ontology rules."""

    def __init__(self, ontology: DomainOntology, embedding_client: Optional[Any] = None):
        """
        Initialize entity extractor.

        Args:
            ontology: Domain ontology with extraction rules
            embedding_client: Optional embedding client for semantic matching
        """
        self.ontology = ontology
        self.embedding_client = embedding_client
        self.extracted_entities = []
        self.extracted_relationships = []

    def extract_from_elements(self, elements: List[Dict[str, Any]],
                             run_id: str = None) -> Dict[str, Any]:
        """
        Extract entities and relationships from document elements.

        Args:
            elements: List of document elements from analytics storage
            run_id: Optional run identifier for tracking

        Returns:
            Dictionary with extracted entities and relationships
        """
        if not run_id:
            run_id = f"extraction_{datetime.now().strftime('%Y%m%d_%H%M%S')}"

        logger.info(f"Starting entity extraction with run_id: {run_id}")
        logger.info(f"Processing {len(elements)} elements with {len(self.ontology.terms)} terms")

        # Reset extraction state
        self.extracted_entities = []
        self.extracted_relationships = []

        # Group elements by document for relationship processing
        elements_by_doc = defaultdict(list)
        for element in elements:
            doc_id = element.get('doc_id', 'unknown')
            elements_by_doc[doc_id].append(element)

        # Extract entities from each element
        total_entities = 0
        for element in elements:
            entities = self._extract_entities_from_element(element, run_id)
            self.extracted_entities.extend(entities)
            total_entities += len(entities)

        logger.info(f"Extracted {total_entities} entities")

        # Extract derived entities from metadata
        derived_entities = self._extract_derived_entities(elements, run_id)
        self.extracted_entities.extend(derived_entities)
        logger.info(f"Extracted {len(derived_entities)} derived entities")

        # Extract relationships between elements
        total_relationships = 0
        for doc_id, doc_elements in elements_by_doc.items():
            relationships = self._extract_relationships_from_document(doc_elements, run_id)
            self.extracted_relationships.extend(relationships)
            total_relationships += len(relationships)

        logger.info(f"Extracted {total_relationships} element relationships")

        # Extract entity-to-entity relationships
        entity_relationships = self._extract_entity_relationships(run_id)
        self.extracted_relationships.extend(entity_relationships)
        logger.info(f"Extracted {len(entity_relationships)} entity relationships")

        return {
            'run_id': run_id,
            'extraction_summary': {
                'total_elements_processed': len(elements),
                'total_entities_extracted': len(self.extracted_entities),
                'total_relationships_extracted': len(self.extracted_relationships),
                'documents_processed': len(elements_by_doc),
                'extraction_timestamp': datetime.now().isoformat()
            },
            'entities': [self._entity_to_dict(e) for e in self.extracted_entities],
            'relationships': [self._relationship_to_dict(r) for r in self.extracted_relationships]
        }

    def _extract_entities_from_element(self, element: Dict[str, Any],
                                      run_id: str) -> List[ExtractedEntity]:
        """Extract entities from a single element using ontology rules."""
        entities = []
        element_type = element.get('element_type', '')
        content = element.get('content_preview', element.get('content', ''))

        if not content:
            return entities

        # Check each term's mapping rules
        for term in self.ontology.terms:
            mapping_rules = self.ontology.get_mapping_rules(term.id)

            for rule in mapping_rules:
                # Check if rule applies to this element type
                if not rule.matches_element_type(element_type):
                    continue

                # Apply rule based on type
                confidence = self._apply_mapping_rule(rule, element, content)

                if confidence >= self.ontology.settings.default_confidence_threshold:
                    # Create extracted entity
                    entity_id = f"{term.id}_{element.get('element_id', 'unknown')}_{len(entities)}"

                    entity = ExtractedEntity(
                        entity_id=entity_id,
                        entity_type=term.id,
                        term_id=term.id,
                        source_element_id=element.get('element_id', ''),
                        content=content,
                        confidence=confidence,
                        metadata={
                            'rule_type': rule.type.value,
                            'element_type': element_type,
                            'structural_name': element.get('structural_name'),
                            'hierarchy_depth': element.get('hierarchy_depth'),
                            'position': element.get('position'),
                            'extraction_rule': rule.type.value
                        },
                        doc_id=element.get('doc_id', ''),
                        source_name=element.get('source_name', ''),
                        extracted_at=datetime.now().isoformat()
                    )

                    entities.append(entity)

        return entities

    def _apply_mapping_rule(self, rule, element: Dict[str, Any], content: str) -> float:
        """Apply a mapping rule and return confidence score."""

        if rule.type == RuleType.SEMANTIC:
            # Use embedding similarity for semantic matching
            if self.embedding_client and rule.semantic_phrase:
                try:
                    similarity = self.embedding_client.compute_similarity(
                        content, rule.semantic_phrase
                    )
                    return similarity
                except Exception as e:
                    logger.debug(f"Embedding similarity failed: {e}")
                    return 0.0
            else:
                # Fallback to keyword matching
                phrase_words = set(rule.semantic_phrase.lower().split())
                content_words = set(content.lower().split())
                overlap = len(phrase_words.intersection(content_words))
                return overlap / len(phrase_words) if phrase_words else 0.0

        elif rule.type == RuleType.REGEX:
            # Apply regex pattern
            pattern = rule.get_pattern()
            if pattern and pattern.search(content):
                return 1.0  # Regex either matches or doesn't
            return 0.0

        elif rule.type == RuleType.KEYWORDS:
            # Check keyword matches
            pattern = rule.get_pattern()
            if pattern:
                matches = pattern.findall(content)
                if matches:
                    # Calculate confidence based on number of matches
                    unique_matches = set(match[0] if isinstance(match, tuple) else match
                                       for match in matches)
                    return min(1.0, len(unique_matches) / len(rule.keywords))
            return 0.0

        return 0.0

    def _extract_derived_entities(self, elements: List[Dict[str, Any]],
                                 run_id: str) -> List[ExtractedEntity]:
        """Extract derived entities from element metadata."""
        derived_entities = []

        for rule in self.ontology.derived_entity_rules:
            # Find elements that match the source types
            matching_elements = []
            for element in elements:
                element_type = element.get('element_type', '')
                if ('*' in rule.source_element_types or
                    element_type in rule.source_element_types):
                    matching_elements.append(element)

            # Extract entities from matching elements
            entity_cache = {}  # For deduplication

            for element in matching_elements:
                metadata = element.get('metadata', {})
                if not metadata:
                    continue

                # Extract values from specified metadata fields
                extracted_values = {}
                for field in rule.metadata_fields:
                    if field in metadata:
                        extracted_values[field] = metadata[field]

                if not extracted_values:
                    continue

                # Generate entity ID using template
                try:
                    # Create template variables
                    template_vars = {
                        'entity_type': rule.entity_type,
                        **extracted_values
                    }

                    entity_id = rule.id_template.format(**template_vars)

                    # Check for deduplication
                    dedup_key = extracted_values.get(rule.deduplication_key)
                    if dedup_key and dedup_key in entity_cache:
                        continue  # Skip duplicate

                    # Create derived entity
                    content = extracted_values.get('name', str(extracted_values))
                    if rule.extract_content:
                        content = element.get(rule.content_field, content)

                    derived_entity = ExtractedEntity(
                        entity_id=entity_id,
                        entity_type=rule.entity_type,
                        term_id=rule.entity_type,
                        source_element_id=element.get('element_id', ''),
                        content=content,
                        confidence=0.90,  # High confidence for metadata extraction
                        metadata={
                            'derived_from': 'metadata',
                            'source_fields': rule.metadata_fields,
                            'extracted_values': extracted_values,
                            'element_type': element.get('element_type')
                        },
                        doc_id=element.get('doc_id', ''),
                        source_name=element.get('source_name', ''),
                        extracted_at=datetime.now().isoformat()
                    )

                    derived_entities.append(derived_entity)

                    # Cache for deduplication
                    if dedup_key:
                        entity_cache[dedup_key] = derived_entity

                except Exception as e:
                    logger.debug(f"Failed to create derived entity: {e}")
                    continue

        return derived_entities

    def _extract_relationships_from_document(self, elements: List[Dict[str, Any]],
                                           run_id: str) -> List[ExtractedRelationship]:
        """Extract relationships between elements in a document."""
        relationships = []

        # Get entities from these elements
        doc_entities = [e for e in self.extracted_entities
                       if e.source_element_id in [elem.get('element_id') for elem in elements]]

        if len(doc_entities) < 2:
            return relationships

        # Apply relationship rules
        for rule in self.ontology.relationship_rules:
            relationships.extend(
                self._apply_relationship_rule(rule, doc_entities, elements, run_id)
            )

        return relationships

    def _apply_relationship_rule(self, rule, entities: List[ExtractedEntity],
                               elements: List[Dict[str, Any]], run_id: str) -> List[ExtractedRelationship]:
        """Apply a relationship rule to extract relationships."""
        relationships = []

        # Find source and target entities
        source_entities = [e for e in entities if e.term_id == rule.source.term_id]
        target_entities = [e for e in entities if e.term_id == rule.target.term_id]

        if not source_entities or not target_entities:
            return relationships

        # Create element lookup for constraint checking
        element_lookup = {e.get('element_id'): e for e in elements}

        for source_entity in source_entities:
            for target_entity in target_entities:
                if source_entity.entity_id == target_entity.entity_id:
                    continue  # Skip self-relationships

                # Check constraints
                if not self._check_relationship_constraints(
                    rule.constraints, source_entity, target_entity, element_lookup
                ):
                    continue

                # Calculate confidence
                source_confidence = self._calculate_relationship_confidence(
                    rule.source, source_entity
                )
                target_confidence = self._calculate_relationship_confidence(
                    rule.target, target_entity
                )

                combined_confidence = rule.confidence.calculate(
                    source_confidence, target_confidence
                )

                if combined_confidence >= rule.confidence.minimum:
                    # Create relationship
                    rel_id = f"{rule.id}_{source_entity.entity_id}_{target_entity.entity_id}"

                    relationship = ExtractedRelationship(
                        relationship_id=rel_id,
                        relationship_type=rule.relationship_type,
                        source_entity_id=source_entity.entity_id,
                        target_entity_id=target_entity.entity_id,
                        confidence=combined_confidence,
                        metadata={
                            'rule_id': rule.id,
                            'source_confidence': source_confidence,
                            'target_confidence': target_confidence,
                            'extraction_method': 'ontology_rule'
                        },
                        doc_id=source_entity.doc_id,
                        source_name=source_entity.source_name,
                        extracted_at=datetime.now().isoformat()
                    )

                    relationships.append(relationship)

                    # Add bidirectional relationship if specified
                    if rule.bidirectional:
                        reverse_rel_id = f"{rule.id}_{target_entity.entity_id}_{source_entity.entity_id}"
                        reverse_relationship = ExtractedRelationship(
                            relationship_id=reverse_rel_id,
                            relationship_type=rule.relationship_type,
                            source_entity_id=target_entity.entity_id,
                            target_entity_id=source_entity.entity_id,
                            confidence=combined_confidence,
                            metadata={
                                'rule_id': rule.id,
                                'source_confidence': target_confidence,
                                'target_confidence': source_confidence,
                                'extraction_method': 'ontology_rule',
                                'bidirectional': True
                            },
                            doc_id=source_entity.doc_id,
                            source_name=source_entity.source_name,
                            extracted_at=datetime.now().isoformat()
                        )
                        relationships.append(reverse_relationship)

        return relationships

    def _check_relationship_constraints(self, constraints, source_entity: ExtractedEntity,
                                      target_entity: ExtractedEntity,
                                      element_lookup: Dict[str, Dict]) -> bool:
        """Check if relationship constraints are satisfied."""
        if not constraints:
            return True

        # Check hierarchy level constraint
        if constraints.hierarchy_level is not None:
            source_element = element_lookup.get(source_entity.source_element_id)
            target_element = element_lookup.get(target_entity.source_element_id)

            if source_element and target_element:
                if constraints.hierarchy_level == -1:
                    # Same document
                    if source_entity.doc_id != target_entity.doc_id:
                        return False
                else:
                    # Check hierarchy relationship
                    source_path = source_element.get('structural_path', '')
                    target_path = target_element.get('structural_path', '')

                    if not self._check_hierarchy_level(
                        source_path, target_path, constraints.hierarchy_level
                    ):
                        return False

        # Check direction constraint
        if constraints.direction != RelationshipDirection.ANY:
            source_element = element_lookup.get(source_entity.source_element_id)
            target_element = element_lookup.get(target_entity.source_element_id)

            if source_element and target_element:
                source_pos = source_element.get('position', 0)
                target_pos = target_element.get('position', 0)

                if constraints.direction == RelationshipDirection.FORWARD:
                    if source_pos >= target_pos:
                        return False
                elif constraints.direction == RelationshipDirection.BACKWARD:
                    if source_pos <= target_pos:
                        return False

        return True

    def _check_hierarchy_level(self, source_path: str, target_path: str, level: int) -> bool:
        """Check if two elements are at the specified hierarchy level."""
        if level == 0:
            # Same parent - check if paths have same parent
            source_parts = source_path.split('/')[:-1]  # Remove element itself
            target_parts = target_path.split('/')[:-1]
            return source_parts == target_parts

        # For other levels, check common ancestor depth
        source_parts = source_path.split('/')
        target_parts = target_path.split('/')

        common_depth = 0
        for i in range(min(len(source_parts), len(target_parts))):
            if source_parts[i] == target_parts[i]:
                common_depth += 1
            else:
                break

        # level 1 = same grandparent, etc.
        required_depth = len(source_parts) - level - 1
        return common_depth >= required_depth

    def _calculate_relationship_confidence(self, endpoint, entity: ExtractedEntity) -> float:
        """Calculate confidence for relationship endpoint."""
        if not endpoint.semantic_phrase:
            return entity.confidence

        # Use semantic similarity if available
        if self.embedding_client:
            try:
                similarity = self.embedding_client.compute_similarity(
                    entity.content, endpoint.semantic_phrase
                )
                return similarity
            except Exception:
                pass

        # Fallback to simple word matching
        phrase_words = set(endpoint.semantic_phrase.lower().split())
        content_words = set(entity.content.lower().split())
        overlap = len(phrase_words.intersection(content_words))
        return (overlap / len(phrase_words)) if phrase_words else entity.confidence

    def _extract_entity_relationships(self, run_id: str) -> List[ExtractedRelationship]:
        """Extract relationships between domain entities."""
        relationships = []

        # Group entities by type and document
        entities_by_type = defaultdict(list)
        for entity in self.extracted_entities:
            entities_by_type[entity.entity_type].append(entity)

        # Apply entity relationship rules
        for rule in self.ontology.entity_relationship_rules:
            source_entities = entities_by_type.get(rule.source_entity_type, [])
            target_entities = entities_by_type.get(rule.target_entity_type, [])

            for source_entity in source_entities:
                for target_entity in target_entities:
                    if source_entity.entity_id == target_entity.entity_id:
                        continue

                    # Check matching criteria
                    if self._check_entity_matching_criteria(
                        rule.matching_criteria, source_entity, target_entity
                    ):
                        rel_id = f"entity_{rule.name}_{source_entity.entity_id}_{target_entity.entity_id}"

                        relationship = ExtractedRelationship(
                            relationship_id=rel_id,
                            relationship_type=rule.relationship_type,
                            source_entity_id=source_entity.entity_id,
                            target_entity_id=target_entity.entity_id,
                            confidence=rule.confidence,
                            metadata={
                                'rule_name': rule.name,
                                'extraction_method': 'entity_rule',
                                'matching_criteria': rule.matching_criteria.__dict__
                            },
                            doc_id=source_entity.doc_id,
                            source_name=source_entity.source_name,
                            extracted_at=datetime.now().isoformat()
                        )

                        relationships.append(relationship)

                        # Add bidirectional if specified
                        if rule.bidirectional:
                            reverse_rel_id = f"entity_{rule.name}_{target_entity.entity_id}_{source_entity.entity_id}"
                            reverse_relationship = ExtractedRelationship(
                                relationship_id=reverse_rel_id,
                                relationship_type=rule.relationship_type,
                                source_entity_id=target_entity.entity_id,
                                target_entity_id=source_entity.entity_id,
                                confidence=rule.confidence,
                                metadata={
                                    'rule_name': rule.name,
                                    'extraction_method': 'entity_rule',
                                    'matching_criteria': rule.matching_criteria.__dict__,
                                    'bidirectional': True
                                },
                                doc_id=source_entity.doc_id,
                                source_name=source_entity.source_name,
                                extracted_at=datetime.now().isoformat()
                            )
                            relationships.append(reverse_relationship)

        return relationships

    def _check_entity_matching_criteria(self, criteria, source_entity: ExtractedEntity,
                                       target_entity: ExtractedEntity) -> bool:
        """Check if entities match the specified criteria."""

        # Same source element
        if criteria.same_source_element:
            if source_entity.source_element_id != target_entity.source_element_id:
                return False

        # Same document
        if criteria.same_document:
            if source_entity.doc_id != target_entity.doc_id:
                return False

        # Metadata matching
        for match_spec in criteria.metadata_match:
            source_field = match_spec.get('source_field')
            target_field = match_spec.get('target_field')

            if source_field and target_field:
                source_value = source_entity.metadata.get(source_field)
                target_value = target_entity.metadata.get(target_field)

                if source_value != target_value:
                    return False

        return True

    def _entity_to_dict(self, entity: ExtractedEntity) -> Dict[str, Any]:
        """Convert ExtractedEntity to dictionary."""
        return {
            'entity_id': entity.entity_id,
            'entity_type': entity.entity_type,
            'term_id': entity.term_id,
            'source_element_id': entity.source_element_id,
            'content': entity.content,
            'confidence': entity.confidence,
            'metadata': entity.metadata,
            'doc_id': entity.doc_id,
            'source_name': entity.source_name,
            'extracted_at': entity.extracted_at
        }

    def _relationship_to_dict(self, relationship: ExtractedRelationship) -> Dict[str, Any]:
        """Convert ExtractedRelationship to dictionary."""
        return {
            'relationship_id': relationship.relationship_id,
            'relationship_type': relationship.relationship_type,
            'source_entity_id': relationship.source_entity_id,
            'target_entity_id': relationship.target_entity_id,
            'confidence': relationship.confidence,
            'metadata': relationship.metadata,
            'doc_id': relationship.doc_id,
            'source_name': relationship.source_name,
            'extracted_at': relationship.extracted_at
        }