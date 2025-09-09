"""
Integration tests for SharePoint adapter and content source.

These tests verify the full pipeline from SharePoint retrieval to document parsing.
"""

import os
import pytest
import json
from typing import Dict, Any, List
from unittest.mock import Mock, patch

from go_doc_go.adapter.sharepoint import SharePointAdapter
from go_doc_go.content_source.sharepoint import SharePointContentSource
from go_doc_go.document_parser.factory import get_parser_for_content


# Skip all tests if SharePoint not configured
pytestmark = pytest.mark.skipif(
    not os.getenv('SHAREPOINT_TENANT_ID'),
    reason="SharePoint credentials not configured"
)


@pytest.mark.integration
class TestSharePointIntegration:
    """Integration tests for SharePoint document processing."""
    
    @pytest.fixture
    def sharepoint_config(self):
        """Get SharePoint configuration."""
        return {
            'tenant_id': os.getenv('SHAREPOINT_TENANT_ID'),
            'client_id': os.getenv('SHAREPOINT_CLIENT_ID'),
            'client_secret': os.getenv('SHAREPOINT_CLIENT_SECRET'),
            'site_url': os.getenv('SHAREPOINT_SITE_URL'),
            'api_type': os.getenv('SHAREPOINT_API_TYPE', 'auto')
        }
    
    @pytest.fixture
    def test_documents_path(self):
        """Get test documents path."""
        return os.getenv('SHAREPOINT_TEST_PATH', '/Shared Documents')
    
    @pytest.fixture
    def adapter(self, sharepoint_config):
        """Create SharePoint adapter."""
        if not all([sharepoint_config['tenant_id'], sharepoint_config['client_id'],
                   sharepoint_config['client_secret'], sharepoint_config['site_url']]):
            pytest.skip("SharePoint credentials not fully configured")
        return SharePointAdapter(sharepoint_config)
    
    @pytest.fixture
    def content_source(self, sharepoint_config):
        """Create SharePoint content source."""
        site_url = sharepoint_config['site_url']
        test_path = os.getenv('SHAREPOINT_TEST_PATH', '/Shared Documents')
        
        if not all([sharepoint_config['tenant_id'], sharepoint_config['client_id'],
                   sharepoint_config['client_secret'], site_url]):
            pytest.skip("SharePoint credentials not fully configured")
        
        return SharePointContentSource({
            **sharepoint_config,
            'site_url': site_url,
            'libraries': [test_path.replace('/', '').replace('/Shared Documents', 'Shared Documents')],
            'max_items': 10,
            'include_subfolders': True
        })
    
    def test_adapter_initialization(self, adapter):
        """Test that adapter initializes correctly."""
        assert adapter is not None
        assert adapter.site_url is not None
        assert adapter.api_type in ['rest', 'graph']
    
    def test_content_source_initialization(self, content_source):
        """Test that content source initializes correctly."""
        assert content_source is not None
        assert content_source.site_url is not None
    
    @pytest.mark.slow
    def test_retrieve_document_via_adapter(self, adapter, test_documents_path):
        """Test retrieving a document using the adapter."""
        # This test requires a known document to exist
        # We'll skip if no documents are found
        
        try:
            # Try to retrieve from test path
            location = {'source': test_documents_path}
            
            # This might fail if it's a folder not a file
            # In real tests, you'd have a known test file
            pytest.skip("Requires known test file in SharePoint")
            
        except Exception as e:
            if "not found" in str(e).lower():
                pytest.skip(f"Test document not found: {e}")
            raise
    
    @pytest.mark.slow
    def test_list_documents_via_content_source(self, content_source):
        """Test listing documents via content source."""
        documents = content_source.list_documents()
        
        # Should return a list (might be empty)
        assert isinstance(documents, list)
        
        # If we have documents, validate structure
        if documents:
            doc = documents[0]
            assert 'id' in doc
            assert 'metadata' in doc
            
            metadata = doc['metadata']
            assert 'name' in metadata or 'title' in metadata
    
    @pytest.mark.slow
    def test_document_processing_pipeline(self, content_source):
        """Test full pipeline from retrieval to parsing."""
        # Get list of documents
        documents = content_source.list_documents()
        
        if not documents:
            pytest.skip("No documents found in test library")
        
        # Find a parseable document (text, docx, pdf, etc.)
        test_doc = None
        for doc in documents[:5]:  # Check first 5 documents
            doc_type = doc.get('doc_type', '')
            if doc_type in ['text', 'docx', 'pdf', 'html']:
                test_doc = doc
                break
        
        if not test_doc:
            pytest.skip("No parseable documents found")
        
        # Fetch the document content
        doc_id = test_doc['id']
        fetched = content_source.fetch_document(doc_id)
        
        assert 'content' in fetched or 'binary_content' in fetched
        assert 'doc_type' in fetched
        assert 'metadata' in fetched
        
        # Parse the document
        parser = get_parser_for_content(fetched)
        if parser:
            # Prepare content for parser
            parse_input = {
                'id': fetched['id'],
                'content': fetched.get('content', ''),
                'metadata': fetched['metadata']
            }
            
            if fetched.get('binary_content'):
                parse_input['binary_content'] = fetched['binary_content']
            
            result = parser.parse(parse_input)
            
            # Validate parse result
            assert 'document' in result
            assert 'elements' in result
            assert 'relationships' in result
            
            # Should have at least one element
            assert len(result['elements']) > 0
    
    @pytest.mark.slow
    def test_api_type_switching(self, sharepoint_config):
        """Test switching between REST and Graph APIs."""
        # Test REST API
        try:
            from office365.sharepoint.client_context import ClientContext
            rest_config = {**sharepoint_config, 'api_type': 'rest'}
            rest_adapter = SharePointAdapter(rest_config)
            assert rest_adapter.api_type == 'rest'
        except ImportError:
            pass  # REST client not installed
        
        # Test Graph API
        try:
            from msgraph import GraphServiceClient
            graph_config = {**sharepoint_config, 'api_type': 'graph'}
            graph_adapter = SharePointAdapter(graph_config)
            assert graph_adapter.api_type == 'graph'
        except ImportError:
            pass  # Graph client not installed
    
    @pytest.mark.slow
    def test_metadata_extraction(self, content_source):
        """Test that metadata is properly extracted."""
        documents = content_source.list_documents()
        
        if not documents:
            pytest.skip("No documents found")
        
        # Check first document's metadata
        doc = documents[0]
        metadata = doc['metadata']
        
        # Should have basic metadata
        assert any(key in metadata for key in ['name', 'title'])
        assert any(key in metadata for key in ['server_relative_url', 'url'])
        
        # Fetch full document to get more metadata
        fetched = content_source.fetch_document(doc['id'])
        full_metadata = fetched['metadata']
        
        # Should have additional metadata after fetch
        assert any(key in full_metadata for key in ['last_modified', 'size'])
    
    @pytest.mark.slow
    def test_binary_file_handling(self, adapter):
        """Test handling of binary files like images or PDFs."""
        # This requires a known binary file in SharePoint
        pytest.skip("Requires known binary test file")
    
    @pytest.mark.slow
    def test_large_library_handling(self, content_source):
        """Test handling of large document libraries."""
        # Reconfigure for larger test
        content_source.max_items = 100
        
        documents = content_source.list_documents()
        
        # Should respect max_items limit
        assert len(documents) <= 100
    
    @pytest.mark.slow
    def test_error_handling(self, adapter):
        """Test error handling for invalid requests."""
        # Test non-existent file
        with pytest.raises(ValueError):
            adapter.get_content({
                'source': '/NonExistent/DoesNotExist_xyz123.docx'
            })
    
    @pytest.mark.slow
    def test_authentication_methods(self, sharepoint_config):
        """Test different authentication methods."""
        # Currently only testing client credentials
        # User credentials would require different config
        
        adapter = SharePointAdapter(sharepoint_config)
        assert adapter is not None
        
        # Future: test user credentials, certificate auth, etc.


@pytest.mark.integration
@pytest.mark.slow
class TestSharePointDocumentTypes:
    """Test processing of different document types from SharePoint."""
    
    @pytest.fixture
    def setup(self, sharepoint_config):
        """Setup for document type tests."""
        if not all([sharepoint_config.get('tenant_id'), 
                   sharepoint_config.get('client_id'),
                   sharepoint_config.get('client_secret'),
                   sharepoint_config.get('site_url')]):
            pytest.skip("SharePoint not fully configured")
        
        return SharePointContentSource({
            **sharepoint_config,
            'max_items': 50,
            'include_subfolders': True
        })
    
    def test_docx_processing(self, setup):
        """Test processing of Word documents."""
        content_source = setup
        documents = content_source.list_documents()
        
        # Find a .docx file
        docx_doc = None
        for doc in documents:
            if doc.get('doc_type') == 'docx' or \
               doc['metadata'].get('extension') == 'docx':
                docx_doc = doc
                break
        
        if not docx_doc:
            pytest.skip("No Word documents found")
        
        # Process the document
        fetched = content_source.fetch_document(docx_doc['id'])
        assert fetched['doc_type'] == 'docx'
    
    def test_xlsx_processing(self, setup):
        """Test processing of Excel documents."""
        content_source = setup
        documents = content_source.list_documents()
        
        # Find an .xlsx file
        xlsx_doc = None
        for doc in documents:
            if doc.get('doc_type') == 'xlsx' or \
               doc['metadata'].get('extension') == 'xlsx':
                xlsx_doc = doc
                break
        
        if not xlsx_doc:
            pytest.skip("No Excel documents found")
        
        # Process the document
        fetched = content_source.fetch_document(xlsx_doc['id'])
        assert fetched['doc_type'] == 'xlsx'
    
    def test_pdf_processing(self, setup):
        """Test processing of PDF documents."""
        content_source = setup
        documents = content_source.list_documents()
        
        # Find a .pdf file
        pdf_doc = None
        for doc in documents:
            if doc.get('doc_type') == 'pdf' or \
               doc['metadata'].get('extension') == 'pdf':
                pdf_doc = doc
                break
        
        if not pdf_doc:
            pytest.skip("No PDF documents found")
        
        # Process the document
        fetched = content_source.fetch_document(pdf_doc['id'])
        assert fetched['doc_type'] == 'pdf'