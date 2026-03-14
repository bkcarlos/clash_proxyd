# Setup Instructions

## Initial Setup (Requires Internet Access)

### 1. Install Go Dependencies

**Important**: This step requires internet access to download Go modules from GitHub.

```bash
cd /code/clash_proxyd

# Download Go dependencies
go mod download

# Tidy Go modules (ensures correct versions)
go mod tidy

# Verify dependencies
go mod verify
```

If you're in China or experiencing GitHub connectivity issues, you can use a proxy:

```bash
# Use Go module proxy (for China)
export GOPROXY=https://goproxy.cn,direct

# Then run go mod tidy
go mod tidy
```

### 2. Install Frontend Dependencies

```bash
cd web-ui

# Install npm dependencies
npm install

# Or use a mirror for faster downloads in China
npm install --registry=https://registry.npmmirror.com
```

### 3. Build the Project

```bash
# From project root
cd /code/clash_proxyd

# Build everything
make build

# Or build separately
make build-backend
make build-frontend
```

### 4. Run Tests (Recommended)

```bash
# Backend tests
go test ./...

# Frontend type-check + build
npm --prefix web-ui run build
```

### 5. Initialize Database

```bash
# Create necessary directories
mkdir -p data/db logs

# Initialize database schema
./build/proxyd -c config.example.yaml -init-db
```

### 6. Acceptance Gate (Required before next phase)

After the backend is up, run the acceptance script:

```bash
# Terminal 1
./build/proxyd -c config.example.yaml

# Terminal 2
PROXYD_PASSWORD=admin ./scripts/acceptance.sh
```

Report output:
- `logs/acceptance-*.md`
- includes pass/fail/skip summary and reproduction command

> Notes:
> - In environments where mihomo cannot be started (missing binary / restricted sandbox), mihomo-dependent checks are marked as `SKIP` instead of `FAIL`.
> - For full end-to-end production acceptance, run once in a host where real mihomo is installed and reachable.

### 6.1 Phase 4 Verification Add-on (Rollback / Cache / WS / Alerts)

After `Acceptance Gate` passes, run this add-on checklist to verify phase-4 capabilities.

#### A) Revision rollback API

```bash
TOKEN=$(curl -sS -X POST http://127.0.0.1:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"admin"}' | \
  python3 -c 'import json,sys; print(json.load(sys.stdin).get("token",""))')

# pick the 2nd latest revision (if exists)
REV_ID=$(curl -sS "http://127.0.0.1:8080/api/v1/config/revisions?limit=2" \
  -H "Authorization: Bearer ${TOKEN}" | \
  python3 -c 'import json,sys; data=json.load(sys.stdin); print(data[1]["id"] if len(data) > 1 else "")')

# rollback (when REV_ID is not empty)
curl -sS -X POST "http://127.0.0.1:8080/api/v1/config/revisions/${REV_ID}/rollback" \
  -H "Authorization: Bearer ${TOKEN}"
```

Expected: success response with `revision` and `path`.

#### B) Proxy delay cache (requires mihomo available)

```bash
PROXY_NAME=$(curl -sS http://127.0.0.1:8080/api/v1/proxy/groups \
  -H "Authorization: Bearer ${TOKEN}" | \
  python3 -c 'import json,sys; d=json.load(sys.stdin); g=(d.get("groups") or [{}])[0]; p=(g.get("proxies") or [""])[0]; print(p)')

# first call -> usually from_cache=false
curl -sS -X POST "http://127.0.0.1:8080/api/v1/proxy/proxies/${PROXY_NAME}/test" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H 'Content-Type: application/json' \
  -d '{"url":"http://www.gstatic.com/generate_204","timeout":3000}'

# second call (same params, short interval) -> expect from_cache=true
curl -sS -X POST "http://127.0.0.1:8080/api/v1/proxy/proxies/${PROXY_NAME}/test" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H 'Content-Type: application/json' \
  -d '{"url":"http://www.gstatic.com/generate_204","timeout":3000}'
```

Expected: second response contains `"from_cache": true`.

#### C) WS + alert summary refresh (Dashboard)

1. Open Dashboard page and keep it visible.
2. Trigger an alert event (for example, stop mihomo abnormally or simulate source update failures).
3. Confirm `Last Alert` block updates automatically (without manual page refresh).

Expected: `Last Alert` action/details/time update within a few seconds.

### 7. Run the Application

```bash
# Start backend
./build/proxyd -c config.example.yaml
```

The backend will start on `http://localhost:8080`

## Offline/No-Internet Setup

If you're in an environment without internet access, you have a few options:

### Option 1: Use Vendor Directory

If you have access to a machine with internet:

```bash
# On machine with internet
go mod vendor
tar -czf vendor.tar.gz vendor/

# Transfer vendor.tar.gz to target machine
# On target machine
tar -xzf vendor.tar.gz
go build -mod=vendor ./...
```

### Option 2: Pre-built Binary

Build the binary on a machine with internet and transfer it:

```bash
# On machine with internet
make build-backend

# Transfer build/proxyd to target machine
# Then just run it
./proxyd -c config.yaml
```

### Option 3: Air-gapped Installation

1. Download all Go module dependencies manually
2. Set up a local Go module proxy
3. Configure GOPROXY to use your local proxy

## Troubleshooting

### "module ... not found" Errors

This means dependencies haven't been downloaded. Run:
```bash
go mod download
```

### "git ls-remote" Errors

This indicates network connectivity issues to GitHub. Solutions:
- Check your internet connection
- Use a VPN if needed
- Configure Go module proxy:
  ```bash
  export GOPROXY=https://goproxy.cn,direct
  ```

### CGO Errors

The sqlite3 package requires CGO. Ensure you have GCC installed:

**Ubuntu/Debian:**
```bash
sudo apt-get install build-essential libsqlite3-dev
```

**CentOS/RHEL:**
```bash
sudo yum install gcc sqlite-devel
```

**macOS:**
```bash
xcode-select --install
```

### Port Already in Use

If port 8080 is already in use:
```bash
# Find the process
lsof -i :8080

# Kill it
kill -9 <PID>

# Or change port in config.yaml
```

## Verification

Test that everything is working:

```bash
# Test backend API
curl http://localhost:8080/health

# Expected response:
# {"status":"ok","service":"proxyd"}

# Test database
ls -lh data/db/proxyd.db

# Test frontend (if running)
curl http://localhost:5173
```

## Development Setup

For development, use the provided script:

```bash
# This will start both backend and frontend
./scripts/dev.sh
```

Or manually:

```bash
# Terminal 1 - Backend
./build/proxyd -c config.example.yaml

# Terminal 2 - Frontend
cd web-ui
npm run dev
```

## Production Deployment

See `docs/deployment.md` for detailed production deployment instructions.

## System Requirements

- **Go**: 1.22 or higher
- **Node.js**: 20 or higher
- **GCC/Clang**: For CGO (sqlite3)
- **RAM**: 512MB minimum, 1GB recommended
- **Disk**: 100MB minimum

## Next Steps

Once setup is complete:

1. Login to web UI at `http://localhost:8080`
2. Default credentials: `admin` / `admin`
3. **Important**: Change the password immediately!
4. Add your subscription sources
5. Generate and apply configuration
6. Start mihomo through the web interface

## Getting Help

If you encounter issues:

1. Check logs: `tail -f logs/proxyd.log`
2. Check system logs: `journalctl -u proxyd -f` (if using systemd)
3. Review documentation in `docs/`
4. Open an issue on GitHub
