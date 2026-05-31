#!/usr/bin/env bash
# deploy.sh — Download a published proxyd release and install it as a systemd
# service under a fixed prefix (default /opt/proxyd).
#
# This runs standalone: no repo checkout, no local build, no dependency on the
# current working directory. The installed service uses absolute paths, so it
# is independent of where this script was run from.
#
# Quick install (latest release):
#   curl -fsSL https://raw.githubusercontent.com/bkcarlos/clash_proxyd/master/scripts/deploy.sh | sudo bash
#
# Or, with a specific version / options:
#   sudo VERSION=v1.0.2 ./scripts/deploy.sh
#
# Offline / GitHub-blocked servers — install from a local tarball:
#   On a machine with access, download proxyd_<ver>_linux_<arch>.tar.gz (and
#   optionally its .sha256) from the Releases page, copy it + this script to the
#   server, then run from the same directory:
#     sudo ./deploy.sh                              # auto-detects ./proxyd_*_linux_<arch>.tar.gz
#     sudo ./deploy.sh /path/to/proxyd_..._.tar.gz  # or pass the path explicitly
#   The local archive is used when present; GitHub is only the fallback.
#
# Environment overrides (all optional):
#   ARCHIVE=<path>            install from this local .tar.gz (skips download)
#   VERSION=latest            release tag to install (default: latest)
#   INSTALL_DIR=/opt/proxyd    install prefix
#   SERVICE_USER=...           user to run the service
#                              (default: the sudo invoker; else a created
#                               'proxyd' system user)
#   REPO=bkcarlos/clash_proxyd GitHub owner/repo to download from
#   BIND_HOST=127.0.0.1        API listen host (use 0.0.0.0 to expose on LAN)
#   API_PORT=8080              proxyd API/UI port
#   MIHOMO_PORT=7890           mihomo mixed-proxy port

set -euo pipefail

# ── Defaults ────────────────────────────────────────────────────────────────
REPO="${REPO:-bkcarlos/clash_proxyd}"
INSTALL_DIR="${INSTALL_DIR:-/opt/proxyd}"
VERSION="${VERSION:-latest}"
BIND_HOST="${BIND_HOST:-127.0.0.1}"
API_PORT="${API_PORT:-8080}"
MIHOMO_PORT="${MIHOMO_PORT:-7890}"
SERVICE_FILE="/etc/systemd/system/proxyd.service"

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; CYAN='\033[0;36m'; NC='\033[0m'
info()  { echo -e "${GREEN}[INFO]${NC}  $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC}  $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*" >&2; }
step()  { echo -e "\n${CYAN}▶ $*${NC}"; }

# ── Pre-flight ──────────────────────────────────────────────────────────────
[[ $EUID -eq 0 ]] || { error "Run with sudo or as root."; exit 1; }

for cmd in curl tar systemctl install; do
    command -v "$cmd" >/dev/null 2>&1 || { error "Required command not found: $cmd"; exit 1; }
done

# sha256 tool (Linux: sha256sum; fall back to shasum -a 256)
if command -v sha256sum >/dev/null 2>&1; then
    SHA256() { sha256sum "$1" | awk '{print $1}'; }
elif command -v shasum >/dev/null 2>&1; then
    SHA256() { shasum -a 256 "$1" | awk '{print $1}'; }
else
    SHA256() { echo ""; }  # checksum verification skipped
fi

# ── Architecture ────────────────────────────────────────────────────────────
case "$(uname -m)" in
    x86_64|amd64)   ARCH=amd64 ;;
    aarch64|arm64)  ARCH=arm64 ;;
    *) error "Unsupported architecture: $(uname -m) (release builds: amd64, arm64)"; exit 1 ;;
esac
info "Architecture: linux/${ARCH}"

# ── Resolve archive source: local file first, else GitHub ──────────────────
# Priority: positional arg > $ARCHIVE env > ./proxyd_*_linux_<arch>.tar.gz in CWD.
# A local archive lets you install on servers that cannot reach GitHub.
SRC_ARCHIVE="${1:-${ARCHIVE:-}}"
if [[ -z "$SRC_ARCHIVE" ]]; then
    if [[ "$VERSION" != "latest" && -f "$PWD/proxyd_${VERSION}_linux_${ARCH}.tar.gz" ]]; then
        SRC_ARCHIVE="$PWD/proxyd_${VERSION}_linux_${ARCH}.tar.gz"
    else
        SRC_ARCHIVE="$(ls -1t "$PWD"/proxyd_*_linux_${ARCH}.tar.gz 2>/dev/null | head -1 || true)"
    fi
fi

if [[ -n "$SRC_ARCHIVE" ]]; then
    [[ -f "$SRC_ARCHIVE" ]] || { error "Archive not found: $SRC_ARCHIVE"; exit 1; }
    # Derive the version from proxyd_<ver>_linux_<arch>.tar.gz, else "local".
    base="$(basename "$SRC_ARCHIVE")"
    if [[ "$base" =~ ^proxyd_(.+)_linux_(amd64|arm64)\.tar\.gz$ ]]; then
        VERSION="${BASH_REMATCH[1]}"
        [[ "${BASH_REMATCH[2]}" == "$ARCH" ]] || warn "Archive arch '${BASH_REMATCH[2]}' differs from host arch '$ARCH'."
    else
        VERSION="local"
    fi
    info "Source: local archive — $SRC_ARCHIVE (version: ${VERSION})"
else
    if [[ "$VERSION" == "latest" ]]; then
        step "Resolving latest release of $REPO"
        VERSION="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
            | grep -o '"tag_name"[[:space:]]*:[[:space:]]*"[^"]*"' | head -1 \
            | sed 's/.*"\([^"]*\)"$/\1/')"
        if [[ -z "$VERSION" ]]; then
            error "Could not resolve the latest release (is GitHub reachable?)."
            error "On servers without GitHub access: download proxyd_<ver>_linux_${ARCH}.tar.gz"
            error "elsewhere, copy it next to this script, and re-run: sudo ./deploy.sh"
            exit 1
        fi
    fi
    info "Source: GitHub release ${VERSION} (${REPO})"
fi

# ── Service user ────────────────────────────────────────────────────────────
if [[ -z "${SERVICE_USER:-}" ]]; then
    if [[ -n "${SUDO_USER:-}" && "${SUDO_USER}" != "root" ]]; then
        SERVICE_USER="$SUDO_USER"
    else
        SERVICE_USER="proxyd"
    fi
fi
if ! getent passwd "$SERVICE_USER" >/dev/null 2>&1; then
    step "Creating system user '$SERVICE_USER'"
    useradd --system --no-create-home --shell /usr/sbin/nologin "$SERVICE_USER" \
        || { error "Failed to create user '$SERVICE_USER'."; exit 1; }
fi
SERVICE_GROUP="$(id -gn "$SERVICE_USER")"
info "Service runs as: ${SERVICE_USER}:${SERVICE_GROUP}"

# ── Obtain & verify the archive ─────────────────────────────────────────────
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
ARCHIVE_TGZ="$TMP/proxyd.tar.gz"

if [[ -n "$SRC_ARCHIVE" ]]; then
    step "Using local archive"
    cp "$SRC_ARCHIVE" "$ARCHIVE_TGZ"
    if [[ -f "${SRC_ARCHIVE}.sha256" ]]; then
        EXPECTED="$(awk '{print $1}' "${SRC_ARCHIVE}.sha256")"
        ACTUAL="$(SHA256 "$ARCHIVE_TGZ")"
        if [[ -n "$ACTUAL" && -n "$EXPECTED" ]]; then
            [[ "$ACTUAL" == "$EXPECTED" ]] || { error "Checksum mismatch for $SRC_ARCHIVE"; exit 1; }
            info "Checksum verified (local .sha256)."
        fi
    else
        warn "No sibling .sha256 — skipping checksum verification."
    fi
else
    ASSET="proxyd_${VERSION}_linux_${ARCH}.tar.gz"
    BASE_URL="https://github.com/${REPO}/releases/download/${VERSION}"
    step "Downloading ${ASSET}"
    curl -fL# -o "$ARCHIVE_TGZ" "${BASE_URL}/${ASSET}" \
        || { error "Download failed: ${BASE_URL}/${ASSET}";
             error "If this server can't reach GitHub, copy the tarball here and re-run (or pass its path)."; exit 1; }
    if curl -fsSL -o "$ARCHIVE_TGZ.sha256" "${BASE_URL}/${ASSET}.sha256" 2>/dev/null; then
        EXPECTED="$(awk '{print $1}' "$ARCHIVE_TGZ.sha256")"
        ACTUAL="$(SHA256 "$ARCHIVE_TGZ")"
        if [[ -n "$ACTUAL" && -n "$EXPECTED" ]]; then
            [[ "$ACTUAL" == "$EXPECTED" ]] || { error "Checksum mismatch! expected=$EXPECTED actual=$ACTUAL"; exit 1; }
            info "Checksum verified."
        else
            warn "Checksum tool unavailable — skipping verification."
        fi
    else
        warn "No .sha256 published — skipping checksum verification."
    fi
fi

step "Extracting"
tar -xzf "$ARCHIVE_TGZ" -C "$TMP"
BIN_SRC="$(find "$TMP" -type f -name proxyd | head -1)"
[[ -n "$BIN_SRC" && -f "$BIN_SRC" ]] || { error "proxyd binary not found in archive."; exit 1; }

# ── Detect upgrade ──────────────────────────────────────────────────────────
UPGRADE=false
[[ -f "$INSTALL_DIR/bin/proxyd" ]] && { UPGRADE=true; warn "Existing install at $INSTALL_DIR — upgrading (config preserved)."; }

# ── Directory layout ────────────────────────────────────────────────────────
step "Installing to $INSTALL_DIR"
install -d -m 755 "$INSTALL_DIR/bin"
install -d -m 750 -o "$SERVICE_USER" -g "$SERVICE_GROUP" \
    "$INSTALL_DIR/data/db" "$INSTALL_DIR/data/mihomo" \
    "$INSTALL_DIR/data/generated" "$INSTALL_DIR/data/cache" "$INSTALL_DIR/logs"

install -m 755 "$BIN_SRC" "$INSTALL_DIR/bin/proxyd"
info "Binary: $INSTALL_DIR/bin/proxyd ($VERSION)"

# ── Config (absolute paths; generated once) ─────────────────────────────────
CONFIG_FILE="$INSTALL_DIR/config.yaml"
if [[ "$UPGRADE" == true && -f "$CONFIG_FILE" ]]; then
    info "Keeping existing config: $CONFIG_FILE"
else
    JWT_SECRET="$(openssl rand -hex 32 2>/dev/null || head -c 32 /dev/urandom | od -An -tx1 | tr -d ' \n')"
    cat > "$CONFIG_FILE" <<YAML
# proxyd configuration — generated by deploy.sh ($VERSION)
# Edit then: systemctl restart proxyd
server:
  host: "${BIND_HOST}"
  port: ${API_PORT}
  enable_cors: false
database:
  path: "${INSTALL_DIR}/data/db/proxyd.db"
  foreign_keys: true
mihomo:
  binary_path: ""          # empty = bundled mihomo, extracted next to the proxyd binary
  config_dir: "${INSTALL_DIR}/data/mihomo"
  api_port: 9090
  api_secret: ""
  log_dir: "${INSTALL_DIR}/logs"
  auto_update_enabled: false
  auto_update_check_on_start: false
  release_api: "https://api.github.com/repos/MetaCubeX/mihomo/releases/latest"
  download_dir: "${INSTALL_DIR}/bin"
auth:
  jwt_secret: "${JWT_SECRET}"
  session_timeout: 86400
  max_login_attempts: 5
  lockout_duration: 300
logging:
  level: "info"
  output: "file"
  file_path: "${INSTALL_DIR}/logs/proxyd.log"
  max_size: 100
  max_backups: 5
  max_age: 14
  compress: true
subscription:
  default_interval: 3600
  timeout: 30
  user_agent: "clash.meta"
  max_retries: 3
  retry_delay: 5
policy:
  mixed_port: ${MIHOMO_PORT}
  allow_lan: false
  bind_address: "127.0.0.1"
  log_level: "info"
  mode: "rule"
  external_controller: "127.0.0.1:9090"
  ipv6: false
scheduler:
  enabled: true
  workers: 3
YAML
    info "Config: $CONFIG_FILE (random jwt_secret generated)"
fi
chown root:"$SERVICE_GROUP" "$CONFIG_FILE"
chmod 640 "$CONFIG_FILE"

# Service user owns data/logs; bin writable for bundled-mihomo extraction.
chown -R "$SERVICE_USER":"$SERVICE_GROUP" "$INSTALL_DIR/data" "$INSTALL_DIR/logs"
chown "$SERVICE_USER":"$SERVICE_GROUP" "$INSTALL_DIR/bin" 2>/dev/null || true
chmod 775 "$INSTALL_DIR/bin"

# ── Initialise database ─────────────────────────────────────────────────────
if [[ ! -f "$INSTALL_DIR/data/db/proxyd.db" ]]; then
    step "Initialising database"
    sudo -u "$SERVICE_USER" "$INSTALL_DIR/bin/proxyd" -c "$CONFIG_FILE" -init-db
fi

# ── systemd unit (absolute paths → CWD-independent) ─────────────────────────
step "Installing systemd service"
cat > "$SERVICE_FILE" <<UNIT
[Unit]
Description=Proxyd - Mihomo Proxy Manager
Documentation=https://github.com/${REPO}
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=${SERVICE_USER}
WorkingDirectory=${INSTALL_DIR}
EnvironmentFile=-${INSTALL_DIR}/proxyd.env
ExecStart=${INSTALL_DIR}/bin/proxyd -c ${INSTALL_DIR}/config.yaml -web
Restart=always
RestartSec=3s
TimeoutStopSec=15s
LimitNOFILE=65535
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=${INSTALL_DIR}/data ${INSTALL_DIR}/logs ${INSTALL_DIR}/bin
StandardOutput=journal
StandardError=journal
SyslogIdentifier=proxyd

[Install]
WantedBy=multi-user.target
UNIT
chmod 644 "$SERVICE_FILE"

systemctl daemon-reload
systemctl enable proxyd >/dev/null 2>&1 || true

step "Starting service"
if systemctl is-active --quiet proxyd; then
    systemctl restart proxyd
else
    systemctl start proxyd
fi

# ── Summary ─────────────────────────────────────────────────────────────────
echo ""
echo -e "${GREEN}proxyd ${VERSION} installed.${NC}"
echo -e "  Web UI : http://${BIND_HOST}:${API_PORT}   (login: admin / admin — change it!)"
echo -e "  Config : ${CONFIG_FILE}"
echo -e "  Logs   : journalctl -u proxyd -f"
echo -e "  Manage : systemctl {status,restart,stop} proxyd"
[[ "$BIND_HOST" == "127.0.0.1" ]] && \
    echo -e "  ${YELLOW}Note:${NC} UI bound to localhost. Use an SSH tunnel/reverse proxy, or re-run with BIND_HOST=0.0.0.0."
