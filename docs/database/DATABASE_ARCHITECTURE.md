# Database Architecture

## Version: v1.2.0 — Advanced Risk + Backtesting

## Overview

PostgreSQL 17 with TimescaleDB extension for time-series hypertables. pgvector for AI vector embeddings. Valkey for hot cache.

## Schemas

| Schema | Purpose |
|--------|---------|
| `iam` | Users, roles, sessions, MFA, organizations, permissions |
| `control` | Plans, entitlements, platform operations |
| `billing` | Subscriptions, invoices, payments, refunds, credits, coupons |
| `referral` | Referral relationships, commission ledger, payouts, affiliate profiles |
| `licensing` | Licenses, devices, activations, credentials, MT accounts, session leases |
| `trading` | Strategies, signals, candidates, risk decisions, PTB, recovery, hedging, sentiment, backtesting |
| `market` | Ticks, candles, indicator history, regime history, broker profiles, sessions |
| `audit` | Audit events, security events |
| `ai` | Model registry, training jobs, inference history, vector embeddings |
| `research` | Backtest runs, trades, fold results, validation reports, walk-forward runs |
| `system` | System configuration, notifications, backup metadata, WAL archive status |
| `support` | Support tickets, support messages |

## Migrations (15 total, 165 tables)

| # | File | Description | Tables Added |
|---|------|-------------|-------------|
| 001 | `001_create_schemas_and_roles.sql` | Schemas, roles, extensions | Schemas + roles |
| 002 | `002_iam_tables.sql` | IAM tables (users, roles, sessions, MFA, orgs) | ~12 |
| 003 | `003_plans_billing_licensing.sql` | Plans, billing, licensing, devices, MT accounts | ~25 |
| 004 | `004_referral_commission_payout.sql` | Referral, commission, payout | ~15 |
| 005 | `005_trading_market_tables.sql` | Trading, market data, strategy configs, signals, candidates, risk | ~40 |
| 006 | `006_session_token_rotation.sql` | Session token rotation security | ~2 |
| 007 | `007_auth_hardening.sql` | Auth hardening (recovery codes, login events) | ~3 |
| 008 | `008_device_activation_sessions.sql` | Device activation sessions | ~3 |
| 009 | `009_signal_delivery_replay.sql` | Signal delivery, replay, AI model registry, training jobs | ~8 |
| 010 | `010_database_completion_audit.sql` | Strategy evaluations, indicator history, regime history, COT, vector embeddings, backup config | ~15 |
| 011 | `011_cot_capability_wal.sql` | COT features, positions, reports, ingestion runs, WAL archive | ~8 |
| 012 | `012_ptb_intelligence_tables.sql` | PTB feature flags, evidence snapshots, data provenance log | ~4 |
| 013 | `013_ptb_synthesis_performance.sql` | PTB analysis history (hypertable), signal performance (hypertable) | ~2 |
| 014 | `014_advanced_risk_adaptation_intelligence.sql` | Recovery states, trade results, blocked signals, adaptation history, hedge positions, hedge history, RL training history, sentiment snapshots, sentiment items | ~9 |
| 015 | `015_backtesting_tables.sql` | Backtest runs, trades, fold results, artifacts, parameter sets | ~5 |

## Advanced Risk Tables (Migration 014)

### trading.recovery_states
Loss recovery state machine per account+strategy. States: NORMAL, RECOVERY, HALTED, DAILY_LIMIT. Tracks consecutive losses, daily PnL, cooldown, halt expiry. Isolated per account+strategy+symbol.

### trading.trade_results
Closed trade outcomes: entry/exit, PnL, PnL percent, R multiple, MAE/MFE, commission, slippage, duration, regime, session, confluence, confidence, setup grade. Dedup via close_event_id. TimescaleDB hypertable.

### trading.blocked_signals
Audit trail of signals blocked by recovery/risk gates. Records block reason, recovery state, adaptation phase.

### trading.adaptation_history
Rule-based and ML-based adaptation decisions with full context: market phase, regime, volatility, manipulation index, adjustments, reason, source. TimescaleDB hypertable.

### trading.hedge_positions
Active hedge lifecycle: original/hedge trade correlation, direction, size, entry, SL/TP, reason opened, status, aggregate exposure, expiry.

### trading.hedge_history
Closed hedge audit trail: full lifecycle data plus net PnL (original + hedge), duration.

### trading.rl_training_history
RL model training runs: mode (disabled/shadow/filter/live), episodes, OOS validation metrics (reward, drawdown, Sharpe, Sortino, profit factor, win rate, expectancy), walk-forward folds, reward config, hyperparameters, status.

### trading.sentiment_snapshots
Cached real-time sentiment state: overall score (-100 to +100), confidence, category, provider health, last successful update, data age. Unique per symbol.

### trading.sentiment_items
Individual sentiment data points with full provenance: source, provider, headline ID, score, confidence, category, text preview, source timestamp, fetched_at, age. TimescaleDB hypertable.

## Backtesting Tables (Migration 015)

### trading.backtest_runs
Top-level run tracking: run_id, symbol, strategy, timeframe, start/end, initial balance, seed, status, metrics, configuration, execution assumptions, data hash, feature/model versions, git commit SHA, artifact locations.

### trading.backtest_trades
Individual trade records: direction, entry/exit, PnL, PnL_R, commission, slippage, MAE/MFE, duration, regime, session, confluence, confidence, setup grade.

### trading.backtest_fold_results
Walk-forward fold outcomes: fold ID, train/test boundaries, in-sample and OOS metrics.

### trading.backtest_artifacts
File locations for generated reports: artifact type, file path, file size.

### trading.backtest_parameter_sets
Parameter search grid for sensitivity analysis: parameter name, value, is_base, metrics.

## PTB Tables (Migrations 012-013)

### trading.ptb_feature_flags
Module activation states. All modules default SHADOW. Institutional Footprint = UNSUPPORTED.

### trading.ptb_evidence_snapshots
Per-evaluation PTB module results (JSONB columns for each of 20 modules). Indexed by signal_id and timestamp.

### trading.data_provenance_log
Data source authenticity tracking. Records source_type, is_live, is_stale, data_age_ms.

### trading.ptb_analysis_history
Full PTB synthesis records: regime, gold_role, bias, confidence, confluence_score, setup_quality, action, narrative, component_scores, reason_codes, position_size_multiplier, stop_distance_multiplier. TimescaleDB hypertable.

### trading.signal_performance
Signal outcome feedback: entry/exit prices, PnL, MAE, MFE, time in trade, slippage, TP1/TP2/TP3/SL hit, regime at entry. TimescaleDB hypertable.

## Hypertables

| Table | Time Column | Migration |
|-------|------------|-----------|
| market.ticks | time | 005 |
| market.candles | time | 005 |
| market.market_states | time | 005 |
| market.flow_features | time | 005 |
| trading.strategy_evaluations | timestamp | 010 |
| trading.indicator_history | timestamp | 010 |
| trading.regime_history | timestamp | 010 |
| trading.ptb_analysis_history | timestamp | 013 |
| trading.signal_performance | created_at | 013 |
| trading.trade_results | closed_at | 014 |
| trading.adaptation_history | timestamp | 014 |
| trading.sentiment_items | fetched_at | 014 |

## Notes

- All migrations are additive (`CREATE TABLE IF NOT EXISTS`) — no destructive changes
- TimescaleDB hypertable creation is idempotent (fails silently if extension not present)
- pgvector extension creates `ai.vector_embeddings` table (migration 010)
- WAL archiving tables created in migration 011 for PITR support
- All financial tables use exact-decimal types (DECIMAL/NUMERIC)
