# Go-Doc-Go Documentation Implementation Plan

## Executive Summary

**Goal**: Create comprehensive, balanced documentation covering all user types (end users, developers, contributors, operators).

**Status**: Phase 1 Complete (11 of 11 tasks ✓)

**Current Coverage**:
- Godoc: 8 critical packages documented (up from 0.5%)
- Troubleshooting: 16 common issues documented
- Standards: Documentation standards established
- Templates: 3 reusable templates created
- Infrastructure: Directory structure established

**Target Coverage**:
- Godoc: 100% of packages (214 files)
- Troubleshooting: Top 30 issues documented
- Features: All 28+ parsers documented
- API: All 4 major interfaces documented
- Operations: Complete deployment guides

---

## Phase 1: Foundation & Quick Wins ✅ COMPLETED

**Duration**: 2 weeks (Completed)

### Deliverables ✓

1. **Documentation Infrastructure**
   - ✓ Created directory structure (`docs/{troubleshooting,processes,reference,operations,development,features,_templates}`)
   - ✓ Added documentation templates (3 templates)
   - ✓ Created `docs/DOCUMENTATION_STANDARDS.md`

2. **Critical Package Documentation (8 doc.go files)**
   - ✓ `go/internal/parser/doc.go` - Parser system overview
   - ✓ `go/internal/analytics/doc.go` - Storage backends
   - ✓ `go/internal/worker/doc.go` - Worker coordination
   - ✓ `go/internal/contentsource/doc.go` - Content ingestion
   - ✓ `go/internal/jobcontrol/doc.go` - Job queue system
   - ✓ `go/internal/embeddings/doc.go` - Embeddings generation
   - ✓ `go/internal/udml/doc.go` - UDML system
   - ✓ `go/internal/export/doc.go` - Graph export

3. **Troubleshooting Documentation**
   - ✓ `docs/troubleshooting/common-issues.md` - 16 common issues with solutions

### Metrics ✓

- 11/11 tasks completed
- 8 critical packages documented
- 16 troubleshooting entries
- 3 documentation templates
- 1 standards document

---

## Phase 2: Feature Documentation

**Duration**: 2 weeks

**Goal**: Document all parsers and discovery features to help users understand capabilities.

### Tasks

#### 2.1 Parser Overview Documentation

**File**: `docs/features/parsing/overview.md`

**Content**:
- Catalog of all 28+ parsers
- Comparison table (supported elements, performance, limitations)
- Quick start for each parser type
- When to use which parser
- Performance characteristics comparison

**Steps**:
1. List all parsers from `go/internal/parser/`
2. Create comparison table (columns: parser, extensions, elements, speed, memory, limitations)
3. Add quick start examples for common use cases
4. Document element types produced by each parser
5. Add performance benchmarks

**Acceptance Criteria**:
- [ ] All parsers listed with examples
- [ ] Comparison table with 6+ attributes
- [ ] At least 3 quick start examples
- [ ] Links to individual parser docs

---

#### 2.2 Individual Parser Documentation

**Files** (6 files):
- `docs/features/parsing/pdf-parsing.md`
- `docs/features/parsing/office-documents.md` (DOCX, XLSX, PPTX)
- `docs/features/parsing/code-parsing.md` (update existing)
- `docs/features/parsing/data-formats.md` (JSON, CSV, Parquet, XML, YAML)
- `docs/features/parsing/web-content.md` (HTML, Markdown)
- `docs/features/parsing/text-parsing.md` (Plain text)

**Template for Each Parser Doc**:
```markdown
# [Format] Parsing

## Capabilities
- What elements are extracted
- Structure preservation
- Metadata extraction

## Configuration
[TOML config examples with annotations]

## Examples
### Basic Usage
[Example with input and output]

### Advanced Usage
[Complex scenarios]

## Output Elements
[Element types produced]

## Performance
- Speed benchmarks
- Memory usage
- Optimization tips

## Limitations
- Known issues
- Unsupported features
- Workarounds

## Troubleshooting
[Common issues specific to this parser]
```

**Steps for Each Parser**:
1. Review parser source code (`go/internal/parser/[parser].go`)
2. Identify element types produced
3. Create example input/output
4. Document configuration options
5. Add performance notes from testing
6. Document limitations from code comments
7. Test all examples

**Acceptance Criteria (per parser doc)**:
- [ ] Lists all element types produced
- [ ] 2+ working examples (basic + advanced)
- [ ] Configuration reference with defaults
- [ ] Performance characteristics documented
- [ ] Known limitations listed
- [ ] Links to troubleshooting

---

#### 2.3 Discovery System Documentation

**Files** (3 files):
- `docs/features/discovery/overview.md`
- `docs/features/discovery/hyperlink-crawling.md`
- `docs/features/discovery/code-dependencies.md`

**Content**:

**overview.md**:
- Discovery system architecture
- How it works (document → parse → extract links → queue → repeat)
- Configuration overview
- Use cases (web crawling, codebase analysis)

**hyperlink-crawling.md**:
- URL resolution (relative → absolute)
- Pattern filtering (include/exclude)
- Depth control
- External domain handling
- Rate limiting
- Examples: crawling documentation sites

**code-dependencies.md**:
- Import resolution by language (Go, Python, JavaScript, Java)
- Package type filtering (stdlib, local, external)
- Depth control
- Module/project detection
- Examples: analyzing code dependencies

**Steps**:
1. Review discovery implementation (`go/internal/worker/worker.go`, content source code)
2. Document configuration schema
3. Create examples for each use case
4. Diagram the discovery flow (Mermaid)
5. Test all examples
6. Document language-specific resolution behavior

**Acceptance Criteria**:
- [ ] Discovery flow diagram
- [ ] Configuration examples for web and code
- [ ] 2+ working examples per discovery type
- [ ] Language-specific resolution documented
- [ ] Pattern filtering explained

---

### Phase 2 Summary

**Deliverables**: 10 feature documentation files

**Time Estimate**: 2 weeks
- Week 1: Parser overview + 3 individual parser docs
- Week 2: Remaining 3 parser docs + 3 discovery docs

**Success Metrics**:
- All 28+ parsers documented
- Discovery system fully documented
- 15+ working examples
- Users can find parser capabilities quickly

---

## Phase 2.5: Advanced Ontology Features

**Duration**: 1.5 weeks

**Goal**: Document the complete ontology system including advanced features (named rules, attribute extraction, validation, catalogs).

### Tasks

#### 2.5.1 Core Ontology System Documentation

**Files** (8 files):

1. **`docs/features/ontology/rule-system.md`**
   - Named extraction rules (name and description fields)
   - Rule inheritance via parent entity types
   - Explicit rule referencing by name (child references parent rules)
   - Rule organization best practices
   - Examples showing inherited and custom rules
   - CatalogSource tracking (global, domain, llm_generated)

2. **`docs/features/ontology/catalog-system.md`**
   - Built-in catalog structure (global + domain catalogs)
   - 6 W categories (who, what, where, when, why, how)
   - Domain-specific catalogs (medical, financial, legal, etc.)
   - Creating custom catalogs for your domain
   - Catalog loading and registration
   - Testing catalogs with example data

3. **`docs/features/ontology/attribute-extraction.md`**
   - Schema parsimony concept (fewer entity types, more attributes)
   - Attribute extraction types: constant, regex
   - Extraction scopes: entity_match, element, proximity
   - Real-world example: physician.specialty vs 50+ physician types
   - Relationship attributes (e.g., employment_status, start_date, role)
   - Best practices for parsimonious schema design

4. **`docs/features/ontology/filters-and-validation.md`**
   - Multi-stage filter pipeline (pattern → proximity → dictionary → semantic → LLM)
   - ProximityFilter: co-occurrence with signal terms
   - DictionaryFilter: linguistic validation (POS tags, proper nouns)
   - SemanticFilter: embedding similarity thresholds
   - LLMValidation: complex validation via LLM prompts
   - Filter combination strategies and performance tuning

5. **`docs/features/ontology/schema-validation.md`**
   - ValidationWarning types and severity levels (CRITICAL, HIGH, MEDIUM, LOW)
   - SchemaQualityReport structure and metrics
   - Duplicate pattern detection across entities
   - Confidence mismatch detection
   - Running validation before extraction
   - Interpreting and fixing validation errors
   - CI/CD integration for automated validation

6. **`docs/features/ontology/interview-cli.md`**
   - Interactive ontology creation workflow
   - LLM-guided schema generation process
   - Domain-specific question flow
   - Review and refinement iteration
   - Exporting interview results to YAML
   - Customizing interview questions for domains

7. **`docs/features/ontology/cross-domain-entities.md`**
   - Cross-domain entity merging configuration
   - Evidence types (fuzzy_match, semantic_similarity)
   - Similarity thresholds and min_domains settings
   - Performance implications of merging
   - When to enable/disable cross-domain merging
   - Troubleshooting entity duplicates

8. **`docs/features/ontology/llm-integration.md`**
   - LLM configuration (llm_model, llm_validation_model)
   - Model selection criteria (Sonnet, Opus, Haiku trade-offs)
   - LLM validation vs canonicalization
   - Batch sizing for cost management
   - Error handling and retry logic
   - API quota management and rate limiting

**Template for Ontology Feature Docs**:
```markdown
# [Feature Name]

## Overview
[What this feature does and why it's useful]

## Configuration
[YAML configuration examples with annotations]

## How It Works
[Detailed explanation of mechanics]

## Examples

### Basic Usage
[Simple example with input and output]

### Advanced Usage
[Complex scenario with multiple features]

## Best Practices
- [Practice 1]
- [Practice 2]
- [Practice 3]

## Common Pitfalls
- [Pitfall 1 and how to avoid]
- [Pitfall 2 and how to avoid]

## Performance Considerations
[Performance impact and tuning tips]

## Related Documentation
- [Link to related doc 1]
- [Link to related doc 2]
```

**Steps for Each Feature Doc**:
1. Review implementation in `go/internal/udml/ontology/`
2. Extract configuration schema from types.go
3. Create working examples from examples/ directory
4. Document best practices from code comments
5. Add performance notes from testing
6. Test all examples
7. Cross-link to related docs

**Acceptance Criteria (per feature doc)**:
- [ ] Clear purpose and use cases documented
- [ ] Complete configuration reference
- [ ] 2+ working examples (basic + advanced)
- [ ] Best practices section with 3+ items
- [ ] Common pitfalls documented
- [ ] Performance considerations included
- [ ] Links to 2+ related docs
- [ ] All examples tested

---

#### 2.5.2 Update Existing Ontology Documentation

**Files to Update** (5 files):

1. **`docs/features/ontology/README.md`**
   - Update examples to use named rules format
   - Add references to new advanced feature docs
   - Update workflow diagram to show validation step
   - Add attribute extraction to workflow

2. **`docs/features/ontology/quick-start.md`**
   - Update YAML examples to current format with named rules
   - Add attribute extraction quick example
   - Reference schema validation step
   - Update catalog loading examples

3. **`docs/features/ontology/domain-quickstart.md`**
   - Update to named rules format throughout
   - Add catalog system references
   - Show inheritance examples with parent type referencing
   - Add attribute extraction for domain entities

4. **`docs/features/ontology/examples.md`**
   - Update all examples to current format
   - Add attribute extraction examples
   - Add filter combination examples
   - Add cross-domain merging example

5. **`docs/features/ontology/workflows.md`**
   - Add validation workflow diagram
   - Add interview workflow diagram
   - Update extraction workflow with new filter stages
   - Add attribute extraction to extraction workflow

**Steps for Updates**:
1. Review current content for outdated examples
2. Replace with current YAML format
3. Add new sections for new features
4. Update diagrams (Mermaid)
5. Test all updated examples
6. Add cross-references to new docs

**Acceptance Criteria (per updated doc)**:
- [ ] All examples use current format
- [ ] References to new feature docs added
- [ ] Diagrams updated (if applicable)
- [ ] All examples tested
- [ ] Cross-links verified

---

### Phase 2.5 Summary

**Deliverables**:
- 8 new ontology feature documents
- 5 updated ontology documents
- Total: 13 documents created/updated

**Time Estimate**: 1.5 weeks
- Week 1: Create 4 new docs (rule-system, catalog-system, attribute-extraction, filters-and-validation)
- Week 2 (partial): Create 4 new docs (schema-validation, interview-cli, cross-domain-entities, llm-integration)
- Remaining time: Update 5 existing docs

**Success Metrics**:
- Complete ontology system documented
- Advanced features (attributes, validation, catalogs) explained
- 20+ working examples across all docs
- Users can leverage built-in catalogs
- Users can create custom domain ontologies
- Validation workflow documented

---

## Phase 3: Process & Workflow Documentation

**Duration**: 1 week

**Goal**: Explain how the system works end-to-end for developers and operators.

### Tasks

#### 3.1 Core Workflow Documents

**Files** (5 files):

1. **`docs/processes/document-processing-lifecycle.md`**
   - End-to-end flow: enqueue → claim → fetch → parse → embed → store
   - State transitions diagram
   - Error handling paths
   - Performance characteristics at each stage
   - Monitoring points

2. **`docs/processes/worker-coordination.md`**
   - Multi-worker job claiming (atomic operations)
   - Heartbeat protocol
   - Document reclamation (stale locks)
   - Leader election process
   - Graceful shutdown

3. **`docs/processes/discovery-recursion.md`**
   - How crawling works recursively
   - Depth tracking
   - Visited URL tracking
   - Queue population
   - Termination conditions

4. **`docs/processes/relationship-building.md`**
   - Structural relationship detection (parent-child)
   - Semantic relationship detection (embeddings)
   - Cross-document relationships
   - Relationship storage

5. **`docs/processes/error-handling-flow.md`**
   - Error classification (transient, permanent)
   - Retry logic (exponential backoff)
   - Max retries behavior
   - Failed document handling
   - Error reporting

**Template for Process Docs** (use `docs/_templates/process-template.md`):
- Overview
- Process flow (Mermaid diagram)
- Detailed steps
- State transitions
- Sequence diagram
- Concurrency considerations
- Performance characteristics
- Monitoring and observability
- Error handling
- Testing

**Steps for Each Process**:
1. Review implementation in `go/internal/worker/`
2. Create Mermaid flow diagram
3. Document state transitions
4. Add sequence diagram for interactions
5. Document timing/performance
6. Add monitoring log examples
7. Document error scenarios
8. Test all diagrams render on GitHub

**Acceptance Criteria (per process doc)**:
- [ ] Flow diagram (Mermaid)
- [ ] State transition diagram
- [ ] Sequence diagram (if multi-component)
- [ ] Timing/performance estimates
- [ ] Error handling documented
- [ ] Monitoring log examples

---

#### 3.2 Update Process Diagrams

**File**: `docs/PROCESS_DIAGRAMS.md` (existing - update)

**Updates**:
- Convert ASCII diagrams to Mermaid (GitHub-renderable)
- Add new diagrams from Phase 3.1
- Organize by category (ingestion, processing, storage)
- Add diagram index with links

**Steps**:
1. Review existing diagrams
2. Convert to Mermaid format
3. Add new diagrams from process docs
4. Create index/navigation
5. Test rendering on GitHub

**Acceptance Criteria**:
- [ ] All diagrams use Mermaid
- [ ] Diagrams render correctly on GitHub
- [ ] Index links to all diagrams
- [ ] Organized by category

---

### Phase 3 Summary

**Deliverables**: 5 process documents + updated PROCESS_DIAGRAMS.md

**Time Estimate**: 1 week
- Days 1-2: Document processing lifecycle + worker coordination
- Days 3-4: Discovery recursion + relationship building
- Day 5: Error handling + update PROCESS_DIAGRAMS.md

**Success Metrics**:
- 5 process workflows documented
- 10+ Mermaid diagrams created
- Developers understand system behavior
- Operators know how to monitor processes

---

## Phase 4: API Reference & Complete Godoc

**Duration**: 2 weeks

**Goal**: Provide comprehensive API documentation for all packages and interfaces.

### Tasks

#### 4.1 API Reference Documentation

**Files** (5 files):

1. **`docs/reference/api/README.md`** - API overview
   - Package organization
   - Key interfaces
   - Getting started as library user
   - Example: embedding Go-Doc-Go in another app

2. **`docs/reference/api/parser-interface.md`**
   - Parser interface definition
   - Implementing custom parsers
   - Parser registration
   - Testing custom parsers
   - Example: custom parser for new format

3. **`docs/reference/api/storage-interface.md`**
   - Storage interface definition
   - Implementing custom storage backends
   - Storage registration
   - Testing custom storage
   - Example: custom storage for Elasticsearch

4. **`docs/reference/api/content-source-api.md`**
   - ContentSource interface definition
   - Implementing custom content sources
   - Source registration
   - Testing custom sources
   - Example: custom source for Google Drive

5. **`docs/reference/api/programmatic-usage.md`**
   - Using Go-Doc-Go as a library
   - Embedding in applications
   - API stability guarantees
   - Versioning and compatibility
   - Example: full application using Go-Doc-Go

**Steps for Each API Doc**:
1. Review interface definition in code
2. Extract method signatures and documentation
3. Create usage examples
4. Document implementation requirements
5. Add testing guidance
6. Document error handling
7. Test all examples

**Acceptance Criteria (per API doc)**:
- [ ] Interface definition documented
- [ ] 2+ implementation examples
- [ ] Testing guidance provided
- [ ] Error handling documented
- [ ] All examples tested

---

#### 4.2 Data Model Documentation

**Files** (6 files - 4 data models + 2 schema reference):

**Data Models** (4 files):

1. **`docs/reference/data-models/elements.md`**
   - Element struct definition
   - All element types (from element_taxonomy.json)
   - Promoted fields (page_number, section_level, etc.)
   - Code-specific fields
   - JSON overflow fields
   - Examples for each element type

2. **`docs/reference/data-models/relationships.md`**
   - Relationship types
   - Structural vs semantic relationships
   - Cross-document relationships
   - Relationship storage format
   - Examples

3. **`docs/reference/data-models/parquet-schema.md`**
   - Parquet file structure
   - Hive partitioning scheme
   - Schema definitions (elements, relationships, embeddings)
   - Querying with DuckDB
   - Examples

4. **`docs/reference/data-models/database-schema.md`**
   - PostgreSQL schema (job_control tables)
   - SQLite schema
   - Indexes and constraints
   - Query examples
   - Migration strategy

**Schema Reference** (2 files):

5. **`docs/reference/schemas/README.md`**
   - Overview of all JSON schemas
   - Schema organization (UDML schemas vs Ontology schemas)
   - Links to detailed schema documentation
   - Validation workflow overview
   - IDE integration setup guide

6. **`docs/reference/schemas/ontology-compiler-schema.md`**
   - Complete field-by-field reference for ontology-compiler-1.0.schema.json
   - Required vs optional fields
   - Field types and constraints
   - Validation workflow using JSON schema
   - IDE integration setup (VSCode, JetBrains)
   - Common schema patterns and examples
   - Troubleshooting schema validation errors
   - Cross-references to configuration docs

**Steps for Each Data Model Doc**:
1. Review schema definitions in code
2. Generate schema diagrams
3. Document field types and constraints
4. Add query examples
5. Document relationships between tables/files
6. Test all examples

**Acceptance Criteria (per data model doc)**:
- [ ] Complete schema documented
- [ ] Field types and constraints listed
- [ ] 3+ query examples
- [ ] Relationships explained
- [ ] Examples tested

---

#### 4.3 Complete Godoc Coverage

**Goal**: Add godoc comments to remaining 205 Go files

**Approach**: Prioritize by package importance

**Priority 1** (most used packages - 20 files):
- `go/internal/graph/` - Graph building (5 files)
- `go/internal/cache/` - Caching (3 files)
- `go/internal/detector/` - Document type detection (2 files)
- `go/internal/temporal/` - Temporal analysis (3 files)
- `go/internal/importer/` - Import resolution (3 files)
- `go/internal/resolver/` - Path resolution (2 files)
- `go/internal/udml/ontology/` - Ontology extraction (2 files)

**Priority 2** (supporting packages - 30 files):
- `go/internal/udml/builder/` - Ontology builder (5 files)
- `go/internal/udml/query/` - Query system (8 files)
- `go/internal/udml/sampler/` - Sampling (3 files)
- `go/internal/udml/versioning/` - Versioning (4 files)
- Individual parsers - code parsers (10 files)

**Priority 3** (remaining packages - 155 files):
- Individual parsers - document parsers (60 files)
- Test files that need documentation (95 files)

**Template for Function Godoc**:
```go
// FunctionName performs [specific action].
//
// [Detailed description of what the function does, when to use it,
// and how it fits into the larger system.]
//
// Parameters:
//   - param1: Description of param1
//   - param2: Description of param2
//
// Returns:
//   - ReturnType: Description of return value
//   - error: Error conditions
//
// Example:
//
//	result, err := FunctionName(arg1, arg2)
//	if err != nil {
//		log.Fatal(err)
//	}
func FunctionName(param1 Type1, param2 Type2) (ReturnType, error) {
	// Implementation
}
```

**Steps**:
1. Run godoc coverage tool: `go doc -all ./... | grep "^package" | wc -l`
2. Identify files without package docs
3. For each package:
   - Add package-level doc.go or package comment
   - Document all exported types
   - Document all exported functions
   - Add examples for complex functions
4. Verify with `go doc` command
5. Check rendering on pkg.go.dev

**Acceptance Criteria**:
- [ ] 100% of packages have package-level docs
- [ ] 100% of exported types documented
- [ ] 100% of exported functions documented
- [ ] Key functions have examples
- [ ] Renders correctly on pkg.go.dev

---

### Phase 4 Summary

**Deliverables**:
- 5 API reference docs
- 4 data model docs
- 2 schema reference docs
- 205 Go files with godoc
- Total: 11 reference docs + godoc

**Time Estimate**: 2.5 weeks
- Week 1: 5 API docs + 4 data model docs + 2 schema reference docs
- Week 2-2.5: Priority 1 godoc (20 files) + Priority 2 godoc (30 files)
- Note: Priority 3 godoc (155 files) may extend into Phase 5 or be deprioritized for test files

**Success Metrics**:
- API interfaces fully documented
- Data models fully documented
- JSON schemas documented with IDE integration
- Godoc coverage > 80% (target 100%)
- Examples tested and working

---

## Phase 5: Extended Troubleshooting

**Duration**: 1 week

**Goal**: Comprehensive troubleshooting coverage for all common issues.

### Tasks

#### 5.1 Additional Troubleshooting Documents

**Files** (6 files):

1. **`docs/troubleshooting/parsing-errors.md`**
   - PDF-specific errors (10+ issues)
   - DOCX-specific errors (5+ issues)
   - Code parsing errors (5+ issues)
   - Data format errors (5+ issues)
   - Generic parsing issues (5+ issues)

2. **`docs/troubleshooting/performance-issues.md`**
   - Slow document processing
   - High memory usage
   - Slow embedding generation
   - Database connection pooling
   - Storage I/O bottlenecks

3. **`docs/troubleshooting/configuration-errors.md`**
   - TOML syntax errors
   - Required field validation
   - Type mismatches
   - Path resolution issues
   - Credential configuration

4. **`docs/troubleshooting/database-connection.md`**
   - PostgreSQL connection issues
   - SQLite file permissions
   - Connection pooling
   - Transaction deadlocks
   - Migration failures

5. **`docs/troubleshooting/worker-coordination.md`**
   - Stale locks
   - Heartbeat failures
   - Leader election issues
   - Multi-worker conflicts
   - Graceful shutdown

6. **`docs/troubleshooting/ontology-extraction.md`**
   - LLM API failures (rate limits, timeouts, quota exceeded)
   - LLM API error handling and retry logic
   - Schema validation errors (missing required fields, invalid types)
   - Low entity extraction confidence (tuning filters)
   - False positives (over-extraction, too many matches)
   - False negatives (under-extraction, missing expected entities)
   - Attribute extraction not working (scope issues, regex errors)
   - Cross-domain entity merging creating duplicates
   - Consolidation/canonicalization issues
   - Performance issues (slow LLM validation calls)
   - Named rule inheritance not working as expected
   - Filter pipeline debugging (which stage is failing)

**Template** (use `docs/_templates/troubleshooting-template.md`):
- Problem description
- Symptoms (error messages, behavior)
- Common causes
- Step-by-step solution
- Verification steps
- Prevention tips
- Related documentation

**Steps for Each Troubleshooting Doc**:
1. Review existing GitHub issues
2. Review support requests
3. Review code error messages
4. Identify top 5-10 issues per category
5. Test each issue reproduction
6. Document solution steps
7. Verify solutions work
8. Add prevention tips

**Acceptance Criteria (per troubleshooting doc)**:
- [ ] 5+ issues documented (10+ for ontology-extraction.md)
- [ ] Each issue has symptoms, causes, solution
- [ ] Solutions tested and verified
- [ ] Prevention tips included
- [ ] Links to related docs

---

#### 5.2 Troubleshooting Index

**File**: `docs/troubleshooting/README.md`

**Content**:
- Overview of troubleshooting resources
- Quick diagnostic flowchart
- Issue index by category
- Links to all troubleshooting docs
- How to report bugs

**Steps**:
1. Create diagnostic flowchart (Mermaid)
2. Index all issues by category
3. Add search keywords for each issue
4. Link to all troubleshooting docs
5. Add bug reporting template

**Acceptance Criteria**:
- [ ] Diagnostic flowchart
- [ ] Complete issue index
- [ ] Search keywords for quick lookup
- [ ] Bug reporting instructions

---

### Phase 5 Summary

**Deliverables**:
- 6 troubleshooting category docs
- 1 troubleshooting index
- 35+ additional issues documented (5+ per category, 10+ for ontology)

**Time Estimate**: 1 week
- Days 1-2: Parsing errors + performance issues
- Days 3-4: Configuration + database + worker coordination
- Day 5: Ontology extraction (10+ issues) + troubleshooting index

**Success Metrics**:
- Top 35 issues documented (total 51 with Phase 1's 16)
- Most categories have 5+ issues, ontology has 10+
- Diagnostic flowchart created
- Users can self-service common problems

---

## Phase 6: Operations & Deployment

**Duration**: 2 weeks

**Goal**: Enable production deployments with comprehensive operational guides.

### Tasks

#### 6.1 Deployment Guides

**Files** (4 files):

1. **`docs/operations/deployment/production-checklist.md`**
   - Pre-deployment checklist (30+ items)
   - Configuration validation
   - Resource requirements
   - Security hardening
   - Backup strategy
   - Monitoring setup
   - Disaster recovery

2. **`docs/operations/deployment/docker.md`**
   - Dockerfile best practices
   - Multi-stage builds
   - Environment variables
   - Volume mounts
   - Docker Compose examples
   - Health checks
   - Resource limits

3. **`docs/operations/deployment/kubernetes.md`**
   - Deployment manifests
   - StatefulSet for workers
   - ConfigMaps and Secrets
   - Persistent volumes
   - Horizontal Pod Autoscaling
   - Service definitions
   - Ingress configuration
   - Helm charts (optional)

4. **`docs/operations/deployment/security.md`**
   - Authentication and authorization
   - Credential management (secrets)
   - Network security (TLS, firewalls)
   - Database security (connection encryption, user permissions)
   - API security (if applicable)
   - Audit logging
   - Compliance considerations

**Steps for Each Deployment Doc**:
1. Create example configurations
2. Test deployments
3. Document step-by-step instructions
4. Add troubleshooting section
5. Include resource requirements
6. Document monitoring integration
7. Test all examples

**Acceptance Criteria (per deployment doc)**:
- [ ] Complete working example
- [ ] Step-by-step instructions
- [ ] Resource requirements documented
- [ ] Security considerations covered
- [ ] Troubleshooting included
- [ ] Examples tested

---

#### 6.2 Operations Guides

**Files** (3 files):

1. **`docs/operations/backup-recovery.md`**
   - Backup strategies (Parquet files, databases, models)
   - Backup frequency recommendations
   - Restore procedures
   - Point-in-time recovery
   - Disaster recovery testing
   - Data retention policies

2. **`docs/operations/performance-tuning.md`**
   - Worker pool sizing
   - Batch size optimization
   - Memory tuning
   - Database connection pooling
   - Embedding model optimization
   - Storage I/O optimization
   - Benchmarking methodology

3. **`docs/operations/upgrade-migration.md`**
   - Version compatibility matrix
   - Upgrade procedures (rolling, blue-green)
   - Configuration migration
   - Database migrations
   - Breaking changes by version
   - Rollback procedures
   - Testing upgrades

**Steps for Each Operations Doc**:
1. Document best practices from production use
2. Create tuning examples
3. Add performance benchmarks
4. Document monitoring integration
5. Test all procedures
6. Add troubleshooting

**Acceptance Criteria (per operations doc)**:
- [ ] Complete procedures documented
- [ ] Examples tested
- [ ] Performance benchmarks included
- [ ] Monitoring integration documented
- [ ] Troubleshooting included

---

### Phase 6 Summary

**Deliverables**:
- 4 deployment guides
- 3 operations guides
- Production-ready documentation

**Time Estimate**: 2 weeks
- Week 1: 4 deployment guides (checklist, docker, kubernetes, security)
- Week 2: 3 operations guides (backup, tuning, upgrades)

**Success Metrics**:
- Production deployment checklist (30+ items)
- Docker and Kubernetes examples tested
- Security hardening guide complete
- Operational procedures documented

---

## Phase 7: Developer Guides

**Duration**: 1 week

**Goal**: Enable contributors to effectively contribute to the project.

### Tasks

#### 7.1 Contribution Documentation

**Files** (7 files):

1. **`CONTRIBUTING.md`** (root level)
   - How to contribute (code, docs, issues)
   - Code of conduct
   - Pull request process
   - Code review guidelines
   - Testing requirements
   - Documentation requirements
   - Community channels

2. **`docs/development/architecture-overview.md`**
   - System architecture diagram
   - Component interactions
   - Data flow
   - Design decisions
   - Architectural patterns used
   - Scalability considerations

3. **`docs/development/adding-parsers.md`**
   - Parser interface overview
   - Step-by-step guide to adding new parser
   - Parser registration
   - Element type selection
   - Testing parsers
   - Example: implementing a new parser

4. **`docs/development/custom-storage.md`**
   - Storage interface overview
   - Implementing custom backends
   - Storage registration
   - Testing custom storage
   - Example: implementing S3 storage

5. **`docs/development/custom-content-source.md`**
   - ContentSource interface overview
   - Implementing custom sources
   - Source registration
   - Testing custom sources
   - Example: implementing SFTP source

6. **`docs/development/testing-guide.md`**
   - Test organization (unit, integration, e2e)
   - Running tests
   - Writing tests
   - Test coverage requirements
   - Test data management
   - CI/CD integration
   - Performance testing

7. **`docs/development/code-standards.md`**
   - Extract from CLAUDE.md
   - Go code style
   - Naming conventions
   - Error handling patterns
   - Documentation requirements
   - Git commit conventions
   - Pre-commit checks

**Steps for Each Development Doc**:
1. Review CLAUDE.md and DEVELOPMENT.md
2. Extract relevant content
3. Add detailed examples
4. Create step-by-step guides
5. Test all examples
6. Add troubleshooting

**Acceptance Criteria (per development doc)**:
- [ ] Step-by-step instructions
- [ ] Working examples
- [ ] Testing guidance
- [ ] Troubleshooting included
- [ ] Links to related docs

---

### Phase 7 Summary

**Deliverables**:
- 1 contribution guide (CONTRIBUTING.md)
- 6 development guides

**Time Estimate**: 1 week
- Days 1-2: CONTRIBUTING.md + architecture overview + testing guide
- Days 3-4: Adding parsers + custom storage + custom content source
- Day 5: Code standards + review

**Success Metrics**:
- New contributors can get started quickly
- Adding parsers documented step-by-step
- Testing procedures clear
- Code standards established

---

## Implementation Timeline

### Summary

| Phase | Duration | Deliverables | Status |
|-------|----------|--------------|--------|
| 1. Foundation | 2 weeks | 11 items | ✅ Complete |
| 2. Features | 2 weeks | 10 docs | ⏳ Next |
| **2.5. Advanced Ontology** | **1.5 weeks** | **13 docs** | **📋 Planned** |
| 3. Processes | 1 week | 6 docs | 📋 Planned |
| 4. API/Godoc | 2.5 weeks | 11 docs + 205 files | 📋 Planned |
| 5. Troubleshooting | 1 week | 7 docs | 📋 Planned |
| 6. Operations | 2 weeks | 7 docs | 📋 Planned |
| 7. Development | 1 week | 7 docs | 📋 Planned |

**Total**: 12.5 weeks for comprehensive coverage

### Milestones

**Week 3-4 (Phase 2 Complete)**: Core feature documentation
- Parser overview complete
- All parser docs complete (6 docs)
- Discovery system documented (3 docs)

**Week 5-6 (Phase 2.5 Complete)**: Advanced ontology documentation
- 8 new ontology feature docs complete
- 5 existing ontology docs updated
- Ontology system fully documented

**Week 7 (Phase 3 Complete)**: Processes documented
- 5 workflow docs complete
- Updated process diagrams

**Week 8-10 (Phase 4 Complete)**: API reference complete
- All API docs complete
- Data models documented
- Schema reference docs complete
- Priority 1 & 2 godoc complete

**Week 11 (Phase 5 Complete)**: Extended troubleshooting
- 51 total issues documented (16 from Phase 1 + 35 new)
- Diagnostic flowchart complete
- Ontology troubleshooting comprehensive (10+ issues)

**Week 12-13 (Phase 6 Complete)**: Operations ready
- Production deployment guides complete
- Operational procedures documented

**Week 14 (Phase 7 Complete)**: Contribution ready
- All documentation complete
- Project ready for contributors

---

## Success Metrics

### Coverage Metrics

**Godoc Coverage**:
- Current: 8 packages (3.7%)
- Target: 100% of packages
- Minimum: 80% (acceptable for deprioritized test files)

**Issue Coverage**:
- Current: 16 issues documented
- Target: 51 issues (35 new + 16 from Phase 1)
- Ontology issues: 10+ issues with solutions

**Feature Coverage**:
- Current: 0 parsers fully documented
- Target: 28+ parsers documented

**Ontology Coverage** (NEW):
- Current: 5 basic docs (README, quick-start, etc.)
- Target: 13 comprehensive docs (8 new + 5 updated)
- Advanced features: Attribute extraction, validation, catalogs, filters
- Built-in catalogs: 42 catalog files across 6 W categories + domains

**API Coverage**:
- Current: 0 interfaces documented
- Target: 4 major interfaces + data models
- Schema coverage: 2 schema reference docs (UDML + Ontology Compiler)

### Quality Metrics

**Every Document Must Have**:
- [ ] Working examples (all tested)
- [ ] Links to related documentation
- [ ] Clear structure (from templates)
- [ ] Proper formatting (Markdown linting passes)

**Every Code Example Must**:
- [ ] Be tested and working
- [ ] Include expected output
- [ ] Have comments explaining non-obvious parts
- [ ] Follow documentation standards

**Every Diagram Must**:
- [ ] Use Mermaid (GitHub-renderable)
- [ ] Render correctly on GitHub
- [ ] Be referenced from text
- [ ] Have a caption/description

### User Impact Metrics

**Reduction in Support Burden**:
- Target: 50% reduction in "how do I..." issues
- Target: 30% reduction in duplicate issues

**Time to Productivity**:
- New users: < 30 minutes to process first document
- New contributors: < 30 minutes to setup dev environment
- New operators: < 1 hour to deploy to production

**Self-Service Success**:
- Target: 70% of common issues solved via docs (no support needed)

---

## Prioritization Strategy

### If Time-Constrained

**Must Have** (Phases 1-2.5 + partial Phase 4):
1. ✅ Phase 1: Foundation (complete)
2. Phase 2: Feature docs (parsers, discovery - critical for users)
3. Phase 2.5: Advanced Ontology (critical for ontology users)
4. Phase 3: Process docs (critical for developers)
5. Phase 4: API reference + Priority 1 godoc (critical for library users)

**Should Have** (Complete Phase 4 + Phases 5-6):
6. Phase 4: Complete godoc (nice to have for all files)
7. Phase 5: Extended troubleshooting (reduces support burden)
8. Phase 6: Operations guides (enables production use)

**Nice to Have** (Phase 7):
9. Phase 7: Development guides (enables contributions)

### Quick Wins for Immediate Impact

If only 1 week available:
1. Complete Phase 2.1 (Parser overview) - high user value
2. Complete Phase 2.3 (Discovery docs) - fills major gap
3. Complete Phase 2.5.1 partial (2-3 critical ontology docs) - fills ontology gap

If only 2 weeks available:
- Complete Phase 2 (Features)
- Complete Phase 2.5 partial (4-5 ontology docs)

If only 3-4 weeks available:
- Complete Phase 2 (Features)
- Complete Phase 2.5 (Advanced Ontology)
- Complete Phase 3 (Processes)

---

## Maintenance Plan

### Ongoing Documentation

**When Adding Code**:
- Add godoc for all new packages/functions
- Update affected documentation
- Add examples for complex features
- Update troubleshooting if new error cases

**When Fixing Bugs**:
- Update troubleshooting docs
- Add example showing the issue and fix
- Update limitations/known issues

**When Changing APIs**:
- Update API reference docs
- Update affected examples
- Mark deprecated features
- Document migration path

**When Adding Features**:
- Add feature documentation
- Update process docs if workflow changes
- Add configuration reference
- Add examples

### Documentation Review

**Quarterly** (every 3 months):
- Review godoc coverage (run coverage tool)
- Review troubleshooting docs (are they still relevant?)
- Update examples (test all examples still work)
- Check for broken links
- Update version-specific information

**On Each Release**:
- Update version numbers in docs
- Document breaking changes
- Update upgrade guide
- Update compatibility matrix
- Review and update examples

### Tools

**Markdown Linting**:
```bash
# Install markdownlint
npm install -g markdownlint-cli

# Lint all docs
markdownlint docs/**/*.md
```

**Link Checking**:
```bash
# Install markdown-link-check
npm install -g markdown-link-check

# Check all links
find docs -name '*.md' -exec markdown-link-check {} \;
```

**Godoc Coverage**:
```bash
# Check package documentation coverage
go doc -all ./... | grep "^package" | wc -l

# Generate godoc locally
godoc -http=:6060
# Visit http://localhost:6060/pkg/github.com/kennethstott/doculyzer-go-conversion/
```

**Example Testing**:
```bash
# Extract and test code examples from markdown
# (Custom script needed)
./scripts/test-examples.sh
```

---

## Getting Started with Phase 2

### Next Steps (Week 3)

**Day 1**: Parser overview research
- [ ] List all parsers from `go/internal/parser/`
- [ ] Create comparison table structure
- [ ] Gather performance benchmarks

**Day 2**: Parser overview writing
- [ ] Write `docs/features/parsing/overview.md`
- [ ] Add quick start examples
- [ ] Test all examples

**Day 3**: PDF parsing documentation
- [ ] Write `docs/features/parsing/pdf-parsing.md`
- [ ] Create examples with sample PDFs
- [ ] Test examples

**Day 4**: Office documents documentation
- [ ] Write `docs/features/parsing/office-documents.md`
- [ ] Create examples for DOCX, XLSX, PPTX
- [ ] Test examples

**Day 5**: Code parsing documentation
- [ ] Update `docs/features/code-parsing.md`
- [ ] Add language-specific examples
- [ ] Test examples

### First Milestone

**Target**: End of Week 3
- [ ] Parser overview complete
- [ ] 3 individual parser docs complete
- [ ] 5+ working examples tested

---

## Resources

### Templates
- `docs/_templates/troubleshooting-template.md`
- `docs/_templates/feature-template.md`
- `docs/_templates/process-template.md`

### Standards
- `docs/DOCUMENTATION_STANDARDS.md`

### Examples
- Existing good docs: `docs/features/ontology/`
- Existing process diagrams: `docs/PROCESS_DIAGRAMS.md`

### Tools
- Markdown linter: markdownlint
- Link checker: markdown-link-check
- Godoc: go doc
- Mermaid: https://mermaid.js.org/

---

## Notes

### Design Decisions

1. **Godoc First**: Package-level docs in Phase 1, complete coverage in Phase 4
2. **Templates**: Ensure consistency across all documentation types
3. **Examples**: Every doc must have tested, working examples
4. **Diagrams**: Use Mermaid for GitHub compatibility
5. **Cross-linking**: Extensive linking between related docs

### Risks and Mitigations

**Risk**: Godoc coverage (205 files) too time-consuming
**Mitigation**: Prioritize by package importance, accept 80% coverage

**Risk**: Examples break as code changes
**Mitigation**: Add example testing to CI/CD

**Risk**: Documentation becomes stale
**Mitigation**: Quarterly review process, update-on-change policy

**Risk**: Too much documentation, hard to navigate
**Mitigation**: Strong README.md hub, clear organization, search keywords

---

## Questions and Clarifications

**When to Start Phase 2?**
- Ready to start immediately (Phase 1 complete)
- Recommended: Start with parser overview

**Who Will Write the Docs?**
- Developer(s) familiar with codebase
- Technical writer (optional, for polish)
- Community contributions (encouraged)

**How to Track Progress?**
- This document (update status as you go)
- GitHub issues (one per phase or per document)
- GitHub project board (recommended)

**How to Get Feedback?**
- Internal review before publishing
- Community review after publishing
- Iterate based on user questions/issues

---

**Last Updated**: 2025-11-02
**Status**: Phase 1 Complete, Phase 2 Ready to Start, Plan Updated with Phase 2.5 (Advanced Ontology)
**Next Review**: After Phase 2.5 completion
