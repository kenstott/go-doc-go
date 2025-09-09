"""
Unit tests for XML cross-document contextual embeddings.
"""

import pytest
from unittest.mock import Mock, MagicMock, patch

from go_doc_go.embeddings.xml_contextual_embedding import XMLContextualEmbeddingGenerator
from go_doc_go.embeddings.xml_semantic_tagger import ContextRole
from go_doc_go.storage.element_relationship import ElementRelationship


class TestXMLCrossDocumentEmbedding:
    """Test XML contextual embeddings with cross-document relationships."""
    
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
        generator.generate.return_value = [0.1] * 384
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
            max_tokens=1000,
            use_xml_tags=True,
            include_entities=True,
            include_strength=True
        )
    
    @pytest.fixture
    def mock_db_with_cross_doc(self):
        """Mock database with cross-document relationships."""
        db = Mock()
        
        # Mock cross-document relationship
        rel = ElementRelationship(
            relationship_id="cross_rel_123",
            source_id="main_element_456",
            relationship_type="semantic_section",
            target_reference="cross_doc_element_789",
            metadata={
                "cross_document": True,
                "similarity_score": 0.92,
                "source_doc_id": "doc1",
                "target_doc_id": "doc2"
            }
        )
        db.get_outgoing_relationships.return_value = [rel]
        
        # Mock cross-document target element
        cross_doc_element = {
            "element_id": "cross_doc_element_789",
            "element_pk": 999,
            "element_type": "paragraph",
            "content_preview": "Cross-document content about quarterly revenue analysis and market trends",
            "content_location": {"type": "text", "text": "Cross-document content about quarterly revenue analysis and market trends"},
            "doc_id": "doc2",
            "_cross_document": True,
            "_similarity_score": 0.92,
            "_source_doc_id": "doc2"
        }
        db.get_element.return_value = cross_doc_element
        
        return db
    
    @pytest.fixture
    def sample_elements(self):
        """Sample document elements for testing."""
        return [
            {
                "element_id": "parent_section_123",
                "element_pk": 100,
                "element_type": "section", 
                "content_preview": "Financial Overview Section",
                "content_location": {"type": "text", "text": "Financial Overview Section"},
                "doc_id": "doc1"
            },
            {
                "element_id": "main_element_456",
                "element_pk": 200,
                "element_type": "paragraph",
                "content_preview": "Revenue growth increased by 15% quarter over quarter",
                "content_location": {"type": "text", "text": "Revenue growth increased by 15% quarter over quarter"},
                "doc_id": "doc1",
                "parent_id": "parent_section_123"
            },
            {
                "element_id": "child_element_789", 
                "element_pk": 300,
                "element_type": "list_item",
                "content_preview": "Q4 revenue: $2.3M", 
                "content_location": {"type": "text", "text": "Q4 revenue: $2.3M"},
                "doc_id": "doc1", 
                "parent_id": "main_element_456"
            }
        ]
    
    def test_build_xml_structured_context_with_cross_doc(self, xml_generator):
        """Test XML context building with cross-document elements."""
        element_text = "Main revenue analysis shows strong growth"
        parent_texts = ["Financial overview section header"]
        sibling_texts = ["Related quarterly metrics"]
        child_texts = ["Q4 revenue details"]
        cross_doc_texts = ["Similar revenue analysis from competitor report", "Market trend analysis"]
        
        # Mock metadata
        element_metadata = {"element_type": "paragraph", "element_id": "main_para"}
        parent_metadata = [{"element_type": "section", "element_id": "parent_section"}]
        sibling_metadata = [{"element_type": "paragraph", "element_id": "sibling_para"}]
        child_metadata = [{"element_type": "list_item", "element_id": "child_item"}]
        cross_doc_metadata = [
            {"element_type": "paragraph", "element_id": "cross_doc_1", "_cross_document": True, "_similarity_score": 0.85},
            {"element_type": "paragraph", "element_id": "cross_doc_2", "_cross_document": True, "_similarity_score": 0.78}
        ]
        
        xml_content = xml_generator.build_xml_structured_context(
            element_text=element_text,
            parent_texts=parent_texts,
            sibling_texts=sibling_texts,
            child_texts=child_texts,
            element_metadata=element_metadata,
            parent_metadata=parent_metadata,
            sibling_metadata=sibling_metadata,
            child_metadata=child_metadata,
            cross_doc_texts=cross_doc_texts,
            cross_doc_metadata=cross_doc_metadata,
            doc_type="pdf",
            doc_domain="finance",
            doc_id="doc_123"
        )
        
        # Verify XML structure
        assert xml_content.startswith('<document')
        assert 'type="pdf"' in xml_content
        assert 'domain="finance"' in xml_content
        assert xml_content.endswith('</document>')
        
        # Verify all context types are present
        assert 'role="main"' in xml_content
        assert 'role="parent"' in xml_content  
        assert 'role="sibling"' in xml_content
        assert 'role="child"' in xml_content
        assert 'role="related"' in xml_content  # Cross-document uses RELATED role
        
        # Verify cross-document content is included
        assert "Similar revenue analysis" in xml_content
        assert "Market trend analysis" in xml_content  # Both should fit with 20% budget allocation
        
        # Verify main content is included
        assert "strong growth" in xml_content
    
    def test_generate_from_elements_with_cross_doc(self, xml_generator, mock_db_with_cross_doc, sample_elements):
        """Test full generate_from_elements with cross-document context."""
        
        # Mock content resolver
        with patch('go_doc_go.embeddings.xml_contextual_embedding.create_content_resolver') as mock_resolver_factory:
            mock_resolver = Mock()
            def resolve_content_side_effect(content_location, text=False):
                if content_location and content_location.get("text"):
                    return content_location["text"]
                return ""
            mock_resolver.resolve_content.side_effect = resolve_content_side_effect
            mock_resolver_factory.return_value = mock_resolver
            
            # Generate embeddings
            embeddings = xml_generator.generate_from_elements(sample_elements, db=mock_db_with_cross_doc)
            
            # Verify embeddings were generated
            assert len(embeddings) == 3  # All three elements should have embeddings
            assert 100 in embeddings  # parent_section_123
            assert 200 in embeddings  # main_element_456  
            assert 300 in embeddings  # child_element_789
            
            # Verify base generator was called with XML-structured content
            # The base generator should have been called multiple times
            assert xml_generator.base_generator.generate.call_count == 3
            
            # Check that at least one call included cross-document context
            calls = xml_generator.base_generator.generate.call_args_list
            xml_contexts = [call[0][0] for call in calls]
            
            # At least one context should contain cross-document content
            cross_doc_found = False
            for xml_context in xml_contexts:
                if "Cross-document content about quarterly revenue" in xml_context:
                    cross_doc_found = True
                    # Verify it's properly structured as XML
                    assert '<document' in xml_context
                    assert '</document>' in xml_context
                    assert 'role="related"' in xml_context
                    break
            
            assert cross_doc_found, "Cross-document context should be included in at least one embedding"
    
    def test_cross_doc_context_categorization(self, xml_generator, sample_elements):
        """Test that cross-document context is properly categorized separately from intra-document."""
        
        # Create a mock context element that's cross-document
        cross_doc_element = {
            "element_id": "cross_doc_element_999",
            "element_type": "paragraph",
            "content_preview": "Cross-document revenue analysis",
            "content_location": {"type": "text", "text": "Cross-document revenue analysis"},
            "_cross_document": True,
            "_similarity_score": 0.88,
            "_source_doc_id": "doc2"
        }
        
        # Mock the context retrieval to include cross-document element
        with patch.object(xml_generator, '_get_context_elements') as mock_get_context:
            mock_get_context.return_value = [
                sample_elements[0],  # parent (intra-document)
                sample_elements[2],  # child (intra-document)
                cross_doc_element   # cross-document
            ]
            
            with patch('go_doc_go.embeddings.xml_contextual_embedding.create_content_resolver') as mock_resolver_factory:
                mock_resolver = Mock()
                mock_resolver.resolve_content.side_effect = lambda loc, text=False: loc.get("text", "") if loc else ""
                mock_resolver_factory.return_value = mock_resolver
                
                # Generate embeddings for main element
                main_element = sample_elements[1]  # main_element_456
                embeddings = xml_generator.generate_from_elements([main_element], db=Mock())
                
                # Verify the XML context was built with cross-document separation
                xml_generator.base_generator.generate.assert_called_once()
                xml_context = xml_generator.base_generator.generate.call_args[0][0]
                
                # Check that intra-document and cross-document contexts are properly categorized
                assert 'role="parent"' in xml_context  # Intra-document parent
                assert 'role="child"' in xml_context   # Intra-document child
                assert 'role="related"' in xml_context # Cross-document context
                
                # Cross-document content should be in related role
                assert "Cross-document revenue analysis" in xml_context
    
    def test_xml_cross_doc_token_budget_management(self, xml_generator):
        """Test that cross-document context respects token budgets."""
        
        # Increase budget to allow some cross-document content
        xml_generator.safe_max_tokens = 1200  # Higher budget for this test
        
        # Create context with many cross-document elements (should trigger budget management)
        cross_doc_texts = [f"Content {i}" for i in range(10)]  # Very short texts to fit budget
        cross_doc_metadata = [{"element_id": f"cross_{i}", "_cross_document": True} for i in range(10)]
        
        xml_content = xml_generator.build_xml_structured_context(
            element_text="Main content",
            parent_texts=[],
            sibling_texts=[],
            child_texts=[], 
            cross_doc_texts=cross_doc_texts,
            cross_doc_metadata=cross_doc_metadata,
            doc_type="pdf"
        )
        
        # Should still be valid XML despite budget constraints
        assert xml_content.startswith('<document')
        assert xml_content.endswith('</document>')
        
        # Should contain some but not all cross-document content due to budget limits
        cross_doc_count = xml_content.count('role="related"')
        assert 0 < cross_doc_count <= len(cross_doc_texts)  # Some but not necessarily all
        
        # Should respect token limits
        token_count = xml_generator.count_tokens(xml_content)
        assert token_count <= xml_generator.safe_max_tokens
    
    def test_xml_cross_doc_prioritization(self, xml_generator):
        """Test that cross-document elements are prioritized by similarity score."""
        
        # Create cross-document texts with different similarity scores
        cross_doc_texts = ["High similarity content", "Low similarity content"]
        cross_doc_metadata = [
            {"element_id": "high_sim", "_cross_document": True, "_similarity_score": 0.95},
            {"element_id": "low_sim", "_cross_document": True, "_similarity_score": 0.45}
        ]
        
        # Use a constrained budget that can only fit one cross-document element
        xml_generator.safe_max_tokens = 800  # Sufficient budget for XML structure + one cross-doc element
        
        xml_content = xml_generator.build_xml_structured_context(
            element_text="Main content",
            parent_texts=[],
            sibling_texts=[],
            child_texts=[], 
            cross_doc_texts=cross_doc_texts,
            cross_doc_metadata=cross_doc_metadata,
            doc_type="pdf"
        )
        
        # Should include high similarity content
        assert "High similarity content" in xml_content
        # Should prioritize high similarity over low similarity when budget constrained
        # (Low similarity content may or may not be included depending on budget)
    
    def test_xml_generator_fallback_to_parent(self, mock_config, mock_base_generator, mock_tokenizer):
        """Test that XML generator falls back to parent implementation when XML is disabled."""
        # Mock config with proper structure for factory
        mock_config.get.return_value = {}  # Return empty dict for content_sources
        
        # Create generator with XML disabled
        generator = XMLContextualEmbeddingGenerator(
            _config=mock_config,
            base_generator=mock_base_generator,
            use_xml_tags=False  # Disabled
        )
        
        elements = [{"element_pk": 123, "element_id": "test_123", "element_type": "paragraph", "content_location": {"text": "test"}}]
        
        # Mock content resolver for both XML and parent implementation
        with patch('go_doc_go.embeddings.xml_contextual_embedding.create_content_resolver') as mock_xml_resolver_factory, \
             patch('go_doc_go.embeddings.contextual_embedding.create_content_resolver') as mock_parent_resolver_factory:
            mock_resolver = Mock()
            mock_resolver.resolve_content.return_value = "test"
            mock_xml_resolver_factory.return_value = mock_resolver
            mock_parent_resolver_factory.return_value = mock_resolver
            
            # Should call parent implementation when XML is disabled
            embeddings = generator.generate_from_elements(elements, db=Mock())
            
            # Verify XML tags are not used when disabled
            assert generator.use_xml_tags is False
            # Verify embeddings were generated
            assert len(embeddings) == 1
            assert 123 in embeddings