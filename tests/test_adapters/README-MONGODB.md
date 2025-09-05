# MongoDB Adapter and Content Source Tests

This directory contains comprehensive tests for the MongoDB adapter and content source components, using a self-contained MongoDB Docker instance.

## Overview

The MongoDB test suite provides:
- **Unit tests** for MongoDB adapter and content source components
- **Integration tests** for end-to-end document processing with MongoDB
- **Docker-based MongoDB** instance for realistic testing
- **Comprehensive fixtures** for MongoDB document management

## Prerequisites

1. **Docker** - Required for running MongoDB container
2. **Python dependencies** - Install with MongoDB support:
   ```bash
   pip install -e ".[development,db-mongodb,source-mongodb]"
   ```

## MongoDB URI Format

The MongoDB adapter supports URIs in the following format:
```
mongodb://[username:password@]host[:port]/database/collection[/document_id][/field_path]
```

Examples:
- `mongodb://localhost/mydb/mycollection` - Collection level
- `mongodb://localhost/mydb/mycollection/507f1f77bcf86cd799439011` - Document by ObjectId
- `mongodb://localhost/mydb/mycollection/doc123/metadata/author` - Specific field

## Test Structure

```
tests/
├── test_adapters/
│   ├── test_mongodb_adapter.py     # MongoDB adapter tests
│   └── conftest.py                 # MongoDB fixtures
├── test_content_sources/
│   └── test_mongodb_content_source.py # Content source tests
├── test_integration/
│   └── test_mongodb_integration.py # End-to-end tests
└── test_data/
    └── mongodb/
        ├── sample_documents.json   # Sample MongoDB documents
        ├── init/                   # Initialization scripts
        └── README.md               # Test data documentation
```

## Running Tests

### Quick Start

Use the provided helper script to run all MongoDB tests:

```bash
# Run all MongoDB tests with Docker MongoDB
./scripts/test-mongodb.sh

# Run with coverage reporting
./scripts/test-mongodb.sh --coverage

# Run in verbose mode
./scripts/test-mongodb.sh --verbose

# Skip Docker setup (use existing MongoDB)
./scripts/test-mongodb.sh --skip-docker

# Run only integration tests
./scripts/test-mongodb.sh --integration
```

### Manual Testing

1. **Start MongoDB for development:**
   ```bash
   ./scripts/setup-mongodb.sh start
   
   # With sample data
   ./scripts/setup-mongodb.sh start --init-data
   ```

2. **Open MongoDB shell:**
   ```bash
   ./scripts/setup-mongodb.sh shell
   ```

3. **Run specific test files:**
   ```bash
   # Adapter tests only
   pytest tests/test_adapters/test_mongodb_adapter.py -v
   
   # Content source tests only
   pytest tests/test_content_sources/test_mongodb_content_source.py -v
   
   # Integration tests only
   pytest tests/test_integration/test_mongodb_integration.py -v
   ```

4. **Run with coverage:**
   ```bash
   pytest tests/test_adapters/test_mongodb_adapter.py \
     --cov=src/go_doc_go/adapter/mongodb \
     --cov-report=term-missing \
     --cov-report=html
   ```

5. **Stop MongoDB:**
   ```bash
   ./scripts/setup-mongodb.sh stop
   ```

### Docker Compose Commands

```bash
# Start MongoDB manually
docker-compose -f docker-compose.test.yml up -d mongodb

# View MongoDB logs
docker-compose -f docker-compose.test.yml logs -f mongodb

# Initialize test data
docker-compose -f docker-compose.test.yml up mongodb-init

# Stop and clean up
docker-compose -f docker-compose.test.yml down -v
```

## Test Configuration

### MongoDB Configuration

The test MongoDB instance is configured with:
- **Host:** localhost
- **Port:** 27017
- **Username:** admin
- **Password:** admin123
- **Database:** test_db
- **Auth Database:** admin
- **Connection String:** `mongodb://admin:admin123@localhost:27017/`

### Test Collections

- `test_collection` - Main test collection
- `documents` - Document storage tests
- `users` - User data tests
- `products` - Product catalog tests

## Test Coverage

The test suite covers:

### MongoDB Adapter Tests
- ✅ Initialization with various configurations
- ✅ URI parsing (database/collection/document/field)
- ✅ ObjectId handling and conversion
- ✅ Field extraction with dot notation
- ✅ Array element access
- ✅ Connection string masking
- ✅ Error handling (missing documents, invalid fields)
- ✅ Content type detection
- ✅ Binary content handling
- ✅ Metadata extraction

### MongoDB Content Source Tests
- ✅ Document fetching from collections
- ✅ Document listing with queries
- ✅ Change detection via timestamps
- ✅ Reference following between documents
- ✅ Query filters and projections
- ✅ Sorting and limiting
- ✅ Content caching
- ✅ BSON type serialization
- ✅ Connection pooling

### Integration Tests
- ✅ End-to-end document processing
- ✅ Complex JSON document handling
- ✅ Nested structure navigation
- ✅ Array processing
- ✅ Reference resolution
- ✅ Change tracking
- ✅ Large collection handling
- ✅ BSON type compatibility
- ✅ Query and projection testing
- ✅ Concurrent access

## Fixtures

Key pytest fixtures provided:

- `mongodb_config` - MongoDB connection configuration
- `mongodb_client` - Configured PyMongo client
- `mongodb_collection` - Test collection access
- `insert_test_documents` - Helper to insert test documents
- `sample_mongodb_documents` - Sample document data
- `requires_pymongo` - Skip marker for pymongo dependency

## Sample Test Data

The test suite includes various document types:

### Simple Documents
```json
{
  "name": "Test Document",
  "content": "Document content",
  "tags": ["test", "sample"]
}
```

### Nested Documents
```json
{
  "metadata": {
    "author": {
      "name": "John Doe",
      "email": "john@example.com"
    }
  }
}
```

### Documents with References
```json
{
  "_id": "507f1f77bcf86cd799439011",
  "references": ["507f1f77bcf86cd799439012", "507f1f77bcf86cd799439013"]
}
```

### Documents with Arrays
```json
{
  "items": [
    {"index": 0, "value": "first"},
    {"index": 1, "value": "second"}
  ]
}
```

## Troubleshooting

### MongoDB Connection Issues

If tests can't connect to MongoDB:
```bash
# Check MongoDB is running
docker ps | grep mongodb

# Test connection manually
docker exec doculyzer-test-mongodb mongosh --eval "db.adminCommand('ping')"

# Check MongoDB logs
docker logs doculyzer-test-mongodb
```

### Authentication Issues

If authentication fails:
```bash
# Verify credentials
docker exec doculyzer-test-mongodb mongosh \
  -u admin -p admin123 --authenticationDatabase admin

# Check user exists
docker exec doculyzer-test-mongodb mongosh \
  -u admin -p admin123 --authenticationDatabase admin \
  --eval "db.getUsers()"
```

### Test Failures

If tests fail unexpectedly:
```bash
# Run tests in verbose mode
pytest tests/test_adapters/test_mongodb_adapter.py -vvs

# Check for stale test data
docker-compose -f docker-compose.test.yml down -v
docker volume prune

# Reinstall dependencies
pip install -e ".[development,db-mongodb,source-mongodb]" --force-reinstall
```

## CI/CD Integration

To run MongoDB tests in CI/CD pipelines:

```yaml
# GitHub Actions example
- name: Start MongoDB
  run: docker-compose -f docker-compose.test.yml up -d mongodb
  
- name: Wait for MongoDB
  run: |
    timeout 60 bash -c 'until docker exec doculyzer-test-mongodb mongosh --quiet --eval "db.adminCommand(\"ping\")"; do sleep 2; done'

- name: Run MongoDB Tests
  run: |
    pip install -e ".[development,db-mongodb,source-mongodb]"
    pytest tests/test_adapters/test_mongodb_adapter.py \
           tests/test_content_sources/test_mongodb_content_source.py \
           tests/test_integration/test_mongodb_integration.py -v

- name: Cleanup
  if: always()
  run: docker-compose -f docker-compose.test.yml down -v
```

## Performance Considerations

### Connection Pooling
- The adapter maintains a connection pool for efficiency
- Connections are reused across multiple operations
- Use `cleanup()` method to close connections when done

### Caching
- Document content is cached to reduce database queries
- Metadata is cached separately
- Clear cache with `content_cache.clear()` when needed

### Query Optimization
- Use projections to limit returned fields
- Apply query filters to reduce result sets
- Use indexes for frequently queried fields

## Security Notes

- Connection strings are masked in logs and errors
- Credentials should use environment variables in production
- Use authentication and SSL/TLS in production environments
- Apply principle of least privilege for database users

## Contributing

When adding new MongoDB tests:

1. **Use existing fixtures** - Leverage the fixtures in `conftest.py`
2. **Clean up test data** - Ensure documents are removed after tests
3. **Handle BSON types** - Test ObjectId, Date, and other BSON types
4. **Test error cases** - Include tests for missing documents, invalid queries
5. **Document test purpose** - Add clear docstrings explaining test objectives

## License

These tests are part of the go-doc-go project and follow the same MIT license.