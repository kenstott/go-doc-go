"""
API routes for database sampling functionality.
"""

import json
import logging
from typing import Dict, List, Any, Optional
from flask import Blueprint, request, jsonify

from ..storage.base import DocumentDatabase
from ..mcp.database_sampler import DatabaseSampler

logger = logging.getLogger(__name__)

# Create Blueprint
sampling_bp = Blueprint('sampling', __name__, url_prefix='/api/sampling')


def get_sampler() -> DatabaseSampler:
    """Get or create database sampler instance."""
    if not hasattr(get_sampler, '_instance'):
        # Get analytics backend connection
        from flask import current_app
        from ..storage_adapters.factory import StorageFactory

        # Try to get from app config first
        analytics_backend = current_app.config.get('analytics_backend')

        if not analytics_backend:
            # Create default analytics backend (parquet)
            analytics_config = {
                'type': 'parquet',
                'base_path': './data-lake',
                # For direct SQL access, we'd need PostgreSQL config
                # but for now we'll use parquet-based sampling
            }
            analytics_backend = StorageFactory.create_analytics_storage(analytics_config)

        # For SQL-based sampling, we need direct database connection
        # This is a temporary workaround - should be refactored
        import psycopg2
        import os

        # Get PostgreSQL connection for analytics database
        conn = psycopg2.connect(
            host=os.environ.get('POSTGRES_HOST', 'localhost'),
            port=os.environ.get('POSTGRES_PORT', '5432'),
            database=os.environ.get('POSTGRES_DB', 'doculyzer'),
            user=os.environ.get('POSTGRES_USER', 'postgres'),
            password=os.environ.get('POSTGRES_PASSWORD', 'postgres')
        )

        get_sampler._instance = DatabaseSampler(conn)
    return get_sampler._instance


@sampling_bp.route('/elements', methods=['POST'])
def sample_elements():
    """
    Sample elements from the database with flexible filtering.

    Request body:
    {
        "filters": {"element_type": "xml_element", ...},
        "limit": 100,
        "stratify_by": "element_type",
        "include_document_attrs": true,
        "random_seed": 42
    }
    """
    try:
        sampler = get_sampler()

        data = request.get_json() or {}

        results = sampler.sample_elements(
            filters=data.get('filters'),
            limit=data.get('limit', 100),
            stratify_by=data.get('stratify_by'),
            include_document_attrs=data.get('include_document_attrs', True),
            random_seed=data.get('random_seed')
        )

        return jsonify({
            'success': True,
            'elements': results,
            'count': len(results),
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
        "filters": {"format_type": "xml", ...}
    }
    """
    try:
        sampler = get_sampler()

        data = request.get_json() or {}

        stats = sampler.get_corpus_stats(
            filters=data.get('filters')
        )

        return jsonify({
            'success': True,
            'statistics': stats,
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
        "filters": {"source": "*form*", ...},
        "limit": 50,
        "random_seed": 42
    }
    """
    try:
        sampler = get_sampler()

        data = request.get_json() or {}

        results = sampler.sample_documents(
            filters=data.get('filters'),
            limit=data.get('limit', 50),
            random_seed=data.get('random_seed')
        )

        return jsonify({
            'success': True,
            'documents': results,
            'count': len(results),
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
        "query": "SELECT * FROM elements WHERE ...",
        "params": [param1, param2, ...]
    }
    """
    try:
        sampler = get_sampler()

        data = request.get_json() or {}

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

        results = sampler.execute_custom_query(
            query=query,
            params=data.get('params')
        )

        return jsonify({
            'success': True,
            'results': results,
            'count': len(results),
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
    """
    try:
        sampler = get_sampler()

        # Query to get column information
        query = """
        SELECT
            column_name,
            data_type,
            is_nullable
        FROM information_schema.columns
        WHERE table_name = 'element_document_enriched'
        ORDER BY ordinal_position
        """

        cursor = sampler.db.cursor()
        cursor.execute(query)

        columns = []
        for row in cursor.fetchall():
            columns.append({
                'name': row[0],
                'type': row[1],
                'nullable': row[2] == 'YES'
            })

        # Get some example values for key columns
        examples = {}
        example_columns = ['element_type', 'format_type', 'document_category', 'structural_name']

        for col in example_columns:
            query = f"""
            SELECT DISTINCT {col}, COUNT(*) as count
            FROM element_document_enriched
            WHERE {col} IS NOT NULL
            GROUP BY {col}
            ORDER BY COUNT(*) DESC
            LIMIT 10
            """
            cursor.execute(query)
            examples[col] = [{'value': row[0], 'count': row[1]} for row in cursor.fetchall()]

        return jsonify({
            'success': True,
            'columns': columns,
            'examples': examples
        })

    except Exception as e:
        logger.error(f"Error getting schema info: {e}")
        return jsonify({
            'success': False,
            'error': str(e)
        }), 500


# Helper endpoint for ontology generation
@sampling_bp.route('/ontology-sample', methods=['POST'])
def sample_for_ontology():
    """
    Comprehensive sampling specifically for ontology generation.

    Request body:
    {
        "domain_keywords": ["insider", "trading", ...],
        "max_elements": 200,
        "include_stats": true
    }
    """
    try:
        sampler = get_sampler()

        data = request.get_json() or {}
        keywords = data.get('domain_keywords', [])
        max_elements = data.get('max_elements', 200)

        # Build filters from keywords
        keyword_filters = []
        if keywords:
            for keyword in keywords:
                keyword_filters.append({
                    'structural_name': f'*{keyword}*'
                })

        response = {
            'success': True,
            'domain_keywords': keywords
        }

        # Get corpus stats
        if data.get('include_stats', True):
            response['corpus_stats'] = sampler.get_corpus_stats()

        # Sample diverse elements
        all_samples = []

        # High-frequency elements
        high_freq_query = """
        SELECT e.*, COUNT(*) OVER (PARTITION BY structural_name) as freq
        FROM element_document_enriched e
        WHERE structural_name IN (
            SELECT structural_name
            FROM element_document_enriched
            GROUP BY structural_name
            HAVING COUNT(*) >= 50
            ORDER BY COUNT(*) DESC
            LIMIT 20
        )
        ORDER BY RANDOM()
        LIMIT %s
        """
        high_freq = sampler.execute_custom_query(high_freq_query, [max_elements // 4])
        all_samples.extend(high_freq)

        # Keyword-based samples
        for kw_filter in keyword_filters[:5]:  # Limit to 5 keywords
            keyword_samples = sampler.sample_elements(
                filters=kw_filter,
                limit=max_elements // 10
            )
            all_samples.extend(keyword_samples)

        # Temporal elements
        temporal_samples = sampler.sample_elements(
            filters={'has_temporal_value': True},
            limit=max_elements // 4
        )
        all_samples.extend(temporal_samples)

        # Deduplicate by element_id
        seen = set()
        unique_samples = []
        for sample in all_samples:
            if sample.get('element_id') not in seen:
                seen.add(sample.get('element_id'))
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