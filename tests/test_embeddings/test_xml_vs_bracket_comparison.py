"""
A/B comparison tests between XML and bracket format contextual embeddings.
"""

import numpy as np
import pytest
from unittest.mock import Mock, MagicMock, patch
from typing import List, Dict, Any

from go_doc_go.embeddings.contextual_embedding import ContextualEmbeddingGenerator
from go_doc_go.embeddings.xml_contextual_embedding import XMLContextualEmbeddingGenerator
from go_doc_go.embeddings.xml_semantic_tagger import ContextRole


class MockEmbeddingGenerator:
    """Mock embedding generator that returns predictable embeddings based on content."""
    
    def __init__(self, dimensions: int = 384):
        self.dimensions = dimensions
    
    def generate(self, text: str) -> List[float]:
        """Generate mock embedding based on text content characteristics."""
        # Use text characteristics to create different embeddings
        text_lower = text.lower()
        
        # Create base embedding
        embedding = [0.0] * self.dimensions
        
        # Add patterns based on content
        if "revenue" in text_lower:
            embedding[0] = 0.8
            embedding[1] = 0.6
        if "database" in text_lower:
            embedding[2] = 0.7
            embedding[3] = 0.5
        if "performance" in text_lower:
            embedding[4] = 0.9
            embedding[5] = 0.4
        if "main" in text_lower or "element" in text_lower:
            embedding[10] = 0.8
        if "parent" in text_lower or "context" in text_lower:
            embedding[11] = 0.6
        if "xml" in text_lower or "<" in text:
            embedding[12] = 0.7  # XML structure marker
        
        # Add some randomness based on text length
        text_hash = hash(text) % 1000
        for i in range(20, min(50, self.dimensions)):
            embedding[i] = (text_hash % 100) / 100.0
        
        # Normalize to unit vector
        norm = np.linalg.norm(embedding)
        if norm > 0:
            embedding = [x / norm for x in embedding]
        
        return embedding
    
    def generate_batch(self, texts: List[str]) -> List[List[float]]:
        return [self.generate(text) for text in texts]
    
    def get_dimensions(self) -> int:
        return self.dimensions
    
    def get_model_name(self) -> str:
        return "mock-embedding-model"
    
    def clear_cache(self) -> None:
        pass


class TestXMLvsBracketComparison:
    """A/B comparison tests between XML and bracket formatting."""
    
    @pytest.fixture
    def mock_config(self):
        """Mock configuration."""
        config = Mock()
        config.get = Mock(return_value=None)
        return config
    
    @pytest.fixture
    def base_generator(self):
        """Create mock base embedding generator."""
        return MockEmbeddingGenerator(dimensions=384)
    
    @pytest.fixture
    def mock_tokenizer(self):
        """Mock tokenizer for consistent token counting."""
        with patch('go_doc_go.embeddings.contextual_embedding.TIKTOKEN_AVAILABLE', True):
            with patch('go_doc_go.embeddings.xml_contextual_embedding.TIKTOKEN_AVAILABLE', True):
                with patch('tiktoken.get_encoding') as mock_get_encoding:
                    mock_tokenizer = Mock()
                    # Simple token counting: ~1 token per 4 characters
                    mock_tokenizer.encode.side_effect = lambda text: list(range(len(text) // 4 + 1))
                    mock_tokenizer.decode.side_effect = lambda tokens: "decoded_" + "_".join(map(str, tokens[:10]))
                    mock_get_encoding.return_value = mock_tokenizer
                    yield mock_tokenizer
    
    @pytest.fixture
    def bracket_generator(self, mock_config, base_generator, mock_tokenizer):
        """Create bracket format contextual embedding generator."""
        return ContextualEmbeddingGenerator(
            _config=mock_config,
            base_generator=base_generator,
            max_tokens=1000,
            use_semantic_tags=True
        )
    
    @pytest.fixture
    def xml_generator(self, mock_config, base_generator, mock_tokenizer):
        """Create XML format contextual embedding generator."""
        return XMLContextualEmbeddingGenerator(
            _config=mock_config,
            base_generator=base_generator,
            max_tokens=1000,
            use_xml_tags=True,
            include_entities=True,
            include_strength=True
        )
    
    @pytest.fixture
    def sample_elements(self):
        """Create sample document elements for testing."""
        return [
            {
                "element_pk": "elem_1",
                "element_id": "para_main",
                "element_type": "paragraph",
                "content_preview": "The company's quarterly revenue increased by 15% reaching $2.3 million",
                "content_location": {"source": "main_content.txt", "type": "text"},
                "parent_id": None,
                "metadata": {
                    "page_number": 1,
                    "position": 1
                }
            },
            {
                "element_pk": "elem_2", 
                "element_id": "section_parent",
                "element_type": "section",
                "content_preview": "Financial Performance Overview for Q4 2023",
                "content_location": {"source": "parent_content.txt", "type": "text"},
                "parent_id": None,
                "metadata": {
                    "level": 1,
                    "page_number": 1
                }
            },
            {
                "element_pk": "elem_3",
                "element_id": "para_sibling",
                "element_type": "paragraph", 
                "content_preview": "Database performance metrics showed significant improvement",
                "content_location": {"source": "sibling_content.txt", "type": "text"},
                "parent_id": "section_parent",
                "metadata": {
                    "page_number": 1,
                    "position": 2
                }
            }
        ]
    
    def create_mock_resolver(self):
        """Create mock content resolver."""
        resolver = Mock()
        content_map = {
            "main_content.txt": "The company's quarterly revenue increased by 15% reaching $2.3 million in Q4 2023",
            "parent_content.txt": "Financial Performance Overview for Q4 2023 showing strong growth",
            "sibling_content.txt": "Database performance metrics showed significant improvement in query response time"
        }
        resolver.resolve_content.side_effect = lambda location, text=True: content_map.get(location["source"], "")
        return resolver
    
    def test_embedding_generation_comparison(self, bracket_generator, xml_generator, sample_elements):
        """Compare embedding generation between formats."""
        with patch('go_doc_go.embeddings.contextual_embedding.create_content_resolver') as mock_resolver1:
            with patch('go_doc_go.embeddings.xml_contextual_embedding.create_content_resolver') as mock_resolver2:
                # Setup mock resolvers
                resolver = self.create_mock_resolver()
                mock_resolver1.return_value = resolver
                mock_resolver2.return_value = resolver
                
                # Generate embeddings with both formats
                bracket_embeddings = bracket_generator.generate_from_elements(sample_elements)
                xml_embeddings = xml_generator.generate_from_elements(sample_elements)
                
                # Both should generate embeddings for all elements
                assert len(bracket_embeddings) == len(xml_embeddings)
                assert len(bracket_embeddings) == len(sample_elements)
                
                # Embeddings should be different (different input formats)
                for elem_pk in bracket_embeddings:
                    if elem_pk in xml_embeddings:
                        bracket_emb = np.array(bracket_embeddings[elem_pk])
                        xml_emb = np.array(xml_embeddings[elem_pk])
                        
                        # Should be different embeddings
                        cosine_sim = np.dot(bracket_emb, xml_emb) / (np.linalg.norm(bracket_emb) * np.linalg.norm(xml_emb))
                        assert cosine_sim < 0.99  # Should not be identical
    
    def test_content_structure_comparison(self, bracket_generator, xml_generator):
        """Compare the structure of generated content."""
        element_text = "Database connection timeout affects query performance"
        parent_texts = ["System Configuration Section"]
        sibling_texts = ["Performance metrics improved by 20%"]
        child_texts = ["Query timeout set to 30 seconds"]
        
        # Generate content with both formats
        bracket_content = bracket_generator._combine_text_with_context(
            element_text, parent_texts + sibling_texts + child_texts
        )
        
        xml_content = xml_generator._build_simple_xml_context(
            element_text, parent_texts + sibling_texts + child_texts
        )
        
        # Both should contain the main content
        assert "Database connection timeout" in bracket_content
        assert "Database connection timeout" in xml_content
        
        # XML should have structured tags
        assert "<document" in xml_content
        assert "</document>" in xml_content
        assert 'role="main"' in xml_content
        
        # Bracket should have section headers
        assert "===" in bracket_content
        
        # XML should be longer due to structure
        assert len(xml_content) > len(bracket_content)
    
    def test_entity_extraction_advantage(self, xml_generator):
        """Test that XML format provides entity extraction advantages."""
        element_text = "The quarterly revenue increased by 15% to $2.3 million in Q4 2023"
        context_texts = ["Database performance improved significantly", "Server connection timeout reduced"]
        
        xml_content = xml_generator._build_simple_xml_context(element_text, context_texts)
        
        # Should contain entity information
        assert 'entities=' in xml_content
        
        # Check that financial and technical entities are extracted
        content_lower = xml_content.lower()
        assert any(term in content_lower for term in ['revenue', '2.3', 'q4', '2023'])
        assert any(term in content_lower for term in ['database', 'performance', 'server', 'timeout'])
    
    def test_token_efficiency_comparison(self, bracket_generator, xml_generator):
        """Compare token efficiency between formats."""
        element_text = "Main content about revenue performance"
        context_texts = ["Context 1", "Context 2", "Context 3"]
        
        # Generate content with both formats
        bracket_content = bracket_generator._combine_text_with_context(element_text, context_texts)
        xml_content = xml_generator._build_simple_xml_context(element_text, context_texts)
        
        # Count tokens
        bracket_tokens = bracket_generator.count_tokens(bracket_content)
        xml_tokens = xml_generator.count_tokens(xml_content)
        
        # XML will use more tokens due to structure, but should provide more information
        assert xml_tokens > bracket_tokens
        
        # Information density: XML should have entities and structure info
        xml_info_markers = xml_content.count('entities=') + xml_content.count('role=') + xml_content.count('type=')
        bracket_info_markers = bracket_content.count('===') + bracket_content.count('[') + bracket_content.count(']')
        
        # XML should have more structured information
        assert xml_info_markers > bracket_info_markers
    
    def test_semantic_similarity_patterns(self, bracket_generator, xml_generator):
        """Test semantic similarity patterns between formats."""
        # Test related content
        related_texts = [
            "Database performance optimization techniques",
            "Query performance improvement strategies", 
            "Database query optimization methods"
        ]
        
        # Test unrelated content
        unrelated_texts = [
            "Weather forecast for tomorrow",
            "Cooking recipe for pasta",
            "Sports scores from yesterday"
        ]
        
        # Generate embeddings
        with patch('go_doc_go.embeddings.contextual_embedding.create_content_resolver'):
            with patch('go_doc_go.embeddings.xml_contextual_embedding.create_content_resolver'):
                # Mock elements
                related_elements = [
                    {
                        "element_pk": f"elem_{i}",
                        "element_id": f"elem_{i}",
                        "element_type": "paragraph",
                        "content_preview": text[:50],
                        "content_location": {"source": f"file_{i}.txt", "type": "text"},
                        "parent_id": None,
                        "metadata": {}
                    }
                    for i, text in enumerate(related_texts)
                ]
                
                # Mock resolver
                resolver = Mock()
                content_map = {f"file_{i}.txt": text for i, text in enumerate(related_texts)}
                resolver.resolve_content.side_effect = lambda location, text=True: content_map.get(location["source"], "")
                
                with patch('go_doc_go.embeddings.contextual_embedding.create_content_resolver', return_value=resolver):
                    with patch('go_doc_go.embeddings.xml_contextual_embedding.create_content_resolver', return_value=resolver):
                        bracket_embeddings = bracket_generator.generate_from_elements(related_elements)
                        xml_embeddings = xml_generator.generate_from_elements(related_elements)
                
                # Calculate similarity matrices
                bracket_sims = []
                xml_sims = []
                
                elem_pks = list(bracket_embeddings.keys())
                for i in range(len(elem_pks)):
                    for j in range(i+1, len(elem_pks)):
                        pk1, pk2 = elem_pks[i], elem_pks[j]
                        
                        # Bracket similarity
                        emb1 = np.array(bracket_embeddings[pk1])
                        emb2 = np.array(bracket_embeddings[pk2])
                        bracket_sim = np.dot(emb1, emb2) / (np.linalg.norm(emb1) * np.linalg.norm(emb2))
                        bracket_sims.append(bracket_sim)
                        
                        # XML similarity  
                        emb1 = np.array(xml_embeddings[pk1])
                        emb2 = np.array(xml_embeddings[pk2])
                        xml_sim = np.dot(emb1, emb2) / (np.linalg.norm(emb1) * np.linalg.norm(emb2))
                        xml_sims.append(xml_sim)
                
                # Both formats should show high similarity for related content
                avg_bracket_sim = np.mean(bracket_sims)
                avg_xml_sim = np.mean(xml_sims)
                
                assert avg_bracket_sim > 0.5  # Should be similar
                assert avg_xml_sim > 0.5      # Should be similar
                
                # The difference indicates different structural understanding
                assert abs(avg_xml_sim - avg_bracket_sim) > 0.01  # Should be measurably different
    
    def test_robustness_to_content_variations(self, bracket_generator, xml_generator):
        """Test robustness of both formats to content variations."""
        base_content = "Database performance analysis shows improvement"
        variations = [
            base_content,
            base_content.upper(),
            base_content.replace("Database", "DB"),
            base_content + " with additional metrics data",
            base_content.replace("performance", "efficiency")
        ]
        
        bracket_embeddings = []
        xml_embeddings = []
        
        for content in variations:
            bracket_emb = bracket_generator.generate(content)
            xml_emb = xml_generator.generate(content)
            
            bracket_embeddings.append(bracket_emb)
            xml_embeddings.append(xml_emb)
        
        # Calculate stability (similarity to base)
        base_bracket = np.array(bracket_embeddings[0])
        base_xml = np.array(xml_embeddings[0])
        
        bracket_stabilities = []
        xml_stabilities = []
        
        for i in range(1, len(variations)):
            bracket_sim = np.dot(base_bracket, bracket_embeddings[i]) / (
                np.linalg.norm(base_bracket) * np.linalg.norm(bracket_embeddings[i])
            )
            xml_sim = np.dot(base_xml, xml_embeddings[i]) / (
                np.linalg.norm(base_xml) * np.linalg.norm(xml_embeddings[i])
            )
            
            bracket_stabilities.append(bracket_sim)
            xml_stabilities.append(xml_sim)
        
        # Both should show reasonable stability
        assert np.mean(bracket_stabilities) > 0.7
        assert np.mean(xml_stabilities) > 0.7
        
        # Different approaches may have different stability characteristics
        stability_diff = abs(np.mean(xml_stabilities) - np.mean(bracket_stabilities))
        assert stability_diff < 0.3  # Should not be drastically different
    
    @pytest.mark.slow
    def test_performance_benchmark(self, bracket_generator, xml_generator, sample_elements):
        """Benchmark performance difference between formats."""
        import time
        
        with patch('go_doc_go.embeddings.contextual_embedding.create_content_resolver') as mock_resolver1:
            with patch('go_doc_go.embeddings.xml_contextual_embedding.create_content_resolver') as mock_resolver2:
                resolver = self.create_mock_resolver()
                mock_resolver1.return_value = resolver
                mock_resolver2.return_value = resolver
                
                # Benchmark bracket format
                start_time = time.time()
                for _ in range(10):  # Multiple runs for averaging
                    bracket_generator.generate_from_elements(sample_elements)
                bracket_time = (time.time() - start_time) / 10
                
                # Benchmark XML format
                start_time = time.time()
                for _ in range(10):
                    xml_generator.generate_from_elements(sample_elements)
                xml_time = (time.time() - start_time) / 10
                
                # XML should be slower due to more processing, but not excessively
                assert xml_time > bracket_time  # XML does more work
                assert xml_time < bracket_time * 5  # But not more than 5x slower
                
                print(f"Bracket format: {bracket_time:.4f}s per batch")
                print(f"XML format: {xml_time:.4f}s per batch")
                print(f"Overhead factor: {xml_time/bracket_time:.2f}x")