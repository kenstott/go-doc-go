# Go-Doc-Go Documentation

Welcome to the complete documentation for Go-Doc-Go, a universal document processing and knowledge extraction system built in Go.

---

## 🚀 Getting Started

New to Go-Doc-Go? Start here:

1. **[Quick Reference](../QUICK_REFERENCE.md)** - Essential commands, flags, and troubleshooting (5 minutes)
2. **[Getting Started Guide](getting-started/README.md)** - Complete walkthrough from installation to first query (15 minutes)
3. **[Configuration Overview](configuration/README.md)** - Understanding the configuration system

**Quick Start**: Build and run in 60 seconds:
```bash
cd go && go build -o ../bin/goworker ./cmd/worker
../bin/goworker --config config.toml --workers 4
```

---

## 📚 Documentation by Topic

### Configuration

Learn how to configure Go-Doc-Go for your use case:

- **[Configuration Overview](configuration/README.md)** - All configuration options explained
- **[Content Sources](configuration/sources.md)** - Files, S3, databases, web scraping
- **[Storage Backends](configuration/storage.md)** - SQLite, PostgreSQL, Neo4j, and hybrid setups

### Features

Understand Go-Doc-Go's core capabilities:

- **[UDML Specification](features/udml/specification.md)** - Universal Document Markup Language
- **[UDML Schemas](features/udml/schemas.md)** - JSON Schema validation for UDML
- **[UDML Ontology System](features/udml/ontology-system.md)** - Knowledge graph integration
- **[Ontology System](features/ontology/README.md)** - Automated entity extraction
  - [Quick Start](features/ontology/quick-start.md)
  - [Workflows](features/ontology/workflows.md)
  - [Examples](features/ontology/examples.md)
  - [Domain Quick Start](features/ontology/domain-quickstart.md)
- **[Embeddings Guide](features/embeddings/README.md)** - Contextual semantic search

### Operations

Deploy and manage Go-Doc-Go in production:

- **[Scaling Guide](operations/scaling.md)** - Distributed workers and performance tuning
- **[Monitoring](operations/monitoring.md)** - Health checks, metrics, and alerting
- **[Troubleshooting](operations/troubleshooting.md)** - Common issues and solutions

### Reference

Technical reference documentation:

- **[CLI Reference](reference/cli.md)** - Complete command-line interface documentation
- **[Architecture](architecture/worker-design.md)** - System design and internals
- **[UDML-Ontology Complete](architecture/udml-ontology-complete.md)** - Comprehensive architecture

---

## 🎯 Common Use Cases

### Process Local Documents
```toml
[[content_sources]]
name = "local_docs"
type = "file"
base_path = "./documents"
file_pattern = "**/*.{pdf,docx,xlsx}"
```
→ See [Content Sources](configuration/sources.md)

### Export to Neo4j Knowledge Graph
```toml
[processing.neo4j_export]
enabled = true

[processing.neo4j_export.connection]
uri = "bolt://localhost:7687"
username = "neo4j"
password = "password"
```
→ See [Configuration Overview](configuration/README.md)

### Enable Semantic Search
```toml
[embedding]
enabled = true
provider = "onnx"
model_path = "./models/all-MiniLM-L6-v2"
contextual = true
```
→ See [Embeddings Guide](features/embeddings/README.md)

### Scale to Multiple Workers
```bash
# Server 1
./bin/goworker --config config.toml --worker-id "worker-01" --workers 8

# Server 2
./bin/goworker --config config.toml --worker-id "worker-02" --workers 8
```
→ See [Scaling Guide](operations/scaling.md)

### Extract Domain Entities
```toml
[ontology]
enabled = true
schema_path = "./ontologies/financial.yaml"
```
→ See [Ontology System](features/ontology/README.md)

---

## 🔍 Quick Links

### Essential Documentation
- [Quick Reference](../QUICK_REFERENCE.md) - Commands and troubleshooting
- [Getting Started](getting-started/README.md) - Installation and first steps
- [Configuration](configuration/README.md) - Config file reference
- [CLI Reference](reference/cli.md) - Command-line flags

### Common Tasks
- [Troubleshooting](operations/troubleshooting.md) - Fix common issues
- [Performance Tuning](operations/scaling.md) - Optimize throughput
- [Monitoring](operations/monitoring.md) - Track worker health

### Advanced Topics
- [UDML Specification](features/udml/specification.md) - Document model
- [Ontology Extraction](features/ontology/README.md) - Knowledge graphs
- [System Architecture](architecture/worker-design.md) - Design details

---

## 📖 Documentation Index

### By Experience Level

**Beginner** (New to Go-Doc-Go)
1. [Quick Reference](../QUICK_REFERENCE.md)
2. [Getting Started Guide](getting-started/README.md)
3. [Configuration Overview](configuration/README.md)

**Intermediate** (Building production systems)
1. [Scaling Guide](operations/scaling.md)
2. [Content Sources](configuration/sources.md)
3. [Storage Backends](configuration/storage.md)
4. [Monitoring](operations/monitoring.md)

**Advanced** (Customizing and extending)
1. [UDML Specification](features/udml/specification.md)
2. [Ontology System](features/ontology/README.md)
3. [System Architecture](architecture/worker-design.md)
4. [Embeddings Guide](features/embeddings/README.md)

### By Document Type

**Guides** (Step-by-step tutorials)
- [Getting Started](getting-started/README.md)
- [Ontology Quick Start](features/ontology/quick-start.md)
- [Domain Quick Start](features/ontology/domain-quickstart.md)

**Reference** (Technical specifications)
- [CLI Reference](reference/cli.md)
- [UDML Specification](features/udml/specification.md)
- [UDML Schemas](features/udml/schemas.md)

**Operations** (Running in production)
- [Scaling Guide](operations/scaling.md)
- [Monitoring](operations/monitoring.md)
- [Troubleshooting](operations/troubleshooting.md)

**Configuration** (Setup and customization)
- [Configuration Overview](configuration/README.md)
- [Content Sources](configuration/sources.md)
- [Storage Backends](configuration/storage.md)

---

## 💡 Need Help?

### Quick Troubleshooting
1. **Worker won't start?** → [Troubleshooting Guide](operations/troubleshooting.md#worker-issues)
2. **No documents processing?** → [Troubleshooting Guide](operations/troubleshooting.md#document-processing-issues)
3. **Performance slow?** → [Scaling Guide](operations/scaling.md)
4. **Config errors?** → [Configuration Overview](configuration/README.md)

### Resources
- **Quick Reference**: [QUICK_REFERENCE.md](../QUICK_REFERENCE.md)
- **Configuration Help**: [configuration/README.md](configuration/README.md)
- **Common Errors**: [Troubleshooting](operations/troubleshooting.md#common-error-messages)
- **GitHub Issues**: Report bugs and request features

---

**Last Updated**: 2025-01-16
**Version**: 1.0