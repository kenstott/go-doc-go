# Elasticsearch Test Container

This directory contains the Elasticsearch configuration for running tests.

## Quick Start

```bash
# Start the container
docker-compose -f compose.yaml up -d

# Wait for health check (this can take 30-60 seconds)
docker-compose -f compose.yaml ps

# Test the connection
curl -X GET "localhost:9200/_cluster/health?pretty"

# Stop the container
docker-compose -f compose.yaml down
```

## Configuration

- **Port**: 9200 (HTTP API)
- **Port**: 9300 (Transport, for cluster communication)
- **Security**: Disabled (for testing only)
- **Mode**: Single-node
- **Memory**: 512MB heap

## Index Setup

Create test indices:
```bash
# Create documents index
curl -X PUT "localhost:9200/go_doc_go_documents" -H 'Content-Type: application/json' -d'
{
  "settings": {
    "number_of_shards": 1,
    "number_of_replicas": 0
  },
  "mappings": {
    "properties": {
      "doc_id": { "type": "keyword" },
      "doc_type": { "type": "keyword" },
      "source": { "type": "text" },
      "metadata": { "type": "object", "enabled": true },
      "created_at": { "type": "date" }
    }
  }
}'

# Create elements index
curl -X PUT "localhost:9200/go_doc_go_elements" -H 'Content-Type: application/json' -d'
{
  "settings": {
    "number_of_shards": 1,
    "number_of_replicas": 0
  },
  "mappings": {
    "properties": {
      "element_id": { "type": "keyword" },
      "doc_id": { "type": "keyword" },
      "element_type": { "type": "keyword" },
      "content_preview": { "type": "text" },
      "metadata": { "type": "object", "enabled": true },
      "embedding": { "type": "dense_vector", "dims": 384 }
    }
  }
}'
```

## Connection String

```python
# Python with elasticsearch-py
from elasticsearch import Elasticsearch

es = Elasticsearch(
    ["http://localhost:9200"],
    verify_certs=False,
    ssl_show_warn=False
)

# Environment variables
export TEST_ES_HOST=localhost
export TEST_ES_PORT=9200
```

## Performance Notes

The instance is configured for testing with:
- Single shard, no replicas (faster indexing)
- Security disabled (no SSL/TLS overhead)
- ML features disabled (saves memory)
- 512MB heap (sufficient for tests)

⚠️ **WARNING**: These settings are ONLY for testing. Never use in production!

## Troubleshooting

If Elasticsearch fails to start:
```bash
# Check logs
docker-compose -f compose.yaml logs elasticsearch-test

# Common issues:
# 1. Not enough memory - increase Docker memory allocation
# 2. Port conflict - check if 9200 is already in use
# 3. vm.max_map_count too low - run:
sudo sysctl -w vm.max_map_count=262144

# Reset everything
docker-compose -f compose.yaml down -v
docker-compose -f compose.yaml up -d
```

## Integration with Tests

```python
import os
from elasticsearch import Elasticsearch

def elasticsearch_available():
    try:
        es = Elasticsearch(
            [f"http://{os.getenv('TEST_ES_HOST', 'localhost')}:{os.getenv('TEST_ES_PORT', 9200)}"],
            verify_certs=False,
            max_retries=0,
            timeout=5
        )
        return es.ping()
    except:
        return False
```