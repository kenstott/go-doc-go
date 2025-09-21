"""
Two-pass document processing system with multi-worker support.

This module provides a two-pass processing approach that separates document parsing
from embedding generation for optimal performance across heterogeneous hardware.

Pass 1: Multi-threaded document parsing and storage
Pass 2: Batch embedding generation with optional GPU acceleration
"""

import logging
import time
import threading
from concurrent.futures import ThreadPoolExecutor, as_completed
from typing import Dict, Any, List, Optional, Tuple
from datetime import datetime
import psutil
import os

from ..config import Config
from ..content_source.factory import get_content_source
from ..document_parser.factory import get_parser_for_content
from ..embeddings.factory import get_embedding_generator
from ..relationships import create_relationship_detector
from ..storage import get_document_database
from ..storage_adapters.factory import StorageFactory

logger = logging.getLogger(__name__)


class TwoPassProcessor:
    """
    Local two-pass processor for single-machine processing.
    Handles both parsing and embedding generation locally.
    """
    
    def __init__(self, config: Config):
        """
        Initialize two-pass processor.
        
        Args:
            config: Application configuration
        """
        self.config = config
        
        # Initialize dual storage architecture
        storage_config = config.config.get('storage', {})
        
        # Dual storage is MANDATORY
        if not ('job' in storage_config and 'analytics' in storage_config):
            raise ValueError("Dual storage configuration is REQUIRED. Must specify both 'job' and 'analytics' in storage config")
        
        # Initialize dual storage architecture
        self.job_storage, self.analytics_storage = StorageFactory.create_from_pipeline_config(config.config)
        logger.info(f"Initialized dual storage: job={storage_config['job']['type']}, "
                   f"analytics={storage_config['analytics']['type']}")
        
        # Generate run ID for this processing session
        import hashlib
        import json
        config_str = json.dumps(config.config.get('content_sources', []), sort_keys=True)
        self.run_id = hashlib.md5(config_str.encode()).hexdigest()
        
        # Generate worker ID for this processor instance
        import uuid
        self.worker_id = f"worker_{uuid.uuid4().hex[:8]}"
        
        # Get processing configuration
        proc_config = config.config.get('processing', {})
        
        # Parser configuration (Pass 1)
        self.parser_threads = proc_config.get('parser_threads', psutil.cpu_count() or 4)
        
        # Embedder configuration (Pass 2)
        self.embedder_threads = proc_config.get('embedder_threads', 2)
        self.embedding_batch_size = proc_config.get('embedding_batch_size', 1000)
        self.use_gpu = proc_config.get('use_gpu', True)
        
        # Initialize embedding generator if enabled
        # Use analytics backend configuration from registry to ensure consistency
        self.embedding_generator = None
        if config.is_embedding_enabled():
            self.embedding_generator = self._get_analytics_embedding_generator(config)
            logger.info(f"Initialized embedding generator: {type(self.embedding_generator).__name__}")
        
        # Initialize relationship detector if enabled
        self.relationship_detector = None
        if config.config.get('relationship_detection', {}).get('enabled', False):
            self.relationship_detector = create_relationship_detector(config, self.embedding_generator)
            logger.info(f"Initialized relationship detector: {type(self.relationship_detector).__name__}")
        
        # Track statistics
        self.stats = {
            'documents': 0,
            'elements': 0,
            'relationships': 0,
            'embeddings': 0,
            'errors': 0,
            'pass1_time': 0,
            'pass2_time': 0
        }
        
        logger.info(f"Initialized TwoPassProcessor with {self.parser_threads} parser threads, "
                   f"{self.embedder_threads} embedder threads")
    
    def _get_analytics_embedding_generator(self, config: Config):
        """
        Create embedding generator using analytics registry configuration.
        
        This ensures embedding generators in Pass 2 use the same model as configured
        in the analytics registry for the target backend.
        
        Args:
            config: Application configuration
            
        Returns:
            EmbeddingGenerator instance using analytics registry configuration
        """
        # Get the analytics backend name from configuration
        storage_config = config.config.get('storage', {})
        analytics_config = storage_config.get('analytics')
        
        if not analytics_config:
            logger.warning("No analytics storage configured, falling back to global embedding config")
            return get_embedding_generator(config)
        
        # Handle both direct config and registry name formats
        analytics_backend_name = None
        if isinstance(analytics_config, str):
            # Analytics specified as registry name (e.g., "parquet_duckdb")
            analytics_backend_name = analytics_config
        elif isinstance(analytics_config, dict):
            # Analytics specified as direct config - look for type field
            backend_type = analytics_config.get('type', '')
            # Map common backend types to likely registry names
            type_to_registry = {
                'parquet': 'parquet_duckdb',
                'elasticsearch': 'elasticsearch',
                'mongodb': 'mongodb',
                'neo4j': 'neo4j',
                'solr': 'solr'
            }
            analytics_backend_name = type_to_registry.get(backend_type)
        
        if not analytics_backend_name:
            logger.warning("Could not determine analytics backend name, falling back to global embedding config")
            return get_embedding_generator(config)
        
        # Get analytics registry configuration
        backends = config.list_analytics_backends()
        backend_config = backends.get(analytics_backend_name)
        
        if not backend_config:
            logger.warning(f"Analytics backend '{analytics_backend_name}' not found in registry, "
                         f"falling back to global embedding config")
            return get_embedding_generator(config)
        
        # Get embedding configuration from backend
        embedding_config = backend_config.get('embedding', {})
        
        if not embedding_config:
            logger.warning(f"No embedding configuration found for backend '{analytics_backend_name}', "
                         f"falling back to global embedding config")
            return get_embedding_generator(config)
        
        # Create embedder using analytics registry configuration
        try:
            from ..embeddings.factory import get_embedder_from_analytics_registry
            embedder = get_embedder_from_analytics_registry(analytics_backend_name, embedding_config, config)
            logger.info(f"Created embedding generator for analytics backend '{analytics_backend_name}' "
                       f"using model: {embedding_config.get('model', 'unknown')}")
            return embedder
            
        except ImportError as e:
            logger.error(f"Failed to import embedding factory: {e}")
            return get_embedding_generator(config)
        except Exception as e:
            logger.error(f"Failed to create embedder for analytics backend '{analytics_backend_name}': {e}")
            return get_embedding_generator(config)
    
    def process_local(self, source_configs: Optional[List[Dict[str, Any]]] = None,
                     max_link_depth: Optional[int] = None, 
                     progress_callback: Optional[callable] = None) -> Dict[str, Any]:
        """
        DEPRECATED: Local processing mode creates divergent code paths.
        
        This method now redirects to queue-based processing to ensure
        uniform document processing through the work queue.
        
        Args:
            source_configs: Content source configurations
            max_link_depth: Maximum depth for following links
            progress_callback: Optional callback function for progress updates
            
        Returns:
            Processing statistics
        """
        logger.warning("process_local is DEPRECATED - redirecting to queue-based processing")
        logger.info("All documents will be enqueued and processed uniformly through the work queue")
        
        # Import here to avoid circular dependency
        from ..main import ingest_documents
        
        # Use ingest_documents with coordinator mode to enqueue and process
        # This ensures all processing goes through the same queue-based pathway
        return ingest_documents(
            config=self.config,
            source_configs=source_configs,
            max_link_depth=max_link_depth,
            processing_mode='coordinator',  # Enqueue documents then process through queue
            progress_callback=progress_callback
        )
    
    def _run_pass1(self, source_configs: List[Dict[str, Any]], 
                   max_link_depth: Optional[int] = None):
        """
        Pass 1: Multi-threaded document parsing and storage.
        """
        logger.info(f"Starting Pass 1 with {self.parser_threads} threads")
        
        # Create content sources and store them for reuse
        self.content_sources = {}
        
        # Collect all documents to process
        all_documents = []
        for source_config in source_configs:
            try:
                source = get_content_source(source_config)
                source_name = source_config.get('name', 'unknown')
                
                # Store source for later use when fetching content
                self.content_sources[source_name] = source
                
                # Set max link depth
                if max_link_depth is not None and hasattr(source, 'max_link_depth'):
                    source.max_link_depth = max_link_depth
                
                # Get documents from source
                documents_found = 0
                documents_queued = 0
                documents_skipped = 0
                
                for doc in source.list_documents():
                    documents_found += 1
                    doc['_source_name'] = source_name
                    
                    # Generate compound document key: filename::timestamp  
                    doc_metadata = doc.get('metadata', {})
                    last_modified = doc_metadata.get('last_modified')
                    
                    logger.debug(f"Processing document: {doc['id']}, metadata: {doc_metadata}")
                    
                    if last_modified is not None:
                        # Create compound key with timestamp
                        compound_doc_id = f"{doc['id']}::{int(last_modified)}"
                        logger.debug(f"Created compound doc_id: {compound_doc_id}")
                    else:
                        # Fallback to original doc_id if no timestamp available
                        compound_doc_id = doc['id']
                        logger.warning(f"No last_modified timestamp for {doc['id']}, using basic doc_id")
                    
                    logger.debug(f"About to check if document already processed: {compound_doc_id}")
                    
                    # Check if this exact document version already exists in analytics storage
                    if self._document_already_processed(compound_doc_id):
                        logger.info(f"SKIPPING already processed document: {compound_doc_id}")
                        documents_skipped += 1
                        continue
                    
                    logger.debug(f"Document {compound_doc_id} will be processed (not found in analytics storage)")
                    
                    # Use compound doc_id for processing while preserving original  
                    doc['_original_id'] = doc['id']  # Preserve original for content source operations
                    doc['id'] = compound_doc_id     # Use compound ID for processing
                    all_documents.append(doc)
                    documents_queued += 1
                
                logger.info(f"Completed source {source_name}: {documents_queued}/{documents_found} queued, {documents_skipped} skipped (already processed)")
                    
            except Exception as e:
                logger.error(f"Error getting documents from source: {e}")
                self.stats['errors'] += 1
        
        logger.info(f"Collected {len(all_documents)} documents to process")
        
        # Store documents total for use in all callbacks
        self.documents_total = len(all_documents)
        
        # Call initial progress callback with actual document total
        if hasattr(self, 'progress_callback') and self.progress_callback:
            self.progress_callback({
                'documents': 0,
                'elements': 0,
                'relationships': 0,
                'embeddings': 0,
                'documents_parsed': 0,
                'documents_embedded': 0,
                'documents_total': len(all_documents),  # Now we know the actual total!
                'pass': 'parsing_started',
                'parsing_complete': False,
                'embedding_complete': False
            })
        
        # Process documents in parallel
        with ThreadPoolExecutor(max_workers=self.parser_threads) as executor:
            futures = []
            for doc in all_documents:
                future = executor.submit(self._process_document, doc)
                futures.append(future)
            
            # Wait for completion and collect results
            for future in as_completed(futures):
                try:
                    future.result()
                except Exception as e:
                    logger.error(f"Error processing document: {e}")
                    self.stats['errors'] += 1
    
    def _process_document(self, doc: Dict[str, Any]):
        """
        Process a single document (parsing, storage, relationships).
        """
        try:
            # If document doesn't have content, fetch it
            if 'content' not in doc and 'binary_path' not in doc:
                # Get the source name to fetch from correct source
                source_name = doc.get('_source_name')
                if source_name and hasattr(self, 'content_sources'):
                    source = self.content_sources.get(source_name)
                    if source:
                        doc_id = doc.get('id')
                        if doc_id:
                            # Extract original doc_id for content source operations
                            # Compound doc_id format: "original_path::timestamp"
                            if '::' in doc_id:
                                original_doc_id = doc_id.split('::')[0]
                                logger.debug(f"Extracted original doc_id: {original_doc_id} from compound: {doc_id}")
                            else:
                                original_doc_id = doc_id
                            
                            # Fetch the full document with content using original doc_id
                            full_doc = source.fetch_document(original_doc_id)
                            # Preserve compound doc_id before merging
                            compound_doc_id = doc_id
                            # Merge with existing metadata
                            doc.update(full_doc)
                            # Restore compound doc_id after merge
                            doc['id'] = compound_doc_id
            
            # Parse document
            parser = get_parser_for_content(doc)
            if not parser:
                logger.warning(f"No parser for document type: {doc.get('doc_type')}")
                return
            
            result = parser.parse(doc)
            
            # Store using dual storage architecture (MANDATORY)
            # Note: Local processing doesn't use job queue, so no mark_completed needed
            # Job storage is only for distributed queue-based processing
            
            # Analytics storage for permanent data
            # Append to analytics storage
            docs_written = self.analytics_storage.append_documents(
                [result['document']], self.run_id
            )
            elems_written = self.analytics_storage.append_elements(
                result['elements'], self.run_id
            )
            rels_written = self.analytics_storage.append_relationships(
                result.get('relationships', []), self.run_id
            )
            
            logger.info(f"Stored to analytics storage for {doc_id}: {docs_written} docs, "
                       f"{elems_written} elements, {rels_written} relationships")
            
            # Update statistics (thread-safe)
            self.stats['documents'] += 1
            self.stats['elements'] += len(result['elements'])
            self.stats['relationships'] += len(result.get('relationships', []))
            
            # Call progress callback if provided
            if hasattr(self, 'progress_callback') and self.progress_callback:
                self.progress_callback({
                    'documents': self.stats['documents'],
                    'elements': self.stats['elements'], 
                    'relationships': self.stats['relationships'],
                    'embeddings': self.stats['embeddings'],
                    'documents_parsed': self.stats['documents'],
                    'documents_embedded': 0,  # Still in Pass 1
                    'documents_total': getattr(self, 'documents_total', self.stats['documents']),
                    'pass': 'parsing',
                    'parsing_complete': False,
                    'embedding_complete': False
                })
            
            # Detect additional relationships if enabled
            if self.relationship_detector:
                relationships = self.relationship_detector.detect_relationships(
                    result['document'],
                    result['elements'],
                    result.get('links', [])
                )
                if relationships:
                    # Append to analytics storage
                    self.analytics_storage.append_relationships(relationships, self.run_id)
                    self.stats['relationships'] += len(relationships)
            
            logger.debug(f"Processed document: {doc.get('id', 'unknown')}")
            
        except Exception as e:
            logger.error(f"Error processing document {doc.get('id')}: {e}")
            self.stats['errors'] += 1
    
    def _document_already_processed(self, compound_doc_id: str) -> bool:
        """
        Check if a document (with compound ID) has already been processed.
        
        Args:
            compound_doc_id: Document ID in format "filename::timestamp"
            
        Returns:
            True if document has already been processed, False otherwise
        """
        try:
            logger.info(f"*** CHECKING IF DOCUMENT ALREADY PROCESSED: {compound_doc_id} ***")
            logger.debug(f"Analytics storage type: {type(self.analytics_storage).__name__}")
            
            # Check if any elements exist for this document in analytics storage
            elements = self.analytics_storage.get_document_elements(compound_doc_id)
            logger.debug(f"get_document_elements({compound_doc_id}) returned {len(elements)} elements")
            
            # If elements exist, document has been processed
            is_processed = len(elements) > 0
            if is_processed:
                logger.info(f"*** DOCUMENT {compound_doc_id} ALREADY PROCESSED ({len(elements)} elements found) ***")
                # Show first element as proof
                if elements:
                    first_element = elements[0]
                    logger.info(f"First element ID: {first_element.get('element_id', 'NO_ID')}")
            else:
                logger.info(f"*** DOCUMENT {compound_doc_id} NOT FOUND - WILL PROCESS ***")
            
            return is_processed
            
        except Exception as e:
            logger.warning(f"*** ERROR checking if document {compound_doc_id} already processed: {e} ***")
            import traceback
            logger.warning(f"Traceback: {traceback.format_exc()}")
            # If we can't check, assume it needs processing (safer)
            return False
    
    def _run_pass2(self):
        """
        Pass 2: Batch embedding generation for elements without embeddings.
        """
        logger.info(f"Starting Pass 2 with {self.embedder_threads} threads")
        
        # Get elements needing embeddings
        elements_to_embed = self._get_elements_without_embeddings()
        logger.info(f"Found {len(elements_to_embed)} elements needing embeddings")
        
        if not elements_to_embed:
            return
        
        # Process in batches
        for i in range(0, len(elements_to_embed), self.embedding_batch_size):
            batch = elements_to_embed[i:i + self.embedding_batch_size]
            self._process_embedding_batch(batch)
    
    def _get_elements_without_embeddings(self) -> List[Dict[str, Any]]:
        """
        Get elements that don't have embeddings yet.
        """
        # Read elements from analytics storage for this run
        import pandas as pd
        import glob
        import os
        
        # Get the elements path for this run
        elements_path = self.analytics_storage.get_partition_path(self.run_id, 'elements')
        embeddings_path = self.analytics_storage.get_partition_path(self.run_id, 'embeddings')
        
        # Read all element parquet files for this run
        element_files = glob.glob(os.path.join(elements_path, '*.parquet'))
        if not element_files:
            logger.warning(f"No element files found in {elements_path}")
            return []
        
        # Check if embeddings already exist
        embedding_files = glob.glob(os.path.join(embeddings_path, '*.parquet'))
        if embedding_files:
            logger.info(f"Found existing embeddings in {embeddings_path}, skipping generation")
            return []
        
        # Load all elements
        all_elements = []
        for file in element_files:
            try:
                df = pd.read_parquet(file)
                # Convert DataFrame rows to dictionaries
                elements = df.to_dict('records')
                all_elements.extend(elements)
            except Exception as e:
                logger.error(f"Error reading element file {file}: {e}")
        
        logger.info(f"Found {len(all_elements)} elements needing embeddings")
        return all_elements
    
    def _process_embedding_batch(self, elements: List[Dict[str, Any]]):
        """
        Process a batch of elements to generate embeddings using contextual embedding generation.
        
        This method now uses the proper generate_from_elements() method which:
        - Filters to only leaf elements (skips containers)
        - Builds contextual graphlets with parent hierarchy  
        - Generates embeddings from enriched content
        """
        try:
            if not elements:
                return
            
            # Use the contextual embedding generator's generate_from_elements method
            # This automatically handles:
            # 1. Leaf element filtering (skips containers)
            # 2. Contextual graphlet building (parent hierarchy + siblings + children)
            # 3. Cross-document relationships (if database provided)
            # 4. Token-aware context management
            embeddings_dict = self.embedding_generator.generate_from_elements(
                elements, 
                db=self.analytics_storage  # Provide database for cross-document context
            )
            
            if not embeddings_dict:
                logger.debug("No embeddings generated for this batch")
                return
            
            # Prepare embeddings for analytics storage (dual storage is mandatory)
            embedding_docs = []
            for element_id, embedding in embeddings_dict.items():
                # Find the element data for this element_id
                element_data = None
                for element in elements:
                    if element.get('element_id') == element_id:
                        element_data = element
                        break
                
                if element_data:
                    embedding_docs.append({
                        'element_id': element_id,
                        'embedding': embedding.tolist() if hasattr(embedding, 'tolist') else embedding,
                        'model': getattr(self.embedding_generator, 'model_name', 'unknown')
                    })
            
            # Append to analytics storage
            written = self.analytics_storage.append_embeddings(embedding_docs, self.run_id)
            self.stats['embeddings'] += written
            
            logger.debug(f"Generated {len(embeddings_dict)} contextual embeddings "
                        f"(filtered from {len(elements)} elements, wrote {written} to storage)")
                    
            # Call progress callback for embedding progress
            if hasattr(self, 'progress_callback') and self.progress_callback:
                # Estimate documents_embedded based on embeddings generated
                # Rough approximation: embeddings per document varies but estimate average
                estimated_docs_embedded = min(self.stats['embeddings'] // 10, self.stats['documents'])
                
                self.progress_callback({
                    'documents': self.stats['documents'],
                    'elements': self.stats['elements'],
                    'relationships': self.stats['relationships'],
                    'embeddings': self.stats['embeddings'],
                    'documents_parsed': self.stats['documents'],
                    'documents_embedded': estimated_docs_embedded,
                    'documents_total': getattr(self, 'documents_total', self.stats['documents']),
                    'pass': 'embedding',
                    'parsing_complete': True,
                    'embedding_complete': False
                })
                
        except Exception as e:
            logger.error(f"Error processing embedding batch: {e}")
            self.stats['errors'] += 1


class TwoPassWorker:
    """
    Distributed two-pass worker that processes documents from a work queue.
    Can be configured to focus on parsing (Pass 1), embedding (Pass 2), or both.
    """
    
    def __init__(self, config: Config, worker_id: Optional[str] = None):
        """
        Initialize two-pass worker.
        
        Args:
            config: Application configuration
            worker_id: Unique worker identifier
        """
        self.config = config
        self.worker_id = worker_id or f"worker-{os.getpid()}"
        
        # Dual storage is MANDATORY - get both storages
        storage_config = config.config.get('storage', {})
        if not ('job' in storage_config and 'analytics' in storage_config):
            raise ValueError(
                "Dual storage configuration is REQUIRED for workers. "
                "Must specify both 'job' and 'analytics' in storage config"
            )
        
        # Create storage adapters
        from ..storage_adapters.factory import StorageFactory
        self.job_storage, self.analytics_storage = StorageFactory.create_from_pipeline_config(config.config)
        
        # Get processing configuration
        proc_config = config.config.get('processing', {})
        
        # Worker role configuration
        self.worker_role = proc_config.get('worker_role', 'both')  # parser/embedder/both
        
        # Parser configuration (Pass 1)
        self.parser_threads = proc_config.get('parser_threads', psutil.cpu_count() or 4)
        
        # Embedder configuration (Pass 2)
        self.embedder_threads = proc_config.get('embedder_threads', 2)
        self.embedding_batch_size = proc_config.get('embedding_batch_size', 1000)
        self.use_gpu = proc_config.get('use_gpu', True)
        self.max_gpu_workers = proc_config.get('max_gpu_workers', 1)
        
        # Queue configuration
        self.queue_poll_interval = proc_config.get('queue_poll_interval', 1)
        self.claim_timeout = proc_config.get('claim_timeout', 300)
        
        # Initialize work queue with job storage
        from ..work_queue import WorkQueue, RunCoordinator
        self.work_queue = WorkQueue(self.job_storage, self.worker_id)
        self.coordinator = RunCoordinator(self.job_storage, self.worker_id)
        
        # Initialize embedding generator if needed
        self.embedding_generator = None
        if self.worker_role in ['embedder', 'both'] and config.is_embedding_enabled():
            self.embedding_generator = self._get_analytics_embedding_generator(config)
            logger.info(f"Worker {self.worker_id} initialized embedding generator")
        
        # Track statistics
        self.stats = {
            'documents_parsed': 0,
            'elements_created': 0,
            'embeddings_generated': 0,
            'errors': 0,
            'start_time': datetime.now()
        }
        
        logger.info(f"Initialized TwoPassWorker {self.worker_id} with role: {self.worker_role}")
    
    def _get_analytics_embedding_generator(self, config: Config):
        """
        Create embedding generator using analytics registry configuration.
        
        This ensures embedding generators in Pass 2 use the same model as configured
        in the analytics registry for the target backend.
        """
        # Get the analytics backend name from configuration
        storage_config = config.config.get('storage', {})
        analytics_config = storage_config.get('analytics')
        
        if not analytics_config:
            logger.warning("No analytics storage configured, falling back to global embedding config")
            return get_embedding_generator(config)
        
        # Handle both direct config and registry name formats
        analytics_backend_name = None
        embedding_config = None
        
        if isinstance(analytics_config, str):
            # Analytics specified as registry name (e.g., "parquet_duckdb")
            analytics_backend_name = analytics_config
            # Get the full backend configuration from the registry
            backend_config = config.get_analytics_backend(analytics_backend_name)
            if backend_config:
                # Extract embedding configuration from the backend config
                embedding_config = backend_config.get('embedding', {})
        elif isinstance(analytics_config, dict):
            # Analytics specified as direct config - look for type field
            backend_type = analytics_config.get('type', '')
            # Map common backend types to likely registry names
            type_to_registry = {
                'parquet': 'parquet_duckdb',
                'elasticsearch': 'elasticsearch',
                'mongodb': 'mongodb',
                'neo4j': 'neo4j',
                'solr': 'solr'
            }
            analytics_backend_name = type_to_registry.get(backend_type)
            if analytics_backend_name:
                # Try to get backend config from registry
                backend_config = config.get_analytics_backend(analytics_backend_name)
                if backend_config:
                    embedding_config = backend_config.get('embedding', {})
                else:
                    # Use embedding config from analytics config if available
                    embedding_config = analytics_config.get('embedding', {})
        
        if not analytics_backend_name:
            logger.warning(f"Could not determine analytics backend name from config: {analytics_config}")
            return get_embedding_generator(config)
        
        if not embedding_config:
            logger.warning(f"No embedding configuration found for analytics backend: {analytics_backend_name}")
            # Fall back to global embedding config
            embedding_config = config.config.get('embedding', {})
        
        # Import the factory function
        from ..embeddings.factory import get_embedder_from_analytics_registry
        
        try:
            # Create embedder using analytics registry configuration
            logger.info(f"Creating embedder for analytics backend: {analytics_backend_name}")
            return get_embedder_from_analytics_registry(analytics_backend_name, embedding_config, config)
        except Exception as e:
            logger.error(f"Failed to create embedder from analytics registry for {analytics_backend_name}: {e}")
            logger.info("Falling back to global embedding configuration")
            return get_embedding_generator(config)
    
    def run(self, progress_callback: Optional[callable] = None, run_id: Optional[str] = None) -> Dict[str, Any]:
        """
        Run the worker, processing documents from the queue.
        
        Args:
            progress_callback: Optional callback function for progress updates
            run_id: Specific run to process (if None, processes any available)
            
        Returns:
            Worker statistics
        """
        logger.info(f"Worker {self.worker_id} starting (role: {self.worker_role})")
        
        # Store progress callback
        self.progress_callback = progress_callback
        
        try:
            if self.worker_role == 'parser':
                self._run_parser_role(run_id)
            elif self.worker_role == 'embedder':
                self._run_embedder_role(run_id)
            else:  # both
                self._run_both_roles(run_id)
                
        except KeyboardInterrupt:
            logger.info(f"Worker {self.worker_id} interrupted by user")
        except Exception as e:
            logger.error(f"Worker {self.worker_id} error: {e}")
            self.stats['errors'] += 1
        
        # Calculate runtime
        self.stats['runtime_seconds'] = (datetime.now() - self.stats['start_time']).total_seconds()
        
        logger.info(f"Worker {self.worker_id} finished: {self.stats}")
        return self.stats
    
    def _run_parser_role(self, run_id: Optional[str] = None):
        """
        Run as a parser worker (Pass 1 only).
        """
        logger.info(f"Worker {self.worker_id} running as parser")
        
        with ThreadPoolExecutor(max_workers=self.parser_threads) as executor:
            while True:
                # Claim next document from queue
                doc = self.work_queue.claim_next_document(run_id)
                if not doc:
                    # No more documents to process
                    time.sleep(self.queue_poll_interval)
                    continue
                
                # Process document in thread pool
                future = executor.submit(self._process_queued_document, doc)
                
                try:
                    future.result(timeout=self.claim_timeout)
                except Exception as e:
                    logger.error(f"Error processing document {doc.get('doc_id')}: {e}")
                    self.work_queue.mark_failed(doc['queue_id'], str(e))
                    self.stats['errors'] += 1
    
    def _run_embedder_role(self, run_id: Optional[str] = None):
        """
        Run as an embedder worker (Pass 2 only).
        """
        logger.info(f"Worker {self.worker_id} running as embedder")
        
        # Get the actual run_id from config if not provided
        if not run_id:
            from ..work_queue.work_queue import RunCoordinator
            run_id = RunCoordinator.get_run_id_from_config(self.config.config)
            logger.info(f"Embedder using run_id from config: {run_id}")
        
        # Ensure the run exists before trying to use it
        if not self.coordinator.ensure_run_exists(run_id, self.config.config):
            logger.warning(f"Could not ensure run exists for embedder: {run_id}")
        
        # Check if we should be the GPU leader
        is_gpu_worker = self.use_gpu and self._has_gpu_available()
        
        while True:
            if is_gpu_worker:
                # Try to become GPU leader using the proper run_id
                if not self.coordinator.attempt_leader_election(run_id):
                    # Another worker is GPU leader, wait
                    time.sleep(self.queue_poll_interval * 5)
                    continue
            
            # Get batch of elements needing embeddings
            elements = self._claim_embedding_batch()
            if not elements:
                time.sleep(self.queue_poll_interval)
                continue
            
            # Process embedding batch
            self._process_embedding_batch(elements)
            
            # Renew leadership if GPU worker
            if is_gpu_worker:
                self.coordinator.renew_leadership(run_id)
    
    def _run_both_roles(self, run_id: Optional[str] = None):
        """
        Run as both parser and embedder (alternating between passes).
        """
        logger.info(f"Worker {self.worker_id} running both parser and embedder roles")
        
        # Run parser in one thread, embedder in another
        parser_thread = threading.Thread(target=self._run_parser_role, args=(run_id,))
        embedder_thread = threading.Thread(target=self._run_embedder_role, args=(run_id,))
        
        parser_thread.start()
        embedder_thread.start()
        
        # Wait for both to complete
        parser_thread.join()
        embedder_thread.join()
    
    def _process_queued_document(self, doc: Dict[str, Any]):
        """
        Process a document claimed from the queue.
        """
        try:
            # Get the actual document content
            source_config = {'name': doc['source_name'], 'type': doc['source_type']}
            source = get_content_source(source_config)
            
            # Parse document
            parser = get_parser_for_content(doc)
            if not parser:
                raise ValueError(f"No parser for document type: {doc.get('doc_type')}")
            
            result = parser.parse(doc)
            
            # Store in analytics storage (dual storage is mandatory)
            doc_id = doc.get('doc_id', 'unknown')
            logger.info(f"Worker {self.worker_id} storing document {doc_id} to analytics storage")

            docs_stored = self.analytics_storage.append_documents([result['document']], run_id=doc.get('run_id'))
            logger.info(f"Worker {self.worker_id} stored {docs_stored} document records for {doc_id}")

            elements_stored = self.analytics_storage.append_elements(result['elements'], run_id=doc.get('run_id'))
            logger.info(f"Worker {self.worker_id} stored {elements_stored}/{len(result['elements'])} elements for {doc_id}")

            if result.get('relationships'):
                rels_stored = self.analytics_storage.append_relationships(result['relationships'], run_id=doc.get('run_id'))
                logger.info(f"Worker {self.worker_id} stored {rels_stored}/{len(result['relationships'])} relationships for {doc_id}")
            
            # Mark as completed in queue
            self.work_queue.mark_completed(doc['queue_id'])
            
            # Update statistics
            self.stats['documents_parsed'] += 1
            self.stats['elements_created'] += len(result['elements'])
            
            logger.debug(f"Worker {self.worker_id} processed document: {doc.get('doc_id')}")
            
        except Exception as e:
            logger.error(f"Worker {self.worker_id} error processing {doc.get('doc_id')}: {e}")
            raise
    
    def _claim_embedding_batch(self) -> List[Dict[str, Any]]:
        """
        Claim a batch of elements needing embeddings.
        """
        # TODO: Implement claiming elements without embeddings from database
        # This would query for elements without embeddings and claim them atomically
        return []
    
    def _process_embedding_batch(self, elements: List[Dict[str, Any]]):
        """
        Process a batch of elements to generate embeddings using contextual embedding generation.
        
        This method uses the proper generate_from_elements() method which:
        - Filters to only leaf elements (skips containers)
        - Builds contextual graphlets with parent hierarchy  
        - Generates embeddings from enriched content
        """
        if not self.embedding_generator:
            return
        
        try:
            if not elements:
                return
            
            # Use the contextual embedding generator's generate_from_elements method
            # This automatically handles leaf element filtering and contextual graphlet building
            embeddings_dict = self.embedding_generator.generate_from_elements(
                elements, 
                db=self.analytics_storage  # Provide database for cross-document context
            )
            
            logger.debug(f"Worker {self.worker_id} generated {len(embeddings_dict)} contextual embeddings")
            
            # Convert to list format for storage
            embedding_docs = []
            for element_id, embedding in embeddings_dict.items():
                embedding_docs.append({
                    'element_id': element_id,
                    'embedding': embedding.tolist() if hasattr(embedding, 'tolist') else embedding,
                    'model': getattr(self.embedding_generator, 'model_name', 'unknown')
                })
            
            # Append to analytics storage
            written = self.analytics_storage.append_embeddings(embedding_docs, self.run_id)
            self.stats['embeddings_generated'] += written
                
        except Exception as e:
            logger.error(f"Worker {self.worker_id} error generating embeddings: {e}")
            self.stats['errors'] += 1
    
    def _has_gpu_available(self) -> bool:
        """
        Check if GPU is available for this worker.
        """
        try:
            import onnxruntime as ort
            providers = ort.get_available_providers()
            return any(p in providers for p in [
                'CUDAExecutionProvider',
                'CoreMLExecutionProvider',
                'DmlExecutionProvider'
            ])
        except:
            return False