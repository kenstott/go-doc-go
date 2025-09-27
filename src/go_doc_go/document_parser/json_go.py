"""
Go JSON parser implementation wrapper.

This module provides a wrapper around the Go JSON parser binary,
following the same pattern as the HTML Go parser wrapper.
"""

import json
import logging
import os
import subprocess
import tempfile
from pathlib import Path
from typing import Dict, Any, List, Optional, Union

from .base import DocumentParser

logger = logging.getLogger(__name__)


class GoJSONParser(DocumentParser):
    """Go-based JSON parser wrapper using subprocess calls."""

    def __init__(self, config: Optional[Dict[str, Any]] = None):
        """
        Initialize Go JSON parser wrapper.

        Args:
            config: Parser configuration
        """
        super().__init__(config)

        # Find the Go binary
        project_root = Path(__file__).parent.parent.parent.parent
        self.binary_path = project_root / "bin" / "jsonparser"

        if not self.binary_path.exists():
            raise FileNotFoundError(
                f"Go JSON parser binary not found at {self.binary_path}. "
                "Please build it with: cd go && go build -o ../bin/jsonparser ./cmd/jsonparser"
            )

    def parse(self, content: Dict[str, Any]) -> Dict[str, Any]:
        """
        Parse JSON content using Go implementation.

        Args:
            content: Document content with metadata

        Returns:
            Dict with parsed elements and relationships

        Raises:
            ValueError: If content is missing required fields
            RuntimeError: If Go parser execution fails
        """
        # Validate input
        if "id" not in content:
            raise ValueError("Content must include 'id' field")
        if "content" not in content:
            raise ValueError("Content must include 'content' field")

        doc_id = content["id"]
        json_content = content["content"]
        metadata = content.get("metadata", {})

        # Handle both string and bytes content
        if isinstance(json_content, bytes):
            json_content = json_content.decode('utf-8')

        # Write content to temporary file
        with tempfile.NamedTemporaryFile(mode='w', suffix='.json', delete=False) as temp_file:
            temp_file.write(json_content)
            temp_file_path = temp_file.name

        try:
            # Run Go parser
            cmd = [
                str(self.binary_path),
                "-input", temp_file_path,
                "-json",
                "-id", doc_id
            ]

            result = subprocess.run(
                cmd,
                capture_output=True,
                text=True,
                timeout=30  # 30 second timeout
            )

            if result.returncode != 0:
                error_msg = f"Go JSON parser failed: {result.stderr}"
                logger.error(error_msg)
                raise RuntimeError(error_msg)

            # Parse the JSON output
            try:
                parsed_result = json.loads(result.stdout)
            except json.JSONDecodeError as e:
                error_msg = f"Failed to parse Go parser output: {e}"
                logger.error(error_msg)
                raise RuntimeError(error_msg)

            # Convert to standard parser format
            return self._convert_go_output_to_standard_format(parsed_result, metadata)

        except subprocess.TimeoutExpired:
            error_msg = "Go JSON parser timed out"
            logger.error(error_msg)
            raise RuntimeError(error_msg)
        except Exception as e:
            error_msg = f"Error running Go JSON parser: {e}"
            logger.error(error_msg)
            raise RuntimeError(error_msg)
        finally:
            # Clean up temporary file
            try:
                os.unlink(temp_file_path)
            except OSError:
                pass

    def _convert_go_output_to_standard_format(self, go_output: Dict[str, Any],
                                             original_metadata: Dict[str, Any]) -> Dict[str, Any]:
        """
        Convert Go parser output to standard parser format.

        Args:
            go_output: Output from Go parser
            original_metadata: Original document metadata

        Returns:
            Dict with converted data in standard parser format
        """
        # Extract document info
        document = go_output.get("document", {})

        # Merge original metadata with Go parser metadata
        merged_metadata = {**original_metadata, **document.get("metadata", {})}

        # Update document with merged metadata
        document_info = {
            "doc_id": document.get("doc_id"),
            "doc_type": document.get("doc_type", "json"),
            "source": document.get("source"),
            "content_hash": document.get("content_hash"),
            "metadata": merged_metadata
        }

        # Convert elements
        elements = []
        for element in go_output.get("elements", []):
            converted_element = {
                "element_id": element.get("element_id"),
                "doc_id": element.get("doc_id"),
                "element_type": element.get("element_type"),
                "parent_id": element.get("parent_id"),
                "content_preview": element.get("content_preview", ""),
                "content_location": element.get("content_location", {}),
                "content_hash": element.get("content_hash"),
                "element_order": element.get("element_order", 0),
                "document_position": element.get("document_position", 0),
                "metadata": element.get("metadata", {}),
                "text": element.get("text", ""),
                "content": element.get("content", ""),
                "temporal_value": element.get("temporal_value")
            }
            elements.append(converted_element)

        # Convert relationships
        relationships = []
        for rel in go_output.get("relationships", []):
            converted_rel = {
                "relationship_id": rel.get("relationship_id"),
                "source_element_id": rel.get("source_element_id"),
                "target_element_id": rel.get("target_element_id"),
                "relationship_type": rel.get("relationship_type"),
                "confidence": rel.get("confidence", 1.0),
                "metadata": rel.get("metadata", {})
            }
            relationships.append(converted_rel)

        # Convert links
        links = []
        for link in go_output.get("links", []):
            converted_link = {
                "source_id": link.get("source_id"),
                "link_text": link.get("link_text"),
                "link_target": link.get("link_target"),
                "link_type": link.get("link_type")
            }
            links.append(converted_link)

        # Handle dates
        dates = go_output.get("dates", {})

        return {
            "document": document_info,
            "elements": elements,
            "relationships": relationships,
            "links": links,
            "dates": dates
        }

    def get_supported_formats(self) -> List[str]:
        """Return list of supported formats."""
        return ["json"]

    def can_parse(self, content: Dict[str, Any]) -> bool:
        """
        Check if this parser can handle the given content.

        Args:
            content: Document content to check

        Returns:
            True if parser can handle this content
        """
        doc_type = content.get("doc_type", "").lower()
        if doc_type == "json":
            return True

        # Check metadata for JSON indicators
        metadata = content.get("metadata", {})
        content_type = metadata.get("content_type", "").lower()
        filename = metadata.get("filename", "").lower()

        return (
            "json" in content_type or
            filename.endswith(('.json', '.jsonl', '.ndjson'))
        )

    def _resolve_element_content(self, location_data: Dict[str, Any],
                                source_content: Optional[Union[str, bytes]]) -> str:
        """
        Resolve content for JSON-specific element types.

        Args:
            location_data: Content location data
            source_content: Optional preloaded source content

        Returns:
            Resolved content string
        """
        element_type = location_data.get("type", "")
        path = location_data.get("path", "")

        if element_type in ["json_object", "json_array", "json_field", "json_item"]:
            # For JSON elements, we could parse the source and extract the specific path
            # For now, return a placeholder indicating the JSON path
            return f"JSON element at path: {path}"

        return "Content not available for this element type"

    def _resolve_element_text(self, location_data: Dict[str, Any],
                             source_content: Optional[Union[str, bytes]]) -> str:
        """
        Resolve text content for JSON-specific element types.

        Args:
            location_data: Content location data
            source_content: Optional preloaded source content

        Returns:
            Resolved text string
        """
        element_type = location_data.get("type", "")
        path = location_data.get("path", "")

        if element_type in ["json_object", "json_array", "json_field", "json_item"]:
            # For JSON elements, extract text representation
            return f"JSON text at path: {path}"

        return "Text not available for this element type"

    def supports_location(self, content_location: str) -> bool:
        """
        Check if this parser supports resolving the given location.

        Args:
            content_location: Content location pointer

        Returns:
            True if supported, False otherwise
        """
        try:
            location_data = json.loads(content_location) if isinstance(content_location, str) else content_location
            element_type = location_data.get("type", "")
            return element_type in ["json_object", "json_array", "json_field", "json_item", "root"]
        except (json.JSONDecodeError, TypeError):
            return False