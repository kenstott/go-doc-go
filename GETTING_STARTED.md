# Getting Started with Go-Doc-Go

## 🚀 Quick Start (30 seconds)

```bash
# 1. Start everything with hot reload
./dev-start.sh

# That's it! Everything is running with hot reload:
# 🌐 Frontend: http://localhost:3000  
# 🔧 Backend:  http://localhost:8000
```

## What You Just Got

### ✅ Hot-Reload Development Environment
- **Edit Python code** in `src/go_doc_go/` → Backend reloads instantly
- **Edit React code** in `frontend/src/` → Frontend reloads instantly  
- **All your data persists** between restarts

### ✅ Full-Stack Application
- **Pipeline Manager UI** at http://localhost:3000
- **REST API** at http://localhost:8000/api/pipelines
- **Health monitoring** at http://localhost:8000/health

## First Steps

### 1. Check Everything is Working
```bash
# Health check
curl http://localhost:8000/health

# Should return: {"status":"healthy",...}
```

### 2. Create Your First Pipeline
```bash
curl -X POST http://localhost:8000/api/pipelines \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My First Pipeline", 
    "description": "Learning the ropes",
    "config": {"source": {"type": "file"}}
  }'
```

### 3. View in the UI
Open http://localhost:3000 to see your pipeline in the web interface.

### 4. Make Your First Code Change
Edit `src/go_doc_go/server.py` or `frontend/src/components/Pipeline/PipelineManager.tsx` and watch it reload instantly!

## Development Workflow

### Daily Usage
```bash
# Start development (only needed once)
./dev-start.sh

# Edit your code - it auto-reloads!
# Frontend: frontend/src/
# Backend:  src/go_doc_go/

# Stop everything
# Press Ctrl+C or run:
docker-compose -f docker-compose.dev.simple.yml down
```

### File Structure
```
📁 Project Root
├── 🐍 src/go_doc_go/          # Python backend code (auto-reloads)
│   ├── server.py              # Main Flask app
│   ├── api/pipeline_routes.py # Pipeline API endpoints  
│   └── config_db/             # Database models
├── ⚛️  frontend/src/           # React frontend code (auto-reloads)
│   └── components/Pipeline/   # Pipeline management UI
├── 🚀 dev-start.sh           # One-command startup
└── 📖 GETTING_STARTED.md     # This file
```

## API Quick Reference

```bash
# List pipelines
curl http://localhost:8000/api/pipelines

# Create pipeline
curl -X POST http://localhost:8000/api/pipelines \
  -H "Content-Type: application/json" \
  -d '{"name":"Test","config":{}}'

# Execute pipeline  
curl -X POST http://localhost:8000/api/pipelines/{id}/execute

# View execution status
curl http://localhost:8000/api/pipelines/{id}/executions
```

## Troubleshooting

### Port Already in Use?
```bash
# Stop everything and try again
docker-compose -f docker-compose.dev.simple.yml down
./dev-start.sh
```

### Need to Reset Database?
```bash
# Remove all data and restart
docker-compose -f docker-compose.dev.simple.yml down -v
./dev-start.sh
```

### Want to See Logs?
```bash
# In another terminal
docker-compose -f docker-compose.dev.simple.yml logs -f backend
docker-compose -f docker-compose.dev.simple.yml logs -f frontend
```

## What's Next?

1. **Explore the Pipeline Manager UI** at http://localhost:3000
2. **Create some pipelines** using the API or UI
3. **Edit the React components** in `frontend/src/components/Pipeline/`
4. **Add new API endpoints** in `src/go_doc_go/api/`
5. **Check out the full documentation** in `DEVELOPMENT.md`

## Need Help?

- 📖 Full docs: `DEVELOPMENT.md`
- 🔧 API details: `src/go_doc_go/api/pipeline_routes.py`
- 🌐 UI components: `frontend/src/components/Pipeline/`

**Happy coding! 🎉**