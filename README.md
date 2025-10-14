# Go-Doc-Go: Universal Document Knowledge Engine

**Version 1.0** - Transform any document into intelligent, searchable knowledge graphs at massive scale.

## What Makes It Unique

### 🌐 Universal Document Graph Model
Converts **any document** (PDFs, Word docs, Excel sheets, JSON, HTML, Markdown) into a standardized graph of elements with relationships. Your heterogeneous data becomes a unified, queryable structure.

### 🚀 Massive Scale Data Ingestion
**Horizontally scalable pipeline** built for enterprise workloads:
- Process thousands of documents concurrently
- Distributed work queues with PostgreSQL coordination
- Single binary deployment - no runtime dependencies
- Handle everything from single files to enterprise data lakes

### 🔧 Universal Storage Flexibility
Works with **multiple storage backends** - you choose what fits your needs:
- **Development**: Parquet files, SQLite
- **Production**: PostgreSQL job control, Parquet analytics
- **Graph Analytics**: Neo4j export
- **Vector Search**: ONNX Runtime embeddings with contextual awareness

### 🧠 Ontology-Driven Knowledge Graphs
Apply **business rules** to automatically extract domain entities and relationships:
- Define what matters in your domain (customers, products, regulations, components)
- Extract entities using semantic similarity, patterns, or keywords
- Discover relationships across documents automatically
- Build true knowledge graphs from your document corpus

### 🎯 GraphRAG-lite Embeddings
Smart **contextual embeddings** that use document structure to improve semantic search:
- Elements know about their neighbors, parents, and children through "graph-lets"
- Vector search considers document hierarchy and context
- Better results than flat text embeddings
- Built on ONNX Runtime for maximum performance

## Mental Model

```
Any Data Source → Universal Graph → Knowledge Graph → Analytics
     │                    │               │              │
 Documents            Elements &        Domain       Parquet/Neo4j
   Files             Relationships     Entities       Embeddings
  Database
    API
```

## The Universal Document Model (UDML)

Go-Doc-Go implements **Universal Document Markup Language (UDML)** - a standardized framework that destructures **any document format** into a consistent 5-category taxonomy with standard relationship types.

### The Five Element Categories

Every element in every document—regardless of source format—is classified into exactly one of five categories:

| Category | Description | Examples | Use Case |
|----------|-------------|----------|----------|
| **Container** | Top-level organizational units that group content | `page`, `slide`, `sheet`, `section`, `article` | Document structure, navigation, chunking |
| **Content** | Primary textual information blocks | `paragraph`, `header`, `text_box`, `blockquote` | Reading, search, embedding generation |
| **Structure** | Organizational scaffolding | `table`, `list`, `table_row` | Layout understanding, data extraction |
| **Component** | Leaf elements within structures | `table_cell`, `list_item`, `shape` | Granular data access, cell-level queries |
| **Metadata** | Supplementary information | `comment`, `footnote`, `slide_notes`, `chart` | Context, provenance, annotations |

### Standard Relationships

Documents are connected through universal relationship types:

1. **contains** - Hierarchical parent-child relationships
2. **references** - Cross-references and citations
3. **next** / **previous** - Sequential ordering
4. **links_to** - Hyperlinks and external references

This consistent model creates a **unified knowledge backbone** for:
- 📊 **Cross-format analytics** - Query slides, sheets, and pages identically
- 🤖 **AI model training** - Single vocabulary across all document types
- 🔍 **Universal search** - Find content regardless of original format
- 📈 **Knowledge graphs** - Seamless relationship discovery across sources

---

## Real-World Impact

**Financial Services**: "We process 10,000+ earnings transcripts to automatically extract company-executive-metric relationships, turning months of analyst work into automated knowledge graphs."

**Manufacturing**: "Our safety compliance docs become queryable knowledge - instantly find which components must comply with which standards across 50,000+ technical documents."

**Legal**: "Contract analysis at scale - extract parties, obligations, and terms from thousands of agreements, then discover patterns and risks automatically."

## Quick Start

### Installation

```bash
# Clone the repository
git clone https://github.com/kenstott/go-doc-go.git
cd go-doc-go/go

# Build the worker binary
go build -o ../bin/goworker ./cmd/worker
```

### Basic Configuration

```toml
# config.toml
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

[embedding]
enabled = false  # Start without embeddings
```

### Process Documents

```bash
# Run the worker
../bin/goworker --config config.toml --workers 4
```

**That's it!** The worker will:
1. Discover documents from `./docs`
2. Parse them into UDML elements and relationships
3. Store results in Parquet format
4. Exit when queue is empty

### Query Your Data

```bash
# Using DuckDB to query Parquet output
duckdb
D SELECT element_type, COUNT(*) as count
  FROM read_parquet('./data/analytics.parquet/elements/*.parquet')
  GROUP BY element_type
  ORDER BY count DESC;

# View documents
D SELECT doc_id, source_name, metadata
  FROM read_parquet('./data/analytics.parquet/documents/*.parquet')
  LIMIT 10;
```

### Export to Neo4j

```toml
# Add Neo4j export to config.toml
[processing.neo4j_export]
enabled = true
empty_queue_wait_time = 60

[processing.neo4j_export.connection]
uri = "bolt://localhost:7687"
username = "neo4j"
password = "password"
```

The worker will automatically export to Neo4j when the queue is idle.

---

## Core Capabilities

- **📄 Universal Parsing**: PDF, DOCX, PPTX, XLSX, HTML, Markdown, JSON, CSV, XML, Parquet, plain text
- **🔌 Flexible Sources**: Files, S3/MinIO, Web/HTTP (databases coming soon)
- **🏗️ Scalable Architecture**: Distributed processing, horizontal scaling, work queue coordination
- **📊 Analytics Storage**: Parquet (columnar), Neo4j (graph), SQLite (development)
- **🧠 Knowledge Extraction**: Ontology-based entity extraction and relationship discovery
- **⚡ Performance**: ONNX Runtime embeddings, concurrent processing, optimized for large datasets
- **🔧 Single Binary**: No runtime dependencies, statically compiled Go

## Architecture

Go-Doc-Go is built on three core pillars:

### 1. **Massive Input** - Ingest from anywhere
- File systems (local, network, cloud)
- S3 and S3-compatible storage (MinIO, DigitalOcean Spaces)
- Web/HTTP with link following
- Coming soon: Databases, SharePoint, Confluence, Google Drive

### 2. **Flexible Storage** - Store however works best
- **Parquet** - Columnar format for analytics (recommended)
- **Neo4j** - Graph database for relationship queries
- **SQLite** - Development and testing
- **PostgreSQL** - Job control and coordination

### 3. **Smart Output** - Knowledge, not just data
- Contextual vector embeddings using document structure
- Automated ontology-based entity extraction
- Full document reconstruction and format conversion
- Cross-format analytics and queries

## Documentation

- **[Quick Start Guide](docs/QUICK_START.md)** - Get started in 5 minutes
- **[UDML Specification](docs/UDML_SPECIFICATION.md)** - Complete UDML spec
- **[Configuration Reference](go/README.md)** - All configuration options
- **[Installation Guide](docs/installation.md)** - Detailed installation
- **[Ontology System](docs/ontology.md)** - Knowledge graph extraction
- **[Embeddings Guide](docs/embeddings.md)** - Contextual embeddings setup
- **[Scaling Guide](docs/scaling.md)** - Horizontal scaling patterns

## Performance

### Document Processing (without embeddings)

| Document Type | Size   | Processing Time | Throughput      |
|---------------|--------|-----------------|-----------------|
| PDF (text)    | 1MB    | 50-100ms        | 10-20 docs/sec  |
| DOCX          | 500KB  | 30-80ms         | 12-33 docs/sec  |
| XLSX          | 2MB    | 100-200ms       | 5-10 docs/sec   |
| JSON          | 100KB  | 5-10ms          | 100-200 docs/sec|
| HTML          | 50KB   | 3-8ms           | 125-333 docs/sec|
| Markdown      | 10KB   | 1-3ms           | 333-1000 docs/sec|

### With ONNX Embeddings

| Batch Size | Time per Batch | Max Throughput  |
|------------|----------------|-----------------|
| 32         | 100-200ms      | 160-320 docs/sec|

### Distributed Processing (10 Workers)

| Setup                    | Total Throughput | Documents/Hour |
|--------------------------|------------------|----------------|
| 10 workers, no embeddings| 300+ docs/sec    | 1,080,000      |
| 10 workers, with embeddings | 150 docs/sec  | 540,000        |

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
```

```bash
docker build -t godocgo/worker:latest .
docker run -v $(pwd)/config.toml:/etc/godocgo/config.toml godocgo/worker:latest
```

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
        - name: GO_DOC_GO_CONFIG_PATH
          value: /config/config.toml
        - name: NUM_WORKERS
          value: "4"
```

### Distributed Workers

```toml
# config.toml - shared by all workers
[processing.job_control]
backend = "postgres"
path = "postgres://user:pass@db-server:5432/godocgo"
```

```bash
# Worker 1 (server-01)
./bin/goworker --config config.toml --worker-id "worker-01" --workers 4

# Worker 2 (server-02)
./bin/goworker --config config.toml --worker-id "worker-02" --workers 4

# All workers coordinate via PostgreSQL automatically
```

---

## Examples

### Processing Local Documents

```toml
[[content_sources]]
name = "local_documents"
type = "file"
base_path = "/data/documents"
file_pattern = "**/*.{pdf,docx,xlsx,pptx,html,md,txt,json,csv,xml}"
```

### Processing from S3

```toml
[[content_sources]]
name = "s3_documents"
type = "s3"
bucket = "my-documents"
prefix = "uploads/"
region = "us-east-1"
```

### Web Scraping

```toml
[[content_sources]]
name = "documentation"
type = "web"
base_url = "https://docs.example.com"
follow_links = true
max_link_depth = 3
```

### Ontology Extraction

```yaml
# ontologies/companies.yaml
name: companies
domain: business
version: "1.0"

element_entity_mappings:
  - domain: "business"
    entity_type: "Organization"
    element_types: ["paragraph"]
    extraction_rules:
      - type: "regex_pattern"
        pattern: '\b([A-Z][a-z]+(?:\s+[A-Z][a-z]+)*\s+(?:Inc|LLC|Corp))\b'
```

```toml
[ontology]
enabled = true
schema_path = "./ontologies/companies.yaml"
```

---

## Contributing

Contributions welcome! See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup and guidelines.

## License

MIT License - see [LICENSE](LICENSE) for details.

---

**Ready to process millions of documents?** 🚀

Build the worker and start parsing:

```bash
cd go
go build -o ../bin/goworker ./cmd/worker
../bin/goworker --config ../config.toml --workers 4
```