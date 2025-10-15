# UDML Ontology Builder CLI

Interactive command-line tool for automatically generating ontology schemas from UDML document corpora using LLM-powered analysis.

## Overview

The Ontology Builder analyzes your UDML Parquet storage to automatically discover:
- Domain(s) within your document corpus
- Entity types (organizations, people, products, concepts, etc.)
- Relationship types between entities
- Extraction rules (keywords, regex patterns, metadata paths)

The output is an OntologySchema JSON file that can be used with the rule-based extractor for fast, deterministic entity extraction.

## Architecture

**Two-Phase Design:**
1. **Ontology Building** (ONE-TIME, uses LLM):
   - Samples corpus using stratified sampling
   - Analyzes entity frequencies
   - Uses Claude API to identify domains and generate extraction rules
   - Automatically detects aliases for frequent entities
   - Outputs: OntologySchema JSON

2. **Entity Extraction** (RUNTIME, NO LLM):
   - Uses the generated OntologySchema
   - Applies rules deterministically
   - Fast, cheap, no API costs per document

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

## Best Practices

1. **Start with automatic generation** - Let the LLM create initial draft
2. **Use interactive refinement** - Add domain-specific entities and rules
3. **Test on sample documents** - Validate extraction before full corpus processing
4. **Iterate on rules** - Refine based on extraction results
5. **Version control schemas** - Track schema evolution over time

## Related Documentation

- [UDML Architecture](../../docs/UDML_ARCHITECTURE.md)
- [Rule-Based Extractor](../../internal/udml/ontology/README.md)
- [UDML Sampler](../../internal/udml/sampler/README.md)
