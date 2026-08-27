-- Migration 029: Remove default user, create plan-based test users
-- Creates 4 users (Free, Standard, Pro, Elite) with subscriptions and licenses
-- Password for all users: Demo@1234 (bcrypt hash below)
--
-- SEC-3 (HIGH): these are seeded test accounts with a KNOWN password. They must
-- NEVER be created in a production environment. The whole block below only runs
-- when BOTH conditions hold:
--   1. the `app.env` setting is not 'production' (when set), AND
--   2. the connected database name does not contain 'prod'.
-- If either indicates production, the INSERTs/UPDATEs are skipped entirely.

DO $$
BEGIN
  IF current_setting('app.env', true) IS DISTINCT FROM 'production'
     AND current_database() NOT LIKE '%prod%' THEN

    -- 1. Soft-delete default user@predictatrade.com
    UPDATE iam.users SET deleted_at = now(), status = 'DELETED' WHERE email = 'user@predictatrade.com';
    UPDATE licensing.licenses SET status = 'REVOKED', revoked_at = now(), revocation_reason = 'Default user removed — replaced with plan-based test users' 
      WHERE user_id = 'fbae762d-6fbc-4e37-9856-222036cdc783';
    UPDATE billing.subscriptions SET status = 'CANCELLED', cancelled_at = now(), cancel_reason = 'Default user removed' 
      WHERE user_id = 'fbae762d-6fbc-4e37-9856-222036cdc783';
    DELETE FROM iam.memberships WHERE user_id = 'fbae762d-6fbc-4e37-9856-222036cdc783';

    -- 2. Create 4 new users (password: Demo@1234)
    INSERT INTO iam.users (id, email, email_verified, full_name, password_hash, status, created_at, updated_at) VALUES
      ('a1b2c3d4-0001-4000-8000-000000000001', 'free@predictatrade.com',     true, 'Free Plan User',     '$2b$12$VwI2OdbqJsbe0x0szgvXTuVeU8kvbEo11j23.qxVg7leIC4X2NeNu', 'ACTIVE', now(), now()),
      ('a1b2c3d4-0002-4000-8000-000000000002', 'standard@predictatrade.com', true, 'Standard Plan User', '$2b$12$VwI2OdbqJsbe0x0szgvXTuVeU8kvbEo11j23.qxVg7leIC4X2NeNu', 'ACTIVE', now(), now()),
      ('a1b2c3d4-0003-4000-8000-000000000003', 'pro@predictatrade.com',     true, 'Pro Plan User',     '$2b$12$VwI2OdbqJsbe0x0szgvXTuVeU8kvbEo11j23.qxVg7leIC4X2NeNu', 'ACTIVE', now(), now()),
      ('a1b2c3d4-0004-4000-8000-000000000004', 'elite@predictatrade.com',   true, 'Elite Plan User',   '$2b$12$VwI2OdbqJsbe0x0szgvXTuVeU8kvbEo11j23.qxVg7leIC4X2NeNu', 'ACTIVE', now(), now())
    ON CONFLICT (email) DO NOTHING;

    -- 3. Assign USER role (org: 4ca19b6a-7a24-4eef-9d34-950483be4ed5, role: 57d09bce-...)
    INSERT INTO iam.memberships (id, user_id, organization_id, role_id, created_at) VALUES
      (gen_random_uuid(), 'a1b2c3d4-0001-4000-8000-000000000001', '4ca19b6a-7a24-4eef-9d34-950483be4ed5', '57d09bce-4740-4bb1-bb60-0b25206231ce', now()),
      (gen_random_uuid(), 'a1b2c3d4-0002-4000-8000-000000000002', '4ca19b6a-7a24-4eef-9d34-950483be4ed5', '57d09bce-4740-4bb1-bb60-0b25206231ce', now()),
      (gen_random_uuid(), 'a1b2c3d4-0003-4000-8000-000000000003', '4ca19b6a-7a24-4eef-9d34-950483be4ed5', '57d09bce-4740-4bb1-bb60-0b25206231ce', now()),
      (gen_random_uuid(), 'a1b2c3d4-0004-4000-8000-000000000004', '4ca19b6a-7a24-4eef-9d34-950483be4ed5', '57d09bce-4740-4bb1-bb60-0b25206231ce', now())
    ON CONFLICT DO NOTHING;

    -- 4. Create subscriptions (FREE=8b4ab1a7, STANDARD=3ef7b1eb, PRO=38413df8, ELITE=7f62ef28)
    INSERT INTO billing.subscriptions (id, user_id, plan_id, status, billing_period_start, billing_period_end, next_billing_date, auto_renew, created_at, updated_at) VALUES
      ('b1b2c3d4-0001-4000-8000-000000000001', 'a1b2c3d4-0001-4000-8000-000000000001', '8b4ab1a7-06e4-48ad-bd36-98902c026d98', 'ACTIVE', now(), now() + interval '30 days', now() + interval '30 days', false, now(), now()),
      ('b1b2c3d4-0002-4000-8000-000000000002', 'a1b2c3d4-0002-4000-8000-000000000002', '3ef7b1eb-37fe-4aae-84a7-3695b9a9d991', 'ACTIVE', now(), now() + interval '30 days', now() + interval '30 days', false, now(), now()),
      ('b1b2c3d4-0003-4000-8000-000000000003', 'a1b2c3d4-0003-4000-8000-000000000003', '38413df8-2109-4566-b293-c915805cdee4', 'ACTIVE', now(), now() + interval '30 days', now() + interval '30 days', false, now(), now()),
      ('b1b2c3d4-0004-4000-8000-000000000004', 'a1b2c3d4-0004-4000-8000-000000000004', '7f62ef28-773a-4f25-865b-2eb1d35eda05', 'ACTIVE', now(), now() + interval '30 days', now() + interval '30 days', false, now(), now())
    ON CONFLICT DO NOTHING;

    -- 5. Create licenses with subscriber keys
    INSERT INTO licensing.licenses (id, user_id, plan_id, subscription_id, status, license_key, issued_at, valid_from, max_devices, max_mt_accounts, allowed_strategies, allowed_execution_modes, created_by, created_at, updated_at) VALUES
      ('c1b2c3d4-0001-4000-8000-000000000001', 'a1b2c3d4-0001-4000-8000-000000000001', '8b4ab1a7-06e4-48ad-bd36-98902c026d98', 'b1b2c3d4-0001-4000-8000-000000000001', 'ACTIVE', 'PAT-A1B2C3D4-0001-4000-8000-000000000001', now(), now(), 1, 1, '["STANDARD_SCALPING"]'::jsonb, '["SIGNAL_ONLY"]'::jsonb, '6d3c51bb-5f91-4494-bdc7-a0b52a572b92', now(), now()),
      ('c1b2c3d4-0002-4000-8000-000000000002', 'a1b2c3d4-0002-4000-8000-000000000002', '3ef7b1eb-37fe-4aae-84a7-3695b9a9d991', 'b1b2c3d4-0002-4000-8000-000000000002', 'ACTIVE', 'PAT-A1B2C3D4-0002-4000-8000-000000000002', now(), now(), 2, 2, '["STANDARD_SCALPING","STANDARD_SWING"]'::jsonb, '["SIGNAL_ONLY","MANUAL"]'::jsonb, '6d3c51bb-5f91-4494-bdc7-a0b52a572b92', now(), now()),
      ('c1b2c3d4-0003-4000-8000-8000-000000000003', 'a1b2c3d4-0003-4000-8000-000000000003', '38413df8-2109-4566-b293-c915805cdee4', 'b1b2c3d4-0003-4000-8000-000000000003', 'ACTIVE', 'PAT-A1B2C3D4-0003-4000-8000-000000000003', now(), now(), 3, 3, '["STANDARD_SCALPING","ULTRA_SCALPING","STANDARD_SWING","TREND_SWING"]'::jsonb, '["SIGNAL_ONLY","MANUAL","SEMI_AUTO"]'::jsonb, '6d3c51bb-5f91-4494-bdc7-a0b52a572b92', now(), now()),
      ('c1b2c3d4-0004-4000-8000-000000000004', 'a1b2c3d4-0004-4000-8000-000000000004', '7f62ef28-773a-4f25-865b-2eb1d35eda05', 'b1b2c3d4-0004-4000-8000-000000000004', 'ACTIVE', 'PAT-A1B2C3D4-0004-4000-8000-000000000004', now(), now(), 5, 5, '["STANDARD_SCALPING","ULTRA_SCALPING","STANDARD_SWING","TREND_SWING","MARNIE_FIB"]'::jsonb, '["SIGNAL_ONLY","MANUAL","SEMI_AUTO","FULL_AUTO"]'::jsonb, '6d3c51bb-5f91-4494-bdc7-a0b52a572b92', now(), now())
    ON CONFLICT DO NOTHING;

    -- 6. Log license events
    INSERT INTO licensing.license_events (license_id, event_type, reason, metadata, created_at) VALUES
      ('c1b2c3d4-0001-4000-8000-000000000001', 'ASSIGNED', 'Plan-based test user created', '{"plan":"FREE"}'::jsonb, now()),
      ('c1b2c3d4-0002-4000-8000-000000000002', 'ASSIGNED', 'Plan-based test user created', '{"plan":"STANDARD"}'::jsonb, now()),
      ('c1b2c3d4-0003-4000-8000-000000000003', 'ASSIGNED', 'Plan-based test user created', '{"plan":"PRO"}'::jsonb, now()),
      ('c1b2c3d4-0004-4000-8000-000000000004', 'ASSIGNED', 'Plan-based test user created', '{"plan":"ELITE"}'::jsonb, now())
    ON CONFLICT DO NOTHING;

  END IF;
END
$$;
