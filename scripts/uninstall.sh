#!/usr/bin/env bash
# uninstall.sh — Remove proxyd and clean up all installed files
#
# Usage:
#   sudo ./scripts/uninstall.sh            # remove service + files, keep data
#   sudo ./scripts/uninstall.sh --purge    # remove everything including data & user

set -euo pipefail

INSTALL_DIR="${INSTALL_DIR:-/opt/proxyd}"
SERVICE_FILE="/etc/systemd/system/proxyd.service"

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; CYAN='\033[0;36m'; NC='\033[0m'
info()  { echo -e "${GREEN}[INFO]${NC}  $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC}  $*"; }
step()  { echo -e "\n${CYAN}▶ $*${NC}"; }

PURGE=false
for arg in "$@"; do
    [[ "$arg" == "--purge" ]] && PURGE=true
done

if [[ $EUID -ne 0 ]]; then
    echo -e "${RED}[ERROR]${NC} Run with sudo or as root." >&2
    exit 1
fi

echo -e "${YELLOW}This will remove proxyd from this system.${NC}"
if [[ "$PURGE" == true ]]; then
    echo -e "${RED}--purge: ALL data in $INSTALL_DIR will be permanently deleted.${NC}"
fi
read -r -p "Continue? [y/N] " confirm
[[ "${confirm,,}" == "y" ]] || { info "Aborted."; exit 0; }

# ── Stop and disable service ──────────────────────────────────────────────────
step "Stopping service"
if systemctl is-active --quiet proxyd 2>/dev/null; then
    systemctl stop proxyd
    info "Service stopped."
fi
if systemctl is-enabled --quiet proxyd 2>/dev/null; then
    systemctl disable proxyd
    info "Service disabled."
fi

# ── Remove service file ───────────────────────────────────────────────────────
step "Removing systemd unit"
if [[ -f "$SERVICE_FILE" ]]; then
    rm -f "$SERVICE_FILE"
    systemctl daemon-reload
    info "Removed: $SERVICE_FILE"
fi

# ── Remove binary ─────────────────────────────────────────────────────────────
step "Removing binary"
if [[ -f "$INSTALL_DIR/bin/proxyd" ]]; then
    rm -f "$INSTALL_DIR/bin/proxyd"
    info "Removed: $INSTALL_DIR/bin/proxyd"
fi
if [[ -f "$INSTALL_DIR/bin/mihomo" ]]; then
    rm -f "$INSTALL_DIR/bin/mihomo"
    info "Removed: $INSTALL_DIR/bin/mihomo"
fi
[[ -d "$INSTALL_DIR/bin" ]] && rmdir --ignore-fail-on-non-empty "$INSTALL_DIR/bin"

# ── Purge: remove all data ────────────────────────────────────────────────────
if [[ "$PURGE" == true ]]; then
    step "Purging install directory"
    if [[ -d "$INSTALL_DIR" ]]; then
        rm -rf "$INSTALL_DIR"
        info "Removed: $INSTALL_DIR"
    fi
else
    warn "Data preserved at $INSTALL_DIR/data and $INSTALL_DIR/logs."
    warn "Run with --purge to delete everything."
fi

echo ""
info "proxyd uninstalled."
