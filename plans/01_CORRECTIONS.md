# Critical Corrections from Original Plan

## Overview

This document summarizes the key corrections made to the original `next_plan.md` based on code review and architecture discussions.

---

## 1. Global Domain Structure

### ❌ Original Plan
- 9 global entity types (person, organization, location, date, event, document, identifier, assertion, hypothesis)
- Flat structure, no subtypes

### ✅ Corrected
- **33 global entity types**: 10 top-level + 23 subtypes
- Organized by 5 W's categories (who, what, where, when, why)
- Hierarchical structure with parent-child relationships

**Rationale**: Existing `common.go` had 50+ templates. We distilled these to 33 high-value, reusable global types organized hierarchically.

---

## 2. Multiple Definitions Model

### ❌ Original Plan Assumption
- Each entity_type appears once in schema
- Single extraction rule per entity

### ✅ Corrected
- **Multiple definitions allowed**: Same `entity_type` can have multiple `ElementEntityMapping` entries
- Each definition has different `confidence` level (priority tier)
- Definitions can have different `parent_type` and `w_category` values
- Progressive refinement: high-confidence → medium → low fallback

**Rationale**: User confirmed this is intentional design for progressive refinement via confidence-based priority.

---

## 3. Hierarchy Computation Model

### ❌ Original Plan
- Static hierarchy defined in catalogs
- Parent must list all children explicitly
- Child must reference parent explicitly

### ✅ Corrected
- **Bidirectional auto-computation**: `parent_type` and `children` are both optional
- Runtime algorithm fills missing relationships
- `parent_type` is source of truth for upward links
- `children` array auto-filled by scanning all entities

**Rationale**: Resolves circular dependency problem - global domain doesn't need to know about domain catalogs at authoring time.

---

## 4. Validation Rules

### ❌ Original Plan Implied
- Multiple definitions of same entity must have consistent `parent_type`
- Multiple definitions must have consistent `w_category`
- Strict bidirectional consistency required

### ✅ Corrected
- **No consistency requirements** across multiple definitions
- **Only validate**: broken references (orphans) + circular hierarchies
- Same entity can have different `parent_type` in different contexts
- Same entity can have different `w_category` in different contexts

**Rationale**: Flexibility for entities serving different roles in different contexts.

---

## 5. Type Field Removal

### ❌ Original Plan Inconsistency
- Section 8-9 said "remove Type field"
- Section 11 still validated Type field
- Unclear if fully committed to removal

### ✅ Corrected
- **Fully committed**: Type field completely removed from `ExtractionRule`
- All code references updated
- All validation logic updated
- All LLM prompts updated

**Rationale**: Unified extraction structure is cleaner, more flexible, and reduces complexity.

---

## 6. File Organization

### ❌ Original Plan
- Keep `common.go` with modified content
- 50+ entity templates in one file

### ✅ Corrected
- **Delete `common.go`**
- **Create 5 new files**:
  - `global_who.go` (person, organization + 10 subtypes)
  - `global_what.go` (document, identifier, role + 6 subtypes)
  - `global_where.go` (location + 5 subtypes)
  - `global_when.go` (date, event + 2 subtypes)
  - `global_why.go` (assertion, hypothesis)
- Each file ~150-200 lines (manageable)

**Rationale**: Better organization, easier navigation, aligns with 5 W's framework.

---

## 7. Hierarchy Materialization Integration

### ❌ Original Plan
- Unclear where `MaterializeHierarchy()` gets called
- No integration with existing extraction flow

### ✅ Corrected
- **Called AFTER all entity extraction completes**
- Integrated into `ExtractFromElements()` pipeline:
  1. Extract leaf entities
  2. Materialize hierarchy (create composites + IS-A relationships)
  3. Extract domain relationships
  4. Combine all entities + relationships

**Rationale**: Clear execution order prevents confusion.

---

## 8. Migration Strategy

### ❌ Original Plan Unclear
- Mentioned "migration path for existing schemas"
- Backward compatibility considerations

### ✅ Corrected
- **No migration path provided**
- **Breaking changes accepted**
- Clean slate approach preferred
- Existing schemas must be manually updated

**Rationale**: Clean design over backward compatibility (per project guidelines).

---

## 9. Boolean Logic Model

### ❌ Original Plan Confusion
- Unclear relationship between multiple definitions vs. multiple rules
- Extraction rule structure ambiguous

### ✅ Corrected - Clear 3-Level Model

**Level 1: Multiple Entity Definitions (OR)**
- Multiple `ElementEntityMapping` entries with same `entity_type`
- Each has ONE `ExtractionRule` object
- Try all definitions, extract all matches
- Dedup at graph phase (highest confidence wins)

**Level 2: Filters Within ONE Rule (AND - fail fast)**
- Each `ExtractionRule` has multiple optional filter fields
- All present filters must pass (AND logic)
- Short-circuit on first failure (performance optimization)

**Level 3: OR Within Individual Filters**
- `phrase_list`: Match any phrase (OR)
- `proximity.keywords`: Match any keyword (OR)
- `semantic.reference_concepts`: Match any concept (OR)
- Regex alternation: Built-in OR (`Dr\.|Prof\.|Mr\.`)

**Rationale**: Clear mental model for extraction logic at all levels.

---

## 10. Role Entity Categorization

### ❌ Original Plan
- `role` categorized as WHO (person-related)

### ✅ Corrected
- **`role` categorized as WHAT** (w_category: what)
- Rationale: Role is a position/function (what), not a person (who)
- Examples: CEO, Professor, Manager

**Rationale**: Clearer semantics - roles are labels/classifications, not agents.

---

## 11. Organization Subtypes

### ❌ Original Plan
- `organization` was standalone (no children)

### ✅ Corrected
- **7 organization subtypes**:
  - business
  - nonprofit
  - government
  - educational
  - healthcare
  - religious
  - media

**Rationale**: Common, high-value subtypes worth including in global domain.

---

## Summary of Key Decisions

| Decision | Rationale |
|----------|-----------|
| 33 global types (not 9) | Leverage existing common.go templates, organized hierarchically |
| Multiple definitions allowed | Progressive refinement via confidence levels |
| Auto-compute hierarchies | Resolve circular dependency (global ↔ domain catalogs) |
| Minimal validation | Flexibility for multi-context entities |
| Remove Type field fully | Cleaner unified extraction structure |
| Split into 5 files | Better organization by W category |
| No migration path | Clean design over backward compatibility |
| 3-level boolean logic | Clear mental model for extraction |

---

## Impact Assessment

### Breaking Changes
- ✅ **Accepted**: Type field removal
- ✅ **Accepted**: common.go deletion
- ✅ **Accepted**: Schema structure changes
- ✅ **Accepted**: Domain catalog format changes

### Migration Required
- All 36 domain catalogs must be updated
- Existing schemas will fail validation
- No automated migration tool provided

### Benefits
- Cleaner architecture
- More flexible entity modeling
- Better performance (fail-fast filters)
- Easier maintenance (5 smaller files vs 1 large file)
- Reusable global entity templates

---

## Next Steps

Proceed to:
- [02_GLOBAL_DOMAIN.md](02_GLOBAL_DOMAIN.md) - See the 33 entity types
- [03_UNIFIED_EXTRACTION.md](03_UNIFIED_EXTRACTION.md) - Understand new rule structure
- [04_HIERARCHY_SYSTEM.md](04_HIERARCHY_SYSTEM.md) - Learn auto-computation algorithm