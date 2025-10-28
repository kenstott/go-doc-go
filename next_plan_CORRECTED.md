# CORRECTED Ontology Interview Refactoring: Implementation Plan

## ⚠️ CRITICAL CORRECTIONS FROM ORIGINAL PLAN

This document contains the **corrected implementation plan** based on code review and architecture discussions. The original `next_plan.md` had several critical issues that have been resolved here.

### Key Corrections

1. **Global Domain Structure**: 10 top-level types + 23 subtypes = **33 global entity types** (not 9)
2. **Multiple Definitions Allowed**: Same `entity_type` can have multiple definitions with different `confidence`/`parent_type`/`w_category`
3. **Bidirectional Hierarchy**: `parent_type` and `children` are both optional, auto-computed at runtime
4. **Validation**: Only check broken references + circular hierarchies (no consistency requirements across definitions)
5. **Type Field Removal**: Fully committed to removing `Type` discriminator from `ExtractionRule`
6. **File Organization**: Split `common.go` into 5 files: `global_who.go`, `global_what.go`, `global_where.go`, `global_when.go`, `global_why.go`
7. **Hierarchy Materialization**: Called AFTER all entity extraction completes

---

## Global Domain Structure (33 Entity Types)

### WHO (2 top-level, 10 subtypes = 12 total)

**person** (w_category: who)
- public_figure
- executive
- employee

**organization** (w_category: who)
- business (for-profit companies, corporations)
- nonprofit (charities, foundations, NGOs)
- government (agencies, departments, bureaus)
- educational (universities, schools, research institutions)
- healthcare (hospitals, clinics, medical centers)
- religious (churches, temples, mosques, religious institutions)
- media (news outlets, publishers, broadcasters)

### WHAT (3 top-level, 6 subtypes = 9 total)

**document** (w_category: what) - standalone
- Description: Referenced documents, reports, papers, publications

**identifier** (w_category: what)
- email
- phone
- url
- code
- id_number

**role** (w_category: what) - standalone
- Description: Job titles, positions, functional roles
- Note: Role is WHAT (a position), not WHO (a person)
- Examples: CEO, Professor, Manager, Director

### WHERE (1 top-level, 5 subtypes = 6 total)

**location** (w_category: where)
- city
- country
- region
- address
- building

### WHEN (2 top-level, 2 subtypes = 4 total)

**date** (w_category: when)
- time
- duration

**event** (w_category: when) - standalone
- Description: Occurrences with start/end times, significant happenings

### WHY (2 top-level, 0 subtypes = 2 total)

**assertion** (w_category: why) - standalone
- Description: Claims, requirements, declarations, statements presented as fact

**hypothesis** (w_category: why) - standalone
- Description: Testable, provisional explanations, theories

---

## Architecture: Multiple Definitions Model

### Design Principle: Progressive Refinement via Confidence

**Key Insight**: The same `entity_type` can have multiple extraction rule definitions with different confidence levels, representing different priority tiers.

#### Example: Physician Entity with Multiple Definitions

```yaml
# High confidence/priority - very specific pattern
- entity_type: physician
  domain: medical
  parent_type: global.person
  w_category: who
  confidence: 0.95
  extraction_rules:
    - instance_name: (?P<name>Dr\. [A-Z][a-z]+ [A-Z][a-z]+)
      proximity:
        keywords: [patient, diagnosis, surgery]
        max_distance: 100

# Medium confidence - broader pattern with semantic filter
- entity_type: physician
  domain: medical
  parent_type: global.person
  w_category: who
  confidence: 0.80
  extraction_rules:
    - instance_name: (?P<name>[A-Z][a-z]+ [A-Z][a-z]+)
      semantic:
        reference_concepts: [medical practice, healthcare provider]
        similarity_threshold: 0.75

# Low confidence - very broad, fallback pattern
- entity_type: physician
  domain: medical
  parent_type: global.person
  w_category: who
  confidence: 0.65
  extraction_rules:
    - instance_name: (?P<name>[A-Z][a-z]+ [A-Z][a-z]+)
      proximity:
        keywords: [hospital, clinic, medical]
        max_distance: 150
```

**Extraction Logic**:
1. Try high-confidence rules first (0.95)
2. Fall back to medium-confidence (0.80) if no match
3. Fall back to low-confidence (0.65) if still no match
4. Canonical entity = highest confidence match

### Design Principle: Flexible Parent/Category Relationships

**ALLOWED**: Different definitions of same entity can have:
- Different `parent_type` values (same entity serving different roles)
- Different `w_category` values (same entity answering different W questions)
- Different `extraction_rules` (progressive refinement)

**Example**: Role entity serving different purposes

```yaml
# Role as organizational position (child of organization context)
- entity_type: ceo
  domain: business
  parent_type: global.role
  w_category: what  # What position
  confidence: 0.90

# Role as person attribute (child of person context)
- entity_type: ceo
  domain: business
  parent_type: global.person
  w_category: who   # Who holds this role
  confidence: 0.85
```

---

## Hierarchy: Bidirectional Auto-Computation

### Design Principle: Optional Parent/Children Fields

Both `parent_type` and `children` are **OPTIONAL**:
- Specify one, both, or neither
- Missing relationships computed at runtime
- `parent_type` is source of truth for upward relationships
- `children` is auto-filled by scanning all entities for matching `parent_type`

### Runtime Computation Algorithm

```go
func (schema *OntologySchema) ComputeHierarchies() error {
    entityMap := make(map[string]*ElementEntityMapping)

    // Build entity map (first occurrence per qualified name)
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

### Example: Global Domain Declaration

Global domain doesn't know about domain catalogs at authoring time:

```yaml
# catalogs/global_who.go
- entity_type: person
  domain: global
  w_category: who
  parent_type: ""      # Top-level
  children: []         # Empty - will be auto-filled at runtime
  extraction_rules:
    - instance_name: (?P<name>[A-Z][a-z]+ [A-Z][a-z]+)
```

### Example: Medical Domain Catalog

Medical catalog declares its parent relationship:

```yaml
# examples/ontologies/industry_sectors/medical.yaml
- entity_type: physician
  domain: medical
  w_category: who
  parent_type: global.person  # Declares upward link
  children: []                # Could list surgeon, cardiologist, etc.
  extraction_rules:
    - instance_name: (?P<name>Dr\. [A-Z][a-z]+ [A-Z][a-z]+)
      proximity:
        keywords: [patient, diagnosis]
```

### Runtime Result

After loading both catalogs and calling `ComputeHierarchies()`:

```yaml
# global.person NOW has children auto-filled
- entity_type: person
  domain: global
  children: [medical.physician, medical.patient, ...]  # Auto-computed!

# medical.physician unchanged (already had parent_type)
- entity_type: physician
  domain: medical
  parent_type: global.person
```

---

## Validation: Only Real Errors

### ✅ Allowed (Design Features)

- Multiple definitions of same `entity_type` with different confidence/parent/w_category
- Entities with no `parent_type` (top-level types)
- Entities with no `children` (leaves - extraction targets)
- `parent_type` and `children` both empty (standalone types)

### ❌ Validation Errors (Reject Schema)

1. **Broken references (orphans)**
   ```yaml
   - entity_type: physician
     parent_type: global.foobar  # ❌ global.foobar doesn't exist

   - entity_type: person
     children: [medical.surgeon]  # ❌ medical.surgeon doesn't exist
   ```

2. **Circular hierarchies**
   ```yaml
   - entity_type: physician
     parent_type: medical.surgeon

   - entity_type: surgeon
     parent_type: medical.physician  # ❌ Circular loop!
   ```

### Validation Algorithm

```go
func (schema *OntologySchema) ValidateHierarchies() error {
    // Build entity existence map
    entityMap := make(map[string]bool)
    for _, mapping := range schema.ElementEntityMappings {
        key := mapping.Domain + "." + mapping.EntityType
        entityMap[key] = true
    }

    // Check for broken references
    for _, mapping := range schema.ElementEntityMappings {
        qualifiedName := mapping.Domain + "." + mapping.EntityType

        // Check parent exists
        if mapping.ParentType != "" && !entityMap[mapping.ParentType] {
            return fmt.Errorf("entity %s references non-existent parent: %s",
                qualifiedName, mapping.ParentType)
        }

        // Check children exist
        for _, childRef := range mapping.Children {
            if !entityMap[childRef] {
                return fmt.Errorf("entity %s references non-existent child: %s",
                    qualifiedName, childRef)
            }
        }
    }

    // Check for circular hierarchies
    for _, mapping := range schema.ElementEntityMappings {
        qualifiedName := mapping.Domain + "." + mapping.EntityType
        if schema.hasCycle(qualifiedName, make(map[string]bool)) {
            return fmt.Errorf("circular hierarchy detected involving: %s", qualifiedName)
        }
    }

    return nil
}
```

**Note**: No consistency checks across multiple definitions of the same entity_type. Each definition is independent.

---

## Unified Extraction Rule Structure

### Removal of Type Discriminator

**OLD Structure** (with Type field):
```go
type ExtractionRule struct {
    Type                 ExtractionRuleType   // content_extraction, metadata_field, jsonpath_query
    InstanceName         string
    Pattern              string
    ProximityFilter      *ProximityFilter
    SemanticFilter       *SemanticFilter
    FieldPath            string               // For metadata_field
    JSONPathExpr         string               // For jsonpath_query
    DictionaryFilter     *DictionaryFilter
    LLMFalsePositiveTest *LLMValidationPrompt
}
```

**NEW Structure** (unified, no Type):
```go
type ExtractionRule struct {
    // Universal addressing
    JSONPath         string                   `json:"jsonpath,omitempty"`        // OPTIONAL: $.metadata.field or $.content.path

    // Extraction methods (choose one)
    PhraseList       []string                 `json:"phrase_list,omitempty"`     // Exact string matching (fast)
    InstanceName     string                   `json:"instance_name"`             // Regex with (?P<name>...)

    // Pre-filters (all optional, fail-fast AND logic)
    Pattern          string                   `json:"pattern,omitempty"`         // Cheap regex pre-filter
    Proximity        *ProximityFilter         `json:"proximity,omitempty"`       // Co-occurrence (renamed, no _Filter)
    Dictionary       *DictionaryFilter        `json:"dictionary,omitempty"`      // Linguistic validation (renamed)
    Semantic         *SemanticFilter          `json:"semantic,omitempty"`        // Embedding similarity (renamed)
    LLMValidation    *LLMValidationPrompt     `json:"llm_validation,omitempty"`  // LLM filtering (renamed from LLMFalsePositiveTest)
}
```

### Performance-Optimized Extraction Pipeline

```
Step 1: JSONPath (OPTIONAL) → FAIL FAST if doesn't match
  - If omitted: extract from $.content (fast path, no JSON conversion)
  - If specified: validate element structure, extract from JSONPath result

Step 2: Phrase List (OPTIONAL) → FAIL FAST if doesn't match
  - 10-100x faster than regex
  - Exact string matching
  - Use for known entity names

Step 3: Instance Name (REQUIRED if no PhraseList) → FAIL FAST if empty
  - Regex with (?P<name>...) capture group
  - SHORT CIRCUIT before expensive filters

Step 4: Pattern (OPTIONAL) → FAIL FAST if doesn't match
  - Cheap regex pre-filter

Step 5: Proximity (OPTIONAL) → FAIL FAST if doesn't match
  - Moderate cost
  - Co-occurrence validation

Step 6: Dictionary (OPTIONAL) → FAIL FAST if doesn't match
  - Moderate cost
  - Linguistic validation (POS, proper nouns)

Step 7: Semantic (OPTIONAL) → FAIL FAST if doesn't match
  - Expensive (embeddings)
  - Element-level similarity

Step 8: LLM Validation (OPTIONAL)
  - Very expensive
  - Batched during canonicalization (not during extraction)
```

### Migration from Old to New

**Old Rule (metadata_field)**:
```yaml
- type: metadata_field
  field_path: author.name
```

**New Rule** (unified):
```yaml
- jsonpath: $.metadata.author.name
  instance_name: (?P<name>.+)
```

**Old Rule (jsonpath_query)**:
```yaml
- type: jsonpath_query
  jsonpath_expr: $.content.company_info.ticker
```

**New Rule** (unified):
```yaml
- jsonpath: $.content.company_info.ticker
  instance_name: (?P<name>[A-Z]{1,5})
```

**Old Rule (content_extraction with filters)**:
```yaml
- type: content_extraction
  instance_name: (?P<name>Dr\. .+)
  proximity_filter:
    cooccurrence_terms: [patient, diagnosis]
    max_distance: 100
  semantic_filter:
    reference_concepts: [medical practice]
    similarity_threshold: 0.70
```

**New Rule** (unified, renamed filters):
```yaml
- instance_name: (?P<name>Dr\. .+)
  proximity:
    cooccurrence_terms: [patient, diagnosis]
    max_distance: 100
  semantic:
    reference_concepts: [medical practice]
    similarity_threshold: 0.70
```

---

## Hierarchy Materialization

### Purpose

After extracting leaf entities, automatically generate:
1. **Composite parent entities** - Aggregate instances at each hierarchy level
2. **IS-A relationships** - Link children to parents

### Example

**Extracted Leaf Entities**:
```
- surgeon: "Dr. Smith" (element: doc1_para5)
- cardiologist: "Dr. Jones" (element: doc1_para10)
- patient: "John Doe" (element: doc1_para5)
```

**Materialized Hierarchy**:
```
Entities:
  - surgeon: "Dr. Smith" (leaf, extracted)
  - cardiologist: "Dr. Jones" (leaf, extracted)
  - patient: "John Doe" (leaf, extracted)
  - physician: "Dr. Smith" (composite, from surgeon)
  - physician: "Dr. Jones" (composite, from cardiologist)
  - person: "Dr. Smith" (composite, from physician)
  - person: "Dr. Jones" (composite, from physician)
  - person: "John Doe" (composite, from patient)

Relationships:
  - surgeon["Dr. Smith"] IS-A physician["Dr. Smith"]
  - cardiologist["Dr. Jones"] IS-A physician["Dr. Jones"]
  - physician["Dr. Smith"] IS-A person["Dr. Smith"]
  - physician["Dr. Jones"] IS-A person["Dr. Jones"]
  - patient["John Doe"] IS-A person["John Doe"]
```

### Integration Point

```go
func (e *RuleBasedExtractor) ExtractFromElements(ctx context.Context, docID string, elements []Element) (*Ontology, error) {
    // Phase 1: Extract leaf entities (entities with no children)
    leafEntities, err := e.extractEntities(ctx, elements)

    // Phase 2: Materialize hierarchy (AFTER all extraction)
    compositeEntities, isaRelationships, err := e.MaterializeHierarchy(leafEntities, e.schema)

    // Phase 3: Combine leaf + composite entities
    allEntities := append(leafEntities, compositeEntities...)

    // Phase 4: Extract domain relationships
    domainRelationships, err := e.extractRelationships(ctx, elements, allEntities)

    // Phase 5: Combine IS-A + domain relationships
    allRelationships := append(isaRelationships, domainRelationships...)

    return ontology, nil
}
```

---

## File Organization

### Current Structure (to be replaced)

```
catalogs/
  └─ common.go  (50+ entity templates, using old rule structure)
```

### New Structure

```
catalogs/
  ├─ global_who.go       # person (3 subtypes), organization (7 subtypes)
  ├─ global_what.go      # document, identifier (5 subtypes), role
  ├─ global_where.go     # location (5 subtypes)
  ├─ global_when.go      # date (2 subtypes), event
  └─ global_why.go       # assertion, hypothesis
```

### Benefits

- Clear organization by W category
- Easier to navigate and maintain
- Each file ~150-200 lines (manageable)
- Aligns with 5 W's conceptual framework

---

## Implementation Checklist

### Step 0: Create Global Domain Catalogs (NEW) ✅ COMPLETE

**Status**: ✅ COMPLETE - Implemented as YAML catalogs instead of Go files
- Created `catalogs/global/*.yaml` (6 files: who, what, where, when, why, how)
- Embedded catalogs using `//go:embed catalogs/**/*.yaml`
- 37 global entity types defined across 6 W's
- All rules use unified structure (no `Type` field)
- All entities have `w_category` and `parent_type` fields
- Auto-loaded and merged into every generated schema

**Implementation Notes**:
- Used YAML format instead of Go code for better maintainability
- Catalogs embedded in binary for zero external dependencies
- ExternalCatalogPath config allows custom extensions

---

### Step 1: Update ElementEntityMapping and Entity Schemas ✅ COMPLETE

**File**: `go/internal/udml/ontology/types.go`

**Add fields to ElementEntityMapping** (lines 90-99):
```go
type ElementEntityMapping struct {
    EntityType      string           `json:"entity_type"`
    ParentType      string           `json:"parent_type,omitempty"`     // NEW
    Children        []string         `json:"children,omitempty"`        // NEW
    Domain          string           `json:"domain"`
    WCategory       string           `json:"w_category"`                // NEW
    Description     string           `json:"description"`
    ElementTypes    []string         `json:"element_types,omitempty"`
    ElementFilter   string           `json:"element_filter,omitempty"`
    Confidence      float64          `json:"confidence"`
    ExtractionRules []ExtractionRule `json:"extraction_rules"`
}
```

**Add field to Entity** (for rule tracking):
```go
type Entity struct {
    ID         string
    Name       string
    Type       EntityType
    Domain     string
    Confidence float64
    Attributes map[string]interface{}
    ElementID  string
    Mentions   []Mention

    // NEW: Track which rule(s) extracted this entity
    ExtractedByRules []string  `json:"extracted_by_rules"`  // Rule IDs

    CreatedAt  time.Time
    UpdatedAt  time.Time
}
```

**Status**: ✅ COMPLETE
- Added `ParentType`, `Children`, `WCategory` fields to ElementEntityMapping
- Separated config (ElementEntityMappingConfig) from runtime (ElementEntityMapping)
- Runtime version has computed Children array populated by ComputeHierarchies()

**Estimated Changes**: +4 lines

---

### Step 2: Remove Type Field from ExtractionRule ✅ COMPLETE

**File**: `go/internal/udml/ontology/types.go` (lines 101-166)

**Delete**:
- Lines 101-108: `ExtractionRuleType` constants
- Line 157: `Type ExtractionRuleType` field
- Lines 162-163: `FieldPath` and `JSONPathExpr` fields

**Add**:
```go
type ExtractionRule struct {
    JSONPath         string                   `json:"jsonpath,omitempty"`
    PhraseList       []string                 `json:"phrase_list,omitempty"`
    InstanceName     string                   `json:"instance_name"`
    Pattern          string                   `json:"pattern,omitempty"`
    Proximity        *ProximityFilter         `json:"proximity,omitempty"`        // Renamed
    Semantic         *SemanticFilter          `json:"semantic,omitempty"`         // Renamed
    Dictionary       *DictionaryFilter        `json:"dictionary,omitempty"`       // Renamed
    LLMValidation    *LLMValidationPrompt     `json:"llm_validation,omitempty"`   // Renamed
}
```

**Update validation** (lines 521-580):
- Remove Type-based switch statement
- Add validation: `instance_name` OR `phrase_list` required
- No Type field to validate

**Status**: ✅ COMPLETE
- Removed `Type` field and `ExtractionRuleType` enum
- Unified structure with JSONPath, PhraseList, InstanceName
- Renamed filter fields: ProximityFilter → Proximity, etc.
- Validation updated to require instance_name OR phrase_list

**Estimated Changes**: -50 deleted, +10 added = -40 net

---

### Step 3: Add Hierarchy Computation & Validation ✅ COMPLETE

**File**: `go/internal/udml/ontology/types.go`

**Add to `Validate()` method** (after line 583):
```go
// Compute bidirectional hierarchies
if err := s.ComputeHierarchies(); err != nil {
    return fmt.Errorf("hierarchy computation failed: %w", err)
}

// Validate hierarchies
if err := s.ValidateHierarchies(); err != nil {
    return fmt.Errorf("hierarchy validation failed: %w", err)
}
```

**Add new methods** (end of file):
```go
// ComputeHierarchies fills missing parent_type and children relationships
func (s *OntologySchema) ComputeHierarchies() error { ... }

// ValidateHierarchies checks for broken references and cycles
func (s *OntologySchema) ValidateHierarchies() error { ... }

// hasCycle detects circular parent references
func (s *OntologySchema) hasCycle(entityName string, visited map[string]bool) bool { ... }

// Helper: contains checks if string slice contains value
func contains(slice []string, value string) bool { ... }
```

**Status**: ✅ COMPLETE
- Implemented ComputeHierarchies() to auto-fill parent_type and children
- Implemented ValidateHierarchies() to detect broken references and cycles
- Added hasCycle() for circular dependency detection
- Integrated into schema Validate() method

**Estimated Changes**: +150 lines

---

### Step 4: Unify Extraction Logic ✅ COMPLETE

**File**: `go/internal/udml/ontology/extractor.go`

**Replace `tryExtractWithRule()`** (lines 206-222):
- Delete Type-based switch
- Implement unified extraction with performance-ordered pipeline

**Delete old methods**:
- `tryExtractWithMetadata()` (lines 226-260)
- `tryExtractWithJSONPath()` (lines 262-336)

**Add new helpers**:
```go
func (e *RuleBasedExtractor) elementToJSON(elem *Element) map[string]interface{} { ... }
func (e *RuleBasedExtractor) findPhraseMatch(content string, phrases []string) string { ... }
func (e *RuleBasedExtractor) checkProximityFilter(...) bool { ... }
```

**Status**: ✅ COMPLETE
- Unified extraction pipeline with fail-fast logic
- Single tryExtractWithRule() handles all extraction types
- JSONPath optional for content/metadata addressing
- PhraseList for fast exact matching
- Performance-ordered filters (cheap → expensive)

**Estimated Changes**: -220 deleted, +180 added = -40 net

---

### Step 5: Add Canonicalization and Hierarchy Materialization ✅ COMPLETE

**File**: `go/internal/udml/ontology/extractor.go`

**Add canonicalization method**:
```go
// canonicalizeEntities deduplicates extracted entities
// Key: name|type|element_id
// Keeps highest confidence, merges mentions
func (e *RuleBasedExtractor) canonicalizeEntities(extractedEntities []Entity) ([]Entity, error) {
    canonicalIndex := make(map[string]*Entity)

    for _, entity := range extractedEntities {
        key := entity.Name + "|" + string(entity.Type) + "|" + entity.ElementID

        if canonical, exists := canonicalIndex[key]; exists {
            // Merge: Keep highest confidence, combine mentions
            if entity.Confidence > canonical.Confidence {
                canonical.Confidence = entity.Confidence
            }
            canonical.Mentions = append(canonical.Mentions, entity.Mentions...)
            canonical.ExtractedByRules = append(canonical.ExtractedByRules, entity.ExtractedByRules...)
        } else {
            entityCopy := entity
            canonicalIndex[key] = &entityCopy
        }
    }

    var canonicalEntities []Entity
    for _, entity := range canonicalIndex {
        canonicalEntities = append(canonicalEntities, *entity)
    }

    return canonicalEntities, nil
}
```

**Add materialization method**:
```go
// MaterializeMissingAncestors creates composite parent entities and IS-A relationships
// Called AFTER canonicalization (only works with canonical entities)
func (e *RuleBasedExtractor) MaterializeMissingAncestors(canonicalLeaves []Entity, schema *OntologySchema) ([]Entity, []Relationship, error) {
    // Build entity type map for lookup
    // Canonical index for composites (prevent duplicates)
    // For each canonical leaf entity, walk up parent chain
    // Materialize missing ancestors as canonical composites
    // Create IS-A relationships only between canonical entities
    return compositeAncestors, isaRelationships, nil
}
```

**Update `ExtractFromElements()`** (line 88-135):
```go
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
```

**Key architecture**: Canonical-only graph (all entities are canonical, no duplicates)

**Status**: ✅ COMPLETE
- Implemented canonicalizeEntities() with name|type|element_id key
- Merges mentions, keeps highest confidence
- Implemented MaterializeMissingAncestors()
- Walks up parent_type chain from each canonical leaf
- Creates synthesized parent entities with `synthesized: true` attribute
- Generates IS-A relationships between canonical entities
- Updated ExtractFromElements() with 6-step canonical flow
- Domain relationships only between extracted (non-synthesized) entities

**Estimated Changes**: +180 lines (canonicalization + updated materialization)

---

### Step 6: Update Domain Catalogs ✅ COMPLETE

**Files**: All YAML files in `examples/ontologies/` (36 files)

**Actions**:
1. Remove `type` field from all extraction rules
2. Convert `type: metadata_field` → `jsonpath: $.metadata.field_path`
3. Convert `type: jsonpath_query` → `jsonpath: $.content.path`
4. Rename filter fields: `proximity_filter` → `proximity`, etc.
5. Add `parent_type` to entities extending global types
6. Add `w_category` to all entities

**Status**: ✅ COMPLETE (Step 6 Part A)
- Migrated 36 domain catalog files from examples/ontologies/ to catalogs/
- Removed `type` field from all extraction rules
- Added `parent_type` field (e.g., `global.product`, `global.person`)
- Added `w_category` field (who/what/where/when/why/how)
- Renamed filter fields (proximity_filter → proximity, etc.)
- Python migration script created: migrate_catalogs.py
- Organized by category: academic, business_functions, cross_cutting, industry_sectors, organizational_types

**Additional Implementation (Step 6 Part B)**:
- ✅ Implemented mergeGlobalCatalog() using embedded files
- ✅ Implemented mergeDomainCatalog() to load domain-specific catalogs
- ✅ Embedded 42 catalogs (6 global + 36 domain) using go:embed
- ✅ Added ExternalCatalogPath config for custom catalog extensions
- ✅ LLM domain selection automatically merges selected domain catalogs

**Estimated Changes**: ~200 lines across 36 files

---

### Step 7: Update Catalog Loader ⚠️ NOT APPLICABLE

**File**: `go/internal/udml/ontology/catalogs/loader.go`

**Update `ExtractionRuleConfig` struct** (lines 41-49):
```go
type ExtractionRuleConfig struct {
    // REMOVE: Type, FieldPath, JSONPathExpr
    JSONPath      string                     `yaml:"jsonpath,omitempty"`
    PhraseList    []string                   `yaml:"phrase_list,omitempty"`
    InstanceName  string                     `yaml:"instance_name,omitempty"`
    Pattern       string                     `yaml:"pattern,omitempty"`
    Proximity     *ProximityFilterConfig     `yaml:"proximity,omitempty"`
    Semantic      *SemanticFilterConfig      `yaml:"semantic,omitempty"`
    Dictionary    *DictionaryFilterConfig    `yaml:"dictionary,omitempty"`
    LLMValidation *LLMValidationPromptConfig `yaml:"llm_validation,omitempty"`
}
```

**Update conversion functions**:
- `convertToOntologyRule()` - map config → ontology types
- `convertFromOntologyRule()` - map ontology → config types

**Status**: ⚠️ NOT APPLICABLE - Catalogs implemented as embedded YAML files instead of Go code with loader
- No catalogs/loader.go file exists
- Catalogs loaded directly from YAML using yaml.Unmarshal
- Types defined in types.go use same structure as YAML

**Estimated Changes**: ~40 lines modified (not needed)

---

### Step 8: Add Unit Tests ⏳ PENDING

**New file**: `go/internal/udml/ontology/types_test.go`

**Tests**:
- `TestComputeHierarchies_BidirectionalFill`
- `TestComputeHierarchies_ParentToChild`
- `TestComputeHierarchies_ChildToParent`
- `TestValidateHierarchies_BrokenParentReference`
- `TestValidateHierarchies_BrokenChildReference`
- `TestValidateHierarchies_CircularDetection`
- `TestValidateHierarchies_ValidHierarchy`
- `TestValidateHierarchies_MultipleDefinitions`

**Update file**: `go/internal/udml/ontology/extractor_test.go`

**Tests**:
- `TestMaterializeHierarchy_SimpleChain`
- `TestMaterializeHierarchy_MultiLevelChain`
- `TestMaterializeHierarchy_Deduplication`
- `TestTryExtractWithRule_UnifiedStructure`
- `TestTryExtractWithRule_JSONPathOptional`
- `TestFindPhraseMatch_ExactMatching`
- `TestFindPhraseMatch_LongestMatch`

**Status**: ⏳ PENDING - Tests need updates for Steps 1-6 changes
- Existing tests in extractor_test.go, builder_test.go need updates
- Tests currently disabled/failing due to structural changes
- Need to update for canonical architecture
- Need to update for config/runtime separation

**Estimated Changes**: +400 lines (new tests)

---

### Step 9: Update Builder LLM Prompts ⏳ PARTIAL

**File**: `go/internal/udml/ontology/builder.go`

**Actions**:
1. Load global domain catalogs at initialization
2. Update LLM prompts to present 10 top-level global types
3. Guide LLM to create domain-specific subtypes
4. Instruct LLM to set `parent_type` when extending global types
5. Remove references to old `Type` field in prompts
6. Add instructions for unified extraction rule structure

**Status**: ⏳ PARTIAL - Some updates done, more needed
- ✅ loadPredefinedDomains() loads from embedded + external catalogs
- ✅ identifyDomains() presents 36 domains to LLM for selection
- ✅ mergeDomainCatalog() merges selected domain catalogs into schema
- ⏳ LLM prompts need updates to:
  - Guide LLM to use parent_type when extending global types
  - Remove references to old Type field
  - Add instructions for unified extraction rule structure
  - Emphasize that domain catalogs provide templates

**Estimated Changes**: +150 lines (prompt updates)

---

## Testing Strategy

### Unit Tests
```bash
cd go/internal/udml/ontology
go test -v -run TestComputeHierarchies
go test -v -run TestValidateHierarchies
go test -v -run TestMaterializeHierarchy
go test -v -run TestTryExtractWithRule
go test -v -run TestFindPhraseMatch
```

### Integration Tests
```bash
# Build binary
cd go
go build -o ../bin/ontology ./cmd/ontology

# Run interview
../bin/ontology interview \
  --config ../tests/test_configs/test.toml \
  --output ../tests/test_output/ontology_results/refactored_schema.json \
  --non-interactive
```

### Validation Tests
```bash
# Verify hierarchy computation
cat tests/test_output/ontology_results/refactored_schema.json | jq '
  .element_entity_mappings[] |
  select(.parent_type != null) |
  {entity_type, parent_type, children, w_category}
'

# Check for orphaned references
cat tests/test_output/ontology_results/refactored_schema.json | jq '
  .element_entity_mappings[] |
  select(.parent_type != "" and (.parent_type | test("^(global|medical|legal)\\."))) |
  {entity_type, parent_type}
'

# Verify no Type fields in extraction rules
cat tests/test_output/ontology_results/refactored_schema.json | jq '
  .element_entity_mappings[].extraction_rules[] |
  select(.type != null) |
  "ERROR: Type field found"
'
```

---

## Summary of Changes

| Component | File | Change Type | Est. Lines |
|-----------|------|-------------|-----------|
| Global catalogs | catalogs/global_*.go | NEW (5 files) | +800 |
| Common catalog | catalogs/common.go | DELETE | -730 |
| Schema types | types.go | ADD fields | +3 |
| Extraction rules | types.go | REMOVE Type | -40 |
| Hierarchy logic | types.go | ADD validation | +150 |
| Extractor | extractor.go | UNIFY extraction | -40 |
| Materialization | extractor.go | ADD hierarchy | +150 |
| Catalog loader | catalogs/loader.go | UPDATE schema | +40 |
| Domain catalogs | examples/ontologies/*.yaml | CONVERT (36 files) | +200 |
| Unit tests | *_test.go | NEW tests | +400 |
| Builder | builder.go | UPDATE prompts | +150 |
| **NET TOTAL** | | | **~1,083 lines** |

---

## Benefits

1. ✅ **33 reusable global entity types** organized by 5 W's
2. ✅ **Progressive refinement** via multiple definitions with confidence levels
3. ✅ **Flexible hierarchies** - optional parent/children, auto-computed at runtime
4. ✅ **Unified extraction structure** - no Type discriminator
5. ✅ **Hierarchy materialization** - automatic parent entity generation
6. ✅ **Minimal validation** - only broken references + cycles
7. ✅ **Clean file organization** - global domain split by W category
8. ✅ **No migration burden** - breaking changes accepted

---

## Migration Notes

**No backward compatibility** - this is a breaking change:
- Old schemas with `Type` field will fail validation
- Old `common.go` templates replaced entirely
- Domain catalogs must be updated to new format

**Migration script NOT provided** - clean slate approach preferred.

---

## Implementation Status Summary

### ✅ Completed Steps (Steps 0-6)

**Steps 0-5**: Core architecture changes
- ✅ Step 0: Global domain catalogs (37 types, embedded YAML)
- ✅ Step 1: ParentType, Children, WCategory fields
- ✅ Step 2: Remove Type field from ExtractionRule
- ✅ Step 3: Hierarchy computation & validation
- ✅ Step 4: Unified extraction logic
- ✅ Step 5: Canonicalization & hierarchy materialization

**Step 6**: Domain catalog ecosystem
- ✅ Step 6 Part A: Migrated 36 domain catalogs
- ✅ Step 6 Part B: Global catalog auto-merge
- ✅ Domain-specific catalog loading and merging
- ✅ Embedded catalogs with external extension support

**Commits**:
- `04b70e64` - Step 5: Canonicalization and hierarchy materialization
- `ec6d19c4` - Step 6 Part B: Global catalog auto-merge
- `46b79260` - Step 6 Part A: Domain catalog migration and embedding

### ⏳ Remaining Steps

**Step 7**: Not applicable (YAML-based catalogs, no loader.go)

**Step 8**: Update tests ⏳
- Tests need updates for Steps 1-6 changes
- Canonical architecture updates required
- Config/runtime separation updates required

**Step 9**: Builder LLM prompts ⏳ PARTIAL
- Domain loading complete
- Prompts need refinement for parent_type guidance
- Need to remove old Type field references

### 🎯 Next Actions

1. ✅ **Complete Step 8**: Update all tests for new architecture
2. ✅ **Complete Step 9**: Refine LLM prompts for unified extraction
3. 🔄 **Strengthen catalogs**: Enhance extraction rules and relationships in domain catalogs
4. 🧪 **Integration testing**: End-to-end test with real corpus
5. 📝 **Documentation**: Update user documentation

---

## Next Steps

1. ✅ Steps 0-6 complete - Core implementation finished
2. ⏳ Step 8 pending - Update tests for new architecture
3. ⏳ Step 9 partial - Refine LLM prompts
4. 🔄 Enhancement needed - Strengthen domain catalogs with robust rules
5. 🧪 Integration test - Validate with real corpus