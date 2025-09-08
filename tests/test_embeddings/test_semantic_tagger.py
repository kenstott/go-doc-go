"""
Tests for semantic tagging in contextual embeddings.
"""

import pytest
from unittest.mock import Mock, MagicMock
from go_doc_go.embeddings.semantic_tagger import SemanticTagger, ContextRole
from go_doc_go.embeddings.contextual_embedding import ContextualEmbeddingGenerator
from go_doc_go.embeddings.token_aware_contextual import TokenAwareContextualEmbedding


@pytest.mark.unit
class TestSemanticTagger:
    """Unit tests for SemanticTagger class."""
    
    def test_generate_basic_tag(self):
        """Test basic tag generation."""
        tagger = SemanticTagger()
        
        element = {
            "element_type": "paragraph",
            "element_id": "para_123",
            "metadata": {}
        }
        
        tag = tagger.generate_tag(element, context_role=ContextRole.PARENT)
        
        assert tag.startswith("[PARENT:")
        assert "paragraph" in tag
        assert "para_123" in tag
        assert tag.endswith("]")
    
    def test_generate_tag_with_metadata(self):
        """Test tag generation with metadata."""
        tagger = SemanticTagger(include_metadata=True)
        
        element = {
            "element_type": "table",
            "element_id": "table_revenue",
            "metadata": {
                "row_count": 10,
                "column_count": 4,
                "page_number": 15
            }
        }
        
        tag = tagger.generate_tag(element, context_role=ContextRole.CHILD)
        
        assert tag.startswith("[CHILD:table:")
        assert "table_revenue" in tag
        assert "rows=10" in tag
        assert "cols=4" in tag
        assert "page=15" in tag
    
    def test_tag_truncation(self):
        """Test that long tags are truncated."""
        tagger = SemanticTagger(max_tag_length=50)
        
        element = {
            "element_type": "paragraph",
            "element_id": "this_is_a_very_long_element_id_that_should_be_truncated",
            "metadata": {
                "section_title": "This is an extremely long section title that goes on and on"
            }
        }
        
        tag = tagger.generate_tag(element)
        
        assert len(tag) <= 50
        assert tag.endswith("...]")
    
    def test_parse_tag(self):
        """Test parsing a tag back to components."""
        tagger = SemanticTagger()
        
        tag = "[PARENT:Section:accounting_policies:level=2,page=10]"
        parsed = tagger.parse_tag(tag)
        
        assert parsed["role"] == "PARENT"
        assert parsed["element_type"] == "Section"
        assert parsed["element_id"] == "accounting_policies"
        assert parsed["metadata"]["level"] == 2
        assert parsed["metadata"]["page"] == 10
    
    def test_relationship_mapping(self):
        """Test relationship type to context role mapping."""
        tagger = SemanticTagger()
        
        element = {"element_type": "text", "element_id": "text_1"}
        
        # Test different relationship types
        tag1 = tagger.generate_tag(element, relationship_type="contains")
        assert tag1.startswith("[PARENT:")
        
        tag2 = tagger.generate_tag(element, relationship_type="contained_by")
        assert tag2.startswith("[CHILD:")
        
        tag3 = tagger.generate_tag(element, relationship_type="references")
        assert tag3.startswith("[REFERENCES:")
    
    def test_format_tagged_content(self):
        """Test formatting tagged content."""
        tagger = SemanticTagger()
        
        tag = "[PARENT:Section:intro]"
        content = "This is the introduction section."
        
        formatted = tagger.format_tagged_content(tag, content)
        
        assert formatted == "[PARENT:Section:intro] This is the introduction section."
    
    def test_extract_tags_from_text(self):
        """Test extracting tags from text."""
        tagger = SemanticTagger()
        
        text = """[PARENT:Section:intro] Introduction text.
        Some regular text here.
        [CHILD:Table:revenue:rows=10] Revenue data.
        [SIBLING:Paragraph:next] Next paragraph."""
        
        tags = tagger.extract_tags_from_text(text)
        
        assert len(tags) == 3
        assert "[PARENT:Section:intro]" in tags
        assert "[CHILD:Table:revenue:rows=10]" in tags
        assert "[SIBLING:Paragraph:next]" in tags


@pytest.mark.unit
class TestContextualEmbeddingWithTags:
    """Test ContextualEmbedding with semantic tags."""
    
    @pytest.fixture
    def mock_base_generator(self):
        """Create mock base generator."""
        mock = Mock()
        mock.generate.return_value = [0.1] * 768  # Mock embedding
        return mock
    
    @pytest.fixture
    def mock_config(self):
        """Create mock config."""
        config = Mock()
        config.get.return_value = {}
        return config
    
    def test_contextual_embedding_with_semantic_tags(self, mock_config, mock_base_generator):
        """Test that semantic tags are used when enabled."""
        generator = ContextualEmbeddingGenerator(
            _config=mock_config,
            base_generator=mock_base_generator,
            use_semantic_tags=True
        )
        
        element_text = "Main content text"
        parent_texts = ["Parent content"]
        sibling_texts = ["Sibling content"]
        child_texts = ["Child content"]
        
        element_metadata = {
            "element_type": "paragraph",
            "element_id": "para_main"
        }
        parent_metadata = [{
            "element_type": "section",
            "element_id": "section_1"
        }]
        sibling_metadata = [{
            "element_type": "paragraph",
            "element_id": "para_prev"
        }]
        child_metadata = [{
            "element_type": "list",
            "element_id": "list_1"
        }]
        
        result = generator.build_structured_context(
            element_text,
            parent_texts,
            sibling_texts,
            child_texts,
            element_metadata,
            parent_metadata,
            sibling_metadata,
            child_metadata
        )
        
        # Check that semantic tags are present
        assert "[PARENT:" in result
        assert "[SIBLING:" in result or "[PRECEDING:" in result or "[FOLLOWING:" in result
        assert "[CHILD:" in result
        assert "[MAIN:" in result  # Main element tag (changed from RELATED)
        
        # Check that old format is not present
        assert "=== Parent Context ===" not in result
        assert "=== Main Content ===" not in result
    
    def test_contextual_embedding_without_semantic_tags(self, mock_config, mock_base_generator):
        """Test that generic headers are used when semantic tags disabled."""
        generator = ContextualEmbeddingGenerator(
            _config=mock_config,
            base_generator=mock_base_generator,
            use_semantic_tags=False
        )
        
        element_text = "Main content text"
        parent_texts = ["Parent content"]
        sibling_texts = ["Sibling content"]
        child_texts = ["Child content"]
        
        result = generator.build_structured_context(
            element_text,
            parent_texts,
            sibling_texts,
            child_texts
        )
        
        # Check that old format is present
        assert "=== Parent Context ===" in result
        assert "=== Sibling Context ===" in result
        assert "=== Child Context ===" in result
        assert "=== Main Content ===" in result
        
        # Check that semantic tags are not present
        assert "[PARENT:" not in result
        assert "[CHILD:" not in result


@pytest.mark.unit
class TestTokenAwareContextualWithTags:
    """Test TokenAwareContextualEmbedding with semantic tags."""
    
    @pytest.fixture
    def mock_base_generator(self):
        """Create mock base generator."""
        mock = Mock()
        mock.generate.return_value = [0.1] * 768
        return mock
    
    def test_token_aware_with_semantic_tags(self, mock_base_generator):
        """Test token-aware embedding with semantic tags."""
        embedder = TokenAwareContextualEmbedding(
            base_generator=mock_base_generator,
            max_tokens=500,
            use_semantic_tags=True
        )
        
        element_text = "Main content that is important"
        parent_texts = ["Parent section content"]
        sibling_texts = ["Previous paragraph", "Next paragraph"]
        child_texts = ["Child item 1", "Child item 2"]
        
        element_metadata = {"element_type": "paragraph", "element_id": "p1"}
        parent_metadata = [{"element_type": "section", "element_id": "s1"}]
        sibling_metadata = [
            {"element_type": "paragraph", "element_id": "p0"},
            {"element_type": "paragraph", "element_id": "p2"}
        ]
        child_metadata = [
            {"element_type": "list_item", "element_id": "li1"},
            {"element_type": "list_item", "element_id": "li2"}
        ]
        
        result = embedder.build_context_with_budget(
            element_text,
            parent_texts,
            sibling_texts,
            child_texts,
            element_metadata,
            parent_metadata,
            sibling_metadata,
            child_metadata
        )
        
        # Verify semantic tags are present
        assert "[PARENT:section:" in result
        assert "[SIBLING:paragraph:" in result
        assert "[CHILD:list_item:" in result
        assert "[MAIN:paragraph:" in result  # Main element uses MAIN tag
    
    def test_token_budget_with_tags(self, mock_base_generator):
        """Test that token budget accounts for tag overhead."""
        embedder = TokenAwareContextualEmbedding(
            base_generator=mock_base_generator,
            max_tokens=200,  # Small budget to test truncation
            use_semantic_tags=True
        )
        
        # Create long texts that will exceed budget
        element_text = "Main " * 50  # Long main text
        parent_texts = ["Parent " * 50]
        sibling_texts = ["Sibling " * 50]
        child_texts = ["Child " * 50]
        
        metadata = {"element_type": "text", "element_id": "t1"}
        metadata_list = [metadata]
        
        result = embedder.build_context_with_budget(
            element_text,
            parent_texts,
            sibling_texts,
            child_texts,
            metadata,
            metadata_list,
            metadata_list,
            metadata_list
        )
        
        # Verify result is within token budget
        token_count = embedder.count_tokens(result)
        assert token_count <= embedder.safe_max_tokens
        
        # Verify tags are still present despite truncation
        assert "[" in result and "]" in result


@pytest.mark.integration
class TestSemanticTaggingIntegration:
    """Integration tests for semantic tagging with real parsers."""
    
    def test_generate_context_tags(self):
        """Test generating tags for different context elements."""
        tagger = SemanticTagger()
        
        context_elements = {
            "parents": [
                {"element_type": "section", "element_id": "sec1", "content_preview": "Section 1"},
                {"element_type": "chapter", "element_id": "ch1", "content_preview": "Chapter 1"}
            ],
            "siblings": [
                {"element_type": "paragraph", "element_id": "p1", "content_preview": "Para 1"},
                {"element_type": "paragraph", "element_id": "p2", "content_preview": "Para 2"}
            ],
            "children": [
                {"element_type": "list", "element_id": "l1", "content_preview": "List 1"}
            ],
            "element": {"document_position": 5}
        }
        
        tagged = tagger.generate_context_tags(context_elements)
        
        assert "parents" in tagged
        assert len(tagged["parents"]) == 2
        assert tagged["parents"][0][0].startswith("[PARENT:")
        
        assert "siblings" in tagged
        assert len(tagged["siblings"]) == 2
        
        assert "children" in tagged
        assert len(tagged["children"]) == 1
        assert tagged["children"][0][0].startswith("[CHILD:")
    
    def test_adaptive_context_strategy(self):
        """Test adaptive context strategy selection."""
        from go_doc_go.embeddings.token_aware_contextual import AdaptiveContextStrategy
        
        strategy = AdaptiveContextStrategy()
        
        # Test small document strategy
        small_doc_stats = {"total_elements": 30, "max_depth": 3, "avg_siblings": 5}
        ratios = strategy.select_strategy({}, small_doc_stats)
        assert ratios["parents"] == 0.30  # More parent context for small docs
        
        # Test large document strategy
        large_doc_stats = {"total_elements": 2000, "max_depth": 4, "avg_siblings": 8}
        ratios = strategy.select_strategy({}, large_doc_stats)
        assert ratios["element"] == 0.50  # Focus on element for large docs
        
        # Test deep hierarchy strategy
        deep_doc_stats = {"total_elements": 500, "max_depth": 10, "avg_siblings": 3}
        ratios = strategy.select_strategy({}, deep_doc_stats)
        assert ratios["parents"] == 0.35  # More parent context for deep trees
        
        # Test flat structure strategy
        flat_doc_stats = {"total_elements": 400, "max_depth": 2, "avg_siblings": 20}
        ratios = strategy.select_strategy({}, flat_doc_stats)
        assert ratios["siblings"] == 0.35  # More sibling context for flat docs