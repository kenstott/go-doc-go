"""
Go-based PDF parser wrapper for Python.

This module provides a Python interface to the Go PDF parser implementation.
It uses subprocess communication to interact with the compiled Go binary.
"""

import json
import logging
import os
import subprocess
from pathlib import Path
from typing import Dict, Any, Optional

from .base import DocumentParser

logger = logging.getLogger(__name__)


class GoPDFParser(DocumentParser):
    """PDF parser using Go implementation."""

    def __init__(self, config: Optional[Dict[str, Any]] = None):
        """Initialize Go PDF parser with configuration."""
        super().__init__(config)

        # Determine the path to the Go PDF parser binary
        self.binary_path = self._find_binary()
        if not self.binary_path:
            raise RuntimeError("Go PDF parser binary not found. Please build the Go parser first.")
        logger.info(f"Using PDF parser binary: {self.binary_path}")

        # General configuration from base class
        self.max_content_preview = self.config.get("max_content_preview", 100)

        # PDF-specific configuration
        self.max_pages = self.config.get("max_pages", 1000)
        self.extract_links = self.config.get("extract_links", True)

    def _find_binary(self) -> Optional[str]:
        """Find the Go PDF parser binary."""
        # Look for the binary in standard location
        possible_paths = [
            Path(__file__).parent.parent.parent.parent / "bin" / "pdfparser",
            Path.cwd() / "bin" / "pdfparser",
        ]

        for path in possible_paths:
            if path.exists() and path.is_file():
                logger.debug(f"Found PDF parser binary at: {path}")
                return str(path)

        # Check if it's in the system PATH
        try:
            result = subprocess.run(
                ["which", "pdfparser"],
                capture_output=True,
                text=True,
                check=False
            )
            if result.returncode == 0:
                path = result.stdout.strip()
                logger.debug(f"Found PDF parser binary in PATH: {path}")
                return path
        except Exception:
            pass

        logger.warning("PDF parser binary not found")
        return None

    def parse(self, content: Dict[str, Any]) -> Dict[str, Any]:
        """
        Parse PDF document using Go implementation.

        Args:
            content: Dictionary containing:
                - id: Document identifier
                - content: PDF content (bytes or file path)
                - metadata: Optional metadata dict

        Returns:
            Parsed document structure with elements and relationships
        """
        doc_id = content.get("id", "unknown")
        pdf_content = content.get("content", "")
        metadata = content.get("metadata", {})

        # If content is empty but binary_path is provided, use that
        if not pdf_content and "binary_path" in content:
            pdf_content = content["binary_path"]
            logger.debug(f"Using binary_path for content: {pdf_content}")

        # Prepare command arguments
        cmd = [
            self.binary_path,
            "-json",  # Output in JSON format
            "-id", doc_id,
            "-max-pages", str(self.max_pages),
            "-max-preview", str(self.max_content_preview),
        ]

        # Add boolean flags
        if self.extract_links:
            cmd.extend(["-extract-links=true"])
        else:
            cmd.extend(["-extract-links=false"])

        try:
            # Check if content is a file path or raw bytes
            if isinstance(pdf_content, str) and os.path.isfile(pdf_content):
                # File path provided
                cmd.extend(["-input", pdf_content])
                logger.info(f"Calling PDF parser with command: {' '.join(cmd)}")
                result = subprocess.run(
                    cmd,
                    capture_output=True,
                    text=True,
                    check=True
                )
                logger.info(f"PDF parser stdout length: {len(result.stdout)} bytes")
                logger.debug(f"PDF parser stdout preview: {result.stdout[:500]}")
            elif isinstance(pdf_content, (bytes, bytearray)):
                # Raw PDF bytes - use stdin mode
                cmd.append("-stdin")
                result = subprocess.run(
                    cmd,
                    input=pdf_content,
                    capture_output=True,
                    check=True
                )
                # Decode output if we got bytes
                if isinstance(result.stdout, bytes):
                    result.stdout = result.stdout.decode('utf-8')
            elif isinstance(pdf_content, str):
                # String content (might be base64 or raw text representation)
                cmd.append("-stdin")
                result = subprocess.run(
                    cmd,
                    input=pdf_content.encode('utf-8'),
                    capture_output=True,
                    check=True
                )
                if isinstance(result.stdout, bytes):
                    result.stdout = result.stdout.decode('utf-8')
            else:
                raise ValueError(f"Unsupported content type: {type(pdf_content)}")

            # Parse JSON output
            parsed_result = json.loads(result.stdout)

            # Check if parsing returned None or invalid data
            if parsed_result is None:
                raise ValueError(f"Go parser returned null result. Stdout was: '{result.stdout[:200]}'")

            # Add any Python-side metadata
            if metadata:
                if "metadata" not in parsed_result:
                    parsed_result["metadata"] = {}
                parsed_result["metadata"].update(metadata)

            # Ensure all required fields are present
            if "document" not in parsed_result:
                parsed_result["document"] = {
                    "id": doc_id,
                    "doc_type": "pdf",
                    "metadata": metadata
                }

            if "elements" not in parsed_result:
                parsed_result["elements"] = []

            if "relationships" not in parsed_result:
                parsed_result["relationships"] = []

            return parsed_result

        except subprocess.CalledProcessError as e:
            error_msg = str(e)
            if hasattr(e, 'stderr') and e.stderr:
                error_msg = e.stderr if isinstance(e.stderr, str) else e.stderr.decode('utf-8', errors='ignore')
            logger.error(f"PDF parser failed: {error_msg}", exc_info=True)
            # Return minimal valid structure on error
            return {
                "document": {
                    "id": doc_id,
                    "doc_type": "pdf",
                    "metadata": {
                        **metadata,
                        "error": str(e),
                        "parser_error": True
                    }
                },
                "elements": [],
                "relationships": []
            }
        except json.JSONDecodeError as e:
            logger.error(f"Failed to parse PDF parser output: {e}", exc_info=True)
            return {
                "document": {
                    "id": doc_id,
                    "doc_type": "pdf",
                    "metadata": {
                        **metadata,
                        "error": f"JSON decode error: {e}",
                        "parser_error": True
                    }
                },
                "elements": [],
                "relationships": []
            }
        except Exception as e:
            logger.error(f"Unexpected error in PDF parsing: {e}", exc_info=True)
            return {
                "document": {
                    "id": doc_id,
                    "doc_type": "pdf",
                    "metadata": {
                        **metadata,
                        "error": str(e),
                        "parser_error": True
                    }
                },
                "elements": [],
                "relationships": []
            }

    def can_parse(self, content: Dict[str, Any]) -> bool:
        """
        Check if this parser can handle the given content.

        Args:
            content: Document content dictionary

        Returns:
            True if content appears to be a PDF document
        """
        # Check if binary is available
        if not self.binary_path:
            return False

        # Check file extension if filename is provided
        metadata = content.get("metadata", {})
        filename = metadata.get("filename", "")
        if filename.lower().endswith(".pdf"):
            return True

        # Check content type
        content_type = metadata.get("content_type", "")
        if content_type and "pdf" in content_type.lower():
            return True

        # Check for PDF magic bytes
        raw_content = content.get("content", "")
        if isinstance(raw_content, bytes):
            if raw_content.startswith(b"%PDF"):
                return True
        elif isinstance(raw_content, str):
            if raw_content.startswith("%PDF"):
                return True
            # Check if it's a file path to a PDF
            if os.path.isfile(raw_content):
                try:
                    with open(raw_content, "rb") as f:
                        header = f.read(4)
                        if header == b"%PDF":
                            return True
                except Exception:
                    pass

        return False

    def _resolve_element_content(self, element: Dict[str, Any]) -> str:
        """
        Resolve the content of an element.

        Args:
            element: Element dictionary

        Returns:
            Element content as string
        """
        return element.get("content", element.get("content_preview", ""))

    def _resolve_element_text(self, element: Dict[str, Any]) -> str:
        """
        Resolve the text content of an element.

        Args:
            element: Element dictionary

        Returns:
            Element text content
        """
        # For PDF elements, content and text are typically the same
        return self._resolve_element_content(element)

    def supports_location(self) -> bool:
        """
        Check if this parser supports location extraction.

        Returns:
            True if location extraction is supported
        """
        # PDF parser supports page-based location
        return True