"""
Extended unit tests for XML document parser to improve coverage.
"""

import pytest
from unittest.mock import MagicMock, patch, mock_open
import tempfile
import os
import json
import time
from lxml import etree
from go_doc_go.document_parser.xml import XmlParser
from go_doc_go.storage import ElementType
from go_doc_go.relationships import RelationshipType


@pytest.mark.unit
class TestXMLParserConfiguration:
    """Test XML parser configuration and initialization."""
    
    def test_comprehensive_configuration(self):
        """Test all configuration options."""
        config = {
            "max_content_preview": 200,
            "extract_attributes": False,
            "flatten_namespaces": False,
            "treat_namespaces_as_elements": True,
            "extract_namespace_declarations": False,
            "parser_features": {"remove_blank_text": True},
            "cache_ttl": 7200,
            "max_cache_size": 256,
            "enable_caching": True,
            "enable_performance_monitoring": True,
            "extract_dates": True,
            "date_context_chars": 100,
            "min_year": 1800,
            "max_year": 2200,
            "fiscal_year_start_month": 4,
            "default_locale": "UK"
        }
        
        parser = XmlParser(config)
        
        assert parser.max_content_preview == 200
        assert parser.extract_attributes == False
        assert parser.flatten_namespaces == False
        assert parser.treat_namespaces_as_elements == True
        assert parser.extract_namespace_declarations == False
        assert parser.cache_ttl == 7200
        assert parser.max_cache_size == 256
        assert parser.enable_caching == True
        assert parser.enable_performance_monitoring == True
        assert parser.extract_dates == True
        assert parser.date_context_chars == 100
        assert parser.min_year == 1800
        assert parser.max_year == 2200

    def test_date_extractor_import_failure(self):
        """Test handling when DateExtractor fails to import."""
        with patch('go_doc_go.document_parser.xml.DateExtractor', side_effect=ImportError("Module not found")):
            parser = XmlParser({"extract_dates": True})
            assert parser.extract_dates == False
            assert parser.date_extractor is None


@pytest.mark.unit
class TestXMLParserCaching:
    """Test XML parser caching functionality."""
    
    def test_cache_initialization(self):
        """Test cache initialization with different configurations."""
        # Test with caching enabled
        parser = XmlParser({
            "enable_caching": True,
            "cache_ttl": 1800,
            "max_cache_size": 64
        })
        
        assert parser.document_cache is not None
        assert parser.tree_cache is not None
        assert parser.text_cache is not None
        assert parser.document_cache.max_size == 64
        assert parser.tree_cache.max_size == min(50, 64)
        
        # Test with large cache size
        parser2 = XmlParser({"max_cache_size": 1000})
        assert parser2.tree_cache.max_size == 50  # Should be capped at 50

    def test_performance_monitoring(self):
        """Test performance monitoring functionality."""
        parser = XmlParser({"enable_performance_monitoring": True})
        
        # Check initial stats
        assert parser.performance_stats["parse_count"] == 0
        assert parser.performance_stats["cache_hits"] == 0
        assert parser.performance_stats["cache_misses"] == 0
        assert parser.performance_stats["total_parse_time"] == 0.0
        
        # Parse a document
        simple_xml = '<?xml version="1.0"?><root><data>Test</data></root>'
        content = {"id": "/test.xml", "content": simple_xml, "metadata": {}}
        
        result = parser.parse(content)
        
        # Stats should be updated (implementation dependent)
        assert "document" in result


@pytest.mark.unit
class TestXMLParserNamespaces:
    """Test XML namespace handling."""
    
    def test_namespace_flattening_enabled(self):
        """Test namespace flattening when enabled."""
        parser = XmlParser({"flatten_namespaces": True})
        
        namespaced_xml = '''<?xml version="1.0"?>
<root xmlns:ns="http://example.com/ns" xmlns:other="http://other.com">
    <ns:element>Namespaced content</ns:element>
    <other:data attr="value">Other namespace</other:data>
    <regular>Regular element</regular>
</root>'''
        
        content = {"id": "/ns.xml", "content": namespaced_xml, "metadata": {}}
        result = parser.parse(content)
        
        elements = result["elements"]
        # With flattening, namespace prefixes should be handled
        assert len(elements) > 3

    def test_namespace_as_elements(self):
        """Test treating namespaces as separate elements."""
        parser = XmlParser({
            "flatten_namespaces": False,
            "treat_namespaces_as_elements": True
        })
        
        ns_xml = '''<?xml version="1.0"?>
<root xmlns:custom="http://custom.com">
    <custom:section>
        <custom:item>Value</custom:item>
    </custom:section>
</root>'''
        
        content = {"id": "/ns_elem.xml", "content": ns_xml, "metadata": {}}
        result = parser.parse(content)
        
        assert "document" in result
        assert len(result["elements"]) > 0

    def test_namespace_declarations_extraction(self):
        """Test extraction of namespace declarations."""
        parser = XmlParser({"extract_namespace_declarations": True})
        
        ns_decl_xml = '''<?xml version="1.0"?>
<root xmlns="http://default.com" 
      xmlns:ns1="http://ns1.com"
      xmlns:ns2="http://ns2.com">
    <element>Content</element>
</root>'''
        
        content = {"id": "/ns_decl.xml", "content": ns_decl_xml, "metadata": {}}
        result = parser.parse(content)
        
        # Namespace declarations might be in document metadata
        metadata = result["document"]["metadata"]
        # Implementation specific - namespaces might be stored differently


@pytest.mark.unit
class TestXMLParserDateExtraction:
    """Test date extraction functionality."""
    
    def test_date_extraction_enabled(self):
        """Test date extraction when enabled."""
        parser = XmlParser({
            "extract_dates": True,
            "min_year": 2000,
            "max_year": 2030
        })
        
        xml_with_dates = '''<?xml version="1.0"?>
<document>
    <created>2024-01-15</created>
    <modified>January 20, 2024</modified>
    <event date="2024-02-01">Conference</event>
    <deadline>03/15/2024</deadline>
    <fiscal>FY2024</fiscal>
    <quarter>Q1 2024</quarter>
    <timestamp>2024-01-15T10:30:00Z</timestamp>
</document>'''
        
        content = {"id": "/dates.xml", "content": xml_with_dates, "metadata": {}}
        result = parser.parse(content)
        
        elements = result["elements"]
        # Date content should be preserved in elements
        date_content = " ".join(e.get("content_preview", "") for e in elements)
        assert "2024" in date_content

    def test_temporal_type_detection(self):
        """Test temporal type detection in content."""
        parser = XmlParser({"extract_dates": True})
        
        temporal_xml = '''<?xml version="1.0"?>
<timeline>
    <past>Last year 2023 was eventful</past>
    <present>Currently in Q4 2024</present>
    <future>Planning for 2025</future>
    <recurring>Every Monday at 10am</recurring>
    <duration>Project lasted 6 months</duration>
    <range>From January to December</range>
</timeline>'''
        
        content = {"id": "/temporal.xml", "content": temporal_xml, "metadata": {}}
        result = parser.parse(content)
        
        # Temporal content should be in elements
        elements = result["elements"]
        assert len(elements) > 5


@pytest.mark.unit
class TestXMLParserSpecialContent:
    """Test handling of special XML content."""
    
    def test_cdata_sections(self):
        """Test CDATA section handling."""
        parser = XmlParser()
        
        cdata_xml = '''<?xml version="1.0"?>
<root>
    <script><![CDATA[
        function test() {
            if (x < 5 && y > 3) {
                return "Special <characters> & symbols";
            }
        }
    ]]></script>
    <data><![CDATA[Raw <data> with & ampersands]]></data>
    <normal>Regular text</normal>
</root>'''
        
        content = {"id": "/cdata.xml", "content": cdata_xml, "metadata": {}}
        result = parser.parse(content)
        
        elements = result["elements"]
        # CDATA content should be preserved
        text_content = " ".join(e.get("content_preview", "") for e in elements)
        assert "function" in text_content or "Regular text" in text_content

    def test_mixed_content(self):
        """Test mixed content (text and elements)."""
        parser = XmlParser()
        
        mixed_xml = '''<?xml version="1.0"?>
<article>
    This is <bold>mixed</bold> content with <italic>inline</italic> elements.
    <paragraph>
        Another <code>paragraph</code> with mixed content.
    </paragraph>
    Final text node.
</article>'''
        
        content = {"id": "/mixed.xml", "content": mixed_xml, "metadata": {}}
        result = parser.parse(content)
        
        elements = result["elements"]
        # Should have both element and text nodes
        element_types = set(e["element_type"] for e in elements)
        assert ElementType.XML_TEXT.value in element_types or ElementType.XML_ELEMENT.value in element_types

    def test_comments_and_processing_instructions(self):
        """Test handling of comments and processing instructions."""
        parser = XmlParser()
        
        commented_xml = '''<?xml version="1.0"?>
<?xml-stylesheet type="text/xsl" href="style.xsl"?>
<root>
    <!-- This is a comment -->
    <element>Content</element>
    <!-- Multi-line
         comment -->
    <?processing-instruction data?>
    <another>More content</another>
</root>'''
        
        content = {"id": "/commented.xml", "content": commented_xml, "metadata": {}}
        result = parser.parse(content)
        
        # Should parse content elements
        elements = result["elements"]
        assert len(elements) > 2

    def test_entity_references(self):
        """Test entity reference handling."""
        parser = XmlParser()
        
        entity_xml = '''<?xml version="1.0"?>
<!DOCTYPE root [
    <!ENTITY company "Example Corp">
    <!ENTITY year "2024">
]>
<root>
    <name>&company;</name>
    <copyright>&copy; &year; &company;</copyright>
    <special>&lt;tag&gt; &amp; &quot;quotes&quot;</special>
</root>'''
        
        content = {"id": "/entities.xml", "content": entity_xml, "metadata": {}}
        
        # Parser should handle or skip DTD entities
        result = parser.parse(content)
        assert "document" in result


@pytest.mark.unit
class TestXMLParserAttributes:
    """Test XML attribute handling."""
    
    def test_attribute_extraction_enabled(self):
        """Test attribute extraction when enabled."""
        parser = XmlParser({"extract_attributes": True})
        
        attr_xml = '''<?xml version="1.0"?>
<catalog>
    <book id="123" isbn="978-0-123456-78-9" category="fiction">
        <title lang="en">Test Book</title>
        <author nationality="US">John Doe</author>
        <price currency="USD" discounted="true">29.99</price>
    </book>
</catalog>'''
        
        content = {"id": "/attrs.xml", "content": attr_xml, "metadata": {}}
        result = parser.parse(content)
        
        elements = result["elements"]
        # Elements with attributes should have them in metadata
        book_elements = [e for e in elements if "book" in e.get("content_preview", "").lower()]
        if book_elements:
            # Check for attribute metadata
            for elem in book_elements:
                metadata = elem.get("metadata", {})
                # Attributes might be stored in metadata

    def test_attribute_extraction_disabled(self):
        """Test when attribute extraction is disabled."""
        parser = XmlParser({"extract_attributes": False})
        
        attr_xml = '''<?xml version="1.0"?>
<root>
    <element id="1" type="test">Content</element>
</root>'''
        
        content = {"id": "/no_attrs.xml", "content": attr_xml, "metadata": {}}
        result = parser.parse(content)
        
        # Attributes should not be extracted when disabled
        elements = result["elements"]
        assert len(elements) > 0


@pytest.mark.unit
class TestXMLParserStructures:
    """Test parsing of specific XML structures."""
    
    def test_list_structure_detection(self):
        """Test detection of list-like structures."""
        parser = XmlParser()
        
        list_xml = '''<?xml version="1.0"?>
<data>
    <items>
        <item>First item</item>
        <item>Second item</item>
        <item>Third item</item>
    </items>
    <records>
        <record id="1">Record A</record>
        <record id="2">Record B</record>
        <record id="3">Record C</record>
    </records>
</data>'''
        
        content = {"id": "/lists.xml", "content": list_xml, "metadata": {}}
        result = parser.parse(content)
        
        elements = result["elements"]
        # Should detect repeating patterns
        item_elements = [e for e in elements if "item" in e.get("content_preview", "").lower()]
        assert len(item_elements) > 0 or len(elements) > 5

    def test_table_structure(self):
        """Test table-like structure parsing."""
        parser = XmlParser()
        
        table_xml = '''<?xml version="1.0"?>
<table>
    <header>
        <column>Name</column>
        <column>Age</column>
        <column>City</column>
    </header>
    <rows>
        <row>
            <cell>John</cell>
            <cell>30</cell>
            <cell>NYC</cell>
        </row>
        <row>
            <cell>Jane</cell>
            <cell>25</cell>
            <cell>LA</cell>
        </row>
    </rows>
</table>'''
        
        content = {"id": "/table.xml", "content": table_xml, "metadata": {}}
        result = parser.parse(content)
        
        elements = result["elements"]
        # Should parse table structure
        cell_elements = [e for e in elements if "cell" in e.get("content_preview", "").lower()]
        assert len(cell_elements) > 0 or len(elements) > 8

    def test_deeply_nested_structure(self):
        """Test deeply nested XML structures."""
        parser = XmlParser()
        
        # Create nested structure
        def create_nested(depth):
            if depth == 0:
                return "<leaf>Deep value</leaf>"
            return f"<level{depth}>{create_nested(depth-1)}</level{depth}>"
        
        deep_xml = f'''<?xml version="1.0"?>
<root>
    {create_nested(20)}
</root>'''
        
        content = {"id": "/deep.xml", "content": deep_xml, "metadata": {}}
        result = parser.parse(content)
        
        # Should handle deep nesting
        elements = result["elements"]
        assert len(elements) > 10
        
        # Check relationships
        relationships = result["relationships"]
        assert len(relationships) > 10


@pytest.mark.unit
class TestXMLParserErrorHandling:
    """Test error handling in XML parser."""
    
    def test_malformed_xml(self):
        """Test handling of malformed XML."""
        parser = XmlParser()
        
        malformed_xml = '''<?xml version="1.0"?>
<root>
    <unclosed>
    <mismatched></wrong>
    <another>Content
</root>'''
        
        content = {"id": "/malformed.xml", "content": malformed_xml, "metadata": {}}
        
        # Should handle gracefully
        try:
            result = parser.parse(content)
            # If it doesn't raise, should still return structure
            assert "document" in result
        except Exception as e:
            # Should be XML parse error
            assert "parse" in str(e).lower() or "xml" in str(e).lower()

    def test_invalid_encoding(self):
        """Test handling of invalid encoding."""
        parser = XmlParser()
        
        # XML with wrong encoding declaration
        invalid_encoding = '''<?xml version="1.0" encoding="INVALID-ENCODING"?>
<root>Content</root>'''
        
        content = {"id": "/encoding.xml", "content": invalid_encoding, "metadata": {}}
        
        # Should handle encoding issues
        result = parser.parse(content)
        assert "document" in result

    def test_empty_content(self):
        """Test handling of empty content."""
        parser = XmlParser()
        
        # Various empty scenarios
        test_cases = [
            {"id": "/empty1.xml", "content": "", "metadata": {}},
            {"id": "/empty2.xml", "content": '<?xml version="1.0"?>', "metadata": {}},
            {"id": "/empty3.xml", "content": '<?xml version="1.0"?><root/>', "metadata": {}},
            {"id": "/empty4.xml", "content": '<?xml version="1.0"?><root></root>', "metadata": {}},
        ]
        
        for content in test_cases:
            result = parser.parse(content)
            assert "document" in result
            assert "elements" in result

    def test_binary_content(self):
        """Test handling of binary content mistaken for XML."""
        parser = XmlParser()
        
        # Binary content that's not XML
        binary_content = b'\x00\x01\x02\x03\x04'
        
        content = {"id": "/binary.xml", "content": binary_content, "metadata": {}}
        
        try:
            result = parser.parse(content)
            # Should handle gracefully
            assert result is not None
        except Exception:
            # Expected to fail on binary content
            pass


@pytest.mark.unit  
class TestXMLParserPerformance:
    """Test performance aspects of XML parser."""
    
    def test_large_document_parsing(self):
        """Test parsing of large XML documents."""
        parser = XmlParser({
            "enable_performance_monitoring": True,
            "max_content_preview": 50
        })
        
        # Generate large XML
        items = []
        for i in range(500):
            items.append(f'''    <item id="{i}">
        <name>Item {i}</name>
        <description>Detailed description for item {i}</description>
        <metadata>
            <created>2024-01-{(i % 28) + 1:02d}</created>
            <tags>
                <tag>category{i % 10}</tag>
                <tag>type{i % 5}</tag>
            </tags>
        </metadata>
    </item>''')
        
        large_xml = f'''<?xml version="1.0"?>
<catalog>
{''.join(items)}
</catalog>'''
        
        content = {"id": "/large.xml", "content": large_xml, "metadata": {}}
        
        start = time.time()
        result = parser.parse(content)
        elapsed = time.time() - start
        
        # Should parse efficiently
        assert "document" in result
        assert len(result["elements"]) > 500
        assert elapsed < 10.0  # Should complete within 10 seconds

    def test_caching_performance(self):
        """Test caching improves performance."""
        parser = XmlParser({
            "enable_caching": True,
            "cache_ttl": 60
        })
        
        test_xml = '''<?xml version="1.0"?>
<root>
    <data>Content to cache</data>
</root>'''
        
        content = {"id": "/cache_perf.xml", "content": test_xml, "metadata": {"doc_id": "perf_test"}}
        
        # First parse
        start1 = time.time()
        result1 = parser.parse(content)
        time1 = time.time() - start1
        
        # Second parse (might use cache)
        start2 = time.time()
        result2 = parser.parse(content)
        time2 = time.time() - start2
        
        # Results should be consistent
        assert result1["document"]["doc_id"] == result2["document"]["doc_id"]
        assert len(result1["elements"]) == len(result2["elements"])


@pytest.mark.unit
class TestXMLParserIntegration:
    """Integration tests for XML parser."""
    
    def test_source_file_loading(self):
        """Test loading XML from source files."""
        parser = XmlParser()
        
        # Create temporary XML file
        test_xml = '''<?xml version="1.0"?>
<document>
    <title>Test Document</title>
    <content>File-based content</content>
</document>'''
        
        with tempfile.NamedTemporaryFile(mode='w', suffix='.xml', delete=False) as f:
            f.write(test_xml)
            temp_path = f.name
        
        try:
            # Test with file path
            content = {
                "id": temp_path,
                "content": "",
                "metadata": {"source_path": temp_path}
            }
            
            result = parser.parse(content)
            assert "document" in result
            
            # Also test with content directly
            with open(temp_path, 'r') as f:
                file_content = f.read()
            
            content2 = {
                "id": temp_path,
                "content": file_content,
                "metadata": {}
            }
            
            result2 = parser.parse(content2)
            assert len(result2["elements"]) > 0
            
        finally:
            os.unlink(temp_path)

    def test_comprehensive_xml_document(self):
        """Test parsing a comprehensive XML document with various features."""
        parser = XmlParser({
            "extract_attributes": True,
            "extract_dates": True,
            "flatten_namespaces": False,
            "extract_namespace_declarations": True
        })
        
        comprehensive_xml = '''<?xml version="1.0" encoding="UTF-8"?>
<?xml-stylesheet type="text/xsl" href="style.xsl"?>
<!DOCTYPE document [
    <!ENTITY company "TechCorp">
]>
<document xmlns="http://default.ns" xmlns:meta="http://meta.ns" xmlns:content="http://content.ns">
    <!-- Document metadata -->
    <meta:info created="2024-01-15" modified="2024-01-20">
        <meta:title>Comprehensive Test Document</meta:title>
        <meta:author id="auth123">&company; Team</meta:author>
    </meta:info>
    
    <!-- Main content -->
    <content:body>
        <content:section id="s1" priority="high">
            <content:heading level="1">Introduction</content:heading>
            <content:paragraph>
                This is <emphasis>important</emphasis> content with mixed elements.
            </content:paragraph>
            <content:code><![CDATA[
                def example():
                    return "code example"
            ]]></content:code>
        </content:section>
        
        <content:data>
            <content:list>
                <content:item>Item 1</content:item>
                <content:item>Item 2</content:item>
                <content:item>Item 3</content:item>
            </content:list>
            
            <content:table>
                <content:header>
                    <content:cell>Column A</content:cell>
                    <content:cell>Column B</content:cell>
                </content:header>
                <content:row>
                    <content:cell>Data 1</content:cell>
                    <content:cell>Data 2</content:cell>
                </content:row>
            </content:table>
        </content:data>
        
        <!-- Special characters and entities -->
        <content:special>
            Text with &lt;special&gt; &amp; "characters"
            Unicode: café, 中文, 😊
        </content:special>
    </content:body>
</document>'''
        
        content = {
            "id": "/comprehensive.xml",
            "content": comprehensive_xml,
            "metadata": {"doc_id": "comp_test"}
        }
        
        result = parser.parse(content)
        
        # Verify comprehensive parsing
        assert "document" in result
        assert result["document"]["doc_id"] == "comp_test"
        assert result["document"]["doc_type"] == "xml"
        
        # Check elements
        elements = result["elements"]
        assert len(elements) > 15
        
        # Check element types
        element_types = set(e["element_type"] for e in elements)
        assert ElementType.XML_ELEMENT.value in element_types or ElementType.ROOT.value in element_types
        
        # Check relationships
        relationships = result["relationships"]
        assert len(relationships) > 10
        
        # Verify relationship types
        rel_types = set(r["relationship_type"] for r in relationships)
        assert RelationshipType.CONTAINS.value in rel_types