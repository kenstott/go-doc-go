"""
Simple job control package for document processing coordination.

This package provides a simplified job control database interface that eliminates
run_id complexity while providing robust document processing coordination.
"""

from .base import SimpleJobControlDB
from .sqlite import SimpleSQLiteJobControlDB
from .sqlalchemy_impl import SQLAlchemyJobControlDB

__all__ = [
    'SimpleJobControlDB',
    'SimpleSQLiteJobControlDB',
    'SQLAlchemyJobControlDB'
]

# Convenience factory function
def create_job_control(config) -> SimpleJobControlDB:
    """
    Create appropriate job control backend from configuration.

    Args:
        config: Configuration object with get_job_control_config() method

    Returns:
        SimpleJobControlDB instance
    """
    return SimpleJobControlDB.create(config)