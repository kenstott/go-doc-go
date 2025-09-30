"""Go-based XML parser implementation."""

import json
import subprocess
import os
from pathlib import Path
from typing import Dict, Any, Optional

from src.go_doc_go.document_parser.base import DocumentParser
from src.go_doc_go.storage.element_element import ElementType


class GoXMLParser(DocumentParser):
    """XML parser that uses Go binary for processing."""

    def __init__(self, config: Optional[Dict] = None):
        """Initialize Go XML parser."""
        super().__init__(config)

        # Find the Go binary
        project_root = Path(__file__).parent.parent.parent.parent
        self.binary_path = project_root / "bin" / "xmlparser"

        if not self.binary_path.exists():
            raise RuntimeError(f"Go XML parser binary not found at {self.binary_path}")

        # Parser configuration
        self.max_content_preview = self.config.get("max_content_preview", 100)
        self.extract_attributes = self.config.get("extract_attributes", True)
        self.flatten_namespaces = self.config.get("flatten_namespaces", True)
        self.extract_namespaces = self.config.get("extract_namespaces", True)
        self.max_depth = self.config.get("max_depth", 20)
        self.enable_caching = self.config.get("enable_caching", True)

    def parse(self, content: Dict[str, Any]) -> Dict[str, Any]:
        """Parse XML document using Go binary.

        Args:
            content: Dictionary containing document content and metadata

        Returns:
            Parsed document with elements and relationships
        """
        # Validate input
        doc_id = content.get("id", "")
        xml_content = content.get("content", "")
        metadata = content.get("metadata", {})

        if not doc_id:
            raise ValueError("Document ID is required")
        if not xml_content:
            raise ValueError("Content is required")

        # Prepare command arguments
        cmd_args = [str(self.binary_path), "-stdin", "-json"]

        # Add document ID
        cmd_args.extend(["-id", doc_id])

        # Add configuration options
        cmd_args.extend(["-max-preview", str(self.max_content_preview)])
        cmd_args.extend(["-max-depth", str(self.max_depth)])

        # Add boolean flags
        if not self.extract_attributes:
            cmd_args.append("-extract-attrs=false")
        if not self.flatten_namespaces:
            cmd_args.append("-flatten-namespaces=false")
        if not self.extract_namespaces:
            cmd_args.append("-extract-namespaces=false")
        if not self.enable_caching:
            cmd_args.append("-enable-caching=false")

        try:
            # Convert bytes to string if necessary
            if isinstance(xml_content, bytes):
                xml_content = xml_content.decode('utf-8', errors='replace')

            # Call Go binary with JSON output
            result = subprocess.run(
                cmd_args,
                input=xml_content,
                capture_output=True,
                text=True,
                timeout=60  # 60 second timeout for large XML files
            )

            if result.returncode != 0:
                error_msg = result.stderr or result.stdout
                # Handle XML syntax errors gracefully - return valid structure with error info
                if "XML syntax error" in str(error_msg) or "invalid character entity" in str(error_msg):
                    return {
                        "document": {
                            "id": doc_id,
                            "doc_type": "xml",
                            "metadata": {
                                **metadata,
                                "error": f"XML parsing error: {error_msg}",
                                "parser_error": True,
                                "malformed_xml": True
                            }
                        },
                        "elements": [],
                        "relationships": [],
                        "links": []
                    }
                raise RuntimeError(f"Go XML parser failed: {error_msg}")

            # Parse JSON response
            response = json.loads(result.stdout)

            # Convert to Python format - the Go parser already uses the right structure
            # Just need to convert element types to Python ElementType enum values
            for element in response.get("elements", []):
                go_type = element.get("element_type", "")
                if go_type == "document_root":
                    element["element_type"] = ElementType.ROOT.value
                elif go_type == "xml_element":
                    element["element_type"] = "xml_element"  # Custom type
                elif go_type == "xml_text":
                    element["element_type"] = "xml_text"  # Custom type
                elif go_type == "xml_list":
                    element["element_type"] = ElementType.LIST.value
                elif go_type == "xml_object":
                    element["element_type"] = "xml_object"  # Custom type
                else:
                    # Keep original type if not recognized
                    pass

            # Convert Go response format to Python format
            python_response = {
                "document": response.get("document", {}),
                "elements": response.get("elements", []),
                "relationships": response.get("relationships", []),
                "links": response.get("links", []),
            }

            # Convert link format from Go to Python
            converted_links = []
            for link in python_response.get("links", []):
                converted_link = {
                    "link_id": f"link_{hash(link.get('source_id', '') + link.get('link_target', '')) % 1000000}",
                    "source_element_id": link.get("source_id", ""),
                    "link_type": link.get("link_type", ""),
                    "link_target": link.get("link_target", ""),
                    "context": link.get("link_text", ""),
                }
                converted_links.append(converted_link)
            python_response["links"] = converted_links

            return python_response

        except subprocess.TimeoutExpired:
            raise RuntimeError("XML parsing timed out - file may be too large")
        except json.JSONDecodeError as e:
            raise RuntimeError(f"Failed to parse Go XML parser output: {e}")
        except Exception as e:
            raise RuntimeError(f"Unexpected error during XML parsing: {e}")

    def _resolve_element_content(self, element: Dict[str, Any]) -> str:
        """Resolve content for an element.

        Args:
            element: The element to resolve content for

        Returns:
            The resolved content string
        """
        # For XML elements, content might be in 'text' or 'content' field
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
        # For XML elements, prioritize text field
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
            "xml_element", "xml_text", "xml_path", "xml_namespace",
            "xml_attributes", "xpath"
        ]
