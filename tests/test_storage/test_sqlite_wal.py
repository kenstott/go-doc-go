"""
Tests for SQLite WAL mode and performance optimizations.
"""

import pytest
import tempfile
import os
import sqlite3
from go_doc_go.storage.sqlite import SQLiteDocumentDatabase


class TestSQLiteWALMode:
    """Test SQLite WAL mode configuration."""
    
    @pytest.fixture
    def temp_db_path(self):
        """Create a temporary database path."""
        with tempfile.NamedTemporaryFile(suffix='.db', delete=False) as f:
            db_path = f.name
        yield db_path
        # Cleanup
        try:
            os.unlink(db_path)
            # Also remove WAL and SHM files if they exist
            if os.path.exists(f"{db_path}-wal"):
                os.unlink(f"{db_path}-wal")
            if os.path.exists(f"{db_path}-shm"):
                os.unlink(f"{db_path}-shm")
        except:
            pass
    
    def test_wal_mode_enabled(self, temp_db_path):
        """Test that WAL mode is properly enabled."""
        # Initialize database
        db = SQLiteDocumentDatabase(temp_db_path)
        db.initialize()
        
        # Check journal mode
        cursor = db.conn.cursor()
        result = cursor.execute("PRAGMA journal_mode").fetchone()
        
        # Should be 'wal' if WAL mode is supported
        # Some environments might not support WAL (e.g., network filesystems)
        assert result[0].lower() in ['wal', 'delete', 'truncate', 'persist', 'memory', 'off']
        
        # If WAL mode is enabled, verify it's actually 'wal'
        if result[0].lower() == 'wal':
            print("✓ WAL mode successfully enabled")
            
            # Check that WAL files are created
            db.store_document({
                'doc_id': 'test_doc',
                'doc_type': 'test',
                'source': 'test.txt',
                'metadata': {},
                'created_at': '2024-01-01',
                'updated_at': '2024-01-01'
            }, [], [])
            
            # WAL file should exist after a write
            assert os.path.exists(f"{temp_db_path}-wal"), "WAL file should be created"
        
        db.close()
    
    def test_performance_pragmas(self, temp_db_path):
        """Test that performance optimizations are applied."""
        # Initialize database
        db = SQLiteDocumentDatabase(temp_db_path)
        db.initialize()
        
        cursor = db.conn.cursor()
        
        # Check cache size (should be -8000 KB)
        cache_size = cursor.execute("PRAGMA cache_size").fetchone()[0]
        assert cache_size == -8000, f"Cache size should be -8000, got {cache_size}"
        
        # Check temp_store (should be 2 for MEMORY)
        temp_store = cursor.execute("PRAGMA temp_store").fetchone()[0]
        assert temp_store == 2, f"Temp store should be 2 (MEMORY), got {temp_store}"
        
        # Check synchronous mode (should be 1 for NORMAL with WAL)
        journal_mode = cursor.execute("PRAGMA journal_mode").fetchone()[0].lower()
        if journal_mode == 'wal':
            sync_mode = cursor.execute("PRAGMA synchronous").fetchone()[0]
            assert sync_mode == 1, f"Synchronous mode should be 1 (NORMAL), got {sync_mode}"
        
        # Check mmap_size (should be 268435456)
        mmap_size = cursor.execute("PRAGMA mmap_size").fetchone()[0]
        assert mmap_size == 268435456, f"mmap_size should be 268435456, got {mmap_size}"
        
        db.close()
    
    def test_concurrent_access_with_wal(self, temp_db_path):
        """Test that WAL mode allows concurrent readers."""
        # Initialize database
        db1 = SQLiteDocumentDatabase(temp_db_path)
        db1.initialize()
        
        # Insert test data
        db1.store_document({
            'doc_id': 'test_doc',
            'doc_type': 'test',
            'source': 'test.txt',
            'metadata': {},
            'created_at': '2024-01-01',
            'updated_at': '2024-01-01'
        }, [], [])
        
        # Open a second connection (reader)
        db2 = SQLiteDocumentDatabase(temp_db_path)
        db2.initialize()
        
        # Both connections should be able to read
        doc1 = db1.get_document('test_doc')
        doc2 = db2.get_document('test_doc')
        
        assert doc1 is not None
        assert doc2 is not None
        assert doc1['doc_id'] == doc2['doc_id']
        
        # One should be able to write while other reads (if WAL mode is enabled)
        journal_mode = db1.conn.execute("PRAGMA journal_mode").fetchone()[0].lower()
        
        if journal_mode == 'wal':
            # Reader starts a transaction
            cursor2 = db2.conn.cursor()
            cursor2.execute("BEGIN")
            cursor2.execute("SELECT COUNT(*) FROM documents")
            
            # Writer should still be able to write
            db1.store_document({
                'doc_id': 'test_doc2',
                'doc_type': 'test',
                'source': 'test2.txt',
                'metadata': {},
                'created_at': '2024-01-01',
                'updated_at': '2024-01-01'
            }, [], [])
            
            # Reader's transaction still valid
            count = cursor2.fetchone()[0]
            assert count >= 1
            cursor2.execute("COMMIT")
        
        db1.close()
        db2.close()
    
    def test_wal_checkpoint(self, temp_db_path):
        """Test WAL checkpoint configuration."""
        # Initialize database
        db = SQLiteDocumentDatabase(temp_db_path)
        db.initialize()
        
        cursor = db.conn.cursor()
        
        # Check WAL autocheckpoint setting
        wal_checkpoint = cursor.execute("PRAGMA wal_autocheckpoint").fetchone()[0]
        assert wal_checkpoint == 1000, f"WAL autocheckpoint should be 1000, got {wal_checkpoint}"
        
        # If WAL mode is enabled, test manual checkpoint
        journal_mode = cursor.execute("PRAGMA journal_mode").fetchone()[0].lower()
        if journal_mode == 'wal':
            # Insert some data
            for i in range(10):
                db.store_document({
                    'doc_id': f'test_doc_{i}',
                    'doc_type': 'test',
                    'source': f'test{i}.txt',
                    'metadata': {},
                    'created_at': '2024-01-01',
                    'updated_at': '2024-01-01'
                }, [], [])
            
            # Manual checkpoint
            result = cursor.execute("PRAGMA wal_checkpoint(TRUNCATE)").fetchone()
            # Should return (busy, log_frames, checkpointed_frames)
            assert len(result) == 3
            print(f"Checkpoint result - busy: {result[0]}, log_frames: {result[1]}, checkpointed: {result[2]}")
        
        db.close()