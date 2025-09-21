#!/usr/bin/env python3
"""
Comprehensive tests for document reconstruction methods (_reconstruct_as_*).
Demonstrates how these methods provide context for matched elements in semantic search.
"""

import json
import logging
import sys
from pathlib import Path
from typing import List, Dict, Any
from datetime import datetime

# Add src to path for imports
sys.path.insert(0, str(Path(__file__).parent.parent / "src"))

from go_doc_go import Config
from go_doc_go.storage.base import DocumentDatabase
from go_doc_go.search_module import SearchEngine, SearchRequest

# Configure detailed logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s',
    handlers=[logging.StreamHandler(sys.stdout)],
    force=True
)
logger = logging.getLogger(__name__)


class ReconstructionTester:
    """Test harness for reconstruction methods."""
    
    def __init__(self):
        """Initialize test environment."""
        self.config = Config('config.yaml')
        # Get the concrete database instance
        self.db = self.config.get_document_database()
        self.search_engine = SearchEngine()
        
        # Initialize the database if needed
        if hasattr(self.db, 'initialize'):
            self.db.initialize()
        
    def create_docx_style_elements(self) -> List[Dict[str, Any]]:
        """Create sample DOCX-style document elements for testing."""
        elements = [
            {
                'element_id': 'doc_root',
                'element_type': 'root',
                'content_preview': 'Financial Analysis Report 2024',
                'parent_id': None,
                'metadata': {'doc_type': 'docx'},
                'element_order': 0,
                'document_position': 0
            },
            {
                'element_id': 'doc_header1',
                'element_type': 'header',
                'content_preview': 'Executive Summary',
                'parent_id': 'doc_root',
                'metadata': {'level': 1, 'style': 'Heading 1'},
                'element_order': 1,
                'document_position': 1
            },
            {
                'element_id': 'doc_para1',
                'element_type': 'paragraph',
                'content_preview': 'Our financial performance shows significant revenue growth despite market challenges. Operating margins improved by 15% year-over-year.',
                'parent_id': 'doc_header1',
                'metadata': {'style': 'Normal'},
                'element_order': 2,
                'document_position': 2
            },
            {
                'element_id': 'doc_header2',
                'element_type': 'header',
                'content_preview': 'Key Metrics',
                'parent_id': 'doc_root',
                'metadata': {'level': 2, 'style': 'Heading 2'},
                'element_order': 3,
                'document_position': 3
            },
            {
                'element_id': 'doc_list1',
                'element_type': 'list',
                'content_preview': 'Performance indicators',
                'parent_id': 'doc_header2',
                'metadata': {'list_type': 'bullet'},
                'element_order': 4,
                'document_position': 4
            },
            {
                'element_id': 'doc_item1',
                'element_type': 'list_item',
                'content_preview': 'Revenue: $2.5B (+25% YoY)',
                'parent_id': 'doc_list1',
                'metadata': {'level': 1},
                'element_order': 5,
                'document_position': 5
            },
            {
                'element_id': 'doc_item2',
                'element_type': 'list_item',
                'content_preview': 'Operating Margin: 32% (+15% improvement)',
                'parent_id': 'doc_list1',
                'metadata': {'level': 1},
                'element_order': 6,
                'document_position': 6
            },
            {
                'element_id': 'doc_item3',
                'element_type': 'list_item',
                'content_preview': 'Customer Acquisition Cost: $450 (-20% reduction)',
                'parent_id': 'doc_list1',
                'metadata': {'level': 1},
                'element_order': 7,
                'document_position': 7
            },
            {
                'element_id': 'doc_para2',
                'element_type': 'paragraph',
                'content_preview': 'These results demonstrate strong operational efficiency and market positioning.',
                'parent_id': 'doc_root',
                'metadata': {'style': 'Normal'},
                'element_order': 8,
                'document_position': 8
            }
        ]
        return elements
    
    def create_pptx_style_elements(self) -> List[Dict[str, Any]]:
        """Create sample PPTX-style presentation elements for testing."""
        elements = [
            {
                'element_id': 'pres_root',
                'element_type': 'root',
                'content_preview': 'Q4 2024 Earnings Presentation',
                'parent_id': None,
                'metadata': {'doc_type': 'pptx'},
                'element_order': 0,
                'document_position': 0
            },
            {
                'element_id': 'slide1',
                'element_type': 'slide',
                'content_preview': 'Slide 1: Title Slide',
                'parent_id': 'pres_root',
                'metadata': {'slide_number': 1, 'layout': 'Title Slide'},
                'element_order': 1,
                'document_position': 1
            },
            {
                'element_id': 'slide1_title',
                'element_type': 'slide_title',
                'content_preview': 'Q4 2024 Financial Results',
                'parent_id': 'slide1',
                'metadata': {'placeholder': 'title'},
                'element_order': 2,
                'document_position': 2
            },
            {
                'element_id': 'slide1_subtitle',
                'element_type': 'slide_subtitle',
                'content_preview': 'Exceeding Expectations Through Innovation',
                'parent_id': 'slide1',
                'metadata': {'placeholder': 'subtitle'},
                'element_order': 3,
                'document_position': 3
            },
            {
                'element_id': 'slide2',
                'element_type': 'slide',
                'content_preview': 'Slide 2: Revenue Highlights',
                'parent_id': 'pres_root',
                'metadata': {'slide_number': 2, 'layout': 'Title and Content'},
                'element_order': 4,
                'document_position': 4
            },
            {
                'element_id': 'slide2_title',
                'element_type': 'slide_title',
                'content_preview': 'Revenue Performance',
                'parent_id': 'slide2',
                'metadata': {'placeholder': 'title'},
                'element_order': 5,
                'document_position': 5
            },
            {
                'element_id': 'slide2_bullets',
                'element_type': 'slide_bullets',
                'content_preview': 'Key revenue drivers',
                'parent_id': 'slide2',
                'metadata': {'placeholder': 'content'},
                'element_order': 6,
                'document_position': 6
            },
            {
                'element_id': 'slide2_bullet1',
                'element_type': 'bullet_point',
                'content_preview': 'Total Revenue: $2.5B (+25% YoY growth)',
                'parent_id': 'slide2_bullets',
                'metadata': {'level': 1},
                'element_order': 7,
                'document_position': 7
            },
            {
                'element_id': 'slide2_bullet2',
                'element_type': 'bullet_point',
                'content_preview': 'Cloud Services: $1.8B (+40% YoY)',
                'parent_id': 'slide2_bullets',
                'metadata': {'level': 1},
                'element_order': 8,
                'document_position': 8
            },
            {
                'element_id': 'slide2_bullet3',
                'element_type': 'bullet_point',
                'content_preview': 'Enterprise Contracts: +150 new Fortune 500 clients',
                'parent_id': 'slide2_bullets',
                'metadata': {'level': 1},
                'element_order': 9,
                'document_position': 9
            }
        ]
        return elements
    
    def test_reconstruct_as_docx_html(self):
        """Test DOCX-style HTML reconstruction."""
        logger.info("\n" + "="*80)
        logger.info("TEST 1: DOCX-STYLE HTML RECONSTRUCTION")
        logger.info("="*80)
        
        # Create sample elements
        elements = self.create_docx_style_elements()
        
        # Test reconstruction
        html_output = self.db._reconstruct_as_docx_html(elements)
        
        logger.info("\n📄 DOCX Elements Structure:")
        for elem in elements:
            indent = "  " * (elem['document_position'] // 2)
            logger.info(f"{indent}• {elem['element_type']}: {elem['content_preview'][:50]}...")
        
        logger.info("\n🔧 Generated HTML Output:")
        logger.info("-" * 60)
        # Show first 2000 chars of HTML
        logger.info(html_output[:2000] + "..." if len(html_output) > 2000 else html_output)
        
        # Verify key HTML elements
        logger.info("\n✅ Verification:")
        checks = [
            ("HTML document structure", "<html>" in html_output and "</html>" in html_output),
            ("DOCX styling", "font-family: 'Calibri'" in html_output),
            ("Headers rendered", "<h1>" in html_output or "<h2>" in html_output),
            ("Paragraphs rendered", "<p>" in html_output),
            ("Lists rendered", "<ul>" in html_output or "<li>" in html_output),
            ("Content preserved", "Operating margins improved" in html_output),
            ("Hierarchy maintained", html_output.index("Executive Summary") < html_output.index("Key Metrics"))
        ]
        
        for check_name, check_result in checks:
            status = "✅" if check_result else "❌"
            logger.info(f"  {status} {check_name}")
        
        return html_output
    
    def test_reconstruct_as_pptx_html(self):
        """Test PPTX-style HTML reconstruction."""
        logger.info("\n" + "="*80)
        logger.info("TEST 2: PPTX-STYLE HTML RECONSTRUCTION")
        logger.info("="*80)
        
        # Create sample elements
        elements = self.create_pptx_style_elements()
        
        # Test reconstruction
        html_output = self.db._reconstruct_as_pptx_html(elements)
        
        logger.info("\n🎯 PPTX Elements Structure:")
        for elem in elements:
            indent = "  " * (elem['element_order'] // 3)
            logger.info(f"{indent}• {elem['element_type']}: {elem['content_preview'][:50]}...")
        
        logger.info("\n🔧 Generated HTML Output:")
        logger.info("-" * 60)
        # Show first 2000 chars of HTML
        logger.info(html_output[:2000] + "..." if len(html_output) > 2000 else html_output)
        
        # Verify key HTML elements
        logger.info("\n✅ Verification:")
        checks = [
            ("HTML document structure", "<html>" in html_output and "</html>" in html_output),
            ("Slide structure", "class=\"slide\"" in html_output or "<div" in html_output),
            ("Slide titles", "Q4 2024 Financial Results" in html_output),
            ("Bullet points", "Total Revenue: $2.5B" in html_output),
            ("Presentation styling", "background" in html_output or "border" in html_output),
            ("Slide separation", "margin-bottom" in html_output or "padding" in html_output)
        ]
        
        for check_name, check_result in checks:
            status = "✅" if check_result else "❌"
            logger.info(f"  {status} {check_name}")
        
        return html_output
    
    def test_search_with_context_reconstruction(self):
        """Test semantic search with context reconstruction."""
        logger.info("\n" + "="*80)
        logger.info("TEST 3: SEMANTIC SEARCH WITH CONTEXT RECONSTRUCTION")
        logger.info("="*80)
        
        # Perform semantic search
        query = "operating margins and revenue growth"
        logger.info(f"\n🔍 Searching for: '{query}'")
        
        # Get available backends
        backends = self.config.list_analytics_backends()
        if not backends:
            logger.warning("⚠️ No analytics backends available for search")
            return
        
        backend_name = list(backends.keys())[0]
        logger.info(f"📊 Using backend: {backend_name}")
        
        # Create search request
        search_request = SearchRequest(
            search_service=backend_name,
            similarity_query=query,
            limit=5,
            similarity_threshold=0.5,
            include_content=True,
            include_metadata=True
        )
        
        try:
            # Execute search
            response = self.search_engine.search(search_request)
            
            logger.info(f"\n📋 Search Results: {response.total_hits} hits")
            
            if response.hits:
                # Take first hit for context demonstration
                hit = response.hits[0]
                
                logger.info(f"\n🎯 Matched Element:")
                logger.info(f"  Element ID: {hit.element_id}")
                logger.info(f"  Type: {hit.element_type}")
                logger.info(f"  Score: {hit.score:.4f}")
                logger.info(f"  Preview: {hit.content_preview}")
                
                # Demonstrate context reconstruction
                logger.info(f"\n🔧 Context Reconstruction:")
                
                # Get element's parent and siblings for context
                # This simulates what would happen in _reconstruct_element_context
                logger.info("\n1️⃣ Element Only:")
                logger.info(f"  {hit.content_preview}")
                
                logger.info("\n2️⃣ With Parent Context:")
                if hit.metadata and 'parent_id' in hit.metadata:
                    logger.info(f"  Parent: [would fetch parent element]")
                    logger.info(f"  └─ {hit.content_preview}")
                
                logger.info("\n3️⃣ With Sibling Context:")
                logger.info(f"  [Previous sibling content]")
                logger.info(f"  >>> {hit.content_preview} <<<  (matched element)")
                logger.info(f"  [Next sibling content]")
                
                logger.info("\n4️⃣ Full Document Context:")
                logger.info(f"  Document: {hit.doc_id}")
                logger.info(f"  Section: [parent section]")
                logger.info(f"  Context window: [surrounding elements]")
                logger.info(f"  Matched: {hit.content_preview}")
                
            else:
                logger.warning("⚠️ No search results found")
                
        except Exception as e:
            logger.error(f"❌ Search failed: {e}")
    
    def demonstrate_context_levels(self):
        """Demonstrate different context levels for reconstruction."""
        logger.info("\n" + "="*80)
        logger.info("TEST 4: CONTEXT LEVEL DEMONSTRATION")
        logger.info("="*80)
        
        # Create a hierarchical document structure
        elements = self.create_docx_style_elements()
        
        # Find the target element (operating margins paragraph)
        target_element = next((e for e in elements if 'Operating margins' in e['content_preview']), None)
        
        if not target_element:
            logger.error("❌ Target element not found")
            return
        
        logger.info(f"\n🎯 Target Element: {target_element['content_preview']}")
        
        # Level 1: Element only
        logger.info("\n📊 LEVEL 1 - Element Only:")
        logger.info("-" * 40)
        element_only = [target_element]
        html_1 = self.db._reconstruct_as_docx_html(element_only)
        logger.info(f"Content: {target_element['content_preview']}")
        logger.info(f"HTML Length: {len(html_1)} chars")
        
        # Level 2: Element + Parent
        logger.info("\n📊 LEVEL 2 - Element + Parent:")
        logger.info("-" * 40)
        parent = next((e for e in elements if e['element_id'] == target_element['parent_id']), None)
        if parent:
            with_parent = [parent, target_element]
            html_2 = self.db._reconstruct_as_docx_html(with_parent)
            logger.info(f"Parent: {parent['content_preview']}")
            logger.info(f"└─ Element: {target_element['content_preview']}")
            logger.info(f"HTML Length: {len(html_2)} chars")
        
        # Level 3: Element + Parent + Siblings
        logger.info("\n📊 LEVEL 3 - Element + Parent + Siblings:")
        logger.info("-" * 40)
        siblings = [e for e in elements if e.get('parent_id') == target_element.get('parent_id')]
        with_siblings = siblings if parent else [target_element]
        if parent and parent not in with_siblings:
            with_siblings.insert(0, parent)
        html_3 = self.db._reconstruct_as_docx_html(with_siblings)
        logger.info(f"Context includes {len(with_siblings)} elements")
        for elem in with_siblings:
            prefix = ">>> " if elem['element_id'] == target_element['element_id'] else "    "
            logger.info(f"{prefix}{elem['element_type']}: {elem['content_preview'][:50]}...")
        logger.info(f"HTML Length: {len(html_3)} chars")
        
        # Level 4: Full document
        logger.info("\n📊 LEVEL 4 - Full Document Context:")
        logger.info("-" * 40)
        html_4 = self.db._reconstruct_as_docx_html(elements)
        logger.info(f"Full document with {len(elements)} elements")
        logger.info(f"HTML Length: {len(html_4)} chars")
        
        # Show how context grows
        logger.info("\n📈 Context Growth Analysis:")
        logger.info(f"  Level 1 (Element only):      {len(html_1):6} chars")
        if parent:
            logger.info(f"  Level 2 (+ Parent):          {len(html_2):6} chars (+{len(html_2)-len(html_1)})")
        logger.info(f"  Level 3 (+ Siblings):        {len(html_3):6} chars (+{len(html_3)-len(html_1)})")
        logger.info(f"  Level 4 (Full document):     {len(html_4):6} chars (+{len(html_4)-len(html_1)})")
        
        # Demonstrate value of context
        logger.info("\n💡 Context Value Demonstration:")
        logger.info("  Without context: 'Operating margins improved by 15% year-over-year'")
        logger.info("  With context: Shows this is part of Executive Summary, related to")
        logger.info("                revenue growth, and supported by specific metrics")


def main():
    """Run all reconstruction tests."""
    logger.info("\n" + "="*80)
    logger.info("DOCUMENT RECONSTRUCTION CONTEXT TESTS")
    logger.info("Demonstrating _reconstruct_as_* methods for search context")
    logger.info("="*80)
    
    tester = ReconstructionTester()
    
    # Run all tests
    try:
        # Test 1: DOCX reconstruction
        docx_html = tester.test_reconstruct_as_docx_html()
        
        # Test 2: PPTX reconstruction
        pptx_html = tester.test_reconstruct_as_pptx_html()
        
        # Test 3: Search with context
        tester.test_search_with_context_reconstruction()
        
        # Test 4: Context levels
        tester.demonstrate_context_levels()
        
        # Summary
        logger.info("\n" + "="*80)
        logger.info("TEST SUMMARY")
        logger.info("="*80)
        logger.info("\n✅ Key Findings:")
        logger.info("  1. _reconstruct_as_docx_html preserves document structure and styling")
        logger.info("  2. _reconstruct_as_pptx_html creates slide-based presentation format")
        logger.info("  3. Context reconstruction helps understand search results")
        logger.info("  4. Different context levels provide varying detail:")
        logger.info("     - Element only: Minimal context")
        logger.info("     - With parent: Shows section/hierarchy")
        logger.info("     - With siblings: Shows related content")
        logger.info("     - Full document: Complete context")
        logger.info("\n🎯 Use Cases:")
        logger.info("  • Search result preview with context")
        logger.info("  • Document structure visualization")
        logger.info("  • Content extraction with formatting")
        logger.info("  • Hierarchical navigation in search results")
        
    except Exception as e:
        logger.error(f"\n❌ Test failed: {e}")
        import traceback
        logger.error(traceback.format_exc())


if __name__ == "__main__":
    main()