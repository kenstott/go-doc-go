#!/usr/bin/env python3
"""
Simple search test for 'decreased sales' query.
Interactive exploration test using the modern SearchEngine.
"""

import logging
import sys
import json
from pathlib import Path

# Add the src directory to Python path
sys.path.insert(0, str(Path(__file__).parent.parent / "src"))

from go_doc_go import Config
from go_doc_go.search_module import SearchEngine, SearchRequest

# Set up logging for real-time output
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s',
    handlers=[logging.StreamHandler(sys.stdout)],
    force=True
)
logger = logging.getLogger(__name__)


def test_decreased_sales_search():
    """
    Simple semantic search test for 'developments that could negatively impact operating margins' query.
    Shows detailed results for interactive exploration.
    """
    logger.info("🔍 SEMANTIC SEARCH TEST: 'developments that could negatively impact operating margins'")
    logger.info("=" * 60)
    
    try:
        # Load config and check available analytics backends
        config = Config('config.yaml')
        logger.info("✅ Config loaded successfully")
        
        # List available analytics backends
        backends = config.list_analytics_backends()
        logger.info(f"📊 Available analytics backends: {list(backends.keys())}")
        
        if not backends:
            logger.error("❌ No analytics backends configured")
            return
        
        # Use the first available backend
        backend_name = list(backends.keys())[0]
        backend_config = backends[backend_name]
        logger.info(f"🎯 Using backend: {backend_name} (type: {backend_config.get('type')})")
        
        # Initialize SearchEngine
        search_engine = SearchEngine()
        logger.info("✅ SearchEngine initialized")
        
        # Create search request for "developments that could negatively impact operating margins"
        search_request = SearchRequest(
            search_service=backend_name,
            similarity_query="developments that could negatively impact operating margins",
            limit=10,
            similarity_threshold=0.7,
            include_content=False,
            include_metadata=True
        )
        
        logger.info("🔍 Executing semantic search...")
        logger.info(f"   Query: '{search_request.similarity_query}'")
        logger.info(f"   Backend: {search_request.search_service}")
        logger.info(f"   Limit: {search_request.limit}")
        logger.info(f"   Threshold: {search_request.similarity_threshold}")
        
        # Execute search
        response = search_engine.search(search_request)
        
        # Display results
        logger.info("=" * 60)
        logger.info("📋 SEARCH RESULTS")
        logger.info("=" * 60)
        logger.info(f"🔍 Query: {response.similarity_query}")
        logger.info(f"📊 Total hits: {response.total_hits}")
        logger.info(f"⏱️  Execution time: {response.took_ms}ms")
        
        if response.filters_applied:
            logger.info(f"🔧 Filters applied: {response.filters_applied}")
        
        if not response.hits:
            logger.warning("⚠️  No results found")
            logger.info("💡 Suggestions:")
            logger.info("   - Check if documents are ingested in the database")
            logger.info("   - Try a lower similarity threshold")
            logger.info("   - Verify analytics backend is properly configured")
            return
        
        logger.info(f"\n📄 DETAILED RESULTS ({len(response.hits)} hits):")
        logger.info("-" * 60)
        
        for i, hit in enumerate(response.hits, 1):
            logger.info(f"\n🔖 Result #{i}")
            logger.info(f"   📊 Score: {hit.score:.4f}")
            logger.info(f"   🆔 Element ID: {hit.element_id}")
            logger.info(f"   📄 Document ID: {hit.doc_id}")
            logger.info(f"   🏷️  Element Type: {hit.element_type}")
            logger.info(f"   📝 Preview: {hit.content_preview}")
            
            if hit.metadata:
                logger.info(f"   🔧 Metadata: {json.dumps(hit.metadata, indent=6)}")
            
            if hit.content:
                logger.info(f"   📄 Full Content: {hit.content[:200]}...")
        
        logger.info("\n" + "=" * 60)
        logger.info("✅ Search completed successfully!")
        
        return response
        
    except Exception as e:
        logger.error(f"❌ Search failed: {e}")
        import traceback
        logger.error(f"📋 Traceback:\n{traceback.format_exc()}")
        raise


def test_check_available_backends():
    """Helper function to check what analytics backends are available."""
    logger.info("🔧 CHECKING AVAILABLE ANALYTICS BACKENDS")
    logger.info("=" * 50)
    
    try:
        config = Config('config.yaml')
        backends = config.list_analytics_backends()
        
        if not backends:
            logger.warning("⚠️  No analytics backends configured")
            logger.info("💡 To enable search:")
            logger.info("   1. Configure an analytics backend in config.yaml")
            logger.info("   2. Common backends: parquet, elasticsearch, mongodb")
            logger.info("   3. Ensure documents are ingested")
            return
        
        logger.info(f"📊 Found {len(backends)} configured backends:")
        for name, config in backends.items():
            status = "✅ enabled" if config.get('enabled', True) else "❌ disabled"
            backend_type = config.get('type', 'unknown')
            logger.info(f"   • {name} ({backend_type}) - {status}")
            
            # Show key config details
            if 'base_path' in config:
                logger.info(f"     📁 Path: {config['base_path']}")
            if 'host' in config:
                logger.info(f"     🌐 Host: {config['host']}")
        
        return backends
        
    except Exception as e:
        logger.error(f"❌ Failed to check backends: {e}")
        return {}


if __name__ == "__main__":
    # Check backends first
    backends = test_check_available_backends()
    
    if backends:
        print("\n")
        # Run the search test
        test_decreased_sales_search()
    else:
        logger.error("🚫 Cannot run search test - no analytics backends available")