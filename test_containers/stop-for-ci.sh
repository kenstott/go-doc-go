#!/bin/bash
# Helper script to stop test containers for CI/automated testing
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "🛑 Stopping test containers..."

# Stop and remove containers, networks, and volumes
docker-compose down -v

echo "🧹 Cleaning up any remaining containers..."

# Clean up any orphaned containers
docker container prune -f --filter "label=com.docker.compose.project=test_containers" || true

# Clean up any unused networks
docker network prune -f || true

# Clean up any unused volumes (be careful - this removes all unused volumes)
if [ "${CLEANUP_VOLUMES:-false}" = "true" ]; then
    echo "⚠️  Removing unused volumes..."
    docker volume prune -f || true
fi

echo "✅ Test containers stopped and cleaned up!"