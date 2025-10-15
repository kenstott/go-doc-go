# Custom Domain Catalog Configuration

This directory contains example YAML configurations for defining custom domain catalogs. Domain catalogs can be defined in YAML or JSON files and loaded at runtime, allowing you to extend the ontology system without modifying code.

## Quick Start

### Option 1: Load from Directory

```go
import "github.com/kennethstott/doculyzer-go-conversion/internal/udml/ontology/catalogs"

// Load all YAML files from directory and register them
err := catalogs.RegisterFromDirectory("./examples/ontologies")
if err != nil {
    log.Fatal(err)
}

// Now use catalogs normally
catalog, exists := catalogs.GetCatalog("education")
```

### Option 2: Load Individual File

```go
// Load a specific catalog file
catalog, err := catalogs.LoadFromFile("./examples/ontologies/education.yaml")
if err != nil {
    log.Fatal(err)
}

// Register it
catalogs.RegisterCatalog(catalog)
```

### Option 3: Export Built-in Catalogs to YAML

```go
// Export a built-in catalog to YAML for customization
catalog, _ := catalogs.GetCatalog("financial")
err := catalogs.SaveToFile(catalog, "./my-custom-financial.yaml")
```

## YAML File Format

### Basic Structure

```yaml
domain: your_domain_name
description: Brief description of the domain

subdomains:
  - subdomain1
  - subdomain2

terms:
  - name: term_name
    synonyms:
      - synonym1
      - synonym2
    description: What this term means

entity_types:
  - entity_type: entity_name
    description: What this entity represents
    aliases:
      - alias1
      - alias2
    element_types:
      - paragraph
      - table_cell
    sample_rules:
      - type: keyword_match
        keywords:
          - keyword1
          - keyword2
      - type: regex_pattern
        pattern: '\\b[A-Z]{2,4}\\s*\\d{3,4}\\b'

common_entity_refs:
  - person
  - date
  - address
  # ... see common.go for all 30 available entities

relationships:
  - name: relationship_name
    source_type: source_entity
    target_type: target_entity
    relationship_type: related_to  # or part_of, is_a, etc.
    description: What this relationship means
    sample_patterns:
      - "{source} does something to {target}"
```

## Entity Extraction Rule Types

### 1. Keyword Match

Matches if any keyword is found in the text.

```yaml
- type: keyword_match
  keywords:
    - teacher
    - professor
    - instructor
```

### 2. Regex Pattern

Matches using regular expressions.

```yaml
- type: regex_pattern
  pattern: '\\b[A-Z][a-z]+\\s+University\\b'
```

### 3. Text Similarity

Matches based on semantic similarity to reference text.

```yaml
- type: text_similarity
  reference_text: "the student enrolled in"
  similarity_threshold: 0.7
```

### 4. Metadata Field

Extracts from document metadata fields.

```yaml
- type: metadata_field
  field_path: "author.institution"
```

### 5. JSONPath Query

Extracts using JSONPath expressions.

```yaml
- type: jsonpath_query
  jsonpath_expr: "$.students[*].name"
```

## Relationship Types

Use these standard relationship types:

- `is_a` - Inheritance/taxonomy
- `part_of` - Composition
- `related_to` - General association
- `located_in` - Spatial relationship
- `occurred_at` - Temporal relationship
- `created_by` - Authorship
- `mentions` - Reference
- `depends_on` - Dependency
- `implements` - Technical implementation
- `extends` - Technical extension
- `contains` - Containment
- `referenced_by` - Back-reference

## Common Entity References

Instead of redefining common entities like `person`, `date`, `email`, etc., reference them from the shared common entity templates:

**Available Common Entities** (30 total):

### Location
- `city`, `street`, `address`, `building`, `postal_code`, `country`, `region`

### Person
- `person`, `public_figure`, `role`, `executive`, `employee`

### Descriptive
- `color`, `size`, `dimension`, `weight`, `volume`, `material`, `texture`, `shape`

### Temporal
- `date`, `time`, `duration`

### Contact
- `email`, `phone`, `url`

### Numeric
- `percentage`, `number`

### Identifier
- `id_number`, `code`

## Example Use Cases

### 1. Industry-Specific Domain

Create a catalog for your specific industry (e.g., aviation, hospitality, construction):

```yaml
domain: aviation
description: Aviation operations, flight records, maintenance logs, and safety reports

entity_types:
  - entity_type: aircraft
    description: Aircraft or airplane
    aliases:
      - airplane
      - plane
      - jet
    sample_rules:
      - type: regex_pattern
        pattern: '\\b[A-Z]{1,3}-[A-Z]{3,5}\\b'  # Tail number
```

### 2. Company-Specific Domain

Create a catalog matching your company's internal terminology:

```yaml
domain: acme_corp_operations
description: Acme Corporation internal operations and processes

terms:
  - name: SKU
    synonyms:
      - product code
      - item number
      - ACME-ID
    description: Acme's internal product identifier

entity_types:
  - entity_type: acme_product
    description: Acme product with company-specific format
    sample_rules:
      - type: regex_pattern
        pattern: '\\bACME-[0-9]{6}\\b'
```

### 3. Multi-Language Domain

Define entities with multilingual keywords:

```yaml
entity_types:
  - entity_type: invoice
    description: Invoice or bill
    aliases:
      - factura     # Spanish
      - Rechnung    # German
      - facture     # French
    sample_rules:
      - type: keyword_match
        keywords:
          - invoice
          - factura
          - Rechnung
          - facture
```

## File Organization

Organize your catalog files however makes sense for your use case:

```
ontologies/
├── industries/
│   ├── aviation.yaml
│   ├── hospitality.yaml
│   └── construction.yaml
├── company-specific/
│   ├── acme-operations.yaml
│   └── acme-products.yaml
└── regional/
    ├── eu-compliance.yaml
    └── us-regulations.yaml
```

Load all at once:

```go
catalogs.RegisterFromDirectory("./ontologies/industries")
catalogs.RegisterFromDirectory("./ontologies/company-specific")
catalogs.RegisterFromDirectory("./ontologies/regional")
```

## Best Practices

1. **Start with an Example**: Export a built-in catalog as a template
2. **Use Common Entities**: Reference common entities instead of redefining them
3. **Test Extraction Rules**: Ensure regex patterns and keywords match your documents
4. **Document Thoroughly**: Add clear descriptions for future maintainers
5. **Version Control**: Keep catalog files in git for tracking changes
6. **Iterative Refinement**: Start simple, add complexity as needed

## Integration with Ontology Builder

The ontology builder can use your custom catalogs for domain suggestion:

```go
// Load custom catalogs
catalogs.RegisterFromDirectory("./my-catalogs")

// Create builder
builder, _ := ontology.NewOntologyBuilder(config)

// Builder will now suggest your custom domains via cosine similarity
result, _ := builder.Build(ctx)
```

## Examples in This Directory

- `education.yaml` - Complete education domain with K-12 and higher ed
- `financial.yaml` - Built-in financial domain (export example)
- `legal.yaml` - Built-in legal domain (export example)

## Need Help?

See the main catalog documentation: `go/internal/udml/ontology/catalogs/README.md`
