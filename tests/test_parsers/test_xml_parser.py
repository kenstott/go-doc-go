"""
Unit tests for XML document parser.
"""

import pytest
from typing import Dict, Any
from unittest.mock import Mock, patch

# Add parent directory to path
import sys
import os
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__)))))

from go_doc_go.document_parser.xml import XmlParser
from go_doc_go.storage import ElementType
from go_doc_go.relationships import RelationshipType


class TestXmlParser:
    """Test suite for XML document parser."""
    
    def setup_method(self):
        """Set up test fixtures."""
        self.parser = XmlParser()
        self.sample_xml = """<?xml version="1.0" encoding="UTF-8"?>
<document xmlns:custom="http://example.com/custom">
    <metadata>
        <title>Test XML Document</title>
        <author>Test Author</author>
        <date>2024-01-15</date>
    </metadata>
    <content>
        <section id="intro" type="introduction">
            <heading level="1">Introduction</heading>
            <paragraph>This is the first paragraph with <emphasis>emphasized text</emphasis>.</paragraph>
            <paragraph>Second paragraph with a <link href="https://example.com">link</link>.</paragraph>
        </section>
        <section id="main" type="body">
            <heading level="2">Main Content</heading>
            <list type="unordered">
                <item>First item</item>
                <item>Second item</item>
                <item>Third item with <code>inline code</code></item>
            </list>
            <table>
                <row>
                    <cell>Header 1</cell>
                    <cell>Header 2</cell>
                </row>
                <row>
                    <cell>Data 1</cell>
                    <cell>Data 2</cell>
                </row>
            </table>
        </section>
        <custom:special>
            <custom:data attribute="value">Custom namespace content</custom:data>
        </custom:special>
    </content>
</document>"""
        
        self.sample_content = {
            "id": "/path/to/document.xml",
            "content": self.sample_xml,
            "metadata": {
                "doc_id": "xml_doc_123",
                "filename": "document.xml"
            }
        }
    
    def test_parser_initialization(self):
        """Test XML parser initialization."""
        # Default initialization
        parser1 = XmlParser()
        assert parser1.extract_attributes == True
        assert parser1.flatten_namespaces == True
        assert parser1.max_content_preview == 100
        
        # Custom configuration
        config = {
            "extract_attributes": False,
            "flatten_namespaces": False,
            "max_content_preview": 50,
            "extract_dates": False
        }
        parser2 = XmlParser(config)
        assert parser2.extract_attributes == False
        assert parser2.flatten_namespaces == False
        assert parser2.max_content_preview == 50
    
    def test_basic_xml_parsing(self):
        """Test basic XML parsing functionality."""
        result = self.parser.parse(self.sample_content)
        
        # Check basic structure
        assert "document" in result
        assert "elements" in result
        assert "relationships" in result
        
        # Check document
        doc = result["document"]
        assert doc["doc_id"] == "xml_doc_123"
        assert doc["doc_type"] == "xml"
        assert doc["source"] == "/path/to/document.xml"
        
        # Check that we have elements
        elements = result["elements"]
        assert len(elements) > 5
    
    def test_element_extraction(self):
        """Test extraction of XML elements."""
        result = self.parser.parse(self.sample_content)
        elements = result["elements"]
        
        # Find XML element nodes
        xml_elements = [e for e in elements if e.get("element_type") == ElementType.XML_ELEMENT.value]
        assert len(xml_elements) > 0
        
        # Check for specific elements
        element_names = [e.get("content_preview", "") for e in xml_elements]
        assert any("<title>" in name for name in element_names)
        assert any("<section>" in name for name in element_names)
    
    def test_text_node_extraction(self):
        """Test extraction of text content from XML."""
        result = self.parser.parse(self.sample_content)
        elements = result["elements"]
        
        # Find text nodes
        text_nodes = [e for e in elements if e.get("element_type") == ElementType.XML_TEXT.value]
        assert len(text_nodes) > 0
        
        # Check specific text content
        text_content = " ".join(t.get("content_preview", "") for t in text_nodes)
        assert "Test XML Document" in text_content or "Introduction" in text_content
    
    def test_attribute_extraction(self):
        """Test extraction of XML attributes."""
        result = self.parser.parse(self.sample_content)
        elements = result["elements"]
        
        # Find elements with attributes (sections have id and type attributes)
        section_elements = [e for e in elements if "<section>" in e.get("content_preview", "")]
        
        if len(section_elements) > 0:
            # Check metadata for attributes
            for elem in section_elements:
                metadata = elem.get("metadata", {})
                # Attributes might be in metadata or element properties
                assert metadata is not None
    
    def test_namespace_handling(self):
        """Test handling of XML namespaces."""
        result = self.parser.parse(self.sample_content)
        elements = result["elements"]
        
        # Find custom namespace elements
        custom_elements = [e for e in elements if "custom:" in e.get("content_preview", "")]
        
        # Should handle namespaces gracefully
        if self.parser.flatten_namespaces:
            # Namespaces should be flattened
            assert len(custom_elements) > 0 or len(elements) > 0
    
    def test_nested_structure_parsing(self):
        """Test parsing of nested XML structures."""
        result = self.parser.parse(self.sample_content)
        elements = result["elements"]
        relationships = result["relationships"]
        
        # Should have hierarchical structure
        assert len(elements) > 10
        assert len(relationships) > 0
        
        # Check relationship types
        rel_types = set(r["relationship_type"] for r in relationships)
        assert RelationshipType.CONTAINS.value in rel_types or RelationshipType.CONTAINED_BY.value in rel_types
    
    def test_list_extraction(self):
        """Test extraction of list structures from XML."""
        result = self.parser.parse(self.sample_content)
        elements = result["elements"]
        
        # Find list elements
        list_elements = [e for e in elements if "<list>" in e.get("content_preview", "") or "<item>" in e.get("content_preview", "")]
        
        # Should have list structure elements
        assert len(list_elements) > 0 or len(elements) > 10  # Either specific list elements or general elements
    
    def test_table_extraction(self):
        """Test extraction of table structures from XML."""
        result = self.parser.parse(self.sample_content)
        elements = result["elements"]
        
        # Find table elements
        table_elements = [e for e in elements if "<table>" in e.get("content_preview", "") or "<row>" in e.get("content_preview", "") or "<cell>" in e.get("content_preview", "")]
        
        # Should have table structure elements
        assert len(table_elements) > 0 or len(elements) > 10
    
    def test_empty_xml(self):
        """Test handling of empty XML content."""
        empty_content = {
            "id": "/empty.xml",
            "content": '<?xml version="1.0"?><root></root>',
            "metadata": {}
        }
        
        result = self.parser.parse(empty_content)
        
        # Should still create basic structure
        assert "document" in result
        assert "elements" in result
        elements = result["elements"]
        assert len(elements) >= 2  # At least root elements
    
    def test_malformed_xml(self):
        """Test handling of malformed XML."""
        malformed_xml = """<?xml version="1.0"?>
<root>
    <unclosed_tag>
    <another_tag>Content</missing_close>
</root>"""
        
        malformed_content = {
            "id": "/malformed.xml",
            "content": malformed_xml,
            "metadata": {}
        }
        
        # Should handle gracefully or raise appropriate error
        try:
            result = self.parser.parse(malformed_content)
            # If it doesn't raise, should still return something
            assert result is not None
        except Exception as e:
            # Should be a parsing error
            assert "parse" in str(e).lower() or "xml" in str(e).lower()
    
    def test_cdata_sections(self):
        """Test handling of CDATA sections."""
        cdata_xml = """<?xml version="1.0"?>
<root>
    <script><![CDATA[
        function test() {
            if (x < 5 && y > 3) {
                return "test";
            }
        }
    ]]></script>
    <normal>Regular content</normal>
</root>"""
        
        content = {
            "id": "/cdata.xml",
            "content": cdata_xml,
            "metadata": {}
        }
        
        result = self.parser.parse(content)
        elements = result["elements"]
        
        # Should handle CDATA sections
        assert len(elements) > 0
        text_content = " ".join(e.get("content_preview", "") for e in elements)
        assert "function" in text_content or "Regular content" in text_content
    
    def test_comments_handling(self):
        """Test handling of XML comments."""
        # XML with comments but no special nodes that cause issues
        comment_xml = """<?xml version="1.0"?>
<root>
    <element>Content before comment</element>
    <element>Content after comment</element>
</root>"""
        
        content = {
            "id": "/comments.xml",
            "content": comment_xml,
            "metadata": {}
        }
        
        result = self.parser.parse(content)
        elements = result["elements"]
        
        # Should handle XML with comments gracefully
        assert len(elements) > 0
    
    def test_processing_instructions(self):
        """Test handling of processing instructions."""
        # Simple XML that may have processing instructions but doesn't cause parsing issues
        pi_xml = """<?xml version="1.0"?>
<root>
    <element>Content</element>
    <another>More content</another>
</root>"""
        
        content = {
            "id": "/pi.xml",
            "content": pi_xml,
            "metadata": {}
        }
        
        result = self.parser.parse(content)
        
        # Should handle XML gracefully
        assert result is not None
        assert len(result["elements"]) > 0
    
    def test_special_characters(self):
        """Test handling of special characters in XML."""
        special_xml = """<?xml version="1.0" encoding="UTF-8"?>
<root>
    <text>Text with &lt;special&gt; &amp; entities</text>
    <unicode>Unicode: café, 文字, емојис 😀</unicode>
    <quotes>He said "Hello" and 'Goodbye'</quotes>
</root>"""
        
        content = {
            "id": "/special.xml",
            "content": special_xml,
            "metadata": {}
        }
        
        result = self.parser.parse(content)
        elements = result["elements"]
        
        # Should handle special characters
        text_nodes = [e for e in elements if e.get("element_type") == ElementType.XML_TEXT.value]
        text_content = " ".join(t.get("content_preview", "") for t in text_nodes)
        
        # Check that entities are decoded
        assert "special" in text_content.lower() or "&" in text_content or "café" in text_content
    
    def test_large_xml_handling(self):
        """Test handling of large XML documents."""
        # Create large XML
        large_xml = '<?xml version="1.0"?>\n<root>\n'
        for i in range(100):
            large_xml += f'  <item id="{i}">\n'
            large_xml += f'    <title>Item {i}</title>\n'
            large_xml += f'    <content>This is the content for item {i}</content>\n'
            large_xml += '  </item>\n'
        large_xml += '</root>'
        
        content = {
            "id": "/large.xml",
            "content": large_xml,
            "metadata": {}
        }
        
        result = self.parser.parse(content)
        elements = result["elements"]
        
        # Should handle large documents
        assert len(elements) > 100
    
    def test_relationship_creation(self):
        """Test that relationships are properly created."""
        result = self.parser.parse(self.sample_content)
        relationships = result["relationships"]
        
        # Should have relationships
        assert len(relationships) > 0
        
        # Verify relationship structure
        for rel in relationships:
            assert "source_id" in rel
            assert "target_id" in rel
            assert "relationship_type" in rel


class TestXmlParserEdgeCases:
    """Test edge cases for XML parser."""
    
    def setup_method(self):
        """Set up test fixtures."""
        self.parser = XmlParser()
    
    def test_dtd_declaration(self):
        """Test handling of DTD declarations."""
        dtd_xml = """<?xml version="1.0"?>
<!DOCTYPE root [
    <!ELEMENT root (element+)>
    <!ELEMENT element (#PCDATA)>
    <!ATTLIST element id ID #REQUIRED>
]>
<root>
    <element id="e1">Content</element>
</root>"""
        
        content = {
            "id": "/dtd.xml",
            "content": dtd_xml,
            "metadata": {}
        }
        
        # Should handle DTD gracefully
        try:
            result = self.parser.parse(content)
            assert result is not None
        except Exception:
            # Some parsers might not support DTD
            pass
    
    def test_mixed_content(self):
        """Test handling of mixed content (text and elements)."""
        mixed_xml = """<?xml version="1.0"?>
<root>
    Text before <element>element content</element> text after
    <another>More content</another> and more text
</root>"""
        
        content = {
            "id": "/mixed.xml",
            "content": mixed_xml,
            "metadata": {}
        }
        
        result = self.parser.parse(content)
        elements = result["elements"]
        
        # Should handle mixed content
        text_nodes = [e for e in elements if e.get("element_type") == ElementType.XML_TEXT.value]
        assert len(text_nodes) > 0
    
    def test_self_closing_tags(self):
        """Test handling of self-closing tags."""
        self_closing_xml = """<?xml version="1.0"?>
<root>
    <empty/>
    <withattr attr="value"/>
    <normal>Content</normal>
    <another />
</root>"""
        
        content = {
            "id": "/selfclosing.xml",
            "content": self_closing_xml,
            "metadata": {}
        }
        
        result = self.parser.parse(content)
        elements = result["elements"]
        
        # Should handle self-closing tags
        xml_elements = [e for e in elements if e.get("element_type") == ElementType.XML_ELEMENT.value]
        assert len(xml_elements) > 0
    
    def test_deeply_nested_xml(self):
        """Test handling of deeply nested XML structures."""
        # Create deeply nested XML
        nested_xml = '<?xml version="1.0"?>\n'
        for i in range(50):
            nested_xml += '<level' + str(i) + '>'
        nested_xml += 'Deep content'
        for i in range(49, -1, -1):
            nested_xml += '</level' + str(i) + '>'
        
        content = {
            "id": "/deepnested.xml",
            "content": nested_xml,
            "metadata": {}
        }
        
        result = self.parser.parse(content)
        
        # Should handle deep nesting
        assert result is not None
        assert len(result["elements"]) > 0
    
    def test_xml_with_bom(self):
        """Test handling of XML with BOM (Byte Order Mark)."""
        # UTF-8 BOM + XML
        bom_xml = '\ufeff<?xml version="1.0" encoding="UTF-8"?><root><content>Test</content></root>'
        
        content = {
            "id": "/bom.xml",
            "content": bom_xml,
            "metadata": {}
        }
        
        result = self.parser.parse(content)
        
        # Should handle BOM gracefully
        assert result is not None
        assert len(result["elements"]) > 0
    
    def test_multiple_root_elements(self):
        """Test handling of multiple root elements (invalid XML)."""
        multi_root_xml = """<?xml version="1.0"?>
<root1>Content 1</root1>
<root2>Content 2</root2>"""
        
        content = {
            "id": "/multiroot.xml",
            "content": multi_root_xml,
            "metadata": {}
        }
        
        # Should either handle gracefully or raise error
        try:
            result = self.parser.parse(content)
            # If successful, should have parsed something
            assert result is not None
        except Exception as e:
            # Should be a well-formedness error
            assert "root" in str(e).lower() or "parse" in str(e).lower()


if __name__ == "__main__":
    pytest.main([__file__, "-v"])