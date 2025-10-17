# Go-Doc-Go Glossary

**Version 1.0** - Standard terminology for Go-Doc-Go documentation.

This glossary defines the standard terms used throughout Go-Doc-Go documentation. Use these terms consistently to avoid confusion.

---

## Core Concepts

### Worker

A single instance of the `goworker` binary process that discovers, claims, and processes documents. Each worker can run multiple goroutine workers concurrently.

**Usage**: "Start a worker", "The worker processes documents", "Run multiple workers"

**Not**: "worker node", "worker instance", "worker process" (use "worker" alone)

### Goroutine Worker

A concurrent goroutine within a single worker process that processes documents in parallel. Controlled by the `--workers` flag.

**Usage**: "Configure 8 goroutine workers", "Each goroutine worker processes documents concurrently"

**Not**: "thread", "worker thread", "concurrent worker"

### Job Control

The system that manages document discovery, claiming, and coordination between workers. Implemented using SQLite or PostgreSQL.

**Usage**: "Job control database", "Job control backend", "Job control coordinates workers"

**Not**: "work queue", "task queue", "job queue"

**Related**: Job control backend, job control system

### Content Source

A configured location from which documents are discovered and ingested. Examples: file system, S3 bucket, web URL.

**Usage**: "Configure content sources", "Add a content source", "File content source"

**Not**: "data source", "document source", "input source"

### Analytics Output

The destination where processed document data is written. Examples: Parquet files, Neo4j database.

**Usage**: "Configure analytics outputs", "Parquet analytics output"

**Not**: "output", "export", "destination"

---

## Configuration

### TOML

The **only** configuration file format supported by Go-Doc-Go. All configuration examples must use TOML syntax.

**Usage**: "Create a TOML configuration file", "Edit config.toml"

**Not**: "YAML config", "JSON config", "config file" (always specify TOML)

**File Extension**: `.toml`

**Format**:
```toml
[section]
key = "value"
```

### Configuration File

A TOML file containing Go-Doc-Go settings. The default name is `config.toml`.

**Usage**: "Edit the configuration file", "Create config.toml"

**Not**: "config", "settings file"

---

## Document Processing

### Document

A single file or item to be processed. Documents are discovered from content sources.

**Usage**: "Process documents", "A PDF document", "Document processing"

**Not**: "doc", "file" (use "document" consistently)

### UDML

Universal Document Markup Language - the standardized format for representing any document as elements and relationships.

**Usage**: "Parse to UDML", "UDML elements", "UDML specification"

**Full Name**: Universal Document Markup Language (use on first mention)

### Element

A discrete unit of content within a document. Every element has a type (from the 5-category taxonomy).

**Usage**: "UDML element", "Paragraph element", "Element types"

**Types**: Container, Content, Structure, Component, Metadata

### Element Type

The category classification of an element. Must be one of the standardized types defined in the UDML specification.

**Usage**: "Element type: paragraph", "Valid element types"

**Examples**: `paragraph`, `table`, `header`, `page`, `slide`

### Relationship

A connection between two elements. Relationships have types that describe the nature of the connection.

**Usage**: "Element relationships", "Contains relationship", "Relationship types"

**Types**: `contains`, `references`, `next`, `previous`, `links_to`

### Relationship Type

The category of connection between elements.

**Usage**: "Relationship type: contains", "Standard relationship types"

---

## Architecture

### Binary

The compiled Go executable. The main binary is `goworker`.

**Usage**: "Build the binary", "Run the goworker binary"

**Location**: `bin/goworker`

**Not**: "executable", "program", "application"

### Distributed Processing

Multiple workers coordinating via PostgreSQL job control to process documents in parallel.

**Usage**: "Distributed processing architecture", "Scale with distributed workers"

**Not**: "parallel processing", "multi-worker", "cluster processing"

### Horizontal Scaling

Adding more worker processes to increase throughput.

**Usage**: "Scale horizontally by adding workers", "Horizontal scaling patterns"

**Not**: "scaling out", "distributed scaling"

---

## Storage

### Job Control Backend

The database used for job control (SQLite or PostgreSQL).

**Usage**: "PostgreSQL job control backend", "SQLite job control backend"

**Not**: "queue database", "work database"

### Analytics Backend

The storage system for processed document data (Parquet, Neo4j, etc.).

**Usage**: "Parquet analytics backend", "Neo4j analytics backend"

**Not**: "output database", "results storage"

### Parquet

Columnar file format used for analytics output. Efficient for analytical queries.

**Usage**: "Parquet analytics output", "Query Parquet files"

**Format**: Apache Parquet

### Neo4j

Graph database optionally used as an analytics backend for relationship queries.

**Usage**: "Export to Neo4j", "Neo4j graph analytics"

**Not**: "graph database" (always specify Neo4j)

---

## Features

### Ontology

A domain-specific schema defining entities and relationships to extract from documents.

**Usage**: "Ontology extraction", "Define an ontology schema", "Enable ontology features"

**File Format**: YAML (ontology schemas use YAML, not TOML)

### Ontology Schema

A YAML file defining the entities, relationships, and extraction rules for a specific domain.

**Usage**: "Create an ontology schema", "Load ontology schema"

**File Extension**: `.yaml`

### Entity

A domain-specific object extracted from documents using ontology rules.

**Usage**: "Extract entities", "Company entity", "Entity types"

**Not**: "object", "item", "concept"

### Embedding

A vector representation of document content used for semantic search.

**Usage**: "Generate embeddings", "Contextual embeddings", "Embedding model"

**Not**: "vector", "embedding vector" (use "embedding" alone)

### Contextual Embedding

An embedding that includes context from surrounding elements (predecessors and successors).

**Usage**: "Enable contextual embeddings", "Contextual embedding generation"

**Related**: GraphRAG-lite embeddings

### GraphRAG-lite

Go-Doc-Go's approach to contextual embeddings using document structure (graph-lets).

**Usage**: "GraphRAG-lite embeddings", "GraphRAG-lite approach"

---

## CLI & Operations

### Flag

A command-line argument for the `goworker` binary.

**Usage**: "The --workers flag", "CLI flags", "Command-line flags"

**Format**: `--flag-name`

### Environment Variable

A system environment variable that configures Go-Doc-Go behavior.

**Usage**: "Set the NUM_WORKERS environment variable", "Environment variables"

**Format**: `UPPERCASE_WITH_UNDERSCORES`

### Worker ID

A unique identifier for a worker instance. Auto-generated or specified with `--worker-id`.

**Usage**: "Set a custom worker ID", "Worker ID: worker-01"

**Format**: `worker_<hostname>_<pid>` (auto-generated) or custom string

---

## Standard Terms to Use

### Prefer These Terms

| Use This | Not This |
|----------|----------|
| Worker | Worker node, worker instance, worker process |
| Job control | Work queue, task queue, job queue |
| Document | Doc, file |
| Content source | Data source, document source, input source |
| Analytics output | Output, export, destination |
| TOML | YAML, JSON, config format |
| Binary | Executable, program, application |
| Goroutine worker | Thread, worker thread, concurrent worker |
| Element type | Element kind, element category |
| Relationship type | Relationship kind, link type |
| Distributed processing | Parallel processing, cluster processing |
| Horizontal scaling | Scaling out |

---

## Acronyms and Abbreviations

### UDML
**Full Name**: Universal Document Markup Language
**First Use**: Write out full name, then use acronym
**Example**: "Universal Document Markup Language (UDML) provides..."

### TOML
**Full Name**: Tom's Obvious Minimal Language
**Usage**: Can use acronym without expansion (well-known format)

### CLI
**Full Name**: Command-Line Interface
**Usage**: Can use acronym without expansion (well-known term)

### API
**Full Name**: Application Programming Interface
**Usage**: Can use acronym without expansion (well-known term)

---

## Capitalization Rules

### Product Names
- **Go-Doc-Go**: Hyphenated, capital G and D (product name)
- **UDML**: All caps (acronym)
- **PostgreSQL**: One word, capital P and SQL
- **SQLite**: One word, capital SQL
- **Neo4j**: One word, lowercase j
- **DuckDB**: One word, capital D and DB

### File Names
- `config.toml`: Lowercase (file name)
- `goworker`: Lowercase (binary name)
- `README.md`: All caps README (convention)

### Technical Terms
- goroutine: Lowercase (Go language term)
- ONNX: All caps (acronym)
- Parquet: Capital P (Apache Parquet)

---

## Usage Examples

### Correct Usage

✅ "Configure a worker with 8 goroutine workers using the `--workers` flag"

✅ "The job control backend coordinates document claiming between workers"

✅ "Create a TOML configuration file at `config.toml`"

✅ "Process documents from multiple content sources"

✅ "Export analytics output to Parquet files"

✅ "Define entities in the ontology schema YAML file"

### Incorrect Usage

❌ "Configure a worker node with 8 threads using the config"

❌ "The work queue database manages the task queue"

❌ "Create a YAML config file at `config.toml`"

❌ "Process docs from multiple data sources"

❌ "Export results to output files"

❌ "Define objects in the ontology config file"

---

## Related Documentation

- **Up**: [Documentation Home](README.md)

### Quick Links

- [Documentation Home](README.md)
- [Quick Reference](../QUICK_REFERENCE.md)
- [Configuration Overview](configuration/README.md)
- [UDML Specification](features/udml/specification.md)

---

**Last Updated**: 2025-01-16
**Version**: 1.0