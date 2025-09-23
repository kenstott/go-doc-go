"""
API routes for database sampling functionality with pipeline support.
"""

import json
import logging
import yaml
from typing import Dict, List, Any, Optional
from flask import Blueprint, request, jsonify

from ..config_db.database import PipelineConfigDB
from ..storage_adapters.factory import StorageFactory
import duckdb

logger = logging.getLogger(__name__)

# Create Blueprint
sampling_bp = Blueprint('sampling', __name__, url_prefix='/api/sampling')

# Cache for analytics backends by pipeline
_backend_cache = {}


def get_analytics_backend_for_pipeline(pipeline_name: str):
    """Get or create analytics backend for a specific pipeline."""

    # Check cache first
    if pipeline_name in _backend_cache:
        return _backend_cache[pipeline_name]

    try:
        # Get pipeline configuration
        db_path = "pipeline_exec.db"  # Default path
        pipeline_db = PipelineConfigDB(db_path)
        pipeline = pipeline_db.get_pipeline_by_name(pipeline_name)

        if not pipeline or not pipeline.config_yaml:
            raise ValueError(f"Pipeline '{pipeline_name}' not found or has no configuration")

        # Parse pipeline config
        config = yaml.safe_load(pipeline.config_yaml)

        # Get analytics configuration
        analytics_config = config.get('analytics', {})
        if not analytics_config:
            # Default to parquet if not specified
            analytics_config = {
                'type': 'parquet',
                'base_path': f'./data-lake/{pipeline_name}'
            }

        # Create analytics backend
        backend = StorageFactory.create_analytics_storage(analytics_config)
        _backend_cache[pipeline_name] = backend
        return backend

    except Exception as e:
        logger.error(f"Error getting analytics backend for pipeline {pipeline_name}: {e}")
        raise


def query_parquet_with_duckdb(pipeline_name: str, query: str, params: Optional[List] = None) -> List[Dict[str, Any]]:
    """
    Execute a query against parquet files using DuckDB.

    Args:
        pipeline_name: Name of the pipeline
        query: SQL query to execute
        params: Optional query parameters

    Returns:
        List of result dictionaries
    """
    # Get base path for pipeline's data
    base_path = f'./data-lake/{pipeline_name}'

    # Create DuckDB connection
    conn = duckdb.connect(':memory:')

    try:
        # Register parquet files as views
        conn.execute(f"""
            CREATE VIEW elements AS
            SELECT * FROM read_parquet('{base_path}/elements/**/*.parquet', hive_partitioning=true)
        """)

        conn.execute(f"""
            CREATE VIEW documents AS
            SELECT * FROM read_parquet('{base_path}/documents/**/*.parquet', hive_partitioning=true)
        """)

        conn.execute(f"""
            CREATE VIEW relationships AS
            SELECT * FROM read_parquet('{base_path}/relationships/**/*.parquet', hive_partitioning=true)
        """)

        # Create enriched view joining elements and documents
        conn.execute("""
            CREATE VIEW element_document_enriched AS
            SELECT
                e.*,
                d.source as doc_source,
                d.doc_type,
                d.metadata as doc_metadata,
                -- Extract structural info from element metadata
                json_extract_string(e.metadata, '$.element_name') as structural_name,
                json_extract_string(e.metadata, '$.path') as structural_path,
                CASE
                    WHEN e.element_type = 'xml_element' THEN 'xml'
                    WHEN e.element_type = 'json_field' THEN 'json'
                    WHEN e.element_type = 'csv_cell' THEN 'csv'
                    ELSE 'other'
                END as format_type,
                -- Check for temporal values
                CASE
                    WHEN e.content_preview SIMILAR TO '[0-9]{4}-[0-9]{2}-[0-9]{2}' THEN true
                    ELSE false
                END as has_temporal_value
            FROM elements e
            LEFT JOIN documents d ON e.doc_id = d.doc_id
        """)

        # Execute the actual query
        if params:
            result = conn.execute(query, params).fetchall()
        else:
            result = conn.execute(query).fetchall()

        # Get column names
        columns = [desc[0] for desc in conn.description()]

        # Convert to list of dictionaries
        results = []
        for row in result:
            results.append(dict(zip(columns, row)))

        return results

    finally:
        conn.close()


@sampling_bp.route('/elements', methods=['POST'])
def sample_elements():
    """
    Sample elements from the database with flexible filtering.

    Request body:
    {
        "pipeline_name": "test-updated1",
        "filters": {"element_type": "xml_element", ...},
        "limit": 100,
        "stratify_by": "element_type",
        "random_seed": 42
    }
    """
    try:
        data = request.get_json() or {}
        pipeline_name = data.get('pipeline_name', 'test-updated1')

        filters = data.get('filters', {})
        limit = data.get('limit', 100)
        stratify_by = data.get('stratify_by')
        random_seed = data.get('random_seed')

        # Build WHERE clause
        where_conditions = []
        params = []

        if filters:
            for column, value in filters.items():
                if isinstance(value, list):
                    placeholders = ','.join(['?' for _ in value])
                    where_conditions.append(f"{column} IN ({placeholders})")
                    params.extend(value)
                elif isinstance(value, str) and '*' in value:
                    where_conditions.append(f"{column} LIKE ?")
                    params.append(value.replace('*', '%'))
                else:
                    where_conditions.append(f"{column} = ?")
                    params.append(value)

        where_clause = " AND ".join(where_conditions) if where_conditions else "TRUE"

        # Build query
        if stratify_by and limit > 10:
            # Stratified sampling
            query = f"""
            WITH stratified AS (
                SELECT *,
                    ROW_NUMBER() OVER (
                        PARTITION BY {stratify_by}
                        ORDER BY RANDOM({random_seed if random_seed else ''})
                    ) as rn
                FROM element_document_enriched
                WHERE {where_clause}
            )
            SELECT * FROM stratified
            WHERE rn <= ? / (SELECT COUNT(DISTINCT {stratify_by}) FROM element_document_enriched WHERE {where_clause})
            LIMIT ?
            """
            params.extend([limit, limit])
        else:
            # Simple random sampling
            seed_clause = f"USING SAMPLE {limit} ROWS (RANDOM({random_seed}))" if random_seed else f"USING SAMPLE {limit} ROWS"
            query = f"""
            SELECT * FROM element_document_enriched
            WHERE {where_clause}
            {seed_clause}
            """

        results = query_parquet_with_duckdb(pipeline_name, query, params if where_conditions else None)

        return jsonify({
            'success': True,
            'elements': results,
            'count': len(results),
            'pipeline': pipeline_name,
            'filters_applied': filters,
            'stratified_by': stratify_by
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
        "filters": {"format_type": "xml", ...}
    }
    """
    try:
        data = request.get_json() or {}
        pipeline_name = data.get('pipeline_name', 'test-updated1')
        filters = data.get('filters', {})

        # Build WHERE clause
        where_conditions = []
        params = []

        if filters:
            for column, value in filters.items():
                if isinstance(value, list):
                    placeholders = ','.join(['?' for _ in value])
                    where_conditions.append(f"{column} IN ({placeholders})")
                    params.extend(value)
                elif isinstance(value, str) and '*' in value:
                    where_conditions.append(f"{column} LIKE ?")
                    params.append(value.replace('*', '%'))
                else:
                    where_conditions.append(f"{column} = ?")
                    params.append(value)

        where_clause = " AND ".join(where_conditions) if where_conditions else "TRUE"

        # Get comprehensive statistics
        stats_query = f"""
        SELECT
            COUNT(*) as total_elements,
            COUNT(DISTINCT doc_id) as total_documents,
            COUNT(DISTINCT element_type) as distinct_element_types,
            COUNT(DISTINCT structural_name) as distinct_structural_names,
            COUNT(*) FILTER (WHERE has_temporal_value = TRUE) as temporal_elements
        FROM element_document_enriched
        WHERE {where_clause}
        """

        stats_result = query_parquet_with_duckdb(pipeline_name, stats_query, params if where_conditions else None)

        # Get distribution data
        type_dist_query = f"""
        SELECT element_type, COUNT(*) as count
        FROM element_document_enriched
        WHERE {where_clause}
        GROUP BY element_type
        ORDER BY count DESC
        """

        type_dist = query_parquet_with_duckdb(pipeline_name, type_dist_query, params if where_conditions else None)

        # Format statistics
        stats = stats_result[0] if stats_result else {}
        stats['element_type_distribution'] = {row['element_type']: row['count'] for row in type_dist}

        return jsonify({
            'success': True,
            'statistics': stats,
            'pipeline': pipeline_name,
            'filters_applied': filters
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
        "filters": {"source": "*form*", ...},
        "limit": 50,
        "random_seed": 42
    }
    """
    try:
        data = request.get_json() or {}
        pipeline_name = data.get('pipeline_name', 'test-updated1')

        filters = data.get('filters', {})
        limit = data.get('limit', 50)
        random_seed = data.get('random_seed')

        # Build WHERE clause
        where_conditions = []
        params = []

        if filters:
            for column, value in filters.items():
                if isinstance(value, list):
                    placeholders = ','.join(['?' for _ in value])
                    where_conditions.append(f"{column} IN ({placeholders})")
                    params.extend(value)
                elif isinstance(value, str) and '*' in value:
                    where_conditions.append(f"{column} LIKE ?")
                    params.append(value.replace('*', '%'))
                else:
                    where_conditions.append(f"{column} = ?")
                    params.append(value)

        where_clause = " AND ".join(where_conditions) if where_conditions else "TRUE"

        # Sample documents
        seed_clause = f"USING SAMPLE {limit} ROWS (RANDOM({random_seed}))" if random_seed else f"USING SAMPLE {limit} ROWS"
        query = f"""
        SELECT DISTINCT doc_id, source, doc_type, metadata
        FROM documents
        WHERE {where_clause}
        {seed_clause}
        """

        results = query_parquet_with_duckdb(pipeline_name, query, params if where_conditions else None)

        return jsonify({
            'success': True,
            'documents': results,
            'count': len(results),
            'pipeline': pipeline_name,
            'filters_applied': filters
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

        results = query_parquet_with_duckdb(pipeline_name, query, data.get('params'))

        return jsonify({
            'success': True,
            'results': results,
            'count': len(results),
            'pipeline': pipeline_name,
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

        # Get sample element to determine schema
        query = """
        SELECT * FROM element_document_enriched
        LIMIT 1
        """

        sample = query_parquet_with_duckdb(pipeline_name, query)

        if not sample:
            return jsonify({
                'success': False,
                'error': 'No data available in pipeline'
            }), 404

        # Extract column information from sample
        columns = []
        for key, value in sample[0].items():
            columns.append({
                'name': key,
                'type': type(value).__name__,
                'nullable': value is None
            })

        # Get some example values for key columns
        examples = {}
        example_columns = ['element_type', 'format_type', 'structural_name']

        for col in example_columns:
            query = f"""
            SELECT DISTINCT {col}, COUNT(*) as count
            FROM element_document_enriched
            WHERE {col} IS NOT NULL
            GROUP BY {col}
            ORDER BY count DESC
            LIMIT 10
            """

            results = query_parquet_with_duckdb(pipeline_name, query)
            examples[col] = [{'value': row[col], 'count': row['count']} for row in results]

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