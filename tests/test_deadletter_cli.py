"""
Test dead letter queue CLI functionality with the new architecture.
"""

import json
import os
import tempfile
import unittest
from pathlib import Path
from unittest.mock import MagicMock, patch

from go_doc_go.shared.simple_job_control.sqlite import SimpleSQLiteJobControlDB
from go_doc_go.config import Config


class TestDeadLetterCLI(unittest.TestCase):
    """Test dead letter queue CLI functionality."""

    def setUp(self):
        """Set up test environment."""
        self.test_dir = tempfile.mkdtemp()
        self.config_path = Path(self.test_dir) / "config.yaml"
        self.job_db_path = Path(self.test_dir) / "job_queue.db"

        # Create test configuration
        config_data = {
            'processing': {
                'job_control': {
                    'backend': 'sqlite',
                    'path': str(self.job_db_path),
                    'claim_timeout': 300,
                    'heartbeat_interval': 30,
                    'max_retries': 3
                }
            }
        }

        import yaml
        with open(self.config_path, 'w') as f:
            yaml.dump(config_data, f)

        # Initialize job control
        self.config = Config(str(self.config_path))
        self.job_control = SimpleSQLiteJobControlDB(self.config)

    def tearDown(self):
        """Clean up test environment."""
        import shutil
        if os.path.exists(self.test_dir):
            shutil.rmtree(self.test_dir)

    def test_dead_letter_queue_basics(self):
        """Test that we can track failed documents properly."""
        # Add a document to the queue
        doc_id = "test-doc-1"
        source = "test-source"
        metadata = {"url": "http://example.com/doc1"}

        self.job_control.enqueue_document(doc_id, source, metadata)

        # Register a worker
        worker_id = "test-worker-1"
        worker_info = {"hostname": "localhost", "pid": 12345}
        self.job_control.register_worker(worker_id, worker_info)

        # Claim and fail the document multiple times
        for attempt in range(4):  # Try one more than max retries
            # Claim the document
            claimed = self.job_control.claim_next_document(worker_id)

            if attempt < 3:  # First 3 attempts should succeed in claiming
                self.assertIsNotNone(claimed, f"Should be able to claim on attempt {attempt + 1}")
                self.assertEqual(claimed['doc_id'], doc_id)

                # Mark as failed
                error_msg = f"Processing error attempt {attempt + 1}"
                self.job_control.complete_document(
                    doc_id, worker_id, False, error_msg
                )
            else:
                # After max retries, should not be able to claim
                self.assertIsNone(claimed, "Should not be able to claim after max retries")

        # Check the document status
        with self.job_control._get_connection() as conn:
            cursor = conn.execute("""
                SELECT status, retry_count
                FROM document_queue
                WHERE doc_id = ?
            """, (doc_id,))
            row = cursor.fetchone()

        # Document should be 'pending' with retry_count >= max_retries (dead letter state)
        self.assertEqual(row['status'], 'pending', "Document should be pending")
        self.assertEqual(row['retry_count'], 3, "Should have max retry count")

    def test_failed_document_retrieval(self):
        """Test that we can retrieve failed documents."""
        # Add multiple documents and fail them
        failed_docs = []
        for i in range(5):
            doc_id = f"test-doc-{i}"
            source = "test-source"
            metadata = {"url": f"http://example.com/doc{i}"}

            self.job_control.enqueue_document(doc_id, source, metadata)

            # Register worker if not done
            worker_id = "test-worker"
            if i == 0:
                self.job_control.register_worker(
                    worker_id, {"hostname": "localhost", "pid": 12345}
                )

            # Claim and fail until max retries
            for _ in range(3):
                claimed = self.job_control.claim_next_document(worker_id)
                if claimed:
                    self.job_control.complete_document(
                        doc_id, worker_id, False, f"Error for doc {i}"
                    )

            failed_docs.append(doc_id)

        # Now we need a way to query failed documents
        # This demonstrates what's missing in the current implementation
        with self.job_control._get_connection() as conn:
            cursor = conn.execute("""
                SELECT doc_id, source, error_message, retry_count
                FROM document_queue
                WHERE status = 'failed' OR (status = 'pending' AND retry_count >= ?)
            """, (self.job_control.max_retries,))

            failed_results = cursor.fetchall()

        self.assertEqual(len(failed_results), 5, "Should have 5 failed documents")

        for row in failed_results:
            self.assertIn(row['doc_id'], failed_docs)
            self.assertEqual(row['retry_count'], 3)
            self.assertIsNotNone(row['error_message'])

    def test_retry_failed_document(self):
        """Test that we can retry a failed document."""
        doc_id = "retry-test-doc"
        source = "test-source"
        metadata = {"url": "http://example.com/retry"}

        self.job_control.enqueue_document(doc_id, source, metadata)

        # Register worker
        worker_id = "test-worker"
        self.job_control.register_worker(
            worker_id, {"hostname": "localhost", "pid": 12345}
        )

        # Fail the document max times
        for _ in range(3):
            claimed = self.job_control.claim_next_document(worker_id)
            if claimed:
                self.job_control.complete_document(
                    doc_id, worker_id, False, "Test failure"
                )

        # Document should now be failed
        claimed = self.job_control.claim_next_document(worker_id)
        self.assertIsNone(claimed, "Should not be able to claim failed document")

        # Now simulate retry functionality (what dead letter queue should do)
        # Reset the document to pending with retry_count = 0
        with self.job_control._get_connection() as conn:
            conn.execute("""
                UPDATE document_queue
                SET status = 'pending', retry_count = 0, error_message = NULL
                WHERE doc_id = ?
            """, (doc_id,))
            conn.commit()

        # Should be able to claim again
        claimed = self.job_control.claim_next_document(worker_id)
        self.assertIsNotNone(claimed, "Should be able to claim after retry")
        self.assertEqual(claimed['doc_id'], doc_id)

    def test_failure_pattern_analysis(self):
        """Test that we can analyze failure patterns."""
        # Create documents with different error patterns
        error_patterns = [
            ("ConnectionError", 3),
            ("ParseError", 2),
            ("ConnectionError", 2),
            ("TimeoutError", 1),
        ]

        worker_id = "test-worker"
        self.job_control.register_worker(
            worker_id, {"hostname": "localhost", "pid": 12345}
        )

        for i, (error_type, _) in enumerate(error_patterns):
            doc_id = f"pattern-doc-{i}"
            self.job_control.enqueue_document(
                doc_id, "test-source", {"index": i}
            )

            # Fail with specific error
            for _ in range(3):
                claimed = self.job_control.claim_next_document(worker_id)
                if claimed:
                    self.job_control.complete_document(
                        doc_id, worker_id, False, f"{error_type}: Test error message"
                    )

        # Analyze patterns
        with self.job_control._get_connection() as conn:
            cursor = conn.execute("""
                SELECT error_message, COUNT(*) as frequency
                FROM document_queue
                WHERE error_message IS NOT NULL
                GROUP BY error_message
                ORDER BY frequency DESC
            """)

            patterns = cursor.fetchall()

        # Should have 3 unique error patterns
        self.assertEqual(len(patterns), 3, "Should have 3 unique error patterns")

        # Most common should be ConnectionError (appears 2 times)
        top_pattern = patterns[0]
        self.assertIn("ConnectionError", top_pattern['error_message'])
        self.assertEqual(top_pattern['frequency'], 2)


if __name__ == '__main__':
    unittest.main()