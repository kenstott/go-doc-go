"""
Integration tests for Web content source and adapter.
"""

import pytest
import json
import os
import time
from typing import Dict, Any, List
from unittest.mock import patch, MagicMock

# Import the content source and adapter
from go_doc_go.content_source.web import WebContentSource
from go_doc_go.adapter.web import WebAdapter
from go_doc_go.document_parser.factory import create_parser


class TestWebIntegration:
    """Integration tests combining web content source, adapter, and parsers."""
    
    @pytest.fixture
    def web_source(self, web_server_config):
        """Create a WebContentSource configured for the test server."""
        config = {
            "name": "test-web-source",
            "base_url": web_server_config["base_url"],
            "url_list": [
                "index.html",
                "page1.html",
                "page2.html",
                "data.json",
                "api/data.csv"
            ],
            "max_link_depth": 2,
            "include_patterns": [r".*\.(html|json|csv)$"],
            "exclude_patterns": [r".*error.*"]
        }
        return WebContentSource(config)
    
    @pytest.fixture
    def web_adapter(self):
        """Create a WebAdapter for testing."""
        config = {
            "timeout": 30,
            "follow_links": True,
            "max_link_depth": 2
        }
        return WebAdapter(config)
    
    def test_end_to_end_html_processing(self, web_source, web_adapter, web_server_config):
        """Test complete workflow: fetch, adapt, and parse HTML document."""
        # Fetch document using content source
        doc = web_source.fetch_document(f"{web_server_config['base_url']}/index.html")
        
        assert doc["id"] == f"{web_server_config['base_url']}/index.html"
        assert "Welcome to Test Site" in doc["content"]
        
        # Get content using adapter
        location_data = {"source": doc["id"]}
        adapted_content = web_adapter.get_content(location_data)
        
        assert adapted_content["content_type"] == "html"
        assert "Welcome to Test Site" in adapted_content["content"]
        
        # Parse the document
        parser = create_parser("html")
        parsed = parser.parse({
            "id": doc["id"],
            "content": adapted_content["content"],
            "metadata": adapted_content["metadata"]
        })
        
        # Verify parsing results
        assert parsed["document"]["doc_id"]  # Parser generates its own ID
        assert parsed["document"]["doc_type"] == "html"
        assert len(parsed["elements"]) > 0
        
        # Check for expected elements
        element_types = [e["element_type"] for e in parsed["elements"]]
        assert "header" in element_types or "heading" in element_types
        assert "paragraph" in element_types or "text" in element_types
        assert len(element_types) > 5  # Should have multiple elements
    
    def test_end_to_end_json_processing(self, web_source, web_adapter, web_server_config):
        """Test complete workflow for JSON document."""
        # Fetch JSON document
        doc = web_source.fetch_document(f"{web_server_config['base_url']}/data.json")
        
        assert "Test JSON Data" in doc["content"]
        
        # Adapt content
        location_data = {"source": doc["id"]}
        adapted_content = web_adapter.get_content(location_data)
        
        assert adapted_content["content_type"] == "json"
        
        # Parse JSON
        parser = create_parser("json")
        parsed = parser.parse({
            "id": doc["id"],
            "content": adapted_content["content"],
            "metadata": adapted_content["metadata"]
        })
        
        # Verify JSON structure was parsed
        assert parsed["document"]["doc_id"]  # Parser generates its own ID  
        assert parsed["document"]["doc_type"] == "json"
        assert len(parsed["elements"]) > 0
        
        # Check for expected JSON structure elements
        element_contents = [e["content_preview"] for e in parsed["elements"]]
        assert any("Test JSON Data" in c for c in element_contents)
    
    def test_end_to_end_csv_processing(self, web_source, web_adapter, web_server_config):
        """Test complete workflow for CSV document."""
        # Fetch CSV document
        doc = web_source.fetch_document(f"{web_server_config['base_url']}/api/data.csv")
        
        assert "Alice Johnson" in doc["content"]
        
        # Adapt content
        location_data = {"source": doc["id"]}
        adapted_content = web_adapter.get_content(location_data)
        
        assert adapted_content["content_type"] == "csv"
        
        # Parse CSV
        parser = create_parser("csv")
        parsed = parser.parse({
            "id": doc["id"],
            "content": adapted_content["content"],
            "metadata": adapted_content["metadata"]
        })
        
        # Verify CSV was parsed
        assert parsed["document"]["doc_id"]  # Parser generates its own ID
        assert parsed["document"]["doc_type"] == "csv"
        assert len(parsed["elements"]) > 0
        
        # Check for table structure
        element_types = [e["element_type"] for e in parsed["elements"]]
        assert "table" in element_types
    
    def test_link_following_integration(self, web_source, web_server_config):
        """Test link following across multiple documents."""
        # Start from index page
        index_doc = web_source.fetch_document(f"{web_server_config['base_url']}/index.html")
        
        # Follow links from index
        linked_docs = web_source.follow_links(
            index_doc["content"],
            index_doc["id"],
            current_depth=0
        )
        
        # Should have found multiple linked documents
        assert len(linked_docs) > 0
        
        # Verify we got the expected pages
        linked_ids = [doc["id"] for doc in linked_docs]
        assert any("page1.html" in id for id in linked_ids)
        assert any("page2.html" in id for id in linked_ids)
        
        # Verify content was fetched
        for doc in linked_docs:
            assert doc["content"]
            assert doc["metadata"]
    
    def test_recursive_link_following(self, web_source, web_server_config):
        """Test recursive link following with depth control."""
        web_source.max_link_depth = 2
        
        # Start from page1
        page1_doc = web_source.fetch_document(f"{web_server_config['base_url']}/page1.html")
        
        # Follow links recursively
        all_linked = web_source.follow_links(
            page1_doc["content"],
            page1_doc["id"],
            current_depth=0
        )
        
        # Should have followed links to depth 2
        linked_ids = [doc["id"] for doc in all_linked]
        
        # Should include pages linked from page1 and their children
        assert len(linked_ids) > 0
        
        # Verify no duplicates (global visited tracking)
        assert len(linked_ids) == len(set(linked_ids))
    
    def test_authentication_integration(self, web_server_config):
        """Test authenticated access through content source and adapter."""
        # Create protected content
        protected_dir = os.path.join(
            os.path.dirname(__file__), '..', 'assets', 'web', 'protected'
        )
        os.makedirs(protected_dir, exist_ok=True)
        
        protected_file = os.path.join(protected_dir, 'secure.json')
        with open(protected_file, 'w') as f:
            json.dump({"secure": "content", "value": 42}, f)
        
        try:
            # Configure source with authentication
            source_config = {
                "name": "secure-source",
                "base_url": web_server_config["base_url"],
                "authentication": {
                    "type": "basic",
                    "username": web_server_config["auth"]["username"],
                    "password": web_server_config["auth"]["password"]
                }
            }
            source = WebContentSource(source_config)
            
            # Configure adapter with authentication
            adapter_config = {
                "authentication": {
                    "type": "basic",
                    "username": web_server_config["auth"]["username"],
                    "password": web_server_config["auth"]["password"]
                }
            }
            adapter = WebAdapter(adapter_config)
            
            # Fetch protected content
            doc = source.fetch_document(f"{web_server_config['base_url']}/protected/secure.json")
            assert "secure" in doc["content"]
            
            # Adapt protected content
            adapted = adapter.get_content({"source": doc["id"]})
            assert adapted["content_type"] == "json"
            
            # Parse the JSON
            json_data = json.loads(adapted["content"])
            assert json_data["secure"] == "content"
            assert json_data["value"] == 42
            
        finally:
            if os.path.exists(protected_file):
                os.unlink(protected_file)
            if os.path.exists(protected_dir):
                os.rmdir(protected_dir)
    
    def test_concurrent_document_fetching(self, web_source, web_server_config):
        """Test fetching multiple documents concurrently."""
        import concurrent.futures
        
        urls = [
            f"{web_server_config['base_url']}/index.html",
            f"{web_server_config['base_url']}/page1.html",
            f"{web_server_config['base_url']}/page2.html",
            f"{web_server_config['base_url']}/data.json",
            f"{web_server_config['base_url']}/api/data.csv"
        ]
        
        # Fetch documents concurrently
        with concurrent.futures.ThreadPoolExecutor(max_workers=3) as executor:
            futures = [executor.submit(web_source.fetch_document, url) for url in urls]
            results = [f.result() for f in concurrent.futures.as_completed(futures)]
        
        # Verify all documents were fetched
        assert len(results) == len(urls)
        
        for doc in results:
            assert doc["id"] in urls
            assert doc["content"]
            assert doc["metadata"]
    
    def test_caching_across_components(self, web_source, web_adapter, web_server_config):
        """Test that caching works across content source and adapter."""
        url = f"{web_server_config['base_url']}/index.html"
        
        # Clear caches
        web_source.content_cache.clear()
        web_adapter.content_cache.clear()
        
        # First fetch through content source
        doc1 = web_source.fetch_document(url)
        
        # Should be in source cache
        assert url in web_source.content_cache
        
        # Fetch through adapter
        adapted1 = web_adapter.get_content({"source": url})
        
        # Should be in adapter cache
        assert url in web_adapter.content_cache
        
        # Mock the sessions to verify cache usage
        original_source_session = web_source.session
        original_adapter_session = web_adapter.session
        
        mock_source_session = MagicMock()
        mock_adapter_session = MagicMock()
        
        web_source.session = mock_source_session
        web_adapter.session = mock_adapter_session
        
        # Fetch again - should use cache
        doc2 = web_source.fetch_document(url)
        adapted2 = web_adapter.get_content({"source": url})
        
        # Sessions should not have been called
        mock_source_session.get.assert_not_called()
        mock_adapter_session.get.assert_not_called()
        
        # Content should be the same
        assert doc2["content"] == doc1["content"]
        assert adapted2["content"] == adapted1["content"]
        
        # Restore sessions
        web_source.session = original_source_session
        web_adapter.session = original_adapter_session
    
    def test_error_propagation(self, web_source, web_adapter, web_server_config):
        """Test that errors are properly propagated through the pipeline."""
        # Try to fetch non-existent page
        with pytest.raises(Exception) as exc_info:
            web_source.fetch_document(f"{web_server_config['base_url']}/nonexistent.html")
        
        assert "404" in str(exc_info.value) or "Not Found" in str(exc_info.value)
        
        # Try through adapter
        with pytest.raises(ValueError) as exc_info:
            web_adapter.get_content({"source": f"{web_server_config['base_url']}/missing.html"})
        
        assert "Error fetching URL" in str(exc_info.value)
    
    def test_metadata_preservation(self, web_source, web_adapter, web_server_config):
        """Test that metadata is preserved through the pipeline."""
        url = f"{web_server_config['base_url']}/page2.html"
        
        # Fetch through content source
        doc = web_source.fetch_document(url)
        source_metadata = doc["metadata"]
        
        # Adapt content
        adapted = web_adapter.get_content({"source": url})
        adapter_metadata = adapted["metadata"]
        
        # Common metadata should be preserved
        assert source_metadata["url"] == adapter_metadata["url"]
        assert source_metadata["status_code"] == adapter_metadata["status_code"]
        
        # Adapter should add additional metadata
        assert "title" in adapter_metadata  # HTML-specific metadata
        assert "element_counts" in adapter_metadata
    
    def test_performance_multiple_documents(self, web_source, web_server_config):
        """Test performance when fetching multiple documents."""
        start_time = time.time()
        
        # List all documents
        documents = web_source.list_documents()
        
        # Fetch each document
        for doc_info in documents[:5]:  # Limit to 5 for test speed
            doc = web_source.fetch_document(doc_info["id"])
            assert doc["content"]
        
        elapsed = time.time() - start_time
        
        # Should complete reasonably quickly (adjust threshold as needed)
        assert elapsed < 10.0, f"Fetching 5 documents took {elapsed}s"
    
    def test_pattern_filtering_integration(self, web_server_config):
        """Test that include/exclude patterns work correctly."""
        config = {
            "name": "filtered-source",
            "base_url": web_server_config["base_url"],
            "include_patterns": [r".*\.json$", r".*\.csv$"],  # Only JSON and CSV
            "exclude_patterns": [r".*api.*"],  # Exclude api directory
            "max_link_depth": 1
        }
        source = WebContentSource(config)
        
        # Fetch index page
        index_doc = source.fetch_document(f"{web_server_config['base_url']}/index.html")
        
        # Follow links with filtering
        linked_docs = source.follow_links(index_doc["content"], index_doc["id"], 0)
        
        linked_ids = [doc["id"] for doc in linked_docs]
        
        # Should include JSON files
        assert any("data.json" in id for id in linked_ids)
        
        # Should NOT include CSV in api directory (excluded by pattern)
        assert not any("api/data.csv" in id for id in linked_ids)
        
        # Should NOT include HTML files (not in include patterns)
        assert not any("page1.html" in id for id in linked_ids)
        assert not any("page2.html" in id for id in linked_ids)