# Getting Started with Go-Doc-Go

## 🚀 Quick Start (2 minutes)

```bash
# 1. Install Go-Doc-Go
pip install go-doc-go

# 2. Create a basic configuration
cat > config.yaml << EOF
storage:
  backend: sqlite
  path: ./documents.db

content_sources:
  - name: my_docs
    type: file
    base_path: ./documents
EOF

# 3. Process your documents
python -m go_doc_go worker --max-documents 10

# 4. Search your documents
python -m go_doc_go search "important topic"
```

That's it! You're now processing and searching documents.

## What You Just Got

### ✅ Document Processing Pipeline
- **Parse any document format** - PDF, DOCX, HTML, Markdown, and more
- **Extract structured content** - Elements, metadata, relationships
- **Generate embeddings** - For semantic search
- **Store in queryable format** - SQLite, PostgreSQL, MongoDB, etc.

### ✅ Command-Line Tools
- **`worker`** - Process documents from configured sources
- **`search`** - Search processed documents
- **`analytics`** - View processing statistics
- **`status`** - Monitor processing in real-time

## First Steps

### 1. Check Everything is Working

```bash
# Show available commands
python -m go_doc_go --help

# Check your configuration
python -m go_doc_go analytics
```

### 2. Add Your Documents

Create a directory with some documents:

```bash
mkdir documents
# Add PDFs, Word docs, text files, etc. to this directory
```

### 3. Process Documents

```bash
# Process all documents
python -m go_doc_go worker

# Or process a specific number
python -m go_doc_go worker --max-documents 100

# Monitor progress in another terminal
python -m go_doc_go status --follow
```

### 4. Search Your Documents

```bash
# Basic search
python -m go_doc_go search "quarterly revenue"

# Search with filters
python -m go_doc_go search "project plan" --include-types heading,paragraph

# Export results
python -m go_doc_go search "analysis" --output json > results.json
```

## Configuration Basics

### Minimal Configuration

```yaml
# config.yaml - Minimal setup
storage:
  backend: sqlite
  path: ./documents.db

content_sources:
  - name: local_docs
    type: file
    base_path: ./documents
```

### Adding Embeddings for Semantic Search

```yaml
# config.yaml - With semantic search
storage:
  backend: sqlite
  path: ./documents.db

content_sources:
  - name: local_docs
    type: file
    base_path: ./documents

embedding:
  enabled: true
  provider: fastembed
  model: BAAI/bge-small-en-v1.5
```

### Processing from Multiple Sources

```yaml
# config.yaml - Multiple sources
storage:
  backend: sqlite
  path: ./documents.db

content_sources:
  # Local files
  - name: local_docs
    type: file
    base_path: ./documents

  # Database content
  - name: cms_content
    type: database
    connection_string: postgresql://user:pass@host/db
    query: SELECT id, title, content FROM articles

  # Web scraping
  - name: wiki
    type: web
    url_list:
      - https://wiki.example.com/docs
    max_link_depth: 2
```

## Common Workflows

### Initial Setup and Processing

```bash
# 1. Install with desired features
pip install "go-doc-go[fastembed]"  # Include fast embeddings

# 2. Create configuration
cp config.yaml.example config.yaml
# Edit config.yaml with your settings

# 3. Test with small batch
python -m go_doc_go worker --max-documents 10

# 4. Check results
python -m go_doc_go analytics

# 5. Process everything
python -m go_doc_go worker
```

### Daily Usage

```bash
# Start processing new documents
python -m go_doc_go worker

# Search for information
python -m go_doc_go search "meeting notes from January"

# Check processing status
python -m go_doc_go status --detailed

# View analytics
python -m go_doc_go analytics --detailed
```

### Monitoring and Troubleshooting

```bash
# Live monitoring
python -m go_doc_go status --follow

# Check for failures
python -m go_doc_go deadletter list

# Retry failed documents
python -m go_doc_go deadletter retry

# Debug mode
python -m go_doc_go worker --log-level debug --max-documents 1
```

## File Structure

After running Go-Doc-Go, your directory will contain:

```
📁 Project Directory
├── 📄 config.yaml           # Your configuration
├── 🗄️ documents.db          # SQLite database (if using SQLite)
├── 📁 documents/            # Your source documents
├── 📁 analytics-output/     # Analytics data (if configured)
├── 📁 logs/                 # Processing logs
└── 🗄️ job_queue.db          # Job coordination database
```

## Tips for Success

### Start Small
Test with a few documents first:
```bash
python -m go_doc_go worker --max-documents 5 --log-level debug
```

### Use Appropriate Storage
- **Development**: SQLite (simple, no setup)
- **Production**: PostgreSQL (scalable, reliable)
- **Search-heavy**: Elasticsearch (optimized for search)

### Enable Embeddings for Better Search
```yaml
embedding:
  enabled: true
  provider: fastembed  # 15x faster than transformers
```

### Monitor Progress
Keep a status window open:
```bash
watch -n 5 'python -m go_doc_go status'
```

### Check the Logs
```bash
python -m go_doc_go status --follow --lines 50
```

## Next Steps

### Explore More Commands

```bash
# See all available commands
python -m go_doc_go --help

# Get help for specific commands
python -m go_doc_go search --help
python -m go_doc_go worker --help
```

### Advanced Configuration

- **[Configuration Guide](docs/configuration.md)** - All configuration options
- **[Data Sources](docs/sources.md)** - Configure different content sources
- **[Storage Backends](docs/storage.md)** - Choose the right storage
- **[CLI Reference](docs/cli.md)** - Complete command reference

### Scale Up

- Process from databases, APIs, cloud storage
- Deploy distributed workers for large-scale processing
- Configure knowledge graph extraction with ontologies
- Set up production monitoring and analytics

## Troubleshooting

### Common Issues

**No documents found**
```bash
# Check your base_path in config.yaml
# Verify file patterns match your documents
python -m go_doc_go worker --log-level debug
```

**Search returns no results**
```bash
# Ensure documents were processed successfully
python -m go_doc_go analytics

# Check if embeddings are enabled for semantic search
grep "embedding:" config.yaml
```

**Processing is slow**
```yaml
# config.yaml - Optimize performance
processing:
  batch_size: 100    # Increase batch size
  max_workers: 8     # Use more CPU cores

embedding:
  provider: fastembed  # Use faster embedding provider
  batch_size: 64      # Process embeddings in batches
```

### Getting Help

- Use `--help` with any command for options
- Check logs: `python -m go_doc_go status --follow`
- Review failed documents: `python -m go_doc_go deadletter list`

## What's Next?

1. **Process your real documents** - Point to your actual document directory
2. **Configure more sources** - Add databases, APIs, cloud storage
3. **Enable knowledge extraction** - Set up ontologies for your domain
4. **Deploy for production** - Scale with PostgreSQL and distributed workers

**Happy document processing! 🎉**