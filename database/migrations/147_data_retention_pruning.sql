-- 147_data_retention_pruning.sql
-- Automated stale-data pruning for NON-timeseries tables, with a scheduled
-- daily cleanup. Scheduled via the already-active TimescaleDB job scheduler
-- (timescaledb.add_job) — pg_cron is NOT used because its shared library is
-- not bundled in this TimescaleDB image (loading it crashes postgres start).
--
-- Design:
--   * system.data_retention_policies  — configurable registry (table, time column,
--                                       retention_days, optional extra WHERE).
--   * system.pruning_runs             — audit log of every run (rows deleted, status).
--   * system.run_retention_policy(p)  — deletes rows older than retention for one policy.
--   * system.run_all_retention_policies() — runs every enabled policy; resilient to
--                                       a single policy failing (logs to pruning_runs).
--   * A daily job 'pat_daily_pruning' calls run_all_retention_policies at 04:00 UTC.
--
-- Only SAFE, non-audit/non-financial/non-compliance tables are seeded. Audit,
-- compliance, and financial ledgers are intentionally excluded (their retention is
-- governed by migrations 064_audit_retention_and_logging.sql and 088_gdpr_erasure_retention.sql).

CREATE SCHEMA IF NOT EXISTS system;

CREATE TABLE IF NOT EXISTS system.data_retention_policies (
  id             SERIAL PRIMARY KEY,
  table_schema   TEXT NOT NULL,
  table_name     TEXT NOT NULL,
  time_column    TEXT NOT NULL,
  retention_days INT  NOT NULL,
  extra_where    TEXT NOT NULL DEFAULT '',
  enabled        BOOLEAN NOT NULL DEFAULT TRUE,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (table_schema, table_name)
);

CREATE TABLE IF NOT EXISTS system.pruning_runs (
  id            BIGSERIAL PRIMARY KEY,
  policy_id     INT REFERENCES system.data_retention_policies(id),
  target_table  TEXT,
  rows_deleted  INT,
  started_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  finished_at   TIMESTAMPTZ,
  status        TEXT,
  error         TEXT
);

CREATE OR REPLACE FUNCTION system.run_retention_policy(p_policy_id INT)
RETURNS INT LANGUAGE plpgsql AS $$
DECLARE
  v_rec RECORD;
  v_sql TEXT;
  v_del INT;
BEGIN
  SELECT * INTO v_rec
  FROM system.data_retention_policies
  WHERE id = p_policy_id AND enabled;
  IF NOT FOUND THEN
    RETURN 0;
  END IF;

  v_sql := format(
    'DELETE FROM %I.%I WHERE %I < now() - (%L || '' days'')::interval',
    v_rec.table_schema, v_rec.table_name, v_rec.time_column, v_rec.retention_days
  );
  IF v_rec.extra_where <> '' THEN
    v_sql := v_sql || ' AND ' || v_rec.extra_where;
  END IF;

  EXECUTE v_sql;
  GET DIAGNOSTICS v_del = ROW_COUNT;

  INSERT INTO system.pruning_runs (policy_id, target_table, rows_deleted, finished_at, status)
  VALUES (p_policy_id, v_rec.table_schema || '.' || v_rec.table_name, v_del, now(), 'ok');
  RETURN v_del;
END $$;

CREATE OR REPLACE FUNCTION system.run_all_retention_policies()
RETURNS INT LANGUAGE plpgsql AS $$
DECLARE
  v_id INT;
  v_total INT := 0;
  v_target TEXT;
BEGIN
  FOR v_id IN SELECT id FROM system.data_retention_policies WHERE enabled LOOP
    BEGIN
      v_total := v_total + system.run_retention_policy(v_id);
    EXCEPTION WHEN OTHERS THEN
      SELECT table_schema || '.' || table_name INTO v_target
      FROM system.data_retention_policies WHERE id = v_id;
      INSERT INTO system.pruning_runs (policy_id, target_table, rows_deleted, finished_at, status, error)
      VALUES (v_id, v_target, 0, now(), 'error', SQLERRM);
    END;
  END LOOP;
  RETURN v_total;
END $$;

-- Seed safe, non-critical pruning targets.
INSERT INTO system.data_retention_policies (table_schema, table_name, time_column, retention_days, extra_where) VALUES
  ('live_preview', 'anonymous_trials',       'trial_expires_at', 7,   ''),
  ('live_preview', 'trial_events',           'occurred_at',       7,   ''),
  ('system',       'notifications',          'created_at',       90,  ''),
  ('trading',      'signal_outbox',          'created_at',       30,  'state = ''PUBLISHED'''),
  ('trading',      'signal_delivery_ledger', 'created_at',       180, '')
ON CONFLICT (table_schema, table_name) DO NOTHING;

-- Schedule a daily run via the TimescaleDB job scheduler.
-- Guarded so re-running this migration is a no-op if the job already exists.
-- (This TimescaleDB version's add_job takes positional args: proc name + interval;
-- named params such as schedule/job_name are not supported here.)
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM timescaledb_information.jobs
    WHERE proc_schema = 'system' AND proc_name = 'run_all_retention_policies'
  ) THEN
    PERFORM add_job('system.run_all_retention_policies', INTERVAL '1 day');
  END IF;
END $$;
