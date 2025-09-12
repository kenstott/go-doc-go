#!/bin/bash
# Helper script to completely reset test containers for development
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "⚠️  CAUTION: This will completely reset all test containers and volumes!"
echo "   All data will be lost. Press Ctrl+C to cancel."
echo ""
echo "Continuing in 5 seconds..."
sleep 5

echo "🗑️  Stopping and removing containers, networks, and volumes..."

# Stop and remove everything
docker-compose down -v

# Clean up any unused resources
echo "🧹 Cleaning up unused Docker resources..."
docker container prune -f
docker network prune -f
docker volume prune -f

# Rebuild containers from scratch if needed
if [ "${REBUILD:-false}" = "true" ]; then
    echo "🔨 Rebuilding containers..."
    docker-compose build --no-cache
fi

echo ""
echo "✅ Test environment completely reset!"
echo "💡 Run './start-dev.sh' to start fresh containers"