"""Go-based text parser implementation."""

import json
import subprocess
import os
from pathlib import Path
from typing import Dict, Any, Optional

from src.go_doc_go.document_parser.base import DocumentParser
from src.go_doc_go.storage.element_element import ElementType


class GoTextParser(DocumentParser):
    """Text parser that uses Go binary for processing."""

    def __init__(self, config: Optional[Dict] = None):
        """Initialize Go text parser."""
        super().__init__(config)

        # Find the Go binary
        project_root = Path(__file__).parent.parent.parent.parent
        self.binary_path = project_root / "bin" / "textparser"

        if not self.binary_path.exists():
            raise RuntimeError(f"Go text parser binary not found at {self.binary_path}")

        # Parser configuration
        self.max_content_preview = self.config.get("max_content_preview", 100)
        self.paragraph_separator = self.config.get("paragraph_separator", "\n\n")
        self.min_paragraph_length = self.config.get("min_paragraph_length", 5)
        self.max_elements = self.config.get("max_elements", 1000)
        self.extract_dates = self.config.get("extract_dates", True)
        self.extract_numbers = self.config.get("extract_numbers", True)
        self.enable_link_extraction = self.config.get("enable_link_extraction", True)
        self.strip_whitespace = self.config.get("strip_whitespace", True)

    def parse(self, content: Dict[str, Any]) -> Dict[str, Any]:
        """Parse text document using Go binary.

        Args:
            content: Dictionary containing document content and metadata

        Returns:
            Parsed document with elements and relationships
        """
        # Validate input
        doc_id = content.get("id", "")
        text_content = content.get("content", "")
        metadata = content.get("metadata", {})

        if not doc_id:
            raise ValueError("Document ID is required")
        if not text_content:
            raise ValueError("Content is required")

        # Prepare command arguments
        cmd_args = [str(self.binary_path), "-stdin", "-json"]

        # Add document ID
        cmd_args.extend(["-id", doc_id])

        # Add paragraph separator if not default
        if self.paragraph_separator != "\n\n":
            cmd_args.extend(["-paragraph-sep", self.paragraph_separator])

        # Add minimum paragraph length if not default
        if self.min_paragraph_length != 5:
            cmd_args.extend(["-min-length", str(self.min_paragraph_length)])

        # Add max elements if not default
        if self.max_elements != 1000:
            cmd_args.extend(["-max-elements", str(self.max_elements)])

        # Add feature toggles
        if not self.enable_link_extraction:
            cmd_args.append("-no-links")

        if not self.extract_dates:
            cmd_args.append("-no-dates")

        if not self.extract_numbers:
            cmd_args.append("-no-numbers")

        if not self.strip_whitespace:
            cmd_args.append("-no-whitespace")

        try:
            # Call Go binary with JSON output
            result = subprocess.run(
                cmd_args,
                input=text_content,
                capture_output=True,
                text=True,
                timeout=60  # 60 second timeout for large text files
            )

            if result.returncode != 0:
                error_msg = result.stderr or result.stdout
                raise RuntimeError(f"Go text parser failed: {error_msg}")

            # Parse JSON response
            response = json.loads(result.stdout)

            # Convert element types to Python ElementType enum values
            for element in response.get("elements", []):
                go_type = element.get("element_type", "")
                if go_type == "root":
                    element["element_type"] = ElementType.ROOT.value
                elif go_type == "paragraph":
                    element["element_type"] = ElementType.PARAGRAPH.value
                elif go_type == "line":
                    element["element_type"] = ElementType.LINE.value
                elif go_type == "range":
                    element["element_type"] = ElementType.RANGE.value
                elif go_type == "substring":
                    element["element_type"] = ElementType.SUBSTRING.value
                else:
                    # Keep original type if not recognized
                    pass

            return response

        except subprocess.TimeoutExpired:
            raise RuntimeError("Text parsing timed out - file may be too large")
        except json.JSONDecodeError as e:
            raise RuntimeError(f"Failed to parse Go text parser output: {e}")
        except Exception as e:
            raise RuntimeError(f"Unexpected error during text parsing: {e}")

    def _resolve_element_content(self, element: Dict[str, Any]) -> str:
        """Resolve content for an element.

        Args:
            element: The element to resolve content for

        Returns:
            The resolved content string
        """
        # For text elements, content is usually in 'text' or 'content' field
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
        # For text elements, prioritize text field
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
        return location_type in ["paragraph", "line", "range", "text_position"]