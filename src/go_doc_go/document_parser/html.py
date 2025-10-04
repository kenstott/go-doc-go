"""
HTML document parser module with caching strategies and date extraction for the document pointer system.

This module parses HTML documents into structured elements with improved performance
and comprehensive date extraction and temporal analysis.
"""

import hashlib
import json
import logging
import os
from typing import Dict, Any, Optional, List, Union, Tuple

from bs4 import BeautifulSoup

from .base import DocumentParser
from .extract_dates import DateExtractor
from .lru_cache import LRUCache, ttl_cache
from ..relationships import RelationshipType
from ..storage import ElementType

logger = logging.getLogger(__name__)


class HtmlParser(DocumentParser):
    """Parser for HTML documents with caching and comprehensive date extraction."""

    def __init__(self, config: Optional[Dict[str, Any]] = None):
        """Initialize the HTML parser with caching capabilities and date extraction."""
        super().__init__(config)
        # Define HTML-specific link patterns
        self.link_patterns = [
            r'<a\s+href=["\']([^"\']+)["\'][^>]*>(.*?)</a>'  # HTML links
        ]
        self.max_content_preview = self.config.get("max_content_preview", 100)

        # Cache configurations
        self.cache_ttl = self.config.get("cache_ttl", 3600)  # Default 1 hour TTL
        self.max_cache_size = self.config.get("max_cache_size", 128)  # Default max cache size
        self.enable_caching = self.config.get("enable_caching", True)

        # Date extraction configuration
        self.extract_dates = self.config.get("extract_dates", True)
        self.date_context_chars = self.config.get("date_context_chars", 50)  # Small context window
        self.min_year = self.config.get("min_year", 1900)
        self.max_year = self.config.get("max_year", 2100)
        self.fiscal_year_start_month = self.config.get("fiscal_year_start_month", 10)
        self.default_locale = self.config.get("default_locale", "US")

        # Initialize date extractor if enabled
        self.date_extractor = None
        if self.extract_dates:
            try:
                self.date_extractor = DateExtractor(
                    context_chars=self.date_context_chars,
                    min_year=self.min_year,
                    max_year=self.max_year,
                    fiscal_year_start_month=self.fiscal_year_start_month,
                    default_locale=self.default_locale
                )
                # logger.debug("Date extraction enabled with comprehensive temporal analysis")
            except ImportError as e:
                logger.warning(f"Date extraction disabled: {e}")
                self.extract_dates = False

        # Initialize caches
        if self.enable_caching:
            self.document_cache = LRUCache(max_size=self.max_cache_size, ttl=self.cache_ttl)
            self.soup_cache = LRUCache(max_size=self.max_cache_size, ttl=self.cache_ttl)
            self.content_cache = LRUCache(max_size=self.max_cache_size * 2, ttl=self.cache_ttl)
            self.selector_cache = LRUCache(max_size=self.max_cache_size * 2, ttl=self.cache_ttl)

    def _extract_dates_from_text(self, text: str, element_id: str, element_dates: Dict[str, List[Dict[str, Any]]]):
        """
        Extract dates from text content and add to element_dates.

        Args:
            text: Text content to extract dates from
            element_id: ID of the element containing the text
            element_dates: Dictionary to store extracted dates
        """
        if not self.extract_dates or not self.date_extractor or not text.strip():
            return

        try:
            dates = self.date_extractor.extract_dates_as_dicts(text)
            if dates:
                element_dates[element_id] = dates
                logger.debug(f"Extracted {len(dates)} dates from element {element_id}")
        except Exception as e:
            logger.warning(f"Error extracting dates from element {element_id}: {e}")

    def clear_caches(self):
        """Clear all caches."""
        if self.enable_caching:
            self.document_cache.clear()
            self.soup_cache.clear()
            self.content_cache.clear()
            self.selector_cache.clear()
            logger.info("All HTML parser caches cleared")

    def _load_source_content(self, source_path: str) -> Tuple[Union[str, bytes], Optional[str]]:
        """
        Load content from a source file with proper error handling and caching.

        Args:
            source_path: Path to the source file (may include ::timestamp suffix)

        Returns:
            Tuple of (content, error_message)
            - content: The file content as string
            - error_message: Error message if loading failed, None otherwise
        """
        # Strip timestamp suffix if present (format: path::timestamp)
        actual_path = source_path.split('::')[0] if '::' in source_path else source_path

        # Check cache first if enabled
        if self.enable_caching:
            content = self.content_cache.get(source_path)
            if content is not None:
                logger.debug(f"Content cache hit for {source_path}")
                return content, None

        if not os.path.exists(actual_path):
            error_msg = f"Error: Source file not found: {source_path}"
            logger.error(error_msg)
            return None, error_msg

        try:
            # Try different encodings for HTML files
            for encoding in ['utf-8', 'latin1', 'cp1252']:
                try:
                    with open(actual_path, 'r', encoding=encoding) as f:
                        content = f.read()

                    # Cache the content if enabled
                    if self.enable_caching:
                        self.content_cache.set(source_path, content)

                    return content, None
                except UnicodeDecodeError:
                    if encoding == 'cp1252':  # Last attempt
                        raise
                    continue
        except Exception as e:
            error_msg = f"Error: Cannot read content from {source_path}: {str(e)}"
            logger.error(error_msg)
            return None, error_msg

    def _get_or_create_soup(self, content: str) -> BeautifulSoup:
        """
        Get a cached BeautifulSoup object or create one if not cached.

        Args:
            content: HTML content

        Returns:
            BeautifulSoup object
        """
        if not self.enable_caching:
            return BeautifulSoup(content, 'html.parser')

        # Create hash of content for cache key
        content_hash = hashlib.md5(content.encode('utf-8')).hexdigest()

        # Check cache
        soup = self.soup_cache.get(content_hash)
        if soup is not None:
            logger.debug(f"Soup cache hit for content hash {content_hash[:8]}")
            return soup

        # Create new soup object
        soup = BeautifulSoup(content, 'html.parser')

        # Cache it
        self.soup_cache.set(content_hash, soup)

        return soup

    @ttl_cache(maxsize=256, ttl=3600)
    def _get_element_type(self, tag_name: str) -> str:
        """
        Map HTML tag names to element types with caching.

        Args:
            tag_name: HTML tag name

        Returns:
            Element type string
        """
        if tag_name in ['h1', 'h2', 'h3', 'h4', 'h5', 'h6']:
            return ElementType.HEADER.value
        elif tag_name == 'p':
            return ElementType.PARAGRAPH.value
        elif tag_name in ['ul', 'ol']:
            return ElementType.LIST.value
        elif tag_name == 'li':
            return ElementType.LIST_ITEM.value
        elif tag_name == 'table':
            return ElementType.TABLE.value
        elif tag_name == 'tr':
            return ElementType.TABLE_ROW.value
        elif tag_name == 'th':
            return ElementType.TABLE_HEADER.value
        elif tag_name == 'td':
            return ElementType.TABLE_CELL.value
        elif tag_name == 'img':
            return ElementType.IMAGE.value
        elif tag_name in ['pre', 'code']:
            return ElementType.CODE_BLOCK.value
        elif tag_name == 'blockquote':
            return ElementType.BLOCKQUOTE.value
        else:
            # Check for namespace-prefixed elements (iXBRL, XBRL, etc.)
            if ':' in tag_name:
                namespace, local_name = tag_name.split(':', 1)
                # Create element type that preserves namespace info
                # e.g., "ix:nonnumeric" becomes "ix_nonnumeric"
                return f"{namespace}_{local_name}"
            # For container elements
            return tag_name  # Use the tag name as the element type (div, article, etc.)

    def _add_selectors(self, element, parent_selector=""):
        """
        Add CSS selectors to elements for location with caching for complex paths.
        Generates unique selectors using nth-of-type positioning when needed.

        Args:
            element: BeautifulSoup element
            parent_selector: Parent's selector
        """
        if not hasattr(element, 'name') or not element.name:
            return

        # Use the cached selector if available
        element_id = id(element)
        cache_key = f"{element_id}_{parent_selector}"

        if self.enable_caching:
            cached_selector = self.selector_cache.get(cache_key)
            if cached_selector is not None:
                element['_selector'] = cached_selector
                return

        # Build selector for this element
        if element.name == 'body':
            selector = 'body'
        else:
            tag_selector = element.name

            # Priority 1: Use ID if present (most specific)
            if element.get('id'):
                id_selector = f"#{element.get('id')}"
                tag_selector = f"{tag_selector}{id_selector}"

            # Priority 2: Use unique combination of classes
            elif element.get('class'):
                # Use all classes for better specificity
                classes = element.get('class')
                if isinstance(classes, list):
                    class_selector = ''.join([f".{cls}" for cls in classes[:3]])  # Limit to 3 classes
                else:
                    class_selector = f".{classes}"
                tag_selector = f"{tag_selector}{class_selector}"

            # Priority 3: Use data attributes if available
            elif element.attrs:
                # Look for data attributes or other unique attributes
                unique_attrs = []
                for attr, value in element.attrs.items():
                    if attr.startswith('data-') or attr in ['aria-label', 'name', 'type']:
                        if isinstance(value, str) and value:
                            # Escape special characters in attribute values
                            escaped_value = value.replace('"', '\\"')
                            unique_attrs.append(f'[{attr}="{escaped_value}"]')
                            if len(unique_attrs) >= 2:  # Limit to 2 attributes
                                break

                if unique_attrs:
                    tag_selector = f"{tag_selector}{''.join(unique_attrs)}"

            # Always add nth-of-type if there are siblings with the same tag
            # (unless we already have an ID which makes it unique)
            if not element.get('id'):
                # Count preceding siblings with the same tag name
                position = 1
                for sibling in element.find_previous_siblings(element.name):
                    position += 1

                # Also check if there are following siblings
                has_following_siblings = bool(element.find_next_siblings(element.name))

                # Add nth-of-type if not the only child of this type
                if position > 1 or has_following_siblings:
                    tag_selector = f"{tag_selector}:nth-of-type({position})"

            # Combine with parent selector
            if parent_selector:
                selector = f"{parent_selector} > {tag_selector}"
            else:
                selector = tag_selector

        # Store selector on element
        element['_selector'] = selector

        # Cache the selector if enabled
        if self.enable_caching:
            self.selector_cache.set(cache_key, selector)

        # Process children
        for child in element.children:
            if hasattr(child, 'name') and child.name:
                self._add_selectors(child, selector)

    def _resolve_element_text(self, location_data: Dict[str, Any], source_content: Optional[Union[str, bytes]] = None) -> str:
        """
        Resolve the plain text representation of an HTML element with caching.

        Args:
            location_data: Content location data
            source_content: Optional preloaded source content

        Returns:
            Plain text representation of the element
        """
        # Generate cache key based on location data
        if self.enable_caching:
            cache_key = f"text_{hash(json.dumps(location_data, sort_keys=True))}"
            cached_text = self.content_cache.get(cache_key)
            if cached_text is not None:
                logger.debug(f"Text cache hit for {cache_key}")
                return cached_text

        # First, get the HTML content of the element
        html_content = self._resolve_element_content(location_data, source_content)
        if not html_content:
            return ""

        # Parse the HTML fragment
        soup = BeautifulSoup(html_content, 'html.parser')

        element_type = location_data.get("type", "")

        # Handle specific element types
        if element_type == "header":
            # For headers, just return the text
            result = soup.get_text(separator=' ', strip=True)

        elif element_type == "paragraph":
            # For paragraphs, return the text
            result = soup.get_text(separator=' ', strip=True)

        elif element_type == "list":
            # For lists, format each item on a new line with a bullet or number
            list_type = location_data.get("list_type", "unordered")
            items = soup.find_all('li')
            formatted_items = []

            for i, item in enumerate(items):
                if list_type == "ordered":
                    formatted_items.append(f"{i + 1}. {item.get_text(separator=' ', strip=True)}")
                else:
                    formatted_items.append(f"• {item.get_text(separator=' ', strip=True)}")

            result = "\n".join(formatted_items)

        elif element_type == "list_item":
            # For a single list item, return the text
            result = soup.get_text(separator=' ', strip=True)

        elif element_type in ["table", "table_row", "table_cell", "table_header"]:
            if element_type == "table":
                # For a complete table, return a structured representation
                rows = soup.find_all('tr')
                formatted_rows = []

                # Check if table has headers
                headers = soup.find_all('th')
                if headers:
                    header_texts = [h.get_text(separator=' ', strip=True) for h in headers]
                    formatted_rows.append(" | ".join(header_texts))
                    formatted_rows.append("-" * (sum(len(h) + 3 for h in header_texts)))

                # Process rows
                for row in rows:
                    # Skip header row if we already processed headers
                    if row.find('th') and headers:
                        continue

                    cells = row.find_all(['td', 'th'])
                    row_text = " | ".join(cell.get_text(separator=' ', strip=True) for cell in cells)
                    if row_text.strip():  # Skip empty rows
                        formatted_rows.append(row_text)

                result = "\n".join(formatted_rows)

            elif element_type == "table_row":
                # For a table row, return cells separated by |
                cells = soup.find_all(['td', 'th'])
                result = " | ".join(cell.get_text(separator=' ', strip=True) for cell in cells)

            elif element_type in ["table_cell", "table_header"]:
                # For a single cell, just return the text
                result = soup.get_text(separator=' ', strip=True)

        elif element_type == "image":
            # For an image, return the alt text or a description
            img = soup.find('img')
            if img and img.get('alt'):
                result = f"Image: {img.get('alt')}"
            elif img and img.get('src'):
                result = f"Image: {img.get('src').split('/')[-1]}"
            else:
                result = "Image"

        elif element_type == "code_block":
            # For code blocks, preserve formatting but remove the tags
            code = soup.find('code') or soup
            text = code.get_text()
            language = location_data.get("language", "")
            if language:
                result = f"Code ({language}):\n{text}"
            else:
                result = f"Code:\n{text}"

        elif element_type == "blockquote":
            # For blockquotes, prefix each line with >
            lines = soup.get_text(separator=' ', strip=True).split('\n')
            result = '\n'.join(f"> {line}" for line in lines)

        else:
            # Default case: return only direct text content, not all descendants
            # This prevents container elements from including all child text
            # (similar to the fix we applied for XML elements)
            direct_text = []
            for child in soup.children:
                if isinstance(child, str):
                    text = child.strip()
                    if text:
                        direct_text.append(text)

            # For namespace elements, also include key attributes
            if soup.name and ':' in soup.name:
                # Include important attributes for iXBRL/XBRL elements
                important_attrs = []
                for attr, value in (soup.attrs or {}).items():
                    # Skip style and class attributes, but include data attributes
                    if attr not in ['style', 'class'] and value and len(str(value)) < 100:
                        important_attrs.append(f"{attr}={value}")

                result = ' '.join(direct_text) if direct_text else ""
                if important_attrs:
                    result = result + " [" + ", ".join(important_attrs) + "]" if result else "[" + ", ".join(important_attrs) + "]"
            else:
                result = ' '.join(direct_text) if direct_text else ""

        # Cache the result if enabled
        if self.enable_caching:
            self.content_cache.set(cache_key, result)

        return result

    def _select_with_xml_namespaces(self, soup: BeautifulSoup, selector: str) -> List:
        """
        Select elements using a CSS selector that may contain XML namespace prefixes.

        Args:
            soup: BeautifulSoup object
            selector: CSS selector potentially containing XML namespaces (ix:nonnumeric, xbrl:*, etc.)

        Returns:
            List of matching BeautifulSoup elements
        """
        try:
            # First try standard CSS selector
            return soup.select(selector)
        except Exception as e:
            # If standard selector fails with namespace error, try XML-aware approach
            if "pseudo-class" in str(e) and ":" in selector:
                # Only log at debug level if we actually need to troubleshoot
                # This is expected behavior for XML/XBRL documents
                logger.debug(f"Using XML namespace fallback for selector: {selector}")
                return self._select_xml_fallback(soup, selector)
            else:
                # Re-raise if it's not a namespace issue
                raise

    def _select_xml_fallback(self, soup: BeautifulSoup, selector: str) -> List:
        """
        Fallback selector for XML namespaced elements.

        Args:
            soup: BeautifulSoup object
            selector: Original CSS selector with XML namespaces

        Returns:
            List of matching BeautifulSoup elements
        """
        # Parse the selector and navigate step by step
        # Example: "body > div > ix:nonnumeric#F_abc > div > custom:element#F_def > table > tr"
        parts = [part.strip() for part in selector.split('>')]
        current_elements = [soup]

        for part in parts:
            if not current_elements:
                break

            next_elements = []

            for element in current_elements:
                # Handle any XML namespace elements
                if ':' in part:
                    # Extract namespace and tag name with attributes
                    if '#' in part:
                        # Handle namespace with ID (e.g., "ix:nonnumeric#F_abc")
                        namespace_tag, id_value = part.split('#', 1)
                        namespace, tag_name = namespace_tag.split(':', 1)

                        # Find elements with this namespace:tag combination and matching ID
                        matches = element.find_all(lambda tag:
                            tag.name and ':' in tag.name and
                            tag.name.split(':')[0] == namespace and
                            tag.name.split(':')[-1] == tag_name and
                            tag.get('id') == id_value
                        )
                    elif '.' in part:
                        # Handle namespace with class (e.g., "ix:nonnumeric.classname")
                        namespace_tag, class_name = part.split('.', 1)
                        namespace, tag_name = namespace_tag.split(':', 1)

                        matches = element.find_all(lambda tag:
                            tag.name and ':' in tag.name and
                            tag.name.split(':')[0] == namespace and
                            tag.name.split(':')[-1] == tag_name,
                            class_=class_name
                        )
                    else:
                        # Just namespace:tag (e.g., "ix:nonnumeric")
                        namespace, tag_name = part.split(':', 1)
                        matches = element.find_all(lambda tag:
                            tag.name and ':' in tag.name and
                            tag.name.split(':')[0] == namespace and
                            tag.name.split(':')[-1] == tag_name
                        )
                else:
                    # Standard HTML element
                    if '#' in part:
                        tag_name, id_value = part.split('#', 1)
                        matches = element.find_all(tag_name, id=id_value)
                    elif '.' in part:
                        tag_name, class_name = part.split('.', 1)
                        matches = element.find_all(tag_name, class_=class_name)
                    else:
                        matches = element.find_all(part)

                # Add direct children only (respecting > combinator)
                for match in matches:
                    if element == soup:
                        # For root level searches, add all matches
                        next_elements.append(match)
                    elif match.parent == element:
                        # For nested searches, only add direct children
                        next_elements.append(match)

            current_elements = next_elements

        return current_elements

    def _resolve_element_content(self, location_data: Dict[str, Any],
                                 source_content: Optional[Union[str, bytes]]) -> str:
        """
        Resolve content for specific HTML element types with caching.

        Args:
            location_data: Content location data
            source_content: Optional preloaded source content

        Returns:
            Resolved content string (HTML format)
        """
        # Generate cache key based on location data
        if self.enable_caching:
            cache_key = f"content_{hash(json.dumps(location_data, sort_keys=True))}"
            cached_content = self.content_cache.get(cache_key)
            if cached_content is not None:
                logger.debug(f"Content cache hit for {cache_key}")
                return cached_content

        source = location_data.get("source", "")
        element_type = location_data.get("type", "")
        selector = location_data.get("selector", "")

        # Load content if not provided
        content = source_content
        if content is None:
            content, error = self._load_source_content(source)
            if error:
                logger.error(f"Error loading source content: {error}")
                return ""

        # Parse HTML with BeautifulSoup
        soup = self._get_or_create_soup(content)

        # Initialize result
        result = ""

        # If a CSS selector is provided, use it
        if selector:
            elements = self._select_with_xml_namespaces(soup, selector)
            if elements:
                # Should always return 1 item
                result = str(elements[0])

        # Handle element type-specific content if no result yet
        elif element_type == "header":
            # Extract header by level and/or text
            header_level = location_data.get("level")
            header_text = location_data.get("text", "")

            # Find header by level and text
            if header_level:
                headers = soup.find_all(f'h{header_level}')

                if header_text:
                    # Find header with matching text
                    for header in headers:
                        if header_text in header.get_text():
                            result = str(header)
                            break

                # If no text match, but we have headers at this level, return the first one
                if not result and headers:
                    result = str(headers[0])

            # If no level specified, search all header levels
            if not result and header_text:
                for level in range(1, 7):
                    headers = soup.find_all(f'h{level}')
                    for header in headers:
                        if header_text in header.get_text():
                            result = str(header)
                            break
                    if result:
                        break

        elif element_type == "paragraph":
            # Extract paragraph by text or index
            para_text = location_data.get("text", "")
            para_index = location_data.get("index", 0)

            paragraphs = soup.find_all('p')

            if para_text:
                # Find paragraph with matching text
                for para in paragraphs:
                    if para_text in para.get_text():
                        result = str(para)
                        break

            # Return paragraph by index if no match by text
            if not result and 0 <= para_index < len(paragraphs):
                result = str(paragraphs[para_index])

        elif element_type == "list" or element_type == "list_item":
            # Extract list or list item
            list_type = location_data.get("list_type", "")
            list_tag = 'ol' if list_type == 'ordered' else 'ul'
            index = location_data.get("index", 0)

            lists = soup.find_all(list_tag)

            if lists:
                if element_type == "list":
                    # Return the whole list
                    result = str(lists[0])
                else:
                    # Return specific list item
                    items = lists[0].find_all('li')
                    if 0 <= index < len(items):
                        result = str(items[index])

        elif element_type in ["table", "table_row", "table_cell", "table_header"]:
            # Extract table element
            table_index = location_data.get("table_index", 0)
            row = location_data.get("row", 0)
            col = location_data.get("col", 0)

            tables = soup.find_all('table')

            if tables and table_index < len(tables):
                table = tables[table_index]

                if element_type == "table":
                    # Return the whole table
                    result = str(table)
                else:
                    # Get rows
                    rows = table.find_all('tr')
                    if row < len(rows):
                        if element_type == "table_row":
                            # Return the whole row
                            result = str(rows[row])
                        else:
                            # Get cells
                            cells = rows[row].find_all(['td', 'th'])
                            if col < len(cells):
                                # Return specific cell
                                result = str(cells[col])

        elif element_type == "image":
            # Extract image information
            src = location_data.get("src", "")

            images = soup.find_all('img')
            for img in images:
                if src and src == img.get('src'):
                    # Return the img tag as string
                    result = str(img)
                    break

        elif element_type == "code_block":
            # Extract code block
            language = location_data.get("language", "")

            code_blocks = soup.find_all('pre')
            for block in code_blocks:
                code_tag = block.find('code')
                if code_tag and language:
                    # Check for language in class
                    classes = code_tag.get('class', [])
                    for cls in classes:
                        if cls.startswith('language-') and cls[9:] == language:
                            result = str(code_tag)
                            break
                elif code_tag:
                    # Return first code block if no language specified
                    result = str(code_tag)
                    break
                else:
                    # Return pre tag content if no code tag inside
                    result = str(block)
                    break

        elif element_type == "blockquote":
            # Extract blockquote
            blockquotes = soup.find_all('blockquote')
            if blockquotes:
                result = str(blockquotes[0])

        # For other element types or if no result, return full document HTML
        if not result:
            result = str(soup)

        # Cache the result if enabled
        if self.enable_caching:
            self.content_cache.set(cache_key, result)

        return result

    def parse(self, doc_content: Dict[str, Any]) -> Dict[str, Any]:
        """Parse an HTML document into structured elements with caching and comprehensive date extraction."""
        content = doc_content["content"]
        source_id = doc_content["id"]  # Should already be a fully qualified path
        metadata = doc_content.get("metadata", {}).copy()  # Make a copy to avoid modifying original

        # Generate document cache key for caching
        if self.enable_caching:
            content_hash = hashlib.md5(content.encode('utf-8')).hexdigest()
            doc_cache_key = f"{source_id}_{content_hash}"

            # Check document cache
            cached_doc = self.document_cache.get(doc_cache_key)
            if cached_doc is not None:
                logger.info(f"Document cache hit for {source_id}")
                return cached_doc

        # Generate document ID if not present
        doc_id = doc_content.get("id") or metadata.get("doc_id") or self._generate_id("doc_")

        # Create document record
        document = {
            "doc_id": doc_id,
            "doc_type": "html",
            "source": source_id,
            "metadata": metadata,
            "content_hash": doc_content.get("content_hash", self._generate_hash(content))
        }

        # Create root element
        elements: List = [self._create_root_element(doc_id, source_id)]
        root_id = elements[0]["element_id"]

        # Initialize relationships list and element_dates dictionary
        relationships = []
        element_dates = {}

        # Extract dates from full document content first
        if self.extract_dates and self.date_extractor:
            try:
                # Get plain text from HTML for date extraction
                soup = self._get_or_create_soup(content)
                # Use separator to add spaces between elements
                document_text = soup.get_text(separator=' ', strip=True)
                document_dates = self.date_extractor.extract_dates_as_dicts(document_text)
                if document_dates:
                    element_dates[root_id] = document_dates
                    logger.debug(f"Extracted {len(document_dates)} dates from document")
            except Exception as e:
                logger.warning(f"Error during document date extraction: {e}")

        # Parse HTML with caching
        soup = self._get_or_create_soup(content)

        # Add CSS selectors to elements for better location tracking
        self._add_selectors(soup)

        # Extract links directly from HTML
        extracted_links = []
        for a in soup.find_all('a', href=True):
            extracted_links.append({
                "source_id": root_id,  # Initially assign to root, will update later
                "link_text": a.get_text(separator=' ', strip=True),
                "link_target": a['href'],
                "link_type": "html"
            })

        # Parse HTML elements with relationships and date extraction
        parsed_elements, element_links, element_relationships = self._parse_document(soup, doc_id, root_id, source_id, element_dates)
        elements.extend(parsed_elements)
        relationships.extend(element_relationships)

        # Update link source_ids with the correct element IDs
        self._update_link_sources(extracted_links, parsed_elements)
        extracted_links.extend(element_links)

        # Add date statistics to document metadata
        if element_dates:
            total_dates = sum(len(dates) for dates in element_dates.values())
            document["metadata"]["date_extraction"] = {
                "total_dates_found": total_dates,
                "elements_with_dates": len(element_dates),
                "extraction_enabled": True
            }
        else:
            document["metadata"]["date_extraction"] = {
                "total_dates_found": 0,
                "elements_with_dates": 0,
                "extraction_enabled": self.extract_dates
            }

        # Create the final result
        result = {
            "document": document,
            "elements": elements,
            "links": extracted_links,
            "relationships": relationships
        }

        # Add dates if any were extracted
        if element_dates:
            result["element_dates"] = element_dates

        # Cache the result if enabled
        if self.enable_caching:
            self.document_cache.set(doc_cache_key, result)

        return result

    def _parse_document(self, soup, doc_id, parent_id, source_id, element_dates):
        """Parse the entire document in a unified way and create relationships."""
        elements = []
        links = []
        relationships = []

        # Create a map to track element IDs by tag reference
        element_id_map = {}

        # Start with the body if it exists
        if soup.body:
            # Process the body element first
            body_element = self._create_element_for_tag(soup.body, doc_id, parent_id, source_id, element_dates)
            if body_element:
                elements.append(body_element)
                element_id_map[soup.body] = body_element["element_id"]
                body_id = body_element["element_id"]

                # Create relationship from root to body
                contains_relationship = {
                    "relationship_id": self._generate_id("rel_"),
                    "source_id": parent_id,
                    "target_id": body_id,
                    "relationship_type": RelationshipType.CONTAINS.value,
                    "metadata": {
                        "confidence": 1.0
                    }
                }
                relationships.append(contains_relationship)

                # Create inverse relationship
                contained_by_relationship = {
                    "relationship_id": self._generate_id("rel_"),
                    "source_id": body_id,
                    "target_id": parent_id,
                    "relationship_type": RelationshipType.CONTAINED_BY.value,
                    "metadata": {
                        "confidence": 1.0
                    }
                }
                relationships.append(contained_by_relationship)
            else:
                body_id = parent_id

            # Use a breadth-first approach to process children
            child_elements, child_links, child_relationships = self._process_tag_children(
                soup.body, doc_id, body_id, source_id, element_id_map, element_dates)

            elements.extend(child_elements)
            links.extend(child_links)
            relationships.extend(child_relationships)

        return elements, links, relationships

    def _process_tag_children(self, parent_tag, doc_id, parent_id, source_id, element_id_map, element_dates):
        """Process all children of a tag and create relationships."""
        elements = []
        links = []
        relationships = []

        # Get direct children
        for child in parent_tag.children:
            # Skip text nodes and other non-element nodes
            if not hasattr(child, 'name') or not child.name:
                continue

            # Skip ix: namespace tags completely - we don't want to parse or store them
            # Also skip processing their children to avoid any ix: content
            if child.name.startswith('ix:'):
                continue

            # Create an element for this tag
            # Include standard HTML tags only (removed namespace-prefixed tag processing)
            # NOTE: Added 'li' and 'body' to capture full hierarchy
            if child.name in ['h1', 'h2', 'h3', 'h4', 'h5', 'h6', 'p', 'ul', 'ol', 'li',
                               'pre', 'code', 'blockquote', 'table', 'img', 'div',
                               'article', 'section', 'nav', 'aside', 'figure', 'body']:

                # Create an element
                element = self._create_element_for_tag(child, doc_id, parent_id, source_id, element_dates)

                if element:
                    elements.append(element)
                    element_id = element["element_id"]
                    element_id_map[child] = element_id

                    # Create relationship from parent to element
                    rel_type = RelationshipType.CONTAINS.value
                    if child.name == 'p':
                        rel_type = RelationshipType.CONTAINS_TEXT.value

                    contains_relationship = {
                        "relationship_id": self._generate_id("rel_"),
                        "source_id": parent_id,
                        "target_id": element_id,
                        "relationship_type": rel_type,
                        "metadata": {
                            "confidence": 1.0,
                            "tag": child.name
                        }
                    }
                    relationships.append(contains_relationship)

                    # Create inverse relationship
                    contained_by_relationship = {
                        "relationship_id": self._generate_id("rel_"),
                        "source_id": element_id,
                        "target_id": parent_id,
                        "relationship_type": RelationshipType.CONTAINED_BY.value,
                        "metadata": {
                            "confidence": 1.0
                        }
                    }
                    relationships.append(contained_by_relationship)

                    # Extract links from this element
                    for a in child.find_all('a', href=True):
                        links.append({
                            "source_id": element_id,
                            "link_text": a.get_text(separator=' ', strip=True),
                            "link_target": a['href'],
                            "link_type": "html"
                        })

                    # Special handling for specific element types
                    if child.name == 'table':
                        table_elements, table_links, table_relationships = self._process_table(
                            child, doc_id, element_id, source_id, element_dates)
                        elements.extend(table_elements)
                        links.extend(table_links)
                        relationships.extend(table_relationships)
                    # NOTE: Removed special _process_list() handling for ul/ol
                    # Lists and list items are now processed through normal whitelist processing
                    # This prevents duplicate list/list_item creation

                    # Process this tag's children recursively
                    child_elements, child_links, child_relationships = self._process_tag_children(
                        child, doc_id, element_id, source_id, element_id_map, element_dates)
                    elements.extend(child_elements)
                    links.extend(child_links)
                    relationships.extend(child_relationships)
                else:
                    # If no element was created, still process children with parent_id
                    child_elements, child_links, child_relationships = self._process_tag_children(
                        child, doc_id, parent_id, source_id, element_id_map, element_dates)
                    elements.extend(child_elements)
                    links.extend(child_links)
                    relationships.extend(child_relationships)
            else:
                # For non-content tags, just process their children with the same parent_id
                child_elements, child_links, child_relationships = self._process_tag_children(
                    child, doc_id, parent_id, source_id, element_id_map, element_dates)
                elements.extend(child_elements)
                links.extend(child_links)
                relationships.extend(child_relationships)

        return elements, links, relationships

    def _create_element_for_tag(self, tag, doc_id, parent_id, source_id, element_dates):
        """Create an appropriate element based on tag type."""
        element_type = self._get_element_type(tag.name)

        # Get only direct text content, not all descendants
        # This applies to ALL elements to prevent container pollution
        direct_text = []
        for child in tag.children:
            if isinstance(child, str):
                text = child.strip()
                if text:
                    direct_text.append(text)
        content_text = ' '.join(direct_text) if direct_text else ""

        # For specific element types that should include all text, use get_text
        if tag.name in ['p', 'h1', 'h2', 'h3', 'h4', 'h5', 'h6', 'li', 'th', 'td', 'blockquote', 'pre', 'code']:
            # These elements typically don't have complex nested structure
            # and we want all their text content
            content_text = tag.get_text(separator=' ', strip=True)

        # NOTE: We create elements even if they have no direct text content
        # Structural elements (ul, ol, section, nav, etc.) are important for hierarchy
        # The embedding system will skip empty ancestors when building context
        # Empty elements will have empty content_preview but still exist in the model

        element_id = self._generate_id(f"{element_type}_")

        # Extract dates from content text
        self._extract_dates_from_text(content_text, element_id, element_dates)

        # Create content preview
        if len(content_text) > self.max_content_preview:
            content_preview = content_text[:self.max_content_preview] + "..."
        else:
            content_preview = content_text

        # Create element with common fields
        element = {
            "element_id": element_id,
            "doc_id": doc_id,
            "element_type": element_type,
            "parent_id": parent_id,
            "content_preview": content_preview,
            "content_location": json.dumps({
                "source": source_id,
                "type": element_type,
                "selector": tag.get('_selector', '')
            }),
            "content_hash": self._generate_hash(str(tag)),
            "metadata": {
                "id": tag.get('id', ''),
                "class": tag.get('class', ''),
                "full_path": source_id
            }
        }

        # Add full content if configured to store it
        if self.config.get('store_full_content', False):
            element["content"] = content_text  # Store full untruncated content

        # Add element-specific metadata
        if tag.name in ['h1', 'h2', 'h3', 'h4', 'h5', 'h6']:
            element["metadata"]["level"] = int(tag.name[1])
            element["metadata"]["text"] = content_text
            element["content_location"] = json.dumps({
                "source": source_id,
                "type": element_type,
                "selector": tag.get('_selector', ''),
                "level": int(tag.name[1]),
                "text": content_text[:50] if len(content_text) > 50 else content_text
            })
        elif tag.name == 'img':
            alt_text = tag.get('alt', '')
            element["metadata"]["src"] = tag.get('src', '')
            element["metadata"]["alt"] = alt_text
            element["metadata"]["width"] = tag.get('width', '')
            element["metadata"]["height"] = tag.get('height', '')
            element["content_preview"] = alt_text or 'Image'
            element["content_location"] = json.dumps({
                "source": source_id,
                "type": element_type,
                "selector": tag.get('_selector', ''),
                "src": tag.get('src', '')
            })
            # Extract dates from alt text
            if alt_text:
                self._extract_dates_from_text(alt_text, element_id, element_dates)
        elif tag.name == 'pre' or tag.name == 'code':
            language = ""
            if tag.name == 'code' and tag.has_attr('class'):
                for cls in tag['class']:
                    if cls.startswith('language-'):
                        language = cls[9:]
                        break
            element["metadata"]["language"] = language
            element["content_location"] = json.dumps({
                "source": source_id,
                "type": element_type,
                "selector": tag.get('_selector', ''),
                "language": language
            })

        # Store the element ID on the tag for reference
        tag._element_id = element_id

        return element

    def _update_link_sources(self, links, elements):
        """Update link source IDs based on their position in the document."""
        # This would be a more sophisticated implementation that uses the
        # selectors or positions to determine which element contains each link
        # For now, we'll keep it simple and leave links assigned to the root
        pass

    def _process_list(self, tag, doc_id, parent_id, source_id, element_dates):
        """Process a list element and create relationships."""
        elements = []
        links = []
        relationships = []

        list_type = 'ordered' if tag.name == 'ol' else 'unordered'
        list_id = self._generate_id("list_")
        list_text = tag.get_text(separator=' ', strip=True)

        # Extract dates from list text
        self._extract_dates_from_text(list_text, list_id, element_dates)

        # Create meaningful content preview from actual list content
        if list_text and len(list_text.strip()) > 0:
            # Get first few list items for preview
            items = tag.find_all('li')
            if items:
                item_texts = []
                char_count = 0
                for item in items[:3]:  # Max 3 items for preview
                    item_text = item.get_text(separator=' ', strip=True)
                    if item_text and char_count + len(item_text) + 3 <= self.max_content_preview:  # +3 for ", "
                        item_texts.append(item_text)
                        char_count += len(item_text) + 3
                    else:
                        # Add truncated version if we have room
                        remaining = self.max_content_preview - char_count - 3
                        if remaining > 10 and item_text:
                            item_texts.append(item_text[:remaining] + "...")
                        break

                if item_texts:
                    content_preview = ", ".join(item_texts)
                    if len(items) > len(item_texts):
                        content_preview += f"... ({len(items)} items)"
                else:
                    content_preview = f"{list_type.capitalize()} list"
            else:
                content_preview = f"{list_type.capitalize()} list"
        else:
            content_preview = f"{list_type.capitalize()} list"

        list_element = {
            "element_id": list_id,
            "doc_id": doc_id,
            "element_type": "list",
            "parent_id": parent_id,
            "content_preview": content_preview,
            "content_location": json.dumps({
                "source": source_id,
                "type": "list",
                "list_type": list_type,
                "selector": tag.get('_selector', '')
            }),
            "content_hash": self._generate_hash(list_text),
            "metadata": {
                "list_type": list_type,
                "class": tag.get('class', ''),
                "full_path": source_id
            }
        }

        elements.append(list_element)
        tag._element_id = list_id

        # Create relationship from parent to list
        contains_relationship = {
            "relationship_id": self._generate_id("rel_"),
            "source_id": parent_id,
            "target_id": list_id,
            "relationship_type": RelationshipType.CONTAINS.value,
            "metadata": {
                "confidence": 1.0,
                "list_type": list_type
            }
        }
        relationships.append(contains_relationship)

        # Create inverse relationship
        contained_by_relationship = {
            "relationship_id": self._generate_id("rel_"),
            "source_id": list_id,
            "target_id": parent_id,
            "relationship_type": RelationshipType.CONTAINED_BY.value,
            "metadata": {
                "confidence": 1.0
            }
        }
        relationships.append(contained_by_relationship)

        # Process list items
        for i, item in enumerate(tag.find_all('li', recursive=False)):
            item_text = item.get_text(separator=' ', strip=True)
            if not item_text:
                continue

            item_id = self._generate_id("item_")

            # Extract dates from item text
            self._extract_dates_from_text(item_text, item_id, element_dates)

            item_element = {
                "element_id": item_id,
                "doc_id": doc_id,
                "element_type": "list_item",
                "parent_id": list_id,
                "content_preview": item_text[:self.max_content_preview] + (
                    "..." if len(item_text) > self.max_content_preview else ""),
                "content_location": json.dumps({
                    "source": source_id,
                    "type": "list_item",
                    "list_type": list_type,
                    "index": i,
                    "selector": item.get('_selector', '')
                }),
                "content_hash": self._generate_hash(item_text),
                "metadata": {
                    "index": i,
                    "full_path": source_id
                }
            }

            elements.append(item_element)
            item._element_id = item_id

            # Create relationship from list to item
            contains_item_relationship = {
                "relationship_id": self._generate_id("rel_"),
                "source_id": list_id,
                "target_id": item_id,
                "relationship_type": RelationshipType.CONTAINS_LIST_ITEM.value,
                "metadata": {
                    "confidence": 1.0,
                    "index": i
                }
            }
            relationships.append(contains_item_relationship)

            # Create inverse relationship
            item_contained_relationship = {
                "relationship_id": self._generate_id("rel_"),
                "source_id": item_id,
                "target_id": list_id,
                "relationship_type": RelationshipType.CONTAINED_BY.value,
                "metadata": {
                    "confidence": 1.0
                }
            }
            relationships.append(item_contained_relationship)

            # Extract links from list item
            for a in item.find_all('a', href=True):
                links.append({
                    "source_id": item_id,
                    "link_text": a.get_text(separator=' ', strip=True),
                    "link_target": a['href'],
                    "link_type": "html"
                })

        return elements, links, relationships

    def _process_table(self, tag, doc_id, parent_id, source_id, element_dates):
        """Process a table element and create relationships."""
        elements = []
        links = []
        relationships = []

        table_id = self._generate_id("table_")
        table_html = str(tag)
        table_text = tag.get_text(separator=' ', strip=True)

        # Extract dates from table text
        self._extract_dates_from_text(table_text, table_id, element_dates)

        table_element = {
            "element_id": table_id,
            "doc_id": doc_id,
            "element_type": "table",
            "parent_id": parent_id,
            "content_preview": "Table",
            "content_location": json.dumps({
                "source": source_id,
                "type": "table",
                "selector": tag.get('_selector', '')
            }),
            "content_hash": self._generate_hash(table_html),
            "metadata": {
                "rows": len(tag.find_all('tr')),
                "has_header": bool(tag.find('thead') or tag.find('th')),
                "id": tag.get('id', ''),
                "class": tag.get('class', ''),
                "full_path": source_id
            }
        }

        elements.append(table_element)
        tag._element_id = table_id

        # Create relationship from parent to table
        contains_relationship = {
            "relationship_id": self._generate_id("rel_"),
            "source_id": parent_id,
            "target_id": table_id,
            "relationship_type": RelationshipType.CONTAINS.value,
            "metadata": {
                "confidence": 1.0
            }
        }
        relationships.append(contains_relationship)

        # Create inverse relationship
        contained_by_relationship = {
            "relationship_id": self._generate_id("rel_"),
            "source_id": table_id,
            "target_id": parent_id,
            "relationship_type": RelationshipType.CONTAINED_BY.value,
            "metadata": {
                "confidence": 1.0
            }
        }
        relationships.append(contained_by_relationship)

        # Process headers
        header_row = tag.find('thead')
        if header_row:
            header_cells = header_row.find_all('th')
            for i, cell in enumerate(header_cells):
                cell_text = cell.get_text(separator=' ', strip=True)
                if not cell_text:
                    continue

                cell_id = self._generate_id("th_")

                # Extract dates from header cell text
                self._extract_dates_from_text(cell_text, cell_id, element_dates)

                cell_element = {
                    "element_id": cell_id,
                    "doc_id": doc_id,
                    "element_type": "table_header",
                    "parent_id": table_id,
                    "content_preview": cell_text[:self.max_content_preview] + (
                        "..." if len(cell_text) > self.max_content_preview else ""),
                    "content_location": json.dumps({
                        "source": source_id,
                        "type": "table_header",
                        "col": i,
                        "selector": cell.get('_selector', '')
                    }),
                    "content_hash": self._generate_hash(cell_text),
                    "metadata": {
                        "col": i,
                        "full_path": source_id
                    }
                }

                elements.append(cell_element)

                # Create relationship from table to header cell
                contains_header_relationship = {
                    "relationship_id": self._generate_id("rel_"),
                    "source_id": table_id,
                    "target_id": cell_id,
                    "relationship_type": RelationshipType.CONTAINS_TABLE_HEADER.value,
                    "metadata": {
                        "confidence": 1.0,
                        "col": i
                    }
                }
                relationships.append(contains_header_relationship)

                # Create inverse relationship
                header_contained_relationship = {
                    "relationship_id": self._generate_id("rel_"),
                    "source_id": cell_id,
                    "target_id": table_id,
                    "relationship_type": RelationshipType.CONTAINED_BY.value,
                    "metadata": {
                        "confidence": 1.0
                    }
                }
                relationships.append(header_contained_relationship)

                # Extract links from header cell
                for a in cell.find_all('a', href=True):
                    links.append({
                        "source_id": cell_id,
                        "link_text": a.get_text(separator=' ', strip=True),
                        "link_target": a['href'],
                        "link_type": "html"
                    })

        # Process rows
        tbody = tag.find('tbody') or tag
        for i, row in enumerate(tbody.find_all('tr')):
            row_text = row.get_text(separator=' ', strip=True)

            # Skip completely empty rows
            if not row_text or len(row_text.strip()) == 0:
                continue

            # Also skip rows that only contain separators (e.g., "| | |")
            cells = row.find_all(['td', 'th'])
            cell_texts = [cell.get_text(separator=' ', strip=True) for cell in cells]
            non_empty_cells = [text for text in cell_texts if text.strip()]
            if not non_empty_cells:
                continue

            row_id = self._generate_id("tr_")

            # Extract dates from row text
            self._extract_dates_from_text(row_text, row_id, element_dates)

            # Create meaningful content preview from actual row content
            if row_text and len(row_text.strip()) > 0:
                # Get cell contents and create a readable preview
                cells = row.find_all(['td', 'th'])
                cell_texts = [cell.get_text(separator=' ', strip=True) for cell in cells if cell.get_text(separator=' ', strip=True)]

                if cell_texts:
                    # Join first few non-empty cells for preview
                    preview_parts = []
                    char_count = 0
                    for cell_text in cell_texts:
                        if char_count + len(cell_text) + 3 <= self.max_content_preview:  # +3 for " | "
                            preview_parts.append(cell_text)
                            char_count += len(cell_text) + 3
                        else:
                            # Add truncated version of current cell if we have room
                            remaining = self.max_content_preview - char_count - 3
                            if remaining > 10:  # Only add if meaningful space left
                                preview_parts.append(cell_text[:remaining] + "...")
                            break

                    content_preview = " | ".join(preview_parts) if preview_parts else f"Row {i + 1}"
                else:
                    content_preview = f"Row {i + 1}"  # Fallback for empty rows
            else:
                content_preview = f"Row {i + 1}"  # Fallback for empty rows

            row_element = {
                "element_id": row_id,
                "doc_id": doc_id,
                "element_type": "table_row",
                "parent_id": table_id,
                "content_preview": content_preview,
                "content_location": json.dumps({
                    "source": source_id,
                    "type": "table_row",
                    "row": i,
                    "selector": row.get('_selector', '')
                }),
                "content_hash": self._generate_hash(str(row)),
                "metadata": {
                    "row": i,
                    "full_path": source_id
                }
            }

            elements.append(row_element)

            # Create relationship from table to row
            contains_row_relationship = {
                "relationship_id": self._generate_id("rel_"),
                "source_id": table_id,
                "target_id": row_id,
                "relationship_type": RelationshipType.CONTAINS_TABLE_ROW.value,
                "metadata": {
                    "confidence": 1.0,
                    "row": i
                }
            }
            relationships.append(contains_row_relationship)

            # Create inverse relationship
            row_contained_relationship = {
                "relationship_id": self._generate_id("rel_"),
                "source_id": row_id,
                "target_id": table_id,
                "relationship_type": RelationshipType.CONTAINED_BY.value,
                "metadata": {
                    "confidence": 1.0
                }
            }
            relationships.append(row_contained_relationship)

            # Process cells
            for j, cell in enumerate(row.find_all(['td', 'th'])):
                cell_text = cell.get_text(separator=' ', strip=True)
                if not cell_text:
                    continue

                cell_id = self._generate_id("td_")
                cell_type = "table_header" if cell.name == 'th' else "table_cell"

                # Extract dates from cell text
                self._extract_dates_from_text(cell_text, cell_id, element_dates)

                cell_element = {
                    "element_id": cell_id,
                    "doc_id": doc_id,
                    "element_type": cell_type,
                    "parent_id": row_id,
                    "content_preview": cell_text[:self.max_content_preview] + (
                        "..." if len(cell_text) > self.max_content_preview else ""),
                    "content_location": json.dumps({
                        "source": source_id,
                        "type": cell_type,
                        "row": i,
                        "col": j,
                        "selector": cell.get('_selector', '')
                    }),
                    "content_hash": self._generate_hash(cell_text),
                    "metadata": {
                        "row": i,
                        "col": j,
                        "colspan": cell.get('colspan', '1'),
                        "rowspan": cell.get('rowspan', '1'),
                        "full_path": source_id
                    }
                }

                elements.append(cell_element)

                # Create relationship from row to cell
                rel_type = RelationshipType.CONTAINS_TABLE_HEADER.value if cell_type == "table_header" else RelationshipType.CONTAINS_TABLE_CELL.value
                contains_cell_relationship = {
                    "relationship_id": self._generate_id("rel_"),
                    "source_id": row_id,
                    "target_id": cell_id,
                    "relationship_type": rel_type,
                    "metadata": {
                        "confidence": 1.0,
                        "col": j
                    }
                }
                relationships.append(contains_cell_relationship)

                # Create inverse relationship
                cell_contained_relationship = {
                    "relationship_id": self._generate_id("rel_"),
                    "source_id": cell_id,
                    "target_id": row_id,
                    "relationship_type": RelationshipType.CONTAINED_BY.value,
                    "metadata": {
                        "confidence": 1.0
                    }
                }
                relationships.append(cell_contained_relationship)

                # Extract links from cell
                for a in cell.find_all('a', href=True):
                    links.append({
                        "source_id": cell_id,
                        "link_text": a.get_text(separator=' ', strip=True),
                        "link_target": a['href'],
                        "link_type": "html"
                    })

        return elements, links, relationships

    def supports_location(self, content_location: Dict[str, Any]) -> bool:
        """
        Check if this parser supports resolving the given location.

        Args:
            content_location: Content location pointer

        Returns:
            True if supported, False otherwise
        """
        try:
            location_data = content_location
            source = location_data.get("source", "")

            # Strip timestamp suffix if present (format: path::timestamp)
            actual_source = source.split('::')[0] if '::' in source else source

            # Check if source exists and is a file
            if not os.path.exists(actual_source) or not os.path.isfile(actual_source):
                return False

            # Check file extension for HTML
            _, ext = os.path.splitext(actual_source.lower())
            return ext in ['.html', '.htm', '.xhtml']

        except (json.JSONDecodeError, TypeError):
            return False

    def _extract_links(self, content: str, element_id: str) -> List[Dict[str, Any]]:
        """
        Extract links from HTML content.

        Args:
            content: HTML content
            element_id: ID of the element containing the links

        Returns:
            List of extracted links
        """
        links = []

        # Parse HTML with BeautifulSoup
        soup = BeautifulSoup(content, 'html.parser')

        # Extract all links
        for a in soup.find_all('a', href=True):
            links.append({
                "source_id": element_id,
                "link_text": a.get_text(separator=' ', strip=True),
                "link_target": a['href'],
                "link_type": "html"
            })

        return links
