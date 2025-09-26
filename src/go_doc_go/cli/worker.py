#!/usr/bin/env python3
"""
Refactored document worker using SimpleJobControlDB architecture.
Replaces the complex PostgreSQL-based worker system with a clean SQLite-based approach.
"""

import click
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

# Package imports (no path manipulation needed for proper package installation)

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

    def __init__(self, config: Config, worker_id: Optional[str] = None, max_documents: Optional[int] = None, discovery_interval: int = 60):
        """Initialize the simple document worker."""
        self.config = config
        self.worker_id = worker_id or f"worker_{uuid.uuid4().hex[:8]}"
        self.max_documents = max_documents
        self.documents_processed = 0
        self.running = False
        self.shutdown_requested = False
        self.is_leader = False
        self.discovery_thread = None
        self.discovery_interval = discovery_interval  # Discovery interval in seconds

        # Queue monitoring for Neo4j export
        self.last_queue_empty_time = None
        self.neo4j_export_enabled = config.config.get('processing', {}).get('neo4j_export', {}).get('enabled', False)
        self.neo4j_export_wait_time = config.config.get('processing', {}).get('neo4j_export', {}).get('empty_queue_wait_time', 600)  # 10 minutes
        self.last_neo4j_export_time = None

        # Initialize job control database
        self.job_control = SimpleJobControlDB.create(config)

        # Initialize content sources with per-source discovery tracking
        self.content_sources = {}
        self.source_discovery_intervals = {}  # Per-source discovery intervals
        self.source_last_discovery = {}  # Track last discovery time per source

        for source_config in config.get_content_sources():
            source_name = source_config.get('name')
            if source_name:
                content_source = get_content_source(source_config)
                # Inject job control for asynchronous link queuing
                content_source.set_job_control(self.job_control)
                self.content_sources[source_name] = content_source

                # Set per-source discovery interval
                # Use 'discovery_interval' consistently across all sources
                source_interval = source_config.get('discovery_interval', self.discovery_interval)
                self.source_discovery_intervals[source_name] = source_interval
                self.source_last_discovery[source_name] = 0  # Will trigger immediate discovery

                logger.debug(f"Source {source_name} discovery interval: {source_interval}s")

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

    def discover_and_queue_documents_for_source(self, source_name: str, use_continuous_discovery: bool = False):
        """Discover documents from a single content source and queue them for processing."""
        content_source = self.content_sources.get(source_name)
        if not content_source:
            logger.warning(f"Source {source_name} not found")
            return 0

        queued_count = 0
        try:
            logger.info(f"Discovering documents from source: {source_name}")

            # Use continuous discovery for supported sources if requested
            if use_continuous_discovery and hasattr(content_source, 'supports_continuous_discovery'):
                if content_source.supports_continuous_discovery():
                    # Get last discovery time
                    last_discovery = getattr(self, f'_last_discovery_{source_name}', None)
                    documents = content_source.discover_new_documents(last_discovery)
                    setattr(self, f'_last_discovery_{source_name}', time.time())
                    logger.debug(f"Used continuous discovery for {source_name}")
                else:
                    documents = content_source.list_documents()
                    logger.debug(f"Used standard discovery for {source_name}")
            else:
                documents = content_source.list_documents()

            for doc_info in documents:
                doc_id = doc_info.get('id') or doc_info.get('source_id') or doc_info.get('url') or str(uuid.uuid4())

                # Check if document has changed using content source change detection
                current_modified = doc_info.get('metadata', {}).get('last_modified')
                should_process = False

                # Always check if document is already queued first
                if self.job_control.is_document_queued(doc_id):
                    logger.debug(f"Document {doc_id} is already queued, skipping")
                    continue

                # Use job control change tracking if available
                if hasattr(self.job_control, 'has_document_changed'):
                    # For comprehensive change detection
                    current_hash = None
                    if hasattr(content_source, 'has_changed'):
                        # Get stored metadata for comparison
                        stored_metadata = self.job_control.get_document_metadata(doc_id)
                        stored_modified = stored_metadata.get('last_modified') if stored_metadata else None

                        # Use content source has_changed method
                        source_changed = content_source.has_changed(doc_id, stored_modified)

                        # Also check job control tracking
                        jc_changed = self.job_control.has_document_changed(
                            doc_id, source_name, current_modified, current_hash
                        )

                        should_process = source_changed or jc_changed
                    else:
                        # Fallback to job control only
                        should_process = self.job_control.has_document_changed(
                            doc_id, source_name, current_modified, current_hash
                        )
                else:
                    # Fallback to content source change detection only
                    if hasattr(content_source, 'has_changed'):
                        try:
                            should_process = content_source.has_changed(doc_id, None)
                        except Exception:
                            # If change detection fails, process the document
                            should_process = True
                    else:
                        # No change detection available, always process
                        should_process = True

                if should_process:
                    logger.debug(f"Document {doc_id} has changed or is new, queuing for processing")
                    self.job_control.enqueue_document(
                        doc_id=doc_id,
                        source=source_name,
                        metadata=doc_info
                    )
                    queued_count += 1
                else:
                    logger.debug(f"Document {doc_id} has not changed, skipping")

            logger.info(f"Queued {queued_count} new documents from {source_name}")
        except Exception as e:
            logger.error(f"Failed to discover documents from {source_name}: {e}")

        return queued_count

    def discover_and_queue_documents(self, use_continuous_discovery: bool = False):
        """Discover documents from all content sources and queue them for processing."""
        total_queued = 0

        for source_name in self.content_sources:
            queued = self.discover_and_queue_documents_for_source(source_name, use_continuous_discovery)
            total_queued += queued

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

                # Check each source to see if it's time for discovery
                current_time = time.time()
                total_queued = 0

                for source_name in self.content_sources:
                    time_since_last = current_time - self.source_last_discovery.get(source_name, 0)
                    source_interval = self.source_discovery_intervals.get(source_name, self.discovery_interval)

                    # Check if this source is due for discovery
                    if time_since_last >= source_interval:
                        logger.debug(f"Discovering documents from {source_name} (interval: {source_interval}s)")
                        queued = self.discover_and_queue_documents_for_source(
                            source_name,
                            use_continuous_discovery=True
                        )
                        self.source_last_discovery[source_name] = current_time
                        total_queued += queued

                        if queued > 0:
                            logger.info(f"Leader discovered {queued} new documents from {source_name}")

                if total_queued > 0:
                    logger.info(f"Leader discovered {total_queued} total new documents")

                # Check queue status for Neo4j export
                if self.neo4j_export_enabled:
                    self.check_queue_for_neo4j_export()

                # Sleep for a short interval before checking sources again
                # Use the minimum of all source intervals, but cap at 60 seconds for responsiveness
                min_interval = min(self.source_discovery_intervals.values()) if self.source_discovery_intervals else self.discovery_interval
                sleep_interval = min(60, min_interval)

                for _ in range(sleep_interval):
                    if not self.running or self.shutdown_requested:
                        break
                    time.sleep(1)

            except Exception as e:
                logger.error(f"Discovery loop error: {e}")
                time.sleep(10)  # Wait before retrying

        logger.info(f"Leader {self.worker_id} stopping discovery loop")

    def check_queue_for_neo4j_export(self):
        """Check if queue is empty and trigger Neo4j export if conditions are met."""
        try:
            # Get current queue status
            status = self.job_control.get_processing_status()
            queue_status = status.get('queue_status', 'active')
            doc_counts = status.get('documents', {})

            pending_docs = doc_counts.get('pending', 0)
            processing_docs = doc_counts.get('processing', 0)

            is_queue_empty = (pending_docs == 0 and processing_docs == 0)

            current_time = time.time()

            if is_queue_empty:
                if self.last_queue_empty_time is None:
                    # Queue just became empty
                    self.last_queue_empty_time = current_time
                    logger.info("Queue is now empty, starting timer for Neo4j export")
                elif current_time - self.last_queue_empty_time >= self.neo4j_export_wait_time:
                    # Queue has been empty long enough
                    time_since_last_export = float('inf')
                    if self.last_neo4j_export_time:
                        time_since_last_export = current_time - self.last_neo4j_export_time

                    # Only export if it's been at least the wait time since last export
                    if time_since_last_export >= self.neo4j_export_wait_time:
                        logger.info("Queue has been empty long enough, triggering Neo4j export")
                        self.trigger_neo4j_export()
                        self.last_neo4j_export_time = current_time
                        self.last_queue_empty_time = None  # Reset timer
                    else:
                        logger.debug(f"Queue empty but exported recently, waiting {self.neo4j_export_wait_time - time_since_last_export:.1f}s")
            else:
                # Queue is not empty, reset timer
                if self.last_queue_empty_time is not None:
                    logger.debug("Queue is no longer empty, resetting export timer")
                    self.last_queue_empty_time = None

        except Exception as e:
            logger.error(f"Error checking queue for Neo4j export: {e}")

    def trigger_neo4j_export(self):
        """Trigger Neo4j graph export."""
        try:
            logger.info("Starting Neo4j graph export...")

            # Get Neo4j export configuration
            neo4j_config = self.config.config.get('processing', {}).get('neo4j_export', {})
            source_analytics = neo4j_config.get('source_analytics', 'parquet')

            # Import here to avoid circular imports
            from go_doc_go.graph_export import GraphExporter

            exporter = GraphExporter(self.config, source_analytics)
            result = exporter.export_to_neo4j()

            if result.get('success'):
                exported_counts = result.get('exported', {})
                logger.info(f"Neo4j export completed successfully: "
                           f"docs={exported_counts.get('documents', 0)}, "
                           f"elements={exported_counts.get('elements', 0)}, "
                           f"relationships={exported_counts.get('relationships', 0)}")
            else:
                logger.error(f"Neo4j export failed: {result.get('error', 'Unknown error')}")

        except ImportError:
            logger.warning("GraphExporter not available - Neo4j export disabled")
        except Exception as e:
            logger.error(f"Neo4j export failed: {e}")

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

            # Extract and queue links for asynchronous processing (web content sources only)
            current_depth = document_info.get('metadata', {}).get('discovery_depth', 0)
            max_depth = document_info.get('metadata', {}).get('max_link_depth', content_source.max_link_depth)

            # Only follow links if we haven't exceeded max depth
            if hasattr(content_source, 'follow_links') and 'content' in doc_content and current_depth < max_depth:
                try:
                    # Temporarily update content source max_link_depth for this document
                    original_max_depth = content_source.max_link_depth
                    content_source.max_link_depth = max_depth

                    content_source.follow_links(
                        doc_content['content'],
                        source_id,
                        current_depth
                    )

                    # Restore original max depth
                    content_source.max_link_depth = original_max_depth

                    logger.debug(f"Link discovery completed for document {doc_id} at depth {current_depth}/{max_depth}")
                except Exception as e:
                    # Restore original max depth on error
                    if 'original_max_depth' in locals():
                        content_source.max_link_depth = original_max_depth
                    logger.warning(f"Link discovery failed for document {doc_id}: {str(e)}")
            else:
                if current_depth >= max_depth:
                    logger.debug(f"Skipping link discovery for {doc_id} - at max depth {current_depth}/{max_depth}")
                else:
                    logger.debug(f"Skipping link discovery for {doc_id} - not a web content source or no HTML content")

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
                # Add doc_id to document and elements for relationship detection
                doc_content['doc_id'] = doc_id
                for element in parse_result['elements']:
                    element['doc_id'] = doc_id

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
                    }])

                    # Store elements
                    if 'elements' in parse_result:
                        for element in parse_result['elements']:
                            element['doc_id'] = doc_id
                            element['source_name'] = source_name
                        storage.append_elements(parse_result['elements'])

                    # Store relationships
                    if 'relationships' in parse_result:
                        for relationship in parse_result['relationships']:
                            relationship['doc_id'] = doc_id
                            relationship['source_name'] = source_name
                        storage.append_relationships(parse_result['relationships'])

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
                        storage.append_embeddings(embeddings)

                except Exception as e:
                    logger.error(f"Failed to store results in analytics storage: {e}")

            logger.debug(f"Successfully processed document {doc_id}")

            # Store document metadata for change tracking if supported
            if hasattr(self.job_control, 'store_document_metadata'):
                try:
                    # Extract metadata from doc_content and parse_result
                    doc_metadata = doc_content.get('metadata', {})
                    last_modified = doc_metadata.get('last_modified')
                    content_hash = doc_content.get('content_hash')
                    file_size = doc_metadata.get('size') or doc_metadata.get('file_size')

                    # Create processing stats
                    processing_stats = {
                        'element_count': len(parse_result.get('elements', [])),
                        'relationship_count': len(parse_result.get('relationships', [])),
                        'embedding_count': sum(1 for e in parse_result.get('elements', []) if 'embedding' in e),
                        'processed_at': datetime.now().isoformat(),
                        'worker_id': self.worker_id
                    }

                    self.job_control.store_document_metadata(
                        doc_id=doc_id,
                        source=source_name,
                        last_modified=last_modified,
                        content_hash=content_hash,
                        file_size=file_size,
                        processing_stats=processing_stats
                    )

                    logger.debug(f"Stored metadata for document {doc_id}")

                except Exception as e:
                    logger.warning(f"Failed to store metadata for document {doc_id}: {e}")

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


@click.command()
@click.option(
    "--config", "-c",
    help="Path to configuration file (overrides GO_DOC_GO_CONFIG_PATH)"
)
@click.option(
    "--worker-id",
    help="Custom worker ID (auto-generated if not provided)"
)
@click.option(
    "--log-level", "-l",
    type=click.Choice(["DEBUG", "INFO", "WARNING", "ERROR"], case_sensitive=False),
    default="INFO",
    help="Logging level"
)
@click.option(
    "--max-documents", "-m",
    type=int,
    help="Maximum number of documents to process before stopping"
)
@click.option(
    "--discovery-interval", "-d",
    type=int,
    default=86400,  # 1 day = 86400 seconds
    help="Default interval in seconds between discovery cycles (default: 86400 = 1 day)"
)
def main(config, worker_id, log_level, max_documents, discovery_interval):
    """Go-Doc-Go Simple Document Worker (New Architecture).

    Process documents using a distributed, queue-based system with automatic
    leader election and job coordination.

    \b
    Examples:
      # Run single worker with default config
      python -m go_doc_go.cli.worker

      # Run worker with custom config file
      python -m go_doc_go.cli.worker --config /path/to/config.yaml

      # Process limited number of documents
      python -m go_doc_go.cli.worker --max-documents 10

      # Run with custom worker ID and debug logging
      python -m go_doc_go.cli.worker --worker-id my-worker-01 --log-level DEBUG

    \b
    Environment Variables:
      GO_DOC_GO_CONFIG_PATH    Path to configuration file (default: ./config.yaml)
    """
    # Configure logging
    log_format = "%(asctime)s - %(name)s - %(levelname)s - %(message)s"
    log_level_obj = getattr(logging, log_level.upper())

    logging.basicConfig(
        level=log_level_obj,
        format=log_format
    )

    logger = logging.getLogger(__name__)

    try:
        # Determine config file path
        config_path = (
            config or
            os.environ.get('GO_DOC_GO_CONFIG_PATH') or
            './config.yaml'
        )

        if not os.path.exists(config_path):
            click.echo(f"Error: Configuration file not found: {config_path}", err=True)
            sys.exit(1)

        logger.info(f"Loading configuration from: {config_path}")
        config_obj = Config(config_path)

        # Create and run worker
        worker = SimpleDocumentWorker(
            config=config_obj,
            worker_id=worker_id,
            max_documents=max_documents,
            discovery_interval=discovery_interval
        )

        worker.run()

        logger.info("Worker completed successfully")

    except KeyboardInterrupt:
        click.echo("\nWorker interrupted by user")
        sys.exit(0)
    except Exception as e:
        logger.error(f"Worker failed: {e}", exc_info=True)
        click.echo(f"Error: Worker failed: {e}", err=True)
        sys.exit(1)


if __name__ == "__main__":
    main()