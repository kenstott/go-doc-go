# Ontology Interview Refactoring - Plan Index

## Overview

This directory contains the corrected implementation plan for refactoring the ontology interview system, organized into focused sub-documents for better comprehension.

---

## Plan Structure

### Core Architecture Documents

1. **[01_CORRECTIONS.md](01_CORRECTIONS.md)** - Critical corrections from original plan
   - What changed and why
   - Key architectural decisions
   - Validation model

2. **[02_GLOBAL_DOMAIN.md](02_GLOBAL_DOMAIN.md)** - Global domain structure (33 entity types)
   - 10 top-level types organized by 5 W's
   - 23 subtypes with hierarchies
   - File organization (5 files)

3. **[03_UNIFIED_EXTRACTION.md](03_UNIFIED_EXTRACTION.md)** - Unified extraction rule structure
   - Removal of Type discriminator
   - Performance-optimized pipeline
   - Migration examples

4. **[04_HIERARCHY_SYSTEM.md](04_HIERARCHY_SYSTEM.md)** - Hierarchy computation & materialization
   - Bidirectional auto-computation
   - Runtime relationship discovery
   - Composite entity generation

5. **[05_VALIDATION.md](05_VALIDATION.md)** - Validation rules & algorithms
   - Broken reference detection
   - Circular hierarchy detection
   - What's allowed vs. errors

---

### Implementation Documents

6. **[06_IMPLEMENTATION_STEPS.md](06_IMPLEMENTATION_STEPS.md)** - Step-by-step implementation checklist
   - Steps 0-9 with file locations
   - Estimated line changes
   - Code snippets

7. **[07_TESTING_STRATEGY.md](07_TESTING_STRATEGY.md)** - Testing approach
   - Unit tests
   - Integration tests
   - Validation scripts

8. **[08_MIGRATION_GUIDE.md](08_MIGRATION_GUIDE.md)** - Migration notes
   - Breaking changes
   - Domain catalog updates (entity extraction + relationship rules)
   - No backward compatibility

9. **[09_RELATIONSHIP_RULES.md](09_RELATIONSHIP_RULES.md)** - Relationship extraction rules
   - EntityRelationshipRule schema
   - 5 extraction pattern types (proximity, regex, template, dependency, cooccurrence)
   - Synthesized entity filtering
   - Domain catalog examples

---

## Quick Reference

### Total Scope
- **Files changed**: ~56 (5 new global catalogs, 36 domain catalogs updated, 9 code files, 1 new plan doc)
- **Net line change**: ~1,400 lines (~1,083 code + ~300 relationship rules)
- **Implementation steps**: 11 (Steps 0-9 + Step 6.2 for relationship rules)
- **Estimated time**: 3-4 days

### Key Numbers
- **33 global entity types** (10 top-level + 23 subtypes)
- **5 W's categories** (who, what, where, when, why)
- **36 domain catalogs** to update (entity extraction rules)
- **~20 domain catalogs** with relationship rules to migrate
- **5 relationship pattern types** (proximity, regex, template, dependency, cooccurrence)
- **400+ unit test lines** to add

---

## Reading Order

### For Architects/Reviewers
1. Start with [01_CORRECTIONS.md](01_CORRECTIONS.md) - understand what changed
2. Read [02_GLOBAL_DOMAIN.md](02_GLOBAL_DOMAIN.md) - see the entity structure
3. Review [04_HIERARCHY_SYSTEM.md](04_HIERARCHY_SYSTEM.md) - understand key innovation
4. Check [05_VALIDATION.md](05_VALIDATION.md) - verify error handling

### For Implementers
1. Read [06_IMPLEMENTATION_STEPS.md](06_IMPLEMENTATION_STEPS.md) - get the checklist
2. Reference [03_UNIFIED_EXTRACTION.md](03_UNIFIED_EXTRACTION.md) - understand entity extraction code
3. Reference [09_RELATIONSHIP_RULES.md](09_RELATIONSHIP_RULES.md) - understand relationship extraction
4. Use [07_TESTING_STRATEGY.md](07_TESTING_STRATEGY.md) - validate each step
5. Consult [08_MIGRATION_GUIDE.md](08_MIGRATION_GUIDE.md) - update catalogs

---

## Status

- [ ] Plan review complete
- [ ] Architecture approved
- [ ] Implementation started
- [ ] Tests passing
- [ ] Domain catalogs updated
- [ ] Documentation complete

---

## Questions/Issues

Track open questions and issues here:
- [ ] Confirm global domain entity list is complete
- [ ] Verify domain catalog update approach
- [ ] Validate testing coverage is sufficient