# UDML JSON Schemas

This directory contains JSON Schema definitions for UDML (Universal Document Model Language) validation.

## Schema Files

UDML provides **three JSON schemas** for different format variants:

### 1. Persistence Format Schema (`udml-1.0.schema.json`)

**Purpose**: Validates the flat array format used for storage and SQL queries

**Used by**:
- Parsers (output validation)
- Database ingestion pipelines
- Parquet file generation
- Storage systems (PostgreSQL, DuckDB)

**Key characteristics**:
- Elements in flat array with `parent_id` field
- `position` field for global ordering
- No separate relationships array (hierarchy via `parent_id` only)
- Optimized for columnar storage

**When to use**: Validate parser output before storage, validate data being loaded into databases

---

### 2. Hierarchical Format Schema (`udml-1.0-hierarchical.schema.json`)

**Purpose**: Validates the nested format used for processing and JSONPath queries

**Used by**:
- Document reconstruction (`DocumentBuilder`)
- Ontology extraction (JSONPath evaluation)
- LLM prompt formatting
- Tree visualization tools

**Key characteristics**:
- Elements nested with `children` arrays
- No `parent_id` field (implicit in nesting)
- `element_order` instead of `position`
- No separate `relationships` array

**When to use**: Validate hierarchical documents built from flat format, validate nested exports

---

### 3. Graph Format Schema (`udml-1.0-graph.schema.json`)

**Purpose**: Validates the graph format (UDML-G) used for graph database exports and network analysis

**Used by**:
- Graph database exports (Neo4j, ArangoDB, TigerGraph)
- RDF/OWL triple generation
- Network analysis tools (NetworkX, igraph)
- Graph visualization (Cytoscape, Gephi, Graphviz)
- Knowledge graph construction

**Key characteristics**:
- `nodes` array contains all vertices (simplified field names: `id`, `type`)
- `edges` array contains structural relationships (fields: `source`, `target`, `type`)
- Edges generated from `parent_id` during export (not stored in persistence format)
- Only `contains` edge type (structural hierarchy)
- Optimized for property graph databases and RDF exports
- Direct mapping to graph query languages (Cypher, Gremlin, SPARQL)

**When to use**: Validate graph exports before importing to graph databases, validate RDF generation

---

### Example Files

- **`example-valid.json`** - Example valid UDML document (persistence format) that passes schema validation

## Usage

### Validate with Python

```bash
pip install jsonschema
```

```python
import json
import jsonschema

# Load schema
with open('schemas/udml-1.0.schema.json') as f:
    schema = json.load(f)

# Load document
with open('document.json') as f:
    doc = json.load(f)

# Validate
try:
    jsonschema.validate(doc, schema)
    print("✓ Valid UDML document")
except jsonschema.ValidationError as e:
    print(f"✗ Validation error: {e.message}")
    print(f"  At path: {' > '.join(str(p) for p in e.path)}")
```

### Validate with Node.js (AJV)

```bash
npm install ajv
```

```javascript
const Ajv = require('ajv');
const fs = require('fs');

const ajv = new Ajv();
const schema = JSON.parse(fs.readFileSync('schemas/udml-1.0.schema.json'));
const doc = JSON.parse(fs.readFileSync('document.json'));

const validate = ajv.compile(schema);
const valid = validate(doc);

if (valid) {
    console.log('✓ Valid UDML document');
} else {
    console.log('✗ Validation errors:');
    validate.errors.forEach(err => {
        console.log(`  ${err.instancePath}: ${err.message}`);
    });
}
```

### Validate with Go

```bash
go get github.com/xeipuuv/gojsonschema
```

```go
package main

import (
    "fmt"
    "github.com/xeipuuv/gojsonschema"
)

func main() {
    schemaLoader := gojsonschema.NewReferenceLoader("file://schemas/udml-1.0.schema.json")
    docLoader := gojsonschema.NewReferenceLoader("file://document.json")

    result, err := gojsonschema.Validate(schemaLoader, docLoader)
    if err != nil {
        panic(err)
    }

    if result.Valid() {
        fmt.Println("✓ Valid UDML document")
    } else {
        fmt.Println("✗ Validation errors:")
        for _, err := range result.Errors() {
            fmt.Printf("  - %s\n", err)
        }
    }
}
```

### Validate with CLI Tools

**Using `ajv-cli`:**
```bash
npm install -g ajv-cli
ajv validate -s schemas/udml-1.0.schema.json -d document.json
```

**Using `check-jsonschema`:**
```bash
pip install check-jsonschema
check-jsonschema --schemafile schemas/udml-1.0.schema.json document.json
```

## Test the Example

Validate the example document:

```bash
# Python
python -c "import json, jsonschema; jsonschema.validate(json.load(open('schemas/example-valid.json')), json.load(open('schemas/udml-1.0.schema.json')))" && echo "✓ Valid"

# Node.js / ajv-cli
ajv validate -s schemas/udml-1.0.schema.json -d schemas/example-valid.json

# check-jsonschema
check-jsonschema --schemafile schemas/udml-1.0.schema.json schemas/example-valid.json
```

## Schema Features

The UDML v1.0 JSON Schema validates:

### Required Fields
- **Document level**: `document_id`, `metadata`, `elements`
- **Metadata level**: `source_type`
- **Element level**: `element_id`, `element_type`, `content`

### Data Types
- **Strings**: document_id, element_id, content, etc.
- **Integers**: page_number (≥1), section_level (1-6), row/column indices (≥0)
- **Numbers**: confidence (0.0-1.0)
- **Arrays**: elements, embeddings vector
- **Objects**: metadata, properties, embeddings

### Enumerations

**Element types** (54 types organized by category):
- **Document structure**: root, body, section, document
- **Text**: paragraph, text, heading, header
- **Lists**: list, list_item
- **Links**: link (container element with link_target property)
- **Tables**: table, data_table, data_tables, table_header, table_header_row, table_header_cell, table_row, table_cell
- **Media**: image, figure, chart
- **Code**: code_block
- **HTML/Web**: div, span, article, blockquote
- **JSON**: json_field, json_object, json_array, json_item
- **XML**: xml_element
- **Office - Word**: headers, footers, comments
- **Office - Excel**: sheet, workbook, comment
- **Office - PowerPoint**: slide, slide_notes, shape_group
- **PDF**: page
- **Markdown**: front_matter
- **Text parser**: line, range, substring
- **Metadata**: formula, footnote, quote, caption, metadata, other

**Source types** (11 types):
- pdf, docx, xlsx, pptx, html, markdown, text, json, xml, csv, other

**Temporal types** (4 types):
- date, time, duration, timestamp

### Constraints
- **Confidence scores**: Must be in range [0.0, 1.0]
- **Page numbers**: Must be ≥ 1 (1-indexed)
- **Section levels**: Must be 1-6
- **Row/column indices**: Must be ≥ 0 (0-indexed)
- **Element IDs**: Must be unique (not enforced by schema, validated by application)
- **Language codes**: Must be 2-letter ISO 639-1 codes

### Extensibility
- **Properties bag**: Accepts any additional properties (e.g., link_target, link_type for link elements)
- **Custom metadata**: `metadata.custom` accepts any object

## Common Validation Errors

### Missing Required Field
```json
{
  "error": "Missing required property: document_id"
}
```
**Fix**: Add `document_id` at document root level.

### Invalid Element Type
```json
{
  "error": "Value 'invalid_type' does not match any of the enum values"
}
```
**Fix**: Use valid element type from enum (section, paragraph, heading, etc.).

### Confidence Out of Range
```json
{
  "error": "Value 1.5 is greater than maximum 1.0"
}
```
**Fix**: Ensure confidence is in range [0.0, 1.0].

### Invalid Parent Reference
```json
{
  "error": "Element parent_id 'elem-999' does not reference existing element"
}
```
**Note**: This validation requires application logic, not enforced by JSON Schema. Hierarchy is encoded via `parent_id` field only.

## Integration with IDEs

### VS Code

Add to `.vscode/settings.json`:
```json
{
  "json.schemas": [
    {
      "fileMatch": ["**/udml/**/*.json", "**/documents/**/*.json"],
      "url": "./schemas/udml-1.0.schema.json"
    }
  ]
}
```

### IntelliJ IDEA / PyCharm

1. Go to **Preferences → Languages & Frameworks → Schemas and DTDs → JSON Schema Mappings**
2. Click **+** to add new mapping
3. **Schema file**: Select `schemas/udml-1.0.schema.json`
4. **Schema version**: JSON Schema version 7
5. **File path pattern**: `**/*udml*.json` or `**/documents/*.json`

### Sublime Text

Install **LSP-json** package and add to settings:
```json
{
  "json_schemas": [
    {
      "file_patterns": ["**/udml/**/*.json"],
      "schema_path": "schemas/udml-1.0.schema.json"
    }
  ]
}
```

## CI/CD Integration

### GitHub Actions

```yaml
name: Validate UDML Documents

on: [push, pull_request]

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Install validator
        run: pip install check-jsonschema

      - name: Validate UDML documents
        run: |
          find . -name "*.udml.json" -exec \
            check-jsonschema --schemafile schemas/udml-1.0.schema.json {} \;
```

### Pre-commit Hook

Add to `.pre-commit-config.yaml`:
```yaml
repos:
  - repo: https://github.com/python-jsonschema/check-jsonschema
    rev: 0.27.0
    hooks:
      - id: check-jsonschema
        name: Validate UDML documents
        files: '\.udml\.json$'
        args: ['--schemafile', 'schemas/udml-1.0.schema.json']
```

## Schema Evolution

### Versioning
- **Current version**: 1.0.0
- **Schema URL**: `https://github.com/kennethstott/doculyzer-go-conversion/schemas/udml-1.0.schema.json`
- **Future versions**: Will be published as `udml-1.1.schema.json`, `udml-2.0.schema.json`, etc.

### Backward Compatibility
- **Minor versions** (1.0 → 1.1): Backward compatible (add optional fields only)
- **Major versions** (1.x → 2.0): May break compatibility (use migration tools)

### Migration
See `docs/UDML_SPECIFICATION.md` for migration guides between versions.

## Resources

- **UDML Specification**: `docs/UDML_SPECIFICATION.md`
- **Quick Start Guide**: `docs/QUICK_START.md`
- **Example Ontologies**: `examples/ontologies/`
- **Integration Tests**: `go/internal/udml/integration_test.go`

## Support

- **JSON Schema Docs**: https://json-schema.org/
- **UDML Issues**: https://github.com/kennethstott/doculyzer-go-conversion/issues
- **Validation Libraries**:
  - Python: https://python-jsonschema.readthedocs.io/
  - Node.js: https://ajv.js.org/
  - Go: https://github.com/xeipuuv/gojsonschema

---

**Version**: 1.0.0
**Last Updated**: 2024-10-13
**Status**: Stable
