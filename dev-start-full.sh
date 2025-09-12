#!/bin/bash
# Full development environment with all storage backends

echo "🚀 Starting Go-Doc-Go FULL Development Environment"
echo "=================================================="
echo ""

# Parse command line arguments
START_BACKENDS=false
if [[ "$1" == "--with-backends" ]] || [[ "$1" == "-b" ]]; then
    START_BACKENDS=true
fi

# Function to check if Docker is running
check_docker() {
    if ! docker info > /dev/null 2>&1; then
        echo "❌ Docker is not running. Please start Docker Desktop first."
        exit 1
    fi
}

# Check Docker
check_docker

# Start backend services if requested
if [ "$START_BACKENDS" = true ]; then
    echo "🗄️  Starting all storage backend services..."
    echo "============================================"
    
    # Stop any existing backend containers
    docker-compose -f docker-compose.backends.yml down 2>/dev/null
    
    # Start all backends
    docker-compose -f docker-compose.backends.yml up -d
    
    echo ""
    echo "⏳ Waiting for backends to be ready..."
    
    # Wait for services to be healthy
    SERVICES=("godocgo-postgres" "godocgo-mongo" "godocgo-elastic" "godocgo-neo4j" "godocgo-solr")
    MAX_WAIT=60
    WAITED=0
    
    while [ $WAITED -lt $MAX_WAIT ]; do
        ALL_READY=true
        for SERVICE in "${SERVICES[@]}"; do
            STATUS=$(docker inspect --format='{{.State.Health.Status}}' $SERVICE 2>/dev/null || echo "not-found")
            if [ "$STATUS" != "healthy" ]; then
                ALL_READY=false
                break
            fi
        done
        
        if [ "$ALL_READY" = true ]; then
            echo "✅ All backends are ready!"
            break
        fi
        
        sleep 2
        WAITED=$((WAITED + 2))
        echo -n "."
    done
    echo ""
else
    echo "💡 TIP: Use './dev-start-full.sh --with-backends' to start all storage backends"
    echo ""
fi

# Check available backends
echo ""
echo "📊 Checking available storage backends..."
python3 scripts/check_backends.py 2>/dev/null || {
    echo "⚠️  Backend check script not available or missing dependencies"
    echo "   Run: pip install rich pyyaml"
}

# Stop any existing containers
echo ""
echo "🧹 Cleaning up existing application containers..."
docker-compose -f docker-compose.dev.simple.yml down

# Start application with hot reloading
echo ""
echo "🔥 Starting application with hot reload..."
docker-compose -f docker-compose.dev.simple.yml up --build

echo ""
echo "✅ Development environment started!"
echo ""
echo "🌐 Frontend: http://localhost:3000"
echo "🔧 Backend API: http://localhost:8000"
echo "❤️  Health Check: http://localhost:8000/health"
echo "📊 Pipeline API: http://localhost:8000/api/pipelines"

if [ "$START_BACKENDS" = true ]; then
    echo ""
    echo "📦 Storage Backends:"
    echo "   PostgreSQL: localhost:5432 (user: postgres, pass: postgres)"
    echo "   MongoDB: localhost:27017"
    echo "   Elasticsearch: http://localhost:9200"
    echo "   Neo4j: http://localhost:7474 (user: neo4j, pass: password)"
    echo "   Solr: http://localhost:8983"
fi

echo ""
echo "📁 Edit files in:"
echo "   - src/go_doc_go/ (Python backend - auto-reloads)"
echo "   - frontend/src/ (React frontend - auto-reloads)"
echo ""
echo "🛑 To stop: Ctrl+C"

if [ "$START_BACKENDS" = true ]; then
    echo "🛑 To stop backends: docker-compose -f docker-compose.backends.yml down"
fi