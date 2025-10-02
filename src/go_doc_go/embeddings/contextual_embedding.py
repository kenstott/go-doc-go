from typing import Optional, List, Dict, Any, Tuple
import logging
import json

from .base import EmbeddingGenerator
# Semantic tagging removed - using simple text curation
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
                 predecessor_count: int = 1,
                 successor_count: int = 1,
                 max_tokens: int = 16384,
                 tokenizer_model: str = "cl100k_base",
                 # Legacy parameters - kept for backward compatibility but NOT USED
                 window_size: int = 3,  # LEGACY: Not used
                 overlap_size: int = 1,  # LEGACY: Not used
                 ancestor_depth: int = 1,  # LEGACY: Not used - always traverses to root
                 use_semantic_tags: bool = False):  # LEGACY: Always False
        """
        Initialize the contextual embedding generator.

        Args:
            _config: Configuration object
            base_generator: Base embedding generator
            predecessor_count: Number of preceding elements to include
            successor_count: Number of following elements to include
            max_tokens: Maximum tokens for context
            tokenizer_model: Tokenizer model to use
            window_size: LEGACY - NOT USED
            overlap_size: LEGACY - NOT USED
            ancestor_depth: LEGACY - NOT USED (always traverses to root)
        """
        super().__init__(_config)
        self.base_generator = base_generator
        self.predecessor_count = predecessor_count
        self.successor_count = successor_count

        # Legacy parameters - stored for compatibility but NOT USED
        self.window_size = window_size  # LEGACY: Not used
        self.overlap_size = overlap_size  # LEGACY: Not used
        self.ancestor_depth = ancestor_depth  # LEGACY: Not used

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

        # Simple text approach - no semantic tagging
        self.use_semantic_tags = False  # Always disabled

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
        import time
        t1 = time.time()
        combined_text = self._combine_text_with_context(text, context)
        self._time_text_combining = getattr(self, '_time_text_combining', 0) + (time.time() - t1)

        # Generate embedding for combined text
        t2 = time.time()
        result = self.base_generator.generate(combined_text)
        self._time_actual_embedding = getattr(self, '_time_actual_embedding', 0) + (time.time() - t2)

        return result

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
            parent_context = self._select_texts_within_budget(parent_texts, parent_budget, "Parent")
            if parent_context:
                contexts.append(parent_context)

        # Add sibling context
        if sibling_texts and sibling_budget > 0:
            sibling_context = self._select_texts_within_budget(sibling_texts, sibling_budget, "Sibling")
            if sibling_context:
                contexts.append(sibling_context)

        # Add child context
        if child_texts and child_budget > 0:
            child_context = self._select_texts_within_budget(child_texts, child_budget, "Child")
            if child_context:
                contexts.append(child_context)

        # Combine all parts - MAIN element first, then context in priority order
        # Add punctuation separators for semantic boundaries while maintaining token density
        all_parts = [element_processed] + contexts
        combined = ". ".join(part.rstrip('.') for part in all_parts if part.strip()) + "."

        # Final safety check
        total_tokens = self.count_tokens(combined)
        if total_tokens > self.safe_max_tokens:
            logger.error(f"Emergency: Combined context {total_tokens} still exceeds limit after budgeting")
            combined = self.truncate_to_tokens(combined, self.safe_max_tokens)

        # Debug logging to verify no headers are present
        if "=== Context ===" in combined or "=== Main Content ===" in combined:
            logger.error(f"WARNING: Found header markers in embedding text! First 200 chars: {combined[:200]}")
        else:
            logger.debug(f"Clean embedding text generated, {total_tokens} tokens, first 100 chars: {combined[:100]}")

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

        result = ". ".join(text.rstrip('.') for text in selected if text.strip())
        logger.debug(f"{context_type} context: {len(selected)}/{len(texts)} texts, {used_tokens}/{budget} tokens")

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

        # Combine with main content first, then context in priority order
        if selected_context:
            # Put main content first, then context elements separated by space/newline
            combined = f"{text}\n{chr(10).join(selected_context)}"
        else:
            combined = text

        # Final safety check
        total_tokens = self.count_tokens(combined)
        if total_tokens > self.safe_max_tokens:
            logger.warning(f"Final combined text {total_tokens} exceeds limit, applying emergency truncation")
            combined = self.truncate_to_tokens(combined, self.safe_max_tokens)

        # Debug logging to verify no headers are present
        if "=== Context ===" in combined or "=== Main Content ===" in combined:
            logger.error(f"WARNING: Found header markers in combined text! First 200 chars: {combined[:200]}")

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

    def _aggregate_table_cells(self, table_row_element: Dict[str, Any],
                              all_elements: List[Dict[str, Any]],
                              hierarchy: Dict[str, List[str]],
                              resolver) -> Dict[str, str]:
        """
        Aggregate table cell contents AND header contents for a table row.

        Args:
            table_row_element: The table row element
            all_elements: All document elements
            hierarchy: Element hierarchy mapping
            resolver: Content resolver

        Returns:
            Dictionary with 'cells' and 'headers' content
        """
        import re

        # Get child cell IDs
        row_id = table_row_element["element_id"]
        cell_ids = hierarchy.get(row_id, [])

        # Find cell elements
        id_to_element = {e["element_id"]: e for e in all_elements}
        cell_contents = []

        for cell_id in cell_ids:
            cell_element = id_to_element.get(cell_id)
            if cell_element and cell_element.get("element_type") in ("table_cell", "table_header"):
                # Get cell content
                cell_content = resolver.resolve_content(cell_element.get('content_location'), text=True)
                if cell_content:
                    # Strip HTML/XML tags
                    clean_content = re.sub(r'<[^>]+>', '', cell_content).strip()
                    # Skip pure numeric cells for token density
                    if clean_content and not self._is_pure_numeric(clean_content):
                        cell_contents.append(clean_content)

        # Find table headers
        header_contents = []

        # Find parent table element
        parent_table_id = None
        current_element = table_row_element
        max_depth = 5  # Prevent infinite loops
        depth = 0

        while current_element and depth < max_depth:
            parent_id = current_element.get("parent_id")
            if not parent_id:
                break
            parent_element = id_to_element.get(parent_id)
            if parent_element:
                if parent_element.get("element_type") == "table":
                    parent_table_id = parent_id
                    break
                current_element = parent_element
            depth += 1

        # If we found the table, get all header rows
        if parent_table_id:
            table_children = hierarchy.get(parent_table_id, [])
            for child_id in table_children:
                child_element = id_to_element.get(child_id)
                if child_element and child_element.get("element_type") == "table_header_row":
                    # Get all cells in this header row
                    header_row_children = hierarchy.get(child_id, [])
                    for header_cell_id in header_row_children:
                        header_cell = id_to_element.get(header_cell_id)
                        if header_cell and header_cell.get("element_type") in ("table_header", "table_cell"):
                            header_text = resolver.resolve_content(header_cell.get('content_location'), text=True)
                            if header_text:
                                # Strip HTML/XML tags
                                clean_header = re.sub(r'<[^>]+>', '', header_text).strip()
                                # Skip numeric headers (rare but possible)
                                if clean_header and not self._is_pure_numeric(clean_header):
                                    header_contents.append(clean_header)

        return {
            "cells": " ".join(cell_contents),
            "headers": " ".join(header_contents)
        }

    def _is_pure_numeric(self, content: str) -> bool:
        """
        Check if content is pure numeric (numbers, currencies, percentages) without dates.

        Args:
            content: Content to check

        Returns:
            True if content is pure numeric and should be skipped
        """
        import re

        if not content:
            return False

        # Remove whitespace for checking
        clean = content.strip()

        # Common date patterns - if it looks like a date, keep it
        date_patterns = [
            r'\d{1,2}[-/]\d{1,2}[-/]\d{2,4}',  # MM-DD-YYYY or similar
            r'\d{4}[-/]\d{1,2}[-/]\d{1,2}',    # YYYY-MM-DD
            r'(jan|feb|mar|apr|may|jun|jul|aug|sep|oct|nov|dec)',  # Month names
            r'(january|february|march|april|may|june|july|august|september|october|november|december)',
            r'(q1|q2|q3|q4)',  # Quarters
            r'\d{4}',  # Years (keep standalone years as they're often important)
        ]

        for pattern in date_patterns:
            if re.search(pattern, clean.lower()):
                return False  # It's a date, keep it

        # Pure numeric patterns to skip
        numeric_patterns = [
            r'^[\d,]+$',                    # Pure numbers with commas
            r'^[\d,]+\.\d+$',               # Decimals
            r'^\$?[\d,]+\.?\d*$',           # Currency
            r'^[\d,]+\.?\d*%$',             # Percentages
            r'^\([\d,]+\.?\d*\)$',          # Negative numbers in parentheses
            r'^-?[\d,]+\.?\d*$',            # Negative numbers
        ]

        for pattern in numeric_patterns:
            if re.match(pattern, clean):
                return True

        return False

    def generate_from_elements(self, elements: List[Dict[str, Any]], db=None) -> Dict[str, List[float]]:
        """
        Generate contextual embeddings for LEAF elements only.

        Only leaf elements (elements with no children) get embeddings, with full context including:
        - Full parent lineage to root
        - Predecessor and successor siblings
        - Cross-document relationships (if db provided)

        Skips:
        - Non-leaf elements (have children)
        - Pure numeric content
        - Elements without meaningful text content
        - Certain granular elements (table_cell, json_item, json_field)

        Args:
            elements: List of document elements to generate embeddings for
            db: Optional database connection for cross-document relationships
        """
        import time

        # Skip container elements - they have no meaningful content to embed
        # Their context is preserved through relationships when embedding content elements

        # Build element hierarchy
        hierarchy = self._build_element_hierarchy(elements)
        resolver = create_content_resolver(self._config)

        # Define maximum content size for effective embedding (approximate word count)
        max_words_for_embedding = 500

        # Batch processing settings
        BATCH_SIZE = 32  # Process 32 elements at a time

        # Generate embeddings with context
        embeddings = {}

        # Timing stats
        total_elements = len(elements)
        skipped_elements = 0
        processed = 0
        time_context_building = 0
        time_text_combining = 0
        time_embedding_gen = 0
        time_content_resolution = 0
        content_resolutions = 0
        start_time = time.time()

        # Batch processing arrays
        batch_element_ids = []
        batch_combined_texts = []
        child_exceptions = {"table_header_row", "table_row"}

        for element in elements:
            # Safety check: ensure element is a dictionary
            if not isinstance(element, dict):
                skipped_elements += 1
                logger.warning(f"Skipping invalid element (not a dict): {type(element)} - {element}")
                continue

            # Only process leaf elements (elements with no children)
            # Container elements get their representation through their leaf children
            element_id = element.get("element_id")
            if not element_id:
                skipped_elements += 1
                logger.warning(f"Skipping element without element_id: {element}")
                continue

            # Also skip certain elements that should use their parent
            if element.get("element_type") in {'table_cell', 'json_item', 'json_field'}:
                skipped_elements += 1
                continue

            # Also skip certain elements that are always containers
            if element.get("element_type") in {'table'}:
                skipped_elements += 1
                continue

            # Check if this element has any children (with safety check)
            has_children = any(
                isinstance(e, dict) and e.get("parent_id") == element_id
                for e in elements
            ) and element.get("element_type") not in child_exceptions
            if has_children:
                skipped_elements += 1
                logger.debug(f"Skipping non-leaf element: {element.get('element_type')} (has children)")
                continue

            # Skip XML root elements specifically (they have path="/")
            content_location = element.get('content_location', {})
            if isinstance(content_location, str):
                try:
                    content_location = json.loads(content_location)
                except (json.JSONDecodeError, TypeError):
                    content_location = {}

            if content_location.get('path') == '/' or 'xml_root_' in element.get('element_id', ''):
                skipped_elements += 1
                logger.debug(f"Skipping XML root element: {element.get('element_id')}")
                continue


            # Get full text content for all elements using the resolver
            t0 = time.time()
            content = resolver.resolve_content(element.get('content_location'), text=True)

            # Special handling for table rows - aggregate their cells AND headers
            if element.get("element_type") == "table_row":
                table_data = self._aggregate_table_cells(element, elements, hierarchy, resolver)
                if table_data:
                    cells_text = table_data.get("cells", "")
                    headers_text = table_data.get("headers", "")

                    # Combine cells and headers with punctuation separator for token density
                    if cells_text and headers_text:
                        # Format: "row_content. header_content."
                        aggregated_content = f"{cells_text}. {headers_text}."
                    elif cells_text:
                        aggregated_content = cells_text
                    elif headers_text:
                        aggregated_content = headers_text
                    else:
                        # Skip if no meaningful content
                        continue

                    # Combine with any existing content
                    content = aggregated_content

            time_content_resolution += (time.time() - t0)
            content_resolutions += 1

            # Skip if no meaningful content
            if not content:
                continue

            # Skip empty table rows (content is just "Row N")
            if element.get("element_type") == "table_row" and content.strip().startswith("Row ") and content.strip()[4:].isdigit():
                skipped_elements += 1
                logger.debug(f"Skipping empty table row: {content}")
                continue

            # Skip single-character elements (likely labels, bullets, or structural markers)
            if len(content.strip()) <= 1:
                skipped_elements += 1
                logger.debug(f"Skipping single-character element: {element.get('element_type')} - '{content}'")
                continue

            # Skip pure numeric values (but keep dates)
            # This especially applies to XML elements that are purely numeric
            if self._is_pure_numeric(content):
                skipped_elements += 1
                logger.debug(f"Skipping pure numeric element: {element.get('element_type')} - {content[:50]}")
                continue

            # Check content length
            word_count = len(content.split())
            if word_count > max_words_for_embedding:
                # For root elements, skip entirely
                if element["element_type"] == "root":
                    continue

                # For non-root elements, truncate to threshold
                content = " ".join(content.split()[:max_words_for_embedding])

            # TIMING: Context building phase
            t1 = time.time()

            # Get context elements
            context_elements = self._get_context_elements(element, elements, hierarchy, db)

            # Get context contents using the resolver for text
            context_contents = []
            for ctx_element in context_elements:
                if not ctx_element.get("element_type") in {'table'}:
                    t_ctx = time.time()
                    ctx_content = resolver.resolve_content(ctx_element.get('content_location'), text=True)
                    ctx_raw = resolver.resolve_content(ctx_element.get('content_location'), text=False)
                    time_content_resolution += (time.time() - t_ctx)
                    content_resolutions += 1

                    if ctx_content and not self.is_number(ctx_content):
                        # Also check size of context elements and truncate if needed
                        ctx_words = len(ctx_content.split())
                        if ctx_words > max_words_for_embedding:
                            ctx_content = " ".join(ctx_content.split()[:max_words_for_embedding])
                        context_contents.append(ctx_content)

            t2 = time.time()
            time_context_building += (t2 - t1)

            # TIMING: Text combination phase
            t_combine_start = time.time()
            combined_text = self._combine_text_with_context(content, context_contents).replace(' |', '')
            time_text_combining += (time.time() - t_combine_start)

            # Add to batch
            batch_element_ids.append(element_id)
            batch_combined_texts.append(combined_text)

            # Process batch when full or at end
            if len(batch_element_ids) >= BATCH_SIZE or element == elements[-1]:
                if batch_combined_texts:
                    # Generate embeddings for batch
                    t_batch_start = time.time()

                    # Check if base generator has batch support
                    if hasattr(self.base_generator, 'generate_batch'):
                        batch_embeddings = self.base_generator.generate_batch(batch_combined_texts)
                    else:
                        # Fallback to individual generation
                        batch_embeddings = [self.base_generator.generate(text) for text in batch_combined_texts]

                    time_embedding_gen += (time.time() - t_batch_start)

                    # Store results with both embedding and text
                    for element_id, embedding, combined_text in zip(batch_element_ids, batch_embeddings, batch_combined_texts):
                        embeddings[element_id] = {
                            'embedding': embedding,
                            'embedding_text': combined_text
                        }
                        processed += 1

                    # Clear batch
                    batch_element_ids = []
                    batch_combined_texts = []

            # Log progress every 100 elements
            if processed > 0 and processed % 100 == 0:
                elapsed = time.time() - start_time
                rate = processed / elapsed if elapsed > 0 else 0
                avg_resolutions_per_elem = content_resolutions / processed if processed > 0 else 0
                effective_total = total_elements - skipped_elements
                logger.info(f"Embedding progress: {processed}/{effective_total} ({processed*100/effective_total:.1f}%) - "
                          f"Skipped: {skipped_elements} - "
                          f"Rate: {rate:.1f} elem/s - "
                          f"Content resolution: {time_content_resolution:.1f}s ({time_content_resolution*100/elapsed:.1f}%) - "
                          f"Context: {time_context_building:.1f}s ({time_context_building*100/elapsed:.1f}%) - "
                          f"Embed: {time_embedding_gen:.1f}s ({time_embedding_gen*100/elapsed:.1f}%) - "
                          f"Resolutions: {content_resolutions} ({avg_resolutions_per_elem:.1f}/elem)")

        # Process any remaining batch
        if batch_combined_texts:
            t_batch_start = time.time()
            if hasattr(self.base_generator, 'generate_batch'):
                batch_embeddings = self.base_generator.generate_batch(batch_combined_texts)
            else:
                batch_embeddings = [self.base_generator.generate(text) for text in batch_combined_texts]
            time_embedding_gen += (time.time() - t_batch_start)

            for element_id, embedding, combined_text in zip(batch_element_ids, batch_embeddings, batch_combined_texts):
                embeddings[element_id] = {
                    'embedding': embedding,
                    'embedding_text': combined_text
                }
                processed += 1

        # Final timing report
        total_time = time.time() - start_time
        if processed > 0:
            avg_resolutions_per_elem = content_resolutions / processed if processed > 0 else 0

            logger.info(f"Embedding generation complete: {processed} elements processed, {skipped_elements} skipped in {total_time:.1f}s")
            logger.info(f"  Skipped: table_cell, json_item, json_field, root, body, pure numerics")
            logger.info(f"  Table rows enhanced with aggregated cell content")
            logger.info(f"  Batch processing: {BATCH_SIZE} elements per batch")
            logger.info(f"  Content resolution: {time_content_resolution:.1f}s ({time_content_resolution*100/total_time:.1f}%) - {content_resolutions} total, {avg_resolutions_per_elem:.1f}/elem")
            logger.info(f"  Context building: {time_context_building:.1f}s ({time_context_building*100/total_time:.1f}%)")
            logger.info(f"  Text combining: {time_text_combining:.1f}s ({time_text_combining*100/total_time:.1f}%)")
            logger.info(f"  Actual embedding: {time_embedding_gen:.1f}s ({time_embedding_gen*100/total_time:.1f}%)")
            logger.info(f"  Average per element: {total_time/processed:.3f}s")

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
                              hierarchy: Dict[str, List[str]],
                              db=None) -> List[Dict[str, Any]]:
        """
        Get context elements for an element.

        This includes:
        - Ancestors up to configured depth (skipping those with blank content)
        - Meaningful predecessors (elements that come before in document order)
        - Meaningful successors (elements that come after in document order)
        - A limited number of meaningful children (directly nested elements)
        - Cross-document relationships (if db is provided)

        Args:
            element: Element to get context for
            all_elements: List of all elements
            hierarchy: Element hierarchy
            db: Database connection for cross-document relationships (optional)

        Returns:
            List of context elements
        """
        element_id = element["element_id"]
        context_ids = list()

        # Build a mapping from element_id to element for quicker lookups
        id_to_element = {e["element_id"]: e for e in all_elements}

        # Collect ALL meaningful ancestors up to root (not limited by depth)
        # This ensures we never lose important topic context regardless of nesting
        current_element = element
        traversal_depth = 0
        ancestors_collected = []

        while current_element:
            parent_id = current_element.get("parent_id")
            if not parent_id:
                break  # Reached the top of the hierarchy

            # Find parent element to continue up the hierarchy
            parent_element = id_to_element.get(parent_id)
            if not parent_element:
                break  # Parent not found in elements

            # Collect parent if it has meaningful content (skip empty containers)
            # Check if parent is an XML root element
            content_loc = parent_element.get('content_location', {})
            if isinstance(content_loc, str):
                try:
                    content_loc = json.loads(content_loc)
                except (json.JSONDecodeError, TypeError):
                    content_loc = {}

            is_xml_root = ('xml_root_' in parent_element.get('element_id', ''))

            # Headers, sections with titles, etc. are crucial for context
            if (parent_element.get("content_preview") and
                    parent_element["element_type"] != "root" and
                    # not is_xml_root and
                    not self._is_structural_only_container(parent_element)):
                ancestors_collected.append(parent_id)

            # Move up to the next level, even if we skipped this parent
            current_element = parent_element
            traversal_depth += 1

            # Safety check - prevent infinite loops in malformed hierarchies
            if traversal_depth > 20:  # Very deep but reasonable limit
                logger.warning(f"Very deep hierarchy detected (>20 levels) for element {element_id}")
                break

        # Add ancestors to context (they'll be prioritized by token budget)
        ancestors_collected.reverse()
        context_ids = context_ids + ancestors_collected

        # Find meaningful predecessors and successors
        current_index = -1
        for i, e in enumerate(all_elements):
            if e["element_id"] == element_id:
                current_index = i
                break

        if current_index >= 0:
            # Get meaningful predecessors (elements that come before)
            # Stop immediately when we hit the parent - all earlier elements are ancestors
            pred_count = 0
            i = current_index - 1
            parent_id = element.get("parent_id")

            while i >= 0 and pred_count < self.predecessor_count:
                pred_element = all_elements[i]

                # If we hit the parent, stop - all earlier elements are ancestors
                if pred_element.get("element_id") == parent_id:
                    break

                # Check if predecessor is an XML root element
                pred_content_loc = pred_element.get('content_location', {})
                if isinstance(pred_content_loc, str):
                    try:
                        pred_content_loc = json.loads(pred_content_loc)
                    except (json.JSONDecodeError, TypeError):
                        pred_content_loc = {}

                is_pred_xml_root = (pred_content_loc.get('path') == '/' or
                                   'xml_root_' in pred_element.get('element_id', ''))

                # Skip elements that:
                # 1. Are root elements (including XML roots)
                # 2. Don't have content (empty content_preview)
                # 3. Are just container elements
                if (pred_element["element_type"] != "root" and
                        not is_pred_xml_root and
                        pred_element.get("content_preview") and
                        not self._is_structural_only_container(pred_element)):
                    context_ids.append(pred_element["element_id"])
                    pred_count += 1

                i -= 1

            # Get meaningful successors (elements that come after)
            succ_count = 0
            i = current_index + 1

            parent_ids = {element_id}
            while i < len(all_elements) and succ_count < self.successor_count:
                succ_element = all_elements[i]

                # Check if successor is an XML root element
                succ_content_loc = succ_element.get('content_location', {})
                if isinstance(succ_content_loc, str):
                    try:
                        succ_content_loc = json.loads(succ_content_loc)
                    except (json.JSONDecodeError, TypeError):
                        succ_content_loc = {}

                is_succ_xml_root = (succ_content_loc.get('path') == '/' or
                                   'xml_root_' in succ_element.get('element_id', ''))

                # Same filtering as for predecessors
                if (succ_element["element_type"] != "root" and
                        not is_succ_xml_root and
                        not succ_element.get("parent_id") in parent_ids and
                        succ_element.get("content_preview") and
                        not self._is_structural_only_container(succ_element)):
                    context_ids.append(succ_element["element_id"])
                    parent_ids.add(succ_element["element_id"])
                    succ_count += 1

                i += 1

        # Skip children context since we only process leaf elements
        # (Leaf elements by definition have no children)

        # Convert IDs to elements
        context_elements = []
        for context_id in context_ids:
            if context_id in id_to_element:
                context_elements.append(id_to_element[context_id])

        # Add cross-document relationships if database is available
        if db and hasattr(element, 'get') and element.get('element_id'):
            try:
                cross_doc_elements = self._get_cross_document_context(element, db)
                context_elements.extend(cross_doc_elements)
                logger.debug(f"Added {len(cross_doc_elements)} cross-document context elements for {element_id}")
            except Exception as e:
                logger.warning(f"Failed to retrieve cross-document context for {element_id}: {e}")

        return context_elements

    def _get_cross_document_context(self, element: Dict[str, Any], db) -> List[Dict[str, Any]]:
        """
        Get cross-document context elements for an element.

        Retrieves elements from other documents that have semantic relationships
        with this element, prioritizing by relationship strength and type.

        Args:
            element: Element to get cross-document context for
            db: Database connection for relationship queries

        Returns:
            List of cross-document context elements
        """
        cross_doc_elements = []
        element_id = element.get('element_id')
        if not element_id:
            return cross_doc_elements

        try:
            # Get outgoing relationships (where this element is the source)
            relationships = db.get_outgoing_relationships(element_id)

            # Filter for cross-document semantic relationships
            cross_doc_relationships = []
            for rel in relationships:
                metadata = rel.metadata or {}
                if metadata.get('cross_document', False) and rel.relationship_type == 'semantic_section':
                    cross_doc_relationships.append(rel)

            # Sort by similarity score (highest first)
            cross_doc_relationships.sort(
                key=lambda r: r.metadata.get('similarity_score', 0.0) if r.metadata else 0.0,
                reverse=True
            )

            # Limit cross-document context to avoid overwhelming the token budget
            max_cross_doc_context = 3  # Conservative limit

            for rel in cross_doc_relationships[:max_cross_doc_context]:
                try:
                    target_element = db.get_element(rel.target_reference)
                    if target_element and target_element.get('content_preview'):
                        # Add metadata indicating this is cross-document context
                        target_element = dict(target_element)  # Make a copy
                        target_element['_cross_document'] = True
                        target_element['_similarity_score'] = rel.metadata.get('similarity_score', 0.0)
                        target_element['_source_doc_id'] = rel.metadata.get('target_doc_id')
                        cross_doc_elements.append(target_element)

                except Exception as e:
                    logger.warning(f"Failed to retrieve cross-document element {rel.target_reference}: {e}")
                    continue

            logger.debug(f"Retrieved {len(cross_doc_elements)} cross-document context elements")

        except Exception as e:
            logger.warning(f"Error retrieving cross-document relationships: {e}")

        return cross_doc_elements

    @staticmethod
    def _is_structural_only_container(element: Dict[str, Any]) -> bool:
        """
        Check if an element is a structural container without its own text content.
        These containers provide document structure but don't have searchable content.

        Args:
            element: The element to check

        Returns:
            True if the element is a structural-only container, False otherwise
        """
        # Consider these element types as structural-only containers
        # They organize content but typically have no text of their own
        structural_types = ["table", "span", "article", "section", "nav", "aside", "tbody", "thead"]

        # Check if it's a structural type
        if element["element_type"] in structural_types:
            # Check if it has no meaningful content of its own
            content = element.get("content_preview", "").strip()
            return not content or content in ["", "..."] or len(content) < 10

        return False

    @staticmethod
    def _is_container_element(element: Dict[str, Any]) -> bool:
        """
        Check if an element is a container type that should not get its own embedding.
        Containers provide structure but no searchable content.

        Args:
            element: The element to check

        Returns:
            True if element is a container that should be skipped for embedding
        """
        # Comprehensive list of container types from ElementBase.is_container()
        container_types = {
            "root", "article", "section",
            "list", "table", "page", "xml_list", "xml_object",
            "presentation_body",
            "slide", "comments_container", "comments", "json_array",
            "json_object", "slide_masters", "slide_templates",
            "headers", "footers", "page_header", "page_footer", "body",
            # Excel containers
            "workbook", "sheet", "merged_cells", "data_tables"
        }
        return element.get("element_type", "").lower() in container_types
