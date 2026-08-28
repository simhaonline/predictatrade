#!/usr/bin/env bash
# Build the Predict-A-Trade Windows Agent (and reference gateway) as native
# Windows executables via cross-compilation. No Windows host required.
#
# Usage:
#   ./scripts/build-windows-agent.sh            # build agent + gateway
#   ./scripts/build-windows-agent.sh --bump     # also bump patch version
#
# Output: dist/pat-windows-agent.exe, dist/pat-gateway.exe
set -euo pipefail
cd "$(dirname "$0")/.."

OUTDIR="dist"
mkdir -p "$OUTDIR"

if [ "${1:-}" = "--bump" ]; then
  echo "bump requested — edit internal/version/version.go Version before tagging"
fi

echo "building pat-windows-agent.exe (cmd/agent) ..."
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o "$OUTDIR/pat-windows-agent.exe" ./cmd/agent

echo "building pat-gateway.exe (cmd/gateway) ..."
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o "$OUTDIR/pat-gateway.exe" ./cmd/gateway

echo "OK -> $OUTDIR/pat-windows-agent.exe  $OUTDIR/pat-gateway.exe"
