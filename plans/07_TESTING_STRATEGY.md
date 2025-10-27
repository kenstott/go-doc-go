# Testing Strategy

## Overview

Comprehensive testing approach covering unit tests, integration tests, and validation scripts.

---

## Unit Tests

### Hierarchy Validation Tests

**File**: `go/internal/udml/ontology/types_test.go`

**Tests**:

1. **TestComputeHierarchies_BidirectionalFill**
   - Verifies parent's `children` auto-filled from child's `parent_type`
   - Verifies child's `parent_type` auto-filled from parent's `children`

2. **TestComputeHierarchies_MultipleChildren**
   - Multiple children reference same parent
   - Parent's `children` array contains all children

3. **TestValidateHierarchies_BrokenParentReference**
   - Entity references non-existent `parent_type`
   - Validation returns error

4. **TestValidateHierarchies_BrokenChildReference**
   - Entity references non-existent child in `children` array
   - Validation returns error

5. **TestValidateHierarchies_CircularDetection**
   - Two entities form circular loop
   - Validation detects and returns error

6. **TestValidateHierarchies_MultiLevelChain**
   - Valid 3-level hierarchy (person → physician → surgeon)
   - Validation passes

7. **TestValidateHierarchies_MultipleDef initionsAllowed**
   - Same entity_type with different `parent_type`/`w_category`
   - Validation passes (allowed by design)

**Run**:
```bash
cd go/internal/udml/ontology
go test -v -run TestComputeHierarchies
go test -v -run TestValidateHierarchies
```

---

### Hierarchy Materialization Tests

**File**: `go/internal/udml/ontology/extractor_test.go`

**Tests**:

1. **TestMaterializeHierarchy_SimpleChain**
   - Input: surgeon["Dr. Smith"]
   - Expected: physician["Dr. Smith"], person["Dr. Smith"] (composites)
   - Expected: 2 IS-A relationships

2. **TestMaterializeHierarchy_MultiLevelChain**
   - Input: surgeon["Dr. Smith"]
   - Hierarchy: person → physician → surgeon (3 levels)
   - Expected: 2 composites, 2 IS-A relationships

3. **TestMaterializeHierarchy_Deduplication**
   - Input: surgeon["Dr. Smith"], cardiologist["Dr. Smith"] (same element)
   - Expected: 1 shared physician composite (deduped by name+type+element)

4. **TestMaterializeHierarchy_DifferentElements**
   - Input: surgeon["Dr. Smith"] (element A), surgeon["Dr. Smith"] (element B)
   - Expected: 2 separate physician composites (different elements)

5. **TestMaterializeHierarchy_NoParent**
   - Input: person["John Doe"] (top-level, no parent)
   - Expected: 0 composites, 0 IS-A relationships

6. **TestMaterializeHierarchy_ConfidenceInheritance**
   - Verify composite entities have parent mapping's confidence
   - Not inherited from child

**Run**:
```bash
cd go/internal/udml/ontology
go test -v -run TestMaterializeHierarchy
```

---

### Unified Extraction Tests

**File**: `go/internal/udml/ontology/extractor_test.go`

**Tests**:

1. **TestTryExtractWithRule_PhraseList**
   - Rule with `phrase_list` only
   - Exact string matching works

2. **TestTryExtractWithRule_InstanceName**
   - Rule with `instance_name` regex
   - Capture group extraction works

3. **TestTryExtractWithRule_JSONPathOptional**
   - Rule without `jsonpath` (fast path)
   - Extracts from content directly

4. **TestTryExtractWithRule_JSONPathPresent**
   - Rule with `jsonpath: $.metadata.author`
   - Navigates to nested field

5. **TestTryExtractWithRule_FailFastPattern**
   - Pattern filter fails
   - Returns nil immediately (doesn't check other filters)

6. **TestTryExtractWithRule_AllFiltersPassed**
   - Pattern + Proximity + Dictionary + Semantic all present
   - All pass, entity extracted

7. **TestFindPhraseMatch_ExactMatching**
   - Phrase list: ["CEO", "CFO", "CTO"]
   - Content contains "CFO"
   - Returns "CFO"

8. **TestFindPhraseMatch_LongestMatch**
   - Phrase list: ["Dr. Smith", "Dr."]
   - Content contains "Dr. Smith"
   - Returns "Dr. Smith" (longer match)

**Run**:
```bash
cd go/internal/udml/ontology
go test -v -run TestTryExtractWithRule
go test -v -run TestFindPhraseMatch
```

---

## Integration Tests

### End-to-End Ontology Interview

**Build binary**:
```bash
cd go
go build -o ../bin/ontology ./cmd/ontology
```

**Run interview** (non-interactive):
```bash
bin/ontology interview \
  --config tests/test_configs/test.toml \
  --output tests/test_output/ontology_results/refactored_schema.json \
  --non-interactive \
  --verbose
```

**Verify output**:
```bash
# Check schema structure
cat tests/test_output/ontology_results/refactored_schema.json | jq '.element_entity_mappings | length'

# Verify hierarchy relationships
cat tests/test_output/ontology_results/refactored_schema.json | jq '
  .element_entity_mappings[] |
  select(.parent_type != null) |
  {entity_type, parent_type, children, w_category}
'

# Check for Type field (should be absent)
cat tests/test_output/ontology_results/refactored_schema.json | jq '
  .element_entity_mappings[].extraction_rules[] |
  select(.type != null) |
  "ERROR: Type field found"
'
```

**Expected results**:
- Schema loads successfully
- All entities have `w_category`
- Entities with `parent_type` have auto-filled `children`
- No `type` field in extraction rules
- Validation passes

---

### Extraction with Hierarchy Materialization

**Run extraction**:
```bash
bin/ontology extract \
  --schema tests/test_output/ontology_results/refactored_schema.json \
  --parquet tests/test_output/wikipedia_medical_mining_analytics \
  --job-db tests/test_output/ontology_results/extraction_test.db \
  --doc-batch-size 50 \
  --verbose
```

**Verify output**:
```bash
# Check extracted entities (should include composites)
sqlite3 tests/test_output/ontology_results/extraction_test.db "
  SELECT entity_type, COUNT(*), 
         SUM(CASE WHEN json_extract(attributes, '$.composite') = 1 THEN 1 ELSE 0 END) as composite_count
  FROM entities 
  GROUP BY entity_type;
"

# Check IS-A relationships
sqlite3 tests/test_output/ontology_results/extraction_test.db "
  SELECT COUNT(*) FROM relationships WHERE type = 'is_a';
"

# Verify hierarchy chain (example: surgeon → physician → person)
sqlite3 tests/test_output/ontology_results/extraction_test.db "
  WITH RECURSIVE hierarchy AS (
    SELECT id, name, type, 0 as level FROM entities WHERE type = 'surgeon' LIMIT 1
    UNION ALL
    SELECT e.id, e.name, e.type, h.level + 1
    FROM entities e
    JOIN relationships r ON r.target_id = e.id
    JOIN hierarchy h ON h.id = r.source_id
    WHERE r.type = 'is_a' AND h.level < 5
  )
  SELECT level, type, name FROM hierarchy ORDER BY level;
"
```

**Expected results**:
- Leaf entities extracted (surgeon, cardiologist, etc.)
- Composite entities created (physician, person)
- IS-A relationships linking children to parents
- Deduplication working (same name+element → single composite)

---

## Validation Scripts

### Schema Validation

**Verify hierarchy computation**:
```bash
# Create test schema with missing children
cat > /tmp/test_schema.yaml << 'YAML'
element_entity_mappings:
  - entity_type: person
    domain: global
    parent_type: ""
    children: []          # Empty - should be auto-filled
  - entity_type: physician
    domain: medical
    parent_type: global.person
    children: []
YAML

# Load schema and check computation
go run ./cmd/ontology validate --schema /tmp/test_schema.yaml
```

**Expected**: After loading, `person.children` contains `["medical.physician"]`

---

### Broken Reference Detection

**Test orphaned parent**:
```bash
cat > /tmp/broken_schema.yaml << 'YAML'
element_entity_mappings:
  - entity_type: physician
    domain: medical
    parent_type: global.foobar  # Doesn't exist
YAML

go run ./cmd/ontology validate --schema /tmp/broken_schema.yaml
```

**Expected error**: `entity medical.physician references non-existent parent: global.foobar`

---

### Circular Hierarchy Detection

**Test circular reference**:
```bash
cat > /tmp/circular_schema.yaml << 'YAML'
element_entity_mappings:
  - entity_type: physician
    domain: medical
    parent_type: medical.surgeon
  - entity_type: surgeon
    domain: medical
    parent_type: medical.physician
YAML

go run ./cmd/ontology validate --schema /tmp/circular_schema.yaml
```

**Expected error**: `circular hierarchy detected involving: medical.physician`

---

## Performance Testing

### Extraction Performance

**Benchmark different rule types**:
```bash
# Phrase list (should be fastest)
go test -bench=BenchmarkPhraseListExtraction -benchmem

# Instance name regex (fallback)
go test -bench=BenchmarkInstanceNameExtraction -benchmem

# With filters (progressive cost)
go test -bench=BenchmarkWithFilters -benchmem
```

**Expected**:
- Phrase list: 10-100x faster than regex
- Pattern filter: Cheap (< 1ms per element)
- Proximity filter: Moderate (1-5ms per element)
- Semantic filter: Expensive (10-50ms per element)

---

### Hierarchy Materialization Performance

**Benchmark materialization**:
```bash
go test -bench=BenchmarkMaterializeHierarchy -benchmem
```

**Measure**:
- Time complexity: O(m * d) where m = entities, d = depth
- Memory: O(m * d) for composites
- Deduplication cache effectiveness

---

## Test Data

### Sample Schemas

**Valid schema** (`tests/fixtures/valid_schema.yaml`):
- Global domain with 10 types
- Medical domain with 5 types extending global
- All hierarchies valid

**Broken schema** (`tests/fixtures/broken_schema.yaml`):
- Orphaned parent reference
- For testing error handling

**Circular schema** (`tests/fixtures/circular_schema.yaml`):
- Circular hierarchy
- For testing cycle detection

### Sample Documents

**Medical document** (`tests/fixtures/medical_sample.json`):
- Contains: physicians, organizations, locations
- For testing extraction + materialization

**Multi-domain document** (`tests/fixtures/multi_domain_sample.json`):
- Contains entities from multiple domains
- For testing cross-domain relationships

---

## Continuous Integration

### Pre-Commit Checks

```bash
#!/bin/bash
# tests/pre_commit.sh

echo "Running pre-commit checks..."

# Build check
go build ./... || exit 1
echo "✓ Build passed"

# Unit tests
go test ./... || exit 1
echo "✓ Unit tests passed"

# Lint
golangci-lint run ./... || exit 1
echo "✓ Lint passed"

# Format check
if [ -n "$(gofmt -l .)" ]; then
    echo "✗ Code needs formatting"
    exit 1
fi
echo "✓ Format check passed"

# Integration test
bin/ontology interview \
  --config tests/test_configs/test.toml \
  --output /tmp/test_schema.json \
  --non-interactive || exit 1
echo "✓ Integration test passed"

echo "All checks passed!"
```

---

## Next Steps

Proceed to:
- [08_MIGRATION_GUIDE.md](08_MIGRATION_GUIDE.md) - Update domain catalogs
- [06_IMPLEMENTATION_STEPS.md](06_IMPLEMENTATION_STEPS.md) - Start implementing
