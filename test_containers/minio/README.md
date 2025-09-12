# MinIO (S3-Compatible) Test Container

This directory contains the MinIO configuration for S3 testing.

## Quick Start

```bash
# Start the containers
docker-compose -f compose.yaml up -d

# Wait for initialization
docker-compose -f compose.yaml ps

# Test the connection
curl http://localhost:9000/minio/health/live

# Access the web console
open http://localhost:9001
# Login: minioadmin/minioadmin

# Stop the containers
docker-compose -f compose.yaml down
```

## Configuration

- **S3 API Port**: 9000
- **Console Port**: 9001
- **Root User**: minioadmin / minioadmin
- **Test User**: testuser / testpass123

## Buckets Created

- `test-bucket` - Main test bucket with versioning
- `test-private` - Private bucket for auth testing
- `test-public` - Public read bucket

## Connection Examples

### Python with boto3
```python
import boto3

# Create S3 client
s3 = boto3.client(
    's3',
    endpoint_url='http://localhost:9000',
    aws_access_key_id='minioadmin',
    aws_secret_access_key='minioadmin',
    region_name='us-east-1'
)

# List buckets
buckets = s3.list_buckets()
print(buckets['Buckets'])

# Upload file
s3.upload_file('local_file.txt', 'test-bucket', 'remote_file.txt')

# Download file
s3.download_file('test-bucket', 'remote_file.txt', 'downloaded.txt')
```

### Environment Variables
```bash
export TEST_S3_ENDPOINT=http://localhost:9000
export TEST_S3_ACCESS_KEY=minioadmin
export TEST_S3_SECRET_KEY=minioadmin
export TEST_S3_BUCKET=test-bucket
export TEST_S3_REGION=us-east-1
```

## MinIO Client (mc) Commands

```bash
# Configure mc client
mc config host add local http://localhost:9000 minioadmin minioadmin

# List buckets
mc ls local/

# List bucket contents
mc ls local/test-bucket/

# Upload file
mc cp file.txt local/test-bucket/

# Download file
mc cp local/test-bucket/file.txt ./

# Mirror directory
mc mirror ./local-dir local/test-bucket/remote-dir/
```

## Sample Data

The initialization script uploads sample data from `tests/test_data/s3/` if available, or creates minimal test files:
- `test1.txt`
- `folder1/test2.txt`
- `data/test.json`
- `docs/test.md`

## IAM and Policies

A test user `testuser` is created with limited permissions to `test-bucket` only. This is useful for testing authentication and authorization.

## Troubleshooting

If MinIO fails to start:
```bash
# Check logs
docker-compose -f compose.yaml logs minio-test

# Check initialization logs
docker-compose -f compose.yaml logs minio-init

# Common issues:
# 1. Port 9000 or 9001 already in use
# 2. Permission issues with ./data directory
# 3. Docker volume conflicts

# Reset everything
docker-compose -f compose.yaml down -v
rm -rf ./data
docker-compose -f compose.yaml up -d
```

## Integration with Tests

```python
import os
import boto3
from botocore.exceptions import ClientError

def minio_available():
    try:
        s3 = boto3.client(
            's3',
            endpoint_url=os.getenv('TEST_S3_ENDPOINT', 'http://localhost:9000'),
            aws_access_key_id=os.getenv('TEST_S3_ACCESS_KEY', 'minioadmin'),
            aws_secret_access_key=os.getenv('TEST_S3_SECRET_KEY', 'minioadmin'),
            region_name='us-east-1'
        )
        s3.list_buckets()
        return True
    except (ClientError, Exception):
        return False

# pytest fixture
@pytest.fixture
def s3_client():
    if not minio_available():
        pytest.skip("MinIO not available")
    
    return boto3.client(
        's3',
        endpoint_url='http://localhost:9000',
        aws_access_key_id='minioadmin',
        aws_secret_access_key='minioadmin',
        region_name='us-east-1'
    )
```

## Performance Notes

MinIO is lightweight and suitable for testing:
- Single node setup
- No erasure coding overhead
- Local filesystem backend
- Minimal resource usage

For production, use actual AWS S3 or a production MinIO cluster.