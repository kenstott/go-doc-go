"""
Storage adapter system for dual OLTP/OLAP architecture.

Provides separate storage backends for:
- Job coordination (OLTP): Transient, mutable state during processing
- Analytics (OLAP): Permanent, append-only archive of results
"""

from .base import JobStorage, AnalyticsStorage
from .factory import StorageFactory

__all__ = [
    'JobStorage',
    'AnalyticsStorage', 
    'StorageFactory'
]