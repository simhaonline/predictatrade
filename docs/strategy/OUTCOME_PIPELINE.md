# Outcome Capture Pipeline — Design (Phase 0.5, Candidate B)
Date: 2026-09-17 · Status: IMPLEMENTED (Phase 0.5 complete — see PHASE0_5_REPORT.md)

## 1. Lifecycle (current, verified end-to-end)

```
signal emit (processCandle, EXECUTABLE)
  └─ trading.signals row (id)                         ← persist-before-deliver
  └─ trading.signal_feature_snapshots (1:1, P0)       ← raw indicator reads
  └─ licensing.edge_signal_queue (EXECUTABLE)         ← delivery ledger
        │ edge-poll (3s cadence, exec-role device)
        ▼
  [EDGE-ACK] delivery confirmed                        ← delivery ≠ outcome
        ▼
  EA opens position (broker)                           ← no server record yet
  └─ EXECUTION_ACK → reconciliation
        ▼
  EA manages: TP1/TP2/TP3 partials, BE-after-TP1, ATR trail
        ▼
  EA closes: TP1 / TP2 / TP3 / SL / manual / friday
        ▼
  EA sends TRADE_RESULT (ingest, signal_id + exit_reason + pnl …)
        ▼
  agent_provider "TRADE_RESULT" → SetTradeResultFn
        ▼
  main.go:2423 writer → INSERT trading.trade_results  ← EXISTS, works
        ▼
  trading.prediction_outcomes                          ← ✗ NO WRITER (always empty)
  trading.predictions                                  ← ✗ NO WRITER (always empty)
```

## 2. Break-point table (verified 2026-09-17)

| # | Stage | State | Root cause / evidence |
|---|-------|-------|----------------------|
| 1 | signal → prediction row | **BROKEN** | `trading.predictions` has 0 rows; no writer exists. Outcome FK requires it. |
| 2 | trade close → prediction_outcomes | **BROKEN** | 0 rows; no writer anywhere in repo. |
| 3 | TRADE_RESULT arrival | **SILENT 6 days** | Last message Sep 11 20:00 UTC. Exec-role devices OFFLINE since Sep 12 — trades only occur when a client EA runs. Server side is fine. |
| 4 | EA MAE/MFE recording | **BROKEN (EA-side)** | `PredictATrade_MT5.mq5:1338-1339` hardcodes `mae:0.0, mfe:0.0` — explains inverted/unusable MAE/MFE in trade_results. Server fix impossible; EA must track running extremes per position. |
| 5 | legacy linkage | **82/227 unlinked** | trade_results rows whose signal_id does not resolve to trading.signals (fallback-UUID path for old EAs). Never invent — record UNLINKED. |
| 6 | ACK ledger | OK (by design) | 43,981 ACKED = delivery confirmations, not outcomes. Cleaned by EDGE-SWEEP TTL. |

## 3. Schema (migration 150_outcome_pipeline.sql — idempotent, recorded)

**3a. `trading.predictions` — one row per EXECUTABLE signal at emit time**
(the pre-registered expectation; parent of outcomes):

```
id UUID PK DEFAULT gen_random_uuid()
signal_id   UUID UNIQUE NOT NULL   -- 1:1 with trading.signals.id
symbol, timeframe, strategy_id, direction  TEXT NOT NULL
entry, stop_loss, tp1, tp2, tp3        NUMERIC(18,8)
raw_score, calibrated_probability      NUMERIC(10,4)
grade, signal_class, tier, tier_reason TEXT
composite_score                        NUMERIC(10,4)
family_sub_scores                      JSONB
regime, session                        TEXT
gate_decisions                         JSONB
astro_sizing_multiplier, yoga_bias     NUMERIC(10,4)
macro_bias_x6                          NUMERIC(10,4)
feature_snapshot_id                    UUID NULL
model_version, strategy_version        TEXT
created_at, expires_at                 TIMESTAMPTZ
```
FK: signal_id → trading.signals(id). Written inside the SaveSignal flow
(idempotent on signal_id).

**3b. `trading.prediction_outcomes` — EXTEND (additive columns; existing FK kept):**
`signal_id UUID UNIQUE` (join key; also = predictions.id), `link_status`
('LINKED' | 'UNLINKED'), `strategy_id`, `close_reason` (AUTO/MANUAL/BE/TRAIL/
TP1/TP2/TP3/FRIDAY_FLATTEN/STOP/HARD_STOP/TIMEOUT), `realized_pnl`,
`r_multiple`, `mae`, `mfe`, `duration_seconds`, `slippage_est`,
`spread_at_entry`, `spread_at_exit`, `trade_result_id` (FK → trade_results),
`schema_version`, `created_at`.
`outcome_type` stays ('WIN'/'LOSS'/'BREAKEVEN'/'EXPIRED'), `outcome_value`
boolean = is_win. Existing columns preserved (backward compat).

## 4. Writers

| Writer | Trigger | Behavior |
|--------|---------|----------|
| `SavePrediction` | signal emit (EXECUTABLE only) | INSERT prediction row (idempotent on signal_id); sets feature_snapshot_id from the signal's snapshot |
| `SaveOutcomeFromTradeResult` | SetTradeResultFn, after trade_results INSERT | UPSERT prediction_outcomes on signal_id; derive r_multiple = (exit−entry)·dir / (entry−SL); outcome_type from pnl sign; close_reason passed through (MANUAL preserved); link_status='LINKED' when the signal resolves, 'UNLINKED' otherwise — never dropped, logged + counted |
| `BackfillOutcomes` (one-shot script) | operator run | walks 227 trade_results oldest→newest, attempts linkage, reports linked/ambiguous/unlinked counts; never invents ids |

## 5. Metrics (query pack — parity-tested against prediction_outcomes)

- `SELECT strategy_id, outcome_type, count(*) FROM trading.prediction_outcomes GROUP BY 1,2`
- expectancy_R = avg(r_multiple) FILTER (WHERE outcome_type != 'EXPIRED')
- realized N per strategy/tier/regime (join predictions for tier/regime)
- link coverage = count(link_status='LINKED') / count(*)
- alerting: writer silent > 30 min while EXECUTABLE signals emit; UNLINKED ratio > 20%

## 6. What this does NOT do (guardrails)
- No tuning of weights/thresholds/tiers/gates (Phase 1 blocked until N≥300/strategy).
- No fabrication: ambiguous linkage ⇒ link_status='UNLINKED' + alert.
- No behavior change on the signal/delivery path (writers fire-and-forget
  after persist-before-deliver; failures logged, never block emission).

## 7. Shadow-resolver continuity (discovered during design)

`trading.cross_market_shadow_snapshots` already carries **2,891
price-resolved outcomes with R-multiples** (1,657 SL / 905 TP1 / 55 TP2 /
152 TP3 / 122 expired) — uncontaminated by manual closes. Per-strategy R
expectancy from that sample: ULTRA_SCALPING **+0.72R**, STANDARD_SCALPING
−0.17R, STANDARD_SWING −0.38R, TREND_SWING −0.80R. LIMITATIONS: rows cover
only Aug 24–25 (resolver stopped writing outcomes after that — investigate
separately), and legacy rows have empty `signal_id` (linkage impossible for
them). Going forward the resolver will stamp `signal_id` so shadow outcomes
link to predictions — giving an independent, execution-bias-free calibration
channel alongside the realized one.

## 8. Time-to-300 estimate (after B + client EAs online)
Executed flow ~300/day when devices are active (Sep 17 rate).
STANDARD_SCALPING (~40% of exec flow) → 300 outcomes in ~3–5 trading days of
live client-EA activity; slower strategies (TREND_SWING, MARNIE_FIB) 2–4
weeks. Shadow channel adds an independent sample stream in parallel.

## 9. EA-side gap (operator action required)
MAE/MFE must be recorded by the client EA (running high/low extremes per
position) and sent in TRADE_RESULT — the current build hardcodes 0.0. No
server-side fix exists. Compile-time change in `mql/mt5/PredictATrade_MT5.mq5`.