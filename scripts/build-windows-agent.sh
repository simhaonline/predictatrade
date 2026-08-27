#!/usr/bin/env bash
#
# build-windows-agent.sh — Build Windows Agent (Client + Master) and auto-update deploy folders
#
# Usage:
#   ./scripts/build-windows-agent.sh              # Build with current version
#   ./scripts/build-windows-agent.sh --bump       # Build + bump patch version
#   ./scripts/build-windows-agent.sh --version 1.3.0  # Build with specific version
#
# What it does (in order):
#   1. Reads (or sets) the version from windows-agent/internal/version.go
#   2. Cross-compiles pat-agent.exe     (Client / execution)  → bin/pat-agent.exe
#   3. Cross-compiles pat-master.exe    (Master Node / data)  → bin/pat-master.exe
#   4. Copies standalone deploy binaries into the role subdirs the installers
#      fetch from (deploy/client/pat-agent.exe and deploy/master/pat-master.exe),
#      plus the legacy root copies for backward compatibility.
#   5. Calculates SHA256 checksums of both binaries.
#   6. Updates deploy/version.txt (shared) and both role update-manifests.
#   7. Prints a summary.
#
# The deploy folder is served live by nginx at:
#   https://downloads.predictatrade.com/windows-agent/
#
set -euo pipefail

# ─── Paths ───
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
AGENT_DIR="$ROOT_DIR/windows-agent"

BIN_CLIENT="$AGENT_DIR/bin/pat-agent.exe"
BIN_MASTER="$AGENT_DIR/bin/pat-master.exe"

DEPLOY_DIR="$AGENT_DIR/deploy"
DEPLOY_CLIENT_DIR="$DEPLOY_DIR/client"
DEPLOY_MASTER_DIR="$DEPLOY_DIR/master"

DEPLOY_CLIENT_EXE="$DEPLOY_CLIENT_DIR/pat-agent.exe"
DEPLOY_MASTER_EXE="$DEPLOY_MASTER_DIR/pat-master.exe"
DEPLOY_ROOT_CLIENT_EXE="$DEPLOY_DIR/pat-agent.exe"
DEPLOY_ROOT_MASTER_EXE="$DEPLOY_DIR/pat-master.exe"

VERSION_FILE="$DEPLOY_DIR/version.txt"
CLIENT_MANIFEST="$DEPLOY_DIR/update-manifest.json"
MASTER_MANIFEST="$DEPLOY_MASTER_DIR/update-manifest.json"
VERSION_GO="$AGENT_DIR/internal/version.go"

# ─── Helpers ───
log()  { echo -e "\033[32m[build-agent]\033[0m $*"; }
err()  { echo -e "\033[31m[build-agent ERROR]\033[0m $*" >&2; }
fatal(){ err "$*"; exit 1; }

# ─── Step 1: Determine version ───
CURRENT_VERSION=$(grep -oP '"[0-9]+\.[0-9]+\.[0-9]+"' "$VERSION_GO" | head -1 | tr -d '"')
log "Current version: v$CURRENT_VERSION"

NEW_VERSION="$CURRENT_VERSION"

if [[ "${1:-}" == "--bump" ]]; then
    IFS='.' read -r MAJOR MINOR PATCH <<< "$CURRENT_VERSION"
    NEW_VERSION="$MAJOR.$MINOR.$((PATCH + 1))"
elif [[ "${1:-}" == "--version" && -n "${2:-}" ]]; then
    NEW_VERSION="$2"
fi

log "Building version: v$NEW_VERSION"

# ─── Step 2: Update version.go if version changed ───
if [[ "$NEW_VERSION" != "$CURRENT_VERSION" ]]; then
    cat > "$VERSION_GO" << EOF
package agent

// AgentVersion is the single source of truth for the agent binary version.
// The installer's version.txt on the server must match this value.
// When pushing a new release: increment this, rebuild both binaries, update deploy/version.txt.
const AgentVersion = "$NEW_VERSION"
EOF
    log "Updated version.go → v$NEW_VERSION"
fi

# ─── Step 3: Build the Windows binaries ───
log "Cross-compiling for Windows amd64..."
cd "$AGENT_DIR"
# Build WITHOUT .syso resource file — the manifest was causing Windows
# App Control to reject the binary.
rm -f "$AGENT_DIR/cmd/client/resource_windows_amd64.syso" "$AGENT_DIR/cmd/master/resource_windows_amd64.syso"

GOTOOLCHAIN=go1.23.0 GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath \
  -ldflags="-s -w -X github.com/predictatrade/windows-agent/internal/agent.AgentVersion=$NEW_VERSION" \
  -o "$BIN_CLIENT" ./cmd/client/ || fatal "Client build failed"

GOTOOLCHAIN=go1.23.0 GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath \
  -ldflags="-s -w -X github.com/predictatrade/windows-agent/internal/agent.AgentVersion=$NEW_VERSION" \
  -o "$BIN_MASTER" ./cmd/master/ || fatal "Master build failed"

log "Client binary built: $BIN_CLIENT ($(du -h "$BIN_CLIENT" | cut -f1))"
log "Master binary built: $BIN_MASTER ($(du -h "$BIN_MASTER" | cut -f1))"

# ─── Step 4: Copy standalone deployment binaries ───
# The Nginx container mounts deploy/ only. A symlink to ../bin is therefore
# broken inside the container and produces a public 404.
mkdir -p "$DEPLOY_CLIENT_DIR" "$DEPLOY_MASTER_DIR"
rm -f "$DEPLOY_CLIENT_EXE" "$DEPLOY_ROOT_CLIENT_EXE" "$DEPLOY_MASTER_EXE" "$DEPLOY_ROOT_MASTER_EXE"
cp "$BIN_CLIENT" "$DEPLOY_CLIENT_EXE"
cp "$BIN_CLIENT" "$DEPLOY_ROOT_CLIENT_EXE"
cp "$BIN_MASTER" "$DEPLOY_MASTER_EXE"
cp "$BIN_MASTER" "$DEPLOY_ROOT_MASTER_EXE"
chmod 0644 "$DEPLOY_CLIENT_EXE" "$DEPLOY_ROOT_CLIENT_EXE" "$DEPLOY_MASTER_EXE" "$DEPLOY_ROOT_MASTER_EXE"
log "Copied deploy binaries (client + master, role subdirs + root)"

# ─── Step 5: Calculate SHA256 checksums ───
CLIENT_CHECKSUM=$(sha256sum "$BIN_CLIENT" | cut -d' ' -f1)
MASTER_CHECKSUM=$(sha256sum "$BIN_MASTER" | cut -d' ' -f1)
log "Client SHA256: $CLIENT_CHECKSUM"
log "Master SHA256: $MASTER_CHECKSUM"

# ─── Step 6: Update version.txt (shared) ───
echo -n "$NEW_VERSION" > "$VERSION_FILE"
log "Updated version.txt → v$NEW_VERSION"

# ─── Step 6b: Update install.ps1 version strings ───
INSTALL_PS1="$DEPLOY_DIR/install.ps1"
if [[ -f "$INSTALL_PS1" ]]; then
    sed -i "s/Installer v[0-9]\+\.[0-9]\+\.[0-9]\+/Installer v$NEW_VERSION/" "$INSTALL_PS1"
    sed -i "s/\$serverVersion = \"[0-9]\+\.[0-9]\+\.[0-9]\+\"/\$serverVersion = \"$NEW_VERSION\"/" "$INSTALL_PS1"
    log "Updated install.ps1 version strings → v$NEW_VERSION"
fi

# ─── Step 7: Update update-manifests ───
TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
cat > "$CLIENT_MANIFEST" << EOF
{
    "version": "$NEW_VERSION",
    "download_url": "https://downloads.predictatrade.com/windows-agent/client/pat-agent.exe",
    "checksum": "$CLIENT_CHECKSUM",
    "min_version": "1.0.0",
    "release_notes": "v$NEW_VERSION — Windows Agent (Client). Distinct client binary (pat-agent.exe) and master binary (pat-master.exe) now shipped separately per role.",
    "timestamp": "$TIMESTAMP"
}
EOF
log "Updated client update-manifest.json"

cat > "$MASTER_MANIFEST" << EOF
{
    "version": "$NEW_VERSION",
    "download_url": "https://downloads.predictatrade.com/windows-agent/master/pat-master.exe",
    "checksum": "$MASTER_CHECKSUM",
    "min_version": "1.0.0",
    "release_notes": "v$NEW_VERSION — Windows Master Node (data-only). Distinct master binary (pat-master.exe).",
    "timestamp": "$TIMESTAMP"
}
EOF
log "Updated master update-manifest.json"

# ─── Step 8: Verify live endpoints ───
log "Verifying live download endpoints..."
for ep in "windows-agent/client/pat-agent.exe" "windows-agent/master/pat-master.exe"; do
    HTTP_CODE=$(curl -sI -o /dev/null -w "%{http_code}" "https://downloads.predictatrade.com/$ep" 2>/dev/null || echo "000")
    if [[ "$HTTP_CODE" == "200" ]]; then
        log "✅ Live endpoint OK ($ep): HTTP 200"
    else
        log "⚠️  Live endpoint $ep returned HTTP $HTTP_CODE (may be expected in dev)"
    fi
done

# ─── Summary ───
echo ""
echo "═══════════════════════════════════════════════"
echo "  Windows Agent Build Complete"
echo "═══════════════════════════════════════════════"
echo "  Version:        v$NEW_VERSION"
echo "  Client binary:  $BIN_CLIENT"
echo "  Master binary:  $BIN_MASTER"
echo "  Client deploy:  $DEPLOY_CLIENT_EXE"
echo "  Master deploy:  $DEPLOY_MASTER_EXE"
echo "  Client checksum:  $CLIENT_CHECKSUM"
echo "  Master checksum:  $MASTER_CHECKSUM"
echo "  Client URL:    https://downloads.predictatrade.com/windows-agent/client/pat-agent.exe"
echo "  Master URL:    https://downloads.predictatrade.com/windows-agent/master/pat-master.exe"
echo "  Install:        irm https://downloads.predictatrade.com/windows-agent/install.ps1 | iex"
echo "═══════════════════════════════════════════════"
