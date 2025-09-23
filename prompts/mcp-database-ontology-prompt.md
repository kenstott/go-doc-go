# MCP Database-Driven Ontology Generation

## Prerequisites

**REQUIRED**: This prompt requires access to the following MCP tools:
- `sample_elements`: Sample elements from analytics database
- `get_corpus_stats`: Get corpus statistics
- `sample_documents`: Sample documents from database
- `custom_query`: Execute custom SQL queries

**If these tools are not available, STOP and inform the user that the MCP database sampling component must be installed and configured first.**

---

## Your Task

You are an expert knowledge engineer creating domain-specific ontologies. You will use MCP database tools to sample real document and element data, then generate comprehensive ontology rules based on actual patterns found in the corpus.

## Step 1: Verify MCP Tools

First, verify that you have access to the required MCP tools by attempting to get corpus statistics:

```
Use: get_corpus_stats with no filters to get overview of the entire corpus
```

If this fails, **STOP** and inform the user that MCP database tools are required.

## Step 2: Domain Exploration

Ask the user to provide:

**Domain Name**: What domain are you creating an ontology for?

**Domain Description**: Detailed description including key concepts, business context, regulations, etc.

**Expected Entities**: What types of entities do you expect to find?

**Expected Relationships**: What relationships might exist between entities?

## Step 3: Corpus Analysis

Use the MCP tools to analyze the available data:

### 3a. Get Corpus Overview
```
Use: get_corpus_stats
Examine: total elements, document types, format distribution
```

### 3b. Explore Relevant Documents
Based on the domain, use filters to find relevant documents:
```
Use: sample_documents with filters like:
{"source": "*form*4*"} for SEC forms
{"doc_type": "xml"} for XML documents
{"source": "*earnings*"} for earnings reports
```

### 3c. Sample Domain-Relevant Elements
Use targeted sampling to find elements related to your domain:
```
Use: sample_elements with domain-specific filters:
{"structural_name": "*owner*"} - Find ownership-related elements
{"structural_name": "*date*"} - Find date elements
{"element_type": "xml_element"} - Focus on XML elements
{"format_type": "xml", "document_category": "sec_form_4"} - Specific document types
```

### 3d. Analyze Patterns
Look for:
- **Common structural names**: What element names appear frequently?
- **Path patterns**: What hierarchical structures exist?
- **Content patterns**: What actual data values are present?
- **Temporal elements**: What date/time patterns exist?
- **Attribute usage**: What metadata is available?

## Step 4: Strategic Sampling

Based on initial analysis, do targeted sampling:

### High-Frequency Elements
```
Use: sample_elements with stratify_by="structural_name" to get diverse samples
```

### Domain-Specific Elements
Create filters based on domain keywords:
```
For insider trading: {"structural_name": "*owner*"}
For financial reports: {"structural_name": "*revenue*"}
For clinical trials: {"structural_name": "*patient*"}
```

### Temporal Elements
```
Use: sample_elements with filters={"has_temporal_value": true}
```

### Document Structure Analysis
```
Use: custom_query to analyze element relationships:
"SELECT parent_id, element_type, structural_name, COUNT(*)
 FROM element_document_enriched
 WHERE structural_name IS NOT NULL
 GROUP BY parent_id, element_type, structural_name
 ORDER BY COUNT(*) DESC LIMIT 50"
```

## Step 5: Pattern Analysis

Analyze the sampled data to identify:

### Entity Patterns
- Which `structural_name` values represent entities in your domain?
- What `content_preview` patterns indicate entity types?
- Which `structural_path` patterns show entity locations?

### Relationship Patterns
- What parent-child relationships exist in the document structure?
- Which elements co-occur in the same documents?
- What temporal relationships can be inferred?

### Naming Conventions
- How are domain entities named across different document types?
- What variations exist (e.g., `rptOwnerName` vs `reportingOwnerName`)?
- What path patterns are consistent?

## Step 6: Ontology Generation

Based on the MCP-gathered data, generate a comprehensive ontology:

### Terms
Create terms for entities you actually found in the data:
```yaml
terms:
  - id: "entity_name"
    label: "Human Label"
    description: "Based on finding [X] elements named [structural_names] in [Y] documents"
    aliases: [actual variations found in structural_name field]
```

### Element Mappings
Create precise mappings using the actual patterns discovered:
```yaml
element_mappings:
  - term_id: "entity_name"
    rules:
      # Use exact structural names found in sampling
      - type: keywords
        search_scope: "element_name"
        keywords: [list of actual structural_name values found]
        confidence_threshold: 0.95

      # Use path patterns discovered in analysis
      - type: keywords
        search_scope: "path"
        keywords: [path components found in structural_path]
        confidence_threshold: 0.85

      # Use content patterns for semantic matching
      - type: semantic
        search_scope: "all"
        semantic_phrase: "description based on actual content_preview examples"
        confidence_threshold: 0.75
```

### Relationship Rules
Define relationships based on observed document structures:
```yaml
relationship_rules:
  - id: "relationship_name"
    relationship_type: "RELATIONSHIP_TYPE"
    source:
      term_id: "source_entity"
    target:
      term_id: "target_entity"
    constraints:
      same_document: true  # Based on analysis of parent-child patterns
```

## Quality Requirements

Your generated ontology must:

1. **Be grounded in sampled data**: Every term and rule must reference actual findings from MCP sampling
2. **Achieve high coverage**: Rules should capture the majority of relevant elements found in samples
3. **Use precise matching**: Leverage exact structural names and paths discovered
4. **Include statistical justification**: Reference frequency counts and pattern analysis
5. **Cover format variations**: Account for XML, JSON, CSV, etc. variations found

## Documentation

For each section of the ontology, document:
- **Data source**: Which MCP query provided the evidence
- **Frequency**: How often patterns were observed
- **Variations**: Different naming/structural patterns found
- **Coverage estimate**: Percentage of relevant elements the rule should capture

## Example Process

1. `get_corpus_stats` → "Found 15,420 elements across 342 documents"
2. `sample_elements {"format_type": "xml"}` → "Found 89 instances of 'rptOwnerName'"
3. `sample_elements {"structural_name": "*owner*"}` → "Discovered ownership patterns"
4. Generate term for "reporting_person" based on actual findings
5. Create element mapping using exact structural names found
6. Validate coverage against sample data

**Remember**: If MCP tools are not available at any point, stop and request proper setup before continuing.