# Quality Gates - NON-NEGOTIABLE

This document defines the mandatory quality gates that MUST be satisfied before claiming any task as complete. These standards are adapted from enterprise Java project guidelines to ensure reliability and maintainability.

## Definition: When is a Task "Complete"?

A task is ONLY complete when ALL of the following pass:

### 1. Functionality Verified ✅
- [ ] **Real execution** with production-like inputs and expected outputs
- [ ] **End-to-end functionality** works without manual intervention
- [ ] **All public methods** return meaningful data (not empty lists/None)
- [ ] **No `NotImplementedError`** in production code paths
- [ ] **Error handling** covers expected failure modes

### 2. Tests Passing ✅
- [ ] **All unit tests** green: `pytest -m unit`
- [ ] **All integration tests** green for modified components
- [ ] **Performance tests** meet SLA requirements where applicable
- [ ] **New functionality** has corresponding test coverage
- [ ] **Test execution output** provided as evidence

### 3. Code Quality ✅
- [ ] **No compilation errors**: `python -m py_compile src/go_doc_go/**/*.py`
- [ ] **Linting passes**: `flake8 src/ tests/`
- [ ] **Type checking passes**: `mypy src/`
- [ ] **Code formatting**: `black --check src/ tests/`
- [ ] **No debug artifacts**: No `print()` statements, temporary files, or commented code
- [ ] **Proper error logging** with appropriate context

### 4. Documentation Current ✅
- [ ] **API documentation** reflects actual behavior
- [ ] **Code comments** explain complex logic and design decisions
- [ ] **README/guides** updated for user-facing changes
- [ ] **Stub code** properly marked with `# STUB:` and TODO comments

### 5. Regression Tested ✅
- [ ] **Related functionality** still works correctly
- [ ] **Existing tests** still pass after changes
- [ ] **Performance** has not degraded for existing features
- [ ] **Integration points** verified if interfaces changed

### 6. Performance Verified ✅
- [ ] **SLAs met** under expected load (see Performance SLAs below)
- [ ] **Memory usage** within acceptable bounds
- [ ] **Response times** meet requirements
- [ ] **Concurrency** handling verified for multi-threaded components

---

## Performance SLAs by Component

### Document Processing
- Standard document (< 10MB): Parse in < 1 second
- Large document (< 100MB): Parse in < 10 seconds  
- Memory usage: < 5x document size
- Concurrent parsing: Support 10 simultaneous parsers

### Work Queue System  
- Document claiming latency: < 10ms per document
- Sustained throughput: > 1000 docs/second with 10 concurrent workers
- Memory usage per worker: < 100MB base memory
- Maximum concurrent workers: 50 workers supported
- Atomic operations: No duplicate claims under high concurrency

### Database Operations
- Query response time: < 100ms for typical operations
- Bulk operations: > 500 inserts/second
- Connection pooling: Support 50 concurrent connections
- Transaction consistency: All ACID properties maintained

---

## Verification Evidence Required

When claiming completion, you MUST provide:

### 1. Test Execution Output
```bash
# Example evidence format
$ pytest tests/test_queue/test_work_queue.py::TestWorkQueueIntegration -v
======================= 8 passed in 12.34s =======================

$ pytest -m performance tests/test_queue/test_work_queue.py::TestWorkQueuePerformance::test_concurrent_throughput -v
test_concurrent_throughput PASSED - Throughput: 1,459.5 docs/second
```

### 2. Sample Data Demonstration  
```python
# Show actual input/output with real data
input_document = {
    "id": "real_doc.pdf",
    "content": "<actual PDF bytes>",
    "metadata": {"source": "test"}
}

result = parser.parse(input_document)
# result = {
#   "document": {...},
#   "elements": [47 elements parsed],
#   "relationships": [23 relationships found]
# }
```

### 3. Performance Metrics
```
Memory Usage: 45MB (within 100MB limit)
Processing Time: 0.85s (within 1.0s limit)  
Throughput: 1,459 docs/second (exceeds 1,000 minimum)
Concurrency: 10 workers, 0 duplicate claims (atomic requirement met)
```

### 4. Command History
```bash
# Exact commands used for verification
pytest -m unit --cov=src/go_doc_go --cov-report=term
flake8 src/ tests/
mypy src/
./scripts/pre-commit-checklist.sh
```

---

## Common Quality Gate Failures

### ❌ FAILURE: "Implementation" is Actually Stubs
```python
# This is NOT complete - it's a stub
def claim_next_document(self, run_id: str) -> Optional[Dict]:
    """Claim next document from queue."""
    # TODO: Implement atomic claiming logic
    return None
```

**Fix**: Implement the actual functionality and test with real data.

### ❌ FAILURE: Tests Pass But Don't Test Real Functionality
```python  
# This test is meaningless - it tests nothing
def test_parser_works():
    parser = SomeParser()
    assert parser is not None
```

**Fix**: Test actual parsing with real documents and verify outputs.

### ❌ FAILURE: Performance Not Verified
```
"The queue system is complete and handles concurrency."
```

**Fix**: Provide evidence - run concurrent tests, measure throughput, verify no race conditions.

### ❌ FAILURE: Error Handling Missing
```python
# This will crash in production
def parse_document(content):
    return json.loads(content["data"])  # What if content["data"] doesn't exist?
```

**Fix**: Add proper error handling, logging, and test error scenarios.

---

## Quality Gate Enforcement

### Pre-Commit Requirements
- **MANDATORY**: Run `./scripts/pre-commit-checklist.sh` before every commit
- **MANDATORY**: All checks must pass before code can be committed
- **MANDATORY**: Provide evidence when claiming task completion

### Code Review Requirements
- **Reviewer MUST verify** all quality gates before approval
- **Evidence MUST be provided** in PR description or comments
- **Performance tests MUST be run** for performance-critical changes
- **Documentation MUST be updated** for API changes

### Escalation Protocol
If quality gates cannot be satisfied:
1. **Document specific technical obstacles** preventing completion
2. **Show attempted solutions** and their failures  
3. **Request specific guidance** or approval to defer
4. **Never abandon work** without explicit approval

---

## Implementation Honesty - ZERO TOLERANCE

### PROHIBITED Practices
- ❌ Claiming "implemented" when only stubs exist
- ❌ Saying "it should work" without testing
- ❌ Committing with `feat:` prefix for non-working code
- ❌ Ignoring test failures or commenting out failing tests
- ❌ Claiming completion without providing verification evidence

### REQUIRED Practices  
- ✅ Be explicit about what is implemented vs stubbed
- ✅ Test with real data and provide evidence
- ✅ Use appropriate commit prefixes (`wip:`, `stub:`, `feat:`)
- ✅ Fix failing tests, don't ignore them
- ✅ Provide verification evidence for all completion claims

---

## Quality Gate Checklist Template

Copy this checklist for every significant task:

```
## Quality Gates Verification

### Functionality ✅
- [ ] Tested with real data: [command/evidence]
- [ ] End-to-end functionality verified: [evidence]
- [ ] No NotImplementedError in production paths
- [ ] Error handling covers expected failures

### Tests ✅  
- [ ] Unit tests pass: [command output]
- [ ] Integration tests pass: [command output]
- [ ] Performance tests pass: [metrics]
- [ ] Coverage meets requirements: [percentage]

### Code Quality ✅
- [ ] Pre-commit checklist passes: `./scripts/pre-commit-checklist.sh`
- [ ] No debug artifacts remain
- [ ] Proper error logging implemented

### Documentation ✅
- [ ] API docs updated for changes
- [ ] Code comments explain complex logic
- [ ] Stub code properly marked if any

### Performance ✅
- [ ] SLAs verified: [specific metrics]
- [ ] Memory usage acceptable: [measurements]
- [ ] No performance regressions

### Evidence Provided
- [ ] Test execution output attached
- [ ] Sample input/output data shown
- [ ] Performance measurements included
- [ ] Commands used for verification listed

**Task Status**: [Complete/In Progress/Blocked]
```

Remember: **A task is only complete when ALL quality gates pass AND evidence is provided.**