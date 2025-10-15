# UDML Ontology System - Quick Start Guide

## Installation

```bash
# Clone the repository
git clone https://github.com/kennethstott/doculyzer-go-conversion
cd doculyzer-go-conversion/go

# Build the tools
go build -o bin/ontology ./cmd/ontology
go build -o bin/export_catalogs ./cmd/export_catalogs
```

## Using Domain Catalogs

### List Available Domains

```go
package main

import (
    "fmt"
    "github.com/kennethstott/doculyzer-go-conversion/internal/udml/ontology/catalogs"
)

func main() {
    // Catalogs auto-load from examples/ontologies/
    domains := catalogs.ListDomains()
    fmt.Printf("Available domains: %v\n", domains)
    // Output: [education financial insurance logistics manufacturing medical legal retail technical]
}
```

### Get a Specific Catalog

```go
catalog, exists := catalogs.GetCatalog("financial")
if exists {
    fmt.Printf("Domain: %s\n", catalog.Domain)
    fmt.Printf("Entity types: %d\n", len(catalog.EntityTypes))
    fmt.Printf("Relationships: %d\n", len(catalog.Relationships))
}
```

### Load Custom Catalogs

```go
// Load from custom directory
err := catalogs.RegisterFromDirectory("./my-company-catalogs")
if err != nil {
    log.Fatal(err)
}

// Now your custom catalogs are available
catalog, exists := catalogs.GetCatalog("my_custom_domain")
```

## Creating a Custom Domain Catalog

### Step 1: Create YAML File

Create `my-domain.yaml`:

```yaml
domain: automotive
description: Automotive industry documents, service records, specifications

subdomains:
  - manufacturing
  - service
  - sales

terms:
  - name: VIN
    synonyms:
      - vehicle identification number
      - chassis number
    description: Unique vehicle identifier

entity_types:
  - entity_type: vehicle
    description: Motor vehicle
    aliases:
      - car
      - automobile
      - truck
    element_types:
      - paragraph
      - table_cell
    sample_rules:
      - type: keyword_match
        keywords:
          - vehicle
          - car
      - type: regex_pattern
        pattern: '\b[A-Z0-9]{17}\b'  # VIN pattern

common_entity_refs:
  - person
  - date
  - address

relationships:
  - name: vehicle_owned_by_person
    source_type: vehicle
    target_type: person
    relationship_type: related_to
    description: Vehicle owned by person
    sample_patterns:
      - "{vehicle} owned by {person}"
```

### Step 2: Load Your Catalog

```go
err := catalogs.RegisterFromDirectory("./my-catalogs")
if err != nil {
    log.Fatal(err)
}
```

## Generating an Ontology

```go
package main

import (
    "context"
    "log"
    "os"

    "github.com/kennethstott/doculyzer-go-conversion/internal/udml/ontology"
)

func main() {
    config := ontology.BuilderConfig{
        ParquetPath: "./output/udml.parquet",
        SampleSize:  1000,
        LLMProvider: "anthropic",
        LLMModel:    "claude-sonnet-4-5-20250929",
        LLMAPIKey:   os.Getenv("ANTHROPIC_API_KEY"),
        SchemaName: "MyOntology",
    }

    builder, err := ontology.NewOntologyBuilder(config)
    if err != nil {
        log.Fatal(err)
    }
    defer builder.Close()

    result, err := builder.Build(context.Background())
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Detected domains: %v", result.Schema.Domains)
    log.Printf("Entity mappings: %d", len(result.Schema.ElementEntityMappings))
    log.Printf("Relationships: %d", len(result.Schema.EntityRelationshipRules))
}
```

## Command-Line Tools

### Export Catalogs

```bash
go run ./cmd/export_catalogs ./output/catalogs
```

### Test Catalog Loading

```bash
go run ./cmd/test_catalog_load
```

## Resources

- **Full Documentation**: `docs/UDML_ONTOLOGY_SYSTEM.md`
- **Catalog Examples**: `examples/ontologies/*.yaml`
- **Catalog Guide**: `examples/ontologies/README.md`
