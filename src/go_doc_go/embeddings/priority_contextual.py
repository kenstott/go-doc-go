"""
Priority-based contextual embedding generator.
Orders elements by priority and adds until token limit reached.
"""

import logging
from typing import List, Dict, Any, Optional, Tuple
from dataclasses import dataclass
from enum import Enum

from .semantic_tagger import SemanticTagger, ContextRole

try:
    import tiktoken
    TIKTOKEN_AVAILABLE = True
except ImportError:
    TIKTOKEN_AVAILABLE = False

logger = logging.getLogger(__name__)


class Priority(Enum):
    """Priority levels for context elements."""
    CRITICAL = 1    # Main element - always included
    HIGH = 2        # Direct parents and immediate siblings
    MEDIUM = 3      # Extended context (grandparents, nearby siblings)
    LOW = 4         # Children and distant relatives
    OPTIONAL = 5    # Nice to have if space permits


@dataclass
class PrioritizedContent:
    """Content with priority and metadata."""
    text: str
    priority: Priority
    role: ContextRole
    metadata: Dict[str, Any]
    element_id: Optional[str] = None
    relationship_distance: int = 0  # How far from main element


class PriorityContextualEmbedding:
    """
    Contextual embedding using priority-based token management.
    
    Simple approach:
    1. Assign priorities to all context elements
    2. Sort by priority (and distance within same priority)
    3. Add elements until token limit reached
    4. Always include semantic tags for better context
    """
    
    def __init__(self,
                 base_generator,
                 max_tokens: int = 16384,
                 tokenizer_model: str = "cl100k_base",
                 use_semantic_tags: bool = True,
                 reserve_ratio: float = 0.05):
        """
        Initialize priority-based contextual embedding.
        
        Args:
            base_generator: Base embedding generator
            max_tokens: Maximum tokens for embedding model
            tokenizer_model: Tokenizer to use for counting
            use_semantic_tags: Whether to use semantic tags
            reserve_ratio: Reserve ratio for safety (default 5%)
        """
        self.base_generator = base_generator
        self.max_tokens = max_tokens
        self.safe_max_tokens = int(max_tokens * (1 - reserve_ratio))
        
        # Initialize tokenizer
        self.tokenizer = None
        if TIKTOKEN_AVAILABLE:
            try:
                self.tokenizer = tiktoken.get_encoding(tokenizer_model)
                logger.info(f"Initialized tokenizer: {tokenizer_model}")
            except Exception as e:
                logger.warning(f"Could not load tokenizer {tokenizer_model}: {e}")
        
        # Semantic tagging
        self.use_semantic_tags = use_semantic_tags
        self.semantic_tagger = SemanticTagger(include_metadata=True) if use_semantic_tags else None
    
    def count_tokens(self, text: str) -> int:
        """
        Count tokens in text.
        
        Args:
            text: Text to count
            
        Returns:
            Number of tokens
        """
        if self.tokenizer:
            return len(self.tokenizer.encode(text))
        else:
            # Approximate: 1 token ≈ 4 characters or 0.75 words
            return max(len(text) // 4, len(text.split()) * 4 // 3)
    
    def build_prioritized_context(self,
                                 element: Dict[str, Any],
                                 context_elements: Dict[str, List[Dict[str, Any]]]) -> Tuple[str, Dict[str, Any]]:
        """
        Build context using priority ordering.
        
        Args:
            element: Main element with text and metadata
            context_elements: Dict with 'parents', 'siblings', 'children' lists
            
        Returns:
            Tuple of (combined_context, stats_dict)
        """
        prioritized = []
        
        # 1. Main element is always CRITICAL priority
        main_text = element.get("content_preview", element.get("text", ""))
        prioritized.append(PrioritizedContent(
            text=main_text,
            priority=Priority.CRITICAL,
            role=ContextRole.MAIN,  # Use MAIN role for the primary element
            metadata=element,
            element_id=element.get("element_id")
        ))
        
        # 2. Direct parents are HIGH priority
        for i, parent in enumerate(context_elements.get("parents", [])):
            parent_text = parent.get("content_preview", parent.get("text", ""))
            prioritized.append(PrioritizedContent(
                text=parent_text,
                priority=Priority.HIGH if i == 0 else Priority.MEDIUM,
                role=ContextRole.PARENT,
                metadata=parent,
                element_id=parent.get("element_id"),
                relationship_distance=i + 1
            ))
        
        # 3. Immediate siblings are HIGH, distant ones are MEDIUM
        for i, sibling in enumerate(context_elements.get("siblings", [])):
            sibling_text = sibling.get("content_preview", sibling.get("text", ""))
            # Determine if preceding or following
            sibling_pos = sibling.get("document_position", 0)
            element_pos = element.get("document_position", 0)
            role = ContextRole.PRECEDING if sibling_pos < element_pos else ContextRole.FOLLOWING
            
            prioritized.append(PrioritizedContent(
                text=sibling_text,
                priority=Priority.HIGH if i < 2 else Priority.MEDIUM,
                role=role,
                metadata=sibling,
                element_id=sibling.get("element_id"),
                relationship_distance=abs(i - element_pos) if element_pos else i
            ))
        
        # 4. Children are LOW priority (unless specifically important)
        for i, child in enumerate(context_elements.get("children", [])):
            child_text = child.get("content_preview", child.get("text", ""))
            # First few children might be more important
            priority = Priority.MEDIUM if i < 3 else Priority.LOW
            
            prioritized.append(PrioritizedContent(
                text=child_text,
                priority=priority,
                role=ContextRole.CHILD,
                metadata=child,
                element_id=child.get("element_id"),
                relationship_distance=i
            ))
        
        # 5. Sort by priority, then by distance
        # But keep CRITICAL (main element) at the beginning
        prioritized.sort(key=lambda x: (x.priority.value, x.relationship_distance))
        
        # Ensure main element (CRITICAL) is first
        critical = [p for p in prioritized if p.priority == Priority.CRITICAL]
        non_critical = [p for p in prioritized if p.priority != Priority.CRITICAL]
        prioritized = critical + non_critical
        
        # 6. Build context until token limit
        combined_parts = []
        total_tokens = 0
        included_count = 0
        excluded_count = 0
        
        for item in prioritized:
            # Generate tagged or plain text
            if self.use_semantic_tags:
                tag = self.semantic_tagger.generate_tag(item.metadata, context_role=item.role)
                formatted_text = f"{tag} {item.text}"
            else:
                # Use simple role prefix
                formatted_text = f"[{item.role.value}] {item.text}"
            
            # Check if it fits
            text_tokens = self.count_tokens(formatted_text)
            if total_tokens + text_tokens <= self.safe_max_tokens:
                combined_parts.append(formatted_text)
                total_tokens += text_tokens
                included_count += 1
                logger.debug(f"Added {item.role.value} ({item.priority.name}): {text_tokens} tokens, total: {total_tokens}")
            else:
                excluded_count += 1
                logger.debug(f"Excluded {item.role.value} ({item.priority.name}): would exceed limit")
                # Don't break - maybe smaller items later could fit
                if item.priority == Priority.CRITICAL:
                    # If we can't fit the main element, truncate it
                    remaining = self.safe_max_tokens - total_tokens
                    if remaining > 20:  # Lower threshold for including truncated main element
                        truncated = self.truncate_to_tokens(item.text, remaining - 10)
                        if self.use_semantic_tags:
                            tag = self.semantic_tagger.generate_tag(item.metadata, context_role=item.role)
                            formatted_text = f"{tag} {truncated} [truncated]"
                        else:
                            formatted_text = f"[{item.role.value}] {truncated} [truncated]"
                        combined_parts.append(formatted_text)
                        total_tokens = self.safe_max_tokens
                        included_count += 1
                        break
        
        combined_context = "\n\n".join(combined_parts)
        
        # Stats for debugging/monitoring
        stats = {
            "total_tokens": total_tokens,
            "max_tokens": self.max_tokens,
            "safe_max_tokens": self.safe_max_tokens,
            "included_elements": included_count,
            "excluded_elements": excluded_count,
            "utilization": total_tokens / self.safe_max_tokens if self.safe_max_tokens > 0 else 0
        }
        
        logger.info(f"Built context: {total_tokens}/{self.safe_max_tokens} tokens, "
                   f"{included_count} included, {excluded_count} excluded")
        
        return combined_context, stats
    
    def truncate_to_tokens(self, text: str, max_tokens: int) -> str:
        """
        Truncate text to fit within token limit.
        
        Args:
            text: Text to truncate
            max_tokens: Maximum tokens
            
        Returns:
            Truncated text
        """
        if self.tokenizer:
            tokens = self.tokenizer.encode(text)
            if len(tokens) <= max_tokens:
                return text
            return self.tokenizer.decode(tokens[:max_tokens])
        else:
            # Approximate truncation
            current_tokens = self.count_tokens(text)
            if current_tokens <= max_tokens:
                return text
            
            # Proportional word truncation
            words = text.split()
            target_words = len(words) * max_tokens // current_tokens
            return " ".join(words[:target_words])
    
    def generate_with_context(self,
                             element: Dict[str, Any],
                             context_elements: Dict[str, List[Dict[str, Any]]]) -> Tuple[List[float], Dict[str, Any]]:
        """
        Generate embedding with prioritized context.
        
        Args:
            element: Main element
            context_elements: Context elements dict
            
        Returns:
            Tuple of (embedding_vector, stats)
        """
        # Build prioritized context
        combined_context, stats = self.build_prioritized_context(element, context_elements)
        
        # Generate embedding
        embedding = self.base_generator.generate(combined_context)
        
        return embedding, stats


class DynamicPriorityStrategy:
    """
    Dynamic priority assignment based on document structure and content.
    """
    
    def __init__(self):
        """Initialize dynamic priority strategy."""
        self.important_types = {
            "header", "title", "heading",  # Structural elements
            "summary", "abstract", "conclusion",  # Summary elements  
            "definition", "theorem", "equation"  # Key content
        }
        
        self.contextual_keywords = {
            "refers to", "see also", "as defined in",
            "according to", "based on", "derived from"
        }
    
    def calculate_priority(self,
                          element: Dict[str, Any],
                          relationship: str,
                          distance: int = 0) -> Priority:
        """
        Calculate dynamic priority for an element.
        
        Args:
            element: Element dict with type and metadata
            relationship: Relationship to main element
            distance: Distance from main element
            
        Returns:
            Priority level
        """
        element_type = element.get("element_type", "").lower()
        
        # Special handling for important element types
        if element_type in self.important_types:
            if relationship == "parent":
                return Priority.HIGH
            elif relationship == "sibling" and distance <= 1:
                return Priority.HIGH
            else:
                return Priority.MEDIUM
        
        # Check for cross-references in content
        content = element.get("content_preview", "").lower()
        if any(keyword in content for keyword in self.contextual_keywords):
            return Priority.MEDIUM if distance <= 2 else Priority.LOW
        
        # Default priority based on relationship and distance
        if relationship == "parent":
            return Priority.HIGH if distance == 0 else Priority.MEDIUM
        elif relationship == "sibling":
            if distance <= 1:
                return Priority.HIGH
            elif distance <= 3:
                return Priority.MEDIUM
            else:
                return Priority.LOW
        elif relationship == "child":
            return Priority.MEDIUM if distance == 0 else Priority.LOW
        else:
            return Priority.OPTIONAL
    
    def reorder_by_relevance(self,
                            prioritized: List[PrioritizedContent],
                            main_element: Dict[str, Any]) -> List[PrioritizedContent]:
        """
        Reorder elements within same priority by relevance.
        
        Args:
            prioritized: List of prioritized content
            main_element: Main element for relevance calculation
            
        Returns:
            Reordered list
        """
        # Group by priority
        priority_groups = {}
        for item in prioritized:
            if item.priority not in priority_groups:
                priority_groups[item.priority] = []
            priority_groups[item.priority].append(item)
        
        # Reorder within each group by relevance
        reordered = []
        for priority in sorted(priority_groups.keys(), key=lambda x: x.value):
            group = priority_groups[priority]
            
            # Sort by relevance metrics within group
            if priority != Priority.CRITICAL:  # Don't reorder main element
                group.sort(key=lambda x: (
                    x.relationship_distance,  # Closer is better
                    -len(x.text),  # Longer might be more informative
                    x.element_id or ""  # Stable sort by ID
                ))
            
            reordered.extend(group)
        
        return reordered