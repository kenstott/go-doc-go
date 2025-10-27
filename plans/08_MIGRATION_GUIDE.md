# Migration Guide

## Overview

This refactoring introduces **breaking changes**. No automated migration is provided - catalogs must be manually updated.

---

## Breaking Changes

### 1. Type Field Removed

**Impact**: All extraction rules with `type` field will fail validation.

**Old**:
```yaml
- type: content_extraction
  instance_name: (?P<name>.+)
```

**New**:
```yaml
- instance_name: (?P<name>.+)
```

**Migration**: Remove `type` field from all extraction rules.

---

### 2. Filter Field Renames

**Impact**: Old filter names will not be recognized.

**Changes**:
| Old Name | New Name |
|----------|----------|
| `proximity_filter` | `proximity` |
| `cooccurrence_terms` | `keywords` (inside `proximity`) |
| `semantic_filter` | `semantic` |
| `dictionary_filter` | `dictionary` |
| `llm_false_positive_test` | `llm_validation` |

**Old**:
```yaml
proximity_filter:
  cooccurrence_terms: [patient, diagnosis]
  max_distance: 100
semantic_filter:
  reference_concepts: [medical]
  similarity_threshold: 0.70
```

**New**:
```yaml
proximity:
  keywords: [patient, diagnosis]
  max_distance: 100
semantic:
  reference_concepts: [medical]
  similarity_threshold: 0.70
```

---

### 3. Metadata/JSONPath Unification

**Impact**: `field_path` and `jsonpath_expr` fields removed.

**Old (metadata_field)**:
```yaml
- type: metadata_field
  field_path: author.name
```

**New**:
```yaml
- jsonpath: $.metadata.author.name
  instance_name: (?P<name>.+)
```

**Old (jsonpath_query)**:
```yaml
- type: jsonpath_query
  jsonpath_expr: $.content.company_info.ticker
```

**New**:
```yaml
- jsonpath: $.content.company_info.ticker
  instance_name: (?P<name>[A-Z]{1,5})
```

---

### 4. New Required Fields

**Impact**: Schemas missing these fields will fail validation.

**Required additions**:
- `parent_type` (optional, but recommended for hierarchies)
- `w_category` (who/what/where/when/why)

**Example**:
```yaml
- entity_type: physician
  domain: medical
  parent_type: global.person    # NEW (optional)
  w_category: who               # NEW (required)
  description: Medical doctor
  confidence: 0.90
```

---

### 5. Global Domain Restructure

**Impact**: `catalogs/common.go` deleted, replaced with 5 files.

**Old location**:
```
catalogs/common.go (50+ entity templates)
```

**New location**:
```
catalogs/
  ├─ global_who.go
  ├─ global_what.go
  ├─ global_where.go
  ├─ global_when.go
  └─ global_why.go
```

**Migration**: No action needed for users - this is internal restructure.

---

## Domain Catalog Update Procedure

### Step 1: Backup Existing Catalogs

```bash
cp -r examples/ontologies examples/ontologies.backup
```

### Step 2: Update Each Catalog File

For each file in `examples/ontologies/`:

1. **Remove `type` field**:
   ```bash
   # Find all occurrences
   grep -rn "type:" examples/ontologies/
   
   # Remove manually or with sed (careful!)
   sed -i '' '/^[[:space:]]*type:/d' examples/ontologies/*.yaml
   ```

2. **Rename filter fields**:
   ```bash
   # proximity_filter → proximity
   sed -i '' 's/proximity_filter:/proximity:/g' examples/ontologies/*.yaml
   
   # cooccurrence_terms → keywords
   sed -i '' 's/cooccurrence_terms:/keywords:/g' examples/ontologies/*.yaml
   
   # semantic_filter → semantic
   sed -i '' 's/semantic_filter:/semantic:/g' examples/ontologies/*.yaml
   
   # dictionary_filter → dictionary
   sed -i '' 's/dictionary_filter:/dictionary:/g' examples/ontologies/*.yaml
   
   # llm_false_positive_test → llm_validation
   sed -i '' 's/llm_false_positive_test:/llm_validation:/g' examples/ontologies/*.yaml
   ```

3. **Add `parent_type` and `w_category`**:
   
   **Manual edit required** - must determine:
   - Which entities extend global types
   - Appropriate w_category for each entity
   
   **Example**:
   ```yaml
   # Medical domain
   - entity_type: physician
     domain: medical
     parent_type: global.person    # ADD THIS
     w_category: who               # ADD THIS
   
   - entity_type: condition
     domain: medical
     parent_type: ""               # No parent
     w_category: what              # ADD THIS
   ```

4. **Convert metadata/JSONPath rules**:
   
   **Find old patterns**:
   ```bash
   grep -A 2 "type: metadata_field" examples/ontologies/*.yaml
   grep -A 2 "type: jsonpath_query" examples/ontologies/*.yaml
   ```
   
   **Replace manually**:
   - `field_path: X` → `jsonpath: $.metadata.X`
   - `jsonpath_expr: X` → `jsonpath: X`
   - Add `instance_name` if missing

### Step 3: Validate Updated Catalogs

```bash
# Validate each catalog
for file in examples/ontologies/*.yaml; do
    echo "Validating $file..."
    go run ./cmd/ontology validate --schema "$file" || echo "FAILED: $file"
done
```

**Fix errors** reported by validation:
- Missing `instance_name` or `phrase_list`
- Invalid `instance_name` (missing `(?P<name>...)`)
- Broken `parent_type` references
- Missing `w_category`

---

## Example Migration: Medical Domain

### Before (medical.yaml)

```yaml
name: Medical Domain
version: 1.0
domains:
  - name: medical
    description: Healthcare and medical domain

element_entity_mappings:
  - entity_type: physician
    domain: medical
    description: Medical doctor
    confidence: 0.90
    element_types: [paragraph, div]
    extraction_rules:
      - type: content_extraction
        instance_name: (?P<name>Dr\. [A-Z][a-z]+ [A-Z][a-z]+)
        proximity_filter:
          cooccurrence_terms: [patient, diagnosis]
          max_distance: 100
        semantic_filter:
          reference_concepts: [medical practice]
          similarity_threshold: 0.70

  - entity_type: hospital
    domain: medical
    description: Medical facility
    confidence: 0.85
    element_types: [paragraph]
    extraction_rules:
      - type: metadata_field
        field_path: facility.name
```

### After (medical.yaml)

```yaml
name: Medical Domain
version: 2.0
domains:
  - name: medical
    description: Healthcare and medical domain

element_entity_mappings:
  - entity_type: physician
    domain: medical
    parent_type: global.person        # ADDED
    w_category: who                   # ADDED
    description: Medical doctor
    confidence: 0.90
    element_types: [paragraph, div]
    extraction_rules:
      - instance_name: (?P<name>Dr\. [A-Z][a-z]+ [A-Z][a-z]+)
        proximity:                    # RENAMED
          keywords: [patient, diagnosis]  # RENAMED
          max_distance: 100
        semantic:                     # RENAMED
          reference_concepts: [medical practice]
          similarity_threshold: 0.70

  - entity_type: hospital
    domain: medical
    parent_type: global.organization  # ADDED
    w_category: who                   # ADDED
    description: Medical facility
    confidence: 0.85
    element_types: [paragraph]
    extraction_rules:
      - jsonpath: $.metadata.facility.name  # CHANGED
        instance_name: (?P<name>.+)         # ADDED
```

**Changes**:
1. ✅ Removed `type` field
2. ✅ Renamed `proximity_filter` → `proximity`
3. ✅ Renamed `cooccurrence_terms` → `keywords`
4. ✅ Renamed `semantic_filter` → `semantic`
5. ✅ Added `parent_type: global.person` (physician extends person)
6. ✅ Added `parent_type: global.organization` (hospital extends organization)
7. ✅ Added `w_category: who` to both
8. ✅ Converted `field_path` → `jsonpath: $.metadata...`
9. ✅ Added `instance_name` to metadata rule
10. ✅ Bumped version to 2.0

---

## Relationship Rule Migration

### 6. Relationship Rules Field Rename

**Impact**: All domain catalogs with relationship rules must update field structure.

**Old Format**:
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

**New Format**:
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
```

**Changes Required**:
1. Rename `relationship_types` → `entity_relationship_rules`
2. Rename `relationship_type` (field name) → `name` (at top level)
3. Move `relationship_type` value to dedicated field (e.g., `treats`)
4. Rename `source_entity` → `source_entity_type`
5. Rename `target_entity` → `target_entity_type`
6. Add `confidence` field (0.0-1.0)
7. Rename `sample_rules` → `extraction_patterns`
8. Update pattern structure:
   - `type: keyword_match` → `type: proximity`
   - `keywords` → `signal_words`
   - Add `max_distance` and `direction`

**Migration Script**:
```bash
# Step 1: Rename top-level field
sed -i '' 's/^relationship_types:/entity_relationship_rules:/g' examples/ontologies/**/*.yaml

# Step 2: Rename entity fields
sed -i '' 's/source_entity:/source_entity_type:/g' examples/ontologies/**/*.yaml
sed -i '' 's/target_entity:/target_entity_type:/g' examples/ontologies/**/*.yaml

# Step 3: Rename sample_rules to extraction_patterns
sed -i '' 's/sample_rules:/extraction_patterns:/g' examples/ontologies/**/*.yaml

# Step 4: Rename keyword_match to proximity
sed -i '' 's/type: keyword_match/type: proximity/g' examples/ontologies/**/*.yaml

# Step 5: Rename keywords to signal_words
sed -i '' 's/keywords:/signal_words:/g' examples/ontologies/**/*.yaml
```

**Manual Updates Required**:
1. **Add `name` field**: Must be unique identifier (e.g., `drug_treats_indication`)
2. **Add `confidence` field**: Estimate pattern reliability (0.60-0.95)
3. **Convert `relationship_type` value to enum**: Map to RelationshipType constants
4. **Add pattern parameters**: `max_distance`, `direction` for proximity patterns

**Relationship Type Mapping**:
| Old Value | New Enum Constant |
|-----------|-------------------|
| `treats` | `treats` |
| `caused_by` | `caused_by` |
| `approved_by` | `approved_by` |
| `manufactured_by` | `manufactured_by` |
| `contains` | `contains` |
| `tested_in` | `tested_in` |

### Example Migration: Pharmaceutical Domain

**Before**:
```yaml
relationship_types:
  - relationship_type: treats_indication
    description: Drug treats indication
    source_entity: drug_product
    target_entity: indication
    sample_rules:
      - type: keyword_match
        keywords: [treats, indicated for, approved for]

  - relationship_type: causes_adverse_event
    description: Drug causes adverse event
    source_entity: drug_product
    target_entity: adverse_event
    sample_rules:
      - type: keyword_match
        keywords: [causes, associated with, side effect]
```

**After**:
```yaml
entity_relationship_rules:
  - name: drug_treats_indication
    source_entity_type: drug_product
    target_entity_type: indication
    relationship_type: treats
    description: Drug treats indication
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
    description: Drug causes adverse event
    confidence: 0.75
    extraction_patterns:
      - type: proximity
        signal_words: [causes, associated with, side effect, adverse event]
        max_distance: 100
        direction: forward
```

**Changes Applied**:
1. ✅ Renamed `relationship_types` → `entity_relationship_rules`
2. ✅ Added `name` field (unique identifier)
3. ✅ Renamed `source_entity` → `source_entity_type`
4. ✅ Renamed `target_entity` → `target_entity_type`
5. ✅ Moved `relationship_type` value to dedicated field
6. ✅ Added `confidence` scores (0.85, 0.75)
7. ✅ Renamed `sample_rules` → `extraction_patterns`
8. ✅ Changed `type: keyword_match` → `type: proximity`
9. ✅ Renamed `keywords` → `signal_words`
10. ✅ Added `max_distance` and `direction`

---

## W-Category Assignment Guide

### Who (Agent Entities)
- **People**: person, physician, executive, employee, patient
- **Organizations**: organization, hospital, company, university

### What (Object Entities)
- **Documents**: document, report, paper, publication
- **Identifiers**: email, phone, url, code, id_number
- **Roles**: CEO, professor, manager (positions, not people)
- **Domain objects**: condition, treatment, product, service

### Where (Location Entities)
- **Places**: location, city, country, region, address, building
- **Facilities**: hospital, clinic, office

### When (Temporal Entities)
- **Time**: date, time, duration, timestamp
- **Events**: event, occurrence, incident

### Why (Causal/Explanatory Entities)
- **Assertions**: requirement, claim, declaration, statement
- **Hypotheses**: theory, hypothesis, explanation

---

## Parent Type Assignment Guide

### Extending Global Types

**Extend `global.person`**:
- physician, surgeon, patient, nurse
- professor, student, researcher
- attorney, judge, lawyer
- author, journalist, editor

**Extend `global.organization`**:
- hospital, clinic, medical_center (→ global.organization.healthcare)
- university, college, school (→ global.organization.educational)
- court, agency, department (→ global.organization.government)
- company, corporation, startup (→ global.organization.business)

**Extend `global.location`**:
- office, facility, campus (→ global.location.building)
- state, province (→ global.location.region)

**Extend `global.date`**:
- appointment_time, deadline (→ global.date.time)
- treatment_period, project_duration (→ global.date.duration)

**Extend `global.identifier`**:
- patient_id, case_number (→ global.identifier.id_number)
- product_code, diagnosis_code (→ global.identifier.code)

### Domain-Specific (No Parent)

**Medical**:
- condition, symptom, treatment, diagnosis (no global equivalent)

**Legal**:
- statute, case_law, precedent (no global equivalent)

**Financial**:
- stock, bond, derivative (no global equivalent)

---

## Validation Checklist

After migration, verify:

**Entity Extraction Rules:**
- [ ] No `type` field in extraction rules
- [ ] All filter fields renamed correctly (`proximity_filter` → `proximity`, etc.)
- [ ] All entities have `w_category`
- [ ] Entities extending global types have `parent_type`
- [ ] All `instance_name` have `(?P<name>...)` capture group
- [ ] Metadata rules use `jsonpath: $.metadata...`

**Relationship Rules:**
- [ ] `relationship_types` renamed to `entity_relationship_rules`
- [ ] All rules have `name` field (unique identifier)
- [ ] All rules have `confidence` field (0.0-1.0)
- [ ] `source_entity` renamed to `source_entity_type`
- [ ] `target_entity` renamed to `target_entity_type`
- [ ] `sample_rules` renamed to `extraction_patterns`
- [ ] Pattern types updated (`keyword_match` → `proximity`)
- [ ] Pattern fields updated (`keywords` → `signal_words`)
- [ ] `max_distance` and `direction` added to proximity patterns

**Overall Validation:**
- [ ] Schema validates successfully (`go run ./cmd/ontology validate --schema <path>`)
- [ ] Test extraction produces expected results
- [ ] Relationship extraction creates domain relationships only between extracted entities

---

## Rollback Plan

If migration fails:

1. **Restore backup**:
   ```bash
   rm -rf examples/ontologies
   mv examples/ontologies.backup examples/ontologies
   ```

2. **Revert code changes**:
   ```bash
   git checkout go/internal/udml/ontology/types.go
   git checkout go/internal/udml/ontology/extractor.go
   git checkout go/internal/udml/ontology/catalogs/
   ```

3. **Rebuild old binary**:
   ```bash
   go build -o bin/ontology_old ./cmd/ontology
   ```

---

## Support

**Common issues**:

1. **Validation error: "must have phrase_list or instance_name"**
   - Add `instance_name: (?P<name>.+)` to rule

2. **Validation error: "instance_name must contain (?P<name>...)"**
   - Fix regex to include named capture group: `(?P<name>pattern)`

3. **Validation error: "references non-existent parent"**
   - Check `parent_type` spelling
   - Verify parent entity exists in schema

4. **Validation error: "circular hierarchy detected"**
   - Check for loops in `parent_type` chain
   - Break cycle by removing one `parent_type`

---

## Timeline

**Recommended migration schedule**:

1. **Week 1**: Update global domain catalogs (Step 0)
2. **Week 2**: Update code (Steps 1-5)
3. **Week 3**: Update domain catalogs (Step 6), test
4. **Week 4**: Deploy and monitor

**Estimated effort**: 2-3 days for 36 domain catalogs (~1 hour per catalog)

---

## Next Steps

Proceed to:
- [06_IMPLEMENTATION_STEPS.md](06_IMPLEMENTATION_STEPS.md) - Start implementation
- [07_TESTING_STRATEGY.md](07_TESTING_STRATEGY.md) - Validate migration
