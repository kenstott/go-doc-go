# Go-Doc-Go Configuration Guide

This guide covers all configuration options for Go-Doc-Go. Configuration is provided via YAML files, typically named `config.yaml`.

## Configuration File Structure

The configuration file has several main sections:

```yaml
storage:          # Storage backend configuration
embedding:        # Embedding generation settings
content_sources:  # Document sources to process
relationship_detection:  # Relationship extraction settings
processing:       # Processing pipeline configuration
analytics:        # Analytics output configuration
logging:          # Logging configuration
```

## Storage Configuration

Configure where processed documents and metadata are stored.

### SQLite (Default)

```yaml
storage:
  backend: "sqlite"
  path: "./documents.db"

  # Optional SQLite-specific settings
  enable_fts: true           # Full-text search
  page_size: 4096
  cache_size: 10000
  journal_mode: "WAL"
```

### PostgreSQL

```yaml
storage:
  backend: "postgresql"
  host: "localhost"
  port: 5432
  database: "go_doc_go"
  username: "postgres"
  password: "${DB_PASSWORD}"  # Use environment variable

  # Connection pooling
  pool_size: 20
  max_overflow: 30
  pool_timeout: 30

  # Performance tuning
  statement_timeout: 300000
  batch_size: 1000
```

### MongoDB

```yaml
storage:
  backend: "mongodb"
  host: "localhost"
  port: 27017
  database: "go_doc_go"
  username: "${MONGO_USER}"
  password: "${MONGO_PASSWORD}"

  # Connection options
  max_pool_size: 100
  min_pool_size: 10

  # Collections
  documents_collection: "documents"
  elements_collection: "elements"
  relationships_collection: "relationships"
```

### Elasticsearch

```yaml
storage:
  backend: "elasticsearch"
  hosts:
    - "http://localhost:9200"

  # Authentication
  username: "elastic"
  password: "${ELASTIC_PASSWORD}"

  # Index configuration
  index_prefix: "go-doc-go"
  number_of_shards: 3
  number_of_replicas: 1
```

## Embedding Configuration

Configure how text embeddings are generated for semantic search.

### FastEmbed (Recommended)

```yaml
embedding:
  enabled: true
  provider: "fastembed"
  model: "BAAI/bge-small-en-v1.5"
  dimensions: 384

  # Chunking settings
  chunk_size: 512
  overlap: 128

  # Contextual embeddings (GraphRAG-lite)
  contextual: true
  predecessor_count: 1
  successor_count: 1
  include_metadata: true

  # Performance
  batch_size: 64
  max_sequence_length: 512
```

### HuggingFace Transformers

```yaml
embedding:
  enabled: true
  provider: "huggingface"
  model: "sentence-transformers/all-MiniLM-L6-v2"
  dimensions: 384

  # Device configuration
  device: "cuda"  # or "cpu", "mps"
  precision: "float16"
```

### OpenAI

```yaml
embedding:
  enabled: true
  provider: "openai"
  model: "text-embedding-ada-002"
  dimensions: 1536
  api_key: "${OPENAI_API_KEY}"

  # Rate limiting
  max_requests_per_minute: 3000
  max_tokens_per_minute: 1000000
```

## Content Sources

Configure where documents come from. Multiple sources can be configured.

### File System

```yaml
content_sources:
  - name: "local_docs"
    type: "file"
    base_path: "./documents"

    # File selection
    file_pattern: "**/*.{pdf,docx,md,txt}"
    include_patterns:
      - "**/*.pdf"
      - "**/*.docx"
    exclude_patterns:
      - "**/archive/**"
      - "**/temp/**"

    # Options
    watch_for_changes: true
    follow_links: true
    max_link_depth: 2

    # Discovery interval (seconds) - how often to check for new documents
    # Defaults to global --discovery-interval CLI option or 86400 (1 day)
    discovery_interval: 3600  # Check every hour
```

### Database

```yaml
content_sources:
  - name: "cms_content"
    type: "database"
    connection_string: "postgresql://user:pass@host/db"

    # Query for documents
    query: |
      SELECT id, title, content, created_at, author
      FROM articles
      WHERE status = 'published'

    # Field mapping
    field_mapping:
      doc_id: "id"
      title: "title"
      content: "content"
      metadata:
        author: "author"
        created: "created_at"

    # Processing options
    batch_size: 1000
    stream_results: true

    # Discovery interval for this database source
    discovery_interval: 21600  # Check every 6 hours
```

### Web Scraping

```yaml
content_sources:
  - name: "wiki_docs"
    type: "web"
    base_url: "https://wiki.example.com"

    # URLs to start from
    url_list:
      - "https://wiki.example.com/docs"

    # URL filtering
    include_patterns:
      - "^https://wiki.example.com/docs/"
    exclude_patterns:
      - "/archive/"
      - "/temp/"

    # Crawling settings
    max_link_depth: 2
    max_pages: 1000

    # Rate limiting
    delay: 1.0

    # Headers
    headers:
      User-Agent: "Go-Doc-Go/1.0"

    # Refresh interval (seconds)
    refresh_interval: 86400  # 1 day
```

### S3

```yaml
content_sources:
  - name: "s3_documents"
    type: "s3"
    bucket: "my-documents"
    prefix: "docs/"

    # AWS credentials
    aws_access_key_id: "${AWS_ACCESS_KEY}"
    aws_secret_access_key: "${AWS_SECRET_KEY}"
    region: "us-west-2"

    # Options
    include_extensions: [".pdf", ".docx", ".txt"]
    max_workers: 10
```

### SharePoint

```yaml
content_sources:
  - name: "sharepoint"
    type: "sharepoint"
    site_url: "https://company.sharepoint.com/sites/docs"

    # Authentication
    client_id: "${SHAREPOINT_CLIENT_ID}"
    client_secret: "${SHAREPOINT_CLIENT_SECRET}"
    tenant_id: "${SHAREPOINT_TENANT_ID}"

    # Libraries to scan
    libraries:
      - "Shared Documents"
      - "Project Files"
```

### Confluence

```yaml
content_sources:
  - name: "confluence"
    type: "confluence"
    url: "https://company.atlassian.net"
    username: "${CONFLUENCE_USER}"
    api_token: "${CONFLUENCE_TOKEN}"

    # Spaces to include
    spaces:
      - "TECH"
      - "PROD"

    # Content filtering
    include_page_types: ["page", "blogpost"]
    exclude_labels: ["draft", "obsolete"]
```

## Relationship Detection

Configure how relationships between elements are discovered.

```yaml
relationship_detection:
  enabled: true

  # Structural relationships (parent-child, siblings)
  structural: true

  # Semantic similarity relationships
  semantic: true
  similarity_threshold: 0.7

  # Cross-document relationships
  cross_document_semantic:
    enabled: true
    similarity_threshold: 0.75
    max_relationships_per_element: 10

  # Domain ontology relationships
  domain:
    enabled: true
    ontologies:
      - path: "./ontologies/financial.yaml"
        active: true
      - path: "./ontologies/technical.yaml"
        active: true
```

## Processing Configuration

Configure the document processing pipeline.

```yaml
processing:
  # Batch processing
  batch_size: 100
  max_workers: 4
  timeout_seconds: 300

  # Job control for distributed processing
  job_control:
    backend: "sqlite"  # or "postgresql"
    path: "./job_queue.db"

    # Worker coordination
    claim_timeout: 300        # 5 minutes
    heartbeat_interval: 30    # 30 seconds
    max_retries: 3

    # Dead letter queue
    enable_dlq: true
    dlq_threshold: 3

  # Discovery intervals
  # Per-source intervals can be configured in each content source
  # Priority: source-specific > CLI --discovery-interval > default
  default_discovery_interval: 86400  # Default: 1 day

  # Memory management
  max_memory_per_worker: "1GB"
  stream_large_files: true
  large_file_threshold: 10485760  # 10MB
```

## Analytics Configuration

Configure analytics outputs for monitoring and reporting.

```yaml
analytics:
  enabled: true

  outputs:
    # Parquet files for data warehouse
    - type: "parquet"
      path: "./analytics-output"
      partitioning: ["date", "source"]
      compression: "snappy"

    # SQLite for local analytics
    - type: "sqlite"
      path: "./analytics.db"
      tables: ["documents", "elements", "relationships", "embeddings", "metrics"]

    # PostgreSQL for production analytics
    - type: "postgresql"
      connection_string: "postgresql://user:pass@host/analytics"
      schema: "go_doc_go_analytics"
```

## Logging Configuration

Configure logging behavior.

```yaml
logging:
  level: "INFO"  # DEBUG, INFO, WARNING, ERROR, CRITICAL

  # Log outputs
  handlers:
    - type: "console"
      level: "INFO"
      format: "%(asctime)s - %(name)s - %(levelname)s - %(message)s"

    - type: "file"
      filename: "./logs/go_doc_go.log"
      level: "DEBUG"
      max_bytes: 10485760  # 10MB
      backup_count: 5

  # Component-specific logging
  loggers:
    go_doc_go.worker: "DEBUG"
    go_doc_go.storage: "INFO"
    go_doc_go.embeddings: "WARNING"
```

## Environment Variables

Configuration values can reference environment variables using `${VAR_NAME}` syntax:

```yaml
storage:
  backend: "postgresql"
  host: "${DB_HOST}"
  password: "${DB_PASSWORD}"
```

Set environment variables:

```bash
export DB_HOST=localhost
export DB_PASSWORD=secretpassword
```

Or use a `.env` file:

```bash
# .env
DB_HOST=localhost
DB_PASSWORD=secretpassword
```

## Configuration Profiles

Use different configurations for different environments:

### Development

```yaml
# config.dev.yaml
storage:
  backend: "sqlite"
  path: "./dev.db"

embedding:
  enabled: false  # Skip embeddings for faster development

processing:
  max_workers: 2
  batch_size: 10

logging:
  level: "DEBUG"
```

### Production

```yaml
# config.prod.yaml
storage:
  backend: "postgresql"
  host: "${PROD_DB_HOST}"
  database: "go_doc_go_prod"
  pool_size: 50

embedding:
  enabled: true
  provider: "fastembed"
  batch_size: 128

processing:
  max_workers: 16
  batch_size: 1000

logging:
  level: "WARNING"
```

## Validation

Validate your configuration:

```bash
# Check configuration syntax
python -m go_doc_go worker --config config.yaml --validate-only

# Test with small batch
python -m go_doc_go worker --config config.yaml --max-documents 10
```

## Common Configuration Patterns

### Minimal Configuration

```yaml
# Minimal config for testing
storage:
  backend: "sqlite"
  path: "./test.db"

content_sources:
  - name: "test"
    type: "file"
    base_path: "./test-docs"
```

### High-Performance Configuration

```yaml
# Optimized for performance
storage:
  backend: "postgresql"
  pool_size: 100
  batch_size: 5000

embedding:
  provider: "fastembed"
  batch_size: 256
  device: "cuda"

processing:
  max_workers: 32
  batch_size: 1000

analytics:
  outputs:
    - type: "parquet"
      compression: "snappy"
```

### Enterprise Configuration

```yaml
# Enterprise features
storage:
  backend: "postgresql"
  host: "${DB_CLUSTER}"
  ssl_mode: "require"

embedding:
  provider: "openai"
  api_key: "${OPENAI_API_KEY}"

content_sources:
  - name: "sharepoint"
    type: "sharepoint"
    # ... SharePoint config

  - name: "confluence"
    type: "confluence"
    # ... Confluence config

processing:
  job_control:
    backend: "postgresql"

logging:
  handlers:
    - type: "syslog"
      address: "${SYSLOG_SERVER}"
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
# Check file path
python -m go_doc_go worker --config ./config.yaml --validate-only

# Check YAML syntax
python -c "import yaml; yaml.safe_load(open('config.yaml'))"
```

### Missing dependencies

```bash
# Install storage backend
pip install "go-doc-go[db-postgresql]"

# Install embedding provider
pip install "go-doc-go[fastembed]"

# Install content source
pip install "go-doc-go[source-confluence]"
```

### Performance issues

- Reduce `batch_size` if memory is limited
- Decrease `max_workers` if CPU is overloaded
- Enable `stream_results` for large datasets
- Use `fastembed` instead of `huggingface` for faster embeddings

## See Also

- [CLI Reference](cli.md) - Command-line interface documentation
- [Data Sources](sources.md) - Detailed source configuration
- [Storage Backends](storage.md) - Storage options and configuration
- [Embeddings Guide](embeddings.md) - Embedding configuration details