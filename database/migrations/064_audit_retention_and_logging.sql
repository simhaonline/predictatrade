-- 064: Audit retention policies, compression, and signal_executions fix
-- 
-- 1. Enable TimescaleDB compression on all audit hypertables
-- 2. Add retention policies (configurable via env vars, defaults below)
-- 3. Add missing columns to signal_executions for complete audit trail
-- 4. Add application_version column to pipeline_executions for version traceability

-- === Compression ===
-- Compress chunks older than 7 days to save space while keeping recent data fast
ALTER TABLE audit.pipeline_executions SET (timescaledb.compress, timescaledb.compress_segmentby = 'pipeline_execution_id');
SELECT add_compression_policy('audit.pipeline_executions', INTERVAL '7 days', if_not_exists => TRUE);

ALTER TABLE audit.pipeline_steps SET (timescaledb.compress, timescaledb.compress_segmentby = 'pipeline_execution_id');
SELECT add_compression_policy('audit.pipeline_steps', INTERVAL '7 days', if_not_exists => TRUE);

ALTER TABLE audit.score_executions SET (timescaledb.compress, timescaledb.compress_segmentby = 'score_execution_id');
SELECT add_compression_policy('audit.score_executions', INTERVAL '7 days', if_not_exists => TRUE);

ALTER TABLE audit.score_components SET (timescaledb.compress, timescaledb.compress_segmentby = 'score_execution_id');
SELECT add_compression_policy('audit.score_components', INTERVAL '7 days', if_not_exists => TRUE);

ALTER TABLE audit.signal_executions SET (timescaledb.compress, timescaledb.compress_segmentby = 'signal_id');
SELECT add_compression_policy('audit.signal_executions', INTERVAL '7 days', if_not_exists => TRUE);

ALTER TABLE audit.client_events SET (timescaledb.compress, timescaledb.compress_segmentby = 'user_id');
SELECT add_compression_policy('audit.client_events', INTERVAL '7 days', if_not_exists => TRUE);

-- === Retention policies ===
-- Default: 90 days for audit data. Override with env vars on the server.
-- AUDIT_LOG_RETENTION_DAYS (default 90) — client_events, audit_events
-- PIPELINE_LOG_RETENTION_DAYS (default 90) — pipeline_executions, pipeline_steps
-- SCORE_LOG_RETENTION_DAYS (default 90) — score_executions, score_components
-- SIGNAL_LOG_RETENTION_DAYS (default 365) — signal_executions (longer for trade audit)
-- 
-- Note: TimescaleDB retention policies are set here with defaults.
-- To change, drop and re-add with a different interval.
SELECT add_retention_policy('audit.client_events', INTERVAL '90 days', if_not_exists => TRUE);
SELECT add_retention_policy('audit.pipeline_executions', INTERVAL '90 days', if_not_exists => TRUE);
SELECT add_retention_policy('audit.pipeline_steps', INTERVAL '90 days', if_not_exists => TRUE);
SELECT add_retention_policy('audit.score_executions', INTERVAL '90 days', if_not_exists => TRUE);
SELECT add_retention_policy('audit.score_components', INTERVAL '90 days', if_not_exists => TRUE);
SELECT add_retention_policy('audit.signal_executions', INTERVAL '365 days', if_not_exists => TRUE);

-- === Add application_version to pipeline_executions ===
ALTER TABLE audit.pipeline_executions 
  ADD COLUMN IF NOT EXISTS application_version TEXT;
ALTER TABLE audit.pipeline_executions
  ADD COLUMN IF NOT EXISTS git_commit TEXT;
ALTER TABLE audit.pipeline_executions
  ADD COLUMN IF NOT EXISTS build_id TEXT;

-- === Add git_commit and build_id to signal_executions ===
ALTER TABLE audit.signal_executions
  ADD COLUMN IF NOT EXISTS git_commit TEXT;
ALTER TABLE audit.signal_executions
  ADD COLUMN IF NOT EXISTS build_id TEXT;

-- === Add feature_name to pipeline_steps (for pillar-level detail) ===
ALTER TABLE audit.pipeline_steps
  ADD COLUMN IF NOT EXISTS feature_name TEXT;

-- === Grant permissions ===
GRANT INSERT, SELECT ON ALL TABLES IN SCHEMA audit TO pat_admin;
