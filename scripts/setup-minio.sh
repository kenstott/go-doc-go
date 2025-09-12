#!/bin/bash
# Script to set up Minio for local development

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== Minio Development Setup ===${NC}"
echo ""

# Parse command line arguments
ACTION="start"
DETACHED=true

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
        --attach|-a)
            DETACHED=false
            shift
            ;;
        --help|-h)
            echo "Usage: $0 [ACTION] [OPTIONS]"
            echo ""
            echo "Actions:"
            echo "  start         Start Minio container (default)"
            echo "  stop          Stop Minio container"
            echo "  status        Show Minio container status"
            echo "  logs          Show Minio logs"
            echo ""
            echo "Options:"
            echo "  --attach, -a  Run in attached mode (show logs)"
            echo "  --help, -h    Show this help message"
            echo ""
            echo "Examples:"
            echo "  $0                  # Start Minio in background"
            echo "  $0 start --attach   # Start Minio and show logs"
            echo "  $0 stop             # Stop Minio"
            echo "  $0 status           # Check if Minio is running"
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
        echo -e "${YELLOW}Starting Minio...${NC}"
        
        if [ "$DETACHED" = true ]; then
            $DOCKER_COMPOSE -f docker-compose.test.yml up -d minio
        else
            $DOCKER_COMPOSE -f docker-compose.test.yml up minio
        fi
        
        if [ "$DETACHED" = true ]; then
            # Wait for Minio to be ready
            echo -e "${YELLOW}Waiting for Minio to be ready...${NC}"
            MAX_ATTEMPTS=30
            ATTEMPT=0
            
            while [ $ATTEMPT -lt $MAX_ATTEMPTS ]; do
                if curl -s http://localhost:9000/minio/health/live > /dev/null; then
                    echo -e "${GREEN}✓ Minio is ready!${NC}"
                    echo ""
                    echo -e "${BLUE}Minio Web Console:${NC} http://localhost:9001"
                    echo -e "${BLUE}S3 Endpoint:${NC} http://localhost:9000"
                    echo -e "${BLUE}Access Key:${NC} minioadmin"
                    echo -e "${BLUE}Secret Key:${NC} minioadmin"
                    echo ""
                    
                    # Initialize buckets
                    echo -e "${YELLOW}Initializing test buckets...${NC}"
                    $DOCKER_COMPOSE -f docker-compose.test.yml up -d minio-init
                    sleep 3
                    echo -e "${GREEN}✓ Buckets initialized${NC}"
                    break
                fi
                ATTEMPT=$((ATTEMPT + 1))
                echo -n "."
                sleep 2
            done
            
            if [ $ATTEMPT -eq $MAX_ATTEMPTS ]; then
                echo -e "${RED}Error: Minio failed to start${NC}"
                $DOCKER_COMPOSE -f docker-compose.test.yml logs minio
                exit 1
            fi
        fi
        ;;
        
    stop)
        echo -e "${YELLOW}Stopping Minio...${NC}"
        $DOCKER_COMPOSE -f docker-compose.test.yml down -v
        echo -e "${GREEN}✓ Minio stopped${NC}"
        ;;
        
    status)
        if docker ps | grep -q go-doc-go-test-minio; then
            echo -e "${GREEN}✓ Minio is running${NC}"
            echo ""
            docker ps --filter name=go-doc-go-test-minio --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
            echo ""
            echo -e "${BLUE}Web Console:${NC} http://localhost:9001"
            echo -e "${BLUE}S3 Endpoint:${NC} http://localhost:9000"
        else
            echo -e "${YELLOW}⚠ Minio is not running${NC}"
            echo ""
            echo "Run '$0 start' to start Minio"
        fi
        ;;
        
    logs)
        echo -e "${YELLOW}Showing Minio logs...${NC}"
        $DOCKER_COMPOSE -f docker-compose.test.yml logs -f minio
        ;;
        
    *)
        echo -e "${RED}Unknown action: $ACTION${NC}"
        exit 1
        ;;
esac