# PostgreSQL Test Container

This directory contains the PostgreSQL configuration for running tests.

## Quick Start

```bash
# Start the container
docker-compose -f compose.yaml up -d

# Wait for health check
docker-compose -f compose.yaml ps

# Run tests
TEST_PG_HOST=localhost TEST_PG_PORT=15432 pytest tests/

# Stop the container
docker-compose -f compose.yaml down
```

## Configuration

- **Port**: 15432 (mapped from container's 5432)
- **Database**: go_doc_go_test
- **Username**: testuser
- **Password**: testpass

## Performance Optimizations

The PostgreSQL instance is configured for testing with:
- `fsync=off` - Disable disk sync for speed
- `synchronous_commit=off` - Don't wait for WAL writes
- `full_page_writes=off` - Reduce WAL size
- tmpfs volume - Database runs in memory

⚠️ **WARNING**: These settings are ONLY for testing. Never use in production!

## Initialization

The `init.sql` script automatically:
1. Creates required extensions (uuid-ossp, pg_trgm)
2. Sets up all required tables
3. Creates indexes for performance
4. Adds the atomic leader election function
5. Inserts sample test data

## Connection String

```python
# Python/SQLAlchemy
connection_string = "postgresql://testuser:testpass@localhost:15432/go_doc_go_test"

# Environment variables
export TEST_PG_HOST=localhost
export TEST_PG_PORT=15432
export TEST_PG_DB=go_doc_go_test
export TEST_PG_USER=testuser
export TEST_PG_PASSWORD=testpass
```

## Troubleshooting

If the container fails to start:
```bash
# Check logs
docker-compose -f compose.yaml logs postgres-test

# Reset everything
docker-compose -f compose.yaml down -v
docker-compose -f compose.yaml up -d
```

## Integration with Tests

Tests can check if PostgreSQL is available:
```python
import os
import psycopg2

def postgres_available():
    try:
        conn = psycopg2.connect(
            host=os.getenv('TEST_PG_HOST', 'localhost'),
            port=int(os.getenv('TEST_PG_PORT', 15432)),
            database=os.getenv('TEST_PG_DB', 'go_doc_go_test'),
            user=os.getenv('TEST_PG_USER', 'testuser'),
            password=os.getenv('TEST_PG_PASSWORD', 'testpass')
        )
        conn.close()
        return True
    except:
        return False
```