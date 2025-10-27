# Unified Extraction Rule Structure

## Overview

This document explains the **unified extraction rule structure** after removing the `Type` discriminator field.

---

## Key Change: No Type Field

### OLD Structure (with Type discriminator)

```go
type ExtractionRule struct {
    Type                 ExtractionRuleType   // "content_extraction", "metadata_field", "jsonpath_query"

    // For content_extraction
    InstanceName         string
    Pattern              string
    ProximityFilter      *ProximityFilter
    SemanticFilter       *SemanticFilter

    // For metadata_field
    FieldPath            string

    // For jsonpath_query
    JSONPathExpr         string

    // Shared
    DictionaryFilter     *DictionaryFilter
    LLMFalsePositiveTest *LLMValidationPrompt
}
```

**Problems**:
- Type field creates artificial separation
- Different code paths for metadata vs content vs jsonpath
- Redundant fields (`FieldPath` vs `JSONPathExpr`)
- Inconsistent naming (`ProximityFilter` vs `LLMFalsePositiveTest`)

---

### NEW Structure (unified, no Type)

```go
type ExtractionRule struct {
    // Universal addressing (OPTIONAL)
    JSONPath         string                   `json:"jsonpath,omitempty"`

    // Extraction methods (choose ONE)
    PhraseList       []string                 `json:"phrase_list,omitempty"`     // Exact string matching
    InstanceName     string                   `json:"instance_name,omitempty"`   // Regex with (?P<name>...)

    // Pre-filters (all OPTIONAL, AND logic, fail-fast)
    Pattern          string                   `json:"pattern,omitempty"`         // Cheap regex pre-filter
    Proximity        *ProximityFilter         `json:"proximity,omitempty"`       // Co-occurrence
    Dictionary       *DictionaryFilter        `json:"dictionary,omitempty"`      // Linguistic validation
    Semantic         *SemanticFilter          `json:"semantic,omitempty"`        // Embedding similarity
    LLMValidation    *LLMValidationPrompt     `json:"llm_validation,omitempty"`  // LLM filtering
}
```

**Benefits**:
- Single code path for all extraction
- Clear field purposes
- Consistent naming (removed `_Filter` suffix except where type name requires it)
- Optional fields compose flexibly

---

## Boolean Logic Model

### Level 1: Multiple Entity Definitions (OR)

```yaml
# Definition 1 - high confidence, specific pattern
- entity_type: physician
  domain: medical
  confidence: 0.95
  extraction_rules:
    - instance_name: (?P<name>Dr\. [A-Z][a-z]+ [A-Z][a-z]+)
      proximity:
        keywords: [patient, diagnosis]

# Definition 2 - medium confidence, broader pattern
- entity_type: physician
  domain: medical
  confidence: 0.80
  extraction_rules:
    - instance_name: (?P<name>[A-Z][a-z]+ [A-Z][a-z]+, MD)
      semantic:
        reference_concepts: [medical practice]
```

**Extraction Logic**:
1. Try Definition 1 on element → extract if matches
2. Try Definition 2 on element → extract if matches
3. If both match same entity → dedup at graph phase (keep higher confidence)

**This is OR logic**: Definition 1 **OR** Definition 2

---

### Level 2: Filters Within ONE Rule (AND - fail fast)

```yaml
extraction_rules:
  - jsonpath: $.content               # Filter 0 (optional)
    phrase_list: [Dr. Smith]          # Filter 1 (optional, OR within list)
    pattern: \b[A-Z]                  # Filter 2 (optional)
    proximity:                        # Filter 3 (optional, OR within keywords)
      keywords: [patient, diagnosis]
    dictionary:                       # Filter 4 (optional)
      require_unknown_words: true
    semantic:                         # Filter 5 (optional, OR within concepts)
      reference_concepts: [medical]
```

**Execution Order** (fail-fast AND):
1. If `jsonpath` present → evaluate, FAIL FAST if no match
2. If `phrase_list` present → match, FAIL FAST if no match
3. If `instance_name` present → extract, FAIL FAST if empty
4. If `pattern` present → validate, FAIL FAST if no match
5. If `proximity` present → check, FAIL FAST if no match
6. If `dictionary` present → validate, FAIL FAST if no match
7. If `semantic` present → check similarity, FAIL FAST if below threshold
8. ALL filters passed → extract entity

**This is AND logic**: Filter 0 **AND** Filter 1 **AND** Filter 2 **AND** ... **AND** Filter N

---

### Level 3: OR Within Individual Filters

**PhraseList** (implicit OR):
```yaml
phrase_list: [Dr. Smith, Dr. Jones, Dr. Brown]
```
Match "Dr. Smith" **OR** "Dr. Jones" **OR** "Dr. Brown"

**Proximity Keywords** (implicit OR):
```yaml
proximity:
  keywords: [patient, diagnosis, surgery]
  max_distance: 100
```
Appear near "patient" **OR** "diagnosis" **OR** "surgery"

**Semantic Concepts** (implicit OR):
```yaml
semantic:
  reference_concepts:
    - medical practice
    - healthcare provider
  similarity_threshold: 0.70
```
Similar to "medical practice" **OR** "healthcare provider"

**Regex Alternation** (built-in OR):
```yaml
instance_name: (?P<name>Dr\.|Prof\.|Mr\.) [A-Z][a-z]+
```
Match "Dr." **OR** "Prof." **OR** "Mr."

---

## Performance-Optimized Pipeline

### Step-by-Step Execution

```go
func (e *RuleBasedExtractor) tryExtractWithRule(mapping ElementEntityMapping, rule ExtractionRule, elem Element) *Entity {
    // STEP 1: JSONPath validation (OPTIONAL - instant if omitted)
    var sourceData string
    if rule.JSONPath != "" {
        elemJSON := e.elementToJSON(&elem)
        results := e.evaluateJSONPath(elemJSON, rule.JSONPath)
        if len(results) == 0 {
            return nil // FAIL FAST
        }
        sourceData = fmt.Sprint(results[0])
    } else {
        // Fast path: extract from content directly (no JSON conversion)
        sourceData = e.resolveContent(&elem)
    }

    if sourceData == "" {
        return nil // FAIL FAST
    }

    // STEP 2: Phrase list matching (OPTIONAL - 10-100x faster than regex)
    var entityName string
    if len(rule.PhraseList) > 0 {
        entityName = e.findPhraseMatch(sourceData, rule.PhraseList)
        if entityName == "" {
            return nil // FAIL FAST
        }
    } else if rule.InstanceName != "" {
        // STEP 3: Instance name extraction (regex fallback)
        entityName = e.extractInstanceName(sourceData, rule, sourceData)
        if entityName == "" {
            return nil // FAIL FAST
        }
    } else {
        return nil // No extraction method specified
    }

    // STEP 4: Pattern filter (OPTIONAL - cheap regex)
    if rule.Pattern != "" {
        matched, _ := regexp.MatchString(rule.Pattern, sourceData)
        if !matched {
            return nil // FAIL FAST
        }
    }

    // STEP 5: Proximity filter (OPTIONAL - moderate cost)
    if rule.Proximity != nil {
        if !e.checkProximityFilter(sourceData, entityName, rule.Proximity) {
            return nil // FAIL FAST
        }
    }

    // STEP 6: Dictionary filter (OPTIONAL - moderate cost)
    if rule.Dictionary != nil {
        if !e.checkDictionaryFilter(entityName, rule.Dictionary) {
            return nil // FAIL FAST
        }
    }

    // STEP 7: Semantic filter (OPTIONAL - expensive)
    if rule.Semantic != nil {
        if !e.checkSemanticFilter(&elem, rule.Semantic) {
            return nil // FAIL FAST
        }
    }

    // STEP 8: Create entity (LLM validation happens during canonicalization, not here)
    return &Entity{
        ID:         e.generateID("ent"),
        Name:       entityName,
        Type:       EntityType(mapping.EntityType),
        Domain:     mapping.Domain,
        Confidence: mapping.Confidence,
        ...
    }
}
```

**Performance Optimizations**:
- ✅ Short-circuit on first failure (fail-fast AND)
- ✅ Phrase matching before regex (10-100x faster)
- ✅ Cheap filters before expensive (pattern → proximity → dictionary → semantic)
- ✅ Skip JSON conversion if `jsonpath` omitted
- ✅ LLM validation batched separately (not in hot path)

---

## Migration Examples

### Example 1: Metadata Field → JSONPath

**OLD**:
```yaml
- type: metadata_field
  field_path: author.name
```

**NEW**:
```yaml
- jsonpath: $.metadata.author.name
  instance_name: (?P<name>.+)
```

---

### Example 2: JSONPath Query → JSONPath

**OLD**:
```yaml
- type: jsonpath_query
  jsonpath_expr: $.content.company_info.ticker
```

**NEW**:
```yaml
- jsonpath: $.content.company_info.ticker
  instance_name: (?P<name>[A-Z]{1,5})
```

---

### Example 3: Content Extraction → Unified

**OLD**:
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

**NEW**:
```yaml
- instance_name: (?P<name>Dr\. .+)
  proximity:
    keywords: [patient, diagnosis]
    max_distance: 100
  semantic:
    reference_concepts: [medical practice]
    similarity_threshold: 0.70
```

**Changes**:
- Removed `type` field
- Renamed `proximity_filter` → `proximity`
- Renamed `cooccurrence_terms` → `keywords`
- Renamed `semantic_filter` → `semantic`

---

### Example 4: Keyword List → Phrase List

**OLD**:
```yaml
- type: content_extraction
  keywords: [CEO, CFO, CTO, President]
```

**NEW**:
```yaml
- phrase_list: [CEO, CFO, CTO, President]
```

**Benefits**:
- Faster (exact string matching vs regex)
- Clearer semantics (phrase_list vs keywords)

---

## Field Reference

### JSONPath (optional)
- **Purpose**: Universal addressing of element structure
- **Default**: `$.content` (extract from element content)
- **Examples**:
  - `$.metadata.author.name` - Navigate to nested metadata
  - `$.content.sections[0].title` - Array indexing
  - `$.content..heading` - Recursive descent
- **Performance**: Skip conversion if omitted (fast path)

### PhraseList (optional, OR within list)
- **Purpose**: Exact string matching (10-100x faster than regex)
- **Use when**: Known entity names, closed vocabulary
- **Examples**:
  - `[Dr. Smith, Dr. Jones]` - Specific names
  - `[CEO, CFO, CTO]` - Role titles
- **Returns**: Longest matching phrase

### InstanceName (optional, regex with capture group)
- **Purpose**: Extract entity name using regex pattern
- **Required**: Must have `(?P<name>...)` named capture group
- **Examples**:
  - `(?P<name>Dr\. [A-Z][a-z]+ [A-Z][a-z]+)` - Title + name
  - `(?P<name>[A-Z]{2,5})` - Stock ticker
- **Fallback**: If `phrase_list` doesn't match, try `instance_name`

### Pattern (optional, cheap pre-filter)
- **Purpose**: Quick regex validation before extraction
- **Use when**: Can cheaply eliminate non-matches
- **Examples**:
  - `\b[A-Z]` - Must contain uppercase letter
  - `\d{3}-\d{4}` - Phone number format
- **Performance**: Evaluated before expensive filters

### Proximity (optional, moderate cost)
- **Purpose**: Entity must appear near certain terms
- **Fields**:
  - `keywords` (OR within list): Terms to search for
  - `max_distance`: Maximum distance in words/chars
  - `distance_unit`: "element" (default), "word", "character"
- **Example**:
  ```yaml
  proximity:
    keywords: [patient, diagnosis, surgery]
    max_distance: 100
    distance_unit: word
  ```

### Dictionary (optional, moderate cost)
- **Purpose**: Linguistic validation using dictionary lookups
- **Fields**:
  - `require_unknown_words`: At least one proper noun (not in dict)
  - `max_known_words_ratio`: Max ratio of common words (0.0-1.0)
  - `reject_if_all_pos`: Reject if all words match POS (e.g., `["noun"]`)
  - `reject_if_all_categories`: Reject categories (e.g., `["place"]`)
- **Example**:
  ```yaml
  dictionary:
    require_unknown_words: true
    max_known_words_ratio: 0.5
    reject_if_all_categories: [place, temporal]
  ```

### Semantic (optional, expensive)
- **Purpose**: Element-level embedding similarity
- **Fields**:
  - `reference_concepts` (OR within list): Concepts to compare against
  - `similarity_threshold`: Minimum cosine similarity (0.0-1.0)
- **Example**:
  ```yaml
  semantic:
    reference_concepts:
      - medical practice
      - healthcare provider
    similarity_threshold: 0.70
  ```
- **Performance**: Most expensive filter, evaluated last

### LLMValidation (optional, batched separately)
- **Purpose**: LLM-based false positive filtering
- **Fields**:
  - `prompt`: Validation question
  - `batch_size`: Batch size for API calls (default: 50)
- **Example**:
  ```yaml
  llm_validation:
    prompt: "Is '{entity}' a valid person name?"
    batch_size: 50
  ```
- **Note**: Applied during canonicalization, not during extraction (batched for performance)

---

## Validation Rules

### Required
- At least ONE extraction method: `phrase_list` OR `instance_name`
- If `instance_name` used: Must have `(?P<name>...)` capture group

### Optional
- All filters are optional
- Filters compose with AND logic
- No `type` field allowed (will fail validation)

### Renamed Fields
- ❌ `proximity_filter` → ✅ `proximity`
- ❌ `semantic_filter` → ✅ `semantic`
- ❌ `dictionary_filter` → ✅ `dictionary`
- ❌ `llm_false_positive_test` → ✅ `llm_validation`
- ❌ `cooccurrence_terms` → ✅ `keywords` (inside `proximity`)
- ❌ `field_path` → ✅ `jsonpath` (use `$.metadata.field.path`)
- ❌ `jsonpath_expr` → ✅ `jsonpath`

---

## Entity Rule Tracking

### Requirement

**All entities must track which rule(s) extracted them** - not just at the Mention level, but at the Entity level.

### Purpose

- **Provenance**: Know exactly how each entity was discovered
- **Debugging**: Trace extraction decisions back to specific rules
- **Mentions**: When multiple definitions extract same entity (duplicates), track all rules that matched
- **Canonical entities**: After canonicalization, entity tracks rules from all merged duplicates

### Entity Structure Update

**Add field to Entity struct**:

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
    ExtractedByRules []string  `json:"extracted_by_rules"`  // Rule IDs or descriptions

    CreatedAt  time.Time
    UpdatedAt  time.Time
}
```

### Implementation

**During extraction** (`tryExtractWithRule`):

```go
func (e *RuleBasedExtractor) tryExtractWithRule(ctx context.Context, mapping ElementEntityMapping, rule ExtractionRule, elem Element) *Entity {
    // ... extraction logic ...

    // All filters passed - create entity
    entity := &Entity{
        ID:         e.generateID("ent"),
        Name:       entityName,
        Type:       EntityType(mapping.EntityType),
        Domain:     mapping.Domain,
        Confidence: mapping.Confidence,
        ElementID:  elem.ElementID,

        // NEW: Track which rule extracted this entity
        ExtractedByRules: []string{
            fmt.Sprintf("%s.%s[conf=%.2f]", mapping.Domain, mapping.EntityType, mapping.Confidence),
        },

        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
    }

    return entity
}
```

**During canonicalization** (`canonicalizeEntities`):

```go
func (e *RuleBasedExtractor) canonicalizeEntities(extractedEntities []Entity) ([]Entity, error) {
    canonicalIndex := make(map[string]*Entity)

    for _, entity := range extractedEntities {
        key := entity.Name + "|" + string(entity.Type) + "|" + entity.ElementID

        if canonical, exists := canonicalIndex[key]; exists {
            // Merge: Keep highest confidence, combine mentions AND rules
            if entity.Confidence > canonical.Confidence {
                canonical.Confidence = entity.Confidence
            }
            canonical.Mentions = append(canonical.Mentions, entity.Mentions...)

            // NEW: Merge rules from duplicate entity
            canonical.ExtractedByRules = append(canonical.ExtractedByRules, entity.ExtractedByRules...)

        } else {
            entityCopy := entity
            canonicalIndex[key] = &entityCopy
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

### Example

**Extraction** (3 definitions match same entity):
```
Entity: surgeon "Dr. Smith"
  ExtractedByRules: [
    "medical.surgeon[conf=0.90]",  // high_confidence rule
    "medical.surgeon[conf=0.80]",  // generic rule
    "medical.surgeon[conf=0.85]"   // context_aware rule
  ]
```

**After canonicalization**:
```
Canonical Entity: surgeon "Dr. Smith"
  Confidence: 0.90  // highest
  ExtractedByRules: [
    "medical.surgeon[conf=0.90]",
    "medical.surgeon[conf=0.80]",
    "medical.surgeon[conf=0.85]"
  ]
  Mentions: [merged from all 3 extractions]
```

### Benefits

1. **Full provenance**: Know all extraction paths that found this entity
2. **Rule effectiveness**: Analyze which rules are most productive
3. **Debugging**: Trace why entity was extracted
4. **Progressive refinement**: See how multiple definitions work together

---

## Next Steps

Proceed to:
- [04_HIERARCHY_SYSTEM.md](04_HIERARCHY_SYSTEM.md) - Learn hierarchy computation and canonicalization
- [05_VALIDATION.md](05_VALIDATION.md) - Understand validation rules
- [06_IMPLEMENTATION_STEPS.md](06_IMPLEMENTATION_STEPS.md) - Start implementing