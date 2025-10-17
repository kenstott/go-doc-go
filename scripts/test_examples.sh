#!/bin/bash
# Test configuration examples from documentation
# Validates that documented examples actually work

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
TEST_OUTPUT_DIR="$PROJECT_ROOT/tests/test_output/example_validation"
WORKER_BINARY="$PROJECT_ROOT/bin/goworker"

PASSED=0
FAILED=0
SKIPPED=0

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo "=== Configuration Example Testing ==="
echo "Project root: $PROJECT_ROOT"
echo "Worker binary: $WORKER_BINARY"
echo ""

# Ensure test output directory exists
mkdir -p "$TEST_OUTPUT_DIR"

# Check if worker binary exists
if [ ! -f "$WORKER_BINARY" ]; then
    echo -e "${RED}✗ Worker binary not found: $WORKER_BINARY${NC}"
    echo "  Build it with: cd go && go build -o ../bin/goworker ./cmd/worker"
    exit 1
fi

# Test helper function
test_config() {
    local test_name="$1"
    local config_file="$2"
    local max_docs="${3:-1}"
    local workers="${4:-1}"
    local expected_exit_code="${5:-0}"

    echo -e "${BLUE}Testing: $test_name${NC}"
    echo "  Config: $config_file"

    # Check if config file exists
    if [ ! -f "$config_file" ]; then
        echo -e "  ${RED}✗ Config file not found${NC}"
        FAILED=$((FAILED + 1))
        return 1
    fi

    # Validate TOML syntax first
    if command -v tomlv &> /dev/null; then
        if ! tomlv "$config_file" &> /dev/null; then
            echo -e "  ${RED}✗ Invalid TOML syntax${NC}"
            FAILED=$((FAILED + 1))
            return 1
        fi
    fi

    # Run worker with timeout
    local log_file="$TEST_OUTPUT_DIR/${test_name}.log"
    if timeout 30 "$WORKER_BINARY" \
        --config "$config_file" \
        --max-documents "$max_docs" \
        --workers "$workers" \
        > "$log_file" 2>&1; then

        local actual_exit=$?
        if [ "$actual_exit" -eq "$expected_exit_code" ]; then
            echo -e "  ${GREEN}✓ Passed${NC}"
            PASSED=$((PASSED + 1))
            return 0
        else
            echo -e "  ${RED}✗ Failed (exit code: $actual_exit, expected: $expected_exit_code)${NC}"
            echo "  Log: $log_file"
            FAILED=$((FAILED + 1))
            return 1
        fi
    else
        local actual_exit=$?
        if [ "$actual_exit" -eq 124 ]; then
            echo -e "  ${RED}✗ Timeout (30s exceeded)${NC}"
        else
            echo -e "  ${RED}✗ Failed (exit code: $actual_exit)${NC}"
        fi
        echo "  Log: $log_file"
        FAILED=$((FAILED + 1))
        return 1
    fi
}

# Test 1: Minimal SQLite configuration (from README.md)
echo -e "\n${YELLOW}=== Test 1: Minimal Configuration ===${NC}"
cat > "$TEST_OUTPUT_DIR/minimal_config.toml" <<'EOF'
[processing.job_control]
backend = "sqlite"
path = "$TEST_OUTPUT_DIR/minimal_queue.db"

[[content_sources]]
name = "test_documents"
type = "file"
base_path = "$TEST_OUTPUT_DIR/test_docs"
file_pattern = "*.txt"

[analytics]
enabled = true

[[analytics.outputs]]
type = "parquet"
path = "$TEST_OUTPUT_DIR/minimal_analytics.parquet"
EOF

# Substitute TEST_OUTPUT_DIR in config
sed -i.bak "s|\$TEST_OUTPUT_DIR|$TEST_OUTPUT_DIR|g" "$TEST_OUTPUT_DIR/minimal_config.toml"

# Create test document
mkdir -p "$TEST_OUTPUT_DIR/test_docs"
echo "Test document content" > "$TEST_OUTPUT_DIR/test_docs/test1.txt"

test_config "minimal_config" "$TEST_OUTPUT_DIR/minimal_config.toml" 1 1

# Test 2: Production PostgreSQL configuration (requires PostgreSQL)
echo -e "\n${YELLOW}=== Test 2: PostgreSQL Configuration ===${NC}"

# Check if PostgreSQL is available
if ! command -v psql &> /dev/null; then
    echo -e "  ${YELLOW}⊘ Skipped (PostgreSQL not installed)${NC}"
    SKIPPED=$((SKIPPED + 1))
else
    # Check if we can connect to a test database
    if psql -U postgres -d postgres -c "SELECT 1" &> /dev/null 2>&1; then
        cat > "$TEST_OUTPUT_DIR/postgres_config.toml" <<'EOF'
[processing.job_control]
backend = "postgresql"
path = "postgresql://postgres:postgres@localhost:5432/go_doc_go_test"
claim_timeout = 300
heartbeat_interval = 30

[[content_sources]]
name = "test_documents"
type = "file"
base_path = "$TEST_OUTPUT_DIR/test_docs"
file_pattern = "*.txt"

[analytics]
enabled = true

[[analytics.outputs]]
type = "parquet"
path = "$TEST_OUTPUT_DIR/postgres_analytics.parquet"
EOF
        sed -i.bak "s|\$TEST_OUTPUT_DIR|$TEST_OUTPUT_DIR|g" "$TEST_OUTPUT_DIR/postgres_config.toml"

        # Create test database if it doesn't exist
        psql -U postgres -d postgres -c "CREATE DATABASE go_doc_go_test" &> /dev/null || true

        test_config "postgres_config" "$TEST_OUTPUT_DIR/postgres_config.toml" 1 1
    else
        echo -e "  ${YELLOW}⊘ Skipped (PostgreSQL not accessible)${NC}"
        SKIPPED=$((SKIPPED + 1))
    fi
fi

# Test 3: Distributed workers configuration
echo -e "\n${YELLOW}=== Test 3: Distributed Workers ===${NC}"
cat > "$TEST_OUTPUT_DIR/distributed_config.toml" <<'EOF'
[processing.job_control]
backend = "sqlite"
path = "$TEST_OUTPUT_DIR/distributed_queue.db"

[[content_sources]]
name = "test_documents"
type = "file"
base_path = "$TEST_OUTPUT_DIR/test_docs"
file_pattern = "*.txt"

[analytics]
enabled = true

[[analytics.outputs]]
type = "parquet"
path = "$TEST_OUTPUT_DIR/distributed_analytics.parquet"
EOF

sed -i.bak "s|\$TEST_OUTPUT_DIR|$TEST_OUTPUT_DIR|g" "$TEST_OUTPUT_DIR/distributed_config.toml"

test_config "distributed_config" "$TEST_OUTPUT_DIR/distributed_config.toml" 1 4

# Test 4: Embeddings configuration (requires ONNX Runtime)
echo -e "\n${YELLOW}=== Test 4: Embeddings Configuration ===${NC}"

if [ ! -d "$PROJECT_ROOT/.venv/lib/python3.12/site-packages/onnxruntime" ]; then
    echo -e "  ${YELLOW}⊘ Skipped (ONNX Runtime not installed)${NC}"
    SKIPPED=$((SKIPPED + 1))
else
    cat > "$TEST_OUTPUT_DIR/embeddings_config.toml" <<'EOF'
[processing.job_control]
backend = "sqlite"
path = "$TEST_OUTPUT_DIR/embeddings_queue.db"

[[content_sources]]
name = "test_documents"
type = "file"
base_path = "$TEST_OUTPUT_DIR/test_docs"
file_pattern = "*.txt"

[embedding]
enabled = true
provider = "fastembed"
model = "BAAI/bge-small-en-v1.5"
dimensions = 384
batch_size = 32

[analytics]
enabled = true

[[analytics.outputs]]
type = "parquet"
path = "$TEST_OUTPUT_DIR/embeddings_analytics.parquet"
EOF

    sed -i.bak "s|\$TEST_OUTPUT_DIR|$TEST_OUTPUT_DIR|g" "$TEST_OUTPUT_DIR/embeddings_config.toml"

    test_config "embeddings_config" "$TEST_OUTPUT_DIR/embeddings_config.toml" 1 1
fi

# Test 5: Ontology extraction (requires ANTHROPIC_API_KEY)
echo -e "\n${YELLOW}=== Test 5: Ontology Extraction ===${NC}"

if [ -z "${ANTHROPIC_API_KEY:-}" ]; then
    echo -e "  ${YELLOW}⊘ Skipped (ANTHROPIC_API_KEY not set)${NC}"
    SKIPPED=$((SKIPPED + 1))
else
    cat > "$TEST_OUTPUT_DIR/ontology_schema.yaml" <<'EOF'
domains:
  - name: testing
    description: Software testing domain

entity_mappings:
  - entity: TestCase
    domain: testing
    extraction_rules:
      - type: keyword
        patterns: ["test", "testing"]
EOF

    cat > "$TEST_OUTPUT_DIR/ontology_config.toml" <<'EOF'
[processing.job_control]
backend = "sqlite"
path = "$TEST_OUTPUT_DIR/ontology_queue.db"

[[content_sources]]
name = "test_documents"
type = "file"
base_path = "$TEST_OUTPUT_DIR/test_docs"
file_pattern = "*.txt"

[ontology]
enabled = true
schema_path = "$TEST_OUTPUT_DIR/ontology_schema.yaml"
diversity_threshold = 0.85

[analytics]
enabled = true

[[analytics.outputs]]
type = "parquet"
path = "$TEST_OUTPUT_DIR/ontology_analytics.parquet"
EOF

    sed -i.bak "s|\$TEST_OUTPUT_DIR|$TEST_OUTPUT_DIR|g" "$TEST_OUTPUT_DIR/ontology_config.toml"

    test_config "ontology_config" "$TEST_OUTPUT_DIR/ontology_config.toml" 1 1
fi

# Test 6: Invalid configuration (should fail)
echo -e "\n${YELLOW}=== Test 6: Invalid Configuration (Expected Failure) ===${NC}"
cat > "$TEST_OUTPUT_DIR/invalid_config.toml" <<'EOF'
[processing.job_control]
backend = "invalid_backend"
path = "/tmp/test.db"

[[content_sources]]
name = "test"
type = "invalid_type"
EOF

# This test should fail (exit code 1)
if timeout 5 "$WORKER_BINARY" \
    --config "$TEST_OUTPUT_DIR/invalid_config.toml" \
    --max-documents 0 \
    > "$TEST_OUTPUT_DIR/invalid_config.log" 2>&1; then
    echo -e "  ${RED}✗ Failed (should have rejected invalid config)${NC}"
    FAILED=$((FAILED + 1))
else
    echo -e "  ${GREEN}✓ Passed (correctly rejected invalid config)${NC}"
    PASSED=$((PASSED + 1))
fi

# Summary
echo ""
echo "=== Test Summary ==="
echo -e "Passed:  ${GREEN}$PASSED${NC}"
echo -e "Failed:  ${RED}$FAILED${NC}"
echo -e "Skipped: ${YELLOW}$SKIPPED${NC}"
echo ""

if [ $FAILED -gt 0 ]; then
    echo -e "${RED}Some tests failed. Check logs in: $TEST_OUTPUT_DIR${NC}"
    exit 1
else
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
fi
