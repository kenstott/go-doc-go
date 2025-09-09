"""
Tests for Google Drive content source.

These are primarily integration tests since mocking Google Drive API is complex.
Requires proper Google Drive credentials to run.
"""

import os
import pytest
from unittest.mock import Mock, patch, MagicMock
from typing import Dict, Any

from go_doc_go.content_source.google_drive import GoogleDriveContentSource


class TestGoogleDriveContentSourceUnit:
    """Unit tests for Google Drive content source with mocked dependencies."""
    
    @pytest.fixture
    def mock_drive_service(self):
        """Create a mock Google Drive service."""
        service = MagicMock()
        return service
    
    @pytest.fixture
    def mock_credentials(self):
        """Create mock Google credentials."""
        creds = MagicMock()
        creds.valid = True
        creds.expired = False
        return creds
    
    def test_init_with_missing_packages(self):
        """Test initialization when Google API packages are not available."""
        with patch('go_doc_go.content_source.google_drive.GOOGLE_API_AVAILABLE', False):
            with pytest.raises(ImportError, match="Google API packages are required"):
                GoogleDriveContentSource({})
    
    @patch('go_doc_go.content_source.google_drive.GOOGLE_API_AVAILABLE', True)
    @patch('go_doc_go.content_source.google_drive.build')
    def test_init_with_service_account(self, mock_build, mock_credentials):
        """Test initialization with service account authentication."""
        mock_build.return_value = MagicMock()
        
        with patch.object(GoogleDriveContentSource, '_get_credentials', return_value=mock_credentials):
            config = {
                'auth_type': 'service_account',
                'service_account_file': '/path/to/service-account.json',
                'scopes': ['https://www.googleapis.com/auth/drive.readonly']
            }
            
            source = GoogleDriveContentSource(config)
            
            assert source.auth_type == 'service_account'
            assert source.service_account_file == '/path/to/service-account.json'
            assert source.drive_service is not None
    
    @patch('go_doc_go.content_source.google_drive.GOOGLE_API_AVAILABLE', True)
    @patch('go_doc_go.content_source.google_drive.build')
    def test_init_with_oauth(self, mock_build, mock_credentials):
        """Test initialization with OAuth authentication."""
        mock_build.return_value = MagicMock()
        
        with patch.object(GoogleDriveContentSource, '_get_credentials', return_value=mock_credentials):
            config = {
                'auth_type': 'oauth',
                'credentials_path': '/path/to/credentials.json',
                'token_path': '/path/to/token.pickle'
            }
            
            source = GoogleDriveContentSource(config)
            
            assert source.auth_type == 'oauth'
            assert source.credentials_path == '/path/to/credentials.json'
            assert source.token_path == '/path/to/token.pickle'
    
    @patch('go_doc_go.content_source.google_drive.GOOGLE_API_AVAILABLE', True)
    @patch('go_doc_go.content_source.google_drive.build')
    def test_get_safe_connection_string(self, mock_build, mock_credentials):
        """Test safe connection string generation."""
        mock_build.return_value = MagicMock()
        
        with patch.object(GoogleDriveContentSource, '_get_credentials', return_value=mock_credentials):
            # Test OAuth
            oauth_config = {'auth_type': 'oauth'}
            source = GoogleDriveContentSource(oauth_config)
            conn_str = source.get_safe_connection_string()
            assert 'OAuth' in conn_str
            assert 'token' in conn_str.lower()
            
            # Test Service Account
            sa_config = {
                'auth_type': 'service_account',
                'service_account_file': 'sa.json',
                'impersonate_user': 'user@example.com'
            }
            source = GoogleDriveContentSource(sa_config)
            conn_str = source.get_safe_connection_string()
            assert 'Service Account' in conn_str
            assert 'user@example.com' in conn_str


@pytest.mark.integration
class TestGoogleDriveContentSourceIntegration:
    """Integration tests for Google Drive content source."""
    
    @pytest.fixture
    def google_drive_config(self) -> Dict[str, Any]:
        """Get Google Drive configuration from environment variables."""
        # Check which authentication method is available
        service_account_file = os.getenv('GOOGLE_DRIVE_SERVICE_ACCOUNT_FILE')
        credentials_path = os.getenv('GOOGLE_DRIVE_CREDENTIALS_PATH')
        
        if service_account_file and os.path.exists(service_account_file):
            # Use service account authentication
            return {
                'auth_type': 'service_account',
                'service_account_file': service_account_file,
                'scopes': ['https://www.googleapis.com/auth/drive.readonly'],
                'include_shared': True,
                'max_results': 10
            }
        elif credentials_path and os.path.exists(credentials_path):
            # Use OAuth authentication
            token_path = os.getenv('GOOGLE_DRIVE_TOKEN_PATH', 'token.pickle')
            return {
                'auth_type': 'oauth',
                'credentials_path': credentials_path,
                'token_path': token_path,
                'scopes': ['https://www.googleapis.com/auth/drive.readonly'],
                'include_shared': True,
                'max_results': 10
            }
        else:
            pytest.skip("Google Drive credentials not configured")
    
    def _extract_id_from_url(self, url_or_id: str) -> str:
        """Extract file/folder ID from Google Drive URL or return as-is if already an ID."""
        if not url_or_id:
            return None
        
        # If it doesn't look like a URL, assume it's already an ID
        if not url_or_id.startswith('http'):
            return url_or_id
        
        # Patterns to extract IDs from various Google URLs
        import re
        patterns = [
            r'/file/d/([a-zA-Z0-9_-]+)',  # Drive file URL
            r'/folders/([a-zA-Z0-9_-]+)',  # Drive folder URL
            r'/document/d/([a-zA-Z0-9_-]+)',  # Docs URL
            r'/spreadsheets/d/([a-zA-Z0-9_-]+)',  # Sheets URL
            r'/presentation/d/([a-zA-Z0-9_-]+)',  # Slides URL
            r'[?&]id=([a-zA-Z0-9_-]+)',  # Old style with ?id=
        ]
        
        for pattern in patterns:
            match = re.search(pattern, url_or_id)
            if match:
                return match.group(1)
        
        return url_or_id  # Return as-is if no pattern matches
    
    @pytest.fixture
    def test_file_id(self) -> str:
        """Get test file ID from environment."""
        file_id = os.getenv('GOOGLE_DRIVE_TEST_FILE_ID')
        if not file_id:
            pytest.skip("GOOGLE_DRIVE_TEST_FILE_ID not set")
        return self._extract_id_from_url(file_id)
    
    @pytest.fixture
    def test_folder_id(self) -> str:
        """Get test folder ID from environment."""
        folder_id = os.getenv('GOOGLE_DRIVE_TEST_FOLDER_ID')
        if not folder_id:
            # Optional - tests can work without folder
            return None
        return self._extract_id_from_url(folder_id)
    
    def test_connection(self, google_drive_config):
        """Test basic connection to Google Drive."""
        try:
            from googleapiclient.discovery import build
        except ImportError:
            pytest.skip("Google API packages not installed")
        
        source = GoogleDriveContentSource(google_drive_config)
        
        # Verify service is initialized
        assert source.drive_service is not None
        
        # Try to list files (basic API call)
        documents = list(source.get_documents())
        
        # Should not raise an exception
        assert isinstance(documents, list)
    
    def test_list_documents(self, google_drive_config):
        """Test listing documents from Google Drive."""
        try:
            from googleapiclient.discovery import build
        except ImportError:
            pytest.skip("Google API packages not installed")
        
        source = GoogleDriveContentSource(google_drive_config)
        documents = list(source.get_documents())
        
        # Should return a list
        assert isinstance(documents, list)
        
        # If documents exist, validate structure
        if documents:
            doc = documents[0]
            assert 'id' in doc
            assert 'metadata' in doc
            
            metadata = doc['metadata']
            assert 'name' in metadata
            assert 'mime_type' in metadata or 'mimeType' in metadata
    
    def test_fetch_specific_document(self, google_drive_config, test_file_id):
        """Test fetching a specific document by ID."""
        try:
            from googleapiclient.discovery import build
        except ImportError:
            pytest.skip("Google API packages not installed")
        
        source = GoogleDriveContentSource(google_drive_config)
        
        # Fetch the test document
        document = source.fetch_document(test_file_id)
        
        # Validate document structure
        assert document is not None
        assert 'id' in document
        assert 'content' in document or 'binary_path' in document
        assert 'metadata' in document
        
        metadata = document['metadata']
        assert 'name' in metadata
        assert 'size' in metadata or 'file_size' in metadata
    
    def test_folder_filtering(self, google_drive_config, test_folder_id):
        """Test filtering documents by folder."""
        if not test_folder_id:
            pytest.skip("Test folder ID not provided")
        
        try:
            from googleapiclient.discovery import build
        except ImportError:
            pytest.skip("Google API packages not installed")
        
        # Configure to only get files from specific folder
        config = {**google_drive_config, 'folders': [test_folder_id]}
        source = GoogleDriveContentSource(config)
        
        documents = list(source.get_documents())
        
        # Should return documents
        assert isinstance(documents, list)
        
        # If we have documents, they should be from the specified folder
        # (This would need actual validation against the API)
    
    def test_mime_type_filtering(self, google_drive_config):
        """Test filtering documents by MIME type."""
        try:
            from googleapiclient.discovery import build
        except ImportError:
            pytest.skip("Google API packages not installed")
        
        # Configure to only get specific file types
        config = {
            **google_drive_config,
            'file_types': ['application/pdf', 'text/plain']
        }
        source = GoogleDriveContentSource(config)
        
        documents = list(source.get_documents())
        
        # Should return documents
        assert isinstance(documents, list)
        
        # If we have documents, check their types
        for doc in documents:
            if 'metadata' in doc and 'mime_type' in doc['metadata']:
                mime_type = doc['metadata']['mime_type']
                # Should match our filter (or be a Google Docs type that exports to our filter)
                assert (mime_type in config['file_types'] or 
                       mime_type in source.GOOGLE_DOCUMENT_MIME_TYPES)
    
    def test_error_handling_invalid_file_id(self, google_drive_config):
        """Test error handling for invalid file ID."""
        try:
            from googleapiclient.discovery import build
        except ImportError:
            pytest.skip("Google API packages not installed")
        
        source = GoogleDriveContentSource(google_drive_config)
        
        # Try to fetch a non-existent file
        with pytest.raises(Exception):  # Could be HttpError or wrapped exception
            source.fetch_document("invalid_file_id_that_does_not_exist")
    
    def test_google_docs_export(self, google_drive_config):
        """Test exporting Google Docs in different formats."""
        try:
            from googleapiclient.discovery import build
        except ImportError:
            pytest.skip("Google API packages not installed")
        
        # This test would need a known Google Doc ID
        # It's optional since it requires specific test data
        google_doc_id = os.getenv('GOOGLE_DRIVE_TEST_GOOGLE_DOC_ID')
        if not google_doc_id:
            pytest.skip("No Google Doc ID provided for testing")
        
        # Extract ID from URL if needed
        google_doc_id = self._extract_id_from_url(google_doc_id)
        
        source = GoogleDriveContentSource(google_drive_config)
        document = source.fetch_document(google_doc_id)
        
        # Should have content (exported from Google Docs)
        assert 'content' in document
        assert document['content'] is not None
        assert len(document['content']) > 0
    
    def test_shared_drive_support(self, google_drive_config):
        """Test support for shared drives."""
        try:
            from googleapiclient.discovery import build
        except ImportError:
            pytest.skip("Google API packages not installed")
        
        # Enable shared drive support
        config = {**google_drive_config, 'include_shared': True}
        source = GoogleDriveContentSource(config)
        
        documents = list(source.get_documents())
        
        # Should not raise an error
        assert isinstance(documents, list)
        
        # Check if any documents are from shared drives
        # (This is informational, not a hard requirement)
        shared_count = sum(1 for doc in documents 
                          if doc.get('metadata', {}).get('shared', False))
        print(f"Found {shared_count} shared documents out of {len(documents)} total")