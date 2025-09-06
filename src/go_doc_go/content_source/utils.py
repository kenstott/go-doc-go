"""
Utility functions for document parsing.

This module provides helper functions for detecting content types and routing documents
to the appropriate parser.
"""

import logging
import re
from typing import Dict, Any

logger = logging.getLogger(__name__)


def detect_content_type(content: str, metadata: Dict[str, Any] = None) -> str:
    """
    Detect content type from content and metadata.

    Args:
        content: Document content
        metadata: Optional metadata that might contain content type information

    Returns:
        Content type: 'json', 'csv', 'xml', 'markdown', 'html', or 'text'
    """
    metadata = metadata or {}

    # Check metadata first
    content_type = metadata.get("content_type", "").lower()
    if content_type:
        if "json" in content_type:
            return "json"
        elif "csv" in content_type:
            return "csv"
        elif "xml" in content_type:
            return "xml"
        elif "markdown" in content_type or "md" in content_type:
            return "markdown"
        elif "html" in content_type or "xhtml" in content_type:
            return "html"

    # Try to detect JSON
    content_stripped = content.strip()
    if content_stripped and content_stripped[0] in '{[' and content_stripped[-1] in '}]':
        try:
            import json
            json.loads(content)
            return "json"
        except (json.JSONDecodeError, ValueError):
            pass

    # Try to detect XML
    if content_stripped.startswith('<?xml') or (content_stripped.startswith('<') and 
                                                 content_stripped.endswith('>') and 
                                                 '</' in content_stripped):
        # Simple XML detection - check for XML declaration or matching tags
        if re.search(r'<(\w+)[^>]*>.*?</\1>', content_stripped, re.DOTALL):
            return "xml"

    # Check for HTML specifically
    if content_stripped.startswith(('<!DOCTYPE html>', '<html')):
        return "html"

    # Try to detect CSV
    lines = content_stripped.split('\n')
    if len(lines) > 1:
        # Check for consistent delimiters (comma, tab, semicolon, pipe)
        for delimiter in [',', '\t', ';', '|']:
            first_line_count = lines[0].count(delimiter)
            if first_line_count > 0:
                # Check if first few lines have consistent delimiter count
                consistent = True
                for line in lines[1:min(5, len(lines))]:
                    if line.strip() and abs(line.count(delimiter) - first_line_count) > 1:
                        consistent = False
                        break
                if consistent:
                    return "csv"

    # Count markdown-specific patterns
    md_patterns = 0
    if re.search(r'^#{1,6}\s+', content, re.MULTILINE):  # Headers
        md_patterns += 1
    if re.search(r'^[-*]\s+', content, re.MULTILINE):  # List items
        md_patterns += 1
    if re.search(r'\[.+?\]\(.+?\)', content):  # Links
        md_patterns += 1
    if re.search(r'^```', content, re.MULTILINE):  # Code blocks
        md_patterns += 1
    if re.search(r'^\|.+\|', content, re.MULTILINE):  # Tables
        md_patterns += 1

    if md_patterns >= 2:
        return "markdown"

    # Count HTML-specific patterns
    html_patterns = 0
    if re.search(r'<[a-z]+[^>]*>', content):  # Opening tags
        html_patterns += 1
    if re.search(r'</[a-z]+>', content):  # Closing tags
        html_patterns += 1
    if re.search(r'<[a-z]+[^>]*/>', content):  # Self-closing tags
        html_patterns += 1

    if html_patterns >= 2:
        return "html"

    # Default to text
    return "text"


def extract_url_links(content: str, element_id: str) -> list:
    """
    Extract URL links from plain text.

    Args:
        content: Text content
        element_id: Source element ID

    Returns:
        List of link objects
    """
    links = []

    # Simple URL pattern for plain text
    url_pattern = r'https?://[^\s()<>]+(?:\([\w\d]+\)|(?:[^,.;:`!()\[\]{}<>"\'\s]|/))'

    matches = re.findall(url_pattern, content)

    for url in matches:
        links.append({
            "source_id": element_id,
            "link_text": url,  # Use the URL as the text for plain text links
            "link_target": url,
            "link_type": "url"
        })

    return links
