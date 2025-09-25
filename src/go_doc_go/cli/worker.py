#!/usr/bin/env python3
"""
Refactored document worker using SimpleJobControlDB architecture.
Replaces the complex PostgreSQL-based worker system with a clean SQLite-based approach.
"""

import argparse
import logging
import os
import sys
import signal
import time
import uuid
import threading
import socket
from pathlib import Path
from datetime import datetime
from typing import Optional, Dict, Any

# Add the src directory to Python path
sys.path.insert(0, str(Path(__file__).parent.parent.parent))

from go_doc_go.config import Config
from go_doc_go.shared.simple_job_control import SimpleJobControlDB
from go_doc_go.content_source.factory import get_content_source
from go_doc_go.document_parser.factory import get_parser_for_content
from go_doc_go.embeddings.factory import get_embedding_generator
from go_doc_go.relationships import create_relationship_detector
from go_doc_go.storage_adapters.factory import StorageFactory


logger = logging.getLogger(__name__)


class SimpleDocumentWorker:
    """
    Simplified document worker using the new job control architecture.
    Uses SimpleJobControlDB for document claiming and analytics outputs for storage.
    """

    def __init__(self, config: Config, worker_id: Optional[str] = None, max_documents: Optional[int] = None):
        """Initialize the simple document worker."""
        self.config = config
        self.worker_id = worker_id or f"worker_{uuid.uuid4().hex[:8]}"
        self.max_documents = max_documents
        self.documents_processed = 0
        self.running = False
        self.shutdown_requested = False
        self.is_leader = False
        self.discovery_thread = None
        self.discovery_interval = 60  # Discovery every 60 seconds

        # Initialize job control database
        self.job_control = SimpleJobControlDB.create(config)

        # Initialize content sources
        self.content_sources = {}
        for source_config in config.get_content_sources():
            source_name = source_config.get('name')
            if source_name:
                self.content_sources[source_name] = get_content_source(source_config)

        # Initialize analytics storage
        self.analytics_storage = []
        analytics_config = config.get_analytics_config()
        if analytics_config and analytics_config.get('enabled'):
            for output_config in analytics_config.get('outputs', []):
                storage = StorageFactory.create_analytics_storage(output_config)
                self.analytics_storage.append(storage)

        # Initialize processing components
        self.embedding_generator = None
        if config.is_embedding_enabled():
            self.embedding_generator = get_embedding_generator(config)

        self.relationship_detector = create_relationship_detector(config)

        logger.info(f"Initialized SimpleDocumentWorker: {self.worker_id}")
        logger.info(f"Content sources: {list(self.content_sources.keys())}")
        logger.info(f"Analytics outputs: {len(self.analytics_storage)}")

    def setup_signal_handlers(self):
        """Setup graceful shutdown signal handlers."""
        def signal_handler(signum, frame):
            logger.info(f"Worker {self.worker_id} received signal {signum}, requesting shutdown")
            self.shutdown_requested = True

        signal.signal(signal.SIGTERM, signal_handler)
        signal.signal(signal.SIGINT, signal_handler)

    def discover_and_queue_documents(self, use_continuous_discovery: bool = False):
        """Discover documents from content sources and queue them for processing."""
        total_queued = 0
        import time

        for source_name, content_source in self.content_sources.items():
            try:
                logger.info(f"Discovering documents from source: {source_name}")

                # Use continuous discovery for supported sources if requested
                if use_continuous_discovery and hasattr(content_source, 'supports_continuous_discovery'):
                    if content_source.supports_continuous_discovery():
                        # Get last discovery time (simplified - in production, this would be tracked)
                        last_discovery = getattr(self, f'_last_discovery_{source_name}', None)
                        documents = content_source.discover_new_documents(last_discovery)
                        setattr(self, f'_last_discovery_{source_name}', time.time())
                        logger.debug(f"Used continuous discovery for {source_name}")
                    else:
                        documents = content_source.list_documents()
                        logger.debug(f"Used standard discovery for {source_name}")
                else:
                    documents = content_source.list_documents()

                queued_count = 0
                for doc_info in documents:
                    doc_id = doc_info.get('id') or doc_info.get('source_id') or doc_info.get('url') or str(uuid.uuid4())

                    # Check if already queued
                    if not self.job_control.is_document_queued(doc_id):
                        self.job_control.enqueue_document(
                            doc_id=doc_id,
                            source=source_name,
                            metadata=doc_info
                        )
                        queued_count += 1

                logger.info(f"Queued {queued_count} new documents from {source_name}")
                total_queued += queued_count

            except Exception as e:
                logger.error(f"Failed to discover documents from {source_name}: {e}")

        if total_queued > 0:
            logger.info(f"Total documents queued for processing: {total_queued}")
        else:
            logger.info("No new documents found to queue")

        return total_queued

    def attempt_leader_election(self) -> bool:
        """Attempt to become the leader. Returns True if successful."""
        worker_info = {
            'hostname': socket.gethostname(),
            'pid': os.getpid(),
            'started_at': datetime.now().isoformat()
        }

        success = self.job_control.elect_leader(self.worker_id, worker_info)
        if success:
            self.is_leader = True
            logger.info(f"Worker {self.worker_id} became leader")
        return success

    def discovery_loop(self):
        """Background thread for leader to continuously discover documents."""
        logger.info(f"Leader {self.worker_id} starting discovery loop")

        while self.running and not self.shutdown_requested and self.is_leader:
            try:
                # Update leader heartbeat
                self.job_control.update_leader_heartbeat(self.worker_id)

                # Discover and queue new documents using continuous discovery
                queued = self.discover_and_queue_documents(use_continuous_discovery=True)
                if queued > 0:
                    logger.info(f"Leader discovered {queued} new documents")

                # Sleep for discovery interval
                for _ in range(self.discovery_interval):
                    if not self.running or self.shutdown_requested:
                        break
                    time.sleep(1)

            except Exception as e:
                logger.error(f"Discovery loop error: {e}")
                time.sleep(10)  # Wait before retrying

        logger.info(f"Leader {self.worker_id} stopping discovery loop")

    def start_discovery_thread(self):
        """Start the discovery thread if this worker is the leader."""
        if self.is_leader and self.discovery_thread is None:
            self.discovery_thread = threading.Thread(
                target=self.discovery_loop,
                name=f"discovery-{self.worker_id}",
                daemon=True
            )
            self.discovery_thread.start()

    def stop_discovery_thread(self):
        """Stop the discovery thread."""
        if self.discovery_thread and self.discovery_thread.is_alive():
            logger.info("Stopping discovery thread")
            self.discovery_thread.join(timeout=5)
            self.discovery_thread = None

    def process_document(self, document_info: Dict[str, Any]) -> bool:
        """Process a single document."""
        doc_id = document_info['doc_id']
        source_name = document_info['source']

        try:
            logger.debug(f"Processing document {doc_id} from {source_name}")

            # Get content source
            content_source = self.content_sources.get(source_name)
            if not content_source:
                raise ValueError(f"Unknown content source: {source_name}")

            # Fetch document content
            source_id = document_info['metadata'].get('source_id', doc_id)
            doc_content = content_source.fetch_document(source_id)
            if not doc_content:
                logger.warning(f"No content retrieved for document {doc_id}")
                return False

            # Parse document
            parser = get_parser_for_content(doc_content)
            if not parser:
                logger.warning(f"No parser found for document {doc_id}")
                return False

            parse_result = parser.parse(doc_content)

            # Generate embeddings if enabled
            if self.embedding_generator and 'elements' in parse_result:
                for element in parse_result['elements']:
                    if element.get('content_preview'):
                        # Use generate method for ContextualEmbeddingGenerator
                        if hasattr(self.embedding_generator, 'generate'):
                            embedding = self.embedding_generator.generate(
                                element['content_preview']
                            )
                        else:
                            embedding = self.embedding_generator.generate_embedding(
                                element['content_preview']
                            )
                        if embedding:
                            element['embedding'] = embedding

            # Detect relationships
            if self.relationship_detector and 'elements' in parse_result:
                relationships = self.relationship_detector.detect_relationships(
                    doc_content,  # document
                    parse_result['elements']  # elements
                )
                if 'relationships' not in parse_result:
                    parse_result['relationships'] = []
                parse_result['relationships'].extend(relationships)

            # Store in analytics outputs
            run_id = f"worker_{self.worker_id}_{int(datetime.now().timestamp())}"
            for storage in self.analytics_storage:
                try:
                    # Store document
                    storage.append_documents([{
                        'doc_id': doc_id,
                        'source_name': source_name,
                        'title': doc_content.get('title', ''),
                        'url': doc_content.get('url', ''),
                        'content_type': doc_content.get('content_type', ''),
                        'processed_at': datetime.now().isoformat(),
                        'element_count': len(parse_result.get('elements', [])),
                        'relationship_count': len(parse_result.get('relationships', []))
                    }], run_id)

                    # Store elements
                    if 'elements' in parse_result:
                        for element in parse_result['elements']:
                            element['doc_id'] = doc_id
                            element['source_name'] = source_name
                        storage.append_elements(parse_result['elements'], run_id)

                    # Store relationships
                    if 'relationships' in parse_result:
                        for relationship in parse_result['relationships']:
                            relationship['doc_id'] = doc_id
                            relationship['source_name'] = source_name
                        storage.append_relationships(parse_result['relationships'], run_id)

                    # Store embeddings
                    embeddings = []
                    for element in parse_result.get('elements', []):
                        if 'embedding' in element:
                            embeddings.append({
                                'element_id': element['element_id'],
                                'doc_id': doc_id,
                                'source_name': source_name,
                                'embedding': element['embedding'],
                                'text': element.get('content_preview', '')
                            })

                    if embeddings:
                        storage.append_embeddings(embeddings, run_id)

                except Exception as e:
                    logger.error(f"Failed to store results in analytics storage: {e}")

            logger.debug(f"Successfully processed document {doc_id}")
            return True

        except Exception as e:
            logger.error(f"Failed to process document {doc_id}: {e}")
            return False

    def run(self):
        """Main worker loop with leader election."""
        logger.info(f"Starting worker {self.worker_id}")

        if self.max_documents:
            logger.info(f"Limited to processing {self.max_documents} documents")

        self.setup_signal_handlers()
        self.running = True

        try:
            # Register worker
            worker_info = {
                'hostname': socket.gethostname(),
                'pid': os.getpid(),
                'started_at': datetime.now().isoformat()
            }
            self.job_control.register_worker(self.worker_id, worker_info)

            # Attempt leader election
            if self.attempt_leader_election():
                logger.info(f"Worker {self.worker_id} is now leader")
                # Initial discovery as leader
                self.discover_and_queue_documents()
                # Start continuous discovery thread
                self.start_discovery_thread()
            else:
                logger.info(f"Worker {self.worker_id} is a follower")

            # Main processing loop (all workers participate)
            while self.running and not self.shutdown_requested:
                # Check document limit
                if self.max_documents and self.documents_processed >= self.max_documents:
                    logger.info(f"Reached maximum document limit ({self.max_documents})")
                    break

                # Update worker heartbeat
                self.job_control.update_worker_heartbeat(self.worker_id)

                # Claim a document to process
                document_info = self.job_control.claim_next_document(self.worker_id)

                if document_info:
                    # Process the document
                    success = self.process_document(document_info)

                    if success:
                        self.job_control.complete_document(
                            document_info['doc_id'],
                            self.worker_id,
                            True
                        )
                        self.documents_processed += 1
                        logger.info(f"Worker {self.worker_id} completed document {document_info['doc_id']} ({self.documents_processed} total)")
                    else:
                        self.job_control.complete_document(
                            document_info['doc_id'],
                            self.worker_id,
                            False,
                            "Processing failed"
                        )
                        logger.warning(f"Worker {self.worker_id} failed to process document {document_info['doc_id']}")
                else:
                    # No documents available, wait a bit
                    if self.documents_processed == 0 and not self.is_leader:
                        logger.info("No documents available to process")
                        time.sleep(5)
                        # Check if we can become leader if none exists
                        current_leader = self.job_control.get_current_leader()
                        if not current_leader:
                            if self.attempt_leader_election():
                                self.start_discovery_thread()
                    else:
                        logger.debug("No more documents available, waiting...")
                        time.sleep(5)

        finally:
            # Cleanup
            if self.is_leader:
                logger.info(f"Releasing leadership for worker {self.worker_id}")
                self.stop_discovery_thread()
                self.job_control.release_leadership(self.worker_id)

        logger.info(f"Worker {self.worker_id} stopping. Processed {self.documents_processed} documents")
        self.running = False


def main():
    """Main entry point for the refactored document worker."""
    parser = argparse.ArgumentParser(
        description="Go-Doc-Go Simple Document Worker (New Architecture)",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Run single worker with default config
  python -m go_doc_go.cli.worker_new

  # Run worker with custom config file
  python -m go_doc_go.cli.worker_new --config /path/to/config.yaml

  # Process limited number of documents
  python -m go_doc_go.cli.worker_new --max-documents 10

  # Run with custom worker ID
  python -m go_doc_go.cli.worker_new --worker-id my-worker-01

Environment Variables:
  GO_DOC_GO_CONFIG_PATH: Path to configuration file (default: ./config.yaml)
        """
    )

    parser.add_argument(
        "--config", "-c",
        help="Path to configuration file (overrides GO_DOC_GO_CONFIG_PATH)"
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
        "--max-documents", "-m",
        type=int,
        help="Maximum number of documents to process before stopping"
    )

    args = parser.parse_args()

    # Configure logging
    log_format = "%(asctime)s - %(name)s - %(levelname)s - %(message)s"
    log_level = getattr(logging, args.log_level)

    logging.basicConfig(
        level=log_level,
        format=log_format
    )

    logger = logging.getLogger(__name__)

    try:
        # Determine config file path
        config_path = (
            args.config or
            os.environ.get('GO_DOC_GO_CONFIG_PATH') or
            './config.yaml'
        )

        if not os.path.exists(config_path):
            logger.error(f"Configuration file not found: {config_path}")
            sys.exit(1)

        logger.info(f"Loading configuration from: {config_path}")
        config = Config(config_path)

        # Create and run worker
        worker = SimpleDocumentWorker(
            config=config,
            worker_id=args.worker_id,
            max_documents=args.max_documents
        )

        worker.run()

        logger.info("Worker completed successfully")

    except KeyboardInterrupt:
        logger.info("Worker interrupted by user")
        sys.exit(0)
    except Exception as e:
        logger.error(f"Worker failed: {e}", exc_info=True)
        sys.exit(1)


if __name__ == "__main__":
    main()