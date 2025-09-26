import logging
import os
import re
from typing import Dict, Any, List, Optional
from urllib.parse import urljoin, urlparse

import requests
from bs4 import BeautifulSoup

from .base import ContentSource

logger = logging.getLogger(__name__)


class WebContentSource(ContentSource):
    """Content source for web URLs."""

    def __init__(self, config: Dict[str, Any]):
        """Initialize the web content source."""
        super().__init__(config)
        self.base_url = config.get("base_url")
        self.url_list = config.get("url_list", [])
        self.url_list_file = config.get("url_list_file")
        self.refresh_interval = config.get("refresh_interval", 86400)  # Default: 1 day
        self.headers = config.get("headers", {})
        self.include_patterns = config.get("include_patterns", [])
        self.exclude_patterns = config.get("exclude_patterns", [])

        # Load URLs from file if specified
        if self.url_list_file and os.path.exists(self.url_list_file):
            with open(self.url_list_file, 'r') as f:
                self.url_list.extend([line.strip() for line in f if line.strip()])

        # Initialize session for requests
        self.session = requests.Session()
        if config.get("authentication"):
            auth_type = config["authentication"].get("type")
            if auth_type == "basic":
                self.session.auth = (
                    config["authentication"].get("username", ""),
                    config["authentication"].get("password", "")
                )
            elif auth_type == "bearer":
                self.session.headers.update({
                    "Authorization": f"Bearer {config['authentication'].get('token', '')}"
                })

        # Add custom headers
        self.session.headers.update(self.headers)
        
        # Initialize content cache
        self.content_cache = {}

    def fetch_document(self, source_id: str) -> Dict[str, Any]:
        """Fetch document content from web URL."""
        # For web URLs, source_id is already expected to be a full URL
        url = source_id

        # Add base URL only if it's a relative URL and base_url is set
        if self.base_url and not url.startswith(('http://', 'https://')):
            url = urljoin(self.base_url, url)

        # Check cache first
        if url in self.content_cache:
            logger.debug(f"Returning cached content for URL: {url}")
            return self.content_cache[url]

        try:
            logger.debug(f"Fetching URL: {url}")
            response = self.session.get(url)
            response.raise_for_status()

            content = response.text
            content_type = response.headers.get('Content-Type', '')
            logger.debug(f"Successfully fetched URL: {url} (size: {len(content)} bytes)")

            result = {
                "id": url,  # URL is already a fully qualified path
                "content": content,
                "metadata": {
                    "url": url,
                    "content_type": content_type,
                    "last_modified": self._get_last_modified(response),
                    "status_code": response.status_code,
                    "headers": dict(response.headers)
                },
                "content_hash": self.get_content_hash(content)
            }
            
            # Cache the result
            self.content_cache[url] = result
            
            return result
        except Exception as e:
            logger.error(f"Error fetching URL {url}: {str(e)}")
            raise

    def list_documents(self) -> List[Dict[str, Any]]:
        """List available web URLs."""
        results = []
        logger.debug(f"Listing {len(self.url_list)} URLs from source configuration")

        for url in self.url_list:
            # Add base URL if relative and base_url is set
            if self.base_url and not url.startswith(('http://', 'https://')):
                full_url = urljoin(self.base_url, url)
            else:
                full_url = url

            results.append({
                "id": full_url,  # Use full URL as ID
                "metadata": {"url": full_url}
            })

        return results

    def has_changed(self, source_id: str, last_modified: Optional[float] = None) -> bool:
        """Check if web content has changed using HTTP headers."""
        # source_id should be a full URL
        url = source_id
        logger.debug(f"Checking if URL has changed: {url}")

        headers = {}

        # Add If-Modified-Since header if we have a last_modified timestamp
        if last_modified is not None:
            from email.utils import formatdate
            headers['If-Modified-Since'] = formatdate(last_modified, localtime=False, usegmt=True)
            logger.debug(f"Using If-Modified-Since: {headers['If-Modified-Since']}")

        try:
            # Make HEAD request to check if modified
            logger.debug(f"Making HEAD request to: {url}")
            response = self.session.head(url, headers=headers)
            logger.debug(f"HEAD response status: {response.status_code}")

            # If we get a 304 Not Modified, the content hasn't changed
            if response.status_code == 304:
                logger.debug(f"URL not modified: {url}")
                return False

            # If If-Modified-Since was set, and we got 200 OK, content has changed
            if 'If-Modified-Since' in headers and response.status_code == 200:
                logger.debug(f"URL modified since last check: {url}")
                return True

            # Otherwise, we need to compare the Last-Modified header
            current_last_modified = self._get_last_modified(response)
            logger.debug(f"Current Last-Modified: {current_last_modified}, Previous: {last_modified}")

            if last_modified is None or current_last_modified is None:
                logger.debug(f"Cannot determine if changed due to missing timestamp: {url}")
                return True

            changed = current_last_modified > last_modified
            logger.debug(f"URL changed: {changed} - {url}")
            return changed
        except Exception as e:
            logger.error(f"Error checking changes for URL {url}: {str(e)}")
            return True

    def follow_links(self, content: str, source_id: str, current_depth: int = 0, global_visited_docs=None) -> List[
        Dict[str, Any]]:
        """Extract and queue links for asynchronous processing by distributed workers."""
        if current_depth >= self.max_link_depth:
            logger.debug(f"Max link depth {self.max_link_depth} reached for {source_id}")
            return []

        # Parse HTML content
        logger.debug(f"Parsing content from {source_id} to extract links (depth: {current_depth})")
        soup = BeautifulSoup(content, 'html.parser')

        # source_id should already be a full URL
        base_url = source_id
        base_domain = urlparse(base_url).netloc
        logger.debug(f"Base domain: {base_domain}")

        # Extract links
        links = set()
        for a_tag in soup.find_all('a', href=True):
            href = a_tag['href']

            # Resolve relative URLs
            absolute_url = urljoin(base_url, href)

            # Skip external links (different domain)
            if urlparse(absolute_url).netloc != base_domain:
                continue

            # Skip URL fragments (same page anchors)
            if absolute_url == base_url or absolute_url.startswith(f"{base_url}#"):
                continue

            # Check include/exclude patterns
            if self._should_include_url(absolute_url):
                links.add(absolute_url)

        logger.debug(f"Found {len(links)} unique links to extract from {source_id}")

        # Queue links for asynchronous processing instead of processing synchronously
        queued_count = 0
        for link in links:
            if self.queue_discovered_link(link, source_id, current_depth):
                queued_count += 1
                logger.debug(f"Queued link for processing: {link}")
            else:
                logger.debug(f"Skipped queuing link (depth limit or no job control): {link}")

        logger.info(f"Queued {queued_count} links from {source_id} for distributed processing")

        # Return empty list since we're using asynchronous processing
        # Links will be processed by any available worker in the distributed system
        return []

    @staticmethod
    def _get_last_modified(response) -> Optional[float]:
        """Extract Last-Modified timestamp from response headers."""
        last_modified = response.headers.get('Last-Modified')
        if not last_modified:
            return None

        # Parse Last-Modified header to timestamp
        from email.utils import parsedate_to_datetime
        try:
            dt = parsedate_to_datetime(last_modified)
            return dt.timestamp()
        except Exception as e:
            logger.error(f"Error parsing Last-Modified header: {str(e)}")
            return None

    def _should_include_url(self, url: str) -> bool:
        """Check if URL should be included based on patterns."""
        # Check exclude patterns first
        for pattern in self.exclude_patterns:
            if re.search(pattern, url):
                logger.debug(f"URL excluded by pattern {pattern}: {url}")
                return False

        # If no include patterns, include all
        if not self.include_patterns:
            return True

        # Check include patterns
        for pattern in self.include_patterns:
            if re.search(pattern, url):
                logger.debug(f"URL included by pattern {pattern}: {url}")
                return True

        # Default to exclude if no include pattern matched
        logger.debug(f"URL excluded (no matching include pattern): {url}")
        return False

    def supports_continuous_discovery(self) -> bool:
        """Web sources support continuous discovery for crawling new links."""
        return True

    def discover_new_documents(self, last_discovery_time: Optional[float] = None) -> List[Dict[str, Any]]:
        """
        Discover new documents by re-crawling and finding new links.

        Args:
            last_discovery_time: Unix timestamp of last discovery

        Returns:
            List of newly discovered documents
        """
        import time
        current_time = time.time()

        if last_discovery_time and (current_time - last_discovery_time) < self.refresh_interval:
            logger.debug(f"Skipping discovery - refresh interval not yet reached")
            return []

        logger.info(f"Web source {self.name} discovering new documents")

        # For web sources, we need to re-crawl existing URLs to find new links
        # This is a simplified implementation - more sophisticated crawlers would
        # maintain state about previously seen links
        new_documents = []

        try:
            # Re-crawl base URLs to find new links
            for base_url in self.url_list:
                try:
                    response = self.session.get(base_url, timeout=30)
                    if response.status_code == 200:
                        # Use link following to find new documents
                        linked_docs = self.follow_links(
                            response.text,
                            base_url,
                            current_depth=0
                        )
                        new_documents.extend(linked_docs)
                        logger.debug(f"Found {len(linked_docs)} linked documents from {base_url}")

                except Exception as e:
                    logger.warning(f"Failed to crawl {base_url} for new links: {e}")

            logger.info(f"Web discovery found {len(new_documents)} potential new documents")
            return new_documents

        except Exception as e:
            logger.error(f"Error during web discovery: {e}")
            return []
