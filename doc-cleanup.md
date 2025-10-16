/sm# Documentation Cleanup & Fix Plan

**Version**: 1.0 Documentation Standards
**Status**: 🔴 Not Started
**Created**: 2025-01-16
**Target Completion**: 4 weeks (or 2 days minimum viable)
**Progress**: 0% Complete

---

## Executive Summary

**Note**: This is version 1.0 of Go-Doc-Go. All documentation should reflect the current implementation with no references to deprecated features, legacy approaches, or previous versions.

### Problems Identified
- ❌ **30% of documentation is completely wrong** (documents non-existent Python implementation)
- ❌ **70% uses wrong config format** (YAML instead of TOML)
- ❌ **4 overlapping getting started guides** create confusion
- ❌ **Missing critical docs** (troubleshooting, CLI reference, sources)
- ❌ **Poor organization** (20+ files in root, unclear entry point)

### Solution Overview
4-phase plan to fix all documentation issues:
1. **Week 1**: Delete wrong docs, convert formats, create quick reference
2. **Week 2**: Reorganize structure, create missing docs, improve navigation
3. **Week 3**: Consistency pass, validate all examples, setup CI/CD
4. **Week 4+**: Advanced features (videos, diagrams, examples)

---

## Quick Stats

| Metric | Current | Target |
|--------|---------|--------|
| **Incorrect docs** | 3 files (cli.md, installation.md, config.md) | 0 files |
| **YAML examples** | ~80 blocks | 0 blocks (all TOML) |
| **Missing docs** | 5 critical gaps | 0 gaps |
| **First-time success rate** | ~30% (estimated) | >90% |
| **Time to first success** | 30+ min (estimated) | <10 min |

---

## PHASE 1: EMERGENCY CLEANUP 🚨
**Timeline**: Week 1 (5 days)
**Status**: ⬜ Not Started
**Priority**: CRITICAL

### Day 1: Delete Misleading Documentation
**Status**: ✅ Completed (2025-01-16)

#### ✅ Task 1.1: Delete Python-focused docs
- [x] Delete `docs/cli.md` (391 lines - documents non-existent Python CLI)
- [x] Delete `docs/installation.md` (243 lines - pip install doesn't exist)
- [x] Delete `docs/optional-dependencies.md` (313 lines - all Python pip packages)

**Commands**:
```bash
git rm docs/cli.md docs/installation.md
git commit -m "docs: remove Python CLI/installation docs (implementation doesn't exist)"
```

---

### Day 2-3: Fix Critical Format Issues
**Status**: ⬜ Not Started

#### ✅ Task 2.1: docs/configuration.md - Convert all YAML → TOML
**Total**: 57 YAML blocks to convert

**Checklist**:
- [ ] Lines 25-35: SQLite config
- [ ] Lines 38-56: PostgreSQL config
- [ ] Lines 59-77: MongoDB config
- [ ] Lines 80-95: Elasticsearch config
- [ ] Lines 103-123: FastEmbed config
- [ ] Lines 126-137: HuggingFace config
- [ ] Lines 140-152: OpenAI config
- [ ] Lines 160-183: File source config
- [ ] Lines 186-214: Database source config
- [ ] Lines 217-248: Web source config
- [ ] Lines 251-267: S3 source config
- [ ] Lines 270-286: SharePoint config
- [ ] Lines 289-306: Confluence config
- [ ] Lines 312-337: Relationship detection
- [ ] Lines 343-373: Processing config
- [ ] Lines 380-399: Analytics config
- [ ] Lines 405-426: Logging config
- [ ] Lines 460-475: Development profile
- [ ] Lines 479-498: Production profile
- [ ] Lines 516-526: Minimal config
- [ ] Lines 530-550: High-performance config
- [ ] Lines 555-582: Enterprise config

**Conversion example**:
```yaml
# BEFORE (WRONG):
storage:
  backend: "postgresql"
  host: "localhost"

# AFTER (CORRECT):
[storage]
backend = "postgresql"
host = "localhost"
```

#### ✅ Task 2.2: docs/ontology.md - Convert config examples
- [ ] Lines 80-107: Ontology config
- [ ] Lines 219-240: Entity extraction config
- [ ] Lines 356-376: Confidence tuning config

#### ✅ Task 2.3: docs/scaling.md - Convert config examples
- [ ] Lines 50-65: Minimal config
- [ ] Lines 76-98: Cluster config
- [ ] Lines 182-221: Worker optimization config
- [ ] Lines 262-278: Monitoring config

#### ✅ Task 2.4: docs/embeddings.md - Convert config examples
- [ ] Lines 67-84: Basic embedding config
- [ ] Lines 89-111: Advanced contextual config
- [ ] Lines 119-135: FastEmbed config
- [ ] Lines 141-154: HuggingFace config
- [ ] Lines 160-177: OpenAI config

**Verification**:
```bash
# Check no YAML remains (excluding URLs)
grep -n "^[a-z_]*:" docs/*.md | grep -v "http:" | grep -v "https:"
```

---

### Day 4: Update CLAUDE.md
**Status**: ⬜ Not Started

#### ✅ Task 3.1: Remove Python sections
- [ ] Delete lines 247-293: Pre-commit checklist (Python tools)
- [ ] Delete lines 295-300: "Before Committing (Legacy)"
- [ ] Delete lines 302-306: Python coverage goals

#### ✅ Task 3.2: Add Go development workflow
- [ ] Add Go pre-commit checklist section
- [ ] Add Go-specific testing requirements
- [ ] Add Go formatting standards (gofmt)

**New section to add**:
```markdown
### Pre-Commit Verification Checklist - MANDATORY

```bash
#!/bin/bash
cd go
go build ./... || { echo "✗ Build failed"; exit 1; }
go test ./... || { echo "✗ Tests failed"; exit 1; }
gofmt -l . | grep . && { echo "✗ Needs formatting"; exit 1; }
go vet ./... || { echo "✗ Go vet failed"; exit 1; }
```
```

#### ✅ Task 3.3: Update performance benchmarks
- [ ] Replace Python SLAs with Go implementation benchmarks
- [ ] Update from go/cmd/worker/main.go actual performance

---

### Day 5: Create Quick Reference
**Status**: ⬜ Not Started

#### ✅ Task 4.1: Create QUICK_REFERENCE.md
- [ ] Installation instructions (Go only)
- [ ] All CLI flags (from main.go:77-85)
- [ ] Environment variables
- [ ] Basic config examples (TOML)
- [ ] Common commands

**File to create**: `/QUICK_REFERENCE.md`

**Content outline**:
```markdown
# Go-Doc-Go Quick Reference

## Installation
## CLI Flags (ALL of them)
## Environment Variables
## Configuration (TOML examples only)
## Common Commands
## Troubleshooting Quick Tips
```

---

## PHASE 2: CONTENT REORGANIZATION 📁
**Timeline**: Week 2 (5 days)
**Status**: ⬜ Not Started
**Priority**: HIGH

### Day 6-7: Restructure docs/ directory
**Status**: ⬜ Not Started

#### ✅ Task 5.1: Create new directory structure
```bash
mkdir -p docs/getting-started
mkdir -p docs/configuration
mkdir -p docs/features/embeddings
mkdir -p docs/features/ontology
mkdir -p docs/features/udml
mkdir -p docs/operations
mkdir -p docs/reference
mkdir -p docs/architecture
```

- [ ] Create all new directories

#### ✅ Task 5.2: Move files to new locations

**Getting Started**:
- [ ] `GETTING_STARTED.md` → `docs/getting-started/README.md`

**Configuration**:
- [ ] `docs/configuration.md` → `docs/configuration/README.md`
- [ ] Create `docs/configuration/sources.md`
- [ ] Create `docs/configuration/storage.md`

**Features - Embeddings**:
- [ ] `docs/embeddings.md` → `docs/features/embeddings/README.md`

**Features - Ontology**:
- [ ] `docs/ontology.md` → `docs/features/ontology/README.md`
- [ ] `docs/ontology-examples.md` → `docs/features/ontology/examples.md`
- [ ] `docs/ONTOLOGY_QUICK_START.md` → `docs/features/ontology/quick-start.md`
- [ ] `docs/ONTOLOGY_WORKFLOWS.md` → `docs/features/ontology/workflows.md`
- [ ] `docs/domain-quickstart.md` → `docs/features/ontology/domain-quickstart.md`

**Features - UDML**:
- [ ] `docs/UDML_SPECIFICATION.md` → `docs/features/udml/specification.md`
- [ ] `docs/UDML_SCHEMAS.md` → `docs/features/udml/schemas.md`
- [ ] `docs/UDML_ONTOLOGY_SYSTEM.md` → `docs/features/udml/ontology-system.md`

**Operations**:
- [ ] `docs/scaling.md` → `docs/operations/scaling.md`

**Architecture**:
- [ ] `GOROUTINE_WORKER_DESIGN.md` → `docs/architecture/worker-design.md`
- [ ] `UDML_ONTOLOGY_COMPLETE.md` → `docs/architecture/udml-ontology-complete.md`

#### ✅ Task 5.3: Update all internal links
- [ ] Create and run `scripts/update_links.sh`
- [ ] Verify all links work
- [ ] Test with markdown link checker

---

### Day 8: Create Missing Documentation
**Status**: ⬜ Not Started

#### ✅ Task 6.1: Create docs/operations/troubleshooting.md
**Content sections**:
- [ ] Worker not starting
- [ ] Configuration errors
- [ ] ONNX Runtime issues
- [ ] PostgreSQL connection issues
- [ ] Memory issues
- [ ] Performance issues

#### ✅ Task 6.2: Create docs/operations/monitoring.md
**Content sections**:
- [ ] Built-in metrics (log-based)
- [ ] Health checks (placeholder)
- [ ] Prometheus integration (placeholder)

#### ✅ Task 6.3: Create docs/reference/cli.md (CORRECT VERSION)
**Content sections**:
- [ ] All flags from main.go:77-85
- [ ] Environment variables
- [ ] Examples of each flag usage
- [ ] Exit codes

#### ✅ Task 6.4: Create docs/configuration/sources.md
**Content sections**:
- [ ] File system sources
- [ ] S3/MinIO sources
- [ ] Web/HTTP sources
- [ ] All source types with TOML examples

#### ✅ Task 6.5: Create docs/configuration/storage.md
**Content sections**:
- [ ] SQLite (development)
- [ ] PostgreSQL (production)
- [ ] Job control configuration
- [ ] Analytics outputs (Parquet, Neo4j)

---

### Day 9: Update Root README.md
**Status**: ⬜ Not Started

#### ✅ Task 7.1: Simplify README.md
- [ ] Reduce from 386 lines to ~200 lines
- [ ] Perfect "Quick Start" section
- [ ] Clear documentation navigation
- [ ] Link to QUICK_REFERENCE.md

**Target structure**:
```markdown
# Title & Description (50 lines)
## Quick Start (60 seconds) (40 lines)
## What Makes It Unique (40 lines)
## Documentation (30 lines - links only)
## Performance (20 lines)
## Deployment (20 lines)
```

---

### Day 10: Create Navigation
**Status**: ⬜ Not Started

#### ✅ Task 8.1: Create docs/README.md (Documentation Hub)
- [ ] Welcome section
- [ ] Getting started path
- [ ] Documentation by topic
- [ ] Quick links
- [ ] Search/index

#### ✅ Task 8.2: Add navigation footers
- [ ] Create footer template
- [ ] Add to all major docs
- [ ] Include "Previous/Next/Up" links
- [ ] Include quick links section

**Footer template**:
```markdown
---

## Related Documentation
- **Previous**: [Link]
- **Next**: [Link]
- **Up**: [Parent]

## Quick Links
- [Documentation Home](../)
- [Quick Reference](../../QUICK_REFERENCE.md)
- [Troubleshooting](../../operations/troubleshooting.md)
```

---

## PHASE 3: POLISH & VALIDATION ✨
**Timeline**: Week 3 (4 days)
**Status**: ⬜ Not Started
**Priority**: MEDIUM

### Day 11-12: Consistency Pass
**Status**: ⬜ Not Started

#### ✅ Task 9.1: Fix terminology inconsistencies
- [ ] Create `docs/GLOSSARY.md`
- [ ] "Worker Node" → "Worker" (everywhere)
- [ ] "Work Queue" → "Job Control" (everywhere)
- [ ] "Doc/File" → "Document" (consistent usage)
- [ ] Run find/replace scripts

**Glossary terms to define**:
- Worker
- Goroutine Worker
- Job Control
- Content Source
- Analytics Output
- TOML (only format)
- UDML
- Element
- Relationship

#### ✅ Task 9.2: Fix heading hierarchy
- [ ] Check no h1 → h3 skips
- [ ] Ensure consistent h1 usage (only one per doc)
- [ ] Fix any hierarchy violations

#### ✅ Task 9.3: Standardize code blocks
- [ ] Add language tags to all code blocks
- [ ] Ensure bash/toml/go tags are correct
- [ ] No bare ``` blocks remain

---

### Day 13: Integration Testing
**Status**: ⬜ Not Started

#### ✅ Task 10.1: Create test suite for examples
- [ ] Create `scripts/test_examples.sh`
- [ ] Test minimal config from README
- [ ] Test Neo4j export config
- [ ] Test embeddings config
- [ ] Test distributed worker config

#### ✅ Task 10.2: Validate all TOML examples
- [ ] Create `scripts/validate_toml.sh`
- [ ] Extract TOML from markdown
- [ ] Validate each with tomlv
- [ ] Report any invalid examples

#### ✅ Task 10.3: Create docs CI/CD
- [ ] Create `.github/workflows/docs-validation.yml`
- [ ] Build worker in CI
- [ ] Run test_examples.sh
- [ ] Run validate_toml.sh
- [ ] Check for broken links

---

### Day 14: Final Review & Cleanup
**Status**: ⬜ Not Started

#### ✅ Task 11.1: Complete quality checklist
- [ ] All code examples tested ✓
- [ ] All configuration examples validated ✓
- [ ] All CLI flags documented ✓
- [ ] No Python references remain ✓
- [ ] Getting started guide complete ✓
- [ ] All features documented ✓
- [ ] Troubleshooting comprehensive ✓
- [ ] Directory structure clear ✓
- [ ] Navigation works ✓
- [ ] Terminology uniform ✓
- [ ] Code blocks tagged ✓
- [ ] Heading hierarchy correct ✓
- [ ] Links not broken ✓
- [ ] All configs TOML ✓

#### ✅ Task 11.2: Generate table of contents
- [ ] Install markdown-toc
- [ ] Generate TOCs for all major docs
- [ ] Verify TOC links work

#### ✅ Task 11.3: Spell check
- [ ] Install aspell
- [ ] Run on all markdown files
- [ ] Fix spelling errors

---

## PHASE 4: LONG-TERM IMPROVEMENTS 🚀
**Timeline**: Week 4+ (ongoing)
**Status**: ⬜ Not Started
**Priority**: LOW

### Future Enhancements

#### ✅ Task 12.1: Interactive examples
- [ ] Create interactive config wizard
- [ ] Add to CLI: `goworker config-wizard`

#### ✅ Task 12.2: Video tutorials
- [ ] "First 5 minutes" video
- [ ] "Scaling to production" video
- [ ] Embed in documentation

#### ✅ Task 12.3: Add diagrams
- [ ] Install mermaid-cli
- [ ] Create architecture diagrams
- [ ] Create workflow diagrams
- [ ] Render as SVG in docs

#### ✅ Task 12.4: Create example projects
- [ ] `examples/simple-pdf-processing/`
- [ ] `examples/distributed-workers/`
- [ ] `examples/neo4j-knowledge-graph/`
- [ ] `examples/semantic-search/`

Each with:
- README.md
- config.toml
- Sample data
- Expected output

---

## EMERGENCY SHORTCUTS ⚡

### Absolute Minimum (2 days)
If time critical, do ONLY this:

**Day 1**:
- [ ] DELETE `docs/cli.md` and `docs/installation.md`
- [ ] ADD warning to top of `docs/configuration.md`:
  ```markdown
  > ⚠️ Go-Doc-Go uses TOML config (not YAML). Convert examples accordingly.
  ```
- [ ] CREATE `QUICK_REFERENCE.md` with correct info

**Day 2**:
- [ ] UPDATE `README.md` with link to QUICK_REFERENCE.md
- [ ] CREATE `docs/reference/cli.md` with actual flags from main.go

**Result**: Users can at least find correct information, even if not perfect.

---

## SCRIPTS TO CREATE 🛠️

### scripts/yaml_to_toml.py
```python
#!/usr/bin/env python3
"""Convert YAML config examples to TOML in markdown files."""
# TODO: Implement
```
- [ ] Create script

### scripts/update_links.sh
```bash
#!/bin/bash
"""Update all internal links after reorganization."""
# TODO: Implement
```
- [ ] Create script

### scripts/test_examples.sh
```bash
#!/bin/bash
"""Test all configuration examples actually work."""
# TODO: Implement
```
- [ ] Create script

### scripts/validate_toml.sh
```bash
#!/bin/bash
"""Extract and validate all TOML blocks from docs."""
# TODO: Implement
```
- [ ] Create script

### scripts/check_terminology.sh
```bash
#!/bin/bash
"""Check for terminology inconsistencies."""
# TODO: Implement
```
- [ ] Create script

### scripts/validate_all.sh
```bash
#!/bin/bash
"""Master validation - run all checks."""
# TODO: Implement
```
- [ ] Create script

---

## COMMIT STRATEGY 💾

### Commit Message Format

Use these prefixes:

```bash
# Emergency cleanup
git commit -m "docs: delete Python CLI documentation (doesn't exist)"
git commit -m "docs: add TOML-only warning to config docs"

# Format fixes
git commit -m "docs(config): convert all YAML examples to TOML"
git commit -m "docs(ontology): convert config examples to TOML"

# Reorganization
git commit -m "docs: restructure into topic-based directories"
git commit -m "docs: create missing troubleshooting guide"

# Polish
git commit -m "docs: fix terminology consistency"
git commit -m "docs: add navigation between guides"
```

### Pull Request Strategy

Create separate PRs:

1. **PR #1: Emergency Cleanup**
   - Delete misleading docs
   - Add warnings
   - Create quick reference

2. **PR #2: Format Fixes**
   - YAML → TOML conversion
   - All config files

3. **PR #3: Reorganization**
   - New directory structure
   - Move files
   - Update links

4. **PR #4: New Content**
   - Missing docs
   - Troubleshooting
   - CLI reference

5. **PR #5: Polish**
   - Consistency
   - Navigation
   - Final review

---

## SUCCESS METRICS 📊

### Current State (Baseline)
- First-time success rate: ~30% (estimated)
- Time to first success: 30+ minutes
- Config examples that work: ~20%
- Docs-related GitHub issues: Unknown
- User confusion reports: High

### Target State (After Completion)
- [ ] First-time success rate: >90%
- [ ] Time to first success: <10 minutes
- [ ] Config examples that work: 100%
- [ ] Docs-related issues: <5% of total
- [ ] User confusion: Minimal

### Tracking
Create issues in GitHub to track:
- "User couldn't get worker running" (should decrease)
- "Config examples don't work" (should be zero)
- "Documentation unclear" (should decrease)

---

## PROGRESS DASHBOARD

### Overall Progress: 0% Complete

```
Phase 1: Emergency Cleanup     [░░░░░░░░░░] 0%  (0/15 tasks)
Phase 2: Reorganization        [░░░░░░░░░░] 0%  (0/25 tasks)
Phase 3: Polish & Validation   [░░░░░░░░░░] 0%  (0/12 tasks)
Phase 4: Long-term             [░░░░░░░░░░] 0%  (0/4 tasks)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Total Progress:                [░░░░░░░░░░] 0%  (0/56 tasks)
```

### Key Milestones
- [ ] Emergency fixes complete (Phase 1)
- [ ] Reorganization complete (Phase 2)
- [ ] All examples validated (Phase 3)
- [ ] User success >90% (Metrics)

---

## NOTES & DECISIONS

### Decision Log

**2025-01-16**: Created comprehensive fix plan for version 1.0
- Version 1.0 philosophy: No deprecated features or legacy references
- Delete incorrect Python documentation (not deprecate)
- TOML as only config format (YAML was never supported)
- Set 4-week timeline with 2-day emergency option

**2025-01-16**: Completed Task 1.1
- Deleted 3 Python documentation files (949 lines)
- No deprecation notices needed (version 1.0)

### Open Questions
- [ ] Keep DEMO_USAGE.md or move to examples/? (TBD)
- [ ] Rename repository from "doculyzer-go-conversion" to "go-doc-go"? (TBD)

### Risks & Mitigations
1. **Risk**: Links break during reorganization
   - **Mitigation**: Automated link update script, CI/CD validation

2. **Risk**: Examples don't actually work
   - **Mitigation**: Integration test suite in CI/CD

3. **Risk**: Users reference old documentation from git history
   - **Mitigation**: Version 1.0 has no legacy - old docs were simply wrong, not deprecated

---

## SIGN-OFF

### Phase Completion Sign-Off

**Phase 1: Emergency Cleanup**
- [ ] Completed by: _____________ Date: _______
- [ ] Reviewed by: _____________ Date: _______

**Phase 2: Reorganization**
- [ ] Completed by: _____________ Date: _______
- [ ] Reviewed by: _____________ Date: _______

**Phase 3: Polish & Validation**
- [ ] Completed by: _____________ Date: _______
- [ ] Reviewed by: _____________ Date: _______

**Phase 4: Long-term**
- [ ] Completed by: _____________ Date: _______
- [ ] Reviewed by: _____________ Date: _______

### Final Approval
- [ ] All tasks completed
- [ ] All metrics met
- [ ] CI/CD passing
- [ ] User feedback positive

**Approved by**: _____________ **Date**: _______

---

## APPENDIX

### Files Affected Summary

**To Delete** (3):
- docs/cli.md
- docs/installation.md
- docs/optional-dependencies.md (verify first)

**To Convert YAML→TOML** (5 major files, ~80 code blocks):
- docs/configuration.md (57 blocks)
- docs/ontology.md (8 blocks)
- docs/scaling.md (6 blocks)
- docs/embeddings.md (5 blocks)
- docs/QUICK_START.md (4 blocks)

**To Move** (15):
- GETTING_STARTED.md → docs/getting-started/README.md
- docs/ontology.md → docs/features/ontology/README.md
- [... see Phase 2 for complete list]

**To Create** (8):
- QUICK_REFERENCE.md
- docs/README.md
- docs/operations/troubleshooting.md
- docs/operations/monitoring.md
- docs/reference/cli.md
- docs/configuration/sources.md
- docs/configuration/storage.md
- docs/GLOSSARY.md

**To Update** (3):
- README.md (simplify to ~200 lines)
- CLAUDE.md (remove Python, add Go)
- go/README.md (add missing flags)

---

**Last Updated**: 2025-01-16
**Next Review**: After Phase 1 completion
**Maintained By**: Documentation Team