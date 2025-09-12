# Installation Guide

Go-Doc-Go supports flexible, modular installation where you only install the components you need.

## Quick Start

```bash
# Minimal installation (core functionality)
pip install go-doc-go

# Production setup with PostgreSQL + fast embeddings  
pip install "go-doc-go[db-postgresql,fastembed]"

# Everything included
pip install "go-doc-go[all]"
```

## Modular Installation Options

### Database Backends

```bash
# SQLite with extensions
pip install "go-doc-go[db-core]"

# PostgreSQL support
pip install "go-doc-go[db-postgresql]"

# MongoDB support
pip install "go-doc-go[db-mongodb]"

# Neo4j graph database
pip install "go-doc-go[db-neo4j]"

# Elasticsearch 
pip install "go-doc-go[db-elasticsearch]"

# MySQL/MariaDB
pip install "go-doc-go[db-mysql]"

# Oracle Enterprise
pip install "go-doc-go[db-oracle]"

# Microsoft SQL Server
pip install "go-doc-go[db-mssql]"

# All database backends
pip install "go-doc-go[db-all]"
```

### Content Sources

```bash
# Database content sources (SQL/NoSQL)
pip install "go-doc-go[source-database]"

# Confluence wiki
pip install "go-doc-go[source-confluence]"

# JIRA issues
pip install "go-doc-go[source-jira]"

# Google Drive (auto-exports Office docs)
pip install "go-doc-go[source-gdrive]"

# Microsoft SharePoint
pip install "go-doc-go[source-sharepoint]"

# ServiceNow platform
pip install "go-doc-go[source-servicenow]"

# MongoDB collections
pip install "go-doc-go[source-mongodb]"

# All content sources
pip install "go-doc-go[source-all]"
```

### Embedding Providers

```bash
# HuggingFace/PyTorch models
pip install "go-doc-go[huggingface]"

# OpenAI API embeddings
pip install "go-doc-go[openai]"

# FastEmbed (15x faster than transformers)
pip install "go-doc-go[fastembed]"

# All embedding providers
pip install "go-doc-go[embedding-all]"
```

### Cloud & Additional Components

```bash
# AWS S3 and cloud services
pip install "go-doc-go[cloud-aws]"

# Scientific libraries (NumPy, etc.)
pip install "go-doc-go[scientific]"

# Additional document parsing utilities
pip install "go-doc-go[document_parsing]"
```

## Recommended Configurations

### Development Setup
```bash
pip install "go-doc-go[db-core,fastembed]"
```
- SQLite with extensions for local development
- FastEmbed for fast, local embeddings
- No external dependencies required

### Production Setup
```bash
pip install "go-doc-go[db-postgresql,fastembed,cloud-aws]"
```
- PostgreSQL for production reliability
- FastEmbed for performance
- AWS S3 support for cloud storage

### Enterprise Setup
```bash
pip install "go-doc-go[db-all,source-all,embedding-all,cloud-aws]"
```
- All storage backends available
- All content sources supported
- Multiple embedding providers
- Full cloud integration

### High-Performance Search
```bash
pip install "go-doc-go[db-elasticsearch,fastembed,scientific]"
```
- Elasticsearch for optimized search
- FastEmbed for fast embeddings
- Scientific libraries for advanced analytics

## System Requirements

### Minimum Requirements
- Python 3.9+
- 4GB RAM
- 1GB disk space

### Recommended Requirements
- Python 3.11+
- 8GB+ RAM (for large document processing)
- 10GB+ disk space
- SSD storage for better performance

### For Large Scale Processing
- Python 3.11+
- 16GB+ RAM
- 50GB+ disk space
- Dedicated database server
- Multiple CPU cores for parallel processing

## Verification

After installation, verify your setup:

```python
from go_doc_go import Config
print("Go-Doc-Go installed successfully!")

# Check available backends
config = Config()
available_backends = config.get_available_storage_backends()
print(f"Available storage backends: {available_backends}")

# Check embedding providers
available_embeddings = config.get_available_embedding_providers()
print(f"Available embedding providers: {available_embeddings}")
```

## Troubleshooting

### Common Issues

**ImportError for optional dependencies**
```bash
# Install the specific extra you need
pip install "go-doc-go[db-postgresql]"  # for PostgreSQL
pip install "go-doc-go[source-confluence]"  # for Confluence
```

**Slow embeddings**
```bash
# Switch to FastEmbed for 15x speed improvement
pip install "go-doc-go[fastembed]"
```

**Memory issues with large documents**
```yaml
# config.yaml - reduce memory usage
embedding:
  batch_size: 32  # default is 64
  max_content_length: 1000  # truncate long documents
```

**PostgreSQL connection issues**
```bash
# Ensure PostgreSQL client libraries are installed
pip install psycopg2-binary
# or for development
pip install psycopg2
```

### Docker Setup

For containerized deployment:

```dockerfile
FROM python:3.11-slim

# Install system dependencies
RUN apt-get update && apt-get install -y \
    gcc \
    libpq-dev \
    && rm -rf /var/lib/apt/lists/*

# Install Go-Doc-Go
RUN pip install "go-doc-go[db-postgresql,fastembed]"

# Copy your configuration
COPY config.yaml /app/config.yaml
WORKDIR /app

# Run ingestion
CMD ["python", "-m", "go_doc_go", "ingest", "config.yaml"]
```

## Next Steps

- [Configuration Guide](configuration.md) - Configure your setup
- [Data Sources](sources.md) - Connect your data sources
- [Storage Backends](storage.md) - Choose your storage backend
- [Quick Start Tutorial](../README.md#quick-start) - Get started with basic usage