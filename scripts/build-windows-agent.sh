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
#   2. Cross-compiles pat-agent.exe for Windows amd64
#   3. Binary goes to windows-agent/bin/pat-agent.exe
#   4. deploy/pat-agent.exe is a real copied file so the Docker/Nginx deploy mount
#      can serve it (symlinks outside the mounted directory are not portable)
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
BIN_PATH="$AGENT_DIR/bin/pat-agent.exe"
DEPLOY_DIR="$AGENT_DIR/deploy"
DEPLOY_EXE="$DEPLOY_DIR/pat-agent.exe"
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
// When pushing a new release: increment this, rebuild pat-agent.exe, update deploy/version.txt.
const AgentVersion = "$NEW_VERSION"
EOF
    log "Updated version.go → v$NEW_VERSION"
fi

# ─── Step 3: Build the Windows binary ───
log "Cross-compiling for Windows amd64..."
cd "$AGENT_DIR"
# Build WITHOUT .syso resource file — the manifest was causing Windows
# App Control to reject the binary. The agent works fine without a manifest
# when managed by NSSM (NSSM handles the service protocol itself).
# Remove the .syso file if it exists to prevent Go from embedding it
rm -f "$AGENT_DIR/cmd/agent/resource_windows_amd64.syso"

GOTOOLCHAIN=go1.23.0 GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath \
  -ldflags="-s -w -X github.com/predictatrade/windows-agent/internal/agent.AgentVersion=$NEW_VERSION" \
  -o "$BIN_PATH" ./cmd/agent/ || fatal "Build failed"
log "Binary built: $BIN_PATH ($(du -h "$BIN_PATH" | cut -f1))"

# ─── Step 3b: Code-sign the binary with Authenticode (osslsigncode) ───
# The repo ships a self-signed Predict-A-Trade code-signing cert
# (certs/pat-code-sign.pfx). Signing the PE makes Windows Defender / SmartScreen
# treat it as a known, signed binary instead of an unsigned download. The
# install.ps1 also imports the matching root (pat-code-sign.crt) into the
# Trusted Root store so the self-signed signature is fully trusted on the box.
# Signing is intentionally non-fatal: if the cert or tool is missing the build
# still succeeds (just unsigned).
CERT_PFX="$AGENT_DIR/certs/pat-code-sign.pfx"
CERT_PASS="pat-local-dev"
if [[ -f "$CERT_PFX" ]] && which osslsigncode >/dev/null 2>&1; then
    log "Code-signing binary with Authenticode..."
    SIGNED_BIN="$BIN_PATH.signed"
    if osslsigncode sign -pkcs12 "$CERT_PFX" -pass "$CERT_PASS" -in "$BIN_PATH" -out "$SIGNED_BIN" 2>&1 | grep -q "Succeeded"; then
        mv "$SIGNED_BIN" "$BIN_PATH"
        log "Binary code-signed with Authenticode (self-signed) ✓"
    else
        log "WARN: Code signing failed — binary will be unsigned"
        rm -f "$SIGNED_BIN"
    fi
else
    log "WARN: No code signing certificate or osslsigncode — binary will be unsigned"
fi

# ─── Step 3c: Publish the PUBLIC code-signing cert ───
# install.ps1 imports this into the Trusted Root store so the self-signed
# Authenticode signature is trusted by Windows Defender / SmartScreen. Only the
# public .crt is published — never the .pfx / .key.
if [[ -f "$AGENT_DIR/certs/pat-code-sign.crt" ]]; then
    cp "$AGENT_DIR/certs/pat-code-sign.crt" "$DEPLOY_DIR/pat-code-sign.crt"
    log "Published public code-signing cert → $DEPLOY_DIR/pat-code-sign.crt"
fi

# ─── Step 4: Copy a standalone deployment binary ───
# The Nginx container mounts deploy/ only. A symlink to ../bin is therefore
# broken inside the container and produces a public 404 for pat-agent.exe.
rm -f "$DEPLOY_EXE"
cp "$BIN_PATH" "$DEPLOY_EXE"
chmod 0644 "$DEPLOY_EXE"
log "Copied standalone deploy binary (client): $DEPLOY_EXE"

# ─── Step 4b: Copy the Master Node binary (same build, distinct filename) ─
# The Master Node runs the IDENTICAL agent but is deployed as pat-master.exe so
# it never collides with the Client Agent (pat-agent.exe) when both roles are
# installed on one machine. Role is chosen at runtime via --mode=data.
DEPLOY_MASTER_EXE="$DEPLOY_DIR/pat-master.exe"
rm -f "$DEPLOY_MASTER_EXE"
cp "$BIN_PATH" "$DEPLOY_MASTER_EXE"
chmod 0644 "$DEPLOY_MASTER_EXE"
log "Copied standalone deploy binary (master): $DEPLOY_MASTER_EXE"

# ─── Step 5: Calculate SHA256 checksum ───
CHECKSUM=$(sha256sum "$BIN_PATH" | cut -d' ' -f1)
log "SHA256: $CHECKSUM"

# ─── Step 6: Update version.txt ───
echo -n "$NEW_VERSION" > "$VERSION_FILE"
log "Updated version.txt → v$NEW_VERSION"

# ─── Step 6b: Update install.ps1 version strings ───
INSTALL_PS1="$DEPLOY_DIR/install.ps1"
if [[ -f "$INSTALL_PS1" ]]; then
    # Update the installer banner version (e.g., "Installer v1.2.21" → "Installer v1.2.32")
    sed -i "s/Installer v[0-9]\+\.[0-9]\+\.[0-9]\+/Installer v$NEW_VERSION/" "$INSTALL_PS1"
    # Update the fallback $serverVersion default (e.g., "1.2.26" → "1.2.32")
    sed -i "s/\$serverVersion = "[0-9]\+\.[0-9]\+\.[0-9]\+"/\$serverVersion = "$NEW_VERSION"/" "$INSTALL_PS1"
    log "Updated install.ps1 version strings → v$NEW_VERSION"
fi

# ─── Step 7: Update update-manifest.json ───
TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
cat > "$MANIFEST_FILE" << EOF
{
    "version": "$NEW_VERSION",
    "download_url": "https://downloads.predictatrade.com/windows-agent/pat-agent.exe",
    "checksum": "$CHECKSUM",
    "min_version": "1.0.0",
    "release_notes": "v$NEW_VERSION — FIX: MT client connection. The agent now discovers and uses the user's real MetaQuotes Common\\Files folder so it meets the EA (which uses MQL FILE_COMMON and never needed to change). Agent auto-removes orphaned IPC files from older versions, so subscribers only need to update the agent — no EA recompile required.",
    "timestamp": "$TIMESTAMP"
}
EOF
log "Updated update-manifest.json"

# ─── Step 8: Verify live endpoint ───
log "Verifying live download endpoint..."
HTTP_CODE=$(curl -sI -o /dev/null -w "%{http_code}" "https://downloads.predictatrade.com/windows-agent/pat-agent.exe" 2>/dev/null || echo "000")
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
echo "  Deploy:      $DEPLOY_EXE (standalone copy)"
echo "  Checksum:    $CHECKSUM"
echo "  Manifest:    $MANIFEST_FILE"
echo "  Live URL:    https://downloads.predictatrade.com/windows-agent/pat-agent.exe"
echo "  Install cmd: irm https://downloads.predictatrade.com/windows-agent/install.ps1 | iex"
echo "═══════════════════════════════════════════════"
# Binary will be UNSIGNED (self-signed causes Windows SmartScreen issues)
