# Go-Doc-Go JSON Schemas

This directory contains JSON Schema definitions for the various data formats used in Go-Doc-Go.

## Available Schemas

### UDML (Universal Document Markup Language) Schemas

UDML represents the parsed structure of documents.

#### `udml-1.0.schema.json`
**Purpose:** Main UDML persistence/storage format (UDML-S)

**Description:** Flat array structure optimized for databases, Parquet files, and SQL queries. Includes Universal Code Markup Language (UCML) subset for source code parsing.

**Use Cases:**
- Validating parsed document output
- Database schema generation
- Parquet file structure validation
- Storage system integration

**Key Features:**
- Flat array of elements with parent references
- Optimized for analytical queries
- Supports all document types (PDF, DOCX, HTML, code, etc.)

#### `udml-1.0-hierarchical.schema.json`
**Purpose:** Hierarchical UDML format (UDML-H)

**Description:** Nested tree structure with parent-child relationships represented as nested objects.

**Use Cases:**
- Rendering document trees
- UI display
- Document navigation
- Tree-based transformations

**Key Features:**
- Nested parent-child structure
- Easier to traverse for display purposes
- Human-readable format

#### `udml-1.0-graph.schema.json`
**Purpose:** Graph-oriented UDML format (UDML-G)

**Description:** Graph database format with explicit node and edge representations.

**Use Cases:**
- Neo4j import
- Graph analytics
- Relationship-focused queries
- Knowledge graph construction

**Key Features:**
- Explicit nodes and edges
- Optimized for graph databases
- Relationship-first design

### Ontology Schemas

Ontology schemas define extraction rules for entities and relationships.

#### `ontology-compiler-1.0.schema.json`
**Purpose:** Ontology Compiler configuration format

**Description:** Configuration schema for defining entity and relationship extraction rules. This is the format produced by the ontology interview process and consumed by the ontology extractor.

**Use Cases:**
- Validating ontology interview output
- Schema-driven entity extraction
- IDE autocomplete for ontology configs
- Configuration validation before extraction

**Key Features:**
- Multi-domain support
- Entity type hierarchies with inheritance
- Extraction rules with filters (regex, proximity, semantic, LLM)
- Relationship extraction patterns
- Attribute extraction for schema parsimony
- Cross-domain entity merging configuration

**Main Components:**

1. **Domains**: Domain registry for data mesh architecture
   - Domain ownership
   - Inter-domain dependencies
   - Domain-scoped entities

2. **Element Entity Mappings**: Rules for extracting entities from UDML elements
   - Entity type definitions
   - Parent-child inheritance (e.g., physician extends person)
   - W-category classification (who, what, where, when, why, how)
   - Confidence scores
   - Extraction rules with multiple filter types

3. **Extraction Rules**: Multi-stage filtering pipeline
   - **Extraction methods**: phrase_list (keywords) or instance_name (regex)
   - **Pre-filters** (fail-fast, AND logic):
     - Pattern: Regex pre-filter
     - Proximity: Co-occurrence with signal terms
     - Dictionary: Linguistic/POS validation
     - Semantic: Embedding similarity
     - LLM Validation: Complex validation via LLM
   - **Attribute extraction**: Extract metadata from matched entities

4. **Relationship Rules**: Extract relationships between entities
   - Source/target entity type constraints
   - Extraction patterns (text templates, proximity, regex)
   - Inheritance from parent relationships
   - Relationship attributes

5. **Cross-Domain Merging**: Merge duplicate entities across domains
   - Fuzzy matching
   - Semantic similarity
   - Configurable thresholds

**Example Structure:**
```yaml
name: "Medical Knowledge Extraction"
version: "1.0.0"
llm_model: "claude-sonnet-4"

domains:
  - name: medical
    description: "Medical domain"

element_entity_mappings:
  - entity_type: physician
    parent_type: global.person
    domain: medical
    w_category: who
    confidence: 0.85
    extraction_rules:
      - instance_name: '(?P<name>Dr\\.\\s+[A-Z][a-z]+)'
        proximity:
          cooccurrence_terms: ["hospital", "clinic"]
        attributes:
          - name: specialty
            type: regex
            regex_pattern: '(?P<value>cardiology|neurology)'

entity_relationship_rules:
  - name: physician_works_at_hospital
    source_entity_type: physician
    target_entity_type: hospital
    relationship_type: employed_at
    extraction_patterns:
      - type: text_template
        template: "{physician} works at {hospital}"
```

## Schema Validation

### Command Line Validation

Using `ajv-cli`:

```bash
# Install ajv-cli
npm install -g ajv-cli

# Validate UDML output
ajv validate -s schemas/udml-1.0.schema.json -d output/document.json

# Validate ontology config
ajv validate -s schemas/ontology-compiler-1.0.schema.json -d examples/medical_ontology.yaml
```

### Go Validation

```go
import (
    "github.com/xeipuuv/gojsonschema"
)

func validateOntologySchema(configPath, schemaPath string) error {
    schemaLoader := gojsonschema.NewReferenceLoader("file://" + schemaPath)
    documentLoader := gojsonschema.NewReferenceLoader("file://" + configPath)

    result, err := gojsonschema.Validate(schemaLoader, documentLoader)
    if err != nil {
        return err
    }

    if !result.Valid() {
        for _, err := range result.Errors() {
            fmt.Printf("- %s\n", err)
        }
        return fmt.Errorf("validation failed")
    }

    return nil
}
```

### Python Validation

```python
import json
import jsonschema
import yaml

# Load schema
with open('schemas/ontology-compiler-1.0.schema.json') as f:
    schema = json.load(f)

# Load and validate YAML config
with open('examples/medical_ontology.yaml') as f:
    config = yaml.safe_load(f)

try:
    jsonschema.validate(instance=config, schema=schema)
    print("✓ Valid ontology configuration")
except jsonschema.ValidationError as e:
    print(f"✗ Validation error: {e.message}")
```

## IDE Integration

### VSCode

Add to `.vscode/settings.json`:

```json
{
  "yaml.schemas": {
    "./schemas/ontology-compiler-1.0.schema.json": [
      "examples/**/*_ontology.yaml",
      "configs/**/*_extraction.yaml"
    ]
  },
  "json.schemas": [
    {
      "fileMatch": ["**/udml_output/**/*.json"],
      "url": "./schemas/udml-1.0.schema.json"
    }
  ]
}
```

### JetBrains IDEs (IntelliJ, PyCharm, GoLand)

1. Go to Settings → Languages & Frameworks → Schemas and DTDs → JSON Schema Mappings
2. Add new mapping:
   - Schema file: `schemas/ontology-compiler-1.0.schema.json`
   - File pattern: `**/*_ontology.yaml`

## Schema Versioning

Schemas follow semantic versioning:

- **Major version** (1.0.0 → 2.0.0): Breaking changes to structure
- **Minor version** (1.0.0 → 1.1.0): Backward-compatible additions
- **Patch version** (1.0.0 → 1.0.1): Bug fixes, clarifications

## Contributing

When modifying schemas:

1. Update the schema file
2. Increment version appropriately
3. Update this README with changes
4. Add/update examples in `examples/`
5. Test validation with existing configs

## Related Documentation

- [Ontology Interview Process](../docs/ontology/interview.md)
- [UDML Specification](../docs/udml/specification.md)
- [Entity Extraction Guide](../docs/ontology/extraction.md)
- [Catalog Development](../docs/ontology/catalogs.md)