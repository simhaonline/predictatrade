# 04 — Database / TimescaleDB Audit (read-only live inspection)

Engine: PostgreSQL 17.10 · timescaledb 2.29.1 + toolkit 1.24.0 · pgvector 0.8.6 · pgcrypto · uuid-ossp.
Schemas: ai(4) audit(5) billing(10) control(8) finance(3) iam(14) licensing(18) market(12) referral(15) research(6) support(2) system(5) trading(90).

## Hypertables & aggregates

`timescaledb_information.hypertables`: market.candles, market.market_states, market.ticks, trading.indicator_history, trading.regime_history, trading.signals, trading.strategy_evaluations. Continuous aggregates: `candles_m5_agg`, `candles_h1_agg`.

## Migration discipline — VERIFIED with caveats

Bookkeeping lives in `audit.migration_history` (+aliases/catalog): **40/40 COMPLETED, 0 failed** including reconciliation migrations (`024_financial_integrity_webhooks_reconciliation`, `026/027 history catalog`). Repo has 28 files; extra entries are historical aliases recorded by migration 026/027 — no drift detected between applied set and repo set.

## Row counts (2026-08-22 21:4x UTC)

| Table | Rows | Note |
|---|---|---|
| iam.users | 12 | 9 DELETED/anonymized; admin@, user@ (email_verified=f!), 1 SUSPENDED |
| billing.subscriptions | 1 | ACTIVE ELITE 2026-08-17→09-17, strategies=[] |
| control.plans | 5 | FREE/BASIC(hidden)/STANDARD/PRO/ELITE |
| billing.payments / payment_events | 1 / 1 | **STRIPE evt_local_invoice_1787381953, signature_verified=false** |
| billing.invoices | 1 | matches fake payment |
| trading.signals | 16,853 | refs: 6,451 empty (Aug 18–21), 10,402 unique non-empty |
| trading.signal_candidates | 17,983 | APPROVED/REJECTED/ADVISORY persisted w/ rejection gates |
| trading.signal_outbox | 2,697 | **100% PENDING** — dispatcher dead |
| signal_deliveries / receipts / delivery_ledger | 0 / 0 / 0 | delivery+quota tracking never written |
| licensing.licenses / devices | 1 / 2 | |
| referral.referral_codes / commission_ledger / payouts | 3 / 0 / 0 | engine dead |
| audit.audit_events | 186 | append-only schema |
| market.ticks / candles(M5) | 7,109,183 / 418,041 | last tick Fri 20:59:57 UTC = weekend close (correct) |

## Data integrity findings

1. **Candle corruption (P1):** 553 rows Aug 18–21 with `open<=0 OR low<=0` while `quality='COMPLETE', source=AGGREGATOR` — e.g. M5 20:55 open=0 low=0 high=4605.50 close=4603.54 vol=48307. OHLC invariants otherwise hold (violations=0). Indicators computed over these bars are contaminated; quality flag lies to downstream consumers and dashboards.
2. **Signal reference gap (P2):** 38% of signals (all Aug 18–21) have empty `signal_reference`; sequence-based refs unique since fix.
3. **M5 weekday gaps:** 9 gaps ≠300s in trailing 72h weekdays (minor; holiday/close-edge candidates).
4. **Fake financial row (P0):** the only payment/invoice/sub chain originates from an unverified locally-fabricated Stripe event — no PSP integration exists that could have produced it.

## Constraints / roles

- Exact-decimal NUMERIC(18,8)/(30,8) used across finance/trading — good.
- Referral schema has strong idempotency/dedup indexes (commission dedup partial unique, v3 external_reference unique) — **idle**, no writers.
- DB roles: single app superuser `pat_admin` used by ALL services (violates least-privilege); roles dir exists but effective runtime uses superuser; pg_hba `host all all all scram-sha-256` + public exposure = P0 (see 18).
- Backup evidence → 27 (stale).
