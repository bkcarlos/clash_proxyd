#!/bin/bash

set -euo pipefail

echo "Starting Proxyd Development Environment..."

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "Go is not installed. Please install Go 1.22+ first."
    exit 1
fi

# Check if Node.js is installed
if ! command -v node &> /dev/null; then
    echo "Node.js is not installed. Please install Node.js 20+ first."
    exit 1
fi

# Install Go dependencies
echo "Installing Go dependencies..."
go mod download

# Install frontend dependencies
echo "Installing frontend dependencies..."
cd web-ui
if [ ! -d "node_modules" ]; then
    npm install
fi
cd ..

# Initialize database if not exists
if [ ! -f "data/db/proxyd.db" ]; then
    echo "Initializing database..."
    mkdir -p data/db logs
    go run cmd/proxyd/main.go -c config.example.yaml -init-db
fi

# Build and run backend
echo "Building backend..."
go build -o build/proxyd cmd/proxyd/main.go

# Start backend in background
echo "Starting backend server..."
./build/proxyd -c config.example.yaml &
BACKEND_PID=$!

# Wait for backend health endpoint
for _ in {1..20}; do
    if curl -fsS http://localhost:8080/health >/dev/null 2>&1; then
        break
    fi
    sleep 1
done

# Start frontend
echo "Starting frontend development server..."
cd web-ui
npm run dev &
FRONTEND_PID=$!

echo ""
echo "Development environment started"
echo ""
echo "Backend running at: http://localhost:8080"
echo "Frontend running at: http://localhost:5173"
echo ""
echo "Quick acceptance run:"
echo "  PROXYD_PASSWORD=admin ./scripts/acceptance.sh"
echo ""
echo "Press Ctrl+C to stop all services"
echo ""

# Handle shutdown
trap "echo 'Stopping services...'; kill $BACKEND_PID $FRONTEND_PID; exit 0" INT TERM

# Wait for processes
wait
