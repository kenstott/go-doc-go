# MCP Database Sampling Usage Examples

This document shows practical examples of using the MCP database sampling tools for ontology generation.

## Tool Overview

The MCP component provides 4 tools:

1. **`sample_elements`**: Sample element records with flexible filtering
2. **`get_corpus_stats`**: Get statistical overview of the corpus
3. **`sample_documents`**: Sample document records
4. **`custom_query`**: Execute custom SQL for advanced analysis

## Example 1: Insider Trading Domain

### Step 1: Corpus Overview
```
Tool: get_corpus_stats
Input: (no filters)
Output: {
  "total_elements": 15420,
  "total_documents": 342,
  "distinct_element_types": 8,
  "format_distribution": {
    "format_xml": 12850,
    "format_json": 1890,
    "format_csv": 680
  }
}
```

### Step 2: Find SEC Documents
```
Tool: sample_documents
Input:
  filters: {"source": "*form*", "document_category": "sec_*"}
  limit: 20
Output: [
  {
    "doc_id": "doc_12345",
    "source": "/sec/form4_apple_2023.xml",
    "doc_type": "xml",
    "document_category": "sec_form_4"
  },
  ...
]
```

### Step 3: Sample Ownership Elements
```
Tool: sample_elements
Input:
  filters: {"structural_name": "*owner*", "format_type": "xml"}
  limit: 50
  stratify_by: "structural_name"
Output: {
  "elements": [
    {
      "structural_name": "rptOwnerName",
      "structural_path": "/ownershipDocument/reportingOwner/rptOwnerName",
      "content_preview": "BELL JAMES A",
      "element_type": "xml_element"
    },
    {
      "structural_name": "issuerName",
      "structural_path": "/ownershipDocument/issuer/issuerName",
      "content_preview": "Apple Inc.",
      "element_type": "xml_element"
    }
  ],
  "count": 50
}
```

### Step 4: Transaction Elements
```
Tool: sample_elements
Input:
  filters: {"structural_name": "*transaction*"}
  limit: 30
Output: {
  "elements": [
    {
      "structural_name": "transactionDate",
      "content_preview": "2023-02-01",
      "has_temporal_value": true
    },
    {
      "structural_name": "transactionShares",
      "content_preview": "1685"
    }
  ]
}
```

### Step 5: Custom Pattern Analysis
```
Tool: custom_query
Input:
  query: "SELECT structural_name, COUNT(*) as freq FROM element_document_enriched WHERE structural_name ILIKE '%owner%' OR structural_name ILIKE '%transaction%' GROUP BY structural_name ORDER BY freq DESC"
Output: {
  "results": [
    {"structural_name": "rptOwnerName", "freq": 89},
    {"structural_name": "transactionDate", "freq": 76},
    {"structural_name": "transactionShares", "freq": 71}
  ]
}
```

## Example 2: Financial Reports Domain

### Step 1: Explore Financial Documents
```
Tool: sample_documents
Input:
  filters: {"source": "*earnings*"}
  limit: 15
```

### Step 2: Revenue/Earnings Elements
```
Tool: sample_elements
Input:
  filters: {"structural_name": "*revenue*"}
  limit: 40
```

### Step 3: Financial Metrics
```
Tool: sample_elements
Input:
  filters: {"structural_name": "*income*"}
  limit: 40
```

## Example 3: Clinical Trial Domain

### Step 1: Medical Documents
```
Tool: sample_documents
Input:
  filters: {"source": "*clinical*", "doc_type": "xml"}
```

### Step 2: Patient Data Elements
```
Tool: sample_elements
Input:
  filters: {"structural_name": "*patient*"}
  stratify_by: "document_category"
```

### Step 3: Trial Outcome Elements
```
Tool: sample_elements
Input:
  filters: {"structural_name": "*outcome*"}
```

## Advanced Usage Patterns

### Stratified Sampling by Document Type
```
Tool: sample_elements
Input:
  filters: {"element_type": "xml_element"}
  limit: 200
  stratify_by: "document_category"
  random_seed: 42
```
This ensures you get balanced samples across different document categories.

### Temporal Element Analysis
```
Tool: sample_elements
Input:
  filters: {"has_temporal_value": true}
  limit: 100
```
Focus on date/time elements for temporal relationship analysis.

### Custom Relationship Analysis
```
Tool: custom_query
Input:
  query: "SELECT parent_id, structural_name, COUNT(*) FROM element_document_enriched WHERE parent_id IS NOT NULL GROUP BY parent_id, structural_name HAVING COUNT(*) > 5 ORDER BY COUNT(*) DESC LIMIT 20"
```
Find parent-child patterns for relationship rules.

### Content Pattern Analysis
```
Tool: custom_query
Input:
  query: "SELECT structural_name, content_preview, COUNT(*) FROM element_document_enriched WHERE structural_name ILIKE '%name%' GROUP BY structural_name, content_preview ORDER BY COUNT(*) DESC LIMIT 50"
```
Analyze actual content values for entity identification.

## Best Practices

### 1. Start Broad, Then Focus
- Begin with `get_corpus_stats` for overview
- Use broad filters to explore document types
- Narrow down to domain-specific patterns

### 2. Use Stratification for Diversity
- Always use `stratify_by` when sampling >50 elements
- Stratify by `element_type`, `document_category`, or `structural_name`

### 3. Combine Multiple Sampling Strategies
- High-frequency elements for coverage
- Domain-specific elements for precision
- Temporal elements for event analysis
- Custom queries for relationship discovery

### 4. Document Your Findings
For each sampling result, note:
- Which filters were used
- How many elements were found
- Key patterns discovered
- Confidence in the patterns

This data will directly inform your ontology rules and confidence thresholds.