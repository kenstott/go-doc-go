# UDML Ontology System Documentation

## Overview

The UDML Ontology System provides automatic ontology schema generation for document corpora using LLM-guided analysis, domain catalog templates, and cosine similarity-based domain detection.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Ontology Builder                          │
│  (LLM-guided schema generation + refinement)                │
└───────────────┬─────────────────────────────────────────────┘
                │
                ├─→ Domain Similarity Engine
                │   (Cosine similarity for domain suggestion)
                │
                ├─→ Domain Catalog System (YAML-based)
                │   - 9 built-in domains
                │   - 30 common entity templates
                │   - User-extensible via YAML
                │
                ├─→ Corpus Sampler
                │   (Stratified sampling from UDML Parquet)
                │
                └─→ MCP Server (in progress)
                    (Interactive corpus exploration tools)
```

## Features Completed

### 1. Domain Catalog System

**Purpose**: Pre-built ontology templates for different industries/domains, now fully configurable via YAML.

**Built-in Domains** (9 total):
1. **Financial** - Financial reports, earnings, transactions
2. **Legal** - Contracts, court documents, regulations
3. **Medical** - Clinical notes, prescriptions, lab reports
4. **Technical** - Software docs, API specs, technical manuals
5. **Insurance** - Policies, claims, underwriting (8 subdomains)
6. **Manufacturing** - Specs, production reports, QC docs
7. **Retail** - Sales reports, inventory, pricing
8. **Logistics** - Shipping docs, bills of lading, freight
9. **Education** - Curricula, student records, programs

**Common Entity Templates** (30 shared entities):
- **Location**: city, street, address, building, postal_code, country, region
- **Person**: person, public_figure, role, executive, employee
- **Descriptive**: color, size, dimension, weight, volume, material, texture, shape
- **Temporal**: date, time, duration
- **Contact**: email, phone, url
- **Numeric**: percentage, number
- **Identifier**: id_number, code

**YAML Configuration**:
- All catalogs exported to `examples/ontologies/*.yaml`
- Users can create custom domains without code changes
- Auto-loading from YAML directory
- Export built-in catalogs for customization

**Loading Catalogs**:
```go
// Auto-loaded from ./examples/ontologies/ on import
import "github.com/kennethstott/doculyzer-go-conversion/internal/udml/ontology/catalogs"

// Or manually load custom catalogs
catalogs.RegisterFromDirectory("./my-custom-catalogs")

// Access catalogs
catalog, exists := catalogs.GetCatalog("financial")
allDomains := catalogs.ListDomains()
```

### 2. Domain Similarity Analysis

**Purpose**: Automatically suggest relevant domains from corpus before LLM refinement using vector similarity.

**How It Works**:
1. **Corpus Embedding**: Generate average embedding from sample texts
2. **Domain Embeddings**: Generate embeddings for each domain (description + terms + entities)
3. **Cosine Similarity**: Compute similarity scores between corpus and all domains
4. **Ranking**: Sort domains by similarity score
5. **LLM Validation**: LLM validates top N suggestions

**API**:
```go
// In OntologyBuilder
builder := ontology.NewOntologyBuilder(config)

// Compute domain similarities
catalogProvider := catalogs.NewCatalogProvider()
result, err := builder.ComputeDomainSimilarities(
    ctx,
    samples,              // Sampling result
    catalogProvider,      // Domain catalog provider
    embGen,               // Embedding generator
    5,                    // Top N domains
)

// Access results
for _, domainSim := range result.TopDomains {
    fmt.Printf("%s: %.3f\n", domainSim.Domain, domainSim.Similarity)
}
```

**Benefits**:
- **Performance**: Pre-filters domains before expensive LLM call
- **Accuracy**: Embeddings capture semantic similarity
- **Extensibility**: Works with custom domain catalogs
- **Multi-domain**: Detects multiple relevant domains

### 3. MCP Server Structure (In Progress)

**Purpose**: Model Context Protocol server providing LLM tools for interactive corpus exploration during ontology refinement.

**6 MCP Tools Planned**:

1. **search_corpus**
   - Semantic/keyword/regex search across corpus
   - Filter by element types
   - Configurable similarity threshold
   - Returns matching elements with context

2. **analyze_patterns**
   - Analyze regex pattern matches
   - Element type distribution
   - Match frequency
   - Example matches with context

3. **compute_frequencies**
   - Term/entity frequency counts
   - Document distribution
   - Element type breakdown
   - Case-sensitive option

4. **find_cooccurrences**
   - Entity co-occurrence analysis
   - Configurable context window (element/paragraph/document)
   - Example contexts
   - Relationship discovery

5. **get_element_context**
   - Retrieve element hierarchy
   - Parent/sibling/child elements
   - Configurable depth
   - Document structure understanding

6. **aggregate_statistics**
   - Element type distribution
   - Document counts
   - Average content lengths
   - Entity type distribution

**Usage**:
```bash
# Start MCP server
ontology_mcp ./output/udml.parquet ./assets/all-MiniLM-L6-v2-onnx

# LLM client connects via stdio and can call tools
# Example tool call:
{
  "tool": "search_corpus",
  "query": "insurance policy",
  "search_type": "semantic",
  "limit": 10
}
```

**Status**:
- Server structure implemented
- Tool definitions complete
- Implementation needs query refactoring (query package uses backend interface pattern)

## Configuration Files

### Domain Catalog YAML Format

```yaml
domain: education
description: Educational institutions, curricula, student records

subdomains:
  - k12_education
  - higher_education

terms:
  - name: curriculum
    synonyms:
      - syllabus
      - course of study
    description: Planned educational program

entity_types:
  - entity_type: student
    description: Student or learner
    aliases:
      - learner
      - pupil
    element_types:
      - paragraph
      - table_cell
    sample_rules:
      - type: keyword_match
        keywords:
          - student
          - pupil
      - type: regex_pattern
        pattern: '\b[A-Z]{2,4}\s*\d{3,4}\b'

common_entity_refs:
  - person
  - date
  - address

relationships:
  - name: student_enrolled_in_course
    source_type: student
    target_type: course
    relationship_type: related_to
    description: Student enrolled in course
    sample_patterns:
      - "{student} enrolled in {course}"
```

### Extraction Rule Types

1. **keyword_match** - Match by keywords
2. **regex_pattern** - Match by regex
3. **text_similarity** - Match by semantic similarity
4. **metadata_field** - Extract from metadata
5. **jsonpath_query** - Extract using JSONPath

### Relationship Types

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

## Builder Configuration

```go
config := ontology.BuilderConfig{
    // Corpus sampling
    ParquetPath:   "./output/udml.parquet",
    SampleSize:    1000,

    // LLM configuration
    LLMProvider:   "anthropic",
    LLMModel:      "claude-sonnet-4-5-20250929",
    LLMAPIKey:     os.Getenv("ANTHROPIC_API_KEY"),
    LLMMaxTokens:  4096,

    // Analysis parameters
    TopEntityCount:      50,
    MinEntityFrequency:  5,
    ConfidenceThreshold: 0.7,

    // Output
    SchemaName:    "MyCorpusOntology",
    SchemaVersion: "1.0.0",
}

builder, err := ontology.NewOntologyBuilder(config)
if err != nil {
    log.Fatal(err)
}
defer builder.Close()

// Run ontology generation
result, err := builder.Build(context.Background())
if err != nil {
    log.Fatal(err)
}

// Access generated schema
schema := result.Schema
fmt.Printf("Generated %d entity mappings\n", len(schema.ElementEntityMappings))
fmt.Printf("Generated %d relationship rules\n", len(schema.EntityRelationshipRules))
```

## Ontology Schema Output

The generated `OntologySchema` contains:

```go
type OntologySchema struct {
    Name        string
    Version     string
    Description string

    // Discovered domains
    Domains []Domain

    // Key concepts
    KeyConcepts []string

    // Entity extraction mappings
    ElementEntityMappings []ElementEntityMapping

    // Relationship extraction rules
    EntityRelationshipRules []EntityRelationshipRule

    CreatedAt time.Time
}
```

**Domain Structure**:
```go
type Domain struct {
    Name        string   // e.g., "financial"
    Description string   // What this domain covers
    Owner       string   // Organizational owner
    KeyConcepts []string // Domain-specific concepts
}
```

**Entity Mapping Structure**:
```go
type ElementEntityMapping struct {
    EntityType      string           // Entity type name
    Domain          string           // Assigned domain
    Description     string           // What this entity represents
    ElementTypes    []string         // UDML element types
    Confidence      float64          // Extraction confidence (0.0-1.0)
    ExtractionRules []ExtractionRule // How to extract
}
```

**Relationship Rule Structure**:
```go
type EntityRelationshipRule struct {
    Name                string            // Rule name
    SourceEntityType    string            // Source entity type
    TargetEntityType    string            // Target entity type
    RelationshipType    RelationshipType  // Relationship category
    Description         string            // What this relationship means
    Confidence          float64           // Pattern reliability
    ExtractionPatterns  []ExtractionPattern // Detection patterns
}
```

## Usage Examples

### Example 1: Generate Ontology for Financial Corpus

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
        ParquetPath:   "./financial_reports.parquet",
        SampleSize:    1000,
        LLMProvider:   "anthropic",
        LLMModel:      "claude-sonnet-4-5-20250929",
        LLMAPIKey:     os.Getenv("ANTHROPIC_API_KEY"),
        SchemaName:    "FinancialReportsOntology",
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

    log.Printf("Sampled: %d elements", result.Samples.SampledCount)
    log.Printf("Domains: %v", result.Schema.Domains)
    log.Printf("Entities: %d", len(result.Schema.ElementEntityMappings))
    log.Printf("Relationships: %d", len(result.Schema.EntityRelationshipRules))
    log.Printf("LLM calls: %d, Tokens: %d", result.LLMCallCount, result.TotalLLMTokens)
}
```

### Example 2: Custom Domain Catalog

```go
package main

import (
    "log"

    "github.com/kennethstott/doculyzer-go-conversion/internal/udml/ontology/catalogs"
)

func main() {
    // Load custom domain catalogs
    err := catalogs.RegisterFromDirectory("./my-company-catalogs")
    if err != nil {
        log.Fatal(err)
    }

    // Now builder will use these catalogs for domain suggestion
    // ...
}
```

### Example 3: Export Built-in Catalog for Customization

```go
package main

import (
    "log"

    "github.com/kennethstott/doculyzer-go-conversion/internal/udml/ontology/catalogs"
)

func main() {
    // Get built-in catalog
    catalog, exists := catalogs.GetCatalog("financial")
    if !exists {
        log.Fatal("Catalog not found")
    }

    // Export to YAML for customization
    err := catalogs.SaveToFile(catalog, "./my-financial-catalog.yaml")
    if err != nil {
        log.Fatal(err)
    }

    // Edit the YAML file, then reload
    err = catalogs.RegisterFromDirectory(".")
    if err != nil {
        log.Fatal(err)
    }
}
```

## System Integration

### With UDML Parser

```
Document → Parser → UDML Elements → Parquet → Ontology Builder → Schema
```

1. Parse documents into UDML elements
2. Store in Parquet with embeddings
3. Sample corpus with stratification
4. Generate ontology schema
5. Apply schema for entity extraction

### With Query System

```
Ontology Schema → Entity Extraction → Relationships → Knowledge Graph
```

1. Load ontology schema
2. Apply extraction rules to corpus
3. Extract entities and relationships
4. Build queryable knowledge graph

## Command-Line Tools

### Export Catalogs

```bash
# Export all built-in catalogs to YAML
go run ./cmd/export_catalogs ./output/catalogs

# Output:
#   ✓ Exported financial → ./output/catalogs/financial.yaml
#   ✓ Exported legal → ./output/catalogs/legal.yaml
#   ...
```

### Test Catalog Loading

```bash
# Verify catalogs load correctly
go run ./cmd/test_catalog_load

# Output:
#   Loaded 9 domain catalogs:
#   ✓ financial
#     Description: Financial reports, earnings...
#     Entity types: 5 domain-specific + 14 common refs
#     Relationships: 4
#     Terms: 8
```

### Ontology Generation CLI

```bash
# Generate ontology for a corpus
ontology --parquet ./corpus.parquet \
         --sample-size 1000 \
         --output ontology.json \
         --provider anthropic \
         --model claude-sonnet-4-5-20250929
```

### MCP Server (when complete)

```bash
# Start MCP server for corpus exploration
ontology_mcp ./corpus.parquet ./embedding-model
```

## Migration Path

### From Hardcoded to YAML Catalogs

**Before** (hardcoded in Go):
```go
var FinancialCatalog = &DomainCatalog{
    Domain: "financial",
    // ... hundreds of lines of Go code
}
```

**After** (YAML configuration):
```yaml
# examples/ontologies/financial.yaml
domain: financial
description: Financial reports and analysis
entity_types:
  - entity_type: company
    description: Business organization
    # ... clean YAML definition
```

**Benefits**:
- No recompilation for catalog changes
- Version control friendly
- User-extensible without code changes
- Easier to maintain and review
- Multilingual support (YAML comments in any language)

## Performance Considerations

### Domain Similarity
- **Pre-filters** domains before LLM call
- **O(N×D)** complexity where N=domains, D=embedding_dim
- **Caching**: Consider caching domain embeddings

### Corpus Sampling
- **Stratified sampling** ensures representative samples
- **Element type distribution** maintained
- **Configurable sample size** (default: 1000)

### LLM Optimization
- **Batching**: Group similar prompts
- **Caching**: Reuse entity/relationship definitions
- **Token limits**: Truncate long samples

### 4. Interactive Interview Builder ✅

**Purpose**: LLM-guided conversational interface for ontology generation with iterative refinement.

**4-Phase Interview Process**:

1. **Phase 1: Domain Selection**
   - LLM analyzes corpus samples
   - Suggests relevant built-in domains
   - Asks clarifying questions about use case
   - User provides context
   - Finalizes domain selection with organizational owners

2. **Phase 2: Entity Discovery**
   - Shows most frequent entities found in corpus
   - LLM proposes entity types with extraction patterns
   - Asks questions about specific entities user cares about
   - User provides feedback
   - Finalizes entity definitions with complete extraction rules

3. **Phase 3: Relationship Discovery**
   - LLM analyzes entity co-occurrences
   - Proposes relationships with sample patterns
   - Asks about important connections to capture
   - User provides guidance
   - Finalizes relationship patterns

4. **Phase 4: Refinement**
   - Shows complete schema summary
   - Allows user to request changes
   - LLM applies modifications
   - Iterative refinement until user satisfied

**Usage**:
```bash
# Start interactive interview
export ANTHROPIC_API_KEY=your_key_here
ontology_interview ./corpus.parquet ./output-schema.json

# The tool will:
# 1. Sample your corpus
# 2. Guide you through 4 phases
# 3. Ask clarifying questions
# 4. Generate high-quality schema
# 5. Save to output file
```

**Benefits**:
- **Higher Quality**: LLM asks targeted questions to understand your specific needs
- **User Control**: You guide the discovery process with your domain knowledge
- **Iterative**: Refine until you're satisfied with the schema
- **Educational**: Learn about your corpus through the analysis process
- **Time Efficient**: 5-10 minute interview vs hours of manual schema design

**Example Interaction**:
```
╔═══════════════════════════════════════════════════════════╗
║   Interactive Ontology Builder - Guided Interview        ║
╚═══════════════════════════════════════════════════════════╝

→ Sampling document corpus...
  ✓ Sampled 1000 elements from 50000 total documents
  ✓ Found 347 unique entities

╔═══════════════════════════════════════════════════════════╗
║  Phase 1: Domain Selection                               ║
╚═══════════════════════════════════════════════════════════╝

LLM Analysis:
─────────────
  • financial (confidence: 0.92)
    I see references to revenue, EBITDA, quarterly earnings in samples
  • legal (confidence: 0.78)
    Multiple mentions of contracts, compliance, regulatory terms

LLM: What is the primary business purpose of these documents?
You: These are annual reports for publicly traded companies

LLM: Are these internal reports or external filings?
You: External SEC filings (10-K, 10-Q)

Selected Domains:
  ✓ financial - Financial performance and metrics (Owner: CFO Office)
  ✓ legal - Regulatory compliance and disclosures (Owner: Legal)

╔═══════════════════════════════════════════════════════════╗
║  Phase 2: Entity Discovery                               ║
╚═══════════════════════════════════════════════════════════╝

Most frequent entities found:
  1. revenue (342 occurrences)
  2. net income (298 occurrences)
  ...

Proposed Entity Types:
──────────────────────
  • company (financial domain)
    Business organization or corporation
  • financial_metric (financial domain)
    Performance metrics like revenue, profit
  ...

LLM: Should we distinguish between subsidiaries and parent companies?
You: Yes, we need to track corporate hierarchy

✓ Defined 15 entity types

...
```

## Future Enhancements

1. **MCP Tool Implementation**
   - Complete query implementations
   - Refactor query package for simpler API
   - Add batch operations

3. **Multi-lingual Support**
   - Entity synonyms in multiple languages
   - Cross-lingual domain matching

4. **Catalog Marketplace**
   - Community-contributed catalogs
   - Industry-specific templates
   - Versioning and dependencies

5. **Schema Evolution**
   - Incremental schema updates
   - Backward compatibility
   - Migration scripts

## References

- **UDML Specification**: `docs/features/udml/specification.md`
- **Catalog Examples**: `examples/ontologies/*.yaml`
- **Catalog README**: `examples/ontologies/README.md`
- **Migration Plan**: `UDML_MIGRATION_PLAN.md`

## Support

For questions, issues, or contributions:
- GitHub Issues: [Report a bug or request a feature]
- Documentation: See `docs/` directory
- Examples: See `examples/ontologies/` directory
