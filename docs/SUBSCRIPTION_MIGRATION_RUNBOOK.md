# Migration Runbook

1. Capture migration history and commercial/financial counts.
2. Apply `024_subscription_referral_v3.sql` with the canonical migration runner.
3. Verify plan codes, legacy BASIC visibility, entitlement counts, v3 rule versions, and uniqueness indexes.
4. Leave commercial feature flags disabled until provider and distribution authorization evidence exists.

The migration is additive and was syntax-tested in a transaction against the running PostgreSQL instance; that transaction was rolled back.

