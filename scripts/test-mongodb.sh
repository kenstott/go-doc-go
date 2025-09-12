#!/bin/bash
# Simple MongoDB test setup script
# Starts MongoDB for testing, then you can use pytest directly

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

echo "=== MongoDB Test Setup ==="

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
        echo "Starting MongoDB container..."
        $DOCKER_COMPOSE -f docker-compose.test.yml up -d mongodb
        
        echo "Waiting for MongoDB to be ready..."
        for i in {1..30}; do
            if docker exec go-doc-go-test-mongodb mongosh --quiet --eval "db.adminCommand('ping')" &> /dev/null; then
                echo "MongoDB is ready!"
                
                echo "Initializing test data..."
                $DOCKER_COMPOSE -f docker-compose.test.yml up -d mongodb-init
                sleep 3
                break
            else
                if [[ $i -eq 30 ]]; then
                    echo "Error: MongoDB failed to start after 60s"
                    $DOCKER_COMPOSE -f docker-compose.test.yml logs mongodb
                    exit 1
                else
                    echo "Waiting... ($i/30)"
                    sleep 2
                fi
            fi
        done
        
        echo ""
        echo "MongoDB is running! Now you can run tests with pytest:"
        echo ""
        echo "  # Run all MongoDB tests"
        echo "  pytest tests/test_adapters/test_mongodb_adapter.py tests/test_content_sources/test_mongodb_content_source.py tests/test_integration/test_mongodb_integration.py -v"
        echo ""
        echo "  # Run only unit tests"
        echo "  pytest tests/test_adapters/test_mongodb_adapter.py::TestMongoDBAdapterUnit tests/test_content_sources/test_mongodb_content_source.py::TestMongoDBContentSourceUnit -v"
        echo ""
        echo "  # Run only integration tests"
        echo "  pytest tests/test_adapters/test_mongodb_adapter.py::TestMongoDBAdapterIntegration tests/test_content_sources/test_mongodb_content_source.py::TestMongoDBContentSourceIntegration tests/test_integration/test_mongodb_integration.py -v"
        echo ""
        echo "  # Run with coverage"
        echo "  pytest tests/test_*/*mongodb* -v --cov=src/go_doc_go/adapter/mongodb --cov=src/go_doc_go/content_source/mongodb --cov-report=html"
        echo ""
        echo "When done, stop MongoDB with:"
        echo "  $0 stop"
        ;;
    
    stop|down)
        echo "Stopping MongoDB container..."
        $DOCKER_COMPOSE -f docker-compose.test.yml stop mongodb mongodb-init
        $DOCKER_COMPOSE -f docker-compose.test.yml rm -f mongodb-init
        echo "MongoDB stopped."
        ;;
    
    restart)
        echo "Restarting MongoDB container..."
        $DOCKER_COMPOSE -f docker-compose.test.yml stop mongodb mongodb-init
        $DOCKER_COMPOSE -f docker-compose.test.yml rm -f mongodb-init
        $DOCKER_COMPOSE -f docker-compose.test.yml up -d mongodb
        
        echo "Waiting for MongoDB to be ready..."
        sleep 5
        if docker exec go-doc-go-test-mongodb mongosh --quiet --eval "db.adminCommand('ping')" &> /dev/null; then
            echo "MongoDB is ready!"
            echo "Initializing test data..."
            $DOCKER_COMPOSE -f docker-compose.test.yml up -d mongodb-init
            sleep 3
        else
            echo "Warning: MongoDB might not be fully ready"
        fi
        ;;
    
    status)
        echo "MongoDB container status:"
        $DOCKER_COMPOSE -f docker-compose.test.yml ps mongodb mongodb-init
        ;;
    
    logs)
        echo "MongoDB logs:"
        $DOCKER_COMPOSE -f docker-compose.test.yml logs mongodb
        ;;
        
    help|--help|-h)
        echo "MongoDB Test Setup Script"
        echo ""
        echo "Usage: $0 [COMMAND]"
        echo ""
        echo "Commands:"
        echo "  start, up      Start MongoDB container (default)"
        echo "  stop, down     Stop MongoDB container"
        echo "  restart        Restart MongoDB container"
        echo "  status         Show container status"
        echo "  logs           Show MongoDB logs"
        echo "  help           Show this help"
        echo ""
        echo "After starting MongoDB, use pytest directly for more control:"
        echo "  pytest tests/test_*/*mongodb* -v"
        ;;
    
    *)
        echo "Unknown command: $1"
        echo "Use '$0 help' for available commands"
        exit 1
        ;;
esac