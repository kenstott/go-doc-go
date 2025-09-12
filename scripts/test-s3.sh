#!/bin/bash
# Simple S3 test setup script
# Starts Minio for testing, then you can use pytest directly

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

echo "=== S3 Test Setup ==="

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Check Docker
if ! command_exists docker; then
    echo "Error: Docker is required but not installed"
    exit 1
fi

# Check docker-compose vs docker compose
if command_exists docker-compose; then
    DOCKER_COMPOSE="docker-compose"
elif docker compose version &> /dev/null; then
    DOCKER_COMPOSE="docker compose"
else
    echo "Error: Neither docker-compose nor 'docker compose' is available"
    exit 1
fi

cd "$PROJECT_ROOT"

case "${1:-start}" in
    start|up)
        echo "Starting Minio container..."
        $DOCKER_COMPOSE -f docker-compose.test.yml up -d
        
        echo "Waiting for Minio to be ready..."
        for i in {1..30}; do
            if docker exec go-doc-go-test-minio mc alias set minio http://localhost:9000 minioadmin minioadmin &> /dev/null; then
                echo "Minio is ready!"
                break
            else
                if [[ $i -eq 30 ]]; then
                    echo "Error: Minio failed to start after 60s"
                    $DOCKER_COMPOSE -f docker-compose.test.yml logs minio
                    exit 1
                else
                    echo "Waiting... ($i/30)"
                    sleep 2
                fi
            fi
        done
        
        echo ""
        echo "Minio is running! Now you can run tests with pytest:"
        echo ""
        echo "  # Run all S3 tests"
        echo "  pytest tests/test_adapters/test_s3_adapter.py tests/test_content_sources/test_s3_content_source.py tests/test_integration/test_s3_integration.py -v"
        echo ""
        echo "  # Run only unit tests"
        echo "  pytest tests/test_adapters/test_s3_adapter.py::TestS3AdapterUnit tests/test_content_sources/test_s3_content_source.py::TestS3ContentSourceUnit -v"
        echo ""
        echo "  # Run only integration tests"
        echo "  pytest tests/test_adapters/test_s3_adapter.py::TestS3AdapterIntegration tests/test_content_sources/test_s3_content_source.py::TestS3ContentSourceIntegration tests/test_integration/test_s3_integration.py -v"
        echo ""
        echo "  # Run with coverage"
        echo "  pytest tests/test_*/*s3* -v --cov=src/go_doc_go/adapter/s3 --cov=src/go_doc_go/content_source/s3 --cov-report=html"
        echo ""
        echo "When done, stop Minio with:"
        echo "  $0 stop"
        ;;
    
    stop|down)
        echo "Stopping Minio container..."
        $DOCKER_COMPOSE -f docker-compose.test.yml down -v
        echo "Minio stopped."
        ;;
    
    restart)
        echo "Restarting Minio container..."
        $DOCKER_COMPOSE -f docker-compose.test.yml down -v
        $DOCKER_COMPOSE -f docker-compose.test.yml up -d
        
        echo "Waiting for Minio to be ready..."
        sleep 5
        if docker exec go-doc-go-test-minio mc alias set minio http://localhost:9000 minioadmin minioadmin &> /dev/null; then
            echo "Minio is ready!"
        else
            echo "Warning: Minio might not be fully ready"
        fi
        ;;
    
    status)
        echo "Minio container status:"
        $DOCKER_COMPOSE -f docker-compose.test.yml ps
        ;;
    
    logs)
        echo "Minio logs:"
        $DOCKER_COMPOSE -f docker-compose.test.yml logs minio
        ;;
        
    help|--help|-h)
        echo "S3 Test Setup Script"
        echo ""
        echo "Usage: $0 [COMMAND]"
        echo ""
        echo "Commands:"
        echo "  start, up      Start Minio container (default)"
        echo "  stop, down     Stop Minio container"
        echo "  restart        Restart Minio container"
        echo "  status         Show container status"
        echo "  logs           Show Minio logs"
        echo "  help           Show this help"
        echo ""
        echo "After starting Minio, use pytest directly for more control:"
        echo "  pytest tests/test_*/*s3* -v"
        ;;
    
    *)
        echo "Unknown command: $1"
        echo "Use '$0 help' for available commands"
        exit 1
        ;;
esac