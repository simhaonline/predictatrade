#!/usr/bin/env bash
# Predict-A-Trade — Disaster Recovery / Operations Kit (unified entry point)
#
# Thin, safe orchestrator over the existing, battle-tested backup / restore /
# migration / health scripts. It does NOT reimplement them — it wires them into
# one predictable command surface so future backup/restore/migration is a single
# call instead of four disparate scripts.
#
# Usage:
#   ./scripts/dr-kit.sh backup           # logical pg_dump + verify (scripts/backup/backup.sh)
#   ./scripts/dr-kit.sh offhost          # sync latest dump to S3/NFS (scripts/backup/offhost_backup.sh)
#   ./scripts/dr-kit.sh restore-test     # restore latest dump into disposable test DB + validate
#   ./scripts/dr-kit.sh migrate-up       # run pending forward migrations
#   ./scripts/dr-kit.sh migrate-status   # show applied/pending migrations
#   ./scripts/dr-kit.sh migrate-test     # run migration self-tests
#   ./scripts/dr-kit.sh verify-s3       # pre-flight check of S3 backup config (no secrets printed)
#   ./scripts/dr-kit.sh codebase        # SEPARATE code+config snapshot -> R2 predictatrade/code/
#   ./scripts/dr-kit.sh health          # production + Cloudflare edge health checks
#   ./scripts/dr-kit.sh all             # backup + codebase + restore-test + migrate-status + health
#
# Every subcommand fails fast and prints what it ran. No destructive action runs
# against the live database except 'backup' (read-only pg_dump) and 'migrate-up'
# (intentional schema change). 'restore-test' always targets a throwaway DB.

set -uo pipefail

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${PROJECT_ROOT}"

log()  { echo "[$(date -u '+%Y-%m-%dT%H:%M:%SZ')] $*"; }
ok()   { echo "✅ $*"; }
err()  { echo "❌ $*" >&2; }

run() {
  log "▶ $*"
  if "$@"; then ok "exit 0 — $1"; else err "exit $? — $1"; return 1; fi
}

case "${1:-help}" in
  backup)
    run bash scripts/backup/backup.sh
    ;;
  offhost)
    run bash scripts/backup/offhost_backup.sh
    ;;
  restore-test)
    # Ensure a dump exists; if not, make one first (read-only backup).
    if ! ls -t /var/backups/predictatrade/backup_*.dump >/dev/null 2>&1; then
      log "no local dump found — running backup first"
      run bash scripts/backup/backup.sh
    fi
    run bash scripts/backup/restore_test.sh
    ;;
  migrate-up)
    run bash scripts/migrate.sh up
    ;;
  migrate-status)
    run bash scripts/migrate.sh status
    ;;
  migrate-test)
    run bash scripts/migrate.sh test
    ;;
  verify-s3)
    run bash scripts/backup/verify_s3.sh
    ;;
  codebase)
    run bash scripts/backup/codebase_backup.sh
    ;;
  health)
    run bash scripts/verify_live_production.sh
    # Cloudflare edge check is advisory (non-fatal).
    if [ -x scripts/verify-cloudflare.sh ]; then
      log "▶ scripts/verify-cloudflare.sh (advisory)"
      bash scripts/verify-cloudflare.sh || log "Cloudflare edge check reported issues (non-blocking)"
    fi
    ;;
  all)
    run bash scripts/backup/backup.sh && \
    run bash scripts/backup/codebase_backup.sh && \
    run bash scripts/backup/restore_test.sh && \
    run bash scripts/migrate.sh status && \
    run bash scripts/verify_live_production.sh
    ;;
  help|-h|--help)
    sed -n '2,28p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//'
    ;;
  *)
    err "unknown subcommand: '$1'"; sed -n '2,28p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//'; exit 2
    ;;
esac
