# Ontology System: Knowledge Graph Construction

Transform your documents into intelligent knowledge graphs by defining domain ontologies that automatically extract entities and discover relationships at scale.

## Overview

The ontology system operates in two phases:
1. **Entity Extraction** - Identify domain-specific concepts in documents
2. **Relationship Discovery** - Find connections between entities using business rules

## Quick Start

### 1. Define Your Domain

Create an ontology file describing your domain:

```yaml
# ontologies/automotive.yaml
name: automotive_safety
version: "1.0"
description: "Automotive safety and compliance knowledge extraction"

# Define what entities matter in your domain
entities:
  brake_system:
    name: "Brake System"
    description: "Vehicle braking systems and components"
    extraction_rules:
      - type: "semantic"
        phrase: "brake ABS anti-lock rotor pad caliper disc drum"
        threshold: 0.7
      - type: "pattern"
        patterns: ["ABS", "anti-lock.*brake", "brake.*system"]
        
  safety_standard:
    name: "Safety Standard"
    description: "Automotive safety regulations and standards"
    extraction_rules:
      - type: "pattern"  
        patterns: ["FMVSS-[0-9]+", "ISO[\\s-]?26262", "UN ECE R[0-9]+"]
      - type: "keyword"
        keywords: ["crash test", "safety rating", "NCAP"]
        
  vehicle_model:
    name: "Vehicle Model"
    description: "Specific vehicle makes and models"
    extraction_rules:
      - type: "pattern"
        patterns: ["Model Year [0-9]{4}", "[A-Z][a-z]+ [A-Z][a-z0-9]+"]
      - type: "semantic"
        phrase: "vehicle car truck SUV sedan coupe"
        threshold: 0.6

# Define how entities relate to each other
relationships:
  compliance:
    source_entities: ["brake_system"]
    target_entities: ["safety_standard"] 
    relationship_type: "complies_with"
    extraction_rules:
      - type: "proximity"
        max_distance: 2  # within 2 sentences
        confidence: 0.8
      - type: "pattern"
        patterns: ["meets.*standard", "complies.*with", "certified.*under"]
        
  installation:
    source_entities: ["brake_system"]
    target_entities: ["vehicle_model"]
    relationship_type: "installed_in"
    extraction_rules:
      - type: "semantic_similarity"
        threshold: 0.75
      - type: "document_structure"
        same_section: true
```

### 2. Configure Processing

```yaml
# config.yaml
storage:
  backend: "postgresql"
  
relationship_detection:
  domain:
    enabled: true
    ontologies:
      - path: "./ontologies/automotive.yaml"
        active: true
    
    # Extraction settings
    entity_extraction:
      min_confidence: 0.6
      max_entities_per_element: 10
      batch_size: 100
      
    relationship_detection:
      min_confidence: 0.7
      max_relationships_per_entity: 20
      cross_document: true  # Find relationships across documents
      
embedding:
  enabled: true  # Required for semantic matching
  provider: "fastembed"
  model: "BAAI/bge-small-en-v1.5"
```

### 3. Process Documents

```python
from go_doc_go import Config, ingest_documents

config = Config("config.yaml")
result = ingest_documents(config)

print(f"Extracted {result['entities']} entities")
print(f"Found {result['entity_relationships']} relationships") 
```

### 4. Query Knowledge Graph

```python
from go_doc_go import Config

config = Config("config.yaml")
db = config.get_storage_backend()

# Find brake systems
brake_systems = db.get_entities(entity_type="brake_system")
print(f"Found {len(brake_systems)} brake systems")

# Find what standards they comply with
for brake_system in brake_systems[:5]:
    standards = db.get_related_entities(
        entity=brake_system,
        relationship_type="complies_with"
    )
    print(f"{brake_system.name} complies with {len(standards)} standards")
    
    for standard in standards:
        print(f"  - {standard.name} (confidence: {standard.confidence})")

# Cross-document analysis
vehicles_with_abs = db.query_entities("""
    MATCH (brake:brake_system)-[:installed_in]->(vehicle:vehicle_model)
    WHERE brake.name CONTAINS 'ABS'
    RETURN vehicle.name, COUNT(brake) as brake_systems
    ORDER BY brake_systems DESC
""")
```

## Entity Extraction Methods

### 1. Semantic Similarity

Match concepts using AI embeddings - great for finding related terms:

```yaml
entities:
  financial_metric:
    extraction_rules:
      - type: "semantic"
        phrase: "revenue profit margin EBITDA cash flow ROI earnings"
        threshold: 0.7
        context_window: 50  # words around match
```

**How it works:**
- Embeds your phrase and document segments
- Finds segments with high semantic similarity
- Captures synonyms and related concepts automatically
- Example: "quarterly earnings" matches "Q4 profit", "revenue growth"

### 2. Pattern Matching (Regex)

Extract structured data like codes, IDs, references:

```yaml
entities:
  product_code:
    extraction_rules:
      - type: "pattern"
        patterns: 
          - "PROD-[0-9]{4,6}"           # PROD-12345
          - "SKU[:\\s]+[A-Z0-9-]+"      # SKU: ABC-123
          - "Part#?\\s*([A-Z0-9-]+)"    # Part#ABC-123
        case_sensitive: false
```

**Best for:**
- Product codes, IDs, serial numbers
- Regulatory references (e.g., "Section 4.2.1")
- Structured identifiers
- Date patterns, currency amounts

### 3. Keyword Matching

Simple exact or fuzzy word matching:

```yaml
entities:
  executive_role:
    extraction_rules:
      - type: "keyword"
        keywords: ["CEO", "Chief Executive Officer", "CFO", "President"]
        fuzzy_match: true
        min_similarity: 0.8
```

**Options:**
- `exact_match: true` - Must match exactly
- `fuzzy_match: true` - Allow typos and variations
- `case_sensitive: false` - Ignore case
- `whole_words_only: true` - Don't match partial words

### 4. Combined Rules

Use multiple methods together for better coverage:

```yaml
entities:
  company:
    extraction_rules:
      # Catch ticker symbols
      - type: "pattern"
        patterns: ["NASDAQ:[A-Z]{2,5}", "NYSE:[A-Z]{2,5}"]
        confidence: 0.9
        
      # Catch company names semantically  
      - type: "semantic"
        phrase: "corporation company business enterprise firm"
        threshold: 0.6
        confidence: 0.7
        
      # Catch obvious keywords
      - type: "keyword"
        keywords: ["Inc.", "Corp.", "LLC", "Ltd.", "Co."]
        confidence: 0.8
```

## Relationship Discovery

### 1. Proximity-Based

Find entities mentioned near each other:

```yaml
relationships:
  drug_side_effect:
    source_entities: ["medication"]
    target_entities: ["side_effect"]
    relationship_type: "causes"
    extraction_rules:
      - type: "proximity"
        max_distance: 3        # within 3 sentences
        max_word_distance: 50  # or 50 words
        confidence: 0.7
```

### 2. Pattern-Based

Use linguistic patterns to identify relationships:

```yaml
relationships:
  acquisition:
    source_entities: ["company"]
    target_entities: ["company"] 
    relationship_type: "acquires"
    extraction_rules:
      - type: "pattern"
        patterns:
          - "{source}.*acquired.*{target}"
          - "{source}.*bought.*{target}"
          - "{target}.*acquired by.*{source}"
        confidence: 0.9
```

### 3. Semantic Relationships

Use AI to understand relationship meaning:

```yaml
relationships:
  treatment_efficacy:
    source_entities: ["treatment"]
    target_entities: ["condition"]
    relationship_type: "treats"
    extraction_rules:
      - type: "semantic_similarity"
        relationship_phrase: "treats cures helps alleviates improves"
        threshold: 0.75
        context_window: 100
```

### 4. Document Structure

Leverage document hierarchy and structure:

```yaml
relationships:
  section_content:
    source_entities: ["topic"]
    target_entities: ["content"]
    relationship_type: "discusses"
    extraction_rules:
      - type: "document_structure"
        same_section: true     # Same document section
        same_page: false       # Can span pages
        same_document: true    # Must be same document
        confidence: 0.8
```

## Advanced Configuration

### Cross-Document Relationships

Find relationships spanning multiple documents:

```yaml
relationship_detection:
  domain:
    relationship_detection:
      cross_document: true
      cross_doc_similarity_threshold: 0.8
      max_cross_doc_distance: 10000  # words apart
      cross_doc_methods:
        - "entity_co_occurrence"
        - "semantic_similarity"
        - "shared_metadata"
```

### Entity Disambiguation

Handle entities with multiple meanings:

```yaml
entities:
  apple:
    disambiguation:
      contexts:
        technology: 
          patterns: ["Apple Inc", "iPhone", "Mac", "iOS"]
          confidence_boost: 0.2
        fruit:
          patterns: ["orchard", "harvest", "eat", "juice"]  
          confidence_boost: 0.2
      default_context: "technology"
```

### Confidence Tuning

Optimize precision vs. recall:

```yaml
relationship_detection:
  domain:
    # High precision (fewer false positives)
    entity_extraction:
      min_confidence: 0.8
      max_entities_per_element: 5
      
    relationship_detection:
      min_confidence: 0.85
      max_relationships_per_entity: 10
      
    # High recall (catch more relationships)  
    # entity_extraction:
    #   min_confidence: 0.5
    #   max_entities_per_element: 20
    #
    # relationship_detection:
    #   min_confidence: 0.6
    #   max_relationships_per_entity: 50
```

## Real-World Examples

### Financial Services

```yaml
# ontologies/financial.yaml
name: financial_markets
entities:
  company:
    extraction_rules:
      - type: "pattern"
        patterns: ["NASDAQ:[A-Z]+", "NYSE:[A-Z]+"]
      - type: "semantic"
        phrase: "corporation company business firm enterprise"
        
  executive:
    extraction_rules:
      - type: "keyword"
        keywords: ["CEO", "CFO", "CTO", "President", "Chairman"]
      - type: "pattern"
        patterns: ["Chief [A-Z][a-z]+ Officer"]
        
  financial_metric:
    extraction_rules:
      - type: "semantic"
        phrase: "revenue profit earnings EBITDA margin cash flow"
      - type: "pattern"
        patterns: ["\\$[0-9,.]+\\s*(million|billion)", "[0-9]+%\\s*growth"]

relationships:
  executive_discusses_metric:
    source_entities: ["executive"]
    target_entities: ["financial_metric"]
    relationship_type: "discusses"
    extraction_rules:
      - type: "proximity"
        max_distance: 2
      - type: "pattern"
        patterns: ["{source} reported {target}", "{source} announced {target}"]
```

### Legal/Compliance

```yaml
# ontologies/legal.yaml  
name: contract_analysis
entities:
  party:
    extraction_rules:
      - type: "pattern"
        patterns: ["\\\"[A-Z][a-z\\s]+\\\"", "Party [A-Z]"]
      - type: "semantic"
        phrase: "company corporation entity party counterparty"
        
  obligation:
    extraction_rules:
      - type: "keyword"
        keywords: ["shall", "must", "required to", "obligated", "covenant"]
      - type: "pattern"
        patterns: ["Party [A-Z] shall", "agrees to"]
        
  clause_reference:
    extraction_rules:
      - type: "pattern"
        patterns: ["Section [0-9.]+", "Article [IVX]+", "Exhibit [A-Z]"]

relationships:
  party_obligation:
    source_entities: ["party"]
    target_entities: ["obligation"]
    relationship_type: "bound_by"
    extraction_rules:
      - type: "proximity"
        max_distance: 1
      - type: "document_structure"
        same_section: true
```

### Healthcare/Pharmaceutical

```yaml
# ontologies/medical.yaml
name: clinical_research
entities:
  drug:
    extraction_rules:
      - type: "semantic"
        phrase: "medication drug pharmaceutical compound therapy treatment"
      - type: "pattern"
        patterns: ["[A-Z][a-z]+[0-9]+", "[A-Z]{2,4}-[0-9]+"]  # Drug codes
        
  adverse_event:
    extraction_rules:
      - type: "semantic" 
        phrase: "side effect adverse event reaction toxicity"
      - type: "keyword"
        keywords: ["nausea", "headache", "fatigue", "rash"]
        
  clinical_trial:
    extraction_rules:
      - type: "pattern"
        patterns: ["Phase [I]+", "NCT[0-9]+", "Study [A-Z0-9-]+"]
      - type: "semantic"
        phrase: "clinical trial study research protocol"

relationships:
  drug_causes_event:
    source_entities: ["drug"]
    target_entities: ["adverse_event"]
    relationship_type: "causes"
    extraction_rules:
      - type: "pattern"
        patterns: ["{source} caused {target}", "patients taking {source} experienced {target}"]
      - type: "proximity"
        max_distance: 3
        confidence: 0.6
```

## Performance Optimization

### Batch Processing

```yaml
relationship_detection:
  domain:
    entity_extraction:
      batch_size: 500         # Process many elements at once
      parallel_workers: 4     # Parallel extraction
      cache_embeddings: true  # Cache semantic embeddings
      
    relationship_detection:
      batch_size: 100
      parallel_workers: 2
      relationship_cache_size: 10000
```

### Memory Management

```yaml
relationship_detection:
  domain:
    # Limit memory usage
    max_entities_in_memory: 50000
    max_relationships_in_memory: 100000
    
    # Streaming for large datasets
    stream_processing: true
    checkpoint_frequency: 1000  # Save progress every N documents
```

### Database Optimization

```yaml
storage:
  backend: "postgresql"
  
  # Indexes for entity queries
  create_entity_indexes: true
  entity_index_fields: ["entity_type", "name", "confidence"]
  
  # Relationship query optimization  
  create_relationship_indexes: true
  relationship_index_fields: ["source_entity_id", "target_entity_id", "relationship_type"]
```

## Quality Assessment

### Evaluation Metrics

```python
from go_doc_go.domain import evaluate_extraction_quality

# Test extraction on known data
evaluation = evaluate_extraction_quality(
    ontology_path="ontologies/automotive.yaml",
    test_documents="test_data/automotive_docs.json",
    ground_truth="test_data/automotive_entities.json"
)

print(f"Entity extraction:")
print(f"  Precision: {evaluation.entity_precision:.3f}")
print(f"  Recall: {evaluation.entity_recall:.3f}")  
print(f"  F1 Score: {evaluation.entity_f1:.3f}")

print(f"Relationship detection:")
print(f"  Precision: {evaluation.relationship_precision:.3f}")
print(f"  Recall: {evaluation.relationship_recall:.3f}")
print(f"  F1 Score: {evaluation.relationship_f1:.3f}")
```

### A/B Testing

Test different configurations:

```python
# Compare semantic vs keyword extraction
configs = [
    {"entity_method": "semantic", "threshold": 0.7},
    {"entity_method": "semantic", "threshold": 0.8}, 
    {"entity_method": "keyword", "fuzzy_match": True},
]

for i, config in enumerate(configs):
    results = run_extraction_test(config, test_documents)
    print(f"Config {i}: F1={results.f1:.3f}, Runtime={results.runtime:.2f}s")
```

## Troubleshooting

### Low Recall (Missing Entities)

```yaml
# Lower thresholds
entity_extraction:
  min_confidence: 0.5      # was 0.7
  max_entities_per_element: 20  # was 10

# Add more extraction rules
entities:
  my_entity:
    extraction_rules:
      - type: "semantic"
        phrase: "broader set of related terms"
        threshold: 0.6
      - type: "keyword"  # Add keyword fallback
        keywords: ["specific", "terms"]
      - type: "pattern"  # Add pattern matching
        patterns: ["Entity[0-9]+"]
```

### High False Positives

```yaml
# Raise thresholds
entity_extraction:
  min_confidence: 0.8      # was 0.6
  max_entities_per_element: 5   # was 20

# Add negative examples
entities:
  my_entity:
    extraction_rules:
      - type: "semantic"
        phrase: "positive terms"
        negative_phrase: "exclude these terms"  # Avoid false matches
        threshold: 0.8
```

### Slow Performance

```yaml
# Optimize batch processing
entity_extraction:
  batch_size: 1000         # Larger batches
  parallel_workers: 8      # More parallelism
  cache_embeddings: true   # Cache results

# Limit scope
relationship_detection:
  cross_document: false    # Disable cross-doc if not needed
  max_relationships_per_entity: 10  # Limit relationships
```

## Next Steps

- [Examples Repository](../examples/ontologies/) - Pre-built ontologies for common domains
- [API Reference](api.md) - Programmatic ontology management
- [Configuration Reference](configuration.md) - Complete configuration options
- [Embeddings Guide](embeddings.md) - Optimize semantic matching performance