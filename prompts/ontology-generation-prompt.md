# Domain-Driven Ontology Generation Prompt

## Your Task

You are an expert knowledge engineer specializing in creating domain-specific ontologies for document analysis systems. Your task is to analyze existing parsed document elements and generate comprehensive ontology rules that can automatically extract entities and relationships relevant to a specific domain.

## Domain Context

**Domain**: {DOMAIN_NAME}

**Domain Description**:
{DOMAIN_DESCRIPTION}

**Key Concepts**: {KEY_CONCEPTS}

**Important Relationships**: {IMPORTANT_RELATIONSHIPS}

**Regulatory/Business Context**: {REGULATORY_CONTEXT}

## Document Analysis Data

### Document Types Present
{DOCUMENT_TYPES}

### Sample Element Records
Below are actual element records from parsed documents in this domain:

```json
{SAMPLE_ELEMENTS}
```

### Element Type Distribution
{ELEMENT_TYPE_STATS}

### Common Metadata Patterns
{METADATA_PATTERNS}

### Frequent Element Names/Paths
{FREQUENT_NAMES_AND_PATHS}

## Ontology Generation Requirements

Based on the domain context and document analysis data above, generate a comprehensive ontology YAML that includes:

### 1. Core Terms
Identify and define the most important entities in this domain by analyzing:
- **Element names** that represent domain concepts (e.g., `rptOwnerName` → person entity)
- **Content patterns** that indicate domain-specific entities
- **Structural relationships** in the parsed documents
- **Temporal elements** that mark important events/dates

For each term, provide:
- **ID**: Snake_case identifier
- **Label**: Human-readable name
- **Description**: Clear explanation of what this entity represents in the domain
- **Aliases**: Alternative names/terms found in documents

### 2. Element Mappings
Create precise mapping rules to identify each term by analyzing:
- **Element names**: XML tags, JSON keys, CSV headers, HTML classes/IDs
- **Paths**: XPath, JSONPath, CSS selectors showing where entities appear
- **Content patterns**: Regex or keyword patterns in element text
- **Contextual clues**: Parent-child relationships and surrounding elements

Use the enhanced search capabilities with `search_scope` options:
- `"element_name"`: For XML tags, JSON keys, CSV headers
- `"path"`: For hierarchical location patterns
- `"attributes"`: For XML/HTML attribute values
- `"all"`: For comprehensive text + metadata search

### 3. Relationship Rules
Define how entities connect by examining:
- **Hierarchical patterns**: Parent-child relationships in document structure
- **Co-occurrence patterns**: Entities that appear together in documents
- **Temporal relationships**: Date/time-based connections
- **Cross-document patterns**: Entities that link across multiple documents

### 4. Derived Entity Rules (if applicable)
For complex analytical entities that aggregate or combine simpler entities:
- **Aggregation patterns**: How to group related entities
- **Temporal windows**: Time-based grouping criteria
- **Classification rules**: How to categorize or score derived entities

## Output Format

Generate a complete ontology YAML file with this structure:

```yaml
domain:
  name: "{domain_name}"
  version: "1.0.0"
  description: "{domain_description}"
  settings:
    default_confidence_threshold: 0.75
    max_relationships_per_pair: 5
    enable_transitive_inference: true

terms:
  # Core domain entities

element_mappings:
  # Precise extraction rules

relationship_rules:
  # Entity connection patterns

derived_entity_rules:
  # Complex analytical entities (if needed)
```

## Analysis Guidelines

### Pattern Recognition
1. **Look for domain-specific naming conventions** in element names
2. **Identify structural patterns** that indicate entity types
3. **Recognize temporal patterns** that suggest event sequences
4. **Find cross-references** that indicate relationships

### Confidence Scoring
- Use **high confidence (0.9+)** for exact element name matches
- Use **medium confidence (0.7-0.9)** for path-based or contextual matches
- Use **lower confidence (0.6-0.8)** for semantic/content-based matches

### Coverage Optimization
- Ensure rules cover **all major entity types** found in the document samples
- Create **redundant matching rules** with different confidence levels
- Include **path-based fallbacks** for when element names change
- Consider **attribute-based matching** for XML/HTML documents

### Relationship Precision
- Focus on **high-value relationships** that provide analytical insight
- Use **constraint rules** to avoid false positive relationships
- Consider **temporal constraints** for time-sensitive domains
- Enable **cross-document relationships** for pattern analysis

## Quality Criteria

Your generated ontology should:
1. **Cover 80%+ of relevant entities** present in the sample documents
2. **Use precise matching rules** that minimize false positives
3. **Include meaningful relationships** that enable domain analysis
4. **Leverage the enhanced search system** with appropriate search scopes
5. **Follow domain expertise** and regulatory/business requirements
6. **Be extensible** to handle variations in document formats

## Example Analysis Process

1. **Scan element names** for domain concepts → Create terms
2. **Analyze element paths** for structural patterns → Create mapping rules
3. **Examine content patterns** for entity indicators → Add keyword/regex rules
4. **Map hierarchical relationships** → Create relationship rules
5. **Identify temporal patterns** → Add time-based constraints
6. **Consider analytical needs** → Create derived entity rules

Generate the ontology now based on the provided domain context and document analysis data.