-- Predict-A-Trade v1.0.0 — Migration 116
-- MFA recovery codes: add a column to store single-use recovery codes
-- generated at setup time so the UI can display them after verification.
ALTER TABLE iam.mfa_methods
    ADD COLUMN IF NOT EXISTS recovery_codes TEXT[];

COMMENT ON COLUMN iam.mfa_methods.recovery_codes IS 'Single-use recovery codes generated at MFA setup; displayed once after verification';
