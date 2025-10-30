# Ontology Schema Quality Improvements

## Overview

This document outlines improvements needed to address critical weaknesses in LLM-generated ontology schemas, categorized by solution approach: human intervention, feature enhancements, and LLM prompt improvements.

---

## Critical Weaknesses Identified

### 1. Overly Generic Regex Patterns
- **Issue**: Patterns like `[A-Z][a-z]+(?:\s+[A-Z][a-z]+)*` match ANY capitalized text
- **Examples**: location, city, country, region all use identical patterns
- **Impact**: Massive false positive rate - extracts random capitalized words as entities

### 2. No Semantic Filtering
- **Issue**: Only syntactic pattern matching without contextual understanding
- **Impact**: Cannot distinguish "Apple" (company) from "apple" (fruit) or "Washington" (person) from "Washington" (location)

### 3. Arbitrary Confidence Levels
- **Issue**: No correlation between pattern specificity and confidence scores
- **Example**: Generic person pattern (0.75) vs specific taxonomic pattern (0.92)
- **Impact**: Misleading confidence scores don't reflect actual extraction quality

### 4. Pattern Duplication Without Disambiguation
- **Issue**: Same regex patterns reused across multiple entity types
- **Examples**: Person patterns duplicated 6+ times, location patterns 8+ times
- **Impact**: Multiple conflicting matches for same text with no resolution strategy

### 5. Missing Critical Features
- No negative patterns (what NOT to match)
- No context window requirements
- No post-extraction validation rules
- No pattern precedence/priority when multiple rules match

### 6. Hardcoded Value Lists
- **Issue**: Fixed country names, anatomical terms in patterns
- **Impact**: Limited to predefined vocabularies, won't adapt to new content

### 7. Excessive Proximity Distances in Relationship Rules
- **Issue**: 24% of proximity rules use max_distance > 50 tokens (ranging from 55-100 tokens)
- **Examples**: Rules with distances of 100, 80, 75, 70, 65, 60, 55 tokens
- **Impact**: Entities 100 tokens apart are essentially unrelated - creates spurious relationship matches
- **Root Cause**: No guidance in LLM prompts on appropriate distance ranges
- **Recommendation**: Most proximity rules should be ≤ 30 tokens; use structural patterns for longer ranges

### 8. Other Relationship Pattern Issues
- Text templates too specific - will miss variations
- Structural patterns vague (e.g., "section contains")

---

## Solution Approaches

### 1. Human Intervention Required

#### Arbitrary Confidence Levels
- **Action**: User domain expertise needed to validate confidence scores
- **Solution**: Prompt user to review/adjust confidence during ontology approval phase
- **Implementation**: Add confidence review step to ontology interview process

#### Hardcoded Vocabularies
- **Action**: User should provide extensible vocabularies (gazetteers)
- **Solution**: Support external gazetteer files instead of hardcoded lists
- **Implementation**: Add gazetteer import functionality to ontology builder

---

### 2. Feature Improvements Needed

#### Missing Critical Features

##### A. Negative Patterns
Add `exclusion_patterns` field to extraction rules:
```go
type ExtractionRule struct {
    Type              string   `json:"type"`
    Pattern           string   `json:"pattern,omitempty"`
    ExclusionPatterns []string `json:"exclusion_patterns,omitempty"` // NEW
    // ... existing fields
}
```

##### B. Pattern Precedence
Add `priority` field to rules (higher priority wins conflicts):
```go
type ExtractionRule struct {
    Type              string   `json:"type"`
    Pattern           string   `json:"pattern,omitempty"`
    Priority          int      `json:"priority,omitempty"` // NEW (default: 0)
    // ... existing fields
}
```

##### C. Post-Extraction Validation
Add `validation_rules` field to entity mappings:
```go
type EntityMapping struct {
    EntityType       string            `json:"entity_type"`
    ExtractionRules  []ExtractionRule  `json:"extraction_rules"`
    ValidationRules  *ValidationRules  `json:"validation_rules,omitempty"` // NEW
    // ... existing fields
}

type ValidationRules struct {
    MinLength       int      `json:"min_length,omitempty"`
    MaxLength       int      `json:"max_length,omitempty"`
    RequiredContext []string `json:"required_context,omitempty"` // Keywords that must appear nearby
    ForbiddenTokens []string `json:"forbidden_tokens,omitempty"` // Tokens that invalidate match
}
```

##### D. Entity Disambiguation
Implement multi-pass extraction with conflict resolution:
```go
func (b *Builder) validateEntityMappings() []ValidationWarning {
    warnings := []ValidationWarning{}

    // Check for duplicate patterns
    patternMap := make(map[string][]string)
    for _, mapping := range b.schema.ElementEntityMappings {
        for _, rule := range mapping.ExtractionRules {
            if rule.InstanceName != "" {
                patternMap[rule.InstanceName] = append(patternMap[rule.InstanceName], mapping.EntityType)
            }
        }
    }

    for pattern, types := range patternMap {
        if len(types) > 1 {
            warnings = append(warnings, ValidationWarning{
                Severity: "HIGH",
                Message: fmt.Sprintf("Pattern %s used by multiple entity types: %v - add disambiguation filters", pattern, types),
            })
        }
    }

    return warnings
}
```

---

### 3. LLM Prompt Improvements (MOST CRITICAL)

#### A. Enhanced CRITICAL INSTRUCTIONS

**Location**: `builder.go:2382-2388`

Add three new mandatory instructions:

```
**CRITICAL INSTRUCTIONS**:
1. Generate AT LEAST 3-6 extraction patterns per relationship rule (more patterns = more matches)
2. For each entity type pair (e.g., person-organization), create 2-4 relationship rules with different confidence levels
3. Use COMPREHENSIVE signal word/keyword lists (aim for 10-30 words per proximity/cooccurrence pattern)
4. Consider synonyms, variations, and domain-specific terminology in all patterns
5. Use entity constraints (source_constraints/target_constraints) when relationships only apply to specific entity subtypes
6. MUST use source_constraints/target_constraints when relationship applies to entity subtypes (e.g., only academics→universities, not all people→organizations)
7. **NEW** Regex patterns MUST be specific enough to avoid false positives - use context-aware patterns, not just capitalization
8. **NEW** When multiple entity types could match same pattern (e.g., person vs organization), use proximity_filter or semantic_filter to disambiguate
9. **NEW** Assign confidence based on pattern specificity:
   - Highly specific patterns (0.85-0.95)
   - Context-dependent (0.75-0.85)
   - Generic patterns (0.60-0.75)
```

#### B. Enhanced ENTITY CONSTRAINTS Section

**Location**: `builder.go:2429-2449`

Emphasize semantic_filter for disambiguation:

```
**Constraint Fields** (all optional, applied in order):
1. pattern (string): Pre-filter regex - entity name must match (e.g., "\\b(Dr|Professor)\\b.*" for academics)
2. proximity_filter (object): Co-occurrence filter on entity context
   - required_keywords: Keywords that must appear in entity's context
   - window_size: Context window in tokens
3. instance_name (string): Named capture regex - entity name must match
4. **semantic_filter (object): RECOMMENDED for disambiguation** - Embedding similarity on entity context
   - query: Semantic query describing the entity subtype
   - similarity_threshold: Minimum similarity (0.0-1.0)
   - **Example use cases:**
     * Disambiguate "Washington" (person) vs "Washington" (location): Use query="person name, historical figure" vs query="city, geographic location"
     * Disambiguate "Apple" (company) vs "apple" (fruit): Use query="technology company, corporation" vs query="food, fruit"
     * Distinguish entity types with similar surface forms: person vs organization, product vs concept

**IMPORTANT**: When regex patterns are generic (match capitalized text), you MUST add semantic_filter or proximity_filter to reduce false positives.
```

#### C. New Section: PATTERN SPECIFICITY & DISAMBIGUATION

**Location**: After ENTITY CONSTRAINTS section (`builder.go:~2450`)

Add comprehensive disambiguation guidance:

```
## PATTERN SPECIFICITY & DISAMBIGUATION

**Avoid Generic Patterns**: Patterns like `[A-Z][a-z]+(?:\s+[A-Z][a-z]+)*` match ANY capitalized text (names, titles, random words).

**Disambiguation Strategies** (use at least one):

### 1. Contextual Keywords (proximity_filter)
```json
{
  "instance_name": "(?P<name>[A-Z][a-z]+(?:\\s+[A-Z][a-z]+)*)",
  "proximity_filter": {
    "required_keywords": ["city", "located", "municipality", "population"],
    "window_size": 50
  }
}
```

### 2. Semantic Context (semantic_filter)
```json
{
  "instance_name": "(?P<name>[A-Z][a-z]+(?:\\s+[A-Z][a-z]+)*)",
  "semantic_filter": {
    "query": "person name, individual, human being",
    "similarity_threshold": 0.70
  }
}
```

### 3. Prefix/Suffix Indicators (more specific regex)
```json
{
  "instance_name": "(?:Dr\\.?|Professor|Mr\\.?|Ms\\.?|Mrs\\.?)\\s+(?P<name>[A-Z][a-z]+(?:\\s+[A-Z][a-z]+)*)"
}
```

### 4. Structured Context (co-occurrence patterns)
```json
{
  "type": "cooccurrence",
  "entity_keywords": ["person_name_pattern"],
  "context_keywords": ["said", "stated", "according to", "interviewed"],
  "max_distance": 20
}
```

**Confidence Assignment Guidelines**:
- **0.90-0.95**: Highly specific patterns with structural markers
  - Examples: `DOI: 10.xxxx/yyyy`, binomial nomenclature with italics
- **0.80-0.90**: Context-dependent patterns with semantic_filter or proximity_filter
  - Requirement: similarity_threshold >= 0.75
- **0.70-0.80**: Moderate specificity with keyword co-occurrence
  - Requirement: 5+ required_keywords
- **0.60-0.70**: Generic patterns with weak context
  - Use as fallback only, not primary extraction
```

---

## Implementation Priorities

### Phase 1: LLM Prompt Improvements (Immediate) ✅ COMPLETED
1. ✅ Add instructions #7-9 to CRITICAL INSTRUCTIONS (builder.go:2389-2391)
2. ✅ Enhance ENTITY CONSTRAINTS section with semantic_filter emphasis (builder.go:2438-2446)
3. ✅ Add new PATTERN SPECIFICITY & DISAMBIGUATION section (builder.go:2477-2485)
4. ✅ Test with constraint_usage_test to verify improvements (test passed - schema validated successfully)

### Phase 2: Validation & Quality Checks (Short-term) ✅ COMPLETED
1. ✅ Implement `ValidateSchemaQuality()` function (validation.go)
2. ✅ Add duplicate pattern detection (checkDuplicatePatterns)
3. ✅ Add confidence-specificity correlation checks (checkConfidenceSpecificityCorrelation)
4. ✅ Display warnings during ontology approval phase (interview.go integration)
5. ✅ Add proximity distance validation (checkExcessiveProximityDistance):
   - MEDIUM severity warning for distances > 50 tokens
   - HIGH severity warning for distances > 75 tokens
   - Suggest using structural patterns for long-range relationships

### Phase 2.5: Entity Type Uniqueness (Critical) ✅ COMPLETED
**Problem**: Multiple entity_type definitions with same name across domains (e.g., 'person' defined 5 times) causing extraction conflicts

**Solution Principles**:
- Minimize qualification - only qualify when conflicts exist
- Prefer simple, global entity definitions
- Reuse global entities by default
- If conflict with global: If unique enough → qualify + add parent; If not unique → remove
- If conflict with non-global: Qualify both entity names

**Implementation Tasks**:
1. ✅ Add `checkDuplicateEntityTypes()` validation function (validation.go:365-394)
2. ✅ Display CRITICAL severity warnings for duplicate entity_type names
3. ✅ Integrated into ValidateSchemaQuality() pipeline (validation.go:39)
4. ✅ Update LLM prompts with entity reuse guidance (builder.go:1730-1737):
   - Instructs to reuse global entities by default
   - Only create domain-specific qualified subtypes if they provide clear value
   - Format: `{domain}_{entity_name}` (e.g., `banking_account`, `medical_procedure`)
5. ✅ Parent field already exists in ElementEntityMappingConfig (types.go:99)
6. ⏳ Implement parent reference validation (ensure parent exists if specified) - DEFERRED to Phase 3

**Example Conflict Resolution**:
```yaml
# BEFORE (Conflict):
- entity_type: person
  domain: global
- entity_type: person  # CONFLICT!
  domain: banking

# AFTER (Resolved):
- entity_type: person
  domain: global
- entity_type: banking_customer  # Qualified name
  parent: global.person           # Inheritance reference
  domain: banking
```

**Note**: Do not use "person" as entity name in code or LLM prompts - it's only an example here.

### Phase 3: Feature Enhancements (Medium-term)
1. Add `exclusion_patterns` field to schema
2. Add `priority` field for pattern precedence
3. Implement `ValidationRules` for post-extraction filtering
4. Add gazetteer import functionality

### Phase 4: Advanced Disambiguation (Long-term)
1. Multi-pass entity extraction with conflict resolution
2. Cross-entity validation rules
3. Automated pattern optimization based on corpus statistics
4. Interactive pattern refinement UI

---

## Diagnostic Commands

### Find Duplicate Patterns
```bash
grep -A5 "instance_name:" validation_prefill_test.yaml | \
  awk '/entity_type:/{type=$2} /instance_name:/{patterns[type]=$0} END{for(t in patterns) print t, patterns[t]}' | \
  sort -k2 | uniq -f1 -D
```

### Check Semantic Filter Usage
```bash
grep -c "semantic_filter:" validation_prefill_test.yaml
```

### Analyze Confidence Distribution
```bash
grep "confidence:" validation_prefill_test.yaml | \
  awk '{print $2}' | sort | uniq -c | sort -rn
```

---

## Success Metrics

### Short-term (after LLM prompt improvements)
- [ ] At least 30% of entity mappings use semantic_filter or proximity_filter
- [ ] No duplicate patterns without disambiguation filters
- [ ] Confidence scores correlate with pattern specificity (0.7+ correlation)

### Medium-term (after validation features)
- [ ] Zero HIGH severity validation warnings in generated schemas
- [ ] 80%+ entity mappings have context-aware patterns
- [ ] False positive rate < 10% on test corpus

### Long-term (after full implementation)
- [ ] Automated pattern optimization reduces manual review time by 50%
- [ ] Entity extraction precision > 0.85, recall > 0.80
- [ ] User satisfaction score > 8/10 for ontology quality

---

## Related Files

- **Prompt generation**: `go/internal/udml/ontology/builder.go`
- **Schema types**: `go/internal/udml/ontology/types.go`
- **Test output**: `tests/test_output/ontology_results/validation_prefill_test.yaml`
- **Debug relationships**: `tests/test_output/ontology_debug/intra_domain_relationships_*.txt`

---

**Last Updated**: 2025-10-30
**Status**: Phase 1, Phase 2, and Phase 2.5 Complete ✅
**Next Action**: Phase 3 Feature Enhancements (exclusion_patterns, priority, ValidationRules, gazetteer import)
