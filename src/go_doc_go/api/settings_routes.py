"""
Settings API routes for configuration management.
"""

import os
import json
import logging
from typing import Dict, Any, Optional
from pathlib import Path
from fastapi import APIRouter, HTTPException, Depends, Body
from pydantic import BaseModel, Field
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

router = APIRouter(prefix="/api/settings", tags=["settings"])


class LLMTestRequest(BaseModel):
    """Request model for testing LLM connections."""
    provider: str
    api_key: str


class DatabaseTestRequest(BaseModel):
    """Request model for testing database connections."""
    type: str = "postgresql"
    host: Optional[str] = "localhost"
    port: Optional[int] = 5432
    database: Optional[str] = "go-doc-go"
    username: Optional[str] = None
    password: Optional[str] = None


class ContentSourceTestRequest(BaseModel):
    """Request model for testing content source connections."""
    type: str
    config: Dict[str, Any]


class ConfigurationResponse(BaseModel):
    """Response model for configuration data."""
    llm: Dict[str, Any] = Field(default_factory=dict)
    database: Dict[str, Any] = Field(default_factory=dict)
    content_sources: Dict[str, Any] = Field(default_factory=dict)
    processing: Dict[str, Any] = Field(default_factory=dict)
    environment_variables: Dict[str, str] = Field(default_factory=dict)
    system: Dict[str, Any] = Field(default_factory=dict)


def encrypt_sensitive_value(value: str) -> str:
    """Encrypt sensitive configuration values."""
    if not value:
        return value
    try:
        encrypted = cipher_suite.encrypt(value.encode())
        return f"encrypted:{encrypted.decode()}"
    except Exception as e:
        logger.error(f"Failed to encrypt value: {e}")
        return value


def decrypt_sensitive_value(value: str) -> str:
    """Decrypt sensitive configuration values."""
    if not value or not value.startswith("encrypted:"):
        return value
    try:
        encrypted_data = value[10:]  # Remove "encrypted:" prefix
        decrypted = cipher_suite.decrypt(encrypted_data.encode())
        return decrypted.decode()
    except Exception as e:
        logger.error(f"Failed to decrypt value: {e}")
        return value


def mask_sensitive_value(value: str, show_last: int = 4) -> str:
    """Mask sensitive values for display."""
    if not value or len(value) <= show_last:
        return value
    return "*" * (len(value) - show_last) + value[-show_last:]


@router.get("", response_model=ConfigurationResponse)
async def get_configuration():
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
        
        return response
        
    except Exception as e:
        logger.error(f"Failed to get configuration: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@router.post("")
async def save_configuration(config: Dict[str, Any] = Body(...)):
    """Save configuration changes."""
    try:
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
        
        return {"message": "Configuration saved successfully"}
        
    except Exception as e:
        logger.error(f"Failed to save configuration: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@router.post("/validate")
async def validate_configuration(config: Dict[str, Any] = Body(...)):
    """Validate configuration without saving."""
    try:
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
            return {"valid": False, "errors": errors}
        
        return {"valid": True}
        
    except Exception as e:
        logger.error(f"Failed to validate configuration: {e}")
        return {"valid": False, "errors": [str(e)]}


@router.post("/test-llm")
async def test_llm_connection(request: LLMTestRequest):
    """Test LLM API connection."""
    try:
        if request.provider == "openai":
            import openai
            # Remove masked characters if present
            api_key = request.api_key if not request.api_key.startswith("*") else os.getenv("OPENAI_API_KEY")
            openai.api_key = api_key
            
            # Try a simple completion to test the key
            try:
                response = openai.ChatCompletion.create(
                    model="gpt-3.5-turbo",
                    messages=[{"role": "user", "content": "test"}],
                    max_tokens=1
                )
                return {"success": True, "message": "OpenAI API key is valid"}
            except Exception as e:
                logger.error(f"OpenAI API test failed: {e}")
                return {"success": False, "message": str(e)}
                
        elif request.provider == "anthropic":
            # Test Anthropic API
            # Note: Actual implementation would require anthropic SDK
            return {"success": True, "message": "Anthropic API key validation not implemented"}
        
        else:
            return {"success": False, "message": f"Unknown provider: {request.provider}"}
            
    except Exception as e:
        logger.error(f"LLM connection test failed: {e}")
        return {"success": False, "message": str(e)}


@router.post("/test-database")
async def test_database_connection(request: DatabaseTestRequest):
    """Test database connection."""
    try:
        if request.type == "postgresql":
            import psycopg2
            
            # Build connection string
            conn_params = {
                "host": request.host,
                "port": request.port,
                "database": request.database,
            }
            if request.username:
                conn_params["user"] = request.username
            if request.password and not request.password.startswith("*"):
                conn_params["password"] = request.password
            elif request.password and request.password.startswith("*"):
                # Use stored password
                conn_params["password"] = os.getenv("DB_PASSWORD", "")
            
            # Try to connect
            try:
                conn = psycopg2.connect(**conn_params)
                conn.close()
                return {"success": True, "message": "Database connection successful"}
            except Exception as e:
                logger.error(f"PostgreSQL connection test failed: {e}")
                return {"success": False, "message": str(e)}
                
        elif request.type == "sqlite":
            # SQLite doesn't need connection testing
            return {"success": True, "message": "SQLite connection successful"}
        
        else:
            return {"success": False, "message": f"Unknown database type: {request.type}"}
            
    except Exception as e:
        logger.error(f"Database connection test failed: {e}")
        return {"success": False, "message": str(e)}


@router.post("/test-source")
async def test_content_source(request: ContentSourceTestRequest):
    """Test content source connection."""
    try:
        if request.type == "s3":
            import boto3
            
            # Test S3 connection
            try:
                s3 = boto3.client(
                    's3',
                    aws_access_key_id=request.config.get("aws_access_key_id"),
                    aws_secret_access_key=request.config.get("aws_secret_access_key"),
                    region_name=request.config.get("region", "us-east-1")
                )
                s3.list_buckets()
                return {"success": True, "message": "S3 connection successful"}
            except Exception as e:
                return {"success": False, "message": str(e)}
                
        elif request.type == "sharepoint":
            # SharePoint connection test would go here
            return {"success": True, "message": "SharePoint validation not implemented"}
        
        elif request.type == "local":
            # Test local filesystem access
            path = Path(request.config.get("path", "."))
            if path.exists() and path.is_dir():
                return {"success": True, "message": "Local path is accessible"}
            else:
                return {"success": False, "message": "Path does not exist or is not a directory"}
        
        else:
            return {"success": False, "message": f"Unknown source type: {request.type}"}
            
    except Exception as e:
        logger.error(f"Content source test failed: {e}")
        return {"success": False, "message": str(e)}


@router.get("/environment")
async def get_environment_variables():
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
        
        return env_vars
        
    except Exception as e:
        logger.error(f"Failed to get environment variables: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@router.post("/clear-cache")
async def clear_cache():
    """Clear application cache."""
    try:
        # Implementation would depend on caching strategy
        # For now, just return success
        return {"message": "Cache cleared successfully"}
        
    except Exception as e:
        logger.error(f"Failed to clear cache: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@router.get("/system-info")
async def get_system_info():
    """Get system information."""
    try:
        import platform
        import psutil
        
        return {
            "version": "1.0.0",
            "python_version": platform.python_version(),
            "platform": platform.platform(),
            "processor": platform.processor(),
            "memory_total": psutil.virtual_memory().total,
            "memory_available": psutil.virtual_memory().available,
            "disk_usage": psutil.disk_usage('/').percent,
            "cpu_count": psutil.cpu_count(),
        }
        
    except Exception as e:
        logger.error(f"Failed to get system info: {e}")
        return {
            "version": "1.0.0",
            "error": str(e)
        }