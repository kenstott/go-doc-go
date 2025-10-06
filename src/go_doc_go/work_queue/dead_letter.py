"""
Dead letter queue implementation for failed document processing.

This module provides dead letter queue functionality on top of the simple job control system.
"""

import json
import logging
from dataclasses import dataclass
from datetime import datetime, timedelta
from typing import Dict, Any, Optional, List

logger = logging.getLogger(__name__)


@dataclass
class DeadLetterItem:
    """Represents a failed document in the dead letter queue."""
    queue_id: int
    doc_id: str
    run_id: str
    source_name: str
    failed_at: str
    retry_count: int
    error_message: str
    error_details: Optional[str] = None
    metadata: Optional[Dict[str, Any]] = None


@dataclass
class FailurePattern:
    """Represents a pattern of failures."""
    error_type: str
    frequency: int
    affected_documents: int
    first_occurrence: str
    last_occurrence: str
    sample_error_messages: List[str]
    affected_sources: List[str]


class DeadLetterQueue:
    """
    Dead letter queue for managing failed documents.

    This wraps the simple job control system to provide dead letter functionality.
    """

    def __init__(self, job_control_db):
        """
        Initialize dead letter queue.

        Args:
            job_control_db: Simple job control database instance
        """
        self.db = job_control_db
        self.max_retries = getattr(job_control_db, 'max_retries', 3)

    def list_dead_letter_items(self, run_id: Optional[str] = None,
                               limit: int = 50) -> List[DeadLetterItem]:
        """
        List items in the dead letter queue.

        Args:
            run_id: Optional run ID to filter by
            limit: Maximum number of items to return

        Returns:
            List of dead letter items
        """
        items = []

        # For SQLite implementation
        if hasattr(self.db, '_get_connection'):
            with self.db._get_connection() as conn:
                query = """
                    SELECT
                        ROWID as queue_id,
                        doc_id,
                        source,
                        claimed_at as failed_at,
                        retry_count,
                        error_message,
                        metadata,
                        claimed_by as run_id
                    FROM document_queue
                    WHERE (status = 'failed' OR
                           (status = 'pending' AND retry_count >= ?))
                """
                params = [self.max_retries]

                if run_id:
                    query += " AND claimed_by = ?"
                    params.append(run_id)

                query += f" ORDER BY claimed_at DESC LIMIT {limit}"

                cursor = conn.execute(query, params)

                for row in cursor.fetchall():
                    metadata = None
                    if row['metadata']:
                        try:
                            metadata = json.loads(row['metadata'])
                        except json.JSONDecodeError:
                            metadata = {'raw': row['metadata']}

                    items.append(DeadLetterItem(
                        queue_id=row['queue_id'],
                        doc_id=row['doc_id'],
                        run_id=row['run_id'] or '',
                        source_name=row['source'],
                        failed_at=row['failed_at'] or '',
                        retry_count=row['retry_count'],
                        error_message=row['error_message'] or 'Unknown error',
                        error_details=None,
                        metadata=metadata
                    ))

        # For SQLAlchemy implementation
        elif hasattr(self.db, 'Session'):
            with self.db.Session() as session:
                from sqlalchemy import and_, or_

                query = session.query(self.db.document_queue)

                # Filter for failed or max-retry documents
                query = query.filter(
                    or_(
                        self.db.document_queue.c.status == 'failed',
                        and_(
                            self.db.document_queue.c.status == 'pending',
                            self.db.document_queue.c.retry_count >= self.max_retries
                        )
                    )
                )

                if run_id:
                    query = query.filter(self.db.document_queue.c.claimed_by == run_id)

                query = query.order_by(self.db.document_queue.c.claimed_at.desc())
                query = query.limit(limit)

                for row in query:
                    metadata = None
                    if row.metadata:
                        try:
                            metadata = json.loads(row.metadata)
                        except json.JSONDecodeError:
                            metadata = {'raw': row.metadata}

                    items.append(DeadLetterItem(
                        queue_id=row.id if hasattr(row, 'id') else 0,
                        doc_id=row.doc_id,
                        run_id=row.claimed_by or '',
                        source_name=row.source,
                        failed_at=str(row.claimed_at) if row.claimed_at else '',
                        retry_count=row.retry_count,
                        error_message=row.error_message or 'Unknown error',
                        error_details=None,
                        metadata=metadata
                    ))

        return items

    def retry_from_dead_letter(self, queue_id: int) -> bool:
        """
        Retry a specific dead letter item.

        Args:
            queue_id: Queue ID of the item to retry

        Returns:
            True if successfully moved back to queue
        """
        # For SQLite implementation
        if hasattr(self.db, '_get_connection'):
            with self.db._get_connection() as conn:
                result = conn.execute("""
                    UPDATE document_queue
                    SET status = 'pending',
                        retry_count = 0,
                        error_message = NULL,
                        claimed_by = NULL,
                        claimed_at = NULL
                    WHERE ROWID = ? AND
                          (status = 'failed' OR
                           (status = 'pending' AND retry_count >= ?))
                """, (queue_id, self.max_retries))

                conn.commit()
                return result.rowcount > 0

        # For SQLAlchemy implementation
        elif hasattr(self.db, 'Session'):
            with self.db.Session() as session:
                from sqlalchemy import and_, or_

                result = session.execute(
                    self.db.document_queue.update()
                    .where(
                        and_(
                            self.db.document_queue.c.id == queue_id,
                            or_(
                                self.db.document_queue.c.status == 'failed',
                                and_(
                                    self.db.document_queue.c.status == 'pending',
                                    self.db.document_queue.c.retry_count >= self.max_retries
                                )
                            )
                        )
                    )
                    .values(
                        status='pending',
                        retry_count=0,
                        error_message=None,
                        claimed_by=None,
                        claimed_at=None
                    )
                )

                session.commit()
                return result.rowcount > 0

        return False

    def purge_old_items(self, days_old: int) -> int:
        """
        Purge dead letter items older than specified days.

        Args:
            days_old: Age threshold in days

        Returns:
            Number of items purged
        """
        cutoff_date = datetime.now() - timedelta(days=days_old)

        # For SQLite implementation
        if hasattr(self.db, '_get_connection'):
            with self.db._get_connection() as conn:
                result = conn.execute("""
                    DELETE FROM document_queue
                    WHERE (status = 'failed' OR
                           (status = 'pending' AND retry_count >= ?))
                    AND claimed_at < ?
                """, (self.max_retries, cutoff_date.isoformat()))

                conn.commit()
                return result.rowcount

        # For SQLAlchemy implementation
        elif hasattr(self.db, 'Session'):
            with self.db.Session() as session:
                from sqlalchemy import and_, or_

                result = session.execute(
                    self.db.document_queue.delete()
                    .where(
                        and_(
                            or_(
                                self.db.document_queue.c.status == 'failed',
                                and_(
                                    self.db.document_queue.c.status == 'pending',
                                    self.db.document_queue.c.retry_count >= self.max_retries
                                )
                            ),
                            self.db.document_queue.c.claimed_at < cutoff_date
                        )
                    )
                )

                session.commit()
                return result.rowcount

        return 0

    def close(self):
        """Close database connection if needed."""
        # Most implementations handle this automatically
        pass


class DeadLetterProcessor:
    """
    Processor for analyzing and handling dead letter queue items.
    """

    def __init__(self, job_control_db):
        """
        Initialize dead letter processor.

        Args:
            job_control_db: Simple job control database instance
        """
        self.db = job_control_db
        self.dlq = DeadLetterQueue(job_control_db)

    def analyze_failure_patterns(self, run_id: Optional[str] = None) -> List[FailurePattern]:
        """
        Analyze failure patterns in the dead letter queue.

        Args:
            run_id: Optional run ID to filter by

        Returns:
            List of failure patterns
        """
        patterns = []
        pattern_data = {}

        # Get all dead letter items
        items = self.dlq.list_dead_letter_items(run_id=run_id, limit=1000)

        # Group by error type
        for item in items:
            # Extract error type from message
            error_type = item.error_message.split(':')[0] if ':' in item.error_message else item.error_message
            error_type = error_type.strip()[:50]  # Limit length

            if error_type not in pattern_data:
                pattern_data[error_type] = {
                    'count': 0,
                    'documents': set(),
                    'sources': set(),
                    'messages': [],
                    'first_occurrence': item.failed_at,
                    'last_occurrence': item.failed_at
                }

            pattern_data[error_type]['count'] += 1
            pattern_data[error_type]['documents'].add(item.doc_id)
            pattern_data[error_type]['sources'].add(item.source_name)
            pattern_data[error_type]['messages'].append(item.error_message)

            # Update occurrence times
            if item.failed_at < pattern_data[error_type]['first_occurrence']:
                pattern_data[error_type]['first_occurrence'] = item.failed_at
            if item.failed_at > pattern_data[error_type]['last_occurrence']:
                pattern_data[error_type]['last_occurrence'] = item.failed_at

        # Convert to FailurePattern objects
        for error_type, data in pattern_data.items():
            patterns.append(FailurePattern(
                error_type=error_type,
                frequency=data['count'],
                affected_documents=len(data['documents']),
                first_occurrence=data['first_occurrence'],
                last_occurrence=data['last_occurrence'],
                sample_error_messages=list(set(data['messages']))[:5],
                affected_sources=list(data['sources'])
            ))

        # Sort by frequency
        patterns.sort(key=lambda p: p.frequency, reverse=True)

        return patterns

    def get_retry_candidates(self, max_age_hours: int = 24) -> List[DeadLetterItem]:
        """
        Get candidates for retry based on age.

        Args:
            max_age_hours: Maximum age in hours for retry candidates

        Returns:
            List of dead letter items suitable for retry
        """
        all_items = self.dlq.list_dead_letter_items(limit=1000)
        cutoff_time = datetime.now() - timedelta(hours=max_age_hours)

        candidates = []
        for item in all_items:
            try:
                failed_time = datetime.fromisoformat(item.failed_at.replace('Z', '+00:00'))
                if failed_time >= cutoff_time:
                    candidates.append(item)
            except Exception:
                # Skip items with invalid timestamps
                continue

        return candidates

    def close(self):
        """Close connections."""
        self.dlq.close()