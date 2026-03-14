#!/bin/bash

# Proxyd Installation Script

set -e

# Configuration
INSTALL_DIR="/opt/proxyd"
SERVICE_NAME="proxyd"
REPO_URL="https://github.com/clash-proxyd/proxyd"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    log_error "Please run as root"
    exit 1
fi

log_info "Installing proxyd..."

# Install dependencies
if command -v apt-get &> /dev/null; then
    log_info "Installing dependencies with apt..."
    apt-get update
    apt-get install -y build-essential wget sqlite3
elif command -v yum &> /dev/null; then
    log_info "Installing dependencies with yum..."
    yum install -y gcc wget sqlite
else
    log_error "Unsupported package manager"
    exit 1
fi

# Install Go if not present
if ! command -v go &> /dev/null; then
    log_info "Installing Go..."
    GO_VERSION="1.21.5"
    wget -q https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz
    tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz
    echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile
    export PATH=$PATH:/usr/local/go/bin
    rm go${GO_VERSION}.linux-amd64.tar.gz
fi

# Create directories
log_info "Creating directories..."
mkdir -p $INSTALL_DIR/bin
mkdir -p $INSTALL_DIR/data/db
mkdir -p $INSTALL_DIR/logs
mkdir -p $INSTALL_DIR/web-ui

# Build proxyd
log_info "Building proxyd..."
cd /tmp
git clone $REPO_URL proxyd
cd proxyd
make build
make install PREFIX=$INSTALL_DIR

# Install systemd service
log_info "Installing systemd service..."
cat > /etc/systemd/system/${SERVICE_NAME}.service << EOF
[Unit]
Description=Proxyd - Mihomo Proxy Manager
After=network.target

[Service]
Type=simple
User=www-data
Group=www-data
WorkingDirectory=$INSTALL_DIR
ExecStart=$INSTALL_DIR/bin/proxyd -c $INSTALL_DIR/config.yaml
Restart=on-failure
RestartSec=5s
StandardOutput=journal
StandardError=journal
SyslogIdentifier=proxyd

[Install]
WantedBy=multi-user.target
EOF

# Create config if not exists
if [ ! -f "$INSTALL_DIR/config.yaml" ]; then
    log_info "Creating default configuration..."
    cp config.example.yaml $INSTALL_DIR/config.yaml
fi

# Set permissions
chown -R www-data:www-data $INSTALL_DIR

# Enable and start service
systemctl daemon-reload
systemctl enable proxyd
systemctl start proxyd

log_info "Installation complete!"
log_info "Proxyd is now running at http://localhost:8080"
log_warn "Please edit $INSTALL_DIR/config.yaml and restart the service"
