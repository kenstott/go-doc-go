# Ontology Interview Refactoring: Implementation Checklist

## Overview
Refactor ontology interview to:
1. **Leverage 36 predefined domain catalogs** as templates (not generate from scratch)
2. Use **per-domain workflow** (2 LLM calls per domain: entities + relationships)
3. Apply **5 W's framework** (Who, What, Where, When, Why) with explicit **w_category tracking**
4. **Validate extraction rule types** (restrict to: content_extraction, metadata_field, jsonpath_query)
5. Fix "2-5" hard constraint on entity counts

---

## Goals

- [ ] Use Catalog Templates: Adapt 36 predefined domain templates to corpus (not generate from scratch)
- [ ] Per-Domain Workflow: Process each domain completely (entities + relationships) before moving to next
- [ ] Smaller LLM Contexts: Avoid 26KB JSON responses by per-domain processing
- [ ] 9 Global Entity Types: Expand from 4 to 9 types aligned with 5 W's
- [ ] W-Category Tracking: Every entity must specify which W question it answers
- [ ] Rule Type Validation: Ensure LLM only generates valid extraction rule types
- [ ] Remove Hard Constraints: Eliminate "2-5" entity limit, use corpus-driven flexible approach
- [ ] Cross-Domain Relationships: Final phase with conflict checking

---

## Architecture Changes

### Current Approach (Problematic)
```
Phase 1: Domain Selection → 1 LLM call ✅ (already uses catalogs)
Phase 2: Entity Generation
  - Universal entities (ALL domains) → 4 LLM calls
  - Domain-specific entities (ALL domains, ignores catalogs) → 4 LLM calls ❌
Phase 3: Relationship Generation
  - ALL relationships → 1 LLM call ❌ FAILS (26KB response)

Total: ~9 LLM calls
Problems:
  - Ignores 36 domain catalog templates
  - Final relationship step too large (JSON parsing failure)
  - No entity type hierarchy support
```

### New Approach (Global Domain + Per-Domain with Catalog Templates)
```
Initialization: Load Global Domain
  - Load 9 global entity types from catalogs/common.go
  - Global types: person, organization, location, date, event, document, identifier, assertion, hypothesis
  - Global types have baseline extraction rules (name patterns, date formats, etc.)
  - Can be used directly as leaves OR extended with domain-specific children

Phase 1: Domain Selection → 1 LLM call ✅
  - Already loads 36 catalogs from examples/ontologies/
  - LLM selects matching domains from catalog
  - No changes needed

Phase 2: Per-Domain Processing (foreach selected domain):
  Step A: Entities (template-adapted + corpus-specific + global children) → 1 LLM call
    - Present 9 global entity types as starting point
    - Guide user to create domain-specific children (e.g., physician → person)
    - Load catalog templates for domain
    - Adapt templates to corpus (customize extraction rules)
    - Add corpus-specific entities not in templates
    - Support multi-level hierarchies (e.g., person → physician → surgeon)
    - Assign w_category (inherited from parent or specified)

  Step B: Relationships (intra-domain) → 1 LLM call
    - Generate relationships within domain
    - Use catalog relationship patterns as guidance

Phase 3: Cross-Domain Relationships → 1 LLM call
  - Only relationships between different domains
  - Conflict checking

Phase 4: Hierarchy Materialization (automatic)
  - Walk leaf entities up parent chain
  - Create composite entities for parent types
  - Generate IS-A relationships

Total for 4 domains: 1 + (4 × 2) + 1 = 10 LLM calls
Benefits:
  - Leverages curated catalog templates (36 domains)
  - Global domain provides reusable entity templates
  - Smaller contexts per domain
  - Better error recovery
  - Explainable entity selection (w_category)
  - Entity type hierarchies with rule inheritance
  - No duplication of global entities across domains
```

### Domain-Qualified Reference Notation

**Problem**: When entities reference parents or children across domains, we need unambiguous notation.

**Solution**: **Option B - Implicit Same-Domain, Explicit Cross-Domain**

#### Reference Rules
1. **Same domain**: No prefix needed
   - `surgeon` in medical domain resolves to `medical.surgeon`
2. **Cross domain**: Explicit `domain.entity_type` prefix required
   - `global.person`, `medical.physician`, `legal.attorney`
3. **Global domain**: Always uses explicit prefixes (all children are cross-domain)

#### Reference Resolution
```go
func ResolveEntityReference(ref string, currentDomain string) (domain, entityType string) {
    if strings.Contains(ref, ".") {
        // Fully qualified: "medical.physician"
        parts := strings.Split(ref, ".")
        return parts[0], parts[1]
    }
    // Implicit same-domain: "surgeon" → "medical.surgeon"
    return currentDomain, ref
}
```

#### Leaf Detection (Trivial)
```go
func (m *ElementEntityMapping) IsLeaf() bool {
    return len(m.Children) == 0  // O(1) check
}
```

#### Example: Medical Domain Hierarchy
```yaml
domains:
  - name: global
    entities:
      - entity_type: person
        w_category: who
        children:
          - medical.physician      # Cross-domain = explicit
          - medical.patient
          - legal.attorney
        extraction_rules:          # Baseline rules (inherited by children)
          - instance_name: (?P<name>[A-Z][a-z]+\s+[A-Z][a-z]+)

  - name: medical
    entities:
      - entity_type: physician
        parent_type: global.person      # Inherits name pattern from person
        children:
          - surgeon                     # Same domain = implicit
          - cardiologist
        extraction_rules:               # Adds proximity filter to inherited rules
          - proximity:
              keywords: [patient, diagnosis, clinic]
              max_distance: 100

      - entity_type: surgeon
        parent_type: physician          # Same domain = implicit
        children: []                    # Leaf = will be extracted
        extraction_rules:               # Further refines with surgery keywords
          - proximity:
              keywords: [surgery, operation, surgical]
              max_distance: 80
```

---

## Global Entity Types: 9 Types with Hard-Coded W-Categories

| # | Entity Type | W Category | Hard-Coded | Description |
|---|-------------|------------|------------|-------------|
| 1 | **person** | who | ✅ | Individual humans |
| 2 | **organization** | who | ✅ | Companies, institutions, groups |
| 3 | **location** | where | ✅ | Geographic places, addresses |
| 4 | **date** | when | ✅ | Temporal points |
| 5 | **event** | when | ✅ | Occurrences with start/end times |
| 6 | **document** | what | ✅ | Referenced documents, reports |
| 7 | **identifier** | what | ✅ | IDs, codes, reference labels |
| 8 | **assertion** | why | ✅ | Claims, requirements, declarations |
| 9 | **hypothesis** | why | ✅ | Testable, provisional explanations |

### Global Domain Concept

**Key Principles:**
1. Global types are **pre-defined in `catalogs/common.go`** with baseline extraction rules
2. Global types can be **used directly as leaves** (extracted) OR **extended** by creating domain-specific children
3. When used as parent (has children), global type provides baseline rules that children inherit
4. When used as leaf (no children), global type extracts with its own rules
5. Domain entities use **direct parent references** (no intermediate wrappers)

**Usage Patterns:**

**Pattern 1: Use Global Type Directly (No Specialization Needed)**
```yaml
domains:
  - name: global
    entities:
      - entity_type: person
        w_category: who
        domain: global
        children: []                    # No children = LEAF (extracted)
        extraction_rules:
          - instance_name: (?P<name>[A-Z][a-z]+\s+[A-Z][a-z]+)
```
→ Extracts generic "person" entities with baseline name pattern.

**Pattern 2: Create Domain-Specific Children (Specialization)**
```yaml
domains:
  - name: global
    entities:
      - entity_type: person
        w_category: who
        domain: global
        children:
          - medical.physician    # Has children = provides baseline rules
          - medical.patient
        extraction_rules:
          - instance_name: (?P<name>[A-Z][a-z]+\s+[A-Z][a-z]+)

  - name: medical
    entities:
      - entity_type: physician
        parent_type: global.person    # Inherits name pattern
        domain: medical
        children: []                  # Leaf - will be extracted
        extraction_rules:
          - proximity:              # Adds domain-specific refinement
              keywords: [patient, diagnosis]
              max_distance: 100
```
→ Extracts "physician" entities with inherited + specialized rules.

**Benefits:**
- **Flexible usage**: Use global types directly OR create specialized children
- **Rule inheritance**: Children inherit baseline patterns from global parents
- **Bidirectional navigation**: When children exist, both parent and children reference each other
- **Domain discovery**: Query `person.children` to find all domains using person entities
- **No duplication**: Baseline extraction logic defined once in global types

---

## Unified Extraction Rule (Single JSONPath Method)

**CRITICAL CHANGE**: Eliminate `type`, `source`, and `source_path` fields. Use **JSONPath as universal addressing mechanism** for UDML elements.

**Rationale**:
- Previously: 3 rule types (content_extraction, metadata_field, jsonpath_query) required `type` discriminator
- `metadata_field` and `jsonpath_query` are just different JSONPath expressions
- **JSONPath can address ALL parts of UDML element**: `$.content`, `$.metadata.author.name`, `$.element_type`, etc.
- Single extraction method → no `type` field needed
- Optional JSONPath field acts as **pre-filter** to validate element matches before extraction
- Default behavior: extract from `$.content` (the leaf node) if `jsonpath` omitted
- Avoids expensive JSON conversion when JSONPath not specified (90%+ of rules)

**Extraction Rule Structure** (single method, no `type` field):

```json
{
  "jsonpath": "$.metadata.author.name", // OPTIONAL: validates element matches JSONPath, extracts from result
  "instance_name": "(?P<name>...)",     // REQUIRED: regex to extract entity name from JSONPath result (or content)
  "pattern": "pre-filter regex",        // OPTIONAL: cheap pre-filter
  "proximity": {...},                   // OPTIONAL: co-occurrence validation
  "semantic": {...},                    // OPTIONAL: embedding similarity validation
  "dictionary": {...},                  // OPTIONAL: linguistic validation (proper names vs common nouns)
  "llm_validation": {...}               // OPTIONAL: LLM-based false positive filtering
}
```

**Examples**:

Extract from content (default - no jsonpath needed):
```json
{
  "instance_name": "(?P<name>Dr\\.\\s+[A-Z][a-z]+\\s+[A-Z][a-z]+)",
  "proximity": {"keywords": ["patient", "diagnosis"], "max_distance": 100}
}
```

Extract from content, but ONLY from elements with author metadata:
```json
{
  "jsonpath": "$.metadata.author",
  "instance_name": "(?P<name>Dr\\.\\s+[A-Z][a-z]+ [A-Z][a-z]+)"
}
```

Extract author name from metadata:
```json
{
  "jsonpath": "$.metadata.author.name",
  "instance_name": "(?P<name>.+)",
  "proximity": {"keywords": ["PhD", "researcher"], "max_distance": 50}
}
```

Extract from JSON content with semantic validation:
```json
{
  "jsonpath": "$.content.company_info.ticker",
  "instance_name": "(?P<name>[A-Z]{1,5})",
  "semantic": {"reference_concepts": ["stock market", "equity"], "similarity_threshold": 0.70}
}
```

Extract person names with dictionary validation:
```json
{
  "instance_name": "(?P<name>[A-Z][a-z]+\\s+[A-Z][a-z]+)",
  "dictionary": {
    "require_unknown_words": true,
    "max_known_words_ratio": 0.5,
    "reject_if_all_categories": ["place", "ui_action"]
  }
}
```

**Performance Order** (FAIL FAST on ALL steps - each is AND condition):
1. **JSONPath validation** (OPTIONAL) → FAIL FAST if doesn't match (skip expensive JSON conversion if omitted)
2. **Phrase list matching** (OPTIONAL) → FAIL FAST if doesn't match (10-100x faster than regex, exact string matching)
3. **Instance name extraction** → FAIL FAST if empty (SHORT CIRCUIT before expensive filters)
4. **Pattern filter** (OPTIONAL) → FAIL FAST if doesn't match (cheap regex)
5. **Proximity filter** (OPTIONAL) → FAIL FAST if doesn't match (moderate cost)
6. **Dictionary filter** (OPTIONAL) → FAIL FAST if doesn't match (moderate cost - linguistic validation)
7. **Semantic filter** (OPTIONAL) → FAIL FAST if doesn't match (expensive - embeddings)
8. **LLM validation** (OPTIONAL) → FAIL FAST if doesn't match (very expensive - batched during canonicalization)

**Key Optimization**: If no JSONPath specified (90%+ of rules), skip expensive `elementToJSON()` conversion and extract directly from `elem.Content`.

**INVALID**: `keyword_match`, `regex_match`, `metadata_field`, `jsonpath_query`, `type`, `source`, `source_path` - Must be rejected

---

## Filter Types: When to Use Each

Understanding the purpose of each filter helps LLMs and developers choose the right validation strategy.

### Filter Analysis Framework

| Filter | Analysis Type | Purpose | Cost | Use When |
|--------|---------------|---------|------|----------|
| **jsonpath** | Structural validation | Filter elements by structure/metadata | Instant/Expensive* | Only process elements matching specific structure (*expensive only if used, instant if omitted) |
| **pattern** | Syntactic validation | Pre-filter by textual structure/format | Cheap | You know the exact format (emails, IDs, structured patterns) |
| **dictionary** | Linguistic validation | Part-of-speech, proper nouns, word categories | Moderate | Distinguishing proper names from common words, filtering by POS |
| **proximity** | Co-occurrence validation | Related-by-nearness in text | Moderate | Disambiguation by context clues (keywords near entity) |
| **semantic** | Topical/conceptual validation | Related-by-meaning/subject matter | Expensive | Topical filtering (medical vs. legal context), conceptual relevance |
| **llm_validation** | Contextual validation | Complex reasoning, false positive filtering | Very Expensive | Disambiguation requiring deep understanding, multi-hop reasoning |

*JSONPath cost: Instant if omitted (90%+ of rules), moderate if used (requires JSON conversion)

### Detailed Filter Purposes

#### **jsonpath** - Structural Validation
**What it checks**: Does the element **structure or metadata** match specific criteria?

- **Speed**: Instant if omitted (default: extract from content), moderate if used (requires JSON conversion of element)
- **Use when**:
  - Extracting from metadata fields (not content)
  - Extracting from JSON content (structured data)
  - Only processing elements with specific metadata present
  - Filtering by element type or other structural properties
- **Examples**:
  - Extract from metadata: `"jsonpath": "$.metadata.author.name"`
  - Extract from JSON content: `"jsonpath": "$.content.company_info.ticker"`
  - Only process elements with author: `"jsonpath": "$.metadata.author"` (then extract from content)
  - Only process specific element types: `"jsonpath": "$.element_type"` where result = "paragraph"

**Question to ask**: "Should I extract from a **specific structural location** (not default content)?"

**Design note**: JSONPath is optional. If omitted, extraction defaults to `$.content` (fast path, no JSON conversion needed).

#### **pattern** - Syntactic Validation
**What it checks**: Does the text match a specific **format or structure**?

- **Speed**: Fast (simple regex matching)
- **Use when**: You know the exact format
- **Examples**:
  - Email addresses: `^\w+@\w+\.\w+$`
  - Stock tickers: `^[A-Z]{1,5}$`
  - Phone numbers: `^\d{3}-\d{3}-\d{4}$`
  - IDs: `^[A-Z]{2}\d{6}$`

**Question to ask**: "Is this the right **structure**?"

#### **dictionary** - Linguistic Validation
**What it checks**: What are the **linguistic properties** of the words?

- **Speed**: Moderate (dictionary/WordNet lookups)
- **Use when**: Distinguishing proper names from common words, filtering by part-of-speech
- **Examples**:
  - Person names: Should contain unknown words (not in dictionary)
  - Reject "World Health" (both common nouns)
  - Reject "Capitol Reef" (place + common noun)
  - Accept "John Smith" (proper name, unknown words)

**Question to ask**: "Is this the right **kind of word**?"

#### **proximity** - Co-occurrence Validation
**What it checks**: Is the entity **near relevant signal words**?

- **Speed**: Moderate (token distance calculation)
- **Use when**: Disambiguation by context clues (keywords that appear near entity)
- **Examples**:
  - "aspirin" near ["prescribed", "dose", "mg"] → medication
  - "aspirin" near ["brand", "product", "manufacturer"] → product name
  - "Smith" near ["Dr", "physician", "patient"] → person (not company)

**Question to ask**: "Is this **near the right context clues**?"

#### **semantic** - Topical/Conceptual Validation
**What it checks**: Is the **broader topic/subject** of the element relevant?

- **Speed**: Expensive (element-level embedding + similarity calculation)
- **Use when**: Topical filtering (medical vs. legal context), conceptual relationships
- **Examples**:
  - Element embedding similar to ["stock market trading", "financial securities"] → financial context
  - Element embedding similar to ["medical diagnosis", "patient care"] → medical context
  - Filters out "Apple" (company) mentions in cooking articles

**Question to ask**: "Is this about the right **topic**?"

#### **llm_validation** - Contextual Validation
**What it checks**: **Complex reasoning** that requires understanding context, intent, nuance

- **Speed**: Very expensive (LLM inference, batched during canonicalization)
- **Use when**: Disambiguation requiring deep understanding, multi-hop reasoning
- **Examples**:
  - "Apple announced Q3 earnings" → company (not fruit)
  - "Apple pie recipe uses Granny Smith apples" → fruit (not company)
  - "Paris agreement signed" → international treaty (not city Paris)
  - Requires understanding: sarcasm, negation, hypotheticals, references

**Question to ask**: "Does this make sense **in context**?"

### Progressive Filtering Strategy

Filters are **ordered by cost** (cheap → expensive) and **specificity** (narrow → broad):

```
1. jsonpath      → Structural (which elements/data to process?) ← INSTANT if omitted, MODERATE if used
2. instance_name → Extraction (what to extract from source?) ← CHEAP
3. pattern       → Syntactic (is format valid?) ← CHEAP
4. proximity     → Co-occurrence (near signal words?) ← MODERATE
5. dictionary    → Linguistic (proper name? POS?) ← MODERATE
6. semantic      → Conceptual (topically relevant?) ← EXPENSIVE
7. llm_validation → Contextual (complex reasoning?) ← VERY EXPENSIVE (batched)
```

**Design rationale**: Fail fast on cheap filters before running expensive semantic/LLM analysis. Each filter is an AND condition - if any fails, skip remaining filters.

**Performance optimization**: JSONPath is optional - if omitted (90%+ of rules), extraction defaults to `$.content` with no JSON conversion overhead.

### Example: Extracting "medication" Entities

```json
{
  "instance_name": "(?P<name>aspirin|ibuprofen|acetaminophen|metformin)",
  "pattern": "^[a-z]+$",
  "proximity": {
    "keywords": ["prescribed", "medication", "drug", "dose", "mg", "tablet"],
    "max_distance": 80
  },
  "semantic": {
    "reference_concepts": [
      "pharmaceutical medication and drugs",
      "medical prescription and dosage",
      "patient treatment with medicines"
    ],
    "similarity_threshold": 0.70
  }
}
```

**How it works**:
1. **instance_name**: Extract "aspirin" from text
2. **pattern**: Verify it's lowercase letters only (reject "ASPIRIN" brand name)
3. **proximity**: Check if near medication keywords (reject "aspirin" in product reviews)
4. **semantic**: Verify element discusses medical treatment (reject "aspirin" in chemistry articles about salicylic acid synthesis)

All filters must pass → high-confidence medication entity extraction.

---

## File Locations (Per CLAUDE.md)

**Test Outputs** (gitignored, disposable):
- `tests/test_output/ontology_results/*.json` - Can be safely deleted and regenerated

**Catalog Templates** (version controlled):
- `examples/ontologies/*.yaml` - Must NOT be modified by tests

**Binary Output**:
- `bin/ontology` - All binaries MUST go in `bin/` directory

---

## Implementation Checklist

### 0. Audit Existing Catalog Templates for Rule Type Compliance

File: All YAML files in `examples/ontologies/`

- [ ] Verify catalog directory structure
  ```bash
  pwd
  ls -la examples/ontologies/ | head -10
  find examples/ontologies -name "*.yaml" | wc -l  # Should show 36
  ```

- [ ] Search for invalid rule types in catalogs
  ```bash
  cd examples/ontologies
  grep -r "type: keyword_match" . || echo "No keyword_match found"
  grep -r "type: regex_match" . || echo "No regex_match found"
  ```

- [ ] If found, update to `content_extraction` with `pattern` field

Estimated Time: 30 minutes

---

### 1. Add W-Category Field to Schema (builder.go)

File: `go/internal/udml/ontology/builder.go`

- [ ] Update `ElementEntityMapping` struct to add `WCategory` field:
  ```go
  type ElementEntityMapping struct {
      EntityType              string           `json:"entity_type"`
      Domain                  string           `json:"domain"`
      WCategory               string           `json:"w_category"`  // NEW
      Description             string           `json:"description"`
      Confidence              float64          `json:"confidence"`
      ApplicableElementTypes  []string         `json:"applicable_element_types"`
      ExtractionRules         []ExtractionRule `json:"extraction_rules"`
  }
  ```

- [ ] Add `getWCategoryForEntityType()` function with hard-coded mappings:
  - Universal types: person/org→who, location→where, date/event→when, doc/id→what, assertion/hypothesis→why
  - Common domain types: patient/physician→who, condition/medication→what, symptom→why, etc.
  - Default: "what" for unknown types

- [ ] Add `validateEntityMapping()` function that validates:
  - W-category is present
  - W-category is valid (who/what/where/when/why)
  - Extraction rules have valid types (content_extraction, metadata_field, jsonpath_query)
  - content_extraction rules have instance_name field

**Error Handling**:
- Return errors with fmt.Errorf and %w for wrapping
- Log validation failures with slog

Estimated Changes: ~120 lines added

---

### 2. Update Universal Entity Types (builder.go)

File: `go/internal/udml/ontology/builder.go`

- [ ] Expand universal types from 4 to 9:
  ```go
  var universalTypes = []string{
      "person", "organization", "location", "date",
      "event", "document", "identifier", "assertion", "hypothesis",
  }
  ```

- [ ] Add `universalEntityTemplates` constant with 5 W's descriptions

Estimated Changes: ~30 lines modified

---

### 3. Refactor to Per-Domain Workflow with Catalog Integration (builder.go)

File: `go/internal/udml/ontology/builder.go`

- [ ] Create `defineEntityTypesAndRelationshipsPerDomain()` - main orchestrator
  - Takes catalogPath parameter
  - Loops through each domain
  - Calls entity generation (Step A)
  - Calls relationship generation (Step B)
  - Returns (entities, relationships, calls, tokens, error)
  - Wraps errors with fmt.Errorf and %w
  - Logs progress with slog

- [ ] Create `generateEntitiesWithCatalog()` - entity generation with catalog integration
  - Load catalog template for domain (if exists)
  - Build LLM prompt with:
    - 9 universal entity types
    - Catalog templates (as guidance)
    - Corpus samples
    - 5 W's framework instructions
    - Valid rule type restrictions
    - No "2-5" hard limit
  - Call LLM
  - Parse and return entities

- [ ] Create `generateIntraDomainRelationships()` - per-domain relationships
  - Takes domain entities and catalog template
  - Uses catalog relationship patterns as guidance
  - Generates relationships within domain only

- [ ] Create `generateCrossDomainRelationships()` - cross-domain with conflict checking
  - Generates relationships between domains
  - Filters out conflicts (target type exists in source domain)
  - Reports skipped intra-domain and conflicts

- [ ] Validate all entities after generation:
  - Call `validateEntityMapping()` for each entity
  - Assign w_category if missing using `getWCategoryForEntityType()`

**Estimated Changes**: ~400 lines added

**Key LLM Prompt Requirements**:
- Explicitly list 9 universal types with w_category
- Include catalog templates as "guidance only"
- Instruct to adapt templates to corpus
- Require w_category field in output
- Restrict to valid rule types (content_extraction, metadata_field, jsonpath_query)
- Remove "2-5" limit, use "3-10 based on corpus richness"

---

### 4. Generate Intra-Domain Relationships (builder.go)

**File**: `go/internal/udml/ontology/builder.go`

- [ ] Implement `generateIntraDomainRelationships()` function
  - Load catalog relationship patterns (if available)
  - Build LLM prompt with domain entities, catalog patterns, corpus samples
  - Call LLM to generate relationships
  - Parse and return relationship rules

**Estimated Changes**: ~100 lines added

---

### 5. Cross-Domain Relationships with Conflict Checking (builder.go)

**File**: `go/internal/udml/ontology/builder.go`

- [ ] Implement `generateCrossDomainRelationships()` function
  - Build entity domain map and entity types by domain map
  - Call LLM to generate cross-domain relationships
  - Filter proposed relationships:
    - Skip if source domain == target domain
    - Skip if target entity type exists in source domain (conflict)
  - Report filtering statistics

**Estimated Changes**: ~80 lines added

---

### 6. Wire Up in Interview (interview.go)

**File**: `go/internal/udml/ontology/interview.go`

- [ ] Replace current entity generation call (line ~618) with new function:
  ```go
  entityMappings, relationshipRules, calls, tokens, err :=
    ib.builder.defineEntityTypesAndRelationshipsPerDomain(
      ctx, ib.samples, topEntities, ib.schema.Domains, "./examples/ontologies")
  ```

- [ ] Add W-category coverage report after generation:
  ```go
  wCoverage := make(map[string]int)
  for _, entity := range entityMappings {
      wCoverage[entity.WCategory]++
  }
  fmt.Printf("5 W's Coverage: WHO=%d, WHAT=%d, WHERE=%d, WHEN=%d, WHY=%d\n", ...)
  ```

- [ ] Remove old separate entity and relationship generation calls

**Estimated Changes**: ~40 lines modified

---

### 7. Add Catalog Loading Function (catalogs.go)

**File**: `go/internal/udml/ontology/catalogs.go`

- [ ] Add `loadDomainCatalog(catalogPath, domainName)` function
  - Searches subdirectories: academic, business_functions, industry_sectors, organizational_types, cross_cutting
  - Returns DomainCatalog or error if not found

**Estimated Changes**: ~30 lines added

---

### 8. Eliminate `type` Field and Add `jsonpath` (types.go)

**File**: `go/internal/udml/ontology/types.go`

**Purpose**: Unify extraction rules using JSONPath as universal addressing mechanism for UDML elements.

- [ ] Update `ExtractionRule` struct (lines ~156-166):
  ```go
  type ExtractionRule struct {
      // REMOVE: Type field (no longer needed with single rule structure)
      // Type ExtractionRuleType `json:"type" yaml:"type"`

      // ADD: JSONPath field (optional - validates element matches before extraction)
      JSONPath      string            `json:"jsonpath,omitempty" yaml:"jsonpath,omitempty"`       // Optional: JSONPath to validate/extract from element
      InstanceName  string            `json:"instance_name" yaml:"instance_name"`                 // REQUIRED: regex with (?P<name>...)
      Pattern       string            `json:"pattern,omitempty" yaml:"pattern,omitempty"`         // Optional: cheap pre-filter regex

      // RENAME: Simplified filter names (remove _filter suffix, rename llm_false_positive_test)
      Proximity     *ProximityFilter  `json:"proximity,omitempty" yaml:"proximity,omitempty"`     // Optional: co-occurrence validation
      Semantic      *SemanticFilter   `json:"semantic,omitempty" yaml:"semantic,omitempty"`       // Optional: embedding similarity
      Dictionary    *DictionaryFilter `json:"dictionary,omitempty" yaml:"dictionary,omitempty"`   // Optional: linguistic validation
      LLMValidation *LLMValidationPrompt `json:"llm_validation,omitempty" yaml:"llm_validation,omitempty"` // Optional: LLM-based filtering

      // REMOVE: Old field names
      // ProximityFilter  *ProximityFilter     `json:"proximity_filter,omitempty" yaml:"proximity_filter,omitempty"`
      // SemanticFilter   *SemanticFilter      `json:"semantic_filter,omitempty" yaml:"semantic_filter,omitempty"`
      // DictionaryFilter *DictionaryFilter    `json:"dictionary_filter,omitempty" yaml:"dictionary_filter,omitempty"`
      // LLMFalsePositiveTest *LLMValidationPrompt `json:"llm_false_positive_test,omitempty" yaml:"llm_false_positive_test,omitempty"`

      // REMOVE: All source-related fields (replaced by jsonpath)
      // FieldPath     string `json:"field_path,omitempty" yaml:"field_path,omitempty"`
      // JSONPathExpr  string `json:"jsonpath_expr,omitempty" yaml:"jsonpath_expr,omitempty"`
  }
  ```

- [ ] Remove `ExtractionRuleType` constants (lines ~104-108):
  ```go
  // DELETE these constants entirely
  // type ExtractionRuleType string
  // const (
  //     RuleTypeContent  ExtractionRuleType = "content_extraction"
  //     RuleTypeMetadata ExtractionRuleType = "metadata_field"
  //     RuleTypeJSONPath ExtractionRuleType = "jsonpath_query"
  // )
  ```

- [ ] Update validation logic in `Validate()` method (lines ~558-567):
  ```go
  // REPLACE type-based validation with JSONPath-based validation
  for j, rule := range mapping.ExtractionRules {
      // Validate instance_name is present (REQUIRED)
      if rule.InstanceName == "" {
          return NewValidationError(fmt.Sprintf("entity mapping %d (%s), rule %d: instance_name is required", i, mapping.EntityType, j))
      }

      // JSONPath is optional - no validation needed (empty means extract from $.content)
  }
  ```

**Estimated Changes**: ~25 lines modified (removals + minimal additions)

---

### 9. Unify Extraction Logic in extractor.go

**File**: `go/internal/udml/ontology/extractor.go`

**Purpose**: Replace 3 separate extraction functions with 1 unified function implementing optimal performance order.

- [ ] Remove existing functions (mark for deletion):
  - `tryExtractWithContent()` (lines ~149-224)
  - `tryExtractWithMetadata()` (lines ~226-258)
  - `tryExtractWithJSONPath()` (lines ~262-336)

- [ ] Create new unified `tryExtractEntity()` function:
  ```go
  // tryExtractEntity attempts to extract an entity using unified JSONPath extraction logic
  // Implements optimal performance order with FAIL FAST on ALL steps (each is AND condition)
  func (e *RuleBasedExtractor) tryExtractEntity(mapping ElementEntityMapping, rule ExtractionRule, elem Element) *Entity {
      var sourceData string

      // Step 1: JSONPath validation (OPTIONAL - skip expensive JSON conversion if omitted)
      if rule.JSONPath != "" {
          // Convert element to JSON structure (expensive - only when needed)
          elemJSON := map[string]interface{}{
              "element_id":       elem.ElementID,
              "element_type":     elem.ElementType,
              "content":          elem.Content,
              "content_preview":  elem.ContentPreview,
              "content_location": elem.ContentLocation,
              "parent_id":        elem.ParentID,
              "element_order":    elem.ElementOrder,
              "metadata":         elem.Metadata,
          }

          // Evaluate JSONPath against element structure
          results := e.evaluateJSONPath(elemJSON, rule.JSONPath)
          if len(results) == 0 {
              return nil // FAIL FAST: JSONPath didn't match
          }
          sourceData = fmt.Sprint(results[0])
      } else {
          // Default: extract from content (fast - no JSON conversion)
          sourceData = elem.Content
      }

      if sourceData == "" {
          return nil // FAIL FAST: No source data
      }

      // Step 2: Instance name extraction (FAIL FAST - SHORT CIRCUIT before expensive filters)
      entityName := e.extractInstanceName(rule.InstanceName, sourceData)
      if entityName == "" {
          return nil // FAIL FAST: No entity found - skip all expensive filters
      }

      // Step 3: Pattern filter (OPTIONAL - cheap regex pre-filter)
      if rule.Pattern != "" {
          matched, err := regexp.MatchString(rule.Pattern, sourceData)
          if err != nil || !matched {
              return nil // FAIL FAST: Pattern didn't match
          }
      }

      // Step 4: Proximity filter (OPTIONAL - moderate cost)
      if rule.Proximity != nil {
          if !e.checkProximity(sourceData, entityName, rule.Proximity) {
              return nil // FAIL FAST: Proximity check failed
          }
      }

      // Step 5: Dictionary filter (OPTIONAL - moderate cost, linguistic validation)
      if rule.Dictionary != nil {
          if !e.checkDictionary(entityName, rule.Dictionary) {
              return nil // FAIL FAST: Dictionary validation failed
          }
      }

      // Step 6: Semantic filter (OPTIONAL - most expensive, element-level embedding)
      if rule.Semantic != nil {
          if !e.checkSemantic(elem, rule.Semantic) {
              return nil // FAIL FAST: Semantic validation failed
          }
      }

      // Step 7: LLM validation (OPTIONAL - very expensive, batched during canonicalization)
      // Note: LLM validation applied later during entity canonicalization, not here

      // All filters passed - create entity
      return &Entity{
          Name:       entityName,
          Type:       mapping.EntityType,
          Domain:     mapping.Domain,
          WCategory:  mapping.WCategory,
          Confidence: mapping.Confidence,
          Attributes: map[string]interface{}{
              "jsonpath": rule.JSONPath,
          },
          ElementID: elem.ElementID,
      }
  }
  ```

- [ ] Update `tryExtractEntityFromElement()` to call unified function:
  ```go
  func (e *RuleBasedExtractor) tryExtractEntityFromElement(mapping ElementEntityMapping, elem Element) []*Entity {
      var entities []*Entity
      for _, rule := range mapping.ExtractionRules {
          if entity := e.tryExtractEntity(mapping, rule, elem); entity != nil {
              entities = append(entities, entity)
          }
      }
      return entities
  }
  ```

- [ ] Update helper method signatures:
  - `extractInstanceName(pattern, content string) string` - extract with named group
  - `checkProximity(content, entityName string, filter *ProximityFilter) bool`
  - `checkSemantic(elem Element, filter *SemanticFilter) bool` - use element-level embedding

**Estimated Changes**: ~180 lines (delete ~220, add ~180 unified)

---

### 10. Update 36 Catalog YAML Files

**Files**: All YAML files in `examples/ontologies/`

**Purpose**: Convert `metadata_field` and `jsonpath_query` rules to unified format.

- [ ] Find all metadata_field usages:
  ```bash
  cd examples/ontologies
  grep -r "type: metadata_field" . | wc -l
  ```

- [ ] Find all jsonpath_query usages:
  ```bash
  grep -r "type: jsonpath_query" . | wc -l
  ```

- [ ] Convert metadata_field rules:
  **OLD**:
  ```yaml
  extraction_rules:
    - type: metadata_field
      field_path: author.name
  ```

  **NEW**:
  ```yaml
  extraction_rules:
    - jsonpath: $.metadata.author.name
      instance_name: (?P<name>.+)
  ```

- [ ] Convert jsonpath_query rules:
  **OLD**:
  ```yaml
  extraction_rules:
    - type: jsonpath_query
      jsonpath_expr: $.metadata.company_info.ticker
  ```

  **NEW**:
  ```yaml
  extraction_rules:
    - jsonpath: $.metadata.company_info.ticker
      instance_name: (?P<name>[A-Z]{1,5})
  ```

- [ ] Remove `type` field from all content_extraction rules and rename filter fields:
  **OLD**:
  ```yaml
  extraction_rules:
    - type: content_extraction
      instance_name: (?P<name>...)
      proximity_filter:
        keywords: [patient, diagnosis]
        max_distance: 100
  ```

  **NEW**:
  ```yaml
  extraction_rules:
    - instance_name: (?P<name>...)
      proximity:
        keywords: [patient, diagnosis]
        max_distance: 100
  ```

**Estimated Changes**: Variable (depends on catalog usage, estimate ~50 rules across 36 catalogs)

---

### 11. Update LLM Prompts in builder.go

**File**: `go/internal/udml/ontology/builder.go`

**Purpose**: Update LLM prompts to reflect:
1. Unified rule structure (no `type` field)
2. Global domain registration pattern with direct parents
3. Entity type hierarchies

#### A. Update Rule Structure Prompts

- [ ] Update entity generation prompts (lines ~800-1200):
  - Remove references to `type` field
  - Remove explanations of 3 rule types
  - Add explanation of single unified structure with `jsonpath` field
  - Update examples to show jsonpath-based rules

- [ ] Update prompt sections:
  **OLD**:
  ```
  You have 3 rule types:
  1. content_extraction - type: "content_extraction"
  2. metadata_field - type: "metadata_field", field_path: "..."
  3. jsonpath_query - type: "jsonpath_query", jsonpath_expr: "..."
  ```

  **NEW**:
  ```
  Extraction rules have a SINGLE unified structure (no type field):
  - jsonpath: OPTIONAL - JSONPath to validate/extract from element (default: extract from $.content)
  - instance_name: REQUIRED - regex with (?P<name>...) to extract entity name
  - Optional filters: pattern, proximity, semantic, dictionary, llm_validation

  Examples:
  {
    "instance_name": "(?P<name>Dr\\. [A-Z][a-z]+ [A-Z][a-z]+)",
    "proximity": {"keywords": ["patient"], "max_distance": 100}
  }

  {
    "jsonpath": "$.metadata.author.name",
    "instance_name": "(?P<name>.+)",
    "proximity": {"keywords": ["PhD", "researcher"], "max_distance": 50}
  }

  {
    "jsonpath": "$.content.company_info.ticker",
    "instance_name": "(?P<name>[A-Z]{1,5})",
    "semantic": {"reference_concepts": ["stock market", "equity"], "similarity_threshold": 0.70}
  }
  ```

#### B. Add Global Domain Registration Instructions

- [ ] Add new prompt section for entity type hierarchies:
  ```
  ## GLOBAL ENTITY TYPES AND DOMAIN REGISTRATION

  There are 9 pre-defined global entity types that serve as templates:
  1. person (who) - Individual humans
  2. organization (who) - Companies, institutions, groups
  3. location (where) - Geographic places, addresses
  4. date (when) - Temporal points
  5. event (when) - Occurrences with start/end times
  6. document (what) - Referenced documents, reports
  7. identifier (what) - IDs, codes, reference labels
  8. assertion (why) - Claims, requirements, declarations
  9. hypothesis (why) - Testable, provisional explanations

  ### CRITICAL: Domain Registration Pattern

  For each global type relevant to your domain, you MUST:
  1. Create domain-specific LEAF subtypes (children)
  2. Register children in the global parent's children array
  3. Set parent_type field in child to reference global parent

  **DO:**
  - Create specific subtypes (physician, patient NOT generic "person")
  - Use DIRECT parent references (physician → global.person)
  - Support multi-level hierarchies (surgeon → physician → global.person)

  **DO NOT:**
  - Create intermediate wrappers (NO "medical_person" as middleman)
  - Duplicate global types in your domain
  - Create leaf entities without registering them as children

  ### Example: Medical Domain with Person Entities

  **CORRECT** (Direct parent registration):
  ```json
  {
    "domains": [
      {
        "name": "global",
        "entities": [
          {
            "entity_type": "person",
            "w_category": "who",
            "domain": "global",
            "children": [
              "medical.physician",
              "medical.patient",
              "medical.researcher"
            ],
            "extraction_rules": []
          }
        ]
      },
      {
        "name": "medical",
        "entities": [
          {
            "entity_type": "physician",
            "parent_type": "global.person",
            "domain": "medical",
            "w_category": "who",
            "children": ["surgeon", "cardiologist"],
            "extraction_rules": [
              {
                "instance_name": "(?P<name>Dr\\. [A-Z][a-z]+ [A-Z][a-z]+)",
                "proximity": {
                  "keywords": ["patient", "diagnosis", "clinic"],
                  "max_distance": 100
                }
              }
            ]
          },
          {
            "entity_type": "surgeon",
            "parent_type": "physician",
            "domain": "medical",
            "children": [],
            "extraction_rules": [
              {
                "proximity": {
                  "keywords": ["surgery", "operation", "surgical"],
                  "max_distance": 80
                }
              }
            ]
          },
          {
            "entity_type": "patient",
            "parent_type": "global.person",
            "domain": "medical",
            "children": [],
            "extraction_rules": [
              {
                "proximity": {
                  "keywords": ["admitted", "discharged", "diagnosed"],
                  "max_distance": 80
                }
              }
            ]
          }
        ]
      }
    ]
  }
  ```

  **INCORRECT** (Don't create wrapper):
  ```json
  {
    "domains": [
      {
        "name": "medical",
        "entities": [
          {
            "entity_type": "medical_person",  // ❌ NO! Don't create wrappers
            "parent_type": "global.person",
            "children": ["physician", "patient"]
          }
        ]
      }
    ]
  }
  ```

  ### Key Points:
  1. **Bidirectional references**: Global parent lists domain children, children reference global parent
  2. **Direct inheritance**: NO intermediate wrappers (physician → global.person NOT physician → medical_person → global.person)
  3. **Leaf extraction**: Only entities with empty children array are extracted
  4. **Domain qualification**: Cross-domain references use "domain.entity_type" notation
  5. **Same-domain references**: Within same domain, just use "entity_type" (no prefix)

  ### Multi-Level Hierarchy Example:
  ```json
  {
    "entity_type": "surgeon",
    "parent_type": "physician",           // Same domain = no prefix
    "children": [],                       // Leaf = will be extracted
    "extraction_rules": [...]
  }
  ```
  Result: surgeon → physician → global.person (3-level hierarchy, no wrappers)

  ### Guidance for Each Global Type:

  **If domain needs person entities:**
  - Ask: "What specific types of people? (e.g., physician, patient, attorney, analyst)"
  - Create domain-specific subtypes as direct children of global.person
  - Register them in global.person.children array

  **If domain needs organization entities:**
  - Ask: "What specific types of organizations? (e.g., hospital, law_firm, bank)"
  - Create domain-specific subtypes as direct children of global.organization

  **If domain needs location entities:**
  - Ask: "What specific types of locations? (e.g., clinic, courtroom, trading_floor)"
  - Create domain-specific subtypes as direct children of global.location

  Follow this pattern for all 9 global types relevant to the domain.
  ```

**Estimated Changes**: ~200 lines added (new prompt sections)

---

### 12. Update Tests in extractor_test.go

**File**: `go/internal/udml/ontology/extractor_test.go`

**Purpose**: Update test cases to use unified rule structure.

- [ ] Update test fixtures (lines ~20-50):
  **OLD**:
  ```go
  ExtractionRules: []ExtractionRule{
      {
          Type:      RuleTypeMetadata,
          FieldPath: "speaker",
      },
  }
  ```

  **NEW**:
  ```go
  ExtractionRules: []ExtractionRule{
      {
          JSONPath:     "$.metadata.speaker",
          InstanceName: "(?P<name>.+)",
      },
  }
  ```

- [ ] Update all test cases that remove `Type` field
- [ ] Update test cases that convert `FieldPath` to `JSONPath` (with `$.metadata.` prefix)
- [ ] Update test cases that convert `JSONPathExpr` to `JSONPath`
- [ ] Update test cases that rename filter fields (`ProximityFilter` → `Proximity`, etc.)

**Estimated Changes**: ~30 lines modified

---

### 13. Update Catalog Loader in catalogs/loader.go

**File**: `go/internal/udml/ontology/catalogs/loader.go`

**Purpose**: Update catalog YAML parsing to support new unified structure.

- [ ] Update `ExtractionRuleConfig` struct (lines ~41-49):
  ```go
  type ExtractionRuleConfig struct {
      // REMOVE: Type, FieldPath, JSONPathExpr
      // Type         string `yaml:"type"`
      // FieldPath    string `yaml:"field_path,omitempty"`
      // JSONPathExpr string `yaml:"jsonpath_expr,omitempty"`

      // ADD: JSONPath field (unified addressing)
      JSONPath      string   `yaml:"jsonpath,omitempty" json:"jsonpath,omitempty"`
      InstanceName  string   `yaml:"instance_name,omitempty" json:"instance_name,omitempty"`
      Pattern       string   `yaml:"pattern,omitempty" json:"pattern,omitempty"`

      // RENAME: Simplified filter field names
      Proximity     *ProximityFilterConfig     `yaml:"proximity,omitempty" json:"proximity,omitempty"`
      Semantic      *SemanticFilterConfig      `yaml:"semantic,omitempty" json:"semantic,omitempty"`
      Dictionary    *DictionaryFilterConfig    `yaml:"dictionary,omitempty" json:"dictionary,omitempty"`
      LLMValidation *LLMValidationPromptConfig `yaml:"llm_validation,omitempty" json:"llm_validation,omitempty"`

      // REMOVE: Old filter field names
      // ProximityFilter  *ProximityFilterConfig `yaml:"proximity_filter,omitempty"`
      // SemanticFilter   *SemanticFilterConfig  `yaml:"semantic_filter,omitempty"`
      // ... etc
  }
  ```

- [ ] Update conversion functions (lines ~170-178, ~252-260):
  ```go
  // convertToOntologyRule
  rules[i] = ontology.ExtractionRule{
      JSONPath:      c.JSONPath,
      InstanceName:  c.InstanceName,
      Pattern:       c.Pattern,
      Proximity:     convertProximityFilter(c.Proximity),
      Semantic:      convertSemanticFilter(c.Semantic),
      Dictionary:    convertDictionaryFilter(c.Dictionary),
      LLMValidation: convertLLMValidation(c.LLMValidation),
  }

  // convertFromOntologyRule
  configs[i] = ExtractionRuleConfig{
      JSONPath:      r.JSONPath,
      InstanceName:  r.InstanceName,
      Pattern:       r.Pattern,
      Proximity:     convertProximityFilterConfig(r.Proximity),
      Semantic:      convertSemanticFilterConfig(r.Semantic),
      Dictionary:    convertDictionaryFilterConfig(r.Dictionary),
      LLMValidation: convertLLMValidationConfig(r.LLMValidation),
  }
  ```

**Estimated Changes**: ~20 lines modified

---

### 14. Add Entity Type Hierarchy Support (types.go + extractor.go)

**Concept from UIMA**: Entity types can form IS-A hierarchies with rule inheritance from parent types.

**Files**: `go/internal/udml/ontology/types.go`, `go/internal/udml/ontology/extractor.go`

**Purpose**: Allow entity types to form parent-child hierarchies for DRY catalog authoring and query-time classification.

**Key Design Principles**:
- Hierarchies are **completely optional** (flat taxonomies work fine)
- Rule inheritance is **opt-out** (inherit by default if parent exists)
- **Only leaf types are extracted** (types with `children` are templates only)
- Parent rules are **pre-compiled** at schema load time (not during extraction)
- All fields (domain, w_category, confidence, etc.) inherit from parent with child override

#### Single Unified Structure

- [ ] Update `ElementEntityMapping` struct in types.go:
  ```go
  type ElementEntityMapping struct {
      EntityType              string           `json:"entity_type"`
      ParentType              string           `json:"parent_type,omitempty"`           // OPTIONAL: reference to parent entity
      Children                []string         `json:"children,omitempty"`              // OPTIONAL: child entity types (makes this a template)
      InheritParentRules      *bool            `json:"inherit_parent_rules,omitempty"`  // OPTIONAL: default true, set false to opt-out
      Domain                  string           `json:"domain"`
      WCategory               string           `json:"w_category"`
      Description             string           `json:"description"`
      Confidence              float64          `json:"confidence"`
      ApplicableElementTypes  []string         `json:"applicable_element_types"`
      ExtractionRules         []ExtractionRule `json:"extraction_rules"`
  }

  // ShouldInheritParentRules returns true if should inherit from parent
  func (m *ElementEntityMapping) ShouldInheritParentRules() bool {
      if m.ParentType == "" {
          return false  // No parent to inherit from
      }
      if m.InheritParentRules != nil {
          return *m.InheritParentRules  // Explicit opt-in/opt-out
      }
      return true  // Default: inherit if parent exists
  }

  // IsLeafType returns true if entity has no children (extractable type)
  func (m *ElementEntityMapping) IsLeafType() bool {
      return len(m.Children) == 0
  }
  ```

#### Pre-Compilation at Schema Load

- [ ] Add pre-compilation logic in extractor.go:
  ```go
  type RuleBasedExtractor struct {
      schema                 *OntologySchema
      compiledLeafMappings   map[string]ElementEntityMapping  // Pre-compiled leaf types with inherited rules
      embeddingModel         *EmbeddingModel
  }

  func NewRuleBasedExtractor(schema *OntologySchema) *RuleBasedExtractor {
      extractor := &RuleBasedExtractor{
          schema: schema,
      }

      // Pre-compile leaf type mappings with inherited rules (once at initialization)
      extractor.compiledLeafMappings = extractor.compileLeafTypes()

      return extractor
  }

  // compileLeafTypes identifies leaf types and pre-compiles their effective rules
  func (e *RuleBasedExtractor) compileLeafTypes() map[string]ElementEntityMapping {
      compiled := make(map[string]ElementEntityMapping)

      // Find all leaf types (no children)
      for _, entity := range e.schema.ElementEntityMappings {
          if entity.IsLeafType() {
              // Recursively build effective mapping with parent inheritance
              effective := e.buildEffectiveMappingRecursive(entity)
              compiled[entity.EntityType] = effective
          }
      }

      return compiled
  }

  // buildEffectiveMappingRecursive recursively merges child with parent hierarchy
  func (e *RuleBasedExtractor) buildEffectiveMappingRecursive(mapping ElementEntityMapping) ElementEntityMapping {
      effective := mapping

      // Recursively merge with parent (if exists and inheritance enabled)
      if mapping.ShouldInheritParentRules() && mapping.ParentType != "" {
          parent := e.findEntityMapping(mapping.ParentType)
          if parent != nil {
              // First, recursively build parent's effective mapping (handles multi-level hierarchies)
              effectiveParent := e.buildEffectiveMappingRecursive(*parent)

              // Merge parent fields into child (child overrides parent)
              if effective.Domain == "" {
                  effective.Domain = effectiveParent.Domain
              }
              if effective.WCategory == "" {
                  effective.WCategory = effectiveParent.WCategory
              }
              if effective.Description == "" {
                  effective.Description = effectiveParent.Description
              }
              if effective.Confidence == 0 {
                  effective.Confidence = effectiveParent.Confidence
              }
              if len(effective.ApplicableElementTypes) == 0 {
                  effective.ApplicableElementTypes = effectiveParent.ApplicableElementTypes
              }

              // Combine extraction rules: parent rules FIRST, then child rules
              if len(effectiveParent.ExtractionRules) > 0 {
                  effective.ExtractionRules = append(effectiveParent.ExtractionRules, effective.ExtractionRules...)
              }
          }
      }

      return effective
  }

  // findEntityMapping finds entity mapping by type name
  func (e *RuleBasedExtractor) findEntityMapping(entityType string) *ElementEntityMapping {
      for i := range e.schema.ElementEntityMappings {
          if e.schema.ElementEntityMappings[i].EntityType == entityType {
              return &e.schema.ElementEntityMappings[i]
          }
      }
      return nil
  }
  ```

#### Extract Only Leaf Types

- [ ] Update extraction to use pre-compiled leaf mappings:
  ```go
  func (e *RuleBasedExtractor) ExtractEntities(elements []Element) []*Entity {
      var entities []*Entity

      // Only extract leaf types (pre-compiled with inherited rules)
      for _, leafMapping := range e.compiledLeafMappings {
          for _, elem := range elements {
              extracted := e.tryExtractEntityFromElement(leafMapping, elem)
              entities = append(entities, extracted...)
          }
      }

      return entities
  }

  func (e *RuleBasedExtractor) tryExtractEntityFromElement(mapping ElementEntityMapping, elem Element) []*Entity {
      var entities []*Entity

      // Use pre-compiled rules (already includes inherited parent rules)
      for _, rule := range mapping.ExtractionRules {
          if entity := e.tryExtractEntity(mapping, rule, elem); entity != nil {
              entities = append(entities, entity)
          }
      }

      return entities
  }
  ```

#### Validation Updates

- [ ] Add hierarchy validation in types.go:
  ```go
  func (schema *OntologySchema) Validate() error {
      entityTypes := make(map[string]bool)

      // First pass: collect all entity types
      for _, mapping := range schema.ElementEntityMappings {
          if mapping.EntityType == "" {
              return errors.New("entity_type is required")
          }
          if entityTypes[mapping.EntityType] {
              return fmt.Errorf("duplicate entity_type: %s", mapping.EntityType)
          }
          entityTypes[mapping.EntityType] = true
      }

      // Second pass: validate references
      for _, mapping := range schema.ElementEntityMappings {
          // Validate parent_type exists
          if mapping.ParentType != "" {
              if !entityTypes[mapping.ParentType] {
                  return fmt.Errorf("entity_type '%s' references non-existent parent_type '%s'",
                      mapping.EntityType, mapping.ParentType)
              }
          }

          // Validate children exist
          for _, child := range mapping.Children {
              if !entityTypes[child] {
                  return fmt.Errorf("entity_type '%s' declares non-existent child '%s'",
                      mapping.EntityType, child)
              }
          }

          // Validate no cycles in hierarchy
          if err := schema.validateNoCycles(mapping.EntityType); err != nil {
              return err
          }

          // Warn if parent type has no extraction rules to inherit
          if mapping.ShouldInheritParentRules() && mapping.ParentType != "" {
              parent := schema.findEntityByType(mapping.ParentType)
              if parent != nil && len(parent.ExtractionRules) == 0 {
                  log.Warn("entity inherits from parent with no extraction rules",
                      "entity_type", mapping.EntityType, "parent_type", mapping.ParentType)
              }
          }
      }

      return nil
  }
  ```

#### Query-Time Classification

- [ ] Add query helper to find descendants:
  ```go
  // FindDescendants returns all child types recursively
  func (schema *OntologySchema) FindDescendants(entityType string) []string {
      var descendants []string

      for _, entity := range schema.ElementEntityMappings {
          if entity.EntityType == entityType {
              // Add direct children
              descendants = append(descendants, entity.Children...)

              // Recursively find descendants of children
              for _, child := range entity.Children {
                  descendants = append(descendants, schema.FindDescendants(child)...)
              }
              break
          }
      }

      return descendants
  }
  ```

#### Example Configurations

**Example 1: No hierarchy (most common)**
```yaml
element_entity_mappings:
  - entity_type: person
    w_category: who
    extraction_rules:
      - instance_name: (?P<name>[A-Z][a-z]+ [A-Z][a-z]+)
```
**Result**: Simple, flat taxonomy - extracts "person" entities

**Example 2: Complete hierarchy with default inheritance**
```yaml
element_entity_mappings:
  # Parent type (template, not extracted - has children)
  - entity_type: person
    children: [physician, patient]  # Has children = template only, not extracted
    w_category: who
    domain: universal
    confidence: 0.85
    extraction_rules:
      - phrase_list: [Dr., Mr., Ms., Mrs., Prof.]
        instance_name: (?P<name>[A-Z][a-z]+ [A-Z][a-z]+)

  # Intermediate parent (template and child)
  - entity_type: physician
    parent_type: person  # Inherits from person
    children: [surgeon, cardiologist]  # Has children = template only, not extracted
    domain: medical  # Overrides parent's domain
    confidence: 0.90
    extraction_rules:
      # Combines with parent's rules
      - proximity:
          keywords: [patient, diagnosis, prescribed, clinic]
          max_distance: 100

  # Leaf type (extracted)
  - entity_type: surgeon
    parent_type: physician  # Inherits from physician -> person (multi-level)
    # No children = LEAF TYPE (will be extracted)
    confidence: 0.92  # Overrides parent's confidence
    extraction_rules:
      # Combines with physician + person rules
      - proximity:
          keywords: [surgery, operation, surgical, OR]
          max_distance: 80

  # Leaf type (extracted)
  - entity_type: cardiologist
    parent_type: physician
    # No children = LEAF TYPE
    extraction_rules:
      - proximity:
          keywords: [heart, cardiac, cardiology, ECG]
          max_distance: 80

  # Leaf type (extracted)
  - entity_type: patient
    parent_type: person
    # No children = LEAF TYPE
    domain: medical
    confidence: 0.88
    extraction_rules:
      - proximity:
          keywords: [admitted, discharged, treated, examined]
          max_distance: 80
```

**Pre-Compiled Result** (at schema load):
```
compiledLeafMappings = {
  "surgeon": {
    EntityType: "surgeon",
    WCategory: "who",           // from person
    Domain: "medical",          // from physician (overrides person's "universal")
    Confidence: 0.92,           // from surgeon (overrides physician's 0.90)
    ExtractionRules: [
      // From person (grandparent)
      {PhraseList: [Dr., Mr., Ms., Mrs., Prof.], InstanceName: ...},
      // From physician (parent)
      {Proximity: {Keywords: [patient, diagnosis, prescribed, clinic], MaxDistance: 100}},
      // From surgeon (self)
      {Proximity: {Keywords: [surgery, operation, surgical, OR], MaxDistance: 80}},
    ]
  },
  "cardiologist": {...},  // Similar structure
  "patient": {...}        // Similar structure
}
```

**Extraction**: Only iterates over `compiledLeafMappings` (surgeon, cardiologist, patient)

**Example 3: Hierarchy with explicit opt-out**
```yaml
element_entity_mappings:
  - entity_type: person
    children: [physician]
    extraction_rules:
      - phrase_list: [Dr., Mr., Ms.]

  - entity_type: physician
    parent_type: person
    inherit_parent_rules: false  # Explicitly opt-out of inheritance
    # No children = LEAF TYPE
    w_category: who
    extraction_rules:
      # Complete rules here, parent's phrase_list NOT inherited
      - instance_name: (?P<name>Dr\. [A-Z][a-z]+ [A-Z][a-z]+)
        proximity:
          keywords: [patient, diagnosis]
```
**Result**: physician does NOT inherit person's rules

**Benefits**:
- **Single unified structure** - No separate TypeHierarchy, just `parent_type` + `children` fields
- **DRY catalog authoring** - Define base patterns once in parent types
- **Clear extraction model** - Only leaf types (no children) are extracted
- **Pre-compiled performance** - Hierarchy traversal happens once at initialization
- **Complete field inheritance** - All fields (domain, w_category, confidence, etc.) inherit from parent
- **Multi-level hierarchies** - Supports surgeon → physician → person chains
- **Query-time classification** - Use parent_type/children for IS-A queries
- **Completely optional** - Flat taxonomies work fine (no parent_type, no children)
- **Opt-out design** - Only specify `inherit_parent_rules: false` when needed

**Estimated Changes**: ~150 lines added

---

### 15. Add Phrase List Matching (types.go + extractor.go)

**Concept from spaCy**: Exact string matching before regex (10-100x faster for known terms).

**Files**: `go/internal/udml/ontology/types.go`, `go/internal/udml/ontology/extractor.go`

**Purpose**: Add high-performance exact string matching for known entity names before expensive regex.

- [ ] Update `ExtractionRule` struct in types.go:
  ```go
  type ExtractionRule struct {
      JSONPath      string            `json:"jsonpath,omitempty" yaml:"jsonpath,omitempty"`
      PhraseList    []string          `json:"phrase_list,omitempty" yaml:"phrase_list,omitempty"`  // NEW: exact string matches (FAST)
      InstanceName  string            `json:"instance_name" yaml:"instance_name"`
      Pattern       string            `json:"pattern,omitempty" yaml:"pattern,omitempty"`
      Proximity     *ProximityFilter  `json:"proximity,omitempty" yaml:"proximity,omitempty"`
      Semantic      *SemanticFilter   `json:"semantic,omitempty" yaml:"semantic,omitempty"`
      Dictionary    *DictionaryFilter `json:"dictionary,omitempty" yaml:"dictionary,omitempty"`
      LLMValidation *LLMValidationPrompt `json:"llm_validation,omitempty" yaml:"llm_validation,omitempty"`
  }
  ```

- [ ] Add phrase matching logic in extractor.go (Step 2 in pipeline):
  ```go
  func (e *RuleBasedExtractor) tryExtractEntity(mapping ElementEntityMapping, rule ExtractionRule, elem Element) *Entity {
      var sourceData string

      // Step 1: JSONPath validation (OPTIONAL)
      if rule.JSONPath != "" {
          elemJSON := map[string]interface{}{...}
          results := e.evaluateJSONPath(elemJSON, rule.JSONPath)
          if len(results) == 0 {
              return nil // FAIL FAST
          }
          sourceData = fmt.Sprint(results[0])
      } else {
          sourceData = elem.Content
      }

      if sourceData == "" {
          return nil // FAIL FAST
      }

      // Step 2: Phrase list matching (OPTIONAL - 10-100x faster than regex)
      var entityName string
      if len(rule.PhraseList) > 0 {
          entityName = e.findPhraseMatch(sourceData, rule.PhraseList)
          if entityName == "" {
              return nil // FAIL FAST: No phrase matched
          }
      } else if rule.InstanceName != "" {
          // Step 3: Instance name extraction (regex fallback)
          entityName = e.extractInstanceName(rule.InstanceName, sourceData)
          if entityName == "" {
              return nil // FAIL FAST: No entity found
          }
      } else {
          return nil // FAIL FAST: No extraction method specified
      }

      // Step 4: Pattern filter (OPTIONAL - cheap regex pre-filter)
      if rule.Pattern != "" {
          matched, err := regexp.MatchString(rule.Pattern, sourceData)
          if err != nil || !matched {
              return nil // FAIL FAST
          }
      }

      // ... rest of filters (proximity, dictionary, semantic, llm_validation)

      return &Entity{
          Name:       entityName,
          Type:       mapping.EntityType,
          Domain:     mapping.Domain,
          WCategory:  mapping.WCategory,
          Confidence: mapping.Confidence,
          Attributes: map[string]interface{}{
              "jsonpath":    rule.JSONPath,
              "phrase_list": len(rule.PhraseList) > 0,
          },
          ElementID: elem.ElementID,
      }
  }
  ```

- [ ] Implement `findPhraseMatch()` helper:
  ```go
  // findPhraseMatch performs case-insensitive exact string matching
  // Returns longest matching phrase (handles overlaps like "New York" vs "York")
  func (e *RuleBasedExtractor) findPhraseMatch(content string, phrases []string) string {
      contentLower := strings.ToLower(content)

      var longestMatch string
      for _, phrase := range phrases {
          phraseLower := strings.ToLower(phrase)
          if strings.Contains(contentLower, phraseLower) {
              // Keep longest match to handle nested phrases
              if len(phrase) > len(longestMatch) {
                  longestMatch = phrase
              }
          }
      }

      return longestMatch
  }
  ```

**Example Usage in Catalogs**:
```yaml
# Known medication names (fast exact matching)
entity_types:
  - name: medication
    w_category: what
    extraction_rules:
      - phrase_list:
          - aspirin
          - ibuprofen
          - acetaminophen
          - metformin
          - lisinopril
        proximity:
          keywords: [prescribed, mg, dose, tablet]
          max_distance: 80

# Known organization names
  - name: organization
    w_category: who
    extraction_rules:
      - phrase_list:
          - World Health Organization
          - Centers for Disease Control
          - National Institutes of Health
          - Food and Drug Administration
        # No other filters needed for exact matches
```

**Performance Order Update** (with phrase_list):
```
1. jsonpath       → Structural (instant if omitted, moderate if used)
2. phrase_list    → Exact string match (VERY FAST - 10-100x faster than regex)
3. instance_name  → Regex extraction (fallback if no phrase_list)
4. pattern        → Syntactic validation
5. proximity      → Co-occurrence validation
6. dictionary     → Linguistic validation
7. semantic       → Topical validation
8. llm_validation → Contextual validation (batched)
```

**Key Design Decision**: `phrase_list` and `instance_name` are mutually exclusive per rule:
- If `phrase_list` present: Use exact matching (fast path)
- If only `instance_name`: Use regex extraction (flexible path)
- A single entity type can have multiple rules with different methods

**Benefits**:
- 10-100x faster for known entity names (no regex compilation/backtracking)
- Simpler for LLMs to specify (just list of strings, no regex syntax)
- Handles common case: extracting known terms from medical/legal/domain vocabularies
- Reduces false positives (exact match = high precision)

**Estimated Changes**: ~60 lines added

---

### 16. Materialize Entity Hierarchy as Composite Entities (extractor.go)

**Purpose**: After extracting leaf types, create composite parent entities and IS-A relationships to represent the hierarchy in the knowledge graph.

**Files**: `go/internal/udml/ontology/extractor.go`

**Rationale**:
- Leaf extraction produces specific entities (surgeon, cardiologist, patient)
- Queries like "find all person entities" need composite parent entities
- IS-A relationships enable hierarchy traversal and reasoning

#### Post-Processing Phase

- [ ] Add hierarchy materialization after extraction completes:
  ```go
  // MaterializeHierarchy creates composite parent entities and IS-A relationships
  func (e *RuleBasedExtractor) MaterializeHierarchy(extractedEntities []*Entity) ([]*Entity, []*Relationship, error) {
      // Group extracted entities by type
      entitiesByType := make(map[string][]*Entity)
      for _, entity := range extractedEntities {
          entitiesByType[entity.Type] = append(entitiesByType[entity.Type], entity)
      }

      var compositeEntities []*Entity
      var isaRelationships []*Relationship

      // For each extracted leaf entity, create parent composites + IS-A relationships
      for _, entity := range extractedEntities {
          mapping := e.findEntityMapping(entity.Type)
          if mapping == nil {
              continue
          }

          // Walk up parent chain, creating composites and IS-A relationships
          currentType := entity.Type
          currentEntityID := entity.ID

          for {
              parent := e.findEntityMapping(mapping.ParentType)
              if parent == nil {
                  break // No more parents
              }

              // Create composite parent entity (if not already created)
              compositeID := e.getOrCreateCompositeEntity(
                  entity.Name,
                  parent.EntityType,
                  entity.ElementID,
                  parent,
                  &compositeEntities,
              )

              // Create IS-A relationship: child IS-A parent
              isaRelationships = append(isaRelationships, &Relationship{
                  ID:             generateRelationshipID(),
                  SourceEntityID: currentEntityID,
                  TargetEntityID: compositeID,
                  Type:           "IS_A",
                  Confidence:     1.0, // Hierarchy relationships are definitive
                  Properties: map[string]interface{}{
                      "hierarchy_level": "direct_parent",
                  },
              })

              // Move up hierarchy
              currentType = parent.EntityType
              currentEntityID = compositeID
              mapping = parent
          }
      }

      return compositeEntities, isaRelationships, nil
  }

  // getOrCreateCompositeEntity creates or returns existing composite entity
  func (e *RuleBasedExtractor) getOrCreateCompositeEntity(
      name string,
      entityType string,
      elementID string,
      mapping *ElementEntityMapping,
      compositeEntities *[]*Entity,
  ) string {
      // Check if composite already exists (same name + type + element)
      compositeKey := fmt.Sprintf("%s|%s|%s", name, entityType, elementID)

      // Search existing composites
      for _, composite := range *compositeEntities {
          if composite.Attributes["composite_key"] == compositeKey {
              return composite.ID
          }
      }

      // Create new composite entity
      composite := &Entity{
          ID:         generateEntityID(),
          Name:       name,
          Type:       entityType,
          Domain:     mapping.Domain,
          WCategory:  mapping.WCategory,
          Confidence: mapping.Confidence,
          Attributes: map[string]interface{}{
              "composite_key": compositeKey,
              "composite":     true, // Mark as composite (not directly extracted)
          },
          ElementID: elementID,
      }

      *compositeEntities = append(*compositeEntities, composite)
      return composite.ID
  }
  ```

- [ ] Update main extraction workflow to include materialization:
  ```go
  func (e *RuleBasedExtractor) ExtractEntitiesAndRelationships(elements []Element) (*ExtractionResult, error) {
      // Phase 1: Extract leaf types
      leafEntities := e.ExtractEntities(elements)

      // Phase 2: Materialize hierarchy (create parent composites + IS-A relationships)
      compositeEntities, isaRelationships, err := e.MaterializeHierarchy(leafEntities)
      if err != nil {
          return nil, fmt.Errorf("failed to materialize hierarchy: %w", err)
      }

      // Combine leaf + composite entities
      allEntities := append(leafEntities, compositeEntities...)

      // Phase 3: Extract domain relationships (existing logic)
      domainRelationships := e.ExtractRelationships(allEntities, elements)

      // Combine IS-A + domain relationships
      allRelationships := append(isaRelationships, domainRelationships...)

      return &ExtractionResult{
          Entities:      allEntities,
          Relationships: allRelationships,
      }, nil
  }
  ```

#### Example: Hierarchy Materialization

**Input**: Extracted leaf entities
```
- surgeon: "Dr. Smith" (element: doc1_para5)
- cardiologist: "Dr. Smith" (element: doc1_para5)
- patient: "John Doe" (element: doc1_para5)
```

**Output**: Leaf + Composite entities + IS-A relationships
```
Entities:
  - surgeon: "Dr. Smith" (leaf)
  - cardiologist: "Dr. Smith" (leaf)
  - patient: "John Doe" (leaf)
  - physician: "Dr. Smith" (composite, from surgeon)
  - physician: "Dr. Smith" (composite, from cardiologist) [deduplicated as same]
  - person: "Dr. Smith" (composite, from physician)
  - person: "John Doe" (composite, from patient)

Relationships:
  - surgeon["Dr. Smith"] IS-A physician["Dr. Smith"]
  - cardiologist["Dr. Smith"] IS-A physician["Dr. Smith"]
  - physician["Dr. Smith"] IS-A person["Dr. Smith"]
  - patient["John Doe"] IS-A person["John Doe"]
```

**Query Benefits**:
- Query: "Find all person entities" → Returns: Dr. Smith (composite), John Doe (composite)
- Query: "Find all physician entities" → Returns: Dr. Smith (composite)
- Query: "Find all surgeon entities" → Returns: Dr. Smith (leaf)
- Traversal: Follow IS-A relationships for specialization/generalization reasoning

#### Deduplication Logic

- [ ] Add composite deduplication to prevent duplicate parent entities:
  ```go
  // Composites are identified by: name + type + element
  // "Dr. Smith" + "physician" + "doc1_para5" → single composite
  // Even if created from multiple children (surgeon, cardiologist)
  ```

**Benefits**:
- **Queryable hierarchy**: Search at any level (person, physician, surgeon)
- **Explicit relationships**: IS-A relationships enable graph traversal
- **Reasoning support**: Infer properties from parent types
- **Composite entities**: Aggregate all children of a parent type
- **Deduplication**: Same entity at parent level not duplicated

**Estimated Changes**: ~120 lines added

---

## Testing Strategy

### Test 0: Audit Catalog Templates

- [ ] Run audit commands:
  ```bash
  cd examples/ontologies
  grep -r "type: keyword_match" . || echo "✓ No keyword_match found"
  grep -r "type: regex_match" . || echo "✓ No regex_match found"
  ```

- [ ] If found, update to `content_extraction`

---

### Test 1: Build and Compile

- [ ] Build the ontology binary:
  ```bash
  cd go
  go build -o ../bin/ontology ./cmd/ontology
  ```

- [ ] Expected: Clean build with no errors

---

### Test 2: Run Interview with Catalog Integration

- [ ] Run ontology interview:
  ```bash
  bin/ontology interview \
    --config tests/test_configs/test.toml \
    --output tests/test_output/ontology_results/catalog_integrated_schema.json \
    --non-interactive
  ```

- [ ] Expected outcomes:
  - Phase 1: Loads 36 catalog domains ✓
  - Phase 2:
    - For each domain: "✓ Loaded catalog template: X entity types, Y relationships"
    - 9 universal entities per domain
    - Template-adapted entities (from catalog)
    - Corpus-specific entities (not in catalog)
    - All entities have valid w_category
    - All extraction rules have valid types
  - Phase 3: Cross-domain relationships with conflict checking
  - Total ~10 LLM calls for 4 domains
  - W-category coverage report displayed

---

### Test 3: Verify Schema Structure

- [ ] Check schema structure:
  ```bash
  cat tests/test_output/ontology_results/catalog_integrated_schema.json | jq '
  {
    domains: .domains | length,
    entities: .element_entity_mappings | length,
    w_coverage: (.element_entity_mappings | group_by(.w_category) |
      map({w: .[0].w_category, count: length})),
    template_entities: [.element_entity_mappings[] |
      select(.description | contains("catalog") or contains("template")) | .entity_type],
    corpus_entities: [.element_entity_mappings[] |
      select(.description | contains("corpus-specific")) | .entity_type],
    relationships: .relationship_rules | length
  }'
  ```

- [ ] Expected:
  - W coverage: who (~10), what (~35), where (~4), when (~8), why (~8)
  - Mix of template-adapted and corpus-specific entities
  - All entities have w_category

---

### Test 4: Validate W-Categories and Rule Types

- [ ] Check for missing w_category:
  ```bash
  cat tests/test_output/ontology_results/catalog_integrated_schema.json | jq '
  [.element_entity_mappings[] | select(.w_category == null or .w_category == "")] | length
  '
  # Expected: 0
  ```

- [ ] Check for invalid w_categories:
  ```bash
  cat tests/test_output/ontology_results/catalog_integrated_schema.json | jq '
  [.element_entity_mappings[] |
    select(.w_category | IN("who", "what", "where", "when", "why") | not)] | length
  '
  # Expected: 0
  ```

- [ ] Check for invalid rule types:
  ```bash
  cat tests/test_output/ontology_results/catalog_integrated_schema.json | jq '
  [.element_entity_mappings[].extraction_rules[] |
    select(.type | IN("content_extraction", "metadata_field", "jsonpath_query") | not)]
  '
  # Expected: []
  ```

---

## Summary of Changes

| File | Function/Area | Change Type | Est. Lines |
|------|--------------|-------------|-----------|
| **Catalog audit** | examples/ontologies/*.yaml | Fix invalid rule types | Variable |
| `builder.go` | `ElementEntityMapping` struct | Add w_category field | +1 |
| `builder.go` | `getWCategoryForEntityType()` | Add W-category mapping | +50 |
| `builder.go` | `validateEntityMapping()` | Add validation (w_category + rule types) | +70 |
| `builder.go` | Universal types | Expand 4 → 9 types | +30 |
| `builder.go` | `defineEntityTypesAndRelationshipsPerDomain()` | New per-domain orchestrator | +100 |
| `builder.go` | `generateEntitiesWithCatalog()` | Catalog integration + LLM prompt | +150 |
| `builder.go` | `generateIntraDomainRelationships()` | Per-domain relationships | +100 |
| `builder.go` | `generateCrossDomainRelationships()` | Cross-domain with conflict check | +80 |
| `builder.go` | LLM prompts | Update to unified rule structure | ~100 modified |
| `catalogs.go` | `loadDomainCatalog()` | Load single domain catalog | +30 |
| `interview.go` | Phase 2 wiring | Wire new workflow + W coverage report | +40 |
| **types.go** | `ExtractionRule` struct | Remove `type`, add `source`/`source_path` | ~40 modified |
| **types.go** | Constants | Remove `ExtractionRuleType` constants | -10 deleted |
| **types.go** | `Validate()` | Update validation for unified rules | ~30 modified |
| **extractor.go** | Extraction functions | Unify 3 functions into 1 with SHORT CIRCUIT | -220 deleted, +180 added |
| **extractor_test.go** | Test fixtures | Update to unified rule structure | ~30 modified |
| **catalogs/loader.go** | `ExtractionRuleConfig` | Update struct and conversions | ~20 modified |
| **examples/ontologies/*.yaml** | 36 catalog files | Convert metadata_field/jsonpath_query rules | ~50 rules |
| **types.go** | `ElementEntityMapping` struct | Add `parent_type` + `children` fields | +2 |
| **types.go** | Helper methods | `ShouldInheritParentRules()`, `IsLeafType()` | +15 |
| **extractor.go** | `RuleBasedExtractor` struct | Add `compiledLeafMappings` field | +1 |
| **extractor.go** | `NewRuleBasedExtractor()` | Pre-compile leaf types at initialization | +5 |
| **extractor.go** | `compileLeafTypes()` | Identify and compile leaf types | +15 |
| **extractor.go** | `buildEffectiveMappingRecursive()` | Recursive field+rule inheritance | +35 |
| **extractor.go** | `findEntityMapping()` | Helper for entity lookup | +10 |
| **extractor.go** | `ExtractEntities()` | Extract only pre-compiled leaf types | ~20 modified |
| **types.go** | `OntologySchema.Validate()` | Hierarchy validation (parent/child/cycles) | +50 |
| **types.go** | `OntologySchema.FindDescendants()` | Query-time classification helper | +15 |
| **types.go** | `ExtractionRule` struct | Add `phrase_list` field | +1 |
| **extractor.go** | `findPhraseMatch()` | Exact string matching helper | +20 |
| **extractor.go** | `tryExtractEntity()` | Update to use phrase_list (Step 2) | ~30 modified |
| **extractor.go** | `MaterializeHierarchy()` | Create composite parent entities + IS-A relationships | +60 |
| **extractor.go** | `getOrCreateCompositeEntity()` | Deduplication helper for composites | +25 |
| **extractor.go** | `ExtractEntitiesAndRelationships()` | Orchestrate leaf extraction + materialization | +35 |
| **Total** | | | **~1240 lines (651 original + 260 unification + 329 new features)** |

---

## Expected Benefits

### Per-Domain Workflow Benefits
- [x] **Leverages 36 curated catalog templates** (not generating from scratch)
- [x] **Smaller LLM contexts** per domain (no 26KB failures)
- [x] **10 LLM calls** instead of 14 (more efficient)
- [x] **9 universal types** with 5 W's coverage
- [x] **W-category tracking** (explainability - answers "which W question does this entity address?")
- [x] **Flexible entity counts** (3-10, corpus-driven, no hard "2-5" constraint)
- [x] **Template + corpus hybrid** (best of curated + discovered)
- [x] **Cross-domain conflict checking** (prevents ambiguous relationships)
- [x] **Per-domain error recovery** (failure in one domain doesn't block others)

### Rule Type Unification Benefits
- [x] **Single extraction rule structure** (simpler for LLMs - no `type` field needed)
- [x] **Filters work on ALL sources** (proximity/semantic filters now available for metadata/jsonpath)
- [x] **Optimal performance with SHORT CIRCUIT** (instance_name extraction moved to step 2, fails fast before expensive filters)
- [x] **Cleaner implementation** (1 extraction function instead of 3, ~40 lines net reduction)
- [x] **Unified JSONPath handling** (metadata is just JSONPath with simple dot notation)
- [x] **Simpler LLM prompts** (explain 1 structure instead of 3 rule types)

---

## Key Decisions Made

1. **Template Augmentation**: Option A - Templates as examples/guidance (LLM free to customize heavily)
2. **W-Category Assignment**: Option C - Hard-code mappings for known types, LLM assigns for new ones
3. **Rule Type Validation**: Yes - restrict to valid types only (content_extraction, metadata_field, jsonpath_query)
4. **Catalog Evolution**: Option B - Use for current schema only, don't update catalogs

---

## Notes

- Phase 1 (Domain Selection) already works correctly with catalogs - **no changes needed**
- Per-domain workflow reduces LLM context size and prevents 26KB JSON parsing failures
- W-category provides explainability for every entity extraction decision
- Catalog templates provide high-quality starting point while allowing corpus adaptation
- Validation ensures LLM output conforms to expected schema