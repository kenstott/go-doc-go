#!/bin/bash

# Simple database test setup script
# Starts PostgreSQL for testing, then you can use pytest directly

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

echo "=== Database Test Setup ==="

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Check Docker
if ! command_exists docker; then
    echo "Error: Docker is required but not installed"
    exit 1
fi

if ! command_exists docker-compose; then
    echo "Error: Docker Compose is required but not installed"
    exit 1
fi

cd "$PROJECT_DIR"

case "${1:-start}" in
    start|up)
        echo "Starting PostgreSQL container..."
        docker-compose -f docker-compose.database.yml up -d
        
        echo "Waiting for PostgreSQL to be ready..."
        for i in {1..12}; do
            if docker-compose -f docker-compose.database.yml exec -T postgres pg_isready -U testuser -d testdb 2>/dev/null; then
                echo "PostgreSQL is ready!"
                break
            else
                if [[ $i -eq 12 ]]; then
                    echo "Warning: PostgreSQL readiness check failed after 60s"
                    exit 1
                else
                    echo "Waiting... ($i/12)"
                    sleep 5
                fi
            fi
        done
        
        echo ""
        echo "PostgreSQL is running! Now you can run tests with pytest:"
        echo ""
        echo "  # Run all database tests"
        echo "  pytest tests/test_database/ -v"
        echo ""
        echo "  # Run only unit tests (no Docker needed)"
        echo "  pytest tests/test_database/ -v -m 'not requires_docker'"
        echo ""
        echo "  # Run only integration tests"
        echo "  pytest tests/test_database/ -v -m 'requires_docker'"
        echo ""
        echo "  # Run SQLite tests only"
        echo "  pytest tests/test_database/ -v -k 'sqlite'"
        echo ""
        echo "  # Run PostgreSQL tests only" 
        echo "  pytest tests/test_database/ -v -k 'postgres'"
        echo ""
        echo "When done, stop PostgreSQL with:"
        echo "  $0 stop"
        ;;
    
    stop|down)
        echo "Stopping PostgreSQL container..."
        docker-compose -f docker-compose.database.yml down -v
        echo "PostgreSQL stopped."
        ;;
    
    restart)
        echo "Restarting PostgreSQL container..."
        docker-compose -f docker-compose.database.yml down -v
        docker-compose -f docker-compose.database.yml up -d
        
        echo "Waiting for PostgreSQL to be ready..."
        sleep 10
        if docker-compose -f docker-compose.database.yml exec -T postgres pg_isready -U testuser -d testdb; then
            echo "PostgreSQL is ready!"
        else
            echo "Warning: PostgreSQL might not be fully ready"
        fi
        ;;
    
    status)
        echo "Database container status:"
        docker-compose -f docker-compose.database.yml ps
        ;;
    
    logs)
        echo "PostgreSQL logs:"
        docker-compose -f docker-compose.database.yml logs postgres
        ;;
        
    help|--help|-h)
        echo "Database Test Setup Script"
        echo ""
        echo "Usage: $0 [COMMAND]"
        echo ""
        echo "Commands:"
        echo "  start, up      Start PostgreSQL container (default)"
        echo "  stop, down     Stop PostgreSQL container"
        echo "  restart        Restart PostgreSQL container"
        echo "  status         Show container status"
        echo "  logs           Show PostgreSQL logs"
        echo "  help           Show this help"
        echo ""
        echo "After starting PostgreSQL, use pytest directly:"
        echo "  pytest tests/test_database/ -v"
        ;;
    
    *)
        echo "Unknown command: $1"
        echo "Use '$0 help' for available commands"
        exit 1
        ;;
esac