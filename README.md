wa w# Go-Doc-Go: Universal Document Knowledge Engine

**Version 1.0** - Transform any document into intelligent, searchable knowledge graphs at massive scale.

Go-Doc-Go implements **Universal Document Markup Language (UDML)** - a standardized framework that converts any document format (PDFs, Word, Excel, JSON, HTML, Markdown, etc.) into a consistent graph of elements with typed relationships. Your heterogeneous document corpus becomes a unified, queryable knowledge base.

## Quick Start (60 seconds)

### 1. Build the Worker

```bash
## Clone and build
git clone https://github.com/kenstott/go-doc-go.git
cd go-doc-go/go
go build -o ../bin/goworker ./cmd/worker
```toml

### 2. Create Configuration

```toml
## config.toml
[processing.job_control]
backend = "sqlite"
path = "./data/jobs.db"

[[content_sources]]
name = "documents"
type = "file"
base_path = "./docs"
file_pattern = "**/*.{pdf,docx,xlsx,json,html,md}"

[analytics]
enabled = true

[[analytics.outputs]]
type = "parquet"
path = "./data/analytics.parquet"
```bash

### 3. Process Documents

```bash
## Add your documents
mkdir -p docs
## Copy PDFs, DOCX, XLSX, etc. to docs/

## Run worker
../bin/goworker --config config.toml --workers 4
```bash

### 4. Query Results

```bash
## Query with DuckDB
duckdb
D SELECT element_type, COUNT(*) FROM read_parquet('./data/analytics.parquet/elements/*.parquet') GROUP BY element_type;
```bash

**Done!** See [Quick Reference](QUICK_REFERENCE.md) for common commands and troubleshooting.

---

## What Makes It Unique

### 🌐 Universal Document Model (UDML)

Converts **any document format** into a standardized 5-category taxonomy:

- **Container** - Pages, slides, sheets, sections (top-level structure)
- **Content** - Paragraphs, headers, text blocks (primary information)
- **Structure** - Tables, lists (organizational scaffolding)
- **Component** - Table cells, list items (leaf elements)
- **Metadata** - Comments, footnotes, chart data (supplementary info)

**Four standard relationship types** connect elements:
- `contains` - Hierarchical parent-child
- `references` - Cross-references and citations
- `next`/`previous` - Sequential ordering
- `links_to` - Hyperlinks

**Result**: Query PDFs, Word docs, and Excel sheets identically. Build cross-format analytics. Train AI models on unified vocabulary.

### 🚀 Massive Scale Architecture

**Horizontally scalable** distributed processing:
- Coordinate 10-50+ workers via PostgreSQL job control
- Process thousands of documents concurrently
- Single binary deployment (no runtime dependencies)
- Handle enterprise data lakes with millions of documents

### 🧠 Ontology-Driven Knowledge Graphs

Apply **business rules** to extract domain entities automatically:
- Define domain vocabulary (customers, products, regulations)
- Extract entities using semantic similarity, patterns, or keywords
- Discover relationships across documents
- Export to Neo4j for graph analytics

### 🎯 GraphRAG-lite Embeddings

**Contextual embeddings** using document structure:
- Elements embedded with awareness of neighbors, parents, children
- Vector search considers document hierarchy
- Better results than flat text embeddings
- Built on ONNX Runtime for performance

### 🔧 Storage Flexibility

Works with **multiple backends**:
- **Development**: Parquet files + SQLite
- **Production**: PostgreSQL + Parquet analytics
- **Graph Analytics**: Neo4j export
- **Vector Search**: ONNX embeddings

---

## Documentation

### Getting Started
- **[Quick Reference](QUICK_REFERENCE.md)** - Common commands, flags, troubleshooting (start here!)
- **[Getting Started Guide](docs/getting-started/README.md)** - Complete setup walkthrough

### Configuration
- **[Configuration Overview](docs/configuration/README.md)** - All configuration options
- **[Content Sources](docs/configuration/sources.md)** - Files, S3, web scraping
- **[Storage Backends](docs/configuration/storage.md)** - SQLite, PostgreSQL, Neo4j

### Features
- **[UDML Specification](docs/features/udml/specification.md)** - Complete UDML spec
- **[Ontology System](docs/features/ontology/README.md)** - Knowledge extraction
- **[Embeddings Guide](docs/features/embeddings/README.md)** - Contextual embeddings

### Operations
- **[Scaling Guide](docs/operations/scaling.md)** - Distributed workers, performance tuning
- **[Troubleshooting](docs/operations/troubleshooting.md)** - Common issues and solutions
- **[Monitoring](docs/operations/monitoring.md)** - Health checks and metrics

### Reference
- **[CLI Reference](docs/reference/cli.md)** - All command-line flags and options
- **[Architecture](docs/architecture/worker-design.md)** - System design and internals

---

## Performance

### Single Worker Throughput

| Document Type | Size   | Processing Time | Docs/Second     |
|---------------|--------|-----------------|-----------------|
| PDF (text)    | 1MB    | 50-100ms        | 10-20           |
| DOCX          | 500KB  | 30-80ms         | 12-33           |
| XLSX          | 2MB    | 100-200ms       | 5-10            |
| JSON          | 100KB  | 5-10ms          | 100-200         |
| HTML          | 50KB   | 3-8ms           | 125-333         |

### Distributed (10 Workers)

| Configuration            | Total Throughput | Docs/Hour   |
|--------------------------|------------------|-------------|
| Without embeddings       | 300+ docs/sec    | 1,080,000   |
| With ONNX embeddings     | 150 docs/sec     | 540,000     |

**Horizontal Scaling**: Add more workers to linearly increase throughput.

---

## Deployment

### Docker

```dockerfile
FROM golang:1.24 AS builder
WORKDIR /build
COPY go/ .
RUN go build -o worker ./cmd/worker

FROM ubuntu:22.04
COPY --from=builder /build/worker /usr/local/bin/
ENTRYPOINT ["/usr/local/bin/worker"]
CMD ["--config", "/etc/godocgo/config.toml"]
```toml

### Distributed Workers (PostgreSQL Coordination)

```toml
## config.toml - shared by all workers
[processing.job_control]
backend = "postgres"
path = "postgres://user:pass@db-server:5432/godocgo"
```bash

```bash
## Run on multiple servers
./bin/goworker --config config.toml --worker-id "worker-01" --workers 8  # Server 1
./bin/goworker --config config.toml --worker-id "worker-02" --workers 8  # Server 2
## Workers coordinate automatically via PostgreSQL
```toml

### Kubernetes

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: godocgo-worker
spec:
  replicas: 10
  template:
    spec:
      containers:
      - name: worker
        image: godocgo/worker:latest
        env:
        - name: NUM_WORKERS
          value: "4"
```

---

## Contributing

Contributions welcome! See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup.

## License

MIT License - see [LICENSE](LICENSE) for details.

---

**Ready to process millions of documents?** 🚀

```bash
cd go && go build -o ../bin/goworker ./cmd/worker
../bin/goworker --config config.toml --workers 8
```bash
