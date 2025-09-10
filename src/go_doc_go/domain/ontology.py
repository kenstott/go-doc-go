"""
Domain ontology classes for managing entity and relationship definitions.
"""
import re
from dataclasses import dataclass, field
from enum import Enum
from typing import Dict, List, Optional, Any, Union
import numpy as np


class RuleType(Enum):
    """Types of matching rules."""
    SEMANTIC = "semantic"
    REGEX = "regex"
    KEYWORDS = "keywords"


class ConfidenceCalculation(Enum):
    """Methods for calculating combined confidence."""
    AVERAGE = "average"
    MIN = "min"
    MAX = "max"
    WEIGHTED = "weighted"


class RelationshipDirection(Enum):
    """Direction constraints for relationships."""
    FORWARD = "forward"   # Source must come before target
    BACKWARD = "backward"  # Target must come before source
    ANY = "any"           # No direction constraint


@dataclass
class DomainSettings:
    """Global settings for a domain ontology."""
    default_confidence_threshold: float = 0.70
    max_relationships_per_pair: int = 3
    enable_transitive_inference: bool = False
    
    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> 'DomainSettings':
        """Create from dictionary."""
        return cls(
            default_confidence_threshold=data.get('default_confidence_threshold', 0.70),
            max_relationships_per_pair=data.get('max_relationships_per_pair', 3),
            enable_transitive_inference=data.get('enable_transitive_inference', False)
        )


@dataclass
class Term:
    """A domain term/entity definition."""
    id: str
    label: str
    description: str
    aliases: Optional[List[str]] = None
    
    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> 'Term':
        """Create from dictionary."""
        return cls(
            id=data['id'],
            label=data['label'],
            description=data['description'],
            aliases=data.get('aliases')
        )
    
    def get_all_names(self) -> List[str]:
        """Get all names including aliases."""
        names = [self.id, self.label]
        if self.aliases:
            names.extend(self.aliases)
        return names


@dataclass
class MappingRule:
    """A rule for mapping elements to terms."""
    type: RuleType
    element_types: Optional[List[str]] = None  # None means match all
    
    # For semantic rules
    semantic_phrase: Optional[str] = None
    confidence_threshold: Optional[float] = None
    
    # For regex rules
    pattern: Optional[str] = None
    case_sensitive: bool = False
    
    # For keyword rules
    keywords: Optional[List[str]] = None
    word_boundary: bool = True
    
    # Cached compiled regex
    _compiled_pattern: Optional[re.Pattern] = field(default=None, init=False, repr=False)
    
    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> 'MappingRule':
        """Create from dictionary."""
        rule_type = RuleType(data['type'])
        return cls(
            type=rule_type,
            element_types=data.get('element_types'),
            semantic_phrase=data.get('semantic_phrase'),
            confidence_threshold=data.get('confidence_threshold'),
            pattern=data.get('pattern'),
            case_sensitive=data.get('case_sensitive', False),
            keywords=data.get('keywords'),
            word_boundary=data.get('word_boundary', True)
        )
    
    def get_pattern(self) -> Optional[re.Pattern]:
        """Get compiled regex pattern (cached)."""
        if self.type == RuleType.SEMANTIC:
            return None
            
        if self._compiled_pattern is None:
            if self.type == RuleType.REGEX:
                flags = 0 if self.case_sensitive else re.IGNORECASE
                self._compiled_pattern = re.compile(self.pattern, flags)
                
            elif self.type == RuleType.KEYWORDS:
                pattern_str = self._keywords_to_pattern()
                flags = 0 if self.case_sensitive else re.IGNORECASE
                self._compiled_pattern = re.compile(pattern_str, flags)
                
        return self._compiled_pattern
    
    def _keywords_to_pattern(self) -> str:
        """Convert keywords to regex pattern."""
        if not self.keywords:
            return ""
        escaped = [re.escape(k) for k in self.keywords]
        joined = '|'.join(escaped)
        
        if self.word_boundary:
            return f"\\b({joined})\\b"
        else:
            return f"({joined})"
    
    def matches_element_type(self, element_type: str) -> bool:
        """Check if element type matches this rule's filter."""
        if self.element_types is None or len(self.element_types) == 0:
            return True
            
        if "*" in self.element_types:
            return True
            
        for filter_pattern in self.element_types:
            # Check if it's a regex pattern
            if any(char in filter_pattern for char in r'.*+?[]{}()^$|\\'):
                try:
                    if re.match(filter_pattern, element_type):
                        return True
                except re.error:
                    # Invalid regex, treat as literal
                    if filter_pattern == element_type:
                        return True
            else:
                # Exact match
                if filter_pattern == element_type:
                    return True
                    
        return False


@dataclass
class ElementMapping:
    """Mapping rules from elements to a term."""
    term_id: str
    rules: List[MappingRule]
    
    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> 'ElementMapping':
        """Create from dictionary."""
        return cls(
            term_id=data['term_id'],
            rules=[MappingRule.from_dict(r) for r in data.get('rules', [])]
        )


@dataclass
class RelationshipConstraints:
    """Constraints for relationship discovery using document hierarchy."""
    hierarchy_level: Optional[int] = None  # None=no constraint (cross-doc OK), -1=same document, 0=same parent, 1=same grandparent, etc.
    direction: RelationshipDirection = RelationshipDirection.ANY
    
    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> 'RelationshipConstraints':
        """Create from dictionary."""
        direction = data.get('direction', 'any')
        
        # Support both old and new formats
        hierarchy_level = data.get('hierarchy_level')
        if hierarchy_level is None:
            # Check for legacy fields
            if data.get('same_section', False):
                hierarchy_level = 0  # Same section maps to same parent
            elif data.get('max_distance') is not None:
                # If max_distance is set, default to same document
                hierarchy_level = -1
        
        return cls(
            hierarchy_level=hierarchy_level,
            direction=RelationshipDirection(direction)
        )


@dataclass
class RelationshipEndpoint:
    """Configuration for one endpoint of a relationship."""
    term_id: str
    semantic_phrase: str
    confidence_threshold: float
    element_types: Optional[List[str]] = None
    
    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> 'RelationshipEndpoint':
        """Create from dictionary."""
        return cls(
            term_id=data['term_id'],
            semantic_phrase=data['semantic_phrase'],
            confidence_threshold=data['confidence_threshold'],
            element_types=data.get('element_types')
        )


@dataclass
class RelationshipConfidence:
    """Configuration for relationship confidence calculation."""
    minimum: float = 0.70
    calculation: ConfidenceCalculation = ConfidenceCalculation.AVERAGE
    weights: Optional[Dict[str, float]] = None  # For weighted calculation
    
    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> 'RelationshipConfidence':
        """Create from dictionary."""
        return cls(
            minimum=data.get('minimum', 0.70),
            calculation=ConfidenceCalculation(data.get('calculation', 'average')),
            weights=data.get('weights')
        )
    
    def calculate(self, source_score: float, target_score: float) -> float:
        """Calculate combined confidence."""
        if self.calculation == ConfidenceCalculation.AVERAGE:
            return (source_score + target_score) / 2
        elif self.calculation == ConfidenceCalculation.MIN:
            return min(source_score, target_score)
        elif self.calculation == ConfidenceCalculation.MAX:
            return max(source_score, target_score)
        elif self.calculation == ConfidenceCalculation.WEIGHTED:
            if not self.weights:
                # Default to equal weights
                return (source_score + target_score) / 2
            source_weight = self.weights.get('source', 0.5)
            target_weight = self.weights.get('target', 0.5)
            total_weight = source_weight + target_weight
            return (source_score * source_weight + target_score * target_weight) / total_weight
        else:
            return (source_score + target_score) / 2


@dataclass
class RelationshipRule:
    """Rule for discovering relationships between elements."""
    id: str
    relationship_type: str
    description: str
    source: RelationshipEndpoint
    target: RelationshipEndpoint
    confidence: RelationshipConfidence
    constraints: Optional[RelationshipConstraints] = None
    bidirectional: bool = False
    
    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> 'RelationshipRule':
        """Create from dictionary."""
        return cls(
            id=data['id'],
            relationship_type=data['relationship_type'],
            description=data.get('description', ''),
            source=RelationshipEndpoint.from_dict(data['source']),
            target=RelationshipEndpoint.from_dict(data['target']),
            confidence=RelationshipConfidence.from_dict(data.get('confidence', {})),
            constraints=RelationshipConstraints.from_dict(data['constraints']) 
                       if 'constraints' in data else None,
            bidirectional=data.get('bidirectional', False)
        )


@dataclass
class DomainOntology:
    """Complete domain ontology definition."""
    name: str
    version: str
    description: str
    settings: DomainSettings
    terms: List[Term]
    element_mappings: List[ElementMapping]
    relationship_rules: List[RelationshipRule]
    
    # Lookup dictionaries for performance
    _terms_by_id: Dict[str, Term] = field(default_factory=dict, init=False, repr=False)
    _mappings_by_term: Dict[str, List[MappingRule]] = field(default_factory=dict, init=False, repr=False)
    _rules_by_relationship: Dict[str, List[RelationshipRule]] = field(default_factory=dict, init=False, repr=False)
    
    def __post_init__(self):
        """Build lookup dictionaries after initialization."""
        self._build_lookups()
    
    def _build_lookups(self):
        """Build internal lookup dictionaries."""
        # Terms by ID
        self._terms_by_id = {term.id: term for term in self.terms}
        
        # Mapping rules by term ID
        self._mappings_by_term = {}
        for mapping in self.element_mappings:
            if mapping.term_id not in self._mappings_by_term:
                self._mappings_by_term[mapping.term_id] = []
            self._mappings_by_term[mapping.term_id].extend(mapping.rules)
        
        # Relationship rules by type
        self._rules_by_relationship = {}
        for rule in self.relationship_rules:
            if rule.relationship_type not in self._rules_by_relationship:
                self._rules_by_relationship[rule.relationship_type] = []
            self._rules_by_relationship[rule.relationship_type].append(rule)
    
    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> 'DomainOntology':
        """Create from dictionary (parsed from YAML/JSON)."""
        domain_info = data.get('domain', {})
        return cls(
            name=domain_info.get('name', 'unnamed'),
            version=domain_info.get('version', '1.0.0'),
            description=domain_info.get('description', ''),
            settings=DomainSettings.from_dict(domain_info.get('settings', {})),
            terms=[Term.from_dict(t) for t in data.get('terms', [])],
            element_mappings=[ElementMapping.from_dict(m) for m in data.get('element_mappings', [])],
            relationship_rules=[RelationshipRule.from_dict(r) for r in data.get('relationship_rules', [])]
        )
    
    def get_term(self, term_id: str) -> Optional[Term]:
        """Get term by ID."""
        return self._terms_by_id.get(term_id)
    
    def get_mapping_rules(self, term_id: str) -> List[MappingRule]:
        """Get all mapping rules for a term."""
        return self._mappings_by_term.get(term_id, [])
    
    def get_relationship_rules(self, relationship_type: Optional[str] = None) -> List[RelationshipRule]:
        """Get relationship rules, optionally filtered by type."""
        if relationship_type:
            return self._rules_by_relationship.get(relationship_type, [])
        return self.relationship_rules
    
    def validate(self) -> List[str]:
        """Validate the ontology configuration."""
        issues = []
        
        # Check all term IDs are unique
        term_ids = [t.id for t in self.terms]
        if len(term_ids) != len(set(term_ids)):
            issues.append("Duplicate term IDs found")
        
        # Check all term references exist
        for mapping in self.element_mappings:
            if mapping.term_id not in self._terms_by_id:
                issues.append(f"Element mapping references unknown term: {mapping.term_id}")
        
        for rule in self.relationship_rules:
            if rule.source.term_id not in self._terms_by_id:
                issues.append(f"Relationship rule references unknown source term: {rule.source.term_id}")
            if rule.target.term_id not in self._terms_by_id:
                issues.append(f"Relationship rule references unknown target term: {rule.target.term_id}")
        
        # Validate regex patterns
        for mapping in self.element_mappings:
            for rule in mapping.rules:
                if rule.type == RuleType.REGEX and rule.pattern:
                    try:
                        re.compile(rule.pattern)
                    except re.error as e:
                        issues.append(f"Invalid regex pattern in mapping: {rule.pattern} - {e}")
        
        # Check for required fields in rules
        for mapping in self.element_mappings:
            for rule in mapping.rules:
                if rule.type == RuleType.SEMANTIC:
                    if not rule.semantic_phrase:
                        issues.append(f"Semantic rule missing phrase for term {mapping.term_id}")
                    if rule.confidence_threshold is None:
                        issues.append(f"Semantic rule missing threshold for term {mapping.term_id}")
                elif rule.type == RuleType.REGEX:
                    if not rule.pattern:
                        issues.append(f"Regex rule missing pattern for term {mapping.term_id}")
                elif rule.type == RuleType.KEYWORDS:
                    if not rule.keywords:
                        issues.append(f"Keywords rule missing keywords for term {mapping.term_id}")
        
        return issues