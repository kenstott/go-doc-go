#!/bin/bash
# One-command development environment with hot reloading

echo "🚀 Starting Go-Doc-Go Development Environment"
echo "============================================="

# Check available backends
echo ""
echo "📊 Checking available storage backends..."
python3 scripts/check_backends.py 2>/dev/null || {
    echo "⚠️  Backend check script not available or missing dependencies"
    echo "   Run: pip install rich pyyaml"
}

# Stop any existing containers
echo ""
echo "🧹 Cleaning up existing containers..."
docker-compose -f docker-compose.dev.simple.yml down

# Start everything with hot reloading
echo ""
echo "🔥 Starting with hot reload..."
docker-compose -f docker-compose.dev.simple.yml up --build

echo "✅ Development environment started!"
echo ""
echo "🌐 Frontend: http://localhost:3000"
echo "🔧 Backend API: http://localhost:8000"
echo "❤️  Health Check: http://localhost:8000/health"
echo "📊 Pipeline API: http://localhost:8000/api/pipelines"
echo ""
echo "📁 Edit files in:"
echo "   - src/go_doc_go/ (Python backend - auto-reloads)"
echo "   - frontend/src/ (React frontend - auto-reloads)"
echo ""
echo "🛑 To stop: Ctrl+C or run: docker-compose -f docker-compose.dev.simple.yml down"