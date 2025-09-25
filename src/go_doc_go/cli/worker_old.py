#!/usr/bin/env python3
"""
Command-line interface for running document workers in distributed processing.
"""

import argparse
import logging
import os
import sys
from pathlib import Path

# Add the src directory to Python path
sys.path.insert(0, str(Path(__file__).parent.parent.parent))

from go_doc_go import Config
from go_doc_go.work_queue.worker import DocumentWorker, WorkerManager


def main():
    """Main entry point for the document worker."""
    parser = argparse.ArgumentParser(
        description="Go-Doc-Go Distributed Document Worker",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Run single worker with default config
  python -m go_doc_go.cli.worker
  
  # Run worker with custom config file
  python -m go_doc_go.cli.worker --config /path/to/config.yaml
  
  # Run multiple workers in the same process
  python -m go_doc_go.cli.worker --workers 4
  
  # Run with custom worker ID
  python -m go_doc_go.cli.worker --worker-id worker-prod-01

Environment Variables:
  GO_DOC_GO_CONFIG_PATH: Path to configuration file (default: ./config.yaml)
        """
    )
    
    parser.add_argument(
        "--config", "-c",
        help="Path to configuration file (overrides GO_DOC_GO_CONFIG_PATH)"
    )
    
    parser.add_argument(
        "--workers", "-w",
        type=int,
        default=1,
        help="Number of worker threads to run in this process (default: 1)"
    )
    
    parser.add_argument(
        "--worker-id",
        help="Custom worker ID (auto-generated if not provided)"
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
    
    parser.add_argument(
        "--max-documents", "-m",
        type=int,
        help="Maximum number of documents to process before stopping (default: unlimited)"
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
        
        logger.info(f"Starting Go-Doc-Go Document Worker(s)")
        logger.info(f"Config file: {config_path}")
        logger.info(f"Workers: {args.workers}")
        logger.info(f"Log level: {args.log_level}")
        if args.worker_id:
            logger.info(f"Worker ID: {args.worker_id}")
        
        # Load configuration
        config = Config(config_path)
        
        # Ensure processing mode is set to worker
        if config.config.get('processing', {}).get('mode') != 'worker':
            logger.warning("Config processing mode is not 'worker' - forcing worker mode")
            if 'processing' not in config.config:
                config.config['processing'] = {}
            config.config['processing']['mode'] = 'worker'
        
        # Run workers
        if args.workers == 1:
            # Single worker mode
            logger.info("Starting single document worker")
            
            worker = DocumentWorker(config, args.worker_id)
            
            # Override max_documents if specified
            if args.max_documents:
                worker.max_documents = args.max_documents
                logger.info(f"Limited to processing {args.max_documents} documents")
            
            stats = worker.start()
            
            # Log final statistics
            logger.info("Document worker completed successfully!")
            logger.info(f"Worker statistics:")
            logger.info(f"  Worker ID: {worker.worker_id}")
            logger.info(f"  Documents processed: {stats.get('documents_processed', 0)}")
            logger.info(f"  Documents failed: {stats.get('documents_failed', 0)}")
            logger.info(f"  Elements created: {stats.get('elements_created', 0)}")
            logger.info(f"  Relationships created: {stats.get('relationships_created', 0)}")
            logger.info(f"  Links discovered: {stats.get('links_discovered', 0)}")
            
            if stats.get('start_time') and stats.get('end_time'):
                runtime = stats['end_time'] - stats['start_time']
                logger.info(f"  Runtime: {runtime:.1f} seconds")
            
            # Print summary to stdout for visibility
            print(f"\n🎉 Document Worker Completed!")
            print(f"👷 Worker ID: {worker.worker_id}")
            print(f"📄 Documents processed: {stats.get('documents_processed', 0)}")
            print(f"❌ Documents failed: {stats.get('documents_failed', 0)}")
            print(f"📝 Elements created: {stats.get('elements_created', 0)}")
            print(f"🔗 Relationships created: {stats.get('relationships_created', 0)}")
            print(f"🔍 Links discovered: {stats.get('links_discovered', 0)}")
            
            if stats.get('start_time') and stats.get('end_time'):
                runtime = stats['end_time'] - stats['start_time']
                print(f"⏱️  Runtime: {runtime:.1f} seconds")
        
        else:
            # Multi-worker mode
            logger.info(f"Starting {args.workers} document workers")
            
            manager = WorkerManager(config, args.workers)
            
            # Note: max_documents limitation not supported in multi-worker mode
            if args.max_documents:
                logger.warning("--max-documents not supported in multi-worker mode, ignoring")
            
            combined_stats = manager.start_all()
            
            # Log final statistics
            logger.info("All document workers completed successfully!")
            logger.info(f"Combined worker statistics:")
            logger.info(f"  Total workers: {combined_stats.get('total_workers', 0)}")
            logger.info(f"  Documents processed: {combined_stats.get('documents_processed', 0)}")
            logger.info(f"  Documents failed: {combined_stats.get('documents_failed', 0)}")
            logger.info(f"  Elements created: {combined_stats.get('elements_created', 0)}")
            logger.info(f"  Relationships created: {combined_stats.get('relationships_created', 0)}")
            logger.info(f"  Links discovered: {combined_stats.get('links_discovered', 0)}")
            
            # Print summary to stdout for visibility
            print(f"\n🎉 All Document Workers Completed!")
            print(f"👷 Total workers: {combined_stats.get('total_workers', 0)}")
            print(f"📄 Documents processed: {combined_stats.get('documents_processed', 0)}")
            print(f"❌ Documents failed: {combined_stats.get('documents_failed', 0)}")
            print(f"📝 Elements created: {combined_stats.get('elements_created', 0)}")
            print(f"🔗 Relationships created: {combined_stats.get('relationships_created', 0)}")
            print(f"🔍 Links discovered: {combined_stats.get('links_discovered', 0)}")
        
        return 0
        
    except KeyboardInterrupt:
        logger.info("Worker shutdown requested by user")
        return 1
        
    except Exception as e:
        logger.error(f"Worker failed: {str(e)}")
        logger.debug("Exception details:", exc_info=True)
        return 1


if __name__ == "__main__":
    sys.exit(main())