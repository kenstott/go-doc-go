#!/usr/bin/env python3
"""
Analytics storage shim for Go worker.
Provides a bridge between Go and Python analytics storage.
"""

import sys
import warnings

# Suppress all warnings FIRST, before any other imports
warnings.filterwarnings('ignore')

# Disable all logging to stdout - must be done before importing anything
import logging
logging.disable(logging.CRITICAL)

import json
from typing import Dict, Any, List
from datetime import datetime

from go_doc_go.storage_adapters.factory import StorageFactory

logger = logging.getLogger(__name__)


def append_documents(storage, data: List[Dict[str, Any]]) -> None:
    """Append documents to analytics storage."""
    # Convert datetime strings back to datetime objects
    for doc in data:
        if "processed_at" in doc and isinstance(doc["processed_at"], str):
            doc["processed_at"] = datetime.fromisoformat(doc["processed_at"].replace("Z", "+00:00"))

    storage.append_documents(data)


def append_elements(storage, data: List[Dict[str, Any]]) -> None:
    """Append elements to analytics storage."""
    storage.append_elements(data)


def append_relationships(storage, data: List[Dict[str, Any]]) -> None:
    """Append relationships to analytics storage."""
    storage.append_relationships(data)


def append_embeddings(storage, data: List[Dict[str, Any]]) -> None:
    """Append embeddings to analytics storage."""
    storage.append_embeddings(data)


def main():
    """Main entry point for analytics shim."""
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
        config = input_data.get("config", {})
        data = input_data.get("data", [])

        if not operation:
            raise ValueError("Missing operation")

        # Create analytics storage
        storage = StorageFactory.create_analytics_storage(config)

        # Execute operation
        if operation == "append_documents":
            append_documents(storage, data)
        elif operation == "append_elements":
            append_elements(storage, data)
        elif operation == "append_relationships":
            append_relationships(storage, data)
        elif operation == "append_embeddings":
            append_embeddings(storage, data)
        else:
            raise ValueError(f"Unknown operation: {operation}")

        # Return success
        result = {
            "success": True
        }
        print(json.dumps(result))
        sys.exit(0)

    except Exception as e:
        logger.error(f"Analytics shim error: {e}", exc_info=True)
        result = {
            "success": False,
            "error": str(e)
        }
        print(json.dumps(result))
        sys.exit(1)


if __name__ == "__main__":
    main()
