#!/bin/bash
# One-command development environment with hot reloading

# Load environment variables from .env.dev
if [ -f .env.dev ]; then
    echo "📋 Loading environment variables from .env.dev..."
    export $(cat .env.dev | grep -v '^#' | xargs)
else
    echo "⚠️  Warning: .env.dev file not found, using defaults"
fi

# Kill any existing processes using the ports
echo ""
echo "🔪 Killing any existing processes on ports ${FRONTEND_PORT:-3000} and ${API_PORT:-5002}..."
lsof -ti:${FRONTEND_PORT:-3000} | xargs kill -9 2>/dev/null && echo "   Killed process on port ${FRONTEND_PORT:-3000}" || echo "   Port ${FRONTEND_PORT:-3000} is free"
lsof -ti:${API_PORT:-5002} | xargs kill -9 2>/dev/null && echo "   Killed process on port ${API_PORT:-5002}" || echo "   Port ${API_PORT:-5002} is free"

echo ""
echo "🚀 Starting Go-Doc-Go Development Environment"
echo "============================================="
echo "Configuration:"
echo "  Frontend Port: ${FRONTEND_PORT:-3000}"
echo "  API Port: ${API_PORT:-5002}"

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
docker-compose -f docker-compose.dev.yml down

# Start everything with hot reloading
echo ""
echo "🔥 Starting with hot reload..."
docker-compose -f docker-compose.dev.yml up --build

echo "✅ Development environment started!"
echo ""
echo "🌐 Frontend: http://localhost:${FRONTEND_PORT:-3000}"
echo "🔧 Backend API: http://localhost:${API_PORT:-5002}"
echo "❤️  Health Check: http://localhost:${API_PORT:-5002}/health"
echo "📊 Pipeline API: http://localhost:${API_PORT:-5002}/api/pipelines"
echo ""
echo "📁 Edit files in:"
echo "   - src/go_doc_go/ (Python backend - auto-reloads)"
echo "   - frontend/src/ (React frontend - auto-reloads)"
echo ""
echo "🛑 To stop: Ctrl+C or run: docker-compose -f docker-compose.dev.yml down"