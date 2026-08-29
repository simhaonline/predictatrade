# Database ERD — Core Domains
## v1.17.3 — 29 August 2026

16 application schemas (see `DATABASE_ARCHITECTURE.md` for full inventory). This ERD covers the
business-critical core: identity, licensing/devices, commercial (plans → subscriptions →
billing), finance (commissions/payouts/ledger), trading (signals → execution → results) and
market data. Column lists are abridged to key columns; `_timescaledb_*` internals omitted.

```mermaid
erDiagram
    %% ─── IAM ───
    iam_users ||--o{ iam_memberships : "has"
    iam_roles ||--o{ iam_memberships : "grants"
    iam_users ||--o{ iam_sessions : "opens"
    iam_users ||--o{ iam_api_credentials : "owns"
    iam_users ||--o{ iam_consent_records : "grants"

    iam_users {
        uuid id PK
        citext email UK
        string password_hash
        string status "ACTIVE|SUSPENDED|DELETED"
        timestamptz anonymized_at "GDPR"
        jsonb preferences
        timestamptz consent_timestamp
    }

    %% ─── LICENSING ───
    iam_users ||--o{ lic_licenses : "issued"
    control_plans ||--o{ lic_licenses : "plan"
    billing_subscriptions ||--o{ lic_licenses : "derives"
    lic_licenses ||--o{ lic_devices : "max_devices"
    lic_licenses ||--o{ lic_device_activations : "activates"
    lic_licenses ||--o{ lic_mt_accounts : "max_mt_accounts"
    lic_licenses ||--o{ lic_license_events : "audits"
    lic_devices ||--o{ lic_device_activations : "one per device"

    lic_licenses {
        uuid id PK
        uuid user_id FK
        uuid plan_id FK
        string status "PENDING|ACTIVE|SUSPENDED|REVOKED"
        string license_key UK "PAT-XXXXXXXX"
        timestamptz expires_at
        timestamptz revoked_at
        int max_devices
        int max_mt_accounts
        jsonb allowed_strategies
        jsonb allowed_execution_modes
    }

    lic_devices {
        uuid id PK
        string fingerprint_hash UK
        string hardware_id
    }

    lic_device_activations {
        uuid id PK
        uuid license_id FK
        uuid device_id FK
        string client_type "MT4|MT5"
        string broker_server
        string mt_account_login
        numeric account_equity
        timestamptz activated_at
    }

    %% ─── COMMERCIAL ───
    control_plans ||--o{ bill_subscriptions : "subscribed"
    iam_users ||--o{ bill_subscriptions : "owner"
    bill_subscriptions ||--o{ bill_invoices : "billed"
    bill_subscriptions ||--o{ bill_subscription_events : "lifecycle"
    bill_invoices ||--o{ bill_payments : "settles"
    control_coupons }o--o{ bill_subscriptions : "discounts"

    control_plans {
        uuid id PK
        string code UK "FREE|BASIC|STANDARD|PRO|ELITE"
        string name
        numeric monthly_price
        numeric annual_price
        bool billing_enabled
        jsonb allowed_strategies
        int max_active_strategy_slots
        numeric daily_loss_cap_pct
        numeric weekly_loss_cap_pct
        numeric monthly_loss_cap_pct
        numeric per_trade_risk_pct
    }

    bill_subscriptions {
        uuid id PK
        uuid user_id FK
        uuid plan_id FK
        string billing_interval "MONTHLY|ANNUAL"
        string status "INCOMPLETE|ACTIVE|PAUSED|CANCELLED"
        jsonb selected_strategies
        timestamptz billing_period_end
    }

    bill_invoices {
        uuid id PK
        uuid subscription_id FK
        numeric amount
        string currency
        string status
        string provider "STRIPE|NOWPAYMENTS"
        string provider_ref
    }

    %% ─── FINANCE (exact-decimal, ledger-backed) ───
    iam_users ||--o{ ref_affiliate_profiles : "referrers"
    ref_affiliate_profiles ||--o{ fin_commission_ledger : "earns"
    bill_invoices ||--o{ fin_commission_ledger : "canonical revenue"
    ia_users_for_payouts ||--o{ fin_payouts : "requests"
    fin_commission_ledger ||--o{ fin_payouts : "paid from"
    fin_payouts ||--o{ fin_ledger_entries : "movement"

    fin_commission_ledger {
        uuid id PK
        uuid invoice_id FK "canonical eligible revenue"
        uuid beneficiary_id FK
        decimal amount "DECIMAL(18,8)"
        string status "PENDING|AVAILABLE|HELD|PAID|REVERSED"
        text reason
    }

    fin_payouts {
        uuid id PK
        uuid user_id FK
        decimal amount "DECIMAL(18,2)"
        string status "REQUESTED|APPROVED|PROCESSING|PAID|REJECTED|FAILED"
        string idempotency_key UK
    }

    %% ─── TRADING ───
    lic_licenses ||--o{ tr_signals : "entitled delivery"
    tr_signals ||--o{ tr_execution_commands : "CLOSE_POSITION etc"
    tr_signals ||--o{ tr_trade_results : "real broker outcomes"
    iam_users ||--o{ tr_signals : "delivery audience"

    tr_signals {
        uuid id PK
        uuid user_id FK
        string strategy_id "5 engines"
        string signal_class "ADVISORY|EXECUTABLE"
        string direction "BUY|SELL|NO-TRADE"
        numeric entry_price
        numeric stop_loss
        numeric tp1_tp2_tp3
        numeric raw_score
        numeric calibrated_probability "VALIDATED only"
        string quality_grade "A+|A|B"
        jsonb reason_codes
        timestamptz created_at
    }

    tr_execution_commands {
        uuid id PK
        uuid signal_id FK
        string command_type "CLOSE|EMERGENCY_STOP|KILL_SWITCH"
        string status
        bigint ticket
    }

    tr_trade_results {
        uuid id PK
        uuid signal_id FK
        string broker_ticket
        string direction
        numeric entry_price
        numeric exit_price
        numeric pnl
        string close_reason "tp|sl|manual"
        bool sl_correct "server-verified"
        int time_in_trade_seconds
        numeric mae
        numeric mfe
    }

    %% ─── MARKET (Timescale hypertables) ───
    tr_signals }o--|| mk_candles : "decided on"
    mk_candles ||--o{ mk_cot_reports : "macro context"

    mk_candles {
        timestamptz time PK hypertable
        string symbol PK
        string timeframe PK
        double precision open
        double precision high
        double precision low
        double precision close
        bigint volume
        string source
    }

    %% ─── AUDIT / COMPLIANCE ───
    iam_users ||--o{ aud_audit_events : "actor"
    iam_users ||--o{ aud_client_events : "subject"
    compliance_gdpr_operations }o--|| iam_users : "erases"

    aud_audit_events {
        uuid id PK
        uuid actor_id
        string event_type
        jsonb metadata
        timestamptz event_time
    }
```

## Companion registries (referenced above)

| Area | Tables |
|---|---|
| Cross-market intelligence | `trading.cross_market_*` (confluence, correlation regimes, driver snapshots, shadow validation, ablation, provider health) |
| Backtest/research | `trading.backtest_runs/artifacts/datasets/fold_results/parameter_sets/trades`, `research.feature_parity_runs` |
| PTB / calibration | `calibration.calibration_profiles/reports`, `ptb_feature_flags`, `ptb.*` |
| Devil Liquidity | `trading.*` mark/qualification artifacts via `liquidity` API (engine state, persisted best-effort) |
| Live preview | `live_preview.anonymous_trials`, `funnel_stats` |
| Support/ops | `support.*`, `control.commercial_feature_flags`, `audit.migration_history` |
| News/macro | `market.economic_events`, `market.cot_*`, `market.data_capabilities/provenance_log` |

## Money & time invariants

- All money columns are `NUMERIC(18,8)` / `NUMERIC(10,4)` — **never** float (control plane also
  uses `decimal.js` for payout math).
- All timestamps are `TIMESTAMPTZ`; internal truth is UTC; broker-server time is a display
  conversion only (`market/proxy`, `features/session.go`).
- Financial history is append-only — corrections are compensating/reversal rows, never UPDATEs.
- Hypertables: `market.candles` (retention 081), `trading.client_event_log` (+ audit retention 064).
- GDPR: `iam.users.anonymized_at` + `compliance.gdpr_operations` (migration 088).