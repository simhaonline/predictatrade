-- =====================================================================
-- Migration 119: add role to licensing.devices (Option B, v1.19.0)
-- =====================================================================
-- WHY: with the Windows-agent transport gone (v1.19.0), the role of a
-- device ('data' = Master market-data feeder | 'exec' = Client executor)
-- exists ONLY in the device JWT minted at activation. The refresh grant
-- (POST /api/v1/devices/refresh) receives no role from the EAs and had no
-- persisted column to read, so it defaulted to 'exec' — a refreshed Master
-- token would carry role='exec' and the engine would silently DROP its
-- MARKET_SNAPSHOT messages (IsDataNode()==false). Persisting the role at
-- activation lets refresh() mint data-role JWTs for Master devices.
--
-- Backfill: the two Option B devices are the admin's Master (activated
-- first, role 'data' per its activation body) + Client ('exec'). No
-- production rows beyond these; column is nullable for safety.
-- =====================================================================

ALTER TABLE licensing.devices
  ADD COLUMN IF NOT EXISTS role TEXT;

COMMENT ON COLUMN licensing.devices.role IS
  'Device transport role: data (Master market-data feeder) | exec (Client executor). Set at activation; consumed by refresh() when minting ingest JWTs.';