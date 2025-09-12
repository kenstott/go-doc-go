#!/bin/bash
# Helper script to stop test containers for development
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "🛑 Stopping test containers..."

# Stop containers but keep volumes for development
docker-compose down

echo "📊 Remaining containers:"
docker-compose ps

echo ""
echo "💡 To completely reset (remove volumes):"
echo "  ./reset-dev.sh"
echo ""
echo "✅ Development containers stopped!"