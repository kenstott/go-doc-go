#!/usr/bin/env python3
"""
Parser shim for Go worker.
Provides a bridge to use Python parsers (PDF) from Go.
"""

import sys
import warnings

# Suppress all warnings FIRST, before any other imports
warnings.filterwarnings('ignore')

# Disable all logging to stdout - must be done before importing anything
import logging
logging.disable(logging.CRITICAL)

import json
import base64
from typing import Dict, Any

from go_doc_go.document_parser.pdf import PdfParser

logger = logging.getLogger(__name__)


def parse_pdf(doc_content: Dict[str, Any]) -> Dict[str, Any]:
    """Parse PDF document using Python parser."""
    parser = PdfParser()
    result = parser.parse(doc_content)
    return result


def main():
    """Main entry point for parser shim."""
    try:
        # Read input from stdin
        input_json = sys.stdin.read()
        if not input_json:
            result = {
                "success": False,
                "error": "No input data provided on stdin"
            }
            print(json.dumps(result))
            sys.exit(1)

        # Parse input JSON
        input_data = json.loads(input_json)

        parser_type = input_data.get("parser_type")
        doc_id = input_data.get("doc_id")
        content_base64 = input_data.get("content")  # Base64 encoded binary content

        if not parser_type:
            raise ValueError("Missing parser_type")
        if not doc_id:
            raise ValueError("Missing doc_id")
        if not content_base64:
            raise ValueError("Missing content")

        # Decode binary content
        content = base64.b64decode(content_base64)

        # Create document content structure
        doc_content = {
            "id": doc_id,
            "content": content,
            "metadata": input_data.get("metadata", {})
        }

        # Parse based on type
        if parser_type == "pdf":
            parse_result = parse_pdf(doc_content)
        else:
            raise ValueError(f"Unsupported parser type: {parser_type}")

        # Return success with parse result
        result = {
            "success": True,
            "result": parse_result
        }
        print(json.dumps(result))
        sys.exit(0)

    except Exception as e:
        logger.error(f"Parser shim error: {e}", exc_info=True)
        result = {
            "success": False,
            "error": str(e)
        }
        print(json.dumps(result))
        sys.exit(1)


if __name__ == "__main__":
    main()
