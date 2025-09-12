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
- Elements know about their neighbors, parents, and children
- Vector search considers document hierarchy and context
- Better results than flat text embeddings
- Foundation for full GraphRAG implementations

## Mental Model

```
Any Data Source → Universal Graph → Knowledge Graph → Smart Search
     │                    │               │              │
 Documents            Elements &        Domain         GraphRAG-lite
 Databases           Relationships     Entities &      Embeddings
 APIs                                  Rules           
```

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
from go_doc_go import search_with_documents

# Contextual search with document reconstruction
results = search_with_documents(
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

## Core Capabilities

- **📄 Universal Parsing**: PDF, DOCX, PPTX, XLSX, HTML, Markdown, JSON, CSV, XML, plain text
- **🔌 Flexible Sources**: Files, databases, S3, SharePoint, Confluence, APIs, Google Drive
- **🏗️ Scalable Architecture**: Distributed processing, horizontal scaling, work queue coordination
- **🔍 Smart Search**: Semantic, structured, pattern-based with contextual embeddings
- **📊 Document Reconstruction**: Convert parsed elements back to readable formats (MD, HTML, JSON)
- **🧠 Knowledge Extraction**: Domain entity recognition and relationship discovery
- **⚡ Performance**: FastEmbed integration, bulk processing, optimized for large datasets

## Documentation

- **[Installation Guide](docs/installation.md)** - All installation options and dependencies
- **[Data Sources](docs/sources.md)** - Comprehensive source support (databases, files, APIs, cloud)
- **[Storage Backends](docs/storage.md)** - All storage options and trade-offs
- **[Scaling Guide](docs/scaling.md)** - Horizontal pipeline architecture and performance
- **[Ontology System](docs/ontology.md)** - Knowledge graph and entity extraction
- **[Embeddings](docs/embeddings.md)** - GraphRAG-lite and contextual embeddings
- **[Configuration](docs/configuration.md)** - Complete configuration reference
- **[API Reference](docs/api.md)** - Full API documentation

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