# MCP-Driven Ontology Generation Prompt

## Your Task

You are an expert knowledge engineer creating domain-specific ontologies. You have access to an MCP tool that can intelligently sample document and element records from the corpus. Use this tool to gather data-driven insights and generate comprehensive ontology rules.

## Step 1: Domain Definition

First, clearly define your domain:

**Domain Name**: [e.g., "insider_trading_detection"]

**Domain Description**: [Detailed explanation of the domain, key concepts, regulations, business context]

**Key Keywords**: [Terms that would appear in relevant documents]

**Document Types**: [SEC forms, financial reports, etc.]

**Entity Hints**: [Types of entities you expect to find]

**Relationship Hints**: [Types of relationships you expect to discover]

## Step 2: Data Sampling

Use the MCP tool to sample relevant data:

```
Use the sample_for_ontology tool with your domain parameters:
- domain_name: Your domain identifier
- domain_description: Detailed description
- keywords: Comma-separated relevant terms
- document_types: Comma-separated document types
- entity_hints: Expected entity types
- relationship_hints: Expected relationship types
- max_elements: Number of elements to sample (default 200)
```

## Step 3: Analysis and Generation

After receiving the sampling data, analyze the following sections:

### Corpus Statistics
- **Total coverage**: How many documents/elements are available?
- **Format distribution**: What document types dominate?
- **Element type patterns**: What structural patterns exist?
- **Domain relevance**: How well does the corpus match your domain?

### Sampled Elements
Examine the actual element records:
- **Structural names**: What element names/tags appear frequently?
- **Path patterns**: What hierarchical structures exist?
- **Content patterns**: What actual data values are present?
- **Temporal elements**: What date/time patterns exist?

### Pattern Analysis
Review the discovered patterns:
- **Naming conventions**: How are domain entities named?
- **Structural relationships**: How are entities organized?
- **Attribute usage**: What metadata is available?
- **Cross-document patterns**: What links exist between documents?

## Step 4: Ontology Generation

Based on the sampled data, generate a comprehensive ontology with:

### Terms
Create terms based on **actual element names and content patterns** found in the data:
```yaml
terms:
  - id: "entity_name"
    label: "Human Label"
    description: "Based on actual patterns in: [element_names found]"
    aliases: [actual variations discovered]
```

### Element Mappings
Create precise mappings using **actual patterns from the sample**:
```yaml
element_mappings:
  - term_id: "entity_name"
    rules:
      # High-confidence element name matches
      - type: keywords
        search_scope: "element_name"
        keywords: [actual element names from sample]
        confidence_threshold: 0.95

      # Path-based matches for structure
      - type: keywords
        search_scope: "path"
        keywords: [actual path patterns from sample]
        confidence_threshold: 0.85

      # Content-based semantic matches
      - type: semantic
        search_scope: "all"
        semantic_phrase: "description based on actual content"
        confidence_threshold: 0.75
```

### Relationship Rules
Define relationships based on **observed document structures**:
```yaml
relationship_rules:
  - id: "relationship_name"
    relationship_type: "RELATIONSHIP_TYPE"
    source:
      term_id: "source_entity"
      semantic_phrase: "based on observed patterns"
    target:
      term_id: "target_entity"
      semantic_phrase: "based on document structure"
    constraints:
      same_document: true  # or cross-document based on analysis
```

## Quality Guidelines

### Data-Driven Approach
- **Ground all rules in actual sample data** - no theoretical mappings
- **Use exact element names** found in the corpus
- **Leverage discovered path patterns** for structural matching
- **Incorporate observed attribute patterns** for precision

### Coverage Optimization
- **Prioritize high-frequency patterns** for maximum coverage
- **Include rare but important patterns** for completeness
- **Use multiple matching strategies** for robustness
- **Test against sample elements** mentally to validate rules

### Search Scope Selection
Choose the most appropriate search scope based on sample analysis:
- **"element_name"**: When element names clearly indicate entities
- **"path"**: When hierarchical position determines meaning
- **"attributes"**: When XML/HTML attributes contain key information
- **"all"**: When entity identification requires multiple signals

### Confidence Tuning
Set confidence thresholds based on pattern analysis:
- **0.9+**: Exact element name matches with clear semantics
- **0.8-0.9**: Path-based or strong contextual matches
- **0.7-0.8**: Semantic matches or keyword combinations
- **0.6-0.7**: Experimental or broad semantic matches

## Example Workflow

1. **Sample**: `domain_name="insider_trading", keywords="insider,transaction,SEC"`
2. **Analyze**: Find elements like `rptOwnerName`, `transactionDate`, `issuerName`
3. **Generate**: Create terms and mappings based on actual findings
4. **Validate**: Ensure rules would capture the sampled elements
5. **Optimize**: Adjust confidence and search scopes for precision

## Output Requirements

Produce a complete, production-ready ontology YAML that:
- **Captures 80%+ of relevant entities** in the sampled data
- **Uses precise, data-driven mapping rules** based on actual patterns
- **Includes meaningful relationships** derived from document structure
- **Leverages the enhanced search system** with appropriate scopes
- **Follows domain expertise** while staying grounded in data

The result should be immediately deployable for knowledge graph generation from the analyzed document corpus.