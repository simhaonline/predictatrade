#!/usr/bin/env bash
# check_migrations.sh — CI guard (DB-6)
#
# FAILS (exit 1) when ANY of the following are true:
#   (a) any numeric migration prefix is duplicated in database/migrations/*.sql
#       (duplicate prefixes are a tolerated LEGACY issue until a renumber
#        maintenance window; this guard blocks NEW duplicates only by failing
#        loudly — see MIGRATION_ORDER.md).
#   (b) audit.migration_history filenames != on-disk filenames
#       (drift between bookkeeping and reality).
#
# Prints clear, actionable errors. Designed to run in CI.
#
set -uo pipefail

DB_HOST="${DB_HOST:-127.0.0.1}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-predictatrade}"
DB_USER="${DB_USER:-pat_admin}"
DB_PASSWORD="${DB_PASSWORD:-pat_local_dev_only}"
MIGRATIONS_DIR="${MIGRATIONS_DIR:-database/migrations}"
export PGPASSWORD="$DB_PASSWORD"
PSQL="psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -t -A"

rc=0

echo "== (a) Duplicate numeric migration prefixes =="
prefixes=$(ls "$MIGRATIONS_DIR"/*.sql 2>/dev/null | xargs -n1 basename | sed -E 's/^([0-9]+)_.*/\1/' | sort | uniq -d || true)
if [ -n "$prefixes" ]; then
  echo "FAIL (DB-6a): duplicate migration sequence prefixes detected:"
  for p in $prefixes; do
    files=$(ls "$MIGRATIONS_DIR"/${p}_*.sql 2>/dev/null | xargs -n1 basename | tr '\n' ' ')
    echo "  prefix $p -> $files"
  done
  echo "  ACTION: renumber these in a future maintenance window (MIGRATION_ORDER.md)."
  echo "          No NEW duplicate prefixes may be introduced."
  rc=1
else
  echo "OK: no duplicate migration prefixes."
fi

echo ""
echo "== (b) History filenames vs on-disk filenames =="
disk_set=$(ls "$MIGRATIONS_DIR"/*.sql 2>/dev/null | xargs -n1 basename | sort || true)
hist_set=$(echo "SELECT filename FROM audit.migration_history;" | $PSQL 2>/dev/null | sort || true)
missing=$(comm -23 <(printf '%s\n' "$disk_set") <(printf '%s\n' "$hist_set") || true)
orphans=$(comm -13 <(printf '%s\n' "$disk_set") <(printf '%s\n' "$hist_set") || true)

if [ -n "$missing" ] || [ -n "$orphans" ]; then
  echo "FAIL (DB-6b): audit.migration_history does not match on-disk migration set."
  if [ -n "$missing" ]; then
    echo "  on disk but NOT in history ($(printf '%s\n' "$missing" | grep -c .)):"
    printf '%s\n' "$missing" | sed 's/^/    + /'
  fi
  if [ -n "$orphans" ]; then
    echo "  in history but NOT on disk ($(printf '%s\n' "$orphans" | grep -c .)):"
    printf '%s\n' "$orphans" | sed 's/^/    - /'
  fi
  echo "  ACTION: run scripts/reconcile_migrations.sh --apply to resync."
  rc=1
else
  echo "OK: history matches disk ($(printf '%s\n' "$disk_set" | grep -c .) files)."
fi

echo ""
if [ $rc -ne 0 ]; then
  echo "check_migrations.sh: FAILED (exit 1)"
  exit 1
fi
echo "check_migrations.sh: PASSED"
exit 0
