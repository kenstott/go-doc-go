"""
Google Drive adapter for document retrieval.

This adapter provides integration with Google Drive for fetching documents.
"""

import logging
import re
from typing import Dict, Any, Optional
from urllib.parse import urlparse, parse_qs

from .base import ContentSourceAdapter
from ..content_source.google_drive import GoogleDriveContentSource

logger = logging.getLogger(__name__)


class GoogleDriveAdapter(ContentSourceAdapter):
    """Adapter for fetching documents from Google Drive."""
    
    def __init__(self, config: Dict[str, Any]):
        """
        Initialize Google Drive adapter.
        
        Args:
            config: Configuration dictionary containing:
                - auth_type: 'service_account' or 'oauth'
                - service_account_file: Path to service account JSON (for service account auth)
                - credentials_path: Path to OAuth credentials (for OAuth auth)
                - token_path: Path to store OAuth token (for OAuth auth)
                - scopes: List of Google API scopes
                - impersonate_user: Email to impersonate (optional, for domain-wide delegation)
        """
        super().__init__(config)
        
        # Initialize content source
        self.content_source = GoogleDriveContentSource(config)
        
        logger.info("Google Drive adapter initialized")
    
    def fetch_document(self, source: str) -> Dict[str, Any]:
        """
        Fetch a document from Google Drive.
        
        Args:
            source: Document identifier, can be:
                - File ID (e.g., "1234567890abcdef")
                - gdrive:// URI (e.g., "gdrive://1234567890abcdef")
                - Full Google Drive URL (e.g., "https://drive.google.com/file/d/1234567890abcdef/view")
                - Google Docs URL (e.g., "https://docs.google.com/document/d/1234567890abcdef/edit")
        
        Returns:
            Document dictionary with content and metadata
        
        Raises:
            Exception: If document cannot be fetched
        """
        # Extract file ID from various formats
        file_id = self._parse_uri(source)
        
        logger.debug(f"Fetching Google Drive document: {file_id}")
        
        # Fetch document through content source
        return self.content_source.fetch_document(file_id)
    
    def _parse_uri(self, uri: str) -> str:
        """
        Parse various Google Drive URI formats to extract file ID.
        
        Args:
            uri: URI in various formats
        
        Returns:
            Google Drive file ID
        """
        # If it's already just a file ID, return it
        if not any(prefix in uri for prefix in ['://', 'http', 'gdrive']):
            return uri
        
        # Handle gdrive:// URIs
        if uri.startswith('gdrive://'):
            return uri.replace('gdrive://', '')
        
        # Handle various Google URLs
        patterns = [
            # Google Drive file URL
            r'/file/d/([a-zA-Z0-9_-]+)',
            # Google Drive folder URL
            r'/folders/([a-zA-Z0-9_-]+)',
            # Google Docs URL
            r'/document/d/([a-zA-Z0-9_-]+)',
            # Google Sheets URL
            r'/spreadsheets/d/([a-zA-Z0-9_-]+)',
            # Google Slides URL
            r'/presentation/d/([a-zA-Z0-9_-]+)',
            # Old style with ?id=
            r'[?&]id=([a-zA-Z0-9_-]+)',
            # Direct file ID in open URL
            r'/open\?id=([a-zA-Z0-9_-]+)',
        ]
        
        for pattern in patterns:
            match = re.search(pattern, uri)
            if match:
                return match.group(1)
        
        # If no pattern matches, assume it's a file ID
        logger.warning(f"Could not parse Google Drive URI: {uri}, treating as file ID")
        return uri
    
    def get_safe_connection_string(self) -> str:
        """
        Get a safe connection string for logging.
        
        Returns:
            Connection string with sensitive information redacted
        """
        return self.content_source.get_safe_connection_string()
    
    def test_connection(self) -> bool:
        """
        Test the connection to Google Drive.
        
        Returns:
            True if connection is successful, False otherwise
        """
        try:
            # Try to list a few documents to test the connection
            documents = list(self.content_source.get_documents())
            logger.info(f"Google Drive connection test successful, found {len(documents)} documents")
            return True
        except Exception as e:
            logger.error(f"Google Drive connection test failed: {str(e)}")
            return False
    
    def get_content(self, location_data: Dict[str, Any]) -> Dict[str, Any]:
        """
        Get content from Google Drive.
        
        Args:
            location_data: Dictionary containing 'source' or 'file_id'
        
        Returns:
            Document dictionary with content and metadata
        """
        source = location_data.get('source') or location_data.get('file_id')
        if not source:
            raise ValueError("No source or file_id provided in location_data")
        
        return self.fetch_document(source)
    
    def supports_location(self, location_data: Dict[str, Any]) -> bool:
        """
        Check if this adapter supports the location.
        
        Args:
            location_data: Dictionary containing location information
        
        Returns:
            True if this adapter can handle the location
        """
        source = location_data.get('source', '')
        
        # Support gdrive:// URIs
        if source.startswith('gdrive://'):
            return True
        
        # Support Google Drive URLs
        if 'drive.google.com' in source or 'docs.google.com' in source:
            return True
        
        # Support if explicitly marked as Google Drive
        if location_data.get('type') == 'google_drive':
            return True
        
        return False
    
    def get_binary_content(self, location_data: Dict[str, Any]) -> bytes:
        """
        Get the content as binary data.
        
        Args:
            location_data: Dictionary containing 'source' or 'file_id'
        
        Returns:
            Binary content
        """
        document = self.get_content(location_data)
        content = document.get('content', b'')
        
        # If content is string, encode it
        if isinstance(content, str):
            return content.encode('utf-8')
        
        return content