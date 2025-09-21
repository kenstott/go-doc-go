"""
SQLite adapter for queue system - provides PostgreSQL-compatible interface.

This adapter implements the database interface expected by the WorkQueue system,
providing SQLite as a backend while maintaining compatibility with PostgreSQL
SQL queries and transaction semantics.
"""

import contextlib
import json
import logging
import sqlite3
import threading
from datetime import datetime
from pathlib import Path
from typing import Dict, Any, Optional, Union

logger = logging.getLogger(__name__)


class SQLiteQueueAdapter:
    """
    SQLite database adapter for the queue system.
    
    Provides a PostgreSQL-compatible interface for the WorkQueue system,
    translating PostgreSQL SQL to SQLite equivalents and maintaining
    transaction semantics expected by the queue coordination logic.
    """
    
    def __init__(self, db_path: str):
        """
        Initialize SQLite queue adapter.

        Args:
            db_path: Path to SQLite database file
        """
        self.db_path = db_path
        self._local = threading.local()
        self._transaction_conn = None  # Track transaction connection

        # Ensure parent directory exists
        Path(self.db_path).parent.mkdir(parents=True, exist_ok=True)

        # Test connection to validate path
        conn = self._get_connection()
        conn.close()
        delattr(self._local, 'conn')

        logger.info(f"SQLiteQueueAdapter initialized: {self.db_path}")
    
    def _get_connection(self) -> sqlite3.Connection:
        """Get thread-local database connection."""
        if not hasattr(self._local, 'conn'):
            self._local.conn = sqlite3.connect(
                self.db_path,
                timeout=30.0,
                isolation_level=None  # Autocommit mode, we'll handle transactions manually
            )
            self._local.conn.row_factory = sqlite3.Row
            
            # Enable WAL mode for better concurrency
            self._local.conn.execute("PRAGMA journal_mode=WAL")
            
            # Enable foreign keys
            self._local.conn.execute("PRAGMA foreign_keys=ON")
            
        return self._local.conn
    
    def _convert_postgresql_to_sqlite(self, query: str) -> str:
        """
        Convert PostgreSQL SQL to SQLite equivalent.
        
        Args:
            query: PostgreSQL SQL query
            
        Returns:
            SQLite-compatible SQL query
        """
        sqlite_query = query
        
        # Parameter placeholders: %s -> ?
        sqlite_query = sqlite_query.replace('%s', '?')
        
        # Data types
        sqlite_query = sqlite_query.replace('SERIAL PRIMARY KEY', 'INTEGER PRIMARY KEY AUTOINCREMENT')
        sqlite_query = sqlite_query.replace('JSONB', 'TEXT')  # Store as JSON text
        sqlite_query = sqlite_query.replace('VARCHAR(', 'TEXT(')  # SQLite treats VARCHAR as TEXT
        sqlite_query = sqlite_query.replace('BIGINT', 'INTEGER')
        sqlite_query = sqlite_query.replace('NUMERIC', 'REAL')
        
        # Timestamp functions
        sqlite_query = sqlite_query.replace('CURRENT_TIMESTAMP', "datetime('now')")
        
        # Interval arithmetic - handle parameterized intervals
        import re
        
        # Handle INTERVAL '? seconds' format (with parameter placeholder)
        # CURRENT_TIMESTAMP - INTERVAL '? seconds' -> datetime('now', '-' || ? || ' seconds')
        sqlite_query = re.sub(
            r"datetime\('now'\) - INTERVAL '\? seconds'",
            r"datetime('now', '-' || ? || ' seconds')",
            sqlite_query
        )
        
        # CURRENT_TIMESTAMP + INTERVAL '? seconds' -> datetime('now', '+' || ? || ' seconds')
        sqlite_query = re.sub(
            r"datetime\('now'\) \+ INTERVAL '\? seconds'",
            r"datetime('now', '+' || ? || ' seconds')",
            sqlite_query
        )
        
        # Handle hardcoded numeric intervals (from earlier patterns)
        interval_pattern = r"datetime\('now'\) - \((\d+) \|\| ' seconds'\)::INTERVAL"
        sqlite_query = re.sub(interval_pattern, r"datetime('now', '-\1 seconds')", sqlite_query)
        
        interval_pattern = r"datetime\('now'\) \+ \((\d+) \|\| ' seconds'\)::INTERVAL"
        sqlite_query = re.sub(interval_pattern, r"datetime('now', '+\1 seconds')", sqlite_query)
        
        # LEAST/GREATEST functions - SQLite uses MIN/MAX
        # LEAST(a, b) -> MIN(a, b)
        sqlite_query = re.sub(r'LEAST\s*\(', 'MIN(', sqlite_query)
        sqlite_query = re.sub(r'GREATEST\s*\(', 'MAX(', sqlite_query)
        
        # Remove PostgreSQL row-level locking clauses
        # SQLite uses database-level locking, so these aren't needed
        sqlite_query = re.sub(r'\s+FOR\s+UPDATE\s+SKIP\s+LOCKED', '', sqlite_query, flags=re.IGNORECASE)
        sqlite_query = re.sub(r'\s+FOR\s+UPDATE', '', sqlite_query, flags=re.IGNORECASE)
        sqlite_query = re.sub(r'\s+FOR\s+SHARE', '', sqlite_query, flags=re.IGNORECASE)
        sqlite_query = re.sub(r'\s+NOWAIT', '', sqlite_query, flags=re.IGNORECASE)
        
        # ON CONFLICT clause
        sqlite_query = sqlite_query.replace('ON CONFLICT (', 'ON CONFLICT(')
        sqlite_query = sqlite_query.replace(') DO NOTHING', ') DO NOTHING')
        sqlite_query = sqlite_query.replace(') DO UPDATE', ') DO UPDATE')
        
        # REFERENCES constraint (SQLite supports but syntax is stricter)
        # Leave as-is, SQLite will handle
        
        return sqlite_query
    
    def execute(self, query: str, params: Optional[tuple] = None) -> Optional[Dict[str, Any]]:
        """
        Execute SQL query and return first result as dict.

        Args:
            query: SQL query (PostgreSQL format)
            params: Query parameters

        Returns:
            First result row as dictionary, or None if no results
        """
        # Use transaction connection if we're in a transaction
        conn = self._transaction_conn if self._transaction_conn else self._get_connection()
        cursor = conn.cursor()
        
        try:
            # Handle PostgreSQL function calls that SQLite doesn't support
            if 'SELECT attempt_leader_election' in query:
                # Implement leader election logic for SQLite
                return self._handle_leader_election(params)
            
            # Convert PostgreSQL SQL to SQLite
            sqlite_query = self._convert_postgresql_to_sqlite(query)
            
            # Handle INSERT...RETURNING for SQLite (which doesn't support it natively)
            if 'INSERT' in sqlite_query.upper() and 'RETURNING' in sqlite_query.upper():
                # Extract the RETURNING column name
                import re
                returning_match = re.search(r'RETURNING\s+(\w+)', sqlite_query, re.IGNORECASE)
                if returning_match:
                    return_column = returning_match.group(1)
                    # Remove RETURNING clause for SQLite
                    sqlite_query = re.sub(r'\s*RETURNING\s+\w+\s*$', '', sqlite_query, flags=re.IGNORECASE)

                    # Execute the INSERT
                    if params:
                        cursor.execute(sqlite_query, params)
                    else:
                        cursor.execute(sqlite_query)

                    # For INSERT...ON CONFLICT statements, we need to query back the actual ID
                    # because lastrowid only works for actual inserts, not updates from conflicts
                    if return_column.lower() in ['queue_id', 'id']:
                        if 'ON CONFLICT' in sqlite_query.upper():
                            # Query back the ID using the unique constraints from the original query
                            # For document_queue, we know the unique constraint is (run_id, doc_id, source_name)
                            if 'document_queue' in sqlite_query and len(params) >= 4:
                                doc_id, source_name, source_type, run_id = params[0], params[1], params[2], params[3]
                                cursor.execute(
                                    f"SELECT {return_column} FROM document_queue WHERE run_id = ? AND doc_id = ? AND source_name = ?",
                                    (run_id, doc_id, source_name)
                                )
                                row = cursor.fetchone()
                                result_id = row[0] if row else None
                            else:
                                # Fallback to lastrowid for other cases
                                result_id = cursor.lastrowid
                        else:
                            # Simple INSERT without conflict - use lastrowid
                            result_id = cursor.lastrowid

                        cursor.close()  # Close cursor
                        # Only commit if not in a transaction
                        if not self._transaction_conn:
                            conn.commit()
                        return {return_column: result_id}
                    else:
                        # For other columns, we'd need to query back
                        cursor.close()  # Close cursor
                        # Only commit if not in a transaction
                        if not self._transaction_conn:
                            conn.commit()
                        return None
            
            # Execute query
            if params:
                cursor.execute(sqlite_query, params)
            else:
                cursor.execute(sqlite_query)
            
            # For SELECT queries, return first result
            if sqlite_query.strip().upper().startswith('SELECT'):
                row = cursor.fetchone()
                if row:
                    # Convert Row to dict and handle JSON fields
                    result = dict(row)
                    # Convert JSON text fields back to objects where needed
                    for key, value in result.items():
                        if isinstance(value, str) and (
                            key.endswith('_snapshot') or 
                            key in ['metadata', 'config_snapshot', 'error_details', 'capabilities']
                        ):
                            try:
                                result[key] = json.loads(value)
                            except (json.JSONDecodeError, TypeError):
                                # Keep as string if not valid JSON
                                pass
                    cursor.close()  # Close cursor
                    return result
                cursor.close()  # Close cursor
                return None
            
            # For non-SELECT queries, close cursor and return None
            cursor.close()
            # Only commit if not in a transaction (transaction will handle commit)
            if not self._transaction_conn:
                conn.commit()
            return None
            
        except Exception as e:
            cursor.close()  # Ensure cursor is closed on error
            conn.rollback()
            logger.error(f"Error executing query: {e}")
            logger.debug(f"Query: {query}")
            logger.debug(f"Converted: {self._convert_postgresql_to_sqlite(query)}")
            raise
    
    def execute_raw(self, sql: str, params: Optional[tuple] = None) -> None:
        """
        Execute raw SQL without returning results.
        
        Args:
            sql: SQL to execute (PostgreSQL format)
            params: Query parameters
        """
        conn = self._get_connection()
        cursor = conn.cursor()
        
        try:
            # Convert PostgreSQL SQL to SQLite
            sqlite_sql = self._convert_postgresql_to_sqlite(sql)
            
            # Check if this is a trigger (which contains BEGIN...END block)
            if 'CREATE TRIGGER' in sqlite_sql.upper():
                # Execute trigger as single statement - don't split on semicolons
                if params:
                    cursor.execute(sqlite_sql, params)
                else:
                    cursor.execute(sqlite_sql)
            # For schema creation, we may need to execute multiple statements
            elif 'CREATE' in sqlite_sql.upper() and ';' in sqlite_sql:
                # Split on semicolons and execute each statement
                statements = [stmt.strip() for stmt in sqlite_sql.split(';') if stmt.strip()]
                for stmt in statements:
                    try:
                        if params:
                            cursor.execute(stmt, params)
                        else:
                            cursor.execute(stmt)
                    except sqlite3.OperationalError as e:
                        if "already exists" in str(e).lower():
                            # Ignore "already exists" errors for CREATE IF NOT EXISTS
                            continue
                        raise
            else:
                # Single statement
                if params:
                    cursor.execute(sqlite_sql, params)
                else:
                    cursor.execute(sqlite_sql)
            
            conn.commit()
            
        except Exception as e:
            conn.rollback()
            logger.error(f"Error executing raw SQL: {e}")
            logger.debug(f"SQL: {sql}")
            logger.debug(f"Converted: {self._convert_postgresql_to_sqlite(sql)}")
            raise
    
    def mark_completed(self, queue_id: int, content_hash: Optional[str] = None,
                      file_size: Optional[int] = None) -> None:
        """
        Mark a document as successfully processed.
        
        Args:
            queue_id: The queue ID of the document
            content_hash: Optional content hash of the processed document
            file_size: Optional file size of the processed document
        """
        with self.transaction() as conn:
            cursor = conn.cursor()
            
            # Update document_queue status to 'completed'
            # Note: SQLite doesn't support RETURNING clause in UPDATE
            # Also: SQLiteQueueAdapter doesn't track worker_id, so we update by queue_id only
            cursor.execute("""
                UPDATE document_queue
                SET status = 'completed',
                    completed_at = CURRENT_TIMESTAMP,
                    content_hash = COALESCE(?, content_hash),
                    file_size = COALESCE(?, file_size)
                WHERE queue_id = ?
            """, (content_hash, file_size, queue_id))
            
            # Get the run_id and worker_id for this queue item
            cursor.execute("""
                SELECT run_id, worker_id
                FROM document_queue
                WHERE queue_id = ?
            """, (queue_id,))
            
            result = cursor.fetchone()
            if result:
                run_id = result[0]
                worker_id = result[1] if len(result) > 1 else None
                
                # Update processing_runs statistics
                cursor.execute("""
                    UPDATE processing_runs
                    SET documents_processed = documents_processed + 1
                    WHERE run_id = ?
                """, (run_id,))
                
                # Update run_workers statistics if worker_id is known
                if worker_id:
                    cursor.execute("""
                        UPDATE run_workers
                        SET documents_processed = documents_processed + 1,
                            last_heartbeat = CURRENT_TIMESTAMP
                        WHERE run_id = ? AND worker_id = ?
                    """, (run_id, worker_id))
                
                logger.debug(f"Marked document {queue_id} as completed for run {run_id}")
            else:
                logger.warning(f"Could not find queue item {queue_id} to mark as completed")
    
    @contextlib.contextmanager
    def transaction(self):
        """
        Transaction context manager.

        Provides atomic transaction support compatible with queue operations.
        Uses IMMEDIATE transactions for better SQLite concurrency.
        """
        conn = self._get_connection()

        # Start immediate transaction
        conn.execute("BEGIN IMMEDIATE")

        # Set transaction connection so execute() uses it
        self._transaction_conn = conn

        try:
            yield self  # Yield self so execute() calls work within transaction
            conn.commit()
            logger.debug("SQLite transaction committed")
        except Exception as e:
            conn.rollback()
            logger.error(f"SQLite transaction rolled back: {e}")
            raise
        finally:
            # Clear transaction connection
            self._transaction_conn = None
    
    def _handle_leader_election(self, params: Optional[tuple]) -> Optional[Dict[str, Any]]:
        """
        Handle leader election for SQLite.
        
        Simulates PostgreSQL's attempt_leader_election function using SQLite.
        
        Args:
            params: Tuple of (run_id, worker_id, lease_duration)
            
        Returns:
            Dict with 'elected' key indicating if election was successful
        """
        if not params or len(params) < 3:
            logger.error("Invalid parameters for leader election")
            return {'elected': False}
        
        run_id, worker_id, lease_duration = params
        
        conn = self._get_connection()
        cursor = conn.cursor()
        
        try:
            # Begin immediate transaction for atomicity
            conn.execute("BEGIN IMMEDIATE")
            
            # Check if there's already a leader for this run
            cursor.execute("""
                SELECT leader_worker_id, leader_lease_expires
                FROM processing_runs
                WHERE run_id = ?
            """, (run_id,))
            
            result = cursor.fetchone()
            
            if not result:
                # No run exists - cannot elect leader
                conn.rollback()
                logger.warning(f"No processing run exists for run_id: {run_id}")
                return {'elected': False}
            
            current_leader, lease_expires = result
            
            # Check if we can become leader
            can_elect = False
            
            if not current_leader:
                # No current leader
                can_elect = True
            elif current_leader == worker_id:
                # We are already the leader - renew lease
                can_elect = True
            elif lease_expires:
                # Check if current lease has expired
                cursor.execute("""
                    SELECT datetime('now') > datetime(?)
                """, (lease_expires,))
                is_expired = cursor.fetchone()[0]
                can_elect = bool(is_expired)
            
            if can_elect:
                # Attempt to become leader
                new_lease_expires = f"datetime('now', '+{lease_duration} seconds')"
                cursor.execute(f"""
                    UPDATE processing_runs
                    SET leader_worker_id = ?,
                        leader_lease_expires = {new_lease_expires}
                    WHERE run_id = ?
                    AND (
                        leader_worker_id IS NULL
                        OR leader_worker_id = ?
                        OR leader_lease_expires < datetime('now')
                    )
                """, (worker_id, run_id, worker_id))
                
                # Check if update was successful
                if cursor.rowcount > 0:
                    conn.commit()
                    logger.info(f"Worker {worker_id} elected as leader for run {run_id}")
                    return {'elected': True}
                else:
                    conn.rollback()
                    logger.debug(f"Worker {worker_id} failed to become leader for run {run_id}")
                    return {'elected': False}
            else:
                conn.rollback()
                logger.debug(f"Worker {worker_id} cannot become leader - current leader: {current_leader}")
                return {'elected': False}
                
        except Exception as e:
            conn.rollback()
            logger.error(f"Error in leader election: {e}")
            return {'elected': False}
    
    def close(self) -> None:
        """Close database connections."""
        if hasattr(self._local, 'conn'):
            try:
                self._local.conn.close()
            except Exception:
                pass
            delattr(self._local, 'conn')


class SQLiteQueueSchemaAdapter:
    """
    Adapter to create SQLite-compatible queue schema.
    
    Handles the conversion of PostgreSQL-specific schema elements
    to SQLite equivalents, including functions and triggers.
    """
    
    @staticmethod
    def create_sqlite_schema(adapter: SQLiteQueueAdapter) -> None:
        """
        Create SQLite-compatible queue schema.
        
        Args:
            adapter: SQLite queue adapter instance
        """
        logger.info("Creating SQLite queue schema...")
        
        # Convert the PostgreSQL schema to SQLite
        sqlite_schema = SQLiteQueueSchemaAdapter._get_sqlite_schema()
        
        # Execute schema creation
        adapter.execute_raw(sqlite_schema)
        
        # Create SQLite-specific functions as triggers/views
        SQLiteQueueSchemaAdapter._create_sqlite_functions(adapter)
        
        logger.info("SQLite queue schema created successfully")
    
    @staticmethod
    def _get_sqlite_schema() -> str:
        """Get SQLite-compatible schema SQL."""
        return """
        -- Processing runs table - tracks coordinated processing batches
        CREATE TABLE IF NOT EXISTS processing_runs (
            run_id TEXT PRIMARY KEY,  -- Config hash (first 16 chars of SHA256)
            config_hash TEXT NOT NULL,  -- Full SHA256 hash for verification
            config_snapshot TEXT NOT NULL,  -- Complete config as JSON
            
            status TEXT NOT NULL DEFAULT 'active',
            -- Status values: 'active', 'processing_complete', 'post_processing', 'completed', 'failed', 'abandoned'
            
            -- Timing
            created_at TEXT DEFAULT (datetime('now')),
            first_worker_at TEXT,
            last_activity_at TEXT DEFAULT (datetime('now')),
            processing_completed_at TEXT,
            post_processing_started_at TEXT,
            post_processing_completed_at TEXT,
            completed_at TEXT,
            
            -- Statistics
            worker_count INTEGER DEFAULT 0,
            documents_queued INTEGER DEFAULT 0,
            documents_processed INTEGER DEFAULT 0,
            documents_failed INTEGER DEFAULT 0,
            documents_retried INTEGER DEFAULT 0,
            
            -- Leader election
            leader_worker_id TEXT,
            leader_elected_at TEXT,
            leader_heartbeat TEXT,
            leader_lease_expires TEXT,
            
            -- Post-processing coordination (handled by leader)
            post_processor_worker_id TEXT,
            post_processing_lock_acquired_at TEXT,
            
            -- Metadata
            metadata TEXT,  -- JSON
            
            -- Ensure config hash uniqueness
            UNIQUE(config_hash)
        );

        -- Document queue table - the main work queue
        CREATE TABLE IF NOT EXISTS document_queue (
            queue_id INTEGER PRIMARY KEY AUTOINCREMENT,
            
            -- Document identification
            doc_id TEXT NOT NULL,
            source_name TEXT NOT NULL,
            source_type TEXT NOT NULL DEFAULT 'configured',
            -- source_type values: 'configured', 'linked', 'discovered'
            
            -- Run coordination
            run_id TEXT NOT NULL REFERENCES processing_runs(run_id),
            
            -- Processing status
            status TEXT NOT NULL DEFAULT 'pending',
            -- Status values: 'pending', 'processing', 'completed', 'failed', 'retry'
            
            -- Worker assignment
            worker_id TEXT,
            claimed_at TEXT,
            started_at TEXT,
            completed_at TEXT,
            failed_at TEXT,
            
            -- Retry handling
            retry_count INTEGER DEFAULT 0,
            max_retries INTEGER DEFAULT 3,
            error_message TEXT,
            error_details TEXT,  -- JSON
            
            -- Link traversal
            parent_doc_id TEXT,  -- Document that linked to this one
            link_depth INTEGER DEFAULT 0,  -- How deep in the link chain
            max_link_depth INTEGER DEFAULT 3,  -- Max depth to traverse
            
            -- Change detection
            content_hash TEXT,
            last_modified TEXT,
            file_size INTEGER,
            
            -- Priority and scheduling
            priority INTEGER DEFAULT 0,  -- Higher number = higher priority
            scheduled_for TEXT DEFAULT (datetime('now')),
            
            -- Metadata
            metadata TEXT,  -- JSON
            created_at TEXT DEFAULT (datetime('now')),
            updated_at TEXT DEFAULT (datetime('now')),
            
            -- Ensure document uniqueness per run
            UNIQUE(run_id, doc_id, source_name),
            
            FOREIGN KEY (run_id) REFERENCES processing_runs(run_id)
        );

        -- Create indexes for document_queue performance
        CREATE INDEX IF NOT EXISTS idx_queue_status_run ON document_queue (run_id, status, scheduled_for);
        CREATE INDEX IF NOT EXISTS idx_queue_worker ON document_queue (worker_id, status);
        CREATE INDEX IF NOT EXISTS idx_queue_parent ON document_queue (parent_doc_id);
        CREATE INDEX IF NOT EXISTS idx_queue_priority ON document_queue (priority DESC, scheduled_for ASC);

        -- Run workers table - tracks which workers are processing which run
        CREATE TABLE IF NOT EXISTS run_workers (
            run_id TEXT NOT NULL REFERENCES processing_runs(run_id),
            worker_id TEXT NOT NULL,
            
            -- Worker lifecycle
            joined_at TEXT DEFAULT (datetime('now')),
            last_heartbeat TEXT DEFAULT (datetime('now')),
            left_at TEXT,
            
            -- Worker status
            status TEXT DEFAULT 'active',
            -- Status values: 'active', 'idle', 'processing', 'stopped', 'failed'
            
            -- Statistics
            documents_claimed INTEGER DEFAULT 0,
            documents_processed INTEGER DEFAULT 0,
            documents_failed INTEGER DEFAULT 0,
            processing_time_seconds REAL DEFAULT 0,
            
            -- Worker metadata
            hostname TEXT,
            process_id INTEGER,
            version TEXT,
            capabilities TEXT,  -- JSON
            
            PRIMARY KEY (run_id, worker_id)
        );

        -- Create index for run_workers performance
        CREATE INDEX IF NOT EXISTS idx_worker_heartbeat ON run_workers (last_heartbeat DESC);

        -- Document dependencies table - tracks linked document relationships
        CREATE TABLE IF NOT EXISTS document_dependencies (
            parent_doc_id TEXT NOT NULL,
            child_doc_id TEXT NOT NULL,
            source_name TEXT NOT NULL,
            run_id TEXT NOT NULL REFERENCES processing_runs(run_id),
            
            -- Link metadata
            link_type TEXT,  -- 'explicit', 'discovered', 'inferred'
            link_depth INTEGER NOT NULL,
            discovered_at TEXT DEFAULT (datetime('now')),
            discovered_by_worker TEXT,
            
            PRIMARY KEY (run_id, parent_doc_id, child_doc_id, source_name)
        );

        -- Create indexes for document_dependencies performance  
        CREATE INDEX IF NOT EXISTS idx_deps_child ON document_dependencies (child_doc_id);
        CREATE INDEX IF NOT EXISTS idx_deps_run ON document_dependencies (run_id);
        """
    
    @staticmethod
    def _create_sqlite_functions(adapter: SQLiteQueueAdapter) -> None:
        """
        Create SQLite equivalents of PostgreSQL functions.
        
        Args:
            adapter: SQLite queue adapter instance
        """
        # Create trigger for updated_at column
        adapter.execute_raw("""
            CREATE TRIGGER IF NOT EXISTS update_document_queue_updated_at 
            AFTER UPDATE ON document_queue 
            FOR EACH ROW 
            BEGIN
                UPDATE document_queue 
                SET updated_at = datetime('now') 
                WHERE queue_id = NEW.queue_id;
            END;
        """)
        
        logger.debug("SQLite functions and triggers created")


def create_sqlite_queue_adapter(config: Dict[str, Any]) -> SQLiteQueueAdapter:
    """
    Factory function to create SQLite queue adapter.
    
    Args:
        config: Configuration dictionary with 'path' key
        
    Returns:
        Configured SQLiteQueueAdapter instance
    """
    db_path = config.get('path', 'queue.db')
    adapter = SQLiteQueueAdapter(db_path)
    
    # Ensure schema is current - drop and recreate if needed
    _ensure_schema_current(adapter)
    
    return adapter


def _ensure_schema_current(adapter: SQLiteQueueAdapter) -> None:
    """
    Ensure the database schema is current. Drop and recreate if schema is outdated.
    
    Args:
        adapter: SQLiteQueueAdapter instance
    """
    try:
        # Check if processing_runs table has config_hash column
        result = adapter.execute("""
            SELECT sql FROM sqlite_master 
            WHERE type='table' AND name='processing_runs'
        """)
        
        if result:
            table_sql = result.get('sql', '')
            # If table exists but missing required columns, drop all and recreate
            if 'config_hash' not in table_sql or 'leader_worker_id' not in table_sql:
                logger.info("Schema outdated, dropping and recreating all queue tables...")
                _drop_all_queue_tables(adapter)
                SQLiteQueueSchemaAdapter.create_sqlite_schema(adapter)
                logger.info("Queue schema recreated successfully")
            else:
                # Schema is current
                logger.debug("Queue schema is current")
        else:
            # No table exists, create schema
            logger.info("Creating SQLite queue schema...")
            SQLiteQueueSchemaAdapter.create_sqlite_schema(adapter)
            
    except Exception as e:
        logger.error(f"Error checking/creating schema: {e}")
        # Try to drop and recreate as last resort
        try:
            logger.info("Attempting to drop and recreate queue schema...")
            _drop_all_queue_tables(adapter)
            SQLiteQueueSchemaAdapter.create_sqlite_schema(adapter)
            logger.info("Queue schema recreated successfully")
        except Exception as recreate_error:
            logger.error(f"Failed to recreate schema: {recreate_error}")
            raise


def _drop_all_queue_tables(adapter: SQLiteQueueAdapter) -> None:
    """
    Drop all queue-related tables.
    
    Args:
        adapter: SQLiteQueueAdapter instance
    """
    tables = [
        'document_dependencies',
        'run_workers',
        'document_queue',
        'processing_runs'
    ]
    
    for table in tables:
        try:
            adapter.execute_raw(f"DROP TABLE IF EXISTS {table}")
            logger.debug(f"Dropped table: {table}")
        except Exception as e:
            logger.debug(f"Error dropping table {table}: {e}")