-- Predict-A-Trade v1.0.0 — Migration 001
-- Create logical schemas and least-privilege database roles
-- SOW Sections 55, 60, 61, 62, 63, 63A, 140.1

-- ============================================================
-- Schemas (SOW Section 55)
-- ============================================================
CREATE SCHEMA IF NOT EXISTS iam;
CREATE SCHEMA IF NOT EXISTS control;
CREATE SCHEMA IF NOT EXISTS licensing;
CREATE SCHEMA IF NOT EXISTS billing;
CREATE SCHEMA IF NOT EXISTS referral;
CREATE SCHEMA IF NOT EXISTS finance;
CREATE SCHEMA IF NOT EXISTS trading;
CREATE SCHEMA IF NOT EXISTS market;
CREATE SCHEMA IF NOT EXISTS research;
CREATE SCHEMA IF NOT EXISTS audit;
CREATE SCHEMA IF NOT EXISTS support;

-- ============================================================
-- Extensions (SOW Sections 55, 57, 58)
-- ============================================================
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS IF pg_version_num >= 170000 THEN "timescaledb" ELSE NULL END IF;
CREATE EXTENSION IF NOT EXISTS IF pg_version_num >= 170000 THEN "vector" ELSE NULL END IF;

-- ============================================================
-- Least-Privilege Roles (SOW Section 55)
-- ============================================================
-- Note: In production these are created by the DBA with password auth.
-- For local/dev we create them without passwords (peer/trust auth).

DO $$
BEGIN
    -- Migration role (can run DDL)
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'pat_migration') THEN
        CREATE ROLE pat_migration LOGIN;
    END IF;

    -- NestJS control plane (read/write to iam, control, licensing, billing, referral, finance, audit, support)
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'nest_control') THEN
        CREATE ROLE nest_control LOGIN;
    END IF;

    -- Billing worker (limited to billing + referral + finance)
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'billing_worker') THEN
        CREATE ROLE billing_worker LOGIN;
    END IF;

    -- Commission worker
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'commission_worker') THEN
        CREATE ROLE commission_worker LOGIN;
    END IF;

    -- Payout worker
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'payout_worker') THEN
        CREATE ROLE payout_worker LOGIN;
    END IF;

    -- Go real-time (read config, write trading/market data, limited control read)
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'go_realtime') THEN
        CREATE ROLE go_realtime LOGIN;
    END IF;

    -- Python research (read-only on most, read/write on research)
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'python_research') THEN
        CREATE ROLE python_research LOGIN;
    END IF;

    -- Read-only analytics
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'readonly_analytics') THEN
        CREATE ROLE readonly_analytics LOGIN;
    END IF;

    -- Audit reader
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'audit_reader') THEN
        CREATE ROLE audit_reader LOGIN;
    END IF;

    -- Backup
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'pat_backup') THEN
        CREATE ROLE pat_backup LOGIN;
    END IF;
END
$$;

-- Grant schema usage to roles
GRANT USAGE ON SCHEMA iam TO nest_control, go_realtime, readonly_analytics, audit_reader, pat_backup;
GRANT USAGE ON SCHEMA control TO nest_control, go_realtime, readonly_analytics, audit_reader, pat_backup;
GRANT USAGE ON SCHEMA licensing TO nest_control, go_realtime, readonly_analytics, audit_reader, pat_backup;
GRANT USAGE ON SCHEMA billing TO nest_control, billing_worker, commission_worker, payout_worker, readonly_analytics, audit_reader, pat_backup;
GRANT USAGE ON SCHEMA referral TO nest_control, billing_worker, commission_worker, payout_worker, readonly_analytics, audit_reader, pat_backup;
GRANT USAGE ON SCHEMA finance TO nest_control, billing_worker, commission_worker, payout_worker, readonly_analytics, audit_reader, pat_backup;
GRANT USAGE ON SCHEMA trading TO nest_control, go_realtime, readonly_analytics, audit_reader, pat_backup;
GRANT USAGE ON SCHEMA market TO nest_control, go_realtime, python_research, readonly_analytics, pat_backup;
GRANT USAGE ON SCHEMA research TO python_research, readonly_analytics, pat_backup;
GRANT USAGE ON SCHEMA audit TO nest_control, audit_reader, pat_backup;
GRANT USAGE ON SCHEMA support TO nest_control, readonly_analytics, pat_backup;

-- ============================================================
-- Default privileges
-- ============================================================
ALTER DEFAULT PRIVILEGES IN SCHEMA iam GRANT SELECT, INSERT, UPDATE, DELETE TO nest_control;
ALTER DEFAULT PRIVILEGES IN SCHEMA control GRANT SELECT, INSERT, UPDATE, DELETE TO nest_control;
ALTER DEFAULT PRIVILEGES IN SCHEMA licensing GRANT SELECT, INSERT, UPDATE, DELETE TO nest_control;
ALTER DEFAULT PRIVILEGES IN SCHEMA billing GRANT SELECT, INSERT, UPDATE, DELETE TO nest_control, billing_worker;
ALTER DEFAULT PRIVILEGES IN SCHEMA referral GRANT SELECT, INSERT, UPDATE, DELETE TO nest_control, commission_worker;
ALTER DEFAULT PRIVILEGES IN SCHEMA finance GRANT SELECT, INSERT, UPDATE, DELETE TO nest_control, payout_worker;
ALTER DEFAULT PRIVILEGES IN SCHEMA trading GRANT SELECT, INSERT, UPDATE TO go_realtime;
ALTER DEFAULT PRIVILEGES IN SCHEMA trading GRANT SELECT TO nest_control, readonly_analytics;
ALTER DEFAULT PRIVILEGES IN SCHEMA market GRANT SELECT, INSERT, UPDATE TO go_realtime;
ALTER DEFAULT PRIVILEGES IN SCHEMA market GRANT SELECT TO nest_control, python_research, readonly_analytics;
ALTER DEFAULT PRIVILEGES IN SCHEMA research GRANT SELECT, INSERT, UPDATE, DELETE TO python_research;
ALTER DEFAULT PRIVILEGES IN SCHEMA research GRANT SELECT TO readonly_analytics;
ALTER DEFAULT PRIVILEGES IN SCHEMA audit GRANT SELECT TO audit_reader, nest_control;
ALTER DEFAULT PRIVILEGES IN SCHEMA support GRANT SELECT, INSERT, UPDATE, DELETE TO nest_control;
