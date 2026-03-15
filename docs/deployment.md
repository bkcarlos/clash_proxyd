# Deployment Guide

This guide covers various deployment options for proxyd.

## Prerequisites

- Linux server (Ubuntu 20.04+, Debian 11+, or CentOS 8+)
- At least 512MB RAM
- 100MB free disk space
- mihomo binary installed

## Installation Methods

### Method 1: Automated Script (Normal User - Recommended)

The easiest way to install proxyd, running as your normal user:

```bash
# Download and run installer
git clone https://github.com/clash-proxyd/proxyd.git
cd proxyd
make build-all
sudo ./scripts/install.sh
```

**Key features:**
- ✅ Service runs as your user (not `nobody`)
- ✅ You own all data and log files
- ✅ No permission issues with mihomo downloads
- ✅ Can debug without root access

The installer automatically detects your username from `SUDO_USER`.

**Specify a different user:**
```bash
sudo SERVICE_USER=alice ./scripts/install.sh
```

### Method 2: Manual Installation with Normal User

#### 1. Install Dependencies

**Ubuntu/Debian:**
```bash
sudo apt update
sudo apt install -y build-essential wget sqlite3 git
```

**CentOS/RHEL:**
```bash
sudo yum install -y gcc wget sqlite git
```

#### 2. Install Go

```bash
wget https://go.dev/dl/go1.22.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.22.0.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc
```

#### 3. Clone and Build

```bash
git clone https://github.com/clash-proxyd/proxyd.git
cd proxyd
make build
make install PREFIX=/opt/proxyd
```

#### 4. Configure

```bash
cd /opt/proxyd
cp config.yaml.example config.yaml
nano config.yaml
```

Edit the configuration file according to your needs.

#### 5. Initialize Database

```bash
/opt/proxyd/bin/proxyd -c /opt/proxyd/config.yaml -init-db
```

#### 6. Install Systemd Service

```bash
# Install service (runs as your user)
sudo make install-service PREFIX=/opt/proxyd
sudo sed -i "s/User=nobody/User=$USER/" /etc/systemd/system/proxyd.service

# Enable and start
sudo systemctl enable proxyd
sudo systemctl start proxyd
```

### Method 3: Docker Deployment

#### Using Docker Compose

```bash
# Clone repository
git clone https://github.com/clash-proxyd/proxyd.git
cd proxyd

# Create configuration
cp config.example.yaml config.yaml
nano config.yaml

# Run with Docker Compose
docker-compose up -d
```

#### Using Docker

```bash
# Build image
docker build -t proxyd:latest .

# Run container
docker run -d \
  --name proxyd \
  -p 8080:8080 \
  -v $(pwd)/data:/opt/proxyd/data \
  -v $(pwd)/logs:/opt/proxyd/logs \
  -v $(pwd)/config.yaml:/opt/proxyd/config.yaml \
  proxyd:latest
```

### Method 4: Binary Deployment

1. Download the latest release binary from GitHub Releases
2. Extract and copy to server
3. Follow manual installation steps from step 4

## Configuration

### Basic Configuration

Edit `/opt/proxyd/config.yaml`:

```yaml
server:
  host: "0.0.0.0"
  port: 8080
  enable_cors: true

database:
  path: "/opt/proxyd/data/db/proxyd.db"

mihomo:
  binary_path: "/usr/local/bin/mihomo"
  config_dir: "/etc/mihomo"
  api_port: 9090

auth:
  jwt_secret: "change-this-to-a-random-secret"
  session_timeout: 86400
```

### Security Hardening

1. **Change JWT Secret:**
   ```yaml
   auth:
     jwt_secret: "$(openssl rand -base64 32)"
   ```

2. **Change Default Password:**
   - Login to web UI
   - Go to Settings
   - Change password

3. **Enable HTTPS:**
   - Set up Nginx reverse proxy
   - Configure SSL certificates

## Reverse Proxy Configuration

### Nginx

```nginx
server {
    listen 443 ssl http2;
    server_name proxyd.example.com;

    ssl_certificate /etc/ssl/certs/proxyd.crt;
    ssl_certificate_key /etc/ssl/private/proxyd.key;

    location /api/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }

    location / {
        root /opt/proxyd/web-ui/dist;
        try_files $uri $uri/ /index.html;
    }
}
```

### Caddy

```
proxyd.example.com {
    reverse_proxy /api/* localhost:8080
    root * /opt/proxyd/web-ui/dist
    try_files {path} /index.html
}
```

## Service Management

### Using Systemd

```bash
# Start service
sudo systemctl start proxyd

# Stop service
sudo systemctl stop proxyd

# Restart service
sudo systemctl restart proxyd

# Check status
sudo systemctl status proxyd

# View logs
sudo journalctl -u proxyd -f

# Enable on boot
sudo systemctl enable proxyd

# Disable on boot
sudo systemctl disable proxyd
```

### Using Docker

```bash
# Start container
docker-compose up -d

# Stop container
docker-compose down

# View logs
docker-compose logs -f

# Restart container
docker-compose restart
```

## Operations Runbook (Systemd / Docker Compose)

Use this runbook for day-2 operations and troubleshooting.

### 1) Fast Health Triage

```bash
# API health
curl -fsS http://127.0.0.1:8080/health

# Runtime status summary (requires login token)
curl -fsS http://127.0.0.1:8080/api/v1/system/status
```

If `/health` fails, check service/container status first, then logs.

### 2) Systemd Path

```bash
# Service state
sudo systemctl status proxyd --no-pager

# Recent logs
sudo journalctl -u proxyd -n 200 --no-pager

# Follow logs
sudo journalctl -u proxyd -f

# Restart
sudo systemctl restart proxyd
```

Recommended unit settings:

```ini
[Service]
Restart=always
RestartSec=3
LimitNOFILE=65535
```

After modifying unit file:

```bash
sudo systemctl daemon-reload
sudo systemctl restart proxyd
```

### 3) Docker Compose Path

```bash
# Service state
docker compose ps

# Recent logs
docker compose logs --tail=200 proxyd

# Follow logs
docker compose logs -f proxyd

# Restart only proxyd service
docker compose restart proxyd

# Recreate container after config/image change
docker compose up -d --force-recreate proxyd
```

Recommended compose policy:

```yaml
services:
  proxyd:
    restart: unless-stopped
```

### 4) Startup Self-check and Audit Signals

proxyd writes startup checks into both logs and audit logs:

- `mihomo_binary`
- `mihomo_config_dir`
- `database_path`

Use these to quickly identify misconfiguration (missing binary, unwritable paths, etc.).

### 5) Common Failure Patterns

#### A. `bind: address already in use`

- Change `server.port` or release the port.
- Re-run:

```bash
sudo lsof -i :8080
```

#### B. `mihomo did not become ready`

- Verify `mihomo.binary_path` points to a real executable.
- Verify `mihomo.api_port` is reachable and not occupied.
- Verify config file path is inside `mihomo.config_dir`.

#### C. `path not writable` / database write failures

- Check parent directory ownership and permissions.
- Ensure runtime user can write:

```bash
# If running as normal user (recommended)
touch /opt/proxyd/data/db/.write-test
ls -la /opt/proxyd/data/db/

# Fix permissions if needed
sudo chown -R $USER:$USER /opt/proxyd/data /opt/proxyd/logs
sudo chown root:$USER /opt/proxyd/config.yaml
```

#### D. Login returns 401 unexpectedly

- Confirm DB was initialized with schema and settings.
- Validate admin credentials in settings table.

### 6) Safe Recovery Sequence

1. Stop traffic entry (reverse proxy / LB) if needed.
2. Capture logs (`journalctl` or `docker compose logs`).
3. Verify config + writable paths.
4. Restart proxyd.
5. Verify `/health` and `/api/v1/system/status`.
6. Re-enable traffic.

---

## Updates

### Update from Source

```bash
cd /opt/proxyd
git pull
make clean
make build
sudo systemctl restart proxyd
```

### Update Docker

```bash
docker-compose pull
docker-compose up -d --build
```

## Monitoring

### Health Check

```bash
curl http://localhost:8080/health
```

Expected response:
```json
{
  "status": "ok",
  "service": "proxyd"
}
```

### Logs

**Systemd:**
```bash
sudo journalctl -u proxyd -f
```

**Docker:**
```bash
docker-compose logs -f proxyd
```

**Application:**
```bash
tail -f /opt/proxyd/logs/proxyd.log
```

## Backup

### Database Backup

```bash
# Backup database
cp /opt/proxyd/data/db/proxyd.db /backup/proxyd-$(date +%Y%m%d).db

# Automated backup
crontab -e
# Add: 0 2 * * * cp /opt/proxyd/data/db/proxyd.db /backup/proxyd-$(date +\%Y\%m\%d).db
```

### Configuration Backup

```bash
# Backup config
cp /opt/proxyd/config.yaml /backup/config.yaml
```

## Troubleshooting

### Service Won't Start

1. Check configuration:
   ```bash
   /opt/proxyd/bin/proxyd -c /opt/proxyd/config.yaml -check
   ```

2. Check logs:
   ```bash
   sudo journalctl -u proxyd -n 50
   ```

3. Check ports:
   ```bash
   sudo lsof -i :8080
   ```

### Database Issues

1. Check database file:
   ```bash
   ls -lh /opt/proxyd/data/db/proxyd.db
   ```

2. Reinitialize:
   ```bash
   /opt/proxyd/bin/proxyd -c /opt/proxyd/config.yaml -init-db
   ```

### mihomo Connection Issues

1. Verify mihomo is running:
   ```bash
   ps aux | grep mihomo
   ```

2. Test mihomo API:
   ```bash
   curl http://localhost:9090
   ```

3. Check mihomo configuration

### Performance Issues

1. Check system resources:
   ```bash
   top
   df -h
   ```

2. Check log level (use `info` or `warn` in production)

3. Monitor database size and optimize if needed

## Firewall Configuration

### UFW (Ubuntu)

```bash
sudo ufw allow 8080/tcp
sudo ufw reload
```

### firewalld (CentOS)

```bash
sudo firewall-cmd --permanent --add-port=8080/tcp
sudo firewall-cmd --reload
```

## SSL Certificate Setup

### Let's Encrypt with Certbot

```bash
# Install certbot
sudo apt install certbot

# Generate certificate
sudo certbot certonly --standalone -d proxyd.example.com

# Certificates will be at:
# /etc/letsencrypt/live/proxyd.example.com/fullchain.pem
# /etc/letsencrypt/live/proxyd.example.com/privkey.pem

# Auto-renewal is configured by default
```

## Production Checklist

- [ ] Service runs as normal user (not root/nobody)
- [ ] Change JWT secret
- [ ] Change default password
- [ ] Configure HTTPS
- [ ] Set up firewall rules
- [ ] Configure log rotation
- [ ] Set up automated backups
- [ ] Configure monitoring
- [ ] Test disaster recovery
- [ ] Document custom configurations
- [ ] Set up update procedure

**See also:** [普通用户部署指南](user-deployment.md) for detailed permission setup.
