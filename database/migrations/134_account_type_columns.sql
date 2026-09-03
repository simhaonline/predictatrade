-- Migration 134: account-type persistence columns for edge devices
-- Companion to mig 133 (account_types/strategy_parameters reference tables).
-- The fleet EAs tag every heartbeat with account_type/account_type_verified;
-- edge-poll.service.ts persists them here. Fail-open additive columns.

ALTER TABLE licensing.edge_device_state
    ADD COLUMN IF NOT EXISTS account_type          VARCHAR(50),
    ADD COLUMN IF NOT EXISTS account_type_verified BOOLEAN;

ALTER TABLE licensing.devices
    ADD COLUMN IF NOT EXISTS account_type          VARCHAR(50);

-- Audit trail row (audit.migration_history tracks filename + status)
INSERT INTO audit.migration_history (filename, status, completed_at)
VALUES ('134_account_type_columns.sql', 'APPLIED', now());