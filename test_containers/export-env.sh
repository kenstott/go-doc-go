#!/bin/bash
# Export environment variables for test containers
# Usage: source test_containers/export-env.sh

# PostgreSQL
export TEST_PG_HOST=localhost
export TEST_PG_PORT=15432
export TEST_PG_USER=testuser
export TEST_PG_PASSWORD=testpass
export TEST_PG_DB=go_doc_go_test

# MongoDB
export TEST_MONGO_HOST=localhost
export TEST_MONGO_PORT=27017
export TEST_MONGO_USER=testuser
export TEST_MONGO_PASSWORD=testpass
export TEST_MONGO_DB=go_doc_go_test

# MinIO (S3)
export TEST_S3_ENDPOINT=http://localhost:9000
export TEST_S3_ACCESS_KEY=minioadmin
export TEST_S3_SECRET_KEY=minioadmin
export TEST_S3_BUCKET=test-bucket

# Elasticsearch
export TEST_ES_HOST=localhost
export TEST_ES_PORT=9200

# Solr
export TEST_SOLR_HOST=localhost
export TEST_SOLR_PORT=8983

# Neo4j
export TEST_NEO4J_URI=bolt://localhost:7687
export TEST_NEO4J_USER=neo4j
export TEST_NEO4J_PASSWORD=testpass123

echo "✅ Test container environment variables exported!"
echo "💡 Available services:"
echo "  - PostgreSQL: $TEST_PG_HOST:$TEST_PG_PORT"
echo "  - MongoDB: $TEST_MONGO_HOST:$TEST_MONGO_PORT"  
echo "  - Elasticsearch: $TEST_ES_HOST:$TEST_ES_PORT"
echo "  - MinIO: $TEST_S3_ENDPOINT"
echo "  - Neo4j: $TEST_NEO4J_URI"
echo "  - Solr: $TEST_SOLR_HOST:$TEST_SOLR_PORT"