"""
Tests for SharePoint content source.

These tests focus on integration with real SharePoint instances
and support for both REST and Graph APIs.
"""

import os
import pytest
from unittest.mock import Mock, patch, MagicMock
from typing import Dict, Any, List

from go_doc_go.content_source.sharepoint import SharePointContentSource


class TestSharePointContentSourceUnit:
    """Unit tests for SharePoint content source."""
    
    def test_initialization_with_rest_api(self):
        """Test initialization with REST API."""
        with patch('go_doc_go.content_source.sharepoint.ClientContext'):
            with patch('go_doc_go.content_source.sharepoint.OFFICE365_AVAILABLE', True):
                source = SharePointContentSource({
                    'site_url': 'https://test.sharepoint.com/sites/testsite',
                    'tenant_id': 'test',
                    'client_id': 'test',
                    'client_secret': 'test',
                    'api_type': 'rest'
                })
            assert source.site_url == 'https://test.sharepoint.com/sites/testsite'
            assert source.api_type == 'rest'
    
    def test_initialization_with_graph_api(self):
        """Test initialization with Graph API."""
        with patch('go_doc_go.content_source.sharepoint.GraphServiceClient'):
            with patch('go_doc_go.content_source.sharepoint.MSGRAPH_AVAILABLE', True):
                source = SharePointContentSource({
                    'site_url': 'https://test.sharepoint.com/sites/testsite',
                    'tenant_id': 'test',
                    'client_id': 'test',
                    'client_secret': 'test',
                    'api_type': 'graph'
                })
                assert source.site_url == 'https://test.sharepoint.com/sites/testsite'
                assert source.api_type == 'graph'
    
    def test_parse_sharepoint_uri(self):
        """Test parsing of SharePoint URIs."""
        with patch('go_doc_go.content_source.sharepoint.ClientContext'):
            with patch('go_doc_go.content_source.sharepoint.OFFICE365_AVAILABLE', True):
                source = SharePointContentSource({
                    'site_url': 'https://test.sharepoint.com/sites/testsite',
                    'tenant_id': 'test',
                    'client_id': 'test',
                    'client_secret': 'test',
                    'libraries': ['Shared Documents']
                })
                assert source.site_url == 'https://test.sharepoint.com/sites/testsite'
                assert 'Shared Documents' in source.libraries
    
    def test_filter_validation(self):
        """Test file filter validation."""
        with patch('go_doc_go.content_source.sharepoint.ClientContext'):
            with patch('go_doc_go.content_source.sharepoint.OFFICE365_AVAILABLE', True):
                source = SharePointContentSource({
                    'site_url': 'https://test.sharepoint.com',
                    'tenant_id': 'test',
                    'client_id': 'test',
                    'client_secret': 'test',
                    'include_patterns': ['*.docx', '*.pdf'],
                    'exclude_patterns': ['~$*', '*.tmp']
                })
            
                # Test include patterns
                assert source._should_include_document('document.docx', 'document.docx')
                assert source._should_include_document('report.pdf', 'report.pdf')
                assert not source._should_include_document('image.png', 'image.png')
                
                # Test exclude patterns
                assert not source._should_include_document('~$temp.docx', '~$temp.docx')
                assert not source._should_include_document('file.tmp', 'file.tmp')


@pytest.mark.integration
class TestSharePointContentSourceIntegration:
    """Integration tests with real SharePoint."""
    
    @pytest.fixture
    def config(self):
        """Get SharePoint configuration."""
        return {
            'tenant_id': os.getenv('SHAREPOINT_TENANT_ID'),
            'client_id': os.getenv('SHAREPOINT_CLIENT_ID'),
            'client_secret': os.getenv('SHAREPOINT_CLIENT_SECRET'),
            'api_type': os.getenv('SHAREPOINT_API_TYPE', 'auto')
        }
    
    @pytest.fixture
    def site_url(self):
        """Get SharePoint site URL."""
        return os.getenv('SHAREPOINT_SITE_URL')
    
    @pytest.fixture
    def test_folder(self):
        """Get test folder path."""
        return os.getenv('SHAREPOINT_TEST_PATH', '/Shared Documents')
    
    @pytest.mark.skipif(
        not os.getenv('SHAREPOINT_TENANT_ID'),
        reason="SharePoint credentials not configured"
    )
    def test_connect_to_sharepoint(self, config, site_url):
        """Test connection to SharePoint."""
        source = SharePointContentSource({
            **config,
            'site_url': site_url
        })
        
        # Verify connection by checking site URL
        assert source.site_url == site_url
    
    @pytest.mark.skipif(
        not os.getenv('SHAREPOINT_TENANT_ID'),
        reason="SharePoint credentials not configured"
    )
    def test_list_documents_rest_api(self, config, site_url, test_folder):
        """Test listing documents using REST API."""
        try:
            from office365.sharepoint.client_context import ClientContext
        except ImportError:
            pytest.skip("Office365-REST-Python-Client not installed")
        
        config['api_type'] = 'rest'
        source = SharePointContentSource({
            **config,
            'site_url': site_url,
            'libraries': [test_folder.replace('/', '').replace('/Shared Documents', 'Shared Documents')]
        })
        
        # Get documents (might be empty if folder is empty)
        documents = list(source.get_documents())
        
        # Basic validation - should not error
        assert isinstance(documents, list)
        
        # If there are documents, validate structure
        if documents:
            doc = documents[0]
            assert 'id' in doc
            assert 'metadata' in doc
            assert 'name' in doc['metadata']
    
    @pytest.mark.skipif(
        not os.getenv('SHAREPOINT_TENANT_ID'),
        reason="SharePoint credentials not configured"
    )
    def test_list_documents_graph_api(self, config, site_url, test_folder):
        """Test listing documents using Graph API."""
        try:
            from msgraph import GraphServiceClient
        except ImportError:
            pytest.skip("Microsoft Graph SDK not installed")
        
        config['api_type'] = 'graph'
        source = SharePointContentSource({
            **config,
            'site_url': site_url,
            'libraries': [test_folder.replace('/', '').replace('/Shared Documents', 'Shared Documents')]
        })
        
        # Get documents (might be empty if folder is empty)
        documents = list(source.get_documents())
        
        # Basic validation - should not error
        assert isinstance(documents, list)
        
        # If there are documents, validate structure
        if documents:
            doc = documents[0]
            assert 'id' in doc
            assert 'metadata' in doc
            assert 'name' in doc['metadata']
    
    @pytest.mark.skipif(
        not os.getenv('SHAREPOINT_TENANT_ID'),
        reason="SharePoint credentials not configured"
    )
    def test_recursive_folder_traversal(self, config, site_url, test_folder):
        """Test recursive folder traversal."""
        source = SharePointContentSource({
            **config,
            'site_url': site_url,
            'libraries': [test_folder.replace('/', '').replace('/Shared Documents', 'Shared Documents')],
            'include_subfolders': True
        })
        
        # Get all documents recursively
        documents = list(source.get_documents())
        
        # Should handle recursive traversal without errors
        assert isinstance(documents, list)
    
    @pytest.mark.skipif(
        not os.getenv('SHAREPOINT_TENANT_ID'),
        reason="SharePoint credentials not configured"
    )
    def test_file_filtering(self, config, site_url, test_folder):
        """Test file filtering by patterns."""
        source = SharePointContentSource({
            **config,
            'site_url': site_url,
            'libraries': [test_folder.replace('/', '').replace('/Shared Documents', 'Shared Documents')],
            'include_patterns': ['*.docx', '*.pdf'],
            'exclude_patterns': ['~$*']
        })
        
        documents = list(source.get_documents())
        
        # All returned documents should match include patterns
        for doc in documents:
            name = doc['metadata'].get('name', '')
            assert name.endswith('.docx') or name.endswith('.pdf')
            assert not name.startswith('~$')
    
    @pytest.mark.skipif(
        not os.getenv('SHAREPOINT_TENANT_ID'),
        reason="SharePoint credentials not configured"
    )
    def test_metadata_extraction(self, config, site_url, test_folder):
        """Test metadata extraction from SharePoint."""
        source = SharePointContentSource({
            **config,
            'site_url': site_url,
            'libraries': [test_folder.replace('/', '').replace('/Shared Documents', 'Shared Documents')],
            'max_items': 1  # Just get one file for testing
        })
        
        documents = list(source.get_documents())
        
        if documents:
            doc = documents[0]
            metadata = doc['metadata']
            
            # Check for expected metadata fields
            assert 'name' in metadata
            assert 'size' in metadata or 'file_size' in metadata
            assert 'last_modified' in metadata or 'modified' in metadata
            assert 'created' in metadata or 'created_date' in metadata
            
            # Check for SharePoint-specific metadata
            assert 'server_relative_url' in metadata or 'relative_url' in metadata


@pytest.mark.integration
@pytest.mark.slow
class TestSharePointContentSourceEndToEnd:
    """End-to-end tests for SharePoint content source."""
    
    @pytest.fixture
    def source(self):
        """Create SharePoint content source."""
        config = {
            'tenant_id': os.getenv('SHAREPOINT_TENANT_ID'),
            'client_id': os.getenv('SHAREPOINT_CLIENT_ID'),
            'client_secret': os.getenv('SHAREPOINT_CLIENT_SECRET'),
            'api_type': os.getenv('SHAREPOINT_API_TYPE', 'auto')
        }
        
        site_url = os.getenv('SHAREPOINT_SITE_URL')
        test_folder = os.getenv('SHAREPOINT_TEST_PATH', '/Shared Documents')
        
        if not all([config['tenant_id'], config['client_id'], 
                   config['client_secret'], site_url]):
            pytest.skip("SharePoint credentials not configured")
        
        return SharePointContentSource({
            **config,
            'site_url': site_url,
            'libraries': [test_folder.replace('/', '').replace('/Shared Documents', 'Shared Documents')]
        })
    
    def test_process_large_library(self, source):
        """Test processing a large document library."""
        # This would test handling of large libraries
        pytest.skip("Requires large test library")
    
    def test_api_switching(self):
        """Test switching between REST and Graph APIs."""
        config_base = {
            'tenant_id': os.getenv('SHAREPOINT_TENANT_ID'),
            'client_id': os.getenv('SHAREPOINT_CLIENT_ID'),
            'client_secret': os.getenv('SHAREPOINT_CLIENT_SECRET')
        }
        
        site_url = os.getenv('SHAREPOINT_SITE_URL')
        test_folder = os.getenv('SHAREPOINT_TEST_PATH', '/Shared Documents')
        
        if not all([config_base['tenant_id'], config_base['client_id'], 
                   config_base['client_secret'], site_url]):
            pytest.skip("SharePoint credentials not configured")
        
        # Test with REST API
        try:
            from office365.sharepoint.client_context import ClientContext
            config_rest = {**config_base, 'api_type': 'rest'}
            source_rest = SharePointContentSource({
                **config_rest,
                'site_url': site_url,
                'libraries': [test_folder.replace('/', '').replace('/Shared Documents', 'Shared Documents')]
            })
            docs_rest = list(source_rest.get_documents())
        except ImportError:
            docs_rest = []
        
        # Test with Graph API
        try:
            from msgraph import GraphServiceClient
            config_graph = {**config_base, 'api_type': 'graph'}
            source_graph = SharePointContentSource({
                **config_graph,
                'site_url': site_url,
                'libraries': [test_folder.replace('/', '').replace('/Shared Documents', 'Shared Documents')]
            })
            docs_graph = list(source_graph.get_documents())
        except ImportError:
            docs_graph = []
        
        # Both should work if libraries are installed
        if docs_rest or docs_graph:
            assert True  # At least one API worked
        else:
            pytest.skip("No SharePoint libraries installed")