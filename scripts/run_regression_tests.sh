#!/bin/bash
# run_regression_tests.sh - Comprehensive regression test suite
# Based on Java project patterns adapted for Go-Doc-Go

echo "🧪 Go-Doc-Go Regression Test Suite"
echo "=================================="

# Get project root and setup
PROJECT_ROOT=$(git rev-parse --show-toplevel)
cd "$PROJECT_ROOT"

# Test execution configuration
TIMEOUT_UNIT=300        # 5 minutes for unit tests
TIMEOUT_INTEGRATION=600 # 10 minutes for integration tests  
TIMEOUT_PERFORMANCE=900 # 15 minutes for performance tests

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Track results
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0
FAILED_SUITES=()

echo_status() {
    local status=$1
    local message=$2
    case $status in
        "PASS") echo -e "${GREEN}✓ PASS${NC}: $message" ;;
        "FAIL") echo -e "${RED}✗ FAIL${NC}: $message" ;;
        "WARN") echo -e "${YELLOW}⚠ WARN${NC}: $message" ;;
        "INFO") echo -e "${BLUE}ℹ INFO${NC}: $message" ;;
    esac
}

run_test_suite() {
    local suite_name=$1
    local test_command=$2
    local timeout=$3
    
    echo ""
    echo_status "INFO" "Running $suite_name..."
    echo "Command: $test_command"
    
    if timeout $timeout bash -c "$test_command"; then
        echo_status "PASS" "$suite_name completed successfully"
        ((PASSED_TESTS++))
    else
        echo_status "FAIL" "$suite_name failed"
        FAILED_SUITES+=("$suite_name")
        ((FAILED_TESTS++))
    fi
    ((TOTAL_TESTS++))
}

# Ensure clean environment
echo_status "INFO" "Setting up clean test environment..."

# Create test database if needed
if command -v docker &> /dev/null && [ -f "tests/test_queue/docker-compose.yml" ]; then
    echo_status "INFO" "Starting PostgreSQL test container..."
    cd tests/test_queue
    docker-compose down > /dev/null 2>&1
    docker-compose up -d > /dev/null 2>&1
    cd "$PROJECT_ROOT"
    sleep 5  # Wait for PostgreSQL to be ready
else
    echo_status "WARN" "Docker not available - PostgreSQL tests may fail"
fi

echo ""
echo "🏃 Starting Regression Test Execution"
echo "====================================="

# 1. Unit Tests - Core Components
run_test_suite \
    "Unit Tests - Document Parsers" \
    "pytest -m unit tests/test_parsers/ -v --tb=short" \
    $TIMEOUT_UNIT

run_test_suite \
    "Unit Tests - Storage Layer" \
    "pytest -m unit tests/test_storage/ -v --tb=short" \
    $TIMEOUT_UNIT

run_test_suite \
    "Unit Tests - Work Queue System" \
    "pytest -m unit tests/test_queue/ -v --tb=short" \
    $TIMEOUT_UNIT

run_test_suite \
    "Unit Tests - Embedding System" \
    "pytest -m unit tests/test_embeddings/ -v --tb=short" \
    $TIMEOUT_UNIT

# 2. Integration Tests - Component Interactions  
run_test_suite \
    "Integration Tests - Parser Factory" \
    "pytest -m integration tests/test_parsers/test_factory.py -v --tb=short" \
    $TIMEOUT_INTEGRATION

run_test_suite \
    "Integration Tests - Document Processing Pipeline" \
    "pytest -m integration tests/test_integration/ -v --tb=short" \
    $TIMEOUT_INTEGRATION

# 3. PostgreSQL-Dependent Tests
if command -v docker &> /dev/null; then
    run_test_suite \
        "PostgreSQL Tests - Work Queue Atomic Operations" \
        "TEST_PG_HOST=localhost TEST_PG_PORT=15432 TEST_PG_DB=go_doc_go_test TEST_PG_USER=testuser TEST_PG_PASSWORD=testpass pytest -m requires_postgres tests/test_queue/ -v --tb=short" \
        $TIMEOUT_INTEGRATION

    run_test_suite \
        "PostgreSQL Tests - Storage Operations" \
        "TEST_PG_HOST=localhost TEST_PG_PORT=15432 TEST_PG_DB=go_doc_go_test TEST_PG_USER=testuser TEST_PG_PASSWORD=testpass pytest -m requires_postgres tests/test_storage/ -v --tb=short" \
        $TIMEOUT_INTEGRATION
fi

# 4. Performance Tests - SLA Verification
run_test_suite \
    "Performance Tests - Document Parser SLAs" \
    "pytest -m performance tests/test_parsers/ --tb=short" \
    $TIMEOUT_PERFORMANCE

run_test_suite \
    "Performance Tests - Work Queue Throughput SLAs" \
    "TEST_PG_HOST=localhost TEST_PG_PORT=15432 TEST_PG_DB=go_doc_go_test TEST_PG_USER=testuser TEST_PG_PASSWORD=testpass pytest -m performance tests/test_queue/ --tb=short" \
    $TIMEOUT_PERFORMANCE

# 5. Cross-System Integration Tests
run_test_suite \
    "End-to-End Tests - Full Document Processing" \
    "pytest tests/test_e2e/ -v --tb=short --timeout=1800" \
    1800  # 30 minutes for E2E

# 6. Content Source Tests (if available)
if [ -d "tests/test_content_sources" ]; then
    run_test_suite \
        "Integration Tests - Content Sources" \
        "pytest tests/test_content_sources/ -v --tb=short" \
        $TIMEOUT_INTEGRATION
fi

# 7. Adapter Tests (if available) 
if [ -d "tests/test_adapters" ]; then
    run_test_suite \
        "Integration Tests - Storage Adapters" \
        "pytest tests/test_adapters/ -v --tb=short" \
        $TIMEOUT_INTEGRATION
fi

# Cleanup
if command -v docker &> /dev/null && [ -f "tests/test_queue/docker-compose.yml" ]; then
    echo_status "INFO" "Cleaning up test containers..."
    cd tests/test_queue
    docker-compose down > /dev/null 2>&1
    cd "$PROJECT_ROOT"
fi

# Results Summary
echo ""
echo "📊 REGRESSION TEST RESULTS"
echo "=========================="
echo_status "INFO" "Total Test Suites: $TOTAL_TESTS"
echo_status "INFO" "Passed: $PASSED_TESTS"
echo_status "INFO" "Failed: $FAILED_TESTS"

if [ $FAILED_TESTS -eq 0 ]; then
    echo ""
    echo_status "PASS" "🎉 ALL REGRESSION TESTS PASSED!"
    echo_status "INFO" "System is ready for deployment"
    exit 0
else
    echo ""
    echo_status "FAIL" "❌ REGRESSION TEST FAILURES DETECTED"
    echo_status "FAIL" "Failed test suites:"
    for suite in "${FAILED_SUITES[@]}"; do
        echo "  - $suite"
    done
    echo ""
    echo_status "FAIL" "System is NOT ready for deployment"
    echo_status "INFO" "Fix all failures before proceeding"
    exit 1
fi