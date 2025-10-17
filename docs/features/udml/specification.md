# UDML Specification v1.0

**Universal Document Model Language (UDML)** - A comprehensive JSON-based format for representing parsed documents with rich semantic structure, relationships, and provenance tracking.

## Table of Contents

- [Overview](#overview)
- [Design Principles](#design-principles)
- [UDML Format Variants](#udml-format-variants)
- [Document Structure](#document-structure)
- [Element Types](#element-types)
- [Relationship Types](#relationship-types)
- [Navigation Patterns](#navigation-patterns)
- [Properties Bag](#properties-bag)
- [Provenance Tracking](#provenance-tracking)
- [Schema Version](#schema-version)
- [Examples](#examples)

---

## Overview

UDML is designed to be the universal interchange format for document parsing systems. It provides:

- **Unified representation** across all document types (PDF, DOCX, HTML, Markdown, etc.)
- **Rich semantic structure** with elements, relationships, and metadata
- **Provenance tracking** to trace every piece of data back to its source
- **Extensible properties** via properties bag pattern
- **LLM-friendly** structure for knowledge extraction and ontology mapping

### Key Features

1. **Format-agnostic**: Works with any document type
2. **Queryable**: Supports advanced JSONPath queries with regex
3. **Scalable**: Efficient storage in DuckDB/Parquet
4. **Version-controlled**: Track schema evolution over time
5. **Ontology-ready**: Direct mapping to knowledge graphs

---

## Design Principles

### 1. Promoted Fields Pattern

UDML uses a **properties bag** for extensibility while promoting frequently-accessed fields to top level:

```json
{
  "element_id": "elem_abc123",           // PROMOTED: Always needed for joins
  "element_type": "section",             // PROMOTED: Core classification
  "content": "Introduction",             // PROMOTED: Primary data
  "properties": {                        // BAG: Domain-specific fields
    "page_number": 1,
    "section_level": 1,
    "font_size": 14
  }
}
```

**Why promote fields?**
- `element_id`: Required for relationships and joins
- `element_type`: Essential for filtering and classification
- `content`: Primary payload accessed in 90%+ of queries
- `properties`: Flexible bag for everything else

### 2. Flat Element Array

Elements are stored in a **flat array**, not nested hierarchy:

```json
{
  "elements": [
    {"element_id": "1", "element_type": "section", "content": "Chapter 1"},
    {"element_id": "2", "element_type": "paragraph", "content": "Introduction text"},
    {"element_id": "3", "element_type": "paragraph", "content": "More content"}
  ],
  "relationships": [
    {"source_id": "1", "target_id": "2", "relationship_type": "contains"},
    {"source_id": "1", "target_id": "3", "relationship_type": "contains"}
  ]
}
```

**Benefits:**
- Efficient querying without recursive traversal
- Clean relationship modeling
- Easy to filter and aggregate
- Avoids JSON nesting depth limits

### 3. Explicit Relationships

Relationships are **first-class citizens**, not implicit in structure:

```json
{
  "relationships": [
    {
      "source_id": "chapter_1",
      "target_id": "section_1_1",
      "relationship_type": "contains",
      "confidence": 1.0
    },
    {
      "source_id": "section_1_1",
      "target_id": "table_1",
      "relationship_type": "references",
      "confidence": 0.95
    }
  ]
}
```

**Advantages:**
- Query relationships independently
- Support many-to-many relationships
- Add confidence scores
- Easy to export to graph databases

---

## UDML Format Variants

UDML supports **three representation formats** that serve different purposes in the document processing pipeline:

### Format 1: Persistence Format (Flat Array)

**Purpose**: Storage, schema validation, query optimization

**Used by**:
- Parsers (output format)
- Databases (PostgreSQL, DuckDB)
- Parquet files
- JSON Schema validation
- SQL queries

**Structure**: Elements in flat array with structural metadata fields

```json
{
  "document_id": "doc-001",
  "elements": [
    {
      "element_id": "elem-1",
      "element_type": "section",
      "content": "Introduction",
      "parent_id": "",
      "position": 0
    },
    {
      "element_id": "elem-2",
      "element_type": "paragraph",
      "content": "First paragraph",
      "parent_id": "elem-1",
      "position": 1
    },
    {
      "element_id": "elem-3",
      "element_type": "paragraph",
      "content": "Second paragraph",
      "parent_id": "elem-1",
      "position": 2
    }
  ],
  "relationships": [
    {
      "source_id": "elem-1",
      "target_id": "elem-2",
      "relationship_type": "contains"
    }
  ]
}
```

**Key Characteristics**:
- Elements stored in flat array (no nesting)
- `parent_id` field links child → parent
- `position` field provides global document ordering (never restarts)
- `row_index`, `column_index` provide local structural addressing
- Separate `relationships` array for semantic links
- Efficient for SQL queries and columnar storage

**Navigation Pattern** (SQL):
```sql
-- All children of an element (ordered)
SELECT * FROM elements
WHERE parent_id = 'elem-1'
ORDER BY position;

-- Next sibling
SELECT * FROM elements
WHERE parent_id = (SELECT parent_id FROM elements WHERE element_id = 'elem-2')
AND position > (SELECT position FROM elements WHERE element_id = 'elem-2')
ORDER BY position LIMIT 1;

-- Parent of an element
SELECT * FROM elements
WHERE element_id = (SELECT parent_id FROM elements WHERE element_id = 'elem-2');
```

### Format 2: Hierarchical Format (Nested Children)

**Purpose**: JSONPath queries, ontology extraction, LLM context, visualization

**Used by**:
- Ontology extractor (JSONPath evaluation)
- LLM prompts (document context)
- Document reconstruction (`DocumentBuilder`)
- Nested format exports (XML, nested JSON)
- Tree visualization tools

**Structure**: Nested elements with `children` arrays

```json
{
  "doc_id": "doc-001",
  "elements": [
    {
      "element_id": "elem-1",
      "element_type": "section",
      "content": "Introduction",
      "element_order": 0,
      "children": [
        {
          "element_id": "elem-2",
          "element_type": "paragraph",
          "content": "First paragraph",
          "element_order": 1,
          "children": []
        },
        {
          "element_id": "elem-3",
          "element_type": "paragraph",
          "content": "Second paragraph",
          "element_order": 2,
          "children": []
        }
      ]
    }
  ]
}
```

**Key Characteristics**:
- Elements nested via `children` arrays
- No `parent_id` field (parent-child relationship implicit in structure)
- `element_order` for sibling ordering (renamed from `position` for clarity)
- Tree structure mirrors document hierarchy
- Efficient for recursive traversal and JSONPath queries

**Navigation Pattern** (JSONPath):
```jsonpath
# All children of an element
$.elements[?(@.element_id=='elem-1')].children

# All paragraphs at any depth
$..children[?(@.element_type=='paragraph')]

# All grandchildren
$.elements[?(@.element_id=='elem-1')].children[*].children

# First child
$.elements[?(@.element_id=='elem-1')].children[0]

# All headings with specific level
$..children[?(@.element_type=='heading' && @.properties.section_level==2)]
```

### Format 3: Graph Format (UDML-G)

**Purpose**: Graph database export, RDF/OWL ontologies, network analysis, graph visualization

**Used by**:
- Neo4j, ArangoDB, TigerGraph exports
- RDF triple stores (Fuseki, GraphDB)
- NetworkX and graph analytics
- Cytoscape and graph visualization tools
- Knowledge graph construction

**Structure**: Nodes and edges (property graph model)

```json
{
  "document_id": "doc-001",
  "nodes": [
    {
      "id": "elem-1",
      "type": "section",
      "content": "Introduction",
      "properties": {
        "position": 0
      }
    },
    {
      "id": "elem-2",
      "type": "paragraph",
      "content": "First paragraph",
      "properties": {
        "position": 1
      }
    },
    {
      "id": "elem-3",
      "type": "paragraph",
      "content": "Second paragraph",
      "properties": {
        "position": 2
      }
    }
  ],
  "edges": [
    {
      "source": "elem-1",
      "target": "elem-2",
      "type": "contains",
      "confidence": 1.0
    },
    {
      "source": "elem-1",
      "target": "elem-3",
      "type": "contains",
      "confidence": 1.0
    }
  ]
}
```

**Key Characteristics**:
- **Nodes** array contains all vertices (maps from `elements`)
- **Edges** array contains all relationships (maps from `relationships`)
- Simplified field names: `id` (not `element_id`), `type` (not `element_type`)
- Direct mapping to property graph databases (Neo4j, ArangoDB)
- Compatible with RDF export via triples: `(source, type, target)`
- Optimized for graph algorithms (PageRank, community detection, shortest path)

**Navigation Pattern** (Cypher for Neo4j):
```cypher
// All children of a node
MATCH (parent {id: 'elem-1'})-[r:contains]->(child)
RETURN child
ORDER BY child.properties.position;

// All descendants (recursive)
MATCH path = (ancestor {id: 'elem-1'})-[:contains*]->(descendant)
RETURN descendant;

// Siblings (nodes with same parent)
MATCH (parent)-[:contains]->(sibling)
WHERE EXISTS {
  MATCH (parent)-[:contains]->(node {id: 'elem-2'})
}
RETURN sibling
ORDER BY sibling.properties.position;
```

**Use Cases**:
- **Knowledge graphs**: Build domain-specific ontologies from document structure
- **Graph analytics**: Run PageRank, centrality, community detection algorithms
- **Relationship mining**: Discover implicit relationships between elements
- **Provenance graphs**: Track data lineage across transformations
- **Visualization**: Render document structure as interactive graph

**Export Targets**:
- **Neo4j**: Direct import via Cypher `CREATE` statements or CSV import
- **RDF/OWL**: Convert to triples `(subject, predicate, object)` format
- **NetworkX**: Python graph analysis library
- **GraphML**: XML-based graph format for Gephi, Cytoscape
- **DOT/Graphviz**: Visualization and layout

### Format Conversion

The hierarchical format is **built on-demand** from the persistence format:

**Go Implementation**:
```go
import "github.com/kennethstott/doculyzer-go-conversion/internal/udml/builder"

// Build hierarchical document from flat elements
builder := builder.NewDocumentBuilder(backend, builder.DefaultBuildOptions())
hierarchicalDoc, err := builder.BuildDocument(ctx, docID)

// Access nested structure
for _, element := range hierarchicalDoc.Elements {
    processElement(element)
    for _, child := range element.Children {
        processChild(child)
    }
}
```

**Conversion Process**:
1. Query all elements for document (flat format)
2. Build element map by `element_id`
3. Iterate elements and attach to parent via `parent_id`
4. Sort children by `position` (renamed to `element_order` in hierarchical format)
5. Return root elements with nested children

### Field Semantics Across Formats

| Field | Persistence Format | Hierarchical Format | Graph Format (UDML-G) | Purpose |
|-------|-------------------|---------------------|-----------------------|---------|
| `element_id` / `id` | `element_id` | `element_id` | `id` (simplified) | Unique identifier |
| `element_type` / `type` | `element_type` | `element_type` | `type` (simplified) | Element classification |
| `parent_id` | ✓ Present (upward link) | ✗ Absent (implicit) | ✗ Absent (edges only) | Child→parent navigation |
| `position` | ✓ Present (global counter) | Renamed to `element_order` | In `properties` bag | Document-wide ordering |
| `children` | ✗ Absent (flat array) | ✓ Present (nested array) | ✗ Absent (edges only) | Parent→child navigation |
| `row_index` | ✓ Present | ✓ Present | In `properties` bag | Local table row address |
| `column_index` | ✓ Present | ✓ Present | In `properties` bag | Local table column address |
| `relationships` / `edges` | `relationships` array | Not present | `edges` array (renamed) | Explicit relationships |

**Position vs Index Semantics**:
- **position** (global): Document sequence number (0, 1, 2, 3...) - never restarts, used for ordering siblings
- **{x}_index** (local): Structural address within container (row_index, column_index) - used for addressing specific positions

### When to Use Each Format

| Task | Format | Reason |
|------|--------|--------|
| Parser output | Persistence (flat) | Standard output format, schema validation |
| Database storage | Persistence (flat) | Efficient queries, columnar storage (Parquet) |
| SQL queries | Persistence (flat) | Promoted fields enable fast WHERE clauses |
| JSONPath queries | Hierarchical (nested) | JSONPath requires nested structure |
| Ontology extraction | Hierarchical (nested) | Rule-based extraction uses JSONPath expressions |
| LLM prompts | Hierarchical (nested) | Preserve document structure for context |
| Graph database export | Graph (UDML-G) | Direct import to Neo4j, ArangoDB, TigerGraph |
| RDF/OWL export | Graph (UDML-G) | Convert to RDF triples for semantic web |
| Graph analytics | Graph (UDML-G) | Run PageRank, centrality, community detection |
| Network visualization | Graph (UDML-G) | Render as force-directed graphs (Cytoscape, Gephi) |
| Knowledge graphs | Graph (UDML-G) | Build domain ontologies and semantic networks |
| Tree visualization | Hierarchical (nested) | Tree rendering for document structure |

### Schema Validation

**Persistence Format**: Validated by `schemas/udml-1.0.schema.json`

**Hierarchical Format**: Validated by `schemas/udml-1.0-hierarchical.schema.json`

**Graph Format**: Validated by `schemas/udml-1.0-graph.schema.json`

All three formats represent the same logical document structure but optimize for different access patterns and use cases.

**📖 See [JSON Schema Validation Guide](features/udml/schemas.md)** for complete validation documentation, usage examples in Python/Node.js/Go, IDE integration, and CI/CD setup.

---

## Document Structure

### Top-Level Schema

```json
{
  "document_id": "string (required)",
  "metadata": { /* DocumentMetadata */ },
  "elements": [ /* Element[] */ ],
  "relationships": [ /* Relationship[] */ ]
}
```

### Document Metadata

```json
{
  "title": "string (optional)",
  "author": "string (optional)",
  "created_at": "ISO8601 timestamp (optional)",
  "modified_at": "ISO8601 timestamp (optional)",
  "source_type": "string (required)",  // "pdf", "docx", "html", etc.
  "source_path": "string (optional)",
  "page_count": "integer (optional)",
  "word_count": "integer (optional)",
  "language": "string (optional)",    // ISO 639-1 code
  "custom": { /* extensible metadata */ }
}
```

---

## Element Types

UDML supports 55 element types organized into the following categories:

### Document Structure

| Type | Description | Common Use Cases |
|------|-------------|------------------|
| `root` | Document root container | Top-level element |
| `body` | Main document body | Body content container |
| `section` | Logical section or chapter | Chapters, divisions |
| `document` | Document-level metadata | Document wrapper |

### Text Elements

| Type | Description | Common Use Cases |
|------|-------------|------------------|
| `paragraph` | Text paragraph | Body text, content |
| `text` | Raw text content | Plain text elements |
| `heading` | Section heading | Titles, headers |
| `header` | Header region | Page headers |

### List Elements

| Type | Description | Common Use Cases |
|------|-------------|------------------|
| `list` | Ordered or unordered list | Bullet lists, numbered lists |
| `list_item` | Individual list item | List entries |

### Table Elements

| Type | Description | Common Use Cases |
|------|-------------|------------------|
| `table` | Generic table structure | Simple tables |
| `data_table` | Data-oriented table | Structured data tables |
| `data_tables` | Collection of data tables | Multiple table container |
| `table_header` | Table header section | Column headers |
| `table_header_row` | Header row | Header row container |
| `table_header_cell` | Header cell | Column header cell |
| `table_row` | Table row | Data row |
| `table_cell` | Table cell | Cell content |

### Media Elements

| Type | Description | Common Use Cases |
|------|-------------|------------------|
| `image` | Image or photo | Photos, graphics |
| `figure` | Figure with caption | Labeled images |
| `chart` | Chart or graph | Visualizations, plots |

### Code Elements

| Type | Description | Common Use Cases |
|------|-------------|------------------|
| `code_block` | Code snippet | Source code, scripts |

### HTML/Web Elements

| Type | Description | Common Use Cases |
|------|-------------|------------------|
| `div` | HTML div element | Sections, containers |
| `span` | HTML span element | Inline content |
| `article` | HTML article element | Article content |
| `blockquote` | Block quotation | Quoted text |

### Structured Data - JSON

| Type | Description | Common Use Cases |
|------|-------------|------------------|
| `json_field` | JSON field/property | Object property |
| `json_object` | JSON object | Nested object |
| `json_array` | JSON array | Array container |
| `json_item` | JSON array item | Array element |

### Structured Data - XML

| Type | Description | Common Use Cases |
|------|-------------|------------------|
| `xml_element` | XML element | XML nodes |

### Office Documents - Word

| Type | Description | Common Use Cases |
|------|-------------|------------------|
| `headers` | Document headers | Page headers |
| `footers` | Document footers | Page footers |
| `comments` | Document comments | Review comments container |

### Office Documents - Excel

| Type | Description | Common Use Cases |
|------|-------------|------------------|
| `sheet` | Excel worksheet | Individual sheet |
| `workbook` | Excel workbook | Workbook container |
| `comment` | Cell comment | Cell annotation |
| `merged_cell` | Merged cell | Spanning cell |
| `merged_cells` | Merged cells container | Multiple merged cells |

### Office Documents - PowerPoint

| Type | Description | Common Use Cases |
|------|-------------|------------------|
| `slide` | Presentation slide | Individual slide |
| `slide_notes` | Speaker notes | Notes section |
| `shape_group` | Shape group | Grouped shapes |

### PDF-Specific

| Type | Description | Common Use Cases |
|------|-------------|------------------|
| `page` | PDF page | Individual page |

### Markdown-Specific

| Type | Description | Common Use Cases |
|------|-------------|------------------|
| `front_matter` | YAML/TOML front matter | Metadata block |

### Text Parser

| Type | Description | Common Use Cases |
|------|-------------|------------------|
| `line` | Text line | Single line |
| `range` | Text range | Character range |
| `substring` | Text substring | Substring extraction |

### Metadata & Other

| Type | Description | Common Use Cases |
|------|-------------|------------------|
| `formula` | Mathematical formula | Equations, expressions |
| `footnote` | Footnote reference | Citations, notes |
| `link` | Hyperlink | URLs, references |
| `quote` | Quotation | Quoted text |
| `caption` | Caption text | Image/table captions |
| `metadata` | Metadata element | Descriptive metadata |
| `other` | Other/unclassified | Fallback type |

### Element Schema

```json
{
  "element_id": "string (required)",      // Unique identifier
  "element_type": "string (required)",    // From ElementType enum
  "content": "string (required)",         // Primary content
  "properties": {                         // Extensible properties bag
    "page_number": "integer",
    "section_level": "integer",
    "row_index": "integer",
    "column_index": "integer",
    "temporal_type": "string",
    "tag_name": "string"
  },
  "provider_name": "string (optional)",   // Parser that created element
  "confidence": "float (optional)",       // 0.0-1.0 confidence score
  "embeddings": {                         // Optional embeddings
    "model": "string",
    "vector": [float array],
    "dimensions": integer
  }
}
```

### Promoted Properties

These properties are frequently used and promoted to the `properties` bag by convention:

| Property | Type | Description |
|----------|------|-------------|
| `page_number` | integer | Page number (1-indexed) |
| `section_level` | integer | Heading level (1-6) |
| `row_index` | integer | Table row index (0-indexed) |
| `column_index` | integer | Table column index (0-indexed) |
| `temporal_type` | string | Temporal classification ("date", "time", "duration") |
| `tag_name` | string | HTML/XML tag name |

---

## Relationship Types

### Simplified Relationship Model

UDML v1.0 uses a **simplified relationship model** with a single relationship type: `contains`.

**Rationale**:
- **Structural navigation** is handled via promoted fields (`parent_id`, `position`, `row_index`, `column_index`)
- **Sequential ordering** is represented by the `position` field (global document sequence)
- **Hierarchy traversal** uses `parent_id` (upward) or `children` array in hierarchical format (downward)
- **Relationships** are reserved for **semantic links** only

### Core Relationship Type

| Type | Description | Example | Direction |
|------|-------------|---------|-----------|
| `contains` | Parent-child containment | Section contains paragraphs | Unidirectional (parent→child) |

**Key Design Decision**: Relationships are **unidirectional** (parent→child only). Child→parent navigation uses the `parent_id` field directly, avoiding redundant bidirectional relationships.

### Relationship Schema

```json
{
  "source_id": "string (required)",           // Source element ID
  "target_id": "string (required)",           // Target element ID
  "relationship_type": "string (required)",   // From RelationshipType enum
  "confidence": "float (optional)",           // 0.0-1.0 confidence
  "properties": {                             // Extensible properties
    "weight": "float",
    "label": "string",
    "metadata": "object"
  }
}
```

---

## Navigation Patterns

UDML provides multiple navigation strategies depending on the format and access pattern needed.

### Persistence Format Navigation (SQL)

When querying the flat/persistence format, use these SQL patterns:

#### Parent→Child Navigation
```sql
-- All direct children of an element (ordered by position)
SELECT * FROM elements
WHERE parent_id = 'parent-elem-id'
ORDER BY position;

-- Count children
SELECT COUNT(*) FROM elements
WHERE parent_id = 'parent-elem-id';

-- First child
SELECT * FROM elements
WHERE parent_id = 'parent-elem-id'
ORDER BY position LIMIT 1;

-- Last child
SELECT * FROM elements
WHERE parent_id = 'parent-elem-id'
ORDER BY position DESC LIMIT 1;
```

#### Child→Parent Navigation
```sql
-- Get parent of an element
SELECT * FROM elements
WHERE element_id = (
    SELECT parent_id FROM elements
    WHERE element_id = 'child-elem-id'
);

-- Get all ancestors (recursive)
WITH RECURSIVE ancestors AS (
    SELECT * FROM elements WHERE element_id = 'start-elem-id'
    UNION ALL
    SELECT e.* FROM elements e
    JOIN ancestors a ON e.element_id = a.parent_id
    WHERE a.parent_id IS NOT NULL
)
SELECT * FROM ancestors;
```

#### Sibling Navigation
```sql
-- Next sibling (same parent, higher position)
SELECT * FROM elements
WHERE parent_id = (SELECT parent_id FROM elements WHERE element_id = 'elem-id')
AND position > (SELECT position FROM elements WHERE element_id = 'elem-id')
ORDER BY position LIMIT 1;

-- Previous sibling (same parent, lower position)
SELECT * FROM elements
WHERE parent_id = (SELECT parent_id FROM elements WHERE element_id = 'elem-id')
AND position < (SELECT position FROM elements WHERE element_id = 'elem-id')
ORDER BY position DESC LIMIT 1;

-- All siblings (including self)
SELECT * FROM elements
WHERE parent_id = (SELECT parent_id FROM elements WHERE element_id = 'elem-id')
ORDER BY position;

-- All siblings (excluding self)
SELECT * FROM elements
WHERE parent_id = (SELECT parent_id FROM elements WHERE element_id = 'elem-id')
AND element_id != 'elem-id'
ORDER BY position;
```

#### Table Navigation
```sql
-- All cells in a specific row (ordered by column)
SELECT * FROM elements
WHERE parent_id = 'row-elem-id'
AND element_type = 'table_cell'
ORDER BY column_index;

-- Cell in specific row and column
SELECT * FROM elements
WHERE element_type = 'table_cell'
AND row_index = 3
AND column_index = 2;

-- Next cell in row (same row, next column)
SELECT * FROM elements
WHERE parent_id = 'row-elem-id'
AND column_index > (SELECT column_index FROM elements WHERE element_id = 'cell-id')
ORDER BY column_index LIMIT 1;

-- All rows in table (ordered)
SELECT * FROM elements
WHERE parent_id = 'table-elem-id'
AND element_type = 'table_row'
ORDER BY row_index;
```

### Hierarchical Format Navigation (JSONPath)

When using the hierarchical/nested format (built via `DocumentBuilder`), use JSONPath:

#### Basic Navigation
```jsonpath
# All direct children of root element
$.elements[0].children

# All grandchildren
$.elements[0].children[*].children

# Nth child (0-indexed)
$.elements[?(@.element_id=='parent-1')].children[2]

# First child
$.elements[?(@.element_id=='parent-1')].children[0]

# Last child
$.elements[?(@.element_id=='parent-1')].children[-1]
```

#### Recursive Descent
```jsonpath
# All paragraphs at any depth
$..children[?(@.element_type=='paragraph')]

# All elements with specific attribute at any depth
$..children[?(@.properties.page_number==5)]

# All headings at any depth with specific level
$..children[?(@.element_type=='heading' && @.properties.section_level==2)]
```

#### Type-Based Queries
```jsonpath
# All tables at any depth
$..children[?(@.element_type=='table')]

# All code blocks
$..children[?(@.element_type=='code_block')]

# All list items
$..children[?(@.element_type=='list_item')]
```

#### Filtering by Properties
```jsonpath
# Elements on specific page
$..children[?(@.properties.page_number==10)]

# Table cells in specific row
$..children[?(@.element_type=='table_row' && @.properties.row_index==3)].children[?(@.element_type=='table_cell')]

# High-confidence elements
$..children[?(@.confidence > 0.9)]
```

### Format-Specific Use Cases

| Navigation Task | Best Format | Query Approach |
|----------------|-------------|----------------|
| Find next sibling | Persistence (SQL) | `WHERE parent_id=$same AND position > $current` |
| Get all descendants of type X | Hierarchical (JSONPath) | `$..children[?(@.element_type=='X')]` |
| Table cell lookup (row, col) | Persistence (SQL) | `WHERE row_index=$r AND column_index=$c` |
| Traverse tree depth-first | Hierarchical (Go) | Recursive function on `Children` array |
| Count elements by type | Persistence (SQL) | `SELECT element_type, COUNT(*) GROUP BY element_type` |
| Find elements on page N | Persistence (SQL) | `WHERE page_number=$n` (promoted field query) |
| Extract all headings in order | Hierarchical (JSONPath) | `$..children[?(@.element_type=='heading')]` |
| Get parent of element | Persistence (SQL) | `WHERE element_id = (SELECT parent_id ...)` |

### Performance Considerations

**Flat Format (SQL)**:
- ✓ Fast promoted field queries (page_number, row_index, column_index) - indexed columns
- ✓ Efficient sibling navigation (position field)
- ✓ Parent lookup (parent_id field)
- ✗ Recursive queries (ancestors, descendants) are expensive

**Hierarchical Format (JSONPath)**:
- ✓ Fast recursive descent (`$..children`)
- ✓ Natural tree traversal
- ✗ Must build hierarchy first (conversion overhead)
- ✗ Not suitable for aggregations or large-scale queries

**Best Practice**: Use flat format for querying/filtering, convert to hierarchical format for processing results.

---

## Properties Bag

The `properties` object is an **extensible properties bag** that allows parsers to add domain-specific metadata without modifying the core schema.

### Best Practices

1. **Use standard names** when possible (see promoted properties)
2. **Namespace custom properties**: `pdf:bookmark_level`, `docx:style_name`
3. **Keep values JSON-serializable**: strings, numbers, booleans, arrays, objects
4. **Document custom properties** in parser documentation

### Examples

**PDF-specific properties:**
```json
{
  "properties": {
    "page_number": 15,
    "pdf:bookmark_level": 2,
    "pdf:annotation_type": "highlight",
    "pdf:font_name": "Times New Roman"
  }
}
```

**HTML-specific properties:**
```json
{
  "properties": {
    "tag_name": "div",
    "html:class_name": "content-section",
    "html:id": "introduction",
    "html:href": "https://example.com"
  }
}
```

**Table-specific properties:**
```json
{
  "properties": {
    "row_index": 3,
    "column_index": 2,
    "table:header_row": false,
    "table:colspan": 1,
    "table:rowspan": 2
  }
}
```

---

## Provenance Tracking

UDML supports **full provenance tracking** to trace every element back to its source document and parser.

### Provenance Fields

```json
{
  "element_id": "elem_abc123",
  "provider_name": "pypdf-parser-v2.1",   // Parser name and version
  "confidence": 0.95,                     // Parser confidence
  "properties": {
    "provenance": {
      "document_id": "doc_original_123",
      "source_file": "/data/documents/report.pdf",
      "extraction_timestamp": "2024-01-15T10:30:00Z",
      "parser_version": "2.1.0",
      "page_number": 5,
      "bounding_box": [100, 200, 300, 250]
    }
  }
}
```

### Benefits

- **Debugging**: Trace parsing errors to source
- **Validation**: Verify extracted data against original
- **Auditing**: Full audit trail for compliance
- **Reprocessing**: Re-extract specific elements

---

## Schema Version

UDML uses semantic versioning to track schema evolution.

### Version Metadata

```json
{
  "document_id": "doc_123",
  "schema_version": "1.0.0",
  "metadata": {
    "udml_version": "1.0.0",
    "parser_version": "2.1.0",
    "created_at": "2024-01-15T10:30:00Z"
  }
}
```

### Version History

| Version | Changes |
|---------|---------|
| 1.0.0 | Initial UDML specification with 6 promoted properties |

---

## Examples

### Example 1: Simple Document

```json
{
  "document_id": "simple_doc_001",
  "metadata": {
    "title": "Introduction to UDML",
    "source_type": "markdown",
    "created_at": "2024-01-15T10:00:00Z"
  },
  "elements": [
    {
      "element_id": "elem_001",
      "element_type": "heading",
      "content": "Introduction",
      "properties": {
        "section_level": 1
      },
      "provider_name": "markdown-parser",
      "confidence": 1.0
    },
    {
      "element_id": "elem_002",
      "element_type": "paragraph",
      "content": "UDML is a universal document model.",
      "properties": {},
      "provider_name": "markdown-parser",
      "confidence": 1.0
    }
  ],
  "relationships": [
    {
      "source_id": "elem_001",
      "target_id": "elem_002",
      "relationship_type": "contains",
      "confidence": 1.0
    }
  ]
}
```

### Example 2: PDF with Tables

```json
{
  "document_id": "pdf_report_001",
  "metadata": {
    "title": "Q4 Financial Report",
    "source_type": "pdf",
    "page_count": 25,
    "created_at": "2024-01-15T10:00:00Z"
  },
  "elements": [
    {
      "element_id": "elem_table_001",
      "element_type": "table",
      "content": "Revenue Summary Table",
      "properties": {
        "page_number": 5,
        "table:row_count": 10,
        "table:column_count": 4
      },
      "provider_name": "pypdf-parser",
      "confidence": 0.95
    },
    {
      "element_id": "elem_cell_001",
      "element_type": "table_cell",
      "content": "$1,234,567",
      "properties": {
        "page_number": 5,
        "row_index": 2,
        "column_index": 3,
        "table:header": false
      },
      "provider_name": "pypdf-parser",
      "confidence": 0.98
    }
  ],
  "relationships": [
    {
      "source_id": "elem_table_001",
      "target_id": "elem_cell_001",
      "relationship_type": "contains",
      "confidence": 1.0
    }
  ]
}
```

### Example 3: Web Page with Links

```json
{
  "document_id": "webpage_001",
  "metadata": {
    "title": "Product Documentation",
    "source_type": "html",
    "source_path": "https://example.com/docs",
    "created_at": "2024-01-15T10:00:00Z"
  },
  "elements": [
    {
      "element_id": "elem_heading_001",
      "element_type": "heading",
      "content": "Getting Started",
      "properties": {
        "section_level": 2,
        "tag_name": "h2",
        "html:id": "getting-started"
      },
      "provider_name": "html-parser",
      "confidence": 1.0
    },
    {
      "element_id": "elem_link_001",
      "element_type": "link",
      "content": "Installation Guide",
      "properties": {
        "html:href": "https://example.com/docs/install",
        "html:target": "_blank"
      },
      "provider_name": "html-parser",
      "confidence": 1.0
    }
  ],
  "relationships": [
    {
      "source_id": "elem_heading_001",
      "target_id": "elem_link_001",
      "relationship_type": "contains",
      "confidence": 1.0
    },
    {
      "source_id": "elem_link_001",
      "target_id": "external_page",
      "relationship_type": "links_to",
      "confidence": 1.0,
      "properties": {
        "url": "https://example.com/docs/install"
      }
    }
  ]
}
```

---

## Query Examples

UDML is designed to be efficiently queried using DuckDB SQL or JSONPath.

### DuckDB Queries

```sql
-- Find all sections on page 5
SELECT element_id, content
FROM udml_elements
WHERE element_type = 'section'
AND properties->>'page_number' = '5';

-- Count elements by type
SELECT element_type, COUNT(*) as count
FROM udml_elements
GROUP BY element_type
ORDER BY count DESC;

-- Find tables with high confidence
SELECT element_id, content, confidence
FROM udml_elements
WHERE element_type = 'table'
AND confidence > 0.9;

-- Get relationship graph for element
SELECT r.relationship_type, e.content as target_content
FROM udml_relationships r
JOIN udml_elements e ON r.target_id = e.element_id
WHERE r.source_id = 'elem_001';
```

### JSONPath Queries

```jsonpath
-- All section elements
$.elements[?(@.element_type=='section')]

-- Headings on page 1
$.elements[?(@.element_type=='heading' && @.properties.page_number==1)]

-- High confidence elements
$.elements[?(@.confidence > 0.9)]

-- Elements from specific parser
$.elements[?(@.provider_name=='pypdf-parser')]

-- Tables with more than 5 rows
$.elements[?(@.element_type=='table' && @.properties.table:row_count > 5)]
```

---

## Schema Evolution

UDML supports schema evolution through versioning:

1. **Backward compatible changes**: Add new optional fields
2. **Breaking changes**: Increment major version
3. **Migration tools**: Provided for version upgrades

### Future Extensions

Planned for future versions:

- **UDML 1.1**: Add `speaker_id` and `geo_location` promoted properties
- **UDML 2.0**: Add support for multimedia (audio/video) elements
- **UDML 3.0**: Add semantic annotation layers

---

## Validation

UDML documents should validate against the following rules:

1. **Required fields**: `document_id`, `element_id`, `element_type`, `content`
2. **Valid types**: Element types must be from defined enum
3. **Unique IDs**: All `element_id` values must be unique
4. **Valid relationships**: `source_id` and `target_id` must reference existing elements
5. **Confidence range**: Confidence scores must be in range [0.0, 1.0]

### JSON Schema

A formal JSON Schema (Draft 07) for UDML validation is available at:
**`schemas/udml-1.0.schema.json`**

**Schema Features:**
- Validates document structure (document_id, metadata, elements, relationships)
- Enforces required fields and data types
- Validates element types against enum
- Validates relationship types against enum
- Ensures confidence scores are in range [0.0, 1.0]
- Supports extensible properties bag
- Validates promoted properties (page_number, section_level, etc.)

**Validation Examples:**

Using Python:
```python
import json
import jsonschema

# Load schema
with open('schemas/udml-1.0.schema.json') as f:
    schema = json.load(f)

# Load UDML document
with open('document.json') as f:
    doc = json.load(f)

# Validate
try:
    jsonschema.validate(doc, schema)
    print("✓ Valid UDML document")
except jsonschema.ValidationError as e:
    print(f"✗ Invalid: {e.message}")
```

Using Node.js (AJV):
```javascript
const Ajv = require('ajv');
const ajv = new Ajv();

const schema = require('./schemas/udml-1.0.schema.json');
const doc = require('./document.json');

const validate = ajv.compile(schema);
const valid = validate(doc);

if (valid) {
    console.log('✓ Valid UDML document');
} else {
    console.log('✗ Invalid:', validate.errors);
}
```

Using Go:
```go
import (
    "github.com/xeipuuv/gojsonschema"
)

schemaLoader := gojsonschema.NewReferenceLoader("file://schemas/udml-1.0.schema.json")
docLoader := gojsonschema.NewReferenceLoader("file://document.json")

result, err := gojsonschema.Validate(schemaLoader, docLoader)
if err != nil {
    panic(err)
}

if result.Valid() {
    fmt.Println("✓ Valid UDML document")
} else {
    for _, err := range result.Errors() {
        fmt.Printf("✗ %s\n", err)
    }
}
```

**Schema URL:**
```
https://github.com/kennethstott/doculyzer-go-conversion/schemas/udml-1.0.schema.json
```

---

## Best Practices

1. **Use consistent IDs**: Generate UUIDs or prefixed IDs (`elem_`, `rel_`)
2. **Set confidence scores**: Always include parser confidence
3. **Add provenance**: Include `provider_name` and timestamps
4. **Namespace custom properties**: Use `parser:property` format
5. **Document extensions**: Document any custom properties or types
6. **Validate output**: Run UDML validator before storage
7. **Version tracking**: Include schema version in metadata

---

## Tools and Libraries

### Python
- `udml-py`: Python library for UDML creation and validation
- `udml-validator`: CLI tool for validation

### Go
- `github.com/kennethstott/doculyzer-go-conversion/internal/udml`: Go UDML implementation
- Includes query backend, ontology extraction, and graph export

### Utilities
- `udml-convert`: Convert between UDML versions
- `udml-query`: Query UDML documents via CLI
- `udml-viz`: Visualize UDML structure

---

## References

- [UDML Migration Plan](../UDML_MIGRATION_PLAN.md)
- [JSONPath Extensions Guide](./JSONPATH_EXTENSIONS.md)
- [Ontology YAML Schema](./ONTOLOGY_YAML_SCHEMA.md)
- [Export Formats Guide](./EXPORT_FORMATS.md)

---

## License

UDML Specification is released under CC BY 4.0.

## Version

- **Specification Version**: 1.0.0
- **Last Updated**: 2024-10-13
- **Status**: Stable

---

## Related Documentation

- **Next**: [UDML Schemas](schemas.md)
- **Up**: [Documentation Home](../../README.md)

### Quick Links

- [Documentation Home](../../README.md)
- [Quick Reference](../../../QUICK_REFERENCE.md)
- [Configuration Overview](../../configuration/README.md)
- [Troubleshooting](../../operations/troubleshooting.md)
