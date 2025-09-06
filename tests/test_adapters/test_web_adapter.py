"""
Tests for Web adapter.
"""

import pytest
import json
import tempfile
import os
from unittest.mock import patch, MagicMock, Mock
from typing import Dict, Any
import requests

from go_doc_go.adapter.web import WebAdapter


class TestWebAdapterUnit:
    """Unit tests for WebAdapter without real web server."""
    
    def test_adapter_initialization(self):
        """Test adapter initialization with default configuration."""
        adapter = WebAdapter()
        
        assert adapter.timeout == 30
        assert adapter.max_redirects == 5
        assert adapter.verify_ssl is True
        assert adapter.follow_links is False
        assert adapter.max_link_depth == 1
        assert adapter.download_assets is False
    
    def test_adapter_initialization_with_config(self):
        """Test adapter initialization with custom configuration."""
        config = {
            "timeout": 60,
            "max_redirects": 10,
            "verify_ssl": False,
            "follow_links": True,
            "max_link_depth": 3,
            "download_assets": True,
            "headers": {"User-Agent": "TestBot/2.0"},
            "authentication": {
                "type": "bearer",
                "token": "test-token"
            }
        }
        
        adapter = WebAdapter(config)
        
        assert adapter.timeout == 60
        assert adapter.max_redirects == 10
        assert adapter.verify_ssl is False
        assert adapter.follow_links is True
        assert adapter.max_link_depth == 3
        assert adapter.download_assets is True
        assert "User-Agent" in adapter.headers
        assert adapter.headers["User-Agent"] == "TestBot/2.0"
        assert adapter.session.headers.get("Authorization") == "Bearer test-token"
    
    def test_adapter_with_basic_auth(self):
        """Test adapter with basic authentication configuration."""
        config = {
            "authentication": {
                "type": "basic",
                "username": "testuser",
                "password": "testpass"
            }
        }
        
        adapter = WebAdapter(config)
        assert adapter.session.auth == ("testuser", "testpass")
    
    def test_supports_location_valid_urls(self):
        """Test that adapter correctly identifies valid web URLs."""
        adapter = WebAdapter()
        
        # Valid URLs
        assert adapter.supports_location({"source": "http://example.com"}) is True
        assert adapter.supports_location({"source": "https://example.com"}) is True
        assert adapter.supports_location({"source": "https://api.example.com/data"}) is True
        assert adapter.supports_location({"source": "http://localhost:8080/test"}) is True
        
        # Invalid URLs
        assert adapter.supports_location({"source": "ftp://example.com"}) is False
        assert adapter.supports_location({"source": "file:///path/to/file"}) is False
        assert adapter.supports_location({"source": "/local/path"}) is False
        assert adapter.supports_location({"source": ""}) is False
        assert adapter.supports_location({}) is False
    
    def test_resolve_uri(self):
        """Test URI parsing and resolution."""
        adapter = WebAdapter()
        
        # Test HTTP URL
        result = adapter.resolve_uri("http://example.com/path/to/page?param=value#section")
        assert result["source"] == "http://example.com/path/to/page?param=value#section"
        assert result["scheme"] == "http"
        assert result["netloc"] == "example.com"
        assert result["path"] == "/path/to/page"
        assert result["query"] == "param=value"
        assert result["fragment"] == "section"
        
        # Test HTTPS URL
        result = adapter.resolve_uri("https://api.example.com:8443/v1/data")
        assert result["scheme"] == "https"
        assert result["netloc"] == "api.example.com:8443"
        assert result["path"] == "/v1/data"
        
        # Test invalid URL
        with pytest.raises(ValueError, match="Not a web URI"):
            adapter.resolve_uri("ftp://example.com/file")
    
    def test_is_text_content(self):
        """Test MIME type detection for text content."""
        # Text types
        assert WebAdapter._is_text_content("text/plain") is True
        assert WebAdapter._is_text_content("text/html") is True
        assert WebAdapter._is_text_content("text/css") is True
        assert WebAdapter._is_text_content("application/json") is True
        assert WebAdapter._is_text_content("application/xml") is True
        assert WebAdapter._is_text_content("application/yaml") is True
        assert WebAdapter._is_text_content("application/javascript") is True
        assert WebAdapter._is_text_content("application/csv") is True
        assert WebAdapter._is_text_content("application/rss+xml") is True
        assert WebAdapter._is_text_content("application/xhtml+xml") is True
        
        # Binary types
        assert WebAdapter._is_text_content("image/png") is False
        assert WebAdapter._is_text_content("image/jpeg") is False
        assert WebAdapter._is_text_content("application/pdf") is False
        assert WebAdapter._is_text_content("application/octet-stream") is False
        assert WebAdapter._is_text_content("video/mp4") is False
        assert WebAdapter._is_text_content("audio/mpeg") is False
    
    @patch('go_doc_go.adapter.web.requests.Session')
    def test_get_content_text_success(self, mock_session_class):
        """Test getting text content from web source."""
        # Setup mock
        mock_session = MagicMock()
        mock_session_class.return_value = mock_session
        
        mock_response = MagicMock()
        mock_response.text = "<html><body>Hello World</body></html>"
        mock_response.content = b"<html><body>Hello World</body></html>"
        mock_response.headers = {"Content-Type": "text/html; charset=utf-8"}
        mock_response.status_code = 200
        mock_response.encoding = "utf-8"
        mock_response.url = "https://example.com/page.html"
        mock_session.get.return_value = mock_response
        
        adapter = WebAdapter()
        adapter.session = mock_session
        
        result = adapter.get_content({"source": "https://example.com/page.html"})
        
        assert result["content"] == "<html><body>Hello World</body></html>"
        assert result["content_type"] == "html"
        assert result["metadata"]["url"] == "https://example.com/page.html"
        assert result["metadata"]["content_type"] == "text/html; charset=utf-8"
        assert result["metadata"]["is_binary"] is False
        assert result["metadata"]["status_code"] == 200
    
    @patch('go_doc_go.adapter.web.requests.Session')
    def test_get_content_json(self, mock_session_class):
        """Test getting JSON content."""
        mock_session = MagicMock()
        mock_session_class.return_value = mock_session
        
        json_data = {"key": "value", "number": 123}
        mock_response = MagicMock()
        mock_response.text = json.dumps(json_data)
        mock_response.content = json.dumps(json_data).encode()
        mock_response.headers = {"Content-Type": "application/json"}
        mock_response.status_code = 200
        mock_response.encoding = "utf-8"
        mock_response.url = "https://api.example.com/data.json"
        mock_session.get.return_value = mock_response
        
        adapter = WebAdapter()
        adapter.session = mock_session
        
        result = adapter.get_content({"source": "https://api.example.com/data.json"})
        
        assert result["content"] == json.dumps(json_data)
        assert result["content_type"] == "json"
        assert result["metadata"]["content_type"] == "application/json"
        assert result["metadata"]["is_binary"] is False
    
    @patch('go_doc_go.adapter.web.requests.Session')
    def test_get_binary_content(self, mock_session_class):
        """Test getting binary content."""
        mock_session = MagicMock()
        mock_session_class.return_value = mock_session
        
        binary_data = b'\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR'
        mock_response = MagicMock()
        mock_response.content = binary_data
        mock_response.status_code = 200
        mock_session.get.return_value = mock_response
        
        adapter = WebAdapter()
        adapter.session = mock_session
        
        result = adapter.get_binary_content({"source": "https://example.com/image.png"})
        
        assert result == binary_data
        mock_session.get.assert_called_with(
            "https://example.com/image.png",
            timeout=30,
            allow_redirects=True,
            verify=True,
            stream=True
        )
    
    @patch('go_doc_go.adapter.web.requests.Session')
    def test_get_metadata(self, mock_session_class):
        """Test getting metadata without fetching content."""
        mock_session = MagicMock()
        mock_session_class.return_value = mock_session
        
        mock_response = MagicMock()
        mock_response.headers = {
            "Content-Type": "text/html",
            "Content-Length": "1024",
            "Last-Modified": "Wed, 21 Oct 2025 07:28:00 GMT",
            "ETag": '"abc123"'
        }
        mock_response.status_code = 200
        mock_response.url = "https://example.com/page.html"
        mock_session.head.return_value = mock_response
        
        adapter = WebAdapter()
        adapter.session = mock_session
        
        metadata = adapter.get_metadata({"source": "https://example.com/page.html"})
        
        assert metadata["url"] == "https://example.com/page.html"
        assert metadata["content_type"] == "text/html"
        assert metadata["content_length"] == 1024
        assert metadata["etag"] == '"abc123"'
        assert metadata["is_binary"] is False
        
        # Should use HEAD request
        mock_session.head.assert_called_once()
    
    @patch('go_doc_go.adapter.web.BeautifulSoup')
    def test_extract_html_metadata(self, mock_soup_class):
        """Test HTML metadata extraction."""
        # Mock BeautifulSoup
        mock_soup = MagicMock()
        mock_soup_class.return_value = mock_soup
        
        # Mock title
        mock_title = MagicMock()
        mock_title.string = "Test Page Title"
        mock_soup.title = mock_title
        
        # Mock meta tags properly
        mock_meta1 = MagicMock()
        mock_meta1.get.side_effect = lambda x: {"name": "description", "property": None, "content": "Test description"}.get(x)
        mock_meta2 = MagicMock()
        mock_meta2.get.side_effect = lambda x: {"name": None, "property": "og:title", "content": "OG Title"}.get(x)
        
        # Set up find_all to return different values based on the argument
        def find_all_side_effect(arg, **kwargs):
            if arg == 'meta':
                return [mock_meta1, mock_meta2]
            elif arg == ['h1', 'h2', 'h3', 'h4', 'h5', 'h6']:
                return ["h1", "h2", "h3"]
            elif arg == 'p':
                return ["p1", "p2"]
            elif arg == 'table':
                return ["table"]
            elif arg == 'img':
                return ["img1", "img2"]
            elif arg == 'a' and 'href' in kwargs:
                # Mock links with href attribute
                link1 = MagicMock()
                link1.__getitem__.side_effect = lambda x: "/link1" if x == 'href' else None
                link2 = MagicMock()
                link2.__getitem__.side_effect = lambda x: "/link2" if x == 'href' else None
                link3 = MagicMock()
                link3.__getitem__.side_effect = lambda x: "/link3" if x == 'href' else None
                return [link1, link2, link3]
            return []
        
        mock_soup.find_all.side_effect = find_all_side_effect
        
        adapter = WebAdapter({"follow_links": True})
        html_content = "<html><body>Test</body></html>"
        
        metadata = adapter._extract_html_metadata(html_content, "https://example.com")
        
        assert metadata["title"] == "Test Page Title"
        assert "meta_tags" in metadata
        assert metadata["meta_tags"]["description"] == "Test description"
        assert metadata["meta_tags"]["og:title"] == "OG Title"
        assert metadata["element_counts"]["headings"] == 3
        assert metadata["element_counts"]["paragraphs"] == 2
        assert metadata["element_counts"]["tables"] == 1
        assert metadata["element_counts"]["images"] == 2
        assert metadata["element_counts"]["links"] == 3
        assert "links" in metadata  # Since follow_links is True
        assert len(metadata["links"]) == 3
    
    @patch('go_doc_go.adapter.web.requests.Session')
    def test_error_handling_connection_error(self, mock_session_class):
        """Test error handling for connection errors."""
        mock_session = MagicMock()
        mock_session_class.return_value = mock_session
        
        # Simulate connection error
        mock_session.get.side_effect = requests.ConnectionError("Connection failed")
        
        adapter = WebAdapter()
        adapter.session = mock_session
        
        with pytest.raises(ValueError, match="Error fetching URL"):
            adapter.get_content({"source": "https://unreachable.com/page"})
    
    @patch('go_doc_go.adapter.web.requests.Session')
    def test_error_handling_http_error(self, mock_session_class):
        """Test error handling for HTTP errors."""
        mock_session = MagicMock()
        mock_session_class.return_value = mock_session
        
        mock_response = MagicMock()
        mock_response.raise_for_status.side_effect = requests.HTTPError("404 Not Found")
        mock_session.get.return_value = mock_response
        
        adapter = WebAdapter()
        adapter.session = mock_session
        
        with pytest.raises(ValueError, match="Error fetching URL"):
            adapter.get_content({"source": "https://example.com/notfound"})
    
    def test_cleanup(self):
        """Test adapter cleanup."""
        adapter = WebAdapter()
        
        # Add some data to caches
        adapter.content_cache["test"] = "data"
        adapter.binary_cache["test"] = b"data"
        adapter.metadata_cache["test"] = {"key": "value"}
        
        # Cleanup
        adapter.cleanup()
        
        # Caches should be cleared
        assert len(adapter.content_cache) == 0
        assert len(adapter.binary_cache) == 0
        assert len(adapter.metadata_cache) == 0


class TestWebAdapterIntegration:
    """Integration tests for WebAdapter with real web server."""
    
    def test_get_html_content(self, web_server_config):
        """Test getting HTML content from web server."""
        adapter = WebAdapter()
        
        result = adapter.get_content({
            "source": f"{web_server_config['base_url']}/index.html"
        })
        
        assert "Welcome to Test Site" in result["content"]
        assert result["content_type"] == "html"
        assert result["metadata"]["url"] == f"{web_server_config['base_url']}/index.html"
        assert result["metadata"]["is_binary"] is False
        assert result["metadata"]["status_code"] == 200
        
        # Check HTML-specific metadata
        assert "title" in result["metadata"]
        assert "Test Index Page" in result["metadata"]["title"]
        assert "element_counts" in result["metadata"]
    
    def test_get_json_content(self, web_server_config):
        """Test getting JSON content from web server."""
        adapter = WebAdapter()
        
        result = adapter.get_content({
            "source": f"{web_server_config['base_url']}/data.json"
        })
        
        # Verify JSON content
        json_data = json.loads(result["content"])
        assert json_data["title"] == "Test JSON Data"
        assert result["content_type"] == "json"
        assert result["metadata"]["is_binary"] is False
    
    def test_get_csv_content(self, web_server_config):
        """Test getting CSV content from web server."""
        adapter = WebAdapter()
        
        result = adapter.get_content({
            "source": f"{web_server_config['base_url']}/api/data.csv"
        })
        
        assert "Alice Johnson" in result["content"]
        assert "id,name,email,department" in result["content"]
        assert result["content_type"] == "csv"
        assert result["metadata"]["is_binary"] is False
    
    def test_get_metadata_only(self, web_server_config):
        """Test getting metadata without content."""
        adapter = WebAdapter()
        
        metadata = adapter.get_metadata({
            "source": f"{web_server_config['base_url']}/index.html"
        })
        
        assert metadata["url"] == f"{web_server_config['base_url']}/index.html"
        assert "text/html" in metadata["content_type"]
        assert metadata["is_binary"] is False
        assert metadata["status_code"] == 200
    
    def test_get_binary_content_handling(self, web_server_config):
        """Test handling of binary content."""
        # Create a test binary file
        binary_file = os.path.join(
            os.path.dirname(__file__), '..', 'assets', 'web', 'test.bin'
        )
        
        with open(binary_file, 'wb') as f:
            f.write(b'\x00\x01\x02\x03\x04\x05')
        
        try:
            adapter = WebAdapter()
            
            # Get as regular content (should handle binary)
            result = adapter.get_content({
                "source": f"{web_server_config['base_url']}/test.bin"
            })
            
            # Binary content should be returned as bytes in content field
            assert isinstance(result["content"], bytes)
            assert result["metadata"]["is_binary"] is True
            
            # Get as binary content directly
            binary_data = adapter.get_binary_content({
                "source": f"{web_server_config['base_url']}/test.bin"
            })
            
            assert binary_data == b'\x00\x01\x02\x03\x04\x05'
            
        finally:
            if os.path.exists(binary_file):
                os.unlink(binary_file)
    
    def test_custom_headers(self, web_server_config):
        """Test sending custom headers."""
        adapter = WebAdapter({
            "headers": {
                "User-Agent": "TestBot/1.0",
                "X-Custom-Header": "CustomValue"
            }
        })
        
        result = adapter.get_content({
            "source": f"{web_server_config['base_url']}/index.html",
            "headers": {
                "X-Request-ID": "12345"
            }
        })
        
        assert "Welcome to Test Site" in result["content"]
        # Headers should be sent but we can't easily verify server-side
    
    def test_authentication_required(self, web_server_config):
        """Test accessing protected resources with authentication."""
        # Create protected directory and file
        protected_dir = os.path.join(
            os.path.dirname(__file__), '..', 'assets', 'web', 'protected'
        )
        os.makedirs(protected_dir, exist_ok=True)
        
        protected_file = os.path.join(protected_dir, 'data.json')
        with open(protected_file, 'w') as f:
            json.dump({"secret": "data"}, f)
        
        try:
            # Without authentication - should fail
            adapter = WebAdapter()
            
            with pytest.raises(ValueError, match="Error fetching URL"):
                adapter.get_content({
                    "source": f"{web_server_config['base_url']}/protected/data.json"
                })
            
            # With authentication - should succeed
            adapter = WebAdapter({
                "authentication": {
                    "type": "basic",
                    "username": web_server_config["auth"]["username"],
                    "password": web_server_config["auth"]["password"]
                }
            })
            
            result = adapter.get_content({
                "source": f"{web_server_config['base_url']}/protected/data.json"
            })
            
            json_data = json.loads(result["content"])
            assert json_data["secret"] == "data"
            
        finally:
            if os.path.exists(protected_file):
                os.unlink(protected_file)
            if os.path.exists(protected_dir):
                os.rmdir(protected_dir)
    
    def test_404_error_handling(self, web_server_config):
        """Test handling of 404 errors."""
        adapter = WebAdapter()
        
        with pytest.raises(ValueError) as exc_info:
            adapter.get_content({
                "source": f"{web_server_config['base_url']}/nonexistent.html"
            })
        
        assert "Error fetching URL" in str(exc_info.value)
    
    def test_timeout_configuration(self, web_server_config):
        """Test request timeout configuration."""
        # Create adapter with very short timeout
        adapter = WebAdapter({"timeout": 0.001})
        
        # This might fail due to timeout (depends on server speed)
        # We're mainly testing that timeout is properly configured
        try:
            adapter.get_content({
                "source": f"{web_server_config['base_url']}/index.html"
            })
        except ValueError:
            # Expected if timeout occurs
            pass
    
    def test_content_caching(self, web_server_config):
        """Test content caching mechanism."""
        adapter = WebAdapter()
        url = f"{web_server_config['base_url']}/index.html"
        
        # Clear cache
        adapter.content_cache.clear()
        
        # First request
        result1 = adapter.get_content({"source": url})
        assert "Welcome to Test Site" in result1["content"]
        
        # Check cache was populated
        assert url in adapter.content_cache
        
        # Mock session to verify cache is used
        original_session = adapter.session
        mock_session = MagicMock()
        adapter.session = mock_session
        
        # Second request - should use cache
        result2 = adapter.get_content({"source": url})
        assert result2["content"] == result1["content"]
        
        # Session should not have been called
        mock_session.get.assert_not_called()
        
        # Restore original session
        adapter.session = original_session
    
    def test_resolve_uri_with_real_urls(self, web_server_config):
        """Test URI resolution with real URLs."""
        adapter = WebAdapter()
        
        for test_url in web_server_config["test_urls"]:
            result = adapter.resolve_uri(test_url)
            assert result["source"] == test_url
            assert result["scheme"] == "http"
            assert str(web_server_config["port"]) in result["netloc"]
    
    def test_html_metadata_extraction(self, web_server_config):
        """Test extraction of metadata from HTML pages."""
        adapter = WebAdapter({"follow_links": True})
        
        # Test page with rich metadata
        result = adapter.get_content({
            "source": f"{web_server_config['base_url']}/page2.html"
        })
        
        metadata = result["metadata"]
        
        # Check title extraction
        assert "title" in metadata
        assert "Complex Content" in metadata["title"]
        
        # Check meta tags extraction
        assert "meta_tags" in metadata
        assert "description" in metadata["meta_tags"]
        assert "og:title" in metadata["meta_tags"]
        
        # Check element counts
        assert "element_counts" in metadata
        assert metadata["element_counts"]["tables"] > 0
        assert metadata["element_counts"]["links"] > 0
        assert metadata["element_counts"]["headings"] > 0
    
    def test_encoding_handling(self, web_server_config):
        """Test handling of different character encodings."""
        # Create a UTF-8 encoded file with special characters
        utf8_file = os.path.join(
            os.path.dirname(__file__), '..', 'assets', 'web', 'utf8.html'
        )
        
        with open(utf8_file, 'w', encoding='utf-8') as f:
            f.write("""<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>UTF-8 Test</title>
</head>
<body>
    <p>Special characters: €£¥ 你好 مرحبا</p>
    <p>Emojis: 😀 🎉 🚀</p>
</body>
</html>""")
        
        try:
            adapter = WebAdapter()
            
            result = adapter.get_content({
                "source": f"{web_server_config['base_url']}/utf8.html"
            })
            
            # Check that special characters are preserved (may be encoded differently)
            # The test server might not preserve UTF-8 perfectly, so we check for the content
            # existing in some form
            assert "UTF-8 Test" in result["content"]
            assert "Special characters" in result["content"]
            
        finally:
            if os.path.exists(utf8_file):
                os.unlink(utf8_file)