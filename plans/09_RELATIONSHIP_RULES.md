# Relationship Rule Schema and Extraction

## Overview

This document defines the **relationship extraction rule schema** and how relationships are identified in the target architecture.

---

## Key Principles

### 1. Synthesized Entity Filtering

**Domain relationships ONLY connect extracted entities** (entities found by extraction rules):

```go
// Domain relationships: ONLY between extracted entities
if synthesized, ok := sourceEntity.Attributes["synthesized"].(bool); ok && synthesized {
    continue  // Skip - source is synthesized
}
if synthesized, ok := targetEntity.Attributes["synthesized"].(bool); ok && synthesized {
    continue  // Skip - target is synthesized
}
```

**IS-A relationships connect ALL entities** (extracted and synthesized):

```go
// IS-A relationships: Connect all entities for hierarchical queries
isARelationship := Relationship{
    Type:       "IS-A",
    SourceID:   childEntity.ID,    // Can be extracted OR synthesized
    TargetID:   parentEntity.ID,   // Can be extracted OR synthesized
    Confidence: 1.0,
}
```

**Rationale:** Domain relationships should reflect what text actually said, not inferred hierarchy. Hierarchical queries can traverse IS-A when needed.

### 2. Relationship Ownership

Relationships are owned by the **consumer domain** (source entity's domain):

```go
// source_entity --[ENRICHED_BY]--> target_entity
rel := Relationship{
    Type:   "ENRICHED_BY",
    Domain: sourceEntity.Domain,  // Consumer domain owns enrichment
    SourceID: sourceEntity.ID,    // Entity being enriched
    TargetID: targetEntity.ID,    // Entity providing enrichment
}
```

---

## EntityRelationshipRule Schema

```go
type EntityRelationshipRule struct {
    Name               string                          // Rule name
    SourceEntityType   string                          // Source entity type
    TargetEntityType   string                          // Target entity type
    RelationshipType   RelationshipType                // Relationship type
    Description        string                          // Rule description
    Confidence         float64                         // Pattern reliability (0.0-1.0)
    ExtractionPatterns []RelationshipExtractionPattern // Patterns for extraction (OR logic)
    SourceConstraints  *EntityConstraints              // Optional source entity filters
    TargetConstraints  *EntityConstraints              // Optional target entity filters
}
```

### Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Unique rule identifier |
| `source_entity_type` | string | Yes | Source entity type name |
| `target_entity_type` | string | Yes | Target entity type name |
| `relationship_type` | RelationshipType | Yes | Relationship type constant |
| `description` | string | No | Human-readable description |
| `confidence` | float64 | Yes | Pattern reliability (0.0-1.0) |
| `extraction_patterns` | []Pattern | Yes | Extraction patterns (OR logic) |
| `source_constraints` | EntityConstraints | No | Filter source entities |
| `target_constraints` | EntityConstraints | No | Filter target entities |

---

## Extraction Pattern Types

### 1. Text Template Pattern

**Use case:** Structured text with entity placeholders

```yaml
entity_relationship_rules:
  - name: person_works_at_org
    source_entity_type: person
    target_entity_type: organization
    relationship_type: part_of
    description: Person employed by organization
    confidence: 0.85
    extraction_patterns:
      - type: text_template
        template: "{person} works at {organization}"
        direction: forward
        examples:
          - "John Smith works at Acme Corp"
          - "Jane Doe works at Tech Inc"
```

**How it works:**
1. Find source and target entities in same element
2. Extract text between entities
3. Check if text matches template (fuzzy match)
4. Create relationship if confidence threshold met

### 2. Proximity Pattern

**Use case:** Entities near each other with signal words

```yaml
entity_relationship_rules:
  - name: drug_treats_condition
    source_entity_type: drug
    target_entity_type: condition
    relationship_type: treats
    confidence: 0.80
    extraction_patterns:
      - type: proximity
        signal_words: [treats, indicated for, approved for, therapy for]
        max_distance: 50  # tokens
        direction: forward
        examples:
          - "Aspirin treats headaches"
          - "Drug X is indicated for hypertension"
```

**How it works:**
1. Find source and target entities within max_distance tokens
2. Check if signal words appear between or near entities
3. Verify direction if specified (forward/backward/bidirectional)
4. Create relationship if all conditions met

### 3. Regex Pattern

**Use case:** Complex text patterns with named groups

```yaml
entity_relationship_rules:
  - name: authored_by
    source_entity_type: document
    target_entity_type: person
    relationship_type: created_by
    confidence: 0.90
    extraction_patterns:
      - type: regex
        pattern: '(?P<document>.+?)\s+(authored|written|published)\s+by\s+(?P<person>.+)'
        examples:
          - "Paper on AI authored by Dr. Smith"
          - "Report written by Jane Doe"
```

**How it works:**
1. Apply regex pattern to element content
2. Extract entity names from named groups (`(?P<document>...)`, `(?P<person>...)`)
3. Match extracted names to existing entities
4. Create relationship if both entities found

### 4. Dependency Pattern

**Use case:** Grammatical relationships (requires NLP)

```yaml
entity_relationship_rules:
  - name: company_acquires_company
    source_entity_type: organization
    target_entity_type: organization
    relationship_type: acquires
    confidence: 0.75
    extraction_patterns:
      - type: dependency
        pattern: "nsubj(acquire, {source}) && dobj(acquire, {target})"
        examples:
          - "Microsoft acquired GitHub"  # nsubj(acquired, Microsoft), dobj(acquired, GitHub)
```

**How it works:**
1. Parse element with dependency parser (spaCy, Stanford)
2. Match dependency tree against pattern
3. Extract entities from matched dependencies
4. Create relationship if pattern satisfied

**Note:** Requires NLP library integration (not implemented in basic extractor).

### 5. Co-occurrence Pattern

**Use case:** Statistical co-occurrence within element scope

```yaml
entity_relationship_rules:
  - name: technology_related_to_concept
    source_entity_type: technology
    target_entity_type: concept
    relationship_type: related_to
    confidence: 0.60
    extraction_patterns:
      - type: cooccurrence
        max_distance: 0  # Same element
        min_occurrences: 3  # Must co-occur in 3+ elements
        examples:
          - "Machine learning and neural networks"  # Must appear together frequently
```

**How it works:**
1. Count co-occurrences of entity pairs across all elements
2. If count >= min_occurrences, create relationship
3. Confidence can be adjusted based on frequency

---

## Relationship Rule Inheritance

### Overview

Relationship rules can **extend parent rules** to inherit extraction patterns and constraints, enabling progressive refinement from general to domain-specific relationships.

### Schema Addition

```go
type EntityRelationshipRule struct {
    Name               string
    ParentRelationship string  // NEW: Reference to parent rule (e.g., "global.person_works_at_organization")

    SourceEntityType   string  // Override parent's source type (must be subtype)
    TargetEntityType   string  // Override parent's target type (must be subtype)
    RelationshipType   RelationshipType
    Description        string
    Confidence         float64  // Can increase confidence (more specific = higher)

    ExtractionPatterns []RelationshipExtractionPattern  // APPENDED to parent patterns (OR logic)
    SourceConstraints  *EntityConstraints  // ADDED to parent constraints (AND logic)
    TargetConstraints  *EntityConstraints  // ADDED to parent constraints (AND logic)
}
```

### Inheritance Rules

1. **Source/Target Types**: Child types MUST be subtypes of parent types
2. **Extraction Patterns**: Child patterns are **APPENDED** to parent patterns (OR logic)
3. **Entity Constraints**: Child constraints are **ADDED** to parent constraints (AND logic)
4. **Confidence**: Child can increase but NOT decrease parent confidence
5. **Relationship Type**: Inherited from parent (cannot override)

### Example 1: Basic Inheritance

**Global catalog** (parent rule):
```yaml
entity_relationship_rules:
  - name: person_works_at_organization
    source_entity_type: person
    target_entity_type: organization
    relationship_type: part_of
    description: Person employed by organization
    confidence: 0.70
    extraction_patterns:
      - type: proximity
        signal_words: [works at, employed by, employee of]
        max_distance: 50
```

**Medical domain catalog** (child rule):
```yaml
entity_relationship_rules:
  - name: physician_works_at_hospital
    parent_relationship: global.person_works_at_organization  # INHERIT FROM PARENT

    # Override types (more specific)
    source_entity_type: physician  # Subtype of person
    target_entity_type: hospital   # Subtype of organization

    # Increase confidence (more specific = higher confidence)
    confidence: 0.85

    # Add domain-specific extraction patterns (APPENDED to parent)
    extraction_patterns:
      - type: proximity
        signal_words: [practicing at, on staff at, attending physician at]
        max_distance: 50
      - type: regex
        pattern: '(?P<physician>Dr\. .+?)\s+at\s+(?P<hospital>.+?\s+(Hospital|Medical Center))'

    # Add domain-specific constraints (ADDED to parent)
    source_constraints:
      proximity:
        keywords: [MD, physician, doctor]
        max_distance: 100
```

**Resolved rule** (after inheritance):
```yaml
# Effective rule after parent merge
entity_relationship_rules:
  - name: physician_works_at_hospital
    source_entity_type: physician
    target_entity_type: hospital
    relationship_type: part_of
    confidence: 0.85
    extraction_patterns:
      # Inherited from parent:
      - type: proximity
        signal_words: [works at, employed by, employee of]
        max_distance: 50
      # Added by child:
      - type: proximity
        signal_words: [practicing at, on staff at, attending physician at]
        max_distance: 50
      - type: regex
        pattern: '(?P<physician>Dr\. .+?)\s+at\s+(?P<hospital>.+?\s+(Hospital|Medical Center))'
    source_constraints:
      proximity:
        keywords: [MD, physician, doctor]
        max_distance: 100
```

### Example 2: Progressive Refinement Chain

**Level 1: Global** (most general):
```yaml
entity_relationship_rules:
  - name: person_affiliated_with_organization
    source_entity_type: person
    target_entity_type: organization
    relationship_type: related_to
    confidence: 0.60
    extraction_patterns:
      - type: proximity
        signal_words: [affiliated with, associated with]
        max_distance: 100
```

**Level 2: Medical domain** (intermediate):
```yaml
entity_relationship_rules:
  - name: medical_professional_works_at_healthcare_org
    parent_relationship: global.person_affiliated_with_organization
    source_entity_type: medical_professional  # Specializes person
    target_entity_type: healthcare_organization  # Specializes organization
    relationship_type: part_of  # More specific than related_to
    confidence: 0.75
    extraction_patterns:
      - type: proximity
        signal_words: [works at, employed by, practices at]
        max_distance: 75
```

**Level 3: Medical subdomain** (most specific):
```yaml
entity_relationship_rules:
  - name: surgeon_operates_at_hospital
    parent_relationship: medical.medical_professional_works_at_healthcare_org
    source_entity_type: surgeon  # Specializes medical_professional
    target_entity_type: hospital  # Specializes healthcare_organization
    confidence: 0.90
    extraction_patterns:
      - type: proximity
        signal_words: [operates at, surgical privileges at, operating room at]
        max_distance: 50
    source_constraints:
      semantic:
        reference_concepts: [surgeon, surgical specialist]
        similarity_threshold: 0.80
```

**Final resolved rule has**:
- **6 extraction patterns** (2 from each level)
- **Source constraint** (surgeon-specific from level 3)
- **Confidence 0.90** (highest specificity)
- **Highly specific types**: surgeon → hospital

### Example 3: Sibling Rules (Multiple Children)

**Parent**:
```yaml
entity_relationship_rules:
  - name: person_works_at_organization
    source_entity_type: person
    target_entity_type: organization
    relationship_type: part_of
    confidence: 0.70
    extraction_patterns:
      - type: proximity
        signal_words: [works at, employed by]
        max_distance: 50
```

**Child 1** (medical):
```yaml
  - name: physician_works_at_hospital
    parent_relationship: global.person_works_at_organization
    source_entity_type: physician
    target_entity_type: hospital
    confidence: 0.85
    extraction_patterns:
      - type: proximity
        signal_words: [practicing at, on staff at]
        max_distance: 50
```

**Child 2** (legal):
```yaml
  - name: attorney_works_at_law_firm
    parent_relationship: global.person_works_at_organization
    source_entity_type: attorney
    target_entity_type: law_firm
    confidence: 0.85
    extraction_patterns:
      - type: proximity
        signal_words: [partner at, associate at, counsel at]
        max_distance: 50
```

**Child 3** (academic):
```yaml
  - name: professor_works_at_university
    parent_relationship: global.person_works_at_organization
    source_entity_type: professor
    target_entity_type: university
    confidence: 0.90
    extraction_patterns:
      - type: proximity
        signal_words: [teaches at, faculty at, professor at]
        max_distance: 50
```

Each child inherits base patterns `[works at, employed by]` and adds domain-specific vocabulary.

### Benefits

1. **DRY Principle**: Define common patterns once in global catalog
2. **Progressive Specificity**: Each level adds domain knowledge
3. **Confidence Scaling**: More specific rules = higher confidence (enforced)
4. **Vocabulary Extension**: Domain catalogs add their terminology
5. **Constraint Layering**: Each level can add more constraints
6. **Maintenance**: Update parent rule → all children automatically updated

### Implementation Algorithm

```go
// Resolve relationship rule inheritance at schema load time
func (schema *OntologySchema) ResolveRelationshipInheritance() error {
    // Build rule index across all schemas (global + domain)
    ruleIndex := make(map[string]*EntityRelationshipRule)
    for _, s := range allSchemas {
        for i := range s.EntityRelationshipRules {
            rule := &s.EntityRelationshipRules[i]
            qualifiedName := s.Domain + "." + rule.Name
            ruleIndex[qualifiedName] = rule
        }
    }

    // Resolve each rule with parent (depth-first)
    for i := range schema.EntityRelationshipRules {
        rule := &schema.EntityRelationshipRules[i]

        if rule.ParentRelationship != "" {
            if err := resolveParentRule(rule, ruleIndex); err != nil {
                return err
            }
        }
    }

    return nil
}

func resolveParentRule(rule *EntityRelationshipRule, index map[string]*EntityRelationshipRule) error {
    parent := index[rule.ParentRelationship]
    if parent == nil {
        return fmt.Errorf("rule '%s' references non-existent parent: %s",
            rule.Name, rule.ParentRelationship)
    }

    // Recursively resolve parent first (handle chains)
    if parent.ParentRelationship != "" {
        if err := resolveParentRule(parent, index); err != nil {
            return err
        }
    }

    // Validate inheritance constraints
    if err := validateRelationshipInheritance(rule, parent); err != nil {
        return err
    }

    // Merge parent patterns (prepend parent patterns)
    rule.ExtractionPatterns = append(
        copyPatterns(parent.ExtractionPatterns),
        rule.ExtractionPatterns...,
    )

    // Merge parent constraints (AND logic)
    if parent.SourceConstraints != nil {
        if rule.SourceConstraints == nil {
            rule.SourceConstraints = &EntityConstraints{}
        }
        mergeConstraints(rule.SourceConstraints, parent.SourceConstraints)
    }
    if parent.TargetConstraints != nil {
        if rule.TargetConstraints == nil {
            rule.TargetConstraints = &EntityConstraints{}
        }
        mergeConstraints(rule.TargetConstraints, parent.TargetConstraints)
    }

    // Inherit relationship_type if not specified
    if rule.RelationshipType == "" {
        rule.RelationshipType = parent.RelationshipType
    }

    // Inherit description if not specified
    if rule.Description == "" {
        rule.Description = parent.Description
    }

    return nil
}
```

### Validation Rules

```go
func validateRelationshipInheritance(child, parent *EntityRelationshipRule) error {
    // 1. Check for circular references
    if child.Name == parent.ParentRelationship {
        return fmt.Errorf("circular inheritance detected: %s <-> %s",
            child.Name, parent.Name)
    }

    // 2. Source type must be subtype of parent source type
    if !isSubtypeOf(child.SourceEntityType, parent.SourceEntityType) {
        return fmt.Errorf("child source type '%s' must be subtype of parent source type '%s'",
            child.SourceEntityType, parent.SourceEntityType)
    }

    // 3. Target type must be subtype of parent target type
    if !isSubtypeOf(child.TargetEntityType, parent.TargetEntityType) {
        return fmt.Errorf("child target type '%s' must be subtype of parent target type '%s'",
            child.TargetEntityType, parent.TargetEntityType)
    }

    // 4. Confidence must not decrease
    if child.Confidence < parent.Confidence {
        return fmt.Errorf("child confidence (%.2f) cannot be lower than parent (%.2f)",
            child.Confidence, parent.Confidence)
    }

    // 5. Relationship type must match parent (if both specified)
    if parent.RelationshipType != "" && child.RelationshipType != "" {
        if child.RelationshipType != parent.RelationshipType {
            return fmt.Errorf("child relationship_type must match parent relationship_type '%s'",
                parent.RelationshipType)
        }
    }

    return nil
}

func isSubtypeOf(childType, parentType string) bool {
    // Check if childType is in the hierarchy chain of parentType
    // Example: physician -> person -> entity
    // isSubtypeOf("physician", "person") = true
    // isSubtypeOf("person", "physician") = false
    // Implementation uses entity hierarchy from schema
    return entityHierarchy.IsDescendantOf(childType, parentType)
}
```

### Usage Guidelines

**When to use inheritance**:
- ✅ Domain-specific entity types that need same relationship patterns as parent
- ✅ Need to add domain-specific signal words to common relationships
- ✅ Want to increase confidence for more specific contexts
- ✅ Adding constraints to filter false positives

**When NOT to use inheritance**:
- ❌ Relationship is completely different from parent
- ❌ Need to remove parent patterns (not supported - create new rule instead)
- ❌ Relationship type differs from parent

### Migration Impact

**Purely additive** - No breaking changes:
- Existing rules without `parent_relationship` work unchanged
- Domain catalogs can optionally adopt inheritance
- Global catalogs define reusable base rules
- Gradual migration possible (rule by rule)

---

## Entity Constraints

Filter which entities qualify as source/target using the same filter architecture as extraction rules:

```yaml
entity_relationship_rules:
  - name: senior_physician_mentors_resident
    source_entity_type: physician
    target_entity_type: physician
    relationship_type: mentors
    confidence: 0.80

    # Only senior physicians as source
    source_constraints:
      instance_name: '(?P<name>Dr\. .+, (MD|PhD))'
      semantic:
        reference_concepts: [senior physician, attending doctor]
        similarity_threshold: 0.70

    # Only residents as target
    target_constraints:
      instance_name: '(?P<name>Dr\. .+, MD)'
      proximity:
        keywords: [resident, intern, trainee]
        max_distance: 50

    extraction_patterns:
      - type: proximity
        signal_words: [mentors, supervises, trains]
        max_distance: 100
```

**EntityConstraints fields:**
- `instance_name`: Regex with named capture - entity name must match
- `pattern`: Pre-filter regex for entity name
- `proximity_filter`: Co-occurrence filter on entity context
- `semantic_filter`: Embedding similarity on entity context

---

## Complete Example: Medical Domain

```yaml
name: Medical Domain
version: 2.0
domains:
  - name: medical
    description: Healthcare and medical domain

element_entity_mappings:
  - entity_type: physician
    domain: medical
    parent_type: global.person
    w_category: who
    description: Medical doctor
    confidence: 0.90
    extraction_rules:
      - instance_name: (?P<name>Dr\. [A-Z][a-z]+ [A-Z][a-z]+)
        proximity:
          keywords: [patient, diagnosis]
          max_distance: 100

  - entity_type: condition
    domain: medical
    w_category: what
    description: Medical condition or disease
    confidence: 0.85
    extraction_rules:
      - instance_name: (?P<name>[A-Z][a-z]+(itis|osis|pathy|ia))
        semantic:
          reference_concepts: [disease, illness, condition]
          similarity_threshold: 0.70

entity_relationship_rules:
  # Relationship 1: Proximity-based
  - name: physician_treats_condition
    source_entity_type: physician
    target_entity_type: condition
    relationship_type: treats
    description: Physician treats medical condition
    confidence: 0.85
    extraction_patterns:
      - type: proximity
        signal_words: [treats, diagnosed, manages, treating]
        max_distance: 100
        direction: forward
      - type: text_template
        template: "{physician} specializes in {condition}"
        direction: forward

  # Relationship 2: Regex-based
  - name: condition_caused_by_exposure
    source_entity_type: condition
    target_entity_type: substance
    relationship_type: caused_by
    confidence: 0.80
    extraction_patterns:
      - type: regex
        pattern: '(?P<condition>\w+itis|asthma)\s+(caused by|due to|from)\s+(?P<substance>.+)'

  # Relationship 3: Co-occurrence (low confidence)
  - name: condition_related_to_condition
    source_entity_type: condition
    target_entity_type: condition
    relationship_type: related_to
    description: Conditions that frequently co-occur
    confidence: 0.65
    extraction_patterns:
      - type: cooccurrence
        max_distance: 0  # Same paragraph
        min_occurrences: 5
```

---

## Old vs New Format Migration

### Old Format (Deprecated)

```yaml
relationship_types:
  - relationship_type: treats_indication
    description: Drug treats indication
    source_entity: drug_product
    target_entity: indication
    sample_rules:
      - type: keyword_match
        keywords: [treats, indicated for, approved for]
```

**Problems:**
- Simple keyword matching only
- No pattern diversity (proximity, regex, etc.)
- No entity constraints
- No confidence scores
- Field naming inconsistent with entity extraction

### New Format (Target)

```yaml
entity_relationship_rules:
  - name: drug_treats_indication
    source_entity_type: drug_product
    target_entity_type: indication
    relationship_type: treats
    description: Drug treats indication
    confidence: 0.80
    extraction_patterns:
      - type: proximity
        signal_words: [treats, indicated for, approved for, therapy for]
        max_distance: 50
        direction: forward
      - type: text_template
        template: "{drug_product} is approved for {indication}"
```

**Benefits:**
- Multiple pattern types (proximity, regex, template, dependency, cooccurrence)
- Entity constraints for filtering
- Confidence scores
- Consistent with entity extraction rule structure

---

## Validation Rules

### Schema Validation

```go
func (rule *EntityRelationshipRule) Validate(schema *OntologySchema) error {
    // 1. Check required fields
    if rule.Name == "" {
        return fmt.Errorf("relationship rule missing name")
    }
    if rule.SourceEntityType == "" {
        return fmt.Errorf("relationship rule '%s' missing source_entity_type", rule.Name)
    }
    if rule.TargetEntityType == "" {
        return fmt.Errorf("relationship rule '%s' missing target_entity_type", rule.Name)
    }
    if rule.Confidence <= 0.0 || rule.Confidence > 1.0 {
        return fmt.Errorf("relationship rule '%s' confidence must be in (0.0, 1.0]", rule.Name)
    }

    // 2. Check entity types exist
    sourceExists := false
    targetExists := false
    for _, mapping := range schema.ElementEntityMappings {
        if mapping.EntityType == rule.SourceEntityType {
            sourceExists = true
        }
        if mapping.EntityType == rule.TargetEntityType {
            targetExists = true
        }
    }
    if !sourceExists {
        return fmt.Errorf("relationship rule '%s' references non-existent source entity type: %s",
            rule.Name, rule.SourceEntityType)
    }
    if !targetExists {
        return fmt.Errorf("relationship rule '%s' references non-existent target entity type: %s",
            rule.Name, rule.TargetEntityType)
    }

    // 3. Check at least one extraction pattern
    if len(rule.ExtractionPatterns) == 0 {
        return fmt.Errorf("relationship rule '%s' must have at least one extraction pattern", rule.Name)
    }

    // 4. Validate each pattern
    for i, pattern := range rule.ExtractionPatterns {
        if err := validateExtractionPattern(pattern); err != nil {
            return fmt.Errorf("relationship rule '%s' pattern %d invalid: %w", rule.Name, i, err)
        }
    }

    return nil
}
```

---

## Extraction Algorithm

### High-Level Flow

```go
func (e *RuleBasedExtractor) extractRelationships(
    ctx context.Context,
    elements []Element,
    entities []Entity,
) ([]Relationship, error) {
    // STEP 1: Filter to only extracted entities (synthesized=false)
    extractedEntities := filterExtractedEntities(entities)

    // STEP 2: Build entity index by type
    entityIndex := buildEntityIndex(extractedEntities)

    // STEP 3: Apply each relationship rule
    var relationships []Relationship
    for _, rule := range e.schema.EntityRelationshipRules {
        rels := e.applyRelationshipRule(rule, entityIndex, elements)
        relationships = append(relationships, rels...)
    }

    return relationships, nil
}

func filterExtractedEntities(entities []Entity) []Entity {
    var extracted []Entity
    for _, entity := range entities {
        // Only include entities that were extracted by rules (not synthesized)
        if synthesized, ok := entity.Attributes["synthesized"].(bool); !ok || !synthesized {
            extracted = append(extracted, entity)
        }
    }
    return extracted
}
```

### Pattern-Specific Extraction

```go
func (e *RuleBasedExtractor) applyRelationshipRule(
    rule EntityRelationshipRule,
    entityIndex map[string][]Entity,
    elements []Element,
) []Relationship {
    var relationships []Relationship

    // Get source and target entities
    sourceEntities := entityIndex[rule.SourceEntityType]
    targetEntities := entityIndex[rule.TargetEntityType]

    // Apply entity constraints (filter)
    if rule.SourceConstraints != nil {
        sourceEntities = filterByConstraints(sourceEntities, rule.SourceConstraints)
    }
    if rule.TargetConstraints != nil {
        targetEntities = filterByConstraints(targetEntities, rule.TargetConstraints)
    }

    // For each extraction pattern (OR logic)
    for _, pattern := range rule.ExtractionPatterns {
        switch pattern.Type {
        case RelPatternTextTemplate:
            rels := e.applyTextTemplatePattern(pattern, sourceEntities, targetEntities, elements)
            relationships = append(relationships, rels...)

        case RelPatternProximity:
            rels := e.applyProximityPattern(pattern, sourceEntities, targetEntities, elements)
            relationships = append(relationships, rels...)

        case RelPatternRegex:
            rels := e.applyRegexPattern(pattern, sourceEntities, targetEntities, elements)
            relationships = append(relationships, rels...)

        case RelPatternDependency:
            rels := e.applyDependencyPattern(pattern, sourceEntities, targetEntities, elements)
            relationships = append(relationships, rels...)

        case RelPatternCooccurrence:
            rels := e.applyCooccurrencePattern(pattern, sourceEntities, targetEntities, elements)
            relationships = append(relationships, rels...)
        }
    }

    // Deduplicate relationships
    return deduplicateRelationships(relationships)
}
```

---

## Domain Catalog Examples

### Pharmaceutical Domain

```yaml
name: Pharmaceutical Domain
version: 2.0
domains:
  - name: pharmaceutical
    description: Drug development and commercialization

entity_relationship_rules:
  - name: drug_treats_indication
    source_entity_type: drug_product
    target_entity_type: indication
    relationship_type: treats
    confidence: 0.85
    extraction_patterns:
      - type: proximity
        signal_words: [treats, indicated for, approved for, therapy for]
        max_distance: 50
        direction: forward

  - name: drug_causes_adverse_event
    source_entity_type: drug_product
    target_entity_type: adverse_event
    relationship_type: causes
    confidence: 0.75
    extraction_patterns:
      - type: proximity
        signal_words: [causes, associated with, side effect, adverse event]
        max_distance: 100
        direction: forward
      - type: regex
        pattern: '(?P<drug>.+?)\s+(may cause|can cause|causes)\s+(?P<event>.+)'

  - name: trial_tests_drug
    source_entity_type: clinical_trial
    target_entity_type: drug_product
    relationship_type: tests
    confidence: 0.90
    extraction_patterns:
      - type: text_template
        template: "{clinical_trial} evaluated {drug_product}"
      - type: proximity
        signal_words: [tested, evaluated, studied, investigating]
        max_distance: 75
```

### Legal Domain

```yaml
name: Legal Domain
version: 2.0
domains:
  - name: legal
    description: Legal documents and case law

entity_relationship_rules:
  - name: case_cites_precedent
    source_entity_type: case
    target_entity_type: precedent
    relationship_type: cites
    confidence: 0.95
    extraction_patterns:
      - type: regex
        pattern: '(?P<case>.+?)\s+(cites|citing|relied on|following)\s+(?P<precedent>.+)'

  - name: attorney_represents_party
    source_entity_type: attorney
    target_entity_type: party
    relationship_type: represents
    confidence: 0.90
    extraction_patterns:
      - type: text_template
        template: "{attorney} represents {party}"
      - type: proximity
        signal_words: [represents, counsel for, attorney for]
        max_distance: 50
```

---

## Testing Strategy

### Unit Tests

```go
func TestRelationshipExtraction_ProximityPattern(t *testing.T) {
    schema := &OntologySchema{
        EntityRelationshipRules: []EntityRelationshipRule{
            {
                Name:             "physician_treats_condition",
                SourceEntityType: "physician",
                TargetEntityType: "condition",
                RelationshipType: RelationshipTypeTreats,
                Confidence:       0.85,
                ExtractionPatterns: []RelationshipExtractionPattern{
                    {
                        Type:        RelPatternProximity,
                        SignalWords: []string{"treats", "diagnosed", "managing"},
                        MaxDistance: 100,
                        Direction:   "forward",
                    },
                },
            },
        },
    }

    entities := []Entity{
        {
            ID:   "ent1",
            Name: "Dr. Smith",
            Type: EntityTypePerson,
            Attributes: map[string]interface{}{
                "synthesized": false,  // EXTRACTED
            },
            ElementID: "elem1",
        },
        {
            ID:   "ent2",
            Name: "pneumonia",
            Type: EntityTypeCustom,
            Attributes: map[string]interface{}{
                "synthesized": false,  // EXTRACTED
            },
            ElementID: "elem1",
        },
    }

    elements := []Element{
        {
            ElementID: "elem1",
            Content:   "Dr. Smith treats pneumonia patients.",
        },
    }

    extractor := NewRuleBasedExtractor(schema, nil)
    relationships, err := extractor.extractRelationships(context.Background(), elements, entities)

    require.NoError(t, err)
    require.Len(t, relationships, 1)
    assert.Equal(t, "ent1", relationships[0].SourceID)
    assert.Equal(t, "ent2", relationships[0].TargetID)
    assert.Equal(t, RelationshipTypeTreats, relationships[0].Type)
}

func TestRelationshipExtraction_SynthesizedEntityFiltering(t *testing.T) {
    // Test that synthesized entities are excluded from domain relationships
    entities := []Entity{
        {
            ID:   "ent1",
            Name: "Dr. Smith",
            Attributes: map[string]interface{}{
                "synthesized": false,  // EXTRACTED
            },
        },
        {
            ID:   "ent2",
            Name: "Hospital",
            Attributes: map[string]interface{}{
                "synthesized": true,  // SYNTHESIZED
            },
        },
    }

    // Should NOT create relationship because ent2 is synthesized
    // ...
}
```

---

## Next Steps

See related documents:
- [06_IMPLEMENTATION_STEPS.md](06_IMPLEMENTATION_STEPS.md) - Implementation details
- [08_MIGRATION_GUIDE.md](08_MIGRATION_GUIDE.md) - Migration from old format
- [07_TESTING_STRATEGY.md](07_TESTING_STRATEGY.md) - Testing approach
