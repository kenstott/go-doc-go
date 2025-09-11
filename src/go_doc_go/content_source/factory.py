"""
Content Source Module for the document pointer system.
This module contains adapters for different content sources such as:
- Markdown files
- Database blobs
- Web URLs
- Confluence
- JIRA
- S3
- ServiceNow
- MongoDB
- SharePoint
- Google Drive
"""
import logging
from typing import Dict, Any

from .base import ContentSource
from .confluence import ConfluenceContentSource
from .database import DatabaseContentSource
from .file import FileContentSource
from .google_drive import GoogleDriveContentSource
from .jira import JiraContentSource
from .mongodb import MongoDBContentSource
from .s3 import S3ContentSource
from .servicenow import ServiceNowContentSource
from .sharepoint import SharePointContentSource
from .web import WebContentSource

logger = logging.getLogger(__name__)

# Global registry for content sources by name
_content_source_registry = {}

def get_content_source(source_config: Dict[str, Any]) -> ContentSource:
    """
    Factory function to create appropriate content source from config.

    Args:
        source_config: Content source configuration

    Returns:
        ContentSource instance

    Raises:
        ValueError: If source type is not supported
    """
    source_type = source_config.get("type")

    if source_type == "file":
        return FileContentSource(source_config)
    elif source_type == "database":
        return DatabaseContentSource(source_config)
    elif source_type == "web":
        return WebContentSource(source_config)
    elif source_type == "confluence":
        return ConfluenceContentSource(source_config)
    elif source_type == "jira":
        return JiraContentSource(source_config)
    elif source_type == "s3":
        return S3ContentSource(source_config)
    elif source_type == "servicenow":
        return ServiceNowContentSource(source_config)
    elif source_type == "mongodb":
        return MongoDBContentSource(source_config)
    elif source_type == "sharepoint":
        return SharePointContentSource(source_config)
    elif source_type == "google_drive":
        return GoogleDriveContentSource(source_config)
    else:
        raise ValueError(f"Unsupported content source type: {source_type}")


def register_content_source(name: str, source: ContentSource) -> None:
    """
    Register a content source instance by name for queue-based processing.
    
    Args:
        name: Unique name for the content source
        source: ContentSource instance
    """
    _content_source_registry[name] = source
    logger.debug(f"Registered content source: {name}")


def get_content_source_by_name(name: str) -> ContentSource:
    """
    Get a registered content source by name.
    
    Args:
        name: Name of the registered content source
        
    Returns:
        ContentSource instance
        
    Raises:
        ValueError: If source name is not registered
    """
    if name not in _content_source_registry:
        raise ValueError(f"Content source '{name}' not registered. Available sources: {list(_content_source_registry.keys())}")
    
    return _content_source_registry[name]


def clear_content_source_registry() -> None:
    """Clear all registered content sources."""
    _content_source_registry.clear()
    logger.debug("Cleared content source registry")
