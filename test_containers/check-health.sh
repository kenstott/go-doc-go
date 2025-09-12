#!/bin/bash
# Helper script to check the health of test containers
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "🏥 Checking test container health..."
echo ""

# Show container status
echo "📊 Container Status:"
docker-compose ps
echo ""

# Function to check service health
check_service() {
    local service_name="$1"
    local host="$2"
    local port="$3"
    local protocol="${4:-tcp}"
    
    if [ "$protocol" = "http" ]; then
        if curl -s --max-time 5 "$host:$port" >/dev/null 2>&1; then
            echo "✅ $service_name: Healthy ($host:$port)"
        else
            echo "❌ $service_name: Not responding ($host:$port)"
        fi
    else
        if timeout 5 bash -c "cat < /dev/null > /dev/tcp/${host#*://}/$port" 2>/dev/null; then
            echo "✅ $service_name: Healthy ($host:$port)"
        else
            echo "❌ $service_name: Not responding ($host:$port)"
        fi
    fi
}

echo "🔍 Service Health Checks:"

# Check PostgreSQL
check_service "PostgreSQL" "localhost" "15432"

# Check MongoDB
check_service "MongoDB" "localhost" "27017"

# Check Elasticsearch
check_service "Elasticsearch" "http://localhost" "9200" "http"

# Check MinIO
check_service "MinIO S3" "http://localhost" "9000" "http"

# Check MinIO Console
check_service "MinIO Console" "http://localhost" "9001" "http"

# Check Neo4j HTTP
check_service "Neo4j HTTP" "http://localhost" "7474" "http"

# Check Neo4j Bolt
check_service "Neo4j Bolt" "localhost" "7687"

# Check Solr
check_service "Solr" "http://localhost" "8983" "http"

echo ""
echo "📋 Quick Tests:"

# Test PostgreSQL connection
if command -v psql >/dev/null 2>&1; then
    if PGPASSWORD=testpass psql -h localhost -p 15432 -U testuser -d go_doc_go_test -c "SELECT 1;" >/dev/null 2>&1; then
        echo "✅ PostgreSQL: Connection test passed"
    else
        echo "❌ PostgreSQL: Connection test failed"
    fi
else
    echo "⚠️  PostgreSQL: psql not available for testing"
fi

# Test MongoDB connection
if command -v mongosh >/dev/null 2>&1; then
    if mongosh --host localhost:27017 --username testuser --password testpass --authenticationDatabase admin --eval "db.runCommand('ping')" >/dev/null 2>&1; then
        echo "✅ MongoDB: Connection test passed"
    else
        echo "❌ MongoDB: Connection test failed"
    fi
elif command -v mongo >/dev/null 2>&1; then
    if mongo --host localhost:27017 --username testuser --password testpass --authenticationDatabase admin --eval "db.runCommand('ping')" >/dev/null 2>&1; then
        echo "✅ MongoDB: Connection test passed"
    else
        echo "❌ MongoDB: Connection test failed"
    fi
else
    echo "⚠️  MongoDB: mongo/mongosh not available for testing"
fi

# Test Elasticsearch
if curl -s --max-time 5 "http://localhost:9200/_cluster/health" | grep -q "green\|yellow"; then
    echo "✅ Elasticsearch: Cluster health check passed"
else
    echo "❌ Elasticsearch: Cluster health check failed"
fi

# Test Neo4j
if curl -s --max-time 5 "http://localhost:7474/db/data/" | grep -q "neo4j"; then
    echo "✅ Neo4j: HTTP endpoint accessible"
else
    echo "❌ Neo4j: HTTP endpoint not accessible"
fi

echo ""
echo "🎉 Health check complete!"