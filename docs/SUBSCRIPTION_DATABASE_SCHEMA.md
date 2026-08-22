# Subscription Database Schema

Migration `024_subscription_referral_v3.sql` adds plan metadata, versioned entitlement records, strategy preferences, explicit subscription events, user signal delivery ledger, commission snapshot columns, v3 effective-dated rules, and commercial feature flags. All money uses DECIMAL(18,8). The signal ledger intentionally uses an indexed signal identifier because the deployed Timescale hypertable has a composite `(id, created_at)` key and cannot accept a scalar foreign key without changing trading storage.

