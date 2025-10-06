"""
Document type detector module for the document pointer system.

This module provides utilities to detect document types based on file extension,
MIME type, or content inspection.
"""

import base64
import json
import logging
import mimetypes
import os
import re
import platform
import subprocess
from pathlib import Path
from threading import RLock

logger = logging.getLogger(__name__)

# Check if running on Linux
IS_LINUX = platform.system() == 'Linux'

# Define paths to vendor magic files for Linux
if IS_LINUX:
    # Import get_vendor_path from wherever it's defined in your project
    from go_doc_go.vendor import get_vendor_path  # Adjust import path as needed

    VENDOR_PATH = get_vendor_path()
    VENDOR_MAGIC_PATH = os.path.join(VENDOR_PATH, 'libmagic')
    MAGIC_DB_PATH = os.path.join(VENDOR_MAGIC_PATH, 'magic.mgc')
else:
    # Set to None for non-Linux systems
    VENDOR_MAGIC_PATH = None
    MAGIC_DB_PATH = None

# Global flag for magic availability
MAGIC_AVAILABLE = False

def initialize_magic():
    """Initialize and configure magic library based on current platform."""
    global MAGIC_AVAILABLE

    try:
        import magic

        # Configure magic to use vendor paths on Linux
        if IS_LINUX and VENDOR_MAGIC_PATH and os.path.exists(VENDOR_MAGIC_PATH):
            logger.info(f"Running on Linux, using custom magic binaries at {VENDOR_MAGIC_PATH}")

            # Check which python-magic library we have
            if hasattr(magic, 'Magic'):
                # This is python-magic from PyPI
                try:
                    # Create a new Magic instance with our custom database
                    magic_instance = magic.Magic(magic_file=MAGIC_DB_PATH, mime=True)

                    # Test if it works
                    test_result = magic_instance.from_buffer(b"test")

                    # Store original functions for fallback
                    original_from_file = magic.from_file
                    original_from_buffer = magic.from_buffer

                    # Create wrapper functions that use our custom instance
                    def magic_from_file(filename, mime=False):
                        try:
                            return magic_instance.from_file(filename)
                        except Exception as e:
                            logger.warning(f"Custom magic failed, falling back to default: {e}")
                            return original_from_file(filename, mime)

                    def magic_from_buffer(buffer, mime=False):
                        try:
                            return magic_instance.from_buffer(buffer)
                        except Exception as e:
                            logger.warning(f"Custom magic failed, falling back to default: {e}")
                            return original_from_buffer(buffer, mime)

                    # Replace the default functions with our wrappers
                    magic.from_file = magic_from_file
                    magic.from_buffer = magic_from_buffer

                    logger.info("Successfully configured python-magic with custom database")
                except Exception as e:
                    logger.warning(f"Failed to initialize custom magic instance: {e}")

            elif hasattr(magic, 'MAGIC_MIME_TYPE'):
                # This is file-magic from PyPI or ctypes-based libmagic

                # Set environment variables for libmagic to find the database
                os.environ['MAGIC'] = MAGIC_DB_PATH
                os.environ['MAGICPATH'] = VENDOR_MAGIC_PATH

                # Set LD_LIBRARY_PATH to include our vendor libmagic location
                lib_path = os.path.join(VENDOR_MAGIC_PATH, 'lib')
                if os.path.exists(lib_path):
                    if 'LD_LIBRARY_PATH' in os.environ:
                        os.environ['LD_LIBRARY_PATH'] = f"{lib_path}:{os.environ['LD_LIBRARY_PATH']}"
                    else:
                        os.environ['LD_LIBRARY_PATH'] = lib_path

                logger.info("Set environment variables for libmagic")

        MAGIC_AVAILABLE = True
        logger.info("Magic library initialized successfully")
    except ImportError:
        MAGIC_AVAILABLE = False
        logger.warning("python-magic not available. Install with 'pip install python-magic' for better content detection.")

# Initialize magic at module import
initialize_magic()


class GoDocumentTypeDetector:
    """Go-based document type detector implementation via subprocess calls."""

    def __init__(self):
        """Initialize Go document type detector wrapper."""
        self._lock = RLock()

        # Find the Go binary
        project_root = Path(__file__).parent.parent.parent.parent
        self.binary_path = project_root / "bin" / "doctype"

        if not self.binary_path.exists():
            raise FileNotFoundError(
                f"Go document type detector binary not found. Expected at {self.binary_path}. "
                "Please build it with: cd go && go build -o ../bin/doctype ./cmd/doctype"
            )

    def _run_command(self, *args):
        """Run the Go binary with the given arguments."""
        cmd = [str(self.binary_path), *args]

        try:
            result = subprocess.run(
                cmd,
                capture_output=True,
                text=True,
                timeout=5  # 5 second timeout
            )

            if result.returncode != 0:
                logger.error(f"Go detector command failed: {result.stderr}")
                return None

            return result.stdout.strip()
        except subprocess.TimeoutExpired:
            logger.error("Go detector command timed out")
            return None
        except Exception as e:
            logger.error(f"Error running Go detector: {e}")
            return None

    def detect_from_path(self, path):
        """
        Detect document type from file path.

        Args:
            path: Path to the file

        Returns:
            String representing document type: 'markdown', 'docx', etc.
        """
        # Match Python behavior: return None for empty path
        if not path:
            return None

        with self._lock:
            result = self._run_command("detect_from_path", str(path))
            if result:
                try:
                    response = json.loads(result)
                    return response.get("document_type", "text")
                except json.JSONDecodeError:
                    logger.error(f"Failed to parse Go detector response: {result}")
            return "text"

    def detect_from_mime(self, path):
        """
        Detect document type from MIME type.

        Args:
            path: Path to the file

        Returns:
            String representing document type
        """
        # For compatibility, use detect_from_path since Go's MIME detection
        # is integrated into the path detection
        return self.detect_from_path(path)

    def detect_from_content(self, content, metadata=None):
        """
        Detect document type by inspecting content.

        Args:
            content: File content (bytes or string)
            metadata: Optional metadata that might provide hints

        Returns:
            String representing document type
        """
        with self._lock:
            # Convert content to bytes if it's a string
            if isinstance(content, str):
                content_bytes = content.encode('utf-8')
            else:
                content_bytes = content

            # Encode content as base64 for transmission
            content_b64 = base64.b64encode(content_bytes).decode('ascii')

            # Prepare arguments
            args = ["detect_from_content", content_b64]
            if metadata:
                metadata_json = json.dumps(metadata)
                args.append(metadata_json)

            result = self._run_command(*args)
            if result:
                try:
                    response = json.loads(result)
                    return response.get("document_type", "text")
                except json.JSONDecodeError:
                    logger.error(f"Failed to parse Go detector response: {result}")
            return "text"

    def detect(self, path=None, content=None, metadata=None):
        """
        Detect document type using all available methods.

        Args:
            path: Optional file path
            content: Optional file content
            metadata: Optional metadata hints

        Returns:
            String representing document type
        """
        # Match Python behavior: try path detection first, then content detection
        # Try path-based detection first (if we have a path)
        if path:
            doc_type = self.detect_from_path(path)
            # Only use path result if it's definitive (not None and not default text)
            if doc_type and doc_type != "text":
                return doc_type

        # Then try content-based detection
        if content is not None and len(content) > 0:
            doc_type = self.detect_from_content(content, metadata)
            # Only use content result if it's definitive
            if doc_type and doc_type != "text":
                return doc_type

        # Default to text
        return "text"

    # Provide static methods for compatibility with existing code
    @staticmethod
    def _create_go_instance():
        """Create a Go detector instance if available."""
        try:
            return GoDocumentTypeDetector()
        except FileNotFoundError:
            return None

    _go_instance = None

    @classmethod
    def get_go_instance(cls):
        """Get or create the Go detector instance."""
        if cls._go_instance is None:
            cls._go_instance = cls._create_go_instance()
        return cls._go_instance


def create_document_type_detector():
    """
    Factory function to create document type detector instance.

    Uses Go implementation if USE_GO_DETECTOR environment variable is set,
    otherwise uses Python implementation.

    Returns:
        DocumentTypeDetector or GoDocumentTypeDetector instance
    """
    # Check if Go modules should be used (unified flag or specific flag)
    use_go_modules = os.environ.get("USE_GO_MODULES", "").lower() in ("true", "1", "yes")
    use_go_detector = os.environ.get("USE_GO_DETECTOR", "").lower() in ("true", "1", "yes")
    use_go = use_go_modules or use_go_detector

    if use_go:
        try:
            detector = GoDocumentTypeDetector()
            logger.info("Using Go document type detector implementation")
            return detector
        except FileNotFoundError as e:
            logger.warning(f"Go detector not available: {e}. Falling back to Python implementation.")

    logger.debug("Using Python document type detector implementation")
    return DocumentTypeDetector()


class DocumentTypeDetector:
    """Detects document type from various inputs."""

    # Centralized MIME type to document type mapping
    MIME_TYPE_MAP = {
        'text/markdown': 'markdown',
        'application/vnd.openxmlformats-officedocument.wordprocessingml.document': 'docx',
        'application/msword': 'docx',
        'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet': 'xlsx',
        'application/vnd.ms-excel': 'xlsx',
        'application/vnd.openxmlformats-officedocument.presentationml.presentation': 'pptx',
        'application/vnd.ms-powerpoint': 'pptx',
        'application/pdf': 'pdf',
        'text/html': 'html',
        'application/xhtml+xml': 'html',
        'text/plain': 'text',
        'text/csv': 'csv',
        'text/tab-separated-values': 'csv',
        'application/csv': 'csv',
        'application/json': 'json',
        'application/xml': 'xml',
        'text/xml': 'xml',
        'application/x-yaml': 'yaml',
        'text/yaml': 'yaml',
        'application/yaml': 'yaml',
        'image/svg+xml': 'xml',
        'application/rdf+xml': 'xml',
        'application/rss+xml': 'xml',
        'application/xslt+xml': 'xml',
        'application/wsdl+xml': 'xml'
    }

    # Centralized file extension mapping
    EXTENSION_MAP = {
        '.md': 'markdown',
        '.markdown': 'markdown',
        '.mdown': 'markdown',
        '.docx': 'docx',
        '.doc': 'docx',
        '.xlsx': 'xlsx',
        '.xls': 'xlsx',
        '.pptx': 'pptx',
        '.ppt': 'pptx',
        '.pdf': 'pdf',
        '.html': 'html',
        '.htm': 'html',
        '.xhtml': 'html',
        '.txt': 'text',
        '.text': 'text',
        '.csv': 'csv',
        '.tsv': 'csv',
        '.json': 'json',
        '.xml': 'xml',
        '.xsd': 'xml',
        '.rdf': 'xml',
        '.rss': 'xml',
        '.svg': 'xml',
        '.wsdl': 'xml',
        '.xslt': 'xml',
        '.yaml': 'yaml',
        '.yml': 'yaml'
    }

    # Binary file signatures (magic numbers)
    BINARY_SIGNATURES = {
        b'%PDF': 'pdf',
        b'PK\x03\x04': 'zip',  # ZIP files (could be docx, xlsx, pptx)
        b'\xD0\xCF\x11\xE0\xA1\xB1\x1A\xE1': 'ms_compound',  # MS Compound File (older doc, xls, ppt)
    }

    @staticmethod
    def detect_from_path(path):
        """
        Detect document type from file path.

        Args:
            path: Path to the file

        Returns:
            String representing document type: 'markdown', 'docx', etc.
        """
        if not path:
            return None

        # Use file extension for detection
        extension = Path(path).suffix.lower()

        # Return matched type or fallback to MIME detection
        return DocumentTypeDetector.EXTENSION_MAP.get(extension, DocumentTypeDetector.detect_from_mime(path))

    @staticmethod
    def detect_from_mime(path):
        """
        Detect document type from MIME type.

        Args:
            path: Path to the file

        Returns:
            String representing document type
        """
        # Get MIME type
        mime_type, _ = mimetypes.guess_type(path)

        # If MIME type detection failed and python-magic is available, try it
        if not mime_type and os.path.exists(path) and MAGIC_AVAILABLE:
            try:
                import magic
                mime_type = magic.from_file(path, mime=True)
            except Exception as e:
                logger.debug(f"Error detecting MIME type with python-magic: {str(e)}")

        # Return matched type or default to text
        return DocumentTypeDetector.MIME_TYPE_MAP.get(mime_type, 'text')

    @staticmethod
    def detect_from_content(content, metadata=None):
        """
        Detect document type by inspecting content.

        Args:
            content: File content (bytes or string)
            metadata: Optional metadata that might provide hints

        Returns:
            String representing document type
        """
        # Check metadata hints first if provided
        if metadata:
            # Check explicit content type hint
            content_type = metadata.get('content_type')
            if content_type and content_type in DocumentTypeDetector.MIME_TYPE_MAP:
                return DocumentTypeDetector.MIME_TYPE_MAP[content_type]

            # Check column name hint for database content
            content_column = metadata.get('content_column', '')
            if content_column:
                if content_column.endswith('_html'):
                    return 'html'
                elif content_column.endswith(('_md', '_markdown')):
                    return 'markdown'
                elif content_column.endswith('_json'):
                    return 'json'
                elif content_column.endswith('_xml'):
                    return 'xml'
                elif content_column.endswith('_csv'):
                    return 'csv'

        # Ensure we have bytes for binary detection and string for text detection
        # content_bytes = None
        content_str = None

        if isinstance(content, str):
            content_str = content
            content_bytes = content.encode('utf-8')
        else:
            content_bytes = content
            try:
                content_str = content.decode('utf-8')
            except UnicodeDecodeError:
                # Content is definitely binary if it can't be decoded as UTF-8
                pass

        # Check binary signatures for binary content
        if content_bytes:
            for signature, doc_type in DocumentTypeDetector.BINARY_SIGNATURES.items():
                if content_bytes.startswith(signature):
                    # Special handling for ZIP-based Office formats
                    if doc_type == 'zip':
                        # Look for Office XML signatures in the first 4000 bytes
                        content_start = content_bytes[:4000]
                        if b'word/' in content_start:
                            return 'docx'
                        elif b'xl/' in content_start:
                            return 'xlsx'
                        elif b'ppt/' in content_start:
                            return 'pptx'
                    # Return the detected binary type
                    return doc_type

        # Use python-magic if available
        if MAGIC_AVAILABLE:
            try:
                import magic
                mime_type = magic.from_buffer(content_bytes, mime=True)
                doc_type = DocumentTypeDetector.MIME_TYPE_MAP.get(mime_type)
                if doc_type and doc_type != 'text':
                    return doc_type
            except Exception as e:
                logger.debug(f"Error detecting content type with python-magic: {str(e)}")

        # Fallback to text analysis (including for empty strings)
        # Only process if we have valid text content
        if content_str is None:
            return 'binary'

        # Check for JSON first (before CSV to avoid false positives)
        trimmed = content_str.strip()
        if ((trimmed.startswith('{') and trimmed.endswith('}')) or
            (trimmed.startswith('[') and trimmed.endswith(']'))):
            try:
                json.loads(content_str)
                return 'json'
            except json.JSONDecodeError:
                pass

        # Check for CSV format
        if DocumentTypeDetector._is_likely_csv(content_str):
            return 'csv'

        # Check for markdown headers
        if re.search(r'^#{1,6}\s+', content_str, re.MULTILINE):
            return 'markdown'

        # Check for HTML
        if re.search(r'<!DOCTYPE html>|<html|<body|<div|<span|<p>', content_str, re.IGNORECASE):
            return 'html'

        # Check for XML
        if trimmed.startswith('<') and trimmed.endswith('>'):
            if re.search(r'<\?xml|<[a-zA-Z]+>[^<>]*</[a-zA-Z]+>', content_str):
                return 'xml'

        # Default to text for all text content (including empty strings)
        return 'text'

    @staticmethod
    def _is_likely_csv(text):
        """
        Detect if a text string is likely a CSV file.

        Args:
            text: Text content to check

        Returns:
            Boolean indicating if text is likely CSV format
        """
        # Quick check for empty content
        if not text or not text.strip():
            return False

        # Get first few lines
        lines = text.splitlines()[:5]
        if not lines:
            return False

        # Check if consistent delimiters exist
        potential_delimiters = [',', '\t', ';', '|']

        # Count delimiters in each line
        delimiter_counts = {}
        for delimiter in potential_delimiters:
            counts = [line.count(delimiter) for line in lines]
            # If delimiter appears consistently and at least once per line
            if all(count > 0 for count in counts) and max(counts) - min(counts) <= 1:
                delimiter_counts[delimiter] = sum(counts)

        # If we found consistent delimiters
        if delimiter_counts:
            # Choose the most frequent delimiter
            most_frequent = max(delimiter_counts, key=delimiter_counts.get)
            # Verify most lines have approximately same number of fields
            fields_per_line = [len(line.split(most_frequent)) for line in lines]
            avg_fields = sum(fields_per_line) / len(fields_per_line)
            # Check if field count is consistent (within 1 of average)
            if all(abs(fields - avg_fields) <= 1 for fields in fields_per_line):
                return True

        # Check for fixed-width format (harder to detect)
        # TODO: Add fixed-width detection if needed

        return False

    @staticmethod
    def detect(path=None, content=None, metadata=None):
        """
        Detect document type using all available methods.

        Args:
            path: Optional file path
            content: Optional file content
            metadata: Optional metadata hints

        Returns:
            String representing document type
        """
        # Try path-based detection first
        if path:
            doc_type = DocumentTypeDetector.detect_from_path(path)
            # Only use path result if it's definitive (not default fallback to text)
            if doc_type and doc_type != 'text':
                return doc_type

        # Then try content-based detection with metadata hints
        if content is not None and len(content) > 0:
            doc_type = DocumentTypeDetector.detect_from_content(content, metadata)
            # Only use content result if it's definitive (not default fallback)
            if doc_type and doc_type != 'text':
                return doc_type

        # Default to text
        return 'text'
