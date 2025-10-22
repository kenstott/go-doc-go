# UDML Ontology Management CLI

Unified command-line tool for creating, extracting, and analyzing ontologies from UDML document corpora.

## Overview

The Ontology CLI provides three main capabilities:

### 1. Interview (Schema Creation)
Automatically generates ontology schemas from UDML Parquet storage:
- Domain detection within your document corpus
- Entity types (organizations, people, products, concepts, etc.)
- Relationship types between entities
- Extraction rules (keywords, regex patterns, metadata paths)

### 2. Extract (Entity Extraction)
Applies ontology schemas to extract entities and relationships from documents:
- Rule-based extraction (fast, deterministic, no LLM costs)
- Optional LLM validation for high-precision extraction
- Distributed processing with job control

### 3. Analyze-Graph (Knowledge Graph Analysis)
Converts extracted ontologies into knowledge graphs with advanced analysis:
- Community detection (Louvain, Label Propagation)
- Graph query engine with native Go API
- RDF export (N-Triples, Turtle, RDF/XML)
- MCP server integration for LLM-driven exploration

## Architecture

**Three-Phase Design:**
1. **Ontology Building** (ONE-TIME, uses LLM):
   - Samples corpus using stratified sampling
   - Analyzes entity frequencies
   - Uses Claude API to identify domains and generate extraction rules
   - Automatically detects aliases for frequent entities
   - Outputs: OntologySchema JSON

2. **Entity Extraction** (RUNTIME, minimal LLM):
   - Uses the generated OntologySchema
   - Applies rules deterministically
   - Optional LLM validation for precision tuning
   - Fast, cheap, minimal API costs per document

3. **Graph Analysis** (POST-EXTRACTION, NO LLM):
   - Converts entities/relationships to property graph
   - Runs community detection algorithms
   - Supports graph queries and pathfinding
   - Exports to standard graph formats

## Installation

```bash
cd /path/to/doculyzer-go-conversion/go
go build -o bin/ontology ./cmd/ontology
```

## Usage

### Basic Usage

```bash
# Generate ontology schema with automatic domain detection
./bin/ontology \
  --parquet /path/to/udml/storage \
  --name financial_reports \
  --api-key YOUR_ANTHROPIC_API_KEY

# Or use environment variable for API key
export ANTHROPIC_API_KEY=your_key_here
./bin/ontology \
  --parquet /path/to/udml/storage \
  --name financial_reports
```

### Advanced Options

```bash
./bin/ontology \
  --parquet /path/to/udml/storage \
  --name sec_filings \
  --domain financial \
  --sample-size 200 \
  --top-entities 100 \
  --output schemas/sec_filings.ontology.json \
  --llm-model claude-3-5-sonnet-20241022
```

### Non-Interactive Mode

```bash
# Skip interactive refinement, just generate and save
./bin/ontology \
  --parquet /path/to/udml/storage \
  --name my_schema \
  --non-interactive
```

## Command-Line Flags

| Flag | Required | Default | Description |
|------|----------|---------|-------------|
| `--parquet` | Yes | - | Path to UDML Parquet storage directory |
| `--name` | Yes | - | Schema name (used for output filename) |
| `--domain` | No | Auto-detect | Domain name (e.g., "financial", "technical", "medical") |
| `--output` | No | `<name>.ontology.json` | Output path for schema JSON |
| `--sample-size` | No | 100 | Number of elements to sample for analysis |
| `--top-entities` | No | 50 | Number of top entities to analyze for aliases |
| `--llm-provider` | No | anthropic | LLM provider (currently only "anthropic" supported) |
| `--llm-model` | No | claude-3-5-sonnet-20241022 | Claude model to use |
| `--api-key` | No | `$ANTHROPIC_API_KEY` | Anthropic API key |
| `--non-interactive` | No | false | Skip interactive refinement |

## Environment Variables

- `ANTHROPIC_API_KEY` - Anthropic API key (alternative to `--api-key` flag)
- `CLAUDE_API_KEY` - Also accepted as fallback

## Interactive Refinement

After automatic generation, the CLI offers interactive refinement options:

### Main Menu
1. **Edit entity type** - Modify description and extraction rules
2. **Add entity type** - Add new entity type with custom rules
3. **Remove entity type** - Remove entity type
4. **Edit relationship rule** - Modify relationship descriptions
5. **Add relationship rule** - Add new relationship between entities
6. **Remove relationship rule** - Remove relationship
7. **Edit domain/metadata** - Update domain and key concepts
8. **Review top entities** - View most frequent entities in corpus
9. **Done** - Save and exit

### Entity Type Refinement
- Update descriptions
- Add/remove extraction rules:
  - **Keyword rules**: Match exact words (e.g., "Microsoft", "MSFT", "MS")
  - **Regex rules**: Pattern matching (e.g., `\b[A-Z]{2,5}\b` for stock tickers)
  - **Metadata rules**: Extract from structured metadata paths

### Relationship Rule Refinement
- Define relationships between entity types:
  - `is_a` - Taxonomic relationship
  - `part_of` - Compositional relationship
  - `related_to` - General association
  - `located_in` - Spatial relationship
  - `mentions` - Reference relationship

## Output Schema Structure

The generated schema is a JSON file with this structure:

```json
{
  "name": "financial_reports",
  "version": "1.0.0",
  "description": "Automatically generated ontology for financial domain",
  "domain": "financial",
  "key_concepts": ["revenue", "earnings", "profit", "market"],
  "element_entity_mappings": [
    {
      "entity_type": "organization",
      "description": "Companies and organizations",
      "element_types": ["paragraph", "heading"],
      "extraction_rules": [
        {
          "type": "keyword_match",
          "keywords": ["Microsoft", "MSFT", "MS", "Microsoft Corporation"],
          "confidence": 0.9
        }
      ]
    }
  ],
  "entity_relationship_rules": [
    {
      "name": "company_subsidiary",
      "source_entity_type": "organization",
      "target_entity_type": "organization",
      "relationship_type": "part_of",
      "description": "Parent-subsidiary relationship",
      "confidence_threshold": 0.7
    }
  ],
  "created_at": "2025-10-12T15:30:00Z"
}
```

## Example Workflow

### 1. Generate Schema

```bash
./bin/ontology \
  --parquet /data/sec_filings/udml_storage \
  --name sec_10k_filings \
  --sample-size 150
```

**Output:**
```
========================================
UDML Ontology Builder
========================================
Parquet path: /data/sec_filings/udml_storage
Schema name: sec_10k_filings
Sample size: 150
Top entities: 50
LLM provider: anthropic
========================================

Starting ontology build process...
Phase 1: Sampling UDML corpus...
Sampled 150 elements from 12500 total
Phase 2: Analyzing entity frequencies...
Found 342 unique entities, top 50 selected
Phase 3: Generating draft ontology schema with LLM...
Generated draft schema with 8 entity types, 5 relationship types

========================================
BUILD COMPLETE
========================================
Build time: 45.2s
LLM calls: 3
Total tokens: 8243

DRAFT ONTOLOGY SCHEMA
========================================
Domain: financial
Key concepts: revenue, earnings, assets, liabilities, equity
Entity types: 8
Relationship types: 5

Entity Types:
  1. organization - Companies and financial institutions
     Element types: paragraph
     Extraction rules: 3
       - keyword_match: Microsoft, MSFT, MS, ... (confidence: 0.90)
       - keyword_match: Apple, AAPL, Apple Inc, ... (confidence: 0.90)
       ... (1 more rules)

  2. financial_metric - Financial metrics and KPIs
     Element types: paragraph, table_cell
     Extraction rules: 2
       ...

Relationship Types:
  1. company_reports_metric: organization -> financial_metric (mentions)
     Company reports a financial metric
  ...

Would you like to review and refine the schema interactively? (y/n):
```

### 2. Interactive Refinement (Optional)

```
y

Refinement options:
  1. Edit entity type
  2. Add entity type
  ...
  9. Done (save and exit)

Choice: 1

Entity types:
  1. organization - Companies and financial institutions
  2. financial_metric - Financial metrics and KPIs
  ...

Select entity type number: 1

Editing: organization
Current description: Companies and financial institutions
New description (or press Enter to keep): [Enter]

Would you like to edit extraction rules? (y/n): y

Extraction rules for organization:
  1. keyword_match (keywords: Microsoft, MSFT, MS, ...)
  2. keyword_match (keywords: Apple, AAPL, Apple Inc, ...)
  3. regex_pattern (pattern: \b[A-Z][a-z]+ (Inc|Corp|LLC)\b)

Options:
  1. Add keyword rule
  2. Add regex rule
  3. Remove rule
  4. Done

Choice: 1

Enter keywords (comma-separated): Amazon, AMZN, Amazon.com
Enter confidence (0.0-1.0, default 0.8): 0.95
Keyword rule added.

...

Choice: 9
Done
```

### 3. Use Generated Schema

```go
package main

import (
    "context"
    "encoding/json"
    "os"

    "github.com/kennethstott/go-doc-go/internal/udml/ontology"
)

func main() {
    // Load schema
    schemaData, _ := os.ReadFile("sec_10k_filings.ontology.json")
    var schema ontology.OntologySchema
    json.Unmarshal(schemaData, &schema)

    // Create extractor
    extractor := ontology.NewRuleBasedExtractor(&schema)

    // Extract entities from documents
    ctx := context.Background()
    result, err := extractor.ExtractFromElements(ctx, docID, elements)

    // result.Entities contains all extracted entities
    // result.Relationships contains all extracted relationships
}
```

## Automatic Alias Detection

The ontology builder automatically detects aliases for frequent entities. For example:

**Input:** Corpus with frequent mentions of "Microsoft"

**LLM Analysis:** Generates aliases automatically
- Microsoft
- MSFT (stock ticker)
- MS (abbreviation)
- Microsoft Corporation (legal name)

**Output:** All aliases encoded in keyword_match rule:
```json
{
  "type": "keyword_match",
  "keywords": ["Microsoft", "MSFT", "MS", "Microsoft Corporation"],
  "confidence": 0.9
}
```

**Result:** During extraction, all variations are found and deduplicated to single entity.

## Supported Extraction Rule Types

### 1. Keyword Match
Exact word matching with case-insensitive, word-boundary aware matching.

```json
{
  "type": "keyword_match",
  "keywords": ["revenue", "sales", "income"],
  "confidence": 0.85
}
```

### 2. Regex Pattern
Regular expression matching for complex patterns.

```json
{
  "type": "regex_pattern",
  "pattern": "\\b[A-Z]{2,5}\\b",
  "confidence": 0.8
}
```

### 3. Metadata Field
Extract from structured metadata using path notation.

```json
{
  "type": "metadata_field",
  "field_path": "author.name",
  "confidence": 1.0
}
```

### 4. Text Similarity
Jaccard similarity matching against reference text.

```json
{
  "type": "text_similarity",
  "reference_text": "quarterly earnings report",
  "similarity_threshold": 0.7,
  "confidence": 0.75
}
```

### 5. JSONPath Query
Extract values from nested JSON structures.

```json
{
  "type": "jsonpath_query",
  "jsonpath_expr": "$.company.subsidiaries[*].name",
  "confidence": 0.9
}
```

## Performance

**Ontology Building (One-Time):**
- Sample size 100: ~30-60 seconds
- Sample size 200: ~60-120 seconds
- LLM calls: 3 (domains, entities, relationships)
- Token usage: ~5,000-10,000 tokens

**Entity Extraction (Runtime):**
- 1,000 elements: ~50-100ms
- 10,000 elements: ~500ms-1s
- NO LLM calls, NO API costs

## Troubleshooting

### "API key required" error
```bash
# Set environment variable
export ANTHROPIC_API_KEY=your_key_here

# Or use flag
./bin/ontology --api-key your_key_here ...
```

### "Parquet path not found" error
Ensure the path points to a valid UDML Parquet storage directory with Hive partitioning:
```
/path/to/storage/
  element_type=paragraph/
    version=1/
      date=20250101/
        source=docs/
          part-0.parquet
```

### Schema has too few/many entity types
Adjust sampling parameters:
```bash
# More samples for better coverage
--sample-size 200

# Analyze more entities for better alias detection
--top-entities 100
```

## Knowledge Graph Analysis

### Overview

The `analyze-graph` command converts extracted ontologies into property graphs and provides advanced analysis capabilities.

### Basic Usage

```bash
# Analyze extracted ontology and export graph
ontology analyze-graph \
  --parquet /data/sec_filings/udml_storage \
  --run-id extraction_12345 \
  --output-gob ./knowledge_graph.gob \
  --output-json ./graph_metadata.json
```

### Command-Line Flags

| Flag | Required | Default | Description |
|------|----------|---------|-------------|
| `--parquet` | Yes | - | Path to UDML Parquet storage directory |
| `--run-id` | Yes | - | Run ID from extraction (found in job database) |
| `--algorithm` | No | louvain | Community detection algorithm: `louvain` or `label_propagation` |
| `--enrich-communities` | No | true | Add explicit community nodes to graph |
| `--output-gob` | No | - | Output path for graph in GOB format |
| `--output-json` | No | - | Output path for graph metadata in JSON |
| `--stats-only` | No | false | Only print statistics, don't write output files |

### Community Detection Algorithms

**Louvain Algorithm:**
- Hierarchical community detection
- Optimizes modularity (Newman-Girvan metric)
- Fast for large graphs (O(n log n))
- Generally produces better-defined communities
- Recommended for most use cases

**Label Propagation:**
- Simple, fast algorithm (O(m + n))
- Nodes adopt most frequent label among neighbors
- Non-deterministic (may produce different results)
- Good for very large graphs
- Useful for quick exploratory analysis

### Example Workflow

**1. Extract Entities:**
```bash
ontology extract \
  --schema medical_ontology.json \
  --parquet /data/wikipedia/udml_storage \
  --job-db extraction.db \
  --doc-batch-size 50
```

**2. Find Run ID:**
```bash
sqlite3 extraction.db "SELECT run_id FROM runs ORDER BY created_at DESC LIMIT 1"
# Output: extraction_2025_01_22_abc123
```

**3. Analyze Graph:**
```bash
ontology analyze-graph \
  --parquet /data/wikipedia/udml_storage \
  --run-id extraction_2025_01_22_abc123 \
  --algorithm louvain \
  --enrich-communities \
  --output-gob medical_knowledge_graph.gob \
  --output-json medical_graph_metadata.json
```

**Output:**
```
========================================
KNOWLEDGE GRAPH ANALYSIS
========================================
  Parquet: /data/wikipedia/udml_storage
  Run ID: extraction_2025_01_22_abc123
  Algorithm: louvain
  Enrich communities: true
========================================

📊 Initializing Parquet storage...
📖 Loading ontology from Parquet...
  ✓ Loaded ontology:
    - Entities: 1,247
    - Relationships: 3,891

🔄 Converting ontology to knowledge graph...
  ✓ Knowledge graph created:
    - Nodes: 1,247
    - Edges: 3,891

🔍 Preparing graph for analysis...
🧩 Running community detection (louvain)...
  ✓ Community detection complete:
    - Communities found: 23
    - Modularity: 0.7234

  Top communities by size:
    1. Community 0: 312 nodes
    2. Community 1: 198 nodes
    3. Community 2: 147 nodes
    4. Community 3: 89 nodes
    5. Community 4: 76 nodes

🌟 Enriching graph with community nodes...
  ✓ Graph enrichment complete:
    - Original nodes: 1,247
    - Enriched nodes: 1,270
    - Community nodes added: 23
    - BELONGS_TO edges added: 1,247
    - Avg community size: 54.2 nodes

🔎 Analyzing graph structure...
  Entity nodes: 1,247
  Community nodes: 23

  Nodes by domain:
    - medical: 1,247
    - community: 23

  Edges by type:
    - RELATED_TO: 2,341
    - PART_OF: 892
    - MENTIONED_BY: 658
    - BELONGS_TO: 1,247

💾 Saving graph to GOB format: medical_knowledge_graph.gob
  ✓ Graph saved successfully
💾 Saving graph metadata to JSON: medical_graph_metadata.json
  ✓ Metadata saved successfully

========================================
GRAPH ANALYSIS COMPLETE
========================================
  Graph (GOB): medical_knowledge_graph.gob
  Metadata (JSON): medical_graph_metadata.json
========================================
```

### Graph Query Engine

The generated graph can be queried using the native Go query API:

```go
package main

import (
    "github.com/kennethstott/go-doc-go/internal/graph"
)

func main() {
    // Load graph from GOB file
    kg, err := graph.LoadGraph("medical_knowledge_graph.gob")
    if err != nil {
        panic(err)
    }

    // Create query engine
    qe := graph.NewGraphQueryEngine(kg)

    // Find all person entities
    people := qe.FindNodesByLabel("Entity:person", graph.DefaultQueryOptions())
    fmt.Printf("Found %d people\n", len(people))

    // Find high-confidence entities
    highConfidence := qe.FindNodesByConfidence(0.9, graph.DefaultQueryOptions())

    // Find shortest path between two entities
    path := qe.FindShortestPath("person_123", "org_456", 10)
    if path != nil {
        fmt.Printf("Path length: %d hops\n", len(path.Nodes)-1)
    }

    // Get neighbors of an entity
    neighbors := qe.GetNeighbors("person_123", graph.BothNeighbors, graph.DefaultQueryOptions())

    // Extract subgraph of medical domain
    medicalSubgraph := qe.GetSubgraph(graph.DomainFilter("medical"))
}
```

### RDF Export

Export graphs to RDF for SPARQL querying:

```go
import (
    "os"
    "github.com/kennethstott/go-doc-go/internal/graph/export"
)

func main() {
    kg, _ := graph.LoadGraph("knowledge_graph.gob")

    // Export as N-Triples
    exporter := export.NewRDFExporter(
        export.WithFormat(export.FormatNTriples),
        export.WithBaseURI("http://example.org/kg/"),
    )

    f, _ := os.Create("knowledge_graph.nt")
    defer f.Close()
    exporter.ExportGraph(kg, f)

    // Export as Turtle (more compact)
    turtleExporter := export.NewRDFExporter(
        export.WithFormat(export.FormatTurtle),
        export.WithBaseURI("http://example.org/kg/"),
    )

    f2, _ := os.Create("knowledge_graph.ttl")
    defer f2.Close()
    turtleExporter.ExportGraph(kg, f2)
}
```

**Supported RDF Formats:**
- **N-Triples** (`.nt`): Line-based, easy to parse, large file size
- **Turtle** (`.ttl`): Compact, human-readable, supports prefixes
- **RDF/XML** (`.rdf`): XML-based, widely supported, verbose

### MCP Server Integration

The graph package includes MCP (Model Context Protocol) server integration for LLM-driven exploration:

```go
import (
    "github.com/kennethstott/go-doc-go/internal/graph/mcp"
    "github.com/kennethstott/go-doc-go/internal/analytics"
)

func main() {
    // Initialize storage
    storage, _ := analytics.NewHiveParquetStorage(map[string]interface{}{
        "path":    "/data/udml_storage",
        "version": "v2.0.0",
    })

    // Create MCP server for graph exploration
    explorer := mcp.NewGraphQueryExplorer(storage)
    mcpServer := explorer.CreateMCPServer()

    // Start MCP server (stdio transport)
    mcpServer.Start()
}
```

**Available MCP Tools:**
1. `query_graph_nodes` - Query nodes by label, property, domain, confidence
2. `query_graph_edges` - Query edges by type, source, target
3. `find_shortest_path` - BFS pathfinding between nodes
4. `get_neighbors` - Retrieve connected nodes (incoming/outgoing/both)
5. `get_subgraph` - Extract filtered subgraphs
6. `analyze_communities` - Run community detection algorithms
7. `graph_statistics` - Compute graph-wide statistics

### Graph Metadata JSON Structure

The `--output-json` file contains graph metadata:

```json
{
  "id": "graph_analysis_extraction_2025_01_22_abc123",
  "name": "Knowledge Graph from extraction_2025_01_22_abc123",
  "version": "1.0.0",
  "node_count": 1270,
  "edge_count": 5138,
  "domains": {
    "medical": 1247,
    "community": 23
  },
  "edge_types": {
    "RELATED_TO": 2341,
    "PART_OF": 892,
    "MENTIONED_BY": 658,
    "BELONGS_TO": 1247
  },
  "run_id": "extraction_2025_01_22_abc123",
  "community_count": 23,
  "modularity": 0.7234,
  "algorithm": "louvain"
}
```

### Performance

**Graph Conversion:**
- 1,000 entities: ~50-100ms
- 10,000 entities: ~500ms-1s
- Conversion is linear O(n + m) where n=entities, m=relationships

**Community Detection:**
- Louvain (1,000 nodes): ~100-200ms
- Louvain (10,000 nodes): ~1-2s
- Label Propagation (10,000 nodes): ~500ms-1s

**Graph Export:**
- GOB serialization (10,000 nodes): ~100-200ms
- RDF N-Triples (10,000 nodes): ~300-500ms
- RDF Turtle (10,000 nodes): ~400-600ms

## Best Practices

1. **Start with automatic generation** - Let the LLM create initial draft
2. **Use interactive refinement** - Add domain-specific entities and rules
3. **Test on sample documents** - Validate extraction before full corpus processing
4. **Iterate on rules** - Refine based on extraction results
5. **Version control schemas** - Track schema evolution over time
6. **Analyze extraction results** - Use analyze-graph to understand entity relationships
7. **Choose appropriate algorithms** - Louvain for quality, Label Propagation for speed
8. **Export for downstream analysis** - Use RDF export for SPARQL querying

## Related Documentation

- [UDML Architecture](../../docs/UDML_ARCHITECTURE.md)
- [Rule-Based Extractor](../../internal/udml/ontology/README.md)
- [UDML Sampler](../../internal/udml/sampler/README.md)
- [Knowledge Graph Package](../../internal/graph/README.md)
- [MCP Server Integration](../../internal/graph/mcp/README.md)
