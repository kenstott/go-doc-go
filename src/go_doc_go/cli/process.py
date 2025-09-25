#!/usr/bin/env python3
"""
Command-line interface for processing documents based on config.yaml configuration.
Simple, direct processing without pipeline database complexity.
"""

import argparse
import logging
import os
import sys
import time
from datetime import datetime
from pathlib import Path
from typing import Dict, Any, Optional, List

# Add the src directory to Python path
sys.path.insert(0, str(Path(__file__).parent.parent.parent))

from go_doc_go.config import Config


def format_timestamp() -> str:
    """Format current timestamp for display."""
    return datetime.now().strftime("%Y-%m-%d %H:%M:%S")


class DocumentProcessor:
    """Simple document processor that reads config and processes content sources."""

    def __init__(self, config: Config):
        """Initialize document processor."""
        self.config = config
        self.logger = logging.getLogger(__name__)

    def process_all_sources(self, sources_filter: Optional[List[str]] = None) -> Dict[str, Any]:
        """Process all configured content sources."""
        content_sources = self.config.get_content_sources()

        if sources_filter:
            content_sources = [s for s in content_sources if s.get('name') in sources_filter]

        if not content_sources:
            if sources_filter:
                self.logger.warning(f"No content sources found matching filter: {sources_filter}")
            else:
                self.logger.warning("No content sources configured")
            return {"sources_processed": 0, "total_documents": 0, "errors": []}

        results = {
            "sources_processed": 0,
            "total_documents": 0,
            "errors": [],
            "start_time": datetime.now(),
            "processing_config": self.config.get_processing_config()
        }

        self.logger.info(f"Starting processing of {len(content_sources)} content sources")

        for source in content_sources:
            source_name = source.get('name', 'unnamed')
            source_type = source.get('type', 'unknown')

            try:
                self.logger.info(f"Processing source: {source_name} ({source_type})")

                # Here we would integrate with the existing document processing logic
                # For now, this is a placeholder that demonstrates the structure
                source_result = self._process_single_source(source)

                results["sources_processed"] += 1
                results["total_documents"] += source_result.get("documents_processed", 0)

                self.logger.info(f"Completed source {source_name}: {source_result.get('documents_processed', 0)} documents")

            except Exception as e:
                error_msg = f"Failed to process source {source_name}: {str(e)}"
                self.logger.error(error_msg)
                results["errors"].append(error_msg)

        results["end_time"] = datetime.now()
        results["duration_seconds"] = (results["end_time"] - results["start_time"]).total_seconds()

        self.logger.info(f"Processing completed: {results['sources_processed']} sources, "
                        f"{results['total_documents']} documents in {results['duration_seconds']:.1f}s")

        return results

    def _process_single_source(self, source_config: Dict[str, Any]) -> Dict[str, Any]:
        """Process a single content source."""
        # This is where we would integrate with existing source processing logic
        # For now, return a mock result
        source_type = source_config.get('type', 'unknown')
        source_name = source_config.get('name', 'unnamed')

        # Mock processing based on source type
        if source_type == "web":
            # Mock web source processing
            return {"documents_processed": 5, "source_type": "web"}
        elif source_type == "file":
            # Mock file source processing
            return {"documents_processed": 12, "source_type": "file"}
        elif source_type == "database":
            # Mock database source processing
            return {"documents_processed": 100, "source_type": "database"}
        else:
            # Unknown source type
            self.logger.warning(f"Unknown source type '{source_type}' for source '{source_name}'")
            return {"documents_processed": 0, "source_type": source_type}

    def validate_config(self) -> List[str]:
        """Validate configuration for processing."""
        errors = []

        # Check content sources
        content_sources = self.config.get_content_sources()
        if not content_sources:
            errors.append("No content sources configured")

        for i, source in enumerate(content_sources):
            if 'name' not in source:
                errors.append(f"Content source {i} missing 'name' field")
            if 'type' not in source:
                errors.append(f"Content source {i} missing 'type' field")

        # Check storage configuration
        storage_backend = self.config.get_storage_backend()
        if not storage_backend:
            errors.append("Storage backend not configured")

        # Check processing configuration
        processing_config = self.config.get_processing_config()
        batch_size = processing_config.get("batch_size", 0)
        if batch_size <= 0:
            errors.append("Processing batch_size must be > 0")

        max_workers = processing_config.get("max_workers", 0)
        if max_workers <= 0:
            errors.append("Processing max_workers must be > 0")

        return errors


def display_processing_results(results: Dict[str, Any]):
    """Display processing results in formatted output."""
    print(f"\n{'='*60}")
    print(f"DOCUMENT PROCESSING RESULTS")
    print(f"{'='*60}")

    print(f"Started:             {results.get('start_time', 'N/A')}")
    print(f"Completed:           {results.get('end_time', 'N/A')}")
    print(f"Duration:            {results.get('duration_seconds', 0):.1f} seconds")
    print(f"Sources Processed:   {results.get('sources_processed', 0)}")
    print(f"Documents Processed: {results.get('total_documents', 0)}")

    processing_config = results.get('processing_config', {})
    if processing_config:
        print(f"\nProcessing Configuration:")
        print(f"  Batch Size:        {processing_config.get('batch_size', 'N/A')}")
        print(f"  Max Workers:       {processing_config.get('max_workers', 'N/A')}")
        print(f"  Timeout:           {processing_config.get('timeout_seconds', 'N/A')}s")

    errors = results.get('errors', [])
    if errors:
        print(f"\n❌ Errors ({len(errors)}):")
        for error in errors:
            print(f"  • {error}")
    else:
        print(f"\n✅ Processing completed successfully")


def main():
    """Main entry point for the document processing CLI."""
    parser = argparse.ArgumentParser(
        description="Go-Doc-Go Document Processing CLI",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Process all content sources defined in config
  python -m go_doc_go.cli.process

  # Process specific sources
  python -m go_doc_go.cli.process --sources wikipedia,documents

  # Process with custom config file
  python -m go_doc_go.cli.process --config /path/to/config.yaml

  # Validate configuration without processing
  python -m go_doc_go.cli.process --validate-only

  # Process with verbose logging
  python -m go_doc_go.cli.process --log-level DEBUG
        """
    )

    parser.add_argument(
        "--config", "-c",
        help="Path to configuration file (overrides GO_DOC_GO_CONFIG_PATH)"
    )

    parser.add_argument(
        "--sources", "-s",
        help="Comma-separated list of content source names to process (process all if not specified)"
    )

    parser.add_argument(
        "--validate-only", "-v",
        action="store_true",
        help="Only validate configuration, do not process documents"
    )

    parser.add_argument(
        "--log-level", "-l",
        choices=["DEBUG", "INFO", "WARNING", "ERROR"],
        default="INFO",
        help="Logging level (default: INFO)"
    )

    parser.add_argument(
        "--quiet", "-q",
        action="store_true",
        help="Suppress output (only show errors)"
    )

    args = parser.parse_args()

    # Configure logging
    log_level = getattr(logging, args.log_level)
    if args.quiet and args.log_level == "INFO":
        log_level = logging.ERROR

    logging.basicConfig(
        level=log_level,
        format="%(asctime)s - %(name)s - %(levelname)s - %(message)s"
    )

    logger = logging.getLogger(__name__)

    try:
        # Load configuration
        config_path = (
            args.config or
            os.environ.get("GO_DOC_GO_CONFIG_PATH", "./config.yaml")
        )

        if not os.path.exists(config_path):
            print(f"❌ Configuration file not found: {config_path}")
            return 1

        config = Config(config_path)
        processor = DocumentProcessor(config)

        # Validate configuration
        if not args.quiet:
            print(f"📋 Loading configuration from: {config_path}")

        validation_errors = processor.validate_config()
        if validation_errors:
            print(f"❌ Configuration validation failed:")
            for error in validation_errors:
                print(f"  • {error}")
            return 1

        if not args.quiet:
            print(f"✅ Configuration validation passed")

        if args.validate_only:
            print(f"✅ Configuration is valid")
            return 0

        # Parse sources filter
        sources_filter = None
        if args.sources:
            sources_filter = [s.strip() for s in args.sources.split(',')]
            if not args.quiet:
                print(f"📊 Processing sources: {', '.join(sources_filter)}")
        else:
            if not args.quiet:
                print(f"📊 Processing all configured sources")

        # Process documents
        results = processor.process_all_sources(sources_filter)

        if not args.quiet:
            display_processing_results(results)

        # Exit with error if there were processing errors
        return 1 if results.get('errors') else 0

    except KeyboardInterrupt:
        print(f"\n⏹️  Processing interrupted by user")
        return 130
    except Exception as e:
        logger.error(f"Document processing failed: {str(e)}")
        logger.debug("Exception details:", exc_info=True)
        return 1


if __name__ == "__main__":
    sys.exit(main())