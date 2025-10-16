# Getting Started with Go-Doc-Go

**Version 1.0** - Quick start guide to get you processing documents in minutes.

## 🚀 Quick Start (5 minutes)

### 1. Prerequisites

```bash
# Go 1.24 or later
go version

# Clone the repository
git clone https://github.com/kenstott/go-doc-go.git
cd go-doc-go
```

### 2. Build the Worker

```bash
cd go
go build -o ../bin/goworker ./cmd/worker
```

### 3. Create Configuration

```bash
cd ..
cat > config.toml << 'EOF'
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
enabled = false
EOF
```

### 4. Add Your Documents

```bash
mkdir -p docs
# Copy your PDF, DOCX, XLSX, or other documents to ./docs/
```

### 5. Run the Worker

```bash
./bin/goworker --config config.toml --workers 4
```

**That's it!** The worker will:
- Discover documents from `./docs`
- Parse them into UDML (Universal Document Markup Language)
- Extract elements, relationships, and metadata
- Store results in `./data/analytics.parquet`

---

## 📊 Query Your Data

### Using DuckDB

```bash
# Install DuckDB
brew install duckdb  # macOS
# or download from https://duckdb.org

# Query your documents
duckdb

D SELECT element_type, COUNT(*) as count
  FROM read_parquet('./data/analytics.parquet/elements/*.parquet')
  GROUP BY element_type
  ORDER BY count DESC;

# View document list
D SELECT doc_id, source_name, metadata
  FROM read_parquet('./data/analytics.parquet/documents/*.parquet')
  LIMIT 10;

# Search content
D SELECT element_id, element_type, content_preview
  FROM read_parquet('./data/analytics.parquet/elements/*.parquet')
  WHERE content_preview LIKE '%search term%'
  LIMIT 20;
```

### Using Python/Pandas

```python
import pandas as pd

# Load elements
elements = pd.read_parquet('./data/analytics.parquet/elements')

# View element types
print(elements.groupby('element_type').size())

# Filter by type
paragraphs = elements[elements['element_type'] == 'paragraph']
print(paragraphs[['element_id', 'content_preview']].head())

# Load relationships
relationships = pd.read_parquet('./data/analytics.parquet/relationships')
print(relationships.head())
```

---

## What You Just Got

### ✅ Document Processing Pipeline
- **Parse any document format** - PDF, DOCX, XLSX, PPTX, JSON, CSV, HTML, Markdown, XML
- **Extract structured content** - Elements with types (paragraph, table, header, etc.)
- **Capture relationships** - Parent-child hierarchy, references, links
- **Store in analytics format** - Columnar Parquet for fast querying

### ✅ Universal Document Model (UDML)
- **5 Element Categories**: Container, Content, Structure, Component, Metadata
- **Standard Relationships**: contains, references, next, links_to
- **Cross-format compatibility**: Query PDFs, Word docs, and Excel sheets identically

---

## Next Steps

### 1. Process from Multiple Sources

```toml
# Add to config.toml

# S3/MinIO
[[content_sources]]
name = "s3_docs"
type = "s3"
bucket = "my-documents"
prefix = "uploads/"
region = "us-east-1"

# Web scraping
[[content_sources]]
name = "documentation"
type = "web"
base_url = "https://docs.example.com"
follow_links = true
max_link_depth = 2
```

### 2. Enable Embeddings for Semantic Search

```toml
[embedding]
enabled = true
provider = "onnx"
model_path = "./models/all-MiniLM-L6-v2"

# Contextual embeddings
contextual = true
predecessor_count = 2
successor_count = 2
```

**Setup**:
```bash
# Export model to ONNX format (one-time setup)
pip install onnx sentence-transformers torch optimum[onnxruntime]

# Export using Python
python -c "
from optimum.onnxruntime import ORTModelForFeatureExtraction
from transformers import AutoTokenizer

model_id = 'sentence-transformers/all-MiniLM-L6-v2'
ort_model = ORTModelForFeatureExtraction.from_pretrained(model_id, export=True)
tokenizer = AutoTokenizer.from_pretrained(model_id)

ort_model.save_pretrained('./models/all-MiniLM-L6-v2')
tokenizer.save_pretrained('./models/all-MiniLM-L6-v2')
"

# Install ONNX Runtime library
pip install onnxruntime  # or onnxruntime-coreml for macOS

# Set library path (if not using system-wide install)
export ONNXRUNTIME_SHARED_LIBRARY_PATH=".venv/lib/python3.12/site-packages/onnxruntime/capi/libonnxruntime.1.23.0.dylib"
```

### 3. Export to Neo4j Graph Database

```toml
[processing.neo4j_export]
enabled = true
empty_queue_wait_time = 60

[processing.neo4j_export.connection]
uri = "bolt://localhost:7687"
username = "neo4j"
password = "password"
```

The worker will automatically export to Neo4j when the queue is empty.

### 4. Extract Domain Entities with Ontologies

```yaml
# ontologies/financial.yaml
name: financial
domain: business
version: "1.0"

element_entity_mappings:
  - domain: "business"
    entity_type: "Company"
    element_types: ["paragraph"]
    extraction_rules:
      - type: "regex_pattern"
        pattern: '\b([A-Z][a-z]+\s+(?:Inc|LLC|Corp|Corporation))\b'
```

```toml
# config.toml
[ontology]
enabled = true
schema_path = "./ontologies/financial.yaml"
```

### 5. Scale to Distributed Processing

```toml
# config.toml - use PostgreSQL for job coordination
[processing.job_control]
backend = "postgres"
path = "postgres://user:pass@db-server:5432/godocgo"
```

```bash
# Run multiple workers on different servers
# Server 1
./bin/goworker --config config.toml --worker-id "worker-01" --workers 8

# Server 2
./bin/goworker --config config.toml --worker-id "worker-02" --workers 8

# They coordinate automatically via PostgreSQL!
```

---

## Common Workflows

### Development: Process a few test documents

```bash
# Process just 10 documents to test configuration
./bin/goworker --config config.toml --max-documents 10
```

### Production: Continuous processing

```bash
# Run worker continuously, processing new documents as they appear
./bin/goworker --config config.toml --workers 8

# Or with Docker
docker run -v $(pwd)/config.toml:/etc/config.toml \
           -v $(pwd)/docs:/docs \
           -v $(pwd)/data:/data \
           godocgo/worker:latest --workers 8
```

### Analytics: Query processed data

```bash
# DuckDB - Fast SQL queries
duckdb
D FROM read_parquet('./data/analytics.parquet/elements/*.parquet') LIMIT 10;

# Python - Data science workflows
python
>>> import pandas as pd
>>> df = pd.read_parquet('./data/analytics.parquet/elements')
>>> df.head()
```

---

## Tips for Success

### Start Small
Test with a few documents first to verify configuration:
```bash
mkdir docs/test
# Add 2-3 test documents
./bin/goworker --config config.toml --max-documents 5
```

### Monitor Progress
```bash
# Check log output for processing status
./bin/goworker --config config.toml 2>&1 | tee worker.log

# In another terminal, watch the analytics output
watch -n 5 'ls -lh ./data/analytics.parquet/*/*.parquet'
```

### Choose Appropriate Storage
- **Development**: SQLite job control + Parquet analytics (simple, no setup)
- **Production**: PostgreSQL job control + Parquet analytics (scalable, reliable)
- **Graph queries**: Add Neo4j export for relationship analysis

### Verify Output
```bash
# Check that Parquet files were created
ls -la ./data/analytics.parquet/

# Quick stats with DuckDB
duckdb -c "SELECT COUNT(*) FROM read_parquet('./data/analytics.parquet/documents/*.parquet')"
duckdb -c "SELECT COUNT(*) FROM read_parquet('./data/analytics.parquet/elements/*.parquet')"
```

---

## Troubleshooting

### No documents found

```bash
# Verify file pattern matches your documents
ls docs/**/*.pdf

# Check configuration
cat config.toml | grep -A 5 content_sources

# Run with debug logging
./bin/goworker --config config.toml --max-documents 1
```

### Worker exits immediately

```bash
# Check if SQLite job control database exists
ls -la ./data/jobs.db

# Create data directory if needed
mkdir -p ./data
```

### Out of memory

```bash
# Reduce concurrent workers
./bin/goworker --config config.toml --workers 2

# Process in smaller batches
./bin/goworker --config config.toml --max-documents 100
```

---

## Documentation

- **[README](README.md)** - Project overview and features
- **[Go Implementation Guide](go/README.md)** - Detailed Go worker documentation
- **[UDML Specification](docs/UDML_SPECIFICATION.md)** - Complete UDML spec
- **[Configuration Reference](go/README.md#configuration-reference)** - All config options
- **[Ontology System](docs/ontology.md)** - Knowledge graph extraction
- **[Embeddings Guide](docs/embeddings.md)** - Semantic search setup

---

## What's Next?

1. **Process your real documents** - Point `base_path` to your actual document directory
2. **Enable embeddings** - Add semantic search capabilities
3. **Configure more sources** - Add S3, web scraping, databases
4. **Extract domain knowledge** - Set up ontologies for your industry
5. **Deploy at scale** - Use PostgreSQL and distribute across multiple workers

**Happy document processing! 🎉**