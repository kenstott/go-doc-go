"""
Integration tests for Phase 2 distributed work queue system.
Tests the complete integration between coordinators, workers, and the main pipeline.
"""

import os
import tempfile
import time
import threading
import uuid
from typing import Dict, Any
import yaml
import pytest

from go_doc_go import Config
from go_doc_go.main import ingest_documents
from go_doc_go.queue.coordinator import ProcessingCoordinator
from go_doc_go.queue.worker import DocumentWorker, WorkerManager
from go_doc_go.queue.work_queue import WorkQueue, RunCoordinator
from go_doc_go.content_source.factory import register_content_source, clear_content_source_registry
from go_doc_go.content_source.base import ContentSource


class MockContentSource(ContentSource):
    """Mock content source for testing."""
    
    def __init__(self, config: Dict[str, Any]):
        super().__init__(config)
        self.documents = config.get('documents', [])
    
    def list_documents(self):
        """Return list of mock documents."""
        return self.documents
    
    def get_document(self, document_id: str) -> Dict[str, Any]:
        """Get a specific document."""
        for doc in self.documents:
            if doc['id'] == document_id:
                return {
                    'id': doc['id'],
                    'content': doc.get('content', f'Content for {document_id}'),
                    'doc_type': doc.get('doc_type', 'text'),
                    'metadata': doc.get('metadata', {}),
                    'source': f"mock://{document_id}"
                }
        raise ValueError(f"Document not found: {document_id}")


@pytest.mark.integration
@pytest.mark.queue_lifecycle  
class TestPhase2Integration:
    """Test Phase 2 distributed processing integration."""
    
    @pytest.fixture
    def temp_config(self):
        """Create temporary configuration for distributed processing."""
        # Create test documents
        test_docs = [
            {
                'id': 'doc_001',
                'content': 'This is a test document with some content for processing.',
                'doc_type': 'text',
                'metadata': {'source': 'test'}
            },
            {
                'id': 'doc_002', 
                'content': 'Another document with different content and relationships.',
                'doc_type': 'text',
                'metadata': {'source': 'test'}
            },
            {
                'id': 'doc_003',
                'content': 'A third document to test parallel processing capabilities.',
                'doc_type': 'text',
                'metadata': {'source': 'test'}
            }
        ]
        
        # Create temporary config
        config_data = {
            'storage': {
                'backend': 'sqlite',
                'path': ':memory:'
            },
            'processing': {
                'mode': 'distributed'
            },
            'embedding': {
                'enabled': False  # Disable for faster testing
            },
            'relationship_detection': {
                'enabled': True,
                'similarity_threshold': 0.7
            },
            'content_sources': [
                {
                    'name': 'test_source',
                    'type': 'mock',  # Will be registered manually
                    'documents': test_docs
                }
            ],
            'logging': {
                'level': 'INFO'
            }
        }
        
        # Create temporary config file
        with tempfile.NamedTemporaryFile(mode='w', suffix='.yaml', delete=False) as f:
            yaml.dump(config_data, f)
            config_path = f.name
        
        try:
            # Register mock content source
            config = Config(config_path)
            mock_source = MockContentSource({'documents': test_docs})
            register_content_source('test_source', mock_source)
            
            yield config, test_docs
        finally:
            clear_content_source_registry()
            os.unlink(config_path)
    
    def test_distributed_coordinator_integration(self, temp_config):
        """Test complete distributed processing via coordinator."""
        config, test_docs = temp_config
        
        # Initialize database
        config.initialize_database()
        
        # Create processing coordinator
        coordinator = ProcessingCoordinator(config, coordinator_id="test_coordinator")
        
        # Run distributed processing
        stats = coordinator.coordinate_processing_run()
        
        # Verify results
        assert stats['run_id'] is not None
        assert stats['coordinator_id'] == "test_coordinator"
        assert stats['documents_queued'] == len(test_docs)
        
        # Since no workers are running, documents should remain pending
        # This tests the queuing/coordination phase
        db = config.get_document_database()
        work_queue = WorkQueue(db, "test_queue")
        
        queue_status = work_queue.get_queue_status(stats['run_id'])
        assert queue_status['pending'] == len(test_docs)
        assert queue_status['completed'] == 0
    
    @pytest.mark.skip("Requires queue infrastructure which is not compatible with SQLite")
    def test_worker_document_processing(self, temp_config):
    
    def test_multiple_workers_parallel_processing(self, temp_config, real_db):
        """Test parallel processing with multiple workers."""
        config, test_docs = temp_config
        
        # Add more documents for better parallel testing
        additional_docs = [
            {'id': f'doc_{i:03d}', 'content': f'Document {i} content', 'doc_type': 'text'}
            for i in range(4, 10)
        ]
        test_docs.extend(additional_docs)
        
        # Update mock source
        mock_source = MockContentSource({'documents': test_docs})
        register_content_source('test_source', mock_source)
        
        # Initialize database
        config.initialize_database()
        
        # Queue documents
        coordinator = ProcessingCoordinator(config, coordinator_id="test_coordinator")
        run_id = RunCoordinator.get_run_id_from_config(config.config)
        
        run_stats = coordinator._discover_and_queue_documents(
            config.get_content_sources(),
            run_id
        )
        
        assert run_stats['documents_queued'] == len(test_docs)
        
        # Create multiple workers
        config.config['processing']['mode'] = 'worker'
        num_workers = 3
        
        manager = WorkerManager(config, num_workers=num_workers)
        
        # Process documents with multiple workers
        start_time = time.time()
        combined_stats = manager.start_all()
        processing_time = time.time() - start_time
        
        # Verify results
        assert combined_stats['total_workers'] == num_workers
        assert combined_stats['documents_processed'] == len(test_docs)
        assert combined_stats['documents_failed'] == 0
        assert combined_stats['elements_created'] > 0
        assert processing_time < 60  # Multiple workers should be faster
        
        # Verify no duplicate processing (all documents processed exactly once)
        db = config.get_document_database()
        processed_docs = []
        for doc_info in test_docs:
            try:
                doc = db.get_document(doc_info['id'])
                if doc:
                    processed_docs.append(doc_info['id'])
            except:
                pass
        
        assert len(processed_docs) == len(test_docs)
        assert len(set(processed_docs)) == len(processed_docs)  # No duplicates
    
    def test_end_to_end_distributed_processing(self, temp_config, real_db):
        """Test complete end-to-end distributed processing workflow."""
        config, test_docs = temp_config
        
        # Initialize database
        config.initialize_database()
        
        # Test using main ingest_documents interface with distributed mode
        config.config['processing']['mode'] = 'distributed'
        
        # Run in background thread to simulate coordinator
        coordination_stats = None
        coordination_error = None
        
        def run_coordinator():
            nonlocal coordination_stats, coordination_error
            try:
                coordination_stats = ingest_documents(
                    config=config, 
                    processing_mode='distributed'
                )
            except Exception as e:
                coordination_error = e
        
        # Start coordinator in background
        coordinator_thread = threading.Thread(target=run_coordinator)
        coordinator_thread.start()
        
        # Wait for documents to be queued (small delay)
        time.sleep(2)
        
        # Start workers to process the queued documents
        config.config['processing']['mode'] = 'worker'
        worker = DocumentWorker(config, worker_id="e2e_worker")
        
        worker_stats = worker.start()
        
        # Wait for coordinator to complete
        coordinator_thread.join(timeout=30)
        
        # Verify results
        assert coordination_error is None, f"Coordinator error: {coordination_error}"
        assert coordination_stats is not None
        assert coordination_stats['documents_queued'] == len(test_docs)
        
        assert worker_stats['documents_processed'] == len(test_docs)
        assert worker_stats['documents_failed'] == 0
        
        # Verify all documents are processed and stored
        db = config.get_document_database()
        all_docs = db.get_all_documents()
        stored_doc_ids = {doc['doc_id'] for doc in all_docs}
        expected_doc_ids = {doc['id'] for doc in test_docs}
        
        assert stored_doc_ids.issuperset(expected_doc_ids)
    
    def test_configuration_modes_routing(self, temp_config, real_db):
        """Test that different processing modes route correctly."""
        config, test_docs = temp_config
        
        # Initialize database
        config.initialize_database()
        
        # Test single mode (should process directly)
        config.config['processing']['mode'] = 'single'
        single_stats = ingest_documents(config, processing_mode='single')
        
        assert single_stats['processed_docs'] == len(test_docs)
        assert 'documents_queued' not in single_stats  # Single mode doesn't queue
        
        # Clear database for next test
        db = config.get_document_database()
        for doc_info in test_docs:
            try:
                # Note: Would need actual delete method in production
                pass
            except:
                pass
        
        # Test distributed mode routing
        config.config['processing']['mode'] = 'distributed'
        distributed_stats = ingest_documents(config, processing_mode='distributed')
        
        assert 'documents_queued' in distributed_stats
        assert distributed_stats['documents_queued'] == len(test_docs)
        
        # Test explicit mode override
        explicit_stats = ingest_documents(config, processing_mode='single')
        assert 'processed_docs' in explicit_stats  # Should use single despite config
    
    @pytest.mark.performance
    def test_distributed_processing_performance(self, temp_config, real_db):
        """Test performance characteristics of distributed processing."""
        config, _ = temp_config
        
        # Create larger document set for performance testing
        large_doc_set = [
            {
                'id': f'perf_doc_{i:04d}',
                'content': f'Performance test document {i} with substantial content to process. ' * 10,
                'doc_type': 'text',
                'metadata': {'batch': 'performance_test'}
            }
            for i in range(20)  # 20 documents
        ]
        
        # Update mock source
        mock_source = MockContentSource({'documents': large_doc_set})
        register_content_source('test_source', mock_source)
        
        # Initialize database
        config.initialize_database()
        
        # Test coordination performance
        coordinator = ProcessingCoordinator(config, coordinator_id="perf_coordinator")
        
        coord_start = time.time()
        run_stats = coordinator._discover_and_queue_documents(
            config.get_content_sources(),
            RunCoordinator.get_run_id_from_config(config.config)
        )
        coord_time = time.time() - coord_start
        
        # Coordination should be fast
        assert coord_time < 5.0, f"Document queuing took {coord_time}s, should be <5s"
        assert run_stats['documents_queued'] == len(large_doc_set)
        
        # Test worker processing performance  
        config.config['processing']['mode'] = 'worker'
        worker = DocumentWorker(config, worker_id="perf_worker")
        
        worker_start = time.time()
        worker_stats = worker.start()
        worker_time = time.time() - worker_start
        
        # Processing should complete in reasonable time
        assert worker_time < 30.0, f"Document processing took {worker_time}s, should be <30s"
        assert worker_stats['documents_processed'] == len(large_doc_set)
        
        # Calculate throughput
        throughput = len(large_doc_set) / worker_time
        assert throughput > 0.5, f"Throughput {throughput:.2f} docs/sec is too low"
        
        print(f"Performance results:")
        print(f"  Coordination time: {coord_time:.2f}s")
        print(f"  Processing time: {worker_time:.2f}s") 
        print(f"  Throughput: {throughput:.2f} docs/sec")