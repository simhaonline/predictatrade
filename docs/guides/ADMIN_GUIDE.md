# Admin Guide

**Version:** v1.2.0 — Advanced Risk + Backtesting  
**Date:** 18 August 2026

---

## Admin Console

The admin console is available at `https://platform.predictatrade.com/admin` and requires ADMIN role JWT authentication.

### Admin Pages

| Page | Route | Description |
|------|-------|-------------|
| Overview | `/admin` | Platform-wide stats: users, subscriptions, revenue, signals |
| Users | `/admin/users` | User management: list, search, suspend/activate |
| Subscriptions | `/admin/subscriptions` | All subscriptions with plan/status filtering |
| Commissions | `/admin/commissions` | Commission ledger with summary |
| Payouts | `/admin/payouts` | Payout requests and history with stats |
| Licenses | `/admin/licenses` | License management |
| Devices | `/admin/devices` | Registered device management |
| Strategies | `/admin/strategies` | Strategy enable/disable, configuration |
| Risk | `/admin/risk` | Risk engine settings, loss recovery states |
| Executions | `/admin/executions` | Trade execution history |
| Signal Replay | `/admin/signal-replay` | Signal replay and audit |
| Market Data | `/admin/market-data` | Market data provenance and quality |
| AI Models | `/admin/ai` | ML/RL model registry and training jobs |
| Audit | `/admin/audit` | Audit event log |
| Billing | `/admin/billing` | Invoices and payment history |
| Finance | `/admin/finance` | Financial overview: revenue, commissions, payouts |
| Infrastructure | `/admin/infrastructure` | System health, service status |
| Releases | `/admin/releases` | Release management and deployment status |
| Support | `/admin/support` | Support ticket management |

---

## PTB Feature Flag Management

### View All Module Modes

```sql
SELECT module_name, mode, reason, set_by, set_at
FROM trading.ptb_feature_flags
ORDER BY module_name;
```

### Activate a Module (after validation)

```sql
UPDATE trading.ptb_feature_flags
SET mode = 'ACTIVE', set_by = 'admin', set_at = now(), reason = 'validated'
WHERE module_name = 'liquidity_void';
```

### Disable a Module

```sql
UPDATE trading.ptb_feature_flags
SET mode = 'DISABLED', set_by = 'admin', set_at = now(), reason = 'performance issue'
WHERE module_name = 'manipulation_proxy';
```

### Available Modes

| Mode | Behavior |
|------|---------|
| OFF | Module not executed |
| SHADOW | Module calculates and persists, zero score impact (default) |
| ACTIVE | Module contributes to production score |
| DISABLED | Module explicitly turned off |
| UNSUPPORTED | Feature cannot work with current data source |
| RESEARCH | Module in research/experimental phase |

---

## PTB Analysis History

```sql
SELECT analysis_id, timestamp, regime, bias, confidence, confluence_score,
       setup_quality, action, position_size_multiplier, stop_distance_multiplier
FROM trading.ptb_analysis_history
ORDER BY timestamp DESC LIMIT 20;
```

## Signal Performance Feedback

```sql
SELECT signal_id, strategy, setup_quality, regime_at_entry, pnl, pnl_percent,
       tp1_hit, tp2_hit, tp3_hit, sl_hit, mae, mfe, time_in_trade
FROM trading.signal_performance
ORDER BY created_at DESC LIMIT 20;
```

## Data Provenance

```sql
SELECT source_type, is_live, is_stale, data_age_ms, symbol, market_timestamp
FROM trading.data_provenance_log
ORDER BY market_timestamp DESC LIMIT 20;
```

---

## Advanced Risk Management (v1.1.0)

### Loss Recovery State Inspection

```sql
SELECT account_id, strategy_id, symbol, state, consecutive_losses,
       daily_pnl_percent, cooldown_until, halt_until
FROM trading.recovery_states
ORDER BY state, updated_at DESC;
```

### Trade Results Audit

```sql
SELECT signal_id, strategy, symbol, direction, entry_price, exit_price,
       pnl, pnl_percent, r_multiple, mae, mfe, commission, slippage,
       regime_at_entry, session_at_entry, confluence_at_entry, setup_grade
FROM trading.trade_results
ORDER BY closed_at DESC LIMIT 20;
```

### Blocked Signals Audit

```sql
SELECT signal_id, strategy, symbol, block_reason, recovery_state,
       adaptation_phase, blocked_at
FROM trading.blocked_signals
ORDER BY blocked_at DESC LIMIT 20;
```

### Adaptation History

```sql
SELECT timestamp, strategy_id, symbol, market_phase, regime, volatility,
       manipulation_index, risk_multiplier, stop_distance_multiplier,
       confluence_bonus, adjustment_source, reason
FROM trading.adaptation_history
ORDER BY timestamp DESC LIMIT 20;
```

### Hedge Position Monitoring

```sql
SELECT * FROM trading.hedge_positions WHERE status = 'ACTIVE';
SELECT * FROM trading.hedge_history ORDER BY closed_at DESC LIMIT 20;
```

### RL Training History

```sql
SELECT run_id, mode, episodes, oos_reward, oos_drawdown, oos_sharpe,
       oos_sortino, oos_profit_factor, oos_win_rate, status
FROM trading.rl_training_history
ORDER BY created_at DESC LIMIT 20;
```

### Sentiment Monitoring

```sql
SELECT symbol, overall_score, confidence, category, provider_health,
       last_successful_update, data_age_ms
FROM trading.sentiment_snapshots;

SELECT source, provider, headline_id, score, confidence, category,
       text_preview, source_timestamp, fetched_at, age_ms
FROM trading.sentiment_items
ORDER BY fetched_at DESC LIMIT 20;
```

---

## Operations Controls

The admin can control trading operations via the Operations API:

| Action | Endpoint | Effect |
|--------|----------|--------|
| Halt trading | `POST /operations/halt-trading` | Emergency stop — no new trades |
| Resume trading | `POST /operations/resume-trading` | Resume after halt |
| Pause signals | `POST /operations/pause-signals` | Stop signal delivery (keep evaluating) |
| Resume signals | `POST /operations/resume-signals` | Resume signal delivery |
| Enable strategy | `POST /operations/strategy/:id/enable` | Enable specific strategy |
| Disable strategy | `POST /operations/strategy/:id/disable` | Disable specific strategy |

### AI Model Management

| Action | Endpoint | Description |
|--------|----------|-------------|
| List models | `GET /operations/ai/models` | All models in registry |
| List training jobs | `GET /operations/ai/training-jobs` | Training run history |
| Inference history | `GET /operations/ai/inference` | Model inference logs |
| Activate model | `POST /operations/ai/model/:id/activate` | Deploy model to production |
| Deactivate model | `POST /operations/ai/model/:id/deactivate` | Remove model from production |

---

## Backtesting Management (v1.2.0)

### View Backtest Runs

```sql
SELECT run_id, symbol, strategy, timeframe, status, initial_balance,
       final_balance, total_return, max_drawdown, sharpe_ratio,
       total_trades, win_rate, seed, git_commit_sha
FROM trading.backtest_runs
ORDER BY created_at DESC LIMIT 20;
```

### View Backtest Trades

```sql
SELECT run_id, direction, entry_price, exit_price, pnl, pnl_r,
       commission, slippage, mae, mfe, duration_seconds, regime, session,
       confluence, confidence, setup_grade
FROM trading.backtest_trades
ORDER BY run_id, entry_time DESC LIMIT 50;
```

### Walk-Forward Fold Results

```sql
SELECT run_id, fold_id, train_start, train_end, test_start, test_end,
       in_sample_return, oos_return, in_sample_sharpe, oos_sharpe
FROM trading.backtest_fold_results
ORDER BY run_id, fold_id;
```

### Backtest Artifacts

```sql
SELECT run_id, artifact_type, file_path, file_size
FROM trading.backtest_artifacts
ORDER BY run_id;
```

### Parameter Sensitivity Sets

```sql
SELECT run_id, parameter_name, parameter_value, is_base,
       return_pct, max_drawdown, sharpe_ratio
FROM trading.backtest_parameter_sets
ORDER BY run_id, parameter_name, parameter_value;
```

---

## Four Strategy Status

All four strategies are ACTIVE and evaluate independently every eligible cycle:

| Strategy | Timeframes | Threshold | Min RR | Cooldown |
|----------|-----------|-----------|--------|----------|
| STANDARD_SCALPING | M1/M5 + M15/M30 | 65 | 1.2 | 15m |
| ULTRA_SCALPING | M1 + M5 | 85 | 1.0 | 15m |
| STANDARD_SWING | M15/M30/H1 + H4/D1 | 55 | 1.8 | 120m |
| TREND_SWING | H1/H4 + D1/W1 | 50 | 2.5 | 360m |

## Signal Direction Types

| Direction | Meaning | Executable? | Signal Class |
|-----------|---------|-------------|--------------|
| **BUY** | Qualified long — score ≥ trade threshold + all 12 gates passed | ✅ | EXECUTABLE |
| **SELL** | Qualified short — score ≥ trade threshold + all 12 gates passed | ✅ | EXECUTABLE |
| **BUY_CANDIDATE** | Advisory long — candidate threshold ≤ score < trade threshold | ❌ | ADVISORY |
| **SELL_CANDIDATE** | Advisory short — candidate threshold ≤ score < trade threshold | ❌ | ADVISORY |
| **WAIT** | Setup exists, MTF conflict | ❌ | — |
| **NO-TRADE** | Score below candidate threshold or strategy NO-TRADE | ❌ | — |
| **BLOCKED** | Had direction (BUY/SELL) but hard gate vetoed or safety block | ❌ | ADVISORY |
| **ERROR** | Missing data or processing error | ❌ | — |

### Admin Signal Panel (`/admin/signals`)

The admin signal panel displays all signals (including NO-TRADE and candidates) from the Go real-time engine:

- **Direction filters**: ALL, BUY, BUY_CANDIDATE, SELL, SELL_CANDIDATE, NO-TRADE
- **Strategy tabs**: ALL, STANDARD_SCALPING, ULTRA_SCALPING, STANDARD_SWING, TREND_SWING
- **Columns**: Time, Direction, Strategy, Symbol, Prob, Score, Entry, SL, TP1, TP2, TP3, Regime, Session, Status
- **Expandable rows**: Evidence breakdown, gate results, NO-TRADE reasons, full entry/SL/TP grid
- **PROB column**: Shows "Pending" when calibration model is UNVERIFIED (SOW §16, §36). Shows percentage when VALIDATED/PROMOTED.
- **WebSocket live updates**: New signals appear in real-time

### Candidate Thresholds

| Strategy | Candidate Threshold | Trade Threshold |
|----------|--------------------:|----------------:|
| STANDARD_SCALPING | 40 | 65 |
| ULTRA_SCALPING | 40 | 65 |
| STANDARD_SWING | 35 | 55 |
| TREND_SWING | 30 | 50 |

See `docs/SIGNAL_TYPES_AND_PROBABILITY.md` for the full reference.

---

## Configuration Reference

### Advanced Risk Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `LOSS_RECOVERY_ENABLED` | true | Loss recovery / capital protection |
| `MAX_DAILY_LOSS_PERCENT` | 2.0 | Max daily loss percentage |
| `MAX_CONSECUTIVE_LOSSES` | 2 | Max consecutive losses before recovery |
| `ADAPTATION_ENABLED` | true | Rule-based market phase adaptation |
| `HEDGING_ENABLED` | false | Controlled hedging (DISABLED by default) |
| `ML_ADAPTATION_ENABLED` | false | ML-based adaptation (research/offline) |
| `RL_MODE` | disabled | RL mode: disabled, shadow, filter_only, live_approved |
| `SENTIMENT_ENABLED` | false | Sentiment engine (requires API credentials) |

### PTB Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PTB_ENABLED` | true | PTB master switch |
| `PTB_SHADOW_MODE` | true | PTB shadow mode (zero score impact) |

---

## Prometheus Metrics

Key metrics to monitor:

| Metric | Description |
|--------|-------------|
| `pat_signals_generated_total` | Signal count by strategy/direction |
| `pat_ptb_analysis_total` | PTB analysis count by action |
| `pat_ptb_analysis_latency_ms` | PTB evaluation latency |
| `pat_ptb_setup_quality_total` | Setup quality grade distribution |
| `pat_ptb_regime_total` | Regime distribution |
| `pat_ptb_manipulation_index` | Current manipulation index |
| `pat_ptb_confluence_score` | Current confluence score |
| `pat_ptb_component_failure_total` | Component failures |
| `pat_ptb_stale_input_total` | Stale input occurrences |
| `pat_recovery_state_total` | Recovery state distribution |
| `pat_adaptation_phase_total` | Adaptation phase distribution |
| `pat_hedge_active_total` | Active hedge count |
| `pat_sentiment_score` | Current sentiment score |
| `pat_rl_filter_veto_total` | RL filter veto count |
| `pat_ml_inference_total` | ML inference count |

Grafana dashboard: `infra/grafana/dashboards/gate-health.json`
