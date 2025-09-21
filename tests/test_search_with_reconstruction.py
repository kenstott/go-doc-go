#!/usr/bin/env python3
"""
Enhanced semantic search test with document reconstruction.
Shows how search results can be augmented with reconstructed HTML context.
"""

import logging
import sys
import json
from pathlib import Path
from typing import Optional, Dict, Any, List
import html

# Add the src directory to Python path
sys.path.insert(0, str(Path(__file__).parent.parent / "src"))

from go_doc_go import Config
from go_doc_go.search_module import SearchEngine, SearchRequest
from go_doc_go.storage import DocumentDatabase

# Set up logging for real-time output
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s',
    handlers=[logging.StreamHandler(sys.stdout)],
    force=True
)
logger = logging.getLogger(__name__)


class SearchWithReconstruction:
    """Enhanced search with document reconstruction capabilities."""
    
    def __init__(self):
        """Initialize search engine and database connection."""
        self.config = Config('config.yaml')
        self.search_engine = SearchEngine()
        self.db = self.config.get_document_database()
        
        # Initialize database if needed
        if hasattr(self.db, 'initialize'):
            self.db.initialize()
    
    def get_element_hierarchy(self, element_id: str, doc_id: str) -> Dict[str, Any]:
        """
        Get element with its parent and sibling context.
        
        Args:
            element_id: The element ID to get hierarchy for
            doc_id: The document ID containing the element
            
        Returns:
            Dict with element, parent, and siblings information
        """
        try:
            # Get the main element
            cursor = self.db.conn.cursor()
            cursor.execute("""
                SELECT element_id, element_type, parent_id, content_preview, 
                       element_order, document_position, metadata
                FROM elements 
                WHERE element_id = ? AND doc_id = ?
            """, (element_id, doc_id))
            
            element = cursor.fetchone()
            if not element:
                return None
            
            result = {
                'element': dict(element),
                'parent': None,
                'siblings': []
            }
            
            # Get parent if exists
            if element['parent_id']:
                cursor.execute("""
                    SELECT element_id, element_type, parent_id, content_preview,
                           element_order, document_position, metadata
                    FROM elements 
                    WHERE element_id = ? AND doc_id = ?
                """, (element['parent_id'], doc_id))
                
                parent = cursor.fetchone()
                if parent:
                    result['parent'] = dict(parent)
                    
                    # Get siblings (other children of same parent)
                    cursor.execute("""
                        SELECT element_id, element_type, parent_id, content_preview,
                               element_order, document_position, metadata
                        FROM elements 
                        WHERE parent_id = ? AND doc_id = ? AND element_id != ?
                        ORDER BY element_order, document_position
                    """, (element['parent_id'], doc_id, element_id))
                    
                    siblings = cursor.fetchall()
                    result['siblings'] = [dict(s) for s in siblings]
            
            return result
            
        except Exception as e:
            logger.error(f"Error getting element hierarchy: {e}")
            return None
    
    def reconstruct_with_context(self, hit, reconstruction_format: str = 'docx', 
                                context_level: str = 'with_parent') -> Optional[str]:
        """
        Reconstruct HTML for a search hit with specified context level.
        
        Args:
            hit: Search result hit object
            reconstruction_format: 'docx' or 'pptx'
            context_level: 'element_only', 'with_parent', 'with_siblings', or 'full_document'
            
        Returns:
            Reconstructed HTML string or None
        """
        try:
            # Get element hierarchy
            hierarchy = self.get_element_hierarchy(hit.element_id, hit.doc_id)
            if not hierarchy:
                return None
            
            # Build element list based on context level
            elements_to_reconstruct = []
            
            if context_level == 'element_only':
                elements_to_reconstruct = [hierarchy['element']]
                
            elif context_level == 'with_parent':
                if hierarchy['parent']:
                    elements_to_reconstruct = [hierarchy['parent'], hierarchy['element']]
                else:
                    elements_to_reconstruct = [hierarchy['element']]
                    
            elif context_level == 'with_siblings':
                if hierarchy['parent']:
                    elements_to_reconstruct.append(hierarchy['parent'])
                elements_to_reconstruct.append(hierarchy['element'])
                elements_to_reconstruct.extend(hierarchy['siblings'])
                # Sort by position
                elements_to_reconstruct.sort(key=lambda x: (x.get('document_position', 0), x.get('element_order', 0)))
                
            elif context_level == 'full_document':
                # Get all elements from document
                cursor = self.db.conn.cursor()
                cursor.execute("""
                    SELECT element_id, element_type, parent_id, content_preview,
                           element_order, document_position, metadata
                    FROM elements 
                    WHERE doc_id = ?
                    ORDER BY document_position, element_order
                """, (hit.doc_id,))
                
                elements_to_reconstruct = [dict(row) for row in cursor.fetchall()]
            
            # Call the appropriate reconstruction method
            if reconstruction_format == 'docx':
                return self.db._reconstruct_as_docx_html(elements_to_reconstruct, hit.element_id)
            elif reconstruction_format == 'pptx':
                return self.db._reconstruct_as_pptx_html(elements_to_reconstruct, hit.element_id)
            else:
                logger.warning(f"Unknown reconstruction format: {reconstruction_format}")
                return None
                
        except Exception as e:
            logger.error(f"Error reconstructing with context: {e}")
            return None
    
    def search_with_reconstruction(self, query: str, limit: int = 5, 
                                  threshold: float = 0.7,
                                  reconstruction_format: str = 'docx',
                                  context_level: str = 'with_parent') -> Dict[str, Any]:
        """
        Perform search and augment results with reconstructed HTML context.
        
        Args:
            query: Search query string
            limit: Maximum number of results
            threshold: Similarity threshold
            reconstruction_format: 'docx' or 'pptx'
            context_level: Level of context to include
            
        Returns:
            Enhanced search results with reconstruction
        """
        logger.info(f"🔍 SEARCH WITH RECONSTRUCTION: '{query}'")
        logger.info("=" * 60)
        
        # Get available backends
        backends = self.config.list_analytics_backends()
        if not backends:
            logger.error("❌ No analytics backends configured")
            return None
        
        backend_name = list(backends.keys())[0]
        logger.info(f"📊 Using backend: {backend_name}")
        logger.info(f"🎨 Reconstruction format: {reconstruction_format}")
        logger.info(f"📐 Context level: {context_level}")
        
        # Create and execute search request
        search_request = SearchRequest(
            search_service=backend_name,
            similarity_query=query,
            limit=limit,
            similarity_threshold=threshold,
            include_content=True,
            include_metadata=True
        )
        
        response = self.search_engine.search(search_request)
        
        # Augment results with reconstruction
        enhanced_results = {
            'query': query,
            'total_hits': response.total_hits,
            'took_ms': response.took_ms,
            'results': []
        }
        
        logger.info(f"\n📋 SEARCH RESULTS WITH RECONSTRUCTION")
        logger.info(f"Total hits: {response.total_hits}")
        logger.info("-" * 60)
        
        for i, hit in enumerate(response.hits, 1):
            logger.info(f"\n🔖 Result #{i}")
            logger.info(f"   Score: {hit.score:.4f}")
            logger.info(f"   Element: {hit.element_type} - {hit.element_id[:20]}...")
            logger.info(f"   Preview: {hit.content_preview[:100]}...")
            
            # Get reconstructed HTML
            reconstructed_html = self.reconstruct_with_context(
                hit, 
                reconstruction_format, 
                context_level
            )
            
            # Create enhanced result
            enhanced_result = {
                'rank': i,
                'score': hit.score,
                'element_id': hit.element_id,
                'doc_id': hit.doc_id,
                'element_type': hit.element_type,
                'content_preview': hit.content_preview,
                'metadata': hit.metadata,
                'reconstructed_html': reconstructed_html,
                'reconstruction_format': reconstruction_format,
                'context_level': context_level
            }
            
            if reconstructed_html:
                logger.info(f"   ✅ Reconstruction: {len(reconstructed_html)} chars")
                
                # Show a snippet of the reconstructed HTML
                clean_html = reconstructed_html.replace('\n', ' ')
                snippet = clean_html[:200] + '...' if len(clean_html) > 200 else clean_html
                logger.info(f"   📄 HTML snippet: {snippet}")
            else:
                logger.info(f"   ⚠️  Reconstruction failed")
            
            enhanced_results['results'].append(enhanced_result)
        
        return enhanced_results


def test_search_with_reconstruction():
    """Test semantic search with document reconstruction."""
    searcher = SearchWithReconstruction()
    
    # Test with simpler queries that should return results from parquet_duckdb
    test_cases = [
        {
            'query': 'sales',
            'reconstruction_format': 'docx',
            'context_level': 'with_parent',
            'description': 'Simple sales query with parent context'
        },
        {
            'query': 'revenue',
            'reconstruction_format': 'docx',
            'context_level': 'with_siblings',
            'description': 'Revenue query with sibling context'
        },
        {
            'query': 'financial performance',
            'reconstruction_format': 'pptx',
            'context_level': 'with_parent',
            'description': 'Financial performance query as presentation format'
        },
        {
            'query': 'north america',
            'reconstruction_format': 'docx',
            'context_level': 'element_only',
            'description': 'Geographic query with element only context'
        }
    ]
    
    logger.info("\n" + "=" * 80)
    logger.info("TESTING SEARCH WITH DOCUMENT RECONSTRUCTION")
    logger.info("=" * 80)
    
    all_results = []
    
    for test_case in test_cases:
        logger.info(f"\n📋 Test: {test_case['description']}")
        logger.info("-" * 60)
        
        results = searcher.search_with_reconstruction(
            query=test_case['query'],
            reconstruction_format=test_case['reconstruction_format'],
            context_level=test_case['context_level'],
            limit=3
        )
        
        if results and results['results']:
            logger.info(f"\n✅ Found {len(results['results'])} results with reconstruction")
            
            # Analyze reconstruction quality
            for result in results['results']:
                if result['reconstructed_html']:
                    html_len = len(result['reconstructed_html'])
                    has_styling = '<style>' in result['reconstructed_html']
                    has_structure = any(tag in result['reconstructed_html'] 
                                      for tag in ['<h1>', '<h2>', '<p>', '<div class="slide"'])
                    
                    logger.info(f"\n   Result #{result['rank']} Reconstruction Analysis:")
                    logger.info(f"     • HTML length: {html_len} chars")
                    logger.info(f"     • Has styling: {'✅' if has_styling else '❌'}")
                    logger.info(f"     • Has structure: {'✅' if has_structure else '❌'}")
                    logger.info(f"     • Format: {result['reconstruction_format']}")
                    logger.info(f"     • Context: {result['context_level']}")
        else:
            logger.warning("⚠️  No results found for this query")
        
        all_results.append({
            'test_case': test_case,
            'results': results
        })
    
    # Summary
    logger.info("\n" + "=" * 80)
    logger.info("RECONSTRUCTION TEST SUMMARY")
    logger.info("=" * 80)
    
    successful_tests = sum(1 for r in all_results if r['results'] and r['results']['results'])
    total_reconstructions = sum(
        len(r['results']['results']) for r in all_results 
        if r['results'] and r['results']['results']
    )
    
    logger.info(f"\n📊 Results:")
    logger.info(f"   • Tests run: {len(test_cases)}")
    logger.info(f"   • Successful tests: {successful_tests}")
    logger.info(f"   • Total reconstructions: {total_reconstructions}")
    
    logger.info(f"\n💡 Key Benefits Demonstrated:")
    logger.info(f"   1. Search results include formatted HTML context")
    logger.info(f"   2. Multiple reconstruction formats (DOCX, PPTX)")
    logger.info(f"   3. Flexible context levels (element, parent, siblings, full)")
    logger.info(f"   4. Preserves document structure and styling")
    logger.info(f"   5. Helps users understand result location in document")
    
    return all_results


def test_reconstruction_comparison():
    """Compare different reconstruction formats and context levels."""
    searcher = SearchWithReconstruction()
    
    logger.info("\n" + "=" * 80)
    logger.info("RECONSTRUCTION FORMAT & CONTEXT LEVEL COMPARISON")
    logger.info("=" * 80)
    
    # Use a single query for comparison
    query = "financial performance revenue growth"
    
    # Test all combinations
    formats = ['docx', 'pptx']
    context_levels = ['element_only', 'with_parent', 'with_siblings']
    
    comparison_results = {}
    
    for format_type in formats:
        for context_level in context_levels:
            logger.info(f"\n🔍 Testing: {format_type.upper()} format with {context_level}")
            logger.info("-" * 40)
            
            results = searcher.search_with_reconstruction(
                query=query,
                reconstruction_format=format_type,
                context_level=context_level,
                limit=1  # Just get top result for comparison
            )
            
            key = f"{format_type}_{context_level}"
            comparison_results[key] = results
            
            if results and results['results'] and results['results'][0]['reconstructed_html']:
                html = results['results'][0]['reconstructed_html']
                
                # Analyze characteristics
                logger.info(f"   📏 HTML size: {len(html)} chars")
                
                # Check for format-specific elements
                if format_type == 'docx':
                    has_times_new_roman = "'Times New Roman'" in html
                    has_page_styling = "page-header" in html or "margin: 1in" in html
                    logger.info(f"   📝 DOCX styling: {has_times_new_roman and has_page_styling}")
                else:  # pptx
                    has_slides = "class=\"slide\"" in html
                    has_calibri = "'Calibri'" in html
                    logger.info(f"   🎯 PPTX styling: {has_slides and has_calibri}")
                
                # Count structural elements
                h_tags = html.count('<h') 
                p_tags = html.count('<p>')
                div_tags = html.count('<div')
                
                logger.info(f"   📊 Structure: {h_tags} headers, {p_tags} paragraphs, {div_tags} divs")
    
    # Display comparison matrix
    logger.info("\n" + "=" * 80)
    logger.info("COMPARISON MATRIX - HTML SIZE (chars)")
    logger.info("-" * 80)
    
    logger.info(f"{'Context Level':<20} | {'DOCX':<15} | {'PPTX':<15}")
    logger.info("-" * 52)
    
    for context_level in context_levels:
        docx_key = f"docx_{context_level}"
        pptx_key = f"pptx_{context_level}"
        
        docx_size = 0
        pptx_size = 0
        
        if docx_key in comparison_results and comparison_results[docx_key]:
            if comparison_results[docx_key]['results']:
                if comparison_results[docx_key]['results'][0]['reconstructed_html']:
                    docx_size = len(comparison_results[docx_key]['results'][0]['reconstructed_html'])
        
        if pptx_key in comparison_results and comparison_results[pptx_key]:
            if comparison_results[pptx_key]['results']:
                if comparison_results[pptx_key]['results'][0]['reconstructed_html']:
                    pptx_size = len(comparison_results[pptx_key]['results'][0]['reconstructed_html'])
        
        logger.info(f"{context_level:<20} | {docx_size:<15} | {pptx_size:<15}")
    
    logger.info("\n✅ Comparison complete - shows how format and context affect output")
    
    return comparison_results


def demonstrate_html_output():
    """Save sample reconstructed HTML files for visual inspection."""
    searcher = SearchWithReconstruction()
    
    logger.info("\n" + "=" * 80)
    logger.info("GENERATING SAMPLE HTML FILES")
    logger.info("=" * 80)
    
    # Perform a search
    results = searcher.search_with_reconstruction(
        query="financial performance metrics",
        reconstruction_format='docx',
        context_level='with_siblings',
        limit=1
    )
    
    if results and results['results'] and results['results'][0]['reconstructed_html']:
        # Save DOCX format
        docx_html = results['results'][0]['reconstructed_html']
        docx_file = Path("sample_reconstruction_docx.html")
        docx_file.write_text(docx_html)
        logger.info(f"✅ Saved DOCX format to: {docx_file}")
        
        # Get PPTX format for same result
        results_pptx = searcher.search_with_reconstruction(
            query="financial performance metrics",
            reconstruction_format='pptx',
            context_level='with_siblings',
            limit=1
        )
        
        if results_pptx and results_pptx['results'] and results_pptx['results'][0]['reconstructed_html']:
            pptx_html = results_pptx['results'][0]['reconstructed_html']
            pptx_file = Path("sample_reconstruction_pptx.html")
            pptx_file.write_text(pptx_html)
            logger.info(f"✅ Saved PPTX format to: {pptx_file}")
            
            logger.info("\n📌 Open these files in a browser to see the visual formatting:")
            logger.info(f"   • {docx_file.absolute()}")
            logger.info(f"   • {pptx_file.absolute()}")
    else:
        logger.warning("⚠️  No results to save - ensure documents are indexed")


if __name__ == "__main__":
    # Run all tests
    logger.info("\n" + "🚀 STARTING ENHANCED SEARCH TESTS WITH RECONSTRUCTION")
    
    # Test 1: Basic search with reconstruction
    test_search_with_reconstruction()
    
    # Test 2: Compare formats and context levels
    print("\n" + "=" * 80)
    test_reconstruction_comparison()
    
    # Test 3: Generate sample HTML files
    print("\n" + "=" * 80)
    demonstrate_html_output()
    
    logger.info("\n" + "=" * 80)
    logger.info("✅ ALL TESTS COMPLETED")
    logger.info("=" * 80)