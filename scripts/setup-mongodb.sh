#!/bin/bash
# Script to set up MongoDB for local development

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== MongoDB Development Setup ===${NC}"
echo ""

# Parse command line arguments
ACTION="start"
DETACHED=true
INIT_DATA=false

while [[ $# -gt 0 ]]; do
    case $1 in
        start)
            ACTION="start"
            shift
            ;;
        stop)
            ACTION="stop"
            shift
            ;;
        status)
            ACTION="status"
            shift
            ;;
        logs)
            ACTION="logs"
            shift
            ;;
        shell)
            ACTION="shell"
            shift
            ;;
        --attach|-a)
            DETACHED=false
            shift
            ;;
        --init-data)
            INIT_DATA=true
            shift
            ;;
        --help|-h)
            echo "Usage: $0 [ACTION] [OPTIONS]"
            echo ""
            echo "Actions:"
            echo "  start         Start MongoDB container (default)"
            echo "  stop          Stop MongoDB container"
            echo "  status        Show MongoDB container status"
            echo "  logs          Show MongoDB logs"
            echo "  shell         Open MongoDB shell"
            echo ""
            echo "Options:"
            echo "  --attach, -a  Run in attached mode (show logs)"
            echo "  --init-data   Initialize with sample data"
            echo "  --help, -h    Show this help message"
            echo ""
            echo "Examples:"
            echo "  $0                    # Start MongoDB in background"
            echo "  $0 start --attach     # Start MongoDB and show logs"
            echo "  $0 start --init-data  # Start with sample data"
            echo "  $0 shell              # Open MongoDB shell"
            echo "  $0 stop               # Stop MongoDB"
            exit 0
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            exit 1
            ;;
    esac
done

# Check if Docker is available
if ! command -v docker &> /dev/null; then
    echo -e "${RED}Error: Docker is not installed or not in PATH${NC}"
    exit 1
fi

# Check if docker-compose is available
if ! command -v docker-compose &> /dev/null; then
    DOCKER_COMPOSE="docker compose"
else
    DOCKER_COMPOSE="docker-compose"
fi

# Change to project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_ROOT"

case $ACTION in
    start)
        echo -e "${YELLOW}Starting MongoDB...${NC}"
        
        if [ "$DETACHED" = true ]; then
            $DOCKER_COMPOSE -f docker-compose.test.yml up -d mongodb
        else
            $DOCKER_COMPOSE -f docker-compose.test.yml up mongodb
        fi
        
        if [ "$DETACHED" = true ]; then
            # Wait for MongoDB to be ready
            echo -e "${YELLOW}Waiting for MongoDB to be ready...${NC}"
            MAX_ATTEMPTS=30
            ATTEMPT=0
            
            while [ $ATTEMPT -lt $MAX_ATTEMPTS ]; do
                if docker exec go-doc-go-test-mongodb mongosh --quiet --eval "db.adminCommand('ping')" &> /dev/null; then
                    echo -e "${GREEN}✓ MongoDB is ready!${NC}"
                    echo ""
                    echo -e "${BLUE}MongoDB Connection Info:${NC}"
                    echo -e "${BLUE}Host:${NC} localhost"
                    echo -e "${BLUE}Port:${NC} 27017"
                    echo -e "${BLUE}Username:${NC} admin"
                    echo -e "${BLUE}Password:${NC} admin123"
                    echo -e "${BLUE}Database:${NC} test_db"
                    echo -e "${BLUE}Connection String:${NC} mongodb://admin:admin123@localhost:27017/"
                    echo ""
                    
                    if [ "$INIT_DATA" = true ]; then
                        echo -e "${YELLOW}Initializing test data...${NC}"
                        $DOCKER_COMPOSE -f docker-compose.test.yml up -d mongodb-init
                        sleep 3
                        
                        # Import sample documents if they exist
                        if [ -f "tests/test_data/mongodb/sample_documents.json" ]; then
                            echo -e "${YELLOW}Importing sample documents...${NC}"
                            docker exec -i go-doc-go-test-mongodb mongoimport \
                                --host localhost \
                                --username admin \
                                --password admin123 \
                                --authenticationDatabase admin \
                                --db test_db \
                                --collection documents \
                                --file /docker-entrypoint-initdb.d/sample_documents.json \
                                --jsonArray 2>/dev/null || true
                        fi
                        echo -e "${GREEN}✓ Test data initialized${NC}"
                    fi
                    break
                fi
                ATTEMPT=$((ATTEMPT + 1))
                echo -n "."
                sleep 2
            done
            
            if [ $ATTEMPT -eq $MAX_ATTEMPTS ]; then
                echo -e "${RED}Error: MongoDB failed to start${NC}"
                $DOCKER_COMPOSE -f docker-compose.test.yml logs mongodb
                exit 1
            fi
        fi
        ;;
        
    stop)
        echo -e "${YELLOW}Stopping MongoDB...${NC}"
        $DOCKER_COMPOSE -f docker-compose.test.yml stop mongodb mongodb-init
        $DOCKER_COMPOSE -f docker-compose.test.yml rm -f mongodb-init
        echo -e "${GREEN}✓ MongoDB stopped${NC}"
        ;;
        
    status)
        if docker ps | grep -q go-doc-go-test-mongodb; then
            echo -e "${GREEN}✓ MongoDB is running${NC}"
            echo ""
            docker ps --filter name=go-doc-go-test-mongodb --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
            echo ""
            echo -e "${BLUE}Connection String:${NC} mongodb://admin:admin123@localhost:27017/"
            
            # Show database stats
            echo ""
            echo -e "${BLUE}Database Statistics:${NC}"
            docker exec go-doc-go-test-mongodb mongosh --quiet \
                -u admin -p admin123 --authenticationDatabase admin \
                --eval "use test_db; db.stats()" 2>/dev/null || echo "Unable to get stats"
        else
            echo -e "${YELLOW}⚠ MongoDB is not running${NC}"
            echo ""
            echo "Run '$0 start' to start MongoDB"
        fi
        ;;
        
    logs)
        echo -e "${YELLOW}Showing MongoDB logs...${NC}"
        $DOCKER_COMPOSE -f docker-compose.test.yml logs -f mongodb
        ;;
        
    shell)
        if docker ps | grep -q go-doc-go-test-mongodb; then
            echo -e "${YELLOW}Opening MongoDB shell...${NC}"
            echo -e "${BLUE}Connecting as admin to test_db${NC}"
            echo ""
            docker exec -it go-doc-go-test-mongodb mongosh \
                -u admin -p admin123 --authenticationDatabase admin test_db
        else
            echo -e "${RED}Error: MongoDB is not running${NC}"
            echo "Run '$0 start' first"
            exit 1
        fi
        ;;
        
    *)
        echo -e "${RED}Unknown action: $ACTION${NC}"
        exit 1
        ;;
esac