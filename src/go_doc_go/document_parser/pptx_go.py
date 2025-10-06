"""
Python wrapper for the Go PPTX parser implementation.

This module provides a Python interface to the high-performance Go PPTX parser,
enabling seamless integration with existing Python codebases while leveraging
Go's superior performance for presentation processing.
"""

import json
import logging
import os
import subprocess
import tempfile
from pathlib import Path
from typing import Dict, Any, Optional

from .base import DocumentParser

logger = logging.getLogger(__name__)


class PptxGoParser(DocumentParser):
    """Python wrapper for the Go PPTX parser implementation."""

    def __init__(self, config: Optional[Dict[str, Any]] = None):
        """Initialize the Go PPTX parser wrapper."""
        super().__init__(config)
        self.config = config or {}

        # Configuration options
        self.max_content_preview = self.config.get("max_content_preview", 100)
        self.extract_notes = self.config.get("extract_notes", True)
        self.extract_comments = self.config.get("extract_comments", True)
        self.extract_shapes = self.config.get("extract_shapes", True)
        self.extract_tables = self.config.get("extract_tables", True)
        self.extract_images = self.config.get("extract_images", True)

        # Go binary path
        self.go_binary_path = self._find_go_binary()
        if not self.go_binary_path:
            raise RuntimeError("Go PPTX parser binary not found. Please ensure it's built and in PATH or bin/ directory.")

    def _find_go_binary(self) -> Optional[str]:
        """Find the Go PPTX parser binary."""
        # Check relative path first (for development)
        current_dir = Path(__file__).parent
        relative_binary = current_dir.parent.parent.parent / "bin" / "pptxparser"

        if relative_binary.exists() and os.access(relative_binary, os.X_OK):
            return str(relative_binary)

        # Check if it's in PATH
        try:
            result = subprocess.run(["which", "pptxparser"], capture_output=True, text=True)
            if result.returncode == 0:
                return result.stdout.strip()
        except (subprocess.SubprocessError, FileNotFoundError):
            pass

        return None

    def parse(self, doc_content: Dict[str, Any]) -> Dict[str, Any]:
        """
        Parse a PPTX document using the Go parser.

        Args:
            doc_content: Document content and metadata. Expected format:
                {
                    "id": "document_id",
                    "content": "/path/to/file.pptx" or binary_content,
                    "binary_path": "/path/to/file.pptx" (alternative to content),
                    "metadata": {...}
                }

        Returns:
            Dictionary containing parsed document structure with elements,
            relationships, and metadata.
        """
        # Extract parameters
        doc_id = doc_content.get("id", "unknown")
        content = doc_content.get("content")
        binary_path = doc_content.get("binary_path")
        metadata = doc_content.get("metadata", {})

        # Determine input file path
        input_file_path = None
        temp_file = None

        try:
            if binary_path and os.path.isfile(binary_path):
                # Use provided file path
                input_file_path = binary_path
            elif content and os.path.isfile(content):
                # Content is a file path
                input_file_path = content
            elif content and isinstance(content, (bytes, str)):
                # Content is binary data, write to temporary file
                temp_file = tempfile.NamedTemporaryFile(suffix='.pptx', delete=False)
                if isinstance(content, str):
                    # If content is string (path), try to read it
                    if os.path.isfile(content):
                        input_file_path = content
                    else:
                        # Assume it's base64 or similar encoded content
                        temp_file.write(content.encode('utf-8'))
                else:
                    # Binary content
                    temp_file.write(content)
                temp_file.close()
                input_file_path = temp_file.name
            else:
                raise ValueError("No valid PPTX content or file path provided")

            # Validate file exists
            if not os.path.isfile(input_file_path):
                raise FileNotFoundError(f"PPTX file not found: {input_file_path}")

            # Prepare command arguments
            cmd = [
                self.go_binary_path,
                "-file", input_file_path,
                "-max-preview", str(self.max_content_preview),
                "-extract-notes=" + str(self.extract_notes).lower(),
                "-extract-comments=" + str(self.extract_comments).lower(),
                "-extract-shapes=" + str(self.extract_shapes).lower(),
                "-extract-tables=" + str(self.extract_tables).lower(),
                "-extract-images=" + str(self.extract_images).lower(),
                "-verbose=false"  # Disable verbose output for cleaner JSON
            ]

            # Execute Go parser
            logger.debug(f"Executing Go PPTX parser: {' '.join(cmd)}")

            try:
                result = subprocess.run(
                    cmd,
                    capture_output=True,
                    text=True,
                    timeout=30,  # 30 second timeout
                    check=True
                )
            except subprocess.TimeoutExpired:
                raise RuntimeError("Go PPTX parser timed out after 30 seconds")
            except subprocess.CalledProcessError as e:
                error_msg = e.stderr.strip() if e.stderr else "Unknown error"
                raise RuntimeError(f"Go PPTX parser failed: {error_msg}")

            # Parse JSON output
            try:
                parsed_result = json.loads(result.stdout)
            except json.JSONDecodeError as e:
                logger.error(f"Failed to parse Go parser output as JSON: {e}")
                logger.error(f"Parser stdout: {result.stdout[:500]}...")
                logger.error(f"Parser stderr: {result.stderr}")
                raise RuntimeError("Go parser produced invalid JSON output")

            # Update document metadata with Python wrapper info
            if "document" in parsed_result and "metadata" in parsed_result["document"]:
                parsed_result["document"]["metadata"].update({
                    "python_wrapper_version": "1.0.0",
                    "go_parser_used": True,
                    "wrapper_config": {
                        "max_content_preview": self.max_content_preview,
                        "extract_notes": self.extract_notes,
                        "extract_comments": self.extract_comments,
                        "extract_shapes": self.extract_shapes,
                        "extract_tables": self.extract_tables,
                        "extract_images": self.extract_images,
                    }
                })

            # Ensure proper document ID
            if "document" in parsed_result:
                parsed_result["document"]["doc_id"] = doc_id

            logger.debug(f"Successfully parsed PPTX with {len(parsed_result.get('elements', []))} elements")
            return parsed_result

        except Exception as e:
            logger.error(f"Error parsing PPTX document: {e}")
            raise
        finally:
            # Clean up temporary file
            if temp_file and os.path.exists(temp_file.name):
                try:
                    os.unlink(temp_file.name)
                except OSError as e:
                    logger.warning(f"Failed to remove temporary file {temp_file.name}: {e}")

    def supports_location(self, content_location: Dict[str, Any]) -> bool:
        """
        Check if this parser supports resolving the given location.

        Args:
            content_location: Content location pointer

        Returns:
            True if supported, False otherwise
        """
        try:
            source = content_location.get("source", "")

            # Strip timestamp suffix if present
            actual_source = source.split('::')[0] if '::' in source else source

            # Check if source exists and is a PPTX file
            if not os.path.exists(actual_source) or not os.path.isfile(actual_source):
                return False

            # Check file extension
            _, ext = os.path.splitext(actual_source.lower())
            return ext == '.pptx'

        except (TypeError, AttributeError):
            return False

    def _resolve_element_content(self, location_data: Dict[str, Any],
                                source_content: Optional[Any] = None) -> str:
        """
        Resolve content for specific PPTX element types.

        Note: This is a simplified implementation. For full content resolution,
        consider re-parsing the document or implementing caching.

        Args:
            location_data: Content location data
            source_content: Optional pre-loaded source content

        Returns:
            Resolved content string
        """
        # For the Go wrapper, we return basic information
        # Full content resolution would require re-parsing or caching
        element_type = location_data.get("type", "")
        slide_index = location_data.get("slide_index", -1)

        # Return type-specific placeholder content
        if element_type == "slide":
            return f"Slide {slide_index + 1} content (use full parse for complete text)"
        elif element_type == "title":
            return "Title text (use full parse for complete text)"
        elif element_type == "text_box":
            return "Text box content (use full parse for complete text)"
        elif element_type == "table":
            return "Table content (use full parse for complete structure)"
        elif element_type == "slide_notes":
            return f"Notes for slide {slide_index + 1} (use full parse for complete text)"
        elif element_type == "shape":
            shape_index = location_data.get("shape_index", 0)
            return f"Shape {shape_index} on slide {slide_index + 1} (use full parse for complete content)"
        else:
            return "Content (use full parse for complete text)"

    def _resolve_element_text(self, location_data: Dict[str, Any],
                             source_content: Optional[Any] = None) -> str:
        """
        Resolve the plain text representation of a PPTX element.

        Args:
            location_data: Content location data
            source_content: Optional preloaded source content

        Returns:
            Plain text representation of the element
        """
        # For the Go wrapper, we use the content resolution
        return self._resolve_element_content(location_data, source_content)

    def get_parser_info(self) -> Dict[str, Any]:
        """
        Get information about the parser implementation.

        Returns:
            Dictionary with parser information
        """
        return {
            "name": "PptxGoParser",
            "version": "1.0.0",
            "backend": "Go",
            "binary_path": self.go_binary_path,
            "supported_formats": [".pptx"],
            "features": [
                "Slide parsing",
                "Text extraction",
                "Shape processing",
                "Table extraction",
                "Notes extraction",
                "Comment extraction",
                "Image detection",
                "Link extraction",
                "Metadata extraction"
            ],
            "performance": "High (Go implementation)",
            "config": {
                "max_content_preview": self.max_content_preview,
                "extract_notes": self.extract_notes,
                "extract_comments": self.extract_comments,
                "extract_shapes": self.extract_shapes,
                "extract_tables": self.extract_tables,
                "extract_images": self.extract_images,
            }
        }