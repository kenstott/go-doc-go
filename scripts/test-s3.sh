#!/bin/bash
# Script to run S3 adapter and content source tests with Minio

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}=== S3 Testing Script ===${NC}"
echo ""

# Check if Docker is available
if ! command -v docker &> /dev/null; then
    echo -e "${RED}Error: Docker is not installed or not in PATH${NC}"
    exit 1
fi

# Check if docker-compose is available
if ! command -v docker-compose &> /dev/null; then
    echo -e "${YELLOW}Warning: docker-compose not found, trying 'docker compose'${NC}"
    DOCKER_COMPOSE="docker compose"
else
    DOCKER_COMPOSE="docker-compose"
fi

# Parse command line arguments
SKIP_DOCKER=false
COVERAGE=false
VERBOSE=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --skip-docker)
            SKIP_DOCKER=true
            shift
            ;;
        --coverage)
            COVERAGE=true
            shift
            ;;
        --verbose|-v)
            VERBOSE=true
            shift
            ;;
        --help|-h)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --skip-docker    Skip Docker setup (use existing Minio)"
            echo "  --coverage       Run tests with coverage reporting"
            echo "  --verbose, -v    Run tests in verbose mode"
            echo "  --help, -h       Show this help message"
            exit 0
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            exit 1
            ;;
    esac
done

# Change to project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_ROOT"

# Start Minio if not skipping Docker
if [ "$SKIP_DOCKER" = false ]; then
    echo -e "${YELLOW}Starting Minio container...${NC}"
    $DOCKER_COMPOSE -f docker-compose.test.yml up -d
    
    # Wait for Minio to be ready
    echo -e "${YELLOW}Waiting for Minio to be ready...${NC}"
    MAX_ATTEMPTS=30
    ATTEMPT=0
    
    while [ $ATTEMPT -lt $MAX_ATTEMPTS ]; do
        if docker exec doculyzer-test-minio mc alias set minio http://localhost:9000 minioadmin minioadmin &> /dev/null; then
            echo -e "${GREEN}Minio is ready!${NC}"
            break
        fi
        ATTEMPT=$((ATTEMPT + 1))
        echo -n "."
        sleep 2
    done
    
    if [ $ATTEMPT -eq $MAX_ATTEMPTS ]; then
        echo -e "${RED}Error: Minio failed to start${NC}"
        $DOCKER_COMPOSE -f docker-compose.test.yml logs minio
        $DOCKER_COMPOSE -f docker-compose.test.yml down
        exit 1
    fi
fi

# Install test dependencies
echo -e "${YELLOW}Installing test dependencies...${NC}"
pip install -e ".[development,cloud-aws]" -q

# Build pytest command
PYTEST_CMD="python -m pytest"
PYTEST_ARGS=""

if [ "$COVERAGE" = true ]; then
    PYTEST_ARGS="$PYTEST_ARGS --cov=src/go_doc_go/adapter/s3 --cov=src/go_doc_go/content_source/s3"
    PYTEST_ARGS="$PYTEST_ARGS --cov-report=term-missing --cov-report=html"
fi

if [ "$VERBOSE" = true ]; then
    PYTEST_ARGS="$PYTEST_ARGS -v"
fi

# Run tests
echo ""
echo -e "${GREEN}Running S3 adapter tests...${NC}"
$PYTEST_CMD tests/test_adapters/test_s3_adapter.py $PYTEST_ARGS

echo ""
echo -e "${GREEN}Running S3 content source tests...${NC}"
$PYTEST_CMD tests/test_content_sources/test_s3_content_source.py $PYTEST_ARGS

echo ""
echo -e "${GREEN}Running S3 integration tests...${NC}"
$PYTEST_CMD tests/test_integration/test_s3_integration.py $PYTEST_ARGS

# Cleanup
if [ "$SKIP_DOCKER" = false ]; then
    echo ""
    echo -e "${YELLOW}Stopping Minio container...${NC}"
    $DOCKER_COMPOSE -f docker-compose.test.yml down -v
fi

echo ""
echo -e "${GREEN}=== S3 Tests Complete ===${NC}"

if [ "$COVERAGE" = true ]; then
    echo -e "${YELLOW}Coverage report saved to htmlcov/index.html${NC}"
fi