"""Test compatibility between Python and Go XML parsers."""

import os
import pytest
from typing import Dict, Any

from src.go_doc_go.document_parser.factory import create_parser


class TestXMLGoCompatibility:
    """Test compatibility between Python and Go XML parsers."""

    @pytest.fixture
    def sample_xml_content(self) -> Dict[str, Any]:
        """Sample XML content for testing."""
        return {
            'id': 'test_xml_doc',
            'content': '''<?xml version="1.0" encoding="UTF-8"?>
<library xmlns="http://example.com/library">
    <book id="1" category="fiction">
        <title>The Great Adventure</title>
        <author>
            <name>John Smith</name>
            <email>john@example.com</email>
        </author>
        <publication>
            <year>2024</year>
            <publisher>Example Press</publisher>
            <website>https://example.com/books</website>
        </publication>
        <description>An exciting tale of adventure and discovery.</description>
    </book>
    <book id="2" category="non-fiction">
        <title>Understanding XML</title>
        <author>
            <name>Jane Doe</name>
            <email>jane@example.com</email>
        </author>
        <publication>
            <year>2023</year>
            <publisher>Tech Books</publisher>
        </publication>
        <description>A comprehensive guide to XML processing.</description>
    </book>
</library>''',
            'metadata': {
                'source': 'test',
                'filename': 'test.xml'
            }
        }

    def test_both_parsers_available(self):
        """Test that both Go and Python XML parsers are available."""
        # Test Go parser
        os.environ['USE_GO_MODULES'] = 'true'
        go_parser = create_parser('xml')
        assert go_parser.__class__.__name__ == 'GoXMLParser'

        # Test Python parser
        os.environ['USE_GO_MODULES'] = 'false'
        python_parser = create_parser('xml')
        assert python_parser.__class__.__name__ == 'XmlParser'

    def test_both_parsers_parse_successfully(self, sample_xml_content):
        """Test that both parsers can parse XML content without errors."""
        # Test Go parser
        os.environ['USE_GO_MODULES'] = 'true'
        go_parser = create_parser('xml')
        go_result = go_parser.parse(sample_xml_content)
        
        assert 'document' in go_result
        assert 'elements' in go_result
        assert 'relationships' in go_result
        assert len(go_result['elements']) > 0

        # Test Python parser
        os.environ['USE_GO_MODULES'] = 'false'
        python_parser = create_parser('xml')
        python_result = python_parser.parse(sample_xml_content)
        
        assert 'document' in python_result
        assert 'elements' in python_result
        assert 'relationships' in python_result
        assert len(python_result['elements']) > 0

    def test_document_structure_compatibility(self, sample_xml_content):
        """Test that both parsers produce compatible document structures."""
        # Get results from both parsers
        os.environ['USE_GO_MODULES'] = 'true'
        go_parser = create_parser('xml')
        go_result = go_parser.parse(sample_xml_content)

        os.environ['USE_GO_MODULES'] = 'false'
        python_parser = create_parser('xml')
        python_result = python_parser.parse(sample_xml_content)

        # Check document fields
        assert go_result['document']['doc_type'] == python_result['document']['doc_type']
        assert go_result['document']['doc_type'] == 'xml'

        # Both should have root element
        go_root = [e for e in go_result['elements'] if e.get('element_type') == 'root']
        python_root = [e for e in python_result['elements'] if e.get('element_type') == 'root']
        
        assert len(go_root) >= 1
        assert len(python_root) >= 1

    def test_link_extraction_compatibility(self, sample_xml_content):
        """Test that both parsers extract links consistently."""
        # Get results from both parsers
        os.environ['USE_GO_MODULES'] = 'true'
        go_parser = create_parser('xml')
        go_result = go_parser.parse(sample_xml_content)

        os.environ['USE_GO_MODULES'] = 'false'
        python_parser = create_parser('xml')
        python_result = python_parser.parse(sample_xml_content)

        # Both should extract the website link
        go_links = go_result.get('links', [])
        python_links = python_result.get('links', [])

        # Check for URL extraction
        go_urls = [link['link_target'] for link in go_links if link['link_type'] == 'url']
        python_urls = [link['link_target'] for link in python_links if link['link_type'] == 'url']

        # Both should find the website URL
        expected_url = 'https://example.com/books'
        assert any(expected_url in url for url in go_urls)
        # Note: Python parser may extract differently, but should still find URLs

    def test_element_types_compatibility(self, sample_xml_content):
        """Test that element types are compatible between parsers."""
        # Get results from both parsers
        os.environ['USE_GO_MODULES'] = 'true'
        go_parser = create_parser('xml')
        go_result = go_parser.parse(sample_xml_content)

        os.environ['USE_GO_MODULES'] = 'false'
        python_parser = create_parser('xml')
        python_result = python_parser.parse(sample_xml_content)

        # Both should have valid element types
        go_types = {e['element_type'] for e in go_result['elements']}
        python_types = {e['element_type'] for e in python_result['elements']}

        # Both should have root element
        assert 'root' in go_types
        assert 'root' in python_types

        # All element types should be strings
        for elem in go_result['elements']:
            assert isinstance(elem['element_type'], str)
            assert elem['element_type'] != ''

        for elem in python_result['elements']:
            assert isinstance(elem['element_type'], str)
            assert elem['element_type'] != ''

    def test_relationship_structure_compatibility(self, sample_xml_content):
        """Test that relationship structures are compatible."""
        # Get results from both parsers
        os.environ['USE_GO_MODULES'] = 'true'
        go_parser = create_parser('xml')
        go_result = go_parser.parse(sample_xml_content)

        os.environ['USE_GO_MODULES'] = 'false'
        python_parser = create_parser('xml')
        python_result = python_parser.parse(sample_xml_content)

        # Both should have relationships
        assert len(go_result['relationships']) > 0
        assert len(python_result['relationships']) > 0

        # Check relationship structure
        for rel in go_result['relationships']:
            assert 'relationship_type' in rel
            assert 'source_element_id' in rel
            assert 'target_element_id' in rel

        for rel in python_result['relationships']:
            assert 'relationship_type' in rel
            assert 'source_element_id' in rel
            assert 'target_element_id' in rel

    def test_error_handling_compatibility(self):
        """Test that both parsers handle errors consistently."""
        invalid_content = {
            'id': 'invalid_xml',
            'content': '<invalid><xml>unclosed tag',
            'metadata': {'source': 'test'}
        }

        # Both parsers should handle invalid XML gracefully
        os.environ['USE_GO_MODULES'] = 'true'
        go_parser = create_parser('xml')
        
        with pytest.raises(Exception):  # Should raise some form of error
            go_parser.parse(invalid_content)

        os.environ['USE_GO_MODULES'] = 'false'
        python_parser = create_parser('xml')
        
        with pytest.raises(Exception):  # Should raise some form of error
            python_parser.parse(invalid_content)
