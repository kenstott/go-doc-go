"""
Relationship Detector Module for the document pointer system.

This module detects relationships between document elements,
including explicit links, semantic relationships, structural connections,
and domain-specific relationships based on ontologies.
"""

import logging
from typing import Dict, Any, Optional

from .base import RelationshipDetector
from .composite import CompositeRelationshipDetector
from .explicit import ExplicitLinkDetector
from .semantic import SemanticRelationshipDetector
from .structural import StructuralRelationshipDetector
from .domain import DomainRelationshipDetector

logger = logging.getLogger(__name__)


def create_relationship_detector(config: Dict[str, Any], 
                                embedding_generator=None, 
                                db=None,
                                ontology_manager=None,
                                extractor_registry=None) -> RelationshipDetector:
    """
    Factory function to create a relationship detector from configuration.

    Args:
        config: Configuration dictionary
        embedding_generator: Optional embedding generator for semantic relationships
        db: Optional database instance for domain detector
        ontology_manager: Optional ontology manager for domain detector
        extractor_registry: Optional registry of entity extractors

    Returns:
        RelationshipDetector instance
    """
    detectors = []
    
    # Add explicit link detector (always enabled to handle parser-extracted links)
    detectors.append(ExplicitLinkDetector(config))

    # Add structural relationship detector
    if config.get("structural", True):
        detectors.append(StructuralRelationshipDetector(config))

    # Add semantic relationship detector if embeddings are enabled
    if config.get("semantic", False) and embedding_generator:
        semantic_config = config.get("semantic_config", {})
        detectors.append(SemanticRelationshipDetector(embedding_generator, semantic_config))
    
    # Add domain relationship detector if ontology manager is available
    if config.get("domain", False) and db and ontology_manager:
        domain_config = config.get("domain_config", {})
        detectors.append(DomainRelationshipDetector(
            db=db,
            ontology_manager=ontology_manager,
            embedding_generator=embedding_generator,
            config=domain_config,
            extractor_registry=extractor_registry
        ))
        logger.info("Domain relationship detector enabled")

    # Return composite detector
    return CompositeRelationshipDetector(detectors)
