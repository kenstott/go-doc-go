i#!/bin/bash
# Start both API and frontend servers with environment variables from .env.dev

# Load environment variables from .env.dev
if [ -f .env.dev ]; then
    echo "Loading environment variables from .env.dev..."
    export $(cat .env.dev | grep -v '^#' | xargs)
else
    echo "Warning: .env.dev file not found, using defaults"
fi

echo "🚀 Starting Go-Doc-Go Development Environment"
echo "=============================================="
echo ""
echo "Configuration:"
echo "  Frontend Port: ${FRONTEND_PORT:-3000}"
echo "  API Port: ${API_PORT:-5002}"
echo ""

# Kill any existing processes using the ports
echo "🔪 Killing any existing processes on ports ${FRONTEND_PORT:-3000} and ${API_PORT:-5002}..."
lsof -ti:${FRONTEND_PORT:-3000} | xargs kill -9 2>/dev/null && echo "   Killed process on port ${FRONTEND_PORT:-3000}" || echo "   Port ${FRONTEND_PORT:-3000} is free"
lsof -ti:${API_PORT:-5002} | xargs kill -9 2>/dev/null && echo "   Killed process on port ${API_PORT:-5002}" || echo "   Port ${API_PORT:-5002} is free"
echo ""

# Function to kill both processes on exit
cleanup() {
    echo ""
    echo "Shutting down servers..."
    kill $API_PID $FRONTEND_PID 2>/dev/null
    wait $API_PID $FRONTEND_PID 2>/dev/null
    echo "Servers stopped."
    exit 0
}

# Set up trap to call cleanup on Ctrl+C
trap cleanup INT TERM

# Start API server in background
echo "Starting API server on port ${API_PORT:-5002}..."
PYTHONPATH=src python -m go_doc_go.server &
API_PID=$!

# Give API server a moment to start
sleep 2

# Start frontend server in background
echo "Starting frontend server on port ${FRONTEND_PORT:-3000}..."
(cd frontend && npm run dev) &
FRONTEND_PID=$!

echo ""
echo "✅ Development environment started!"
echo ""
echo "🌐 Frontend: http://localhost:${FRONTEND_PORT:-3000}"
echo "🔧 Backend API: http://localhost:${API_PORT:-5002}"
echo "❤️  Health Check: http://localhost:${API_PORT:-5002}/health"
echo "📊 Pipeline API: http://localhost:${API_PORT:-5002}/api/pipelines"
echo ""
echo "🛑 To stop: Press Ctrl+C"
echo ""

# Wait for both processes
wait $API_PID $FRONTEND_PID