"""
Go-based XLSX parser wrapper for Python.

This module provides a Python interface to the Go XLSX parser implementation.
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


class GoXLSXParser(DocumentParser):
    """XLSX parser using Go implementation."""

    def __init__(self, config: Optional[Dict[str, Any]] = None):
        """Initialize Go XLSX parser with configuration."""
        super().__init__(config)

        # Determine the path to the Go XLSX parser binary
        self.binary_path = self._find_binary()
        if not self.binary_path:
            raise RuntimeError("Go XLSX parser binary not found. Please build the Go parser first.")

        # General configuration from base class
        self.max_content_preview = self.config.get("max_content_preview", 100)

        # XLSX-specific configuration
        self.max_rows = self.config.get("max_rows", 10000)
        self.max_cols = self.config.get("max_cols", 100)
        self.detect_tables = self.config.get("detect_tables", True)
        self.min_table_rows = self.config.get("min_table_rows", 2)
        self.min_table_cols = self.config.get("min_table_cols", 2)
        self.extract_comments = self.config.get("extract_comments", True)
        self.extract_formulas = self.config.get("extract_formulas", True)
        self.extract_links = self.config.get("extract_links", True)

    def _find_binary(self) -> Optional[str]:
        """Find the Go XLSX parser binary."""
        # Look for the binary in common locations
        possible_paths = [
            Path(__file__).parent.parent.parent.parent / "go" / "bin" / "xlsxparser",
            Path(__file__).parent.parent.parent.parent / "bin" / "xlsxparser",
            Path.cwd() / "go" / "bin" / "xlsxparser",
            Path.cwd() / "bin" / "xlsxparser",
        ]

        for path in possible_paths:
            if path.exists() and path.is_file():
                logger.debug(f"Found XLSX parser binary at: {path}")
                return str(path)

        # Check if it's in the system PATH
        try:
            result = subprocess.run(
                ["which", "xlsxparser"],
                capture_output=True,
                text=True,
                check=False
            )
            if result.returncode == 0:
                path = result.stdout.strip()
                logger.debug(f"Found XLSX parser binary in PATH: {path}")
                return path
        except Exception:
            pass

        logger.warning("XLSX parser binary not found")
        return None

    def parse(self, content: Dict[str, Any]) -> Dict[str, Any]:
        """
        Parse XLSX document using Go implementation.

        Args:
            content: Dictionary containing:
                - id: Document identifier
                - content: XLSX content (bytes or file path)
                - metadata: Optional metadata dict

        Returns:
            Parsed document structure with elements and relationships
        """
        doc_id = content.get("id", "unknown")
        xlsx_content = content.get("content", "")
        metadata = content.get("metadata", {})

        # Prepare command arguments
        cmd = [
            self.binary_path,
            "-json",  # Output in JSON format
            "-id", doc_id,
            "-max-rows", str(self.max_rows),
            "-max-cols", str(self.max_cols),
            "-max-preview", str(self.max_content_preview),
            "-min-table-rows", str(self.min_table_rows),
            "-min-table-cols", str(self.min_table_cols),
        ]

        # Add boolean flags
        if self.detect_tables:
            cmd.extend(["-detect-tables=true"])
        else:
            cmd.extend(["-detect-tables=false"])

        if self.extract_comments:
            cmd.extend(["-extract-comments=true"])
        else:
            cmd.extend(["-extract-comments=false"])

        if self.extract_formulas:
            cmd.extend(["-extract-formulas=true"])
        else:
            cmd.extend(["-extract-formulas=false"])

        if self.extract_links:
            cmd.extend(["-extract-links=true"])
        else:
            cmd.extend(["-extract-links=false"])

        try:
            # Check if content is a file path or raw bytes
            if isinstance(xlsx_content, str) and os.path.isfile(xlsx_content):
                # File path provided
                cmd.extend(["-input", xlsx_content])
                result = subprocess.run(
                    cmd,
                    capture_output=True,
                    text=True,
                    check=True
                )
            elif isinstance(xlsx_content, (bytes, bytearray)):
                # Raw XLSX bytes - use stdin mode
                cmd.append("-stdin")
                result = subprocess.run(
                    cmd,
                    input=xlsx_content,
                    capture_output=True,
                    check=True
                )
                # Decode output if we got bytes
                if isinstance(result.stdout, bytes):
                    result.stdout = result.stdout.decode('utf-8')
            elif isinstance(xlsx_content, str):
                # String content (might be base64 or raw text representation)
                cmd.append("-stdin")
                result = subprocess.run(
                    cmd,
                    input=xlsx_content.encode('utf-8'),
                    capture_output=True,
                    check=True
                )
                if isinstance(result.stdout, bytes):
                    result.stdout = result.stdout.decode('utf-8')
            else:
                raise ValueError(f"Unsupported content type: {type(xlsx_content)}")

            # Parse JSON output
            parsed_result = json.loads(result.stdout)

            # Add any Python-side metadata
            if metadata:
                if "metadata" not in parsed_result:
                    parsed_result["metadata"] = {}
                parsed_result["metadata"].update(metadata)

            # Ensure all required fields are present
            if "document" not in parsed_result:
                parsed_result["document"] = {
                    "id": doc_id,
                    "doc_type": "xlsx",
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
            logger.error(f"XLSX parser failed: {error_msg}")
            # Return minimal valid structure on error
            return {
                "document": {
                    "id": doc_id,
                    "doc_type": "xlsx",
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
            logger.error(f"Failed to parse XLSX parser output: {e}")
            return {
                "document": {
                    "id": doc_id,
                    "doc_type": "xlsx",
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
            logger.error(f"Unexpected error in XLSX parsing: {e}")
            return {
                "document": {
                    "id": doc_id,
                    "doc_type": "xlsx",
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
            True if content appears to be an XLSX document
        """
        # Check if binary is available
        if not self.binary_path:
            return False

        # Check file extension if filename is provided
        metadata = content.get("metadata", {})
        filename = metadata.get("filename", "")
        if filename.lower().endswith((".xlsx", ".xlsm", ".xltx", ".xltm")):
            return True

        # Check content type
        content_type = metadata.get("content_type", "") or ""
        if content_type and any(xlsx_type in content_type.lower() for xlsx_type in [
            "excel", "xlsx", "spreadsheet",
            "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
        ]):
            return True

        # Check for XLSX magic bytes (ZIP signature + Excel files)
        raw_content = content.get("content", "")
        if isinstance(raw_content, bytes):
            # XLSX files are ZIP archives starting with PK
            if raw_content.startswith(b"PK"):
                # Could be XLSX - the binary parser will validate further
                return True
        elif isinstance(raw_content, str):
            # Check if it's a file path to an XLSX
            if os.path.isfile(raw_content):
                try:
                    with open(raw_content, "rb") as f:
                        header = f.read(2)
                        if header == b"PK":
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
        # For XLSX elements, content and text are typically the same
        return self._resolve_element_content(element)

    def supports_location(self) -> bool:
        """
        Check if this parser supports location extraction.

        Returns:
            True if location extraction is supported
        """
        # XLSX parser supports cell/sheet-based location
        return True