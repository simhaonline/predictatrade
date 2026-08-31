-- Migration 111: Capture onboarding/profile fields required by the MASTER
-- PROMPT signup flow (prompt.md). All columns are nullable so existing rows
-- and the registration path remain backward compatible.

ALTER TABLE iam.users
  ADD COLUMN IF NOT EXISTS address_line1 text,
  ADD COLUMN IF NOT EXISTS address_line2 text,
  ADD COLUMN IF NOT EXISTS city text,
  ADD COLUMN IF NOT EXISTS state_region text,
  ADD COLUMN IF NOT EXISTS postal_code text,
  ADD COLUMN IF NOT EXISTS country text,
  ADD COLUMN IF NOT EXISTS mobile text,
  ADD COLUMN IF NOT EXISTS whatsapp text,
  ADD COLUMN IF NOT EXISTS telegram text,
  ADD COLUMN IF NOT EXISTS instagram_handle text,
  ADD COLUMN IF NOT EXISTS facebook_profile text,
  ADD COLUMN IF NOT EXISTS preferred_broker text,
  ADD COLUMN IF NOT EXISTS referral_source text;
