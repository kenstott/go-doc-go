# Go-Doc-Go Ontology Extraction Demo

This directory contains files for the complete ontology extraction demo walkthrough.

## Quick Start

```bash
# Follow the step-by-step instructions in:
cat ../ONTOLOGY_DEMO_WALKTHROUGH.md
```

## Directory Structure

```
demo/
├── README.md                    # This file
├── documents/                   # Source documents (copied from tests/assets)
│   ├── *.md                    # Markdown documentation
│   ├── *.docx                  # Word documents
│   ├── *.pdf                   # PDF presentations
│   └── wikipedia_graphql.md    # Fetched Wikipedia page
├── ontologies/                  # Ontology schemas
│   └── business_demo.yaml      # Business/technology entity extraction rules
├── output/                      # Processing output (created during demo)
│   ├── jobs.db                 # SQLite job queue
│   └── analytics/              # Hive-partitioned Parquet files
│       ├── documents/          # Document metadata
│       ├── elements/           # UDML elements
│       ├── relationships/      # Document relationships
│       ├── ontology_entities/  # Extracted entities
│       ├── ontology_mentions/  # Entity mentions with positions
│       └── ontology_relationships/ # Entity relationships
├── config_initial.toml          # Initial ingestion config
├── config_ontology.toml         # Ontology extraction config
└── config_with_neo4j.toml       # Neo4j export config
```

## What You'll Learn

1. How to ingest documents into UDML format
2. How to create ontology schemas with extraction rules
3. How to extract entities and relationships from documents
4. How to query results with DuckDB
5. How to visualize knowledge graphs in Neo4j
6. How to iterate and refine extraction rules

## Expected Results

- **Documents**: 6-8 documents (DOCX, PDF, MD)
- **UDML Elements**: 300-500 elements (paragraphs, headings, tables)
- **Entities**: 50-100 entities (organizations, technologies, people, locations)
- **Relationships**: 30-60 relationships (dependencies, locations, usage)
- **Processing Time**: 30-60 seconds

## Demo Scenarios

### Scenario 1: Technology Stack Discovery
Discover technologies mentioned in technical documentation and their dependencies.

### Scenario 2: Organization and Location Mapping
Extract companies and their office locations from business documents.

### Scenario 3: Knowledge Graph Building
Build a complete knowledge graph and export to Neo4j for visualization.

## Cleanup

```bash
# Remove generated files
rm -rf demo/output
rm -f demo/config_*.toml
rm -rf demo/documents
rm -rf demo/ontologies

# Or keep for future reference
```

## Next Steps

After completing the demo:
- Modify the ontology schema for your domain
- Add more documents to process
- Enable embeddings for semantic search
- Deploy distributed workers for scale
- Export to your preferred knowledge graph database

## Resources

- [ONTOLOGY_DEMO_WALKTHROUGH.md](../ONTOLOGY_DEMO_WALKTHROUGH.md) - Complete step-by-step guide
- [go/README.md](../go/README.md) - Go worker implementation guide
- [docs/ontology.md](../docs/ontology.md) - Ontology system documentation
- [UDML_SPECIFICATION.md](../docs/UDML_SPECIFICATION.md) - UDML spec

Happy knowledge extraction! 🎉
