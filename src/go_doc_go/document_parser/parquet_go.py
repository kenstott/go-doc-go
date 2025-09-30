"""
Python wrapper for the Go Parquet parser implementation.

This module provides a Python interface to the high-performance Go Parquet parser,
enabling seamless integration with existing Python codebases while leveraging
Go's superior performance for tabular data processing.
"""

import json
import logging
import os
import subprocess
import tempfile
from pathlib import Path
from typing import Dict, Any, Optional, List

from .base import DocumentParser

logger = logging.getLogger(__name__)


class ParquetGoParser(DocumentParser):
    """Python wrapper for the Go Parquet parser implementation."""

    def __init__(self, config: Optional[Dict[str, Any]] = None):
        """Initialize the Go Parquet parser wrapper."""
        super().__init__(config)
        self.config = config or {}

        # Generic configuration for tabular data
        self.text_column = self.config.get('text_column', '')  # Auto-detect if empty
        self.group_by_column = self.config.get('group_by_column', '')  # Optional grouping
        self.max_content_preview = self.config.get('max_content_preview', 100)
        # Configurable list of metadata columns to extract from first row
        self.metadata_columns = self.config.get('metadata_columns', [])

        # Go binary path
        self.go_binary_path = self._find_go_binary()
        if not self.go_binary_path:
            raise RuntimeError("Go Parquet parser binary not found. Please ensure it's built and in PATH or bin/ directory.")

    def _find_go_binary(self) -> Optional[str]:
        """Find the Go Parquet parser binary."""
        # Check relative path first (for development)
        current_dir = Path(__file__).parent
        relative_binary = current_dir.parent.parent.parent / "bin" / "parquetparser"

        if relative_binary.exists() and os.access(relative_binary, os.X_OK):
            return str(relative_binary)

        # Check if it's in PATH
        try:
            result = subprocess.run(["which", "parquetparser"], capture_output=True, text=True)
            if result.returncode == 0:
                return result.stdout.strip()
        except (subprocess.SubprocessError, FileNotFoundError):
            pass

        return None

    def parse(self, doc_content: Dict[str, Any]) -> Dict[str, Any]:
        """
        Parse a Parquet document using the Go parser.

        Args:
            doc_content: Document content and metadata. Expected format:
                {
                    "id": "document_id",
                    "binary_path": "/path/to/file.parquet",
                    "metadata": {...}
                }

        Returns:
            Dictionary containing parsed document structure with elements,
            relationships, and metadata.
        """
        # Extract parameters
        doc_id = doc_content.get("id", "unknown")
        binary_path = doc_content.get("binary_path")
        metadata = doc_content.get("metadata", {})

        if not binary_path:
            raise ValueError("Parquet parser requires 'binary_path' in content")

        # Validate file exists
        if not os.path.isfile(binary_path):
            raise FileNotFoundError(f"Parquet file not found: {binary_path}")

        # Prepare command arguments
        cmd = [
            self.go_binary_path,
            "-file", binary_path,
            "-max-preview", str(self.max_content_preview),
        ]

        # Add optional parameters
        if self.text_column:
            cmd.extend(["-text-column", self.text_column])

        if self.group_by_column:
            cmd.extend(["-group-by", self.group_by_column])

        if self.metadata_columns:
            cmd.extend(["-metadata-columns", ",".join(self.metadata_columns)])

        # Execute Go parser
        logger.debug(f"Executing Go Parquet parser: {' '.join(cmd)}")

        try:
            result = subprocess.run(
                cmd,
                capture_output=True,
                text=True,
                timeout=30,  # 30 second timeout
                check=True
            )
        except subprocess.TimeoutExpired:
            raise RuntimeError("Go Parquet parser timed out after 30 seconds")
        except subprocess.CalledProcessError as e:
            error_msg = e.stderr.strip() if e.stderr else "Unknown error"
            raise RuntimeError(f"Go Parquet parser failed: {error_msg}")

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
                    "text_column": self.text_column,
                    "group_by_column": self.group_by_column,
                    "metadata_columns": self.metadata_columns,
                }
            })

        # Ensure proper document ID
        if "document" in parsed_result:
            parsed_result["document"]["doc_id"] = doc_id

        logger.debug(f"Successfully parsed Parquet with {len(parsed_result.get('elements', []))} elements")
        return parsed_result

    def supports_location(self, content_location: Dict[str, Any]) -> bool:
        """
        Check if this parser supports resolving the given location.

        Args:
            content_location: Content location pointer

        Returns:
            True if supported, False otherwise
        """
        return False  # Parquet parser doesn't support location-based content retrieval

    def _resolve_element_content(self, location_data: Dict[str, Any],
                                source_content: Optional[Any] = None) -> str:
        """
        Resolve content for specific Parquet element types.

        Args:
            location_data: Content location data
            source_content: Optional pre-loaded source content

        Returns:
            Resolved content string
        """
        # For Parquet, content is in the metadata
        return location_data.get('content_preview', '')

    def _resolve_element_text(self, location_data: Dict[str, Any],
                             source_content: Optional[Any] = None) -> str:
        """
        Resolve the plain text representation of a Parquet element.

        Args:
            location_data: Content location data
            source_content: Optional preloaded source content

        Returns:
            Plain text representation of the element
        """
        return self._resolve_element_content(location_data, source_content)

    def get_parser_info(self) -> Dict[str, Any]:
        """
        Get information about the parser implementation.

        Returns:
            Dictionary with parser information
        """
        return {
            "name": "ParquetGoParser",
            "version": "1.0.0",
            "backend": "Go",
            "binary_path": self.go_binary_path,
            "supported_formats": [".parquet"],
            "features": [
                "Tabular data parsing",
                "Configurable metadata extraction",
                "Optional row grouping",
                "Auto-detect text column",
                "All columns preserved as metadata",
                "High-performance Arrow library"
            ],
            "performance": "High (Go implementation with Apache Arrow)",
            "config": {
                "max_content_preview": self.max_content_preview,
                "text_column": self.text_column,
                "group_by_column": self.group_by_column,
                "metadata_columns": self.metadata_columns,
            }
        }