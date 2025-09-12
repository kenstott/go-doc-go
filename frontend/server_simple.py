#!/usr/bin/env python3
"""Simple Flask server for serving the React UI and API endpoints."""

import os
import sys
import json
import yaml
from pathlib import Path
from flask import Flask, request, jsonify, send_from_directory
from flask_cors import CORS

# Add src to path for imports
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..', 'src'))

app = Flask(__name__, static_folder='dist', static_url_path='')
CORS(app, resources={r"/api/*": {"origins": "*"}})

# Paths
CONFIG_DIR = Path(__file__).parent.parent / 'config'
ONTOLOGY_DIR = Path(__file__).parent.parent / 'examples' / 'ontologies'

# Serve React app
@app.route('/')
def serve():
    return send_from_directory(app.static_folder, 'index.html')

@app.route('/<path:path>')
def serve_static(path):
    if path and os.path.exists(os.path.join(app.static_folder, path)):
        return send_from_directory(app.static_folder, path)
    else:
        return send_from_directory(app.static_folder, 'index.html')

# API endpoints
@app.route('/api/info', methods=['GET'])
def api_info():
    """Get API information."""
    return jsonify({
        'name': 'Document Search API',
        'version': '1.0.0',
        'status': 'running',
        'links': {
            'api_documentation': '/docs',
            'api_info': '/api/info',
            'health': '/health',
            'openapi_spec': '/api/spec'
        }
    })

@app.route('/api/config', methods=['GET'])
def get_config():
    """Get the current configuration."""
    config_file = CONFIG_DIR / 'config.yaml'
    if config_file.exists():
        with open(config_file, 'r') as f:
            config = yaml.safe_load(f)
        return jsonify(config)
    return jsonify({'error': 'Configuration not found'}), 404

@app.route('/api/config', methods=['POST'])
def update_config():
    """Update the configuration."""
    try:
        config = request.json
        config_file = CONFIG_DIR / 'config.yaml'
        
        # Ensure config directory exists
        CONFIG_DIR.mkdir(parents=True, exist_ok=True)
        
        # Save configuration
        with open(config_file, 'w') as f:
            yaml.dump(config, f, default_flow_style=False, sort_keys=False)
        
        return jsonify({'message': 'Configuration updated successfully'}), 200
    except Exception as e:
        return jsonify({'error': str(e)}), 500

@app.route('/api/ontologies', methods=['GET'])
def list_ontologies():
    """List all available ontologies."""
    ontologies = []
    if ONTOLOGY_DIR.exists():
        for file in ONTOLOGY_DIR.glob('*.yaml'):
            with open(file, 'r') as f:
                data = yaml.safe_load(f)
                if data and 'ontology' in data:
                    ontology = data['ontology']
                    ontologies.append({
                        'name': ontology.get('name', file.stem),
                        'file': file.name,
                        'version': ontology.get('version', '1.0.0'),
                        'description': ontology.get('description', ''),
                        'path': str(file)
                    })
    return jsonify(ontologies)

@app.route('/api/ontologies/<name>', methods=['GET'])
def get_ontology(name):
    """Get a specific ontology."""
    ontology_file = ONTOLOGY_DIR / f'{name}.yaml'
    if ontology_file.exists():
        with open(ontology_file, 'r') as f:
            data = yaml.safe_load(f)
        return jsonify(data)
    return jsonify({'error': 'Ontology not found'}), 404

@app.route('/api/ontologies/<name>', methods=['PUT'])
def update_ontology(name):
    """Update an existing ontology."""
    try:
        data = request.json
        ontology_file = ONTOLOGY_DIR / f'{name}.yaml'
        
        # Ensure ontology directory exists
        ONTOLOGY_DIR.mkdir(parents=True, exist_ok=True)
        
        # Save ontology
        with open(ontology_file, 'w') as f:
            yaml.dump(data, f, default_flow_style=False, sort_keys=False)
        
        return jsonify({'message': 'Ontology updated successfully'}), 200
    except Exception as e:
        return jsonify({'error': str(e)}), 500

@app.route('/api/ontologies', methods=['POST'])
def create_ontology():
    """Create a new ontology."""
    try:
        data = request.json
        name = data.get('ontology', {}).get('name')
        if not name:
            return jsonify({'error': 'Ontology name is required'}), 400
        
        ontology_file = ONTOLOGY_DIR / f'{name}.yaml'
        if ontology_file.exists():
            return jsonify({'error': 'Ontology already exists'}), 409
        
        # Ensure ontology directory exists
        ONTOLOGY_DIR.mkdir(parents=True, exist_ok=True)
        
        # Save new ontology
        with open(ontology_file, 'w') as f:
            yaml.dump(data, f, default_flow_style=False, sort_keys=False)
        
        return jsonify({'message': 'Ontology created successfully', 'name': name}), 201
    except Exception as e:
        return jsonify({'error': str(e)}), 500

@app.route('/api/ontologies/<name>', methods=['DELETE'])
def delete_ontology(name):
    """Delete an ontology."""
    try:
        ontology_file = ONTOLOGY_DIR / f'{name}.yaml'
        if ontology_file.exists():
            ontology_file.unlink()
            return jsonify({'message': 'Ontology deleted successfully'}), 200
        return jsonify({'error': 'Ontology not found'}), 404
    except Exception as e:
        return jsonify({'error': str(e)}), 500

@app.route('/api/domains', methods=['GET'])
def get_domains():
    """Get list of domains and their activation status."""
    # For now, return mock data since we can't import the domain manager
    domains = []
    if ONTOLOGY_DIR.exists():
        for file in ONTOLOGY_DIR.glob('*.yaml'):
            with open(file, 'r') as f:
                data = yaml.safe_load(f)
                if data and 'ontology' in data:
                    ontology = data['ontology']
                    domains.append({
                        'name': ontology.get('name', file.stem),
                        'description': ontology.get('description', ''),
                        'active': False  # Would need domain manager to check actual status
                    })
    return jsonify(domains)

@app.route('/api/domains/<name>/activate', methods=['POST'])
def activate_domain(name):
    """Activate a domain."""
    # Mock implementation
    return jsonify({'message': f'Domain {name} activated (mock)'}), 200

@app.route('/api/domains/<name>/deactivate', methods=['POST'])
def deactivate_domain(name):
    """Deactivate a domain."""
    # Mock implementation
    return jsonify({'message': f'Domain {name} deactivated (mock)'}), 200

@app.route('/health', methods=['GET'])
def health():
    """Health check endpoint."""
    return jsonify({'status': 'healthy'}), 200

if __name__ == '__main__':
    port = int(os.environ.get('SERVER_PORT', 5001))
    app.run(host='0.0.0.0', port=port, debug=True)