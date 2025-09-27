"""
Go-based HTML parser wrapper.

This module provides a Python interface to the Go HTML parser implementation
via subprocess calls. It maintains compatibility with the existing HTML parser
interface while leveraging Go's performance for HTML parsing.
"""

import json
import subprocess
import tempfile
import os
from typing import Dict, Any, Optional

from .base import DocumentParser


class GoHTMLParser(DocumentParser):
    """
    Python wrapper for the Go HTML parser implementation.

    This class provides a drop-in replacement for the Python HTML parser
    by calling the Go binary via subprocess and parsing the JSON response.
    """

    def __init__(self, config: Optional[Dict[str, Any]] = None):
        super().__init__(config)
        self.go_binary_path = self._find_go_binary()

    def _find_go_binary(self) -> str:
        """
        Locate the Go HTML parser binary.

        Returns:
            Path to the htmlparser binary

        Raises:
            FileNotFoundError: If the binary cannot be found
        """
        # Look for binary in various locations
        possible_paths = [
            # Relative to the project root
            os.path.join(os.path.dirname(__file__), "..", "..", "..", "go", "bin", "htmlparser"),
            # In system PATH
            "htmlparser",
            # Development location
            os.path.join(os.path.dirname(__file__), "..", "..", "..", "go", "cmd", "htmlparser", "htmlparser"),
        ]

        for path in possible_paths:
            if os.path.isfile(path) and os.access(path, os.X_OK):
                return os.path.abspath(path)

        # Try to build it if source exists
        go_dir = os.path.join(os.path.dirname(__file__), "..", "..", "..", "go")
        if os.path.isdir(go_dir):
            try:
                build_result = subprocess.run(
                    ["go", "build", "-o", "bin/htmlparser", "./cmd/htmlparser"],
                    cwd=go_dir,
                    capture_output=True,
                    text=True,
                    timeout=30
                )
                if build_result.returncode == 0:
                    binary_path = os.path.join(go_dir, "bin", "htmlparser")
                    if os.path.isfile(binary_path):
                        return os.path.abspath(binary_path)
            except (subprocess.TimeoutExpired, subprocess.SubprocessError, FileNotFoundError):
                pass

        raise FileNotFoundError(
            "Go HTML parser binary not found. Please build it with: "
            "cd go && go build -o bin/htmlparser ./cmd/htmlparser"
        )

    def parse(self, content: Dict[str, Any]) -> Dict[str, Any]:
        """
        Parse HTML content using the Go implementation.

        Args:
            content: Dictionary containing 'id', 'content', and optional 'metadata'

        Returns:
            Dictionary with parsed elements, relationships, and links

        Raises:
            ValueError: If required fields are missing
            RuntimeError: If Go parser fails
        """
        # Validate input
        if not isinstance(content, dict):
            raise ValueError("Content must be a dictionary")

        if "id" not in content:
            raise ValueError("Content must contain 'id' field")

        if "content" not in content:
            raise ValueError("Content must contain 'content' field")

        # Prepare configuration
        config = {
            "max_content_preview": self.config.get("max_content_preview", 100),
            "extract_dates": self.config.get("extract_dates", True),
            "enable_caching": self.config.get("enable_caching", True),
            "metadata": content.get("metadata", {})
        }

        # Create temporary files for input and config
        with tempfile.NamedTemporaryFile(mode='w', suffix='.json', delete=False) as config_file:
            json.dump(config, config_file)
            config_file_path = config_file.name

        try:
            # Prepare command arguments
            cmd = [
                self.go_binary_path,
                "-id", content["id"],
                "-config", config_file_path
            ]

            # Add optional flags
            if "max_content_preview" in self.config:
                cmd.extend(["-max-preview", str(self.config["max_content_preview"])])

            if "extract_dates" in self.config:
                cmd.extend(["-extract-dates", str(self.config["extract_dates"]).lower()])

            if "enable_caching" in self.config:
                cmd.extend(["-enable-caching", str(self.config["enable_caching"]).lower()])

            # Execute Go parser
            process = subprocess.run(
                cmd,
                input=content["content"],
                text=True,
                capture_output=True,
                timeout=60  # 60 second timeout
            )

            if process.returncode != 0:
                error_msg = f"Go HTML parser failed with exit code {process.returncode}"
                if process.stderr:
                    error_msg += f": {process.stderr}"
                raise RuntimeError(error_msg)

            # Parse JSON response
            try:
                result = json.loads(process.stdout)
            except json.JSONDecodeError as e:
                raise RuntimeError(f"Failed to parse Go parser output as JSON: {e}")

            return result

        finally:
            # Clean up temporary config file
            try:
                os.unlink(config_file_path)
            except OSError:
                pass

    def _normalize_response(self, go_response: Dict[str, Any]) -> Dict[str, Any]:
        """
        Normalize Go parser response to match Python parser format.

        The Go parser response should already be in the correct format,
        but this method can be used for any necessary transformations.

        Args:
            go_response: Raw response from Go parser

        Returns:
            Normalized response
        """
        # The Go implementation already produces the correct format
        # This method is here for future compatibility if needed
        return go_response

    def get_supported_extensions(self) -> list:
        """
        Get list of supported file extensions.

        Returns:
            List of supported extensions
        """
        return [".html", ".htm", ".xhtml"]

    def get_parser_info(self) -> Dict[str, Any]:
        """
        Get information about the parser implementation.

        Returns:
            Dictionary with parser information
        """
        return {
            "name": "GoHTMLParser",
            "version": "1.0.0",
            "implementation": "Go subprocess",
            "binary_path": self.go_binary_path,
            "supported_extensions": self.get_supported_extensions()
        }

    def _resolve_element_content(self, location_data: Dict[str, Any],
                                 source_content: Optional[str] = None) -> str:
        """
        Resolve element content from location data.

        Note: The Go parser provides content directly in elements,
        so this method is mainly for compatibility with the base class.

        Args:
            location_data: Content location information
            source_content: Optional source HTML content

        Returns:
            Resolved content string
        """
        # For Go parser, content is already resolved during parsing
        # This method is for compatibility with the base class
        if source_content and "selector" in location_data:
            # Could implement CSS selector resolution here if needed
            # For now, return empty string as content is already in elements
            return ""
        return ""

    def _resolve_element_text(self, location_data: Dict[str, Any],
                              source_content: Optional[str] = None) -> str:
        """
        Resolve element text content from location data.

        Note: The Go parser provides text directly in elements,
        so this method is mainly for compatibility with the base class.

        Args:
            location_data: Content location information
            source_content: Optional source HTML content

        Returns:
            Resolved text string
        """
        # For Go parser, text content is already resolved during parsing
        # This method is for compatibility with the base class
        if source_content and "selector" in location_data:
            # Could implement CSS selector text extraction here if needed
            # For now, return empty string as text is already in elements
            return ""
        return ""

    def supports_location(self, content_location: str) -> bool:
        """
        Check if this parser supports the given content location format.

        Args:
            content_location: Content location string or format identifier

        Returns:
            True if the parser supports this location format
        """
        # Go HTML parser generates CSS selectors and standard locations
        supported_types = ["css_selector", "xpath", "html", "dom"]
        return any(loc_type in content_location.lower() for loc_type in supported_types)


def create_go_html_parser(config: Optional[Dict[str, Any]] = None) -> GoHTMLParser:
    """
    Factory function to create a Go HTML parser instance.

    Args:
        config: Optional configuration dictionary

    Returns:
        GoHTMLParser instance
    """
    return GoHTMLParser(config)