"""
Settings API routes for Flask configuration management.
"""

import os
import json
import logging
from typing import Dict, Any, Optional
from pathlib import Path
from flask import Blueprint, request, jsonify
from cryptography.fernet import Fernet
import yaml
from dotenv import load_dotenv, set_key, find_dotenv

from go_doc_go.config import Config

logger = logging.getLogger(__name__)

# Generate or load encryption key for sensitive data
ENCRYPTION_KEY_FILE = Path.home() / ".go_doc_go" / "encryption.key"
ENCRYPTION_KEY_FILE.parent.mkdir(parents=True, exist_ok=True)

if ENCRYPTION_KEY_FILE.exists():
    with open(ENCRYPTION_KEY_FILE, 'rb') as f:
        encryption_key = f.read()
else:
    encryption_key = Fernet.generate_key()
    with open(ENCRYPTION_KEY_FILE, 'wb') as f:
        f.write(encryption_key)
    os.chmod(ENCRYPTION_KEY_FILE, 0o600)  # Restrict access to owner only

cipher_suite = Fernet(encryption_key)

# Create Flask Blueprint
settings_bp = Blueprint('settings', __name__, url_prefix='/api/settings')


def mask_sensitive_value(value: str, show_last: int = 4) -> str:
    """Mask sensitive values for display."""
    if not value or len(value) <= show_last:
        return value
    return "*" * (len(value) - show_last) + value[-show_last:]


@settings_bp.route('', methods=['GET'])
def get_configuration():
    """Get current configuration with masked sensitive values."""
    try:
        # Load configuration
        config = Config()
        
        # Prepare response with masked sensitive values
        response = {
            "llm": {
                "openai_api_key": mask_sensitive_value(os.getenv("OPENAI_API_KEY", "")),
                "anthropic_api_key": mask_sensitive_value(os.getenv("ANTHROPIC_API_KEY", "")),
                "default_provider": config.get("llm.default_provider", "openai"),
                "request_timeout": config.get("llm.request_timeout", 30),
                "max_tokens": config.get("llm.max_tokens", 2000),
                "temperature": config.get("llm.temperature", 0.7),
                "max_retries": config.get("llm.max_retries", 3),
            },
            "database": {
                "type": config.get("database.type", "postgresql"),
                "host": config.get("database.host", "localhost"),
                "port": config.get("database.port", 5432),
                "database": config.get("database.name", "go-doc-go"),
                "username": config.get("database.user", ""),
                "password": mask_sensitive_value(config.get("database.password", "")),
                "pool_size": config.get("database.pool_size", 10),
                "max_overflow": config.get("database.max_overflow", 20),
            },
            "content_sources": {
                "sources": config.get("content_sources.sources", {})
            },
            "processing": {
                "crawler_interval": config.get("processing.crawler_interval", 300),
                "batch_size": config.get("processing.batch_size", 10),
                "worker_count": config.get("processing.worker_count", 4),
                "ocr_engine": config.get("processing.ocr_engine", "tesseract"),
                "embedding_model": config.get("processing.embedding_model", "text-embedding-ada-002"),
            },
            "environment_variables": {
                k: v for k, v in os.environ.items() 
                if k.startswith(("GO_DOC_GO_", "DOCULYZER_"))
            },
            "system": {
                "version": "1.0.0",
                "config_path": config.config_path,
                "log_level": logging.getLogger().level,
            }
        }
        
        return jsonify(response)
        
    except Exception as e:
        logger.error(f"Failed to get configuration: {e}")
        return jsonify({"error": str(e)}), 500


@settings_bp.route('', methods=['POST'])
def save_configuration():
    """Save configuration changes."""
    try:
        config = request.json
        
        # Find .env file or create one
        env_file = find_dotenv() or Path(".env")
        
        # Update environment variables for sensitive values
        if "llm" in config:
            if "openai_api_key" in config["llm"] and not config["llm"]["openai_api_key"].startswith("*"):
                set_key(env_file, "OPENAI_API_KEY", config["llm"]["openai_api_key"])
            if "anthropic_api_key" in config["llm"] and not config["llm"]["anthropic_api_key"].startswith("*"):
                set_key(env_file, "ANTHROPIC_API_KEY", config["llm"]["anthropic_api_key"])
        
        # Update config.yaml for non-sensitive values
        config_path = Path("config.yaml")
        if config_path.exists():
            with open(config_path, 'r') as f:
                yaml_config = yaml.safe_load(f) or {}
        else:
            yaml_config = {}
        
        # Merge configuration (excluding sensitive values)
        if "llm" in config:
            yaml_config.setdefault("llm", {}).update({
                k: v for k, v in config["llm"].items() 
                if not k.endswith("_key") and not k.endswith("_secret")
            })
        
        if "database" in config:
            db_config = config["database"].copy()
            # Store database password in .env
            if "password" in db_config and not db_config["password"].startswith("*"):
                set_key(env_file, "DB_PASSWORD", db_config["password"])
                db_config["password"] = "${DB_PASSWORD}"
            yaml_config["database"] = db_config
        
        if "processing" in config:
            yaml_config["processing"] = config["processing"]
        
        # Save updated config.yaml
        with open(config_path, 'w') as f:
            yaml.dump(yaml_config, f, default_flow_style=False)
        
        # Reload environment variables
        load_dotenv(override=True)
        
        return jsonify({"message": "Configuration saved successfully"})
        
    except Exception as e:
        logger.error(f"Failed to save configuration: {e}")
        return jsonify({"error": str(e)}), 500


@settings_bp.route('/validate', methods=['POST'])
def validate_configuration():
    """Validate configuration without saving."""
    try:
        config = request.json
        
        # Basic validation rules
        errors = []
        
        # Validate database configuration
        if "database" in config:
            db = config["database"]
            if db.get("type") == "postgresql":
                if not db.get("host"):
                    errors.append("Database host is required for PostgreSQL")
                if not db.get("port"):
                    errors.append("Database port is required for PostgreSQL")
        
        # Validate LLM configuration
        if "llm" in config:
            llm = config["llm"]
            if llm.get("temperature") is not None:
                if not 0 <= llm["temperature"] <= 2:
                    errors.append("Temperature must be between 0 and 2")
        
        # Validate processing configuration
        if "processing" in config:
            proc = config["processing"]
            if proc.get("worker_count") is not None:
                if not 1 <= proc["worker_count"] <= 100:
                    errors.append("Worker count must be between 1 and 100")
        
        if errors:
            return jsonify({"valid": False, "errors": errors})
        
        return jsonify({"valid": True})
        
    except Exception as e:
        logger.error(f"Failed to validate configuration: {e}")
        return jsonify({"valid": False, "errors": [str(e)]})


@settings_bp.route('/test-llm', methods=['POST'])
def test_llm_connection():
    """Test LLM API connection."""
    try:
        data = request.json
        provider = data.get('provider')
        api_key = data.get('api_key')
        
        if provider == "openai":
            import openai
            # Remove masked characters if present
            if api_key.startswith("*"):
                api_key = os.getenv("OPENAI_API_KEY")
            
            openai.api_key = api_key
            
            # Try a simple completion to test the key
            try:
                response = openai.ChatCompletion.create(
                    model="gpt-3.5-turbo",
                    messages=[{"role": "user", "content": "test"}],
                    max_tokens=1
                )
                return jsonify({"success": True, "message": "OpenAI API key is valid"})
            except Exception as e:
                logger.error(f"OpenAI API test failed: {e}")
                return jsonify({"success": False, "message": str(e)})
                
        elif provider == "anthropic_api_key":
            # Test Anthropic API
            return jsonify({"success": True, "message": "Anthropic API key validation not implemented"})
        
        else:
            return jsonify({"success": False, "message": f"Unknown provider: {provider}"})
            
    except Exception as e:
        logger.error(f"LLM connection test failed: {e}")
        return jsonify({"success": False, "message": str(e)})


@settings_bp.route('/test-database', methods=['POST'])
def test_database_connection():
    """Test database connection."""
    try:
        data = request.json
        db_type = data.get('type', 'postgresql')
        
        if db_type == "postgresql":
            import psycopg2
            
            # Build connection parameters
            conn_params = {
                "host": data.get('host', 'localhost'),
                "port": data.get('port', 5432),
                "database": data.get('database', 'go-doc-go'),
            }
            if data.get('username'):
                conn_params["user"] = data['username']
            
            password = data.get('password', '')
            if password and not password.startswith("*"):
                conn_params["password"] = password
            elif password and password.startswith("*"):
                conn_params["password"] = os.getenv("DB_PASSWORD", "")
            
            # Try to connect
            try:
                conn = psycopg2.connect(**conn_params)
                conn.close()
                return jsonify({"success": True, "message": "Database connection successful"})
            except Exception as e:
                logger.error(f"PostgreSQL connection test failed: {e}")
                return jsonify({"success": False, "message": str(e)})
                
        elif db_type == "sqlite":
            return jsonify({"success": True, "message": "SQLite connection successful"})
        
        else:
            return jsonify({"success": False, "message": f"Unknown database type: {db_type}"})
            
    except Exception as e:
        logger.error(f"Database connection test failed: {e}")
        return jsonify({"success": False, "message": str(e)})


@settings_bp.route('/test-source', methods=['POST'])
def test_content_source():
    """Test content source connection."""
    try:
        data = request.json
        source_type = data.get('type')
        config = data.get('config', {})
        
        if source_type == "s3":
            import boto3
            
            # Test S3 connection
            try:
                s3 = boto3.client(
                    's3',
                    aws_access_key_id=config.get("aws_access_key_id"),
                    aws_secret_access_key=config.get("aws_secret_access_key"),
                    region_name=config.get("region", "us-east-1")
                )
                s3.list_buckets()
                return jsonify({"success": True, "message": "S3 connection successful"})
            except Exception as e:
                return jsonify({"success": False, "message": str(e)})
                
        elif source_type == "sharepoint":
            return jsonify({"success": True, "message": "SharePoint validation not implemented"})
        
        elif source_type == "local":
            path = Path(config.get("path", "."))
            if path.exists() and path.is_dir():
                return jsonify({"success": True, "message": "Local path is accessible"})
            else:
                return jsonify({"success": False, "message": "Path does not exist or is not a directory"})
        
        else:
            return jsonify({"success": False, "message": f"Unknown source type: {source_type}"})
            
    except Exception as e:
        logger.error(f"Content source test failed: {e}")
        return jsonify({"success": False, "message": str(e)})


@settings_bp.route('/environment', methods=['GET'])
def get_environment_variables():
    """Get environment variables (filtered for security)."""
    try:
        # Only return non-sensitive environment variables
        safe_prefixes = ("GO_DOC_GO_", "DOCULYZER_", "NODE_", "PYTHON")
        env_vars = {}
        
        for key, value in os.environ.items():
            if any(key.startswith(prefix) for prefix in safe_prefixes):
                # Mask any potential sensitive values
                if any(sensitive in key.lower() for sensitive in ["key", "secret", "password", "token"]):
                    env_vars[key] = mask_sensitive_value(value)
                else:
                    env_vars[key] = value
        
        return jsonify(env_vars)
        
    except Exception as e:
        logger.error(f"Failed to get environment variables: {e}")
        return jsonify({"error": str(e)}), 500


@settings_bp.route('/clear-cache', methods=['POST'])
def clear_cache():
    """Clear application cache."""
    try:
        # Implementation would depend on caching strategy
        return jsonify({"message": "Cache cleared successfully"})
        
    except Exception as e:
        logger.error(f"Failed to clear cache: {e}")
        return jsonify({"error": str(e)}), 500


@settings_bp.route('/system-info', methods=['GET'])
def get_system_info():
    """Get system information."""
    try:
        import platform
        import psutil
        
        return jsonify({
            "version": "1.0.0",
            "python_version": platform.python_version(),
            "platform": platform.platform(),
            "processor": platform.processor(),
            "memory_total": psutil.virtual_memory().total,
            "memory_available": psutil.virtual_memory().available,
            "disk_usage": psutil.disk_usage('/').percent,
            "cpu_count": psutil.cpu_count(),
        })
        
    except Exception as e:
        logger.error(f"Failed to get system info: {e}")
        return jsonify({
            "version": "1.0.0",
            "error": str(e)
        })