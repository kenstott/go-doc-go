# Hierarchy System: Computation & Materialization

## Overview

The hierarchy system has two phases:
1. **Bidirectional Computation** (runtime) - Auto-fill missing `parent_type` and `children` relationships
2. **Hierarchy Materialization** (post-extraction) - Create composite parent entities and IS-A relationships

---

## Phase 1: Bidirectional Computation

### Purpose

Resolve circular dependency between global domain and domain catalogs:
- Global domain defines `person` but doesn't know about `medical.physician`
- Medical catalog defines `physician` with `parent_type: global.person`
- At runtime, auto-fill `person.children = [medical.physician]`

### Algorithm

```go
func (schema *OntologySchema) ComputeHierarchies() error {
    // Build entity map (first occurrence per qualified name)
    entityMap := make(map[string]*ElementEntityMapping)
    for i := range schema.ElementEntityMappings {
        mapping := &schema.ElementEntityMappings[i]
        key := mapping.Domain + "." + mapping.EntityType

        if _, exists := entityMap[key]; !exists {
            entityMap[key] = mapping
        }
    }

    // Phase 1: Fill parent's children from child's parent_type
    for i := range schema.ElementEntityMappings {
        mapping := &schema.ElementEntityMappings[i]

        if mapping.ParentType != "" {
            parent := entityMap[mapping.ParentType]
            if parent != nil {
                childRef := mapping.Domain + "." + mapping.EntityType
                if !contains(parent.Children, childRef) {
                    parent.Children = append(parent.Children, childRef)
                }
            }
        }
    }

    // Phase 2: Fill child's parent_type from parent's children
    for i := range schema.ElementEntityMappings {
        mapping := &schema.ElementEntityMappings[i]

        for _, childRef := range mapping.Children {
            child := entityMap[childRef]
            if child != nil && child.ParentType == "" {
                child.ParentType = mapping.Domain + "." + mapping.EntityType
            }
        }
    }

    return nil
}
```

### Example

**Before Computation**:
```yaml
# Global domain (global_who.go)
- entity_type: person
  domain: global
  parent_type: ""
  children: []        # Empty - doesn't know about medical.physician

# Medical catalog (medical.yaml)
- entity_type: physician
  domain: medical
  parent_type: global.person  # Declares parent
  children: []
```

**After Computation** (runtime):
```yaml
# Global domain (modified at runtime)
- entity_type: person
  domain: global
  parent_type: ""
  children: [medical.physician, legal.attorney, ...]  # Auto-filled!

# Medical catalog (unchanged)
- entity_type: physician
  domain: medical
  parent_type: global.person
  children: []
```

---

## Phase 2: Canonicalization and Hierarchy Materialization

### Purpose

After extracting entities (with duplicates from multiple definitions), create a **canonical-only graph**:
1. **Canonicalize** extracted entities (deduplicate, merge mentions, keep highest confidence)
2. **Materialize missing ancestors** by walking up hierarchy from canonical entities
3. Create **IS-A relationships** only between canonical entities
4. Enable querying at any hierarchy level

### When It Runs

```go
func (e *RuleBasedExtractor) ExtractFromElements(ctx context.Context, docID string, elements []Element) (*Ontology, error) {
    // STEP 1: Extract entities (with duplicates from multiple definitions)
    extractedEntities, err := e.extractEntities(ctx, elements)

    // STEP 2: Canonicalize (deduplicate, merge mentions, keep highest confidence)
    canonicalLeaves, err := e.canonicalizeEntities(extractedEntities)

    // STEP 3: Materialize missing ancestors (walk up, create composites as needed)
    compositeAncestors, isaRelationships, err := e.MaterializeMissingAncestors(canonicalLeaves, e.schema)

    // STEP 4: Combine canonical leaves + canonical composites
    allCanonicalEntities := append(canonicalLeaves, compositeAncestors...)

    // STEP 5: Extract domain relationships (between canonical entities only)
    domainRelationships, err := e.extractRelationships(ctx, elements, allCanonicalEntities)

    // STEP 6: Combine IS-A + domain relationships
    allRelationships := append(isaRelationships, domainRelationships...)

    return ontology, nil
}
```

### Canonicalization Algorithm

**Step 2A: Deduplicate extracted entities**

```go
func (e *RuleBasedExtractor) canonicalizeEntities(extractedEntities []Entity) ([]Entity, error) {
    // Canonicalization key: name|type|element_id
    canonicalIndex := make(map[string]*Entity)

    for _, entity := range extractedEntities {
        key := entity.Name + "|" + string(entity.Type) + "|" + entity.ElementID

        if canonical, exists := canonicalIndex[key]; exists {
            // Merge: Keep highest confidence, combine mentions
            if entity.Confidence > canonical.Confidence {
                canonical.Confidence = entity.Confidence
            }
            // Merge mentions from both entities
            canonical.Mentions = append(canonical.Mentions, entity.Mentions...)
        } else {
            // First occurrence - make canonical
            canonicalIndex[key] = &entity
        }
    }

    // Convert map to slice
    var canonicalEntities []Entity
    for _, entity := range canonicalIndex {
        canonicalEntities = append(canonicalEntities, *entity)
    }

    return canonicalEntities, nil
}
```

### Materialization Algorithm

**Step 2B: Walk up from canonical entities, materialize missing ancestors**

```go
func (e *RuleBasedExtractor) MaterializeMissingAncestors(canonicalLeaves []Entity, schema *OntologySchema) ([]Entity, []Relationship, error) {
    var compositeAncestors []Entity
    var isaRelationships []Relationship

    // Build entity type map
    entityTypeMap := make(map[string]*ElementEntityMapping)
    for i := range schema.ElementEntityMappings {
        mapping := &schema.ElementEntityMappings[i]
        key := mapping.Domain + "." + mapping.EntityType
        if _, exists := entityTypeMap[key]; !exists {
            entityTypeMap[key] = mapping
        }
    }

    // Canonical index for composites (prevent duplicates during materialization)
    // Key: name|type|element_id
    canonicalIndex := make(map[string]*Entity)

    // For each canonical leaf entity, walk up parent chain
    for _, leafEntity := range canonicalLeaves {
        currentEntity := leafEntity
        currentKey := leafEntity.Domain + "." + string(leafEntity.Type)

        // Walk up hierarchy
        for {
            mapping := entityTypeMap[currentKey]
            if mapping == nil || mapping.ParentType == "" {
                break // Reached top or orphan
            }

            parentMapping := entityTypeMap[mapping.ParentType]
            if parentMapping == nil {
                break // Invalid parent
            }

            // Check if parent already exists in canonical index
            canonicalKey := currentEntity.Name + "|" + parentMapping.EntityType + "|" + currentEntity.ElementID

            var parentEntity *Entity
            if cached, exists := canonicalIndex[canonicalKey]; exists {
                // Reuse existing canonical composite
                parentEntity = cached
            } else {
                // Materialize new canonical composite (synthesized entity)
                parentEntity = &Entity{
                    ID:         e.generateID("ent"),
                    Name:       currentEntity.Name,  // Inherit name from child
                    Type:       EntityType(parentMapping.EntityType),
                    Domain:     parentMapping.Domain,
                    Attributes: map[string]interface{}{
                        "synthesized": true,  // Created during materialization (not extracted)
                        "w_category":  parentMapping.WCategory,
                    },
                    ElementID:  currentEntity.ElementID,
                    Mentions:   currentEntity.Mentions,  // Inherit mentions
                    CreatedAt:  time.Now(),
                    UpdatedAt:  time.Now(),
                }
                compositeAncestors = append(compositeAncestors, *parentEntity)
                canonicalIndex[canonicalKey] = parentEntity
            }

            // Create IS-A relationship: child IS-A parent (only between canonical entities)
            isaRelationships = append(isaRelationships, Relationship{
                ID:         e.generateID("rel"),
                Type:       RelationshipIsA,
                Domain:     currentEntity.Domain,
                SourceID:   currentEntity.ID,
                TargetID:   parentEntity.ID,
                Confidence: 1.0,
                Attributes: map[string]interface{}{
                    "hierarchy_level": "direct_parent",
                },
                ElementID: currentEntity.ElementID,
                Evidence:  fmt.Sprintf("%s is a %s", currentEntity.Type, parentEntity.Type),
                CreatedAt: time.Now(),
            })

            // Move up hierarchy
            currentEntity = *parentEntity
            currentKey = mapping.ParentType
        }
    }

    return compositeAncestors, isaRelationships, nil
}
```

### Example: Canonical-Only Graph

**Step 1: Extract entities (with duplicates)**:
```
Extracted (3 definitions of surgeon, 2 of patient):
  - surgeon: "Dr. Smith" (element: doc1_para5, confidence: 0.90, rule: high_confidence_surgeon)
  - surgeon: "Dr. Smith" (element: doc1_para5, confidence: 0.80, rule: generic_surgeon)
  - surgeon: "Dr. Smith" (element: doc1_para5, confidence: 0.85, rule: context_aware_surgeon)
  - patient: "John Doe" (element: doc1_para5, confidence: 0.75, rule: generic_patient)
  - patient: "John Doe" (element: doc1_para5, confidence: 0.70, rule: keyword_patient)
  - cardiologist: "Dr. Jones" (element: doc1_para10, confidence: 0.90, rule: cardiologist_rule)
```

**Step 2A: Canonicalize (deduplicate)**:
```
Canonical leaves (duplicates merged):
  - surgeon: "Dr. Smith" (element: doc1_para5, confidence: 0.90)
    - Mentions: [rule: high_confidence_surgeon, rule: generic_surgeon, rule: context_aware_surgeon]
  - patient: "John Doe" (element: doc1_para5, confidence: 0.75)
    - Mentions: [rule: generic_patient, rule: keyword_patient]
  - cardiologist: "Dr. Jones" (element: doc1_para10, confidence: 0.90)
    - Mentions: [rule: cardiologist_rule]
```

**Hierarchy** (from schema):
```
global.person
  └─ medical.physician
       ├─ medical.surgeon
       └─ medical.cardiologist

global.person
  └─ medical.patient
```

**Step 2B: Materialize missing ancestors**:
```
Synthesized ancestors (canonical, generated):
  - physician: "Dr. Smith" (synthesized=true, element: doc1_para5)
  - physician: "Dr. Jones" (synthesized=true, element: doc1_para10)
  - person: "Dr. Smith" (synthesized=true, element: doc1_para5)
  - person: "Dr. Jones" (synthesized=true, element: doc1_para10)
  - person: "John Doe" (synthesized=true, element: doc1_para5)
```

**Final canonical graph (ALL entities are canonical)**:
```
Entities:
  - surgeon: "Dr. Smith" (extracted, canonical, synthesized=false)
  - cardiologist: "Dr. Jones" (extracted, canonical, synthesized=false)
  - patient: "John Doe" (extracted, canonical, synthesized=false)
  - physician: "Dr. Smith" (synthesized, canonical, synthesized=true)
  - physician: "Dr. Jones" (synthesized, canonical, synthesized=true)
  - person: "Dr. Smith" (synthesized, canonical, synthesized=true)
  - person: "Dr. Jones" (synthesized, canonical, synthesized=true)
  - person: "John Doe" (synthesized, canonical, synthesized=true)

Relationships (IS-A, only between canonical entities):
  - surgeon["Dr. Smith"] IS-A physician["Dr. Smith"]
  - physician["Dr. Smith"] IS-A person["Dr. Smith"]
  - cardiologist["Dr. Jones"] IS-A physician["Dr. Jones"]
  - physician["Dr. Jones"] IS-A person["Dr. Jones"]
  - patient["John Doe"] IS-A person["John Doe"]
```

### Benefits

1. **No duplicates in graph**: All entities are canonical (extracted or materialized)
2. **Query at any level**: "Find all persons" returns Dr. Smith, Dr. Jones, John Doe
3. **Type-specific queries**: "Find all surgeons" returns only Dr. Smith
4. **Relationship navigation**: Follow IS-A links to get more general/specific types
5. **Mentions track extraction details**: Access which rules extracted each entity
6. **Progressive refinement**: Multiple definitions with different confidence merged into one canonical entity

---

## Design Considerations

### Canonicalization Key

**Question**: What defines a unique entity?

**Answer**: Key = `name|type|element_id`

**Rationale**:
- Same name+type in **same element** → same entity (merge)
- Same name+type in **different elements** → different entities (keep separate)

**Example**:
```
surgeon["Dr. Smith"] from element doc1_para5 → canonical entity A
surgeon["Dr. Smith"] from element doc1_para10 → canonical entity B (different)
```

### Canonical-Only Graph

**Design principle**: ALL entities in final graph must be canonical

**Implications**:
- Extracted entities: Canonicalized before materialization
- Composite entities: Created as canonical entities (no duplicates)
- IS-A relationships: Only connect canonical entities
- Mentions: Track extraction details, but don't appear in graph

**Benefits**:
- No duplicate entities in graph
- Simplified querying (no need to filter duplicates)
- Clear provenance via mentions array

### Synthesized Entity Properties

**Attributes inherited from child**:
```go
parentEntity = &Entity{
    Name:      currentEntity.Name,      // Inherit name
    ElementID: currentEntity.ElementID, // Same element as child
    Mentions:  currentEntity.Mentions,  // Inherit mentions
}
```

**Attributes from parent mapping**:
```go
parentEntity = &Entity{
    Type:   EntityType(parentMapping.EntityType),
    Domain: parentMapping.Domain,
    Attributes: map[string]interface{}{
        "synthesized": true,  // Created during materialization (not extracted)
        "w_category":  parentMapping.WCategory,
    },
}
```

**Confidence**: Not set (irrelevant - no deduplication needed for synthesized entities)

### Attribute Marking

**Extracted entities** (found by extraction rules):
```go
Attributes: map[string]interface{}{
    "synthesized": false,  // Extracted from text
    "w_category":  mapping.WCategory,
}
```

**Synthesized entities** (created during materialization):
```go
Attributes: map[string]interface{}{
    "synthesized": true,  // Created to complete hierarchy
    "w_category":  parentMapping.WCategory,
}
```

**Benefits**:
- **Clear provenance**: `synthesized: false` = extracted from text, `synthesized: true` = created for hierarchy
- **Relationship filtering**: Domain relationships only between extracted entities (synthesized=false)
- **Query flexibility**: Filter by `synthesized` to distinguish real data from inferred hierarchy

---

## Example: Multi-Level Hierarchy with Canonicalization

**Schema**:
```yaml
global.person
  └─ medical.physician
       └─ medical.surgeon
```

**Step 1: Extract (with duplicates)**:
```
surgeon: "Dr. Smith" (element: doc1_para5, confidence: 0.90, rule: high_confidence)
surgeon: "Dr. Smith" (element: doc1_para5, confidence: 0.85, rule: medium_confidence)
```

**Step 2A: Canonicalize**:
```
surgeon: "Dr. Smith" (element: doc1_para5, confidence: 0.90, canonical)
  Mentions: [rule: high_confidence, rule: medium_confidence]
```

**Step 2B: Materialize ancestors**:
```
1. surgeon: "Dr. Smith" (extracted, canonical, synthesized=false)
   └─ IS-A → physician: "Dr. Smith" (synthesized, canonical, synthesized=true)
        └─ IS-A → person: "Dr. Smith" (synthesized, canonical, synthesized=true)
```

**Result**: 3 canonical entities (1 extracted, 2 synthesized), 2 IS-A relationships

---

## Relationship Extraction with Synthesized Entities

### Design Principle

**Domain relationships ONLY between extracted entities** (synthesized=false)

**Rationale**:
- Relationships should reflect what was actually extracted from text
- Synthesized entities exist only to complete the hierarchy
- Creating relationships involving synthesized entities creates noise
- Hierarchical queries can traverse IS-A relationships when needed

### Implementation

**During relationship extraction** (`extractRelationships`):

```go
func (e *RuleBasedExtractor) extractRelationships(ctx context.Context, elements []Element, entities []Entity) ([]Relationship, error) {
    // Build index of extracted entities (synthesized=false)
    extractedIndex := make(map[string]*Entity)
    for i := range entities {
        entity := &entities[i]
        // Only include entities that were extracted (not synthesized)
        if synthesized, ok := entity.Attributes["synthesized"].(bool); !ok || !synthesized {
            extractedIndex[entity.ID] = entity
        }
    }

    var relationships []Relationship

    // For each relationship rule...
    for _, relRule := range schema.RelationshipRules {
        // Try to find source and target in extracted entities only
        sourceEntity := extractedIndex[sourceID]
        targetEntity := extractedIndex[targetID]

        // Only create relationship if BOTH are extracted (not synthesized)
        if sourceEntity != nil && targetEntity != nil {
            relationships = append(relationships, relationship)
        }
    }

    return relationships, nil
}
```

### Example

**Entities**:
```
- surgeon: "Dr. Smith" (extracted, synthesized=false)
- physician: "Dr. Smith" (synthesized, synthesized=true)
- person: "Dr. Smith" (synthesized, synthesized=true)
- hospital: "City Hospital" (extracted, synthesized=false)
- organization: "City Hospital" (synthesized, synthesized=true)
```

**Relationships created**:
```
IS-A relationships (all entities):
  ✅ surgeon["Dr. Smith"] IS-A physician["Dr. Smith"]
  ✅ physician["Dr. Smith"] IS-A person["Dr. Smith"]
  ✅ hospital["City Hospital"] IS-A organization["City Hospital"]

Domain relationships (extracted entities only):
  ✅ surgeon["Dr. Smith"] WORKS_AT hospital["City Hospital"]
  ❌ person["Dr. Smith"] WORKS_AT hospital["City Hospital"]  // person is synthesized
  ❌ surgeon["Dr. Smith"] WORKS_AT organization["City Hospital"]  // organization is synthesized
  ❌ person["Dr. Smith"] WORKS_AT organization["City Hospital"]  // both synthesized
```

### Query Patterns

**To find all people working at hospitals** (including hierarchical types):
```cypher
// Find direct relationships (extracted entities only)
MATCH (s:surgeon)-[:WORKS_AT]->(h:hospital)
// Traverse IS-A to get person
MATCH (s)-[:IS_A*]->(p:person)
RETURN p, h
```

This approach keeps the relationship graph clean while enabling hierarchical queries.

---

## Integration with Validation

### Before Materialization

Schema validation ensures:
- No circular hierarchies (validation catches this)
- No broken parent references (validation catches this)

### During Canonicalization and Materialization

**Canonicalization checks**:
- Deduplication key must be unique
- Highest confidence entity becomes canonical
- All mentions merged into canonical entity

**Materialization checks**:
```go
if mapping == nil || mapping.ParentType == "" {
    break // Gracefully handle missing mappings (reached top)
}

parentMapping := entityTypeMap[mapping.ParentType]
if parentMapping == nil {
    break // Gracefully handle broken references
}
```

**Note**: Validation should catch broken references, but materialization is defensive.

---

## Performance Considerations

### Time Complexity

- **Computation**: O(n) where n = number of entity mappings
- **Canonicalization**: O(m) where m = extracted entities
- **Materialization**: O(c * d) where c = canonical entities, d = max depth

### Space Complexity

- **Canonical index**: O(c) where c = canonical entities
- **Composite cache**: O(c * d) in worst case
- **Relationship array**: O(c * d)

### Optimization

- **Canonicalization**: Single pass with hash map (O(1) lookups)
- **Materialization**: Cache lookups are O(1) with map
- **Early termination**: Stop at orphan/top-level
- **Deduplication**: Prevents exponential blowup of composites
- **Canonical-only**: Reduces graph size by eliminating duplicates before materialization

---

## Next Steps

Proceed to:
- [05_VALIDATION.md](05_VALIDATION.md) - Understand validation rules
- [06_IMPLEMENTATION_STEPS.md](06_IMPLEMENTATION_STEPS.md) - Start implementing
- [07_TESTING_STRATEGY.md](07_TESTING_STRATEGY.md) - Test hierarchy logic
