"""
Semantic tagging for contextual embeddings.
Generates relationship-aware tags for context elements.
"""

import logging
from typing import Dict, Any, Optional, Tuple, List
from enum import Enum

logger = logging.getLogger(__name__)


class ContextRole(Enum):
    """Semantic roles for context elements."""
    MAIN = "MAIN"  # The primary element being embedded
    PARENT = "PARENT"
    CHILD = "CHILD"
    SIBLING = "SIBLING"
    PRECEDING = "PRECEDING"
    FOLLOWING = "FOLLOWING"
    REFERENCES = "REFERENCES"
    REFERENCED_BY = "REFERENCED_BY"
    CONTAINS = "CONTAINS"
    CONTAINED_BY = "CONTAINED_BY"
    RELATED = "RELATED"


class SemanticTagger:
    """
    Generate and parse semantic tags for contextual embeddings.
    
    Tags encode relationship type, element type, and metadata
    to preserve semantic information in embeddings.
    """
    
    def __init__(self, include_metadata: bool = True, max_tag_length: int = 100):
        """
        Initialize semantic tagger.
        
        Args:
            include_metadata: Whether to include metadata in tags
            max_tag_length: Maximum length for generated tags
        """
        self.include_metadata = include_metadata
        self.max_tag_length = max_tag_length
        
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
    
    def generate_tag(self,
                    element: Dict[str, Any],
                    relationship_type: Optional[str] = None,
                    context_role: Optional[ContextRole] = None) -> str:
        """
        Generate semantic tag for a context element.
        
        Args:
            element: Element dictionary with type, id, and metadata
            relationship_type: Type of relationship (if known)
            context_role: Explicit context role (overrides relationship_type)
            
        Returns:
            Semantic tag string like "[PARENT:Section:accounting_policies:depth=2]"
        """
        # Determine context role
        if context_role:
            role = context_role
        elif relationship_type:
            role = self.relationship_mapping.get(
                relationship_type.lower(),
                ContextRole.RELATED
            )
        else:
            role = ContextRole.RELATED
        
        # Extract element information
        element_type = element.get("element_type", "unknown")
        element_id = element.get("element_id", "")
        
        # Build base tag
        if element_id:
            # Truncate ID if too long
            max_id_length = 30
            if len(element_id) > max_id_length:
                display_id = element_id[:max_id_length-3] + "..."
            else:
                display_id = element_id
            base_tag = f"[{role.value}:{element_type}:{display_id}"
        else:
            base_tag = f"[{role.value}:{element_type}"
        
        # Add metadata if enabled
        if self.include_metadata and element.get("metadata"):
            metadata_parts = []
            metadata = element["metadata"]
            
            # Select key metadata fields
            if "level" in metadata:  # Header level
                metadata_parts.append(f"level={metadata['level']}")
            if "row_count" in metadata:  # Table rows
                metadata_parts.append(f"rows={metadata['row_count']}")
            if "column_count" in metadata:  # Table columns
                metadata_parts.append(f"cols={metadata['column_count']}")
            if "page_number" in metadata:  # Page reference
                metadata_parts.append(f"page={metadata['page_number']}")
            if "position" in metadata:  # Document position
                metadata_parts.append(f"pos={metadata['position']}")
            if "section_title" in metadata:  # Section name
                title = metadata['section_title'][:20]
                if len(metadata['section_title']) > 20:
                    title += "..."
                metadata_parts.append(f"section={title}")
            
            if metadata_parts:
                metadata_str = ",".join(metadata_parts)
                base_tag += f":{metadata_str}"
        
        base_tag += "]"
        
        # Truncate if exceeds max length
        if len(base_tag) > self.max_tag_length:
            base_tag = base_tag[:self.max_tag_length-4] + "...]"
        
        return base_tag
    
    def parse_tag(self, tag_string: str) -> Optional[Dict[str, Any]]:
        """
        Parse a semantic tag back to its components.
        
        Args:
            tag_string: Tag string like "[PARENT:Section:id:metadata]"
            
        Returns:
            Dictionary with role, element_type, id, and metadata
        """
        if not tag_string.startswith("[") or not tag_string.endswith("]"):
            return None
        
        # Remove brackets
        content = tag_string[1:-1]
        
        # Split by colons (careful with metadata)
        parts = content.split(":", 3)
        
        if len(parts) < 2:
            return None
        
        result = {
            "role": parts[0],
            "element_type": parts[1]
        }
        
        if len(parts) > 2:
            result["element_id"] = parts[2]
        
        if len(parts) > 3:
            # Parse metadata
            metadata_str = parts[3]
            metadata = {}
            for item in metadata_str.split(","):
                if "=" in item:
                    key, value = item.split("=", 1)
                    # Try to convert to appropriate type
                    try:
                        if value.isdigit():
                            metadata[key] = int(value)
                        elif value.replace(".", "").isdigit():
                            metadata[key] = float(value)
                        else:
                            metadata[key] = value
                    except:
                        metadata[key] = value
            result["metadata"] = metadata
        
        return result
    
    def generate_context_tags(self,
                             context_elements: Dict[str, List[Dict[str, Any]]]) -> Dict[str, List[Tuple[str, str]]]:
        """
        Generate tags for all context elements.
        
        Args:
            context_elements: Dict with 'parents', 'siblings', 'children' lists
            
        Returns:
            Dict with same structure but (tag, content) tuples
        """
        tagged_context = {}
        
        # Process parents
        if "parents" in context_elements:
            tagged_parents = []
            for parent in context_elements["parents"]:
                tag = self.generate_tag(parent, context_role=ContextRole.PARENT)
                content = parent.get("content_preview", "")
                tagged_parents.append((tag, content))
            tagged_context["parents"] = tagged_parents
        
        # Process siblings
        if "siblings" in context_elements:
            tagged_siblings = []
            for i, sibling in enumerate(context_elements["siblings"]):
                # Determine if preceding or following
                if sibling.get("document_position", 0) < context_elements.get("element", {}).get("document_position", 0):
                    role = ContextRole.PRECEDING
                else:
                    role = ContextRole.FOLLOWING
                tag = self.generate_tag(sibling, context_role=role)
                content = sibling.get("content_preview", "")
                tagged_siblings.append((tag, content))
            tagged_context["siblings"] = tagged_siblings
        
        # Process children
        if "children" in context_elements:
            tagged_children = []
            for child in context_elements["children"]:
                tag = self.generate_tag(child, context_role=ContextRole.CHILD)
                content = child.get("content_preview", "")
                tagged_children.append((tag, content))
            tagged_context["children"] = tagged_children
        
        return tagged_context
    
    def format_tagged_content(self, tag: str, content: str, separator: str = " ") -> str:
        """
        Format tagged content for embedding.
        
        Args:
            tag: Semantic tag
            content: Text content
            separator: Separator between tag and content
            
        Returns:
            Formatted string with tag and content
        """
        return f"{tag}{separator}{content}"
    
    def extract_tags_from_text(self, text: str) -> List[str]:
        """
        Extract all semantic tags from text.
        
        Args:
            text: Text potentially containing semantic tags
            
        Returns:
            List of found tags
        """
        import re
        pattern = r'\[[A-Z_]+:[^]]+\]'
        return re.findall(pattern, text)