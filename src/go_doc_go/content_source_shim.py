#!/usr/bin/env python3
"""
Content source shim for Go worker.
Provides a bridge between Go and Python content sources.
"""

import sys
import warnings

# Suppress all warnings FIRST, before any other imports
warnings.filterwarnings('ignore')

# Disable all logging to stdout - must be done before importing anything
import logging
logging.disable(logging.CRITICAL)

import json
from typing import Dict, Any

from go_doc_go.content_source.factory import get_content_source

logger = logging.getLogger(__name__)


def fetch_document(source, params: Dict[str, Any]) -> Dict[str, Any]:
    """Fetch a document from the content source."""
    source_id = params.get("source_id")
    if not source_id:
        raise ValueError("Missing source_id parameter")

    doc_content = source.fetch_document(source_id)
    return doc_content


def list_documents(source, params: Dict[str, Any]) -> list:
    """List documents from the content source."""
    documents = source.list_documents()

    # Convert to simple format
    result = []
    for doc in documents:
        result.append({
            "id": doc.get("id") or doc.get("source_id"),
            "metadata": doc.get("metadata", {})
        })

    return result


def has_changed(source, params: Dict[str, Any]) -> bool:
    """Check if a document has changed."""
    source_id = params.get("source_id")
    last_modified = params.get("last_modified")

    if not source_id:
        raise ValueError("Missing source_id parameter")

    return source.has_changed(source_id, last_modified)


def main():
    """Main entry point for content source shim."""
    if len(sys.argv) < 2:
        result = {
            "success": False,
            "error": "No input data provided"
        }
        print(json.dumps(result))
        sys.exit(1)

    try:
        # Parse input JSON
        input_data = json.loads(sys.argv[1])

        operation = input_data.get("operation")
        source_name = input_data.get("source_name")
        config = input_data.get("config", {})
        params = input_data.get("params") or {}

        if not operation:
            raise ValueError("Missing operation")
        if not source_name:
            raise ValueError("Missing source_name")

        # Create content source
        source = get_content_source(config)

        # Execute operation
        if operation == "fetch_document":
            data = fetch_document(source, params)
        elif operation == "list_documents":
            data = list_documents(source, params)
        elif operation == "has_changed":
            data = has_changed(source, params)
        else:
            raise ValueError(f"Unknown operation: {operation}")

        # Return success
        result = {
            "success": True,
            "data": data
        }
        print(json.dumps(result))
        sys.exit(0)

    except Exception as e:
        logger.error(f"Content source shim error: {e}", exc_info=True)
        result = {
            "success": False,
            "error": str(e)
        }
        print(json.dumps(result))
        sys.exit(1)


if __name__ == "__main__":
    main()
