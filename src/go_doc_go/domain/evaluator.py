"""
Evaluator for applying domain ontology rules to elements.
"""
import re
import numpy as np
from typing import Dict, List, Optional, Any, Tuple
from dataclasses import dataclass
import logging

from .ontology import (
    DomainOntology, 
    MappingRule, 
    RuleType,
    RelationshipRule
)

logger = logging.getLogger(__name__)


@dataclass
class ElementTermMapping:
    """Result of mapping an element to a term."""
    element_pk: int
    element_id: str
    term_id: str
    domain: str
    confidence: float
    mapping_rule: str  # 'semantic', 'regex', or 'keywords'
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary for storage."""
        return {
            'element_pk': self.element_pk,
            'element_id': self.element_id,
            'term': self.term_id,
            'domain': self.domain,
            'confidence': self.confidence,
            'mapping_rule': self.mapping_rule
        }


@dataclass
class DomainRelationship:
    """A discovered domain relationship between elements."""
    source_element_id: str
    target_element_id: str
    relationship_type: str
    domain: str
    confidence: float
    source_term: str
    target_term: str
    metadata: Dict[str, Any]
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary for storage."""
        return {
            'source_id': self.source_element_id,
            'target_reference': self.target_element_id,
            'relationship_type': f"domain:{self.relationship_type}",
            'metadata': {
                **self.metadata,
                'domain': self.domain,
                'source_term': self.source_term,
                'target_term': self.target_term,
                'confidence': self.confidence
            }
        }


class OntologyEvaluator:
    """
    Evaluates elements against domain ontology rules.
    """
    
    def __init__(self, ontology: DomainOntology, 
                 embedding_provider: Optional[Any] = None):
        """
        Initialize the evaluator.
        
        Args:
            ontology: Domain ontology to use
            embedding_provider: Optional provider for semantic embeddings
        """
        self.ontology = ontology
        self.embedding_provider = embedding_provider
        
        # Cache for semantic phrase embeddings
        self._phrase_embeddings: Dict[str, np.ndarray] = {}
    
    def map_element_to_terms(self, element: Dict[str, Any]) -> List[ElementTermMapping]:
        """
        Map an element to domain terms based on mapping rules.
        
        Args:
            element: Element dictionary with keys:
                     - element_pk: int
                     - element_id: str
                     - element_type: str
                     - text: str (element text content)
                     - embedding: np.ndarray (optional, for semantic matching)
                     
        Returns:
            List of term mappings (can be multiple terms per element)
        """
        mappings = []
        
        for term_id, rules in self.ontology._mappings_by_term.items():
            for rule in rules:
                # Check element type filter
                if not rule.matches_element_type(element.get('element_type', '')):
                    continue
                
                # Evaluate rule
                confidence = self._evaluate_mapping_rule(element, rule)
                
                if confidence is not None:
                    mapping = ElementTermMapping(
                        element_pk=element['element_pk'],
                        element_id=element['element_id'],
                        term_id=term_id,
                        domain=self.ontology.name,
                        confidence=confidence,
                        mapping_rule=rule.type.value
                    )
                    mappings.append(mapping)
                    logger.debug(f"Mapped element {element['element_id']} to term {term_id} "
                               f"with confidence {confidence:.3f} using {rule.type.value}")
        
        return mappings
    
    def discover_relationships(self, 
                              elements_with_terms: Dict[str, List[Tuple[Dict, ElementTermMapping]]],
                              element_lookup: Dict[str, Dict[str, Any]]) -> List[DomainRelationship]:
        """
        Discover relationships between elements based on their term mappings.
        
        Args:
            elements_with_terms: Dictionary mapping term_id to list of (element, mapping) tuples
            element_lookup: Dictionary for quick element lookup by ID
            
        Returns:
            List of discovered domain relationships
        """
        relationships = []
        
        for rule in self.ontology.relationship_rules:
            # Get candidate source and target elements
            source_candidates = elements_with_terms.get(rule.source.term_id, [])
            target_candidates = elements_with_terms.get(rule.target.term_id, [])
            
            for source_elem, source_mapping in source_candidates:
                for target_elem, target_mapping in target_candidates:
                    # Skip self-relationships
                    if source_elem['element_id'] == target_elem['element_id']:
                        continue
                    
                    # Check constraints
                    if not self._check_constraints(source_elem, target_elem, rule.constraints):
                        continue
                    
                    # Evaluate relationship
                    relationship = self._evaluate_relationship_rule(
                        source_elem, target_elem, 
                        source_mapping, target_mapping, 
                        rule
                    )
                    
                    if relationship:
                        relationships.append(relationship)
                        
                        # Add reverse relationship if bidirectional
                        if rule.bidirectional:
                            reverse = DomainRelationship(
                                source_element_id=relationship.target_element_id,
                                target_element_id=relationship.source_element_id,
                                relationship_type=relationship.relationship_type,
                                domain=relationship.domain,
                                confidence=relationship.confidence,
                                source_term=relationship.target_term,
                                target_term=relationship.source_term,
                                metadata={**relationship.metadata, 'bidirectional': True}
                            )
                            relationships.append(reverse)
        
        return relationships
    
    def _evaluate_mapping_rule(self, element: Dict[str, Any], rule: MappingRule) -> Optional[float]:
        """
        Evaluate a single mapping rule against an element.
        
        Returns:
            Confidence score if rule matches, None otherwise
        """
        element_text = element.get('text', '')
        
        if rule.type == RuleType.SEMANTIC:
            # Semantic similarity matching
            if not self.embedding_provider:
                logger.warning("No embedding provider for semantic matching")
                return None
                
            element_embedding = element.get('embedding')
            if element_embedding is None:
                return None
                
            # Get or compute phrase embedding
            phrase_embedding = self._get_phrase_embedding(rule.semantic_phrase)
            if phrase_embedding is None:
                return None
                
            # Calculate similarity
            similarity = self._cosine_similarity(element_embedding, phrase_embedding)
            
            # Check threshold
            if similarity >= rule.confidence_threshold:
                return similarity
            return None
            
        elif rule.type == RuleType.REGEX:
            # Regex pattern matching
            pattern = rule.get_pattern()
            if pattern and pattern.search(element_text):
                return 1.0
            return None
            
        elif rule.type == RuleType.KEYWORDS:
            # Keywords matching (internally uses regex)
            pattern = rule.get_pattern()
            if pattern and pattern.search(element_text):
                return 1.0
            return None
            
        else:
            logger.warning(f"Unknown rule type: {rule.type}")
            return None
    
    def _evaluate_relationship_rule(self, 
                                   source_elem: Dict[str, Any],
                                   target_elem: Dict[str, Any],
                                   source_mapping: ElementTermMapping,
                                   target_mapping: ElementTermMapping,
                                   rule: RelationshipRule) -> Optional[DomainRelationship]:
        """
        Evaluate a relationship rule for a pair of elements.
        
        Returns:
            DomainRelationship if rule matches with sufficient confidence, None otherwise
        """
        if not self.embedding_provider:
            logger.warning("No embedding provider for relationship evaluation")
            return None
        
        # Get element embeddings
        source_embedding = source_elem.get('embedding')
        target_embedding = target_elem.get('embedding')
        
        if source_embedding is None or target_embedding is None:
            return None
        
        # Evaluate source semantic similarity
        source_phrase_embedding = self._get_phrase_embedding(rule.source.semantic_phrase)
        if source_phrase_embedding is None:
            return None
        source_similarity = self._cosine_similarity(source_embedding, source_phrase_embedding)
        
        if source_similarity < rule.source.confidence_threshold:
            return None
        
        # Evaluate target semantic similarity
        target_phrase_embedding = self._get_phrase_embedding(rule.target.semantic_phrase)
        if target_phrase_embedding is None:
            return None
        target_similarity = self._cosine_similarity(target_embedding, target_phrase_embedding)
        
        if target_similarity < rule.target.confidence_threshold:
            return None
        
        # Calculate combined confidence
        combined_confidence = rule.confidence.calculate(source_similarity, target_similarity)
        
        if combined_confidence < rule.confidence.minimum:
            return None
        
        # Create relationship
        return DomainRelationship(
            source_element_id=source_elem['element_id'],
            target_element_id=target_elem['element_id'],
            relationship_type=rule.relationship_type,
            domain=self.ontology.name,
            confidence=combined_confidence,
            source_term=rule.source.term_id,
            target_term=rule.target.term_id,
            metadata={
                'rule_id': rule.id,
                'source_similarity': source_similarity,
                'target_similarity': target_similarity,
                'source_mapping_confidence': source_mapping.confidence,
                'target_mapping_confidence': target_mapping.confidence
            }
        )
    
    def _check_constraints(self, source_elem: Dict[str, Any], 
                          target_elem: Dict[str, Any],
                          constraints: Optional[Any]) -> bool:
        """
        Check if element pair satisfies relationship constraints.
        """
        if not constraints:
            return True
        
        # Check hierarchy constraint
        if constraints.hierarchy_level is not None:
            # -1 = same document
            if constraints.hierarchy_level == -1:
                source_doc = source_elem.get('doc_id')
                target_doc = target_elem.get('doc_id')
                if source_doc != target_doc:
                    return False
            
            # 0 = same parent, 1 = same grandparent, etc.
            elif constraints.hierarchy_level >= 0:
                if not self._check_shared_ancestor(source_elem, target_elem, 
                                                  constraints.hierarchy_level):
                    return False
        
        # Check direction constraint
        if constraints.direction.value == 'forward':
            source_pos = source_elem.get('document_position', 0)
            target_pos = target_elem.get('document_position', 0)
            if source_pos >= target_pos:
                return False
                
        elif constraints.direction.value == 'backward':
            source_pos = source_elem.get('document_position', 0)
            target_pos = target_elem.get('document_position', 0)
            if source_pos <= target_pos:
                return False
        
        return True
    
    def _check_shared_ancestor(self, elem1: Dict[str, Any], elem2: Dict[str, Any], 
                               level: int) -> bool:
        """
        Check if two elements share an ancestor at the specified level.
        
        Args:
            elem1: First element
            elem2: Second element  
            level: Ancestor level (0=parent, 1=grandparent, etc.)
            
        Returns:
            True if elements share ancestor at specified level
        """
        # Get ancestor chain for each element
        ancestor1 = elem1.get('parent_id')
        ancestor2 = elem2.get('parent_id')
        
        # Walk up the hierarchy to the specified level
        for _ in range(level):
            if not ancestor1 or not ancestor2:
                return False
            # TODO: Need to look up parent's parent from storage
            # For now, just check immediate parent at level 0
            if level == 0:
                break
        
        return ancestor1 == ancestor2
    
    def _get_phrase_embedding(self, phrase: str) -> Optional[np.ndarray]:
        """Get or compute embedding for a semantic phrase."""
        if phrase in self._phrase_embeddings:
            return self._phrase_embeddings[phrase]
        
        if self.embedding_provider:
            try:
                embedding = self.embedding_provider.generate(phrase)
                self._phrase_embeddings[phrase] = embedding
                return embedding
            except Exception as e:
                logger.error(f"Failed to generate embedding for phrase: {e}")
                return None
        
        return None
    
    @staticmethod
    def _cosine_similarity(vec1: np.ndarray, vec2: np.ndarray) -> float:
        """Calculate cosine similarity between two vectors."""
        vec1 = np.array(vec1).flatten()
        vec2 = np.array(vec2).flatten()
        
        dot_product = np.dot(vec1, vec2)
        norm1 = np.linalg.norm(vec1)
        norm2 = np.linalg.norm(vec2)
        
        if norm1 == 0 or norm2 == 0:
            return 0.0
        
        return float(dot_product / (norm1 * norm2))