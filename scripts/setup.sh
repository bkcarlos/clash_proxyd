#!/bin/bash

# Dependency Setup Script for Proxyd
# This script downloads and sets up all required dependencies

set -e

echo "🔧 Proxyd Dependency Setup"
echo "=========================="
echo ""

# Check for internet connectivity
echo "📡 Checking internet connectivity..."
if ! ping -c 1 github.com &> /dev/null; then
    echo "⚠️  Warning: Cannot reach GitHub"
    echo ""
    echo "Options:"
    echo "1. Use a Go module proxy (recommended for China):"
    echo "   export GOPROXY=https://goproxy.cn,direct"
    echo ""
    echo "2. Use a VPN"
    echo ""
    echo "3. Run this script from a location with internet access"
    echo ""
    read -p "Continue anyway? (y/N) " -n 1 -r
    echo ""
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# Ask about Go module proxy
read -p "Use Go module proxy for faster downloads? (y/N) " -n 1 -r
echo ""
if [[ $REPLY =~ ^[Yy]$ ]]; then
    export GOPROXY=https://goproxy.cn,direct
    echo "✅ GOPROXY set to: $GOPROXY"
fi

echo ""
echo "📦 Setting up Go dependencies..."
echo "================================"

# Download Go modules
echo "Downloading Go modules..."
go mod download

# Tidy modules
echo "Tidying Go modules..."
go mod tidy

# Verify dependencies
echo "Verifying dependencies..."
go mod verify

echo "✅ Go dependencies ready!"
echo ""

echo "📦 Setting up frontend dependencies..."
echo "======================================"

cd web-ui

# Ask about npm registry
read -p "Use npm mirror for faster downloads? (y/N) " -n 1 -r
echo ""
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "Installing with npmmirror.com..."
    npm install --registry=https://registry.npmmirror.com
else
    echo "Installing from npmjs.org..."
    npm install
fi

cd ..

echo "✅ Frontend dependencies ready!"
echo ""

echo "🏗️  Building project..."
echo "======================="

mkdir -p build

# Build backend
echo "Building backend..."
CGO_ENABLED=1 go build -o build/proxyd ./cmd/proxyd

echo "✅ Build complete!"
echo ""

echo "📁 Setting up directories..."
echo "==========================="

mkdir -p data/db logs

echo "✅ Directories created!"
echo ""

echo "🗄️  Initializing database..."
echo "==========================="

./build/proxyd -c config.example.yaml -init-db

echo "✅ Database initialized!"
echo ""

echo "🧪 Optional acceptance check..."
echo "==============================="
if [ -x "./scripts/acceptance.sh" ]; then
    echo "Run after server startup:"
    echo "  PROXYD_PASSWORD=admin ./scripts/acceptance.sh"
else
    echo "Acceptance script not executable yet."
    echo "Run: chmod +x ./scripts/acceptance.sh"
fi
echo ""

echo "✨ Setup complete!"
echo "=================="
echo ""
echo "Next steps:"
echo "1. Start the server:"
echo "   ./build/proxyd -c config.example.yaml"
echo ""
echo "2. Open http://localhost:8080 in your browser"
echo ""
echo "3. Login with: admin / admin"
echo ""
echo "4. Change the password immediately!"
echo ""
