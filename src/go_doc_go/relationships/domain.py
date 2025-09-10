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