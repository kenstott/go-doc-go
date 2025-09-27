"""Go-based CSV parser implementation."""

import json
import subprocess
import tempfile
import os
from pathlib import Path
from typing import Dict, Any, List, Optional, Tuple

from src.go_doc_go.document_parser.base import DocumentParser
from src.go_doc_go.storage.element_element import ElementType


class GoCSVParser(DocumentParser):
    """CSV parser that uses Go binary for processing."""

    def __init__(self, config: Optional[Dict] = None):
        """Initialize Go CSV parser."""
        super().__init__(config)

        # Find the Go binary
        project_root = Path(__file__).parent.parent.parent.parent
        self.binary_path = project_root / "bin" / "csvparser"

        if not self.binary_path.exists():
            raise RuntimeError(f"Go CSV parser binary not found at {self.binary_path}")

        # Parser configuration
        self.max_content_preview = self.config.get("max_content_preview", 100)
        self.extract_header = self.config.get("extract_header", True)
        self.delimiter = self.config.get("delimiter", ",")
        self.max_rows = self.config.get("max_rows", 1000)
        self.strip_whitespace = self.config.get("strip_whitespace", True)
        self.enable_link_extraction = self.config.get("enable_link_extraction", True)

    def parse(self, content: Dict[str, Any]) -> Dict[str, Any]:
        """Parse CSV document using Go binary.

        Args:
            content: Dictionary containing document content and metadata

        Returns:
            Parsed document with elements and relationships
        """
        # Validate input
        doc_id = content.get("id", "")
        csv_content = content.get("content", "")
        metadata = content.get("metadata", {})

        if not doc_id:
            raise ValueError("Document ID is required")
        if not csv_content:
            raise ValueError("Content is required")

        # Prepare command arguments
        cmd_args = [str(self.binary_path), "-stdin", "-json"]

        # Add delimiter if not default
        if self.delimiter != ",":
            cmd_args.extend(["-delimiter", self.delimiter])

        # Add no-header flag if needed
        if not self.extract_header:
            cmd_args.append("-no-header")

        # Add max rows if not default
        if self.max_rows != 1000:
            cmd_args.extend(["-max-rows", str(self.max_rows)])

        # Add document ID
        cmd_args.extend(["-id", doc_id])

        try:
            # Call Go binary with JSON output
            result = subprocess.run(
                cmd_args,
                input=csv_content,
                capture_output=True,
                text=True,
                timeout=60  # 60 second timeout for large CSV files
            )

            if result.returncode != 0:
                error_msg = result.stderr or result.stdout
                raise RuntimeError(f"Go CSV parser failed: {error_msg}")

            # Parse JSON response
            response = json.loads(result.stdout)

            # Convert element types to Python ElementType enum values
            for element in response.get("elements", []):
                go_type = element.get("element_type", "")
                if go_type == "root":
                    element["element_type"] = ElementType.ROOT.value
                elif go_type == "table":
                    element["element_type"] = ElementType.TABLE.value
                elif go_type == "table_row":
                    element["element_type"] = ElementType.TABLE_ROW.value
                elif go_type == "table_header_row":
                    element["element_type"] = ElementType.TABLE_HEADER_ROW.value
                elif go_type == "table_cell":
                    element["element_type"] = ElementType.TABLE_CELL.value
                else:
                    # Keep original type if not recognized
                    pass

            return response

        except subprocess.TimeoutExpired:
            raise RuntimeError("CSV parsing timed out - file may be too large")
        except json.JSONDecodeError as e:
            raise RuntimeError(f"Failed to parse Go CSV parser output: {e}")
        except Exception as e:
            raise RuntimeError(f"Unexpected error during CSV parsing: {e}")

    def _resolve_element_content(self, element: Dict[str, Any]) -> str:
        """Resolve content for an element.

        Args:
            element: The element to resolve content for

        Returns:
            The resolved content string
        """
        # For CSV elements, content is usually in 'text' or 'content' field
        if "text" in element and element["text"]:
            return element["text"]
        if "content" in element and element["content"]:
            return element["content"]

        # Otherwise use content preview
        return element.get("content_preview", "")

    def _resolve_element_text(self, element: Dict[str, Any]) -> str:
        """Resolve text content for an element.

        Args:
            element: The element to resolve text for

        Returns:
            The resolved text string
        """
        # For CSV elements, prioritize text field
        if "text" in element and element["text"]:
            return element["text"]

        # Fall back to content
        if "content" in element and element["content"]:
            return element["content"]

        # Use content preview as last resort
        return element.get("content_preview", "")

    def supports_location(self, location_type: str) -> bool:
        """Check if parser supports a location type.

        Args:
            location_type: Type of location to check

        Returns:
            True if location type is supported
        """
        return location_type in ["csv_row", "csv_cell", "table_position"]