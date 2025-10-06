"""Go-based markdown parser implementation."""

import json
import subprocess
import os
from pathlib import Path
from typing import Dict, Any, Optional

from src.go_doc_go.document_parser.base import DocumentParser
from src.go_doc_go.storage.element_element import ElementType


class GoMarkdownParser(DocumentParser):
    """Markdown parser that uses Go binary for processing."""

    def __init__(self, config: Optional[Dict] = None):
        """Initialize Go markdown parser."""
        super().__init__(config)

        # Find the Go binary
        project_root = Path(__file__).parent.parent.parent.parent
        self.binary_path = project_root / "bin" / "markdownparser"

        if not self.binary_path.exists():
            raise RuntimeError(f"Go markdown parser binary not found at {self.binary_path}")

        # Parser configuration
        self.max_content_preview = self.config.get("max_content_preview", 100)
        self.extract_front_matter = self.config.get("extract_front_matter", True)
        self.paragraph_threshold = self.config.get("paragraph_threshold", 1)
        self.max_elements = self.config.get("max_elements", 1000)
        self.extract_dates = self.config.get("extract_dates", True)
        self.extract_numbers = self.config.get("extract_numbers", True)
        self.enable_link_extraction = self.config.get("enable_link_extraction", True)
        self.strip_whitespace = self.config.get("strip_whitespace", True)

    def parse(self, content: Dict[str, Any]) -> Dict[str, Any]:
        """Parse markdown document using Go binary.

        Args:
            content: Dictionary containing document content and metadata

        Returns:
            Parsed document with elements and relationships
        """
        # Validate input
        doc_id = content.get("id", "")
        markdown_content = content.get("content", "")
        metadata = content.get("metadata", {})

        if not doc_id:
            raise ValueError("Document ID is required")
        if not markdown_content:
            raise ValueError("Content is required")

        # Prepare command arguments
        cmd_args = [str(self.binary_path), "-stdin", "-json"]

        # Add document ID
        cmd_args.extend(["-id", doc_id])

        # Add front matter flag if disabled
        if not self.extract_front_matter:
            cmd_args.append("-no-front-matter")

        # Add paragraph threshold if not default
        if self.paragraph_threshold != 1:
            cmd_args.extend(["-paragraph-threshold", str(self.paragraph_threshold)])

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
                input=markdown_content,
                capture_output=True,
                text=True,
                timeout=60  # 60 second timeout for large markdown files
            )

            if result.returncode != 0:
                error_msg = result.stderr or result.stdout
                raise RuntimeError(f"Go markdown parser failed: {error_msg}")

            # Parse JSON response
            response = json.loads(result.stdout)

            # Convert element types to Python ElementType enum values
            for element in response.get("elements", []):
                go_type = element.get("element_type", "")
                if go_type == "root":
                    element["element_type"] = ElementType.ROOT.value
                elif go_type == "header":
                    element["element_type"] = ElementType.HEADER.value
                elif go_type == "paragraph":
                    element["element_type"] = ElementType.PARAGRAPH.value
                elif go_type == "code_block":
                    element["element_type"] = ElementType.CODE_BLOCK.value
                elif go_type == "list":
                    element["element_type"] = ElementType.LIST.value
                elif go_type == "list_item":
                    element["element_type"] = ElementType.LIST_ITEM.value
                elif go_type == "blockquote":
                    element["element_type"] = ElementType.BLOCKQUOTE.value
                elif go_type == "table":
                    element["element_type"] = ElementType.TABLE.value
                elif go_type == "table_row":
                    element["element_type"] = ElementType.TABLE_ROW.value
                elif go_type == "table_cell":
                    element["element_type"] = ElementType.TABLE_CELL.value
                elif go_type == "front_matter":
                    # Use a generic element type since FRONT_MATTER doesn't exist in enum
                    element["element_type"] = "front_matter"
                else:
                    # Keep original type if not recognized
                    pass

            return response

        except subprocess.TimeoutExpired:
            raise RuntimeError("Markdown parsing timed out - file may be too large")
        except json.JSONDecodeError as e:
            raise RuntimeError(f"Failed to parse Go markdown parser output: {e}")
        except Exception as e:
            raise RuntimeError(f"Unexpected error during markdown parsing: {e}")

    def _resolve_element_content(self, element: Dict[str, Any]) -> str:
        """Resolve content for an element.

        Args:
            element: The element to resolve content for

        Returns:
            The resolved content string
        """
        # For markdown elements, content is usually in 'text' or 'content' field
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
        # For markdown elements, prioritize text field
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
        return location_type in [
            "header", "paragraph", "code_block", "list", "blockquote",
            "table", "front_matter", "markdown_position"
        ]