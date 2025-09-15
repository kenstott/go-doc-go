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
    """
    try:
        engine = get_execution_engine()
        cancelled = engine.cancel_execution(run_id)
        
        if not cancelled:
            return jsonify({'error': 'Not Found', 'message': f'Execution {run_id} not found or not active'}), 404
        
        return jsonify({'message': 'Execution cancellation requested'})
        
    except Exception as e:
        logger.error(f"Error cancelling execution {run_id}: {e}")
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