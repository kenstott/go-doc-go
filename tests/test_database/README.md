# Database Tests

This directory contains comprehensive tests for the Go-Doc-Go database components, including both SQLite and PostgreSQL support. The tests follow the same structure and patterns as the S3 tests, providing thorough coverage of database adapters, content sources, and integration scenarios.

## Overview

The database testing suite covers two main components:

1. **DatabaseAdapter** (`go_doc_go.adapter.database`) - Uses native database drivers (sqlite3, psycopg2, mysql.connector)
2. **DatabaseContentSource** (`go_doc_go.content_source.database`) - Uses SQLAlchemy for advanced ORM features

## Test Structure

```
tests/test_database/
├── conftest.py                          # Test fixtures and configuration
├── init-postgres.sql                   # PostgreSQL initialization script
├── test_database_adapter.py            # Tests for native driver-based adapter
├── test_database_content_source.py     # Tests for SQLAlchemy-based content source
├── test_database_integration.py        # End-to-end integration tests
└── README.md                           # This file
```

## Test Categories

### Unit Tests (Fast, No External Dependencies)

- **DatabaseAdapter Unit Tests**: Test URI parsing, connection logic, error handling
- **DatabaseContentSource Unit Tests**: Test SQLAlchemy integration, configuration handling

### Integration Tests (Require Docker for PostgreSQL)

- **SQLite Integration Tests**: Test with actual SQLite databases
- **PostgreSQL Integration Tests**: Test with Docker-based PostgreSQL instance
- **Document Pipeline Tests**: Test complete document processing pipeline
- **Performance Tests**: Test connection caching, large documents, cleanup

## Prerequisites

### Required Dependencies

```bash
# Core testing
pip install pytest

# Database support
pip install sqlalchemy    # For DatabaseContentSource
pip install psycopg2     # For PostgreSQL support

# Optional but recommended
pip install pytest-cov   # For coverage reports
```

### Docker Setup

For PostgreSQL integration tests, Docker and Docker Compose are required:

```bash
# Check Docker installation
docker --version
docker-compose --version
```

## Running the Tests

### Quick Start (Unit Tests Only)

```bash
# Run all unit tests (no external dependencies)
pytest tests/test_database/ -v -m "not requires_docker"

# Run specific test files
pytest tests/test_database/test_database_adapter.py -v
pytest tests/test_database/test_database_content_source.py -v
```

### Integration Tests (Requires PostgreSQL)

```bash
# 1. Start PostgreSQL container
./scripts/test-database.sh start

# 2. Run integration tests
pytest tests/test_database/ -v -m "requires_docker"

# 3. Stop PostgreSQL container
./scripts/test-database.sh stop
```

### All Tests

```bash
# Start PostgreSQL
./scripts/test-database.sh start

# Run all tests (unit + integration)
pytest tests/test_database/ -v

# Stop PostgreSQL
./scripts/test-database.sh stop
```

### Test Categories with pytest

```bash
# Unit tests only (fast, no Docker)
pytest tests/test_database/ -v -m "not requires_docker"

# Integration tests only (requires Docker)
pytest tests/test_database/ -v -m "requires_docker"

# SQLite tests only
pytest tests/test_database/ -v -k "sqlite"

# PostgreSQL tests only
pytest tests/test_database/ -v -k "postgres"

# Adapter tests only
pytest tests/test_database/test_database_adapter.py -v

# Content source tests only  
pytest tests/test_database/test_database_content_source.py -v

# Integration tests only
pytest tests/test_database/test_database_integration.py -v

# With coverage reporting
pytest tests/test_database/ -v --cov=go_doc_go --cov-report=html
```

## Test Configuration

### SQLite Tests

SQLite tests use temporary databases created with test data:

- **Documents table**: Contains markdown, text, JSON, and CSV sample documents
- **Binary_docs table**: Contains binary test data (PNG headers)
- **Json_records table**: Contains structured data for JSON mode testing

### PostgreSQL Tests

PostgreSQL tests use Docker Compose with:

- **Container**: postgres:15-alpine
- **Database**: testdb
- **User**: testuser / testpass
- **Port**: 5432
- **Features**: JSONB support, array columns, full-text search capabilities

## Test Data

### Sample Documents

The test suite includes various document types:

```sql
-- Markdown documents
"# Header\n\nThis is a markdown document with headers."

-- JSON documents  
'{"key": "value", "number": 42, "array": [1, 2, 3]}'

-- CSV documents
"Name,Age,City\nJohn,30,NYC\nJane,25,LA\nBob,35,Chicago"

-- Binary data
PNG headers and other binary content
```

### Test Tables

#### SQLite Schema

```sql
CREATE TABLE documents (
    id INTEGER PRIMARY KEY,
    title TEXT NOT NULL,
    content TEXT,
    doc_type TEXT DEFAULT 'text',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    metadata TEXT
);

CREATE TABLE json_records (
    id INTEGER PRIMARY KEY,
    name TEXT,
    description TEXT,
    data_field TEXT,
    status TEXT,
    config TEXT,
    tags TEXT,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

#### PostgreSQL Schema

```sql
CREATE TABLE documents (
    id SERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    content TEXT,
    doc_type VARCHAR(50) DEFAULT 'text',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    metadata JSONB
);

CREATE TABLE json_records (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255),
    description TEXT,
    data_field TEXT,
    status VARCHAR(50),
    config JSONB,
    tags TEXT[],
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## Key Test Scenarios

### DatabaseAdapter Tests

1. **URI Parsing**: Test db:// URI format parsing
2. **Connection Management**: Test SQLite, PostgreSQL, MySQL connections
3. **Content Retrieval**: Test text, JSON, CSV, binary content
4. **Error Handling**: Test invalid URIs, missing records, connection failures
5. **Connection Caching**: Test connection reuse and cleanup

### DatabaseContentSource Tests (SQLAlchemy)

1. **Blob Mode**: Test single-column content extraction
2. **JSON Mode**: Test multi-column JSON document creation
3. **Metadata Handling**: Test metadata column processing
4. **Change Detection**: Test timestamp-based change detection
5. **PostgreSQL Features**: Test JSONB, arrays, advanced types

### Integration Tests

1. **Document Pipeline**: Test complete parsing workflow
2. **Content Type Detection**: Test automatic content type detection
3. **Parser Integration**: Test with markdown, JSON, CSV parsers
4. **Error Recovery**: Test graceful error handling
5. **Performance**: Test large documents, concurrent access

## Configuration Examples

### DatabaseAdapter URI Format

```
db://<connection_id>/<table>/<pk_column>/<pk_value>/<content_column>[/<content_type>]

Examples:
- db://test.db/documents/id/123/content
- db://postgres://user:pass@host:5432/db/documents/id/123/content/markdown
```

### DatabaseContentSource Configuration

```python
# Blob mode configuration
config = {
    "name": "db-source",
    "connection_string": "sqlite:///documents.db",
    "query": "documents",
    "id_column": "id", 
    "content_column": "content",
    "metadata_columns": ["title", "doc_type"],
    "timestamp_column": "created_at"
}

# JSON mode configuration
config = {
    "name": "db-json-source",
    "connection_string": "postgresql://user:pass@localhost/db",
    "query": "records",
    "id_column": "id",
    "json_mode": True,
    "json_columns": ["name", "data", "config"],
    "metadata_columns": ["status"],
    "timestamp_column": "updated_at"
}
```

## Error Handling

The tests verify proper error handling for:

- **Connection Errors**: Invalid connection strings, network issues
- **Missing Dependencies**: SQLAlchemy, psycopg2 not available
- **Invalid Data**: Malformed URIs, non-existent records
- **Type Errors**: Binary data in text contexts, JSON serialization issues
- **Resource Cleanup**: Connection cleanup, temp file management

## Performance Considerations

### Connection Caching

Both adapter and content source implement connection caching:

```python
# Connections are cached by connection string/ID
adapter.connections[connection_id] = connection

# Cleanup releases all cached connections
adapter.cleanup()
```

### Large Document Handling

Tests verify proper handling of:
- Large JSON documents (1000+ objects)
- Binary content streams
- Memory-efficient processing

### Concurrent Access

Tests verify thread-safe connection handling and resource management.

## Debugging

### Common Issues

1. **PostgreSQL Connection Failed**
   ```bash
   # Check Docker services
   docker-compose -f docker-compose.database.yml ps
   
   # Check PostgreSQL logs
   docker-compose -f docker-compose.database.yml logs postgres
   ```

2. **SQLAlchemy Import Error**
   ```bash
   pip install sqlalchemy
   ```

3. **psycopg2 Import Error**
   ```bash
   pip install psycopg2-binary
   ```

### Verbose Testing

```bash
# Run with maximum verbosity
pytest tests/test_database/ -v -s --tb=long

# Run specific failing test
pytest tests/test_database/test_database_adapter.py::TestDatabaseAdapterSQLiteIntegration::test_get_content_from_sqlite -v -s
```

## Extending the Tests

### Adding New Database Types

1. Update `DatabaseAdapter._get_connection()` to support new database type
2. Add connection string parsing logic
3. Create test fixtures for new database type
4. Add integration tests following existing patterns

### Adding New Test Scenarios

1. Create test data in `conftest.py` fixtures
2. Follow naming convention: `test_<scenario>_<database_type>`
3. Use appropriate pytest markers for dependencies
4. Include both unit and integration test variations

## CI/CD Integration

The test suite is designed for CI/CD environments:

```yaml
# Example GitHub Actions snippet
- name: Run Database Tests
  run: |
    # Quick unit tests first
    ./scripts/test-database.sh --unit-only
    
    # Full integration tests 
    ./scripts/test-database.sh --integration-only
```

### Test Markers

Use pytest markers to control test execution:

```python
@requires_sqlalchemy
@requires_psycopg2 
@requires_docker
def test_postgres_feature():
    pass
```

This comprehensive test suite ensures robust database support across SQLite and PostgreSQL, with proper error handling, performance optimization, and integration with the document processing pipeline.