"""
Unit tests for cross-document contextual embeddings.
"""

import pytest
from unittest.mock import Mock, MagicMock, patch

from go_doc_go.embeddings.contextual_embedding import ContextualEmbeddingGenerator
from go_doc_go.storage.element_relationship import ElementRelationship


class TestCrossDocumentContext:
    """Test cross-document context integration."""
    
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
    def mock_db(self):
        """Mock database with cross-document relationships."""
        db = Mock()
        
        # Mock cross-document relationship
        rel = ElementRelationship(
            relationship_id="rel_123",
            source_id="test_element_123",
            relationship_type="semantic_section",
            target_reference="cross_doc_element_123",
            metadata={
                "cross_document": True,
                "similarity_score": 0.85,
                "source_doc_id": "doc1",
                "target_doc_id": "doc2"
            }
        )
        db.get_outgoing_relationships.return_value = [rel]
        
        # Mock target element
        target_element = {
            "element_id": "cross_doc_element_123",
            "element_pk": 456,
            "element_type": "paragraph",
            "content_preview": "Related content from another document about revenue analysis",
            "doc_id": "doc2"
        }
        db.get_element.return_value = target_element
        
        return db
    
    @pytest.fixture
    def contextual_generator(self, mock_config, mock_base_generator):
        """Create contextual embedding generator."""
        return ContextualEmbeddingGenerator(
            _config=mock_config,
            base_generator=mock_base_generator,
            max_tokens=1000,
            use_semantic_tags=False
        )
    
    def test_get_cross_document_context_success(self, contextual_generator, mock_db):
        """Test successful cross-document context retrieval."""
        element = {
            "element_id": "test_element_123",
            "element_pk": 789,
            "element_type": "paragraph",
            "content_preview": "Main element content",
            "doc_id": "doc1"
        }
        
        cross_doc_elements = contextual_generator._get_cross_document_context(element, mock_db)
        
        # Should retrieve one cross-document element
        assert len(cross_doc_elements) == 1
        
        cross_elem = cross_doc_elements[0]
        assert cross_elem["element_id"] == "cross_doc_element_123"
        assert cross_elem["_cross_document"] is True
        assert cross_elem["_similarity_score"] == 0.85
        assert cross_elem["_source_doc_id"] == "doc2"
        assert "Related content from another document" in cross_elem["content_preview"]
    
    def test_get_cross_document_context_no_element_pk(self, contextual_generator, mock_db):
        """Test cross-document context with missing element_pk."""
        element = {
            "element_id": "test_element_123",
            "element_type": "paragraph",
            "content_preview": "Main element content"
            # Missing element_pk
        }
        
        cross_doc_elements = contextual_generator._get_cross_document_context(element, mock_db)
        
        # Should return empty list
        assert cross_doc_elements == []
    
    def test_get_cross_document_context_no_relationships(self, contextual_generator):
        """Test cross-document context with no relationships."""
        mock_db = Mock()
        mock_db.get_outgoing_relationships.return_value = []
        
        element = {
            "element_id": "test_element_123",
            "element_pk": 789,
            "element_type": "paragraph",
            "content_preview": "Main element content",
            "doc_id": "doc1"
        }
        
        cross_doc_elements = contextual_generator._get_cross_document_context(element, mock_db)
        
        # Should return empty list
        assert cross_doc_elements == []
    
    def test_get_cross_document_context_filters_non_cross_doc(self, contextual_generator):
        """Test that only cross-document relationships are included."""
        mock_db = Mock()
        
        # Mix of relationships
        relationships = [
            ElementRelationship(
                relationship_id="rel_intra",
                source_id="test_element_123",
                relationship_type="contains",
                target_reference="intra_doc_element",
                metadata={"cross_document": False}  # Not cross-document
            ),
            ElementRelationship(
                relationship_id="rel_cross",
                source_id="test_element_123",
                relationship_type="semantic_section",
                target_reference="cross_doc_element",
                metadata={"cross_document": True, "similarity_score": 0.9}  # Cross-document
            )
        ]
        mock_db.get_outgoing_relationships.return_value = relationships
        
        # Mock only the cross-document target
        mock_db.get_element.return_value = {
            "element_id": "cross_doc_element",
            "content_preview": "Cross-document content",
            "doc_id": "doc2"
        }
        
        element = {
            "element_id": "test_element_123",
            "element_pk": 789,
            "element_type": "paragraph",
            "doc_id": "doc1"
        }
        
        cross_doc_elements = contextual_generator._get_cross_document_context(element, mock_db)
        
        # Should only include the cross-document relationship
        assert len(cross_doc_elements) == 1
        assert cross_doc_elements[0]["element_id"] == "cross_doc_element"
        assert cross_doc_elements[0]["_cross_document"] is True
    
    def test_get_context_elements_with_cross_document(self, contextual_generator, mock_db):
        """Test that _get_context_elements includes cross-document elements."""
        # Mock elements for intra-document context
        all_elements = [
            {
                "element_id": "parent_element",
                "element_type": "section",
                "content_preview": "Parent section content"
            },
            {
                "element_id": "test_element_123",
                "element_pk": 789,
                "element_type": "paragraph",
                "content_preview": "Main element content",
                "parent_id": "parent_element"
            }
        ]
        
        hierarchy = {"parent_element": ["test_element_123"]}
        
        element = {
            "element_id": "test_element_123",
            "element_pk": 789,
            "element_type": "paragraph",
            "content_preview": "Main element content",
            "parent_id": "parent_element"
        }
        
        context_elements = contextual_generator._get_context_elements(
            element, all_elements, hierarchy, mock_db
        )
        
        # Should include both intra-document (parent) and cross-document context
        assert len(context_elements) >= 2  # At least parent + cross-document
        
        # Check that cross-document element is included
        cross_doc_found = False
        for ctx_elem in context_elements:
            if ctx_elem.get("_cross_document"):
                cross_doc_found = True
                assert ctx_elem["element_id"] == "cross_doc_element_123"
                break
        
        assert cross_doc_found, "Cross-document element should be included in context"
    
    def test_generate_from_elements_with_db(self, contextual_generator, mock_db):
        """Test generate_from_elements passes database to context gathering."""
        elements = [
            {
                "element_id": "test_element_123",
                "element_pk": 789,
                "element_type": "paragraph",
                "content_preview": "Test content",
                "content_location": {"type": "text", "text": "Test content"}
            }
        ]
        
        # Mock the content resolver
        with patch('go_doc_go.embeddings.contextual_embedding.create_content_resolver') as mock_resolver_factory:
            mock_resolver = Mock()
            mock_resolver.resolve_content.return_value = "Test content"
            mock_resolver_factory.return_value = mock_resolver
            
            embeddings = contextual_generator.generate_from_elements(elements, db=mock_db)
            
            # Should generate embeddings
            assert len(embeddings) == 1
            assert 789 in embeddings
            
            # Verify database was passed to relationship queries
            mock_db.get_outgoing_relationships.assert_called_once_with(789)