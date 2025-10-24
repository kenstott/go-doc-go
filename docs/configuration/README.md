# Go-Doc-Go Configuration Guide

This guide covers all configuration options for Go-Doc-Go. Configuration is provided via TOML files, typically named `config.toml`.

## Configuration File Structure

The configuration file has several main sections:

```toml
[storage]          # Storage backend configuration
[embedding]        # Embedding generation settings
[[content_sources]]  # Document sources to process (array of tables)
[relationship_detection]  # Relationship extraction settings
[processing]       # Processing pipeline configuration
[analytics]        # Analytics output configuration
[logging]          # Logging configuration
```

## Storage Configuration

Configure where processed documents and metadata are stored.

### SQLite (Default)

```toml
[storage]
backend = "sqlite"
path = "./documents.db"

## Optional SQLite-specific settings
enable_fts = true           # Full-text search
page_size = 4096
cache_size = 10000
journal_mode = "WAL"
```toml

### PostgreSQL

```toml
[storage]
backend = "postgresql"
host = "localhost"
port = 5432
database = "go_doc_go"
username = "postgres"
password = "${DB_PASSWORD}"  # Use environment variable

## Connection pooling
pool_size = 20
max_overflow = 30
pool_timeout = 30

## Performance tuning
statement_timeout = 300000
batch_size = 1000
```toml

### MongoDB

```toml
[storage]
backend = "mongodb"
host = "localhost"
port = 27017
database = "go_doc_go"
username = "${MONGO_USER}"
password = "${MONGO_PASSWORD}"

## Connection options
max_pool_size = 100
min_pool_size = 10

## Collections
documents_collection = "documents"
elements_collection = "elements"
relationships_collection = "relationships"
```toml

### Elasticsearch

```toml
[storage]
backend = "elasticsearch"
hosts = ["http://localhost:9200"]

## Authentication
username = "elastic"
password = "${ELASTIC_PASSWORD}"

## Index configuration
index_prefix = "go-doc-go"
number_of_shards = 3
number_of_replicas = 1
```go

## Embedding Configuration

Configure how text embeddings are generated for semantic search.

### FastEmbed (Recommended)

```toml
[embedding]
enabled = true
provider = "fastembed"
model = "BAAI/bge-small-en-v1.5"
dimensions = 384

## Chunking settings
chunk_size = 512
overlap = 128

## Contextual embeddings (GraphRAG-lite)
contextual = true
predecessor_count = 1
successor_count = 1
include_metadata = true

## Performance
batch_size = 64
max_sequence_length = 512
```toml

### HuggingFace Transformers

```toml
[embedding]
enabled = true
provider = "huggingface"
model = "sentence-transformers/all-MiniLM-L6-v2"
dimensions = 384

## Device configuration
device = "cuda"  # or "cpu", "mps"
precision = "float16"
```toml

### OpenAI

```toml
[embedding]
enabled = true
provider = "openai"
model = "text-embedding-ada-002"
dimensions = 1536
api_key = "${OPENAI_API_KEY}"

## Rate limiting
max_requests_per_minute = 3000
max_tokens_per_minute = 1000000
```

## Content Sources

Configure where documents come from. Multiple sources can be configured.

### File System

```toml
[[content_sources]]
name = "local_docs"
type = "file"
base_path = "./documents"

## File selection
file_pattern = "**/*.{pdf,docx,md,txt}"
include_patterns = [
  "**/*.pdf",
  "**/*.docx"
]
exclude_patterns = [
  "**/archive/**",
  "**/temp/**"
]

## Options
watch_for_changes = true
follow_links = true
max_link_depth = 2

## Discovery interval (seconds) - how often to check for new documents
## Defaults to global --discovery-interval CLI option or 86400 (1 day)
discovery_interval = 3600  # Check every hour
```toml

### Database

```toml
[[content_sources]]
name = "cms_content"
type = "database"
connection_string = "postgresql://user:pass@host/db"

## Query for documents
query = """
SELECT id, title, content, created_at, author
FROM articles
WHERE status = 'published'
"""

## Field mapping
[content_sources.field_mapping]
doc_id = "id"
title = "title"
content = "content"

[content_sources.field_mapping.metadata]
author = "author"
created = "created_at"

## Processing options
batch_size = 1000
stream_results = true

## Discovery interval for this database source
discovery_interval = 21600  # Check every 6 hours
```toml

### Web Scraping

```toml
[[content_sources]]
name = "wiki_docs"
type = "web"
base_url = "https://wiki.example.com"

## URLs to start from
url_list = [
  "https://wiki.example.com/docs"
]

## URL filtering
include_patterns = [
  "^https://wiki.example.com/docs/"
]
exclude_patterns = [
  "/archive/",
  "/temp/"
]

## Crawling settings
max_link_depth = 2
max_pages = 1000

## Rate limiting
delay = 1.0

## Headers
[content_sources.headers]
User-Agent = "Go-Doc-Go/1.0"

## Refresh interval (seconds)
refresh_interval = 86400  # 1 day
```toml

### S3

```toml
[[content_sources]]
name = "s3_documents"
type = "s3"
bucket = "my-documents"
prefix = "docs/"

## AWS credentials
aws_access_key_id = "${AWS_ACCESS_KEY}"
aws_secret_access_key = "${AWS_SECRET_KEY}"
region = "us-west-2"

## Options
include_extensions = [".pdf", ".docx", ".txt"]
workers = 10
```toml

### SharePoint

```toml
[[content_sources]]
name = "sharepoint"
type = "sharepoint"
site_url = "https://company.sharepoint.com/sites/docs"

## Authentication
client_id = "${SHAREPOINT_CLIENT_ID}"
client_secret = "${SHAREPOINT_CLIENT_SECRET}"
tenant_id = "${SHAREPOINT_TENANT_ID}"

## Libraries to scan
libraries = [
  "Shared Documents",
  "Project Files"
]
```toml

### Confluence

```toml
[[content_sources]]
name = "confluence"
type = "confluence"
url = "https://company.atlassian.net"
username = "${CONFLUENCE_USER}"
api_token = "${CONFLUENCE_TOKEN}"

## Spaces to include
spaces = [
  "TECH",
  "PROD"
]

## Content filtering
include_page_types = ["page", "blogpost"]
exclude_labels = ["draft", "obsolete"]
```

## Relationship Detection

Configure how relationships between elements are discovered.

```toml
[relationship_detection]
enabled = true

## Structural relationships (parent-child, siblings)
structural = true

## Semantic similarity relationships
semantic = true
similarity_threshold = 0.7

## Cross-document relationships
[relationship_detection.cross_document_semantic]
enabled = true
similarity_threshold = 0.75
max_relationships_per_element = 10

## Domain ontology relationships
[relationship_detection.domain]
enabled = true

[[relationship_detection.domain.ontologies]]
path = "./ontologies/financial.yaml"
active = true

[[relationship_detection.domain.ontologies]]
path = "./ontologies/technical.yaml"
active = true
```yaml

## Processing Configuration

Configure the document processing pipeline.

```toml
[processing]
## Batch processing
batch_size = 100
workers = 4
timeout_seconds = 300

## Job control for distributed processing
[processing.job_control]
backend = "sqlite"  # or "postgresql"
path = "./job_queue.db"

## Worker coordination
claim_timeout = 300        # 5 minutes
heartbeat_interval = 30    # 30 seconds
max_retries = 3

## Dead letter queue
enable_dlq = true
dlq_threshold = 3

## Discovery intervals
## Per-source intervals can be configured in each content source
## Priority: source-specific > CLI --discovery-interval > default
default_discovery_interval = 86400  # Default: 1 day

## Memory management
max_memory_per_worker = "1GB"
stream_large_files = true
large_file_threshold = 10485760  # 10MB
```

## Analytics Configuration

Configure analytics outputs for monitoring and reporting.

```toml
[analytics]
enabled = true

## Parquet files for data warehouse
[[analytics.outputs]]
type = "parquet"
path = "./analytics-output"
partitioning = ["date", "source"]
compression = "snappy"

## SQLite for local analytics
[[analytics.outputs]]
type = "sqlite"
path = "./analytics.db"
tables = ["documents", "elements", "relationships", "embeddings", "metrics"]

## PostgreSQL for production analytics
[[analytics.outputs]]
type = "postgresql"
connection_string = "postgresql://user:pass@host/analytics"
schema = "go_doc_go_analytics"
```go

## Logging Configuration

Configure logging behavior.

```toml
[logging]
level = "INFO"  # DEBUG, INFO, WARNING, ERROR, CRITICAL

## Log outputs
[[logging.handlers]]
type = "console"
level = "INFO"
format = "%(asctime)s - %(name)s - %(levelname)s - %(message)s"

[[logging.handlers]]
type = "file"
filename = "./logs/go_doc_go.log"
level = "DEBUG"
max_bytes = 10485760  # 10MB
backup_count = 5

## Component-specific logging
[logging.loggers]
"go_doc_go.worker" = "DEBUG"
"go_doc_go.storage" = "INFO"
"go_doc_go.embeddings" = "WARNING"
```go

## Environment Variables

Configuration values can reference environment variables using `${VAR_NAME}` syntax:

```toml
[storage]
backend = "postgresql"
host = "${DB_HOST}"
password = "${DB_PASSWORD}"
```bash

Set environment variables:

```bash
export DB_HOST=localhost
export DB_PASSWORD=secretpassword
```bash

Or use a `.env` file:

```bash
## .env
DB_HOST=localhost
DB_PASSWORD=secretpassword
```

## Configuration Profiles

Use different configurations for different environments:

### Development

```toml
## config.dev.toml
[storage]
backend = "sqlite"
path = "./dev.db"

[embedding]
enabled = false  # Skip embeddings for faster development

[processing]
workers = 2
batch_size = 10

[logging]
level = "DEBUG"
```toml

### Production

```toml
## config.prod.toml
[storage]
backend = "postgresql"
host = "${PROD_DB_HOST}"
database = "go_doc_go_prod"
pool_size = 50

[embedding]
enabled = true
provider = "fastembed"
batch_size = 128

[processing]
workers = 16
batch_size = 1000

[logging]
level = "WARNING"
```

## Validation

Validate your configuration:

```bash
## Test with small batch
../bin/goworker --config config.toml --max-documents 10
```bash

## Common Configuration Patterns

### Minimal Configuration

```toml
## Minimal config for testing
[processing.job_control]
backend = "sqlite"
path = "./test.db"

[[content_sources]]
name = "test"
type = "file"
base_path = "./test-docs"
```toml

### High-Performance Configuration

```toml
## Optimized for performance
[processing.job_control]
backend = "postgresql"
path = "postgresql://user:pass@host/db"

[embedding]
provider = "fastembed"
batch_size = 256

[processing]
workers = 32
batch_size = 1000

[[analytics.outputs]]
type = "parquet"
compression = "snappy"
```toml

### Enterprise Configuration

```toml
## Enterprise features
[processing.job_control]
backend = "postgresql"
path = "postgresql://${DB_USER}:${DB_PASS}@${DB_CLUSTER}/godocgo"

[embedding]
provider = "openai"
api_key = "${OPENAI_API_KEY}"

[[content_sources]]
name = "sharepoint"
type = "sharepoint"
## ... SharePoint config

[[content_sources]]
name = "confluence"
type = "confluence"
## ... Confluence config

[processing]
workers = 16

[logging]
level = "INFO"
```

## Configuration Best Practices

1. **Use environment variables for secrets** - Never commit passwords or API keys
2. **Start with minimal config** - Add complexity as needed
3. **Test with small batches** - Use `--max-documents` to test
4. **Monitor resource usage** - Adjust workers and batch sizes based on system capacity
5. **Enable logging** - Use DEBUG level when troubleshooting
6. **Validate before production** - Test configuration thoroughly
7. **Use appropriate storage** - SQLite for dev, PostgreSQL for production
8. **Configure timeouts** - Set reasonable timeouts for your document complexity
9. **Enable analytics** - Monitor processing performance
10. **Document custom settings** - Comment unusual configuration choices

## Troubleshooting

### Configuration not loading

```bash
## Test your configuration by running the worker
../bin/goworker --config ./config.toml --max-documents 1

## Check TOML syntax (requires toml-cli or similar tool)
## Or use an online TOML validator
```toml

### Performance issues

- Reduce `batch_size` if memory is limited
- Decrease `workers` if CPU is overloaded
- Monitor goroutine worker performance with `--workers` flag
- Adjust job control `claim_timeout` if workers are slow

## See Also

- [CLI Reference](cli.md) - Command-line interface documentation
- [Data Sources](configuration/sources.md) - Detailed source configuration
- [Storage Backends](configuration/storage.md) - Storage options and configuration
- [Embeddings Guide](features/embeddings/README.md) - Embedding configuration details
---

## Related Documentation

- **Previous**: [Getting Started](../getting-started/README.md)
- **Next**: [Content Sources](sources.md)
- **Up**: [Documentation Home](../README.md)

### Quick Links

- [Documentation Home](../README.md)
- [Quick Reference](../../QUICK_REFERENCE.md)
- [CLI Reference](../reference/cli.md)
- [Troubleshooting](../operations/troubleshooting.md)
