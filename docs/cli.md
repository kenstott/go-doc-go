# Go-Doc-Go CLI Reference

Go-Doc-Go provides a comprehensive command-line interface for document processing, analysis, and search. This reference covers all available commands and their options.

## Installation

```bash
pip install go-doc-go
```

## Basic Usage

```bash
python -m go_doc_go [COMMAND] [OPTIONS]
```

## Global Options

- `--help` - Show help message for any command

## Commands Overview

| Command | Purpose | Common Use Cases |
|---------|---------|------------------|
| `worker` | Process documents from configured sources | Ingestion, parsing, embedding generation |
| `search` | Search processed documents | Find content, filter by type, export results |
| `analytics` | View processing statistics | Monitor performance, check data completeness |
| `status` | Monitor processing status | Live monitoring, troubleshooting |
| `deadletter` | Manage failed documents | Retry failures, investigate issues |
| `ontology-generator` | Create domain ontologies | Define business concepts interactively |
| `ontology-extract` | Extract entities using ontology | Apply knowledge graph rules |
| `ontology-analytics` | Generate ontology from data | LLM-based rule discovery |

## Core Commands

### `worker` - Document Processing

The main document processing command that reads from configured sources and processes documents.

```bash
python -m go_doc_go worker [OPTIONS]
```

**Options:**
- `-c, --config PATH` - Configuration file path (default: `./config.yaml`)
- `--worker-id TEXT` - Custom worker ID (auto-generated if not provided)
- `-l, --log-level [debug|info|warning|error]` - Logging level
- `-m, --max-documents INTEGER` - Maximum documents to process before stopping
- `-d, --discovery-interval INTEGER` - Seconds between discovery cycles (default: 86400)

**Examples:**

```bash
# Basic processing with default config
python -m go_doc_go worker

# Process 100 documents with custom config
python -m go_doc_go worker --config production.yaml --max-documents 100

# Debug mode with custom worker ID
python -m go_doc_go worker --worker-id worker-01 --log-level debug

# Process with hourly discovery
python -m go_doc_go worker --discovery-interval 3600
```

### `search` - Document Search

Search through processed documents with various filters and output formats.

```bash
python -m go_doc_go search [QUERY] [OPTIONS]
```

**Options:**
- `-c, --config PATH` - Configuration file path
- `--include-types TEXT` - Element types to include (e.g., `paragraph,heading`)
- `--exclude-types TEXT` - Element types to exclude
- `--regex TEXT` - Filter content with regex pattern
- `--similarity-threshold FLOAT` - Minimum similarity (0.0-1.0)
- `-l, --limit INTEGER` - Maximum results (default: 10)
- `-o, --output [table|json|summary|markdown|html]` - Output format
- `-v, --verbose` - Enable verbose logging
- `--data-path PATH` - Override data lake path
- `--full-text` - Return full content (not previews)
- `--context [none|parents|siblings|children|all]` - Include context
- `--context-depth INTEGER` - Depth of context (default: 1)

**Examples:**

```bash
# Basic text search
python -m go_doc_go search "quarterly revenue"

# Search specific element types
python -m go_doc_go search "financial data" --include-types paragraph,heading

# Regex pattern search
python -m go_doc_go search --regex "Q[1-4].*2024"

# Export as JSON with full text
python -m go_doc_go search "analysis" --output json --full-text

# Semantic search with threshold
python -m go_doc_go search "machine learning" --similarity-threshold 0.7

# Include document context
python -m go_doc_go search "compliance" --context parents --context-depth 2
```

### `analytics` - Processing Analytics

View statistics and analytics about processed documents.

```bash
python -m go_doc_go analytics [OPTIONS]
```

**Options:**
- `-c, --config PATH` - Configuration file path
- `-d, --detailed` - Show detailed analytics with schemas
- `--json` - Output as JSON

**Examples:**

```bash
# Basic analytics summary
python -m go_doc_go analytics

# Detailed analytics with table schemas
python -m go_doc_go analytics --detailed

# Export analytics as JSON
python -m go_doc_go analytics --json

# Custom configuration
python -m go_doc_go analytics --config production.yaml
```

### `status` - Live Status Monitoring

Monitor document processing status in real-time.

```bash
python -m go_doc_go status [OPTIONS]
```

**Options:**
- `-c, --config PATH` - Configuration file path
- `-d, --detailed` - Show detailed status with logs
- `-f, --follow` - Follow log file (like `tail -f`)
- `-n, --lines INTEGER` - Lines to show when following (default: 10)
- `--refresh INTEGER` - Refresh interval in seconds (default: 5)

**Examples:**

```bash
# Current status snapshot
python -m go_doc_go status

# Detailed status with recent logs
python -m go_doc_go status --detailed

# Follow logs in real-time
python -m go_doc_go status --follow

# Follow with more context
python -m go_doc_go status --follow --lines 50

# Custom refresh rate
python -m go_doc_go status --refresh 2
```

## Ontology Commands

### `ontology-generator` - Interactive Ontology Creation

Create domain-specific ontologies interactively.

```bash
python -m go_doc_go ontology-generator [OPTIONS]
```

**Options:**
- `-o, --output PATH` - Output file path for ontology YAML
- `-d, --domain TEXT` - Domain name (e.g., financial, medical)
- `--interview-mode` - Use interactive interview mode

**Examples:**

```bash
# Interactive ontology creation
python -m go_doc_go ontology-generator --interview-mode

# Generate financial ontology
python -m go_doc_go ontology-generator --domain financial --output financial.yaml
```

### `ontology-extract` - Entity Extraction

Extract entities and relationships using configured ontologies.

```bash
python -m go_doc_go ontology-extract [OPTIONS]
```

**Options:**
- `-c, --config PATH` - Configuration file with ontology settings
- `--ontology PATH` - Path to ontology YAML file
- `--input PATH` - Input document or directory
- `--output PATH` - Output path for extracted entities

**Examples:**

```bash
# Extract using configured ontology
python -m go_doc_go ontology-extract --config config.yaml

# Use specific ontology file
python -m go_doc_go ontology-extract --ontology domain.yaml --input docs/
```

### `ontology-analytics` - LLM-Based Ontology Generation

Generate ontology rules automatically using LLM analysis of existing data.

```bash
python -m go_doc_go ontology-analytics [OPTIONS]
```

**Options:**
- `-c, --config PATH` - Configuration file path
- `--sample-size INTEGER` - Number of documents to analyze
- `--output PATH` - Output path for generated ontology

**Examples:**

```bash
# Analyze data and generate ontology
python -m go_doc_go ontology-analytics --sample-size 100 --output generated.yaml
```

## Dead Letter Queue Management

### `deadletter` - Manage Failed Documents

Handle documents that failed processing.

```bash
python -m go_doc_go deadletter [SUBCOMMAND] [OPTIONS]
```

**Subcommands:**
- `list` - List failed documents
- `retry` - Retry failed documents
- `clear` - Clear dead letter queue
- `inspect` - Inspect specific failed document

**Examples:**

```bash
# List all failed documents
python -m go_doc_go deadletter list

# Retry all failed documents
python -m go_doc_go deadletter retry

# Inspect specific failure
python -m go_doc_go deadletter inspect --doc-id doc_123

# Clear dead letter queue
python -m go_doc_go deadletter clear --confirm
```

## Configuration

All commands use a YAML configuration file. The default location is `./config.yaml`, but can be overridden with the `-c/--config` option or the `GO_DOC_GO_CONFIG_PATH` environment variable.

### Configuration Precedence

1. Command-line `--config` option
2. `GO_DOC_GO_CONFIG_PATH` environment variable
3. Default `./config.yaml`

### Example Configuration

```yaml
# config.yaml
storage:
  backend: postgresql
  host: localhost
  database: go_doc_go

content_sources:
  - name: documents
    type: file
    base_path: ./docs

embedding:
  enabled: true
  provider: fastembed
  model: BAAI/bge-small-en-v1.5

analytics:
  backend: parquet
  output_path: ./analytics-output
```

## Output Formats

Most commands support multiple output formats:

- **table** - Human-readable table (default)
- **json** - Machine-readable JSON
- **markdown** - Markdown formatted output
- **html** - HTML formatted output
- **summary** - Condensed summary view

## Environment Variables

- `GO_DOC_GO_CONFIG_PATH` - Default configuration file path
- `GO_DOC_GO_LOG_LEVEL` - Default logging level
- `GO_DOC_GO_DATA_PATH` - Override data storage path

## Common Workflows

### Initial Document Ingestion

```bash
# 1. Configure your sources in config.yaml
# 2. Run worker to process documents
python -m go_doc_go worker --max-documents 1000

# 3. Check processing status
python -m go_doc_go status --detailed

# 4. View analytics
python -m go_doc_go analytics
```

### Continuous Processing

```bash
# Run worker continuously with daily discovery
python -m go_doc_go worker --discovery-interval 86400 --log-level info
```

### Search and Export

```bash
# Search for content
python -m go_doc_go search "important topic" --limit 50

# Export results as JSON for further processing
python -m go_doc_go search "data analysis" --output json > results.json
```

### Monitoring and Troubleshooting

```bash
# Live monitoring
python -m go_doc_go status --follow

# Check for failures
python -m go_doc_go deadletter list

# Retry failed documents
python -m go_doc_go deadletter retry
```

## Tips and Best Practices

1. **Start Small**: Use `--max-documents` to test configuration with a small batch
2. **Monitor Progress**: Keep `status --follow` running in another terminal
3. **Check Failures**: Regularly review dead letter queue for issues
4. **Use JSON Output**: For scripting and automation, use `--output json`
5. **Enable Debug Logging**: Use `--log-level debug` when troubleshooting
6. **Test Search**: Verify search works before large-scale ingestion

## Getting Help

- Use `--help` with any command for detailed options
- Check logs with `status --follow` for processing issues
- Review configuration with `analytics --detailed`

## See Also

- [Configuration Guide](configuration.md) - Detailed configuration options
- [Data Sources](sources.md) - Configuring content sources
- [Storage Backends](storage.md) - Storage configuration
- [Ontology System](ontology.md) - Knowledge graph configuration