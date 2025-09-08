"""
Tests for priority-based contextual embeddings.
"""

import pytest
from unittest.mock import Mock, MagicMock
from go_doc_go.embeddings.priority_contextual import (
    PriorityContextualEmbedding, 
    Priority, 
    PrioritizedContent,
    DynamicPriorityStrategy,
    ContextRole
)


@pytest.mark.unit
class TestPriorityContextualEmbedding:
    """Unit tests for priority-based contextual embedding."""
    
    @pytest.fixture
    def mock_base_generator(self):
        """Create mock base generator."""
        mock = Mock()
        mock.generate.return_value = [0.1] * 768  # Mock embedding vector
        return mock
    
    @pytest.fixture
    def sample_element(self):
        """Create sample main element."""
        return {
            "element_id": "main_para",
            "element_type": "paragraph",
            "content_preview": "This is the main paragraph content.",
            "text": "This is the main paragraph content.",
            "document_position": 10,
            "metadata": {"page": 5}
        }
    
    @pytest.fixture
    def sample_context(self):
        """Create sample context elements."""
        return {
            "parents": [
                {
                    "element_id": "section_1",
                    "element_type": "section",
                    "content_preview": "Parent section content",
                    "document_position": 5
                },
                {
                    "element_id": "chapter_1",
                    "element_type": "chapter",
                    "content_preview": "Chapter content",
                    "document_position": 1
                }
            ],
            "siblings": [
                {
                    "element_id": "para_prev",
                    "element_type": "paragraph",
                    "content_preview": "Previous paragraph",
                    "document_position": 9
                },
                {
                    "element_id": "para_next",
                    "element_type": "paragraph",
                    "content_preview": "Next paragraph",
                    "document_position": 11
                }
            ],
            "children": [
                {
                    "element_id": "list_1",
                    "element_type": "list",
                    "content_preview": "Child list content",
                    "document_position": 12
                }
            ]
        }
    
    def test_priority_assignment(self, mock_base_generator, sample_element, sample_context):
        """Test that priorities are correctly assigned."""
        embedder = PriorityContextualEmbedding(
            base_generator=mock_base_generator,
            max_tokens=1000,
            use_semantic_tags=True
        )
        
        combined_context, stats = embedder.build_prioritized_context(
            sample_element,
            sample_context
        )
        
        # Check that main element is included (CRITICAL priority)
        assert "main paragraph content" in combined_context
        
        # Check that high priority elements are included
        assert "Parent section content" in combined_context
        assert "Previous paragraph" in combined_context or "Next paragraph" in combined_context
        
        # Check stats
        assert stats["included_elements"] > 0
        assert stats["total_tokens"] > 0
        assert stats["utilization"] > 0
    
    def test_token_limit_enforcement(self, mock_base_generator, sample_element):
        """Test that token limit is enforced."""
        embedder = PriorityContextualEmbedding(
            base_generator=mock_base_generator,
            max_tokens=200,  # Small limit
            use_semantic_tags=True
        )
        
        # Create many context elements
        large_context = {
            "parents": [
                {
                    "element_id": f"parent_{i}",
                    "element_type": "section",
                    "content_preview": "Long parent content " * 20,  # Long text
                    "document_position": i
                }
                for i in range(10)
            ],
            "siblings": [
                {
                    "element_id": f"sibling_{i}",
                    "element_type": "paragraph",
                    "content_preview": "Long sibling content " * 20,
                    "document_position": 10 + i
                }
                for i in range(10)
            ],
            "children": []
        }
        
        combined_context, stats = embedder.build_prioritized_context(
            sample_element,
            large_context
        )
        
        # Check that we're within token limit
        actual_tokens = embedder.count_tokens(combined_context)
        assert actual_tokens <= embedder.safe_max_tokens
        
        # Check that some elements were excluded
        assert stats["excluded_elements"] > 0
        assert stats["utilization"] <= 1.0
    
    def test_semantic_tags_generation(self, mock_base_generator, sample_element, sample_context):
        """Test that semantic tags are properly generated."""
        embedder = PriorityContextualEmbedding(
            base_generator=mock_base_generator,
            max_tokens=1000,
            use_semantic_tags=True
        )
        
        combined_context, stats = embedder.build_prioritized_context(
            sample_element,
            sample_context
        )
        
        # Check for semantic tags
        assert "[RELATED:" in combined_context or "[" in combined_context
        assert "[PARENT:" in combined_context
        assert "[PRECEDING:" in combined_context or "[FOLLOWING:" in combined_context
        
        # Main element should have a tag
        assert "main_para" in combined_context
    
    def test_without_semantic_tags(self, mock_base_generator, sample_element, sample_context):
        """Test context building without semantic tags."""
        embedder = PriorityContextualEmbedding(
            base_generator=mock_base_generator,
            max_tokens=1000,
            use_semantic_tags=False
        )
        
        combined_context, stats = embedder.build_prioritized_context(
            sample_element,
            sample_context
        )
        
        # Check for simple role prefixes
        assert "[RELATED]" in combined_context or "[PARENT]" in combined_context
        
        # Should still include content
        assert "main paragraph content" in combined_context
        assert stats["included_elements"] > 0
    
    def test_priority_ordering(self, mock_base_generator):
        """Test that elements are processed in priority order."""
        embedder = PriorityContextualEmbedding(
            base_generator=mock_base_generator,
            max_tokens=300,  # Limited space
            use_semantic_tags=False  # Simpler output for testing
        )
        
        element = {
            "element_id": "main",
            "element_type": "text",
            "content_preview": "Main",
            "document_position": 5
        }
        
        context = {
            "parents": [
                {
                    "element_id": "parent1",
                    "element_type": "section",
                    "content_preview": "Parent One",  # HIGH priority
                    "document_position": 1
                }
            ],
            "siblings": [
                {
                    "element_id": "sib1",
                    "element_type": "text",
                    "content_preview": "Sibling One",  # HIGH priority (immediate)
                    "document_position": 4
                },
                {
                    "element_id": "sib5",
                    "element_type": "text",
                    "content_preview": "Sibling Five " * 50,  # MEDIUM priority but very long
                    "document_position": 10
                }
            ],
            "children": [
                {
                    "element_id": "child1",
                    "element_type": "item",
                    "content_preview": "Child One",  # MEDIUM/LOW priority
                    "document_position": 6
                }
            ]
        }
        
        combined_context, stats = embedder.build_prioritized_context(element, context)
        
        # Main should always be included
        assert "Main" in combined_context
        
        # High priority items should be included
        assert "Parent One" in combined_context
        assert "Sibling One" in combined_context
        
        # Very long medium priority item might be excluded
        if "Sibling Five" not in combined_context:
            assert stats["excluded_elements"] > 0
    
    def test_truncation_of_critical_element(self, mock_base_generator):
        """Test that critical element is truncated if too large."""
        embedder = PriorityContextualEmbedding(
            base_generator=mock_base_generator,
            max_tokens=100,  # Very small limit
            use_semantic_tags=False
        )
        
        element = {
            "element_id": "main",
            "element_type": "text",
            "content_preview": "Very long main content " * 100,  # Too long
            "text": "Very long main content " * 100,
            "document_position": 1
        }
        
        context = {"parents": [], "siblings": [], "children": []}
        
        combined_context, stats = embedder.build_prioritized_context(element, context)
        
        # Should include truncated main element
        assert "[MAIN]" in combined_context  # MAIN role is used for the primary element
        assert "[truncated]" in combined_context
        assert embedder.count_tokens(combined_context) <= embedder.safe_max_tokens


@pytest.mark.unit
class TestDynamicPriorityStrategy:
    """Test dynamic priority calculation."""
    
    @pytest.fixture
    def strategy(self):
        """Create strategy instance."""
        return DynamicPriorityStrategy()
    
    def test_important_type_priority(self, strategy):
        """Test that important element types get higher priority."""
        header_element = {
            "element_type": "header",
            "content_preview": "Section Header"
        }
        
        # Headers as parents should be HIGH priority
        priority = strategy.calculate_priority(header_element, "parent", distance=0)
        assert priority == Priority.HIGH
        
        # Headers as distant siblings should be MEDIUM
        priority = strategy.calculate_priority(header_element, "sibling", distance=5)
        assert priority == Priority.MEDIUM
    
    def test_cross_reference_detection(self, strategy):
        """Test that cross-references increase priority."""
        ref_element = {
            "element_type": "paragraph",
            "content_preview": "This refers to the previous section for details."
        }
        
        # Should get MEDIUM priority due to cross-reference
        priority = strategy.calculate_priority(ref_element, "sibling", distance=2)
        assert priority == Priority.MEDIUM
        
        # Without cross-reference keyword
        normal_element = {
            "element_type": "paragraph",
            "content_preview": "This is normal paragraph text."
        }
        
        priority = strategy.calculate_priority(normal_element, "sibling", distance=4)
        assert priority == Priority.LOW
    
    def test_distance_based_priority(self, strategy):
        """Test that distance affects priority."""
        element = {
            "element_type": "paragraph",
            "content_preview": "Content"
        }
        
        # Close siblings are HIGH priority
        priority = strategy.calculate_priority(element, "sibling", distance=1)
        assert priority == Priority.HIGH
        
        # Medium distance siblings are MEDIUM priority
        priority = strategy.calculate_priority(element, "sibling", distance=2)
        assert priority == Priority.MEDIUM
        
        # Distant siblings are LOW priority
        priority = strategy.calculate_priority(element, "sibling", distance=5)
        assert priority == Priority.LOW
    
    def test_reorder_by_relevance(self, strategy):
        """Test reordering within priority groups."""
        main_element = {
            "element_id": "main",
            "content_preview": "Main content"
        }
        
        prioritized = [
            PrioritizedContent(
                text="Far sibling",
                priority=Priority.MEDIUM,
                role=ContextRole.SIBLING,
                metadata={},
                relationship_distance=5
            ),
            PrioritizedContent(
                text="Close sibling with more content",
                priority=Priority.MEDIUM,
                role=ContextRole.SIBLING,
                metadata={},
                relationship_distance=2
            ),
            PrioritizedContent(
                text="Critical",
                priority=Priority.CRITICAL,
                role=ContextRole.RELATED,
                metadata={},
                relationship_distance=0
            ),
            PrioritizedContent(
                text="Another close sibling",
                priority=Priority.MEDIUM,
                role=ContextRole.SIBLING,
                metadata={},
                relationship_distance=2
            )
        ]
        
        reordered = strategy.reorder_by_relevance(prioritized, main_element)
        
        # Critical should be first
        assert reordered[0].priority == Priority.CRITICAL
        
        # Within MEDIUM priority, closer elements should come first
        medium_items = [item for item in reordered if item.priority == Priority.MEDIUM]
        if len(medium_items) > 1:
            assert medium_items[0].relationship_distance <= medium_items[-1].relationship_distance


@pytest.mark.integration
class TestPriorityContextualIntegration:
    """Integration tests for priority-based embedding."""
    
    def test_full_embedding_generation(self):
        """Test complete embedding generation with priority context."""
        # Create mock base generator
        mock_generator = Mock()
        mock_generator.generate.return_value = [0.5] * 768
        
        embedder = PriorityContextualEmbedding(
            base_generator=mock_generator,
            max_tokens=500,
            use_semantic_tags=True
        )
        
        element = {
            "element_id": "test_elem",
            "element_type": "paragraph",
            "content_preview": "Test content",
            "metadata": {"section": "intro"}
        }
        
        context = {
            "parents": [
                {"element_id": "p1", "element_type": "section", "content_preview": "Parent"},
            ],
            "siblings": [
                {"element_id": "s1", "element_type": "paragraph", "content_preview": "Sibling", "document_position": 8},
            ],
            "children": []
        }
        
        embedding, stats = embedder.generate_with_context(element, context)
        
        # Check embedding was generated
        assert len(embedding) == 768
        assert all(v == 0.5 for v in embedding)
        
        # Check stats
        assert "total_tokens" in stats
        assert "included_elements" in stats
        assert stats["included_elements"] >= 1  # At least main element
        
        # Check that generate was called with combined context
        mock_generator.generate.assert_called_once()
        combined_context = mock_generator.generate.call_args[0][0]
        assert "Test content" in combined_context
    
    def test_token_counting_accuracy(self):
        """Test that token counting is reasonably accurate."""
        embedder = PriorityContextualEmbedding(
            base_generator=Mock(),
            max_tokens=1000
        )
        
        # Test various text lengths
        test_cases = [
            ("Hello world", 2, 10),  # Short text
            ("This is a longer sentence with multiple words.", 8, 15),  # Medium
            ("The quick brown fox jumps over the lazy dog. " * 10, 80, 120),  # Long
        ]
        
        for text, min_tokens, max_tokens in test_cases:
            token_count = embedder.count_tokens(text)
            assert min_tokens <= token_count <= max_tokens, \
                f"Token count {token_count} not in range [{min_tokens}, {max_tokens}] for text: {text[:50]}..."