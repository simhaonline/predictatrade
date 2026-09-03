-- 123: backtest_runs subscription attribution (R9)
--
-- Stamps each backtest run with the subscription + plan snapshot in force
-- when the run executed, so the operator can report which subscription /
-- plan generated which backtest runs and strategy usage:
--   * subscription_id → billing.subscriptions (ACTIVE/TRIALING at run time)
--   * plan_code / plan_name → denormalized snapshot (survives plan changes)
--
-- resolveActiveSubscription() picks the highest-priced qualifying plan
-- (plan allowed_strategies OR subscription selected_strategies must contain
-- the strategy, or the plan is unlimited). Backfill: historical runs have
-- user_id NULL (pre-R1) and stay unstamped.
ALTER TABLE trading.backtest_runs
    ADD COLUMN IF NOT EXISTS subscription_id uuid REFERENCES billing.subscriptions(id),
    ADD COLUMN IF NOT EXISTS plan_code text,
    ADD COLUMN IF NOT EXISTS plan_name text;

CREATE INDEX IF NOT EXISTS idx_backtest_runs_subscription
    ON trading.backtest_runs(subscription_id);
