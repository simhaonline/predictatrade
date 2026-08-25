-- Reproducible DDL for finance.ledger_entries.
-- VERIFIED: this table already exists in the live database, but was not present
-- in any prior migration. Without it, a fresh `migrate.sh` would not recreate the
-- table and payout completion (which writes ledger entries) would fail on new
-- deployments. Created idempotently so it is a no-op on the running system.
CREATE TABLE IF NOT EXISTS finance.ledger_entries (
    id                uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    account_user_id   uuid        NOT NULL REFERENCES iam.users(id) ON DELETE CASCADE,
    entry_type        varchar(64) NOT NULL,
    direction         varchar(8)  NOT NULL CHECK (direction IN ('CREDIT','DEBIT')),
    amount            numeric(18,8) NOT NULL CHECK (amount > 0),
    currency          varchar(8)  NOT NULL DEFAULT 'USD',
    source_type       varchar(64) NOT NULL,
    source_id         uuid,
    idempotency_key   varchar(255) NOT NULL UNIQUE,
    metadata          jsonb       DEFAULT '{}'::jsonb,
    created_at        timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_ledger_entries_account_user
    ON finance.ledger_entries (account_user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ledger_entries_source
    ON finance.ledger_entries (source_type, source_id);
