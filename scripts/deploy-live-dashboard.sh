#!/usr/bin/env bash
# Deploy the static live dashboard (live.predictatrade.com) to the host web root.
#
# IMPORTANT (2026-08-30): The public live dashboard is served by the HOST nginx
# from /var/www/pat-live (NOT the live-terminal Docker container). Editing
# live-dashboard/index.html has NO effect on the live site until this script runs
# and copies the files into the host web root.
#
# Usage:  scripts/deploy-live-dashboard.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SRC="$ROOT/live-dashboard"
DEST="/var/www/pat-live"

if [ ! -d "$SRC" ]; then echo "source dir $SRC missing"; exit 1; fi
mkdir -p "$DEST"

# Copy all live-dashboard assets into the host web root (extras already present
# in DEST, e.g. whatsapp-qr.* and *.bak, are preserved — cp does not delete).
cp -r "$SRC"/. "$DEST"/

echo "Live dashboard deployed: $SRC -> $DEST"
echo "If nginx config changed, reload with: sudo nginx -s reload"
