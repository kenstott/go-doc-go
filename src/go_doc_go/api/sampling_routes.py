"""
API routes for database sampling functionality using pipeline's analytics backend.
Architecture: MCP → API → Analytics DB Interface → Physical Analytics DB
"""

import json
import logging
import os
import yaml
from typing import Dict, List, Any, Optional
from flask import Blueprint, request, jsonify

from ..config_db.database import PipelineConfigDB
from ..storage_adapters.factory import StorageFactory

logger = logging.getLogger(__name__)

# Create Blueprint
sampling_bp = Blueprint('sampling', __name__, url_prefix='/api/sampling')

# Cache for analytics backends by pipeline
_backend_cache = {}




def get_analytics_backend_for_pipeline(pipeline_name: str):
    """
    Get or create analytics backend for a specific pipeline.

    Args:
        pipeline_name: Name of the pipeline (e.g., "test-updated1")

    Returns:
        Analytics storage backend instance
    """
    # Check cache first
    cache_key = pipeline_name
    if cache_key in _backend_cache:
        return _backend_cache[cache_key]

    try:
        # Get pipeline configuration - use same database as pipeline routes
        db_path = os.environ.get('PIPELINE_CONFIG_DB', 'pipeline_config.db')
        logger.info(f"Looking for pipeline '{pipeline_name}' in database: {db_path}")
        pipeline_db = PipelineConfigDB(db_path)
        pipeline = pipeline_db.get_pipeline_by_name(pipeline_name)

        if not pipeline or not pipeline.config_yaml:
            raise ValueError(f"Pipeline '{pipeline_name}' not found or has no configuration")

        # Parse pipeline config
        config = yaml.safe_load(pipeline.config_yaml)

        # Get analytics configuration - must be explicitly configured
        # Check under storage.analytics (correct location per config structure)
        storage_config = config.get('storage', {})
        analytics_config = storage_config.get('analytics')

        if not analytics_config:
            raise ValueError(f"Pipeline '{pipeline_name}' has no analytics configuration in storage.analytics")

        # Validate required fields
        if 'type' not in analytics_config:
            raise ValueError(f"Analytics configuration for pipeline '{pipeline_name}' missing required 'type' field")

        # Create analytics backend
        backend = StorageFactory.create_analytics_storage(analytics_config)
        _backend_cache[cache_key] = backend
        return backend

    except Exception as e:
        logger.error(f"Error getting analytics backend for pipeline {pipeline_name}: {e}")
        raise


@sampling_bp.route('/elements', methods=['POST'])
def sample_elements():
    """
    Sample elements from the database with flexible filtering.

    Request body:
    {
        "pipeline_name": "test-updated1",
        "run_id": "322241525733860c",  # optional - filters to specific run if provided
        "filters": {"element_type": "xml_element", ...},
        "limit": 100,
        "stratify_by": "element_type",
        "random_seed": 42
    }
    """
    try:
        data = request.get_json() or {}
        pipeline_name = data.get('pipeline_name', 'test-updated1')

        # Use provided run_id if available (optional parameter)
        run_id = data.get('run_id')

        # Get analytics backend for pipeline
        backend = get_analytics_backend_for_pipeline(pipeline_name)

        # Use the backend's sampling method with run_id filter
        results = backend.sample_elements(
            filters=data.get('filters'),
            limit=data.get('limit', 100),
            stratify_by=data.get('stratify_by'),
            random_seed=data.get('random_seed'),
            run_id=run_id
        )

        return jsonify({
            'success': True,
            'elements': results,
            'count': len(results),
            'pipeline': pipeline_name,
            'run_id': run_id,
            'filters_applied': data.get('filters'),
            'stratified_by': data.get('stratify_by')
        })

    except Exception as e:
        logger.error(f"Error sampling elements: {e}")
        return jsonify({
            'success': False,
            'error': str(e)
        }), 500


@sampling_bp.route('/corpus-stats', methods=['POST'])
def get_corpus_stats():
    """
    Get statistics about the document corpus.

    Request body:
    {
        "pipeline_name": "test-updated1",
        "run_id": "322241525733860c",  # optional - filters to specific run if provided
        "filters": {"format_type": "xml", ...}
    }
    """
    try:
        data = request.get_json() or {}
        pipeline_name = data.get('pipeline_name', 'test-updated1')

        # Use provided run_id if available (optional parameter)
        run_id = data.get('run_id')

        # Get analytics backend for pipeline
        backend = get_analytics_backend_for_pipeline(pipeline_name)

        # Use the backend's statistics method with run_id filter
        stats = backend.get_corpus_stats(
            filters=data.get('filters'),
            run_id=run_id
        )

        return jsonify({
            'success': True,
            'statistics': stats,
            'pipeline': pipeline_name,
            'run_id': run_id,
            'filters_applied': data.get('filters')
        })

    except Exception as e:
        logger.error(f"Error getting corpus stats: {e}")
        return jsonify({
            'success': False,
            'error': str(e)
        }), 500


@sampling_bp.route('/documents', methods=['POST'])
def sample_documents():
    """
    Sample documents from the database.

    Request body:
    {
        "pipeline_name": "test-updated1",
        "run_id": "322241525733860c",  # optional - filters to specific run if provided
        "filters": {"source": "*form*", ...},
        "limit": 50,
        "random_seed": 42
    }
    """
    try:
        data = request.get_json() or {}
        pipeline_name = data.get('pipeline_name', 'test-updated1')

        # Use provided run_id if available (optional parameter)
        run_id = data.get('run_id')

        # Get analytics backend for pipeline
        backend = get_analytics_backend_for_pipeline(pipeline_name)

        # Use the backend's sampling method with run_id filter
        results = backend.sample_documents(
            filters=data.get('filters'),
            limit=data.get('limit', 50),
            random_seed=data.get('random_seed'),
            run_id=run_id
        )

        return jsonify({
            'success': True,
            'documents': results,
            'count': len(results),
            'pipeline': pipeline_name,
            'run_id': run_id,
            'filters_applied': data.get('filters')
        })

    except Exception as e:
        logger.error(f"Error sampling documents: {e}")
        return jsonify({
            'success': False,
            'error': str(e)
        }), 500


@sampling_bp.route('/custom-query', methods=['POST'])
def execute_custom_query():
    """
    Execute a custom SQL query for advanced sampling.

    Request body:
    {
        "pipeline_name": "test-updated1",
        "run_id": "322241525733860c",  # optional - filters to specific run if provided
        "query": "SELECT * FROM elements WHERE ...",
        "params": [param1, param2, ...]
    }
    """
    try:
        data = request.get_json() or {}
        pipeline_name = data.get('pipeline_name', 'test-updated1')

        if not data.get('query'):
            return jsonify({
                'success': False,
                'error': 'Query is required'
            }), 400

        # Basic safety check - only allow SELECT queries
        query = data['query'].strip()
        if not query.upper().startswith('SELECT'):
            return jsonify({
                'success': False,
                'error': 'Only SELECT queries are allowed'
            }), 403

        # Additional safety checks
        forbidden_keywords = ['INSERT', 'UPDATE', 'DELETE', 'DROP', 'CREATE', 'ALTER', 'TRUNCATE']
        query_upper = query.upper()
        for keyword in forbidden_keywords:
            if keyword in query_upper:
                return jsonify({
                    'success': False,
                    'error': f'Query contains forbidden keyword: {keyword}'
                }), 403

        # Use provided run_id if available (optional parameter)
        run_id = data.get('run_id')

        # Get analytics backend for pipeline
        backend = get_analytics_backend_for_pipeline(pipeline_name)

        # Use the backend's custom query method with run_id filter
        results = backend.execute_custom_query(
            query=query,
            params=data.get('params'),
            run_id=run_id
        )

        return jsonify({
            'success': True,
            'results': results,
            'count': len(results),
            'pipeline': pipeline_name,
            'run_id': run_id,
            'query': query
        })

    except Exception as e:
        logger.error(f"Error executing custom query: {e}")
        return jsonify({
            'success': False,
            'error': str(e)
        }), 500


@sampling_bp.route('/schema', methods=['GET'])
def get_schema_info():
    """
    Get information about available columns and types for filtering.

    Query Parameters:
    - pipeline_name: Name of the pipeline (default: test-updated1)
    """
    try:
        pipeline_name = request.args.get('pipeline_name', 'test-updated1')

        # Get analytics backend for pipeline
        backend = get_analytics_backend_for_pipeline(pipeline_name)

        # Get a sample element to determine schema
        sample_elements = backend.sample_elements(limit=1)

        if not sample_elements:
            return jsonify({
                'success': False,
                'error': 'No data available in pipeline'
            }), 404

        # Extract column information from sample
        columns = []
        for key, value in sample_elements[0].items():
            columns.append({
                'name': key,
                'type': type(value).__name__,
                'nullable': value is None
            })

        # Get some example values for key columns
        examples = {}
        example_columns = ['element_type', 'format_type', 'structural_name']

        for col in example_columns:
            # Use custom query to get distinct values
            try:
                query = f"""
                SELECT DISTINCT {col}, COUNT(*) as count
                FROM elements
                WHERE {col} IS NOT NULL
                GROUP BY {col}
                ORDER BY count DESC
                LIMIT 10
                """

                results = backend.execute_custom_query(query)
                examples[col] = [{'value': row.get(col), 'count': row.get('count')} for row in results]
            except:
                # If column doesn't exist or query fails, skip
                continue

        return jsonify({
            'success': True,
            'pipeline': pipeline_name,
            'columns': columns,
            'examples': examples
        })

    except Exception as e:
        logger.error(f"Error getting schema info: {e}")
        return jsonify({
            'success': False,
            'error': str(e)
        }), 500


@sampling_bp.route('/ontology-sample', methods=['POST'])
def sample_for_ontology():
    """
    Comprehensive sampling specifically for ontology generation.

    Request body:
    {
        "pipeline_name": "test-updated1",
        "run_id": "322241525733860c",  # optional - filters to specific run if provided
        "domain_keywords": ["insider", "trading", ...],
        "max_elements": 200,
        "include_stats": true
    }
    """
    try:
        data = request.get_json() or {}
        pipeline_name = data.get('pipeline_name', 'test-updated1')
        keywords = data.get('domain_keywords', [])
        max_elements = data.get('max_elements', 200)

        # Use provided run_id if available (optional parameter)
        run_id = data.get('run_id')

        # Get analytics backend for pipeline
        backend = get_analytics_backend_for_pipeline(pipeline_name)

        response = {
            'success': True,
            'pipeline': pipeline_name,
            'run_id': run_id,
            'domain_keywords': keywords
        }

        # Get corpus stats
        if data.get('include_stats', True):
            response['corpus_stats'] = backend.get_corpus_stats(run_id=run_id)

        # Sample diverse elements
        all_samples = []

        # Sample by keywords
        for keyword in keywords[:5]:  # Limit to 5 keywords
            keyword_samples = backend.sample_elements(
                filters={'structural_name': f'*{keyword}*'},
                limit=max_elements // 10,
                run_id=run_id
            )
            all_samples.extend(keyword_samples)

        # Sample different element types
        for element_type in ['xml_element', 'json_field', 'csv_cell']:
            type_samples = backend.sample_elements(
                filters={'element_type': element_type},
                limit=max_elements // 6,
                run_id=run_id
            )
            all_samples.extend(type_samples)

        # Deduplicate by element_id
        seen = set()
        unique_samples = []
        for sample in all_samples:
            element_id = sample.get('element_id')
            if element_id and element_id not in seen:
                seen.add(element_id)
                unique_samples.append(sample)
                if len(unique_samples) >= max_elements:
                    break

        response['samples'] = unique_samples
        response['sample_count'] = len(unique_samples)

        # Pattern analysis
        patterns = {
            'structural_names': {},
            'element_types': {},
            'paths': []
        }

        for sample in unique_samples:
            name = sample.get('structural_name', 'unknown')
            patterns['structural_names'][name] = patterns['structural_names'].get(name, 0) + 1

            etype = sample.get('element_type', 'unknown')
            patterns['element_types'][etype] = patterns['element_types'].get(etype, 0) + 1

            if sample.get('structural_path'):
                patterns['paths'].append(sample['structural_path'])

        # Sort patterns by frequency
        patterns['structural_names'] = dict(sorted(
            patterns['structural_names'].items(),
            key=lambda x: x[1],
            reverse=True
        )[:30])

        patterns['element_types'] = dict(sorted(
            patterns['element_types'].items(),
            key=lambda x: x[1],
            reverse=True
        ))

        response['patterns'] = patterns

        return jsonify(response)

    except Exception as e:
        logger.error(f"Error in ontology sampling: {e}")
        return jsonify({
            'success': False,
            'error': str(e)
        }), 500