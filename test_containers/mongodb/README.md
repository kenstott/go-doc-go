# MongoDB Test Container

This directory contains the MongoDB configuration for running tests.

## Quick Start

```bash
# Start the container
docker-compose -f compose.yaml up -d

# Wait for health check
docker-compose -f compose.yaml ps

# Test the connection
docker exec -it go-doc-go-test-mongodb mongosh -u admin -p admin123 --authenticationDatabase admin

# Stop the container
docker-compose -f compose.yaml down
```

## Configuration

- **Port**: 27017
- **Root User**: admin / admin123
- **Test User**: testuser / testpass
- **Database**: go_doc_go_test

## Performance Optimizations

The MongoDB instance is configured for testing with:
- `--nojournal` - Disable journaling for speed
- `--wiredTigerCacheSizeGB 0.5` - Limit memory usage
- `--quiet` - Reduce log verbosity

⚠️ **WARNING**: These settings are ONLY for testing. Never use in production!

## Collections Created

The initialization script creates:
- `documents` - Document metadata
- `elements` - Document elements/chunks
- `relationships` - Element relationships
- `entities` - Extracted entities
- `mongodb_documents` - Sample source documents
- `test_collection` - General testing
- `test_logs` - Capped collection for log testing

## Connection Strings

```python
# Python with pymongo
from pymongo import MongoClient

# Using root credentials
client = MongoClient('mongodb://admin:admin123@localhost:27017/')

# Using test user (recommended)
client = MongoClient('mongodb://testuser:testpass@localhost:27017/go_doc_go_test')

# Environment variables
export TEST_MONGO_HOST=localhost
export TEST_MONGO_PORT=27017
export TEST_MONGO_DB=go_doc_go_test
export TEST_MONGO_USER=testuser
export TEST_MONGO_PASSWORD=testpass
```

## Sample Queries

```javascript
// In mongosh
use go_doc_go_test;

// Find all documents
db.documents.find();

// Search elements by text
db.elements.find({ $text: { $search: "test" } });

// Aggregation example
db.documents.aggregate([
  { $match: { doc_type: "pdf" } },
  { $group: { _id: "$doc_type", count: { $sum: 1 } } }
]);
```

## Troubleshooting

If MongoDB fails to start:
```bash
# Check logs
docker-compose -f compose.yaml logs mongodb-test

# Connect without auth to debug
docker exec -it go-doc-go-test-mongodb mongosh

# Reset everything
docker-compose -f compose.yaml down -v
docker-compose -f compose.yaml up -d
```

## Integration with Tests

```python
import os
from pymongo import MongoClient
from pymongo.errors import ServerSelectionTimeoutError

def mongodb_available():
    try:
        client = MongoClient(
            host=os.getenv('TEST_MONGO_HOST', 'localhost'),
            port=int(os.getenv('TEST_MONGO_PORT', 27017)),
            username=os.getenv('TEST_MONGO_USER', 'testuser'),
            password=os.getenv('TEST_MONGO_PASSWORD', 'testpass'),
            serverSelectionTimeoutMS=5000
        )
        client.server_info()
        client.close()
        return True
    except ServerSelectionTimeoutError:
        return False
```

## Test Data

The init script inserts sample data:
- 2 documents in `documents` collection
- 2 elements in `elements` collection
- 2 MongoDB source documents in `mongodb_documents`

This data is available immediately after container startup for testing.