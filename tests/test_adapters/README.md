# S3 Adapter and Content Source Tests

This directory contains comprehensive tests for the S3 adapter and content source components, using a self-contained Minio instance to simulate S3 storage.

## Overview

The test suite provides:
- **Unit tests** for S3 adapter and content source components
- **Integration tests** for end-to-end document processing
- **Docker-based Minio** for S3-compatible storage testing
- **Comprehensive fixtures** for test data management

## Prerequisites

1. **Docker** - Required for running Minio container
2. **Python dependencies** - Install with test dependencies:
   ```bash
   pip install -e ".[development,cloud-aws]"
   ```

## Test Structure

```
tests/
├── test_adapters/
│   ├── conftest.py              # Shared fixtures and utilities
│   ├── test_s3_adapter.py       # S3 adapter unit and integration tests
│   └── README.md                # This file
├── test_content_sources/
│   ├── conftest.py              # Imports fixtures from test_adapters
│   └── test_s3_content_source.py # S3 content source tests
├── test_integration/
│   ├── conftest.py              # Imports fixtures from test_adapters
│   └── test_s3_integration.py   # End-to-end integration tests
└── test_data/
    └── s3/                      # Sample test data files
        ├── sample-doc.md
        ├── data.json
        ├── employees.csv
        └── nested/
            └── details.md
```

## Running Tests

### Quick Start

Use the provided helper script to run all S3 tests:

```bash
# Run all S3 tests with Minio
./scripts/test-s3.sh

# Run with coverage reporting
./scripts/test-s3.sh --coverage

# Run in verbose mode
./scripts/test-s3.sh --verbose

# Skip Docker setup (use existing Minio)
./scripts/test-s3.sh --skip-docker
```

### Manual Testing

1. **Start Minio for development:**
   ```bash
   ./scripts/setup-minio.sh start
   ```

2. **Run specific test files:**
   ```bash
   # Adapter tests only
   pytest tests/test_adapters/test_s3_adapter.py -v
   
   # Content source tests only
   pytest tests/test_content_sources/test_s3_content_source.py -v
   
   # Integration tests only
   pytest tests/test_integration/test_s3_integration.py -v
   ```

3. **Run with coverage:**
   ```bash
   pytest tests/test_adapters/test_s3_adapter.py \
     --cov=src/go_doc_go/adapter/s3 \
     --cov-report=term-missing \
     --cov-report=html
   ```

4. **Stop Minio:**
   ```bash
   ./scripts/setup-minio.sh stop
   ```

### Docker Compose Commands

```bash
# Start Minio manually
docker-compose -f docker-compose.test.yml up -d

# View Minio logs
docker-compose -f docker-compose.test.yml logs -f minio

# Stop and clean up
docker-compose -f docker-compose.test.yml down -v
```

## Test Configuration

### Minio Configuration

The test Minio instance is configured with:
- **Endpoint:** http://localhost:9000
- **Console:** http://localhost:9001
- **Access Key:** minioadmin
- **Secret Key:** minioadmin
- **Test Buckets:**
  - `test-bucket` - Main test bucket
  - `test-private-bucket` - Private access tests
  - `test-public-bucket` - Public access tests

### Environment Variables

You can override test configuration with environment variables:

```bash
# Use custom Minio endpoint
export MINIO_ENDPOINT=http://minio.local:9000

# Use real AWS S3 (not recommended for tests)
export AWS_ACCESS_KEY_ID=your-key
export AWS_SECRET_ACCESS_KEY=your-secret
export S3_TEST_BUCKET=your-test-bucket
```

## Test Coverage

The test suite covers:

### S3 Adapter Tests
- ✅ Initialization with various configurations
- ✅ URI parsing and validation
- ✅ Content retrieval (text and binary)
- ✅ Metadata extraction
- ✅ Error handling (missing files, access denied)
- ✅ Encoding detection and fallbacks
- ✅ Content type detection

### S3 Content Source Tests
- ✅ Document fetching from S3
- ✅ Document listing with filters
- ✅ Change detection
- ✅ Link following in documents
- ✅ Content caching
- ✅ Include/exclude patterns
- ✅ Binary file handling
- ✅ Temporary file management

### Integration Tests
- ✅ End-to-end document processing
- ✅ Multiple document type support (MD, JSON, CSV, PDF, etc.)
- ✅ Link traversal between documents
- ✅ Concurrent access handling
- ✅ Large file processing
- ✅ Metadata preservation
- ✅ Error recovery

## Fixtures

Key pytest fixtures provided:

- `docker_compose_up` - Starts/stops Docker services
- `minio_config` - Minio connection configuration
- `s3_client` - Configured boto3 S3 client
- `minio_client` - Python Minio client
- `upload_test_files` - Helper to upload test files
- `sample_text_content` - Sample markdown content
- `sample_json_content` - Sample JSON content
- `sample_csv_content` - Sample CSV content

## Troubleshooting

### Docker Issues

If Docker containers fail to start:
```bash
# Check Docker daemon is running
docker ps

# Clean up existing containers
docker-compose -f docker-compose.test.yml down -v

# Remove orphaned containers
docker container prune

# Check port conflicts
lsof -i :9000
lsof -i :9001
```

### Minio Connection Issues

If tests can't connect to Minio:
```bash
# Check Minio is running
docker ps | grep minio

# Test connection manually
curl http://localhost:9000/minio/health/live

# Check Minio logs
docker logs go-doc-go-test-minio
```

### Test Failures

If tests fail unexpectedly:
```bash
# Run tests in verbose mode
pytest tests/test_adapters/test_s3_adapter.py -vvs

# Check for stale test data
docker-compose -f docker-compose.test.yml down -v
docker volume prune

# Reinstall dependencies
pip install -e ".[development,cloud-aws]" --force-reinstall
```

## CI/CD Integration

To run S3 tests in CI/CD pipelines:

```yaml
# GitHub Actions example
- name: Start Minio
  run: docker-compose -f docker-compose.test.yml up -d
  
- name: Wait for Minio
  run: |
    timeout 60 bash -c 'until curl -s http://localhost:9000/minio/health/live; do sleep 2; done'

- name: Run S3 Tests
  run: |
    pip install -e ".[development,cloud-aws]"
    pytest tests/test_adapters/ tests/test_content_sources/ tests/test_integration/ -v

- name: Cleanup
  if: always()
  run: docker-compose -f docker-compose.test.yml down -v
```

## Contributing

When adding new S3-related tests:

1. **Use existing fixtures** - Leverage the fixtures in `conftest.py`
2. **Clean up resources** - Ensure test data is cleaned up after tests
3. **Mock external calls** - Use mocks for unit tests, real Minio for integration tests
4. **Document test purpose** - Add clear docstrings explaining what each test validates
5. **Handle Docker gracefully** - Skip tests if Docker is not available

## License

These tests are part of the go-doc-go project and follow the same MIT license.