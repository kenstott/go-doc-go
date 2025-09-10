"""
Domain ontology package for document element mapping and relationship discovery.
"""

from .ontology import (
    DomainOntology,
    DomainSettings,
    Term,
    MappingRule,
    ElementMapping,
    RelationshipRule,
    RelationshipEndpoint,
    RelationshipConfidence,
    RelationshipConstraints,
    RuleType,
    ConfidenceCalculation,
    RelationshipDirection
)

from .loader import (
    OntologyLoader,
    OntologyManager
)

from .evaluator import (
    OntologyEvaluator,
    ElementTermMapping,
    DomainRelationship
)

__all__ = [
    # Ontology classes
    'DomainOntology',
    'DomainSettings',
    'Term',
    'MappingRule',
    'ElementMapping',
    'RelationshipRule',
    'RelationshipEndpoint',
    'RelationshipConfidence',
    'RelationshipConstraints',
    
    # Enums
    'RuleType',
    'ConfidenceCalculation',
    'RelationshipDirection',
    
    # Loader classes
    'OntologyLoader',
    'OntologyManager',
    
    # Evaluator classes
    'OntologyEvaluator',
    'ElementTermMapping',
    'DomainRelationship',
]