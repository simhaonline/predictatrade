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

# ─── Steps 3-7: Build per-role/per-arch, publish binaries + manifests ───
cd "$AGENT_DIR"
# Build WITHOUT .syso resource file — the manifest was causing Windows
# App Control to reject the binary.
rm -f "$AGENT_DIR/cmd/client/resource_windows_amd64.syso" "$AGENT_DIR/cmd/master/resource_windows_amd64.syso"

ARCHES=( amd64 386 arm64 )
TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

write_manifest() {
  local role="$1" arch="$2" exe="$3" sum="$4"
  local dir="$DEPLOY_DIR/$role/$arch"
  mkdir -p "$dir"
  cat > "$dir/update-manifest.json" << EOF
{
    "version": "$NEW_VERSION",
    "download_url": "https://downloads.predictatrade.com/windows-agent/$role/$arch/$exe.exe",
    "checksum": "$sum",
    "min_version": "1.0.0",
    "release_notes": "v$NEW_VERSION — Windows Agent ($role). Multi-arch build ($arch): $exe.exe.",
    "timestamp": "$TIMESTAMP"
}
EOF
}

CLIENT_CHECKSUM_AMD64=""
MASTER_CHECKSUM_AMD64=""

for role in client master; do
  exe=$( [ "$role" = "client" ] && echo "pat-agent" || echo "pat-master" )
  for arch in "${ARCHES[@]}"; do
    BIN="bin/${role}-${arch}.exe"
    log "Cross-compiling $role ($arch)..."
    GOTOOLCHAIN=go1.23.0 GOOS=windows GOARCH=$arch CGO_ENABLED=0 go build -trimpath \
      -ldflags="-s -w -X github.com/predictatrade/windows-agent/internal/agent.AgentVersion=$NEW_VERSION" \
      -o "$BIN" "./cmd/$role/" || fatal "$role/$arch build failed"
    SUM=$(sha256sum "$BIN" | cut -d' ' -f1)
    log "$role/$arch SHA256: $SUM"
    mkdir -p "$DEPLOY_DIR/$role/$arch"
    cp "$BIN" "$DEPLOY_DIR/$role/$arch/$exe.exe"
    chmod 0644 "$DEPLOY_DIR/$role/$arch/$exe.exe"
    write_manifest "$role" "$arch" "$exe" "$SUM"
    if [ "$arch" = "amd64" ]; then
      if [ "$role" = "client" ]; then CLIENT_CHECKSUM_AMD64=$SUM; else MASTER_CHECKSUM_AMD64=$SUM; fi
    fi
  done
  # Role-root (amd64) copy kept for backward-compatibility / simplest URLs
  cp "$DEPLOY_DIR/$role/amd64/$exe.exe" "$DEPLOY_DIR/$role/$exe.exe"
  chmod 0644 "$DEPLOY_DIR/$role/$exe.exe"
done

# Legacy deploy-root copies (older installers / direct links)
cp "$DEPLOY_DIR/client/amd64/pat-agent.exe" "$DEPLOY_ROOT_CLIENT_EXE"
cp "$DEPLOY_DIR/master/amd64/pat-master.exe" "$DEPLOY_ROOT_MASTER_EXE"
chmod 0644 "$DEPLOY_ROOT_CLIENT_EXE" "$DEPLOY_ROOT_MASTER_EXE"

# Role-root manifests (amd64) — what pre-multi-arch agents fetch as fallback
cat > "$CLIENT_MANIFEST" << EOF
{
    "version": "$NEW_VERSION",
    "download_url": "https://downloads.predictatrade.com/windows-agent/client/pat-agent.exe",
    "checksum": "$CLIENT_CHECKSUM_AMD64",
    "min_version": "1.0.0",
    "release_notes": "v$NEW_VERSION — Windows Agent (Client).",
    "timestamp": "$TIMESTAMP"
}
EOF
cat > "$MASTER_MANIFEST" << EOF
{
    "version": "$NEW_VERSION",
    "download_url": "https://downloads.predictatrade.com/windows-agent/master/pat-master.exe",
    "checksum": "$MASTER_CHECKSUM_AMD64",
    "min_version": "1.0.0",
    "release_notes": "v$NEW_VERSION — Windows Master Node (data-only).",
    "timestamp": "$TIMESTAMP"
}
EOF

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

# ─── Step 8: Verify live endpoints ───
log "Verifying live download endpoints..."
for ep in \
    "windows-agent/client/pat-agent.exe" \
    "windows-agent/master/pat-master.exe" \
    "windows-agent/client/amd64/pat-agent.exe" \
    "windows-agent/client/386/pat-agent.exe" \
    "windows-agent/client/arm64/pat-agent.exe" \
    "windows-agent/master/amd64/pat-master.exe" \
    "windows-agent/master/386/pat-master.exe" \
    "windows-agent/master/arm64/pat-master.exe" \
    "windows-agent/client/amd64/update-manifest.json" \
    "windows-agent/master/arm64/update-manifest.json" ; do
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
echo "  Windows Agent Build Complete (multi-arch)"
echo "═══════════════════════════════════════════════"
echo "  Version:        v$NEW_VERSION"
echo "  Roles:          client (pat-agent.exe) / master (pat-master.exe)"
echo "  Architectures:  amd64, 386, arm64"
echo "  Client URL:    https://downloads.predictatrade.com/windows-agent/client/{arch}/pat-agent.exe"
echo "  Master URL:    https://downloads.predictatrade.com/windows-agent/master/{arch}/pat-master.exe"
echo "  Install:        irm https://downloads.predictatrade.com/windows-agent/install.ps1 | iex"
echo "═══════════════════════════════════════════════"
