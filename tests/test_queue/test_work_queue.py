"""
Unit tests for the work queue system.
"""

import json
import os
import threading
import time
import uuid
from concurrent.futures import ThreadPoolExecutor, as_completed
from datetime import datetime
from unittest.mock import Mock, MagicMock, patch

import pytest

from go_doc_go.work_queue.work_queue import WorkQueue, RunCoordinator
from go_doc_go.work_queue.migrations import create_schema, check_schema_exists, validate_schema


class MockDatabase:
    """Mock database for testing without PostgreSQL."""
    
    def __init__(self):
        self.data = {}
        self.lock = threading.Lock()
        self.transaction_active = False
        
    def transaction(self):
        """Context manager for transactions."""
        class TransactionContext:
            def __init__(self, db):
                self.db = db
                
            def __enter__(self):
                self.db.transaction_active = True
                return self
                
            def __exit__(self, exc_type, exc_val, exc_tb):
                self.db.transaction_active = False
                
        return TransactionContext(self)
    
    def execute(self, query, params=None):
        """Mock execute method."""
        # Simple mock - return None for most queries
        if "SELECT" in query and "document_queue" in query:
            return None  # No documents available
        if "INSERT" in query:
            return {'queue_id': 1}
        if "UPDATE" in query:
            return {'run_id': 'test_run_123'}
        return None
    
    def execute_raw(self, query):
        """Mock execute_raw for schema creation."""
        return None


@pytest.fixture
def mock_db():
    """Create a mock database."""
    return MockDatabase()


@pytest.fixture
def real_db():
    """
    Create a real PostgreSQL database connection for integration tests.
    Skip if PostgreSQL is not available.
    """
    pytest.importorskip("psycopg2")
    
    from go_doc_go.storage.postgres import PostgreSQLDocumentDatabase
    from .test_db_adapter import QueueDatabaseAdapter
    
    # Use test database
    config = {
        'host': os.environ.get('TEST_PG_HOST', 'localhost'),
        'port': int(os.environ.get('TEST_PG_PORT', 5432)),
        'database': os.environ.get('TEST_PG_DB', 'go_doc_go_test'),
        'user': os.environ.get('TEST_PG_USER', 'postgres'),
        'password': os.environ.get('TEST_PG_PASSWORD', 'postgres')
    }
    
    try:
        pg_db = PostgreSQLDocumentDatabase(config)
        pg_db.initialize()
        
        # Wrap with adapter
        db = QueueDatabaseAdapter(pg_db)
        
        # Create queue schema
        create_schema(db, force=True)
        
        yield db
        
        # Cleanup
        db.execute_raw("""
            DROP TABLE IF EXISTS document_dependencies CASCADE;
            DROP TABLE IF EXISTS run_workers CASCADE;
            DROP TABLE IF EXISTS document_queue CASCADE;
            DROP TABLE IF EXISTS processing_runs CASCADE;
        """)
        db.close()
        
    except Exception as e:
        pytest.skip(f"PostgreSQL not available: {e}")


class TestRunCoordinator:
    """Test the RunCoordinator class."""
    
    def test_config_hash_deterministic(self):
        """Test that config hashing is deterministic."""
        config1 = {
            'content_sources': [{'name': 'test', 'path': '/data'}],
            'storage': {'backend': 'postgresql'},
            'embedding': {'enabled': True},
            'logging': {'level': 'INFO'}  # Should be ignored
        }
        
        config2 = {
            'logging': {'level': 'DEBUG'},  # Different but ignored
            'content_sources': [{'name': 'test', 'path': '/data'}],
            'embedding': {'enabled': True},
            'storage': {'backend': 'postgresql'}
        }
        
        run_id1 = RunCoordinator.get_run_id_from_config(config1)
        run_id2 = RunCoordinator.get_run_id_from_config(config2)
        
        assert run_id1 == run_id2
        assert len(run_id1) == 16
    
    def test_config_hash_different(self):
        """Test that different configs produce different hashes."""
        config1 = {
            'content_sources': [{'name': 'test1', 'path': '/data1'}]
        }
        
        config2 = {
            'content_sources': [{'name': 'test2', 'path': '/data2'}]
        }
        
        run_id1 = RunCoordinator.get_run_id_from_config(config1)
        run_id2 = RunCoordinator.get_run_id_from_config(config2)
        
        assert run_id1 != run_id2
    
    def test_ensure_run_exists(self, mock_db):
        """Test ensuring a run exists."""
        coordinator = RunCoordinator(mock_db)
        config = {'content_sources': []}
        run_id = RunCoordinator.get_run_id_from_config(config)
        
        result = coordinator.ensure_run_exists(run_id, config)
        
        assert result['run_id'] == run_id
        assert result['status'] == 'active'
    
    def test_register_worker(self, mock_db):
        """Test worker registration."""
        coordinator = RunCoordinator(mock_db)
        run_id = "test_run_123"
        worker_id = "worker_001"
        
        # Should not raise
        coordinator.register_worker(run_id, worker_id, {'version': '1.0.0'})


class TestWorkQueue:
    """Test the WorkQueue class."""
    
    def test_add_document(self, mock_db):
        """Test adding a document to the queue."""
        queue = WorkQueue(mock_db, "worker_001")
        
        queue_id = queue.add_document(
            doc_id="doc_001",
            source_name="test_source",
            run_id="test_run_123",
            metadata={'test': 'data'}
        )
        
        assert queue_id == 1
    
    def test_claim_next_document_empty(self, mock_db):
        """Test claiming when no documents are available."""
        queue = WorkQueue(mock_db, "worker_001")
        
        doc = queue.claim_next_document("test_run_123")
        
        assert doc is None
    
    def test_mark_completed(self, mock_db):
        """Test marking a document as completed."""
        queue = WorkQueue(mock_db, "worker_001")
        
        # Should not raise
        queue.mark_completed(1, content_hash="abc123")
    
    def test_mark_failed_with_retry(self, mock_db):
        """Test marking a document as failed with retry."""
        mock_db.execute = Mock(side_effect=[
            {'retry_count': 0, 'max_retries': 3, 'run_id': 'test_run'},
            None,
            None,
            None
        ])
        
        queue = WorkQueue(mock_db, "worker_001")
        queue.mark_failed(1, "Test error")
        
        # Verify retry was scheduled
        assert mock_db.execute.call_count == 4
    
    def test_mark_failed_max_retries(self, mock_db):
        """Test marking a document as failed after max retries."""
        mock_db.execute = Mock(side_effect=[
            {'retry_count': 3, 'max_retries': 3, 'run_id': 'test_run'},
            None,
            None,
            None
        ])
        
        queue = WorkQueue(mock_db, "worker_001")
        queue.mark_failed(1, "Test error")
        
        # Verify marked as failed, not retry
        calls = mock_db.execute.call_args_list
        assert "status = 'failed'" in str(calls[1])
    
    def test_add_linked_document(self, mock_db):
        """Test adding a linked document."""
        queue = WorkQueue(mock_db, "worker_001")
        
        result = queue.add_linked_document(
            parent_doc_id="parent_001",
            child_doc_id="child_001",
            source_name="test_source",
            run_id="test_run_123",
            link_depth=1
        )
        
        assert result is True
    
    def test_get_queue_status(self, mock_db):
        """Test getting queue status."""
        mock_db.execute = Mock(return_value={
            'pending': 10,
            'processing': 2,
            'completed': 50,
            'failed': 1,
            'retry': 3,
            'total': 66
        })
        
        queue = WorkQueue(mock_db, "worker_001")
        status = queue.get_queue_status("test_run_123")
        
        assert status['pending'] == 10
        assert status['completed'] == 50
        assert status['total'] == 66


@pytest.mark.integration
class TestWorkQueueIntegration:
    """Integration tests with real PostgreSQL."""
    
    def test_schema_creation(self, real_db):
        """Test that schema is created correctly."""
        assert check_schema_exists(real_db)
        assert validate_schema(real_db)
    
    def test_full_document_lifecycle(self, real_db):
        """Test complete document processing lifecycle."""
        # Create run
        coordinator = RunCoordinator(real_db)
        config = {'content_sources': [{'name': 'test'}]}
        run_id = RunCoordinator.get_run_id_from_config(config)
        coordinator.ensure_run_exists(run_id, config)
        
        # Create worker and queue
        worker_id = f"test_worker_{uuid.uuid4().hex[:8]}"
        queue = WorkQueue(real_db, worker_id)
        coordinator.register_worker(run_id, worker_id)
        
        # Add document
        queue_id = queue.add_document(
            doc_id="test_doc_001",
            source_name="test_source",
            run_id=run_id,
            metadata={'test': 'data'}
        )
        
        # Claim document
        doc = queue.claim_next_document(run_id)
        assert doc is not None
        assert doc['doc_id'] == "test_doc_001"
        assert doc['queue_id'] == queue_id
        
        # Mark completed
        queue.mark_completed(queue_id, content_hash="test_hash")
        
        # Verify status
        status = queue.get_queue_status(run_id)
        assert status['completed'] == 1
        assert status['pending'] == 0
    
    def test_atomic_claiming(self, real_db):
        """Test that claiming is atomic with multiple workers."""
        # Setup
        coordinator = RunCoordinator(real_db)
        config = {'content_sources': [{'name': 'test'}]}
        run_id = RunCoordinator.get_run_id_from_config(config)
        coordinator.ensure_run_exists(run_id, config)
        
        # Add multiple documents
        queue = WorkQueue(real_db, "setup_worker")
        doc_ids = []
        for i in range(10):
            doc_id = f"doc_{i:03d}"
            doc_ids.append(doc_id)
            queue.add_document(doc_id, "test", run_id)
        
        # Create multiple workers
        claimed_docs = []
        claim_lock = threading.Lock()
        
        def worker_claim(worker_num):
            worker_id = f"worker_{worker_num:02d}"
            worker_queue = WorkQueue(real_db, worker_id)
            coordinator.register_worker(run_id, worker_id)
            
            claimed = []
            while True:
                doc = worker_queue.claim_next_document(run_id)
                if not doc:
                    break
                claimed.append(doc['doc_id'])
                # Simulate processing
                time.sleep(0.01)
                worker_queue.mark_completed(doc['queue_id'])
            
            with claim_lock:
                claimed_docs.extend(claimed)
            
            return claimed
        
        # Run workers concurrently
        with ThreadPoolExecutor(max_workers=5) as executor:
            futures = [executor.submit(worker_claim, i) for i in range(5)]
            results = [f.result() for f in as_completed(futures)]
        
        # Verify no duplicates
        assert len(claimed_docs) == len(set(claimed_docs))
        assert set(claimed_docs) == set(doc_ids)
    
    def test_stale_work_reclaim(self, real_db):
        """Test that stale work is reclaimed."""
        # Setup
        coordinator = RunCoordinator(real_db)
        config = {'content_sources': [{'name': 'test'}]}
        run_id = RunCoordinator.get_run_id_from_config(config)
        coordinator.ensure_run_exists(run_id, config)
        
        # Worker 1 claims but doesn't complete
        worker1 = WorkQueue(real_db, "worker_001")
        worker1.claim_timeout = 1  # 1 second for testing
        
        queue_id = worker1.add_document("stale_doc", "test", run_id)
        doc1 = worker1.claim_next_document(run_id)
        assert doc1 is not None
        
        # Wait for timeout
        time.sleep(2)
        
        # Worker 2 should be able to claim it
        worker2 = WorkQueue(real_db, "worker_002")
        worker2.claim_timeout = 1
        
        doc2 = worker2.claim_next_document(run_id)
        assert doc2 is not None
        assert doc2['doc_id'] == "stale_doc"
    
    def test_retry_mechanism(self, real_db):
        """Test document retry mechanism."""
        # Setup
        coordinator = RunCoordinator(real_db)
        config = {'content_sources': [{'name': 'test'}]}
        run_id = RunCoordinator.get_run_id_from_config(config)
        coordinator.ensure_run_exists(run_id, config)
        
        # Add document
        queue = WorkQueue(real_db, "worker_001")
        queue_id = queue.add_document("retry_doc", "test", run_id)
        
        # Claim and fail
        doc = queue.claim_next_document(run_id)
        queue.mark_failed(doc['queue_id'], "First failure")
        
        # Check status - should be retry
        result = real_db.execute("""
            SELECT status, retry_count FROM document_queue
            WHERE queue_id = %s
        """, (queue_id,))
        
        assert result['status'] == 'retry'
        assert result['retry_count'] == 1
    
    def test_linked_document_handling(self, real_db):
        """Test linked document discovery and queueing."""
        # Setup
        coordinator = RunCoordinator(real_db)
        config = {'content_sources': [{'name': 'test'}]}
        run_id = RunCoordinator.get_run_id_from_config(config)
        coordinator.ensure_run_exists(run_id, config)
        
        # Add parent document
        queue = WorkQueue(real_db, "worker_001")
        parent_id = queue.add_document("parent_doc", "test", run_id)
        
        # Add linked documents
        queue.add_linked_document("parent_doc", "child_1", "test", run_id, 1)
        queue.add_linked_document("parent_doc", "child_2", "test", run_id, 1)
        
        # Verify they're in the queue
        status = queue.get_queue_status(run_id)
        assert status['total'] == 3
        
        # Verify dependencies recorded
        deps = real_db.execute("""
            SELECT COUNT(*) as count FROM document_dependencies
            WHERE run_id = %s AND parent_doc_id = 'parent_doc'
        """, (run_id,))
        
        assert deps['count'] == 2


@pytest.mark.performance
class TestWorkQueuePerformance:
    """Performance tests for the work queue."""
    
    def test_claim_performance(self, real_db):
        """Test that claiming is fast enough."""
        # Setup
        coordinator = RunCoordinator(real_db)
        config = {'content_sources': [{'name': 'test'}]}
        run_id = RunCoordinator.get_run_id_from_config(config)
        coordinator.ensure_run_exists(run_id, config)
        
        # Add many documents
        queue = WorkQueue(real_db, "setup_worker")
        for i in range(100):
            queue.add_document(f"perf_doc_{i:03d}", "test", run_id)
        
        # Measure claim time
        worker = WorkQueue(real_db, "perf_worker")
        
        claim_times = []
        for _ in range(10):
            start = time.time()
            doc = worker.claim_next_document(run_id)
            elapsed = time.time() - start
            claim_times.append(elapsed)
            
            if doc:
                worker.mark_completed(doc['queue_id'])
        
        avg_time = sum(claim_times) / len(claim_times)
        assert avg_time < 0.01  # Should be under 10ms
    
    def test_concurrent_throughput(self, real_db):
        """Test throughput with many concurrent workers."""
        # Setup
        coordinator = RunCoordinator(real_db)
        config = {'content_sources': [{'name': 'test'}]}
        run_id = RunCoordinator.get_run_id_from_config(config)
        coordinator.ensure_run_exists(run_id, config)
        
        # Add documents
        num_docs = 500
        queue = WorkQueue(real_db, "setup_worker")
        for i in range(num_docs):
            queue.add_document(f"throughput_doc_{i:04d}", "test", run_id)
        
        # Process with multiple workers
        processed_count = {'total': 0}
        process_lock = threading.Lock()
        
        def worker_process(worker_num):
            worker_id = f"throughput_worker_{worker_num:02d}"
            worker = WorkQueue(real_db, worker_id)
            count = 0
            
            while True:
                doc = worker.claim_next_document(run_id)
                if not doc:
                    break
                
                # Simulate minimal processing
                time.sleep(0.001)
                worker.mark_completed(doc['queue_id'])
                count += 1
            
            with process_lock:
                processed_count['total'] += count
            
            return count
        
        # Run workers
        start_time = time.time()
        with ThreadPoolExecutor(max_workers=10) as executor:
            futures = [executor.submit(worker_process, i) for i in range(10)]
            [f.result() for f in as_completed(futures)]
        elapsed = time.time() - start_time
        
        # Verify all processed
        assert processed_count['total'] == num_docs
        
        # Calculate throughput
        throughput = num_docs / elapsed
        print(f"Throughput: {throughput:.1f} docs/second")
        assert throughput > 50  # Should handle at least 50 docs/second