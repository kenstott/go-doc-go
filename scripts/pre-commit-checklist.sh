#!/bin/bash
# pre-commit-checklist.sh - run before any git commit
# Based on Java project guidelines adapted for Python Go-Doc-Go project

echo "Pre-Commit Checklist - MANDATORY"
echo "=================================="

# Get project root directory
PROJECT_ROOT=$(git rev-parse --show-toplevel)
cd "$PROJECT_ROOT"

# Exit on any error
set -e

# 1. All modified code builds without errors
echo "Checking compilation..."
find src/go_doc_go -name "*.py" -exec python -m py_compile {} \; || {
    echo "✗ Compilation errors found"
    exit 1
}
echo "✓ Code compiles without errors"

# 2. All related tests pass (provide command + output)
echo "Running tests..."
pytest -v || {
    echo "✗ Tests failed"
    exit 1
}
echo "✓ All tests pass"

# 3. No debugging artifacts left in code
echo "Checking for debug artifacts..."
if grep -r "print(" src/go_doc_go/ --include="*.py" | grep -v "__main__" | grep -v "logger\." | head -5; then
    echo "✗ Debug print statements found - remove before commit"
    exit 1
fi
echo "✓ No debugging artifacts"

# 4. Code quality checks
echo "Running linting..."
flake8 src/ tests/ || {
    echo "✗ Linting errors found"
    exit 1
}
echo "✓ Linting passed"

echo "Running type checking..."
mypy src/ || {
    echo "✗ Type checking errors found"
    exit 1
}
echo "✓ Type checking passed"

echo "Checking code formatting..."
black --check src/ tests/ || {
    echo "✗ Code formatting required - run 'black src/ tests/'"
    exit 1
}
echo "✓ Code formatting verified"

# 5. Coverage requirements met
echo "Checking coverage..."
pytest --cov=src/go_doc_go --cov-report=term --cov-fail-under=70 -q || {
    echo "✗ Coverage below 70%"
    exit 1
}
echo "✓ Coverage requirements met"

# 6. Performance benchmarks (if performance-critical code changed)
CHANGED_FILES=$(git diff --cached --name-only || git diff --name-only HEAD~1)
if echo "$CHANGED_FILES" | grep -E "(queue|parser)" > /dev/null; then
    echo "Performance-critical files changed, running performance tests..."
    pytest -m performance --tb=short || {
        echo "✗ Performance tests failed"
        exit 1
    }
    echo "✓ Performance benchmarks met"
else
    echo "✓ No performance-critical files changed"
fi

# 7. Check for proper stub marking (if any incomplete features)
echo "Checking for proper stub marking..."
STUB_METHODS=$(grep -r "raise NotImplementedError\|TODO:" src/go_doc_go/ --include="*.py" | wc -l)
if [ "$STUB_METHODS" -gt 0 ]; then
    echo "⚠ Found $STUB_METHODS stub methods/TODOs - ensure they are properly documented"
    grep -r "raise NotImplementedError\|TODO:" src/go_doc_go/ --include="*.py" | head -5
fi

echo ""
echo "🎉 All checks passed - ready to commit!"
echo ""
echo "Remember to use appropriate commit prefixes:"
echo "  feat: for working features"
echo "  wip: for work-in-progress with stubs"  
echo "  stub: for adding structure without implementation"
echo "  fix: for bug fixes"
echo "  test: for test-only changes"