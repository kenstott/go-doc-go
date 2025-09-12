"""
Entity extractors and normalizers for domain-specific processing.
"""

from .base import EntityExtractor, ExtractorRegistry
from .normalizers import *
from .financial import *
from .temporal import *
from .legal import *
from .medical import *

__all__ = [
    'EntityExtractor',
    'ExtractorRegistry',
    'get_extractor',
    'register_extractor'
]

# Global registry instance
_registry = ExtractorRegistry()

def get_extractor(name: str) -> EntityExtractor:
    """Get a registered extractor by name."""
    return _registry.get(name)

def register_extractor(name: str, extractor: EntityExtractor):
    """Register a new extractor."""
    _registry.register(name, extractor)