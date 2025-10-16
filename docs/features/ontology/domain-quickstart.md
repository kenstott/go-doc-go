# Knowledge Engine - Quick Start Guide

## 5-Minute Setup

### Step 1: Enable Knowledge Graph Construction

The Go-Doc-Go Knowledge Engine uses declarative configuration to define your data pipeline. Edit `config.yaml`:

```yaml
relationship_detection:
  domain:
    enabled: true  # Enable knowledge graph construction
    ontologies:
      - path: "./ontologies/my-domain.yaml"
        active: true
```

### Step 2: Create Your First Declarative Ontology

The Knowledge Engine uses extensible YAML-based ontology definitions. Create `ontologies/my-domain.yaml`:

```yaml
name: my_domain
version: "1.0"
domain: business

# Define your domain concepts
terms:
  - id: customer
    name: "Customer"
    aliases: ["client", "user", "account holder"]
    
  - id: product
    name: "Product" 
    aliases: ["item", "service", "offering"]
    
  - id: invoice
    name: "Invoice"
    aliases: ["bill", "receipt", "statement"]

# Extensible rules to find these concepts in unstructured text
element_mappings:
  # Find customers by keywords
  - term: customer
    rule_type: keywords
    keywords: ["customer", "client", "account", "subscriber"]
    
  # Find products by pattern (e.g., PROD-12345)
  - term: product
    rule_type: regex
    patterns: ["PROD-[0-9]+", "SKU[0-9]{6}"]
    
  # Find invoices semantically
  - term: invoice
    rule_type: semantic
    semantic_phrase: "invoice billing payment amount due date"
    threshold: 0.7

# Declarative rules to discover relationships in your knowledge graph
relationship_rules:
  - id: customer_purchased_product
    source_terms: ["customer"]
    target_terms: ["product"]
    relationship_type: PURCHASED
    constraints:
      hierarchy_level: -1  # Same document
```

### Step 3: Run the Knowledge Engine Pipeline

```bash
# Process unstructured data through the knowledge extraction pipeline
python -m go_doc_go.main ingest --config config.yaml
```

### Step 4: Query Your Knowledge Graph

```python
from go_doc_go.storage.sqlite import SQLiteDocumentDatabase

db = SQLiteDocumentDatabase("documents.db")

# Find all customer mentions
customers = db.find_elements_by_term("customer", domain="my_domain")
for element_pk, element_id, confidence in customers:
    print(f"Found customer reference: {element_id} (confidence: {confidence:.2f})")

# Get statistics
stats = db.get_term_statistics("my_domain")
print(f"Terms found: {stats}")
```

## Common Patterns

### Pattern 1: Technical Documentation

```yaml
terms:
  - id: api_endpoint
    name: "API Endpoint"
    
  - id: error_code
    name: "Error Code"
    
element_mappings:
  - term: api_endpoint
    rule_type: regex
    patterns: 
      - "GET|POST|PUT|DELETE\\s+/[a-z/{}]+"
      - "https?://[^\\s]+/api/[^\\s]+"
    
  - term: error_code
    rule_type: regex
    patterns: ["ERR[0-9]{4}", "E[0-9]{3}", "0x[0-9A-F]{4}"]
```

### Pattern 2: Legal Documents

```yaml
terms:
  - id: legal_clause
    name: "Legal Clause"
    
  - id: party
    name: "Contracting Party"
    
element_mappings:
  - term: legal_clause
    rule_type: keywords
    keywords: 
      - "hereby agrees"
      - "shall be liable"
      - "in accordance with"
      - "terms and conditions"
    
  - term: party
    rule_type: regex
    patterns: 
      - "^Party [A-B]:"
      - "\\(\"[^\"]+\"\\)"  # Defined terms in quotes
```

### Pattern 3: Financial Reports

```yaml
terms:
  - id: financial_metric
    name: "Financial Metric"
    
  - id: currency_amount
    name: "Currency Amount"
    
element_mappings:
  - term: financial_metric
    rule_type: keywords
    keywords: ["revenue", "EBITDA", "gross margin", "net income"]
    
  - term: currency_amount
    rule_type: regex
    patterns: 
      - "\\$[0-9,]+\\.?[0-9]*[MBK]?"  # $1.5M, $23,456
      - "USD\\s+[0-9,]+"
```

## Tips for Success

### 1. Start with Keywords
Begin with simple keyword rules to test your ontology:

```yaml
- term: my_concept
  rule_type: keywords
  keywords: ["simple", "match", "words"]
```

### 2. Add Patterns for Structured Data
Use regex for codes, IDs, and formatted text:

```yaml
- term: ticket_number  
  rule_type: regex
  patterns: ["TICK-[0-9]{6}", "#[0-9]+"]
```

### 3. Use Semantic Matching for Concepts
For abstract concepts, use semantic similarity:

```yaml
- term: risk_assessment
  rule_type: semantic
  semantic_phrase: "risk hazard danger threat mitigation probability impact"
  threshold: 0.65
```

### 4. Combine Multiple Rules
Use multiple rules for better coverage:

```yaml
element_mappings:
  # Match by keyword
  - term: database
    rule_type: keywords
    keywords: ["database", "DB", "SQL"]
    
  # Also match by semantic similarity  
  - term: database
    rule_type: semantic
    semantic_phrase: "database table query schema SQL index"
    threshold: 0.7
```

### 5. Test and Iterate
Monitor what's being matched:

```python
# Check what was matched
stats = db.get_term_statistics()
for term, info in stats.items():
    print(f"{term}: {info['count']} matches, avg confidence: {info['avg_confidence']:.2f}")
```

## Troubleshooting Checklist

✅ **Knowledge extraction not working?**
- Check `enabled: true` in config.yaml
- Verify ontology file path is correct
- Check ontology YAML syntax is valid

✅ **No terms being matched?**
- Lower confidence threshold (try 0.5)
- Check element_types match your documents
- Test patterns with regex101.com

✅ **Too many false matches?**
- Increase confidence threshold
- Make patterns more specific
- Add element_type constraints

✅ **Performance issues?**
- Reduce batch_size in config
- Limit element_types in rules
- Use hierarchy constraints

## Next Steps

1. 📖 Read the [full knowledge graph documentation](domain-ontology.md)
2. 🔧 Explore the [example ontologies](../examples/ontologies/)
3. 🧪 Run the [test script](../examples/test_domain_ontology.py)
4. 🚀 Build your custom extensible knowledge pipeline!