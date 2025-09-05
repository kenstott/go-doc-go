#!/bin/bash
# Script to run MongoDB adapter and content source tests

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}=== MongoDB Testing Script ===${NC}"
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
INTEGRATION_ONLY=false

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
        --integration)
            INTEGRATION_ONLY=true
            shift
            ;;
        --help|-h)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --skip-docker     Skip Docker setup (use existing MongoDB)"
            echo "  --coverage        Run tests with coverage reporting"
            echo "  --verbose, -v     Run tests in verbose mode"
            echo "  --integration     Run only integration tests"
            echo "  --help, -h        Show this help message"
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

# Start MongoDB if not skipping Docker
if [ "$SKIP_DOCKER" = false ]; then
    echo -e "${YELLOW}Starting MongoDB container...${NC}"
    $DOCKER_COMPOSE -f docker-compose.test.yml up -d mongodb
    
    # Wait for MongoDB to be ready
    echo -e "${YELLOW}Waiting for MongoDB to be ready...${NC}"
    MAX_ATTEMPTS=30
    ATTEMPT=0
    
    while [ $ATTEMPT -lt $MAX_ATTEMPTS ]; do
        if docker exec doculyzer-test-mongodb mongosh --quiet --eval "db.adminCommand('ping')" &> /dev/null; then
            echo -e "${GREEN}MongoDB is ready!${NC}"
            
            # Run initialization
            echo -e "${YELLOW}Initializing test data...${NC}"
            $DOCKER_COMPOSE -f docker-compose.test.yml up -d mongodb-init
            sleep 3
            break
        fi
        ATTEMPT=$((ATTEMPT + 1))
        echo -n "."
        sleep 2
    done
    
    if [ $ATTEMPT -eq $MAX_ATTEMPTS ]; then
        echo -e "${RED}Error: MongoDB failed to start${NC}"
        $DOCKER_COMPOSE -f docker-compose.test.yml logs mongodb
        $DOCKER_COMPOSE -f docker-compose.test.yml down
        exit 1
    fi
fi

# Install test dependencies
echo -e "${YELLOW}Installing test dependencies...${NC}"
pip install -e ".[development,db-mongodb,source-mongodb]" -q

# Build pytest command
PYTEST_CMD="python -m pytest"
PYTEST_ARGS=""

if [ "$COVERAGE" = true ]; then
    PYTEST_ARGS="$PYTEST_ARGS --cov=src/go_doc_go/adapter/mongodb --cov=src/go_doc_go/content_source/mongodb"
    PYTEST_ARGS="$PYTEST_ARGS --cov-report=term-missing --cov-report=html"
fi

if [ "$VERBOSE" = true ]; then
    PYTEST_ARGS="$PYTEST_ARGS -v"
fi

# Run tests
if [ "$INTEGRATION_ONLY" = true ]; then
    echo ""
    echo -e "${GREEN}Running MongoDB integration tests only...${NC}"
    $PYTEST_CMD tests/test_integration/test_mongodb_integration.py $PYTEST_ARGS
else
    echo ""
    echo -e "${GREEN}Running MongoDB adapter tests...${NC}"
    $PYTEST_CMD tests/test_adapters/test_mongodb_adapter.py $PYTEST_ARGS

    echo ""
    echo -e "${GREEN}Running MongoDB content source tests...${NC}"
    $PYTEST_CMD tests/test_content_sources/test_mongodb_content_source.py $PYTEST_ARGS

    echo ""
    echo -e "${GREEN}Running MongoDB integration tests...${NC}"
    $PYTEST_CMD tests/test_integration/test_mongodb_integration.py $PYTEST_ARGS
fi

# Cleanup
if [ "$SKIP_DOCKER" = false ]; then
    echo ""
    echo -e "${YELLOW}Stopping MongoDB container...${NC}"
    $DOCKER_COMPOSE -f docker-compose.test.yml stop mongodb mongodb-init
    $DOCKER_COMPOSE -f docker-compose.test.yml rm -f mongodb-init
fi

echo ""
echo -e "${GREEN}=== MongoDB Tests Complete ===${NC}"

if [ "$COVERAGE" = true ]; then
    echo -e "${YELLOW}Coverage report saved to htmlcov/index.html${NC}"
fi