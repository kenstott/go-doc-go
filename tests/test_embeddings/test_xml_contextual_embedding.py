"""
Unit tests for XMLContextualEmbeddingGenerator.
"""

import pytest
from unittest.mock import Mock, MagicMock, patch

from go_doc_go.embeddings.xml_contextual_embedding import XMLContextualEmbeddingGenerator
from go_doc_go.embeddings.xml_semantic_tagger import ContextRole


class TestXMLContextualEmbeddingGenerator:
    """Unit tests for XML contextual embedding generator."""
    
    @pytest.fixture
    def mock_config(self):
        """Mock configuration."""
        config = Mock()
        config.get = Mock(return_value=None)
        return config
    
    @pytest.fixture
    def mock_base_generator(self):
        """Mock base embedding generator."""
        generator = Mock()
        generator.generate.return_value = [0.1] * 384  # Mock 384-dimensional embedding
        generator.generate_batch.return_value = [[0.1] * 384, [0.2] * 384]
        generator.get_dimensions.return_value = 384
        generator.get_model_name.return_value = "mock-model"
        generator.clear_cache.return_value = None
        return generator
    
    @pytest.fixture
    def mock_tokenizer(self):
        """Mock tokenizer for consistent token counting."""
        with patch('go_doc_go.embeddings.contextual_embedding.TIKTOKEN_AVAILABLE', True):
            with patch('tiktoken.get_encoding') as mock_get_encoding:
                mock_tokenizer = Mock()
                # Simple token counting: ~1 token per 4 characters
                mock_tokenizer.encode.side_effect = lambda text: list(range(len(text) // 4 + 1))
                mock_get_encoding.return_value = mock_tokenizer
                yield mock_tokenizer
    
    @pytest.fixture
    def xml_generator(self, mock_config, mock_base_generator, mock_tokenizer):
        """Create XMLContextualEmbeddingGenerator instance."""
        return XMLContextualEmbeddingGenerator(
            _config=mock_config,
            base_generator=mock_base_generator,
            max_tokens=800,  # Reasonable limit for XML testing
            use_xml_tags=True,
            include_entities=True,
            include_strength=True
        )
    
    def test_initialization(self, mock_config, mock_base_generator):
        """Test XML generator initialization."""
        generator = XMLContextualEmbeddingGenerator(
            _config=mock_config,
            base_generator=mock_base_generator,
            use_xml_tags=True,
            include_entities=True,
            include_strength=True,
            max_entities_per_element=3
        )
        
        assert generator.use_xml_tags is True
        assert generator.include_entities is True
        assert generator.include_strength is True
        assert generator.max_entities_per_element == 3
        assert hasattr(generator, 'xml_tagger')
    
    def test_initialization_without_xml(self, mock_config, mock_base_generator):
        """Test initialization with XML disabled."""
        generator = XMLContextualEmbeddingGenerator(
            _config=mock_config,
            base_generator=mock_base_generator,
            use_xml_tags=False
        )
        
        assert generator.use_xml_tags is False
    
    def test_estimate_xml_overhead(self, xml_generator):
        """Test XML overhead estimation."""
        # With XML tags enabled
        overhead_5_elements = xml_generator.estimate_xml_overhead(5)
        overhead_10_elements = xml_generator.estimate_xml_overhead(10)
        
        # More elements should have more overhead
        assert overhead_10_elements > overhead_5_elements
        assert overhead_5_elements > 0
        
        # Test with XML disabled
        xml_generator.use_xml_tags = False
        assert xml_generator.estimate_xml_overhead(5) == 0
    
    def test_build_xml_element(self, xml_generator):
        """Test building a single XML element."""
        text = "This is test content with database and performance keywords."
        metadata = {
            "element_type": "paragraph",
            "element_id": "para_123",
            "metadata": {"page_number": 5}
        }
        
        xml_element = xml_generator._build_xml_element(
            text=text,
            metadata=metadata,
            context_role=ContextRole.MAIN,
            max_tokens=100
        )
        
        assert xml_element is not None
        assert xml_element.startswith('<element')
        assert 'role="main"' in xml_element
        assert 'type="paragraph"' in xml_element
        assert xml_element.endswith('</element>')
        assert "database" in xml_element.lower()
    
    def test_build_xml_element_token_limit(self, xml_generator):
        """Test XML element building respects token limits."""
        # Very long text
        long_text = "This is a very long text. " * 100
        metadata = {"element_type": "paragraph", "element_id": "test"}
        
        # Small token budget
        xml_element = xml_generator._build_xml_element(
            text=long_text,
            metadata=metadata,
            context_role=ContextRole.MAIN,
            max_tokens=50  # Very small budget
        )
        
        if xml_element:  # Might be None if budget too small
            tokens = xml_generator.count_tokens(xml_element)
            assert tokens <= 50
    
    def test_build_xml_element_insufficient_budget(self, xml_generator):
        """Test XML element building with insufficient budget."""
        text = "Test content"
        metadata = {"element_type": "paragraph", "element_id": "test"}
        
        # Budget smaller than XML overhead
        xml_element = xml_generator._build_xml_element(
            text=text,
            metadata=metadata,
            context_role=ContextRole.MAIN,
            max_tokens=5  # Too small
        )
        
        # Should return None when budget is insufficient
        assert xml_element is None
    
    def test_build_xml_context_elements(self, xml_generator):
        """Test building multiple XML context elements."""
        texts = [
            "First context element about revenue.",
            "Second context element about growth.",
            "Third context element about performance."
        ]
        metadata_list = [
            {"element_type": "paragraph", "element_id": f"para_{i}"}
            for i in range(len(texts))
        ]
        
        xml_elements = xml_generator._build_xml_context_elements(
            texts=texts,
            metadata_list=metadata_list,
            context_role=ContextRole.PARENT,
            budget=150
        )
        
        assert isinstance(xml_elements, list)
        assert len(xml_elements) > 0
        
        # All elements should be valid XML
        for element in xml_elements:
            assert element.startswith('<context')
            assert 'role="parent"' in element
            assert element.endswith('</context>')
    
    def test_build_xml_context_elements_budget_exhausted(self, xml_generator):
        """Test context element building when budget is exhausted."""
        texts = ["Text"] * 10  # Many short texts
        metadata_list = [{"element_type": "paragraph", "element_id": f"para_{i}"} for i in range(10)]
        
        # Very small budget
        xml_elements = xml_generator._build_xml_context_elements(
            texts=texts,
            metadata_list=metadata_list,
            context_role=ContextRole.SIBLING,
            budget=20  # Very small budget
        )
        
        # Should return fewer elements than input due to budget constraints
        assert len(xml_elements) < len(texts)
    
    def test_build_xml_structured_context_complete(self, xml_generator):
        """Test building complete XML structured context."""
        element_text = "Main element about quarterly revenue growth."
        parent_texts = ["Parent context about financial reporting."]
        sibling_texts = ["Sibling context about market analysis."]
        child_texts = ["Child context about detailed metrics."]
        
        element_metadata = {"element_type": "paragraph", "element_id": "main_para"}
        parent_metadata = [{"element_type": "section", "element_id": "parent_section"}]
        sibling_metadata = [{"element_type": "paragraph", "element_id": "sibling_para"}]
        child_metadata = [{"element_type": "list_item", "element_id": "child_item"}]
        
        xml_content = xml_generator.build_xml_structured_context(
            element_text=element_text,
            parent_texts=parent_texts,
            sibling_texts=sibling_texts,
            child_texts=child_texts,
            element_metadata=element_metadata,
            parent_metadata=parent_metadata,
            sibling_metadata=sibling_metadata,
            child_metadata=child_metadata,
            doc_type="pdf",
            doc_domain="finance",
            doc_id="doc_123"
        )
        
        assert xml_content.startswith('<document')
        assert 'type="pdf"' in xml_content
        assert 'domain="finance"' in xml_content
        assert xml_content.endswith('</document>')
        
        # Should contain all context elements
        assert 'role="main"' in xml_content
        assert 'role="parent"' in xml_content
        assert 'role="sibling"' in xml_content
        assert 'role="child"' in xml_content
        
        # Should contain the actual content
        assert "quarterly revenue growth" in xml_content
        assert "financial reporting" in xml_content
    
    def test_build_xml_structured_context_fallback(self, xml_generator):
        """Test fallback to bracket format when XML fails."""
        # Create a scenario where XML overhead exceeds budget
        xml_generator.safe_max_tokens = 10  # Impossibly small budget
        
        element_text = "Test content"
        parent_texts = ["Parent content"]
        
        with patch.object(xml_generator, 'build_structured_context') as mock_fallback:
            mock_fallback.return_value = "=== Fallback format ==="
            
            result = xml_generator.build_xml_structured_context(
                element_text=element_text,
                parent_texts=parent_texts,
                sibling_texts=[],
                child_texts=[]
            )
            
            # Should have called the fallback method
            mock_fallback.assert_called_once()
            assert result == "=== Fallback format ==="
    
    def test_emergency_xml_truncation(self, xml_generator):
        """Test emergency XML truncation preserves structure."""
        # Create XML content that exceeds token limit
        long_xml = """<document type="pdf">
            <element role="main" type="paragraph">Main content here</element>
            <context role="parent" type="section">Very long parent content that exceeds budget limits and needs to be truncated</context>
            <context role="sibling" type="paragraph">Another long sibling content that also exceeds the budget</context>
        </document>"""
        
        truncated = xml_generator._emergency_xml_truncation(long_xml, max_tokens=50)
        
        # Should still be valid XML
        assert xml_generator._validate_xml_structure(truncated)
        assert truncated.startswith('<document')
        assert truncated.endswith('</document>')
        
        # Should have fewer tokens than original
        original_tokens = xml_generator.count_tokens(long_xml)
        truncated_tokens = xml_generator.count_tokens(truncated)
        assert truncated_tokens < original_tokens
    
    def test_validate_xml_structure(self, xml_generator):
        """Test XML structure validation."""
        # Valid XML
        valid_xml = '<document><element role="main">Content</element></document>'
        assert xml_generator._validate_xml_structure(valid_xml) is True
        
        # Invalid XML
        invalid_xml = '<document><element role="main">Unclosed content</document>'
        assert xml_generator._validate_xml_structure(invalid_xml) is False
    
    def test_generate_with_xml_context(self, xml_generator, mock_base_generator):
        """Test generate method with XML context."""
        main_text = "Main content about database performance"
        context = ["Context about server configuration", "Context about query optimization"]
        
        result = xml_generator.generate(main_text, context)
        
        # Should call base generator with XML-structured content
        mock_base_generator.generate.assert_called_once()
        call_args = mock_base_generator.generate.call_args[0][0]
        
        # Generated content should be XML
        assert '<document' in call_args
        assert '</document>' in call_args
        assert 'role="main"' in call_args
        
        assert result == [0.1] * 384  # Mock embedding
    
    def test_generate_without_context(self, xml_generator, mock_base_generator):
        """Test generate method without context."""
        main_text = "Main content"
        
        result = xml_generator.generate(main_text, None)
        
        # Should use parent implementation
        mock_base_generator.generate.assert_called_once_with(main_text)
        assert result == [0.1] * 384
    
    def test_generate_with_xml_disabled(self, mock_config, mock_base_generator, mock_tokenizer):
        """Test generate method with XML disabled."""
        generator = XMLContextualEmbeddingGenerator(
            _config=mock_config,
            base_generator=mock_base_generator,
            use_xml_tags=False
        )
        
        main_text = "Main content"
        context = ["Context"]
        
        result = generator.generate(main_text, context)
        
        # Should use parent implementation
        mock_base_generator.generate.assert_called_once()
        assert result == [0.1] * 384
    
    def test_get_model_name(self, xml_generator):
        """Test model name generation."""
        # With XML enabled
        xml_generator.use_xml_tags = True
        assert xml_generator.get_model_name() == "xml-contextual-mock-model"
        
        # With XML disabled
        xml_generator.use_xml_tags = False
        assert xml_generator.get_model_name() == "contextual-mock-model"
    
    def test_build_simple_xml_context(self, xml_generator):
        """Test building simple XML context without metadata."""
        main_text = "Main content about revenue"
        context_texts = ["Context about growth", "Context about performance"]
        
        xml_context = xml_generator._build_simple_xml_context(main_text, context_texts)
        
        assert xml_context.startswith('<document')
        assert xml_context.endswith('</document>')
        assert 'role="main"' in xml_context
        assert 'role="related"' in xml_context
        assert "revenue" in xml_context
        assert "growth" in xml_context
    
    def test_build_simple_xml_context_insufficient_budget(self, xml_generator):
        """Test simple XML context with insufficient budget."""
        xml_generator.safe_max_tokens = 5  # Too small for XML
        
        with patch.object(xml_generator, '_combine_text_with_context') as mock_combine:
            mock_combine.return_value = "Combined text"
            
            result = xml_generator._build_simple_xml_context("Main", ["Context"])
            
            # Should fall back to combining text
            mock_combine.assert_called_once_with("Main", ["Context"])
            assert result == "Combined text"