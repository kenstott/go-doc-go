# Go-Doc-Go: Universal Document Knowledge Engine

**Think of it as a universal translator for unstructured data** - transforms documents from any source into intelligent, searchable knowledge graphs at massive scale.

## What Makes It Unique

### 🌐 Universal Document Graph Model
Converts **any document** (PDFs, Word docs, databases, APIs, emails) into a standardized graph of elements with relationships. Your heterogeneous data becomes a unified, queryable structure.

### 🚀 Massive Scale Data Ingestion  
**Horizontally scalable pipeline** that can move huge volumes of data quickly:
- Process thousands of documents concurrently
- Distributed work queues with PostgreSQL coordination
- Handle everything from single files to enterprise data lakes
- Database TEXT/VARCHAR fields, cloud storage, APIs - if it has unstructured data, we can ingest it

### 🔧 Universal Storage Flexibility
Works with **almost any storage backend** - you choose what fits your needs:
- **Development**: SQLite, File-based
- **Production**: PostgreSQL, MongoDB, Elasticsearch  
- **Graph Analytics**: Neo4j integration
- **Vector Search**: pgvector, Elasticsearch vectors
- **Enterprise**: Oracle, SQL Server, MySQL

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
- Foundation for full GraphRAG implementations using graph-lets

## Mental Model

```
Any Data Source → Universal Graph → Knowledge Graph → Smart Search
     │                    │               │              │
 Documents            Elements &        Domain         GraphRAG-lite
 Databases           Relationships     Entities &      Embeddings
 APIs                                  Rules
```

## The Universal Document Model

Go-Doc-Go's **groundbreaking Universal Document Model** is a patent-pending framework that destructures **any document format** into a consistent 5-category taxonomy with 7 standard relationship types. This revolutionary approach enables unprecedented cross-format analysis, knowledge extraction, and AI-powered insights.

### The Five Element Categories

Every element in every document—regardless of source format—is classified into exactly one of five categories:

| Category | Description | Examples | Use Case |
|----------|-------------|----------|----------|
| **Container** | Top-level organizational units that group content | `page`, `slide`, `sheet`, `section`, `article` | Document structure, navigation, chunking |
| **Content** | Primary textual information blocks | `paragraph`, `header`, `text_box`, `blockquote` | Reading, search, embedding generation |
| **Structure** | Organizational scaffolding | `table`, `list`, `table_row` | Layout understanding, data extraction |
| **Component** | Leaf elements within structures | `table_cell`, `list_item`, `shape` | Granular data access, cell-level queries |
| **Metadata** | Supplementary information | `comment`, `footnote`, `slide_notes`, `chart` | Context, provenance, annotations |

### Seven Standard Relationships

Documents are connected through seven universal relationship types:

1. **contains** / **contained_by** - Hierarchical parent-child relationships
2. **references** / **referenced_by** - Cross-references and citations
3. **next** / **previous** - Sequential ordering
4. **links_to** - Hyperlinks and external references

This consistent model creates a **unified knowledge backbone** for:
- 📊 **Cross-format analytics** - Query slides, sheets, and pages identically
- 🤖 **AI model training** - Single vocabulary across all document types
- 🔍 **Universal search** - Find content regardless of original format
- 📈 **Knowledge graphs** - Seamless relationship discovery across sources

### Category-Based Queries

The Universal Document Model enables powerful category-based queries:

```python
# Find all top-level containers across ALL document types
containers = db.query(element_category="container")
# Returns: pages from PDFs, slides from PPTX, sheets from XLSX

# Get only textual content for embedding
content = db.query(element_category="content")
# Returns: paragraphs, headers, text blocks - no tables or images

# Extract structured data across formats
structures = db.query(element_category="structure")
# Returns: tables from Word, Excel, presentations - unified
```

### Extensible Taxonomy

The element taxonomy is defined in `element_taxonomy.json`, making it:

✅ **Easy to update** - Edit JSON, no code changes
✅ **Version controlled** - Track taxonomy evolution
✅ **Portable** - Shared between Go and Python implementations
✅ **Self-documenting** - Clear category descriptions and element mappings

Example taxonomy entry:
```json
{
  "container": {
    "description": "Top-level organizational units",
    "element_types": ["page", "slide", "sheet", "section"]
  }
}
```

### Automatic Enrichment

Categories are **automatically applied** during document processing—no manual tagging required:

1. Parser extracts elements with specific types (`slide`, `paragraph`, `table_cell`)
2. Analytics layer enriches with categories using taxonomy lookup
3. Storage persists both type and category for maximum flexibility

This **zero-overhead enrichment** means every document gets categorized automatically, creating instant cross-format compatibility.

### Patent-Pending Innovation

The Universal Document Model represents a **fundamental breakthrough** in document understanding:

- **Unified representation** of inherently different formats (paginated vs. gridded vs. hierarchical)
- **Semantic categories** that transcend format-specific nomenclature
- **Automatic inference** of document structure without ML models
- **Bidirectional mapping** preserving original types while enabling universal queries

This innovation enables **true format-agnostic knowledge extraction**—the foundation for next-generation document AI and enterprise knowledge graphs.

---

## Real-World Impact

**Financial Services**: "We process 10,000+ earnings transcripts to automatically extract company-executive-metric relationships, turning months of analyst work into automated knowledge graphs."

**Manufacturing**: "Our safety compliance docs become queryable knowledge - instantly find which components must comply with which standards across 50,000+ technical documents."

**Legal**: "Contract analysis at scale - extract parties, obligations, and terms from thousands of agreements, then discover patterns and risks automatically."

## Quick Start

### Installation
```bash
# Minimal setup
pip install go-doc-go

# Production with PostgreSQL + fast embeddings  
pip install "go-doc-go[db-postgresql,fastembed]"

# Everything (all sources, storage, embeddings)
pip install "go-doc-go[all]"
```

### Basic Configuration
```yaml
# config.yaml
storage:
  backend: "postgresql"  # or sqlite, elasticsearch, mongodb
  
embedding:
  enabled: true
  provider: "fastembed"  # 15x faster than transformers
  model: "BAAI/bge-small-en-v1.5"

content_sources:
  - name: "documents"
    type: "file" 
    base_path: "./docs"
  - name: "database"
    type: "database"
    connection_string: "postgresql://user:pass@host/db"
    query: "SELECT id, title, content FROM articles"
```

### Process Documents
```python
from go_doc_go import Config, ingest_documents

config = Config("config.yaml")
result = ingest_documents(config)

print(f"Processed {result['documents']} documents")
print(f"Created {result['elements']} elements") 
print(f"Found {result['relationships']} relationships")
```

### Search with Context
```python
from go_doc_go import SearchEngine, SearchRequest

# Unified search interface with contextual XML embeddings
engine = SearchEngine()
results = engine.search(SearchRequest(
    query_text="quarterly revenue analysis",
    limit=10,
    document_format="markdown"  # Get full documents as markdown
)

for item in results.results:
    print(f"Found: {item.content_preview}")
    print(f"Similarity: {item.similarity:.3f}")

# Access full reconstructed documents
for doc_id, doc in results.materialized_documents.items():
    print(f"Document: {doc.title}")
    print(f"Content: {doc.formatted_content}")
```

### Build Knowledge Graphs
```python
# Define your domain ontology
# ontologies/financial.yaml
```yaml
name: financial
entities:
  company:
    patterns: ["NASDAQ:\\w+", "NYSE:\\w+"] 
    semantic: "company corporation business"
  executive:
    patterns: ["CEO", "CFO", "CTO"]
    semantic: "chief executive officer president"
  metric:
    semantic: "revenue profit margin growth"
    
relationships:
  - source: executive
    target: metric  
    type: "discusses"
    constraints:
      same_document: true
```

```python
# Process with domain extraction
config = Config("config.yaml")  # includes ontology path
result = ingest_documents(config)

# Query extracted knowledge
companies = db.get_entities(entity_type="company")
for company in companies:
    executives = db.get_related_entities(company, "employs")
    metrics = db.get_related_entities(company, "reports")
    print(f"{company.name}: {len(executives)} execs, {len(metrics)} metrics")
```

## Command-Line Interface

Go-Doc-Go provides **simple, config-driven CLI tools** for document processing and monitoring:

### 🚀 **Document Processing**
- **Process all sources** defined in your config.yaml
- **Process specific sources** with filtering
- **Validate configuration** before processing
- **Batch processing** with configurable workers and timeouts

### 📊 **Status & Monitoring**
- **Real-time status** of processing and storage
- **Follow logs** for live updates (like tail -f)
- **Storage information** with size and statistics
- **File-based monitoring** without database dependencies

### 📈 **Analytics & Reporting**
- **Analytics summary** from configured outputs
- **Storage backend** information and statistics
- **Parquet, SQLite, JSON** output analysis
- **Detailed file listings** and metadata

### 🛠️ **CLI Tools Available**

```bash
# Document processing
PYTHONPATH=src python -m go_doc_go.cli.process --help

# Process all content sources in config
PYTHONPATH=src python -m go_doc_go.cli.process

# Process specific sources only
PYTHONPATH=src python -m go_doc_go.cli.process --sources wikipedia,documents

# Validate configuration without processing
PYTHONPATH=src python -m go_doc_go.cli.process --validate-only

# Status and monitoring
PYTHONPATH=src python -m go_doc_go.cli.status

# Follow logs in real-time
PYTHONPATH=src python -m go_doc_go.cli.status --follow

# Analytics summary
PYTHONPATH=src python -m go_doc_go.cli.analytics

# Detailed analytics with file listings
PYTHONPATH=src python -m go_doc_go.cli.analytics --detailed
```

### 🚀 **Getting Started with CLI**

```bash
# Process documents using your config.yaml
PYTHONPATH=src python -m go_doc_go.cli.process

# Check processing status and storage information
PYTHONPATH=src python -m go_doc_go.cli.status

# View analytics from configured outputs
PYTHONPATH=src python -m go_doc_go.cli.analytics
```

**Key Features:**
- 🖥️ **Config-driven** - all configuration in single config.yaml file
- 📊 **Rich formatting** - human-readable tables and status displays
- ⚡ **File-based monitoring** - no database dependencies for status
- 🔧 **Comprehensive help** - detailed help for every command and option
- 📈 **Analytics integration** - supports parquet, SQLite, JSON outputs

**Simplified Architecture:**
- **Single config.yaml** - eliminates pipeline database complexity
- **Direct processing** - process content sources without job orchestration
- **File-based status** - monitoring via logs and analytics outputs
- **Declarative configuration** - infrastructure as code approach

> **Note**: Web UI has been deprecated in favor of the more powerful and automation-friendly CLI interface. UI code preserved in `deprecated/frontend_*` for reference.

## Core Capabilities

- **📄 Universal Parsing**: PDF, DOCX, PPTX, XLSX, HTML, Markdown, JSON, CSV, XML, plain text
- **🔌 Flexible Sources**: Files, databases, S3, SharePoint, Confluence, APIs, Google Drive
- **🏗️ Scalable Architecture**: Distributed processing, horizontal scaling, work queue coordination
- **🔍 Smart Search**: Semantic, structured, pattern-based with contextual embeddings
- **📊 Document Reconstruction**: Convert parsed elements back to readable formats (MD, HTML, JSON)
- **🧠 Knowledge Extraction**: Domain entity recognition and relationship discovery
- **⚡ Performance**: FastEmbed integration, bulk processing, optimized for large datasets
- **🖥️ CLI Tools**: Comprehensive command-line interface for pipeline management, monitoring, and automation

## Documentation

- **[Installation Guide](docs/installation.md)** - All installation options and dependencies
- **[CLI Reference](docs/cli.md)** - Complete CLI command reference and examples
- **[Data Sources](docs/sources.md)** - Comprehensive source support (databases, files, APIs, cloud)
- **[Storage Backends](docs/storage.md)** - All storage options and trade-offs
- **[Scaling Guide](docs/scaling.md)** - Horizontal pipeline architecture and performance
- **[Ontology System](docs/ontology.md)** - Knowledge graph and entity extraction
- **[Embeddings](docs/embeddings.md)** - GraphRAG-lite and contextual embeddings
- **[Configuration](docs/configuration.md)** - Complete configuration reference

## Architecture

Go-Doc-Go is built on three core pillars:

### 1. **Massive Input** - Ingest from anywhere
- File systems (local, network, cloud)
- Databases (SQL TEXT/VARCHAR fields, NoSQL)
- APIs (REST, GraphQL, proprietary)  
- Cloud storage (S3, Google Drive, SharePoint)
- Streaming sources (message queues, webhooks)

### 2. **Flexible Storage** - Store however works best
- **SQLite** - Development, small datasets
- **PostgreSQL** - Production, ACID compliance, temporal queries
- **Elasticsearch** - Full-text search optimization
- **MongoDB** - Document flexibility
- **Neo4j** - Graph relationships as first-class citizens

### 3. **Smart Output** - Knowledge, not just data
- Contextual vector embeddings using document structure
- Automated entity extraction and relationship discovery
- Full document reconstruction and format conversion
- Advanced structured search with semantic understanding

## Contributing

Contributions welcome! See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup and guidelines.

## License

MIT License - see [LICENSE](LICENSE) for details.