"""
Builder for creating and managing domain ontologies.
"""

import json
import yaml
from typing import Dict, Any, List, Optional
from pathlib import Path


class OntologyBuilder:
    """Builder for creating domain ontologies."""
    
    def __init__(self, template: Optional[Dict[str, Any]] = None):
        """
        Initialize the ontology builder.
        
        Args:
            template: Base template to extend from
        """
        self.template = template or {}
    
    def build_from_config(self, config: Dict[str, Any]) -> Dict[str, Any]:
        """
        Build ontology from a configuration dictionary.
        
        Args:
            config: Configuration dictionary
            
        Returns:
            Complete ontology dictionary
        """
        # Start with template if provided
        ontology = self.template.copy()
        
        # Merge with provided config
        ontology.update(config)
        
        # Ensure required fields
        self._ensure_required_fields(ontology)
        
        # Validate structure
        self._validate_ontology(ontology)
        
        return ontology
    
    def to_yaml(self, ontology: Dict[str, Any]) -> str:
        """
        Convert ontology to YAML format with anchors.
        
        Args:
            ontology: Ontology dictionary
            
        Returns:
            YAML string
        """
        # Process for YAML anchors
        processed = self._add_yaml_anchors(ontology)
        
        # Convert to YAML with proper formatting
        return yaml.dump(
            processed,
            default_flow_style=False,
            sort_keys=False,
            allow_unicode=True,
            width=120
        )
    
    def to_dict(self, ontology: Dict[str, Any]) -> Dict[str, Any]:
        """
        Convert ontology to plain dictionary (for JSON export).
        
        Args:
            ontology: Ontology dictionary
            
        Returns:
            Plain dictionary
        """
        return ontology
    
    def merge_ontologies(self, *ontologies: Dict[str, Any]) -> Dict[str, Any]:
        """
        Merge multiple ontologies into one.
        
        Args:
            *ontologies: Ontology dictionaries to merge
            
        Returns:
            Merged ontology
        """
        result = {}
        
        for ontology in ontologies:
            # Merge metadata
            if "metadata" in ontology:
                if "metadata" not in result:
                    result["metadata"] = {}
                result["metadata"].update(ontology["metadata"])
            
            # Merge terms
            if "terms" in ontology:
                if "terms" not in result:
                    result["terms"] = []
                result["terms"].extend(ontology["terms"])
            
            # Merge entity mappings
            if "element_entity_mappings" in ontology:
                if "element_entity_mappings" not in result:
                    result["element_entity_mappings"] = []
                result["element_entity_mappings"].extend(ontology["element_entity_mappings"])
            
            # Merge relationships
            if "entity_relationship_rules" in ontology:
                if "entity_relationship_rules" not in result:
                    result["entity_relationship_rules"] = []
                result["entity_relationship_rules"].extend(ontology["entity_relationship_rules"])
            
            # Merge derived entities
            if "derived_entities" in ontology:
                if "derived_entities" not in result:
                    result["derived_entities"] = []
                result["derived_entities"].extend(ontology["derived_entities"])
        
        # Deduplicate
        self._deduplicate_ontology(result)
        
        return result
    
    def validate(self, ontology: Dict[str, Any]) -> List[str]:
        """
        Validate an ontology and return any issues.
        
        Args:
            ontology: Ontology to validate
            
        Returns:
            List of validation issues (empty if valid)
        """
        issues = []
        
        # Check required fields
        if "name" not in ontology:
            issues.append("Missing required field: name")
        
        if "version" not in ontology:
            issues.append("Missing required field: version")
        
        # Validate terms
        if "terms" in ontology:
            for i, term in enumerate(ontology["terms"]):
                if "term" not in term:
                    issues.append(f"Term {i} missing 'term' field")
        
        # Validate entity mappings
        if "element_entity_mappings" in ontology:
            for i, mapping in enumerate(ontology["element_entity_mappings"]):
                if "entity_type" not in mapping:
                    issues.append(f"Entity mapping {i} missing 'entity_type' field")
                
                if "extraction_rules" in mapping:
                    for j, rule in enumerate(mapping["extraction_rules"]):
                        if "type" not in rule:
                            issues.append(f"Extraction rule {i}.{j} missing 'type' field")
        
        # Validate relationships
        if "entity_relationship_rules" in ontology:
            entity_types = set()
            if "element_entity_mappings" in ontology:
                entity_types = {m["entity_type"] for m in ontology["element_entity_mappings"]}
            
            for i, rel in enumerate(ontology["entity_relationship_rules"]):
                if "source_entity_type" not in rel:
                    issues.append(f"Relationship {i} missing 'source_entity_type' field")
                elif entity_types and rel["source_entity_type"] not in entity_types:
                    issues.append(f"Relationship {i} references unknown source entity: {rel['source_entity_type']}")
                
                if "target_entity_type" not in rel:
                    issues.append(f"Relationship {i} missing 'target_entity_type' field")
                elif entity_types and rel["target_entity_type"] not in entity_types:
                    issues.append(f"Relationship {i} references unknown target entity: {rel['target_entity_type']}")
                
                if "relationship_type" not in rel:
                    issues.append(f"Relationship {i} missing 'relationship_type' field")
        
        return issues
    
    def _ensure_required_fields(self, ontology: Dict[str, Any]):
        """Ensure ontology has required fields."""
        if "name" not in ontology:
            ontology["name"] = "unnamed_ontology"
        
        if "version" not in ontology:
            ontology["version"] = "1.0.0"
        
        if "description" not in ontology:
            ontology["description"] = "Generated ontology"
    
    def _validate_ontology(self, ontology: Dict[str, Any]):
        """Validate ontology structure."""
        issues = self.validate(ontology)
        if issues:
            raise ValueError(f"Ontology validation failed: {'; '.join(issues)}")
    
    def _add_yaml_anchors(self, ontology: Dict[str, Any]) -> Dict[str, Any]:
        """
        Add YAML anchors for repeated structures.
        
        Args:
            ontology: Original ontology
            
        Returns:
            Ontology with YAML anchors
        """
        result = ontology.copy()
        
        # Create anchors for common patterns
        anchors = {}
        
        # Find repeated confidence values
        confidence_values = set()
        if "element_entity_mappings" in result:
            for mapping in result["element_entity_mappings"]:
                if "extraction_rules" in mapping:
                    for rule in mapping["extraction_rules"]:
                        if "confidence" in rule:
                            confidence_values.add(rule["confidence"])
        
        # Create anchors for common confidence values
        for conf in confidence_values:
            anchor_name = f"confidence_{str(conf).replace('.', '_')}"
            anchors[anchor_name] = conf
        
        # Find repeated element types
        element_type_sets = []
        if "element_entity_mappings" in result:
            for mapping in result["element_entity_mappings"]:
                if "element_types" in mapping:
                    element_types = tuple(sorted(mapping["element_types"]))
                    if element_types not in element_type_sets:
                        element_type_sets.append(element_types)
        
        # Apply anchors (simplified for now - full YAML anchor support would require custom YAML representer)
        # For now, just return the processed ontology
        return result
    
    def _deduplicate_ontology(self, ontology: Dict[str, Any]):
        """Remove duplicate entries from ontology."""
        # Deduplicate terms
        if "terms" in ontology:
            seen = set()
            unique_terms = []
            for term in ontology["terms"]:
                term_key = term.get("term", "")
                if term_key not in seen:
                    seen.add(term_key)
                    unique_terms.append(term)
            ontology["terms"] = unique_terms
        
        # Deduplicate entity mappings
        if "element_entity_mappings" in ontology:
            seen = set()
            unique_mappings = []
            for mapping in ontology["element_entity_mappings"]:
                mapping_key = mapping.get("entity_type", "")
                if mapping_key not in seen:
                    seen.add(mapping_key)
                    unique_mappings.append(mapping)
            ontology["element_entity_mappings"] = unique_mappings
        
        # Deduplicate relationships
        if "entity_relationship_rules" in ontology:
            seen = set()
            unique_rels = []
            for rel in ontology["entity_relationship_rules"]:
                rel_key = (
                    rel.get("source_entity_type", ""),
                    rel.get("relationship_type", ""),
                    rel.get("target_entity_type", "")
                )
                if rel_key not in seen:
                    seen.add(rel_key)
                    unique_rels.append(rel)
            ontology["entity_relationship_rules"] = unique_rels