"""
SQLite database implementation for pipeline configuration management.
"""

import json
import logging
import sqlite3
import uuid
from pathlib import Path
from typing import List, Optional, Dict, Any, Tuple
from datetime import datetime

import yaml

from .models import (
    Pipeline, PipelineExecution, PipelineTemplate, ExecutionStage,
    ConcurrencyError, PipelineNotFoundError, ValidationError
)

logger = logging.getLogger(__name__)


class PipelineConfigDB:
    """
    SQLite database for managing pipeline configurations with optimistic locking.
    """
    
    def __init__(self, db_path: str = "pipeline_config.db"):
        """
        Initialize the pipeline configuration database.
        
        Args:
            db_path: Path to SQLite database file
        """
        self.db_path = db_path
        self._init_database()

    def _init_database(self):
        """Initialize database with schema."""
        # Ensure directory exists
        Path(self.db_path).parent.mkdir(parents=True, exist_ok=True)
        
        # Read and execute schema
        schema_path = Path(__file__).parent / "schema.sql"
        with open(schema_path, 'r') as f:
            schema_sql = f.read()
        
        with sqlite3.connect(self.db_path) as conn:
            conn.executescript(schema_sql)
            conn.commit()
        
        logger.info(f"Initialized pipeline configuration database: {self.db_path}")

    def _get_connection(self) -> sqlite3.Connection:
        """Get database connection with row factory and proper concurrency settings."""
        conn = sqlite3.connect(self.db_path, timeout=30.0)
        conn.row_factory = sqlite3.Row  # Enable dict-like access

        # Enable WAL mode for better concurrency
        conn.execute("PRAGMA journal_mode=WAL")
        conn.execute("PRAGMA synchronous=NORMAL")
        conn.execute("PRAGMA cache_size=-64000")  # 64MB cache
        conn.execute("PRAGMA temp_store=memory")
        conn.execute("PRAGMA mmap_size=268435456")  # 256MB mmap

        return conn

    # Pipeline CRUD Operations

    def create_pipeline(self, pipeline: Pipeline) -> Pipeline:
        """
        Create a new pipeline configuration.
        
        Args:
            pipeline: Pipeline to create
            
        Returns:
            Created pipeline with assigned ID
            
        Raises:
            ValidationError: If pipeline configuration is invalid
        """
        # Validate YAML configuration
        self._validate_pipeline_config(pipeline.config_yaml)
        
        with self._get_connection() as conn:
            cursor = conn.cursor()
            
            # Check for duplicate name
            cursor.execute("SELECT id FROM pipelines WHERE name = ?", (pipeline.name,))
            if cursor.fetchone():
                raise ValidationError(f"Pipeline with name '{pipeline.name}' already exists")
            
            # Insert pipeline
            tags_json = json.dumps(pipeline.tags) if pipeline.tags else None
            cursor.execute("""
                INSERT INTO pipelines (name, description, config_yaml, created_by, tags, template_name)
                VALUES (?, ?, ?, ?, ?, ?)
            """, (
                pipeline.name,
                pipeline.description,
                pipeline.config_yaml,
                pipeline.created_by,
                tags_json,
                pipeline.template_name
            ))
            
            pipeline_id = cursor.lastrowid
            conn.commit()
            
            # Return created pipeline
            return self.get_pipeline(pipeline_id)

    def get_pipeline(self, pipeline_id: int) -> Pipeline:
        """
        Get pipeline by ID.
        
        Args:
            pipeline_id: Pipeline ID
            
        Returns:
            Pipeline object
            
        Raises:
            PipelineNotFoundError: If pipeline not found
        """
        with self._get_connection() as conn:
            cursor = conn.cursor()
            cursor.execute("SELECT * FROM pipelines WHERE id = ?", (pipeline_id,))
            row = cursor.fetchone()
            
            if not row:
                raise PipelineNotFoundError(f"Pipeline with ID {pipeline_id} not found")
            
            return Pipeline.from_db_row(dict(row))

    def get_pipeline_by_name(self, name: str) -> Pipeline:
        """
        Get pipeline by name.
        
        Args:
            name: Pipeline name
            
        Returns:
            Pipeline object
            
        Raises:
            PipelineNotFoundError: If pipeline not found
        """
        with self._get_connection() as conn:
            cursor = conn.cursor()
            cursor.execute("SELECT * FROM pipelines WHERE name = ?", (name,))
            row = cursor.fetchone()
            
            if not row:
                raise PipelineNotFoundError(f"Pipeline with name '{name}' not found")
            
            return Pipeline.from_db_row(dict(row))

    def list_pipelines(self, active_only: bool = True, tags: Optional[List[str]] = None) -> List[Pipeline]:
        """
        List all pipelines with optional filtering.
        
        Args:
            active_only: If True, return only active pipelines
            tags: Optional list of tags to filter by
            
        Returns:
            List of pipelines
        """
        with self._get_connection() as conn:
            cursor = conn.cursor()
            
            query = "SELECT * FROM pipelines"
            params = []
            
            conditions = []
            if active_only:
                conditions.append("is_active = 1")
            
            if tags:
                # Simple tag matching - could be improved with proper JSON queries
                tag_conditions = []
                for tag in tags:
                    tag_conditions.append("tags LIKE ?")
                    params.append(f"%{tag}%")
                if tag_conditions:
                    conditions.append(f"({' OR '.join(tag_conditions)})")
            
            if conditions:
                query += " WHERE " + " AND ".join(conditions)
            
            query += " ORDER BY updated_at DESC"
            
            cursor.execute(query, params)
            rows = cursor.fetchall()
            
            return [Pipeline.from_db_row(dict(row)) for row in rows]

    def update_pipeline(self, pipeline: Pipeline, expected_version: int) -> Pipeline:
        """
        Update pipeline with optimistic locking.
        
        Args:
            pipeline: Pipeline with updates
            expected_version: Expected current version for concurrency control
            
        Returns:
            Updated pipeline
            
        Raises:
            PipelineNotFoundError: If pipeline not found
            ConcurrencyError: If version mismatch (concurrent modification)
            ValidationError: If configuration is invalid
        """
        # Validate YAML configuration
        self._validate_pipeline_config(pipeline.config_yaml)
        
        with self._get_connection() as conn:
            cursor = conn.cursor()
            
            # Check current version
            cursor.execute("SELECT version FROM pipelines WHERE id = ?", (pipeline.id,))
            row = cursor.fetchone()
            if not row:
                raise PipelineNotFoundError(f"Pipeline with ID {pipeline.id} not found")
            
            current_version = row['version']
            if current_version != expected_version:
                raise ConcurrencyError(
                    f"Pipeline was modified by another process. Current version: {current_version}, expected: {expected_version}",
                    current_version,
                    expected_version
                )
            
            # Update pipeline with incremented version
            new_version = current_version + 1
            tags_json = json.dumps(pipeline.tags) if pipeline.tags else None
            
            cursor.execute("""
                UPDATE pipelines 
                SET name = ?, description = ?, config_yaml = ?, version = ?, 
                    is_active = ?, tags = ?, template_name = ?
                WHERE id = ?
            """, (
                pipeline.name,
                pipeline.description,
                pipeline.config_yaml,
                new_version,
                pipeline.is_active,
                tags_json,
                pipeline.template_name,
                pipeline.id
            ))
            
            if cursor.rowcount == 0:
                raise PipelineNotFoundError(f"Pipeline with ID {pipeline.id} not found")
            
            conn.commit()
            
            # Return updated pipeline
            return self.get_pipeline(pipeline.id)

    def delete_pipeline(self, pipeline_id: int) -> bool:
        """
        Delete pipeline and all related executions.
        
        Args:
            pipeline_id: Pipeline ID to delete
            
        Returns:
            True if deleted, False if not found
        """
        with self._get_connection() as conn:
            cursor = conn.cursor()
            cursor.execute("DELETE FROM pipelines WHERE id = ?", (pipeline_id,))
            deleted = cursor.rowcount > 0
            conn.commit()
            return deleted

    def clone_pipeline(self, source_id: int, new_name: str, created_by: Optional[str] = None) -> Pipeline:
        """
        Clone an existing pipeline with a new name.
        
        Args:
            source_id: ID of pipeline to clone
            new_name: Name for the cloned pipeline
            created_by: Creator of the cloned pipeline
            
        Returns:
            Cloned pipeline
            
        Raises:
            PipelineNotFoundError: If source pipeline not found
            ValidationError: If new name already exists
        """
        source_pipeline = self.get_pipeline(source_id)
        
        cloned = Pipeline(
            name=new_name,
            description=f"Cloned from: {source_pipeline.name}",
            config_yaml=source_pipeline.config_yaml,
            created_by=created_by,
            tags=source_pipeline.tags.copy() if source_pipeline.tags else None,
            template_name=source_pipeline.template_name
        )
        
        return self.create_pipeline(cloned)

    # Template Operations

    def list_templates(self, category: Optional[str] = None) -> List[PipelineTemplate]:
        """
        List available pipeline templates.
        
        Args:
            category: Optional category filter
            
        Returns:
            List of pipeline templates
        """
        with self._get_connection() as conn:
            cursor = conn.cursor()
            
            if category:
                cursor.execute(
                    "SELECT * FROM pipeline_templates WHERE category = ? ORDER BY name",
                    (category,)
                )
            else:
                cursor.execute("SELECT * FROM pipeline_templates ORDER BY category, name")
            
            rows = cursor.fetchall()
            return [PipelineTemplate.from_db_row(dict(row)) for row in rows]

    def create_pipeline_from_template(self, template_id: int, pipeline_name: str, 
                                    created_by: Optional[str] = None) -> Pipeline:
        """
        Create a new pipeline from a template.
        
        Args:
            template_id: Template ID
            pipeline_name: Name for new pipeline
            created_by: Creator of the pipeline
            
        Returns:
            Created pipeline
            
        Raises:
            PipelineNotFoundError: If template not found
        """
        with self._get_connection() as conn:
            cursor = conn.cursor()
            cursor.execute("SELECT * FROM pipeline_templates WHERE id = ?", (template_id,))
            row = cursor.fetchone()
            
            if not row:
                raise PipelineNotFoundError(f"Template with ID {template_id} not found")
            
            template = PipelineTemplate.from_db_row(dict(row))
            
            pipeline = Pipeline(
                name=pipeline_name,
                description=template.description,
                config_yaml=template.config_yaml,
                created_by=created_by,
                tags=template.tags.copy() if template.tags else None,
                template_name=template.name
            )
            
            return self.create_pipeline(pipeline)

    # Validation

    def _validate_pipeline_config(self, config_yaml: str) -> Dict[str, Any]:
        """
        Validate pipeline configuration YAML.
        
        Args:
            config_yaml: YAML configuration string
            
        Returns:
            Parsed configuration dict
            
        Raises:
            ValidationError: If YAML is invalid or missing required fields
        """
        try:
            config = yaml.safe_load(config_yaml)
        except yaml.YAMLError as e:
            raise ValidationError(f"Invalid YAML: {e}")
        
        if not isinstance(config, dict):
            raise ValidationError("Configuration must be a YAML object/dictionary")
        
        # Validate required sections
        required_sections = ['storage']
        for section in required_sections:
            if section not in config:
                raise ValidationError(f"Missing required section: {section}")
        
        # Validate storage configuration
        storage = config['storage']
        if not isinstance(storage, dict):
            raise ValidationError("Storage section must be an object")
        
        # Check for dual storage architecture (new format)
        if 'job' in storage and 'analytics' in storage:
            # Validate dual storage format
            job_storage = storage['job']
            analytics_storage = storage['analytics']
            
            if not isinstance(job_storage, dict):
                raise ValidationError("Storage job section must be an object")
                
            # Analytics can be either a string (registry name) or a dict (full config)
            if isinstance(analytics_storage, str):
                # Registry backend name - this is valid
                pass
            elif isinstance(analytics_storage, dict):
                # Full configuration object
                if 'type' not in analytics_storage:
                    raise ValidationError("Analytics storage type is required")
            else:
                raise ValidationError("Storage analytics must be either a registry name (string) or configuration object")
                
            if 'type' not in job_storage:
                raise ValidationError("Job storage type is required")
        
        
        else:
            raise ValidationError("Storage configuration must specify dual storage (job + analytics)")
        
        return config


class PipelineExecutionTracker:
    """
    Tracks pipeline execution history and progress.
    """
    
    def __init__(self, db: PipelineConfigDB):
        """
        Initialize execution tracker.
        
        Args:
            db: Pipeline configuration database
        """
        self.db = db

    def start_execution(self, pipeline_id: int, run_id: str, config_snapshot: Optional[str] = None,
                       worker_count: int = 1, documents_total: int = 0) -> PipelineExecution:
        """
        Start a new pipeline execution.

        Args:
            pipeline_id: Pipeline ID
            run_id: Pre-computed hash-based run ID from configuration
            config_snapshot: Snapshot of configuration at execution time
            worker_count: Number of workers for this execution
            documents_total: Total number of documents to process

        Returns:
            Created execution record
        """
        # Get pipeline to verify it exists and get current version
        pipeline = self.db.get_pipeline(pipeline_id)
        
        execution = PipelineExecution(
            pipeline_id=pipeline_id,
            pipeline_version=pipeline.version,
            run_id=run_id,
            status="pending",
            started_at=datetime.now(),
            worker_count=worker_count,
            documents_total=documents_total,
            config_snapshot=config_snapshot or pipeline.config_yaml
        )
        
        with self.db._get_connection() as conn:
            cursor = conn.cursor()
            
            metadata_json = json.dumps(execution.metadata) if execution.metadata else None
            cursor.execute("""
                INSERT INTO pipeline_executions (
                    pipeline_id, pipeline_version, run_id, status, started_at,
                    documents_total, worker_count, config_snapshot, metadata
                ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
            """, (
                execution.pipeline_id,
                execution.pipeline_version,
                execution.run_id,
                execution.status,
                execution.started_at.isoformat(),
                execution.documents_total,
                execution.worker_count,
                execution.config_snapshot,
                metadata_json
            ))
            
            execution.id = cursor.lastrowid
            conn.commit()
            
            return execution

    def update_execution_progress(self, run_id: str, documents_processed: int = None,
                                documents_total: int = None, status: str = None,
                                errors_count: int = None, warnings_count: int = None,
                                heartbeat: bool = False) -> bool:
        """
        Update execution progress.
        
        Args:
            run_id: Execution run ID
            documents_processed: Number of documents processed
            documents_total: Total number of documents (if changed)
            status: Execution status
            errors_count: Number of errors
            warnings_count: Number of warnings
            
        Returns:
            True if updated successfully
        """
        with self.db._get_connection() as conn:
            cursor = conn.cursor()
            
            updates = []
            params = []
            
            if documents_processed is not None:
                updates.append("documents_processed = ?")
                params.append(documents_processed)
            
            if documents_total is not None:
                updates.append("documents_total = ?")
                params.append(documents_total)
            
            if status is not None:
                updates.append("status = ?")
                params.append(status)
                
                # Set completion time if status is final
                if status in ['completed', 'failed', 'cancelled']:
                    updates.append("completed_at = ?")
                    params.append(datetime.now().isoformat())
            
            if errors_count is not None:
                updates.append("errors_count = ?")
                params.append(errors_count)
            
            if warnings_count is not None:
                updates.append("warnings_count = ?")
                params.append(warnings_count)
            
            if not updates:
                return False
            
            params.append(run_id)
            query = f"UPDATE pipeline_executions SET {', '.join(updates)} WHERE run_id = ?"
            
            cursor.execute(query, params)
            updated = cursor.rowcount > 0
            conn.commit()
            
            return updated

    def get_execution(self, run_id: str) -> Optional[PipelineExecution]:
        """
        Get execution by run ID.
        
        Args:
            run_id: Execution run ID
            
        Returns:
            Execution record or None if not found
        """
        with self.db._get_connection() as conn:
            cursor = conn.cursor()
            cursor.execute("SELECT * FROM pipeline_executions WHERE run_id = ?", (run_id,))
            row = cursor.fetchone()
            
            if not row:
                return None
            
            return PipelineExecution.from_db_row(dict(row))

    def list_executions(self, pipeline_id: Optional[int] = None, limit: int = 50) -> List[PipelineExecution]:
        """
        List recent executions.

        Args:
            pipeline_id: Optional pipeline ID filter
            limit: Maximum number of executions to return

        Returns:
            List of executions ordered by start time (most recent first)
        """
        with self.db._get_connection() as conn:
            cursor = conn.cursor()

            if pipeline_id:
                cursor.execute("""
                    SELECT * FROM pipeline_executions
                    WHERE pipeline_id = ?
                    ORDER BY started_at DESC
                    LIMIT ?
                """, (pipeline_id, limit))
            else:
                cursor.execute("""
                    SELECT * FROM pipeline_executions
                    ORDER BY started_at DESC
                    LIMIT ?
                """, (limit,))

            rows = cursor.fetchall()
            return [PipelineExecution.from_db_row(dict(row)) for row in rows]

    def get_most_recent_successful_run_id(self, pipeline_id: int) -> Optional[str]:
        """
        Get the most recent successful run_id for a pipeline.

        Args:
            pipeline_id: Pipeline ID

        Returns:
            run_id of the most recent successful execution, or None if no successful runs found
        """
        with self.db._get_connection() as conn:
            cursor = conn.cursor()
            cursor.execute("""
                SELECT run_id FROM pipeline_executions
                WHERE pipeline_id = ? AND status = 'completed'
                ORDER BY started_at DESC
                LIMIT 1
            """, (pipeline_id,))
            row = cursor.fetchone()

            return row['run_id'] if row else None

    def reset_execution(self, run_id: str, pipeline_id: int, config_snapshot: str,
                       worker_count: int = 1, documents_total: int = 0) -> PipelineExecution:
        """
        Reset an existing execution for re-running.

        Args:
            run_id: Execution run ID to reset
            pipeline_id: Pipeline ID
            config_snapshot: Updated configuration snapshot
            worker_count: Number of workers for this execution
            documents_total: Total number of documents (0 for leader to discover)

        Returns:
            Reset execution record
        """
        # Get pipeline to verify it exists and get current version
        pipeline = self.db.get_pipeline(pipeline_id)

        with self.db._get_connection() as conn:
            cursor = conn.cursor()

            # Reset the execution record
            cursor.execute("""
                UPDATE pipeline_executions
                SET status = 'pending',
                    started_at = ?,
                    completed_at = NULL,
                    documents_processed = 0,
                    documents_total = ?,
                    errors_count = 0,
                    warnings_count = 0,
                    worker_count = ?,
                    config_snapshot = ?,
                    pipeline_version = ?
                WHERE run_id = ?
            """, (
                datetime.now().isoformat(),
                documents_total,
                worker_count,
                config_snapshot,
                pipeline.version,
                run_id
            ))

            conn.commit()

            # Return the reset execution
            return self.get_execution(run_id)

    def get_or_reset_execution(self, pipeline_id: int, run_id: str, config_snapshot: Optional[str] = None,
                              worker_count: int = 1, documents_total: int = 0,
                              execution_engine=None) -> Tuple[PipelineExecution, bool]:
        """
        Get an existing execution or reset it if completed/failed, or create new if doesn't exist.

        Args:
            pipeline_id: Pipeline ID
            run_id: Pre-computed hash-based run ID from configuration
            config_snapshot: Snapshot of configuration at execution time
            worker_count: Number of workers for this execution
            documents_total: Total number of documents to process
            execution_engine: Optional execution engine to check if execution is actually active

        Returns:
            Tuple of (execution record, was_reset) where was_reset is True if an existing execution was reset

        Raises:
            RuntimeError: If execution is already running
        """
        existing = self.get_execution(run_id)

        if existing:
            if existing.status in ['running', 'pending']:
                # Check if execution is actually still running
                is_actually_running = False

                if execution_engine:
                    # Check if thread is still alive in execution engine
                    with execution_engine._execution_lock:
                        if run_id in execution_engine._active_executions:
                            active_info = execution_engine._active_executions[run_id]
                            is_actually_running = active_info['thread'].is_alive()
                else:
                    # Fallback: If no execution engine provided, check timestamp for stale detection
                    from datetime import datetime, timedelta
                    try:
                        started_dt = datetime.fromisoformat(existing.started_at.replace('Z', '+00:00'))
                        stale_threshold = datetime.now() - timedelta(minutes=30)  # 30 minute threshold
                        is_actually_running = started_dt > stale_threshold
                        if not is_actually_running:
                            logger.warning(f"Execution {run_id} appears stale (started: {existing.started_at}) - will reset")
                    except Exception as e:
                        logger.warning(f"Could not parse start time for execution {run_id}: {e} - assuming not running")
                        is_actually_running = False

                if is_actually_running:
                    raise RuntimeError(f"Execution {run_id} is already {existing.status}")
                else:
                    # Execution is marked as running but thread is dead or stale - reset it
                    logger.warning(f"Execution {run_id} marked as {existing.status} but thread is dead or stale - resetting")

            # Additional check: verify analytics files exist for this run_id
            # If analytics files are missing, reset the execution regardless of status
            try:
                analytics_exist = self._check_analytics_files_exist(run_id, pipeline_id)
                if not analytics_exist:
                    logger.warning(f"Execution {run_id} marked as {existing.status} but analytics files are missing - resetting")
            except Exception as e:
                logger.warning(f"Error checking analytics files for {run_id}: {e} - proceeding with reset")

            # Reset completed/failed/cancelled execution or dead execution
            logger.info(f"Resetting existing execution {run_id} (was {existing.status})")
            execution = self.reset_execution(
                run_id=run_id,
                pipeline_id=pipeline_id,
                config_snapshot=config_snapshot or existing.config_snapshot,
                worker_count=worker_count,
                documents_total=documents_total
            )
            return execution, True
        else:
            # Create new execution
            execution = self.start_execution(
                pipeline_id=pipeline_id,
                run_id=run_id,
                config_snapshot=config_snapshot,
                worker_count=worker_count,
                documents_total=documents_total
            )
            return execution, False

    def _check_analytics_files_exist(self, run_id: str, pipeline_id: int) -> bool:
        """
        Check if analytics files exist for the given run_id.

        Args:
            run_id: Run ID to check
            pipeline_id: Pipeline ID to get configuration

        Returns:
            True if analytics files exist for this run, False otherwise
        """
        try:
            # Get pipeline configuration to determine analytics storage
            pipeline = self.db.get_pipeline(pipeline_id)
            if not pipeline:
                logger.warning(f"Pipeline {pipeline_id} not found - cannot check analytics files")
                return True  # Assume exists to be safe

            # Parse pipeline configuration
            import yaml
            try:
                pipeline_config = yaml.safe_load(pipeline.config_yaml)
            except yaml.YAMLError as e:
                logger.warning(f"Invalid pipeline configuration YAML: {e}")
                return True  # Assume exists to be safe

            # Create analytics storage and check if run exists
            from ..storage_adapters.factory import StorageFactory
            from ..config import Config

            # Load analytics registry for string reference resolution
            go_config = Config()
            registry = go_config.list_analytics_backends()

            # Create analytics storage using proper factory method
            job_storage, analytics_storage = StorageFactory.create_from_pipeline_config(pipeline_config, registry)

            # Check if analytics files exist for this run
            if hasattr(analytics_storage, 'has_run'):
                result = analytics_storage.has_run(run_id)
                logger.debug(f"Analytics files exist check for {run_id}: {result}")
                return result
            else:
                # If storage doesn't implement has_run, assume files exist
                logger.debug(f"Analytics storage doesn't implement has_run method - assuming files exist")
                return True

        except Exception as e:
            logger.warning(f"Error checking analytics files for run {run_id}: {e}")
            return True  # Assume exists to be safe on error