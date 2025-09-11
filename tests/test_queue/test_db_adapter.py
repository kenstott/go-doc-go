"""
Database adapter for queue tests to bridge between queue interface
and existing PostgreSQL database class.
"""

import contextlib
import threading
from typing import Dict, Any, Optional


class QueueDatabaseAdapter:
    """Adapter to make PostgreSQL database compatible with queue interface."""
    
    def __init__(self, pg_db):
        """Initialize with PostgreSQL database instance."""
        self.pg_db = pg_db
        self._local = threading.local()
    
    def _get_connection(self):
        """Get thread-local connection."""
        if not hasattr(self._local, 'conn'):
            # Create a new connection for this thread
            import psycopg2
            from psycopg2.extras import RealDictCursor
            
            # Use the same connection parameters as main connection
            conn_params = dict(self.pg_db.conn_params)
            conn_params['cursor_factory'] = RealDictCursor
            
            # Handle dbname vs database parameter
            if 'database' in conn_params and 'dbname' not in conn_params:
                conn_params['dbname'] = conn_params['database']
                del conn_params['database']
            
            self._local.conn = psycopg2.connect(**conn_params)
            self._local.conn.autocommit = False
            
        return self._local.conn
    
    def _get_cursor(self):
        """Get thread-local cursor."""
        if not hasattr(self._local, 'cursor'):
            conn = self._get_connection()
            self._local.cursor = conn.cursor()
        return self._local.cursor
    
    @contextlib.contextmanager
    def transaction(self):
        """Transaction context manager."""
        conn = self._get_connection()
        try:
            yield
            conn.commit()
        except Exception:
            conn.rollback()
            raise
    
    def execute(self, query: str, params: Optional[tuple] = None) -> Optional[Dict[str, Any]]:
        """Execute SQL query and return first result as dict."""
        cursor = self._get_cursor()
        cursor.execute(query, params or ())
        
        if cursor.description:
            # Query returns results
            try:
                result = cursor.fetchone()
                if result:
                    # RealDictCursor already returns dict-like objects
                    return dict(result)
            except Exception:
                # No results available (e.g., UPDATE with no rows affected)
                pass
        return None
    
    def execute_raw(self, sql: str, params: Optional[tuple] = None) -> None:
        """Execute raw SQL without returning results."""
        cursor = self._get_cursor()
        cursor.execute(sql, params or ())
    
    def close(self):
        """Close database connection."""
        # Close thread-local connections
        if hasattr(self._local, 'cursor'):
            try:
                self._local.cursor.close()
            except:
                pass
        
        if hasattr(self._local, 'conn'):
            try:
                self._local.conn.close()
            except:
                pass
        
        # Close main connection
        self.pg_db.close()