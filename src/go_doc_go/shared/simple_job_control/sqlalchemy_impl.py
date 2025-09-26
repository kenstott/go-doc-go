"""
SQLAlchemy implementation of simplified job control database.
Supports PostgreSQL, MySQL, SQL Server, Oracle, and other SQL databases.
"""

import json
import logging
import os
from typing import Dict, Any, Optional, List

from .base import SimpleJobControlDB

logger = logging.getLogger(__name__)

try:
    from sqlalchemy import (
        create_engine, MetaData, Table, Column, String, Integer, DateTime,
        Text, Float, text, func, select, update, delete, insert, and_
    )
    from sqlalchemy.orm import sessionmaker
    from sqlalchemy.dialects.postgresql import JSONB, insert as pg_insert
    from sqlalchemy.dialects.mysql import JSON as MySQL_JSON, insert as mysql_insert
    SQLALCHEMY_AVAILABLE = True
except ImportError:
    SQLALCHEMY_AVAILABLE = False
    logger.warning("SQLAlchemy not available. Install with: pip install sqlalchemy[postgresql]")


class SQLAlchemyJobControlDB(SimpleJobControlDB):
    """SQLAlchemy implementation supporting PostgreSQL, MySQL, SQL Server, Oracle, etc."""

    def __init__(self, config):
        if not SQLALCHEMY_AVAILABLE:
            raise ImportError("SQLAlchemy required for database job control")

        self.config = config
        job_control_config = config.get_job_control_config()

        # Get connection URL - can be explicit or built from components
        self.database_url = job_control_config.get('url')
        if not self.database_url:
            # Build URL from components
            backend = job_control_config.get('backend', 'postgresql')
            connection_config = job_control_config.get('connection', {})

            host = connection_config.get('host', 'localhost')
            port = connection_config.get('port')
            database = connection_config.get('database', 'godocgo')
            username = connection_config.get('username', connection_config.get('user', 'postgres'))
            password = connection_config.get('password', '')

            # Set default ports if not specified
            if port is None:
                port_defaults = {
                    'postgresql': 5432,
                    'mysql': 3306,
                    'mssql': 1433,
                    'oracle': 1521
                }
                port = port_defaults.get(backend, 5432)

            # Build connection URL
            if backend == 'postgresql':
                self.database_url = f'postgresql://{username}:{password}@{host}:{port}/{database}'
            elif backend == 'mysql':
                self.database_url = f'mysql+pymysql://{username}:{password}@{host}:{port}/{database}'
            elif backend == 'mssql':
                self.database_url = f'mssql+pyodbc://{username}:{password}@{host}:{port}/{database}?driver=ODBC+Driver+17+for+SQL+Server'
            elif backend == 'oracle':
                self.database_url = f'oracle+cx_oracle://{username}:{password}@{host}:{port}/{database}'
            else:
                raise ValueError(f"Unsupported database backend: {backend}")

        self.claim_timeout = job_control_config.get('claim_timeout', 300)
        self.heartbeat_interval = job_control_config.get('heartbeat_interval', 30)
        self.max_retries = job_control_config.get('max_retries', 3)

        # Create engine and session
        self.engine = create_engine(self.database_url, pool_pre_ping=True)
        self.Session = sessionmaker(bind=self.engine)
        self.metadata = MetaData()

        # Choose JSON type based on database
        if 'postgresql' in self.database_url:
            json_type = JSONB
        elif 'mysql' in self.database_url:
            json_type = MySQL_JSON
        else:
            json_type = Text  # Fallback to Text for other databases

        # Define tables
        self.document_queue = Table(
            'document_queue', self.metadata,
            Column('doc_id', String(255), primary_key=True),
            Column('source', String(255), nullable=False),
            Column('metadata', json_type),
            Column('status', String(50), default='pending'),
            Column('claimed_by', String(255)),
            Column('claimed_at', DateTime),
            Column('completed_at', DateTime),
            Column('retry_count', Integer, default=0),
            Column('error_message', Text),
            Column('created_at', DateTime, default=func.now())
        )

        self.workers = Table(
            'workers', self.metadata,
            Column('worker_id', String(255), primary_key=True),
            Column('hostname', String(255)),
            Column('pid', Integer),
            Column('started_at', DateTime, default=func.now()),
            Column('last_heartbeat', DateTime, default=func.now()),
            Column('status', String(50), default='active'),
            Column('processed_count', Integer, default=0),
            Column('worker_info', json_type)
        )

        self.leaders = Table(
            'leaders', self.metadata,
            Column('id', Integer, primary_key=True, default=1),
            Column('worker_id', String(255), nullable=False),
            Column('elected_at', DateTime, default=func.now()),
            Column('last_heartbeat', DateTime, default=func.now()),
            Column('worker_info', json_type)
        )

        self.document_metadata = Table(
            'document_metadata', self.metadata,
            Column('doc_id', String(255), primary_key=True),
            Column('source', String(255), nullable=False),
            Column('last_modified', Float),  # Unix timestamp
            Column('content_hash', String(32)),  # MD5 hash
            Column('file_size', Integer),
            Column('last_processed_at', DateTime),
            Column('processing_stats', json_type)
        )

    def initialize_schema(self):
        """Create database schema if it doesn't exist."""
        self.metadata.create_all(self.engine)

        # Create indexes
        try:
            with self.engine.connect() as conn:
                # Create indexes if they don't exist (database-agnostic)
                indexes = [
                    "CREATE INDEX IF NOT EXISTS idx_document_status ON document_queue(status)",
                    "CREATE INDEX IF NOT EXISTS idx_document_claimed_at ON document_queue(claimed_at)",
                    "CREATE INDEX IF NOT EXISTS idx_worker_heartbeat ON workers(last_heartbeat)",
                    "CREATE INDEX IF NOT EXISTS idx_document_metadata_source ON document_metadata(source)",
                    "CREATE INDEX IF NOT EXISTS idx_document_metadata_last_modified ON document_metadata(last_modified)"
                ]

                for index_sql in indexes:
                    try:
                        conn.execute(text(index_sql))
                    except Exception:
                        # Index might already exist or syntax might be different
                        pass
                conn.commit()
        except Exception as e:
            logger.debug(f"Index creation warning (may be harmless): {e}")

    def enqueue_document(self, doc_id: str, source: str, metadata: Dict[str, Any]):
        """Add a document to the processing queue."""
        with self.Session() as session:
            # Use upsert logic appropriate for the database
            if 'postgresql' in self.database_url:
                stmt = pg_insert(self.document_queue).values(
                    doc_id=doc_id,
                    source=source,
                    metadata=metadata,
                    status='pending',
                    retry_count=0
                )
                stmt = stmt.on_conflict_do_update(
                    index_elements=['doc_id'],
                    set_=dict(
                        source=stmt.excluded.source,
                        metadata=stmt.excluded.metadata,
                        status='pending',
                        retry_count=0
                    )
                )
            elif 'mysql' in self.database_url:
                stmt = mysql_insert(self.document_queue).values(
                    doc_id=doc_id,
                    source=source,
                    metadata=metadata,
                    status='pending',
                    retry_count=0
                )
                stmt = stmt.on_duplicate_key_update(
                    source=stmt.inserted.source,
                    metadata=stmt.inserted.metadata,
                    status='pending',
                    retry_count=0
                )
            else:
                # Fallback: delete then insert
                delete_stmt = delete(self.document_queue).where(
                    self.document_queue.c.doc_id == doc_id
                )
                session.execute(delete_stmt)

                stmt = insert(self.document_queue).values(
                    doc_id=doc_id,
                    source=source,
                    metadata=metadata,
                    status='pending',
                    retry_count=0
                )

            session.execute(stmt)
            session.commit()

    def claim_next_document(self, worker_id: str) -> Optional[Dict[str, Any]]:
        """Atomically claim next available document for processing."""
        with self.Session() as session:
            # Use database-specific locking for atomic operations
            if 'postgresql' in self.database_url:
                # PostgreSQL SKIP LOCKED for best performance
                stmt = select(
                    self.document_queue.c.doc_id,
                    self.document_queue.c.source,
                    self.document_queue.c.metadata
                ).where(
                    and_(
                        self.document_queue.c.status == 'pending',
                        self.document_queue.c.retry_count < self.max_retries
                    )
                ).order_by(self.document_queue.c.created_at).limit(1)

                result = session.execute(stmt.with_for_update(skip_locked=True)).fetchone()
                if result:
                    doc_id = result[0]
                    # Try to claim it
                    update_stmt = update(self.document_queue).where(
                        and_(
                            self.document_queue.c.doc_id == doc_id,
                            self.document_queue.c.status == 'pending'
                        )
                    ).values(
                        status='processing',
                        claimed_by=worker_id,
                        claimed_at=func.now()
                    )
                    update_result = session.execute(update_stmt)
                    if update_result.rowcount > 0:
                        session.commit()
                        return {
                            'doc_id': result[0],
                            'source': result[1],
                            'metadata': result[2] or {}
                        }
            else:
                # Generic approach for other databases
                stmt = select(
                    self.document_queue.c.doc_id,
                    self.document_queue.c.source,
                    self.document_queue.c.metadata
                ).where(
                    and_(
                        self.document_queue.c.status == 'pending',
                        self.document_queue.c.retry_count < self.max_retries
                    )
                ).order_by(self.document_queue.c.created_at).limit(1)
                result = session.execute(stmt).fetchone()
                if result:
                    doc_id = result[0]
                    # Try to claim it
                    update_stmt = update(self.document_queue).where(
                        and_(
                            self.document_queue.c.doc_id == doc_id,
                            self.document_queue.c.status == 'pending'
                        )
                    ).values(
                        status='processing',
                        claimed_by=worker_id,
                        claimed_at=func.now()
                    )
                    update_result = session.execute(update_stmt)
                    if update_result.rowcount > 0:
                        session.commit()
                        return {
                            'doc_id': result[0],
                            'source': result[1],
                            'metadata': result[2] or {}
                        }

            session.rollback()
            return None

    def complete_document(self, doc_id: str, worker_id: str, success: bool, error_message: Optional[str] = None):
        """Mark a document as completed (successfully or failed)."""
        with self.Session() as session:
            if success:
                stmt = update(self.document_queue).where(
                    and_(
                        self.document_queue.c.doc_id == doc_id,
                        self.document_queue.c.claimed_by == worker_id
                    )
                ).values(
                    status='completed',
                    completed_at=func.now(),
                    error_message=None
                )
            else:
                new_status = 'pending' if error_message else 'failed'
                stmt = update(self.document_queue).where(
                    and_(
                        self.document_queue.c.doc_id == doc_id,
                        self.document_queue.c.claimed_by == worker_id
                    )
                ).values(
                    status=new_status,
                    retry_count=self.document_queue.c.retry_count + 1,
                    error_message=error_message
                )

            session.execute(stmt)

            # Update worker processed count if successful
            if success:
                worker_stmt = update(self.workers).where(
                    self.workers.c.worker_id == worker_id
                ).values(
                    processed_count=self.workers.c.processed_count + 1
                )
                session.execute(worker_stmt)

            session.commit()

    def release_document(self, doc_id: str, worker_id: str):
        """Release a claimed document back to the queue."""
        with self.Session() as session:
            stmt = update(self.document_queue).where(
                and_(
                    self.document_queue.c.doc_id == doc_id,
                    self.document_queue.c.claimed_by == worker_id
                )
            ).values(
                status='pending',
                claimed_by=None,
                claimed_at=None
            )
            session.execute(stmt)
            session.commit()

    def register_worker(self, worker_id: str, worker_info: Dict[str, Any]):
        """Register a worker in the system."""
        with self.Session() as session:
            # Use upsert for worker registration
            if 'postgresql' in self.database_url:
                stmt = pg_insert(self.workers).values(
                    worker_id=worker_id,
                    hostname=worker_info.get('hostname', ''),
                    pid=worker_info.get('pid', 0),
                    worker_info=worker_info,
                    last_heartbeat=func.now()
                )
                stmt = stmt.on_conflict_do_update(
                    index_elements=['worker_id'],
                    set_=dict(
                        hostname=stmt.excluded.hostname,
                        pid=stmt.excluded.pid,
                        worker_info=stmt.excluded.worker_info,
                        last_heartbeat=func.now()
                    )
                )
            else:
                # Delete then insert for other databases
                delete_stmt = delete(self.workers).where(
                    self.workers.c.worker_id == worker_id
                )
                session.execute(delete_stmt)

                stmt = insert(self.workers).values(
                    worker_id=worker_id,
                    hostname=worker_info.get('hostname', ''),
                    pid=worker_info.get('pid', 0),
                    worker_info=worker_info,
                    last_heartbeat=func.now()
                )

            session.execute(stmt)
            session.commit()

    def update_worker_heartbeat(self, worker_id: str):
        """Update worker heartbeat timestamp."""
        with self.Session() as session:
            stmt = update(self.workers).where(
                self.workers.c.worker_id == worker_id
            ).values(last_heartbeat=func.now())
            session.execute(stmt)
            session.commit()

    def get_processing_status(self) -> Dict[str, Any]:
        """Get current processing status."""
        with self.Session() as session:
            # Get document counts by status
            doc_counts_stmt = select(
                self.document_queue.c.status,
                func.count().label('count')
            ).group_by(self.document_queue.c.status)
            doc_counts = {row[0]: row[1] for row in session.execute(doc_counts_stmt)}

            # Get active worker count (database-agnostic interval)
            if 'postgresql' in self.database_url:
                active_workers_stmt = text("""
                    SELECT COUNT(*) as count
                    FROM workers
                    WHERE last_heartbeat > CURRENT_TIMESTAMP - INTERVAL ':interval seconds'
                """).bindparam(interval=self.heartbeat_interval * 2)
            elif 'mysql' in self.database_url:
                active_workers_stmt = text("""
                    SELECT COUNT(*) as count
                    FROM workers
                    WHERE last_heartbeat > DATE_SUB(NOW(), INTERVAL :interval SECOND)
                """).bindparam(interval=self.heartbeat_interval * 2)
            else:
                # Generic SQL approach
                active_workers_stmt = text("""
                    SELECT COUNT(*) as count
                    FROM workers
                    WHERE last_heartbeat > DATEADD(second, -:interval, GETDATE())
                """).bindparam(interval=self.heartbeat_interval * 2)

            try:
                active_workers = session.execute(active_workers_stmt).scalar() or 0
            except Exception:
                # Fallback: count all workers
                active_workers = session.execute(select(func.count()).select_from(self.workers)).scalar() or 0

            return {
                'documents': doc_counts,
                'active_workers': active_workers,
                'queue_status': 'idle' if doc_counts.get('pending', 0) == 0 and doc_counts.get('processing', 0) == 0 else 'active'
            }

    def cleanup_stale_claims(self, timeout_seconds: int = 300):
        """Release documents claimed by inactive workers."""
        with self.Session() as session:
            if 'postgresql' in self.database_url:
                stmt = update(self.document_queue).where(
                    and_(
                        self.document_queue.c.status == 'processing',
                        self.document_queue.c.claimed_at < func.now() - text(f"INTERVAL '{timeout_seconds} seconds'")
                    )
                ).values(
                    status='pending',
                    claimed_by=None,
                    claimed_at=None
                )
            else:
                # Generic approach using datetime functions
                stmt = update(self.document_queue).where(
                    and_(
                        self.document_queue.c.status == 'processing',
                        self.document_queue.c.claimed_at < func.datetime('now', f'-{timeout_seconds} seconds')
                    )
                ).values(
                    status='pending',
                    claimed_by=None,
                    claimed_at=None
                )
            result = session.execute(stmt)
            count = result.rowcount
            session.commit()

            if count > 0:
                logger.info(f"Released {count} stale document claims")

    def is_document_queued(self, doc_id: str) -> bool:
        """Check if document is in the queue."""
        with self.Session() as session:
            stmt = select(self.document_queue.c.doc_id).where(
                self.document_queue.c.doc_id == doc_id
            )
            result = session.execute(stmt).fetchone()
            return result is not None

    def elect_leader(self, worker_id: str, worker_info: Dict[str, Any]) -> bool:
        """Attempt to elect this worker as leader."""
        with self.Session() as session:
            try:
                # Check if leader exists and is active
                if 'postgresql' in self.database_url:
                    leader_check = text("""
                        SELECT worker_id FROM leaders
                        WHERE id = 1 AND last_heartbeat > CURRENT_TIMESTAMP - INTERVAL ':interval seconds'
                    """).bindparam(interval=self.heartbeat_interval * 3)
                else:
                    leader_check = text("""
                        SELECT worker_id FROM leaders
                        WHERE id = 1 AND last_heartbeat > DATEADD(second, -:interval, GETDATE())
                    """).bindparam(interval=self.heartbeat_interval * 3)

                active_leader = session.execute(leader_check).fetchone()
                if active_leader:
                    return False

                # Clear any stale leader
                session.execute(delete(self.leaders).where(self.leaders.c.id == 1))

                # Insert as new leader
                stmt = insert(self.leaders).values(
                    id=1,
                    worker_id=worker_id,
                    worker_info=worker_info
                )
                session.execute(stmt)
                session.commit()
                return True

            except Exception as e:
                session.rollback()
                logger.debug(f"Leader election failed for {worker_id}: {e}")
                return False

    def is_leader(self, worker_id: str) -> bool:
        """Check if worker is current leader."""
        with self.Session() as session:
            stmt = select(self.leaders.c.worker_id).where(
                and_(
                    self.leaders.c.id == 1,
                    self.leaders.c.worker_id == worker_id
                )
            )
            result = session.execute(stmt).fetchone()
            return result is not None

    def update_leader_heartbeat(self, worker_id: str):
        """Update leader heartbeat."""
        with self.Session() as session:
            stmt = update(self.leaders).where(
                and_(
                    self.leaders.c.id == 1,
                    self.leaders.c.worker_id == worker_id
                )
            ).values(last_heartbeat=func.now())
            session.execute(stmt)
            session.commit()

    def get_current_leader(self) -> Optional[Dict[str, Any]]:
        """Get current leader information."""
        with self.Session() as session:
            stmt = select(
                self.leaders.c.worker_id,
                self.leaders.c.elected_at,
                self.leaders.c.last_heartbeat,
                self.leaders.c.worker_info
            ).where(self.leaders.c.id == 1)

            result = session.execute(stmt).fetchone()
            if result:
                return {
                    'worker_id': result[0],
                    'elected_at': result[1],
                    'last_heartbeat': result[2],
                    'worker_info': result[3]
                }
            return None

    def release_leadership(self, worker_id: str):
        """Release leadership."""
        with self.Session() as session:
            stmt = delete(self.leaders).where(
                and_(
                    self.leaders.c.id == 1,
                    self.leaders.c.worker_id == worker_id
                )
            )
            session.execute(stmt)
            session.commit()

    def store_document_metadata(self, doc_id: str, source: str,
                               last_modified: Optional[float] = None,
                               content_hash: Optional[str] = None,
                               file_size: Optional[int] = None,
                               processing_stats: Optional[Dict[str, Any]] = None):
        """Store document metadata for change tracking."""
        with self.Session() as session:
            # Use upsert logic appropriate for the database
            if 'postgresql' in self.database_url:
                stmt = pg_insert(self.document_metadata).values(
                    doc_id=doc_id,
                    source=source,
                    last_modified=last_modified,
                    content_hash=content_hash,
                    file_size=file_size,
                    last_processed_at=func.now(),
                    processing_stats=processing_stats
                )
                stmt = stmt.on_conflict_do_update(
                    index_elements=['doc_id'],
                    set_=dict(
                        source=stmt.excluded.source,
                        last_modified=stmt.excluded.last_modified,
                        content_hash=stmt.excluded.content_hash,
                        file_size=stmt.excluded.file_size,
                        last_processed_at=stmt.excluded.last_processed_at,
                        processing_stats=stmt.excluded.processing_stats
                    )
                )
            elif 'mysql' in self.database_url:
                stmt = mysql_insert(self.document_metadata).values(
                    doc_id=doc_id,
                    source=source,
                    last_modified=last_modified,
                    content_hash=content_hash,
                    file_size=file_size,
                    last_processed_at=func.now(),
                    processing_stats=processing_stats
                )
                stmt = stmt.on_duplicate_key_update(
                    source=stmt.inserted.source,
                    last_modified=stmt.inserted.last_modified,
                    content_hash=stmt.inserted.content_hash,
                    file_size=stmt.inserted.file_size,
                    last_processed_at=stmt.inserted.last_processed_at,
                    processing_stats=stmt.inserted.processing_stats
                )
            else:
                # Fallback: delete then insert
                delete_stmt = delete(self.document_metadata).where(
                    self.document_metadata.c.doc_id == doc_id
                )
                session.execute(delete_stmt)

                stmt = insert(self.document_metadata).values(
                    doc_id=doc_id,
                    source=source,
                    last_modified=last_modified,
                    content_hash=content_hash,
                    file_size=file_size,
                    last_processed_at=func.now(),
                    processing_stats=processing_stats
                )

            session.execute(stmt)
            session.commit()

    def get_document_metadata(self, doc_id: str) -> Optional[Dict[str, Any]]:
        """Get stored document metadata."""
        with self.Session() as session:
            stmt = select(self.document_metadata).where(
                self.document_metadata.c.doc_id == doc_id
            )
            result = session.execute(stmt).fetchone()

            if result:
                return {
                    'doc_id': result.doc_id,
                    'source': result.source,
                    'last_modified': result.last_modified,
                    'content_hash': result.content_hash,
                    'file_size': result.file_size,
                    'last_processed_at': result.last_processed_at,
                    'processing_stats': result.processing_stats
                }
            return None

    def has_document_changed(self, doc_id: str, source: str,
                            current_modified: Optional[float] = None,
                            current_hash: Optional[str] = None) -> bool:
        """Check if document has changed since last processing."""
        stored_metadata = self.get_document_metadata(doc_id)

        if not stored_metadata:
            # No metadata stored - treat as changed (new document)
            return True

        # Check modification time if provided
        if current_modified is not None and stored_metadata.get('last_modified') is not None:
            if current_modified > stored_metadata['last_modified']:
                return True

        # Check content hash if provided
        if current_hash is not None and stored_metadata.get('content_hash') is not None:
            if current_hash != stored_metadata['content_hash']:
                return True

        # No change detected
        return False

    def get_source_documents(self, source: str, pattern: Optional[str] = None) -> List[Dict[str, Any]]:
        """Get all document metadata for a specific source.

        Args:
            source: The content source name
            pattern: Optional SQL LIKE pattern for filtering doc_ids

        Returns:
            List of document metadata dictionaries
        """
        with self.Session() as session:
            stmt = select(
                self.document_metadata.c.doc_id,
                self.document_metadata.c.source,
                self.document_metadata.c.last_modified,
                self.document_metadata.c.content_hash,
                self.document_metadata.c.file_size,
                self.document_metadata.c.last_processed_at,
                self.document_metadata.c.processing_stats
            ).where(self.document_metadata.c.source == source)

            if pattern:
                stmt = stmt.where(self.document_metadata.c.doc_id.like(pattern))

            stmt = stmt.order_by(self.document_metadata.c.last_processed_at.desc())

            documents = []
            for row in session.execute(stmt).fetchall():
                metadata = {
                    'doc_id': row.doc_id,
                    'source': row.source,
                    'last_modified': row.last_modified,
                    'content_hash': row.content_hash,
                    'file_size': row.file_size,
                    'last_processed_at': row.last_processed_at,
                    'processing_stats': row.processing_stats
                }
                documents.append(metadata)

            return documents

    def get_document_statistics(self) -> Dict[str, Any]:
        """Get overall document processing statistics.

        Returns:
            Dictionary with statistics including:
            - total_documents: Total number of documents
            - documents_by_source: Count per source
            - recently_processed: Documents processed in last hour
            - documents_with_changes: Documents with multiple versions
        """
        from datetime import datetime, timedelta

        with self.Session() as session:
            # Total documents
            total_docs = session.execute(
                select(func.count(self.document_metadata.c.doc_id))
            ).scalar()

            # Documents by source
            stmt = select(
                self.document_metadata.c.source,
                func.count(self.document_metadata.c.doc_id)
            ).group_by(self.document_metadata.c.source)
            source_counts = session.execute(stmt).fetchall()
            docs_by_source = dict(source_counts)

            # Recently processed (last hour)
            one_hour_ago = datetime.utcnow() - timedelta(hours=1)
            recent_count = session.execute(
                select(func.count(self.document_metadata.c.doc_id)).where(
                    self.document_metadata.c.last_processed_at > one_hour_ago
                )
            ).scalar()

            # Documents with multiple processing (could indicate changes)
            # This would require a more complex query with history tracking
            # For now, we return 0 as we don't track full history
            docs_with_changes = 0

            return {
                'total_documents': total_docs,
                'documents_by_source': docs_by_source,
                'recently_processed': recent_count,
                'documents_with_changes': docs_with_changes
            }