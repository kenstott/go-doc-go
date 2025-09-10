# Knowledge Graph Construction through Domain Ontologies

## Overview

The Go-Doc-Go Knowledge Engine's domain ontology system transforms unstructured text into intelligent knowledge graphs through declarative domain definitions. This extensible framework enables automatic extraction of domain-specific entities from document elements and discovery of semantic relationships between them, allowing you to apply business knowledge and domain expertise to document analysis through configurable rules and patterns.

## Key Concepts

### Domain Ontology
A declarative, structured representation of domain knowledge that defines:
- **Terms**: Domain-specific concepts (e.g., "brake_system", "safety_requirement")
- **Mapping Rules**: Extensible patterns to identify terms in document elements
- **Relationship Rules**: Configurable patterns to discover relationships between terms

### Two-Phase Processing

1. **Entity Extraction Phase**: Maps document elements to domain terms
2. **Relationship Detection Phase**: Discovers relationships between mapped elements

## Architecture

```
Document Elements
    ↓
[Entity Extraction]
    ↓
Element-Term Mappings
    ↓
[Relationship Detection]
    ↓
Domain Relationships
```

## Declarative Configuration

### Enable Knowledge Graph Construction

The Go-Doc-Go Knowledge Engine uses declarative YAML configuration to define the knowledge extraction pipeline. In `config.yaml`:

```yaml
relationship_detection:
  domain:
    enabled: true
    ontologies:
      - path: "./ontologies/automotive.yaml"
        active: true
      - path: "./ontologies/pharmaceutical.yaml"
        active: false
    entity_extraction:
      min_confidence: 0.5      # Minimum confidence for term mappings
      batch_size: 100          # Elements processed per batch
    relationship_detection:
      min_confidence: 0.6      # Minimum confidence for relationships
```

## Extensible Ontology Definition Format

Ontologies are defined declaratively in YAML or JSON files with the following extensible structure:

### Basic Structure

```yaml
name: automotive
version: "1.0"
domain: engineering
description: "Automotive engineering domain ontology"

terms:
  - id: brake_system
    name: "Brake System"
    description: "Vehicle braking components and systems"
    aliases: ["brakes", "braking system"]
    
  - id: safety_requirement
    name: "Safety Requirement"
    description: "Safety standards and requirements"

element_mappings:
  - rules for mapping elements to terms

relationship_rules:
  - rules for discovering relationships
```

## Mapping Rules

### Rule Types

#### 1. Semantic Similarity
Matches elements based on semantic similarity using embeddings:

```yaml
element_mappings:
  - term: brake_system
    rule_type: semantic
    threshold: 0.7
    semantic_phrase: "vehicle braking system anti-lock ABS brake pad rotor"
    element_types: ["paragraph", "heading", "list_item"]
```

#### 2. Regular Expression
Matches elements using regex patterns:

```yaml
element_mappings:
  - term: diagnostic_code
    rule_type: regex
    patterns:
      - "P[0-9]{4}"           # OBD-II codes
      - "DTC[-\\s]?[0-9]{3,5}" # Diagnostic trouble codes
    case_sensitive: false
```

#### 3. Keywords
Simplified syntax for keyword matching (converted to regex internally):

```yaml
element_mappings:
  - term: safety_standard
    rule_type: keywords
    keywords:
      - "FMVSS"
      - "ISO 26262"
      - "safety critical"
    element_types: ["paragraph", "list_item"]
```

### Confidence Calculation

Each rule can specify how confidence is calculated:

```yaml
element_mappings:
  - term: component
    rule_type: semantic
    threshold: 0.6
    confidence_calculation: weighted  # Options: weighted, min, max, average
    confidence_weight: 0.8
```

## Relationship Rules

Define patterns for discovering relationships between mapped elements:

```yaml
relationship_rules:
  - id: safety_compliance
    name: "Safety Compliance"
    source_terms: ["safety_requirement"]
    target_terms: ["test_result", "brake_system"]
    relationship_type: COMPLIANCE
    
    semantic_patterns:
      - source: "requirement standard specification"
        target: "complies meets passes satisfies"
        similarity_threshold: 0.7
    
    constraints:
      hierarchy_level: -1  # Same document only
      direction: any       # source_to_target, target_to_source, any
    
    confidence_weight: 0.9
```

### Hierarchy Constraints

Control relationship scope using document hierarchy:

- `null`: No constraint (cross-document allowed)
- `-1`: Same document only
- `0`: Same parent element
- `1`: Same grandparent element
- `2`: Same great-grandparent element

## Usage Examples

### Example 1: Automotive Domain

```yaml
name: automotive
version: "1.0"
domain: engineering

terms:
  - id: brake_system
    name: "Brake System"
    aliases: ["brakes", "ABS", "anti-lock"]
    
  - id: diagnostic_code
    name: "Diagnostic Code"
    aliases: ["DTC", "fault code", "error code"]
    
  - id: safety_requirement
    name: "Safety Requirement"
    description: "Safety standards and compliance requirements"

element_mappings:
  # Semantic matching for brake systems
  - term: brake_system
    rule_type: semantic
    semantic_phrase: "brake system ABS anti-lock brake pad rotor caliper"
    threshold: 0.7
    element_types: ["paragraph", "heading", "list_item"]
    
  # Pattern matching for diagnostic codes
  - term: diagnostic_code
    rule_type: regex
    patterns:
      - "P[0-9]{4}"
      - "C[0-9]{4}"
      - "B[0-9]{4}"
      - "U[0-9]{4}"
    
  # Keyword matching for safety requirements
  - term: safety_requirement
    rule_type: keywords
    keywords:
      - "FMVSS"
      - "safety standard"
      - "compliance"
      - "requirement"

relationship_rules:
  - id: diagnosis_for_component
    source_terms: ["diagnostic_code"]
    target_terms: ["brake_system"]
    relationship_type: DIAGNOSES
    
    semantic_patterns:
      - source: "diagnostic code fault error"
        target: "brake system component"
        similarity_threshold: 0.6
    
    constraints:
      hierarchy_level: -1  # Same document
```

### Example 2: Pharmaceutical Domain

```yaml
name: pharmaceutical
version: "1.0"
domain: healthcare

terms:
  - id: drug_compound
    name: "Drug Compound"
    aliases: ["compound", "molecule", "substance"]
    
  - id: clinical_trial
    name: "Clinical Trial"
    aliases: ["trial", "study", "experiment"]
    
  - id: adverse_event
    name: "Adverse Event"
    aliases: ["side effect", "adverse reaction", "AE"]

element_mappings:
  # Match drug compounds by pattern
  - term: drug_compound
    rule_type: regex
    patterns:
      - "[A-Z]{2,}-[0-9]{3,}"  # e.g., ABC-123
      - "[A-Za-z]+mab"         # Monoclonal antibodies
    
  # Match clinical trials semantically
  - term: clinical_trial
    rule_type: semantic
    semantic_phrase: "clinical trial phase study patient randomized placebo"
    threshold: 0.65
    
  # Match adverse events by keywords
  - term: adverse_event
    rule_type: keywords
    keywords:
      - "adverse event"
      - "side effect"
      - "toxicity"
      - "reaction"

relationship_rules:
  - id: trial_tests_drug
    source_terms: ["clinical_trial"]
    target_terms: ["drug_compound"]
    relationship_type: TESTS
    
  - id: drug_causes_ae
    source_terms: ["drug_compound"]
    target_terms: ["adverse_event"]
    relationship_type: CAUSES
    constraints:
      hierarchy_level: 0  # Same section
```

## API Usage

### Python Code Example

```python
from go_doc_go import Config
from go_doc_go.domain import OntologyManager
from go_doc_go.relationships.domain import DomainRelationshipDetector

# Load configuration for the Knowledge Engine
config = Config("config.yaml")

# Initialize the extensible ontology manager
ontology_manager = config.get_ontology_manager()

# Load and activate an ontology
ontology_manager.load_ontology("./ontologies/automotive.yaml")
ontology_manager.activate_domain("automotive")

# Create domain relationship detector
detector = DomainRelationshipDetector(
    db=database,
    ontology_manager=ontology_manager,
    embedding_generator=embedder,
    config={
        "min_mapping_confidence": 0.5,
        "min_relationship_confidence": 0.6
    }
)

# Process document
relationships = detector.detect_relationships(
    document=doc_metadata,
    elements=doc_elements
)

# Get term usage statistics
report = detector.get_term_usage_report(domain="automotive")
print(f"Found {report['total_terms']} unique terms")
print(f"Total mappings: {report['total_mappings']}")
```

### Query Mapped Elements

```python
# Find all elements mapped to a specific term
elements = db.find_elements_by_term(
    term="brake_system",
    domain="automotive",
    min_confidence=0.7
)

for element_pk, element_id, confidence in elements:
    print(f"Element {element_id}: confidence {confidence:.2f}")

# Get term statistics
stats = db.get_term_statistics(domain="automotive")
for term, info in stats.items():
    print(f"{term}: {info['count']} elements, "
          f"avg confidence: {info['avg_confidence']:.2f}")
```

## Storage Schema

### Element-Term Mappings Table

```sql
CREATE TABLE element_ontology_mappings (
    mapping_id INTEGER PRIMARY KEY,
    element_pk INTEGER REFERENCES elements(element_pk),
    term TEXT NOT NULL,
    domain TEXT NOT NULL,
    confidence REAL DEFAULT 1.0,
    mapping_rule TEXT,
    created_at TIMESTAMP,
    UNIQUE(element_pk, term, domain)
);
```

### Relationships Table Extension

Domain relationships use the existing relationships table with additional metadata:

```json
{
  "source_id": "element_123",
  "target_id": "element_456",
  "relationship_type": "COMPLIANCE",
  "metadata": {
    "domain": "automotive",
    "rule_id": "safety_compliance",
    "confidence": 0.85,
    "source_term": "safety_requirement",
    "target_term": "test_result"
  }
}
```

## Performance Considerations

### Batch Processing
- Elements are processed in configurable batches (default: 100)
- Reduces memory usage for large documents

### Caching
- Ontology evaluators are cached per domain
- Embeddings are reused from the database when available

### Optimization Tips
1. **Use specific element_types** in mapping rules to reduce search space
2. **Set appropriate confidence thresholds** to balance precision and recall
3. **Use hierarchy constraints** to limit relationship search scope
4. **Pre-compute embeddings** during document ingestion for faster semantic matching

## Troubleshooting

### Common Issues

#### No Terms Mapped
- Check confidence thresholds (may be too high)
- Verify semantic phrases match your document vocabulary
- Ensure element types in rules match your documents

#### Too Many False Positives
- Increase confidence thresholds
- Make patterns more specific
- Use hierarchy constraints to limit scope

#### Slow Performance
- Reduce batch size for memory-constrained systems
- Use element_type filters in mapping rules
- Consider indexing the element_ontology_mappings table

### Debug Logging

Enable debug logging to trace the mapping process:

```python
import logging

logging.basicConfig(level=logging.DEBUG)
logger = logging.getLogger('go_doc_go.domain')
logger.setLevel(logging.DEBUG)
```

## Best Practices

### Ontology Design

1. **Start Simple**: Begin with core terms and basic rules
2. **Iterate Based on Results**: Refine rules based on actual mappings
3. **Use Multiple Rule Types**: Combine semantic, regex, and keywords for best coverage
4. **Document Rules**: Include descriptions for maintainability

### Rule Optimization

1. **Semantic Phrases**: Use representative vocabulary from your domain
2. **Regex Patterns**: Test patterns thoroughly, consider edge cases
3. **Keywords**: Use most distinctive terms to reduce false positives
4. **Confidence Weights**: Adjust based on rule reliability

### Maintenance

1. **Version Ontologies**: Track changes over time
2. **Monitor Statistics**: Regular review term usage reports
3. **Update Rules**: Refine based on new document types
4. **Test Changes**: Validate rule changes on sample documents

## Advanced Features

### Multiple Ontologies

Apply multiple ontologies simultaneously:

```yaml
relationship_detection:
  domain:
    enabled: true
    ontologies:
      - path: "./ontologies/automotive.yaml"
        active: true
      - path: "./ontologies/safety.yaml"
        active: true
      - path: "./ontologies/regulatory.yaml"
        active: true
```

### Cross-Domain Relationships

Discover relationships between terms from different ontologies:

```python
# Process with multiple active ontologies
detector.detect_cross_document_relationships(
    doc_ids=["doc1", "doc2", "doc3"]
)
```

### Custom Evaluators

Extend the system with custom matching logic:

```python
from go_doc_go.domain import OntologyEvaluator

class CustomEvaluator(OntologyEvaluator):
    def evaluate_custom_rule(self, element, rule):
        # Implement custom matching logic
        return confidence_score
```

## Integration with Knowledge Engine Pipeline

### Works With

- **Embedding-based search**: Semantic matching leverages the extensible embedding pipeline
- **Document parsing**: Applied to all parsed element types through the declarative processing pipeline
- **Relationship detection**: Extends the existing relationship system with domain intelligence
- **Cross-document analysis**: Supports document-spanning knowledge graph construction

### Complements

- **Structural relationships**: Adds semantic layer to structural analysis
- **Link extraction**: Enriches explicit links with semantic relationships
- **Metadata extraction**: Terms become searchable metadata

## Future Enhancements

Planned improvements:

1. **LLM-assisted ontology generation**: Use AI to suggest terms and rules
2. **Active learning**: Refine rules based on user feedback
3. **Visualization**: Graph visualization of term relationships
4. **Export formats**: Support for RDF, OWL, and other standards
5. **Real-time updates**: Dynamic ontology updates without restart