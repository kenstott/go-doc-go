"""
Integration tests for Phase 3 advanced features:
- Advanced monitoring and metrics
- Dead letter queue system  
- Distributed cross-document relationships
"""

import pytest
import tempfile
import time
from datetime import datetime, timedelta
from unittest.mock import Mock, patch

from go_doc_go.config import Config
from go_doc_go.work_queue.monitoring import MetricsCollector, AlertManager, WorkerMetrics, RunMetrics
from go_doc_go.work_queue.dead_letter import DeadLetterQueue, DeadLetterProcessor, FailurePattern
from go_doc_go.work_queue.work_queue import WorkQueue, RunCoordinator
from go_doc_go.work_queue.coordinator import ProcessingCoordinator
from go_doc_go.work_queue.document_processor import QueuedDocumentProcessor

from .test_db_adapter import QueueDatabaseAdapter


@pytest.fixture
def temp_config():
    """Create temporary configuration for testing."""
    config_data = {
        'storage': {
            'backend': 'sqlite',
            'path': ':memory:'
        },
        'embedding': {'enabled': False},
        'relationship_detection': {'enabled': True},
        'work_queue': {
            'claim_timeout': 300,
            'heartbeat_interval': 30,
            'max_retries': 3,
            'stale_threshold': 600
        }
    }
    
    with tempfile.NamedTemporaryFile(mode='w', suffix='.yaml', delete=False) as f:
        import yaml
        yaml.dump(config_data, f)
        config_path = f.name
    
    yield Config(config_path)
    
    import os
    os.unlink(config_path)


@pytest.fixture  
def setup_phase3_components(temp_config):
    """Set up Phase 3 components for testing."""
    # Initialize database
    db_raw = temp_config.get_document_database()
    db_raw.initialize()
    db = QueueDatabaseAdapter(db_raw)
    
    # Create schema
    from go_doc_go.work_queue.migrations import create_schema
    create_schema(db, force=True)
    
    # Initialize components
    work_queue = WorkQueue(db, "test_worker")
    coordinator = RunCoordinator(db)
    metrics_collector = MetricsCollector(db)
    alert_manager = AlertManager(metrics_collector)
    dead_letter_queue = DeadLetterQueue(db)
    dead_letter_processor = DeadLetterProcessor(db)
    
    yield {
        'db': db,
        'work_queue': work_queue,
        'coordinator': coordinator,
        'metrics_collector': metrics_collector,
        'alert_manager': alert_manager,
        'dead_letter_queue': dead_letter_queue,
        'dead_letter_processor': dead_letter_processor
    }
    
    db.close()


class TestAdvancedMonitoring:
    """Test advanced monitoring and metrics collection."""
    
    def test_worker_metrics_collection(self, setup_phase3_components):
        """Test collection of worker metrics."""
        components = setup_phase3_components
        db = components['db']
        work_queue = components['work_queue'] 
        metrics_collector = components['metrics_collector']
        
        # Create test run and simulate worker activity
        run_id = "test_run_metrics"
        work_queue.add_document("doc1", "test_source", run_id)
        work_queue.add_document("doc2", "test_source", run_id)
        
        # Simulate worker processing
        doc1 = work_queue.claim_next_document(run_id)
        time.sleep(0.1)  # Simulate processing time
        work_queue.mark_completed(doc1['queue_id'])
        
        doc2 = work_queue.claim_next_document(run_id)  
        work_queue.mark_failed(doc2['queue_id'], "Test error")
        
        # Collect metrics
        worker_metrics = metrics_collector.get_worker_metrics(run_id)
        
        assert len(worker_metrics) == 1
        worker = worker_metrics[0]
        assert worker.worker_id == "test_worker"
        assert worker.documents_processed == 1
        assert worker.documents_failed == 1
        assert worker.avg_processing_time_seconds > 0
    
    def test_run_metrics_collection(self, setup_phase3_components):
        """Test collection of run-level metrics."""
        components = setup_phase3_components
        metrics_collector = components['metrics_collector']
        work_queue = components['work_queue']
        
        # Create test run with various document states
        run_id = "test_run_overall" 
        work_queue.add_document("completed_doc", "test_source", run_id)
        work_queue.add_document("failed_doc", "test_source", run_id)
        work_queue.add_document("pending_doc", "test_source", run_id)
        
        # Process some documents
        completed = work_queue.claim_next_document(run_id)
        work_queue.mark_completed(completed['queue_id'])
        
        failed = work_queue.claim_next_document(run_id)
        work_queue.mark_failed(failed['queue_id'], "Test failure")
        
        # Collect run metrics
        run_metrics = metrics_collector.get_run_metrics(run_id)
        
        assert run_metrics is not None
        assert run_metrics.run_id == run_id
        assert run_metrics.documents_completed == 1
        assert run_metrics.documents_failed == 1
        assert run_metrics.documents_pending == 1
        assert run_metrics.total_workers >= 1
    
    def test_queue_health_monitoring(self, setup_phase3_components):
        """Test queue health metrics."""
        components = setup_phase3_components
        metrics_collector = components['metrics_collector']
        work_queue = components['work_queue']
        
        # Create test scenario
        run_id = "test_health_monitoring"
        
        # Add documents with different states
        for i in range(5):
            work_queue.add_document(f"doc_{i}", "test_source", run_id)
        
        # Process some, fail some, leave some pending
        for i in range(2):
            doc = work_queue.claim_next_document(run_id)
            work_queue.mark_completed(doc['queue_id'])
            
        for i in range(2):
            doc = work_queue.claim_next_document(run_id) 
            work_queue.mark_failed(doc['queue_id'], f"Error {i}")
        
        # Get health metrics
        health = metrics_collector.get_queue_health(run_id)
        
        assert health is not None
        assert health.total_documents == 5
        assert health.completed_count == 2
        assert health.failed_count == 2
        assert health.pending_count == 1
        assert health.overall_health in ['healthy', 'degraded', 'unhealthy']
    
    def test_alert_system(self, setup_phase3_components):
        """Test alert generation based on thresholds."""
        components = setup_phase3_components
        alert_manager = components['alert_manager']
        work_queue = components['work_queue']
        
        # Create scenario with high failure rate
        run_id = "test_alerts"
        
        # Add and fail many documents to trigger alerts
        for i in range(10):
            work_queue.add_document(f"fail_doc_{i}", "test_source", run_id)
        
        # Fail most documents
        for i in range(8):
            doc = work_queue.claim_next_document(run_id)
            work_queue.mark_failed(doc['queue_id'], f"Simulated failure {i}")
        
        # Process a few successfully
        for i in range(2):
            doc = work_queue.claim_next_document(run_id)
            work_queue.mark_completed(doc['queue_id'])
        
        # Check for alerts
        alerts = alert_manager.check_run_alerts(run_id)
        
        # Should have alerts for high failure rate
        failure_alerts = [a for a in alerts if 'failure' in a.message.lower()]
        assert len(failure_alerts) > 0


class TestDeadLetterQueue:
    """Test dead letter queue functionality."""
    
    def test_move_to_dead_letter(self, setup_phase3_components):
        """Test moving documents to dead letter queue."""
        components = setup_phase3_components
        work_queue = components['work_queue']
        dead_letter_queue = components['dead_letter_queue']
        
        # Create and fail a document
        run_id = "test_dead_letter"
        queue_id = work_queue.add_document("dead_doc", "test_source", run_id)
        
        # Move to dead letter
        success = dead_letter_queue.move_to_dead_letter(
            queue_id=queue_id,
            error_message="Critical parsing error",
            error_details={"error_type": "ParseError", "timestamp": time.time()}
        )
        
        assert success
        
        # Verify it's in dead letter queue
        items = dead_letter_queue.list_dead_letter_items()
        assert len(items) == 1
        assert items[0].doc_id == "dead_doc"
        assert items[0].error_message == "Critical parsing error"
    
    def test_retry_from_dead_letter(self, setup_phase3_components):
        """Test retrying documents from dead letter queue."""
        components = setup_phase3_components
        work_queue = components['work_queue']
        dead_letter_queue = components['dead_letter_queue']
        
        # Create document and move to dead letter
        run_id = "test_retry"
        queue_id = work_queue.add_document("retry_doc", "test_source", run_id)
        dead_letter_queue.move_to_dead_letter(queue_id, "Temporary error")
        
        # Verify in dead letter
        items = dead_letter_queue.list_dead_letter_items()
        assert len(items) == 1
        
        # Retry from dead letter
        success = dead_letter_queue.retry_from_dead_letter(queue_id)
        assert success
        
        # Should be back in processing queue
        items = dead_letter_queue.list_dead_letter_items()
        assert len(items) == 0
        
        # Should be claimable again
        claimed = work_queue.claim_next_document(run_id)
        assert claimed is not None
        assert claimed['doc_id'] == "retry_doc"
    
    def test_failure_pattern_analysis(self, setup_phase3_components):
        """Test analysis of failure patterns."""
        components = setup_phase3_components
        work_queue = components['work_queue']
        dead_letter_queue = components['dead_letter_queue']
        dead_letter_processor = components['dead_letter_processor']
        
        # Create multiple failures with similar error types
        run_id = "test_patterns"
        
        # Create parsing errors
        for i in range(3):
            queue_id = work_queue.add_document(f"parse_fail_{i}", "test_source", run_id)
            dead_letter_queue.move_to_dead_letter(
                queue_id, 
                f"Parsing error: Invalid PDF format in document {i}",
                {"error_type": "ParseError"}
            )
        
        # Create permission errors  
        for i in range(2):
            queue_id = work_queue.add_document(f"perm_fail_{i}", "test_source", run_id)
            dead_letter_queue.move_to_dead_letter(
                queue_id,
                f"Permission denied accessing document {i}", 
                {"error_type": "PermissionError"}
            )
        
        # Analyze patterns
        patterns = dead_letter_processor.analyze_failure_patterns(run_id)
        
        assert len(patterns) >= 2  # Should find at least 2 patterns
        
        # Check for parsing error pattern
        parse_patterns = [p for p in patterns if "parsing" in p.error_type.lower()]
        assert len(parse_patterns) >= 1
        assert parse_patterns[0].frequency >= 3
        
        # Check for permission error pattern
        perm_patterns = [p for p in patterns if "permission" in p.error_type.lower()]
        assert len(perm_patterns) >= 1
        assert perm_patterns[0].frequency >= 2


class TestDistributedCrossDocumentRelationships:
    """Test distributed cross-document relationship computation."""
    
    @patch('go_doc_go.main._compute_cross_document_container_relationships')
    def test_post_processing_coordination(self, mock_cross_doc_relationships, setup_phase3_components):
        """Test that post-processing correctly coordinates cross-document relationships."""
        mock_cross_doc_relationships.return_value = 5  # Mock 5 relationships created
        
        components = setup_phase3_components
        work_queue = components['work_queue']
        
        # Create a coordinator with mocked config
        mock_config = Mock()
        mock_config.config = {"test": "config"}
        mock_config.get_document_database.return_value = components['db']
        mock_config.get_content_sources.return_value = []
        mock_config.is_embedding_enabled.return_value = True
        
        coordinator = ProcessingCoordinator(mock_config, "test_coordinator")
        
        # Simulate completed processing run
        run_id = "test_cross_doc"
        
        # Add and complete some documents
        for i in range(3):
            queue_id = work_queue.add_document(f"doc_{i}", "test_source", run_id)
            claimed = work_queue.claim_next_document(run_id)
            work_queue.mark_completed(claimed['queue_id'])
        
        # Mock the coordinator components
        coordinator.db = components['db']
        coordinator.work_queue = work_queue
        coordinator.run_coordinator = components['coordinator']
        
        # Test post-processing
        post_stats = coordinator._perform_post_processing(run_id)
        
        # Verify cross-document relationships were computed
        assert post_stats['relationships_created'] == 5
        mock_cross_doc_relationships.assert_called_once()
        
        # Verify correct document IDs were passed
        call_args = mock_cross_doc_relationships.call_args
        doc_ids = call_args[0][1]  # Second argument should be document IDs
        assert len(doc_ids) == 3
        assert all(doc_id.startswith('doc_') for doc_id in doc_ids)


class TestIntegratedWorkflow:
    """Test integrated workflow with all Phase 3 features."""
    
    def test_full_phase3_workflow(self, setup_phase3_components):
        """Test complete workflow with monitoring, dead letter, and coordination."""
        components = setup_phase3_components
        work_queue = components['work_queue']
        dead_letter_queue = components['dead_letter_queue']
        metrics_collector = components['metrics_collector']
        
        run_id = "test_full_workflow"
        
        # Step 1: Add documents to queue
        document_ids = []
        for i in range(5):
            doc_id = f"workflow_doc_{i}"
            work_queue.add_document(doc_id, "test_source", run_id)
            document_ids.append(doc_id)
        
        # Step 2: Process documents with mixed outcomes
        # Process 2 successfully
        for i in range(2):
            doc = work_queue.claim_next_document(run_id)
            time.sleep(0.05)  # Simulate processing time
            work_queue.mark_completed(doc['queue_id'])
        
        # Fail 2 with retries
        for i in range(2):
            doc = work_queue.claim_next_document(run_id)
            work_queue.mark_failed(doc['queue_id'], f"Temporary error {i}")
        
        # Fail 1 critically (to dead letter)
        doc = work_queue.claim_next_document(run_id)
        dead_letter_queue.move_to_dead_letter(
            doc['queue_id'], 
            "Critical parsing error - corrupted file"
        )
        
        # Step 3: Check monitoring metrics
        run_metrics = metrics_collector.get_run_metrics(run_id)
        assert run_metrics.documents_completed == 2
        assert run_metrics.documents_failed == 2
        assert run_metrics.documents_pending == 0  # All processed
        
        worker_metrics = metrics_collector.get_worker_metrics(run_id)
        assert len(worker_metrics) == 1
        assert worker_metrics[0].documents_processed == 2
        assert worker_metrics[0].documents_failed == 2
        
        # Step 4: Check dead letter queue
        dead_items = dead_letter_queue.list_dead_letter_items(run_id)
        assert len(dead_items) == 1
        assert "corrupted file" in dead_items[0].error_message
        
        # Step 5: Check queue health
        health = metrics_collector.get_queue_health(run_id)
        assert health.total_documents == 5
        assert health.completed_count == 2
        assert health.failed_count == 2  # Regular failures, not including dead letter
        
        # Step 6: Test recovery from dead letter
        dead_item = dead_items[0]
        retry_success = dead_letter_queue.retry_from_dead_letter(dead_item.queue_id)
        assert retry_success
        
        # Should be claimable again
        retried_doc = work_queue.claim_next_document(run_id)
        assert retried_doc is not None
        work_queue.mark_completed(retried_doc['queue_id'])
        
        # Final metrics should show recovery
        final_run_metrics = metrics_collector.get_run_metrics(run_id)
        assert final_run_metrics.documents_completed == 3  # One more completed


# Performance test for Phase 3 components
@pytest.mark.performance
class TestPhase3Performance:
    """Performance tests for Phase 3 advanced features."""
    
    def test_metrics_collection_performance(self, setup_phase3_components):
        """Test that metrics collection doesn't significantly impact performance."""
        components = setup_phase3_components
        work_queue = components['work_queue']
        metrics_collector = components['metrics_collector']
        
        run_id = "perf_test"
        
        # Create large number of documents
        start_time = time.time()
        for i in range(100):
            work_queue.add_document(f"perf_doc_{i}", "test_source", run_id)
        queue_time = time.time() - start_time
        
        # Collect metrics multiple times
        start_time = time.time()
        for _ in range(10):
            run_metrics = metrics_collector.get_run_metrics(run_id)
            worker_metrics = metrics_collector.get_worker_metrics(run_id)
            health = metrics_collector.get_queue_health(run_id)
        metrics_time = time.time() - start_time
        
        # Metrics collection should be fast relative to queue operations
        assert metrics_time < queue_time * 2  # Should be at most 2x queue time
        assert metrics_time < 1.0  # Should complete in under 1 second
    
    def test_dead_letter_scalability(self, setup_phase3_components):
        """Test dead letter queue with large number of failures."""
        components = setup_phase3_components
        work_queue = components['work_queue']
        dead_letter_queue = components['dead_letter_queue']
        
        run_id = "scale_test"
        
        # Create and fail many documents
        start_time = time.time()
        for i in range(50):
            queue_id = work_queue.add_document(f"scale_doc_{i}", "test_source", run_id)
            dead_letter_queue.move_to_dead_letter(
                queue_id,
                f"Simulated error for doc {i}"
            )
        processing_time = time.time() - start_time
        
        # List operations should be reasonably fast
        start_time = time.time() 
        items = dead_letter_queue.list_dead_letter_items(run_id, limit=100)
        list_time = time.time() - start_time
        
        assert len(items) == 50
        assert processing_time < 5.0  # Should complete in under 5 seconds
        assert list_time < 1.0  # Listing should be under 1 second