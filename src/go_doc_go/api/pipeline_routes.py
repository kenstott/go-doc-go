"""
Flask API routes for pipeline configuration management.
"""

import json
import logging
import os
import tempfile
import yaml
from datetime import datetime, timedelta
from flask import Blueprint, request, jsonify, send_file
from typing import Dict, Any, List
from werkzeug.exceptions import BadRequest, NotFound, Conflict, UnprocessableEntity

from ..config_db import (
    PipelineConfigDB, PipelineExecutionTracker,
    Pipeline, PipelineExecution, PipelineTemplate,
    ConcurrencyError, PipelineNotFoundError, ValidationError
)
from ..pipeline import PipelineExecutionEngine
from ..config import Config

logger = logging.getLogger(__name__)

# Create Blueprint
pipeline_bp = Blueprint('pipeline', __name__, url_prefix='/api/pipelines')

# Initialize database connection
# TODO: Make this configurable or use dependency injection
_db = None
_execution_tracker = None
_execution_engine = None

def get_db() -> PipelineConfigDB:
    """Get or create database connection."""
    global _db
    if _db is None:
        db_path = os.environ.get('PIPELINE_CONFIG_DB', 'pipeline_config.db')
        _db = PipelineConfigDB(db_path)
    return _db

def get_execution_tracker() -> PipelineExecutionTracker:
    """Get or create execution tracker."""
    global _execution_tracker
    if _execution_tracker is None:
        _execution_tracker = PipelineExecutionTracker(get_db())
    return _execution_tracker


def get_execution_engine() -> PipelineExecutionEngine:
    """Get or create the execution engine."""
    global _execution_engine
    if _execution_engine is None:
        from ..config import Config
        # Get the base config from Flask app context
        config = Config()  # Will use default config path or environment variable
        _execution_engine = PipelineExecutionEngine(config)
    return _execution_engine


# Error handlers
@pipeline_bp.errorhandler(ValidationError)
def handle_validation_error(e):
    """Handle validation errors."""
    return jsonify({'error': 'Validation Error', 'message': str(e)}), 400

@pipeline_bp.errorhandler(PipelineNotFoundError)
def handle_not_found_error(e):
    """Handle pipeline not found errors."""
    return jsonify({'error': 'Pipeline Not Found', 'message': str(e)}), 404

@pipeline_bp.errorhandler(ConcurrencyError)
def handle_concurrency_error(e):
    """Handle concurrency errors."""
    return jsonify({
        'error': 'Concurrency Conflict',
        'message': str(e),
        'current_version': e.current_version,
        'expected_version': e.expected_version
    }), 409


# Pipeline CRUD Routes

@pipeline_bp.route('', methods=['GET'])
def list_pipelines():
    """
    List all pipelines with optional filtering.
    
    Query Parameters:
    - active_only: boolean (default: true)
    - tags: comma-separated list of tags
    - limit: integer (default: 50)
    """
    try:
        db = get_db()
        
        # Parse query parameters
        active_only = request.args.get('active_only', 'true').lower() == 'true'
        tags_param = request.args.get('tags', '')
        tags = [t.strip() for t in tags_param.split(',') if t.strip()] if tags_param else None
        
        pipelines = db.list_pipelines(active_only=active_only, tags=tags)
        
        return jsonify({
            'pipelines': [pipeline.to_dict() for pipeline in pipelines],
            'total': len(pipelines)
        })
        
    except Exception as e:
        logger.error(f"Error listing pipelines: {e}")
        return jsonify({'error': 'Internal Server Error', 'message': str(e)}), 500


@pipeline_bp.route('', methods=['POST'])
def create_pipeline():
    """
    Create a new pipeline.
    
    Request Body:
    {
        "name": "Pipeline Name",
        "description": "Description",
        "config_yaml": "YAML configuration",
        "tags": ["tag1", "tag2"],
        "template_name": "Optional template name"
    }
    """
    try:
        data = request.get_json()
        if not data:
            raise BadRequest("Request body must be JSON")
        
        # Validate required fields
        if 'name' not in data:
            raise BadRequest("Pipeline name is required")
        if 'config_yaml' not in data:
            raise BadRequest("Configuration YAML is required")
        
        db = get_db()
        
        pipeline = Pipeline(
            name=data['name'],
            description=data.get('description', ''),
            config_yaml=data['config_yaml'],
            tags=data.get('tags'),
            template_name=data.get('template_name'),
            created_by=data.get('created_by')
        )
        
        created_pipeline = db.create_pipeline(pipeline)
        
        return jsonify({
            'message': 'Pipeline created successfully',
            'pipeline': created_pipeline.to_dict()
        }), 201
        
    except (BadRequest, ValidationError) as e:
        return jsonify({'error': 'Bad Request', 'message': str(e)}), 400
    except Exception as e:
        logger.error(f"Error creating pipeline: {e}")
        return jsonify({'error': 'Internal Server Error', 'message': str(e)}), 500


@pipeline_bp.route('/<int:pipeline_id>', methods=['GET'])
def get_pipeline(pipeline_id: int):
    """Get pipeline by ID."""
    try:
        db = get_db()
        pipeline = db.get_pipeline(pipeline_id)
        return jsonify({'pipeline': pipeline.to_dict()})
        
    except PipelineNotFoundError as e:
        return jsonify({'error': 'Not Found', 'message': str(e)}), 404
    except Exception as e:
        logger.error(f"Error getting pipeline {pipeline_id}: {e}")
        return jsonify({'error': 'Internal Server Error', 'message': str(e)}), 500


@pipeline_bp.route('/<int:pipeline_id>', methods=['PUT'])
def update_pipeline(pipeline_id: int):
    """
    Update pipeline with optimistic locking.
    
    Request Body:
    {
        "name": "Updated name",
        "description": "Updated description",
        "config_yaml": "Updated YAML",
        "tags": ["tag1", "tag2"],
        "is_active": true,
        "expected_version": 2
    }
    """
    try:
        data = request.get_json()
        if not data:
            raise BadRequest("Request body must be JSON")
        
        if 'expected_version' not in data:
            raise BadRequest("expected_version is required for concurrent updates")
        
        db = get_db()
        
        # Get current pipeline
        current_pipeline = db.get_pipeline(pipeline_id)
        
        # Update fields
        pipeline = Pipeline(
            id=pipeline_id,
            name=data.get('name', current_pipeline.name),
            description=data.get('description', current_pipeline.description),
            config_yaml=data.get('config_yaml', current_pipeline.config_yaml),
            tags=data.get('tags', current_pipeline.tags),
            is_active=data.get('is_active', current_pipeline.is_active),
            template_name=data.get('template_name', current_pipeline.template_name)
        )
        
        updated_pipeline = db.update_pipeline(pipeline, data['expected_version'])
        
        return jsonify({
            'message': 'Pipeline updated successfully',
            'pipeline': updated_pipeline.to_dict()
        })
        
    except (BadRequest, ValidationError) as e:
        return jsonify({'error': 'Bad Request', 'message': str(e)}), 400
    except ConcurrencyError:
        raise  # Let error handler deal with it
    except PipelineNotFoundError:
        raise  # Let error handler deal with it
    except Exception as e:
        logger.error(f"Error updating pipeline {pipeline_id}: {e}")
        return jsonify({'error': 'Internal Server Error', 'message': str(e)}), 500


@pipeline_bp.route('/<int:pipeline_id>', methods=['DELETE'])
def delete_pipeline(pipeline_id: int):
    """Delete pipeline and all related executions."""
    try:
        db = get_db()
        deleted = db.delete_pipeline(pipeline_id)
        
        if not deleted:
            return jsonify({'error': 'Not Found', 'message': f'Pipeline {pipeline_id} not found'}), 404
        
        return jsonify({'message': 'Pipeline deleted successfully'}), 200
        
    except Exception as e:
        logger.error(f"Error deleting pipeline {pipeline_id}: {e}")
        return jsonify({'error': 'Internal Server Error', 'message': str(e)}), 500


@pipeline_bp.route('/<int:pipeline_id>/clone', methods=['POST'])
def clone_pipeline(pipeline_id: int):
    """
    Clone an existing pipeline.
    
    Request Body:
    {
        "name": "New pipeline name",
        "created_by": "username"
    }
    """
    try:
        data = request.get_json()
        if not data or 'name' not in data:
            raise BadRequest("New pipeline name is required")
        
        db = get_db()
        cloned_pipeline = db.clone_pipeline(
            source_id=pipeline_id,
            new_name=data['name'],
            created_by=data.get('created_by')
        )
        
        return jsonify({
            'message': 'Pipeline cloned successfully',
            'pipeline': cloned_pipeline.to_dict()
        }), 201
        
    except (BadRequest, ValidationError) as e:
        return jsonify({'error': 'Bad Request', 'message': str(e)}), 400
    except PipelineNotFoundError:
        raise  # Let error handler deal with it
    except Exception as e:
        logger.error(f"Error cloning pipeline {pipeline_id}: {e}")
        return jsonify({'error': 'Internal Server Error', 'message': str(e)}), 500


# Template Routes

@pipeline_bp.route('/templates', methods=['GET'])
def list_templates():
    """
    List pipeline templates.
    
    Query Parameters:
    - category: filter by category
    """
    try:
        db = get_db()
        category = request.args.get('category')
        templates = db.list_templates(category=category)
        
        return jsonify({
            'templates': [template.to_dict() for template in templates],
            'total': len(templates)
        })
        
    except Exception as e:
        logger.error(f"Error listing templates: {e}")
        return jsonify({'error': 'Internal Server Error', 'message': str(e)}), 500


@pipeline_bp.route('/templates/<int:template_id>/create', methods=['POST'])
def create_from_template(template_id: int):
    """
    Create pipeline from template.
    
    Request Body:
    {
        "name": "New Pipeline Name",
        "created_by": "username"
    }
    """
    try:
        data = request.get_json()
        if not data or 'name' not in data:
            raise BadRequest("Pipeline name is required")
        
        db = get_db()
        pipeline = db.create_pipeline_from_template(
            template_id=template_id,
            pipeline_name=data['name'],
            created_by=data.get('created_by')
        )
        
        return jsonify({
            'message': 'Pipeline created from template successfully',
            'pipeline': pipeline.to_dict()
        }), 201
        
    except (BadRequest, ValidationError) as e:
        return jsonify({'error': 'Bad Request', 'message': str(e)}), 400
    except PipelineNotFoundError:
        raise  # Let error handler deal with it
    except Exception as e:
        logger.error(f"Error creating pipeline from template {template_id}: {e}")
        return jsonify({'error': 'Internal Server Error', 'message': str(e)}), 500


# Execution Routes

@pipeline_bp.route('/<int:pipeline_id>/execute', methods=['POST'])
def execute_pipeline(pipeline_id: int):
    """
    Start pipeline execution.
    
    Request Body:
    {
        "worker_count": 1,
        "documents_total": 100,
        "execution_metadata": {}
    }
    """
    try:
        data = request.get_json() or {}
        
        db = get_db()
        execution_tracker = get_execution_tracker()
        
        # Get pipeline to ensure it exists
        pipeline = db.get_pipeline(pipeline_id)
        
        # Start execution using the execution engine
        engine = get_execution_engine()
        execution = engine.execute_pipeline(
            pipeline_id=pipeline_id,
            execution_params={
                'worker_count': data.get('worker_count', 1),
                'documents_total': data.get('documents_total', 0)
            }
        )
        
        return jsonify({
            'message': 'Pipeline execution started',
            'execution': execution.to_dict()
        }), 201
        
    except PipelineNotFoundError:
        raise  # Let error handler deal with it
    except Exception as e:
        logger.error(f"Error starting execution for pipeline {pipeline_id}: {e}")
        return jsonify({'error': 'Internal Server Error', 'message': str(e)}), 500


@pipeline_bp.route('/<int:pipeline_id>/executions', methods=['GET'])
def get_pipeline_executions(pipeline_id: int):
    """
    Get execution history for a pipeline.
    
    Query Parameters:
    - limit: integer (default: 20)
    """
    try:
        limit = int(request.args.get('limit', 20))
        
        execution_tracker = get_execution_tracker()
        executions = execution_tracker.list_executions(pipeline_id=pipeline_id, limit=limit)
        
        return jsonify({
            'executions': [execution.to_dict() for execution in executions],
            'total': len(executions)
        })
        
    except Exception as e:
        logger.error(f"Error getting executions for pipeline {pipeline_id}: {e}")
        return jsonify({'error': 'Internal Server Error', 'message': str(e)}), 500


@pipeline_bp.route('/executions/recent', methods=['GET'])
def get_recent_executions():
    """
    Get recent pipeline executions across all pipelines.

    Query Parameters:
        limit: Number of executions to return (default: 20, max: 100)
    """
    try:
        limit = min(int(request.args.get('limit', 20)), 100)

        execution_tracker = get_execution_tracker()
        executions = execution_tracker.list_executions(limit=limit)

        # Convert to dict and include pipeline names
        db = get_db()
        result_executions = []

        for execution in executions:
            execution_dict = execution.to_dict()

            # Add pipeline name if available
            try:
                pipeline = db.get_pipeline(execution.pipeline_id)
                execution_dict['pipeline_name'] = pipeline.name
            except:
                execution_dict['pipeline_name'] = f"Pipeline {execution.pipeline_id}"

            result_executions.append(execution_dict)

        return jsonify({
            'executions': result_executions,
            'total': len(result_executions)
        })

    except Exception as e:
        logger.error(f"Error getting recent executions: {e}")
        return jsonify({'error': 'Internal Server Error', 'message': str(e)}), 500


@pipeline_bp.route('/executions/<string:run_id>', methods=['GET'])
def get_execution(run_id: str):
    """Get execution details by run ID."""
    try:
        execution_tracker = get_execution_tracker()
        execution = execution_tracker.get_execution(run_id)
        
        if not execution:
            return jsonify({'error': 'Not Found', 'message': f'Execution {run_id} not found'}), 404
        
        return jsonify({'execution': execution.to_dict()})
        
    except Exception as e:
        logger.error(f"Error getting execution {run_id}: {e}")
        return jsonify({'error': 'Internal Server Error', 'message': str(e)}), 500


@pipeline_bp.route('/executions/<string:run_id>/progress', methods=['PUT'])
def update_execution_progress(run_id: str):
    """
    Update execution progress.
    
    Request Body:
    {
        "documents_processed": 50,
        "documents_total": 100,
        "status": "running",
        "errors_count": 2,
        "warnings_count": 5
    }
    """
    try:
        data = request.get_json()
        if not data:
            raise BadRequest("Request body must be JSON")
        
        execution_tracker = get_execution_tracker()
        updated = execution_tracker.update_execution_progress(
            run_id=run_id,
            documents_processed=data.get('documents_processed'),
            documents_total=data.get('documents_total'),
            status=data.get('status'),
            errors_count=data.get('errors_count'),
            warnings_count=data.get('warnings_count')
        )
        
        if not updated:
            return jsonify({'error': 'Not Found', 'message': f'Execution {run_id} not found'}), 404
        
        return jsonify({'message': 'Execution progress updated successfully'})
        
    except BadRequest as e:
        return jsonify({'error': 'Bad Request', 'message': str(e)}), 400
    except Exception as e:
        logger.error(f"Error updating execution progress {run_id}: {e}")
        return jsonify({'error': 'Internal Server Error', 'message': str(e)}), 500


@pipeline_bp.route('/executions/<string:run_id>/status', methods=['GET'])
def get_execution_status(run_id: str):
    """
    Get real-time execution status including progress monitor data.
    """
    try:
        engine = get_execution_engine()
        
        # Get execution status from the engine (includes real-time info)
        status = engine.get_execution_status(run_id)
        if not status:
            return jsonify({'error': 'Not Found', 'message': f'Execution {run_id} not found'}), 404
        
        # Get progress monitor data
        progress_status = engine.progress_monitor.get_current_status(run_id)
        recent_events = engine.progress_monitor.get_recent_events(run_id, limit=10)
        
        return jsonify({
            'execution': status,
            'progress': progress_status,
            'recent_events': recent_events
        })
        
    except Exception as e:
        logger.error(f"Error getting execution status {run_id}: {e}")
        return jsonify({'error': 'Internal Server Error', 'message': str(e)}), 500


@pipeline_bp.route('/executions/<string:run_id>/cancel', methods=['POST'])
def cancel_execution(run_id: str):
    """
    Cancel an active execution.

    Request Body (optional):
    {
        "cleanup": false,  // Whether to cleanup data after cancellation
        "reason": "User requested cancellation"
    }
    """
    try:
        from ..pipeline.pipeline_monitor import PipelineMonitor, JobStatus

        data = request.get_json() or {}
        cleanup_after = data.get('cleanup', False)
        reason = data.get('reason', 'User requested cancellation')

        # Update monitoring status
        monitor = PipelineMonitor()
        monitor.update_job_status(
            run_id,
            status=JobStatus.CANCELLED,
            last_error=reason
        )

        # Cancel via execution engine
        engine = get_execution_engine()
        cancelled = engine.cancel_execution(run_id)

        if not cancelled:
            return jsonify({'error': 'Not Found', 'message': f'Execution {run_id} not found or not active'}), 404

        # If cleanup requested, mark for cleanup
        if cleanup_after:
            monitor.update_job_status(run_id, cleanup_status='pending')

        return jsonify({
            'message': 'Execution cancelled',
            'run_id': run_id,
            'cleanup_pending': cleanup_after
        })

    except Exception as e:
        logger.error(f"Error cancelling execution {run_id}: {e}")
        return jsonify({'error': 'Internal Server Error', 'message': str(e)}), 500


@pipeline_bp.route('/executions/<string:run_id>/cleanup', methods=['POST'])
def cleanup_execution(run_id: str):
    """
    Clean up all data from a specific execution.

    This will delete all documents, elements, and relationships created by this run.

    Request Body:
    {
        "revert_to_previous": false,  // Make previous successful run current
        "delete_files": false,         // Also delete stored files
        "force": false                 // Force cleanup even if job is running
    }

    Returns:
    {
        "message": "Cleanup completed",
        "stats": {
            "documents_deleted": 100,
            "elements_deleted": 500,
            "relationships_deleted": 200
        }
    }
    """
    try:
        from ..pipeline.pipeline_monitor import PipelineMonitor
        from ..storage import get_storage_backend

        data = request.get_json() or {}
        revert_to_previous = data.get('revert_to_previous', False)
        delete_files = data.get('delete_files', False)
        force = data.get('force', False)

        monitor = PipelineMonitor()

        # Get job status
        job = monitor.get_job_status(run_id)
        if not job:
            return jsonify({'error': 'Not Found', 'message': f'Execution {run_id} not found'}), 404

        # Check if job is still running
        if not force and job['status'] in ['running', 'initializing']:
            return jsonify({
                'error': 'Conflict',
                'message': 'Cannot cleanup while job is running. Cancel first or use force=true'
            }), 409

        # Update cleanup status
        monitor.update_job_status(run_id, cleanup_status='in_progress')

        try:
            # Get storage backend
            storage = get_storage_backend()

            # Count items to be deleted (for stats)
            stats = {
                'documents_deleted': 0,
                'elements_deleted': 0,
                'relationships_deleted': 0
            }

            # Delete from storage tables
            # This would need to be implemented in the storage backend
            if hasattr(storage, 'cleanup_run'):
                cleanup_stats = storage.cleanup_run(run_id, delete_files=delete_files)
                stats.update(cleanup_stats)
            else:
                # Fallback: manual deletion
                # You would implement this based on your storage schema
                logger.warning(f"Storage backend doesn't support cleanup_run, using fallback")

            # Update monitoring
            monitor.update_job_status(
                run_id,
                cleanup_status='completed',
                documents_cleaned=stats['documents_deleted'],
                elements_cleaned=stats['elements_deleted']
            )

            # Handle revert to previous
            if revert_to_previous:
                # Find the last successful run for this pipeline
                with monitor._get_connection() as conn:
                    cursor = conn.cursor()
                    cursor.execute("""
                        SELECT run_id FROM pipeline_job_status
                        WHERE pipeline_id = ? AND status = 'completed'
                        AND run_id != ?
                        ORDER BY completed_at DESC
                        LIMIT 1
                    """, (job['pipeline_id'], run_id))

                    row = cursor.fetchone()
                    if row:
                        previous_run_id = row[0]
                        # Update pipeline to use previous run as current
                        # This would need to be added to pipeline config
                        logger.info(f"Would revert pipeline {job['pipeline_id']} to run {previous_run_id}")

            return jsonify({
                'message': 'Cleanup completed',
                'run_id': run_id,
                'stats': stats,
                'reverted_to': previous_run_id if revert_to_previous and 'previous_run_id' in locals() else None
            })

        except Exception as cleanup_error:
            # Update status to failed
            monitor.update_job_status(
                run_id,
                cleanup_status='failed',
                last_error=str(cleanup_error)
            )
            raise

    except Exception as e:
        logger.error(f"Error cleaning up execution {run_id}: {e}")
        return jsonify({'error': 'Internal Server Error', 'message': str(e)}), 500


@pipeline_bp.route('/pipelines/<int:pipeline_id>/active-runs', methods=['GET'])
def get_active_runs(pipeline_id: int):
    """
    Check if a pipeline has any active runs.

    Returns:
    {
        "has_active": true,
        "active_runs": [
            {
                "run_id": "run_123",
                "status": "running",
                "started_at": "2024-01-01T10:00:00Z",
                "progress_percentage": 45.2
            }
        ]
    }
    """
    try:
        from ..pipeline.pipeline_monitor import PipelineMonitor

        monitor = PipelineMonitor()

        # Get active jobs for this pipeline
        active_jobs = [
            job for job in monitor.get_active_jobs(pipeline_id)
            if job['status'] in ['pending', 'initializing', 'running']
        ]

        return jsonify({
            'has_active': len(active_jobs) > 0,
            'active_runs': [
                {
                    'run_id': job['run_id'],
                    'status': job['status'],
                    'started_at': job['started_at'],
                    'progress_percentage': job.get('progress_percentage', 0)
                }
                for job in active_jobs
            ]
        })

    except Exception as e:
        logger.error(f"Error checking active runs for pipeline {pipeline_id}: {e}")
        return jsonify({'error': 'Internal Server Error', 'message': str(e)}), 500


@pipeline_bp.route('/executions/<string:run_id>/logs', methods=['GET'])
def get_execution_logs(run_id: str):
    """
    Get execution logs with optional pagination.
    
    Query Parameters:
    - start_index: Starting index for logs (default: 0)
    
    Response:
    {
        "run_id": "run_20241124_123456_abc123",
        "logs": [
            {
                "timestamp": "2024-11-24T12:34:56",
                "level": "INFO",
                "message": "Starting pipeline execution",
                "module": "go_doc_go.pipeline"
            }
        ],
        "total_count": 100,
        "start_index": 0,
        "has_more": true
    }
    """
    try:
        start_index = request.args.get('start_index', 0, type=int)
        
        engine = get_execution_engine()
        logs_data = engine.get_execution_logs(run_id, start_index=start_index)
        
        return jsonify(logs_data)
        
    except Exception as e:
        logger.error(f"Error getting execution logs {run_id}: {e}")
        return jsonify({'error': 'Internal Server Error', 'message': str(e)}), 500


@pipeline_bp.route('/executions/active', methods=['GET'])
def list_active_executions():
    """
    List all currently active executions across all pipelines.
    """
    try:
        engine = get_execution_engine()
        active_executions = engine.list_active_executions()
        
        # Also get progress monitor data
        monitor_data = engine.progress_monitor.get_all_active_executions()
        
        return jsonify({
            'active_executions': active_executions,
            'progress_data': monitor_data,
            'total': len(active_executions)
        })
        
    except Exception as e:
        logger.error(f"Error listing active executions: {e}")
        return jsonify({'error': 'Internal Server Error', 'message': str(e)}), 500

# Progress Monitoring Routes

@pipeline_bp.route('/progress/websocket-handler', methods=['GET'])
def get_websocket_handler():
    """
    Get WebSocket handler function for progress monitoring.
    This endpoint returns information about how to connect to WebSocket for real-time updates.
    """
    try:
        return jsonify({
            'message': 'WebSocket progress monitoring available',
            'endpoints': {
                'all_progress': '/ws/progress',
                'execution_progress': '/ws/progress/<run_id>',
                'pipeline_progress': '/ws/pipelines/<pipeline_id>/progress'
            },
            'usage': {
                'description': 'Connect to WebSocket endpoints for real-time progress updates',
                'message_types': ['progress_event', 'current_status', 'historical_event', 'ping']
            }
        })
        
    except Exception as e:
        logger.error(f"Error getting WebSocket info: {e}")
        return jsonify({'error': 'Internal Server Error', 'message': str(e)}), 500


@pipeline_bp.route('/progress/events/<string:run_id>', methods=['GET'])
def get_progress_events(run_id: str):
    """
    Get progress events for an execution (REST fallback for WebSocket).
    
    Query Parameters:
    - limit: integer (default: 50)
    - since: timestamp to get events since
    """
    try:
        limit = int(request.args.get('limit', 50))
        
        engine = get_execution_engine()
        events = engine.progress_monitor.get_recent_events(run_id, limit)
        
        return jsonify({
            'run_id': run_id,
            'events': events,
            'total': len(events)
        })
        
    except Exception as e:
        logger.error(f"Error getting progress events for {run_id}: {e}")
        return jsonify({'error': 'Internal Server Error', 'message': str(e)}), 500


# Dashboard Routes

@pipeline_bp.route('/dashboard', methods=['GET'])
def get_pipeline_dashboard():
    """
    Get comprehensive dashboard data for all pipelines and executions.
    """
    try:
        db = get_db()
        execution_tracker = get_execution_tracker()
        engine = get_execution_engine()
        
        # Get pipeline summary
        all_pipelines = db.list_pipelines(active_only=False)
        active_pipelines = db.list_pipelines(active_only=True)
        
        # Get execution summary
        recent_executions = execution_tracker.list_executions(limit=50)
        active_executions = engine.list_active_executions()
        
        # Get progress monitor data
        progress_data = engine.progress_monitor.get_all_active_executions()
        
        # Calculate statistics
        execution_stats = {
            'total_executions': len(recent_executions),
            'active_executions': len(active_executions),
            'completed_executions': len([e for e in recent_executions if e.status == 'completed']),
            'failed_executions': len([e for e in recent_executions if e.status == 'failed']),
            'cancelled_executions': len([e for e in recent_executions if e.status == 'cancelled'])
        }
        
        pipeline_stats = {
            'total_pipelines': len(all_pipelines),
            'active_pipelines': len(active_pipelines),
            'inactive_pipelines': len(all_pipelines) - len(active_pipelines),
            'templates_count': len(db.list_templates())
        }
        
        # Get recent activity (last 10 executions)
        recent_activity = []
        for execution in recent_executions[:10]:
            pipeline = next((p for p in all_pipelines if p.id == execution.pipeline_id), None)
            activity_item = {
                'type': 'execution',
                'timestamp': execution.created_at,
                'pipeline_name': pipeline.name if pipeline else f"Pipeline {execution.pipeline_id}",
                'pipeline_id': execution.pipeline_id,
                'run_id': execution.run_id,
                'status': execution.status,
                'documents_processed': execution.documents_processed,
                'documents_total': execution.documents_total
            }
            recent_activity.append(activity_item)
        
        return jsonify({
            'pipeline_stats': pipeline_stats,
            'execution_stats': execution_stats,
            'active_executions': [exec.to_dict() for exec in active_executions],
            'recent_activity': recent_activity,
            'progress_data': progress_data,
            'system_info': {
                'total_active_monitors': len(engine.progress_monitor._listeners),
                'database_path': os.environ.get('PIPELINE_CONFIG_DB', 'pipeline_config.db')
            }
        })
        
    except Exception as e:
        logger.error(f"Error getting dashboard data: {e}")
        return jsonify({'error': 'Internal Server Error', 'message': str(e)}), 500


@pipeline_bp.route('/dashboard/summary', methods=['GET'])
def get_dashboard_summary():
    """
    Get lightweight dashboard summary for frequent polling.
    """
    try:
        db = get_db()
        execution_tracker = get_execution_tracker()
        engine = get_execution_engine()
        
        # Get active executions count
        active_executions = engine.list_active_executions()
        
        # Get pipeline counts
        active_pipelines = db.list_pipelines(active_only=True)
        
        # Get recent execution stats (last 24 hours)
        recent_executions = execution_tracker.list_executions(limit=100)
        now = datetime.now()
        day_ago = now - timedelta(days=1)
        
        recent_24h = [
            e for e in recent_executions 
            if datetime.fromisoformat(e.created_at) > day_ago
        ]
        
        return jsonify({
            'active_executions_count': len(active_executions),
            'active_pipelines_count': len(active_pipelines),
            'executions_last_24h': len(recent_24h),
            'completed_last_24h': len([e for e in recent_24h if e.status == 'completed']),
            'failed_last_24h': len([e for e in recent_24h if e.status == 'failed']),
            'system_healthy': len([e for e in recent_24h if e.status == 'failed']) < 5,  # Simple health check
            'last_updated': datetime.now().isoformat()
        })
        
    except Exception as e:
        logger.error(f"Error getting dashboard summary: {e}")
        return jsonify({'error': 'Internal Server Error', 'message': str(e)}), 500


# Unified Monitoring Routes

@pipeline_bp.route('/monitor', methods=['GET'])
def monitor_pipelines():
    """
    Unified monitoring endpoint for all pipeline jobs.

    Provides comprehensive real-time monitoring data for all running and recent pipeline executions.

    Query Parameters:
    - status: Filter by status (running, completed, failed, all) - default: running
    - include_workers: Include detailed worker status (true/false) - default: false
    - include_history: Include historical performance data (true/false) - default: false
    - pipeline_id: Filter by specific pipeline ID
    - since: Return jobs updated since timestamp (ISO format)
    - limit: Maximum number of jobs to return (default: 50, max: 200)

    Returns:
    {
        "jobs": [
            {
                "run_id": "run_20241201_123456_abc",
                "pipeline_id": 1,
                "pipeline_name": "Financial Analysis",
                "status": "running",
                "phase": "embedding",
                "health": "healthy",
                "progress": {
                    "percentage": 45.2,
                    "documents_total": 1000,
                    "documents_claimed": 500,
                    "documents_processed": 452,
                    "documents_failed": 3,
                    "documents_remaining": 545
                },
                "workers": {
                    "total": 10,
                    "active": 8,
                    "failed": 1,
                    "idle": 1,
                    "details": []  // Included if include_workers=true
                },
                "performance": {
                    "avg_processing_time_ms": 250,
                    "estimated_completion": "2024-12-01T14:30:00Z",
                    "throughput_per_minute": 28.5
                },
                "timing": {
                    "started_at": "2024-12-01T10:00:00Z",
                    "running_time_minutes": 210,
                    "last_heartbeat": "2024-12-01T13:30:00Z"
                },
                "errors": {
                    "count": 3,
                    "last_error": "Failed to process document X"
                }
            }
        ],
        "summary": {
            "total_jobs": 15,
            "running_jobs": 3,
            "completed_jobs": 10,
            "failed_jobs": 2,
            "total_workers_active": 25,
            "total_documents_processing": 3456,
            "system_health": "good",
            "oldest_running_job": "2024-12-01T08:00:00Z"
        },
        "performance_history": [],  // Included if include_history=true
        "timestamp": "2024-12-01T13:30:15Z"
    }
    """
    try:
        from ..pipeline.pipeline_monitor import PipelineMonitor, JobStatus

        # Initialize monitor
        monitor = PipelineMonitor()

        # Parse query parameters
        status_filter = request.args.get('status', 'running')
        include_workers = request.args.get('include_workers', 'false').lower() == 'true'
        include_history = request.args.get('include_history', 'false').lower() == 'true'
        pipeline_id = request.args.get('pipeline_id', type=int)
        since = request.args.get('since')
        limit = min(int(request.args.get('limit', 50)), 200)

        # Get jobs based on status filter
        if status_filter == 'all':
            # Get all recent jobs
            with monitor._get_connection() as conn:
                cursor = conn.cursor()
                query = """
                    SELECT * FROM pipeline_monitoring_dashboard
                    WHERE 1=1
                """
                params = []

                if pipeline_id:
                    query += " AND pipeline_id = ?"
                    params.append(pipeline_id)

                if since:
                    query += " AND updated_at >= ?"
                    params.append(since)

                query += " ORDER BY started_at DESC LIMIT ?"
                params.append(limit)

                cursor.execute(query, params)
                jobs = [dict(row) for row in cursor.fetchall()]
        else:
            # First, detect and fix any stale jobs automatically
            try:
                stale_fixed = monitor.detect_and_fix_stale_jobs(stale_threshold_minutes=15)
                if stale_fixed > 0:
                    logger.info(f"Automatically fixed {stale_fixed} stale jobs during monitoring query")
            except Exception as e:
                logger.error(f"Failed to auto-detect stale jobs: {e}")

            # Get active jobs
            jobs = monitor.get_active_jobs(pipeline_id)

            # Filter by specific status if needed
            if status_filter != 'running':
                jobs = [j for j in jobs if j['status'] == status_filter]

        # Format job data for response
        formatted_jobs = []
        total_active_workers = 0
        total_documents_processing = 0

        for job in jobs[:limit]:
            job_data = {
                'run_id': job['run_id'],
                'pipeline_id': job['pipeline_id'],
                'pipeline_name': job.get('pipeline_name', f"Pipeline {job['pipeline_id']}"),
                'status': job['status'],
                'phase': job.get('phase'),
                'health': job.get('calculated_health', job.get('health_status', 'unknown')),
                'progress': {
                    'percentage': round(job.get('calculated_progress_pct', 0), 2),
                    'documents_total': job.get('documents_total', 0),
                    'documents_claimed': job.get('documents_claimed', 0),
                    'documents_processed': job.get('documents_processed', 0),
                    'documents_failed': job.get('documents_failed', 0),
                    'documents_remaining': job.get('documents_remaining', 0)
                },
                'workers': {
                    'total': job.get('total_workers', 0),
                    'active': job.get('active_workers', 0),
                    'failed': job.get('failed_workers', 0),
                    'idle': job.get('idle_workers', 0)
                },
                'performance': {
                    'avg_processing_time_ms': job.get('avg_processing_time_ms', 0),
                    'estimated_completion': job.get('calculated_eta'),
                },
                'timing': {
                    'started_at': job.get('started_at'),
                    'running_time_minutes': job.get('duration_minutes', 0),
                    'last_heartbeat': job.get('last_heartbeat')
                },
                'errors': {
                    'count': job.get('error_count', 0),
                    'last_error': job.get('last_error')
                }
            }

            # Calculate throughput
            if job.get('duration_minutes', 0) > 0:
                job_data['performance']['throughput_per_minute'] = round(
                    job.get('documents_processed', 0) / job['duration_minutes'], 2
                )

            # Include worker details if requested
            if include_workers:
                job_data['workers']['details'] = monitor.get_worker_status(job['run_id'])

            formatted_jobs.append(job_data)

            # Accumulate summary stats
            if job['status'] == 'running':
                total_active_workers += job.get('active_workers', 0)
                total_documents_processing += job.get('documents_remaining', 0)

        # Get monitoring summary
        summary_data = monitor.get_monitoring_summary()

        # Build response summary
        summary = {
            'total_jobs': len(jobs),
            'running_jobs': len([j for j in jobs if j['status'] == 'running']),
            'completed_jobs': len([j for j in jobs if j['status'] == 'completed']),
            'failed_jobs': len([j for j in jobs if j['status'] == 'failed']),
            'total_workers_active': total_active_workers,
            'total_documents_processing': total_documents_processing,
            'system_health': 'good' if summary_data.get('total_active_jobs', 0) < 10 else 'busy',
            'oldest_running_job': min(
                (j['started_at'] for j in jobs if j['status'] == 'running'),
                default=None
            )
        }

        response = {
            'jobs': formatted_jobs,
            'summary': summary,
            'timestamp': datetime.now().isoformat()
        }

        # Include historical performance if requested
        if include_history:
            response['performance_history'] = summary_data.get('recent_history', [])
            response['phase_performance'] = summary_data.get('phase_performance', [])

        return jsonify(response)

    except Exception as e:
        logger.error(f"Error in monitoring endpoint: {e}", exc_info=True)
        return jsonify({
            'error': 'Failed to get monitoring data',
            'message': str(e)
        }), 500


@pipeline_bp.route('/monitor/<string:run_id>', methods=['GET'])
def monitor_job(run_id: str):
    """
    Get detailed monitoring data for a specific job.

    Parameters:
        run_id: The execution run ID

    Query Parameters:
        include_workers: Include detailed worker status (true/false)
        include_events: Include recent processing events (true/false)
        events_limit: Number of events to include (default: 100)

    Returns detailed monitoring data for the specified job.
    """
    try:
        from ..pipeline.pipeline_monitor import PipelineMonitor

        monitor = PipelineMonitor()

        # Parse query parameters
        include_workers = request.args.get('include_workers', 'true').lower() == 'true'
        include_events = request.args.get('include_events', 'false').lower() == 'true'
        events_limit = min(int(request.args.get('events_limit', 100)), 1000)

        # Get job status
        job = monitor.get_job_status(run_id)
        if not job:
            return jsonify({'error': 'Job not found', 'run_id': run_id}), 404

        # Get job health
        health_status, health_reason = monitor.get_job_health(run_id)

        response = {
            'job': dict(job),
            'health': {
                'status': health_status.value,
                'reason': health_reason
            }
        }

        # Include worker details
        if include_workers:
            response['workers'] = monitor.get_worker_status(run_id)

        # Include processing events
        if include_events:
            with monitor._get_connection() as conn:
                cursor = conn.cursor()
                cursor.execute("""
                    SELECT * FROM pipeline_processing_events
                    WHERE run_id = ?
                    ORDER BY created_at DESC
                    LIMIT ?
                """, (run_id, events_limit))
                response['events'] = [dict(row) for row in cursor.fetchall()]

        # Get phase checkpoints
        with monitor._get_connection() as conn:
            cursor = conn.cursor()
            cursor.execute("""
                SELECT * FROM pipeline_phase_checkpoints
                WHERE run_id = ?
                ORDER BY
                    CASE phase
                        WHEN 'setup' THEN 1
                        WHEN 'ingestion' THEN 2
                        WHEN 'parsing' THEN 3
                        WHEN 'extraction' THEN 4
                        WHEN 'embedding' THEN 5
                        WHEN 'storage' THEN 6
                        WHEN 'cleanup' THEN 7
                        ELSE 8
                    END
            """, (run_id,))
            response['phases'] = [dict(row) for row in cursor.fetchall()]

        return jsonify(response)

    except Exception as e:
        logger.error(f"Error getting job monitoring data: {e}", exc_info=True)
        return jsonify({
            'error': 'Failed to get job monitoring data',
            'message': str(e)
        }), 500


@pipeline_bp.route('/monitor/<string:run_id>/heartbeat', methods=['POST'])
def send_job_heartbeat(run_id: str):
    """
    Send a heartbeat for a job or worker.

    Request Body (optional):
    {
        "worker_id": "worker_123",  // If sending worker heartbeat
        "stats": {  // Optional statistics update
            "documents_processed": 100,
            "documents_failed": 2,
            "memory_usage_mb": 512,
            "cpu_usage_percent": 45.5
        }
    }
    """
    try:
        from ..pipeline.pipeline_monitor import PipelineMonitor

        monitor = PipelineMonitor()
        data = request.get_json() or {}

        worker_id = data.get('worker_id')
        stats = data.get('stats')

        if worker_id:
            # Worker heartbeat
            success = monitor.worker_heartbeat(worker_id, stats)
        else:
            # Job heartbeat
            success = monitor.job_heartbeat(run_id)

            # Update job stats if provided
            if stats:
                monitor.update_job_status(
                    run_id,
                    documents_processed=stats.get('documents_processed'),
                    documents_failed=stats.get('documents_failed'),
                    active_workers=stats.get('active_workers')
                )

        if success:
            return jsonify({'message': 'Heartbeat received'}), 200
        else:
            return jsonify({'error': 'Failed to record heartbeat'}), 500

    except Exception as e:
        logger.error(f"Error processing heartbeat: {e}")
        return jsonify({'error': 'Failed to process heartbeat', 'message': str(e)}), 500


@pipeline_bp.route('/monitor/cleanup-stale', methods=['POST'])
def cleanup_stale_jobs():
    """
    Manually trigger cleanup of stale jobs.

    Request Body (optional):
    {
        "stale_threshold_minutes": 30  // Default: 30 minutes
    }

    Returns:
    {
        "fixed_count": 2,
        "message": "Fixed 2 stale jobs"
    }
    """
    try:
        from ..pipeline.pipeline_monitor import PipelineMonitor

        # Parse request data
        data = request.get_json() or {}
        stale_threshold_minutes = data.get('stale_threshold_minutes', 30)

        # Initialize monitor
        monitor = PipelineMonitor()

        # Detect and fix stale jobs
        fixed_count = monitor.detect_and_fix_stale_jobs(stale_threshold_minutes)

        message = f"Fixed {fixed_count} stale job{'s' if fixed_count != 1 else ''}"
        return jsonify({
            'fixed_count': fixed_count,
            'message': message
        }), 200

    except Exception as e:
        logger.error(f"Error cleaning up stale jobs: {e}")
        return jsonify({
            'error': 'Failed to cleanup stale jobs',
            'message': str(e)
        }), 500


@pipeline_bp.route('/monitor/migrate-executions', methods=['POST'])
def migrate_executions_to_monitoring():
    """
    Migrate existing pipeline executions to the monitoring system.

    Returns:
    {
        "migrated_count": 71,
        "message": "Migrated 71 executions to monitoring system"
    }
    """
    try:
        from ..pipeline.pipeline_monitor import PipelineMonitor

        # Initialize monitor
        monitor = PipelineMonitor()

        # Migrate existing executions
        migrated_count = monitor.migrate_executions_to_monitoring()

        message = f"Migrated {migrated_count} execution{'s' if migrated_count != 1 else ''} to monitoring system"
        return jsonify({
            'migrated_count': migrated_count,
            'message': message
        }), 200

    except Exception as e:
        logger.error(f"Error migrating executions to monitoring: {e}")
        return jsonify({
            'error': 'Failed to migrate executions',
            'message': str(e)
        }), 500


# Query Routes

@pipeline_bp.route('/<int:pipeline_id>/query', methods=['POST'])
def query_pipeline(pipeline_id: int):
    """
    Query data processed by a specific pipeline.

    This endpoint allows searching through documents that have been processed
    by the specified pipeline, but only if the pipeline has been successfully executed.

    Request Body:
    {
        "query": "search text",
        "run_id": "optional_run_id_for_time_travel",  // Optional: query specific execution
        "limit": 10,
        "offset": 0,
        "similarity_threshold": 0.7,
        "filters": {
            "date_range": {
                "operator": "relative_days",
                "relative_value": 30
            },
            "element_types": ["paragraph", "header"],
            "document_types": ["pdf", "docx"],
            "metadata": {
                "key": "value"
            }
        },
        "include_content": false,
        "include_metadata": true
    }

    Returns:
    {
        "query": "search text",
        "results": [...],
        "total_results": 42,
        "execution_time_ms": 150,
        "pipeline_id": 123,
        "pipeline_name": "My Pipeline"
    }
    """
    try:
        # Validate pipeline exists
        db = get_db()
        try:
            pipeline = db.get_pipeline(pipeline_id)
        except PipelineNotFoundError:
            return jsonify({'error': 'Pipeline not found'}), 404

        # Parse request data first to check for run_id parameter
        data = request.get_json()
        if not data:
            return jsonify({'error': 'Request body is required'}), 400

        # Check if client specified a run_id for time travel
        requested_run_id = data.get('run_id')

        if requested_run_id:
            # Use the specified run_id
            target_run_id = requested_run_id
            logger.info(f"Using client-specified run_id: {target_run_id}")
        else:
            # Get the latest run_id from the analytics storage
            try:
                from ..storage_adapters.factory import StorageFactory
                from ..config import Config
                config = Config()
                # Get analytics storage configuration from config
                storage_config = config.config.get('storage', {})
                analytics_config = storage_config.get('analytics', {'type': 'parquet', 'base_path': './data-lake'})
                analytics_storage = StorageFactory.create_analytics_storage(analytics_config)

                # Get the most recent run_id by scanning the data lake chronologically
                import os
                from pathlib import Path

                # Scan for run_ids in elements data, organized by date
                elements_path = Path(analytics_storage.base_path) / 'elements'
                latest_run_id = None
                latest_date = (0, 0, 0)  # (year, month, day)

                if elements_path.exists():
                    # Find all date directories
                    for year_dir in elements_path.glob('year=*'):
                        year = int(year_dir.name.split('=')[1])
                        for month_dir in year_dir.glob('month=*'):
                            month = int(month_dir.name.split('=')[1])
                            for day_dir in month_dir.glob('day=*'):
                                day = int(day_dir.name.split('=')[1])
                                current_date = (year, month, day)

                                # Check if this date is more recent
                                if current_date > latest_date:
                                    # Find run_ids in this date
                                    for run_dir in day_dir.glob('run_id=*'):
                                        latest_date = current_date
                                        latest_run_id = run_dir.name.split('=')[1]

                if not latest_run_id:
                    return jsonify({
                        'error': 'No data available',
                        'message': 'No processed data found. Run the pipeline to generate data.'
                    }), 400

                target_run_id = latest_run_id
                logger.info(f"Using latest run_id from date {latest_date[0]}-{latest_date[1]:02d}-{latest_date[2]:02d}: {target_run_id}")

            except Exception as e:
                import traceback
                logger.error(f"Failed to get latest run_id from analytics storage: {e}")
                logger.error(f"Full traceback: {traceback.format_exc()}")
                return jsonify({
                    'error': 'Failed to determine latest data',
                    'message': 'Could not identify latest processed data',
                    'details': str(e)
                }), 500

        query_text = data.get('query', '').strip()
        if not query_text:
            return jsonify({'error': 'Query text is required'}), 400

        # Extract search parameters
        limit = min(data.get('limit', 10), 100)  # Cap at 100 results
        offset = max(data.get('offset', 0), 0)
        similarity_threshold = max(0.0, min(data.get('similarity_threshold', 0.7), 1.0))
        include_content = data.get('include_content', False)
        include_metadata = data.get('include_metadata', True)

        # Build filters from request
        filters = data.get('filters', {})
        search_filters = {}

        # Filter by run_id to only search the latest successful execution
        search_filters['_run_id'] = target_run_id
        logger.info(f"Query filtering by run_id: {target_run_id}")

        # Extract element type filters
        if 'element_types' in filters:
            search_filters['element_type'] = filters['element_types']

        # Extract document type filters
        if 'document_types' in filters:
            search_filters['doc_type'] = filters['document_types']

        # Extract date filters
        if 'date_range' in filters:
            date_filter = filters['date_range']
            operator = date_filter.get('operator')

            if operator == 'relative_days':
                from datetime import datetime, timedelta
                days = date_filter.get('relative_value', 30)
                date_after = (datetime.now() - timedelta(days=days)).isoformat()
                search_filters['date_after'] = date_after
            elif operator == 'within':
                if 'start_date' in date_filter:
                    search_filters['date_after'] = date_filter['start_date']
                if 'end_date' in date_filter:
                    search_filters['date_before'] = date_filter['end_date']
            elif operator == 'after':
                search_filters['date_after'] = date_filter.get('date')
            elif operator == 'before':
                search_filters['date_before'] = date_filter.get('date')

        # Extract custom metadata filters
        if 'metadata' in filters:
            search_filters['metadata'] = filters['metadata']

        # Perform the search using the SearchEngine
        from ..search_module import SearchEngine, SearchRequest

        # Get the pipeline's configuration to determine which analytics backend to use
        pipeline_config = {}
        if pipeline.config_yaml:
            import yaml
            try:
                pipeline_config = yaml.safe_load(pipeline.config_yaml)
            except Exception as e:
                logger.warning(f"Could not parse pipeline config: {e}")

        # Determine search service from pipeline config (required - no defaults)
        # Analytics backend is stored under storage.analytics.type in the pipeline config
        if 'storage' not in pipeline_config:
            return jsonify({
                'error': 'Pipeline configuration error',
                'message': 'Pipeline has no storage configuration. Please configure the pipeline storage settings.'
            }), 400

        if 'analytics' not in pipeline_config['storage']:
            return jsonify({
                'error': 'Pipeline configuration error',
                'message': 'Pipeline has no analytics backend configured. Please configure the analytics storage.'
            }), 400

        analytics_type = pipeline_config['storage']['analytics'].get('type')
        if not analytics_type:
            return jsonify({
                'error': 'Pipeline configuration error',
                'message': 'Analytics backend type is not specified in pipeline configuration.'
            }), 400

        # Map the storage type to the search service name
        # parquet -> parquet_duckdb is the correct mapping
        if analytics_type == 'parquet':
            search_service = 'parquet_duckdb'
        else:
            search_service = analytics_type

        logger.info(f"Using search service: {search_service} for pipeline {pipeline_id}")

        # Create search request
        search_request = SearchRequest(
            search_service=search_service,
            similarity_query=query_text,
            limit=limit,
            offset=offset,
            filters=search_filters,
            similarity_threshold=similarity_threshold,
            include_content=include_content,
            include_metadata=include_metadata
        )

        # Execute search
        import time
        start_time = time.time()

        try:
            search_engine = SearchEngine()
            search_response = search_engine.search(search_request)
            execution_time_ms = int((time.time() - start_time) * 1000)

            # Format response
            results = []
            for hit in search_response.hits:
                result = {
                    'element_id': hit.element_id,
                    'doc_id': hit.doc_id,
                    'score': hit.score,
                    'content_preview': hit.content_preview,
                    'element_type': hit.element_type
                }

                if include_metadata and hit.metadata:
                    result['metadata'] = hit.metadata

                if include_content and hit.content:
                    result['content'] = hit.content

                results.append(result)

            return jsonify({
                'query': query_text,
                'results': results,
                'total_results': search_response.total_hits,
                'execution_time_ms': execution_time_ms,
                'pipeline_id': pipeline_id,
                'pipeline_name': pipeline.name,
                'search_service': search_service,
                'filters_applied': search_response.filters_applied
            })

        except Exception as search_error:
            logger.error(f"Search execution failed: {search_error}")
            return jsonify({
                'error': 'Search execution failed',
                'message': str(search_error)
            }), 500

    except Exception as e:
        logger.error(f"Error querying pipeline {pipeline_id}: {e}")
        return jsonify({'error': 'Internal Server Error', 'message': str(e)}), 500


# Import/Export Routes

@pipeline_bp.route('/<int:pipeline_id>/export', methods=['GET'])
def export_pipeline(pipeline_id: int):
    """Export pipeline configuration as YAML file."""
    try:
        db = get_db()
        pipeline = db.get_pipeline(pipeline_id)
        
        # Create temporary file with YAML content
        with tempfile.NamedTemporaryFile(mode='w', suffix='.yaml', delete=False) as f:
            # Add metadata header
            export_data = {
                'pipeline_metadata': {
                    'name': pipeline.name,
                    'description': pipeline.description,
                    'version': pipeline.version,
                    'tags': pipeline.tags,
                    'exported_at': pipeline.updated_at.isoformat() if pipeline.updated_at else None
                },
                'configuration': yaml.safe_load(pipeline.config_yaml)
            }
            
            yaml.dump(export_data, f, default_flow_style=False)
            temp_path = f.name
        
        # Send file
        filename = f"{pipeline.name.replace(' ', '_')}_v{pipeline.version}.yaml"
        return send_file(
            temp_path,
            as_attachment=True,
            download_name=filename,
            mimetype='application/x-yaml'
        )
        
    except PipelineNotFoundError:
        raise  # Let error handler deal with it
    except Exception as e:
        logger.error(f"Error exporting pipeline {pipeline_id}: {e}")
        return jsonify({'error': 'Internal Server Error', 'message': str(e)}), 500


@pipeline_bp.route('/import', methods=['POST'])
def import_pipeline():
    """
    Import pipeline from YAML file.
    
    Form Data:
    - file: YAML file to import
    - name: Optional override for pipeline name
    - created_by: Creator username
    """
    try:
        if 'file' not in request.files:
            raise BadRequest("No file provided")
        
        file = request.files['file']
        if file.filename == '':
            raise BadRequest("No file selected")
        
        # Read and parse YAML
        content = file.read().decode('utf-8')
        try:
            import_data = yaml.safe_load(content)
        except yaml.YAMLError as e:
            raise BadRequest(f"Invalid YAML file: {e}")
        
        # Extract configuration and metadata
        if 'configuration' in import_data and 'pipeline_metadata' in import_data:
            # New export format with metadata
            config_yaml = yaml.dump(import_data['configuration'], default_flow_style=False)
            metadata = import_data['pipeline_metadata']
            pipeline_name = request.form.get('name', metadata.get('name', 'Imported Pipeline'))
            description = metadata.get('description', '')
            tags = metadata.get('tags', [])
        else:
            # Direct configuration file
            config_yaml = content
            pipeline_name = request.form.get('name', 'Imported Pipeline')
            description = f"Imported from {file.filename}"
            tags = []
        
        db = get_db()
        
        pipeline = Pipeline(
            name=pipeline_name,
            description=description,
            config_yaml=config_yaml,
            tags=tags,
            created_by=request.form.get('created_by')
        )
        
        created_pipeline = db.create_pipeline(pipeline)
        
        return jsonify({
            'message': 'Pipeline imported successfully',
            'pipeline': created_pipeline.to_dict()
        }), 201
        
    except (BadRequest, ValidationError) as e:
        return jsonify({'error': 'Bad Request', 'message': str(e)}), 400
    except Exception as e:
        logger.error(f"Error importing pipeline: {e}")
        return jsonify({'error': 'Internal Server Error', 'message': str(e)}), 500


# Validation Routes

@pipeline_bp.route('/validate', methods=['POST'])
def validate_configuration():
    """
    Validate pipeline configuration without saving.
    
    Request Body:
    {
        "config_yaml": "YAML configuration to validate"
    }
    """
    try:
        data = request.get_json()
        if not data or 'config_yaml' not in data:
            raise BadRequest("Configuration YAML is required")
        
        db = get_db()
        # This will raise ValidationError if invalid
        parsed_config = db._validate_pipeline_config(data['config_yaml'])
        
        return jsonify({
            'valid': True,
            'message': 'Configuration is valid',
            'parsed_config': parsed_config
        })
        
    except (BadRequest, ValidationError) as e:
        return jsonify({
            'valid': False,
            'error': str(e)
        }), 400
    except Exception as e:
        logger.error(f"Error validating configuration: {e}")
        return jsonify({'error': 'Internal Server Error', 'message': str(e)}), 500


@pipeline_bp.route('/<int:pipeline_id>/elements/<element_id>/content', methods=['GET'])
def get_element_content(pipeline_id: int, element_id: str):
    """
    Get the full content for a specific element.

    This endpoint retrieves the complete content text for a single element
    that was processed by the specified pipeline using a direct lookup.
    """
    try:
        # Get database instance
        db = get_db()

        # Get pipeline configuration
        pipeline = db.get_pipeline(pipeline_id)
        if not pipeline:
            raise NotFound(f"Pipeline {pipeline_id} not found")

        # Parse pipeline config
        config_dict = yaml.safe_load(pipeline.config_yaml)

        # Get the analytics backend through the factory
        from ..storage_adapters.factory import StorageFactory

        # Extract analytics configuration
        analytics_config = config_dict.get('storage', {}).get('analytics', {})
        if not analytics_config:
            return jsonify({'error': 'No analytics configuration found'}), 400

        # Create the analytics backend
        analytics_backend = StorageFactory.create_analytics_storage(analytics_config)

        # Use the backend's get_element method
        element_data = analytics_backend.get_element_by_id(element_id)

        if not element_data:
            raise NotFound(f"Element {element_id} not found")

        # Determine what type of content we have
        content_type = None
        content_value = None

        # Priority 1: Check for stored full content
        if element_data.get('content'):
            content_type = 'full_content'
            content_value = element_data['content']
        # Priority 2: Use embedding text if available
        elif element_data.get('embedding_text'):
            content_type = 'embedding_text'
            content_value = element_data['embedding_text']
        # Priority 3: Try to resolve from content_location
        elif element_data.get('content_location'):
            try:
                from ..adapter.factory import ContentResolverFactory
                # Create content resolver
                resolver_config = config_dict.get('content_sources', {})
                resolver = ContentResolverFactory.create_resolver(resolver_config)

                # Resolve content from source
                content_value = resolver.resolve_content(element_data['content_location'], text=True)
                content_type = 'resolved_from_source'
            except Exception as e:
                logger.warning(f"Failed to resolve content from source: {str(e)}")
                # Fall back to content preview
                content_type = 'preview_only'
                content_value = element_data.get('content_preview', '')
        else:
            # Last resort: use content preview
            content_type = 'preview_only'
            content_value = element_data.get('content_preview', '')

        return jsonify({
            'element_id': element_id,
            'content': content_value,
            'content_type': content_type,  # Indicates what type of content is being returned
            'element_type': element_data.get('element_type'),
            'doc_id': element_data.get('doc_id')
        }), 200

    except NotFound as e:
        return jsonify({'error': str(e)}), 404
    except Exception as e:
        logger.error(f"Error getting element content for pipeline {pipeline_id}, element {element_id}: {str(e)}", exc_info=True)
        return jsonify({
            'error': 'Failed to get element content',
            'details': str(e),
            'element_id': element_id,
            'pipeline_id': pipeline_id
        }), 500