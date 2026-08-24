#!/usr/bin/env bash

# ─── B-07: Duplicate sequence number check ───
duplicates=$(ls database/migrations/*.sql 2>/dev/null | sed 's/.*\/\([0-9]*\)_.*/\1/' | sort | uniq -d)
if [ -n "$duplicates" ]; then
    echo "WARNING: Duplicate migration sequence numbers found:"
    for d in $duplicates; do
        echo "  $d: $(ls database/migrations/${d}_*.sql 2>/dev/null)"
    done
    echo "Existing duplicates are already applied and cannot be renamed."
    echo "New migrations must use unique sequence numbers."
fi

# Predict-A-Trade v1.0.0 — Canonical Migration Runner
# SOW Section 60: One authoritative migration sequence.

set -euo pipefail

DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-predictatrade}"
DB_USER="${DB_USER:-pat_admin}"
DB_PASSWORD="${DB_PASSWORD:-pat_local_dev_only}"
MIGRATIONS_DIR="${MIGRATIONS_DIR:-database/migrations}"
SEEDS_DIR="${SEEDS_DIR:-database/seeds}"

export PGPASSWORD="$DB_PASSWORD"

usage() {
    echo "Usage: $0 {up|down|seed|test|status}"
    echo "  up    - Run all pending forward migrations"
    echo "  down  - Rollback last migration (not implemented in v1.0.0 — use PITR)"
    echo "  seed  - Run seed files for initial configuration"
    echo "  test  - Run migration tests"
    echo "  status- Show migration status"
    exit 1
}

# Check if psql is available
check_psql() {
    if ! command -v psql &>/dev/null; then
        # Try docker
        if docker exec pat-postgres psql --version &>/dev/null 2>&1; then
            PSQL_CMD="docker exec -i pat-postgres psql"
            return 0
        fi
        echo "ERROR: psql not found and no Docker postgres container running."
        echo "Run 'make infra-up' first."
        exit 1
    fi
    PSQL_CMD="psql"
}

run_migration() {
    local file="$1"
    local name=$(basename "$file")
    echo "Running migration: $name"

    # Record start
    $PSQL_CMD -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -v ON_ERROR_STOP=1 -c "
        INSERT INTO audit.migration_history (filename, status, started_at)
        VALUES ('$name', 'RUNNING', now())
        ON CONFLICT (filename) WHERE status = 'COMPLETED' DO NOTHING
    " 2>/dev/null || true

    if $PSQL_CMD -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -v ON_ERROR_STOP=1 -f "$file"; then
        $PSQL_CMD -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -v ON_ERROR_STOP=1 -c "
            INSERT INTO audit.migration_history (filename, status, completed_at)
            VALUES ('$name', 'COMPLETED', now())
            ON CONFLICT (filename) DO UPDATE SET status = 'COMPLETED', completed_at = now()
        " 2>/dev/null || true
        echo "  ✓ $name applied successfully"
    else
        $PSQL_CMD -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -v ON_ERROR_STOP=1 -c "
            INSERT INTO audit.migration_history (filename, status, completed_at)
            VALUES ('$name', 'FAILED', now())
            ON CONFLICT (filename) DO UPDATE SET status = 'FAILED', completed_at = now()
        " 2>/dev/null || true
        echo "  ✗ $name FAILED"
        exit 1
    fi
}

case "${1:-}" in
    up)
        check_psql
        # Create migration history table if not exists
        $PSQL_CMD -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -v ON_ERROR_STOP=1 -c "
            CREATE SCHEMA IF NOT EXISTS audit;
            CREATE TABLE IF NOT EXISTS audit.migration_history (
                id SERIAL PRIMARY KEY,
                filename VARCHAR(255) NOT NULL UNIQUE,
                status VARCHAR(20) NOT NULL,
                started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
                completed_at TIMESTAMPTZ
            );
        " 2>/dev/null || true

        # Run migrations in order
        for file in $(ls "$MIGRATIONS_DIR"/*.sql 2>/dev/null | sort); do
            name=$(basename "$file")
            # Check if already completed
            already=$($PSQL_CMD -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -tAc "
                SELECT COUNT(*) FROM audit.migration_history
                WHERE filename = '$name' AND status = 'COMPLETED'
            " 2>/dev/null || echo "0")
            if [ "$already" -eq 0 ]; then
                run_migration "$file"
            else
                echo "Skipping (already applied): $name"
            fi
        done
        echo "All migrations complete."
        ;;

    seed)
        check_psql
        for file in $(ls "$SEEDS_DIR"/*.sql 2>/dev/null | sort); do
            echo "Running seed: $(basename "$file")"
            $PSQL_CMD -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -v ON_ERROR_STOP=1 -f "$file"
        done
        echo "Seeding complete."
        ;;

    test)
        check_psql
        echo "Running migration tests..."
        $PSQL_CMD -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -v ON_ERROR_STOP=1 -c "
            -- Verify all schemas exist
            SELECT 'Schemas check' AS test,
                   COUNT(*) AS schemas_count
            FROM information_schema.schemata
            WHERE schema_name IN ('iam','control','licensing','billing','referral','finance','trading','market','research','audit','support');

            -- Verify key tables exist
            SELECT 'Tables check' AS test,
                   COUNT(*) AS tables_count
            FROM information_schema.tables
            WHERE table_schema IN ('iam','control','licensing','billing','referral','trading','market','research','audit','support');

            -- Verify plans seeded
            SELECT 'Plans check' AS test,
                   COUNT(*) AS plans_count
            FROM control.plans;

            -- Verify commission rules seeded
            SELECT 'Commission rules check' AS test,
                   COUNT(*) AS rules_count
            FROM referral.commission_rules;

            -- Verify purchase rules seeded
            SELECT 'Purchase rules check' AS test,
                   COUNT(*) AS purchase_rules_count
            FROM referral.purchase_commission_rules;
        "
        echo "Migration tests complete."
        ;;

    status)
        check_psql
        $PSQL_CMD -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "
            SELECT filename, status, started_at, completed_at
            FROM audit.migration_history
            ORDER BY id;
        "
        ;;

    *)
        usage
        ;;
esac
