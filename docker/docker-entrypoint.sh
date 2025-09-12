#!/bin/bash
set -e

# Go-Doc-Go Docker Entrypoint Script
# Handles initialization and service startup for different container modes

echo "Go-Doc-Go Docker Entrypoint"
echo "Container ID: $(hostname)"
echo "Mode: ${GO_DOC_GO_MODE:-backend}"
echo "Config: ${GO_DOC_GO_CONFIG_PATH:-/app/config/config.yaml}"

# Function to wait for service to be ready
wait_for_service() {
    local host=$1
    local port=$2
    local service_name=$3
    local max_attempts=30
    local attempt=1

    echo "Waiting for $service_name at $host:$port..."
    
    while ! nc -z "$host" "$port"; do
        if [ $attempt -eq $max_attempts ]; then
            echo "ERROR: $service_name is not available after $max_attempts attempts"
            exit 1
        fi
        echo "Attempt $attempt: $service_name not yet available, waiting 2s..."
        sleep 2
        attempt=$((attempt + 1))
    done
    
    echo "$service_name is ready!"
}

# Function to check PostgreSQL readiness
wait_for_postgres() {
    if [[ -n "$DATABASE_URL" ]]; then
        # Extract host and port from DATABASE_URL
        local db_info=$(echo "$DATABASE_URL" | sed -n 's/.*@\([^:]*\):\([0-9]*\)\/.*/\1 \2/p')
        if [[ -n "$db_info" ]]; then
            wait_for_service $db_info "PostgreSQL"
        fi
    fi
}

# Function to check Redis readiness
wait_for_redis() {
    if [[ -n "$REDIS_URL" ]]; then
        local redis_host=$(echo "$REDIS_URL" | sed -n 's/.*:\/\/\([^:]*\):\([0-9]*\)/\1/p')
        local redis_port=$(echo "$REDIS_URL" | sed -n 's/.*:\/\/\([^:]*\):\([0-9]*\)/\2/p')
        if [[ -n "$redis_host" && -n "$redis_port" ]]; then
            wait_for_service "$redis_host" "$redis_port" "Redis"
        fi
    fi
}

# Function to initialize pipeline configuration database
init_pipeline_db() {
    echo "Initializing pipeline configuration database..."
    python -c "
import sys
sys.path.append('/app')
from src.go_doc_go.config_db import PipelineConfigDB
try:
    db = PipelineConfigDB('${PIPELINE_CONFIG_DB:-/app/data/pipeline_config.db}')
    print('Pipeline configuration database initialized successfully')
except Exception as e:
    print(f'ERROR: Failed to initialize pipeline database: {e}')
    sys.exit(1)
"
}

# Function to load API keys from files (Docker secrets)
load_api_keys() {
    if [[ -f "${OPENAI_API_KEY_FILE:-}" ]]; then
        export OPENAI_API_KEY=$(cat "$OPENAI_API_KEY_FILE")
        echo "Loaded OpenAI API key from file"
    fi
    
    if [[ -f "${ANTHROPIC_API_KEY_FILE:-}" ]]; then
        export ANTHROPIC_API_KEY=$(cat "$ANTHROPIC_API_KEY_FILE")
        echo "Loaded Anthropic API key from file"
    fi
    
    if [[ -f "${CLAUDE_API_KEY_FILE:-}" ]]; then
        export CLAUDE_API_KEY=$(cat "$CLAUDE_API_KEY_FILE")
        echo "Loaded Claude API key from file"
    fi
}

# Function to validate configuration
validate_config() {
    echo "Validating configuration..."
    if [[ ! -f "${GO_DOC_GO_CONFIG_PATH:-/app/config/config.yaml}" ]]; then
        echo "WARNING: Configuration file not found at ${GO_DOC_GO_CONFIG_PATH}"
        echo "Using default configuration"
    fi
    
    python -c "
import sys
sys.path.append('/app')
import yaml
try:
    with open('${GO_DOC_GO_CONFIG_PATH:-/app/config/config.yaml}', 'r') as f:
        config = yaml.safe_load(f)
    print('Configuration file is valid YAML')
except Exception as e:
    print(f'ERROR: Invalid configuration file: {e}')
    sys.exit(1)
"
}

# Function to create necessary directories
setup_directories() {
    echo "Setting up directories..."
    mkdir -p /app/data /app/logs /app/cache /app/temp
    
    # Ensure proper permissions
    if [[ $(id -u) == 0 ]]; then
        chown -R goDocGo:goDocGo /app/data /app/logs /app/cache /app/temp
    fi
}

# Main initialization
main() {
    echo "Starting initialization..."
    
    # Setup directories
    setup_directories
    
    # Load API keys from files if they exist
    load_api_keys
    
    # Wait for dependent services
    case "${GO_DOC_GO_MODE:-backend}" in
        backend|worker)
            wait_for_postgres
            wait_for_redis
            ;;
    esac
    
    # Validate configuration
    validate_config
    
    # Initialize pipeline database
    case "${GO_DOC_GO_MODE:-backend}" in
        backend)
            init_pipeline_db
            ;;
        worker)
            # Workers wait for backend to initialize the database
            echo "Waiting for pipeline configuration database to be initialized..."
            max_attempts=60
            attempt=1
            while ! python -c "from src.go_doc_go.config_db import PipelineConfigDB; PipelineConfigDB('${PIPELINE_CONFIG_DB:-/app/data/pipeline_config.db}')" 2>/dev/null; do
                if [ $attempt -eq $max_attempts ]; then
                    echo "ERROR: Pipeline database not available after $max_attempts attempts"
                    exit 1
                fi
                echo "Attempt $attempt: Pipeline database not ready, waiting 5s..."
                sleep 5
                attempt=$((attempt + 1))
            done
            echo "Pipeline configuration database is ready!"
            ;;
    esac
    
    echo "Initialization complete!"
    
    # Execute the main command
    case "${GO_DOC_GO_MODE:-backend}" in
        backend)
            echo "Starting Go-Doc-Go Backend Server..."
            exec python -m src.go_doc_go.server
            ;;
        worker)
            echo "Starting Go-Doc-Go Worker..."
            echo "Worker ID: $(hostname)"
            exec python -m src.go_doc_go.cli.worker
            ;;
        *)
            echo "Unknown mode: ${GO_DOC_GO_MODE}"
            echo "Supported modes: backend, worker"
            exit 1
            ;;
    esac
}

# Trap signals for graceful shutdown
trap 'echo "Received shutdown signal, cleaning up..."; exit 0' SIGTERM SIGINT

# Check if we're being called with a command
if [[ $# -gt 0 ]]; then
    # If arguments are provided, execute them directly
    exec "$@"
else
    # Otherwise, run our initialization and startup logic
    main
fi