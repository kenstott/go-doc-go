#!/bin/bash
# Go-Doc-Go Development Docker Management Script

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

# Docker compose file
COMPOSE_FILE="$PROJECT_DIR/docker-compose.dev.yml"

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check if Docker is running
check_docker() {
    if ! docker info &> /dev/null; then
        print_error "Docker is not running. Please start Docker and try again."
        exit 1
    fi
}

# Function to check if docker-compose file exists
check_compose_file() {
    if [[ ! -f "$COMPOSE_FILE" ]]; then
        print_error "Docker compose file not found: $COMPOSE_FILE"
        exit 1
    fi
}

# Function to load environment variables
load_env() {
    if [[ -f "$PROJECT_DIR/.env.dev" ]]; then
        print_status "Loading development environment variables from .env.dev"
        set -a
        source "$PROJECT_DIR/.env.dev"
        set +a
    else
        print_warning "No .env.dev file found. Using default values."
    fi
}

# Function to start services
start_services() {
    local profile=${1:-""}
    local service=${2:-""}
    
    print_status "Starting Go-Doc-Go development environment..."
    
    if [[ -n "$profile" ]]; then
        print_status "Using profile: $profile"
        docker-compose -f "$COMPOSE_FILE" --profile "$profile" up -d $service
    else
        docker-compose -f "$COMPOSE_FILE" up -d $service
    fi
    
    print_success "Services started successfully!"
    print_status "Frontend: http://localhost:3000"
    print_status "Backend API: http://localhost:8000"
    print_status "PostgreSQL: localhost:5432"
    print_status "Redis: localhost:6379"
}

# Function to stop services
stop_services() {
    print_status "Stopping Go-Doc-Go development services..."
    docker-compose -f "$COMPOSE_FILE" down
    print_success "Services stopped successfully!"
}

# Function to restart services
restart_services() {
    print_status "Restarting Go-Doc-Go development services..."
    docker-compose -f "$COMPOSE_FILE" restart
    print_success "Services restarted successfully!"
}

# Function to show logs
show_logs() {
    local service=${1:-""}
    local follow=${2:-false}
    
    if [[ "$follow" == "true" ]]; then
        docker-compose -f "$COMPOSE_FILE" logs -f $service
    else
        docker-compose -f "$COMPOSE_FILE" logs $service
    fi
}

# Function to show status
show_status() {
    print_status "Go-Doc-Go development environment status:"
    docker-compose -f "$COMPOSE_FILE" ps
}

# Function to rebuild services
rebuild_services() {
    local service=${1:-""}
    
    print_status "Rebuilding Go-Doc-Go services..."
    if [[ -n "$service" ]]; then
        docker-compose -f "$COMPOSE_FILE" build --no-cache $service
        docker-compose -f "$COMPOSE_FILE" up -d $service
    else
        docker-compose -f "$COMPOSE_FILE" build --no-cache
        docker-compose -f "$COMPOSE_FILE" up -d
    fi
    print_success "Services rebuilt successfully!"
}

# Function to enter container shell
enter_shell() {
    local service=${1:-"backend"}
    local shell=${2:-"/bin/bash"}
    
    print_status "Entering $service container..."
    docker-compose -f "$COMPOSE_FILE" exec $service $shell
}

# Function to clean up everything
cleanup() {
    print_warning "This will remove all containers, volumes, and images for Go-Doc-Go development environment."
    read -p "Are you sure? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        print_status "Cleaning up development environment..."
        docker-compose -f "$COMPOSE_FILE" down -v --rmi all
        docker system prune -f
        print_success "Cleanup completed!"
    else
        print_status "Cleanup cancelled."
    fi
}

# Function to initialize development environment
init_dev() {
    print_status "Initializing Go-Doc-Go development environment..."
    
    # Create .env.dev if it doesn't exist
    if [[ ! -f "$PROJECT_DIR/.env.dev" ]]; then
        print_status "Creating .env.dev file..."
        cat > "$PROJECT_DIR/.env.dev" << EOF
# Go-Doc-Go Development Environment Variables

# API Keys (optional - set these if you want to test with real APIs)
OPENAI_API_KEY=
ANTHROPIC_API_KEY=
CLAUDE_API_KEY=

# Database Configuration
POSTGRES_DB=go_doc_go_dev
POSTGRES_USER=go_doc_go
POSTGRES_PASSWORD=go_doc_go_dev

# Worker Configuration
WORKER_REPLICAS=2

# Logging
LOG_LEVEL=DEBUG

# Development URLs
FRONTEND_API_URL=http://localhost:8000
CORS_ORIGINS=http://localhost:3000,http://localhost:5173
EOF
        print_success ".env.dev created. Please edit it to add your API keys."
    fi
    
    # Create data directories
    mkdir -p "$PROJECT_DIR"/{data,logs,cache}
    
    # Build and start basic services
    print_status "Building and starting core services..."
    docker-compose -f "$COMPOSE_FILE" build
    docker-compose -f "$COMPOSE_FILE" up -d postgres redis backend
    
    # Wait for services to be ready
    print_status "Waiting for services to be ready..."
    sleep 10
    
    # Start frontend
    docker-compose -f "$COMPOSE_FILE" up -d frontend
    
    # Start workers
    docker-compose -f "$COMPOSE_FILE" up -d worker
    
    print_success "Development environment initialized!"
    print_status "You can now access:"
    print_status "  - Frontend: http://localhost:3000"
    print_status "  - Backend API: http://localhost:8000"
    print_status "  - API Docs: http://localhost:8000/docs"
}

# Function to show help
show_help() {
    cat << EOF
Go-Doc-Go Development Docker Management Script

Usage: $0 [COMMAND] [OPTIONS]

Commands:
    init                Initialize development environment
    start [PROFILE]     Start services (optionally with profile: full, search, graph, etc.)
    stop                Stop all services
    restart             Restart all services
    status              Show service status
    logs [SERVICE]      Show logs (add -f for follow mode)
    logs -f [SERVICE]   Follow logs for service
    rebuild [SERVICE]   Rebuild and restart service(s)
    shell [SERVICE]     Enter service container shell (default: backend)
    cleanup             Remove all containers, volumes, and images
    
Profiles:
    full                Start with all optional services
    search              Start with Elasticsearch and Solr
    graph               Start with Neo4j
    nosql               Start with MongoDB
    s3                  Start with MinIO
    tools               Start with development tools container

Examples:
    $0 init                     # Initialize development environment
    $0 start                    # Start core services
    $0 start full               # Start all services
    $0 logs backend             # Show backend logs
    $0 logs -f worker           # Follow worker logs
    $0 shell backend            # Enter backend container
    $0 rebuild frontend         # Rebuild frontend service
    $0 cleanup                  # Clean everything up

Environment:
    Create .env.dev file in project root to set environment variables.
    
EOF
}

# Main script logic
main() {
    cd "$PROJECT_DIR"
    
    check_docker
    check_compose_file
    load_env
    
    case "${1:-help}" in
        init)
            init_dev
            ;;
        start)
            start_services "$2" "$3"
            ;;
        stop)
            stop_services
            ;;
        restart)
            restart_services
            ;;
        status)
            show_status
            ;;
        logs)
            if [[ "$2" == "-f" ]]; then
                show_logs "$3" true
            else
                show_logs "$2" false
            fi
            ;;
        rebuild)
            rebuild_services "$2"
            ;;
        shell)
            enter_shell "$2" "$3"
            ;;
        cleanup)
            cleanup
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            print_error "Unknown command: $1"
            show_help
            exit 1
            ;;
    esac
}

# Run main function
main "$@"