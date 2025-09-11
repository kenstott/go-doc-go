"""
Unit tests for Phase 2 distributed work queue system components.
Tests the core classes and functionality without database dependencies.
"""

import pytest
from unittest.mock import Mock, patch, MagicMock
import tempfile
import os
import yaml
from typing import Dict, Any

from go_doc_go import Config
from go_doc_go.queue.coordinator import ProcessingCoordinator
from go_doc_go.queue.worker import DocumentWorker, WorkerManager
from go_doc_go.queue.document_processor import QueuedDocumentProcessor
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
    
    def fetch_document(self, document_id: str) -> Dict[str, Any]:
        """Fetch a document by ID."""
        return self.get_document(document_id)
    
    def has_changed(self, document_id: str, last_modified: str = None) -> bool:
        """Mock implementation - always return False for testing."""
        return False


@pytest.mark.unit
class TestPhase2Components:
    """Unit tests for Phase 2 components."""
    
    @pytest.fixture
    def mock_config(self):
        """Create mock configuration."""
        config_data = {
            'storage': {
                'backend': 'sqlite',
                'path': ':memory:'
            },
            'processing': {
                'mode': 'distributed'
            },
            'embedding': {
                'enabled': False
            },
            'relationship_detection': {
                'enabled': True
            },
            'content_sources': [
                {
                    'name': 'test_source',
                    'type': 'mock',
                    'documents': [
                        {'id': 'doc_1', 'content': 'Test content 1'},
                        {'id': 'doc_2', 'content': 'Test content 2'}
                    ]
                }
            ]
        }
        
        with tempfile.NamedTemporaryFile(mode='w', suffix='.yaml', delete=False) as f:
            yaml.dump(config_data, f)
            config_path = f.name
        
        try:
            config = Config(config_path)
            yield config
        finally:
            os.unlink(config_path)
    
    def test_processing_coordinator_initialization(self, mock_config):
        """Test ProcessingCoordinator initialization."""
        coordinator = ProcessingCoordinator(mock_config, coordinator_id="test_coord")
        
        assert coordinator.config == mock_config
        assert coordinator.coordinator_id == "test_coord"
        assert coordinator.db is None  # Not initialized yet
        assert coordinator.work_queue is None
        assert coordinator.run_coordinator is None
    
    def test_document_worker_initialization(self, mock_config):
        """Test DocumentWorker initialization."""
        worker = DocumentWorker(mock_config, worker_id="test_worker")
        
        assert worker.config == mock_config
        assert worker.worker_id == "test_worker"
        assert not worker.running
        assert not worker.shutdown_requested
        assert worker.stats['documents_processed'] == 0
    
    def test_worker_manager_initialization(self, mock_config):
        """Test WorkerManager initialization."""
        num_workers = 3
        manager = WorkerManager(mock_config, num_workers=num_workers)
        
        assert manager.config == mock_config
        assert manager.num_workers == num_workers
        assert len(manager.workers) == 0  # Not started yet
        assert len(manager.worker_threads) == 0
    
    @patch('go_doc_go.queue.coordinator.get_content_source')
    def test_coordinator_document_discovery(self, mock_get_source, mock_config):
        """Test coordinator document discovery without database."""
        # Setup mock content source
        mock_source = MockContentSource({
            'documents': [
                {'id': 'doc_1', 'content': 'Test 1'},
                {'id': 'doc_2', 'content': 'Test 2'}
            ]
        })
        mock_get_source.return_value = mock_source
        
        # Mock work queue and database
        mock_db = Mock()
        mock_work_queue = Mock()
        mock_work_queue.add_document.return_value = 'queue_id_123'
        
        coordinator = ProcessingCoordinator(mock_config, coordinator_id="test_coord")
        coordinator.db = mock_db
        coordinator.work_queue = mock_work_queue
        
        # Test document discovery
        source_configs = [{'name': 'test_source', 'type': 'mock'}]
        run_id = "test_run_123"
        
        stats = coordinator._discover_and_queue_documents(source_configs, run_id)
        
        # Verify results
        assert stats['documents_queued'] == 2
        assert stats['sources_processed'] == 1
        assert mock_work_queue.add_document.call_count == 2
    
    def test_queued_document_processor_initialization(self):
        """Test QueuedDocumentProcessor initialization."""
        mock_db = Mock()
        mock_work_queue = Mock()
        mock_relationship_detector = Mock()
        mock_embedding_generator = Mock()
        
        processor = QueuedDocumentProcessor(
            db=mock_db,
            work_queue=mock_work_queue,
            relationship_detector=mock_relationship_detector,
            embedding_generator=mock_embedding_generator
        )
        
        assert processor.db == mock_db
        assert processor.work_queue == mock_work_queue
        assert processor.relationship_detector == mock_relationship_detector
        assert processor.embedding_generator == mock_embedding_generator
    
    def test_run_coordinator_config_hash(self):
        """Test RunCoordinator config hash generation."""
        config1 = {
            'content_sources': [{'name': 'source1', 'type': 'file'}],
            'storage': {'backend': 'sqlite'}
        }
        config2 = {
            'content_sources': [{'name': 'source1', 'type': 'file'}],
            'storage': {'backend': 'sqlite'}
        }
        config3 = {
            'content_sources': [{'name': 'source2', 'type': 'file'}],
            'storage': {'backend': 'sqlite'}
        }
        
        hash1 = RunCoordinator.get_run_id_from_config(config1)
        hash2 = RunCoordinator.get_run_id_from_config(config2)
        hash3 = RunCoordinator.get_run_id_from_config(config3)
        
        # Same config should produce same hash
        assert hash1 == hash2
        # Different config should produce different hash
        assert hash1 != hash3
        # Hashes should be valid format (16-character hex strings)
        assert len(hash1) == 16
        assert len(hash2) == 16
        assert len(hash3) == 16
        # Should be valid hex
        int(hash1, 16)
        int(hash2, 16)
        int(hash3, 16)
    
    def test_content_source_registry(self):
        """Test content source registry functionality."""
        # Clear registry first
        clear_content_source_registry()
        
        # Create mock source
        mock_source = MockContentSource({'documents': []})
        
        # Register source
        register_content_source('test_source', mock_source)
        
        # Retrieve source
        from go_doc_go.content_source.factory import get_content_source_by_name
        retrieved = get_content_source_by_name('test_source')
        
        assert retrieved == mock_source
        
        # Test error for non-existent source
        with pytest.raises(ValueError, match="Content source 'nonexistent' not registered"):
            get_content_source_by_name('nonexistent')
        
        # Clean up
        clear_content_source_registry()
    
    def test_main_processing_mode_routing(self, mock_config):
        """Test main.py processing mode routing."""
        from go_doc_go.main import ingest_documents
        
        # Test explicit mode override
        with patch('go_doc_go.main._ingest_documents_single') as mock_single, \
             patch('go_doc_go.main._ingest_documents_distributed') as mock_distributed, \
             patch('go_doc_go.main._ingest_documents_worker') as mock_worker:
            
            mock_single.return_value = {'mode': 'single'}
            mock_distributed.return_value = {'mode': 'distributed'}  
            mock_worker.return_value = {'mode': 'worker'}
            
            # Test single mode
            result = ingest_documents(mock_config, processing_mode='single')
            assert result['mode'] == 'single'
            mock_single.assert_called_once()
            
            # Test distributed mode
            result = ingest_documents(mock_config, processing_mode='distributed')
            assert result['mode'] == 'distributed'
            mock_distributed.assert_called_once()
            
            # Test worker mode
            result = ingest_documents(mock_config, processing_mode='worker')
            assert result['mode'] == 'worker'
            mock_worker.assert_called_once()


@pytest.mark.unit
class TestPhase2ErrorHandling:
    """Test error handling in Phase 2 components."""
    
    def test_coordinator_error_handling_invalid_source(self):
        """Test coordinator handles invalid content sources gracefully."""
        mock_config = Mock()
        coordinator = ProcessingCoordinator(mock_config, coordinator_id="test_coord")
        
        # Mock components
        coordinator.db = Mock()
        coordinator.work_queue = Mock()
        
        # Test with invalid source config (missing required fields)
        invalid_source_configs = [
            {'name': 'bad_source'}  # Missing 'type'
        ]
        
        with patch('go_doc_go.queue.coordinator.get_content_source') as mock_get_source:
            mock_get_source.side_effect = ValueError("Invalid source type")
            
            stats = coordinator._discover_and_queue_documents(invalid_source_configs, "run_123")
            
            # Should handle error gracefully
            assert stats['documents_queued'] == 0
            assert stats['sources_processed'] == 1
            assert len(stats['source_details']) == 1
            assert 'error' in stats['source_details'][0]
    
    def test_worker_graceful_shutdown(self):
        """Test worker graceful shutdown handling."""
        mock_config = Mock()
        worker = DocumentWorker(mock_config, worker_id="test_worker")
        
        # Test stop method
        worker.stop()
        
        assert worker.shutdown_requested
        assert not worker.running
    
    def test_processor_document_processing_error(self):
        """Test document processor error handling."""
        mock_db = Mock()
        mock_work_queue = Mock()
        mock_relationship_detector = Mock()
        
        processor = QueuedDocumentProcessor(
            db=mock_db,
            work_queue=mock_work_queue,
            relationship_detector=mock_relationship_detector,
            embedding_generator=None
        )
        
        # Mock document claim with processing error
        mock_work_queue.claim_next_document.return_value = {
            'queue_id': 123,
            'doc_id': 'failing_doc',
            'source_name': 'test_source',
            'metadata': {}
        }
        
        # Mock content source that fails
        with patch('go_doc_go.content_source.factory.get_content_source_by_name') as mock_get_source:
            mock_source = Mock()
            mock_source.get_document.side_effect = Exception("Processing failed")
            mock_get_source.return_value = mock_source
            
            stats = processor.process_documents("run_123", max_documents=1)
            
            # Should handle error and mark document as failed
            assert stats['documents_processed'] == 0
            assert stats['documents_failed'] == 1
            mock_work_queue.mark_failed.assert_called_once()


@pytest.mark.performance
class TestPhase2Performance:
    """Performance tests for Phase 2 components."""
    
    def test_config_hash_performance(self):
        """Test config hash generation performance."""
        import time
        
        large_config = {
            'content_sources': [
                {'name': f'source_{i}', 'type': 'file', 'path': f'/path/{i}'}
                for i in range(100)
            ],
            'storage': {'backend': 'postgresql'},
            'embedding': {'model': 'large-model'}
        }
        
        start_time = time.time()
        for _ in range(100):
            RunCoordinator.get_run_id_from_config(large_config)
        elapsed = time.time() - start_time
        
        # Should be fast even for large configs
        assert elapsed < 1.0, f"Config hash generation too slow: {elapsed}s for 100 iterations"
    
    def test_content_source_registry_performance(self):
        """Test content source registry performance."""
        import time
        
        clear_content_source_registry()
        
        # Register many sources
        sources = []
        for i in range(100):
            source = MockContentSource({'documents': []})
            sources.append(source)
            register_content_source(f'source_{i}', source)
        
        # Test retrieval performance
        start_time = time.time()
        for i in range(100):
            from go_doc_go.content_source.factory import get_content_source_by_name
            retrieved = get_content_source_by_name(f'source_{i}')
            assert retrieved == sources[i]
        elapsed = time.time() - start_time
        
        # Should be fast
        assert elapsed < 0.1, f"Content source registry too slow: {elapsed}s for 100 lookups"
        
        clear_content_source_registry()