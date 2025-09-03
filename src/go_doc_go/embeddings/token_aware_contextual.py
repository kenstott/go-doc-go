"""
Token-aware contextual embedding generator.
Handles token limits intelligently when building context.
"""

import logging
from typing import List, Dict, Any, Optional, Tuple
import tiktoken  # OpenAI's tokenizer, can be replaced with model-specific

logger = logging.getLogger(__name__)


class TokenAwareContextualEmbedding:
    """
    Enhanced contextual embedding that respects token limits.
    """
    
    def __init__(self, 
                 base_generator,
                 max_tokens: int = 8192,  # Common limit for embedding models
                 target_ratio: Dict[str, float] = None,
                 tokenizer_model: str = "cl100k_base"):  # GPT-3.5/4 tokenizer
        """
        Initialize token-aware contextual embedding.
        
        Args:
            base_generator: Base embedding generator
            max_tokens: Maximum tokens for embedding model
            target_ratio: Target distribution of tokens (element:parent:siblings:children)
            tokenizer_model: Tokenizer model to use
        """
        self.base_generator = base_generator
        self.max_tokens = max_tokens
        
        # Reserve some tokens for safety margin
        self.safe_max_tokens = int(max_tokens * 0.95)
        
        # Default token distribution strategy
        self.target_ratio = target_ratio or {
            "element": 0.40,    # 40% for the main element
            "parents": 0.25,    # 25% for parent context
            "siblings": 0.20,   # 20% for sibling context  
            "children": 0.15    # 15% for child context
        }
        
        # Initialize tokenizer
        try:
            self.tokenizer = tiktoken.get_encoding(tokenizer_model)
        except:
            logger.warning(f"Could not load tokenizer {tokenizer_model}, using approximate counting")
            self.tokenizer = None
    
    def count_tokens(self, text: str) -> int:
        """
        Count tokens in text.
        
        Args:
            text: Text to count tokens for
            
        Returns:
            Number of tokens
        """
        if self.tokenizer:
            return len(self.tokenizer.encode(text))
        else:
            # Approximate: 1 token ≈ 4 characters or 0.75 words
            return max(len(text) // 4, len(text.split()) * 4 // 3)
    
    def truncate_to_tokens(self, text: str, max_tokens: int) -> str:
        """
        Truncate text to fit within token limit.
        
        Args:
            text: Text to truncate
            max_tokens: Maximum tokens allowed
            
        Returns:
            Truncated text
        """
        if self.tokenizer:
            tokens = self.tokenizer.encode(text)
            if len(tokens) <= max_tokens:
                return text
            truncated_tokens = tokens[:max_tokens]
            return self.tokenizer.decode(truncated_tokens)
        else:
            # Approximate truncation
            current_tokens = self.count_tokens(text)
            if current_tokens <= max_tokens:
                return text
            
            # Binary search for right truncation point
            words = text.split()
            target_words = len(words) * max_tokens // current_tokens
            return " ".join(words[:target_words])
    
    def build_context_with_budget(self,
                                  element_text: str,
                                  parent_texts: List[str],
                                  sibling_texts: List[str],
                                  child_texts: List[str]) -> str:
        """
        Build context respecting token budget.
        
        Args:
            element_text: Main element text
            parent_texts: Parent context texts (ordered by proximity)
            sibling_texts: Sibling context texts (ordered by proximity)
            child_texts: Child context texts (ordered by importance)
            
        Returns:
            Combined context within token limits
        """
        # Calculate token budgets
        element_budget = int(self.safe_max_tokens * self.target_ratio["element"])
        parent_budget = int(self.safe_max_tokens * self.target_ratio["parents"])
        sibling_budget = int(self.safe_max_tokens * self.target_ratio["siblings"])
        child_budget = int(self.safe_max_tokens * self.target_ratio["children"])
        
        # Process main element
        element_tokens = self.count_tokens(element_text)
        if element_tokens > element_budget:
            # Element alone exceeds budget - use smart truncation
            element_processed = self.smart_truncate(element_text, element_budget)
            logger.warning(f"Element truncated from {element_tokens} to {element_budget} tokens")
        else:
            element_processed = element_text
            # Redistribute unused tokens
            unused = element_budget - element_tokens
            parent_budget += unused // 3
            sibling_budget += unused // 3
            child_budget += unused - (unused // 3) * 2
        
        # Process contexts with their budgets
        parent_context = self.select_context_within_budget(parent_texts, parent_budget, "parent")
        sibling_context = self.select_context_within_budget(sibling_texts, sibling_budget, "sibling")
        child_context = self.select_context_within_budget(child_texts, child_budget, "child")
        
        # Combine all contexts
        combined_parts = []
        
        if parent_context:
            combined_parts.append("=== Parent Context ===")
            combined_parts.append(parent_context)
        
        if sibling_context:
            combined_parts.append("=== Sibling Context ===")
            combined_parts.append(sibling_context)
            
        combined_parts.append("=== Main Content ===")
        combined_parts.append(element_processed)
        
        if child_context:
            combined_parts.append("=== Child Context ===")
            combined_parts.append(child_context)
        
        combined = "\n\n".join(combined_parts)
        
        # Final safety check
        total_tokens = self.count_tokens(combined)
        if total_tokens > self.safe_max_tokens:
            logger.warning(f"Combined context {total_tokens} exceeds limit, applying final truncation")
            combined = self.truncate_to_tokens(combined, self.safe_max_tokens)
        
        return combined
    
    def select_context_within_budget(self, 
                                    texts: List[str], 
                                    budget: int,
                                    context_type: str) -> str:
        """
        Select and combine context texts within token budget.
        
        Args:
            texts: List of context texts (ordered by priority)
            budget: Token budget for this context type
            context_type: Type of context (for logging)
            
        Returns:
            Combined context within budget
        """
        if not texts or budget <= 0:
            return ""
        
        selected = []
        used_tokens = 0
        
        for text in texts:
            text_tokens = self.count_tokens(text)
            
            if used_tokens + text_tokens <= budget:
                # Fits completely
                selected.append(text)
                used_tokens += text_tokens
            elif used_tokens < budget:
                # Partially fits - truncate to fit remaining budget
                remaining_budget = budget - used_tokens
                if remaining_budget > 50:  # Only include if meaningful amount remains
                    truncated = self.truncate_to_tokens(text, remaining_budget)
                    selected.append(truncated)
                    break
            else:
                # No more room
                break
        
        if selected:
            logger.debug(f"Selected {len(selected)}/{len(texts)} {context_type} contexts, used {used_tokens}/{budget} tokens")
        
        return "\n---\n".join(selected)
    
    def smart_truncate(self, text: str, max_tokens: int) -> str:
        """
        Smart truncation that preserves beginning and end.
        
        Args:
            text: Text to truncate
            max_tokens: Maximum tokens
            
        Returns:
            Truncated text with beginning and end preserved
        """
        current_tokens = self.count_tokens(text)
        if current_tokens <= max_tokens:
            return text
        
        # Reserve tokens for ellipsis
        ellipsis = "\n\n[... content truncated ...]\n\n"
        ellipsis_tokens = self.count_tokens(ellipsis)
        
        # Calculate tokens for beginning and end
        content_budget = max_tokens - ellipsis_tokens
        begin_budget = content_budget * 2 // 3
        end_budget = content_budget - begin_budget
        
        # Truncate
        begin_text = self.truncate_to_tokens(text, begin_budget)
        
        # Get end part
        if self.tokenizer:
            tokens = self.tokenizer.encode(text)
            end_tokens = tokens[-end_budget:]
            end_text = self.tokenizer.decode(end_tokens)
        else:
            # Approximate
            words = text.split()
            end_words = end_budget * len(words) // current_tokens
            end_text = " ".join(words[-end_words:])
        
        return begin_text + ellipsis + end_text
    
    def generate_with_context(self,
                             element: Dict[str, Any],
                             context_elements: Dict[str, List[Dict[str, Any]]],
                             resolver) -> List[float]:
        """
        Generate embedding with token-aware context.
        
        Args:
            element: Element to embed
            context_elements: Dict with 'parents', 'siblings', 'children' lists
            resolver: Content resolver for getting text
            
        Returns:
            Embedding vector
        """
        # Get element text
        element_text = resolver.resolve_content(
            element.get('content_location'), 
            text=True
        )
        
        # Get context texts
        parent_texts = []
        for parent in context_elements.get('parents', []):
            text = resolver.resolve_content(parent.get('content_location'), text=True)
            if text:
                parent_texts.append(text)
        
        sibling_texts = []
        for sibling in context_elements.get('siblings', []):
            text = resolver.resolve_content(sibling.get('content_location'), text=True)
            if text:
                sibling_texts.append(text)
        
        child_texts = []
        for child in context_elements.get('children', [])[:5]:  # Limit children
            text = resolver.resolve_content(child.get('content_location'), text=True)
            if text:
                child_texts.append(text)
        
        # Build context with token budget
        combined_context = self.build_context_with_budget(
            element_text,
            parent_texts,
            sibling_texts,
            child_texts
        )
        
        # Log token usage
        total_tokens = self.count_tokens(combined_context)
        logger.info(f"Generating embedding with {total_tokens}/{self.max_tokens} tokens")
        
        # Generate embedding
        return self.base_generator.generate(combined_context)


class AdaptiveContextStrategy:
    """
    Adaptive strategy for handling different document types and sizes.
    """
    
    def __init__(self, max_tokens: int = 8192):
        self.max_tokens = max_tokens
        
        # Different strategies for different scenarios
        self.strategies = {
            "small_doc": {
                "element": 0.30,
                "parents": 0.30,
                "siblings": 0.25,
                "children": 0.15
            },
            "large_doc": {
                "element": 0.50,  # Focus more on element itself
                "parents": 0.20,
                "siblings": 0.15,
                "children": 0.15
            },
            "deep_hierarchy": {
                "element": 0.35,
                "parents": 0.35,  # More parent context for deep trees
                "siblings": 0.15,
                "children": 0.15
            },
            "flat_structure": {
                "element": 0.35,
                "parents": 0.15,
                "siblings": 0.35,  # More sibling context for flat docs
                "children": 0.15
            }
        }
    
    def select_strategy(self, 
                       element: Dict[str, Any],
                       doc_stats: Dict[str, Any]) -> Dict[str, float]:
        """
        Select best token distribution strategy based on document characteristics.
        
        Args:
            element: Current element
            doc_stats: Document statistics (depth, breadth, size)
            
        Returns:
            Token distribution ratios
        """
        # Analyze document characteristics
        total_elements = doc_stats.get('total_elements', 0)
        max_depth = doc_stats.get('max_depth', 0)
        avg_siblings = doc_stats.get('avg_siblings', 0)
        
        # Select strategy
        if total_elements < 50:
            return self.strategies["small_doc"]
        elif total_elements > 1000:
            return self.strategies["large_doc"]
        elif max_depth > 6:
            return self.strategies["deep_hierarchy"]
        elif avg_siblings > 10:
            return self.strategies["flat_structure"]
        else:
            # Default balanced strategy
            return {
                "element": 0.40,
                "parents": 0.25,
                "siblings": 0.20,
                "children": 0.15
            }