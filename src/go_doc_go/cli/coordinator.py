#!/usr/bin/env python3
"""
Command-line interface for running the document processing coordinator.
"""

import argparse
import logging
import os
import sys
from pathlib import Path

# Add the src directory to Python path
sys.path.insert(0, str(Path(__file__).parent.parent.parent))

from go_doc_go import Config
from go_doc_go.main import ingest_documents


def main():
    """Main entry point for the processing coordinator."""
    parser = argparse.ArgumentParser(
        description="Go-Doc-Go Distributed Processing Coordinator",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Run coordinator with default config
  python -m go_doc_go.cli.coordinator

  # Run with custom config file
  python -m go_doc_go.cli.coordinator --config /path/to/config.yaml

  # Run with specific sources and link depth
  python -m go_doc_go.cli.coordinator --max-link-depth 3

Environment Variables:
  GO_DOC_GO_CONFIG_PATH: Path to configuration file (default: ./config.yaml)
        """
    )
    
    parser.add_argument(
        "--config", "-c",
        help="Path to configuration file (overrides GO_DOC_GO_CONFIG_PATH)"
    )
    
    parser.add_argument(
        "--max-link-depth", "-d",
        type=int,
        help="Maximum link depth to follow (overrides config)"
    )
    
    parser.add_argument(
        "--log-level", "-l",
        choices=["DEBUG", "INFO", "WARNING", "ERROR"],
        default="INFO",
        help="Logging level (default: INFO)"
    )
    
    parser.add_argument(
        "--log-file",
        help="Log to file instead of stdout"
    )
    
    args = parser.parse_args()
    
    # Configure logging
    log_format = "%(asctime)s - %(name)s - %(levelname)s - %(message)s"
    log_level = getattr(logging, args.log_level)
    
    if args.log_file:
        logging.basicConfig(
            level=log_level,
            format=log_format,
            filename=args.log_file,
            filemode='a'
        )
    else:
        logging.basicConfig(
            level=log_level,
            format=log_format
        )
    
    logger = logging.getLogger(__name__)
    
    try:
        # Determine config file path
        config_path = (
            args.config or 
            os.environ.get("GO_DOC_GO_CONFIG_PATH", "./config.yaml")
        )
        
        logger.info(f"Starting Go-Doc-Go Processing Coordinator")
        logger.info(f"Config file: {config_path}")
        logger.info(f"Log level: {args.log_level}")
        
        # Load configuration
        config = Config(config_path)
        
        # Ensure processing mode is set to distributed
        if config.config.get('processing', {}).get('mode') != 'distributed':
            logger.warning("Config processing mode is not 'distributed' - forcing distributed mode")
            if 'processing' not in config.config:
                config.config['processing'] = {}
            config.config['processing']['mode'] = 'distributed'
        
        # Run distributed processing
        logger.info("Starting distributed document processing coordination")
        
        stats = ingest_documents(
            config=config,
            max_link_depth=args.max_link_depth,
            processing_mode='distributed'
        )
        
        # Log final statistics
        logger.info("Distributed processing coordination completed successfully!")
        logger.info(f"Final statistics:")
        logger.info(f"  Run ID: {stats.get('run_id', 'N/A')}")
        logger.info(f"  Documents queued: {stats.get('documents_queued', 0)}")
        logger.info(f"  Documents processed: {stats.get('documents_processed', 0)}")
        logger.info(f"  Documents failed: {stats.get('documents_failed', 0)}")
        logger.info(f"  Cross-document relationships: {stats.get('cross_document_relationships', 0)}")
        logger.info(f"  Total runtime: {stats.get('total_runtime_seconds', 0):.1f} seconds")
        
        # Print summary to stdout for visibility
        print(f"\n🎉 Distributed Processing Completed!")
        print(f"📊 Run ID: {stats.get('run_id', 'N/A')}")
        print(f"📄 Documents processed: {stats.get('documents_processed', 0)}")
        print(f"❌ Documents failed: {stats.get('documents_failed', 0)}")
        print(f"🔗 Cross-document relationships: {stats.get('cross_document_relationships', 0)}")
        print(f"⏱️  Total runtime: {stats.get('total_runtime_seconds', 0):.1f} seconds")
        
        return 0
        
    except KeyboardInterrupt:
        logger.info("Coordinator shutdown requested by user")
        return 1
        
    except Exception as e:
        logger.error(f"Coordinator failed: {str(e)}")
        logger.debug("Exception details:", exc_info=True)
        return 1


if __name__ == "__main__":
    sys.exit(main())