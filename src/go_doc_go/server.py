import json
import logging
import os
from datetime import datetime
from typing import List

import yaml
from flask import Flask, request, jsonify, render_template_string, send_from_directory, send_file
from flask_cors import CORS
from werkzeug.exceptions import BadRequest, InternalServerError

from go_doc_go.adapter import create_content_resolver
from go_doc_go.config import Config
from go_doc_go.search import search_with_content, search_by_text, get_document_sources, SearchResult, search_structured, \
    search_simple_structured
from go_doc_go.api.flask_settings_routes import settings_bp
from go_doc_go.api.pipeline_routes import pipeline_bp

# Configure logging
log_level = os.environ.get('LOG_LEVEL', 'INFO')
log_format = os.environ.get('LOG_FORMAT', '%(asctime)s - %(name)s - %(levelname)s - %(message)s')
logging.basicConfig(level=getattr(logging, log_level), format=log_format)
logger = logging.getLogger(__name__)

# Lazy initialization to prevent import-time database connections
_config = None
db = None
resolver = None

def _ensure_initialized():
    """Ensure server components are initialized."""
    global _config, db, resolver
    if _config is None:
        _config = Config(os.environ.get('GO_DOC_GO_CONFIG_PATH', 'config.yaml'))
        db = _config.get_document_database()
        db.initialize()
        resolver = create_content_resolver(_config)

# Initialize Flask app
app = Flask(__name__)

# Configure CORS
cors_origins = os.environ.get('CORS_ORIGINS', '*').split(',')
CORS(app, origins=cors_origins)

# Register blueprints
app.register_blueprint(settings_bp)
app.register_blueprint(pipeline_bp)

# Get the directory where server.py is located
SERVER_DIR = os.path.dirname(os.path.abspath(__file__))

# Configuration from environment variables
CONFIG = {
    'HOST': os.environ.get('SERVER_HOST', '0.0.0.0'),
    'PORT': int(os.environ.get('SERVER_PORT', '5000')),
    'DEBUG': os.environ.get('DEBUG', 'False').lower() == 'true',
    'MAX_RESULTS': int(os.environ.get('MAX_RESULTS', '100')),
    'DEFAULT_RESULTS': int(os.environ.get('DEFAULT_RESULTS', '10')),
    'MIN_SCORE_THRESHOLD': float(os.environ.get('MIN_SCORE_THRESHOLD', '0.0')),
    'TIMEOUT': int(os.environ.get('REQUEST_TIMEOUT', '30')),
    'MAX_CONTENT_LENGTH': int(os.environ.get('MAX_CONTENT_LENGTH', '16777216')),  # 16MB
    'RATE_LIMIT': os.environ.get('RATE_LIMIT', '100 per minute'),
    'API_KEY': os.environ.get('API_KEY'),  # Optional API key for authentication
    'API_KEY_HEADER': os.environ.get('API_KEY_HEADER', 'X-API-Key'),
    'OPENAPI_SPEC_PATH': os.environ.get('OPENAPI_SPEC_PATH', os.path.join(SERVER_DIR, 'openapi.yaml')),
    'SWAGGER_UI_ENABLED': os.environ.get('SWAGGER_UI_ENABLED', 'True').lower() == 'true',
    'SWAGGER_UI_PATH': os.environ.get('SWAGGER_UI_PATH', '/docs'),
    'API_SPEC_PATH': os.environ.get('API_SPEC_PATH', '/api/spec'),
}

# Set Flask configuration
app.config['MAX_CONTENT_LENGTH'] = CONFIG['MAX_CONTENT_LENGTH']


# Load OpenAPI specification
def load_openapi_spec():
    """Load the OpenAPI specification from file."""
    try:
        spec_path = CONFIG['OPENAPI_SPEC_PATH']
        if not os.path.exists(spec_path):
            logger.warning(f"OpenAPI spec file not found at {spec_path}")
            return None

        with open(spec_path, 'r') as f:
            if spec_path.endswith('.yaml') or spec_path.endswith('.yml'):
                spec = yaml.safe_load(f)
            else:
                spec = json.load(f)

        # Update server URLs with current configuration
        if 'servers' not in spec:
            spec['servers'] = []

        # Add current server URL
        current_server = f"http://{CONFIG['HOST']}:{CONFIG['PORT']}"
        spec['servers'].insert(0, {
            'url': current_server,
            'description': 'Current server'
        })

        return spec
    except Exception as e:
        logger.error(f"Error loading OpenAPI spec: {str(e)}")
        return None


# Authentication middleware
def check_api_key():
    """Check API key if configured."""
    if CONFIG['API_KEY']:
        api_key = request.headers.get(CONFIG['API_KEY_HEADER'])
        if not api_key or api_key != CONFIG['API_KEY']:
            return jsonify({
                'error': 'Unauthorized',
                'message': 'Invalid or missing API key'
            }), 401
    return None


# Error handlers
@app.errorhandler(400)
def bad_request(error):
    return jsonify({
        'error': 'Bad Request',
        'message': str(error.description)
    }), 400


@app.errorhandler(404)
def not_found(error):
    return jsonify({
        'error': 'Not Found',
        'message': 'Resource not found'
    }), 404


@app.errorhandler(500)
def internal_error(error):
    logger.error(f"Internal server error: {str(error)}")
    return jsonify({
        'error': 'Internal Server Error',
        'message': 'An unexpected error occurred'
    }), 500


# Root endpoint with API documentation links
# Commented out to allow React app to be served at root
# @app.route('/', methods=['GET'])
# def root():
#     """Root endpoint with links to documentation."""
#     response_data = {
#         'name': 'Document Search API',
#         'version': '1.0.0',
#         'status': 'running',
#         'links': {
#             'api_documentation': CONFIG['SWAGGER_UI_PATH'] if CONFIG['SWAGGER_UI_ENABLED'] else None,
#             'openapi_spec': CONFIG['API_SPEC_PATH'],
#             'health': '/health',
#             'api_info': '/api/info'
#         }
#     }
#
#     return jsonify({k: v for k, v in response_data.items() if v is not None})

# API info endpoint (moved from root)
@app.route('/api/info', methods=['GET'])
def api_info():
    """API information endpoint."""
    response_data = {
        'name': 'Document Search API',
        'version': '1.0.0',
        'status': 'running',
        'links': {
            'api_documentation': CONFIG['SWAGGER_UI_PATH'] if CONFIG['SWAGGER_UI_ENABLED'] else None,
            'openapi_spec': CONFIG['API_SPEC_PATH'],
            'health': '/health'
        }
    }

    return jsonify({k: v for k, v in response_data.items() if v is not None})


# OpenAPI specification endpoint
@app.route(CONFIG['API_SPEC_PATH'], methods=['GET'])
def openapi_spec():
    """Serve the OpenAPI specification."""
    spec = load_openapi_spec()
    if spec is None:
        return jsonify({
            'error': 'Not Found',
            'message': 'OpenAPI specification not available'
        }), 404

    return jsonify(spec)


# Swagger UI endpoint
if CONFIG['SWAGGER_UI_ENABLED']:
    # Swagger UI HTML template
    SWAGGER_UI_TEMPLATE = """
    <!DOCTYPE html>
    <html lang="en">
    <head>
        <meta charset="UTF-8">
        <title>Document Search API - Swagger UI</title>
        <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@5.10.3/swagger-ui.css">
        <style>
            html { box-sizing: border-box; overflow: -moz-scrollbars-vertical; overflow-y: scroll; }
            *, *:before, *:after { box-sizing: inherit; }
            body { margin:0; background: #fafafa; }
        </style>
    </head>
    <body>
        <div id="swagger-ui"></div>
        <script src="https://unpkg.com/swagger-ui-dist@5.10.3/swagger-ui-bundle.js"></script>
        <script src="https://unpkg.com/swagger-ui-dist@5.10.3/swagger-ui-standalone-preset.js"></script>
        <script>
            window.onload = function() {
                const ui = SwaggerUIBundle({
                    url: "{{ openapi_url }}",
                    dom_id: '#swagger-ui',
                    deepLinking: true,
                    presets: [
                        SwaggerUIBundle.presets.apis,
                        SwaggerUIStandalonePreset
                    ],
                    plugins: [
                        SwaggerUIBundle.plugins.DownloadUrl
                    ],
                    layout: "StandaloneLayout",
                    defaultModelsExpandDepth: 1,
                    defaultModelExampleFormat: "value",
                    tryItOutEnabled: true,
                    persistAuthorization: true
                });

                window.ui = ui;
            };
        </script>
    </body>
    </html>
    """


    @app.route(CONFIG['SWAGGER_UI_PATH'], methods=['GET'])
    def swagger_ui():
        """Serve Swagger UI."""
        openapi_url = f"{CONFIG['API_SPEC_PATH']}"
        return render_template_string(SWAGGER_UI_TEMPLATE, openapi_url=openapi_url)


# Health check endpoint
@app.route('/health', methods=['GET'])
def health_check():
    """Health check endpoint."""
    return jsonify({
        'status': 'healthy',
        'timestamp': datetime.utcnow().isoformat(),
        'version': '1.0.0'
    })


# API Info endpoint - commented out to avoid duplicate
# @app.route('/api/info', methods=['GET'])
# def api_info_old():
#     """Get API information and available endpoints."""
#     info_data = {
#         'name': 'Document Search API',
#         'version': '1.0.0',
#         'endpoints': {
#             '/health': 'Health check',
#             '/api/info': 'API information',
#             '/api/search': 'Search for elements',
#             '/api/search/advanced': 'Advanced search with full results',
#             '/api/search/structured': 'Structured search with complex criteria',
#             '/api/search/sources': 'Get document sources'
#         },
#         'configuration': {
#             'max_results': CONFIG['MAX_RESULTS'],
#             'default_results': CONFIG['DEFAULT_RESULTS'],
#             'min_score_threshold': CONFIG['MIN_SCORE_THRESHOLD'],
#             'timeout': CONFIG['TIMEOUT']
#         }
#     }
# 
#     if CONFIG['SWAGGER_UI_ENABLED']:
#         info_data['documentation'] = CONFIG['SWAGGER_UI_PATH']
#         info_data['openapi_spec'] = CONFIG['API_SPEC_PATH']
# 
#     return jsonify(info_data)


# Helper function to extract topic parameters
def extract_topic_parameters(data):
    """Extract topic-related parameters from request data."""
    include_topics = data.get('include_topics')
    exclude_topics = data.get('exclude_topics')
    min_confidence = data.get('min_confidence')

    # Validate topic parameters
    if include_topics is not None and not isinstance(include_topics, list):
        raise BadRequest("'include_topics' must be a list of strings")

    if exclude_topics is not None and not isinstance(exclude_topics, list):
        raise BadRequest("'exclude_topics' must be a list of strings")

    if min_confidence is not None:
        if not isinstance(min_confidence, (int, float)) or min_confidence < 0.0 or min_confidence > 1.0:
            raise BadRequest("'min_confidence' must be a number between 0.0 and 1.0")

    return include_topics, exclude_topics, min_confidence


# NEW STRUCTURED SEARCH ENDPOINT
@app.route('/api/search/structured', methods=['POST'])
def structured_search_endpoint():
    """
    Structured search with complex criteria and logical operators.

    Request body should be a complete SearchQueryRequest JSON object with criteria groups,
    logical operators, and multiple search types (semantic, topic, date, metadata, element).

    Example request body:
    {
        "criteria_group": {
            "operator": "AND",
            "semantic_search": {
                "query_text": "machine learning algorithms",
                "similarity_threshold": 0.8
            },
            "topic_search": {
                "include_topics": ["ai%", "ml%"],
                "exclude_topics": ["deprecated%"],
                "min_confidence": 0.8
            },
            "date_search": {
                "operator": "relative_days",
                "relative_value": 30
            }
        },
        "limit": 20,
        "include_similarity_scores": true,
        "include_topics": true
    }

    Optional query parameters:
    - text: boolean - Whether to resolve text content
    - content: boolean - Whether to resolve content
    - flat: boolean - Whether to return flat results
    - include_parents: boolean - Whether to include parent elements
    """
    # Check API key if required
    auth_response = check_api_key()
    if auth_response:
        return auth_response

    try:
        # Parse request JSON
        data = request.get_json()
        if not data:
            raise BadRequest("Request body must be valid JSON containing a SearchQueryRequest")

        # Extract optional query parameters for content materialization
        text = request.args.get('text', 'false').lower() == 'true'
        content = request.args.get('content', 'false').lower() == 'true'
        flat = request.args.get('flat', 'false').lower() == 'true'
        include_parents = request.args.get('include_parents', 'true').lower() == 'true'

        # Log the structured search request
        logger.info(f"Structured search request: text={text}, content={content}, "
                    f"flat={flat}, include_parents={include_parents}")

        # Log a summary of the criteria for debugging
        criteria_group = data.get('criteria_group', {})
        search_types = []
        if criteria_group.get('semantic_search'):
            search_types.append('semantic')
        if criteria_group.get('topic_search'):
            search_types.append('topic')
        if criteria_group.get('date_search'):
            search_types.append('date')
        if criteria_group.get('metadata_search'):
            search_types.append('metadata')
        if criteria_group.get('element_search'):
            search_types.append('element')

        logger.info(f"Structured search types: {search_types}, operator: {criteria_group.get('operator', 'N/A')}")

        # Call structured search with the JSON data and content materialization options
        results = search_structured(
            query=data,  # Pass the entire JSON payload
            text=text,
            content=content,
            flat=flat,
            include_parents=include_parents
        )

        # Convert results to JSON using model_dump_json() for proper serialization
        json_str = results.model_dump_json()
        json_dict = json.loads(json_str)

        return jsonify(json_dict)

    except ValueError as e:
        # Handle Pydantic validation errors
        logger.error(f"Structured search validation error: {str(e)}")
        raise BadRequest(f"Invalid structured query format: {str(e)}")

    except Exception as e:
        logger.error(f"Structured search error: {str(e)}")
        raise InternalServerError(f"Structured search operation failed: {str(e)}")


# SIMPLE STRUCTURED SEARCH ENDPOINT (convenience method)
@app.route('/api/search/structured/simple', methods=['POST'])
def simple_structured_search_endpoint():
    """
    Simple structured search with common parameters.

    Request body:
    {
        "query_text": "machine learning algorithms",
        "limit": 10,
        "similarity_threshold": 0.7,
        "include_topics": ["ai%", "ml%"],
        "exclude_topics": ["deprecated%"],
        "days_back": 30,
        "element_types": ["paragraph", "header"]
    }

    Optional query parameters:
    - text: boolean - Whether to resolve text content
    - content: boolean - Whether to resolve content
    - flat: boolean - Whether to return flat results
    - include_parents: boolean - Whether to include parent elements
    """
    # Check API key if required
    auth_response = check_api_key()
    if auth_response:
        return auth_response

    try:
        # Parse request JSON
        data = request.get_json()
        if not data:
            raise BadRequest("Request body must be valid JSON")

        # Extract required parameter
        query_text = data.get('query_text')
        if not query_text:
            raise BadRequest("'query_text' parameter is required")

        # Extract optional parameters with defaults
        limit = min(data.get('limit', CONFIG['DEFAULT_RESULTS']), CONFIG['MAX_RESULTS'])
        similarity_threshold = data.get('similarity_threshold', 0.7)
        include_topics = data.get('include_topics')
        exclude_topics = data.get('exclude_topics')
        days_back = data.get('days_back')
        element_types = data.get('element_types')

        # Extract content materialization options from query parameters
        text = request.args.get('text', 'false').lower() == 'true'
        content = request.args.get('content', 'false').lower() == 'true'
        flat = request.args.get('flat', 'false').lower() == 'true'
        include_parents = request.args.get('include_parents', 'true').lower() == 'true'

        # Validate parameters
        if similarity_threshold < 0.0 or similarity_threshold > 1.0:
            raise BadRequest("'similarity_threshold' must be between 0.0 and 1.0")

        if include_topics is not None and not isinstance(include_topics, list):
            raise BadRequest("'include_topics' must be a list of strings")

        if exclude_topics is not None and not isinstance(exclude_topics, list):
            raise BadRequest("'exclude_topics' must be a list of strings")

        if element_types is not None and not isinstance(element_types, list):
            raise BadRequest("'element_types' must be a list of strings")

        if days_back is not None and (not isinstance(days_back, int) or days_back <= 0):
            raise BadRequest("'days_back' must be a positive integer")

        # Log the simple structured search request
        logger.info(f"Simple structured search: query='{query_text}', limit={limit}, "
                    f"similarity_threshold={similarity_threshold}, include_topics={include_topics}, "
                    f"exclude_topics={exclude_topics}, days_back={days_back}, element_types={element_types}")

        # Call simple structured search
        results = search_simple_structured(
            query_text=query_text,
            limit=limit,
            similarity_threshold=similarity_threshold,
            include_topics=include_topics,
            exclude_topics=exclude_topics,
            days_back=days_back,
            element_types=element_types,
            text=text,
            content=content,
            flat=flat,
            include_parents=include_parents
        )

        # Convert results to JSON using model_dump_json() for proper serialization
        json_str = results.model_dump_json()
        json_dict = json.loads(json_str)

        return jsonify(json_dict)

    except Exception as e:
        logger.error(f"Simple structured search error: {str(e)}")
        raise InternalServerError(f"Simple structured search operation failed: {str(e)}")


# Standard search endpoint
@app.route('/api/search', methods=['POST'])
def search_endpoint():
    """
    Search for elements and return basic results.

    Request body:
    {
        "query": "search text",
        "limit": 10,
        "min_score": 0.0,
        "filter_criteria": {},
        "include_topics": ["security%", "%.policy%"],
        "exclude_topics": ["deprecated%"],
        "min_confidence": 0.7,
        "text": false,
        "content": false,
        "flat": false,
        "include_parents": true
    }
    """
    # Check API key if required
    auth_response = check_api_key()
    if auth_response:
        return auth_response

    try:
        # Parse request JSON
        data = request.get_json()
        if not data:
            raise BadRequest("Request body must be valid JSON")

        # Extract parameters
        query_text = data.get('query')
        if not query_text:
            raise BadRequest("'query' parameter is required")

        limit = min(data.get('limit', CONFIG['DEFAULT_RESULTS']), CONFIG['MAX_RESULTS'])
        flat = data.get('flat', False)
        include_parents = data.get('include_parents', True)
        min_score = max(data.get('min_score', CONFIG['MIN_SCORE_THRESHOLD']), 0.0)
        filter_criteria = data.get('filter_criteria', {})
        text = data.get('text', False)
        content = data.get('content', False)

        # Extract topic parameters
        include_topics, exclude_topics, min_confidence = extract_topic_parameters(data)

        # Perform search
        logger.info(f"Search request: query='{query_text}', limit={limit}, min_score={min_score}, "
                   f"flat={flat}, include_parents={include_parents}, include_topics={include_topics}, "
                   f"exclude_topics={exclude_topics}, min_confidence={min_confidence}")

        results = search_by_text(
            query_text=query_text,
            limit=limit,
            filter_criteria=filter_criteria,
            include_topics=include_topics,
            exclude_topics=exclude_topics,
            min_confidence=min_confidence,
            min_score=min_score,
            text=text,
            content=content,
            include_parents=include_parents,
            flat=flat
        )

        # Use model_dump_json() to properly serialize all nested objects
        # Then convert back to dict for jsonify()
        json_str = results.model_dump_json()
        json_dict = json.loads(json_str)
        return jsonify(json_dict)

    except Exception as e:
        logger.error(f"Search error: {str(e)}")
        raise InternalServerError(f"Search operation failed: {str(e)}")


# Advanced search endpoint
@app.route('/api/search/advanced', methods=['POST'])
def advanced_search_endpoint():
    """
    Advanced search with full results including relationships.

    Request body:
    {
        "query": "search text",
        "limit": 10,
        "min_score": 0.0,
        "filter_criteria": {},
        "include_topics": ["security%", "%.policy%"],
        "exclude_topics": ["deprecated%"],
        "min_confidence": 0.7,
        "resolve_content": true,
        "include_relationships": true
    }
    """
    # Check API key if required
    auth_response = check_api_key()
    if auth_response:
        return auth_response

    try:
        # Parse request JSON
        data = request.get_json()
        if not data:
            raise BadRequest("Request body must be valid JSON")

        # Extract parameters
        query_text = data.get('query')
        if not query_text:
            raise BadRequest("'query' parameter is required")

        limit = min(data.get('limit', CONFIG['DEFAULT_RESULTS']), CONFIG['MAX_RESULTS'])
        min_score = max(data.get('min_score', CONFIG['MIN_SCORE_THRESHOLD']), 0.0)
        filter_criteria = data.get('filter_criteria', {})
        resolve_content = data.get('resolve_content', True)
        include_relationships = data.get('include_relationships', True)

        # Extract topic parameters
        include_topics, exclude_topics, min_confidence = extract_topic_parameters(data)

        # Perform advanced search
        logger.info(f"Advanced search: query='{query_text}', limit={limit}, min_score={min_score}, "
                   f"include_topics={include_topics}, exclude_topics={exclude_topics}, "
                   f"min_confidence={min_confidence}")

        results: List[SearchResult] = search_with_content(
            query_text=query_text,
            limit=limit,
            filter_criteria=filter_criteria,
            include_topics=include_topics,
            exclude_topics=exclude_topics,
            min_confidence=min_confidence,
            resolve_content=resolve_content,
            include_relationships=include_relationships,
            min_score=min_score
        )

        # Convert to serializable format
        response_data = {
            'query': query_text,
            'total_results': len(results),
            'min_score': min_score,
            'include_topics': include_topics,
            'exclude_topics': exclude_topics,
            'min_confidence': min_confidence,
            'results': []
        }

        for result in results:
            result_dict = {
                'similarity': result.similarity,
                'confidence': result.confidence,
                'topics': result.topics,
                'element_pk': result.element_pk,
                'element_id': result.element_id,
                'element_type': result.element_type,
                'content_preview': result.content_preview,
                'content_location': result.content_location,
                'doc_id': result.doc_id,
                'doc_type': result.doc_type,
                'source': result.source,
                'resolved_content': result.resolved_content,
                'resolved_text': result.resolved_text,
                'resolution_error': result.resolution_error,
                'relationship_count': result.get_relationship_count()
            }

            # Add relationship information if requested
            if include_relationships:
                result_dict['relationships'] = {
                    'by_type': result.get_relationships_by_type(),
                    'contained_elements': [
                        {
                            'relationship_type': rel.relationship_type,
                            'target_element_pk': rel.target_element_pk,
                            'target_element_type': rel.target_element_type,
                            'target_reference': rel.target_reference
                        }
                        for rel in result.get_contained_elements()
                    ],
                    'linked_elements': [
                        {
                            'relationship_type': rel.relationship_type,
                            'target_element_pk': rel.target_element_pk,
                            'target_element_type': rel.target_element_type,
                            'target_reference': rel.target_reference
                        }
                        for rel in result.get_linked_elements()
                    ],
                    'semantic_relationships': [
                        {
                            'relationship_type': rel.relationship_type,
                            'target_element_pk': rel.target_element_pk,
                            'target_element_type': rel.target_element_type,
                            'target_reference': rel.target_reference
                        }
                        for rel in result.get_semantic_relationships()
                    ]
                }

            response_data['results'].append(result_dict)

        return jsonify(response_data)

    except Exception as e:
        logger.error(f"Advanced search error: {str(e)}")
        raise InternalServerError(f"Advanced search operation failed: {str(e)}")


# Document sources endpoint
@app.route('/api/search/sources', methods=['POST'])
def document_sources_endpoint():
    """
    Get document sources from search results.

    Request body:
    {
        "query": "search text",
        "limit": 10,
        "min_score": 0.0,
        "filter_criteria": {},
        "include_topics": ["security%", "%.policy%"],
        "exclude_topics": ["deprecated%"],
        "min_confidence": 0.7
    }
    """
    # Check API key if required
    auth_response = check_api_key()
    if auth_response:
        return auth_response

    try:
        # Parse request JSON
        data = request.get_json()
        if not data:
            raise BadRequest("Request body must be valid JSON")

        # Extract parameters
        query_text = data.get('query')
        if not query_text:
            raise BadRequest("'query' parameter is required")

        limit = min(data.get('limit', CONFIG['DEFAULT_RESULTS']), CONFIG['MAX_RESULTS'])
        min_score = max(data.get('min_score', CONFIG['MIN_SCORE_THRESHOLD']), 0.0)
        filter_criteria = data.get('filter_criteria', {})

        # Extract topic parameters
        include_topics, exclude_topics, min_confidence = extract_topic_parameters(data)

        # Perform search to get results
        logger.info(f"Document sources search: query='{query_text}', limit={limit}, min_score={min_score}, "
                   f"include_topics={include_topics}, exclude_topics={exclude_topics}, "
                   f"min_confidence={min_confidence}")

        search_results = search_by_text(
            query_text=query_text,
            limit=limit,
            filter_criteria=filter_criteria,
            include_topics=include_topics,
            exclude_topics=exclude_topics,
            min_confidence=min_confidence,
            min_score=min_score
        )

        # Get document sources
        document_sources = get_document_sources(search_results)

        return jsonify({
            'query': query_text,
            'total_results': search_results.total_results,
            'include_topics': include_topics,
            'exclude_topics': exclude_topics,
            'min_confidence': min_confidence,
            'document_sources': document_sources
        })

    except Exception as e:
        logger.error(f"Document sources error: {str(e)}")
        raise InternalServerError(f"Document sources operation failed: {str(e)}")


# =============================================================================
# CONFIGURATION AND ONTOLOGY MANAGEMENT API ENDPOINTS
# =============================================================================

@app.route('/api/config', methods=['GET'])
def get_config_endpoint():
    """
    Get current configuration.
    
    Returns the current configuration as JSON.
    """
    # Check API key if required
    auth_response = check_api_key()
    if auth_response:
        return auth_response
    
    try:
        _ensure_initialized()
        
        # Get configuration, removing sensitive information
        config_dict = _config.config.copy()
        
        # Remove any sensitive fields if needed
        # (API keys, passwords, etc. - currently none in our config)
        
        return jsonify({
            'config': config_dict,
            'status': 'success'
        })
        
    except Exception as e:
        logger.error(f"Get config error: {str(e)}")
        raise InternalServerError(f"Failed to get configuration: {str(e)}")


@app.route('/api/config', methods=['PUT'])
def update_config_endpoint():
    """
    Update configuration.
    
    Request body should contain the new configuration.
    """
    # Check API key if required  
    auth_response = check_api_key()
    if auth_response:
        return auth_response
        
    try:
        # Parse request JSON
        data = request.get_json()
        if not data or 'config' not in data:
            raise BadRequest("Request body must contain 'config' field")
        
        new_config = data['config']
        
        # Validate the new configuration by creating a temporary Config object
        import tempfile
        import os
        
        with tempfile.NamedTemporaryFile(mode='w', suffix='.yaml', delete=False) as f:
            yaml.dump(new_config, f)
            temp_config_path = f.name
        
        try:
            # Try to create a Config object with the new configuration
            from go_doc_go.config import Config
            test_config = Config(temp_config_path)
            
            # If successful, save to the actual config file
            config_path = os.environ.get('GO_DOC_GO_CONFIG_PATH', 'config.yaml')
            with open(config_path, 'w') as f:
                yaml.dump(new_config, f, default_flow_style=False)
            
            # Reload the global configuration
            global _config, db, resolver
            _config = None  # Force re-initialization
            db = None
            resolver = None
            _ensure_initialized()
            
            logger.info("Configuration updated successfully")
            
            return jsonify({
                'status': 'success',
                'message': 'Configuration updated successfully'
            })
            
        finally:
            # Clean up temp file
            os.unlink(temp_config_path)
            
    except Exception as e:
        logger.error(f"Update config error: {str(e)}")
        raise InternalServerError(f"Failed to update configuration: {str(e)}")


@app.route('/api/config/validate', methods=['POST'])
def validate_config_endpoint():
    """
    Validate configuration without saving.
    
    Request body should contain the configuration to validate.
    """
    # Check API key if required
    auth_response = check_api_key()
    if auth_response:
        return auth_response
        
    try:
        # Parse request JSON
        data = request.get_json()
        if not data or 'config' not in data:
            raise BadRequest("Request body must contain 'config' field")
        
        new_config = data['config']
        
        # Validate by creating a temporary Config object
        import tempfile
        import os
        
        with tempfile.NamedTemporaryFile(mode='w', suffix='.yaml', delete=False) as f:
            yaml.dump(new_config, f)
            temp_config_path = f.name
        
        validation_errors = []
        try:
            from go_doc_go.config import Config
            test_config = Config(temp_config_path)
            
        except Exception as e:
            validation_errors.append(str(e))
        finally:
            os.unlink(temp_config_path)
        
        if validation_errors:
            return jsonify({
                'valid': False,
                'errors': validation_errors
            })
        else:
            return jsonify({
                'valid': True,
                'message': 'Configuration is valid'
            })
            
    except Exception as e:
        logger.error(f"Validate config error: {str(e)}")
        raise InternalServerError(f"Failed to validate configuration: {str(e)}")


@app.route('/api/ontologies', methods=['GET'])
def list_ontologies_endpoint():
    """
    List all available ontologies.
    
    Returns a list of ontologies with basic information.
    """
    # Check API key if required
    auth_response = check_api_key()
    if auth_response:
        return auth_response
    
    try:
        _ensure_initialized()
        
        ontology_manager = _config.get_ontology_manager()
        if not ontology_manager:
            return jsonify({
                'ontologies': [],
                'message': 'Domain detection not enabled'
            })
        
        ontologies = []
        for name in ontology_manager.loader.list_ontologies():
            ontology = ontology_manager.loader.get_ontology(name)
            if ontology:
                ontologies.append({
                    'name': ontology.name,
                    'version': ontology.version,
                    'domain': ontology.domain,
                    'description': ontology.description,
                    'active': name in ontology_manager.active_domains,
                    'terms_count': len(ontology.terms),
                    'entity_mappings_count': len(ontology.element_entity_mappings),
                    'relationship_rules_count': len(ontology.entity_relationship_rules)
                })
        
        return jsonify({
            'ontologies': ontologies,
            'total': len(ontologies),
            'active_domains': ontology_manager.active_domains
        })
        
    except Exception as e:
        logger.error(f"List ontologies error: {str(e)}")
        raise InternalServerError(f"Failed to list ontologies: {str(e)}")


@app.route('/api/ontologies/<string:name>', methods=['GET'])
def get_ontology_endpoint(name):
    """
    Get a specific ontology by name.
    
    Returns the full ontology configuration.
    """
    # Check API key if required
    auth_response = check_api_key()
    if auth_response:
        return auth_response
    
    try:
        _ensure_initialized()
        
        ontology_manager = _config.get_ontology_manager()
        if not ontology_manager:
            return jsonify({
                'error': 'Domain detection not enabled'
            }), 404
        
        ontology = ontology_manager.loader.get_ontology(name)
        if not ontology:
            return jsonify({
                'error': f'Ontology "{name}" not found'
            }), 404
        
        # Convert ontology to dictionary for JSON serialization
        ontology_dict = {
            'name': ontology.name,
            'version': ontology.version,
            'domain': ontology.domain,
            'description': ontology.description,
            'document_types': ontology.document_types,
            'terms': [
                {
                    'id': term.id,
                    'name': term.name,
                    'description': term.description,
                    'synonyms': term.synonyms,
                    'category': term.category,
                    'attributes': term.attributes
                }
                for term in ontology.terms
            ],
            'element_entity_mappings': [
                {
                    'entity_type': mapping.entity_type,
                    'description': mapping.description,
                    'document_types': mapping.document_types,
                    'element_types': mapping.element_types,
                    'extraction_rules': [
                        {
                            'type': rule.type,
                            'pattern': rule.pattern,
                            'field_path': rule.field_path,
                            'confidence': rule.confidence,
                            'required': rule.required,
                            'description': rule.description
                        }
                        for rule in mapping.extraction_rules
                    ]
                }
                for mapping in ontology.element_entity_mappings
            ],
            'entity_relationship_rules': [
                {
                    'name': rule.name,
                    'description': rule.description,
                    'source_entity_type': rule.source_entity_type,
                    'relationship_type': rule.relationship_type,
                    'target_entity_type': rule.target_entity_type,
                    'confidence': rule.confidence,
                    'matching_criteria': {
                        'same_source_element': rule.matching_criteria.same_source_element,
                        'metadata_match': rule.matching_criteria.metadata_match,
                        'content_proximity': rule.matching_criteria.content_proximity
                    }
                }
                for rule in ontology.entity_relationship_rules
            ]
        }
        
        return jsonify({
            'ontology': ontology_dict,
            'active': name in ontology_manager.active_domains
        })
        
    except Exception as e:
        logger.error(f"Get ontology error: {str(e)}")
        raise InternalServerError(f"Failed to get ontology: {str(e)}")


@app.route('/api/ontologies/<string:name>', methods=['PUT'])
def update_ontology_endpoint(name):
    """
    Update a specific ontology.
    
    Request body should contain the ontology configuration.
    """
    # Check API key if required
    auth_response = check_api_key()
    if auth_response:
        return auth_response
    
    try:
        # Parse request JSON
        data = request.get_json()
        if not data or 'ontology' not in data:
            raise BadRequest("Request body must contain 'ontology' field")
        
        ontology_data = data['ontology']
        
        # Validate ontology name matches URL parameter
        if ontology_data.get('name') != name:
            raise BadRequest("Ontology name in body must match URL parameter")
        
        # Find the ontology file path (this is a simplification)
        import os
        from pathlib import Path
        
        # Look in common ontology directories
        possible_paths = [
            Path('examples/ontologies') / f'{name}.yaml',
            Path('ontologies') / f'{name}.yaml',
            Path('.') / f'{name}.yaml'
        ]
        
        ontology_path = None
        for path in possible_paths:
            if path.exists():
                ontology_path = str(path)
                break
        
        if not ontology_path:
            # Create new ontology file in examples/ontologies
            ontology_path = f'examples/ontologies/{name}.yaml'
            os.makedirs(os.path.dirname(ontology_path), exist_ok=True)
        
        # Validate the ontology by loading it
        try:
            from go_doc_go.domain import OntologyLoader
            loader = OntologyLoader()
            test_ontology = loader.load_from_dict(ontology_data)
            
            # If validation passes, save the file
            with open(ontology_path, 'w') as f:
                yaml.dump(ontology_data, f, default_flow_style=False)
            
            # Reload in the ontology manager
            _ensure_initialized()
            ontology_manager = _config.get_ontology_manager()
            if ontology_manager:
                # Clear and reload
                ontology_manager.loader.clear()
                ontology_manager.active_domains.clear()
                
                # Reload all ontologies from config
                domain_config = _config.config.get("relationship_detection", {}).get("domain", {})
                ontologies = domain_config.get("ontologies", [])
                for ontology_config in ontologies:
                    if isinstance(ontology_config, dict):
                        path = ontology_config.get("path")
                        active = ontology_config.get("active", True)
                        
                        if path and os.path.exists(path):
                            try:
                                loaded_name = ontology_manager.load_ontology(path)
                                if active:
                                    ontology_manager.activate_domain(loaded_name)
                            except Exception as e:
                                logger.error(f"Failed to reload ontology from {path}: {e}")
            
            logger.info(f"Ontology '{name}' updated successfully")
            
            return jsonify({
                'status': 'success',
                'message': f'Ontology "{name}" updated successfully'
            })
            
        except Exception as e:
            return jsonify({
                'status': 'error',
                'message': f'Invalid ontology: {str(e)}'
            }), 400
            
    except Exception as e:
        logger.error(f"Update ontology error: {str(e)}")
        raise InternalServerError(f"Failed to update ontology: {str(e)}")


@app.route('/api/domain/active', methods=['GET'])
def get_active_domains_endpoint():
    """
    Get active domains.
    
    Returns list of currently active domains.
    """
    # Check API key if required
    auth_response = check_api_key()
    if auth_response:
        return auth_response
    
    try:
        _ensure_initialized()
        
        ontology_manager = _config.get_ontology_manager()
        if not ontology_manager:
            return jsonify({
                'active_domains': [],
                'message': 'Domain detection not enabled'
            })
        
        return jsonify({
            'active_domains': ontology_manager.active_domains
        })
        
    except Exception as e:
        logger.error(f"Get active domains error: {str(e)}")
        raise InternalServerError(f"Failed to get active domains: {str(e)}")


@app.route('/api/domain/<string:name>/activate', methods=['POST'])
def activate_domain_endpoint(name):
    """
    Activate a domain.
    
    Makes the specified domain active.
    """
    # Check API key if required
    auth_response = check_api_key()
    if auth_response:
        return auth_response
    
    try:
        _ensure_initialized()
        
        ontology_manager = _config.get_ontology_manager()
        if not ontology_manager:
            return jsonify({
                'error': 'Domain detection not enabled'
            }), 400
        
        # Check if domain exists
        if name not in ontology_manager.loader.ontologies:
            return jsonify({
                'error': f'Domain "{name}" not found'
            }), 404
        
        ontology_manager.activate_domain(name)
        
        return jsonify({
            'status': 'success',
            'message': f'Domain "{name}" activated',
            'active_domains': ontology_manager.active_domains
        })
        
    except Exception as e:
        logger.error(f"Activate domain error: {str(e)}")
        raise InternalServerError(f"Failed to activate domain: {str(e)}")


@app.route('/api/domain/<string:name>/deactivate', methods=['POST'])
def deactivate_domain_endpoint(name):
    """
    Deactivate a domain.
    
    Makes the specified domain inactive.
    """
    # Check API key if required
    auth_response = check_api_key()
    if auth_response:
        return auth_response
    
    try:
        _ensure_initialized()
        
        ontology_manager = _config.get_ontology_manager()
        if not ontology_manager:
            return jsonify({
                'error': 'Domain detection not enabled'
            }), 400
        
        ontology_manager.deactivate_domain(name)
        
        return jsonify({
            'status': 'success', 
            'message': f'Domain "{name}" deactivated',
            'active_domains': ontology_manager.active_domains
        })
        
    except Exception as e:
        logger.error(f"Deactivate domain error: {str(e)}")
        raise InternalServerError(f"Failed to deactivate domain: {str(e)}")


# Static file serving for React frontend
# Get the path to the frontend build directory
FRONTEND_BUILD_DIR = os.path.join(os.path.dirname(os.path.dirname(os.path.dirname(__file__))), 'frontend', 'dist')

@app.route('/')
def serve_index():
    """Serve the React app index.html for the root route."""
    if os.path.exists(os.path.join(FRONTEND_BUILD_DIR, 'index.html')):
        return send_file(os.path.join(FRONTEND_BUILD_DIR, 'index.html'))
    else:
        return jsonify({
            "message": "React frontend not built. Run 'npm run build' in the frontend directory.",
            "api_docs": f"http://{CONFIG['HOST']}:{CONFIG['PORT']}{CONFIG['SWAGGER_UI_PATH']}"
        }), 404

@app.route('/static/<path:filename>')
def serve_static(filename):
    """Serve static files from the React build."""
    return send_from_directory(os.path.join(FRONTEND_BUILD_DIR, 'static'), filename)

@app.route('/assets/<path:filename>')
def serve_assets(filename):
    """Serve asset files from the React build."""
    return send_from_directory(os.path.join(FRONTEND_BUILD_DIR, 'assets'), filename)

# Catch-all route for React Router (must be after API routes)
@app.route('/<path:path>')
def serve_react_app(path):
    """
    Catch-all route to serve the React app for client-side routing.
    This should handle all frontend routes like /config, /ontologies, etc.
    """
    # Don't intercept API routes
    if path.startswith('api/'):
        return jsonify({"error": "API endpoint not found"}), 404
    
    # Serve index.html for all other routes (React Router will handle them)
    if os.path.exists(os.path.join(FRONTEND_BUILD_DIR, 'index.html')):
        return send_file(os.path.join(FRONTEND_BUILD_DIR, 'index.html'))
    else:
        return jsonify({
            "message": "React frontend not built. Run 'npm run build' in the frontend directory.",
            "available_apis": {
                "search": f"/api/search",
                "config": f"/api/config", 
                "ontologies": f"/api/ontologies",
                "domains": f"/api/domain"
            }
        }), 404


# Optional: Rate limiting if using Flask-Limiter
try:
    from flask_limiter import Limiter
    from flask_limiter.util import get_remote_address

    # Initialize Limiter with first argument as the key_func (not a parameter name)
    limiter = Limiter(
        get_remote_address,  # First argument is key_func (no parameter name)
        app=app,  # Pass app as a keyword argument
        default_limits=[CONFIG['RATE_LIMIT']],
        storage_uri="memory://",
        strategy="fixed-window"
    )

    # Apply rate limiting to search endpoints
    limiter.limit(CONFIG['RATE_LIMIT'])(search_endpoint)
    limiter.limit(CONFIG['RATE_LIMIT'])(advanced_search_endpoint)
    limiter.limit(CONFIG['RATE_LIMIT'])(structured_search_endpoint)
    limiter.limit(CONFIG['RATE_LIMIT'])(simple_structured_search_endpoint)
    limiter.limit(CONFIG['RATE_LIMIT'])(document_sources_endpoint)
    
    # Apply rate limiting to config/ontology endpoints
    limiter.limit(CONFIG['RATE_LIMIT'])(get_config_endpoint)
    limiter.limit(CONFIG['RATE_LIMIT'])(update_config_endpoint)
    limiter.limit(CONFIG['RATE_LIMIT'])(validate_config_endpoint)
    limiter.limit(CONFIG['RATE_LIMIT'])(list_ontologies_endpoint)
    limiter.limit(CONFIG['RATE_LIMIT'])(get_ontology_endpoint)
    limiter.limit(CONFIG['RATE_LIMIT'])(update_ontology_endpoint)

    logger.info(f"Rate limiting enabled: {CONFIG['RATE_LIMIT']}")
except ImportError:
    logger.warning("Flask-Limiter not installed, rate limiting disabled")


# Startup message
def print_startup_info():
    """Print startup information."""
    logger.info("=" * 50)
    logger.info("Document Search API Server Starting")
    logger.info("=" * 50)
    logger.info(f"Server URL: http://{CONFIG['HOST']}:{CONFIG['PORT']}")
    logger.info(f"API Documentation: http://{CONFIG['HOST']}:{CONFIG['PORT']}{CONFIG['SWAGGER_UI_PATH']}")
    logger.info(f"OpenAPI Spec: http://{CONFIG['HOST']}:{CONFIG['PORT']}{CONFIG['API_SPEC_PATH']}")
    logger.info(f"Debug Mode: {CONFIG['DEBUG']}")
    logger.info(f"Authentication: {'Enabled' if CONFIG['API_KEY'] else 'Disabled'}")
    logger.info(f"Rate Limiting: {CONFIG['RATE_LIMIT']}")
    logger.info("Available Endpoints:")
    logger.info("  Search Endpoints:")
    logger.info("    /api/search - Basic search")
    logger.info("    /api/search/advanced - Advanced search with relationships")
    logger.info("    /api/search/structured - Structured search with complex criteria")
    logger.info("    /api/search/structured/simple - Simple structured search")
    logger.info("    /api/search/sources - Document sources")
    logger.info("  Configuration Management:")
    logger.info("    GET /api/config - Get current configuration")
    logger.info("    PUT /api/config - Update configuration")
    logger.info("    POST /api/config/validate - Validate configuration")
    logger.info("  Ontology Management:")
    logger.info("    GET /api/ontologies - List all ontologies")
    logger.info("    GET /api/ontologies/<name> - Get specific ontology")
    logger.info("    PUT /api/ontologies/<name> - Update ontology")
    logger.info("  Domain Management:")
    logger.info("    GET /api/domain/active - Get active domains")
    logger.info("    POST /api/domain/<name>/activate - Activate domain")
    logger.info("    POST /api/domain/<name>/deactivate - Deactivate domain")
    logger.info("=" * 50)


# Main entry point
if __name__ == '__main__':
    print_startup_info()
    app.run(
        host=CONFIG['HOST'],
        port=CONFIG['PORT'],
        debug=CONFIG['DEBUG'],
        threaded=True
    )
