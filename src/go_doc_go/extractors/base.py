"""
Base classes for extensible entity extraction and normalization.
"""

import re
import logging
from abc import ABC, abstractmethod
from typing import Dict, Any, List, Optional, Union, Pattern, Callable
from dataclasses import dataclass, field

logger = logging.getLogger(__name__)


@dataclass
class ExtractedEntity:
    """Represents an extracted and normalized entity."""
    raw_value: str              # Original value from document
    normalized_value: str       # Normalized for comparison
    display_value: str         # For display to users
    entity_type: str           # Type of entity (monetary, date, company, etc.)
    metadata: Dict[str, Any] = field(default_factory=dict)  # Additional metadata
    confidence: float = 1.0    # Extraction confidence
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary for storage."""
        return {
            'raw_value': self.raw_value,
            'normalized_value': self.normalized_value,
            'display_value': self.display_value,
            'entity_type': self.entity_type,
            'metadata': self.metadata,
            'confidence': self.confidence
        }


class EntityExtractor(ABC):
    """
    Base class for entity extractors.
    
    Each extractor handles a specific type of entity extraction and normalization.
    """
    
    def __init__(self, config: Optional[Dict[str, Any]] = None):
        """
        Initialize extractor with optional configuration.
        
        Args:
            config: Configuration dictionary for the extractor
        """
        self.config = config or {}
        self.enabled = self.config.get('enabled', True)
        
    @property
    @abstractmethod
    def entity_type(self) -> str:
        """Return the type of entity this extractor handles."""
        pass
    
    @property
    @abstractmethod
    def description(self) -> str:
        """Return a description of what this extractor does."""
        pass
    
    @abstractmethod
    def extract(self, text: str) -> List[ExtractedEntity]:
        """
        Extract entities from text.
        
        Args:
            text: Input text to extract from
            
        Returns:
            List of extracted entities
        """
        pass
    
    @abstractmethod
    def normalize(self, value: str) -> str:
        """
        Normalize a value for comparison.
        
        Args:
            value: Value to normalize
            
        Returns:
            Normalized value
        """
        pass
    
    def is_match(self, value1: str, value2: str) -> bool:
        """
        Check if two values match after normalization.
        
        Args:
            value1: First value
            value2: Second value
            
        Returns:
            True if values match after normalization
        """
        return self.normalize(value1) == self.normalize(value2)


class RegexExtractor(EntityExtractor):
    """
    Generic regex-based extractor.
    
    Can be configured with patterns and normalization functions.
    """
    
    def __init__(self, 
                 entity_type: str,
                 patterns: List[Union[str, Pattern]],
                 normalizer: Optional[Callable[[str], str]] = None,
                 formatter: Optional[Callable[[str], str]] = None,
                 description: str = "",
                 config: Optional[Dict[str, Any]] = None):
        """
        Initialize regex extractor.
        
        Args:
            entity_type: Type of entity being extracted
            patterns: List of regex patterns to match
            normalizer: Function to normalize extracted values
            formatter: Function to format display values
            description: Description of the extractor
            config: Additional configuration
        """
        super().__init__(config)
        self._entity_type = entity_type
        self._description = description or f"Regex extractor for {entity_type}"
        self._normalizer = normalizer or str.lower
        self._formatter = formatter or str
        
        # Compile patterns
        self.patterns = []
        for pattern in patterns:
            if isinstance(pattern, str):
                self.patterns.append(re.compile(pattern, re.IGNORECASE))
            else:
                self.patterns.append(pattern)
    
    @property
    def entity_type(self) -> str:
        return self._entity_type
    
    @property
    def description(self) -> str:
        return self._description
    
    def extract(self, text: str) -> List[ExtractedEntity]:
        """Extract entities using regex patterns."""
        entities = []
        
        for pattern in self.patterns:
            for match in pattern.finditer(text):
                raw_value = match.group(0)
                
                # Extract groups if present
                groups = match.groups()
                metadata = {}
                if groups:
                    metadata['groups'] = groups
                
                entity = ExtractedEntity(
                    raw_value=raw_value,
                    normalized_value=self.normalize(raw_value),
                    display_value=self._formatter(raw_value),
                    entity_type=self.entity_type,
                    metadata=metadata
                )
                entities.append(entity)
        
        return entities
    
    def normalize(self, value: str) -> str:
        """Normalize value using configured normalizer."""
        return self._normalizer(value)


class ExtractorRegistry:
    """Registry for managing entity extractors."""
    
    def __init__(self):
        """Initialize empty registry."""
        self._extractors: Dict[str, EntityExtractor] = {}
        self._type_extractors: Dict[str, List[str]] = {}  # entity_type -> [extractor_names]
    
    def register(self, name: str, extractor: EntityExtractor):
        """
        Register an extractor.
        
        Args:
            name: Unique name for the extractor
            extractor: Extractor instance
        """
        if name in self._extractors:
            logger.warning(f"Overwriting existing extractor: {name}")
        
        self._extractors[name] = extractor
        
        # Track by entity type
        entity_type = extractor.entity_type
        if entity_type not in self._type_extractors:
            self._type_extractors[entity_type] = []
        if name not in self._type_extractors[entity_type]:
            self._type_extractors[entity_type].append(name)
        
        logger.info(f"Registered extractor '{name}' for type '{entity_type}'")
    
    def get(self, name: str) -> Optional[EntityExtractor]:
        """Get an extractor by name."""
        return self._extractors.get(name)
    
    def get_by_type(self, entity_type: str) -> List[EntityExtractor]:
        """Get all extractors for a specific entity type."""
        names = self._type_extractors.get(entity_type, [])
        return [self._extractors[name] for name in names if name in self._extractors]
    
    def get_all(self) -> Dict[str, EntityExtractor]:
        """Get all registered extractors."""
        return self._extractors.copy()
    
    def extract_all(self, text: str, 
                   entity_types: Optional[List[str]] = None) -> List[ExtractedEntity]:
        """
        Run all extractors on text.
        
        Args:
            text: Text to extract from
            entity_types: Optional list of entity types to extract (None = all)
            
        Returns:
            List of all extracted entities
        """
        entities = []
        
        if entity_types:
            # Extract specific types
            for entity_type in entity_types:
                for extractor in self.get_by_type(entity_type):
                    if extractor.enabled:
                        entities.extend(extractor.extract(text))
        else:
            # Extract all types
            for extractor in self._extractors.values():
                if extractor.enabled:
                    entities.extend(extractor.extract(text))
        
        return entities


class CompositeExtractor(EntityExtractor):
    """
    Combines multiple extractors for a single entity type.
    """
    
    def __init__(self, 
                 entity_type: str,
                 extractors: List[EntityExtractor],
                 description: str = "",
                 config: Optional[Dict[str, Any]] = None):
        """
        Initialize composite extractor.
        
        Args:
            entity_type: Type of entity
            extractors: List of sub-extractors
            description: Description
            config: Configuration
        """
        super().__init__(config)
        self._entity_type = entity_type
        self._description = description or f"Composite extractor for {entity_type}"
        self.extractors = extractors
    
    @property
    def entity_type(self) -> str:
        return self._entity_type
    
    @property
    def description(self) -> str:
        return self._description
    
    def extract(self, text: str) -> List[ExtractedEntity]:
        """Run all sub-extractors and combine results."""
        entities = []
        for extractor in self.extractors:
            if extractor.enabled:
                entities.extend(extractor.extract(text))
        return entities
    
    def normalize(self, value: str) -> str:
        """Use first extractor's normalizer."""
        if self.extractors:
            return self.extractors[0].normalize(value)
        return value.lower()