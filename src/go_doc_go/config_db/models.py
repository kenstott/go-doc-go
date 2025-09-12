"""
Data models for pipeline configuration database.
"""

import json
from dataclasses import dataclass, asdict
from datetime import datetime
from typing import Optional, List, Dict, Any


@dataclass
class Pipeline:
    """Pipeline configuration model."""
    id: Optional[int] = None
    name: str = ""
    description: str = ""
    config_yaml: str = ""
    version: int = 1
    created_at: Optional[datetime] = None
    updated_at: Optional[datetime] = None
    created_by: Optional[str] = None
    is_active: bool = True
    tags: Optional[List[str]] = None
    template_name: Optional[str] = None

    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary for JSON serialization."""
        result = asdict(self)
        # Convert datetime objects to ISO strings
        if self.created_at:
            result['created_at'] = self.created_at.isoformat()
        if self.updated_at:
            result['updated_at'] = self.updated_at.isoformat()
        return result

    @classmethod
    def from_db_row(cls, row: Dict[str, Any]) -> 'Pipeline':
        """Create Pipeline from database row."""
        tags = None
        if row.get('tags'):
            try:
                tags = json.loads(row['tags'])
            except json.JSONDecodeError:
                tags = []
        
        return cls(
            id=row.get('id'),
            name=row.get('name', ''),
            description=row.get('description', ''),
            config_yaml=row.get('config_yaml', ''),
            version=row.get('version', 1),
            created_at=datetime.fromisoformat(row['created_at']) if row.get('created_at') else None,
            updated_at=datetime.fromisoformat(row['updated_at']) if row.get('updated_at') else None,
            created_by=row.get('created_by'),
            is_active=bool(row.get('is_active', True)),
            tags=tags,
            template_name=row.get('template_name')
        )


@dataclass
class PipelineExecution:
    """Pipeline execution tracking model."""
    id: Optional[int] = None
    pipeline_id: int = 0
    pipeline_version: int = 1
    run_id: str = ""
    status: str = "pending"
    started_at: Optional[datetime] = None
    completed_at: Optional[datetime] = None
    documents_processed: int = 0
    documents_total: int = 0
    errors_count: int = 0
    warnings_count: int = 0
    worker_count: int = 1
    config_snapshot: Optional[str] = None
    execution_log: Optional[str] = None
    metadata: Optional[Dict[str, Any]] = None

    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary for JSON serialization."""
        result = asdict(self)
        # Convert datetime objects to ISO strings
        if self.started_at:
            result['started_at'] = self.started_at.isoformat()
        if self.completed_at:
            result['completed_at'] = self.completed_at.isoformat()
        return result

    @classmethod
    def from_db_row(cls, row: Dict[str, Any]) -> 'PipelineExecution':
        """Create PipelineExecution from database row."""
        metadata = None
        if row.get('metadata'):
            try:
                metadata = json.loads(row['metadata'])
            except json.JSONDecodeError:
                metadata = {}
        
        return cls(
            id=row.get('id'),
            pipeline_id=row.get('pipeline_id', 0),
            pipeline_version=row.get('pipeline_version', 1),
            run_id=row.get('run_id', ''),
            status=row.get('status', 'pending'),
            started_at=datetime.fromisoformat(row['started_at']) if row.get('started_at') else None,
            completed_at=datetime.fromisoformat(row['completed_at']) if row.get('completed_at') else None,
            documents_processed=row.get('documents_processed', 0),
            documents_total=row.get('documents_total', 0),
            errors_count=row.get('errors_count', 0),
            warnings_count=row.get('warnings_count', 0),
            worker_count=row.get('worker_count', 1),
            config_snapshot=row.get('config_snapshot'),
            execution_log=row.get('execution_log'),
            metadata=metadata
        )

    @property
    def duration_seconds(self) -> Optional[int]:
        """Calculate execution duration in seconds."""
        if self.started_at and self.completed_at:
            return int((self.completed_at - self.started_at).total_seconds())
        return None

    @property
    def progress_percentage(self) -> float:
        """Calculate progress percentage."""
        if self.documents_total > 0:
            return (self.documents_processed / self.documents_total) * 100
        return 0.0


@dataclass
class ExecutionStage:
    """Individual stage within a pipeline execution."""
    id: Optional[int] = None
    execution_id: int = 0
    stage_name: str = ""
    status: str = "pending"
    started_at: Optional[datetime] = None
    completed_at: Optional[datetime] = None
    documents_processed: int = 0
    progress_percentage: float = 0.0
    stage_log: Optional[str] = None

    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary for JSON serialization."""
        result = asdict(self)
        # Convert datetime objects to ISO strings
        if self.started_at:
            result['started_at'] = self.started_at.isoformat()
        if self.completed_at:
            result['completed_at'] = self.completed_at.isoformat()
        return result

    @classmethod
    def from_db_row(cls, row: Dict[str, Any]) -> 'ExecutionStage':
        """Create ExecutionStage from database row."""
        return cls(
            id=row.get('id'),
            execution_id=row.get('execution_id', 0),
            stage_name=row.get('stage_name', ''),
            status=row.get('status', 'pending'),
            started_at=datetime.fromisoformat(row['started_at']) if row.get('started_at') else None,
            completed_at=datetime.fromisoformat(row['completed_at']) if row.get('completed_at') else None,
            documents_processed=row.get('documents_processed', 0),
            progress_percentage=row.get('progress_percentage', 0.0),
            stage_log=row.get('stage_log')
        )


@dataclass
class PipelineTemplate:
    """Pipeline template model."""
    id: Optional[int] = None
    name: str = ""
    description: str = ""
    category: str = ""
    config_yaml: str = ""
    is_builtin: bool = False
    created_at: Optional[datetime] = None
    tags: Optional[List[str]] = None

    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary for JSON serialization."""
        result = asdict(self)
        # Convert datetime objects to ISO strings
        if self.created_at:
            result['created_at'] = self.created_at.isoformat()
        return result

    @classmethod
    def from_db_row(cls, row: Dict[str, Any]) -> 'PipelineTemplate':
        """Create PipelineTemplate from database row."""
        tags = None
        if row.get('tags'):
            try:
                tags = json.loads(row['tags'])
            except json.JSONDecodeError:
                tags = []
        
        return cls(
            id=row.get('id'),
            name=row.get('name', ''),
            description=row.get('description', ''),
            category=row.get('category', ''),
            config_yaml=row.get('config_yaml', ''),
            is_builtin=bool(row.get('is_builtin', False)),
            created_at=datetime.fromisoformat(row['created_at']) if row.get('created_at') else None,
            tags=tags
        )


class ConcurrencyError(Exception):
    """Raised when a concurrent modification is detected."""
    def __init__(self, message: str, current_version: int, expected_version: int):
        super().__init__(message)
        self.current_version = current_version
        self.expected_version = expected_version


class PipelineNotFoundError(Exception):
    """Raised when a requested pipeline is not found."""
    pass


class ValidationError(Exception):
    """Raised when pipeline configuration validation fails."""
    pass