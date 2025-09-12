#!/bin/bash
# Helper script to start specific test containers for development
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Default profile is "storage" (all storage services)
PROFILE="${1:-storage}"

echo "🚀 Starting test containers with profile: $PROFILE"
echo "Available profiles:"
echo "  - all: All services"
echo "  - storage: All storage services (PostgreSQL, Elasticsearch, MongoDB, MinIO, Neo4j)"
echo "  - sql: SQL databases (PostgreSQL)"
echo "  - nosql: NoSQL databases (MongoDB)"
echo "  - search: Search services (Elasticsearch, Solr)"
echo "  - s3: S3-compatible storage (MinIO)"
echo "  - graph: Graph database (Neo4j)"
echo ""

# Start containers for the specified profile
docker-compose --profile "$PROFILE" up -d

echo "⏳ Waiting for services to be ready..."
sleep 10

# Show status
echo "📊 Container Status:"
docker-compose ps

echo ""
echo "🔗 Service Endpoints:"
echo "  PostgreSQL: localhost:15432 (user: testuser, password: testpass, db: go_doc_go_test)"
echo "  MongoDB: localhost:27017 (user: testuser, password: testpass, db: go_doc_go_test)" 
echo "  Elasticsearch: http://localhost:9200"
echo "  MinIO: http://localhost:9000 (user: minioadmin, password: minioadmin)"
echo "  MinIO Console: http://localhost:9001"
echo "  Neo4j Browser: http://localhost:7474 (user: neo4j, password: testpass123)"
echo "  Solr: http://localhost:8983"
echo ""
echo "💡 Environment variables:"
echo "  export TEST_PG_HOST=localhost"
echo "  export TEST_PG_PORT=15432"
echo "  export TEST_PG_USER=testuser"
echo "  export TEST_PG_PASSWORD=testpass"
echo "  export TEST_PG_DB=go_doc_go_test"
echo "  export TEST_MONGO_HOST=localhost"
echo "  export TEST_MONGO_PORT=27017"
echo "  export TEST_MONGO_USER=testuser"
echo "  export TEST_MONGO_PASSWORD=testpass"
echo "  export TEST_MONGO_DB=go_doc_go_test"
echo "  export TEST_S3_ENDPOINT=http://localhost:9000"
echo "  export TEST_S3_ACCESS_KEY=minioadmin"
echo "  export TEST_S3_SECRET_KEY=minioadmin"
echo "  export TEST_NEO4J_URI=bolt://localhost:7687"
echo "  export TEST_NEO4J_USER=neo4j"
echo "  export TEST_NEO4J_PASSWORD=testpass123"
echo ""
echo "🎉 Ready for development!"