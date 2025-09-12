"""
Document processing queue system for distributed work coordination.

This module provides a work queue implementation that allows multiple workers
to process documents in parallel without duplication. It uses PostgreSQL's
row-level locking to ensure atomic work claiming.

Key Features:
- Config-based run coordination (same config = same run)
- Atomic document claiming with FOR UPDATE SKIP LOCKED
- Automatic retry with exponential backoff
- Worker heartbeat and failure detection
- Link discovery and dynamic queue addition
"""

from .work_queue import WorkQueue, RunCoordinator

__all__ = ['WorkQueue', 'RunCoordinator']