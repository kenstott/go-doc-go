"""
XML-aware contextual embedding generator.
Extends the base contextual embedding with XML-style tagging and structure-aware token management.
"""

import logging
import xml.etree.ElementTree as ET
from typing import Optional, List, Dict, Any, Tuple

from .contextual_embedding import ContextualEmbeddingGenerator
from .xml_semantic_tagger import XMLSemanticTagger, ContextRole
from ..adapter import create_content_resolver

logger = logging.getLogger(__name__)


class XMLContextualEmbeddingGenerator(ContextualEmbeddingGenerator):
    """
    XML-aware contextual embedding generator.
    
    Creates XML-structured context that aligns with LLM training patterns
    while maintaining proper token management and valid XML structure.
    """

    def __init__(self,
                 _config,
                 base_generator,
                 window_size: int = 3,
                 overlap_size: int = 1,
                 predecessor_count: int = 1,
                 successor_count: int = 1,
                 ancestor_depth: int = 1,
                 child_count: int = 1,
                 max_tokens: int = 16384,
                 tokenizer_model: str = "cl100k_base",
                 use_xml_tags: bool = True,
                 include_entities: bool = True,
                 include_strength: bool = True,
                 max_entities_per_element: int = 5):
        """
        Initialize XML-aware contextual embedding generator.
        
        Args:
            _config: Configuration object
            base_generator: Base embedding generator
            window_size: Number of elements in context window
            overlap_size: Number of elements to overlap between windows
            predecessor_count: Number of preceding elements to include
            successor_count: Number of following elements to include
            ancestor_depth: Number of ancestral levels to include
            child_count: Number of children to include
            max_tokens: Maximum tokens for embedding model
            tokenizer_model: Tokenizer model to use
            use_xml_tags: Whether to use XML-style tags
            include_entities: Whether to include entity extraction
            include_strength: Whether to include relationship strength
            max_entities_per_element: Maximum entities per element
        """
        super().__init__(
            _config=_config,
            base_generator=base_generator,
            window_size=window_size,
            overlap_size=overlap_size,
            predecessor_count=predecessor_count,
            successor_count=successor_count,
            ancestor_depth=ancestor_depth,
            child_count=child_count,
            max_tokens=max_tokens,
            tokenizer_model=tokenizer_model,
            use_semantic_tags=use_xml_tags
        )
        
        # XML-specific configuration
        self.use_xml_tags = use_xml_tags
        self.include_entities = include_entities
        self.include_strength = include_strength
        self.max_entities_per_element = max_entities_per_element
        
        # Initialize XML semantic tagger
        if self.use_xml_tags:
            self.xml_tagger = XMLSemanticTagger(
                include_metadata=True,
                include_entities=include_entities,
                include_strength=include_strength,
                max_entities=max_entities_per_element
            )
        
        # XML overhead estimation (opening/closing tags, attributes)
        self.xml_overhead_per_element = 50  # Average XML tag overhead in tokens
    
    def estimate_xml_overhead(self, num_elements: int) -> int:
        """
        Estimate the token overhead for XML structure.
        
        Args:
            num_elements: Number of XML elements
            
        Returns:
            Estimated token overhead
        """
        if not self.use_xml_tags:
            return 0
        
        # Document wrapper: <document>...</document>
        document_overhead = 10
        
        # Per-element overhead for opening/closing tags with attributes
        element_overhead = num_elements * self.xml_overhead_per_element
        
        return document_overhead + element_overhead
    
    def build_xml_structured_context(self,
                                   element_text: str,
                                   parent_texts: List[str],
                                   sibling_texts: List[str],
                                   child_texts: List[str],
                                   element_metadata: Optional[Dict[str, Any]] = None,
                                   parent_metadata: Optional[List[Dict[str, Any]]] = None,
                                   sibling_metadata: Optional[List[Dict[str, Any]]] = None,
                                   child_metadata: Optional[List[Dict[str, Any]]] = None,
                                   cross_doc_texts: Optional[List[str]] = None,
                                   cross_doc_metadata: Optional[List[Dict[str, Any]]] = None,
                                   doc_type: str = "unknown",
                                   doc_domain: str = "",
                                   doc_id: str = "") -> str:
        """
        Build XML-structured context with proper token management.
        
        Args:
            element_text: Main element text
            parent_texts: Parent context texts
            sibling_texts: Sibling context texts  
            child_texts: Child context texts
            element_metadata: Metadata for main element
            parent_metadata: Metadata for parent elements
            sibling_metadata: Metadata for sibling elements
            child_metadata: Metadata for child elements
            cross_doc_texts: Cross-document context texts
            cross_doc_metadata: Metadata for cross-document elements
            doc_type: Document type (pdf, docx, etc.)
            doc_domain: Document domain/category
            doc_id: Document identifier
            
        Returns:
            XML-structured context within token limits
        """
        if not self.use_xml_tags:
            # Fall back to original method
            return self.build_structured_context(
                element_text, parent_texts, sibling_texts, child_texts,
                element_metadata, parent_metadata, sibling_metadata, child_metadata
            )
        
        # Estimate total number of elements (including cross-document)
        cross_doc_count = len(cross_doc_texts) if cross_doc_texts else 0
        total_elements = 1 + len(parent_texts) + len(sibling_texts) + len(child_texts) + cross_doc_count
        
        # Reserve tokens for XML overhead
        xml_overhead = self.estimate_xml_overhead(total_elements)
        content_budget = self.safe_max_tokens - xml_overhead
        
        if content_budget <= 0:
            logger.warning("XML overhead exceeds token budget, falling back to simple format")
            return self.build_structured_context(
                element_text, parent_texts, sibling_texts, child_texts,
                element_metadata, parent_metadata, sibling_metadata, child_metadata
            )
        
        # Calculate individual content budgets (excluding XML overhead)
        element_budget = int(content_budget * self.token_ratios["element"])
        parent_budget = int(content_budget * self.token_ratios["parents"])
        sibling_budget = int(content_budget * self.token_ratios["siblings"])
        child_budget = int(content_budget * self.token_ratios["children"])
        
        # Reserve 20% of budget for cross-document context if available
        # XML format requires more overhead, so cross-document needs larger allocation
        cross_doc_budget = 0
        if cross_doc_texts:
            cross_doc_budget = int(content_budget * 0.2)  # 20% for cross-document
            # Reduce other budgets proportionally to make room
            reduction_factor = 0.8
            element_budget = int(element_budget * reduction_factor)
            parent_budget = int(parent_budget * reduction_factor)
            sibling_budget = int(sibling_budget * reduction_factor)
            child_budget = int(child_budget * reduction_factor)
        
        logger.debug(f"XML context budgets: element={element_budget}, parent={parent_budget}, "
                    f"sibling={sibling_budget}, child={child_budget}, cross_doc={cross_doc_budget}, overhead={xml_overhead}")
        
        # Build XML elements with budget management
        xml_elements = []
        
        # Add parent context elements
        if parent_texts and parent_budget > 0:
            parent_elements = self._build_xml_context_elements(
                parent_texts, parent_metadata or [], ContextRole.PARENT, parent_budget
            )
            xml_elements.extend(parent_elements)
        
        # Add main element
        main_element_metadata = element_metadata or {}
        main_element = self._build_xml_element(
            element_text, main_element_metadata, ContextRole.MAIN, element_budget
        )
        if main_element:
            xml_elements.append(main_element)
        
        # Add sibling context elements
        if sibling_texts and sibling_budget > 0:
            sibling_elements = self._build_xml_context_elements(
                sibling_texts, sibling_metadata or [], ContextRole.SIBLING, sibling_budget
            )
            xml_elements.extend(sibling_elements)
        
        # Add child context elements
        if child_texts and child_budget > 0:
            child_elements = self._build_xml_context_elements(
                child_texts, child_metadata or [], ContextRole.CHILD, child_budget
            )
            xml_elements.extend(child_elements)
        
        # Add cross-document context elements
        if cross_doc_texts and cross_doc_budget > 0:
            cross_doc_elements = self._build_xml_context_elements(
                cross_doc_texts, cross_doc_metadata or [], ContextRole.RELATED, cross_doc_budget
            )
            xml_elements.extend(cross_doc_elements)
        
        # Wrap in document tags
        doc_opening, doc_closing = self.xml_tagger.generate_document_wrapper(
            doc_type=doc_type, domain=doc_domain, doc_id=doc_id
        )
        
        # Combine all XML elements
        xml_content_parts = [doc_opening] + xml_elements + [doc_closing]
        xml_content = "\n".join(xml_content_parts)
        
        # Final validation and token check
        total_tokens = self.count_tokens(xml_content)
        if total_tokens > self.safe_max_tokens:
            logger.warning(f"XML content {total_tokens} tokens exceeds limit {self.safe_max_tokens}")
            # Try emergency truncation while preserving XML structure
            xml_content = self._emergency_xml_truncation(xml_content, self.safe_max_tokens)
        
        # Validate XML structure
        if not self._validate_xml_structure(xml_content):
            logger.error("Generated XML content is invalid, falling back to bracket format")
            return self.build_structured_context(
                element_text, parent_texts, sibling_texts, child_texts,
                element_metadata, parent_metadata, sibling_metadata, child_metadata
            )
        
        logger.debug(f"Generated XML context: {total_tokens} tokens")
        return xml_content
    
    def _build_xml_context_elements(self,
                                  texts: List[str],
                                  metadata_list: List[Dict[str, Any]],
                                  context_role: ContextRole,
                                  budget: int) -> List[str]:
        """
        Build XML elements for context within token budget.
        
        Args:
            texts: List of content texts
            metadata_list: List of metadata dictionaries
            context_role: Role of these context elements
            budget: Token budget for all these elements
            
        Returns:
            List of XML element strings within budget
        """
        if budget <= 0:
            return []
        
        xml_elements = []
        used_tokens = 0
        
        for i, text in enumerate(texts):
            if used_tokens >= budget:
                break
            
            # Get metadata for this element
            element_metadata = metadata_list[i] if i < len(metadata_list) else {}
            
            # Calculate remaining budget
            remaining_budget = budget - used_tokens
            
            # Build XML element with remaining budget
            xml_element = self._build_xml_element(
                text, element_metadata, context_role, remaining_budget
            )
            
            if xml_element:
                element_tokens = self.count_tokens(xml_element)
                if used_tokens + element_tokens <= budget:
                    xml_elements.append(xml_element)
                    used_tokens += element_tokens
                else:
                    # Try to fit a truncated version
                    if remaining_budget > self.xml_overhead_per_element + 20:  # Minimum meaningful content
                        content_budget = remaining_budget - self.xml_overhead_per_element
                        truncated_text = self.truncate_to_tokens(text, content_budget)
                        truncated_element = self._build_xml_element(
                            truncated_text, element_metadata, context_role, remaining_budget
                        )
                        if truncated_element:
                            xml_elements.append(truncated_element)
                    break
        
        logger.debug(f"Built {len(xml_elements)}/{len(texts)} {context_role.value} elements, "
                    f"used {used_tokens}/{budget} tokens")
        
        return xml_elements
    
    def _build_xml_element(self,
                          text: str,
                          metadata: Dict[str, Any],
                          context_role: ContextRole,
                          max_tokens: int) -> Optional[str]:
        """
        Build a single XML element within token limit.
        
        Args:
            text: Element text content
            metadata: Element metadata
            context_role: Role of this element
            max_tokens: Maximum tokens for this element
            
        Returns:
            XML element string or None if cannot fit
        """
        if max_tokens <= self.xml_overhead_per_element:
            return None
        
        # Reserve tokens for XML structure
        content_budget = max_tokens - self.xml_overhead_per_element
        
        # Truncate content if necessary
        if self.count_tokens(text) > content_budget:
            text = self.truncate_to_tokens(text, content_budget)
        
        # Generate XML element
        element_distance = metadata.get("distance", 1)
        xml_element = self.xml_tagger.generate_xml_tag(
            element=metadata,
            content=text,
            context_role=context_role,
            element_distance=element_distance
        )
        
        # Verify it fits within budget
        if self.count_tokens(xml_element) <= max_tokens:
            return xml_element
        else:
            logger.warning(f"XML element exceeds budget: {self.count_tokens(xml_element)} > {max_tokens}")
            return None
    
    def _emergency_xml_truncation(self, xml_content: str, max_tokens: int) -> str:
        """
        Emergency truncation that preserves XML structure.
        
        Args:
            xml_content: XML content that exceeds token limit
            max_tokens: Maximum allowed tokens
            
        Returns:
            Truncated XML content with valid structure
        """
        try:
            # Parse XML to understand structure
            root = ET.fromstring(xml_content)
            
            # Get document wrapper attributes
            doc_attrs = root.attrib
            
            # Keep main element, remove context elements until under budget
            main_elements = []
            context_elements = []
            
            for child in root:
                if child.attrib.get("role") == "main":
                    main_elements.append(child)
                else:
                    context_elements.append(child)
            
            # Build new XML with main elements first
            new_root = ET.Element("document", doc_attrs)
            
            # Add main elements
            for element in main_elements:
                new_root.append(element)
            
            # Add context elements until budget exhausted
            current_xml = ET.tostring(new_root, encoding='unicode')
            current_tokens = self.count_tokens(current_xml)
            
            for element in context_elements:
                # Create temporary XML with this element added
                temp_root = ET.Element("document", doc_attrs)
                for existing in new_root:
                    temp_root.append(existing)
                temp_root.append(element)
                
                temp_xml = ET.tostring(temp_root, encoding='unicode')
                temp_tokens = self.count_tokens(temp_xml)
                
                if temp_tokens <= max_tokens:
                    new_root.append(element)
                    current_tokens = temp_tokens
                else:
                    break
            
            # Add comment about truncation
            if len(context_elements) > len(list(new_root)) - len(main_elements):
                comment = ET.Comment(" Additional context omitted due to token limits ")
                new_root.append(comment)
            
            final_xml = ET.tostring(new_root, encoding='unicode')
            logger.info(f"Emergency XML truncation: {self.count_tokens(xml_content)} -> "
                       f"{self.count_tokens(final_xml)} tokens")
            
            return final_xml
            
        except ET.ParseError as e:
            logger.error(f"Failed to parse XML for emergency truncation: {e}")
            # Fall back to simple truncation
            return self.truncate_to_tokens(xml_content, max_tokens)
    
    def _validate_xml_structure(self, xml_content: str) -> bool:
        """
        Validate XML structure.
        
        Args:
            xml_content: XML content to validate
            
        Returns:
            True if valid XML, False otherwise
        """
        try:
            ET.fromstring(xml_content)
            return True
        except ET.ParseError:
            return False
    
    def generate(self, text: str, context: Optional[List[str]] = None) -> List[float]:
        """
        Generate embedding with XML-structured context.
        
        Args:
            text: Main text to embed
            context: List of context texts (optional)
            
        Returns:
            Vector embedding
        """
        if not self.use_xml_tags or not context:
            # Use parent implementation
            return super().generate(text, context)
        
        # Build XML-structured context
        xml_context = self._build_simple_xml_context(text, context)
        
        # Generate embedding using base generator
        return self.base_generator.generate(xml_context)
    
    def _build_simple_xml_context(self, main_text: str, context_texts: List[str]) -> str:
        """
        Build simple XML context when metadata is not available.
        
        Args:
            main_text: Main element text
            context_texts: Context element texts
            
        Returns:
            XML-structured context
        """
        # Estimate budgets
        total_elements = 1 + len(context_texts)
        xml_overhead = self.estimate_xml_overhead(total_elements)
        content_budget = self.safe_max_tokens - xml_overhead
        
        if content_budget <= 0:
            return self._combine_text_with_context(main_text, context_texts)
        
        # Simple budget allocation
        main_budget = int(content_budget * 0.6)  # 60% for main element
        context_budget = content_budget - main_budget
        
        # Build XML elements
        xml_elements = []
        
        # Add main element
        main_metadata = {"element_type": "text", "element_id": "main"}
        main_element = self._build_xml_element(
            main_text, main_metadata, ContextRole.MAIN, main_budget
        )
        if main_element:
            xml_elements.append(main_element)
        
        # Add context elements
        if context_texts and context_budget > 0:
            context_budget_per_element = context_budget // len(context_texts)
            for i, context_text in enumerate(context_texts):
                context_metadata = {"element_type": "text", "element_id": f"context_{i}"}
                context_element = self._build_xml_element(
                    context_text, context_metadata, ContextRole.RELATED, context_budget_per_element
                )
                if context_element:
                    xml_elements.append(context_element)
        
        # Wrap in document
        doc_opening, doc_closing = self.xml_tagger.generate_document_wrapper("text")
        xml_parts = [doc_opening] + xml_elements + [doc_closing]
        
        return "\n".join(xml_parts)
    
    def get_model_name(self) -> str:
        """Get embedding model name."""
        base_name = self.base_generator.get_model_name()
        if self.use_xml_tags:
            return f"xml-contextual-{base_name}"
        else:
            return f"contextual-{base_name}"
    
    def generate_from_elements(self, elements: List[Dict[str, Any]], db=None) -> Dict[str, List[float]]:
        """
        Generate XML contextual embeddings for document elements.
        
        Overrides parent to properly integrate cross-document relationships into XML structure.
        """
        if not self.use_xml_tags:
            # Use parent implementation for non-XML mode
            return super().generate_from_elements(elements, db)
        
        # Build element hierarchy
        hierarchy = self._build_element_hierarchy(elements)
        resolver = create_content_resolver(self._config)
        
        # Maximum content size for effective embedding
        max_words_for_embedding = 500
        
        # Generate embeddings with XML context
        embeddings = {}
        
        for element in elements:
            element_pk = element["element_pk"]
            
            # Get full text content using resolver
            content = resolver.resolve_content(element.get('content_location'), text=True)
            
            # Skip if no meaningful content
            if not content and not ContextualEmbeddingGenerator.is_number(content):
                continue
            
            # Check content length and truncate if necessary
            word_count = len(content.split())
            if word_count > max_words_for_embedding:
                if element["element_type"] == "root":
                    continue  # Skip root elements that are too large
                content = " ".join(content.split()[:max_words_for_embedding])
            
            # Get context elements (including cross-document if db provided)
            context_elements = self._get_context_elements(element, elements, hierarchy, db)
            
            # Separate contexts by type
            intra_doc_contexts = {"parent": [], "sibling": [], "child": []}
            cross_doc_contexts = []
            
            for ctx_element in context_elements:
                ctx_content = resolver.resolve_content(ctx_element.get('content_location'), text=True)
                if not ctx_content:
                    continue
                
                # Truncate context if too long
                ctx_words = len(ctx_content.split())
                if ctx_words > max_words_for_embedding:
                    ctx_content = " ".join(ctx_content.split()[:max_words_for_embedding])
                
                # Determine context type and categorize
                if ctx_element.get('_cross_document'):
                    # Cross-document context
                    cross_doc_contexts.append({
                        "text": ctx_content,
                        "metadata": ctx_element
                    })
                else:
                    # Intra-document context - categorize by relationship
                    element_id = element["element_id"]
                    ctx_id = ctx_element["element_id"]
                    
                    # Simple heuristic for context type based on hierarchy
                    if element.get("parent_id") == ctx_id:
                        intra_doc_contexts["parent"].append({"text": ctx_content, "metadata": ctx_element})
                    elif ctx_element.get("parent_id") == element_id:
                        intra_doc_contexts["child"].append({"text": ctx_content, "metadata": ctx_element})
                    else:
                        intra_doc_contexts["sibling"].append({"text": ctx_content, "metadata": ctx_element})
            
            # Build XML structured context
            xml_context = self.build_xml_structured_context(
                element_text=content,
                parent_texts=[ctx["text"] for ctx in intra_doc_contexts["parent"]],
                sibling_texts=[ctx["text"] for ctx in intra_doc_contexts["sibling"]],
                child_texts=[ctx["text"] for ctx in intra_doc_contexts["child"]],
                element_metadata=element,
                parent_metadata=[ctx["metadata"] for ctx in intra_doc_contexts["parent"]],
                sibling_metadata=[ctx["metadata"] for ctx in intra_doc_contexts["sibling"]],
                child_metadata=[ctx["metadata"] for ctx in intra_doc_contexts["child"]],
                cross_doc_texts=[ctx["text"] for ctx in cross_doc_contexts],
                cross_doc_metadata=[ctx["metadata"] for ctx in cross_doc_contexts],
                doc_type=element.get("doc_type", "unknown"),
                doc_id=element.get("doc_id", "")
            )
            
            # Generate embedding using base generator
            embedding = self.base_generator.generate(xml_context)
            embeddings[element_pk] = embedding
            
        return embeddings