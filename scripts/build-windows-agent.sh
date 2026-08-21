#!/usr/bin/env bash
#
# build-windows-agent.sh — Build Windows Agent and auto-update deploy folder
#
# Usage:
#   ./scripts/build-windows-agent.sh              # Build with current version
#   ./scripts/build-windows-agent.sh --bump       # Build + bump patch version
#   ./scripts/build-windows-agent.sh --version 1.3.0  # Build with specific version
#
# What it does (in order):
#   1. Reads (or sets) the version from windows-agent/internal/version.go
#   2. Cross-compiles agent.exe for Windows amd64
#   3. Binary goes to windows-agent/bin/PredictATrade-Agent.exe
#   4. deploy/agent.exe is a symlink → bin/PredictATrade-Agent.exe (auto-updated)
#   5. Calculates SHA256 checksum of the new binary
#   6. Updates deploy/version.txt
#   7. Updates deploy/update-manifest.json with new version, checksum, timestamp
#   8. Prints a summary
#
# The deploy folder is served live by nginx at:
#   https://downloads.predictatrade.com/windows-agent/
#
set -euo pipefail

# ─── Paths ───
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
AGENT_DIR="$ROOT_DIR/windows-agent"
BIN_PATH="$AGENT_DIR/bin/PredictATrade-Agent.exe"
DEPLOY_DIR="$AGENT_DIR/deploy"
DEPLOY_EXE="$DEPLOY_DIR/agent.exe"
VERSION_FILE="$DEPLOY_DIR/version.txt"
MANIFEST_FILE="$DEPLOY_DIR/update-manifest.json"
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
    # Bump patch version: 1.2.0 → 1.2.1
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
// When pushing a new release: increment this, rebuild agent.exe, update deploy/version.txt.
const AgentVersion = "$NEW_VERSION"
EOF
    log "Updated version.go → v$NEW_VERSION"
fi

# ─── Step 3: Build the Windows binary ───
log "Cross-compiling for Windows amd64..."
cd "$AGENT_DIR"
GOOS=windows GOARCH=amd64 go build -o "$BIN_PATH" ./cmd/agent/ || fatal "Build failed"
log "Binary built: $BIN_PATH ($(du -h "$BIN_PATH" | cut -f1))"

# ─── Step 4: Ensure symlink exists ───
if [[ ! -L "$DEPLOY_EXE" ]]; then
    if [[ -e "$DEPLOY_EXE" ]]; then
        rm -f "$DEPLOY_EXE"
    fi
    ln -s ../bin/PredictATrade-Agent.exe "$DEPLOY_EXE"
    log "Created symlink: deploy/agent.exe → bin/PredictATrade-Agent.exe"
else
    log "Symlink already exists (auto-updated by build)"
fi

# Verify symlink resolves
if [[ ! -f "$DEPLOY_EXE" ]]; then
    fatal "Symlink broken: $DEPLOY_EXE does not resolve to a file"
fi

# ─── Step 5: Calculate SHA256 checksum ───
CHECKSUM=$(sha256sum "$BIN_PATH" | cut -d' ' -f1)
log "SHA256: $CHECKSUM"

# ─── Step 6: Update version.txt ───
echo -n "$NEW_VERSION" > "$VERSION_FILE"
log "Updated version.txt → v$NEW_VERSION"

# ─── Step 7: Update update-manifest.json ───
TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
cat > "$MANIFEST_FILE" << EOF
{
    "version": "$NEW_VERSION",
    "download_url": "https://downloads.predictatrade.com/windows-agent/agent.exe",
    "checksum": "$CHECKSUM",
    "min_version": "1.0.0",
    "release_notes": "v$NEW_VERSION — Auto-built via scripts/build-windows-agent.sh",
    "timestamp": "$TIMESTAMP"
}
EOF
log "Updated update-manifest.json"

# ─── Step 8: Verify live endpoint ───
log "Verifying live download endpoint..."
HTTP_CODE=$(curl -sI -o /dev/null -w "%{http_code}" "https://downloads.predictatrade.com/windows-agent/agent.exe" 2>/dev/null || echo "000")
if [[ "$HTTP_CODE" == "200" ]]; then
    log "✅ Live endpoint OK (HTTP 200)"
else
    log "⚠️  Live endpoint returned HTTP $HTTP_CODE (may be expected in dev)"
fi

# ─── Summary ───
echo ""
echo "═══════════════════════════════════════════════"
echo "  Windows Agent Build Complete"
echo "═══════════════════════════════════════════════"
echo "  Version:     v$NEW_VERSION"
echo "  Binary:      $BIN_PATH"
echo "  Deploy:      $DEPLOY_EXE → $(readlink "$DEPLOY_EXE")"
echo "  Checksum:    $CHECKSUM"
echo "  Manifest:    $MANIFEST_FILE"
echo "  Live URL:    https://downloads.predictatrade.com/windows-agent/agent.exe"
echo "  Install cmd: irm https://downloads.predictatrade.com/windows-agent/install.ps1 | iex"
echo "═══════════════════════════════════════════════"
