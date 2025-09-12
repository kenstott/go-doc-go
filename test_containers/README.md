# Test Containers

This directory contains Docker configurations for all services used in testing the go-doc-go system. Each service is configured for optimal test performance with minimal resource usage.

## Quick Start

```bash
# Start all test containers
docker-compose up -d

# Start specific services
docker-compose up -d postgres-test minio-test

# Start by profile
docker-compose --profile storage up -d     # All storage services
docker-compose --profile search up -d      # Search services (Elasticsearch, Solr)
docker-compose --profile sql up -d         # SQL databases (PostgreSQL)
docker-compose --profile nosql up -d       # NoSQL databases (MongoDB)
docker-compose --profile s3 up -d          # S3-compatible storage (MinIO)

# Check status
docker-compose ps

# View logs
docker-compose logs -f [service-name]

# Stop all containers
docker-compose down

# Stop and remove volumes (full reset)
docker-compose down -v
```

## Services

| Service | Port | Purpose | Profile |
|---------|------|---------|---------|
| PostgreSQL | 15432 | Relational database | sql, storage |
| Elasticsearch | 9200 | Full-text search | search, storage |
| MongoDB | 27017 | Document database | nosql, storage |
| MinIO | 9000/9001 | S3-compatible storage | s3, storage |
| Solr | 8983 | Search platform | search, storage |
| Neo4j | 7474/7687 | Graph database | graph, storage |

## Directory Structure

```
test_containers/
├── README.md                  # This file
├── docker-compose.yml         # Master compose file
├── postgres/                  # PostgreSQL configuration
│   ├── compose.yaml
│   ├── init.sql
│   └── README.md
├── elasticsearch/             # Elasticsearch configuration
│   ├── compose.yaml
│   ├── elasticsearch.yml
│   └── README.md
├── mongodb/                   # MongoDB configuration
│   ├── compose.yaml
│   ├── init.js
│   └── README.md
├── minio/                     # MinIO S3 configuration
│   ├── compose.yaml
│   ├── init-buckets.sh
│   └── README.md
├── neo4j/                     # Neo4j graph database
│   ├── compose.yaml
│   ├── init.cypher
│   └── README.md
└── solr/                      # Solr configuration
    ├── compose.yaml
    ├── Dockerfile
    ├── init-cores.sh
    └── README.md
```

## Default Credentials

| Service | Username | Password | Database/Bucket |
|---------|----------|----------|-----------------|
| PostgreSQL | testuser | testpass | go_doc_go_test |
| MongoDB | testuser | testpass | go_doc_go_test |
| MinIO | minioadmin | minioadmin | test-bucket |
| Neo4j | neo4j | testpass123 | - |
| Elasticsearch | - | - | - |
| Solr | - | - | - |

## Environment Variables

Set these in your test configuration or shell:

```bash
# PostgreSQL
export TEST_PG_HOST=localhost
export TEST_PG_PORT=15432
export TEST_PG_DB=go_doc_go_test
export TEST_PG_USER=testuser
export TEST_PG_PASSWORD=testpass

# MongoDB
export TEST_MONGO_HOST=localhost
export TEST_MONGO_PORT=27017
export TEST_MONGO_DB=go_doc_go_test
export TEST_MONGO_USER=testuser
export TEST_MONGO_PASSWORD=testpass

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
```

## Running Tests

### With pytest

```python
# conftest.py
import pytest
import os

@pytest.fixture(scope="session")
def ensure_containers():
    """Ensure test containers are running."""
    import subprocess
    
    # Start containers if not running
    result = subprocess.run(
        ["docker-compose", "ps", "-q"],
        cwd="test_containers",
        capture_output=True
    )
    
    if not result.stdout:
        subprocess.run(
            ["docker-compose", "up", "-d"],
            cwd="test_containers",
            check=True
        )
        import time
        time.sleep(10)  # Wait for services to be ready

@pytest.mark.usefixtures("ensure_containers")
class TestWithContainers:
    """Base class for tests requiring containers."""
    pass
```

### Manual Testing

```bash
# Start required services
cd test_containers
docker-compose up -d postgres-test minio-test

# Run tests
cd ..
pytest tests/test_adapters/ -v

# Cleanup
cd test_containers
docker-compose down
```

## Performance Optimizations

All services are configured for testing with:
- Reduced memory usage
- Disabled persistence features (where appropriate)
- Optimized for speed over durability
- Single-node configurations
- Minimal logging

⚠️ **WARNING**: These configurations are ONLY for testing. Never use in production!

## Troubleshooting

### Port Conflicts
If you get port binding errors:
```bash
# Find what's using the port (example for 5432)
lsof -i :5432

# Change the port in the service's compose.yaml
```

### Memory Issues
If containers fail due to memory:
```bash
# Increase Docker Desktop memory allocation
# Docker Desktop > Preferences > Resources > Memory

# Or reduce service memory usage in compose files
```

### Reset Everything
```bash
# Stop all containers and remove volumes
docker-compose down -v

# Remove any persisted data
rm -rf */data

# Start fresh
docker-compose up -d
```

### Health Checks
Each service has health checks configured. Wait for healthy status:
```bash
# Watch service health
watch docker-compose ps

# Check specific service health
docker inspect go-doc-go-test-postgres | jq '.[0].State.Health'
```

## CI/CD Integration

For GitHub Actions:
```yaml
services:
  postgres:
    image: postgres:15-alpine
    env:
      POSTGRES_PASSWORD: testpass
    options: >-
      --health-cmd pg_isready
      --health-interval 10s
      --health-timeout 5s
      --health-retries 5
```

For local CI:
```bash
# Start services before tests
./test_containers/start-for-ci.sh

# Run tests
pytest

# Cleanup
./test_containers/stop-for-ci.sh
```

## Adding New Services

To add a new test service:

1. Create a directory: `test_containers/<service-name>/`
2. Add `compose.yaml` with service configuration
3. Add initialization scripts if needed
4. Add `README.md` with usage instructions
5. Update master `docker-compose.yml` to include the service
6. Update this README with service details

## Best Practices

1. **Use profiles** to group related services
2. **Set health checks** for reliable startup
3. **Use tmpfs** for databases when possible (faster)
4. **Limit resources** to prevent runaway containers
5. **Document credentials** and connection strings
6. **Provide initialization scripts** for test data
7. **Use consistent naming** (go-doc-go-test-*)
8. **Share networks** for inter-service communication

## Support

For issues with specific services, see the README in each service directory.
For general issues, check the main project documentation.