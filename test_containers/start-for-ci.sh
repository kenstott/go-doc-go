#!/bin/bash
# Helper script to start test containers for CI/automated testing
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "🚀 Starting test containers for CI..."

# Start containers in detached mode
docker-compose up -d

# Wait for services to be healthy
echo "⏳ Waiting for services to be ready..."
timeout=120
start_time=$(date +%s)

while true; do
    current_time=$(date +%s)
    elapsed=$((current_time - start_time))
    
    if [ $elapsed -ge $timeout ]; then
        echo "❌ Timeout waiting for services to be ready"
        docker-compose ps
        docker-compose logs
        exit 1
    fi
    
    # Check if all services are healthy
    healthy_services=$(docker-compose ps --services --filter "status=running" | wc -l)
    total_services=$(docker-compose ps --services | wc -l)
    
    if [ "$healthy_services" -eq "$total_services" ]; then
        echo "✅ All services are ready!"
        break
    fi
    
    echo "   $healthy_services/$total_services services ready..."
    sleep 5
done

echo "🎉 Test containers started successfully!"
docker-compose ps