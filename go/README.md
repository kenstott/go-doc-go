# Go-Doc-Go: Go Implementation

**Version 1.0** - High-performance document processing worker built in Go for enterprise-scale document ingestion and knowledge extraction.

## Why Go?

### 📦 **Single Binary Deployment**
- **Zero dependencies**: Statically compiled binary (except ONNX Runtime for embeddings)
- **Cross-platform**: Linux, macOS, Windows from same codebase
- **Container-friendly**: 15-30MB Docker images
- **Instant startup**: No interpreter, no warm-up time

### ⚡ **Performance**
- **Native concurrency**: Go goroutines enable efficient parallel processing
- **Memory efficient**: <100MB base memory per worker
- **True parallelism**: Full multi-core utilization
- **Fast parsing**: 100-1000+ docs/sec depending on format
- **Compiled binary**: No runtime overhead

### 🔧 **Production Ready**
- **Distributed processing**: PostgreSQL-coordinated work queues
- **Atomic operations**: Row-level locking for safe multi-worker coordination
- **Auto-recovery**: Heartbeat monitoring with automatic claim timeout
- **Horizontal scaling**: Run 10, 100, or 1000 workers identically

---

## Quick Start (60 seconds)

### 1. **Build the Worker**

```bash
cd go
go build -o ../bin/goworker ./cmd/worker
```

### 2. **Create Configuration**

```bash
cat > ../config.toml << 'EOF'
[processing.job_control]
backend = "sqlite"
path = "./data/jobs.db"

[[content_sources]]
name = "documents"
type = "file"
base_path = "./docs"
file_pattern = "**/*.{pdf,docx,xlsx}"

[analytics]
enabled = true

[[analytics.outputs]]
type = "parquet"
path = "./data/analytics.parquet"

[embedding]
enabled = false  # Start without embeddings
EOF
```

### 3. **Run**

```bash
../bin/goworker --config ../config.toml
```

**That's it!** The worker will:
1. Discover documents from `./docs`
2. Parse them into UDML elements and relationships
3. Store results in `./data/analytics.parquet`
4. Exit when queue is empty

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                    Go Worker Binary                         │
├─────────────────────────────────────────────────────────────┤
│  Content Sources  │  Parsers       │  Analytics             │
│  ┌──────────────┐ │ ┌────────────┐ │ ┌──────────────┐      │
│  │ File         │ │ │ PDF        │ │ │ Parquet      │      │
│  │ S3/MinIO     │ │ │ DOCX       │ │ │ Neo4j        │      │
│  │ Web/HTTP     │ │ │ XLSX       │ │ │ SQLite       │      │
│  └──────────────┘ │ │ PPTX       │ │ └──────────────┘      │
│                   │ │ JSON       │ │                        │
│  Job Control      │ │ CSV        │ │  Embeddings            │
│  ┌──────────────┐ │ │ HTML       │ │ ┌──────────────┐      │
│  │ SQLite       │ │ │ Markdown   │ │ │ ONNX Runtime │      │
│  │ PostgreSQL   │ │ │ XML        │ │ └──────────────┘      │
│  └──────────────┘ │ │ Text       │ │                        │
│                   │ │ Parquet    │ │                        │
│                   │ └────────────┘ │                        │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
              Unified Document Graph Model
              (UDML Elements + Relationships)
```

---

## Installation & Building

### Prerequisites

```bash
# Go 1.24 or later
go version

# Optional: For embeddings
# - ONNX Runtime (C++ library)
```

### Build Options

#### **Standard Build** (SQLite only)

```bash
cd go
go build -o ../bin/goworker ./cmd/worker
```

**Pros**: Single binary, works everywhere
**Cons**: Only SQLite job control
**Size**: ~25MB

#### **Full Build** (PostgreSQL support)

```bash
cd go
CGO_ENABLED=1 go build -o ../bin/goworker ./cmd/worker
```

**Pros**: PostgreSQL job control for distributed workers
**Cons**: Requires CGO, platform-specific
**Size**: ~30MB

---

## Configuration Reference

### Complete Example

```toml
# config.toml - Production configuration

# ============================================================
# Processing Configuration
# ============================================================

# Job Control - Coordinates distributed workers
[processing.job_control]
backend = "postgres"  # "sqlite" or "postgres"

# SQLite path (for backend = "sqlite")
path = "./data/jobs.db"

# PostgreSQL DSN (for backend = "postgres")
# path = "postgres://user:pass@localhost:5432/godocgo?sslmode=disable"

claim_timeout = 300           # Seconds before claimed doc is released
heartbeat_interval = 30       # Seconds between heartbeats
max_retries = 3              # Max processing attempts per document

# Neo4j Export - Auto-export when queue is empty
[processing.neo4j_export]
enabled = true
empty_queue_wait_time = 300  # Seconds to wait before triggering export
batch_size = 1000           # Batch size for Neo4j import

[processing.neo4j_export.connection]
uri = "bolt://localhost:7687"
username = "neo4j"
password = "password"
database = "neo4j"

# ============================================================
# Content Sources - Where to find documents
# ============================================================

# File System Source
[[content_sources]]
name = "local_documents"
type = "file"
base_path = "/data/documents"
file_pattern = "**/*.{pdf,docx,xlsx,pptx,html,md,txt,json,csv,xml}"

# S3/MinIO Source
[[content_sources]]
name = "s3_documents"
type = "s3"
bucket = "documents"
prefix = "uploads/"
region = "us-east-1"
# endpoint = "http://localhost:9000"  # For MinIO

# Web/HTTP Source
[[content_sources]]
name = "web_documents"
type = "web"
base_url = "https://example.com/docs"
follow_links = true
max_link_depth = 3

# ============================================================
# Analytics - Where to store parsed data
# ============================================================
[analytics]
enabled = true

# Parquet Output (Recommended for performance)
[[analytics.outputs]]
type = "parquet"
path = "./data/analytics.parquet"

# Neo4j Output (Direct write to graph database)
[[analytics.outputs]]
type = "neo4j"
uri = "bolt://localhost:7687"
username = "neo4j"
password = "password"
database = "neo4j"
batch_size = 1000

# ============================================================
# Embeddings - Vector embeddings for semantic search
# ============================================================
[embedding]
enabled = true
provider = "onnx"
model_path = "./models/bge-small-en-v1.5"

# Contextual embeddings - include neighbor elements
contextual = true
predecessor_count = 2  # Include 2 preceding elements
successor_count = 2    # Include 2 following elements

# ============================================================
# Ontology - Domain entity extraction
# ============================================================
[ontology]
enabled = true
schema_path = "./ontologies/financial.yaml"
queue_idle_trigger_minutes = 5
min_interval_minutes = 60

# ============================================================
# Relationship Detection - Semantic relationships
# ============================================================
[relationship_detection]
enabled = true
structural = true
semantic = true

[relationship_detection.cross_document_semantic]
enabled = true
similarity_threshold = 0.85
queue_idle_trigger_minutes = 5
```

### Minimal Configuration (Development)

```toml
[processing.job_control]
backend = "sqlite"
path = "./data/jobs.db"

[[content_sources]]
name = "documents"
type = "file"
base_path = "./docs"

[analytics]
enabled = true

[[analytics.outputs]]
type = "parquet"
path = "./data/analytics.parquet"

[embedding]
enabled = false
```

---

## Running the Worker

### Basic Usage

```bash
# Use config file
../bin/goworker --config config.toml

# Set number of concurrent goroutine workers (default: 1)
../bin/goworker --config config.toml --workers 4

# Process limited number of documents
../bin/goworker --config config.toml --max-documents 100

# Custom worker ID
../bin/goworker --config config.toml --worker-id "worker-prod-01"
```

### Distributed Workers (PostgreSQL)

```toml
# config.toml - shared by all workers
[processing.job_control]
backend = "postgres"
path = "postgres://user:pass@db-server:5432/godocgo?sslmode=disable"
```

```bash
# Worker 1 (server-01)
../bin/goworker --config config.toml --worker-id "worker-01" --workers 4

# Worker 2 (server-02)
../bin/goworker --config config.toml --worker-id "worker-02" --workers 4

# Worker 3 (server-03)
../bin/goworker --config config.toml --worker-id "worker-03" --workers 4

# All workers coordinate via PostgreSQL - no conflicts!
```

**Key Features**:
- **Atomic claiming**: PostgreSQL row-level locking prevents duplicate processing
- **Heartbeat monitoring**: Dead workers automatically release claims
- **Auto-recovery**: Failed documents retry up to `max_retries`
- **Leader election**: First worker handles discovery, others process

---

## Embedding Options

### Option 1: ONNX Runtime (Recommended)

**Performance**: ~100-500ms per batch
**Dependency**: ⚠️ Requires ONNX Runtime shared library
**Batching**: Automatically aggregates embeddings across documents

```toml
[embedding]
enabled = true
provider = "onnx"
model_path = "./models/all-MiniLM-L6-v2"

# Contextual embeddings include neighboring elements
contextual = true
predecessor_count = 2
successor_count = 2
```

#### Step 1: Export Model to ONNX Format

```bash
# Install required packages
pip install onnx sentence-transformers torch

# Export model
python scripts/export_model_to_onnx.py

# Or export a different model:
python scripts/export_model_to_onnx.py \
  "sentence-transformers/all-MiniLM-L6-v2" \
  "go/models/all-MiniLM-L6-v2"
```

#### Step 2: Install ONNX Runtime Library

```bash
# Find the library in your Python environment
find .venv -name "libonnxruntime*.dylib" -o -name "libonnxruntime*.so"

# Set environment variable (macOS example)
export ONNXRUNTIME_SHARED_LIBRARY_PATH=".venv/lib/python3.12/site-packages/onnxruntime/capi/libonnxruntime.1.23.0.dylib"

# Or install system-wide
# macOS (Homebrew)
brew install onnxruntime

# Linux (Ubuntu/Debian)
wget https://github.com/microsoft/onnxruntime/releases/download/v1.23.0/onnxruntime-linux-x64-1.23.0.tgz
tar -xzf onnxruntime-linux-x64-1.23.0.tgz
sudo cp onnxruntime-linux-x64-1.23.0/lib/libonnxruntime.so* /usr/local/lib/
sudo ldconfig
```

#### Step 3: Run Worker with Embeddings

```bash
# Set ONNX Runtime path
export ONNXRUNTIME_SHARED_LIBRARY_PATH=".venv/lib/python3.12/site-packages/onnxruntime/capi/libonnxruntime.1.23.0.dylib"

# Run worker
../bin/goworker --config config.toml
```

### Option 2: Disabled (Fastest)

```toml
[embedding]
enabled = false
```

Generate embeddings later as a separate batch process from Parquet/SQLite output.

---

## Performance Benchmarks

### Document Processing (No Embeddings)

| Document Type | Size   | Processing Time | Throughput      |
|---------------|--------|-----------------|-----------------|
| PDF (text)    | 1MB    | 50-100ms        | 10-20 docs/sec  |
| DOCX          | 500KB  | 30-80ms         | 12-33 docs/sec  |
| XLSX          | 2MB    | 100-200ms       | 5-10 docs/sec   |
| JSON          | 100KB  | 5-10ms          | 100-200 docs/sec|
| HTML          | 50KB   | 3-8ms           | 125-333 docs/sec|
| Markdown      | 10KB   | 1-3ms           | 333-1000 docs/sec|

**Concurrency scaling** (4-core machine):
- 1 worker: 10 docs/sec
- 2 workers: 18 docs/sec
- 4 workers: 32 docs/sec
- 8 workers: 35 docs/sec (I/O bound)

### With Embeddings

| Provider      | Batch Size | Time per Batch | Max Throughput  |
|---------------|------------|----------------|-----------------|
| ONNX Runtime  | 32         | 100-200ms      | 160-320 docs/sec|

**Memory usage**:
- Base worker: ~50MB
- Per goroutine: ~2MB
- ONNX Runtime: +200MB (model loaded)

### Distributed Processing (10 Workers)

| Setup                    | Total Throughput | Documents/Hour |
|--------------------------|------------------|----------------|
| 10 workers, no embeddings| 300+ docs/sec    | 1,080,000      |
| 10 workers, ONNX embed   | 150 docs/sec     | 540,000        |

**Tested on**: AWS c6i.2xlarge (8 vCPU, 16GB RAM) × 10 instances

---

## Content Source Support

### File System
```toml
[[content_sources]]
name = "documents"
type = "file"
base_path = "/data/docs"
file_pattern = "**/*.{pdf,docx,xlsx}"
```

**Features**:
- Recursive directory traversal
- Glob pattern matching with `**` support
- Follows symlinks

### S3 / MinIO
```toml
[[content_sources]]
name = "s3_docs"
type = "s3"
bucket = "documents"
prefix = "uploads/"
region = "us-east-1"
```

**Features**:
- AWS S3 and S3-compatible (MinIO, DigitalOcean Spaces, etc.)
- Credential chain (env vars, instance profile, config file)
- Automatic pagination for large buckets

### Web / HTTP
```toml
[[content_sources]]
name = "web_docs"
type = "web"
base_url = "https://example.com/docs"
follow_links = true
max_link_depth = 3
```

**Features**:
- HTTP/HTTPS fetching
- Link following with depth control
- Pattern-based filtering

---

## Parser Support

All parsers are **fully implemented in pure Go** and statically compiled into the worker binary.

### Supported Formats

| Format     | Parser          | Dependencies | Element Types                    |
|------------|-----------------|--------------|----------------------------------|
| PDF        | `ledongthuc/pdf`| None         | page, paragraph, header, table   |
| DOCX       | Native          | None         | paragraph, header, table, list   |
| XLSX       | `xuri/excelize` | None         | sheet, table, table_row, cell    |
| PPTX       | Native          | None         | slide, text_box, shape, table    |
| JSON       | Native          | None         | object, array, key_value         |
| CSV        | Native          | None         | table, table_row, cell           |
| HTML       | `goquery`       | None         | div, p, h1-h6, table, list       |
| Markdown   | Native          | None         | paragraph, header, list, code    |
| XML        | `xmlquery`      | None         | element, attribute, text         |
| Text       | Native          | None         | paragraph, line                  |
| Parquet    | Apache Arrow Go | None         | schema, row_group, column        |

---

## Analytics Storage Options

### Parquet (Recommended)
```toml
[[analytics.outputs]]
type = "parquet"
path = "./data/analytics.parquet"
```

**Features**:
- Columnar storage for fast analytics
- Automatic compression (Snappy)
- Schema versioning with category enrichment
- Compatible with: Pandas, DuckDB, Spark, BigQuery

**Output Structure**:
```
analytics.parquet/
├── documents/
│   └── part-0001.parquet
├── elements/
│   └── part-0001.parquet
├── relationships/
│   └── part-0001.parquet
└── embeddings/
    └── part-0001.parquet
```

### Neo4j (Direct Write)
```toml
[[analytics.outputs]]
type = "neo4j"
uri = "bolt://localhost:7687"
username = "neo4j"
password = "password"
batch_size = 1000
```

**Features**:
- Direct write to graph database
- Batch inserts with configurable size
- Automatic index creation
- MERGE operations (idempotent)

**Schema**:
```cypher
// Nodes
(:Document {doc_id, source_name, metadata})
(:Element {element_id, element_type, element_category, content_preview})

// Relationships
(:Element)-[:CONTAINS]->(:Element)
(:Element)-[:NEXT]->(:Element)
(:Element)-[:REFERENCES]->(:Element)
```

---

## Deployment Patterns

### Pattern 1: Single Server (Development)

```bash
# Single worker, SQLite job control
../bin/goworker --config config.toml --workers 4
```

**Use when**:
- Development/testing
- <10,000 documents
- Single machine

### Pattern 2: Distributed (Multi-Server)

```toml
# config.toml - shared via NFS or config management
[processing.job_control]
backend = "postgres"
path = "postgres://user:pass@db.example.com/godocgo"
```

```bash
# Server 1
../bin/goworker --config config.toml --worker-id "srv-01" --workers 8

# Server 2
../bin/goworker --config config.toml --worker-id "srv-02" --workers 8

# Server 3
../bin/goworker --config config.toml --worker-id "srv-03" --workers 8
```

**Use when**:
- Production at scale
- 100,000+ documents
- High availability required

### Pattern 3: Kubernetes

```yaml
# deployment.yaml
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

**Use when**:
- Cloud-native deployment
- Auto-scaling needed
- Millions of documents

---

## Docker Deployment

### Dockerfile

```dockerfile
FROM golang:1.24 AS builder
WORKDIR /build
COPY go/ .
RUN go build -o worker ./cmd/worker

FROM ubuntu:22.04
COPY --from=builder /build/worker /usr/local/bin/
ENV ONNXRUNTIME_SHARED_LIBRARY_PATH=/usr/local/lib/libonnxruntime.so
ENTRYPOINT ["/usr/local/bin/worker"]
CMD ["--config", "/etc/godocgo/config.toml"]
```

### Build and Run

```bash
# Build image
docker build -t godocgo/worker:latest .

# Run with config
docker run -v $(pwd)/config.toml:/etc/godocgo/config.toml \
           -v $(pwd)/data:/data \
           godocgo/worker:latest
```

---

## Troubleshooting

### Worker Not Finding Documents

```bash
# Check content source config
cat config.toml | grep -A 10 content_sources

# Test file pattern manually
find ./docs -name "*.pdf"
```

### ONNX Runtime Not Found

```bash
# Check library path
ls -la $ONNXRUNTIME_SHARED_LIBRARY_PATH

# Set explicitly
export ONNXRUNTIME_SHARED_LIBRARY_PATH=/usr/local/lib/libonnxruntime.so
```

**Solution**: Disable embeddings if not needed:

```toml
[embedding]
enabled = false
```

### PostgreSQL Connection Errors

```bash
# Test connection
psql "postgres://user:pass@host:5432/godocgo?sslmode=disable"

# Check PostgreSQL is running
pg_isready -h host -p 5432
```

---

## Development

### Project Structure

```
go/
├── cmd/
│   ├── worker/           # Worker entry point
│   ├── csvparser/        # Standalone CSV parser
│   ├── docxparser/       # Standalone DOCX parser
│   └── ...              # Other standalone parsers
├── internal/
│   ├── analytics/        # Analytics storage (Parquet, Neo4j)
│   ├── cache/           # LRU and memory-mapped caching
│   ├── contentsource/   # Content source implementations
│   ├── detector/        # Document type detection
│   ├── embeddings/      # Embedding generation (ONNX)
│   ├── export/          # Graph export to Neo4j
│   ├── jobcontrol/      # Job control (SQLite, PostgreSQL)
│   ├── ontology/        # Ontology-based extraction
│   ├── parser/          # All document parsers
│   ├── resolver/        # Content resolver
│   ├── temporal/        # Temporal analysis
│   ├── udml/            # UDML query and builder
│   └── worker/          # Worker orchestration
├── go.mod               # Go module definition
└── go.sum               # Dependency checksums
```

### Running Tests

```bash
cd go

# Run all tests
go test ./...

# Run specific package tests
go test ./internal/parser

# Run with coverage
go test -cover ./...

# Run with race detector
go test -race ./...
```

### Building for Multiple Platforms

```bash
# Linux AMD64
GOOS=linux GOARCH=amd64 go build -o worker-linux-amd64 ./cmd/worker

# Linux ARM64 (AWS Graviton)
GOOS=linux GOARCH=arm64 go build -o worker-linux-arm64 ./cmd/worker

# macOS Intel
GOOS=darwin GOARCH=amd64 go build -o worker-darwin-amd64 ./cmd/worker

# macOS Apple Silicon
GOOS=darwin GOARCH=arm64 go build -o worker-darwin-arm64 ./cmd/worker

# Windows
GOOS=windows GOARCH=amd64 go build -o worker-windows-amd64.exe ./cmd/worker
```

---

## Command-Line Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--config` | string | `./config.toml` | Path to configuration file |
| `--worker-id` | string | auto-generated | Custom worker ID (hostname_pid) |
| `--workers` | int | 1 | Number of concurrent goroutine workers |
| `--max-documents` | int | 0 | Maximum documents to process (0=unlimited) |

---

## Support & Contributing

### Getting Help

- 📚 **Documentation**: [/docs](../docs/)
- 🐛 **Issues**: [GitHub Issues](https://github.com/kenstott/go-doc-go/issues)

### Contributing

```bash
# Fork repo
git clone https://github.com/YOUR_USERNAME/go-doc-go.git
cd go-doc-go/go

# Create branch
git checkout -b feature/my-feature

# Make changes and run tests
go test ./...
go fmt ./...

# Submit PR
git push origin feature/my-feature
```

---

## License

MIT License - see [LICENSE](../LICENSE) for details.

---

## Acknowledgments

Built with:
- [excelize](https://github.com/xuri/excelize) - XLSX parsing
- [goquery](https://github.com/PuerkitoBio/goquery) - HTML parsing
- [ledongthuc/pdf](https://github.com/ledongthuc/pdf) - PDF parsing
- [Apache Arrow Go](https://github.com/apache/arrow/tree/main/go) - Parquet I/O
- [onnxruntime_go](https://github.com/yalue/onnxruntime_go) - ONNX Runtime bindings
- [neo4j-go-driver](https://github.com/neo4j/neo4j-go-driver) - Neo4j integration

---

**Ready to process millions of documents?** Build the worker and start parsing! 🚀

```bash
go build -o ../bin/goworker ./cmd/worker
../bin/goworker --config ../config.toml --workers 4
```