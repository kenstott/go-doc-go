"""
Unit tests for XMLSemanticTagger.
"""

import pytest
from unittest.mock import Mock

from go_doc_go.embeddings.xml_semantic_tagger import XMLSemanticTagger, ContextRole


class TestXMLSemanticTagger:
    """Unit tests for XML semantic tagger."""
    
    @pytest.fixture
    def tagger(self):
        """Create XMLSemanticTagger instance."""
        return XMLSemanticTagger(
            include_metadata=True,
            include_entities=True,
            include_strength=True,
            max_entities=6
        )
    
    @pytest.fixture
    def sample_element(self):
        """Sample element metadata."""
        return {
            "element_type": "paragraph",
            "element_id": "para_123",
            "metadata": {
                "page_number": 5,
                "level": 2,
                "position": 10
            }
        }
    
    def test_entity_extraction(self, tagger):
        """Test entity extraction from text."""
        text = "The company revenue increased by 15% to $2.3 million in Q4 2023"
        entities = tagger.extract_entities(text)
        
        assert len(entities) > 0
        # Should extract financial and numerical entities
        assert any("revenue" in entity.lower() for entity in entities)
        assert any("2.3" in entity or "million" in entity.lower() for entity in entities)
        assert any("q4" in entity.lower() or "2023" in entity for entity in entities)
    
    def test_entity_extraction_technical(self, tagger):
        """Test entity extraction from technical text."""
        text = "Database connection timeout affects query performance and server response"
        entities = tagger.extract_entities(text)
        
        assert len(entities) > 0
        # Should extract technical entities
        expected_entities = ["database", "connection", "timeout", "query", "performance", "server"]
        found_entities = [entity.lower() for entity in entities]
        
        for expected in expected_entities:
            assert any(expected in found for found in found_entities)
    
    def test_relationship_strength_calculation(self, tagger):
        """Test relationship strength calculation."""
        # Test different roles
        assert tagger.calculate_relationship_strength(ContextRole.MAIN, 1) == 1.0
        assert tagger.calculate_relationship_strength(ContextRole.PARENT, 1) == 0.9
        assert tagger.calculate_relationship_strength(ContextRole.CHILD, 1) == 0.7
        
        # Test distance penalty
        parent_strength_1 = tagger.calculate_relationship_strength(ContextRole.PARENT, 1)
        parent_strength_2 = tagger.calculate_relationship_strength(ContextRole.PARENT, 2)
        assert parent_strength_1 > parent_strength_2
        
        # Test important type bonus
        header_strength = tagger.calculate_relationship_strength(ContextRole.PARENT, 1, "header")
        para_strength = tagger.calculate_relationship_strength(ContextRole.PARENT, 1, "paragraph")
        assert header_strength >= para_strength
    
    def test_generate_xml_tag_main_element(self, tagger, sample_element):
        """Test XML tag generation for main element."""
        content = "This is the main paragraph content with revenue data."
        
        xml_tag = tagger.generate_xml_tag(
            element=sample_element,
            content=content,
            context_role=ContextRole.MAIN,
            element_distance=1
        )
        
        assert xml_tag.startswith('<element')
        assert 'role="main"' in xml_tag
        assert 'type="paragraph"' in xml_tag
        assert 'id="para_123"' in xml_tag
        assert 'page="5"' in xml_tag
        assert 'entities=' in xml_tag  # Should contain extracted entities
        assert xml_tag.endswith('</element>')
        assert "revenue" in xml_tag.lower()  # Content should be in the tag
    
    def test_generate_xml_tag_context_element(self, tagger, sample_element):
        """Test XML tag generation for context element."""
        content = "This is parent context about database configuration."
        
        xml_tag = tagger.generate_xml_tag(
            element=sample_element,
            content=content,
            context_role=ContextRole.PARENT,
            element_distance=1
        )
        
        assert xml_tag.startswith('<context')
        assert 'role="parent"' in xml_tag
        assert 'type="paragraph"' in xml_tag
        assert 'strength=' in xml_tag  # Should include relationship strength
        assert 'entities=' in xml_tag  # Should contain extracted entities
        assert xml_tag.endswith('</context>')
        assert "database" in xml_tag.lower()
    
    def test_generate_document_wrapper(self, tagger):
        """Test document wrapper generation."""
        opening, closing = tagger.generate_document_wrapper(
            doc_type="pdf",
            domain="finance",
            doc_id="doc_123"
        )
        
        assert opening.startswith('<document')
        assert 'type="pdf"' in opening
        assert 'domain="finance"' in opening
        assert 'id="doc_123"' in opening
        assert opening.endswith('>')
        
        assert closing == '</document>'
    
    def test_xml_escaping(self, tagger):
        """Test that content is properly XML-escaped."""
        element = {
            "element_type": "paragraph",
            "element_id": "test<>&\"'",  # Contains XML special characters
            "metadata": {}
        }
        content = "Content with <tags> & \"quotes\" and 'apostrophes'"
        
        xml_tag = tagger.generate_xml_tag(
            element=element,
            content=content,
            context_role=ContextRole.MAIN
        )
        
        # Should not contain unescaped XML characters
        assert '<tags>' not in xml_tag
        assert '&quot;' in xml_tag or '&amp;' in xml_tag  # Should be escaped
        assert 'test&lt;&gt;&amp;' in xml_tag  # ID should be escaped
    
    def test_parse_xml_tag(self, tagger):
        """Test parsing XML tag back to components."""
        element = {
            "element_type": "paragraph",
            "element_id": "para_123",
            "metadata": {"page_number": 5}
        }
        content = "Database performance shows revenue growth"
        
        # Generate XML tag
        xml_tag = tagger.generate_xml_tag(
            element=element,
            content=content,
            context_role=ContextRole.MAIN
        )
        
        # Parse it back
        parsed = tagger.parse_xml_tag(xml_tag)
        
        assert parsed is not None
        assert parsed["role"] == "main"
        assert parsed["element_type"] == "paragraph"
        assert parsed["content"] == content
        assert "entities" in parsed
    
    def test_validate_xml_structure_valid(self, tagger):
        """Test XML validation with valid content."""
        valid_xml = """<document type="pdf">
            <element role="main" type="paragraph">Content</element>
            <context role="parent" type="section">Parent content</context>
        </document>"""
        
        assert tagger.validate_xml_structure(valid_xml) is True
    
    def test_validate_xml_structure_invalid(self, tagger):
        """Test XML validation with invalid content."""
        invalid_xml = """<document type="pdf">
            <element role="main" type="paragraph">Content</element>
            <context role="parent" type="section">Unclosed parent content
        </document>"""
        
        assert tagger.validate_xml_structure(invalid_xml) is False
    
    def test_extract_entities_from_xml(self, tagger):
        """Test extracting entities from XML content."""
        xml_content = """<document>
            <element entities="revenue,growth,quarterly">Content 1</element>
            <context entities="database,server,connection">Content 2</context>
        </document>"""
        
        entities = tagger.extract_entities_from_xml(xml_content)
        
        expected_entities = ["revenue", "growth", "quarterly", "database", "server", "connection"]
        for expected in expected_entities:
            assert expected in entities
    
    def test_max_entities_limit(self, tagger):
        """Test that entity extraction respects max_entities limit."""
        # Text with many potential entities
        text = "The database server connection timeout affects query performance and response time while processing customer revenue data from sales transactions in Q4 2023"
        
        entities = tagger.extract_entities(text)
        
        # Should not exceed max_entities
        assert len(entities) <= tagger.max_entities
    
    def test_empty_content_handling(self, tagger):
        """Test handling of empty or None content."""
        element = {"element_type": "paragraph", "element_id": "test"}
        
        # Empty string content
        xml_tag = tagger.generate_xml_tag(element, "", ContextRole.MAIN)
        assert xml_tag.startswith('<element')
        assert xml_tag.endswith('</element>')
        
        # None content should not crash
        xml_tag = tagger.generate_xml_tag(element, None, ContextRole.MAIN)
        assert xml_tag is not None
    
    def test_metadata_inclusion(self, tagger):
        """Test that metadata is properly included in tags."""
        element = {
            "element_type": "paragraph",
            "element_id": "para_123",
            "metadata": {
                "page_number": 5,
                "level": 2,
                "position": 10
            }
        }
        
        xml_tag = tagger.generate_xml_tag(element, "test content", ContextRole.MAIN)
        
        assert 'page="5"' in xml_tag
        assert 'level="2"' in xml_tag
        assert 'position="10"' in xml_tag
    
    def test_configuration_options(self):
        """Test different configuration options."""
        # Test with entities disabled
        tagger_no_entities = XMLSemanticTagger(include_entities=False)
        element = {"element_type": "paragraph", "element_id": "test"}
        xml_tag = tagger_no_entities.generate_xml_tag(element, "revenue increased", ContextRole.MAIN)
        assert 'entities=' not in xml_tag
        
        # Test with strength disabled
        tagger_no_strength = XMLSemanticTagger(include_strength=False)
        xml_tag = tagger_no_strength.generate_xml_tag(element, "content", ContextRole.PARENT)
        assert 'strength=' not in xml_tag
        
        # Test with metadata disabled
        tagger_no_metadata = XMLSemanticTagger(include_metadata=False)
        element_with_metadata = {
            "element_type": "paragraph",
            "element_id": "test",
            "metadata": {"page_number": 5}
        }
        xml_tag = tagger_no_metadata.generate_xml_tag(element_with_metadata, "content", ContextRole.MAIN)
        assert 'page=' not in xml_tag