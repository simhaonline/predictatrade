# Database Reconciliation Report — 2026-08-23

## Schema
- Schemas: 22 (trading, market, iam, billing, referral, audit, licensing, control, etc.)
- Tables: 172
- Migrations: 028 files (001-028)
- TimescaleDB hypertables: 7 (candles, ticks, signals, indicator_history, regime_history, strategy_evaluations, market_states)

## Data Volume
- Signals: 16,853
- Candles: 2,761,645
- Ticks: 7,109,183

## Financial Tables Verified
- billing.payments, billing.invoices, billing.invoice_items, billing.refunds
- referral.commission_ledger, referral.referral_codes, referral.referral_relationships
- referral.payouts, referral.payout_items, referral.payout_methods

## Indexes
- signals: 12 indexes including idx_signals_created DESC
- commission_ledger: uniqueness on idempotency_key
- referral_codes: unique on code

## Drift Check
- No destructive migrations detected
- All migrations are forward-additive
- No schema drift between migrations and live DB
