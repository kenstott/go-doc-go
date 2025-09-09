"""
XML-style semantic tagging for contextual embeddings.
Generates XML tags that align with LLM training patterns.
"""

import html
import logging
import re
from typing import Dict, Any, Optional, List, Set
from enum import Enum

logger = logging.getLogger(__name__)


class ContextRole(Enum):
    """Semantic roles for context elements."""
    MAIN = "main"  # The primary element being embedded
    PARENT = "parent"
    CHILD = "child"
    SIBLING = "sibling"
    PRECEDING = "preceding"
    FOLLOWING = "following"
    REFERENCES = "references"
    REFERENCED_BY = "referenced_by"
    CONTAINS = "contains"
    CONTAINED_BY = "contained_by"
    RELATED = "related"


class XMLSemanticTagger:
    """
    Generate XML-style semantic tags that align with LLM training patterns.
    
    Creates tags like:
    <element role="main" type="paragraph" id="p123" entities="revenue,growth">content</element>
    <context role="parent" type="section" strength="0.9">parent content</context>
    """
    
    def __init__(self, 
                 include_metadata: bool = True,
                 include_entities: bool = True,
                 include_strength: bool = True,
                 max_entities: int = 5):
        """
        Initialize XML semantic tagger.
        
        Args:
            include_metadata: Whether to include metadata in tags
            include_entities: Whether to extract and include entities
            include_strength: Whether to include relationship strength
            max_entities: Maximum number of entities to extract per element
        """
        self.include_metadata = include_metadata
        self.include_entities = include_entities
        self.include_strength = include_strength
        self.max_entities = max_entities
        
        # Map relationship types to context roles
        self.relationship_mapping = {
            "contains": ContextRole.PARENT,
            "contained_by": ContextRole.CHILD,
            "references": ContextRole.REFERENCES,
            "referenced_by": ContextRole.REFERENCED_BY,
            "follows": ContextRole.PRECEDING,
            "precedes": ContextRole.FOLLOWING,
            "related_to": ContextRole.RELATED,
            "parent": ContextRole.PARENT,
            "child": ContextRole.CHILD,
            "sibling": ContextRole.SIBLING
        }
        
        # Entity extraction patterns (simple keyword-based approach)
        self.entity_patterns = [
            # Financial terms
            r'\b(?:revenue|profit|loss|income|expenses|assets|liabilities|equity|cash|debt)\b',
            # Technical terms
            r'\b(?:database|server|connection|timeout|query|performance|api|endpoint)\b',
            # Business terms
            r'\b(?:customer|client|user|product|service|market|sales|growth|strategy)\b',
            # Numbers and measurements
            r'\b\d+(?:\.\d+)?(?:%|million|billion|thousand|k|m|b)?\b',
            # Dates and times
            r'\b(?:Q[1-4]|january|february|march|april|may|june|july|august|september|october|november|december|\d{4}|\d{1,2}\/\d{1,2}\/\d{4})\b',
        ]
        
        # Compile patterns for efficiency
        self.compiled_patterns = [re.compile(pattern, re.IGNORECASE) for pattern in self.entity_patterns]
    
    def extract_entities(self, text: str) -> List[str]:
        """
        Extract key entities from text content.
        
        Args:
            text: Text content to extract entities from
            
        Returns:
            List of extracted entity strings
        """
        if not self.include_entities or not text:
            return []
        
        entities = set()
        
        # Apply each pattern to find entities
        for pattern in self.compiled_patterns:
            matches = pattern.findall(text.lower())
            entities.update(matches)
        
        # Remove very short entities and common words
        stop_words = {'the', 'and', 'or', 'but', 'in', 'on', 'at', 'to', 'for', 'of', 'with', 'by', 'a', 'an'}
        filtered_entities = [
            entity for entity in entities 
            if len(entity) > 2 and entity not in stop_words
        ]
        
        # Sort by length (longer entities first) and limit count
        sorted_entities = sorted(filtered_entities, key=len, reverse=True)
        return sorted_entities[:self.max_entities]
    
    def calculate_relationship_strength(self, 
                                      context_role: ContextRole,
                                      element_distance: int = 1,
                                      element_type: str = "") -> float:
        """
        Calculate relationship strength based on role, distance, and type.
        
        Args:
            context_role: Role of the context element
            element_distance: Distance from main element (1 = direct parent/child)
            element_type: Type of element (used for type-specific weighting)
            
        Returns:
            Strength value between 0.0 and 1.0
        """
        if not self.include_strength:
            return 1.0
        
        # Base strength by role
        role_strengths = {
            ContextRole.MAIN: 1.0,
            ContextRole.PARENT: 0.9,
            ContextRole.CHILD: 0.7,
            ContextRole.SIBLING: 0.6,
            ContextRole.PRECEDING: 0.5,
            ContextRole.FOLLOWING: 0.5,
            ContextRole.REFERENCES: 0.8,
            ContextRole.REFERENCED_BY: 0.8,
            ContextRole.CONTAINS: 0.9,
            ContextRole.CONTAINED_BY: 0.7,
            ContextRole.RELATED: 0.4
        }
        
        base_strength = role_strengths.get(context_role, 0.5)
        
        # Reduce strength based on distance
        distance_penalty = max(0.1, 1.0 - (element_distance - 1) * 0.2)
        
        # Type-specific bonuses
        important_types = {"header", "title", "h1", "h2", "h3", "table_header", "section"}
        type_bonus = 1.1 if element_type in important_types else 1.0
        
        final_strength = min(1.0, base_strength * distance_penalty * type_bonus)
        return round(final_strength, 2)
    
    def generate_xml_tag(self,
                        element: Dict[str, Any],
                        content: str,
                        relationship_type: Optional[str] = None,
                        context_role: Optional[ContextRole] = None,
                        element_distance: int = 1) -> str:
        """
        Generate XML-style semantic tag for a context element.
        
        Args:
            element: Element dictionary with type, id, and metadata
            content: Text content of the element
            relationship_type: Type of relationship (if known)
            context_role: Explicit context role (overrides relationship_type)
            element_distance: Distance from main element
            
        Returns:
            XML-formatted string like:
            <element role="main" type="paragraph" entities="revenue,growth">content</element>
        """
        # Determine context role
        if context_role:
            role = context_role
        elif relationship_type:
            # Handle case where relationship_type might be a ContextRole (from test calls)
            if isinstance(relationship_type, ContextRole):
                role = relationship_type
            else:
                role = self.relationship_mapping.get(
                    relationship_type.lower(),
                    ContextRole.RELATED
                )
        else:
            role = ContextRole.RELATED
        
        # Extract element information
        element_type = element.get("element_type", "unknown")
        element_id = element.get("element_id", "")
        
        # Determine tag name based on role
        tag_name = "element" if role == ContextRole.MAIN else "context"
        
        # Build attributes
        attributes = [f'role="{role.value}"']
        attributes.append(f'type="{element_type}"')
        
        if element_id:
            # Truncate ID if too long and escape for XML
            max_id_length = 30
            if len(element_id) > max_id_length:
                display_id = element_id[:max_id_length-3] + "..."
            else:
                display_id = element_id
            escaped_id = html.escape(display_id)
            attributes.append(f'id="{escaped_id}"')
        
        # Add relationship strength
        if self.include_strength and role != ContextRole.MAIN:
            strength = self.calculate_relationship_strength(role, element_distance, element_type)
            attributes.append(f'strength="{strength}"')
        
        # Add entities
        if self.include_entities:
            entities = self.extract_entities(content)
            if entities:
                entities_str = ",".join(entities[:self.max_entities])
                escaped_entities = html.escape(entities_str)
                attributes.append(f'entities="{escaped_entities}"')
        
        # Add key metadata
        if self.include_metadata and element.get("metadata"):
            metadata = element["metadata"]
            
            # Select key metadata fields
            if "level" in metadata:  # Header level
                attributes.append(f'level="{metadata["level"]}"')
            if "page_number" in metadata:  # Page reference
                attributes.append(f'page="{metadata["page_number"]}"')
            if "position" in metadata:  # Document position
                attributes.append(f'position="{metadata["position"]}"')
        
        # Escape content for XML
        escaped_content = html.escape(content.strip()) if content else ""
        
        # Build final XML tag
        attributes_str = " ".join(attributes)
        xml_tag = f"<{tag_name} {attributes_str}>{escaped_content}</{tag_name}>"
        
        return xml_tag
    
    def generate_document_wrapper(self,
                                 doc_type: str = "unknown",
                                 domain: str = "",
                                 doc_id: str = "") -> tuple[str, str]:
        """
        Generate opening and closing document wrapper tags.
        
        Args:
            doc_type: Type of document (pdf, docx, json, etc.)
            domain: Domain or category of document
            doc_id: Document identifier
            
        Returns:
            Tuple of (opening_tag, closing_tag)
        """
        attributes = [f'type="{doc_type}"']
        
        if domain:
            escaped_domain = html.escape(domain)
            attributes.append(f'domain="{escaped_domain}"')
        
        if doc_id:
            escaped_id = html.escape(doc_id)
            attributes.append(f'id="{escaped_id}"')
        
        attributes_str = " ".join(attributes)
        opening_tag = f"<document {attributes_str}>"
        closing_tag = "</document>"
        
        return opening_tag, closing_tag
    
    def validate_xml_structure(self, xml_content: str) -> bool:
        """
        Validate that XML content has proper structure.
        
        Args:
            xml_content: XML content to validate
            
        Returns:
            True if valid XML structure, False otherwise
        """
        try:
            # Simple validation - check that tags are balanced
            import xml.etree.ElementTree as ET
            ET.fromstring(xml_content)
            return True
        except ET.ParseError:
            return False
    
    def parse_xml_tag(self, xml_string: str) -> Optional[Dict[str, Any]]:
        """
        Parse an XML tag back to its components.
        
        Args:
            xml_string: XML string like "<element role='main' type='paragraph'>content</element>"
            
        Returns:
            Dictionary with role, element_type, content, and attributes
        """
        try:
            import xml.etree.ElementTree as ET
            root = ET.fromstring(xml_string)
            
            result = {
                "tag_name": root.tag,
                "content": root.text or "",
                "attributes": root.attrib
            }
            
            # Extract common attributes
            if "role" in root.attrib:
                result["role"] = root.attrib["role"]
            if "type" in root.attrib:
                result["element_type"] = root.attrib["type"]
            if "entities" in root.attrib:
                result["entities"] = root.attrib["entities"].split(",")
            if "strength" in root.attrib:
                try:
                    result["strength"] = float(root.attrib["strength"])
                except ValueError:
                    pass
            
            return result
            
        except Exception as e:
            logger.warning(f"Error parsing XML tag: {e}")
            return None
    
    def extract_entities_from_xml(self, xml_content: str) -> List[str]:
        """
        Extract all entities from XML content.
        
        Args:
            xml_content: XML content containing entity attributes
            
        Returns:
            List of all unique entities found
        """
        entities = set()
        
        # Use regex to find entities attributes
        entity_pattern = r'entities="([^"]+)"'
        matches = re.findall(entity_pattern, xml_content)
        
        for match in matches:
            entity_list = match.split(",")
            entities.update(entity.strip() for entity in entity_list)
        
        return list(entities)