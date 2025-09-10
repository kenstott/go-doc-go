# Knowledge Engine Ontology Management Guide

## Table of Contents
1. [Introduction](#introduction)
2. [Ontology Design Principles](#ontology-design-principles)
3. [Creating Your First Ontology](#creating-your-first-ontology)
4. [Advanced Ontology Features](#advanced-ontology-features)
5. [Best Practices](#best-practices)
6. [Testing and Validation](#testing-and-validation)
7. [Maintenance and Evolution](#maintenance-and-evolution)
8. [Performance Optimization](#performance-optimization)
9. [Common Patterns](#common-patterns)
10. [Troubleshooting](#troubleshooting)

## Introduction

Ontologies in the Go-Doc-Go Knowledge Engine are declarative, extensible knowledge representations that define:
- **Terms**: Domain-specific concepts your business cares about
- **Entity Extraction Rules**: How to identify these concepts in unstructured text
- **Relationship Rules**: How concepts relate in your knowledge graph

Think of ontologies as the declarative business logic layer that transforms raw unstructured text analysis into actionable business intelligence through an extensible pipeline architecture.

## Ontology Design Principles

### 1. Start with Business Questions

Before creating an ontology, identify the key questions you want to answer from your unstructured data:
- What entities do we need to track across documents?
- What relationships matter for decision-making in our knowledge graph?
- What patterns indicate important events or conditions?

### 2. Extensible, Iterative Development

The Knowledge Engine's extensible architecture supports iterative ontology development. Start simple and refine based on results:
```yaml
# Version 1: Basic identification
terms:
  - id: customer
    name: "Customer"

# Version 2: Add aliases
terms:
  - id: customer
    name: "Customer"
    aliases: ["client", "account", "subscriber"]

# Version 3: Add semantic understanding
element_mappings:
  - term: customer
    rule_type: semantic
    semantic_phrase: "customer client account subscriber user buyer"
    threshold: 0.7
```

### 3. Balance Precision and Recall

- **High Precision** (few false positives): Use higher thresholds, specific patterns
- **High Recall** (few false negatives): Use lower thresholds, broader patterns

## Creating Your First Declarative Ontology

### Step 1: Define Your Domain

The Knowledge Engine uses declarative YAML configuration for domain definitions:

```yaml
name: retail_analytics
version: "1.0"
domain: retail
description: "Ontology for retail business analysis"
author: "Data Science Team"
created_date: "2024-01-15"
```

### Step 2: Identify Core Terms

```yaml
terms:
  - id: product
    name: "Product"
    description: "Items sold by the company"
    category: "inventory"
    
  - id: customer_segment
    name: "Customer Segment"
    description: "Categories of customers"
    category: "customers"
    
  - id: promotion
    name: "Promotion"
    description: "Sales and marketing campaigns"
    category: "marketing"
```

### Step 3: Create Extensible Extraction Rules

#### Semantic Rules
The Knowledge Engine's extensible rule system supports semantic matching for concepts with varied expressions:

```yaml
element_mappings:
  - term: customer_complaint
    rule_type: semantic
    semantic_phrase: "complaint issue problem dissatisfied unhappy frustrated refund return"
    threshold: 0.65
    element_types: ["paragraph", "list_item"]
    confidence_calculation: weighted
```

#### Pattern Rules
Best for structured identifiers:

```yaml
element_mappings:
  - term: order_number
    rule_type: regex
    patterns:
      - "ORD-[0-9]{8}"           # ORD-12345678
      - "ORDER#[0-9]+"            # ORDER#123456
      - "[0-9]{4}-[0-9]{6}"       # 2024-123456
    case_sensitive: false
    element_types: ["*"]  # Search all element types
```

#### Keyword Rules
Best for exact terminology:

```yaml
element_mappings:
  - term: return_policy
    rule_type: keywords
    keywords:
      - "return policy"
      - "refund policy"
      - "30-day return"
      - "money back guarantee"
    case_sensitive: false
    element_types: ["paragraph", "heading"]
```

### Step 4: Define Knowledge Graph Relationships

Define how concepts connect in your knowledge graph:

```yaml
relationship_rules:
  - id: product_in_promotion
    name: "Product In Promotion"
    description: "Links products to active promotions"
    source_terms: ["product"]
    target_terms: ["promotion"]
    relationship_type: FEATURED_IN
    
    # Use semantic patterns to identify relationships
    semantic_patterns:
      - source: "product item SKU merchandise"
        target: "promotion sale discount offer campaign"
        similarity_threshold: 0.7
    
    # Constrain to same document section
    constraints:
      hierarchy_level: 0  # Same parent element
      direction: any
    
    confidence_weight: 0.85
```

## Advanced Knowledge Engine Features

### Extensible Composite Rules

The Knowledge Engine's extensible architecture allows combining multiple detection methods for better accuracy:

```yaml
element_mappings:
  # Primary: Semantic matching
  - term: financial_metric
    rule_type: semantic
    semantic_phrase: "revenue profit margin EBITDA cash flow ROI"
    threshold: 0.7
    confidence_weight: 0.8
    
  # Secondary: Pattern matching for specific formats
  - term: financial_metric
    rule_type: regex
    patterns:
      - "\\$[0-9,]+[MBK]?"  # $1.5M, $23K
      - "[0-9]+\\.?[0-9]*%"  # 15.5%
    confidence_weight: 1.0
    
  # Tertiary: Keywords for exact matches
  - term: financial_metric
    rule_type: keywords
    keywords: ["gross margin", "net income", "operating expense"]
    confidence_weight: 0.9
```

### Hierarchical Knowledge Structures

Create declarative term hierarchies for better knowledge graph organization:

```yaml
terms:
  - id: vehicle
    name: "Vehicle"
    parent: null
    
  - id: passenger_vehicle
    name: "Passenger Vehicle"
    parent: vehicle
    
  - id: sedan
    name: "Sedan"
    parent: passenger_vehicle
    aliases: ["saloon", "4-door"]
    
  - id: suv
    name: "SUV"
    parent: passenger_vehicle
    aliases: ["sport utility vehicle", "crossover"]
```

### Contextual Rules

The pipeline supports rules that consider unstructured text context:

```yaml
element_mappings:
  - term: executive_statement
    rule_type: semantic
    semantic_phrase: "pleased announce progress strategic outlook forecast"
    threshold: 0.6
    # Only in specific contexts
    element_types: ["paragraph"]
    context_requirements:
      # Must be near a speaker identification
      near_elements:
        - type: "paragraph"
          contains_term: "executive"
          max_distance: 2
```

### Dynamic Confidence Calculation

The extensible confidence system adjusts scores based on multiple factors:

```yaml
element_mappings:
  - term: risk_indicator
    rule_type: semantic
    semantic_phrase: "risk threat vulnerability exposure hazard"
    threshold: 0.6
    
    # Confidence modifiers
    confidence_calculation: weighted
    confidence_modifiers:
      - factor: "element_type"
        heading: 1.2      # Higher confidence in headings
        paragraph: 1.0
        list_item: 0.9
      - factor: "document_position"
        early: 1.1        # Higher confidence early in document
        middle: 1.0
        late: 0.9
```

## Best Practices

### 1. Naming Conventions

Use consistent, descriptive names:

```yaml
# GOOD: Clear, descriptive, consistent
terms:
  - id: customer_acquisition_cost
  - id: customer_lifetime_value
  - id: customer_churn_rate

# BAD: Inconsistent, unclear
terms:
  - id: CAC
  - id: cust_ltv
  - id: ChurnRate
```

### 2. Documentation

Document your rules and reasoning:

```yaml
element_mappings:
  - term: regulatory_reference
    rule_type: regex
    patterns:
      # US Federal Regulations (e.g., "26 CFR 1.401")
      - "[0-9]+ (CFR|USC) [0-9]+\\.?[0-9]*"
      
      # EU Regulations (e.g., "Regulation (EU) 2016/679")
      - "Regulation \\(EU\\) [0-9]{4}/[0-9]+"
      
      # ISO Standards (e.g., "ISO 9001:2015")
      - "ISO [0-9]+:[0-9]{4}"
    
    # Why: Legal documents frequently reference these patterns
    # Tested on: 500 sample legal documents
    # Accuracy: 95% precision, 88% recall
```

### 3. Version Control

Track ontology changes:

```yaml
name: financial_ontology
version: "2.1.0"
changelog:
  - version: "2.1.0"
    date: "2024-01-15"
    changes:
      - "Added cryptocurrency terms"
      - "Improved revenue recognition patterns"
      - "Fixed false positives in date patterns"
  - version: "2.0.0"
    date: "2023-12-01"
    changes:
      - "Major refactor of relationship rules"
      - "Added IFRS terminology"
```

### 4. Extensible Modular Design

The Knowledge Engine supports modular ontology architecture. Split large ontologies into reusable modules:

```yaml
# main_ontology.yaml
name: enterprise_ontology
imports:
  - "./modules/financial.yaml"
  - "./modules/legal.yaml"
  - "./modules/operational.yaml"
  
# modules/financial.yaml
module: financial
terms:
  - id: revenue
  - id: expense
  - id: profit_margin
```

## Testing and Validation

### 1. Unit Testing Rules

Create test cases for your rules:

```python
import pytest
from go_doc_go.domain import OntologyLoader, OntologyEvaluator

class TestFinancialOntology:
    def test_revenue_extraction(self):
        """Test that revenue terms are correctly identified."""
        ontology = OntologyLoader().load_from_file("financial.yaml")
        evaluator = OntologyEvaluator(ontology)
        
        test_cases = [
            ("Q3 revenue increased to $2.3M", True),
            ("Total sales reached 2.3 million", True),
            ("The weather was revenue", False),  # Should not match
        ]
        
        for text, should_match in test_cases:
            element = {"text": text, "element_type": "paragraph"}
            mappings = evaluator.map_element_to_terms(element)
            matched = any(m.term_id == "revenue" for m in mappings)
            assert matched == should_match
```

### 2. Coverage Analysis

Measure how well your ontology covers your documents:

```python
def analyze_ontology_coverage(ontology_path, document_samples):
    """Analyze what percentage of documents have term matches."""
    ontology = load_ontology(ontology_path)
    
    stats = {
        "total_documents": len(document_samples),
        "documents_with_matches": 0,
        "total_terms_found": 0,
        "unmapped_elements": []
    }
    
    for doc in document_samples:
        mappings = extract_terms(doc, ontology)
        if mappings:
            stats["documents_with_matches"] += 1
            stats["total_terms_found"] += len(mappings)
        else:
            stats["unmapped_elements"].append(doc["id"])
    
    stats["coverage_rate"] = stats["documents_with_matches"] / stats["total_documents"]
    return stats
```

### 3. Precision/Recall Testing

Measure accuracy against labeled data:

```python
def evaluate_precision_recall(ontology, labeled_data):
    """Evaluate precision and recall against manually labeled data."""
    true_positives = 0
    false_positives = 0
    false_negatives = 0
    
    for doc in labeled_data:
        predicted = extract_terms(doc["content"], ontology)
        actual = doc["labels"]
        
        for term in predicted:
            if term in actual:
                true_positives += 1
            else:
                false_positives += 1
        
        for term in actual:
            if term not in predicted:
                false_negatives += 1
    
    precision = true_positives / (true_positives + false_positives)
    recall = true_positives / (true_positives + false_negatives)
    f1_score = 2 * (precision * recall) / (precision + recall)
    
    return {
        "precision": precision,
        "recall": recall,
        "f1_score": f1_score
    }
```

## Maintenance and Evolution

### 1. Regular Review Cycles

Schedule periodic reviews:

- **Weekly**: Review false positives/negatives
- **Monthly**: Analyze coverage statistics
- **Quarterly**: Major updates and refactoring

### 2. Feedback Integration

Create a feedback loop:

```python
class OntologyFeedbackCollector:
    def record_false_positive(self, term_id, element_text, user_id):
        """Record when a term match was incorrect."""
        
    def record_false_negative(self, suggested_term, element_text, user_id):
        """Record when a term should have been matched."""
        
    def generate_improvement_report(self):
        """Generate suggestions for ontology improvements."""
```

### 3. A/B Testing

Test ontology changes:

```yaml
# ontology_v1.yaml (control)
element_mappings:
  - term: customer
    threshold: 0.7

# ontology_v2.yaml (test)
element_mappings:
  - term: customer
    threshold: 0.65  # Lower threshold
```

### 4. Migration Strategies

Handle ontology updates gracefully:

```python
def migrate_ontology(old_version, new_version, database):
    """Migrate extracted entities to new ontology version."""
    migration_map = load_migration_map(old_version, new_version)
    
    for old_term, new_term in migration_map.items():
        database.update_term_references(old_term, new_term)
    
    # Re-process recent documents with new rules
    recent_docs = database.get_recent_documents(days=30)
    reprocess_with_ontology(recent_docs, new_version)
```

## Performance Optimization

### 1. Rule Ordering

Place most selective rules first:

```yaml
element_mappings:
  # Fast: Exact pattern match (runs first)
  - term: product_code
    rule_type: regex
    patterns: ["PROD-[0-9]{6}"]
    
  # Slower: Semantic similarity (runs if pattern doesn't match)
  - term: product_code
    rule_type: semantic
    semantic_phrase: "product code SKU item number"
    threshold: 0.8
```

### 2. Element Type Filtering

Restrict rules to relevant element types:

```yaml
element_mappings:
  # Only search in likely locations
  - term: document_title
    rule_type: semantic
    element_types: ["heading", "title"]  # Don't search paragraphs
    
  - term: footnote_reference
    rule_type: regex
    element_types: ["footnote", "endnote"]  # Very specific
```

### 3. Caching Strategies

Cache frequently used patterns:

```yaml
element_mappings:
  - term: common_product
    rule_type: keywords
    keywords: ["iPhone", "iPad", "MacBook"]  # These will be cached
    cache_ttl: 3600  # Cache for 1 hour
```

### 4. Batch Processing

Configure optimal batch sizes:

```yaml
processing:
  batch_size: 100  # Process 100 elements at a time
  parallel_workers: 4  # Use 4 parallel workers
  memory_limit: 2048  # MB
```

## Common Patterns

### Financial Documents

```yaml
name: financial_documents
terms:
  - id: fiscal_period
    patterns: ["Q[1-4] [0-9]{4}", "FY[0-9]{2,4}"]
    
  - id: currency_amount
    patterns: ["\\$[0-9,]+\\.?[0-9]*[MBK]?", "USD [0-9,]+"]
    
  - id: percentage_change
    patterns: ["[+-]?[0-9]+\\.?[0-9]*%", "increased? [0-9]+%"]
```

### Legal Documents

```yaml
name: legal_documents
terms:
  - id: contract_clause
    semantic: "hereby agrees shall liable pursuant whereas"
    
  - id: legal_entity
    patterns: ["[A-Z][A-Za-z\\s&,.]+ (Inc|LLC|Corp|LLP)\\.?"]
    
  - id: case_citation
    patterns: ["[0-9]+ [A-Z][a-z]+ [0-9]+", "No\\.? [0-9]+-[A-Z]+-[0-9]+"]
```

### Technical Documentation

```yaml
name: technical_docs
terms:
  - id: error_code
    patterns: ["ERR[0-9]{4}", "0x[0-9A-Fa-f]{4,8}", "E[0-9]{3}"]
    
  - id: version_number
    patterns: ["v?[0-9]+\\.[0-9]+(\\.[0-9]+)?", "version [0-9]+"]
    
  - id: api_endpoint
    patterns: ["(GET|POST|PUT|DELETE) /[a-z0-9/_-]+"]
```

### Medical/Healthcare

```yaml
name: healthcare
terms:
  - id: medical_condition
    semantic: "diagnosis symptoms treatment chronic acute"
    
  - id: drug_name
    patterns: ["[A-Z][a-z]+[a-z]*(mab|nib|cillin|prazole)"]
    
  - id: icd_code
    patterns: ["[A-Z][0-9]{2}(\\.[0-9]{1,2})?"]  # ICD-10
```

## Troubleshooting

### Problem: Too Many False Positives

**Symptoms**: Terms matching irrelevant content

**Solutions**:
1. Increase confidence thresholds
2. Add negative patterns (exclusions)
3. Restrict element types
4. Use more specific patterns

```yaml
element_mappings:
  - term: email_address
    rule_type: regex
    patterns: ["[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}"]
    # Add exclusions
    exclusion_patterns: ["example@example.com", "test@test.com"]
    # Increase threshold
    min_confidence: 0.8
```

### Problem: Missing Expected Matches

**Symptoms**: Known terms not being extracted

**Solutions**:
1. Lower confidence thresholds
2. Add more aliases and variations
3. Broaden semantic phrases
4. Check element type restrictions

```yaml
element_mappings:
  - term: customer
    rule_type: semantic
    # Broader semantic phrase
    semantic_phrase: "customer client user account subscriber member patron buyer purchaser"
    # Lower threshold
    threshold: 0.6  # Was 0.8
    # Search more element types
    element_types: ["*"]  # Was ["paragraph"]
```

### Problem: Slow Processing

**Symptoms**: Ontology processing takes too long

**Solutions**:
1. Optimize rule order (fast rules first)
2. Reduce batch size for memory constraints
3. Limit element types for rules
4. Use pattern caching

```yaml
performance:
  rule_execution_order: ["regex", "keywords", "semantic"]
  cache_compiled_patterns: true
  max_parallel_rules: 10
  element_sampling:
    enabled: true
    sample_rate: 0.1  # Test on 10% first
```

### Problem: Relationship Explosion

**Symptoms**: Too many relationships being created

**Solutions**:
1. Add hierarchy constraints
2. Increase relationship confidence thresholds
3. Add relationship filters

```yaml
relationship_rules:
  - id: limited_relationship
    source_terms: ["term1"]
    target_terms: ["term2"]
    constraints:
      hierarchy_level: 0  # Same section only
      max_distance: 5     # Within 5 elements
      same_document: true
    min_confidence: 0.8   # High threshold
    max_relationships_per_source: 3  # Limit
```

## Conclusion

Effective ontology management in the Go-Doc-Go Knowledge Engine is an iterative process that requires:
- Clear business objectives for your unstructured data pipeline
- Systematic testing and validation of knowledge extraction
- Regular maintenance and updates to your declarative configurations
- Performance monitoring of the extensible pipeline
- User feedback integration for continuous improvement

Start simple with the extensible architecture, measure results from your knowledge graph, and refine continuously. The goal is not perfection but continuous improvement aligned with business value through intelligent search and knowledge discovery.

For specific questions or advanced use cases, consult the [Knowledge Graph Construction documentation](domain-ontology.md) or join our [community forum](https://github.com/go-doc-go/discussions).