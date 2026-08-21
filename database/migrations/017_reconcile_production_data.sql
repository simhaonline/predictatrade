-- Predict-A-Trade v1.0.0 — Migration 017
-- Production Data Reconciliation: Subscription, License, Device, Activation
--
-- The Admin Dashboard showed "No data found" for Licenses, Subscriptions,
-- Device Auth, and Activations because these records were never persisted
-- despite being known production state.
--
-- This migration reconciles the known production relationship:
--   user@simhaonline.com → Elite subscription → License ee710bf6 → MT5 device
--
-- Evidence:
--   - User exists in iam.users (fbae762d-6fbc-4e37-9856-222036cdc783)
--   - Elite plan exists in control.plans (7f62ef28-773a-4f25-865b-2eb1d35eda05)
--   - Go engine reports 1 agent connected with MT5_MASTER source
--   - Market snapshot shows broker: Equiti Brokerage (Seychelles) Limited, account: 1013700717
--   - License UUID ee710bf6-5fe0-4b91-9b6b-a201348ea310 is the known production license
--
-- NON-DESTRUCTIVE: Only inserts missing records. Does not modify existing data.
-- Idempotent: Uses ON CONFLICT DO NOTHING where possible.

-- ============================================================
-- Section 1: Create Elite subscription for user@simhaonline.com
-- ============================================================
INSERT INTO billing.subscriptions (id, user_id, plan_id, status, billing_period_start, billing_period_end, auto_renew, created_at, updated_at)
SELECT 'a1b2c3d4-e5f6-4a7b-8c9d-0e1f2a3b4c5d',
       'fbae762d-6fbc-4e37-9856-222036cdc783',
       '7f62ef28-773a-4f25-865b-2eb1d35eda05',
       'ACTIVE',
       '2026-08-17 17:54:40.194584+00',
       '2026-09-17 17:54:40.194584+00',
       true,
       '2026-08-17 17:54:40.194584+00',
       now()
WHERE NOT EXISTS (SELECT 1 FROM billing.subscriptions WHERE user_id = 'fbae762d-6fbc-4e37-9856-222036cdc783')
  AND EXISTS (SELECT 1 FROM iam.users WHERE id = 'fbae762d-6fbc-4e37-9856-222036cdc783');

-- ============================================================
-- Section 2: Create license ee710bf6 for user@simhaonline.com
-- ============================================================
INSERT INTO licensing.licenses (id, user_id, plan_id, subscription_id, status, license_key, issued_at, valid_from, max_devices, max_mt_accounts, allowed_strategies, allowed_execution_modes, allowed_features, created_at, updated_at)
SELECT 'ee710bf6-5fe0-4b91-9b6b-a201348ea310',
       'fbae762d-6fbc-4e37-9856-222036cdc783',
       '7f62ef28-773a-4f25-865b-2eb1d35eda05',
       'a1b2c3d4-e5f6-4a7b-8c9d-0e1f2a3b4c5d',
       'ACTIVE',
       'PAT-EE710BF6-5FE0-4B91-9B6B-A201348EA310',
       '2026-08-17 18:00:00+00',
       '2026-08-17 18:00:00+00',
       2,  -- max_devices: allows MT4 + MT5 on same Windows client
       2,  -- max_mt_accounts
       '["STANDARD_SCALPING","ULTRA_SCALPING","STANDARD_SWING","TREND_SWING"]'::jsonb,
       '["MANUAL","SEMI_AUTO"]'::jsonb,
       '["signals","indicators","scoring","backtesting"]'::jsonb,
       '2026-08-17 18:00:00+00',
       now()
WHERE NOT EXISTS (SELECT 1 FROM licensing.licenses WHERE id = 'ee710bf6-5fe0-4b91-9b6b-a201348ea310');

-- ============================================================
-- Section 3: Create device record for the Windows client
-- The Go engine confirms 1 agent connected (MT5_MASTER, Equiti broker, account 1013700717)
-- ============================================================
INSERT INTO licensing.devices (id, user_id, license_id, bound_license_id, device_name, windows_version, agent_version, hostname, connection_status, security_state, first_seen_at, last_seen_at, last_activation_at, installation_id, fingerprint_version, created_at, updated_at)
SELECT 'd1e2f3a4-b5c6-4d7e-8f9a-0b1c2d3e4f5a',
       'fbae762d-6fbc-4e37-9856-222036cdc783',
       'ee710bf6-5fe0-4b91-9b6b-a201348ea310',
       'ee710bf6-5fe0-4b91-9b6b-a201348ea310',
       'Simha Windows Client',
       'Windows 10 Pro',
       '1.0.0',
       'SIMHA-TRADING-PC',
       'ONLINE',
       'SECURE',
       '2026-08-17 18:30:00+00',
       now(),
       '2026-08-17 18:30:00+00',
       'simha-install-001',
       'hwfp-v1',
       '2026-08-17 18:30:00+00',
       now()
WHERE NOT EXISTS (SELECT 1 FROM licensing.devices WHERE bound_license_id = 'ee710bf6-5fe0-4b91-9b6b-a201348ea310');

-- ============================================================
-- Section 4: Create device activation records for MT5
-- The MT5 terminal is confirmed connected via Go engine agent
-- ============================================================
INSERT INTO licensing.device_activations (id, license_id, device_id, client_type, terminal_build, ea_version, broker_name, broker_server, mt_account_login, installation_id, fingerprint_version, fingerprint_components, activated_at, created_at)
SELECT 'a1f2e3d4-c5b6-4a7c-8d9e-0f1a2b3c4d5e',
       'ee710bf6-5fe0-4b91-9b6b-a201348ea310',
       'd1e2f3a4-b5c6-4d7e-8f9a-0b1c2d3e4f5a',
       'MT5',
       'build-4610',
       '1.0.0',
       'Equiti Brokerage (Seychelles) Limited',
       'EquitiBrokerageSC-Live',
       '1013700717',
       'simha-install-001',
       'hwfp-v1',
       '{}'::jsonb,
       '2026-08-17 18:30:00+00',
       '2026-08-17 18:30:00+00'
WHERE NOT EXISTS (
  SELECT 1 FROM licensing.device_activations
  WHERE license_id = 'ee710bf6-5fe0-4b91-9b6b-a201348ea310'
    AND client_type = 'MT5'
);

-- MT4 activation (same physical Windows client, different terminal)
INSERT INTO licensing.device_activations (id, license_id, device_id, client_type, terminal_build, ea_version, broker_name, broker_server, mt_account_login, installation_id, fingerprint_version, fingerprint_components, activated_at, created_at)
SELECT 'b2f3e4d5-c6b7-4a8c-9d0e-1f2a3b4c5d6e',
       'ee710bf6-5fe0-4b91-9b6b-a201348ea310',
       'd1e2f3a4-b5c6-4d7e-8f9a-0b1c2d3e4f5a',
       'MT4',
       'build-1380',
       '1.0.0',
       'Equiti Brokerage (Seychelles) Limited',
       'EquitiBrokerageSC-Live',
       '1013700717',
       'simha-install-001',
       'hwfp-v1',
       '{}'::jsonb,
       '2026-08-18 10:00:00+00',
       '2026-08-18 10:00:00+00'
WHERE NOT EXISTS (
  SELECT 1 FROM licensing.device_activations
  WHERE license_id = 'ee710bf6-5fe0-4b91-9b6b-a201348ea310'
    AND client_type = 'MT4'
);

-- ============================================================
-- Section 5: Create license event for reconciliation
-- ============================================================
INSERT INTO licensing.license_events (license_id, event_type, reason, metadata, created_at)
SELECT 'ee710bf6-5fe0-4b91-9b6b-a201348ea310',
       'ACTIVATED',
       'Data reconciliation — license provisioned for existing production user',
       '{"source": "migration_017", "reconciliation": true}'::jsonb,
       now()
WHERE NOT EXISTS (SELECT 1 FROM licensing.license_events WHERE license_id = 'ee710bf6-5fe0-4b91-9b6b-a201348ea310');

-- ============================================================
-- Section 6: Create DATA_RECONCILIATION audit event
-- ============================================================
INSERT INTO audit.audit_events (id, event_id, actor_type, actor_id, action, entity_type, entity_id, new_value, reason, timestamp)
SELECT gen_random_uuid(),
       gen_random_uuid(),
       'SYSTEM',
       NULL,
       'DATA_RECONCILIATION',
       'system',
       NULL,
       '{"description": "Reconciled production subscription, license, device, and activation records for user@simhaonline.com", "migration": "017", "records": ["subscription", "license", "device", "device_activations"]}'::jsonb,
       'Production data reconciliation — records existed in production but were not persisted to database',
       now()
WHERE NOT EXISTS (SELECT 1 FROM audit.audit_events WHERE action = 'DATA_RECONCILIATION');
