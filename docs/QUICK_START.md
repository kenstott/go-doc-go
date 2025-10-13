# UDML Quick Start Guide

## Overview

This guide covers the essential documentation for the UDML (Universal Document Model Language) system.

## Core Documentation

### 1. UDML Specification
See [UDML_SPECIFICATION.md](./UDML_SPECIFICATION.md) for complete UDML format specification.

### 2. JSONPath Extensions

UDML supports advanced JSONPath queries with regex matching:

**Basic Query:**
```jsonpath
$.elements[?(@.element_type=='section')]
```

**Regex Matching:**
```jsonpath
$.elements[?(@.content =~ /Customer.*/)]
```

**Nested Properties:**
```jsonpath
$.elements[?(@.properties.page_number > 5)]
```

**Combined Filters:**
```jsonpath
$.elements[?(@.element_type=='section' && @.properties.page_number==1)]
```

### 3. Ontology YAML Schema

Ontologies define how to extract entities and relationships from UDML:

```yaml
version: "1.0.0"
description: "Sample ontology"
domains:
  - name: "sales"
    description: "Sales domain"
    entities:
      - name: "Customer"
        description: "Customer entity"
        queries:
          - path: "$.elements[?(@.content =~ /Customer.*/)]"
            id_columns: ["content"]
            confidence: 0.9
    relationships:
      - name: "PURCHASED"
        description: "Customer purchased product"
        source_query:
          path: "$.elements[?(@.content =~ /Customer.*/)]"
          id_columns: ["content"]
        target_query:
          path: "$.elements[?(@.content =~ /Product.*/)]"
          id_columns: ["content"]
```

See `go/internal/udml/ontology/` for complete implementation.

### 4. Export Formats

UDML graphs can be exported to multiple formats:

- **RDF/Turtle**: W3C Semantic Web standard
- **JSON-LD**: W3C Linked Data format
- **GraphML**: Universal graph interchange (Gephi, yEd compatible)
- **Cypher**: Neo4j native query language
- **RDF Triples**: SPARQL-compatible CSV format

See `go/internal/export/` for exporters.

### 5. Integration Testing

Integration tests cover:

- End-to-end pipeline: Document → UDML → Ontology → Graph → Export
- Multi-format export validation
- Ontology extraction with JSONPath
- Version migration workflows
- Performance benchmarks

See:
- `go/internal/udml/integration_test.go`
- `go/internal/ontology/extraction_test.go`
- `go/internal/export/export_integration_test.go`

## Example Workflows

### Complete Pipeline Example

```go
// 1. Parse document (creates UDML)
doc := parseDocument("report.pdf")

// 2. Query UDML
backend := query.NewMemoryBackend()
backend.StoreDocument(ctx, doc)
results, _ := backend.ExecuteQuery(ctx, query)

// 3. Extract ontology
ont := loadOntology("ontology.yaml")
extractor := ontology.NewExtractor(ont)
extracted, _ := extractor.Extract(ctx, doc)

// 4. Build knowledge graph
builder := graph.NewBuilder()
kg, _ := builder.FromExtraction(extracted)

// 5. Export to format
exporter := export.NewRDFExporter()
exporter.Export(kg, outputFile)
```

## Example Ontologies

Example ontologies for common domains:

- **Automotive**: Vehicle specifications, parts, recalls
- **Financial**: Transactions, accounts, statements
- **Legal**: Contracts, clauses, parties

See `examples/ontologies/` directory.

## Performance

**Query Performance:**
- 10K element document: < 100ms query time
- 100K element document: < 1s query time
- Supports columnar storage (DuckDB/Parquet) for 100M+ elements

**Export Performance:**
- 1K nodes: < 50ms per format
- 10K nodes: < 500ms per format
- 100K nodes: < 5s per format

## Version Tracking

UDML supports system versioning to track schema evolution:

```go
vm := versioning.NewVersionManager(backend)

// Create version
v1 := &versioning.VersionMetadata{
    Version:         "1.0.0",
    ParserVersion:   "2.1.0",
    EmbeddingModel:  "text-embedding-ada-002",
    OntologyVersion: "1.0.0",
}
vm.CreateVersion(ctx, v1)

// Compare versions
diff, _ := vm.CompareVersions(ctx, "1.0.0", "2.0.0")

// Time-travel query
version, _ := vm.GetVersionAtTime(ctx, timestamp)
```

See `go/internal/udml/versioning/` for complete API.

## Resources

- [UDML Migration Plan](../UDML_MIGRATION_PLAN.md) - Complete implementation roadmap
- [UDML Specification](./UDML_SPECIFICATION.md) - Format specification
- [Integration Tests](../go/internal/udml/integration_test.go) - Usage examples
- [Export Package](../go/internal/export/) - Multi-format export

## Getting Help

- GitHub Issues: Report bugs and request features
- Documentation: Check docs/ directory for guides
- Examples: See internal/*_test.go files for code examples

---

**Version**: 1.0.0
**Last Updated**: 2024-10-13
**Status**: Production Ready
