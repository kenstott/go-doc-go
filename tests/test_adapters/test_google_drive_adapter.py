"""
Tests for Google Drive adapter.

These are primarily integration tests since mocking Google Drive API is complex.
"""

import os
import pytest
from unittest.mock import Mock, patch, MagicMock
from typing import Dict, Any

# The adapter might not exist yet, so we'll check for it
try:
    from go_doc_go.adapter.google_drive import GoogleDriveAdapter
    ADAPTER_EXISTS = True
except ImportError:
    ADAPTER_EXISTS = False
    GoogleDriveAdapter = None


@pytest.mark.skipif(not ADAPTER_EXISTS, reason="GoogleDriveAdapter not implemented yet")
class TestGoogleDriveAdapterUnit:
    """Unit tests for Google Drive adapter."""
    
    @patch('go_doc_go.adapter.google_drive.GoogleDriveContentSource')
    def test_parse_google_drive_uri(self, mock_content_source):
        """Test parsing Google Drive URIs."""
        if not GoogleDriveAdapter:
            pytest.skip("GoogleDriveAdapter not implemented")
        
        # Mock the content source to avoid initialization issues
        mock_instance = MagicMock()
        mock_content_source.return_value = mock_instance
        
        adapter = GoogleDriveAdapter({})
        
        # Test file ID extraction from various URL formats
        test_cases = [
            ("gdrive://1234567890abcdef", "1234567890abcdef"),
            ("https://drive.google.com/file/d/1234567890abcdef/view", "1234567890abcdef"),
            ("https://docs.google.com/document/d/1234567890abcdef/edit", "1234567890abcdef"),
            ("https://drive.google.com/open?id=1234567890abcdef", "1234567890abcdef"),
        ]
        
        for uri, expected_id in test_cases:
            file_id = adapter._parse_uri(uri)
            assert file_id == expected_id, f"Failed to parse {uri}"
    
    def test_init_with_content_source(self):
        """Test adapter initialization with content source."""
        if not GoogleDriveAdapter:
            pytest.skip("GoogleDriveAdapter not implemented")
        
        config = {
            'auth_type': 'service_account',
            'service_account_file': '/path/to/sa.json'
        }
        
        with patch('go_doc_go.adapter.google_drive.GoogleDriveContentSource') as mock_source:
            adapter = GoogleDriveAdapter(config)
            mock_source.assert_called_once_with(config)
            assert adapter.content_source is not None


@pytest.mark.integration
class TestGoogleDriveAdapterIntegration:
    """Integration tests for Google Drive adapter."""
    
    @pytest.fixture
    def google_drive_config(self) -> Dict[str, Any]:
        """Get Google Drive configuration from environment."""
        service_account_file = os.getenv('GOOGLE_DRIVE_SERVICE_ACCOUNT_FILE')
        credentials_path = os.getenv('GOOGLE_DRIVE_CREDENTIALS_PATH')
        
        if service_account_file and os.path.exists(service_account_file):
            return {
                'auth_type': 'service_account',
                'service_account_file': service_account_file,
                'scopes': ['https://www.googleapis.com/auth/drive.readonly']
            }
        elif credentials_path and os.path.exists(credentials_path):
            token_path = os.getenv('GOOGLE_DRIVE_TOKEN_PATH', 'token.pickle')
            return {
                'auth_type': 'oauth',
                'credentials_path': credentials_path,
                'token_path': token_path,
                'scopes': ['https://www.googleapis.com/auth/drive.readonly']
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
    
    def test_fetch_document(self, google_drive_config, test_file_id):
        """Test fetching a document through the adapter."""
        if not ADAPTER_EXISTS:
            pytest.skip("GoogleDriveAdapter not implemented")
        
        try:
            from googleapiclient.discovery import build
        except ImportError:
            pytest.skip("Google API packages not installed")
        
        adapter = GoogleDriveAdapter(google_drive_config)
        
        # Test with direct file ID
        document = adapter.fetch_document(test_file_id)
        assert document is not None
        assert 'content' in document or 'binary_path' in document
        assert 'metadata' in document
        
        # Test with gdrive:// URI
        gdrive_uri = f"gdrive://{test_file_id}"
        document = adapter.fetch_document(gdrive_uri)
        assert document is not None
        
        # Test with full URL
        drive_url = f"https://drive.google.com/file/d/{test_file_id}/view"
        document = adapter.fetch_document(drive_url)
        assert document is not None
    
    def test_file_not_found(self, google_drive_config):
        """Test handling of non-existent files."""
        if not ADAPTER_EXISTS:
            pytest.skip("GoogleDriveAdapter not implemented")
        
        try:
            from googleapiclient.discovery import build
        except ImportError:
            pytest.skip("Google API packages not installed")
        
        adapter = GoogleDriveAdapter(google_drive_config)
        
        # Should raise an exception for non-existent file
        with pytest.raises(Exception):
            adapter.fetch_document("nonexistent_file_id_12345")
    
    def test_google_doc_export(self, google_drive_config):
        """Test exporting Google Docs to different formats."""
        if not ADAPTER_EXISTS:
            pytest.skip("GoogleDriveAdapter not implemented")
        
        google_doc_id = os.getenv('GOOGLE_DRIVE_TEST_GOOGLE_DOC_ID')
        if not google_doc_id:
            pytest.skip("No Google Doc ID provided")
        
        # Extract ID from URL if needed
        google_doc_id = self._extract_id_from_url(google_doc_id)
        
        try:
            from googleapiclient.discovery import build
        except ImportError:
            pytest.skip("Google API packages not installed")
        
        adapter = GoogleDriveAdapter(google_drive_config)
        
        # Fetch Google Doc (should be exported as HTML by default)
        document = adapter.fetch_document(google_doc_id)
        
        assert document is not None
        assert 'content' in document
        assert len(document['content']) > 0
        
        # Check metadata
        metadata = document.get('metadata', {})
        assert 'name' in metadata
        assert 'doc_type' in document or 'mime_type' in metadata