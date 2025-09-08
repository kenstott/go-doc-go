from typing import Optional, List, Dict, Any, Tuple
import logging

from .base import EmbeddingGenerator
from .semantic_tagger import SemanticTagger, ContextRole
from ..adapter import create_content_resolver
from ..config import Config

try:
    import tiktoken
    TIKTOKEN_AVAILABLE = True
except ImportError:
    TIKTOKEN_AVAILABLE = False
    logging.warning("tiktoken not available, using approximate token counting")

logger = logging.getLogger(__name__)


class ContextualEmbeddingGenerator(EmbeddingGenerator):
    """
    Embedding generator that includes context for better semantic understanding.

    This generator creates embeddings that include context from surrounding elements,
    creating overlapping context windows to improve semantic search quality.
    """

    def __init__(self,
                 _config: Config,
                 base_generator: EmbeddingGenerator,
                 window_size: int = 3,
                 overlap_size: int = 1,
                 predecessor_count: int = 1,
                 successor_count: int = 1,
                 ancestor_depth: int = 1,
                 child_count: int = 1,
                 max_tokens: int = 8192,
                 tokenizer_model: str = "cl100k_base",
                 use_semantic_tags: bool = True):
        """
        Initialize the contextual embedding generator.

        Args:
            base_generator: Base embedding generator
            window_size: Number of elements in context window
            overlap_size: Number of elements to overlap between windows
            predecessor_count: Number of preceding elements to include
            successor_count: Number of following elements to include
            ancestor_depth: Number of ancestral levels to include
        """
        super().__init__(_config)
        self.base_generator = base_generator
        self.window_size = window_size
        self.overlap_size = overlap_size
        self.predecessor_count = predecessor_count
        self.successor_count = successor_count
        self.ancestor_depth = ancestor_depth
        self.child_count = child_count
        
        # Token management
        self.max_tokens = max_tokens
        self.safe_max_tokens = int(max_tokens * 0.95)  # Safety margin
        
        # Token distribution strategy
        self.token_ratios = {
            "element": 0.40,
            "parents": 0.25,
            "siblings": 0.20,
            "children": 0.15
        }
        
        # Semantic tagging
        self.use_semantic_tags = use_semantic_tags
        self.semantic_tagger = SemanticTagger(include_metadata=True) if use_semantic_tags else None
        
        # Initialize tokenizer
        self.tokenizer = None
        if TIKTOKEN_AVAILABLE:
            try:
                self.tokenizer = tiktoken.get_encoding(tokenizer_model)
                logger.info(f"Initialized tokenizer: {tokenizer_model}")
            except Exception as e:
                logger.warning(f"Could not load tokenizer {tokenizer_model}: {e}")
        else:
            logger.warning("Using approximate token counting")

    def generate(self, text: str, context: Optional[List[str]] = None) -> List[float]:
        """
        Generate embedding for text with context.

        Args:
            text: Main text to embed
            context: List of context texts (optional)

        Returns:
            Vector embedding
        """
        if not context:
            # No context, just generate embedding for text
            return self.base_generator.generate(text)

        # Combine text with context
        combined_text = self._combine_text_with_context(text, context)

        # Generate embedding for combined text
        return self.base_generator.generate(combined_text)
    
    def count_tokens(self, text: str) -> int:
        """
        Count tokens in text using tiktoken or approximation.
        
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
            
            # Proportional truncation
            words = text.split()
            target_words = len(words) * max_tokens // current_tokens
            return " ".join(words[:target_words])
    
    def smart_truncate(self, text: str, max_tokens: int) -> str:
        """
        Smart truncation that preserves beginning and end of content.
        
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
        ellipsis = "\n[...truncated...]\n"
        ellipsis_tokens = self.count_tokens(ellipsis)
        
        if max_tokens <= ellipsis_tokens:
            # Very small budget, just truncate normally
            return self.truncate_to_tokens(text, max_tokens)
        
        # Calculate tokens for beginning and end
        content_budget = max_tokens - ellipsis_tokens
        begin_budget = content_budget * 2 // 3
        end_budget = content_budget - begin_budget
        
        # Get beginning part
        begin_text = self.truncate_to_tokens(text, begin_budget)
        
        # Get end part
        if self.tokenizer:
            tokens = self.tokenizer.encode(text)
            if len(tokens) > end_budget:
                end_tokens = tokens[-end_budget:]
                end_text = self.tokenizer.decode(end_tokens)
            else:
                end_text = text
        else:
            # Approximate
            words = text.split()
            end_words = end_budget * len(words) // current_tokens
            end_text = " ".join(words[-end_words:]) if end_words > 0 else ""
        
        return begin_text + ellipsis + end_text
    
    def build_structured_context(self,
                                 element_text: str,
                                 parent_texts: List[str],
                                 sibling_texts: List[str], 
                                 child_texts: List[str],
                                 element_metadata: Optional[Dict[str, Any]] = None,
                                 parent_metadata: Optional[List[Dict[str, Any]]] = None,
                                 sibling_metadata: Optional[List[Dict[str, Any]]] = None,
                                 child_metadata: Optional[List[Dict[str, Any]]] = None) -> str:
        """
        Build context with separate budgets for different context types.
        
        Args:
            element_text: Main element text
            parent_texts: Parent context texts (ordered by proximity)
            sibling_texts: Sibling context texts (ordered by document position)
            child_texts: Child context texts (ordered by importance)
            
        Returns:
            Combined context within token limits
        """
        # Calculate individual budgets
        element_budget = int(self.safe_max_tokens * self.token_ratios["element"])
        parent_budget = int(self.safe_max_tokens * self.token_ratios["parents"])
        sibling_budget = int(self.safe_max_tokens * self.token_ratios["siblings"])
        child_budget = int(self.safe_max_tokens * self.token_ratios["children"])
        
        # Process main element
        element_tokens = self.count_tokens(element_text)
        if element_tokens > element_budget:
            element_processed = self.smart_truncate(element_text, element_budget)
            logger.warning(f"Element truncated from {element_tokens} to {element_budget} tokens")
        else:
            element_processed = element_text
            # Redistribute unused element tokens
            unused = element_budget - element_tokens
            parent_budget += unused // 3
            sibling_budget += unused // 3
            child_budget += unused - (unused // 3) * 2
        
        # Process different context types
        contexts = []
        
        # Add parent context
        if parent_texts and parent_budget > 0:
            if self.use_semantic_tags and parent_metadata:
                parent_context = self._select_tagged_texts_within_budget(
                    parent_texts, parent_metadata, parent_budget, ContextRole.PARENT
                )
            else:
                parent_context = self._select_texts_within_budget(parent_texts, parent_budget, "Parent")
                if parent_context:
                    parent_context = f"=== Parent Context ===\n{parent_context}"
            if parent_context:
                contexts.append(parent_context)
        
        # Add sibling context
        if sibling_texts and sibling_budget > 0:
            if self.use_semantic_tags and sibling_metadata:
                sibling_context = self._select_tagged_texts_within_budget(
                    sibling_texts, sibling_metadata, sibling_budget, ContextRole.SIBLING
                )
            else:
                sibling_context = self._select_texts_within_budget(sibling_texts, sibling_budget, "Sibling")
                if sibling_context:
                    sibling_context = f"=== Sibling Context ===\n{sibling_context}"
            if sibling_context:
                contexts.append(sibling_context)
        
        # Add child context
        if child_texts and child_budget > 0:
            if self.use_semantic_tags and child_metadata:
                child_context = self._select_tagged_texts_within_budget(
                    child_texts, child_metadata, child_budget, ContextRole.CHILD
                )
            else:
                child_context = self._select_texts_within_budget(child_texts, child_budget, "Child")
                if child_context:
                    child_context = f"=== Child Context ===\n{child_context}"
            if child_context:
                contexts.append(child_context)
        
        # Combine all parts - MAIN element first
        if self.use_semantic_tags and element_metadata:
            element_tag = self.semantic_tagger.generate_tag(element_metadata, context_role=ContextRole.MAIN)
            main_content = f"{element_tag} {element_processed}"
        else:
            main_content = f"=== Main Content ===\n{element_processed}"
        
        # Main element goes first, then context
        all_parts = [main_content] + contexts
        combined = "\n\n".join(all_parts)
        
        # Final safety check
        total_tokens = self.count_tokens(combined)
        if total_tokens > self.safe_max_tokens:
            logger.error(f"Emergency: Combined context {total_tokens} still exceeds limit after budgeting")
            combined = self.truncate_to_tokens(combined, self.safe_max_tokens)
        
        return combined
    
    def _select_texts_within_budget(self, texts: List[str], budget: int, context_type: str) -> str:
        """
        Select and combine texts within token budget.
        
        Args:
            texts: List of texts (ordered by priority)
            budget: Token budget
            context_type: Type of context for logging
            
        Returns:
            Combined text within budget
        """
        if not texts or budget <= 0:
            return ""
        
        selected = []
        used_tokens = 0
        
        for text in texts:
            text_tokens = self.count_tokens(text)
            
            if used_tokens + text_tokens <= budget:
                selected.append(text)
                used_tokens += text_tokens
            elif used_tokens < budget and budget - used_tokens > 50:
                # Partial fit with meaningful remaining space
                remaining = budget - used_tokens
                truncated = self.truncate_to_tokens(text, remaining)
                selected.append(truncated + " [...]")
                break
            else:
                break
        
        result = "\n---\n".join(selected)
        logger.debug(f"{context_type} context: {len(selected)}/{len(texts)} texts, {used_tokens}/{budget} tokens")
        
        return result
    
    def _select_tagged_texts_within_budget(self,
                                          texts: List[str],
                                          metadata_list: List[Dict[str, Any]],
                                          budget: int,
                                          context_role: ContextRole) -> str:
        """
        Select and tag texts within token budget.
        
        Args:
            texts: List of context texts
            metadata_list: List of metadata dicts for each text
            budget: Token budget
            context_role: Role of this context
            
        Returns:
            Tagged and combined text within budget
        """
        if not texts or budget <= 0:
            return ""
        
        selected = []
        used_tokens = 0
        
        for i, text in enumerate(texts):
            # Get metadata for this text
            metadata = metadata_list[i] if i < len(metadata_list) else {}
            
            # Generate semantic tag
            tag = self.semantic_tagger.generate_tag(metadata, context_role=context_role)
            tagged_text = f"{tag} {text}"
            
            text_tokens = self.count_tokens(tagged_text)
            
            if used_tokens + text_tokens <= budget:
                selected.append(tagged_text)
                used_tokens += text_tokens
            elif used_tokens < budget and budget - used_tokens > 50:
                # Partial fit with meaningful remaining space
                remaining = budget - used_tokens
                # Ensure tag is included even in truncation
                tag_tokens = self.count_tokens(tag + " ")
                if tag_tokens < remaining:
                    text_budget = remaining - tag_tokens
                    truncated = self.truncate_to_tokens(text, text_budget)
                    selected.append(f"{tag} {truncated} [...]")
                break
            else:
                break
        
        result = "\n".join(selected)
        logger.debug(f"{context_role.value} context: {len(selected)}/{len(texts)} texts, {used_tokens}/{budget} tokens")
        
        return result

    def generate_batch(self, texts: List[str], contexts: Optional[List[List[str]]] = None) -> List[List[float]]:
        """
        Generate embeddings for multiple texts with contexts.

        Args:
            texts: List of input texts
            contexts: List of context lists (optional)

        Returns:
            List of vector embeddings
        """
        if not contexts:
            # No contexts, just generate embeddings for texts
            return self.base_generator.generate_batch(texts)

        # Ensure contexts list has same length as texts
        if len(contexts) != len(texts):
            raise ValueError("Length of contexts must match length of texts")

        # Combine texts with contexts
        combined_texts = [
            self._combine_text_with_context(text, context)
            for text, context in zip(texts, contexts)
        ]

        # Generate embeddings for combined texts
        return self.base_generator.generate_batch(combined_texts)

    def get_dimensions(self) -> int:
        """Get embedding dimensions."""
        return self.base_generator.get_dimensions()

    def get_model_name(self) -> str:
        """Get embedding model name."""
        return f"contextual-{self.base_generator.get_model_name()}"

    def clear_cache(self) -> None:
        """Clear the embedding cache."""
        self.base_generator.clear_cache()

    def _combine_text_with_context(self, text: str, context: List[str]) -> str:
        """
        Combine text with context texts using token-aware budget management.

        Args:
            text: Main text
            context: List of context texts

        Returns:
            Combined text within token limits
        """
        # Calculate token budgets
        element_budget = int(self.safe_max_tokens * self.token_ratios["element"])
        context_budget = self.safe_max_tokens - element_budget
        
        # Process main element text
        element_tokens = self.count_tokens(text)
        if element_tokens > element_budget:
            text = self.smart_truncate(text, element_budget)
            logger.warning(f"Element text truncated from {element_tokens} to {element_budget} tokens")
            actual_element_tokens = element_budget
        else:
            actual_element_tokens = element_tokens
            # Redistribute unused tokens to context
            unused = element_budget - actual_element_tokens
            context_budget += unused
        
        # Process context with remaining budget
        if not context or context_budget <= 0:
            return text
        
        # Select context texts that fit within budget
        selected_context = []
        used_tokens = 0
        
        for ctx in context:
            ctx_tokens = self.count_tokens(ctx)
            
            if used_tokens + ctx_tokens <= context_budget:
                # Fits completely
                selected_context.append(ctx)
                used_tokens += ctx_tokens
            elif used_tokens < context_budget:
                # Partially fits - truncate to fit
                remaining_budget = context_budget - used_tokens
                if remaining_budget > 50:  # Only include if meaningful amount remains
                    truncated_ctx = self.truncate_to_tokens(ctx, remaining_budget)
                    selected_context.append(truncated_ctx)
                break
            else:
                # No more room
                break
        
        # Combine with separators
        if selected_context:
            combined = f"=== Context ===\n{chr(10).join(selected_context)}\n\n=== Main Content ===\n{text}"
        else:
            combined = text
        
        # Final safety check
        total_tokens = self.count_tokens(combined)
        if total_tokens > self.safe_max_tokens:
            logger.warning(f"Final combined text {total_tokens} exceeds limit, applying emergency truncation")
            combined = self.truncate_to_tokens(combined, self.safe_max_tokens)
        
        logger.debug(f"Combined context: {total_tokens}/{self.max_tokens} tokens, {len(selected_context)}/{len(context)} contexts used")
        
        return combined

    @staticmethod
    def is_number(value: str) -> bool:
        """
        Check if the given string represents an integer or a float.

        Args:
            value: The string to check.
        Returns:
            True if the string is an int or a float, False otherwise.
        """
        try:
            float(value)  # Try converting to float (handles integers too)
            return True
        except ValueError:
            return False

    def generate_from_elements(self, elements: List[Dict[str, Any]]) -> Dict[str, List[float]]:
        """
        Generate contextual embeddings for document elements, with size handling:
        - Skip root elements that exceed the size threshold
        - Truncate non-root elements that exceed the size threshold
        """
        # Build element hierarchy
        hierarchy = self._build_element_hierarchy(elements)
        resolver = create_content_resolver(self._config)

        # Define maximum content size for effective embedding (approximate word count)
        max_words_for_embedding = 500

        # Generate embeddings with context
        embeddings = {}

        for element in elements:
            element_pk = element["element_pk"]

            # Get full text content for all elements using the resolver
            content = resolver.resolve_content(element.get('content_location'), text=True)

            # Skip if no meaningful content
            if not content and not self.is_number(content):
                continue

            # Check content length
            word_count = len(content.split())
            if word_count > max_words_for_embedding:
                # For root elements, skip entirely
                if element["element_type"] == "root":
                    continue

                # For non-root elements, truncate to threshold
                content = " ".join(content.split()[:max_words_for_embedding])

            # Get context elements
            context_elements = self._get_context_elements(element, elements, hierarchy)

            # Get context contents using the resolver for text
            context_contents = []
            for ctx_element in context_elements:
                ctx_content = resolver.resolve_content(ctx_element.get('content_location'), text=True)
                if ctx_content and not self.is_number(ctx_content):
                    # Also check size of context elements and truncate if needed
                    ctx_words = len(ctx_content.split())
                    if ctx_words > max_words_for_embedding:
                        ctx_content = " ".join(ctx_content.split()[:max_words_for_embedding])
                    context_contents.append(ctx_content)

            # Generate embedding with context
            embedding = self.generate(content, context_contents)
            embeddings[element_pk] = embedding

        return embeddings

    @staticmethod
    def _build_element_hierarchy(elements: List[Dict[str, Any]]) -> Dict[str, List[str]]:
        """
        Build element hierarchy for context lookup.

        Args:
            elements: List of document elements

        Returns:
            Dictionary mapping parent_id to list of child element_ids
        """
        hierarchy = {}

        for element in elements:
            parent_id = element.get("parent_id")
            element_id = element["element_id"]

            if parent_id:
                if parent_id not in hierarchy:
                    hierarchy[parent_id] = []

                hierarchy[parent_id].append(element_id)

        return hierarchy

    def _get_context_elements(self, element: Dict[str, Any],
                              all_elements: List[Dict[str, Any]],
                              hierarchy: Dict[str, List[str]]) -> List[Dict[str, Any]]:
        """
        Get context elements for an element.

        This includes:
        - Ancestors up to configured depth (skipping those with blank content)
        - Meaningful predecessors (elements that come before in document order)
        - Meaningful successors (elements that come after in document order)
        - A limited number of meaningful children (directly nested elements)

        Args:
            element: Element to get context for
            all_elements: List of all elements
            hierarchy: Element hierarchy

        Returns:
            List of context elements
        """
        element_id = element["element_id"]
        context_ids = set()

        # Build a mapping from element_id to element for quicker lookups
        id_to_element = {e["element_id"]: e for e in all_elements}

        # Add ancestors up to configured depth
        current_element = element
        current_depth = 0
        ancestors_added = 0

        while ancestors_added < self.ancestor_depth:
            parent_id = current_element.get("parent_id")
            if not parent_id:
                break  # No more ancestors

            # Find parent element to continue up the hierarchy
            parent_element = id_to_element.get(parent_id)
            if not parent_element:
                break  # Parent not found

            # Only include parent if it has content and is not an empty container
            if (parent_element.get("content_preview") and
                    parent_element["element_type"] != "root" and
                    not self._is_empty_container(parent_element)):
                context_ids.add(parent_id)
                ancestors_added += 1

            # Move up to the next level, even if we skipped this parent
            current_element = parent_element
            current_depth += 1

            # Safety check - don't go too far up (avoid infinite loops)
            if current_depth > 10:  # Arbitrary depth limit
                break

        # Find meaningful predecessors and successors
        current_index = -1
        for i, e in enumerate(all_elements):
            if e["element_id"] == element_id:
                current_index = i
                break

        if current_index >= 0:
            # Get meaningful predecessors (elements that come before)
            pred_count = 0
            i = current_index - 1

            while i >= 0 and pred_count < self.predecessor_count:
                pred_element = all_elements[i]

                # Skip elements that:
                # 1. Are root elements
                # 2. Don't have content (empty content_preview)
                # 3. Are just container elements
                if (pred_element["element_type"] != "root" and
                        pred_element.get("content_preview") and
                        not self._is_empty_container(pred_element)):
                    context_ids.add(pred_element["element_id"])
                    pred_count += 1

                i -= 1

            # Get meaningful successors (elements that come after)
            succ_count = 0
            i = current_index + 1

            while i < len(all_elements) and succ_count < self.successor_count:
                succ_element = all_elements[i]

                # Same filtering as for predecessors
                if (succ_element["element_type"] != "root" and
                        succ_element.get("content_preview") and
                        not self._is_empty_container(succ_element)):
                    context_ids.add(succ_element["element_id"])
                    succ_count += 1

                i += 1

        # Add a limited number of meaningful children
        if element_id in hierarchy and self.child_count > 0:
            children_added = 0

            for child_id in hierarchy[element_id]:
                # Apply same filtering as for predecessors/successors
                child_element = id_to_element.get(child_id)
                if (child_element and
                        child_element["element_type"] != "root" and
                        child_element.get("content_preview") and
                        not self._is_empty_container(child_element)):
                    context_ids.add(child_id)
                    children_added += 1
                    if children_added >= self.child_count:
                        break

        # Convert IDs to elements
        context_elements = []
        for context_id in context_ids:
            if context_id in id_to_element:
                context_elements.append(id_to_element[context_id])

        return context_elements

    @staticmethod
    def _is_empty_container(element: Dict[str, Any]) -> bool:
        """
        Check if an element is just an empty container (like a div with no text).

        Args:
            element: The element to check

        Returns:
            True if the element is an empty container, False otherwise
        """
        # Consider these element types as potential empty containers
        container_types = ["div", "span", "article", "section", "nav", "aside"]

        # Check if it's a container type
        if element["element_type"] in container_types:
            # Check if it has no meaningful content
            content = element.get("content_preview", "").strip()
            return not content or content in ["", "..."]

        return False
