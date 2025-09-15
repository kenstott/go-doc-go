"""
Pipeline execution engine that integrates pipeline configurations with document processing.
"""

import asyncio
import json
import logging
import threading
import time
import uuid
import yaml
import io
from datetime import datetime
from typing import Dict, Any, Optional, List
from collections import deque

from ..config import Config
from ..config_db import PipelineConfigDB, PipelineExecutionTracker, PipelineExecution
from ..main import ingest_documents
from .progress_monitor import ProgressMonitor


logger = logging.getLogger(__name__)


class ExecutionLogCapture(logging.Handler):
    """Captures logs for a specific execution."""
    
    def __init__(self, run_id: str, log_buffer: deque, max_lines: int = 1000):
        super().__init__()
        self.run_id = run_id
        self.log_buffer = log_buffer
        self.max_lines = max_lines
        
    def emit(self, record):
        """Capture log record."""
        try:
            msg = self.format(record)
            timestamp = datetime.now().isoformat()
            log_entry = {
                'timestamp': timestamp,
                'level': record.levelname,
                'message': msg,
                'module': record.name
            }
            
            # Add to buffer (thread-safe since deque.append is atomic)
            self.log_buffer.append(log_entry)
            
            # Limit buffer size
            while len(self.log_buffer) > self.max_lines:
                self.log_buffer.popleft()
                
        except Exception:
            pass  # Don't let logging errors break execution


class PipelineExecutionEngine:
    """
    Executes pipeline configurations using the main document processing system.
    """
    
    def __init__(self, config: Config, db_path: Optional[str] = None):
        """
        Initialize pipeline execution engine.
        
        Args:
            config: Go-Doc-Go configuration
            db_path: Optional path to pipeline config database
        """
        self.config = config
        self.pipeline_db = PipelineConfigDB(db_path or 'pipeline_config.db')
        self.execution_tracker = PipelineExecutionTracker(self.pipeline_db)
        self.progress_monitor = ProgressMonitor()
        
        # Track active executions
        self._active_executions = {}
        self._execution_lock = threading.Lock()
        
        # Log buffers for each execution
        self._execution_logs = {}
        
        logger.info("Pipeline execution engine initialized")
    
    def execute_pipeline(self, pipeline_id: int, execution_params: Optional[Dict[str, Any]] = None) -> PipelineExecution:
        """
        Execute a pipeline by ID.
        
        Args:
            pipeline_id: Pipeline to execute
            execution_params: Optional execution parameters (worker_count, documents_total, etc.)
            
        Returns:
            PipelineExecution object with execution details
            
        Raises:
            ValueError: If pipeline not found or invalid
            RuntimeError: If execution fails to start
        """
        # Get pipeline configuration
        pipeline = self.pipeline_db.get_pipeline(pipeline_id)
        if not pipeline:
            raise ValueError(f"Pipeline {pipeline_id} not found")
        
        if not pipeline.is_active:
            raise ValueError(f"Pipeline {pipeline_id} is not active")
        
        logger.info(f"Starting execution of pipeline: {pipeline.name} (ID: {pipeline_id})")
        
        # Parse pipeline configuration
        try:
            pipeline_config = yaml.safe_load(pipeline.config_yaml)
        except yaml.YAMLError as e:
            raise ValueError(f"Invalid pipeline configuration YAML: {e}")
        
        # Create execution record
        execution_params = execution_params or {}
        execution = self.execution_tracker.start_execution(
            pipeline_id=pipeline_id,
            config_snapshot=pipeline.config_yaml,
            worker_count=execution_params.get('worker_count', 1),
            documents_total=execution_params.get('documents_total', 0)
        )
        
        # Notify progress monitor that execution is starting
        self.progress_monitor.execution_started(execution.run_id, pipeline_config)
        
        # Start execution in background thread
        execution_thread = threading.Thread(
            target=self._execute_pipeline_async,
            args=(execution, pipeline_config),
            daemon=True,
            name=f"pipeline-{pipeline_id}-{execution.run_id}"
        )
        
        # Create log buffer for this execution
        log_buffer = deque(maxlen=1000)
        self._execution_logs[execution.run_id] = log_buffer
        
        with self._execution_lock:
            self._active_executions[execution.run_id] = {
                'execution': execution,
                'thread': execution_thread,
                'start_time': datetime.now(),
                'pipeline_config': pipeline_config,
                'log_buffer': log_buffer
            }
        
        execution_thread.start()
        
        logger.info(f"Pipeline execution started: {execution.run_id}")
        return execution
    
    def _execute_pipeline_async(self, execution: PipelineExecution, pipeline_config: Dict[str, Any]):
        """
        Execute pipeline asynchronously in background thread.
        
        Args:
            execution: Execution record
            pipeline_config: Parsed pipeline configuration
        """
        run_id = execution.run_id
        
        # Set up log capture for this execution
        log_handler = None
        if run_id in self._execution_logs:
            log_handler = ExecutionLogCapture(
                run_id=run_id,
                log_buffer=self._execution_logs[run_id]
            )
            log_handler.setFormatter(logging.Formatter(
                '%(asctime)s - %(name)s - %(levelname)s - %(message)s'
            ))
            # Add handler to root logger to capture all logs
            logging.getLogger().addHandler(log_handler)
            log_handler.setLevel(logging.DEBUG)
        
        try:
            logger.info(f"Executing pipeline in background: {run_id}")
            
            # Update status to running
            self.execution_tracker.update_execution_progress(
                run_id=run_id,
                status='running'
            )
            
            # Register progress callback
            def progress_callback(stats: Dict[str, Any]):
                """Callback to update execution progress."""
                logger.info(f"Progress callback received for {run_id}: {stats}")
                
                # Update documents_total if we got the actual count
                documents_total = execution.documents_total
                if stats.get('documents_total') and stats.get('documents_total') > documents_total:
                    documents_total = stats.get('documents_total')
                    # Update the execution record with correct total
                    execution.documents_total = documents_total
                    logger.info(f"Updated documents_total for {run_id}: {documents_total}")
                
                # Update execution tracker
                self.execution_tracker.update_execution_progress(
                    run_id=run_id,
                    documents_processed=stats.get('documents', 0),
                    documents_total=documents_total or stats.get('documents', 0),
                    status='running'
                )
                
                # Notify progress monitor with enhanced 2-pass stats
                enhanced_stats = stats.copy()
                enhanced_stats.update({
                    'documents_total': documents_total or stats.get('documents_total', 0),
                    'documents_parsed': stats.get('documents_parsed', stats.get('documents', 0)),
                    'documents_embedded': stats.get('documents_embedded', 0),
                    'parsing_complete': stats.get('parsing_complete', False),
                    'embedding_complete': stats.get('embedding_complete', False)
                })
                self.progress_monitor.update_progress(run_id, enhanced_stats)
            
            # Create execution-specific configuration
            exec_config = self._create_execution_config(pipeline_config)
            
            # Execute the pipeline using main ingestion system
            stats = ingest_documents(
                config=exec_config,
                source_configs=pipeline_config.get('content_sources'),
                max_link_depth=pipeline_config.get('max_link_depth'),
                processing_mode=pipeline_config.get('processing', {}).get('mode', 'single'),
                progress_callback=progress_callback
            )
            
            logger.info(f"Pipeline execution completed successfully: {run_id}")
            logger.info(f"Execution stats: {stats}")
            
            # Update final status
            self.execution_tracker.update_execution_progress(
                run_id=run_id,
                documents_processed=stats.get('documents', 0),
                documents_total=stats.get('documents', 0),
                status='completed',
                errors_count=stats.get('errors', 0),
                warnings_count=stats.get('warnings', 0)
            )
            
            # Execution tracking is already updated in update_execution_progress above
            
            # Notify progress monitor
            self.progress_monitor.execution_completed(run_id, stats)
            
        except Exception as e:
            logger.error(f"Pipeline execution failed: {run_id} - {str(e)}")
            import traceback
            error_details = traceback.format_exc()
            
            # Update status to failed
            self.execution_tracker.update_execution_progress(
                run_id=run_id,
                status='failed',
                errors_count=1
            )
            
            # Log the detailed error for debugging
            logger.error(f"Pipeline execution error details: {error_details}")
            
            # Notify progress monitor
            self.progress_monitor.execution_failed(run_id, str(e))
            
        finally:
            # Remove log handler if it was added
            if log_handler:
                logging.getLogger().removeHandler(log_handler)
            
            # Clean up active execution tracking
            with self._execution_lock:
                if run_id in self._active_executions:
                    del self._active_executions[run_id]
    
    def _create_execution_config(self, pipeline_config: Dict[str, Any]) -> Config:
        """
        Create execution-specific configuration from pipeline config.
        
        Args:
            pipeline_config: Pipeline configuration dictionary
            
        Returns:
            Config object for execution
        """
        import os
        
        # Start with base configuration
        exec_config_dict = self.config.config.copy()
        
        # Override with pipeline-specific settings
        if 'storage' in pipeline_config:
            exec_config_dict['storage'] = pipeline_config['storage']
            
            # Fix SQLite path handling
            if exec_config_dict['storage'].get('backend') == 'sqlite':
                # Check if there's a nested sqlite config with path
                sqlite_config = exec_config_dict['storage'].get('sqlite', {})
                if 'path' in sqlite_config:
                    # Use the nested sqlite path
                    db_path = sqlite_config['path']
                else:
                    # Get the base path
                    db_path = exec_config_dict['storage'].get('path', ':memory:')
                    # If it's a directory path, append a default database name
                    if db_path != ':memory:' and not db_path.startswith(':'):
                        if not db_path.endswith('.db'):
                            # It's a directory, append database filename
                            db_path = os.path.join(db_path, 'pipeline_execution.db')
                
                # Update the path in storage config for SQLite
                exec_config_dict['storage']['path'] = db_path
                
                # Ensure database directory exists
                if db_path != ':memory:' and not db_path.startswith(':'):
                    db_dir = os.path.dirname(db_path)
                    if db_dir and not os.path.exists(db_dir):
                        logger.info(f"Creating database directory: {db_dir}")
                        os.makedirs(db_dir, exist_ok=True)
        
        if 'embedding' in pipeline_config:
            exec_config_dict['embedding'] = pipeline_config['embedding']
        
        if 'relationship_detection' in pipeline_config:
            exec_config_dict['relationship_detection'] = pipeline_config['relationship_detection']
        
        if 'processing' in pipeline_config:
            exec_config_dict['processing'] = pipeline_config['processing']
        
        # Create new Config instance
        exec_config = Config()
        exec_config.config = exec_config_dict
        
        # Only initialize database for single storage mode
        # Dual storage is handled by TwoPassProcessor
        storage_config = exec_config_dict.get('storage', {})
        if not ('job' in storage_config and 'analytics' in storage_config):
            # Single storage mode - initialize database
            exec_config.initialize_database()
        
        return exec_config
    
    def get_execution_status(self, run_id: str) -> Optional[Dict[str, Any]]:
        """
        Get current execution status.
        
        Args:
            run_id: Execution run ID
            
        Returns:
            Execution status dictionary or None if not found
        """
        # Get from database
        execution = self.execution_tracker.get_execution(run_id)
        if not execution:
            return None
        
        status = execution.to_dict()
        
        # Add real-time info if execution is active
        with self._execution_lock:
            if run_id in self._active_executions:
                active_info = self._active_executions[run_id]
                status['is_active'] = True
                status['actual_start_time'] = active_info['start_time'].isoformat()
                status['thread_alive'] = active_info['thread'].is_alive()
            else:
                status['is_active'] = False
        
        return status
    
    def cancel_execution(self, run_id: str) -> bool:
        """
        Cancel an active execution.
        
        Args:
            run_id: Execution run ID
            
        Returns:
            True if cancellation initiated, False if not found or not active
        """
        with self._execution_lock:
            if run_id not in self._active_executions:
                return False
            
            active_info = self._active_executions[run_id]
            
            # Mark as cancelled in database
            self.execution_tracker.update_execution_progress(
                run_id=run_id,
                status='cancelled'
            )
            
            # Notify progress monitor
            self.progress_monitor.execution_cancelled(run_id)
            
            # Note: We can't forcibly kill threads in Python safely
            # The execution will check status periodically and exit gracefully
            logger.warning(f"Cancellation requested for execution: {run_id}")
            logger.warning("Note: Execution will stop at next checkpoint")
            
            return True
    
    def list_active_executions(self) -> List[Dict[str, Any]]:
        """
        List all currently active executions.
        
        Returns:
            List of active execution status dictionaries
        """
        active = []
        
        with self._execution_lock:
            for run_id, info in self._active_executions.items():
                execution = info['execution']
                status = execution.to_dict()
                status['is_active'] = True
                status['actual_start_time'] = info['start_time'].isoformat()
                status['thread_alive'] = info['thread'].is_alive()
                active.append(status)
        
        return active
    
    def get_execution_logs(self, run_id: str, start_index: int = 0) -> Dict[str, Any]:
        """
        Get execution logs for a specific run.
        
        Args:
            run_id: Execution run ID
            start_index: Starting index for log entries (for pagination)
            
        Returns:
            Dict with logs and metadata
        """
        if run_id not in self._execution_logs:
            return {
                'run_id': run_id,
                'logs': [],
                'total_count': 0,
                'start_index': start_index
            }
        
        log_buffer = self._execution_logs[run_id]
        logs = list(log_buffer)  # Convert deque to list
        
        # Return logs from start_index onwards
        return {
            'run_id': run_id,
            'logs': logs[start_index:] if start_index < len(logs) else [],
            'total_count': len(logs),
            'start_index': start_index,
            'has_more': start_index + len(logs[start_index:]) < len(logs)
        }
    
    def cleanup_completed_executions(self, max_age_hours: int = 24):
        """
        Clean up tracking for completed executions older than specified age.
        
        Args:
            max_age_hours: Maximum age in hours for completed executions to keep
        """
        cutoff_time = datetime.now().timestamp() - (max_age_hours * 3600)
        
        with self._execution_lock:
            # Find executions to remove
            to_remove = []
            for run_id, info in self._active_executions.items():
                if info['start_time'].timestamp() < cutoff_time:
                    if not info['thread'].is_alive():
                        to_remove.append(run_id)
            
            # Remove old completed executions
            for run_id in to_remove:
                del self._active_executions[run_id]
                # Also clean up log buffer
                if run_id in self._execution_logs:
                    del self._execution_logs[run_id]
                logger.debug(f"Cleaned up completed execution tracking: {run_id}")
        
        logger.info(f"Cleaned up {len(to_remove)} completed execution records")