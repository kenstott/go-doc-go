# Go-Doc-Go Development Workflow

## Quick Start

### Option 1: Simple Containerized Development (Recommended)

```bash
# 1. Build and start the backend
docker build -f docker/Dockerfile.simple . -t go-doc-go-simple
docker run -d --name go-doc-go-dev -p 8000:8000 go-doc-go-simple

# 2. Verify it's working
curl http://localhost:8000/health
# Should return: {"status":"healthy","timestamp":"...","version":"1.0.0"}

# 3. Test the pipeline API
curl http://localhost:8000/api/pipelines
# Should return: {"pipelines":[],"total":0}
```

### Option 2: Full Development Environment (Frontend + Backend)

```bash
# Start the full development environment
docker-compose -f docker-compose.simple.yml up --build

# This will start:
# - Backend API on http://localhost:8000
# - Frontend dev server on http://localhost:3000 (if configured)
```

## Development Workflow

### 1. Initial Setup
```bash
# Clone and navigate to the project
cd /path/to/go-doc-go-ui-improvements

# Build the development container
docker build -f docker/Dockerfile.simple . -t go-doc-go-simple
```

### 2. Daily Development Loop

```bash
# Start your dev environment
docker run -d --name go-doc-go-dev -p 8000:8000 go-doc-go-simple

# Make changes to your code in:
# - src/go_doc_go/ (Python backend)
# - frontend/src/ (React frontend)

# For backend changes, rebuild and restart:
docker stop go-doc-go-dev && docker rm go-doc-go-dev
docker build -f docker/Dockerfile.simple . -t go-doc-go-simple
docker run -d --name go-doc-go-dev -p 8000:8000 go-doc-go-simple

# Test your changes
curl http://localhost:8000/api/pipelines
```

### 3. Hot Reload Development (Advanced)

For faster iteration, you can mount your source code:

```bash
docker run -d --name go-doc-go-dev \
  -p 8000:8000 \
  -v $(pwd)/src:/app/src \
  -v $(pwd)/config.yaml:/app/config/config.yaml \
  go-doc-go-simple
```

## API Usage Examples

### Managing Pipelines

```bash
# List all pipelines
curl http://localhost:8000/api/pipelines

# Create a new pipeline
curl -X POST http://localhost:8000/api/pipelines \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My Test Pipeline",
    "description": "A test pipeline for development",
    "config": {
      "source": {"type": "file", "path": "/data/input"},
      "sink": {"type": "file", "path": "/data/output"}
    }
  }'

# Get a specific pipeline
curl http://localhost:8000/api/pipelines/{pipeline_id}

# Update a pipeline
curl -X PUT http://localhost:8000/api/pipelines/{pipeline_id} \
  -H "Content-Type: application/json" \
  -d '{"name": "Updated Pipeline Name"}'

# Delete a pipeline
curl -X DELETE http://localhost:8000/api/pipelines/{pipeline_id}
```

### Pipeline Execution

```bash
# Execute a pipeline
curl -X POST http://localhost:8000/api/pipelines/{pipeline_id}/execute

# Check execution status
curl http://localhost:8000/api/pipelines/{pipeline_id}/executions

# Get specific execution details
curl http://localhost:8000/api/pipelines/{pipeline_id}/executions/{execution_id}
```

## Debugging

### View Container Logs
```bash
# See what's happening in your container
docker logs go-doc-go-dev

# Follow logs in real-time
docker logs -f go-doc-go-dev
```

### Access Container Shell
```bash
# Get a shell inside the container for debugging
docker exec -it go-doc-go-dev /bin/bash

# Check if the Flask app is running
ps aux | grep python

# Check the configuration
cat /app/config/config.yaml
```

### Health Checks
```bash
# Basic health check
curl http://localhost:8000/health

# Check if the database is working
curl http://localhost:8000/api/pipelines
```

## File Structure

Your development environment includes:

```
/app/                          # Container working directory
├── src/                       # Python source code (mounted from host)
│   └── go_doc_go/
│       ├── server.py         # Main Flask application
│       ├── api/              # API endpoints
│       └── config_db/        # Pipeline configuration database
├── config/
│   └── config.yaml           # Main configuration file
├── data/                     # SQLite database and pipeline data
└── logs/                     # Application logs
```

## Common Issues and Solutions

### Port Already in Use
```bash
# Stop any running containers
docker stop go-doc-go-dev && docker rm go-doc-go-dev

# Or use a different port
docker run -d --name go-doc-go-dev -p 8001:8000 go-doc-go-simple
```

### Import Errors
The container is configured with the correct PYTHONPATH. If you see import errors:

```bash
# Check the container environment
docker exec go-doc-go-dev env | grep PYTHON
```

### Database Issues
```bash
# Reset the pipeline database
docker exec go-doc-go-dev rm -f /app/data/pipeline_config.db

# Restart the container to recreate the database
docker restart go-doc-go-dev
```

## Next Steps

1. **Frontend Development**: The React components are ready in `frontend/src/components/Pipeline/`
2. **Pipeline Configuration**: Use the API to create and manage pipeline configurations
3. **Testing**: Run `pytest` inside the container for unit tests
4. **Production**: Use the full `docker-compose.prod.yml` for production deployment

## Available Scripts

```bash
# Quick start development environment
./scripts/docker-dev.sh start

# View logs
./scripts/docker-dev.sh logs

# Stop everything
./scripts/docker-dev.sh stop

# Clean up
./scripts/docker-dev.sh cleanup
```