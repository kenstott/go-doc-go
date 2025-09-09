"""
Tests for SharePoint adapter.

These tests focus on integration testing with real SharePoint instances.
Unit tests with mocks are minimal due to complexity of SharePoint API mocking.
"""

import os
import pytest
import tempfile
from unittest.mock import Mock, patch, MagicMock
from typing import Dict, Any

from go_doc_go.adapter.sharepoint import SharePointAdapter


class TestSharePointAdapterUnit:
    """Minimal unit tests for SharePoint adapter."""
    
    def test_adapter_initialization_without_libraries(self):
        """Test that adapter raises error when no libraries available."""
        with patch('go_doc_go.adapter.sharepoint.OFFICE365_AVAILABLE', False):
            with patch('go_doc_go.adapter.sharepoint.MSGRAPH_AVAILABLE', False):
                with pytest.raises(ImportError, match="At least one SharePoint library is required"):
                    SharePointAdapter({
                        'tenant_id': 'test',
                        'client_id': 'test',
                        'client_secret': 'test',
                        'site_url': 'https://test.sharepoint.com'
                    })
    
    def test_adapter_initialization_missing_config(self):
        """Test that adapter raises error when required config missing."""
        with patch('go_doc_go.adapter.sharepoint.OFFICE365_AVAILABLE', True):
            # Patch environment variables to ensure they don't interfere
            with patch.dict(os.environ, {}, clear=True):
                with pytest.raises(ValueError, match="SharePoint configuration incomplete"):
                    SharePointAdapter({'tenant_id': 'test'})  # Missing other required fields
    
    def test_validate_location_sharepoint_uri(self):
        """Test location validation for SharePoint URIs."""
        with patch('go_doc_go.adapter.sharepoint.OFFICE365_AVAILABLE', True):
            adapter = SharePointAdapter({
                'tenant_id': 'test',
                'client_id': 'test',
                'client_secret': 'test',
                'site_url': 'https://test.sharepoint.com'
            })
            
            # Valid SharePoint URIs
            assert adapter.validate_location({'source': 'sharepoint://test.sharepoint.com/file.docx'})
            assert adapter.validate_location({'source': '/Shared Documents/file.pdf'})
            assert adapter.validate_location({'source': 'Documents/file.xlsx'})
            
            # Invalid URIs
            assert not adapter.validate_location({'source': 'http://example.com/file.txt'})
            assert not adapter.validate_location({'source': 'https://example.com/file.txt'})
            assert not adapter.validate_location({'source': 'file:///local/file.txt'})
    
    def test_parse_sharepoint_uri(self):
        """Test parsing of SharePoint URIs."""
        with patch('go_doc_go.adapter.sharepoint.OFFICE365_AVAILABLE', True):
            with patch('go_doc_go.adapter.sharepoint.ClientContext') as mock_context:
                with patch('go_doc_go.adapter.sharepoint.ClientCredential'):
                    # Setup mock for REST API
                    mock_file = MagicMock()
                    mock_file.properties = {'Name': 'file.docx', 'Length': 1000, 'TimeLastModified': '2024-01-01'}
                    mock_file.read.return_value = MagicMock(value=b'test content')
                    
                    mock_web = MagicMock()
                    mock_web.get_file_by_server_relative_url.return_value = mock_file
                    
                    mock_client = MagicMock()
                    mock_client.web = mock_web
                    mock_client.load = MagicMock()
                    mock_client.execute_query = MagicMock()
                    
                    mock_context.return_value.with_credentials.return_value = mock_client
                    
                    adapter = SharePointAdapter({
                        'tenant_id': 'test',
                        'client_id': 'test',
                        'client_secret': 'test',
                        'site_url': 'https://test.sharepoint.com',
                        'api_type': 'rest'  # Force REST API for this test
                    })
                    
                    # Test custom SharePoint URI parsing
                    location = {'source': 'sharepoint://test.sharepoint.com/Shared Documents/file.docx'}
                    result = adapter.get_content(location)
                    
                    # Verify the file path was parsed correctly
                    mock_web.get_file_by_server_relative_url.assert_called_with('/Shared Documents/file.docx')
                    assert result['metadata']['name'] == 'file.docx'
    
    def test_api_type_selection(self):
        """Test API type selection logic."""
        # Test auto-selection with Graph available
        with patch('go_doc_go.adapter.sharepoint.OFFICE365_AVAILABLE', False):
            with patch('go_doc_go.adapter.sharepoint.MSGRAPH_AVAILABLE', True):
                adapter = SharePointAdapter({
                    'tenant_id': 'test',
                    'client_id': 'test',
                    'client_secret': 'test',
                    'site_url': 'https://test.sharepoint.com',
                    'api_type': 'auto'
                })
                assert adapter.api_type == 'graph'
        
        # Test auto-selection with REST available
        with patch('go_doc_go.adapter.sharepoint.OFFICE365_AVAILABLE', True):
            with patch('go_doc_go.adapter.sharepoint.MSGRAPH_AVAILABLE', False):
                adapter = SharePointAdapter({
                    'tenant_id': 'test',
                    'client_id': 'test',
                    'client_secret': 'test',
                    'site_url': 'https://test.sharepoint.com',
                    'api_type': 'auto'
                })
                assert adapter.api_type == 'rest'
        
        # Test explicit selection
        with patch('go_doc_go.adapter.sharepoint.OFFICE365_AVAILABLE', True):
            with patch('go_doc_go.adapter.sharepoint.MSGRAPH_AVAILABLE', True):
                adapter = SharePointAdapter({
                    'tenant_id': 'test',
                    'client_id': 'test',
                    'client_secret': 'test',
                    'site_url': 'https://test.sharepoint.com',
                    'api_type': 'rest'
                })
                assert adapter.api_type == 'rest'


@pytest.mark.integration
class TestSharePointAdapterIntegration:
    """Integration tests with real SharePoint instance."""
    
    @pytest.fixture
    def adapter_config(self):
        """Get SharePoint configuration from environment."""
        return {
            'tenant_id': os.getenv('SHAREPOINT_TENANT_ID'),
            'client_id': os.getenv('SHAREPOINT_CLIENT_ID'),
            'client_secret': os.getenv('SHAREPOINT_CLIENT_SECRET'),
            'site_url': os.getenv('SHAREPOINT_SITE_URL'),
            'api_type': os.getenv('SHAREPOINT_API_TYPE', 'auto')
        }
    
    @pytest.fixture
    def test_path(self):
        """Get test path from environment."""
        return os.getenv('SHAREPOINT_TEST_PATH', '/Shared Documents')
    
    @pytest.mark.skipif(
        not os.getenv('SHAREPOINT_TENANT_ID'),
        reason="SharePoint credentials not configured"
    )
    def test_rest_api_connection(self, adapter_config):
        """Test connection to SharePoint using REST API."""
        try:
            from office365.runtime.auth.client_credential import ClientCredential
            from office365.sharepoint.client_context import ClientContext
        except ImportError:
            pytest.skip("Office365-REST-Python-Client not installed")
        
        # Force REST API
        adapter_config['api_type'] = 'rest'
        adapter = SharePointAdapter(adapter_config)
        
        # Test that we can initialize the adapter
        assert adapter.api_type == 'rest'
        assert adapter.site_url == adapter_config['site_url']
    
    @pytest.mark.skipif(
        not os.getenv('SHAREPOINT_TENANT_ID'),
        reason="SharePoint credentials not configured"
    )
    def test_graph_api_connection(self, adapter_config):
        """Test connection to SharePoint using Graph API."""
        try:
            from msgraph import GraphServiceClient
            from azure.identity import ClientSecretCredential
        except ImportError:
            pytest.skip("Microsoft Graph SDK not installed")
        
        # Force Graph API
        adapter_config['api_type'] = 'graph'
        adapter = SharePointAdapter(adapter_config)
        
        # Test that we can initialize the adapter
        assert adapter.api_type == 'graph'
        assert adapter.site_url == adapter_config['site_url']
    
    @pytest.mark.skipif(
        not os.getenv('SHAREPOINT_TENANT_ID'),
        reason="SharePoint credentials not configured"
    )
    def test_retrieve_text_file(self, adapter_config, test_path):
        """Test retrieving a text file from SharePoint."""
        adapter = SharePointAdapter(adapter_config)
        
        # First, upload a test file
        test_content = "This is a test file for SharePoint integration testing."
        test_filename = f"test_file_{os.getpid()}.txt"
        
        # This would require uploading a file first - skipping for now
        # as it requires write permissions
        pytest.skip("Requires pre-existing test file in SharePoint")
    
    @pytest.mark.skipif(
        not os.getenv('SHAREPOINT_TENANT_ID'),
        reason="SharePoint credentials not configured"
    )
    def test_retrieve_binary_file(self, adapter_config, test_path):
        """Test retrieving a binary file from SharePoint."""
        adapter = SharePointAdapter(adapter_config)
        
        # This would require a pre-existing binary file
        pytest.skip("Requires pre-existing binary test file in SharePoint")
    
    @pytest.mark.skipif(
        not os.getenv('SHAREPOINT_TENANT_ID'),
        reason="SharePoint credentials not configured"
    )
    def test_file_not_found(self, adapter_config):
        """Test handling of non-existent file."""
        adapter = SharePointAdapter(adapter_config)
        
        # Try to get a non-existent file
        with pytest.raises(ValueError, match="Failed to retrieve SharePoint content"):
            adapter.get_content({
                'source': '/NonExistent/DoesNotExist_12345.txt'
            })
    
    @pytest.mark.skipif(
        not os.getenv('SHAREPOINT_TENANT_ID'),
        reason="SharePoint credentials not configured"
    )
    def test_get_resolver_config(self, adapter_config):
        """Test getting resolver configuration."""
        adapter = SharePointAdapter(adapter_config)
        config = adapter.get_resolver_config()
        
        assert config['adapter_type'] == 'sharepoint'
        assert config['api_type'] in ['rest', 'graph']
        assert config['site_url'] == adapter_config['site_url']
        assert 'max_file_size' in config


@pytest.mark.integration
@pytest.mark.slow
class TestSharePointAdapterEndToEnd:
    """End-to-end tests with real documents."""
    
    @pytest.fixture
    def adapter(self):
        """Create SharePoint adapter for tests."""
        config = {
            'tenant_id': os.getenv('SHAREPOINT_TENANT_ID'),
            'client_id': os.getenv('SHAREPOINT_CLIENT_ID'),
            'client_secret': os.getenv('SHAREPOINT_CLIENT_SECRET'),
            'site_url': os.getenv('SHAREPOINT_SITE_URL'),
            'api_type': os.getenv('SHAREPOINT_API_TYPE', 'auto')
        }
        
        if not all([config['tenant_id'], config['client_id'], 
                   config['client_secret'], config['site_url']]):
            pytest.skip("SharePoint credentials not configured")
        
        return SharePointAdapter(config)
    
    def test_document_types(self, adapter):
        """Test retrieving different document types."""
        # This test would iterate through various document types
        # if they exist in the test SharePoint site
        pytest.skip("Requires pre-uploaded test documents")
    
    def test_large_file_handling(self, adapter):
        """Test handling of large files."""
        # This test would retrieve a large file to test memory handling
        pytest.skip("Requires pre-uploaded large test file")
    
    def test_concurrent_retrieval(self, adapter):
        """Test concurrent file retrieval."""
        # This test would retrieve multiple files concurrently
        pytest.skip("Requires pre-uploaded test files")